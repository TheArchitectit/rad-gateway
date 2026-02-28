// Package api provides HTTP handlers for RAD Gateway.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"radgateway/internal/db"
)

// HealthHandler provides health check endpoints
type HealthHandler struct {
	database db.Database
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(database db.Database) *HealthHandler {
	return &HealthHandler{database: database}
}

// RegisterRoutes registers health endpoints
func (h *HealthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/health/db", h.handleDBHealth)
	mux.HandleFunc("/health/metrics", h.handleMetrics)
}

// handleHealth returns general health status
func (h *HealthHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
	}

	// Check database if available
	if h.database != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		dbStatus := "healthy"
		if err := h.database.Ping(ctx); err != nil {
			status["status"] = "degraded"
			dbStatus = "unhealthy"
		}
		status["database"] = dbStatus
	} else {
		status["database"] = "not_configured"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// handleDBHealth returns detailed database health
func (h *HealthHandler) handleDBHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.database == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "unavailable",
			"message": "Database not configured",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check connectivity
	dbStatus := "healthy"
	dbError := ""
	if err := h.database.Ping(ctx); err != nil {
		dbStatus = "unhealthy"
		dbError = err.Error()
	}

	// Get metrics if available
	metrics := make(map[string]interface{})
	if pgDB, ok := h.database.(interface{ GetMetrics() *db.MetricsCollector }); ok {
		collector := pgDB.GetMetrics()
		stats := collector.GetStats()
		health := collector.HealthCheck()

		metrics["queries"] = stats.QueryCount
		metrics["errors"] = stats.QueryErrors
		metrics["avg_latency_ms"] = stats.AvgLatencyMs
		metrics["healthy"] = health.Healthy
		metrics["error_rate"] = health.ErrorRate
	}

	response := map[string]interface{}{
		"status":    dbStatus,
		"timestamp": time.Now().UTC(),
		"metrics":   metrics,
	}

	if dbError != "" {
		response["error"] = dbError
	}

	w.Header().Set("Content-Type", "application/json")
	if dbStatus != "healthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(response)
}

// handleMetrics returns Prometheus-compatible metrics
func (h *HealthHandler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get database metrics if available
	var output string
	if h.database != nil {
		if pgDB, ok := h.database.(interface{ GetMetrics() *db.MetricsCollector }); ok {
			collector := pgDB.GetMetrics()
			output += formatDBMetrics(collector)
		}
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(output))
}

func formatDBMetrics(collector *db.MetricsCollector) string {
	stats := collector.GetStats()
	var output string

	output += fmt.Sprintf("# HELP radgateway_db_queries_total Total database queries\n")
	output += fmt.Sprintf("# TYPE radgateway_db_queries_total counter\n")
	output += fmt.Sprintf("radgateway_db_queries_total %d\n\n", stats.QueryCount)

	output += fmt.Sprintf("# HELP radgateway_db_query_errors_total Total database query errors\n")
	output += fmt.Sprintf("# TYPE radgateway_db_query_errors_total counter\n")
	output += fmt.Sprintf("radgateway_db_query_errors_total %d\n\n", stats.QueryErrors)

	output += fmt.Sprintf("# HELP radgateway_db_query_duration_avg_ms Average query duration\n")
	output += fmt.Sprintf("# TYPE radgateway_db_query_duration_avg_ms gauge\n")
	output += fmt.Sprintf("radgateway_db_query_duration_avg_ms %.2f\n\n", stats.AvgLatencyMs)

	return output
}
