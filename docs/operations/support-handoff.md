# Support Handoff

## Handoff Package

- Service overview: `README.md`
- Security and secret handling policy: `SECURITY.md`
- Contributor/developer workflow: `CONTRIBUTING.md`
- Gate and operational docs:
  - `docs/phase-gates/*.md`
  - `docs/operations/*.md`

## Operational Owners

- Reliability and observability ownership: Team 11
- Release/change and L3 escalation ownership: Team 12

## Escalation Flow

1. NOC analyst validates incident signal.
2. Incident manager forms war room and assigns severity.
3. L3 support engineer triages with service owner.
4. Release manager coordinates rollback/hotfix if needed.

## Support Readiness Criteria

- Runbooks available and up to date
- Alert conditions defined
- Release checklist accepted
- Failure logging and regression checks executable
