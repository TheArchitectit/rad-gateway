package admin

import (
	"context"
	"crypto/rand"
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

type APIKeyCreateRequest struct {
	Name          string          `json:"name"`
	WorkspaceID   string          `json:"workspaceId"`
	ExpiresAt     *string         `json:"expiresAt,omitempty"`
	RateLimit     *int            `json:"rateLimit,omitempty"`
	AllowedModels []string        `json:"allowedModels,omitempty"`
	AllowedAPIs   []string        `json:"allowedAPIs,omitempty"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}

type APIKeyUpdateRequest struct {
	Name          string          `json:"name,omitempty"`
	Status        string          `json:"status,omitempty"`
	ExpiresAt     *string         `json:"expiresAt,omitempty"`
	RateLimit     *int            `json:"rateLimit,omitempty"`
	AllowedModels []string        `json:"allowedModels,omitempty"`
	AllowedAPIs   []string        `json:"allowedAPIs,omitempty"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}

type APIKeyRotateRequest struct {
	ExpiresAt *string `json:"expiresAt,omitempty"`
}

type APIKeyListResponse struct {
	Data     []APIKeyResponse `json:"data"`
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
	HasMore  bool             `json:"hasMore"`
}

type APIKeyResponse struct {
	ID            string     `json:"id"`
	WorkspaceID   string     `json:"workspaceId"`
	Name          string     `json:"name"`
	KeyPreview    string     `json:"keyPreview"`
	Status        string     `json:"status"`
	CreatedBy     *string    `json:"createdBy,omitempty"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt    *time.Time `json:"lastUsedAt,omitempty"`
	RateLimit     *int       `json:"rateLimit,omitempty"`
	AllowedModels []string   `json:"allowedModels,omitempty"`
	AllowedAPIs   []string   `json:"allowedAPIs,omitempty"`
	Metadata      []byte     `json:"metadata,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type APIKeyWithSecretResponse struct {
	APIKeyResponse
	KeySecret string `json:"keySecret"`
}

type BulkAPIKeyRequest struct {
	IDs    []string `json:"ids"`
	Action string   `json:"action"`
}

type APIKeyFilter struct {
	Status      string
	WorkspaceID string
	Search      string
}

type APIKeyHandler struct {
	log *slog.Logger
	db  db.Database
}

func NewAPIKeyHandler(database db.Database) *APIKeyHandler {
	return &APIKeyHandler{log: logger.WithComponent("admin.apikeys"), db: database}
}

func (h *APIKeyHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v0/admin/apikeys", h.handleAPIKeys)
	mux.HandleFunc("/v0/admin/apikeys/", h.handleAPIKeyDetail)
	mux.HandleFunc("/v0/admin/apikeys/bulk", h.handleBulkOperation)
}

func (h *APIKeyHandler) handleAPIKeys(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database not configured"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.listAPIKeys(w, r)
	case http.MethodPost:
		h.createAPIKey(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *APIKeyHandler) handleAPIKeyDetail(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database not configured"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v0/admin/apikeys/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "api key id required"})
		return
	}
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
	case http.MethodPatch:
		h.patchAPIKey(w, r, id)
	case http.MethodDelete:
		h.deleteAPIKey(w, r, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *APIKeyHandler) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	filter := APIKeyFilter{
		Status:      r.URL.Query().Get("status"),
		WorkspaceID: r.URL.Query().Get("workspaceId"),
		Search:      strings.TrimSpace(strings.ToLower(r.URL.Query().Get("search"))),
	}
	page := readIntParam(r, "page", 1)
	pageSize := readIntParam(r, "pageSize", 50)
	if pageSize > 500 {
		pageSize = 500
	}
	offset := (page - 1) * pageSize

	var keys []db.APIKey
	var err error
	if filter.WorkspaceID != "" {
		keys, err = h.db.APIKeys().GetByWorkspace(r.Context(), filter.WorkspaceID, pageSize, offset)
	} else {
		keys, err = h.listAllAPIKeys(r.Context(), pageSize, offset)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list api keys"})
		return
	}

	filtered := make([]APIKeyResponse, 0, len(keys))
	for _, key := range keys {
		if filter.Status != "" && key.Status != filter.Status {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(key.Name), filter.Search) {
			continue
		}
		filtered = append(filtered, toAPIKeyResponse(&key))
	}

	writeJSON(w, http.StatusOK, APIKeyListResponse{
		Data:     filtered,
		Total:    len(filtered),
		Page:     page,
		PageSize: pageSize,
		HasMore:  len(keys) == pageSize,
	})
}

func (h *APIKeyHandler) getAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	key, err := h.db.APIKeys().GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get api key"})
		return
	}
	if key == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "api key not found"})
		return
	}
	writeJSON(w, http.StatusOK, toAPIKeyResponse(key))
}

func (h *APIKeyHandler) createAPIKey(w http.ResponseWriter, r *http.Request) {
	var req APIKeyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.WorkspaceID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and workspaceId are required"})
		return
	}
	secret := generateAPIKeySecret()
	preview := "rad..." + secret[len(secret)-3:]
	now := time.Now().UTC()
	metadata := []byte("{}")
	if len(req.Metadata) > 0 {
		metadata = req.Metadata
	}
	allowedModels, _ := json.Marshal(req.AllowedModels)
	allowedAPIs, _ := json.Marshal(req.AllowedAPIs)
	if len(allowedModels) == 0 {
		allowedModels = []byte("[]")
	}
	if len(allowedAPIs) == 0 {
		allowedAPIs = []byte("[]")
	}
	apiKey := &db.APIKey{
		ID:            generateID("key"),
		WorkspaceID:   req.WorkspaceID,
		Name:          req.Name,
		KeyHash:       hashAPIKey(secret),
		KeyPreview:    preview,
		Status:        "active",
		RateLimit:     req.RateLimit,
		AllowedModels: req.AllowedModels,
		AllowedAPIs:   req.AllowedAPIs,
		Metadata:      metadata,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		if t, err := time.Parse(time.RFC3339, *req.ExpiresAt); err == nil {
			apiKey.ExpiresAt = &t
		}
	}
	_ = allowedModels
	_ = allowedAPIs
	if err := h.db.APIKeys().Create(r.Context(), apiKey); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create api key"})
		return
	}
	writeJSON(w, http.StatusCreated, APIKeyWithSecretResponse{APIKeyResponse: toAPIKeyResponse(apiKey), KeySecret: secret})
}

func (h *APIKeyHandler) updateAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	var req APIKeyUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	key, err := h.db.APIKeys().GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get api key"})
		return
	}
	if key == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "api key not found"})
		return
	}
	if req.Name != "" {
		key.Name = req.Name
	}
	if req.Status != "" {
		key.Status = req.Status
	}
	if req.RateLimit != nil {
		key.RateLimit = req.RateLimit
	}
	if req.AllowedModels != nil {
		key.AllowedModels = req.AllowedModels
	}
	if req.AllowedAPIs != nil {
		key.AllowedAPIs = req.AllowedAPIs
	}
	if len(req.Metadata) > 0 {
		key.Metadata = req.Metadata
	}
	if req.ExpiresAt != nil {
		if *req.ExpiresAt == "" {
			key.ExpiresAt = nil
		} else if t, err := time.Parse(time.RFC3339, *req.ExpiresAt); err == nil {
			key.ExpiresAt = &t
		}
	}
	key.UpdatedAt = time.Now().UTC()
	if err := h.db.APIKeys().Update(r.Context(), key); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update api key"})
		return
	}
	writeJSON(w, http.StatusOK, toAPIKeyResponse(key))
}

func (h *APIKeyHandler) patchAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	h.updateAPIKey(w, r, id)
}

func (h *APIKeyHandler) revokeAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	key, err := h.db.APIKeys().GetByID(r.Context(), id)
	if err != nil || key == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "api key not found"})
		return
	}
	key.Status = "revoked"
	key.UpdatedAt = time.Now().UTC()
	if err := h.db.APIKeys().Update(r.Context(), key); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to revoke api key"})
		return
	}
	writeJSON(w, http.StatusOK, toAPIKeyResponse(key))
}

func (h *APIKeyHandler) rotateAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req APIKeyRotateRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	key, err := h.db.APIKeys().GetByID(r.Context(), id)
	if err != nil || key == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "api key not found"})
		return
	}
	secret := generateAPIKeySecret()
	key.KeyHash = hashAPIKey(secret)
	key.KeyPreview = "rad..." + secret[len(secret)-3:]
	key.Status = "active"
	if req.ExpiresAt != nil {
		if *req.ExpiresAt == "" {
			key.ExpiresAt = nil
		} else if t, err := time.Parse(time.RFC3339, *req.ExpiresAt); err == nil {
			key.ExpiresAt = &t
		}
	}
	key.UpdatedAt = time.Now().UTC()
	if err := h.db.APIKeys().Update(r.Context(), key); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to rotate api key"})
		return
	}
	writeJSON(w, http.StatusOK, APIKeyWithSecretResponse{APIKeyResponse: toAPIKeyResponse(key), KeySecret: secret})
}

func (h *APIKeyHandler) deleteAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.db.APIKeys().Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete api key"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *APIKeyHandler) handleBulkOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req BulkAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	processed := 0
	for _, id := range req.IDs {
		switch req.Action {
		case "delete":
			if err := h.db.APIKeys().Delete(r.Context(), id); err == nil {
				processed++
			}
		case "revoke", "activate":
			key, err := h.db.APIKeys().GetByID(r.Context(), id)
			if err != nil || key == nil {
				continue
			}
			if req.Action == "revoke" {
				key.Status = "revoked"
			} else {
				key.Status = "active"
			}
			key.UpdatedAt = time.Now().UTC()
			if err := h.db.APIKeys().Update(r.Context(), key); err == nil {
				processed++
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"processed": processed, "action": req.Action, "success": true})
}

func (h *APIKeyHandler) listAllAPIKeys(ctx context.Context, limit, offset int) ([]db.APIKey, error) {
	workspaces, err := h.db.Workspaces().List(ctx, 500, 0)
	if err != nil {
		return nil, err
	}
	all := make([]db.APIKey, 0)
	for _, ws := range workspaces {
		keys, err := h.db.APIKeys().GetByWorkspace(ctx, ws.ID, 500, 0)
		if err != nil {
			return nil, err
		}
		all = append(all, keys...)
	}
	if offset >= len(all) {
		return []db.APIKey{}, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

func toAPIKeyResponse(key *db.APIKey) APIKeyResponse {
	return APIKeyResponse{
		ID:            key.ID,
		WorkspaceID:   key.WorkspaceID,
		Name:          key.Name,
		KeyPreview:    key.KeyPreview,
		Status:        key.Status,
		CreatedBy:     key.CreatedBy,
		ExpiresAt:     key.ExpiresAt,
		LastUsedAt:    key.LastUsedAt,
		RateLimit:     key.RateLimit,
		AllowedModels: key.AllowedModels,
		AllowedAPIs:   key.AllowedAPIs,
		Metadata:      key.Metadata,
		CreatedAt:     key.CreatedAt,
		UpdatedAt:     key.UpdatedAt,
	}
}

func generateAPIKeySecret() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "rad_" + hex.EncodeToString(b)
}

func hashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
