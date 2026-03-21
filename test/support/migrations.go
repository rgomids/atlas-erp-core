package support

import (
	"path/filepath"
	"testing"

	"github.com/rgomids/atlas-erp-core/internal/shared/config"
	"github.com/rgomids/atlas-erp-core/internal/shared/postgres"
)

func RunMigrations(t testing.TB, databaseConfig config.DatabaseConfig) {
	t.Helper()

	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("resolve migrations path: %v", err)
	}

	if err := postgres.RunMigrations("up", migrationsPath, databaseConfig.ConnectionString()); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
}
