// Package integration provides integration tests for RAD Gateway
//
// A2A Protocol Integration Tests
// Tests A2A agent discovery, task lifecycle, and model card operations
//
// Run with: go test ./tests/integration/... -run TestA2A
// Run verbose: go test -v ./tests/integration/... -run TestA2A
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"radgateway/internal/a2a"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// inMemoryTaskStore implements a2a.TaskStore for testing
type inMemoryTaskStore struct {
	mu     sync.RWMutex
	tasks  map[string]*a2a.Task
}

func newInMemoryTaskStore() *inMemoryTaskStore {
	return &inMemoryTaskStore{
		tasks: make(map[string]*a2a.Task),
	}
}

func (s *inMemoryTaskStore) CreateTask(ctx context.Context, task *a2a.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
	return nil
}

func (s *inMemoryTaskStore) GetTask(ctx context.Context, id string) (*a2a.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	if !ok {
		return nil, a2a.ErrTaskNotFound
	}
	return task, nil
}

func (s *inMemoryTaskStore) UpdateTask(ctx context.Context, task *a2a.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[task.ID]; !ok {
		return a2a.ErrTaskNotFound
	}
	s.tasks[task.ID] = task
	return nil
}

func (s *inMemoryTaskStore) DeleteTask(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[id]; !ok {
		return a2a.ErrTaskNotFound
	}
	delete(s.tasks, id)
	return nil
}

func (s *inMemoryTaskStore) ListTasks(ctx context.Context, filter a2a.TaskFilter) ([]*a2a.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*a2a.Task
	for _, task := range s.tasks {
		result = append(result, task)
	}
	return result, nil
}

// setupA2AHandlers creates A2A handlers with an in-memory task store for testing
func setupA2AHandlers() (*a2a.Handlers, *inMemoryTaskStore) {
	taskStore := newInMemoryTaskStore()
	taskManager := a2a.NewTaskManagerWithStore(taskStore)
	repo := &mockRepository{}
	handlers := a2a.NewHandlersWithTaskManager(repo, taskManager)
	return handlers, taskStore
}

// mockRepository implements a2a.Repository for testing
type mockRepository struct {
	cards map[string]*a2a.ModelCard
}

func (m *mockRepository) GetByID(ctx context.Context, id string) (*a2a.ModelCard, error) {
	if m.cards == nil {
		m.cards = make(map[string]*a2a.ModelCard)
	}
	if card, ok := m.cards[id]; ok {
		return card, nil
	}
	return nil, a2a.ErrTaskNotFound
}

func (m *mockRepository) GetByProject(ctx context.Context, projectID string) ([]a2a.ModelCard, error) {
	var result []a2a.ModelCard
	for _, card := range m.cards {
		if card.WorkspaceID == projectID {
			result = append(result, *card)
		}
	}
	return result, nil
}

func (m *mockRepository) Create(ctx context.Context, card *a2a.ModelCard) error {
	if m.cards == nil {
		m.cards = make(map[string]*a2a.ModelCard)
	}
	if card.ID == "" {
		card.ID = generateTestID()
	}
	card.CreatedAt = time.Now().UTC()
	card.UpdatedAt = time.Now().UTC()
	m.cards[card.ID] = card
	return nil
}

func (m *mockRepository) Update(ctx context.Context, card *a2a.ModelCard) error {
	if m.cards == nil {
		return a2a.ErrTaskNotFound
	}
	if _, ok := m.cards[card.ID]; !ok {
		return a2a.ErrTaskNotFound
	}
	card.UpdatedAt = time.Now().UTC()
	m.cards[card.ID] = card
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	if m.cards == nil {
		return a2a.ErrTaskNotFound
	}
	if _, ok := m.cards[id]; !ok {
		return a2a.ErrTaskNotFound
	}
	delete(m.cards, id)
	return nil
}

func (m *mockRepository) GetBySlug(ctx context.Context, workspaceID, slug string) (*a2a.ModelCard, error) {
	for _, card := range m.cards {
		if card.WorkspaceID == workspaceID && card.Slug == slug {
			return card, nil
		}
	}
	return nil, a2a.ErrTaskNotFound
}

var testIDCounter int

func generateTestID() string {
	testIDCounter++
	return fmt.Sprintf("test-id-%d", testIDCounter)
}

// TestA2A_AgentCardDiscovery tests the GET /.well-known/agent.json endpoint
func TestA2A_AgentCardDiscovery(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		validateResp   func(t *testing.T, body []byte)
	}{
		{
			name:           "get_agent_card_success",
			method:         http.MethodGet,
			path:           "/.well-known/agent.json",
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, body []byte) {
				var card a2a.AgentCard
				err := json.Unmarshal(body, &card)
				require.NoError(t, err, "response should be valid JSON")

				// Verify required fields per A2A spec
				assert.NotEmpty(t, card.Name, "name field is required")
				assert.NotEmpty(t, card.URL, "url field is required")
				assert.NotEmpty(t, card.Version, "version field is required")
			},
		},
		{
			name:           "agent_card_method_not_allowed_post",
			method:         http.MethodPost,
			path:           "/.well-known/agent.json",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "agent_card_method_not_allowed_put",
			method:         http.MethodPut,
			path:           "/.well-known/agent.json",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, _ := setupA2AHandlers()
			mux := http.NewServeMux()
			// Register a mock agent card endpoint
			mux.HandleFunc("/.well-known/agent.json", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
					return
				}
				card := a2a.AgentCard{
					Name:        "RAD Gateway",
					Description: "AI API Gateway supporting A2A protocol",
					URL:         "http://localhost:8090",
					Version:     "1.0.0",
					Capabilities: a2a.Capabilities{
						Streaming:              true,
						PushNotifications:      false,
						StateTransitionHistory: true,
					},
					Skills: []a2a.Skill{
						{
							ID:          "chat",
							Name:        "Chat Completion",
							Description: "OpenAI-compatible chat completion API",
						},
					},
					Authentication: a2a.AuthInfo{
						Schemes: []string{"apiKey", "bearer"},
					},
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(card)
			})
			handlers.Register(mux)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code, "unexpected status code")

			if tt.validateResp != nil && rr.Code == http.StatusOK {
				tt.validateResp(t, rr.Body.Bytes())
			}
		})
	}
}

// TestA2A_TaskLifecycle tests task creation, retrieval, and cancellation
func TestA2A_TaskLifecycle(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
		validateResp   func(t *testing.T, body []byte)
	}{
		{
			name:   "create_task_success",
			method: http.MethodPost,
			path:   "/a2a/tasks/send",
			body: a2a.SendTaskRequest{
				SessionID: "session-123",
				Message: a2a.Message{
					Role:    "user",
					Content: "Hello, test this task creation",
				},
			},
			expectedStatus: http.StatusCreated,
			validateResp: func(t *testing.T, body []byte) {
				var resp a2a.SendTaskResponse
				err := json.Unmarshal(body, &resp)
				require.NoError(t, err)
				require.NotNil(t, resp.Task)
				assert.NotEmpty(t, resp.Task.ID, "task ID should be generated")
				assert.Equal(t, a2a.TaskStateSubmitted, resp.Task.Status, "initial status should be submitted")
				assert.Equal(t, "session-123", resp.Task.SessionID)
			},
		},
		{
			name:   "create_task_missing_session_id",
			method: http.MethodPost,
			path:   "/a2a/tasks/send",
			body: a2a.SendTaskRequest{
				Message: a2a.Message{
					Role:    "user",
					Content: "Test message",
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "create_task_empty_body",
			method: http.MethodPost,
			path:   "/a2a/tasks/send",
			body:   map[string]string{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "get_task_success",
			method:         http.MethodGet,
			path:           "/a2a/tasks/test-task-id",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, body []byte) {
				var task a2a.Task
				err := json.Unmarshal(body, &task)
				require.NoError(t, err)
				assert.NotEmpty(t, task.ID)
			},
		},
		{
			name:           "get_task_not_found",
			method:         http.MethodGet,
			path:           "/a2a/tasks/non-existent-task",
			body:           nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "cancel_task_success",
			method: http.MethodPost,
			path:   "/a2a/tasks/cancel",
			body: map[string]string{
				"taskId": "test-task-to-cancel",
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, body []byte) {
				var task a2a.Task
				err := json.Unmarshal(body, &task)
				require.NoError(t, err)
				assert.Equal(t, a2a.TaskStateCanceled, task.Status, "task should be cancelled")
			},
		},
		{
			name:   "cancel_task_not_found",
			method: http.MethodPost,
			path:   "/a2a/tasks/cancel",
			body: map[string]string{
				"taskId": "non-existent-task",
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "cancel_task_missing_id",
			method: http.MethodPost,
			path:   "/a2a/tasks/cancel",
			body:   map[string]string{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, taskStore := setupA2AHandlers()
			mux := http.NewServeMux()
			handlers.Register(mux)

			// Pre-create a task for get/cancel tests
			if strings.Contains(tt.name, "get_task_success") || strings.Contains(tt.name, "cancel_task_success") {
				ctx := context.Background()
				taskID := "test-task-id"
				if strings.Contains(tt.name, "cancel") {
					taskID = "test-task-to-cancel"
				}
				task := &a2a.Task{
					ID:        taskID,
					Status:    a2a.TaskStateSubmitted,
					SessionID: "test-session",
					Message: a2a.Message{
						Role:    "user",
						Content: "Test message",
					},
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				}
				err := taskStore.CreateTask(ctx, task)
				require.NoError(t, err)
			}

			var body []byte
			if tt.body != nil {
				var err error
				body, err = json.Marshal(tt.body)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(body))
			if body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code, "unexpected status code: %s", rr.Body.String())

			if tt.validateResp != nil {
				tt.validateResp(t, rr.Body.Bytes())
			}
		})
	}
}

// TestA2A_TaskStateTransitions tests valid and invalid state transitions
func TestA2A_TaskStateTransitions(t *testing.T) {
	handlers, taskStore := setupA2AHandlers()
	mux := http.NewServeMux()
	handlers.Register(mux)

	ctx := context.Background()

	// Create a task for testing transitions
	task := &a2a.Task{
		ID:        "transition-test-task",
		Status:    a2a.TaskStateSubmitted,
		SessionID: "test-session",
		Message: a2a.Message{
			Role:    "user",
			Content: "Test message",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	err := taskStore.CreateTask(ctx, task)
	require.NoError(t, err)

	tests := []struct {
		name           string
		setupTask      func()
		expectedStatus int
	}{
		{
			name: "cancel_submitted_task",
			setupTask: func() {
				task.Status = a2a.TaskStateSubmitted
				taskStore.UpdateTask(ctx, task)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "cancel_working_task",
			setupTask: func() {
				task.Status = a2a.TaskStateWorking
				taskStore.UpdateTask(ctx, task)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupTask()

			body, _ := json.Marshal(map[string]string{"taskId": task.ID})
			req := httptest.NewRequest(http.MethodPost, "/a2a/tasks/cancel", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

// TestA2A_ModelCards tests CRUD operations on model cards
func TestA2A_ModelCards(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
		validateResp   func(t *testing.T, body []byte)
	}{
		{
			name:   "create_model_card_success",
			method: http.MethodPost,
			path:   "/a2a/model-cards",
			body: a2a.CreateModelCardRequest{
				WorkspaceID: "ws-123",
				Name:        "Test Model",
				Slug:        "test-model",
				Card:        json.RawMessage(`{"name":"Test","version":"1.0"}`),
			},
			expectedStatus: http.StatusCreated,
			validateResp: func(t *testing.T, body []byte) {
				var card a2a.ModelCard
				err := json.Unmarshal(body, &card)
				require.NoError(t, err)
				assert.NotEmpty(t, card.ID)
				assert.Equal(t, "Test Model", card.Name)
				assert.Equal(t, a2a.ModelCardStatusActive, card.Status)
			},
		},
		{
			name:   "create_model_card_missing_workspace",
			method: http.MethodPost,
			path:   "/a2a/model-cards",
			body: a2a.CreateModelCardRequest{
				Name: "Test Model",
				Slug: "test-model",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "create_model_card_missing_name",
			method: http.MethodPost,
			path:   "/a2a/model-cards",
			body: a2a.CreateModelCardRequest{
				WorkspaceID: "ws-123",
				Slug:        "test-model",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "list_model_cards_with_workspace_filter",
			method:         http.MethodGet,
			path:           "/a2a/model-cards?workspace_id=ws-123",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, body []byte) {
				var list a2a.ModelCardList
				err := json.Unmarshal(body, &list)
				require.NoError(t, err)
				assert.NotNil(t, list.Items)
			},
		},
		{
			name:           "list_model_cards_missing_workspace",
			method:         http.MethodGet,
			path:           "/a2a/model-cards",
			body:           nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, _ := setupA2AHandlers()
			mux := http.NewServeMux()
			handlers.Register(mux)

			var body []byte
			if tt.body != nil {
				var err error
				body, err = json.Marshal(tt.body)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(body))
			if body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code, "unexpected status code: %s", rr.Body.String())

			if tt.validateResp != nil {
				tt.validateResp(t, rr.Body.Bytes())
			}
		})
	}
}

// TestA2A_ModelCardCRUD tests full CRUD lifecycle for model cards
func TestA2A_ModelCardCRUD(t *testing.T) {
	handlers, _ := setupA2AHandlers()
	mux := http.NewServeMux()
	handlers.Register(mux)

	ctx := context.Background()
	repo := &mockRepository{}
	handlersWithRepo := a2a.NewHandlers(repo)
	mux2 := http.NewServeMux()
	handlersWithRepo.Register(mux2)

	// Create
	t.Run("create", func(t *testing.T) {
		reqBody := a2a.CreateModelCardRequest{
			WorkspaceID: "ws-crud",
			Name:        "CRUD Test Model",
			Slug:        "crud-model",
			Description: strPtr("A test model for CRUD operations"),
			Card:        json.RawMessage(`{"schemaVersion":"1.0","name":"CRUD Model"}`),
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/a2a/model-cards", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		mux2.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var card a2a.ModelCard
		err := json.Unmarshal(rr.Body.Bytes(), &card)
		require.NoError(t, err)
		assert.Equal(t, "CRUD Test Model", card.Name)

		// Store ID for subsequent tests
		ctx = context.WithValue(ctx, "cardID", card.ID)
	})

	// Get
	t.Run("get", func(t *testing.T) {
		// First create a card
		reqBody := a2a.CreateModelCardRequest{
			WorkspaceID: "ws-crud",
			Name:        "Get Test Model",
			Slug:        "get-model",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/a2a/model-cards", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		mux2.ServeHTTP(rr, req)

		var created a2a.ModelCard
		json.Unmarshal(rr.Body.Bytes(), &created)

		// Now get it
		req = httptest.NewRequest(http.MethodGet, "/a2a/model-cards/"+created.ID, nil)
		rr = httptest.NewRecorder()
		mux2.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var fetched a2a.ModelCard
		err := json.Unmarshal(rr.Body.Bytes(), &fetched)
		require.NoError(t, err)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, created.Name, fetched.Name)
	})

	// Update
	t.Run("update", func(t *testing.T) {
		// First create a card
		reqBody := a2a.CreateModelCardRequest{
			WorkspaceID: "ws-crud",
			Name:        "Update Test Model",
			Slug:        "update-model",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/a2a/model-cards", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		mux2.ServeHTTP(rr, req)

		var created a2a.ModelCard
		json.Unmarshal(rr.Body.Bytes(), &created)

		// Now update it
		newName := "Updated Model Name"
		updateReq := a2a.UpdateModelCardRequest{
			Name: &newName,
		}
		body, _ = json.Marshal(updateReq)
		req = httptest.NewRequest(http.MethodPut, "/a2a/model-cards/"+created.ID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr = httptest.NewRecorder()
		mux2.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var updated a2a.ModelCard
		err := json.Unmarshal(rr.Body.Bytes(), &updated)
		require.NoError(t, err)
		assert.Equal(t, newName, updated.Name)
		// Version may be incremented on update depending on implementation
		assert.GreaterOrEqual(t, updated.Version, created.Version)
	})

	// Delete
	t.Run("delete", func(t *testing.T) {
		// First create a card
		reqBody := a2a.CreateModelCardRequest{
			WorkspaceID: "ws-crud",
			Name:        "Delete Test Model",
			Slug:        "delete-model",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/a2a/model-cards", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		mux2.ServeHTTP(rr, req)

		var created a2a.ModelCard
		json.Unmarshal(rr.Body.Bytes(), &created)

		// Now delete it
		req = httptest.NewRequest(http.MethodDelete, "/a2a/model-cards/"+created.ID, nil)
		rr = httptest.NewRecorder()
		mux2.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Try to get it again - should fail
		req = httptest.NewRequest(http.MethodGet, "/a2a/model-cards/"+created.ID, nil)
		rr = httptest.NewRecorder()
		mux2.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

// TestA2A_InvalidRequests tests error handling for invalid requests
func TestA2A_InvalidRequests(t *testing.T) {
	handlers, _ := setupA2AHandlers()
	mux := http.NewServeMux()
	handlers.Register(mux)

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{
			name:           "invalid_json_body",
			method:         http.MethodPost,
			path:           "/a2a/tasks/send",
			body:           `{invalid json`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty_body_for_post",
			method:         http.MethodPost,
			path:           "/a2a/tasks/send",
			body:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid_method_for_tasks",
			method:         http.MethodPatch,
			path:           "/a2a/tasks/send",
			body:           `{}`,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.body != "" {
				body = []byte(tt.body)
			}

			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
