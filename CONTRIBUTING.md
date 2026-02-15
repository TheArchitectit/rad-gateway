# Contributing

## Before You Start

- Read `.guardrails/pre-work-check.md`.
- Keep changes scoped and testable.
- Never commit secrets.

## Secret Policy

- Use `.env` for local secrets.
- Keep `.env` out of git.
- Update `.env.example` when adding new required env vars.

## Development Checks

- `go test ./...`
- `go build ./...`

## Guardrails Workflow

- Initialize team structure (already done for this repo):
  - `python3 scripts/team_manager.py --project rad-gateway status`
- Use architecture/security/QA review checklist in `docs/review-teams.md`.
