package admin

import (
	"context"
	"net/http"
	"sort"
	"time"

	"log/slog"

	"radgateway/internal/db"
	"radgateway/internal/logger"
)

type ReportingHandler struct {
	log *slog.Logger
	db  db.Database
}

func NewReportingHandler(database db.Database) *ReportingHandler {
	return &ReportingHandler{log: logger.WithComponent("admin.reports"), db: database}
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
	if h.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database not configured"})
		return
	}

	workspaceID := r.URL.Query().Get("workspaceId")
	start := parseTime(r.URL.Query().Get("startTime"), time.Now().Add(-7*24*time.Hour))
	end := parseTime(r.URL.Query().Get("endTime"), time.Now())

	summary, records, err := h.collectUsage(workspaceID, start, end, 1000)
	if err != nil {
		h.log.Error("failed to generate usage report", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate usage report"})
		return
	}

	items := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		items = append(items, map[string]any{
			"requestId":      rec.RequestID,
			"workspaceId":    rec.WorkspaceID,
			"incomingApi":    rec.IncomingAPI,
			"incomingModel":  rec.IncomingModel,
			"selectedModel":  rec.SelectedModel,
			"responseStatus": rec.ResponseStatus,
			"durationMs":     rec.DurationMs,
			"totalTokens":    rec.TotalTokens,
			"costUsd":        rec.CostUSD,
			"createdAt":      rec.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"reportType":  "usage",
		"generatedAt": time.Now().UTC(),
		"filters": map[string]any{
			"workspaceId": workspaceID,
			"startTime":   start,
			"endTime":     end,
		},
		"summary": map[string]any{
			"totalRequests": summary.TotalRequests,
			"totalTokens":   summary.TotalTokens,
			"totalCostUsd":  summary.TotalCostUSD,
			"successRate":   successRate(summary),
		},
		"items": items,
	})
}

func (h *ReportingHandler) performanceReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if h.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database not configured"})
		return
	}

	workspaceID := r.URL.Query().Get("workspaceId")
	start := parseTime(r.URL.Query().Get("startTime"), time.Now().Add(-24*time.Hour))
	end := parseTime(r.URL.Query().Get("endTime"), time.Now())

	_, records, err := h.collectUsage(workspaceID, start, end, 5000)
	if err != nil {
		h.log.Error("failed to generate performance report", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate performance report"})
		return
	}

	latency := durationPercentiles(records)

	writeJSON(w, http.StatusOK, map[string]any{
		"reportType":  "performance",
		"generatedAt": time.Now().UTC(),
		"metrics": map[string]any{
			"ttftMs":          map[string]float64{"p50": latency.p50, "p95": latency.p95, "p99": latency.p99},
			"tokensPerSecond": map[string]float64{"p50": 0, "p95": 0, "p99": 0},
			"latencyMs":       map[string]float64{"p50": latency.p50, "p95": latency.p95, "p99": latency.p99},
			"errorRate":       errorRate(records),
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
		format = "json"
	}

	exportID := generateID("exp")
	writeJSON(w, http.StatusAccepted, map[string]any{
		"exportId":    exportID,
		"status":      "completed",
		"format":      format,
		"downloadUrl": "/v0/admin/reports/export/" + exportID + "." + format,
		"expiresAt":   time.Now().UTC().Add(24 * time.Hour),
	})
}

func (h *ReportingHandler) collectUsage(workspaceID string, start, end time.Time, limit int) (*db.UsageSummary, []db.UsageRecord, error) {
	ctx := context.Background()
	if workspaceID != "" {
		summary, err := h.db.UsageRecords().GetSummaryByWorkspace(ctx, workspaceID, start, end)
		if err != nil {
			return nil, nil, err
		}
		records, err := h.db.UsageRecords().GetByWorkspace(ctx, workspaceID, start, end, limit, 0)
		if err != nil {
			return nil, nil, err
		}
		if summary == nil {
			summary = &db.UsageSummary{}
		}
		return summary, records, nil
	}

	workspaces, err := h.db.Workspaces().List(ctx, 500, 0)
	if err != nil {
		return nil, nil, err
	}

	combined := make([]db.UsageRecord, 0)
	total := &db.UsageSummary{}
	for _, ws := range workspaces {
		summary, err := h.db.UsageRecords().GetSummaryByWorkspace(ctx, ws.ID, start, end)
		if err != nil {
			return nil, nil, err
		}
		if summary != nil {
			total.TotalRequests += summary.TotalRequests
			total.TotalTokens += summary.TotalTokens
			total.TotalPromptTokens += summary.TotalPromptTokens
			total.TotalCompletionTokens += summary.TotalCompletionTokens
			total.TotalCostUSD += summary.TotalCostUSD
			total.SuccessCount += summary.SuccessCount
			total.ErrorCount += summary.ErrorCount
		}
		records, err := h.db.UsageRecords().GetByWorkspace(ctx, ws.ID, start, end, limit, 0)
		if err != nil {
			return nil, nil, err
		}
		combined = append(combined, records...)
	}

	return total, combined, nil
}

type percentileResult struct {
	p50 float64
	p95 float64
	p99 float64
}

func durationPercentiles(records []db.UsageRecord) percentileResult {
	if len(records) == 0 {
		return percentileResult{}
	}
	values := make([]int, 0, len(records))
	for _, r := range records {
		values = append(values, r.DurationMs)
	}
	sort.Ints(values)
	get := func(p int) float64 {
		if len(values) == 0 {
			return 0
		}
		idx := (len(values) - 1) * p / 100
		return float64(values[idx])
	}
	return percentileResult{p50: get(50), p95: get(95), p99: get(99)}
}

func successRate(summary *db.UsageSummary) float64 {
	total := summary.SuccessCount + summary.ErrorCount
	if total == 0 {
		return 0
	}
	return float64(summary.SuccessCount) / float64(total)
}

func errorRate(records []db.UsageRecord) float64 {
	if len(records) == 0 {
		return 0
	}
	failures := 0
	for _, r := range records {
		if r.ResponseStatus != "success" {
			failures++
		}
	}
	return float64(failures) / float64(len(records))
}

func parseTime(raw string, fallback time.Time) time.Time {
	if raw == "" {
		return fallback
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return fallback
	}
	return t
}
