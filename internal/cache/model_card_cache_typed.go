package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"radgateway/internal/a2a"
)

// ModelCard is an alias for a2a.ModelCard for convenience.
// This ensures cache and a2a packages use the same type.
type ModelCard = a2a.ModelCard

// TypedModelCardCache defines high-level cache operations for A2A Model Cards.
// It handles JSON serialization/deserialization and provides typed methods.
type TypedModelCardCache interface {
	// Get retrieves a model card by ID from cache.
	Get(ctx context.Context, id string) (*ModelCard, error)
	// Set stores a model card in cache.
	Set(ctx context.Context, id string, card *ModelCard, ttl time.Duration) error
	// Delete removes a model card from cache.
	Delete(ctx context.Context, id string) error

	// GetProjectCards retrieves the list of model cards for a project.
	GetProjectCards(ctx context.Context, projectID string) ([]ModelCard, error)
	// SetProjectCards stores the list of model cards for a project.
	SetProjectCards(ctx context.Context, projectID string, cards []ModelCard, ttl time.Duration) error
	// DeleteProjectCards removes the project cards list from cache.
	DeleteProjectCards(ctx context.Context, projectID string) error

	// GetBySkill retrieves model cards by skill ID.
	GetBySkill(ctx context.Context, skillID string) ([]ModelCard, error)
	// SetBySkill stores model cards by skill ID.
	SetBySkill(ctx context.Context, skillID string, cards []ModelCard, ttl time.Duration) error
	// DeleteBySkill removes the skill-based cache entry.
	DeleteBySkill(ctx context.Context, skillID string) error

	// InvalidateCard removes a card and its associated cache entries.
	InvalidateCard(ctx context.Context, id string, projectID string) error
	// InvalidatePattern removes all cache entries matching a pattern.
	InvalidatePattern(ctx context.Context, pattern string) error
}

// typedModelCardCache implements TypedModelCardCache using the Cache interface.
type typedModelCardCache struct {
	cache      Cache
	defaultTTL time.Duration
}

// NewTypedModelCardCache creates a new TypedModelCardCache instance.
func NewTypedModelCardCache(cache Cache, defaultTTL time.Duration) TypedModelCardCache {
	if defaultTTL == 0 {
		defaultTTL = 5 * time.Minute
	}
	return &typedModelCardCache{
		cache:      cache,
		defaultTTL: defaultTTL,
	}
}

// cacheKeyCard returns the cache key for a single model card.
// Format: model_card:{id}
func (t *typedModelCardCache) cacheKeyCard(id string) string {
	return fmt.Sprintf("model_card:%s", id)
}

// cacheKeyProject returns the cache key for project cards.
// Format: model_cards:project:{project_id}
func (t *typedModelCardCache) cacheKeyProject(projectID string) string {
	return fmt.Sprintf("model_cards:project:%s", projectID)
}

// cacheKeySkill returns the cache key for skill-based card lookup.
// Format: model_cards:skill:{skill_id}
func (t *typedModelCardCache) cacheKeySkill(skillID string) string {
	return fmt.Sprintf("model_cards:skill:%s", skillID)
}

// Get retrieves a model card by ID from cache.
func (t *typedModelCardCache) Get(ctx context.Context, id string) (*ModelCard, error) {
	key := t.cacheKeyCard(id)
	data, err := t.cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get model card from cache: %w", err)
	}
	if data == nil {
		return nil, nil // Cache miss
	}

	var card ModelCard
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("failed to unmarshal model card: %w", err)
	}
	return &card, nil
}

// Set stores a model card in cache.
func (t *typedModelCardCache) Set(ctx context.Context, id string, card *ModelCard, ttl time.Duration) error {
	if ttl == 0 {
		ttl = t.defaultTTL
	}
	// Use 5 minute TTL for single cards as per specification
	if ttl == t.defaultTTL {
		ttl = 5 * time.Minute
	}

	key := t.cacheKeyCard(id)
	data, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("failed to marshal model card: %w", err)
	}

	if err := t.cache.Set(ctx, key, data, ttl); err != nil {
		return fmt.Errorf("failed to set model card in cache: %w", err)
	}
	return nil
}

// Delete removes a model card from cache.
func (t *typedModelCardCache) Delete(ctx context.Context, id string) error {
	key := t.cacheKeyCard(id)
	if err := t.cache.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete model card from cache: %w", err)
	}
	return nil
}

// GetProjectCards retrieves the list of model cards for a project.
func (t *typedModelCardCache) GetProjectCards(ctx context.Context, projectID string) ([]ModelCard, error) {
	key := t.cacheKeyProject(projectID)
	data, err := t.cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get project cards from cache: %w", err)
	}
	if data == nil {
		return nil, nil // Cache miss
	}

	var cards []ModelCard
	if err := json.Unmarshal(data, &cards); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project cards: %w", err)
	}
	return cards, nil
}

// SetProjectCards stores the list of model cards for a project.
func (t *typedModelCardCache) SetProjectCards(ctx context.Context, projectID string, cards []ModelCard, ttl time.Duration) error {
	if ttl == 0 {
		ttl = 2 * time.Minute // Shorter TTL for project lists
	}

	key := t.cacheKeyProject(projectID)
	data, err := json.Marshal(cards)
	if err != nil {
		return fmt.Errorf("failed to marshal project cards: %w", err)
	}

	if err := t.cache.Set(ctx, key, data, ttl); err != nil {
		return fmt.Errorf("failed to set project cards in cache: %w", err)
	}
	return nil
}

// DeleteProjectCards removes the project cards list from cache.
func (t *typedModelCardCache) DeleteProjectCards(ctx context.Context, projectID string) error {
	key := t.cacheKeyProject(projectID)
	if err := t.cache.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete project cards from cache: %w", err)
	}
	return nil
}

// GetBySkill retrieves model cards by skill ID.
func (t *typedModelCardCache) GetBySkill(ctx context.Context, skillID string) ([]ModelCard, error) {
	key := t.cacheKeySkill(skillID)
	data, err := t.cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get cards by skill from cache: %w", err)
	}
	if data == nil {
		return nil, nil // Cache miss
	}

	var cards []ModelCard
	if err := json.Unmarshal(data, &cards); err != nil {
		return nil, fmt.Errorf("failed to unmarshal skill cards: %w", err)
	}
	return cards, nil
}

// SetBySkill stores model cards by skill ID.
func (t *typedModelCardCache) SetBySkill(ctx context.Context, skillID string, cards []ModelCard, ttl time.Duration) error {
	if ttl == 0 {
		ttl = 5 * time.Minute // Same as single card TTL
	}

	key := t.cacheKeySkill(skillID)
	data, err := json.Marshal(cards)
	if err != nil {
		return fmt.Errorf("failed to marshal skill cards: %w", err)
	}

	if err := t.cache.Set(ctx, key, data, ttl); err != nil {
		return fmt.Errorf("failed to set skill cards in cache: %w", err)
	}
	return nil
}

// DeleteBySkill removes the skill-based cache entry.
func (t *typedModelCardCache) DeleteBySkill(ctx context.Context, skillID string) error {
	key := t.cacheKeySkill(skillID)
	if err := t.cache.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete skill cards from cache: %w", err)
	}
	return nil
}

// InvalidateCard removes a card and its associated cache entries.
func (t *typedModelCardCache) InvalidateCard(ctx context.Context, id string, projectID string) error {
	// Delete the individual card
	if err := t.Delete(ctx, id); err != nil {
		return err
	}

	// Delete the project list if projectID is provided
	if projectID != "" {
		if err := t.DeleteProjectCards(ctx, projectID); err != nil {
			return err
		}
	}

	// Invalidate skill-based patterns that might include this card
	if err := t.cache.DeletePattern(ctx, "model_cards:skill:*"); err != nil {
		return fmt.Errorf("failed to invalidate skill cache: %w", err)
	}

	return nil
}

// InvalidatePattern removes all cache entries matching a pattern.
func (t *typedModelCardCache) InvalidatePattern(ctx context.Context, pattern string) error {
	if err := t.cache.DeletePattern(ctx, pattern); err != nil {
		return fmt.Errorf("failed to invalidate cache pattern: %w", err)
	}
	return nil
}

// Ensure typedModelCardCache implements TypedModelCardCache interface
var _ TypedModelCardCache = (*typedModelCardCache)(nil)
