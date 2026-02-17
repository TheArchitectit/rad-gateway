# RAD Gateway Admin UI - Security Architecture Document

**Version:** 1.0.0
**Date:** 2026-02-17
**Status:** Security Review Complete - ACTION REQUIRED
**Author:** Security Engineer (Agent 4: Frontend Security Reviewer)
**Classification:** CONFIDENTIAL - Security Critical

---

## Executive Summary

This document defines the comprehensive security architecture for the RAD Gateway Admin UI. It addresses critical vulnerabilities identified in the current implementation and establishes defense-in-depth security controls.

### Critical Finding

**SEVERITY: CRITICAL** - The `/v0/management/*` endpoints currently have **NO AUTHENTICATION**.

- **Location:** `/mnt/ollama/git/RADAPI01/cmd/rad-gateway/main.go:68`
- **Issue:** Management endpoints bypass all authentication via `withConditionalAuth`
- **Impact:** Any unauthorized user can access sensitive data including:
  - Full gateway configuration (`/v0/management/config`)
  - Usage records with API keys (`/v0/management/usage`)
  - Request traces with potentially sensitive data (`/v0/management/traces`)

**Immediate Action Required:** Implement authentication before deployment to production.

---

## Table of Contents

1. [Threat Model](#1-threat-model)
2. [Authentication Architecture](#2-authentication-architecture)
3. [Authorization & RBAC](#3-authorization--rbac)
4. [Token Storage Strategy](#4-token-storage-strategy)
5. [XSS Protection](#5-xss-protection)
6. [CSRF Protection](#6-csrf-protection)
7. [Security Headers](#7-security-headers)
8. [Session Management](#8-session-management)
9. [Audit Logging](#9-audit-logging)
10. [API Security](#10-api-security)
11. [Implementation Roadmap](#11-implementation-roadmap)
12. [Security Checklist](#12-security-checklist)

---

## 1. Threat Model

### 1.1 Identified Threats

| Threat ID | Threat | Severity | Mitigation |
|-----------|--------|----------|------------|
| T-001 | Unauthenticated access to admin endpoints | CRITICAL | Implement JWT-based auth |
| T-002 | XSS attacks via injected payloads | HIGH | CSP, input sanitization, output encoding |
| T-003 | CSRF attacks on state-changing operations | HIGH | CSRF tokens, SameSite cookies |
| T-004 | Token theft via XSS | HIGH | httpOnly cookies, short-lived tokens |
| T-005 | Session hijacking | HIGH | Secure cookies, rotation, binding |
| T-006 | Privilege escalation | HIGH | RBAC enforcement, permission checks |
| T-007 | API key exposure in logs/responses | MEDIUM | Key masking, secure logging |
| T-008 | Brute force attacks on login | MEDIUM | Rate limiting, account lockout |
| T-009 | Sensitive data in browser storage | MEDIUM | httpOnly cookies, memory-only tokens |
| T-010 | Man-in-the-middle attacks | MEDIUM | TLS 1.3, HSTS, certificate pinning |

### 1.2 Attack Scenarios

#### Scenario 1: Direct API Access
**Attacker:** External actor with network access
**Attack:** Direct GET request to `/v0/management/config`
**Current State:** SUCCEEDS - No authentication required
**Required Mitigation:** JWT validation middleware

#### Scenario 2: Stored XSS via Provider Config
**Attacker:** User with Developer role
**Attack:** Inject `<script>` in provider name field
**Impact:** Admin session hijacking when admin views providers
**Required Mitigation:** Output encoding, CSP, input validation

#### Scenario 3: CSRF on API Key Creation
**Attacker:** Malicious website
**Attack:** Forge POST request to create API key
**Impact:** Unauthorized API key creation
**Required Mitigation:** CSRF tokens, SameSite=Lax cookies

---

## 2. Authentication Architecture

### 2.1 Authentication Flow

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│   Browser   │────▶│   Backend    │────▶│    Database  │
└─────────────┘     └──────────────┘     └──────────────┘
       │                    │                    │
       │ 1. POST /auth/login│                    │
       │ {email, password}  │                    │
       │───────────────────▶│                    │
       │                    │                    │
       │                    │ 2. Verify password │
       │                    │ 3. Check MFA       │
       │                    │◀───────────────────│
       │                    │                    │
       │ 4. Set-Cookie:     │                    │
       │    access_token    │                    │
       │    (httpOnly,      │                    │
       │     Secure,        │                    │
       │     SameSite=Lax)  │                    │
       │◀───────────────────│                    │
       │                    │                    │
       │ 5. Store CSRF      │                    │
       │    token in        │                    │
       │    memory store    │                    │
       │───────────────────▶│                    │
```

### 2.2 Token Architecture

**Dual-Token System:**

| Token Type | Storage | Lifetime | Purpose |
|------------|---------|----------|-----------|
| Access Token | httpOnly Cookie | 15 minutes | API authentication |
| Refresh Token | httpOnly Cookie | 7 days | Token rotation |
| CSRF Token | Memory (Zustand) + Header | Session | CSRF protection |

### 2.3 Backend Implementation

**New Files Required:**

```
/internal/auth/
├── handlers.go          # Login, logout, refresh endpoints
├── middleware.go        # JWT validation middleware
├── tokens.go           # Token generation/validation
├── csrf.go             # CSRF token management
└── rbac.go             # Permission checking
```

**JWT Claims Structure:**

```go
type JWTClaims struct {
    jwt.RegisteredClaims
    UserID      string   `json:"userId"`
    Email       string   `json:"email"`
    WorkspaceID string   `json:"workspaceId"`
    Role        string   `json:"role"`        // admin, developer, viewer
    Permissions []string `json:"permissions"` // explicit permissions
    SessionID   string   `json:"sessionId"`   // for revocation
}
```

### 2.4 Login Flow Specification

**Endpoint:** `POST /v0/auth/login`

**Request:**
```json
{
  "email": "user@example.com",
  "password": "securePassword123",
  "workspaceId": "optional-workspace-id"
}
```

**Success Response (200):**
```http
Set-Cookie: access_token=eyJhbG...; HttpOnly; Secure; SameSite=Lax; Max-Age=900; Path=/
Set-Cookie: refresh_token=eyJhbG...; HttpOnly; Secure; SameSite=Lax; Max-Age=604800; Path=/v0/auth/refresh
X-CSRF-Token: a1b2c3d4e5f6...

{
  "user": {
    "id": "user-123",
    "email": "user@example.com",
    "role": "developer",
    "workspace": {
      "id": "ws-456",
      "name": "Production"
    }
  },
  "csrfToken": "a1b2c3d4e5f6...",
  "expiresIn": 900
}
```

**Error Responses:**
- `401 Unauthorized`: Invalid credentials
- `403 Forbidden`: Account disabled or workspace access denied
- `429 Too Many Requests`: Rate limit exceeded

### 2.5 Token Refresh Flow

**Endpoint:** `POST /v0/auth/refresh`

Automatically called by frontend when access token expires (5 min before expiry).

**Response:**
```http
Set-Cookie: access_token=eyJhbG...; HttpOnly; Secure; SameSite=Lax; Max-Age=900; Path=/

{
  "csrfToken": "new-csrf-token...",
  "expiresIn": 900
}
```

**Security Considerations:**
- Refresh token rotation: New refresh token issued on each use
- Refresh token binding to session fingerprint
- Maximum refresh token lifetime: 7 days
- Automatic revocation on suspicious activity

### 2.6 Logout Flow

**Endpoint:** `POST /v0/auth/logout`

**Actions:**
1. Invalidate session in database
2. Clear cookies (set expired)
3. Add tokens to revocation list (if using JWT blacklist)
4. Clear CSRF token from memory

---

## 3. Authorization & RBAC

### 3.1 Role Definitions

Based on existing database models and frontend spec:

| Role | Description | Permissions |
|------|-------------|-------------|
| **Admin** | Full workspace control | `*:*` (all actions on all resources) |
| **Developer** | Can configure providers, view analytics | `providers:*`, `apikeys:*`, `usage:read`, `traces:read`, `controlrooms:*` |
| **Viewer** | Read-only access | `usage:read`, `traces:read`, `providers:read`, `controlrooms:read` |

### 3.2 Permission Matrix

| Resource | Action | Admin | Developer | Viewer |
|----------|--------|-------|-----------|--------|
| Workspaces | read | Y | Y | Y |
| | create | Y | N | N |
| | update | Y | N | N |
| | delete | Y | N | N |
| Providers | read | Y | Y | Y |
| | create | Y | Y | N |
| | update | Y | Y | N |
| | delete | Y | N | N |
| API Keys | read | Y | Y | N |
| | create | Y | Y | N |
| | revoke | Y | Y | N |
| Usage | read | Y | Y | Y |
| Traces | read | Y | Y | Y |
| Control Rooms | read | Y | Y | Y |
| | create | Y | Y | N |
| | update | Y | Y | N |
| Users | read | Y | N | N |
| | invite | Y | N | N |
| | update | Y | N | N |
| Settings | read | Y | Y | N |
| | update | Y | N | N |

### 3.3 Middleware Implementation

**Required Middleware Stack (Order Matters):**

```go
// Request handling order:
1. Security Headers (CSP, HSTS, etc.)
2. CORS (for API)
3. Rate Limiting
4. Request ID / Logging
5. CSRF Protection (for state-changing methods)
6. Authentication (JWT validation)
7. Authorization (RBAC check)
8. Audit Logging
```

**RBAC Middleware:**

```go
// RequirePermission checks if user has required permission
func RequirePermission(permission string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := GetUserFromContext(r.Context())
            if user == nil {
                http.Error(w, `{"error":"unauthorized"}`, 401)
                return
            }

            if !user.HasPermission(permission) {
                http.Error(w, `{"error":"forbidden"}`, 403)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

// Usage:
// mux.Handle("/v0/providers", RequirePermission("providers:read")(handlers))
```

### 3.4 Frontend Authorization

**Route Protection:**

```typescript
// Route configuration with role-based access
const routes = [
  {
    path: '/app/providers',
    component: ProvidersPage,
    requiredPermission: 'providers:read'
  },
  {
    path: '/app/providers/new',
    component: CreateProviderPage,
    requiredPermission: 'providers:create'
  },
  {
    path: '/app/admin/users',
    component: UsersPage,
    requiredPermission: 'users:read',
    requiredRole: 'admin'
  }
];

// ProtectedRoute component
function ProtectedRoute({ children, requiredPermission }: Props) {
  const { user, hasPermission } = useAuth();

  if (!user) return <Navigate to="/login" />;
  if (!hasPermission(requiredPermission)) return <ForbiddenPage />;

  return children;
}
```

---

## 4. Token Storage Strategy

### 4.1 Security Comparison

| Storage | XSS Risk | CSRF Risk | Persistence | Recommendation |
|---------|----------|-----------|-------------|----------------|
| localStorage | HIGH | LOW | Permanent | NEVER for tokens |
| sessionStorage | MEDIUM | LOW | Session only | Acceptable for CSRF token only |
| httpOnly Cookie | LOW | MEDIUM | Configurable | REQUIRED for JWT |
| Memory (React state) | LOW | LOW | Lost on refresh | Acceptable for CSRF token |

### 4.2 Recommended Storage Strategy

**Access Token:**
- Storage: httpOnly, Secure, SameSite=Lax cookie
- Lifetime: 15 minutes
- Path: `/v0/`
- JavaScript: NOT accessible

**Refresh Token:**
- Storage: httpOnly, Secure, SameSite=Strict cookie
- Lifetime: 7 days
- Path: `/v0/auth/refresh` (isolated)
- JavaScript: NOT accessible

**CSRF Token:**
- Storage: React Context/Zustand (memory only)
- Sent via: `X-CSRF-Token` header on mutations
- Lifetime: Session or access token lifetime
- Regenerated: On token refresh

### 4.3 Cookie Configuration

```go
// Backend cookie setting
func setAuthCookies(w http.ResponseWriter, accessToken, refreshToken string) {
    // Access token cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "access_token",
        Value:    accessToken,
        HttpOnly: true,
        Secure:   true, // Requires HTTPS
        SameSite: http.SameSiteLaxMode,
        MaxAge:   900, // 15 minutes
        Path:     "/v0/",
    })

    // Refresh token cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "refresh_token",
        Value:    refreshToken,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
        MaxAge:   604800, // 7 days
        Path:     "/v0/auth/refresh",
    })
}
```

### 4.4 Frontend Token Management

```typescript
// authStore.ts
interface AuthState {
  user: User | null;
  csrfToken: string | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshToken: () => Promise<void>;
}

// API client with automatic token refresh
const apiClient = axios.create({
  baseURL: '/v0',
  withCredentials: true, // Send cookies automatically
});

// Add CSRF token to mutations
apiClient.interceptors.request.use((config) => {
  if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(config.method?.toUpperCase() || '')) {
    const csrfToken = useAuthStore.getState().csrfToken;
    config.headers['X-CSRF-Token'] = csrfToken;
  }
  return config;
});

// Auto-refresh on 401
apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401 && !error.config._retry) {
      error.config._retry = true;
      await useAuthStore.getState().refreshToken();
      return apiClient(error.config);
    }
    return Promise.reject(error);
  }
);
```

---

## 5. XSS Protection

### 5.1 Content Security Policy (CSP)

**Recommended CSP Header:**

```http
Content-Security-Policy:
  default-src 'self';
  script-src 'self' 'nonce-{random}' https://cdn.jsdelivr.net;
  style-src 'self' 'unsafe-inline' https://fonts.googleapis.com;
  font-src 'self' https://fonts.gstatic.com;
  img-src 'self' data: https:;
  connect-src 'self' wss://gateway.example.com;
  frame-ancestors 'none';
  base-uri 'self';
  form-action 'self';
  upgrade-insecure-requests;
```

**Backend Implementation:**

```go
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        nonce := generateNonce()
        ctx := context.WithValue(r.Context(), "csp-nonce", nonce)

        w.Header().Set("Content-Security-Policy",
            fmt.Sprintf("default-src 'self'; script-src 'self' 'nonce-%s'; ...", nonce))
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 5.2 Output Encoding

**React (Default Safe):**
- React automatically escapes content rendered in JSX
- Use `dangerouslySetInnerHTML` ONLY with DOMPurify

```typescript
import DOMPurify from 'dompurify';

// Safe: React escapes by default
<div>{userInput}</div>

// UNSAFE - Never do this
<div dangerouslySetInnerHTML={{ __html: userInput }} />

// Safe with sanitization
<div dangerouslySetInnerHTML={{
  __html: DOMPurify.sanitize(userInput, { ALLOWED_TAGS: ['b', 'i', 'em'] })
}} />
```

**URL Sanitization:**

```typescript
// Safe URL construction
const sanitizedUrl = new URL(userInput, window.location.origin).toString();

// For provider URLs (backend responsibility)
func sanitizeProviderURL(rawURL string) (string, error) {
    u, err := url.Parse(rawURL)
    if err != nil {
        return "", err
    }
    // Only allow https
    if u.Scheme != "https" {
        return "", errors.New("only HTTPS URLs allowed")
    }
    return u.String(), nil
}
```

### 5.3 Input Validation

**Backend Validation (Go):**

```go
// validateProviderRequest validates provider configuration
type CreateProviderRequest struct {
    Name     string `json:"name" validate:"required,min=1,max=100,alphanumspace"`
    BaseURL  string `json:"baseUrl" validate:"required,url,https"`
    APIKey   string `json:"apiKey" validate:"required,min=32"`
    ProviderType string `json:"providerType" validate:"required,oneof=openai anthropic gemini"`
}

func (r CreateProviderRequest) Validate() error {
    validate := validator.New()
    return validate.Struct(r)
}
```

**Frontend Validation (Zod):**

```typescript
import { z } from 'zod';

const providerSchema = z.object({
  name: z.string().min(1).max(100).regex(/^[\w\s-]+$/),
  baseUrl: z.string().url().refine(url => url.startsWith('https://'), {
    message: 'Only HTTPS URLs allowed'
  }),
  apiKey: z.string().min(32),
  providerType: z.enum(['openai', 'anthropic', 'gemini'])
});
```

---

## 6. CSRF Protection

### 6.1 Threat Overview

**Attack:** Malicious site tricks authenticated user into submitting unwanted request
**Impact:** Unauthorized state changes (create API key, delete provider, etc.)
**Mitigation:** CSRF tokens + SameSite cookies

### 6.2 CSRF Token Implementation

**Token Generation (Backend):**

```go
// Generate CSRF token (32 bytes random, base64 encoded)
func generateCSRFToken() string {
    b := make([]byte, 32)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)
}

// CSRF middleware
func CSRFProtection(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Skip for safe methods
        if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
            next.ServeHTTP(w, r)
            return
        }

        // Validate CSRF token
        token := r.Header.Get("X-CSRF-Token")
        sessionToken := getSessionCSRFToken(r)

        if !secureCompare(token, sessionToken) {
            http.Error(w, `{"error":"invalid CSRF token"}`, 403)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

### 6.3 Frontend CSRF Handling

```typescript
// CSRF token from login/refresh response
interface AuthResponse {
  csrfToken: string;
  user: User;
}

// Store in Zustand (memory only)
const useAuthStore = create<AuthState>((set, get) => ({
  csrfToken: null,

  setCSRFToken: (token) => set({ csrfToken: token }),

  // Auto-attach to mutations
  apiCall: async (method, url, data) => {
    const config: AxiosRequestConfig = {
      method,
      url,
      data,
      withCredentials: true,
    };

    // Add CSRF token for state-changing operations
    if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(method)) {
      config.headers = {
        'X-CSRF-Token': get().csrfToken
      };
    }

    return apiClient(config);
  }
}));
```

---

## 7. Security Headers

### 7.1 Required Headers

| Header | Value | Purpose |
|--------|-------|---------|
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains; preload` | Force HTTPS |
| `X-Content-Type-Options` | `nosniff` | Prevent MIME sniffing |
| `X-Frame-Options` | `DENY` | Prevent clickjacking |
| `X-XSS-Protection` | `1; mode=block` | XSS filter (legacy) |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Limit referrer info |
| `Content-Security-Policy` | See section 5.1 | XSS protection |
| `Permissions-Policy` | `geolocation=(), microphone=(), camera=()` | Restrict features |

### 7.2 Middleware Implementation

```go
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // HSTS - only in production
        if isProduction {
            w.Header().Set("Strict-Transport-Security",
                "max-age=31536000; includeSubDomains; preload")
        }

        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        w.Header().Set("Permissions-Policy",
            "geolocation=(), microphone=(), camera=(), payment=()")

        // CSP with nonce for inline scripts
        nonce := generateNonce()
        w.Header().Set("Content-Security-Policy",
            fmt.Sprintf("default-src 'self'; script-src 'self' 'nonce-%s'; style-src 'self' 'unsafe-inline'; ...", nonce))

        next.ServeHTTP(w, r)
    })
}
```

---

## 8. Session Management

### 8.1 Session Architecture

**Session Store Options:**

| Store | Pros | Cons | Recommendation |
|-------|------|------|----------------|
| In-Memory (Go map) | Fast, simple | No persistence, no clustering | Development only |
| Redis | Fast, TTL, clustering | Additional dependency | Production recommended |
| PostgreSQL | Persistent, ACID | Slower, more DB load | Alternative to Redis |

**Recommended: Redis**

```go
// Session structure
type Session struct {
    ID            string    `redis:"id"`
    UserID        string    `redis:"user_id"`
    WorkspaceID   string    `redis:"workspace_id"`
    Role          string    `redis:"role"`
    CSRFToken     string    `redis:"csrf_token"`
    IPAddress     string    `redis:"ip_address"`
    UserAgent     string    `redis:"user_agent"`
    CreatedAt     time.Time `redis:"created_at"`
    LastActiveAt  time.Time `redis:"last_active_at"`
    RefreshCount  int       `redis:"refresh_count"`
}
```

### 8.2 Session Security Features

**Session Fingerprinting:**

```go
func createSessionFingerprint(r *http.Request) string {
    // Hash of IP + User-Agent (not perfect, but helps)
    data := r.RemoteAddr + r.UserAgent()
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}

func validateSession(r *http.Request, session *Session) bool {
    fingerprint := createSessionFingerprint(r)
    // Allow slight variation for mobile networks
    return subtle.ConstantTimeCompare(
        []byte(session.Fingerprint),
        []byte(fingerprint)
    ) == 1
}
```

**Concurrent Session Limits:**

```go
func enforceSessionLimit(userID string, maxSessions int) error {
    sessions, _ := getUserSessions(userID)
    if len(sessions) >= maxSessions {
        // Revoke oldest session
        revokeSession(sessions[0].ID)
    }
    return nil
}
```

### 8.3 Session Cleanup

```go
// Background goroutine for session cleanup
func StartSessionCleanup(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Minute)
    go func() {
        for {
            select {
            case <-ticker.C:
                cleanupExpiredSessions()
            case <-ctx.Done():
                return
            }
        }
    }()
}
```

---

## 9. Audit Logging

### 9.1 Audit Event Types

| Category | Events |
|----------|--------|
| Authentication | login, logout, token_refresh, failed_login |
| Authorization | permission_denied, role_changed |
| Resource | provider_created, provider_updated, provider_deleted, apikey_created, apikey_revoked |
| Data | config_viewed, usage_exported, traces_accessed |
| Admin | user_invited, user_removed, workspace_created, settings_changed |

### 9.2 Audit Log Structure

```go
type AuditEvent struct {
    ID            string                 `json:"id"`
    Timestamp     time.Time              `json:"timestamp"`
    EventType     string                 `json:"eventType"`
    Severity      string                 `json:"severity"` // info, warning, critical
    UserID        string                 `json:"userId"`
    UserEmail     string                 `json:"userEmail"`
    WorkspaceID   string                 `json:"workspaceId"`
    IPAddress     string                 `json:"ipAddress"`
    UserAgent     string                 `json:"userAgent"`
    ResourceType  string                 `json:"resourceType,omitempty"`
    ResourceID    string                 `json:"resourceId,omitempty"`
    Action        string                 `json:"action"`
    Status        string                 `json:"status"` // success, failure
    Changes       map[string]interface{} `json:"changes,omitempty"` // before/after
    Metadata      map[string]interface{} `json:"metadata,omitempty"`
}
```

### 9.3 Audit Middleware

```go
func AuditLogger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Wrap response writer to capture status
        wrapped := &responseRecorder{ResponseWriter: w, statusCode: 200}

        next.ServeHTTP(wrapped, r)

        // Log after request completes
        duration := time.Since(start)
        event := AuditEvent{
            ID:           generateID(),
            Timestamp:    time.Now(),
            EventType:    "api_request",
            Severity:     determineSeverity(wrapped.statusCode),
            UserID:       getUserID(r.Context()),
            IPAddress:    getClientIP(r),
            UserAgent:    r.UserAgent(),
            Action:       r.Method + " " + r.URL.Path,
            Status:       http.StatusText(wrapped.statusCode),
            Metadata: map[string]interface{}{
                "duration_ms": duration.Milliseconds(),
                "status_code": wrapped.statusCode,
            },
        }

        auditLogChannel <- event
    })
}
```

### 9.4 Sensitive Data Masking

```go
func maskSensitiveData(data map[string]interface{}) map[string]interface{} {
    masked := make(map[string]interface{})
    for k, v := range data {
        switch {
        case strings.Contains(strings.ToLower(k), "password"):
            masked[k] = "[REDACTED]"
        case strings.Contains(strings.ToLower(k), "apikey"):
            masked[k] = maskAPIKey(fmt.Sprintf("%v", v))
        case strings.Contains(strings.ToLower(k), "secret"):
            masked[k] = "[REDACTED]"
        default:
            masked[k] = v
        }
    }
    return masked
}

func maskAPIKey(key string) string {
    if len(key) <= 8 {
        return "****"
    }
    return key[:4] + "****" + key[len(key)-4:]
}
```

---

## 10. API Security

### 10.1 Rate Limiting

**Rate Limit Configuration:**

| Endpoint | Limit | Window | Action |
|----------|-------|--------|--------|
| `/v0/auth/login` | 5 | 15 min | Account lockout after 5 failures |
| `/v0/auth/refresh` | 60 | 1 hour | Normal usage |
| `/v0/management/*` | 100 | 1 min | Standard API |
| `/v0/providers` | 30 | 1 min | Resource management |
| `/v0/usage` | 60 | 1 min | Data export |

**Implementation:**

```go
// Token bucket rate limiter
type RateLimiter struct {
    store  *redis.Client
    limits map[string]RateLimit
}

type RateLimit struct {
    Requests int
    Window   time.Duration
}

func (rl *RateLimiter) Allow(key string, limit RateLimit) bool {
    ctx := context.Background()
    pipe := rl.store.Pipeline()

    now := time.Now().Unix()
    window := now - int64(limit.Window.Seconds())

    // Remove old entries
    pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", window))
    // Count current entries
    pipe.ZCard(ctx, key)
    // Add current request
    pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
    // Set expiry
    pipe.Expire(ctx, key, limit.Window)

    results, _ := pipe.Exec(ctx)
    currentCount := results[1].(*redis.IntCmd).Val()

    return currentCount < int64(limit.Requests)
}
```

### 10.2 API Key Handling

**Current Issue:** API keys returned in `/v0/management/config` snapshot.

**Required Changes:**

1. Remove API keys from config snapshot
2. Return only key names and previews
3. Generate new keys via separate endpoint
4. Show key only once on creation

```go
// Safe config snapshot
func (c Config) PublicSnapshot() map[string]any {
    return map[string]any{
        "listenAddr":     c.ListenAddr,
        "retryBudget":    c.RetryBudget,
        "keysConfigured": len(c.APIKeys), // Count only
        "models":         c.ModelRoutes,
    }
}

// API key listing (safe)
type APIKeyResponse struct {
    ID         string    `json:"id"`
    Name       string    `json:"name"`
    Preview    string    `json:"preview"` // e.g., "sk-...abc"
    CreatedAt  time.Time `json:"createdAt"`
    LastUsedAt time.Time `json:"lastUsedAt,omitempty"`
}
```

### 10.3 Request/Response Validation

**Size Limits:**

```go
func ValidateRequestSize(maxSize int64) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            r.Body = http.MaxBytesReader(w, r.Body, maxSize)
            next.ServeHTTP(w, r)
        })
    }
}

// Usage: ValidateRequestSize(1 << 20) // 1MB limit
```

---

## 11. Implementation Roadmap

### Phase 1: Critical Fixes (Week 1)

- [ ] **SECURITY-001**: Add authentication middleware to `/v0/management/*`
- [ ] **SECURITY-002**: Remove API keys from config snapshot
- [ ] **SECURITY-003**: Add security headers middleware
- [ ] **SECURITY-004**: Implement rate limiting on auth endpoints

### Phase 2: Core Security (Weeks 2-3)

- [ ] **SECURITY-005**: Implement JWT authentication
- [ ] **SECURITY-006**: Implement session management (Redis)
- [ ] **SECURITY-007**: Implement RBAC middleware
- [ ] **SECURITY-008**: Add audit logging
- [ ] **SECURITY-009**: Add CSRF protection

### Phase 3: Frontend Security (Weeks 3-4)

- [ ] **SECURITY-010**: Implement login/logout flows
- [ ] **SECURITY-011**: Add CSRF token handling
- [ ] **SECURITY-012**: Implement protected routes
- [ ] **SECURITY-013**: Add permission-based UI hiding

### Phase 4: Hardening (Week 5)

- [ ] **SECURITY-014**: Penetration testing
- [ ] **SECURITY-015**: Security review with Team Charlie
- [ ] **SECURITY-016**: Documentation update
- [ ] **SECURITY-017**: Security training for developers

---

## 12. Security Checklist

### Pre-Deployment Checklist

#### Authentication
- [ ] JWT tokens use strong signing algorithm (RS256 or ES256)
- [ ] Access tokens expire in 15 minutes or less
- [ ] Refresh tokens rotate on each use
- [ ] Tokens are stored in httpOnly, Secure, SameSite cookies
- [ ] Login endpoint has rate limiting
- [ ] Failed login attempts are logged
- [ ] Password requirements enforced (min 12 chars, complexity)

#### Authorization
- [ ] All admin endpoints require authentication
- [ ] RBAC checks enforced on every endpoint
- [ ] Permission checks logged for audit
- [ ] Users cannot access other workspaces' data
- [ ] API keys cannot be read after creation (write-only)

#### XSS Protection
- [ ] CSP header implemented with strict policy
- [ ] `X-Content-Type-Options: nosniff` header set
- [ ] Output encoding for dynamic content
- [ ] DOMPurify used for any HTML rendering
- [ ] `eval()` and `innerHTML` avoided

#### CSRF Protection
- [ ] CSRF tokens required for state-changing operations
- [ ] SameSite=Lax on session cookies
- [ ] Origin/Referer validation on sensitive endpoints
- [ ] CSRF tokens rotated on privilege change

#### Session Security
- [ ] Sessions expire after inactivity (30 minutes)
- [ ] Sessions bound to IP/User-Agent fingerprint
- [ ] Concurrent session limits enforced
- [ ] Proper logout invalidates session server-side
- [ ] Session store secure (Redis with AUTH)

#### Transport Security
- [ ] TLS 1.3 required (minimum 1.2)
- [ ] HSTS header with max-age >= 1 year
- [ ] Certificate valid and not expiring soon
- [ ] Weak ciphers disabled

#### Audit & Monitoring
- [ ] All authentication events logged
- [ ] All authorization failures logged
- [ ] Sensitive data access logged
- [ ] Logs do not contain passwords or keys
- [ ] Audit logs retained for 90 days minimum

#### API Security
- [ ] Rate limiting enabled on all endpoints
- [ ] Request size limits enforced
- [ ] API versioned (/v0/)
- [ ] No sensitive data in URL parameters
- [ ] Proper error messages (no stack traces to client)

#### Dependency Security
- [ ] `go mod` dependencies scanned for vulnerabilities
- [ ] npm packages audited
- [ ] No known CVEs in dependencies
- [ ] Renovate/Dependabot configured

### Security Testing Checklist

- [ ] Unit tests for authentication middleware
- [ ] Unit tests for RBAC enforcement
- [ ] Unit tests for CSRF validation
- [ ] Integration tests for login/logout flows
- [ ] Penetration test by external security team
- [ ] OWASP ZAP scan passing
- [ ] Security headers validated (securityheaders.com)
- [ ] SSL Labs scan grade A+

---

## Appendix A: File Changes Required

### New Files

```
/internal/auth/
├── handlers.go      # Login, logout, refresh endpoints
├── middleware.go    # JWT validation, RBAC
├── tokens.go         # JWT generation/validation
├── csrf.go           # CSRF token management
├── rbac.go           # Permission checking
└── session.go        # Session management

/internal/audit/
├── logger.go         # Audit event logging
├── events.go         # Event type definitions
└── masking.go        # Sensitive data masking
```

### Modified Files

```
/cmd/rad-gateway/main.go
- Remove exemption for /v0/management/* from auth
- Add security middleware stack

/internal/admin/handlers.go
- Add permission checks to each handler
- Remove sensitive data from responses

/internal/config/config.go
- Remove API keys from Snapshot()
```

---

## Appendix B: Security Response Contacts

| Role | Contact | Responsibility |
|------|---------|---------------|
| Security Lead | security@radgateway.io | Security reviews, incident response |
| DevOps Lead | devops@radgateway.io | Infrastructure security |
| Team Charlie Lead | team-charlie@radgateway.io | Security hardening |

---

**Document Status:** APPROVED for implementation
**Reviewed By:** Security Engineer (Agent 4)
**Next Review Date:** 2026-03-17
**Approval:** Team Charlie Security Review Required Before Deployment

---

*This document contains security-sensitive information. Distribution is restricted to authorized personnel only.*
