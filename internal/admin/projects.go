package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"log/slog"

	"radgateway/internal/db"
	"radgateway/internal/logger"
)

// WorkspaceCreateRequest represents a request to create a workspace
type WorkspaceCreateRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	Settings    []byte `json:"settings,omitempty"`
}

// WorkspaceUpdateRequest represents a request to update a workspace
type WorkspaceUpdateRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	Settings    []byte `json:"settings,omitempty"`
}

// BulkWorkspaceRequest represents a bulk operation request
type BulkWorkspaceRequest struct {
	IDs    []string `json:"ids"`
	Action string   `json:"action"`
}

// WorkspaceListResponse represents the list response
type WorkspaceListResponse struct {
	Data       []db.Workspace `json:"data"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"pageSize"`
	HasMore    bool           `json:"hasMore"`
}

// WorkspaceFilter represents filter options
type WorkspaceFilter struct {
	Status    string
	Search    string
	CreatedAfter  time.Time
	CreatedBefore time.Time
	SortBy    string
	SortOrder string
}

// ProjectHandler handles workspace/project management endpoints
type ProjectHandler struct {
	log *slog.Logger
}

// NewProjectHandler creates a new project handler
func NewProjectHandler() *ProjectHandler {
	return &ProjectHandler{
		log: logger.WithComponent("admin.projects"),
	}
}

// RegisterRoutes registers the project management routes
func (h *ProjectHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v0/admin/projects", h.handleProjects)
	mux.HandleFunc("/v0/admin/projects/", h.handleProjectDetail)
	mux.HandleFunc("/v0/admin/projects/bulk", h.handleBulkOperation)
	mux.HandleFunc("/v0/admin/projects/stream", h.handleStreamUpdates)
}

func (h *ProjectHandler) handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listWorkspaces(w, r)
	case http.MethodPost:
		h.createWorkspace(w, r)
	default:
		h.log.Warn("method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *ProjectHandler) handleProjectDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v0/admin/projects/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "project id required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getWorkspace(w, r, id)
	case http.MethodPut:
		h.updateWorkspace(w, r, id)
	case http.MethodDelete:
		h.deleteWorkspace(w, r, id)
	case http.MethodPatch:
		h.patchWorkspace(w, r, id)
	default:
		h.log.Warn("method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// listWorkspaces returns a paginated list of workspaces with filtering
func (h *ProjectHandler) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	filter := h.parseFilter(r)
	page := readIntParam(r, "page", 1)
	pageSize := readIntParam(r, "pageSize", 50)
	if pageSize > 500 {
		pageSize = 500
	}

	h.log.Debug("listing workspaces",
		"page", page,
		"pageSize", pageSize,
		"status", filter.Status,
		"search", filter.Search,
	)

	// In a real implementation, this would query the database
	// For now, return mock data
	workspaces := []db.Workspace{
		{
			ID:          "ws_001",
			Slug:        "production",
			Name:        "Production",
			Description: strPtr("Production workspace"),
			Status:      "active",
			CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "ws_002",
			Slug:        "staging",
			Name:        "Staging",
			Description: strPtr("Staging workspace"),
			Status:      "active",
			CreatedAt:   time.Now().Add(-20 * 24 * time.Hour),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "ws_003",
			Slug:        "development",
			Name:        "Development",
			Description: strPtr("Development workspace"),
			Status:      "active",
			CreatedAt:   time.Now().Add(-10 * 24 * time.Hour),
			UpdatedAt:   time.Now(),
		},
	}

	// Apply status filter
	if filter.Status != "" {
		var filtered []db.Workspace
		for _, ws := range workspaces {
			if ws.Status == filter.Status {
				filtered = append(filtered, ws)
			}
		}
		workspaces = filtered
	}

	// Apply search filter
	if filter.Search != "" {
		var filtered []db.Workspace
		searchLower := strings.ToLower(filter.Search)
		for _, ws := range workspaces {
			if strings.Contains(strings.ToLower(ws.Name), searchLower) ||
				strings.Contains(strings.ToLower(ws.Slug), searchLower) {
				filtered = append(filtered, ws)
			}
		}
		workspaces = filtered
	}

	total := len(workspaces)

	// Apply pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	pagedWorkspaces := workspaces[start:end]

	response := WorkspaceListResponse{
		Data:     pagedWorkspaces,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  end < total,
	}

	writeJSON(w, http.StatusOK, response)
}

// getWorkspace returns a single workspace by ID
func (h *ProjectHandler) getWorkspace(w http.ResponseWriter, r *http.Request, id string) {
	h.log.Debug("getting workspace", "id", id)

	// In a real implementation, this would query the database
	workspace := db.Workspace{
		ID:          id,
		Slug:        "production",
		Name:        "Production",
		Description: strPtr("Production workspace"),
		Status:      "active",
		Settings:    []byte(`{"retention_days":90}`),
		CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:   time.Now(),
	}

	writeJSON(w, http.StatusOK, workspace)
}

// createWorkspace creates a new workspace
func (h *ProjectHandler) createWorkspace(w http.ResponseWriter, r *http.Request) {
	var req WorkspaceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	if req.Slug == "" {
		req.Slug = slugify(req.Name)
	}

	h.log.Info("creating workspace",
		"name", req.Name,
		"slug", req.Slug,
	)

	// In a real implementation, this would insert into database
	workspace := db.Workspace{
		ID:          generateID("ws"),
		Slug:        req.Slug,
		Name:        req.Name,
		Description: &req.Description,
		Status:      "active",
		Settings:    req.Settings,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	writeJSON(w, http.StatusCreated, workspace)
}

// updateWorkspace fully updates a workspace
func (h *ProjectHandler) updateWorkspace(w http.ResponseWriter, r *http.Request, id string) {
	var req WorkspaceUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	h.log.Info("updating workspace", "id", id)

	// In a real implementation, this would update in database
	workspace := db.Workspace{
		ID:          id,
		Slug:        "production",
		Name:        req.Name,
		Description: &req.Description,
		Status:      req.Status,
		Settings:    req.Settings,
		UpdatedAt:   time.Now(),
	}

	writeJSON(w, http.StatusOK, workspace)
}

// patchWorkspace partially updates a workspace
func (h *ProjectHandler) patchWorkspace(w http.ResponseWriter, r *http.Request, id string) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	h.log.Info("patching workspace", "id", id, "fields", len(updates))

	// In a real implementation, this would apply partial updates
	workspace := db.Workspace{
		ID:          id,
		Slug:        "production",
		Name:        "Production",
		Description: strPtr("Production workspace"),
		Status:      "active",
		UpdatedAt:   time.Now(),
	}

	writeJSON(w, http.StatusOK, workspace)
}

// deleteWorkspace deletes a workspace
func (h *ProjectHandler) deleteWorkspace(w http.ResponseWriter, r *http.Request, id string) {
	force := r.URL.Query().Get("force") == "true"

	h.log.Info("deleting workspace", "id", id, "force", force)

	// In a real implementation, this would delete from database
	// or soft delete if force is false

	writeJSON(w, http.StatusNoContent, nil)
}

// handleBulkOperation handles bulk operations on workspaces
func (h *ProjectHandler) handleBulkOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req BulkWorkspaceRequest
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

	h.log.Info("bulk workspace operation",
		"action", req.Action,
		"count", len(req.IDs),
	)

	// Validate action
	validActions := map[string]bool{
		"activate":   true,
		"deactivate": true,
		"delete":     true,
		"archive":    true,
	}
	if !validActions[req.Action] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid action"})
		return
	}

	// In a real implementation, this would perform the bulk operation
	result := map[string]interface{}{
		"processed": len(req.IDs),
		"action":    req.Action,
		"success":   true,
	}

	writeJSON(w, http.StatusOK, result)
}

// handleStreamUpdates handles SSE stream for real-time workspace updates
func (h *ProjectHandler) handleStreamUpdates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	h.log.Info("starting workspace update stream")

	// In a real implementation, this would subscribe to workspace update events
	// For now, just send a keepalive
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.log.Error("streaming not supported")
		return
	}

	// Send initial event
	_, _ = w.Write([]byte("event: connected\ndata: {\"status\":\"connected\"}\n\n"))
	flusher.Flush()

	// Keep connection alive (in real implementation, this would push actual updates)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			h.log.Info("workspace stream closed")
			return
		case <-ticker.C:
			_, _ = w.Write([]byte("event: ping\ndata: {}\n\n"))
			flusher.Flush()
		}
	}
}

// parseFilter parses filter parameters from request
func (h *ProjectHandler) parseFilter(r *http.Request) WorkspaceFilter {
	filter := WorkspaceFilter{
		Status:    r.URL.Query().Get("status"),
		Search:    r.URL.Query().Get("search"),
		SortBy:    r.URL.Query().Get("sortBy"),
		SortOrder: r.URL.Query().Get("sortOrder"),
	}

	if filter.SortBy == "" {
		filter.SortBy = "createdAt"
	}
	if filter.SortOrder == "" {
		filter.SortOrder = "desc"
	}

	return filter
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func slugify(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

func generateID(prefix string) string {
	return prefix + "_" + strconv.FormatInt(time.Now().Unix(), 36)
}

func intPtr(i int) *int {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func readIntParam(r *http.Request, name string, fallback int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
