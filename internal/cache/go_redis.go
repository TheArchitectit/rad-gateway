package cache

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// GoRedisCache implements Cache interface using go-redis library.
// This provides connection pooling, automatic retries, and better performance
type GoRedisCache struct {
	client    *redis.Client
	keyPrefix string
}

// GoRedisConfig holds configuration for go-redis connection
type GoRedisConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolTimeout  time.Duration
	KeyPrefix    string
	UseTLS       bool
}

// DefaultGoRedisConfig returns default configuration from environment variables
func DefaultGoRedisConfig() *GoRedisConfig {
	return &GoRedisConfig{
		Addr:         getEnv("REDIS_URL", "localhost:6379"),
		Password:     getEnv("REDIS_PASSWORD", ""),
		DB:           getEnvInt("REDIS_DB", 0),
		PoolSize:     getEnvInt("REDIS_POOL_SIZE", 10),
		MinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 2),
		MaxRetries:   getEnvInt("REDIS_MAX_RETRIES", 3),
		DialTimeout:  getEnvDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
		ReadTimeout:  getEnvDuration("REDIS_READ_TIMEOUT", 5*time.Second),
		WriteTimeout: getEnvDuration("REDIS_WRITE_TIMEOUT", 5*time.Second),
		PoolTimeout:  getEnvDuration("REDIS_POOL_TIMEOUT", 5*time.Second),
		KeyPrefix:    getEnv("REDIS_KEY_PREFIX", "rad:"),
		UseTLS:       getEnvBool("REDIS_USE_TLS", false),
	}
}

// NewGoRedis creates a new Cache implementation using go-redis
func NewGoRedis(config *GoRedisConfig) (*GoRedisCache, error) {
	if config == nil {
		config = DefaultGoRedisConfig()
	}

	opts := &redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolTimeout:  config.PoolTimeout,
	}

	if config.UseTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), config.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &GoRedisCache{
		client:    client,
		keyPrefix: config.KeyPrefix,
	}, nil
}

// prefixKey adds the configured prefix to a key
func (c *GoRedisCache) prefixKey(key string) string {
	return c.keyPrefix + key
}

// Get retrieves a value from cache
func (c *GoRedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	prefixedKey := c.prefixKey(key)
	result, err := c.client.Get(ctx, prefixedKey).Result()
	if err == redis.Nil {
		return nil, nil // Key not found, not an error
	}
	if err != nil {
		return nil, fmt.Errorf("cache get failed: %w", err)
	}
	return []byte(result), nil
}

// Set stores a value in cache with TTL
func (c *GoRedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	prefixedKey := c.prefixKey(key)
	err := c.client.Set(ctx, prefixedKey, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("cache set failed: %w", err)
	}
	return nil
}

// Delete removes a key from cache
func (c *GoRedisCache) Delete(ctx context.Context, key string) error {
	prefixedKey := c.prefixKey(key)
	err := c.client.Del(ctx, prefixedKey).Err()
	if err != nil {
		return fmt.Errorf("cache delete failed: %w", err)
	}
	return nil
}

// DeletePattern removes all keys matching a pattern using SCAN + DEL
func (c *GoRedisCache) DeletePattern(ctx context.Context, pattern string) error {
	prefixedPattern := c.prefixKey(pattern)
	var cursor uint64
	var keys []string

	for {
		var err error
		keys, cursor, err = c.client.Scan(ctx, cursor, prefixedPattern, 100).Result()
		if err != nil {
			return fmt.Errorf("cache scan failed: %w", err)
		}

		// Delete found keys
		if len(keys) > 0 {
			err = c.client.Del(ctx, keys...).Err()
			if err != nil {
				return fmt.Errorf("cache delete pattern failed: %w", err)
			}
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

// Ping checks the cache connection
func (c *GoRedisCache) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("cache ping failed: %w", err)
	}
	return nil
}

// Close gracefully shuts down the cache connection
func (c *GoRedisCache) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("cache close failed: %w", err)
	}
	return nil
}

// Client returns the underlying redis.Client for advanced operations
func (c *GoRedisCache) Client() *redis.Client {
	return c.client
}

// Stats returns connection pool statistics
func (c *GoRedisCache) Stats() *redis.PoolStats {
	return c.client.PoolStats()
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// Ensure GoRedisCache implements Cache interface
var _ Cache = (*GoRedisCache)(nil)
