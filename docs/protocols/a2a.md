# A2A Protocol Implementation

## Overview

The A2A (Agent-to-Agent) protocol enables autonomous agents to communicate and collaborate within RAD Gateway. This implementation provides task lifecycle management, agent discovery via Agent Cards, and real-time streaming updates via Server-Sent Events (SSE).

**Key Features:**
- Agent discovery via well-known endpoint
- Task lifecycle with state machine
- Model Card management for provider capabilities
- Streaming task updates via SSE

---

## Agent Card

The Agent Card describes an agent's capabilities, skills, and authentication requirements. It follows the A2A protocol specification for agent discovery.

### Structure

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Display name of the agent |
| `description` | string | What the agent does |
| `url` | string | Endpoint URL for the agent |
| `version` | string | Agent version |
| `capabilities` | Capabilities | What the agent can do |
| `skills` | []Skill | Capabilities offered by the agent |
| `authentication` | AuthInfo | Supported authentication schemes |

### Capabilities

| Field | Type | Description |
|-------|------|-------------|
| `streaming` | bool | Supports streaming responses |
| `pushNotifications` | bool | Supports push notifications |
| `stateTransitionHistory` | bool | Tracks state transitions |

### Skills

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique skill identifier |
| `name` | string | Display name |
| `description` | string | What the skill does |
| `tags` | []string | Categorization labels |
| `examples` | []string | Sample use cases |
| `input` | SkillSchema | Input schema |
| `output` | SkillSchema | Output schema |

### Discovery Endpoint

```
GET /.well-known/agent.json
```

Returns the gateway's Agent Card for discovery by other agents.

### Example Agent Card

```json
{
  "name": "RAD Gateway",
  "description": "AI API Gateway with A2A protocol support",
  "url": "https://gateway.example.com",
  "version": "0.1.0",
  "capabilities": {
    "streaming": true,
    "pushNotifications": false,
    "stateTransitionHistory": true
  },
  "skills": [
    {
      "id": "chat-completion",
      "name": "Chat Completion",
      "description": "Generate chat responses via LLM providers",
      "tags": ["llm", "chat"],
      "examples": ["Summarize this text", "Answer this question"],
      "input": {
        "type": "object",
        "properties": {
          "messages": { "type": "array" },
          "model": { "type": "string" }
        },
        "required": ["messages"]
      }
    }
  ],
  "authentication": {
    "schemes": ["Bearer", "APIKey"]
  }
}
```

---

## Task Lifecycle

Tasks represent units of work submitted to agents. Each task progresses through a defined state machine.

### Task States

```
submitted -> working -> completed
     |           |
     |           v
     +----> input-required
     |           |
     |           v
     +----> failed / cancelled
```

| State | Description |
|-------|-------------|
| `submitted` | Task created, awaiting processing |
| `working` | Task is being processed |
| `input-required` | Agent needs additional input |
| `completed` | Task finished successfully |
| `failed` | Task failed with error |
| `cancelled` | Task was cancelled by user |

### State Transitions

```
submitted: -> working, cancelled
working:   -> completed, failed, input-required, cancelled
input-required: -> working, cancelled
```

Terminal states: `completed`, `failed`, `cancelled`

### Task Structure

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique task identifier |
| `status` | TaskState | Current state |
| `sessionId` | string | Session grouping ID |
| `message` | Message | Initial task message |
| `artifacts` | []Artifact | Task outputs |
| `history` | []Message | Message history |
| `metadata` | object | Additional metadata |
| `createdAt` | timestamp | Creation time |
| `updatedAt` | timestamp | Last update time |

### Endpoints

#### Create Task (Sync)

```
POST /a2a/tasks/send
```

Creates a task and waits for completion (synchronous).

**Request:**
```json
{
  "sessionId": "session-123",
  "message": {
    "role": "user",
    "content": "Summarize the quarterly report"
  },
  "metadata": {
    "model": "gpt-4o-mini",
    "api_type": "chat"
  }
}
```

**Response:**
```json
{
  "task": {
    "id": "task-456",
    "status": "completed",
    "sessionId": "session-123",
    "message": { ... },
    "artifacts": [...],
    "createdAt": "2026-02-28T10:00:00Z",
    "updatedAt": "2026-02-28T10:00:05Z"
  }
}
```

#### Create Task (Streaming)

```
POST /a2a/tasks/sendSubscribe
```

Creates a task and streams progress via SSE.

**Request:** Same as sync endpoint

**Response:** SSE stream with events:

```
data: {"type":"status","taskId":"task-456","status":"submitted","timestamp":"2026-02-28T10:00:00Z"}

data: {"type":"status","taskId":"task-456","status":"working","timestamp":"2026-02-28T10:00:01Z"}

data: {"type":"artifact","taskId":"task-456","artifact":{...},"timestamp":"2026-02-28T10:00:05Z"}

data: {"type":"completed","taskId":"task-456","status":"completed","timestamp":"2026-02-28T10:00:05Z"}
```

#### Get Task

```
GET /a2a/tasks/{taskId}
```

Retrieves task details by ID.

**Response:**
```json
{
  "task": {
    "id": "task-456",
    "status": "completed",
    ...
  }
}
```

#### Cancel Task

```
POST /a2a/tasks/cancel
```

Cancels a task that is not in a terminal state.

**Request:**
```json
{
  "taskId": "task-456"
}
```

**Response:** Returns the updated task

---

## Model Cards

Model Cards define provider-specific model capabilities within the A2A framework.

### Endpoints

#### List Model Cards

```
GET /a2a/model-cards?workspace_id={workspaceId}
```

**Response:**
```json
{
  "items": [...],
  "total": 10,
  "limit": 10,
  "offset": 0
}
```

#### Create Model Card

```
POST /a2a/model-cards
```

**Request:**
```json
{
  "workspaceId": "ws-123",
  "name": "GPT-4 Turbo",
  "slug": "gpt-4-turbo",
  "description": "OpenAI GPT-4 Turbo model",
  "card": {
    "schemaVersion": "1.0",
    "name": "gpt-4-turbo",
    "capabilities": [
      {"type": "streaming", "enabled": true},
      {"type": "vision", "enabled": true}
    ],
    "pricing": {
      "inputPricePerToken": 0.00001,
      "outputPricePerToken": 0.00003,
      "currency": "USD"
    }
  }
}
```

#### Get Model Card

```
GET /a2a/model-cards/{id}
```

#### Update Model Card

```
PUT /a2a/model-cards/{id}
```

#### Delete Model Card

```
DELETE /a2a/model-cards/{id}
```

---

## Examples

### Go Client Example

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type SendTaskRequest struct {
    SessionID string `json:"sessionId"`
    Message   struct {
        Role    string `json:"role"`
        Content string `json:"content"`
    } `json:"message"`
}

func main() {
    // Create a task
    req := SendTaskRequest{
        SessionID: "session-123",
        Message: struct {
            Role    string `json:"role"`
            Content string `json:"content"`
        }{
            Role:    "user",
            Content: "Hello, agent!",
        },
    }

    data, _ := json.Marshal(req)
    resp, err := http.Post(
        "http://localhost:8090/a2a/tasks/send",
        "application/json",
        bytes.NewBuffer(data),
    )
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    fmt.Printf("Task created: %+v\n", result)
}
```

### cURL Examples

**Discover Agent:**
```bash
curl http://localhost:8090/.well-known/agent.json
```

**Create Task:**
```bash
curl -X POST http://localhost:8090/a2a/tasks/send \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "sessionId": "session-123",
    "message": {
      "role": "user",
      "content": "Generate a summary"
    }
  }'
```

**Get Task:**
```bash
curl http://localhost:8090/a2a/tasks/task-456 \
  -H "Authorization: Bearer your-api-key"
```

**Cancel Task:**
```bash
curl -X POST http://localhost:8090/a2a/tasks/cancel \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{"taskId": "task-456"}'
```

---

## Error Responses

### HTTP Status Codes

| Status | Description |
|--------|-------------|
| 400 | Bad Request - Invalid input |
| 404 | Not Found - Task or resource not found |
| 405 | Method Not Allowed |
| 409 | Conflict - Invalid state transition |
| 500 | Internal Server Error |
| 503 | Service Unavailable - Task store not configured |

### Error Format

```json
{
  "error": "error message"
}
```

---

## Usage Notes

1. **Authentication**: All endpoints (except `/.well-known/agent.json`) require authentication via:
   - `Authorization: Bearer {token}` header
   - `x-api-key: {token}` header

2. **Session Management**: Tasks are grouped by `sessionId` for conversational context

3. **Task Expiration**: Tasks may have an `expiresAt` field for automatic cleanup

4. **Streaming**: Use `sendSubscribe` for long-running tasks to receive real-time updates

5. **State Validation**: State transitions are validated; invalid transitions return 409 Conflict
