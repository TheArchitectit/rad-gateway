// Package a2a provides A2A (Agent-to-Agent) protocol support for RAD Gateway.
package a2a

import (
	"errors"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	store := NewStore()
	if store == nil {
		t.Fatal("NewStore() returned nil")
	}

	if store.cards == nil {
		t.Error("store.cards map not initialized")
	}

	if store.Count() != 0 {
		t.Errorf("expected empty store, got %d cards", store.Count())
	}
}

func TestStore_Save(t *testing.T) {
	store := NewStore()

	tests := []struct {
		name    string
		card    AgentCard
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid new card",
			card: AgentCard{
				Name:        "Test Agent",
				Description: "A test agent",
				URL:         "https://example.com/agent",
				Version:     "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "card without version gets default",
			card: AgentCard{
				Name: "Agent No Version",
				URL:  "https://example.com/no-version",
			},
			wantErr: false,
		},
		{
			name:    "empty URL fails",
			card:    AgentCard{Name: "Invalid", URL: ""},
			wantErr: true,
			errMsg:  "URL is required",
		},
		{
			name:    "whitespace-only URL fails",
			card:    AgentCard{Name: "Invalid", URL: "   "},
			wantErr: true,
			errMsg:  "URL is required",
		},
		{
			name: "update existing card",
			card: AgentCard{
				Name:        "Updated Agent",
				Description: "Updated description",
				URL:         "https://example.com/agent", // Same URL as first test
				Version:     "2.0.0",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Save(tt.card)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Save() expected error but got none")
					return
				}
				if !errors.Is(err, ErrInvalidCard) {
					t.Errorf("Save() error = %v, want ErrInvalidCard", err)
				}
				return
			}

			if err != nil {
				t.Errorf("Save() unexpected error = %v", err)
			}
		})
	}
}

func TestStore_Save_SetsTimestamps(t *testing.T) {
	store := NewStore()

	beforeSave := time.Now().UTC()

	card := AgentCard{
		Name: "Timestamp Test",
		URL:  "https://example.com/timestamp",
	}

	if err := store.Save(card); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	afterSave := time.Now().UTC()

	saved, err := store.Get("https://example.com/timestamp")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Check CreatedAt is set and within expected range
	if saved.CreatedAt.IsZero() {
		t.Error("CreatedAt not set")
	}
	if saved.CreatedAt.Before(beforeSave) || saved.CreatedAt.After(afterSave) {
		t.Errorf("CreatedAt %v not within expected range [%v, %v]", saved.CreatedAt, beforeSave, afterSave)
	}

	// Check UpdatedAt is set and equals CreatedAt for new cards
	if saved.UpdatedAt.IsZero() {
		t.Error("UpdatedAt not set")
	}
	if !saved.UpdatedAt.Equal(saved.CreatedAt) {
		t.Errorf("UpdatedAt %v != CreatedAt %v for new card", saved.UpdatedAt, saved.CreatedAt)
	}
}

func TestStore_Save_UpdatesExistingCard(t *testing.T) {
	store := NewStore()

	// Create initial card
	original := AgentCard{
		Name:        "Original Name",
		Description: "Original description",
		URL:         "https://example.com/update-test",
		Version:     "1.0.0",
	}

	if err := store.Save(original); err != nil {
		t.Fatalf("Save() original error = %v", err)
	}

	saved, _ := store.Get("https://example.com/update-test")
	originalCreatedAt := saved.CreatedAt

	// Wait a bit to ensure different timestamps
	time.Sleep(10 * time.Millisecond)

	// Update the card
	updated := AgentCard{
		Name:        "Updated Name",
		Description: "Updated description",
		URL:         "https://example.com/update-test",
		Version:     "2.0.0",
	}

	if err := store.Save(updated); err != nil {
		t.Fatalf("Save() updated error = %v", err)
	}

	saved, err := store.Get("https://example.com/update-test")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Verify fields were updated
	if saved.Name != "Updated Name" {
		t.Errorf("Name not updated, got %s, want %s", saved.Name, "Updated Name")
	}
	if saved.Description != "Updated description" {
		t.Errorf("Description not updated, got %s, want %s", saved.Description, "Updated description")
	}
	if saved.Version != "2.0.0" {
		t.Errorf("Version not updated, got %s, want %s", saved.Version, "2.0.0")
	}

	// Verify CreatedAt is preserved
	if !saved.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("CreatedAt changed from %v to %v", originalCreatedAt, saved.CreatedAt)
	}

	// Verify UpdatedAt is newer
	if !saved.UpdatedAt.After(saved.CreatedAt) {
		t.Errorf("UpdatedAt %v should be after CreatedAt %v", saved.UpdatedAt, saved.CreatedAt)
	}
}

func TestStore_Save_SetsDefaultVersion(t *testing.T) {
	store := NewStore()

	card := AgentCard{
		Name:    "No Version",
		URL:     "https://example.com/no-version",
		Version: "", // Empty version
	}

	if err := store.Save(card); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	saved, err := store.Get("https://example.com/no-version")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if saved.Version != "1.0.0" {
		t.Errorf("Version = %s, want %s", saved.Version, "1.0.0")
	}
}

func TestStore_Get(t *testing.T) {
	store := NewStore()

	// Add a test card
	card := AgentCard{
		Name:        "Test Agent",
		Description: "Test description",
		URL:         "https://example.com/test",
		Version:     "1.0.0",
	}

	if err := store.Save(card); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "existing card",
			url:     "https://example.com/test",
			wantErr: false,
		},
		{
			name:    "non-existent card",
			url:     "https://example.com/nonexistent",
			wantErr: true,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.Get(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Error("Get() expected error but got none")
					return
				}
				if !errors.Is(err, ErrNotFound) {
					t.Errorf("Get() error = %v, want ErrNotFound", err)
				}
				return
			}

			if err != nil {
				t.Errorf("Get() unexpected error = %v", err)
				return
			}

			if got.URL != tt.url {
				t.Errorf("Get() URL = %s, want %s", got.URL, tt.url)
			}
		})
	}
}

func TestStore_GetByName(t *testing.T) {
	store := NewStore()

	// Add test cards
	cards := []AgentCard{
		{
			Name:    "Alpha Agent",
			URL:     "https://example.com/alpha",
			Version: "1.0.0",
		},
		{
			Name:    "Beta Agent",
			URL:     "https://example.com/beta",
			Version: "1.0.0",
		},
		{
			Name:    "alpha agent", // lowercase duplicate for case-insensitive test
			URL:     "https://example.com/alpha-lower",
			Version: "1.0.0",
		},
	}

	for _, card := range cards {
		if err := store.Save(card); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	tests := []struct {
		name        string
		searchName  string
		wantURL     string
		wantErr     bool
		expectErrIs error
	}{
		{
			name:       "find alpha agent",
			searchName: "Alpha Agent",
			wantURL:    "https://example.com/alpha",
			wantErr:    false,
		},
		{
			name:       "find beta agent",
			searchName: "Beta Agent",
			wantURL:    "https://example.com/beta",
			wantErr:    false,
		},
		{
			name:       "case insensitive search",
			searchName: "ALPHA AGENT",
			wantURL:    "https://example.com/alpha",
			wantErr:    false,
		},
		{
			name:        "non-existent agent",
			searchName:  "Gamma Agent",
			wantErr:     true,
			expectErrIs: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.GetByName(tt.searchName)
			if tt.wantErr {
				if err == nil {
					t.Error("GetByName() expected error but got none")
					return
				}
				if tt.expectErrIs != nil && !errors.Is(err, tt.expectErrIs) {
					t.Errorf("GetByName() error = %v, want %v", err, tt.expectErrIs)
				}
				return
			}

			if err != nil {
				t.Errorf("GetByName() unexpected error = %v", err)
				return
			}

			if got.URL != tt.wantURL {
				t.Errorf("GetByName() URL = %s, want %s", got.URL, tt.wantURL)
			}
		})
	}
}

func TestStore_List(t *testing.T) {
	store := NewStore()

	// Initially empty
	cards := store.List()
	if len(cards) != 0 {
		t.Errorf("List() on empty store = %d cards, want 0", len(cards))
	}

	// Add some cards
	testCards := []AgentCard{
		{Name: "Agent 1", URL: "https://example.com/1", Version: "1.0.0"},
		{Name: "Agent 2", URL: "https://example.com/2", Version: "1.0.0"},
		{Name: "Agent 3", URL: "https://example.com/3", Version: "1.0.0"},
	}

	for _, card := range testCards {
		if err := store.Save(card); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// List should return all cards
	cards = store.List()
	if len(cards) != 3 {
		t.Errorf("List() = %d cards, want 3", len(cards))
	}

	// Verify all cards are present
	urlSet := make(map[string]bool)
	for _, card := range cards {
		urlSet[card.URL] = true
	}

	for _, card := range testCards {
		if !urlSet[card.URL] {
			t.Errorf("List() missing card with URL %s", card.URL)
		}
	}
}

func TestStore_Delete(t *testing.T) {
	store := NewStore()

	// Add a test card
	card := AgentCard{
		Name:    "To Delete",
		URL:     "https://example.com/delete",
		Version: "1.0.0",
	}

	if err := store.Save(card); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify it exists
	if _, err := store.Get("https://example.com/delete"); err != nil {
		t.Error("Card should exist before deletion")
	}

	// Delete it
	if err := store.Delete("https://example.com/delete"); err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify it's gone
	if _, err := store.Get("https://example.com/delete"); !errors.Is(err, ErrNotFound) {
		t.Error("Card should not exist after deletion")
	}

	// Verify count is 0
	if store.Count() != 0 {
		t.Errorf("Count() = %d, want 0 after deletion", store.Count())
	}
}

func TestStore_Delete_NotFound(t *testing.T) {
	store := NewStore()

	err := store.Delete("https://example.com/nonexistent")
	if err == nil {
		t.Error("Delete() expected error for non-existent card")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
}

func TestStore_Count(t *testing.T) {
	store := NewStore()

	if store.Count() != 0 {
		t.Errorf("Count() on empty store = %d, want 0", store.Count())
	}

	// Add cards
	for i := 1; i <= 5; i++ {
		card := AgentCard{
			Name:    "Agent",
			URL:     "https://example.com/" + string(rune('0'+i)),
			Version: "1.0.0",
		}
		if err := store.Save(card); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	if store.Count() != 5 {
		t.Errorf("Count() = %d, want 5", store.Count())
	}
}

func TestStore_Clear(t *testing.T) {
	store := NewStore()

	// Add some cards
	for i := 1; i <= 3; i++ {
		card := AgentCard{
			Name:    "Agent",
			URL:     "https://example.com/" + string(rune('0'+i)),
			Version: "1.0.0",
		}
		if err := store.Save(card); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	if store.Count() != 3 {
		t.Errorf("Count() before clear = %d, want 3", store.Count())
	}

	store.Clear()

	if store.Count() != 0 {
		t.Errorf("Count() after clear = %d, want 0", store.Count())
	}

	if len(store.List()) != 0 {
		t.Error("List() should return empty slice after Clear()")
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	store := NewStore()
	const numGoroutines = 100

	// Concurrent writes
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			card := AgentCard{
				Name:    "Concurrent Agent",
				URL:     "https://example.com/concurrent/" + string(rune('0'+n%10)),
				Version: "1.0.0",
			}
			store.Save(card)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			url := "https://example.com/concurrent/" + string(rune('0'+n%10))
			store.Get(url)
			store.List()
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify store is still consistent
	if store.Count() != 10 {
		t.Errorf("Count() after concurrent access = %d, want 10", store.Count())
	}
}

func TestStore_SaveGetRoundtrip(t *testing.T) {
	store := NewStore()

	original := AgentCard{
		Name:        "Full Agent",
		Description: "Complete agent card",
		URL:         "https://example.com/full",
		Version:     "2.1.0",
		Capabilities: Capabilities{
			Streaming:              true,
			PushNotifications:      true,
			StateTransitionHistory: false,
		},
		Skills: []Skill{
			{
				ID:          "skill-1",
				Name:        "Test Skill",
				Description: "A test skill",
				Tags:        []string{"test", "demo"},
			},
		},
		Authentication: AuthInfo{
			Schemes: []string{"Bearer", "ApiKey"},
		},
	}

	if err := store.Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	retrieved, err := store.Get("https://example.com/full")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Name != original.Name {
		t.Errorf("Name mismatch: got %s, want %s", retrieved.Name, original.Name)
	}
	if retrieved.Description != original.Description {
		t.Errorf("Description mismatch: got %s, want %s", retrieved.Description, original.Description)
	}
	if retrieved.URL != original.URL {
		t.Errorf("URL mismatch: got %s, want %s", retrieved.URL, original.URL)
	}
	if retrieved.Version != original.Version {
		t.Errorf("Version mismatch: got %s, want %s", retrieved.Version, original.Version)
	}
	if retrieved.Capabilities.Streaming != original.Capabilities.Streaming {
		t.Errorf("Streaming capability mismatch")
	}
	if len(retrieved.Skills) != len(original.Skills) {
		t.Errorf("Skills length mismatch: got %d, want %d", len(retrieved.Skills), len(original.Skills))
	}
	if len(retrieved.Authentication.Schemes) != len(original.Authentication.Schemes) {
		t.Errorf("Auth schemes length mismatch")
	}
}
