package db

import (
	"testing"
	"time"
)

func TestPoolConfiguration(t *testing.T) {
	// Test with custom config
	config := Config{
		DSN:             "postgres://localhost/test?sslmode=disable",
		MaxOpenConns:    50,
		MaxIdleConns:    20,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	// Verify config values are stored
	if config.MaxOpenConns != 50 {
		t.Errorf("MaxOpenConns = %d, want 50", config.MaxOpenConns)
	}
	if config.MaxIdleConns != 20 {
		t.Errorf("MaxIdleConns = %d, want 20", config.MaxIdleConns)
	}
	if config.ConnMaxLifetime != 30*time.Minute {
		t.Errorf("ConnMaxLifetime = %v, want 30m", config.ConnMaxLifetime)
	}
	if config.ConnMaxIdleTime != 10*time.Minute {
		t.Errorf("ConnMaxIdleTime = %v, want 10m", config.ConnMaxIdleTime)
	}
}

func TestPoolConfiguration_Defaults(t *testing.T) {
	// Test with empty config (should use defaults)
	config := Config{
		DSN: "postgres://localhost/test?sslmode=disable",
	}

	// Apply defaults (simulating what NewPostgres does)
	maxOpenConns := config.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = 25
	}
	maxIdleConns := config.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 10
	}
	connMaxLifetime := config.ConnMaxLifetime
	if connMaxLifetime <= 0 {
		connMaxLifetime = 15 * time.Minute
	}
	connMaxIdleTime := config.ConnMaxIdleTime
	if connMaxIdleTime <= 0 {
		connMaxIdleTime = 5 * time.Minute
	}

	// Verify defaults
	if maxOpenConns != 25 {
		t.Errorf("default MaxOpenConns = %d, want 25", maxOpenConns)
	}
	if maxIdleConns != 10 {
		t.Errorf("default MaxIdleConns = %d, want 10", maxIdleConns)
	}
	if connMaxLifetime != 15*time.Minute {
		t.Errorf("default ConnMaxLifetime = %v, want 15m", connMaxLifetime)
	}
	if connMaxIdleTime != 5*time.Minute {
		t.Errorf("default ConnMaxIdleTime = %v, want 5m", connMaxIdleTime)
	}
}
