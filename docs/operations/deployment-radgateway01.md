# RAD Gateway Deployment Specification

**Container Group**: `<your-container-name>`
**Primary Port**: 8090
**Status**: Alpha Single-Node Deployment

This specification defines a production-ready deployment of RAD Gateway using Podman/Docker containers with optional systemd integration.

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
│  │  Container Pod: <your-container-name>                        │   │
│  │                                                     │   │
│  │  ┌──────────────┐  ┌──────────────┐                │   │
│  │  │ <your-container-name> │  │<your-container-name>- │                │   │
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

### <your-container-name> Pod

```bash
# Pod creation
sudo podman pod create \
  --name <your-container-name> \
  --publish 8090:8090 \
  --network bridge \
  --infra-name <your-container-name>-infra
```

### Containers

#### 1. <your-container-name>-app (Main Application)

```bash
# Build image first
sudo podman build -t <your-container-name>:latest .

# Create container
sudo podman run -d \
  --pod <your-container-name> \
  --name <your-container-name>-app \
  --restart unless-stopped \
  --env-file /opt/<your-container-name>/config/env \
  --volume <your-container-name>-data:/data \
  --health-cmd "curl -f http://localhost:8090/health || exit 1" \
  --health-interval 30s \
  --health-timeout 10s \
  --health-retries 3 \
  localhost/<your-container-name>:latest
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

#### 2. <your-container-name>-postgres (Future - Phase 2)

```bash
# For usage/trace persistence (Milestone 2)
sudo podman run -d \
  --pod <your-container-name> \
  --name <your-container-name>-postgres \
  --restart unless-stopped \
  --env POSTGRES_USER=radgateway \
  --env POSTGRES_PASSWORD_FILE=/run/secrets/db-password \
  --env POSTGRES_DB=radgateway \
  --volume <your-container-name>-postgres-data:/var/lib/postgresql/data \
  docker.io/library/postgres:16-alpine
```

---

## Startup Script with Infisical Integration

### /opt/<your-container-name>/bin/startup.sh

```bash
#!/bin/bash
set -e

echo "[<your-container-name>] Starting up..."

# Infisical configuration - UPDATE THESE VALUES
INFISICAL_URL="http://localhost:8080"
INFISICAL_TOKEN="${INFISICAL_SERVICE_TOKEN%.*}"
WORKSPACE_ID="<your-workspace-id>"

# Fetch secrets from Infisical
echo "[<your-container-name>] Fetching secrets from Infisical..."

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
echo "[<your-container-name>] Starting rad-gateway..."
exec /usr/local/bin/rad-gateway
```

---

## Directory Structure

```
/opt/<your-container-name>/
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
    └── <your-container-name>.service    # Systemd unit file
```

---

## Systemd Service

### /etc/systemd/system/<your-container-name>.service

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
WorkingDirectory=/opt/<your-container-name>

# Environment
Environment="INFISICAL_SERVICE_TOKEN_FILE=/opt/<your-container-name>/config/infisical-token"
Environment="PATH=/usr/local/bin:/usr/bin:/bin"

# Startup
ExecStartPre=-/usr/bin/podman pull localhost/<your-container-name>:latest
ExecStart=/usr/bin/podman run \
    --pod <your-container-name> \
    --name <your-container-name>-app \
    --rm \
    --env-file /opt/<your-container-name>/config/env \
    --volume <your-container-name>-data:/data \
    localhost/<your-container-name>:latest

# Stop
ExecStop=/usr/bin/podman stop -t 30 <your-container-name>-app
ExecStopPost=-/usr/bin/podman rm <your-container-name>-app

# Restart
Restart=always
RestartSec=10

# Security
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/<your-container-name>/data

[Install]
WantedBy=multi-user.target
```

---

## Networking

### Host Ports

| Port | Service | External Access |
|------|---------|-----------------|
| 8090 | <your-container-name>-app | Yes - Primary API |
| 8080 | Infisical | Internal only |
| 5433 | Infisical Postgres | Internal only |
| 6380 | Infisical Redis | Internal only |

### Firewall Rules

```bash
# Allow <your-container-name> port
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
sudo podman logs <your-container-name>-app
```

### Systemd Status

```bash
sudo systemctl status <your-container-name>
sudo journalctl -u <your-container-name> -f
```

---

## Backup and Recovery

### Automated Backup Script

```bash
#!/bin/bash
# /opt/<your-container-name>/bin/backup.sh

BACKUP_DIR="/backup/<your-container-name>"
DATE=$(date +%Y%m%d_%H%M%S)

# Create backup
mkdir -p "${BACKUP_DIR}/${DATE}"

# Backup usage data
cp -r /opt/<your-container-name>/data/* "${BACKUP_DIR}/${DATE}/"

# Backup config
cp /opt/<your-container-name>/config/* "${BACKUP_DIR}/${DATE}/"

# Cleanup old backups (keep 7 days)
find "${BACKUP_DIR}" -type d -mtime +7 -exec rm -rf {} + 2>/dev/null

echo "[<your-container-name>] Backup completed: ${BACKUP_DIR}/${DATE}"
```

### Recovery Procedure

1. Stop service: `sudo systemctl stop <your-container-name>`
2. Restore data from backup
3. Restart service: `sudo systemctl start <your-container-name>`

---

## Monitoring (Future)

### Prometheus Scraping

```yaml
# prometheus.yml addition
scrape_configs:
  - job_name: '<your-container-name>'
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
   sudo podman build -t <your-container-name>:v0.2.0 .
   ```

2. Rolling update:
   ```bash
   sudo systemctl stop <your-container-name>
   sudo podman tag <your-container-name>:v0.2.0 <your-container-name>:latest
   sudo systemctl start <your-container-name>
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

**Fix**: Ensure Infisical is running before starting <your-container-name>

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
sudo chown -R radgateway:radgateway /opt/<your-container-name>/data
sudo chmod 750 /opt/<your-container-name>/data
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
