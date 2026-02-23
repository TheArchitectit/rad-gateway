package cache

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestTypedModelCardCache_GetAndSet(t *testing.T) {
	skipIfNoRedis(t)

	// Create cache
	redisConfig := &GoRedisConfig{
		Addr:      getTestRedisAddr(),
		KeyPrefix: "test:typed:",
	}
	baseCache, err := NewGoRedis(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create base cache: %v", err)
	}
	defer baseCache.Close()

	cache := NewTypedModelCardCache(baseCache, 5*time.Minute)
	ctx := context.Background()

	// Create a test model card
	card := &ModelCard{
		ID:          "test-card-1",
		WorkspaceID: "workspace-1",
		Name:        "Test Model",
		Slug:        "test-model",
		Version:     1,
		Status:      "active",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Test Set
	if err := cache.Set(ctx, card.ID, card, 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get
	retrieved, err := cache.Get(ctx, card.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected card, got nil")
	}
	if retrieved.ID != card.ID {
		t.Errorf("Expected ID %q, got %q", card.ID, retrieved.ID)
	}
	if retrieved.Name != card.Name {
		t.Errorf("Expected Name %q, got %q", card.Name, retrieved.Name)
	}

	// Test Get non-existent
	nonExistent, err := cache.Get(ctx, "non-existent-id")
	if err != nil {
		t.Fatalf("Get non-existent failed: %v", err)
	}
	if nonExistent != nil {
		t.Error("Expected nil for non-existent card")
	}

	// Cleanup
	cache.Delete(ctx, card.ID)
}

func TestTypedModelCardCache_Delete(t *testing.T) {
	skipIfNoRedis(t)

	redisConfig := &GoRedisConfig{
		Addr:      getTestRedisAddr(),
		KeyPrefix: "test:typed:",
	}
	baseCache, err := NewGoRedis(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create base cache: %v", err)
	}
	defer baseCache.Close()

	cache := NewTypedModelCardCache(baseCache, 5*time.Minute)
	ctx := context.Background()

	card := &ModelCard{
		ID:          "delete-test-card",
		WorkspaceID: "workspace-1",
		Name:        "Delete Test Model",
		Slug:        "delete-test-model",
		Status:      "active",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Set
	if err := cache.Set(ctx, card.ID, card, 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify exists
	retrieved, _ := cache.Get(ctx, card.ID)
	if retrieved == nil {
		t.Fatal("Card should exist after set")
	}

	// Delete
	if err := cache.Delete(ctx, card.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	retrieved, _ = cache.Get(ctx, card.ID)
	if retrieved != nil {
		t.Error("Card should not exist after delete")
	}
}

func TestTypedModelCardCache_ProjectCards(t *testing.T) {
	skipIfNoRedis(t)

	redisConfig := &GoRedisConfig{
		Addr:      getTestRedisAddr(),
		KeyPrefix: "test:typed:",
	}
	baseCache, err := NewGoRedis(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create base cache: %v", err)
	}
	defer baseCache.Close()

	cache := NewTypedModelCardCache(baseCache, 5*time.Minute)
	ctx := context.Background()
	projectID := "project-123"

	// Create test cards
	cards := []ModelCard{
		{
			ID:          "card-1",
			WorkspaceID: projectID,
			Name:        "Card 1",
			Slug:        "card-1",
			Status:      "active",
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
		{
			ID:          "card-2",
			WorkspaceID: projectID,
			Name:        "Card 2",
			Slug:        "card-2",
			Status:      "active",
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
	}

	// Test SetProjectCards
	if err := cache.SetProjectCards(ctx, projectID, cards, 0); err != nil {
		t.Fatalf("SetProjectCards failed: %v", err)
	}

	// Test GetProjectCards
	retrieved, err := cache.GetProjectCards(ctx, projectID)
	if err != nil {
		t.Fatalf("GetProjectCards failed: %v", err)
	}
	if len(retrieved) != 2 {
		t.Errorf("Expected 2 cards, got %d", len(retrieved))
	}

	// Test DeleteProjectCards
	if err := cache.DeleteProjectCards(ctx, projectID); err != nil {
		t.Fatalf("DeleteProjectCards failed: %v", err)
	}

	// Verify deleted
	retrieved, _ = cache.GetProjectCards(ctx, projectID)
	if retrieved != nil {
		t.Error("Project cards should be nil after delete")
	}
}

func TestTypedModelCardCache_BySkill(t *testing.T) {
	skipIfNoRedis(t)

	redisConfig := &GoRedisConfig{
		Addr:      getTestRedisAddr(),
		KeyPrefix: "test:typed:",
	}
	baseCache, err := NewGoRedis(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create base cache: %v", err)
	}
	defer baseCache.Close()

	cache := NewTypedModelCardCache(baseCache, 5*time.Minute)
	ctx := context.Background()
	skillID := "skill-vision"

	// Create test cards with capabilities in Card JSON
	cardData, _ := json.Marshal(map[string]interface{}{
		"capabilities": []map[string]interface{}{
			{"type": "vision", "enabled": true},
		},
	})

	cards := []ModelCard{
		{
			ID:          "vision-card-1",
			WorkspaceID: "workspace-1",
			Name:        "Vision Model 1",
			Slug:        "vision-model-1",
			Card:        cardData,
			Status:      "active",
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
		{
			ID:          "vision-card-2",
			WorkspaceID: "workspace-1",
			Name:        "Vision Model 2",
			Slug:        "vision-model-2",
			Card:        cardData,
			Status:      "active",
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		},
	}

	// Test SetBySkill
	if err := cache.SetBySkill(ctx, skillID, cards, 0); err != nil {
		t.Fatalf("SetBySkill failed: %v", err)
	}

	// Test GetBySkill
	retrieved, err := cache.GetBySkill(ctx, skillID)
	if err != nil {
		t.Fatalf("GetBySkill failed: %v", err)
	}
	if len(retrieved) != 2 {
		t.Errorf("Expected 2 cards, got %d", len(retrieved))
	}

	// Test DeleteBySkill
	if err := cache.DeleteBySkill(ctx, skillID); err != nil {
		t.Fatalf("DeleteBySkill failed: %v", err)
	}

	// Verify deleted
	retrieved, _ = cache.GetBySkill(ctx, skillID)
	if retrieved != nil {
		t.Error("Skill cards should be nil after delete")
	}
}

func TestTypedModelCardCache_InvalidateCard(t *testing.T) {
	skipIfNoRedis(t)

	redisConfig := &GoRedisConfig{
		Addr:      getTestRedisAddr(),
		KeyPrefix: "test:typed:",
	}
	baseCache, err := NewGoRedis(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create base cache: %v", err)
	}
	defer baseCache.Close()

	cache := NewTypedModelCardCache(baseCache, 5*time.Minute)
	ctx := context.Background()
	cardID := "invalidate-card-1"
	projectID := "project-456"

	// Set up card
	card := &ModelCard{
		ID:          cardID,
		WorkspaceID: projectID,
		Name:        "Invalidate Test Model",
		Slug:        "invalidate-test-model",
		Status:      "active",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Set card and project cards
	if err := cache.Set(ctx, cardID, card, 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	projectCards := []ModelCard{*card}
	if err := cache.SetProjectCards(ctx, projectID, projectCards, 0); err != nil {
		t.Fatalf("SetProjectCards failed: %v", err)
	}

	// Verify both exist
	retrievedCard, _ := cache.Get(ctx, cardID)
	if retrievedCard == nil {
		t.Fatal("Card should exist")
	}
	retrievedProjectCards, _ := cache.GetProjectCards(ctx, projectID)
	if retrievedProjectCards == nil {
		t.Fatal("Project cards should exist")
	}

	// Invalidate card
	if err := cache.InvalidateCard(ctx, cardID, projectID); err != nil {
		t.Fatalf("InvalidateCard failed: %v", err)
	}

	// Verify both are invalidated
	retrievedCard, _ = cache.Get(ctx, cardID)
	if retrievedCard != nil {
		t.Error("Card should be invalidated")
	}
	retrievedProjectCards, _ = cache.GetProjectCards(ctx, projectID)
	if retrievedProjectCards != nil {
		t.Error("Project cards should be invalidated")
	}
}

func TestTypedModelCardCache_CardWithA2AData(t *testing.T) {
	skipIfNoRedis(t)

	redisConfig := &GoRedisConfig{
		Addr:      getTestRedisAddr(),
		KeyPrefix: "test:typed:",
	}
	baseCache, err := NewGoRedis(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create base cache: %v", err)
	}
	defer baseCache.Close()

	cache := NewTypedModelCardCache(baseCache, 5*time.Minute)
	ctx := context.Background()

	// Create a card with A2A-like data
	a2aCardData := map[string]interface{}{
		"schemaVersion": "1.0",
		"name":          "GPT-4",
		"description":   "Advanced language model",
		"capabilities": []map[string]interface{}{
			{"type": "text", "name": "Text Generation", "enabled": true},
			{"type": "code", "name": "Code Generation", "enabled": true},
		},
		"pricing": map[string]interface{}{
			"inputPricePerToken":  0.00003,
			"outputPricePerToken": 0.00006,
			"currency":            "USD",
		},
		"metadata": map[string]interface{}{
			"provider": "openai",
			"family":   "gpt",
		},
	}

	cardData, err := json.Marshal(a2aCardData)
	if err != nil {
		t.Fatalf("Failed to marshal A2A card: %v", err)
	}

	description := "Advanced language model"
	card := &ModelCard{
		ID:          "a2a-card-1",
		WorkspaceID: "workspace-1",
		Name:        "GPT-4",
		Slug:        "gpt-4",
		Description: &description,
		Card:        cardData,
		Version:     1,
		Status:      "active",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Set
	if err := cache.Set(ctx, card.ID, card, 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get and verify
	retrieved, err := cache.Get(ctx, card.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected card, got nil")
	}

	// Verify card data integrity
	if retrieved.ID != card.ID {
		t.Errorf("Expected ID %q, got %q", card.ID, retrieved.ID)
	}

	// Verify A2A card data is preserved
	var parsedCardData map[string]interface{}
	if err := json.Unmarshal(retrieved.Card, &parsedCardData); err != nil {
		t.Fatalf("Failed to unmarshal card data: %v", err)
	}
	if parsedCardData["schemaVersion"] != "1.0" {
		t.Errorf("Expected schema version '1.0', got %v", parsedCardData["schemaVersion"])
	}

	// Cleanup
	cache.Delete(ctx, card.ID)
}

func TestTypedModelCardCache_KeyFormats(t *testing.T) {
	skipIfNoRedis(t)

	redisConfig := &GoRedisConfig{
		Addr:      getTestRedisAddr(),
		KeyPrefix: "test:typed:",
	}
	baseCache, err := NewGoRedis(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create base cache: %v", err)
	}
	defer baseCache.Close()

	// Use the concrete type to access key methods
	typedCache := &typedModelCardCache{
		cache:      baseCache,
		defaultTTL: 5 * time.Minute,
	}

	// Test key formats
	cardKey := typedCache.cacheKeyCard("card-123")
	expectedCardKey := "model_card:card-123"
	if cardKey != expectedCardKey {
		t.Errorf("Expected card key %q, got %q", expectedCardKey, cardKey)
	}

	projectKey := typedCache.cacheKeyProject("project-456")
	expectedProjectKey := "model_cards:project:project-456"
	if projectKey != expectedProjectKey {
		t.Errorf("Expected project key %q, got %q", expectedProjectKey, projectKey)
	}

	skillKey := typedCache.cacheKeySkill("skill-789")
	expectedSkillKey := "model_cards:skill:skill-789"
	if skillKey != expectedSkillKey {
		t.Errorf("Expected skill key %q, got %q", expectedSkillKey, skillKey)
	}
}
