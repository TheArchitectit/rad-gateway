# Beta vs Production Deployment

**Purpose**: Document what can be skipped for beta vs what is required for production

**Status**: Reference Guide
**Last Updated**: 2026-02-16

---

## Quick Reference

| Feature/Requirement | Beta | Production | Notes |
|--------------------|------|------------|-------|
| **Secrets Management** ||||
| Infisical | ✅ Required | ✅ Required | Primary secrets platform |
| OpenBao | ⚠️ Reserved | ✅ Required | Cold vault for compliance |
| **Security** ||||
| PostgreSQL internal only | ✅ Required | ✅ Required | No host port exposure |
| Firewall rules | ⏸️ **SKIP** | ✅ Required | User decision for beta |
| TLS certificates | ⏸️ Optional | ✅ Required | Can use HTTP for beta testing |
| Secret rotation automation | ⏸️ **SKIP** | ✅ Required | Manual rotation OK for beta |
| Audit logging | ⚠️ Partial | ✅ Required | Infisical logs OK for beta |
| **Operations** ||||
| Backup procedures | ✅ Required | ✅ Required | Daily backups |
| Backup encryption | ⏸️ **SKIP** | ✅ Required | Unencrypted OK for beta |
| Off-site backups | ⏸️ **SKIP** | ✅ Required | Local only for beta |
| Monitoring/alerting | ⚠️ Basic | ✅ Required | Basic health checks OK |
| Log aggregation | ⏸️ **SKIP** | ✅ Required | Local logs OK for beta |
| Auto-scaling | ⏸️ **SKIP** | ✅ Required | Single node for beta |
| Load balancing | ⏸️ **SKIP** | ✅ Required | Direct access for beta |
| **High Availability** ||||
| PostgreSQL HA | ⏸️ **SKIP** | ✅ Required | Single instance for beta |
| Infisical HA | ⏸️ **SKIP** | ✅ Required | Single instance for beta |
| Multi-node deployment | ⏸️ **SKIP** | ✅ Required | Single host for beta |
| **QA/Testing** ||||
| Smoke tests | ✅ Required | ✅ Required | E2E validation |
| Load tests | ⏸️ **SKIP** | ✅ Required | Minimal load for beta |
| Chaos engineering | ⏸️ **SKIP** | ✅ Required | Not needed for beta |
| Penetration testing | ⏸️ **SKIP** | ✅ Required | Security review OK for beta |
| Pre-deployment validation | ⏸️ **SKIP** | ✅ Required | Manual checks OK for beta |
| Health integration tests | ⏸️ **SKIP** | ✅ Required | Basic health checks OK for beta |
| **Documentation** ||||
| Runbooks | ✅ Required | ✅ Required | Incident procedures |
| Post-mortem process | ⚠️ Basic | ✅ Required | Learn from incidents |
| DR procedures | ✅ Required | ✅ Required | Documented restore process |

---

## Legend

| Symbol | Meaning |
|--------|---------|
| ✅ Required | Must implement |
| ⚠️ Partial | Partial implementation OK |
| ⏸️ SKIP | Can skip for beta |

---

## Detailed Rationale

### Can SKIP for Beta (15 items)

**Note**: P1 items from Team India review can be deferred to production phase for beta.

**P1 QA items can be deferred to production** - The following pre-deployment validation and health check integration tests are production requirements but can be skipped for beta deployment.

1. **Firewall Rules** ⏸️
   - Beta: Internal network only, firewall not required
   - Production: Network segmentation required
   - Risk: Low for beta single-host deployment

2. **Backup Encryption** ⏸️
   - Beta: Backups stored on same host
   - Production: Encrypt backups before off-site transfer
   - Risk: Medium (physical access to host)

3. **Off-site Backups** ⏸️
   - Beta: Local backups sufficient
   - Production: Must have off-site copy
   - Risk: High if single host fails

4. **Log Aggregation** ⏸️
   - Beta: Local logs accessible
   - Production: Centralized logging required
   - Risk: Low for troubleshooting

5. **Auto-scaling** ⏸️
   - Beta: Fixed capacity acceptable
   - Production: Must scale with demand
   - Risk: Performance limitations

6. **PostgreSQL HA** ⏸️
   - Beta: Single instance OK
   - Production: Replica required
   - Risk: Database downtime

7. **Infisical HA** ⏸️
   - Beta: Single instance OK
   - Production: HA deployment
   - Risk: Secrets unavailability

8. **Load Testing** ⏸️
   - Beta: Functional testing sufficient
   - Production: Must validate capacity
   - Risk: Unknown performance limits

9. **Token File Backup Exclusion Documentation** ⏸️ *[P1 - Team India]*
   - Beta: Basic exclusion note sufficient
   - Production: Full security audit of backup exclusions
   - Risk: Low (token is short-lived bootstrap credential)

10. **Startup Script Error Handling** ⏸️ *[P1 - Team India]*
    - Beta: Basic error handling with exit codes
    - Production: Comprehensive error handling with retries and alerting
    - Risk: Medium (service may fail silently)

11. **Advanced Monitoring/Alerting** ⏸️ *[P1 - Team India]*
    - Beta: Basic health checks and manual monitoring OK
    - Production: Automated alerting with PagerDuty/Opsgenie integration
    - Risk: Low (manual monitoring sufficient for beta)

12. **Production-Grade Deployment/Rollback** ⏸️ *[P1 - Team India]*
    - Beta: Simple binary swap rollback OK
    - Production: Blue/green deployment with automated canary analysis
    - Risk: Low (beta can tolerate brief downtime)

13. **Pre-Deployment Validation Script** ⏸️ *[P1 QA - Team India]*
    - Beta: Manual deployment checks OK
    - Production: Automated pre-deployment validation required
    - Risk: Low (manual verification sufficient for beta)

14. **Health Check Integration Tests** ⏸️ *[P1 QA - Team India]*
    - Beta: Basic health endpoint verification OK
    - Production: Comprehensive integration test suite required
    - Risk: Low (manual health checks sufficient for beta)

### Must Have Even for Beta (6 items)

1. **PostgreSQL Internal Only** ✅
   - Security: Database must not be exposed
   - Required: YES for both

2. **Backup Procedures** ✅
   - Recovery: Must be able to restore
   - Required: YES for both

3. **Health Checks** ✅
   - Operations: Must verify system health
   - Required: YES for both

4. **Smoke Tests** ✅
   - QA: Must validate basic functionality
   - Required: YES for both

5. **Runbooks** ✅
   - Operations: Must handle incidents
   - Required: YES for both

6. **DR Procedures** ✅
   - Recovery: Must document restore process
   - Required: YES for both

### Reserved for Post-Beta (3 items)

1. **OpenBao Cold Vault** ⚠️
   - Current: Reserved (deployed, not configured)
   - Beta: Not used
   - Production: Required for compliance

2. **Secret Rotation Automation** ⚠️
   - Beta: Manual rotation OK
   - Production: Automated rotation required

3. **Audit Logging (OpenBao)** ⚠️
   - Beta: Infisical logs sufficient
   - Production: Immutable audit trail required

---

## Migration Path

### Beta → Production Checklist

```
□ Deploy OpenBao (configure for cold vault)
□ Implement firewall rules (network segmentation)
□ Enable backup encryption
□ Configure off-site backup replication
□ Set up centralized log aggregation
□ Implement auto-scaling (Kubernetes/ECS)
□ Deploy PostgreSQL HA (primary + replica)
□ Deploy Infisical HA (multi-instance)
□ Configure load balancing
□ Perform load testing (validate capacity)
□ Automate secret rotation
□ Enable OpenBao audit logging
□ Implement TLS everywhere
□ Complete security penetration test
□ Document production runbooks
```

---

## Decision Log

| Date | Decision | By | Rationale |
|------|----------|-----|-----------|
| 2026-02-16 | Skip firewall rules for beta | User | Single-host deployment, internal network only |
| 2026-02-16 | Reserve OpenBao for post-beta | Team | Reduce complexity for beta |
| 2026-02-16 | Require backup procedures for beta | Team India | Must be able to recover from failure |

---

**See Also**:
- [Beta Deployment](beta-deployment.md) - Beta-specific architecture
- [Golden Stack](../operations/golden-stack.md) - Full deployment documentation
- [Team India Review](../reviews/team-india-final-report.md) - Review findings
