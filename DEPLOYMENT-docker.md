# RAD Gateway Docker Deployment Guide

**Version**: 1.0
**Target**: Test Container Host (Docker)
**Port**: 8090

---

## Overview

This guide covers deploying RAD Gateway using Docker on a test container host.

**Difference from Production:**
- Production uses Podman (per CLAUDE.md guardrails)
- This test deployment uses Docker for evaluation/testing

---

## Prerequisites

| Requirement | Specification |
|-------------|---------------|
| OS | Ubuntu 22.04 LTS, RHEL 8+, or similar |
| CPU | 2+ cores |
| Memory | 4GB+ RAM |
| Docker | 24.0+ installed |
| Network | Port 8090 available |

---

## Installation

### Step 1: Install Docker

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y docker.io

# RHEL/CentOS/Fedora
sudo dnf install -y docker

# Start Docker
sudo systemctl enable docker
sudo systemctl start docker

# Test
sudo docker --version
```

### Step 2: Create Directories

```bash
sudo mkdir -p /opt/radgateway01/{bin,config,data,logs}
sudo useradd -r -s /bin/false radgateway 2>/dev/null || true
sudo chown -R radgateway:radgateway /opt/radgateway01
```

### Step 3: Install Binary

```bash
# Copy binary to server
scp rad-gateway user@<HOST>:/tmp/

# Move to location
sudo mv /tmp/rad-gateway /opt/radgateway01/bin/
sudo chmod +x /opt/radgateway01/bin/rad-gateway
sudo chown radgateway:radgateway /opt/radgateway01/bin/rad-gateway
```

### Step 4: Create Dockerfile

```bash
sudo tee /opt/radgateway01/Dockerfile << 'EOF'
FROM alpine:latest

RUN apk add --no-cache ca-certificates
RUN adduser -D -s /bin/false radgateway

COPY bin/rad-gateway /usr/local/bin/rad-gateway
RUN chmod +x /usr/local/bin/rad-gateway

EXPOSE 8090
USER radgateway
CMD ["/usr/local/bin/rad-gateway"]
EOF
```

### Step 5: Build Image

```bash
cd /opt/radgateway01
sudo docker build -t radgateway01:latest .
sudo docker images | grep radgateway01
```

### Step 6: Configure Environment

```bash
sudo tee /opt/radgateway01/config/env << 'EOF'
RAD_LISTEN_ADDR=:8090
RAD_LOG_LEVEL=info
RAD_ENVIRONMENT=testing
RAD_DB_DRIVER=sqlite
RAD_DB_DSN=/data/radgateway.db
RAD_API_KEYS=test:rad_test_key_001
EOF

sudo chown radgateway:radgateway /opt/radgateway01/config/env
sudo chmod 600 /opt/radgateway01/config/env
```

### Step 7: Create Systemd Service

```bash
sudo tee /etc/systemd/system/radgateway01.service << 'EOF'
[Unit]
Description=RAD Gateway (Docker Test)
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=radgateway
Group=radgateway
WorkingDirectory=/opt/radgateway01
EnvironmentFile=/opt/radgateway01/config/env

ExecStartPre=-/usr/bin/docker rm -f radgateway01-app
ExecStart=/usr/bin/docker run \
    --name radgateway01-app \
    --rm \
    --publish 8090:8090 \
    --env-file /opt/radgateway01/config/env \
    --volume /opt/radgateway01/data:/data \
    --health-cmd "wget -q --spider http://localhost:8090/health || exit 1" \
    --health-interval 30s \
    --health-timeout 10s \
    --health-retries 3 \
    radgateway01:latest

ExecStop=/usr/bin/docker stop -t 30 radgateway01-app
ExecStopPost=-/usr/bin/docker rm radgateway01-app

Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable radgateway01
```

### Step 8: Start Service

```bash
sudo systemctl start radgateway01
sudo systemctl status radgateway01
```

---

## Verification

### Check Container

```bash
sudo docker ps | grep radgateway01
```

### Health Check

```bash
curl http://<HOST>:8090/health
```

### API Test

```bash
curl -X POST http://<HOST>:8090/v1/chat/completions \
  -H "Authorization: Bearer rad_test_key_001" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"Hello"}]}'
```

---

## Management

### View Logs

```bash
sudo docker logs -f radgateway01-app
sudo journalctl -u radgateway01 -f
```

### Restart

```bash
sudo systemctl restart radgateway01
```

### Stop

```bash
sudo systemctl stop radgateway01
sudo docker stop radgateway01-app
```

### Update

```bash
# 1. Stop
sudo systemctl stop radgateway01

# 2. Replace binary
sudo cp /tmp/rad-gateway /opt/radgateway01/bin/

# 3. Rebuild
cd /opt/radgateway01
sudo docker build -t radgateway01:latest .

# 4. Start
sudo systemctl start radgateway01
```

---

## Troubleshooting

### Container Won't Start

```bash
sudo docker logs radgateway01-app
sudo journalctl -u radgateway01 -n 50
```

### Port Conflict

```bash
sudo ss -tlnp | grep 8090
sudo lsof -i :8090
```

### Permission Issues

```bash
sudo chown -R radgateway:radgateway /opt/radgateway01
sudo chmod +x /opt/radgateway01/bin/rad-gateway
```

---

## Comparison: Docker vs Podman

| Feature | Docker | Podman |
|---------|--------|--------|
| Rootless | Optional | Default |
| Daemon | Required | Daemonless |
| systemd | Requires setup | Native support |
| Production | Testing only | Required per CLAUDE.md |

---

**Note**: For production deployments, use Podman per project guardrails.
