# Container-Only Deployment Policy

**Policy ID**: ADR-011
**Status**: Active
**Effective Date**: 2026-02-19
**Owner**: Team Alpha (Architecture) + Team Hotel (Infrastructure)
**Target**: RAD Gateway v0.1.0+ on 172.16.30.45

---

## Policy Statement

**CRITICAL**: The RAD Gateway application MUST run exclusively inside Podman containers on 172.16.30.45. Direct installation of the binary or any application components on the host VM is FORBIDDEN.

---

## Rationale

### Operational Isolation

Containerization provides clear boundaries between the application and the host system. This isolation:

- Prevents resource conflicts (port conflicts, library version conflicts)
- Enables clean removal and reinstallation
- Provides consistent environment across deployments
- Simplifies debugging and troubleshooting

### Security Enhancement

Containers provide additional security layers:

- Process isolation via Linux namespaces
- Resource limiting via cgroups
- Immutable base image (alpine) reduces attack surface
- Easy to apply security patches via image rebuild
- Seccomp profiles can restrict syscalls

### Reproducibility

Container images ensure the same runtime environment everywhere:

- Dependencies are baked into the image
- No configuration drift over time
- No "it works on my machine" issues
- Easy to recreate environment in new locations

### Deployment Efficiency

Container-based deployment enables:

- Fast rollback to previous images
- Canary deployments with multiple versions
- A/B testing capabilities
- Horizontal scaling via replication

---

## Technical Implementation

### Required Deployment Pattern

```
Build (Local/CI)
    |
    v
Container Image (Dockerfile)
    |
    v
Image Repository (Local Podman Registry)
    |
    v
Container Runtime (Podman on 172.16.30.45)
    |
    v
Running Service (radgateway01-app container)
```

### Container Specification

```yaml
Container Image:
  - Base: alpine:latest
  - Binary: /usr/local/bin/rad-gateway (COPY from builder)
  - Workdir: /app
  - User: radgateway (non-root)

Resources:
  - Memory: Limited via cgroups
  - CPU: Managed via systemd slice
  - Network: Bridge network, port 8090 exposed

Volumes:
  - /opt/radgateway01/data -> /app/data
  - /opt/radgateway01/logs -> /app/logs
  - /opt/radgateway01/config/infisical-token -> /app/.infisical-token

Runtime:
  - Runtime: Podman 5.6.0+
  - Init: Systemd unit file (radgateway01.service)
  - Health: Container healthcheck with /health endpoint
```

### Dockerfile Pattern

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o rad-gateway ./cmd/rad-gateway

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
RUN addgroup -S radgateway && adduser -S radgateway -G radgateway
WORKDIR /app
COPY --from=builder /app/rad-gateway /usr/local/bin/rad-gateway
RUN chmod +x /usr/local/bin/rad-gateway
USER radgateway
ENTRYPOINT ["/usr/local/bin/rad-gateway"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8090/health || exit 1
```

---

## Forbidden Actions

The following actions are explicitly FORBIDDEN on 172.16.30.45:

### Binary Installation

- [x] Copying `rad-gateway` to `/opt/`, `/usr/local/bin/`, or any path on host
- [x] Installing via `make install` that places binaries on host
- [x] Creating soft links to binaries in host directories
- [x] Executing binary directly from `/opt/`, `/data/`, or other host paths

### Systemd Direct Executions

- [x] Creating systemd services with `ExecStart=/path/to/rad-gateway`
- [x] Running binary as daemon directly on host
- [x] Adding binary to init systems that execute on host

### Package Installation

- [x] Creating .rpm or .deb packages for host installation
- [x] Installing via system package managers (dnf, apt, etc.)
- [x] Placing application files in system directories

### Configuration on Host

- [x] Writing config files to `/etc/` for the application
- [x] Placing data files in `/var/` for the application
- [x] Creating user accounts for direct binary execution

---

## Required Actions

### Before Deployment

1. **Verify container image exists**:
   ```bash
   sudo podman images | grep radgateway01
   ```

2. **Verify image contains binary**:
   ```bash
   sudo podman inspect radgateway01:latest
   # Check Env, Entrypoint, and Cmd
   ```

3. **Verify pod configuration**:
   ```bash
   sudo podman pod inspect radgateway01
   ```

### During Deployment

1. **Run container from image**:
   ```bash
   sudo podman run -d \
     --name radgateway01-app \
     --pod radgateway01 \
     --volume /opt/radgateway01/data:/app/data \
     --volume /opt/radgateway01/logs:/app/logs \
     --volume /opt/radgateway01/config/infisical-token:/app/.infisical-token:ro \
     localhost/radgateway01:latest
   ```

2. **Verify container is running**:
   ```bash
   sudo podman ps --pod
   ```

### After Deployment

1. **Verify application responds through container**:
   ```bash
   curl http://localhost:8090/health
   ```

2. **Verify NO direct binary process**:
   ```bash
   ps aux | grep rad-gateway
   # Should show conmon process, NOT direct rad-gateway
   ```

---

## Enforcement Mechanisms

### Code Review

All deployment-related changes must include:

- Dockerfile updates reviewed
- Podman command arguments verified
- systemd service file checked for container execution
- No direct binary paths in any configuration

### Automated Checks

Pre-deployment validation:

```bash
# Check if binary path is on host
if [ -f /opt/radgateway01/bin/rad-gateway ]; then
    echo "ERROR: Binary found on host path. Use container deployment only."
    exit 1
fi

# Check if systemd runs binary directly
if systemctl cat radgateway01 | grep -q "/opt/radgateway01/bin/rad-gateway"; then
    echo "ERROR: Direct binary execution in systemd. Use container only."
    exit 1
fi

# Verify container is used
if ! systemctl cat radgateway01 | grep -q "podman run\|podman start"; then
    echo "ERROR: No container execution in systemd. Use container only."
    exit 1
fi
```

### Incident Response

If direct installation is detected:

1. Immediately stop the service
2. Verify no data corruption
3. Uninstall the direct binary
4. Deploy using approved container method
5. Review process failure root cause

---

## Migration Path

If RAD Gateway was previously installed directly:

### Step 1: Data Backup

```bash
sudo /opt/radgateway01/bin/backup.sh
```

### Step 2: Stop Direct Installation

```bash
sudo systemctl stop radgateway01
```

### Step 3: Build Container Image

```bash
cd /mnt/ollama/git/RADAPI01
sudo podman build -t radgateway01:latest .
```

### Step 4: Deploy Container

```bash
cd /mnt/ollama/git/RADAPI01/deploy
sudo ./install.sh
```

### Step 5: Verify

```bash
sudo systemctl start radgateway01
curl http://localhost:8090/health
```

---

## Exceptions

**No exceptions are permitted**. Any request to deviate from container-only deployment must:

1. Be submitted in writing as an Exception Request
2. Include detailed technical justification
3. Include risk assessment
4. Include mitigation plan
5. Be approved by:
   - Team Alpha (Architecture) - Technical validity
   - Team Charlie (Security) - Security implications
   - Team Echo (Operations) - Operational impact
   - Team Hotel (Infrastructure) - Infrastructure constraints

### Exception Review Process

- Exception requests reviewed weekly
- Valid for maximum 7 days
- Must include deprecation plan
- Requires explicit approval from all four teams

---

## Compliance

### Auditing

定期 audits will verify:

- All deployments use containers
- No direct binary installations exist
- Compliance with this policy is maintained

### Reporting

- Metrics available in operations dashboard
- Monthly compliance reports to Team Alpha
- Non-compliance requires immediate remediation

### Consequences

- First violation: Warning + immediate remediation
- Second violation: Access revocation + escalation
- Third violation: Security incident review

---

## References

- Related ADR: ADR-010 (Deployment Architecture)
- Documentation: `DEPLOYMENT_CHECKLIST.md`
- Operations: `docs/operations/`
- Team Structure: `docs/team-structure-compliance.md`

---

## Change History

| Date | Version | Author | Description |
|------|---------|--------|-------------|
| 2026-02-19 | 1.0 | Team Alpha + Team Hotel | Initial policy creation |

---

**Policy Status**: Active
**Review Date**: 2026-03-19 (monthly review)
**Next Review**: 2026-03-19
