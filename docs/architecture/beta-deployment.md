# Beta Deployment Architecture

**Status**: Current (v0.2.0-alpha)  
**Target**: Beta Release  
**Last Updated**: 2026-02-16

---

## Overview

The beta deployment uses a simplified secrets management approach with **Infisical only**. OpenBao is deployed but reserved for post-beta cold vault requirements.

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      BETA STACK                          │
│                                                          │
│  ┌──────────────┐    ┌──────────────┐    ┌─────────────┐ │
│  │  RAD Gateway │───▶│  Infisical   │───▶│  PostgreSQL │ │
│  │   :8090      │    │   :8080      │    │   :5432     │ │
│  └──────────────┘    └──────────────┘    └─────────────┘ │
│                              │                           │
│                              ▼                           │
│                       ┌──────────────┐                   │
│                       │ Secrets Mgmt │                   │
│                       │ - API Keys   │                   │
│                       │ - DB Creds   │                   │
│                       │ - Provider   │                   │
│                       │   Tokens     │                   │
│                       └──────────────┘                   │
└─────────────────────────────────────────────────────────┘
```

---

## Services

| Service | Purpose | Port | Status |
|---------|---------|------|--------|
| **radgateway01** | API Gateway | 8090 | ✅ Active |
| **Infisical** | Secrets Management | 8080 | ✅ Active |
| **PostgreSQL** | Database | 5432 | ✅ Active |
| **OpenBao** | Cold Vault (Future) | 8200 | ⚠️ Reserved |
| **Redis** | Cache | 6379 | ✅ Active |

---

## Secrets Management (Beta)

### Infisical Handles:
- ✅ Provider API keys (OpenAI, Anthropic, Gemini)
- ✅ PostgreSQL credentials
- ✅ JWT secrets
- ✅ Encryption keys
- ✅ Service tokens

### OpenBao Reserved For:
- ⏸️ Cold vault archival (5+ year retention)
- ⏸️ Compliance audit trails
- ⏸️ Advanced PKI features
- ⏸️ Post-beta requirements

---

## Access URLs

| Service | URL | Purpose |
|---------|-----|---------|
| RAD Gateway Health | http://172.16.30.45:8090/health | Gateway status |
| Infisical UI | http://172.16.30.45:8080 | Secrets management |
| OpenBao UI | http://172.16.30.45:8200 | **Reserved** |

---

## Configuration

### RAD Gateway Secrets Path in Infisical
```
/rad-gateway/
├── providers/
│   ├── openai/
│   │   └── api-key
│   ├── anthropic/
│   │   └── api-key
│   └── gemini/
│       └── api-key
├── database/
│   └── postgres-url
└── gateway/
    ├── jwt-secret
    └── encryption-key
```

---

## Notes

- OpenBao is **not configured** for active use in beta
- All secrets flow through Infisical only
- OpenBao can be enabled post-beta for compliance requirements
- This keeps beta deployment simple and maintainable

See: [Golden Stack Documentation](../operations/golden-stack.md) for full deployment details.
