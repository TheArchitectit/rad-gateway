# A2A Protocol Compliance Implementation Plan

## Overview

**Date**: 2026-02-20
**Status**: Draft - Awaiting Review
**Priority**: Critical (Sprint Goal)
**Estimated Effort**: 3-4 days
**Complexity**: High (7/10)
**Risk**: Medium

This document provides the complete implementation plan for bringing RAD Gateway's A2A (Agent-to-Agent) protocol support into full compliance with the official Google A2A specification.

---

## 1. Gap Analysis Summary

### 1.1 Current State

Our A2A implementation provides basic task management but deviates from the spec in several critical areas:

| Component | Status | Notes |
|-----------|--------|-------|
| **Task Model** | ⚠️ Partial | Missing `artifacts` array, `status` only has state, no message history |
| **Streaming** | ✅ Implemented | SSE streaming works but format needs verification |
| **Push Notifications** | ❌ Missing | No webhook support for task updates |
| **Authentication** | ⚠️ Basic | Uses API keys, no A2A-specific auth flows |
| **Agent Card** | ✅ Implemented | But missing some optional capabilities |
| **Message Format** | ⚠️ Partial | Parts structure correct, but some types missing |

### 1.2 Critical Gaps

1. **Task Model Non-Compliance** (`internal/a2a/task.go:15-50`)
   - Current struct lacks `artifacts` field
   - No support for task metadata
   - Missing `status.history` for message tracking

2. **Missing Push Notification Support**
   - No webhook URL storage
   - No webhook delivery mechanism
   - No retry logic for failed webhooks

3. **Incomplete Message Part Types**
   - Missing `FilePart` and `DataPart` implementations
   - No streaming part support

4. **Agent Card Limitations**
   - Missing `authentication` schemes detail
   - No `capabilities.pushNotifications` flag

---

## 2. Detailed Implementation Tasks

### Task 2.1: Update Task Model (4 hours)

**File**: `internal/a2a/task.go`

**Current State**:
```go
type Task struct {
    ID        string    `json:"id"`
    SessionID string    `json:"sessionId"`
    Status    TaskStatus `json:"status"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}

type TaskStatus struct {
    State     string `json:"state"` // pending, working, completed, failed
    Message   *Message `json:"message,omitempty"`
}
```

**Target State**:
```go
// Task represents an A2A task as per the specification
type Task struct {
    ID          string       `json:"id"`
    SessionID   string       `json:"sessionId"`
    Status      TaskStatus   `json:"status"`
    Artifacts   []Artifact   `json:"artifacts,omitempty"`
    History     []Message    `json:"history,omitempty"` // Full message history
    Metadata    map[string]any `json:"metadata,omitempty"`
    CreatedAt   time.Time    `json:"createdAt"`
    UpdatedAt   time.Time    `json:"updatedAt"`
}

type TaskStatus struct {
    State     string     `json:"state"` // working, input-required, completed, canceled, failed
    Message   *Message   `json:"message,omitempty"`
    Timestamp time.Time  `json:"timestamp"`
}

type Artifact struct {
    Name      string     `json:"name,omitempty"`
    Parts     []Part     `json:"parts"`
    Metadata  map[string]any `json:"metadata,omitempty"`
    Index     int        `json:"index,omitempty"` // For ordered artifacts
    Append    bool       `json:"append,omitempty"` // For streaming artifacts
}
```

**Implementation Steps**:
1. Add new struct definitions for `Artifact` and update `Task`
2. Update `TaskStatus` to include `Timestamp`
3. Add state constants for spec compliance:
   ```go
   const (
       TaskStateWorking       = "working"
       TaskStateInputRequired = "input-required"
       TaskStateCompleted     = "completed"
       TaskStateCanceled      = "canceled"
       TaskStateFailed        = "failed"
   )
   ```
4. Update database layer to store new fields
5. Migrate existing tasks (add artifacts column as JSON)

**Acceptance Criteria**:
- [ ] Task struct matches A2A specification
- [ ] All task states from spec are supported
- [ ] Database schema updated with migration
- [ ] Existing tasks remain functional after migration

---

### Task 2.2: Implement Push Notification System (8 hours)

**New Files**:
- `internal/a2a/push_notifications.go` - Core push notification logic
- `internal/a2a/webhook.go` - Webhook delivery implementation
- `internal/a2a/push_test.go` - Unit tests

**Modified Files**:
- `internal/a2a/task.go` - Add push notification config to Task
- `internal/a2a/handlers.go` - Add subscribe/unsubscribe endpoints

**Implementation**:

```go
// internal/a2a/push_notifications.go

package a2a

import (
    "bytes"
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "radgateway/internal/logger"
)

// PushNotificationConfig holds webhook configuration for a task
type PushNotificationConfig struct {
    URL       string            `json:"url"`
    Token     string            `json:"token,omitempty"` // Optional auth token
    Headers   map[string]string `json:"headers,omitempty"` // Custom headers
}

// PushNotificationManager handles webhook delivery
type PushNotificationManager struct {
    client     *http.Client
    secretKey  []byte           // For signing webhooks
    maxRetries int
    retryDelay time.Duration
}

// NewPushNotificationManager creates a new manager
func NewPushNotificationManager(secretKey string) *PushNotificationManager {
    return &PushNotificationManager{
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
        secretKey:  []byte(secretKey),
        maxRetries: 3,
        retryDelay: 1 * time.Second,
    }
}

// SendNotification sends a task update webhook
func (pm *PushNotificationManager) SendNotification(
    ctx context.Context,
    config PushNotificationConfig,
    task *Task,
) error {
    payload := NotificationPayload{
        TaskID:    task.ID,
        SessionID: task.SessionID,
        Status:    task.Status,
        Timestamp: time.Now().UTC(),
    }

    body, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("marshaling notification: %w", err)
    }

    // Sign the payload
    signature := pm.signPayload(body)

    return pm.sendWithRetry(ctx, config, body, signature)
}

func (pm *PushNotificationManager) sendWithRetry(
    ctx context.Context,
    config PushNotificationConfig,
    body []byte,
    signature string,
) error {
    var lastErr error

    for attempt := 0; attempt < pm.maxRetries; attempt++ {
        if attempt > 0 {
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(pm.retryDelay * time.Duration(attempt)):
            }
        }

        req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.URL, bytes.NewReader(body))
        if err != nil {
            return fmt.Errorf("creating request: %w", err)
        }

        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("X-A2A-Signature", signature)

        if config.Token != "" {
            req.Header.Set("Authorization", "Bearer "+config.Token)
        }

        for k, v := range config.Headers {
            req.Header.Set(k, v)
        }

        resp, err := pm.client.Do(req)
        if err != nil {
            lastErr = err
            logger.Warn("webhook delivery failed, will retry",
                "attempt", attempt+1,
                "error", err,
            )
            continue
        }
        resp.Body.Close()

        if resp.StatusCode >= 200 && resp.StatusCode < 300 {
            return nil // Success
        }

        lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
        if resp.StatusCode >= 400 && resp.StatusCode < 500 {
            // Client errors - don't retry
            return lastErr
        }
    }

    return fmt.Errorf("webhook delivery failed after %d attempts: %w", pm.maxRetries, lastErr)
}

func (pm *PushNotificationManager) signPayload(body []byte) string {
    h := hmac.New(sha256.New, pm.secretKey)
    h.Write(body)
    return hex.EncodeToString(h.Sum(nil))
}

type NotificationPayload struct {
    TaskID    string     `json:"taskId"`
    SessionID string     `json:"sessionId"`
    Status    TaskStatus `json:"status"`
    Timestamp time.Time  `json:"timestamp"`
}
```

**API Endpoints to Add**:

```go
// Subscribe to push notifications for a task
POST /a2a/tasks/{taskId}/pushNotifications
Body: {
    "url": "https://agent.example.com/webhooks/a2a",
    "token": "optional-auth-token",
    "headers": {
        "X-Custom-Header": "value"
    }
}

// Unsubscribe from push notifications
DELETE /a2a/tasks/{taskId}/pushNotifications
```

**Acceptance Criteria**:
- [ ] Can subscribe to push notifications via API
- [ ] Webhooks delivered on task status changes
- [ ] Failed webhooks retried with exponential backoff
- [ ] Payload signed with HMAC-SHA256
- [ ] Webhook delivery logged for debugging
- [ ] Unsubscribe removes webhook configuration

---

### Task 2.3: Extend Message Part Types (6 hours)

**Files**:
- `internal/a2a/message.go` (modify)

**Current State**: Basic `TextPart` and `FilePart`

**Additions**:

```go
// DataPart represents structured data (JSON)
type DataPart struct {
    Type string `json:"type"` // "data"
    Data map[string]any `json:"data"`
}

// FunctionCallPart represents a function call from the agent
type FunctionCallPart struct {
    Type string `json:"type"` // "function_call"
    ID   string `json:"id"`   // Unique call ID
    Name string `json:"name"` // Function name
    Args map[string]any `json:"args"` // Arguments
}

// FunctionResponsePart represents a function result
type FunctionResponsePart struct {
    Type     string `json:"type"` // "function_response"
    CallID   string `json:"id"`   // Matches FunctionCallPart.ID
    Response map[string]any `json:"response"`
}

// Part wrapper for JSON unmarshaling
type Part struct {
    Type     string          `json:"type"`
    Raw      json.RawMessage `json:-"`
}

func (p *Part) UnmarshalJSON(data []byte) error {
    // Custom unmarshal to handle polymorphic parts
    // Based on "type" field, unmarshal into appropriate struct
}
```

**Streaming Part Support**:

```go
// StreamingPart is sent during streaming responses
type StreamingPart struct {
    Type      string    `json:"type"` // "status", "artifact", "message"
    TaskID    string    `json:"taskId"`
    Timestamp time.Time `json:"timestamp"`

    // One of these will be populated based on Type
    StatusUpdate *TaskStatus `json:"status,omitempty"`
    Artifact     *Artifact   `json:"artifact,omitempty"`
    Message      *Message    `json:"message,omitempty"`
    Final        bool        `json:"final,omitempty"` // true for last chunk
}
```

**Acceptance Criteria**:
- [ ] All A2A part types implemented
- [ ] Custom JSON unmarshaling for polymorphic parts
- [ ] Streaming parts work with SSE
- [ ] Backward compatibility with existing messages

---

### Task 2.4: Update Agent Card (3 hours)

**File**: `internal/a2a/agent_card.go`

**Additions**:

```go
// AgentCard represents the agent's capabilities and endpoints
type AgentCard struct {
    Name               string              `json:"name"`
    Description        string              `json:"description"`
    URL                string              `json:"url"`
    Provider           *ProviderInfo       `json:"provider,omitempty"`
    Version            string              `json:"version"`
    DocumentationURL   string              `json:"documentationUrl,omitempty"`

    // Enhanced capabilities
    Capabilities       AgentCapabilities   `json:"capabilities"`

    // Authentication schemes supported
    Authentication     AuthenticationInfo  `json:"authentication"`

    // Default input modes
    DefaultInputModes  []string            `json:"defaultInputModes"`
    DefaultOutputModes []string            `json:"defaultOutputModes"`

    // Skills exposed by this agent
    Skills             []Skill             `json:"skills,omitempty"`
}

type AgentCapabilities struct {
    Streaming          bool `json:"streaming"`
    PushNotifications  bool `json:"pushNotifications"` // NEW
    StateTransition    bool `json:"stateTransition"`
    HistoryTruncation  bool `json:"historyTruncation"`
}

type AuthenticationInfo struct {
    Schemes []string `json:"schemes"` // "apiKey", "oauth2", "none"
    // OAuth2 specific configuration
    OAuth2 *OAuth2Config `json:"oauth2,omitempty"`
}

type OAuth2Config struct {
    AuthorizationEndpoint string   `json:"authorizationEndpoint"`
    TokenEndpoint         string   `json:"tokenEndpoint"`
    Scopes                []string `json:"scopes"`
}

type Skill struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Tags        []string `json:"tags,omitempty"`
    Examples    []string `json:"examples,omitempty"`
    InputModes  []string `json:"inputModes,omitempty"`
    OutputModes []string `json:"outputModes,omitempty"`
}
```

**Acceptance Criteria**:
- [ ] Agent card includes push notification capability
- [ ] Authentication schemes properly documented
- [ ] Skills array exposed
- [ ] Backward compatible with clients expecting old format

---

### Task 2.5: Update Streaming Format (4 hours)

**File**: `internal/a2a/streaming.go`

**Current Issue**: Streaming format doesn't match A2A spec exactly

**Target Implementation**:

```go
// SSEEvent represents a server-sent event per A2A spec
type SSEEvent struct {
    Event string // Event type (status, artifact, message)
    Data  string // JSON payload
    ID    string // Optional event ID
}

// StreamWriter handles SSE output
func (s *StreamWriter) WriteEvent(event SSEEvent) error {
    if event.Event != "" {
        fmt.Fprintf(s.w, "event: %s\n", event.Event)
    }
    if event.ID != "" {
        fmt.Fprintf(s.w, "id: %s\n", event.ID)
    }

    // Data can be multi-line
    lines := strings.Split(event.Data, "\n")
    for _, line := range lines {
        fmt.Fprintf(s.w, "data: %s\n", line)
    }

    fmt.Fprint(s.w, "\n")

    if flusher, ok := s.w.(http.Flusher); ok {
        flusher.Flush()
    }

    return s.w.Err()
}

// TaskStatusEvent sends a status update
type TaskStatusEvent struct {
    TaskID string     `json:"id"`
    Status TaskStatus `json:"status"`
    Final  bool       `json:"final,omitempty"`
}

// TaskArtifactEvent sends an artifact
type TaskArtifactEvent struct {
    TaskID   string    `json:"id"`
    Artifact Artifact  `json:"artifact"`
    Final    bool      `json:"final,omitempty"`
}
```

**Acceptance Criteria**:
- [ ] SSE format matches A2A specification
- [ ] Events properly formatted with event/data/fields
- [ ] Multi-line data handled correctly
- [ ] Flushing works for real-time updates

---

### Task 2.6: Add Task History Support (3 hours)

**Files**:
- `internal/a2a/task.go` - Add history field
- `internal/a2a/task_store.go` - Update store methods

**Implementation**:

```go
// TaskStore interface with history support
type TaskStore interface {
    Create(task *Task) error
    Get(id string) (*Task, error)
    Update(task *Task) error
    UpdateStatus(id string, status TaskStatus) error
    AddMessage(id string, message Message) error // NEW
    Delete(id string) error
    List(sessionID string) ([]Task, error)

    // History management
    GetHistory(id string, offset, limit int) ([]Message, error)
    TruncateHistory(id string, keep int) error
}

// AddMessage adds a message to task history and updates status
func (s *SQLTaskStore) AddMessage(id string, message Message) error {
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Add to history
    historyJSON, err := json.Marshal(message)
    if err != nil {
        return err
    }

    _, err = tx.Exec(`
        UPDATE tasks
        SET history = jsonb_insert(history, '$[0]', ?::jsonb),
            updated_at = NOW()
        WHERE id = ?
    `, historyJSON, id)
    if err != nil {
        return err
    }

    return tx.Commit()
}
```

**Acceptance Criteria**:
- [ ] Messages stored in task history
- [ ] History can be retrieved with pagination
- [ ] History truncation supported
- [ ] Works with PostgreSQL and SQLite backends

---

### Task 2.7: Update HTTP Handlers (4 hours)

**File**: `internal/a2a/handlers.go`

**New Endpoints**:

```go
// Subscribe to push notifications
func (h *Handler) SubscribePushNotifications(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    taskID := vars["taskId"]

    var req struct {
        URL     string            `json:"url"`
        Token   string            `json:"token,omitempty"`
        Headers map[string]string `json:"headers,omitempty"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Validate URL
    if req.URL == "" {
        http.Error(w, "URL is required", http.StatusBadRequest)
        return
    }

    // Store subscription
    config := PushNotificationConfig{
        URL:     req.URL,
        Token:   req.Token,
        Headers: req.Headers,
    }

    if err := h.store.SubscribePushNotifications(taskID, config); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "subscribed",
    })
}

// Cancel task
func (h *Handler) CancelTask(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    taskID := vars["taskId"]

    task, err := h.store.Get(taskID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    // Can only cancel working or input-required tasks
    if task.Status.State != TaskStateWorking && task.Status.State != TaskStateInputRequired {
        http.Error(w, "Task cannot be canceled in current state", http.StatusConflict)
        return
    }

    task.Status.State = TaskStateCanceled
    task.Status.Timestamp = time.Now()

    if err := h.store.Update(task); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Send notification
    if h.pushManager != nil {
        h.pushManager.SendNotification(r.Context(), task.PushConfig, task)
    }

    json.NewEncoder(w).Encode(task)
}

// Get task history
func (h *Handler) GetTaskHistory(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    taskID := vars["taskId"]

    offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit == 0 {
        limit = 50
    }

    history, err := h.store.GetHistory(taskID, offset, limit)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]interface{}{
        "messages": history,
        "offset":   offset,
        "limit":    limit,
    })
}
```

**Route Updates**:

```go
// Add to router setup
router.HandleFunc("/a2a/tasks/{taskId}/pushNotifications", h.SubscribePushNotifications).Methods("POST")
router.HandleFunc("/a2a/tasks/{taskId}/pushNotifications", h.UnsubscribePushNotifications).Methods("DELETE")
router.HandleFunc("/a2a/tasks/{taskId}/cancel", h.CancelTask).Methods("POST")
router.HandleFunc("/a2a/tasks/{taskId}/history", h.GetTaskHistory).Methods("GET")
```

**Acceptance Criteria**:
- [ ] All new endpoints functional
- [ ] Proper error handling and status codes
- [ ] Input validation
- [ ] Push notifications sent on state changes

---

### Task 2.8: Authentication Enhancement (3 hours)

**File**: `internal/a2a/auth.go` (new)

```go
package a2a

import (
    "context"
    "net/http"
    "strings"
)

// AuthMiddleware provides A2A-specific authentication
type AuthMiddleware struct {
    apiKeyAuth   func(key string) (bool, error)
    oauth2Auth   func(token string) (bool, error)
    allowedSchemes []string
}

func NewAuthMiddleware(schemes []string) *AuthMiddleware {
    return &AuthMiddleware{
        allowedSchemes: schemes,
    }
}

func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Check for Authorization header
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            // Check for API key in header
            apiKey := r.Header.Get("X-API-Key")
            if apiKey != "" && m.apiKeyAuth != nil {
                valid, err := m.apiKeyAuth(apiKey)
                if valid && err == nil {
                    next.ServeHTTP(w, r)
                    return
                }
            }

            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 {
            http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
            return
        }

        scheme := strings.ToLower(parts[0])
        token := parts[1]

        switch scheme {
        case "bearer":
            if m.oauth2Auth != nil {
                valid, err := m.oauth2Auth(token)
                if valid && err == nil {
                    next.ServeHTTP(w, r)
                    return
                }
            }
        case "apikey":
            if m.apiKeyAuth != nil {
                valid, err := m.apiKeyAuth(token)
                if valid && err == nil {
                    next.ServeHTTP(w, r)
                    return
                }
            }
        }

        http.Error(w, "Unauthorized", http.StatusUnauthorized)
    })
}
```

**Acceptance Criteria**:
- [ ] Supports API key auth (X-API-Key header)
- [ ] Supports Bearer token auth
- [ ] Respects agent card authentication schemes
- [ ] Proper error responses

---

### Task 2.9: Testing & Validation (4 hours)

**Test Files**:
- `internal/a2a/task_test.go` - Task model tests
- `internal/a2a/push_notifications_test.go` - Push notification tests
- `internal/a2a/handlers_test.go` - HTTP handler tests
- `internal/a2a/integration_test.go` - Integration tests

**Test Cases**:

1. **Task Model Tests**:
   ```go
   func TestTaskStates(t *testing.T) {
       states := []string{
           TaskStateWorking,
           TaskStateInputRequired,
           TaskStateCompleted,
           TaskStateCanceled,
           TaskStateFailed,
       }

       for _, state := range states {
           task := &Task{
               ID:    "test-task",
               Status: TaskStatus{State: state},
           }
           // Verify JSON serialization
           data, err := json.Marshal(task)
           require.NoError(t, err)
           require.Contains(t, string(data), state)
       }
   }
   ```

2. **Push Notification Tests**:
   ```go
   func TestPushNotificationDelivery(t *testing.T) {
       // Start test server
       server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
           assert.Equal(t, "POST", r.Method)
           assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
           assert.NotEmpty(t, r.Header.Get("X-A2A-Signature"))
           w.WriteHeader(http.StatusOK)
       }))
       defer server.Close()

       manager := NewPushNotificationManager("test-secret")

       task := &Task{
           ID:    "task-1",
           Status: TaskStatus{State: TaskStateCompleted},
       }

       config := PushNotificationConfig{
           URL:   server.URL,
           Token: "test-token",
       }

       err := manager.SendNotification(context.Background(), config, task)
       require.NoError(t, err)
   }
   ```

3. **Handler Tests**:
   ```go
   func TestSubscribePushNotifications(t *testing.T) {
       // Setup handler with mock store
       // POST subscription
       // Verify 201 Created
       // Verify config stored
   }

   func TestCancelTask(t *testing.T) {
       // Create working task
       // POST cancel
       // Verify state is "canceled"
       // Verify notification sent
   }
   ```

**Validation Against Spec**:

Create a compliance checklist:
- [ ] All task states match spec
- [ ] Message parts match spec
- [ ] Streaming format matches spec
- [ ] Agent card format matches spec
- [ ] Authentication schemes match spec

**Acceptance Criteria**:
- [ ] All new code has unit tests
- [ ] Integration tests pass
- [ ] Tests validate spec compliance
- [ ] Coverage > 80% for new code

---

## 3. Implementation Order

### Phase 1: Foundation (Day 1)
1. **Task 2.1**: Update Task Model
2. **Task 2.6**: Add Task History Support
3. **Task 2.4**: Update Agent Card

### Phase 2: Core Features (Day 2)
4. **Task 2.2**: Implement Push Notifications
5. **Task 2.3**: Extend Message Part Types
6. **Task 2.5**: Update Streaming Format

### Phase 3: API & Auth (Day 3)
7. **Task 2.7**: Update HTTP Handlers
8. **Task 2.8**: Authentication Enhancement

### Phase 4: Testing (Day 4)
9. **Task 2.9**: Testing & Validation
10. Documentation updates
11. Integration verification

---

## 4. Database Migrations

### Migration 1: Add Task Artifacts and History

```sql
-- PostgreSQL
ALTER TABLE a2a_tasks
ADD COLUMN IF NOT EXISTS artifacts JSONB DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS history JSONB DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb,
ADD COLUMN IF NOT EXISTS push_config JSONB DEFAULT NULL;

-- Create index for history queries
CREATE INDEX IF NOT EXISTS idx_a2a_tasks_session ON a2a_tasks(session_id);

-- SQLite
ALTER TABLE a2a_tasks ADD COLUMN artifacts TEXT DEFAULT '[]';
ALTER TABLE a2a_tasks ADD COLUMN history TEXT DEFAULT '[]';
ALTER TABLE a2a_tasks ADD COLUMN metadata TEXT DEFAULT '{}';
ALTER TABLE a2a_tasks ADD COLUMN push_config TEXT DEFAULT NULL;
```

### Migration 2: Add Webhook Delivery Log

```sql
-- PostgreSQL
CREATE TABLE IF NOT EXISTS a2a_webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id TEXT NOT NULL REFERENCES a2a_tasks(id) ON DELETE CASCADE,
    webhook_url TEXT NOT NULL,
    payload JSONB NOT NULL,
    response_status INT,
    response_body TEXT,
    error_message TEXT,
    attempt_count INT DEFAULT 1,
    delivered_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_a2a_webhook_task ON a2a_webhook_deliveries(task_id);
CREATE INDEX idx_a2a_webhook_created ON a2a_webhook_deliveries(created_at);
```

---

## 5. Configuration

### Environment Variables

```bash
# Push notifications
A2A_WEBHOOK_SECRET=your-webhook-signing-secret
A2A_WEBHOOK_MAX_RETRIES=3
A2A_WEBHOOK_TIMEOUT_SECONDS=30

# Authentication
A2A_AUTH_SCHEMES=apikey,oauth2  # Comma-separated list
A2A_REQUIRE_AUTH=true
```

### Config File Structure

```yaml
a2a:
  authentication:
    schemes:
      - apikey
      - oauth2
    required: true

  push_notifications:
    enabled: true
    secret: "${A2A_WEBHOOK_SECRET}"
    max_retries: 3
    timeout: 30s
    retry_delay: 1s

  streaming:
    heartbeat_interval: 15s
    max_chunk_size: 65536
```

---

## 6. Error Codes

Standardize A2A error responses:

```go
var (
    ErrTaskNotFound = &A2AError{
        Code:    "task_not_found",
        Message: "Task not found",
        HTTPStatus: http.StatusNotFound,
    }

    ErrInvalidState = &A2AError{
        Code:    "invalid_state",
        Message: "Operation not allowed in current task state",
        HTTPStatus: http.StatusConflict,
    }

    ErrInvalidPart = &A2AError{
        Code:    "invalid_part",
        Message: "Invalid message part type",
        HTTPStatus: http.StatusBadRequest,
    }

    ErrWebhookDelivery = &A2AError{
        Code:    "webhook_delivery_failed",
        Message: "Failed to deliver webhook notification",
        HTTPStatus: http.StatusInternalServerError,
    }
)

type A2AError struct {
    Code       string `json:"code"`
    Message    string `json:"message"`
    Details    string `json:"details,omitempty"`
    HTTPStatus int    `json:-"`
}

func (e *A2AError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
```

---

## 7. Testing Strategy

### Unit Tests
- All new structs and methods
- JSON serialization/deserialization
- State transitions
- Part type handling

### Integration Tests
- Full task lifecycle
- Push notification delivery
- Streaming SSE format
- Authentication flows

### Compliance Tests
- Verify against A2A spec examples
- Test with reference client
- Validate JSON schema

---

## 8. Documentation Updates

### Files to Update
1. `/docs/protocols/a2a.md` - Protocol documentation
2. `/docs/api/a2a-endpoints.md` - API reference
3. `/docs/tutorials/a2a-integration.md` - Integration guide
4. `README.md` - Feature status

### New Documentation
1. Push notifications guide
2. Authentication setup
3. Webhook troubleshooting

---

## 9. Deployment Considerations

### Breaking Changes
- Task model changes require migration
- Agent card format enhanced (additive only)

### Rollback Plan
- Keep old task columns during migration
- Feature flags for new functionality
- Database rollback scripts

### Monitoring
- Webhook delivery success rate
- Authentication failure rate
- Task state transition latency

---

## 10. Acceptance Criteria Summary

### Functional
- [ ] All A2A spec endpoints implemented
- [ ] Task model fully compliant
- [ ] Push notifications working
- [ ] All part types supported
- [ ] Streaming format correct
- [ ] Authentication schemes implemented

### Non-Functional
- [ ] All tests pass (>80% coverage)
- [ ] Database migrations reversible
- [ ] Backward compatibility maintained
- [ ] Performance impact < 10%
- [ ] Documentation complete

---

## 11. Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Database migration failure | High | Test migrations in staging; backup before deploy |
| Breaking changes to clients | Medium | Additive changes only; deprecate gradually |
| Webhook delivery failures | Medium | Retry logic; dead letter queue; monitoring |
| Performance degradation | Low | Benchmark before/after; optimize queries |

---

## 12. Success Metrics

1. **Compliance**: 100% of A2A spec requirements met
2. **Test Coverage**: >80% for new code
3. **Performance**: No degradation in task throughput
4. **Reliability**: <0.1% webhook delivery failure rate
5. **Adoption**: All internal teams using compliant endpoints

---

**Next Steps**: Review this plan with Team Alpha (Architecture) and Team Charlie (Security), then proceed to implementation starting with Task 2.1.

**Questions?**: Contact the A2A workstream lead or raise in #a2a-protocol channel.
