package support

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	postgresmodule "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/rgomids/atlas-erp-core/internal/shared/config"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
)

func StartPostgres(ctx context.Context, t testing.TB) (config.DatabaseConfig, func()) {
	t.Helper()

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Skipf("skipping test because docker is unavailable: %v", recovered)
		}
	}()

	container, err := postgresmodule.Run(
		ctx,
		"postgres:16-alpine",
		postgresmodule.WithDatabase("atlas"),
		postgresmodule.WithUsername("atlas"),
		postgresmodule.WithPassword("atlas"),
		postgresmodule.BasicWaitStrategies(),
		postgresmodule.WithSQLDriver("pgx"),
	)
	if err != nil {
		if dockerUnavailable(err) {
			t.Skipf("skipping test because docker is unavailable: %v", err)
		}

		t.Fatalf("start postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("get postgres host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("get postgres port: %v", err)
	}

	parsedPort, err := strconv.Atoi(port.Port())
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("parse postgres port: %v", err)
	}

	databaseConfig := config.DatabaseConfig{
		Host:     host,
		Port:     parsedPort,
		User:     "atlas",
		Password: "atlas",
		Name:     "atlas",
	}

	if err := waitUntilPostgresReady(ctx, databaseConfig); err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("wait for postgres readiness: %v", err)
	}

	return databaseConfig, func() {
		_ = container.Terminate(ctx)
	}
}

func waitUntilPostgresReady(ctx context.Context, databaseConfig config.DatabaseConfig) error {
	const (
		startupTimeout = 20 * time.Second
		retryInterval  = 250 * time.Millisecond
	)

	deadlineCtx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()

	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	var lastErr error
	for {
		pool, err := sharedpostgres.Open(deadlineCtx, databaseConfig)
		if err == nil {
			pool.Close()
			return nil
		}

		lastErr = err

		select {
		case <-deadlineCtx.Done():
			if lastErr != nil {
				return fmt.Errorf("postgres did not become ready within %s: %w", startupTimeout, lastErr)
			}

			return fmt.Errorf("postgres did not become ready within %s", startupTimeout)
		case <-ticker.C:
		}
	}
}

func dockerUnavailable(err error) bool {
	message := strings.ToLower(fmt.Sprint(err))

	return strings.Contains(message, "docker") &&
		(strings.Contains(message, "permission denied") ||
			strings.Contains(message, "operation not permitted") ||
			strings.Contains(message, "cannot connect"))
}
