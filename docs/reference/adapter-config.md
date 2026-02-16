# Provider Adapter Configuration Reference

This document describes all configuration options for provider adapters in Brass Relay.

## Configuration Overview

Provider adapter configuration consists of three main components:

1. **Adapter Registration** - Code-level registration in `main.go`
2. **Model Routes** - Mapping model aliases to provider candidates
3. **Environment Variables** - Provider API keys and settings

## Environment Variables

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `RAD_API_KEYS` | Comma-separated API keys for Brass Relay authentication | `admin:secret1,service:secret2` |
| `RAD_LISTEN_ADDR` | HTTP server bind address | `:8090` |

### Provider-Specific Variables

Each adapter should define its own environment variables:

```bash
# OpenAI
OPENAI_API_KEY="sk-..."
OPENAI_BASE_URL="https://api.openai.com/v1"  # Optional

# Anthropic
ANTHROPIC_API_KEY="sk-ant-..."
ANTHROPIC_BASE_URL="https://api.anthropic.com"  # Optional

# Google Gemini
GEMINI_API_KEY="..."
```

### Optional Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RAD_RETRY_BUDGET` | Maximum retry attempts per request | `2` |
| `RAD_LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |

## Model Route Configuration

Model routes define how incoming model names map to provider candidates.

### Route Table Structure

```go
map[string][]Candidate{
    "model-alias": {
        {Provider: "provider-name", Model: "provider-model-id", Weight: 100},
        {Provider: "fallback-provider", Model: "fallback-model", Weight: 50},
    },
}
```

### Configuration Location

Routes are defined in `internal/config/config.go`:

```go
func loadModelRoutes() map[string][]Candidate {
    return map[string][]Candidate{
        // Your routes here
    }
}
```

### Route Configuration Examples

#### Single Provider

```go
"gpt-4o": {
    {Provider: "openai", Model: "gpt-4o", Weight: 100},
},
```

#### Primary with Fallback

```go
"claude-3-5-sonnet": {
    {Provider: "anthropic", Model: "claude-3-5-sonnet-20241022", Weight: 100},
    {Provider: "openai", Model: "gpt-4o", Weight: 50},
},
```

#### Weighted Load Balancing

```go
"smart-chat": {
    {Provider: "openai", Model: "gpt-4o-mini", Weight: 70},
    {Provider: "anthropic", Model: "claude-3-haiku", Weight: 30},
},
```

#### Multi-Region Fallback

```go
"gpt-4o-ha": {
    {Provider: "openai-eu", Model: "gpt-4o", Weight: 100},
    {Provider: "openai-us", Model: "gpt-4o", Weight: 80},
    {Provider: "openai-asia", Model: "gpt-4o", Weight: 60},
},
```

## Candidate Configuration

### Candidate Fields

| Field | Type | Description |
|-------|------|-------------|
| `Provider` | string | Adapter name (must match `Adapter.Name()`) |
| `Model` | string | Provider-specific model identifier |
| `Weight` | int | Routing priority (higher = preferred) |

### Weight Semantics

Weights determine routing priority and can be used for:

- **Failover**: Primary (weight 100) → Secondary (weight 50)
- **Load balancing**: 70/30 split between providers
- **Cost optimization**: Prefer cheaper provider (weight 90) over expensive (weight 10)

```go
// Failover pattern
{Provider: "primary", Model: "gpt-4o", Weight: 100},
{Provider: "backup", Model: "gpt-4o", Weight: 50},

// Load balancing pattern
{Provider: "provider-a", Model: "model", Weight: 70},
{Provider: "provider-b", Model: "model", Weight: 30},
```

## Adapter Registration

### Registry Setup

Adapters are registered at application startup:

```go
// cmd/rad-gateway/main.go
registry := provider.NewRegistry(
    provider.NewMockAdapter(),
    provider.NewOpenAIAdapter(),
    provider.NewAnthropicAdapter(),
    // Add your adapter here
)
```

### Registration Order

Registration order does not affect routing. The `Router` uses the `routeTable` to determine which adapters to use.

## Retry Configuration

### Retry Budget

The retry budget controls how many provider attempts are made per request:

```go
// config.go
RetryBudget: 2  // Try up to 2 candidates
```

### Retry Behavior

1. Candidates are sorted by weight (descending)
2. Up to `retryBudget` candidates are attempted
3. First successful response is returned
4. If all fail, aggregated errors are returned

### Example Retry Scenarios

```go
// Configuration: retryBudget = 2
// Route: A (weight 100), B (weight 80), C (weight 60)

// Scenario 1: A succeeds
// Attempts: A (success) → Return result from A

// Scenario 2: A fails, B succeeds
// Attempts: A (fail), B (success) → Return result from B

// Scenario 3: All fail
// Attempts: A (fail), B (fail) → Return error
```

## Complete Configuration Example

### Environment File (.env)

```bash
# Server
RAD_LISTEN_ADDR=:8090
RAD_API_KEYS=admin:super-secret-key,service:service-key-123
RAD_RETRY_BUDGET=2
RAD_LOG_LEVEL=info

# Providers
OPENAI_API_KEY=sk-openai-key-here
ANTHROPIC_API_KEY=sk-ant-api03-key-here
GEMINI_API_KEY=AIzaSyGoogle-key-here
```

### Route Configuration (config.go)

```go
func loadModelRoutes() map[string][]Candidate {
    return map[string][]Candidate{
        // OpenAI models
        "gpt-4o": {
            {Provider: "openai", Model: "gpt-4o", Weight: 100},
        },
        "gpt-4o-mini": {
            {Provider: "openai", Model: "gpt-4o-mini", Weight: 80},
            {Provider: "anthropic", Model: "claude-3-haiku-20240307", Weight: 20},
        },

        // Anthropic models
        "claude-3-5-sonnet": {
            {Provider: "anthropic", Model: "claude-3-5-sonnet-20241022", Weight: 100},
            {Provider: "openai", Model: "gpt-4o", Weight: 50},
        },
        "claude-3-opus": {
            {Provider: "anthropic", Model: "claude-3-opus-20240229", Weight: 100},
        },

        // Google models
        "gemini-1.5-flash": {
            {Provider: "gemini", Model: "gemini-1.5-flash", Weight: 100},
        },

        // Embedding models
        "text-embedding-3-small": {
            {Provider: "openai", Model: "text-embedding-3-small", Weight: 100},
        },

        // Smart routing aliases
        "smart-chat": {
            {Provider: "openai", Model: "gpt-4o-mini", Weight: 60},
            {Provider: "anthropic", Model: "claude-3-haiku-20240307", Weight: 40},
        },
        "smart-premium": {
            {Provider: "anthropic", Model: "claude-3-5-sonnet-20241022", Weight: 70},
            {Provider: "openai", Model: "gpt-4o", Weight: 30},
        },
    }
}
```

### Main Registration (main.go)

```go
func main() {
    cfg := config.Load()

    registry := provider.NewRegistry(
        provider.NewMockAdapter(),
        provider.NewOpenAIAdapter(),
        provider.NewAnthropicAdapter(),
        provider.NewGeminiAdapter(),
    )

    router := routing.New(registry, cfg.ModelRoutes, cfg.RetryBudget)
    gateway := core.New(router, usage.NewSink(10000), trace.NewStore(50000))

    // ...
}
```

## Runtime Configuration

### Admin API Endpoints

Configuration can be viewed at runtime via the Admin API:

```http
GET /api/v0/admin/config
Authorization: Bearer <admin-key>

Response:
{
    "listenAddr": ":8090",
    "retryBudget": 2,
    "keysConfigured": 2,
    "models": {
        "gpt-4o": [...],
        "claude-3-5-sonnet": [...]
    }
}
```

### Dynamic Updates (Future)

Future versions may support dynamic route updates:

```http
POST /api/v0/admin/config/routes
Authorization: Bearer <admin-key>
Content-Type: application/json

{
    "model": "gpt-4o",
    "candidates": [
        {"provider": "openai", "model": "gpt-4o", "weight": 100}
    ]
}
```

## Configuration Validation

### Startup Validation

The gateway validates configuration at startup:

1. All route providers must be registered
2. Weights must be positive integers
3. At least one model route must be defined
4. Retry budget must be >= 1

### Validation Errors

```
Error: adapter "openai" referenced in routes but not registered
Error: no model routes configured
Error: retry budget must be at least 1
```

## Best Practices

### 1. Use Descriptive Model Aliases

```go
// Good - Clear intent
"gpt-4o-fast": {Provider: "openai", Model: "gpt-4o", Weight: 100}
"claude-coding": {Provider: "anthropic", Model: "claude-3-5-sonnet", Weight: 100}

// Avoid - Ambiguous
"model1": {Provider: "openai", Model: "gpt-4o", Weight: 100}
```

### 2. Always Provide Fallbacks

```go
// Good - Has fallback
"gpt-4o": {
    {Provider: "openai", Model: "gpt-4o", Weight: 100},
    {Provider: "openai", Model: "gpt-4o-mini", Weight: 50},
}

// Risky - No fallback
"gpt-4o": {
    {Provider: "openai", Model: "gpt-4o", Weight: 100},
}
```

### 3. Use Consistent Weight Scales

```go
// Good - Consistent 0-100 scale
{Provider: "primary", Model: "model", Weight: 100},
{Provider: "backup", Model: "model", Weight: 50},

// Avoid - Mixed scales
{Provider: "primary", Model: "model", Weight: 1000},
{Provider: "backup", Model: "model", Weight: 1},
```

### 4. Document Provider-Specific Models

```go
"claude-3-5-sonnet": {
    // claude-3-5-sonnet-20241022 - Latest version as of Oct 2024
    {Provider: "anthropic", Model: "claude-3-5-sonnet-20241022", Weight: 100},
}
```

## Troubleshooting Configuration

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| `adapter not found` | Provider name mismatch | Check `Name()` return value matches route |
| `all route attempts failed` | No valid candidates | Verify at least one candidate is valid |
| `unsupported api type` | Adapter doesn't implement type | Add handler for the API type |

### Debugging Routes

Enable debug logging to see routing decisions:

```bash
RAD_LOG_LEVEL=debug ./rad-gateway
```

Log output:
```
DEBUG: routing request for model "gpt-4o"
DEBUG: found 2 candidates: [openai:100 anthropic:50]
DEBUG: attempting provider=openai model=gpt-4o
DEBUG: provider openai success
```
