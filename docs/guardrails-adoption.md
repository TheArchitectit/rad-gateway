# Guardrails MCP Adoption Plan (RAD Gateway)

## Scope Reviewed

- Template root guidance and tool inventory: `agent-guardrails-template/README.md`
- MCP server runtime and deployment model: `agent-guardrails-template/mcp-server/README.md`
- MCP tool schemas and handlers: `agent-guardrails-template/mcp-server/internal/mcp/server.go`
- Extended safety tools (three-strikes, halt, feature creep): `agent-guardrails-template/mcp-server/internal/mcp/tools_extended.go`
- Team management handlers: `agent-guardrails-template/mcp-server/internal/mcp/team_tool_handlers.go`
- Team layout policy: `agent-guardrails-template/.guardrails/team-layout-rules.json`
- Team CLI backend: `agent-guardrails-template/scripts/team_manager.py`
- Team-tool gaps and risks: `agent-guardrails-template/docs/GAP_ANALYSIS_TEAM_REPORT.md`

## Guardrails MCP Server Model

- Runtime split:
  - MCP protocol endpoint (SSE + JSON-RPC): `/mcp/v1/sse`, `/mcp/v1/message`
  - Web/API endpoint for docs/rules/projects/failures and health/metrics
- Storage and cache:
  - PostgreSQL for persistent rules/sessions/failures/audit artifacts
  - Redis for cache/rate-limit support
- Enforcement style:
  - Session-gated validations (`guardrail_init_session` first)
  - Preflight checks before bash/edit/git/push/commit operations
  - Escalation controls: three-strikes, uncertainty checks, halt-event tracking
  - Regression checks against historical failure patterns

## Fit for RAD Gateway

- Strong alignment with public-day-one needs:
  - command safety (`guardrail_validate_bash`)
  - file edit and secret checks (`guardrail_validate_file_edit`)
  - git safety (`guardrail_validate_git_operation`, `guardrail_validate_push`)
  - release hygiene (`guardrail_validate_commit`, pre-work check)
- Team orchestration tools exist but are less mature than core guardrail tools.

## Adopt Now (MVP)

1. Add baseline local guardrail assets to repo:
   - `.guardrails/pre-work-check.md`
   - `.guardrails/team-layout-rules.json`
   - `scripts/team_manager.py`
2. Configure OpenCode to use guardrails MCP server (remote SSE endpoint + bearer key).
3. Use mandatory runtime flow for all non-trivial tasks:
   - init session -> pre-work check -> validate edit/bash/git -> regression check.
4. Keep secrets in env files only; never in docs/source/config defaults.

## Defer / Watch Items

- Team tools have known gaps (authorization, race conditions, limited tests):
  - `agent-guardrails-template/docs/GAP_ANALYSIS_TEAM_REPORT.md`
  - Use team tools for planning/review coordination first, not as sole governance control.
- Keep branch protection and secret scanning in GitHub as an independent backstop.

## RAD Gateway Integration Points

- Secret ingress and API key handling:
  - `internal/config/config.go`
  - `internal/middleware/middleware.go`
- Public and admin API surfaces to protect with review policy:
  - `internal/api/handlers.go`
  - `internal/admin/handlers.go`
- Bootstrap and environment entrypoint:
  - `cmd/rad-gateway/main.go`

## Execution Checklist

- [x] Pull baseline guardrail policy and team files into repository
- [ ] Add `.env.example` and `.gitignore` with strict secret exclusions
- [ ] Initialize team layout for this project using `scripts/team_manager.py`
- [ ] Add public security docs (contributing + secret policy)
- [ ] Publish to new public GitHub repo with secret scanning defaults enabled
