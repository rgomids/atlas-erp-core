package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
}

type AppConfig struct {
	Name                string
	Env                 string
	Port                int
	LogLevel            string
	CorrelationIDHeader string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

type lookupFunc func(string) (string, bool)

func Load() (Config, error) {
	_ = godotenv.Load()

	return loadFromEnv(os.LookupEnv)
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

	cfg := Config{
		App: AppConfig{
			Name:                optionalString(lookup, "APP_NAME", "atlas-erp-core"),
			Env:                 optionalString(lookup, "APP_ENV", "local"),
			Port:                port,
			LogLevel:            optionalString(lookup, "LOG_LEVEL", "info"),
			CorrelationIDHeader: optionalString(lookup, "CORRELATION_ID_HEADER", "X-Correlation-ID"),
		},
		Database: DatabaseConfig{
			Host:     dbHost,
			Port:     dbPort,
			User:     dbUser,
			Password: dbPassword,
			Name:     dbName,
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

	if cfg.Database.Port <= 0 {
		return errors.New("DB_PORT must be greater than zero")
	}

	if strings.TrimSpace(cfg.App.CorrelationIDHeader) == "" {
		return errors.New("CORRELATION_ID_HEADER cannot be blank")
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
