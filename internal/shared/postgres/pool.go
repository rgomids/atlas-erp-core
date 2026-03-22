package postgres

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rgomids/atlas-erp-core/internal/shared/config"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

func Open(ctx context.Context, cfg config.DatabaseConfig, telemetry ...*observability.Runtime) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("parse postgres pool config: %w", err)
	}

	if runtime := observability.FromOptional(telemetry...); runtime != nil {
		poolConfig.ConnConfig.Tracer = runtime.QueryTracer()
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

func RunMigrations(direction, migrationsPath, databaseURL string) error {
	migration, err := migrate.New("file://"+filepath.Clean(migrationsPath), databaseURL)
	if err != nil {
		return fmt.Errorf("create migration runner: %w", err)
	}
	defer func() {
		_, _ = migration.Close()
	}()

	switch direction {
	case "up":
		err = migration.Up()
	case "down":
		err = migration.Down()
	default:
		return errors.New("direction must be either up or down")
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
