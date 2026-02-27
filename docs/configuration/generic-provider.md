# Generic HTTP Provider Configuration

## Overview

The RAD Gateway includes a generic HTTP adapter that can connect to any OpenAI-compatible API. This is useful for:

- **Self-hosted models** (Ollama, LocalAI, etc.)
- **OpenAI-compatible proxies**
- **Custom LLM endpoints**
- **Other compatible providers** not explicitly supported

The generic adapter supports the OpenAI API format for chat completions and embeddings.

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GENERIC_BASE_URL` | Yes | - | Base URL for the API (e.g., `https://api.example.com/v1`) |
| `GENERIC_API_KEY` | Yes | - | API key for authentication |
| `GENERIC_TIMEOUT` | No | `60s` | Request timeout duration |
| `GENERIC_MAX_RETRIES` | No | `3` | Maximum number of retry attempts |
| `GENERIC_RETRY_DELAY` | No | `500ms` | Initial retry delay (exponential backoff) |
| `GENERIC_CUSTOM_HEADERS` | No | - | JSON object of additional headers (e.g., `{"X-Custom":"value"}`) |
| `GENERIC_AUTH_TYPE` | No | `bearer` | Authentication type: `bearer`, `api-key`, or `custom` |
| `GENERIC_AUTH_HEADER` | No | `Authorization` | Header name for authentication |
| `GENERIC_AUTH_PREFIX` | No | `Bearer ` | Prefix for auth token (include trailing space) |

### Example Configuration

```bash
# For Ollama
export GENERIC_BASE_URL="http://localhost:11434/v1"
export GENERIC_API_KEY="ollama"  # Ollama doesn't require auth, but adapter expects it
export GENERIC_AUTH_PREFIX=""    # No prefix needed for Ollama

# For Azure OpenAI
export GENERIC_BASE_URL="https://your-resource.openai.azure.com/openai/deployments/your-deployment"
export GENERIC_API_KEY="your-azure-key"
export GENERIC_AUTH_TYPE="api-key"
export GENERIC_AUTH_HEADER="api-key"
export GENERIC_AUTH_PREFIX=""

# For custom provider with API key
export GENERIC_BASE_URL="https://api.custom-provider.com/v1"
export GENERIC_API_KEY="your-api-key"
export GENERIC_AUTH_TYPE="api-key"
export GENERIC_AUTH_HEADER="X-API-Key"
export GENERIC_AUTH_PREFIX=""
```

## Authentication Types

### Bearer Token (Default)

```bash
export GENERIC_AUTH_TYPE="bearer"
export GENERIC_AUTH_HEADER="Authorization"
export GENERIC_AUTH_PREFIX="Bearer "
export GENERIC_API_KEY="sk-..."
```

Results in header: `Authorization: Bearer sk-...`

### API Key

```bash
export GENERIC_AUTH_TYPE="api-key"
export GENERIC_AUTH_HEADER="X-API-Key"
export GENERIC_AUTH_PREFIX=""
export GENERIC_API_KEY="your-api-key"
```

Results in header: `X-API-Key: your-api-key`

### Custom Authentication

```bash
export GENERIC_AUTH_TYPE="custom"
export GENERIC_AUTH_HEADER="X-Custom-Auth"
export GENERIC_AUTH_PREFIX="ApiKey "
export GENERIC_API_KEY="your-key"
```

Results in header: `X-Custom-Auth: ApiKey your-key`

### Custom Headers

Additional headers can be configured via environment variable (JSON format):

```bash
export GENERIC_CUSTOM_HEADERS='{"X-Request-ID":"12345","X-Environment":"production"}'
```

Or programmatically:

```go
adapter := generic.NewAdapter(
    baseURL,
    apiKey,
    generic.WithHeaders(map[string]string{
        "X-Request-ID": "12345",
        "X-Environment": "production",
    }),
)
```

## Supported APIs

The generic adapter expects OpenAI-compatible endpoints:

### Chat Completions

```
POST /chat/completions
```

Request format:
```json
{
  "model": "model-name",
  "messages": [
    {"role": "system", "content": "You are helpful"},
    {"role": "user", "content": "Hello!"}
  ],
  "stream": false
}
```

Response format:
```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "model-name",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "Hello! How can I help?"
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30
  }
}
```

### Embeddings

```
POST /embeddings
```

Request format:
```json
{
  "model": "embedding-model",
  "input": "The text to embed"
}
```

Response format:
```json
{
  "object": "list",
  "data": [{
    "object": "embedding",
    "embedding": [0.1, 0.2, 0.3, ...],
    "index": 0
  }],
  "model": "embedding-model",
  "usage": {
    "prompt_tokens": 5,
    "total_tokens": 5
  }
}
```

## Usage Examples

### Basic Chat Completion

```go
adapter := generic.NewAdapter(
    "http://localhost:11434/v1",
    "ollama",
    generic.WithAuthType("bearer", "Authorization", ""),
)

req := models.ProviderRequest{
    APIType: "chat",
    Payload: models.ChatCompletionRequest{
        Messages: []models.Message{
            {Role: "user", Content: "Hello!"},
        },
    },
}

result, err := adapter.Execute(ctx, req, "llama2")
```

### With Custom Headers

```go
adapter := generic.NewAdapter(
    "https://api.custom.com/v1",
    "custom-key",
    generic.WithHeaders(map[string]string{
        "X-Custom-Header": "value",
        "X-Request-ID": uuid.New().String(),
    }),
    generic.WithTimeout(120*time.Second),
    generic.WithRetryConfig(5, 1*time.Second),
)
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

result, err := adapter.Execute(ctx, req, "model-name")
if err != nil {
    // handle error
}

// Cast to StreamingResponse
if stream, ok := result.Payload.(*generic.StreamingResponse); ok {
    // Read from stream.Reader
    // Process SSE events
}
```

### Embeddings

```go
req := models.ProviderRequest{
    APIType: "embeddings",
    Payload: models.EmbeddingsRequest{
        Input: "The quick brown fox",
    },
}

result, err := adapter.Execute(ctx, req, "embeddings-model")
```

## Supported Providers

The generic adapter can work with any OpenAI-compatible API, including:

| Provider | Configuration |
|----------|--------------|
| **Ollama** | `GENERIC_BASE_URL=http://localhost:11434/v1` |
| **LocalAI** | `GENERIC_BASE_URL=http://localhost:8080/v1` |
| **Azure OpenAI** | Use full deployment URL with auth header |
| **Anyscale** | `GENERIC_BASE_URL=https://api.endpoints.anyscale.com/v1` |
| **Together AI** | `GENERIC_BASE_URL=https://api.together.xyz/v1` |
| **Fireworks** | `GENERIC_BASE_URL=https://api.fireworks.ai/inference/v1` |
| **Replicate** | OpenAI-compatible endpoints |
| **Custom** | Any OpenAI-compatible endpoint |

## Retry Behavior

The adapter implements exponential backoff with jitter for retries:

- **Retryable errors**: 5xx server errors, 429 rate limiting, network errors
- **Non-retryable errors**: 400 bad request, 401 unauthorized, 403 forbidden
- **Backoff formula**: `delay = retry_delay * 2^attempt` (capped at 8 seconds)

## Testing

For testing with a mock server:

```go
mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    response := map[string]interface{}{
        "id": "test-id",
        "object": "chat.completion",
        "choices": []map[string]interface{}{
            {
                "message": map[string]string{
                    "role": "assistant",
                    "content": "Hello!",
                },
            },
        },
        "usage": map[string]int{
            "prompt_tokens": 10,
            "completion_tokens": 5,
            "total_tokens": 15,
        },
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}))
defer mockServer.Close()

adapter := generic.NewAdapter(
    mockServer.URL,
    "test-key",
)
```

## Limitations

The generic adapter has some limitations compared to native provider adapters:

1. **No cost tracking** - Generic adapter doesn't have built-in pricing data
2. **No provider-specific features** - Only supports standard OpenAI-compatible features
3. **No automatic model discovery** - You must specify model names
4. **Basic error handling** - Uses standard error parsing

## Error Handling

```go
result, err := adapter.Execute(ctx, req, model)
if err != nil {
    // Check for specific errors
    if strings.Contains(err.Error(), "api returned status") {
        // HTTP error
        statusCode := extractStatusCode(err.Error())
    } else if strings.Contains(err.Error(), "all retries exhausted") {
        // Retries failed
    }
}
```

## See Also

- [OpenAI Provider](./openai-provider.md)
- [Anthropic Provider](./anthropic-provider.md)
- [Provider Adapters](../architecture/provider-adapters.md)
- [Model Routing](../configuration/model-routing.md)
