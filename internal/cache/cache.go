// Package cache provides caching abstractions for RAD Gateway.
package cache

import (
	"context"
	"time"
)

// Cache defines a generic cache interface.
type Cache interface {
	// Get retrieves a value from the cache.
	Get(ctx context.Context, key string) ([]byte, error)
	// Set stores a value in the cache with a TTL.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// Delete removes a value from the cache.
	Delete(ctx context.Context, key string) error
	// DeletePattern removes all values matching a pattern.
	DeletePattern(ctx context.Context, pattern string) error
	// Ping checks the cache connection.
	Ping(ctx context.Context) error
	// Close closes the cache connection.
	Close() error
}

// Config holds cache configuration.
type Config struct {
	// Address is the cache server address (e.g., "localhost:6379").
	Address string
	// Password for cache authentication.
	Password string
	// Database number to use.
	Database int
	// DefaultTTL is the default time-to-live for cache entries.
	DefaultTTL time.Duration
	// KeyPrefix is prepended to all cache keys.
	KeyPrefix string
}

// DefaultConfig returns a default cache configuration.
func DefaultConfig() Config {
	return Config{
		Address:    "localhost:6379",
		Password:   "",
		Database:   0,
		DefaultTTL: 5 * time.Minute,
		KeyPrefix:  "rad:",
	}
}
