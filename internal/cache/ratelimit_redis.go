// Package cache provides Redis-based distributed rate limiting.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter provides distributed rate limiting using Redis.
// This allows rate limits to be shared across multiple gateway instances.
type RedisRateLimiter struct {
	client    *redis.Client
	keyPrefix string
}

// NewRedisRateLimiter creates a new Redis-backed rate limiter.
func NewRedisRateLimiter(redisAddr, password string, db int) (*RedisRateLimiter, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: password,
		DB:       db,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisRateLimiter{
		client:    client,
		keyPrefix: "rad:ratelimit:",
	}, nil
}

// Close closes the Redis connection.
func (r *RedisRateLimiter) Close() error {
	return r.client.Close()
}

// CheckRateLimit checks if a request is allowed under the rate limit.
// Returns true if allowed, false if rate limited.
// Uses sliding window algorithm with Redis sorted sets.
func (r *RedisRateLimiter) CheckRateLimit(ctx context.Context, key string, maxRequests int, window time.Duration) (bool, error) {
	now := time.Now().UnixNano()
	windowStart := now - window.Nanoseconds()

	redisKey := r.keyPrefix + key

	// Use Redis pipeline for atomic operations
	pipe := r.client.Pipeline()

	// Remove expired entries (outside the window)
	pipe.ZRemRangeByScore(ctx, redisKey, "0", fmt.Sprintf("%d", windowStart))

	// Count current entries in the window
	countCmd := pipe.ZCard(ctx, redisKey)

	// Add current request
	pipe.ZAdd(ctx, redisKey, redis.Z{
		Score:  float64(now),
		Member: now,
	})

	// Set expiry on the key
	pipe.Expire(ctx, redisKey, window)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("rate limit pipeline failed: %w", err)
	}

	count := countCmd.Val()

	// Allow if under limit
	return count < int64(maxRequests), nil
}

// GetRateLimitStatus returns the current rate limit status for a key.
func (r *RedisRateLimiter) GetRateLimitStatus(ctx context.Context, key string, window time.Duration) (*RateLimitStatus, error) {
	now := time.Now().UnixNano()
	windowStart := now - window.Nanoseconds()

	redisKey := r.keyPrefix + key

	// Remove expired entries and count remaining
	pipe := r.client.Pipeline()
	pipe.ZRemRangeByScore(ctx, redisKey, "0", fmt.Sprintf("%d", windowStart))
	countCmd := pipe.ZCard(ctx, redisKey)
	oldestCmd := pipe.ZRangeWithScores(ctx, redisKey, 0, 0)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("rate limit status pipeline failed: %w", err)
	}

	status := &RateLimitStatus{
		Current: int(countCmd.Val()),
	}

	// Get oldest entry to calculate reset time
	oldest := oldestCmd.Val()
	if len(oldest) > 0 {
		oldestTimestamp := int64(oldest[0].Score)
		status.ResetAt = time.Unix(0, oldestTimestamp).Add(window)
	} else {
		status.ResetAt = time.Now().Add(window)
	}

	return status, nil
}

// ResetRateLimit resets the rate limit for a key.
func (r *RedisRateLimiter) ResetRateLimit(ctx context.Context, key string) error {
	redisKey := r.keyPrefix + key
	if err := r.client.Del(ctx, redisKey).Err(); err != nil {
		return fmt.Errorf("failed to reset rate limit: %w", err)
	}
	return nil
}

// RateLimitStatus holds the current rate limit status.
type RateLimitStatus struct {
	Current   int       `json:"current"`
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	ResetAt   time.Time `json:"reset_at"`
}

// IsAllowed returns true if the request is allowed based on the status.
func (s *RateLimitStatus) IsAllowed() bool {
	return s.Current < s.Limit
}

// RedisRateLimiterConfig holds configuration for the Redis rate limiter.
type RedisRateLimiterConfig struct {
	// Redis connection settings
	Address  string
	Password string
	DB       int
	// Rate limit settings
	DefaultMaxRequests int
	DefaultWindow      time.Duration
}

// DefaultRedisRateLimiterConfig returns default configuration.
func DefaultRedisRateLimiterConfig() RedisRateLimiterConfig {
	return RedisRateLimiterConfig{
		Address:            "localhost:6379",
		Password:           "",
		DB:                 0,
		DefaultMaxRequests: 100,
		DefaultWindow:      time.Minute,
	}
}
