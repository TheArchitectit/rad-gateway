# Hybrid Approach: PostgreSQL JSONB + Redis for A2A Model Cards

**Position**: The Pragmatist
**Date**: 2026-02-18
**Status**: Proposal for Team Review

## Executive Summary

Rather than choosing between pure PostgreSQL relational schema or pure MongoDB document store, I propose a **pragmatic hybrid approach** that leverages our existing PostgreSQL infrastructure with JSONB flexibility, augmented by Redis for caching. This approach balances operational simplicity, schema flexibility, and query performance without introducing new database operational overhead.

## Current State Analysis

### Existing PostgreSQL Schema
The RAD Gateway already uses JSONB for flexible configuration:

```sql
-- From migrations/001_create_workspaces.sql
settings JSONB DEFAULT '{}'

-- From migrations/005_create_providers.sql
config JSONB DEFAULT '{}'
```

This proves the team is already comfortable with PostgreSQL JSONB patterns.

### What Are A2A Model Cards?
A2A (Agent-to-Agent) Model Cards are metadata documents describing AI model capabilities, similar to:
- Model capabilities and limitations
- Input/output schemas
- Pricing information
- Version history
- Provider-specific extensions

These cards evolve rapidly as the A2A specification matures, making rigid schema migration painful.

## Proposed Hybrid Architecture

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Application Layer                              │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐ │
│  │   API Gateway   │  │  Model Card     │  │   Discovery Service     │ │
│  │   Handlers      │  │  Registry       │  │   (A2A Protocol)        │ │
│  └────────┬────────┘  └────────┬────────┘  └─────────────────────────┘ │
│           │                    │                                        │
└───────────┼────────────────────┼────────────────────────────────────────┘
            │                    │
            ▼                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        Data Access Layer                                 │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                    ModelCardRepository                              │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐   │  │
│  │  │   Cache     │  │   Query     │  │      Write Pipeline       │   │  │
│  │  │   Layer     │  │   Builder   │  │                           │   │  │
│  │  └──────┬──────┘  └──────┬──────┘  └─────────────────────────┘   │  │
│  └─────────┼────────────────┼─────────────────────────────────────────┘  │
└────────────┼────────────────┼────────────────────────────────────────────┘
             │                │
             ▼                ▼
┌────────────────────┐   ┌──────────────────────────────────────────────┐
│      Redis         │   │            PostgreSQL                         │
│  ┌──────────────┐ │   │  ┌────────────────────────────────────────┐  │
│  │  Model Card  │ │   │  │           model_cards                   │  │
│  │    Cache     │ │   │  │  ┌──────────────────────────────────┐  │  │
│  │              │ │   │  │  │ id UUID PRIMARY KEY               │  │  │
│  │  Key:        │ │   │  │  │ provider_id UUID REFERENCES       │  │  │
│  │  mc:{id}     │ │   │  │  │ external_id VARCHAR(255)          │  │  │
│  │  mc:provider│ │   │  │  │ ─────────────────────────────────  │  │  │
│  │     :{pid}   │ │   │  │  │ card_data JSONB NOT NULL          │  │  │
│  │              │ │   │  │  │ ─────────────────────────────────  │  │  │
│  │  TTL: 5min   │ │   │  │  │ capabilities JSONB                │  │  │
│  │  for hot data│ │   │  │  │ pricing JSONB                     │  │  │
│  └──────────────┘ │   │  │  │ schemas JSONB                     │  │  │
│                   │   │  │  │ metadata JSONB                   │  │  │
│  ┌──────────────┐ │   │  │  │ ─────────────────────────────────  │  │  │
│  │   Indexes    │ │   │  │  │ version INTEGER                   │  │  │
│  │  ├─ GIN on   │ │   │  │  │ status VARCHAR(32)                │  │  │
│  │  │  card_data│ │   │  │  │ created_at TIMESTAMPTZ            │  │  │
│  │  ├─ B-tree   │ │   │  │  │ updated_at TIMESTAMPTZ            │  │  │
│  │  │  on status│ │   │  │  │ indexed_at TIMESTAMPTZ            │  │  │
│  │  └─ Hash on  │ │   │  │  └──────────────────────────────────┘  │  │
│  │     external_id    │  │  ┌──────────────────────────────────┐  │  │
│  └──────────────┘ │   │  │  │  GIN INDEX on card_data        │  │  │
└───────────────────┘   │  │  │  BTREE INDEX on (provider_id, │  │  │
                        │  │  │              external_id)       │  │  │
                        │  │  │  BTREE INDEX on status        │  │  │
                        │  │  └──────────────────────────────────┘  │  │
                        │  └────────────────────────────────────────┘  │
                        └──────────────────────────────────────────────┘
```

### Data Flow

#### 1. Read Path (Hot Data)
```
Request → Check Redis Cache → Cache Hit → Return (sub-millisecond)
                    ↓
              Cache Miss → Query PostgreSQL → Populate Cache → Return
```

#### 2. Write Path (Model Card Updates)
```
Update Request → Write to PostgreSQL → Invalidate Cache → Async Reindex
                      ↓
               JSON Validation → Schema Version Check → Transaction Commit
```

#### 3. Discovery Path (A2A Protocol)
```
Discovery Request → Query PostgreSQL (indexed JSONB) → Filter → Return
                         ↓
                   Use GIN indexes for capability queries
```

## Schema Design

### PostgreSQL: model_cards Table

```sql
-- Migration: Create model_cards table
-- Purpose: Store A2A Model Cards with JSONB flexibility

CREATE TABLE IF NOT EXISTS model_cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    external_id VARCHAR(255) NOT NULL,  -- Provider's model ID (e.g., "gpt-4")

    -- Core JSONB columns for flexible A2A schema
    card_data JSONB NOT NULL,           -- Full A2A model card document
    capabilities JSONB,                 -- Extracted for fast filtering
    pricing JSONB,                      -- Pricing info for cost calculations
    schemas JSONB,                      -- Input/output schemas
    metadata JSONB,                     -- Custom metadata extensions

    -- Relational columns for querying
    version INTEGER NOT NULL DEFAULT 1, -- Schema version for migrations
    status VARCHAR(32) DEFAULT 'active'
        CHECK (status IN ('active', 'deprecated', 'archived')),

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    indexed_at TIMESTAMPTZ,             -- For full-text search indexing

    UNIQUE(provider_id, external_id)
);

-- Indexes for common query patterns
CREATE INDEX idx_model_cards_provider ON model_cards(provider_id);
CREATE INDEX idx_model_cards_external ON model_cards(external_id);
CREATE INDEX idx_model_cards_status ON model_cards(status);
CREATE INDEX idx_model_cards_updated ON model_cards(updated_at);

-- GIN indexes for JSONB queries (A2A discovery)
CREATE INDEX idx_model_cards_card_gin ON model_cards USING GIN(card_data);
CREATE INDEX idx_model_cards_capabilities_gin ON model_cards USING GIN(capabilities);

-- Composite index for provider + status lookups
CREATE INDEX idx_model_cards_provider_status ON model_cards(provider_id, status);

-- For capability-based queries (e.g., "supports vision")
CREATE INDEX idx_model_cards_capabilities_path
ON model_cards USING GIN(capabilities jsonb_path_ops);
```

### Redis: Cache Strategy

```
# Individual model card cache
Key: modelcard:{id}
Value: JSON serialized ModelCard
TTL: 300 seconds (5 minutes)

# Provider model list cache
Key: provider:{provider_id}:models
Value: JSON array of model IDs
TTL: 60 seconds (1 minute)

# Capability-based hot cache (popular queries)
Key: cap:{capability_hash}:models
Value: JSON array of matching model IDs
TTL: 120 seconds (2 minutes)

# Write-through invalidation pattern
On Write: DEL modelcard:{id}, DEL provider:{provider_id}:models
```

## Migration Plan

### Phase 1: Schema Preparation (Week 1)
1. Create `model_cards` table with JSONB columns
2. Add GIN indexes for A2A discovery queries
3. Implement base repository layer with PostgreSQL only
4. Add tests for JSONB operations

### Phase 2: Cache Layer Integration (Week 2)
1. Add Redis client to dependency injection
2. Implement cache-aside pattern in repository
3. Add cache warming for popular model cards
4. Configure cache TTL and eviction policies

### Phase 3: A2A Discovery API (Week 3)
1. Implement A2A protocol endpoints
2. Add JSONB query builder for capability filtering
3. Integrate cache with discovery queries
4. Load test discovery endpoints

### Phase 4: Production Hardening (Week 4)
1. Add cache metrics and monitoring
2. Implement cache stampede protection
3. Configure PostgreSQL JSONB statistics
4. Document operational runbooks

## Operational Complexity Analysis

### Compared to Pure MongoDB

| Aspect | Hybrid (PG+Redis) | Pure MongoDB | Advantage |
|--------|-------------------|--------------|-----------|
| **New Infrastructure** | None | MongoDB cluster | Hybrid |
| **Backup Strategy** | Existing pg_dump | New mongodump | Hybrid |
| **Monitoring** | Existing PG alerts | New monitoring | Hybrid |
| **Team Knowledge** | PG + Redis known | Mongo learning curve | Hybrid |
| **Connection Pool** | Existing pools | New pool management | Hybrid |
| **Failover** | Existing PG HA | Mongo replica setup | Hybrid |
| **Query Flexibility** | JSONB + relational | Native document | MongoDB |
| **Write Throughput** | PG WAL + Redis | Mongo Oplog | Comparable |

### Compared to Pure PostgreSQL Relational

| Aspect | Hybrid (PG JSONB) | Pure Relational | Advantage |
|--------|-------------------|-----------------|-----------|
| **Schema Evolution** | JSONB flexibility | Migrations required | Hybrid |
| **A2A Compatibility** | Native document | Complex joins | Hybrid |
| **Query Performance** | GIN indexes + cache | B-tree only | Hybrid |
| **Storage Efficiency** | JSONB compression | Normalized | Comparable |
| **ACID Compliance** | Full ACID | Full ACID | Tie |
| **Team Familiarity** | Uses existing patterns | Uses existing patterns | Tie |
| **Data Integrity** | JSON validation | Schema constraints | Relational |

### Operational Costs

```
Monthly Operational Overhead (Estimated)

Hybrid Approach:
├── PostgreSQL: $0 (existing)
├── Redis: $0 (existing caching layer)
├── Monitoring: $0 (existing dashboards)
└── DBA Time: ~2 hrs/week (familiar stack)

Pure MongoDB Approach:
├── MongoDB Atlas/Cluster: $200-500/mo (new)
├── Backup Infrastructure: $50/mo (new)
├── Monitoring Setup: $0 (new alerts needed)
└── DBA Time: ~8 hrs/week (learning + operations)

Pure Relational Approach:
├── PostgreSQL: $0 (existing)
├── Migration Complexity: High (frequent schema changes)
├── Query Complexity: High (complex joins for nested data)
└── DBA Time: ~4 hrs/week (migration management)
```

## Why PostgreSQL JSONB for Model Cards?

### 1. Schema Flexibility Without Sacrificing Structure
```go
// Example: A2A Model Card can evolve without migrations
type ModelCard struct {
    ID           uuid.UUID       `db:"id"`
    ProviderID   uuid.UUID       `db:"provider_id"`
    ExternalID   string          `db:"external_id"`
    CardData     json.RawMessage `db:"card_data"`  // Flexible A2A schema
    Capabilities json.RawMessage `db:"capabilities"`
    // ... relational fields for querying
}

// Query capabilities directly from JSONB
query := `
    SELECT * FROM model_cards
    WHERE capabilities @> '{"vision": true}'
    AND status = 'active'
`
```

### 2. ACID Compliance for Critical Operations
Model card updates need transactional integrity:
- Version bumps must be atomic
- Provider deletions must cascade properly
- Quota calculations depend on consistent pricing data

### 3. PostgreSQL Performance at Scale
```
PostgreSQL JSONB Benchmarks (approximate):
├── Single row lookup: < 1ms
├── GIN index query (10k rows): ~5ms
├── Full table scan (100k rows): ~200ms
└── With Redis cache: < 1ms for hot data
```

### 4. Existing Infrastructure Reuse
- Backup/restore procedures already defined
- Connection pooling already configured
- Monitoring already in place
- Team already trained

## Why Redis Cache Layer?

### 1. Read Performance
Model cards are read-heavy (discovery, validation) and relatively static.

### 2. A2A Protocol Optimization
A2A discovery requests can be cached by capability hash:
```
Client Query: "models with vision + code generation"
Cache Key: cap:a3f7b2:models
Hit Rate: ~85% for popular queries
```

### 3. Reduced Database Load
```
Without Cache: 10k requests → 10k DB queries
With Cache: 10k requests → 1.5k DB queries (85% hit rate)
```

## Risk Mitigation

### Risk: JSONB Query Performance
**Mitigation**:
- GIN indexes on all JSONB columns
- Query pattern analysis before deployment
- Fallback to relational columns for hot paths

### Risk: Cache Invalidation Complexity
**Mitigation**:
- Simple key-based invalidation (modelcard:{id})
- TTL-based auto-expiry for less critical data
- Circuit breaker for cache failures (degrade to DB)

### Risk: A2A Schema Changes
**Mitigation**:
- Version field in model_cards table
- JSON schema validation at application layer
- Migration scripts for major version bumps

### Risk: Operational Knowledge Gap
**Mitigation**:
- Team already uses JSONB (workspaces.settings, providers.config)
- Incremental adoption (PostgreSQL first, Redis second)
- Runbook documentation for common operations

## Implementation Recommendations

### Code Structure
```
internal/
├── modelcard/
│   ├── repository.go      # PostgreSQL + Redis implementation
│   ├── cache.go           # Redis cache layer
│   ├── query.go           # JSONB query builder
│   ├── models.go          # Domain models
│   └── validation.go      # A2A schema validation
├── database/
│   └── jsonb.go           # JSONB helper types
└── cache/
    └── redis.go           # Redis client wrapper
```

### Query Patterns to Support

1. **Lookup by ID** (cached)
   ```sql
   SELECT * FROM model_cards WHERE id = $1
   ```

2. **Provider Model List** (cached)
   ```sql
   SELECT * FROM model_cards
   WHERE provider_id = $1 AND status = 'active'
   ```

3. **Capability Discovery** (GIN indexed)
   ```sql
   SELECT * FROM model_cards
   WHERE capabilities @> $1 AND status = 'active'
   ```

4. **Full-Text Search** (if needed)
   ```sql
   SELECT * FROM model_cards
   WHERE to_tsvector(card_data::text) @@ plainto_tsquery($1)
   ```

## Conclusion

The hybrid PostgreSQL JSONB + Redis approach offers:

1. **Operational Simplicity**: Leverage existing infrastructure
2. **Schema Flexibility**: JSONB handles evolving A2A spec
3. **Query Performance**: GIN indexes + Redis cache
4. **Data Integrity**: Full ACID compliance
5. **Team Velocity**: No new database learning curve
6. **Cost Efficiency**: No new infrastructure costs

This is the pragmatic choice for a team that needs to move fast without accumulating technical debt or operational burden.

---

**Next Steps**:
1. Review with Team Charlie (Security) for data classification
2. Review with Team Echo (Operations) for capacity planning
3. Create proof-of-concept with 1000 model cards
4. Benchmark discovery queries against A2A spec requirements

**Author**: Pragmatist (Database Administrator)
**Team**: Database Debate - Team Hotel
**Date**: 2026-02-18
