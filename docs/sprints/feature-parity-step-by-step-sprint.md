# RAD Gateway Feature Parity Sprint (Step-by-Step)

## 0) Goal

Ship production-ready parity for:

1. Provider/API Key/Project management (full CRUD, no mock data)
2. A2A task lifecycle (real execution, not placeholder artifacts)
3. OAuth provider flows (real token exchange/refresh/validation)
4. MCP proxy and audio APIs (real provider-backed behavior)
5. Reporting and analytics (real data, filters, exports)

This plan is implementation-first and ordered by risk and dependency.

---

## 1) Current Baseline (from repo state)

- Web app has create forms, but list/detail/edit/bulk flows are incomplete or mock-backed.
- A2A task endpoints exist, but processing is placeholder echo behavior.
- OAuth endpoints exist, but session/token flow is in-memory simulation.
- MCP endpoint exists, but is echo/accept only.
- Reports endpoints exist, but return synthetic values.
- Several admin handlers still rely on mock generation.

---

## 2) Sprint Structure

- Total duration: **6 weeks (30 working days)**
- Cadence: daily execution, weekly integration checkpoint
- Hard quality gates each day:
  - `go test ./...`
  - `go build ./cmd/rad-gateway`
  - `cd web && npm run build`

---

## 3) Week-by-Week Plan

## Week 1 - P0 Foundation: Remove mock-backed admin surfaces

### Day 1 - Provider API truth and route alignment

1. Replace mock response paths in `internal/admin/providers.go` with DB-backed reads/writes.
2. Add missing provider test endpoint contract used by UI (`/v0/admin/providers/{id}/test`) or change UI query hook to existing backend route.
3. Add integration tests for create/update/delete/test provider.
4. Validate with:
   - `go test ./internal/admin ./tests/integration -run Provider`

**Deliverable:** Provider APIs are real and API/UI contracts match.

### Day 2 - API key security correctness

1. Replace placeholder hashing in `internal/admin/apikeys.go` with secure hashing.
2. Ensure created key secret is one-time-return and persisted hashed only.
3. Add revoke/rotate tests with regression checks.
4. Validate with provider + key end-to-end tests.

**Deliverable:** API key lifecycle is secure and production-safe.

### Day 3 - Project/workspace persistence

1. Replace mock workspace list/create paths in `internal/admin/projects.go`.
2. Connect workspace CRUD to real DB repository methods.
3. Add query filters (status/search/pagination) against DB.
4. Add integration tests for workspace CRUD + search.

**Deliverable:** Project/workspace endpoints are real and queryable.

### Day 4 - Usage/cost source of truth

1. Remove synthetic usage generation in `internal/admin/usage.go`.
2. Wire usage list/summary/trend to persisted usage records.
3. Replace synthetic cost summary paths in `internal/admin/costs.go` with calculated DB aggregates.
4. Validate with seed data + deterministic assertions.

**Deliverable:** Usage and cost endpoints reflect real records.

### Day 5 - Week 1 integration checkpoint

1. Run full backend suite.
2. Fix all regressions.
3. Publish a verification note in docs/sprints/checkpoints/week-1.md.

**Exit criteria:** No mock-driven admin behavior in providers, keys, projects, usage, costs.

---

## Week 2 - P0 Frontend parity: Full management UX

### Day 6 - Providers list -> real query data

1. Replace mock array in `web/src/app/providers/page.tsx` with query hooks.
2. Add row actions: detail, edit, delete.
3. Add health/circuit action controls where backend supports it.
4. Verify build and action flows.

### Day 7 - API keys list + bulk operations

1. Replace mock array in `web/src/app/api-keys/page.tsx` with query hooks.
2. Add bulk select + bulk revoke/delete action bar.
3. Add key rotate/revoke row actions.
4. Add optimistic updates/invalidation.

### Day 8 - Projects list + detail

1. Replace mock array in `web/src/app/projects/page.tsx` with store/query backed data.
2. Add project detail page (stats, members, linked keys/providers).
3. Add project edit screen (static-export-safe route strategy if needed).
4. Validate no dead routes.

### Day 9 - Edit/detail pages parity

1. Implement providers detail/edit pages with current deployment constraints.
2. Implement api-keys edit/detail pages.
3. Implement projects edit/detail/member management pages.
4. Remove any stale links to non-existent routes.

### Day 10 - UX hardening + tests

1. Add loading, empty, error states for all management pages.
2. Add component tests for forms + tables.
3. Add E2E smoke for create/edit/delete flows.
4. Weekly checkpoint doc with before/after parity matrix.

**Exit criteria:** Web admin supports full CRUD and bulk operations, no mock dataset rendering.

---

## Week 3 - P1 A2A productionization

### Day 11 - Task model/state guarantees

1. Finalize state transition policy in `internal/a2a/task.go`.
2. Add transition validation on update paths.
3. Add tests for valid/invalid transitions.

### Day 12 - Real task execution abstraction

1. Introduce executor interface for task processing.
2. Replace placeholder echo artifact flow in `internal/a2a/task_handlers.go`.
3. Support timeout/cancel propagation.

### Day 13 - SSE event taxonomy

1. Emit deterministic SSE events (submitted, working, artifact, completed/failed).
2. Add stream closure guarantees.
3. Add integration tests for `sendSubscribe` event order.

### Day 14 - Persistence + retrieval semantics

1. Ensure task updates are atomic and consistent.
2. Add pagination/filtering for task listing (if required by UI).
3. Verify cancel semantics for non-terminal tasks.

### Day 15 - A2A compliance checkpoint

1. Validate required endpoints and response contracts.
2. Validate `/.well-known/agent.json` contract.
3. Add a conformance checklist doc.

**Exit criteria:** A2A tasks are truly executable and observable, not placeholder responses.

---

## Week 4 - P2 OAuth production path

### Day 16 - Provider implementations

1. Replace static/in-memory OAuth manager behavior with provider-specific implementations.
2. Add secure state/nonce handling and callback verification.

### Day 17 - Token exchange and refresh

1. Implement real code->token exchange.
2. Persist token metadata securely.
3. Add refresh + validation using provider APIs.

### Day 18 - Auth boundary and route security

1. Move OAuth admin operations behind correct auth boundary.
2. Keep callback endpoint public but validated.
3. Add tests for unauthorized/forbidden scenarios.

### Day 19 - Provider binding to gateway routing

1. Bind OAuth credentials to provider records.
2. Ensure provider adapter reads OAuth token source correctly.
3. Add route-level integration tests.

### Day 20 - OAuth UI completion

1. Replace basic page with status/connect/disconnect/reconnect UX.
2. Add error reasons and refresh status in UI.
3. Add weekly checkpoint and parity score.

**Exit criteria:** OAuth flow is real, secure, persisted, and integrated with provider runtime.

---

## Week 5 - P3 MCP + Audio real behavior

### Day 21 - MCP execution layer

1. Replace echo behavior in `internal/mcp/handler.go` with real proxy dispatch.
2. Add request validation, timeout, and error mapping.

### Day 22 - MCP session and tool handling

1. Support session continuity and metadata passthrough.
2. Add integration tests for valid/invalid tool calls.

### Day 23 - Audio STT/TTS provider wiring

1. Replace placeholder audio flows in `internal/api/handlers.go` for transcriptions/speech.
2. Add provider-specific request mapping.

### Day 24 - Streaming path cleanup

1. Replace mock stream in `internal/api/streaming.go` with provider-backed stream adapters.
2. Add stream cancellation and error propagation tests.

### Day 25 - MCP/Audio UI completion

1. Upgrade `web/src/app/mcp/page.tsx` and related controls to real response rendering.
2. Add audio UI for upload/transcription/playback where needed.
3. Weekly checkpoint.

**Exit criteria:** MCP and audio endpoints are real and validated end-to-end.

---

## Week 6 - P4 Reporting and release hardening

### Day 26 - Replace synthetic reporting

1. Replace fixed values in `internal/admin/reports.go` with query-backed metrics.
2. Add date range + provider/model/key filters.

### Day 27 - Performance metrics integrity

1. Compute TTFT/TPS/latency/error rate from recorded traces/usage.
2. Validate metric math with test fixtures.

### Day 28 - Export pipeline

1. Implement real report export generation and retrieval.
2. Add expiration and authorization rules.

### Day 29 - Final regression sweep

1. Full backend + web build/test.
2. Resolve all P0/P1 defects.
3. Freeze release candidate branch.

### Day 30 - Release & verification

1. Deploy backend and web containers.
2. Run post-deploy smoke suite:
   - A2A send/sendSubscribe/get/cancel
   - OAuth start/callback/refresh/validate
   - MCP tool invocation
   - Reports usage/performance/export
3. Publish release notes and operational runbook updates.

**Exit criteria:** Feature parity complete, no synthetic behavior in critical paths.

---

## 4) Daily Definition of Done

Every day is only complete when:

1. Code compiles/builds
2. Relevant tests pass
3. Endpoint contracts are verified
4. UI path (if touched) is manually smoke-tested
5. Sprint log updated with what changed and what remains

---

## 5) Risk Register (active)

1. Static-export constraints in Next.js can block route strategy for edit/detail pages.
2. OAuth provider variance can cause unexpected callback/exchange differences.
3. A2A execution complexity may require queue/worker split for stable throughput.
4. Reporting metric correctness depends on consistency of usage/trace ingestion.

Mitigation: keep integration tests expanding each week and enforce checkpoint gate before moving to next week.

---

## 6) Execution Order (strict)

1. Remove mock-backed core admin paths (Week 1)
2. Complete CRUD/bulk UX parity (Week 2)
3. Productionize A2A execution (Week 3)
4. Harden OAuth and auth boundaries (Week 4)
5. Replace MCP/audio placeholders (Week 5)
6. Finish reporting and release hardening (Week 6)

No phase should be skipped; each phase depends on real behavior from the previous one.
