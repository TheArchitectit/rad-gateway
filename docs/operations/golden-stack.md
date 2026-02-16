# Golden Stack: Secrets Management Infrastructure

**Status**: Production Ready
**Owner**: Team Hotel (Deployment & Infrastructure)
**Target Environment**: radgateway01 (Alpha)
**Last Updated**: 2026-02-16

---

## Overview

The Golden Stack provides a defense-in-depth secrets management infrastructure combining three production-grade components:

| Component | Purpose | Role |
|-----------|---------|------|
| **PostgreSQL 16** | Persistent storage | Shared database backend |
| **Infisical** | Active secrets management | Hot vault for operational secrets |
| **OpenBao** | Long-term retention | Cold vault for compliance/archive |

---

## Architecture Diagram

```
┌───────────────────────────────────────────────────────────────────────────────┐
│                           Host: radgateway01                                  │
│                                                                               │
│   ┌─────────────────────────────────────────────────────────────────────┐    │
│   │                        Pod: secret-stack                           │    │
│   │                    (Shared Network Namespace)                      │    │
│   │                                                                     │    │
│   │   ┌─────────────────┐                                               │    │
│   │   │   PostgreSQL 16 │  Port: 5432 (internal only)                 │    │
│   │   │   ┌──────────┐  │                                               │    │
│   │   │   │ infisical│  │  Database: infisical_db                       │    │
│   │   │   │  _db     │  │  User: infisical                             │    │
│   │   │   └──────────┘  │                                               │    │
│   │   │   ┌──────────┐  │                                               │    │
│   │   │   │ openbao  │  │  Database: openbao_db                        │    │
│   │   │   │  _db     │  │  User: openbao                               │    │
│   │   │   └──────────┘  │                                               │    │
│   │   └────────┬────────┘                                               │    │
│   │            │                                                        │    │
│   │   ┌────────┴────────┐                                               │    │
│   │   │    Infisical    │  Port: 8080 (published)                     │    │
│   │   │    ┌────────┐   │  Purpose: Primary secrets management         │    │
│   │   │    │   UI   │   │  Access: http://radgateway01:8080           │    │
│   │   │    │  API   │   │                                               │    │
│   │   │    └────────┘   │  Role: Hot vault (active secrets)           │    │
│   │   └────────┬────────┘                                               │    │
│   │            │                                                        │    │
│   │   ┌────────┴────────┐                                               │    │
│   │   │    OpenBao      │  Port: 8200 (published)                     │    │
│   │   │    ┌────────┐   │  Purpose: Long-term secrets storage         │    │
│   │   │    │   UI   │   │  Access: http://radgateway01:8200           │    │
│   │   │    │  API   │   │                                               │    │
│   │   │    └────────┘   │  Role: Cold vault (compliance/audit)        │    │
│   │   └─────────────────┘                                               │    │
│   │                                                                     │    │
│   └─────────────────────────────────────────────────────────────────────┘    │
│                              │                                                │
│         ┌────────────────────┼────────────────────┐                          │
│         │                    │                    │                          │
│         ▼                    ▼                    ▼                          │
│   ┌──────────┐         ┌──────────┐         ┌──────────┐                    │
│   │   API    │         │   API    │         │   API    │                    │
│   │  :8080   │         │  :8090   │         │  :8200   │                    │
│   └──────────┘         └──────────┘         └──────────┘                    │
│   Infisical API        RAD Gateway         OpenBao API                    │
│                                                                               │
└───────────────────────────────────────────────────────────────────────────────┘
```

---

## Why This Stack

### The Dual-Vault Pattern

The Golden Stack implements a **dual-vault pattern** that separates operational concerns from compliance requirements:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Secrets Lifecycle                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────┐              ┌─────────────┐                 │
│   │  Creation   │─────────────▶│  Active Use │                 │
│   └─────────────┘              └──────┬──────┘                 │
│          │                            │                        │
│          │                            │                        │
│          ▼                            ▼                        │
│   ┌─────────────────────────────────────────┐                  │
│   │           Infisical (Hot Vault)          │                  │
│   │  • Fast access (sub-10ms read)          │                  │
│   │  • Service token authentication         │                  │
│   │  • Environment-based organization       │                  │
│   │  • Version control with 10 versions     │                  │
│   └─────────────────────────────────────────┘                  │
│                    │                                            │
│                    │ Async replication (optional)               │
│                    ▼                                            │
│   ┌─────────────────────────────────────────┐                  │
│   │           OpenBao (Cold Vault)           │                  │
│   │  • Immutable audit logs                 │                  │
│   │  • Long-term retention (10 years)       │                  │
│   │  • Compliance reporting                 │                  │
│   │  • Disaster recovery                    │                  │
│   └─────────────────────────────────────────┘                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Infisical: The Hot Vault

**Use Case**: Operational secrets that applications need to access frequently

**Characteristics**:
- Low-latency reads for application startup
- Service token-based authentication
- Project and environment organization
- Real-time secret updates
- Integration-friendly API

**Examples**:
- RAD_API_KEYS for gateway authentication
- Provider API keys (OpenAI, Anthropic, Gemini)
- Database connection strings
- Service-to-service credentials

### OpenBao: The Cold Vault

**Use Case**: Compliance, audit, and long-term retention

**Characteristics**:
- Immutable audit logging
- Cryptographic sealing/unsealing
- Shamir secret sharing for key management
- Long-term lease TTL (10 years)
- Version history beyond Infisical limits

**Examples**:
- Historical secret versions for compliance
- Audit trail of all secret operations
- Disaster recovery backups
- Regulatory compliance archives

---

## Component Interactions

### Data Flow

```
┌───────────────────────────────────────────────────────────────────────────────┐
│                           Data Flow Patterns                                   │
├───────────────────────────────────────────────────────────────────────────────┤
│                                                                               │
│  Pattern 1: Application Secret Retrieval                                       │
│  ─────────────────────────────────────────                                     │
│                                                                               │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐                │
│  │  RAD     │───▶│ Infisical│───▶│  Service │───▶│ Secrets  │                │
│  │ Gateway  │    │  :8080   │    │  Token   │    │ Returned │                │
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘                │
│       │                                               │                       │
│       │                                               │                       │
│       ▼                                               ▼                       │
│  ┌─────────────────────────────────────────────────────────┐                 │
│  │              Secrets Loaded into Memory                  │                 │
│  │  RAD_API_KEYS, OPENAI_API_KEY, ANTHROPIC_API_KEY, etc. │                 │
│  └─────────────────────────────────────────────────────────┘                 │
│                                                                               │
│  Pattern 2: Compliance Audit                                                   │
│  ─────────────────────────────────                                             │
│                                                                               │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐                │
│  │  Auditor │───▶│ OpenBao  │───▶│  Admin   │───▶│  Audit   │                │
│  │  User    │    │  :8200   │    │  Token   │    │   Logs   │                │
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘                │
│       │                                               │                       │
│       │                                               ▼                       │
│       │                                        ┌──────────┐                  │
│       │                                        │ Historical│                  │
│       └───────────────────────────────────────▶│ Secrets   │                  │
│                                                └──────────┘                  │
│                                                                               │
│  Pattern 3: Shared Storage                                                     │
│  ─────────────────────────                                                     │
│                                                                               │
│  ┌───────────┐         ┌──────────────┐         ┌───────────┐                │
│  │ Infisical │────────▶│  PostgreSQL  │◀────────│  OpenBao  │                │
│  │           │         │   :5432      │         │           │                │
│  └───────────┘         └──────────────┘         └───────────┘                │
│                            │                                                  │
│                            ▼                                                  │
│                    ┌──────────────┐                                           │
│                    │  Separate    │                                           │
│                    │  Databases   │                                           │
│                    │  • infisical │                                           │
│                    │  • openbao   │                                           │
│                    └──────────────┘                                           │
│                                                                               │
└───────────────────────────────────────────────────────────────────────────────┘
```

### API Endpoints

| Component | Endpoint | Purpose | Authentication |
|-----------|----------|---------|----------------|
| Infisical | `GET /api/status` | Health check | None |
| Infisical | `GET /api/v3/secrets/raw/{key}` | Retrieve secret | Service Token |
| Infisical | `POST /api/v3/secrets/raw/{key}` | Create/update secret | Service Token |
| OpenBao | `GET /v1/sys/health` | Health check | None |
| OpenBao | `GET /v1/sys/seal-status` | Seal status | None |
| OpenBao | `POST /v1/sys/unseal` | Unseal vault | Shamir shards |
| OpenBao | `GET /v1/infisical-archive/data/{path}` | Read archived secret | Token |
| PostgreSQL | `localhost:5432` | Database connections | Password |

---

## Security Model

### Defense in Depth

```
┌─────────────────────────────────────────────────────────────────┐
│                    Security Layers                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Layer 1: Network Isolation                                     │
│  ┌─────────────────────────────────────────────────────┐       │
│  │  • PostgreSQL: localhost only, no external exposure  │       │
│  │  • Infisical: internal network, firewall restricted  │       │
│  │  • OpenBao: admin network only, firewall restricted  │       │
│  └─────────────────────────────────────────────────────┘       │
│                                                                 │
│  Layer 2: Authentication                                        │
│  ┌─────────────────────────────────────────────────────┐       │
│  │  • PostgreSQL: SCRAM-SHA-256 password auth          │       │
│  │  • Infisical: Service tokens + user authentication  │       │
│  │  • OpenBao: Token-based + Shamir unseal keys        │       │
│  └─────────────────────────────────────────────────────┘       │
│                                                                 │
│  Layer 3: Encryption                                            │
│  ┌─────────────────────────────────────────────────────┐       │
│  │  • PostgreSQL: SSL/TLS connections                  │       │
│  │  • Infisical: AES-256-GCM for secrets at rest       │       │
│  │  • OpenBao: Shamir seal encryption for all data     │       │
│  └─────────────────────────────────────────────────────┘       │
│                                                                 │
│  Layer 4: Audit Logging                                         │
│  ┌─────────────────────────────────────────────────────┐       │
│  │  • PostgreSQL: Query logging enabled                │       │
│  │  • Infisical: Access logs for all operations        │       │
│  │  • OpenBao: Immutable audit log for all requests    │       │
│  └─────────────────────────────────────────────────────┘       │
│                                                                 │
│  Layer 5: Access Control                                        │
│  ┌─────────────────────────────────────────────────────┐       │
│  │  • PostgreSQL: Role-based database permissions      │       │
│  │  • Infisical: Project-based access control          │       │
│  │  • OpenBao: Path-based policies with fine-grained   │       │
│  │    permissions                                      │       │
│  └─────────────────────────────────────────────────────┘       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Threat Model

| Threat | Mitigation | Component |
|--------|------------|-----------|
| Database breach | Separate DB users, limited permissions | PostgreSQL |
| Secret exfiltration | Service tokens with scoped access | Infisical |
| Unauthorized access | Network isolation, firewall rules | All |
| Audit tampering | Immutable audit logs in OpenBao | OpenBao |
| Key compromise | Shamir secret sharing (3 of 5) | OpenBao |
| Memory dump | Short-lived tokens, sealed vault | OpenBao |
| Insider threat | Separation of duties, audit logging | All |

### Secret Classification

| Classification | Storage | TTL | Rotation | Example |
|----------------|---------|-----|----------|---------|
| Critical | OpenBao + Infisical | 90 days | 30 days | Root CA keys |
| High | Infisical + OpenBao archive | 1 year | 90 days | Provider API keys |
| Medium | Infisical only | 1 year | 180 days | Service tokens |
| Low | Infisical | 2 years | 365 days | Config secrets |

---

## Operational Responsibilities

| Task | Tool | Frequency | Owner |
|------|------|-----------|-------|
| Secret rotation | Infisical UI/API | 30-90 days | Security Team |
| Access review | Infisical audit logs | Monthly | Security Team |
| Backup verification | OpenBao + PostgreSQL | Weekly | Operations |
| Seal status check | OpenBao API | Daily | Operations |
| Compliance export | OpenBao audit logs | Quarterly | Compliance |
| Disaster recovery drill | Full stack | Annually | Operations |

---

## Documentation References

- [Golden Stack Deployment](golden-stack-deployment.md) - Step-by-step deployment guide
- [Golden Stack Operations](golden-stack-operations.md) - Daily operations and procedures
- [Common Issues Runbook](common-issues-runbook.md) - Troubleshooting guide

---

**Document Owner**: Team Hotel (Deployment & Infrastructure)
**Review Schedule**: Monthly during active development, quarterly in maintenance
**Next Review**: 2026-03-16
