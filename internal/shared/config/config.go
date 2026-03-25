package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App           AppConfig
	Database      DatabaseConfig
	Payments      PaymentsConfig
	Observability ObservabilityConfig
}

type AppConfig struct {
	Name                string
	Env                 string
	Port                int
	LogLevel            string
	CorrelationIDHeader string
	FaultProfile        FaultProfile
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

type PaymentsConfig struct {
	GatewayTimeout time.Duration
}

type ObservabilityConfig struct {
	TraceEndpoint string
}

type lookupFunc func(string) (string, bool)

func Load() (Config, error) {
	lookup, err := newEnvLookup(os.LookupEnv)
	if err != nil {
		return Config{}, err
	}

	return loadFromEnv(lookup)
}

func loadFromEnv(lookup lookupFunc) (Config, error) {
	port, err := requiredInt(lookup, "APP_PORT")
	if err != nil {
		return Config{}, err
	}

	dbPort, err := requiredInt(lookup, "DB_PORT")
	if err != nil {
		return Config{}, err
	}

	dbHost, err := requiredString(lookup, "DB_HOST")
	if err != nil {
		return Config{}, err
	}

	dbUser, err := requiredString(lookup, "DB_USER")
	if err != nil {
		return Config{}, err
	}

	dbPassword, err := requiredString(lookup, "DB_PASSWORD")
	if err != nil {
		return Config{}, err
	}

	dbName, err := requiredString(lookup, "DB_NAME")
	if err != nil {
		return Config{}, err
	}

	gatewayTimeoutMS, err := optionalInt(lookup, "PAYMENT_GATEWAY_TIMEOUT_MS", 2000)
	if err != nil {
		return Config{}, err
	}

	faultProfile, err := ParseFaultProfile(optionalString(lookup, "ATLAS_FAULT_PROFILE", string(FaultProfileNone)))
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		App: AppConfig{
			Name:                optionalString(lookup, "APP_NAME", "atlas-erp-core"),
			Env:                 optionalString(lookup, "APP_ENV", "local"),
			Port:                port,
			LogLevel:            optionalString(lookup, "LOG_LEVEL", "info"),
			CorrelationIDHeader: optionalString(lookup, "CORRELATION_ID_HEADER", "X-Correlation-ID"),
			FaultProfile:        faultProfile,
		},
		Database: DatabaseConfig{
			Host:     dbHost,
			Port:     dbPort,
			User:     dbUser,
			Password: dbPassword,
			Name:     dbName,
		},
		Payments: PaymentsConfig{
			GatewayTimeout: time.Duration(gatewayTimeoutMS) * time.Millisecond,
		},
		Observability: ObservabilityConfig{
			TraceEndpoint: optionalString(lookup, "OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		},
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (cfg Config) validate() error {
	if cfg.App.Port <= 0 {
		return errors.New("APP_PORT must be greater than zero")
	}

	if !isSupportedAppEnv(cfg.App.Env) {
		return fmt.Errorf("APP_ENV must be one of local, test, dev, staging or production")
	}

	if cfg.Database.Port <= 0 {
		return errors.New("DB_PORT must be greater than zero")
	}

	if strings.TrimSpace(cfg.App.CorrelationIDHeader) == "" {
		return errors.New("CORRELATION_ID_HEADER cannot be blank")
	}

	if cfg.App.Env == "production" && cfg.App.FaultProfile != FaultProfileNone {
		return errors.New("ATLAS_FAULT_PROFILE must be none when APP_ENV=production")
	}

	if cfg.Payments.GatewayTimeout <= 0 {
		return errors.New("PAYMENT_GATEWAY_TIMEOUT_MS must be greater than zero")
	}

	return nil
}

func (cfg AppConfig) Address() string {
	return fmt.Sprintf(":%d", cfg.Port)
}

func (cfg DatabaseConfig) ConnectionString() string {
	query := url.Values{}
	query.Set("sslmode", "disable")

	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:     cfg.Name,
		RawQuery: query.Encode(),
	}).String()
}

func requiredString(lookup lookupFunc, key string) (string, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s must be set", key)
	}

	return value, nil
}

func requiredInt(lookup lookupFunc, key string) (int, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return 0, fmt.Errorf("%s must be set", key)
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer", key)
	}

	return parsed, nil
}

func optionalString(lookup lookupFunc, key, fallback string) string {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}

func optionalInt(lookup lookupFunc, key string, fallback int) (int, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer", key)
	}

	return parsed, nil
}

func newEnvLookup(process lookupFunc) (lookupFunc, error) {
	baseFileValues, err := readOptionalEnvFile(".env")
	if err != nil {
		return nil, err
	}

	appEnv := firstNonBlank(process, baseFileValues, "APP_ENV")
	if appEnv == "" {
		appEnv = "local"
	}

	merged := map[string]string{}
	mergeEnvValues(merged, baseFileValues)

	envFileValues, err := readOptionalEnvFile(".env." + appEnv)
	if err != nil {
		return nil, err
	}
	mergeEnvValues(merged, envFileValues)

	return func(key string) (string, bool) {
		if value, ok := process(key); ok {
			return value, ok
		}

		value, ok := merged[key]
		return value, ok
	}, nil
}

func mergeEnvValues(target map[string]string, source map[string]string) {
	for key, value := range source {
		target[key] = value
	}
}

func readOptionalEnvFile(path string) (map[string]string, error) {
	values, err := godotenv.Read(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}

		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return values, nil
}

func firstNonBlank(process lookupFunc, fileValues map[string]string, key string) string {
	if value, ok := process(key); ok && strings.TrimSpace(value) != "" {
		return value
	}

	value, ok := fileValues[key]
	if !ok {
		return ""
	}

	return strings.TrimSpace(value)
}

func isSupportedAppEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "local", "test", "dev", "staging", "production":
		return true
	default:
		return false
	}
}
