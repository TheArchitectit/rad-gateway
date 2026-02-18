# RAD Gateway Database Health Report

**Report Date:** 2026-02-18
**Database:** PostgreSQL (with SQLite compatibility)
**Reviewer:** DBA Lead - Team 1: Database Maintenance Review

---

## Executive Summary

The RAD Gateway database schema is well-designed with proper indexing, comprehensive monitoring utilities, and robust connection pooling. This report identifies optimization opportunities and provides recommendations for maintaining high performance and reliability.

**Overall Health Grade: A-**

---

## 1. Schema Overview

### 1.1 Core Tables (13 Tables)

| Table | Purpose | Key Features |
|-------|---------|--------------|
| `workspaces` | Multi-tenancy boundary | UUID primary keys, status tracking |
| `users` | User accounts | Workspace-scoped, email indexing |
| `roles` | RBAC role definitions | System vs custom roles |
| `permissions` | Permission definitions | Resource-action pattern |
| `user_roles` | Many-to-many join | Expiration support |
| `role_permissions` | Permission assignments | Simple join table |
| `tags` | Resource categorization | Category:value hierarchy |
| `providers` | AI provider config | Health tracking, circuit breaker state |
| `control_rooms` | Operational views | Tag-based filtering |
| `api_keys` | Authentication keys | Rate limiting, model restrictions |
| `quotas` | Usage quotas | Period-based tracking |
| `usage_records` | Request logging | Comprehensive telemetry |
| `trace_events` | Request tracing | Event ordering support |
| `a2a_model_cards` | A2A agent definitions | JSONB flexible schema |
| `model_card_versions` | Version history | Audit trail |

### 1.2 Junction Tables (5 Tables)

- `user_roles`, `role_permissions`, `provider_tags`, `api_key_tags`, `usage_record_tags`

---

## 2. Index Health Analysis

### 2.1 Current Index Coverage

**Total Indexes:** 45+ across all tables

#### Well-Indexed Tables:

| Table | Indexes | Coverage |
|-------|---------|----------|
| `workspaces` | 2 | slug, status |
| `users` | 3 | workspace_id, email, composite (workspace, email) |
| `roles` | 2 | workspace_id, composite (workspace, name) |
| `tags` | 4 | workspace_id, composite (workspace, category, value), category |
| `providers` | 4 | workspace_id, composite (workspace, slug), status, type |
| `usage_records` | 7 | workspace, request_id, trace_id, api_key, provider, created_at, status |
| `a2a_model_cards` | 10 | workspace+slug, workspace, user, status, GIN indexes on JSONB |

### 2.2 Index Recommendations

#### HIGH Priority

1. **usage_records table** - Add composite index for time-series queries
   ```sql
   CREATE INDEX CONCURRENTLY idx_usage_records_workspace_created
   ON usage_records(workspace_id, created_at DESC);
   ```

2. **trace_events table** - Add composite index for trace lookups
   ```sql
   CREATE INDEX CONCURRENTLY idx_trace_events_trace_order
   ON trace_events(trace_id, event_order);
   ```

#### MEDIUM Priority

3. **providers table** - Add index for health monitoring queries
   ```sql
   CREATE INDEX CONCURRENTLY idx_providers_health_status
   ON providers(workspace_id, status) WHERE status = 'active';
   ```

4. **api_keys table** - Add partial index for active keys
   ```sql
   CREATE INDEX CONCURRENTLY idx_api_keys_active
   ON api_keys(workspace_id, key_hash) WHERE status = 'active' AND (expires_at IS NULL OR expires_at > NOW());
   ```

#### LOW Priority (Consider if needed)

5. `quota_assignments` - Add index for period queries with high cardinality
6. `control_room_access` - Add index for user access lookups with expiration

---

## 3. Connection Pool Configuration

### 3.1 Current Settings (from `internal/db/postgres.go`)

```go
MaxOpenConns:    10  // Maximum concurrent connections
MaxIdleConns:     3  // Idle connections to maintain
ConnMaxLifetime:  5m // Connection recycling interval
```

### 3.2 Assessment

| Metric | Current | Recommendation | Status |
|--------|---------|------------------|--------|
| Max Open | 10 | 25 (for production) | **LOW** |
| Max Idle | 3 | 5-10 | **LOW** |
| Max Lifetime | 5m | 30m | **OPTIMAL** |
| Max Idle Time | Not set | 15m | **MISSING** |

### 3.3 Recommended Production Configuration

```go
config := db.Config{
    MaxOpenConns:    25,              // Increase for concurrent API requests
    MaxIdleConns:    10,              // Keep warm connections ready
    ConnMaxLifetime: 30 * time.Minute, // Recycle before any firewall timeout
    ConnMaxIdleTime: 15 * time.Minute, // Close truly idle connections
}
```

### 3.4 Connection Pool Monitoring

The metrics collector in `internal/db/metrics.go` tracks:
- Open connections
- In-use connections
- Idle connections
- Wait count and duration
- Connection pool exhaustion events

**Alert Thresholds:**
- WARN: Wait count > 0
- CRITICAL: Open connections >= MaxOpenConns

---

## 4. Table Bloat Analysis

### 4.1 High-Volume Tables (Expected Growth)

| Table | Expected Rows/Day | Vacuum Strategy |
|-------|-------------------|-------------------|
| `usage_records` | 100K-1M+ | Autovacuum aggressive |
| `trace_events` | 500K-5M+ | Autovacuum aggressive |
| `provider_health` | Low | Standard autovacuum |
| `circuit_breaker_states` | Low | Standard autovacuum |

### 4.2 Autovacuum Recommendations

For high-volume tables, configure aggressive autovacuum:

```sql
ALTER TABLE usage_records SET (
    autovacuum_vacuum_scale_factor = 0.05,
    autovacuum_analyze_scale_factor = 0.025,
    autovacuum_vacuum_cost_limit = 1000
);

ALTER TABLE trace_events SET (
    autovacuum_vacuum_scale_factor = 0.05,
    autovacuum_analyze_scale_factor = 0.025,
    autovacuum_vacuum_cost_limit = 1000
);
```

---

## 5. Query Performance Analysis

### 5.1 Slow Query Detection

The codebase includes comprehensive slow query monitoring in `internal/db/slowquery.go`:

**Thresholds:**
- WARNING: > 100ms
- CRITICAL: > 500ms
- FATAL: > 2000ms

**Features:**
- Ring buffer (1000 queries)
- Deduplication (5-minute window)
- Statistics tracking (24-hour window)
- Alert handler support

### 5.2 Query Optimization Utilities

The `QueryOptimizer` in `internal/db/optimization.go` provides:

1. **pg_stat_statements integration** - Track query performance
2. **Index usage analysis** - Identify unused indexes
3. **Table statistics** - Monitor bloat and health
4. **Query plan analysis** - EXPLAIN ANALYZE support

### 5.3 Known Query Patterns

#### JSONB Queries on A2A Model Cards

The A2A model cards use JSONB with GIN indexes for flexible querying:

```sql
-- Capability search (uses idx_a2a_model_cards_capabilities)
SELECT * FROM a2a_model_cards
WHERE card->'capabilities'->>'streaming' = 'true';

-- Skill search (uses idx_a2a_model_cards_skills)
SELECT * FROM a2a_model_cards
WHERE EXISTS (
    SELECT 1 FROM jsonb_array_elements(card->'skills') AS skill
    WHERE skill->>'id' = 'skill-id'
);
```

**Status:** Well-optimized with GIN indexes.

---

## 6. Health Check SQL Queries

### 6.1 Connection Pool Status

```sql
-- Current connections
SELECT count(*) as connections, state
FROM pg_stat_activity
WHERE datname = current_database()
GROUP BY state;

-- Connection waiting
SELECT count(*) as waiting
FROM pg_stat_activity
WHERE wait_event_type = 'Client';
```

### 6.2 Table Health

```sql
-- Table bloat check
SELECT schemaname, relname, n_live_tup, n_dead_tup,
       round(n_dead_tup::numeric/nullif(n_live_tup,0)*100, 2) as dead_pct
FROM pg_stat_user_tables
WHERE n_dead_tup > 1000
ORDER BY n_dead_tup DESC;

-- Missing indexes check (seq_scan vs idx_scan)
SELECT schemaname, relname, seq_scan, seq_tup_read,
       idx_scan, n_live_tup
FROM pg_stat_user_tables
WHERE seq_scan > 0 AND idx_scan = 0 AND n_live_tup > 10000
ORDER BY seq_tup_read DESC;
```

### 6.3 Index Health

```sql
-- Unused indexes
SELECT schemaname, relname, indexrelname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0 AND indexrelname NOT LIKE 'pg_toast%'
ORDER BY pg_relation_size(indexrelid) DESC;

-- Index bloat estimate
SELECT schemaname, relname, indexrelname,
       pg_size_pretty(pg_relation_size(indexrelid)) as index_size
FROM pg_stat_user_indexes
ORDER BY pg_relation_size(indexrelid) DESC;
```

### 6.4 Query Performance

```sql
-- Slow queries (requires pg_stat_statements)
SELECT queryid, query, calls, mean_exec_time, rows
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 20;

-- Query cache hit ratio
SELECT sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read)) as cache_hit_ratio
FROM pg_statio_user_tables;
```

---

## 7. Backup and Recovery

### 7.1 Critical Tables (Priority 1)

- `workspaces` - Small, critical
- `users` - Small, critical
- `roles`, `permissions` - Small, critical
- `providers` - Medium, configuration
- `api_keys` - Medium, security

### 7.2 High-Volume Tables (Archive Strategy)

- `usage_records` - Partition by date, archive after 90 days
- `trace_events` - Partition by date, archive after 7 days
- `model_card_versions` - Archive soft-deleted versions

### 7.3 Recommended Backup Schedule

| Type | Frequency | Retention |
|------|-----------|-----------|
| Full | Daily | 7 days |
| Incremental | Hourly | 24 hours |
| Archive (usage_records) | Weekly | 1 year |

---

## 8. Optimization Recommendations Summary

### Immediate Actions (This Week)

1. [ ] Increase connection pool: MaxOpenConns=25, MaxIdleConns=10
2. [ ] Add ConnMaxIdleTime=15m to connection config
3. [ ] Create composite index on usage_records(workspace_id, created_at DESC)

### Short Term (Next 2 Weeks)

4. [ ] Configure aggressive autovacuum for usage_records and trace_events
5. [ ] Enable pg_stat_statements extension
6. [ ] Add partial indexes for active api_keys

### Medium Term (Next Month)

7. [ ] Implement usage_records partitioning by month
8. [ ] Set up automated index usage monitoring
9. [ ] Implement trace_events data retention policy

### Long Term (Next Quarter)

10. [ ] Evaluate read replica for reporting queries
11. [ ] Implement connection pooling with PgBouncer
12. [ ] Set up automated VACUUM ANALYZE scheduling

---

## 9. Monitoring Checklist

### Daily Checks

- [ ] Connection pool utilization (< 80%)
- [ ] Slow query count (< 10 per hour)
- [ ] Table bloat check (dead tuples < 10%)

### Weekly Checks

- [ ] Index usage analysis
- [ ] Query plan review for top 10 queries
- [ ] Table size growth tracking

### Monthly Checks

- [ ] Full index health review
- [ ] Autovacuum effectiveness
- [ ] Connection pool sizing review

---

## 10. Files Reviewed

| File | Purpose |
|------|---------|
| `internal/db/postgres.go` | PostgreSQL implementation, connection pooling |
| `internal/db/optimization.go` | Query optimization utilities |
| `internal/db/metrics.go` | Metrics collection and monitoring |
| `internal/db/slowquery.go` | Slow query detection and alerting |
| `internal/db/models.go` | Data models |
| `internal/db/migrations/001_initial_schema.sql` | Core schema |
| `internal/db/migrations/006_create_a2a_model_cards.sql` | A2A model cards |
| `internal/db/migrations/007_create_model_card_versions.sql` | Version tracking |
| `internal/db/migrations/008_add_model_card_indexes.sql` | JSONB indexes |

---

## Appendix: Quick Reference

### Database Configuration Template

```yaml
# Production database configuration
database:
  driver: postgres
  max_open_conns: 25
  max_idle_conns: 10
  conn_max_lifetime: 30m
  conn_max_idle_time: 15m

  # Monitoring
  slow_query_threshold: 100ms
  metrics_collection_interval: 30s

  # Pool alerts
  pool_exhaustion_threshold: 0.8  # 80% of max
  wait_duration_alert: 100ms
```

### Emergency Commands

```bash
# Check active connections
psql -c "SELECT count(*), state FROM pg_stat_activity GROUP BY state;"

# Cancel long-running queries
psql -c "SELECT pg_cancel_backend(pid) FROM pg_stat_activity WHERE state = 'active' AND query_start < NOW() - INTERVAL '5 minutes';"

# Force vacuum on bloated table
psql -c "VACUUM ANALYZE usage_records;"

# Check replication lag (if using replicas)
psql -c "SELECT EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp())) AS lag_seconds;"
```

---

**Report Generated:** 2026-02-18 by DBA Lead
**Next Review:** 2026-03-18
