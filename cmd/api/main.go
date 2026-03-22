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

	"github.com/rgomids/atlas-erp-core/internal/customers"
	"github.com/rgomids/atlas-erp-core/internal/invoices"
	"github.com/rgomids/atlas-erp-core/internal/payments"
	"github.com/rgomids/atlas-erp-core/internal/shared/config"
	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
	"github.com/rgomids/atlas-erp-core/internal/shared/logging"
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

	db, err := postgres.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	customerModule := customers.NewModule(db)
	invoiceModule := invoices.NewModule(db, customerModule.ExistenceChecker())
	paymentModule := payments.NewModule(db, invoiceModule.PaymentPort())

	server := &http.Server{
		Addr: cfg.App.Address(),
		Handler: httpapi.NewRouter(
			logger,
			cfg.App.CorrelationIDHeader,
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
