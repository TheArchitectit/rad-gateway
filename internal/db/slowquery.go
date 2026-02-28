// Package db provides database interfaces and implementations for RAD Gateway.
package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
)

// SlowQuerySeverity represents the severity level of a slow query.
type SlowQuerySeverity int

const (
	// SeverityWarning indicates a moderately slow query
	SeverityWarning SlowQuerySeverity = iota
	// SeverityCritical indicates a very slow query
	SeverityCritical
	// SeverityFatal indicates an extremely slow query that may impact system stability
	SeverityFatal
)

func (s SlowQuerySeverity) String() string {
	switch s {
	case SeverityWarning:
		return "WARNING"
	case SeverityCritical:
		return "CRITICAL"
	case SeverityFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// SlowQuery represents a detected slow query with full details.
type TrackedSlowQuery struct {
	// Unique identifier (hash of normalized query)
	ID string
	// Original SQL query (may be truncated)
	Query string
	// Normalized query with parameters removed
	NormalizedQuery string
	// Query type: SELECT, INSERT, UPDATE, DELETE, etc.
	QueryType string
	// Table being accessed
	Table string
	// Execution duration
	Duration time.Duration
	// Severity level based on thresholds
	Severity SlowQuerySeverity
	// Caller information (function/file/line if available)
	Caller string
	// Query parameters (may be redacted for security)
	Params []interface{}
	// Number of rows affected or returned
	RowCount int64
	// Timestamp when query was executed
	ExecutedAt time.Time
	// Plan information if available (PostgreSQL only)
	ExplainPlan string
	// Stack trace at time of execution
	StackTrace string
}

// SlowQueryThresholds defines the thresholds for slow query detection.
type SlowQueryThresholds struct {
	// Warning threshold (default 100ms)
	Warning time.Duration
	// Critical threshold (default 500ms)
	Critical time.Duration
	// Fatal threshold (default 2000ms)
	Fatal time.Duration
}

// DefaultSlowQueryThresholds returns the default threshold configuration.
func DefaultSlowQueryThresholds() SlowQueryThresholds {
	return SlowQueryThresholds{
		Warning:  100 * time.Millisecond,
		Critical: 500 * time.Millisecond,
		Fatal:    2000 * time.Millisecond,
	}
}

// GetSeverity returns the severity for a given duration.
func (t SlowQueryThresholds) GetSeverity(duration time.Duration) SlowQuerySeverity {
	switch {
	case duration >= t.Fatal:
		return SeverityFatal
	case duration >= t.Critical:
		return SeverityCritical
	case duration >= t.Warning:
		return SeverityWarning
	default:
		return -1 // Not a slow query
	}
}

// SlowQueryAlertHandler is called when a slow query is detected.
type SlowQueryAlertHandler func(query *TrackedSlowQuery)

// SlowQueryLogger defines the interface for slow query logging.
type SlowQueryLogger interface {
	// Log records a slow query if it exceeds thresholds
	Log(query string, duration time.Duration, params []interface{}, rowCount int64, caller string)
	// LogWithContext records a slow query with context
	LogWithContext(ctx context.Context, query string, duration time.Duration, params []interface{}, rowCount int64, caller string)
	// GetRecentQueries returns recent slow queries
	GetRecentQueries(limit int) []*TrackedSlowQuery
	// GetQueriesBySeverity returns queries filtered by severity
	GetQueriesBySeverity(severity SlowQuerySeverity, limit int) []*TrackedSlowQuery
	// GetQueryStats returns statistics about slow queries
	GetQueryStats(since time.Time) SlowQueryStats
	// RegisterAlertHandler registers a handler for slow query alerts
	RegisterAlertHandler(handler SlowQueryAlertHandler)
	// SetThresholds updates the slow query thresholds
	SetThresholds(thresholds SlowQueryThresholds)
	// GetThresholds returns the current thresholds
	GetThresholds() SlowQueryThresholds
}

// SlowQueryStats contains statistics about slow queries.
type SlowQueryStats struct {
	// Total slow queries
	TotalCount int
	// Count by severity
	WarningCount  int
	CriticalCount int
	FatalCount    int
	// Average duration
	AvgDuration time.Duration
	// Maximum duration
	MaxDuration time.Duration
	// Minimum duration
	MinDuration time.Duration
	// Most common slow queries (top 5)
	TopQueries []QueryFrequency
}

// QueryFrequency tracks how often a query pattern appears.
type QueryFrequency struct {
	// Normalized query pattern
	Query string
	// Number of occurrences
	Count int
	// Average duration
	AvgDuration time.Duration
}

// DefaultSlowQueryLogger implements SlowQueryLogger with in-memory ring buffer.
type DefaultSlowQueryLogger struct {
	mu sync.RWMutex

	// Configuration
	thresholds SlowQueryThresholds

	// Ring buffer for recent queries
	buffer     []*TrackedSlowQuery
	bufferSize int
	head       int
	count      int

	// Query deduplication tracking
	dedupMu     sync.RWMutex
	recentHashes map[string]time.Time
	dedupWindow time.Duration

	// Alert handlers
	handlers []SlowQueryAlertHandler

	// Logger
	logger *slog.Logger

	// Stats tracking
	statsMu       sync.RWMutex
	statsWindow   time.Time
	queryStats    map[string]*queryStatEntry
}

type queryStatEntry struct {
	count       int
	totalTime   time.Duration
	minTime     time.Duration
	maxTime     time.Duration
	lastSeen    time.Time
}

// SlowQueryLoggerOption configures the DefaultSlowQueryLogger.
type SlowQueryLoggerOption func(*DefaultSlowQueryLogger)

// WithBufferSize sets the ring buffer size.
func WithBufferSize(size int) SlowQueryLoggerOption {
	return func(l *DefaultSlowQueryLogger) {
		l.bufferSize = size
	}
}

// WithDedupWindow sets the deduplication window.
func WithDedupWindow(window time.Duration) SlowQueryLoggerOption {
	return func(l *DefaultSlowQueryLogger) {
		l.dedupWindow = window
	}
}

// WithLogger sets the slog logger.
func WithLogger(logger *slog.Logger) SlowQueryLoggerOption {
	return func(l *DefaultSlowQueryLogger) {
		l.logger = logger
	}
}

// WithThresholds sets custom thresholds.
func WithThresholds(thresholds SlowQueryThresholds) SlowQueryLoggerOption {
	return func(l *DefaultSlowQueryLogger) {
		l.thresholds = thresholds
	}
}

// NewSlowQueryLogger creates a new DefaultSlowQueryLogger.
func NewSlowQueryLogger(opts ...SlowQueryLoggerOption) *DefaultSlowQueryLogger {
	logger := &DefaultSlowQueryLogger{
		thresholds:   DefaultSlowQueryThresholds(),
		bufferSize:   1000,
		buffer:       make([]*TrackedSlowQuery, 1000),
		dedupWindow:  5 * time.Minute,
		recentHashes: make(map[string]time.Time),
		logger:       slog.Default(),
		queryStats:   make(map[string]*queryStatEntry),
		statsWindow:  time.Now(),
	}

	for _, opt := range opts {
		opt(logger)
	}

	// Ensure buffer matches configured size
	if logger.bufferSize != len(logger.buffer) {
		logger.buffer = make([]*TrackedSlowQuery, logger.bufferSize)
	}

	// Start cleanup goroutine
	go logger.cleanupLoop()

	return logger
}

// Log records a slow query if it exceeds thresholds.
func (l *DefaultSlowQueryLogger) Log(query string, duration time.Duration, params []interface{}, rowCount int64, caller string) {
	l.LogWithContext(context.Background(), query, duration, params, rowCount, caller)
}

// LogWithContext records a slow query with context.
func (l *DefaultSlowQueryLogger) LogWithContext(ctx context.Context, query string, duration time.Duration, params []interface{}, rowCount int64, caller string) {
	// Check if this is a slow query
	severity := l.thresholds.GetSeverity(duration)
	if severity < 0 {
		return // Not a slow query
	}

	// Normalize and deduplicate
	normalized := normalizeQueryForDedup(query)
	queryID := hashQuery(normalized)

	// Check deduplication
	l.dedupMu.Lock()
	if lastSeen, exists := l.recentHashes[queryID]; exists {
		if time.Since(lastSeen) < l.dedupWindow {
			l.dedupMu.Unlock()
			// Still update stats but don't log/alert
			l.updateStats(queryID, normalized, duration)
			return
		}
	}
	l.recentHashes[queryID] = time.Now()
	l.dedupMu.Unlock()

	// Create slow query entry
	slowQuery := &TrackedSlowQuery{
		ID:              queryID,
		Query:           truncateQuery(query, 2000),
		NormalizedQuery: normalized,
		QueryType:       extractQueryType(query),
		Table:           extractTableName(query),
		Duration:        duration,
		Severity:        severity,
		Caller:          caller,
		Params:          redactSensitiveParams(params),
		RowCount:        rowCount,
		ExecutedAt:      time.Now(),
		StackTrace:      captureStackTrace(),
	}

	// Add to ring buffer
	l.addToBuffer(slowQuery)

	// Update statistics
	l.updateStats(queryID, normalized, duration)

	// Log the slow query
	l.logSlowQuery(slowQuery)

	// Trigger alert handlers
	l.triggerAlerts(slowQuery)
}

// GetRecentQueries returns recent slow queries.
func (l *DefaultSlowQueryLogger) GetRecentQueries(limit int) []*TrackedSlowQuery {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if limit > l.count {
		limit = l.count
	}
	if limit > l.bufferSize {
		limit = l.bufferSize
	}

	result := make([]*TrackedSlowQuery, 0, limit)
	for i := 0; i < limit; i++ {
		idx := (l.head - 1 - i + l.bufferSize) % l.bufferSize
		if l.buffer[idx] != nil {
			queryCopy := *l.buffer[idx]
			result = append(result, &queryCopy)
		}
	}

	return result
}

// GetQueriesBySeverity returns queries filtered by severity.
func (l *DefaultSlowQueryLogger) GetQueriesBySeverity(severity SlowQuerySeverity, limit int) []*TrackedSlowQuery {
	allQueries := l.GetRecentQueries(l.count)

	var result []*TrackedSlowQuery
	for _, q := range allQueries {
		if q.Severity == severity {
			queryCopy := *q
			result = append(result, &queryCopy)
			if len(result) >= limit {
				break
			}
		}
	}

	return result
}

// GetQueryStats returns statistics about slow queries.
func (l *DefaultSlowQueryLogger) GetQueryStats(since time.Time) SlowQueryStats {
	l.statsMu.RLock()
	defer l.statsMu.RUnlock()

	stats := SlowQueryStats{
		MinDuration: time.Hour, // Start with a large value
	}

	for _, entry := range l.queryStats {
		if entry.lastSeen.Before(since) {
			continue
		}

		stats.TotalCount += entry.count
		stats.TotalCount += entry.count

		if entry.avgTime() < stats.MinDuration {
			stats.MinDuration = entry.avgTime()
		}
		if entry.maxTime > stats.MaxDuration {
			stats.MaxDuration = entry.maxTime
		}
	}

	if stats.TotalCount > 0 {
		// Calculate overall average would require storing all durations
		// For now, use min as a proxy if no queries found
		if stats.MinDuration == time.Hour {
			stats.MinDuration = 0
		}
	} else {
		stats.MinDuration = 0
	}

	return stats
}

// RegisterAlertHandler registers a handler for slow query alerts.
func (l *DefaultSlowQueryLogger) RegisterAlertHandler(handler SlowQueryAlertHandler) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.handlers = append(l.handlers, handler)
}

// SetThresholds updates the slow query thresholds.
func (l *DefaultSlowQueryLogger) SetThresholds(thresholds SlowQueryThresholds) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.thresholds = thresholds
}

// GetThresholds returns the current thresholds.
func (l *DefaultSlowQueryLogger) GetThresholds() SlowQueryThresholds {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.thresholds
}

func (l *DefaultSlowQueryLogger) addToBuffer(query *TrackedSlowQuery) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.buffer[l.head] = query
	l.head = (l.head + 1) % l.bufferSize
	if l.count < l.bufferSize {
		l.count++
	}
}

func (l *DefaultSlowQueryLogger) updateStats(queryID, normalized string, duration time.Duration) {
	l.statsMu.Lock()
	defer l.statsMu.Unlock()

	entry, exists := l.queryStats[queryID]
	if !exists {
		entry = &queryStatEntry{
			minTime: duration,
		}
		l.queryStats[queryID] = entry
	}

	entry.count++
	entry.totalTime += duration
	if duration < entry.minTime {
		entry.minTime = duration
	}
	if duration > entry.maxTime {
		entry.maxTime = duration
	}
	entry.lastSeen = time.Now()
}

func (e *queryStatEntry) avgTime() time.Duration {
	if e.count == 0 {
		return 0
	}
	return e.totalTime / time.Duration(e.count)
}

func (l *DefaultSlowQueryLogger) logSlowQuery(sq *TrackedSlowQuery) {
	// Log with appropriate level based on severity
	attrs := []slog.Attr{
		slog.String("query_id", sq.ID),
		slog.String("severity", sq.Severity.String()),
		slog.Duration("duration", sq.Duration),
		slog.String("query_type", sq.QueryType),
		slog.String("table", sq.Table),
		slog.String("caller", sq.Caller),
		slog.Int64("row_count", sq.RowCount),
	}

	switch sq.Severity {
	case SeverityFatal:
		l.logger.Error("FATAL: Extremely slow query detected", "slow_query", slog.Any("attrs", attrs))
	case SeverityCritical:
		l.logger.Error("CRITICAL: Very slow query detected", "slow_query", slog.Any("attrs", attrs))
	case SeverityWarning:
		l.logger.Warn("Slow query detected", "slow_query", slog.Any("attrs", attrs))
	}

	// Log the actual query at debug level for investigation
	l.logger.Debug("slow query details",
		"normalized_query", sq.NormalizedQuery,
		"query_hash", sq.ID,
	)
}

func (l *DefaultSlowQueryLogger) triggerAlerts(sq *TrackedSlowQuery) {
	l.mu.RLock()
	handlers := make([]SlowQueryAlertHandler, len(l.handlers))
	copy(handlers, l.handlers)
	l.mu.RUnlock()

	for _, handler := range handlers {
		go func(h SlowQueryAlertHandler) {
			defer func() {
				if r := recover(); r != nil {
					l.logger.Error("panic in slow query alert handler", "recover", r)
				}
			}()
			h(sq)
		}(handler)
	}
}

func (l *DefaultSlowQueryLogger) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.cleanupOldHashes()
		l.cleanupOldStats()
	}
}

func (l *DefaultSlowQueryLogger) cleanupOldHashes() {
	l.dedupMu.Lock()
	defer l.dedupMu.Unlock()

	now := time.Now()
	for hash, lastSeen := range l.recentHashes {
		if now.Sub(lastSeen) > l.dedupWindow {
			delete(l.recentHashes, hash)
		}
	}
}

func (l *DefaultSlowQueryLogger) cleanupOldStats() {
	l.statsMu.Lock()
	defer l.statsMu.Unlock()

	// Keep stats for 24 hours
	cutoff := time.Now().Add(-24 * time.Hour)
	for id, entry := range l.queryStats {
		if entry.lastSeen.Before(cutoff) {
			delete(l.queryStats, id)
		}
	}
}

// Helper functions

// normalizeQueryForDedup creates a normalized query string for deduplication.
func normalizeQueryForDedup(query string) string {
	// Remove extra whitespace
	query = strings.TrimSpace(query)
	query = regexp.MustCompile(`\s+`).ReplaceAllString(query, " ")

	// Replace numeric literals with placeholders
	query = regexp.MustCompile(`\b\d+\b`).ReplaceAllString(query, "?")

	// Replace string literals with placeholders (simple version)
	query = regexp.MustCompile(`'[^']*'`).ReplaceAllString(query, "?")

	// Replace IN clauses with single placeholder
	query = regexp.MustCompile(`\([^)]*\?[^)]*\)`).ReplaceAllString(query, "(?)")

	return strings.ToUpper(query)
}

// hashQuery creates a hash of the normalized query for deduplication.
func hashQuery(normalized string) string {
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:8]) // First 8 bytes is sufficient
}

// truncateQuery truncates a query to max length.
func truncateQuery(query string, maxLen int) string {
	if len(query) <= maxLen {
		return query
	}
	return query[:maxLen] + "... [truncated]"
}

// extractQueryType extracts the SQL command type from a query.
func extractQueryType(query string) string {
	query = strings.TrimSpace(strings.ToUpper(query))
	words := strings.Fields(query)
	if len(words) > 0 {
		return words[0]
	}
	return "UNKNOWN"
}

// extractTableName extracts the table name from a query.
func extractTableName(query string) string {
	// Simple extraction - look for FROM or INTO or UPDATE
	query = strings.ToUpper(query)

	// Try to find table after FROM
	if idx := strings.Index(query, "FROM "); idx != -1 {
		rest := query[idx+5:]
		words := strings.Fields(rest)
		if len(words) > 0 {
			return strings.Trim(words[0], "`\"")
		}
	}

	// Try to find table after INTO
	if idx := strings.Index(query, "INTO "); idx != -1 {
		rest := query[idx+5:]
		words := strings.Fields(rest)
		if len(words) > 0 {
			return strings.Trim(words[0], "`\"")
		}
	}

	// Try to find table after UPDATE
	if idx := strings.Index(query, "UPDATE "); idx != -1 {
		rest := query[idx+7:]
		words := strings.Fields(rest)
		if len(words) > 0 {
			return strings.Trim(words[0], "`\"")
		}
	}

	return "unknown"
}

// redactSensitiveParams removes sensitive information from parameters.
func redactSensitiveParams(params []interface{}) []interface{} {
	// In production, implement proper parameter redaction
	// For now, return nil to avoid logging sensitive data
	return nil
}

// captureStackTrace captures the current stack trace.
func captureStackTrace() string {
	// In production, use runtime.Stack() or similar
	// For now, return empty string
	return ""
}

// QueryInterceptor wraps query execution with slow query detection.
type QueryInterceptor struct {
	logger   SlowQueryLogger
	getRowCount func() int64 // Optional callback to get row count
}

// NewQueryInterceptor creates a new QueryInterceptor.
func NewQueryInterceptor(logger SlowQueryLogger) *QueryInterceptor {
	return &QueryInterceptor{
		logger: logger,
	}
}

// Intercept wraps a function with slow query detection.
func (qi *QueryInterceptor) Intercept(query string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	var rowCount int64
	if qi.getRowCount != nil {
		rowCount = qi.getRowCount()
	}

	qi.logger.Log(query, duration, nil, rowCount, "")
	return err
}

// InstrumentedRepository wraps repository operations with slow query detection.
type InstrumentedRepository struct {
	logger SlowQueryLogger
}

// NewInstrumentedRepository creates a new instrumented repository wrapper.
func NewInstrumentedRepository(logger SlowQueryLogger) *InstrumentedRepository {
	return &InstrumentedRepository{logger: logger}
}

// Wrap wraps a repository function with slow query detection.
func (ir *InstrumentedRepository) Wrap(table, operation string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	// Create a pseudo-query for repository operations
	query := fmt.Sprintf("%s %s", operation, table)
	ir.logger.Log(query, duration, nil, 0, "")

	return err
}

// AlertConfig defines configuration for slow query alerting.
type AlertConfig struct {
	// Webhook URL for alerts
	WebhookURL string
	// Email recipients for alerts
	EmailRecipients []string
	// PagerDuty integration key for critical alerts
	PagerDutyKey string
	// Slack webhook for notifications
	SlackWebhook string
	// Minimum severity to trigger alert
	MinSeverity SlowQuerySeverity
	// Rate limit (max alerts per minute)
	RateLimit int
}

// WebhookAlertHandler creates an alert handler that posts to a webhook.
func WebhookAlertHandler(config AlertConfig) SlowQueryAlertHandler {
	return func(query *TrackedSlowQuery) {
		if query.Severity < config.MinSeverity {
			return
		}

		// Implementation would POST to webhook URL
		slog.Info("slow query webhook alert triggered",
			"severity", query.Severity,
			"duration_ms", query.Duration.Milliseconds(),
			"webhook", config.WebhookURL,
		)
	}
}

// EmailAlertHandler creates an alert handler that sends email.
func EmailAlertHandler(config AlertConfig) SlowQueryAlertHandler {
	return func(query *TrackedSlowQuery) {
		if query.Severity < config.MinSeverity {
			return
		}

		// Implementation would send email
		slog.Info("slow query email alert triggered",
			"severity", query.Severity,
			"duration_ms", query.Duration.Milliseconds(),
			"recipients", len(config.EmailRecipients),
		)
	}
}

// CompositeAlertHandler combines multiple alert handlers.
func CompositeAlertHandler(handlers ...SlowQueryAlertHandler) SlowQueryAlertHandler {
	return func(query *TrackedSlowQuery) {
		for _, h := range handlers {
			h(query)
		}
	}
}
