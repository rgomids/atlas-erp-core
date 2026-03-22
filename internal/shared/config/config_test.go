package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

	if cfg.Payments.GatewayTimeout != 2*time.Second {
		t.Fatalf("expected default gateway timeout, got %s", cfg.Payments.GatewayTimeout)
	}

	if cfg.Observability.TraceEndpoint != "" {
		t.Fatalf("expected empty default trace endpoint, got %q", cfg.Observability.TraceEndpoint)
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

func TestLoadFromEnvRejectsUnsupportedAppEnv(t *testing.T) {
	t.Parallel()

	_, err := loadFromEnv(func(key string) (string, bool) {
		values := map[string]string{
			"APP_PORT":    "8080",
			"APP_ENV":     "qa",
			"DB_HOST":     "localhost",
			"DB_PORT":     "5432",
			"DB_USER":     "atlas",
			"DB_PASSWORD": "atlas",
			"DB_NAME":     "atlas",
		}

		value, ok := values[key]
		return value, ok
	})
	if err == nil {
		t.Fatal("expected unsupported APP_ENV to fail")
	}
}

func TestNewEnvLookupAppliesEnvSpecificOverlay(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			t.Fatalf("restore working dir: %v", chdirErr)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("change working dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, ".env"), []byte("APP_ENV=test\nDB_HOST=base-host\n"), 0o600); err != nil {
		t.Fatalf("write base env: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, ".env.test"), []byte("DB_HOST=overlay-host\nPAYMENT_GATEWAY_TIMEOUT_MS=3500\n"), 0o600); err != nil {
		t.Fatalf("write env overlay: %v", err)
	}

	lookup, err := newEnvLookup(func(string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("new env lookup: %v", err)
	}

	host, ok := lookup("DB_HOST")
	if !ok || host != "overlay-host" {
		t.Fatalf("expected DB_HOST from overlay, got %q", host)
	}

	timeout, ok := lookup("PAYMENT_GATEWAY_TIMEOUT_MS")
	if !ok || timeout != "3500" {
		t.Fatalf("expected PAYMENT_GATEWAY_TIMEOUT_MS from overlay, got %q", timeout)
	}
}

func TestLoadFromEnvIncludesObservabilityTraceEndpoint(t *testing.T) {
	t.Parallel()

	cfg, err := loadFromEnv(func(key string) (string, bool) {
		values := map[string]string{
			"APP_PORT":                    "8080",
			"DB_HOST":                     "localhost",
			"DB_PORT":                     "5432",
			"DB_USER":                     "atlas",
			"DB_PASSWORD":                 "atlas",
			"DB_NAME":                     "atlas",
			"OTEL_EXPORTER_OTLP_ENDPOINT": "http://jaeger:4318",
		}

		value, ok := values[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}

	if cfg.Observability.TraceEndpoint != "http://jaeger:4318" {
		t.Fatalf("expected trace endpoint to be loaded, got %q", cfg.Observability.TraceEndpoint)
	}
}
