# Backlog: Automatic Secret Management Integration

**ID**: BACKLOG-001
**Priority**: Medium
**Created**: 2026-02-16
**Updated**: 2026-02-16
**Status**: Post-Beta
**Target**: v0.4.0+ (Post-Beta)
**Sprint**: TBD

---

## Overview

**Beta Status**: Infisical-only for secrets management
**Post-Beta**: Full integration with Infisical + OpenBao

During beta, RAD Gateway uses **Infisical exclusively** for secrets management. This backlog item covers the **post-beta enhancement** to add automatic secret fetching and OpenBao cold vault integration.

### Beta Simplification

For beta, we intentionally simplified to reduce complexity:
- ✅ Infisical handles all secrets (API keys, DB creds, JWT, encryption)
- ✅ OpenBao deployed but **reserved** for post-beta
- ✅ `.env` files acceptable for beta deployments
- ✅ Manual secret rotation acceptable for beta

### Post-Beta Goals

After beta release, implement:
- Automatic secret fetching from Infisical at startup (no `.env` files)
- Secret rotation without gateway restart
- OpenBao cold vault archival for compliance (5+ year retention)
- Audit trails for all secret access
- Provider API key lifecycle management

---

## Goals

1. **Automatic Secret Fetching**: Gateway fetches API keys from Infisical at startup
2. **Secret Rotation**: Automatic rotation without restart
3. **Cold Vault Archival**: Archive old secrets to OpenBao for compliance
4. **Provider API Key Management**: Store/fetch OpenAI, Anthropic, Gemini keys securely
5. **Audit Trail**: Log all secret access to OpenBao audit log

---

## Implementation Ideas

### Option A: Sidecar Pattern
```
┌─────────────────┐     ┌──────────────┐     ┌──────────┐
│ RAD Gateway     │────▶│ Secret Agent │────▶│ Infisical│
│ (reads secrets) │     │ (sidecar)    │     └──────────┘
└─────────────────┘     └──────────────┘     ┌──────────┐
                                              │ OpenBao  │
                                              └──────────┘
```

### Option B: Direct SDK Integration
```go
// internal/secrets/manager.go

type SecretManager interface {
    GetProviderKey(provider string) (string, error)
    RotateKey(provider string) error
    ArchiveToColdVault(key string, metadata map[string]string) error
    WatchForChanges() (<-chan SecretEvent, error)
}

type InfisicalManager struct {
    client *infisical.Client
    vault  *openbao.Client
}
```

### Option C: Kubernetes Operator
- Custom Resource Definition (CRD) for secrets
- Operator watches CRDs and syncs to Infisical/OpenBao
- Gateway reads from Kubernetes secrets

---

## Technical Requirements

### Infisical Integration
- [ ] Service token authentication
- [ ] Secret path: `/rad-gateway/providers/{provider}/api-key`
- [ ] Automatic token refresh (JWT expires every 24h)
- [ ] Webhook support for real-time updates

### OpenBao Integration
- [ ] AppRole authentication for non-interactive access
- [ ] KV v2 secrets engine
- [ ] Audit logging to file
- [ ] Automatic unseal (Shamir's secret sharing)

### Gateway Changes
- [ ] Secrets manager interface
- [ ] Fallback to env vars if vaults unavailable
- [ ] Circuit breaker for vault access
- [ ] Metrics: secret fetch latency, rotation events

---

## Use Cases

1. **Provider Key Rotation**
   ```bash
   # Current: Manual update in .env
   echo "OPENAI_API_KEY=new-key" >> .env
   systemctl restart rad-gateway

   # Future: Automatic via Infisical
   infisical secrets set OPENAI_API_KEY=new-key --path=/rad-gateway/providers/openai
   # Gateway detects change via webhook, no restart needed
   ```

2. **Cold Vault Archival**
   - When rotating OpenAI key, old key archived to OpenBao
   - Retention: 5 years (compliance requirement)
   - Access: Audit trail for who/when accessed old keys

3. **Multi-Environment Sync**
   - Development: Local Infisical (dev environment)
   - Staging: Staging Infisical (staging environment)
   - Production: Production Infisical + OpenBao cold vault

---

## Security Considerations

- **mTLS**: Encrypt communication between gateway and vaults
- **Token Scope**: Infisical service tokens should have minimal permissions
- **Audit Everything**: All secret access logged to OpenBao
- **No Secrets in Logs**: Redact keys from application logs
- **Memory Security**: Zero out secret memory after use

---

## Dependencies

- [Golden Stack Deployment](docs/operations/golden-stack-deployment.md) - Must be deployed first
- [Infisical SDK](https://github.com/Infisical/infisical-go) - Go client library
- [OpenBao SDK](https://github.com/openbao/openbao-go) - Go client library

---

## Acceptance Criteria

- [ ] Gateway starts successfully with secrets from Infisical
- [ ] Secrets rotate without gateway restart
- [ ] Old secrets archived to OpenBao with audit trail
- [ ] Fallback to env vars works when vaults unavailable
- [ ] All tests pass with vault integration
- [ ] Documentation complete

---

## Notes

- Consider using [external-secrets](https://external-secrets.io/) for Kubernetes environments
- OpenBao cold vault could also store old model versions for rollback
- Infisical has native Kubernetes operator if we move to K8s later

**Related**:
- [Golden Stack Deployment](docs/operations/golden-stack-deployment.md)
- [Infisical Docs](https://infisical.com/docs)
- [OpenBao Docs](https://openbao.org/docs)
