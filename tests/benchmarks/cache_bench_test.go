// Package benchmarks provides performance benchmarks for cache operations.
// Sprint 7.2: Cache Hit/Miss Ratio Benchmarks
package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"radgateway/internal/cache"
)

// BenchmarkCacheOperations benchmarks basic cache operations
func BenchmarkCacheOperations(b *testing.B) {
	ctx := context.Background()
	c := cache.NewMemoryCache()
	defer c.Close()

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i)
			value := fmt.Sprintf("value-%d", i)
			err := c.Set(ctx, key, value, time.Hour)
			if err != nil {
				b.Fatalf("Set failed: %v", err)
			}
		}
	})

	b.Run("Get", func(b *testing.B) {
		// Pre-populate cache
		for i := 0; i < 1000; i++ {
			c.Set(ctx, fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i), time.Hour)
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i%1000)
			_, _ = c.Get(ctx, key)
		}
	})

	b.Run("GetWithHit", func(b *testing.B) {
		// Pre-populate cache
		for i := 0; i < 1000; i++ {
			c.Set(ctx, fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i), time.Hour)
		}
		b.ResetTimer()

		hits := 0
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i%1000)
			val, _ := c.Get(ctx, key)
			if val != "" {
				hits++
			}
		}
		b.ReportMetric(float64(hits)/float64(b.N)*100, "%hit")
	})

	b.Run("GetWithMiss", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("missing-key-%d", i)
			_, _ = c.Get(ctx, key)
		}
	})

	b.Run("Delete", func(b *testing.B) {
		// Pre-populate cache
		for i := 0; i < b.N; i++ {
			c.Set(ctx, fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i), time.Hour)
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i)
			err := c.Delete(ctx, key)
			if err != nil {
				b.Fatalf("Delete failed: %v", err)
			}
		}
	})
}

// BenchmarkCacheHitRatio benchmarks cache hit ratio under different scenarios
func BenchmarkCacheHitRatio(b *testing.B) {
	ctx := context.Background()
	c := cache.NewMemoryCache()
	defer c.Close()

	scenarios := []struct {
		name        string
		hitRate     float64
		uniqueKeys  int
	}{
		{"90%_Hit", 0.90, 100},
		{"75%_Hit", 0.75, 100},
		{"50%_Hit", 0.50, 100},
		{"25%_Hit", 0.25, 100},
		{"10%_Hit", 0.10, 100},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Populate cache with expected hits
			hitKeys := int(float64(scenario.uniqueKeys) * scenario.hitRate)
			for i := 0; i < hitKeys; i++ {
				c.Set(ctx, fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i), time.Hour)
			}

			b.ResetTimer()
			hits := 0
			misses := 0

			for i := 0; i < b.N; i++ {
				keyIdx := i % scenario.uniqueKeys
				key := fmt.Sprintf("key-%d", keyIdx)
				val, _ := c.Get(ctx, key)
				if val != "" {
					hits++
				} else {
					misses++
				}
			}

			actualHitRate := float64(hits) / float64(hits+misses) * 100
			b.ReportMetric(actualHitRate, "%hit")
			b.ReportMetric(float64(misses)/float64(hits+misses)*100, "%miss")
		})
	}
}

// BenchmarkCacheTTL benchmarks TTL behavior
func BenchmarkCacheTTL(b *testing.B) {
	ctx := context.Background()

	b.Run("ShortTTL", func(b *testing.B) {
		c := cache.NewMemoryCache()
		defer c.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i)
			c.Set(ctx, key, "value", time.Second)
		}
	})

	b.Run("MediumTTL", func(b *testing.B) {
		c := cache.NewMemoryCache()
		defer c.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i)
			c.Set(ctx, key, "value", time.Minute)
		}
	})

	b.Run("LongTTL", func(b *testing.B) {
		c := cache.NewMemoryCache()
		defer c.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i)
			c.Set(ctx, key, "value", time.Hour)
		}
	})
}

// BenchmarkCacheExpiration benchmarks expiration cleanup
func BenchmarkCacheExpiration(b *testing.B) {
	ctx := context.Background()
	c := cache.NewMemoryCache()
	defer c.Close()

	b.Run("ExpiredEntryCleanup", func(b *testing.B) {
		// Set entries with very short TTL
		for i := 0; i < 10000; i++ {
			c.Set(ctx, fmt.Sprintf("key-%d", i), "value", 1*time.Nanosecond)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			c.CleanupExpired(ctx)
		}
	})
}

// BenchmarkCacheConcurrency benchmarks concurrent access
func BenchmarkCacheConcurrency(b *testing.B) {
	ctx := context.Background()
	c := cache.NewMemoryCache()
	defer c.Close()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		c.Set(ctx, fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i), time.Hour)
	}

	concurrencyLevels := []int{1, 10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrent_%d", concurrency), func(b *testing.B) {
			b.SetParallelism(concurrency)
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := fmt.Sprintf("key-%d", i%1000)
					_, _ = c.Get(ctx, key)
					i++
				}
			})
		})
	}
}

// BenchmarkCacheMemoryUsage benchmarks memory efficiency
func BenchmarkCacheMemoryUsage(b *testing.B) {
	ctx := context.Background()
	c := cache.NewMemoryCache()
	defer c.Close()

	b.ReportAllocs()

	b.Run("SmallValues", func(b *testing.B) {
		value := "small-value"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			c.Set(ctx, fmt.Sprintf("key-%d", i), value, time.Hour)
		}
	})

	b.Run("MediumValues", func(b *testing.B) {
		value := make([]byte, 1024)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			c.Set(ctx, fmt.Sprintf("key-%d", i), string(value), time.Hour)
		}
	})

	b.Run("LargeValues", func(b *testing.B) {
		value := make([]byte, 1024*1024) // 1MB
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			c.Set(ctx, fmt.Sprintf("key-%d", i), string(value), time.Hour)
		}
	})
}

// BenchmarkCachePatternMatching benchmarks pattern-based operations
func BenchmarkCachePatternMatching(b *testing.B) {
	ctx := context.Background()
	c := cache.NewMemoryCache()
	defer c.Close()

	// Pre-populate with patterned keys
	for i := 0; i < 1000; i++ {
		c.Set(ctx, fmt.Sprintf("user:%d:profile", i), fmt.Sprintf("profile-%d", i), time.Hour)
		c.Set(ctx, fmt.Sprintf("user:%d:settings", i), fmt.Sprintf("settings-%d", i), time.Hour)
		c.Set(ctx, fmt.Sprintf("cache:item:%d", i), fmt.Sprintf("item-%d", i), time.Hour)
	}

	b.Run("PatternMatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = c.GetByPattern(ctx, "user:*:profile")
		}
	})

	b.Run("PatternDelete", func(b *testing.B) {
		// Reset cache each iteration
		for i := 0; i < 100; i++ {
			c.Set(ctx, fmt.Sprintf("temp:%d", i), "value", time.Hour)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = c.DeletePattern(ctx, "temp:*")
		}
	})
}

// BenchmarkCacheBatchOperations benchmarks batch operations
func BenchmarkCacheBatchOperations(b *testing.B) {
	ctx := context.Background()
	c := cache.NewMemoryCache()
	defer c.Close()

	b.Run("BatchSet", func(b *testing.B) {
		items := make(map[string]interface{})
		for i := 0; i < 100; i++ {
			items[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d", i)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := c.SetMulti(ctx, items, time.Hour)
			if err != nil {
				b.Fatalf("Batch set failed: %v", err)
			}
		}
	})

	b.Run("BatchGet", func(b *testing.B) {
		keys := make([]string, 100)
		for i := 0; i < 100; i++ {
			keys[i] = fmt.Sprintf("key-%d", i)
			c.Set(ctx, keys[i], fmt.Sprintf("value-%d", i), time.Hour)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = c.GetMulti(ctx, keys)
		}
	})
}

// BenchmarkCacheProviderComparison compares cache implementations
func BenchmarkCacheProviderComparison(b *testing.B) {
	ctx := context.Background()

	b.Run("MemoryCache", func(b *testing.B) {
		c := cache.NewMemoryCache()
		defer c.Close()

		benchmarkCacheProvider(b, c)
	})

	b.Run("RedisCache", func(b *testing.B) {
		c, err := cache.NewRedisCache("localhost:6379", "", 0)
		if err != nil {
			b.Skipf("Redis not available: %v", err)
		}
		defer c.Close()

		benchmarkCacheProvider(b, c)
	})
}

func benchmarkCacheProvider(b *testing.B, c cache.Cache) {
	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		c.Set(ctx, fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i), time.Hour)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		_, _ = c.Get(ctx, key)
	}
}
