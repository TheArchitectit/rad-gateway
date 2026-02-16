# Provider Adapter Architecture

## Overview

The Provider Adapter system is the abstraction layer that enables Brass Relay to communicate with multiple AI providers (OpenAI, Anthropic, Google Gemini, etc.) through a unified interface. This architecture decouples the core gateway logic from provider-specific implementations, enabling seamless failover, load balancing, and multi-provider support.

## Design Philosophy

> **"One interface, many providers"** - The Adapter pattern abstracts provider differences while preserving each provider's unique capabilities.

### Key Principles

1. **Interface Uniformity**: All providers implement the same `Adapter` interface
2. **Protocol Translation**: Adapters handle request/response transformation
3. **Error Isolation**: Provider failures are contained and retriable
4. **Extensibility**: New providers are added without modifying core code

## System Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           BRASS RELAY GATEWAY                                │
│                                                                              │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐                │
│  │   API        │────►│    Core      │────►│   Router     │                │
│  │   Handlers   │     │   Gateway    │     │   Engine     │                │
│  └──────────────┘     └──────────────┘     └──────┬───────┘                │
│                                                   │                         │
│                              ┌────────────────────┘                         │
│                              │                                              │
│                              ▼                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    PROVIDER REGISTRY                                │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌───────────┐  │   │
│  │  │   Mock      │  │   OpenAI    │  │  Anthropic  │  │  Gemini   │  │   │
│  │  │   Adapter   │  │   Adapter   │  │   Adapter   │  │  Adapter  │  │   │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────┬─────┘  │   │
│  │         │                │                │               │        │   │
│  └─────────┼────────────────┼────────────────┼───────────────┼────────┘   │
│            │                │                │               │            │
└────────────┼────────────────┼────────────────┼───────────────┼────────────┘
             │                │                │               │
             ▼                ▼                ▼               ▼
       ┌──────────┐     ┌──────────┐     ┌──────────┐   ┌──────────┐
       │  Mock    │     │  OpenAI  │     │Anthropic │   │  Google  │
       │  Service │     │   API    │     │   API    │   │  Gemini  │
       └──────────┘     └──────────┘     └──────────┘   └──────────┘
```

### Core Interface

The `Adapter` interface (`internal/provider/provider.go:16-19`) defines the contract all provider adapters must implement:

```go
type Adapter interface {
    Name() string
    Execute(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error)
}
```

| Method | Purpose |
|--------|---------|
| `Name()` | Returns the provider identifier (e.g., "openai", "anthropic") |
| `Execute()` | Sends the request to the provider and returns the result |

## Data Flow

### Request Lifecycle

```
┌─────────┐    ┌──────────┐    ┌─────────────┐    ┌──────────┐    ┌─────────┐
│ Client  │───►│ Gateway  │───►│   Router    │───►│ Registry │───►│ Adapter │
└─────────┘    └──────────┘    └─────────────┘    └──────────┘    └────┬────┘
     │                                                                  │
     │  POST /v1/chat/completions                                      │
     │  {model: "gpt-4o", ...}                                          │
     │                                                              ┌───┴───┐
     │                                                              │Provider│
     │                                                              │  API   │
     │                                                              └───┬───┘
     │                                                                  │
     │  {choices: [...], usage: {...}}                                  │
     │◄─────────────────────────────────────────────────────────────────┘
```

### Execution Flow

1. **Request Parsing**: API handler decodes the incoming request
2. **Gateway Dispatch**: `core.Gateway.Handle()` processes the request
3. **Route Selection**: `routing.Router.Dispatch()` selects candidate providers
4. **Adapter Execution**: Registry retrieves adapter, adapter executes request
5. **Result Aggregation**: Response transformed and returned to client

## Provider Registry

The `Registry` (`internal/provider/provider.go:21-39`) maintains a map of all available adapters:

```go
type Registry struct {
    adapters map[string]Adapter
}

func NewRegistry(adapters ...Adapter) *Registry
func (r *Registry) Get(name string) (Adapter, error)
```

### Registration Pattern

Adapters are registered at startup in `cmd/rad-gateway/main.go`:

```go
registry := provider.NewRegistry(
    provider.NewMockAdapter(),
    // Additional adapters registered here
)
```

## Request/Response Types

### ProviderRequest

```go
type ProviderRequest struct {
    APIType string  // "chat", "responses", "messages", "embeddings", etc.
    Model   string  // Model identifier
    Payload any     // Type-specific request payload
}
```

### ProviderResult

```go
type ProviderResult struct {
    Model    string  // Actual model used
    Provider string  // Provider name
    Status   string  // "success" or "error"
    Usage    Usage   // Token and cost information
    Payload  any     // Type-specific response
}
```

## Supported API Types

Adapters must handle these API types:

| API Type | Description | Example Models |
|----------|-------------|----------------|
| `chat` | Chat completions | gpt-4o, claude-3-5-sonnet |
| `responses` | OpenAI Responses API | gpt-4o |
| `messages` | Anthropic Messages API | claude-3-5-sonnet |
| `embeddings` | Text embeddings | text-embedding-3-small |
| `images` | Image generation | gpt-image-1, dall-e-3 |
| `transcriptions` | Audio transcription | whisper-1 |
| `gemini` | Google Gemini API | gemini-1.5-flash |

## Adapter Implementation Pattern

### Mock Adapter Example

The `MockAdapter` (`internal/provider/mock.go`) demonstrates the implementation pattern:

```go
type MockAdapter struct{}

func (m *MockAdapter) Name() string {
    return "mock"
}

func (m *MockAdapter) Execute(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
    switch req.APIType {
    case "chat":
        return m.handleChat(req, model)
    case "embeddings":
        return m.handleEmbeddings(req, model)
    // ... other cases
    default:
        return models.ProviderResult{}, fmt.Errorf("unsupported api type: %s", req.APIType)
    }
}
```

### Implementation Guidelines

1. **API Type Switch**: Route to handler based on `req.APIType`
2. **Payload Type Assertion**: Cast `req.Payload` to appropriate type
3. **Provider Transformation**: Convert internal request to provider format
4. **HTTP Client**: Use provider's REST API with proper authentication
5. **Response Mapping**: Transform provider response to `ProviderResult`
6. **Error Handling**: Return wrapped errors for proper retry logic

## Routing Integration

Adapters integrate with the routing system through the `Candidate` struct:

```go
type Candidate struct {
    Name   string  // Adapter name (matches Adapter.Name())
    Model  string  // Provider-specific model identifier
    Weight int     // Routing priority (higher = preferred)
}
```

### Route Table Configuration

Routes map incoming model names to provider candidates (`internal/config/config.go:64-74`):

```go
"gpt-4o-mini": {
    {Provider: "mock", Model: "gpt-4o-mini", Weight: 80},
    {Provider: "mock", Model: "fallback-mini", Weight: 20},
},
```

## Retry and Failover

The routing engine implements automatic failover:

1. Candidates are sorted by weight (highest first)
2. Router attempts up to `retryBudget` candidates
3. On failure, next candidate is attempted
4. All attempts are recorded for debugging

```go
attempts := make([]Attempt, 0, attemptLimit)
for i := 0; i < attemptLimit; i++ {
    cand := sorted[i]
    adapter, err := r.registry.Get(cand.Name)
    res, err := adapter.Execute(ctx, req, cand.Model)
    // ... handle result or retry
}
```

## Security Considerations

### Authentication

- Adapters receive provider API keys from environment variables
- Keys are never logged or exposed in error messages
- Each provider uses its own authentication scheme

### Error Handling

- Provider errors are wrapped, not exposed directly
- Sensitive information is redacted from error messages
- Failed requests are logged for debugging (without payloads)

## Testing Strategy

### Unit Testing

Test adapters with mocked HTTP clients:

```go
func TestMockAdapter_Execute(t *testing.T) {
    adapter := NewMockAdapter()
    req := models.ProviderRequest{
        APIType: "chat",
        Model:   "test-model",
        Payload: models.ChatCompletionRequest{...},
    }
    result, err := adapter.Execute(context.Background(), req, "test-model")
    // Assert expected behavior
}
```

### Integration Testing

Use the mock adapter for handler testing:

```go
registry := provider.NewRegistry(provider.NewMockAdapter())
router := routing.New(registry, routeTable, 2)
gateway := core.New(router, usage.NewSink(1000), trace.NewStore(5000))
```

## Future Enhancements

### Streaming Support

Extend `Adapter` interface for streaming responses:

```go
type StreamingAdapter interface {
    Adapter
    ExecuteStream(ctx context.Context, req models.ProviderRequest, model string) (<-chan StreamChunk, error)
}
```

### Health Checking

Add health check methods for proactive provider monitoring:

```go
type HealthChecker interface {
    HealthCheck(ctx context.Context) HealthStatus
}
```

### Caching Layer

Implement response caching for idempotent requests:

```go
type CachingAdapter struct {
    inner   Adapter
    cache   Cache
    ttl     time.Duration
}
```

## References

- [Implementation Guide](../guides/implementing-adapters.md)
- [Configuration Reference](../reference/adapter-config.md)
- [Troubleshooting Guide](../troubleshooting/adapters.md)
