# PostgreSQL Enhancements

## Overview

This document describes the PostgreSQL enhancements implemented for RAD Gateway, providing production-ready database capabilities including persistence, scaling, monitoring, and operational tooling.

## Implemented Features

### 1. Task Store Integration

A2A tasks now persist to PostgreSQL, enabling task recovery across restarts.

**Key Components:**
- `internal/a2a/task_store_pg.go` - PostgreSQL task store implementation
- `cmd/rad-gateway/main.go:208` - Automatic wiring when database is available

**Usage:**
```go
// Automatically configured in main.go
if database != nil {
    a2aTaskStore = a2a.NewPostgresTaskStore(sqlDB.DB())
}
```

**Features:**
- Full CRUD operations for tasks
- JSON serialization for messages and artifacts
- Transactional updates
- Automatic fallback to in-memory when database unavailable

---

### 2. Read Replica Router

Routes read queries to replicas and writes to primary, enabling horizontal scaling.

**Key Components:**
- `internal/db/replica.go` - Replica router implementation

**Usage:**
```go
router, err := db.NewReplicaRouter(
    "postgres://primary:5432/radgateway",
    []string{
        "postgres://replica1:5432/radgateway",
        "postgres://replica2:5432/radgateway",
    },
)

// Writes go to primary
writer := router.GetWriter()

// Reads distributed via round-robin
reader := router.GetReader()
```

**Features:**
- Round-robin replica selection
- Automatic fallback to primary if no replicas
- Health checks for all connections
- Separate connection pools for each node

---

### 3. Connection Pool Tuning

Production-optimized connection pool settings.

**Configuration:**
```go
config := db.Config{
    DSN:             "postgres://localhost/radgateway",
    MaxOpenConns:    25,              // Default: 25 connections
    MaxIdleConns:    10,              // 40% of max
    ConnMaxLifetime: 15 * time.Minute, // Connection lifetime
    ConnMaxIdleTime: 5 * time.Minute, // Idle timeout
}
```

**Settings:**
| Setting | Default | Description |
|---------|---------|-------------|
| MaxOpenConns | 25 | Maximum open connections |
| MaxIdleConns | 10 | Maximum idle connections |
| ConnMaxLifetime | 15m | Connection lifetime |
| ConnMaxIdleTime | 5m | Idle connection timeout |

**Monitoring:**
Pool configuration is logged on startup:
```
[db] pool configured: max_open=25 max_idle=25 lifetime=15m0s idle_time=5m0s
```

---

### 4. Database Metrics

Collects query performance metrics for monitoring and alerting.

**Key Components:**
- `internal/db/metrics.go` - Metrics collector

**Usage:**
```go
collector := db.NewMetricsCollector()

// Record query
collector.RecordQuery("SELECT", duration, err)

// Get stats
stats := collector.GetStats()
fmt.Printf("Queries: %d, Errors: %d, Avg: %.2fms",
    stats.QueryCount, stats.QueryErrors, stats.AvgLatencyMs)

// Health check
health := collector.HealthCheck()
if !health.Healthy {
    // Error rate > 5%
}
```

**Metrics Collected:**
- Total query count
- Error count and rate
- Average latency
- Query type breakdown (SELECT, INSERT, UPDATE, DELETE)
- Per-query-type statistics

**Health Thresholds:**
- Healthy: Error rate < 5%
- Unhealthy: Error rate >= 5%

---

### 5. Backup and Restore

Automated database backup with retention management.

**Key Components:**
- `internal/db/backup.go` - Backup manager

**Usage:**
```go
bm := db.NewBackupManager("postgres://localhost/radgateway")

// Create backup
path, err := bm.Backup(ctx, "") // Auto-generates filename
// Output: /var/backups/radgateway/radgateway-20260101-120000.sql

// Restore from backup
err = bm.Restore(ctx, "/path/to/backup.sql")

// List backups
backups, err := bm.ListBackups()
for _, b := range backups {
    fmt.Printf("%s (%d bytes)\n", b.Path, b.Size)
}

// Cleanup old backups (7-day retention)
err = bm.CleanupOldBackups()
```

**Features:**
- `pg_dump` integration for backups
- `psql` integration for restore
- Automatic timestamped filenames
- Configurable retention period (default: 7 days)
- Backup metadata (size, creation time)

---

### 6. Migration Rollback

Safe migration rollbacks with transaction support.

**Key Components:**
- `internal/db/migrator.go` - Migration system

**Usage:**
```go
migrator := db.NewMigrator(db, "postgres")

// Rollback last migration
err := migrator.Down(ctx)

// Rollback N migrations
err := migrator.DownBy(ctx, 3)

// Rollback to specific version
err := migrator.DownTo(ctx, 5)

// Check status
status, err := migrator.GetStatus(ctx)
fmt.Printf("Applied: %d, Pending: %d\n",
    len(status.Applied), len(status.Pending))
```

**Safety Features:**
- All rollbacks run in transactions
- Checksum validation prevents tampering
- Version tracking ensures idempotency
- Advisory locks prevent concurrent migrations

---

## Configuration

### Environment Variables

```bash
# Database DSN
export RAD_DB_DSN="postgres://user:pass@localhost:5432/radgateway?sslmode=require"

# Driver (postgres or sqlite)
export RAD_DB_DRIVER="postgres"

# Connection Pool (optional)
export RAD_DB_MAX_OPEN_CONNS="25"
export RAD_DB_MAX_IDLE_CONNS="10"
export RAD_DB_CONN_MAX_LIFETIME="15m"

# Read Replicas (comma-separated)
export RAD_DB_REPLICAS="postgres://replica1:5432/radgateway,postgres://replica2:5432/radgateway"
```

### Code Configuration

```go
// Database configuration
config := db.Config{
    DSN:             os.Getenv("RAD_DB_DSN"),
    MaxOpenConns:    25,
    MaxIdleConns:    10,
    ConnMaxLifetime: 15 * time.Minute,
    ConnMaxIdleTime: 5 * time.Minute,
}

// Create database
database, err := db.NewPostgres(config)
```

---

## Testing

### Run Database Tests

```bash
# All database tests
go test ./internal/db/... -v

# Specific tests
go test ./internal/db/... -v -run TestReplica
go test ./internal/db/... -v -run TestMetrics
go test ./internal/db/... -v -run TestBackup

# Integration tests (requires PostgreSQL)
export TEST_PRIMARY_DSN="postgres://localhost/radgateway_test"
export TEST_REPLICA_DSN="postgres://replica:5432/radgateway_test"
go test ./internal/db/... -v -run TestReplica
```

---

## Operations

### Creating a Backup

```bash
# Programmatic
backupPath, err := bm.Backup(ctx, "")

# Or using pg_dump directly
pg_dump -h localhost -U radgateway -f backup.sql radgateway
```

### Restoring from Backup

```bash
# Programmatic
err := bm.Restore(ctx, "/path/to/backup.sql")

# Or using psql directly
psql -h localhost -U radgateway radgateway < backup.sql
```

### Rolling Back Migrations

```bash
# Rollback last migration
go run cmd/migrate/main.go down

# Rollback 3 migrations
go run cmd/migrate/main.go down 3

# Rollback to version 5
go run cmd/migrate/main.go down-to 5
```

---

## Monitoring

### Health Checks

```go
// Database health
err := database.Ping(ctx)

// Replica health (if using replicas)
results := router.Health(ctx)
for node, err := range results {
    if err != nil {
        log.Printf("%s unhealthy: %v", node, err)
    }
}

// Metrics health
health := collector.HealthCheck()
if !health.Healthy {
    log.Printf("Unhealthy: error rate %.2f%%", health.ErrorRate*100)
}
```

### Metrics Export

```go
stats := collector.GetStats()
fmt.Printf(`
Queries:      %d
Errors:       %d
Error Rate:   %.2f%%
Avg Latency:  %.2fms
`, stats.QueryCount, stats.QueryErrors,
   float64(stats.QueryErrors)/float64(stats.QueryCount)*100,
   stats.AvgLatencyMs)

// Per-query-type stats
for queryType, typeStats := range stats.QueryTypes {
    fmt.Printf("%s: %d queries, avg %.2fms\n",
        queryType, typeStats.Count,
        float64(typeStats.Latency)/float64(typeStats.Count))
}
```

---

## Files Added/Modified

**New Files:**
- `internal/db/replica.go` - Read replica router
- `internal/db/replica_test.go` - Replica tests
- `internal/db/metrics.go` - Metrics collector
- `internal/db/metrics_test.go` - Metrics tests
- `internal/db/backup.go` - Backup manager
- `internal/db/backup_test.go` - Backup tests
- `internal/db/pool_test.go` - Pool configuration tests

**Modified:**
- `internal/db/postgres.go` - Connection pool tuning
- `internal/db/slowquery.go` - Fixed extractTableName

---

## Next Steps

- [ ] Wire metrics into API handlers for request tracking
- [ ] Create backup CLI command (`cmd/backup`)
- [ ] Add database health endpoint (`/health/db`)
- [ ] Export metrics to Prometheus format
- [ ] Add connection pool visualization in admin UI

---

**Last Updated:** 2026-02-28
**Version:** Phase 4 PostgreSQL Enhancements
