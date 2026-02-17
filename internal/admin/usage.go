package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"radgateway/internal/logger"
)

// UsageQueryRequest represents a request to query usage data
type UsageQueryRequest struct {
	StartTime    *time.Time `json:"startTime,omitempty"`
	EndTime      *time.Time `json:"endTime,omitempty"`
	WorkspaceID  string     `json:"workspaceId,omitempty"`
	APIKeyID     string     `json:"apiKeyId,omitempty"`
	ProviderID   string     `json:"providerId,omitempty"`
	Model        string     `json:"model,omitempty"`
	IncomingAPI  string     `json:"incomingApi,omitempty"`
	Status       string     `json:"status,omitempty"`
	GroupBy      []string   `json:"groupBy,omitempty"`
	Aggregations []string   `json:"aggregations,omitempty"`
}

// UsageRecordResponse represents a usage record in API responses
type UsageRecordResponse struct {
	ID               string     `json:"id"`
	WorkspaceID      string     `json:"workspaceId"`
	RequestID        string     `json:"requestId"`
	TraceID          string     `json:"traceId"`
	APIKeyID         *string    `json:"apiKeyId,omitempty"`
	ControlRoomID    *string    `json:"controlRoomId,omitempty"`
	IncomingAPI      string     `json:"incomingApi"`
	IncomingModel    string     `json:"incomingModel"`
	SelectedModel    *string    `json:"selectedModel,omitempty"`
	ProviderID       *string    `json:"providerId,omitempty"`
	PromptTokens     int64      `json:"promptTokens"`
	CompletionTokens int64      `json:"completionTokens"`
	TotalTokens      int64      `json:"totalTokens"`
	CostUSD          *float64   `json:"costUsd,omitempty"`
	DurationMs       int        `json:"durationMs"`
	ResponseStatus   string     `json:"responseStatus"`
	ErrorCode        *string    `json:"errorCode,omitempty"`
	ErrorMessage     *string    `json:"errorMessage,omitempty"`
	Attempts         int        `json:"attempts"`
	StartedAt        time.Time  `json:"startedAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
}

// UsageAggregation represents aggregated usage data
type UsageAggregation struct {
	Dimension string                 `json:"dimension"`
	Value     string                 `json:"value"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// UsageListResponse represents the list response
type UsageListResponse struct {
	Data         []UsageRecordResponse `json:"data"`
	Aggregations []UsageAggregation    `json:"aggregations,omitempty"`
	Summary      UsageSummary          `json:"summary"`
	Total        int                   `json:"total"`
	Page         int                   `json:"page"`
	PageSize     int                   `json:"pageSize"`
	HasMore      bool                  `json:"hasMore"`
}

// UsageSummary represents summary statistics
type UsageSummary struct {
	TotalRequests      int64   `json:"totalRequests"`
	TotalTokens        int64   `json:"totalTokens"`
	TotalPromptTokens  int64   `json:"totalPromptTokens"`
	TotalOutputTokens  int64   `json:"totalOutputTokens"`
	TotalCostUSD       float64 `json:"totalCostUsd"`
	AvgDurationMs      float64 `json:"avgDurationMs"`
	SuccessCount       int64   `json:"successCount"`
	ErrorCount         int64   `json:"errorCount"`
	ErrorRate          float64 `json:"errorRate"`
}

// UsageTrendResponse represents usage trend data over time
type UsageTrendResponse struct {
	TimeRange TimeRange        `json:"timeRange"`
	Interval  string           `json:"interval"`
	Points    []UsageTrendPoint `json:"points"`
}

// TimeRange represents a time range
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// UsageTrendPoint represents a single point in a trend
type UsageTrendPoint struct {
	Timestamp        time.Time `json:"timestamp"`
	RequestCount     int64     `json:"requestCount"`
	TokenCount       int64     `json:"tokenCount"`
	CostUSD          float64   `json:"costUsd"`
	AvgLatencyMs     float64   `json:"avgLatencyMs"`
	ErrorCount       int64     `json:"errorCount"`
}

// UsageExportRequest represents a request to export usage data
type UsageExportRequest struct {
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Format      string    `json:"format"` // csv, json
	WorkspaceID string    `json:"workspaceId,omitempty"`
	IncludeCost bool      `json:"includeCost"`
}

// UsageExportResponse represents an export response
type UsageExportResponse struct {
	ExportID    string    `json:"exportId"`
	Status      string    `json:"status"`
	DownloadURL *string   `json:"downloadUrl,omitempty"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	RecordCount int       `json:"recordCount"`
}

// UsageHandler handles usage query endpoints
type UsageHandler struct {
	log *slog.Logger
}

// NewUsageHandler creates a new usage handler
func NewUsageHandler() *UsageHandler {
	return &UsageHandler{
		log: logger.WithComponent("admin.usage"),
	}
}

// RegisterRoutes registers the usage query routes
func (h *UsageHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v0/admin/usage", h.handleUsage)
	mux.HandleFunc("/v0/admin/usage/records", h.handleRecords)
	mux.HandleFunc("/v0/admin/usage/trends", h.handleTrends)
	mux.HandleFunc("/v0/admin/usage/summary", h.handleSummary)
	mux.HandleFunc("/v0/admin/usage/export", h.handleExport)
	mux.HandleFunc("/v0/admin/usage/export/", h.handleExportStatus)
}

func (h *UsageHandler) handleUsage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.queryUsage(w, r)
	case http.MethodPost:
		h.advancedQuery(w, r)
	default:
		h.log.Warn("method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *UsageHandler) handleRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	h.getRecords(w, r)
}

func (h *UsageHandler) handleTrends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	h.getTrends(w, r)
}

func (h *UsageHandler) handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	h.getSummary(w, r)
}

func (h *UsageHandler) handleExport(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createExport(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *UsageHandler) handleExportStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	exportID := strings.TrimPrefix(r.URL.Path, "/v0/admin/usage/export/")
	h.getExportStatus(w, r, exportID)
}

// queryUsage handles basic usage queries with URL parameters
func (h *UsageHandler) queryUsage(w http.ResponseWriter, r *http.Request) {
	filter := h.parseUsageFilter(r)
	page := readIntParam(r, "page", 1)
	pageSize := readIntParam(r, "pageSize", 50)
	if pageSize > 500 {
		pageSize = 500
	}

	h.log.Debug("querying usage",
		"page", page,
		"pageSize", pageSize,
		"startTime", filter.StartTime,
		"endTime", filter.EndTime,
	)

	// Generate mock usage records
	records := h.generateMockRecords(filter)

	// Calculate summary
	summary := h.calculateSummary(records)

	total := len(records)

	// Apply pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	pagedRecords := records[start:end]

	response := UsageListResponse{
		Data:    pagedRecords,
		Summary: summary,
		Total:   total,
		Page:    page,
		PageSize: pageSize,
		HasMore: end < total,
	}

	writeJSON(w, http.StatusOK, response)
}

// advancedQuery handles advanced usage queries with aggregations
func (h *UsageHandler) advancedQuery(w http.ResponseWriter, r *http.Request) {
	var req UsageQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	h.log.Debug("advanced usage query",
		"workspace", req.WorkspaceID,
		"groupBy", req.GroupBy,
		"aggregations", req.Aggregations,
	)

	// Generate mock data
	records := h.generateMockRecords(UsageFilter{
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		WorkspaceID: req.WorkspaceID,
		APIKeyID:    req.APIKeyID,
		ProviderID:  req.ProviderID,
		Model:       req.Model,
		Status:      req.Status,
	})

	// Calculate aggregations if requested
	var aggregations []UsageAggregation
	if len(req.GroupBy) > 0 {
		aggregations = h.calculateAggregations(records, req.GroupBy, req.Aggregations)
	}

	summary := h.calculateSummary(records)

	response := UsageListResponse{
		Data:         records,
		Aggregations: aggregations,
		Summary:      summary,
		Total:        len(records),
	}

	writeJSON(w, http.StatusOK, response)
}

// getRecords returns individual usage records
func (h *UsageHandler) getRecords(w http.ResponseWriter, r *http.Request) {
	filter := h.parseUsageFilter(r)
	page := readIntParam(r, "page", 1)
	pageSize := readIntParam(r, "pageSize", 100)
	if pageSize > 1000 {
		pageSize = 1000
	}

	records := h.generateMockRecords(filter)
	total := len(records)

	// Apply pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	pagedRecords := records[start:end]

	response := UsageListResponse{
		Data:    pagedRecords,
		Total:   total,
		Page:    page,
		PageSize: pageSize,
		HasMore: end < total,
	}

	writeJSON(w, http.StatusOK, response)
}

// getTrends returns usage trends over time
func (h *UsageHandler) getTrends(w http.ResponseWriter, r *http.Request) {
	startTime := h.parseTimeParam(r, "startTime", time.Now().Add(-7*24*time.Hour))
	endTime := h.parseTimeParam(r, "endTime", time.Now())
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "hour"
	}

	validIntervals := map[string]bool{"minute": true, "hour": true, "day": true}
	if !validIntervals[interval] {
		interval = "hour"
	}

	h.log.Debug("getting usage trends",
		"startTime", startTime,
		"endTime", endTime,
		"interval", interval,
	)

	// Generate trend points
	points := h.generateTrendPoints(startTime, endTime, interval)

	response := UsageTrendResponse{
		TimeRange: TimeRange{Start: startTime, End: endTime},
		Interval:  interval,
		Points:    points,
	}

	writeJSON(w, http.StatusOK, response)
}

// getSummary returns usage summary statistics
func (h *UsageHandler) getSummary(w http.ResponseWriter, r *http.Request) {
	filter := h.parseUsageFilter(r)

	h.log.Debug("getting usage summary")

	records := h.generateMockRecords(filter)
	summary := h.calculateSummary(records)

	writeJSON(w, http.StatusOK, summary)
}

// createExport creates a usage data export
func (h *UsageHandler) createExport(w http.ResponseWriter, r *http.Request) {
	var req UsageExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Format == "" {
		req.Format = "json"
	}

	validFormats := map[string]bool{"json": true, "csv": true}
	if !validFormats[req.Format] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid format, must be 'json' or 'csv'"})
		return
	}

	h.log.Info("creating usage export",
		"format", req.Format,
		"startTime", req.StartTime,
		"endTime", req.EndTime,
	)

	exportID := generateID("exp")
	expiresAt := time.Now().Add(24 * time.Hour)
	downloadURL := fmt.Sprintf("/v0/admin/usage/export/%s/download", exportID)

	response := UsageExportResponse{
		ExportID:    exportID,
		Status:      "pending",
		DownloadURL: &downloadURL,
		ExpiresAt:   &expiresAt,
		RecordCount: 0,
	}

	writeJSON(w, http.StatusAccepted, response)
}

// getExportStatus returns the status of an export
func (h *UsageHandler) getExportStatus(w http.ResponseWriter, r *http.Request, exportID string) {
	h.log.Debug("getting export status", "exportId", exportID)

	expiresAt := time.Now().Add(24 * time.Hour)
	downloadURL := fmt.Sprintf("/v0/admin/usage/export/%s/download", exportID)

	response := UsageExportResponse{
		ExportID:    exportID,
		Status:      "completed",
		DownloadURL: &downloadURL,
		ExpiresAt:   &expiresAt,
		RecordCount: 1000,
	}

	writeJSON(w, http.StatusOK, response)
}

// Helper methods

func (h *UsageHandler) generateMockRecords(filter UsageFilter) []UsageRecordResponse {
	var records []UsageRecordResponse

	now := time.Now()
	models := []string{"gpt-4o-mini", "claude-3-5-sonnet", "gemini-1.5-flash"}
	apis := []string{"chat", "embeddings", "responses"}
	providers := []string{"openai", "anthropic", "gemini"}
	workspaces := []string{"ws_001", "ws_002", "ws_003"}

	// Generate 50 mock records
	for i := 0; i < 50; i++ {
		startedAt := now.Add(-time.Duration(i*30) * time.Minute)
		completedAt := startedAt.Add(time.Duration(100+i*10) * time.Millisecond)
		totalTokens := int64(500 + i*10)
		promptTokens := int64(totalTokens / 2)
		completionTokens := totalTokens - promptTokens
		cost := float64(totalTokens) * 0.00001

		record := UsageRecordResponse{
			ID:               generateID("req"),
			WorkspaceID:     workspaces[i%len(workspaces)],
			RequestID:        generateID("req"),
			TraceID:          generateID("trace"),
			APIKeyID:         strPtr("key_001"),
			IncomingAPI:      apis[i%len(apis)],
			IncomingModel:    models[i%len(models)],
			SelectedModel:    strPtr(models[i%len(models)]),
			ProviderID:       strPtr(providers[i%len(providers)]),
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
			CostUSD:          &cost,
			DurationMs:       100 + i*10,
			ResponseStatus:   "success",
			Attempts:         1,
			StartedAt:        startedAt,
			CompletedAt:      &completedAt,
		}

		// Apply filters
		if filter.WorkspaceID != "" && record.WorkspaceID != filter.WorkspaceID {
			continue
		}
		if filter.APIKeyID != "" && (record.APIKeyID == nil || *record.APIKeyID != filter.APIKeyID) {
			continue
		}
		if filter.ProviderID != "" && (record.ProviderID == nil || *record.ProviderID != filter.ProviderID) {
			continue
		}
		if filter.Model != "" && (record.SelectedModel == nil || *record.SelectedModel != filter.Model) {
			continue
		}

		records = append(records, record)
	}

	return records
}

func (h *UsageHandler) calculateSummary(records []UsageRecordResponse) UsageSummary {
	var totalTokens, promptTokens, outputTokens int64
	var totalCost float64
	var totalDuration int64
	var successCount, errorCount int64

	for _, r := range records {
		totalTokens += r.TotalTokens
		promptTokens += r.PromptTokens
		outputTokens += r.CompletionTokens
		if r.CostUSD != nil {
			totalCost += *r.CostUSD
		}
		totalDuration += int64(r.DurationMs)
		if r.ResponseStatus == "success" {
			successCount++
		} else {
			errorCount++
		}
	}

	total := int64(len(records))
	avgDuration := float64(0)
	if total > 0 {
		avgDuration = float64(totalDuration) / float64(total)
	}

	errorRate := float64(0)
	if total > 0 {
		errorRate = float64(errorCount) / float64(total) * 100
	}

	return UsageSummary{
		TotalRequests:     total,
		TotalTokens:       totalTokens,
		TotalPromptTokens: promptTokens,
		TotalOutputTokens: outputTokens,
		TotalCostUSD:      totalCost,
		AvgDurationMs:     avgDuration,
		SuccessCount:      successCount,
		ErrorCount:        errorCount,
		ErrorRate:         errorRate,
	}
}

func (h *UsageHandler) calculateAggregations(
	records []UsageRecordResponse,
	groupBy []string,
	aggregations []string,
) []UsageAggregation {
	// Simple aggregation by dimension
	dimensions := make(map[string]map[string]UsageAggregation)

	for _, record := range records {
		for _, dim := range groupBy {
			var value string
			switch dim {
			case "workspaceId":
				value = record.WorkspaceID
			case "providerId":
				if record.ProviderID != nil {
					value = *record.ProviderID
				}
			case "model":
				if record.SelectedModel != nil {
					value = *record.SelectedModel
				}
			case "api":
				value = record.IncomingAPI
			case "status":
				value = record.ResponseStatus
			}

			if value == "" {
				continue
			}

			if dimensions[dim] == nil {
				dimensions[dim] = make(map[string]UsageAggregation)
			}

			agg := dimensions[dim][value]
			agg.Dimension = dim
			agg.Value = value
			if agg.Metrics == nil {
				agg.Metrics = make(map[string]interface{})
			}

			// Update metrics
			agg.Metrics["requestCount"] = toInt64(agg.Metrics["requestCount"]) + 1
			agg.Metrics["totalTokens"] = toInt64(agg.Metrics["totalTokens"]) + record.TotalTokens
			agg.Metrics["promptTokens"] = toInt64(agg.Metrics["promptTokens"]) + record.PromptTokens
			agg.Metrics["completionTokens"] = toInt64(agg.Metrics["completionTokens"]) + record.CompletionTokens
			if record.CostUSD != nil {
				agg.Metrics["costUsd"] = toFloat64(agg.Metrics["costUsd"]) + *record.CostUSD
			}

			dimensions[dim][value] = agg
		}
	}

	// Convert to slice
	var result []UsageAggregation
	for _, dimMap := range dimensions {
		for _, agg := range dimMap {
			result = append(result, agg)
		}
	}

	return result
}

func (h *UsageHandler) generateTrendPoints(start, end time.Time, interval string) []UsageTrendPoint {
	var points []UsageTrendPoint

	duration := end.Sub(start)
	var step time.Duration
	switch interval {
	case "minute":
		step = time.Minute
	case "day":
		step = 24 * time.Hour
	default:
		step = time.Hour
	}

	numPoints := int(duration / step)
	if numPoints > 168 { // Max 1 week of hourly data
		numPoints = 168
	}

	for i := 0; i < numPoints; i++ {
		timestamp := start.Add(time.Duration(i) * step)
		points = append(points, UsageTrendPoint{
			Timestamp:    timestamp,
			RequestCount: int64(10 + i%20),
			TokenCount:   int64(1000 + i*100),
			CostUSD:      float64(10+i%20) * 0.0001,
			AvgLatencyMs: float64(100 + i%50),
			ErrorCount:   int64(i % 5),
		})
	}

	return points
}

// UsageFilter represents usage filter options
type UsageFilter struct {
	StartTime   *time.Time
	EndTime     *time.Time
	WorkspaceID string
	APIKeyID    string
	ProviderID  string
	Model       string
	Status      string
}

func (h *UsageHandler) parseUsageFilter(r *http.Request) UsageFilter {
	filter := UsageFilter{
		WorkspaceID: r.URL.Query().Get("workspaceId"),
		APIKeyID:    r.URL.Query().Get("apiKeyId"),
		ProviderID:  r.URL.Query().Get("providerId"),
		Model:       r.URL.Query().Get("model"),
		Status:      r.URL.Query().Get("status"),
	}

	if start := r.URL.Query().Get("startTime"); start != "" {
		t, err := time.Parse(time.RFC3339, start)
		if err == nil {
			filter.StartTime = &t
		}
	}
	if end := r.URL.Query().Get("endTime"); end != "" {
		t, err := time.Parse(time.RFC3339, end)
		if err == nil {
			filter.EndTime = &t
		}
	}

	return filter
}

func (h *UsageHandler) parseTimeParam(r *http.Request, name string, fallback time.Time) time.Time {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return fallback
	}
	return t
}

func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case float64:
		return int64(val)
	case int:
		return int64(val)
	default:
		return 0
	}
}

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int64:
		return float64(val)
	case int:
		return float64(val)
	default:
		return 0
	}
}
