package cache

import (
	"context"
	"os"
	"testing"
	"time"
)

// getTestRedisAddr returns the Redis address for tests
func getTestRedisAddr() string {
	if addr := os.Getenv("REDIS_URL"); addr != "" {
		return addr
	}
	return "localhost:6379"
}

// skipIfNoRedis skips the test if Redis is not available
func skipIfNoRedis(t *testing.T) {
	addr := getTestRedisAddr()
	config := &GoRedisConfig{
		Addr:        addr,
		DialTimeout: 2 * time.Second,
		KeyPrefix:   "test:",
	}

	cache, err := NewGoRedis(config)
	if err != nil {
		t.Skipf("Redis not available at %s: %v", addr, err)
	}
	cache.Close()
}

func TestNewGoRedis(t *testing.T) {
	skipIfNoRedis(t)

	config := &GoRedisConfig{
		Addr:        getTestRedisAddr(),
		PoolSize:    5,
		DialTimeout: 5 * time.Second,
		KeyPrefix:   "test:",
	}

	cache, err := NewGoRedis(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Test ping
	ctx := context.Background()
	if err := cache.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestGoRedisCache_SetAndGet(t *testing.T) {
	skipIfNoRedis(t)

	config := &GoRedisConfig{
		Addr:      getTestRedisAddr(),
		KeyPrefix: "test:",
	}

	cache, err := NewGoRedis(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// Set
	if err := cache.Set(ctx, key, value, time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get
	result, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(result) != string(value) {
		t.Errorf("Expected %q, got %q", value, result)
	}
}

func TestGoRedisCache_GetNonExistent(t *testing.T) {
	skipIfNoRedis(t)

	config := &GoRedisConfig{
		Addr:      getTestRedisAddr(),
		KeyPrefix: "test:",
	}

	cache, err := NewGoRedis(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	key := "non-existent-key"

	result, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil for non-existent key, got %q", result)
	}
}

func TestGoRedisCache_Delete(t *testing.T) {
	skipIfNoRedis(t)

	config := &GoRedisConfig{
		Addr:      getTestRedisAddr(),
		KeyPrefix: "test:",
	}

	cache, err := NewGoRedis(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	key := "delete-test-key"
	value := []byte("delete-test-value")

	// Set
	if err := cache.Set(ctx, key, value, time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify set
	result, _ := cache.Get(ctx, key)
	if result == nil {
		t.Fatal("Key should exist after set")
	}

	// Delete
	if err := cache.Delete(ctx, key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify delete
	result, _ = cache.Get(ctx, key)
	if result != nil {
		t.Error("Key should not exist after delete")
	}
}

func TestGoRedisCache_DeletePattern(t *testing.T) {
	skipIfNoRedis(t)

	config := &GoRedisConfig{
		Addr:      getTestRedisAddr(),
		KeyPrefix: "test:",
	}

	cache, err := NewGoRedis(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Set multiple keys with pattern
	for i := 0; i < 5; i++ {
		key := "pattern:key:" + string(rune('a'+i))
		if err := cache.Set(ctx, key, []byte("value"), time.Minute); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	// Set a key that shouldn't match
	if err := cache.Set(ctx, "pattern:other", []byte("value"), time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Delete pattern
	if err := cache.DeletePattern(ctx, "pattern:key:*"); err != nil {
		t.Fatalf("DeletePattern failed: %v", err)
	}

	// Verify pattern keys are deleted
	for i := 0; i < 5; i++ {
		key := "pattern:key:" + string(rune('a'+i))
		result, _ := cache.Get(ctx, key)
		if result != nil {
			t.Errorf("Key %s should be deleted", key)
		}
	}

	// Verify non-pattern key still exists
	result, _ := cache.Get(ctx, "pattern:other")
	if result == nil {
		t.Error("Key pattern:other should still exist")
	}

	// Cleanup
	cache.Delete(ctx, "pattern:other")
}

func TestGoRedisCache_TTLExpiration(t *testing.T) {
	skipIfNoRedis(t)

	config := &GoRedisConfig{
		Addr:      getTestRedisAddr(),
		KeyPrefix: "test:",
	}

	cache, err := NewGoRedis(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	key := "ttl-test-key"
	value := []byte("ttl-test-value")

	// Set with very short TTL
	if err := cache.Set(ctx, key, value, 100*time.Millisecond); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify immediate get works
	result, _ := cache.Get(ctx, key)
	if result == nil {
		t.Fatal("Key should exist immediately after set")
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Verify key is gone
	result, _ = cache.Get(ctx, key)
	if result != nil {
		t.Error("Key should be expired")
	}
}

func TestGoRedisCache_KeyPrefix(t *testing.T) {
	skipIfNoRedis(t)

	// Create two caches with different prefixes
	config1 := &GoRedisConfig{
		Addr:      getTestRedisAddr(),
		KeyPrefix: "prefix1:",
	}
	config2 := &GoRedisConfig{
		Addr:      getTestRedisAddr(),
		KeyPrefix: "prefix2:",
	}

	cache1, err := NewGoRedis(config1)
	if err != nil {
		t.Fatalf("Failed to create cache1: %v", err)
	}
	defer cache1.Close()

	cache2, err := NewGoRedis(config2)
	if err != nil {
		t.Fatalf("Failed to create cache2: %v", err)
	}
	defer cache2.Close()

	ctx := context.Background()
	key := "shared-key"
	value1 := []byte("value1")
	value2 := []byte("value2")

	// Set same key in both caches
	if err := cache1.Set(ctx, key, value1, time.Minute); err != nil {
		t.Fatalf("Set cache1 failed: %v", err)
	}
	if err := cache2.Set(ctx, key, value2, time.Minute); err != nil {
		t.Fatalf("Set cache2 failed: %v", err)
	}

	// Verify each cache has its own value
	result1, _ := cache1.Get(ctx, key)
	if string(result1) != string(value1) {
		t.Errorf("cache1 expected %q, got %q", value1, result1)
	}

	result2, _ := cache2.Get(ctx, key)
	if string(result2) != string(value2) {
		t.Errorf("cache2 expected %q, got %q", value2, result2)
	}

	// Cleanup
	cache1.Delete(ctx, key)
	cache2.Delete(ctx, key)
}

func TestGoRedisCache_Stats(t *testing.T) {
	skipIfNoRedis(t)

	config := &GoRedisConfig{
		Addr:      getTestRedisAddr(),
		PoolSize:  5,
		KeyPrefix: "test:",
	}

	cache, err := NewGoRedis(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Get stats
	stats := cache.Stats()
	if stats == nil {
		t.Error("Stats should not be nil")
	}
}

func TestDefaultGoRedisConfig(t *testing.T) {
	// Save current env
	originalAddr := os.Getenv("REDIS_URL")
	defer os.Setenv("REDIS_URL", originalAddr)

	// Test with custom value
	os.Setenv("REDIS_URL", "custom:6379")
	os.Setenv("REDIS_POOL_SIZE", "20")

	config := DefaultGoRedisConfig()

	if config.Addr != "custom:6379" {
		t.Errorf("Expected addr 'custom:6379', got %q", config.Addr)
	}
	if config.PoolSize != 20 {
		t.Errorf("Expected pool size 20, got %d", config.PoolSize)
	}
}

func TestGetEnvHelpers(t *testing.T) {
	// Test getEnv
	os.Setenv("TEST_VAR", "test_value")
	if v := getEnv("TEST_VAR", "default"); v != "test_value" {
		t.Errorf("getEnv: expected 'test_value', got %q", v)
	}
	if v := getEnv("NON_EXISTENT_VAR", "default"); v != "default" {
		t.Errorf("getEnv: expected 'default', got %q", v)
	}

	// Test getEnvInt
	os.Setenv("TEST_INT", "42")
	if v := getEnvInt("TEST_INT", 0); v != 42 {
		t.Errorf("getEnvInt: expected 42, got %d", v)
	}
	if v := getEnvInt("NON_EXISTENT_INT", 10); v != 10 {
		t.Errorf("getEnvInt: expected 10, got %d", v)
	}
	os.Setenv("TEST_INT_INVALID", "not_a_number")
	if v := getEnvInt("TEST_INT_INVALID", 5); v != 5 {
		t.Errorf("getEnvInt: expected 5 for invalid, got %d", v)
	}

	// Test getEnvDuration
	os.Setenv("TEST_DURATION", "30s")
	if v := getEnvDuration("TEST_DURATION", 0); v != 30*time.Second {
		t.Errorf("getEnvDuration: expected 30s, got %v", v)
	}
	if v := getEnvDuration("NON_EXISTENT_DURATION", time.Minute); v != time.Minute {
		t.Errorf("getEnvDuration: expected 1m0s, got %v", v)
	}

	// Test getEnvBool
	os.Setenv("TEST_BOOL", "true")
	if v := getEnvBool("TEST_BOOL", false); v != true {
		t.Errorf("getEnvBool: expected true, got %v", v)
	}
	if v := getEnvBool("NON_EXISTENT_BOOL", true); v != true {
		t.Errorf("getEnvBool: expected true, got %v", v)
	}
}
