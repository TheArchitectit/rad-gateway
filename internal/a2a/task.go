package a2a

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// TaskStatus represents the status of an A2A task.
type TaskStatus string

const (
	// TaskStatusPending indicates the task is waiting to be processed.
	TaskStatusPending TaskStatus = "pending"
	// TaskStatusWorking indicates the task is being processed.
	TaskStatusWorking TaskStatus = "working"
	// TaskStatusInputRequired indicates the task requires additional input.
	TaskStatusInputRequired TaskStatus = "input-required"
	// TaskStatusCompleted indicates the task completed successfully.
	TaskStatusCompleted TaskStatus = "completed"
	// TaskStatusFailed indicates the task failed.
	TaskStatusFailed TaskStatus = "failed"
	// TaskStatusCancelled indicates the task was cancelled.
	TaskStatusCancelled TaskStatus = "cancelled"
)

// IsValidTaskStatus checks if a status string is valid.
func IsValidTaskStatus(status string) bool {
	switch TaskStatus(status) {
	case TaskStatusPending, TaskStatusWorking, TaskStatusInputRequired,
		TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled:
		return true
	}
	return false
}

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

// Part represents content within an artifact.
type Part struct {
	// Type is the content type (e.g., "text", "file", "data").
	Type string `json:"type"`
	// Text is the text content (when type is "text").
	Text string `json:"text,omitempty"`
}

// Message represents communication in task history.
type Message struct {
	// Role is the sender role (e.g., "user", "agent").
	Role string `json:"role"`
	// Content is the message content.
	Content string `json:"content"`
	// Parts are structured content parts.
	Parts []Part `json:"parts,omitempty"`
	// Metadata is additional message metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Artifact represents task output.
type Artifact struct {
	// ID is the unique identifier for the artifact.
	ID string `json:"id"`
	// Type is the artifact type.
	Type string `json:"type"`
	// Parts are the content parts of the artifact.
	Parts []Part `json:"parts"`
	// Metadata is additional artifact metadata.
	Metadata interface{} `json:"metadata,omitempty"`
	// Name is the artifact name.
	Name string `json:"name,omitempty"`
	// Description is the artifact description.
	Description string `json:"description,omitempty"`
	// Content is the raw content (for backward compatibility).
	Content json.RawMessage `json:"content,omitempty"`
}

// Task represents an A2A task.
type Task struct {
	// ID is the unique task identifier.
	ID string `db:"id" json:"id"`
	// Status is the current task status.
	Status TaskState `db:"status" json:"status"`
	// SessionID is the session identifier for grouping tasks.
	SessionID string `db:"session_id" json:"sessionId"`
	// Message is the initial task message.
	Message Message `db:"message" json:"message"`
	// Artifacts are the task outputs.
	Artifacts []Artifact `db:"artifacts" json:"artifacts,omitempty"`
	// History is the message history for the task.
	History []Message `db:"history" json:"history,omitempty"`
	// Metadata is additional task metadata.
	Metadata json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	// CreatedAt is when the task was created.
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	// UpdatedAt is when the task was last updated.
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
	// CompletedAt is when the task was completed (if applicable).
	CompletedAt *time.Time `db:"completed_at" json:"completedAt,omitempty"`
	// ExpiresAt is when the task expires.
	ExpiresAt *time.Time `db:"expires_at" json:"expiresAt,omitempty"`
	// ParentID is the parent task ID.
	ParentID *string `db:"parent_id" json:"parentId,omitempty"`
	// WorkspaceID is the workspace ID.
	WorkspaceID *string `db:"workspace_id" json:"workspaceId,omitempty"`
	// AssignedAgentID is the assigned agent ID.
	AssignedAgentID *string `db:"assigned_agent_id" json:"assignedAgentId,omitempty"`
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

// SendTaskRequest is the request body for task creation.
type SendTaskRequest struct {
	// ID is an optional client-provided task ID.
	ID string `json:"id,omitempty"`
	// SessionID is the session identifier.
	SessionID string `json:"sessionId"`
	// SkillID is the ID of the skill to invoke.
	SkillID string `json:"skillId"`
	// Message is the initial message for the task.
	Message Message `json:"message"`
	// Metadata is additional request metadata.
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// Error represents an A2A error response.
type Error struct {
	// Code is the error code.
	Code int `json:"code"`
	// Message is the human-readable error message.
	Message string `json:"message"`
}

// SendTaskResponse is the response from task creation.
type SendTaskResponse struct {
	// Task is the created task (nil if error).
	Task *Task `json:"task,omitempty"`
	// Error is the error information (nil if successful).
	Error *Error `json:"error,omitempty"`
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
