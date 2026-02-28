# MCP Bridge Implementation

## Overview

The MCP (Model Context Protocol) Bridge provides a standardized interface for tool registration and invocation. It enables AI models to access external capabilities through a unified API, abstracting provider-specific implementations.

**Key Features:**
- Tool registration with JSON Schema validation
- Synchronous and asynchronous tool execution
- Built-in tool library (echo, time, json_parse, chat)
- Gateway integration for LLM-based tool execution
- Resource management for persistent data access

---

## Tools

Tools are executable capabilities that agents can invoke. Each tool has a name, description, and input schema.

### Tool Structure

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique tool identifier |
| `description` | string | What the tool does |
| `inputSchema` | InputSchema | JSON Schema for parameters |

### Input Schema

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Usually "object" |
| `properties` | map[string]Property | Parameter definitions |
| `required` | []string | Required parameter names |

### Property

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Data type (string, number, boolean, etc.) |
| `description` | string | Parameter description |

### Registration

Tools can be registered programmatically:

```go
import "radgateway/internal/mcp"

bridge := mcp.NewBridge()

// Register a tool
tool := mcp.Tool{
    Name:        "weather_lookup",
    Description: "Get weather information for a location",
    InputSchema: mcp.InputSchema{
        Type: "object",
        Properties: map[string]mcp.Property{
            "location": {
                Type:        "string",
                Description: "City name or coordinates",
            },
            "units": {
                Type:        "string",
                Description: "celsius or fahrenheit",
            },
        },
        Required: []string{"location"},
    },
}

if err := bridge.RegisterTool(tool); err != nil {
    log.Fatal(err)
}

// Register handler
bridge.RegisterToolHandler("weather_lookup", func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    location := args["location"].(string)
    // Execute weather lookup...
    return map[string]interface{}{
        "temperature": 72,
        "conditions": "sunny",
    }, nil
})
```

### Built-in Tools

The MCP handler provides several built-in tools:

#### echo
Returns the input unchanged.

**Parameters:**
- `content` (string): Content to echo

**Example:**
```json
{
  "tool": "echo",
  "input": {
    "content": "Hello, World!"
  }
}
```

**Response:**
```json
{
  "success": true,
  "output": {
    "echo": { "content": "Hello, World!" },
    "message": "Echo response"
  }
}
```

#### time
Returns current time information.

**Parameters:** None

**Example:**
```json
{
  "tool": "time",
  "input": {}
}
```

**Response:**
```json
{
  "success": true,
  "output": {
    "timestamp": "2026-02-28T10:00:00Z",
    "unix": 1740736800
  }
}
```

#### json_parse
Parses a JSON string to structured data.

**Parameters:**
- `data` (string): JSON string to parse

**Example:**
```json
{
  "tool": "json_parse",
  "input": {
    "data": "{\"name\":\"test\",\"value\":123}"
  }
}
```

**Response:**
```json
{
  "success": true,
  "output": {
    "parsed": { "name": "test", "value": 123 },
    "type": "map[string]interface {}"
  }
}
```

#### chat
Execute LLM chat completion via the gateway.

**Parameters:**
- `content` (string): The prompt/message
- `model` (string, optional): Model to use (default: gpt-4o-mini)

**Example:**
```json
{
  "tool": "chat",
  "input": {
    "content": "Summarize this text",
    "model": "gpt-4o-mini"
  }
}
```

**Response:**
```json
{
  "success": true,
  "output": {
    "model": "gpt-4o-mini",
    "provider": "openai",
    "result": { ... },
    "usage": { "prompt_tokens": 10, "completion_tokens": 20 }
  }
}
```

---

## Resources

Resources represent persistent data that can be accessed by tools and agents.

### Resource Structure

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique resource identifier |
| `description` | string | Resource description |
| `mimeType` | string | MIME type (e.g., "application/json") |

### Registration

```go
resource := mcp.Resource{
    Name:        "user_profile",
    Description: "User profile data",
    MIMEType:    "application/json",
}

bridge.RegisterResource(resource)
```

### Access Patterns

Resources can be accessed by tools during execution:

```go
// In a tool handler
resources := bridge.ListResources()
for _, r := range resources {
    if r.Name == "user_profile" {
        // Access resource data
    }
}
```

---

## Endpoints

### List Tools

```
GET /mcp/v1/tools/list
```

Returns all registered tools with their schemas.

**Response:**
```json
{
  "tools": [
    {
      "name": "echo",
      "description": "Echoes back the input",
      "parameters": {
        "type": "object",
        "properties": {
          "content": { "type": "string" }
        }
      }
    },
    {
      "name": "time",
      "description": "Returns current time information",
      "parameters": {
        "type": "object",
        "properties": {}
      }
    }
  ],
  "count": 2
}
```

### Invoke Tool

```
POST /mcp/v1/tools/invoke
```

Executes a tool with the provided input.

**Request:**
```json
{
  "tool": "echo",
  "input": {
    "content": "Hello"
  },
  "session": "session-123",
  "metadata": {}
}
```

**Response (Success):**
```json
{
  "success": true,
  "tool": "echo",
  "output": {
    "echo": { "content": "Hello" },
    "message": "Echo response"
  },
  "timestamp": "2026-02-28T10:00:00Z",
  "durationMs": 5,
  "session": "session-123"
}
```

**Response (Error):**
```json
{
  "success": false,
  "tool": "unknown_tool",
  "error": "tool not found: unknown_tool",
  "timestamp": "2026-02-28T10:00:00Z",
  "durationMs": 1,
  "session": "session-123"
}
```

### Stdio Endpoint

```
POST /mcp/v1/stdio
```

Alternative endpoint for tool invocation (same format as `/tools/invoke`).

### Health Check

```
GET /mcp/v1/health
```

Returns MCP service health status.

**Response:**
```json
{
  "status": "healthy",
  "service": "mcp",
  "executor": true,
  "timestamp": "2026-02-28T10:00:00Z"
}
```

---

## Examples

### Go Tool Definition and Handler

```go
package main

import (
    "context"
    "fmt"
    "radgateway/internal/mcp"
)

func main() {
    // Create bridge
    bridge := mcp.NewBridge()

    // Define a custom tool
    calculatorTool := mcp.Tool{
        Name:        "calculator",
        Description: "Perform mathematical calculations",
        InputSchema: mcp.InputSchema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "operation": {
                    Type:        "string",
                    Description: "add, subtract, multiply, divide",
                },
                "a": {
                    Type:        "number",
                    Description: "First operand",
                },
                "b": {
                    Type:        "number",
                    Description: "Second operand",
                },
            },
            Required: []string{"operation", "a", "b"},
        },
    }

    // Register tool
    if err := bridge.RegisterTool(calculatorTool); err != nil {
        panic(err)
    }

    // Register handler
    err := bridge.RegisterToolHandler("calculator", func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
        op := args["operation"].(string)
        a := args["a"].(float64)
        b := args["b"].(float64)

        var result float64
        switch op {
        case "add":
            result = a + b
        case "subtract":
            result = a - b
        case "multiply":
            result = a * b
        case "divide":
            if b == 0 {
                return nil, fmt.Errorf("division by zero")
            }
            result = a / b
        default:
            return nil, fmt.Errorf("unknown operation: %s", op)
        }

        return map[string]interface{}{
            "result": result,
            "operation": op,
            "operands": []float64{a, b},
        }, nil
    })
    if err != nil {
        panic(err)
    }

    // Execute tool
    ctx := context.Background()
    result, err := bridge.CallTool(ctx, "calculator", map[string]interface{}{
        "operation": "add",
        "a": 5,
        "b": 3,
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("Result: %+v\n", result)
    // Output: Result: map[operands:[5 3] operation:add result:8]
}
```

### HTTP API Usage

#### cURL Examples

**List Tools:**
```bash
curl http://localhost:8090/mcp/v1/tools/list \
  -H "Authorization: Bearer your-api-key"
```

**Invoke Tool:**
```bash
curl -X POST http://localhost:8090/mcp/v1/tools/invoke \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "tool": "json_parse",
    "input": {
      "data": "{\"key\": \"value\"}"
    },
    "session": "session-123"
  }'
```

**Check Health:**
```bash
curl http://localhost:8090/mcp/v1/health
```

### Python Client Example

```python
import requests
import json

class MCPClient:
    def __init__(self, base_url, api_key):
        self.base_url = base_url
        self.headers = {"Authorization": f"Bearer {api_key}"}

    def list_tools(self):
        resp = requests.get(
            f"{self.base_url}/mcp/v1/tools/list",
            headers=self.headers
        )
        resp.raise_for_status()
        return resp.json()

    def invoke_tool(self, tool_name, input_data, session=None):
        payload = {
            "tool": tool_name,
            "input": input_data,
        }
        if session:
            payload["session"] = session

        resp = requests.post(
            f"{self.base_url}/mcp/v1/tools/invoke",
            headers={**self.headers, "Content-Type": "application/json"},
            json=payload
        )
        resp.raise_for_status()
        return resp.json()

# Usage
client = MCPClient("http://localhost:8090", "your-api-key")

# List available tools
tools = client.list_tools()
print(f"Available tools: {tools['count']}")

# Invoke echo tool
result = client.invoke_tool("echo", {"content": "Hello"}, session="session-123")
print(f"Echo result: {result['output']}")

# Invoke calculator (custom tool)
result = client.invoke_tool("calculator", {
    "operation": "multiply",
    "a": 10,
    "b": 20
})
print(f"Calculation: {result['output']}")
```

### JavaScript/TypeScript Example

```typescript
interface MCPClient {
  baseUrl: string;
  apiKey: string;
}

interface ToolInvokeRequest {
  tool: string;
  input: Record<string, any>;
  session?: string;
  metadata?: Record<string, any>;
}

interface ToolInvokeResponse {
  success: boolean;
  tool: string;
  output?: Record<string, any>;
  error?: string;
  timestamp: string;
  durationMs: number;
  session?: string;
}

class MCPClient {
  constructor(private baseUrl: string, private apiKey: string) {}

  async listTools(): Promise<any> {
    const response = await fetch(`${this.baseUrl}/mcp/v1/tools/list`, {
      headers: { 'Authorization': `Bearer ${this.apiKey}` }
    });
    return response.json();
  }

  async invokeTool(request: ToolInvokeRequest): Promise<ToolInvokeResponse> {
    const response = await fetch(`${this.baseUrl}/mcp/v1/tools/invoke`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.apiKey}`
      },
      body: JSON.stringify(request)
    });
    return response.json();
  }
}

// Usage
const client = new MCPClient('http://localhost:8090', 'your-api-key');

async function main() {
  // List tools
  const tools = await client.listTools();
  console.log('Available tools:', tools);

  // Invoke echo
  const result = await client.invokeTool({
    tool: 'echo',
    input: { content: 'Hello from TypeScript!' },
    session: 'session-123'
  });
  console.log('Echo result:', result.output);
}

main();
```

---

## Error Responses

### HTTP Status Codes

| Status | Description |
|--------|-------------|
| 200 | Success |
| 400 | Bad Request - Invalid input |
| 405 | Method Not Allowed |
| 500 | Internal Server Error |

### Error Format

```json
{
  "success": false,
  "tool": "tool_name",
  "error": "Error description",
  "timestamp": "2026-02-28T10:00:00Z",
  "durationMs": 0,
  "session": "session-123"
}
```

### Common Errors

| Error | Description | Resolution |
|-------|-------------|------------|
| `tool is required` | Missing tool name in request | Include tool field |
| `tool not found: {name}` | Tool not registered | Check tool name |
| `no handler registered for tool: {name}` | Tool exists but has no handler | Register handler |
| `tool execution failed` | Handler returned error | Check tool implementation |

---

## Usage Notes

1. **Tool Registration Order**: Tools must be registered before their handlers. Attempting to register a handler for a non-existent tool returns an error.

2. **Handler Safety**: Tool handlers should be idempotent and handle errors gracefully. Panics in handlers may crash the server.

3. **Gateway Integration**: When using `NewHandlerWithGateway`, the `chat` tool executes LLM completions through the gateway. Requires gateway configuration.

4. **Session Tracking**: The `session` field in requests can be used to correlate tool invocations with specific user sessions or conversations.

5. **Schema Validation**: Input validation is the responsibility of the tool handler. The MCP bridge does not validate inputs against the JSON Schema.

6. **Resource Management**: Resources are currently stored in-memory. For persistent resources, integrate with a database.

7. **Concurrent Access**: The Bridge uses read-write locks for thread-safe concurrent access. Handlers may be called concurrently.

8. **Custom Tools**: When adding custom tools, follow naming conventions (lowercase, snake_case) and provide clear descriptions for LLM compatibility.
