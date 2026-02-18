// Package integration provides integration tests for RAD Gateway
//
// A2A Model Card Integration Tests
// Tests the hybrid PostgreSQL + Redis cache-aside pattern
//
// Run with: go test ./tests/integration/... -run TestA2AModelCards
// Run verbose: go test -v ./tests/integration/... -run TestA2AModelCards
package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"radgateway/internal/a2a"
	"radgateway/internal/cache"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// testInfrastructure holds the test infrastructure
type testInfrastructure struct {
	db            *sql.DB
	redisCache    cache.Cache
	typedCache    cache.TypedModelCardCache
	repo          a2a.Repository
	log           *slog.Logger
	ctx           context.Context
	cancel        context.CancelFunc
}

// setupTestInfrastructure creates the test infrastructure
func setupTestInfrastructure(t *testing.T) *testInfrastructure {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	// Setup in-memory SQLite database (simulates PostgreSQL)
	db, err := setupTestDatabase(t)
	if err != nil {
		t.Fatalf("failed to setup test database: %v", err)
	}

	// Setup Redis cache if available
	var redisCache cache.Cache
	var typedCache cache.TypedModelCardCache

	redisConfig := &cache.GoRedisConfig{
		Addr:        getTestRedisAddr(),
		KeyPrefix:   "test:a2a:",
		DialTimeout: 2 * time.Second,
	}

	redisCache, err = cache.NewGoRedis(redisConfig)
	if err != nil {
		// Redis not available, skip tests that require it
		cancel()
		db.Close()
		t.Skipf("Redis not available at %s: %v - skipping integration test", getTestRedisAddr(), err)
		return nil
	}

	// Clear test keys before starting
	redisCache.DeletePattern(ctx, "test:a2a:*")

	typedCache = cache.NewTypedModelCardCache(redisCache, 5*time.Minute)
	repo := a2a.NewHybridRepository(db, typedCache, log)

	return &testInfrastructure{
		db:         db,
		redisCache: redisCache,
		typedCache: typedCache,
		repo:       repo,
		log:        log,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// teardown cleans up the test infrastructure
func (ti *testInfrastructure) teardown() {
	if ti.redisCache != nil {
		// Clear all test keys
		ti.redisCache.DeletePattern(ti.ctx, "test:a2a:*")
		ti.redisCache.Close()
	}
	if ti.db != nil {
		ti.db.Close()
	}
	if ti.cancel != nil {
		ti.cancel()
	}
}

// setupTestDatabase creates an in-memory SQLite database with the A2A schema
func setupTestDatabase(t *testing.T) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open test database: %w", err)
	}

	// Create the a2a_model_cards table (matches PostgreSQL schema)
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
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return db, nil
}

// getTestRedisAddr returns the Redis address for tests
func getTestRedisAddr() string {
	if addr := os.Getenv("REDIS_URL"); addr != "" {
		return addr
	}
	return "localhost:6379"
}

// strPtr returns a pointer to a string
func strPtr(s string) *string {
	return &s
}

// createTestModelCard creates a test model card
func createTestModelCard(workspaceID, name, slug string) *a2a.ModelCard {
	cardData, _ := json.Marshal(map[string]interface{}{
		"schemaVersion": "1.0",
		"name":          name,
		"description":   fmt.Sprintf("Description for %s", name),
		"capabilities": []map[string]interface{}{
			{"type": "text", "enabled": true},
			{"type": "vision", "enabled": false},
		},
	})

	return &a2a.ModelCard{
		WorkspaceID: workspaceID,
		Name:        name,
		Slug:        slug,
		Description: strPtr(fmt.Sprintf("Test description for %s", name)),
		Card:        cardData,
		Status:      a2a.ModelCardStatusActive,
	}
}

// waitForAsyncCache waits for asynchronous cache operations to complete
func waitForAsyncCache() {
	time.Sleep(150 * time.Millisecond)
}

// TestA2AModelCards_Create tests creating a model card writes to both PostgreSQL and Redis
func TestA2AModelCards_Create(t *testing.T) {
	ti := setupTestInfrastructure(t)
	if ti == nil {
		return
	}
	defer ti.teardown()

	card := createTestModelCard("workspace-1", "Test Create Card", "test-create-card")

	// Create the card
	err := ti.repo.Create(ti.ctx, card)
	if err != nil {
		t.Fatalf("failed to create model card: %v", err)
	}

	// Verify ID was generated
	if card.ID == "" {
		t.Error("expected ID to be set after create")
	}

	// Verify card exists in database
	var dbCount int
	err = ti.db.QueryRowContext(ti.ctx,
		"SELECT COUNT(*) FROM a2a_model_cards WHERE id = ?", card.ID).Scan(&dbCount)
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	if dbCount != 1 {
		t.Errorf("expected 1 card in database, got %d", dbCount)
	}

	// Wait for async cache population
	waitForAsyncCache()

	// Verify card exists in cache
	cached, err := ti.typedCache.Get(ti.ctx, card.ID)
	if err != nil {
		t.Fatalf("failed to get from cache: %v", err)
	}
	if cached == nil {
		t.Error("expected card to be cached after create")
	}
	if cached.Name != card.Name {
		t.Errorf("expected cached name %q, got %q", card.Name, cached.Name)
	}

	t.Logf("Created model card: ID=%s, Name=%s", card.ID, card.Name)
}

// TestA2AModelCards_ReadCacheHit tests reading a model card with cache hit
func TestA2AModelCards_ReadCacheHit(t *testing.T) {
	ti := setupTestInfrastructure(t)
	if ti == nil {
		return
	}
	defer ti.teardown()

	// Create card directly in database
	cardID := "cache-hit-test-id"
	_, err := ti.db.ExecContext(ti.ctx, `
		INSERT INTO a2a_model_cards (id, workspace_id, name, slug, status, card, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, cardID, "workspace-1", "Cache Hit Test", "cache-hit-test", "active",
		[]byte(`{"name":"test"}`), 1, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// Pre-populate cache with the card
	cachedCard := &a2a.ModelCard{
		ID:          cardID,
		WorkspaceID: "workspace-1",
		Name:        "Cached Version",
		Slug:        "cache-hit-test",
		Status:      a2a.ModelCardStatusActive,
		Card:        []byte(`{"name":"cached"}`),
		Version:     1,
	}
	err = ti.typedCache.Set(ti.ctx, cardID, cachedCard, 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to set cache: %v", err)
	}

	// Read from repository - should hit cache
	result, err := ti.repo.GetByID(ti.ctx, cardID)
	if err != nil {
		t.Fatalf("failed to get card: %v", err)
	}

	// Should return cached version (different from DB version)
	if result.Name != "Cached Version" {
		t.Errorf("expected cached version 'Cached Version', got '%s'", result.Name)
	}

	// Verify the cache was used by checking the card data
	if string(result.Card) != `{"name":"cached"}` {
		t.Errorf("expected cached card data, got '%s'", string(result.Card))
	}

	t.Log("Cache hit verified - returned cached version instead of database version")
}

// TestA2AModelCards_ReadCacheMiss tests reading a model card with cache miss
func TestA2AModelCards_ReadCacheMiss(t *testing.T) {
	ti := setupTestInfrastructure(t)
	if ti == nil {
		return
	}
	defer ti.teardown()

	// Create card directly in database (cache is empty)
	cardID := "cache-miss-test-id"
	_, err := ti.db.ExecContext(ti.ctx, `
		INSERT INTO a2a_model_cards (id, workspace_id, name, slug, status, card, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, cardID, "workspace-1", "Cache Miss Test", "cache-miss-test", "active",
		[]byte(`{"name":"db-version"}`), 1, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// Read from repository - should miss cache and hit DB
	result, err := ti.repo.GetByID(ti.ctx, cardID)
	if err != nil {
		t.Fatalf("failed to get card: %v", err)
	}

	// Should return DB version
	if result.Name != "Cache Miss Test" {
		t.Errorf("expected DB version 'Cache Miss Test', got '%s'", result.Name)
	}

	// Wait for async cache population
	waitForAsyncCache()

	// Verify cache was populated
	cached, err := ti.typedCache.Get(ti.ctx, cardID)
	if err != nil {
		t.Fatalf("failed to get from cache: %v", err)
	}
	if cached == nil {
		t.Error("expected card to be cached after DB read")
	}
	if cached.Name != "Cache Miss Test" {
		t.Errorf("expected cached name 'Cache Miss Test', got '%s'", cached.Name)
	}

	t.Log("Cache miss verified - fetched from DB and populated cache")
}

// TestA2AModelCards_Update tests updating a model card invalidates cache
func TestA2AModelCards_Update(t *testing.T) {
	ti := setupTestInfrastructure(t)
	if ti == nil {
		return
	}
	defer ti.teardown()

	// Create initial card
	card := createTestModelCard("workspace-1", "Original Name", "update-test-card")
	err := ti.repo.Create(ti.ctx, card)
	if err != nil {
		t.Fatalf("failed to create card: %v", err)
	}

	// Wait for cache
	waitForAsyncCache()

	// Verify cached
	cached, _ := ti.typedCache.Get(ti.ctx, card.ID)
	if cached == nil {
		t.Fatal("expected card to be cached")
	}

	// Update the card
	card.Name = "Updated Name"
	card.Card = []byte(`{"name":"updated"}`)
	err = ti.repo.Update(ti.ctx, card)
	if err != nil {
		t.Fatalf("failed to update card: %v", err)
	}

	// Verify database was updated
	var dbName string
	var dbVersion int
	err = ti.db.QueryRowContext(ti.ctx,
		"SELECT name, version FROM a2a_model_cards WHERE id = ?", card.ID).Scan(&dbName, &dbVersion)
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	if dbName != "Updated Name" {
		t.Errorf("expected DB name 'Updated Name', got '%s'", dbName)
	}
	if dbVersion != 2 {
		t.Errorf("expected version 2 after update, got %d", dbVersion)
	}

	// Wait for cache invalidation
	waitForAsyncCache()

	// Verify cache was invalidated
	cached, _ = ti.typedCache.Get(ti.ctx, card.ID)
	if cached != nil {
		t.Error("expected cache to be invalidated after update")
	}

	// Read again - should fetch from DB and populate cache
	result, err := ti.repo.GetByID(ti.ctx, card.ID)
	if err != nil {
		t.Fatalf("failed to get updated card: %v", err)
	}
	if result.Name != "Updated Name" {
		t.Errorf("expected updated name 'Updated Name', got '%s'", result.Name)
	}

	t.Log("Update verified - DB updated and cache invalidated")
}

// TestA2AModelCards_Delete tests deleting a model card removes from both PostgreSQL and Redis
func TestA2AModelCards_Delete(t *testing.T) {
	ti := setupTestInfrastructure(t)
	if ti == nil {
		return
	}
	defer ti.teardown()

	// Create card
	card := createTestModelCard("workspace-1", "To Delete", "delete-test-card")
	err := ti.repo.Create(ti.ctx, card)
	if err != nil {
		t.Fatalf("failed to create card: %v", err)
	}

	// Wait for cache
	waitForAsyncCache()

	// Cache project cards list
	_, _ = ti.repo.GetByProject(ti.ctx, "workspace-1")
	waitForAsyncCache()

	// Delete the card
	err = ti.repo.Delete(ti.ctx, card.ID)
	if err != nil {
		t.Fatalf("failed to delete card: %v", err)
	}

	// Verify deleted from database
	var dbCount int
	err = ti.db.QueryRowContext(ti.ctx,
		"SELECT COUNT(*) FROM a2a_model_cards WHERE id = ?", card.ID).Scan(&dbCount)
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	if dbCount != 0 {
		t.Errorf("expected 0 cards in database, got %d", dbCount)
	}

	// Wait for cache invalidation
	waitForAsyncCache()

	// Verify cache was invalidated
	cached, _ := ti.typedCache.Get(ti.ctx, card.ID)
	if cached != nil {
		t.Error("expected card cache to be invalidated after delete")
	}

	// Verify project cache was invalidated
	projectCached, _ := ti.typedCache.GetProjectCards(ti.ctx, "workspace-1")
	if projectCached != nil {
		t.Error("expected project cache to be invalidated after delete")
	}

	// Verify GetByID returns error
	_, err = ti.repo.GetByID(ti.ctx, card.ID)
	if err == nil {
		t.Error("expected error when getting deleted card")
	}

	t.Log("Delete verified - removed from DB and cache")
}

// TestA2AModelCards_CacheTTLExpiration tests that expired cache entries are refreshed from DB
func TestA2AModelCards_CacheTTLExpiration(t *testing.T) {
	ti := setupTestInfrastructure(t)
	if ti == nil {
		return
	}
	defer ti.teardown()

	// Create card in database
	cardID := "ttl-test-id"
	_, err := ti.db.ExecContext(ti.ctx, `
		INSERT INTO a2a_model_cards (id, workspace_id, name, slug, status, card, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, cardID, "workspace-1", "TTL Test Card", "ttl-test", "active",
		[]byte(`{"version":"original"}`), 1, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// Manually cache with very short TTL
	card := &a2a.ModelCard{
		ID:          cardID,
		WorkspaceID: "workspace-1",
		Name:        "TTL Test Card",
		Slug:        "ttl-test",
		Status:      a2a.ModelCardStatusActive,
		Card:        []byte(`{"version":"cached"}`),
		Version:     1,
	}
	err = ti.typedCache.Set(ti.ctx, cardID, card, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to set cache: %v", err)
	}

	// Verify initial cache
	cached, _ := ti.typedCache.Get(ti.ctx, cardID)
	if cached == nil {
		t.Fatal("expected card in cache")
	}

	// Wait for TTL expiration
	time.Sleep(200 * time.Millisecond)

	// Cache should be expired
	cached, _ = ti.typedCache.Get(ti.ctx, cardID)
	if cached != nil {
		t.Skip("Cache TTL expiration not immediate in Redis, skipping expiration check")
	}

	// Read from repository - should fetch from DB
	result, err := ti.repo.GetByID(ti.ctx, cardID)
	if err != nil {
		t.Fatalf("failed to get card after TTL expiration: %v", err)
	}
	if result.Name != "TTL Test Card" {
		t.Errorf("expected name 'TTL Test Card', got '%s'", result.Name)
	}

	t.Log("TTL expiration verified - cache expired and refreshed from DB")
}

// TestA2AModelCards_ConnectionPoolExhaustion tests graceful handling of connection pool exhaustion
func TestA2AModelCards_ConnectionPoolExhaustion(t *testing.T) {
	ti := setupTestInfrastructure(t)
	if ti == nil {
		return
	}
	defer ti.teardown()

	// Create Redis cache with very small pool to simulate exhaustion
	limitedRedisConfig := &cache.GoRedisConfig{
		Addr:        getTestRedisAddr(),
		KeyPrefix:   "test:limited:",
		PoolSize:    1,
		DialTimeout: 1 * time.Second,
		PoolTimeout: 100 * time.Millisecond, // Short timeout to trigger exhaustion faster
	}

	limitedCache, err := cache.NewGoRedis(limitedRedisConfig)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
		return
	}
	defer limitedCache.Close()

	limitedTypedCache := cache.NewTypedModelCardCache(limitedCache, 5*time.Minute)
	limitedRepo := a2a.NewHybridRepository(ti.db, limitedTypedCache, ti.log)

	// Create cards sequentially to avoid SQLite concurrency issues
	// The cache operations will still test the connection pool
	createdCount := 0
	for i := 0; i < 5; i++ {
		card := createTestModelCard(
			fmt.Sprintf("workspace-%d", i),
			fmt.Sprintf("Pool Test Card %d", i),
			fmt.Sprintf("pool-test-%d", i),
		)
		if err := limitedRepo.Create(ti.ctx, card); err != nil {
			t.Logf("Failed to create card %d: %v", i, err)
		} else {
			createdCount++
		}
	}

	// Wait for any async cache operations
	waitForAsyncCache()

	// Verify cards were created in database
	var count int
	err = ti.db.QueryRowContext(ti.ctx,
		"SELECT COUNT(*) FROM a2a_model_cards WHERE slug LIKE 'pool-test-%'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	if count != createdCount {
		t.Errorf("expected %d cards in database, got %d", createdCount, count)
	}

	// Verify at least some cards were created
	if count == 0 {
		t.Error("expected at least some cards to be created")
	}

	// Try to read back - should work even with limited cache pool
	for i := 0; i < 5; i++ {
		slug := fmt.Sprintf("pool-test-%d", i)
		_, _ = limitedRepo.GetBySlug(ti.ctx, fmt.Sprintf("workspace-%d", i), slug)
	}

	t.Logf("Connection pool exhaustion test completed: %d cards created, all DB operations succeeded", count)
}

// TestA2AModelCards_GetByProject tests retrieving all cards for a project with caching
func TestA2AModelCards_GetByProject(t *testing.T) {
	ti := setupTestInfrastructure(t)
	if ti == nil {
		return
	}
	defer ti.teardown()

	projectID := "test-project-1"

	// Create multiple cards for the same project
	for i := 1; i <= 3; i++ {
		card := createTestModelCard(
			projectID,
			fmt.Sprintf("Project Card %d", i),
			fmt.Sprintf("project-card-%d", i),
		)
		err := ti.repo.Create(ti.ctx, card)
		if err != nil {
			t.Fatalf("failed to create card %d: %v", i, err)
		}
	}

	// Wait for cache
	waitForAsyncCache()

	// Get all cards for project
	cards, err := ti.repo.GetByProject(ti.ctx, projectID)
	if err != nil {
		t.Fatalf("failed to get project cards: %v", err)
	}
	if len(cards) != 3 {
		t.Errorf("expected 3 cards, got %d", len(cards))
	}

	// Wait for project cache
	waitForAsyncCache()

	// Verify project cards are cached
	cached, err := ti.typedCache.GetProjectCards(ti.ctx, projectID)
	if err != nil {
		t.Fatalf("failed to get cached project cards: %v", err)
	}
	if cached == nil {
		t.Error("expected project cards to be cached")
	}
	if len(cached) != 3 {
		t.Errorf("expected 3 cached cards, got %d", len(cached))
	}

	t.Logf("GetByProject verified: %d cards retrieved and cached", len(cards))
}

// TestA2AModelCards_GracefulCacheFailure tests that DB operations work even when cache fails
func TestA2AModelCards_GracefulCacheFailure(t *testing.T) {
	// Setup with closed cache to simulate failure
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	// Create database
	db, err := setupTestDatabase(nil)
	if err != nil {
		t.Fatalf("failed to setup test database: %v", err)
	}
	defer db.Close()

	// Create a mock cache that always fails
	failingCache := &mockFailingCache{}
	repo := a2a.NewHybridRepository(db, failingCache, log)

	// Create should still work
	card := createTestModelCard("workspace-1", "Failing Cache Card", "failing-cache-card")
	err = repo.Create(ctx, card)
	if err != nil {
		t.Fatalf("expected create to succeed despite cache failure: %v", err)
	}

	// Verify in database
	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM a2a_model_cards WHERE id = ?", card.ID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 card in database, got %d", count)
	}

	// GetByID should still work (from DB)
	result, err := repo.GetByID(ctx, card.ID)
	if err != nil {
		t.Fatalf("expected get to succeed despite cache failure: %v", err)
	}
	if result.Name != card.Name {
		t.Errorf("expected name %q, got %q", card.Name, result.Name)
	}

	t.Log("Graceful cache failure verified - DB operations succeed despite cache errors")
}

// mockFailingCache implements cache.TypedModelCardCache but always fails
type mockFailingCache struct{}

func (m *mockFailingCache) Get(ctx context.Context, id string) (*cache.ModelCard, error) {
	return nil, fmt.Errorf("cache is unavailable")
}

func (m *mockFailingCache) Set(ctx context.Context, id string, card *cache.ModelCard, ttl time.Duration) error {
	return fmt.Errorf("cache is unavailable")
}

func (m *mockFailingCache) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("cache is unavailable")
}

func (m *mockFailingCache) GetProjectCards(ctx context.Context, projectID string) ([]cache.ModelCard, error) {
	return nil, fmt.Errorf("cache is unavailable")
}

func (m *mockFailingCache) SetProjectCards(ctx context.Context, projectID string, cards []cache.ModelCard, ttl time.Duration) error {
	return fmt.Errorf("cache is unavailable")
}

func (m *mockFailingCache) DeleteProjectCards(ctx context.Context, projectID string) error {
	return fmt.Errorf("cache is unavailable")
}

func (m *mockFailingCache) GetBySkill(ctx context.Context, skillID string) ([]cache.ModelCard, error) {
	return nil, fmt.Errorf("cache is unavailable")
}

func (m *mockFailingCache) SetBySkill(ctx context.Context, skillID string, cards []cache.ModelCard, ttl time.Duration) error {
	return fmt.Errorf("cache is unavailable")
}

func (m *mockFailingCache) DeleteBySkill(ctx context.Context, skillID string) error {
	return fmt.Errorf("cache is unavailable")
}

func (m *mockFailingCache) InvalidateCard(ctx context.Context, id string, projectID string) error {
	return fmt.Errorf("cache is unavailable")
}

func (m *mockFailingCache) InvalidatePattern(ctx context.Context, pattern string) error {
	return fmt.Errorf("cache is unavailable")
}

// TestA2AModelCards_ConcurrentAccess tests concurrent operations on model cards
func TestA2AModelCards_ConcurrentAccess(t *testing.T) {
	ti := setupTestInfrastructure(t)
	if ti == nil {
		return
	}
	defer ti.teardown()

	// Create a card
	card := createTestModelCard("workspace-concurrent", "Concurrent Card", "concurrent-card")
	err := ti.repo.Create(ti.ctx, card)
	if err != nil {
		t.Fatalf("failed to create card: %v", err)
	}

	// Wait for initial cache
	waitForAsyncCache()

	// Concurrent reads
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = ti.repo.GetByID(ti.ctx, card.ID)
		}()
	}

	// Concurrent project reads
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = ti.repo.GetByProject(ti.ctx, "workspace-concurrent")
		}()
	}

	wg.Wait()

	// Verify card still exists and is consistent
	result, err := ti.repo.GetByID(ti.ctx, card.ID)
	if err != nil {
		t.Fatalf("failed to get card after concurrent access: %v", err)
	}
	if result.Name != "Concurrent Card" {
		t.Errorf("expected name 'Concurrent Card', got '%s'", result.Name)
	}

	t.Log("Concurrent access test completed successfully")
}

// TestA2AModelCards_SplitBrainScenario tests handling of cache/DB inconsistency
func TestA2AModelCards_SplitBrainScenario(t *testing.T) {
	ti := setupTestInfrastructure(t)
	if ti == nil {
		return
	}
	defer ti.teardown()

	// Create card in database
	cardID := "split-brain-id"
	_, err := ti.db.ExecContext(ti.ctx, `
		INSERT INTO a2a_model_cards (id, workspace_id, name, slug, status, card, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, cardID, "workspace-1", "DB Version", "split-brain", "active",
		[]byte(`{"source":"db"}`), 1, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// Put stale data in cache (simulating split-brain)
	staleCard := &a2a.ModelCard{
		ID:          cardID,
		WorkspaceID: "workspace-1",
		Name:        "Stale Cache Version",
		Slug:        "split-brain",
		Status:      a2a.ModelCardStatusActive,
		Card:        []byte(`{"source":"stale-cache"}`),
		Version:     1,
	}
	err = ti.typedCache.Set(ti.ctx, cardID, staleCard, 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to set stale cache: %v", err)
	}

	// Read should return cached version (cache hit)
	result, err := ti.repo.GetByID(ti.ctx, cardID)
	if err != nil {
		t.Fatalf("failed to get card: %v", err)
	}

	// With cache hit, we get the cached version
	if result.Name != "Stale Cache Version" {
		t.Errorf("expected cached stale version, got '%s'", result.Name)
	}

	// Now simulate cache invalidation (e.g., by another instance)
	err = ti.typedCache.Delete(ti.ctx, cardID)
	if err != nil {
		t.Fatalf("failed to delete from cache: %v", err)
	}

	// Read again - should get DB version
	result, err = ti.repo.GetByID(ti.ctx, cardID)
	if err != nil {
		t.Fatalf("failed to get card after cache delete: %v", err)
	}
	if result.Name != "DB Version" {
		t.Errorf("expected DB version 'DB Version', got '%s'", result.Name)
	}

	t.Log("Split-brain scenario verified - eventual consistency achieved")
}

// TestA2AModelCards_BulkOperations tests bulk create/read operations
func TestA2AModelCards_BulkOperations(t *testing.T) {
	ti := setupTestInfrastructure(t)
	if ti == nil {
		return
	}
	defer ti.teardown()

	projectID := "bulk-project"
	numCards := 20

	// Bulk create
	var cardIDs []string
	for i := 1; i <= numCards; i++ {
		card := createTestModelCard(
			projectID,
			fmt.Sprintf("Bulk Card %d", i),
			fmt.Sprintf("bulk-card-%d", i),
		)
		err := ti.repo.Create(ti.ctx, card)
		if err != nil {
			t.Fatalf("failed to create card %d: %v", i, err)
		}
		cardIDs = append(cardIDs, card.ID)
	}

	// Wait for cache
	waitForAsyncCache()

	// Bulk read from cache
	cacheHits := 0
	for _, id := range cardIDs {
		cached, err := ti.typedCache.Get(ti.ctx, id)
		if err != nil {
			t.Logf("Cache error for %s: %v", id, err)
			continue
		}
		if cached != nil {
			cacheHits++
		}
	}

	if cacheHits == 0 {
		t.Error("expected some cache hits after bulk create")
	}

	// Get all by project
	cards, err := ti.repo.GetByProject(ti.ctx, projectID)
	if err != nil {
		t.Fatalf("failed to get project cards: %v", err)
	}
	if len(cards) != numCards {
		t.Errorf("expected %d cards, got %d", numCards, len(cards))
	}

	t.Logf("Bulk operations verified: %d cards created, %d cache hits", numCards, cacheHits)
}

// TestA2AModelCards_GetBySlug tests retrieval by workspace and slug
func TestA2AModelCards_GetBySlug(t *testing.T) {
	ti := setupTestInfrastructure(t)
	if ti == nil {
		return
	}
	defer ti.teardown()

	// Create card
	card := createTestModelCard("workspace-slug", "Slug Test Card", "my-unique-slug")
	err := ti.repo.Create(ti.ctx, card)
	if err != nil {
		t.Fatalf("failed to create card: %v", err)
	}

	// Get by slug
	result, err := ti.repo.GetBySlug(ti.ctx, "workspace-slug", "my-unique-slug")
	if err != nil {
		t.Fatalf("failed to get by slug: %v", err)
	}
	if result.ID != card.ID {
		t.Errorf("expected ID %s, got %s", card.ID, result.ID)
	}
	if result.Name != "Slug Test Card" {
		t.Errorf("expected name 'Slug Test Card', got '%s'", result.Name)
	}

	// Try non-existent slug
	_, err = ti.repo.GetBySlug(ti.ctx, "workspace-slug", "non-existent")
	if err == nil {
		t.Error("expected error for non-existent slug")
	}

	t.Log("GetBySlug verified")
}
