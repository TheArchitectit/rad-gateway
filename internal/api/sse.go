// Package api provides Server-Sent Events (SSE) endpoints for real-time updates.
// This implements the backend streaming infrastructure for Control Rooms.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"radgateway/internal/logger"
	"radgateway/internal/provider"
	"radgateway/internal/streaming"
)

// EventType represents the type of SSE event.
type EventType string

const (
	// EventTypeUsageRealtime provides real-time usage metrics.
	EventTypeUsageRealtime EventType = "usage:realtime"
	// EventTypeProviderHealth provides provider health status updates.
	EventTypeProviderHealth EventType = "provider:health"
	// EventTypeCircuitBreaker provides circuit breaker state changes.
	EventTypeCircuitBreaker EventType = "provider:circuit"
	// EventTypeSystemAlert provides system alerts and notifications.
	EventTypeSystemAlert EventType = "system:alert"
	// EventTypeHeartbeat is a keepalive event.
	EventTypeHeartbeat EventType = "heartbeat"
)

// Event represents a typed event for SSE streaming.
type Event struct {
	Type    EventType   `json:"type"`
	Payload interface{} `json:"payload"`
	ID      string      `json:"id"`
	Time    time.Time   `json:"time"`
}

// UsageRealtimeEvent provides real-time usage metrics.
type UsageRealtimeEvent struct {
	RequestsPerSecond   float64   `json:"requestsPerSecond"`
	LatencyMs           float64   `json:"latencyMs"`
	ActiveConnections   int       `json:"activeConnections"`
	RequestQueueDepth   int       `json:"requestQueueDepth"`
	Timestamp           time.Time `json:"timestamp"`
}

// ProviderHealthEvent provides provider health updates.
type ProviderHealthEvent struct {
	Provider   string    `json:"provider"`
	Status     string    `json:"status"`
	LatencyMs  float64   `json:"latencyMs"`
	CheckedAt  time.Time `json:"checkedAt"`
}

// CircuitBreakerEvent provides circuit breaker state changes.
type CircuitBreakerEvent struct {
	Provider  string    `json:"provider"`
	State     string    `json:"state"`
	Reason    string    `json:"reason,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// SystemAlertEvent provides system alerts.
type SystemAlertEvent struct {
	ID        string    `json:"id"`
	Severity  string    `json:"severity"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// HealthChecker is the interface for health checking providers.
type HealthChecker interface {
	Check(ctx context.Context, provider string) provider.HealthStatus
	OnStatus(callback func(provider string, status provider.HealthStatus))
}

// SSEHandler manages SSE connections and event broadcasting.
type SSEHandler struct {
	clients     map[string]*Client
	clientsMu   sync.RWMutex
	healthCheck HealthChecker
	log         *slog.Logger

	// Configuration
	heartbeatInterval time.Duration
	reconnectTimeout  time.Duration
	maxClients        int
}

// Client represents an SSE client connection.
type Client struct {
	id       string
	sseClient *streaming.Client
	subscribedEvents map[EventType]bool
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewSSEHandler creates a new SSE handler.
func NewSSEHandler(healthCheck HealthChecker) *SSEHandler {
	h := &SSEHandler{
		clients:           make(map[string]*Client),
		healthCheck:       healthCheck,
		log:               logger.WithComponent("sse_handler"),
		heartbeatInterval: 30 * time.Second,
		reconnectTimeout:  10 * time.Second,
		maxClients:        100,
	}

	// Start background goroutines
	go h.heartbeatLoop()
	go h.metricsCollectionLoop()

	return h
}

// RegisterRoutes registers SSE endpoints on the mux.
func (h *SSEHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v0/admin/events", h.handleEvents)
	mux.HandleFunc("/v0/admin/events/subscribe", h.handleSubscribe)
	h.log.Info("SSE routes registered", "endpoints", []string{"/v0/admin/events", "/v0/admin/events/subscribe"})
}

// handleEvents is the main SSE endpoint for real-time events.
// Authentication is expected to be handled by middleware before reaching this handler.
func (h *SSEHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":{"message":"method not allowed","code":405}}`, http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters for event filtering
	eventTypes := h.parseEventTypes(r)

	// Create SSE client connection
	sseClient, err := streaming.NewClient(w, r)
	if err != nil {
		h.log.Error("failed to create SSE client", "error", err, "remote_addr", r.RemoteAddr)
		http.Error(w, `{"error":{"message":"streaming not supported","code":500}}`, http.StatusInternalServerError)
		return
	}

	// Generate client ID
	clientID := fmt.Sprintf("%s-%d", r.RemoteAddr, time.Now().UnixNano())

	// Create context for this client
	ctx, cancel := context.WithCancel(r.Context())

	client := &Client{
		id:               clientID,
		sseClient:        sseClient,
		subscribedEvents: eventTypes,
		ctx:              ctx,
		cancel:           cancel,
	}

	// Register client
	h.clientsMu.Lock()
	if len(h.clients) >= h.maxClients {
		h.clientsMu.Unlock()
		cancel()
		http.Error(w, `{"error":{"message":"too many connections","code":503}}`, http.StatusServiceUnavailable)
		return
	}
	h.clients[clientID] = client
	h.clientsMu.Unlock()

	h.log.Info("SSE client connected", "client_id", clientID, "events", r.URL.Query().Get("events"))

	// Cleanup on disconnect
	defer func() {
		h.clientsMu.Lock()
		delete(h.clients, clientID)
		h.clientsMu.Unlock()
		cancel()
		sseClient.Close()
		h.log.Info("SSE client disconnected", "client_id", clientID)
	}()

	// Send initial connection event
	if err := h.sendEvent(client, Event{
		Type: EventTypeHeartbeat,
		Payload: map[string]string{
			"status":    "connected",
			"client_id": clientID,
		},
		ID:   "init",
		Time: time.Now(),
	}); err != nil {
		h.log.Error("failed to send initial event", "error", err, "client_id", clientID)
		return
	}

	// Keep connection alive until client disconnects
	select {
	case <-ctx.Done():
		h.log.Debug("client context done", "client_id", clientID)
	case <-sseClient.Done():
		h.log.Debug("client connection closed", "client_id", clientID)
	}
}

// handleSubscribe allows clients to subscribe/unsubscribe from event types.
func (h *SSEHandler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, `{"error":{"message":"method not allowed","code":405}}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ClientID string   `json:"client_id"`
		Events   []string `json:"events"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"message":"invalid request body","code":400}}`, http.StatusBadRequest)
		return
	}

	h.clientsMu.RLock()
	client, exists := h.clients[req.ClientID]
	h.clientsMu.RUnlock()

	if !exists {
		http.Error(w, `{"error":{"message":"client not found","code":404}}`, http.StatusNotFound)
		return
	}

	// Update subscriptions
	if r.Method == http.MethodPost {
		// Subscribe to events
		for _, eventType := range req.Events {
			client.subscribedEvents[EventType(eventType)] = true
		}
	} else {
		// Unsubscribe from events
		for _, eventType := range req.Events {
			delete(client.subscribedEvents, EventType(eventType))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"client_id":     req.ClientID,
		"subscribed_to": h.getSubscribedEvents(client),
	})
}

// Broadcast sends an event to all connected clients subscribed to that event type.
func (h *SSEHandler) Broadcast(event Event) {
	h.clientsMu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for _, client := range h.clients {
		if client.subscribedEvents[event.Type] || len(client.subscribedEvents) == 0 {
			clients = append(clients, client)
		}
	}
	h.clientsMu.RUnlock()

	for _, client := range clients {
		if err := h.sendEvent(client, event); err != nil {
			h.log.Error("failed to send event to client", "error", err, "client_id", client.id, "event_type", event.Type)
		}
	}
}

// SendTo sends an event to a specific client.
func (h *SSEHandler) SendTo(clientID string, event Event) error {
	h.clientsMu.RLock()
	client, exists := h.clients[clientID]
	h.clientsMu.RUnlock()

	if !exists {
		return fmt.Errorf("client not found: %s", clientID)
	}

	return h.sendEvent(client, event)
}

// sendEvent sends an event to a client, converting to SSE format.
func (h *SSEHandler) sendEvent(client *Client, event Event) error {
	// Check if client is subscribed to this event type
	if len(client.subscribedEvents) > 0 && !client.subscribedEvents[event.Type] {
		return nil
	}

	// Check if client has valid SSE connection
	if client.sseClient == nil {
		return fmt.Errorf("client has no SSE connection")
	}

	// Marshal event payload to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	// Create SSE event
	sseEvent := streaming.Event{
		ID:    event.ID,
		Event: string(event.Type),
		Data:  string(data),
	}

	// Send with timeout
	done := make(chan error, 1)
	go func() {
		done <- client.sseClient.Send(sseEvent)
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("send event: %w", err)
		}
	case <-time.After(5 * time.Second):
		return fmt.Errorf("send event timeout")
	}

	return nil
}

// parseEventTypes extracts event types from query parameters.
func (h *SSEHandler) parseEventTypes(r *http.Request) map[EventType]bool {
	events := r.URL.Query().Get("events")
	if events == "" {
		// Subscribe to all events by default
		return map[EventType]bool{}
	}

	result := make(map[EventType]bool)
	for _, eventType := range splitEvents(events) {
		result[EventType(eventType)] = true
	}
	return result
}

// getSubscribedEvents returns a slice of subscribed event types.
func (h *SSEHandler) getSubscribedEvents(client *Client) []string {
	events := make([]string, 0, len(client.subscribedEvents))
	for eventType := range client.subscribedEvents {
		events = append(events, string(eventType))
	}
	return events
}

// splitEvents splits a comma-separated list of event types.
func splitEvents(events string) []string {
	if events == "" {
		return nil
	}

	// Simple split by comma
	var result []string
	start := 0
	for i := 0; i < len(events); i++ {
		if events[i] == ',' {
			if i > start {
				result = append(result, events[start:i])
			}
			start = i + 1
		}
	}
	if start < len(events) {
		result = append(result, events[start:])
	}
	return result
}

// heartbeatLoop sends periodic heartbeats to keep connections alive.
func (h *SSEHandler) heartbeatLoop() {
	ticker := time.NewTicker(h.heartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		event := Event{
			Type:    EventTypeHeartbeat,
			Payload: map[string]interface{}{"timestamp": time.Now().Unix()},
			ID:      fmt.Sprintf("hb-%d", time.Now().Unix()),
			Time:    time.Now(),
		}
		h.Broadcast(event)
	}
}

// metricsCollectionLoop collects and broadcasts real-time metrics.
func (h *SSEHandler) metricsCollectionLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Collect current metrics
		metrics := h.collectMetrics()

		event := Event{
			Type:    EventTypeUsageRealtime,
			Payload: metrics,
			ID:      fmt.Sprintf("metrics-%d", time.Now().Unix()),
			Time:    time.Now(),
		}
		h.Broadcast(event)

		// Check provider health and broadcast updates
		if h.healthCheck != nil {
			h.broadcastProviderHealth()
		}
	}
}

// collectMetrics gathers current system metrics.
func (h *SSEHandler) collectMetrics() UsageRealtimeEvent {
	// Get active connection count
	h.clientsMu.RLock()
	activeConnections := len(h.clients)
	h.clientsMu.RUnlock()

	// TODO: Integrate with actual metrics collection from core.Gateway
	// For now, return placeholder values
	return UsageRealtimeEvent{
		RequestsPerSecond: 0, // TODO: Get from gateway metrics
		LatencyMs:         0, // TODO: Get from gateway metrics
		ActiveConnections: activeConnections,
		RequestQueueDepth: 0, // TODO: Get from gateway queue
		Timestamp:         time.Now(),
	}
}

// broadcastProviderHealth broadcasts health status for all providers.
func (h *SSEHandler) broadcastProviderHealth() {
	// TODO: Implement provider health checking
	// This would iterate through all providers and broadcast their health status
}

// GetConnectionCount returns the current number of connected clients.
func (h *SSEHandler) GetConnectionCount() int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	return len(h.clients)
}

// CloseAll closes all active SSE connections.
func (h *SSEHandler) CloseAll() {
	h.clientsMu.Lock()
	clients := make([]*Client, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}
	h.clients = make(map[string]*Client)
	h.clientsMu.Unlock()

	for _, client := range clients {
		client.cancel()
		client.sseClient.Close()
	}

	h.log.Info("all SSE connections closed", "count", len(clients))
}
