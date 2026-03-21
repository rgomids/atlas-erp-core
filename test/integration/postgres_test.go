package integration_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rgomids/atlas-erp-core/internal/shared/postgres"
	"github.com/rgomids/atlas-erp-core/test/support"
)

func TestOpenConnectsAndPingsPostgres(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	pool, err := postgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("expected postgres pool to connect, got error: %v", err)
	}
	defer pool.Close()
}

func TestRunMigrationsExecutesFoundationFiles(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("resolve migrations path: %v", err)
	}

	if err := postgres.RunMigrations("up", migrationsPath, databaseConfig.ConnectionString()); err != nil {
		t.Fatalf("expected migrations to run, got error: %v", err)
	}
}
