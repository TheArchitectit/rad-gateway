# Changelog

All notable changes to RAD Gateway (Brass Relay) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0-alpha] - 2026-02-16

### Added
- Initial alpha release of RAD Gateway (Brass Relay)
- Core HTTP server with configurable timeouts (ReadHeader: 5s, Read: 15s, Write: 30s, Idle: 60s)
- OpenAI-compatible API surface (`/v1/chat/completions`, `/v1/models`, etc.)
- Anthropic-compatible message endpoints (`/v1/messages`)
- Gemini-compatible endpoints (`/v1beta/models/{model}:{action}`)
- Request routing with weighted candidate selection
- Retry budget mechanism (default: 2 retries)
- In-memory usage tracking and trace storage
- Management endpoints (`/v0/management/config`, `/v0/management/usage`, `/v0/management/traces`)
- API key authentication with multiple header support (Authorization Bearer, x-api-key, x-goog-api-key)
- Health check endpoint (`/health`)
- Infisical secrets integration support
- Podman container deployment configuration
- Systemd service integration

### Security
- Environment-based secret loading (no hardcoded credentials)
- Request ID and trace ID injection for audit trails
- Conditional authentication (health/management endpoints public)
- Security policy documentation

### Infrastructure
- Multi-stage Dockerfile with Go 1.24 and Alpine
- Podman pod-based container orchestration
- Systemd service with restart policies
- Firewall configuration documentation
- Resource limit recommendations

### Documentation
- Architecture documentation (Modular Monolith pattern)
- Deployment target specifications (local, alpha, staging, production)
- Team structure compliance (TEAM-007)
- Security policy and vulnerability reporting
- Operations runbook (excluded from public release - contains private network details)

### Development
- GitHub Actions CI pipeline (tests, coverage, security scanning)
- Govulncheck and Gosec security scanning
- Guardrails integration for compliance
- Test coverage reporting

## [Unreleased]

### Planned
- A2A (Agent-to-Agent) protocol support
- AG-UI (Agent-UI) event streaming
- Persistent storage (PostgreSQL) for usage/trace data
- Prometheus metrics endpoint
- Structured logging
- Configuration file support (YAML)

---

**Note**: This is an alpha release intended for internal testing and validation. APIs and configuration may change in future releases.
