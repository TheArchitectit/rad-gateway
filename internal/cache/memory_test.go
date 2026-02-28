// Package cache provides tests for memory cache implementation.
package cache

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCache_SetAndGet(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()
	defer c.Close()

	// Test basic set and get
	key := "test-key"
	value := []byte("test-value")

	err := c.Set(ctx, key, value, time.Hour)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := c.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("Get returned wrong value: got %q, want %q", string(got), string(value))
	}
}

func TestMemoryCache_GetNonExistent(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()
	defer c.Close()

	got, err := c.Get(ctx, "non-existent-key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got != nil {
		t.Errorf("Expected nil for non-existent key, got %q", string(got))
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()
	defer c.Close()

	key := "delete-key"
	value := []byte("delete-value")

	// Set value
	c.Set(ctx, key, value, time.Hour)

	// Delete it
	err := c.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Should be gone
	got, _ := c.Get(ctx, key)
	if got != nil {
		t.Error("Key still exists after delete")
	}
}

func TestMemoryCache_Expiration(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()
	defer c.Close()

	key := "expire-key"
	value := []byte("expire-value")

	// Set with very short TTL
	c.Set(ctx, key, value, 1*time.Millisecond)

	// Should exist immediately
	got, _ := c.Get(ctx, key)
	if got == nil {
		t.Error("Key should exist immediately after set")
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should be expired
	got, _ = c.Get(ctx, key)
	if got != nil {
		t.Error("Key should be expired")
	}
}

func TestMemoryCache_CleanupExpired(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()
	defer c.Close()

	// Set some expired entries
	for i := 0; i < 10; i++ {
		c.Set(ctx, "expired-key", []byte("value"), 1*time.Nanosecond)
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Run cleanup
	err := c.CleanupExpired(ctx)
	if err != nil {
		t.Fatalf("CleanupExpired failed: %v", err)
	}
}

func TestMemoryCache_Multi(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()
	defer c.Close()

	// Set multiple
	items := map[string]interface{}{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	err := c.SetMulti(ctx, items, time.Hour)
	if err != nil {
		t.Fatalf("SetMulti failed: %v", err)
	}

	// Get multiple
	keys := []string{"key1", "key2", "key3", "key4"}
	got, err := c.GetMulti(ctx, keys)
	if err != nil {
		t.Fatalf("GetMulti failed: %v", err)
	}

	if len(got) != 3 {
		t.Errorf("Expected 3 values, got %d", len(got))
	}

	for key, expected := range items {
		if string(got[key]) != string(expected.([]byte)) {
			t.Errorf("Wrong value for %s: got %q, want %q", key, string(got[key]), string(expected.([]byte)))
		}
	}
}

func TestMemoryCache_Ping(t *testing.T) {
	ctx := context.Background()
	c := NewMemoryCache()
	defer c.Close()

	err := c.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}
