# Incident Runbook

## Severity Levels

- SEV-1: Full service outage or critical security incident
- SEV-2: Major degradation, elevated error rate, or sustained latency spike
- SEV-3: Partial feature degradation with workarounds

## Immediate Response Steps

1. Confirm incident with logs, health route, and recent change history.
2. Assign incident commander and communication owner.
3. Contain blast radius (disable risky rollout path, isolate bad route/policy).
4. Execute rollback or mitigation plan.
5. Post status updates at fixed intervals until stable.

## Diagnostic Commands (Local/Bootstrap)

- `go test ./...`
- `go build ./...`
- `python3 scripts/regression_check.py --all --verbose`
- `python3 scripts/log_failure.py --list`

## Post-Incident

- Log failure in registry:
  - `python3 scripts/log_failure.py --interactive`
- Add or update prevention rule in `.guardrails/prevention-rules/`
- Create follow-up regression tests for root-cause path
- Publish concise postmortem and owners
