// Package metrics provides Prometheus-compatible metrics collection for RAD Gateway.
package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Collector collects and exposes Prometheus-compatible metrics
type Collector struct {
	// HTTP metrics
	requestCount   int64
	requestErrors  int64
	requestDuration int64 // total milliseconds

	// Database metrics
	dbQueryCount   int64
	dbQueryErrors  int64
	dbQueryDuration int64

	// Provider metrics
	providerRequests  sync.Map // map[string]*ProviderMetrics
	providerErrors    sync.Map

	// A2A metrics
	a2aTaskCount    int64
	a2aTaskCompleted int64

	// System metrics
	startTime time.Time
}

// ProviderMetrics holds metrics for a specific provider
type ProviderMetrics struct {
	Requests int64
	Errors   int64
	Latency  int64 // total milliseconds
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{
		startTime: time.Now(),
	}
}

// RecordHTTPRequest records an HTTP request
func (c *Collector) RecordHTTPRequest(duration time.Duration, statusCode int) {
	atomic.AddInt64(&c.requestCount, 1)
	atomic.AddInt64(&c.requestDuration, duration.Milliseconds())
	if statusCode >= 400 {
		atomic.AddInt64(&c.requestErrors, 1)
	}
}

// RecordDBQuery records a database query
func (c *Collector) RecordDBQuery(duration time.Duration, err error) {
	atomic.AddInt64(&c.dbQueryCount, 1)
	atomic.AddInt64(&c.dbQueryDuration, duration.Milliseconds())
	if err != nil {
		atomic.AddInt64(&c.dbQueryErrors, 1)
	}
}

// RecordProviderRequest records a provider request
func (c *Collector) RecordProviderRequest(provider string, duration time.Duration, err error) {
	metrics, _ := c.providerRequests.LoadOrStore(provider, &ProviderMetrics{})
	if m, ok := metrics.(*ProviderMetrics); ok {
		atomic.AddInt64(&m.Requests, 1)
		atomic.AddInt64(&m.Latency, duration.Milliseconds())
		if err != nil {
			atomic.AddInt64(&m.Errors, 1)
		}
	}
}

// RecordA2ATask records A2A task creation/completion
func (c *Collector) RecordA2ATask(created bool) {
	if created {
		atomic.AddInt64(&c.a2aTaskCount, 1)
	} else {
		atomic.AddInt64(&c.a2aTaskCompleted, 1)
	}
}

// PrometheusFormat returns metrics in Prometheus exposition format
func (c *Collector) PrometheusFormat() string {
	var output string

	// HTTP metrics
	output += c.formatCounter("radgateway_http_requests_total", "", atomic.LoadInt64(&c.requestCount))
	output += c.formatCounter("radgateway_http_request_errors_total", "", atomic.LoadInt64(&c.requestErrors))
	if count := atomic.LoadInt64(&c.requestCount); count > 0 {
		avg := float64(atomic.LoadInt64(&c.requestDuration)) / float64(count)
		output += c.formatGauge("radgateway_http_request_duration_avg_ms", "", avg)
	}

	// Database metrics
	output += c.formatCounter("radgateway_db_queries_total", "", atomic.LoadInt64(&c.dbQueryCount))
	output += c.formatCounter("radgateway_db_query_errors_total", "", atomic.LoadInt64(&c.dbQueryErrors))
	if count := atomic.LoadInt64(&c.dbQueryCount); count > 0 {
		avg := float64(atomic.LoadInt64(&c.dbQueryDuration)) / float64(count)
		output += c.formatGauge("radgateway_db_query_duration_avg_ms", "", avg)
	}

	// Provider metrics
	c.providerRequests.Range(func(key, value interface{}) bool {
		provider := key.(string)
		if m, ok := value.(*ProviderMetrics); ok {
			output += c.formatCounter("radgateway_provider_requests_total", fmt.Sprintf(`provider="%s"`, provider), atomic.LoadInt64(&m.Requests))
			output += c.formatCounter("radgateway_provider_errors_total", fmt.Sprintf(`provider="%s"`, provider), atomic.LoadInt64(&m.Errors))
		}
		return true
	})

	// A2A metrics
	output += c.formatCounter("radgateway_a2a_tasks_created_total", "", atomic.LoadInt64(&c.a2aTaskCount))
	output += c.formatCounter("radgateway_a2a_tasks_completed_total", "", atomic.LoadInt64(&c.a2aTaskCompleted))

	// System metrics
	uptime := time.Since(c.startTime).Seconds()
	output += c.formatGauge("radgateway_uptime_seconds", "", uptime)

	return output
}

func (c *Collector) formatCounter(name, labels string, value int64) string {
	if labels != "" {
		return fmt.Sprintf("%s{%s} %d\n", name, labels, value)
	}
	return fmt.Sprintf("%s %d\n", name, value)
}

func (c *Collector) formatGauge(name, labels string, value float64) string {
	if labels != "" {
		return fmt.Sprintf("%s{%s} %.2f\n", name, labels, value)
	}
	return fmt.Sprintf("%s %.2f\n", name, value)
}

// Handler returns an HTTP handler for metrics endpoint
func (c *Collector) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(c.PrometheusFormat()))
	}
}
