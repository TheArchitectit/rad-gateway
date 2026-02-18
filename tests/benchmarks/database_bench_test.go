// Package benchmarks provides performance benchmarks for critical paths.
package benchmarks

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"radgateway/internal/cache"
)

// getPostgresDSN returns the PostgreSQL DSN for benchmarks.
func getPostgresDSN() string {
	dsn := os.Getenv("POSTGRES_BENCH_DSN")
	if dsn == "" {
		// Default for local development
		dsn = "postgres://postgres:postgres@localhost:5432/radgateway_test?sslmode=disable"
	}
	return dsn
}

// skipIfNoPostgres skips benchmark if PostgreSQL is unavailable.
func skipIfNoPostgres(b *testing.B) *sql.DB {
	dsn := getPostgresDSN()
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		b.Skipf("PostgreSQL not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := database.PingContext(ctx); err != nil {
		database.Close()
		b.Skipf("PostgreSQL not available: %v", err)
	}

	return database
}

// setupBenchmarkDB initializes the database with test tables.
func setupBenchmarkDB(b *testing.B, database *sql.DB) {
	ctx := context.Background()

	// Create test tables
	schema := `
	CREATE TABLE IF NOT EXISTS bench_model_cards (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		user_id TEXT,
		name TEXT NOT NULL,
		slug TEXT NOT NULL,
		description TEXT,
		card JSONB,
		version INTEGER DEFAULT 1,
		status TEXT DEFAULT 'active',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_bench_cards_workspace ON bench_model_cards(workspace_id);
	CREATE INDEX IF NOT EXISTS idx_bench_cards_slug ON bench_model_cards(workspace_id, slug);
	CREATE INDEX IF NOT EXISTS idx_bench_cards_card_gin ON bench_model_cards USING GIN(card);

	CREATE TABLE IF NOT EXISTS bench_users (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		email TEXT NOT NULL,
		display_name TEXT,
		status TEXT DEFAULT 'active',
		password_hash TEXT,
		last_login_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_bench_users_workspace ON bench_users(workspace_id);
	CREATE INDEX IF NOT EXISTS idx_bench_users_email ON bench_users(email);
	`

	if _, err := database.ExecContext(ctx, schema); err != nil {
		b.Fatalf("Failed to create benchmark tables: %v", err)
	}

	// Clear existing data
	database.ExecContext(ctx, "TRUNCATE TABLE bench_model_cards, bench_users")
}

// cleanupBenchmarkDB removes test data.
func cleanupBenchmarkDB(b *testing.B, database *sql.DB) {
	ctx := context.Background()
	database.ExecContext(ctx, "TRUNCATE TABLE bench_model_cards, bench_users")
}

// BenchmarkPostgresSimpleQuery benchmarks simple SELECT queries.
func BenchmarkPostgresSimpleQuery(b *testing.B) {
	database := skipIfNoPostgres(b)
	defer database.Close()
	setupBenchmarkDB(b, database)
	defer cleanupBenchmarkDB(b, database)

	ctx := context.Background()

	// Insert test data
	for i := 0; i < 1000; i++ {
		_, err := database.ExecContext(ctx,
			"INSERT INTO bench_users (id, workspace_id, email, display_name) VALUES ($1, $2, $3, $4)",
			fmt.Sprintf("user-%d", i), "workspace-1", fmt.Sprintf("user%d@test.com", i), fmt.Sprintf("User %d", i))
		if err != nil {
			b.Fatalf("Failed to insert test data: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var id, email string
		err := database.QueryRowContext(ctx,
			"SELECT id, email FROM bench_users WHERE id = $1", fmt.Sprintf("user-%d", i%1000)).
			Scan(&id, &email)
		if err != nil {
			b.Errorf("Query failed: %v", err)
		}
	}
}

// BenchmarkPostgresJSONBQuery benchmarks JSONB column queries.
func BenchmarkPostgresJSONBQuery(b *testing.B) {
	database := skipIfNoPostgres(b)
	defer database.Close()
	setupBenchmarkDB(b, database)
	defer cleanupBenchmarkDB(b, database)

	ctx := context.Background()

	// Insert test data with JSONB
	cardData := map[string]interface{}{
		"name": "Test Model",
		"capabilities": map[string]interface{}{
			"streaming": true,
			"vision":    true,
			"code":      false,
		},
		"metadata": map[string]interface{}{
			"provider": "openai",
			"family":   "gpt-4",
		},
	}
	cardJSON, _ := json.Marshal(cardData)

	for i := 0; i < 1000; i++ {
		_, err := database.ExecContext(ctx,
			"INSERT INTO bench_model_cards (id, workspace_id, name, slug, card) VALUES ($1, $2, $3, $4, $5)",
			fmt.Sprintf("card-%d", i), "workspace-1", fmt.Sprintf("Model %d", i), fmt.Sprintf("model-%d", i), cardJSON)
		if err != nil {
			b.Fatalf("Failed to insert test data: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var id string
		var card json.RawMessage
		err := database.QueryRowContext(ctx,
			"SELECT id, card FROM bench_model_cards WHERE id = $1", fmt.Sprintf("card-%d", i%1000)).
			Scan(&id, &card)
		if err != nil {
			b.Errorf("Query failed: %v", err)
		}
	}
}

// BenchmarkPostgresJSONBIndexQuery benchmarks JSONB queries using GIN index.
func BenchmarkPostgresJSONBIndexQuery(b *testing.B) {
	database := skipIfNoPostgres(b)
	defer database.Close()
	setupBenchmarkDB(b, database)
	defer cleanupBenchmarkDB(b, database)

	ctx := context.Background()

	// Insert test data with varying capabilities
	for i := 0; i < 1000; i++ {
		cardData := map[string]interface{}{
			"name": fmt.Sprintf("Model %d", i),
			"capabilities": map[string]interface{}{
				"streaming": i%2 == 0,
				"vision":    i%3 == 0,
				"code":      i%5 == 0,
			},
		}
		cardJSON, _ := json.Marshal(cardData)
		_, err := database.ExecContext(ctx,
			"INSERT INTO bench_model_cards (id, workspace_id, name, slug, card) VALUES ($1, $2, $3, $4, $5)",
			fmt.Sprintf("card-%d", i), "workspace-1", fmt.Sprintf("Model %d", i), fmt.Sprintf("model-%d", i), cardJSON)
		if err != nil {
			b.Fatalf("Failed to insert test data: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := database.QueryContext(ctx,
			"SELECT id, name FROM bench_model_cards WHERE workspace_id = $1 AND card->'capabilities'->>'streaming' = 'true' LIMIT 10",
			"workspace-1")
		if err != nil {
			b.Errorf("Query failed: %v", err)
			continue
		}
		rows.Close()
	}
}

// BenchmarkPostgresInsert benchmarks INSERT operations.
func BenchmarkPostgresInsert(b *testing.B) {
	database := skipIfNoPostgres(b)
	defer database.Close()
	setupBenchmarkDB(b, database)
	defer cleanupBenchmarkDB(b, database)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := database.ExecContext(ctx,
			"INSERT INTO bench_users (id, workspace_id, email, display_name) VALUES ($1, $2, $3, $4)",
			fmt.Sprintf("user-%d", i), "workspace-1", fmt.Sprintf("user%d@test.com", i), fmt.Sprintf("User %d", i))
		if err != nil {
			b.Errorf("Insert failed: %v", err)
		}
	}
}

// BenchmarkPostgresBatchInsert benchmarks batch INSERT operations.
func BenchmarkPostgresBatchInsert(b *testing.B) {
	database := skipIfNoPostgres(b)
	defer database.Close()
	setupBenchmarkDB(b, database)
	defer cleanupBenchmarkDB(b, database)

	ctx := context.Background()
	batchSize := 100

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			b.Fatalf("Failed to begin transaction: %v", err)
		}

		stmt, err := tx.PrepareContext(ctx,
			"INSERT INTO bench_users (id, workspace_id, email, display_name) VALUES ($1, $2, $3, $4)")
		if err != nil {
			b.Fatalf("Failed to prepare statement: %v", err)
		}

		base := i * batchSize
		for j := 0; j < batchSize; j++ {
			_, err := stmt.ExecContext(ctx,
				fmt.Sprintf("user-%d", base+j), "workspace-1",
				fmt.Sprintf("user%d@test.com", base+j), fmt.Sprintf("User %d", base+j))
			if err != nil {
				b.Errorf("Batch insert failed: %v", err)
			}
		}

		stmt.Close()
		if err := tx.Commit(); err != nil {
			b.Errorf("Commit failed: %v", err)
		}
	}
}

// BenchmarkPostgresUpdate benchmarks UPDATE operations.
func BenchmarkPostgresUpdate(b *testing.B) {
	database := skipIfNoPostgres(b)
	defer database.Close()
	setupBenchmarkDB(b, database)
	defer cleanupBenchmarkDB(b, database)

	ctx := context.Background()

	// Insert test data
	for i := 0; i < 1000; i++ {
		_, err := database.ExecContext(ctx,
			"INSERT INTO bench_users (id, workspace_id, email, display_name) VALUES ($1, $2, $3, $4)",
			fmt.Sprintf("user-%d", i), "workspace-1", fmt.Sprintf("user%d@test.com", i), fmt.Sprintf("User %d", i))
		if err != nil {
			b.Fatalf("Failed to insert test data: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := database.ExecContext(ctx,
			"UPDATE bench_users SET display_name = $1, updated_at = $2 WHERE id = $3",
			fmt.Sprintf("Updated User %d", i), time.Now(), fmt.Sprintf("user-%d", i%1000))
		if err != nil {
			b.Errorf("Update failed: %v", err)
		}
	}
}

// BenchmarkPostgresDelete benchmarks DELETE operations.
func BenchmarkPostgresDelete(b *testing.B) {
	database := skipIfNoPostgres(b)
	defer database.Close()
	setupBenchmarkDB(b, database)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Insert then delete to keep benchmark consistent
		b.StopTimer()
		id := fmt.Sprintf("delete-user-%d", i)
		_, err := database.ExecContext(ctx,
			"INSERT INTO bench_users (id, workspace_id, email, display_name) VALUES ($1, $2, $3, $4)",
			id, "workspace-1", fmt.Sprintf("%s@test.com", id), "Test User")
		if err != nil {
			b.Fatalf("Failed to insert test data: %v", err)
		}
		b.StartTimer()

		_, err = database.ExecContext(ctx, "DELETE FROM bench_users WHERE id = $1", id)
		if err != nil {
			b.Errorf("Delete failed: %v", err)
		}
	}
}

// BenchmarkPostgresComplexJoin benchmarks complex JOIN queries.
func BenchmarkPostgresComplexJoin(b *testing.B) {
	database := skipIfNoPostgres(b)
	defer database.Close()
	setupBenchmarkDB(b, database)
	defer cleanupBenchmarkDB(b, database)

	ctx := context.Background()

	// Insert test data
	for i := 0; i < 1000; i++ {
		_, err := database.ExecContext(ctx,
			"INSERT INTO bench_users (id, workspace_id, email, display_name) VALUES ($1, $2, $3, $4)",
			fmt.Sprintf("user-%d", i), "workspace-1", fmt.Sprintf("user%d@test.com", i), fmt.Sprintf("User %d", i))
		if err != nil {
			b.Fatalf("Failed to insert test data: %v", err)
		}
	}

	for i := 0; i < 1000; i++ {
		cardJSON, _ := json.Marshal(map[string]interface{}{"name": fmt.Sprintf("Model %d", i)})
		_, err := database.ExecContext(ctx,
			"INSERT INTO bench_model_cards (id, workspace_id, user_id, name, slug, card) VALUES ($1, $2, $3, $4, $5, $6)",
			fmt.Sprintf("card-%d", i), "workspace-1", fmt.Sprintf("user-%d", i%100),
			fmt.Sprintf("Model %d", i), fmt.Sprintf("model-%d", i), cardJSON)
		if err != nil {
			b.Fatalf("Failed to insert test data: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := database.QueryContext(ctx, `
			SELECT u.id, u.email, c.name, c.card
			FROM bench_users u
			JOIN bench_model_cards c ON c.user_id = u.id
			WHERE u.workspace_id = $1
			LIMIT 100`, "workspace-1")
		if err != nil {
			b.Errorf("Query failed: %v", err)
			continue
		}
		rows.Close()
	}
}

// BenchmarkPostgresPoolSizeComparison compares different connection pool sizes.
func BenchmarkPostgresPoolSizeComparison(b *testing.B) {
	dsn := getPostgresDSN()

	poolSizes := []int{1, 5, 10, 20}

	for _, poolSize := range poolSizes {
		b.Run(fmt.Sprintf("PoolSize%d", poolSize), func(b *testing.B) {
			database, err := sql.Open("postgres", dsn)
			if err != nil {
				b.Skipf("PostgreSQL not available: %v", err)
			}
			defer database.Close()

			database.SetMaxOpenConns(poolSize)
			database.SetMaxIdleConns(poolSize)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			if err := database.PingContext(ctx); err != nil {
				b.Skipf("PostgreSQL not available: %v", err)
			}

			// Setup
			database.ExecContext(context.Background(), "CREATE TABLE IF NOT EXISTS bench_pool_test (id TEXT PRIMARY KEY, data TEXT)")
			database.ExecContext(context.Background(), "TRUNCATE TABLE bench_pool_test")

			for i := 0; i < 100; i++ {
				database.ExecContext(context.Background(),
					"INSERT INTO bench_pool_test (id, data) VALUES ($1, $2)",
					fmt.Sprintf("key-%d", i), "test-data")
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					var data string
					database.QueryRowContext(context.Background(),
						"SELECT data FROM bench_pool_test WHERE id = $1",
						fmt.Sprintf("key-%d", i%100)).Scan(&data)
					i++
				}
			})

			database.ExecContext(context.Background(), "DROP TABLE IF EXISTS bench_pool_test")
		})
	}
}

// getBenchmarkRedisAddr returns Redis address for benchmarks.
func getBenchmarkRedisAddr() string {
	addr := os.Getenv("REDIS_BENCH_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	return addr
}

// skipIfNoRedisForBenchmark skips benchmark if Redis is unavailable.
func skipIfNoRedisForBenchmark(b *testing.B) cache.Cache {
	config := &cache.GoRedisConfig{
		Addr:        getBenchmarkRedisAddr(),
		DialTimeout: 2 * time.Second,
		KeyPrefix:   "bench:",
	}

	c, err := cache.NewGoRedis(config)
	if err != nil {
		b.Skipf("Redis not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := c.Ping(ctx); err != nil {
		c.Close()
		b.Skipf("Redis not available: %v", err)
	}

	return c
}

// BenchmarkRedisGet benchmarks Redis GET operations.
func BenchmarkRedisGet(b *testing.B) {
	c := skipIfNoRedisForBenchmark(b)
	defer c.Close()

	ctx := context.Background()
	value := []byte("benchmark-value-data")

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("get-key-%d", i)
		c.Set(ctx, key, value, time.Hour)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("get-key-%d", i%1000)
			c.Get(ctx, key)
			i++
		}
	})

	// Cleanup
	c.DeletePattern(ctx, "*")
}

// BenchmarkRedisSet benchmarks Redis SET operations.
func BenchmarkRedisSet(b *testing.B) {
	c := skipIfNoRedisForBenchmark(b)
	defer c.Close()

	ctx := context.Background()
	value := []byte("benchmark-value-data")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("set-key-%d", i)
			c.Set(ctx, key, value, time.Minute)
			i++
		}
	})

	// Cleanup
	c.DeletePattern(ctx, "*")
}

// BenchmarkRedisSetAndGet benchmarks combined SET and GET operations.
func BenchmarkRedisSetAndGet(b *testing.B) {
	c := skipIfNoRedisForBenchmark(b)
	defer c.Close()

	ctx := context.Background()
	value := []byte("benchmark-value-data")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("combo-key-%d", i)
			c.Set(ctx, key, value, time.Minute)
			c.Get(ctx, key)
			i++
		}
	})

	// Cleanup
	c.DeletePattern(ctx, "*")
}

// BenchmarkRedisCacheHit benchmarks cache hit performance.
func BenchmarkRedisCacheHit(b *testing.B) {
	c := skipIfNoRedisForBenchmark(b)
	defer c.Close()

	ctx := context.Background()
	value := []byte("cached-value-data")

	// Pre-populate all keys
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("hit-key-%d", i)
		c.Set(ctx, key, value, time.Hour)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("hit-key-%d", i%1000)
		data, err := c.Get(ctx, key)
		if err != nil || data == nil {
			b.Errorf("Expected cache hit for key %s", key)
		}
	}

	// Cleanup
	c.DeletePattern(ctx, "*")
}

// BenchmarkRedisCacheMiss benchmarks cache miss performance.
func BenchmarkRedisCacheMiss(b *testing.B) {
	c := skipIfNoRedisForBenchmark(b)
	defer c.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("miss-key-%d", i)
		data, err := c.Get(ctx, key)
		if err != nil || data != nil {
			b.Errorf("Expected cache miss for key %s", key)
		}
	}
}

// BenchmarkRedisJSONValue benchmarks storing/retrieving JSON values.
func BenchmarkRedisJSONValue(b *testing.B) {
	c := skipIfNoRedisForBenchmark(b)
	defer c.Close()

	ctx := context.Background()

	// Create a complex JSON value
	jsonValue := map[string]interface{}{
		"id":          "model-123",
		"name":        "Test Model",
		"description": "A test model for benchmarking",
		"capabilities": []string{"streaming", "vision", "code"},
		"metadata": map[string]interface{}{
			"provider": "openai",
			"version":  "1.0.0",
			"pricing": map[string]float64{
				"input":  0.0001,
				"output": 0.0002,
			},
		},
	}
	data, _ := json.Marshal(jsonValue)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("json-key-%d", i)
		c.Set(ctx, key, data, time.Minute)
		c.Get(ctx, key)
	}

	// Cleanup
	c.DeletePattern(ctx, "*")
}

// BenchmarkRedisDelete benchmarks DELETE operations.
func BenchmarkRedisDelete(b *testing.B) {
	c := skipIfNoRedisForBenchmark(b)
	defer c.Close()

	ctx := context.Background()
	value := []byte("delete-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		key := fmt.Sprintf("delete-key-%d", i)
		c.Set(ctx, key, value, time.Hour)
		b.StartTimer()

		c.Delete(ctx, key)
	}
}

// BenchmarkRedisExpire benchmarks EXPIRE operations.
func BenchmarkRedisExpire(b *testing.B) {
	c := skipIfNoRedisForBenchmark(b)
	defer c.Close()

	ctx := context.Background()
	value := []byte("expire-value")

	// Pre-populate keys
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("expire-key-%d", i)
		c.Set(ctx, key, value, time.Hour)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("expire-key-%d", i%1000)
		c.Set(ctx, key, value, time.Minute)
	}

	// Cleanup
	c.DeletePattern(ctx, "*")
}

// BenchmarkRedisPipeline benchmarks pipeline operations.
func BenchmarkRedisPipeline(b *testing.B) {
	c := skipIfNoRedisForBenchmark(b)
	defer c.Close()

	ctx := context.Background()
	value := []byte("pipeline-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate pipeline by doing multiple operations
		for j := 0; j < 10; j++ {
			key := fmt.Sprintf("pipeline-key-%d-%d", i, j)
			c.Set(ctx, key, value, time.Minute)
		}
	}

	// Cleanup
	c.DeletePattern(ctx, "*")
}

// BenchmarkRedisLargeValue benchmarks large value storage.
func BenchmarkRedisLargeValue(b *testing.B) {
	c := skipIfNoRedisForBenchmark(b)
	defer c.Close()

	ctx := context.Background()

	// Create values of different sizes
	smallValue := make([]byte, 100)
	mediumValue := make([]byte, 1024)
	largeValue := make([]byte, 1024*1024) // 1MB

	for i := range smallValue {
		smallValue[i] = byte('a')
	}
	for i := range mediumValue {
		mediumValue[i] = byte('b')
	}
	for i := range largeValue {
		largeValue[i] = byte('c')
	}

	b.Run("Small100B", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("small-%d", i)
			c.Set(ctx, key, smallValue, time.Minute)
		}
	})

	b.Run("Medium1KB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("medium-%d", i)
			c.Set(ctx, key, mediumValue, time.Minute)
		}
	})

	b.Run("Large1MB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("large-%d", i)
			c.Set(ctx, key, largeValue, time.Minute)
		}
	})

	// Cleanup
	c.DeletePattern(ctx, "*")
}

// BenchmarkCacheHitVsMiss compares cache hit vs miss performance.
func BenchmarkCacheHitVsMiss(b *testing.B) {
	c := skipIfNoRedisForBenchmark(b)
	defer c.Close()

	ctx := context.Background()
	value := []byte("test-value")

	// Pre-populate half the keys
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("hm-key-%d", i)
		c.Set(ctx, key, value, time.Hour)
	}

	b.Run("Hit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("hm-key-%d", i%500) // These exist
			c.Get(ctx, key)
		}
	})

	b.Run("Miss", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("miss-key-%d", i+10000) // These don't exist
			c.Get(ctx, key)
		}
	})

	// Cleanup
	c.DeletePattern(ctx, "*")
}

// BenchmarkHybridRepositoryCacheAside benchmarks the hybrid PostgreSQL+Redis pattern.
func BenchmarkHybridRepositoryCacheAside(b *testing.B) {
	database := skipIfNoPostgres(b)
	defer database.Close()
	setupBenchmarkDB(b, database)
	defer cleanupBenchmarkDB(b, database)

	c := skipIfNoRedisForBenchmark(b)
	defer c.Close()

	ctx := context.Background()

	// Create typed cache
	typedCache := cache.NewTypedModelCardCache(c, 5*time.Minute)

	// Insert test data into PostgreSQL
	cardJSON, _ := json.Marshal(map[string]interface{}{
		"name": "Test Model",
		"capabilities": map[string]interface{}{
			"streaming": true,
		},
	})

	for i := 0; i < 100; i++ {
		_, err := database.ExecContext(ctx,
			"INSERT INTO bench_model_cards (id, workspace_id, name, slug, card) VALUES ($1, $2, $3, $4, $5)",
			fmt.Sprintf("hybrid-card-%d", i), "workspace-1",
			fmt.Sprintf("Model %d", i), fmt.Sprintf("model-%d", i), cardJSON)
		if err != nil {
			b.Fatalf("Failed to insert test data: %v", err)
		}
	}

	b.Run("CacheHit", func(b *testing.B) {
		// Pre-populate cache
		for i := 0; i < 100; i++ {
			card := &cache.ModelCard{
				ID:          fmt.Sprintf("hybrid-card-%d", i),
				WorkspaceID: "workspace-1",
				Name:        fmt.Sprintf("Model %d", i),
				Slug:        fmt.Sprintf("model-%d", i),
			}
			typedCache.Set(ctx, card.ID, card, time.Hour)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			id := fmt.Sprintf("hybrid-card-%d", i%100)
			typedCache.Get(ctx, id) // Should hit cache
		}
	})

	b.Run("CacheMiss", func(b *testing.B) {
		// Clear cache
		c.DeletePattern(ctx, "*")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			id := fmt.Sprintf("hybrid-card-%d", i%100)
			// Simulate cache miss by querying DB directly
			var card cache.ModelCard
			var cardData []byte
			database.QueryRowContext(ctx,
				"SELECT id, workspace_id, name, slug, card FROM bench_model_cards WHERE id = $1", id).
				Scan(&card.ID, &card.WorkspaceID, &card.Name, &card.Slug, &cardData)
		}
	})

	// Cleanup
	c.DeletePattern(ctx, "*")
}

// BenchmarkJSONBVsPlainText compares JSONB vs plain text storage.
func BenchmarkJSONBVsPlainText(b *testing.B) {
	database := skipIfNoPostgres(b)
	defer database.Close()
	setupBenchmarkDB(b, database)
	defer cleanupBenchmarkDB(b, database)

	ctx := context.Background()

	// Create table with text column for comparison
	database.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS bench_cards_text (id TEXT PRIMARY KEY, workspace_id TEXT, data TEXT)")
	database.ExecContext(ctx, "TRUNCATE TABLE bench_cards_text")

	cardData := map[string]interface{}{
		"name": "Test Model",
		"capabilities": map[string]interface{}{
			"streaming": true,
			"vision":    true,
			"code":      false,
		},
		"metadata": map[string]interface{}{
			"provider": "openai",
			"version":  "1.0.0",
		},
	}
	jsonData, _ := json.Marshal(cardData)

	// Insert test data
	for i := 0; i < 1000; i++ {
		database.ExecContext(ctx,
			"INSERT INTO bench_model_cards (id, workspace_id, name, slug, card) VALUES ($1, $2, $3, $4, $5)",
			fmt.Sprintf("jsonb-card-%d", i), "workspace-1", fmt.Sprintf("Model %d", i), fmt.Sprintf("model-%d", i), jsonData)
		database.ExecContext(ctx,
			"INSERT INTO bench_cards_text (id, workspace_id, data) VALUES ($1, $2, $3)",
			fmt.Sprintf("text-card-%d", i), "workspace-1", string(jsonData))
	}

	b.Run("JSONB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var card json.RawMessage
			database.QueryRowContext(ctx,
				"SELECT card FROM bench_model_cards WHERE id = $1",
				fmt.Sprintf("jsonb-card-%d", i%1000)).Scan(&card)
		}
	})

	b.Run("PlainText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var data string
			database.QueryRowContext(ctx,
				"SELECT data FROM bench_cards_text WHERE id = $1",
				fmt.Sprintf("text-card-%d", i%1000)).Scan(&data)
		}
	})

	database.ExecContext(ctx, "DROP TABLE IF EXISTS bench_cards_text")
}
