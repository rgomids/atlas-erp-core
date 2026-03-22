package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rgomids/atlas-erp-core/internal/billing"
	"github.com/rgomids/atlas-erp-core/internal/customers"
	"github.com/rgomids/atlas-erp-core/internal/invoices"
	"github.com/rgomids/atlas-erp-core/internal/payments"
	"github.com/rgomids/atlas-erp-core/internal/shared/config"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
	"github.com/rgomids/atlas-erp-core/internal/shared/logging"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
	"github.com/rgomids/atlas-erp-core/internal/shared/outbox"
	"github.com/rgomids/atlas-erp-core/internal/shared/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application terminated", slog.Any("err", err))
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger, err := logging.New(cfg.App.LogLevel)
	if err != nil {
		return err
	}
	slog.SetDefault(logger)
	bootstrapLogger := logger.With(
		slog.String("module", "bootstrap"),
		slog.String("request_id", ""),
	)

	telemetry, err := observability.New(ctx, observability.Config{
		ServiceName:   cfg.App.Name,
		Environment:   cfg.App.Env,
		TraceEndpoint: cfg.Observability.TraceEndpoint,
	})
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if shutdownErr := telemetry.Shutdown(shutdownCtx); shutdownErr != nil {
			bootstrapLogger.Error("telemetry shutdown failed", slog.Any("err", shutdownErr))
		}
	}()

	db, err := postgres.Open(ctx, cfg.Database, telemetry)
	if err != nil {
		return err
	}
	defer db.Close()

	eventBus := sharedevent.NewSyncBusWithObservability(telemetry, outbox.NewPostgresRecorder(db))
	customerModule := customers.NewModule(db, eventBus, telemetry)
	invoiceModule := invoices.NewModule(db, customerModule.ExistenceChecker(), eventBus, telemetry)
	billingModule := billing.NewModule(db, eventBus, telemetry)
	paymentModule := payments.NewModule(db, billingModule.PaymentPort(), eventBus, nil, payments.ModuleConfig{
		GatewayTimeout: cfg.Payments.GatewayTimeout,
	}, telemetry)

	server := &http.Server{
		Addr: cfg.App.Address(),
		Handler: httpapi.NewRouterWithObservability(
			logger,
			cfg.App.CorrelationIDHeader,
			telemetry,
			customerModule.Routes,
			invoiceModule.Routes,
			paymentModule.Routes,
		),
		ReadHeaderTimeout: 5 * time.Second,
	}

	bootstrapLogger.Info(
		"api starting",
		slog.String("app_name", cfg.App.Name),
		slog.String("app_env", cfg.App.Env),
		slog.Int("app_port", cfg.App.Port),
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err = <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		bootstrapLogger.Info("api shutting down")

		return server.Shutdown(shutdownCtx)
	}
}
