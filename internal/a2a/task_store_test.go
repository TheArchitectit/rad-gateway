package a2a

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupTaskTestDB creates an in-memory SQLite database with a2a_tasks table for testing.
func setupTaskTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Create the a2a_tasks table (SQLite-compatible version)
	schema := `
		CREATE TABLE a2a_tasks (
			id TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			session_id TEXT NOT NULL,
			message TEXT NOT NULL,
			artifacts TEXT DEFAULT '[]',
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME,
			parent_id TEXT,
			workspace_id TEXT,
			assigned_agent_id TEXT
		);

		CREATE INDEX idx_a2a_tasks_status ON a2a_tasks(status);
		CREATE INDEX idx_a2a_tasks_session_id ON a2a_tasks(session_id);
		CREATE INDEX idx_a2a_tasks_workspace_id ON a2a_tasks(workspace_id);
		CREATE INDEX idx_a2a_tasks_created_at ON a2a_tasks(created_at);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create test schema: %v", err)
	}

	return db
}

// createTestTask creates a test task with default values.
func createTestTask(id string) *Task {
	now := time.Now().UTC()
	workspaceID := "test-workspace"
	return &Task{
		ID:        id,
		Status:    TaskStateSubmitted,
		SessionID: "test-session",
		Message: Message{
			Role:    "user",
			Content: "Test message content",
		},
		Artifacts:   []Artifact{},
		Metadata:    json.RawMessage(`{"key": "value"}`),
		CreatedAt:   now,
		UpdatedAt:   now,
		WorkspaceID: &workspaceID,
	}
}

func TestPostgresTaskStore_CreateTask(t *testing.T) {
	db := setupTaskTestDB(t)
	defer db.Close()

	store := NewPostgresTaskStore(db)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		task := createTestTask("task-1")

		err := store.CreateTask(ctx, task)
		if err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}

		// Verify task was created
		retrieved, err := store.GetTask(ctx, task.ID)
		if err != nil {
			t.Fatalf("GetTask failed: %v", err)
		}

		if retrieved.ID != task.ID {
			t.Errorf("expected ID %s, got %s", task.ID, retrieved.ID)
		}

		if retrieved.Status != task.Status {
			t.Errorf("expected status %s, got %s", task.Status, retrieved.Status)
		}

		if retrieved.Message.Content != task.Message.Content {
			t.Errorf("expected message content %s, got %s", task.Message.Content, retrieved.Message.Content)
		}
	})

	t.Run("duplicate ID", func(t *testing.T) {
		task := createTestTask("task-duplicate")

		// First creation should succeed
		err := store.CreateTask(ctx, task)
		if err != nil {
			t.Fatalf("First CreateTask failed: %v", err)
		}

		// Second creation should fail
		err = store.CreateTask(ctx, task)
		if err == nil {
			t.Error("expected error for duplicate ID, got nil")
		}
	})
}

func TestPostgresTaskStore_GetTask(t *testing.T) {
	db := setupTaskTestDB(t)
	defer db.Close()

	store := NewPostgresTaskStore(db)
	ctx := context.Background()

	t.Run("existing task", func(t *testing.T) {
		task := createTestTask("task-get-existing")
		err := store.CreateTask(ctx, task)
		if err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}

		retrieved, err := store.GetTask(ctx, task.ID)
		if err != nil {
			t.Fatalf("GetTask failed: %v", err)
		}

		if retrieved.ID != task.ID {
			t.Errorf("expected ID %s, got %s", task.ID, retrieved.ID)
		}
	})

	t.Run("non-existent task", func(t *testing.T) {
		_, err := store.GetTask(ctx, "non-existent-id")
		if err != ErrTaskNotFound {
			t.Errorf("expected ErrTaskNotFound, got %v", err)
		}
	})
}

func TestPostgresTaskStore_UpdateTask(t *testing.T) {
	db := setupTaskTestDB(t)
	defer db.Close()

	store := NewPostgresTaskStore(db)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		task := createTestTask("task-update")
		err := store.CreateTask(ctx, task)
		if err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}

		// Update the task
		task.Status = TaskStateCompleted
		task.Message.Content = "Updated content"

		err = store.UpdateTask(ctx, task)
		if err != nil {
			t.Fatalf("UpdateTask failed: %v", err)
		}

		// Verify the update
		retrieved, err := store.GetTask(ctx, task.ID)
		if err != nil {
			t.Fatalf("GetTask failed: %v", err)
		}

		if retrieved.Status != TaskStateCompleted {
			t.Errorf("expected status %s, got %s", TaskStateCompleted, retrieved.Status)
		}

		if retrieved.Message.Content != "Updated content" {
			t.Errorf("expected message content 'Updated content', got %s", retrieved.Message.Content)
		}
	})

	t.Run("non-existent task", func(t *testing.T) {
		task := createTestTask("task-update-nonexistent")
		err := store.UpdateTask(ctx, task)
		if err != ErrTaskNotFound {
			t.Errorf("expected ErrTaskNotFound, got %v", err)
		}
	})
}

func TestPostgresTaskStore_DeleteTask(t *testing.T) {
	db := setupTaskTestDB(t)
	defer db.Close()

	store := NewPostgresTaskStore(db)
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		task := createTestTask("task-delete")
		err := store.CreateTask(ctx, task)
		if err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}

		// Delete the task
		err = store.DeleteTask(ctx, task.ID)
		if err != nil {
			t.Fatalf("DeleteTask failed: %v", err)
		}

		// Verify the task is deleted
		_, err = store.GetTask(ctx, task.ID)
		if err != ErrTaskNotFound {
			t.Errorf("expected ErrTaskNotFound after deletion, got %v", err)
		}
	})

	t.Run("non-existent task", func(t *testing.T) {
		err := store.DeleteTask(ctx, "non-existent-id")
		if err != ErrTaskNotFound {
			t.Errorf("expected ErrTaskNotFound, got %v", err)
		}
	})
}

func TestPostgresTaskStore_ListTasks(t *testing.T) {
	db := setupTaskTestDB(t)
	defer db.Close()

	store := NewPostgresTaskStore(db)
	ctx := context.Background()

	// Create test tasks
	workspace1 := "workspace-1"
	workspace2 := "workspace-2"

	tasks := []*Task{
		{ID: "task-1", Status: TaskStateSubmitted, SessionID: "session-1", Message: Message{Role: "user", Content: "Task 1"}, WorkspaceID: &workspace1},
		{ID: "task-2", Status: TaskStateWorking, SessionID: "session-1", Message: Message{Role: "user", Content: "Task 2"}, WorkspaceID: &workspace1},
		{ID: "task-3", Status: TaskStateCompleted, SessionID: "session-2", Message: Message{Role: "user", Content: "Task 3"}, WorkspaceID: &workspace2},
		{ID: "task-4", Status: TaskStateSubmitted, SessionID: "session-2", Message: Message{Role: "user", Content: "Task 4"}, WorkspaceID: &workspace2},
	}

	for _, task := range tasks {
		task.CreatedAt = time.Now().UTC()
		task.UpdatedAt = task.CreatedAt
		task.Artifacts = []Artifact{}
		task.Metadata = json.RawMessage(`{}`)
		if err := store.CreateTask(ctx, task); err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}
	}

	t.Run("list by workspace", func(t *testing.T) {
		filter := TaskFilter{
			WorkspaceID: workspace1,
			Limit:       10,
		}

		result, err := store.ListTasks(ctx, filter)
		if err != nil {
			t.Fatalf("ListTasks failed: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("expected 2 tasks for workspace-1, got %d", len(result))
		}
	})

	t.Run("list by session", func(t *testing.T) {
		filter := TaskFilter{
			SessionID: "session-1",
			Limit:     10,
		}

		result, err := store.ListTasks(ctx, filter)
		if err != nil {
			t.Fatalf("ListTasks failed: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("expected 2 tasks for session-1, got %d", len(result))
		}
	})

	t.Run("list by status", func(t *testing.T) {
		filter := TaskFilter{
			Status: TaskStateSubmitted,
			Limit:  10,
		}

		result, err := store.ListTasks(ctx, filter)
		if err != nil {
			t.Fatalf("ListTasks failed: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("expected 2 tasks with status 'submitted', got %d", len(result))
		}
	})

	t.Run("list with limit", func(t *testing.T) {
		filter := TaskFilter{
			Limit: 2,
		}

		result, err := store.ListTasks(ctx, filter)
		if err != nil {
			t.Fatalf("ListTasks failed: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("expected 2 tasks with limit, got %d", len(result))
		}
	})

	t.Run("list with offset", func(t *testing.T) {
		filter := TaskFilter{
			Limit:  2,
			Offset: 2,
		}

		result, err := store.ListTasks(ctx, filter)
		if err != nil {
			t.Fatalf("ListTasks failed: %v", err)
		}

		// Should get the remaining tasks
		if len(result) != 2 {
			t.Errorf("expected 2 tasks with offset, got %d", len(result))
		}
	})

	t.Run("combined filter", func(t *testing.T) {
		filter := TaskFilter{
			WorkspaceID: workspace1,
			SessionID:   "session-1",
			Status:      TaskStateSubmitted,
			Limit:       10,
		}

		result, err := store.ListTasks(ctx, filter)
		if err != nil {
			t.Fatalf("ListTasks failed: %v", err)
		}

		if len(result) != 1 {
			t.Errorf("expected 1 task with combined filter, got %d", len(result))
		}

		if len(result) > 0 && result[0].ID != "task-1" {
			t.Errorf("expected task-1, got %s", result[0].ID)
		}
	})

	t.Run("empty result", func(t *testing.T) {
		filter := TaskFilter{
			WorkspaceID: "non-existent-workspace",
			Limit:       10,
		}

		result, err := store.ListTasks(ctx, filter)
		if err != nil {
			t.Fatalf("ListTasks failed: %v", err)
		}

		if len(result) != 0 {
			t.Errorf("expected 0 tasks for non-existent workspace, got %d", len(result))
		}
	})
}

func TestPostgresTaskStore_TaskStateTransitions(t *testing.T) {
	db := setupTaskTestDB(t)
	defer db.Close()

	store := NewPostgresTaskStore(db)
	ctx := context.Background()

	task := createTestTask("task-transition")
	err := store.CreateTask(ctx, task)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// Test state transitions
	transitions := []struct {
		from   TaskState
		to     TaskState
		valid  bool
	}{
		{TaskStateSubmitted, TaskStateWorking, true},
		{TaskStateSubmitted, TaskStateCanceled, true},
		{TaskStateSubmitted, TaskStateCompleted, false},
		{TaskStateWorking, TaskStateCompleted, true},
		{TaskStateWorking, TaskStateFailed, true},
		{TaskStateWorking, TaskStateInputRequired, true},
		{TaskStateWorking, TaskStateCanceled, true},
		{TaskStateCompleted, TaskStateWorking, false},
	}

	for _, tc := range transitions {
		t.Run(fmt.Sprintf("%s_to_%s", tc.from, tc.to), func(t *testing.T) {
			task.Status = tc.from
			canTransition := task.CanTransitionTo(tc.to)
			if canTransition != tc.valid {
				t.Errorf("CanTransitionTo(%s) from %s = %v, expected %v", tc.to, tc.from, canTransition, tc.valid)
			}
		})
	}
}

func TestPostgresTaskStore_Artifacts(t *testing.T) {
	db := setupTaskTestDB(t)
	defer db.Close()

	store := NewPostgresTaskStore(db)
	ctx := context.Background()

	t.Run("task with artifacts", func(t *testing.T) {
		task := createTestTask("task-artifacts")
		task.Artifacts = []Artifact{
			{
				Type:        "text",
				Content:     json.RawMessage(`{"result": "success"}`),
				Name:        "output",
				Description: "Task output",
			},
		}

		err := store.CreateTask(ctx, task)
		if err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}

		retrieved, err := store.GetTask(ctx, task.ID)
		if err != nil {
			t.Fatalf("GetTask failed: %v", err)
		}

		if len(retrieved.Artifacts) != 1 {
			t.Fatalf("expected 1 artifact, got %d", len(retrieved.Artifacts))
		}

		if retrieved.Artifacts[0].Type != "text" {
			t.Errorf("expected artifact type 'text', got %s", retrieved.Artifacts[0].Type)
		}

		if retrieved.Artifacts[0].Name != "output" {
			t.Errorf("expected artifact name 'output', got %s", retrieved.Artifacts[0].Name)
		}
	})
}
