# Beta Deployment Architecture

**Status**: Current (v0.2.0-alpha)  
**Target**: Beta Release  
**Last Updated**: 2026-02-16

---

## Overview

The beta deployment uses a simplified secrets management approach with **Infisical only**. OpenBao is deployed but reserved for post-beta cold vault requirements.

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      BETA STACK                          │
│                                                          │
│  ┌──────────────┐    ┌──────────────┐    ┌─────────────┐ │
│  │  RAD Gateway │───▶│  Infisical   │───▶│  PostgreSQL │ │
│  │   :8090      │    │   :8080      │    │   :5432     │ │
│  └──────────────┘    └──────────────┘    └─────────────┘ │
│                              │                           │
│                              ▼                           │
│                       ┌──────────────┐                   │
│                       │ Secrets Mgmt │                   │
│                       │ - API Keys   │                   │
│                       │ - DB Creds   │                   │
│                       │ - Provider   │                   │
│                       │   Tokens     │                   │
│                       └──────────────┘                   │
└─────────────────────────────────────────────────────────┘
```

---

## Services

| Service | Purpose | Port | Status |
|---------|---------|------|--------|
| **radgateway01** | API Gateway | 8090 | ✅ Active |
| **Infisical** | Secrets Management | 8080 | ✅ Active |
| **PostgreSQL** | Database | 5432 | ✅ Active |
| **OpenBao** | Cold Vault (Future) | 8200 | ⚠️ Reserved |
| **Redis** | Cache | 6379 | ✅ Active |

---

## Secrets Management (Beta)

### Infisical Handles:
- ✅ Provider API keys (OpenAI, Anthropic, Gemini)
- ✅ PostgreSQL credentials
- ✅ JWT secrets
- ✅ Encryption keys
- ✅ Service tokens

### OpenBao Reserved For:
- ⏸️ Cold vault archival (5+ year retention)
- ⏸️ Compliance audit trails
- ⏸️ Advanced PKI features
- ⏸️ Post-beta requirements

---

## Access URLs

| Service | URL | Purpose |
|---------|-----|---------|
| RAD Gateway Health | http://172.16.30.45:8090/health | Gateway status |
| Infisical UI | http://172.16.30.45:8080 | Secrets management |
| OpenBao UI | http://172.16.30.45:8200 | **Reserved** |

---

## Configuration

### RAD Gateway Secrets Path in Infisical
```
/rad-gateway/
├── providers/
│   ├── openai/
│   │   └── api-key
│   ├── anthropic/
│   │   └── api-key
│   └── gemini/
│       └── api-key
├── database/
│   └── postgres-url
└── gateway/
    ├── jwt-secret
    └── encryption-key
```

---

## Backup Procedures

### Overview
Beta deployments use automated daily backups to protect configuration and data. Backups are stored locally on the deployment host with a 7-day retention policy. PostgreSQL (used by Infisical) and OpenBao data (if initialized) require separate backup procedures.

### Backup Schedule
- **Frequency**: Daily at 02:00 via cron
- **Retention**: 7 days
- **Location**: `/backup/radgateway01/`
- **Format**: Compressed tar.gz with timestamp

### What Gets Backed Up
- RAD Gateway configuration files (`/opt/radgateway01/config/`)
- RAD Gateway data files (`/opt/radgateway01/data/`)
- Backup manifest with file listing
- SHA256 checksums for integrity verification

### Manual Backup

```bash
# Run backup manually
sudo /mnt/ollama/git/RADAPI01/deploy/bin/backup.sh

# Or with custom retention (e.g., 14 days)
sudo BACKUP_RETENTION_DAYS=14 /mnt/ollama/git/RADAPI01/deploy/bin/backup.sh

# Backup output will show the archive location:
# BACKUP_FILE=/backup/radgateway01/20260216_143022.tar.gz
```

### Restore Procedure

```bash
# 1. Stop services
sudo systemctl stop radgateway01

# 2. Identify backup to restore
BACKUP_FILE="/backup/radgateway01/20260216_143022.tar.gz"

# 3. Extract backup
cd /tmp
sudo tar -xzf "$BACKUP_FILE"

# 4. Verify checksums (optional but recommended)
cd /tmp/$(basename "$BACKUP_FILE" .tar.gz)
sha256sum -c checksums.sha256

# 5. Restore configuration
sudo rm -rf /opt/radgateway01/config
sudo cp -r config /opt/radgateway01/

# 6. Restore data (if applicable)
sudo rm -rf /opt/radgateway01/data
sudo cp -r data /opt/radgateway01/

# 7. Set proper ownership
sudo chown -R radgateway01:radgateway01 /opt/radgateway01/

# 8. Start services
sudo systemctl start radgateway01

# 9. Verify restoration
curl http://172.16.30.45:8090/health
```

### PostgreSQL Backup (Infisical Database)

Since PostgreSQL is used by Infisical, back it up separately:

```bash
# Manual PostgreSQL backup
pg_dump -h localhost -U infisical infisical > /backup/radgateway01/infisical_$(date +%Y%m%d).sql

# Restore PostgreSQL
dropdb -h localhost -U infisical infisical 2>/dev/null || true
createdb -h localhost -U infisical infisical
psql -h localhost -U infisical infisical < /backup/radgateway01/infisical_20260216.sql
```

### OpenBao Backup (If Initialized)

OpenBao is reserved for post-beta use. If initialized:

```bash
# Backup OpenBao (requires VAULT_ADDR and VAULT_TOKEN)
export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_TOKEN="your-token"
vault operator raft snapshot save /backup/radgateway01/openbao_$(date +%Y%m%d).snap

# Restore OpenBao
vault operator raft snapshot restore /backup/radgateway01/openbao_20260216.snap
```

### Exclusions

The following are **NOT** backed up:
- **Infisical token file**: `/opt/radgateway01/config/infisical-token` (must be recreated manually)
  - **Security Note**: This bootstrap token is intentionally excluded from backups as it is a sensitive credential that should not be persisted in backup archives. The token can be regenerated from Infisical if needed.
- **Runtime logs**: `/opt/radgateway01/logs/`
- **Temporary files**: Cache and temp directories
- **Secrets**: Provider API keys must be reconfigured from Infisical

### Verification

Weekly backup verification is recommended for beta:

```bash
# List available backups
ls -la /backup/radgateway01/*.tar.gz

# Verify a specific backup without extracting
tar -tzf /backup/radgateway01/20260216_143022.tar.gz | head -20

# Check backup integrity
tar -tzf /backup/radgateway01/20260216_143022.tar.gz >/dev/null && echo "Backup valid" || echo "Backup corrupt"
```

### Cron Configuration

Add to `/etc/cron.d/radgateway01` for automated backups:

```
# RAD Gateway Backup - Daily at 02:00
0 2 * * * root /mnt/ollama/git/RADAPI01/deploy/bin/backup.sh >> /var/log/radgateway01-backup.log 2>&1
```

---

## Monitoring and Alerting

This section provides basic monitoring setup for the beta deployment. For comprehensive monitoring procedures, see [Monitoring Procedures](../operations/monitoring-procedures.md).

### Health Check Endpoints

| Endpoint | URL | Purpose |
|----------|-----|---------|
| Health | `http://172.16.30.45:8090/health` | Service health status |
| Metrics | `http://172.16.30.45:8090/metrics` | Prometheus-compatible metrics |

### Basic Health Check

```bash
# Quick health check
curl -s http://172.16.30.45:8090/health

# Expected response: {"status":"healthy","version":"0.2.0-alpha"}
```

### Alert Thresholds (Beta)

| Metric | Warning | Critical | Action |
|--------|---------|----------|--------|
| Service Down | - | Health check fails 2+ min | Restart service |
| Error Rate | > 2% for 10 min | > 5% for 5 min | Check provider connectivity |
| P95 Latency | > 1200ms | > 2000ms | Check resource usage |
| Disk Usage | > 80% | > 90% | Clean up logs/backups |

### Simple Monitoring Setup

**Option 1: Cron-based Health Check**

Add to crontab for basic monitoring:
```bash
# Check every 5 minutes, restart if down
*/5 * * * * /opt/radgateway01/bin/health-check.sh || sudo systemctl restart radgateway01
```

**Option 2: Uptime Kuma (Recommended for Beta)**

- Monitor Type: HTTP(s)
- URL: `http://172.16.30.45:8090/health`
- Heartbeat Interval: 60 seconds
- Retries: 3

**Option 3: Prometheus (Optional)**

```yaml
scrape_configs:
  - job_name: 'radgateway01'
    static_configs:
      - targets: ['172.16.30.45:8090']
    metrics_path: /metrics
    scrape_interval: 15s
```

### Log Monitoring

```bash
# Watch logs in real-time
sudo journalctl -u radgateway01 -f

# Check for errors in last hour
sudo journalctl -u radgateway01 --since "1 hour ago" | grep -i error
```

---

## Deployment and Rollback Procedures

### Deploying a New Version

**Prerequisites:**
- Backup current deployment (see Backup Procedures above)
- New binary available at `/usr/local/bin/rad-gateway`

**Deployment Steps:**

```bash
# 1. Create pre-deployment backup
sudo /mnt/ollama/git/RADAPI01/deploy/bin/backup.sh

# 2. Stop current service
sudo systemctl stop radgateway01

# 3. Backup current binary (quick rollback)
sudo cp /usr/local/bin/rad-gateway /usr/local/bin/rad-gateway.backup.$(date +%Y%m%d_%H%M%S)

# 4. Deploy new binary
sudo cp /path/to/new/rad-gateway /usr/local/bin/rad-gateway
sudo chmod +x /usr/local/bin/rad-gateway

# 5. Start service
sudo systemctl start radgateway01

# 6. Verify deployment
curl -s http://172.16.30.45:8090/health
sleep 5
sudo systemctl status radgateway01
```

**Smoke Test After Deploy:**

```bash
# Test basic API functionality
curl -X POST http://172.16.30.45:8090/v1/chat/completions \
  -H "Authorization: Bearer $TEST_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"test"}]}'
```

### Rollback Procedure

**When to Rollback:**
- Service fails health checks after deployment
- Error rate exceeds 5%
- Critical functionality broken

**Quick Rollback Steps:**

```bash
# 1. Stop current service
sudo systemctl stop radgateway01

# 2. Restore previous binary
# Find the most recent backup:
ls -t /usr/local/bin/rad-gateway.backup.* | head -1

# Restore it:
sudo cp /usr/local/bin/rad-gateway.backup.20260216_143022 /usr/local/bin/rad-gateway

# 3. Start service
sudo systemctl start radgateway01

# 4. Verify rollback
curl -s http://172.16.30.45:8090/health
sudo systemctl status radgateway01
```

**Full Restore (if binary rollback insufficient):**

```bash
# 1. Stop service
sudo systemctl stop radgateway01

# 2. Restore from backup (see Backup Procedures section)
# Follow the restore procedure using the pre-deployment backup

# 3. Start service
sudo systemctl start radgateway01
```

### Beta Deployment Notes

- **Keep it simple**: Beta deployments prioritize speed over sophistication
- **Test in dev first**: Always test new versions in development environment
- **Have a backup**: Always create backup before deploying
- **Monitor after deploy**: Watch logs and metrics for 30 minutes post-deployment
- **Document changes**: Note what changed in case rollback is needed

---

## Notes

- OpenBao is **not configured** for active use in beta
- All secrets flow through Infisical only
- OpenBao can be enabled post-beta for compliance requirements
- This keeps beta deployment simple and maintainable

See: [Golden Stack Documentation](../operations/golden-stack.md) for full deployment details.
