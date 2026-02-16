# RAD Gateway Container Architecture

## Overview

RAD Gateway uses a containerized deployment model based on Podman, organized into logical pods that group related containers. This architecture provides resource isolation, simplified networking, and consistent deployment patterns.

## Pod-Based Architecture

### Design Principles

1. **Pod as Deployment Unit**: Containers that share concerns are grouped into pods
2. **Shared Networking**: Containers within a pod share network namespace (localhost communication)
3. **Shared Storage**: Named volumes for persistent data
4. **Lifecycle Management**: Pods manage container startup/shutdown order

### radgateway01 Pod Structure

```
┌─────────────────────────────────────────────────────────────┐
│  Pod: radgateway01                                          │
│  Network: bridge (shared namespace)                         │
│  Published Port: 8090 → 8090                               │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Container: radgateway01-app                        │   │
│  │  - Image: localhost/radgateway01:latest            │   │
│  │  - Port: 8090 (API endpoint)                       │   │
│  │  - Volume: radgateway01-data:/data                 │   │
│  │  - Restart: unless-stopped                         │   │
│  │  - Health: HTTP check on :8090/health              │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Container: radgateway01-postgres (future)         │   │
│  │  - Image: postgres:16-alpine                       │   │
│  │  - Port: 5432 (internal only)                      │   │
│  │  - Volume: radgateway01-postgres-data              │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Container Images

### Base Image Strategy

| Layer | Base Image | Purpose |
|-------|------------|---------|
| Builder | `golang:1.24-alpine` | Compile Go application |
| Runtime | `alpine:latest` | Minimal runtime environment |

### Image Build Process

```dockerfile
# Multi-stage build
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o rad-gateway ./cmd/rad-gateway

FROM alpine:latest
RUN apk --no-cache add ca-certificates curl jq
WORKDIR /root/
COPY --from=builder /app/rad-gateway /usr/local/bin/
EXPOSE 8090
CMD ["/usr/local/bin/rad-gateway"]
```

## Networking Model

### Podman Network Types

| Type | Use Case | Isolation |
|------|----------|-----------|
| bridge | Default pod networking | Containers share network namespace |
| host | Direct host network access | No network isolation |
| slirp4netns | Rootless networking | User-mode networking |

### Port Mapping

```
Host Port → Container Port
   8090    →    8090      (radgateway01-app API)
```

### Inter-Container Communication

Within a pod, containers communicate via localhost:

```
radgateway01-app → curl http://localhost:5432 (postgres, future)
```

### External Service Access

Containers access external services via host networking:

```
radgateway01-app → Infisical on host:8080
```

## Storage Architecture

### Volume Types

| Volume | Purpose | Persistence |
|--------|---------|-------------|
| radgateway01-data | Application data, usage logs | Persistent |
| radgateway01-postgres-data | Database files (future) | Persistent |

### Volume Mount Points

```
Host Path                      → Container Path
/var/lib/containers/storage/... → /data (in container)
```

## Security Model

### Container Security

1. **Non-root Execution**: Service runs as dedicated user
2. **Read-only Root**: Root filesystem mounted read-only
3. **Capability Dropping**: Minimal Linux capabilities
4. **Seccomp**: Default seccomp profile applied

### Network Security

1. **Port Exposure**: Only required ports published
2. **Firewall Integration**: Host firewall (firewalld) controls access
3. **Internal Services**: Database ports not exposed externally

### Secret Management

1. **Runtime Injection**: Secrets loaded from Infisical at startup
2. **No Secrets in Images**: Build process excludes credentials
3. **Environment Isolation**: Secrets scoped to container environment

## Health Monitoring

### Container Health Checks

```bash
# Application health
--health-cmd "curl -f http://localhost:8090/health || exit 1"
--health-interval 30s
--health-timeout 10s
--health-retries 3
```

### Pod Status Indicators

| Status | Meaning |
|--------|---------|
| Created | Pod initialized, containers starting |
| Running | All containers healthy |
| Degraded | Some containers unhealthy |
| Exited | All containers stopped |

## Operational Commands

### Pod Management

```bash
# Create pod
podman pod create --name radgateway01 --publish 8090:8090

# View pod status
podman pod ps
podman pod inspect radgateway01

# Stop pod
podman pod stop radgateway01

# Remove pod
podman pod rm radgateway01
```

### Container Management

```bash
# View containers
podman ps --pod

# View logs
podman logs radgateway01-app
podman logs -f radgateway01-app  # follow

# Execute commands
podman exec -it radgateway01-app sh

# Inspect container
podman inspect radgateway01-app
```

## Scaling Considerations

### Vertical Scaling

- Increase container memory/CPU limits
- Optimize application resource usage
- Database connection pooling

### Horizontal Scaling

- Deploy additional pods (radgateway02, radgateway03)
- Use different host ports (8091, 8092)
- Load balancer distribution (future)

## Integration Points

### External Dependencies

| Service | Access Method | Purpose |
|---------|---------------|---------|
| Infisical | Host network (localhost:8080) | Secret retrieval |
| PostgreSQL | Future: container network | Data persistence |
| Redis | Future: cache layer | Session/cache storage |

### Reverse Proxy Integration

Future deployments will use Traefik for:
- SSL termination
- Path-based routing
- Load balancing across instances

## Version Compatibility

### Tested Versions

| Component | Version | Notes |
|-----------|---------|-------|
| Podman | 5.6.0 | RHEL deployment target |
| Alpine | 3.19+ | Runtime base image |
| Go | 1.24+ | Builder image |

## References

- [Deployment Specification](/docs/operations/deployment-radgateway01.md)
- [Dockerfile Best Practices](/docs/container/dockerfile-best-practices.md)
- [Lessons Learned](/docs/container/lessons-learned.md)
