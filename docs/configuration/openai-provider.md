# OpenAI Provider Configuration

## Overview

The RAD Gateway supports OpenAI as a provider for chat completions and embeddings. This document describes how to configure and use the OpenAI provider.

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OPENAI_API_KEY` | Yes | - | Your OpenAI API key |
| `OPENAI_BASE_URL` | No | `https://api.openai.com/v1` | Custom base URL for OpenAI-compatible APIs |
| `OPENAI_TIMEOUT` | No | `60s` | Request timeout duration |
| `OPENAI_MAX_RETRIES` | No | `3` | Maximum number of retry attempts |
| `OPENAI_RETRY_DELAY` | No | `500ms` | Initial retry delay (uses exponential backoff) |

### Example Configuration

```bash
# Required
export OPENAI_API_KEY="sk-..."

# Optional overrides
export OPENAI_TIMEOUT="120s"
export OPENAI_MAX_RETRIES="5"
```

## Supported Models

### Chat Completion Models

| Model | Input Cost ($/1K tokens) | Output Cost ($/1K tokens) |
|-------|-------------------------|--------------------------|
| `gpt-4o` | $0.0025 | $0.01 |
| `gpt-4o-2024-08-06` | $0.0025 | $0.01 |
| `gpt-4o-2024-05-13` | $0.005 | $0.015 |
| `gpt-4o-mini` | $0.00015 | $0.0006 |
| `gpt-4o-mini-2024-07-18` | $0.00015 | $0.0006 |
| `gpt-4-turbo` | $0.01 | $0.03 |
| `gpt-4-turbo-2024-04-09` | $0.01 | $0.03 |
| `gpt-4-turbo-preview` | $0.01 | $0.03 |
| `gpt-4` | $0.03 | $0.06 |
| `gpt-4-32k` | $0.06 | $0.12 |
| `gpt-4-0613` | $0.03 | $0.06 |
| `gpt-4-32k-0613` | $0.06 | $0.12 |
| `gpt-4-1106-preview` | $0.01 | $0.03 |
| `gpt-3.5-turbo` | $0.0005 | $0.0015 |
| `gpt-3.5-turbo-16k` | $0.003 | $0.004 |
| `gpt-3.5-turbo-0125` | $0.0005 | $0.0015 |
| `gpt-3.5-turbo-1106` | $0.001 | $0.002 |

### Embedding Models

| Model | Cost ($/1K tokens) |
|-------|-------------------|
| `text-embedding-3-small` | $0.00002 |
| `text-embedding-3-large` | $0.00013 |
| `text-embedding-ada-002` | $0.0001 |

## Cost Tracking

The OpenAI adapter automatically tracks costs for each request. Costs are calculated based on:

- **Input tokens**: Tokens in the prompt/request
- **Output tokens**: Tokens in the completion/response
- **Model pricing**: Per-model rates from OpenAI's pricing

### Accessing Cost Information

Costs are available in the response:

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

### Streaming Cost Tracking

For streaming responses, cost is calculated when the stream completes:

```go
result, err := adapter.Execute(ctx, req, model)
if err != nil {
    // handle error
}

// For streaming, cast to StreamingResponse
if stream, ok := result.Payload.(*StreamingResponse); ok {
    // Stream is complete when reader EOF
    // Cost is available via stream.Cost()
}
```

## Retry Behavior

The adapter implements exponential backoff with jitter for retries:

- **Retryable errors**: 5xx server errors, 429 rate limiting, network errors
- **Non-retryable errors**: 400 bad request, 401 unauthorized, 403 forbidden
- **Backoff formula**: `delay = retry_delay * 2^attempt` (capped at 8 seconds)

## Usage Examples

### Chat Completion

```go
adapter := openai.NewAdapter(apiKey)

req := models.ProviderRequest{
    APIType: "chat",
    Payload: models.ChatCompletionRequest{
        Messages: []models.Message{
            {Role: "system", Content: "You are a helpful assistant"},
            {Role: "user", Content: "Hello!"},
        },
        Stream: false,
    },
}

result, err := adapter.Execute(ctx, req, "gpt-4o")
```

### Streaming Chat Completion

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

result, err := adapter.Execute(ctx, req, "gpt-4o-mini")
```

### Embeddings

```go
req := models.ProviderRequest{
    APIType: "embeddings",
    Payload: models.EmbeddingsRequest{
        Input: "The quick brown fox",
    },
}

result, err := adapter.Execute(ctx, req, "text-embedding-3-small")
```

## Error Handling

The adapter returns detailed errors:

```go
result, err := adapter.Execute(ctx, req, model)
if err != nil {
    // Check for specific error types
    if strings.Contains(err.Error(), "openai api error") {
        // API returned an error response
    } else if strings.Contains(err.Error(), "all retries exhausted") {
        // All retry attempts failed
    }
}
```

## Testing

For testing, you can use a mock server:

```go
// Start mock OpenAI server
mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "id": "test-id",
        "choices": []map[string]interface{}{
            {"message": map[string]string{"content": "Hello!"}},
        },
        "usage": map[string]int{
            "prompt_tokens": 10,
            "completion_tokens": 5,
            "total_tokens": 15,
        },
    })
}))
defer mockServer.Close()

// Configure adapter to use mock
adapter := openai.NewAdapter("test-key",
    openai.WithBaseURL(mockServer.URL),
)
```

## See Also

- [Provider Adapters](../architecture/provider-adapters.md)
- [Cost Tracking](../operations/cost-tracking.md)
- [Rate Limiting](../configuration/rate-limiting.md)
