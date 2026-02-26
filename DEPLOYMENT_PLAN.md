# RAD Gateway Deployment Plan

**Target**: 172.16.30.45  
**Method**: Podman Container-Only (per ADR-011)  
**Date**: 2026-02-19  
**Status**: Ready for Deployment

---

## Pre-Deployment Checklist

### Repository State
- [x] Binary removed from git tracking
- [x] .gitignore updated to exclude `rad-gateway`
- [x] Documentation staged (CLAUDE.md, container-only-policy.md)
- [x] Dockerfile validated (multi-stage build)
- [x] Systemd service validated (container-only execution)

### Container Build
- [x] Dockerfile uses multi-stage build
- [x] Binary installed to `/usr/local/bin/rad-gateway` inside container
- [x] Base image: `alpine:latest` (runtime)
- [x] Builder image: `golang:1.24-alpine`
- [x] Port 8090 exposed

### Deployment Configuration
- [x] Install script: `deploy/install.sh`
- [x] Systemd service: `deploy/systemd/radgateway01.service`
- [x] Target directory: `/opt/radgateway01`
- [x] Service user: `radgateway`
- [x] Service group: `radgateway`

---

## Deployment Steps

### Step 1: Build Container Image

```bash
cd /mnt/ollama/git/RADAPI01
podman build -t radgateway01:latest .
```

**Expected Output:**
- Image `localhost/radgateway01:latest` created
- Build stages: builder + runtime
- Binary copied to `/usr/local/bin/rad-gateway`

### Step 2: Verify Image Integrity

```bash
# Check image exists
podman images | grep radgateway01

# Verify binary location inside container
podman run --rm radgateway01:latest which rad-gateway
# Expected: /usr/local/bin/rad-gateway

# Verify binary is executable
podman run --rm radgateway01:latest ls -la /usr/local/bin/rad-gateway
# Expected: -rwxr-xr-x
```

### Step 3: Transfer to Target Host (if building locally)

```bash
# Save image to tar
podman save -o radgateway01.tar radgateway01:latest

# Transfer to target host
scp radgateway01.tar user@172.16.30.45:/tmp/

# On target host, load image
ssh user@172.16.30.45 'podman load -i /tmp/radgateway01.tar'
```

**Alternative: Build on Target Host**
```bash
# Copy source to target
cd /mnt/ollama/git/RADAPI01
podman build -t radgateway01:latest .
```

### Step 4: Install Deployment Artifacts

```bash
# On 172.16.30.45
cd /mnt/ollama/git/RADAPI01/deploy
sudo ./install.sh
```

**This will:**
1. Create `/opt/radgateway01` directory structure
2. Copy scripts to `/opt/radgateway01/bin/`
3. Copy systemd service to `/etc/systemd/system/`
4. Create `radgateway` user and group
5. Configure firewall (port 8090)
6. Configure iptables FORWARD rules for container networking
7. Enable systemd service

### Step 5: Configure Infisical Token

```bash
# Add your Infisical service token
echo "your-infisical-token" | sudo tee /opt/radgateway01/config/infisical-token
sudo chmod 600 /opt/radgateway01/config/infisical-token
sudo chown radgateway:radgateway /opt/radgateway01/config/infisical-token
```

### Step 6: Start Service

```bash
sudo systemctl start radgateway01
sudo systemctl status radgateway01
```

### Step 7: Verify Deployment

```bash
# Health check
curl http://localhost:8090/health

# Verify container is running
sudo podman ps --pod

# Verify NO direct binary process
ps aux | grep rad-gateway
# Should show conmon process only, NOT direct binary
```

---

## Rollback Plan

### If Deployment Fails:

```bash
# Stop service
sudo systemctl stop radgateway01

# Remove container
sudo podman rm radgateway01-app

# Check logs
sudo journalctl -u radgateway01 -n 100
sudo podman logs radgateway01-app

# Restore previous image (if available)
sudo podman run -d --name radgateway01-app-backup --pod radgateway01 localhost/radgateway01:previous
```

### Complete Uninstall:

```bash
cd /mnt/ollama/git/RADAPI01/deploy
sudo ./uninstall.sh
```

---

## Validation Commands

### Container-Only Compliance Check

```bash
# Verify no binary on host
if [ -f /opt/radgateway01/bin/rad-gateway ]; then
    echo "FAIL: Binary found on host"
else
    echo "PASS: No binary on host"
fi

# Verify systemd uses container
if systemctl cat radgateway01 | grep -q "podman run"; then
    echo "PASS: Systemd uses container"
else
    echo "FAIL: Systemd does not use container"
fi

# Verify container is running
if sudo podman ps | grep -q radgateway01-app; then
    echo "PASS: Container is running"
else
    echo "FAIL: Container not running"
fi
```

---

## Post-Deployment Monitoring

### Health Checks

```bash
# Automated health check
/opt/radgateway01/bin/health-check.sh

# Manual health check
curl -f http://localhost:8090/health
```

### Logs

```bash
# Systemd logs
sudo journalctl -u radgateway01 -f

# Container logs
sudo podman logs -f radgateway01-app
```

---

## References

- **Container Policy**: `docs/architecture/container-only-policy.md`
- **Guardrails**: `CLAUDE.md`
- **Deploy README**: `deploy/README.md`
- **Runbook**: `RUNBOOK.md`

---

**Plan Version**: 1.0  
**Created**: 2026-02-19  
**Status**: Approved for Deployment
