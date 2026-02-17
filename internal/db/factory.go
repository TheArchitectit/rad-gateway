// Package db provides database interfaces and implementations for RAD Gateway.
package db

import (
	"fmt"
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

// MustNew creates a new database connection or panics on error.
func MustNew(config Config) Database {
	db, err := New(config)
	if err != nil {
		panic(err)
	}
	return db
}
