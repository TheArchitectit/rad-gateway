package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"radgateway/internal/core"
	"radgateway/internal/logger"
	"radgateway/internal/models"
)

// TaskExecutor defines the interface for executing tasks via the gateway
type TaskExecutor interface {
	Execute(ctx context.Context, apiType string, model string, payload any) (models.ProviderResult, error)
}

type gatewayExecutor struct {
	gateway *core.Gateway
}

func (g *gatewayExecutor) Execute(ctx context.Context, apiType string, model string, payload any) (models.ProviderResult, error) {
	result, _, err := g.gateway.Handle(ctx, apiType, model, payload)
	return result, err
}

type TaskHandlers struct {
	taskStore TaskStore
	executor  TaskExecutor
	log       *slog.Logger
}

func NewTaskHandlers(taskStore TaskStore, executor TaskExecutor) *TaskHandlers {
	return &TaskHandlers{
		taskStore: taskStore,
		executor:  executor,
		log:       logger.WithComponent("a2a_task_handlers"),
	}
}

func NewTaskHandlersWithGateway(taskStore TaskStore, gateway *core.Gateway) *TaskHandlers {
	return NewTaskHandlers(taskStore, &gatewayExecutor{gateway: gateway})
}

func (h *TaskHandlers) handleSendTask(w http.ResponseWriter, r *http.Request) {
	if h.taskStore == nil {
		writeTaskError(w, http.StatusServiceUnavailable, "task store not configured")
		return
	}
	if r.Method != http.MethodPost {
		writeTaskError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SendTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeTaskError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SessionID == "" || strings.TrimSpace(req.Message.Content) == "" {
		writeTaskError(w, http.StatusBadRequest, "sessionId and message.content are required")
		return
	}

	now := time.Now().UTC()
	task := &Task{
		ID:        generateID(),
		Status:    TaskStateSubmitted,
		SessionID: req.SessionID,
		Message:   req.Message,
		Metadata:  req.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.taskStore.CreateTask(r.Context(), task); err != nil {
		h.log.Error("failed to create task", "error", err)
		writeTaskError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	task.Status = TaskStateWorking
	task.UpdatedAt = time.Now().UTC()
	if err := h.taskStore.UpdateTask(r.Context(), task); err != nil {
		h.log.Error("failed to update task status", "id", task.ID, "error", err)
		writeTaskError(w, http.StatusInternalServerError, "failed to update task")
		return
	}

	result, err := h.executeTask(r.Context(), task, req)
	if err != nil {
		h.log.Error("task execution failed", "id", task.ID, "error", err)
		task.Status = TaskStateFailed
		task.UpdatedAt = time.Now().UTC()
		_ = h.taskStore.UpdateTask(r.Context(), task)
		writeTaskError(w, http.StatusInternalServerError, "task execution failed")
		return
	}

	artifact := h.createArtifactFromResult(result)
	task.Artifacts = []Artifact{artifact}
	task.Status = TaskStateCompleted
	task.UpdatedAt = time.Now().UTC()

	if err := h.taskStore.UpdateTask(r.Context(), task); err != nil {
		h.log.Error("failed to complete task", "id", task.ID, "error", err)
		writeTaskError(w, http.StatusInternalServerError, "failed to complete task")
		return
	}

	h.log.Info("task completed",
		"id", task.ID,
		"session_id", task.SessionID,
		"status", task.Status,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(SendTaskResponse{Task: task})
}

func (h *TaskHandlers) handleSendTaskSubscribe(w http.ResponseWriter, r *http.Request) {
	if h.taskStore == nil {
		writeTaskError(w, http.StatusServiceUnavailable, "task store not configured")
		return
	}
	if r.Method != http.MethodPost {
		writeTaskError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SendTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeTaskError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SessionID == "" || strings.TrimSpace(req.Message.Content) == "" {
		writeTaskError(w, http.StatusBadRequest, "sessionId and message.content are required")
		return
	}

	now := time.Now().UTC()
	task := &Task{
		ID:        generateID(),
		Status:    TaskStateSubmitted,
		SessionID: req.SessionID,
		Message:   req.Message,
		Metadata:  req.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.taskStore.CreateTask(r.Context(), task); err != nil {
		writeTaskError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeTaskError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	sendTaskEvent(w, flusher, TaskEvent{
		Type:      string(TaskEventTypeStatusUpdate),
		TaskID:    task.ID,
		Status:    TaskStateSubmitted,
		Message:   "task submitted",
		Timestamp: time.Now().UTC(),
	})

	task.Status = TaskStateWorking
	task.UpdatedAt = time.Now().UTC()
	_ = h.taskStore.UpdateTask(r.Context(), task)

	sendTaskEvent(w, flusher, TaskEvent{
		Type:      string(TaskEventTypeStatusUpdate),
		TaskID:    task.ID,
		Status:    TaskStateWorking,
		Message:   "task processing",
		Timestamp: time.Now().UTC(),
	})

	resultChan := make(chan struct {
		result models.ProviderResult
		err    error
	}, 1)

	go func() {
		result, err := h.executeTask(context.Background(), task, req)
		resultChan <- struct {
			result models.ProviderResult
			err    error
		}{result, err}
	}()

	res := <-resultChan

	if res.err != nil {
		h.log.Error("task execution failed", "id", task.ID, "error", res.err)
		task.Status = TaskStateFailed
		task.UpdatedAt = time.Now().UTC()
		_ = h.taskStore.UpdateTask(r.Context(), task)

		sendTaskEvent(w, flusher, TaskEvent{
			Type:      string(TaskEventTypeFailed),
			TaskID:    task.ID,
			Status:    TaskStateFailed,
			Message:   "task execution failed: " + res.err.Error(),
			Timestamp: time.Now().UTC(),
		})
		return
	}

	artifact := h.createArtifactFromResult(res.result)
	task.Artifacts = []Artifact{artifact}
	task.Status = TaskStateCompleted
	task.UpdatedAt = time.Now().UTC()
	_ = h.taskStore.UpdateTask(r.Context(), task)

	sendTaskEvent(w, flusher, TaskEvent{
		Type:      string(TaskEventTypeArtifact),
		TaskID:    task.ID,
		Artifact:  &artifact,
		Timestamp: time.Now().UTC(),
	})

	sendTaskEvent(w, flusher, TaskEvent{
		Type:      string(TaskEventTypeCompleted),
		TaskID:    task.ID,
		Status:    TaskStateCompleted,
		Message:   "task completed",
		Timestamp: time.Now().UTC(),
	})

	h.log.Info("streaming task completed",
		"id", task.ID,
		"session_id", task.SessionID,
	)
}

func (h *TaskHandlers) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	if h.taskStore == nil {
		writeTaskError(w, http.StatusServiceUnavailable, "task store not configured")
		return
	}

	taskID, action := parseTaskPath(r.URL.Path)
	if taskID == "" {
		writeTaskError(w, http.StatusBadRequest, "task id required")
		return
	}

	if action == "cancel" {
		h.handleCancelTask(w, r, taskID)
		return
	}

	if r.Method != http.MethodGet {
		writeTaskError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	task, err := h.taskStore.GetTask(r.Context(), taskID)
	if err != nil {
		if err == ErrTaskNotFound {
			writeTaskError(w, http.StatusNotFound, "task not found")
			return
		}
		h.log.Error("failed to get task", "id", taskID, "error", err)
		writeTaskError(w, http.StatusInternalServerError, "failed to get task")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(GetTaskResponse{Task: *task})
}

// HandleCancelTask handles POST /a2a/tasks/cancel with taskId in request body.
func (h *TaskHandlers) HandleCancelTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeTaskError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		TaskID string `json:"taskId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("failed to decode cancel task request", "error", err)
		writeTaskError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TaskID == "" {
		writeTaskError(w, http.StatusBadRequest, "taskId is required")
		return
	}

	h.handleCancelTaskWithID(w, r, req.TaskID)
}

func (h *TaskHandlers) handleCancelTask(w http.ResponseWriter, r *http.Request, taskID string) {
	if r.Method != http.MethodPost {
		writeTaskError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	h.handleCancelTaskWithID(w, r, taskID)
}

func (h *TaskHandlers) handleCancelTaskWithID(w http.ResponseWriter, r *http.Request, taskID string) {
	task, err := h.taskStore.GetTask(r.Context(), taskID)
	if err != nil {
		if err == ErrTaskNotFound {
			writeTaskError(w, http.StatusNotFound, "task not found")
			return
		}
		h.log.Error("failed to get task for cancel", "id", taskID, "error", err)
		writeTaskError(w, http.StatusInternalServerError, "failed to get task")
		return
	}

	if !task.CanTransitionTo(TaskStateCanceled) {
		h.log.Warn("invalid cancel transition", "id", taskID, "current_status", task.Status)
		writeTaskError(w, http.StatusConflict, fmt.Sprintf("cannot cancel task in %s state", task.Status))
		return
	}

	task.Status = TaskStateCanceled
	task.UpdatedAt = time.Now().UTC()
	if err := h.taskStore.UpdateTask(r.Context(), task); err != nil {
		h.log.Error("failed to cancel task", "id", taskID, "error", err)
		writeTaskError(w, http.StatusInternalServerError, "failed to cancel task")
		return
	}

	h.log.Info("task canceled", "id", taskID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(CancelTaskResponse{Task: *task})
}

func (h *TaskHandlers) executeTask(ctx context.Context, task *Task, req SendTaskRequest) (models.ProviderResult, error) {
	if h.executor == nil {
		return models.ProviderResult{}, fmt.Errorf("task executor not configured")
	}

	apiType := "chat"
	model := "gpt-4o-mini"

	if len(req.Metadata) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(req.Metadata, &metadata); err == nil {
			if t, ok := metadata["api_type"].(string); ok && t != "" {
				apiType = t
			}
			if m, ok := metadata["model"].(string); ok && m != "" {
				model = m
			}
		}
	}

	payload := models.ChatCompletionRequest{
		Model: model,
		Messages: []models.Message{
			{Role: "user", Content: task.Message.Content},
		},
	}

	if len(req.Metadata) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(req.Metadata, &metadata); err == nil {
			if systemMsg, ok := metadata["system_message"].(string); ok && systemMsg != "" {
				payload.Messages = append([]models.Message{
					{Role: "system", Content: systemMsg},
				}, payload.Messages...)
			}
		}
	}

	return h.executor.Execute(ctx, apiType, model, payload)
}

func (h *TaskHandlers) createArtifactFromResult(result models.ProviderResult) Artifact {
	content, _ := json.Marshal(map[string]interface{}{
		"model":    result.Model,
		"provider": result.Provider,
		"payload":  result.Payload,
		"usage":    result.Usage,
	})

	return Artifact{
		Type:        "llm_result",
		Content:     content,
		Name:        "result",
		Description: fmt.Sprintf("Generated by %s/%s", result.Provider, result.Model),
	}
}

func parseTaskPath(path string) (string, string) {
	trimmed := strings.TrimPrefix(path, "/a2a/tasks/")
	if trimmed == path {
		trimmed = strings.TrimPrefix(path, "/v1/a2a/tasks/")
	}
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", ""
	}
	if len(parts) > 1 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}

func sendTaskEvent(w http.ResponseWriter, flusher http.Flusher, event TaskEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func writeTaskError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
