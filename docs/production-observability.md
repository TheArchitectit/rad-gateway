# Production Observability Stack

## Overview

This document describes the production observability features implemented for RAD Gateway, providing comprehensive monitoring, health checks, and metrics collection.

## Implemented Features

### 1. Prometheus Metrics Collection

Collects and exposes metrics in Prometheus exposition format.

**Key Components:**
- `internal/metrics/collector.go` - Metrics collector implementation
- `internal/middleware/middleware.go` - HTTP middleware for automatic request tracking

**Usage:**
```go
// Initialize collector
metricsCollector := metrics.NewCollector()

// Wrap handler with metrics middleware
handler := middleware.WithMetrics(metricsCollector)(handler)

// Register metrics endpoint
mux.Handle("/metrics", metricsCollector.Handler())
```

**Metrics Collected:**

| Metric | Type | Description |
|--------|------|-------------|
| `radgateway_http_requests_total` | counter | Total HTTP requests |
| `radgateway_http_request_errors_total` | counter | Total HTTP errors (4xx/5xx) |
| `radgateway_http_request_duration_avg_ms` | gauge | Average request duration |
| `radgateway_db_queries_total` | counter | Total database queries |
| `radgateway_db_query_errors_total` | counter | Total database query errors |
| `radgateway_db_query_duration_avg_ms` | gauge | Average query duration |
| `radgateway_provider_requests_total{provider="..."}` | counter | Per-provider requests |
| `radgateway_provider_errors_total{provider="..."}` | counter | Per-provider errors |
| `radgateway_a2a_tasks_created_total` | counter | A2A tasks created |
| `radgateway_a2a_tasks_completed_total` | counter | A2A tasks completed |
| `radgateway_uptime_seconds` | gauge | Process uptime |

**Example Output:**
```
# HELP radgateway_http_requests_total Total HTTP requests
# TYPE radgateway_http_requests_total counter
radgateway_http_requests_total 1523

# HELP radgateway_db_queries_total Total database queries
# TYPE radgateway_db_queries_total counter
radgateway_db_queries_total 456

radgateway_provider_requests_total{provider="openai"} 892
radgateway_provider_requests_total{provider="anthropic"} 631
```

---

### 2. Health Check Endpoints

Provides comprehensive health checking for the gateway and its dependencies.

**Key Components:**
- `internal/api/health.go` - Health check handler

**Endpoints:**

| Endpoint | Description |
|----------|-------------|
| `GET /health` | General health status |
| `GET /health/db` | Database health with metrics |
| `GET /health/metrics` | Prometheus-compatible metrics (legacy) |

**Usage:**
```go
// Register health endpoints
healthHandler := api.NewHealthHandler(database)
healthHandler.RegisterRoutes(mux)
```

**Response Examples:**

`/health`:
```json
{
  "status": "healthy",
  "timestamp": "2026-02-28T12:00:00Z",
  "database": "healthy"
}
```

`/health/db`:
```json
{
  "status": "healthy",
  "timestamp": "2026-02-28T12:00:00Z",
  "metrics": {
    "queries": 456,
    "errors": 2,
    "avg_latency_ms": 12.5,
    "healthy": true,
    "error_rate": 0.0044
  }
}
```

**Status Codes:**
- `200 OK` - Healthy
- `503 Service Unavailable` - Degraded or unhealthy

---

### 3. Structured Logging

Already implemented via `internal/logger` package with component-based logging.

**Features:**
- JSON-formatted logs
- Component tagging
- Request ID correlation
- Multiple log levels (DEBUG, INFO, WARN, ERROR)

---

### 4. Request Context

Provides request tracking through context propagation.

**Context Keys:**
- `request_id` - Unique request identifier
- `trace_id` - Distributed trace identifier
- `api_key` - Authenticated API key
- `api_key_name` - Named API key reference

---

## Configuration

### Environment Variables

```bash
# Metrics and health are always enabled
# No additional configuration required

# Optional: Configure Prometheus scrape interval
export PROMETHEUS_SCRAPE_INTERVAL="15s"
```

### Prometheus Scraping

Add to `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'radgateway'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

---

## Monitoring Dashboards

### Key Metrics to Monitor

**Health Indicators:**
- `radgateway_http_requests_total` - Request volume
- `radgateway_http_request_errors_total` - Error count
- `radgateway_db_query_errors_total` - Database errors
- `radgateway_uptime_seconds` - Availability

**Performance Indicators:**
- `radgateway_http_request_duration_avg_ms` - Response time
- `radgateway_db_query_duration_avg_ms` - Database latency
- `radgateway_provider_requests_total` - Provider distribution

**Alert Thresholds:**
- Error rate > 5%: Warning
- Error rate > 10%: Critical
- P99 latency > 1000ms: Warning
- Database error rate > 1%: Critical

---

## Testing

### Run Observability Tests

```bash
# Metrics tests
go test ./internal/metrics/... -v

# API/health tests
go test ./internal/api/... -v

# Middleware tests
go test ./internal/middleware/... -v
```

### Manual Verification

```bash
# Start server
go run ./cmd/rad-gateway

# Check health
curl http://localhost:8080/health

# Check database health
curl http://localhost:8080/health/db

# Get Prometheus metrics
curl http://localhost:8080/metrics
```

---

## Files Added/Modified

**New Files:**
- `internal/metrics/collector.go` - Metrics collection
- `internal/metrics/collector_test.go` - Metrics tests
- `docs/production-observability.md` - This documentation

**Modified:**
- `internal/api/health.go` - Health check handler
- `internal/middleware/middleware.go` - Metrics middleware
- `cmd/rad-gateway/main.go` - Wiring

---

## Next Steps

- [ ] Add Grafana dashboard JSON
- [ ] Implement distributed tracing with OpenTelemetry
- [ ] Add custom business metrics (requests per model, etc.)
- [ ] Create alerting rules for Prometheus
- [ ] Add pprof endpoints for profiling

---

**Last Updated:** 2026-02-28
**Version:** Phase 4 Observability Stack
