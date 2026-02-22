package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"radgateway/internal/db"
	"radgateway/internal/logger"
)

type ProjectHandler struct {
	log *slog.Logger
	db  db.Database
}

func NewProjectHandler(database db.Database) *ProjectHandler {
	return &ProjectHandler{
		log: logger.WithComponent("admin.projects"),
		db:  database,
	}
}

type WorkspaceCreateRequest struct {
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description string          `json:"description,omitempty"`
	Settings    json.RawMessage `json:"settings,omitempty"`
}

type WorkspaceUpdateRequest struct {
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Status      string          `json:"status,omitempty"`
	Settings    json.RawMessage `json:"settings,omitempty"`
}

type BulkWorkspaceRequest struct {
	IDs    []string `json:"ids"`
	Action string   `json:"action"`
}

type WorkspaceListResponse struct {
	Data     []db.Workspace `json:"data"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
	HasMore  bool           `json:"hasMore"`
}

func (h *ProjectHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v0/admin/projects", h.handleProjects)
	mux.HandleFunc("/v0/admin/projects/", h.handleProjectDetail)
	mux.HandleFunc("/v0/admin/projects/bulk", h.handleBulkOperation)
	mux.HandleFunc("/v0/admin/projects/stream", h.handleStreamUpdates)
}

func (h *ProjectHandler) handleProjects(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database not configured"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.listWorkspaces(w, r)
	case http.MethodPost:
		h.createWorkspace(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *ProjectHandler) handleProjectDetail(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database not configured"})
		return
	}
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
	case http.MethodPatch:
		h.patchWorkspace(w, r, id)
	case http.MethodDelete:
		h.deleteWorkspace(w, r, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *ProjectHandler) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	page := readIntParam(r, "page", 1)
	pageSize := readIntParam(r, "pageSize", 50)
	if pageSize > 500 {
		pageSize = 500
	}
	offset := (page - 1) * pageSize

	rows, err := h.db.Workspaces().List(r.Context(), pageSize, offset)
	if err != nil {
		h.log.Error("failed to list workspaces", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list projects"})
		return
	}

	status := r.URL.Query().Get("status")
	search := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("search")))
	filtered := make([]db.Workspace, 0, len(rows))
	for _, ws := range rows {
		if status != "" && ws.Status != status {
			continue
		}
		if search != "" {
			if !strings.Contains(strings.ToLower(ws.Name), search) && !strings.Contains(strings.ToLower(ws.Slug), search) {
				continue
			}
		}
		filtered = append(filtered, ws)
	}

	writeJSON(w, http.StatusOK, WorkspaceListResponse{
		Data:     filtered,
		Total:    len(filtered),
		Page:     page,
		PageSize: pageSize,
		HasMore:  len(rows) == pageSize,
	})
}

func (h *ProjectHandler) getWorkspace(w http.ResponseWriter, r *http.Request, id string) {
	workspace, err := h.db.Workspaces().GetByID(r.Context(), id)
	if err != nil {
		h.log.Error("failed to get workspace", "id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get project"})
		return
	}
	if workspace == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}
	writeJSON(w, http.StatusOK, workspace)
}

func (h *ProjectHandler) createWorkspace(w http.ResponseWriter, r *http.Request) {
	var req WorkspaceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if strings.TrimSpace(req.Slug) == "" {
		req.Slug = slugify(req.Name)
	}
	settings := []byte("{}")
	if len(req.Settings) > 0 {
		settings = req.Settings
	}
	desc := strings.TrimSpace(req.Description)
	now := time.Now().UTC()
	workspace := &db.Workspace{
		ID:          generateID("ws"),
		Slug:        req.Slug,
		Name:        req.Name,
		Description: nullableString(desc),
		Status:      "active",
		Settings:    settings,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := h.db.Workspaces().Create(r.Context(), workspace); err != nil {
		h.log.Error("failed to create workspace", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create project"})
		return
	}
	writeJSON(w, http.StatusCreated, workspace)
}

func (h *ProjectHandler) updateWorkspace(w http.ResponseWriter, r *http.Request, id string) {
	var req WorkspaceUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	workspace, err := h.db.Workspaces().GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get project"})
		return
	}
	if workspace == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}
	if req.Name != "" {
		workspace.Name = req.Name
	}
	if req.Description != "" {
		workspace.Description = nullableString(strings.TrimSpace(req.Description))
	}
	if req.Status != "" {
		workspace.Status = req.Status
	}
	if len(req.Settings) > 0 {
		workspace.Settings = req.Settings
	}
	workspace.UpdatedAt = time.Now().UTC()
	if err := h.db.Workspaces().Update(r.Context(), workspace); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update project"})
		return
	}
	writeJSON(w, http.StatusOK, workspace)
}

func (h *ProjectHandler) patchWorkspace(w http.ResponseWriter, r *http.Request, id string) {
	h.updateWorkspace(w, r, id)
}

func (h *ProjectHandler) deleteWorkspace(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.db.Workspaces().Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete project"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProjectHandler) handleBulkOperation(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "database not configured"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req BulkWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if len(req.IDs) == 0 || req.Action == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ids and action are required"})
		return
	}
	processed := 0
	for _, id := range req.IDs {
		switch req.Action {
		case "delete":
			if err := h.db.Workspaces().Delete(r.Context(), id); err == nil {
				processed++
			}
		case "activate", "deactivate", "archive":
			ws, err := h.db.Workspaces().GetByID(r.Context(), id)
			if err != nil || ws == nil {
				continue
			}
			switch req.Action {
			case "activate":
				ws.Status = "active"
			case "deactivate":
				ws.Status = "inactive"
			case "archive":
				ws.Status = "archived"
			}
			ws.UpdatedAt = time.Now().UTC()
			if err := h.db.Workspaces().Update(r.Context(), ws); err == nil {
				processed++
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"processed": processed, "action": req.Action, "success": true})
}

func (h *ProjectHandler) handleStreamUpdates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}
	_, _ = w.Write([]byte("event: connected\ndata: {\"status\":\"connected\"}\n\n"))
	flusher.Flush()
	<-r.Context().Done()
}

func slugify(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
