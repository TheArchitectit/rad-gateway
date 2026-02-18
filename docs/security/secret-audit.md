# RAD Gateway Secret Audit Report

**Date:** 2026-02-18
**Auditor:** Security Engineer (Team Charlie)
**Scope:** All hardcoded secrets, API keys, passwords, and tokens
**Status:** Phase 6 Security Hardening

---

## Executive Summary

This audit identifies **14 hardcoded secrets** and **6 secret categories** requiring Infisical integration. The majority are development-only values, but several critical secrets require immediate remediation before production deployment.

**Risk Summary:**
- **Critical:** 2 findings (production-impacting)
- **High:** 4 findings
- **Medium:** 5 findings
- **Low:** 3 findings

**Immediate Action Required:**
1. Remove hardcoded Infisical service token from `.env`
2. Implement JWT secret retrieval from Infisical
3. Move database credentials to Infisical
4. Configure Redis password via Infisical

---

## Secret Inventory

### 1. JWT Secrets

| Location | File | Secret Type | Risk Level | Status |
|----------|------|-------------|------------|--------|
| `JWT_ACCESS_SECRET` | Environment | Runtime-generated | **CRITICAL** | Not persisted |
| `JWT_REFRESH_SECRET` | Environment | Runtime-generated | **CRITICAL** | Not persisted |

**Findings:**

1. **Runtime Generation (CRITICAL)** - `/mnt/ollama/git/RADAPI01/internal/auth/jwt.go:44-53`
   ```go
   func DefaultConfig() JWTConfig {
       accessSecret := os.Getenv("JWT_ACCESS_SECRET")
       if accessSecret == "" {
           accessSecret = generateSecret()  // GENERATED AT RUNTIME!
       }
   }
   ```
   - Tokens invalidated on every restart
   - Prevents horizontal scaling
   - Violates stateless container principles

2. **Production Config (HIGH)** - `/mnt/ollama/git/RADAPI01/internal/auth/auth_production.go:351-352`
   ```go
   func ProductionJWTConfig() JWTConfig {
       return JWTConfig{
           AccessTokenSecret:  []byte(getenv("JWT_ACCESS_SECRET", generateSecret())),
           RefreshTokenSecret: []byte(getenv("JWT_REFRESH_SECRET", generateSecret())),
       }
   }
   ```
   - Same runtime generation issue in production path

**Remediation:**
- Store `jwt_access_secret` and `jwt_refresh_secret` in Infisical
- Require explicit secret configuration (no fallbacks)
- Implement minimum 32-character requirement

---

### 2. Database Credentials

| Location | File | Secret Type | Risk Level | Status |
|----------|------|-------------|------------|--------|
| `RAD_DB_DSN` | `config/env.example:10` | Hardcoded password | **HIGH** | Example file |
| `RAD_DB_DSN` | `config/env.local:8` | Hardcoded password | **HIGH** | Local dev |
| `POSTGRES_PASSWORD` | `deploy/golden-stack/.env:19` | Hardcoded password | **MEDIUM** | Dev environment |
| `DATABASE_URL` | `cmd/migrate/main.go` | Connection string | **MEDIUM** | Env-only |

**Hardcoded Credentials Found:**

1. **env.example** - `/mnt/ollama/git/RADAPI01/config/env.example:10`
   ```
   RAD_DB_DSN=postgresql://radgateway_user:DATABASE_PASSWORD_REDACTED@localhost:5432/radgateway?sslmode=disable
   ```
   - Password: `DATABASE_PASSWORD_REDACTED`
   - Risk: Developers may copy this to production

2. **env.local** - `/mnt/ollama/git/RADAPI01/config/env.local:8`
   ```
   RAD_DB_DSN=postgresql://radgateway_user:DATABASE_PASSWORD_REDACTED@localhost:5432/radgateway?sslmode=disable
   ```
   - Same password as env.example
   - Risk: Committed to git (even if in .gitignore, may leak)

3. **golden-stack .env** - `/mnt/ollama/git/RADAPI01/deploy/golden-stack/.env:19`
   ```
   POSTGRES_PASSWORD=dev-secretstack-password
   ```
   - Weak development password
   - Acceptable for local dev only

**Remediation:**
- Move database credentials to Infisical path: `/radgateway/{env}/database/dsn`
- Use separate credentials per environment
- Implement SSL/TLS for all database connections
- Rotate passwords every 90 days

---

### 3. Infisical Service Tokens (CRITICAL)

| Location | File | Secret Type | Risk Level | Status |
|----------|------|-------------|------------|--------|
| `INFISICAL_SERVICE_TOKEN` | `.env:11` | Hardcoded token | **CRITICAL** | Active token committed |
| `INFISICAL_TOKEN` | `.env:12` | Hardcoded token | **CRITICAL** | Duplicate of above |

**Hardcoded Token Found:**

1. **Root .env file** - `/mnt/ollama/git/RADAPI01/.env:11-12`
   ```
   INFISICAL_SERVICE_TOKEN=INFISICAL_SERVICE_TOKEN_REDACTED
   INFISICAL_TOKEN=INFISICAL_SERVICE_TOKEN_REDACTED
   ```
   - **IMMEDIATE RISK:** Active service token committed to git
   - Format: Infisical service token v3
   - Project: radgateway-p-zwm
   - Token provides full read access to secrets

**Remediation (IMMEDIATE):**
1. Revoke this token in Infisical immediately
2. Generate new token
3. Store in `/opt/radgateway01/config/infisical-token` (not in .env)
4. Set file permissions: `chmod 600`
5. Add `.env` to `.gitignore` if not already present
6. Rotate any secrets that may have been exposed

---

### 4. API Keys

| Location | File | Secret Type | Risk Level | Status |
|----------|------|-------------|------------|--------|
| `RAD_API_KEYS` | `.env:3` | Default key | **HIGH** | Committed |
| `RAD_API_KEYS` | `config/env.example:24` | Test key | **MEDIUM** | Example file |
| `RAD_API_KEYS` | `config/env.local:15` | Dev keys | **LOW** | Local dev |
| `RAD_API_KEYS` | `deploy/golden-stack/.env:94` | Dev key | **LOW** | Dev environment |

**Hardcoded API Keys Found:**

1. **Root .env** - `/mnt/ollama/git/RADAPI01/.env:3`
   ```
   RAD_API_KEYS=default:replace-with-real-key
   ```
   - Weak default key
   - May be used in production if not changed

2. **env.example** - `/mnt/ollama/git/RADAPI01/config/env.example:24`
   ```
   RAD_API_KEYS=test:rad_test_key_12345
   ```
   - Test key that may be copied to production

3. **env.local** - `/mnt/ollama/git/RADAPI01/config/env.local:15`
   ```
   RAD_API_KEYS=test:rad_test_key_12345,dev:dev_key_67890
   ```
   - Multiple weak keys for local development

**Remediation:**
- Store API keys in Infisical: `/radgateway/{env}/api_keys`
- Use format: `name:secret,name2:secret2`
- Generate cryptographically random keys (min 32 chars)
- Implement key rotation without restart

---

### 5. Provider API Keys

| Location | File | Secret Type | Risk Level | Status |
|----------|------|-------------|------------|--------|
| `OPENAI_API_KEY` | `.env` | Environment | **MEDIUM** | Env-only (empty) |
| `ANTHROPIC_API_KEY` | `.env` | Environment | **MEDIUM** | Env-only (empty) |
| `GEMINI_API_KEY` | `.env` | Environment | **MEDIUM** | Env-only (empty) |

**Current Status:**
- Keys are defined but empty in `.env`
- Populated via environment variables at runtime
- Acceptable pattern, but should come from Infisical

**Remediation:**
- Store in Infisical:
  - `/radgateway/{env}/providers/openai/api_key`
  - `/radgateway/{env}/providers/anthropic/api_key`
  - `/radgateway/{env}/providers/gemini/api_key`
- Implement provider key rotation

---

### 6. Redis Password

| Location | File | Secret Type | Risk Level | Status |
|----------|------|-------------|------------|--------|
| `RAD_REDIS_PASSWORD` | `cmd/rad-gateway/main.go:105` | Environment | **MEDIUM** | Env-only |
| `REDIS_PASSWORD` | `internal/cache/go_redis.go:41` | Environment | **MEDIUM** | Env-only |

**Current Status:**
- Password read from environment only
- No hardcoded password found
- Falls back to empty string (no auth)

**Remediation:**
- Store in Infisical: `/radgateway/{env}/redis/password`
- Require password in production (no empty fallback)
- Use Redis AUTH command

---

### 7. Infisical Internal Secrets (Golden Stack)

| Location | File | Secret Type | Risk Level | Status |
|----------|------|-------------|------------|--------|
| `INFISICAL_ENCRYPTION_KEY` | `deploy/golden-stack/.env:33` | Hardcoded | **HIGH** | Dev only |
| `INFISICAL_JWT_SECRET` | `deploy/golden-stack/.env:47` | Hardcoded | **HIGH** | Dev only |
| `POSTGRES_PASSWORD` | `deploy/golden-stack/.env:19` | Hardcoded | **MEDIUM** | Dev only |

**Hardcoded Secrets Found:**

1. **INFISICAL_ENCRYPTION_KEY** - `/mnt/ollama/git/RADAPI01/deploy/golden-stack/.env:33`
   ```
   INFISICAL_ENCRYPTION_KEY=dev-encryption-key-do-not-use-in-production-32bytes
   ```
   - Clearly marked for development only
   - Must be generated with `openssl rand -base64 32` for production

2. **INFISICAL_JWT_SECRET** - `/mnt/ollama/git/RADAPI01/deploy/golden-stack/.env:47`
   ```
   INFISICAL_JWT_SECRET=dev-jwt-secret-min-32-characters-long
   ```
   - Weak development secret
   - Must be changed for production

**Remediation:**
- These are Infisical's own secrets, not RAD Gateway's
- Use `deploy/golden-stack/env-config.sh` to generate secure values
- Store generated values in Infisical for Infisical's own use

---

## Infisical Integration Design

### Proposed Path Structure

```
/radgateway/
├── {environment}/                  # dev, staging, production
│   ├── jwt/
│   │   ├── access_secret          # JWT_ACCESS_SECRET
│   │   └── refresh_secret         # JWT_REFRESH_SECRET
│   ├── database/
│   │   ├── dsn                    # Full connection string
│   │   ├── host                   # Individual components (optional)
│   │   ├── port
│   │   ├── username
│   │   ├── password
│   │   └── database
│   ├── api_keys/
│   │   └── keys                   # Format: name:secret,name2:secret2
│   ├── redis/
│   │   ├── password               # RAD_REDIS_PASSWORD
│   │   └── address                # RAD_REDIS_ADDR (optional)
│   └── providers/
│       ├── openai/
│       │   └── api_key            # OPENAI_API_KEY
│       ├── anthropic/
│       │   └── api_key            # ANTHROPIC_API_KEY
│       └── gemini/
│           └── api_key            # GEMINI_API_KEY
```

### Environment Mapping

| Environment | Infisical Path | Description |
|-------------|----------------|-------------|
| Development | `/radgateway/dev/` | Local development secrets |
| Staging | `/radgateway/staging/` | Pre-production testing |
| Production | `/radgateway/production/` | Live production secrets |

### Access Control Matrix

| Secret Path | Gateway Service | Admin Users | CI/CD | Notes |
|-------------|-----------------|-------------|-------|-------|
| `/radgateway/*/jwt/*` | Read | Read/Write | None | Only gateway needs read |
| `/radgateway/*/database/*` | Read | Read/Write | Read | CI/CD for migrations |
| `/radgateway/*/api_keys/*` | Read | Read/Write | None | Admin rotation only |
| `/radgateway/*/providers/*` | Read | Read/Write | None | Admin rotation only |
| `/radgateway/*/redis/*` | Read | Read/Write | None | Only gateway needs read |

### Implementation Phases

#### Phase 1: JWT Secrets (Critical)

1. Create secrets in Infisical:
   ```bash
   infisical secrets set jwt_access_secret=$(openssl rand -base64 32) --path=/radgateway/production/jwt
   infisical secrets set jwt_refresh_secret=$(openssl rand -base64 32) --path=/radgateway/production/jwt
   ```

2. Update `/mnt/ollama/git/RADAPI01/internal/auth/jwt.go`:
   - Remove `generateSecret()` fallback
   - Fail startup if secrets not available
   - Add Infisical client integration

3. Modify `LoadConfig()` to fetch from Infisical first, env fallback:
   ```go
   func LoadConfigWithInfisical(client *secrets.Client) (JWTConfig, error) {
       accessSecret, err := client.GetSecret(ctx, "jwt_access_secret")
       if err != nil {
           // Fallback to env var
           accessSecret = os.Getenv("JWT_ACCESS_SECRET")
       }
       // ...
   }
   ```

#### Phase 2: Database Credentials (High)

1. Create secrets in Infisical:
   ```bash
   infisical secrets set dsn="postgresql://radgateway:${DB_PASSWORD}@localhost:5432/radgateway?sslmode=require" --path=/radgateway/production/database
   ```

2. Update database initialization to support Infisical:
   ```go
   // internal/db/postgres.go
   func NewFromInfisical(client *secrets.Client) (*PostgresDB, error) {
       dsn, err := client.GetSecret(ctx, "database_dsn")
       // ...
   }
   ```

#### Phase 3: API Keys (High)

1. Migrate existing API keys to Infisical
2. Update `internal/config/config.go` `loadKeys()` function
3. Implement automatic key rotation via Infisical webhooks

#### Phase 4: Provider Keys (Medium)

1. Store provider API keys in Infisical
2. Implement `GetProviderKey()` in secrets client
3. Add caching with TTL to reduce Infisical calls

#### Phase 5: Redis Password (Medium)

1. Store Redis password in Infisical
2. Update Redis connection logic
3. Require password in production (fail if empty)

### Code Changes Required

#### 1. Enhanced Secrets Client

Update `/mnt/ollama/git/RADAPI01/internal/secrets/infisical.go`:

```go
// GetJWTSecrets retrieves JWT configuration from Infisical
func (c *Client) GetJWTSecrets(ctx context.Context) (accessSecret, refreshSecret string, err error) {
    access, err := c.GetSecret(ctx, "jwt_access_secret")
    if err != nil {
        return "", "", fmt.Errorf("jwt_access_secret: %w", err)
    }

    refresh, err := c.GetSecret(ctx, "jwt_refresh_secret")
    if err != nil {
        return "", "", fmt.Errorf("jwt_refresh_secret: %w", err)
    }

    return access, refresh, nil
}

// GetDatabaseDSN retrieves database connection string
func (c *Client) GetDatabaseDSN(ctx context.Context) (string, error) {
    return c.GetSecret(ctx, "database_dsn")
}

// GetAPIKeys retrieves all API keys
func (c *Client) GetAPIKeys(ctx context.Context) (map[string]string, error) {
    keysStr, err := c.GetSecret(ctx, "api_keys")
    if err != nil {
        return nil, err
    }
    return parseKeys(keysStr), nil
}
```

#### 2. Configuration Loading

Modify `/mnt/ollama/git/RADAPI01/internal/config/config.go`:

```go
func Load() Config {
    log := logger.WithComponent("config")

    // Initialize Infisical client
    infisicalCfg := secrets.LoadConfig()
    var secretClient *secrets.Client

    if infisicalCfg.Token != "" {
        var err error
        secretClient, err = secrets.NewClient(infisicalCfg)
        if err != nil {
            log.Warn("Failed to initialize Infisical client", "error", err)
        }
    }

    // Load JWT secrets from Infisical or env
    jwtConfig := loadJWTConfig(secretClient)

    // Load database config from Infisical or env
    dbConfig := loadDBConfig(secretClient)

    // Load API keys from Infisical or env
    apiKeys := loadKeys(secretClient)

    // ...
}
```

#### 3. Secret Rotation Support

Add to `/mnt/ollama/git/RADAPI01/internal/secrets/manager.go`:

```go
// SecretManager handles dynamic secret updates
type SecretManager struct {
    client    *Client
    cache     map[string]*cachedSecret
    mu        sync.RWMutex
}

type cachedSecret struct {
    value     string
    expiresAt time.Time
}

// Get retrieves secret with caching
func (sm *SecretManager) Get(ctx context.Context, key string) (string, error) {
    sm.mu.RLock()
    cached, exists := sm.cache[key]
    sm.mu.RUnlock()

    if exists && time.Now().Before(cached.expiresAt) {
        return cached.value, nil
    }

    // Fetch from Infisical
    value, err := sm.client.GetSecret(ctx, key)
    if err != nil {
        return "", err
    }

    // Cache for 5 minutes
    sm.mu.Lock()
    sm.cache[key] = &cachedSecret{
        value:     value,
        expiresAt: time.Now().Add(5 * time.Minute),
    }
    sm.mu.Unlock()

    return value, nil
}
```

### Deployment Configuration

#### Production Environment File

`/opt/radgateway01/config/env`:

```bash
# RAD Gateway Production Configuration
# Secrets are loaded from Infisical - minimal bootstrap config here

# Server Configuration
RAD_LISTEN_ADDR=:8090
RAD_ENVIRONMENT=production

# Infisical Configuration (bootstrap only)
INFISICAL_API_URL=http://172.16.30.45:8080/api
INFISICAL_ENVIRONMENT=production
# Token is read from /opt/radgateway01/config/infisical-token

# Retry Configuration
RAD_RETRY_BUDGET=2

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

#### Infisical Token File

`/opt/radgateway01/config/infisical-token`:

```
st.<project-id>.<token>.<signature>
```

Permissions: `chmod 600 /opt/radgateway01/config/infisical-token`

### Security Considerations

1. **Token Scope:**
   - Create dedicated service token for RAD Gateway
   - Limit to `/radgateway/production/*` path only
   - Read-only permissions (no write access)

2. **Secret Rotation:**
   - Implement rotation without restart
   - Use caching with TTL to pick up new values
   - Log rotation events for audit trail

3. **Fail-Safe:**
   - If Infisical unavailable, use cached values
   - If cache expired, fail secure (don't start)
   - Never fall back to hardcoded defaults in production

4. **Audit Logging:**
   - Log all secret access (not values, just keys)
   - Log rotation events
   - Forward to SIEM for monitoring

### Migration Plan

| Step | Action | Owner | Timeline |
|------|--------|-------|----------|
| 1 | Revoke exposed Infisical token | Security | Immediate |
| 2 | Create production secrets in Infisical | DevOps | Day 1 |
| 3 | Update gateway to load JWT from Infisical | Engineering | Day 2-3 |
| 4 | Migrate database credentials | DevOps | Day 4 |
| 5 | Migrate API keys | Engineering | Day 5 |
| 6 | Migrate provider keys | DevOps | Day 6 |
| 7 | Migrate Redis password | DevOps | Day 7 |
| 8 | Remove all hardcoded secrets from repo | Security | Day 8 |
| 9 | Audit and verify no secrets in git history | Security | Day 9 |
| 10 | Update documentation and runbooks | Documentation | Day 10 |

---

## Compliance Mapping

| Requirement | Finding | Remediation |
|-------------|---------|-------------|
| SOC 2 CC6.1 | CRITICAL-001: JWT runtime generation | Store in Infisical |
| SOC 2 CC6.2 | CRITICAL-003: Exposed Infisical token | Revoke and rotate |
| SOC 2 CC6.7 | HIGH-001: Hardcoded DB passwords | Move to Infisical |
| ISO 27001 A.9.4.3 | HIGH-002: Weak API keys | Generate strong keys |
| NIST 800-53 IA-5 | MEDIUM: Secret management | Implement Infisical |
| PCI-DSS 3.6 | All secrets | Encrypt at rest |

---

## Appendix A: Secret Scanning Commands

```bash
# Search for potential secrets in codebase
grep -r -E "(password|secret|token|key)\s*=\s*[\"'][^\"']{8,}[\"']" --include="*.go" --include="*.env*" .

# Search for JWT secret patterns
grep -r -E "JWT_.*_SECRET" --include="*.go" --include="*.env*" .

# Search for API key patterns
grep -r -E "(api_key|api-key|apikey)\s*=\s*[\"'][^\"']+[\"']" --include="*.go" --include="*.env*" .

# Search for database connection strings
grep -r -E "(postgres|mysql|mongodb)://[^\"']+:[^@]+@" --include="*.go" --include="*.env*" .

# Search for Infisical tokens
grep -r "st\.[a-f0-9-]+\.[a-f0-9]+\.[a-f0-9]+" --include="*.go" --include="*.env*" .
```

---

## Appendix B: Infisical CLI Commands

```bash
# Login to Infisical
infisical login

# Create project structure
infisical folders create --path=/radgateway/production --name=jwt
infisical folders create --path=/radgateway/production --name=database
infisical folders create --path=/radgateway/production --name=api_keys
infisical folders create --path=/radgateway/production --name=redis
infisical folders create --path=/radgateway/production --name=providers

# Set secrets
infisical secrets set jwt_access_secret=$(openssl rand -base64 32) --path=/radgateway/production/jwt
infisical secrets set jwt_refresh_secret=$(openssl rand -base64 32) --path=/radgateway/production/jwt
infisical secrets set dsn="postgresql://..." --path=/radgateway/production/database
infisical secrets set keys="prod:$(openssl rand -hex 32)" --path=/radgateway/production/api_keys

# Create service token
infisical service-tokens create --name="radgateway-production" --scope="/radgateway/production/*" --access=read

# List secrets
infisical secrets get --path=/radgateway/production/jwt
```

---

**Document History**

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-02-18 | Security Engineer | Initial secret audit |

---

**Classification:** INTERNAL USE ONLY
**Distribution:** Team Charlie (Security), Team Hotel (Deployment), Team Golf (Documentation)
