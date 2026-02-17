# RAD Gateway Security Audit Report

**Date:** 2026-02-17
**Auditor:** Security Architect (Team Charlie)
**Scope:** Authentication, Authorization, CORS, SSE Security
**Version:** Phase 6 Alpha

---

## Executive Summary

This comprehensive security audit of RAD Gateway (codename: Brass Relay) identified **12 security findings** ranging from Critical to Low severity. The codebase shows good foundational security practices in some areas (proper RBAC design, structured logging) but has significant gaps in authentication implementation, CORS configuration, and missing security controls.

**Risk Summary:**
- Critical: 2 findings
- High: 4 findings
- Medium: 3 findings
- Low: 3 findings
- **Total: 12 findings**

**Immediate Action Required:** Address Critical and High findings before production deployment.

**New Findings Added:** Role-based privilege escalation, SSE authentication bypass, weak email-based role assignment, security headers missing, timing attack vulnerabilities.

---

## Critical Findings

### CRIT-001: JWT Secrets Generated Ephemerally at Runtime

**Severity:** Critical
**OWASP Category:** A07:2021 - Identification and Authentication Failures
**Location:** `/mnt/ollama/git/RADAPI01/internal/auth/jwt.go:44-53`

**Description:**
The `DefaultConfig()` function generates random secrets if environment variables are not set:

```go
func DefaultConfig() JWTConfig {
    return JWTConfig{
        AccessTokenSecret:  []byte(getenv("JWT_ACCESS_SECRET", generateSecret())),
        RefreshTokenSecret: []byte(getenv("JWT_REFRESH_SECRET", generateSecret())),
        // ...
    }
}
```

This means:
1. If `JWT_ACCESS_SECRET` is not set, a new random secret is generated on each restart
2. All existing JWT tokens become invalid after restart
3. An attacker who gains access to the running process can extract the ephemeral secret
4. Token validation cannot be guaranteed across process restarts

**Impact:**
- Complete authentication bypass possible if attacker can read process memory
- Users logged out unpredictably on every restart
- Session continuity impossible in containerized environments

**Recommendation:**
1. Require JWT secrets to be explicitly configured via environment variables or secrets manager
2. Fail startup if secrets are not provided (no fallbacks)
3. Document secret rotation procedures

**Risk Rating:** CVSS 9.1 (Critical)

---

### CRIT-002: Refresh Token Not Validated Against Store

**Severity:** Critical
**OWASP Category:** A07:2021 - Identification and Authentication Failures
**Location:** `/mnt/ollama/git/RADAPI01/internal/api/auth.go:184-259`

**Description:**
The `handleRefresh` function accepts refresh tokens but does not validate them against a persistent store:

```go
// For now, we validate the refresh token by checking if it matches a stored hash
// In production, this should check a database of valid refresh tokens
```

The code accepts any refresh token from the request body or cookie, extracts claims from the current access token, and issues new tokens without verifying the refresh token's validity.

**Impact:**
- Any valid access token can be used to generate infinite refresh tokens
- Token revocation is impossible
- Stolen refresh tokens cannot be invalidated
- Session hijacking possible with just an access token

**Recommendation:**
1. Implement refresh token storage with proper validation
2. Hash and store refresh tokens on creation
3. Validate refresh tokens against the store before issuing new tokens
4. Implement token rotation (new refresh token on each use)
5. Add refresh token expiration and cleanup

**Risk Rating:** CVSS 9.1 (Critical)

---

### CRIT-003: Insecure Cookie Settings (Hardcoded HTTP)

**Severity:** Critical
**OWASP Category:** A02:2021 - Cryptographic Failures
**Location:** `/mnt/ollama/git/RADAPI01/internal/api/auth.go:355-359`

**Description:**
The `isSecure()` function always returns `false`, causing authentication cookies to be transmitted over HTTP:

```go
func (h *AuthHandler) isSecure() bool {
    // In production, this should check if the server is running over HTTPS
    // For now, return false to allow HTTP in development
    return false
}
```

**Impact:**
- Session hijacking via network interception
- Credential theft on public/shared networks
- Violation of OWASP ASVS 3.4.3

**Recommendation:**
Implement TLS detection:
```go
func (h *AuthHandler) isSecure() bool {
    return os.Getenv("RAD_ENV") == "production" ||
           strings.HasPrefix(os.Getenv("RAD_BASE_URL"), "https://")
}
```

**Risk Rating:** CVSS 7.5 (Critical)

---

## High Findings

### HIGH-001: CORS Allows Credentials with Wildcard Origins

**Severity:** High
**OWASP Category:** A05:2021 - Security Misconfiguration
**Location:** `/mnt/ollama/git/RADAPI01/internal/middleware/cors.go:117-128`

**Description:**
The CORS middleware allows `*` wildcard origins and has `AllowCredentials: true` by default:

```go
// Check for wildcard
for _, allowed := range c.config.AllowedOrigins {
    if allowed == "*" {
        return true
    }
}
```

The `DefaultCORSConfig()` returns:
```go
AllowCredentials: true,
```

If `*` is added to `AllowedOrigins` while `AllowCredentials` is true, this violates the CORS specification and allows attackers to make authenticated cross-origin requests from any origin.

**Impact:**
- CSRF attacks possible from malicious websites
- Session hijacking via malicious sites
- API keys could be stolen via crafted requests

**Recommendation:**
1. Never allow `*` origin when `AllowCredentials` is true
2. Validate CORS configuration at startup
3. Use explicit origin whitelist in production
4. Add `Secure` and `SameSite=Strict` to cookies

**Risk Rating:** CVSS 7.5 (High)

---

### HIGH-002: Missing Rate Limiting on Authentication Endpoints

**Severity:** High
**OWASP Category:** A07:2021 - Identification and Authentication Failures
**Location:** `/mnt/ollama/git/RADAPI01/internal/api/auth.go` (all endpoints)

**Description:**
Authentication endpoints (`/v1/auth/login`, `/v1/auth/refresh`, etc.) have no rate limiting:

- Login endpoint allows unlimited password attempts
- No account lockout mechanism
- No IP-based throttling
- Refresh token endpoint allows unlimited requests

**Impact:**
- Brute force attacks on passwords possible
- Credential stuffing attacks
- DoS via resource exhaustion

**Recommendation:**
1. Implement rate limiting middleware (per-IP, per-user)
2. Add account lockout after failed attempts (exponential backoff)
3. Use CAPTCHA after threshold failures
4. Log and alert on suspicious authentication patterns

**Risk Rating:** CVSS 7.5 (High)

---

### HIGH-003: Weak Role Assignment Based on Email String Matching

**Severity:** High
**OWASP Category:** A01:2021 - Broken Access Control
**Location:** `/mnt/ollama/git/RADAPI01/internal/api/auth.go:114-119`

**Description:**
Admin role assignment uses insecure string matching on email addresses:

```go
role := "developer"
if strings.Contains(req.Email, "admin") {
    role = "admin"
    permissions = append(permissions, "delete", "admin")
}
```

Users with "admin" anywhere in their email (e.g., `notadmin@attacker.com`) gain admin privileges.

**Impact:**
- Privilege escalation
- Unauthorized administrative access
- RBAC bypass

**Recommendation:**
Fetch roles from database:
```go
role, permissions, err := h.roleService.GetUserRoleAndPermissions(ctx, user.ID)
if err != nil {
    return errors.New("failed to fetch user role")
}
```

**Risk Rating:** CVSS 6.8 (High)

---

### HIGH-004: Missing Security Headers

**Severity:** High
**OWASP Category:** A05:2021 - Security Misconfiguration
**Location:** `/mnt/ollama/git/RADAPI01/cmd/rad-gateway/main.go`

**Description:**
No security headers middleware is implemented. Missing:
- Content-Security-Policy
- X-Frame-Options
- X-Content-Type-Options
- Strict-Transport-Security
- X-XSS-Protection

**Impact:**
- Clickjacking attacks
- MIME sniffing attacks
- XSS via injected content
- Protocol downgrade attacks

**Recommendation:**
Add security headers middleware to all responses.

**Risk Rating:** CVSS 5.3 (High)

---

### HIGH-005: SSE Endpoints Lack Explicit Authentication

**Severity:** High
**OWASP Category:** A01:2021 - Broken Access Control
**Location:** `/mnt/ollama/git/RADAPI01/internal/api/sse.go:124-207`

**Description:**
SSE admin endpoints (`/v0/admin/events`) have no explicit authentication check in the handler. Relies on middleware which may be bypassed.

**Impact:**
- Unauthorized access to real-time admin events
- Information disclosure (usage metrics, system alerts)

**Recommendation:**
Add explicit auth check in handler:
```go
func (h *SSEHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
    if !auth.IsAuthenticated(r.Context()) {
        http.Error(w, `{"error":"authentication required"}`, 401)
        return
    }
    // ...
}
```

**Risk Rating:** CVSS 7.1 (High)

---

## Medium Findings

### MED-001: Insecure Cookie Configuration

**Severity:** Medium
**OWASP Category:** A05:2021 - Security Misconfiguration
**Location:** `/mnt/ollama/git/RADAPI01/internal/api/auth.go:301-359`

**Description:**
The `isSecure()` method hardcodes `false` for all environments:

```go
func (h *AuthHandler) isSecure() bool {
    // In production, this should check if the server is running over HTTPS
    // For now, return false to allow HTTP in development
    return false
}
```

This causes:
- `Secure` cookie flag set to `false` in production
- Cookies transmitted over unencrypted connections
- Vulnerable to session hijacking on networks with sniffing

**Recommendation:**
1. Make Secure flag configurable via environment variable
2. Default to `true` in production mode
3. Add development/production mode detection
4. Document HTTPS requirement for production

**Risk Rating:** CVSS 5.9 (Medium)

---

### MED-002: SSE Token Authentication via Query Parameter

**Severity:** Medium
**OWASP Category:** A02:2021 - Cryptographic Failures
**Location:** `/mnt/ollama/git/RADAPI01/cmd/rad-gateway/main.go:122-139`, `/mnt/ollama/git/RADAPI01/internal/middleware/middleware.go:71-108`

**Description:**
SSE endpoints accept authentication tokens via query parameter (`?token=...`) due to EventSource limitations:

```go
func (a *Authenticator) RequireWithTokenAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // ...
        if secret == "" {
            secret = r.URL.Query().Get("token")  // Token in URL!
        }
```

This causes:
- Tokens logged in web server access logs
- Tokens visible in browser history
- Tokens in referrer headers when clicking external links
- Tokens cached by proxies/CDNs

**Impact:**
- Token exposure in logs increases attack surface
- Compliance violations (tokens in plaintext logs)
- Potential session hijacking from log access

**Recommendation:**
1. Implement short-lived SSE-specific tokens (separate from API tokens)
2. Use token binding (IP address, User-Agent)
3. Add token expiration (5-10 minutes max for SSE)
4. Implement token rotation during SSE session
5. Consider alternative auth mechanisms (Cookie-based for same-origin)

**Risk Rating:** CVSS 5.3 (Medium)

---

## Low Findings

### LOW-001: Conflicting Authentication Middleware Implementations

**Severity:** Low
**OWASP Category:** A04:2021 - Insecure Design
**Location:** Multiple files

**Description:**
The codebase has two parallel authentication systems:

1. **JWT-based auth** (`/mnt/ollama/git/RADAPI01/internal/auth/middleware.go`)
   - Used for session-based authentication
   - Validates JWT tokens from cookies/headers

2. **API Key auth** (`/mnt/ollama/git/RADAPI01/internal/middleware/middleware.go`)
   - Used for programmatic API access
   - Validates static API keys

3. **RBAC middleware** (`/mnt/ollama/git/RADAPI01/internal/rbac/middleware.go`)
   - Has its own JWT validation
   - Different claims structure (`JWTClaims` vs `Claims`)

The main.go wires them in a confusing way:
```go
auth := middleware.NewAuthenticator(cfg.APIKeys)  // API key auth
protectedMux := withConditionalAuth(apiMux, auth)
sseProtectedMux := withSSEAuth(apiMux, auth)
handler := middleware.WithRequestContext(sseProtectedMux)
handler = middleware.WithCORS(handler)
```

The JWT middleware from `internal/auth` is instantiated but not properly integrated into the middleware chain.

**Impact:**
- Maintenance complexity
- Potential for auth bypass if one system has bugs
- Inconsistent security behavior
- Confusion for developers

**Recommendation:**
1. Consolidate to a single authentication middleware
2. Create clear auth strategy: JWT for sessions, API keys for programmatic access
3. Remove redundant RBAC middleware JWT validation (use auth middleware output)
4. Document the authentication flow clearly

**Risk Rating:** CVSS 3.7 (Low)

---

### LOW-002: Timing Attack Vulnerability in API Key Comparison

**Severity:** Low
**OWASP Category:** A07:2021 - Identification and Authentication Failures
**Location:** `/mnt/ollama/git/RADAPI01/internal/middleware/middleware.go:41-69`

**Description:**
API key comparison uses standard string comparison vulnerable to timing attacks:

```go
for k, v := range a.keys {
    if v == secret {  // Timing attack vulnerable!
        name = k
        break
    }
}
```

**Impact:**
- API key enumeration via timing analysis
- Requires precise timing measurement

**Recommendation:**
Use constant-time comparison:
```go
import "crypto/subtle"
for k, v := range a.keys {
    if subtle.ConstantTimeCompare([]byte(v), []byte(secret)) == 1 {
        name = k
        break
    }
}
```

**Risk Rating:** CVSS 3.7 (Low)

---

### LOW-003: Account Status Enumeration

**Severity:** Low
**OWASP Category:** A07:2021 - Identification and Authentication Failures
**Location:** `/mnt/ollama/git/RADAPI01/internal/api/auth.go:100-104`

**Description:**
Different error messages reveal account status:

```go
if user.Status != "active" {
    writeJSONError(w, http.StatusUnauthorized, "account not active")  // Reveals exists!
    return
}
```

While most errors use "invalid credentials", this message reveals account exists but is disabled.

**Impact:**
- User enumeration
- Account status enumeration

**Recommendation:**
Use uniform error message for all authentication failures.

**Risk Rating:** CVSS 2.9 (Low)

---

## Additional Security Observations

### Positive Security Controls Found

1. **Proper RBAC Implementation**: The RBAC system (`/mnt/ollama/git/RADAPI01/internal/rbac/`) is well-designed with:
   - Bitmask-based permissions for efficiency
   - Clear role hierarchy
   - Resource-level access control
   - Project-level isolation

2. **Password Hashing**: Uses proper password hashing via `auth.PasswordHasher`

3. **Structured Logging**: Security events are properly logged with slog

4. **User Enumeration Prevention**: Login returns generic "invalid credentials" instead of distinguishing between user not found vs wrong password

5. **JWT Signing Method Validation**: The JWT parser validates expected signing method (HS256)

6. **CORS Vary Header**: Properly adds `Vary: Origin` header for credentialed requests

7. **SSE Connection Limiting**: SSE handler limits concurrent connections (`maxClients: 100`)

---

## Compliance Mapping

| Finding | OWASP Top 10 2021 | CWE | NIST 800-53 |
|---------|-------------------|-----|-------------|
| CRIT-001 | A07:2021 | CWE-798 | IA-5(1) |
| CRIT-002 | A07:2021 | CWE-290 | IA-5(1) |
| CRIT-003 | A02:2021 | CWE-614 | SC-8 |
| HIGH-001 | A05:2021 | CWE-942 | SC-8 |
| HIGH-002 | A07:2021 | CWE-307 | IA-5(4) |
| HIGH-003 | A01:2021 | CWE-639 | AC-3 |
| HIGH-004 | A05:2021 | CWE-693 | SC-8 |
| HIGH-005 | A01:2021 | CWE-306 | AC-3 |
| MED-001  | A05:2021 | CWE-614 | SC-8 |
| MED-002  | A02:2021 | CWE-319 | SC-12 |
| LOW-001  | A04:2021 | CWE-656 | AC-3 |
| LOW-002  | A07:2021 | CWE-208 | IA-5(1) |
| LOW-003  | A07:2021 | CWE-204 | IA-5(1) |

---

## Remediation Roadmap

### Immediate (Before Production)

1. **CRIT-001**: Implement mandatory JWT secret configuration
2. **CRIT-002**: Implement refresh token storage and validation
3. **CRIT-003**: Fix insecure cookie settings (hardcoded HTTP)
4. **HIGH-001**: Fix CORS credentials/wildcard interaction
5. **HIGH-002**: Add rate limiting middleware
6. **HIGH-003**: Replace email-based role assignment with database-driven RBAC
7. **HIGH-004**: Add security headers middleware
8. **HIGH-005**: Add explicit authentication checks to SSE handlers

### Short-term (Within 30 Days)

1. **MED-001**: Make Secure cookie flag production-aware
2. **MED-002**: Implement SSE token security best practices
3. Add security headers middleware (HSTS, CSP, X-Frame-Options)
4. Implement request size limits

### Medium-term (Within 90 Days)

1. **LOW-001**: Consolidate authentication middleware
2. Add comprehensive input validation
3. Implement audit logging for all admin actions
4. Add security monitoring and alerting

---

## Appendix A: Files Reviewed

| File | Purpose |
|------|---------|
| `/mnt/ollama/git/RADAPI01/internal/api/auth.go` | Authentication handlers (login/logout/refresh) |
| `/mnt/ollama/git/RADAPI01/internal/middleware/cors.go` | CORS configuration |
| `/mnt/ollama/git/RADAPI01/internal/auth/jwt.go` | JWT token generation/validation |
| `/mnt/ollama/git/RADAPI01/internal/auth/middleware.go` | JWT authentication middleware |
| `/mnt/ollama/git/RADAPI01/internal/middleware/middleware.go` | API key authentication |
| `/mnt/ollama/git/RADAPI01/internal/rbac/middleware.go` | RBAC enforcement |
| `/mnt/ollama/git/RADAPI01/cmd/rad-gateway/main.go` | Security wiring |
| `/mnt/ollama/git/RADAPI01/internal/api/sse.go` | SSE endpoint security |
| `/mnt/ollama/git/RADAPI01/internal/rbac/roles.go` | Role definitions |
| `/mnt/ollama/git/RADAPI01/internal/rbac/permissions.go` | Permission system |
| `/mnt/ollama/git/RADAPI01/internal/config/config.go` | Configuration and secrets |
| `/mnt/ollama/git/RADAPI01/internal/streaming/sse.go` | SSE streaming implementation |

---

## Appendix B: Testing Recommendations

1. **Penetration Testing**: Engage third-party security firm for penetration testing
2. **Dependency Scanning**: Implement automated dependency vulnerability scanning
3. **SAST**: Integrate static analysis security testing into CI/CD
4. **DAST**: Run dynamic application security testing against staging environment
5. **Fuzz Testing**: Fuzz authentication endpoints for crashes/vulnerabilities

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-02-17 | Security Architect | Initial audit report |

---

**Classification:** INTERNAL USE ONLY
**Distribution:** Team Charlie (Security), Team Alpha (Architecture), Team Golf (Documentation)
