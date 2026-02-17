// Package db provides database interfaces and implementations for RAD Gateway.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// QueryMetrics tracks performance metrics for database queries.
type QueryMetrics struct {
	// Query identifier (normalized query string or operation name)
	Query string
	// Query type: SELECT, INSERT, UPDATE, DELETE, etc.
	QueryType string
	// Table being accessed
	Table string
	// Total number of executions
	Count uint64
	// Total execution time across all queries
	TotalTime time.Duration
	// Minimum execution time
	MinTime time.Duration
	// Maximum execution time
	MaxTime time.Duration
	// Number of errors
	ErrorCount uint64
	// Last executed timestamp (stored as Unix nanoseconds for atomic operations)
	lastExecutedUnix int64
}

// LastExecuted returns the timestamp of last execution
func (qm *QueryMetrics) LastExecuted() time.Time {
	unix := atomic.LoadInt64(&qm.lastExecutedUnix)
	if unix == 0 {
		return time.Time{}
	}
	return time.Unix(0, unix)
}

// AverageTime returns the average execution time for this query.
func (qm *QueryMetrics) AverageTime() time.Duration {
	count := atomic.LoadUint64(&qm.Count)
	if count == 0 {
		return 0
	}
	total := time.Duration(atomic.LoadInt64((*int64)(&qm.TotalTime)))
	return total / time.Duration(count)
}

// ConnectionPoolMetrics tracks database connection pool statistics.
type ConnectionPoolMetrics struct {
	// Maximum open connections allowed
	MaxOpenConnections int
	// Currently open connections
	OpenConnections int
	// Connections in use
	InUse int
	// Idle connections available
	Idle int
	// Total number of waits for a connection
	WaitCount int64
	// Total time waited for connections
	WaitDuration time.Duration
	// Maximum time waited for a connection
	MaxWaitDuration time.Duration
	// Timestamp when metrics were collected
	CollectedAt time.Time
}

// DatabaseHealthMetrics tracks overall database health indicators.
type DatabaseHealthMetrics struct {
	// Connection status
	Connected bool
	// Last successful ping
	LastPingSuccess time.Time
	// Last ping latency
	LastPingLatency time.Duration
	// Consecutive ping failures
	ConsecutiveFailures int
	// Total queries executed since startup
	TotalQueries uint64
	// Total errors since startup
	TotalErrors uint64
	// Slow queries (> threshold) since startup
	SlowQueries uint64
	// Uptime (time since first connection)
	Uptime time.Duration
	// Timestamp when metrics were collected
	CollectedAt time.Time
}

// MetricsCollector defines the interface for database metrics collection.
type MetricsCollector interface {
	// RecordQuery records metrics for a query execution
	RecordQuery(query, queryType, table string, duration time.Duration, err error)
	// GetQueryMetrics returns metrics for a specific query
	GetQueryMetrics(query string) (*QueryMetrics, bool)
	// GetAllQueryMetrics returns all collected query metrics
	GetAllQueryMetrics() []*QueryMetrics
	// GetTopQueriesByCount returns the most frequently executed queries
	GetTopQueriesByCount(limit int) []*QueryMetrics
	// GetTopQueriesByTime returns the slowest queries by average time
	GetTopQueriesByTime(limit int) []*QueryMetrics
	// GetConnectionPoolMetrics returns current connection pool statistics
	GetConnectionPoolMetrics(db *sql.DB) ConnectionPoolMetrics
	// GetHealthMetrics returns current database health metrics
	GetHealthMetrics() DatabaseHealthMetrics
	// RecordPing records a ping result
	RecordPing(success bool, latency time.Duration)
	// Reset clears all collected metrics
	Reset()
	// StartCollection begins periodic metric collection
	StartCollection(ctx context.Context, db *sql.DB, interval time.Duration)
}

// DefaultMetricsCollector implements MetricsCollector with in-memory storage.
type DefaultMetricsCollector struct {
	mu sync.RWMutex

	// queryMetrics maps normalized query strings to their metrics
	queryMetrics map[string]*QueryMetrics

	// health tracking
	healthMu             sync.RWMutex
	totalQueries         uint64
	totalErrors          uint64
	slowQueries          uint64
	lastPingSuccess      time.Time
	lastPingLatency      time.Duration
	consecutiveFailures  int
	connected            bool
	startupTime          time.Time

	// threshold for slow query counting
	slowQueryThreshold time.Duration
}

// MetricsCollectorOption configures the DefaultMetricsCollector.
type MetricsCollectorOption func(*DefaultMetricsCollector)

// WithSlowQueryThreshold sets the threshold for slow query detection.
func WithSlowQueryThreshold(threshold time.Duration) MetricsCollectorOption {
	return func(c *DefaultMetricsCollector) {
		c.slowQueryThreshold = threshold
	}
}

// NewMetricsCollector creates a new DefaultMetricsCollector.
func NewMetricsCollector(opts ...MetricsCollectorOption) *DefaultMetricsCollector {
	collector := &DefaultMetricsCollector{
		queryMetrics:       make(map[string]*QueryMetrics),
		slowQueryThreshold: 100 * time.Millisecond,
		startupTime:        time.Now(),
	}

	for _, opt := range opts {
		opt(collector)
	}

	return collector
}

// RecordQuery records metrics for a query execution.
func (c *DefaultMetricsCollector) RecordQuery(query, queryType, table string, duration time.Duration, err error) {
	// Normalize the query for consistent tracking
	normalizedQuery := normalizeQuery(query)

	c.mu.Lock()
	metrics, exists := c.queryMetrics[normalizedQuery]
	if !exists {
		metrics = &QueryMetrics{
			Query:     normalizedQuery,
			QueryType: queryType,
			Table:     table,
			MinTime:   duration,
			MaxTime:   duration,
		}
		c.queryMetrics[normalizedQuery] = metrics
	}
	c.mu.Unlock()

	// Update metrics atomically
	atomic.AddUint64(&metrics.Count, 1)
	atomic.AddInt64((*int64)(&metrics.TotalTime), int64(duration))
	// Store LastExecuted as Unix timestamp (nanoseconds) in a separate int64 field
	atomic.StoreInt64(&metrics.lastExecutedUnix, time.Now().UnixNano())

	// Update min/max times with locking
	c.mu.Lock()
	if duration < metrics.MinTime {
		metrics.MinTime = duration
	}
	if duration > metrics.MaxTime {
		metrics.MaxTime = duration
	}
	if err != nil {
		metrics.ErrorCount++
	}
	c.mu.Unlock()

	// Update global counters
	atomic.AddUint64(&c.totalQueries, 1)
	if err != nil {
		atomic.AddUint64(&c.totalErrors, 1)
	}
	if duration > c.slowQueryThreshold {
		atomic.AddUint64(&c.slowQueries, 1)
	}
}

// GetQueryMetrics returns metrics for a specific query.
func (c *DefaultMetricsCollector) GetQueryMetrics(query string) (*QueryMetrics, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	normalizedQuery := normalizeQuery(query)
	metrics, exists := c.queryMetrics[normalizedQuery]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	metricsCopy := *metrics
	return &metricsCopy, true
}

// GetAllQueryMetrics returns all collected query metrics.
func (c *DefaultMetricsCollector) GetAllQueryMetrics() []*QueryMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*QueryMetrics, 0, len(c.queryMetrics))
	for _, metrics := range c.queryMetrics {
		metricsCopy := *metrics
		result = append(result, &metricsCopy)
	}

	return result
}

// GetTopQueriesByCount returns the most frequently executed queries.
func (c *DefaultMetricsCollector) GetTopQueriesByCount(limit int) []*QueryMetrics {
	allMetrics := c.GetAllQueryMetrics()

	// Simple bubble sort for small datasets
	n := len(allMetrics)
	for i := 0; i < n && i < limit; i++ {
		for j := i + 1; j < n; j++ {
			if allMetrics[j].Count > allMetrics[i].Count {
				allMetrics[i], allMetrics[j] = allMetrics[j], allMetrics[i]
			}
		}
	}

	if limit > len(allMetrics) {
		limit = len(allMetrics)
	}
	return allMetrics[:limit]
}

// GetTopQueriesByTime returns the slowest queries by average time.
func (c *DefaultMetricsCollector) GetTopQueriesByTime(limit int) []*QueryMetrics {
	allMetrics := c.GetAllQueryMetrics()

	// Sort by average time
	n := len(allMetrics)
	for i := 0; i < n && i < limit; i++ {
		for j := i + 1; j < n; j++ {
			if allMetrics[j].AverageTime() > allMetrics[i].AverageTime() {
				allMetrics[i], allMetrics[j] = allMetrics[j], allMetrics[i]
			}
		}
	}

	if limit > len(allMetrics) {
		limit = len(allMetrics)
	}
	return allMetrics[:limit]
}

// GetConnectionPoolMetrics returns current connection pool statistics.
func (c *DefaultMetricsCollector) GetConnectionPoolMetrics(db *sql.DB) ConnectionPoolMetrics {
	stats := db.Stats()
	return ConnectionPoolMetrics{
		MaxOpenConnections:  stats.MaxOpenConnections,
		OpenConnections:     stats.OpenConnections,
		InUse:               stats.InUse,
		Idle:                stats.Idle,
		WaitCount:           stats.WaitCount,
		WaitDuration:        stats.WaitDuration,
		// MaxWaitDuration not available in this Go version
		CollectedAt:         time.Now(),
	}
}

// GetHealthMetrics returns current database health metrics.
func (c *DefaultMetricsCollector) GetHealthMetrics() DatabaseHealthMetrics {
	c.healthMu.RLock()
	defer c.healthMu.RUnlock()

	return DatabaseHealthMetrics{
		Connected:           c.connected,
		LastPingSuccess:     c.lastPingSuccess,
		LastPingLatency:     c.lastPingLatency,
		ConsecutiveFailures: c.consecutiveFailures,
		TotalQueries:        atomic.LoadUint64(&c.totalQueries),
		TotalErrors:         atomic.LoadUint64(&c.totalErrors),
		SlowQueries:         atomic.LoadUint64(&c.slowQueries),
		Uptime:              time.Since(c.startupTime),
		CollectedAt:         time.Now(),
	}
}

// RecordPing records a ping result.
func (c *DefaultMetricsCollector) RecordPing(success bool, latency time.Duration) {
	c.healthMu.Lock()
	defer c.healthMu.Unlock()

	c.lastPingLatency = latency
	c.connected = success

	if success {
		c.lastPingSuccess = time.Now()
		c.consecutiveFailures = 0
	} else {
		c.consecutiveFailures++
	}
}

// Reset clears all collected metrics.
func (c *DefaultMetricsCollector) Reset() {
	c.mu.Lock()
	c.queryMetrics = make(map[string]*QueryMetrics)
	c.mu.Unlock()

	atomic.StoreUint64(&c.totalQueries, 0)
	atomic.StoreUint64(&c.totalErrors, 0)
	atomic.StoreUint64(&c.slowQueries, 0)

	c.healthMu.Lock()
	c.consecutiveFailures = 0
	c.healthMu.Unlock()
}

// StartCollection begins periodic metric collection.
func (c *DefaultMetricsCollector) StartCollection(ctx context.Context, db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.collectPeriodicMetrics(ctx, db)
			}
		}
	}()
}

func (c *DefaultMetricsCollector) collectPeriodicMetrics(ctx context.Context, db *sql.DB) {
	// Perform health check ping
	start := time.Now()
	err := db.PingContext(ctx)
	latency := time.Since(start)
	c.RecordPing(err == nil, latency)

	if err != nil {
		slog.Error("database health check failed",
			"error", err,
			"consecutive_failures", c.consecutiveFailures,
		)
	}

	// Log pool metrics if waiting
	poolMetrics := c.GetConnectionPoolMetrics(db)
	if poolMetrics.WaitCount > 0 {
		slog.Warn("database connection pool waiting",
			"wait_count", poolMetrics.WaitCount,
			"wait_duration_ms", poolMetrics.WaitDuration.Milliseconds(),
			"open_connections", poolMetrics.OpenConnections,
			"in_use", poolMetrics.InUse,
			"idle", poolMetrics.Idle,
		)
	}

	// Alert on connection pool exhaustion
	if poolMetrics.OpenConnections >= poolMetrics.MaxOpenConnections {
		slog.Error("database connection pool exhausted",
			"max_open", poolMetrics.MaxOpenConnections,
			"open", poolMetrics.OpenConnections,
			"wait_count", poolMetrics.WaitCount,
		)
	}
}

// normalizeQuery creates a normalized version of a query for consistent tracking.
func normalizeQuery(query string) string {
	// Simple normalization - in production, this would be more sophisticated
	// to handle parameterized queries consistently
	return query
}

// InstrumentedDB wraps a Database and collects metrics on all operations.
type InstrumentedDB struct {
	Database
	collector MetricsCollector
	logger    *slog.Logger
}

// NewInstrumentedDB creates a new instrumented database wrapper.
func NewInstrumentedDB(db Database, collector MetricsCollector, logger *slog.Logger) *InstrumentedDB {
	if logger == nil {
		logger = slog.Default()
	}
	return &InstrumentedDB{
		Database:  db,
		collector: collector,
		logger:    logger,
	}
}

// Collector returns the metrics collector.
func (i *InstrumentedDB) Collector() MetricsCollector {
	return i.collector
}

// ExecContext executes a query without returning rows, with metrics collection.
func (i *InstrumentedDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := i.Database.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	queryType, table := parseQueryInfo(query)
	i.collector.RecordQuery(query, queryType, table, duration, err)

	return result, err
}

// QueryContext executes a query that returns rows, with metrics collection.
func (i *InstrumentedDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := i.Database.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	queryType, table := parseQueryInfo(query)
	i.collector.RecordQuery(query, queryType, table, duration, err)

	return rows, err
}

// QueryRowContext executes a query that returns a single row, with metrics collection.
func (i *InstrumentedDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := i.Database.QueryRowContext(ctx, query, args...)
	duration := time.Since(start)

	// Note: We can't detect errors until Scan() is called, so we record timing only
	queryType, table := parseQueryInfo(query)
	i.collector.RecordQuery(query, queryType, table, duration, nil)

	return row
}

// parseQueryInfo extracts query type and table name from a SQL query.
func parseQueryInfo(query string) (queryType, table string) {
	// Simple parsing - in production, use a proper SQL parser
	query = normalizeQuery(query)

	var firstWord string
	fmt.Sscanf(query, "%s", &firstWord)

	switch firstWord {
	case "SELECT":
		queryType = "SELECT"
	case "INSERT":
		queryType = "INSERT"
	case "UPDATE":
		queryType = "UPDATE"
	case "DELETE":
		queryType = "DELETE"
	case "CREATE":
		queryType = "CREATE"
	case "DROP":
		queryType = "DROP"
	case "ALTER":
		queryType = "ALTER"
	default:
		queryType = "OTHER"
	}

	// Extract table name (simplified)
	// In production, use a proper SQL parser
	table = extractTableName(query)

	return queryType, table
}

// extractTableName extracts the table name from a SQL query (simplified).
func extractTableName(query string) string {
	// This is a simplified implementation
	// In production, use a proper SQL parser like sqlparser

	var table string
	// Very basic extraction - just for demonstration
	if _, err := fmt.Sscanf(query, "INSERT INTO %s", &table); err == nil {
		return table
	}
	if _, err := fmt.Sscanf(query, "UPDATE %s", &table); err == nil {
		return table
	}
	if _, err := fmt.Sscanf(query, "DELETE FROM %s", &table); err == nil {
		return table
	}
	if _, err := fmt.Sscanf(query, "SELECT * FROM %s", &table); err == nil {
		return table
	}

	return "unknown"
}

// DashboardData represents data for the performance dashboard.
type DashboardData struct {
	// Health metrics
	Health DatabaseHealthMetrics `json:"health"`
	// Connection pool metrics
	Pool ConnectionPoolMetrics `json:"pool"`
	// Top queries by execution count
	TopQueriesByCount []*QueryMetrics `json:"top_queries_by_count"`
	// Top queries by average time
	TopQueriesByTime []*QueryMetrics `json:"top_queries_by_time"`
	// Queries with errors
	QueriesWithErrors []*QueryMetrics `json:"queries_with_errors"`
	// Collection timestamp
	Timestamp time.Time `json:"timestamp"`
}

// GetDashboardData returns all metrics formatted for the dashboard.
func (c *DefaultMetricsCollector) GetDashboardData(db *sql.DB) *DashboardData {
	allMetrics := c.GetAllQueryMetrics()

	// Filter queries with errors
	var errorQueries []*QueryMetrics
	for _, m := range allMetrics {
		if m.ErrorCount > 0 {
			mCopy := *m
			errorQueries = append(errorQueries, &mCopy)
		}
	}

	return &DashboardData{
		Health:            c.GetHealthMetrics(),
		Pool:              c.GetConnectionPoolMetrics(db),
		TopQueriesByCount: c.GetTopQueriesByCount(10),
		TopQueriesByTime:  c.GetTopQueriesByTime(10),
		QueriesWithErrors: errorQueries,
		Timestamp:         time.Now(),
	}
}

// MetricsExporter handles exporting metrics to external systems.
type MetricsExporter interface {
	Export(metrics *DashboardData) error
}

// PrometheusExporter exports metrics in Prometheus format.
type PrometheusExporter struct {
	prefix string
}

// NewPrometheusExporter creates a new Prometheus exporter.
func NewPrometheusExporter(prefix string) *PrometheusExporter {
	if prefix == "" {
		prefix = "radgateway_db_"
	}
	return &PrometheusExporter{prefix: prefix}
}

// Export returns metrics in Prometheus exposition format.
func (e *PrometheusExporter) Export(metrics *DashboardData) string {
	var output string

	// Health metrics
	output += fmt.Sprintf("# HELP %sconnected Database connection status\n", e.prefix)
	output += fmt.Sprintf("# TYPE %sconnected gauge\n", e.prefix)
	connected := 0
	if metrics.Health.Connected {
		connected = 1
	}
	output += fmt.Sprintf("%sconnected %d\n", e.prefix, connected)

	output += fmt.Sprintf("# HELP %stotal_queries Total queries executed\n", e.prefix)
	output += fmt.Sprintf("# TYPE %stotal_queries counter\n", e.prefix)
	output += fmt.Sprintf("%stotal_queries %d\n", e.prefix, metrics.Health.TotalQueries)

	output += fmt.Sprintf("# HELP %stotal_errors Total query errors\n", e.prefix)
	output += fmt.Sprintf("# TYPE %stotal_errors counter\n", e.prefix)
	output += fmt.Sprintf("%stotal_errors %d\n", e.prefix, metrics.Health.TotalErrors)

	output += fmt.Sprintf("# HELP %sslow_queries Total slow queries\n", e.prefix)
	output += fmt.Sprintf("# TYPE %sslow_queries counter\n", e.prefix)
	output += fmt.Sprintf("%sslow_queries %d\n", e.prefix, metrics.Health.SlowQueries)

	output += fmt.Sprintf("# HELP %suptime_seconds Database uptime\n", e.prefix)
	output += fmt.Sprintf("# TYPE %suptime_seconds gauge\n", e.prefix)
	output += fmt.Sprintf("%suptime_seconds %.0f\n", e.prefix, metrics.Health.Uptime.Seconds())

	// Pool metrics
	output += fmt.Sprintf("# HELP %sconnections_open Current open connections\n", e.prefix)
	output += fmt.Sprintf("# TYPE %sconnections_open gauge\n", e.prefix)
	output += fmt.Sprintf("%sconnections_open %d\n", e.prefix, metrics.Pool.OpenConnections)

	output += fmt.Sprintf("# HELP %sconnections_in_use Current connections in use\n", e.prefix)
	output += fmt.Sprintf("# TYPE %sconnections_in_use gauge\n", e.prefix)
	output += fmt.Sprintf("%sconnections_in_use %d\n", e.prefix, metrics.Pool.InUse)

	output += fmt.Sprintf("# HELP %sconnections_idle Current idle connections\n", e.prefix)
	output += fmt.Sprintf("# TYPE %sconnections_idle gauge\n", e.prefix)
	output += fmt.Sprintf("%sconnections_idle %d\n", e.prefix, metrics.Pool.Idle)

	output += fmt.Sprintf("# HELP %sconnection_wait_count Total connection wait count\n", e.prefix)
	output += fmt.Sprintf("# TYPE %sconnection_wait_count counter\n", e.prefix)
	output += fmt.Sprintf("%sconnection_wait_count %d\n", e.prefix, metrics.Pool.WaitCount)

	// Query metrics
	output += fmt.Sprintf("# HELP %squery_count Query execution count\n", e.prefix)
	output += fmt.Sprintf("# TYPE %squery_count counter\n", e.prefix)
	for _, qm := range metrics.TopQueriesByCount {
		output += fmt.Sprintf("%squery_count{query=\"%s\",type=\"%s\",table=\"%s\"} %d\n",
			e.prefix, qm.Query, qm.QueryType, qm.Table, qm.Count)
	}

	output += fmt.Sprintf("# HELP %squery_duration_seconds Query execution time\n", e.prefix)
	output += fmt.Sprintf("# TYPE %squery_duration_seconds summary\n", e.prefix)
	for _, qm := range metrics.TopQueriesByTime {
		output += fmt.Sprintf("%squery_duration_seconds{query=\"%s\",type=\"%s\",quantile=\"0.5\"} %.6f\n",
			e.prefix, qm.Query, qm.QueryType, qm.AverageTime().Seconds())
		output += fmt.Sprintf("%squery_duration_seconds{query=\"%s\",type=\"%s\",quantile=\"1.0\"} %.6f\n",
			e.prefix, qm.Query, qm.QueryType, qm.MaxTime.Seconds())
	}

	return output
}
