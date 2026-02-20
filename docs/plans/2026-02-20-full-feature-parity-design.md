# Full Feature Parity Design: RAD Gateway vs AxonHub + Plexus

**Date**: 2026-02-20
**Status**: Approved
**Team**: Hotel (H Agents)
**Approach**: Bottom-Up (A2A Protocol -> Adapters -> APIs -> UI)

---

## 1. Objective

Bring RAD Gateway to full feature parity with both AxonHub (Go-based AI platform) and Plexus (TypeScript/Node.js gateway), including full A2A (Agent-to-Agent) protocol compliance per the official specification at https://a2a-protocol.org/latest/specification/.

## 2. Current State Summary

### RAD Gateway (What We Have)
- **Backend**: Deployed on 172.16.30.45:8090, PostgreSQL, 45+ API endpoints
- **Auth**: JWT + API key (Bearer, x-api-key, x-goog-api-key)
- **Providers**: MockAdapter for OpenAI/Anthropic/Gemini (no real HTTP clients)
- **A2A**: Custom implementation (not JSON-RPC 2.0 compliant, custom method names)
- **UI**: 15 pages in Next.js (Dashboard, Control Rooms, Providers, API Keys, Projects, Usage, A2A, OAuth, MCP, Reports)
- **Streaming**: SSE infrastructure with race condition fixes
- **Cost/Usage**: Calculator, aggregator, worker, in-memory + DB storage

### AxonHub (Reference - Go)
- Real provider adapters: OpenAI, Anthropic, Gemini, Jina, AI SDK
- Full user management: users, roles, projects with RBAC
- Playground/chat with channel selection
- Tracing/threads with timeline visualization
- GraphQL admin API + REST
- Model management with pricing
- Settings pages (appearance, notifications, profile)
- System initialization flow
- Codex + ClaudeCode OAuth integration
- Load balancing with adaptive multi-strategy

### Plexus (Reference - TypeScript)
- Real provider adapters via transformer pipeline
- Quota system: scheduler, enforcer, user quotas
- Cooldown manager per provider
- Performance metrics with Prometheus/Grafana
- Error tracking with dedicated UI
- System logs with dedicated UI
- Debug console
- Live metrics page
- MCP proxy with usage tracking
- OAuth auth manager
- SQLite database with migrations

## 3. Architecture

### Phase 1: A2A Protocol Compliance

The A2A protocol is central to RAD Gateway's identity as an agent-to-agent gateway. Full compliance is the foundation.

#### 3.1 JSON-RPC 2.0 Transport Layer

**New file**: `internal/a2a/jsonrpc.go`

The A2A spec requires JSON-RPC 2.0 envelope for all methods:

```
Request:  {"jsonrpc":"2.0","method":"message/send","params":{...},"id":"req-1"}
Response: {"jsonrpc":"2.0","result":{...},"id":"req-1"}
Error:    {"jsonrpc":"2.0","error":{"code":-32600,"message":"..."},"id":"req-1"}
```

A single HTTP POST endpoint `/a2a` will dispatch to method handlers based on the `method` field. The existing REST-style endpoints (`/a2a/tasks/send`, etc.) will be preserved as aliases for backward compatibility but the canonical interface is JSON-RPC.

#### 3.2 Spec-Compliant Methods

| JSON-RPC Method | Maps To | Description |
|----------------|---------|-------------|
| `message/send` | `handleSendMessage` | Send message, create/continue task |
| `message/stream` | `handleSendStreamingMessage` | SSE streaming response |
| `tasks/get` | `handleGetTask` | Get task by ID with optional history |
| `tasks/list` | `handleListTasks` | List tasks with filtering/pagination |
| `tasks/cancel` | `handleCancelTask` | Cancel active task |
| `tasks/resubscribe` | `handleResubscribe` | Reconnect SSE for active task |
| `tasks/pushNotificationConfig/set` | `handleSetPushConfig` | Configure webhook |
| `tasks/pushNotificationConfig/get` | `handleGetPushConfig` | Get webhook config |
| `tasks/pushNotificationConfig/list` | `handleListPushConfigs` | List webhook configs |
| `tasks/pushNotificationConfig/delete` | `handleDeletePushConfig` | Remove webhook |
| `agent/authenticatedExtendedCard` | `handleGetExtendedCard` | Authenticated agent card |

#### 3.3 AgentCard Schema (v2)

**Update**: `/.well-known/agent.json`

```json
{
  "id": "rad-gateway",
  "name": "RAD Gateway",
  "description": "AI API Gateway with A2A protocol support",
  "version": "1.0.0",
  "provider": {
    "organization": "RAD",
    "url": "https://radgateway.io"
  },
  "url": "https://172.16.30.45:8090/a2a",
  "capabilities": {
    "streaming": true,
    "pushNotifications": true,
    "extendedAgentCard": true
  },
  "skills": [...],
  "interfaces": [{
    "id": "chat",
    "name": "Chat Completion",
    "inputModes": ["text"],
    "outputModes": ["text"]
  }],
  "securitySchemes": [{
    "type": "apiKey",
    "in": "header",
    "name": "Authorization"
  }],
  "security": [{"apiKey": []}]
}
```

#### 3.4 Message/Part Data Model

Replace plain string `content` with the A2A Part model:

```go
type Part interface{ partType() string }
type TextPart struct { Text string }
type FilePart struct { File FileContent }
type DataPart struct { Data map[string]interface{} }

type Message struct {
    Role      string
    Parts     []Part
    Metadata  map[string]interface{}
}
```

#### 3.5 Task Model Updates

- Add `contextId` field for conversation grouping
- Add `history` field ([]Message) for conversation history
- Add `stateTransitionHistory` for audit trail
- Artifacts use Part array instead of raw JSON

#### 3.6 Push Notifications

- New table: `a2a_push_notification_configs`
- Webhook delivery via background goroutine
- HMAC signature for webhook validation
- Retry with exponential backoff

#### 3.7 Error Codes

Standard JSON-RPC 2.0 + A2A-specific:

| Code | Name | Description |
|------|------|-------------|
| -32700 | ParseError | Invalid JSON |
| -32600 | InvalidRequest | Invalid JSON-RPC request |
| -32601 | MethodNotFound | Method not found |
| -32602 | InvalidParams | Invalid parameters |
| -32603 | InternalError | Internal error |
| -32001 | TaskNotFoundError | Task ID not found |
| -32002 | TaskNotCancelableError | Task in terminal state |
| -32003 | PushNotificationNotSupportedError | Push not enabled |
| -32004 | UnsupportedOperationError | Capability not declared |
| -32005 | ContentTypeNotSupportedError | Unsupported content type |

#### 3.8 Files to Create/Modify

| File | Action | Purpose |
|------|--------|---------|
| `internal/a2a/jsonrpc.go` | Create | JSON-RPC 2.0 transport, method dispatch |
| `internal/a2a/jsonrpc_errors.go` | Create | Error code definitions |
| `internal/a2a/methods.go` | Create | All 11 method handlers |
| `internal/a2a/parts.go` | Create | TextPart, FilePart, DataPart types |
| `internal/a2a/agent_card.go` | Create | Spec-compliant AgentCard schema |
| `internal/a2a/push.go` | Create | Push notification config + delivery |
| `internal/a2a/models.go` | Modify | Update ModelCard -> AgentCard alignment |
| `internal/a2a/task.go` | Modify | Add contextId, history, stateTransitionHistory |
| `internal/a2a/task_handlers.go` | Modify | Wire into JSON-RPC dispatch |
| `internal/a2a/handlers.go` | Modify | Add JSON-RPC endpoint registration |
| `internal/a2a/task_store.go` | Modify | Add ListTasks with filtering |
| `internal/a2a/task_store_pg.go` | Modify | PostgreSQL queries for new fields |
| `cmd/rad-gateway/main.go` | Modify | Register `/a2a` JSON-RPC endpoint |

### Phase 2: Real Provider Adapters

Replace MockAdapter with real HTTP clients for all providers.

#### 2.1 OpenAI Adapter

| Endpoint | Method | Details |
|----------|--------|---------|
| `/v1/chat/completions` | POST | Chat with streaming support |
| `/v1/responses` | POST | Responses API (partial) |
| `/v1/embeddings` | POST | Text embeddings |
| `/v1/models` | GET | List models |
| `/v1/images/generations` | POST | DALL-E image generation |
| `/v1/audio/transcriptions` | POST | Whisper transcription |
| `/v1/audio/speech` | POST | TTS |

**Files**: `internal/provider/openai/adapter.go`, `internal/provider/openai/client.go`, `internal/provider/openai/streaming.go`

**Reference**: AxonHub `api.OpenAIHandlers` + Plexus `transformers/openai.ts`

#### 2.2 Anthropic Adapter

| Endpoint | Method | Details |
|----------|--------|---------|
| `/v1/messages` | POST | Messages API with streaming |
| `/anthropic/v1/messages` | POST | Native format passthrough |
| `/anthropic/v1/models` | GET | List models |

**Files**: `internal/provider/anthropic/adapter.go`, `internal/provider/anthropic/client.go`, `internal/provider/anthropic/streaming.go`

**Reference**: AxonHub `api.AnthropicHandlers` + Plexus `transformers/anthropic.ts`

#### 2.3 Gemini Adapter

| Endpoint | Method | Details |
|----------|--------|---------|
| `/v1beta/models/{model}:generateContent` | POST | Content generation |
| `/v1beta/models/{model}:streamGenerateContent` | POST | Streaming |
| `/v1beta/models` | GET | List models |

**Files**: `internal/provider/gemini/adapter.go`, `internal/provider/gemini/client.go`, `internal/provider/gemini/streaming.go`

**Reference**: AxonHub `api.GeminiHandlers` + Plexus `transformers/gemini.ts`

#### 2.4 Shared Infrastructure

| Component | File | Purpose |
|-----------|------|---------|
| HTTP client pool | `internal/provider/httpclient.go` | Shared http.Client with timeouts, retries |
| Request transformer | `internal/provider/transform.go` | Cross-format request translation |
| Response normalizer | `internal/provider/normalize.go` | Normalize responses to internal format |
| Stream bridge | `internal/provider/stream_bridge.go` | Provider SSE -> internal SSE translation |
| Token counter | `internal/provider/tokens.go` | Estimate tokens for cost calculation |

### Phase 3: Missing Backend APIs

#### 3.1 User Management

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v0/admin/users` | GET | List users |
| `/v0/admin/users` | POST | Create user |
| `/v0/admin/users/{id}` | GET | Get user |
| `/v0/admin/users/{id}` | PUT | Update user |
| `/v0/admin/users/{id}` | DELETE | Delete user |

**File**: `internal/admin/users.go`
**Reference**: AxonHub `users/index.tsx`, auth system

#### 3.2 Role Management

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v0/admin/roles` | GET | List roles |
| `/v0/admin/roles` | POST | Create role |
| `/v0/admin/roles/{id}` | PUT | Update role |
| `/v0/admin/roles/{id}` | DELETE | Delete role |
| `/v0/admin/roles/{id}/permissions` | GET | Get role permissions |

**File**: `internal/admin/roles.go`
**Reference**: AxonHub `roles/index.tsx`

#### 3.3 Model Management

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v0/admin/models` | GET | List models with pricing |
| `/v0/admin/models` | POST | Add model mapping |
| `/v0/admin/models/{id}` | PUT | Update model config |
| `/v0/admin/models/{id}` | DELETE | Remove model |
| `/v0/admin/models/{id}/pricing` | PUT | Update pricing |

**File**: `internal/admin/models.go`
**Reference**: AxonHub `models/index.tsx`, model pricing

#### 3.4 System Management

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v0/admin/system/status` | GET | System status |
| `/v0/admin/system/initialize` | POST | First-run initialization |
| `/v0/admin/system/settings` | GET/PUT | System settings |

**File**: `internal/admin/system.go`
**Reference**: AxonHub `system/index.tsx`

#### 3.5 Error Tracking

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v0/admin/errors` | GET | List errors |
| `/v0/admin/errors/{id}` | GET | Error details |
| `/v0/admin/errors/stats` | GET | Error statistics |

**File**: `internal/admin/errors.go`
**Reference**: Plexus `registerErrorRoutes`

#### 3.6 System Logs

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v0/admin/logs` | GET | List system logs |
| `/v0/admin/logs/stream` | GET | SSE log stream |

**File**: `internal/admin/logs.go`
**Reference**: Plexus `registerSystemLogRoutes`

#### 3.7 Playground

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v0/admin/playground/chat` | POST | Chat completion via admin UI |

**File**: `internal/admin/playground.go`
**Reference**: AxonHub `playground/chat` endpoint

#### 3.8 Cooldown Management

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v0/admin/cooldowns` | GET | List provider cooldowns |
| `/v0/admin/cooldowns/{provider}` | PUT | Set cooldown |
| `/v0/admin/cooldowns/{provider}` | DELETE | Clear cooldown |

**File**: `internal/admin/cooldowns.go`
**Reference**: Plexus `CooldownManager`

#### 3.9 Prometheus Metrics

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/metrics` | GET | Prometheus scrape endpoint |

**File**: `internal/admin/prometheus.go`
**Reference**: Plexus `registerMetricsRoutes`

#### 3.10 Control Room Backend CRUD

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v0/admin/control-rooms` | GET/POST | List/create control rooms |
| `/v0/admin/control-rooms/{id}` | GET/PUT/DELETE | CRUD operations |

**File**: `internal/admin/controlrooms.go`

### Phase 4: UI Feature Parity

#### New Pages

| Page | Source Reference | File |
|------|-----------------|------|
| Playground/Chat | AxonHub `chats/index.tsx` | `web/src/app/playground/page.tsx` |
| Trace Explorer | AxonHub tracing/threads | `web/src/app/traces/page.tsx` |
| Users Management | AxonHub `users/index.tsx` | `web/src/app/users/page.tsx` |
| Roles Management | AxonHub `roles/index.tsx` | `web/src/app/roles/page.tsx` |
| Models Management | AxonHub `models/index.tsx` | `web/src/app/models/page.tsx` |
| System/Settings | AxonHub `system/index.tsx`, `settings/*` | `web/src/app/settings/page.tsx` |
| Error Tracking | Plexus `Errors.tsx` | `web/src/app/errors/page.tsx` |
| System Logs | Plexus `SystemLogs.tsx` | `web/src/app/logs/page.tsx` |
| Performance | Plexus `Performance.tsx` | `web/src/app/performance/page.tsx` |
| Quotas | Plexus `Quotas.tsx` | `web/src/app/quotas/page.tsx` |
| Debug Console | Plexus `Debug.tsx` | `web/src/app/debug/page.tsx` |

#### A2A UI Upgrade

The existing A2A page needs upgrading to show:
- JSON-RPC method testing interface
- Task list with filtering/pagination
- Push notification config management
- Agent card viewer/editor
- Conversation history (contextId grouping)

#### Query Integration

Each new page needs TanStack Query hooks in `web/src/queries/`:
- `users.ts`, `roles.ts`, `models.ts`, `system.ts`
- `errors.ts`, `logs.ts`, `performance.ts`, `quotas.ts`
- `playground.ts`, `cooldowns.ts`, `controlrooms.ts`

## 4. H Agent Team Structure

**Team Name**: `hotel-parity`
**Size**: 5 (TEAM-007 compliant)

| Agent | Type | Role | Phase Assignment |
|-------|------|------|-----------------|
| protocol-lead | golang-pro | A2A JSON-RPC, spec compliance | Phase 1 |
| backend-lead | golang-pro | Adapters, APIs, integration | Phase 2 + 3 |
| adapter-engineer | golang-pro | Provider HTTP clients, streaming | Phase 2 |
| frontend-engineer | nextjs-developer | React UI pages, queries | Phase 4 |
| qa-engineer | test-automator | Tests across all phases | All |

## 5. Testing Strategy

| Phase | Test Type | Coverage Target |
|-------|-----------|----------------|
| Phase 1 | A2A JSON-RPC method tests, agent card validation | 90% |
| Phase 2 | Provider adapter unit tests, streaming tests | 85% |
| Phase 3 | API handler tests, RBAC integration tests | 80% |
| Phase 4 | Playwright E2E tests for new pages | Key flows |
| All | `go build ./...` must pass, `go vet ./...` clean | Always |

## 6. Database Migrations

### New Tables

```sql
-- Push notification configs
CREATE TABLE a2a_push_notification_configs (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES a2a_tasks(id),
    url TEXT NOT NULL,
    token TEXT,
    auth_type TEXT,
    auth_credentials TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Control rooms (move from localStorage)
CREATE TABLE control_rooms (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    tags JSONB DEFAULT '[]',
    layout JSONB DEFAULT '{}',
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Error tracking
CREATE TABLE error_logs (
    id TEXT PRIMARY KEY,
    error_type TEXT NOT NULL,
    message TEXT NOT NULL,
    stack_trace TEXT,
    request_id TEXT,
    provider TEXT,
    model TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- System logs
CREATE TABLE system_logs (
    id TEXT PRIMARY KEY,
    level TEXT NOT NULL,
    component TEXT NOT NULL,
    message TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Model configurations
CREATE TABLE model_configs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    provider TEXT NOT NULL,
    model_id TEXT NOT NULL,
    pricing JSONB DEFAULT '{}',
    capabilities JSONB DEFAULT '[]',
    status TEXT DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Roles
CREATE TABLE roles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    permissions JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Cooldowns
CREATE TABLE provider_cooldowns (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    reason TEXT,
    cooldown_until TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Altered Tables

```sql
-- Add fields to a2a_tasks
ALTER TABLE a2a_tasks ADD COLUMN context_id TEXT;
ALTER TABLE a2a_tasks ADD COLUMN history JSONB DEFAULT '[]';
ALTER TABLE a2a_tasks ADD COLUMN state_transition_history JSONB DEFAULT '[]';
CREATE INDEX idx_a2a_tasks_context_id ON a2a_tasks(context_id);
CREATE INDEX idx_a2a_tasks_status ON a2a_tasks(status);
```

## 7. What We Are NOT Building

- GraphQL API (AxonHub has it, REST is sufficient)
- Jina/Rerank endpoints (niche provider)
- Data storage management (AxonHub-specific)
- Help center page (nice-to-have)
- Drag-and-drop dashboard builder (over-engineered)
- i18n/localization (premature)
- ACP/ANP protocols (deferred per protocol decision)

## 8. Success Criteria

1. All A2A JSON-RPC methods pass spec compliance tests
2. `/.well-known/agent.json` matches AgentCard schema
3. Real OpenAI/Anthropic/Gemini requests complete end-to-end
4. Streaming works with all three providers
5. All new UI pages render and interact with backend APIs
6. `go build ./...` and `go vet ./...` pass clean
7. Test coverage >= 80% for new code
8. All 11 new UI pages accessible from sidebar navigation

## 9. References

- [A2A Protocol Specification](https://a2a-protocol.org/latest/specification/)
- [AxonHub Source](file:///mnt/ollama/git/axonhub)
- [Plexus Source](file:///mnt/ollama/git/plexus)
- [RAD Gateway Architecture](../architecture/ARCHITECTURE_SYNTHESIS_REPORT.md)
- [Feature Matrix](../feature-matrix.md)
