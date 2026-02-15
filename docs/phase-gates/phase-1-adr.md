# Phase 1 ADR Pack

## ADR-001: Compatibility-First Gateway Surface

- Status: accepted
- Decision: keep external API contract aligned with OpenAI/Anthropic/Gemini-compatible endpoints while preserving internal module boundaries.
- Why: migration friction stays low and existing SDK clients continue to work.
- Evidence: `docs/feature-matrix.md`, `internal/api/handlers.go`, `docs/implementation-plan.md`.

## ADR-002: Adapter + Router Separation

- Status: accepted
- Decision: isolate provider-specific behavior behind adapters and keep routing/failover in a separate orchestration layer.
- Why: easier provider expansion and deterministic failover policy control.
- Evidence: `internal/provider/provider.go`, `internal/provider/mock.go`, `internal/routing/router.go`, `docs/implementation-plan.md`.

## ADR-003: Public-Day-One Secret Discipline

- Status: accepted
- Decision: no runtime secret defaults in code, all credentials from environment, `.env` excluded from git.
- Why: immediate reduction of secret-leak risk in a public repository.
- Evidence: `internal/config/config.go`, `.gitignore`, `.env.example`, `SECURITY.md`.
