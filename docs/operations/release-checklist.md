# Release Checklist

## Pre-Release

- [ ] Branch protection active on `main`
- [ ] Secret scanning and push protection enabled
- [ ] `go test ./...` passes
- [ ] `go build ./...` passes
- [ ] Guardrails checks clean:
  - `python3 scripts/log_failure.py --list`
  - `python3 scripts/regression_check.py --all --verbose`
- [ ] Phase gate docs updated for current release scope

## Release Execution

- [ ] Merge approved PR(s) to `main`
- [ ] Create release tag and notes
- [ ] Validate health endpoint on target runtime

## Rollback Readiness

- [ ] Previous known-good commit SHA recorded
- [ ] Revert procedure documented in release notes
- [ ] Incident owner and escalation contacts assigned
