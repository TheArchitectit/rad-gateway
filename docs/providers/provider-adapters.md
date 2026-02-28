# Provider Adapters

RAD Gateway supports multiple AI providers through provider-specific adapters that handle request/response transformations.

## Supported Providers

| Provider | Adapter | Status | Features |
|----------|---------|--------|----------|
| **OpenAI** | `internal/provider/openai` | ✅ Complete | Chat, Embeddings, Streaming |
| **Anthropic** | `internal/provider/anthropic` | ✅ Complete | Claude API, Streaming |
| **Gemini** | `internal/provider/gemini` | ✅ Complete | Google AI API, Streaming |
| **Ollama** | `internal/provider/generic` | ✅ Complete | OpenAI-compatible, Local |
| **Generic** | `internal/provider/generic` | ✅ Complete | Any OpenAI-compatible API |

## Architecture

Each provider adapter implements the `provider.Adapter` interface:

```go
type Adapter interface {
    Name() string
    Execute(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error)
}
```

### Adapter Components

```
┌─────────────────────────────────────────┐
│         Provider Adapter                │
├─────────────────────────────────────────┤
│  RequestTransformer                     │
│  - Convert internal → Provider format   │
├─────────────────────────────────────────┤
│  ResponseTransformer                    │
│  - Convert Provider → internal format   │
├─────────────────────────────────────────┤
│  StreamTransformer                      │
│  - Handle SSE streaming                 │
├─────────────────────────────────────────┤
│  CostCalculator                         │
│  - Calculate usage costs                │
└─────────────────────────────────────────┘
```

## Configuration

### OpenAI

```bash
export OPENAI_API_KEY="sk-..."
```

Models:
- `gpt-4o`, `gpt-4o-mini`
- `gpt-4-turbo`, `gpt-4`
- `gpt-3.5-turbo`

### Anthropic

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

Models:
- `claude-3-5-sonnet-20241022`
- `claude-3-opus-20240229`
- `claude-3-sonnet-20240229`
- `claude-3-haiku-20240307`

### Gemini

```bash
export GEMINI_API_KEY="..."
```

Models:
- `gemini-1.5-pro`
- `gemini-1.5-flash`
- `gemini-1.0-pro`

### Ollama (Local)

```bash
export OLLAMA_ENABLED="true"
export OLLAMA_BASE_URL="http://localhost:11434/v1"
export OLLAMA_API_KEY="ollama"  # Not required by Ollama

# Pull models
ollama pull llama3.2:latest
ollama pull mistral:latest
```

## Request Flow

```
1. Client Request (OpenAI format)
        ↓
2. Gateway receives /v1/chat/completions
        ↓
3. Router selects provider based on model
        ↓
4. Provider Adapter:
   a. TransformRequest: Convert to provider format
   b. Execute HTTP request
   c. TransformResponse: Convert back to OpenAI format
        ↓
5. Return unified response to client
```

## Provider-Specific Transformations

### OpenAI

**Request**: Direct mapping (already OpenAI format)
**Response**: Direct mapping
**Streaming**: SSE with `data: {...}` format

### Anthropic

**Request Transformations**:
- System messages → `system` field
- Other messages → `messages` array (user/assistant only)
- Role mapping: `system` → filtered, `assistant` → `assistant`, `user` → `user`

**Response Transformations**:
- Content blocks → message content
- `input_tokens` → `prompt_tokens`
- `output_tokens` → `completion_tokens`

**Streaming**:
- Event-based SSE (`event: content_block_delta`)
- Transformed to OpenAI-compatible SSE

### Gemini

**Request Transformations**:
- Messages → `contents` array
- Role mapping: `user` → `user`, `assistant` → `model`
- System instructions → `systemInstruction` field

**Response Transformations**:
- Candidates → choices
- `promptTokenCount` → `prompt_tokens`
- `candidatesTokenCount` → `completion_tokens`

**Streaming**:
- Server-side streaming with chunked responses
- Transformed to OpenAI-compatible SSE

## Model Routing

Models are routed based on the `model` field in the request:

```go
// Example routing configuration
ModelRoutes: map[string][]Candidate{
    "gpt-4o-mini": {
        {Provider: "openai", Model: "gpt-4o-mini", Weight: 100},
    },
    "claude-3-5-sonnet": {
        {Provider: "anthropic", Model: "claude-3-5-sonnet-20241022", Weight: 100},
    },
    "gemini-1.5-flash": {
        {Provider: "gemini", Model: "gemini-1.5-flash", Weight: 100},
    },
}
```

## Cost Tracking

Each provider has its own pricing calculator:

```go
// OpenAI pricing (per 1K tokens)
Pricing: map[string]ModelPricing{
    "gpt-4o": {
        InputPrice:  0.005,
        OutputPrice: 0.015,
    },
    "gpt-4o-mini": {
        InputPrice:  0.00015,
        OutputPrice: 0.0006,
    },
}
```

Costs are calculated and returned in the response:
```json
{
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30,
    "cost_total": 0.00015
  }
}
```

## Error Handling

Each provider returns errors in its own format. Adapters normalize these to OpenAI-compatible error responses:

```json
{
  "error": {
    "message": "Error message",
    "type": "error_type",
    "code": "error_code"
  }
}
```

## Testing

### Test with specific providers:

```bash
# OpenAI
curl -X POST http://localhost:8090/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

# Anthropic
curl -X POST http://localhost:8090/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

# Ollama (local)
curl -X POST http://localhost:8090/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## Adding a New Provider

To add a new provider:

1. Create `internal/provider/<name>/adapter.go`
2. Implement `provider.Adapter` interface
3. Create transformers for request/response
4. Add cost calculator
5. Register in `main.go`

See existing adapters for examples.

---

**Last Updated**: 2026-02-28
