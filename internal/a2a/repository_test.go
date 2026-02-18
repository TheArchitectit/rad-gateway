// Package a2a provides A2A (Agent-to-Agent) protocol support for RAD Gateway.
package a2a

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"
)

// mockCache implements Cache interface for testing.
type mockCache struct {
	cards         map[string]*ModelCard
	projectCards  map[string][]ModelCard
	getErr        error
	setErr        error
	deleteErr     error
}

func newMockCache() *mockCache {
	return &mockCache{
		cards:        make(map[string]*ModelCard),
		projectCards: make(map[string][]ModelCard),
	}
}

func (m *mockCache) Get(ctx context.Context, id string) (*ModelCard, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	card, ok := m.cards[id]
	if !ok {
		return nil, nil
	}
	return card, nil
}

func (m *mockCache) Set(ctx context.Context, id string, card *ModelCard, ttl time.Duration) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.cards[id] = card
	return nil
}

func (m *mockCache) Delete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.cards, id)
	return nil
}

func (m *mockCache) GetProjectCards(ctx context.Context, projectID string) ([]ModelCard, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	cards, ok := m.projectCards[projectID]
	if !ok {
		return nil, nil
	}
	return cards, nil
}

func (m *mockCache) SetProjectCards(ctx context.Context, projectID string, cards []ModelCard, ttl time.Duration) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.projectCards[projectID] = cards
	return nil
}

func (m *mockCache) DeleteProjectCards(ctx context.Context, projectID string) error {
	delete(m.projectCards, projectID)
	return nil
}

func (m *mockCache) InvalidateCard(ctx context.Context, id string, projectID string) error {
	m.Delete(ctx, id)
	if projectID != "" {
		m.DeleteProjectCards(ctx, projectID)
	}
	return nil
}

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Create the a2a_model_cards table
	schema := `
		CREATE TABLE a2a_model_cards (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			user_id TEXT,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			description TEXT,
			card BLOB NOT NULL DEFAULT '{}',
			version INTEGER NOT NULL DEFAULT 1,
			status TEXT NOT NULL DEFAULT 'active',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE UNIQUE INDEX idx_a2a_model_cards_workspace_slug ON a2a_model_cards(workspace_id, slug);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return db
}

func TestHybridRepository_GetByID_CacheHit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mc := newMockCache()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := NewHybridRepository(db, mc, log)

	ctx := context.Background()

	// Create and cache a card
	card := &ModelCard{
		ID:          "test-id",
		WorkspaceID: "workspace-1",
		Name:        "Test Card",
		Slug:        "test-card",
		Status:      ModelCardStatusActive,
		Card:        []byte(`{"name":"test"}`),
	}
	mc.Set(ctx, "test-id", card, time.Minute)

	// Get from repository - should hit cache
	result, err := repo.GetByID(ctx, "test-id")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.ID != card.ID {
		t.Errorf("expected ID %s, got %s", card.ID, result.ID)
	}
	if result.Name != card.Name {
		t.Errorf("expected Name %s, got %s", card.Name, result.Name)
	}
}

func TestHybridRepository_GetByID_CacheMiss(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mc := newMockCache()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := NewHybridRepository(db, mc, log)

	ctx := context.Background()

	// Insert card directly into database
	_, err := db.Exec(`
		INSERT INTO a2a_model_cards (id, workspace_id, name, slug, status, card, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "db-id", "workspace-1", "DB Card", "db-card", "active", `{"name":"db"}`, 1, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// Get from repository - should miss cache and hit DB
	result, err := repo.GetByID(ctx, "db-id")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.ID != "db-id" {
		t.Errorf("expected ID db-id, got %s", result.ID)
	}
	if result.Name != "DB Card" {
		t.Errorf("expected Name 'DB Card', got %s", result.Name)
	}

	// Wait for async cache population
	time.Sleep(100 * time.Millisecond)

	// Verify it was cached
	cached, err := mc.Get(ctx, "db-id")
	if err != nil {
		t.Fatalf("expected no error getting from cache, got: %v", err)
	}
	if cached == nil {
		t.Error("expected card to be cached after DB read")
	}
}

func TestHybridRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mc := newMockCache()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := NewHybridRepository(db, mc, log)

	ctx := context.Background()

	// Try to get non-existent card
	_, err := repo.GetByID(ctx, "non-existent")
	if err == nil {
		t.Error("expected error for non-existent card, got nil")
	}

	if !errors.Is(err, sql.ErrNoRows) && err.Error() != "model card not found: non-existent" {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestHybridRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mc := newMockCache()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := NewHybridRepository(db, mc, log)

	ctx := context.Background()

	card := &ModelCard{
		WorkspaceID: "workspace-1",
		Name:        "New Card",
		Slug:        "new-card",
		Description: strPtr("A test card"),
		Card:        []byte(`{"name":"new"}`),
	}

	if err := repo.Create(ctx, card); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify ID was set
	if card.ID == "" {
		t.Error("expected ID to be set after create")
	}

	// Verify in database
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM a2a_model_cards WHERE id = ?", card.ID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 card in database, got %d", count)
	}

	// Wait for async cache
	time.Sleep(100 * time.Millisecond)

	// Verify in cache
	cached, err := mc.Get(ctx, card.ID)
	if err != nil {
		t.Fatalf("expected no error getting from cache, got: %v", err)
	}
	if cached == nil {
		t.Error("expected card to be cached after create")
	}
}

func TestHybridRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mc := newMockCache()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := NewHybridRepository(db, mc, log)

	ctx := context.Background()

	// Insert a card
	_, err := db.Exec(`
		INSERT INTO a2a_model_cards (id, workspace_id, name, slug, status, card, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "update-id", "workspace-1", "Original Name", "original-slug", "active", `{"name":"original"}`, 1, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// Pre-populate cache with the card
	original := &ModelCard{
		ID:          "update-id",
		WorkspaceID: "workspace-1",
		Name:        "Original Name",
		Slug:        "original-slug",
		Version:     1,
	}
	mc.Set(ctx, "update-id", original, time.Minute)

	// Update the card (get existing first to have all fields)
	existing, _ := repo.GetByID(ctx, "update-id")
	existing.Name = "Updated Name"
	existing.Card = []byte(`{"name":"updated"}`)

	if err := repo.Update(ctx, existing); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify in database directly (bypassing cache)
	var version int
	var name string
	err = db.QueryRow("SELECT version, name FROM a2a_model_cards WHERE id = ?", "update-id").Scan(&version, &name)
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	if version != 2 {
		t.Errorf("expected version 2 in DB, got %d", version)
	}
	if name != "Updated Name" {
		t.Errorf("expected name 'Updated Name' in DB, got %s", name)
	}

	// Verify cache was invalidated
	cached, _ := mc.Get(ctx, "update-id")
	if cached != nil {
		t.Error("expected cache to be invalidated after update")
	}
}

func TestHybridRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mc := newMockCache()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := NewHybridRepository(db, mc, log)

	ctx := context.Background()

	// Insert a card
	_, err := db.Exec(`
		INSERT INTO a2a_model_cards (id, workspace_id, name, slug, status, card, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "delete-id", "workspace-1", "To Delete", "to-delete", "active", `{}`, 1, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// Cache it
	card := &ModelCard{ID: "delete-id", WorkspaceID: "workspace-1"}
	mc.Set(ctx, "delete-id", card, time.Minute)
	mc.SetProjectCards(ctx, "workspace-1", []ModelCard{*card}, time.Minute)

	// Delete the card
	if err := repo.Delete(ctx, "delete-id"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify deleted from database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM a2a_model_cards WHERE id = ?", "delete-id").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 cards in database, got %d", count)
	}

	// Verify cache invalidated
	cached, _ := mc.Get(ctx, "delete-id")
	if cached != nil {
		t.Error("expected cache to be invalidated after delete")
	}
}

func TestHybridRepository_GetByProject(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mc := newMockCache()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := NewHybridRepository(db, mc, log)

	ctx := context.Background()

	// Insert multiple cards for the same workspace
	for i := 1; i <= 3; i++ {
		_, err := db.Exec(`
			INSERT INTO a2a_model_cards (id, workspace_id, name, slug, status, card, version, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			fmt.Sprintf("card-%d", i),
			"workspace-1",
			fmt.Sprintf("Card %d", i),
			fmt.Sprintf("card-%d", i),
			"active",
			fmt.Sprintf(`{"name":"card%d"}`, i),
			1,
			time.Now(),
			time.Now())
		if err != nil {
			t.Fatalf("failed to insert test data: %v", err)
		}
	}

	// Get all cards for workspace
	cards, err := repo.GetByProject(ctx, "workspace-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(cards) != 3 {
		t.Errorf("expected 3 cards, got %d", len(cards))
	}

	// Wait for async cache
	time.Sleep(100 * time.Millisecond)

	// Verify project cached
	cached, err := mc.GetProjectCards(ctx, "workspace-1")
	if err != nil {
		t.Fatalf("expected no error getting from cache, got: %v", err)
	}
	if cached == nil {
		t.Error("expected project cards to be cached")
	}
}

func TestHybridRepository_SplitBrainScenario(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mc := newMockCache()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := NewHybridRepository(db, mc, log)

	ctx := context.Background()

	// Insert a card into DB
	_, err := db.Exec(`
		INSERT INTO a2a_model_cards (id, workspace_id, name, slug, status, card, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "split-id", "workspace-1", "DB Version", "split", "active", `{"version":"db"}`, 1, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// Cache a stale version (simulating cache inconsistency)
	staleCard := &ModelCard{
		ID:          "split-id",
		WorkspaceID: "workspace-1",
		Name:        "Cache Version",
		Card:        []byte(`{"version":"cache"}`),
		Version:     1,
	}
	mc.Set(ctx, "split-id", staleCard, time.Minute)

	// Read should return cached version (first)
	result, err := repo.GetByID(ctx, "split-id")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// With cache hit, we get cached data
	if result.Name != "Cache Version" {
		t.Errorf("expected cached version 'Cache Version', got '%s'", result.Name)
	}

	// Now simulate cache corruption - delete and read again
	mc.Delete(ctx, "split-id")

	// Now should get from DB
	result, err = repo.GetByID(ctx, "split-id")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Name != "DB Version" {
		t.Errorf("expected DB version 'DB Version', got '%s'", result.Name)
	}
}

func TestHybridRepository_CacheFailureGraceful(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mc := newMockCache()
	mc.getErr = errors.New("cache unavailable")
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := NewHybridRepository(db, mc, log)

	ctx := context.Background()

	// Insert a card
	_, err := db.Exec(`
		INSERT INTO a2a_model_cards (id, workspace_id, name, slug, status, card, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "graceful-id", "workspace-1", "Graceful Card", "graceful", "active", `{}`, 1, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// Should still work even with cache failure
	result, err := repo.GetByID(ctx, "graceful-id")
	if err != nil {
		t.Fatalf("expected no error despite cache failure, got: %v", err)
	}

	if result.Name != "Graceful Card" {
		t.Errorf("expected 'Graceful Card', got '%s'", result.Name)
	}
}

func TestHybridRepository_GetBySlug(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mc := newMockCache()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := NewHybridRepository(db, mc, log)

	ctx := context.Background()

	// Insert a card
	_, err := db.Exec(`
		INSERT INTO a2a_model_cards (id, workspace_id, name, slug, status, card, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "slug-id", "workspace-1", "Slug Card", "my-slug", "active", `{}`, 1, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// Get by slug
	result, err := repo.GetBySlug(ctx, "workspace-1", "my-slug")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.ID != "slug-id" {
		t.Errorf("expected ID 'slug-id', got '%s'", result.ID)
	}
	if result.Slug != "my-slug" {
		t.Errorf("expected slug 'my-slug', got '%s'", result.Slug)
	}
}

// strPtr returns a pointer to a string.
func strPtr(s string) *string {
	return &s
}
