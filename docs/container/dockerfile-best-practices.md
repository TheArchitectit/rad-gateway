# Dockerfile Best Practices Guide

## Overview

This guide documents best practices for building container images for RAD Gateway deployments. Following these practices ensures secure, efficient, and maintainable containers.

## Multi-Stage Builds

### Principle

Use multi-stage builds to separate build dependencies from runtime, resulting in smaller, more secure images.

### Pattern

```dockerfile
# Stage 1: Build
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o app ./cmd/app

# Stage 2: Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/app .
EXPOSE 8090
CMD ["./app"]
```

### Benefits

- **Smaller Images**: Runtime image contains only necessary binaries
- **Reduced Attack Surface**: Build tools not present in production
- **Faster Deploys**: Smaller images pull and start faster

## Base Image Selection

### Recommended Base Images

| Use Case | Base Image | Rationale |
|----------|------------|-----------|
| Go Apps | `alpine:latest` | Minimal, secure, 5MB |
| Go Apps (distroless) | `gcr.io/distroless/static` | Google hardened, no shell |
| Debug/Troubleshooting | `alpine:latest` + tools | Includes curl, jq |

### Image Tags Strategy

```dockerfile
# Good: Specific version
FROM golang:1.24-alpine

# Bad: Latest tag (non-reproducible)
FROM golang:latest

# Good: Specific Alpine version
FROM alpine:3.19
```

### Security Scanning

```bash
# Scan image for vulnerabilities
podman scan localhost/myapp:latest

# Or use Trivy
trivy image localhost/myapp:latest
```

## Layer Optimization

### Order Instructions by Change Frequency

```dockerfile
# Good: Least-frequent changes first
FROM alpine:3.19
RUN apk --no-cache add ca-certificates  # Changes rarely
WORKDIR /app
COPY config.yaml .                      # Changes occasionally
COPY app .                              # Changes frequently
CMD ["./app"]

# Bad: Frequent changes invalidate all subsequent layers
FROM alpine:3.19
COPY app .                              # Changes frequently
RUN apk --no-cache add ca-certificates  # Must re-run every build
COPY config.yaml .
CMD ["./app"]
```

### Combine Related RUN Commands

```dockerfile
# Good: Single layer for related operations
RUN apk --no-cache add \
    ca-certificates \
    curl \
    jq \
 && rm -rf /var/cache/apk/*

# Bad: Multiple layers for single logical operation
RUN apk add ca-certificates
RUN apk add curl
RUN apk add jq
```

### Leverage Build Cache

```dockerfile
# Good: Copy dependency files first
COPY go.mod go.sum ./
RUN go mod download  # Cached unless go.mod/go.sum changes
COPY . .
RUN go build ...

# Bad: Copy everything at once
COPY . .
RUN go mod download  # Runs on every code change
RUN go build ...
```

## Security Best Practices

### Run as Non-Root User

```dockerfile
# Create non-root user
RUN adduser -D -u 1000 appuser
USER appuser

# Or use numeric ID (best practice)
USER 1000
```

### Avoid Sensitive Data in Images

```dockerfile
# Bad: Never do this
ENV API_KEY=secret123
COPY secrets.txt /

# Good: Inject at runtime
ENV API_KEY=""
# Mount secrets as files or use secret injection
```

### Use COPY Instead of ADD

```dockerfile
# Good: Explicit and predictable
COPY app /usr/local/bin/
COPY config.yaml /etc/myapp/

# Bad: ADD has magic behavior (URL extraction, tar auto-extract)
ADD https://example.com/file.tar.gz /tmp/
ADD archive.tar.gz /dest/
```

### Read-Only Root Filesystem

```dockerfile
# Design for read-only root
FROM alpine:3.19
RUN adduser -D appuser
WORKDIR /app
COPY --chown=appuser:appuser app .
USER appuser
# Only write to explicit volumes
VOLUME ["/tmp", "/data"]
CMD ["./app"]
```

## Health Checks

### Container Health Checks

```dockerfile
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8090/health || exit 1
```

### Podman Health Check (Command Line)

```bash
podman run -d \
  --health-cmd "curl -f http://localhost:8090/health || exit 1" \
  --health-interval 30s \
  --health-timeout 10s \
  --health-retries 3 \
  myapp:latest
```

## Build Optimization

### Use .dockerignore

```gitignore
# .dockerignore
.git
.gitignore
*.md
.env
.env.example
Dockerfile
.dockerignore
node_modules/
vendor/
tmp/
*.log
```

### Minimize Image Size

```dockerfile
# Use static linking for Go
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -extldflags '-static'" \
    -a -installsuffix cgo \
    -o app ./cmd/app

# Clean up in same layer
RUN apk add --no-cache --virtual .build-deps gcc musl-dev \
    && go build ... \
    && apk del .build-deps
```
## LABEL Annotations

```dockerfile
LABEL org.opencontainers.image.title="RAD Gateway"
LABEL org.opencontainers.image.description="API Gateway for AI model routing"
LABEL org.opencontainers.image.version="0.1.0"
LABEL org.opencontainers.image.source="https://github.com/org/rad-gateway"
```

## Port Configuration

### Expose Documentation

```dockerfile
# Document intended ports (informational)
EXPOSE 8090/tcp

# Multiple ports
EXPOSE 8090/tcp 8091/tcp
```

Note: `EXPOSE` doesn't publish ports; use `-p` flag at runtime.

## Signal Handling

### Proper PID 1 Handling

```dockerfile
# Use exec form for proper signal handling
CMD ["/usr/local/bin/app"]

# Not shell form (creates shell subprocess)
CMD /usr/local/bin/app
```

### Handle SIGTERM Gracefully

```dockerfile
# Ensure application handles SIGTERM
# Add to systemd service or docker run:
# --stop-timeout 30
# --stop-signal SIGTERM
```

## Go-Specific Practices

### Optimal Go Build

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Download dependencies (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a \
    -installsuffix cgo \
    -o rad-gateway \
    ./cmd/rad-gateway

# Runtime stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates curl jq
WORKDIR /root/
COPY --from=builder /app/rad-gateway /usr/local/bin/
EXPOSE 8090
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
  CMD curl -f http://localhost:8090/health || exit 1
CMD ["/usr/local/bin/rad-gateway"]
```

### Build Flags Reference

| Flag | Purpose |
|------|---------|
| `-w` | Omit DWARF debug info |
| `-s` | Omit symbol table |
| `-extldflags "-static"` | Static linking |
| `CGO_ENABLED=0` | Disable CGO for static binary |

## Testing Images

### Local Testing

```bash
# Build image
podman build -t radgateway:test .

# Test run
podman run -d --name test -p 8090:8090 radgateway:test

# Verify
curl http://localhost:8090/health

# Inspect
podman exec test ps aux
podman exec test netstat -tlnp

# Cleanup
podman stop test && podman rm test
```

### Image Structure Test

```bash
# Check image layers
podman history radgateway:latest

# Check image size
podman images radgateway:latest

# Dive tool for detailed analysis
dive radgateway:latest
```

## Checklist

Before committing a Dockerfile, verify:

- [ ] Multi-stage build used where applicable
- [ ] Specific base image versions pinned
- [ ] Non-root user configured
- [ ] No secrets in image
- [ ] .dockerignore file present
- [ ] Health check configured
- [ ] Minimal layer count
- [ ] Labels added
- [ ] Port exposed/documented
- [ ] Image scanned for vulnerabilities

## References

- [Dockerfile Reference](https://docs.docker.com/engine/reference/builder/)
- [Best Practices for Writing Dockerfiles](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)
- [Podman Documentation](https://docs.podman.io/)
- [Distroless Images](https://github.com/GoogleContainerTools/distroless)
