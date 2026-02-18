package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// getBenchmarkRedisAddr returns Redis address for benchmarks
func getBenchmarkRedisAddr() string {
	return getTestRedisAddr()
}

// skipIfNoRedisForBenchmark skips benchmark if Redis unavailable
func skipIfNoRedisForBenchmark(b *testing.B) {
	config := &GoRedisConfig{
		Addr:        getBenchmarkRedisAddr(),
		DialTimeout: 2 * time.Second,
		KeyPrefix:   "bench:",
	}

	cache, err := NewGoRedis(config)
	if err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	cache.Close()
}

// BenchmarkGoRedisCache_Set benchmarks setting values
func BenchmarkGoRedisCache_Set(b *testing.B) {
	skipIfNoRedisForBenchmark(b)

	config := &GoRedisConfig{
		Addr:      getBenchmarkRedisAddr(),
		PoolSize:  10,
		KeyPrefix: "bench:set:",
	}

	cache, err := NewGoRedis(config)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	value := []byte("benchmark-value-data")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i)
			if err := cache.Set(ctx, key, value, time.Minute); err != nil {
				b.Errorf("Set failed: %v", err)
			}
			i++
		}
	})

	// Cleanup
	cache.DeletePattern(ctx, "*")
}

// BenchmarkGoRedisCache_Get benchmarks getting values
func BenchmarkGoRedisCache_Get(b *testing.B) {
	skipIfNoRedisForBenchmark(b)

	config := &GoRedisConfig{
		Addr:      getBenchmarkRedisAddr(),
		PoolSize:  10,
		KeyPrefix: "bench:get:",
	}

	cache, err := NewGoRedis(config)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	value := []byte("benchmark-value-data")

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(ctx, key, value, time.Hour)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%1000)
			if _, err := cache.Get(ctx, key); err != nil {
				b.Errorf("Get failed: %v", err)
			}
			i++
		}
	})

	// Cleanup
	cache.DeletePattern(ctx, "*")
}

// BenchmarkGoRedisCache_SetAndGet benchmarks combined operations
func BenchmarkGoRedisCache_SetAndGet(b *testing.B) {
	skipIfNoRedisForBenchmark(b)

	config := &GoRedisConfig{
		Addr:      getBenchmarkRedisAddr(),
		PoolSize:  10,
		KeyPrefix: "bench:combo:",
	}

	cache, err := NewGoRedis(config)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	value := []byte("benchmark-value-data")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i)
			if err := cache.Set(ctx, key, value, time.Minute); err != nil {
				b.Errorf("Set failed: %v", err)
			}
			if _, err := cache.Get(ctx, key); err != nil {
				b.Errorf("Get failed: %v", err)
			}
			i++
		}
	})

	// Cleanup
	cache.DeletePattern(ctx, "*")
}

// BenchmarkTypedModelCardCache_Set benchmarks setting model cards
func BenchmarkTypedModelCardCache_Set(b *testing.B) {
	skipIfNoRedisForBenchmark(b)

	redisConfig := &GoRedisConfig{
		Addr:      getBenchmarkRedisAddr(),
		PoolSize:  10,
		KeyPrefix: "bench:typed:set:",
	}

	baseCache, err := NewGoRedis(redisConfig)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer baseCache.Close()

	cache := NewTypedModelCardCache(baseCache, 5*time.Minute)
	ctx := context.Background()

	card := &ModelCard{
		ID:          "bench-card",
		WorkspaceID: "workspace-1",
		Name:        "Benchmark Model",
		Slug:        "benchmark-model",
		Version:     1,
		Status:      "active",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		card.ID = fmt.Sprintf("bench-card-%d", i)
		if err := cache.Set(ctx, card.ID, card, 0); err != nil {
			b.Errorf("Set failed: %v", err)
		}
	}

	// Cleanup
	baseCache.DeletePattern(ctx, "*")
}

// BenchmarkTypedModelCardCache_Get benchmarks getting model cards
func BenchmarkTypedModelCardCache_Get(b *testing.B) {
	skipIfNoRedisForBenchmark(b)

	redisConfig := &GoRedisConfig{
		Addr:      getBenchmarkRedisAddr(),
		PoolSize:  10,
		KeyPrefix: "bench:typed:get:",
	}

	baseCache, err := NewGoRedis(redisConfig)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer baseCache.Close()

	cache := NewTypedModelCardCache(baseCache, 5*time.Minute)
	ctx := context.Background()

	// Pre-populate with cards
	card := &ModelCard{
		ID:          "bench-card",
		WorkspaceID: "workspace-1",
		Name:        "Benchmark Model",
		Slug:        "benchmark-model",
		Version:     1,
		Status:      "active",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	for i := 0; i < 1000; i++ {
		card.ID = fmt.Sprintf("bench-card-%d", i)
		cache.Set(ctx, card.ID, card, time.Hour)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cardID := fmt.Sprintf("bench-card-%d", i%1000)
		if _, err := cache.Get(ctx, cardID); err != nil {
			b.Errorf("Get failed: %v", err)
		}
	}

	// Cleanup
	baseCache.DeletePattern(ctx, "*")
}

// BenchmarkTypedModelCardCache_SetWithA2AData benchmarks setting cards with complex A2A data
func BenchmarkTypedModelCardCache_SetWithA2AData(b *testing.B) {
	skipIfNoRedisForBenchmark(b)

	redisConfig := &GoRedisConfig{
		Addr:      getBenchmarkRedisAddr(),
		PoolSize:  10,
		KeyPrefix: "bench:typed:a2a:",
	}

	baseCache, err := NewGoRedis(redisConfig)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer baseCache.Close()

	cache := NewTypedModelCardCache(baseCache, 5*time.Minute)
	ctx := context.Background()

	// Create complex A2A-like card data
	a2aCardData := map[string]interface{}{
		"schemaVersion": "1.0",
		"name":          "Advanced Model",
		"description":   "A model with many capabilities",
		"capabilities": []map[string]interface{}{
			{"type": "text", "name": "Text", "enabled": true, "config": map[string]interface{}{"max_tokens": 4096}},
			{"type": "vision", "name": "Vision", "enabled": true, "config": map[string]interface{}{"image_size": "1024x1024"}},
			{"type": "code", "name": "Code", "enabled": true, "config": map[string]interface{}{"languages": []string{"python", "go", "javascript"}}},
			{"type": "embedding", "name": "Embeddings", "enabled": true},
		},
		"inputSchema": map[string]interface{}{
			"type": "json",
			"schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"messages": map[string]interface{}{"type": "array"},
					"model":    map[string]interface{}{"type": "string"},
				},
			},
			"contentTypes": []string{"application/json", "text/plain"},
		},
		"outputSchema": map[string]interface{}{
			"type": "json",
			"schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{"type": "string"},
					"role":    map[string]interface{}{"type": "string"},
				},
			},
			"contentTypes": []string{"application/json"},
		},
		"pricing": map[string]interface{}{
			"inputPricePerToken":  0.00003,
			"outputPricePerToken": 0.00006,
			"currency":            "USD",
		},
		"metadata": map[string]interface{}{
			"provider":      "openai",
			"family":        "gpt-4",
			"releaseDate":   "2024-01-01",
			"contextWindow": 8192,
			"trainingData":  "mixed",
		},
	}

	cardData, _ := json.Marshal(a2aCardData)

	card := &ModelCard{
		ID:          "a2a-bench-card",
		WorkspaceID: "workspace-1",
		Name:        "Advanced Model",
		Slug:        "advanced-model",
		Card:        cardData,
		Version:     1,
		Status:      "active",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		card.ID = fmt.Sprintf("a2a-bench-card-%d", i)
		if err := cache.Set(ctx, card.ID, card, 0); err != nil {
			b.Errorf("Set failed: %v", err)
		}
	}

	// Cleanup
	baseCache.DeletePattern(ctx, "*")
}

// BenchmarkTypedModelCardCache_SetProjectCards benchmarks setting project card lists
func BenchmarkTypedModelCardCache_SetProjectCards(b *testing.B) {
	skipIfNoRedisForBenchmark(b)

	redisConfig := &GoRedisConfig{
		Addr:      getBenchmarkRedisAddr(),
		PoolSize:  10,
		KeyPrefix: "bench:typed:project:",
	}

	baseCache, err := NewGoRedis(redisConfig)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer baseCache.Close()

	cache := NewTypedModelCardCache(baseCache, 5*time.Minute)
	ctx := context.Background()

	// Create a list of cards
	cards := make([]ModelCard, 10)
	for i := 0; i < 10; i++ {
		cards[i] = ModelCard{
			ID:          fmt.Sprintf("card-%d", i),
			WorkspaceID: "project-1",
			Name:        fmt.Sprintf("Model %d", i),
			Slug:        fmt.Sprintf("model-%d", i),
			Status:      "active",
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		projectID := fmt.Sprintf("project-%d", i%100)
		if err := cache.SetProjectCards(ctx, projectID, cards, 0); err != nil {
			b.Errorf("SetProjectCards failed: %v", err)
		}
	}

	// Cleanup
	baseCache.DeletePattern(ctx, "*")
}

// BenchmarkTypedModelCardCache_DeletePattern benchmarks pattern deletion
func BenchmarkTypedModelCardCache_DeletePattern(b *testing.B) {
	skipIfNoRedisForBenchmark(b)

	redisConfig := &GoRedisConfig{
		Addr:      getBenchmarkRedisAddr(),
		PoolSize:  10,
		KeyPrefix: "bench:typed:del:",
	}

	baseCache, err := NewGoRedis(redisConfig)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer baseCache.Close()

	ctx := context.Background()
	value := []byte("benchmark-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Setup: create keys to delete
		b.StopTimer()
		for j := 0; j < 100; j++ {
			key := fmt.Sprintf("pattern:batch%d:%d", i, j)
			baseCache.Set(ctx, key, value, time.Hour)
		}
		b.StartTimer()

		// Benchmark: delete the pattern
		pattern := fmt.Sprintf("pattern:batch%d:*", i)
		if err := baseCache.DeletePattern(ctx, pattern); err != nil {
			b.Errorf("DeletePattern failed: %v", err)
		}
	}
}

// BenchmarkGoRedisCache_PoolComparison compares different pool sizes
func BenchmarkGoRedisCache_PoolComparison(b *testing.B) {
	skipIfNoRedisForBenchmark(b)

	poolSizes := []int{5, 10, 20, 50}

	for _, poolSize := range poolSizes {
		b.Run(fmt.Sprintf("PoolSize%d", poolSize), func(b *testing.B) {
			config := &GoRedisConfig{
				Addr:      getBenchmarkRedisAddr(),
				PoolSize:  poolSize,
				KeyPrefix: fmt.Sprintf("bench:pool%d:", poolSize),
			}

			cache, err := NewGoRedis(config)
			if err != nil {
				b.Fatalf("Failed to create cache: %v", err)
			}
			defer cache.Close()

			ctx := context.Background()
			value := []byte("benchmark-value")

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := fmt.Sprintf("key-%d", i)
					if err := cache.Set(ctx, key, value, time.Minute); err != nil {
						b.Errorf("Set failed: %v", err)
					}
					i++
				}
			})

			// Cleanup
			cache.DeletePattern(ctx, "*")
		})
	}
}

// BenchmarkGoRedisCache_LatencyDistribution simulates realistic latency patterns
func BenchmarkGoRedisCache_LatencyDistribution(b *testing.B) {
	skipIfNoRedisForBenchmark(b)

	config := &GoRedisConfig{
		Addr:      getBenchmarkRedisAddr(),
		PoolSize:  10,
		KeyPrefix: "bench:latency:",
	}

	cache, err := NewGoRedis(config)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Different payload sizes
	smallPayload := []byte(`{"id":"1","name":"test"}`)
	mediumPayload := make([]byte, 1024)     // 1KB
	largePayload := make([]byte, 1024*1024) // 1MB

	for i := range mediumPayload {
		mediumPayload[i] = byte('a')
	}
	for i := range largePayload {
		largePayload[i] = byte('b')
	}

	b.Run("SmallPayload", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("small-%d", i)
			cache.Set(ctx, key, smallPayload, time.Minute)
			cache.Get(ctx, key)
		}
	})

	b.Run("MediumPayload", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("medium-%d", i)
			cache.Set(ctx, key, mediumPayload, time.Minute)
			cache.Get(ctx, key)
		}
	})

	b.Run("LargePayload", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("large-%d", i)
			cache.Set(ctx, key, largePayload, time.Minute)
			cache.Get(ctx, key)
		}
	})

	// Cleanup
	cache.DeletePattern(ctx, "*")
}
