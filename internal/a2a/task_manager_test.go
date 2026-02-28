package a2a

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTaskStore is a mock implementation of TaskStore for testing
type mockTaskStore struct {
	tasks map[string]*Task
}

func newMockTaskStore() *mockTaskStore {
	return &mockTaskStore{
		tasks: make(map[string]*Task),
	}
}

func (m *mockTaskStore) CreateTask(ctx context.Context, task *Task) error {
	if _, exists := m.tasks[task.ID]; exists {
		return ErrTaskAlreadyExists
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskStore) GetTask(ctx context.Context, id string) (*Task, error) {
	task, exists := m.tasks[id]
	if !exists {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

func (m *mockTaskStore) UpdateTask(ctx context.Context, task *Task) error {
	if _, exists := m.tasks[task.ID]; !exists {
		return ErrTaskNotFound
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskStore) DeleteTask(ctx context.Context, id string) error {
	if _, exists := m.tasks[id]; !exists {
		return ErrTaskNotFound
	}
	delete(m.tasks, id)
	return nil
}

func (m *mockTaskStore) ListTasks(ctx context.Context, filter TaskFilter) ([]*Task, error) {
	tasks := make([]*Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func TestNewTaskManager(t *testing.T) {
	tm := NewTaskManager()
	require.NotNil(t, tm)
	assert.Equal(t, 0, tm.Count())
}

func TestNewTaskManagerWithStore(t *testing.T) {
	store := newMockTaskStore()
	tm := NewTaskManagerWithStore(store)
	require.NotNil(t, tm)
	assert.Equal(t, 0, tm.Count())
}

func TestTaskManager_CreateTask(t *testing.T) {
	tests := []struct {
		name    string
		req     SendTaskRequest
		wantErr bool
		checkFn func(t *testing.T, task *Task)
	}{
		{
			name: "create task with generated ID",
			req: SendTaskRequest{
				SessionID: "session-1",
				Message: Message{
					Role:    "user",
					Content: "Hello",
				},
			},
			wantErr: false,
			checkFn: func(t *testing.T, task *Task) {
				assert.NotEmpty(t, task.ID)
				assert.Equal(t, TaskStateSubmitted, task.Status)
				assert.Equal(t, "session-1", task.SessionID)
				assert.Equal(t, "user", task.Message.Role)
				assert.Equal(t, "Hello", task.Message.Content)
				assert.NotZero(t, task.CreatedAt)
				assert.NotZero(t, task.UpdatedAt)
				assert.Equal(t, task.CreatedAt, task.UpdatedAt)
				assert.Nil(t, task.CompletedAt)
				assert.Len(t, task.History, 1)
			},
		},
		{
			name: "create task with provided ID",
			req: SendTaskRequest{
				ID:        "custom-task-id",
				SessionID: "session-2",
				Message: Message{
					Role:    "agent",
					Content: "Process this",
				},
			},
			wantErr: false,
			checkFn: func(t *testing.T, task *Task) {
				assert.Equal(t, "custom-task-id", task.ID)
				assert.Equal(t, TaskStateSubmitted, task.Status)
			},
		},
		{
			name: "create task with metadata",
			req: SendTaskRequest{
				SessionID: "session-3",
				Message:   Message{Role: "user", Content: "Test"},
				Metadata:  []byte(`{"key":"value"}`),
			},
			wantErr: false,
			checkFn: func(t *testing.T, task *Task) {
				assert.Equal(t, json.RawMessage(`{"key":"value"}`), task.Metadata)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTaskManager()
			ctx := context.Background()

			task, err := tm.CreateTask(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, task)

			if tt.checkFn != nil {
				tt.checkFn(t, task)
			}
		})
	}
}

func TestTaskManager_CreateTask_DuplicateID(t *testing.T) {
	tm := NewTaskManager()
	ctx := context.Background()

	req := SendTaskRequest{
		ID:        "duplicate-id",
		SessionID: "session-1",
		Message:   Message{Role: "user", Content: "First"},
	}

	// Create first task
	task1, err := tm.CreateTask(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "duplicate-id", task1.ID)

	// Try to create second task with same ID
	req2 := SendTaskRequest{
		ID:        "duplicate-id",
		SessionID: "session-2",
		Message:   Message{Role: "user", Content: "Second"},
	}

	_, err = tm.CreateTask(ctx, req2)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrTaskAlreadyExists))
}

func TestTaskManager_CreateTask_WithStore(t *testing.T) {
	store := newMockTaskStore()
	tm := NewTaskManagerWithStore(store)
	ctx := context.Background()

	req := SendTaskRequest{
		SessionID: "session-1",
		Message:   Message{Role: "user", Content: "Hello"},
	}

	task, err := tm.CreateTask(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, task)

	// Verify task was persisted to store
	storedTask, err := store.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.ID, storedTask.ID)
	assert.Equal(t, task.Status, storedTask.Status)
}

func TestTaskManager_GetTask(t *testing.T) {
	tm := NewTaskManager()
	ctx := context.Background()

	// Create a task first
	req := SendTaskRequest{
		ID:        "test-task",
		SessionID: "session-1",
		Message:   Message{Role: "user", Content: "Hello"},
	}

	created, err := tm.CreateTask(ctx, req)
	require.NoError(t, err)

	// Test getting existing task
	t.Run("get existing task", func(t *testing.T) {
		task, err := tm.GetTask(ctx, "test-task")
		require.NoError(t, err)
		assert.Equal(t, created.ID, task.ID)
		assert.Equal(t, created.Status, task.Status)
	})

	// Test getting non-existent task
	t.Run("get non-existent task", func(t *testing.T) {
		_, err := tm.GetTask(ctx, "non-existent")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrTaskNotFound))
	})

	// Test getting task with empty ID
	t.Run("get task with empty ID", func(t *testing.T) {
		_, err := tm.GetTask(ctx, "")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrTaskNotFound))
	})
}

func TestTaskManager_UpdateTask(t *testing.T) {
	tm := NewTaskManager()
	ctx := context.Background()

	// Create a task first
	req := SendTaskRequest{
		ID:        "update-test",
		SessionID: "session-1",
		Message:   Message{Role: "user", Content: "Hello"},
	}

	_, err := tm.CreateTask(ctx, req)
	require.NoError(t, err)

	tests := []struct {
		name    string
		task    *Task
		wantErr bool
		errType error
	}{
		{
			name: "update existing task",
			task: &Task{
				ID:      "update-test",
				Status:  TaskStateWorking,
				Message: Message{Role: "agent", Content: "Updated"},
			},
			wantErr: false,
		},
		{
			name:    "update non-existent task",
			task:    &Task{ID: "non-existent", Status: TaskStateWorking},
			wantErr: true,
			errType: ErrTaskNotFound,
		},
		{
			name:    "update nil task",
			task:    nil,
			wantErr: true,
		},
		{
			name:    "update task with empty ID",
			task:    &Task{ID: "", Status: TaskStateWorking},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tm.UpdateTask(ctx, tt.task)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.True(t, errors.Is(err, tt.errType))
				}
				return
			}

			require.NoError(t, err)

			// Verify update
			updated, err := tm.GetTask(ctx, tt.task.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.task.Status, updated.Status)
			assert.True(t, updated.UpdatedAt.After(updated.CreatedAt) || updated.UpdatedAt.Equal(updated.CreatedAt))
		})
	}
}

func TestTaskManager_TransitionState(t *testing.T) {
	tests := []struct {
		name      string
		initial   TaskState
		target    TaskStatus
		wantErr   bool
		errType   error
		checkTerm bool
	}{
		// Valid transitions from pending/submitted
		{
			name:    "pending to working",
			initial: TaskStateSubmitted,
			target:  TaskStatusWorking,
			wantErr: false,
		},
		{
			name:      "pending to cancelled",
			initial:   TaskStateSubmitted,
			target:    TaskStatusCancelled,
			wantErr:   false,
			checkTerm: true,
		},
		// Valid transitions from working
		{
			name:      "working to completed",
			initial:   TaskStateWorking,
			target:    TaskStatusCompleted,
			wantErr:   false,
			checkTerm: true,
		},
		{
			name:      "working to failed",
			initial:   TaskStateWorking,
			target:    TaskStatusFailed,
			wantErr:   false,
			checkTerm: true,
		},
		{
			name:    "working to input-required",
			initial: TaskStateWorking,
			target:  TaskStatusInputRequired,
			wantErr: false,
		},
		{
			name:      "working to cancelled",
			initial:   TaskStateWorking,
			target:    TaskStatusCancelled,
			wantErr:   false,
			checkTerm: true,
		},
		// Valid transitions from input-required
		{
			name:    "input-required to working",
			initial: TaskStateInputRequired,
			target:  TaskStatusWorking,
			wantErr: false,
		},
		{
			name:      "input-required to cancelled",
			initial:   TaskStateInputRequired,
			target:    TaskStatusCancelled,
			wantErr:   false,
			checkTerm: true,
		},
		// Invalid transitions from pending
		{
			name:    "pending to completed (invalid)",
			initial: TaskStateSubmitted,
			target:  TaskStatusCompleted,
			wantErr: true,
			errType: ErrInvalidTransition,
		},
		{
			name:    "pending to failed (invalid)",
			initial: TaskStateSubmitted,
			target:  TaskStatusFailed,
			wantErr: true,
			errType: ErrInvalidTransition,
		},
		{
			name:    "pending to input-required (invalid)",
			initial: TaskStateSubmitted,
			target:  TaskStatusInputRequired,
			wantErr: true,
			errType: ErrInvalidTransition,
		},
		// Invalid transitions from working
		{
			name:    "working to pending (invalid)",
			initial: TaskStateWorking,
			target:  TaskStatusPending,
			wantErr: true,
			errType: ErrInvalidTransition,
		},
		// Invalid transitions from terminal states
		{
			name:    "completed to working (invalid)",
			initial: TaskStateCompleted,
			target:  TaskStatusWorking,
			wantErr: true,
			errType: ErrInvalidTransition,
		},
		{
			name:    "failed to pending (invalid)",
			initial: TaskStateFailed,
			target:  TaskStatusPending,
			wantErr: true,
			errType: ErrInvalidTransition,
		},
		{
			name:    "cancelled to working (invalid)",
			initial: TaskStateCanceled,
			target:  TaskStatusWorking,
			wantErr: true,
			errType: ErrInvalidTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTaskManager()
			ctx := context.Background()

			// Create task with initial state
			taskID := uuid.New().String()
			tm.tasks[taskID] = &Task{
				ID:        taskID,
				Status:    tt.initial,
				SessionID: "test-session",
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}

			err := tm.TransitionState(ctx, taskID, tt.target)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.True(t, errors.Is(err, tt.errType), "expected error to wrap %v", tt.errType)
				}
				return
			}

			require.NoError(t, err)

			// Verify state transition
			task, err := tm.GetTask(ctx, taskID)
			require.NoError(t, err)
			assert.Equal(t, taskStatusToTaskState(tt.target), task.Status)
			assert.True(t, task.UpdatedAt.After(task.CreatedAt))

			// Check terminal state
			if tt.checkTerm {
				assert.NotNil(t, task.CompletedAt)
			}
		})
	}
}

func TestTaskManager_TransitionState_NonExistent(t *testing.T) {
	tm := NewTaskManager()
	ctx := context.Background()

	err := tm.TransitionState(ctx, "non-existent", TaskStatusWorking)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrTaskNotFound))
}

func TestTaskManager_CancelTask(t *testing.T) {
	tm := NewTaskManager()
	ctx := context.Background()

	// Create a task
	req := SendTaskRequest{
		SessionID: "session-1",
		Message:   Message{Role: "user", Content: "Hello"},
	}

	task, err := tm.CreateTask(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, TaskStateSubmitted, task.Status)

	// Cancel the task
	err = tm.CancelTask(ctx, task.ID)
	require.NoError(t, err)

	// Verify cancelled
	cancelled, err := tm.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskStateCanceled, cancelled.Status)
	assert.NotNil(t, cancelled.CompletedAt)
}

func TestTaskManager_CancelTask_NonExistent(t *testing.T) {
	tm := NewTaskManager()
	ctx := context.Background()

	err := tm.CancelTask(ctx, "non-existent")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrTaskNotFound))
}

func TestTaskManager_CancelTask_InvalidTransition(t *testing.T) {
	tm := NewTaskManager()
	ctx := context.Background()

	// Create and complete a task
	req := SendTaskRequest{
		SessionID: "session-1",
		Message:   Message{Role: "user", Content: "Hello"},
	}

	task, err := tm.CreateTask(ctx, req)
	require.NoError(t, err)

	// Transition to working
	err = tm.TransitionState(ctx, task.ID, TaskStatusWorking)
	require.NoError(t, err)

	// Complete the task
	err = tm.TransitionState(ctx, task.ID, TaskStatusCompleted)
	require.NoError(t, err)

	// Try to cancel completed task (should fail)
	err = tm.CancelTask(ctx, task.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidTransition))
}

func TestTaskManager_ListTasks(t *testing.T) {
	tm := NewTaskManager()
	ctx := context.Background()

	// Initially empty
	tasks := tm.ListTasks(ctx)
	assert.Empty(t, tasks)

	// Create multiple tasks
	for i := 0; i < 5; i++ {
		req := SendTaskRequest{
			SessionID: "session-1",
			Message:   Message{Role: "user", Content: "Hello"},
		}
		_, err := tm.CreateTask(ctx, req)
		require.NoError(t, err)
	}

	// List tasks
	tasks = tm.ListTasks(ctx)
	assert.Len(t, tasks, 5)

	// Verify all tasks are returned
	taskIDs := make(map[string]bool)
	for _, task := range tasks {
		taskIDs[task.ID] = true
	}
	assert.Len(t, taskIDs, 5)
}

func TestTaskManager_DeleteTask(t *testing.T) {
	tm := NewTaskManager()
	ctx := context.Background()

	// Create a task
	req := SendTaskRequest{
		ID:        "delete-me",
		SessionID: "session-1",
		Message:   Message{Role: "user", Content: "Hello"},
	}

	_, err := tm.CreateTask(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 1, tm.Count())

	// Delete the task
	err = tm.DeleteTask(ctx, "delete-me")
	require.NoError(t, err)
	assert.Equal(t, 0, tm.Count())

	// Verify deletion
	_, err = tm.GetTask(ctx, "delete-me")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrTaskNotFound))
}

func TestTaskManager_DeleteTask_NonExistent(t *testing.T) {
	tm := NewTaskManager()
	ctx := context.Background()

	err := tm.DeleteTask(ctx, "non-existent")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrTaskNotFound))
}

func TestTaskManager_Count(t *testing.T) {
	tm := NewTaskManager()
	ctx := context.Background()

	assert.Equal(t, 0, tm.Count())

	// Add tasks
	for i := 0; i < 3; i++ {
		req := SendTaskRequest{
			SessionID: "session-1",
			Message:   Message{Role: "user", Content: "Hello"},
		}
		_, err := tm.CreateTask(ctx, req)
		require.NoError(t, err)
	}

	assert.Equal(t, 3, tm.Count())
}

func TestTaskManager_Clear(t *testing.T) {
	tm := NewTaskManager()
	ctx := context.Background()

	// Add tasks
	for i := 0; i < 3; i++ {
		req := SendTaskRequest{
			SessionID: "session-1",
			Message:   Message{Role: "user", Content: "Hello"},
		}
		_, err := tm.CreateTask(ctx, req)
		require.NoError(t, err)
	}

	assert.Equal(t, 3, tm.Count())

	// Clear all tasks
	tm.Clear()
	assert.Equal(t, 0, tm.Count())
	assert.Empty(t, tm.ListTasks(ctx))
}

func TestTaskManager_ConcurrentAccess(t *testing.T) {
	tm := NewTaskManager()
	ctx := context.Background()

	// Concurrent creates
	t.Run("concurrent creates", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			go func() {
				req := SendTaskRequest{
					SessionID: "session-1",
					Message:   Message{Role: "user", Content: "Hello"},
				}
				_, _ = tm.CreateTask(ctx, req)
			}()
		}
	})

	// Wait for goroutines to complete
	time.Sleep(100 * time.Millisecond)

	// Verify no race conditions occurred
	count := tm.Count()
	assert.Greater(t, count, 0)

	// Concurrent reads
	t.Run("concurrent reads", func(t *testing.T) {
		tasks := tm.ListTasks(ctx)
		for _, task := range tasks {
			go func(id string) {
				_, _ = tm.GetTask(ctx, id)
			}(task.ID)
		}
	})

	time.Sleep(50 * time.Millisecond)
}

func TestTaskStatusToTaskState(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected TaskState
	}{
		{TaskStatusPending, TaskStateSubmitted},
		{TaskStatusWorking, TaskStateWorking},
		{TaskStatusInputRequired, TaskStateInputRequired},
		{TaskStatusCompleted, TaskStateCompleted},
		{TaskStatusFailed, TaskStateFailed},
		{TaskStatusCancelled, TaskStateCanceled},
		{TaskStatus("unknown"), TaskStateSubmitted}, // default case
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := taskStatusToTaskState(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTaskManager_FullLifecycle(t *testing.T) {
	tm := NewTaskManager()
	ctx := context.Background()

	// 1. Create task
	req := SendTaskRequest{
		SessionID: "session-1",
		Message:   Message{Role: "user", Content: "Process this"},
	}

	task, err := tm.CreateTask(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, TaskStateSubmitted, task.Status)

	// 2. Transition to working
	err = tm.TransitionState(ctx, task.ID, TaskStatusWorking)
	require.NoError(t, err)

	task, err = tm.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskStateWorking, task.Status)
	assert.Nil(t, task.CompletedAt)

	// 3. Transition to input-required
	err = tm.TransitionState(ctx, task.ID, TaskStatusInputRequired)
	require.NoError(t, err)

	task, err = tm.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskStateInputRequired, task.Status)
	assert.Nil(t, task.CompletedAt)

	// 4. Transition back to working
	err = tm.TransitionState(ctx, task.ID, TaskStatusWorking)
	require.NoError(t, err)

	task, err = tm.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskStateWorking, task.Status)

	// 5. Complete the task
	err = tm.TransitionState(ctx, task.ID, TaskStatusCompleted)
	require.NoError(t, err)

	task, err = tm.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskStateCompleted, task.Status)
	assert.NotNil(t, task.CompletedAt)

	// 6. Verify no more transitions allowed
	err = tm.TransitionState(ctx, task.ID, TaskStatusFailed)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidTransition))
}
