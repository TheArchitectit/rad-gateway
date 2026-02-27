# Anthropic Provider Configuration

## Overview

The RAD Gateway supports Anthropic's Claude API for chat completions. This document describes how to configure and use the Anthropic provider.

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ANTHROPIC_API_KEY` | Yes | - | Your Anthropic API key |
| `ANTHROPIC_BASE_URL` | No | `https://api.anthropic.com` | Custom base URL |
| `ANTHROPIC_VERSION` | No | `2023-06-01` | Anthropic API version |
| `ANTHROPIC_TIMEOUT` | No | `60s` | Request timeout duration |
| `ANTHROPIC_MAX_RETRIES` | No | `3` | Maximum number of retry attempts |
| `ANTHROPIC_RETRY_DELAY` | No | `500ms` | Initial retry delay (uses exponential backoff) |

### Example Configuration

```bash
# Required
export ANTHROPIC_API_KEY="sk-ant-..."

# Optional overrides
export ANTHROPIC_TIMEOUT="120s"
export ANTHROPIC_MAX_RETRIES="5"
export ANTHROPIC_VERSION="2023-06-01"
```

## Supported Models

### Claude 3.5 Models

| Model | Input Cost ($/1K tokens) | Output Cost ($/1K tokens) |
|-------|-------------------------|--------------------------|
| `claude-3-5-sonnet-20241022` | $0.003 | $0.015 |
| `claude-3-5-sonnet-20240620` | $0.003 | $0.015 |
| `claude-3-5-sonnet-latest` | $0.003 | $0.015 |
| `claude-3-5-haiku-20241022` | $0.001 | $0.005 |
| `claude-3-5-haiku-latest` | $0.001 | $0.005 |

### Claude 3 Models

| Model | Input Cost ($/1K tokens) | Output Cost ($/1K tokens) |
|-------|-------------------------|--------------------------|
| `claude-3-opus-20240229` | $0.015 | $0.075 |
| `claude-3-opus-latest` | $0.015 | $0.075 |
| `claude-3-sonnet-20240229` | $0.003 | $0.015 |
| `claude-3-haiku-20240307` | $0.00025 | $0.00125 |

### Legacy Models

| Model | Input Cost ($/1K tokens) | Output Cost ($/1K tokens) |
|-------|-------------------------|--------------------------|
| `claude-2.1` | $0.008 | $0.024 |
| `claude-2.0` | $0.008 | $0.024 |
| `claude-instant-1.2` | $0.0008 | $0.0024 |

## Message Format

Anthropic uses a different message format than OpenAI:

- **System messages**: Sent in a separate `system` field, not in the messages array
- **Message roles**: Only `user` and `assistant` roles in the messages array
- **Max tokens**: Required parameter (default: 4096)

### Request Transformation

The adapter automatically transforms OpenAI-style requests to Anthropic format:

```go
// OpenAI-style request
req := models.ChatCompletionRequest{
    Messages: []models.Message{
        {Role: "system", Content: "You are a helpful assistant"},
        {Role: "user", Content: "Hello!"},
    },
}

// Transformed to Anthropic format:
// {
//   "system": "You are a helpful assistant",
//   "messages": [
//     {"role": "user", "content": "Hello!"}
//   ],
//   "max_tokens": 4096
// }
```

## Cost Tracking

The Anthropic adapter automatically tracks costs for each request based on token usage and model pricing.

### Accessing Cost Information

```go
result, err := adapter.Execute(ctx, req, model)
if err != nil {
    // handle error
}

fmt.Printf("Cost: $%.6f\n", result.Usage.CostTotal)
fmt.Printf("Tokens: %d (input: %d, output: %d)\n",
    result.Usage.TotalTokens,
    result.Usage.PromptTokens,
    result.Usage.CompletionTokens)
```

## Streaming Responses

Anthropic uses a different streaming format (Server-Sent Events with event types). The adapter automatically transforms these to OpenAI-compatible SSE format.

### Streaming Events

| Event Type | Description |
|------------|-------------|
| `message_start` | Beginning of message |
| `content_block_start` | Beginning of content block |
| `content_block_delta` | Content delta (text updates) |
| `content_block_stop` | End of content block |
| `message_delta` | Message metadata updates |
| `message_stop` | End of message |

### Usage with Streaming

```go
req := models.ProviderRequest{
    APIType: "chat",
    Payload: models.ChatCompletionRequest{
        Messages: []models.Message{
            {Role: "user", Content: "Tell me a story"},
        },
        Stream: true,
    },
}

result, err := adapter.Execute(ctx, req, "claude-3-5-sonnet-20241022")
if err != nil {
    // handle error
}

// For streaming, cast to StreamingResponse
if stream, ok := result.Payload.(*StreamingResponse); ok {
    // Read from stream.Reader
    // Cost is available via stream.Cost() when complete
}
```

## Retry Behavior

The adapter implements exponential backoff with jitter for retries:

- **Retryable errors**: 5xx server errors, 429 rate limiting, network errors
- **Non-retryable errors**: 400 bad request, 401 unauthorized, 403 forbidden
- **Backoff formula**: `delay = retry_delay * 2^attempt` (capped at 8 seconds)

## Error Handling

Anthropic errors are transformed to standard error format:

```go
result, err := adapter.Execute(ctx, req, model)
if err != nil {
    // Check for specific error types
    if strings.Contains(err.Error(), "anthropic") {
        // Anthropic API error
        if strings.Contains(err.Error(), "rate_limit") {
            // Rate limited
        } else if strings.Contains(err.Error(), "invalid_api_key") {
            // Authentication error
        }
    }
}
```

## Authentication

Anthropic uses a different authentication header than OpenAI:

- **Header**: `x-api-key` (not `Authorization: Bearer`)
- **Version header**: `anthropic-version: 2023-06-01`

The adapter handles this automatically.

## Usage Examples

### Basic Chat Completion

```go
adapter := anthropic.NewAdapter(apiKey)

req := models.ProviderRequest{
    APIType: "chat",
    Payload: models.ChatCompletionRequest{
        Messages: []models.Message{
            {Role: "system", Content: "You are a helpful assistant"},
            {Role: "user", Content: "Hello!"},
        },
        MaxTokens: 1024,
    },
}

result, err := adapter.Execute(ctx, req, "claude-3-5-sonnet-20241022")
```

### With Custom Configuration

```go
adapter := anthropic.NewAdapter(apiKey,
    anthropic.WithTimeout(120*time.Second),
    anthropic.WithRetryConfig(5, 1*time.Second),
    anthropic.WithVersion("2023-06-01"),
)
```

### Testing

For testing, you can use a mock server:

```go
mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "id": "test-id",
        "type": "message",
        "role": "assistant",
        "content": []map[string]string{
            {"type": "text", "text": "Hello!"},
        },
        "usage": map[string]int{
            "input_tokens": 10,
            "output_tokens": 5,
        },
    })
}))
defer mockServer.Close()

adapter := anthropic.NewAdapter("test-key",
    anthropic.WithBaseURL(mockServer.URL),
)
```

## See Also

- [Provider Adapters](../architecture/provider-adapters.md)
- [Cost Tracking](../operations/cost-tracking.md)
- [OpenAI Provider](./openai-provider.md)
- [Rate Limiting](../configuration/rate-limiting.md)
