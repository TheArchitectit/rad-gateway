package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"radgateway/internal/logger"
)

// ProviderCreateRequest represents a request to create a provider
type ProviderCreateRequest struct {
	Name         string                 `json:"name"`
	Slug         string                 `json:"slug"`
	WorkspaceID  string                 `json:"workspaceId"`
	ProviderType string                 `json:"providerType"` // openai, anthropic, gemini
	BaseURL      string                 `json:"baseUrl"`
	APIKey       string                 `json:"apiKey"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Priority     int                    `json:"priority"`
	Weight       int                    `json:"weight"`
}

// ProviderUpdateRequest represents a request to update a provider
type ProviderUpdateRequest struct {
	Name         string                 `json:"name,omitempty"`
	BaseURL      string                 `json:"baseUrl,omitempty"`
	APIKey       string                 `json:"apiKey,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Status       string                 `json:"status,omitempty"`
	Priority     *int                   `json:"priority,omitempty"`
	Weight       *int                   `json:"weight,omitempty"`
}

// ProviderResponse represents a provider in API responses
type ProviderResponse struct {
	ID           string                 `json:"id"`
	WorkspaceID  string                 `json:"workspaceId"`
	Slug         string                 `json:"slug"`
	Name         string                 `json:"name"`
	ProviderType string                 `json:"providerType"`
	BaseURL      string                 `json:"baseUrl"`
	Config       map[string]interface{} `json:"config"`
	Status       string                 `json:"status"`
	Priority     int                    `json:"priority"`
	Weight       int                    `json:"weight"`
	Health       *ProviderHealthStatus  `json:"health,omitempty"`
	CircuitState *CircuitBreakerState   `json:"circuitState,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
}

// ProviderHealthStatus represents provider health status
type ProviderHealthStatus struct {
	Healthy             bool       `json:"healthy"`
	LastCheckAt         time.Time  `json:"lastCheckAt"`
	LastSuccessAt       *time.Time `json:"lastSuccessAt,omitempty"`
	ConsecutiveFailures int        `json:"consecutiveFailures"`
	LatencyMs           *int       `json:"latencyMs,omitempty"`
	ErrorMessage        *string    `json:"errorMessage,omitempty"`
}

// CircuitBreakerState represents circuit breaker state
type CircuitBreakerState struct {
	State            string     `json:"state"` // closed, open, half-open
	Failures         int        `json:"failures"`
	Successes        int        `json:"successes"`
	LastFailureAt    *time.Time `json:"lastFailureAt,omitempty"`
	HalfOpenRequests int        `json:"halfOpenRequests"`
	OpenedAt         *time.Time `json:"openedAt,omitempty"`
}

// ProviderListResponse represents the list response
type ProviderListResponse struct {
	Data     []ProviderResponse `json:"data"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
	HasMore  bool               `json:"hasMore"`
}

// ProviderHealthCheckRequest represents a request to trigger health check
type ProviderHealthCheckRequest struct {
	ProviderID string `json:"providerId"`
}

// ProviderHealthCheckResponse represents health check response
type ProviderHealthCheckResponse struct {
	ProviderID string            `json:"providerId"`
	Healthy    bool              `json:"healthy"`
	LatencyMs  int               `json:"latencyMs"`
	CheckedAt  time.Time         `json:"checkedAt"`
	Details    map[string]string `json:"details,omitempty"`
}

// CircuitBreakerControlRequest represents a request to control circuit breaker
type CircuitBreakerControlRequest struct {
	Action string `json:"action"` // open, close, reset
}

// CircuitBreakerControlResponse represents circuit breaker control response
type CircuitBreakerControlResponse struct {
	ProviderID string              `json:"providerId"`
	Action     string              `json:"action"`
	NewState   string              `json:"newState"`
	AppliedAt  time.Time           `json:"appliedAt"`
}

// BulkProviderRequest represents a bulk operation request
type BulkProviderRequest struct {
	IDs    []string `json:"ids"`
	Action string   `json:"action"`
}

// ProviderMetricsResponse represents provider metrics
type ProviderMetricsResponse struct {
	ProviderID    string    `json:"providerId"`
	ProviderName  string    `json:"providerName"`
	TimeRange     TimeRange `json:"timeRange"`
	RequestCount  int64     `json:"requestCount"`
	SuccessCount  int64     `json:"successCount"`
	ErrorCount    int64     `json:"errorCount"`
	ErrorRate     float64   `json:"errorRate"`
	AvgLatencyMs  float64   `json:"avgLatencyMs"`
	TotalTokens   int64     `json:"totalTokens"`
	TotalCostUSD  float64   `json:"totalCostUsd"`
	P50LatencyMs  float64   `json:"p50LatencyMs"`
	P95LatencyMs  float64   `json:"p95LatencyMs"`
	P99LatencyMs  float64   `json:"p99LatencyMs"`
}

// ProviderHandler handles provider management endpoints
type ProviderHandler struct {
	log *slog.Logger
}

// NewProviderHandler creates a new provider handler
func NewProviderHandler() *ProviderHandler {
	return &ProviderHandler{
		log: logger.WithComponent("admin.providers"),
	}
}

// RegisterRoutes registers the provider management routes
func (h *ProviderHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v0/admin/providers", h.handleProviders)
	mux.HandleFunc("/v0/admin/providers/", h.handleProviderDetail)
	mux.HandleFunc("/v0/admin/providers/bulk", h.handleBulkOperation)
	mux.HandleFunc("/v0/admin/providers/health", h.handleHealthCheck)
	mux.HandleFunc("/v0/admin/providers/circuit", h.handleCircuitControl)
	mux.HandleFunc("/v0/admin/providers/metrics", h.handleProviderMetrics)
}

func (h *ProviderHandler) handleProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listProviders(w, r)
	case http.MethodPost:
		h.createProvider(w, r)
	default:
		h.log.Warn("method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *ProviderHandler) handleProviderDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v0/admin/providers/")
	if id == "" || id == "bulk" || id == "health" || id == "circuit" || id == "metrics" {
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getProvider(w, r, id)
	case http.MethodPut:
		h.updateProvider(w, r, id)
	case http.MethodDelete:
		h.deleteProvider(w, r, id)
	case http.MethodPatch:
		h.patchProvider(w, r, id)
	default:
		h.log.Warn("method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *ProviderHandler) handleBulkOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	h.handleBulkProvider(w, r)
}

func (h *ProviderHandler) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.triggerHealthCheck(w, r)
	case http.MethodGet:
		h.getProviderHealth(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *ProviderHandler) handleCircuitControl(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.controlCircuitBreaker(w, r)
	case http.MethodGet:
		h.getCircuitBreakerState(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *ProviderHandler) handleProviderMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	h.getProviderMetrics(w, r)
}

// listProviders returns a paginated list of providers
func (h *ProviderHandler) listProviders(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspaceId")
	providerType := r.URL.Query().Get("providerType")
	status := r.URL.Query().Get("status")
	page := readIntParam(r, "page", 1)
	pageSize := readIntParam(r, "pageSize", 50)

	h.log.Debug("listing providers",
		"workspace", workspaceID,
		"type", providerType,
		"status", status,
	)

	// Mock providers
	lastSuccess := time.Now().Add(-5 * time.Minute)
	latency := 150
	errorMsg := "Connection timeout"

	providers := []ProviderResponse{
		{
			ID:           "prov_001",
			WorkspaceID:  "ws_001",
			Slug:         "openai-production",
			Name:         "OpenAI Production",
			ProviderType: "openai",
			BaseURL:      "https://api.openai.com",
			Config:       map[string]interface{}{"timeout": 30},
			Status:       "active",
			Priority:     1,
			Weight:       100,
			Health: &ProviderHealthStatus{
				Healthy:             true,
				LastCheckAt:         time.Now(),
				LastSuccessAt:       &lastSuccess,
				ConsecutiveFailures: 0,
				LatencyMs:           &latency,
			},
			CircuitState: &CircuitBreakerState{
				State:     "closed",
				Failures:  0,
				Successes: 1523,
			},
			CreatedAt: time.Now().Add(-30 * 24 * time.Hour),
			UpdatedAt: time.Now(),
		},
		{
			ID:           "prov_002",
			WorkspaceID:  "ws_001",
			Slug:         "anthropic-production",
			Name:         "Anthropic Production",
			ProviderType: "anthropic",
			BaseURL:      "https://api.anthropic.com",
			Config:       map[string]interface{}{"timeout": 30},
			Status:       "active",
			Priority:     2,
			Weight:       80,
			Health: &ProviderHealthStatus{
				Healthy:             false,
				LastCheckAt:         time.Now(),
				ConsecutiveFailures: 3,
				LatencyMs:           nil,
				ErrorMessage:        &errorMsg,
			},
			CircuitState: &CircuitBreakerState{
				State:       "open",
				Failures:    5,
				Successes:   890,
				OpenedAt:    timePtr(time.Now().Add(-10 * time.Minute)),
				LastFailureAt: timePtr(time.Now()),
			},
			CreatedAt: time.Now().Add(-30 * 24 * time.Hour),
			UpdatedAt: time.Now(),
		},
		{
			ID:           "prov_003",
			WorkspaceID:  "ws_001",
			Slug:         "gemini-production",
			Name:         "Google Gemini Production",
			ProviderType: "gemini",
			BaseURL:      "https://generativelanguage.googleapis.com",
			Config:       map[string]interface{}{"timeout": 30},
			Status:       "active",
			Priority:     3,
			Weight:       60,
			Health: &ProviderHealthStatus{
				Healthy:             true,
				LastCheckAt:         time.Now(),
				LastSuccessAt:       timePtr(time.Now().Add(-2 * time.Minute)),
				ConsecutiveFailures: 0,
				LatencyMs:           intPtr(120),
			},
			CircuitState: &CircuitBreakerState{
				State:     "closed",
				Failures:  0,
				Successes: 2341,
			},
			CreatedAt: time.Now().Add(-20 * 24 * time.Hour),
			UpdatedAt: time.Now(),
		},
	}

	// Apply filters
	var filtered []ProviderResponse
	for _, p := range providers {
		if workspaceID != "" && p.WorkspaceID != workspaceID {
			continue
		}
		if providerType != "" && p.ProviderType != providerType {
			continue
		}
		if status != "" && p.Status != status {
			continue
		}
		filtered = append(filtered, p)
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

	response := ProviderListResponse{
		Data:     filtered[start:end],
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  end < total,
	}

	writeJSON(w, http.StatusOK, response)
}

// getProvider returns a single provider
func (h *ProviderHandler) getProvider(w http.ResponseWriter, r *http.Request, id string) {
	h.log.Debug("getting provider", "id", id)

	lastSuccess := time.Now().Add(-5 * time.Minute)
	latency := 150

	provider := ProviderResponse{
		ID:           id,
		WorkspaceID:  "ws_001",
		Slug:         "openai-production",
		Name:         "OpenAI Production",
		ProviderType: "openai",
		BaseURL:      "https://api.openai.com",
		Config:       map[string]interface{}{"timeout": 30},
		Status:       "active",
		Priority:     1,
		Weight:       100,
		Health: &ProviderHealthStatus{
			Healthy:             true,
			LastCheckAt:         time.Now(),
			LastSuccessAt:       &lastSuccess,
			ConsecutiveFailures: 0,
			LatencyMs:           &latency,
		},
		CircuitState: &CircuitBreakerState{
			State:     "closed",
			Failures:  0,
			Successes: 1523,
		},
		CreatedAt: time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt: time.Now(),
	}

	writeJSON(w, http.StatusOK, provider)
}

// createProvider creates a new provider
func (h *ProviderHandler) createProvider(w http.ResponseWriter, r *http.Request) {
	var req ProviderCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if req.ProviderType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "providerType is required"})
		return
	}
	if req.BaseURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "baseUrl is required"})
		return
	}
	if req.APIKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "apiKey is required"})
		return
	}

	if req.Slug == "" {
		req.Slug = slugify(req.Name)
	}

	h.log.Info("creating provider",
		"name", req.Name,
		"type", req.ProviderType,
		"baseUrl", req.BaseURL,
	)

	// Store API key securely (mock)
	_ = req.APIKey

	provider := ProviderResponse{
		ID:           generateID("prov"),
		WorkspaceID:  req.WorkspaceID,
		Slug:         req.Slug,
		Name:         req.Name,
		ProviderType: req.ProviderType,
		BaseURL:      req.BaseURL,
		Config:       req.Config,
		Status:       "active",
		Priority:     req.Priority,
		Weight:       req.Weight,
		Health: &ProviderHealthStatus{
			Healthy:     true,
			LastCheckAt: time.Now(),
		},
		CircuitState: &CircuitBreakerState{
			State:     "closed",
			Failures:  0,
			Successes: 0,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	writeJSON(w, http.StatusCreated, provider)
}

// updateProvider fully updates a provider
func (h *ProviderHandler) updateProvider(w http.ResponseWriter, r *http.Request, id string) {
	var req ProviderUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	h.log.Info("updating provider", "id", id)

	provider := ProviderResponse{
		ID:           id,
		WorkspaceID:  "ws_001",
		Slug:         "openai-production",
		Name:         req.Name,
		ProviderType: "openai",
		BaseURL:      req.BaseURL,
		Config:       req.Config,
		Status:       req.Status,
		UpdatedAt:    time.Now(),
	}

	if req.Priority != nil {
		provider.Priority = *req.Priority
	}
	if req.Weight != nil {
		provider.Weight = *req.Weight
	}

	writeJSON(w, http.StatusOK, provider)
}

// patchProvider partially updates a provider
func (h *ProviderHandler) patchProvider(w http.ResponseWriter, r *http.Request, id string) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	h.log.Info("patching provider", "id", id, "fields", len(updates))

	latency := 150
	provider := ProviderResponse{
		ID:           id,
		WorkspaceID:  "ws_001",
		Slug:         "openai-production",
		Name:         "OpenAI Production",
		ProviderType: "openai",
		BaseURL:      "https://api.openai.com",
		Config:       map[string]interface{}{"timeout": 30},
		Status:       "active",
		Priority:     1,
		Weight:       100,
		Health: &ProviderHealthStatus{
			Healthy:             true,
			LastCheckAt:         time.Now(),
			LastSuccessAt:       timePtr(time.Now().Add(-5 * time.Minute)),
			ConsecutiveFailures: 0,
			LatencyMs:           &latency,
		},
		UpdatedAt: time.Now(),
	}

	writeJSON(w, http.StatusOK, provider)
}

// deleteProvider deletes a provider
func (h *ProviderHandler) deleteProvider(w http.ResponseWriter, r *http.Request, id string) {
	h.log.Info("deleting provider", "id", id)
	writeJSON(w, http.StatusNoContent, nil)
}

// handleBulkProvider handles bulk provider operations
func (h *ProviderHandler) handleBulkProvider(w http.ResponseWriter, r *http.Request) {
	var req BulkProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(req.IDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ids array is required"})
		return
	}

	if req.Action == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "action is required"})
		return
	}

	h.log.Info("bulk provider operation",
		"action", req.Action,
		"count", len(req.IDs),
	)

	validActions := map[string]bool{
		"activate":   true,
		"deactivate": true,
		"delete":     true,
		"health_check": true,
	}
	if !validActions[req.Action] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid action"})
		return
	}

	result := map[string]interface{}{
		"processed": len(req.IDs),
		"action":    req.Action,
		"success":   true,
	}

	writeJSON(w, http.StatusOK, result)
}

// triggerHealthCheck triggers a health check for a provider
func (h *ProviderHandler) triggerHealthCheck(w http.ResponseWriter, r *http.Request) {
	var req ProviderHealthCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.ProviderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "providerId is required"})
		return
	}

	h.log.Info("triggering health check", "providerId", req.ProviderID)

	// Mock health check
	response := ProviderHealthCheckResponse{
		ProviderID: req.ProviderID,
		Healthy:    true,
		LatencyMs:  145,
		CheckedAt:  time.Now(),
		Details: map[string]string{
			"api_version": "v1",
			"models_available": "3",
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// getProviderHealth returns health status for all providers
func (h *ProviderHandler) getProviderHealth(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspaceId")

	h.log.Debug("getting provider health", "workspace", workspaceID)

	// Mock health statuses
	healthStatuses := []ProviderHealthStatus{
		{
			Healthy:             true,
			LastCheckAt:         time.Now(),
			LastSuccessAt:       timePtr(time.Now().Add(-5 * time.Minute)),
			ConsecutiveFailures: 0,
			LatencyMs:           intPtr(145),
		},
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": healthStatuses,
	})
}

// controlCircuitBreaker controls the circuit breaker state
func (h *ProviderHandler) controlCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	var req CircuitBreakerControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	providerID := r.URL.Query().Get("providerId")
	if providerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "providerId is required"})
		return
	}

	validActions := map[string]bool{
		"open":   true,
		"close":  true,
		"reset":  true,
	}
	if !validActions[req.Action] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid action"})
		return
	}

	h.log.Info("controlling circuit breaker",
		"providerId", providerID,
		"action", req.Action,
	)

	newState := "closed"
	switch req.Action {
	case "open":
		newState = "open"
	case "close":
		newState = "closed"
	case "reset":
		newState = "closed"
	}

	response := CircuitBreakerControlResponse{
		ProviderID: providerID,
		Action:     req.Action,
		NewState:   newState,
		AppliedAt:  time.Now(),
	}

	writeJSON(w, http.StatusOK, response)
}

// getCircuitBreakerState returns circuit breaker state
func (h *ProviderHandler) getCircuitBreakerState(w http.ResponseWriter, r *http.Request) {
	providerID := r.URL.Query().Get("providerId")

	h.log.Debug("getting circuit breaker state", "providerId", providerID)

	state := CircuitBreakerState{
		State:     "closed",
		Failures:  0,
		Successes: 1523,
	}

	writeJSON(w, http.StatusOK, state)
}

// getProviderMetrics returns provider metrics
func (h *ProviderHandler) getProviderMetrics(w http.ResponseWriter, r *http.Request) {
	providerID := r.URL.Query().Get("providerId")
	startTime := h.parseTimeParam(r, "startTime", time.Now().Add(-24*time.Hour))
	endTime := h.parseTimeParam(r, "endTime", time.Now())

	h.log.Debug("getting provider metrics",
		"providerId", providerID,
		"startTime", startTime,
		"endTime", endTime,
	)

	// Mock metrics
	metrics := ProviderMetricsResponse{
		ProviderID:   providerID,
		ProviderName: "OpenAI Production",
		TimeRange:    TimeRange{Start: startTime, End: endTime},
		RequestCount: 15234,
		SuccessCount: 15081,
		ErrorCount:   153,
		ErrorRate:    1.0,
		AvgLatencyMs: 145.5,
		TotalTokens:  152340000,
		TotalCostUSD: 1523.45,
		P50LatencyMs: 120.0,
		P95LatencyMs: 250.0,
		P99LatencyMs: 450.0,
	}

	writeJSON(w, http.StatusOK, metrics)
}

// Helper methods

func (h *ProviderHandler) parseTimeParam(r *http.Request, name string, fallback time.Time) time.Time {
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
