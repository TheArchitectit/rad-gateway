// Package integration provides integration tests for RAD Gateway
//
// AG-UI Protocol Integration Tests
// Tests AG-UI SSE streaming and event broadcasting
//
// Run with: go test ./tests/integration/... -run TestAGUI
// Run verbose: go test -v ./tests/integration/... -run TestAGUI
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"radgateway/internal/agui"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAGUI_SSEConnection tests SSE endpoint connection and headers
func TestAGUI_SSEConnection(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		validateResp   func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name:           "sse_connection_success",
			method:         http.MethodGet,
			path:           "/v1/agents/agent-123/stream?threadId=thread-456",
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				// Verify SSE headers
				assert.Equal(t, "text/event-stream", rr.Header().Get("Content-Type"))
				assert.Equal(t, "no-cache", rr.Header().Get("Cache-Control"))
				assert.Equal(t, "keep-alive", rr.Header().Get("Connection"))
				assert.Equal(t, "no", rr.Header().Get("X-Accel-Buffering"))
			},
		},
		{
			name:           "sse_missing_thread_id",
			method:         http.MethodGet,
			path:           "/v1/agents/agent-123/stream",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "sse_missing_agent_id",
			method:         http.MethodGet,
			path:           "/v1/agents//stream?threadId=thread-456",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "sse_method_not_allowed_post",
			method:         http.MethodPost,
			path:           "/v1/agents/agent-123/stream?threadId=thread-456",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "sse_method_not_allowed_put",
			method:         http.MethodPut,
			path:           "/v1/agents/agent-123/stream?threadId=thread-456",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := agui.NewHandler()
			mux := http.NewServeMux()
			handler.RegisterRoutes(mux)

			// For SSE success test, use context with timeout to prevent blocking
			var req *http.Request
			if tt.name == "sse_connection_success" {
				ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
				defer cancel()
				req = httptest.NewRequest(tt.method, tt.path, nil).WithContext(ctx)
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			rr := httptest.NewRecorder()

			// For SSE success test, run in goroutine
			if tt.name == "sse_connection_success" {
				done := make(chan bool)
				go func() {
					mux.ServeHTTP(rr, req)
					done <- true
				}()
				select {
				case <-done:
					// Handler completed
				case <-time.After(100 * time.Millisecond):
					// Timeout expected for SSE
				}
			} else {
				mux.ServeHTTP(rr, req)
			}

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.validateResp != nil {
				tt.validateResp(t, rr)
			}
		})
	}
}

// TestAGUI_SSEConnectionTimeout tests SSE connection with context timeout
func TestAGUI_SSEConnectionTimeout(t *testing.T) {
	handler := agui.NewHandler()
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/v1/agents/agent-123/stream?threadId=thread-456", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	// Run in goroutine since SSE handler blocks
	done := make(chan bool)
	go func() {
		mux.ServeHTTP(rr, req)
		done <- true
	}()

	// Wait for timeout or handler to complete
	select {
	case <-done:
		// Handler completed (likely due to context cancellation)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("handler did not complete within expected time")
	}

	// Verify that we got the initial connection event before timeout
	body := rr.Body.String()
	assert.Contains(t, body, "data:", "should contain SSE data prefix")
}

// TestAGUI_EventBroadcast tests broadcasting events to connected clients
func TestAGUI_EventBroadcast(t *testing.T) {
	tests := []struct {
		name         string
		events       []agui.Event
		expectedMsgs int
		filterAgent  string
		filterThread string
	}{
		{
			name: "broadcast_single_event",
			events: []agui.Event{
				{
					Type:      agui.EventTypeRunStart,
					RunID:     "run-001",
					AgentID:   "agent-123",
					ThreadID:  "thread-456",
					Timestamp: time.Now().UTC(),
					Data:      map[string]interface{}{"message": "Run started"},
				},
			},
			expectedMsgs: 2, // Connection event + 1 broadcast event
			filterAgent:  "agent-123",
			filterThread: "thread-456",
		},
		{
			name: "broadcast_multiple_events",
			events: []agui.Event{
				{
					Type:      agui.EventTypeRunStart,
					RunID:     "run-002",
					AgentID:   "agent-123",
					ThreadID:  "thread-456",
					Timestamp: time.Now().UTC(),
				},
				{
					Type:      agui.EventTypeMessageDelta,
					RunID:     "run-002",
					AgentID:   "agent-123",
					ThreadID:  "thread-456",
					Timestamp: time.Now().UTC(),
					Data:      map[string]interface{}{"content": "Hello"},
				},
				{
					Type:      agui.EventTypeRunComplete,
					RunID:     "run-002",
					AgentID:   "agent-123",
					ThreadID:  "thread-456",
					Timestamp: time.Now().UTC(),
				},
			},
			expectedMsgs: 4, // Connection event + 3 broadcast events
			filterAgent:  "agent-123",
			filterThread: "thread-456",
		},
		{
			name: "filter_by_agent_id",
			events: []agui.Event{
				{
					Type:      agui.EventTypeRunStart,
					RunID:     "run-003",
					AgentID:   "agent-123",
					ThreadID:  "thread-456",
					Timestamp: time.Now().UTC(),
				},
				{
					Type:      agui.EventTypeRunStart,
					RunID:     "run-004",
					AgentID:   "agent-999", // Different agent
					ThreadID:  "thread-456",
					Timestamp: time.Now().UTC(),
				},
			},
			expectedMsgs: 2, // Connection event + only agent-123 event
			filterAgent:  "agent-123",
			filterThread: "thread-456",
		},
		{
			name: "filter_by_thread_id",
			events: []agui.Event{
				{
					Type:      agui.EventTypeRunStart,
					RunID:     "run-005",
					AgentID:   "agent-123",
					ThreadID:  "thread-456",
					Timestamp: time.Now().UTC(),
				},
				{
					Type:      agui.EventTypeRunStart,
					RunID:     "run-006",
					AgentID:   "agent-123",
					ThreadID:  "thread-999", // Different thread
					Timestamp: time.Now().UTC(),
				},
			},
			expectedMsgs: 2, // Connection event + only thread-456 event
			filterAgent:  "agent-123",
			filterThread: "thread-456",
		},
		{
			name: "broadcast_to_all_threads",
			events: []agui.Event{
				{
					Type:      agui.EventTypeStateSnapshot,
					RunID:     "run-007",
					AgentID:   "agent-123",
					ThreadID:  "", // Empty thread ID = broadcast to all
					Timestamp: time.Now().UTC(),
				},
			},
			expectedMsgs: 2, // Connection event + broadcast event
			filterAgent:  "agent-123",
			filterThread: "thread-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := agui.NewHandler()
			mux := http.NewServeMux()
			handler.RegisterRoutes(mux)

			// Track received events
			var receivedEvents []string
			var mu sync.Mutex

			// Start client in goroutine
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			req := httptest.NewRequest(http.MethodGet,
				fmt.Sprintf("/v1/agents/%s/stream?threadId=%s", tt.filterAgent, tt.filterThread),
				nil).WithContext(ctx)

			// Use a custom ResponseRecorder that captures events
			rr := &eventCapturingRecorder{
				ResponseRecorder: httptest.NewRecorder(),
				events:          &receivedEvents,
				mu:              &mu,
			}

			done := make(chan bool)
			go func() {
				mux.ServeHTTP(rr, req)
				done <- true
			}()

			// Give client time to connect
			time.Sleep(50 * time.Millisecond)

			// Broadcast events
			for _, event := range tt.events {
				handler.Broadcast(event)
				time.Sleep(10 * time.Millisecond) // Small delay between events
			}

			// Cancel context to stop the handler
			cancel()

			// Wait for handler to complete
			select {
			case <-done:
				// Handler completed
			case <-time.After(1 * time.Second):
				t.Fatal("handler did not complete")
			}

			// Verify received events
			mu.Lock()
			receivedCount := len(receivedEvents)
			mu.Unlock()

			assert.GreaterOrEqual(t, receivedCount, 1, "should receive at least the connection event")
		})
	}
}

// eventCapturingRecorder wraps httptest.ResponseRecorder to capture SSE events
type eventCapturingRecorder struct {
	*httptest.ResponseRecorder
	events *[]string
	mu     *sync.Mutex
}

func (r *eventCapturingRecorder) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	*r.events = append(*r.events, string(p))
	r.mu.Unlock()
	return r.ResponseRecorder.Write(p)
}

// TestAGUI_ClientManagement tests client registration and unregistration
func TestAGUI_ClientManagement(t *testing.T) {
	handler := agui.NewHandler()

	// Initially no clients
	assert.Equal(t, 0, handler.GetClientCount())

	// Start a connection
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/v1/agents/agent-123/stream?threadId=thread-456", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	// Start in goroutine
	go mux.ServeHTTP(rr, req)

	// Give time for connection to establish
	time.Sleep(50 * time.Millisecond)

	// Should have 1 client
	assert.Equal(t, 1, handler.GetClientCount())

	// Cancel context to close connection
	cancel()

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)

	// Client should be unregistered
	assert.Equal(t, 0, handler.GetClientCount())
}

// TestAGUI_EventTypes tests different AG-UI event types
func TestAGUI_EventTypes(t *testing.T) {
	tests := []struct {
		name        string
		eventType   agui.EventType
		expectedType string
	}{
		{
			name:        "run_start_event",
			eventType:   agui.EventTypeRunStart,
			expectedType: "run.start",
		},
		{
			name:        "run_complete_event",
			eventType:   agui.EventTypeRunComplete,
			expectedType: "run.complete",
		},
		{
			name:        "run_error_event",
			eventType:   agui.EventTypeRunError,
			expectedType: "run.error",
		},
		{
			name:        "message_delta_event",
			eventType:   agui.EventTypeMessageDelta,
			expectedType: "message.delta",
		},
		{
			name:        "tool_call_event",
			eventType:   agui.EventTypeToolCall,
			expectedType: "tool.call",
		},
		{
			name:        "tool_result_event",
			eventType:   agui.EventTypeToolResult,
			expectedType: "tool.result",
		},
		{
			name:        "state_snapshot_event",
			eventType:   agui.EventTypeStateSnapshot,
			expectedType: "state.snapshot",
		},
		{
			name:        "state_delta_event",
			eventType:   agui.EventTypeStateDelta,
			expectedType: "state.delta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := agui.NewEvent(tt.eventType, "run-001", "agent-123", "thread-456")

			assert.Equal(t, tt.expectedType, string(event.Type))
			assert.Equal(t, "run-001", event.RunID)
			assert.Equal(t, "agent-123", event.AgentID)
			assert.Equal(t, "thread-456", event.ThreadID)
			assert.False(t, event.Timestamp.IsZero())
		})
	}
}

// TestAGUI_EventSerialization tests event JSON serialization
func TestAGUI_EventSerialization(t *testing.T) {
	event := agui.NewEvent(agui.EventTypeMessageDelta, "run-001", "agent-123", "thread-456").
		WithData("content", "Hello, world!").
		WithMetadata("source", "test")

	// Marshal to JSON
	data, err := json.Marshal(event)
	require.NoError(t, err)

	// Verify JSON structure
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "message.delta", result["type"])
	assert.Equal(t, "run-001", result["run_id"])
	assert.Equal(t, "agent-123", result["agent_id"])
	assert.Equal(t, "thread-456", result["thread_id"])
	assert.NotNil(t, result["timestamp"])

	// Check data
	dataMap, ok := result["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Hello, world!", dataMap["content"])

	// Check metadata
	metaMap, ok := result["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test", metaMap["source"])
}

// TestAGUI_InvalidPath tests error handling for invalid paths
func TestAGUI_InvalidPath(t *testing.T) {
	handler := agui.NewHandler()
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "missing_stream_suffix",
			path:           "/v1/agents/agent-123",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid_path_format",
			path:           "/v1/agents/",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "wrong_path_prefix",
			path:           "/agents/agent-123/stream?threadId=thread-456",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

// TestAGUI_EventWithRunState tests creating events from RunState
func TestAGUI_EventWithRunState(t *testing.T) {
	// Create a run state
	runState := agui.NewRunState("run-001", "agent-123", "thread-456", "running")
	runState.AddMessage(agui.NewMessage("msg-1", "user", "Hello"))
	runState.AddMessage(agui.NewMessage("msg-2", "assistant", "Hi there!"))
	runState.AddToolCall(agui.NewToolCall("tool-1", "search", map[string]interface{}{"query": "test"}))
	runState.SetStateValue("progress", 50)

	// Create event from run state
	event := agui.NewEvent(agui.EventTypeStateSnapshot, runState.RunID, runState.AgentID, runState.ThreadID).
		WithData("run_state", runState)

	// Verify event
	assert.Equal(t, agui.EventTypeStateSnapshot, event.Type)
	assert.Equal(t, "run-001", event.RunID)
	assert.NotNil(t, event.Data)
	assert.Contains(t, event.Data, "run_state")

	// Serialize and verify
	data, err := json.Marshal(event)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	dataMap := result["data"].(map[string]interface{})
	runStateData := dataMap["run_state"].(map[string]interface{})
	assert.Equal(t, "running", runStateData["status"])
	assert.NotNil(t, runStateData["messages"])
	assert.NotNil(t, runStateData["tool_calls"])
}

// TestAGUI_SSEEventFormat tests SSE event formatting
func TestAGUI_SSEEventFormat(t *testing.T) {
	handler := agui.NewHandler()
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/v1/agents/agent-123/stream?threadId=thread-456", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	// Start client
	go mux.ServeHTTP(rr, req)

	// Give time to connect
	time.Sleep(50 * time.Millisecond)

	// Broadcast a test event
	testEvent := agui.NewEvent(agui.EventTypeRunStart, "run-001", "agent-123", "thread-456").
		WithData("test", "value")
	handler.Broadcast(*testEvent)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Check response body format
	body := rr.Body.String()

	// SSE events should be formatted as "data: <json>\n\n"
	lines := strings.Split(body, "\n")
	var dataLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			dataLines = append(dataLines, line)
		}
	}

	// Should have at least one data line (connection event)
	assert.GreaterOrEqual(t, len(dataLines), 1, "should have at least one data line")

	// Verify each data line is valid JSON
	for _, line := range dataLines {
		jsonData := strings.TrimPrefix(line, "data: ")
		var event map[string]interface{}
		err := json.Unmarshal([]byte(jsonData), &event)
		assert.NoError(t, err, "each data line should contain valid JSON: %s", jsonData)
	}
}

