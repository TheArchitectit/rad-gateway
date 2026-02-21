# A2A Protocol Compliance Sprint

## Overview

**Duration**: 4 days
**Sprint Goal**: Bring RAD Gateway A2A implementation into full compliance with Google A2A specification
**Execution Mode**: Agent-driven development with TDD
**Target Branch**: `feature/a2a-protocol-compliance`

---

## Team Assignment

| Role | Agent | Responsibility |
|------|-------|----------------|
| **Sprint Lead** | `architect` | Technical decisions, unblocking, reviews |
| **Backend Developer** | `golang-pro` | Task model, database, core logic |
| **Webhook Specialist** | `backend-developer` | Push notifications, delivery system |
| **Protocol Expert** | `api-designer` | Message parts, streaming format, spec compliance |
| **QA Engineer** | `qa-expert` | Test coverage, validation, integration tests |

---

## Sprint Structure

### Day 1: Foundation (Task Model & History)
**Theme**: Core data structures and persistence

#### Task 1.1: Update Task Model
**Assignee**: `golang-pro`
**Estimated**: 3 hours
**Dependencies**: None

**Files to Modify**:
- `internal/a2a/task.go` (modify)
- `internal/db/migrations/` (add migration)

**Step-by-Step Execution**:

```bash
# Step 1: Create branch
git checkout -b feature/a2a-protocol-compliance
git push -u origin feature/a2a-protocol-compliance

# Step 2: Read current task.go
head -100 internal/a2a/task.go
```

**Implementation**:

1. **Add new Task struct fields** (`internal/a2a/task.go:15-50`):
```go
// Task represents an A2A task per specification
type Task struct {
    ID            string         `json:"id"`
    SessionID     string         `json:"sessionId"`
    Status        TaskStatus     `json:"status"`
    Artifacts     []Artifact     `json:"artifacts,omitempty"`
    History       []Message      `json:"history,omitempty"`
    Metadata      map[string]any `json:"metadata,omitempty"`
    PushConfig    *PushNotificationConfig `json:"-" db:"push_config"`
    CreatedAt     time.Time      `json:"createdAt"`
    UpdatedAt     time.Time      `json:"updatedAt"`
}
```

2. **Add Artifact struct** (`internal/a2a/task.go:52-70`):
```go
type Artifact struct {
    Name     string         `json:"name,omitempty"`
    Parts    []Part         `json:"parts"`
    Metadata map[string]any `json:"metadata,omitempty"`
    Index    int            `json:"index,omitempty"`
    Append   bool           `json:"append,omitempty"`
}
```

3. **Add state constants** (`internal/a2a/task.go:72-85`):
```go
const (
    TaskStateWorking       = "working"
    TaskStateInputRequired = "input-required"
    TaskStateCompleted     = "completed"
    TaskStateCanceled      = "canceled"
    TaskStateFailed        = "failed"
)
```

4. **Update TaskStatus** (`internal/a2a/task.go:87-95`):
```go
type TaskStatus struct {
    State     string    `json:"state"`
    Message   *Message  `json:"message,omitempty"`
    Timestamp time.Time `json:"timestamp"`
}
```

**Tests to Write** (`internal/a2a/task_test.go`):
```go
func TestTaskJSONSerialization(t *testing.T) {
    task := &Task{
        ID: "task-123",
        Status: TaskStatus{
            State: TaskStateWorking,
            Timestamp: time.Now(),
        },
        Artifacts: []Artifact{
            {Name: "result", Parts: []Part{{Type: "text", Text: "Hello"}}},
        },
    }

    data, err := json.Marshal(task)
    require.NoError(t, err)

    var decoded Task
    err = json.Unmarshal(data, &decoded)
    require.NoError(t, err)

    assert.Equal(t, task.ID, decoded.ID)
    assert.Equal(t, TaskStateWorking, decoded.Status.State)
    assert.Len(t, decoded.Artifacts, 1)
}
```

**Verification Commands**:
```bash
go test ./internal/a2a/ -v -run TestTaskJSONSerialization
go build ./...
```

**Definition of Done**:
- [ ] Task struct includes Artifacts, History, Metadata, PushConfig fields
- [ ] All task state constants defined
- [ ] JSON serialization/deserialization works correctly
- [ ] Tests pass

---

#### Task 1.2: Create Database Migration
**Assignee**: `golang-pro`
**Estimated**: 2 hours
**Dependencies**: Task 1.1

**Files to Modify**:
- `cmd/migrate/migrations/000008_a2a_task_enhancements.up.sql` (new)
- `cmd/migrate/migrations/000008_a2a_task_enhancements.down.sql` (new)

**Implementation**:

**PostgreSQL Migration** (`cmd/migrate/migrations/000008_a2a_task_enhancements.up.sql`):
```sql
-- Add A2A spec-compliant columns to tasks table
ALTER TABLE a2a_tasks
    ADD COLUMN IF NOT EXISTS artifacts JSONB DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS history JSONB DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS push_config JSONB DEFAULT NULL;

-- Add index for history queries
CREATE INDEX IF NOT EXISTS idx_a2a_tasks_session_history
ON a2a_tasks(session_id, updated_at DESC);
```

**SQLite Migration** (`cmd/migrate/migrations/000008_a2a_task_enhancements.sqlite.up.sql`):
```sql
-- SQLite uses TEXT for JSON
ALTER TABLE a2a_tasks ADD COLUMN artifacts TEXT DEFAULT '[]';
ALTER TABLE a2a_tasks ADD COLUMN history TEXT DEFAULT '[]';
ALTER TABLE a2a_tasks ADD COLUMN metadata TEXT DEFAULT '{}';
ALTER TABLE a2a_tasks ADD COLUMN push_config TEXT DEFAULT NULL;

-- Create index
CREATE INDEX IF NOT EXISTS idx_a2a_tasks_session
ON a2a_tasks(session_id);
```

**Down Migration** (`cmd/migrate/migrations/000008_a2a_task_enhancements.down.sql`):
```sql
ALTER TABLE a2a_tasks
    DROP COLUMN IF EXISTS artifacts,
    DROP COLUMN IF EXISTS history,
    DROP COLUMN IF EXISTS metadata,
    DROP COLUMN IF EXISTS push_config;

DROP INDEX IF EXISTS idx_a2a_tasks_session_history;
```

**Verification**:
```bash
# Run migration
go run ./cmd/migrate up

# Verify columns exist
psql $DATABASE_URL -c "\d a2a_tasks"
```

**Definition of Done**:
- [ ] PostgreSQL migration file created
- [ ] SQLite migration file created
- [ ] Migration runs successfully
- [ ] Down migration tested

---

#### Task 1.3: Update Task Store Interface
**Assignee**: `golang-pro`
**Estimated**: 3 hours
**Dependencies**: Task 1.2

**Files to Modify**:
- `internal/a2a/task_store.go` (modify)
- `internal/a2a/task_store_sql.go` (modify)

**Implementation**:

1. **Update interface** (`internal/a2a/task_store.go:25-45`):
```go
type TaskStore interface {
    Create(task *Task) error
    Get(id string) (*Task, error)
    Update(task *Task) error
    UpdateStatus(id string, status TaskStatus) error
    AddMessage(id string, message Message) error        // NEW
    AddArtifact(id string, artifact Artifact) error     // NEW
    GetHistory(id string, offset, limit int) ([]Message, error)  // NEW
    TruncateHistory(id string, keep int) error          // NEW
    SubscribePushNotifications(id string, config PushNotificationConfig) error  // NEW
    Delete(id string) error
    List(sessionID string) ([]Task, error)
}
```

2. **Implement SQL methods** (`internal/a2a/task_store_sql.go`):
```go
// AddMessage adds a message to task history
func (s *SQLTaskStore) AddMessage(id string, message Message) error {
    historyJSON, err := json.Marshal([]Message{message})
    if err != nil {
        return fmt.Errorf("marshaling message: %w", err)
    }

    // Append to JSON array in database
    query := `
        UPDATE a2a_tasks
        SET history = COALESCE(history, '[]'::jsonb) || ?::jsonb,
            updated_at = NOW()
        WHERE id = ?
    `

    _, err = s.db.Exec(query, historyJSON, id)
    return err
}

// GetHistory retrieves paginated message history
func (s *SQLTaskStore) GetHistory(id string, offset, limit int) ([]Message, error) {
    // For PostgreSQL with jsonb
    query := `
        SELECT jsonb_array_elements(history) as message
        FROM a2a_tasks
        WHERE id = ?
        ORDER BY (message->>'timestamp') DESC
        OFFSET ? LIMIT ?
    `

    rows, err := s.db.Query(query, id, offset, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var messages []Message
    for rows.Next() {
        var msgJSON []byte
        if err := rows.Scan(&msgJSON); err != nil {
            return nil, err
        }

        var msg Message
        if err := json.Unmarshal(msgJSON, &msg); err != nil {
            return nil, err
        }
        messages = append(messages, msg)
    }

    return messages, rows.Err()
}

// SubscribePushNotifications stores webhook config
func (s *SQLTaskStore) SubscribePushNotifications(
    id string,
    config PushNotificationConfig,
) error {
    configJSON, err := json.Marshal(config)
    if err != nil {
        return fmt.Errorf("marshaling config: %w", err)
    }

    query := `
        UPDATE a2a_tasks
        SET push_config = ?::jsonb,
            updated_at = NOW()
        WHERE id = ?
    `

    _, err = s.db.Exec(query, configJSON, id)
    return err
}
```

**Tests** (`internal/a2a/task_store_test.go`):
```go
func TestTaskStore_AddMessage(t *testing.T) {
    store := setupTestStore(t)

    task := &Task{
        ID:        "test-task",
        SessionID: "session-1",
        Status:    TaskStatus{State: TaskStateWorking},
    }
    require.NoError(t, store.Create(task))

    msg := Message{
        Role: "user",
        Parts: []Part{{Type: "text", Text: "Hello"}},
    }

    err := store.AddMessage(task.ID, msg)
    require.NoError(t, err)

    history, err := store.GetHistory(task.ID, 0, 10)
    require.NoError(t, err)
    assert.Len(t, history, 1)
    assert.Equal(t, "user", history[0].Role)
}
```

**Verification**:
```bash
go test ./internal/a2a/ -v -run TestTaskStore
```

---

### Day 2: Push Notifications & Webhooks
**Theme**: Real-time notification system

#### Task 2.1: Implement Push Notification Manager
**Assignee**: `backend-developer`
**Estimated**: 4 hours
**Dependencies**: Day 1

**Files to Create**:
- `internal/a2a/push_notifications.go` (new)
- `internal/a2a/push_notifications_test.go` (new)

**Step-by-Step**:

1. **Create notification manager** (`internal/a2a/push_notifications.go:1-150`):
```go
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

// PushNotificationConfig holds webhook configuration
type PushNotificationConfig struct {
    URL       string            `json:"url"`
    Token     string            `json:"token,omitempty"`
    Headers   map[string]string `json:"headers,omitempty"`
}

// NotificationPayload is sent to webhooks
type NotificationPayload struct {
    TaskID    string     `json:"taskId"`
    SessionID string     `json:"sessionId"`
    Status    TaskStatus `json:"status"`
    Timestamp time.Time  `json:"timestamp"`
}

// PushNotificationManager handles webhook delivery
type PushNotificationManager struct {
    client     *http.Client
    secretKey  []byte
    maxRetries int
    retryDelay time.Duration
}

// NewPushNotificationManager creates manager with signing key
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

// SendNotification delivers task update to webhook
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
            delay := pm.retryDelay * time.Duration(1<<uint(attempt-1))
            if delay > 30*time.Second {
                delay = 30 * time.Second
            }

            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(delay):
            }
        }

        req, err := http.NewRequestWithContext(
            ctx,
            http.MethodPost,
            config.URL,
            bytes.NewReader(body),
        )
        if err != nil {
            return fmt.Errorf("creating request: %w", err)
        }

        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("X-A2A-Signature", signature)
        req.Header.Set("X-A2A-Event", "task.status_update")

        if config.Token != "" {
            req.Header.Set("Authorization", "Bearer "+config.Token)
        }

        for k, v := range config.Headers {
            req.Header.Set(k, v)
        }

        resp, err := pm.client.Do(req)
        if err != nil {
            lastErr = err
            logger.Warn("webhook delivery failed, retrying",
                "attempt", attempt+1,
                "error", err,
            )
            continue
        }
        resp.Body.Close()

        if resp.StatusCode >= 200 && resp.StatusCode < 300 {
            return nil
        }

        lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)

        // Don't retry on client errors (except 429)
        if resp.StatusCode >= 400 && resp.StatusCode < 500 {
            if resp.StatusCode != http.StatusTooManyRequests {
                return lastErr
            }
        }
    }

    return fmt.Errorf("webhook delivery failed after %d attempts: %w",
        pm.maxRetries, lastErr)
}

func (pm *PushNotificationManager) signPayload(body []byte) string {
    h := hmac.New(sha256.New, pm.secretKey)
    h.Write(body)
    return hex.EncodeToString(h.Sum(nil))
}
```

2. **Write comprehensive tests**:
```go
func TestPushNotificationManager_SendNotification(t *testing.T) {
    // Setup test server
    received := make(chan *http.Request, 1)
    server := httptest.NewServer(http.HandlerFunc(
        func(w http.ResponseWriter, r *http.Request) {
            received <- r
            w.WriteHeader(http.StatusOK)
        },
    ))
    defer server.Close()

    pm := NewPushNotificationManager("test-secret")

    task := &Task{
        ID:        "task-123",
        SessionID: "session-456",
        Status:    TaskStatus{State: TaskStateCompleted},
    }

    config := PushNotificationConfig{
        URL:   server.URL,
        Token: "test-token",
    }

    err := pm.SendNotification(context.Background(), config, task)
    require.NoError(t, err)

    // Verify request
    req := <-received
    assert.Equal(t, "POST", req.Method)
    assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
    assert.NotEmpty(t, req.Header.Get("X-A2A-Signature"))
    assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
}

func TestPushNotificationManager_RetryOnFailure(t *testing.T) {
    attempts := 0
    server := httptest.NewServer(http.HandlerFunc(
        func(w http.ResponseWriter, r *http.Request) {
            attempts++
            if attempts < 3 {
                w.WriteHeader(http.StatusServiceUnavailable)
                return
            }
            w.WriteHeader(http.StatusOK)
        },
    ))
    defer server.Close()

    pm := NewPushNotificationManager("test-secret")
    pm.maxRetries = 3
    pm.retryDelay = 100 * time.Millisecond

    task := &Task{ID: "task-123", Status: TaskStatus{State: TaskStateWorking}}
    config := PushNotificationConfig{URL: server.URL}

    err := pm.SendNotification(context.Background(), config, task)
    require.NoError(t, err)
    assert.Equal(t, 3, attempts)
}
```

**Verification**:
```bash
go test ./internal/a2a/ -v -run TestPushNotificationManager
```

---

#### Task 2.2: Add Webhook Delivery Logging
**Assignee**: `backend-developer`
**Estimated**: 2 hours
**Dependencies**: Task 2.1

**Files to Create**:
- `internal/a2a/webhook_log.go` (new)
- Database migration for webhook deliveries table

**Implementation**:

```go
// WebhookDeliveryLog tracks webhook attempts
type WebhookDeliveryLog struct {
    ID            string    `json:"id" db:"id"`
    TaskID        string    `json:"taskId" db:"task_id"`
    WebhookURL    string    `json:"webhookUrl" db:"webhook_url"`
    Payload       []byte    `json:"-" db:"payload"`
    ResponseStatus int      `json:"responseStatus" db:"response_status"`
    ResponseBody  string    `json:"responseBody,omitempty" db:"response_body"`
    ErrorMessage  string    `json:"errorMessage,omitempty" db:"error_message"`
    AttemptCount  int       `json:"attemptCount" db:"attempt_count"`
    DeliveredAt   *time.Time `json:"deliveredAt,omitempty" db:"delivered_at"`
    CreatedAt     time.Time `json:"createdAt" db:"created_at"`
}

type WebhookLogStore interface {
    LogAttempt(log *WebhookDeliveryLog) error
    UpdateDelivery(logID string, status int, body string, delivered bool) error
    GetDeliveryHistory(taskID string, limit int) ([]WebhookDeliveryLog, error)
}
```

**Migration**:
```sql
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
CREATE INDEX idx_a2a_webhook_created ON a2a_webhook_deliveries(created_at DESC);
```

---

#### Task 2.3: Integrate Notifications into Task Flow
**Assignee**: `golang-pro`
**Estimated**: 2 hours
**Dependencies**: Task 2.1

**Files to Modify**:
- `internal/a2a/task_handlers.go` (modify)

**Implementation**:

Update task status handlers to trigger notifications:

```go
func (h *TaskHandler) updateTaskStatus(
    ctx context.Context,
    taskID string,
    status TaskStatus,
) error {
    // Update in database
    if err := h.store.UpdateStatus(taskID, status); err != nil {
        return err
    }

    // Fetch complete task for notification
    task, err := h.store.Get(taskID)
    if err != nil {
        return err
    }

    // Send push notification if configured
    if task.PushConfig != nil && h.pushManager != nil {
        if err := h.pushManager.SendNotification(ctx, *task.PushConfig, task); err != nil {
            // Log but don't fail the update
            logger.Error("failed to send push notification",
                "task_id", taskID,
                "error", err,
            )
        }
    }

    return nil
}
```

---

### Day 3: Message Parts & Streaming
**Theme**: Protocol format compliance

#### Task 3.1: Extend Message Part Types
**Assignee**: `api-designer`
**Estimated**: 3 hours
**Dependencies**: None (can parallel with Day 2)

**Files to Modify**:
- `internal/a2a/message.go` (modify)

**Implementation**:

```go
// Part is a polymorphic message part
type Part struct {
    Type string `json:"type"`

    // Text part fields
    Text string `json:"text,omitempty"`

    // File part fields
    File *FileContent `json:"file,omitempty"`

    // Data part fields
    Data map[string]any `json:"data,omitempty"`

    // Function call fields
    FunctionCall *FunctionCall `json:"functionCall,omitempty"`

    // Function response fields
    FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
}

type FileContent struct {
    Name     string `json:"name"`
    MimeType string `json:"mimeType"`
    Bytes    []byte `json:"bytes,omitempty"`
    URI      string `json:"uri,omitempty"`
}

type FunctionCall struct {
    ID   string         `json:"id"`
    Name string         `json:"name"`
    Args map[string]any `json:"args"`
}

type FunctionResponse struct {
    CallID   string         `json:"id"`
    Response map[string]any `json:"response"`
}

// UnmarshalJSON implements custom unmarshaling for polymorphic parts
func (p *Part) UnmarshalJSON(data []byte) error {
    var base struct {
        Type string `json:"type"`
    }
    if err := json.Unmarshal(data, &base); err != nil {
        return err
    }

    p.Type = base.Type

    switch base.Type {
    case "text":
        var text struct {
            Text string `json:"text"`
        }
        if err := json.Unmarshal(data, &text); err != nil {
            return err
        }
        p.Text = text.Text

    case "file":
        var file struct {
            File FileContent `json:"file"`
        }
        if err := json.Unmarshal(data, &file); err != nil {
            return err
        }
        p.File = &file.File

    case "data":
        var dataPart struct {
            Data map[string]any `json:"data"`
        }
        if err := json.Unmarshal(data, &dataPart); err != nil {
            return err
        }
        p.Data = dataPart.Data

    case "function_call":
        var fc struct {
            FunctionCall FunctionCall `json:"functionCall"`
        }
        if err := json.Unmarshal(data, &fc); err != nil {
            return err
        }
        p.FunctionCall = &fc.FunctionCall

    case "function_response":
        var fr struct {
            FunctionResponse FunctionResponse `json:"functionResponse"`
        }
        if err := json.Unmarshal(data, &fr); err != nil {
            return err
        }
        p.FunctionResponse = &fr.FunctionResponse

    default:
        return fmt.Errorf("unknown part type: %s", base.Type)
    }

    return nil
}
```

**Tests**:
```go
func TestPart_UnmarshalJSON(t *testing.T) {
    tests := []struct {
        name     string
        json     string
        expected Part
    }{
        {
            name: "text part",
            json: `{"type":"text","text":"Hello"}`,
            expected: Part{Type: "text", Text: "Hello"},
        },
        {
            name: "file part",
            json: `{"type":"file","file":{"name":"doc.pdf","mimeType":"application/pdf"}}`,
            expected: Part{
                Type: "file",
                File: &FileContent{Name: "doc.pdf", MimeType: "application/pdf"},
            },
        },
        {
            name: "function call",
            json: `{"type":"function_call","functionCall":{"id":"call-1","name":"get_weather","args":{"city":"Paris"}}}`,
            expected: Part{
                Type: "function_call",
                FunctionCall: &FunctionCall{
                    ID:   "call-1",
                    Name: "get_weather",
                    Args: map[string]any{"city": "Paris"},
                },
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var part Part
            err := json.Unmarshal([]byte(tt.json), &part)
            require.NoError(t, err)
            assert.Equal(t, tt.expected, part)
        })
    }
}
```

---

#### Task 3.2: Update Streaming Format
**Assignee**: `api-designer`
**Estimated**: 3 hours
**Dependencies**: Task 3.1

**Files to Create/Modify**:
- `internal/a2a/streaming.go` (new or modify)
- `internal/a2a/handlers.go` (modify streaming handler)

**Implementation**:

```go
// SSEEvent represents a server-sent event per A2A spec
type SSEEvent struct {
    Event string // Event type: status, artifact, message
    Data  string // JSON payload
    ID    string // Optional event ID
}

// StreamWriter handles SSE output
type StreamWriter struct {
    w       http.ResponseWriter
    flusher http.Flusher
}

func NewStreamWriter(w http.ResponseWriter) *StreamWriter {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        panic("ResponseWriter does not support flushing")
    }

    return &StreamWriter{w: w, flusher: flusher}
}

func (s *StreamWriter) WriteEvent(event SSEEvent) error {
    if event.Event != "" {
        fmt.Fprintf(s.w, "event: %s\n", event.Event)
    }
    if event.ID != "" {
        fmt.Fprintf(s.w, "id: %s\n", event.ID)
    }

    // Data can be multi-line - each line prefixed with "data: "
    lines := strings.Split(event.Data, "\n")
    for _, line := range lines {
        fmt.Fprintf(s.w, "data: %s\n", line)
    }

    fmt.Fprint(s.w, "\n")
    s.flusher.Flush()

    return nil
}

// TaskStatusEvent is sent when task status changes
type TaskStatusEvent struct {
    TaskID string     `json:"id"`
    Status TaskStatus `json:"status"`
    Final  bool       `json:"final,omitempty"`
}

// TaskArtifactEvent is sent when new artifacts arrive
type TaskArtifactEvent struct {
    TaskID   string   `json:"id"`
    Artifact Artifact `json:"artifact"`
    Final    bool     `json:"final,omitempty"`
}

// StreamingTaskHandler manages SSE streams
func (h *TaskHandler) HandleTaskStream(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    taskID := vars["taskId"]

    // Set up SSE stream
    stream := NewStreamWriter(w)

    // Get task and subscribe to updates
    task, err := h.store.Get(taskID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    // Send initial status
    statusEvent := TaskStatusEvent{
        TaskID: task.ID,
        Status: task.Status,
    }

    data, _ := json.Marshal(statusEvent)
    stream.WriteEvent(SSEEvent{
        Event: "status",
        Data:  string(data),
    })

    // Set up context for cancellation
    ctx := r.Context()

    // Subscribe to task updates (implementation depends on your pub/sub)
    updates := h.subscribeToTask(taskID)
    defer h.unsubscribeFromTask(taskID)

    for {
        select {
        case update := <-updates:
            // Send update event
            data, _ := json.Marshal(update)
            stream.WriteEvent(SSEEvent{
                Event: string(update.Type),
                Data:  string(data),
            })

            if update.IsFinal {
                return
            }

        case <-ctx.Done():
            return
        }
    }
}
```

---

#### Task 3.3: Update Agent Card
**Assignee**: `api-designer`
**Estimated**: 2 hours
**Dependencies**: Day 2

**Files to Modify**:
- `internal/a2a/agent_card.go`

**Additions**:

```go
type AgentCard struct {
    Name               string              `json:"name"`
    Description        string              `json:"description"`
    URL                string              `json:"url"`
    Provider           *ProviderInfo       `json:"provider,omitempty"`
    Version            string              `json:"version"`
    DocumentationURL   string              `json:"documentationUrl,omitempty"`

    Capabilities       AgentCapabilities   `json:"capabilities"`
    Authentication     AuthenticationInfo  `json:"authentication"`
    DefaultInputModes  []string            `json:"defaultInputModes"`
    DefaultOutputModes []string            `json:"defaultOutputModes"`
    Skills             []Skill             `json:"skills,omitempty"`
}

type AgentCapabilities struct {
    Streaming         bool `json:"streaming"`
    PushNotifications bool `json:"pushNotifications"` // NEW
    StateTransition   bool `json:"stateTransition"`
}

type AuthenticationInfo struct {
    Schemes []string      `json:"schemes"` // "apiKey", "oauth2", "none"
    OAuth2  *OAuth2Config `json:"oauth2,omitempty"`
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
}
```

---

### Day 4: API Handlers & Testing
**Theme**: Complete the implementation

#### Task 4.1: Add New HTTP Endpoints
**Assignee**: `golang-pro`
**Estimated**: 3 hours
**Dependencies**: Days 1-3

**Files to Modify**:
- `internal/a2a/handlers.go` (add new handlers)
- `cmd/rad-gateway/main.go` (register routes)

**New Endpoints to Add**:

```go
// Router setup additions
router.HandleFunc("/a2a/tasks/{taskId}/pushNotifications",
    h.SubscribePushNotifications).Methods("POST")
router.HandleFunc("/a2a/tasks/{taskId}/pushNotifications",
    h.UnsubscribePushNotifications).Methods("DELETE")
router.HandleFunc("/a2a/tasks/{taskId}/cancel",
    h.CancelTask).Methods("POST")
router.HandleFunc("/a2a/tasks/{taskId}/history",
    h.GetTaskHistory).Methods("GET")
```

**Handler Implementations**:

```go
func (h *TaskHandler) SubscribePushNotifications(w http.ResponseWriter, r *http.Request) {
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

    if req.URL == "" {
        http.Error(w, "URL is required", http.StatusBadRequest)
        return
    }

    config := PushNotificationConfig{
        URL:     req.URL,
        Token:   req.Token,
        Headers: req.Headers,
    }

    if err := h.store.SubscribePushNotifications(taskID, config); err != nil {
        if errors.Is(err, ErrTaskNotFound) {
            http.Error(w, err.Error(), http.StatusNotFound)
            return
        }
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "subscribed",
    })
}

func (h *TaskHandler) CancelTask(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    taskID := vars["taskId"]

    task, err := h.store.Get(taskID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    // Can only cancel working or input-required tasks
    if task.Status.State != TaskStateWorking &&
       task.Status.State != TaskStateInputRequired {
        http.Error(w, "Task cannot be canceled in current state",
            http.StatusConflict)
        return
    }

    task.Status = TaskStatus{
        State:     TaskStateCanceled,
        Timestamp: time.Now(),
    }

    if err := h.store.Update(task); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Send push notification
    if h.pushManager != nil {
        h.pushManager.SendNotification(r.Context(),
            *task.PushConfig, task)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) GetTaskHistory(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    taskID := vars["taskId"]

    offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit == 0 || limit > 100 {
        limit = 50
    }

    history, err := h.store.GetHistory(taskID, offset, limit)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "taskId":   taskID,
        "messages": history,
        "offset":   offset,
        "limit":    limit,
    })
}
```

---

#### Task 4.2: Implement Authentication Middleware
**Assignee**: `golang-pro`
**Estimated**: 2 hours
**Dependencies**: Task 4.1

**Files to Create**:
- `internal/a2a/auth.go` (new)

**Implementation**:

```go
package a2a

import (
    "context"
    "net/http"
    "strings"
)

// AuthMiddleware provides A2A-specific authentication
type AuthMiddleware struct {
    apiKeyAuth func(key string) (bool, error)
}

func NewAuthMiddleware(apiKeyAuth func(string) (bool, error)) *AuthMiddleware {
    return &AuthMiddleware{
        apiKeyAuth: apiKeyAuth,
    }
}

func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Check X-API-Key header
        apiKey := r.Header.Get("X-API-Key")
        if apiKey != "" {
            valid, err := m.apiKeyAuth(apiKey)
            if valid && err == nil {
                next.ServeHTTP(w, r)
                return
            }
        }

        // Check Bearer token
        authHeader := r.Header.Get("Authorization")
        if strings.HasPrefix(authHeader, "Bearer ") {
            token := strings.TrimPrefix(authHeader, "Bearer ")
            valid, err := m.apiKeyAuth(token)
            if valid && err == nil {
                next.ServeHTTP(w, r)
                return
            }
        }

        http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
    })
}
```

---

#### Task 4.3: Write Integration Tests
**Assignee**: `qa-expert`
**Estimated**: 3 hours
**Dependencies**: Tasks 4.1-4.2

**Files to Create**:
- `tests/integration/a2a_protocol_test.go` (new)

**Test Implementation**:

```go
package integration

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestA2AProtocol_Compliance(t *testing.T) {
    // Setup test server with real handlers
    server := setupTestServer(t)
    defer server.Close()

    t.Run("task lifecycle", func(t *testing.T) {
        // Create task
        task := createTestTask(t, server)

        // Get task
        fetched := getTask(t, server, task.ID)
        assert.Equal(t, task.ID, fetched.ID)

        // Subscribe to push notifications
        subscribeResponse := subscribePushNotifications(t, server, task.ID)
        assert.Equal(t, "subscribed", subscribeResponse.Status)

        // Cancel task
        canceled := cancelTask(t, server, task.ID)
        assert.Equal(t, "canceled", canceled.Status.State)

        // Get history
        history := getTaskHistory(t, server, task.ID)
        assert.NotEmpty(t, history.Messages)
    })

    t.Run("push notification delivery", func(t *testing.T) {
        // Setup webhook receiver
        webhookReceived := make(chan *http.Request, 1)
        webhookServer := httptest.NewServer(http.HandlerFunc(
            func(w http.ResponseWriter, r *http.Request) {
                webhookReceived <- r
                w.WriteHeader(http.StatusOK)
            },
        ))
        defer webhookServer.Close()

        // Create task with push notification
        task := createTestTask(t, server)
        subscribePushNotifications(t, server, task.ID, webhookServer.URL)

        // Cancel task (triggers notification)
        cancelTask(t, server, task.ID)

        // Verify webhook received
        select {
        case req := <-webhookReceived:
            assert.Equal(t, "POST", req.Method)
            assert.NotEmpty(t, req.Header.Get("X-A2A-Signature"))
        default:
            t.Fatal("Webhook not received")
        }
    })

    t.Run("message part types", func(t *testing.T) {
        // Test text part
        textPart := map[string]any{
            "type": "text",
            "text": "Hello",
        }

        // Test file part
        filePart := map[string]any{
            "type": "file",
            "file": map[string]any{
                "name":     "test.pdf",
                "mimeType": "application/pdf",
            },
        }

        // Test function call part
        functionPart := map[string]any{
            "type": "function_call",
            "functionCall": map[string]any{
                "id":   "call-1",
                "name": "test_function",
                "args": map[string]any{"arg1": "value1"},
            },
        }

        for _, part := range []map[string]any{textPart, filePart, functionPart} {
            data, err := json.Marshal(part)
            require.NoError(t, err)

            // Verify can be unmarshaled
            var decoded map[string]any
            err = json.Unmarshal(data, &decoded)
            require.NoError(t, err)

            assert.Equal(t, part["type"], decoded["type"])
        }
    })
}

func TestA2AProtocol_Streaming(t *testing.T) {
    server := setupTestServer(t)
    defer server.Close()

    task := createTestTask(t, server)

    // Request SSE stream
    req, _ := http.NewRequest("GET",
        server.URL+"/a2a/tasks/"+task.ID+"/stream", nil)
    req.Header.Set("Accept", "text/event-stream")

    client := &http.Client{}
    resp, err := client.Do(req)
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

    // Read SSE events
    // ... validate SSE format
}
```

---

## Quality Gates

### End of Day 1
- [ ] Task model updated with all new fields
- [ ] Database migrations created and tested
- [ ] Task store interface updated
- [ ] All unit tests passing

### End of Day 2
- [ ] Push notification manager implemented
- [ ] Webhook delivery working with retry logic
- [ ] Task status changes trigger notifications
- [ ] Webhook logging in place

### End of Day 3
- [ ] All message part types implemented
- [ ] Polymorphic JSON unmarshaling working
- [ ] Streaming format updated to spec
- [ ] Agent card updated with capabilities

### End of Day 4
- [ ] All new API endpoints functional
- [ ] Authentication middleware in place
- [ ] Integration tests passing (>80% coverage)
- [ ] Documentation updated

---

## Daily Execution Commands

### Morning Routine (Each Agent)
```bash
# Pull latest
git checkout feature/a2a-protocol-compliance
git pull origin feature/a2a-protocol-compliance

# Run tests
go test ./internal/a2a/... -v

# Check build
go build ./...
```

### End of Day (Each Agent)
```bash
# Run full test suite
go test ./... -race
cd web && npm run build && cd ..

# Commit changes
git add .
git commit -m "feat(a2a): [task description]"
git push origin feature/a2a-protocol-compliance
```

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Database migration failure | Test migrations on staging DB first; keep rollback scripts |
| Breaking API changes | Maintain backward compatibility; version endpoints if needed |
| Webhook delivery failures | Implement retry logic with exponential backoff; log all attempts |
| Streaming format mismatch | Validate against A2A spec examples; test with reference client |
| Test coverage gaps | Require tests for all new code; enforce 80% minimum |

---

## Deliverables

1. **Code**:
   - `internal/a2a/task.go` - Updated task model
   - `internal/a2a/push_notifications.go` - Webhook system
   - `internal/a2a/message.go` - Extended part types
   - `internal/a2a/streaming.go` - SSE streaming
   - `internal/a2a/auth.go` - Authentication middleware
   - `cmd/migrate/migrations/000008_*.sql` - Database migrations

2. **Tests**:
   - Unit tests for all new components (>80% coverage)
   - Integration tests validating spec compliance
   - Streaming format validation tests

3. **Documentation**:
   - Updated API documentation
   - Webhook integration guide
   - A2A compliance checklist

---

**Sprint Start**: Upon approval
**Sprint End**: 4 days from start
**Integration**: Merge to main after review
