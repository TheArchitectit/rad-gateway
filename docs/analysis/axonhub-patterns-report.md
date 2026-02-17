# AxonHub Repository Analysis Report

**Analysis Date:** 2026-02-17
**Repository:** /mnt/ollama/git/axonhub
**Purpose:** Identify Go patterns and orchestrator features for RAD Gateway inspiration

---

## Executive Summary

AxonHub is a sophisticated AI API Gateway written in Go that demonstrates enterprise-grade patterns for building scalable, maintainable, and observable API gateways. This analysis identifies key architectural patterns, concurrency models, and design decisions that can inform RAD Gateway's development.

---

## 1. Architecture Overview

### 1.1 High-Level Structure

```
axonhub/
├── cmd/axonhub/           # Main application entry point
├── conf/                  # Configuration management
├── internal/
│   ├── build/            # Build information
│   ├── contexts/         # Context propagation patterns
│   ├── ent/              # Ent ORM (entity framework)
│   ├── log/              # Structured logging
│   ├── metrics/          # OpenTelemetry metrics
│   ├── objects/          # Domain objects
│   ├── pkg/
│   │   └── xcontext/     # Extended context utilities
│   ├── scopes/           # Scope management
│   ├── server/
│   │   ├── api/          # HTTP handlers (REST API)
│   │   ├── biz/          # Business logic layer
│   │   ├── db/           # Database connectivity
│   │   ├── dependencies/ # DI module definitions
│   │   ├── gc/           # Garbage collection worker
│   │   ├── gql/          # GraphQL handlers
│   │   ├── middleware/   # HTTP middleware
│   │   ├── orchestrator/ # Core request orchestration
│   │   └── static/       # Static assets
│   └── tracing/          # Distributed tracing
├── llm/
│   ├── httpclient/       # HTTP client abstraction
│   ├── pipeline/         # Request processing pipeline
│   ├── streams/          # Streaming abstractions
│   └── transformer/      # Provider transformers
└── examples/             # Usage examples
```

### 1.2 Architectural Patterns

**Layered Architecture:**
- **API Layer:** HTTP handlers, middleware, route definitions
- **Business Layer:** Services, orchestrators, business logic
- **Data Layer:** Ent ORM, database operations
- **Infrastructure Layer:** HTTP clients, transformers, metrics

---

## 2. Orchestrator Features

### 2.1 Core Orchestrator (`/mnt/ollama/git/axonhub/internal/server/orchestrator/`)

The orchestrator is the heart of AxonHub's request processing system. Key components:

**ChatCompletionOrchestrator** (`orchestrator.go` lines 18-101):
- Central request processing hub
- Pluggable middleware system
- Retry and failover logic
- Channel selection and load balancing

**Key Features:**
```go
// Multiple load balancing strategies
type ChatCompletionOrchestrator struct {
    adaptiveLoadBalancer       *LoadBalancer  // Primary with multiple strategies
    failoverLoadBalancer       *LoadBalancer  // Failover mode
    circuitBreakerLoadBalancer *LoadBalancer  // Circuit breaker pattern
    modelCircuitBreaker        *biz.ModelCircuitBreaker
}
```

### 2.2 Candidate Selection (`candidates.go`)

**Pattern:** Strategy-based channel selection with caching

```go
type CandidateSelector interface {
    Select(ctx context.Context, req *llm.Request) ([]*ChannelModelsCandidate, error)
}
```

Features:
- Association caching with TTL (5 minutes)
- Thread-safe cache with RWMutex
- Fallback to legacy channel selection
- Model-based association resolution

### 2.3 Load Balancing (`load_balancer.go`)

**Pattern:** Composite strategy with scoring system

**LoadBalanceStrategy Interface:**
```go
type LoadBalanceStrategy interface {
    Score(ctx context.Context, channel *biz.Channel) float64
    ScoreWithDebug(ctx context.Context, channel *biz.Channel) (float64, StrategyScore)
    Name() string
}
```

**Implemented Strategies:**
- `TraceAwareStrategy` - Request tracing aware selection
- `ErrorAwareStrategy` - Error rate based selection
- `WeightRoundRobinStrategy` - Weighted round-robin
- `ConnectionAwareStrategy` - Connection count based
- `ModelAwareCircuitBreakerStrategy` - Circuit breaker pattern
- `RandomStrategy` - Random selection
- `WeightStrategy` - Weight-based selection

**Features:**
- Partial sorting for efficiency (only top K)
- Debug mode with detailed decision logging
- Thread-safe selection tracking
- Configurable retry-aware topK calculation

---

## 3. Concurrency Patterns

### 3.1 Dependency Injection with Uber FX

**Pattern:** Constructor-based DI with lifecycle management

```go
// /mnt/ollama/git/axonhub/internal/server/server.go
func Run(opts ...fx.Option) {
    constructors := []any{
        openapi.NewGraphqlHandlers,
        gql.NewGraphqlHandlers,
        gc.NewWorker,
        New,
    }

    app := fx.New(
        fx.NopLogger,
        fx.Provide(constructors...),
        dependencies.Module,
        biz.Module,
        api.Module,
        fx.Invoke(SetupRoutes),
    )
    app.Run()
}
```

**Benefits:**
- Automatic dependency resolution
- Lifecycle hooks (OnStart, OnStop)
- Modular organization with fx.Module

### 3.2 Context Propagation

**Pattern:** Context container for request-scoped data

```go
// /mnt/ollama/git/axonhub/internal/contexts/context.go
type ContextKey string

const containerContextKey ContextKey = "context_container"

func WithAPIKey(ctx context.Context, apiKey *ent.APIKey) context.Context {
    container := getContainer(ctx)
    container.APIKey = apiKey
    return withContainer(ctx, container)
}
```

**Stored Data:**
- API Key and User entities
- Trace ID and Request ID
- Project ID
- Error collection (thread-safe with mutex)

### 3.3 Stream Processing

**Pattern:** Generic stream interface

```go
// /mnt/ollama/git/axonhub/llm/streams/stream.go
type Stream[T any] interface {
    Next() bool
    Current() T
    Err() error
    Close() error
}
```

**Usage:**
- SSE (Server-Sent Events) streaming
- Response transformation streaming
- Error propagation through streams

### 3.4 Background Task Management

**Pattern:** Scheduled executor with CRON support

```go
// /mnt/ollama/git/axonhub/internal/server/gc/gc.go
type Worker struct {
    Executor executors.ScheduledExecutor
    // ...
}

func (w *Worker) Start(ctx context.Context) error {
    cancelFunc, err := w.Executor.ScheduleFuncAtCronRate(
        w.runCleanup,
        executors.CRONRule{Expr: w.Config.CRON},
    )
    // ...
}
```

**Features:**
- CRON-based scheduling
- Graceful shutdown
- Batch processing for large datasets
- Context-aware cancellation

### 3.5 Connection Tracking

**Pattern:** Atomic counters for concurrent request tracking

```go
// /mnt/ollama/git/axonhub/internal/server/orchestrator/connection_tracker.go
type DefaultConnectionTracker struct {
    connections sync.Map // map[int]*atomic.Int64
}
```

**Use Case:** Load balancing decisions based on active connection count

---

## 4. API Design Patterns

### 4.1 Handler Organization

**Pattern:** Grouped handlers with dependency injection

```go
// /mnt/ollama/git/axonhub/internal/server/routes.go
type Handlers struct {
    fx.In
    Graphql        *gql.GraphqlHandler
    OpenAI         *api.OpenAIHandlers
    Anthropic      *api.AnthropicHandlers
    // ...
}
```

### 4.2 Middleware Chain

**Pattern:** Composable middleware with ordering

```go
// Route groups with specific middleware
publicGroup := server.Group("", middleware.WithTimeout(server.Config.RequestTimeout))
adminGroup := server.Group("/admin", middleware.WithJWTAuth(services.AuthService), middleware.WithProjectID())
apiGroup := server.Group("/",
    middleware.WithTimeout(server.Config.LLMRequestTimeout),
    middleware.WithAPIKeyAuth(services.AuthService),
    middleware.WithSource(request.SourceAPI),
    middleware.WithThread(server.Config.Trace, services.ThreadService),
    middleware.WithTrace(server.Config.Trace, services.TraceService),
)
```

### 4.3 Multi-Provider API Support

**Pattern:** Provider-agnostic request processing

Supported APIs:
- OpenAI-compatible (`/v1/chat/completions`, `/v1/embeddings`)
- Anthropic (`/anthropic/v1/messages`)
- Google Gemini (`/gemini/:version/models/*action`)
- Jina AI (`/jina/v1/embeddings`, `/jina/v1/rerank`)

### 4.4 Unified Request/Response Model

**Pattern:** Internal unified format with provider transformers

```go
// Inbound transformer: Provider format -> Unified format
type Inbound interface {
    TransformRequest(ctx context.Context, request *httpclient.Request) (*llm.Request, error)
    TransformResponse(ctx context.Context, response *llm.Response) (*httpclient.Response, error)
    TransformError(ctx context.Context, err error) *httpclient.Error
}

// Outbound transformer: Unified format -> Provider format
type Outbound interface {
    TransformRequest(ctx context.Context, request *llm.Request) (*httpclient.Request, error)
    TransformResponse(ctx context.Context, response *httpclient.Response) (*llm.Response, error)
}
```

---

## 5. Configuration Approach

### 5.1 Viper-Based Configuration

**Pattern:** Hierarchical config with environment variable support

```go
// /mnt/ollama/git/axonhub/conf/conf.go
func Load() (Config, error) {
    v := viper.New()
    v.SetConfigName("config")
    v.SetConfigType("yml")
    v.AddConfigPath(".")
    v.AddConfigPath("/etc/axonhub/")
    v.AddConfigPath("$HOME/.config/axonhub/")
    v.AddConfigPath("./conf")

    // Environment variable support
    v.AutomaticEnv()
    v.SetEnvPrefix("AXONHUB")
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

    // Set defaults
    setDefaults(v)
    // ...
}
```

### 5.2 Config Structure

**Hierarchical organization:**
```go
type Config struct {
    fx.Out `yaml:"-" json:"-"`  // Exclude from serialization

    DB        db.Config      `conf:"db" yaml:"db" json:"db"`
    Log       log.Config     `conf:"log" yaml:"log" json:"log"`
    APIServer server.Config  `conf:"server" yaml:"server" json:"server"`
    Metrics   metrics.Config `conf:"metrics" yaml:"metrics" json:"metrics"`
    GC        gc.Config      `conf:"gc" yaml:"gc" json:"gc"`
    Cache     xcache.Config  `conf:"cache" yaml:"cache" json:"cache"`
}
```

**Features:**
- Custom decode hooks for complex types (Duration, TextUnmarshaler)
- CLI commands for config validation and preview
- JSON/YAML output support

---

## 6. Pipeline and Middleware System

### 6.1 Request Processing Pipeline

**Pattern:** Chain of responsibility with retry support

```go
// /mnt/ollama/git/axonhub/llm/pipeline/pipeline.go
type pipeline struct {
    Executor              Executor
    Inbound               transformer.Inbound
    Outbound              transformer.Outbound
    middlewares           []Middleware
    maxChannelRetries     int
    maxSameChannelRetries int
    retryDelay            time.Duration
}
```

**Processing Steps:**
1. Transform HTTP request to unified format (Inbound)
2. Apply inbound middlewares
3. Execute with retry logic (same-channel and cross-channel)
4. Transform unified response to provider format (Outbound)
5. Apply outbound middlewares (reverse order)

### 6.2 Middleware Interface

**Comprehensive lifecycle hooks:**

```go
type Middleware interface {
    Name() string

    // Request phases (forward order)
    OnInboundLlmRequest(ctx context.Context, request *llm.Request) (*llm.Request, error)
    OnOutboundRawRequest(ctx context.Context, request *httpclient.Request) (*httpclient.Request, error)

    // Response phases (reverse order)
    OnOutboundRawResponse(ctx context.Context, response *httpclient.Response) (*httpclient.Response, error)
    OnOutboundLlmResponse(ctx context.Context, response *llm.Response) (*llm.Response, error)

    // Stream phases
    OnOutboundRawStream(ctx context.Context, stream streams.Stream[*httpclient.StreamEvent]) (streams.Stream[*httpclient.StreamEvent], error)
    OnOutboundLlmStream(ctx context.Context, stream streams.Stream[*llm.Response]) (streams.Stream[*llm.Response], error)

    // Error handling
    OnOutboundRawError(ctx context.Context, err error)
}
```

---

## 7. Observability Patterns

### 7.1 Structured Logging

**Pattern:** Zap-based with context integration

```go
// /mnt/ollama/git/axonhub/internal/log/logger.go
type Logger struct {
    logger *zap.Logger
    config Config
    hooks  []Hook
}

func (l *Logger) executeHooks(ctx context.Context, msg string, fields ...zap.Field) []zap.Field {
    for _, hook := range globalHooks {
        fields = hook.Apply(ctx, msg, fields...)
    }
    for _, hook := range l.hooks {
        fields = hook.Apply(ctx, msg, fields...)
    }
    return fields
}
```

**Features:**
- Level-based filtering
- Hook system for context enrichment
- File rotation with lumberjack
- Console/JSON encoding options

### 7.2 Distributed Tracing

**Pattern:** Trace ID propagation

```go
// /mnt/ollama/git/axonhub/internal/tracing/tracing.go
func GenerateTraceID() string {
    id := uuid.New()
    return fmt.Sprintf("at-%s", id.String())
}

func GenerateRequestID() string {
    id := uuid.New()
    return fmt.Sprintf("ar-%s", id.String())
}
```

**Features:**
- Configurable trace headers
- Extra trace headers for third-party integration (Claude Code, Codex)
- Body field extraction for trace IDs

### 7.3 Metrics with OpenTelemetry

**Pattern:** Global metrics with categorized counters

```go
// /mnt/ollama/git/axonhub/internal/metrics/metrics.go
type _Metrics struct {
    HTTPRequestCount    metric.Int64Counter
    HTTPRequestDuration metric.Float64Histogram
    ChatRequestCount    metric.Int64Counter
    ChatTokenCount      metric.Int64Counter
}
```

---

## 8. Features That Could Inspire RAD Gateway

### 8.1 High Priority Patterns

| Feature | AxonHub Implementation | RAD Gateway Application |
|---------|----------------------|------------------------|
| **Request Orchestration** | `ChatCompletionOrchestrator` with pluggable middleware | Core gateway request processing pipeline |
| **Load Balancing** | Composite strategy with scoring | Multi-provider routing with health checks |
| **Context Propagation** | Container pattern with typed accessors | Request-scoped data (API keys, tags, traces) |
| **Transformer Pattern** | Inbound/Outbound interfaces | Protocol translation (A2A, AG-UI, MCP) |
| **Pipeline Middleware** | 8 lifecycle hooks | Request/response transformation chain |

### 8.2 Medium Priority Patterns

| Feature | AxonHub Implementation | RAD Gateway Application |
|---------|----------------------|------------------------|
| **Circuit Breaker** | Model-aware circuit breaker | Provider failure handling |
| **Connection Tracking** | Atomic counters | Concurrent request limits |
| **Streaming Abstraction** | Generic Stream[T] interface | A2A/AG-UI streaming support |
| **Config Management** | Viper with custom hooks | Environment-based configuration |
| **GC Worker** | CRON-scheduled cleanup | Audit log cleanup |

### 8.3 Implementation Recommendations

#### 1. Adopt FX Dependency Injection

```go
// Recommended for RAD Gateway
var Module = fx.Module("gateway",
    fx.Provide(NewRouter),
    fx.Provide(NewOrchestrator),
    fx.Provide(NewProviderRegistry),
    fx.Invoke(SetupRoutes),
)
```

#### 2. Implement Transformer Pattern

```go
// Protocol transformer interface
type ProtocolTransformer interface {
    // A2A, AG-UI, MCP support
    TransformToInternal(ctx context.Context, req *protocol.Request) (*internal.Request, error)
    TransformFromInternal(ctx context.Context, resp *internal.Response) (*protocol.Response, error)
}
```

#### 3. Use Context Container Pattern

```go
// RAD Gateway specific context data
type Container struct {
    APIKey      *APIKey
    ProjectID   int
    Tags        []Tag
    TraceID     string
    ControlRoom string
    mu          sync.RWMutex
    Errors      []error
}
```

#### 4. Implement Pipeline Middleware

```go
type GatewayMiddleware interface {
    OnRequest(ctx context.Context, req *Request) (*Request, error)
    OnResponse(ctx context.Context, resp *Response) (*Response, error)
    OnError(ctx context.Context, err error)
}
```

---

## 9. Code Quality Observations

### 9.1 Strengths

1. **Clear Separation of Concerns:** Well-defined layers (API, Biz, Data)
2. **Interface-Based Design:** Heavy use of interfaces for testability
3. **Comprehensive Error Handling:** Structured errors with context
4. **Thread Safety:** Proper use of sync primitives (RWMutex, atomic)
5. **Context Awareness:** Proper context propagation throughout
6. **Extensibility:** Plugin architecture through middleware
7. **Testing:** Mock generation and integration tests

### 9.2 Areas for Improvement

1. **Documentation:** Some packages lack comprehensive documentation
2. **Error Types:** Could benefit from custom error types with error codes
3. **Validation:** Centralized validation logic could be stronger
4. **Magic Numbers:** Some hardcoded values (TTLs, batch sizes)
5. **Function Length:** Some functions are quite long (orchestrator.go Process method)

### 9.3 Code Quality Metrics

| Metric | Observation |
|--------|-------------|
| Cyclomatic Complexity | Moderate - some complex functions in orchestrator |
| Test Coverage | Good - unit and integration tests present |
| Coupling | Low - interface-based decoupling |
| Cohesion | High - single responsibility per package |
| Security | Good - no hardcoded secrets, proper auth middleware |

---

## 10. Conclusion

AxonHub demonstrates several excellent patterns that RAD Gateway should consider:

1. **The Orchestrator Pattern** - Central request processing with pluggable strategies
2. **FX Dependency Injection** - Clean, modular application structure
3. **Transformer Abstraction** - Protocol-agnostic internal representation
4. **Context Container** - Type-safe request-scoped data propagation
5. **Composite Load Balancing** - Strategy pattern for flexible routing
6. **Pipeline Middleware** - Extensible request/response processing

These patterns align well with RAD Gateway's goals of supporting multiple protocols (A2A, AG-UI, MCP) while maintaining clean architecture and observability.

---

## References

- Key Files Analyzed:
  - `/mnt/ollama/git/axonhub/cmd/axonhub/main.go`
  - `/mnt/ollama/git/axonhub/internal/server/server.go`
  - `/mnt/ollama/git/axonhub/internal/server/orchestrator/orchestrator.go`
  - `/mnt/ollama/git/axonhub/internal/server/orchestrator/load_balancer.go`
  - `/mnt/ollama/git/axonhub/llm/pipeline/pipeline.go`
  - `/mnt/ollama/git/axonhub/llm/pipeline/middleware.go`
  - `/mnt/ollama/git/axonhub/internal/contexts/context.go`
  - `/mnt/ollama/git/axonhub/internal/log/logger.go`
  - `/mnt/ollama/git/axonhub/conf/conf.go`
