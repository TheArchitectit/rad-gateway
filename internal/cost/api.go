package cost

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"log/slog"

	"radgateway/internal/logger"
)

// APIHandler provides HTTP endpoints for cost queries.
type APIHandler struct {
	aggregator *Aggregator
	log        *slog.Logger
}

// NewAPIHandler creates a new cost API handler.
func NewAPIHandler(aggregator *Aggregator) *APIHandler {
	return &APIHandler{
		aggregator: aggregator,
		log:        logger.WithComponent("cost_api"),
	}
}

// Register registers the cost API endpoints.
func (h *APIHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v0/costs/summary", h.getCostSummary)
	mux.HandleFunc("/v0/costs/by-model", h.getCostByModel)
	mux.HandleFunc("/v0/costs/by-provider", h.getCostByProvider)
	mux.HandleFunc("/v0/costs/timeseries", h.getCostTimeseries)
	mux.HandleFunc("/v0/costs/pricing", h.getPricing)
}

// getCostSummary returns the overall cost summary for a workspace.
//
// Query Parameters:
//   - workspace_id (required): The workspace ID
//   - start_time (optional): Start time in RFC3339 format (default: 24 hours ago)
//   - end_time (optional): End time in RFC3339 format (default: now)
//
// Response: CostSummary JSON
func (h *APIHandler) getCostSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.log.Warn("cost api: method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	filter, err := h.parseFilter(r)
	if err != nil {
		h.log.Warn("cost api: invalid filter", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	summary, err := h.aggregator.GetCostSummary(r.Context(), filter)
	if err != nil {
		h.log.Error("cost api: failed to get summary", "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to retrieve cost summary"})
		return
	}

	h.log.Debug("cost api: summary retrieved",
		"workspace_id", filter.WorkspaceID,
		"total_cost", summary.TotalCost,
		"request_count", summary.RequestCount)

	writeJSON(w, http.StatusOK, summary)
}

// getCostByModel returns costs grouped by model.
//
// Query Parameters:
//   - workspace_id (required): The workspace ID
//   - start_time (optional): Start time in RFC3339 format
//   - end_time (optional): End time in RFC3339 format
//   - provider_id (optional): Filter by provider
//   - model_id (optional): Filter by model
//
// Response: []CostByModel JSON
func (h *APIHandler) getCostByModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.log.Warn("cost api: method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	filter, err := h.parseFilter(r)
	if err != nil {
		h.log.Warn("cost api: invalid filter", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	costs, err := h.aggregator.GetCostByModel(r.Context(), filter)
	if err != nil {
		h.log.Error("cost api: failed to get costs by model", "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to retrieve costs"})
		return
	}

	h.log.Debug("cost api: costs by model retrieved",
		"workspace_id", filter.WorkspaceID,
		"count", len(costs))

	writeJSON(w, http.StatusOK, costs)
}

// getCostByProvider returns costs grouped by provider.
//
// Query Parameters:
//   - workspace_id (required): The workspace ID
//   - start_time (optional): Start time in RFC3339 format
//   - end_time (optional): End time in RFC3339 format
//
// Response: []CostByProvider JSON
func (h *APIHandler) getCostByProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.log.Warn("cost api: method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	filter, err := h.parseFilter(r)
	if err != nil {
		h.log.Warn("cost api: invalid filter", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	costs, err := h.aggregator.GetCostByProvider(r.Context(), filter)
	if err != nil {
		h.log.Error("cost api: failed to get costs by provider", "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to retrieve costs"})
		return
	}

	h.log.Debug("cost api: costs by provider retrieved",
		"workspace_id", filter.WorkspaceID,
		"count", len(costs))

	writeJSON(w, http.StatusOK, costs)
}

// getCostTimeseries returns cost data over time.
//
// Query Parameters:
//   - workspace_id (required): The workspace ID
//   - start_time (optional): Start time in RFC3339 format
//   - end_time (optional): End time in RFC3339 format
//   - granularity (optional): hourly, daily, weekly, monthly (default: daily)
//
// Response: []CostTimeseriesPoint JSON
func (h *APIHandler) getCostTimeseries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.log.Warn("cost api: method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	filter, err := h.parseFilter(r)
	if err != nil {
		h.log.Warn("cost api: invalid filter", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	timeseries, err := h.aggregator.GetCostTimeseries(r.Context(), filter)
	if err != nil {
		h.log.Error("cost api: failed to get timeseries", "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to retrieve timeseries"})
		return
	}

	h.log.Debug("cost api: timeseries retrieved",
		"workspace_id", filter.WorkspaceID,
		"points", len(timeseries))

	writeJSON(w, http.StatusOK, timeseries)
}

// getPricing returns the current pricing information for known models.
//
// Response: map of model ID to TokenRate
func (h *APIHandler) getPricing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.log.Warn("cost api: method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Get the calculator from aggregator
	// We can't directly access it, so we'll return default pricing
	pricing := make(map[string]TokenRate)
	for model, rate := range DefaultPricing {
		pricing[model] = rate
	}

	h.log.Debug("cost api: pricing retrieved", "model_count", len(pricing))

	writeJSON(w, http.StatusOK, map[string]any{
		"models":   pricing,
		"currency": DefaultCurrency,
		"updated":  time.Now(),
	})
}

// parseFilter parses query parameters into a QueryFilter.
func (h *APIHandler) parseFilter(r *http.Request) (QueryFilter, error) {
	filter := QueryFilter{
		Currency: DefaultCurrency,
	}

	// Required: workspace_id
	workspaceID := r.URL.Query().Get("workspace_id")
	if workspaceID == "" {
		return filter, errMissingParam("workspace_id")
	}
	filter.WorkspaceID = workspaceID

	// Optional: start_time (default: 24 hours ago)
	startTimeStr := r.URL.Query().Get("start_time")
	if startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return filter, errInvalidParam("start_time", err.Error())
		}
		filter.StartTime = startTime
	} else {
		filter.StartTime = time.Now().Add(-24 * time.Hour)
	}

	// Optional: end_time (default: now)
	endTimeStr := r.URL.Query().Get("end_time")
	if endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return filter, errInvalidParam("end_time", err.Error())
		}
		filter.EndTime = endTime
	} else {
		filter.EndTime = time.Now()
	}

	// Validate time range
	if filter.EndTime.Before(filter.StartTime) {
		return filter, errInvalidParam("time_range", "end_time must be after start_time")
	}

	// Optional: granularity/aggregation level
	granularity := r.URL.Query().Get("granularity")
	if granularity == "" {
		granularity = r.URL.Query().Get("aggregation")
	}
	switch granularity {
	case "hourly":
		filter.AggregationLevel = AggHourly
	case "daily", "":
		filter.AggregationLevel = AggDaily
	case "weekly":
		filter.AggregationLevel = AggWeekly
	case "monthly":
		filter.AggregationLevel = AggMonthly
	default:
		return filter, errInvalidParam("granularity", "must be one of: hourly, daily, weekly, monthly")
	}

	// Optional: provider_id
	if providerID := r.URL.Query().Get("provider_id"); providerID != "" {
		filter.ProviderID = &providerID
	}

	// Optional: model_id
	if modelID := r.URL.Query().Get("model_id"); modelID != "" {
		filter.ModelID = &modelID
	}

	// Optional: api_key_id
	if apiKeyID := r.URL.Query().Get("api_key_id"); apiKeyID != "" {
		filter.APIKeyID = &apiKeyID
	}

	// Optional: control_room_id
	if controlRoomID := r.URL.Query().Get("control_room_id"); controlRoomID != "" {
		filter.ControlRoomID = &controlRoomID
	}

	return filter, nil
}

// errMissingParam creates an error for a missing required parameter.
func errMissingParam(name string) error {
	return &paramError{param: name, msg: "missing required parameter"}
}

// errInvalidParam creates an error for an invalid parameter value.
func errInvalidParam(name, reason string) error {
	return &paramError{param: name, msg: reason}
}

// paramError represents a parameter validation error.
type paramError struct {
	param string
	msg   string
}

func (e *paramError) Error() string {
	return e.param + ": " + e.msg
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// parseTimeRange is a convenience function for parsing time range parameters.
// Returns start time, end time, and any error.
func parseTimeRange(r *http.Request) (time.Time, time.Time, error) {
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")

	var startTime, endTime time.Time
	var err error

	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return startTime, endTime, errInvalidParam("start_time", err.Error())
		}
	} else {
		startTime = time.Now().Add(-24 * time.Hour)
	}

	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return startTime, endTime, errInvalidParam("end_time", err.Error())
		}
	} else {
		endTime = time.Now()
	}

	if endTime.Before(startTime) {
		return startTime, endTime, errInvalidParam("time_range", "end_time must be after start_time")
	}

	return startTime, endTime, nil
}

// parseLimit parses the limit parameter with a default value.
func parseLimit(r *http.Request, defaultLimit, maxLimit int) int {
	str := r.URL.Query().Get("limit")
	if str == "" {
		return defaultLimit
	}

	n, err := strconv.Atoi(str)
	if err != nil || n <= 0 {
		return defaultLimit
	}

	if n > maxLimit {
		return maxLimit
	}

	return n
}

// APIResponse represents a standard API response structure.
type APIResponse struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
	Meta  *Meta  `json:"meta,omitempty"`
}

// Meta contains response metadata.
type Meta struct {
	Total      int       `json:"total,omitempty"`
	Page       int       `json:"page,omitempty"`
	PerPage    int       `json:"per_page,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	RequestID  string    `json:"request_id,omitempty"`
	WorkspaceID string   `json:"workspace_id,omitempty"`
}

// NewSuccessResponse creates a successful API response.
func NewSuccessResponse(data any) APIResponse {
	return APIResponse{
		Data: data,
		Meta: &Meta{
			Timestamp: time.Now(),
		},
	}
}

// NewErrorResponse creates an error API response.
func NewErrorResponse(message string) APIResponse {
	return APIResponse{
		Error: message,
		Meta: &Meta{
			Timestamp: time.Now(),
		},
	}
}

// RespondWithJSON sends a JSON response with the given status code.
func RespondWithJSON(w http.ResponseWriter, code int, resp APIResponse) {
	writeJSON(w, code, resp)
}
