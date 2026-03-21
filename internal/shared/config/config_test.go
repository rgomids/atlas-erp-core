package config

import (
	"strings"
	"testing"
)

func TestLoadFromEnvUsesDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := loadFromEnv(func(key string) (string, bool) {
		values := map[string]string{
			"APP_PORT":    "8080",
			"DB_HOST":     "localhost",
			"DB_PORT":     "5432",
			"DB_USER":     "atlas",
			"DB_PASSWORD": "atlas",
			"DB_NAME":     "atlas",
		}

		value, ok := values[key]

		return value, ok
	})
	if err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}

	if cfg.App.Name != "atlas-erp-core" {
		t.Fatalf("expected default app name, got %q", cfg.App.Name)
	}

	if cfg.App.Env != "local" {
		t.Fatalf("expected default app env, got %q", cfg.App.Env)
	}

	if cfg.App.LogLevel != "info" {
		t.Fatalf("expected default log level, got %q", cfg.App.LogLevel)
	}

	if cfg.App.CorrelationIDHeader != "X-Correlation-ID" {
		t.Fatalf("expected default correlation header, got %q", cfg.App.CorrelationIDHeader)
	}
}

func TestLoadFromEnvFailsWhenRequiredValueIsMissing(t *testing.T) {
	t.Parallel()

	_, err := loadFromEnv(func(key string) (string, bool) {
		values := map[string]string{
			"APP_PORT": "8080",
			"DB_PORT":  "5432",
			"DB_USER":  "atlas",
			"DB_NAME":  "atlas",
		}

		value, ok := values[key]

		return value, ok
	})
	if err == nil {
		t.Fatal("expected loadFromEnv to fail for missing database variables")
	}
}

func TestConnectionStringBuildsExpectedURL(t *testing.T) {
	t.Parallel()

	database := DatabaseConfig{
		Host:     "db",
		Port:     5432,
		User:     "atlas",
		Password: "secret",
		Name:     "core",
	}

	connection := database.ConnectionString()
	if !strings.Contains(connection, "postgres://atlas:secret@db:5432/core") {
		t.Fatalf("expected connection string to include credentials and host, got %q", connection)
	}

	if !strings.Contains(connection, "sslmode=disable") {
		t.Fatalf("expected connection string to include sslmode, got %q", connection)
	}
}
