package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"radgateway/internal/logger"
)

// APIKeyCreateRequest represents a request to create an API key
type APIKeyCreateRequest struct {
	Name          string   `json:"name"`
	WorkspaceID   string   `json:"workspaceId"`
	ExpiresAt     *string  `json:"expiresAt,omitempty"`
	RateLimit     *int     `json:"rateLimit,omitempty"`
	AllowedModels []string `json:"allowedModels,omitempty"`
	AllowedAPIs   []string `json:"allowedAPIs,omitempty"`
	Metadata      []byte   `json:"metadata,omitempty"`
}

// APIKeyUpdateRequest represents a request to update an API key
type APIKeyUpdateRequest struct {
	Name          string   `json:"name,omitempty"`
	Status        string   `json:"status,omitempty"`
	ExpiresAt     *string  `json:"expiresAt,omitempty"`
	RateLimit     *int     `json:"rateLimit,omitempty"`
	AllowedModels []string `json:"allowedModels,omitempty"`
	AllowedAPIs   []string `json:"allowedAPIs,omitempty"`
	Metadata      []byte   `json:"metadata,omitempty"`
}

// APIKeyRotateRequest represents a request to rotate an API key
type APIKeyRotateRequest struct {
	ExpiresAt *string `json:"expiresAt,omitempty"`
}

// APIKeyListResponse represents the list response
type APIKeyListResponse struct {
	Data       []APIKeyResponse `json:"data"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"pageSize"`
	HasMore    bool             `json:"hasMore"`
}

// APIKeyResponse represents an API key in responses (with sensitive data masked)
type APIKeyResponse struct {
	ID            string     `json:"id"`
	WorkspaceID   string     `json:"workspaceId"`
	Name          string     `json:"name"`
	KeyPreview    string     `json:"keyPreview"`
	Status        string     `json:"status"`
	CreatedBy     *string    `json:"createdBy,omitempty"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt    *time.Time `json:"lastUsedAt,omitempty"`
	RevokedAt     *time.Time `json:"revokedAt,omitempty"`
	RateLimit     *int       `json:"rateLimit,omitempty"`
	AllowedModels []string   `json:"allowedModels,omitempty"`
	AllowedAPIs   []string   `json:"allowedAPIs,omitempty"`
	Metadata      []byte     `json:"metadata,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// APIKeyWithSecretResponse represents response when creating a new key (includes the secret)
type APIKeyWithSecretResponse struct {
	APIKeyResponse
	KeySecret string `json:"keySecret"`
}

// BulkAPIKeyRequest represents a bulk operation request
type BulkAPIKeyRequest struct {
	IDs    []string `json:"ids"`
	Action string   `json:"action"`
}

// APIKeyFilter represents filter options
type APIKeyFilter struct {
	Status      string
	WorkspaceID string
	Search      string
	SortBy      string
	SortOrder   string
}

// APIKeyHandler handles API key management endpoints
type APIKeyHandler struct {
	log *slog.Logger
}

// NewAPIKeyHandler creates a new API key handler
func NewAPIKeyHandler() *APIKeyHandler {
	return &APIKeyHandler{
		log: logger.WithComponent("admin.apikeys"),
	}
}

// RegisterRoutes registers the API key management routes
func (h *APIKeyHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v0/admin/apikeys", h.handleAPIKeys)
	mux.HandleFunc("/v0/admin/apikeys/", h.handleAPIKeyDetail)
	mux.HandleFunc("/v0/admin/apikeys/bulk", h.handleBulkOperation)
}

func (h *APIKeyHandler) handleAPIKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listAPIKeys(w, r)
	case http.MethodPost:
		h.createAPIKey(w, r)
	default:
		h.log.Warn("method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *APIKeyHandler) handleAPIKeyDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v0/admin/apikeys/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "api key id required"})
		return
	}

	// Check for sub-routes
	if strings.HasSuffix(id, "/revoke") {
		h.revokeAPIKey(w, r, strings.TrimSuffix(id, "/revoke"))
		return
	}
	if strings.HasSuffix(id, "/rotate") {
		h.rotateAPIKey(w, r, strings.TrimSuffix(id, "/rotate"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getAPIKey(w, r, id)
	case http.MethodPut:
		h.updateAPIKey(w, r, id)
	case http.MethodDelete:
		h.deleteAPIKey(w, r, id)
	case http.MethodPatch:
		h.patchAPIKey(w, r, id)
	default:
		h.log.Warn("method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// listAPIKeys returns a paginated list of API keys with filtering
func (h *APIKeyHandler) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	filter := h.parseFilter(r)
	page := readIntParam(r, "page", 1)
	pageSize := readIntParam(r, "pageSize", 50)
	if pageSize > 500 {
		pageSize = 500
	}

	h.log.Debug("listing api keys",
		"page", page,
		"pageSize", pageSize,
		"status", filter.Status,
		"workspace", filter.WorkspaceID,
	)

	// Mock data
	apiKeys := []APIKeyResponse{
		{
			ID:          "key_001",
			WorkspaceID: "ws_001",
			Name:        "Production API Key",
			KeyPreview:  "rad...xyz",
			Status:      "active",
			CreatedBy:   strPtr("user_001"),
			LastUsedAt:  timePtr(time.Now().Add(-1 * time.Hour)),
			RateLimit:   intPtr(1000),
			AllowedModels: []string{"gpt-4o-mini", "claude-3-5-sonnet"},
			AllowedAPIs:   []string{"chat", "embeddings"},
			CreatedAt:     time.Now().Add(-30 * 24 * time.Hour),
			UpdatedAt:     time.Now(),
		},
		{
			ID:          "key_002",
			WorkspaceID: "ws_002",
			Name:        "Staging API Key",
			KeyPreview:  "rad...abc",
			Status:      "active",
			CreatedBy:   strPtr("user_002"),
			LastUsedAt:  timePtr(time.Now().Add(-2 * time.Hour)),
			RateLimit:   intPtr(500),
			CreatedAt:     time.Now().Add(-20 * 24 * time.Hour),
			UpdatedAt:     time.Now(),
		},
		{
			ID:          "key_003",
			WorkspaceID: "ws_001",
			Name:        "Development API Key",
			KeyPreview:  "rad...def",
			Status:      "revoked",
			CreatedBy:   strPtr("user_001"),
			RateLimit:   intPtr(100),
			CreatedAt:     time.Now().Add(-10 * 24 * time.Hour),
			UpdatedAt:     time.Now(),
		},
	}

	// Apply filters
	if filter.Status != "" {
		var filtered []APIKeyResponse
		for _, key := range apiKeys {
			if key.Status == filter.Status {
				filtered = append(filtered, key)
			}
		}
		apiKeys = filtered
	}

	if filter.WorkspaceID != "" {
		var filtered []APIKeyResponse
		for _, key := range apiKeys {
			if key.WorkspaceID == filter.WorkspaceID {
				filtered = append(filtered, key)
			}
		}
		apiKeys = filtered
	}

	if filter.Search != "" {
		var filtered []APIKeyResponse
		searchLower := strings.ToLower(filter.Search)
		for _, key := range apiKeys {
			if strings.Contains(strings.ToLower(key.Name), searchLower) {
				filtered = append(filtered, key)
			}
		}
		apiKeys = filtered
	}

	total := len(apiKeys)

	// Apply pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	pagedKeys := apiKeys[start:end]

	response := APIKeyListResponse{
		Data:     pagedKeys,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  end < total,
	}

	writeJSON(w, http.StatusOK, response)
}

// getAPIKey returns a single API key by ID
func (h *APIKeyHandler) getAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	h.log.Debug("getting api key", "id", id)

	apiKey := APIKeyResponse{
		ID:          id,
		WorkspaceID: "ws_001",
		Name:        "Production API Key",
		KeyPreview:  "rad...xyz",
		Status:      "active",
		CreatedBy:   strPtr("user_001"),
		LastUsedAt:  timePtr(time.Now().Add(-1 * time.Hour)),
		RateLimit:   intPtr(1000),
		AllowedModels: []string{"gpt-4o-mini", "claude-3-5-sonnet"},
		AllowedAPIs:   []string{"chat", "embeddings"},
		Metadata:      []byte(`{"department":"engineering"}`),
		CreatedAt:     time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:     time.Now(),
	}

	writeJSON(w, http.StatusOK, apiKey)
}

// createAPIKey creates a new API key
func (h *APIKeyHandler) createAPIKey(w http.ResponseWriter, r *http.Request) {
	var req APIKeyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	if req.WorkspaceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "workspaceId is required"})
		return
	}

	h.log.Info("creating api key",
		"name", req.Name,
		"workspace", req.WorkspaceID,
	)

	// Generate API key secret
	keySecret := generateAPIKeySecret()
	keyHash := hashAPIKey(keySecret)
	keyPreview := "rad..." + keySecret[len(keySecret)-3:]

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}

	response := APIKeyWithSecretResponse{
		APIKeyResponse: APIKeyResponse{
			ID:            generateID("key"),
			WorkspaceID:   req.WorkspaceID,
			Name:          req.Name,
			KeyPreview:    keyPreview,
			Status:        "active",
			ExpiresAt:     expiresAt,
			RateLimit:     req.RateLimit,
			AllowedModels: req.AllowedModels,
			AllowedAPIs:   req.AllowedAPIs,
			Metadata:      req.Metadata,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		KeySecret: keySecret,
	}

	// Store keyHash in database (not the keySecret)
	_ = keyHash

	writeJSON(w, http.StatusCreated, response)
}

// updateAPIKey fully updates an API key
func (h *APIKeyHandler) updateAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	var req APIKeyUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	h.log.Info("updating api key", "id", id)

	apiKey := APIKeyResponse{
		ID:            id,
		WorkspaceID:   "ws_001",
		Name:          req.Name,
		KeyPreview:    "rad...xyz",
		Status:        req.Status,
		RateLimit:     req.RateLimit,
		AllowedModels: req.AllowedModels,
		AllowedAPIs:   req.AllowedAPIs,
		Metadata:      req.Metadata,
		UpdatedAt:     time.Now(),
	}

	writeJSON(w, http.StatusOK, apiKey)
}

// patchAPIKey partially updates an API key
func (h *APIKeyHandler) patchAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	h.log.Info("patching api key", "id", id, "fields", len(updates))

	apiKey := APIKeyResponse{
		ID:          id,
		WorkspaceID: "ws_001",
		Name:        "Production API Key",
		KeyPreview:  "rad...xyz",
		Status:      "active",
		UpdatedAt:   time.Now(),
	}

	writeJSON(w, http.StatusOK, apiKey)
}

// revokeAPIKey revokes an API key
func (h *APIKeyHandler) revokeAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	h.log.Info("revoking api key", "id", id)

	apiKey := APIKeyResponse{
		ID:          id,
		WorkspaceID: "ws_001",
		Name:        "Production API Key",
		KeyPreview:  "rad...xyz",
		Status:      "revoked",
		RevokedAt:   timePtr(time.Now()),
		UpdatedAt:   time.Now(),
	}

	writeJSON(w, http.StatusOK, apiKey)
}

// rotateAPIKey rotates an API key (creates new, invalidates old)
func (h *APIKeyHandler) rotateAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req APIKeyRotateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Continue without body
	}

	h.log.Info("rotating api key", "id", id)

	// Generate new API key secret
	keySecret := generateAPIKeySecret()
	keyHash := hashAPIKey(keySecret)
	keyPreview := "rad..." + keySecret[len(keySecret)-3:]

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}

	// Store new keyHash in database
	_ = keyHash

	response := APIKeyWithSecretResponse{
		APIKeyResponse: APIKeyResponse{
			ID:          generateID("key"),
			WorkspaceID: "ws_001",
			Name:        "Production API Key (Rotated)",
			KeyPreview:  keyPreview,
			Status:      "active",
			ExpiresAt:   expiresAt,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		KeySecret: keySecret,
	}

	writeJSON(w, http.StatusOK, response)
}

// deleteAPIKey deletes an API key
func (h *APIKeyHandler) deleteAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	h.log.Info("deleting api key", "id", id)

	// In a real implementation, this would delete from database
	writeJSON(w, http.StatusNoContent, nil)
}

// handleBulkOperation handles bulk operations on API keys
func (h *APIKeyHandler) handleBulkOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req BulkAPIKeyRequest
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

	h.log.Info("bulk api key operation",
		"action", req.Action,
		"count", len(req.IDs),
	)

	validActions := map[string]bool{
		"activate": true,
		"revoke":   true,
		"delete":   true,
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

// parseFilter parses filter parameters from request
func (h *APIKeyHandler) parseFilter(r *http.Request) APIKeyFilter {
	return APIKeyFilter{
		Status:      r.URL.Query().Get("status"),
		WorkspaceID: r.URL.Query().Get("workspaceId"),
		Search:      r.URL.Query().Get("search"),
		SortBy:      r.URL.Query().Get("sortBy"),
		SortOrder:   r.URL.Query().Get("sortOrder"),
	}
}

// Helper functions
func generateAPIKeySecret() string {
	// Generate a secure random key: rad_<32 hex chars>
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "rad_" + hex.EncodeToString(b)
}

func hashAPIKey(key string) string {
	// In a real implementation, this would use a proper hash like bcrypt or argon2
	// For now, just return a placeholder
	return key + "_hash"
}
