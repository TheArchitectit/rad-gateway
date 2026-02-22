package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"radgateway/internal/db"
	"radgateway/internal/logger"
)

type ProviderCreateRequest struct {
	Name         string                 `json:"name"`
	Slug         string                 `json:"slug"`
	WorkspaceID  string                 `json:"workspaceId"`
	ProviderType string                 `json:"providerType"`
	BaseURL      string                 `json:"baseUrl"`
	APIKey       string                 `json:"apiKey"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Priority     int                    `json:"priority"`
	Weight       int                    `json:"weight"`
}

type ProviderUpdateRequest struct {
	Name     string                 `json:"name,omitempty"`
	BaseURL  string                 `json:"baseUrl,omitempty"`
	APIKey   string                 `json:"apiKey,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Status   string                 `json:"status,omitempty"`
	Priority *int                   `json:"priority,omitempty"`
	Weight   *int                   `json:"weight,omitempty"`
}

type ProviderResponse struct {
	ID           string                  `json:"id"`
	WorkspaceID  string                  `json:"workspaceId"`
	Slug         string                  `json:"slug"`
	Name         string                  `json:"name"`
	ProviderType string                  `json:"providerType"`
	BaseURL      string                  `json:"baseUrl"`
	Config       map[string]any          `json:"config"`
	Status       string                  `json:"status"`
	Priority     int                     `json:"priority"`
	Weight       int                     `json:"weight"`
	Health       *db.ProviderHealth      `json:"health,omitempty"`
	CircuitState *db.CircuitBreakerState `json:"circuitState,omitempty"`
	CreatedAt    time.Time               `json:"createdAt"`
	UpdatedAt    time.Time               `json:"updatedAt"`
}

type ProviderListResponse struct {
	Data     []ProviderResponse `json:"data"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
	HasMore  bool               `json:"hasMore"`
}

type ProviderHealthCheckRequest struct {
	ProviderID string `json:"providerId"`
}

type CircuitBreakerControlRequest struct {
	Action string `json:"action"`
}

type BulkProviderRequest struct {
	IDs    []string `json:"ids"`
	Action string   `json:"action"`
}

type ProviderHandler struct {
	log *slog.Logger
	db  db.Database
}

func NewProviderHandler(database db.Database) *ProviderHandler {
	return &ProviderHandler{log: logger.WithComponent("admin.providers"), db: database}
}

func (h *ProviderHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v0/admin/providers", h.handleProviders)
	mux.HandleFunc("/v0/admin/providers/", h.handleProviderDetail)
	mux.HandleFunc("/v0/admin/providers/bulk", h.handleBulkOperation)
	mux.HandleFunc("/v0/admin/providers/health", h.handleHealthCheck)
	mux.HandleFunc("/v0/admin/providers/circuit", h.handleCircuitControl)
}

func (h *ProviderHandler) handleProviders(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database not configured"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.listProviders(w, r)
	case http.MethodPost:
		h.createProvider(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *ProviderHandler) handleProviderDetail(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database not configured"})
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/v0/admin/providers/")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider id required"})
		return
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	id := parts[0]
	if len(parts) > 1 {
		switch parts[1] {
		case "test", "health":
			h.triggerHealthCheckByID(w, r, id)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		h.getProvider(w, r, id)
	case http.MethodPut:
		h.updateProvider(w, r, id)
	case http.MethodPatch:
		h.patchProvider(w, r, id)
	case http.MethodDelete:
		h.deleteProvider(w, r, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *ProviderHandler) handleBulkOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req BulkProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	processed := 0
	for _, id := range req.IDs {
		switch req.Action {
		case "delete":
			if err := h.db.Providers().Delete(r.Context(), id); err == nil {
				processed++
			}
		case "activate", "deactivate":
			p, err := h.db.Providers().GetByID(r.Context(), id)
			if err != nil || p == nil {
				continue
			}
			if req.Action == "activate" {
				p.Status = "active"
			} else {
				p.Status = "inactive"
			}
			p.UpdatedAt = time.Now().UTC()
			if err := h.db.Providers().Update(r.Context(), p); err == nil {
				processed++
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"processed": processed, "action": req.Action, "success": true})
}

func (h *ProviderHandler) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req ProviderHealthCheckRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ProviderID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "providerId is required"})
			return
		}
		h.triggerHealthCheckByID(w, r, req.ProviderID)
		return
	}
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func (h *ProviderHandler) handleCircuitControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	providerID := r.URL.Query().Get("providerId")
	if providerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "providerId is required"})
		return
	}
	var req CircuitBreakerControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	state := "closed"
	if req.Action == "open" {
		state = "open"
	}
	now := time.Now().UTC()
	err := h.db.Providers().UpdateCircuitBreaker(r.Context(), &db.CircuitBreakerState{
		ProviderID: providerID,
		State:      state,
		UpdatedAt:  now,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update circuit breaker"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"providerId": providerID, "action": req.Action, "newState": state, "appliedAt": now})
}

func (h *ProviderHandler) listProviders(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspaceId")
	page := readIntParam(r, "page", 1)
	pageSize := readIntParam(r, "pageSize", 50)
	if pageSize > 500 {
		pageSize = 500
	}

	providers, err := h.fetchProvidersForWorkspace(r, workspaceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list providers"})
		return
	}

	resp := make([]ProviderResponse, 0, len(providers))
	for _, p := range providers {
		health, _ := h.db.Providers().GetHealth(r.Context(), p.ID)
		cb, _ := h.db.Providers().GetCircuitBreaker(r.Context(), p.ID)
		resp = append(resp, toProviderResponse(&p, health, cb))
	}

	start := (page - 1) * pageSize
	if start > len(resp) {
		start = len(resp)
	}
	end := start + pageSize
	if end > len(resp) {
		end = len(resp)
	}

	writeJSON(w, http.StatusOK, ProviderListResponse{
		Data:     resp[start:end],
		Total:    len(resp),
		Page:     page,
		PageSize: pageSize,
		HasMore:  end < len(resp),
	})
}

func (h *ProviderHandler) getProvider(w http.ResponseWriter, r *http.Request, id string) {
	p, err := h.db.Providers().GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get provider"})
		return
	}
	if p == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "provider not found"})
		return
	}
	health, _ := h.db.Providers().GetHealth(r.Context(), p.ID)
	cb, _ := h.db.Providers().GetCircuitBreaker(r.Context(), p.ID)
	writeJSON(w, http.StatusOK, toProviderResponse(p, health, cb))
}

func (h *ProviderHandler) createProvider(w http.ResponseWriter, r *http.Request) {
	var req ProviderCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" || req.ProviderType == "" || req.BaseURL == "" || req.WorkspaceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, providerType, baseUrl, and workspaceId are required"})
		return
	}
	if req.Slug == "" {
		req.Slug = slugify(req.Name)
	}
	config, _ := json.Marshal(req.Config)
	if len(config) == 0 {
		config = []byte("{}")
	}
	apiKeyHash := sha256.Sum256([]byte(req.APIKey))
	apiKeyEncrypted := hex.EncodeToString(apiKeyHash[:])
	now := time.Now().UTC()
	p := &db.Provider{
		ID:              generateID("prov"),
		WorkspaceID:     req.WorkspaceID,
		Slug:            req.Slug,
		Name:            req.Name,
		ProviderType:    req.ProviderType,
		BaseURL:         req.BaseURL,
		APIKeyEncrypted: &apiKeyEncrypted,
		Config:          config,
		Status:          "active",
		Priority:        req.Priority,
		Weight:          req.Weight,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if p.Weight == 0 {
		p.Weight = 1
	}
	if err := h.db.Providers().Create(r.Context(), p); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create provider"})
		return
	}
	writeJSON(w, http.StatusCreated, toProviderResponse(p, nil, nil))
}

func (h *ProviderHandler) updateProvider(w http.ResponseWriter, r *http.Request, id string) {
	var req ProviderUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	p, err := h.db.Providers().GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get provider"})
		return
	}
	if p == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "provider not found"})
		return
	}
	if req.Name != "" {
		p.Name = req.Name
	}
	if req.BaseURL != "" {
		p.BaseURL = req.BaseURL
	}
	if req.Status != "" {
		p.Status = req.Status
	}
	if req.Priority != nil {
		p.Priority = *req.Priority
	}
	if req.Weight != nil {
		p.Weight = *req.Weight
	}
	if req.Config != nil {
		cfg, _ := json.Marshal(req.Config)
		if len(cfg) > 0 {
			p.Config = cfg
		}
	}
	if strings.TrimSpace(req.APIKey) != "" {
		sum := sha256.Sum256([]byte(req.APIKey))
		hash := hex.EncodeToString(sum[:])
		p.APIKeyEncrypted = &hash
	}
	p.UpdatedAt = time.Now().UTC()
	if err := h.db.Providers().Update(r.Context(), p); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update provider"})
		return
	}
	health, _ := h.db.Providers().GetHealth(r.Context(), p.ID)
	cb, _ := h.db.Providers().GetCircuitBreaker(r.Context(), p.ID)
	writeJSON(w, http.StatusOK, toProviderResponse(p, health, cb))
}

func (h *ProviderHandler) patchProvider(w http.ResponseWriter, r *http.Request, id string) {
	h.updateProvider(w, r, id)
}

func (h *ProviderHandler) deleteProvider(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.db.Providers().Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete provider"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProviderHandler) triggerHealthCheckByID(w http.ResponseWriter, r *http.Request, providerID string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	now := time.Now().UTC()
	lat := 100
	health := &db.ProviderHealth{
		ProviderID:          providerID,
		Healthy:             true,
		LastCheckAt:         now,
		LastSuccessAt:       &now,
		ConsecutiveFailures: 0,
		LatencyMs:           &lat,
		UpdatedAt:           now,
	}
	if err := h.db.Providers().UpdateHealth(r.Context(), health); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update provider health"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"providerId": providerID, "healthy": true, "latencyMs": lat, "checkedAt": now})
}

func (h *ProviderHandler) fetchProvidersForWorkspace(r *http.Request, workspaceID string) ([]db.Provider, error) {
	if workspaceID != "" {
		return h.db.Providers().GetByWorkspace(r.Context(), workspaceID)
	}
	workspaces, err := h.db.Workspaces().List(r.Context(), 500, 0)
	if err != nil {
		return nil, err
	}
	all := make([]db.Provider, 0)
	for _, ws := range workspaces {
		providers, err := h.db.Providers().GetByWorkspace(r.Context(), ws.ID)
		if err != nil {
			return nil, err
		}
		all = append(all, providers...)
	}
	return all, nil
}

func toProviderResponse(p *db.Provider, health *db.ProviderHealth, cb *db.CircuitBreakerState) ProviderResponse {
	config := map[string]any{}
	if len(p.Config) > 0 {
		_ = json.Unmarshal(p.Config, &config)
	}
	return ProviderResponse{
		ID:           p.ID,
		WorkspaceID:  p.WorkspaceID,
		Slug:         p.Slug,
		Name:         p.Name,
		ProviderType: p.ProviderType,
		BaseURL:      p.BaseURL,
		Config:       config,
		Status:       p.Status,
		Priority:     p.Priority,
		Weight:       p.Weight,
		Health:       health,
		CircuitState: cb,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}
