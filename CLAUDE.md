# CLAUDE.md - RAD Gateway Project Guardrails

MANDATORY: Read this document before performing ANY work on the RAD Gateway codebase.

---

## Project Overview

- **Name**: RAD Gateway (codename: Brass Relay)
- **Type**: Go-based AI API Gateway
- **Repository**: /mnt/ollama/git/RADAPI01
- **Primary Deployment Target**: 172.16.30.45:8090
- **Deployment Method**: Podman container (ONLY)

---

## Deployment Guardrails

### Container-Only Requirement

CRITICAL: Application MUST run in Podman container. Direct VM installation is FORBIDDEN.

- **REQUIRED**: Application MUST run in Podman container
- **FORBIDDEN**: Installing binary directly on VM (/opt/..., /data/..., etc.)
- **BINARY MUST**: Be built INTO container image via Dockerfile COPY directive
- **All data/logs**: Must use container volumes (--volume)

### Rationale

Container-only deployment is enforced for the following reasons:

1. **Isolation**: Processes run in isolated namespace, preventing conflicts
2. **Reproducibility**: Container image provides immutable runtime environment
3. **Security**: Container boundaries provide additional security layer
4. **Portability**: Environment can be recreated consistently anywhere
5. **Easy Rollback**: Previous container images can be restored quickly

### Deployment Flow

1. **Build binary locally**:
   ```bash
   CGO_ENABLED=0 go build -o rad-gateway ./cmd/rad-gateway
   ```

2. **Copy to VM temporary**:
   ```bash
   scp rad-gateway user001@172.16.30.45:/tmp/
   ```

3. **Build image with binary INSIDE container**:
   ```dockerfile
   FROM golang:1.21-alpine AS builder
   # ... build steps ...

   FROM alpine:latest
   COPY --from=builder rad-gateway /usr/local/bin/rad-gateway
   ```

4. **Run container**:
   ```bash
   sudo podman run -d --name radgateway01-app radgateway01:latest
   ```

### Forbidden Actions

Do NOT perform any of the following:

- Copying `rad-gateway` binary to `/opt/`, `/usr/local/bin/`, or any VM path
- Creating systemd services that execute the binary directly
- Installing the application as a package (rpm, deb, etc.)
- Mounting host directories as executable paths for the application

### Verification

Before creating a service or deployment, verify:

```bash
# Check if running in container
sudo podman ps --format "{{.Names}}" | grep radgateway01

# Verify image contains binary
sudo podman run --rm radgateway01:latest which rad-gateway
# Should return: /usr/local/bin/rad-gateway

# Verify NOT running as direct process
ps aux | grep rad-gateway
# Should show only container process, NOT direct binary
```

---

## Code Standards

### Go Development

- Follow Go Proverbs: https://go-proverbs.github.io/
- Use `gofmt` and `golangci-lint`
- All public APIs must have documentation
- Error handling must use `errors.Wrap` for context
- Context must be passed through all blocking operations

### Testing

- Test coverage must exceed 80% for new code
- Use table-driven tests
- Include race detector: `go test -race ./...`
- No external dependencies in unit tests (mock everything)

### Commits

- Conventional commit format: `type(scope): description`
- Types: feat, fix, docs, style, refactor, test, chore
- Example: `feat(streaming): add SSE support for streaming responses`

---

## Security Requirements

- Never commit secrets, tokens, or passwords
- Use Infisical for secret management
- All API endpoints must have authentication
- Rate limiting must be enabled
- Input validation on all external inputs

---

## Architecture Constraints

### Module Boundaries

- `internal/`: Private code, imports only from project
- `cmd/`: Application entry points
- `pkg/`: Public library code (if any)
- Cross-package dependencies must be justified in documentation

### Provider Adapters

- Must implement `Adapter` interface
- Support streaming via separate methods
- Include circuit breaker pattern
- Log all requests for audit trails

### Storage

- Use interface abstraction for storage
- In-memory store for alpha/dev
- PostgreSQL required for production
- Connection pooling mandatory

---

## Quality Gates

### Code Review

- All PRs require at least 1 approval
- Team Charlie security review for sensitive changes
- Team Delta QA review for test coverage requirements

### CI/CD

- All tests must pass
- Security scans must pass (gosec, govulncheck)
- Build must succeed on all platforms
- Container image must be created and scanned

### Deployment

- Team Hotel must verify deployment checklist
- Infisical secrets must be configured
- Health check must pass
- Rollback plan must be documented

---

## Related Documentation

- `.guardrails/pre-work-check.md` - Pre-work regression check
- `DEPLOYMENT_CHECKLIST.md` - Deployment verification
- `docs/architecture/` - Architecture documentation
- `docs/operations/` - Operational procedures
- `docs/team-structure-compliance.md` - Team structure

---

## Emergency Contacts

- Team Alpha (Architecture): Lead architectural decisions
- Team Bravo (Core): Implementation questions
- Team Charlie (Security): Security concerns
- Team Delta (QA): Test coverage and quality
- Team Echo (Operations): Deployment issues
- Team Hotel (Infra): Infrastructure matters

---

**Last Updated**: 2026-02-19
**Version**: 1.0
