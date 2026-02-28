package cache

import (
	"context"
	"testing"
	"time"
)

func TestTypedAPIKeyCache_GetSet(t *testing.T) {
	memCache := NewMemoryCacheWithPrefix("test:")
	apiKeyCache := NewTypedAPIKeyCache(memCache, 5*time.Minute)

	ctx := context.Background()
	keyHash := "test_hash_123"

	// Initially cache miss
	info, err := apiKeyCache.Get(ctx, keyHash)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if info != nil {
		t.Error("Expected cache miss, got hit")
	}

	// Set value
	testInfo := &APIKeyInfo{
		Name:      "test-key",
		KeyHash:   keyHash,
		ProjectID: "proj-123",
		Role:      "admin",
		Valid:     true,
	}

	err = apiKeyCache.Set(ctx, keyHash, testInfo, 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get should hit now
	info, err = apiKeyCache.Get(ctx, keyHash)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if info == nil {
		t.Fatal("Expected cache hit, got miss")
	}
	if info.Name != testInfo.Name {
		t.Errorf("Expected name %q, got %q", testInfo.Name, info.Name)
	}
	if info.ProjectID != testInfo.ProjectID {
		t.Errorf("Expected project ID %q, got %q", testInfo.ProjectID, info.ProjectID)
	}
}

func TestTypedAPIKeyCache_Delete(t *testing.T) {
	memCache := NewMemoryCacheWithPrefix("test:")
	apiKeyCache := NewTypedAPIKeyCache(memCache, 5*time.Minute)

	ctx := context.Background()
	keyHash := "test_hash_delete"

	// Set value
	testInfo := &APIKeyInfo{
		Name:    "delete-key",
		KeyHash: keyHash,
		Valid:   true,
	}

	_ = apiKeyCache.Set(ctx, keyHash, testInfo, 0)

	// Delete it
	err := apiKeyCache.Delete(ctx, keyHash)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Should be gone
	info, err := apiKeyCache.Get(ctx, keyHash)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if info != nil {
		t.Error("Expected cache miss after delete")
	}
}

func TestTypedAPIKeyCache_Expiration(t *testing.T) {
	memCache := NewMemoryCacheWithPrefix("test:")
	apiKeyCache := NewTypedAPIKeyCache(memCache, 5*time.Minute)

	ctx := context.Background()
	keyHash := "test_hash_expire"

	// Set with very short TTL
	testInfo := &APIKeyInfo{
		Name:    "expire-key",
		KeyHash: keyHash,
		Valid:   true,
	}

	err := apiKeyCache.Set(ctx, keyHash, testInfo, 1*time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Wait for expiration
	time.Sleep(50 * time.Millisecond)

	// Should be expired
	info, err := apiKeyCache.Get(ctx, keyHash)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if info != nil {
		t.Error("Expected cache miss after expiration")
	}
}

func TestAPIKeyInfo_IsExpired(t *testing.T) {
	// Not expired
	info := &APIKeyInfo{
		Name:    "test",
		Valid:   true,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if info.IsExpired() {
		t.Error("Expected not expired")
	}

	// Expired
	info.ExpiresAt = time.Now().Add(-time.Hour)
	if !info.IsExpired() {
		t.Error("Expected expired")
	}

	// No expiration
	info.ExpiresAt = time.Time{}
	if info.IsExpired() {
		t.Error("Expected not expired when ExpiresAt is zero")
	}
}

func TestTypedAPIKeyCache_InvalidateByProject(t *testing.T) {
	memCache := NewMemoryCacheWithPrefix("test:")
	apiKeyCache := NewTypedAPIKeyCache(memCache, 5*time.Minute)

	ctx := context.Background()

	// Set multiple keys
	for i := 0; i < 3; i++ {
		info := &APIKeyInfo{
			Name:      "key",
			KeyHash:   "hash_" + string(rune('a'+i)),
			ProjectID: "proj-123",
			Valid:     true,
		}
		_ = apiKeyCache.Set(ctx, info.KeyHash, info, 0)
	}

	// Invalidate by project
	err := apiKeyCache.InvalidateByProject(ctx, "proj-123")
	if err != nil {
		t.Fatalf("InvalidateByProject failed: %v", err)
	}

	// All keys should be gone
	for i := 0; i < 3; i++ {
		info, _ := apiKeyCache.Get(ctx, "hash_"+string(rune('a'+i)))
		if info != nil {
			t.Error("Expected cache miss after project invalidation")
		}
	}
}
