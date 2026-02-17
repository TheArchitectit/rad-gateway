# RAD Gateway API Design Summary

**Version**: 0.1.0
**Date**: 2026-02-17
**Designer**: api-architect (The Pessimist persona)

---

## Executive Summary

This document presents the OpenAPI 3.0 specifications for RAD Gateway (Brass Relay) Phase 5 administrative APIs. Four new API namespaces have been designed:

1. **Admin API** (`/v0/admin/*`) - System administration and configuration
2. **RBAC API** (`/v0/auth/*`) - Role-based access control and API key management
3. **Billing API** (`/v0/billing/*`) - Cost tracking and usage billing
4. **Quotas API** (`/v0/quotas/*`) - Quota management and rate limiting

---

## Design Philosophy

### Pessimist Perspective: Failure Pattern Prevention

This design actively anticipates failure modes:

1. **Pagination**: Cursor-based prevents offset explosion on large datasets
2. **Soft Deletes**: Keys are revoked, not deleted - maintains audit trail
3. **Grace Periods**: Key rotation allows overlapping validity
4. **Burst Handling**: Quotas support temporary overages
5. **Alert Cooldowns**: Prevents alert storms
6. **Conflict Detection**: Returns 409 with clear error context

### Consistency Patterns

| Pattern | Application | Rationale |
|---------|-------------|-----------|
| Resource nesting | `/v0/auth/keys/{id}/usage` | Clear ownership hierarchy |
| Query parameters | Filtering, pagination | RESTful and cacheable |
| Cursor pagination | All list endpoints | O(1) performance at scale |
| RFC 7807 errors | Consistent error format | Standard tooling support |
| Bearer auth | All v0 endpoints | Future JWT support |

---

## API Specifications

### Admin API (`/v0/admin/*`)

**File**: `openapi-v0-admin.yaml`

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/health` | GET | Public health check |
| `/v0/admin/health/detailed` | GET | Detailed component health |
| `/v0/admin/status` | GET | Gateway operational status |
| `/v0/admin/config` | GET, PUT | Configuration management |
| `/v0/admin/config/reload` | POST | Reload from sources |
| `/v0/admin/model-routes` | GET, POST | Model routing rules |
| `/v0/admin/model-routes/{id}` | GET, DELETE | Route management |
| `/v0/admin/logs` | GET | Audit log queries |
| `/v0/admin/maintenance` | GET, PUT | Maintenance mode |
| `/v0/admin/providers` | GET | Provider status |
| `/v0/admin/providers/{name}/health` | POST | Trigger health check |

**Key Design Decisions**:
- Configuration reload is explicit (POST) not automatic - prevents accidental changes
- Model routes use simple weight-based routing (matches existing implementation)
- Logs endpoint consolidates usage and traces (single query interface)
- Maintenance mode enables graceful degradation

---

### RBAC API (`/v0/auth/*`)

**File**: `openapi-v0-rbac.yaml`

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v0/auth/keys` | GET, POST | API key CRUD |
| `/v0/auth/keys/{id}` | GET, PUT, DELETE | Key management |
| `/v0/auth/keys/{id}/rotate` | POST | Key rotation with grace period |
| `/v0/auth/keys/{id}/usage` | GET | Per-key usage |
| `/v0/auth/roles` | GET, POST | Role definitions |
| `/v0/auth/roles/{id}` | GET, PUT, DELETE | Role management |
| `/v0/auth/permissions` | GET | Permission catalog |
| `/v0/auth/check` | POST | Permission checking |
| `/v0/auth/tags` | GET | Tag management |

**Security Model**:
```
API Key -> Roles -> Permissions
        -> Tags -> Resource Access
```

**Permission Format**: `resource:action`
- `models:read` - List available models
- `billing:write` - Modify billing settings
- `*:*` - Full admin access

**Tag Format**: `category:value`
- `env:production` - Production environment
- `team:platform` - Platform team resources
- `project:customer-a` - Customer-specific access

**Key Design Decisions**:
- Keys are revoked, not deleted (audit trail)
- Rotation includes grace period (configurable, default 24h)
- Roles are immutable once assigned (prevents privilege escalation)
- Force parameter on delete cascades to assignments

---

### Billing API (`/v0/billing/*`)

**File**: `openapi-v0-billing.yaml`

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v0/billing/usage` | GET | Usage aggregation |
| `/v0/billing/costs` | GET | Cost calculations |
| `/v0/billing/costs/realtime` | GET | Current month MTD |
| `/v0/billing/pricing` | GET | Model pricing |
| `/v0/billing/pricing/{modelId}` | GET, PUT | Per-model pricing |
| `/v0/billing/invoices` | GET | Invoice list |
| `/v0/billing/invoices/{id}` | GET | Invoice details |
| `/v0/billing/invoices/{id}/pdf` | GET | PDF download |
| `/v0/billing/reports` | GET, POST | Report generation |
| `/v0/billing/reports/{id}` | GET, DELETE | Report management |
| `/v0/billing/projections` | GET | Cost forecasting |
| `/v0/billing/alerts` | GET, POST | Billing alerts |
| `/v0/billing/alerts/{id}` | DELETE | Remove alert |

**Pricing Model**:
```
Cost = (prompt_tokens * prompt_rate + completion_tokens * completion_rate) / 1000
```

**Alert Types**:
- `monthly_spend` - Monthly budget threshold
- `daily_spend` - Daily budget threshold
- `api_key_spend` - Per-key spending

**Key Design Decisions**:
- Pricing is per-model and mutable (with effective_from)
- Reports are async (202 Accepted + polling)
- Projections use rolling average with confidence intervals
- Realtime endpoint has 5-minute delay (batch processing)

---

### Quotas API (`/v0/quotas/*`)

**File**: `openapi-v0-quotas.yaml`

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v0/quotas` | GET, POST | Quota definitions |
| `/v0/quotas/{id}` | GET, PUT, DELETE | Quota management |
| `/v0/quotas/{id}/assignments` | GET, POST | Assign to entities |
| `/v0/quotas/{id}/assignments/{aid}` | DELETE | Remove assignment |
| `/v0/quotas/{id}/usage` | GET | Current usage |
| `/v0/quotas/{id}/reset` | POST | Manual reset |
| `/v0/quotas/{id}/history` | GET | Usage history |
| `/v0/quotas/check` | POST | Pre-flight check |
| `/v0/quotas/limits` | GET | Rate limits |
| `/v0/quotas/alerts` | GET, POST | Quota alerts |
| `/v0/quotas/alerts/{id}` | DELETE | Remove alert |
| `/v0/quotas/templates` | GET | Predefined quotas |

**Quota Types**:

| Type | Description | Unit |
|------|-------------|------|
| `requests` | Request count | count |
| `tokens` | Token consumption | tokens |
| `cost` | Spending amount | currency |
| `rate` | Request rate | req/time |

**Scopes**:

| Scope | Target | Example |
|-------|--------|---------|
| `global` | All requests | Gateway-wide limit |
| `api_key` | Specific key | Per-client limit |
| `model` | Specific model | Per-model limit |
| `provider` | Provider | Per-provider limit |
| `tag` | Tagged resources | Team budgets |

**Limit Types**:
- **Hard**: Request rejected when exceeded (429 Too Many Requests)
- **Soft**: Alert sent when exceeded (request allowed)

**Key Design Decisions**:
- Pre-flight check endpoint for client-side optimization
- Burst allowance for traffic spikes
- Configurable reset schedules (day/hour)
- Templates for common quota patterns

---

## Security Analysis

### Authentication

All `/v0/*` endpoints require Bearer token authentication:
```
Authorization: Bearer <api-key-jwt>
```

**Exception**: `/health` remains public for load balancer checks.

### Authorization Matrix

| Role | Permissions | Typical Assignment |
|------|-------------|-------------------|
| `admin` | `*:*` | System administrators |
| `billing_manager` | `billing:*`, `quotas:read` | Finance team |
| `api_manager` | `auth:*`, `admin:read` | Platform team |
| `service_account` | `models:read`, `billing:read` | Applications |
| `readonly` | `*:read` | Auditors |

### Security Concerns Identified

1. **Admin API exposure**: `/v0/admin/config` reveals model routes (sensitive)
   - Mitigation: Mask provider credentials in config snapshot

2. **Billing data scope**: Cost data may be visible across tags
   - Mitigation: Filter by API key scope

3. **Quota reset**: Manual reset could be abused
   - Mitigation: Require `quotas:write` permission + audit log

---

## Consistency Review: Existing API

### Current Endpoints

| Endpoint | Version | Status | Recommendation |
|----------|---------|--------|----------------|
| `/health` | - | Public | Keep (add version) |
| `/v1/chat/completions` | OpenAI | Stable | No change |
| `/v1/responses` | OpenAI | Stable | No change |
| `/v1/messages` | Anthropic | Stable | No change |
| `/v1/embeddings` | OpenAI | Stable | No change |
| `/v1/images/generations` | OpenAI | Stable | No change |
| `/v1/audio/transcriptions` | OpenAI | Stable | No change |
| `/v1/models` | OpenAI | Stable | No change |
| `/v1beta/models/` | Gemini | Beta | No change |
| `/v0/management/config` | Internal | Deprecated | Migrate to `/v0/admin/config` |
| `/v0/management/usage` | Internal | Deprecated | Migrate to `/v0/admin/logs` |
| `/v0/management/traces` | Internal | Deprecated | Migrate to `/v0/admin/logs` |

### Deprecation Plan

**Phase 1** (Current): New endpoints available
**Phase 2** (Phase 5 completion): Add deprecation headers to old endpoints
**Phase 3** (Phase 6): Remove old endpoints

### Headers for Deprecated Endpoints

```
Deprecation: true
Sunset: Wed, 30 Apr 2026 00:00:00 GMT
Link: </v0/admin/config>; rel="successor-version"
```

---

## Error Response Standardization

### Current Error Format

```json
{
  "error": {
    "message": "method not allowed"
  }
}
```

**Issues**:
- No error code for programmatic handling
- No request ID for tracing
- No structured details

### Proposed Error Format (RFC 7807)

```json
{
  "error": {
    "code": "validation_error",
    "message": "Invalid request parameters",
    "details": {
      "retryBudget": "must be between 0 and 10"
    },
    "request_id": "req_abc123"
  }
}
```

**Migration**: Update existing handlers to include `code` and `request_id` fields.

---

## Pagination Strategy

### Cursor-Based Pagination

All list endpoints use cursor-based pagination:

```json
{
  "data": [...],
  "pagination": {
    "cursor": "eyJpZCI6MTAwfQ==",
    "has_more": true,
    "total_count": 1000
  }
}
```

**Why cursor-based?**
- O(1) performance regardless of dataset size
- Stable results during concurrent writes
- Works with distributed databases

**Limit parameters**:
- Default: 20 (RBAC), 50 (admin), 100 (billing), 30 (quotas)
- Maximum: 100 (RBAC), 500 (admin), 1000 (billing), 100 (quotas)

---

## Rate Limiting Considerations

### Current State

No rate limiting implemented in current code.

### Recommended Rate Limits

| Endpoint Type | Rate Limit | Burst |
|---------------|------------|-------|
| `/v1/*` (inference) | 100/min/key | 150 |
| `/v0/admin/*` | 60/min/admin | 100 |
| `/v0/auth/*` | 30/min/admin | 50 |
| `/v0/billing/*` | 30/min/key | 50 |
| `/v0/quotas/*` | 60/min/key | 100 |
| `/health` | 1000/min/ip | 1000 |

### Rate Limit Response

```json
{
  "error": {
    "code": "rate_limit_exceeded",
    "message": "Rate limit exceeded. Try again in 45 seconds.",
    "details": {
      "retry_after": 45
    }
  }
}
```

**Headers**:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1708195200
Retry-After: 45
```

---

## Known Edge Cases and Vulnerabilities

### 1. Time Window Boundary Attacks

**Scenario**: Quota resets at midnight UTC. Attacker floods at 23:59.

**Mitigation**: Sliding window quotas (not fixed windows).

### 2. Key Rotation Abuse

**Scenario**: Attacker rotates key repeatedly to generate many valid secrets.

**Mitigation**: Rate limit rotation to 10/hour per key.

### 3. Billing Report DoS

**Scenario**: Large date range report generation blocks resources.

**Mitigation**:
- Max 90 days per request
- Async processing with timeout
- Limit concurrent reports per key

### 4. Tag Enumeration

**Scenario**: Attacker queries all tags to discover resource structure.

**Mitigation**: Tag visibility scoped to user's permissions.

### 5. Cursor Tampering

**Scenario**: Attacker modifies cursor to access unauthorized data.

**Mitigation**: Sign cursors with HMAC (or use encrypted tokens).

---

## Implementation Priority

### Phase 5 (Current)

1. Admin API (migrates existing management endpoints)
2. RBAC API (required for multi-tenancy)
3. Quota API (prevents abuse)

### Phase 6

1. Billing API (depends on cost tracking infrastructure)
2. Async report generation
3. Cost projections

### Phase 7

1. Rate limiting implementation
2. Advanced quota features (burst, sliding window)
3. Billing integrations (Stripe, etc.)

---

## File Locations

| Specification | Path |
|---------------|------|
| Admin API | `/mnt/ollama/git/RADAPI01/docs/api/openapi-v0-admin.yaml` |
| RBAC API | `/mnt/ollama/git/RADAPI01/docs/api/openapi-v0-rbac.yaml` |
| Billing API | `/mnt/ollama/git/RADAPI01/docs/api/openapi-v0-billing.yaml` |
| Quotas API | `/mnt/ollama/git/RADAPI01/docs/api/openapi-v0-quotas.yaml` |
| This summary | `/mnt/ollama/git/RADAPI01/docs/api/api-design-summary.md` |

---

## Next Steps

1. **Team Charlie Review**: Security audit of RBAC design
2. **Team Delta QA**: Review error scenarios and test coverage
3. **Team Bravo Implementation**: Implement Admin and RBAC APIs
4. **Team Echo Observability**: Design metrics for quota enforcement
5. **Team Golf Documentation**: Generate SDK examples

---

## References

- [OpenAPI 3.0.3 Specification](https://spec.openapis.org/oas/v3.0.3)
- [RFC 7807 - Problem Details](https://tools.ietf.org/html/rfc7807)
- [AxonHub Patterns](/mnt/ollama/git/RADAPI01/docs/analysis/axonhub-patterns-report.md)
- [Control Room Tagging Design](/mnt/ollama/git/RADAPI01/docs/plans/2026-02-16-control-room-tagging-design.md)

---

**Review Status**: Awaiting security review from Team Charlie

**Pessimist's Prediction**: The RBAC permission system will have edge cases we haven't thought of. Recommend starting with a small permission set and expanding based on real usage patterns.
