// Package a2a provides A2A (Agent-to-Agent) protocol support for RAD Gateway.
package a2a

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"radgateway/internal/core"
	"radgateway/internal/logger"
)

// Handlers provides HTTP handlers for A2A Model Card operations.
type Handlers struct {
	repo      Repository
	taskStore TaskStore
	task      *TaskHandlers
	log       *slog.Logger
}

// NewHandlers creates new A2A handlers with the given repository.
func NewHandlers(repo Repository) *Handlers {
	return &Handlers{
		repo:      repo,
		taskStore: nil,
		task:      nil,
		log:       logger.WithComponent("a2a_handlers"),
	}
}

func NewHandlersWithTaskStore(repo Repository, taskStore TaskStore, gateway *core.Gateway) *Handlers {
	var taskHandlers *TaskHandlers
	if taskStore != nil {
		if gateway != nil {
			taskHandlers = NewTaskHandlersWithGateway(taskStore, gateway)
		} else {
			taskHandlers = NewTaskHandlers(taskStore, nil)
		}
	}
	return &Handlers{
		repo:      repo,
		taskStore: taskStore,
		task:      taskHandlers,
		log:       logger.WithComponent("a2a_handlers"),
	}
}

// Register registers A2A routes on the provided mux.
func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/a2a/model-cards", h.handleModelCards)
	mux.HandleFunc("/v1/a2a/model-cards/", h.handleModelCardByID)
	mux.HandleFunc("/v1/a2a/projects/", h.handleProjectModelCards)
	mux.HandleFunc("/a2a/model-cards", h.handleModelCards)
	mux.HandleFunc("/a2a/model-cards/", h.handleModelCardByID)
	mux.HandleFunc("/a2a/projects/", h.handleProjectModelCards)
	if h.task != nil {
		mux.HandleFunc("/v1/a2a/tasks/send", h.task.handleSendTask)
		mux.HandleFunc("/v1/a2a/tasks/sendSubscribe", h.task.handleSendTaskSubscribe)
		mux.HandleFunc("/v1/a2a/tasks/", h.task.handleTaskByID)
		mux.HandleFunc("/a2a/tasks/send", h.task.handleSendTask)
		mux.HandleFunc("/a2a/tasks/sendSubscribe", h.task.handleSendTaskSubscribe)
		mux.HandleFunc("/a2a/tasks/", h.task.handleTaskByID)
	}
}

// handleModelCards handles listing and creating model cards.
func (h *Handlers) handleModelCards(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listModelCards(w, r)
	case http.MethodPost:
		h.createModelCard(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
	}
}

// handleModelCardByID handles operations on a single model card.
func (h *Handlers) handleModelCardByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /v1/a2a/model-cards/{id}
	path := strings.TrimPrefix(r.URL.Path, "/v1/a2a/model-cards/")
	if path == r.URL.Path {
		path = strings.TrimPrefix(r.URL.Path, "/a2a/model-cards/")
	}
	id := strings.Split(path, "/")[0]

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "model card ID required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getModelCard(w, r, id)
	case http.MethodPut:
		h.updateModelCard(w, r, id)
	case http.MethodDelete:
		h.deleteModelCard(w, r, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
	}
}

// handleProjectModelCards handles listing model cards for a project.
func (h *Handlers) handleProjectModelCards(w http.ResponseWriter, r *http.Request) {
	// Path: /v1/a2a/projects/{project_id}/model-cards
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	// Extract project ID from path
	path := strings.TrimPrefix(r.URL.Path, "/v1/a2a/projects/")
	if path == r.URL.Path {
		path = strings.TrimPrefix(r.URL.Path, "/a2a/projects/")
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "model-cards" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid path"})
		return
	}

	projectID := parts[0]
	if projectID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "project ID required"})
		return
	}

	h.listProjectModelCards(w, r, projectID)
}

// createModelCard handles POST /v1/a2a/model-cards.
func (h *Handlers) createModelCard(w http.ResponseWriter, r *http.Request) {
	var req CreateModelCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("failed to decode request body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	// Validate required fields
	if req.WorkspaceID == "" || req.Name == "" || req.Slug == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "workspace_id, name, and slug are required"})
		return
	}

	card := &ModelCard{
		WorkspaceID: req.WorkspaceID,
		UserID:      req.UserID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Card:        req.Card,
	}

	if err := h.repo.Create(r.Context(), card); err != nil {
		h.log.Error("failed to create model card", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to create model card"})
		return
	}

	h.log.Info("created model card", "id", card.ID, "workspace_id", card.WorkspaceID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(card)
}

// getModelCard handles GET /v1/a2a/model-cards/{id}.
func (h *Handlers) getModelCard(w http.ResponseWriter, r *http.Request, id string) {
	card, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.log.Warn("failed to get model card", "id", id, "error", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "model card not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

// listModelCards handles GET /v1/a2a/model-cards with optional filters.
func (h *Handlers) listModelCards(w http.ResponseWriter, r *http.Request) {
	// For now, require workspace_id query param
	workspaceID := r.URL.Query().Get("workspace_id")
	if workspaceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "workspace_id query parameter required"})
		return
	}

	h.listProjectModelCards(w, r, workspaceID)
}

// listProjectModelCards handles listing model cards for a project.
func (h *Handlers) listProjectModelCards(w http.ResponseWriter, r *http.Request, projectID string) {
	cards, err := h.repo.GetByProject(r.Context(), projectID)
	if err != nil {
		h.log.Error("failed to list project model cards", "project_id", projectID, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to list model cards"})
		return
	}

	// Return empty array instead of nil
	if cards == nil {
		cards = []ModelCard{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ModelCardList{
		Items:  cards,
		Total:  len(cards),
		Limit:  len(cards),
		Offset: 0,
	})
}

// updateModelCard handles PUT /v1/a2a/model-cards/{id}.
func (h *Handlers) updateModelCard(w http.ResponseWriter, r *http.Request, id string) {
	var req UpdateModelCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("failed to decode request body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	// Get existing card
	card, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.log.Warn("failed to get model card for update", "id", id, "error", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "model card not found"})
		return
	}

	// Apply updates
	if req.Name != nil {
		card.Name = *req.Name
	}
	if req.Description != nil {
		card.Description = req.Description
	}
	if req.Card != nil {
		card.Card = *req.Card
	}
	if req.Status != nil {
		card.Status = *req.Status
	}

	if err := h.repo.Update(r.Context(), card); err != nil {
		h.log.Error("failed to update model card", "id", id, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to update model card"})
		return
	}

	h.log.Info("updated model card", "id", card.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

// deleteModelCard handles DELETE /v1/a2a/model-cards/{id}.
func (h *Handlers) deleteModelCard(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.log.Warn("failed to delete model card", "id", id, "error", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "model card not found"})
		return
	}

	h.log.Info("deleted model card", "id", id)

	w.WriteHeader(http.StatusNoContent)
}

// GetBySlugHandler returns an http.HandlerFunc for getting a model card by slug.
// This can be mounted at a specific route.
func (h *Handlers) GetBySlugHandler(workspaceID, slug string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
			return
		}

		card, err := h.repo.GetBySlug(r.Context(), workspaceID, slug)
		if err != nil {
			h.log.Warn("failed to get model card by slug", "workspace_id", workspaceID, "slug", slug, "error", err)
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "model card not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(card)
	}
}

// HealthCheck returns a simple health check handler for A2A services.
func (h *Handlers) HealthCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": "a2a",
		})
	}
}
