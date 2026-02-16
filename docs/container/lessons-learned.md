# Lessons Learned for Next Deployment (radgateway02+)

## Executive Summary

This document captures critical lessons learned from the radgateway01 deployment to ensure smoother, more efficient deployments for radgateway02 and subsequent instances.

**Focus Areas**:
- Podman networking complexities
- iptables FORWARD rule management
- Container security hardening

---

## 1. Podman Networking Deep Dive

### Lesson: Rootful vs Rootless Networking

**The Issue**:
Podman's networking behavior differs significantly between rootful (sudo) and rootless modes. radgateway01 uses rootful Podman, which requires understanding of:
- CNI plugins vs Netavark backend
- Bridge network creation
- Port forwarding at the firewall level

**What Worked**:
```bash
# Rootful pod creation uses bridge networking correctly
sudo podman pod create \
  --name radgateway01 \
  --publish 8090:8090 \
  --network bridge
```

**What Caused Issues**:
- Confusion about when slirp4netvs vs bridge is used
- Port binding conflicts when testing with rootless podman locally

**Recommendations for radgateway02+**:

1. **Standardize on rootful mode** for production deployments
   ```bash
   # Always use sudo for production pod operations
   sudo podman pod create --name radgateway02 --publish 8091:8090
   ```

2. **Document network backend** explicitly
   ```bash
   # Check which backend is in use
   sudo podman info --format '{{.Host.NetworkBackend}}'
   # Should output: netavark (preferred) or cni
   ```

3. **Verify network creation** before container start
   ```bash
   # List available networks
   sudo podman network ls

   # Inspect pod network
   sudo podman pod inspect radgateway02 --format '{{.InfraConfig.Networks}}'
   ```

### Lesson: DNS Resolution Within Pods

**The Issue**:
Containers within a pod communicate via localhost, but external DNS resolution depends on host configuration.

**What Worked**:
```bash
# Infisical access via host networking
# From within container:
curl http://host.containers.internal:8080/api/status
```

**What Didn't Work**:
- Direct `localhost:8080` doesn't resolve to host from rootless containers
- DNS caching issues when host network changes

**Recommendations**:

1. **Use host.containers.internal** for host access
   ```bash
   # In container startup script
   INFISICAL_URL="http://host.containers.internal:8080"
   ```

2. **Add custom DNS if needed**
   ```bash
   sudo podman pod create \
     --name radgateway02 \
     --dns 172.16.30.45 \
     --publish 8091:8090
   ```

3. **Test DNS resolution before deployment**
   ```bash
   sudo podman run --rm --pod radgateway02 alpine nslookup host.containers.internal
   ```

---

## 2. iptables FORWARD Rules

### Lesson: Firewall Configuration is Critical

**The Issue**:
Podman requires specific iptables FORWARD rules for container-to-container and container-to-host communication. These rules are usually created automatically but can be:
- Overwritten by firewall service restarts
- Blocked by restrictive default policies
- Conflicting with Docker rules if both are present

**What We Learned**:

```bash
# Check current FORWARD chain policy
sudo iptables -L FORWARD -n -v

# Default should be ACCEPT or have specific rules
Chain FORWARD (policy ACCEPT 0 packets, 0 bytes)
```

**The Problem Scenario**:
```
# If FORWARD policy is DROP, containers can't communicate
Chain FORWARD (policy DROP 0 packets, 0 bytes)
```

**Recommendations for radgateway02+**:

### 2.1 Pre-Deployment Firewall Checklist

```bash
#!/bin/bash
# /opt/radgateway02/bin/firewall-check.sh

echo "Checking iptables FORWARD policy..."
POLICY=$(sudo iptables -L FORWARD -n | grep "policy" | awk '{print $2}')

if [ "$POLICY" = "DROP" ]; then
    echo "WARNING: FORWARD policy is DROP"
    echo "Run: sudo iptables -P FORWARD ACCEPT"
    exit 1
fi

echo "Checking Podman firewall rules..."
if ! sudo iptables -L FORWARD -n | grep -q "CNI"; then
    echo "WARNING: CNI/Podman FORWARD rules not found"
    echo "Podman may need restart to recreate rules"
fi

echo "Firewall check passed"
```

### 2.2 Persistent Firewall Configuration

```bash
# For firewalld (RHEL/CentOS)
sudo firewall-cmd --permanent --add-port=8091/tcp
sudo firewall-cmd --permanent --zone=trusted --add-interface=cni-podman0
sudo firewall-cmd --reload

# For ufw (Ubuntu)
sudo ufw allow 8091/tcp
sudo ufw route allow in on cni-podman0 out on eth0
```

### 2.3 Automatic Rule Restoration

```bash
# /etc/systemd/system/radgateway02-firewall.service
[Unit]
Description=RAD Gateway 02 Firewall Rules
Before=radgateway02.service

[Service]
Type=oneshot
ExecStart=/opt/radgateway02/bin/firewall-setup.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
```

```bash
# /opt/radgateway02/bin/firewall-setup.sh
#!/bin/bash

# Ensure FORWARD is ACCEPT
iptables -C FORWARD -j ACCEPT 2>/dev/null || iptables -I FORWARD -j ACCEPT

# Ensure NAT for podman
iptables -t nat -C POSTROUTING -s 10.88.0.0/16 -j MASQUERADE 2>/dev/null || \
    iptables -t nat -A POSTROUTING -s 10.88.0.0/16 -j MASQUERADE

echo "Firewall rules applied"
```

### 2.4 Debugging Network Issues

```bash
# Complete network diagnostics script
#!/bin/bash

echo "=== Podman Network Debug ==="
echo ""

echo "1. Podman version:"
podman version --format '{{.Server.Version}}'

echo ""
echo "2. Network backend:"
podman info --format '{{.Host.NetworkBackend}}'

echo ""
echo "3. Pod networks:"
podman network ls

echo ""
echo "4. Bridge interfaces:"
ip addr show | grep -A2 "cni\|podman"

echo ""
echo "5. iptables FORWARD chain:"
iptables -L FORWARD -n -v

echo ""
echo "6. iptables NAT table:"
iptables -t nat -L -n -v | grep -i masquerade

echo ""
echo "7. Container connectivity test:"
podman run --rm alpine ping -c 1 8.8.8.8

echo ""
echo "=== End Debug ==="
```

---

## 3. Container Security Hardening

### Lesson: Defense in Depth is Essential

**What We Learned**:
Security must be layered - no single control is sufficient.

### 3.1 User and Permission Model

**Current State (radgateway01)**:
- Container runs as root inside container (improvement needed)
- Systemd service runs as `radgateway` user

**Target State (radgateway02+)**:
```dockerfile
# Dockerfile improvements
FROM alpine:3.19

# Create non-root user
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

# Install only required packages
RUN apk --no-cache add ca-certificates

# Set up application
WORKDIR /app
COPY --chown=appuser:appgroup rad-gateway .

# Switch to non-root
USER appuser

# Read-only root with explicit write paths
VOLUME ["/tmp", "/data"]

HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8090/health || exit 1

EXPOSE 8090
CMD ["./rad-gateway"]
```

### 3.2 Security Context Recommendations

```bash
# Run container with security options
sudo podman run -d \
  --pod radgateway02 \
  --name radgateway02-app \
  --security-opt=no-new-privileges:true \
  --security-opt=seccomp=unconfined \
  --cap-drop=ALL \
  --cap-add=NET_BIND_SERVICE \
  --read-only \
  --tmpfs /tmp:noexec,nosuid,size=100m \
  localhost/radgateway02:latest
```

### 3.3 Capability Management

| Capability | Required | Reason |
|------------|----------|--------|
| NET_BIND_SERVICE | Yes | Bind to privileged ports (<1024) |
| SETUID/SETGID | No | Drop for security |
| SYS_ADMIN | No | Drop for security |
| NET_ADMIN | No | Drop for security |

```bash
# Drop all, add only what's needed
sudo podman run \
  --cap-drop=ALL \
  --cap-add=NET_BIND_SERVICE \
  ...
```

### 3.4 Read-Only Root Filesystem

**Implementation**:
```bash
# Mark root as read-only
sudo podman run \
  --read-only \
  --volume radgateway02-data:/data:rw \
  --tmpfs /tmp:rw,noexec,nosuid,size=100m \
  ...
```

**Application Changes Needed**:
- Ensure application doesn't write to root filesystem
- Use `/tmp` for temporary files (tmpfs mount)
- Use `/data` for persistent data (volume mount)

### 3.5 Secret Management Improvements

**Current Approach**:
- Startup script fetches secrets from Infisical
- Secrets stored in environment variables

**Improved Approach for radgateway02+**:
```bash
# Use Podman secrets for sensitive data
sudo podman secret create infisical-token /opt/radgateway02/config/token.txt

# Mount secret as file (not environment variable)
sudo podman run -d \
  --pod radgateway02 \
  --secret infisical-token,target=/run/secrets/infisical-token,mode=0400 \
  localhost/radgateway02:latest
```

**Application Modification**:
```go
// Read token from file instead of environment
func loadServiceToken() (string, error) {
    data, err := os.ReadFile("/run/secrets/infisical-token")
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(data)), nil
}
```

---

## 4. Operational Improvements

### 4.1 Deployment Automation

**Create deployment script**:
```bash
#!/bin/bash
# /opt/radgateway02/bin/deploy.sh

set -e

VERSION=${1:-latest}
POD_NAME="radgateway02"
PORT="8091"

echo "[deploy] Starting deployment of radgateway02:${VERSION}..."

# Pre-deployment checks
/opt/radgateway02/bin/pre-deploy-check.sh

# Build/pull image
if [ "$VERSION" = "latest" ]; then
    sudo podman build -t ${POD_NAME}:latest /mnt/ollama/git/RADAPI01
else
    sudo podman pull localhost/${POD_NAME}:${VERSION}
fi

# Stop existing
if sudo podman pod exists ${POD_NAME}; then
    echo "[deploy] Stopping existing pod..."
    sudo podman pod stop ${POD_NAME}
    sudo podman pod rm ${POD_NAME}
fi

# Create pod
sudo podman pod create \
    --name ${POD_NAME} \
    --publish ${PORT}:8090 \
    --network bridge

# Start container
sudo podman run -d \
    --pod ${POD_NAME} \
    --name ${POD_NAME}-app \
    --restart unless-stopped \
    --health-cmd "curl -f http://localhost:8090/health || exit 1" \
    --health-interval 30s \
    localhost/${POD_NAME}:${VERSION}

# Wait for health
sleep 5
if ! curl -sf http://localhost:${PORT}/health; then
    echo "[deploy] ERROR: Health check failed"
    sudo podman logs ${POD_NAME}-app
    exit 1
fi

echo "[deploy] Deployment successful!"
```

### 4.2 Log Management

**Current Issue**: Container logs grow indefinitely

**Solution**:
```bash
# Configure log rotation in container run
sudo podman run -d \
  --log-driver=k8s-file \
  --log-opt max-size=10m \
  --log-opt max-file=3 \
  ...
```

**Systemd Journal Integration**:
```ini
# /etc/systemd/system/radgateway02.service
[Service]
# Forward container logs to journal
StandardOutput=journal
StandardError=journal
SyslogIdentifier=radgateway02
```

### 4.3 Resource Limits

**Add resource constraints**:
```bash
sudo podman run -d \
  --pod radgateway02 \
  --memory=512m \
  --memory-swap=512m \
  --cpus=1.0 \
  --pids-limit=100 \
  ...
```

---

## 5. Checklist for radgateway02 Deployment

### Pre-Deployment

- [ ] Host has Podman 5.0+ installed
- [ ] iptables FORWARD policy is ACCEPT
- [ ] Port 8091 is available
- [ ] Infisical is accessible from host
- [ ] radgateway02 user exists
- [ ] Directory structure created (/opt/radgateway02)
- [ ] Secrets configured in Infisical

### Deployment

- [ ] Run firewall check script
- [ ] Execute deployment script
- [ ] Verify pod is running
- [ ] Verify container is healthy
- [ ] Test HTTP endpoint
- [ ] Test secret injection
- [ ] Check logs for errors

### Post-Deployment

- [ ] Configure systemd service
- [ ] Enable log rotation
- [ ] Set up monitoring alerts
- [ ] Document any deviations
- [ ] Update runbook

---

## 6. Quick Reference

### Essential Commands

```bash
# Pod status
sudo podman pod ps
sudo podman pod inspect radgateway02

# Container operations
sudo podman logs -f radgateway02-app
sudo podman exec -it radgateway02-app sh
sudo podman stats radgateway02-app

# Network debugging
sudo iptables -L FORWARD -n -v
sudo podman network inspect podman

# Cleanup
sudo podman pod stop radgateway02
sudo podman pod rm radgateway02
sudo podman system prune
```

### File Locations

```
/opt/radgateway02/
├── bin/
│   ├── deploy.sh
│   ├── firewall-check.sh
│   └── health-check.sh
├── config/
│   ├── env
│   └── infisical-token
├── data/
└── logs/

/etc/systemd/system/radgateway02.service
/etc/systemd/system/radgateway02-firewall.service
```

---

## Conclusion

The radgateway01 deployment provided valuable insights into:
1. **Podman networking** - Understanding bridge vs rootless modes
2. **Firewall management** - Critical FORWARD rules for container connectivity
3. **Security hardening** - Defense in depth with capabilities, read-only filesystems, and secrets management

Applying these lessons to radgateway02+ will result in:
- Faster deployments
- More reliable networking
- Improved security posture
- Better operational visibility

---

**Document Owner**: Container Engineer, Team Hotel
**Last Updated**: 2026-02-16
**Applies To**: radgateway02 and all future deployments
