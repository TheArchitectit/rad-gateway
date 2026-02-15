# SLO and Alerting Baseline

## Service Level Objectives

- API availability (5m window): 99.5%
- Error rate (`5xx` / total): < 1.0%
- P95 latency for non-stream routes: < 1000ms

## Error Budget Policy

- Monthly budget based on 99.5% availability target.
- Budget burn thresholds:
  - warning: 30% burn within 7 days
  - critical: 60% burn within 7 days

## Initial Alert Conditions

- Availability below 99.0% in 15-minute window
- Error rate above 2.0% in 10-minute window
- P95 latency above 1200ms in 15-minute window
- Repeated failover attempts exceed configured retry budget baseline

## Ownership

- Primary: Team 11 (SRE)
- Escalation: Team 12 (IT Operations & Support)

## Bootstrap Notes

- This repository is in bootstrap stage; monitoring targets are defined and ready for runtime implementation.
- CI verification and guardrails checks act as immediate quality controls until full metrics backend is connected.
