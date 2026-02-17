package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"log/slog"

	"radgateway/internal/logger"
)

// CostSummaryResponse represents a cost summary response
type CostSummaryResponse struct {
	TimeRange           TimeRange            `json:"timeRange"`
	TotalCostUSD        float64              `json:"totalCostUsd"`
	TotalTokens         int64                `json:"totalTokens"`
	TotalRequests       int64                `json:"totalRequests"`
	AvgCostPerRequest   float64              `json:"avgCostPerRequest"`
	AvgCostPer1KTokens  float64              `json:"avgCostPer1KTokens"`
	BreakdownByProvider []ProviderCostBreakdown `json:"breakdownByProvider"`
	BreakdownByModel    []ModelCostBreakdown    `json:"breakdownByModel"`
	BreakdownByWorkspace []WorkspaceCostBreakdown `json:"breakdownByWorkspace"`
	DailyBreakdown      []DailyCostBreakdown    `json:"dailyBreakdown"`
}

// ProviderCostBreakdown represents cost breakdown by provider
type ProviderCostBreakdown struct {
	ProviderID     string  `json:"providerId"`
	ProviderName   string  `json:"providerName"`
	CostUSD        float64 `json:"costUsd"`
	TokenCount     int64   `json:"tokenCount"`
	RequestCount   int64   `json:"requestCount"`
	Percentage     float64 `json:"percentage"`
}

// ModelCostBreakdown represents cost breakdown by model
type ModelCostBreakdown struct {
	ModelID      string  `json:"modelId"`
	ModelName    string  `json:"modelName"`
	ProviderID   string  `json:"providerId"`
	CostUSD      float64 `json:"costUsd"`
	TokenCount   int64   `json:"tokenCount"`
	RequestCount int64   `json:"requestCount"`
	Percentage   float64 `json:"percentage"`
}

// WorkspaceCostBreakdown represents cost breakdown by workspace
type WorkspaceCostBreakdown struct {
	WorkspaceID   string  `json:"workspaceId"`
	WorkspaceName string  `json:"workspaceName"`
	CostUSD       float64 `json:"costUsd"`
	TokenCount    int64   `json:"tokenCount"`
	RequestCount  int64   `json:"requestCount"`
	Percentage    float64 `json:"percentage"`
}

// DailyCostBreakdown represents daily cost breakdown
type DailyCostBreakdown struct {
	Date         string                `json:"date"`
	CostUSD      float64               `json:"costUsd"`
	TokenCount   int64                 `json:"tokenCount"`
	RequestCount int64                 `json:"requestCount"`
	ByProvider   []ProviderCostSummary `json:"byProvider"`
}

// ProviderCostSummary represents provider cost summary
type ProviderCostSummary struct {
	ProviderID string  `json:"providerId"`
	CostUSD    float64 `json:"costUsd"`
}

// CostTrendResponse represents cost trend data
type CostTrendResponse struct {
	TimeRange   TimeRange    `json:"timeRange"`
	Interval    string       `json:"interval"`
	TrendPoints []CostTrendPoint `json:"trendPoints"`
	Forecast    *CostForecast    `json:"forecast,omitempty"`
}

// CostTrendPoint represents a single cost trend point
type CostTrendPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	CostUSD      float64   `json:"costUsd"`
	TokenCount   int64     `json:"tokenCount"`
	RequestCount int64     `json:"requestCount"`
	CostChange   float64   `json:"costChange"` // Percentage change from previous
}

// CostForecast represents a cost forecast
type CostForecast struct {
	ForecastPeriod    TimeRange `json:"forecastPeriod"`
	PredictedCostUSD  float64   `json:"predictedCostUsd"`
	ConfidenceLow     float64   `json:"confidenceLow"`
	ConfidenceHigh    float64   `json:"confidenceHigh"`
	TrendDirection    string    `json:"trendDirection"` // increasing, decreasing, stable
	AvgDailyCost      float64   `json:"avgDailyCost"`
}

// CostAlert represents a cost alert
type CostAlert struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"` // threshold, anomaly, budget
	Severity    string     `json:"severity"` // info, warning, critical
	Message     string     `json:"message"`
	Threshold   float64    `json:"threshold"`
	CurrentValue float64   `json:"currentValue"`
	TriggeredAt time.Time  `json:"triggeredAt"`
	ResolvedAt  *time.Time `json:"resolvedAt,omitempty"`
	Acknowledged bool      `json:"acknowledged"`
}

// CostAlertListResponse represents cost alert list response
type CostAlertListResponse struct {
	Data       []CostAlert `json:"data"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	HasMore    bool        `json:"hasMore"`
}

// CostAlertCreateRequest represents a request to create a cost alert
type CostAlertCreateRequest struct {
	Type      string  `json:"type"`
	Severity  string  `json:"severity"`
	Name      string  `json:"name"`
	Condition string  `json:"condition"` // e.g., "cost > 1000" or "cost_change > 50%"
	Threshold float64 `json:"threshold"`
	Window    string  `json:"window"`    // e.g., "1d", "7d", "30d"
}

// BudgetResponse represents a budget response
type BudgetResponse struct {
	ID              string    `json:"id"`
	WorkspaceID     string    `json:"workspaceId"`
	Name            string    `json:"name"`
	AmountUSD       float64   `json:"amountUsd"`
	Period          string    `json:"period"` // daily, monthly, yearly
	StartDate       time.Time `json:"startDate"`
	EndDate         *time.Time `json:"endDate,omitempty"`
	CurrentSpendUSD float64   `json:"currentSpendUsd"`
	RemainingUSD    float64   `json:"remainingUsd"`
	PercentageUsed  float64   `json:"percentageUsed"`
	Status          string    `json:"status"` // active, exceeded, warning
	AlertThresholds []float64 `json:"alertThresholds"` // Percentage thresholds for alerts
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// BudgetCreateRequest represents a request to create a budget
type BudgetCreateRequest struct {
	WorkspaceID     string    `json:"workspaceId"`
	Name            string    `json:"name"`
	AmountUSD       float64   `json:"amountUsd"`
	Period          string    `json:"period"`
	StartDate       time.Time `json:"startDate"`
	EndDate         *time.Time `json:"endDate,omitempty"`
	AlertThresholds []float64 `json:"alertThresholds"`
}

// CostHandler handles cost tracking endpoints
type CostHandler struct {
	log *slog.Logger
}

// NewCostHandler creates a new cost handler
func NewCostHandler() *CostHandler {
	return &CostHandler{
		log: logger.WithComponent("admin.costs"),
	}
}

// RegisterRoutes registers the cost tracking routes
func (h *CostHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v0/admin/costs", h.handleCosts)
	mux.HandleFunc("/v0/admin/costs/summary", h.handleCostSummary)
	mux.HandleFunc("/v0/admin/costs/trends", h.handleCostTrends)
	mux.HandleFunc("/v0/admin/costs/forecast", h.handleCostForecast)
	mux.HandleFunc("/v0/admin/costs/alerts", h.handleCostAlerts)
	mux.HandleFunc("/v0/admin/costs/alerts/", h.handleCostAlertDetail)
	mux.HandleFunc("/v0/admin/costs/budgets", h.handleBudgets)
	mux.HandleFunc("/v0/admin/costs/budgets/", h.handleBudgetDetail)
}

func (h *CostHandler) handleCosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	h.getCostSummary(w, r)
}

func (h *CostHandler) handleCostSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	h.getCostSummary(w, r)
}

func (h *CostHandler) handleCostTrends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	h.getCostTrends(w, r)
}

func (h *CostHandler) handleCostForecast(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getCostForecast(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *CostHandler) handleCostAlerts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listCostAlerts(w, r)
	case http.MethodPost:
		h.createCostAlert(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *CostHandler) handleCostAlertDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v0/admin/costs/alerts/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "alert id required"})
		return
	}

	// Check for actions
	if strings.HasSuffix(id, "/acknowledge") {
		h.acknowledgeCostAlert(w, r, strings.TrimSuffix(id, "/acknowledge"))
		return
	}
	if strings.HasSuffix(id, "/resolve") {
		h.resolveCostAlert(w, r, strings.TrimSuffix(id, "/resolve"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getCostAlert(w, r, id)
	case http.MethodDelete:
		h.deleteCostAlert(w, r, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *CostHandler) handleBudgets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listBudgets(w, r)
	case http.MethodPost:
		h.createBudget(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *CostHandler) handleBudgetDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v0/admin/costs/budgets/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "budget id required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getBudget(w, r, id)
	case http.MethodPut:
		h.updateBudget(w, r, id)
	case http.MethodDelete:
		h.deleteBudget(w, r, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// getCostSummary returns cost summary with breakdowns
func (h *CostHandler) getCostSummary(w http.ResponseWriter, r *http.Request) {
	startTime := h.parseTimeParam(r, "startTime", time.Now().Add(-30*24*time.Hour))
	endTime := h.parseTimeParam(r, "endTime", time.Now())
	workspaceID := r.URL.Query().Get("workspaceId")

	h.log.Debug("getting cost summary",
		"startTime", startTime,
		"endTime", endTime,
		"workspace", workspaceID,
	)

	// Generate mock cost summary
	totalCost := 1523.45
	totalTokens := int64(152345000)
	totalRequests := int64(30469)

	response := CostSummaryResponse{
		TimeRange: TimeRange{Start: startTime, End: endTime},
		TotalCostUSD:       totalCost,
		TotalTokens:        totalTokens,
		TotalRequests:      totalRequests,
		AvgCostPerRequest:  totalCost / float64(totalRequests),
		AvgCostPer1KTokens: (totalCost / float64(totalTokens)) * 1000,
		BreakdownByProvider: []ProviderCostBreakdown{
			{
				ProviderID:   "prov_001",
				ProviderName: "OpenAI",
				CostUSD:      823.45,
				TokenCount:   82345000,
				RequestCount: 16469,
				Percentage:   54.1,
			},
			{
				ProviderID:   "prov_002",
				ProviderName: "Anthropic",
				CostUSD:      450.00,
				TokenCount:   45000000,
				RequestCount: 9000,
				Percentage:   29.5,
			},
			{
				ProviderID:   "prov_003",
				ProviderName: "Google",
				CostUSD:      250.00,
				TokenCount:   25000000,
				RequestCount: 5000,
				Percentage:   16.4,
			},
		},
		BreakdownByModel: []ModelCostBreakdown{
			{
				ModelID:      "gpt-4o-mini",
				ModelName:    "GPT-4o Mini",
				ProviderID:   "prov_001",
				CostUSD:      523.45,
				TokenCount:   52345000,
				RequestCount: 10469,
				Percentage:   34.4,
			},
			{
				ModelID:      "claude-3-5-sonnet",
				ModelName:    "Claude 3.5 Sonnet",
				ProviderID:   "prov_002",
				CostUSD:      450.00,
				TokenCount:   45000000,
				RequestCount: 9000,
				Percentage:   29.5,
			},
			{
				ModelID:      "gemini-1.5-flash",
				ModelName:    "Gemini 1.5 Flash",
				ProviderID:   "prov_003",
				CostUSD:      250.00,
				TokenCount:   25000000,
				RequestCount: 5000,
				Percentage:   16.4,
			},
			{
				ModelID:      "gpt-4o",
				ModelName:    "GPT-4o",
				ProviderID:   "prov_001",
				CostUSD:      300.00,
				TokenCount:   30000000,
				RequestCount: 6000,
				Percentage:   19.7,
			},
		},
		BreakdownByWorkspace: []WorkspaceCostBreakdown{
			{
				WorkspaceID:   "ws_001",
				WorkspaceName: "Production",
				CostUSD:       1000.00,
				TokenCount:    100000000,
				RequestCount:  20000,
				Percentage:    65.7,
			},
			{
				WorkspaceID:   "ws_002",
				WorkspaceName: "Staging",
				CostUSD:       323.45,
				TokenCount:    32345000,
				RequestCount:   6469,
				Percentage:    21.2,
			},
			{
				WorkspaceID:   "ws_003",
				WorkspaceName: "Development",
				CostUSD:       200.00,
				TokenCount:    20000000,
				RequestCount:  4000,
				Percentage:    13.1,
			},
		},
		DailyBreakdown: h.generateDailyBreakdown(startTime, endTime),
	}

	writeJSON(w, http.StatusOK, response)
}

// getCostTrends returns cost trend data
func (h *CostHandler) getCostTrends(w http.ResponseWriter, r *http.Request) {
	startTime := h.parseTimeParam(r, "startTime", time.Now().Add(-7*24*time.Hour))
	endTime := h.parseTimeParam(r, "endTime", time.Now())
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "day"
	}

	h.log.Debug("getting cost trends",
		"startTime", startTime,
		"endTime", endTime,
		"interval", interval,
	)

	trendPoints := h.generateCostTrendPoints(startTime, endTime, interval)

	response := CostTrendResponse{
		TimeRange:   TimeRange{Start: startTime, End: endTime},
		Interval:    interval,
		TrendPoints: trendPoints,
	}

	writeJSON(w, http.StatusOK, response)
}

// getCostForecast returns cost forecast
func (h *CostHandler) getCostForecast(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	h.log.Debug("getting cost forecast", "days", days)

	// Calculate forecast based on historical data
	forecastStart := time.Now()
	forecastEnd := forecastStart.Add(time.Duration(days) * 24 * time.Hour)

	// Mock forecast calculation
	historicalDailyAvg := 50.78
	predictedCost := historicalDailyAvg * float64(days)
	confidenceLow := predictedCost * 0.85
	confidenceHigh := predictedCost * 1.15

	forecast := CostForecast{
		ForecastPeriod:   TimeRange{Start: forecastStart, End: forecastEnd},
		PredictedCostUSD: predictedCost,
		ConfidenceLow:    confidenceLow,
		ConfidenceHigh:   confidenceHigh,
		TrendDirection:   "increasing",
		AvgDailyCost:     historicalDailyAvg,
	}

	response := CostTrendResponse{
		TimeRange: TimeRange{Start: forecastStart, End: forecastEnd},
		Interval:  "day",
		Forecast:  &forecast,
	}

	writeJSON(w, http.StatusOK, response)
}

// listCostAlerts returns cost alerts
func (h *CostHandler) listCostAlerts(w http.ResponseWriter, r *http.Request) {
	page := readIntParam(r, "page", 1)
	pageSize := readIntParam(r, "pageSize", 50)
	severity := r.URL.Query().Get("severity")
	acknowledged := r.URL.Query().Get("acknowledged")

	h.log.Debug("listing cost alerts", "severity", severity, "acknowledged", acknowledged)

	// Mock alerts
	alerts := []CostAlert{
		{
			ID:             "alert_001",
			Type:           "threshold",
			Severity:       "warning",
			Message:        "Daily cost threshold exceeded",
			Threshold:      100.0,
			CurrentValue:   125.50,
			TriggeredAt:    time.Now().Add(-2 * time.Hour),
			Acknowledged:   false,
		},
		{
			ID:             "alert_002",
			Type:           "budget",
			Severity:       "critical",
			Message:        "Monthly budget 80% consumed",
			Threshold:      80.0,
			CurrentValue:   82.3,
			TriggeredAt:    time.Now().Add(-24 * time.Hour),
			Acknowledged:   true,
		},
		{
			ID:             "alert_003",
			Type:           "anomaly",
			Severity:       "info",
			Message:        "Unusual cost spike detected",
			Threshold:      50.0,
			CurrentValue:   75.0,
			TriggeredAt:    time.Now().Add(-5 * time.Hour),
			Acknowledged:   false,
		},
	}

	// Apply filters
	var filtered []CostAlert
	for _, alert := range alerts {
		if severity != "" && alert.Severity != severity {
			continue
		}
		if acknowledged != "" {
			ack := acknowledged == "true"
			if alert.Acknowledged != ack {
				continue
			}
		}
		filtered = append(filtered, alert)
	}

	total := len(filtered)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	response := CostAlertListResponse{
		Data:     filtered[start:end],
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  end < total,
	}

	writeJSON(w, http.StatusOK, response)
}

// getCostAlert returns a single cost alert
func (h *CostHandler) getCostAlert(w http.ResponseWriter, r *http.Request, id string) {
	h.log.Debug("getting cost alert", "id", id)

	alert := CostAlert{
		ID:             id,
		Type:           "threshold",
		Severity:       "warning",
		Message:        "Daily cost threshold exceeded",
		Threshold:      100.0,
		CurrentValue:   125.50,
		TriggeredAt:    time.Now().Add(-2 * time.Hour),
		Acknowledged:   false,
	}

	writeJSON(w, http.StatusOK, alert)
}

// createCostAlert creates a new cost alert
func (h *CostHandler) createCostAlert(w http.ResponseWriter, r *http.Request) {
	var req CostAlertCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	h.log.Info("creating cost alert",
		"type", req.Type,
		"severity", req.Severity,
		"name", req.Name,
	)

	alert := CostAlert{
		ID:             generateID("alert"),
		Type:           req.Type,
		Severity:       req.Severity,
		Message:        req.Name + " - threshold: " + req.Condition,
		Threshold:      req.Threshold,
		CurrentValue:   0,
		TriggeredAt:    time.Now(),
		Acknowledged:   false,
	}

	writeJSON(w, http.StatusCreated, alert)
}

// acknowledgeCostAlert acknowledges a cost alert
func (h *CostHandler) acknowledgeCostAlert(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	h.log.Info("acknowledging cost alert", "id", id)

	alert := CostAlert{
		ID:             id,
		Type:           "threshold",
		Severity:       "warning",
		Message:        "Daily cost threshold exceeded",
		Acknowledged:   true,
		TriggeredAt:    time.Now().Add(-2 * time.Hour),
	}

	writeJSON(w, http.StatusOK, alert)
}

// resolveCostAlert resolves a cost alert
func (h *CostHandler) resolveCostAlert(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	h.log.Info("resolving cost alert", "id", id)

	now := time.Now()
	alert := CostAlert{
		ID:             id,
		Type:           "threshold",
		Severity:       "warning",
		Message:        "Daily cost threshold exceeded",
		Acknowledged:   true,
		ResolvedAt:     &now,
		TriggeredAt:    time.Now().Add(-2 * time.Hour),
	}

	writeJSON(w, http.StatusOK, alert)
}

// deleteCostAlert deletes a cost alert
func (h *CostHandler) deleteCostAlert(w http.ResponseWriter, r *http.Request, id string) {
	h.log.Info("deleting cost alert", "id", id)
	writeJSON(w, http.StatusNoContent, nil)
}

// listBudgets returns cost budgets
func (h *CostHandler) listBudgets(w http.ResponseWriter, r *http.Request) {
	page := readIntParam(r, "page", 1)
	pageSize := readIntParam(r, "pageSize", 50)
	workspaceID := r.URL.Query().Get("workspaceId")

	h.log.Debug("listing budgets", "workspace", workspaceID)

	// Mock budgets
	budgets := []BudgetResponse{
		{
			ID:              "budget_001",
			WorkspaceID:     "ws_001",
			Name:            "Production Monthly Budget",
			AmountUSD:       5000.00,
			Period:          "monthly",
			StartDate:       time.Now().AddDate(0, 0, -15),
			CurrentSpendUSD: 1523.45,
			RemainingUSD:    3476.55,
			PercentageUsed:    30.5,
			Status:            "active",
			AlertThresholds:   []float64{50, 75, 90, 100},
			CreatedAt:         time.Now().AddDate(0, -1, 0),
			UpdatedAt:         time.Now(),
		},
		{
			ID:              "budget_002",
			WorkspaceID:     "ws_002",
			Name:            "Staging Monthly Budget",
			AmountUSD:       1000.00,
			Period:          "monthly",
			StartDate:       time.Now().AddDate(0, 0, -15),
			CurrentSpendUSD: 323.45,
			RemainingUSD:    676.55,
			PercentageUsed:  32.3,
			Status:          "active",
			AlertThresholds: []float64{75, 90, 100},
			CreatedAt:       time.Now().AddDate(0, -1, 0),
			UpdatedAt:       time.Now(),
		},
	}

	// Apply workspace filter
	var filtered []BudgetResponse
	if workspaceID != "" {
		for _, b := range budgets {
			if b.WorkspaceID == workspaceID {
				filtered = append(filtered, b)
			}
		}
	} else {
		filtered = budgets
	}

	total := len(filtered)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	response := map[string]interface{}{
		"data":     filtered[start:end],
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
		"hasMore":  end < total,
	}

	writeJSON(w, http.StatusOK, response)
}

// getBudget returns a single budget
func (h *CostHandler) getBudget(w http.ResponseWriter, r *http.Request, id string) {
	h.log.Debug("getting budget", "id", id)

	budget := BudgetResponse{
		ID:              id,
		WorkspaceID:     "ws_001",
		Name:            "Production Monthly Budget",
		AmountUSD:       5000.00,
		Period:          "monthly",
		StartDate:       time.Now().AddDate(0, 0, -15),
		CurrentSpendUSD: 1523.45,
		RemainingUSD:    3476.55,
		PercentageUsed:  30.5,
		Status:          "active",
		AlertThresholds: []float64{50, 75, 90, 100},
		CreatedAt:       time.Now().AddDate(0, -1, 0),
		UpdatedAt:       time.Now(),
	}

	writeJSON(w, http.StatusOK, budget)
}

// createBudget creates a new budget
func (h *CostHandler) createBudget(w http.ResponseWriter, r *http.Request) {
	var req BudgetCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if req.AmountUSD <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "amountUsd must be positive"})
		return
	}

	h.log.Info("creating budget",
		"name", req.Name,
		"amount", req.AmountUSD,
		"period", req.Period,
	)

	budget := BudgetResponse{
		ID:              generateID("budget"),
		WorkspaceID:     req.WorkspaceID,
		Name:            req.Name,
		AmountUSD:       req.AmountUSD,
		Period:          req.Period,
		StartDate:       req.StartDate,
		EndDate:         req.EndDate,
		CurrentSpendUSD: 0,
		RemainingUSD:    req.AmountUSD,
		PercentageUsed:  0,
		Status:          "active",
		AlertThresholds: req.AlertThresholds,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	writeJSON(w, http.StatusCreated, budget)
}

// updateBudget updates a budget
func (h *CostHandler) updateBudget(w http.ResponseWriter, r *http.Request, id string) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	h.log.Info("updating budget", "id", id)

	budget := BudgetResponse{
		ID:              id,
		WorkspaceID:     "ws_001",
		Name:            "Production Monthly Budget",
		AmountUSD:       5000.00,
		Period:          "monthly",
		StartDate:       time.Now().AddDate(0, 0, -15),
		CurrentSpendUSD: 1523.45,
		RemainingUSD:    3476.55,
		PercentageUsed:  30.5,
		Status:          "active",
		AlertThresholds: []float64{50, 75, 90, 100},
		UpdatedAt:       time.Now(),
	}

	writeJSON(w, http.StatusOK, budget)
}

// deleteBudget deletes a budget
func (h *CostHandler) deleteBudget(w http.ResponseWriter, r *http.Request, id string) {
	h.log.Info("deleting budget", "id", id)
	writeJSON(w, http.StatusNoContent, nil)
}

// Helper methods

func (h *CostHandler) parseTimeParam(r *http.Request, name string, fallback time.Time) time.Time {
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

func (h *CostHandler) generateDailyBreakdown(start, end time.Time) []DailyCostBreakdown {
	var breakdowns []DailyCostBreakdown

	current := start
	for current.Before(end) || current.Equal(end) {
		dateStr := current.Format("2006-01-02")
		breakdowns = append(breakdowns, DailyCostBreakdown{
			Date:         dateStr,
			CostUSD:      50.78,
			TokenCount:   5078000,
			RequestCount: 1015,
			ByProvider: []ProviderCostSummary{
				{ProviderID: "prov_001", CostUSD: 27.45},
				{ProviderID: "prov_002", CostUSD: 15.00},
				{ProviderID: "prov_003", CostUSD: 8.33},
			},
		})
		current = current.Add(24 * time.Hour)
	}

	return breakdowns
}

func (h *CostHandler) generateCostTrendPoints(start, end time.Time, interval string) []CostTrendPoint {
	var points []CostTrendPoint

	duration := end.Sub(start)
	var step time.Duration
	switch interval {
	case "hour":
		step = time.Hour
	case "day":
		step = 24 * time.Hour
	default:
		step = 24 * time.Hour
	}

	numPoints := int(duration / step)
	if numPoints > 90 { // Max 90 data points
		numPoints = 90
	}

	var prevCost float64 = 45.0
	for i := 0; i < numPoints; i++ {
		timestamp := start.Add(time.Duration(i) * step)
		cost := 45.0 + float64(i%10)*0.5
		change := float64(0)
		if prevCost > 0 {
			change = ((cost - prevCost) / prevCost) * 100
		}
		prevCost = cost

		points = append(points, CostTrendPoint{
			Timestamp:    timestamp,
			CostUSD:      cost,
			TokenCount:   int64(cost * 100000),
			RequestCount: int64(cost * 20),
			CostChange:   change,
		})
	}

	return points
}
