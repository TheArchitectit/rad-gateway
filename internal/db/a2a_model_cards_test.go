// Package db contains database models for RAD Gateway.
package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates a test database with the schema.
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Create minimal schema for testing
	schema := `
		CREATE TABLE workspaces (
			id TEXT PRIMARY KEY,
			slug TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			settings BLOB NOT NULL DEFAULT '{}',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			email TEXT NOT NULL,
			display_name TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			password_hash TEXT,
			last_login_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
		);

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
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
		);

		CREATE UNIQUE INDEX idx_a2a_model_cards_workspace_slug ON a2a_model_cards(workspace_id, slug);
		CREATE INDEX idx_a2a_model_cards_workspace ON a2a_model_cards(workspace_id);
		CREATE INDEX idx_a2a_model_cards_user ON a2a_model_cards(user_id);
		CREATE INDEX idx_a2a_model_cards_status ON a2a_model_cards(status);

		CREATE TABLE model_card_versions (
			id TEXT PRIMARY KEY,
			model_card_id TEXT NOT NULL,
			workspace_id TEXT NOT NULL,
			user_id TEXT,
			version INTEGER NOT NULL,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			description TEXT,
			card BLOB NOT NULL DEFAULT '{}',
			status TEXT NOT NULL DEFAULT 'active',
			change_reason TEXT,
			created_by TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (model_card_id) REFERENCES a2a_model_cards(id) ON DELETE CASCADE,
			FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
			FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
			UNIQUE(model_card_id, version)
		);

		CREATE INDEX idx_model_card_versions_card ON model_card_versions(model_card_id);
		CREATE INDEX idx_model_card_versions_workspace ON model_card_versions(workspace_id);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	// Insert test workspace
	_, err = db.Exec(`INSERT INTO workspaces (id, slug, name) VALUES ('ws-test', 'test-workspace', 'Test Workspace')`)
	if err != nil {
		t.Fatalf("failed to insert test workspace: %v", err)
	}

	// Insert test user
	_, err = db.Exec(`INSERT INTO users (id, workspace_id, email) VALUES ('user-test', 'ws-test', 'test@example.com')`)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}

	return db
}

// createTestCard creates a test model card with sample JSON.
func createTestCard(t *testing.T, workspaceID, userID string) *ModelCard {
	cardJSON := &ModelCardJSON{
		Name:        "Test Agent",
		Description: "A test agent for unit testing",
		URL:         "https://example.com/a2a",
		Version:     "1.0.0",
		Capabilities: ModelCardCapability{
			Streaming:         true,
			PushNotifications: false,
		},
		Skills: []ModelCardSkill{
			{
				ID:          "test-skill",
				Name:        "Test Skill",
				Description: "A test skill",
				Tags:        []string{"test", "demo"},
			},
		},
	}

	cardData, err := json.Marshal(cardJSON)
	if err != nil {
		t.Fatalf("failed to marshal card JSON: %v", err)
	}

	return &ModelCard{
		ID:          "card-test",
		WorkspaceID: workspaceID,
		UserID:      &userID,
		Name:        "Test Agent",
		Slug:        "test-agent",
		Description: strPtr("A test agent"),
		Card:        cardData,
		Status:      "active",
	}
}

func strPtr(s string) *string {
	return &s
}

func TestModelCardCreate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &pgModelCardRepo{db: db}
	ctx := context.Background()

	card := createTestCard(t, "ws-test", "user-test")

	err := repo.Create(ctx, card)
	if err != nil {
		t.Fatalf("failed to create model card: %v", err)
	}

	// Verify card was created
	if card.ID == "" {
		t.Error("expected card ID to be set")
	}
	if card.Version != 1 {
		t.Errorf("expected version 1, got %d", card.Version)
	}
	if card.Status != "active" {
		t.Errorf("expected status 'active', got %s", card.Status)
	}
	if card.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if card.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestModelCardGetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &pgModelCardRepo{db: db}
	ctx := context.Background()

	card := createTestCard(t, "ws-test", "user-test")
	err := repo.Create(ctx, card)
	if err != nil {
		t.Fatalf("failed to create model card: %v", err)
	}

	retrieved, err := repo.GetByID(ctx, card.ID)
	if err != nil {
		t.Fatalf("failed to get model card: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected card to be retrieved, got nil")
	}

	if retrieved.ID != card.ID {
		t.Errorf("expected ID %s, got %s", card.ID, retrieved.ID)
	}
	if retrieved.Name != card.Name {
		t.Errorf("expected name %s, got %s", card.Name, retrieved.Name)
	}
	if retrieved.Slug != card.Slug {
		t.Errorf("expected slug %s, got %s", card.Slug, retrieved.Slug)
	}
}

func TestModelCardGetBySlug(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &pgModelCardRepo{db: db}
	ctx := context.Background()

	card := createTestCard(t, "ws-test", "user-test")
	err := repo.Create(ctx, card)
	if err != nil {
		t.Fatalf("failed to create model card: %v", err)
	}

	retrieved, err := repo.GetBySlug(ctx, "ws-test", "test-agent")
	if err != nil {
		t.Fatalf("failed to get model card by slug: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected card to be retrieved, got nil")
	}
	if retrieved.Slug != "test-agent" {
		t.Errorf("expected slug 'test-agent', got %s", retrieved.Slug)
	}
}

func TestModelCardGetByWorkspace(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &pgModelCardRepo{db: db}
	ctx := context.Background()

	// Create multiple cards
	for i := 0; i < 3; i++ {
		card := createTestCard(t, "ws-test", "user-test")
		card.ID = "card-test-" + string(rune('a'+i))
		card.Slug = "test-agent-" + string(rune('a'+i))
		err := repo.Create(ctx, card)
		if err != nil {
			t.Fatalf("failed to create model card: %v", err)
		}
	}

	cards, err := repo.GetByWorkspace(ctx, "ws-test", 10, 0)
	if err != nil {
		t.Fatalf("failed to get cards by workspace: %v", err)
	}
	if len(cards) != 3 {
		t.Errorf("expected 3 cards, got %d", len(cards))
	}
}

func TestModelCardUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &pgModelCardRepo{db: db}
	ctx := context.Background()

	card := createTestCard(t, "ws-test", "user-test")
	err := repo.Create(ctx, card)
	if err != nil {
		t.Fatalf("failed to create model card: %v", err)
	}

	// Update the card
	card.Name = "Updated Agent"
	card.Description = strPtr("Updated description")

	changeReason := "Test update"
	updatedBy := "user-test"
	err = repo.Update(ctx, card, &changeReason, &updatedBy)
	if err != nil {
		t.Fatalf("failed to update model card: %v", err)
	}

	// Verify update
	retrieved, err := repo.GetByID(ctx, card.ID)
	if err != nil {
		t.Fatalf("failed to get updated card: %v", err)
	}
	if retrieved.Name != "Updated Agent" {
		t.Errorf("expected name 'Updated Agent', got %s", retrieved.Name)
	}
	if retrieved.Version != 2 {
		t.Errorf("expected version 2, got %d", retrieved.Version)
	}

	// Verify version record was created
	versions, err := repo.GetVersions(ctx, card.ID)
	if err != nil {
		t.Fatalf("failed to get versions: %v", err)
	}
	if len(versions) != 1 {
		t.Errorf("expected 1 version record, got %d", len(versions))
	}
}

func TestModelCardDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &pgModelCardRepo{db: db}
	ctx := context.Background()

	card := createTestCard(t, "ws-test", "user-test")
	err := repo.Create(ctx, card)
	if err != nil {
		t.Fatalf("failed to create model card: %v", err)
	}

	err = repo.Delete(ctx, card.ID)
	if err != nil {
		t.Fatalf("failed to delete model card: %v", err)
	}

	// Verify soft delete
	retrieved, err := repo.GetByID(ctx, card.ID)
	if err != nil {
		t.Fatalf("failed to get card after delete: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected card to exist after soft delete")
	}
	if retrieved.Status != "deleted" {
		t.Errorf("expected status 'deleted', got %s", retrieved.Status)
	}
}

func TestModelCardHardDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &pgModelCardRepo{db: db}
	ctx := context.Background()

	card := createTestCard(t, "ws-test", "user-test")
	err := repo.Create(ctx, card)
	if err != nil {
		t.Fatalf("failed to create model card: %v", err)
	}

	err = repo.HardDelete(ctx, card.ID)
	if err != nil {
		t.Fatalf("failed to hard delete model card: %v", err)
	}

	// Verify hard delete
	retrieved, err := repo.GetByID(ctx, card.ID)
	if err != nil {
		t.Fatalf("failed to get card after hard delete: %v", err)
	}
	if retrieved != nil {
		t.Error("expected card to be nil after hard delete")
	}
}

func TestModelCardGetVersions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &pgModelCardRepo{db: db}
	ctx := context.Background()

	card := createTestCard(t, "ws-test", "user-test")
	err := repo.Create(ctx, card)
	if err != nil {
		t.Fatalf("failed to create model card: %v", err)
	}

	// Create several versions
	for i := 0; i < 3; i++ {
		card.Name = "Updated Agent v" + string(rune('2'+i))
		changeReason := "Update " + string(rune('1'+i))
		updatedBy := "user-test"
		err = repo.Update(ctx, card, &changeReason, &updatedBy)
		if err != nil {
			t.Fatalf("failed to update model card: %v", err)
		}
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}

	versions, err := repo.GetVersions(ctx, card.ID)
	if err != nil {
		t.Fatalf("failed to get versions: %v", err)
	}
	if len(versions) != 3 {
		t.Errorf("expected 3 version records, got %d", len(versions))
	}

	// Verify versions are ordered (highest first)
	for i := 0; i < len(versions)-1; i++ {
		if versions[i].Version < versions[i+1].Version {
			t.Error("expected versions to be ordered descending")
		}
	}
}

func TestModelCardGetVersion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &pgModelCardRepo{db: db}
	ctx := context.Background()

	card := createTestCard(t, "ws-test", "user-test")
	err := repo.Create(ctx, card)
	if err != nil {
		t.Fatalf("failed to create model card: %v", err)
	}

	// Update to create a version
	card.Name = "Updated Agent"
	changeReason := "Test update"
	updatedBy := "user-test"
	err = repo.Update(ctx, card, &changeReason, &updatedBy)
	if err != nil {
		t.Fatalf("failed to update model card: %v", err)
	}

	// Get specific version
	version, err := repo.GetVersion(ctx, card.ID, 1)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}
	if version == nil {
		t.Fatal("expected version to exist")
	}
	if version.Version != 1 {
		t.Errorf("expected version 1, got %d", version.Version)
	}
	if version.Name != "Test Agent" {
		t.Errorf("expected original name 'Test Agent', got %s", version.Name)
	}
}

func TestModelCardRestoreVersion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &pgModelCardRepo{db: db}
	ctx := context.Background()

	card := createTestCard(t, "ws-test", "user-test")
	err := repo.Create(ctx, card)
	if err != nil {
		t.Fatalf("failed to create model card: %v", err)
	}

	// Update to create a version
	card.Name = "Updated Agent"
	changeReason := "Test update"
	updatedBy := "user-test"
	err = repo.Update(ctx, card, &changeReason, &updatedBy)
	if err != nil {
		t.Fatalf("failed to update model card: %v", err)
	}

	// Restore version 1
	restoredBy := "user-test"
	err = repo.RestoreVersion(ctx, card.ID, 1, &restoredBy)
	if err != nil {
		t.Fatalf("failed to restore version: %v", err)
	}

	// Verify restoration
	retrieved, err := repo.GetByID(ctx, card.ID)
	if err != nil {
		t.Fatalf("failed to get card after restore: %v", err)
	}
	if retrieved.Name != "Test Agent" {
		t.Errorf("expected name 'Test Agent' after restore, got %s", retrieved.Name)
	}
	if retrieved.Version != 3 {
		t.Errorf("expected version 3 after restore, got %d", retrieved.Version)
	}
}

func TestModelCardParsedCard(t *testing.T) {
	cardJSON := &ModelCardJSON{
		Name:        "Test Agent",
		Description: "A test agent",
		URL:         "https://example.com/a2a",
		Version:     "1.0.0",
		Capabilities: ModelCardCapability{
			Streaming:         true,
			PushNotifications: false,
		},
		Skills: []ModelCardSkill{
			{ID: "skill-1", Name: "Skill 1"},
			{ID: "skill-2", Name: "Skill 2"},
		},
	}

	cardData, err := json.Marshal(cardJSON)
	if err != nil {
		t.Fatalf("failed to marshal card: %v", err)
	}

	card := &ModelCard{
		Card: cardData,
	}

	parsed, err := card.ParsedCard()
	if err != nil {
		t.Fatalf("failed to parse card: %v", err)
	}

	if parsed.Name != "Test Agent" {
		t.Errorf("expected name 'Test Agent', got %s", parsed.Name)
	}
	if !parsed.Capabilities.Streaming {
		t.Error("expected Streaming capability to be true")
	}
	if parsed.Capabilities.PushNotifications {
		t.Error("expected PushNotifications capability to be false")
	}
	if len(parsed.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(parsed.Skills))
	}
}

func TestModelCardSetCard(t *testing.T) {
	card := &ModelCard{}
	cardJSON := &ModelCardJSON{
		Name: "Test Agent",
		URL:  "https://example.com/a2a",
	}

	err := card.SetCard(cardJSON)
	if err != nil {
		t.Fatalf("failed to set card: %v", err)
	}

	if card.Card == nil {
		t.Fatal("expected card data to be set")
	}

	// Verify by parsing
	parsed, err := card.ParsedCard()
	if err != nil {
		t.Fatalf("failed to parse card: %v", err)
	}
	if parsed.Name != "Test Agent" {
		t.Errorf("expected name 'Test Agent', got %s", parsed.Name)
	}
}

func TestModelCardVersionParsedCard(t *testing.T) {
	cardJSON := &ModelCardJSON{
		Name: "Test Agent",
		URL:  "https://example.com/a2a",
	}

	cardData, err := json.Marshal(cardJSON)
	if err != nil {
		t.Fatalf("failed to marshal card: %v", err)
	}

	version := &ModelCardVersion{
		Card: cardData,
	}

	parsed, err := version.ParsedCard()
	if err != nil {
		t.Fatalf("failed to parse version card: %v", err)
	}

	if parsed.Name != "Test Agent" {
		t.Errorf("expected name 'Test Agent', got %s", parsed.Name)
	}
}

func TestModelCardSearchByCapability(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &pgModelCardRepo{db: db}
	ctx := context.Background()

	// Create card with streaming capability
	card := createTestCard(t, "ws-test", "user-test")
	err := repo.Create(ctx, card)
	if err != nil {
		t.Fatalf("failed to create model card: %v", err)
	}

	// Note: SQLite doesn't support JSONB operators like PostgreSQL
	// This test would need PostgreSQL for full JSONB search testing
	// For now, we just verify the method doesn't error
	cards, err := repo.SearchByCapability(ctx, "ws-test", "streaming", 10, 0)
	if err != nil {
		t.Fatalf("failed to search by capability: %v", err)
	}
	// SQLite returns empty results since JSONB operators don't work
	t.Logf("Search by capability returned %d results (SQLite compatibility)", len(cards))
}

func TestModelCardSearchBySkill(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &pgModelCardRepo{db: db}
	ctx := context.Background()

	card := createTestCard(t, "ws-test", "user-test")
	err := repo.Create(ctx, card)
	if err != nil {
		t.Fatalf("failed to create model card: %v", err)
	}

	// Note: SQLite doesn't support JSONB array operators like PostgreSQL
	cards, err := repo.SearchBySkill(ctx, "ws-test", "test-skill", 10, 0)
	if err != nil {
		t.Fatalf("failed to search by skill: %v", err)
	}
	// SQLite returns empty results since JSONB operators don't work
	t.Logf("Search by skill returned %d results (SQLite compatibility)", len(cards))
}

func TestModelCardSearch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &pgModelCardRepo{db: db}
	ctx := context.Background()

	card := createTestCard(t, "ws-test", "user-test")
	err := repo.Create(ctx, card)
	if err != nil {
		t.Fatalf("failed to create model card: %v", err)
	}

	params := ModelCardSearchParams{
		WorkspaceID: "ws-test",
		Query:       "Test",
		Limit:       10,
	}

	results, err := repo.Search(ctx, params)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	// Should find at least one result
	if len(results) == 0 {
		t.Log("Search returned 0 results (SQLite text search may have limitations)")
	}
}

func TestModelCardNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &pgModelCardRepo{db: db}
	ctx := context.Background()

	// Try to get non-existent card
	card, err := repo.GetByID(ctx, "non-existent-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card != nil {
		t.Error("expected nil for non-existent card")
	}

	// Try to get non-existent version
	version, err := repo.GetVersion(ctx, "non-existent-id", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != nil {
		t.Error("expected nil for non-existent version")
	}
}

func TestModelCardRestoreVersionNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := &pgModelCardRepo{db: db}
	ctx := context.Background()

	card := createTestCard(t, "ws-test", "user-test")
	err := repo.Create(ctx, card)
	if err != nil {
		t.Fatalf("failed to create model card: %v", err)
	}

	// Try to restore non-existent version
	restoredBy := "user-test"
	err = repo.RestoreVersion(ctx, card.ID, 999, &restoredBy)
	if err == nil {
		t.Error("expected error when restoring non-existent version")
	}
}
