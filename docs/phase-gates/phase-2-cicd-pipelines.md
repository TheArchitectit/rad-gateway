# Phase 2 Deliverable: CI/CD Pipelines

## Baseline CI Pipeline

- Workflow file: `.github/workflows/ci.yml`
- Trigger conditions:
  - pull requests
  - pushes to `main`
- Validation stages:
  - Go unit/package test run: `go test ./...`
  - Go build verification: `go build ./...`
  - Guardrails failure listing: `python3 scripts/log_failure.py --list`
  - Guardrails regression scan: `python3 scripts/regression_check.py --all --verbose`

## Branch Policy Alignment

- `main` branch protection requires review before merge.
- CI pipeline acts as quality gate for merge readiness.

## Next Upgrade Steps

- Add staged-change regression check in pre-merge jobs.
- Add release workflow for tagged builds.
- Add dependency and container image scanning.
