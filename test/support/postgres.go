package support

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	postgresmodule "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/rgomids/atlas-erp-core/internal/shared/config"
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
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections")),
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

	return config.DatabaseConfig{
			Host:     host,
			Port:     parsedPort,
			User:     "atlas",
			Password: "atlas",
			Name:     "atlas",
		}, func() {
			_ = container.Terminate(ctx)
		}
}

func dockerUnavailable(err error) bool {
	message := strings.ToLower(fmt.Sprint(err))

	return strings.Contains(message, "docker") &&
		(strings.Contains(message, "permission denied") ||
			strings.Contains(message, "operation not permitted") ||
			strings.Contains(message, "cannot connect"))
}
