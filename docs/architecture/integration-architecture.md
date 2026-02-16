# Brass Relay Integration Architecture

## Executive Summary

### Recommended Architecture: Modular Monolith with Clean Internal Boundaries

After evaluating the tradeoffs between deployment complexity, operational overhead, and team velocity, the recommended architecture for Brass Relay is a **Modular Monolith with Clean Internal Boundaries**. This approach balances the simplicity of a single deployable unit with the organizational benefits of well-defined module boundaries.

### Why Modular Monolith Over Microservices

| Factor | Modular Monolith | Microservices |
|--------|------------------|---------------|
| **Deployment Complexity** | Single binary, single container | Orchestration, service mesh, multi-container |
| **Operational Overhead** | Minimal - one process to monitor | High - distributed tracing, circuit breakers, retries |
| **Data Consistency** | In-process, transaction-safe | Distributed transactions, sagas, eventual consistency |
| **Development Velocity** | Fast - refactor across modules easily | Slower - API versioning, contract tests |
| **Scaling** | Horizontal via replicas | Per-service scaling (overkill for current scale) |
| **Resource Efficiency** | Shared memory, no network overhead | Network hops, serialization overhead |

### Architecture Philosophy

> **"Scale teams before scaling services"** - The Modular Monolith enables rapid iteration while the domain model stabilizes. If specific modules require independent scaling later, they can be extracted without architectural revolution.

---

## System Context Diagram

```
                                    External Systems
    ┌─────────────────┬─────────────────┬─────────────────┬─────────────────┐
    │  OpenAI API     │  Anthropic API  │  Google Gemini  │  Other Providers│
    │                 │                 │                 │                 │
    └────────┬────────┴────────┬────────┴────────┬────────┴────────┬────────┘
             │                 │                 │                 │
             └─────────────────┴────────┬────────┴─────────────────┘
                                        │ HTTPS/JSON
                                        ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                           BRASS RELAY GATEWAY                                │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │                        Ingress Layer                                   │  │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────┐  │  │
│  │   │   Health     │  │   Metrics    │  │    A2A       │  │  Admin   │  │  │
│  │   │   /health    │  │  /metrics    │  │ Discovery    │  │   UI     │  │  │
│  │   └──────────────┘  └──────────────┘  └──────────────┘  └──────────┘  │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │                     Protocol Handlers Layer                            │  │
│  │  ┌────────────────┐  ┌────────────────┐  ┌──────────────────────────┐  │  │
│  │  │   Traditional  │  │      A2A       │  │        AG-UI             │  │  │
│  │  │    API Gateway │  │   (Agent-Agent)│  │    (Agent-UI Events)     │  │  │
│  │  │                │  │                │  │                          │  │  │
│  │  │ • /v1/chat/*   │  │ • Agent Cards  │  │ • /v1/agents/{id}/stream │  │  │
│  │  │ • /v1/models   │  │ • Task send    │  │ • Run lifecycle events   │  │  │
│  │  │ • /v1beta/*    │  │ • Task stream  │  │ • Tool call events       │  │  │
│  │  │                │  │ • Task cancel  │  │ • State deltas           │  │  │
│  │  └───────┬────────┘  └───────┬────────┘  └────────────┬─────────────┘  │  │
│  └──────────┼──────────────────┼───────────────────────┼────────────────┘  │
│             │                  │                       │                   │
│             └──────────────────┼───────────────────────┘                   │
│                                ▼                                           │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │                      Core Services Layer                               │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │  │
│  │  │   Gateway    │  │   Router     │  │   Provider   │  │   Task     │  │  │
│  │  │   Core       │  │   Engine     │  │   Registry   │  │   Manager  │  │  │
│  │  │              │  │              │  │              │  │ (A2A/AGUI) │  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  └────────────┘  │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
│                                │                                           │
│                                ▼                                           │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │                     Shared Services Layer                              │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │  │
│  │  │  Auth/AuthZ  │  │    Usage     │  │    Trace     │  │   Config   │  │  │
│  │  │  Middleware  │  │    Sink      │  │    Store     │  │   Manager  │  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  └────────────┘  │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │                     Real-Time Layer (WebSocket/SSE)                    │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐  │  │
│  │  │  Connection Manager  │  Event Router  │  Session Store          │  │  │
│  │  └──────────────────────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────────────┘
                                        │
             ┌──────────────────────────┼──────────────────────────┐
             │                          │                          │
             ▼                          ▼                          ▼
    ┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
    │  Admin Dashboard│      │   Prometheus    │      │   (Future)      │
    │  (React/SPA)    │      │   / Grafana     │      │   Persistent    │
    │                 │      │                 │      │   Storage       │
    │ • Real-time UI  │      │ • Metrics       │      │                 │
    │ • A2A Console   │      │ • Alerting      │      │ • PostgreSQL    │
    │ • System Health │      │ • Dashboards    │      │ • Redis         │
    └─────────────────┘      └─────────────────┘      └─────────────────┘
```

---

## Component Breakdown

### 1. API Gateway (Traditional)

**Responsibility**: Handle all OpenAI-compatible API requests with provider abstraction.

**Current State** (in `/internal/api/`):
- `/v1/chat/completions` - Chat completion proxy
- `/v1/responses` - Response API proxy
- `/v1/messages` - Anthropic-compatible messages
- `/v1/embeddings` - Embedding requests
- `/v1/images/generations` - Image generation
- `/v1/audio/transcriptions` - Audio transcription
- `/v1/models` - Model listing
- `/v1beta/models/*` - Gemini compatibility

**Key Design Decisions**:
- **Protocol Compatibility First**: External contracts remain machine-stable per product principles
- **Provider Adapter Pattern**: Each provider implements a common interface (`internal/provider`)
- **Weighted Routing**: Candidate selection with failover support (`internal/routing`)

**Integration Points**:
```go
// Gateway acts as the unified entry point for all AI operations
type Gateway struct {
    router     *routing.Router        // Routes to providers
    usage      usage.Sink             // Records all usage
    trace      *trace.Store           // Distributed trace events
    taskMgr    *task.Manager          // NEW: A2A task lifecycle
    eventBus   *events.Bus            // NEW: Internal event routing
}
```

### 2. A2A Service (Agent-to-Agent Protocol)

**Responsibility**: Enable agent discovery and task delegation following the A2A specification.

**Endpoints**:
```
GET  /.well-known/agent.json          # Agent Card discovery
POST /a2a/tasks/send                  # Synchronous task execution
POST /a2a/tasks/sendSubscribe         # Streaming task execution (SSE)
GET  /a2a/tasks/{taskId}              # Task status/query
POST /a2a/tasks/{taskId}/cancel       # Task cancellation
```

**Architecture Pattern**: A2A leverages the existing Gateway for actual AI operations

```
Agent A ──A2A Protocol──► Brass Relay ──Traditional API──► Provider
                              │
                              └── Reuses: routing, auth, usage tracking, tracing
```

**Internal Design** (`internal/a2a/`):
```go
package a2a

type Service struct {
    gateway    *core.Gateway      // Reuses existing routing/execution
    agentCard  AgentCard          // Self-description for discovery
    taskStore  TaskStore          // In-memory (future: persistent)
    eventBus   *events.Bus        // For streaming updates
}

// Task execution delegates to the Gateway for provider selection
type Task struct {
    ID          string
    Status      TaskStatus         // pending, working, input-required, completed, canceled
    SessionID   string
    History     []Message
    Artifacts   []Artifact
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### 3. AG-UI Service (Agent-User Interface Protocol)

**Responsibility**: Stream agent run lifecycle events to user interfaces.

**Endpoints**:
```
GET /v1/agents/{agentId}/stream      # SSE stream of agent events
GET /v1/sessions/{sessionId}/events  # Event replay for session continuity
```

**Event Taxonomy** (per AG-UI spec):
- `run.started` - Run initialization
- `run.completed` / `run.failed` / `run.canceled` - Terminal states
- `run.awaiting_input` - Agent needs user input
- `message.delta` - Streaming content
- `tool.called` / `tool.result` - Tool execution
- `state.delta` - Agent state changes

**Integration with A2A**:
- When A2A task executes, AG-UI events are emitted for dashboard visibility
- Single task = single run in AG-UI terms
- Both protocols share the event bus for consistency

### 4. Admin Dashboard (Steampunk-Themed React SPA)

**Responsibility**: Provide real-time operational visibility and management controls.

**Architecture**: Separated SPA served by the Go backend

```
┌─────────────────────────────────────────────────────────┐
│  Go Backend (serves static files + API)                 │
│  ┌─────────────────────────────────────────────────┐    │
│  │  /admin/*  →  Static React build files          │    │
│  │  /api/v0/* →  REST API for dashboard data      │    │
│  │  /ws/*     →  WebSocket for real-time updates   │    │
│  └─────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
                              │
                              │ HTTP / WebSocket
                              ▼
                    ┌─────────────────┐
                    │  React SPA      │
                    │  (Steampunk UI) │
                    │                 │
                    │ • Live metrics  │
                    │ • A2A console   │
                    │ • Provider status│
                    │ • Request traces │
                    └─────────────────┘
```

**Key Dashboard Views**:

| View | Data Source | Update Frequency |
|------|-------------|------------------|
| **System Overview** | Prometheus metrics API | 5s polling |
| **Request Stream** | Usage sink + trace store | WebSocket real-time |
| **A2A Console** | Task store + event bus | WebSocket real-time |
| **Provider Status** | Health checks + routing metrics | 10s polling |
| **Configuration** | Config manager | On-demand |

**Theming Strategy**:
- Steampunk aesthetic is pure presentation layer
- Metric names remain standard (no "brass_cog_rotations" - use `http_requests_total`)
- UI copy uses steampunk language ("Steam Pressure" for load, "Gauge" for metrics)
- Alert messages maintain technical clarity with optional steampunk garnish

### 5. Metrics and Observability

**Three-Tier Observability Stack**:

```
┌─────────────────────────────────────────────────────────────┐
│  LAYER 1: In-Application (Go code)                          │
│  • Usage sink (per-request records)                         │
│  • Trace store (event timeline)                             │
│  • Prometheus metrics (counters, histograms, gauges)        │
├─────────────────────────────────────────────────────────────┤
│  LAYER 2: Aggregation (In-process)                          │
│  • Admin API query endpoints                                │
│  • Real-time event bus for WebSocket distribution           │
│  • Memory-bounded circular buffers                          │
├─────────────────────────────────────────────────────────────┤
│  LAYER 3: External (Optional)                               │
│  • Prometheus scrape endpoint (/metrics)                    │
│  • OpenTelemetry export (future)                            │
│  • Persistent storage adapter (future)                      │
└─────────────────────────────────────────────────────────────┘
```

**Metric Naming Conventions** (standard Prometheus):
```
# Request metrics
brass_requests_total{api_type="chat",provider="openai",status="success"}
brass_request_duration_seconds_bucket{api_type="embeddings",le="0.1"}

# Provider metrics
brass_provider_health{provider="anthropic"}  # 1 = healthy, 0 = unhealthy
brass_provider_latency_seconds{provider="gemini",quantile="0.99"}

# A2A metrics
brass_a2a_tasks_total{status="completed"}
brass_a2a_tasks_active
brass_a2a_task_duration_seconds

# AG-UI metrics
brass_agui_connections_active
brass_agui_events_sent_total{event_type="message.delta"}

# System metrics
brass_memory_usage_bytes
brass_goroutines_active
```

---

## Integration Patterns

### Pattern 1: Unified Request Flow

All requests (traditional API, A2A tasks, AG-UI runs) flow through the same core pipeline:

```
┌──────────┐     ┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Client  │────►│   Handler   │────►│    Core      │────►│   Router    │
│          │     │  (Protocol- │     │   Gateway    │     │   Engine    │
│          │     │   specific) │     │              │     │             │
└──────────┘     └─────────────┘     └──────────────┘     └──────┬──────┘
                                                                  │
                              ┌───────────────────────────────────┘
                              │
                              ▼
                       ┌─────────────┐     ┌─────────────┐
                       │   Usage     │     │   Trace     │
                       │    Sink     │     │    Store    │
                       └─────────────┘     └─────────────┘
```

**Benefits**:
- Single auth enforcement point
- Unified usage tracking across all protocols
- Consistent retry/failover behavior
- Simplified debugging (one trace per request)

### Pattern 2: Event-Driven Real-Time Updates

The event bus decouples protocol handlers from real-time delivery:

```go
// Core publishes events
eventBus.Publish(Event{
    Type: "request.completed",
    Data: RequestCompletedEvent{...},
})

// Multiple subscribers receive independently
- WebSocket hub → dashboard clients
- AG-UI stream → user interfaces
- A2A notification → delegating agents
- Metrics aggregator → Prometheus gauges
```

### Pattern 3: Shared Authentication Context

```go
// Middleware extracts and validates credentials
// Context carries auth identity through entire request lifecycle
type AuthContext struct {
    APIKeyName    string           // For traditional API
    AgentID       string           // For A2A agent identity
    SessionID     string           // For AG-UI sessions
    Permissions   []Permission     // RBAC (future)
    QuotaWindow   QuotaWindow      // Rate limit tracking
}
```

**Auth Flow**:
1. Extract credential (Bearer token, API key, A2A agent assertion)
2. Validate against appropriate store (API keys, Agent registry)
3. Enrich context with identity and permissions
4. Pass to handler - uniform access via `ctx.Value(AuthCtxKey)`

### Pattern 4: Protocol Translation Layer

A2A and AG-UI are **presentation protocols** over the same core capabilities:

```
┌─────────────────────────────────────────────────────────────┐
│                    Protocol Handlers                         │
│  ┌───────────────┐  ┌───────────────┐  ┌─────────────────┐  │
│  │  Traditional  │  │      A2A      │  │     AG-UI       │  │
│  │     API       │  │               │  │                 │  │
│  │               │  │ • Task model  │  │ • Event stream  │  │
│  │ • OpenAI fmt  │  │ • Agent cards │  │ • State deltas  │  │
│  │ • Anthropic   │  │ • JSON-RPC    │  │ • Session replay│  │
│  └───────┬───────┘  └───────┬───────┘  └────────┬────────┘  │
│          │                  │                   │            │
│          └──────────────────┼───────────────────┘            │
│                             ▼                                │
│  ┌────────────────────────────────────────────────────────┐  │
│  │              Unified Core Operations                    │  │
│  │  • Provider selection                                  │  │
│  │  • Request execution                                   │  │
│  │  • Token counting                                      │  │
│  │  • Error handling                                      │  │
│  └────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## API Design

### REST API for Dashboard

**Base Path**: `/api/v0/admin/`

#### System Status
```http
GET /api/v0/admin/status
Response: {
    "status": "operational",
    "version": "0.4.0",
    "uptime": "72h15m",
    "components": {
        "gateway": "healthy",
        "providers": {"openai": "healthy", "anthropic": "degraded"},
        "a2a": "healthy"
    }
}
```

#### Metrics Endpoints
```http
GET /api/v0/admin/metrics/summary?window=5m
Response: {
    "requests": {"total": 15234, "rate": 45.2},
    "latency": {"p50": 120, "p99": 850},
    "providers": {
        "openai": {"requests": 12000, "error_rate": 0.01},
        "anthropic": {"requests": 3234, "error_rate": 0.05}
    }
}

GET /api/v0/admin/metrics/usage?start=2024-01-01&end=2024-01-02
Response: {
    "data": [...],
    "summary": {"total_tokens": 45000000, "total_cost": 12.50}
}
```

#### A2A Management
```http
GET /api/v0/admin/a2a/tasks?status=active&limit=50
Response: {
    "tasks": [...],
    "pagination": {"cursor": "xxx", "has_more": true}
}

GET /api/v0/admin/a2a/tasks/{taskId}
Response: Task details with full history

POST /api/v0/admin/a2a/tasks/{taskId}/cancel
Response: {"status": "canceled"}
```

#### Configuration
```http
GET /api/v0/admin/config
Response: Current configuration (sanitized)

POST /api/v0/admin/config/routes
Body: {"model": "gpt-4o", "candidates": [...]}
Response: {"updated": true}
```

### WebSocket API for Real-Time Updates

**Endpoint**: `/ws/admin`

**Connection Flow**:
1. Client connects with auth token in query param or header
2. Server validates and accepts connection
3. Server sends initial state snapshot
4. Server pushes incremental updates

**Message Protocol**:
```json
// Server → Client: Initial state
{
    "type": "init",
    "data": {
        "active_requests": 23,
        "recent_traces": [...],
        "provider_status": {...}
    }
}

// Server → Client: Live update
{
    "type": "update",
    "channel": "requests",
    "data": {
        "request_id": "abc123",
        "event": "completed",
        "latency_ms": 234,
        "provider": "openai"
    }
}

// Client → Server: Subscribe to channel
{
    "action": "subscribe",
    "channel": "a2a.tasks"
}

// Server → Client: Heartbeat
{
    "type": "ping",
    "timestamp": 1704067200
}
```

**Channels**:
- `requests` - New requests, completions, errors
- `traces` - Trace events as they occur
- `a2a.tasks` - Task lifecycle updates
- `providers` - Provider health changes
- `system` - Configuration changes, alerts

### Internal APIs Between Components

**Event Bus Interface** (`internal/events/`):
```go
type Bus interface {
    Publish(event Event)
    Subscribe(filter EventFilter, handler EventHandler) Subscription
    Close() error
}

type Event struct {
    ID        string
    Type      string           // e.g., "request.completed", "a2a.task.updated"
    Timestamp time.Time
    Source    string           // Component that emitted
    Payload   any              // Type-specific data
    Metadata  map[string]string // Trace IDs, request IDs, etc.
}
```

**Task Manager Interface** (`internal/task/`):
```go
type Manager interface {
    Create(ctx context.Context, spec TaskSpec) (*Task, error)
    Get(ctx context.Context, taskID string) (*Task, error)
    Cancel(ctx context.Context, taskID string) error
    Subscribe(ctx context.Context, taskID string) (<-chan TaskUpdate, error)
    List(ctx context.Context, filter TaskFilter) ([]*Task, error)
}
```

---

## Data Flow Examples

### Example 1: Traditional Request (Chat Completion)

```
┌──────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌─────────┐
│Client│───►│Middleware│───►│  Handler │───►│  Gateway │───►│ Router  │
└──────┘    └──────────┘    └──────────┘    └──────────┘    └────┬────┘
   │                                                              │
   │  POST /v1/chat/completions                                   │
   │  Authorization: Bearer sk-xxx                                │
   │  {model: "gpt-4o", messages: [...]}                          │
   │                                                          ┌───┴───┐
   │                                                          │OpenAI │
   │                                                          │Adapter│
   │                                                          └───┬───┘
   │                                                              │
   │  {choices: [...], usage: {...}}                              │
   │◄─────────────────────────────────────────────────────────────┘
   │
   │  (Parallel: UsageSink.Record(), TraceStore.Add(), EventBus.Publish())
```

### Example 2: A2A Task Delegation

```
┌─────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌─────────┐
│ Agent A │───►│  A2A     │───►│  Task    │───►│  Gateway │───►│ Router  │
│(Client) │    │ Handler  │    │ Manager  │    │  (Core)  │    │         │
└─────────┘    └──────────┘    └──────────┘    └──────────┘    └────┬────┘
                    │                      │                          │
   POST /a2a/tasks/sendSubscribe        │                          │
   {message: {role:"user", parts:[...]}}│                          │
                    │                   │                          │
                    ▼                   │                          ▼
             Create task record         │                     ┌─────────┐
             Return SSE stream          │                     │Provider │
                    │                   │                     └────┬────┘
                    │                   │                          │
                    │    EventBus.Publish(TaskUpdate) ◄────────────┘
                    │                   │
                    ▼                   ▼
             SSE: {type:"task.status", status:"working"}
             SSE: {type:"task.artifact", artifact: {...}}
             SSE: {type:"task.status", status:"completed"}
```

### Example 3: Dashboard Real-Time View

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  React   │───►│WebSocket │───►│   Hub    │───►│Event Bus │
│Dashboard │    │  Client  │    │          │    │          │
└──────────┘    └──────────┘    └──────────┘    └────┬─────┘
     │                                               │
     │ Connect /ws/admin                             │
     │──────────────────────────────────────────────►│
     │                                               │
     │ Send init snapshot                            │
     │◄──────────────────────────────────────────────│
     │                                               │
     │                                    ┌──────────┴──────────┐
     │                                    │  New request event  │
     │                                    │  from Gateway       │
     │                                    └──────────┬──────────┘
     │                                               │
     │ Broadcast to channel subscribers              │
     │◄──────────────────────────────────────────────│
     │                                               │
     │ Update UI: Add request to live table          │
```

---

## Deployment Architecture

### Single Binary Deployment (Current)

```
┌─────────────────────────────────────────────────────────────┐
│                    Docker Container                         │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              brass-relay binary                     │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌────────┐ │   │
│  │  │   API   │  │   A2A   │  │  AG-UI  │  │  Admin │ │   │
│  │  │Handlers │  │Handlers │  │Handlers │  │  API   │ │   │
│  │  └────┬────┘  └────┬────┘  └────┬────┘  └───┬────┘ │   │
│  │       └─────────────┴─────────────┴─────────┘      │   │
│  │                      │                             │   │
│  │              ┌───────┴───────┐                     │   │
│  │              │  Shared Core  │                     │   │
│  │              └───────────────┘                     │   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                 │
│                      ┌────┴────┐                           │
│                      │  :8090  │                           │
│                      └────┬────┘                           │
└───────────────────────────┼─────────────────────────────────┘
                            │
                     ┌──────┴──────┐
                     │  Load Balancer │
                     │  (nginx/traefik)│
                     └─────────────┘
```

**Container Specifications**:
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o brass-relay ./cmd/rad-gateway

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/brass-relay .
EXPOSE 8090
CMD ["./brass-relay"]
```

**Resource Requirements**:
- CPU: 0.5 cores minimum, 2 cores recommended
- Memory: 256MB minimum, 1GB recommended (in-memory stores)
- Network: Standard HTTPS egress to providers

### Environment Configuration

```yaml
# Required
RAD_LISTEN_ADDR: ":8090"
RAD_API_KEYS: "admin:secret-key-1,service:secret-key-2"

# Optional
RAD_RETRY_BUDGET: "2"
RAD_LOG_LEVEL: "info"
RAD_METRICS_ENABLED: "true"
RAD_A2A_ENABLED: "true"
RAD_AGUI_ENABLED: "true"
RAD_ADMIN_ENABLED: "true"
RAD_WEBSOCKET_MAX_CONNS: "1000"
RAD_USAGE_BUFFER_SIZE: "10000"
RAD_TRACE_BUFFER_SIZE: "50000"

# Future (persistent storage)
# RAD_STORAGE_BACKEND: "redis"
# REDIS_URL: "redis://localhost:6379"
```

### Horizontal Scaling Considerations

**Current Limitation**: In-memory stores (usage, trace, task state) are not shared across instances.

**Scaling Options**:

1. **Shared Nothing** (simplest):
   - Each instance independent
   - Client stickiness via load balancer (session affinity)
   - Acceptable for dashboard views (slightly stale data)

2. **Shared State** (future):
   - Redis for task state, session management
   - PostgreSQL for usage/traces persistence
   - WebSocket pub/sub via Redis

3. **Read Replicas**:
   - Write to central store
   - Each instance caches recent data
   - Eventual consistency for dashboard views

---

## Scalability Considerations

### Request Throughput

**Current Architecture Limits**:
- Go HTTP server: 10K+ concurrent connections
- In-memory stores: ~100K records before rotation
- WebSocket: ~1K concurrent connections (configurable)

**Bottlenecks and Mitigations**:

| Bottleneck | Symptom | Mitigation |
|------------|---------|------------|
| Provider latency | Slow responses | Connection pooling, aggressive timeouts |
| Memory usage | OOM kills | Persistent storage adapter, smaller buffers |
| WebSocket connections | Connection refused | Horizontal scaling, Redis pub/sub |
| CPU | High latency | Profile hot paths, provider response caching |

### State Management Strategy

**Transient State** (in-memory, loss acceptable):
- Active request tracking
- WebSocket connections
- Hot metric caches

**Important State** (survive restart):
- A2A task history
- Usage records (long-term)
- Configuration changes

**Critical State** (never lose):
- In-flight payments/charges (future)
- Audit logs
- Security events

### Caching Strategy

```
┌─────────────────────────────────────────────────────────────┐
│                    Caching Layers                           │
├─────────────────────────────────────────────────────────────┤
│  L1: Request-scoped (context values) - zero latency         │
├─────────────────────────────────────────────────────────────┤
│  L2: In-memory LRU (model lists, provider health)           │
│      TTL: 30s, Size: 1000 entries                           │
├─────────────────────────────────────────────────────────────┤
│  L3: Redis (future - shared across instances)               │
│      TTL: 5m, Invalidation: pub/sub                         │
├─────────────────────────────────────────────────────────────┤
│  L4: Provider response caching (respect Cache-Control)      │
│      Model lists, pricing info, etc.                        │
└─────────────────────────────────────────────────────────────┘
```

---

## Security Model

### Authentication Strategy

**Multi-Protocol Auth**:

```
┌─────────────────────────────────────────────────────────────┐
│                  Auth Middleware Chain                      │
├─────────────────────────────────────────────────────────────┤
│  1. Extract credential from request                         │
│     • Bearer token (Authorization header)                   │
│     • API key (X-Api-Key header)                            │
│     • Query parameter (key=xxx)                             │
│     • A2A agent assertion (Agent-ID header)                 │
├─────────────────────────────────────────────────────────────┤
│  2. Validate credential                                     │
│     • API keys: constant-time comparison                    │
│     • Agent assertions: signature verification (future)     │
│     • OAuth tokens: introspection (future)                  │
├─────────────────────────────────────────────────────────────┤
│  3. Enrich context                                          │
│     • Identity (who)                                        │
│     • Permissions (what they can do)                        │
│     • Quota window (rate limit tracking)                    │
├─────────────────────────────────────────────────────────────┤
│  4. Enforce authorization                                   │
│     • Endpoint access control                               │
│     • Model access restrictions                             │
│     • Administrative privileges                             │
└─────────────────────────────────────────────────────────────┘
```

### Service-to-Service Auth (Internal)

Within the monolith, auth is enforced at the ingress:
- Internal components trust the context passed from handlers
- No re-auth between Gateway → Router → Provider
- Defense in depth: validate invariants, not identity

### Dashboard Security

```go
// Admin API middleware
func AdminAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Separate admin credentials from API keys
        adminKey := extractAdminKey(r)
        if !validateAdminKey(adminKey) {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        // Add admin context
        ctx := context.WithValue(r.Context(), IsAdminKey, true)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

**Security Headers**:
```go
w.Header().Set("Content-Security-Policy", "default-src 'self'")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("Strict-Transport-Security", "max-age=31536000")
```

### Secret Management

**Current**: Environment variables
**Future**: Integration with secret management systems

```
Development:  .env file (gitignored)
Staging:      Environment variables (container orchestration)
Production:   Vault / AWS Secrets Manager / Azure Key Vault
              ↓
              Mounted as files or injected as env vars
              ↓
              Application reads at startup
```

---

## Implementation Phases

### Phase 1: Core Integration (Weeks 1-2)

**Goal**: Establish the foundation for unified architecture

**Deliverables**:
- [ ] Event bus implementation (`internal/events/`)
- [ ] WebSocket/SSE infrastructure (`internal/streaming/`)
- [ ] Enhanced metrics collection with Prometheus format
- [ ] Admin API endpoints for dashboard data

**Key Decisions**:
- Event bus uses Go channels (in-process, sufficient for Phase 1)
- WebSocket hub maintains client connections
- Metrics endpoint at `/metrics` (Prometheus format)

**Testing**:
- Unit tests for event bus
- Integration tests for WebSocket lifecycle
- Load tests for concurrent connections

### Phase 2: A2A Integration (Weeks 3-4)

**Goal**: Full A2A protocol support with dashboard visibility

**Deliverables**:
- [ ] Agent Card discovery endpoint (`/.well-known/agent.json`)
- [ ] Task lifecycle endpoints (`/a2a/tasks/*`)
- [ ] Task manager with in-memory store
- [ ] A2A console in dashboard
- [ ] Task event streaming via WebSocket

**Integration Points**:
- A2A service reuses Gateway for provider execution
- Task updates flow through event bus to dashboard
- Usage tracking includes A2A operations

**Testing**:
- A2A protocol compliance tests
- Task lifecycle state machine validation
- Concurrent task execution stress tests

### Phase 3: AG-UI Integration (Weeks 5-6)

**Goal**: User-facing agent event streaming

**Deliverables**:
- [ ] AG-UI event taxonomy implementation
- [ ] `/v1/agents/{id}/stream` SSE endpoint
- [ ] Session replay capability
- [ ] Event compaction for long sessions
- [ ] Dashboard run visualization

**Integration Points**:
- AG-UI events originate from Gateway during execution
- Event bus routes to both AG-UI clients and dashboard
- Session store enables replay

### Phase 4: Dashboard Polish (Weeks 7-8)

**Goal**: Production-ready steampunk-themed admin interface

**Deliverables**:
- [ ] React SPA with steampunk styling
- [ ] Real-time metrics charts
- [ ] A2A task management UI
- [ ] Provider health visualization
- [ ] Request tracing viewer
- [ ] Configuration management UI

**Technical**:
- Static file serving from Go binary
- WebSocket client with reconnection logic
- Responsive design for mobile access

### Phase 5: Hardening (Weeks 9-10)

**Goal**: Production readiness

**Deliverables**:
- [ ] Persistent storage adapter (PostgreSQL)
- [ ] Redis integration for shared state
- [ ] Comprehensive security audit
- [ ] Performance optimization
- [ ] Documentation and runbooks

**Success Criteria**:
- 99.9% availability under load
- <100ms p99 latency for cached operations
- Graceful degradation when providers fail
- Secure against OWASP Top 10

---

## Appendix A: Directory Structure

```
cmd/rad-gateway/
├── main.go                      # Bootstrap, dependency injection

internal/
├── api/                         # Traditional API handlers
│   ├── handlers.go
│   └── handlers_test.go
│
├── a2a/                         # Agent-to-Agent protocol
│   ├── handlers.go              # HTTP handlers for A2A endpoints
│   ├── service.go               # A2A business logic
│   ├── models.go                # Task, AgentCard, Artifact types
│   ├── store.go                 # Task storage interface
│   └── memory_store.go          # In-memory implementation
│
├── agui/                        # Agent-UI protocol
│   ├── handlers.go              # SSE streaming handlers
│   ├── events.go                # Event type definitions
│   ├── session.go               # Session management
│   └── replay.go                # Event replay logic
│
├── admin/                       # Admin API (expanded)
│   ├── handlers.go              # Current: config, usage, traces
│   ├── dashboard.go             # NEW: dashboard data endpoints
│   ├── websocket.go             # NEW: WebSocket hub
│   └── auth.go                  # NEW: admin authentication
│
├── core/                        # Gateway core (existing)
│   ├── gateway.go
│   └── gateway_test.go
│
├── events/                      # NEW: Event bus
│   ├── bus.go                   # Interface and implementation
│   ├── types.go                 # Event type definitions
│   └── channels.go              # Channel-based pub/sub
│
├── streaming/                   # NEW: WebSocket/SSE primitives
│   ├── websocket.go             # WebSocket upgrader and hub
│   ├── sse.go                   # SSE writer utilities
│   └── conn.go                  # Connection management
│
├── metrics/                     # NEW: Metrics collection
│   ├── prometheus.go            # Prometheus registry and collectors
│   ├── recorder.go              # Application metrics recording
│   └── dashboard.go             # Dashboard-oriented aggregations
│
├── middleware/                  # HTTP middleware (existing + new)
│   ├── middleware.go            # Auth, request context
│   ├── middleware_test.go
│   ├── admin.go                 # NEW: Admin auth
│   ├── cors.go                  # NEW: CORS for dashboard
│   └── ratelimit.go             # NEW: Rate limiting (future)
│
├── routing/                     # Provider routing (existing)
│   ├── router.go
│   └── router_test.go
│
├── provider/                    # Provider adapters (existing)
│   ├── provider.go
│   └── mock.go
│
├── models/                      # Shared DTOs (existing)
│   └── models.go
│
├── usage/                       # Usage tracking (existing)
│   ├── usage.go
│   └── projection.go            # NEW: Query projections
│
├── trace/                       # Distributed tracing (existing)
│   └── trace.go
│
└── config/                      # Configuration (existing)
    └── config.go

web/                             # NEW: React dashboard
├── package.json
├── tsconfig.json
├── src/
│   ├── main.tsx                 # Entry point
│   ├── App.tsx                  # Root component
│   ├── components/              # Reusable UI components
│   │   ├── Gauge.tsx            # Steampunk-styled metric display
│   │   ├── PressureMeter.tsx    # Load visualization
│   │   ├── TaskList.tsx         # A2A task management
│   │   └── TraceViewer.tsx      # Request trace display
│   ├── pages/                   # Top-level views
│   │   ├── Dashboard.tsx        # Main overview
│   │   ├── A2AConsole.tsx       # Agent management
│   │   ├── Providers.tsx        # Provider status
│   │   └── Settings.tsx         # Configuration
│   ├── hooks/                   # React hooks
│   │   ├── useWebSocket.ts      # WebSocket connection management
│   │   ├── useMetrics.ts        # Metrics polling
│   │   └── useTasks.ts          # A2A task queries
│   ├── api/                     # API clients
│   │   ├── client.ts            # HTTP client
│   │   └── types.ts             # TypeScript interfaces
│   └── styles/                  # Steampunk theme
│       ├── theme.css
│       └── components.css
└── dist/                        # Build output (served by Go)

docs/
├── architecture/
│   └── integration-architecture.md   # This document
├── operations/
│   ├── deployment.md
│   ├── runbooks.md
│   └── alerting.md
└── protocols/
    ├── a2a-integration.md
    └── agui-integration.md
```

---

## Appendix B: Technology Choices Summary

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| **Backend** | Go 1.24 | Existing stack, excellent concurrency |
| **Frontend** | React + TypeScript | Component model, type safety |
| **Styling** | CSS Modules + Custom | Steampunk theming flexibility |
| **Real-time** | WebSocket + SSE | WebSocket for bidirectional, SSE for streaming |
| **Metrics** | Prometheus client | Industry standard, easy export |
| **Event Bus** | Go channels | In-process, zero latency, no dependencies |
| **Storage** | In-memory (now) / Redis+PG (future) | Progressive enhancement |
| **Container** | Docker + Alpine | Small footprint, secure |

---

## Appendix C: Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Memory exhaustion** | High | Bounded buffers, aggressive rotation, persistent storage path |
| **WebSocket DoS** | Medium | Connection limits, auth required, rate limiting |
| **Provider API changes** | Medium | Adapter pattern, contract tests, feature flags |
| **A2A spec evolution** | Low | Clean abstraction layer, version negotiation |
| **Theme-induced complexity** | Medium | Strict separation: theme = presentation only |
| **Dashboard performance** | Medium | Pagination, virtualization, debounced updates |

---

## Conclusion

This integration architecture provides a cohesive, scalable foundation for Brass Relay that:

1. **Preserves compatibility** - All existing APIs remain stable
2. **Enables agent interoperability** - A2A and AG-UI protocols are first-class
3. **Provides operational visibility** - Real-time dashboard with steampunk character
4. **Scales incrementally** - Start simple, add persistence when needed
5. **Maintains developer velocity** - Modular monolith enables rapid iteration

The architecture honors the product principle: **"Theme without ambiguity"** - the steampunk aesthetic enhances the user experience without compromising operational clarity or protocol compatibility.
