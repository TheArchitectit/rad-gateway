# RAD Gateway Database Operations Runbook

**Version**: 1.0
**Last Updated**: 2026-02-18
**Team**: Team Golf (Documentation & Design)
**Classification**: Internal - Operations Team Only

---

## Table of Contents

1. [Overview](#overview)
2. [Severity Levels](#severity-levels)
3. [Incident Response Procedures](#incident-response-procedures)
4. [Recovery Procedures](#recovery-procedures)
5. [Failover Procedures](#failover-procedures)
6. [Escalation Matrix](#escalation-matrix)
7. [Troubleshooting Guide](#troubleshooting-guide)
8. [Appendices](#appendices)

---

## Overview

This runbook provides standardized procedures for database operations, incident response, and recovery for the RAD Gateway PostgreSQL database infrastructure. It covers:

- Database incident response
- Point-in-time recovery (PITR)
- Failover procedures
- Escalation paths
- Common troubleshooting scenarios

**Scope**: PostgreSQL databases supporting RAD Gateway production and staging environments.

**Target Audience**: Database Administrators, SREs, DevOps Engineers, and On-call Engineers.

---

## Severity Levels

### SEV-1: Critical - Database Unavailable

**Criteria**:
- Complete database outage
- Data corruption affecting production
- Inability to authenticate or process API requests
- Recovery requires immediate attention (RTO < 15 minutes)

**Impact**: Service completely unavailable, all API requests failing

**Examples**:
- PostgreSQL process crashed and won't restart
- Disk full causing database to stop accepting writes
- Corruption in critical system tables
- Primary database hardware failure

---

### SEV-2: High - Severe Degradation

**Criteria**:
- Database performance severely degraded (queries > 10s)
- Connection pool exhaustion
- Replication lag > 5 minutes
- One replica unavailable

**Impact**: Service degraded, high latency, potential for cascading failures

**Examples**:
- Query performance degradation affecting P95 latency
- Connection pool saturation (max connections reached)
- Replication lag causing stale reads
- Index corruption on non-critical tables

---

### SEV-3: Medium - Partial Impact

**Criteria**:
- Single replica lag > 1 minute but < 5 minutes
- Slow queries affecting specific features
- Backup failures (not yet affecting operations)
- Non-critical table issues

**Impact**: Limited impact, workarounds may exist

**Examples**:
- Slow query performance on reporting queries
- Backup job failures
- Minor replication lag
- Non-critical index bloat

---

### SEV-4: Low - Monitoring Required

**Criteria**:
- Warnings or anomalies detected
- Preventive maintenance needed
- Capacity planning concerns
- Minor performance degradation

**Impact**: No immediate user impact

**Examples**:
- Disk space utilization > 70%
- Connection utilization > 50%
- Table bloat > 20%
- Slow query log entries increasing

---

## Incident Response Procedures

### Immediate Response Steps (All Severities)

**Step 1: Acknowledge and Assess (0-2 minutes)**

1. Acknowledge the alert in PagerDuty/Opsgenie
2. Join the incident Slack channel: `#incidents-db`
3. Check current database health:

```bash
# Check if PostgreSQL is running
sudo systemctl status postgresql

# Check database connectivity
psql "postgresql://radgateway_user@localhost:5432/radgateway" -c "SELECT version(), now(), pg_is_in_recovery();"

# Check connection count
psql "postgresql://radgateway_user@localhost:5432/radgateway" -c "SELECT count(*) FROM pg_stat_activity;"

# Check database health endpoint
curl -s http://localhost:8090/health | jq .
```

**Step 2: Classify Severity (2-5 minutes)**

Use the severity criteria above to classify the incident. Update the incident ticket with:
- Severity level
- Initial assessment
- Impact scope (which environments, features affected)
- Error messages or logs

**Step 3: Notify Stakeholders**

| Severity | Notification | Response Time |
|----------|--------------|---------------|
| SEV-1 | Page on-call DBA + SRE Lead + Engineering Manager | Immediate |
| SEV-2 | Page on-call DBA + Notify team Slack | 5 minutes |
| SEV-3 | Create ticket + Slack notification | 30 minutes |
| SEV-4 | Create ticket for tracking | Next business day |

**Step 4: Containment**

For SEV-1/SEV-2:
- Enable maintenance mode if necessary: `curl -X POST http://localhost:8090/admin/maintenance -d '{"enabled":true}'`
- Redirect read traffic to replicas if available
- Stop non-critical background jobs

**Step 5: Diagnostic Data Collection**

```bash
# Create incident data directory
INCIDENT_DIR="/var/log/radgateway/incidents/$(date +%Y%m%d_%H%M%S)"
sudo mkdir -p "$INCIDENT_DIR"

# Collect PostgreSQL logs
sudo journalctl -u postgresql -n 1000 > "$INCIDENT_DIR/postgresql.log"

# Collect database stats
psql "postgresql://radgateway_user@localhost:5432/radgateway" << 'EOF' > "$INCIDENT_DIR/db_stats.txt"
SELECT now() as timestamp;
SELECT pg_size_pretty(pg_database_size('radgateway'));
SELECT count(*) as total_connections FROM pg_stat_activity;
SELECT count(*) as active_queries FROM pg_stat_activity WHERE state = 'active';
SELECT count(*) as waiting_queries FROM pg_stat_activity WHERE wait_event_type IS NOT NULL;
SELECT * FROM pg_stat_database WHERE datname = 'radgateway';
EOF

# Collect slow queries
psql "postgresql://radgateway_user@localhost:5432/radgateway" << 'EOF' > "$INCIDENT_DIR/slow_queries.txt"
SELECT pid, query_start, state, wait_event_type, left(query, 100) as query_preview
FROM pg_stat_activity
WHERE state != 'idle'
ORDER BY query_start ASC;
EOF

# Collect lock information
psql "postgresql://radgateway_user@localhost:5432/radgateway" << 'EOF' > "$INCIDENT_DIR/locks.txt"
SELECT l.locktype, l.relation, l.mode, l.granted, a.query
FROM pg_locks l
JOIN pg_stat_activity a ON l.pid = a.pid
WHERE NOT l.granted;
EOF
```

---

### SEV-1 Response: Database Unavailable

**Step 1: Verify Service Status**

```bash
# Check PostgreSQL service status
sudo systemctl status postgresql

# Check for port listening
sudo ss -tlnp | grep 5432

# Check disk space
df -h /var/lib/postgresql

# Check for OOM kills
sudo dmesg | grep -i "killed process" | tail -20
```

**Step 2: Attempt Service Recovery**

```bash
# Try to restart PostgreSQL
sudo systemctl restart postgresql

# Check logs for errors
sudo journalctl -u postgresql -n 100 --no-pager

# Verify startup
sleep 5
sudo systemctl status postgresql
```

**Step 3: If Service Won't Start**

Check common failure causes:

```bash
# Check PostgreSQL logs
sudo tail -100 /var/log/postgresql/postgresql-*.log

# Check for disk full
df -h

# Check for corrupt data (if in recovery mode)
sudo -u postgres pg_controldata /var/lib/postgresql/data

# Check configuration errors
sudo -u postgres pg_ctl -D /var/lib/postgresql/data check
```

**Step 4: Emergency Failover (If Primary Won't Recover)**

See [Failover Procedures](#failover-procedures) section.

**Step 5: Post-Recovery Verification**

```bash
# Verify database connectivity
psql "postgresql://radgateway_user@localhost:5432/radgateway" -c "SELECT 1;"

# Check table integrity
psql "postgresql://radgateway_user@localhost:5432/radgateway" -c "SELECT schemaname, tablename, n_tup_ins, n_tup_upd, n_tup_del FROM pg_stat_user_tables ORDER BY n_tup_ins DESC LIMIT 10;"

# Verify application connectivity
curl -s http://localhost:8090/health | jq .
```

---

### SEV-2 Response: Severe Degradation

**Step 1: Identify Root Cause**

```bash
# Check active queries
psql "postgresql://radgateway_user@localhost:5432/radgateway" << 'EOF'
SELECT pid, now() - query_start AS duration, state, left(query, 100)
FROM pg_stat_activity
WHERE state = 'active'
ORDER BY duration DESC
LIMIT 20;
EOF

# Check for blocking queries
psql "postgresql://radgateway_user@localhost:5432/radgateway" << 'EOF'
SELECT blocked_locks.pid AS blocked_pid,
       blocked_activity.usename AS blocked_user,
       blocking_locks.pid AS blocking_pid,
       blocking_activity.usename AS blocking_user,
       blocked_activity.query AS blocked_statement,
       blocking_activity.query AS blocking_statement
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks blocking_locks ON blocking_locks.locktype = blocked_locks.locktype
    AND blocking_locks.relation = blocked_locks.relation
    AND blocking_locks.pid != blocked_locks.pid
JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.granted;
EOF

# Check replication lag (if replica exists)
psql "postgresql://radgateway_user@localhost:5432/radgateway" -c "SELECT EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp())) AS lag_seconds;"
```

**Step 2: Mitigation Actions**

For connection pool exhaustion:
```bash
# Identify and terminate idle connections (use with caution!)
psql "postgresql://radgateway_user@localhost:5432/radgateway" << 'EOF'
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE state = 'idle'
  AND state_change < now() - interval '1 hour'
  AND usename = 'radgateway_user';
EOF
```

For slow queries:
```bash
# Identify and terminate runaway queries (use with caution!)
psql "postgresql://radgateway_user@localhost:5432/radgateway" << 'EOF'
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE state = 'active'
  AND now() - query_start > interval '10 minutes'
  AND query NOT LIKE '%pg_stat_activity%';
EOF
```

**Step 3: Monitor Recovery**

```bash
# Monitor active connections
watch -n 2 "psql -c 'SELECT count(*) FROM pg_stat_activity;'"

# Monitor query performance
watch -n 5 "psql -c 'SELECT count(*) FROM pg_stat_activity WHERE state = \\'active\\';'"
```

---

## Recovery Procedures

### Point-in-Time Recovery (PITR)

Point-in-time recovery allows restoring the database to a specific moment in time. This requires:
- Base backup
- Continuous WAL (Write-Ahead Log) archiving

**Prerequisites**:
- WAL archiving enabled
- Regular base backups
- Backup storage accessible

**Recovery Steps**:

**Step 1: Stop the Database**

```bash
# Stop RAD Gateway first
sudo systemctl stop radgateway01

# Stop PostgreSQL
sudo systemctl stop postgresql
```

**Step 2: Prepare Recovery Environment**

```bash
# Create recovery directory
sudo mkdir -p /var/lib/postgresql/recovery
sudo chown postgres:postgres /var/lib/postgresql/recovery

# Backup current data (in case we need to rollback)
sudo -u postgres pg_basebackup -D /var/lib/postgresql/data_backup_$(date +%Y%m%d_%H%M%S) -Ft -z -P
```

**Step 3: Restore Base Backup**

```bash
# Extract base backup
cd /var/lib/postgresql
sudo -u postgres tar -xzf /backup/postgresql/base/backup_YYYYMMDD.tar.gz -C recovery/

# Copy WAL files
sudo cp /backup/postgresql/wal/* /var/lib/postgresql/recovery/pg_wal/

# Set permissions
sudo chown -R postgres:postgres /var/lib/postgresql/recovery
```

**Step 4: Configure Recovery**

```bash
# Create recovery.signal file
sudo -u postgres touch /var/lib/postgresql/recovery/recovery.signal

# Create recovery.conf for PostgreSQL 12+ (via postgresql.conf modifications)
sudo -u postgres tee -a /var/lib/postgresql/recovery/postgresql.conf << 'EOF'
# Recovery settings
restore_command = 'cp /backup/postgresql/wal/%f %p'
recovery_target_time = '2026-02-18 14:30:00'  # Adjust to your target time
recovery_target_action = 'promote'
EOF
```

**Step 5: Start Recovery**

```bash
# Point PostgreSQL to recovery data
sudo mv /var/lib/postgresql/data /var/lib/postgresql/data_failed
sudo mv /var/lib/postgresql/recovery /var/lib/postgresql/data

# Start PostgreSQL
sudo systemctl start postgresql

# Monitor recovery progress
sudo -u postgres psql -c "SELECT pg_last_xact_replay_timestamp(), now(), pg_is_in_recovery();"
```

**Step 6: Verify Recovery**

```bash
# Check if recovery is complete
sudo -u postgres psql -c "SELECT pg_is_in_recovery();"  # Should return 'f' when done

# Verify data integrity
psql "postgresql://radgateway_user@localhost:5432/radgateway" -c "SELECT count(*) FROM usage_records;"

# Start RAD Gateway
sudo systemctl start radgateway01

# Verify application health
curl -s http://localhost:8090/health | jq .
```

**Recovery Time Objective (RTO)**:
- Base backup restore: 10-30 minutes (depends on data size)
- WAL replay: 5-15 minutes per GB of WAL
- Total expected RTO: 15-60 minutes

**Recovery Point Objective (RPO)**:
- With continuous archiving: < 15 minutes of data loss
- With streaming replication: < 1 minute of data loss

---

### Database Restore from Backup

For scenarios where PITR is not required (complete restore to backup time):

**Step 1: Stop Services**

```bash
sudo systemctl stop radgateway01
sudo systemctl stop postgresql
```

**Step 2: Restore from Latest Backup**

```bash
# List available backups
ls -lt /backup/postgresql/ | head -10

# Restore from specific backup
LATEST_BACKUP=$(ls -t /backup/postgresql/base/*.tar.gz | head -1)
sudo -u postgres pg_restore -d radgateway "$LATEST_BACKUP"

# Or for SQL dumps
sudo -u postgres psql radgateway < "$LATEST_BACKUP"
```

**Step 3: Verify and Start**

```bash
sudo systemctl start postgresql
sleep 5
sudo systemctl start radgateway01
curl -s http://localhost:8090/health
```

---

## Failover Procedures

### Primary to Replica Failover

**Scenario**: Primary database is unavailable, replica exists and is current.

**Pre-Failover Checks**:

```bash
# Check replica lag
psql "postgresql://replica_user@replica-host:5432/radgateway" -c "SELECT EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp())) AS lag_seconds;"

# Verify replica is in recovery mode
psql "postgresql://replica_user@replica-host:5432/radgateway" -c "SELECT pg_is_in_recovery();"  # Should return 't'

# Check replica data consistency (sample check)
psql "postgresql://replica_user@replica-host:5432/radgateway" -c "SELECT count(*) FROM usage_records WHERE created_at > now() - interval '1 hour';"
```

**Failover Steps**:

**Step 1: Disable Primary (if accessible)**

```bash
# If primary is accessible but failing, prevent new writes
psql "postgresql://radgateway_user@primary-host:5432/radgateway" -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE usename = 'radgateway_user';"
```

**Step 2: Promote Replica**

```bash
# On replica server, stop replication and promote to primary
sudo -u postgres pg_ctl promote -D /var/lib/postgresql/data

# Verify promotion
psql "postgresql://replica_user@replica-host:5432/radgateway" -c "SELECT pg_is_in_recovery();"  # Should return 'f'
```

**Step 3: Update RAD Gateway Configuration**

```bash
# Update connection string to point to new primary
sudo tee /opt/radgateway01/config/env << 'EOF'
RAD_DB_DRIVER=postgres
RAD_DB_DSN=postgresql://radgateway_user:password@NEW_PRIMARY_HOST:5432/radgateway?sslmode=require
EOF

# Restart RAD Gateway
sudo systemctl restart radgateway01
```

**Step 4: Verify Failover**

```bash
# Test write operations
psql "postgresql://radgateway_user@new-primary:5432/radgateway" -c "CREATE TABLE IF NOT EXISTS failover_test (id serial primary key, tested_at timestamp); INSERT INTO failover_test (tested_at) VALUES (now());"

# Verify application health
curl -s http://localhost:8090/health | jq .

# Check RAD Gateway logs
sudo journalctl -u radgateway01 -n 50
```

**Step 5: Update Monitoring and Alerting**

```bash
# Update Prometheus targets
# Update database health checks
# Update backup configurations to point to new primary
```

---

### Failback to Original Primary

After the original primary is recovered:

**Step 1: Reconfigure Original Primary as Replica**

```bash
# On original primary server
sudo systemctl stop postgresql

# Clean up data directory and clone from new primary
sudo rm -rf /var/lib/postgresql/data/*
sudo -u postgres pg_basebackup -h NEW_PRIMARY_HOST -D /var/lib/postgresql/data -U replication_user -P -v -R

# Start as replica
sudo systemctl start postgresql
```

**Step 2: Verify Replication**

```bash
# Check replica status
psql "postgresql://radgateway_user@original-primary:5432/radgateway" -c "SELECT pg_is_in_recovery(), pg_last_xact_replay_timestamp();"
```

**Step 3: Planned Failover Back (Optional)**

```bash
# During maintenance window, promote original primary back
sudo -u postgres pg_ctl promote -D /var/lib/postgresql/data

# Update RAD Gateway configuration back to original primary
# Update connection strings
# Restart RAD Gateway
```

---

## Escalation Matrix

### Severity-Based Escalation

| Severity | First Response | Escalation Time | Escalation Path |
|----------|---------------|-----------------|-----------------|
| SEV-1 | Immediate | 15 min | DBA Team Lead -> VP Engineering -> CTO |
| SEV-2 | 5 minutes | 30 min | On-call DBA -> DBA Team Lead |
| SEV-3 | 30 minutes | 2 hours | Ticket assignment -> Team Lead review |
| SEV-4 | Next business day | 24 hours | Ticket queue -> Scheduled work |

### Contact Information

| Role | Primary Contact | Escalation Contact |
|------|-----------------|-------------------|
| On-call DBA | PagerDuty rotation | - |
| DBA Team Lead | dba-lead@radgateway.internal | +1-555-DBA-LEAD |
| SRE Team Lead | sre-lead@radgateway.internal | +1-555-SRE-LEAD |
| Engineering Manager | eng-mgr@radgateway.internal | +1-555-ENG-MGR |
| VP Engineering | vp-eng@radgateway.internal | +1-555-VP-ENG |
| CTO | cto@radgateway.internal | +1-555-CTO |

### Communication Channels

| Severity | Primary Channel | Secondary Channel |
|----------|-------------------|-------------------|
| SEV-1 | #incidents-critical (Slack) + Page | Conference bridge |
| SEV-2 | #incidents-db (Slack) + Page | Direct Slack DM |
| SEV-3 | #database-alerts (Slack) | Jira ticket |
| SEV-4 | Jira ticket | Email notification |

### External Escalation

For AWS RDS / Cloud SQL managed databases:

| Provider | Support Channel | Emergency Contact |
|----------|-----------------|-------------------|
| AWS RDS | AWS Support Console | AWS Enterprise Support |
| Google Cloud SQL | Cloud Console | GCP Premium Support |
| Azure Database | Azure Portal | Azure ProDirect Support |

---

## Troubleshooting Guide

### Connection Issues

**Symptom**: "connection refused" or "too many connections"

**Diagnostic Steps**:

```bash
# Check PostgreSQL is listening
sudo ss -tlnp | grep 5432

# Check max_connections
psql "postgresql://radgateway_user@localhost:5432/radgateway" -c "SHOW max_connections;"

# Check current connections
psql "postgresql://radgateway_user@localhost:5432/radgateway" << 'EOF'
SELECT state, count(*)
FROM pg_stat_activity
GROUP BY state;
EOF

# Check connection limits per user
psql "postgresql://radgateway_user@localhost:5432/radgateway" -c "\du+"
```

**Solutions**:

1. **Restart Application Connection Pool**:
   ```bash
   sudo systemctl restart radgateway01
   ```

2. **Terminate Idle Connections**:
   ```sql
   SELECT pg_terminate_backend(pid)
   FROM pg_stat_activity
   WHERE state = 'idle'
     AND state_change < now() - interval '30 minutes';
   ```

3. **Increase max_connections** (requires restart):
   ```bash
   sudo -u postgres psql -c "ALTER SYSTEM SET max_connections = 200;"
   sudo systemctl restart postgresql
   ```

---

### Disk Space Issues

**Symptom**: "could not write to file", "No space left on device"

**Diagnostic Steps**:

```bash
# Check disk usage
df -h /var/lib/postgresql

# Check database size
psql "postgresql://radgateway_user@localhost:5432/radgateway" -c "SELECT pg_size_pretty(pg_database_size('radgateway'));"

# Find largest tables
psql "postgresql://radgateway_user@localhost:5432/radgateway" << 'EOF'
SELECT schemaname, tablename,
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
LIMIT 10;
EOF

# Check WAL directory size
sudo du -sh /var/lib/postgresql/data/pg_wal
```

**Solutions**:

1. **Clean Up Old WAL Files**:
   ```bash
   # Force WAL checkpoint
   psql "postgresql://radgateway_user@localhost:5432/radgateway" -c "CHECKPOINT;"

   # Archive/compress old WAL files
   sudo find /var/lib/postgresql/data/pg_wal -name "*.backup" -mtime +7 -delete
   ```

2. **Clean Up Usage Records** (if retention policy allows):
   ```sql
   -- Archive old data first!
   DELETE FROM usage_records WHERE created_at < now() - interval '90 days';
   VACUUM usage_records;
   ```

3. **Expand Storage**:
   ```bash
   # For cloud environments, expand the volume
   # Then resize the filesystem
   sudo resize2fs /dev/datavg/postgres
   ```

---

### Performance Issues

**Symptom**: Slow queries, high CPU, high memory usage

**Diagnostic Steps**:

```bash
# Check long-running queries
psql "postgresql://radgateway_user@localhost:5432/radgateway" << 'EOF'
SELECT pid, now() - query_start AS duration, state, query
FROM pg_stat_activity
WHERE state = 'active'
ORDER BY duration DESC
LIMIT 10;
EOF

# Check table statistics
psql "postgresql://radgateway_user@localhost:5432/radgateway" << 'EOF'
SELECT schemaname, tablename, n_tup_ins, n_tup_upd, n_tup_del,
       n_live_tup, n_dead_tup, last_vacuum, last_autovacuum
FROM pg_stat_user_tables
ORDER BY n_dead_tup DESC
LIMIT 10;
EOF

# Check index usage
psql "postgresql://radgateway_user@localhost:5432/radgateway" << 'EOF'
SELECT schemaname, tablename, indexrelname, idx_scan, idx_tup_read
FROM pg_stat_user_indexes
ORDER BY idx_scan ASC
LIMIT 20;
EOF
```

**Solutions**:

1. **Analyze Tables**:
   ```sql
   ANALYZE usage_records;
   ANALYZE trace_events;
   ```

2. **Vacuum Tables**:
   ```sql
   VACUUM ANALYZE usage_records;
   ```

3. **Rebuild Indexes**:
   ```sql
   REINDEX INDEX CONCURRENTLY idx_usage_workspace;
   ```

4. **Add Missing Indexes** (after analysis):
   ```sql
   -- Example: Add index for common query pattern
   CREATE INDEX CONCURRENTLY idx_usage_created_at_desc ON usage_records(created_at DESC);
   ```

---

### Replication Issues

**Symptom**: Replication lag, data inconsistency between primary and replica

**Diagnostic Steps**:

```bash
# On primary: Check replication status
psql "postgresql://radgateway_user@primary:5432/radgateway" << 'EOF'
SELECT client_addr, state, sent_lsn, write_lsn, flush_lsn, replay_lsn,
       write_lag, flush_lag, replay_lag
FROM pg_stat_replication;
EOF

# On replica: Check recovery status
psql "postgresql://radgateway_user@replica:5432/radgateway" << 'EOF'
SELECT pg_is_in_recovery(),
       pg_last_xact_replay_timestamp(),
       now() - pg_last_xact_replay_timestamp() AS lag;
EOF

# Check replication slot status
psql "postgresql://radgateway_user@primary:5432/radgateway" -c "SELECT slot_name, active, restart_lsn, confirmed_flush_lsn FROM pg_replication_slots;"
```

**Solutions**:

1. **Restart Replication** (if slot is inactive):
   ```bash
   # On replica
   sudo systemctl restart postgresql
   ```

2. **Recreate Replication Slot** (if corrupted):
   ```sql
   -- On primary
   SELECT pg_drop_replication_slot('replica_slot');
   SELECT pg_create_physical_replication_slot('replica_slot');
   ```

3. **Resync Replica** (if data diverged):
   ```bash
   # Stop replica
   sudo systemctl stop postgresql

   # Reclone from primary
   sudo rm -rf /var/lib/postgresql/data/*
   sudo -u postgres pg_basebackup -h primary-host -D /var/lib/postgresql/data -U replication_user -P -v -R

   # Start replica
   sudo systemctl start postgresql
   ```

---

### Backup Failures

**Symptom**: Automated backup jobs failing, missing recent backups

**Diagnostic Steps**:

```bash
# Check backup logs
sudo tail -100 /var/log/postgresql/backup.log

# Verify backup disk space
df -h /backup

# Test backup manually
sudo -u postgres pg_dump radgateway > /tmp/test_backup.sql
ls -lh /tmp/test_backup.sql

# Verify WAL archiving
ls -lt /backup/postgresql/wal/ | head -10
```

**Solutions**:

1. **Fix Backup Permissions**:
   ```bash
   sudo chown -R postgres:postgres /backup/postgresql
   sudo chmod 755 /backup/postgresql
   ```

2. **Clean Up Old Backups**:
   ```bash
   # Keep last 30 days of backups
   sudo find /backup/postgresql/base -name "*.tar.gz" -mtime +30 -delete
   sudo find /backup/postgresql/wal -name "*.gz" -mtime +7 -delete
   ```

3. **Repair Backup Schedule**:
   ```bash
   # Check cron jobs
   sudo crontab -l -u postgres

   # Re-enable backup script
   echo "0 2 * * * /usr/local/bin/pg-backup.sh" | sudo crontab -u postgres -
   ```

---

## Appendices

### Appendix A: Quick Reference Commands

```bash
# Check database status
sudo systemctl status postgresql

# Connect to database
psql "postgresql://radgateway_user@localhost:5432/radgateway"

# Check active queries
psql -c "SELECT pid, state, query FROM pg_stat_activity WHERE state = 'active';"

# Check locks
psql -c "SELECT * FROM pg_locks WHERE NOT granted;"

# Check table sizes
psql -c "\dt+"

# Check database size
psql -c "SELECT pg_size_pretty(pg_database_size('radgateway'));"

# Check replication status
psql -c "SELECT pg_is_in_recovery(), pg_last_xact_replay_timestamp();"

# Restart PostgreSQL
sudo systemctl restart postgresql

# Reload configuration
sudo systemctl reload postgresql

# View logs
sudo journalctl -u postgresql -f
```

### Appendix B: Important File Locations

| File/Directory | Purpose |
|----------------|---------|
| `/var/lib/postgresql/data` | PostgreSQL data directory |
| `/var/lib/postgresql/data/pg_wal` | Write-ahead log files |
| `/var/log/postgresql/` | PostgreSQL logs |
| `/backup/postgresql/` | Backup storage |
| `/etc/postgresql/15/main/postgresql.conf` | Main configuration |
| `/etc/postgresql/15/main/pg_hba.conf` | Client authentication |
| `/opt/radgateway01/config/env` | RAD Gateway DB connection |

### Appendix C: Monitoring Checklist

**Daily**:
- [ ] Check backup completion status
- [ ] Review slow query log
- [ ] Verify replication lag (if applicable)

**Weekly**:
- [ ] Review table bloat
- [ ] Check index usage
- [ ] Analyze top queries
- [ ] Review disk space trends

**Monthly**:
- [ ] Test backup restore procedure
- [ ] Review user access and permissions
- [ ] Update documentation
- [ ] Capacity planning review

### Appendix D: Change Management

**Database Schema Changes**:
1. Create migration in `/migrations/`
2. Test on staging environment
3. Schedule maintenance window for production
4. Create backup before migration
5. Apply migration with monitoring
6. Verify application functionality
7. Document changes in incident log

**Configuration Changes**:
1. Document current configuration
2. Test changes in staging
3. Prepare rollback procedure
4. Apply during maintenance window
5. Monitor for issues
6. Document outcome

---

**Document Owners**: Team Golf (Documentation & Design)
**Review Schedule**: Quarterly or after any major incident
**Next Review Date**: 2026-05-18
