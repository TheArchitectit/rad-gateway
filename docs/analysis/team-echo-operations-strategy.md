# RAD Gateway Operations Strategy

## Team Echo: Operations & Observability

**Document Version**: 1.0
**Date**: 2026-02-16
**Status**: Alpha Phase Operations Readiness Review

---

## 1. Executive Summary

### Mission Statement
Team Echo ensures RAD Gateway operates with production-grade reliability, observability, and operational excellence. We bridge the gap between feature development and sustainable operations, implementing practices that enable rapid iteration without compromising stability.

### Current State Assessment

| Dimension | Current State | Target State | Gap |
|-----------|--------------|--------------|-----|
| Deployment | Local/Alpha single-node | Multi-environment Kubernetes | High |
| Observability | In-memory usage/trace stores | Prometheus + OTel + Structured Logs | High |
| SLO Compliance | Defined but not instrumented | Full monitoring with error budgets | High |
| Incident Response | Basic runbook defined | Automated escalation + playbooks | Medium |
| Chaos Engineering | Not implemented | Regular resilience testing | High |

### Key Recommendations (Prioritized)

1. **Immediate (Sprint 0-1)**: Implement metrics endpoint and structured logging (Milestone 4 prerequisite)
2. **Short-term (Sprint 2-4)**: Build containerization pipeline and staging environment
3. **Medium-term (Sprint 5-8)**: Deploy full observability stack with Prometheus/Grafana
4. **Long-term (Sprint 9+)**: Implement chaos engineering and automated remediation

---

## 2. Deployment Architecture Review

### 2.1 Current Architecture Analysis

```
┌─────────────────────────────────────────────────────────────┐
│                     Current State (Alpha)                    │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   ┌──────────────┐     ┌──────────────┐                    │
│   │   Client     │────▶│  Reverse     │                    │
│   │              │     │  Proxy       │                    │
│   └──────────────┘     └──────┬───────┘                    │
│                               │                             │
│                        ┌──────▼───────┐                    │
│                        │  RAD Gateway │                    │
│                        │  (Go 1.24)   │                    │
│                        │              │                    │
│                        │ ┌──────────┐ │                    │
│                        │ │ In-Mem   │ │                    │
│                        │ │ Usage    │ │                    │
│                        │ └──────────┘ │                    │
│                        │ ┌──────────┐ │                    │
│                        │ │ In-Mem   │ │                    │
│                        │ │ Trace    │ │                    │
│                        │ └──────────┘ │                    │
│                        └──────┬───────┘                    │
│                               │                             │
│                        ┌──────▼───────┐                    │
│                        │   Infisical  │                    │
│                        │   (Secrets)  │                    │
│                        └──────────────┘                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Architectural Concerns by Team Role

#### SRE Lead Assessment

**Strengths**:
- HTTP timeouts properly configured (read: 15s, write: 30s, idle: 60s)
- Health endpoint exposed at `/health` (unauthenticated)
- Configurable retry budget (default: 2)

**Critical Gaps**:
- **No persistent state**: In-memory stores lose data on restart (violates data durability SLO)
- **No graceful shutdown**: Server does not handle SIGTERM for in-flight requests
- **Single point of failure**: No redundancy at any layer
- **No circuit breaker**: Provider failures cascade to clients
- **No rate limiting**: API keys are validated but request rates are unbounded

**Reliability Requirements**:
1. Implement graceful shutdown with 30s drain period
2. Add readiness probe beyond basic health check
3. Implement circuit breaker pattern for provider calls
4. Add request rate limiting per API key
5. Design for 2+ replica deployment with session affinity for streaming

#### Observability Engineer Assessment

**Current Instrumentation**:
- Usage records captured with requestID, traceID, duration
- Trace events for gateway lifecycle
- Admin endpoints for introspection (`/v0/management/*`)

**Missing Observability**:
- No metrics exposition (Prometheus endpoint needed)
- No distributed tracing (OpenTelemetry not integrated)
- No structured logging (using standard log package)
- No correlation between logs, metrics, and traces

**Required Instrumentation (Golden Signals)**:

| Signal | SLI | Implementation |
|--------|-----|----------------|
| Latency | P50, P95, P99 | Prometheus histogram with route labels |
| Traffic | RPS per endpoint | Prometheus counter |
| Errors | 4xx/5xx rate | Prometheus counter with status code |
| Saturation | Goroutines, memory | Go runtime metrics |

**Recommended Metrics**:
```
# Request metrics
rad_gateway_requests_total{route, status, method}
rad_gateway_request_duration_seconds{route, quantile}

# Provider metrics
rad_gateway_provider_requests_total{provider, model, status}
rad_gateway_provider_latency_seconds{provider, quantile}
rad_gateway_failover_attempts_total{source_provider, target_provider}
rad_gateway_retry_budget_exhausted_total

# Business metrics
rad_gateway_tokens_consumed_total{model, api_key_name}
rad_gateway_active_connections
rad_gateway_queue_depth
```

#### Chaos Engineer Assessment

**Resilience Gaps Identified**:
1. **No failure injection points**: Cannot test provider degradation
2. **No bulkhead pattern**: Single slow provider affects all requests
3. **No timeout handling on providers**: Only HTTP server timeouts configured
4. **No degraded mode**: System fails hard rather than gracefully

**Chaos Experiment Candidates**:
- Provider latency injection (simulate slow LLM responses)
- Provider failure simulation (100% error rate)
- Memory pressure testing (in-memory store overflow)
- Network partition between gateway and Infisical
- High concurrency load (connection pool exhaustion)

#### Incident Manager Assessment

**Current Runbook Gaps**:
- No severity-specific response procedures
- No communication templates
- No automated incident detection/creation
- No stakeholder notification matrix

**Required Additions**:
1. PagerDuty/Opsgenie integration for alert routing
2. Slack notification templates by severity
3. Status page automation (e.g., Statuspage.io)
4. Postmortem template and tracking

#### Release Manager Assessment

**Current State**:
- CI pipeline with tests, security scanning
- Branch protection on main
- No automated deployment pipeline
- No environment promotion gates

**Required Pipeline**:
- Automated Docker image builds
- Environment-specific deployment workflows
- Canary deployment capability
- Automated rollback triggers

### 2.3 Target Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                    Target Production Architecture                     │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌──────────┐     ┌──────────┐     ┌──────────┐                     │
│  │  CDN     │────▶│  WAF     │────▶│  LB      │                     │
│  └──────────┘     └──────────┘     └────┬─────┘                     │
│                                         │                            │
│                    ┌────────────────────┼────────────────────┐      │
│                    │                    │                    │      │
│              ┌─────▼─────┐        ┌─────▼─────┐        ┌─────▼─────┐│
│              │ RAD GW    │◄──────►│ RAD GW    │◄──────►│ RAD GW    ││
│              │ Replica 1 │        │ Replica 2 │        │ Replica N ││
│              └─────┬─────┘        └─────┬─────┘        └─────┬─────┘│
│                    │                    │                    │      │
│         ┌──────────┼──────────┐         │                    │      │
│         │          │          │         │                    │      │
│    ┌────▼───┐ ┌────▼───┐ ┌────▼───┐   ┌▼─────┐            ┌─▼────┐ │
│    │Prometheus│ │Grafana │ │Jaeger  │   │PostgreSQL│        │Infisical│
│    └────────┘ └────────┘ └────────┘   └────────┘            └──────┘ │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 3. CI/CD Pipeline Specification

### 3.1 Pipeline Architecture

```yaml
# Proposed GitHub Actions Workflow Structure

name: ci-cd-pipeline

on:
  pull_request:
    branches: [main]
  push:
    branches: [main, release/*]
  release:
    types: [published]

jobs:
  # Stage 1: Verification (Current)
  verify:
    runs-on: ubuntu-latest
    steps:
      - checkout
      - go-test
      - go-build
      - security-scan (govulncheck, gosec)
      - guardrails-check

  # Stage 2: Container Build
  build:
    needs: verify
    runs-on: ubuntu-latest
    steps:
      - docker-build
      - docker-push-to-registry
      - sbom-generation
      - vulnerability-scan (Trivy/Grype)

  # Stage 3: Staging Deployment
  deploy-staging:
    needs: build
    environment: staging
    steps:
      - deploy-to-k8s-staging
      - smoke-tests
      - integration-tests

  # Stage 4: Production Deployment (Manual Gate)
  deploy-production:
    needs: deploy-staging
    environment: production  # Requires approval
    steps:
      - canary-deploy (10% traffic)
      - automated-canary-analysis
      - full-rollout-or-rollback
```

### 3.2 Dockerfile Specification

```dockerfile
# Multi-stage build for production
FROM golang:1.24.13-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o rad-gateway ./cmd/rad-gateway

# Final stage
FROM gcr.io/distroless/static-debian12:nonroot

# Security: Run as non-root
USER nonroot:nonroot

COPY --from=builder /app/rad-gateway /rad-gateway

# Health check endpoint
EXPOSE 8090

ENTRYPOINT ["/rad-gateway"]
```

### 3.3 Security Scanning Requirements

| Stage | Tool | Purpose | Failure Condition |
|-------|------|---------|-------------------|
| Build | govulncheck | Go vulnerability scan | Any known vulnerability |
| Build | gosec | Security linting | High/Critical issues |
| Container | Trivy | Image vulnerability | Critical vulnerabilities |
| Container | Syft | SBOM generation | Artifact creation |

---

## 4. Environment Promotion Strategy

### 4.1 Environment Topology

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│    Local    │────▶│    Alpha    │────▶│   Staging   │────▶│ Production  │
│  (Dev env)  │     │  (Single)   │     │  (Replica)  │     │  (HA Setup) │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
     │                    │                   │                    │
     │                    │                   │                    │
   .env file          systemd/Docker    K8s (1 replica)    K8s (2+ replicas)
   go run            SQLite/memory      PostgreSQL         PostgreSQL HA
   mock providers    mock/real          real providers     real providers
```

### 4.2 Promotion Gates

| From | To | Automated Gates | Manual Gates | Rollback SLA |
|------|-----|-----------------|--------------|--------------|
| Local | Alpha | Tests pass | N/A | N/A |
| Alpha | Staging | Image build, security scan | Code review | 15 minutes |
| Staging | Prod | Smoke tests, integration tests | Release manager approval | 5 minutes |

### 4.3 Configuration by Environment

```yaml
# Environment-specific configuration matrix

local:
  log_level: debug
  metrics_enabled: false
  tracing_enabled: false
  providers: mock only
  storage: in-memory

alpha:
  log_level: info
  metrics_enabled: true
  tracing_enabled: false
  providers: mock + limited real
  storage: in-memory (with persistence warnings)

staging:
  log_level: info
  metrics_enabled: true
  tracing_enabled: true
  providers: real (rate limited)
  storage: postgresql

production:
  log_level: warn
  metrics_enabled: true
  tracing_enabled: true
  providers: real (full rate)
  storage: postgresql-ha
```

---

## 5. Observability Framework

### 5.1 Metrics Implementation (Observability Engineer)

#### Prometheus Metrics Endpoint

**Implementation Required**:
```go
// internal/metrics/metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    RequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "rad_gateway_requests_total",
            Help: "Total HTTP requests",
        },
        []string{"route", "method", "status"},
    )

    RequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "rad_gateway_request_duration_seconds",
            Help:    "Request duration distribution",
            Buckets: prometheus.DefBuckets,
        },
        []string{"route"},
    )

    ProviderRequests = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "rad_gateway_provider_requests_total",
            Help: "Provider request attempts",
        },
        []string{"provider", "model", "status"},
    )

    FailoverAttempts = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "rad_gateway_failover_attempts_total",
            Help: "Failover attempts between providers",
        },
        []string{"source_provider", "target_provider", "reason"},
    )

    ActiveConnections = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "rad_gateway_active_connections",
            Help: "Current active connections",
        },
    )
)
```

#### Metric Labels Convention

| Metric | Required Labels | Optional Labels |
|--------|-----------------|-----------------|
| requests_total | route, method, status | api_version, key_name |
| request_duration | route | - |
| provider_requests | provider, model, status | error_type |
| failover_attempts | source, target, reason | - |
| tokens_consumed | model | api_key_name (if privacy allows) |

### 5.2 Structured Logging (Observability Engineer)

#### Log Schema

```go
// Required fields for every log entry
type LogEntry struct {
    Timestamp   string                 `json:"@timestamp"`
    Level       string                 `json:"level"`
    Message     string                 `json:"message"`
    Service     string                 `json:"service"`     // "rad-gateway"
    Version     string                 `json:"version"`     // git sha
    Environment string                 `json:"environment"` // env name
    RequestID   string                 `json:"request_id,omitempty"`
    TraceID     string                 `json:"trace_id,omitempty"`
    Fields      map[string]interface{} `json:"fields,omitempty"`
}
```

#### Log Levels by Environment

| Environment | Default Level | Special Rules |
|-------------|---------------|---------------|
| Local | DEBUG | Include SQL queries, request bodies |
| Alpha | INFO | Include provider request/response metadata |
| Staging | INFO | Standard production-like logging |
| Production | WARN | ERROR for 5xx, WARN for 4xx, INFO for auth failures |

### 5.3 Distributed Tracing (Observability Engineer)

#### OpenTelemetry Configuration

```yaml
# tracing configuration
tracing:
  enabled: true
  exporter: otlp  # or stdout for local
  otlp_endpoint: "jaeger-collector:4317"
  sampling_rate: 0.1  # 10% in production

  # Span attributes to capture
  attributes:
    - request.model
    - request.api_type
    - response.provider
    - response.status
    - retry.attempt_count
    - usage.input_tokens
    - usage.output_tokens
```

#### Critical Spans

1. **gateway_request**: Overall request lifecycle
2. **provider_dispatch**: Provider selection and routing
3. **provider_execute**: Individual provider call
4. **authentication**: API key validation
5. **rate_limit_check**: Quota enforcement
6. **usage_record**: Usage tracking

### 5.4 Dashboard Requirements

#### Grafana Dashboard Suite

1. **SLO Dashboard** (Executive View)
   - Availability (99.5% target with burn rate)
   - Error rate trend
   - Latency percentiles (p50, p95, p99)
   - Error budget remaining

2. **Operational Dashboard** (On-call View)
   - Request rate by route
   - Error rate by status code
   - Provider health status
   - Failover frequency
   - Active connections

3. **Provider Performance Dashboard** (Capacity Planning)
   - Latency by provider
   - Success rate by provider
   - Token throughput
   - Cost per provider

4. **Business Metrics Dashboard** (Product View)
   - Requests by API key
   - Model popularity
   - Token consumption trends
   - Error types

---

## 6. Alerting and SLO Compliance

### 6.1 SLO Definitions (SRE Lead)

#### Service Level Objectives

| SLI | SLO | Measurement Window | Data Source |
|-----|-----|-------------------|-------------|
| Availability | 99.5% | Rolling 30 days | Prometheus (up metric) |
| Error Rate | < 1% | Rolling 7 days | 5xx / total requests |
| P95 Latency | < 1000ms | Rolling 7 days | Histogram quantile |
| P99 Latency | < 2000ms | Rolling 7 days | Histogram quantile |

#### Error Budget Calculation

```
Monthly Error Budget (Availability):
- 30 days = 2,592,000 seconds
- 0.5% budget = 12,960 seconds (3.6 hours) of downtime

Burn Rate Alerts:
- Warning: 30% budget burn in 7 days
  - Threshold: 1,080 minutes downtime / 7 days = 108 min/day
- Critical: 60% budget burn in 7 days
  - Threshold: 2,160 minutes downtime / 7 days = 216 min/day
```

### 6.2 Alerting Rules (SRE Lead)

#### PagerDuty Integration

```yaml
# Alert routing
alert_routing:
  team_echo:
    - availability_alerts
    - latency_alerts
    - error_rate_alerts

  platform_team:
    - infrastructure_alerts
    - database_alerts

  escalation:
    - primary: team_echo_oncall
    - secondary: sre_lead
    - tertiary: engineering_manager
```

#### Alert Definitions

| Alert Name | Condition | Severity | Runbook | Auto-Action |
|------------|-----------|----------|---------|-------------|
| HighErrorRate | error_rate > 2% for 5m | P1 | error-rate-runbook | Notify only |
| AvailabilityDrop | availability < 99% for 2m | P1 | availability-runbook | Page on-call |
| HighLatency | p95 > 1200ms for 10m | P2 | latency-runbook | Notify only |
| ErrorBudgetBurn | 30% burn in 7 days | P2 | budget-runbook | Email team |
| ErrorBudgetCritical | 60% burn in 7 days | P1 | budget-runbook | Page on-call |
| ProviderDegraded | provider_error_rate > 10% | P3 | provider-runbook | Auto-failover |
| CircuitBreakerOpen | breaker_open == 1 | P2 | circuit-runbook | Notify only |
| HighMemoryUsage | memory > 80% | P3 | capacity-runbook | Notify only |
| DBConnectionExhausted | db_conns == max | P1 | database-runbook | Page on-call |

### 6.3 Alert Quality Standards

#### Alert Hygiene Requirements

- **Actionable**: Every alert must have a runbook link
- **Relevant**: < 5% false positive rate
- **Urgent**: Clear severity indicating response time
- **Triageable**: Sufficient context to understand scope

#### Alert Fatigue Prevention

1. Group related alerts (e.g., "High latency" across multiple routes)
2. Implement alert silencing during deployments
3. Require post-incident alert review for false positives
4. Weekly alert analytics review

---

## 7. Incident Response Procedures Review

### 7.1 Severity Classification (Incident Manager)

| Severity | Criteria | Response Time | Communication |
|----------|----------|---------------|---------------|
| SEV-1 | Complete outage, data loss, security breach | 5 minutes | Immediate war room, status page, exec notification |
| SEV-2 | Major degradation (>25% error rate), core feature broken | 15 minutes | War room, status page update |
| SEV-3 | Minor degradation, workaround available | 1 hour | Ticket tracking, next-day review |
| SEV-4 | Cosmetic issues, monitoring gaps | 24 hours | Backlog item |

### 7.2 Incident Response Workflow

```
┌─────────────────────────────────────────────────────────────────┐
│                     Incident Response Flow                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  [Detection] ──▶ [Triage] ──▶ [Response] ──▶ [Resolution]       │
│       │              │             │              │             │
│       ▼              ▼             ▼              ▼             │
│   Alert fires   Assess scope   Execute        Verify fix       │
│   or report     Assign sev     mitigation     Close incident   │
│                 Page on-call   Communicate   Postmortem       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 7.3 Runbook Catalog (Incident Manager)

#### Required Runbooks

1. **availability-degradation.md**
   - Health check verification
   - Load balancer investigation
   - Rolling restart procedure

2. **error-rate-spike.md**
   - Identify error type breakdown
   - Recent deployment correlation
   - Provider status check
   - Rollback decision tree

3. **latency-investigation.md**
   - Latency histogram analysis
   - Provider latency breakdown
   - Database query analysis
   - Circuit breaker status

4. **provider-failure.md**
   - Provider health verification
   - Manual failover procedure
   - Provider capacity check

5. **security-incident.md**
   - API key revocation procedure
   - Audit log extraction
   - Forensic data preservation
   - Communication templates

### 7.4 Communication Templates

#### Status Page Updates

```
[Investigating] RAD Gateway - Elevated Error Rates
We are investigating elevated error rates on the chat completions API.
Impact: Some requests may fail. Estimated: 5% of requests.
Started: {{timestamp}}
Next update: 30 minutes
```

```
[Resolved] RAD Gateway - Service Restored
The issue has been resolved. All services are operating normally.
Duration: {{duration}}
Root cause: {{brief_description}}
Postmortem: {{link}}
```

---

## 8. Chaos Engineering Test Plan

### 8.1 Experiment Priorities (Chaos Engineer)

#### Phase 1: Provider Resilience (Milestone 1 completion)

| Experiment | Hypothesis | Blast Radius | Safety |
|------------|------------|--------------|--------|
| Provider Latency Injection | Failover triggers when provider > 5s | Single provider | Abort if all providers affected |
| Provider Failure Simulation | 100% error rate triggers circuit breaker | Single provider | Auto-abort after 5 minutes |
| Retry Budget Exhaustion | System returns 503 when budget exhausted | Test environment only | Hard limit on attempts |

#### Phase 2: Infrastructure Resilience (Kubernetes migration)

| Experiment | Hypothesis | Blast Radius | Safety |
|------------|------------|--------------|--------|
| Pod Failure | Traffic routes to healthy replicas | Single pod | Min replicas = 2 |
| Network Partition | Gateway gracefully handles Infisical unreachability | Infisical connection | Cache last known secrets |
| Memory Pressure | OOMKilled pods restart without data loss | Single pod | PostgreSQL persistence |

#### Phase 3: Dependency Resilience (Full production)

| Experiment | Hypothesis | Blast Radius | Safety |
|------------|------------|--------------|--------|
| Database Degradation | Queue builds, requests timeout gracefully | Database queries | Circuit breaker on DB |
| Certificate Expiry | TLS handshake fails with clear error | TLS layer | Staging only |
| Clock Skew | JWT validation fails appropriately | Auth layer | Canary only |

### 8.2 Experiment Automation

```yaml
# chaos-experiment.yaml
apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  name: rad-gateway-provider-chaos
spec:
  appinfo:
    appns: 'rad-gateway'
    applabel: 'app=rad-gateway'
    appkind: 'deployment'
  # Safety settings
  annotationCheck: 'true'
  engineState: 'active'
  # Abort conditions
  experiments:
    - name: provider-latency
      spec:
        components:
          env:
            - name: TARGET_PROVIDER
              value: "openai"
            - name: LATENCY_MS
              value: "5000"
            - name: DURATION
              value: "300"
```

### 8.3 Game Day Schedule

| Frequency | Scope | Participants | Duration |
|-----------|-------|--------------|----------|
| Weekly | Local development | Chaos Engineer | 1 hour |
| Bi-weekly | Staging environment | Team Echo | 2 hours |
| Monthly | Production (limited) | Team Echo + Stakeholders | 4 hours |
| Quarterly | Full disaster recovery | Organization-wide | Full day |

---

## 9. Release Management Process

### 9.1 Release Lifecycle (Release Manager)

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Release Lifecycle                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Planning ──▶ Development ──▶ Validation ──▶ Deployment ──▶ Monitor │
│     │              │              │              │            │     │
│     ▼              ▼              ▼              ▼            ▼     │
│  Milestone      Feature       CI Pipeline   Staging     Production │
│  alignment      branches      + tests       validation    deploy   │
│                 + PRs                                         │     │
│                                                              │       │
│                                                              ▼       │
│                                                        Post-release   │
│                                                        verification  │
└─────────────────────────────────────────────────────────────────────┘
```

### 9.2 Release Types

| Type | Cadence | Criteria | Approval |
|------|---------|----------|----------|
| Hotfix | As needed | Critical bug or security issue | Release Manager + SRE Lead |
| Patch | Weekly | Bug fixes, minor improvements | Release Manager |
| Minor | Bi-weekly | New features (backward compatible) | Product + Engineering |
| Major | Quarterly | Breaking changes, major milestones | Executive |

### 9.3 Deployment Windows

| Environment | Deployment Window | Freeze Periods |
|-------------|-------------------|----------------|
| Staging | 24/7 (automated) | None |
| Production | Tue-Thu 9AM-4PM local | Black Friday, Year-end |
| Hotfix | 24/7 with approval | None |

### 9.4 Go/No-Go Criteria

#### Pre-Deployment Checklist

- [ ] All CI checks passing
- [ ] Staging deployment validated for 24+ hours
- [ ] No critical or high vulnerabilities in scan
- [ ] Error rate in staging < 0.5%
- [ ] P95 latency in staging < 800ms
- [ ] Database migration tested (if applicable)
- [ ] Rollback procedure tested
- [ ] On-call engineer notified
- [ ] Feature flags configured (if using)

#### Deployment Decision Matrix

| Scenario | Decision | Action |
|----------|----------|--------|
| All criteria met | GO | Proceed with deployment |
| Minor latency increase | GO with monitoring | Deploy with extended canary |
| Staging error rate 0.5-1% | CONDITIONAL GO | Risk assessment required |
| Security vulnerability found | NO-GO | Fix and reschedule |
| Database migration untested | NO-GO | Complete testing first |

### 9.5 Rollback Procedures

#### Automatic Rollback Triggers

- Error rate exceeds 5% for 2 minutes
- P95 latency exceeds 2000ms for 5 minutes
- Availability drops below 95%
- Any SEV-1 incident during deployment

#### Manual Rollback Procedure

```bash
# 1. Identify last known good version
LAST_GOOD=$(git tag --sort=-version:refname | grep -A1 "current" | tail -1)

# 2. Initiate rollback
kubectl rollout undo deployment/rad-gateway

# 3. Verify rollback
kubectl rollout status deployment/rad-gateway
./scripts/smoke-test.sh

# 4. Communicate
# Post in #incidents channel
# Update status page if customer-facing
```

---

## 10. Operations Readiness Checklist

### 10.1 Production Readiness Criteria

#### Infrastructure

- [ ] Container images building successfully
- [ ] Kubernetes manifests created and tested
- [ ] Horizontal Pod Autoscaling configured (min: 2, max: 10)
- [ ] Network policies defined and applied
- [ ] TLS termination configured with valid certificates
- [ ] DNS records created and propagated
- [ ] Load balancer health checks configured

#### Observability

- [ ] Prometheus metrics endpoint exposed at `/metrics`
- [ ] Grafana dashboards imported and validated
- [ ] Alertmanager rules loaded and tested
- [ ] PagerDuty integration tested
- [ ] Log aggregation pipeline configured
- [ ] Distributed tracing configured
- [ ] SLO dashboard with burn rate alerting

#### Reliability

- [ ] Graceful shutdown handling implemented
- [ ] Circuit breaker pattern for providers
- [ ] Retry with exponential backoff configured
- [ ] Rate limiting per API key
- [ ] Database connection pooling configured
- [ ] Health checks (liveness + readiness) implemented
- [ ] Resource limits (CPU/memory) defined

#### Security

- [ ] Secrets management with Infisical validated
- [ ] API key rotation procedure documented
- [ ] Security scanning in CI/CD pipeline
- [ ] Network segmentation between services
- [ ] Audit logging enabled
- [ ] Penetration test completed

#### Documentation

- [ ] Runbooks created for all P1/P2 alerts
- [ ] Architecture diagrams updated
- [ ] On-call handbook published
- [ ] Incident response procedures tested
- [ ] Postmortem template defined
- [ ] Operations playbook complete

### 10.2 Team Readiness

| Role | Readiness Criteria | Verification |
|------|-------------------|--------------|
| SRE Lead | SLOs defined, error budgets calculated, escalation matrix published | Review meeting |
| Observability Engineer | Dashboards live, alerts tested, log pipeline verified | Drill exercise |
| Chaos Engineer | Experiments designed, staging chaos proven | Chaos day completed |
| Incident Manager | Runbooks reviewed, war room tested, comms templates ready | Tabletop exercise |
| Release Manager | Pipeline tested, rollback verified, go/no-go process defined | Release dry-run |

### 10.3 Launch Exit Criteria

Before RAD Gateway graduates from Alpha to Beta:

1. **Reliability**: 7 consecutive days of 99.5% availability in staging
2. **Observability**: All P1 alerts with < 5% false positive rate
3. **Security**: No critical vulnerabilities, penetration test passed
4. **Performance**: P95 latency < 800ms sustained for 48 hours
5. **Operations**: On-call rotation established with trained engineers
6. **Documentation**: All runbooks tested and validated

---

## Appendices

### A. Recommended Tooling Stack

| Category | Tool | Purpose | Alternative |
|----------|------|---------|-------------|
| Metrics | Prometheus | Time-series metrics | InfluxDB |
| Visualization | Grafana | Dashboards | Datadog |
| Logging | Loki/ELK | Log aggregation | Splunk |
| Tracing | Jaeger/Tempo | Distributed tracing | Zipkin |
| Alerting | Alertmanager | Alert routing | PagerDuty native |
| Chaos | LitmusChaos | Chaos engineering | Gremlin |
| Secrets | Infisical | Secret management | Vault |
| CI/CD | GitHub Actions | Build pipeline | GitLab CI |

### B. Cost Estimation

| Component | Monthly Cost (est.) | Notes |
|-----------|---------------------|-------|
| Kubernetes cluster (EKS/GKE) | $200-500 | 3 nodes, t3.medium |
| PostgreSQL (managed) | $100-300 | db.t3.micro to small |
| Monitoring stack | $50-150 | Prometheus + Grafana |
| Log aggregation | $100-200 | Based on volume |
| Secrets management | $0-50 | Infisical self-hosted |
| Load balancer | $20-50 | ALB/NLB |
| **Total** | **$470-1250/month** | Excludes provider API costs |

### C. Key Contacts

| Role | Team | Responsibility | Escalation |
|------|------|----------------|------------|
| SRE Lead | Team Echo | Error budgets, reliability | VP Engineering |
| Observability Eng | Team Echo | Monitoring, metrics, alerts | SRE Lead |
| Chaos Engineer | Team Echo | Resilience testing | SRE Lead |
| Incident Manager | Team Echo | War room, response | SRE Lead |
| Release Manager | Team Echo | Deployments, rollbacks | VP Engineering |

---

**Document Ownership**: Team Echo (Operations & Observability)
**Review Cycle**: Monthly during Alpha, Quarterly in production
**Next Review Date**: 2026-03-16
