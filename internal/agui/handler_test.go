// Package agui provides AG-UI (Agent-User Interface) protocol support for RAD Gateway.
package agui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewHandler(t *testing.T) {
	h := NewHandler()
	if h == nil {
		t.Fatal("NewHandler() returned nil")
	}
	if h.clients == nil {
		t.Error("handler clients map not initialized")
	}
	// Verify the handler is properly configured by checking it works
	if h.GetClientCount() != 0 {
		t.Error("new handler should have 0 clients")
	}
}

func TestHandler_RegisterRoutes(t *testing.T) {
	h := NewHandler()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Test that the route is registered by making a request with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/v1/agents/test-agent/stream?threadId=test-thread", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	// Run in goroutine since handler blocks
	done := make(chan bool)
	go func() {
		mux.ServeHTTP(rr, req)
		done <- true
	}()

	// Wait for handler to complete or timeout
	select {
	case <-done:
		// Handler completed
	case <-time.After(200 * time.Millisecond):
		// Timeout is expected
	}

	// Should not return 404 since route is registered
	if rr.Code == http.StatusNotFound {
		t.Error("Route /v1/agents/{agentId}/stream not registered")
	}
}

func TestHandler_HandleAgentStream_MethodNotAllowed(t *testing.T) {
	h := NewHandler()

	// Test POST method (should be rejected)
	req := httptest.NewRequest(http.MethodPost, "/v1/agents/test-agent/stream?threadId=test-thread", nil)
	rr := httptest.NewRecorder()

	h.handleAgentStream(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestHandler_HandleAgentStream_InvalidPath(t *testing.T) {
	h := NewHandler()

	// Test invalid path (missing agentId)
	req := httptest.NewRequest(http.MethodGet, "/v1/agents//stream?threadId=test-thread", nil)
	req = req.WithContext(context.WithValue(req.Context(), http.LocalAddrContextKey, nil))
	// Manually set path to bypass URL parsing
	req.URL.Path = "/v1/agents//stream"
	rr := httptest.NewRecorder()

	h.handleAgentStream(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandler_HandleAgentStream_MissingThreadId(t *testing.T) {
	h := NewHandler()

	// Test missing threadId
	req := httptest.NewRequest(http.MethodGet, "/v1/agents/test-agent/stream", nil)
	rr := httptest.NewRecorder()

	h.handleAgentStream(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandler_HandleAgentStream_SSEHeaders(t *testing.T) {
	h := NewHandler()

	// Use a context with cancel to stop the stream
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/v1/agents/test-agent/stream?threadId=test-thread", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	// Run in goroutine since handler blocks
	done := make(chan bool)
	go func() {
		h.handleAgentStream(rr, req)
		done <- true
	}()

	// Wait for handler to start or timeout
	select {
	case <-done:
		// Handler completed (likely due to client disconnect)
	case <-time.After(200 * time.Millisecond):
		// Timeout is expected
	}

	// Check headers were set
	if ct := rr.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Expected Content-Type 'text/event-stream', got '%s'", ct)
	}
	if cc := rr.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Expected Cache-Control 'no-cache', got '%s'", cc)
	}
	if conn := rr.Header().Get("Connection"); conn != "keep-alive" {
		t.Errorf("Expected Connection 'keep-alive', got '%s'", conn)
	}
	if xab := rr.Header().Get("X-Accel-Buffering"); xab != "no" {
		t.Errorf("Expected X-Accel-Buffering 'no', got '%s'", xab)
	}
}

func TestHandler_SendEvent(t *testing.T) {
	h := NewHandler()

	event := Event{
		Type:      EventTypeStateSnapshot,
		RunID:     "run-123",
		AgentID:   "agent-456",
		ThreadID:  "thread-789",
		Timestamp: time.Now().UTC(),
		Data: map[string]interface{}{
			"message": "test data",
		},
	}

	rr := httptest.NewRecorder()

	err := h.sendEvent(rr, event)
	if err != nil {
		t.Errorf("sendEvent() error = %v", err)
	}

	body := rr.Body.String()
	if !strings.HasPrefix(body, "data: ") {
		t.Errorf("Expected SSE data prefix, got: %s", body)
	}
	if !strings.HasSuffix(body, "\n\n") {
		t.Errorf("Expected SSE double newline ending, got: %s", body)
	}

	// Verify JSON content
	jsonPart := strings.TrimPrefix(body, "data: ")
	jsonPart = strings.TrimSuffix(jsonPart, "\n\n")

	var receivedEvent Event
	if err := json.Unmarshal([]byte(jsonPart), &receivedEvent); err != nil {
		t.Errorf("Failed to unmarshal event JSON: %v", err)
	}

	if receivedEvent.Type != event.Type {
		t.Errorf("Expected event type %s, got %s", event.Type, receivedEvent.Type)
	}
	if receivedEvent.RunID != event.RunID {
		t.Errorf("Expected run_id %s, got %s", event.RunID, receivedEvent.RunID)
	}
}

func TestHandler_ClientRegistration(t *testing.T) {
	h := NewHandler()

	client := &Client{
		ID:       "client-1",
		AgentID:  "agent-1",
		ThreadID: "thread-1",
		Events:   make(chan Event, 100),
	}

	// Register client
	h.registerClient(client)

	h.mu.RLock()
	if _, exists := h.clients[client.ID]; !exists {
		t.Error("Client not registered")
	}
	h.mu.RUnlock()

	// Unregister client
	h.unregisterClient(client)

	h.mu.RLock()
	if _, exists := h.clients[client.ID]; exists {
		t.Error("Client still registered after unregister")
	}
	h.mu.RUnlock()
}

func TestHandler_Broadcast(t *testing.T) {
	h := NewHandler()

	// Create test clients
	client1 := &Client{
		ID:       "client-1",
		AgentID:  "agent-1",
		ThreadID: "thread-1",
		Events:   make(chan Event, 100),
	}
	client2 := &Client{
		ID:       "client-2",
		AgentID:  "agent-1",
		ThreadID: "thread-1",
		Events:   make(chan Event, 100),
	}
	client3 := &Client{
		ID:       "client-3",
		AgentID:  "agent-2",
		ThreadID: "thread-2",
		Events:   make(chan Event, 100),
	}

	h.registerClient(client1)
	h.registerClient(client2)
	h.registerClient(client3)

	// Broadcast event for agent-1, thread-1
	event := Event{
		Type:      EventTypeMessageDelta,
		RunID:     "run-1",
		AgentID:   "agent-1",
		ThreadID:  "thread-1",
		Timestamp: time.Now().UTC(),
		Data:      map[string]interface{}{"content": "test"},
	}

	h.Broadcast(event)

	// Give some time for broadcast to complete
	time.Sleep(50 * time.Millisecond)

	// client1 should receive the event
	select {
	case received := <-client1.Events:
		if received.Type != event.Type {
			t.Errorf("client1: expected event type %s, got %s", event.Type, received.Type)
		}
	default:
		t.Error("client1 did not receive event")
	}

	// client2 should receive the event
	select {
	case received := <-client2.Events:
		if received.Type != event.Type {
			t.Errorf("client2: expected event type %s, got %s", event.Type, received.Type)
		}
	default:
		t.Error("client2 did not receive event")
	}

	// client3 should NOT receive the event (different agent/thread)
	select {
	case <-client3.Events:
		t.Error("client3 should not have received event for different agent/thread")
	default:
		// Expected - no event
	}
}

func TestHandler_Broadcast_FilterByAgentOnly(t *testing.T) {
	h := NewHandler()

	// Create test clients with same agent but different threads
	client1 := &Client{
		ID:       "client-1",
		AgentID:  "agent-1",
		ThreadID: "thread-1",
		Events:   make(chan Event, 100),
	}
	client2 := &Client{
		ID:       "client-2",
		AgentID:  "agent-1",
		ThreadID: "thread-2",
		Events:   make(chan Event, 100),
	}

	h.registerClient(client1)
	h.registerClient(client2)

	// Broadcast event for agent-1 only (no thread filter)
	event := Event{
		Type:      EventTypeRunStart,
		RunID:     "run-1",
		AgentID:   "agent-1",
		ThreadID:  "", // Empty thread ID should broadcast to all threads for this agent
		Timestamp: time.Now().UTC(),
	}

	h.Broadcast(event)

	// Give some time for broadcast to complete
	time.Sleep(50 * time.Millisecond)

	// Both clients should receive since they're on the same agent
	select {
	case <-client1.Events:
		// Expected
	default:
		t.Error("client1 should receive event when ThreadID is empty")
	}

	select {
	case <-client2.Events:
		// Expected
	default:
		t.Error("client2 should receive event when ThreadID is empty")
	}
}

func TestHandler_ConcurrentClients(t *testing.T) {
	h := NewHandler()

	// Test concurrent registration/unregistration
	const numClients = 100
	done := make(chan bool, numClients*2)

	for i := 0; i < numClients; i++ {
		go func(id int) {
			client := &Client{
				ID:       fmt.Sprintf("client-%d", id),
				AgentID:  "agent-1",
				ThreadID: "thread-1",
				Events:   make(chan Event, 100),
			}
			h.registerClient(client)
			done <- true
		}(i)
	}

	for i := 0; i < numClients; i++ {
		go func(id int) {
			client := &Client{
				ID:       fmt.Sprintf("client-%d", id),
				AgentID:  "agent-1",
				ThreadID: "thread-1",
				Events:   make(chan Event, 100),
			}
			// Small delay to ensure registration happens first
			time.Sleep(time.Millisecond)
			h.unregisterClient(client)
			done <- true
		}(i)
	}

	// Wait for all operations
	for i := 0; i < numClients*2; i++ {
		<-done
	}

	// Handler should remain in consistent state
	h.mu.RLock()
	clientCount := len(h.clients)
	h.mu.RUnlock()

	// May have some clients left depending on timing, but should not panic
	t.Logf("Remaining clients after concurrent ops: %d", clientCount)
}

func TestHandler_ExtractPathParams(t *testing.T) {
	tests := []struct {
		path      string
		wantAgent string
		wantErr   bool
	}{
		{"/v1/agents/agent-123/stream", "agent-123", false},
		{"/v1/agents/my-agent/stream", "my-agent", false},
		{"/v1/agents//stream", "", true},
		{"/v1/agents/stream", "", true},
		{"/v1/agents/agent-123/extra/stream", "agent-123", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			agentID, err := extractAgentID(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractAgentID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if agentID != tt.wantAgent {
				t.Errorf("extractAgentID() = %v, want %v", agentID, tt.wantAgent)
			}
		})
	}
}

func TestHandler_SSEEventFormat(t *testing.T) {
	h := NewHandler()

	event := Event{
		Type:      EventTypeStateSnapshot,
		RunID:     "run-abc",
		AgentID:   "agent-xyz",
		ThreadID:  "thread-123",
		Timestamp: time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC),
		Data: map[string]interface{}{
			"status": "running",
		},
		Metadata: map[string]string{
			"source": "test",
		},
	}

	rr := httptest.NewRecorder()

	err := h.sendEvent(rr, event)
	if err != nil {
		t.Fatalf("sendEvent() error = %v", err)
	}

	body := rr.Body.String()

	// Parse the SSE format
	scanner := bufio.NewScanner(strings.NewReader(body))
	var dataLine string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			dataLine = strings.TrimPrefix(line, "data: ")
			break
		}
	}

	if dataLine == "" {
		t.Fatal("No data line found in SSE output")
	}

	// Verify JSON structure
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(dataLine), &result); err != nil {
		t.Fatalf("Failed to parse SSE data as JSON: %v", err)
	}

	// Check snake_case field names
	if _, ok := result["run_id"]; !ok {
		t.Error("Missing run_id field")
	}
	if _, ok := result["agent_id"]; !ok {
		t.Error("Missing agent_id field")
	}
	if _, ok := result["thread_id"]; !ok {
		t.Error("Missing thread_id field")
	}
	if _, ok := result["type"]; !ok {
		t.Error("Missing type field")
	}
	if _, ok := result["timestamp"]; !ok {
		t.Error("Missing timestamp field")
	}

	// Verify values
	if result["run_id"] != "run-abc" {
		t.Errorf("Expected run_id 'run-abc', got %v", result["run_id"])
	}
	if result["agent_id"] != "agent-xyz" {
		t.Errorf("Expected agent_id 'agent-xyz', got %v", result["agent_id"])
	}
}

