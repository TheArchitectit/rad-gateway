package a2a

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type TaskState string

const (
	TaskStateSubmitted     TaskState = "submitted"
	TaskStateWorking       TaskState = "working"
	TaskStateInputRequired TaskState = "input-required"
	TaskStateCompleted     TaskState = "completed"
	TaskStateCanceled      TaskState = "canceled"
	TaskStateFailed        TaskState = "failed"
)

func ValidTaskStates() []TaskState {
	return []TaskState{
		TaskStateSubmitted,
		TaskStateWorking,
		TaskStateInputRequired,
		TaskStateCompleted,
		TaskStateCanceled,
		TaskStateFailed,
	}
}

func IsValidTaskState(state TaskState) bool {
	for _, s := range ValidTaskStates() {
		if s == state {
			return true
		}
	}
	return false
}

func IsTerminalState(state TaskState) bool {
	return state == TaskStateCompleted || state == TaskStateCanceled || state == TaskStateFailed
}

func (t *Task) CanTransitionTo(target TaskState) bool {
	switch t.Status {
	case TaskStateSubmitted:
		return target == TaskStateWorking || target == TaskStateCanceled
	case TaskStateWorking:
		return target == TaskStateCompleted || target == TaskStateFailed ||
			target == TaskStateInputRequired || target == TaskStateCanceled
	case TaskStateInputRequired:
		return target == TaskStateWorking || target == TaskStateCanceled
	default:
		return false
	}
}

type Message struct {
	Role     string                 `json:"role"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Artifact struct {
	Type        string          `json:"type"`
	Content     json.RawMessage `json:"content"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
}

type Task struct {
	ID              string          `db:"id" json:"id"`
	Status          TaskState       `db:"status" json:"status"`
	SessionID       string          `db:"session_id" json:"sessionId"`
	Message         Message         `db:"message" json:"message"`
	Artifacts       []Artifact      `db:"artifacts" json:"artifacts,omitempty"`
	Metadata        json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	CreatedAt       time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time       `db:"updated_at" json:"updatedAt"`
	ExpiresAt       *time.Time      `db:"expires_at" json:"expiresAt,omitempty"`
	ParentID        *string         `db:"parent_id" json:"parentId,omitempty"`
	WorkspaceID     *string         `db:"workspace_id" json:"workspaceId,omitempty"`
	AssignedAgentID *string         `db:"assigned_agent_id" json:"assignedAgentId,omitempty"`
}

type TaskList struct {
	Items  []Task `json:"items"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// TaskFilter provides filtering options for listing tasks.
type TaskFilter struct {
	WorkspaceID string
	SessionID   string
	Status      TaskState
	Limit       int
	Offset      int
}

type SendTaskRequest struct {
	SessionID string          `json:"sessionId"`
	Message   Message         `json:"message"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

type SendTaskResponse struct {
	Task Task `json:"task"`
}

type GetTaskResponse struct {
	Task Task `json:"task"`
}

type CancelTaskResponse struct {
	Task Task `json:"task"`
}

type TaskEvent struct {
	Type      string    `json:"type"`
	TaskID    string    `json:"taskId"`
	Status    TaskState `json:"status,omitempty"`
	Artifact  *Artifact `json:"artifact,omitempty"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type TaskEventType string

const (
	TaskEventTypeStatusUpdate TaskEventType = "status"
	TaskEventTypeArtifact     TaskEventType = "artifact"
	TaskEventTypeMessage      TaskEventType = "message"
	TaskEventTypeCompleted    TaskEventType = "completed"
	TaskEventTypeFailed       TaskEventType = "failed"
)

// TaskStore defines the interface for task persistence operations.
type TaskStore interface {
	CreateTask(ctx context.Context, task *Task) error
	GetTask(ctx context.Context, id string) (*Task, error)
	UpdateTask(ctx context.Context, task *Task) error
	DeleteTask(ctx context.Context, id string) error
	ListTasks(ctx context.Context, filter TaskFilter) ([]*Task, error)
}

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrInvalidTaskState  = errors.New("invalid task state")
	ErrTaskAlreadyExists = errors.New("task already exists")
	ErrInvalidTransition = errors.New("invalid state transition")
)
