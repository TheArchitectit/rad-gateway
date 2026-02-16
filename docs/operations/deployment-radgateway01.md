# RAD Gateway Deployment Specification

**Container Group**: radgateway01
**Primary Port**: 8090
**Status**: Alpha Single-Node Deployment
**Team**: Team Hotel (Deployment & Infrastructure)

## Overview

This specification defines a production-ready deployment of RAD Gateway using Podman containers with systemd integration.

---

## Deployment Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Host: Production Server                                     │
│  OS: RHEL/Ubuntu with Podman/Docker                         │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Container Pod: radgateway01                        │   │
│  │                                                     │   │
│  │  ┌──────────────┐  ┌──────────────┐                │   │
│  │  │ radgateway01 │  │radgateway01- │                │   │
│  │  │   -app       │  │  postgres    │ (future)       │   │
│  │  │   :8090      │  │   :5432      │                │   │
│  │  └──────────────┘  └──────────────┘                │   │
│  │          │                                        │   │
│  │          └──────┐                                 │   │
│  │                 ▼                                 │   │
│  │  ┌─────────────────────────────────────┐         │   │
│  │  │  Secrets Management (Infisical)     │         │   │
│  │  │  http://localhost:8080              │         │   │
│  │  └─────────────────────────────────────┘         │   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                 │
│                           │ (host networking)               │
│                           ▼                                 │
│                    localhost:8080                           │
└─────────────────────────────────────────────────────────────┘

External Access:
- Direct: http://<host>:8090
- Future: Via reverse proxy (Traefik/Nginx)
```

---

## Container Group Definition

### radgateway01 Pod

```bash
# Pod creation
sudo podman pod create \
  --name radgateway01 \
  --publish 8090:8090 \
  --network bridge \
  --infra-name radgateway01-infra
```

### Containers

#### 1. radgateway01-app (Main Application)

```bash
# Build image first
sudo podman build -t radgateway01:latest .

# Create container
sudo podman run -d \
  --pod radgateway01 \
  --name radgateway01-app \
  --restart unless-stopped \
  --env-file /opt/radgateway01/config/env \
  --volume radgateway01-data:/data \
  --health-cmd "curl -f http://localhost:8090/health || exit 1" \
  --health-interval 30s \
  --health-timeout 10s \
  --health-retries 3 \
  localhost/radgateway01:latest
```

**Environment Variables**:
```
# Application
RAD_LISTEN_ADDR=:8090
RAD_LOG_LEVEL=info
RAD_ENVIRONMENT=alpha

# Infisical (local access)
INFISICAL_API_URL=http://localhost:8080
INFISICAL_PROJECT_SLUG=<your-project-slug>
INFISICAL_WORKSPACE_ID=<your-workspace-id>

# Populated by startup script from Infisical:
# RAD_API_KEYS
# OPENAI_API_KEY
# ANTHROPIC_API_KEY
# GEMINI_API_KEY
```

#### 2. radgateway01-postgres (Future - Phase 2)

```bash
# For usage/trace persistence (Milestone 2)
sudo podman run -d \
  --pod radgateway01 \
  --name radgateway01-postgres \
  --restart unless-stopped \
  --env POSTGRES_USER=radgateway \
  --env POSTGRES_PASSWORD_FILE=/run/secrets/db-password \
  --env POSTGRES_DB=radgateway \
  --volume radgateway01-postgres-data:/var/lib/postgresql/data \
  docker.io/library/postgres:16-alpine
```

---

## Startup Script with Infisical Integration

### /opt/radgateway01/bin/startup.sh

```bash
#!/bin/bash
set -e

echo "[radgateway01] Starting up..."

# Infisical configuration - UPDATE THESE VALUES
INFISICAL_URL="http://localhost:8080"
INFISICAL_TOKEN="${INFISICAL_SERVICE_TOKEN%.*}"
WORKSPACE_ID="<your-workspace-id>"

# Fetch secrets from Infisical
echo "[radgateway01] Fetching secrets from Infisical..."

# Get RAD_API_KEYS
RAD_API_KEYS=$(curl -s \
  -H "Authorization: Bearer ${INFISICAL_TOKEN}" \
  "${INFISICAL_URL}/api/v3/secrets/raw/RAD_API_KEYS?workspaceId=${WORKSPACE_ID}&environment=alpha" \
  | jq -r '.secret.secretValue // empty')

# Get provider keys
OPENAI_API_KEY=$(curl -s \
  -H "Authorization: Bearer ${INFISICAL_TOKEN}" \
  "${INFISICAL_URL}/api/v3/secrets/raw/OPENAI_API_KEY?workspaceId=${WORKSPACE_ID}&environment=alpha" \
  | jq -r '.secret.secretValue // empty')

ANTHROPIC_API_KEY=$(curl -s \
  -H "Authorization: Bearer ${INFISICAL_TOKEN}" \
  "${INFISICAL_URL}/api/v3/secrets/raw/ANTHROPIC_API_KEY?workspaceId=${WORKSPACE_ID}&environment=alpha" \
  | jq -r '.secret.secretValue // empty')

GEMINI_API_KEY=$(curl -s \
  -H "Authorization: Bearer ${INFISICAL_TOKEN}" \
  "${INFISICAL_URL}/api/v3/secrets/raw/GEMINI_API_KEY?workspaceId=${WORKSPACE_ID}&environment=alpha" \
  | jq -r '.secret.secretValue // empty')

# Export environment
export RAD_API_KEYS
export OPENAI_API_KEY
export ANTHROPIC_API_KEY
export GEMINI_API_KEY

# Start application
echo "[radgateway01] Starting rad-gateway..."
exec /usr/local/bin/rad-gateway
```

---

## Directory Structure

```
/opt/radgateway01/
├── bin/
│   ├── startup.sh              # Infisical integration + startup
│   ├── health-check.sh         # Health check script
│   └── backup.sh               # Backup script
├── config/
│   ├── env                     # Static environment variables
│   └── radgateway.yaml         # Application config (future)
├── data/                       # Podman volume mount
│   └── usage/                  # Usage logs
├── logs/                       # Log directory
└── systemd/
    └── radgateway01.service    # Systemd unit file
```

---

## Systemd Service

### /etc/systemd/system/radgateway01.service

```ini
[Unit]
Description=RAD Gateway 01 (Brass Relay)
Documentation=https://github.com/TheArchitectit/rad-gateway
Requires=infisical.service
After=infisical.service network.target

[Service]
Type=simple
User=radgateway
Group=radgateway
WorkingDirectory=/opt/radgateway01

# Environment
Environment="INFISICAL_SERVICE_TOKEN_FILE=/opt/radgateway01/config/infisical-token"
Environment="PATH=/usr/local/bin:/usr/bin:/bin"

# Startup
ExecStartPre=-/usr/bin/podman pull localhost/radgateway01:latest
ExecStart=/usr/bin/podman run \
    --pod radgateway01 \
    --name radgateway01-app \
    --rm \
    --env-file /opt/radgateway01/config/env \
    --volume radgateway01-data:/data \
    localhost/radgateway01:latest

# Stop
ExecStop=/usr/bin/podman stop -t 30 radgateway01-app
ExecStopPost=-/usr/bin/podman rm radgateway01-app

# Restart
Restart=always
RestartSec=10

# Security
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/radgateway01/data

[Install]
WantedBy=multi-user.target
```

---

## Networking

### Host Ports

| Port | Service | External Access |
|------|---------|-----------------|
| 8090 | radgateway01-app | Yes - Primary API |
| 8080 | Infisical | Internal only |
| 5433 | Infisical Postgres | Internal only |
| 6380 | Infisical Redis | Internal only |

### Firewall Rules

```bash
# Allow radgateway01 port
sudo firewall-cmd --permanent --add-port=8090/tcp
sudo firewall-cmd --reload
```

---

## Health Checks

### Application Health

```bash
# Check gateway health
curl http://localhost:8090/health

# Expected response:
# {"status":"healthy","version":"0.1.0-alpha"}
```

### Container Health

```bash
# Check pod status
sudo podman pod ps
sudo podman ps --pod

# Check logs
sudo podman logs radgateway01-app
```

### Systemd Status

```bash
sudo systemctl status radgateway01
sudo journalctl -u radgateway01 -f
```

---

## Backup and Recovery

### Automated Backup Script

```bash
#!/bin/bash
# /opt/radgateway01/bin/backup.sh

BACKUP_DIR="/backup/radgateway01"
DATE=$(date +%Y%m%d_%H%M%S)

# Create backup
mkdir -p "${BACKUP_DIR}/${DATE}"

# Backup usage data
cp -r /opt/radgateway01/data/* "${BACKUP_DIR}/${DATE}/"

# Backup config
cp /opt/radgateway01/config/* "${BACKUP_DIR}/${DATE}/"

# Cleanup old backups (keep 7 days)
find "${BACKUP_DIR}" -type d -mtime +7 -exec rm -rf {} + 2>/dev/null

echo "[radgateway01] Backup completed: ${BACKUP_DIR}/${DATE}"
```

### Recovery Procedure

1. Stop service: `sudo systemctl stop radgateway01`
2. Restore data from backup
3. Restart service: `sudo systemctl start radgateway01`

---

## Monitoring (Future)

### Prometheus Scraping

```yaml
# prometheus.yml addition
scrape_configs:
  - job_name: 'radgateway01'
    static_configs:
      - targets: ['localhost:8090']
    metrics_path: /metrics
```

### Key Metrics

- `radgateway_requests_total`
- `radgateway_request_duration_seconds`
- `radgateway_provider_health`
- `radgateway_usage_tokens_total`

---

## Upgrade Procedure

1. Build new image:
   ```bash
   sudo podman build -t radgateway01:v0.2.0 .
   ```

2. Rolling update:
   ```bash
   sudo systemctl stop radgateway01
   sudo podman tag radgateway01:v0.2.0 radgateway01:latest
   sudo systemctl start radgateway01
   ```

3. Verify:
   ```bash
   curl http://localhost:8090/health
   ```

---

## Troubleshooting

### Issue: Cannot connect to Infisical

**Symptom**: App fails to start, logs show "connection refused"

**Check**:
```bash
curl http://localhost:8080/api/status
sudo systemctl status infisical
```

**Fix**: Ensure Infisical is running before starting radgateway01

### Issue: Port 8090 already in use

**Check**:
```bash
sudo ss -tlnp | grep 8090
sudo podman ps | grep 8090
```

**Fix**: Stop conflicting container or change port mapping

### Issue: Permission denied on data directory

**Fix**:
```bash
sudo chown -R radgateway:radgateway /opt/radgateway01/data
sudo chmod 750 /opt/radgateway01/data
```

---

## Security Considerations

1. **Secrets**: Only bootstrap token in local file; all app secrets from Infisical
2. **Network**: Port 8090 exposed; firewall rules limit access
3. **User**: Service runs as non-root `radgateway` user
4. **Filesystem**: Read-only root, write only to `/data`
5. **Updates**: Image updates require explicit pull and restart

---

## Future Scaling

### Additional Instance Deployment

When deploying additional instances:
- Use different ports (8091, 8092, etc.)
- Share Infisical for secrets
- Add load balancer (Traefik/Nginx)
- Separate PostgreSQL for multi-instance coordination

---

**Deployment Owner**: Team Hotel (Deployment & Infrastructure)
**Last Updated**: 2026-02-16
**Next Review**: Post-MVP launch
