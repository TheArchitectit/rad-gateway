# Implementing Provider Adapters

This guide walks you through implementing a new provider adapter for Brass Relay.

## Prerequisites

- Go 1.24 or later
- Understanding of the provider's API (REST endpoints, authentication, request/response formats)
- Access to the provider's API keys for testing

## Quick Start

To add a new provider adapter, you will:

1. Create a new adapter file in `internal/provider/`
2. Implement the `Adapter` interface
3. Register the adapter in the registry
4. Configure model routes
5. Test the implementation

## Step-by-Step Implementation

### Step 1: Create the Adapter File

Create a new file `internal/provider/{provider_name}.go`:

```go
package provider

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"

    "radgateway/internal/models"
)

type YourProviderAdapter struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
}

func NewYourProviderAdapter() *YourProviderAdapter {
    return &YourProviderAdapter{
        apiKey:  os.Getenv("YOUR_PROVIDER_API_KEY"),
        baseURL: "https://api.yourprovider.com/v1",
        httpClient: &http.Client{
            Timeout: 60 * time.Second,
        },
    }
}

func (a *YourProviderAdapter) Name() string {
    return "yourprovider"
}
```

### Step 2: Implement API Type Handlers

Add handler methods for each supported API type:

```go
func (a *YourProviderAdapter) Execute(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
    switch req.APIType {
    case "chat":
        return a.handleChat(ctx, req, model)
    case "embeddings":
        return a.handleEmbeddings(ctx, req, model)
    default:
        return models.ProviderResult{}, fmt.Errorf("unsupported api type: %s", req.APIType)
    }
}
```

### Step 3: Implement Chat Completions

```go
func (a *YourProviderAdapter) handleChat(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
    // 1. Extract and validate payload
    payload, ok := req.Payload.(models.ChatCompletionRequest)
    if !ok {
        return models.ProviderResult{}, fmt.Errorf("invalid chat payload")
    }

    // 2. Transform to provider format
    providerReq := a.transformChatRequest(payload, model)

    // 3. Make HTTP request
    body, err := json.Marshal(providerReq)
    if err != nil {
        return models.ProviderResult{}, fmt.Errorf("marshal request: %w", err)
    }

    httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewReader(body))
    if err != nil {
        return models.ProviderResult{}, fmt.Errorf("create request: %w", err)
    }

    httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := a.httpClient.Do(httpReq)
    if err != nil {
        return models.ProviderResult{}, fmt.Errorf("provider request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return models.ProviderResult{}, fmt.Errorf("provider returned status %d", resp.StatusCode)
    }

    // 4. Parse provider response
    var providerResp providerChatResponse
    if err := json.NewDecoder(resp.Body).Decode(&providerResp); err != nil {
        return models.ProviderResult{}, fmt.Errorf("decode response: %w", err)
    }

    // 5. Transform to internal format
    return a.transformChatResponse(providerResp), nil
}
```

### Step 4: Define Provider Types

Define provider-specific request/response types:

```go
// Provider request types
type providerChatRequest struct {
    Model    string          `json:"model"`
    Messages []providerMessage `json:"messages"`
}

type providerMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

// Provider response types
type providerChatResponse struct {
    ID      string           `json:"id"`
    Choices []providerChoice `json:"choices"`
    Usage   providerUsage    `json:"usage"`
}

type providerChoice struct {
    Message providerMessage `json:"message"`
}

type providerUsage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}
```

### Step 5: Implement Transformers

```go
func (a *YourProviderAdapter) transformChatRequest(req models.ChatCompletionRequest, model string) providerChatRequest {
    messages := make([]providerMessage, len(req.Messages))
    for i, m := range req.Messages {
        messages[i] = providerMessage{
            Role:    m.Role,
            Content: m.Content,
        }
    }

    return providerChatRequest{
        Model:    model,
        Messages: messages,
    }
}

func (a *YourProviderAdapter) transformChatResponse(resp providerChatResponse) models.ProviderResult {
    var content string
    if len(resp.Choices) > 0 {
        content = resp.Choices[0].Message.Content
    }

    return models.ProviderResult{
        Model:    resp.Model,
        Provider: a.Name(),
        Status:   "success",
        Usage: models.Usage{
            PromptTokens:     resp.Usage.PromptTokens,
            CompletionTokens: resp.Usage.CompletionTokens,
            TotalTokens:      resp.Usage.TotalTokens,
            CostTotal:        a.calculateCost(resp.Usage),
        },
        Payload: models.ChatCompletionResponse{
            ID:     resp.ID,
            Object: "chat.completion",
            Model:  resp.Model,
            Choices: []models.ChatChoice{
                {
                    Index: 0,
                    Message: models.Message{
                        Role:    "assistant",
                        Content: content,
                    },
                },
            },
            Usage: models.Usage{
                PromptTokens:     resp.Usage.PromptTokens,
                CompletionTokens: resp.Usage.CompletionTokens,
                TotalTokens:      resp.Usage.TotalTokens,
                CostTotal:        a.calculateCost(resp.Usage),
            },
        },
    }
}
```

### Step 6: Register the Adapter

Update `cmd/rad-gateway/main.go` to register your adapter:

```go
func main() {
    cfg := config.Load()

    // Register all adapters
    registry := provider.NewRegistry(
        provider.NewMockAdapter(),
        provider.NewYourProviderAdapter(),  // Add your adapter
    )

    router := routing.New(registry, cfg.ModelRoutes, cfg.RetryBudget)
    // ... rest of initialization
}
```

### Step 7: Configure Model Routes

Update `internal/config/config.go` to add model routes:

```go
func loadModelRoutes() map[string][]Candidate {
    return map[string][]Candidate{
        "gpt-4o-mini": {
            {Provider: "mock", Model: "gpt-4o-mini", Weight: 80},
        },
        "your-model": {
            {Provider: "yourprovider", Model: "actual-model-name", Weight: 100},
        },
    }
}
```

## Transformer Patterns

### Request Transformation

Transform internal request to provider format:

```go
// Pattern 1: Direct mapping
func transformDirect(req models.ChatCompletionRequest) providerRequest {
    return providerRequest{
        Model:    req.Model,
        Messages: convertMessages(req.Messages),
    }
}

// Pattern 2: Format conversion
func transformWithConversion(req models.ChatCompletionRequest) providerRequest {
    // Provider uses different message format
    messages := make([]providerMessage, len(req.Messages))
    for i, m := range req.Messages {
        messages[i] = providerMessage{
            Role:    mapRole(m.Role),  // Convert role names
            Content: m.Content,
        }
    }
    return providerRequest{Messages: messages}
}

// Pattern 3: Feature mapping
func transformWithFeatures(req models.ChatCompletionRequest) providerRequest {
    return providerRequest{
        Model:       req.Model,
        Messages:    convertMessages(req.Messages),
        Temperature: req.Temperature,
        MaxTokens:   req.MaxTokens,
        // Only include supported features
    }
}
```

### Response Transformation

Transform provider response to internal format:

```go
// Pattern 1: Standard completion
func transformStandardResponse(resp providerResponse) models.ProviderResult {
    return models.ProviderResult{
        Model:    resp.Model,
        Provider: a.Name(),
        Status:   "success",
        Usage:    transformUsage(resp.Usage),
        Payload:  transformPayload(resp),
    }
}

// Pattern 2: Error handling
func transformWithErrorHandling(resp providerResponse, httpStatus int) (models.ProviderResult, error) {
    if httpStatus != http.StatusOK {
        return models.ProviderResult{}, fmt.Errorf("provider error: %s", resp.Error.Message)
    }
    return transformStandardResponse(resp), nil
}
```

## Testing Your Adapter

### Unit Tests

Create `internal/provider/yourprovider_test.go`:

```go
package provider

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "radgateway/internal/models"
)

func TestYourProviderAdapter_Execute_Chat(t *testing.T) {
    // Create test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        if r.Header.Get("Authorization") != "Bearer test-key" {
            t.Error("missing authorization header")
        }

        // Return mock response
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]any{
            "id": "test-id",
            "choices": []map[string]any{
                {"message": map[string]any{"role": "assistant", "content": "Hello"}},
            },
            "usage": map[string]any{
                "prompt_tokens": 10,
                "completion_tokens": 5,
                "total_tokens": 15,
            },
        })
    }))
    defer server.Close()

    // Create adapter with test server URL
    adapter := &YourProviderAdapter{
        apiKey:  "test-key",
        baseURL: server.URL,
        httpClient: &http.Client{},
    }

    // Execute request
    req := models.ProviderRequest{
        APIType: "chat",
        Model:   "test-model",
        Payload: models.ChatCompletionRequest{
            Messages: []models.Message{{Role: "user", Content: "Hi"}},
        },
    }

    result, err := adapter.Execute(context.Background(), req, "test-model")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if result.Provider != "yourprovider" {
        t.Errorf("expected provider yourprovider, got %s", result.Provider)
    }
}
```

### Integration Tests

Test the full flow with your adapter:

```go
func TestIntegration_YourProvider(t *testing.T) {
    // Skip if no API key
    if os.Getenv("YOUR_PROVIDER_API_KEY") == "" {
        t.Skip("YOUR_PROVIDER_API_KEY not set")
    }

    // Setup
    registry := provider.NewRegistry(provider.NewYourProviderAdapter())
    routes := map[string][]provider.Candidate{
        "test-model": {{Name: "yourprovider", Model: "actual-model", Weight: 100}},
    }
    router := routing.New(registry, routes, 1)
    gateway := core.New(router, usage.NewSink(100), trace.NewStore(100))

    // Execute
    ctx := context.Background()
    result, _, err := gateway.Handle(ctx, "chat", "test-model", models.ChatCompletionRequest{
        Messages: []models.Message{{Role: "user", Content: "Hello"}},
    })

    if err != nil {
        t.Fatalf("request failed: %v", err)
    }

    if result.Status != "success" {
        t.Errorf("expected success, got %s", result.Status)
    }
}
```

## Common Pitfalls

### 1. Context Cancellation

Always respect context cancellation:

```go
// BAD: Ignores context
resp, err := httpClient.Do(req)

// GOOD: Uses context-enabled request
req, err := http.NewRequestWithContext(ctx, "POST", url, body)
resp, err := httpClient.Do(req)
```

### 2. Resource Leaks

Always close response bodies:

```go
// BAD: Body not closed
resp, _ := httpClient.Do(req)
return parseResponse(resp)

// GOOD: Body properly closed
resp, err := httpClient.Do(req)
if err != nil {
    return nil, err
}
defer resp.Body.Close()
return parseResponse(resp)
```

### 3. Type Assertions

Always check type assertions:

```go
// BAD: Unchecked type assertion
payload := req.Payload.(models.ChatCompletionRequest)

// GOOD: Safe type assertion
payload, ok := req.Payload.(models.ChatCompletionRequest)
if !ok {
    return models.ProviderResult{}, fmt.Errorf("invalid payload type")
}
```

### 4. Error Wrapping

Wrap errors for better debugging:

```go
// BAD: Raw error
if err != nil {
    return nil, err
}

// GOOD: Wrapped error with context
if err != nil {
    return nil, fmt.Errorf("transform request: %w", err)
}
```

## Best Practices

1. **Use a custom HTTP client** with appropriate timeouts
2. **Implement retry logic** for transient failures
3. **Add request/response logging** (without sensitive data)
4. **Support all API types** that the provider offers
5. **Document provider-specific behavior** in comments
6. **Add metrics** for provider latency and errors
7. **Handle rate limits** with appropriate backoff

## Example: Complete Adapter

See `internal/provider/mock.go` for a complete, working example that demonstrates:

- Multiple API type handling
- Proper response formatting
- Usage calculation
- Error handling

## Next Steps

- [Configuration Reference](../reference/adapter-config.md) - Learn about route configuration
- [Troubleshooting Guide](../troubleshooting/adapters.md) - Debug common issues
