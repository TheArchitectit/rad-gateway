# Redis Caching Enhancements

## Overview

This document describes the Redis caching enhancements for RAD Gateway, providing distributed caching for API keys, agent cards, and rate limiting across multiple gateway instances.

## Implemented Features

### 1. API Key Cache

Caches validated API keys to reduce database lookups during authentication.

**Key Components:**
- `internal/cache/api_key_cache.go` - API key cache implementation

**Usage:**
```go
// Create cache with Redis backend
redisCache, _ := cache.NewRedis(cache.Config{
    Address:    "localhost:6379",
    Password:   "",
    Database:   0,
    DefaultTTL: 5 * time.Minute,
    KeyPrefix:  "rad:",
})

apiKeyCache := cache.NewTypedAPIKeyCache(redisCache, 5*time.Minute)

// Cache API key info
info := &cache.APIKeyInfo{
    Name:      "my-api-key",
    KeyHash:   hash,
    ProjectID: "proj-123",
    Role:      "admin",
    Valid:     true,
}
err := apiKeyCache.Set(ctx, hash, info, 0)

// Retrieve from cache
cached, err := apiKeyCache.Get(ctx, hash)
if cached != nil {
    // Use cached info
}
```

**Cache Key Format:**
- `api_key:{hash}` - Single API key info

**TTL:** 5 minutes (configurable)

---

### 2. Agent Card Cache

Caches A2A agent cards to reduce database lookups for agent discovery.

**Key Components:**
- `internal/cache/agent_card_cache.go` - Agent card cache implementation

**Usage:**
```go
// Create cache
agentCardCache := cache.NewTypedAgentCardCache(redisCache, 10*time.Minute)

// Cache agent card
card := &a2a.AgentCard{
    ID:   "agent-123",
    Name: "My Agent",
    // ...
}
err := agentCardCache.Set(ctx, card.ID, card, 0)

// Retrieve by ID
cached, err := agentCardCache.Get(ctx, "agent-123")

// Cache by skill
err = agentCardCache.SetBySkill(ctx, "skill-123", []a2a.AgentCard{card}, 0)
cards, err := agentCardCache.GetBySkill(ctx, "skill-123")
```

**Cache Key Formats:**
- `agent_card:{id}` - Single agent card
- `agent_cards:skill:{skill_id}` - Agents by skill
- `agent_cards:name:{name}` - Agents by name

**TTL:** 10 minutes (longer than model cards as agents change less frequently)

---

### 3. Distributed Rate Limiting

Uses Redis for distributed rate limiting across multiple gateway instances.

**Key Components:**
- `internal/cache/ratelimit_redis.go` - Redis-backed rate limiter

**Usage:**
```go
// Create rate limiter
limiter, err := cache.NewRedisRateLimiter("localhost:6379", "", 0)
if err != nil {
    log.Fatal(err)
}
defer limiter.Close()

// Check rate limit
allowed, err := limiter.CheckRateLimit(ctx, "user:123", 100, time.Minute)
if !allowed {
    http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
    return
}

// Get status
status, err := limiter.GetRateLimitStatus(ctx, "user:123", time.Minute)
fmt.Printf("Rate: %d/%d, Resets at: %v\n", status.Current, status.Limit, status.ResetAt)

// Reset if needed
err = limiter.ResetRateLimit(ctx, "user:123")
```

**Algorithm:** Sliding window with Redis sorted sets

**Key Format:**
- `rad:ratelimit:{key}` - Rate limit bucket

---

## Configuration

### Environment Variables

```bash
# Redis connection (required for caching)
export RAD_REDIS_ADDR="localhost:6379"
export RAD_REDIS_PASSWORD=""
export RAD_REDIS_DB="0"

# Cache TTLs (optional)
export RAD_API_KEY_CACHE_TTL="5m"
export RAD_AGENT_CARD_CACHE_TTL="10m"
export RAD_MODEL_CARD_CACHE_TTL="5m"
```

### Code Configuration

```go
// Redis cache configuration
redisConfig := cache.Config{
    Address:    "localhost:6379",
    Password:   "",
    Database:   0,
    DefaultTTL: 5 * time.Minute,
    KeyPrefix:  "rad:",
}

redisCache, err := cache.NewRedis(redisConfig)
if err != nil {
    log.Fatal(err)
}
defer redisCache.Close()

// Create typed caches
apiKeyCache := cache.NewTypedAPIKeyCache(redisCache, 5*time.Minute)
agentCardCache := cache.NewTypedAgentCardCache(redisCache, 10*time.Minute)
modelCardCache := cache.NewTypedModelCardCache(redisCache, 5*time.Minute)
```

---

## Cache Invalidation

### API Key Cache

```go
// Delete specific key
apiKeyCache.Delete(ctx, keyHash)

// Invalidate all keys for a project
apiKeyCache.InvalidateByProject(ctx, "proj-123")

// Invalidate by pattern
apiKeyCache.InvalidatePattern(ctx, "api_key:*")
```

### Agent Card Cache

```go
// Delete specific card
agentCardCache.Delete(ctx, cardID)

// Invalidate card and all related entries
agentCardCache.InvalidateCard(ctx, cardID)

// Invalidate by pattern
agentCardCache.InvalidatePattern(ctx, "agent_cards:*")
```

---

## Testing

### Run Cache Tests

```bash
# All cache tests
go test ./internal/cache/... -v

# Specific cache tests
go test ./internal/cache/... -v -run TestTypedAPIKeyCache
go test ./internal/cache/... -v -run TestTypedAgentCardCache

# Benchmark tests
go test ./internal/cache/... -v -bench=.
```

### Manual Verification

```bash
# Start Redis
redis-server

# Run application
go run ./cmd/rad-gateway

# Make requests and observe cache hits
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -d '{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}'
```

---

## Migration from In-Memory to Redis

### Before (In-Memory)

```go
// No persistence, per-instance only
memoryCache := cache.NewMemoryCache()
apiKeyCache := cache.NewTypedAPIKeyCache(memoryCache, 5*time.Minute)
```

### After (Redis)

```go
// Distributed, shared across instances
redisCache, _ := cache.NewRedis(cache.Config{
    Address: "localhost:6379",
})
apiKeyCache := cache.NewTypedAPIKeyCache(redisCache, 5*time.Minute)
```

---

## Monitoring

### Cache Metrics

```go
// Get cache stats
stats := redisCache.Stats()
fmt.Printf("Hits: %d, Misses: %d, HitRate: %.2f%%\n",
    stats.Hits, stats.Misses, stats.HitRate*100)
```

### Redis CLI Monitoring

```bash
# Monitor Redis commands
redis-cli MONITOR

# Check cache keys
redis-cli KEYS "rad:*"

# Get TTL for a key
redis-cli TTL "rad:api_key:hash123"

# Check memory usage
redis-cli INFO memory
```

---

## Files Added/Modified

**New Files:**
- `internal/cache/api_key_cache.go` - API key cache
- `internal/cache/api_key_cache_test.go` - API key cache tests
- `internal/cache/agent_card_cache.go` - Agent card cache
- `internal/cache/ratelimit_redis.go` - Redis rate limiter
- `docs/redis-caching.md` - This documentation

**Existing:**
- `internal/cache/redis.go` - Redis cache backend
- `internal/cache/model_card_cache_typed.go` - Model card cache

---

## Next Steps

- [ ] Wire API key cache into auth middleware
- [ ] Wire agent card cache into A2A handlers
- [ ] Add cache warming on startup
- [ ] Implement cache statistics endpoint
- [ ] Add Redis Sentinel/Cluster support

---

**Last Updated:** 2026-02-28
**Version:** Phase 4 Redis Caching
