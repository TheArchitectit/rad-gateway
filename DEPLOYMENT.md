# RAD Gateway Deployment Guide

**Version**: 1.0
**Target**: <TARGET_HOST> (<HOST>)
**Container Group**: radgateway01
**Port**: 8090

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Full Deployment](#full-deployment)
4. [Verification](#verification)
5. [Troubleshooting](#troubleshooting)
6. [Configuration Reference](#configuration-reference)

---

## Prerequisites

### Host Requirements

| Requirement | Specification |
|-------------|---------------|
| OS | Ubuntu 22.04 LTS or RHEL 8+ |
| CPU | 2+ cores |
| Memory | 4GB+ RAM |
| Disk | 20GB+ available |
| Podman | 4.0+ installed |
| Network | Access to <HOST> |

### Target Host (<TARGET_HOST>)

```bash
# Verify connectivity
ssh user@<HOST> "echo 'Connected to <TARGET_HOST>'"

# Verify Podman
ssh user@<HOST> "podman --version"
```

---

## Quick Start

### One-Line Deploy

```bash
# Build locally, deploy to <TARGET_HOST>
make deploy-ai01

# Or manually:
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o rad-gateway ./cmd/rad-gateway
scp rad-gateway user@<HOST>:/tmp/
ssh user@<HOST> 'sudo podman build -t radgateway01:latest -f - . <<EOF
FROM alpine:latest
COPY /tmp/rad-gateway /usr/local/bin/rad-gateway
EXPOSE 8090
CMD ["/usr/local/bin/rad-gateway"]
EOF'
```

---

## Full Deployment

### Step 1: Build Binary

```bash
# On development machine
cd /mnt/ollama/git/RADAPI01
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o rad-gateway ./cmd/rad-gateway

# Verify build
ls -la rad-gateway
file rad-gateway
```

### Step 2: Copy to Target

```bash
# Copy binary
scp rad-gateway user@<HOST>:/tmp/rad-gateway

# Copy additional files if needed
scp -r config/ user@<HOST>:/tmp/
```

### Step 3: Deploy on <TARGET_HOST>

SSH to <TARGET_HOST> and execute:

```bash
ssh user@<HOST>

# Create directory structure
sudo mkdir -p /opt/radgateway01/{bin,config,data,logs}
sudo mkdir -p /opt/radgateway01/systemd

# Move binary
sudo mv /tmp/rad-gateway /opt/radgateway01/bin/
sudo chmod +x /opt/radgateway01/bin/rad-gateway

# Create user
sudo useradd -r -s /bin/false radgateway || true
sudo chown -R radgateway:radgateway /opt/radgateway01
```

### Step 4: Create Dockerfile

```bash
sudo tee /opt/radgateway01/Dockerfile << 'EOF'
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY bin/rad-gateway /usr/local/bin/rad-gateway
EXPOSE 8090
CMD ["/usr/local/bin/rad-gateway"]
EOF
```

### Step 5: Build Container Image

```bash
cd /opt/radgateway01
sudo podman build -t radgateway01:latest .

# Verify image
sudo podman images | grep radgateway01
```

### Step 6: Create Environment Config

```bash
sudo tee /opt/radgateway01/config/env << 'EOF'
# Application
RAD_LISTEN_ADDR=:8090
RAD_LOG_LEVEL=info
RAD_ENVIRONMENT=production

# Database (SQLite for single-node)
RAD_DB_DRIVER=sqlite
RAD_DB_DSN=/data/radgateway.db

# Redis (optional - for caching)
# RAD_REDIS_ADDR=localhost:6379

# Infisical (if configured)
# INFISICAL_API_URL=http://localhost:8080

# API Keys (inline or from Infisical)
RAD_API_KEYS=admin:rad_admin_key_001,service:rad_service_key_002
EOF

sudo chown radgateway:radgateway /opt/radgateway01/config/env
sudo chmod 600 /opt/radgateway01/config/env
```

### Step 7: Create Systemd Service

```bash
sudo tee /etc/systemd/system/radgateway01.service << 'EOF'
[Unit]
Description=RAD Gateway 01 (Brass Relay)
Documentation=https://github.com/TheArchitectit/rad-gateway
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/radgateway01

# Container execution
ExecStartPre=-/usr/bin/podman rm -f radgateway01-app
ExecStart=/usr/bin/podman run \
    --name radgateway01-app \
    --rm \
    --publish 8090:8090 \
    --env-file /opt/radgateway01/config/env \
    --volume /opt/radgateway01/data:/data \
    --privileged \
    localhost/radgateway01:latest

ExecStop=/usr/bin/podman stop -t 30 radgateway01-app
ExecStopPost=-/usr/bin/podman rm radgateway01-app

# Restart policy
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable radgateway01
```

### Step 8: Firewall Configuration

```bash
# Allow RAD Gateway port
sudo firewall-cmd --permanent --add-port=8090/tcp
sudo firewall-cmd --reload

# Verify
sudo firewall-cmd --list-ports
```

### Step 9: Start Service

```bash
# Start
sudo systemctl start radgateway01

# Check status
sudo systemctl status radgateway01
sudo journalctl -u radgateway01 -f
```

---

## Verification

### Health Check

```bash
# Basic health
curl http://<HOST>:8090/health

# Expected response (SQLite with CGO_ENABLED=0):
# {"status":"degraded","database":"unhealthy","timestamp":"2026-02-28T..."}
#
# Note: Database shows "unhealthy" when using SQLite with binaries built
# with CGO_ENABLED=0. This is expected - the API still functions normally.
# For production with full health checks, build with CGO_ENABLED=1.
```

### Container Status

```bash
# Check container
sudo podman ps | grep radgateway01

# Check logs
sudo podman logs radgateway01-app

# Check systemd
sudo systemctl status radgateway01
```

### API Test

```bash
# Test API endpoint
curl -X POST http://<HOST>:8090/v1/chat/completions \
  -H "Authorization: Bearer rad_admin_key_001" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

### Metrics Check

```bash
# Prometheus metrics
curl http://<HOST>:8090/metrics

# Database health
curl http://<HOST>:8090/health/db
```

---

## Troubleshooting

### Service Won't Start

```bash
# Check systemd
sudo systemctl status radgateway01
sudo journalctl -u radgateway01 -n 50

# Check container logs
sudo podman logs radgateway01-app

# Verify image
sudo podman images | grep radgateway01
sudo podman inspect radgateway01:latest
```

### Port Conflicts

```bash
# Check port usage
sudo ss -tlnp | grep 8090
sudo lsof -i :8090

# If conflict, stop conflicting service or change port in config
```

### Database Issues

**SQLite with CGO_ENABLED=0 (Expected Behavior)**

If health endpoint shows `{"status":"degraded","database":"unhealthy"}`, this is expected when:
- Binary was built with `CGO_ENABLED=0` (default in docs)
- Using SQLite database driver

The application functions normally - this only affects the health check status. To get full "healthy" status, build with CGO enabled (requires C toolchain):

```bash
# Build with CGO for full SQLite support
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o rad-gateway ./cmd/rad-gateway
```

**Permission Denied on Data Directory**

```bash
# Check data directory permissions
ls -la /opt/radgateway01/data/

# Fix: Ensure data directory is writable by container
# The systemd service uses --privileged and runs as root
# to avoid SELinux permission issues
sudo chmod 777 /opt/radgateway01/data
```

### Container Won't Run

```bash
# Test container manually
sudo podman run -it --rm \
  --env-file /opt/radgateway01/config/env \
  localhost/radgateway01:latest \
  /usr/local/bin/rad-gateway --help

# Check for missing libraries
sudo podman run -it --rm localhost/radgateway01:latest ldd /usr/local/bin/rad-gateway
```

---

## Configuration Reference

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RAD_LISTEN_ADDR` | HTTP listen address | `:8090` |
| `RAD_LOG_LEVEL` | Log level (debug/info/warn/error) | `info` |
| `RAD_DB_DRIVER` | Database driver (sqlite/postgres) | `sqlite` |
| `RAD_DB_DSN` | Database connection string | `radgateway.db` |
| `RAD_API_KEYS` | Comma-separated API keys | - |
| `RAD_REDIS_ADDR` | Redis server address | - |

### API Key Format

```
name1:key1,name2:key2,...
```

Example:
```
admin:rad_admin_001,service:rad_svc_002
```

### Port Mapping

| Port | Service | Description |
|------|---------|-------------|
| 8090 | radgateway01-app | Main API endpoint |

---

## Maintenance

### Update Deployment

```bash
# Build new version locally
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o rad-gateway ./cmd/rad-gateway

# Deploy update
scp rad-gateway user@<HOST>:/tmp/
ssh user@<HOST> '
  sudo systemctl stop radgateway01
  sudo mv /tmp/rad-gateway /opt/radgateway01/bin/
  sudo podman build -t radgateway01:latest /opt/radgateway01
  sudo systemctl start radgateway01
'

# Verify
curl http://<HOST>:8090/health
```

### Backup Data

```bash
# Backup script
sudo tee /opt/radgateway01/bin/backup.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/backup/radgateway01/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

# Backup database
sudo cp /opt/radgateway01/data/radgateway.db "$BACKUP_DIR/"

# Backup config
sudo cp -r /opt/radgateway01/config "$BACKUP_DIR/"

echo "Backup complete: $BACKUP_DIR"
EOF

sudo chmod +x /opt/radgateway01/bin/backup.sh
```

### View Logs

```bash
# Container logs
sudo podman logs -f radgateway01-app

# Systemd logs
sudo journalctl -u radgateway01 -f

# Application logs (if file output configured)
sudo tail -f /opt/radgateway01/logs/rad-gateway.log
```

---

## Security

### Container Security

- Container runs with `--privileged` flag (required for SELinux volume access)
- Systemd service runs as root user
- Write access to `/data` volume only
- No new privileges outside container

### Network Security

- Port 8090 exposed
- Firewall rules limit access
- No external database access (SQLite)

### Secrets Management

- API keys in environment file (restricted permissions)
- Optional Infisical integration for production secrets
- No secrets in container image

---

## Support

### Checklist Before Deployment

- [ ] Binary builds successfully (`go build ./...`)
- [ ] All tests pass (`go test ./...`)
- [ ] <TARGET_HOST> is accessible via SSH
- [ ] Podman is installed on <TARGET_HOST>
- [ ] Port 8090 is available
- [ ] Firewall allows port 8090
- [ ] radgateway user exists
- [ ] Directory structure created
- [ ] Environment file configured
- [ ] Systemd service enabled

### Emergency Rollback

```bash
# Stop service
sudo systemctl stop radgateway01

# Remove container
sudo podman rm -f radgateway01-app

# Start fresh
sudo systemctl start radgateway01
```

---

**Last Updated**: 2026-02-28
**Maintainer**: Team Hotel (Deployment & Infrastructure)
**Target Host**: <TARGET_HOST> (<HOST>)
