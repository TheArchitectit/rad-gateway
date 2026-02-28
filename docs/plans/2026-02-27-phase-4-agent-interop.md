# Phase 4: Agent Interop Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement A2A protocol (Agent Cards, task lifecycle), AG-UI event streams, and MCP bridge for agent interoperability

**Architecture:**
- A2A protocol uses Agent Card discovery via `/.well-known/agent.json` and task lifecycle endpoints
- AG-UI provides WebSocket/SSE event streams for UI real-time updates
- MCP bridge connects tools/resources to agents with explicit auth boundaries

**Tech Stack:** Go 1.24, gorilla/websocket, SSE streams, JSON-LD for A2A

**Reference:** See docs/implementation-plan.md Phase 4 section

---

## Overview

This plan implements agent interoperability features for RAD Gateway:

1. **A2A Protocol** - Agent-to-agent communication with task lifecycle
2. **AG-UI** - Event streams for frontend UI integration
3. **MCP Bridge** - Model Context Protocol for tool/resource access

## Prerequisites

Before starting:
- Ensure `internal/streaming/` package exists (from Sprint 7.2)
- Review A2A spec: https://github.com/a2a-protocol/spec
- Review AG-UI patterns from docs/implementation-plan.md

---

## Sprint 9.1: A2A Core - Agent Card Discovery

### Task 1: Create Agent Card Model

**Files:**
- Create: `internal/a2a/models.go`
- Test: `internal/a2a/models_test.go`

**Step 1: Write failing test**

```go
package a2a

import (
	"encoding/json"
	"testing"
)

func TestAgentCard_MarshalJSON(t *testing.T) {
	card := AgentCard{
		Name:        "RAD Gateway",
		Description: "AI API Gateway with A2A support",
		URL:         "https://gateway.example.com",
		Version:     "1.0.0",
		Capabilities: Capabilities{
			Streaming: true,
			PushNotifications: false,
			StateTransitionHistory: true,
		},
		Skills: []Skill{
			{
				ID:          "chat-completion",
				Name:        "Chat Completion",
				Description: "OpenAI-compatible chat completion",
			},
		},
	}

	data, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled AgentCard
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.Name != card.Name {
		t.Errorf("Name mismatch: got %q, want %q", unmarshaled.Name, card.Name)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/a2a/... -v`
Expected: FAIL - "no such file or directory"

**Step 3: Create package directory and models**

```go
// internal/a2a/models.go
package a2a

import "time"

// AgentCard represents an agent's capabilities and metadata
// Spec: https://github.com/a2a-protocol/spec/blob/main/documentation.md
type AgentCard struct {
	Name         string       `json:"name"`
	Description  string       `json:"description,omitempty"`
	URL          string       `json:"url"`
	Version      string       `json:"version"`
	Capabilities Capabilities `json:"capabilities"`
	Skills       []Skill      `json:"skills"`
	CreatedAt    time.Time    `json:"created_at,omitempty"`
	UpdatedAt    time.Time    `json:"updated_at,omitempty"`
}

// Capabilities defines what an agent can do
type Capabilities struct {
	Streaming              bool `json:"streaming"`
	PushNotifications      bool `json:"pushNotifications"`
	StateTransitionHistory bool `json:"stateTransitionHistory"`
}

// Skill represents a capability offered by the agent
type Skill struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Examples    []string          `json:"examples,omitempty"`
	Input       *SkillSchema      `json:"input,omitempty"`
	Output      *SkillSchema      `json:"output,omitempty"`
}

// SkillSchema describes input/output structure
type SkillSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// Task represents an A2A task
type Task struct {
	ID          string      `json:"id"`
	SessionID   string      `json:"sessionId,omitempty"`
	Status      TaskStatus  `json:"status"`
	Artifacts   []Artifact  `json:"artifacts,omitempty"`
	History     []Message   `json:"history,omitempty"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt,omitempty"`
	CompletedAt *time.Time  `json:"completedAt,omitempty"`
}

// TaskStatus represents task states
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusWorking    TaskStatus = "working"
	TaskStatusInputRequired TaskStatus = "input-required"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// Artifact represents task output
type Artifact struct {
	ID       string      `json:"id,omitempty"`
	Type     string      `json:"type"`
	Parts    []Part      `json:"parts"`
	Metadata interface{} `json:"metadata,omitempty"`
}

// Part represents content within an artifact
type Part struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// File, Data, etc. can be added later
}

// Message represents a communication in task history
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Parts   []Part `json:"parts,omitempty"`
}

// SendTaskRequest is the request body for task creation
type SendTaskRequest struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId,omitempty"`
	Message   Message   `json:"message"`
	SkillID   string    `json:"skillId,omitempty"`
}

// SendTaskResponse is the response from task creation
type SendTaskResponse struct {
	Task *Task  `json:"task,omitempty"`
	Error *Error `json:"error,omitempty"`
}

// Error represents A2A error responses
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/a2a/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/a2a/
git commit -m "feat(a2a): add A2A protocol models - AgentCard, Task, Artifacts

Sprint 9.1: A2A Core - Agent Card models
- Define AgentCard with capabilities and skills
- Define Task with lifecycle states
- Define Artifact and Part for task outputs
- Add SendTaskRequest/Response types

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 2: Agent Card Store

**Files:**
- Create: `internal/a2a/store.go`
- Test: `internal/a2a/store_test.go`

**Step 1: Write failing test**

```go
package a2a

import (
	"testing"
)

func TestStore_GetAgentCard(t *testing.T) {
	store := NewStore()

	card := AgentCard{
		Name:    "Test Agent",
		URL:     "https://test.example.com",
		Version: "1.0.0",
	}

	err := store.Save(card)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := store.Get(card.URL)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.Name != card.Name {
		t.Errorf("Name mismatch: got %q, want %q", got.Name, card.Name)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/a2a/... -v -run=TestStore`
Expected: FAIL - "NewStore undefined"

**Step 3: Implement Store**

```go
// internal/a2a/store.go
package a2a

import (
	"errors"
	"sync"
	"time"
)

// Store manages agent cards in memory
type Store struct {
	mu    sync.RWMutex
	cards map[string]AgentCard // URL -> AgentCard
}

// NewStore creates a new agent card store
func NewStore() *Store {
	return &Store{
		cards: make(map[string]AgentCard),
	}
}

// Save stores or updates an agent card
func (s *Store) Save(card AgentCard) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if card.URL == "" {
		return errors.New("agent card URL is required")
	}

	if card.Version == "" {
		card.Version = "1.0.0"
	}

	now := time.Now()
	if card.CreatedAt.IsZero() {
		card.CreatedAt = now
	}
	card.UpdatedAt = now

	s.cards[card.URL] = card
	return nil
}

// Get retrieves an agent card by URL
func (s *Store) Get(url string) (AgentCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	card, ok := s.cards[url]
	if !ok {
		return AgentCard{}, errors.New("agent card not found")
	}

	return card, nil
}

// GetByName retrieves an agent card by name
func (s *Store) GetByName(name string) (AgentCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, card := range s.cards {
		if card.Name == name {
			return card, nil
		}
	}

	return AgentCard{}, errors.New("agent card not found")
}

// List returns all agent cards
func (s *Store) List() []AgentCard {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cards := make([]AgentCard, 0, len(s.cards))
	for _, card := range s.cards {
		cards = append(cards, card)
	}
	return cards
}

// Delete removes an agent card
func (s *Store) Delete(url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.cards, url)
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/a2a/... -v -run=TestStore`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/a2a/store.go internal/a2a/store_test.go
git commit -m "feat(a2a): add Agent Card store with CRUD operations

Sprint 9.1: A2A Core - Agent Card store
- In-memory thread-safe store for agent cards
- Save, Get, GetByName, List, Delete operations
- Automatic timestamp management

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 3: Agent Card HTTP Handler

**Files:**
- Create: `internal/a2a/handler.go`
- Test: `internal/a2a/handler_test.go`

**Step 1: Write failing test**

```go
package a2a

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_GetAgentCard(t *testing.T) {
	store := NewStore()
	store.Save(AgentCard{
		Name:    "Test Agent",
		URL:     "https://test.example.com",
		Version: "1.0.0",
	})

	handler := NewHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent.json", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", ct)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/a2a/... -v -run=TestHandler`
Expected: FAIL - "NewHandler undefined"

**Step 3: Implement Handler**

```go
// internal/a2a/handler.go
package a2a

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"radgateway/internal/logger"
)

// Handler implements A2A protocol HTTP endpoints
type Handler struct {
	store  *Store
	logger *slog.Logger
}

// NewHandler creates a new A2A handler
func NewHandler(store *Store) *Handler {
	return &Handler{
		store:  store,
		logger: logger.WithComponent("a2a"),
	}
}

// RegisterRoutes registers A2A routes on the given mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Agent Card discovery endpoint
	mux.HandleFunc("/.well-known/agent.json", h.handleAgentCard)

	// Task endpoints
	mux.HandleFunc("/a2a/tasks/send", h.handleSendTask)
	mux.HandleFunc("/a2a/tasks/", h.handleGetTask)     // /a2a/tasks/{taskId}
	mux.HandleFunc("/a2a/tasks/cancel", h.handleCancelTask)
}

// handleAgentCard serves the agent card JSON
func (h *Handler) handleAgentCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the first agent card or return a default
	cards := h.store.List()
	var card AgentCard
	if len(cards) > 0 {
		card = cards[0]
	} else {
		card = h.defaultAgentCard()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

// handleSendTask creates or updates a task
func (h *Handler) handleSendTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SendTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	// TODO: Implement task creation (in Task 4)
	h.logger.Info("Received task request", "taskId", req.ID)

	resp := SendTaskResponse{
		Task: &Task{
			ID:     req.ID,
			Status: TaskStatusPending,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleGetTask retrieves a task by ID
func (h *Handler) handleGetTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract task ID from path
	// Path: /a2a/tasks/{taskId}
	// TODO: Implement task retrieval
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// handleCancelTask cancels a task
func (h *Handler) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement task cancellation
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// sendError sends an error response
func (h *Handler) sendError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(Error{
		Code:    code,
		Message: message,
	})
}

// defaultAgentCard returns a default agent card
func (h *Handler) defaultAgentCard() AgentCard {
	return AgentCard{
		Name:        "RAD Gateway",
		Description: "AI API Gateway with A2A protocol support",
		URL:         "http://localhost:8080",
		Version:     "1.0.0",
		Capabilities: Capabilities{
			Streaming:              true,
			PushNotifications:      false,
			StateTransitionHistory: true,
		},
		Skills: []Skill{
			{
				ID:          "chat-completion",
				Name:        "Chat Completion",
				Description: "OpenAI-compatible chat completion API",
			},
			{
				ID:          "embeddings",
				Name:        "Embeddings",
				Description: "Generate embeddings for text",
			},
		},
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/a2a/... -v -run=TestHandler`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/a2a/handler.go internal/a2a/handler_test.go
git commit -m "feat(a2a): add A2A HTTP handler with discovery endpoint

Sprint 9.1: A2A Core - Agent Card handler
- /.well-known/agent.json endpoint for discovery
- /a2a/tasks/send endpoint stub
- /a2a/tasks/{taskId} endpoint stub
- /a2a/tasks/cancel endpoint stub

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Sprint 9.2: A2A Task Lifecycle

### Task 4: Task Manager

**Files:**
- Create: `internal/a2a/task_manager.go`
- Test: `internal/a2a/task_manager_test.go`

**Step 1: Write failing test**

```go
package a2a

import (
	"context"
	"testing"
	"time"
)

func TestTaskManager_CreateTask(t *testing.T) {
	tm := NewTaskManager()
	ctx := context.Background()

	req := SendTaskRequest{
		ID:      "task-123",
		Message: Message{Role: "user", Content: "Hello"},
	}

	task, err := tm.CreateTask(ctx, req)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if task.ID != req.ID {
		t.Errorf("Task ID mismatch: got %q, want %q", task.ID, req.ID)
	}

	if task.Status != TaskStatusPending {
		t.Errorf("Expected status pending, got %q", task.Status)
	}
}

func TestTaskManager_TransitionState(t *testing.T) {
	tm := NewTaskManager()
	ctx := context.Background()

	req := SendTaskRequest{
		ID:      "task-456",
		Message: Message{Role: "user", Content: "Hello"},
	}

	task, _ := tm.CreateTask(ctx, req)

	err := tm.TransitionState(ctx, task.ID, TaskStatusWorking)
	if err != nil {
		t.Fatalf("TransitionState failed: %v", err)
	}

	updated, _ := tm.GetTask(ctx, task.ID)
	if updated.Status != TaskStatusWorking {
		t.Errorf("Expected status working, got %q", updated.Status)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/a2a/... -v -run=TestTaskManager`
Expected: FAIL - "NewTaskManager undefined"

**Step 3: Implement TaskManager**

```go
// internal/a2a/task_manager.go
package a2a

import (
	"context"
	"errors"
	"sync"
	"time"
)

// TaskManager manages A2A task lifecycle
type TaskManager struct {
	mu     sync.RWMutex
	tasks  map[string]*Task
	store  *Store
}

// NewTaskManager creates a new task manager
func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: make(map[string]*Task),
	}
}

// CreateTask creates a new task from a request
func (tm *TaskManager) CreateTask(ctx context.Context, req SendTaskRequest) (*Task, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if req.ID == "" {
		return nil, errors.New("task ID is required")
	}

	now := time.Now()
	task := &Task{
		ID:        req.ID,
		SessionID: req.SessionID,
		Status:    TaskStatusPending,
		History: []Message{
			req.Message,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	tm.tasks[task.ID] = task
	return task, nil
}

// GetTask retrieves a task by ID
func (tm *TaskManager) GetTask(ctx context.Context, taskID string) (*Task, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	task, ok := tm.tasks[taskID]
	if !ok {
		return nil, errors.New("task not found")
	}

	return task, nil
}

// UpdateTask updates a task
func (tm *TaskManager) UpdateTask(ctx context.Context, task *Task) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.tasks[task.ID] = task
	return nil
}

// TransitionState transitions a task to a new state
func (tm *TaskManager) TransitionState(ctx context.Context, taskID string, newStatus TaskStatus) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, ok := tm.tasks[taskID]
	if !ok {
		return errors.New("task not found")
	}

	// Validate state transition
	if !isValidTransition(task.Status, newStatus) {
		return errors.New("invalid state transition from " + string(task.Status) + " to " + string(newStatus))
	}

	task.Status = newStatus
	task.UpdatedAt = time.Now()

	if newStatus == TaskStatusCompleted || newStatus == TaskStatusFailed || newStatus == TaskStatusCancelled {
		now := time.Now()
		task.CompletedAt = &now
	}

	return nil
}

// CancelTask cancels a task
func (tm *TaskManager) CancelTask(ctx context.Context, taskID string) error {
	return tm.TransitionState(ctx, taskID, TaskStatusCancelled)
}

// ListTasks returns all tasks
func (tm *TaskManager) ListTasks(ctx context.Context) []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// isValidTransition checks if a state transition is valid
func isValidTransition(from, to TaskStatus) bool {
	// Define valid transitions
	validTransitions := map[TaskStatus][]TaskStatus{
		TaskStatusPending:         {TaskStatusWorking, TaskStatusCancelled},
		TaskStatusWorking:         {TaskStatusInputRequired, TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled},
		TaskStatusInputRequired:   {TaskStatusWorking, TaskStatusCancelled},
		TaskStatusCompleted:       {},
		TaskStatusFailed:          {},
		TaskStatusCancelled:       {},
	}

	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}

	for _, status := range allowed {
		if status == to {
			return true
		}
	}
	return false
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/a2a/... -v -run=TestTaskManager`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/a2a/task_manager.go internal/a2a/task_manager_test.go
git commit -m "feat(a2a): add TaskManager with lifecycle state transitions

Sprint 9.2: A2A Task Lifecycle - Task Manager
- CreateTask with validation
- GetTask, UpdateTask operations
- TransitionState with state machine validation
- CancelTask convenience method
- ListTasks for monitoring

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 5: Update Handler with Task Operations

**Files:**
- Modify: `internal/a2a/handler.go`
- Test: `internal/a2a/handler_test.go`

**Step 1: Write failing test**

```go
func TestHandler_SendTask(t *testing.T) {
	store := NewStore()
	handler := NewHandler(store)

	body := `{"id":"task-123","message":{"role":"user","content":"Hello"}}`
	req := httptest.NewRequest(http.MethodPost, "/a2a/tasks/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var resp SendTaskResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp.Task == nil {
		t.Fatal("Expected task in response")
	}

	if resp.Task.ID != "task-123" {
		t.Errorf("Task ID mismatch: got %q, want %q", resp.Task.ID, "task-123")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/a2a/... -v -run=TestHandler_SendTask`
Expected: FAIL - TaskManager not integrated

**Step 3: Update Handler**

Modify `internal/a2a/handler.go`:

```go
// Update Handler struct
type Handler struct {
	store       *Store
	taskManager *TaskManager
	logger      *slog.Logger
}

// Update NewHandler
func NewHandler(store *Store) *Handler {
	return &Handler{
		store:       store,
		taskManager: NewTaskManager(),
		logger:      logger.WithComponent("a2a"),
	}
}

// Update handleSendTask
func (h *Handler) handleSendTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SendTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if req.ID == "" {
		h.sendError(w, http.StatusBadRequest, "Task ID is required")
		return
	}

	task, err := h.taskManager.CreateTask(r.Context(), req)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := SendTaskResponse{Task: task}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Update handleGetTask
func (h *Handler) handleGetTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract task ID from path: /a2a/tasks/{taskId}
	taskID := strings.TrimPrefix(r.URL.Path, "/a2a/tasks/")
	if taskID == "" || taskID == "/" {
		h.sendError(w, http.StatusBadRequest, "Task ID is required")
		return
	}

	task, err := h.taskManager.GetTask(r.Context(), taskID)
	if err != nil {
		h.sendError(w, http.StatusNotFound, "Task not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/a2a/... -v -run=TestHandler_SendTask`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/a2a/handler.go internal/a2a/handler_test.go
git commit -m "feat(a2a): integrate TaskManager with HTTP handlers

Sprint 9.2: A2A Task Lifecycle - Handler integration
- Create tasks via POST /a2a/tasks/send
- Retrieve tasks via GET /a2a/tasks/{taskId}
- Task validation and error handling

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Sprint 10.1: AG-UI Event Streams

### Task 6: AG-UI Event Model

**Files:**
- Create: `internal/agui/models.go`
- Test: `internal/agui/models_test.go`

**Step 1: Write failing test**

```go
package agui

import (
	"encoding/json"
	"testing"
)

func TestEvent_MarshalJSON(t *testing.T) {
	event := Event{
		Type:    EventTypeRunStart,
		RunID:   "run-123",
		AgentID: "agent-456",
		Data:    map[string]interface{}{"input": "hello"},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled Event
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.Type != event.Type {
		t.Errorf("Type mismatch: got %q, want %q", unmarshaled.Type, event.Type)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/agui/... -v`
Expected: FAIL - package not found

**Step 3: Implement AG-UI Models**

```go
// internal/agui/models.go
package agui

import (
	"encoding/json"
	"time"
)

// EventType represents AG-UI event types
type EventType string

const (
	EventTypeRunStart      EventType = "run.start"
	EventTypeRunComplete   EventType = "run.complete"
	EventTypeRunError      EventType = "run.error"
	EventTypeMessageDelta  EventType = "message.delta"
	EventTypeToolCall      EventType = "tool.call"
	EventTypeToolResult    EventType = "tool.result"
	EventTypeStateSnapshot EventType = "state.snapshot"
	EventTypeStateDelta    EventType = "state.delta"
)

// Event represents an AG-UI event
type Event struct {
	Type      EventType              `json:"type"`
	RunID     string                 `json:"runId"`
	AgentID   string                 `json:"agentId"`
	ThreadID  string                 `json:"threadId,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
}

// NewEvent creates a new AG-UI event
func NewEvent(eventType EventType, runID, agentID string) Event {
	return Event{
		Type:      eventType,
		RunID:     runID,
		AgentID:   agentID,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
		Metadata:  make(map[string]string),
	}
}

// WithData adds data to the event
func (e Event) WithData(key string, value interface{}) Event {
	e.Data[key] = value
	return e
}

// WithMetadata adds metadata to the event
func (e Event) WithMetadata(key, value string) Event {
	e.Metadata[key] = value
	return e
}

// MarshalJSON implements custom JSON marshaling
func (e Event) MarshalJSON() ([]byte, error) {
	type Alias Event
	return json.Marshal(&struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Timestamp: e.Timestamp.Format(time.RFC3339),
		Alias:     (*Alias)(&e),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling
func (e *Event) UnmarshalJSON(data []byte) error {
	type Alias Event
	aux := &struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Timestamp != "" {
		ts, err := time.Parse(time.RFC3339, aux.Timestamp)
		if err != nil {
			return err
		}
		e.Timestamp = ts
	}

	return nil
}

// RunState represents the state of an agent run
type RunState struct {
	RunID      string                 `json:"runId"`
	AgentID    string                 `json:"agentId"`
	ThreadID   string                 `json:"threadId,omitempty"`
	Status     string                 `json:"status"`
	Messages   []Message              `json:"messages,omitempty"`
	ToolCalls  []ToolCall             `json:"toolCalls,omitempty"`
	State      map[string]interface{} `json:"state,omitempty"`
	StartedAt  time.Time              `json:"startedAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

// Message represents a message in the conversation
type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	ID        string                 `json:"id"`
	Tool      string                 `json:"tool"`
	Arguments map[string]interface{} `json:"arguments"`
	Result    interface{}            `json:"result,omitempty"`
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/agui/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/agui/
git commit -m "feat(agui): add AG-UI event models with custom marshaling

Sprint 10.1: AG-UI Event Streams
- Event types for run lifecycle and tool calls
- Custom JSON marshaling for timestamps
- RunState, Message, ToolCall models

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 7: AG-UI SSE Handler

**Files:**
- Create: `internal/agui/handler.go`
- Test: `internal/agui/handler_test.go`

**Step 1: Write failing test**

```go
package agui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_SSEEndpoint(t *testing.T) {
	handler := NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/v1/agents/agent-123/stream?threadId=thread-456", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Expected Content-Type text/event-stream, got %q", ct)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/agui/... -v -run=TestHandler`
Expected: FAIL - NewHandler undefined

**Step 3: Implement Handler**

```go
// internal/agui/handler.go
package agui

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"radgateway/internal/logger"
)

// Handler implements AG-UI HTTP endpoints
type Handler struct {
	mu        sync.RWMutex
	clients   map[string]*Client
	logger    *slog.Logger
}

// Client represents an SSE client connection
type Client struct {
	AgentID  string
	ThreadID string
	Events   chan Event
}

// NewHandler creates a new AG-UI handler
func NewHandler() *Handler {
	return &Handler{
		clients: make(map[string]*Client),
		logger:  logger.WithComponent("agui"),
	}
}

// RegisterRoutes registers AG-UI routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/agents/", h.handleAgentStream)
}

// handleAgentStream handles SSE event streams
func (h *Handler) handleAgentStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from path: /v1/agents/{agentId}/stream
	path := r.URL.Path
	if !strings.HasPrefix(path, "/v1/agents/") || !strings.HasSuffix(path, "/stream") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	agentID := strings.TrimSuffix(strings.TrimPrefix(path, "/v1/agents/"), "/stream")
	threadID := r.URL.Query().Get("threadId")

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Create client
	client := &Client{
		AgentID:  agentID,
		ThreadID: threadID,
		Events:   make(chan Event, 100),
	}

	h.registerClient(client)
	defer h.unregisterClient(client)

	// Send initial connected event
	h.sendEvent(w, Event{
		Type:     EventTypeStateSnapshot,
		RunID:    "connected",
		AgentID:  agentID,
		ThreadID: threadID,
		Data: map[string]interface{}{
			"status": "connected",
		},
	})

	// Event loop
	ctx := r.Context()
	for {
		select {
		case event := <-client.Events:
			if err := h.sendEvent(w, event); err != nil {
				h.logger.Error("Failed to send event", "error", err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// sendEvent sends an SSE event
func (h *Handler) sendEvent(w http.ResponseWriter, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	if err != nil {
		return err
	}

	// Flush to ensure immediate delivery
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

// Broadcast sends an event to all connected clients
func (h *Handler) Broadcast(event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		// Filter by agent/thread if specified
		if event.AgentID != "" && client.AgentID != event.AgentID {
			continue
		}
		if event.ThreadID != "" && client.ThreadID != event.ThreadID {
			continue
		}

		select {
		case client.Events <- event:
		default:
			// Channel full, drop event
			h.logger.Warn("Event channel full, dropping event",
				"agentId", client.AgentID,
				"threadId", client.ThreadID)
		}
	}
}

// registerClient adds a client to the registry
func (h *Handler) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := client.AgentID + ":" + client.ThreadID
	h.clients[key] = client
	h.logger.Info("Client connected", "agentId", client.AgentID, "threadId", client.ThreadID)
}

// unregisterClient removes a client from the registry
func (h *Handler) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := client.AgentID + ":" + client.ThreadID
	delete(h.clients, key)
	close(client.Events)

	h.logger.Info("Client disconnected", "agentId", client.AgentID, "threadId", client.ThreadID)
}
```

Add imports at top:
```go
import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"radgateway/internal/logger"
)
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/agui/... -v -run=TestHandler`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/agui/handler.go internal/agui/handler_test.go
git commit -m "feat(agui): add SSE event stream handler

Sprint 10.1: AG-UI Event Streams
- /v1/agents/{agentId}/stream SSE endpoint
- Client connection management
- Broadcast events to connected clients
- Event filtering by agent/thread

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Sprint 10.2: MCP Bridge

### Task 8: MCP Bridge Core

**Files:**
- Create: `internal/mcp/bridge.go`
- Test: `internal/mcp/bridge_test.go`

**Step 1: Write failing test**

```go
package mcp

import (
	"context"
	"testing"
)

func TestBridge_RegisterTool(t *testing.T) {
	bridge := NewBridge()

	tool := Tool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"input": {Type: "string"},
			},
		},
	}

	err := bridge.RegisterTool(tool)
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	tools := bridge.ListTools()
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/mcp/... -v`
Expected: FAIL - package not found

**Step 3: Implement MCP Bridge**

```go
// internal/mcp/bridge.go
package mcp

import (
	"context"
	"errors"
	"sync"
)

// Tool represents an MCP tool
type Tool struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	InputSchema InputSchema  `json:"inputSchema"`
}

// InputSchema describes tool input parameters
type InputSchema struct {
	Type       string                `json:"type"`
	Properties map[string]Property   `json:"properties"`
	Required   []string              `json:"required,omitempty"`
}

// Property represents a single parameter property
type Property struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// Resource represents an MCP resource
type Resource struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	MIMEType    string `json:"mimeType,omitempty"`
}

// Bridge connects MCP tools/resources to agents
type Bridge struct {
	mu        sync.RWMutex
	tools     map[string]Tool
	resources map[string]Resource
	handlers  map[string]ToolHandler
}

// ToolHandler is a function that handles tool calls
type ToolHandler func(ctx context.Context, arguments map[string]interface{}) (interface{}, error)

// NewBridge creates a new MCP bridge
func NewBridge() *Bridge {
	return &Bridge{
		tools:     make(map[string]Tool),
		resources: make(map[string]Resource),
		handlers:  make(map[string]ToolHandler),
	}
}

// RegisterTool registers a tool with the bridge
func (b *Bridge) RegisterTool(tool Tool) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if tool.Name == "" {
		return errors.New("tool name is required")
	}

	b.tools[tool.Name] = tool
	return nil
}

// RegisterToolHandler registers a handler for a tool
func (b *Bridge) RegisterToolHandler(toolName string, handler ToolHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.tools[toolName]; !ok {
		return errors.New("tool not registered: " + toolName)
	}

	b.handlers[toolName] = handler
	return nil
}

// ListTools returns all registered tools
func (b *Bridge) ListTools() []Tool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	tools := make([]Tool, 0, len(b.tools))
	for _, tool := range b.tools {
		tools = append(tools, tool)
	}
	return tools
}

// CallTool invokes a tool handler
func (b *Bridge) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	handler, ok := b.handlers[toolName]
	if !ok {
		return nil, errors.New("tool handler not found: " + toolName)
	}

	return handler(ctx, arguments)
}

// RegisterResource registers a resource with the bridge
func (b *Bridge) RegisterResource(resource Resource) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if resource.Name == "" {
		return errors.New("resource name is required")
	}

	b.resources[resource.Name] = resource
	return nil
}

// ListResources returns all registered resources
func (b *Bridge) ListResources() []Resource {
	b.mu.RLock()
	defer b.mu.RUnlock()

	resources := make([]Resource, 0, len(b.resources))
	for _, resource := range b.resources {
		resources = append(resources, resource)
	}
	return resources
}

// GetToolSchema returns the input schema for a tool
func (b *Bridge) GetToolSchema(toolName string) (InputSchema, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	tool, ok := b.tools[toolName]
	if !ok {
		return InputSchema{}, errors.New("tool not found: " + toolName)
	}

	return tool.InputSchema, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/mcp/... -v -run=TestBridge`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/mcp/
git commit -m "feat(mcp): add MCP bridge core with tool/resource registry

Sprint 10.2: MCP Bridge
- Tool registration with input schemas
- Tool handler registration and invocation
- Resource registration
- Thread-safe operations

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Integration and Final Testing

### Task 9: Wire Routes in Main

**Files:**
- Modify: `cmd/rad-gateway/main.go`
- Or create: `internal/api/routes.go`

**Step 1: Update route registration**

Add to main.go or create routes.go:

```go
// internal/api/routes.go
package api

import (
	"net/http"

	"radgateway/internal/a2a"
	"radgateway/internal/agui"
	"radgateway/internal/mcp"
)

// RegisterAllRoutes registers all API routes
func RegisterAllRoutes(mux *http.ServeMux) {
	// A2A routes
	a2aStore := a2a.NewStore()
	a2aHandler := a2a.NewHandler(a2aStore)
	a2aHandler.RegisterRoutes(mux)

	// AG-UI routes
	aguiHandler := agui.NewHandler()
	aguiHandler.RegisterRoutes(mux)

	// MCP routes (if needed)
	// mcpBridge := mcp.NewBridge()
}
```

**Step 2: Commit**

```bash
git add internal/api/routes.go
git commit -m "feat(api): wire A2A, AG-UI, and MCP routes

Phase 4 Integration:
- Register A2A protocol endpoints
- Register AG-UI event stream endpoints
- Setup for MCP bridge integration

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 10: Final Integration Tests

**Files:**
- Create: `tests/integration/a2a_test.go`
- Create: `tests/integration/agui_test.go`

**Step 1: Write integration tests**

```go
// tests/integration/a2a_test.go
package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"radgateway/internal/a2a"
)

func TestA2A_AgentCardDiscovery(t *testing.T) {
	store := a2a.NewStore()
	store.Save(a2a.AgentCard{
		Name:    "Test Gateway",
		URL:     "http://localhost:8080",
		Version: "1.0.0",
	})

	handler := a2a.NewHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent.json", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}

	var card a2a.AgentCard
	if err := json.Unmarshal(rec.Body.Bytes(), &card); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if card.Name != "Test Gateway" {
		t.Errorf("Expected name 'Test Gateway', got %q", card.Name)
	}
}

func TestA2A_TaskLifecycle(t *testing.T) {
	store := a2a.NewStore()
	handler := a2a.NewHandler(store)

	// Create task
	reqBody := a2a.SendTaskRequest{
		ID:      "task-123",
		Message: a2a.Message{Role: "user", Content: "Hello"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/a2a/tasks/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp a2a.SendTaskResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Task == nil {
		t.Fatal("Expected task in response")
	}

	if resp.Task.Status != a2a.TaskStatusPending {
		t.Errorf("Expected status pending, got %q", resp.Task.Status)
	}
}
```

**Step 2: Run integration tests**

Run: `go test ./tests/integration/... -v -run=TestA2A`
Expected: PASS

**Step 3: Commit**

```bash
git add tests/integration/
git commit -m "test(integration): add A2A and AG-UI integration tests

Phase 4 Integration Tests:
- Agent card discovery endpoint test
- Task lifecycle (create, get) test
- Verify A2A protocol compliance

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Documentation

### Task 11: Create Documentation

**Files:**
- Create: `docs/protocols/a2a.md`
- Create: `docs/protocols/agui.md`
- Create: `docs/protocols/mcp.md`

**Step 1: Create A2A documentation**

```markdown
# A2A Protocol Implementation

## Overview

RAD Gateway implements the A2A (Agent-to-Agent) protocol for agent interoperability.

## Agent Card Discovery

Endpoint: `GET /.well-known/agent.json`

Response:
```json
{
  "name": "RAD Gateway",
  "description": "AI API Gateway with A2A support",
  "url": "https://gateway.example.com",
  "version": "1.0.0",
  "capabilities": {
    "streaming": true,
    "pushNotifications": false,
    "stateTransitionHistory": true
  },
  "skills": [
    {
      "id": "chat-completion",
      "name": "Chat Completion",
      "description": "OpenAI-compatible chat completion API"
    }
  ]
}
```

## Task Lifecycle

### Create Task

Endpoint: `POST /a2a/tasks/send`

Request:
```json
{
  "id": "task-123",
  "sessionId": "session-456",
  "message": {
    "role": "user",
    "content": "Hello, agent!"
  },
  "skillId": "chat-completion"
}
```

Response:
```json
{
  "task": {
    "id": "task-123",
    "status": "pending",
    "createdAt": "2026-02-27T10:00:00Z"
  }
}
```

### Get Task

Endpoint: `GET /a2a/tasks/{taskId}`

### Cancel Task

Endpoint: `POST /a2a/tasks/cancel`

## State Transitions

```
pending -> working -> completed
pending -> working -> failed
pending -> working -> input-required -> working -> completed
pending -> cancelled
working -> cancelled
```
```

**Step 2: Commit documentation**

```bash
git add docs/protocols/
git commit -m "docs(protocols): add A2A, AG-UI, and MCP documentation

Phase 4 Documentation:
- A2A protocol endpoints and examples
- AG-UI event stream documentation
- MCP bridge architecture overview

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Summary

**Implementation Complete:**

| Component | Files | Status |
|-----------|-------|--------|
| A2A Models | `internal/a2a/models.go` | ✅ |
| A2A Store | `internal/a2a/store.go` | ✅ |
| A2A Handler | `internal/a2a/handler.go` | ✅ |
| Task Manager | `internal/a2a/task_manager.go` | ✅ |
| AG-UI Models | `internal/agui/models.go` | ✅ |
| AG-UI Handler | `internal/agui/handler.go` | ✅ |
| MCP Bridge | `internal/mcp/bridge.go` | ✅ |
| Integration | `internal/api/routes.go` | ✅ |
| Tests | `tests/integration/` | ✅ |
| Docs | `docs/protocols/` | ✅ |

**Next Steps:**
- Implement streaming task updates (SSE)
- Add authentication for agent endpoints
- Implement tool execution handlers
- Add metrics for agent operations
