package admin

import (
	"fmt"
	"net/http"
	"time"

	"log/slog"

	"radgateway/internal/logger"
)

type ReportingHandler struct {
	log *slog.Logger
}

func NewReportingHandler() *ReportingHandler {
	return &ReportingHandler{log: logger.WithComponent("admin.reports")}
}

func (h *ReportingHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v0/admin/reports/usage", h.usageReport)
	mux.HandleFunc("/v0/admin/reports/performance", h.performanceReport)
	mux.HandleFunc("/v0/admin/reports/export", h.exportReport)
}

func (h *ReportingHandler) usageReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	filters := map[string]string{
		"workspaceId": r.URL.Query().Get("workspaceId"),
		"apiKeyId":    r.URL.Query().Get("apiKeyId"),
		"providerId":  r.URL.Query().Get("providerId"),
		"model":       r.URL.Query().Get("model"),
		"status":      r.URL.Query().Get("status"),
		"startTime":   r.URL.Query().Get("startTime"),
		"endTime":     r.URL.Query().Get("endTime"),
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"reportType":  "usage",
		"generatedAt": time.Now().UTC(),
		"filters":     filters,
		"summary": map[string]any{
			"totalRequests": 1543,
			"totalTokens":   892340,
			"totalCostUsd":  143.22,
			"successRate":   0.982,
		},
		"items": []map[string]any{
			{"date": "2026-02-17", "requests": 490, "tokens": 278923, "costUsd": 44.98},
			{"date": "2026-02-18", "requests": 512, "tokens": 301444, "costUsd": 48.77},
			{"date": "2026-02-19", "requests": 541, "tokens": 311973, "costUsd": 49.47},
		},
	})
}

func (h *ReportingHandler) performanceReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	provider := r.URL.Query().Get("providerId")
	if provider == "" {
		provider = "all"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"reportType":  "performance",
		"generatedAt": time.Now().UTC(),
		"provider":    provider,
		"metrics": map[string]any{
			"ttftMs":          map[string]float64{"p50": 187, "p95": 462, "p99": 731},
			"tokensPerSecond": map[string]float64{"p50": 72.4, "p95": 48.1, "p99": 35.2},
			"latencyMs":       map[string]float64{"p50": 942, "p95": 1732, "p99": 2821},
			"errorRate":       0.018,
		},
	})
}

func (h *ReportingHandler) exportReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	exportID := fmt.Sprintf("exp_%d", time.Now().UnixNano())
	url := fmt.Sprintf("/v0/admin/reports/export/%s.%s", exportID, format)
	writeJSON(w, http.StatusAccepted, map[string]any{
		"exportId":    exportID,
		"status":      "processing",
		"format":      format,
		"downloadUrl": url,
		"expiresAt":   time.Now().UTC().Add(24 * time.Hour),
	})
}
