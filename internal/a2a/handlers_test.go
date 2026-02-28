package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockRepository is a mock implementation of Repository for testing
type mockRepository struct {
	cards      map[string]*ModelCard
	slugIndex  map[string]*ModelCard
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		cards:     make(map[string]*ModelCard),
		slugIndex: make(map[string]*ModelCard),
	}
}

func (m *mockRepository) Create(ctx context.Context, card *ModelCard) error {
	m.cards[card.ID] = card
	key := card.WorkspaceID + "/" + card.Slug
	m.slugIndex[key] = card
	return nil
}

func (m *mockRepository) GetByID(ctx context.Context, id string) (*ModelCard, error) {
	card, ok := m.cards[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return card, nil
}

func (m *mockRepository) GetBySlug(ctx context.Context, workspaceID, slug string) (*ModelCard, error) {
	key := workspaceID + "/" + slug
	card, ok := m.slugIndex[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return card, nil
}

func (m *mockRepository) GetByProject(ctx context.Context, projectID string) ([]ModelCard, error) {
	var result []ModelCard
	for _, card := range m.cards {
		if card.WorkspaceID == projectID {
			result = append(result, *card)
		}
	}
	return result, nil
}

func (m *mockRepository) Update(ctx context.Context, card *ModelCard) error {
	m.cards[card.ID] = card
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	card, ok := m.cards[id]
	if !ok {
		return errors.New("not found")
	}
	delete(m.cards, id)
	key := card.WorkspaceID + "/" + card.Slug
	delete(m.slugIndex, key)
	return nil
}

func setupTestHandlers() (*Handlers, *TaskManager) {
	repo := newMockRepository()
	taskManager := NewTaskManager()
	handlers := NewHandlersWithTaskManager(repo, taskManager)
	return handlers, taskManager
}

func TestHandlers_handleSendTask(t *testing.T) {
	handlers, _ := setupTestHandlers()

	tests := []struct {
		name           string
		method         string
		body           interface{}
		wantStatusCode int
		wantError      bool
	}{
		{
			name:   "create task successfully",
			method: http.MethodPost,
			body: SendTaskRequest{
				SessionID: "session-123",
				Message: Message{
					Role:    "user",
					Content: "Hello, agent!",
				},
			},
			wantStatusCode: http.StatusCreated,
			wantError:      false,
		},
		{
			name:   "create task with custom ID",
			method: http.MethodPost,
			body: SendTaskRequest{
				ID:        "custom-task-id",
				SessionID: "session-123",
				Message: Message{
					Role:    "user",
					Content: "Hello!",
				},
			},
			wantStatusCode: http.StatusCreated,
			wantError:      false,
		},
		{
			name:   "missing sessionId fails",
			method: http.MethodPost,
			body: SendTaskRequest{
				Message: Message{
					Role:    "user",
					Content: "Hello!",
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
		},
		{
			name:           "invalid JSON fails",
			method:         http.MethodPost,
			body:           "invalid json",
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
		},
		{
			name:           "GET method not allowed",
			method:         http.MethodGet,
			body:           nil,
			wantStatusCode: http.StatusMethodNotAllowed,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					body = []byte(str)
				} else {
					body, _ = json.Marshal(tt.body)
				}
			}

			req := httptest.NewRequest(tt.method, "/a2a/tasks/send", bytes.NewReader(body))
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			rr := httptest.NewRecorder()

			handlers.handleSendTask(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("handleSendTask() status = %v, want %v", rr.Code, tt.wantStatusCode)
			}

			if !tt.wantError {
				var resp SendTaskResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}
				if resp.Task == nil {
					t.Error("Expected task in response, got nil")
				}
				if resp.Task != nil && resp.Task.SessionID != tt.body.(SendTaskRequest).SessionID {
					t.Errorf("Task sessionID = %v, want %v", resp.Task.SessionID, tt.body.(SendTaskRequest).SessionID)
				}
			} else {
				var errResp map[string]string
				if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
					t.Errorf("Failed to unmarshal error response: %v", err)
				}
				if _, ok := errResp["error"]; !ok {
					t.Error("Expected error field in response")
				}
			}
		})
	}
}

func TestHandlers_handleSendTask_DuplicateID(t *testing.T) {
	handlers, _ := setupTestHandlers()

	// Create first task
	reqBody := SendTaskRequest{
		ID:        "duplicate-id",
		SessionID: "session-123",
		Message: Message{
			Role:    "user",
			Content: "First message",
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/a2a/tasks/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handlers.handleSendTask(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("First request failed: %d", rr.Code)
	}

	// Try to create second task with same ID
	req2 := httptest.NewRequest(http.MethodPost, "/a2a/tasks/send", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	handlers.handleSendTask(rr2, req2)

	if rr2.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 for duplicate ID, got %d", rr2.Code)
	}
}

func TestHandlers_handleTaskByID(t *testing.T) {
	handlers, tm := setupTestHandlers()

	// Create a task first
	task, err := tm.CreateTask(t.Context(), SendTaskRequest{
		SessionID: "session-123",
		Message: Message{
			Role:    "user",
			Content: "Test message",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	tests := []struct {
		name           string
		method         string
		path           string
		wantStatusCode int
		wantTask       bool
	}{
		{
			name:           "get existing task",
			method:         http.MethodGet,
			path:           "/a2a/tasks/" + task.ID,
			wantStatusCode: http.StatusOK,
			wantTask:       true,
		},
		{
			name:           "get non-existent task",
			method:         http.MethodGet,
			path:           "/a2a/tasks/non-existent-id",
			wantStatusCode: http.StatusNotFound,
			wantTask:       false,
		},
		{
			name:           "missing task ID",
			method:         http.MethodGet,
			path:           "/a2a/tasks/",
			wantStatusCode: http.StatusBadRequest,
			wantTask:       false,
		},
		{
			name:           "POST method not allowed",
			method:         http.MethodPost,
			path:           "/a2a/tasks/" + task.ID,
			wantStatusCode: http.StatusMethodNotAllowed,
			wantTask:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			handlers.handleTaskByID(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("handleTaskByID() status = %v, want %v", rr.Code, tt.wantStatusCode)
			}

			if tt.wantTask {
				var resp Task
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}
				if resp.ID != task.ID {
					t.Errorf("Task ID = %v, want %v", resp.ID, task.ID)
				}
			}
		})
	}
}

func TestHandlers_handleCancelTask(t *testing.T) {
	handlers, tm := setupTestHandlers()

	// Create a task first
	task, err := tm.CreateTask(t.Context(), SendTaskRequest{
		SessionID: "session-123",
		Message: Message{
			Role:    "user",
			Content: "Test message",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	tests := []struct {
		name           string
		method         string
		body           interface{}
		wantStatusCode int
		wantCancelled  bool
	}{
		{
			name:   "cancel existing task",
			method: http.MethodPost,
			body: map[string]string{
				"taskId": task.ID,
			},
			wantStatusCode: http.StatusOK,
			wantCancelled:  true,
		},
		{
			name:   "cancel non-existent task",
			method: http.MethodPost,
			body: map[string]string{
				"taskId": "non-existent-id",
			},
			wantStatusCode: http.StatusNotFound,
			wantCancelled:  false,
		},
		{
			name:   "missing taskId",
			method: http.MethodPost,
			body:   map[string]string{},
			wantStatusCode: http.StatusBadRequest,
			wantCancelled:  false,
		},
		{
			name:           "invalid JSON",
			method:         http.MethodPost,
			body:           "invalid json",
			wantStatusCode: http.StatusBadRequest,
			wantCancelled:  false,
		},
		{
			name:           "GET method not allowed",
			method:         http.MethodGet,
			body:           nil,
			wantStatusCode: http.StatusMethodNotAllowed,
			wantCancelled:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					body = []byte(str)
				} else {
					body, _ = json.Marshal(tt.body)
				}
			}

			req := httptest.NewRequest(tt.method, "/a2a/tasks/cancel", bytes.NewReader(body))
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			rr := httptest.NewRecorder()

			handlers.handleCancelTask(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("handleCancelTask() status = %v, want %v", rr.Code, tt.wantStatusCode)
			}

			if tt.wantCancelled {
				var resp Task
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}
				if resp.Status != TaskStateCanceled {
					t.Errorf("Task status = %v, want %v", resp.Status, TaskStateCanceled)
				}
			}
		})
	}
}

func TestHandlers_handleCancelTask_InvalidTransition(t *testing.T) {
	handlers, tm := setupTestHandlers()

	// Create a task
	task, err := tm.CreateTask(t.Context(), SendTaskRequest{
		SessionID: "session-123",
		Message: Message{
			Role:    "user",
			Content: "Test message",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Transition to working state, then completed
	err = tm.TransitionState(t.Context(), task.ID, TaskStatusWorking)
	if err != nil {
		t.Fatalf("Failed to transition to working: %v", err)
	}

	err = tm.TransitionState(t.Context(), task.ID, TaskStatusCompleted)
	if err != nil {
		t.Fatalf("Failed to complete task: %v", err)
	}

	// Try to cancel completed task
	body, _ := json.Marshal(map[string]string{"taskId": task.ID})
	req := httptest.NewRequest(http.MethodPost, "/a2a/tasks/cancel", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handlers.handleCancelTask(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("Expected 409 for invalid transition, got %d", rr.Code)
	}
}

func TestHandlers_TaskManagerNotConfigured(t *testing.T) {
	// Create handlers without task manager
	repo := newMockRepository()
	handlers := NewHandlers(repo)

	tests := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{
			name:   "send task without manager",
			method: http.MethodPost,
			path:   "/a2a/tasks/send",
			body:   []byte(`{"sessionId": "test", "message": {"role": "user", "content": "hi"}}`),
		},
		{
			name:   "get task without manager",
			method: http.MethodGet,
			path:   "/a2a/tasks/test-id",
			body:   nil,
		},
		{
			name:   "cancel task without manager",
			method: http.MethodPost,
			path:   "/a2a/tasks/cancel",
			body:   []byte(`{"taskId": "test-id"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(tt.body))
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			rr := httptest.NewRecorder()

			switch tt.path {
			case "/a2a/tasks/send":
				handlers.handleSendTask(rr, req)
			case "/a2a/tasks/cancel":
				handlers.handleCancelTask(rr, req)
			default:
				if strings.Contains(tt.path, "/a2a/tasks/") {
					handlers.handleTaskByID(rr, req)
				}
			}

			if rr.Code != http.StatusServiceUnavailable {
				t.Errorf("Expected 503 when task manager not configured, got %d", rr.Code)
			}
		})
	}
}

func TestHandlers_Register(t *testing.T) {
	handlers, tm := setupTestHandlers()

	// Create a test task for GET and cancel tests
	testTask, err := tm.CreateTask(t.Context(), SendTaskRequest{
		SessionID: "test-session",
		Message:   Message{Role: "user", Content: "test"},
	})
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	mux := http.NewServeMux()
	handlers.Register(mux)

	// Test that routes are registered
	routes := []struct {
		method     string
		path       string
		body       []byte
		wantStatus int // Expected status - should not be 404 if route is registered
	}{
		{http.MethodGet, "/a2a/model-cards", nil, http.StatusBadRequest}, // requires workspace_id
		{http.MethodPost, "/a2a/tasks/send", func() []byte {
			b, _ := json.Marshal(SendTaskRequest{
				SessionID: "test",
				Message:   Message{Role: "user", Content: "test"},
			})
			return b
		}(), http.StatusCreated},
		{http.MethodGet, "/a2a/tasks/" + testTask.ID, nil, http.StatusOK},
		{http.MethodPost, "/a2a/tasks/cancel", func() []byte {
			b, _ := json.Marshal(map[string]string{"taskId": testTask.ID})
			return b
		}(), http.StatusOK},
	}

	for _, route := range routes {
		t.Run(route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, bytes.NewReader(route.body))
			if route.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			// Should not be 404 (Not Found) if route is registered
			if rr.Code == http.StatusNotFound {
				t.Errorf("Route %s %s returned 404, expected registered (got body: %s)",
					route.method, route.path, rr.Body.String())
			}

			// Verify we got the expected status
			if rr.Code != route.wantStatus {
				t.Logf("Route %s %s returned %d, expected %d", route.method, route.path, rr.Code, route.wantStatus)
			}
		})
	}
}
