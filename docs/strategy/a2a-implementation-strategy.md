# A2A Implementation Strategy for Brass Relay

## Executive Summary

### Strategic Recommendation

**Implement full A2A specification compliance as a category-defining differentiator.** While Kong, NGINX, and LiteLLM compete on traditional API gateway features, none offer native Agent-to-Agent protocol support. This is Brass Relay's opportunity to own the emerging agent interoperability layer.

### Key Strategic Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Implementation Scope | Full spec + strategic extensions | Market leadership requires complete compliance plus innovation |
| Architecture Role | Transport + Light Orchestration | Be the "post office" first, "workflow engine" second |
| Initial Target | Agent developers, multi-agent platforms | Early adopters building agent ecosystems |
| Go-to-Market | "The A2A Gateway" positioning | Explicit differentiation from traditional API gateways |

### Success Metrics

- **Technical**: 100% A2A spec compliance score on conformance tests
- **Adoption**: 3+ enterprise pilots within 6 months of release
- **Ecosystem**: Integration partnerships with 2+ major agent frameworks
- **Competitive**: Feature parity with Bee Agent Framework + unique gateway features

---

## 1. A2A Specification Review

### 1.1 Core Endpoints Analysis

The A2A protocol defines a RESTful HTTP-based communication standard for autonomous agents. Brass Relay must implement:

#### Discovery Endpoint

```
GET /.well-known/agent.json
```

**Purpose**: Agent Card publication for capability discovery
**Critical Fields**:
- `name`: Human-readable agent identifier
- `description`: Capability summary for matching algorithms
- `url`: Base endpoint for task submissions
- `capabilities`: Supported task types and streaming modes
- `authentication`: Required auth schemes (OAuth2, API Key, etc.)
- `skills`: Structured capability declarations with input/output schemas

**Strategic Note**: The Agent Card is the "landing page" for agents. Brass Relay should cache and index these for cross-agent discovery.

#### Task Submission Endpoints

```
POST /a2a/tasks/send           # Synchronous request/response
POST /a2a/tasks/sendSubscribe  # Asynchronous with SSE streaming
```

**Request Structure**:
```json
{
  "id": "task-uuid-v4",
  "message": {
    "role": "user",
    "parts": [{"type": "text", "text": "Task description"}]
  },
  "metadata": {"key": "value"}
}
```

**Response Structure** (sync):
```json
{
  "id": "task-uuid-v4",
  "status": "completed",
  "artifacts": [
    {"type": "text", "text": "Result content"}
  ],
  "metadata": {}
}
```

**Strategic Note**: The `sendSubscribe` endpoint using Server-Sent Events (SSE) is critical for long-running agent tasks. Brass Relay's existing streaming infrastructure should be extended to support A2A's event taxonomy.

#### Task Lifecycle Endpoints

```
GET    /a2a/tasks/{taskId}        # Retrieve task status/result
POST   /a2a/tasks/{taskId}/cancel # Cancel ongoing task
```

**Strategic Note**: Task state management requires persistence. Brass Relay should use the existing trace/usage store patterns with A2A-specific task state machines.

### 1.2 Task Lifecycle Deep-Dive

The A2A specification defines a comprehensive state machine for task execution:

```
                    +----------+
                    |  submit  |
                    +----+-----+
                         |
                         v
              +----------+----------+
         +----+      working        +----+
         |    +----------+----------+    |
   cancel|               |               |complete
         |               |               |
         v               v               v
   +-----+-----+   +-----+-----+   +-----+-----+
   |  canceled |   |input-required|  | completed |
   +-----------+   +-----+-----+   +-----------+
                         |
                         | provide input
                         |
                         v
                    +----------+
                    | working  |
                    +----------+
```

**State Definitions**:

| State | Description | Brass Relay Action |
|-------|-------------|-------------------|
| `submitted` | Task received, not yet processing | Validate, authenticate, route to target agent |
| `working` | Agent actively processing | Stream SSE updates, monitor timeout |
| `input-required` | Agent needs clarification | Notify client via SSE, await follow-up |
| `completed` | Task finished successfully | Store final artifacts, emit usage record |
| `canceled` | Task terminated by client | Cleanup resources, notify target agent |
| `failed` | Processing error occurred | Log error details, emit failure trace |

**State Transition Triggers**:
- Client cancel request
- Agent status updates (via webhook or poll)
- Timeout conditions
- Error conditions

### 1.3 Agent Card Format

The Agent Card is the cornerstone of A2A discovery. Brass Relay should support:

**Static Agent Cards** (file-based):
```json
{
  "name": "brass-relay-gateway",
  "description": "AI API Gateway with A2A support",
  "url": "https://gateway.brassrelay.com/a2a",
  "provider": {
    "organization": "Brass Relay",
    "url": "https://brassrelay.com"
  },
  "version": "1.0.0",
  "documentationUrl": "https://docs.brassrelay.com/a2a",
  "capabilities": {
    "streaming": true,
    "pushNotifications": false,
    "stateTransitionHistory": true
  },
  "authentication": {
    "schemes": ["Bearer", "OAuth2"],
    "credentials": "https://gateway.brassrelay.com/auth"
  },
  "defaultInputModes": ["text"],
  "defaultOutputModes": ["text", "file"],
  "skills": [
    {
      "id": "llm-routing",
      "name": "LLM Request Routing",
      "description": "Route LLM requests to optimal providers",
      "tags": ["llm", "routing", "gateway"],
      "examples": ["Route chat completion to best available model"],
      "inputModes": ["text"],
      "outputModes": ["text"]
    },
    {
      "id": "multi-agent-orchestration",
      "name": "Multi-Agent Orchestration",
      "description": "Coordinate tasks across multiple agents",
      "tags": ["orchestration", "multi-agent"],
      "examples": ["Delegate subtasks to specialized agents"],
      "inputModes": ["text"],
      "outputModes": ["text"]
    }
  ]
}
```

**Dynamic Agent Cards** (registry-based):
Brass Relay should maintain a registry of proxied agents, aggregating their capabilities:

```
GET /.well-known/agent.json            # Gateway's own card
GET /a2a/registry                      # List registered agents
GET /a2a/registry/{agentId}/card       # Specific agent card
```

### 1.4 Authentication and Authorization in A2A

A2A allows multiple authentication schemes. Brass Relay should implement:

**Authentication Patterns**:

| Scheme | Use Case | Implementation |
|--------|----------|----------------|
| `Bearer` | API key authentication | Reuse existing `internal/middleware` auth |
| `OAuth2` | Enterprise SSO flows | Extend with OAuth2 handler |
| `None` | Public agent endpoints | Configurable per-route |

**Cross-Agent Authorization**:
When Brass Relay acts as an intermediary between agents:

1. **Inbound Authentication**: Validate caller credentials
2. **Outbound Authentication**: Present gateway credentials to target agent
3. **Delegation Chains**: Propagate original caller identity via JWT claims
4. **Scope Enforcement**: Restrict agent A from accessing agent B's sensitive skills

**Security Model**:
```
Client Agent          Brass Relay          Target Agent
     |                     |                     |
     |--(1) Auth Request--> |                     |
     |<-(2) Token --------- |                     |
     |                     |                     |
     |--(3) Task + Token ->|                     |
     |                     |--(4) Validate ----->|
     |                     |<-(5) OK ------------|
     |                     |--(6) Task + Gateway Token ->
     |                     |                     |
     |<-(7) Stream/Result--|<-(8) Result --------|
```

---

## 2. Implementation Architecture

### 2.1 Package Structure

```
internal/
├── a2a/                      # A2A protocol implementation
│   ├── handler.go            # HTTP handlers for A2A endpoints
│   ├── models.go             # A2A data models (Task, Message, AgentCard)
│   ├── service.go            # Business logic layer
│   ├── store.go              # Task persistence interface
│   ├── store_memory.go       # In-memory implementation
│   ├── store_postgres.go     # PostgreSQL implementation (future)
│   ├── statemachine.go       # Task state transition logic
│   ├── registry.go           # Agent registry and discovery
│   ├── streaming.go          # SSE streaming for sendSubscribe
│   ├── delegation.go         # Cross-agent task delegation
│   └── validation.go         # Request validation utilities
├── a2a/client/               # A2A client for outbound requests
│   ├── client.go             # HTTP client for calling other agents
│   ├── pool.go               # Connection pooling
│   └── retry.go              # Retry logic for agent calls
└── streaming/                # Shared streaming infrastructure (planned)
    ├── sse.go                # SSE writer utilities
    ├── websocket.go          # WebSocket support (future)
    └── events.go             # Common event envelope formats
```

### 2.2 Data Models

#### Task Model

```go
// internal/a2a/models.go

package a2a

import (
    "time"
    "encoding/json"
)

// TaskState represents the lifecycle state of an A2A task
type TaskState string

const (
    TaskStateSubmitted      TaskState = "submitted"
    TaskStateWorking        TaskState = "working"
    TaskStateInputRequired  TaskState = "input-required"
    TaskStateCompleted      TaskState = "completed"
    TaskStateCanceled       TaskState = "canceled"
    TaskStateFailed         TaskState = "failed"
)

// Task represents an A2A task entity
type Task struct {
    ID          string          `json:"id" db:"id"`
    ParentID    *string         `json:"parentId,omitempty" db:"parent_id"` // For subtasks
    SessionID   string          `json:"sessionId" db:"session_id"`
    Status      TaskState       `json:"status" db:"status"`

    // Request details
    Message     Message         `json:"message" db:"message_json"`
    Metadata    json.RawMessage `json:"metadata,omitempty" db:"metadata_json"`

    // Response artifacts
    Artifacts   []Artifact      `json:"artifacts,omitempty" db:"artifacts_json"`

    // Routing information
    SourceAgent string          `json:"sourceAgent" db:"source_agent"`
    TargetAgent string          `json:"targetAgent" db:"target_agent"`

    // Delegation chain for multi-hop tasks
    DelegationChain []string    `json:"delegationChain,omitempty" db:"delegation_chain"`

    // Timestamps
    CreatedAt   time.Time       `json:"createdAt" db:"created_at"`
    UpdatedAt   time.Time       `json:"updatedAt" db:"updated_at"`
    CompletedAt *time.Time      `json:"completedAt,omitempty" db:"completed_at"`

    // Expiration for cleanup
    ExpiresAt   time.Time       `json:"expiresAt" db:"expires_at"`

    // Error details (for failed state)
    Error       *TaskError      `json:"error,omitempty" db:"error_json"`
}

// Message represents an A2A message
type Message struct {
    Role  string `json:"role"`  // "user", "agent", "system"
    Parts []Part `json:"parts"`
}

// Part represents a content part within a message
type Part struct {
    Type string `json:"type"` // "text", "file", "data"

    // For type="text"
    Text string `json:"text,omitempty"`

    // For type="file"
    File *FilePart `json:"file,omitempty"`

    // For type="data"
    Data map[string]interface{} `json:"data,omitempty"`
}

// FilePart represents file content
type FilePart struct {
    Name     string `json:"name"`
    MimeType string `json:"mimeType"`
    Bytes    []byte `json:"bytes,omitempty"`
    URI      string `json:"uri,omitempty"` // For external references
}

// Artifact represents task output
type Artifact struct {
    Type     string          `json:"type"` // "text", "file", "data"
    Name     string          `json:"name,omitempty"`
    Parts    []Part          `json:"parts"`
    Metadata json.RawMessage `json:"metadata,omitempty"`
}

// TaskError represents error details
type TaskError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

// IsTerminal returns true if the task has reached a terminal state
func (t *Task) IsTerminal() bool {
    return t.Status == TaskStateCompleted ||
           t.Status == TaskStateCanceled ||
           t.Status == TaskStateFailed
}

// CanTransitionTo checks if a state transition is valid
func (t *Task) CanTransitionTo(newState TaskState) bool {
    transitions := map[TaskState][]TaskState{
        TaskStateSubmitted:     {TaskStateWorking, TaskStateCanceled, TaskStateFailed},
        TaskStateWorking:       {TaskStateInputRequired, TaskStateCompleted, TaskStateCanceled, TaskStateFailed},
        TaskStateInputRequired: {TaskStateWorking, TaskStateCanceled},
        TaskStateCompleted:     {}, // Terminal
        TaskStateCanceled:      {}, // Terminal
        TaskStateFailed:        {TaskStateWorking}, // Retry possible
    }

    valid, ok := transitions[t.Status]
    if !ok {
        return false
    }

    for _, s := range valid {
        if s == newState {
            return true
        }
    }
    return false
}
```

#### Agent Model

```go
// Agent represents a registered A2A agent
type Agent struct {
    ID            string          `json:"id" db:"id"`
    Name          string          `json:"name" db:"name"`
    Description   string          `json:"description" db:"description"`
    EndpointURL   string          `json:"endpointUrl" db:"endpoint_url"`

    // Parsed Agent Card
    Card          AgentCard       `json:"card" db:"card_json"`

    // Capabilities extracted from card
    Capabilities  Capabilities    `json:"capabilities" db:"capabilities"`
    Skills        []Skill         `json:"skills" db:"skills_json"`

    // Authentication config
    AuthScheme    string          `json:"authScheme" db:"auth_scheme"`
    AuthConfig    json.RawMessage `json:"authConfig,omitempty" db:"auth_config"`

    // Status
    IsActive      bool            `json:"isActive" db:"is_active"`
    LastHealthCheck *time.Time    `json:"lastHealthCheck,omitempty" db:"last_health_check"`

    // Metadata
    CreatedAt     time.Time       `json:"createdAt" db:"created_at"`
    UpdatedAt     time.Time       `json:"updatedAt" db:"updated_at"`
}

// AgentCard represents the A2A Agent Card format
type AgentCard struct {
    Name                 string         `json:"name"`
    Description          string         `json:"description"`
    URL                  string         `json:"url"`
    Provider             AgentProvider  `json:"provider,omitempty"`
    Version              string         `json:"version"`
    DocumentationURL     string         `json:"documentationUrl,omitempty"`
    Capabilities         Capabilities   `json:"capabilities"`
    Authentication       AuthConfig     `json:"authentication"`
    DefaultInputModes    []string       `json:"defaultInputModes"`
    DefaultOutputModes   []string       `json:"defaultOutputModes"`
    Skills               []Skill        `json:"skills,omitempty"`
}

// Capabilities represents agent capabilities
type Capabilities struct {
    Streaming              bool `json:"streaming"`
    PushNotifications      bool `json:"pushNotifications"`
    StateTransitionHistory bool `json:"stateTransitionHistory"`
}

// Skill represents an agent capability
type Skill struct {
    ID           string   `json:"id"`
    Name         string   `json:"name"`
    Description  string   `json:"description"`
    Tags         []string `json:"tags,omitempty"`
    Examples     []string `json:"examples,omitempty"`
    InputModes   []string `json:"inputModes"`
    OutputModes  []string `json:"outputModes"`

    // Extended for Brass Relay
    Parameters   json.RawMessage `json:"parameters,omitempty"` // JSON Schema
}
```

### 2.3 State Machine Implementation

```go
// internal/a2a/statemachine.go

package a2a

import (
    "context"
    "fmt"
    "time"

    "radgateway/internal/trace"
)

// StateMachine manages task state transitions
type StateMachine struct {
    store  TaskStore
    tracer *trace.Store
    hooks  []TransitionHook
}

// TransitionHook is called on state changes
type TransitionHook func(ctx context.Context, task *Task, oldState, newState TaskState) error

// NewStateMachine creates a state machine
func NewStateMachine(store TaskStore, tracer *trace.Store) *StateMachine {
    return &StateMachine{
        store:  store,
        tracer: tracer,
        hooks:  make([]TransitionHook, 0),
    }
}

// RegisterHook adds a transition hook
func (sm *StateMachine) RegisterHook(hook TransitionHook) {
    sm.hooks = append(sm.hooks, hook)
}

// Transition attempts to move a task to a new state
func (sm *StateMachine) Transition(ctx context.Context, taskID string, newState TaskState) (*Task, error) {
    // Fetch current task
    task, err := sm.store.Get(ctx, taskID)
    if err != nil {
        return nil, fmt.Errorf("failed to get task: %w", err)
    }

    oldState := task.Status

    // Validate transition
    if !task.CanTransitionTo(newState) {
        return nil, fmt.Errorf("invalid state transition from %s to %s", oldState, newState)
    }

    // Apply transition
    task.Status = newState
    task.UpdatedAt = time.Now()

    if task.IsTerminal() {
        now := time.Now()
        task.CompletedAt = &now
    }

    // Persist
    if err := sm.store.Update(ctx, task); err != nil {
        return nil, fmt.Errorf("failed to update task: %w", err)
    }

    // Trace
    if sm.tracer != nil {
        sm.tracer.Add(trace.Event{
            Timestamp: time.Now(),
            TraceID:   task.SessionID,
            RequestID: task.ID,
            Message:   fmt.Sprintf("Task state transition: %s -> %s", oldState, newState),
        })
    }

    // Execute hooks
    for _, hook := range sm.hooks {
        if err := hook(ctx, task, oldState, newState); err != nil {
            // Log but don't fail the transition
            // TODO: structured logging
        }
    }

    return task, nil
}

// AutoExpire transitions tasks past their expiration
func (sm *StateMachine) AutoExpire(ctx context.Context) error {
    expired, err := sm.store.ListExpired(ctx, time.Now())
    if err != nil {
        return err
    }

    for _, task := range expired {
        if !task.IsTerminal() {
            task.Error = &TaskError{
                Code:    "TIMEOUT",
                Message: "Task expired before completion",
            }
            sm.Transition(ctx, task.ID, TaskStateFailed)
        }
    }

    return nil
}
```

### 2.4 Integration with Existing Routing

Brass Relay's existing routing layer should be extended to support A2A:

```go
// internal/a2a/delegation.go

package a2a

import (
    "context"
    "fmt"

    "radgateway/internal/provider"
)

// Delegator routes A2A tasks to appropriate agents
type Delegator struct {
    registry     *Registry
    client       *Client
    router       *provider.Router  // Existing router
    stateMachine *StateMachine
}

// Delegate routes a task to a target agent
func (d *Delegator) Delegate(ctx context.Context, task *Task) error {
    // 1. Resolve target agent
    agent, err := d.registry.Resolve(ctx, task.TargetAgent)
    if err != nil {
        return fmt.Errorf("failed to resolve agent: %w", err)
    }

    // 2. Check agent health
    if !agent.IsActive {
        return fmt.Errorf("agent %s is not active", agent.Name)
    }

    // 3. Transition to working
    if _, err := d.stateMachine.Transition(ctx, task.ID, TaskStateWorking); err != nil {
        return err
    }

    // 4. Delegate based on agent type
    if d.isGatewayAgent(agent) {
        // Route through existing provider router for LLM calls
        return d.delegateToLLM(ctx, task, agent)
    }

    // Standard A2A delegation
    return d.delegateToAgent(ctx, task, agent)
}

func (d *Delegator) delegateToLLM(ctx context.Context, task *Task, agent *Agent) error {
    // Convert A2A task to provider request
    req := provider.Request{
        Model:   d.extractModelFromTask(task),
        Prompt:  d.extractPromptFromTask(task),
        // ... other fields
    }

    // Use existing router
    result, err := d.router.Dispatch(ctx, req)
    if err != nil {
        d.stateMachine.Transition(ctx, task.ID, TaskStateFailed)
        return err
    }

    // Convert result to A2A artifact
    task.Artifacts = []Artifact{{
        Type:  "text",
        Parts: []Part{{Type: "text", Text: result.Output}},
    }}

    d.stateMachine.Transition(ctx, task.ID, TaskStateCompleted)
    return nil
}

func (d *Delegator) delegateToAgent(ctx context.Context, task *Task, agent *Agent) error {
    // Use A2A client for cross-agent communication
    return d.client.SendTask(ctx, agent.EndpointURL, task)
}
```

### 2.5 HTTP Handler Design

```go
// internal/a2a/handler.go

package a2a

import (
    "encoding/json"
    "net/http"
    "strings"

    "radgateway/internal/middleware"
)

// Handler implements A2A HTTP endpoints
type Handler struct {
    service     *Service
    stateMachine *StateMachine
}

func NewHandler(service *Service, sm *StateMachine) *Handler {
    return &Handler{service: service, stateMachine: sm}
}

func (h *Handler) Register(mux *http.ServeMux) {
    // Discovery
    mux.HandleFunc("/.well-known/agent.json", h.agentCard)

    // Task endpoints
    mux.HandleFunc("/a2a/tasks/send", h.sendTask)
    mux.HandleFunc("/a2a/tasks/sendSubscribe", h.sendSubscribe)
    mux.HandleFunc("/a2a/tasks/", h.taskOperations) // /{taskId}, /{taskId}/cancel
}

func (h *Handler) agentCard(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    card := h.service.GetAgentCard()
    writeJSON(w, http.StatusOK, card)
}

func (h *Handler) sendTask(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req SendTaskRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Failed to parse request")
        return
    }

    ctx := r.Context()
    apiKeyName := middleware.GetAPIKeyName(ctx)

    // Create and delegate task
    task, err := h.service.CreateTask(ctx, req, apiKeyName)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "TASK_CREATION_FAILED", err.Error())
        return
    }

    // Execute synchronously
    if err := h.service.ExecuteTask(ctx, task); err != nil {
        writeError(w, http.StatusBadGateway, "TASK_EXECUTION_FAILED", err.Error())
        return
    }

    // Return final state
    response := SendTaskResponse{
        ID:        task.ID,
        Status:    task.Status,
        Artifacts: task.Artifacts,
        Metadata:  task.Metadata,
    }

    writeJSON(w, http.StatusOK, response)
}

func (h *Handler) sendSubscribe(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req SendTaskRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Failed to parse request")
        return
    }

    ctx := r.Context()
    apiKeyName := middleware.GetAPIKeyName(ctx)

    // Create task
    task, err := h.service.CreateTask(ctx, req, apiKeyName)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "TASK_CREATION_FAILED", err.Error())
        return
    }

    // Set up SSE
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        writeError(w, http.StatusInternalServerError, "STREAMING_UNSUPPORTED", "Server does not support streaming")
        return
    }

    // Send initial state
    h.sendSSEEvent(w, flusher, "task_status", task)

    // Subscribe to task updates
    updates := h.service.SubscribeToTask(ctx, task.ID)

    for {
        select {
        case update := <-updates:
            h.sendSSEEvent(w, flusher, "task_status", update)

            if update.IsTerminal() {
                return
            }

        case <-ctx.Done():
            return
        }
    }
}

func (h *Handler) taskOperations(w http.ResponseWriter, r *http.Request) {
    path := strings.TrimPrefix(r.URL.Path, "/a2a/tasks/")
    parts := strings.Split(path, "/")

    if len(parts) == 0 || parts[0] == "" {
        http.Error(w, "Task ID required", http.StatusBadRequest)
        return
    }

    taskID := parts[0]

    // Check for cancel sub-path
    if len(parts) == 2 && parts[1] == "cancel" {
        if r.Method != http.MethodPost {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }
        h.cancelTask(w, r, taskID)
        return
    }

    // Default: get task
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    h.getTask(w, r, taskID)
}

func (h *Handler) getTask(w http.ResponseWriter, r *http.Request, taskID string) {
    task, err := h.service.GetTask(r.Context(), taskID)
    if err != nil {
        writeError(w, http.StatusNotFound, "TASK_NOT_FOUND", "Task not found")
        return
    }

    writeJSON(w, http.StatusOK, task)
}

func (h *Handler) cancelTask(w http.ResponseWriter, r *http.Request, taskID string) {
    task, err := h.stateMachine.Transition(r.Context(), taskID, TaskStateCanceled)
    if err != nil {
        writeError(w, http.StatusBadRequest, "CANCEL_FAILED", err.Error())
        return
    }

    writeJSON(w, http.StatusOK, task)
}

func (h *Handler) sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
    payload, _ := json.Marshal(data)
    fmt.Fprintf(w, "event: %s\n", eventType)
    fmt.Fprintf(w, "data: %s\n\n", payload)
    flusher.Flush()
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, errCode, message string) {
    writeJSON(w, code, map[string]interface{}{
        "error": map[string]string{
            "code":    errCode,
            "message": message,
        },
    })
}
```

---

## 3. Feature Prioritization

### 3.1 P0: Must Have for MVP

| Feature | Description | Sprint |
|---------|-------------|--------|
| Agent Card Endpoint | `GET /.well-known/agent.json` serving gateway capabilities | 1 |
| Task Creation | `POST /a2a/tasks/send` synchronous task handling | 1 |
| Task State Machine | Core state transitions (submitted -> working -> completed/failed) | 1 |
| Task Retrieval | `GET /a2a/tasks/{taskId}` for status checking | 1 |
| Task Cancellation | `POST /a2a/tasks/{taskId}/cancel` | 2 |
| SSE Streaming | `POST /a2a/tasks/sendSubscribe` for real-time updates | 2 |
| In-Memory Store | Task persistence with TTL | 1-2 |
| Basic Delegation | Route A2A tasks to registered agents | 2 |
| Authentication | Bearer token validation on A2A endpoints | 1 |
| Error Handling | A2A-compliant error responses | 1 |

**MVP Definition of Done**:
- All P0 endpoints respond correctly to A2A conformance tests
- Tasks can be created, executed, and completed end-to-end
- SSE streaming delivers state updates in real-time
- Documentation with working examples

### 3.2 P1: Important for v1.1

| Feature | Description | Business Value |
|---------|-------------|----------------|
| Agent Registry | CRUD for agent registration and discovery | Multi-agent orchestration |
| Health Checks | Automatic agent health monitoring | Reliability |
| Skill-Based Routing | Route tasks by skill matching vs agent ID | Smart delegation |
| Subtask Delegation | Create child tasks for complex workflows | Workflow composition |
| Task Retry Logic | Automatic retry with backoff | Resilience |
| PostgreSQL Store | Persistent task storage | Production readiness |
| Push Notifications | Webhook support for state changes | Real-time integrations |
| Input-Required Flow | Interactive task clarification | Complex workflows |
| Batch Operations | Create multiple tasks atomically | Efficiency |
| Rate Limiting | Per-agent task rate controls | Resource protection |

### 3.3 P2: Future Enhancements

| Feature | Description | Strategic Value |
|---------|-------------|-----------------|
| Workflow Composition | Visual/scripted multi-agent workflows | Platform play |
| Agent Marketplace | Directory of discoverable agents | Ecosystem building |
| Semantic Routing | LLM-based task-to-agent matching | Intelligence |
| Cross-Protocol Bridge | A2A <-> MCP <-> AG-UI translation | Protocol leadership |
| Federated Discovery | Cross-gateway agent discovery | Scale |
| Policy Engine | Fine-grained access control between agents | Enterprise |
| Cost Attribution | Track and bill by agent/task | Monetization |
| Analytics Dashboard | Task volume, latency, success rates | Operations |
| A2A Extensions | Custom protocol extensions | Innovation |
| Mobile SDKs | iOS/Android A2A clients | Reach |

---

## 4. Competitive Analysis

### 4.1 Current A2A Ecosystem

#### Bee Agent Framework (IBM)

**Status**: Most mature open-source A2A implementation
**Strengths**:
- Full A2A spec compliance
- Python SDK with good ergonomics
- Active community and documentation
- Example agents for common use cases

**Weaknesses**:
- Python-only (no Go SDK)
- Not a gateway (framework for building agents)
- No multi-agent orchestration layer
- Limited production scaling features

**Brass Relay Differentiation**:
- Go-native implementation (performance)
- Gateway architecture (protocol agnostic)
- Existing routing/failover infrastructure
- Enterprise operations features

#### Google A2A Reference Implementation

**Status**: Spec authors, reference Python SDK
**Strengths**:
- Canonical implementation
- Direct access to spec evolution
- Google ecosystem integration potential

**Weaknesses**:
- Reference only, not production-hardened
- Limited to Google's use cases
- No gateway features

**Brass Relay Differentiation**:
- Production gateway features
- Multi-provider LLM support
- Operations and observability

#### Emerging Competitors

| Competitor | A2A Support | Gateway Features | Threat Level |
|------------|-------------|------------------|--------------|
| Kong | None | Full | Low (no A2A roadmap) |
| NGINX | None | Full | Low (no A2A roadmap) |
| LiteLLM | None | Partial | Medium (could add A2A) |
| Traefik | None | Full | Low (no A2A roadmap) |
| Tyk | None | Full | Low (no A2A roadmap) |

### 4.2 Competitive Moat Analysis

**Features Competitors Cannot Easily Copy**:

1. **Protocol Integration Depth**
   - A2A + AG-UI + MCP in single gateway
   - Cross-protocol translation layer
   - Unified observability across protocols

2. **Provider Orchestration + A2A**
   - Route A2A tasks to optimal LLM providers
   - Failover between agent implementations
   - Cost optimization across agent + LLM calls

3. **Agent Registry with Discovery**
   - Indexed Agent Card search
   - Skill-based agent matching
   - Cross-organization agent federation

4. **Operations at Scale**
   - Task-level metrics and tracing
   - Per-agent rate limiting and quotas
   - Production-hardened reliability features

**Defensible Positioning**:

```
Traditional API Gateways          Brass Relay              Agent Frameworks
       |                              |                           |
       |  +------------------------+  |                           |
       |  |  A2A Transport Layer   |  |                           |
       |  +------------------------+  |                           |
       |                              |  +---------------------+  |
       |                              |  |  Agent Orchestration |  |
       |                              |  +---------------------+  |
       v                              v                           v
   Kong/NGINX                   BRASS RELAY              Bee Agent/
                              (Sweet Spot)              LangChain
```

### 4.3 Market Positioning

**Primary Positioning Statement**:

> "Brass Relay is the first AI-native API gateway with built-in Agent-to-Agent protocol support. While traditional gateways handle request routing, Brass Relay enables agent collaboration."

**Key Messages**:

1. **For AI Teams**: "Deploy agents that can discover and delegate to other agents—without building the plumbing."

2. **For Platform Teams**: "One gateway for all your AI traffic: OpenAI APIs, Anthropic calls, and agent-to-agent communication."

3. **For Enterprises**: "Production-grade agent interoperability with the security, observability, and scale you need."

---

## 5. Innovation Opportunities

### 5.1 Extensions to A2A Spec

While maintaining full spec compliance, Brass Relay can offer **optional extensions**:

#### 5.1.1 Batch Task Submission

```json
// POST /a2a/tasks/sendBatch
{
  "tasks": [
    {"id": "task-1", "message": {...}},
    {"id": "task-2", "message": {...}}
  ],
  "executionMode": "parallel", // or "sequential"
  "aggregationStrategy": "merge" // or "individual"
}
```

**Use Case**: Submit 100 tasks at once for map-reduce style processing.

#### 5.1.2 Task Templates

```json
// POST /a2a/tasks/template
{
  "templateId": "customer-onboarding",
  "variables": {
    "customerName": "Acme Corp",
    "industry": "Manufacturing"
  }
}
```

**Use Case**: Predefined workflows with variable substitution.

#### 5.1.3 Skill Query Endpoint

```
GET /a2a/skills?capability=sentiment-analysis&inputMode=text
```

```json
{
  "matches": [
    {
      "agentId": "sentiment-agent-v2",
      "skillId": "analyze-sentiment",
      "confidence": 0.95,
      "estimatedLatency": "500ms"
    }
  ]
}
```

**Use Case**: Dynamic agent selection based on capability matching.

### 5.2 Unique Brass Relay Features

#### 5.2.1 LLM-Augmented Agent Selection

Use an LLM to route tasks to the most appropriate agent:

```go
// Pseudo-code
func (d *Delegator) SmartRoute(ctx context.Context, task *Task) (*Agent, error) {
    // Get all available agents
    agents := d.registry.ListActive()

    // Build routing prompt
    prompt := buildRoutingPrompt(task, agents)

    // Ask LLM to select best agent
    result := d.llm.Complete(ctx, prompt)

    // Parse and validate selection
    selected := parseAgentSelection(result)

    return d.registry.Get(selected.AgentID)
}
```

**Benefit**: Automatically route "Analyze customer feedback" to the sentiment agent without explicit agent ID.

#### 5.2.2 Agent Performance Scoring

Track and score agent performance:

```json
// GET /a2a/agents/{agentId}/metrics
{
  "successRate": 0.987,
  "avgLatency": "450ms",
  "p95Latency": "890ms",
  "costPerTask": 0.023,
  "qualityScore": 4.7,
  "last24Hours": {
    "tasksCompleted": 1543,
    "tasksFailed": 12,
    "avgTokensUsed": 2450
  }
}
```

**Benefit**: Data-driven agent selection and optimization.

#### 5.2.3 Workflow Composition DSL

Define multi-agent workflows:

```yaml
# workflows/customer-support.yml
name: Customer Support Pipeline
version: 1.0.0

steps:
  - id: classify
    agent: intent-classifier
    input: "{{.originalMessage}}"

  - id: route
    condition: "{{.classify.intent}}"
    branches:
      billing:
        - agent: billing-agent
          input: "{{.originalMessage}}"
      technical:
        - agent: troubleshooting-agent
          input: "{{.originalMessage}}"
        - agent: resolution-validator
          input: "{{.troubleshooting.solution}}"

  - id: summarize
    agent: summary-agent
    input: "{{.route.result}}"
    output: final_response
```

**Benefit**: Complex workflows without code changes.

### 5.3 Integration with Existing Gateway Features

| Gateway Feature | A2A Integration | Value |
|-----------------|-----------------|-------|
| Provider Routing | Route A2A subtasks to best LLM | Cost optimization |
| Retry/Failover | Retry failed agent calls | Reliability |
| Usage Tracking | Track per-task costs | Cost visibility |
| Rate Limiting | Rate limit per agent | Resource protection |
| Circuit Breaker | Disable failing agents | Resilience |
| Authentication | Agent-to-agent auth | Security |
| Tracing | End-to-end task tracing | Debuggability |
| Quotas | Task quotas per agent | Governance |

---

## 6. Implementation Roadmap

### 6.1 Sprint Breakdown

#### Sprint 1: Foundation (2 weeks)

**Goals**:
- A2A package structure
- Data models (Task, Agent, Message)
- In-memory task store
- Basic state machine
- Agent Card endpoint

**Deliverables**:
```
internal/a2a/
├── models.go              # Task, Message, Part, Artifact
├── store.go               # TaskStore interface
├── store_memory.go        # InMemoryStore implementation
├── statemachine.go        # State transitions
├── handler.go             # /.well-known/agent.json
└── service.go             # Business logic
```

**Acceptance Criteria**:
- `GET /.well-known/agent.json` returns valid Agent Card
- Tasks can be created and retrieved
- State transitions work correctly

#### Sprint 2: Core Protocol (2 weeks)

**Goals**:
- Task submission endpoints
- SSE streaming for sendSubscribe
- Task cancellation
- Basic error handling
- Authentication integration

**Deliverables**:
- `POST /a2a/tasks/send`
- `POST /a2a/tasks/sendSubscribe` (SSE)
- `GET /a2a/tasks/{taskId}`
- `POST /a2a/tasks/{taskId}/cancel`

**Acceptance Criteria**:
- Synchronous tasks complete end-to-end
- SSE streams state updates correctly
- Authentication enforced on all endpoints

#### Sprint 3: Agent Registry (2 weeks)

**Goals**:
- Agent registration API
- Agent health checks
- Basic delegation to registered agents
- A2A client for outbound calls

**Deliverables**:
```
internal/a2a/
├── registry.go            # Agent registry
├── client/
│   ├── client.go          # A2A HTTP client
│   ├── pool.go            # Connection pooling
│   └── retry.go           # Retry logic
```

**Acceptance Criteria**:
- Agents can be registered via API
- Gateway delegates tasks to registered agents
- Health checks mark unhealthy agents

#### Sprint 4: Integration (2 weeks)

**Goals**:
- Integrate A2A with existing routing layer
- Task-to-LLM routing for gateway-resident capabilities
- Usage tracking for A2A tasks
- Tracing integration

**Deliverables**:
- `internal/a2a/delegation.go`
- Usage records for A2A tasks
- Trace events for task lifecycle

**Acceptance Criteria**:
- A2A tasks can trigger LLM calls through existing providers
- Usage tracking captures A2A task costs
- Full request tracing works

#### Sprint 5: Polish (2 weeks)

**Goals**:
- Comprehensive tests
- Documentation
- Conformance test suite
- Performance optimization

**Deliverables**:
- `internal/a2a/*_test.go` (>80% coverage)
- A2A API documentation
- Conformance test results
- Benchmarks

### 6.2 Dependencies

| Dependency | Status | Blocker? |
|------------|--------|----------|
| Core gateway routing | Exists | No |
| Provider adapters | Partial | No (mock first) |
| Authentication middleware | Exists | No |
| Usage/trace stores | Exists | No |
| Streaming infrastructure | Planned | Partial (can use native SSE) |
| PostgreSQL persistence | Future | No (in-memory first) |

### 6.3 Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| A2A spec changes | High | Design for extensibility; stay active in community |
| Low adoption | Medium | Position as differentiator; offer migration path |
| Performance issues | Medium | Benchmark early; optimize hot paths |
| Security vulnerabilities | High | Security review; fuzz testing; audit logging |
| Integration complexity | Low | Incremental delivery; strong test coverage |

---

## 7. Technical Risks

### 7.1 Specification Instability

**Risk**: A2A is a relatively new protocol and may undergo breaking changes.

**Mitigation Strategy**:
1. **Versioning**: Implement `X-A2A-Version` header support
2. **Abstraction Layer**: Isolate spec-specific code behind interfaces
3. **Community Engagement**: Active participation in A2A working group
4. **Changelog Monitoring**: Automated tracking of spec repository changes

```go
// Version negotiation
func (h *Handler) negotiateVersion(r *http.Request) string {
    requested := r.Header.Get("X-A2A-Version")
    if requested == "" {
        return "1.0" // Default
    }

    // Return closest supported version
    return h.supportedVersions.FindClosest(requested)
}
```

### 7.2 Adoption Challenges

**Risk**: A2A ecosystem may not mature as expected; competing standards emerge.

**Mitigation Strategy**:
1. **Multi-Protocol Support**: Design for A2A + MCP + AG-UI coexistence
2. **Migration Tools**: Build converters between protocol formats
3. **Partnership Strategy**: Early integrations with agent framework vendors
4. **Fallback Mechanisms**: Support non-A2A agents via adapters

### 7.3 Integration Complexity

**Risk**: Complex interactions between A2A, existing routing, and provider adapters.

**Mitigation Strategy**:
1. **Clear Boundaries**: Strict interface contracts between layers
2. **Feature Flags**: A2A features can be disabled if issues arise
3. **Comprehensive Testing**: Contract tests between components
4. **Rollback Plan**: Versioned deployments with quick rollback capability

### 7.4 Security Concerns

**Risk**: Cross-agent delegation introduces new attack vectors.

**Mitigation Strategy**:
1. **Zero Trust**: Authenticate every cross-agent request
2. **Delegation Chains**: Limit chain length; validate at each hop
3. **Scope Enforcement**: Strict skill-level permissions
4. **Audit Logging**: Complete request/response logging for forensics
5. **Rate Limiting**: Prevent agent-based DDoS

---

## 8. Go-to-Market Recommendations

### 8.1 Positioning Against Emerging Standards

| Standard | Positioning |
|----------|-------------|
| A2A | "Native A2A support—no adapters needed" |
| MCP | "A2A for agents, MCP for tools—both supported" |
| AG-UI | "Full-stack: A2A for agents, AG-UI for users" |
| ACP | "ACP merged into A2A—we followed the standard" |
| ANP | "Watching ANP; will adopt when mature" |

### 8.2 Partnership Strategy

**Tier 1 Partners** (Immediate outreach):
- Bee Agent Framework team
- LangChain (A2A integration)
- AutoGen team (Microsoft)

**Tier 2 Partners** (Post-MVP):
- Vector DB vendors (Pinecone, Weaviate)
- Observability platforms (Datadog, Honeycomb)
- Cloud providers (AWS, GCP, Azure marketplace)

### 8.3 Community Engagement

1. **Open Source**: A2A implementation as standalone library
2. **Documentation**: Best practices for A2A at scale
3. **Webinars**: "Building Multi-Agent Systems with A2A"
4. **Conformance Suite**: Open source test suite for A2A implementations

### 8.4 Pricing Strategy

**Open Source** (rad-gateway):
- Full A2A protocol support
- Community support
- Basic agent registry

**Enterprise** (Brass Relay Cloud):
- Advanced orchestration
- Multi-region agent federation
- Performance analytics
- SLA guarantees
- Priority support

---

## 9. Success Metrics and KPIs

### 9.1 Technical Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Spec Compliance | 100% | Conformance test suite |
| Task Latency (p50) | <100ms | Without LLM calls |
| Task Latency (p99) | <500ms | Without LLM calls |
| SSE Stream Reliability | 99.9% | Uptime monitoring |
| State Transition Accuracy | 100% | Test coverage |

### 9.2 Adoption Metrics

| Metric | Target (6mo) | Target (12mo) |
|--------|--------------|---------------|
| Registered Agents | 100 | 1000 |
| Tasks Processed | 10K/day | 100K/day |
| Enterprise Pilots | 3 | 10 |
| GitHub Stars | 500 | 2000 |
| Community Contributors | 10 | 50 |

### 9.3 Business Metrics

| Metric | Target |
|--------|--------|
| A2A as % of Sales Conversations | 50% |
| Competitive Win Rate (vs Kong) | 60% |
| Time-to-Value for A2A Setup | <1 hour |

---

## 10. Conclusion

### Strategic Summary

A2A implementation is **Brass Relay's most important strategic initiative**. While competitors focus on traditional API gateway features, A2A positions Brass Relay at the center of the emerging agent economy.

**Key Decisions**:
1. **Full spec compliance** with strategic extensions
2. **Transport + light orchestration** architecture
3. **8-week implementation** across 5 sprints
4. **"The A2A Gateway"** market positioning

**Competitive Advantages**:
- First-mover in A2A-native gateway space
- Integration with existing routing/failover infrastructure
- Multi-protocol support (A2A + AG-UI + MCP)
- Production-grade operations features

**Next Steps**:
1. Architecture review and approval
2. Sprint 1 kickoff
3. Begin community engagement
4. Draft partnership outreach

### Call to Action

**For Engineering**: Begin Sprint 1 implementation immediately. The foundation (models, store, state machine) is low-risk and enables rapid iteration.

**For Product**: Finalize P0/P1 feature boundaries. Prepare conformance test suite.

**For Marketing**: Develop "A2A Gateway" messaging. Identify launch partners.

**For Leadership**: Approve 8-week roadmap. Allocate resources for community engagement.

---

*Document Version*: 1.0
*Last Updated*: 2026-02-16
*Owner*: Architecture Team
*Review Cycle*: Bi-weekly during implementation
