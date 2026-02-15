# Product Theme: Steampunk Analog Interface Layer

## Product Name

- Primary name: **Brass Relay**
- Service codename: `rad-gateway`
- Purpose: OpenAI-compatible multi-provider AI relay with operator-grade controls.

## Naming Conventions

- Control-plane naming uses mechanical metaphors:
  - Provider endpoint = `Boiler`
  - Route policy = `Track`
  - Retry attempt = `Pulse`
  - API key profile = `Badge`
  - Usage ledger row = `LedgerEntry`
- API protocol naming stays standard and explicit:
  - Keep `/v1/chat/completions`, `/v1/responses`, `/v1/messages`, etc.
  - Keep OpenAI/Anthropic/Gemini request/response field names unchanged.

## UX Tone

- Visual language: brass + parchment + gauge motifs in docs/admin UI.
- Copy style: concise, industrial metaphors for non-critical labels only.
- Status language mapping:
  - healthy -> `steady pressure`
  - degraded -> `pressure drop`
  - failing -> `line rupture`

## Hard Boundaries (Theme Allowed vs Forbidden)

- Allowed:
  - Admin dashboard labels, icons, docs prose, sample config aliases.
  - Optional metadata fields in internal-only APIs.
- Forbidden:
  - Breaking external API compatibility contracts.
  - Replacing standard HTTP status/error semantics.
  - Renaming required JSON fields used by OpenAI/Anthropic/Gemini SDKs.
  - Obscuring operational metrics (latency, token counts, cost, status codes).

## Operational Clarity Rules

- Every themed label must have a plain-technical counterpart in logs and docs.
- Logs and tracing use technical identifiers first, themed aliases second.
- Alerting and SLO dashboards never use metaphor-only names.

## Example Mapping

| Technical Term | Themed Alias | Where Used |
|---|---|---|
| channel | boiler | admin UI only |
| route policy | track | admin UI + docs |
| request attempt | pulse | trace detail UI |
| usage record | ledger entry | usage pages/docs |
| health check | pressure check | docs prose only |
