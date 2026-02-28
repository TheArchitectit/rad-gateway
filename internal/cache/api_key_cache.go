// Package cache provides caching for API keys and authentication.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// APIKeyInfo holds cached API key information.
type APIKeyInfo struct {
	Name      string    `json:"name"`
	KeyHash   string    `json:"key_hash"`
	ProjectID string    `json:"project_id,omitempty"`
	Role      string    `json:"role,omitempty"`
	RateLimit int       `json:"rate_limit,omitempty"`
	Valid     bool      `json:"valid"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// IsExpired checks if the cached API key info has expired.
func (a *APIKeyInfo) IsExpired() bool {
	if a.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(a.ExpiresAt)
}

// TypedAPIKeyCache defines cache operations for API key authentication.
type TypedAPIKeyCache interface {
	// Get retrieves API key info by key hash.
	Get(ctx context.Context, keyHash string) (*APIKeyInfo, error)
	// Set stores API key info in cache.
	Set(ctx context.Context, keyHash string, info *APIKeyInfo, ttl time.Duration) error
	// Delete removes an API key from cache.
	Delete(ctx context.Context, keyHash string) error
	// InvalidateByProject removes all API keys for a project.
	InvalidateByProject(ctx context.Context, projectID string) error
	// InvalidatePattern removes cache entries matching a pattern.
	InvalidatePattern(ctx context.Context, pattern string) error
}

// typedAPIKeyCache implements TypedAPIKeyCache.
type typedAPIKeyCache struct {
	cache      Cache
	defaultTTL time.Duration
}

// NewTypedAPIKeyCache creates a new TypedAPIKeyCache instance.
func NewTypedAPIKeyCache(cache Cache, defaultTTL time.Duration) TypedAPIKeyCache {
	if defaultTTL == 0 {
		defaultTTL = 5 * time.Minute
	}
	return &typedAPIKeyCache{
		cache:      cache,
		defaultTTL: defaultTTL,
	}
}

// cacheKey returns the cache key for an API key.
// Format: api_key:{hash}
func (t *typedAPIKeyCache) cacheKey(keyHash string) string {
	return fmt.Sprintf("api_key:%s", keyHash)
}

// Get retrieves API key info by key hash.
func (t *typedAPIKeyCache) Get(ctx context.Context, keyHash string) (*APIKeyInfo, error) {
	key := t.cacheKey(keyHash)
	data, err := t.cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get api key from cache: %w", err)
	}
	if data == nil {
		return nil, nil // Cache miss
	}

	var info APIKeyInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal api key info: %w", err)
	}

	// Check if expired
	if info.IsExpired() {
		// Delete expired entry
		_ = t.cache.Delete(ctx, key)
		return nil, nil
	}

	return &info, nil
}

// Set stores API key info in cache.
func (t *typedAPIKeyCache) Set(ctx context.Context, keyHash string, info *APIKeyInfo, ttl time.Duration) error {
	if ttl == 0 {
		ttl = t.defaultTTL
	}

	key := t.cacheKey(keyHash)
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal api key info: %w", err)
	}

	if err := t.cache.Set(ctx, key, data, ttl); err != nil {
		return fmt.Errorf("failed to set api key in cache: %w", err)
	}
	return nil
}

// Delete removes an API key from cache.
func (t *typedAPIKeyCache) Delete(ctx context.Context, keyHash string) error {
	key := t.cacheKey(keyHash)
	if err := t.cache.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete api key from cache: %w", err)
	}
	return nil
}

// InvalidateByProject removes all API keys for a project.
func (t *typedAPIKeyCache) InvalidateByProject(ctx context.Context, projectID string) error {
	// Pattern: api_key:* (we can't filter by project in key name)
	// Instead, we use a secondary index: project_keys:{project_id}
	// For now, invalidate all API keys
	if err := t.cache.DeletePattern(ctx, "api_key:*"); err != nil {
		return fmt.Errorf("failed to invalidate project api keys: %w", err)
	}
	return nil
}

// InvalidatePattern removes cache entries matching a pattern.
func (t *typedAPIKeyCache) InvalidatePattern(ctx context.Context, pattern string) error {
	if err := t.cache.DeletePattern(ctx, pattern); err != nil {
		return fmt.Errorf("failed to invalidate cache pattern: %w", err)
	}
	return nil
}

// Ensure typedAPIKeyCache implements TypedAPIKeyCache interface
var _ TypedAPIKeyCache = (*typedAPIKeyCache)(nil)
