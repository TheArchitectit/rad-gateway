# Gemini Adapter Design Document

## Overview

This document describes the architecture for the Google Gemini provider adapter in RAD Gateway. The adapter enables RAD Gateway to communicate with Google's Gemini API by translating between the OpenAI-compatible format used internally by RAD Gateway and Gemini's GenerateContent API format.

**Status:** Design Phase
**Target:** RAD Gateway Alpha
**Author:** Architecture Team
**Last Updated:** 2026-02-16

---

## Architecture Overview

### Design Philosophy

The Gemini adapter follows RAD Gateway's established adapter pattern (Option B), creating a completely new adapter package under `internal/provider/gemini/`. This approach provides clean separation of concerns while maintaining consistency with the existing OpenAI adapter pattern through separate `adapter.go` and `transformer.go` files.

The adapter implements bidirectional transformation between:
- **Internal Format:** OpenAI-compatible format (used by RAD Gateway)
- **Gemini Format:** Google Gemini GenerateContent API format

### Component Structure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           GEMINI ADAPTER                                    │
│                                                                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐      │
│  │   Gemini        │  │   Gemini        │  │   Gemini                │      │
│  │   Adapter       │──│   Request       │──│   Response              │      │
│  │   (Entry Point) │  │   Transformer   │  │   Transformer           │      │
│  └────────┬────────┘  └─────────────────┘  └─────────────────────────┘      │
│           │                                                                 │
│           │                    ┌─────────────────┐                           │
│           └────────────────────│   Gemini        │                           │
│                                │   Stream        │                           │
│                                │   Transformer   │                           │
│                                └─────────────────┘                           │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                      GOOGLE GEMINI API                                      │
│                                                                             │
│  Endpoints:                                                                 │
│  - POST /v1beta/models/{model}:generateContent       (non-streaming)      │
│  - POST /v1beta/models/{model}:streamGenerateContent (streaming)          │
│                                                                             │
│  Auth: x-goog-api-key header (preferred) or ?key= query param (fallback)  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### File Structure

```
internal/provider/gemini/
├── adapter.go           # Main adapter implementation
├── adapter_test.go      # Unit tests
├── transformer.go       # Request/response transformation
└── transformer_test.go  # Transformer tests
```

---

## Key Differences from OpenAI Adapter

| Aspect | OpenAI | Gemini |
|--------|--------|--------|
| **Endpoint Pattern** | `/v1/chat/completions` | `/v1beta/models/{model}:generateContent` |
| **Streaming Endpoint** | Same endpoint with `stream: true` | Separate `:streamGenerateContent` endpoint |
| **Auth Header** | `Authorization: Bearer <key>` | `x-goog-api-key: <key>` (preferred) |
| **Auth Query Param** | Not used | `?key=<key>` (fallback) |
| **Message Format** | `messages: [{role, content}]` | `contents: [{role, parts}]` |
| **Roles** | `system`, `user`, `assistant` | `user`, `model` (no system role) |
| **Content Structure** | `content: "string"` | `parts: [{text: "..."}]` |
| **Generation Config** | Top-level params (`temperature`, `max_tokens`) | Separate `generationConfig` object |
| **Safety Settings** | Not applicable | `safetySettings` array required |
| **Response Structure** | `choices[].message.content` | `candidates[].content.parts[].text` |

---

## Request Transformation (OpenAI → Gemini)

### Transformation Pipeline

```
┌────────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│ OpenAI Format  │───▶│  GeminiRequest       │───▶│ Gemini API      │
│ (Internal)     │    │  Transformer         │    │ Format          │
└────────────────┘    └──────────────────────┘    └─────────────────┘
```

### OpenAI Request Format (Internal)

```json
{
  "model": "gemini-1.5-flash",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello, Gemini!"}
  ],
  "stream": false,
  "temperature": 0.7,
  "max_tokens": 1024,
  "top_p": 0.9
}
```

### Gemini Request Format (Output)

```json
{
  "contents": [
    {
      "role": "user",
      "parts": [
        {"text": "You are a helpful assistant.\n\nHello, Gemini!"}
      ]
    }
  ],
  "generationConfig": {
    "temperature": 0.7,
    "maxOutputTokens": 1024,
    "topP": 0.9
  },
  "safetySettings": [
    {
      "category": "HARM_CATEGORY_DANGEROUS_CONTENT",
      "threshold": "BLOCK_ONLY_HIGH"
    },
    {
      "category": "HARM_CATEGORY_HATE_SPEECH",
      "threshold": "BLOCK_ONLY_HIGH"
    },
    {
      "category": "HARM_CATEGORY_HARASSMENT",
      "threshold": "BLOCK_ONLY_HIGH"
    },
    {
      "category": "HARM_CATEGORY_SEXUALLY_EXPLICIT",
      "threshold": "BLOCK_ONLY_HIGH"
    }
  ]
}
```

### Transformation Logic

```go
// GeminiRequestTransformer performs OpenAI → Gemini conversion
type GeminiRequestTransformer struct {
    config ProviderConfig
}

func (t *GeminiRequestTransformer) Transform(body io.Reader) (io.Reader, error) {
    // 1. Parse OpenAI format
    var openaiReq OpenAIChatRequest
    if err := json.Unmarshal(data, &openaiReq); err != nil {
        return nil, err
    }

    // 2. Initialize Gemini request
    geminiReq := GeminiGenerateRequest{
        Contents: make([]GeminiContent, 0),
    }

    // 3. Transform messages to contents format
    // Gemini only supports "user" and "model" roles
    // System messages are prepended to the first user message
    var systemContent string
    var contents []GeminiContent
    var currentContent GeminiContent
    var currentParts []GeminiPart

    for _, msg := range openaiReq.Messages {
        if msg.Role == "system" {
            // Accumulate system messages
            if systemContent != "" {
                systemContent += "\n\n"
            }
            systemContent += msg.Content
            continue
        }

        // Map OpenAI "assistant" role to Gemini "model" role
        geminiRole := msg.Role
        if msg.Role == "assistant" {
            geminiRole = "model"
        }

        // Start new content block if role changes
        if currentContent.Role != "" && currentContent.Role != geminiRole {
            currentContent.Parts = currentParts
            contents = append(contents, currentContent)
            currentContent = GeminiContent{}
            currentParts = nil
        }

        currentContent.Role = geminiRole

        // Prepend system content to first user message
        content := msg.Content
        if systemContent != "" && geminiRole == "user" {
            content = systemContent + "\n\n" + content
            systemContent = "" // Clear after use
        }

        currentParts = append(currentParts, GeminiPart{Text: content})
    }

    // Append final content block
    if currentContent.Role != "" {
        currentContent.Parts = currentParts
        contents = append(contents, currentContent)
    }

    geminiReq.Contents = contents

    // 4. Build generationConfig from top-level params
    genConfig := GeminiGenerationConfig{}
    if openaiReq.Temperature != nil {
        genConfig.Temperature = *openaiReq.Temperature
    }
    if openaiReq.MaxTokens != nil {
        genConfig.MaxOutputTokens = *openaiReq.MaxTokens
    }
    if openaiReq.TopP != nil {
        genConfig.TopP = *openaiReq.TopP
    }
    if openaiReq.TopK != nil {
        genConfig.TopK = *openaiReq.TopK
    }
    geminiReq.GenerationConfig = genConfig

    // 5. Add required safety settings
    geminiReq.SafetySettings = []GeminiSafetySetting{
        {Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_ONLY_HIGH"},
        {Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_ONLY_HIGH"},
        {Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_ONLY_HIGH"},
        {Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_ONLY_HIGH"},
    }

    // 6. Serialize and return
    return json.Marshal(geminiReq)
}
```

### Gemini Request Types

```go
// GeminiGenerateRequest represents the generateContent API request
type GeminiGenerateRequest struct {
    Contents         []GeminiContent         `json:"contents"`
    GenerationConfig GeminiGenerationConfig  `json:"generationConfig,omitempty"`
    SafetySettings   []GeminiSafetySetting   `json:"safetySettings,omitempty"`
    Tools            []GeminiTool            `json:"tools,omitempty"`
    ToolConfig       *GeminiToolConfig       `json:"toolConfig,omitempty"`
}

type GeminiContent struct {
    Role  string        `json:"role"` // "user" or "model"
    Parts []GeminiPart  `json:"parts"`
}

type GeminiPart struct {
    Text string `json:"text,omitempty"`
    // Future: InlineData, FileData, VideoMetadata, etc.
}

type GeminiGenerationConfig struct {
    Temperature     float64 `json:"temperature,omitempty"`
    MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
    TopP            float64 `json:"topP,omitempty"`
    TopK            int     `json:"topK,omitempty"`
    StopSequences   []string `json:"stopSequences,omitempty"`
    CandidateCount  int     `json:"candidateCount,omitempty"`
}

type GeminiSafetySetting struct {
    Category  string `json:"category"`
    Threshold string `json:"threshold"`
}
```

---

## Response Transformation (Gemini → OpenAI)

### Non-Streaming Response Transformation

```
┌─────────────────┐    ┌──────────────────────┐    ┌────────────────┐
│ Gemini API      │───▶│ GeminiResponse       │───▶│ OpenAI Format  │
│ Response        │    │ Transformer          │    │ (Internal)     │
└─────────────────┘    └──────────────────────┘    └────────────────┘
```

### Gemini Response Format (Input)

```json
{
  "candidates": [
    {
      "content": {
        "role": "model",
        "parts": [
          {"text": "Hello! How can I help you today?"}
        ]
      },
      "finishReason": "STOP",
      "index": 0,
      "safetyRatings": [
        {
          "category": "HARM_CATEGORY_DANGEROUS_CONTENT",
          "probability": "NEGLIGIBLE"
        }
      ]
    }
  ],
  "usageMetadata": {
    "promptTokenCount": 12,
    "candidatesTokenCount": 9,
    "totalTokenCount": 21
  }
}
```

### OpenAI Response Format (Output)

```json
{
  "id": "gemini-generated-id",
  "object": "chat.completion",
  "created": 1704067200,
  "model": "gemini-1.5-flash",
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
func (t *GeminiResponseTransformer) Transform(body io.Reader) (io.Reader, error) {
    // 1. Parse Gemini response
    var geminiResp GeminiGenerateResponse
    if err := json.Unmarshal(data, &geminiResp); err != nil {
        return nil, err
    }

    // 2. Extract content from candidates
    var content string
    var finishReason string
    if len(geminiResp.Candidates) > 0 {
        candidate := geminiResp.Candidates[0]
        for _, part := range candidate.Content.Parts {
            content += part.Text
        }
        finishReason = mapFinishReason(candidate.FinishReason)
    }

    // 3. Build OpenAI-compatible response
    openaiResp := OpenAIChatResponse{
        ID:      generateID("gemini"),
        Object:  "chat.completion",
        Created: time.Now().Unix(),
        Model:   t.model, // Preserved from request
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
            PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
            CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
            TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
        },
    }

    return json.Marshal(openaiResp)
}

// Finish reason mapping
func mapFinishReason(geminiReason string) string {
    switch geminiReason {
    case "STOP":
        return "stop"
    case "MAX_TOKENS":
        return "length"
    case "SAFETY":
        return "content_filter"
    case "RECITATION":
        return "content_filter"
    case "OTHER":
        return "stop"
    default:
        return "stop"
    }
}
```

### Gemini Response Types

```go
// GeminiGenerateResponse represents the generateContent API response
type GeminiGenerateResponse struct {
    Candidates    []GeminiCandidate    `json:"candidates"`
    UsageMetadata GeminiUsageMetadata  `json:"usageMetadata"`
    PromptFeedback *GeminiPromptFeedback `json:"promptFeedback,omitempty"`
}

type GeminiCandidate struct {
    Content       GeminiContent       `json:"content"`
    FinishReason  string              `json:"finishReason"` // "STOP", "MAX_TOKENS", "SAFETY", "RECITATION", "OTHER"
    Index         int                 `json:"index"`
    SafetyRatings []GeminiSafetyRating `json:"safetyRatings,omitempty"`
}

type GeminiSafetyRating struct {
    Category    string `json:"category"`
    Probability string `json:"probability"` // "NEGLIGIBLE", "LOW", "MEDIUM", "HIGH"
    Blocked     bool   `json:"blocked,omitempty"`
}

type GeminiUsageMetadata struct {
    PromptTokenCount     int `json:"promptTokenCount"`
    CandidatesTokenCount int `json:"candidatesTokenCount"`
    TotalTokenCount      int `json:"totalTokenCount"`
}

type GeminiPromptFeedback struct {
    SafetyRatings []GeminiSafetyRating `json:"safetyRatings"`
    BlockReason   string               `json:"blockReason,omitempty"`
}
```

---

## Streaming Event Handling

### Gemini Streaming Architecture

Gemini uses Server-Sent Events (SSE) with a different format than OpenAI:

```
┌────────────────────────────────────────────────────────────────────────────┐
│                    GEMINI STREAMING EVENT FLOW                             │
│                                                                            │
│  data: {"candidates": [{"content": {"parts": [{"text": "Hello"}], ...}}]}  │
│                                                                            │
│  data: {"candidates": [{"content": {"parts": [{"text": " there"}], ...}}]} │
│                                                                            │
│  data: {"candidates": [{"content": {"parts": [{"text": "!"}], ...},         │
│        "usageMetadata": {"promptTokenCount": 12, ...},                    │
│        "finishReason": "STOP"}]}                                          │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

### Streaming Transformation Logic

```go
// GeminiStreamTransformer handles SSE event transformation
type GeminiStreamTransformer struct {
    messageID    string
    model        string
    created      int64
    index        int
    buffer       strings.Builder
    accumulatedContent string
}

func (t *GeminiStreamTransformer) TransformChunk(chunk []byte) ([]byte, error) {
    // Parse SSE data line
    data := parseSSEData(chunk)
    if data == "" || data == "[DONE]" {
        return nil, nil
    }

    // Parse Gemini streaming response
    var geminiResp GeminiGenerateResponse
    if err := json.Unmarshal([]byte(data), &geminiResp); err != nil {
        return nil, fmt.Errorf("unmarshaling gemini stream chunk: %w", err)
    }

    // Extract text from candidates
    var content string
    var finishReason *string

    if len(geminiResp.Candidates) > 0 {
        candidate := geminiResp.Candidates[0]
        for _, part := range candidate.Content.Parts {
            content += part.Text
        }

        // Calculate delta (new content since last chunk)
        delta := content[len(t.accumulatedContent):]
        t.accumulatedContent = content

        if candidate.FinishReason != "" {
            mapped := mapFinishReason(candidate.FinishReason)
            finishReason = &mapped
        }

        // Build OpenAI-style stream chunk
        openaiChunk := OpenAIStreamChunk{
            ID:      t.messageID,
            Object:  "chat.completion.chunk",
            Created: t.created,
            Model:   t.model,
            Choices: []OpenAIStreamChoice{
                {
                    Index: t.index,
                    Delta: OpenAIMessageDelta{
                        Content: delta,
                    },
                    FinishReason: finishReason,
                },
            },
        }

        return formatSSEChunk(openaiChunk)
    }

    return nil, nil
}

func (t *GeminiStreamTransformer) IsDoneMarker(chunk []byte) bool {
    // Check for Gemini completion indicators
    return bytes.Contains(chunk, []byte(`"finishReason":`)) ||
           bytes.Contains(chunk, []byte("[DONE]"))
}
```

---

## Endpoint and URL Transformation

### Endpoint Mapping (AxonHub Pattern)

```go
func (t *GeminiRequestTransformer) TransformURL(req *http.Request) error {
    // Extract model from request body or URL
    model := extractModel(req)

    // Map OpenAI paths to Gemini paths
    path := req.URL.Path

    switch path {
    case "/v1/chat/completions":
        // Determine streaming from request body
        isStreaming := checkStreaming(req)

        if isStreaming {
            req.URL.Path = fmt.Sprintf("/v1beta/models/%s:streamGenerateContent", model)
        } else {
            req.URL.Path = fmt.Sprintf("/v1beta/models/%s:generateContent", model)
        }

    default:
        // Pass through as-is with model substitution
        req.URL.Path = strings.Replace(path, "{model}", model, -1)
    }

    // Set host and scheme
    req.URL.Scheme = "https"
    req.URL.Host = strings.TrimPrefix(t.config.BaseURL, "https://")

    return nil
}
```

### URL Pattern Summary

| Operation | OpenAI Path | Gemini Path |
|-----------|-------------|-------------|
| Chat Completion | `/v1/chat/completions` | `/v1beta/models/{model}:generateContent` |
| Streaming Chat | `/v1/chat/completions` (with `stream: true`) | `/v1beta/models/{model}:streamGenerateContent` |

---

## Authentication Approach

### Header Configuration (Preferred)

Gemini supports two authentication methods:

```go
func (t *GeminiRequestTransformer) TransformHeaders(req *http.Request) error {
    // Method 1: x-goog-api-key header (preferred)
    if t.config.APIKey != "" {
        req.Header.Set("x-goog-api-key", t.config.APIKey)
    }

    // Content type
    req.Header.Set("Content-Type", "application/json")

    // Optional: Custom headers from config
    for key, value := range t.config.Headers {
        req.Header.Set(key, value)
    }

    return nil
}
```

### Query Parameter Fallback

```go
func (t *GeminiRequestTransformer) TransformURL(req *http.Request) error {
    // Method 2: API key as query parameter (fallback)
    if t.config.APIKey != "" {
        query := req.URL.Query()
        query.Set("key", t.config.APIKey)
        req.URL.RawQuery = query.Encode()
    }

    // ... rest of URL transformation
    return nil
}
```

### Auth Configuration

```go
// Gemini adapter configuration
type GeminiConfig struct {
    APIKey        string
    BaseURL       string  // Default: https://generativelanguage.googleapis.com
    Version       string  // Default: v1beta
    Timeout       time.Duration
    MaxRetries    int
    AuthMethod    string  // "header" (default) or "query"
}

// Environment variables
// GEMINI_API_KEY=<your-api-key>
// GEMINI_BASE_URL=https://generativelanguage.googleapis.com (optional)
// GEMINI_AUTH_METHOD=header (or "query" for fallback)
```

### Auth Comparison

| Provider | Method | Header/Param | Format |
|----------|--------|--------------|--------|
| OpenAI | Header | `Authorization` | `Bearer <key>` |
| Anthropic | Header | `x-api-key` | `<key>` |
| Gemini (Preferred) | Header | `x-goog-api-key` | `<key>` |
| Gemini (Fallback) | Query | `?key=` | `<key>` |

---

## Error Handling

### Gemini Error Format

```json
{
  "error": {
    "code": 400,
    "message": "Invalid value at 'contents' (type.googleapis.com/google.ai.generativelanguage.v1beta.Content), \"invalid\"",
    "status": "INVALID_ARGUMENT"
  }
}
```

### Error Transformation

```go
func (t *GeminiResponseTransformer) TransformError(body []byte) error {
    var geminiErr struct {
        Error struct {
            Code    int    `json:"code"`
            Message string `json:"message"`
            Status  string `json:"status"`
        } `json:"error"`
    }

    if err := json.Unmarshal(body, &geminiErr); err != nil {
        return fmt.Errorf("gemini error: %s", string(body))
    }

    // Map to OpenAI-style error
    return fmt.Errorf("gemini error (%d): %s", geminiErr.Error.Code, geminiErr.Error.Message)
}

// HTTP Status Code Mapping
var statusCodeMap = map[int]int{
    400: http.StatusBadRequest,       // INVALID_ARGUMENT
    401: http.StatusUnauthorized,     // UNAUTHENTICATED
    403: http.StatusForbidden,        // PERMISSION_DENIED
    404: http.StatusNotFound,         // NOT_FOUND
    429: http.StatusTooManyRequests,    // RESOURCE_EXHAUSTED
    500: http.StatusInternalServerError, // INTERNAL
    503: http.StatusServiceUnavailable, // UNAVAILABLE
    504: http.StatusGatewayTimeout,   // DEADLINE_EXCEEDED
}
```

---

## Complete Adapter Interface

```go
// GeminiAdapter implements the provider.Adapter interface
type GeminiAdapter struct {
    config            GeminiConfig
    httpClient        *http.Client
    reqTransformer    *GeminiRequestTransformer
    respTransformer   *GeminiResponseTransformer
    streamTransformer *GeminiStreamTransformer
}

func NewGeminiAdapter(apiKey string, opts ...GeminiOption) *GeminiAdapter {
    a := &GeminiAdapter{
        config: GeminiConfig{
            APIKey:     apiKey,
            BaseURL:    "https://generativelanguage.googleapis.com",
            Version:    "v1beta",
            Timeout:    60 * time.Second,
            MaxRetries: 3,
            AuthMethod: "header", // Default to header auth
        },
        reqTransformer:    NewGeminiRequestTransformer(),
        respTransformer:   NewGeminiResponseTransformer(),
        streamTransformer: NewGeminiStreamTransformer(),
    }

    for _, opt := range opts {
        opt(a)
    }

    a.httpClient = &http.Client{Timeout: a.config.Timeout}
    return a
}

func (a *GeminiAdapter) Name() string {
    return "gemini"
}

func (a *GeminiAdapter) Execute(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
    switch req.APIType {
    case "chat":
        return a.executeChat(ctx, req, model)
    case "gemini":
        // Gemini-specific API type for direct access
        return a.executeGemini(ctx, req, model)
    default:
        return models.ProviderResult{}, fmt.Errorf("unsupported api type: %s", req.APIType)
    }
}
```

---

## Testing Strategy

### Unit Tests

```go
func TestGeminiRequestTransformer_Transform(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected GeminiGenerateRequest
    }{
        {
            name: "transforms messages to contents with parts",
            input: `{"messages":[{"role":"user","content":"Hello"}]}`,
            expected: GeminiGenerateRequest{
                Contents: []GeminiContent{
                    {
                        Role:  "user",
                        Parts: []GeminiPart{{Text: "Hello"}},
                    },
                },
                SafetySettings: defaultSafetySettings(),
            },
        },
        {
            name: "maps assistant role to model",
            input: `{"messages":[{"role":"assistant","content":"Hi!"}]}`,
            expected: GeminiGenerateRequest{
                Contents: []GeminiContent{
                    {
                        Role:  "model",
                        Parts: []GeminiPart{{Text: "Hi!"}},
                    },
                },
                SafetySettings: defaultSafetySettings(),
            },
        },
        {
            name: "prepends system message to first user message",
            input: `{"messages":[{"role":"system","content":"Sys"},{"role":"user","content":"Hi"}]}`,
            expected: GeminiGenerateRequest{
                Contents: []GeminiContent{
                    {
                        Role:  "user",
                        Parts: []GeminiPart{{Text: "Sys\n\nHi"}},
                    },
                },
                SafetySettings: defaultSafetySettings(),
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            transformer := NewGeminiRequestTransformer()
            result, err := transformer.Transform(strings.NewReader(tt.input))
            // Assertions...
        })
    }
}
```

### Integration Tests

```go
func TestGeminiAdapter_Execute(t *testing.T) {
    // Create test server that mimics Gemini API
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        assert.Equal(t, "/v1beta/models/gemini-pro:generateContent", r.URL.Path)
        assert.Equal(t, "test-key", r.Header.Get("x-goog-api-key"))

        // Verify URL construction
        model := extractModelFromPath(r.URL.Path)
        assert.Equal(t, "gemini-pro", model)

        // Return mock Gemini response
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]any{
            "candidates": []map[string]any{
                {
                    "content": map[string]any{
                        "role": "model",
                        "parts": []map[string]any{
                            {"text": "Hello!"},
                        },
                    },
                    "finishReason": "STOP",
                },
            },
            "usageMetadata": map[string]int{
                "promptTokenCount":     10,
                "candidatesTokenCount": 2,
                "totalTokenCount":      12,
            },
        })
    }))
    defer server.Close()

    adapter := NewGeminiAdapter("test-key", WithBaseURL(server.URL))

    req := models.ProviderRequest{
        APIType: "chat",
        Payload: models.ChatCompletionRequest{
            Messages: []models.Message{
                {Role: "user", Content: "Hi"},
            },
        },
    }

    result, err := adapter.Execute(context.Background(), req, "gemini-pro")
    // Assertions...
}
```

---

## Implementation Checklist

- [ ] Create `internal/provider/gemini/` directory
- [ ] Implement `GeminiAdapter` struct with `Adapter` interface
- [ ] Implement `GeminiRequestTransformer`
  - [ ] Message to contents transformation
  - [ ] Role mapping (assistant → model)
  - [ ] System message prepending
  - [ ] Parts array construction
  - [ ] GenerationConfig mapping
  - [ ] Safety settings injection
- [ ] Implement `GeminiResponseTransformer`
  - [ ] Candidate content extraction
  - [ ] Parts aggregation
  - [ ] Finish reason mapping
  - [ ] UsageMetadata transformation
- [ ] Implement `GeminiStreamTransformer`
  - [ ] SSE chunk parsing
  - [ ] Content accumulation
  - [ ] Delta calculation
  - [ ] Stream completion detection
- [ ] Add authentication handling
  - [ ] `x-goog-api-key` header (preferred)
  - [ ] `?key=` query parameter (fallback)
- [ ] Add URL transformation
  - [ ] `/v1/chat/completions` → `:generateContent`
  - [ ] `/v1/chat/completions` (stream) → `:streamGenerateContent`
  - [ ] Model extraction and injection
- [ ] Implement error handling and status code mapping
- [ ] Write unit tests for all transformers
- [ ] Write integration tests for the adapter
- [ ] Update factory.go to register Gemini adapter
- [ ] Add configuration loading for Gemini credentials
- [ ] Document known limitations (no native system role, different endpoint patterns)

---

## References

- [OpenAI Adapter Reference](../architecture/provider-adapters.md)
- [Implementing Adapters Guide](../guides/implementing-adapters.md)
- [Anthropic Adapter Design](./anthropic-adapter.md)
- Gemini API Documentation: https://ai.google.dev/api
- Gemini REST API Reference: https://ai.google.dev/api/rest
- AxonHub Patterns (internal): `/v1beta/models/{model}:generateContent`, `x-goog-api-key` auth
- OpenAI Adapter Pattern (reference): `/internal/provider/openai/`
