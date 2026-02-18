// Package api provides tests for Server-Sent Events functionality.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"radgateway/internal/provider"
	"radgateway/internal/streaming"
)

// Ensure mockHealthChecker implements HealthChecker interface
var _ HealthChecker = (*mockHealthChecker)(nil)

// mockHealthChecker implements a mock health checker for testing.
type mockHealthChecker struct {
	statuses map[string]provider.HealthStatus
}

func (m *mockHealthChecker) Check(ctx context.Context, prov string) provider.HealthStatus {
	return m.statuses[prov]
}

func (m *mockHealthChecker) OnStatus(callback func(provider string, status provider.HealthStatus)) {}

func newMockHealthChecker() *mockHealthChecker {
	return &mockHealthChecker{
		statuses: map[string]provider.HealthStatus{
			"openai":    {Provider: "openai", Healthy: true, Latency: 100 * time.Millisecond},
			"anthropic": {Provider: "anthropic", Healthy: true, Latency: 150 * time.Millisecond},
		},
	}
}

func TestSSEHandler_NewSSEHandler(t *testing.T) {
	hc := newMockHealthChecker()
	handler := NewSSEHandler(hc)

	if handler == nil {
		t.Fatal("NewSSEHandler returned nil")
	}

	if handler.clients == nil {
		t.Error("clients map not initialized")
	}

	if handler.heartbeatInterval != 30*time.Second {
		t.Errorf("unexpected heartbeat interval: %v", handler.heartbeatInterval)
	}

	if handler.maxClients != 100 {
		t.Errorf("unexpected max clients: %d", handler.maxClients)
	}
}

func TestSSEHandler_RegisterRoutes(t *testing.T) {
	t.Skip("TODO: Fix SSE handler test - requires mocking Flusher interface")
}

func TestSSEHandler_handleEvents_MethodNotAllowed(t *testing.T) {
	hc := newMockHealthChecker()
	handler := NewSSEHandler(hc)

	req := httptest.NewRequest(http.MethodPost, "/v0/admin/events", nil)
	w := httptest.NewRecorder()

	handler.handleEvents(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestSSEHandler_handleEvents_RequiresFlusher(t *testing.T) {
	hc := newMockHealthChecker()
	handler := NewSSEHandler(hc)

	// Create a response writer that doesn't support flushing
	w := &mockResponseWriter{}
	req := httptest.NewRequest(http.MethodGet, "/v0/admin/events", nil)

	handler.handleEvents(w, req)

	if w.statusCode != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.statusCode)
	}
}

func TestSSEHandler_parseEventTypes(t *testing.T) {
	hc := newMockHealthChecker()
	handler := NewSSEHandler(hc)

	testCases := []struct {
		query    string
		expected map[EventType]bool
	}{
		{"", map[EventType]bool{}}, // Empty = all events
		{"usage:realtime", map[EventType]bool{EventTypeUsageRealtime: true}},
		{"usage:realtime,provider:health", map[EventType]bool{
			EventTypeUsageRealtime:  true,
			EventTypeProviderHealth: true,
		}},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(http.MethodGet, "/v0/admin/events?events="+tc.query, nil)
		result := handler.parseEventTypes(req)

		if len(result) != len(tc.expected) {
			t.Errorf("query '%s': expected %d event types, got %d", tc.query, len(tc.expected), len(result))
			continue
		}

		for eventType := range tc.expected {
			if !result[eventType] {
				t.Errorf("query '%s': expected event type %s to be present", tc.query, eventType)
			}
		}
	}
}

func TestSSEHandler_splitEvents(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"single", []string{"single"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{",leading", []string{"leading"}},
		{"trailing,", []string{"trailing"}},
		{",", nil},
	}

	for _, tc := range testCases {
		result := splitEvents(tc.input)

		if len(result) != len(tc.expected) {
			t.Errorf("splitEvents('%s'): expected %v, got %v", tc.input, tc.expected, result)
			continue
		}

		for i, expected := range tc.expected {
			if result[i] != expected {
				t.Errorf("splitEvents('%s')[%d]: expected '%s', got '%s'", tc.input, i, expected, result[i])
			}
		}
	}
}

func TestSSEHandler_Broadcast(t *testing.T) {
	hc := newMockHealthChecker()
	handler := NewSSEHandler(hc)

	// Create test event
	event := Event{
		Type:    EventTypeUsageRealtime,
		Payload: map[string]interface{}{"requestsPerSecond": 100},
		ID:      "test-1",
		Time:    time.Now(),
	}

	// Broadcast should not panic even with no clients
	handler.Broadcast(event)

	// Test that event data is properly formatted
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if decoded.Type != EventTypeUsageRealtime {
		t.Errorf("expected type %s, got %s", EventTypeUsageRealtime, decoded.Type)
	}
}

func TestSSEHandler_SendTo(t *testing.T) {
	hc := newMockHealthChecker()
	handler := NewSSEHandler(hc)

	// Test sending to non-existent client
	event := Event{
		Type:    EventTypeHeartbeat,
		Payload: map[string]string{"status": "test"},
	}

	err := handler.SendTo("non-existent-client", event)
	if err == nil {
		t.Error("expected error when sending to non-existent client")
	}
}

func TestSSEHandler_GetConnectionCount(t *testing.T) {
	hc := newMockHealthChecker()
	handler := NewSSEHandler(hc)

	if handler.GetConnectionCount() != 0 {
		t.Errorf("expected 0 connections, got %d", handler.GetConnectionCount())
	}
}

func TestSSEHandler_CloseAll(t *testing.T) {
	hc := newMockHealthChecker()
	handler := NewSSEHandler(hc)

	// Should not panic with no clients
	handler.CloseAll()
}

func TestEvent_Struct(t *testing.T) {
	event := Event{
		Type:    EventTypeUsageRealtime,
		Payload: UsageRealtimeEvent{RequestsPerSecond: 100},
		ID:      "test-id",
		Time:    time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if decoded.Type != EventTypeUsageRealtime {
		t.Errorf("expected type %s, got %s", EventTypeUsageRealtime, decoded.Type)
	}

	if decoded.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got '%s'", decoded.ID)
	}
}

func TestUsageRealtimeEvent_Struct(t *testing.T) {
	now := time.Now()
	event := UsageRealtimeEvent{
		RequestsPerSecond: 100.5,
		LatencyMs:         250.5,
		ActiveConnections: 50,
		RequestQueueDepth: 10,
		Timestamp:         now,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal UsageRealtimeEvent: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal UsageRealtimeEvent: %v", err)
	}

	if decoded["requestsPerSecond"] != 100.5 {
		t.Errorf("expected requestsPerSecond 100.5, got %v", decoded["requestsPerSecond"])
	}

	if decoded["latencyMs"] != 250.5 {
		t.Errorf("expected latencyMs 250.5, got %v", decoded["latencyMs"])
	}
}

func TestProviderHealthEvent_Struct(t *testing.T) {
	now := time.Now()
	event := ProviderHealthEvent{
		Provider:  "openai",
		Status:    "healthy",
		LatencyMs: 100.5,
		CheckedAt: now,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal ProviderHealthEvent: %v", err)
	}

	var decoded ProviderHealthEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal ProviderHealthEvent: %v", err)
	}

	if decoded.Provider != "openai" {
		t.Errorf("expected provider 'openai', got '%s'", decoded.Provider)
	}
}

func TestCircuitBreakerEvent_Struct(t *testing.T) {
	now := time.Now()
	event := CircuitBreakerEvent{
		Provider:  "anthropic",
		State:     "open",
		Reason:    "high error rate",
		Timestamp: now,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal CircuitBreakerEvent: %v", err)
	}

	var decoded CircuitBreakerEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal CircuitBreakerEvent: %v", err)
	}

	if decoded.State != "open" {
		t.Errorf("expected state 'open', got '%s'", decoded.State)
	}

	if decoded.Reason != "high error rate" {
		t.Errorf("expected reason 'high error rate', got '%s'", decoded.Reason)
	}
}

func TestSystemAlertEvent_Struct(t *testing.T) {
	now := time.Now()
	event := SystemAlertEvent{
		ID:        "alert-1",
		Severity:  "warning",
		Title:     "High Latency",
		Message:   "Request latency above threshold",
		Timestamp: now,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal SystemAlertEvent: %v", err)
	}

	var decoded SystemAlertEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal SystemAlertEvent: %v", err)
	}

	if decoded.Severity != "warning" {
		t.Errorf("expected severity 'warning', got '%s'", decoded.Severity)
	}
}

// mockResponseWriter is a response writer that doesn't support flushing
type mockResponseWriter struct {
	statusCode int
	header     http.Header
	body       strings.Builder
}

func (m *mockResponseWriter) Header() http.Header {
	if m.header == nil {
		m.header = make(http.Header)
	}
	return m.header
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	return m.body.Write(b)
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

func TestSSEHandler_collectMetrics(t *testing.T) {
	hc := newMockHealthChecker()
	handler := NewSSEHandler(hc)

	metrics := handler.collectMetrics()

	// Should return placeholder values
	if metrics.ActiveConnections < 0 {
		t.Error("active connections should be non-negative")
	}

	if metrics.Timestamp.IsZero() {
		t.Error("timestamp should be set")
	}
}

func TestSSEHandler_handleSubscribe_InvalidMethod(t *testing.T) {
	hc := newMockHealthChecker()
	handler := NewSSEHandler(hc)

	req := httptest.NewRequest(http.MethodGet, "/v0/admin/events/subscribe", nil)
	w := httptest.NewRecorder()

	handler.handleSubscribe(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestSSEHandler_handleSubscribe_InvalidBody(t *testing.T) {
	hc := newMockHealthChecker()
	handler := NewSSEHandler(hc)

	req := httptest.NewRequest(http.MethodPost, "/v0/admin/events/subscribe", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()

	handler.handleSubscribe(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestSSEHandler_handleSubscribe_ClientNotFound(t *testing.T) {
	hc := newMockHealthChecker()
	handler := NewSSEHandler(hc)

	body := `{"client_id": "non-existent", "events": ["usage:realtime"]}`
	req := httptest.NewRequest(http.MethodPost, "/v0/admin/events/subscribe", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.handleSubscribe(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestSSEHandler_sendEvent_SubscriptionFiltering(t *testing.T) {
	hc := newMockHealthChecker()
	handler := NewSSEHandler(hc)

	// Create a mock client that subscribes only to usage:realtime
	client := &Client{
		id:               "test-client",
		subscribedEvents: map[EventType]bool{EventTypeUsageRealtime: true},
	}

	// Try to send a provider:health event - should be filtered out
	event := Event{
		Type:    EventTypeProviderHealth,
		Payload: ProviderHealthEvent{Provider: "openai"},
	}

	err := handler.sendEvent(client, event)
	if err != nil {
		t.Errorf("sendEvent should return nil for filtered events, got: %v", err)
	}

	// Try to send a usage:realtime event - should be attempted (will fail without real connection)
	event2 := Event{
		Type:    EventTypeUsageRealtime,
		Payload: UsageRealtimeEvent{},
	}

	err = handler.sendEvent(client, event2)
	// This will fail because we don't have a real SSE connection, but it proves the filtering works
	if err == nil {
		t.Error("expected error when sending to client without SSE connection")
	}
}

// Integration test for SSE streaming
func TestSSEHandler_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	hc := newMockHealthChecker()
	handler := NewSSEHandler(hc)

	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(handler.handleEvents))
	defer ts.Close()

	// Make request
	resp, err := http.Get(ts.URL + "?events=heartbeat")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer resp.Body.Close()

	// Check headers
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		t.Errorf("expected Content-Type 'text/event-stream', got '%s'", contentType)
	}

	cacheControl := resp.Header.Get("Cache-Control")
	if cacheControl != "no-cache" {
		t.Errorf("expected Cache-Control 'no-cache', got '%s'", cacheControl)
	}

	// Read first event (should be connection event)
	reader := streaming.NewParser(resp.Body)
	event, err := reader.Next()
	if err != nil {
		t.Fatalf("failed to read first event: %v", err)
	}

	if event.Event != "heartbeat" {
		t.Errorf("expected event type 'heartbeat', got '%s'", event.Event)
	}
}
