# Go Implementation Plan (Brass Relay / rad-gateway)

## Module Boundaries

- `cmd/rad-gateway`
  - Process bootstrap, config load, HTTP server wiring.
- `internal/config`
  - Env/config structs, defaults, validation.
- `internal/models`
  - Shared API DTOs for OpenAI-compatible request/response contracts.
- `internal/provider`
  - Adapter interfaces, provider registry, mock provider, execution metadata.
- `internal/routing`
  - Candidate selection, weighted ordering, retry/failover policy.
- `internal/core`
  - Gateway service orchestration: auth context, adapter invocation, usage emit.
- `internal/middleware`
  - API key extraction/validation, request id, trace id, logging hooks.
- `internal/usage`
  - Usage sink interface + in-memory implementation + admin query projection.
- `internal/trace`
  - Trace event model and in-memory tracer.
- `internal/api`
  - Public compatibility handlers (`/v1/*`, `/v1beta/*`) and health route.
- `internal/admin`
  - Management API handlers (`/v0/management/config`, `/v0/management/usage`).

## Request Lifecycle (v1)

1. Middleware extracts API key and trace context.
2. API handler validates payload shape and maps to internal request model.
3. Core service resolves route candidates for requested model.
4. Router executes provider attempts with failover budget.
5. Provider adapter returns normalized response + usage metadata.
6. Usage sink + tracer record outcome.
7. Handler returns provider-compatible response envelope.

## Delivery Phases

### Phase 1 (MVP skeleton)

- Server, config, middleware, health endpoint.
- OpenAI-compatible routes with stub response path.
- In-memory API key validation.

### Phase 2 (Parity core)

- Provider abstraction and registry.
- Weighted routing + retry/failover.
- Usage + trace capture, management read endpoints.
- Gemini and Anthropic compatibility entrypoints.

### Phase 3 (Parity+)

- Persistent storage adapters.
- Quota windows/policies.
- OAuth session workflows.
- Advanced balancing (circuit breaker, adaptive scoring).

## MVP Out of Scope

- Full GraphQL admin plane.
- Rich RBAC scopes/project multi-tenancy.
- Full Responses API edge-case parity.
- Production-grade distributed tracing backend.
