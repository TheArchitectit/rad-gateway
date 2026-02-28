# AG-UI Protocol Implementation

## Overview

AG-UI (Agent-User Interface) provides real-time event streaming for agent UI updates. It uses Server-Sent Events (SSE) to push live updates about agent runs, messages, tool calls, and state changes to connected clients.

**Key Features:**
- Real-time event streaming via SSE
- Event filtering by agent and thread
- Support for run lifecycle events
- Message delta updates
- Tool call and result tracking
- State snapshots and deltas

---

## Event Types

| Event Type | Description |
|------------|-------------|
| `run.start` | Agent run has started |
| `run.complete` | Agent run completed successfully |
| `run.error` | Agent run encountered an error |
| `message.delta` | Incremental message update |
| `tool.call` | Tool is being invoked |
| `tool.result` | Tool execution completed |
| `state.snapshot` | Full state snapshot |
| `state.delta` | Incremental state update |

### Event Type Details

#### Run Events

**run.start**
```json
{
  "type": "run.start",
  "run_id": "run-123",
  "agent_id": "agent-456",
  "thread_id": "thread-789",
  "timestamp": "2026-02-28T10:00:00Z",
  "data": {
    "status": "running",
    "message": "Agent run started"
  }
}
```

**run.complete**
```json
{
  "type": "run.complete",
  "run_id": "run-123",
  "agent_id": "agent-456",
  "thread_id": "thread-789",
  "timestamp": "2026-02-28T10:00:05Z",
  "data": {
    "status": "completed",
    "duration_ms": 5000
  }
}
```

**run.error**
```json
{
  "type": "run.error",
  "run_id": "run-123",
  "agent_id": "agent-456",
  "thread_id": "thread-789",
  "timestamp": "2026-02-28T10:00:03Z",
  "data": {
    "error": "Rate limit exceeded",
    "code": "rate_limited"
  }
}
```

#### Message Events

**message.delta**
```json
{
  "type": "message.delta",
  "run_id": "run-123",
  "agent_id": "agent-456",
  "thread_id": "thread-789",
  "timestamp": "2026-02-28T10:00:01Z",
  "data": {
    "message_id": "msg-001",
    "delta": {
      "content": "Hello"
    }
  }
}
```

#### Tool Events

**tool.call**
```json
{
  "type": "tool.call",
  "run_id": "run-123",
  "agent_id": "agent-456",
  "thread_id": "thread-789",
  "timestamp": "2026-02-28T10:00:02Z",
  "data": {
    "tool_call_id": "call-001",
    "tool": "weather_api",
    "arguments": {
      "location": "San Francisco",
      "units": "celsius"
    }
  }
}
```

**tool.result**
```json
{
  "type": "tool.result",
  "run_id": "run-123",
  "agent_id": "agent-456",
  "thread_id": "thread-789",
  "timestamp": "2026-02-28T10:00:04Z",
  "data": {
    "tool_call_id": "call-001",
    "tool": "weather_api",
    "result": {
      "temperature": 18,
      "conditions": "partly cloudy"
    }
  }
}
```

#### State Events

**state.snapshot**
```json
{
  "type": "state.snapshot",
  "run_id": "run-123",
  "agent_id": "agent-456",
  "thread_id": "thread-789",
  "timestamp": "2026-02-28T10:00:00Z",
  "data": {
    "run_id": "run-123",
    "status": "running",
    "messages": [...],
    "tool_calls": [...],
    "state": {}
  }
}
```

**state.delta**
```json
{
  "type": "state.delta",
  "run_id": "run-123",
  "agent_id": "agent-456",
  "thread_id": "thread-789",
  "timestamp": "2026-02-28T10:00:01Z",
  "data": {
    "messages_added": [...],
    "tool_calls_updated": [...]
  }
}
```

---

## SSE Endpoint

### Connection URL

```
GET /v1/agents/{agentId}/stream?threadId={threadId}
```

### Required Headers

| Header | Value | Description |
|--------|-------|-------------|
| `Authorization` | `Bearer {token}` | Authentication token |
| `Accept` | `text/event-stream` | SSE content type |

### Response Headers

| Header | Value | Description |
|--------|-------|-------------|
| `Content-Type` | `text/event-stream` | SSE stream format |
| `Cache-Control` | `no-cache` | Disable caching |
| `Connection` | `keep-alive` | Keep connection open |
| `X-Accel-Buffering` | `no` | Disable proxy buffering |

### Path Parameters

| Parameter | Description |
|-----------|-------------|
| `agentId` | The agent to subscribe to |

### Query Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `threadId` | Yes | Thread/conversation ID |

---

## Event Structure

### Base Event Schema

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Event type (e.g., `run.start`) |
| `run_id` | string | Unique run identifier |
| `agent_id` | string | Agent identifier |
| `thread_id` | string | Thread identifier |
| `timestamp` | string | ISO 8601 timestamp (RFC3339Nano) |
| `data` | object | Event-specific payload |
| `metadata` | object | Optional contextual information |

### Data Payloads

#### RunState

```go
type RunState struct {
    RunID      string                 `json:"run_id"`
    AgentID    string                 `json:"agent_id"`
    ThreadID   string                 `json:"thread_id"`
    Status     string                 `json:"status"`
    Messages   []Message              `json:"messages,omitempty"`
    ToolCalls  []ToolCall             `json:"tool_calls,omitempty"`
    State      map[string]interface{} `json:"state,omitempty"`
}
```

#### Message

```go
type Message struct {
    ID        string    `json:"id"`
    Role      string    `json:"role"`
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
}
```

#### ToolCall

```go
type ToolCall struct {
    ID        string                 `json:"id"`
    Tool      string                 `json:"tool"`
    Arguments map[string]interface{} `json:"arguments,omitempty"`
    Result    interface{}            `json:"result,omitempty"`
}
```

---

## Reconnection Handling

### Connection Lifecycle

1. Client connects to SSE endpoint
2. Server sends initial `state.snapshot` event
3. Server streams events as they occur
4. Connection may close due to:
   - Client disconnect
   - Server error
   - Timeout

### Reconnection Strategy

When a connection is lost, clients should:

1. **Wait briefly** (exponential backoff: 1s, 2s, 4s, 8s, max 30s)
2. **Reconnect** to the same endpoint
3. **Receive new state snapshot** upon reconnection
4. **Resume processing** from current state

**Note:** There is no `Last-Event-ID` support currently. Clients receive a full state snapshot upon reconnection.

### Client Buffering

The server maintains a per-client event buffer (100 events). If a client's buffer fills up:
- New events are dropped
- A warning is logged
- Client should reconnect to resync state

---

## Examples

### JavaScript EventSource Client

```javascript
const agentId = 'agent-456';
const threadId = 'thread-789';
const apiKey = 'your-api-key';

const eventSource = new EventSource(
  `http://localhost:8090/v1/agents/${agentId}/stream?threadId=${threadId}`,
  {
    headers: {
      'Authorization': `Bearer ${apiKey}`
    }
  }
);

// Connection opened
eventSource.onopen = () => {
  console.log('Connected to AG-UI stream');
};

// Handle messages
eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Received event:', data);

  switch (data.type) {
    case 'run.start':
      console.log('Run started:', data.run_id);
      break;
    case 'run.complete':
      console.log('Run completed:', data.data);
      eventSource.close();
      break;
    case 'run.error':
      console.error('Run error:', data.data.error);
      break;
    case 'message.delta':
      appendMessage(data.data);
      break;
    case 'tool.call':
      showToolCall(data.data);
      break;
    case 'tool.result':
      showToolResult(data.data);
      break;
    case 'state.snapshot':
      updateFullState(data.data);
      break;
  }
};

// Handle errors
eventSource.onerror = (error) => {
  console.error('SSE error:', error);
  // Implement reconnection logic here
};

// Clean up
function disconnect() {
  eventSource.close();
}

// Helper functions
function appendMessage(data) {
  const messageEl = document.getElementById(`msg-${data.message_id}`);
  if (messageEl) {
    messageEl.textContent += data.delta.content;
  }
}

function showToolCall(data) {
  console.log(`Tool ${data.tool} called with:`, data.arguments);
}

function showToolResult(data) {
  console.log(`Tool ${data.tool} returned:`, data.result);
}

function updateFullState(state) {
  console.log('State updated:', state);
  // Refresh UI with new state
}
```

### Python Client Example

```python
import json
import requests

def connect_agui_stream(agent_id, thread_id, api_key):
    url = f"http://localhost:8090/v1/agents/{agent_id}/stream"
    params = {"threadId": thread_id}
    headers = {
        "Authorization": f"Bearer {api_key}",
        "Accept": "text/event-stream"
    }

    response = requests.get(url, params=params, headers=headers, stream=True)
    response.raise_for_status()

    for line in response.iter_lines():
        if line:
            # SSE format: "data: {json}\n\n"
            if line.startswith(b"data: "):
                data = json.loads(line[6:])
                handle_event(data)

def handle_event(event):
    event_type = event.get("type")
    print(f"Event: {event_type}")

    if event_type == "run.start":
        print(f"Run started: {event['run_id']}")
    elif event_type == "run.complete":
        print(f"Run completed: {event['data']}")
    elif event_type == "message.delta":
        print(f"Message delta: {event['data']['delta']}")
    elif event_type == "tool.call":
        print(f"Tool called: {event['data']['tool']}")
    elif event_type == "tool.result":
        print(f"Tool result: {event['data']['result']}")

# Usage
connect_agui_stream("agent-456", "thread-789", "your-api-key")
```

### Go Client Example

```go
package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
)

type Event struct {
    Type      string                 `json:"type"`
    RunID     string                 `json:"run_id"`
    AgentID   string                 `json:"agent_id"`
    ThreadID  string                 `json:"thread_id"`
    Timestamp string                 `json:"timestamp"`
    Data      map[string]interface{} `json:"data,omitempty"`
}

func main() {
    agentID := "agent-456"
    threadID := "thread-789"
    apiKey := "your-api-key"

    req, _ := http.NewRequest("GET",
        fmt.Sprintf("http://localhost:8090/v1/agents/%s/stream?threadId=%s", agentID, threadID),
        nil)
    req.Header.Set("Authorization", "Bearer "+apiKey)
    req.Header.Set("Accept", "text/event-stream")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "data: ") {
            var event Event
            if err := json.Unmarshal([]byte(line[6:]), &event); err != nil {
                fmt.Printf("Error parsing: %v\n", err)
                continue
            }
            fmt.Printf("Event: %s - Run: %s\n", event.Type, event.RunID)
        }
    }
}
```

### cURL Example

```bash
# Connect to SSE stream
curl -N http://localhost:8090/v1/agents/agent-456/stream?threadId=thread-789 \
  -H "Authorization: Bearer your-api-key" \
  -H "Accept: text/event-stream"
```

---

## Error Responses

### Connection Errors

| Status | Description | Resolution |
|--------|-------------|------------|
| 400 | Missing threadId | Add threadId query parameter |
| 401 | Unauthorized | Check API key |
| 405 | Method Not Allowed | Use GET method |
| 500 | Streaming not supported | Check server configuration |

### Error Format

```json
{
  "error": "error message"
}
```

---

## Usage Notes

1. **Connection Limits**: Each client connection maintains a goroutine. Monitor client count for resource management.

2. **Buffer Management**: The 100-event buffer may drop events during high throughput. Design clients to handle missing events via periodic state snapshots.

3. **Thread Isolation**: Events are filtered by both `agentId` and `threadId`. Clients only receive events for their subscribed thread.

4. **Authentication**: The stream requires valid authentication. Expired tokens will result in immediate connection termination.

5. **Cleanup**: Always close connections when no longer needed:
   - JavaScript: `eventSource.close()`
   - Python: `response.close()`
   - Go: `resp.Body.Close()`

6. **Reconnection**: Implement exponential backoff for reconnection to avoid overwhelming the server.

7. **State Sync**: On reconnection, clients receive a fresh `state.snapshot`. Previous state should be replaced, not merged.
