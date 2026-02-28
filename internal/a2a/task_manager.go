package a2a

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TaskManager manages A2A task lifecycle with state transitions.
type TaskManager struct {
	mu    sync.RWMutex
	tasks map[string]*Task
	store TaskStore // optional, for persistence
}

// NewTaskManager creates a new TaskManager instance.
func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: make(map[string]*Task),
	}
}

// NewTaskManagerWithStore creates a new TaskManager with an optional persistence store.
func NewTaskManagerWithStore(store TaskStore) *TaskManager {
	return &TaskManager{
		tasks: make(map[string]*Task),
		store: store,
	}
}

// CreateTask creates a new task with the provided request.
// Generates a UUID if ID is empty, sets initial status to pending,
// and initializes timestamps.
func (tm *TaskManager) CreateTask(ctx context.Context, req SendTaskRequest) (*Task, error) {
	taskID := req.ID
	if taskID == "" {
		taskID = uuid.New().String()
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if task already exists
	if _, exists := tm.tasks[taskID]; exists {
		return nil, fmt.Errorf("%w: task with ID %s", ErrTaskAlreadyExists, taskID)
	}

	now := time.Now().UTC()

	task := &Task{
		ID:        taskID,
		Status:    TaskStateSubmitted, // Map pending -> submitted
		SessionID: req.SessionID,
		Message:   req.Message,
		History:   []Message{req.Message},
		Metadata:  req.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	tm.tasks[taskID] = task

	// Persist if store is configured
	if tm.store != nil {
		if err := tm.store.CreateTask(ctx, task); err != nil {
			// Rollback in-memory creation on persistence failure
			delete(tm.tasks, taskID)
			return nil, fmt.Errorf("failed to persist task: %w", err)
		}
	}

	return task, nil
}

// GetTask retrieves a task by its ID.
// Returns ErrTaskNotFound if the task does not exist.
func (tm *TaskManager) GetTask(ctx context.Context, taskID string) (*Task, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrTaskNotFound, taskID)
	}

	return task, nil
}

// UpdateTask updates an existing task.
// The task ID must exist in the manager.
func (tm *TaskManager) UpdateTask(ctx context.Context, task *Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.tasks[task.ID]; !exists {
		return fmt.Errorf("%w: %s", ErrTaskNotFound, task.ID)
	}

	task.UpdatedAt = time.Now().UTC()
	tm.tasks[task.ID] = task

	// Persist if store is configured
	if tm.store != nil {
		if err := tm.store.UpdateTask(ctx, task); err != nil {
			return fmt.Errorf("failed to persist task update: %w", err)
		}
	}

	return nil
}

// taskStatusToTaskState converts TaskStatus to TaskState.
func taskStatusToTaskState(status TaskStatus) TaskState {
	switch status {
	case TaskStatusPending:
		return TaskStateSubmitted
	case TaskStatusWorking:
		return TaskStateWorking
	case TaskStatusInputRequired:
		return TaskStateInputRequired
	case TaskStatusCompleted:
		return TaskStateCompleted
	case TaskStatusFailed:
		return TaskStateFailed
	case TaskStatusCancelled:
		return TaskStateCanceled
	default:
		return TaskStateSubmitted
	}
}

// TransitionState transitions a task to a new state.
// Validates that the transition is allowed according to the state machine.
func (tm *TaskManager) TransitionState(ctx context.Context, taskID string, newStatus TaskStatus) error {
	targetState := taskStatusToTaskState(newStatus)

	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("%w: %s", ErrTaskNotFound, taskID)
	}

	// Validate state transition
	if !task.CanTransitionTo(targetState) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, task.Status, targetState)
	}

	// Perform transition
	task.Status = targetState
	task.UpdatedAt = time.Now().UTC()

	// Set CompletedAt if transitioning to a terminal state
	if IsTerminalState(targetState) {
		now := time.Now().UTC()
		task.CompletedAt = &now
	}

	// Persist if store is configured
	if tm.store != nil {
		if err := tm.store.UpdateTask(ctx, task); err != nil {
			return fmt.Errorf("failed to persist state transition: %w", err)
		}
	}

	return nil
}

// CancelTask cancels a task by transitioning it to the cancelled state.
// This is a convenience helper that calls TransitionState.
func (tm *TaskManager) CancelTask(ctx context.Context, taskID string) error {
	return tm.TransitionState(ctx, taskID, TaskStatusCancelled)
}

// ListTasks returns all tasks managed by the TaskManager.
// The order of tasks in the returned slice is not guaranteed.
func (tm *TaskManager) ListTasks(ctx context.Context) []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// DeleteTask removes a task from the manager.
// This is primarily useful for testing and cleanup operations.
func (tm *TaskManager) DeleteTask(ctx context.Context, taskID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.tasks[taskID]; !exists {
		return fmt.Errorf("%w: %s", ErrTaskNotFound, taskID)
	}

	delete(tm.tasks, taskID)

	// Persist deletion if store is configured
	if tm.store != nil {
		if err := tm.store.DeleteTask(ctx, taskID); err != nil {
			return fmt.Errorf("failed to persist task deletion: %w", err)
		}
	}

	return nil
}

// Count returns the number of tasks in the manager.
func (tm *TaskManager) Count() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return len(tm.tasks)
}

// Clear removes all tasks from the manager.
// Primarily useful for testing.
func (tm *TaskManager) Clear() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.tasks = make(map[string]*Task)
}
