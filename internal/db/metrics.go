// Package db provides database interfaces and implementations for RAD Gateway.
package db

import (
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects database performance metrics
type MetricsCollector struct {
	queryCount   int64
	queryErrors  int64
	totalLatency int64 // in milliseconds

	// Query type breakdown
	queryTypes sync.Map // map[string]*QueryTypeStats

	// Pool metrics
	poolStats PoolStats
}

// QueryTypeStats holds stats for a specific query type
type QueryTypeStats struct {
	Count   int64
	Errors  int64
	Latency int64 // total latency in ms
}

// PoolStats holds connection pool metrics
type PoolStats struct {
	OpenConns    int32
	IdleConns    int32
	InUseConns   int32
	WaitCount    int64
	WaitDuration int64 // total wait time in ms
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// RecordQuery records a query execution
func (m *MetricsCollector) RecordQuery(queryType string, duration time.Duration, err error) {
	atomic.AddInt64(&m.queryCount, 1)
	atomic.AddInt64(&m.totalLatency, duration.Milliseconds())

	if err != nil {
		atomic.AddInt64(&m.queryErrors, 1)
	}

	// Update query type stats
	stats, _ := m.queryTypes.LoadOrStore(queryType, &QueryTypeStats{})
	if s, ok := stats.(*QueryTypeStats); ok {
		atomic.AddInt64(&s.Count, 1)
		atomic.AddInt64(&s.Latency, duration.Milliseconds())
		if err != nil {
			atomic.AddInt64(&s.Errors, 1)
		}
	}
}

// GetStats returns current database statistics
func (m *MetricsCollector) GetStats() DatabaseStats {
	return DatabaseStats{
		QueryCount:   atomic.LoadInt64(&m.queryCount),
		QueryErrors:  atomic.LoadInt64(&m.queryErrors),
		AvgLatencyMs: m.getAvgLatency(),
		QueryTypes:   m.getQueryTypeStats(),
	}
}

func (m *MetricsCollector) getAvgLatency() float64 {
	count := atomic.LoadInt64(&m.queryCount)
	if count == 0 {
		return 0
	}
	return float64(atomic.LoadInt64(&m.totalLatency)) / float64(count)
}

func (m *MetricsCollector) getQueryTypeStats() map[string]QueryTypeStats {
	result := make(map[string]QueryTypeStats)
	m.queryTypes.Range(func(key, value interface{}) bool {
		if s, ok := value.(*QueryTypeStats); ok {
			result[key.(string)] = QueryTypeStats{
				Count:   atomic.LoadInt64(&s.Count),
				Errors:  atomic.LoadInt64(&s.Errors),
				Latency: atomic.LoadInt64(&s.Latency),
			}
		}
		return true
	})
	return result
}

// DatabaseStats holds database statistics
type DatabaseStats struct {
	QueryCount   int64
	QueryErrors  int64
	AvgLatencyMs float64
	QueryTypes   map[string]QueryTypeStats
}

// HealthCheck returns database health status
func (m *MetricsCollector) HealthCheck() HealthStatus {
	stats := m.GetStats()

	// Calculate error rate
	errorRate := float64(0)
	if stats.QueryCount > 0 {
		errorRate = float64(stats.QueryErrors) / float64(stats.QueryCount)
	}

	return HealthStatus{
		Healthy:    errorRate < 0.05, // Less than 5% error rate
		ErrorRate:  errorRate,
		AvgLatency: stats.AvgLatencyMs,
		QueryCount: stats.QueryCount,
	}
}

// HealthStatus represents database health
type HealthStatus struct {
	Healthy    bool
	ErrorRate  float64
	AvgLatency float64
	QueryCount int64
}
