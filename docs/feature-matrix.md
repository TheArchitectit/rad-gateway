# Feature Parity Matrix (Plexus + AxonHub -> RAD Gateway)

## Capability Matrix

| Capability | Plexus Evidence | AxonHub Evidence | Required in RAD v1 | Notes / Risk |
|---|---|---|---|---|
| OpenAI Chat Completions | `docs/API.md` (`POST /v1/chat/completions`), `packages/backend/src/routes/inference/index.ts` | `internal/server/routes.go` (`/v1/chat/completions`) | Yes | Core compatibility contract |
| OpenAI Responses | `packages/backend/src/routes/inference/responses.ts` | `internal/server/routes.go` (`/v1/responses`), README marks partial | Yes (subset) | Implement practical subset first |
| OpenAI Models list | `packages/backend/src/routes/inference/models.ts` | `internal/server/routes.go` (`/v1/models`) | Yes | Must normalize model inventory |
| Anthropic Messages | `docs/API.md` (`POST /v1/messages`) | `internal/server/routes.go` (`/v1/messages`, `/anthropic/v1/messages`) | Yes | Dual-path compatibility |
| Gemini Compatibility | `docs/API.md` (`/v1beta/models/{model}:{action}`) | `internal/server/routes.go` (`/gemini/:version/models/*action`, `/v1beta/models/*action`) | Yes | Preserve gemini key auth mode |
| Embeddings | `docs/API.md` (`POST /v1/embeddings`) | `internal/server/routes.go` (`/v1/embeddings`) | Yes | Pass-through path in v1 |
| Image Generation / Edits | `docs/API.md` (`/v1/images/generations`, `/v1/images/edits`) | README marks image partial | v1: generations only | Edits in v1.1 |
| Audio Transcriptions / Speech | `docs/API.md` (`/v1/audio/transcriptions`, `/v1/audio/speech`) | README lacks first-class speech API mention | v1: transcriptions only | Speech in v1.1 |
| Provider abstraction/transformers | `packages/backend/src/services/transformer-factory.ts` | `llm/transformer/*`, `internal/server/orchestrator/*` | Yes | Interface-first design required |
| Routing/load balancing/failover | `packages/backend/src/services/router.ts`, `dispatcher.ts` | `orchestrator/load_balancer.go`, `orchestrator/retry.go` | Yes | Must support retries + candidate selection |
| API key auth | `routes/inference/index.ts` bearer + x-api-key + x-goog-api-key normalization | `middleware/auth.go`, route groups with API key auth | Yes | Key attribution optional in v1 |
| OAuth management | `docs/API.md` OAuth session endpoints | Admin OAuth endpoints for codex/claude code | v1: basic token store hooks | Full interactive OAuth deferred |
| Usage tracking/cost stats | `services/usage-storage.ts`, `/v0/management/usage` | `biz/usage`, `biz/cost_calc`, README cost tracking | Yes | Initial in-memory + log sink acceptable |
| Quotas | `/v0/quotas*`, `services/quota/*` | `biz/quota.go` | v1: request/token budget middleware | Advanced checker windows deferred |
| Tracing/threads | usage/perf records in management APIs | README tracing/thread model, middleware trace/thread | Yes | Request ID + Trace ID mandatory |
| Admin APIs | `/v0/management/*` | `/admin/*`, GraphQL + OpenAPI group | v1: lightweight REST admin | Avoid GraphQL in MVP |
| A2A agent interoperability | N/A (not first-class) | N/A (not first-class) | Yes (v1.1) | Add Agent Card, task lifecycle, SSE task updates |
| AG-UI event protocol | N/A | N/A | Yes (v1.1) | Frontend-backend event stream contract for agent UX |
| MCP tool/context bridge | N/A | N/A | Yes (targeted) | Scope to tool/resource access, not agent orchestration |
| ACP compatibility | N/A | N/A | No (defer) | ACP repo is archived and migrated toward A2A direction |
| ANP compatibility | N/A | N/A | No (defer) | Track maturity; evaluate once production SDKs stabilize |

## v1 Scope (Build Now)

- Compatibility endpoints: `/v1/chat/completions`, `/v1/responses` (subset), `/v1/models`, `/v1/messages`, `/v1/embeddings`, `/v1/images/generations`, `/v1/audio/transcriptions`, `/v1beta/models/*action`.
- Auth: Bearer + `x-api-key` + `x-goog-api-key` extraction into one API-key middleware.
- Provider abstraction: adapter interface with request/response transform hooks and execute method.
- Routing policy: weighted candidate selection + failover retries + per-request attempt budget.
- Observability: request id, trace id propagation, timing, attempt history.
- Usage: capture token/cost fields from adapter metadata, store via pluggable usage sink.
- Admin: `/v0/management/config` (read-only snapshot), `/v0/management/usage` (recent records), `/health`.

## Deferred (Post-v1)

- Full OAuth browser/session choreography.
- Rich policy engine (RBAC scopes, project tenancy, profile switching).
- Persistent relational storage and migrations.
- Full Responses API fidelity and image edit/speech APIs.
- Circuit breaker state persistence and adaptive load-balancing telemetry loops.
- ACP and ANP protocol support (watchlist until stronger implementation maturity).
