# Golden Stack Operations Guide

**Status**: Production Ready
**Owner**: Team Hotel (Deployment & Infrastructure)
**Target Environment**: radgateway01 (Alpha)
**Last Updated**: 2026-02-16

---

## Daily Operations

### Morning Health Check

Run this checklist at the start of each shift:

```bash
#!/bin/bash
# /opt/secret-stack/scripts/daily-health-check.sh

echo "=== Golden Stack Daily Health Check ==="
echo "Date: $(date)"
echo ""

# 1. Check PostgreSQL
echo "[1/5] PostgreSQL Health..."
if sudo podman exec secret-stack-postgres pg_isready -U postgres > /dev/null 2>&1; then
    echo "  [OK] PostgreSQL is accepting connections"
else
    echo "  [FAIL] PostgreSQL is not responding"
    exit 1
fi

# 2. Check OpenBao
echo "[2/5] OpenBao Health..."
BAO_HEALTH=$(curl -s http://localhost:8200/v1/sys/health 2>/dev/null)
if [ $? -eq 0 ]; then
    SEALED=$(echo $BAO_HEALTH | jq -r '.sealed')
    if [ "$SEALED" = "false" ]; then
        echo "  [OK] OpenBao is unsealed and healthy"
    else
        echo "  [FAIL] OpenBao is SEALED - requires unseal operation"
    fi
else
    echo "  [FAIL] OpenBao is not responding"
fi

# 3. Check Infisical
echo "[3/5] Infisical Health..."
INF_STATUS=$(curl -s http://localhost:8080/api/status 2>/dev/null)
if [ $? -eq 0 ]; then
    echo "  [OK] Infisical is responding"
else
    echo "  [FAIL] Infisical is not responding"
fi

# 4. Check disk space
echo "[4/5] Disk Usage..."
DISK_USAGE=$(df /opt/secret-stack | tail -1 | awk '{print $5}' | tr -d '%')
if [ "$DISK_USAGE" -lt 80 ]; then
    echo "  [OK] Disk usage: ${DISK_USAGE}%"
else
    echo "  [WARN] Disk usage high: ${DISK_USAGE}%"
fi

# 5. Check memory
echo "[5/5] Memory Usage..."
MEM_USAGE=$(free | grep Mem | awk '{printf "%.0f", $3/$2 * 100.0}')
if [ "$MEM_USAGE" -lt 90 ]; then
    echo "  [OK] Memory usage: ${MEM_USAGE}%"
else
    echo "  [WARN] Memory usage high: ${MEM_USAGE}%"
fi

echo ""
echo "=== Health Check Complete ==="
```

### Quick Status Commands

```bash
# Full stack status
sudo podman pod ps
sudo podman ps --pod

# Individual service status
curl -s http://localhost:8200/v1/sys/health | jq
curl -s http://localhost:8080/api/status | jq
sudo podman exec secret-stack-postgres pg_isready -U postgres

# View logs (last 50 lines)
sudo podman logs secret-stack-postgres --tail 50
sudo podman logs secret-stack-openbao --tail 50
sudo podman logs secret-stack-infisical --tail 50
```

---

## Backup Procedures

### Automated Backup Schedule

| Component | Type | Frequency | Retention | Location |
|-----------|------|-----------|-----------|----------|
| PostgreSQL | Full dump | Daily at 02:00 | 30 days | /backup/secret-stack/postgresql |
| PostgreSQL | Incremental | Hourly | 7 days | /backup/secret-stack/postgresql |
| OpenBao | Vault export | Weekly | 90 days | /backup/secret-stack/openbao |
| Infisical | Config export | Daily | 30 days | /backup/secret-stack/infisical |

### PostgreSQL Backup Script

```bash
#!/bin/bash
# /opt/secret-stack/scripts/backup-postgres.sh

set -e

BACKUP_DIR="/backup/secret-stack/postgresql"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

echo "[backup] Starting PostgreSQL backup at ${DATE}"

# Create backup directory
mkdir -p "${BACKUP_DIR}"

# Full backup of all databases
echo "[backup] Creating full database dump..."
sudo podman exec secret-stack-postgres pg_dumpall -U postgres | \
    gzip > "${BACKUP_DIR}/postgres-full-${DATE}.sql.gz"

# Individual database backups for selective restore
for DB in infisical_db openbao_db; do
    echo "[backup] Backing up ${DB}..."
    sudo podman exec secret-stack-postgres pg_dump -U postgres -d "${DB}" | \
        gzip > "${BACKUP_DIR}/${DB}-${DATE}.sql.gz"
done

# Cleanup old backups
echo "[backup] Cleaning up backups older than ${RETENTION_DAYS} days..."
find "${BACKUP_DIR}" -name "*.sql.gz" -mtime +${RETENTION_DAYS} -delete

# Verify backup integrity
echo "[backup] Verifying backup..."
if [ -f "${BACKUP_DIR}/postgres-full-${DATE}.sql.gz" ]; then
    SIZE=$(du -h "${BACKUP_DIR}/postgres-full-${DATE}.sql.gz" | cut -f1)
    echo "[backup] Backup completed successfully: ${SIZE}"
else
    echo "[backup] ERROR: Backup file not created" >&2
    exit 1
fi
```

### OpenBao Backup Script

```bash
#!/bin/bash
# /opt/secret-stack/scripts/backup-openbao.sh

set -e

BACKUP_DIR="/backup/secret-stack/openbao"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=90
VAULT_ADDR="http://localhost:8200"
VAULT_TOKEN_FILE="/opt/secret-stack/config/openbao-root-token"

if [ ! -f "$VAULT_TOKEN_FILE" ]; then
    echo "[backup] ERROR: Root token file not found" >&2
    exit 1
fi

VAULT_TOKEN=$(cat "$VAULT_TOKEN_FILE")

echo "[backup] Starting OpenBao backup at ${DATE}"

# Create backup directory
mkdir -p "${BACKUP_DIR}/${DATE}"

# Export secrets from KV stores
export VAULT_ADDR VAULT_TOKEN

echo "[backup] Exporting KV secrets..."
vault kv list -format=json infisical-archive/ > \
    "${BACKUP_DIR}/${DATE}/kv-paths.json" 2>/dev/null || true

# Backup audit logs
echo "[backup] Copying audit logs..."
if [ -d "/opt/secret-stack/logs/openbao" ]; then
    cp /opt/secret-stack/logs/openbao/audit.log.* "${BACKUP_DIR}/${DATE}/" 2>/dev/null || true
fi

# Backup configuration
echo "[backup] Exporting configuration..."
vault read -format=json sys/config/state > \
    "${BACKUP_DIR}/${DATE}/sys-config.json" 2>/dev/null || true

# Create tarball
echo "[backup] Creating archive..."
(cd "${BACKUP_DIR}" && tar czf "openbao-backup-${DATE}.tar.gz" "${DATE}")
rm -rf "${BACKUP_DIR}/${DATE}"

# Cleanup old backups
echo "[backup] Cleaning up backups older than ${RETENTION_DAYS} days..."
find "${BACKUP_DIR}" -name "openbao-backup-*.tar.gz" -mtime +${RETENTION_DAYS} -delete

echo "[backup] OpenBao backup completed: openbao-backup-${DATE}.tar.gz"
```

### Manual Backup Procedure

```bash
# Run immediate backup
sudo /opt/secret-stack/scripts/backup-postgres.sh
sudo /opt/secret-stack/scripts/backup-openbao.sh

# Verify backups
ls -la /backup/secret-stack/postgresql/
ls -la /backup/secret-stack/openbao/
```

### Recovery Procedures

#### PostgreSQL Restore

```bash
#!/bin/bash
# /opt/secret-stack/scripts/restore-postgres.sh
# Usage: ./restore-postgres.sh <backup-file>

BACKUP_FILE="$1"

if [ ! -f "$BACKUP_FILE" ]; then
    echo "[restore] ERROR: Backup file not found: $BACKUP_FILE" >&2
    exit 1
fi

echo "[restore] WARNING: This will overwrite existing databases"
read -p "Are you sure? Type 'yes' to continue: " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo "[restore] Aborted"
    exit 1
fi

# Stop dependent services
echo "[restore] Stopping services..."
sudo systemctl stop secret-stack-infisical 2>/dev/null || true
sudo systemctl stop secret-stack-openbao 2>/dev/null || true

# Restore database
echo "[restore] Restoring from ${BACKUP_FILE}..."
zcat "$BACKUP_FILE" | sudo podman exec -i secret-stack-postgres psql -U postgres

# Restart services
echo "[restore] Restarting services..."
sudo systemctl start secret-stack-infisical
sudo systemctl start secret-stack-openbao

echo "[restore] Restore complete. Verify services are running:"
echo "  curl http://localhost:8080/api/status"
echo "  curl http://localhost:8200/v1/sys/health"
```

#### OpenBao Unseal After Restore

```bash
# If OpenBao is sealed after restore, unseal it:
VAULT_ADDR="http://localhost:8200"

# Check seal status
curl -s "$VAULT_ADDR/v1/sys/seal-status" | jq

# Unseal with Shamir shards (need 3 of 5)
# Run this 3 times with different shards
curl -X PUT "$VAULT_ADDR/v1/sys/unseal" \
  -H "Content-Type: application/json" \
  -d '{"key": "SHARD_1_HERE"}'

curl -X PUT "$VAULT_ADDR/v1/sys/unseal" \
  -H "Content-Type: application/json" \
  -d '{"key": "SHARD_2_HERE"}'

curl -X PUT "$VAULT_ADDR/v1/sys/unseal" \
  -H "Content-Type: application/json" \
  -d '{"key": "SHARD_3_HERE"}'
```

---

## Rotation Procedures

### API Key Rotation

```bash
#!/bin/bash
# /opt/secret-stack/scripts/rotate-api-keys.sh
# Rotates provider API keys stored in Infisical

INFISICAL_URL="http://localhost:8080"
WORKSPACE_ID="YOUR_WORKSPACE_ID"
INFISICAL_TOKEN="YOUR_SERVICE_TOKEN"

echo "[rotation] Starting API key rotation"

# 1. Generate new OpenAI key (manual step - get from OpenAI dashboard)
echo "[rotation] 1. Generate new OpenAI API key from: https://platform.openai.com/api-keys"
read -p "Enter new OpenAI key (or 'skip'): " NEW_OPENAI_KEY

if [ "$NEW_OPENAI_KEY" != "skip" ] && [ -n "$NEW_OPENAI_KEY" ]; then
    # Update in Infisical
    curl -s -X PATCH "${INFISICAL_URL}/api/v3/secrets/raw/OPENAI_API_KEY" \
        -H "Authorization: Bearer ${INFISICAL_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{
            \"workspaceId\": \"${WORKSPACE_ID}\",
            \"environment\": \"alpha\",
            \"secretValue\": \"${NEW_OPENAI_KEY}\"
        }"
    echo "[rotation] OpenAI key updated"
fi

# 2. Generate new Anthropic key
echo "[rotation] 2. Generate new Anthropic API key from: https://console.anthropic.com/"
read -p "Enter new Anthropic key (or 'skip'): " NEW_ANTHROPIC_KEY

if [ "$NEW_ANTHROPIC_KEY" != "skip" ] && [ -n "$NEW_ANTHROPIC_KEY" ]; then
    curl -s -X PATCH "${INFISICAL_URL}/api/v3/secrets/raw/ANTHROPIC_API_KEY" \
        -H "Authorization: Bearer ${INFISICAL_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{
            \"workspaceId\": \"${WORKSPACE_ID}\",
            \"environment\": \"alpha\",
            \"secretValue\": \"${NEW_ANTHROPIC_KEY}\"
        }"
    echo "[rotation] Anthropic key updated"
fi

# 3. Generate new Gemini key
echo "[rotation] 3. Generate new Gemini API key from: https://aistudio.google.com/app/apikey"
read -p "Enter new Gemini key (or 'skip'): " NEW_GEMINI_KEY

if [ "$NEW_GEMINI_KEY" != "skip" ] && [ -n "$NEW_GEMINI_KEY" ]; then
    curl -s -X PATCH "${INFISICAL_URL}/api/v3/secrets/raw/GEMINI_API_KEY" \
        -H "Authorization: Bearer ${INFISICAL_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{
            \"workspaceId\": \"${WORKSPACE_ID}\",
            \"environment\": \"alpha\",
            \"secretValue\": \"${NEW_GEMINI_KEY}\"
        }"
    echo "[rotation] Gemini key updated"
fi

echo "[rotation] Rotation complete. Restart RAD Gateway to pick up new keys:"
echo "  sudo systemctl restart radgateway01"
```

### Service Token Rotation

```bash
#!/bin/bash
# Rotate Infisical service tokens

INFISICAL_URL="http://localhost:8080"
INFISICAL_TOKEN="CURRENT_SERVICE_TOKEN"

echo "[rotation] Creating new service token..."

# Create new token
NEW_TOKEN_RESPONSE=$(curl -s -X POST "${INFISICAL_URL}/api/v2/service-token" \
    -H "Authorization: Bearer ${INFISICAL_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "radgateway01-token-new",
        "permissions": ["read"],
        "scopes": ["secrets"]
    }')

NEW_TOKEN=$(echo "$NEW_TOKEN_RESPONSE" | jq -r '.serviceToken')

echo "[rotation] New token created. Update your applications and then revoke old token."
echo "[rotation] New token: ${NEW_TOKEN:0:10}..."

# Update token file
echo "$NEW_TOKEN" | sudo tee /opt/radgateway01/config/infisical-token > /dev/null
sudo chmod 600 /opt/radgateway01/config/infisical-token

echo "[rotation] Token file updated. Restart services:"
echo "  sudo systemctl restart radgateway01"
```

### Database Password Rotation

```bash
#!/bin/bash
# Rotate PostgreSQL database passwords

set -e

echo "[rotation] Starting database password rotation"

# Generate new passwords
NEW_INFISICAL_PASS=$(openssl rand -base64 32)
NEW_OPENBAO_PASS=$(openssl rand -base64 32)

# Update PostgreSQL
echo "[rotation] Updating database passwords..."
sudo podman exec -i secret-stack-postgres psql -U postgres << EOF
ALTER USER infisical WITH PASSWORD '${NEW_INFISICAL_PASS}';
ALTER USER openbao WITH PASSWORD '${NEW_OPENBAO_PASS}';
EOF

# Update password files
echo "$NEW_INFISICAL_PASS" | sudo tee /opt/secret-stack/config/infisical-db-password > /dev/null
echo "$NEW_OPENBAO_PASS" | sudo tee /opt/secret-stack/config/openbao-db-password > /dev/null
sudo chmod 600 /opt/secret-stack/config/*-password

echo "[rotation] Passwords updated. Restarting services..."
sudo systemctl restart secret-stack-infisical
sudo systemctl restart secret-stack-openbao

echo "[rotation] Complete"
```

---

## Monitoring

### Key Metrics

| Metric | Target | Warning | Critical | Query |
|--------|--------|---------|----------|-------|
| PostgreSQL Connections | < 150 | 180 | 200 | `SELECT count(*) FROM pg_stat_activity;` |
| OpenBao Seal Status | Unsealed | - | Sealed | `curl /v1/sys/health` |
| Infisical Response | < 100ms | 500ms | 2000ms | `time curl /api/status` |
| Disk Usage | < 70% | 80% | 90% | `df -h` |
| Memory Usage | < 80% | 90% | 95% | `free` |
| Backup Age | < 24h | 36h | 48h | `find /backup -mtime` |

### Health Check Endpoints

```bash
# PostgreSQL
sudo podman exec secret-stack-postgres pg_isready -U postgres

# OpenBao
curl -s http://localhost:8200/v1/sys/health | jq -r '.sealed'

# Infisical
curl -s http://localhost:8080/api/status | jq -r '.status'
```

### Log Monitoring

```bash
# Watch all logs in real-time
sudo tail -f /opt/secret-stack/logs/*/audit.log \
    /var/log/messages | grep -E "(secret-stack|infisical|openbao|postgres)"

# Check for errors in last hour
sudo journalctl -u secret-stack-infisical --since "1 hour ago" -p err
sudo journalctl -u secret-stack-openbao --since "1 hour ago" -p err
```

---

## Incident Response

### Severity Levels

| Level | Definition | Response Time | Examples |
|-------|------------|---------------|----------|
| P1 Critical | Complete outage, data loss | 15 minutes | All services down, database corruption |
| P2 High | Major functionality impaired | 1 hour | OpenBao sealed, backup failures |
| P3 Medium | Partial degradation | 4 hours | Performance issues, intermittent errors |
| P4 Low | Cosmetic, monitoring | 24 hours | Log rotation warnings, non-critical alerts |

### Incident Response Playbooks

#### P1: Complete Secrets Stack Outage

```bash
# 1. Verify scope of outage
curl http://localhost:8080/api/status
curl http://localhost:8200/v1/sys/health
sudo podman exec secret-stack-postgres pg_isready -U postgres

# 2. Check pod status
sudo podman pod ps
sudo podman ps --pod

# 3. If containers are down, restart
cd /mnt/ollama/git/RADAPI01/deploy
sudo ./start-secret-stack.sh

# 4. If PostgreSQL is corrupted, restore from backup
# See Recovery Procedures section above

# 5. Verify recovery
/opt/secret-stack/scripts/daily-health-check.sh
```

#### P2: OpenBao Sealed

```bash
# Symptoms: curl http://localhost:8200/v1/sys/health shows "sealed": true

# 1. Check seal status
curl http://localhost:8200/v1/sys/seal-status | jq

# 2. Unseal with 3 of 5 shards
# Retrieve shards from secure location
# Run unseal command 3 times with different shards
curl -X PUT http://localhost:8200/v1/sys/unseal \
  -d '{"key": "SHARD_1"}'

# 3. Verify unseal
curl http://localhost:8200/v1/sys/health | jq '.sealed'
# Should return: false

# 4. Document who performed unseal for audit trail
echo "$(date): Unseal performed by $(whoami)" >> /opt/secret-stack/logs/unseal-audit.log
```

#### P2: Infisical Service Token Expired

```bash
# Symptoms: RAD Gateway cannot authenticate, logs show 401 errors

# 1. Verify token issue
curl -H "Authorization: Bearer $(cat /opt/radgateway01/config/infisical-token)" \
  http://localhost:8080/api/v3/secrets/raw/RAD_API_KEYS

# 2. Generate new token via Infisical UI
# Navigate to http://radgateway01:8080
# Project Settings > Service Tokens > Create New

# 3. Update token file
echo "new-token-here" | sudo tee /opt/radgateway01/config/infisical-token
sudo chmod 600 /opt/radgateway01/config/infisical-token

# 4. Restart RAD Gateway
sudo systemctl restart radgateway01

# 5. Verify
curl http://localhost:8090/health
```

#### P3: High Database Connection Count

```bash
# Symptoms: Alert that connections > 180

# 1. Check current connections
sudo podman exec secret-stack-postgres psql -U postgres -c \
    "SELECT count(*) FROM pg_stat_activity;"

# 2. See what connections are active
sudo podman exec secret-stack-postgres psql -U postgres -c \
    "SELECT usename, state, count(*) FROM pg_stat_activity GROUP BY usename, state;"

# 3. If many idle connections, consider restarting services
sudo systemctl restart secret-stack-infisical
sudo systemctl restart secret-stack-openbao

# 4. Check connection pool settings in config
sudo podman exec secret-stack-postgres psql -U postgres -c \
    "SHOW max_connections;"
```

### Post-Incident Actions

After any P1 or P2 incident:

1. **Document the incident**:
   - Timeline of events
   - Root cause analysis
   - Actions taken
   - Resolution time

2. **Update runbooks** if needed

3. **Schedule follow-up actions**:
   - Preventive measures
   - Monitoring improvements
   - Documentation updates

4. **Review with team** within 48 hours

---

## Maintenance Windows

### Scheduled Maintenance

| Window | Frequency | Activities |
|--------|-----------|------------|
| Daily | 02:00-02:30 | Automated backups |
| Weekly | Sunday 03:00 | Log rotation, cleanup |
| Monthly | First Sunday | Security updates, full health check |
| Quarterly | Scheduled | Disaster recovery drill |

### Maintenance Procedures

```bash
# Pre-maintenance checklist
echo "1. Notify stakeholders"
echo "2. Verify backup is current: ls -la /backup/secret-stack/"
echo "3. Document current state: /opt/secret-stack/scripts/daily-health-check.sh"

# During maintenance
sudo systemctl stop secret-stack-openbao
sudo systemctl stop secret-stack-infisical
# ... perform maintenance ...
sudo systemctl start secret-stack-infisical
sudo systemctl start secret-stack-openbao

# Post-maintenance verification
/opt/secret-stack/scripts/daily-health-check.sh
```

---

**Document Owner**: Team Hotel (Deployment & Infrastructure)
**Review Schedule**: Monthly during active development, quarterly in maintenance
**Next Review**: 2026-03-16
