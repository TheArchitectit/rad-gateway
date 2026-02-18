// Package a2a provides A2A (Agent-to-Agent) protocol support for RAD Gateway.
package a2a

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"radgateway/internal/db"
)

// generateID generates a random UUID-like identifier.
func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// Cache defines the cache operations needed by the repository.
// This interface is typically implemented by cache.TypedModelCardCache.
type Cache interface {
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

	// InvalidateCard removes a card and its associated cache entries.
	InvalidateCard(ctx context.Context, id string, projectID string) error
}

// Repository defines the interface for A2A Model Card operations.
type Repository interface {
	// GetByID retrieves a model card by ID.
	GetByID(ctx context.Context, id string) (*ModelCard, error)
	// GetByProject retrieves all model cards for a project/workspace.
	GetByProject(ctx context.Context, projectID string) ([]ModelCard, error)
	// Create creates a new model card.
	Create(ctx context.Context, card *ModelCard) error
	// Update updates an existing model card.
	Update(ctx context.Context, card *ModelCard) error
	// Delete deletes a model card by ID.
	Delete(ctx context.Context, id string) error
	// GetBySlug retrieves a model card by workspace and slug.
	GetBySlug(ctx context.Context, workspaceID, slug string) (*ModelCard, error)
}

// HybridModelCardRepository implements Repository with PostgreSQL + Redis cache-aside pattern.
type HybridModelCardRepository struct {
	db    *sql.DB
	cache Cache
	log   *slog.Logger
}

// NewHybridRepository creates a new hybrid repository.
func NewHybridRepository(database *sql.DB, cache Cache, log *slog.Logger) *HybridModelCardRepository {
	if log == nil {
		log = slog.Default()
	}
	return &HybridModelCardRepository{
		db:    database,
		cache: cache,
		log:   log.With("component", "hybrid_repository"),
	}
}

// NewHybridRepositoryFromDB creates a repository from the db.Database interface.
func NewHybridRepositoryFromDB(database db.Database, cache Cache, log *slog.Logger) (*HybridModelCardRepository, error) {
	// Try to get underlying SQL DB
	type sqlDBer interface {
		DB() *sql.DB
	}

	if sqlDB, ok := database.(sqlDBer); ok {
		return NewHybridRepository(sqlDB.DB(), cache, log), nil
	}

	return nil, errors.New("database does not provide access to underlying *sql.DB")
}

// GetByID retrieves a model card by ID using cache-aside pattern.
func (r *HybridModelCardRepository) GetByID(ctx context.Context, id string) (*ModelCard, error) {
	// 1. Check cache first
	cached, err := r.cache.Get(ctx, id)
	if err == nil && cached != nil {
		r.log.Debug("cache hit for model card", "id", id)
		return cached, nil
	}
	if err != nil {
		// Cache error - log but continue to DB
		r.log.Warn("cache error, falling back to DB", "id", id, "error", err)
	}

	// 2. Cache miss or error - query PostgreSQL
	r.log.Debug("cache miss for model card", "id", id)

	query := `
		SELECT id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at
		FROM a2a_model_cards
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, id)
	card, err := r.scanModelCard(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("model card not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get model card: %w", err)
	}

	// 3. Cache the result (best effort - don't fail if cache fails)
	r.cacheCardAsync(ctx, card)

	return card, nil
}

// GetByProject retrieves all model cards for a project/workspace.
func (r *HybridModelCardRepository) GetByProject(ctx context.Context, projectID string) ([]ModelCard, error) {
	// 1. Check cache first
	cached, err := r.cache.GetProjectCards(ctx, projectID)
	if err == nil && cached != nil {
		r.log.Debug("cache hit for project cards", "project_id", projectID, "count", len(cached))
		return cached, nil
	}
	if err != nil {
		r.log.Warn("cache error for project, falling back to DB", "project_id", projectID, "error", err)
	}

	// 2. Cache miss - query PostgreSQL
	r.log.Debug("cache miss for project cards", "project_id", projectID)

	query := `
		SELECT id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at
		FROM a2a_model_cards
		WHERE workspace_id = $1
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query project cards: %w", err)
	}
	defer rows.Close()

	var cards []ModelCard
	for rows.Next() {
		card, err := r.scanModelCardFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model card: %w", err)
		}
		cards = append(cards, *card)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cards: %w", err)
	}

	// 3. Cache the result (best effort)
	r.cacheProjectCardsAsync(ctx, projectID, cards)

	return cards, nil
}

// Create creates a new model card.
func (r *HybridModelCardRepository) Create(ctx context.Context, card *ModelCard) error {
	// Use transaction for database write
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate ID if not provided (for databases without UUID generation)
	if card.ID == "" {
		card.ID = generateID()
	}

	query := `
		INSERT INTO a2a_model_cards (id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	now := time.Now().UTC()
	card.CreatedAt = now
	card.UpdatedAt = now
	if card.Version == 0 {
		card.Version = 1
	}
	if card.Status == "" {
		card.Status = ModelCardStatusActive
	}

	_, err = tx.ExecContext(ctx, query,
		card.ID,
		card.WorkspaceID,
		card.UserID,
		card.Name,
		card.Slug,
		card.Description,
		card.Card,
		card.Version,
		card.Status,
		card.CreatedAt,
		card.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create model card: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Cache the new card (best effort)
	r.cacheCardAsync(ctx, card)

	// Invalidate project cache since list changed
	if err := r.cache.DeleteProjectCards(ctx, card.WorkspaceID); err != nil {
		r.log.Warn("failed to invalidate project cache on create", "project_id", card.WorkspaceID, "error", err)
	}

	r.log.Debug("created model card", "id", card.ID, "workspace_id", card.WorkspaceID)
	return nil
}

// Update updates an existing model card.
func (r *HybridModelCardRepository) Update(ctx context.Context, card *ModelCard) error {
	// First, get the existing card to know the workspace for cache invalidation
	existing, err := r.GetByID(ctx, card.ID)
	if err != nil {
		return err
	}

	// Use transaction for database write
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE a2a_model_cards
		SET name = $1, description = $2, card = $3, version = $4, status = $5, updated_at = $6
		WHERE id = $7
	`

	card.UpdatedAt = time.Now().UTC()
	card.Version++ // Increment version on update

	result, err := tx.ExecContext(ctx, query,
		card.Name,
		card.Description,
		card.Card,
		card.Version,
		card.Status,
		card.UpdatedAt,
		card.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update model card: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("model card not found: %s", card.ID)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate cache - both the card and project list
	if err := r.cache.InvalidateCard(ctx, card.ID, existing.WorkspaceID); err != nil {
		r.log.Warn("failed to invalidate cache on update", "id", card.ID, "error", err)
	}

	// If workspace changed, invalidate both old and new project caches
	if existing.WorkspaceID != card.WorkspaceID {
		if err := r.cache.DeleteProjectCards(ctx, card.WorkspaceID); err != nil {
			r.log.Warn("failed to invalidate new project cache on update", "project_id", card.WorkspaceID, "error", err)
		}
	}

	r.log.Debug("updated model card", "id", card.ID, "workspace_id", card.WorkspaceID)
	return nil
}

// Delete deletes a model card by ID.
func (r *HybridModelCardRepository) Delete(ctx context.Context, id string) error {
	// First, get the existing card to know the workspace for cache invalidation
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Use transaction for database write
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `DELETE FROM a2a_model_cards WHERE id = $1`

	result, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete model card: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("model card not found: %s", id)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate cache
	if err := r.cache.InvalidateCard(ctx, id, existing.WorkspaceID); err != nil {
		r.log.Warn("failed to invalidate cache on delete", "id", id, "error", err)
	}

	r.log.Debug("deleted model card", "id", id, "workspace_id", existing.WorkspaceID)
	return nil
}

// GetBySlug retrieves a model card by workspace and slug.
func (r *HybridModelCardRepository) GetBySlug(ctx context.Context, workspaceID, slug string) (*ModelCard, error) {
	// Note: We don't cache by slug to avoid cache inconsistency issues
	// Slug lookups go directly to database

	query := `
		SELECT id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at
		FROM a2a_model_cards
		WHERE workspace_id = $1 AND slug = $2
	`

	row := r.db.QueryRowContext(ctx, query, workspaceID, slug)
	card, err := r.scanModelCard(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("model card not found: %s/%s", workspaceID, slug)
		}
		return nil, fmt.Errorf("failed to get model card by slug: %w", err)
	}

	return card, nil
}

// scanModelCard scans a single model card from a Row.
func (r *HybridModelCardRepository) scanModelCard(row *sql.Row) (*ModelCard, error) {
	var card ModelCard
	var userID sql.NullString
	var description sql.NullString
	var cardData []byte

	err := row.Scan(
		&card.ID,
		&card.WorkspaceID,
		&userID,
		&card.Name,
		&card.Slug,
		&description,
		&cardData,
		&card.Version,
		&card.Status,
		&card.CreatedAt,
		&card.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if userID.Valid {
		card.UserID = &userID.String
	}
	if description.Valid {
		card.Description = &description.String
	}
	if cardData != nil {
		card.Card = cardData
	}

	return &card, nil
}

// scanModelCardFromRows scans a model card from Rows (for iterating).
func (r *HybridModelCardRepository) scanModelCardFromRows(rows *sql.Rows) (*ModelCard, error) {
	var card ModelCard
	var userID sql.NullString
	var description sql.NullString
	var cardData []byte

	err := rows.Scan(
		&card.ID,
		&card.WorkspaceID,
		&userID,
		&card.Name,
		&card.Slug,
		&description,
		&cardData,
		&card.Version,
		&card.Status,
		&card.CreatedAt,
		&card.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if userID.Valid {
		card.UserID = &userID.String
	}
	if description.Valid {
		card.Description = &description.String
	}
	if cardData != nil {
		card.Card = cardData
	}

	return &card, nil
}

// cacheCardAsync caches a model card asynchronously (fire and forget).
func (r *HybridModelCardRepository) cacheCardAsync(ctx context.Context, card *ModelCard) {
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := r.cache.Set(cacheCtx, card.ID, card, 5*time.Minute); err != nil {
			r.log.Warn("failed to cache card", "id", card.ID, "error", err)
		}
	}()
}

// cacheProjectCardsAsync caches project cards asynchronously.
func (r *HybridModelCardRepository) cacheProjectCardsAsync(ctx context.Context, projectID string, cards []ModelCard) {
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := r.cache.SetProjectCards(cacheCtx, projectID, cards, 1*time.Minute); err != nil {
			r.log.Warn("failed to cache project cards", "project_id", projectID, "error", err)
		}
	}()
}

// Ensure HybridModelCardRepository implements Repository interface.
var _ Repository = (*HybridModelCardRepository)(nil)
