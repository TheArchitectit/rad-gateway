// Command seed provides a CLI tool for database seeding.
//
// Usage:
//
//	go run cmd/seed/main.go [flags]
//
// Flags:
//
//	--scenario string    Seed scenario to run (default "development")
//	--truncate           Truncate all tables before seeding
//	--dry-run            Show what would be inserted without modifying database
//	--workspace string   Seed only specific workspace ID
//	--db-driver string   Database driver (sqlite, postgres) (default "sqlite")
//	--db-dsn string      Database connection string
//
// Examples:
//
//	# Seed with development data (default)
//	go run cmd/seed/main.go
//
//	# Seed with production-like data, truncating first
//	go run cmd/seed/main.go --scenario production-like --truncate
//
//	# Dry run to see what would be created
//	go run cmd/seed/main.go --scenario stress-test --dry-run
//
//	# Seed with demo data
//	go run cmd/seed/main.go --scenario demo
//
// Available Scenarios:
//
//	development     - Rich development dataset (default)
//	production-like - Production-like dataset with realistic usage
//	stress-test     - Large dataset for performance testing
//	demo            - Polished demo environment
//	minimal         - Minimal required data only
//
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"radgateway/internal/db"
	"radgateway/internal/db/seeds"
)

var (
	scenario   = flag.String("scenario", "development", "Seed scenario to run")
	truncate   = flag.Bool("truncate", false, "Truncate all tables before seeding")
	dryRun     = flag.Bool("dry-run", false, "Show what would be inserted without modifying database")
	workspace  = flag.String("workspace", "", "Seed only specific workspace ID")
	dbDriver   = flag.String("db-driver", "sqlite", "Database driver (sqlite, postgres)")
	dbDSN      = flag.String("db-dsn", "", "Database connection string (default: radgateway.db for sqlite)")
	showScenarios = flag.Bool("scenarios", false, "List available scenarios and exit")
	showHelp   = flag.Bool("help", false, "Show help and exit")
)

func main() {
	flag.Parse()

	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	if *showScenarios {
		printScenarios()
		os.Exit(0)
	}

	// Set up logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Validate scenario
	if seeds.FindScenario(*scenario) == nil {
		logger.Error("Unknown scenario", "scenario", *scenario)
		fmt.Println("\nAvailable scenarios:")
		printScenarios()
		os.Exit(1)
	}

	// Set up database connection
	dsn := *dbDSN
	if dsn == "" {
		if *dbDriver == "sqlite" {
			dsn = "radgateway.db"
		} else {
			logger.Error("Database DSN required for non-SQLite drivers")
			os.Exit(1)
		}
	}

	logger.Info("Connecting to database",
		"driver", *dbDriver,
		"dsn", maskDSN(dsn),
	)

	// Create database connection
	database, err := db.New(db.Config{
		Driver: *dbDriver,
		DSN:    dsn,
	})
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := database.Ping(ctx); err != nil {
		logger.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}
	logger.Info("Database connection established")

	// Run migrations if needed
	logger.Info("Running migrations...")
	if err := database.RunMigrations(); err != nil {
		logger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}
	logger.Info("Migrations completed")

	// Create seeder and run seeding
	seeder := seeds.NewSeeder(database, logger)

	opts := seeds.SeedOptions{
		Scenario:    *scenario,
		Truncate:    *truncate,
		DryRun:      *dryRun,
		WorkspaceID: *workspace,
	}

	startTime := time.Now()
	if err := seeder.Seed(ctx, opts); err != nil {
		logger.Error("Seeding failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Seeding completed",
		"scenario", *scenario,
		"duration", time.Since(startTime),
	)
}

func printHelp() {
	fmt.Println(`RAD Gateway Database Seeding Tool

Usage:
  go run cmd/seed/main.go [flags]

Flags:`)
	flag.PrintDefaults()
	fmt.Println(`
Examples:
  # Seed with development data (default)
  go run cmd/seed/main.go

  # Seed with production-like data, truncating first
  go run cmd/seed/main.go --scenario production-like --truncate

  # Dry run to see what would be created
  go run cmd/seed/main.go --scenario stress-test --dry-run

  # Seed with demo data for presentations
  go run cmd/seed/main.go --scenario demo --db-driver postgres --db-dsn "postgres://user:pass@localhost/radgateway"

  # List available scenarios
  go run cmd/seed/main.go --scenarios`)
}

func printScenarios() {
	fmt.Println("\nAvailable Scenarios:")
	fmt.Println("--------------------")
	for name, desc := range seeds.ScenarioNames() {
		fmt.Printf("  %-15s - %s\n", name, desc)
	}
	fmt.Println()
}

func maskDSN(dsn string) string {
	// Simple masking for display purposes
	// In production, consider using url.Parse for proper URL handling
	if len(dsn) > 20 {
		return dsn[:20] + "..."
	}
	return dsn
}
