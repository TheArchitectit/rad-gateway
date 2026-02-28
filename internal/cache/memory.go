// Package cache provides caching abstractions for RAD Gateway.
package cache

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"
)

// MemoryCache implements an in-memory cache.
type MemoryCache struct {
	mu      sync.RWMutex
	data    map[string]*cacheEntry
	keyPrefix string
}

type cacheEntry struct {
	value      []byte
	expiration time.Time
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache() *MemoryCache {
	mc := &MemoryCache{
		data:      make(map[string]*cacheEntry),
		keyPrefix: "rad:",
	}
	// Start cleanup goroutine
	go mc.cleanupExpired()
	return mc
}

// NewMemoryCacheWithPrefix creates a new in-memory cache with a custom key prefix.
func NewMemoryCacheWithPrefix(prefix string) *MemoryCache {
	mc := &MemoryCache{
		data:      make(map[string]*cacheEntry),
		keyPrefix: prefix,
	}
	go mc.cleanupExpired()
	return mc
}

// Get retrieves a value from the cache.
func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[c.prefixedKey(key)]
	if !ok {
		return nil, nil
	}

	if time.Now().After(entry.expiration) {
		return nil, nil
	}

	return entry.value, nil
}

// Set stores a value in the cache with a TTL.
func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[c.prefixedKey(key)] = &cacheEntry{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
	return nil
}

// Delete removes a value from the cache.
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, c.prefixedKey(key))
	return nil
}

// DeletePattern removes all values matching a pattern.
func (c *MemoryCache) DeletePattern(ctx context.Context, pattern string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Convert glob pattern to regex
	regexPattern := "^" + c.keyPrefix + regexp.QuoteMeta(pattern)
	regexPattern = regexp.MustCompile(`\*`).ReplaceAllString(regexPattern, ".*")
	regexPattern = regexp.MustCompile(`\?`).ReplaceAllString(regexPattern, ".")
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	for key := range c.data {
		if regex.MatchString(key) {
			delete(c.data, key)
		}
	}
	return nil
}

// GetByPattern retrieves all values matching a pattern.
func (c *MemoryCache) GetByPattern(ctx context.Context, pattern string) (map[string][]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Convert glob pattern to regex
	regexPattern := "^" + c.keyPrefix + regexp.QuoteMeta(pattern)
	regexPattern = regexp.MustCompile(`\*`).ReplaceAllString(regexPattern, ".*")
	regexPattern = regexp.MustCompile(`\?`).ReplaceAllString(regexPattern, ".")
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	result := make(map[string][]byte)
	for key, entry := range c.data {
		if regex.MatchString(key) && time.Now().Before(entry.expiration) {
			// Remove prefix from key in result
			cleanKey := key
			if c.keyPrefix != "" && len(key) > len(c.keyPrefix) {
				cleanKey = key[len(c.keyPrefix):]
			}
			result[cleanKey] = entry.value
		}
	}
	return result, nil
}

// SetMulti stores multiple values in the cache.
func (c *MemoryCache) SetMulti(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, value := range items {
		var data []byte
		switch v := value.(type) {
		case string:
			data = []byte(v)
		case []byte:
			data = v
		default:
			return fmt.Errorf("unsupported value type for key %s", key)
		}
		c.data[c.prefixedKey(key)] = &cacheEntry{
			value:      data,
			expiration: time.Now().Add(ttl),
		}
	}
	return nil
}

// GetMulti retrieves multiple values from the cache.
func (c *MemoryCache) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string][]byte)
	for _, key := range keys {
		if entry, ok := c.data[c.prefixedKey(key)]; ok && time.Now().Before(entry.expiration) {
			result[key] = entry.value
		}
	}
	return result, nil
}

// CleanupExpired removes expired entries from the cache.
func (c *MemoryCache) CleanupExpired(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.data {
		if now.After(entry.expiration) {
			delete(c.data, key)
		}
	}
	return nil
}

// Ping checks the cache connection (always returns nil for memory cache).
func (c *MemoryCache) Ping(ctx context.Context) error {
	return nil
}

// Close closes the cache.
func (c *MemoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]*cacheEntry)
	return nil
}

// prefixedKey adds the key prefix if not already present.
func (c *MemoryCache) prefixedKey(key string) string {
	if c.keyPrefix != "" && len(key) >= len(c.keyPrefix) && key[:len(c.keyPrefix)] == c.keyPrefix {
		return key
	}
	return c.keyPrefix + key
}

// cleanupExpired periodically removes expired entries.
func (c *MemoryCache) cleanupExpired() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.CleanupExpired(context.Background())
	}
}
