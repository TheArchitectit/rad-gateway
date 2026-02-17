# Feature Parity Analysis Report

**Agent 4: Feature Parity Mapper**
**Date:** 2026-02-17
**Scope:** RAD Gateway, AxonHub, Plexus

---

## Executive Summary

This report compares three AI API Gateway implementations to identify feature gaps and opportunities for RAD Gateway. The analysis reveals significant functional differences across the platforms, with each having unique strengths that could inform RAD Gateway's roadmap.

### Key Findings

- **RAD Gateway**: Lean, focused gateway with excellent streaming support and clean architecture. Lacks enterprise features like RBAC, quotas, and persistent storage.
- **AxonHub**: Full-featured enterprise platform with comprehensive management UI, RBAC, and sophisticated load balancing. More complex deployment.
- **Plexus**: Feature-rich TypeScript implementation with unique capabilities like OAuth provider integration, responses storage, and quota tracking.

---

## 1. Feature Comparison Matrix

### Core Gateway Capabilities

| Feature | RAD Gateway | AxonHub | Plexus | Notes |
|---------|-------------|---------|--------|-------|
| **Multi-Provider Support** | OpenAI, Anthropic, Gemini | OpenAI, Anthropic, Gemini + more | OpenAI, Anthropic, Gemini + OAuth | All support major providers |
| **OpenAI API Compatibility** | Chat, Responses, Embeddings, Images, Audio | Chat, Responses (partial), Embeddings, Rerank | Chat, Responses (full), Embeddings, Images, Audio | Plexus has most complete Responses API |
| **Anthropic API Compatibility** | Messages API | Full Messages API | Messages API | AxonHub has best Anthropic support |
| **Streaming (SSE)** | Full support | Full support | Full support | All platforms support streaming |
| **Load Balancing** | Round-robin, Weighted, Priority | Error-aware, Weighted, Session-aware | Random, Cost-based, Usage-based | RAD Gateway has clean implementations |
| **Health Checking** | Basic health checks | Comprehensive channel probes | Provider cooldowns | AxonHub has most sophisticated health checks |
| **Circuit Breaker** | Implemented | Not identified | Provider cooldowns | RAD Gateway has explicit circuit breaker |

### Authentication & Authorization

| Feature | RAD Gateway | AxonHub | Plexus | Notes |
|---------|-------------|---------|--------|-------|
| **API Key Management** | Simple env-based keys | Full RBAC with JWT | Admin key + per-key secrets | AxonHub has enterprise-grade RBAC |
| **Role-Based Access Control** | Not implemented | Full RBAC with roles/permissions | Basic key-based access | Significant gap for RAD Gateway |
| **Project Isolation** | Not implemented | Full project-based isolation | Not identified | Critical for multi-tenant deployments |
| **OAuth Provider Support** | Not implemented | Not identified | Anthropic, GitHub Copilot, Codex, etc. | Plexus unique feature |
| **API Key Profiles** | Not implemented | Model mappings, channel restrictions | Not identified | AxonHub feature |

### Observability & Tracing

| Feature | RAD Gateway | AxonHub | Plexus | Notes |
|---------|-------------|---------|--------|-------|
| **Request Tracing** | Basic in-memory trace store | Full thread-aware tracing with DB | Debug logging with snapshots | AxonHub has most complete tracing |
| **Usage Tracking** | In-memory, basic fields | Comprehensive with cost calculation | Full usage storage with attribution | Plexus has detailed usage records |
| **Metrics** | Basic health endpoint | Prometheus/OTLP support | Prometheus metrics endpoint | RAD Gateway needs metrics expansion |
| **Debug Logging** | Structured logging | Request/response dumping | Full debug manager with snapshots | Plexus has sophisticated debug system |
| **Trace IDs** | Supported | AH-Trace-Id header support | Request ID tracking | All support request tracking |
| **Thread IDs** | Not implemented | AH-Thread-Id header support | Not identified | Important for conversation tracking |

### Data Persistence

| Feature | RAD Gateway | AxonHub | Plexus | Notes |
|---------|-------------|---------|--------|-------|
| **Database Support** | None (in-memory only) | SQLite, PostgreSQL, MySQL, TiDB | SQLite, PostgreSQL | RAD Gateway significant gap |
| **Response Storage** | Not implemented | Not identified | Full Responses API storage with TTL | Plexus unique feature |
| **Conversation History** | Not implemented | Thread-aware tracing | Conversation storage | Both competitors have this |
| **Schema Management** | N/A | Ent ORM with auto-migration | Drizzle ORM with migrations | RAD Gateway needs persistence layer |

### Advanced Features

| Feature | RAD Gateway | AxonHub | Plexus | Notes |
|---------|-------------|---------|--------|-------|
| **Quota Management** | Not implemented | Basic quota enforcement | Comprehensive quota tracking + checkers | Plexus leads in quota management |
| **Rate Limiting** | Not implemented | Per-channel rate limits | Not identified | Needed for production |
| **Cost Tracking** | Not implemented | Real-time cost calculation | Usage-based pricing | Business-critical feature |
| **Caching** | Not implemented | Memory + Redis two-level | Not identified | Performance optimization |
| **Model Mapping** | Basic route table | Flexible model associations | Provider-to-model mapping | AxonHub has sophisticated mapping |
| **Admin UI** | Basic HTTP endpoints | Full React-based management UI | Limited management endpoints | RAD Gateway needs UI |
| **Multi-turn Conversation** | Not implemented | Partial | Full previous_response_id support | Plexus has best implementation |
| **Tool Use Support** | Not implemented | Full tool support | Not identified | Important for agent workflows |
| **Parameter Override** | Not implemented | Channel override templates | Not identified | AxonHub enterprise feature |
| **Pricing Management** | Not implemented | Channel-model price versions | Simple pricing config | AxonHub has sophisticated pricing |

### Deployment & Operations

| Feature | RAD Gateway | AxonHub | Plexus | Notes |
|---------|-------------|---------|--------|-------|
| **Container Support** | Docker/Podman | Docker Compose, Render | Docker | All container-ready |
| **Secrets Management** | Infisical integration | Environment variables | Auth.json file | RAD Gateway's Infisical integration is unique |
| **Systemd Service** | Supported | Supported | Not identified | Production deployment |
| **Configuration** | Environment variables | YAML + Environment | YAML | RAD Gateway simpler, others more flexible |
| **Health Checks** | /health endpoint | Comprehensive probes | /health endpoint | All have health checks |
| **Graceful Shutdown** | Likely supported | Supported | Supported | Production requirement |

---

## 2. Missing Features by Priority

### High Priority (Business Critical)

| Feature | Current Status | Impact | Effort | Source Reference |
|---------|---------------|--------|--------|-----------------|
| **Persistent Database** | In-memory only | Cannot survive restarts, limited scale | Medium | AxonHub (Ent), Plexus (Drizzle) |
| **Cost Tracking** | Not implemented | Cannot bill customers or track spend | Medium | AxonHub, Plexus |
| **RBAC / Multi-tenancy** | Not implemented | Cannot support multiple teams safely | High | AxonHub |
| **Quota Management** | Not implemented | Risk of runaway costs, no limits | Medium | Plexus |
| **Rate Limiting** | Not implemented | No protection against abuse | Medium | AxonHub |

### Medium Priority (Production Readiness)

| Feature | Current Status | Impact | Effort | Source Reference |
|---------|---------------|--------|--------|-----------------|
| **Admin Web UI** | HTTP endpoints only | Difficult to manage and monitor | High | AxonHub React UI |
| **Response Storage** | Not implemented | No multi-turn conversation support | Medium | Plexus |
| **Redis Caching** | Not implemented | Higher latency, more API calls | Low | AxonHub |
| **Prometheus Metrics** | Basic health only | Poor observability | Low | AxonHub, Plexus |
| **Advanced Load Balancing** | Basic strategies | Suboptimal provider selection | Medium | AxonHub error-aware |
| **Model Aliasing** | Simple route table | Less flexible routing | Low | Both |

### Low Priority (Nice to Have)

| Feature | Current Status | Impact | Effort | Source Reference |
|---------|---------------|--------|--------|-----------------|
| **OAuth Provider Integration** | Not implemented | Requires manual API key management | Medium | Plexus |
| **Tool Use Support** | Not implemented | Limited agent compatibility | Medium | AxonHub |
| **Channel Override Templates** | Not implemented | Less flexible request modification | Medium | AxonHub |
| **Pricing Versioning** | Not implemented | Cannot track price changes | Low | AxonHub |
| **Connection Tracking** | Not implemented | Limited load balancing insight | Low | AxonHub |

---

## 3. Quick Wins (Easy to Implement, High Value)

### 3.1 Add Prometheus Metrics Endpoint
**Value:** Essential for monitoring and alerting
**Effort:** Low (1-2 days)
**Implementation:** Add `/metrics` endpoint using Prometheus Go client

```go
// Add to internal/metrics/metrics.go
import "github.com/prometheus/client_golang/prometheus"

var (
    requestsTotal = prometheus.NewCounterVec(...)
    requestDuration = prometheus.NewHistogramVec(...)
)
```

**Reference:** AxonHub `/internal/metrics/`

### 3.2 Implement Redis Caching
**Value:** Reduce latency and API costs
**Effort:** Low (2-3 days)
**Implementation:** Cache model lists and health status

**Reference:** AxonHub `config.cache` section

### 3.3 Add Request ID Middleware
**Value:** Essential for debugging and tracing
**Effort:** Low (1 day)
**Implementation:** Generate and propagate request IDs

### 3.4 Expand Model Aliasing
**Value:** More flexible routing
**Effort:** Low (2 days)
**Implementation:** Support regex patterns in route table

**Reference:** AxonHub model association patterns

### 3.5 Add Configuration Hot-Reload
**Value:** No restart required for config changes
**Effort:** Low-Medium (3 days)
**Implementation:** Watch config file for changes

---

## 4. Complex Features (High Value, High Effort)

### 4.1 Database Persistence Layer
**Value:** Production-grade reliability and scale
**Effort:** High (2-3 weeks)
**Implementation Options:**
- Option A: Use Ent (AxonHub approach) - more features, more complex
- Option B: Use GORM - simpler, widely used
- Option C: Use sqlc - type-safe, performant

**Tables needed:**
- `requests` - Request log
- `traces` - Trace storage
- `usage` - Usage records
- `api_keys` - Key management
- `channels` - Provider configuration

**Reference:** AxonHub `/internal/ent/schema/`, Plexus `/drizzle/schema/`

### 4.2 RBAC and Multi-tenancy
**Value:** Enterprise adoption
**Effort:** High (3-4 weeks)
**Components:**
- User management
- Role definitions
- Permission system
- Project isolation
- API key scoping

**Reference:** AxonHub `/internal/ent/schema/user.go`, `/internal/scopes/`

### 4.3 Admin Web UI
**Value:** Essential for non-technical users
**Effort:** High (4-6 weeks)
**Options:**
- Embed React app (AxonHub approach)
- Use HTMX for simplicity
- Generate from OpenAPI spec

**Reference:** AxonHub `/frontend/`

### 4.4 Quota Management System
**Value:** Cost control and customer tiers
**Effort:** Medium-High (2-3 weeks)
**Components:**
- Quota checker framework
- Per-key limits
- Per-project limits
- Usage aggregation
- Alerting

**Reference:** Plexus `/src/services/quota/`

### 4.5 Response Storage for Multi-turn
**Value:** Enable conversational interfaces
**Effort:** Medium (1-2 weeks)
**Implementation:**
- Store responses with TTL
- Lookup by `previous_response_id`
- Automatic cleanup

**Reference:** Plexus `/src/services/responses-storage.ts`

---

## 5. Anti-patterns to Avoid

Based on analysis of all three platforms, here are patterns to avoid:

### 5.1 Over-Engineering Authentication
**AxonHub Issue:** Complex RBAC may be overkill for smaller deployments
**Recommendation:** Start simple, add complexity only when needed

### 5.2 Tight Coupling to Frontend
**AxonHub Issue:** Frontend tightly coupled to backend GraphQL schema
**Recommendation:** Maintain clean API boundaries, use OpenAPI

### 5.3 Complex Configuration
**AxonHub Issue:** YAML configuration is extensive and complex
**Recommendation:** Sensible defaults, validate early, fail fast

### 5.4 Memory Leaks in Streaming
**Plexus Issue:** Debug logs can accumulate without cleanup
**Recommendation:** Always implement TTL and cleanup jobs

### 5.5 Synchronous Database Operations
**Issue:** Blocking DB calls hurt streaming performance
**Recommendation:** Use connection pooling, async writes for non-critical data

### 5.6 Hardcoded Provider Logic
**Issue:** Provider-specific code scattered throughout
**Recommendation:** Use adapter pattern consistently (RAD Gateway does this well)

### 5.7 Ignoring Provider Rate Limits
**Issue:** Many gateways don't track provider quotas
**Recommendation:** Implement quota checkers (learn from Plexus)

### 5.8 Monolithic Schema
**AxonHub Issue:** Large database schema with many tables
**Recommendation:** Start with core tables, expand incrementally

---

## 6. Recommended Implementation Order

### Phase 1: Foundation (Weeks 1-2)
1. Add Prometheus metrics endpoint
2. Implement request ID propagation
3. Add Redis caching layer
4. Expand configuration options

### Phase 2: Persistence (Weeks 3-5)
1. Design database schema
2. Implement request logging
3. Add trace storage
4. Create usage tracking

### Phase 3: Management (Weeks 6-8)
1. Build basic admin API
2. Add simple web UI
3. Implement API key management
4. Add basic rate limiting

### Phase 4: Enterprise (Weeks 9-12)
1. Implement RBAC
2. Add project isolation
3. Build quota management
4. Add cost tracking

### Phase 5: Advanced (Ongoing)
1. Response storage for conversations
2. Tool use support
3. Advanced load balancing
4. OAuth provider integration

---

## 7. Unique Strengths to Preserve

### RAD Gateway Strengths
- **Clean Architecture:** Provider adapter pattern is well-designed
- **Simplicity:** Easy to understand and deploy
- **Streaming:** Excellent SSE implementation
- **Go Ecosystem:** Native performance and type safety
- **Infisical Integration:** Good secrets management

### AxonHub Strengths to Consider
- **Enterprise Features:** Comprehensive RBAC and project management
- **Channel Management:** Flexible provider configuration
- **Tracing:** Thread-aware request tracking
- **Frontend:** Full management UI

### Plexus Strengths to Consider
- **OAuth Integration:** Unique provider authentication
- **Quota System:** Comprehensive rate limit tracking
- **Response Storage:** Multi-turn conversation support
- **Debug System:** Detailed request/response logging

---

## 8. Architecture Recommendations

### 8.1 Database Choice
**Recommendation:** PostgreSQL with GORM or sqlc
- **Why:** Balance of features, simplicity, and performance
- **Migration:** Use golang-migrate for version control
- **Models:** Start with 4-5 core tables

### 8.2 Caching Strategy
**Recommendation:** Redis for distributed caching
- Cache health checks (30s TTL)
- Cache model lists (5m TTL)
- Cache rate limit status (1m TTL)

### 8.3 API Design
**Recommendation:** Continue OpenAPI-first approach
- Generate documentation from code
- Maintain backward compatibility
- Version APIs explicitly

### 8.4 Frontend Strategy
**Recommendation:** Start with simple HTMX-based UI
- Lower complexity than React
- Good enough for management tasks
- Can migrate to React later if needed

---

## 9. Conclusion

RAD Gateway has a solid foundation with clean architecture and good streaming support. The main gaps are around enterprise features (RBAC, persistence, quotas) that competitors have invested heavily in.

### Priority Actions:
1. **Immediate:** Add Prometheus metrics and request IDs
2. **Short-term:** Implement database persistence layer
3. **Medium-term:** Build admin UI and API key management
4. **Long-term:** Add RBAC and quota management

### Competitive Position:
- **Against AxonHub:** RAD Gateway is simpler to deploy but lacks enterprise features
- **Against Plexus:** RAD Gateway has better Go performance but lacks unique features like OAuth

The recommended approach is to preserve RAD Gateway's simplicity while incrementally adding essential production features, avoiding the complexity that makes AxonHub harder to deploy for smaller teams.

---

## Appendix A: File References

### RAD Gateway
- `/internal/provider/provider.go` - Adapter interface
- `/internal/routing/router.go` - Basic routing
- `/internal/provider/loadbalancer.go` - Load balancing
- `/internal/usage/usage.go` - Usage tracking (in-memory)
- `/internal/trace/trace.go` - Tracing (in-memory)

### AxonHub
- `/internal/ent/schema/` - Database schema
- `/internal/server/orchestrator/` - Load balancing and selection
- `/internal/scopes/` - RBAC implementation
- `/frontend/` - React management UI
- `/config.example.yml` - Configuration options

### Plexus
- `/packages/backend/src/services/quota/` - Quota management
- `/packages/backend/src/services/debug-manager.ts` - Debug logging
- `/packages/backend/src/routes/inference/responses.ts` - Response storage
- `/packages/backend/drizzle/schema/` - Database schema
- `/plexus.yaml` - Configuration example

---

*Report generated by Agent 4: Feature Parity Mapper*
*For questions or clarifications, see the individual project documentation*
