# Next Milestones

## Milestone 1: Real Provider Adapters

- Add OpenAI-compatible outbound adapter with configurable base URL and model mapping.
- Add Anthropic and Gemini adapters with transform boundaries.
- Introduce adapter-level timeout and retry budget overrides.

## Milestone 2: Storage + Policy

- Replace in-memory usage/trace stores with persistent backing (PostgreSQL + optional object storage).
- Add quota policy middleware (request/day, token/day, monthly cost).
- Add project/key profile model and per-profile routing constraints.

## Milestone 3: Response Fidelity

- Complete OpenAI Responses API parity for major content block variants.
- Add streaming paths (SSE) for chat/responses and TTFT metrics.
- Add image edit and speech endpoints.

## Milestone 4: Operations

- Add structured logs and OpenTelemetry traces.
- Add metrics endpoint for request rates, latencies, retries, failover count.
- Add health probes for provider readiness and route integrity.

## Milestone 5: Admin and Theme Layer

- Add management auth + RBAC roles.
- Add steampunk-themed admin UI labels while preserving technical metric naming.
- Add API docs site with compatibility examples for OpenAI, Anthropic, Gemini clients.

## Milestone 6: Agent Interop Layer

- Add A2A discovery and task endpoints (`/.well-known/agent.json`, `/a2a/tasks/*`).
- Add AG-UI streaming surfaces for lifecycle/tool/state events.
- Add AG-UI stream endpoint (`GET /v1/agents/{agentId}/stream`) for client subscriptions.
- Add targeted MCP bridge for tool/resource connections with strict auth boundaries.
- Add protocol conformance fixtures and regression checks for A2A and AG-UI flows.
- Keep ACP/ANP as tracked watchlist items; no production dependency in this milestone.
