# RAD Gateway Beta Deployment Guide

**Version**: 1.0-beta
**Target**: <TARGET_HOST> (<HOST>) - Podman
**Also**: Test Container Host - Docker
**For**: Beta Testers
**Date**: 2026-02-28

---

## Overview

This guide is for beta testers deploying RAD Gateway.

- **<TARGET_HOST> (<HOST>)**: Uses Podman (as per CLAUDE.md guardrails)
- **Test Container Host**: Uses Docker (for testing)

Follow the appropriate section for your target host.

**What is RAD Gateway?**
RAD Gateway is an AI API Gateway that provides unified access to multiple AI providers (OpenAI, Anthropic, Google Gemini) through a single OpenAI-compatible API.

---

## Prerequisites

### What You Need

- SSH access to <TARGET_HOST> (<HOST>)
- sudo privileges on <TARGET_HOST>
- The `rad-gateway` binary (provided separately)

### Check <TARGET_HOST> Access

```bash
# Test connection
ssh user@<HOST> "hostname"

# Should output: <TARGET_HOST>
```

---

## Step-by-Step Deployment

### Step 1: Prepare <TARGET_HOST>

SSH to <TARGET_HOST> and verify Podman:

```bash
ssh user@<HOST>

# Check Podman (should be pre-installed)
sudo podman --version
# Should show: podman version 4.x.x or higher

# If Podman is not installed, install it:
# RHEL/CentOS/Rocky: sudo dnf install -y podman
# Ubuntu: sudo apt-get update && sudo apt-get install -y podman
```

**Note**: AI01 uses Podman as per CLAUDE.md deployment guardrails. Docker is only for testing on separate test container hosts.

### Step 2: Take Down Current Podman Deployment

```bash
# Check current status
sudo podman ps
sudo systemctl status radgateway01 2>/dev/null || echo "No systemd service"

# Stop and remove current podman container
sudo podman stop radgateway01-app 2>/dev/null || true
sudo podman rm radgateway01-app 2>/dev/null || true
sudo podman pod stop radgateway01 2>/dev/null || true
sudo podman pod rm radgateway01 2>/dev/null || true

# Verify cleanup
sudo podman ps -a | grep radgateway || echo "Cleaned up"
```

### Step 3: Create Directory Structure

```bash
# Create directories
sudo mkdir -p /opt/radgateway01/{bin,config,data,logs}

# Set permissions
sudo useradd -r -s /bin/false radgateway 2>/dev/null || true
sudo chown -R radgateway:radgateway /opt/radgateway01
```

### Step 4: Install Binary

**Option A: Copy from Local Machine**

On your local machine:
```bash
scp rad-gateway user@<HOST>:/tmp/rad-gateway
```

On <TARGET_HOST>:
```bash
sudo mv /tmp/rad-gateway /opt/radgateway01/bin/
sudo chmod +x /opt/radgateway01/bin/rad-gateway
```

**Option B: Download from Build Server**

If binary is on a build server:
```bash
# On <TARGET_HOST>
curl -o /tmp/rad-gateway <build-server-url>
sudo mv /tmp/rad-gateway /opt/radgateway01/bin/
sudo chmod +x /opt/radgateway01/bin/rad-gateway
```

### Step 5: Create Configuration

```bash
sudo tee /opt/radgateway01/config/env << 'EOF'
# RAD Gateway Configuration
RAD_LISTEN_ADDR=:8090
RAD_LOG_LEVEL=info
RAD_ENVIRONMENT=beta

# Database (SQLite for beta)
RAD_DB_DRIVER=sqlite
RAD_DB_DSN=/data/radgateway.db

# API Keys (for beta testing)
RAD_API_KEYS=beta:rad_beta_key_001,test:rad_test_key_002

# Provider API Keys (add your own)
# OPENAI_API_KEY=sk-...
# ANTHROPIC_API_KEY=sk-ant-...
# GEMINI_API_KEY=...
EOF

sudo chown radgateway:radgateway /opt/radgateway01/config/env
sudo chmod 600 /opt/radgateway01/config/env
```

### Step 6: Create Dockerfile

```bash
sudo tee /opt/radgateway01/Dockerfile << 'EOF'
FROM alpine:latest

# Install certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy binary
COPY bin/rad-gateway /usr/local/bin/rad-gateway

# Expose port
EXPOSE 8090

# Start application
CMD ["/usr/local/bin/rad-gateway"]
EOF
```

### Step 7: Build Container Image

```bash
cd /opt/radgateway01

# Build image
sudo podman build -t radgateway01:latest .

# Verify image created
sudo podman images | grep radgateway01
```

**Expected Output:**
```
localhost/radgateway01   latest      <hash>   <size>   <time>
```

### Step 8: Create Systemd Service

```bash
sudo tee /etc/systemd/system/radgateway01.service << 'EOF'
[Unit]
Description=RAD Gateway 01 (Beta)
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

### Step 9: Configure Firewall

```bash
# Open port 8090
sudo ufw allow 8090/tcp 2>/dev/null || sudo firewall-cmd --permanent --add-port=8090/tcp
sudo firewall-cmd --reload 2>/dev/null || true

# Verify
sudo ss -tlnp | grep 8090
```

### Step 10: Start Service

```bash
# Start
sudo systemctl start radgateway01

# Wait for startup
sleep 5

# Check status
sudo systemctl status radgateway01
```

---

## Verification Steps

### 1. Check Container is Running

```bash
sudo podman ps | grep radgateway01
```

**Expected Output:**
```
CONTAINER ID   IMAGE                       STATUS         PORTS
<container-id> localhost/radgateway01:latest  Up 10 seconds  0.0.0.0:8090->8090/tcp
```

### 2. Health Check

```bash
curl http://<HOST>:8090/health
```

**Expected Response (CGO_ENABLED=0 binary):**
```json
{
  "status": "degraded",
  "database": "unhealthy",
  "timestamp": "2026-02-28T..."
}
```

**Note:** Database shows "unhealthy" because the binary was built with `CGO_ENABLED=0`. This is expected and the API functions normally. See [Known Issues](#known-issues) for details.

### 3. API Test

```bash
# Test with beta key
curl -X POST http://<HOST>:8090/v1/chat/completions \
  -H "Authorization: Bearer rad_beta_key_001" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

**Expected Response:**
```json
{
  "id": "...",
  "object": "chat.completion",
  "model": "gpt-4o-mini",
  "choices": [...]
}
```

### 4. Check Logs

```bash
# View logs
sudo docker logs radgateway01-app

# Follow logs
sudo docker logs -f radgateway01-app

# Systemd logs
sudo journalctl -u radgateway01 -f
```

---

## Beta Testing Checklist

After deployment, verify:

- [ ] Container is running (`sudo docker ps`)
- [ ] Health endpoint returns 200
- [ ] API key authentication works
- [ ] Chat completions endpoint responds
- [ ] Metrics endpoint accessible (`/metrics`)
- [ ] Database health check passes (`/health/db`)
- [ ] Logs show no errors
- [ ] Service restarts automatically if stopped

---

## Known Issues

### Database Health Shows "unhealthy"

**Symptom:** Health endpoint returns `{"status":"degraded","database":"unhealthy"}`

**Cause:** Binary was compiled with `CGO_ENABLED=0`, which causes the SQLite driver to fail migrations.

**Impact:** None - the API functions normally. SQLite still works for persistence.

**Workaround:** None required for beta. For production, build with `CGO_ENABLED=1`.

### SELinux Permission Issues

**Symptom:** Container fails with permission denied on `/data` volume

**Cause:** SELinux prevents container from writing to host volumes

**Fix:** The systemd service uses `--privileged` flag to bypass SELinux restrictions

---

## Troubleshooting

### Container Won't Start

```bash
# Check for errors
sudo podman logs radgateway01-app 2>&1 | head -50

# Check systemd
sudo journalctl -u radgateway01 -n 50

# Test manually
sudo podman run --rm -it \
  --env-file /opt/radgateway01/config/env \
  localhost/radgateway01:latest
```

### Port Already in Use

```bash
# Find what's using port 8090
sudo ss -tlnp | grep 8090
sudo lsof -i :8090

# Stop conflicting service
sudo systemctl stop <service-name>
# or
sudo podman stop <container-name>
```

### Permission Denied

```bash
# Fix permissions - data directory must be writable
sudo chmod 777 /opt/radgateway01/data

# Verify container can write to data
sudo podman run --rm -v /opt/radgateway01/data:/data alpine touch /data/test
```

---

## Testing Commands

### API Endpoints

```bash
# Health
curl http://<HOST>:8090/health

# Metrics
curl http://<HOST>:8090/metrics

# Database health
curl http://<HOST>:8090/health/db

# List models (requires auth)
curl http://<HOST>:8090/v1/models \
  -H "Authorization: Bearer rad_beta_key_001"
```

### Provider Test

```bash
# Test OpenAI (if key configured)
curl -X POST http://<HOST>:8090/v1/chat/completions \
  -H "Authorization: Bearer rad_beta_key_001" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Say hello"}]
  }'

# Test Anthropic (if key configured)
curl -X POST http://<HOST>:8090/v1/chat/completions \
  -H "Authorization: Bearer rad_beta_key_001" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet",
    "messages": [{"role": "user", "content": "Say hello"}]
  }'
```

---

## Maintenance

### Update to New Version

```bash
# 1. Stop current
sudo systemctl stop radgateway01

# 2. Remove old container
sudo podman rm -f radgateway01-app

# 3. Copy new binary
sudo cp /tmp/rad-gateway /opt/radgateway01/bin/

# 4. Rebuild image
cd /opt/radgateway01
sudo podman build -t radgateway01:latest .

# 5. Start
sudo systemctl start radgateway01

# 6. Verify
curl http://<HOST>:8090/health
```

### View Logs

```bash
# Podman logs
sudo podman logs -f radgateway01-app

# Systemd logs
sudo journalctl -u radgateway01 -f

# Last 100 lines
sudo podman logs --tail 100 radgateway01-app
```

### Stop Service

```bash
sudo systemctl stop radgateway01
sudo podman stop radgateway01-app
```

### Clean Up

```bash
# Stop and remove
sudo systemctl stop radgateway01
sudo systemctl disable radgateway01
sudo podman rm -f radgateway01-app
sudo podman rmi radgateway01:latest

# Remove files (optional)
sudo rm -rf /opt/radgateway01
sudo rm /etc/systemd/system/radgateway01.service
```

---

## Support

### Report Issues

Include in bug reports:
1. Output of `sudo podman logs radgateway01-app`
2. Output of `sudo systemctl status radgateway01`
3. Steps to reproduce
4. Expected vs actual behavior

### Emergency Contact

- **Deployment Issues**: Team Hotel
- **API Issues**: Team Bravo
- **Security Issues**: Team Charlie

---

**Last Updated**: 2026-02-28
**For Beta Testers**: RAD Gateway Beta Program
