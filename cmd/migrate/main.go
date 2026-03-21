package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rgomids/atlas-erp-core/internal/shared/config"
	"github.com/rgomids/atlas-erp-core/internal/shared/postgres"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "migration command failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	direction := flag.String("direction", "up", "migration direction: up or down")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if err := validateDirection(*direction); err != nil {
		return err
	}

	migrationsPath, err := filepath.Abs("migrations")
	if err != nil {
		return err
	}

	if err := postgres.RunMigrations(*direction, migrationsPath, cfg.Database.ConnectionString()); err != nil {
		return err
	}

	return nil
}

func validateDirection(direction string) error {
	if direction == "up" || direction == "down" {
		return nil
	}

	return errors.New("direction must be either up or down")
}
