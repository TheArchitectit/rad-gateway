# Reverse-Engineering Report: Plexus + AxonHub

## Goal

Define the parity contract for `rad-gateway` by extracting implementation-critical behavior from Plexus and AxonHub, then mapping that behavior into a Go-native product plan.

## Source Evidence

### Plexus (local)

- Route/auth normalization and protected route registration:
  - `plexus/packages/backend/src/routes/inference/index.ts`
- Routing decision logic (direct routing, alias resolution, selector, cooldown filtering, API-type matching):
  - `plexus/packages/backend/src/services/router.ts`
- Dispatcher behavior (target API selection, pass-through optimization, provider request execution, error cooldown parsing, streaming/non-streaming handlers, modality-specific dispatch):
  - `plexus/packages/backend/src/services/dispatcher.ts`
- Management API composition:
  - `plexus/packages/backend/src/routes/management.ts`
- Usage management endpoints and SSE event stream:
  - `plexus/packages/backend/src/routes/management/usage.ts`
- Quota endpoints and scheduler-backed quota history/check-now APIs:
  - `plexus/packages/backend/src/routes/management/quotas.ts`
  - `plexus/packages/backend/src/services/quota/quota-scheduler.ts`

### AxonHub (local)

- Route groups and middleware boundaries (public/admin/openapi/api/gemini alias):
  - `axonhub/internal/server/routes.go`
- Orchestrator and load balancer strategy model:
  - `axonhub/internal/server/orchestrator/orchestrator.go`
  - `axonhub/internal/server/orchestrator/load_balancer.go`
  - `axonhub/internal/server/orchestrator/retry.go`
  - `axonhub/internal/server/orchestrator/model_circuit_breaker.go`
- Trace/thread middleware:
  - `axonhub/internal/server/middleware/trace.go`
  - `axonhub/internal/server/middleware/thread.go`
- Persistence model (Ent schemas):
  - `axonhub/internal/ent/schema/request.go`
  - `axonhub/internal/ent/schema/request_execution.go`
  - `axonhub/internal/ent/schema/trace.go`
  - `axonhub/internal/ent/schema/thread.go`
  - `axonhub/internal/ent/schema/channel.go`
  - `axonhub/internal/ent/schema/model.go`

Note: paths above are repository-relative and assume local clones of `plexus` and `axonhub` adjacent to this repository.

## Parity-Critical Contracts

1. Compatibility routes must remain first-class and stable:
   - OpenAI-style (`/v1/chat/completions`, `/v1/responses`, `/v1/models`, `/v1/embeddings`)
   - Anthropic-style (`/v1/messages`, `/anthropic/v1/messages`)
   - Gemini-style (`/gemini/:version/models/*action`, `/v1beta/models/*action`)
2. Auth key normalization must support all three inbound styles:
   - `Authorization: Bearer ...`
   - `x-api-key`
   - `x-goog-api-key` and `?key=` for Gemini compatibility
3. Router semantics must preserve:
   - alias + additional_alias lookup
   - provider/model direct routing escape hatch
   - health/cooldown-aware filtering
   - selector strategy (weight/random/priority) and API-type compatibility narrowing
4. Dispatcher/orchestrator semantics must preserve:
   - incoming API type to target API type selection
   - transformation boundaries
   - pass-through path when inbound/outbound API format matches
   - retry/failover behavior and cooldown/circuit-breaker signals
5. Observability and accounting must preserve:
   - request-level usage metadata and cost context
   - trace/thread propagation semantics
   - admin-plane visibility for usage/quota/perf/debug operations

## Design Decisions for `rad-gateway`

### Copy as-is (behavioral parity)

- Compatibility route topology and API-key normalization semantics.
- Retry/failover with channel candidate ordering and bounded attempt budget.
- Trace/thread-aware request context propagation.
- Management-plane read APIs for usage/config and quota visibility.

### Intentionally adapt

- Keep REST admin surface for v1 (defer GraphQL complexity).
- Start with in-memory usage/trace sinks + pluggable storage interfaces.
- Use Go-native modular boundaries already in `internal/*` and add persistent backends later.

## Steampunk Layer Constraints

- Allowed: naming and presentation in docs/admin UX (`boiler`, `track`, `pulse`, `ledger`).
- Forbidden: changing external API fields, status semantics, or protocol paths used by SDKs.
- Rule: every themed term must map to a plain technical term in logs and runbooks.

## Gaps to Close for Full Parity

- Full Responses API fidelity and edge cases.
- Image edit + speech parity across providers.
- Rich OAuth/session lifecycle and token refresh choreography.
- Persistent quota windows and historical analytics.
- More advanced adaptive load-balancing and circuit-breaker state persistence.

## Multi-Agent Protocol Decision (Post-Parity)

1. A2A: adopt for agent-to-agent communication and task lifecycle.
2. AG-UI: adopt for frontend/backend event protocol and interactive agent UX.
3. MCP: adopt selectively for tool/resource context integration, not for orchestration.
4. ACP: defer implementation (project archived and migration path points to A2A).
5. ANP: monitor and reassess later after SDK/runtime ecosystem matures further.

Rationale: parity sources (Plexus/AxonHub) do not provide first-class protocol precedents for this layer; this is a deliberate greenfield extension that should sit on top of parity-complete gateway foundations.
