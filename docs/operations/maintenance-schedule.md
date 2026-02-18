# RAD Gateway Maintenance Schedule

## Overview

This document defines the automated maintenance procedures for RAD Gateway databases and infrastructure. These procedures are designed to maintain system reliability, prevent data loss, and ensure optimal performance.

**Team**: Team Echo (Operations & Observability)
**Owner**: Reliability Engineer
**Last Updated**: 2026-02-18

---

## Maintenance Schedule

### Daily Tasks

| Task | Script | Schedule | Duration | Impact |
|------|--------|----------|----------|--------|
| PostgreSQL VACUUM/ANALYZE | `postgres-maintenance.sh` | 02:00 AM | 10-30 min | Minimal |
| Backup Verification | `backup-verification.sh` | 02:00 AM, 02:00 PM | 5-15 min | None |
| System Health Check | `system-health-check.sh` | 06:00 AM | 1-2 min | None |

### Hourly Tasks

| Task | Script | Schedule | Duration | Impact |
|------|--------|----------|----------|--------|
| Redis Cache Maintenance | `redis-maintenance.sh` | Every hour (xx:00) | <1 min | None |

### Continuous Tasks

| Task | Script | Schedule | Duration | Impact |
|------|--------|----------|----------|--------|
| Health Monitoring | `health-monitoring.sh` | Every 5 minutes | <30 sec | None |

### Weekly Tasks

| Task | Schedule | Description |
|------|----------|-------------|
| Metrics Cleanup | Sunday 03:00 AM | Remove metrics files older than 90 days |
| Alert Log Cleanup | Sunday 03:30 AM | Remove alert logs older than 90 days |
| Full Maintenance Review | Sunday 04:00 AM | Generate maintenance report |

### Monthly Tasks

| Task | Schedule | Description |
|------|----------|-------------|
| Maintenance Log Cleanup | 1st of month | Remove logs older than 30 days |
| SLO Review | 1st of month | Review error budgets and SLO compliance |

---

## Maintenance Scripts

### PostgreSQL Maintenance (`postgres-maintenance.sh`)

**Purpose**: Performs database optimization and health checks.

**Operations**:
1. Connection test
2. Statistics collection
3. VACUUM ANALYZE on tables needing maintenance
4. Bloat detection
5. Index health check
6. Metrics collection

**Configuration**:
```bash
PG_HOST=localhost          # PostgreSQL host
PG_PORT=5432              # PostgreSQL port
PG_USER=radgateway_user   # Database user
PG_DATABASE=radgateway    # Database name
PG_PASSWORD=              # Password (loaded from env)
VACUUM_ANALYZE_THRESHOLD_DAYS=7
BLOAT_THRESHOLD_PERCENT=30
MIN_TABLE_SIZE_MB=10
```

**Logs**:
- Main log: `/opt/radgateway01/logs/postgres-maintenance-YYYYMMDD-HHMMSS.log`
- Metrics: `/opt/radgateway01/logs/postgres-maintenance-metrics.json`

**Alerts**:
- Table bloat > 30%
- High dead tuple ratio (>20%)

---

### Redis Cache Maintenance (`redis-maintenance.sh`)

**Purpose**: Monitors and maintains Redis cache performance.

**Operations**:
1. Connection test
2. Memory usage monitoring
3. Fragmentation ratio check
4. Cache hit/miss analysis
5. Expired key cleanup
6. Eviction policy validation
7. Persistence check

**Configuration**:
```bash
REDIS_HOST=localhost      # Redis host
REDIS_PORT=6379          # Redis port
REDIS_PASSWORD=          # Password (loaded from env)
REDIS_DB=0              # Database number
MEMORY_THRESHOLD_PERCENT=80
FRAGMENTATION_THRESHOLD=1.5
```

**Logs**:
- Main log: `/opt/radgateway01/logs/redis-maintenance-YYYYMMDD-HHMMSS.log`
- Metrics: `/opt/radgateway01/logs/redis-maintenance-metrics.json`
- Alerts: `/opt/radgateway01/logs/redis-maintenance-alerts.log`

**Alerts**:
- Memory usage > 80%
- Fragmentation ratio > 1.5
- Low cache hit rate (<50%)

---

### Backup Verification (`backup-verification.sh`)

**Purpose**: Validates backup integrity and tests restore procedures.

**Operations**:
1. Find backup files
2. Verify file age
3. Check file size
4. Verify checksums
5. Test restore to temporary database
6. Verify restored data
7. Cleanup old backups

**Configuration**:
```bash
BACKUP_DIR=/opt/radgateway01/backups
TEMP_DIR=/tmp/radgateway-backup-verify
MAX_BACKUP_AGE_HOURS=24
MIN_BACKUP_SIZE_MB=1
TEST_RESTORE=true
VERIFY_CHECKSUM=true
RETENTION_DAYS=7
```

**Logs**:
- Main log: `/opt/radgateway01/logs/backup-verification-YYYYMMDD-HHMMSS.log`
- Metrics: `/opt/radgateway01/logs/backup-verification-metrics.json`
- Report: `/opt/radgateway01/logs/backup-verification-report.txt`

**Alerts**:
- Backup age exceeds threshold
- Corrupt backup files
- Failed restore tests

---

### Health Monitoring (`health-monitoring.sh`)

**Purpose**: Continuous monitoring of RAD Gateway and dependencies.

**Operations**:
1. RAD Gateway health endpoint check
2. API latency measurement (P95)
3. PostgreSQL health check
4. Redis health check
5. Disk usage monitoring
6. Memory usage monitoring
7. Container health check
8. Service status check
9. Error budget calculation

**Configuration**:
```bash
RAD_HOST=localhost
RAD_PORT=8090
SLO_AVAILABILITY=99.5
SLO_ERROR_RATE=1.0
SLO_P95_LATENCY=1000
ALERT_AVAILABILITY=99.0
ALERT_ERROR_RATE=2.0
ALERT_P95_LATENCY=1200
ALERT_DISK_USAGE=85
ALERT_MEMORY_USAGE=85
```

**Logs**:
- Main log: `/opt/radgateway01/logs/health-monitoring-YYYYMMDD-HHMMSS.log`
- Metrics: `/opt/radgateway01/logs/health-monitoring-metrics.json`
- Status: `/opt/radgateway01/logs/health-monitoring-status.json`
- Alerts: `/opt/radgateway01/logs/health-monitoring-alerts.log`

**SLO Targets** (from `slo-and-alerting.md`):
- API availability: 99.5%
- Error rate: <1.0%
- P95 latency: <1000ms

**Alerts**:
- Service unavailable
- Database connection failure
- SLO threshold breaches

---

## Installation

### 1. Copy Scripts

```bash
sudo mkdir -p /mnt/ollama/git/RADAPI01/scripts/maintenance
sudo cp /mnt/ollama/git/RADAPI01/scripts/maintenance/*.sh /usr/local/bin/
sudo chmod +x /usr/local/bin/*.sh
```

### 2. Create Log Directory

```bash
sudo mkdir -p /opt/radgateway01/logs
sudo chown radgateway:radgateway /opt/radgateway01/logs
sudo chmod 755 /opt/radgateway01/logs
```

### 3. Install Cron Jobs

```bash
# Copy cron file
sudo cp /mnt/ollama/git/RADAPI01/scripts/maintenance/crontab-maintenance /etc/cron.d/radgateway-maintenance

# Set permissions
sudo chmod 644 /etc/cron.d/radgateway-maintenance

# Reload cron
sudo systemctl restart cron  # Debian/Ubuntu
# or
sudo systemctl restart crond   # RHEL/CentOS
```

### 4. Verify Installation

```bash
# Check cron jobs are loaded
cat /etc/cron.d/radgateway-maintenance

# Test a script manually
sudo /mnt/ollama/git/RADAPI01/scripts/maintenance/health-monitoring.sh

# Check logs
ls -la /opt/radgateway01/logs/
```

---

## Runbooks

### Runbook 1: PostgreSQL Maintenance Failure

**Symptoms**: PostgreSQL maintenance script fails or reports errors.

**Detection**: Alert from maintenance log, or manual check shows failed VACUUM.

**Response**:

1. **Check connection**:
   ```bash
   sudo -u radgateway psql postgresql://radgateway_user@localhost:5432/radgateway -c "SELECT 1"
   ```

2. **Review logs**:
   ```bash
   sudo tail -f /opt/radgateway01/logs/postgres-maintenance-*.log
   ```

3. **Check PostgreSQL status**:
   ```bash
   sudo systemctl status postgresql
   sudo pg_isready -h localhost -p 5432
   ```

4. **Manual VACUUM** (if automated fails):
   ```bash
   sudo -u postgres psql -d radgateway -c "VACUUM ANALYZE;"
   ```

5. **Check for locks**:
   ```bash
   sudo -u postgres psql -c "SELECT * FROM pg_stat_activity WHERE state = 'active';"
   ```

**Escalation**: If PostgreSQL is down, follow the Database Incident Response Runbook.

---

### Runbook 2: Redis Memory High

**Symptoms**: Redis memory usage >80%, high fragmentation ratio.

**Detection**: Alert from `redis-maintenance.sh`, or monitoring dashboard.

**Response**:

1. **Check memory usage**:
   ```bash
   redis-cli INFO MEMORY
   ```

2. **Identify large keys**:
   ```bash
   redis-cli --bigkeys
   ```

3. **Check eviction policy**:
   ```bash
   redis-cli CONFIG GET maxmemory-policy
   ```

4. **Consider actions**:
   - Increase `maxmemory` limit
   - Change eviction policy to `allkeys-lru`
   - Restart Redis (clears fragmentation)

5. **Restart Redis** (if needed):
   ```bash
   sudo systemctl restart redis
   ```

**Escalation**: If restart fails, involve Team Echo (Operations).

---

### Runbook 3: Backup Verification Failure

**Symptoms**: Backup verification reports failed checksums or restore tests.

**Detection**: Alert from `backup-verification.sh`, or daily report.

**Response**:

1. **Check backup files**:
   ```bash
   ls -la /opt/radgateway01/backups/
   ```

2. **Verify file integrity**:
   ```bash
   sha256sum -c /opt/radgateway01/backups/*.sha256
   ```

3. **Check disk space**:
   ```bash
   df -h /opt/radgateway01/
   ```

4. **Create manual backup**:
   ```bash
   pg_dump -h localhost -U radgateway_user radgateway > /opt/radgateway01/backups/manual-$(date +%Y%m%d-%H%M%S).sql
   ```

5. **Generate new checksum**:
   ```bash
   sha256sum backup.sql > backup.sql.sha256
   ```

**Escalation**: If backups are consistently failing, involve Team Hotel (Deployment).

---

### Runbook 4: Health Check Alerts

**Symptoms**: Health monitoring reports SLO breaches or service unavailability.

**Detection**: Alert from `health-monitoring.sh`, or monitoring dashboard.

**Response**:

1. **Check service status**:
   ```bash
   sudo systemctl status radgateway01
   sudo podman ps --pod
   ```

2. **Check health endpoint**:
   ```bash
   curl -s http://localhost:8090/health
   ```

3. **Check logs**:
   ```bash
   sudo journalctl -u radgateway01 -n 100
   sudo podman logs radgateway01-app --tail 100
   ```

4. **Check resource usage**:
   ```bash
   free -h
df -h
   ```

5. **Check database connections**:
   ```bash
   sudo -u postgres psql -c "SELECT count(*) FROM pg_stat_activity;"
   ```

**Severity Levels**:
- **Critical**: Service completely unavailable
- **Warning**: SLO breach but service functional
- **Info**: Below threshold but not critical

**Escalation**:
- Critical: Page on-call engineer immediately
- Warning: Create ticket for investigation
- Info: Monitor and trend

---

## Escalation Matrix

| Severity | Condition | Response Time | Escalation |
|----------|-------------|---------------|------------|
| P1 - Critical | Service down, data loss risk | Immediate | Team Lead + On-call engineer |
| P2 - High | SLO breach, degraded performance | 15 minutes | Team Echo + Team Delta |
| P3 - Medium | Warning thresholds exceeded | 1 hour | Team Echo |
| P4 - Low | Informational alerts | 4 hours | Create ticket |

**Escalation Path**:
1. On-call engineer
2. Team Echo Lead
3. Engineering Manager
4. VP Engineering

---

## Metrics Reference

### PostgreSQL Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `database_size` | Gauge | Total database size |
| `table_count` | Gauge | Number of tables |
| `total_dead_tuples` | Counter | Accumulated dead tuples |
| `total_live_tuples` | Gauge | Current live tuples |

### Redis Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `used_memory_bytes` | Gauge | Memory usage in bytes |
| `used_memory_rss_bytes` | Gauge | Resident set size |
| `total_keys` | Gauge | Total key count |
| `connected_clients` | Gauge | Active client connections |

### Health Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `rad_gateway_response_time` | Gauge | Health endpoint latency (ms) |
| `api_latency_p95` | Gauge | P95 API latency (ms) |
| `postgresql_connections` | Gauge | Active DB connections |
| `redis_memory_usage` | Gauge | Redis memory percentage |
| `error_rate` | Gauge | Error rate percentage |

---

## Related Documentation

- [SLO and Alerting Baseline](/mnt/ollama/git/RADAPI01/docs/operations/slo-and-alerting.md)
- [Incident Runbook](/mnt/ollama/git/RADAPI01/docs/operations/incident-runbook.md)
- [Database Setup](/mnt/ollama/git/RADAPI01/docs/operations/database-setup.md)
- [Deployment Guide](/mnt/ollama/git/RADAPI01/docs/operations/deployment-radgateway01.md)
- [System Health Check](/mnt/ollama/git/RADAPI01/scripts/system-health-check.sh)

---

## Maintenance Checklist

### Weekly Review

- [ ] Review maintenance logs for errors
- [ ] Check backup verification reports
- [ ] Verify SLO compliance
- [ ] Review alert frequency
- [ ] Check disk space usage

### Monthly Review

- [ ] Review all maintenance scripts for updates
- [ ] Analyze maintenance trends
- [ ] Update runbooks based on incidents
- [ ] Review and adjust thresholds
- [ ] Conduct maintenance drill

### Quarterly Review

- [ ] Full maintenance audit
- [ ] Capacity planning review
- [ ] Disaster recovery test
- [ ] Update escalation contacts
- [ ] Review and optimize schedules

---

## Change Log

| Date | Version | Changes |
|------|---------|---------|
| 2026-02-18 | 1.0 | Initial documentation created |
