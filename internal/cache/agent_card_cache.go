// Package cache provides caching for A2A agent cards.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"radgateway/internal/a2a"
)

// TypedAgentCardCache defines high-level cache operations for A2A Agent Cards.
type TypedAgentCardCache interface {
	// Get retrieves an agent card by ID from cache.
	Get(ctx context.Context, id string) (*a2a.AgentCard, error)
	// Set stores an agent card in cache.
	Set(ctx context.Context, id string, card *a2a.AgentCard, ttl time.Duration) error
	// Delete removes an agent card from cache.
	Delete(ctx context.Context, id string) error

	// GetBySkill retrieves agent cards by skill ID.
	GetBySkill(ctx context.Context, skillID string) ([]a2a.AgentCard, error)
	// SetBySkill stores agent cards by skill ID.
	SetBySkill(ctx context.Context, skillID string, cards []a2a.AgentCard, ttl time.Duration) error
	// DeleteBySkill removes the skill-based cache entry.
	DeleteBySkill(ctx context.Context, skillID string) error

	// GetByName retrieves agent cards by name pattern.
	GetByName(ctx context.Context, name string) ([]a2a.AgentCard, error)
	// SetByName stores agent cards by name.
	SetByName(ctx context.Context, name string, cards []a2a.AgentCard, ttl time.Duration) error

	// InvalidateCard removes a card and its associated cache entries.
	InvalidateCard(ctx context.Context, id string) error
	// InvalidatePattern removes all cache entries matching a pattern.
	InvalidatePattern(ctx context.Context, pattern string) error
}

// typedAgentCardCache implements TypedAgentCardCache.
type typedAgentCardCache struct {
	cache      Cache
	defaultTTL time.Duration
}

// NewTypedAgentCardCache creates a new TypedAgentCardCache instance.
func NewTypedAgentCardCache(cache Cache, defaultTTL time.Duration) TypedAgentCardCache {
	if defaultTTL == 0 {
		defaultTTL = 10 * time.Minute // Agent cards change less frequently
	}
	return &typedAgentCardCache{
		cache:      cache,
		defaultTTL: defaultTTL,
	}
}

// cacheKeyCard returns the cache key for a single agent card.
// Format: agent_card:{id}
func (t *typedAgentCardCache) cacheKeyCard(id string) string {
	return fmt.Sprintf("agent_card:%s", id)
}

// cacheKeySkill returns the cache key for skill-based lookup.
// Format: agent_cards:skill:{skill_id}
func (t *typedAgentCardCache) cacheKeySkill(skillID string) string {
	return fmt.Sprintf("agent_cards:skill:%s", skillID)
}

// cacheKeyName returns the cache key for name-based lookup.
// Format: agent_cards:name:{name}
func (t *typedAgentCardCache) cacheKeyName(name string) string {
	return fmt.Sprintf("agent_cards:name:%s", name)
}

// Get retrieves an agent card by ID from cache.
func (t *typedAgentCardCache) Get(ctx context.Context, id string) (*a2a.AgentCard, error) {
	key := t.cacheKeyCard(id)
	data, err := t.cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent card from cache: %w", err)
	}
	if data == nil {
		return nil, nil // Cache miss
	}

	var card a2a.AgentCard
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent card: %w", err)
	}
	return &card, nil
}

// Set stores an agent card in cache.
func (t *typedAgentCardCache) Set(ctx context.Context, id string, card *a2a.AgentCard, ttl time.Duration) error {
	if ttl == 0 {
		ttl = t.defaultTTL
	}

	key := t.cacheKeyCard(id)
	data, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("failed to marshal agent card: %w", err)
	}

	if err := t.cache.Set(ctx, key, data, ttl); err != nil {
		return fmt.Errorf("failed to set agent card in cache: %w", err)
	}
	return nil
}

// Delete removes an agent card from cache.
func (t *typedAgentCardCache) Delete(ctx context.Context, id string) error {
	key := t.cacheKeyCard(id)
	if err := t.cache.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete agent card from cache: %w", err)
	}
	return nil
}

// GetBySkill retrieves agent cards by skill ID.
func (t *typedAgentCardCache) GetBySkill(ctx context.Context, skillID string) ([]a2a.AgentCard, error) {
	key := t.cacheKeySkill(skillID)
	data, err := t.cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent cards by skill from cache: %w", err)
	}
	if data == nil {
		return nil, nil // Cache miss
	}

	var cards []a2a.AgentCard
	if err := json.Unmarshal(data, &cards); err != nil {
		return nil, fmt.Errorf("failed to unmarshal skill agent cards: %w", err)
	}
	return cards, nil
}

// SetBySkill stores agent cards by skill ID.
func (t *typedAgentCardCache) SetBySkill(ctx context.Context, skillID string, cards []a2a.AgentCard, ttl time.Duration) error {
	if ttl == 0 {
		ttl = t.defaultTTL
	}

	key := t.cacheKeySkill(skillID)
	data, err := json.Marshal(cards)
	if err != nil {
		return fmt.Errorf("failed to marshal skill agent cards: %w", err)
	}

	if err := t.cache.Set(ctx, key, data, ttl); err != nil {
		return fmt.Errorf("failed to set skill agent cards in cache: %w", err)
	}
	return nil
}

// DeleteBySkill removes the skill-based cache entry.
func (t *typedAgentCardCache) DeleteBySkill(ctx context.Context, skillID string) error {
	key := t.cacheKeySkill(skillID)
	if err := t.cache.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete skill agent cards from cache: %w", err)
	}
	return nil
}

// GetByName retrieves agent cards by name pattern.
func (t *typedAgentCardCache) GetByName(ctx context.Context, name string) ([]a2a.AgentCard, error) {
	key := t.cacheKeyName(name)
	data, err := t.cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent cards by name from cache: %w", err)
	}
	if data == nil {
		return nil, nil // Cache miss
	}

	var cards []a2a.AgentCard
	if err := json.Unmarshal(data, &cards); err != nil {
		return nil, fmt.Errorf("failed to unmarshal name agent cards: %w", err)
	}
	return cards, nil
}

// SetByName stores agent cards by name.
func (t *typedAgentCardCache) SetByName(ctx context.Context, name string, cards []a2a.AgentCard, ttl time.Duration) error {
	if ttl == 0 {
		ttl = t.defaultTTL
	}

	key := t.cacheKeyName(name)
	data, err := json.Marshal(cards)
	if err != nil {
		return fmt.Errorf("failed to marshal name agent cards: %w", err)
	}

	if err := t.cache.Set(ctx, key, data, ttl); err != nil {
		return fmt.Errorf("failed to set name agent cards in cache: %w", err)
	}
	return nil
}

// InvalidateCard removes a card and its associated cache entries.
func (t *typedAgentCardCache) InvalidateCard(ctx context.Context, id string) error {
	// Delete the individual card
	if err := t.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate skill-based patterns that might include this card
	if err := t.cache.DeletePattern(ctx, "agent_cards:skill:*"); err != nil {
		return fmt.Errorf("failed to invalidate skill cache: %w", err)
	}

	// Invalidate name-based patterns
	if err := t.cache.DeletePattern(ctx, "agent_cards:name:*"); err != nil {
		return fmt.Errorf("failed to invalidate name cache: %w", err)
	}

	return nil
}

// InvalidatePattern removes all cache entries matching a pattern.
func (t *typedAgentCardCache) InvalidatePattern(ctx context.Context, pattern string) error {
	if err := t.cache.DeletePattern(ctx, pattern); err != nil {
		return fmt.Errorf("failed to invalidate cache pattern: %w", err)
	}
	return nil
}

// Ensure typedAgentCardCache implements TypedAgentCardCache interface
var _ TypedAgentCardCache = (*typedAgentCardCache)(nil)
