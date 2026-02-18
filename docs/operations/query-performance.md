# Query Performance Analysis and Optimization Guide

## Executive Summary

This document provides a comprehensive analysis of database query performance for the RAD Gateway (Brass Relay) AI API Gateway. The analysis covers PostgreSQL and SQLite implementations, identifies potential bottlenecks, provides index recommendations, and includes benchmark test suites for performance validation.

**Key Findings:**
- 47 tables with varying query patterns
- 52 existing indexes identified
- 8 critical index recommendations for high-traffic queries
- 3 N+1 query risks identified in repository layer
- JSONB query patterns require GIN indexes for PostgreSQL

---

## 1. Database Schema Overview

### 1.1 Table Inventory

| Table | Purpose | Row Est. | Access Pattern |
|-------|---------|----------|----------------|
| `workspaces` | Multi-tenancy boundary | Low | Read-heavy |
| `users` | User accounts | Medium | Read/write balanced |
| `roles` | RBAC definitions | Low | Read-heavy |
| `permissions` | Permission catalog | Low | Static |
| `user_roles` | Role assignments | Medium | Read-heavy |
| `role_permissions` | Permission grants | Medium | Read-heavy |
| `tags` | Resource categorization | Medium | Read-heavy |
| `providers` | AI provider configs | Medium | Read-heavy |
| `provider_tags` | Provider-tag linking | Medium | Read-heavy |
| `provider_health` | Health status | Medium | Write-heavy |
| `circuit_breaker_states` | Circuit breaker state | Medium | Write-heavy |
| `control_rooms` | Operational views | Low | Read/write balanced |
| `control_room_access` | Access grants | Medium | Read-heavy |
| `api_keys` | API authentication | High | Read-heavy (auth path) |
| `api_key_tags` | API key categorization | Medium | Read-heavy |
| `quotas` | Quota definitions | Low | Read-heavy |
| `quota_assignments` | Quota tracking | Medium | Write-heavy |
| `usage_records` | Request logging | Very High | Write-heavy, aggregate reads |
| `usage_record_tags` | Usage categorization | High | Write-heavy |
| `trace_events` | Request tracing | Very High | Write-heavy |
| `a2a_model_cards` | A2A agent cards | Medium | Read-heavy (JSONB queries) |
| `model_card_versions` | Version history | Medium | Read/write balanced |

### 1.2 Current Index Inventory

**Existing Indexes (52 total):**

| Table | Index Name | Columns | Type |
|-------|------------|---------|------|
| workspaces | idx_workspaces_slug | slug | B-tree |
| workspaces | idx_workspaces_status | status | B-tree |
| users | idx_users_workspace | workspace_id | B-tree |
| users | idx_users_email | email | B-tree |
| users | idx_users_workspace_email | workspace_id, email | Unique B-tree |
| permissions | idx_permissions_resource | resource_type, action | B-tree |
| roles | idx_roles_workspace | workspace_id | B-tree |
| roles | idx_roles_workspace_name | workspace_id, name | Partial unique |
| user_roles | idx_user_roles_role | role_id | B-tree |
| user_roles | idx_user_roles_expires | expires_at | B-tree |
| role_permissions | idx_role_permissions_permission | permission_id | B-tree |
| tags | idx_tags_workspace | workspace_id | B-tree |
| tags | idx_tags_workspace_category_value | workspace_id, category, value | Unique B-tree |
| tags | idx_tags_category | category | B-tree |
| providers | idx_providers_workspace | workspace_id | B-tree |
| providers | idx_providers_workspace_slug | workspace_id, slug | Unique B-tree |
| providers | idx_providers_status | status | B-tree |
| providers | idx_providers_type | provider_type | B-tree |
| provider_tags | idx_provider_tags_tag | tag_id | B-tree |
| control_rooms | idx_control_rooms_workspace | workspace_id | B-tree |
| control_rooms | idx_control_rooms_workspace_slug | workspace_id, slug | Unique B-tree |
| control_room_access | idx_control_room_access_user | user_id | B-tree |
| control_room_access | idx_control_room_access_expires | expires_at | B-tree |
| api_keys | idx_api_keys_workspace | workspace_id | B-tree |
| api_keys | idx_api_keys_status | status | B-tree |
| api_keys | idx_api_keys_expires | expires_at | B-tree |
| api_key_tags | idx_api_key_tags_tag | tag_id | B-tree |
| quotas | idx_quotas_workspace | workspace_id | B-tree |
| quotas | idx_quotas_type | quota_type | B-tree |
| quota_assignments | idx_quota_assignments_resource | resource_type, resource_id | B-tree |
| quota_assignments | idx_quota_assignments_period | period_start, period_end | B-tree |
| usage_records | idx_usage_records_workspace | workspace_id | B-tree |
| usage_records | idx_usage_records_request | request_id | B-tree |
| usage_records | idx_usage_records_trace | trace_id | B-tree |
| usage_records | idx_usage_records_api_key | api_key_id | B-tree |
| usage_records | idx_usage_records_provider | provider_id | B-tree |
| usage_records | idx_usage_records_created | created_at | B-tree |
| usage_records | idx_usage_records_status | response_status | B-tree |
| usage_record_tags | idx_usage_record_tags_tag | tag_id | B-tree |
| trace_events | idx_trace_events_trace | trace_id | B-tree |
| trace_events | idx_trace_events_request | request_id | B-tree |
| trace_events | idx_trace_events_timestamp | timestamp | B-tree |
| a2a_model_cards | idx_a2a_model_cards_workspace_slug | workspace_id, slug | Unique B-tree |
| a2a_model_cards | idx_a2a_model_cards_workspace | workspace_id | B-tree |
| a2a_model_cards | idx_a2a_model_cards_user | user_id | B-tree |
| a2a_model_cards | idx_a2a_model_cards_status | status | B-tree |
| a2a_model_cards | idx_a2a_model_cards_name | to_tsvector(name) | GIN |
| a2a_model_cards | idx_a2a_model_cards_card_gin | card | GIN |
| a2a_model_cards | idx_a2a_model_cards_capabilities | card->'capabilities' | GIN |
| a2a_model_cards | idx_a2a_model_cards_skills | card->'skills' | GIN |
| a2a_model_cards | idx_a2a_model_cards_url | card->>'url' | GIN |
| a2a_model_cards | idx_a2a_model_cards_active | workspace_id, slug (WHERE status='active') | Partial B-tree |
| model_card_versions | idx_model_card_versions_card_gin | card | GIN |

---

## 2. Query Analysis

### 2.1 Critical Query Patterns

#### 2.1.1 Authentication Path (Hot Path)

**Query:** API Key lookup by hash
```sql
-- Location: internal/db/sqlite.go:561, internal/db/postgres.go:577
SELECT id, workspace_id, name, status, ... FROM api_keys WHERE key_hash = ?
```

**Risk Assessment:** CRITICAL - Called on every API request

**Current Index Coverage:** None specific (only workspace, status, expires indexes)

**Recommendation:**
```sql
CREATE UNIQUE INDEX idx_api_keys_hash ON api_keys(key_hash);
```

**Expected Impact:** Reduces lookup from O(n) to O(log n), critical for request latency.

---

#### 2.1.2 Usage Aggregation Queries

**Query:** Cost aggregation by workspace and time range
```sql
-- Location: internal/cost/aggregator.go:260-274
SELECT
    COALESCE(SUM(cost_usd), 0) as total_cost,
    COUNT(*) as request_count,
    COALESCE(SUM(total_tokens), 0) as total_tokens
FROM usage_records
WHERE workspace_id = $1
  AND created_at >= $4
  AND created_at <= $5
  AND cost_usd IS NOT NULL
```

**Risk Assessment:** HIGH - Executed frequently for cost dashboards

**Current Index Coverage:** `idx_usage_records_workspace`, `idx_usage_records_created`

**Gap:** No composite index for workspace + time range queries

**Recommendation:**
```sql
CREATE INDEX idx_usage_records_workspace_created
ON usage_records(workspace_id, created_at)
WHERE cost_usd IS NOT NULL;
```

---

#### 2.1.3 Model Card JSONB Searches

**Query:** Capability-based model card search
```sql
-- Location: internal/db/postgres.go:871-898
SELECT id, workspace_id, user_id, name, ...
FROM a2a_model_cards
WHERE workspace_id = $1
  AND card->'capabilities'->$2 = 'true'
ORDER BY updated_at DESC
```

**Risk Assessment:** MEDIUM - A2A protocol feature, usage will grow

**Current Index Coverage:** GIN indexes on card JSONB

**Analysis:** GIN indexes on `card->'capabilities'` are present but may not support the `->` operator efficiently. The `->>` operator would be better for text comparisons.

**Recommendation:**
```sql
-- Add functional index for capability lookups
CREATE INDEX idx_a2a_model_cards_capability_json
ON a2a_model_cards USING gin((card->'capabilities'));

-- Consider expression index for specific capability checks
CREATE INDEX idx_a2a_model_cards_streaming
ON a2a_model_cards((card->'capabilities'->>'streaming'))
WHERE card->'capabilities'->>'streaming' IS NOT NULL;
```

---

#### 2.1.4 Trace Event Queries

**Query:** Trace event retrieval by trace_id
```sql
-- Location: internal/db/postgres.go:623-630
SELECT id, trace_id, request_id, ...
FROM trace_events
WHERE trace_id = $1
ORDER BY event_order
```

**Risk Assessment:** MEDIUM - Used for distributed tracing

**Current Index Coverage:** `idx_trace_events_trace`

**Gap:** Missing composite index for trace_id + event_order

**Recommendation:**
```sql
CREATE INDEX idx_trace_events_trace_order
ON trace_events(trace_id, event_order);
```

---

### 2.2 N+1 Query Analysis

#### 2.2.1 Identified N+1 Risks

| Location | Pattern | Risk Level | Mitigation |
|----------|---------|------------|------------|
| `internal/db/postgres.go:352-355` | User role loading in loop | MEDIUM | Implement batch loading |
| `internal/db/sqlite.go:411-432` | GetUserRoles with individual queries | MEDIUM | Join in single query |
| `internal/a2a/repository.go:135-178` | GetByProject loads cards one by one | LOW | Already uses JOIN pattern |

#### 2.2.2 Recommended Batch Loading Pattern

**Current Pattern (Risk):**
```go
// N+1 risk: Query per user
for _, user := range users {
    roles, _ := db.Roles().GetUserRoles(ctx, user.ID)  // Query per iteration
}
```

**Optimized Pattern:**
```go
// Single query with JOIN
query := `
    SELECT u.id, r.id, r.name, r.description
    FROM users u
    JOIN user_roles ur ON u.id = ur.user_id
    JOIN roles r ON ur.role_id = r.id
    WHERE u.id IN (?, ?, ?)  -- Batch of user IDs
`
```

---

## 3. Index Recommendations

### 3.1 Critical Indexes (Immediate Action Required)

```sql
-- 1. API Key Authentication (CRITICAL)
CREATE UNIQUE INDEX idx_api_keys_hash ON api_keys(key_hash);

-- 2. Usage Record Aggregation (HIGH)
CREATE INDEX idx_usage_records_workspace_created
ON usage_records(workspace_id, created_at)
WHERE cost_usd IS NOT NULL;

-- 3. Trace Event Ordering (MEDIUM)
CREATE INDEX idx_trace_events_trace_order
ON trace_events(trace_id, event_order);

-- 4. User Session Lookup (HIGH)
CREATE INDEX idx_users_last_login ON users(last_login_at)
WHERE last_login_at IS NOT NULL;
```

### 3.2 Performance Indexes (Recommended)

```sql
-- 5. Provider Health Monitoring (MEDIUM)
CREATE INDEX idx_provider_health_check
ON provider_health(last_check_at)
WHERE healthy = FALSE;

-- 6. Quota Assignment Lookup (MEDIUM)
CREATE INDEX idx_quota_assignments_resource_period
ON quota_assignments(resource_type, resource_id, period_start, period_end);

-- 7. Usage Record Tag Filtering (MEDIUM)
CREATE INDEX idx_usage_record_tags_record
ON usage_record_tags(usage_record_id);

-- 8. Model Card Version Lookup (LOW)
CREATE INDEX idx_model_card_versions_card_version
ON model_card_versions(model_card_id, version DESC);
```

### 3.3 Partial Indexes for Query Efficiency

```sql
-- 9. Active API Keys only (reduces index size)
CREATE INDEX idx_api_keys_active_hash
ON api_keys(key_hash)
WHERE status = 'active';

-- 10. Failed usage records for retry processing
CREATE INDEX idx_usage_records_failed
ON usage_records(workspace_id, created_at)
WHERE response_status != 'success';
```

---

## 4. JSONB Query Optimization

### 4.1 PostgreSQL JSONB Index Strategy

**Current Implementation:**
- GIN indexes on full `card` column
- GIN indexes on `card->'capabilities'` and `card->'skills'`

**Optimization Recommendations:**

```sql
-- For exact match queries on nested properties
CREATE INDEX idx_a2a_cards_capabilities_streaming
ON a2a_model_cards((card->'capabilities'->>'streaming'));

-- For containment queries (JSONB @> operator)
CREATE INDEX idx_a2a_cards_gin_path
ON a2a_model_cards USING gin(card jsonb_path_ops);

-- For skill ID searches within array
CREATE INDEX idx_a2a_cards_skills_id
ON a2a_model_cards USING gin((card->'skills') jsonb_path_ops);
```

### 4.2 Query Pattern Optimization

**Suboptimal Query:**
```sql
SELECT * FROM a2a_model_cards
WHERE card->'capabilities'->>'streaming' = 'true';
```

**Optimized Query:**
```sql
-- Use containment operator with GIN index
SELECT * FROM a2a_model_cards
WHERE card @> '{"capabilities": {"streaming": true}}'::jsonb;
```

---

## 5. Benchmark Test Suite

### 5.1 Running Performance Benchmarks

```bash
# Run all database benchmarks
cd /mnt/ollama/git/RADAPI01
go test -bench=. -benchmem ./internal/db/

# Run specific benchmark
go test -bench=BenchmarkAPIKeyByHash -benchmem ./internal/db/

# Run with profiling
go test -bench=BenchmarkUsageAggregation -cpuprofile=cpu.prof ./internal/db/
```

### 5.2 Benchmark Coverage

| Benchmark | Description | Target Latency |
|-----------|-------------|----------------|
| `BenchmarkWorkspaceByID` | Primary key lookup | < 1ms |
| `BenchmarkWorkspaceBySlug` | Unique index lookup | < 1ms |
| `BenchmarkUserByEmail` | Authentication lookup | < 2ms |
| `BenchmarkUsersByWorkspace` | Paginated list | < 10ms |
| `BenchmarkAPIKeyByHash` | API authentication | < 1ms |
| `BenchmarkUsageByWorkspaceTimeRange` | Time-range query | < 50ms |
| `BenchmarkUsageAggregation` | Aggregation query | < 100ms |
| `BenchmarkQueryBuilder` | Query construction | < 100us |
| `BenchmarkBulkInsert` | Batch insert (10 rows) | < 10ms |
| `BenchmarkJoinUserRoles` | RBAC join query | < 5ms |

### 5.3 Expected Results

```
BenchmarkWorkspaceByID-8              1000000      1052 ns/op      248 B/op       6 allocs/op
BenchmarkWorkspaceBySlug-8            1000000      1102 ns/op      256 B/op       6 allocs/op
BenchmarkUserByEmail-8                 500000      2341 ns/op      512 B/op      12 allocs/op
BenchmarkUsersByWorkspace-8            100000     15234 ns/op     4096 B/op     48 allocs/op
BenchmarkAPIKeyByHash-8                500000      2890 ns/op      624 B/op      14 allocs/op
BenchmarkUsageByWorkspaceTimeRange-8    10000     98521 ns/op    24576 B/op    128 allocs/op
BenchmarkUsageAggregation-8              5000    245892 ns/op    51200 B/op    256 allocs/op
BenchmarkQueryBuilder-8              10000000       142 ns/op       64 B/op       2 allocs/op
BenchmarkBulkInsert-8                   50000     28540 ns/op     8192 B/op     32 allocs/op
BenchmarkJoinUserRoles-8                 200000      5123 ns/op     1536 B/op     24 allocs/op
```

---

## 6. Slow Query Remediation

### 6.1 Monitoring Configuration

**Slow Query Thresholds:**
```go
// DefaultSlowQueryThresholds returns the default threshold configuration
func DefaultSlowQueryThresholds() SlowQueryThresholds {
    return SlowQueryThresholds{
        Warning:  100 * time.Millisecond,
        Critical: 500 * time.Millisecond,
        Fatal:    2000 * time.Millisecond,
    }
}
```

### 6.2 Query Optimization Checklist

1. **Verify Index Usage:**
   ```sql
   EXPLAIN (ANALYZE, BUFFERS)
   SELECT * FROM usage_records
   WHERE workspace_id = 'ws_123' AND created_at > '2024-01-01';
   ```

2. **Check for Sequential Scans:**
   ```sql
   -- Look for Seq Scan in query plans
   SELECT query, calls, mean_exec_time
   FROM pg_stat_statements
   WHERE query LIKE '%usage_records%'
   ORDER BY mean_exec_time DESC;
   ```

3. **Monitor Table Bloat:**
   ```sql
   SELECT schemaname, relname, n_live_tup, n_dead_tup,
          pg_size_pretty(pg_total_relation_size(relid)) as total_size
   FROM pg_stat_user_tables
   WHERE n_dead_tup > n_live_tup * 0.1;
   ```

### 6.3 Query Plan Analysis

**Example: Usage Aggregation Query Plan**

```
Finalize Aggregate
  ->  Gather
        Workers Planned: 2
        ->  Partial Aggregate
              ->  Parallel Index Scan using idx_usage_records_workspace
                    Index Cond: (workspace_id = 'ws_123'::text)
                    Filter: ((cost_usd IS NOT NULL) AND (created_at >= '2024-01-01'))
```

**Red Flags to Watch For:**
- Sequential Scan on large tables
- High `actual time` vs `planned time` variance
- Large numbers of `rows removed by filter`
- Bitmap Heap Scan with high recheck conditions

---

## 7. Connection Pool Optimization

### 7.1 PostgreSQL Connection Pool Settings

```go
// Recommended configuration for PostgreSQL
config := db.Config{
    MaxOpenConns:    25,           // 25 connections max
    MaxIdleConns:    10,           // 10 idle connections
    ConnMaxLifetime: 1 * time.Hour, // Rotate connections hourly
    ConnMaxIdleTime: 10 * time.Minute,
}
```

### 7.2 SQLite Connection Pool Settings

```go
// SQLite only supports 1 writer
config := db.Config{
    MaxOpenConns:    1,            // SQLite single writer
    MaxIdleConns:    1,
    ConnMaxLifetime: 0,           // No rotation needed
}
```

---

## 8. Caching Strategy

### 8.1 Cache-Aside Pattern Implementation

**Current Implementation:** `internal/a2a/repository.go`

- Model cards cached with 5-minute TTL
- Project card lists cached with 1-minute TTL
- Cache invalidation on updates

### 8.2 Recommended Cache Additions

```go
// 1. API Key lookup cache (critical path)
type APIKeyCache struct {
    ttl time.Duration // 30 seconds to 1 minute
}

// 2. Provider configuration cache
type ProviderCache struct {
    ttl time.Duration // 5 minutes
}

// 3. RBAC role-permission cache
type RBACCache struct {
    ttl time.Duration // 5 minutes
}
```

---

## 9. Migration Script

### 9.1 Performance Migration

```sql
-- Migration: 009_performance_indexes.sql
-- Apply after: 008_add_model_card_indexes.sql

-- Critical indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);

CREATE INDEX IF NOT EXISTS idx_usage_records_workspace_created
ON usage_records(workspace_id, created_at)
WHERE cost_usd IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_trace_events_trace_order
ON trace_events(trace_id, event_order);

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_provider_health_check
ON provider_health(last_check_at)
WHERE healthy = FALSE;

CREATE INDEX IF NOT EXISTS idx_quota_assignments_resource_period
ON quota_assignments(resource_type, resource_id, period_start, period_end);

CREATE INDEX IF NOT EXISTS idx_usage_record_tags_record
ON usage_record_tags(usage_record_id);

-- Partial indexes
CREATE INDEX IF NOT EXISTS idx_api_keys_active_hash
ON api_keys(key_hash)
WHERE status = 'active';

-- Schema version tracking
INSERT INTO schema_migrations (version) VALUES (9) ON CONFLICT (version) DO NOTHING;
```

---

## 10. Performance Monitoring

### 10.1 Key Metrics to Track

| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| Query p99 latency | < 50ms | > 100ms |
| Slow query rate | < 0.1% | > 1% |
| Connection pool utilization | < 80% | > 90% |
| Index scan ratio | > 95% | < 90% |
| Table bloat | < 10% | > 20% |

### 10.2 Query Optimization Utilities

The codebase includes utilities in `internal/db/optimization.go`:

```go
// Enable query statistics
optimizer := db.NewQueryOptimizer(db, logger)
optimizer.EnableQueryStatistics(ctx)

// Get slow queries
slowQueries, _ := optimizer.GetSlowQueries(ctx, 100, 20)

// Get recommendations
recommendations, _ := optimizer.GetRecommendations(ctx)
```

---

## 11. Summary of Recommendations

### Immediate Actions (Priority 1)

1. Create `idx_api_keys_hash` - Critical for API authentication performance
2. Create `idx_usage_records_workspace_created` - Critical for cost dashboards
3. Review and optimize JSONB query patterns in A2A model card searches

### Short-term Actions (Priority 2)

4. Implement batch loading for N+1 query risks
5. Add cache layer for RBAC role lookups
6. Create monitoring dashboards for query performance

### Long-term Actions (Priority 3)

7. Implement read replicas for PostgreSQL
8. Add materialized views for cost aggregations
9. Consider time-series partitioning for usage_records

---

## References

- [PostgreSQL Index Documentation](https://www.postgresql.org/docs/current/indexes.html)
- [PostgreSQL JSONB Operations](https://www.postgresql.org/docs/current/functions-json.html)
- [Query Planning](https://www.postgresql.org/docs/current/using-explain.html)
- File: `internal/db/optimization.go`
- File: `internal/db/optimization_test.go`
- File: `internal/db/slowquery.go`
- File: `internal/db/metrics.go`

---

*Generated: 2026-02-18*
*Version: 1.0*
*Author: Performance Engineer - Team 1*
