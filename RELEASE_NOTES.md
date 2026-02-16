# Release Notes - v0.3.0-alpha

**Release Date**: 2026-02-16
**Status**: Alpha
**Codename**: Brass Relay Team Review

---

## Overview

RAD Gateway v0.3.0-alpha completes the Team India security review and implements all P0/P1 fixes for the beta deployment. This release includes the Gemini adapter, comprehensive deployment documentation, and beta-ready architecture.

---

## What's New

### Gemini Adapter (Complete)

- **Full Implementation**: 2,282 lines of production-ready code
- **Test Coverage**: 36 comprehensive tests (1,357 lines)
- **Features**: x-goog-api-key auth, streaming support, role mapping
- **Factory Integration**: Registered with provider factory

### Team India Security Review (Complete)

Comprehensive multi-disciplinary review with all fixes implemented:

**P0 Critical Fixes** (2 items):
- PostgreSQL port exposure removed
- Backup procedures documented

**P1 High Fixes** (6 items):
- Backup exclusion documentation
- Error handling in startup scripts
- Monitoring/Alerting procedures
- Deployment/Rollback procedures
- Pre-deployment validation script
- Health check integration tests

### Beta Deployment Architecture

Complete beta deployment documentation:
- Architecture overview with ASCII diagrams
- Service dependencies and access URLs
- Backup and restore procedures
- Monitoring and alerting setup
- Beta vs Production skip list

---

## Previous Release: v0.2.0-alpha

# Release Notes - v0.2.0-alpha

**Release Date**: 2026-02-16
**Status**: Alpha
**Codename**: Brass Relay Extended

---

## Overview

RAD Gateway v0.2.0-alpha extends the Brass Relay API gateway with real provider adapters and the Golden Stack secrets management infrastructure. This release adds production-ready Anthropic adapter support and a complete secrets management stack.

---

## What's New

### Provider Adapters (Milestone 1 Complete)

- **OpenAI Adapter**: Full implementation with streaming support
- **Anthropic Adapter**: Complete adapter with tests (1,277 lines)
- **Gemini Adapter**: Design complete, implementation pending

### Golden Stack - Secrets Management Infrastructure

Complete secrets management deployment with three integrated services:

- **PostgreSQL 16**: Shared persistence layer for secrets
- **Infisical**: Active secrets management (hot vault)
- **OpenBao**: Long-term secrets storage (cold vault, 5-10 year TTL)

### Deployment Automation

- **Golden Stack Deploy Script**: `./deploy/golden-stack/deploy.sh`
- **Environment Configuration**: Automatic secret generation
- **Systemd Integration**: Production service management
- **Health Monitoring**: Container health checks for all services

---

## Previous Release: v0.1.0-alpha

See below for v0.1.0-alpha release notes.

---

# Release Notes - v0.1.0-alpha

**Release Date**: 2026-02-16
**Status**: Alpha
**Codename**: Brass Relay

---

## Overview

RAD Gateway v0.1.0-alpha is the first public release of the Brass Relay API gateway. This release establishes the foundational runtime with multi-provider compatibility, Team Hotel deployment infrastructure, and production-ready container orchestration.

---

## What's New

### Core Gateway Runtime

- **Multi-Provider API Compatibility**: OpenAI, Anthropic, and Gemini-compatible request surfaces
- **HTTP Server**: Production-hardened server with configurable timeouts (Read: 15s, Write: 30s, Idle: 60s)
- **Authentication**: Multi-format API key support (Bearer, x-api-key, x-goog-api-key, query param)
- **Health Endpoint**: `GET /health` returns `{"status":"healthy"}` for monitoring
- **Management APIs**: Configuration and usage introspection via `/v0/management/*`

### API Endpoints

| Method | Endpoint | Status | Description |
|--------|----------|--------|-------------|
| GET | `/health` | Available | Health check endpoint |
| POST | `/v1/chat/completions` | Available | OpenAI-compatible chat completions |
| POST | `/v1/responses` | Available | Response generation endpoint |
| POST | `/v1/messages` | Available | Anthropic-compatible messages |
| POST | `/v1/embeddings` | Available | Text embeddings |
| POST | `/v1/images/generations` | Available | Image generation |
| POST | `/v1/audio/transcriptions` | Available | Audio transcription |
| GET | `/v1/models` | Available | List available models |
| POST | `/v1beta/models/{model}:{action}` | Available | Gemini-compatible operations |
| GET | `/v0/management/config` | Available | Runtime configuration view |
| GET | `/v0/management/usage` | Available | Usage statistics |
| GET | `/v0/management/traces` | Available | Request traces |

### Team Hotel - Deployment & Infrastructure (5 Members)

Complete deployment automation for radgateway01:

- **Container Orchestration**: Podman-based deployment with systemd integration
- **Infisical Integration**: Automated secret fetching at startup
- **Installation Script**: One-command deployment (`./install.sh`)
- **Backup Automation**: Automated backups with 7-day retention
- **Health Monitoring**: Container health checks and systemd integration
- **Security Hardening**: Non-root service user, read-only root filesystem, systemd security directives
- **Network Configuration**: Firewall rules and iptables FORWARD rules for container networking

### Deployment Artifacts

| Artifact | Location | Purpose | Status |
|----------|----------|---------|--------|
| `deploy/install.sh` | `/mnt/ollama/git/RADAPI01/deploy/` | One-command installer | Production-ready |
| `deploy/uninstall.sh` | `/mnt/ollama/git/RADAPI01/deploy/` | Clean removal | Production-ready |
| `deploy/bin/startup.sh` | `/opt/radgateway01/bin/` | Infisical secret fetching | Production-ready |
| `deploy/bin/health-check.sh` | `/opt/radgateway01/bin/` | Comprehensive health monitoring | Production-ready |
| `deploy/bin/backup.sh` | `/opt/radgateway01/bin/` | Backup automation | Production-ready |
| `deploy/systemd/radgateway01.service` | `/etc/systemd/system/` | Service management | Production-ready |
| `deploy/config/env` | `/opt/radgateway01/config/` | Environment configuration | Production-ready |
| `RUNBOOK.md` | Repository root | Operations guide | Complete |
| `DEPLOYMENT_CHECKLIST.md` | Repository root | Deployment verification | Complete |

### Platform

- **Go Version**: 1.24 (toolchain 1.24.13)
- **Base Image**: Alpine Linux
- **Container Runtime**: Podman 5.6.0+
- **Service Management**: systemd
- **Secrets Management**: Infisical

---

## Breaking Changes

None - this is the initial alpha release.

---

## Known Issues

1. **Provider Adapters**: Currently uses mock adapter; real provider integrations in progress
2. **Database**: Usage/trace persistence is in-memory only; PostgreSQL integration planned
3. **Metrics**: Prometheus metrics endpoint stubbed but not fully implemented
4. **A2A Protocol**: Agent-to-agent endpoints planned for next phase

---

## Deployment Notes

### Prerequisites

- RHEL/Ubuntu with systemd
- Podman 5.6.0+
- Infisical running on localhost:8080 (for secrets)
- Root access for installation

### Quick Deploy

```bash
cd /mnt/ollama/git/RADAPI01/deploy
sudo ./install.sh

# Add Infisical token
echo "your-token" | sudo tee /opt/radgateway01/config/infisical-token
sudo chmod 600 /opt/radgateway01/config/infisical-token

# Start service
sudo systemctl start radgateway01

# Verify
curl http://localhost:8090/health
```

### Network Configuration

The installer automatically configures:
- Firewall rule for port 8090/tcp
- iptables FORWARD rules for Podman container networking
- Proper routing between host and container networks

---

## Security

### Implemented

- Non-root service user (`radgateway`)
- Read-only root filesystem
- Systemd security hardening (NoNewPrivileges, ProtectSystem, ProtectHome)
- Secure secret storage (Infisical with 600 permissions)
- Container network isolation

### In Progress

- TLS/HTTPS termination
- Rate limiting
- Audit logging

See [SECURITY.md](SECURITY.md) for details.

---

## Documentation

- [RUNBOOK.md](RUNBOOK.md) - Operations and troubleshooting
- [deploy/README.md](deploy/README.md) - Deployment guide
- [docs/operations/deployment-radgateway01.md](docs/operations/deployment-radgateway01.md) - Full deployment spec
- [docs/feature-matrix.md](docs/feature-matrix.md) - Feature parity tracking

---

## Team Structure (TEAM-007 Compliant)

| Team | Purpose | Status |
|------|---------|--------|
| Team Alpha | Architecture & Design | Active |
| Team Bravo | Core Implementation | Active |
| Team Charlie | Security Hardening | Active |
| Team Delta | Quality Assurance | Active |
| Team Echo | Operations & Observability | Active |
| **Team Hotel** | **Deployment & Infrastructure** | **Active (radgateway01)** |

### Team Hotel Members

- DevOps Lead - Infrastructure orchestration
- Container Engineer - Podman container management
- Deployment Engineer - Release automation
- Infrastructure Architect - Infrastructure design
- Systems Administrator - Host management

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 0.1.0-alpha | 2026-02-16 | Initial release with Team Hotel deployment |

---

## Upgrade Path

To upgrade from v0.1.0-alpha to future versions:

```bash
# Build new image
sudo podman build -t radgateway01:v0.2.0 .

# Rolling update
sudo systemctl stop radgateway01
sudo podman tag radgateway01:v0.2.0 radgateway01:latest
sudo systemctl start radgateway01

# Verify
curl http://localhost:8090/health
```

---

## Support

- **Team**: Team Hotel (Deployment & Infrastructure)
- **Escalation**: Team Echo (Operations & Observability)
- **Documentation**: [RUNBOOK.md](RUNBOOK.md)

---

## Changelog

### Added
- Initial Go-based gateway runtime
- Multi-provider API compatibility layer (OpenAI, Anthropic, Gemini)
- HTTP server with production timeouts
- API key authentication with multiple formats
- Health check endpoint
- Management APIs for config, usage, and traces
- Team Hotel deployment automation
- Podman container orchestration
- Systemd service integration
- Infisical secrets integration
- Automated backup script
- Firewall and network configuration
- RUNBOOK.md operations guide
- Comprehensive deployment documentation

### Fixed
- HTTP server timeout configuration
- PR review blockers resolved

### Changed
- Pinned Go toolchain to 1.24.13
- Updated LICENSE and contribution guidelines

---

**Full Changelog**: https://github.com/TheArchitectit/rad-gateway/compare/...v0.1.0-alpha
