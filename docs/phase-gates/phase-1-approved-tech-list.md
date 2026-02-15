# Phase 1 Approved Tech List

## Core Runtime

- Go 1.22 (`go.mod`)
- Standard library HTTP stack (`net/http`) for gateway bootstrap

## Guardrails and Governance

- Guardrails policy docs and team rules
  - `.guardrails/pre-work-check.md`
  - `.guardrails/team-layout-rules.json`
- Team orchestration CLI
  - `scripts/team_manager.py`

## Repo and Release Tooling

- GitHub CLI (`gh`) for repository governance and policy setup
- Git branch protection + secret scanning on `TheArchitectit/rad-gateway`

## Security and Config

- `.env` runtime local secret file (not tracked)
- `.env.example` tracked placeholders only
- Security policy document: `SECURITY.md`
