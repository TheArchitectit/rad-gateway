# Provider Adapter Troubleshooting Guide

This guide helps diagnose and resolve common issues with provider adapters in Brass Relay.

## Quick Diagnostics

### Enable Debug Logging

```bash
RAD_LOG_LEVEL=debug ./rad-gateway
```

### Check Adapter Registration

```bash
curl http://localhost:8090/api/v0/admin/config | jq '.models'
```

### Test with Mock Adapter

```bash
curl -X POST http://localhost:8090/v1/chat/completions \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4o-mini", "messages": [{"role": "user", "content": "Hello"}]}'
```

## Common Issues

### Issue: `adapter not found: {name}`

**Error Message:**
```
Error: adapter not found: openai
```

**Root Cause:**
The router is trying to use an adapter that is either:
1. Not registered in the registry
2. Registered with a different name
3. Not compiled into the binary

**Diagnosis:**
```go
// Check registered adapters
registry := provider.NewRegistry(
    provider.NewMockAdapter(),
    // Is your adapter here?
)
```

**Solution:**

1. Verify adapter is registered in `cmd/rad-gateway/main.go`:
```go
registry := provider.NewRegistry(
    provider.NewMockAdapter(),
    provider.NewOpenAIAdapter(),  // Ensure this exists
)
```

2. Check adapter `Name()` matches route:
```go
// In your adapter
func (a *OpenAIAdapter) Name() string {
    return "openai"  // Must match route
}

// In routes
{Provider: "openai", Model: "gpt-4o", Weight: 100}  // Matches Name()
```

3. Rebuild the application:
```bash
go build -o rad-gateway ./cmd/rad-gateway
```

---

### Issue: `unsupported api type: {type}`

**Error Message:**
```
Error: unsupported api type: embeddings
```

**Root Cause:**
The adapter doesn't implement the requested API type.

**Diagnosis:**
Check your adapter's `Execute` method:
```go
func (a *YourAdapter) Execute(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
    switch req.APIType {
    case "chat":
        return a.handleChat(ctx, req, model)
    // Missing "embeddings" case!
    default:
        return models.ProviderResult{}, fmt.Errorf("unsupported api type: %s", req.APIType)
    }
}
```

**Solution:**

Add the missing API type handler:
```go
func (a *YourAdapter) Execute(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
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

**Supported API Types:**
- `chat` - Chat completions
- `responses` - OpenAI Responses API
- `messages` - Anthropic Messages API
- `embeddings` - Text embeddings
- `images` - Image generation
- `transcriptions` - Audio transcription
- `gemini` - Google Gemini API

---

### Issue: `all route attempts failed`

**Error Message:**
```
Error: all route attempts failed
```

**Root Cause:**
All configured providers failed to respond successfully.

**Diagnosis:**

1. Check attempt details in response:
```go
result, attempts, err := gateway.Handle(ctx, "chat", "gpt-4o", payload)
// attempts contains details of each failure
```

2. Enable debug logging to see individual errors:
```bash
RAD_LOG_LEVEL=debug
```

**Common Causes & Solutions:**

| Cause | Check | Solution |
|-------|-------|----------|
| API key missing | `echo $OPENAI_API_KEY` | Set the environment variable |
| API key invalid | Provider error in logs | Generate new key from provider dashboard |
| Network issues | `curl -I https://api.openai.com` | Check firewall/proxy settings |
| Rate limiting | HTTP 429 in logs | Implement exponential backoff |
| Model not available | Provider error message | Use available model ID |

**Solution:**

1. Test provider directly:
```bash
curl https://api.openai.com/v1/chat/completions \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4o", "messages": [{"role": "user", "content": "Hi"}]}'
```

2. Add fallback providers:
```go
"gpt-4o": {
    {Provider: "openai", Model: "gpt-4o", Weight: 100},
    {Provider: "anthropic", Model: "claude-3-5-sonnet", Weight: 50},  // Fallback
},
```

---

### Issue: `invalid chat payload`

**Error Message:**
```
Error: invalid chat payload
```

**Root Cause:**
Type assertion failed when extracting payload from `ProviderRequest`.

**Diagnosis:**
Check payload handling:
```go
payload, ok := req.Payload.(models.ChatCompletionRequest)
if !ok {
    return models.ProviderResult{}, fmt.Errorf("invalid chat payload")
}
```

**Solution:**

1. Verify payload type matches API type:
```go
// For "chat" API type, payload must be ChatCompletionRequest
type ChatCompletionRequest struct {
    Model    string    `json:"model"`
    Messages []Message `json:"messages"`
    Stream   bool      `json:"stream,omitempty"`
    User     string    `json:"user,omitempty"`
}
```

2. Ensure handler uses correct type:
```go
case "chat":
    payload, ok := req.Payload.(models.ChatCompletionRequest)
    // NOT models.ResponseRequest or other type
```

---

### Issue: Response has empty content

**Error Message:**
None (silent failure - response appears but has no content)

**Root Cause:**
Response transformation is not correctly mapping provider fields.

**Diagnosis:**

1. Add logging to adapter:
```go
func (a *YourAdapter) handleChat(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
    // ... make request ...
    var providerResp providerChatResponse
    json.NewDecoder(resp.Body).Decode(&providerResp)

    // Log the response
    log.Printf("Provider response: %+v", providerResp)

    return a.transformChatResponse(providerResp), nil
}
```

2. Check field names match:
```go
// Provider might use different field names
type providerChatResponse struct {
    ID      string `json:"id"`
    Choices []struct {
        // Provider uses "text" not "content"
        Message struct {
            Role    string `json:"role"`
            Text    string `json:"text"`  // Not "content"!
        } `json:"message"`
    } `json:"choices"`
}
```

**Solution:**

Fix the transformation:
```go
func (a *YourAdapter) transformChatResponse(resp providerChatResponse) models.ProviderResult {
    var content string
    if len(resp.Choices) > 0 {
        // Map provider field to standard field
        content = resp.Choices[0].Message.Text  // Provider-specific
    }

    return models.ProviderResult{
        Payload: models.ChatCompletionResponse{
            Choices: []models.ChatChoice{
                {
                    Message: models.Message{
                        Role:    "assistant",
                        Content: content,  // Standard field
                    },
                },
            },
        },
    }
}
```

---

### Issue: Request timeouts

**Error Message:**
```
Error: context deadline exceeded
```

**Root Cause:**
Request took longer than the configured timeout.

**Diagnosis:**

1. Check HTTP client timeout:
```go
type YourAdapter struct {
    httpClient *http.Client
}

func NewYourAdapter() *YourAdapter {
    return &YourAdapter{
        httpClient: &http.Client{
            Timeout: 30 * time.Second,  // Is this too short?
        },
    }
}
```

2. Check provider latency:
```bash
time curl https://api.provider.com/v1/chat/completions ...
```

**Solution:**

1. Increase timeout:
```go
httpClient: &http.Client{
    Timeout: 60 * time.Second,  // Increase for slow providers
}
```

2. Use context with timeout from gateway:
```go
// Gateway already sets reasonable timeout
// Just ensure your adapter respects context cancellation
req, err := http.NewRequestWithContext(ctx, "POST", url, body)
```

---

### Issue: Memory leaks

**Error Message:**
Application consumes increasing memory over time.

**Root Cause:**
HTTP response bodies not being closed.

**Diagnosis:**
Check adapter code:
```go
// BAD - Body not closed
resp, err := a.httpClient.Do(req)
if err != nil {
    return models.ProviderResult{}, err
}
return parseResponse(resp)

// GOOD - Body properly closed
resp, err := a.httpClient.Do(req)
if err != nil {
    return models.ProviderResult{}, err
}
defer resp.Body.Close()
return parseResponse(resp)
```

**Solution:**

Always close response bodies:
```go
resp, err := a.httpClient.Do(req)
if err != nil {
    return models.ProviderResult{}, fmt.Errorf("provider request: %w", err)
}
defer resp.Body.Close()  // Critical!

// Now safe to read body
body, err := io.ReadAll(resp.Body)
```

---

### Issue: Authentication failures

**Error Message:**
```
Error: provider returned status 401
```

**Root Cause:**
Invalid or missing API key.

**Diagnosis:**

1. Check environment variable:
```bash
echo $YOUR_PROVIDER_API_KEY
```

2. Check header format:
```go
// Verify correct header format
req.Header.Set("Authorization", "Bearer "+a.apiKey)
```

3. Test with curl:
```bash
curl -H "Authorization: Bearer $YOUR_PROVIDER_API_KEY" \
  https://api.provider.com/v1/models
```

**Solution:**

1. Set correct environment variable:
```bash
export YOUR_PROVIDER_API_KEY="sk-..."
```

2. Verify header format matches provider requirements:
```go
// OpenAI style
req.Header.Set("Authorization", "Bearer "+a.apiKey)

// Anthropic style
req.Header.Set("x-api-key", a.apiKey)
req.Header.Set("anthropic-version", "2023-06-01")

// Gemini style
req.Header.Set("x-goog-api-key", a.apiKey)
```

---

## Debugging Techniques

### Add Structured Logging

```go
func (a *YourAdapter) Execute(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
    log.Printf("[%s] Executing %s request for model %s", a.Name(), req.APIType, model)

    start := time.Now()
    result, err := a.executeInternal(ctx, req, model)
    duration := time.Since(start)

    if err != nil {
        log.Printf("[%s] Request failed after %v: %v", a.Name(), duration, err)
        return result, err
    }

    log.Printf("[%s] Request succeeded in %v", a.Name(), duration)
    return result, nil
}
```

### Request/Response Logging

```go
func (a *YourAdapter) logRequest(reqBody []byte) {
    if os.Getenv("DEBUG_ADAPTER") == "1" {
        log.Printf("Request: %s", string(reqBody))
    }
}

func (a *YourAdapter) logResponse(respBody []byte) {
    if os.Getenv("DEBUG_ADAPTER") == "1" {
        log.Printf("Response: %s", string(respBody))
    }
}
```

### Trace Integration

```go
func (a *YourAdapter) Execute(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
    // Get trace ID from context
    traceID := middleware.GetTraceID(ctx)

    // Log to trace store if available
    // This enables request tracing in the admin dashboard

    result, err := a.executeInternal(ctx, req, model)

    // Add trace event
    // traceStore.Add(trace.Event{...})

    return result, err
}
```

## Testing Checklist

When implementing a new adapter, verify:

- [ ] Adapter is registered in `main.go`
- [ ] `Name()` returns correct provider identifier
- [ ] All supported API types have handlers
- [ ] Payload type assertions are checked
- [ ] HTTP response bodies are closed with `defer`
- [ ] Context cancellation is respected
- [ ] Errors are wrapped with context
- [ ] Response fields are correctly mapped
- [ ] Usage/cost information is populated
- [ ] Model routes are configured
- [ ] Unit tests pass
- [ ] Integration tests pass (with real API key)

## Common Error Patterns

### Pattern 1: Silent Failures

```go
// BAD - Silent failure on type assertion
payload := req.Payload.(models.ChatCompletionRequest)  // Panic on failure

// GOOD - Explicit error handling
payload, ok := req.Payload.(models.ChatCompletionRequest)
if !ok {
    return models.ProviderResult{}, fmt.Errorf("invalid payload type: expected ChatCompletionRequest")
}
```

### Pattern 2: Leaky Abstractions

```go
// BAD - Exposing provider errors directly
if resp.StatusCode != 200 {
    return models.ProviderResult{}, fmt.Errorf("OpenAI error: %s", respBody)
}

// GOOD - Wrapping errors
if resp.StatusCode != 200 {
    return models.ProviderResult{}, fmt.Errorf("provider request failed: status=%d", resp.StatusCode)
}
```

### Pattern 3: Ignoring Context

```go
// BAD - Ignoring context
req, _ := http.NewRequest("POST", url, body)

// GOOD - Respecting context
req, _ := http.NewRequestWithContext(ctx, "POST", url, body)
```

## Getting Help

If you're still stuck:

1. Check the [Architecture Guide](../architecture/provider-adapters.md) for design patterns
2. Review the [Implementation Guide](../guides/implementing-adapters.md) for examples
3. Look at `internal/provider/mock.go` for a working reference
4. Enable debug logging and examine the full request/response flow
5. Test the provider API directly with curl to isolate issues
