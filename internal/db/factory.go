// Package db provides database interfaces and implementations for RAD Gateway.
package db

import (
	"fmt"
	"log"
)

// New creates a new database connection based on the driver specified in config.
// Supports "sqlite" and "postgres" drivers.
func New(config Config) (Database, error) {
	switch config.Driver {
	case "sqlite", "sqlite3":
		return NewSQLite(config)
	case "postgres", "postgresql":
		return NewPostgres(config)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", config.Driver)
	}
}

// NewWithFallback attempts to connect to PostgreSQL, falling back to SQLite if unavailable.
// This is useful for deployments where PostgreSQL may not be ready at startup.
func NewWithFallback(config Config) (Database, string, error) {
	// If PostgreSQL is requested, try it first
	if config.Driver == "postgres" || config.Driver == "postgresql" {
		db, err := NewPostgres(config)
		if err == nil {
			return db, "postgres", nil
		}
		log.Printf("PostgreSQL connection failed: %v. Falling back to SQLite...", err)

		// Fall back to SQLite
		sqliteConfig := Config{
			Driver:          "sqlite",
			DSN:             "radgateway_fallback.db",
			MaxOpenConns:    1,
			MaxIdleConns:    1,
			ConnMaxLifetime: config.ConnMaxLifetime,
			ConnMaxIdleTime: config.ConnMaxIdleTime,
		}
		sqliteDB, err := NewSQLite(sqliteConfig)
		if err != nil {
			return nil, "", fmt.Errorf("both PostgreSQL and SQLite fallback failed: %w", err)
		}
		return sqliteDB, "sqlite", nil
	}

	// For SQLite or other drivers, use normal flow
	db, err := New(config)
	return db, config.Driver, err
}

// MustNew creates a new database connection or panics on error.
func MustNew(config Config) Database {
	db, err := New(config)
	if err != nil {
		panic(err)
	}
	return db
}
