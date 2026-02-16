# Team Delta: Quality Assurance Strategy
## RAD Gateway Provider Adapter Implementation

**Document Version**: 1.0  
**Date**: 2026-02-16  
**Team**: Delta (Quality Assurance)  
**Scope**: Milestone 1 - Real Provider Adapters

---

## 1. Executive Summary

### Mission Statement
Team Delta ensures RAD Gateway delivers production-ready provider adapter implementations with comprehensive test coverage, reliable contract compliance, and measurable quality metrics. Our focus is on preventing defects in the provider abstraction layer while enabling rapid iteration through automated testing.

### Current Quality State
| Component | Coverage | Status | Risk Level |
|-----------|----------|--------|------------|
| Middleware | 96.2% | Healthy | Low |
| Core Gateway | 80.0% | Acceptable | Low |
| Routing | 79.2% | Acceptable | Low |
| API Handlers | 22.2% | At Risk | Medium |
| Provider Layer | 0.0% | Critical | High |
| Models | 0.0% | Critical | High |
| Usage/Trace | 0.0% | At Risk | Medium |
| **Overall** | **30.8%** | **Critical** | **High** |

### Key Risks Identified
1. **Zero provider adapter test coverage** - Real provider implementations lack any testing infrastructure
2. **Incomplete API handler tests** - Only basic happy-path coverage exists
3. **Missing contract tests** - No validation against actual OpenAI/Anthropic/Gemini API contracts
4. **No performance baselines** - No latency or throughput requirements defined
5. **Mock provider limitations** - Current mock does not simulate real provider behaviors

### Success Criteria for Milestone 1
- Provider adapter test coverage: > 90%
- Contract test coverage for all three providers: 100%
- Integration test pass rate: 100%
- Performance benchmarks established
- Zero critical defects in provider layer

---

## 2. Current Test Coverage Analysis

### 2.1 Test Inventory

| File | Lines | Coverage | Test Cases | Gaps |
|------|-------|----------|------------|------|
| `/internal/api/handlers_test.go` | 71 | 22.2% | 3 (health, chat, models) | Missing: responses, messages, embeddings, images, transcriptions, gemini, error handling |
| `/internal/core/gateway_test.go` | 51 | 80.0% | 1 (success path) | Missing: error paths, retry scenarios, timeout handling |
| `/internal/routing/router_test.go` | 52 | 79.2% | 2 (success, missing adapter) | Missing: retry logic, weighted selection, circuit breaker |
| `/internal/middleware/middleware_test.go` | 105 | 96.2% | 5 (auth, context) | Missing: rate limiting, logging |
| `/internal/provider/mock.go` | 88 | 0.0% | None | Missing: all test scenarios |
| `/internal/models/models.go` | 78 | 0.0% | None | Missing: validation tests |
| `/internal/usage/usage.go` | 60 | 0.0% | None | Missing: persistence, aggregation |
| `/internal/trace/trace.go` | 45 | 0.0% | None | Missing: correlation, filtering |

### 2.2 Coverage Gap Analysis

#### Critical Gaps (Must Fix for Milestone 1)
1. **Provider Adapter Interface** (`/internal/provider/provider.go`)
   - No tests for registry operations
   - No validation of adapter contract compliance
   - Missing error scenario coverage

2. **Mock Provider** (`/internal/provider/mock.go`)
   - Currently only used as test dependency
   - No standalone validation of mock behaviors
   - Mock responses not validated against real schemas

3. **API Type Handlers** (`/internal/api/handlers.go`)
   - Only `chatCompletions` has basic test
   - Missing: `responses`, `messages`, `embeddings`, `images`, `transcriptions`, `geminiCompat`
   - No validation of request/response transformations

#### High Priority Gaps
1. **Models Package** - No validation of struct serialization/deserialization
2. **Usage Tracking** - No verification of metrics accuracy
3. **Trace Store** - No validation of distributed tracing

#### Medium Priority Gaps
1. **Admin Handlers** - No tests for management endpoints
2. **Config Loading** - No validation of configuration parsing
3. **Main Entry Point** - No integration tests for server startup

### 2.3 Test Quality Assessment

**Strengths:**
- Table-driven test patterns used consistently
- Mock dependencies properly injected
- HTTP test recorder pattern for handlers
- Context propagation validated

**Weaknesses:**
- Limited negative test scenarios
- No parallel test execution
- Missing test data fixtures
- No property-based testing
- Limited assertions on response structures

---

## 3. Contract Test Strategy (Per Provider)

### 3.1 OpenAI Adapter Contract Tests

#### Scope
- **Base URL**: Configurable (default: `https://api.openai.com/v1`)
- **Authentication**: Bearer token via `Authorization` header
- **Endpoints**: 
  - `POST /chat/completions`
  - `POST /responses` (subset)
  - `POST /embeddings`
  - `POST /images/generations`
  - `POST /audio/transcriptions`
  - `GET /models`

#### Contract Test Matrix

| Test Case | Request Validation | Response Validation | Error Handling |
|-----------|-------------------|---------------------|----------------|
| Chat Completions | Model, messages, stream flag | ID, object, model, choices, usage | 400, 401, 429, 500 |
| Responses | Model, input, tools (optional) | ID, output, status | 400, 401, 429 |
| Embeddings | Model, input | Object, data array, usage | 400, 401 |
| Image Generations | Prompt, size, quality | Data array (base64 or URL) | 400, 429 |
| Audio Transcriptions | File, model, language | Text, language, duration | 400, 413 |

#### OpenAI-Specific Validations
```go
// Response schema validations
- response.ID must start with "chatcmpl-" (chat) or "resp_" (responses)
- response.Object must be "chat.completion" or "response"
- response.Model must match requested model
- usage.PromptTokens > 0 for non-empty input
- usage.CompletionTokens > 0 for non-empty output
- choices[0].message.role must be "assistant"
```

#### Test Fixtures Required
- `/internal/provider/openai/testdata/chat_completion_request.json`
- `/internal/provider/openai/testdata/chat_completion_response.json`
- `/internal/provider/openai/testdata/responses_request.json`
- `/internal/provider/openai/testdata/responses_response.json`
- `/internal/provider/openai/testdata/embeddings_request.json`
- `/internal/provider/openai/testdata/embeddings_response.json`
- `/internal/provider/openai/testdata/error_401.json`
- `/internal/provider/openai/testdata/error_429.json`

### 3.2 Anthropic Adapter Contract Tests

#### Scope
- **Base URL**: `https://api.anthropic.com/v1`
- **Authentication**: `x-api-key` header + optional `anthropic-version`
- **Endpoints**:
  - `POST /messages`
  - `POST /v1/messages` (dual-path compatibility)

#### Contract Test Matrix

| Test Case | Request Validation | Response Validation | Error Handling |
|-----------|-------------------|---------------------|----------------|
| Messages | Model, messages, max_tokens | ID, type, role, content, usage | 400, 401, 429, 529 |
| Streaming | Stream: true | Event stream format | Connection errors |
| Tool Use | Tools, tool_choice | Content blocks with tool_use | Tool validation errors |

#### Anthropic-Specific Validations
```go
// Response schema validations
- response.ID must start with "msg_"
- response.Type must be "message"
- response.Role must be "assistant"
- content array must contain valid content blocks
- usage.InputTokens > 0 for non-empty input
- usage.OutputTokens > 0 for non-empty output
- StopReason must be one of: "end_turn", "max_tokens", "stop_sequence", "tool_use"
```

#### Test Fixtures Required
- `/internal/provider/anthropic/testdata/messages_request.json`
- `/internal/provider/anthropic/testdata/messages_response.json`
- `/internal/provider/anthropic/testdata/messages_streaming.txt`
- `/internal/provider/anthropic/testdata/error_invalid_request.json`
- `/internal/provider/anthropic/testdata/error_overloaded.json`

### 3.3 Gemini Adapter Contract Tests

#### Scope
- **Base URL**: `https://generativelanguage.googleapis.com/v1beta`
- **Authentication**: `x-goog-api-key` header or query parameter
- **Endpoints**:
  - `POST /v1beta/models/{model}:{action}` (generateContent, streamGenerateContent, countTokens, embedContent)

#### Contract Test Matrix

| Test Case | Request Validation | Response Validation | Error Handling |
|-----------|-------------------|---------------------|----------------|
| Generate Content | Model, contents, generationConfig | Candidates, usageMetadata | 400, 403, 429, 503 |
| Stream Content | Same as above | Stream of chunks | Stream interruption |
| Count Tokens | Model, contents | TotalTokens | 400, 404 |
| Embed Content | Model, content | Embedding values | 400, 429 |

#### Gemini-Specific Validations
```go
// Response schema validations
- candidates array must not be empty on success
- candidates[0].content.role must be "model"
- candidates[0].finishReason must be one of: "STOP", "MAX_TOKENS", "SAFETY", "RECITATION", "OTHER"
- usageMetadata.promptTokenCount >= 0
- usageMetadata.candidatesTokenCount >= 0
- embedding.values array must have consistent dimension
```

#### Test Fixtures Required
- `/internal/provider/gemini/testdata/generate_content_request.json`
- `/internal/provider/gemini/testdata/generate_content_response.json`
- `/internal/provider/gemini/testdata/stream_content_chunk.txt`
- `/internal/provider/gemini/testdata/count_tokens_request.json`
- `/internal/provider/gemini/testdata/count_tokens_response.json`
- `/internal/provider/gemini/testdata/embed_content_request.json`
- `/internal/provider/gemini/testdata/embed_content_response.json`
- `/internal/provider/gemini/testdata/error_permission_denied.json`

### 3.4 Contract Testing Framework

#### Consumer-Driven Contract Pattern
```go
// Contract test interface
type ProviderContract interface {
    Name() string
    BaseURL() string
    AuthHeaders() map[string]string
    
    // Request builders
    BuildChatRequest(model string, messages []Message) ([]byte, error)
    BuildEmbeddingsRequest(model, input string) ([]byte, error)
    
    // Response validators
    ValidateChatResponse(body []byte) (*ChatResponse, error)
    ValidateEmbeddingsResponse(body []byte) (*EmbeddingsResponse, error)
    ValidateErrorResponse(body []byte) (*ProviderError, error)
}
```

#### Contract Recording/Replay
- **Recording Mode**: Capture real provider responses for fixture generation
- **Replay Mode**: Use recorded fixtures for fast, deterministic tests
- **Validation Mode**: Compare adapter output against recorded contracts

#### Contract Test Execution
```bash
# Run all contract tests
make test-contracts

# Run specific provider contract tests
make test-contract-openai
make test-contract-anthropic
make test-contract-gemini

# Record new fixtures (requires API keys)
make record-contracts PROVIDER=openai
```

---

## 4. Mock Provider Enhancement Plan

### 4.1 Current Mock Limitations

The existing `MockAdapter` in `/internal/provider/mock.go` has these deficiencies:

1. **Generic Response for Multiple API Types** - Uses `GenericResponse` for responses, messages, gemini, images, transcriptions
2. **No Request Validation** - Does not validate incoming request structure
3. **Static Responses** - Always returns success with fixed data
4. **No Error Simulation** - Cannot test error handling paths
5. **No Latency Simulation** - Cannot test timeout scenarios
6. **No State Management** - Cannot simulate rate limiting or quotas

### 4.2 Enhanced Mock Specification

#### Configurable Mock Behavior
```go
type MockConfig struct {
    // Response behavior
    SuccessRate      float64       // 0.0 - 1.0 probability of success
    Latency          time.Duration // Artificial delay
    LatencyJitter    float64       // +/- percentage of latency variance
    
    // Error simulation
    ErrorRate        float64       // Probability of returning error
    ErrorTypes       []string      // Types of errors to simulate: "timeout", "rate_limit", "auth", "server"
    
    // Response customization
    ResponseTemplate map[string]interface{} // Override default responses
    
    // Rate limiting simulation
    RateLimitRequests int           // Requests allowed per window
    RateLimitWindow   time.Duration // Rate limit window
    
    // State tracking
    TrackCalls       bool          // Record all calls for verification
    MaxCallHistory   int           // Maximum calls to retain
}
```

#### Enhanced Mock Implementation
```go
type EnhancedMockAdapter struct {
    config     MockConfig
    callCount  int
    callHistory []CallRecord
    mu         sync.RWMutex
}

type CallRecord struct {
    Timestamp time.Time
    APIType   string
    Model     string
    Request   interface{}
    Response  interface{}
    Error     error
    Duration  time.Duration
}
```

### 4.3 Mock Capabilities by API Type

#### Chat Completions Mock
```go
// Request validation
- Validate model name format
- Validate messages structure (role, content required)
- Validate stream flag

// Response generation
- Echo last message content with prefix
- Generate realistic token counts based on content length
- Return proper OpenAI-style chat completion structure

// Error simulation
- Return 400 for invalid model
- Return 429 when rate limit exceeded
- Return 500 randomly based on config
```

#### Embeddings Mock
```go
// Request validation
- Validate model is embedding-capable
- Validate input is not empty

// Response generation
- Return fixed-dimension embedding vector (1536 for text-embedding-3-small)
- Calculate token estimate based on input length

// Error simulation
- Return 400 for oversized input
- Return 401 for invalid auth (when auth checking enabled)
```

#### Error Response Simulation
```go
type MockErrorResponse struct {
    Error struct {
        Message string `json:"message"`
        Type    string `json:"type"`
        Code    string `json:"code,omitempty"`
    } `json:"error"`
}

// Standard error responses by provider type
var OpenAIErrors = map[string]MockErrorResponse{
    "invalid_request": {Error: {Message: "Invalid request", Type: "invalid_request_error"}},
    "authentication":  {Error: {Message: "Invalid API key", Type: "authentication_error", Code: "invalid_api_key"}},
    "rate_limit":      {Error: {Message: "Rate limit exceeded", Type: "rate_limit_error", Code: "rate_limit_exceeded"}},
    "server":          {Error: {Message: "Server error", Type: "server_error"}},
}

var AnthropicErrors = map[string]MockErrorResponse{
    "invalid_request": {Error: {Message: "Invalid request", Type: "invalid_request_error"}},
    "authentication":  {Error: {Message: "Invalid API key", Type: "authentication_error"}},
    "rate_limit":      {Error: {Message: "Rate limit exceeded", Type: "rate_limit_error"}},
    "overloaded":      {Error: {Message: "Overloaded", Type: "overloaded_error"}},
}
```

### 4.4 Mock Usage Patterns

#### Unit Testing with Mock
```go
func TestChatCompletionsWithLatency(t *testing.T) {
    mock := NewEnhancedMockAdapter(MockConfig{
        Latency:       100 * time.Millisecond,
        LatencyJitter: 0.1, // 10% variance
        SuccessRate:   1.0,
    })
    
    registry := provider.NewRegistry(mock)
    // Test with controlled latency...
}

func TestRetryLogicWithFailures(t *testing.T) {
    mock := NewEnhancedMockAdapter(MockConfig{
        SuccessRate: 0.5, // 50% failure rate
        ErrorTypes:  []string{"timeout", "server"},
    })
    
    // Test retry behavior...
}
```

### 4.5 Mock Implementation Timeline

| Phase | Deliverable | Owner | Due Date |
|-------|-------------|-------|----------|
| 1 | EnhancedMockAdapter struct and config | SDET | Day 3 |
| 2 | Chat completions mock with validation | SDET | Day 5 |
| 3 | Embeddings mock implementation | SDET | Day 7 |
| 4 | Error simulation framework | SDET | Day 9 |
| 5 | Rate limiting simulation | SDET | Day 11 |
| 6 | Full test coverage for mock | SDET | Day 13 |

---

## 5. Test Fixtures Design

### 5.1 Fixture Directory Structure

```
/internal/provider/testdata/
├── openai/
│   ├── requests/
│   │   ├── chat_completion_basic.json
│   │   ├── chat_completion_with_tools.json
│   │   ├── chat_completion_streaming.json
│   │   ├── embeddings_single.json
│   │   ├── embeddings_batch.json
│   │   ├── image_generation.json
│   │   └── audio_transcription.json
│   ├── responses/
│   │   ├── chat_completion_success.json
│   │   ├── chat_completion_streaming.txt
│   │   ├── embeddings_success.json
│   │   ├── image_generation_success.json
│   │   └── models_list.json
│   └── errors/
│       ├── invalid_request.json
│       ├── authentication_error.json
│       ├── rate_limit_error.json
│       ├── server_error.json
│       └── timeout_error.json
├── anthropic/
│   ├── requests/
│   │   ├── messages_basic.json
│   │   ├── messages_with_tools.json
│   │   ├── messages_streaming.json
│   │   └── messages_max_tokens.json
│   ├── responses/
│   │   ├── messages_success.json
│   │   ├── messages_with_tool_use.json
│   │   └── messages_streaming.txt
│   └── errors/
│       ├── invalid_request.json
│       ├── authentication_error.json
│       ├── rate_limit_error.json
│       └── overloaded_error.json
└── gemini/
    ├── requests/
    │   ├── generate_content_basic.json
    │   ├── generate_content_streaming.json
    │   ├── count_tokens.json
    │   └── embed_content.json
    ├── responses/
    │   ├── generate_content_success.json
    │   ├── generate_content_streaming.txt
    │   ├── count_tokens_success.json
    │   └── embed_content_success.json
    └── errors/
        ├── invalid_argument.json
        ├── permission_denied.json
        ├── resource_exhausted.json
        └── unavailable_error.json
```

### 5.2 Fixture File Specifications

#### Request Fixture Format
```json
{
  "_meta": {
    "provider": "openai",
    "api_type": "chat.completions",
    "description": "Basic chat completion request",
    "version": "2024-01"
  },
  "request": {
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "user", "content": "Hello, world!"}
    ]
  }
}
```

#### Response Fixture Format
```json
{
  "_meta": {
    "provider": "openai",
    "api_type": "chat.completions",
    "status_code": 200,
    "headers": {
      "content-type": "application/json",
      "x-request-id": "req_12345"
    }
  },
  "response": {
    "id": "chatcmpl-abc123",
    "object": "chat.completion",
    "model": "gpt-4o-mini",
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
      "prompt_tokens": 9,
      "completion_tokens": 9,
      "total_tokens": 18
    }
  }
}
```

#### Error Fixture Format
```json
{
  "_meta": {
    "provider": "openai",
    "api_type": "chat.completions",
    "status_code": 429,
    "error_type": "rate_limit"
  },
  "response": {
    "error": {
      "message": "Rate limit exceeded",
      "type": "rate_limit_error",
      "code": "rate_limit_exceeded"
    }
  },
  "headers": {
    "retry-after": "20"
  }
}
```

### 5.3 Fixture Validation Rules

#### JSON Schema Validation
```go
// Fixture validator interface
type FixtureValidator struct {
    schemas map[string]*jsonschema.Schema
}

func (v *FixtureValidator) ValidateRequest(provider, apiType string, fixture []byte) error
func (v *FixtureValidator) ValidateResponse(provider, apiType string, fixture []byte) error
```

#### Required Validations
1. All fixtures must have `_meta` section with provider, api_type, description
2. Request fixtures must validate against provider request schema
3. Response fixtures must validate against provider response schema
4. Error fixtures must include status_code and error_type
5. JSON must be properly formatted (use `jq` for validation)

### 5.4 Fixture Generation from Recordings

```go
// Recording capture for real provider interactions
type ContractRecorder struct {
    provider string
    baseURL  string
    client   *http.Client
}

func (r *ContractRecorder) RecordChatCompletion(req ChatCompletionRequest) (*Fixture, error) {
    // Make real API call
    // Capture request and response
    // Sanitize sensitive data (API keys, tokens)
    // Generate fixture file
}
```

---

## 6. Regression Test Suite Design

### 6.1 Regression Test Levels

#### Level 1: Unit Regression (Fast - < 30 seconds)
- Provider adapter unit tests
- Request/response transformation tests
- Error handling tests
- **Trigger**: Every commit
- **Execution**: `make test-unit`

#### Level 2: Integration Regression (Medium - < 2 minutes)
- Handler + Gateway + Router integration
- Mock provider + full stack
- Multiple API type scenarios
- **Trigger**: Every PR
- **Execution**: `make test-integration`

#### Level 3: Contract Regression (Slow - < 5 minutes)
- Provider contract validation against fixtures
- Schema compatibility checks
- Error response validation
- **Trigger**: Before merge to main
- **Execution**: `make test-contract`

#### Level 4: E2E Regression (Comprehensive - < 15 minutes)
- Full application with real provider sandboxes
- Cross-provider routing scenarios
- Load testing baseline
- **Trigger**: Nightly + release candidates
- **Execution**: `make test-e2e`

### 6.2 Regression Test Scenarios

#### Core Functionality Regression
```go
// Test suite: CoreRegression
type CoreRegressionSuite struct {
    registry *provider.Registry
    router   *routing.Router
    gateway  *core.Gateway
}

func (s *CoreRegressionSuite) TestChatCompletionRoundTrip(t *testing.T) {
    // Full round-trip: request -> handler -> gateway -> router -> adapter -> response
}

func (s *CoreRegressionSuite) TestFailoverToSecondaryProvider(t *testing.T) {
    // Primary fails, secondary succeeds
}

func (s *CoreRegressionSuite) TestAllRouteAttemptsFail(t *testing.T) {
    // All providers fail, proper error returned
}

func (s *CoreRegressionSuite) TestUsageRecordedOnSuccess(t *testing.T) {
    // Verify usage tracking integration
}

func (s *CoreRegressionSuite) TestTraceEventsEmitted(t *testing.T) {
    // Verify tracing integration
}
```

#### API Compatibility Regression
```go
// Test suite: APICompatibility
func TestOpenAIChatCompletionsCompatibility(t *testing.T) {
    // Validate response format matches OpenAI spec exactly
}

func TestAnthropicMessagesCompatibility(t *testing.T) {
    // Validate response format matches Anthropic spec exactly
}

func TestGeminiGenerateContentCompatibility(t *testing.T) {
    // Validate response format matches Gemini spec exactly
}
```

#### Error Handling Regression
```go
// Test suite: ErrorHandling
func TestInvalidJSONReturns400(t *testing.T) {
    // Malformed JSON in request body
}

func TestMissingModelReturns400(t *testing.T) {
    // Empty model field
}

func TestInvalidAPIMethodReturns405(t *testing.T) {
    // GET instead of POST
}

func TestProviderTimeoutReturns504(t *testing.T) {
    // Upstream timeout handling
}

func TestRateLimitReturns429(t *testing.T) {
    // Rate limit response propagation
}
```

### 6.3 Regression Test Data Sets

#### Minimal Test Set (CI/CD Pipeline)
| Test | Provider | API Type | Scenario |
|------|----------|----------|----------|
| 1 | Mock | chat | Basic success |
| 2 | Mock | chat | Invalid request |
| 3 | Mock | embeddings | Basic success |
| 4 | Mock | responses | Basic success |
| 5 | Mock | messages | Basic success |
| 6 | Mock | gemini | Basic success |
| 7 | Mock | chat | Failover |
| 8 | Mock | chat | All fail |

#### Full Regression Set (Pre-release)
- All minimal tests
- Each provider x each API type x (success + 3 error types)
- Routing scenarios (weighted, failover, retry exhaustion)
- Usage tracking accuracy (50 requests)
- Trace completeness (event count, correlation)

### 6.4 Regression Automation

#### CI/CD Integration
```yaml
# .github/workflows/regression.yml
name: Regression Tests

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]
  schedule:
    - cron: '0 2 * * *'  # Nightly at 2 AM

jobs:
  unit-regression:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: make test-unit

  integration-regression:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: make test-integration

  contract-regression:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: make test-contract

  e2e-regression:
    runs-on: ubuntu-latest
    if: github.event_name == 'schedule' || contains(github.ref, 'release')
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: make test-e2e
    env:
      OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
      ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
      GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}
```

---

## 7. Performance Benchmark Requirements

### 7.1 Benchmark Categories

#### Latency Benchmarks
| Metric | Target | Measurement |
|--------|--------|-------------|
| p50 Request Latency | < 100ms | Time from request to response (mock provider) |
| p95 Request Latency | < 200ms | Time from request to response (mock provider) |
| p99 Request Latency | < 500ms | Time from request to response (mock provider) |
| Adapter Overhead | < 10ms | Time added by adapter transformation |
| Routing Decision | < 1ms | Time to select candidate |

#### Throughput Benchmarks
| Metric | Target | Measurement |
|--------|--------|-------------|
| RPS Single Core | > 1000 | Requests per second on single CPU core |
| RPS Multi Core | > 5000 | Requests per second with full CPU utilization |
| Concurrent Connections | > 10000 | Simultaneous open connections |
| Memory per Request | < 10KB | Peak memory allocated per request |

#### Resource Utilization Benchmarks
| Metric | Target | Measurement |
|--------|--------|-------------|
| CPU Utilization | < 50% | At target RPS |
| Memory Growth | < 1% | Per 1000 requests (no leak) |
| GC Pause Time | < 10ms | Maximum garbage collection pause |
| Goroutine Leak | 0 | Goroutines should not grow unbounded |

### 7.2 Benchmark Test Suite

```go
// Benchmark suite: ProviderPerformance
func BenchmarkChatCompletion(b *testing.B) {
    gateway := setupBenchmarkGateway()
    ctx := context.Background()
    req := models.ChatCompletionRequest{
        Model:    "gpt-4o-mini",
        Messages: []models.Message{{Role: "user", Content: "Hello"}},
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _, err := gateway.Handle(ctx, "chat", "gpt-4o-mini", req)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkEmbeddings(b *testing.B) {
    gateway := setupBenchmarkGateway()
    ctx := context.Background()
    req := models.EmbeddingsRequest{
        Model: "text-embedding-3-small",
        Input: "Test input for embeddings",
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _, err := gateway.Handle(ctx, "embeddings", "text-embedding-3-small", req)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkParallelRequests(b *testing.B) {
    gateway := setupBenchmarkGateway()
    ctx := context.Background()
    req := models.ChatCompletionRequest{
        Model:    "gpt-4o-mini",
        Messages: []models.Message{{Role: "user", Content: "Hello"}},
    }
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, _, err := gateway.Handle(ctx, "chat", "gpt-4o-mini", req)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}
```

### 7.3 Benchmark Execution

```bash
# Run all benchmarks
make bench

# Run specific benchmark
make bench-target BENCH=BenchmarkChatCompletion

# Run with memory profiling
make bench-mem

# Run with CPU profiling
make bench-cpu

# Compare benchmarks (requires benchstat)
make bench-compare BASE=old.txt NEW=new.txt
```

### 7.4 Benchmark Reporting

```go
// Benchmark report generator
type BenchmarkReport struct {
    Timestamp    time.Time
    CommitSHA    string
    GoVersion    string
    Results      []BenchmarkResult
}

type BenchmarkResult struct {
    Name         string
    N            int           // Number of iterations
    NsPerOp      float64       // Nanoseconds per operation
    AllocsPerOp  int64         // Allocations per operation
    BytesPerOp   int64         // Bytes allocated per operation
    MBPerSec     float64       // Throughput in MB/s
}
```

---

## 8. Load Testing Strategy

### 8.1 Load Test Scenarios

#### Scenario 1: Steady State Load
- **Duration**: 10 minutes
- **Request Rate**: 100 RPS
- **Provider Mix**: 70% OpenAI, 20% Anthropic, 10% Gemini
- **API Type Mix**: 80% chat, 15% embeddings, 5% other
- **Success Criteria**: 
  - p95 latency < 200ms
  - Error rate < 0.1%
  - No memory leaks

#### Scenario 2: Ramp Up Test
- **Duration**: 5 minutes
- **Pattern**: Linear ramp from 10 RPS to 500 RPS
- **Provider**: Mock (isolated test)
- **Success Criteria**:
  - Linear scaling of throughput
  - Latency increase < 2x at max load
  - No connection failures

#### Scenario 3: Spike Test
- **Baseline**: 50 RPS for 2 minutes
- **Spike**: Sudden increase to 1000 RPS for 30 seconds
- **Recovery**: Return to 50 RPS
- **Success Criteria**:
  - No crashes during spike
  - Recovery to baseline latency within 60 seconds
  - Error rate during spike < 5%

#### Scenario 4: Endurance Test
- **Duration**: 1 hour
- **Request Rate**: 200 RPS
- **Provider**: Mock with simulated latency (50ms)
- **Success Criteria**:
  - Memory growth < 5% over test duration
  - Goroutine count stable
  - No resource exhaustion

### 8.2 Load Testing Tools

#### Primary Tool: k6
```javascript
// loadtest/scenarios/steady_state.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '2m', target: 100 },  // Ramp up
    { duration: '10m', target: 100 }, // Steady state
    { duration: '2m', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<200'],
    http_req_failed: ['rate<0.001'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const payload = JSON.stringify({
    model: 'gpt-4o-mini',
    messages: [{ role: 'user', content: 'Hello' }],
  });

  const res = http.post(`${BASE_URL}/v1/chat/completions`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 200ms': (r) => r.timings.duration < 200,
  });

  sleep(1);
}
```

#### Secondary Tool: Go-based load generator
```go
// loadtest/cmd/loadtest/main.go
// For scenarios requiring precise control or internal state manipulation
```

### 8.3 Load Test Infrastructure

#### Docker Compose Setup
```yaml
# loadtest/docker-compose.yml
version: '3.8'
services:
  rad-gateway:
    build: ../
    ports:
      - "8080:8080"
    environment:
      - RAD_LOG_LEVEL=warn
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 512M

  k6:
    image: grafana/k6:latest
    volumes:
      - ./scenarios:/scenarios
    command: run /scenarios/steady_state.js
    depends_on:
      - rad-gateway
    environment:
      - BASE_URL=http://rad-gateway:8080

  prometheus:
    image: prom/prometheus
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana
    ports:
      - "3000:3000"
```

### 8.4 Load Test Execution Schedule

| Test | Frequency | Environment | Duration |
|------|-----------|-------------|----------|
| Steady State | Every PR | CI (mock) | 2 min |
| Ramp Up | Daily | Staging | 5 min |
| Spike Test | Weekly | Staging | 10 min |
| Endurance | Bi-weekly | Staging | 1 hour |
| Full Suite | Before release | Production-like | 2 hours |

---

## 9. Test Automation Framework

### 9.1 Framework Architecture

```
test/
├── unit/              # Unit tests (fast, isolated)
│   ├── provider/
│   ├── routing/
│   ├── core/
│   └── api/
├── integration/       # Integration tests (medium speed)
│   ├── handler_integration_test.go
│   └── gateway_integration_test.go
├── contract/          # Contract tests (provider validation)
│   ├── openai/
│   ├── anthropic/
│   └── gemini/
├── e2e/              # End-to-end tests (slow, comprehensive)
│   └── scenarios/
├── fixtures/          # Test data
│   ├── requests/
│   └── responses/
├── mocks/            # Enhanced mocks
│   └── enhanced_mock.go
├── bench/            # Benchmarks
│   └── benchmarks_test.go
└── load/             # Load testing
    └── scenarios/
```

### 9.2 Test Helpers and Utilities

#### Test Fixture Loader
```go
// test/fixtures/loader.go
package fixtures

import (
    "embed"
    "encoding/json"
    "path/filepath"
)

//go:embed openai/* anthropic/* gemini/*
var fixtureFS embed.FS

func LoadRequest(provider, name string, target interface{}) error {
    path := filepath.Join(provider, "requests", name+".json")
    data, err := fixtureFS.ReadFile(path)
    if err != nil {
        return err
    }
    return json.Unmarshal(data, target)
}

func LoadResponse(provider, name string, target interface{}) error {
    path := filepath.Join(provider, "responses", name+".json")
    data, err := fixtureFS.ReadFile(path)
    if err != nil {
        return err
    }
    return json.Unmarshal(data, target)
}

func LoadError(provider, name string, target interface{}) error {
    path := filepath.Join(provider, "errors", name+".json")
    data, err := fixtureFS.ReadFile(path)
    if err != nil {
        return err
    }
    return json.Unmarshal(data, target)
}
```

#### Test Server Builder
```go
// test/helpers/server.go
package helpers

import (
    "net/http"
    "net/http/httptest"
    "radgateway/internal/api"
    "radgateway/internal/core"
    "radgateway/internal/provider"
    "radgateway/internal/routing"
    "radgateway/internal/trace"
    "radgateway/internal/usage"
)

type TestServer struct {
    Server   *httptest.Server
    Gateway  *core.Gateway
    Registry *provider.Registry
    Router   *routing.Router
}

func NewTestServer(adapters ...provider.Adapter) *TestServer {
    registry := provider.NewRegistry(adapters...)
    router := routing.New(registry, buildRouteTable(), 2)
    gateway := core.New(router, usage.NewInMemory(1000), trace.NewStore(1000))
    
    mux := http.NewServeMux()
    api.NewHandlers(gateway).Register(mux)
    
    return &TestServer{
        Server:   httptest.NewServer(mux),
        Gateway:  gateway,
        Registry: registry,
        Router:   router,
    }
}

func buildRouteTable() map[string][]provider.Candidate {
    return map[string][]provider.Candidate{
        "gpt-4o-mini":       {{Name: "openai", Model: "gpt-4o-mini", Weight: 100}},
        "claude-3-5-sonnet": {{Name: "anthropic", Model: "claude-3-5-sonnet", Weight: 100}},
        "gemini-1.5-flash":  {{Name: "gemini", Model: "gemini-1.5-flash", Weight: 100}},
    }
}
```

#### Assertion Helpers
```go
// test/helpers/assertions.go
package helpers

import (
    "testing"
    "radgateway/internal/models"
)

func AssertChatCompletionResponse(t *testing.T, resp models.ChatCompletionResponse) {
    t.Helper()
    
    if resp.ID == "" {
        t.Error("expected non-empty ID")
    }
    if resp.Object != "chat.completion" {
        t.Errorf("expected object 'chat.completion', got %q", resp.Object)
    }
    if len(resp.Choices) == 0 {
        t.Error("expected at least one choice")
    }
    if resp.Choices[0].Message.Role != "assistant" {
        t.Errorf("expected role 'assistant', got %q", resp.Choices[0].Message.Role)
    }
}

func AssertUsageValid(t *testing.T, usage models.Usage) {
    t.Helper()
    
    if usage.PromptTokens < 0 {
        t.Error("prompt tokens cannot be negative")
    }
    if usage.CompletionTokens < 0 {
        t.Error("completion tokens cannot be negative")
    }
    if usage.TotalTokens != usage.PromptTokens+usage.CompletionTokens {
        t.Error("total tokens mismatch")
    }
}
```

### 9.3 Test Execution Makefile Targets

```makefile
# Test targets
.PHONY: test test-unit test-integration test-contract test-e2e test-all

test: test-unit

test-unit:
	@echo "Running unit tests..."
	go test -v -race -coverprofile=coverage.out ./internal/...

test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./test/integration/...

test-contract:
	@echo "Running contract tests..."
	go test -v -tags=contract ./test/contract/...

test-e2e:
	@echo "Running E2E tests..."
	go test -v -tags=e2e ./test/e2e/...

test-all: test-unit test-integration test-contract

test-coverage:
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Benchmark targets
.PHONY: bench bench-cpu bench-mem

bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./test/bench/...

bench-cpu:
	@echo "Running benchmarks with CPU profiling..."
	go test -bench=. -cpuprofile=cpu.prof ./test/bench/...
	go tool pprof cpu.prof

bench-mem:
	@echo "Running benchmarks with memory profiling..."
	go test -bench=. -memprofile=mem.prof ./test/bench/...
	go tool pprof mem.prof

# Load test targets
.PHONY: load-test load-test-steady load-test-spike

load-test:
	k6 run loadtest/scenarios/steady_state.js

load-test-steady:
	BASE_URL=http://localhost:8080 k6 run loadtest/scenarios/steady_state.js

load-test-spike:
	BASE_URL=http://localhost:8080 k6 run loadtest/scenarios/spike.js

# Contract recording
.PHONY: record-contracts

record-contracts:
	@echo "Recording provider contracts (requires API keys)..."
	go test -v -tags=record ./test/contract/... -provider=$(PROVIDER)
```

### 9.4 Test Configuration

```yaml
# test/config/test.yaml
environments:
  unit:
    providers:
      - mock
    mock_latency: 0ms
    mock_error_rate: 0.0
  
  integration:
    providers:
      - mock
    mock_latency: 10ms
    mock_error_rate: 0.01
  
  contract:
    providers:
      - openai
      - anthropic
      - gemini
    fixtures_only: true
  
  e2e:
    providers:
      - openai
      - anthropic
      - gemini
    use_sandbox: true
    rate_limit: 10
```

---

## 10. Quality Gates for Milestone 1

### 10.1 Pre-Commit Gates

#### Code Quality Checks
| Check | Tool | Threshold | Enforcement |
|-------|------|-----------|-------------|
| Linting | golangci-lint | Zero errors | Required |
| Formatting | gofmt | No changes | Required |
| Vetting | go vet | Zero issues | Required |
| Import Order | goimports | Correct order | Required |
| Cyclomatic Complexity | gocyclo | < 15 per function | Warning |

#### Unit Test Gate
| Metric | Threshold | Enforcement |
|--------|-----------|-------------|
| Test Pass Rate | 100% | Required |
| Coverage (new code) | > 90% | Required |
| Coverage (total) | > 50% (current: 30.8%) | Required |
| Race Detection | Pass | Required |

### 10.2 Pre-PR Gates

#### Integration Test Gate
| Metric | Threshold | Enforcement |
|--------|-----------|-------------|
| Integration Pass Rate | 100% | Required |
| Mock Provider Coverage | 100% of scenarios | Required |
| Handler Coverage | > 80% | Required |
| Gateway Coverage | > 90% | Required |

#### Contract Test Gate
| Metric | Threshold | Enforcement |
|--------|-----------|-------------|
| OpenAI Contract | Pass | Required |
| Anthropic Contract | Pass | Required |
| Gemini Contract | Pass | Required |
| Fixture Validation | 100% | Required |

### 10.3 Pre-Merge Gates

#### Performance Gate
| Metric | Threshold | Enforcement |
|--------|-----------|-------------|
| Benchmark Regression | < 5% vs baseline | Required |
| Memory Allocation | No increase | Warning |
| p95 Latency | < 200ms (mock) | Required |

#### Load Test Gate
| Metric | Threshold | Enforcement |
|--------|-----------|-------------|
| Steady State (2 min) | Pass | Required |
| Error Rate | < 0.1% | Required |
| Memory Growth | < 5% | Required |

### 10.4 Pre-Release Gates

#### Full Test Suite
| Component | Coverage | Status |
|-----------|----------|--------|
| Provider Adapters | > 95% | Required |
| API Handlers | > 90% | Required |
| Gateway Core | > 95% | Required |
| Routing | > 90% | Required |
| Middleware | > 95% | Required |
| **Total** | **> 80%** | Required |

#### Documentation Gate
| Document | Status | Enforcement |
|----------|--------|-------------|
| API Documentation | Complete | Required |
| Provider Integration Guide | Complete | Required |
| Changelog | Updated | Required |
| Test Report | Generated | Required |

### 10.5 Quality Gate Implementation

```yaml
# .github/workflows/quality-gates.yml
name: Quality Gates

on:
  pull_request:
    branches: [ main ]

jobs:
  code-quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v3
      - name: go vet
        run: go vet ./...
      - name: gofmt check
        run: |
          if [ -n "$(gofmt -l .)" ]; then
            echo "Code is not formatted. Run 'gofmt -w .'"
            exit 1
          fi

  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - name: Run tests
        run: go test -race -coverprofile=coverage.out ./...
      - name: Check coverage
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
          if (( $(echo "$COVERAGE < 50.0" | bc -l) )); then
            echo "Coverage $COVERAGE% is below threshold 50%"
            exit 1
          fi

  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - name: Run integration tests
        run: go test -tags=integration ./test/integration/...

  contract-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - name: Run contract tests
        run: go test -tags=contract ./test/contract/...

  benchmarks:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - name: Run benchmarks
        run: go test -bench=. ./test/bench/... | tee bench.txt
      - name: Compare benchmarks
        uses: benchmark-action/github-action-benchmark@v1
        with:
          tool: 'go'
          output-file-path: bench.txt
          github-token: ${{ secrets.GITHUB_TOKEN }}
          alert-threshold: '150%'
```

### 10.6 Quality Gate Dashboard

| Gate | Current Status | Target Date | Owner |
|------|----------------|-------------|-------|
| Pre-Commit | In Progress | Day 3 | All |
| Pre-PR | Not Started | Day 7 | SDET |
| Pre-Merge | Not Started | Day 10 | QA Architect |
| Pre-Release | Not Started | Day 14 | QA Architect |

---

## 11. Team Member Responsibilities

### 11.1 QA Architect
**Focus**: Global testing strategy and architecture

**Deliverables**:
- Overall test strategy design
- Quality gate definition
- Test architecture review
- Cross-team coordination

**Key Activities**:
- Review and approve test plans
- Define test coverage standards
- Establish quality metrics
- Escalation point for quality issues

### 11.2 SDET (Software Development Engineer in Test)
**Focus**: Automated test code and frameworks

**Deliverables**:
- Enhanced mock provider implementation
- Test automation framework
- Contract test implementation
- CI/CD test integration

**Key Activities**:
- Write unit and integration tests
- Implement test helpers and utilities
- Build test automation pipeline
- Maintain test fixtures

### 11.3 Performance/Load Engineer
**Focus**: Scale testing and benchmarks

**Deliverables**:
- Benchmark test suite
- Load testing scenarios
- Performance baselines
- Resource utilization analysis

**Key Activities**:
- Define performance SLAs
- Create load test scripts
- Execute load tests
- Analyze performance bottlenecks

### 11.4 Manual QA / UAT Coordinator
**Focus**: User acceptance testing

**Deliverables**:
- UAT test scenarios
- Exploratory testing findings
- Provider compatibility matrix
- User documentation validation

**Key Activities**:
- Execute manual test scenarios
- Validate provider integrations
- Test edge cases
- Coordinate UAT sign-offs

### 11.5 Accessibility (A11y) Expert
**Focus**: WCAG compliance

**Deliverables**:
- Admin UI accessibility audit (future)
- API response accessibility guidelines
- Documentation accessibility review

**Key Activities**:
- Review admin UI for accessibility
- Ensure API responses support assistive technologies
- Validate documentation structure

---

## 12. Risk Register

| ID | Risk | Impact | Likelihood | Mitigation | Owner |
|----|------|--------|------------|------------|-------|
| R1 | Provider API changes break contracts | High | Medium | Contract tests with version pinning | SDET |
| R2 | Mock doesn't accurately represent real provider | High | Medium | Regular fixture updates from recordings | SDET |
| R3 | Load tests don't reflect production patterns | Medium | Medium | Analyze production logs, realistic scenarios | Perf Eng |
| R4 | Test environment differs from production | Medium | Medium | Container-based testing, staging validation | QA Arch |
| R5 | Insufficient test data coverage | Medium | Low | Comprehensive fixture library | SDET |
| R6 | Flaky tests reduce confidence | High | Low | Retry logic, idempotent tests, root cause fixes | SDET |
| R7 | Performance degradation unnoticed | Medium | Medium | Automated benchmark comparison | Perf Eng |
| R8 | Security vulnerabilities in test code | Low | Low | Security scanning, secret management | QA Arch |

---

## 13. Timeline and Milestones

### Sprint 1 (Days 1-7): Foundation
| Day | Deliverable | Owner |
|-----|-------------|-------|
| 1 | Test strategy document complete | QA Architect |
| 2 | Enhanced mock design approved | QA Architect |
| 3 | Enhanced mock implementation started | SDET |
| 4 | Contract test framework started | SDET |
| 5 | Chat completions mock complete | SDET |
| 6 | Fixture library structure defined | SDET |
| 7 | Pre-PR gates implemented | SDET |

### Sprint 2 (Days 8-14): Implementation
| Day | Deliverable | Owner |
|-----|-------------|-------|
| 8 | Embeddings mock complete | SDET |
| 9 | OpenAI contract tests | SDET |
| 10 | Anthropic contract tests | SDET |
| 11 | Gemini contract tests | SDET |
| 12 | Benchmark suite | Perf Eng |
| 13 | Load test scenarios | Perf Eng |
| 14 | Milestone 1 quality gates pass | QA Architect |

---

## 14. Success Metrics

### Coverage Targets
| Component | Current | Milestone 1 Target |
|-----------|---------|-------------------|
| Provider Adapters | 0% | 95% |
| API Handlers | 22.2% | 90% |
| Gateway Core | 80.0% | 95% |
| Routing | 79.2% | 90% |
| Middleware | 96.2% | 95% |
| **Total** | **30.8%** | **> 80%** |

### Quality Metrics
| Metric | Target |
|--------|--------|
| Test Pass Rate | 100% |
| Defect Escape Rate | < 1% |
| Mean Time to Detect | < 1 hour |
| Flaky Test Rate | 0% |
| Contract Compliance | 100% |

### Performance Metrics
| Metric | Target |
|--------|--------|
| Unit Test Execution | < 30 seconds |
| Integration Test Execution | < 2 minutes |
| p95 Latency (mock) | < 200ms |
| Load Test RPS | > 1000 |

---

## 15. Conclusion

This QA strategy provides a comprehensive framework for ensuring RAD Gateway's provider adapter implementations meet production quality standards. By focusing on:

1. **Comprehensive test coverage** across all provider adapters
2. **Contract-driven testing** to ensure API compatibility
3. **Enhanced mocking** for reliable and realistic testing
4. **Automated quality gates** to prevent regressions
5. **Performance validation** to meet scalability requirements

Team Delta will deliver a robust testing infrastructure that enables rapid, confident development of provider adapter functionality while maintaining the highest quality standards.

### Immediate Next Steps
1. Review and approve this strategy document
2. Begin EnhancedMockAdapter implementation
3. Set up test fixture directory structure
4. Implement first contract tests for OpenAI adapter
5. Configure CI/CD quality gates

---

**Document Owners**: Team Delta (Quality Assurance)  
**Review Schedule**: Weekly during Milestone 1  
**Approval Required**: Tech Lead, Product Manager
