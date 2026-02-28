// Package agui provides AG-UI (Agent-User Interface) protocol support for RAD Gateway.
// This file implements the SSE (Server-Sent Events) handler for real-time event streaming.
package agui

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"radgateway/internal/logger"
)

// Handler manages SSE connections for AG-UI event streaming.
// It handles client registration, event broadcasting, and connection lifecycle.
type Handler struct {
	mu      sync.RWMutex
	clients map[string]*Client
	log     *slog.Logger
}

// Client represents a connected SSE client.
// Each client maintains a buffered channel for receiving events.
type Client struct {
	// ID is the unique identifier for this client connection
	ID string
	// AgentID identifies the agent this client is subscribed to
	AgentID string
	// ThreadID identifies the specific thread this client is subscribed to
	ThreadID string
	// Events is a buffered channel for receiving events
	Events chan Event
}

// NewHandler creates a new AG-UI SSE handler with initialized state.
func NewHandler() *Handler {
	return &Handler{
		clients: make(map[string]*Client),
		log:     logger.WithComponent("agui_handler"),
	}
}

// RegisterRoutes registers the AG-UI routes on the provided mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/agents/", h.handleAgentStream)
}

// handleAgentStream handles SSE connections at /v1/agents/{agentId}/stream?threadId=...
// It sets up proper SSE headers and streams events to connected clients.
func (h *Handler) handleAgentStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from path
	agentID, err := extractAgentID(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get thread ID from query parameter
	threadID := r.URL.Query().Get("threadId")
	if threadID == "" {
		http.Error(w, "threadId query parameter is required", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Set status code before flushing
	w.WriteHeader(http.StatusOK)

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Flush headers immediately
	flusher.Flush()

	// Create client
	clientID := fmt.Sprintf("%s-%s-%p", agentID, threadID, r)
	client := &Client{
		ID:       clientID,
		AgentID:  agentID,
		ThreadID: threadID,
		Events:   make(chan Event, 100),
	}

	// Register client
	h.registerClient(client)
	defer h.unregisterClient(client)

	h.log.Info("client connected",
		"client_id", clientID,
		"agent_id", agentID,
		"thread_id", threadID,
	)

	// Send initial connection event
	connectEvent := Event{
		Type:      EventTypeStateSnapshot,
		RunID:     "",
		AgentID:   agentID,
		ThreadID:  threadID,
		Timestamp: time.Now().UTC(),
		Data: map[string]interface{}{
			"status":     "connected",
			"client_id":  clientID,
			"message":    "Successfully connected to AG-UI event stream",
		},
	}

	if err := h.sendEvent(w, connectEvent); err != nil {
		h.log.Error("failed to send connection event", "error", err, "client_id", clientID)
		return
	}

	// Event loop - wait for events or connection close
	ctx := r.Context()
	for {
		select {
		case event, ok := <-client.Events:
			if !ok {
				h.log.Info("client event channel closed", "client_id", clientID)
				return
			}
			if err := h.sendEvent(w, event); err != nil {
				h.log.Error("failed to send event", "error", err, "client_id", clientID)
				return
			}
		case <-ctx.Done():
			h.log.Info("client disconnected", "client_id", clientID, "reason", ctx.Err())
			return
		}
	}
}

// sendEvent sends a single SSE formatted event to the client.
// Events are formatted as: data: <json>\n\n
func (h *Handler) sendEvent(w http.ResponseWriter, event Event) error {
	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	// Write SSE format: data: <json>\n\n
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	if err != nil {
		return fmt.Errorf("write event: %w", err)
	}

	// Get flusher and flush to ensure event is sent immediately
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

// Broadcast sends an event to all matching clients.
// Events are filtered by AgentID and ThreadID. If ThreadID is empty,
// events are broadcast to all threads for the specified agent.
func (h *Handler) Broadcast(event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		// Match by agent ID
		if client.AgentID != event.AgentID {
			continue
		}

		// If event has a specific thread ID, only send to matching threads
		// If event thread ID is empty, broadcast to all threads for this agent
		if event.ThreadID != "" && client.ThreadID != event.ThreadID {
			continue
		}

		// Send event to client (non-blocking)
		select {
		case client.Events <- event:
			// Event sent successfully
		default:
			// Client buffer full, log and skip
			h.log.Warn("client event buffer full, dropping event",
				"client_id", client.ID,
				"event_type", event.Type,
			)
		}
	}
}

// registerClient adds a client to the handler's client map.
func (h *Handler) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.ID] = client
	h.log.Debug("client registered",
		"client_id", client.ID,
		"total_clients", len(h.clients),
	)
}

// unregisterClient removes a client from the handler's client map
// and closes its event channel.
func (h *Handler) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[client.ID]; exists {
		close(client.Events)
		delete(h.clients, client.ID)
		h.log.Debug("client unregistered",
			"client_id", client.ID,
			"total_clients", len(h.clients),
		)
	}
}

// GetClientCount returns the number of currently connected clients.
func (h *Handler) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// extractAgentID extracts the agent ID from the request path.
// Expected path format: /v1/agents/{agentId}/stream
func extractAgentID(path string) (string, error) {
	// Check that path ends with /stream
	if !strings.HasSuffix(path, "/stream") {
		return "", fmt.Errorf("invalid path format: missing /stream suffix")
	}

	// Path should be: /v1/agents/{agentId}/stream
	parts := strings.Split(path, "/")

	// Find "agents" in path
	agentsIndex := -1
	for i, part := range parts {
		if part == "agents" {
			agentsIndex = i
			break
		}
	}

	if agentsIndex == -1 || agentsIndex+1 >= len(parts) {
		return "", fmt.Errorf("invalid path format")
	}

	agentID := parts[agentsIndex+1]
	if agentID == "" || agentID == "stream" {
		return "", fmt.Errorf("agent ID is required")
	}

	return agentID, nil
}
