# Anthropic Adapter Design Document

## Overview

This document describes the architecture for the Anthropic provider adapter in RAD Gateway. The adapter enables RAD Gateway to communicate with Anthropic's Claude API by translating between the OpenAI-compatible format used internally by RAD Gateway and Anthropic's Messages API format.

**Status:** Design Phase
**Target:** RAD Gateway Alpha
**Author:** Architecture Team
**Last Updated:** 2026-02-16

---

## Architecture Overview

### Design Philosophy

The Anthropic adapter follows RAD Gateway's established adapter pattern, implementing bidirectional transformation between:
- **Internal Format:** OpenAI-compatible format (used by RAD Gateway)
- **Anthropic Format:** Messages API format (used by Claude API)

This approach maintains consistency across the gateway while accommodating Anthropic's unique API characteristics.

### Component Structure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ANTHROPIC ADAPTER                                   │
│                                                                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐    │
│  │   Anthropic     │  │   Anthropic     │  │   Anthropic             │    │
│  │   Adapter       │──│   Request       │──│   Response              │    │
│  │   (Entry Point) │  │   Transformer   │  │   Transformer           │    │
│  └────────┬────────┘  └─────────────────┘  └─────────────────────────┘    │
│           │                                                                 │
│           │                    ┌─────────────────┐                       │
│           └────────────────────│   Anthropic     │                       │
│                                │   Stream        │                       │
│                                │   Transformer   │                       │
│                                └─────────────────┘                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                      ANTHROPIC MESSAGES API                                 │
│                                                                             │
│  Endpoint: POST https://api.anthropic.com/v1/messages                       │
│  Auth: x-api-key header                                                     │
│  Version: anthropic-version: 2023-06-01                                   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### File Structure

```
internal/provider/anthropic/
├── adapter.go           # Main adapter implementation
├── adapter_test.go      # Unit tests
├── transformer.go       # Request/response transformation
├── transformer_test.go  # Transformer tests
└── streaming.go         # Streaming event handling (optional separation)
```

---

## Key Differences from OpenAI Adapter

| Aspect | OpenAI | Anthropic |
|--------|--------|-----------|
| **Endpoint** | `/v1/chat/completions` | `/v1/messages` |
| **Auth Header** | `Authorization: Bearer <key>` | `x-api-key: <key>` |
| **Version Header** | None required | `anthropic-version: 2023-06-01` |
| **System Message** | Included in `messages` array | Separate `system` field |
| **Message Roles** | `system`, `user`, `assistant` | `user`, `assistant` (system separate) |
| **Required Params** | `model`, `messages` | `model`, `messages`, `max_tokens` |
| **Streaming Format** | Simple delta chunks | Content blocks with events |
| **Response Structure** | `choices[].message.content` | `content[].text` |
| **Usage Reporting** | Always included | Included in `message_stop` event (streaming) |

---

## Request Transformation (OpenAI → Anthropic)

### Transformation Pipeline

```
┌────────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│ OpenAI Format  │───▶│  AnthropicRequest    │───▶│ Anthropic API   │
│ (Internal)     │    │  Transformer         │    │ Format          │
└────────────────┘    └──────────────────────┘    └─────────────────┘
```

### OpenAI Request Format (Internal)

```json
{
  "model": "claude-3-5-sonnet-20241022",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello, Claude!"}
  ],
  "stream": false,
  "temperature": 0.7,
  "max_tokens": 1024
}
```

### Anthropic Request Format (Output)

```json
{
  "model": "claude-3-5-sonnet-20241022",
  "system": "You are a helpful assistant.",
  "messages": [
    {"role": "user", "content": "Hello, Claude!"}
  ],
  "stream": false,
  "temperature": 0.7,
  "max_tokens": 1024
}
```

### Transformation Logic

```go
// RequestTransformation performs OpenAI → Anthropic conversion
type AnthropicRequestTransformer struct {
    config ProviderConfig
}

func (t *AnthropicRequestTransformer) Transform(body io.Reader) (io.Reader, error) {
    // 1. Parse OpenAI format
    var openaiReq OpenAIChatRequest
    if err := json.Unmarshal(data, &openaiReq); err != nil {
        return nil, err
    }

    // 2. Initialize Anthropic request
    anthropicReq := AnthropicMessageRequest{
        Model:    openaiReq.Model,
        Stream:   openaiReq.Stream,
        MaxTokens: openaiReq.MaxTokens,
    }

    // 3. Extract system message from messages array
    var messages []OpenAIMessage
    for _, msg := range openaiReq.Messages {
        if msg.Role == "system" {
            anthropicReq.System = msg.Content
        } else {
            // Map roles: OpenAI "assistant" → Anthropic "assistant"
            // OpenAI "user" → Anthropic "user"
            messages = append(messages, AnthropicMessage{
                Role:    msg.Role,
                Content: msg.Content,
            })
        }
    }
    anthropicReq.Messages = messages

    // 4. Set defaults for required Anthropic fields
    if anthropicReq.MaxTokens == 0 {
        anthropicReq.MaxTokens = 4096  // Anthropic requires this
    }

    // 5. Map optional parameters
    if openaiReq.Temperature != nil {
        anthropicReq.Temperature = openaiReq.Temperature
    }
    if openaiReq.TopP != nil {
        anthropicReq.TopP = openaiReq.TopP
    }
    if openaiReq.Stop != nil {
        anthropicReq.StopSequences = openaiReq.Stop
    }

    // 6. Serialize and return
    return json.Marshal(anthropicReq)
}
```

### Anthropic Request Types

```go
// AnthropicMessageRequest represents the /v1/messages API request
type AnthropicMessageRequest struct {
    Model         string               `json:"model"`
    System        string               `json:"system,omitempty"`
    Messages      []AnthropicMessage   `json:"messages"`
    MaxTokens     int                  `json:"max_tokens"`
    Metadata      *AnthropicMetadata   `json:"metadata,omitempty"`
    StopSequences []string             `json:"stop_sequences,omitempty"`
    Stream        bool                 `json:"stream,omitempty"`
    Temperature   *float64             `json:"temperature,omitempty"`
    TopP          *float64             `json:"top_p,omitempty"`
    TopK          *int                 `json:"top_k,omitempty"`
}

type AnthropicMessage struct {
    Role    string `json:"role"` // "user" or "assistant"
    Content string `json:"content"`
}

type AnthropicMetadata struct {
    UserID string `json:"user_id,omitempty"`
}
```

---

## Response Transformation (Anthropic → OpenAI)

### Non-Streaming Response Transformation

```
┌─────────────────┐    ┌──────────────────────┐    ┌────────────────┐
│ Anthropic API   │───▶│ AnthropicResponse    │───▶│ OpenAI Format  │
│ Response        │    │ Transformer          │    │ (Internal)     │
└─────────────────┘    └──────────────────────┘    └────────────────┘
```

### Anthropic Response Format (Input)

```json
{
  "id": "msg_01XgVYxVqW32TYn5Ts4RYRPW",
  "type": "message",
  "role": "assistant",
  "model": "claude-3-5-sonnet-20241022",
  "content": [
    {
      "type": "text",
      "text": "Hello! How can I help you today?"
    }
  ],
  "stop_reason": "end_turn",
  "stop_sequence": null,
  "usage": {
    "input_tokens": 12,
    "output_tokens": 9
  }
}
```

### OpenAI Response Format (Output)

```json
{
  "id": "msg_01XgVYxVqW32TYn5Ts4RYRPW",
  "object": "chat.completion",
  "created": 1704067200,
  "model": "claude-3-5-sonnet-20241022",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 12,
    "completion_tokens": 9,
    "total_tokens": 21
  }
}
```

### Transformation Logic

```go
func (t *AnthropicResponseTransformer) Transform(body io.Reader) (io.Reader, error) {
    // 1. Parse Anthropic response
    var anthropicResp AnthropicMessageResponse
    if err := json.Unmarshal(data, &anthropicResp); err != nil {
        return nil, err
    }

    // 2. Extract content from content blocks
    var content string
    for _, block := range anthropicResp.Content {
        if block.Type == "text" {
            content += block.Text
        }
        // Note: Other content types (tool_use, tool_result) handled separately
    }

    // 3. Map stop_reason to OpenAI finish_reason
    finishReason := mapStopReason(anthropicResp.StopReason)

    // 4. Build OpenAI-compatible response
    openaiResp := OpenAIChatResponse{
        ID:      anthropicResp.ID,
        Object:  "chat.completion",
        Created: time.Now().Unix(),
        Model:   anthropicResp.Model,
        Choices: []OpenAIChoice{
            {
                Index: 0,
                Message: OpenAIMessage{
                    Role:    "assistant",
                    Content: content,
                },
                FinishReason: finishReason,
            },
        },
        Usage: OpenAIUsage{
            PromptTokens:     anthropicResp.Usage.InputTokens,
            CompletionTokens: anthropicResp.Usage.OutputTokens,
            TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
        },
    }

    return json.Marshal(openaiResp)
}

// Stop reason mapping
func mapStopReason(anthropicReason string) string {
    switch anthropicReason {
    case "end_turn":
        return "stop"
    case "max_tokens":
        return "length"
    case "stop_sequence":
        return "stop"
    default:
        return "stop"
    }
}
```

### Anthropic Response Types

```go
// AnthropicMessageResponse represents the /v1/messages API response
type AnthropicMessageResponse struct {
    ID           string                   `json:"id"`
    Type         string                   `json:"type"` // "message"
    Role         string                   `json:"role"` // "assistant"
    Model        string                   `json:"model"`
    Content      []AnthropicContentBlock  `json:"content"`
    StopReason   string                   `json:"stop_reason"`   // "end_turn", "max_tokens", "stop_sequence"
    StopSequence *string                  `json:"stop_sequence"`
    Usage        AnthropicUsage           `json:"usage"`
}

type AnthropicContentBlock struct {
    Type string `json:"type"` // "text", "tool_use", "tool_result"
    Text string `json:"text,omitempty"`
    // Additional fields for tool_use/tool_result omitted for brevity
}

type AnthropicUsage struct {
    InputTokens  int `json:"input_tokens"`
    OutputTokens int `json:"output_tokens"`
}
```

---

## Streaming Event Handling

### Anthropic Streaming Architecture

Anthropic uses a sophisticated event-driven streaming format with distinct event types:

```
┌────────────────────────────────────────────────────────────────────────────┐
│                    ANTHROPIC STREAMING EVENT FLOW                          │
│                                                                            │
│  event: message_start                                                      │
│  data: {"type":"message_start","message":{"id":"msg_...",...}}             │
│                                                                            │
│  event: content_block_start                                                │
│  data: {"type":"content_block_start","index":0,"content_block":{...}}    │
│                                                                            │
│  event: content_block_delta                                                │
│  data: {"type":"content_block_delta","index":0,"delta":{...}}              │
│       │                                                                    │
│       ├── text: "Hello"                                                    │
│       ├── text: " there"                                                   │
│       └── text: "!"                                                        │
│                                                                            │
│  event: content_block_stop                                                 │
│  data: {"type":"content_block_stop","index":0}                           │
│                                                                            │
│  event: message_delta                                                      │
│  data: {"type":"message_delta","delta":{"stop_reason":"end_turn",...}}   │
│                                                                            │
│  event: message_stop                                                       │
│  data: {"type":"message_stop"}                                             │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

### Event Types Reference

| Event Type | Description | Transforms To |
|------------|-------------|---------------|
| `message_start` | Initial message metadata | OpenAI `id`, `model`, `created` |
| `content_block_start` | Start of new content block | Ignored (state tracking) |
| `content_block_delta` | Text delta within block | OpenAI `delta.content` |
| `content_block_stop` | End of content block | Ignored |
| `message_delta` | Message-level updates (stop_reason) | OpenAI `finish_reason` |
| `message_stop` | Stream complete | OpenAI `[DONE]` marker |
| `ping` | Keep-alive | Ignored |
| `error` | Error occurred | Error response |

### Streaming Transformation Logic

```go
// AnthropicStreamTransformer handles SSE event transformation
type AnthropicStreamTransformer struct {
    messageID    string
    model        string
    created      int64
    index        int
    buffer       strings.Builder
}

func (t *AnthropicStreamTransformer) TransformChunk(chunk []byte) ([]byte, error) {
    // Parse SSE event
    event, data := parseSSEEvent(chunk)

    switch event {
    case "message_start":
        return t.handleMessageStart(data)
    case "content_block_start":
        return t.handleContentBlockStart(data)
    case "content_block_delta":
        return t.handleContentBlockDelta(data)
    case "content_block_stop":
        return t.handleContentBlockStop(data)
    case "message_delta":
        return t.handleMessageDelta(data)
    case "message_stop":
        return t.handleMessageStop(data)
    case "ping":
        return nil, nil // Ignore keep-alives
    default:
        return nil, fmt.Errorf("unknown event type: %s", event)
    }
}

func (t *AnthropicStreamTransformer) handleMessageStart(data []byte) ([]byte, error) {
    var event struct {
        Message struct {
            ID    string `json:"id"`
            Model string `json:"model"`
            Role  string `json:"role"`
        } `json:"message"`
    }
    if err := json.Unmarshal(data, &event); err != nil {
        return nil, err
    }

    t.messageID = event.Message.ID
    t.model = event.Message.Model
    t.created = time.Now().Unix()

    // Return OpenAI-style start chunk
    openaiChunk := OpenAIStreamChunk{
        ID:      t.messageID,
        Object:  "chat.completion.chunk",
        Created: t.created,
        Model:   t.model,
        Choices: []OpenAIStreamChoice{
            {
                Index: 0,
                Delta: OpenAIMessageDelta{
                    Role: "assistant",
                },
            },
        },
    }

    return formatSSEChunk(openaiChunk)
}

func (t *AnthropicStreamTransformer) handleContentBlockDelta(data []byte) ([]byte, error) {
    var event struct {
        Index int `json:"index"`
        Delta struct {
            Type string `json:"type"`
            Text string `json:"text,omitempty"`
        } `json:"delta"`
    }
    if err := json.Unmarshal(data, &event); err != nil {
        return nil, err
    }

    // Accumulate for usage tracking
    t.buffer.WriteString(event.Delta.Text)

    // Transform to OpenAI format
    openaiChunk := OpenAIStreamChunk{
        ID:      t.messageID,
        Object:  "chat.completion.chunk",
        Created: t.created,
        Model:   t.model,
        Choices: []OpenAIStreamChoice{
            {
                Index: 0,
                Delta: OpenAIMessageDelta{
                    Content: event.Delta.Text,
                },
            },
        },
    }

    return formatSSEChunk(openaiChunk)
}

func (t *AnthropicStreamTransformer) handleMessageDelta(data []byte) ([]byte, error) {
    var event struct {
        Delta struct {
            StopReason   string  `json:"stop_reason"`
            StopSequence *string `json:"stop_sequence"`
        } `json:"delta"`
        Usage *AnthropicUsage `json:"usage,omitempty"`
    }
    if err := json.Unmarshal(data, &event); err != nil {
        return nil, err
    }

    finishReason := mapStopReason(event.Delta.StopReason)

    // Final chunk with finish_reason
    openaiChunk := OpenAIStreamChunk{
        ID:      t.messageID,
        Object:  "chat.completion.chunk",
        Created: t.created,
        Model:   t.model,
        Choices: []OpenAIStreamChoice{
            {
                Index:        0,
                Delta:        OpenAIMessageDelta{},
                FinishReason: &finishReason,
            },
        },
    }

    return formatSSEChunk(openaiChunk)
}

func (t *AnthropicStreamTransformer) handleMessageStop(data []byte) ([]byte, error) {
    // Return OpenAI [DONE] marker
    return []byte("data: [DONE]\n\n"), nil
}

func (t *AnthropicStreamTransformer) IsDoneMarker(chunk []byte) bool {
    // Check for Anthropic message_stop event or OpenAI [DONE]
    return bytes.Contains(chunk, []byte("event: message_stop")) ||
           bytes.Contains(chunk, []byte("data: [DONE]"))
}
```

### Streaming Event Types

```go
// Anthropic streaming event types
type AnthropicStreamEvent struct {
    Type string `json:"type"`
}

type AnthropicMessageStartEvent struct {
    Type    string                 `json:"type"`
    Message AnthropicStreamMessage `json:"message"`
}

type AnthropicStreamMessage struct {
    ID           string                  `json:"id"`
    Type         string                  `json:"type"`
    Role         string                  `json:"role"`
    Model        string                  `json:"model"`
    Content      []AnthropicContentBlock `json:"content"`
    StopReason   *string                 `json:"stop_reason"`
    StopSequence *string                 `json:"stop_sequence"`
    Usage        *AnthropicUsage         `json:"usage"`
}

type AnthropicContentBlockStartEvent struct {
    Type          string                  `json:"type"`
    Index         int                     `json:"index"`
    ContentBlock  AnthropicContentBlock   `json:"content_block"`
}

type AnthropicContentBlockDeltaEvent struct {
    Type  string                 `json:"type"`
    Index int                    `json:"index"`
    Delta AnthropicContentDelta  `json:"delta"`
}

type AnthropicContentDelta struct {
    Type string `json:"type"` // "text_delta"
    Text string `json:"text"`
}

type AnthropicContentBlockStopEvent struct {
    Type  string `json:"type"`
    Index int    `json:"index"`
}

type AnthropicMessageDeltaEvent struct {
    Type  string              `json:"type"`
    Delta AnthropicMessageDelta `json:"delta"`
    Usage *AnthropicUsage      `json:"usage,omitempty"`
}

type AnthropicMessageDelta struct {
    StopReason   string  `json:"stop_reason,omitempty"`
    StopSequence *string `json:"stop_sequence,omitempty"`
}
```

---

## Authentication Approach

### Header Configuration

Anthropic uses a distinct authentication pattern compared to OpenAI:

```go
func (t *AnthropicRequestTransformer) TransformHeaders(req *http.Request) error {
    // Anthropic uses x-api-key instead of Authorization: Bearer
    req.Header.Set("x-api-key", t.config.APIKey)

    // Content type
    req.Header.Set("Content-Type", "application/json")

    // Required version header (per Plexus patterns)
    req.Header.Set("anthropic-version", "2023-06-01")

    // Optional: Custom headers from config
    for key, value := range t.config.Headers {
        req.Header.Set(key, value)
    }

    return nil
}
```

### Auth Comparison

| Provider | Header Name | Format | Required Version |
|----------|-------------|--------|------------------|
| OpenAI | `Authorization` | `Bearer <key>` | None |
| Anthropic | `x-api-key` | `<key>` | `anthropic-version: 2023-06-01` |

### Configuration

```go
// Anthropic adapter configuration
type AnthropicConfig struct {
    APIKey      string
    BaseURL     string  // Default: https://api.anthropic.com
    Version     string  // Default: 2023-06-01
    Timeout     time.Duration
    MaxRetries  int
}

// Environment variables
// ANTHROPIC_API_KEY=<your-api-key>
// ANTHROPIC_BASE_URL=https://api.anthropic.com (optional override)
```

---

## Error Handling

### Anthropic Error Format

```json
{
  "type": "error",
  "error": {
    "type": "invalid_request_error",
    "message": "max_tokens: range error"
  }
}
```

### Error Transformation

```go
func (t *AnthropicResponseTransformer) TransformError(body []byte) error {
    var anthropicErr struct {
        Type  string `json:"type"`
        Error struct {
            Type    string `json:"type"`
            Message string `json:"message"`
        } `json:"error"`
    }

    if err := json.Unmarshal(body, &anthropicErr); err != nil {
        return fmt.Errorf("anthropic error: %s", string(body))
    }

    // Map to OpenAI-style error
    return fmt.Errorf("anthropic %s: %s",
        anthropicErr.Error.Type,
        anthropicErr.Error.Message)
}

// HTTP Status Code Mapping
var statusCodeMap = map[int]int{
    400: http.StatusBadRequest,       // invalid_request_error
    401: http.StatusUnauthorized,     // authentication_error
    403: http.StatusForbidden,        // permission_error
    404: http.StatusNotFound,         // not_found_error
    429: http.StatusTooManyRequests,  // rate_limit_error
    500: http.StatusInternalServerError, // api_error
    529: http.StatusServiceUnavailable,  // overloaded_error
}
```

---

## URL Transformation

### Endpoint Mapping

```go
func (t *AnthropicRequestTransformer) TransformURL(req *http.Request) error {
    // Map OpenAI paths to Anthropic paths
    path := req.URL.Path

    switch path {
    case "/v1/chat/completions":
        req.URL.Path = "/v1/messages"
    case "/v1/embeddings":
        // Anthropic doesn't have embeddings; route to error or fallback
        return fmt.Errorf("embeddings not supported by Anthropic")
    default:
        // Pass through as-is
    }

    // Set host and scheme
    req.URL.Scheme = "https"
    req.URL.Host = strings.TrimPrefix(t.config.BaseURL, "https://")

    return nil
}
```

---

## Complete Adapter Interface

```go
// AnthropicAdapter implements the provider.Adapter interface
type AnthropicAdapter struct {
    config          AnthropicConfig
    httpClient      *http.Client
    reqTransformer  *AnthropicRequestTransformer
    respTransformer *AnthropicResponseTransformer
    streamTransformer *AnthropicStreamTransformer
}

func NewAnthropicAdapter(apiKey string, opts ...AnthropicOption) *AnthropicAdapter {
    a := &AnthropicAdapter{
        config: AnthropicConfig{
            APIKey:     apiKey,
            BaseURL:    "https://api.anthropic.com",
            Version:    "2023-06-01",
            Timeout:    60 * time.Second,
            MaxRetries: 3,
        },
        reqTransformer:  NewAnthropicRequestTransformer(),
        respTransformer: NewAnthropicResponseTransformer(),
        streamTransformer: NewAnthropicStreamTransformer(),
    }

    for _, opt := range opts {
        opt(a)
    }

    a.httpClient = &http.Client{Timeout: a.config.Timeout}
    return a
}

func (a *AnthropicAdapter) Name() string {
    return "anthropic"
}

func (a *AnthropicAdapter) Execute(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
    switch req.APIType {
    case "chat":
        return a.executeChat(ctx, req, model)
    case "messages":
        // Anthropic-specific API type
        return a.executeMessages(ctx, req, model)
    default:
        return models.ProviderResult{}, fmt.Errorf("unsupported api type: %s", req.APIType)
    }
}
```

---

## Testing Strategy

### Unit Tests

```go
func TestAnthropicRequestTransformer_Transform(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected AnthropicMessageRequest
    }{
        {
            name: "extracts system message",
            input: `{"messages":[{"role":"system","content":"sys"},{"role":"user","content":"hi"}]}`,
            expected: AnthropicMessageRequest{
                System: "sys",
                Messages: []AnthropicMessage{
                    {Role: "user", Content: "hi"},
                },
            },
        },
        // Additional test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            transformer := NewAnthropicRequestTransformer()
            result, err := transformer.Transform(strings.NewReader(tt.input))
            // Assertions...
        })
    }
}
```

### Integration Tests

```go
func TestAnthropicAdapter_Execute(t *testing.T) {
    // Create test server that mimics Anthropic API
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "/v1/messages", r.URL.Path)
        assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
        assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

        // Return mock Anthropic response
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]any{
            "id": "msg_test",
            "type": "message",
            "role": "assistant",
            "content": []map[string]any{
                {"type": "text", "text": "Hello!"},
            },
            "usage": map[string]int{
                "input_tokens": 10,
                "output_tokens": 2,
            },
        })
    }))
    defer server.Close()

    adapter := NewAnthropicAdapter("test-key", WithBaseURL(server.URL))

    req := models.ProviderRequest{
        APIType: "chat",
        Payload: models.ChatCompletionRequest{
            Messages: []models.Message{
                {Role: "user", Content: "Hi"},
            },
        },
    }

    result, err := adapter.Execute(context.Background(), req, "claude-3-sonnet")
    // Assertions...
}
```

---

## Implementation Checklist

- [ ] Create `internal/provider/anthropic/` directory
- [ ] Implement `AnthropicAdapter` struct with `Adapter` interface
- [ ] Implement `AnthropicRequestTransformer`
  - [ ] System message extraction
  - [ ] Message format conversion
  - [ ] Parameter mapping (temperature, max_tokens, etc.)
- [ ] Implement `AnthropicResponseTransformer`
  - [ ] Content block aggregation
  - [ ] Stop reason mapping
  - [ ] Usage calculation
- [ ] Implement `AnthropicStreamTransformer`
  - [ ] Event type parsing
  - [ ] Content block delta handling
  - [ ] Message stop detection
- [ ] Add authentication header handling (`x-api-key`, `anthropic-version`)
- [ ] Add URL transformation (`/v1/chat/completions` → `/v1/messages`)
- [ ] Implement error handling and status code mapping
- [ ] Write unit tests for all transformers
- [ ] Write integration tests for the adapter
- [ ] Update factory.go to register Anthropic adapter
- [ ] Add configuration loading for Anthropic credentials
- [ ] Document known limitations (no embeddings support)

---

## References

- [OpenAI Adapter Reference](../architecture/provider-adapters.md)
- [Implementing Adapters Guide](../guides/implementing-adapters.md)
- Anthropic API Documentation: https://docs.anthropic.com/en/api/
- Anthropic Messages API: https://docs.anthropic.com/en/api/messages
- Plexus Patterns (internal): `/messages` endpoint, `x-api-key` auth
- AxonHub Patterns (internal): Unified intermediate format, bidirectional transformers
