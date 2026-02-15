# Brass Relay Product Build Blueprint

## Product Intent

Build a Go-native AI gateway with Plexus + AxonHub feature parity, preserving strict OpenAI/Anthropic/Gemini compatibility while layering a steampunk analog product feel in non-protocol surfaces.

## Build Principles

- Compatibility first: external API contracts remain machine-stable.
- Operator-grade controls: routing, failover, usage, quotas, tracing, and admin visibility are core, not optional.
- Theme without ambiguity: steampunk language is UX garnish, never an operational or protocol substitute.
- Phase-gated delivery: ship iteratively with explicit architecture/security/QA/SRE gate evidence.

## Target System Architecture

### Control Plane

- Config model and route registry (`internal/config`)
- Provider and model registry (`internal/provider`, `internal/models`)
- Policy and quota controls (`internal/quota` planned)
- Admin APIs (`internal/admin`)

### Data Plane

- Ingress handlers (`internal/api`)
- Auth and trace middleware (`internal/middleware`)
- Orchestration core (`internal/core`)
- Routing/failover engine (`internal/routing`)
- Provider adapters and transformers (`internal/provider`, transformer layer planned)

### Telemetry Plane

- Usage sink (`internal/usage`)
- Trace store (`internal/trace`)
- CI + security checks (`.github/workflows/ci.yml`)

## Detailed Feature Workstreams

## 1) Compatibility Surface

- Implement and harden:
  - `/v1/chat/completions`
  - `/v1/responses`
  - `/v1/models`
  - `/v1/messages`
  - `/v1/embeddings`
  - `/v1/images/generations`
  - `/v1/audio/transcriptions`
  - `/v1beta/models/*action`
- Add endpoint-level behavior fixtures for parity assertions.

## 2) Provider Orchestration

- Candidate resolution:
  - model alias resolution
  - direct routing override path
  - health/cooldown-aware filtering
- Retry/failover:
  - bounded attempts
  - retryable error classification
  - backoff strategy options
- Strategy roadmap:
  - v1 weighted + fallback
  - v1.1 trace-aware preference
  - v2 adaptive/circuit-breaker persisted state

## 3) Auth, Access, and Security

- Unified key extraction:
  - bearer
  - `x-api-key`
  - `x-goog-api-key`
  - query `key`
- Admin auth boundaries:
  - public health and bootstrap endpoints
  - protected management endpoints
- Security posture:
  - `.env` secret isolation
  - CI security scans (`govulncheck`, `gosec`)
  - guardrails checks in local and CI workflows

## 4) Usage, Quota, and Cost

- Usage records per request:
  - incoming API/model
  - selected provider/model
  - status, latency, token/cost metrics
- Quota engine plan:
  - hourly/daily/monthly windows
  - manual and scheduled checks
  - historical snapshots and reset projections

## 5) Trace and Thread Continuity

- Request-scoped IDs:
  - request ID
  - trace ID
  - optional thread semantics (next increment)
- Cross-request continuity for conversational and responses APIs.

## 6) Admin and Operations

- Admin endpoints for config snapshot, usage, traces.
- Operational docs and runbooks under `docs/operations/`.
- SLO and release controls defined for Phase 5 gate readiness.

## 7) Multi-Agent Interoperability

- A2A (agent-to-agent) as the primary interop protocol:
  - Agent Card publication and discovery
  - task lifecycle APIs (sync + streaming)
  - agent capability and auth metadata negotiation
- AG-UI as the user-facing event protocol:
  - run lifecycle streaming
  - tool call/status/state event envelopes
  - session replay support for resilient UX
- MCP as a scoped bridge for tools/resources:
  - use for context/tool exposure only
  - do not use MCP as agent orchestration control plane
- ACP and ANP posture:
  - ACP: defer (archived and folded toward A2A direction)
  - ANP: watchlist (promising, but defer until stronger implementer ecosystem)

## Dream Team Execution Model

### Team Topology

- Architecture Ring (Team 2): solution/standards ownership and ADR authority.
- Build Rings (Teams 7, 8): feature delivery and integration surfaces.
- Hardening Rings (Teams 9, 10): security and QA sign-off.
- Sustainment Rings (Teams 11, 12): SRE/release/incident operations.

### Role Assignments (Practical)

- Chief Architect: parity envelope and tradeoff decisions.
- Backend Lead: core API and orchestration implementation.
- Integration Lead: provider adapters/transformers.
- Security Lead: auth, secrets, scanner policy, vulnerability triage.
- QA Lead: behavior fixtures, contract tests, regression coverage.
- SRE Lead: SLOs, alerts, release readiness, incident process.
- Release Manager: changelog, tags, rollout/rollback orchestration.
- Docs Lead: compatibility docs, operator docs, governance docs.

### Decision Cadence

- Daily: build sync (15 min) on blockers and parity deltas.
- Twice weekly: architecture/review board for cross-cutting changes.
- Weekly: release readiness review against phase-gate checklists.
- Gate rule: no phase progression without documented evidence artifacts.

## Delivery Roadmap

### Milestone A - Contract Core (Week 1)

- Complete compatibility endpoints with stable schemas.
- Complete auth normalization and protected-route policy.
- Publish compatibility behavior table and fixture plan.

### Milestone B - Orchestration Core (Week 2)

- Weighted candidate routing + retries.
- Provider adapter abstraction with at least OpenAI + Anthropic + Gemini paths.
- Usage and trace capture with admin read APIs.

### Milestone C - Hardening and Fidelity (Week 3)

- Expand tests for failure-path parity.
- Add quota and policy middleware baseline.
- Improve streaming and responses fidelity coverage.

### Milestone D - Operational Launch Readiness (Week 4)

- Full docs pass (operator + contributor + security).
- Release and incident runbooks validated.
- Public release checklist and branch governance verified.

## Risk Register

- Transformer drift risk: provider contract mismatch under edge fields.
  - Mitigation: fixture-based contract tests per provider path.
- Retry storm risk: cascading retries on upstream instability.
  - Mitigation: bounded attempts, backoff, and circuit-breaker progression.
- Secret leakage risk in a public repo.
  - Mitigation: `.env` isolation, scanner enforcement, pre-push guardrails.
- Theme-induced ambiguity risk.
  - Mitigation: operational/log labels remain plain technical identifiers.

## Definition of Done (Product-Level)

- Parity-critical routes validated against source behavior expectations.
- Security/QA/SRE gates each have explicit artifacts and passing checks.
- Release checklist and support handoff complete.
- Steampunk layer shipped in docs/admin UX without protocol side effects.
