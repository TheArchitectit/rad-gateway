# Review Teams and Gates

## Team Initialization

- Team topology initialized via `scripts/team_manager.py` for project `rad-gateway`.
- Team state file: `.teams/rad-gateway.json`.
- Core review teams staffed for this repo:
  - Team 2 (Architecture): Oracle-Agent, Explore-Agent, Sisyphus, Librarian-Agent
  - Team 9 (Security): Security-Agent, Audit-Agent, RedTeam-Agent, Platform-Agent
  - Team 10 (QA): QA-Lead, Test-Agent, Perf-Agent, UAT-Agent

## Phase Gates for This Repository

1. Architecture Gate (Phase 1 -> 2)
   - Approved module boundaries
   - Compatibility contracts validated
   - Guardrails integration plan accepted
2. Security Gate (Phase 4)
   - Secret-handling policy validated
   - No plaintext credentials in source/docs
   - Push and git-operation guardrails active
3. QA Gate (Phase 4)
   - Build/test checks green
   - Endpoint smoke coverage for compatibility routes

## Review Checklist (Per PR)

- Architecture
  - API compatibility not broken (`/v1/*`, `/v1beta/*`, `/v0/management/*`)
  - Routing and failover behavior unchanged unless explicitly planned
- Security
  - `.env` never committed; only `.env.example` tracked
  - No secrets in logs, docs, fixtures, or defaults
  - Unsafe git operations (force-push, branch rewrites) blocked by policy
- Quality
  - `go build ./...` passes
  - `go test ./...` passes
  - Docs updated for behavior/config changes
