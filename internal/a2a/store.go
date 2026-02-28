// Package a2a provides A2A (Agent-to-Agent) protocol support for RAD Gateway.
package a2a

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ErrNotFound is returned when an agent card is not found in the store.
var ErrNotFound = errors.New("agent card not found")

// ErrInvalidCard is returned when an agent card is invalid.
var ErrInvalidCard = errors.New("invalid agent card")

// Store provides thread-safe in-memory storage for AgentCards.
type Store struct {
	mu    sync.RWMutex
	cards map[string]AgentCard // URL -> AgentCard
}

// NewStore creates a new in-memory store for agent cards.
func NewStore() *Store {
	return &Store{
		cards: make(map[string]AgentCard),
	}
}

// Save stores or updates an agent card in the store.
// If the card is new (URL not in store), it sets CreatedAt.
// Always updates UpdatedAt to the current time.
// Returns an error if the card URL is empty.
func (s *Store) Save(card AgentCard) error {
	if err := validateCard(card); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCard, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	// Check if this is a new card or an update
	if existing, exists := s.cards[card.URL]; exists {
		// Preserve original CreatedAt for updates
		card.CreatedAt = existing.CreatedAt
	} else {
		// Set CreatedAt for new cards
		card.CreatedAt = now
	}

	// Always update UpdatedAt
	card.UpdatedAt = now

	// Set default version if empty
	if card.Version == "" {
		card.Version = "1.0.0"
	}

	s.cards[card.URL] = card
	return nil
}

// Get retrieves an agent card by its URL.
// Returns ErrNotFound if no card exists with the given URL.
func (s *Store) Get(url string) (AgentCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	card, exists := s.cards[url]
	if !exists {
		return AgentCard{}, fmt.Errorf("%w: %s", ErrNotFound, url)
	}

	return card, nil
}

// GetByName retrieves an agent card by its name.
// Performs a case-insensitive search.
// Returns ErrNotFound if no card exists with the given name.
func (s *Store) GetByName(name string) (AgentCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	searchName := strings.ToLower(name)
	for _, card := range s.cards {
		if strings.ToLower(card.Name) == searchName {
			return card, nil
		}
	}

	return AgentCard{}, fmt.Errorf("%w: %s", ErrNotFound, name)
}

// List returns all agent cards in the store.
// Returns an empty slice if no cards exist.
func (s *Store) List() []AgentCard {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cards := make([]AgentCard, 0, len(s.cards))
	for _, card := range s.cards {
		cards = append(cards, card)
	}

	return cards
}

// Delete removes an agent card from the store by its URL.
// Returns ErrNotFound if no card exists with the given URL.
func (s *Store) Delete(url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.cards[url]; !exists {
		return fmt.Errorf("%w: %s", ErrNotFound, url)
	}

	delete(s.cards, url)
	return nil
}

// Count returns the number of agent cards in the store.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.cards)
}

// Clear removes all agent cards from the store.
// Primarily useful for testing.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cards = make(map[string]AgentCard)
}

// validateCard validates that an agent card has required fields.
func validateCard(card AgentCard) error {
	if strings.TrimSpace(card.URL) == "" {
		return errors.New("agent card URL is required")
	}

	return nil
}
