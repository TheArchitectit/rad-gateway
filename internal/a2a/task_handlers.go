package a2a

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (h *Handlers) handleSendTask(w http.ResponseWriter, r *http.Request) {
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
		h.log.Error("failed to update task", "id", task.ID, "error", err)
		writeTaskError(w, http.StatusInternalServerError, "failed to update task")
		return
	}

	payload, err := json.Marshal(map[string]string{"text": fmt.Sprintf("Task processed: %s", req.Message.Content)})
	if err != nil {
		writeTaskError(w, http.StatusInternalServerError, "failed to build artifact")
		return
	}

	task.Artifacts = []Artifact{{
		Type:    "text",
		Content: payload,
		Name:    "result",
	}}
	task.Status = TaskStateCompleted
	task.UpdatedAt = time.Now().UTC()
	if err := h.taskStore.UpdateTask(r.Context(), task); err != nil {
		h.log.Error("failed to complete task", "id", task.ID, "error", err)
		writeTaskError(w, http.StatusInternalServerError, "failed to complete task")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(SendTaskResponse{Task: *task})
}

func (h *Handlers) handleSendTaskSubscribe(w http.ResponseWriter, r *http.Request) {
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

	payload, _ := json.Marshal(map[string]string{"text": fmt.Sprintf("Task processed: %s", req.Message.Content)})
	artifact := Artifact{Type: "text", Content: payload, Name: "result"}
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
}

func (h *Handlers) handleTaskByID(w http.ResponseWriter, r *http.Request) {
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
		writeTaskError(w, http.StatusInternalServerError, "failed to get task")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(GetTaskResponse{Task: *task})
}

func (h *Handlers) handleCancelTask(w http.ResponseWriter, r *http.Request, taskID string) {
	if r.Method != http.MethodPost {
		writeTaskError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	task, err := h.taskStore.GetTask(r.Context(), taskID)
	if err != nil {
		if err == ErrTaskNotFound {
			writeTaskError(w, http.StatusNotFound, "task not found")
			return
		}
		writeTaskError(w, http.StatusInternalServerError, "failed to get task")
		return
	}

	if IsTerminalState(task.Status) {
		writeTaskError(w, http.StatusConflict, "task already terminal")
		return
	}

	task.Status = TaskStateCanceled
	task.UpdatedAt = time.Now().UTC()
	if err := h.taskStore.UpdateTask(r.Context(), task); err != nil {
		writeTaskError(w, http.StatusInternalServerError, "failed to cancel task")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(CancelTaskResponse{Task: *task})
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
