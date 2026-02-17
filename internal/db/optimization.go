// Package db provides query optimization utilities for RAD Gateway.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// QueryOptimizer provides database query optimization utilities.
type QueryOptimizer struct {
	db     *sql.DB
	logger *slog.Logger
}

// SlowQuery represents a slow query from pg_stat_statements.
type SlowQuery struct {
	QueryID       int64
	Query         string
	Calls         int64
	TotalTime     float64
	MeanTime      float64
	Rows          int64
	SharedBlksHit int64
	SharedBlksRead int64
}

// IndexUsage represents index usage statistics.
type IndexUsage struct {
	SchemaName string
	TableName  string
	IndexName  string
	IdxScan    int64
	IdxTupRead int64
	IdxTupFetch int64
	TableSize  string
	IndexSize  string
}

// TableStats represents table statistics.
type TableStats struct {
	SchemaName   string
	TableName    string
	RowCount     int64
	TableSize    string
	IndexSize    string
	TotalSize    string
	LiveTuples   int64
	DeadTuples   int64
	LastVacuum   *time.Time
	LastAnalyze  *time.Time
}

// QueryPlan represents an EXPLAIN ANALYZE output.
type QueryPlan struct {
	PlanningTime float64
	ExecutionTime float64
	Rows         int64
	Loops        int64
	NodeType     string
	ActualRows   int64
	ActualTime   float64
}

// NewQueryOptimizer creates a new query optimizer instance.
func NewQueryOptimizer(db *sql.DB, logger *slog.Logger) *QueryOptimizer {
	if logger == nil {
		logger = slog.Default()
	}
	return &QueryOptimizer{
		db:     db,
		logger: logger,
	}
}

// EnableQueryStatistics enables pg_stat_statements extension (PostgreSQL only).
func (qo *QueryOptimizer) EnableQueryStatistics(ctx context.Context) error {
	// Check if running PostgreSQL
	var version string
	err := qo.db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		return fmt.Errorf("failed to check database version: %w", err)
	}

	if !strings.Contains(strings.ToLower(version), "postgresql") {
		qo.logger.Info("Query statistics only available for PostgreSQL")
		return nil
	}

	// Try to create extension
	_, err = qo.db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements")
	if err != nil {
		qo.logger.Warn("Could not enable pg_stat_statements, may require shared_preload_libraries config", "error", err)
		return nil
	}

	qo.logger.Info("pg_stat_statements extension enabled")
	return nil
}

// GetSlowQueries retrieves slow queries from pg_stat_statements.
func (qo *QueryOptimizer) GetSlowQueries(ctx context.Context, minTimeMs float64, limit int) ([]SlowQuery, error) {
	if limit <= 0 {
		limit = 20
	}

	query := `
		SELECT queryid, query, calls, total_exec_time, mean_exec_time,
		       rows, shared_blks_hit, shared_blks_read
		FROM pg_stat_statements
		WHERE mean_exec_time > $1
		ORDER BY mean_exec_time DESC
		LIMIT $2`

	rows, err := qo.db.QueryContext(ctx, query, minTimeMs, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get slow queries: %w", err)
	}
	defer rows.Close()

	var queries []SlowQuery
	for rows.Next() {
		var sq SlowQuery
		if err := rows.Scan(&sq.QueryID, &sq.Query, &sq.Calls, &sq.TotalTime,
			&sq.MeanTime, &sq.Rows, &sq.SharedBlksHit, &sq.SharedBlksRead); err != nil {
			return nil, fmt.Errorf("failed to scan slow query: %w", err)
		}
		queries = append(queries, sq)
	}

	return queries, rows.Err()
}

// GetIndexUsage retrieves index usage statistics.
func (qo *QueryOptimizer) GetIndexUsage(ctx context.Context, schema string) ([]IndexUsage, error) {
	if schema == "" {
		schema = "public"
	}

	query := `
		SELECT schemaname, relname, indexrelname,
		       idx_scan, idx_tup_read, idx_tup_fetch,
		       pg_size_pretty(pg_relation_size(relid)) as table_size,
		       pg_size_pretty(pg_relation_size(indexrelid)) as index_size
		FROM pg_stat_user_indexes
		WHERE schemaname = $1
		ORDER BY idx_scan ASC`

	rows, err := qo.db.QueryContext(ctx, query, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to get index usage: %w", err)
	}
	defer rows.Close()

	var indexes []IndexUsage
	for rows.Next() {
		var iu IndexUsage
		if err := rows.Scan(&iu.SchemaName, &iu.TableName, &iu.IndexName,
			&iu.IdxScan, &iu.IdxTupRead, &iu.IdxTupFetch, &iu.TableSize, &iu.IndexSize); err != nil {
			return nil, fmt.Errorf("failed to scan index usage: %w", err)
		}
		indexes = append(indexes, iu)
	}

	return indexes, rows.Err()
}

// GetTableStats retrieves table statistics.
func (qo *QueryOptimizer) GetTableStats(ctx context.Context, schema string) ([]TableStats, error) {
	if schema == "" {
		schema = "public"
	}

	query := `
		SELECT schemaname, relname,
		       n_live_tup as live_tuples,
		       n_dead_tup as dead_tuples,
		       pg_size_pretty(pg_total_relation_size(relid)) as total_size,
		       pg_size_pretty(pg_relation_size(relid)) as table_size,
		       pg_size_pretty(pg_indexes_size(relid)) as index_size,
		       last_vacuum, last_autovacuum, last_analyze, last_autoanalyze
		FROM pg_stat_user_tables
		WHERE schemaname = $1
		ORDER BY pg_total_relation_size(relid) DESC`

	rows, err := qo.db.QueryContext(ctx, query, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to get table stats: %w", err)
	}
	defer rows.Close()

	var tables []TableStats
	for rows.Next() {
		var ts TableStats
		var lastVacuum, lastAutoVacuum, lastAnalyze, lastAutoAnalyze *time.Time

		if err := rows.Scan(&ts.SchemaName, &ts.TableName, &ts.LiveTuples, &ts.DeadTuples,
			&ts.TotalSize, &ts.TableSize, &ts.IndexSize, &lastVacuum, &lastAutoVacuum,
			&lastAnalyze, &lastAutoAnalyze); err != nil {
			return nil, fmt.Errorf("failed to scan table stats: %w", err)
		}

		// Use the most recent of vacuum/analyze times
		if lastVacuum != nil {
			ts.LastVacuum = lastVacuum
		}
		if lastAutoVacuum != nil && (ts.LastVacuum == nil || lastAutoVacuum.After(*ts.LastVacuum)) {
			ts.LastVacuum = lastAutoVacuum
		}
		if lastAnalyze != nil {
			ts.LastAnalyze = lastAnalyze
		}
		if lastAutoAnalyze != nil && (ts.LastAnalyze == nil || lastAutoAnalyze.After(*ts.LastAnalyze)) {
			ts.LastAnalyze = lastAutoAnalyze
		}

		tables = append(tables, ts)
	}

	return tables, rows.Err()
}

// AnalyzeTable runs ANALYZE on a specific table.
func (qo *QueryOptimizer) AnalyzeTable(ctx context.Context, tableName string) error {
	_, err := qo.db.ExecContext(ctx, fmt.Sprintf("ANALYZE %s", tableName))
	if err != nil {
		return fmt.Errorf("failed to analyze table %s: %w", tableName, err)
	}
	qo.logger.Info("Table analyzed", "table", tableName)
	return nil
}

// VacuumTable runs VACUUM ANALYZE on a specific table.
func (qo *QueryOptimizer) VacuumTable(ctx context.Context, tableName string) error {
	// VACUUM cannot run inside a transaction block, so we use a raw connection
	_, err := qo.db.ExecContext(ctx, fmt.Sprintf("VACUUM ANALYZE %s", tableName))
	if err != nil {
		return fmt.Errorf("failed to vacuum table %s: %w", tableName, err)
	}
	qo.logger.Info("Table vacuumed and analyzed", "table", tableName)
	return nil
}

// GetQueryPlan retrieves the execution plan for a query.
func (qo *QueryOptimizer) GetQueryPlan(ctx context.Context, query string, args ...interface{}) (string, error) {
	// Use EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) for detailed output
	explainQuery := "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) " + query

	rows, err := qo.db.QueryContext(ctx, explainQuery, args...)
	if err != nil {
		return "", fmt.Errorf("failed to get query plan: %w", err)
	}
	defer rows.Close()

	var plan []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return "", fmt.Errorf("failed to scan plan line: %w", err)
		}
		plan = append(plan, line)
	}

	return strings.Join(plan, "\n"), rows.Err()
}

// OptimizationRecommendation represents a recommendation for query optimization.
type OptimizationRecommendation struct {
	Type        string
	Table       string
	Index       string
	Impact      string
	Description string
	Query       string
}

// GetRecommendations analyzes the database and provides optimization recommendations.
func (qo *QueryOptimizer) GetRecommendations(ctx context.Context) ([]OptimizationRecommendation, error) {
	var recommendations []OptimizationRecommendation

	// Check for unused indexes
	indexes, err := qo.GetIndexUsage(ctx, "public")
	if err != nil {
		qo.logger.Warn("Failed to get index usage", "error", err)
	} else {
		for _, idx := range indexes {
			if idx.IdxScan == 0 && idx.IdxTupRead == 0 {
				recommendations = append(recommendations, OptimizationRecommendation{
					Type:        "UNUSED_INDEX",
					Table:       idx.TableName,
					Index:       idx.IndexName,
					Impact:      "LOW",
					Description: fmt.Sprintf("Index %s on %s has never been used", idx.IndexName, idx.TableName),
				})
			}
		}
	}

	// Check for missing indexes on foreign keys (simplified check)
	fkCheckQuery := `
		SELECT tc.table_name, kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema = 'public'`

	rows, err := qo.db.QueryContext(ctx, fkCheckQuery)
	if err != nil {
		qo.logger.Warn("Failed to check foreign keys", "error", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var table, column string
			if err := rows.Scan(&table, &column); err != nil {
				continue
			}
			// Note: This is a simplified check. In production, you'd verify
			// if the FK column actually has an index.
		}
	}

	// Check table bloat
	tables, err := qo.GetTableStats(ctx, "public")
	if err != nil {
		qo.logger.Warn("Failed to get table stats", "error", err)
	} else {
		for _, t := range tables {
			if t.DeadTuples > t.LiveTuples/10 && t.LiveTuples > 1000 {
				recommendations = append(recommendations, OptimizationRecommendation{
					Type:        "TABLE_BLOAT",
					Table:       t.TableName,
					Impact:      "MEDIUM",
					Description: fmt.Sprintf("Table %s has %d dead tuples (%.1f%% bloat)",
						t.TableName, t.DeadTuples, float64(t.DeadTuples)*100/float64(t.LiveTuples)),
				})
			}
		}
	}

	return recommendations, nil
}

// QueryMonitor monitors query performance over time.
type QueryMonitor struct {
	optimizer      *QueryOptimizer
	slowThreshold  time.Duration
	checkInterval  time.Duration
	stopChan       chan struct{}
	handler        SlowQueryHandler
}

// SlowQueryHandler is called when a slow query is detected.
type SlowQueryHandler func(query SlowQuery)

// NewQueryMonitor creates a new query monitor.
func NewQueryMonitor(db *sql.DB, slowThreshold time.Duration, handler SlowQueryHandler) *QueryMonitor {
	return &QueryMonitor{
		optimizer:     NewQueryOptimizer(db, nil),
		slowThreshold: slowThreshold,
		checkInterval: 5 * time.Minute,
		stopChan:      make(chan struct{}),
		handler:       handler,
	}
}

// Start begins monitoring for slow queries.
func (qm *QueryMonitor) Start() {
	go qm.monitor()
}

// Stop stops the query monitor.
func (qm *QueryMonitor) Stop() {
	close(qm.stopChan)
}

func (qm *QueryMonitor) monitor() {
	ticker := time.NewTicker(qm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			queries, err := qm.optimizer.GetSlowQueries(ctx, float64(qm.slowThreshold.Milliseconds()), 10)
			cancel()

			if err != nil {
				continue
			}

			if qm.handler != nil {
				for _, q := range queries {
					qm.handler(q)
				}
			}

		case <-qm.stopChan:
			return
		}
	}
}

// ConnectionPoolStats represents connection pool statistics.
type ConnectionPoolStats struct {
	MaxOpenConnections int
	OpenConnections    int
	InUse              int
	Idle               int
	WaitCount          int64
	WaitDuration       time.Duration
	MaxIdleClosed      int64
	MaxIdleTimeClosed  int64
	MaxLifetimeClosed  int64
}

// GetConnectionPoolStats returns current connection pool statistics.
func GetConnectionPoolStats(db *sql.DB) ConnectionPoolStats {
	stats := db.Stats()
	return ConnectionPoolStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxIdleTimeClosed:  stats.MaxIdleTimeClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}
}

// OptimizeConnectionPool adjusts connection pool settings based on workload.
func OptimizeConnectionPool(db *sql.DB, maxOpen, maxIdle int, maxLifetime, maxIdleTime time.Duration) {
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(maxLifetime)
	db.SetConnMaxIdleTime(maxIdleTime)
}

// BulkInsert performs an optimized bulk insert operation.
func BulkInsert(ctx context.Context, db *sql.DB, table string, columns []string, values [][]interface{}) error {
	if len(values) == 0 {
		return nil
	}

	// Build query
	placeholders := make([]string, len(columns))
	for i := range columns {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	// For PostgreSQL, use COPY for large batches
	if len(values) > 100 {
		return bulkInsertCopy(ctx, db, table, columns, values)
	}

	// For smaller batches, use multi-value INSERT
	return bulkInsertMulti(ctx, db, table, columns, values)
}

func bulkInsertCopy(ctx context.Context, db *sql.DB, table string, columns []string, values [][]interface{}) error {
	// This is a placeholder - actual COPY implementation would require pgx driver
	// For now, fall back to multi-value insert
	return bulkInsertMulti(ctx, db, table, columns, values)
}

func bulkInsertMulti(ctx context.Context, db *sql.DB, table string, columns []string, values [][]interface{}) error {
	placeholders := make([]string, len(columns))
	for i := range columns {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	baseQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES ", table, strings.Join(columns, ", "))

	var allArgs []interface{}
	var valueStrings []string
	argIndex := 1

	for _, row := range values {
		rowPlaceholders := make([]string, len(row))
		for i := range row {
			rowPlaceholders[i] = fmt.Sprintf("$%d", argIndex)
			argIndex++
		}
		valueStrings = append(valueStrings, "("+strings.Join(rowPlaceholders, ", ")+")")
		allArgs = append(allArgs, row...)
	}

	query := baseQuery + strings.Join(valueStrings, ", ")
	_, err := db.ExecContext(ctx, query, allArgs...)
	return err
}

// QueryBuilder helps build optimized SQL queries.
type QueryBuilder struct {
	baseQuery   string
	whereClauses []string
	orderBy     string
	limit       int
	offset      int
	args        []interface{}
}

// NewQueryBuilder creates a new query builder.
func NewQueryBuilder(baseQuery string) *QueryBuilder {
	return &QueryBuilder{
		baseQuery: baseQuery,
		limit:     -1,
		offset:    -1,
	}
}

// Where adds a WHERE clause.
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	qb.whereClauses = append(qb.whereClauses, condition)
	qb.args = append(qb.args, args...)
	return qb
}

// OrderBy sets the ORDER BY clause.
func (qb *QueryBuilder) OrderBy(orderBy string) *QueryBuilder {
	qb.orderBy = orderBy
	return qb
}

// Limit sets the LIMIT.
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset sets the OFFSET.
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

// Build constructs the final query.
func (qb *QueryBuilder) Build() (string, []interface{}) {
	query := qb.baseQuery

	if len(qb.whereClauses) > 0 {
		query += " WHERE " + strings.Join(qb.whereClauses, " AND ")
	}

	if qb.orderBy != "" {
		query += " ORDER BY " + qb.orderBy
	}

	if qb.limit >= 0 {
		query += fmt.Sprintf(" LIMIT %d", qb.limit)
	}

	if qb.offset >= 0 {
		query += fmt.Sprintf(" OFFSET %d", qb.offset)
	}

	return query, qb.args
}

// QueryHint represents query optimization hints.
type QueryHint string

const (
	// HintSeqScan forces sequential scan
	HintSeqScan QueryHint = "SEQSCAN"
	// HintIndexScan forces index scan
	HintIndexScan QueryHint = "INDEXSCAN"
	// HintBitmapScan forces bitmap scan
	HintBitmapScan QueryHint = "BITMAPSCAN"
	// HintNestLoop forces nested loop join
	HintNestLoop QueryHint = "NESTLOOP"
	// HintMergeJoin forces merge join
	HintMergeJoin QueryHint = "MERGEJOIN"
	// HintHashJoin forces hash join
	HintHashJoin QueryHint = "HASHJOIN"
)

// ApplyHint applies a PostgreSQL query hint.
// Note: This requires the pg_hint_plan extension.
func ApplyHint(query string, hints ...QueryHint) string {
	if len(hints) == 0 {
		return query
	}

	hintStrs := make([]string, len(hints))
	for i, h := range hints {
		hintStrs[i] = string(h)
	}

	return fmt.Sprintf("/*+ %s */ %s", strings.Join(hintStrs, " "), query)
}
