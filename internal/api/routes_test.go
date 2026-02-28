// Package api provides HTTP API handlers and route registration for RAD Gateway.
package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"radgateway/internal/a2a"
)

// mockA2ARepo implements a2a.Repository for testing
type mockA2ARepo struct{}

func (m *mockA2ARepo) GetByID(ctx context.Context, id string) (*a2a.ModelCard, error) {
	return &a2a.ModelCard{ID: id, Name: "Test Card"}, nil
}

func (m *mockA2ARepo) GetByProject(ctx context.Context, projectID string) ([]a2a.ModelCard, error) {
	return []a2a.ModelCard{}, nil
}

func (m *mockA2ARepo) Create(ctx context.Context, card *a2a.ModelCard) error {
	card.ID = "test-id-123"
	return nil
}

func (m *mockA2ARepo) Update(ctx context.Context, card *a2a.ModelCard) error {
	return nil
}

func (m *mockA2ARepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockA2ARepo) GetBySlug(ctx context.Context, workspaceID, slug string) (*a2a.ModelCard, error) {
	return &a2a.ModelCard{ID: "test-id", Name: "Test Card", Slug: slug}, nil
}

// mockTaskStore implements a2a.TaskStore for testing
type mockTaskStore struct{}

func (m *mockTaskStore) CreateTask(ctx context.Context, task *a2a.Task) error {
	task.ID = "task-123"
	return nil
}

func (m *mockTaskStore) GetTask(ctx context.Context, id string) (*a2a.Task, error) {
	return &a2a.Task{ID: id, SessionID: "session-123", Status: a2a.TaskStateSubmitted}, nil
}

func (m *mockTaskStore) UpdateTask(ctx context.Context, task *a2a.Task) error {
	return nil
}

func (m *mockTaskStore) DeleteTask(ctx context.Context, id string) error {
	return nil
}

func (m *mockTaskStore) ListTasks(ctx context.Context, filter a2a.TaskFilter) ([]*a2a.Task, error) {
	return []*a2a.Task{}, nil
}

// mockGateway implements the Gateway interface for testing
type mockGateway struct {
	repo      a2a.Repository
	taskStore a2a.TaskStore
}

func (m *mockGateway) GetA2ARepo() a2a.Repository {
	return m.repo
}

func (m *mockGateway) GetA2ATaskStore() a2a.TaskStore {
	return m.taskStore
}

func TestRegisterAllRoutes(t *testing.T) {
	// Create mock dependencies
	repo := &mockA2ARepo{}
	taskStore := &mockTaskStore{}
	gateway := &mockGateway{
		repo:      repo,
		taskStore: taskStore,
	}

	// Create mux and register routes
	mux := http.NewServeMux()
	RegisterAllRoutes(mux, gateway)

	// Test that routes are registered by making requests
	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		wantStatusCode int
		wantBody       string
	}{
		{
			name:           "A2A model cards list endpoint",
			method:         http.MethodGet,
			path:           "/v1/a2a/model-cards?workspace_id=test-workspace",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "A2A model card get endpoint",
			method:         http.MethodGet,
			path:           "/v1/a2a/model-cards/card-123",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "A2A tasks get endpoint",
			method:         http.MethodGet,
			path:           "/v1/a2a/tasks/task-123",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "MCP tools list endpoint",
			method:         http.MethodGet,
			path:           "/mcp/v1/tools/list",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "MCP tools invoke endpoint",
			method:         http.MethodPost,
			path:           "/mcp/v1/tools/invoke",
			body:           `{"tool":"echo","input":{"content":"hello"}}`,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "MCP health endpoint",
			method:         http.MethodGet,
			path:           "/mcp/v1/health",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "A2A model card without workspace_id returns error",
			method:         http.MethodGet,
			path:           "/v1/a2a/model-cards",
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "workspace_id",
		},
		{
			name:           "MCP invoke without tool returns error",
			method:         http.MethodPost,
			path:           "/mcp/v1/tools/invoke",
			body:           `{"input":{}}`,
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}

			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("RegisterAllRoutes() status = %v, want %v, body = %v", rec.Code, tt.wantStatusCode, rec.Body.String())
			}

			if tt.wantBody != "" && !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("RegisterAllRoutes() body = %v, want to contain %v", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestRegisterAllRoutes_MethodNotAllowed(t *testing.T) {
	repo := &mockA2ARepo{}
	taskStore := &mockTaskStore{}
	gateway := &mockGateway{
		repo:      repo,
		taskStore: taskStore,
	}

	mux := http.NewServeMux()
	RegisterAllRoutes(mux, gateway)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "A2A model cards POST to specific path not allowed",
			method: http.MethodPost,
			path:   "/v1/a2a/model-cards/card-123",
		},
		{
			name:   "MCP tools list POST not allowed",
			method: http.MethodPost,
			path:   "/mcp/v1/tools/list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("RegisterAllRoutes() status = %v, want %v", rec.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestGatewayInterface(t *testing.T) {
	// Test that mockGateway implements the Gateway interface
	var _ Gateway = (*mockGateway)(nil)

	repo := &mockA2ARepo{}
	taskStore := &mockTaskStore{}
	gateway := &mockGateway{
		repo:      repo,
		taskStore: taskStore,
	}

	if gateway.GetA2ARepo() == nil {
		t.Error("GetA2ARepo() returned nil")
	}

	if gateway.GetA2ATaskStore() == nil {
		t.Error("GetA2ATaskStore() returned nil")
	}
}

func TestGatewayWrapper(t *testing.T) {
	repo := &mockA2ARepo{}
	taskStore := &mockTaskStore{}

	// Test NewGatewayWrapper
	wrapper := NewGatewayWrapper(repo, taskStore, nil)

	if wrapper == nil {
		t.Fatal("NewGatewayWrapper returned nil")
	}

	// Test that wrapper implements Gateway interface
	var _ Gateway = wrapper

	// Test GetA2ARepo
	gotRepo := wrapper.GetA2ARepo()
	if gotRepo == nil {
		t.Error("GetA2ARepo() returned nil")
	}

	// Test GetA2ATaskStore
	gotTaskStore := wrapper.GetA2ATaskStore()
	if gotTaskStore == nil {
		t.Error("GetA2ATaskStore() returned nil")
	}
}

func TestRegisterAllRoutes_NilGateway(t *testing.T) {
	// Test that RegisterAllRoutes handles nil gateway gracefully
	mux := http.NewServeMux()
	RegisterAllRoutes(mux, nil)

	// Verify that at least some routes are registered by checking MCP health
	req := httptest.NewRequest(http.MethodGet, "/mcp/v1/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("RegisterAllRoutes with nil gateway: MCP health status = %v, want %v", rec.Code, http.StatusOK)
	}
}

func TestRegisterAllRoutes_WithGatewayWrapper(t *testing.T) {
	repo := &mockA2ARepo{}
	taskStore := &mockTaskStore{}

	// Create gateway wrapper
	wrapper := NewGatewayWrapper(repo, taskStore, nil)

	// Create mux and register routes using wrapper
	mux := http.NewServeMux()
	RegisterAllRoutes(mux, wrapper)

	// Test that routes work
	req := httptest.NewRequest(http.MethodGet, "/v1/a2a/model-cards?workspace_id=test", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("RegisterAllRoutes with wrapper status = %v, want %v", rec.Code, http.StatusOK)
	}
}

func TestMCPRoutes_ResponseFormat(t *testing.T) {
	repo := &mockA2ARepo{}
	taskStore := &mockTaskStore{}
	gateway := &mockGateway{
		repo:      repo,
		taskStore: taskStore,
	}

	mux := http.NewServeMux()
	RegisterAllRoutes(mux, gateway)

	// Test MCP tools list response format
	req := httptest.NewRequest(http.MethodGet, "/mcp/v1/tools/list", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("MCP tools list status = %v, want %v", rec.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse MCP tools list response: %v", err)
	}

	if _, ok := response["tools"]; !ok {
		t.Error("MCP tools list response missing 'tools' field")
	}

	if _, ok := response["count"]; !ok {
		t.Error("MCP tools list response missing 'count' field")
	}

	// Test MCP health response format
	req = httptest.NewRequest(http.MethodGet, "/mcp/v1/health", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("MCP health status = %v, want %v", rec.Code, http.StatusOK)
	}

	var healthResponse map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &healthResponse); err != nil {
		t.Errorf("Failed to parse MCP health response: %v", err)
	}

	if _, ok := healthResponse["status"]; !ok {
		t.Error("MCP health response missing 'status' field")
	}

	if _, ok := healthResponse["service"]; !ok {
		t.Error("MCP health response missing 'service' field")
	}
}

func TestA2ARoutes_ResponseFormat(t *testing.T) {
	repo := &mockA2ARepo{}
	taskStore := &mockTaskStore{}
	gateway := &mockGateway{
		repo:      repo,
		taskStore: taskStore,
	}

	mux := http.NewServeMux()
	RegisterAllRoutes(mux, gateway)

	// Test A2A model cards list response format
	req := httptest.NewRequest(http.MethodGet, "/v1/a2a/model-cards?workspace_id=test-workspace", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("A2A model cards list status = %v, want %v", rec.Code, http.StatusOK)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("A2A response Content-Type = %v, want application/json", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse A2A model cards response: %v", err)
	}

	if _, ok := response["items"]; !ok {
		t.Error("A2A model cards response missing 'items' field")
	}

	if _, ok := response["total"]; !ok {
		t.Error("A2A model cards response missing 'total' field")
	}
}

// Test MCP invoke with different tools
func TestMCP_InvokeTools(t *testing.T) {
	repo := &mockA2ARepo{}
	taskStore := &mockTaskStore{}
	gateway := &mockGateway{
		repo:      repo,
		taskStore: taskStore,
	}

	mux := http.NewServeMux()
	RegisterAllRoutes(mux, gateway)

	tests := []struct {
		name           string
		tool           string
		input          string
		wantStatusCode int
		wantSuccess    bool
	}{
		{
			name:           "Echo tool",
			tool:           "echo",
			input:          `{"tool":"echo","input":{"content":"hello world"}}`,
			wantStatusCode: http.StatusOK,
			wantSuccess:    true,
		},
		{
			name:           "Time tool",
			tool:           "time",
			input:          `{"tool":"time","input":{}}`,
			wantStatusCode: http.StatusOK,
			wantSuccess:    true,
		},
		{
			name:           "JSON parse tool",
			tool:           "json_parse",
			input:          `{"tool":"json_parse","input":{"data":"{\"key\":\"value\"}"}}`,
			wantStatusCode: http.StatusOK,
			wantSuccess:    true,
		},
		{
			name:           "Unknown tool",
			tool:           "unknown",
			input:          `{"tool":"unknown","input":{}}`,
			wantStatusCode: http.StatusOK,
			wantSuccess:    true, // Built-in unknown tool returns success with simulation message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/mcp/v1/tools/invoke", strings.NewReader(tt.input))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("MCP invoke %s status = %v, want %v", tt.name, rec.Code, tt.wantStatusCode)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to parse MCP invoke response: %v", err)
				return
			}

			if success, ok := response["success"].(bool); ok && success != tt.wantSuccess {
				t.Errorf("MCP invoke %s success = %v, want %v", tt.name, success, tt.wantSuccess)
			}
		})
	}
}

// Test that A2A legacy routes (without /v1 prefix) are also registered
func TestA2A_LegacyRoutes(t *testing.T) {
	repo := &mockA2ARepo{}
	taskStore := &mockTaskStore{}
	gateway := &mockGateway{
		repo:      repo,
		taskStore: taskStore,
	}

	mux := http.NewServeMux()
	RegisterAllRoutes(mux, gateway)

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		wantStatusCode int
	}{
		{
			name:           "Legacy A2A model cards list",
			method:         http.MethodGet,
			path:           "/a2a/model-cards?workspace_id=test-workspace",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "Legacy A2A model card get",
			method:         http.MethodGet,
			path:           "/a2a/model-cards/card-123",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "Legacy A2A tasks get",
			method:         http.MethodGet,
			path:           "/a2a/tasks/task-123",
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}

			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("Legacy route %s status = %v, want %v", tt.name, rec.Code, tt.wantStatusCode)
			}
		})
	}
}

// Test AG-UI SSE endpoint with context timeout
func TestAGUI_SSEEndpoint(t *testing.T) {
	repo := &mockA2ARepo{}
	taskStore := &mockTaskStore{}
	gateway := &mockGateway{
		repo:      repo,
		taskStore: taskStore,
	}

	mux := http.NewServeMux()
	RegisterAllRoutes(mux, gateway)

	tests := []struct {
		name           string
		method         string
		path           string
		wantStatusCode int
	}{
		{
			name:           "AG-UI agent stream endpoint with valid params",
			method:         http.MethodGet,
			path:           "/v1/agents/agent-123/stream?threadId=thread-456",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "AG-UI without threadId returns error",
			method:         http.MethodGet,
			path:           "/v1/agents/agent-123/stream",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "AG-UI POST not allowed",
			method:         http.MethodPost,
			path:           "/v1/agents/agent-123/stream?threadId=thread-456",
			wantStatusCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context with timeout for SSE tests to prevent hanging
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			req := httptest.NewRequest(tt.method, tt.path, nil).WithContext(ctx)
			rec := httptest.NewRecorder()

			// Run in goroutine to handle potential timeout
			done := make(chan bool)
			go func() {
				mux.ServeHTTP(rec, req)
				done <- true
			}()

			select {
			case <-done:
				// Request completed
				if rec.Code != tt.wantStatusCode {
					t.Errorf("AG-UI %s status = %v, want %v", tt.name, rec.Code, tt.wantStatusCode)
				}
			case <-ctx.Done():
				// Expected for SSE endpoint with valid params - it streams indefinitely
				if tt.wantStatusCode == http.StatusOK && strings.Contains(tt.path, "threadId=") {
					// This is expected - the SSE endpoint streams forever
					return
				}
				t.Errorf("AG-UI %s timed out unexpectedly", tt.name)
			}
		})
	}
}
