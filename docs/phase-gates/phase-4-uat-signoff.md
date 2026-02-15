# Phase 4 Deliverable: UAT Sign-off

## UAT Scope (Bootstrap)

- Public compatibility endpoints:
  - `/v1/chat/completions`
  - `/v1/responses`
  - `/v1/messages`
  - `/v1/embeddings`
  - `/v1/images/generations`
  - `/v1/audio/transcriptions`
  - `/v1/models`
  - `/v1beta/models/{model}:{action}`
- Management endpoints:
  - `/v0/management/config`
  - `/v0/management/usage`
  - `/v0/management/traces`

## Acceptance Criteria

- Endpoints respond with expected HTTP semantics for supported methods.
- Auth middleware enforces API key requirement for protected routes.
- Health and management routes remain available for operational checks.
- Build and test suite passes in local verification and CI workflow.

## UAT Result

- Bootstrap UAT accepted for Phase 4 progression.
- Any advanced parity requirements (streaming semantics, deeper provider edge cases) are deferred to post-gate milestones.
