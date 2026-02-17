# Production Security Checklist

This checklist covers all security requirements for deploying RAD Gateway in production environments.

## Pre-Deployment Security Checklist

### Infrastructure Security

- [ ] **TLS Configuration**
  - [ ] Valid SSL/TLS certificates installed
  - [ ] TLS 1.2+ enforced (TLS 1.0 and 1.1 disabled)
  - [ ] Strong cipher suites configured
  - [ ] HSTS enabled with appropriate max-age
  - [ ] Certificate auto-renewal configured

- [ ] **Network Security**
  - [ ] Firewall rules configured (deny all, allow specific)
  - [ ] Internal services not exposed publicly
  - [ ] Database access restricted to application servers
  - [ ] VPC/Network segmentation in place
  - [ ] DDoS protection enabled (Cloudflare/AWS Shield/etc)

- [ ] **Secrets Management**
  - [ ] All secrets stored in Infisical
  - [ ] No hardcoded secrets in source code
  - [ ] Secret rotation policy defined
  - [ ] Encryption at rest for secrets
  - [ ] Audit logging for secret access

### Application Security

- [ ] **Authentication**
  - [ ] JWT secrets are strong (minimum 256-bit)
  - [ ] Token expiry times configured (access: 15min, refresh: 7 days)
  - [ ] Refresh token rotation enabled
  - [ ] Secure cookie attributes (Secure, HttpOnly, SameSite)
  - [ ] Password policies enforced (min 12 chars, complexity)
  - [ ] Multi-factor authentication for admin accounts
  - [ ] Brute force protection enabled

- [ ] **Authorization**
  - [ ] RBAC properly configured
  - [ ] Principle of least privilege applied
  - [ ] API key permissions validated
  - [ ] Resource-level access controls implemented

- [ ] **CORS Configuration**
  - [ ] Wildcard origins disabled in production
  - [ ] Specific origins explicitly allowed
  - [ ] Credentials only sent to allowed origins
  - [ ] Origin validation implemented
  - [ ] Vary: Origin header set

- [ ] **Security Headers**
  - [ ] Content-Security-Policy configured
  - [ ] X-Frame-Options: DENY
  - [ ] X-Content-Type-Options: nosniff
  - [ ] X-XSS-Protection: 1; mode=block
  - [ ] Referrer-Policy: strict-origin-when-cross-origin
  - [ ] Permissions-Policy configured
  - [ ] Cache-Control for sensitive endpoints

- [ ] **Rate Limiting**
  - [ ] Rate limiting enabled for all endpoints
  - [ ] Different limits for auth vs API endpoints
  - [ ] IP-based limiting for unauthenticated requests
  - [ ] User-based limiting for authenticated requests
  - [ ] Rate limit headers exposed
  - [ ] Proper 429 responses configured

### Data Security

- [ ] **Encryption**
  - [ ] Data encrypted at rest (database, files)
  - [ ] Data encrypted in transit (TLS)
  - [ ] API keys hashed in database
  - [ ] Sensitive fields encrypted (PII, credentials)

- [ ] **Data Handling**
  - [ ] Input validation on all endpoints
  - [ ] Output encoding to prevent XSS
  - [ ] SQL injection prevention (parameterized queries)
  - [ ] Request size limits configured
  - [ ] File upload restrictions

### Logging & Monitoring

- [ ] **Audit Logging**
  - [ ] Authentication events logged
  - [ ] Authorization failures logged
  - [ ] Data access logged (CRUD operations)
  - [ ] Admin actions logged
  - [ ] Log integrity protection (signing/hashing)

- [ ] **Security Monitoring**
  - [ ] Failed login attempts monitored
  - [ ] Rate limit exceeded events alerted
  - [ ] Error rate anomalies detected
  - [ ] Security headers presence verified
  - [ ] TLS certificate expiry monitored

### Dependencies

- [ ] **Vulnerability Management**
  - [ ] All dependencies scanned (govulncheck)
  - [ ] No high/critical vulnerabilities
  - [ ] Dependency update process defined
  - [ ] SBOM (Software Bill of Materials) generated
  - [ ] License compliance verified

## Production Configuration

### Environment Variables

Required secure configuration:

```bash
# Required for production
ENV=production
LOG_LEVEL=info

# Security
JWT_ACCESS_SECRET=<64-char-random-hex>
JWT_REFRESH_SECRET=<64-char-random-hex>
COOKIE_SECURE=true
COOKIE_HTTPONLY=true
COOKIE_SAMESITE=strict

# CORS (comma-separated list, NO wildcards)
CORS_ALLOWED_ORIGINS=https://app.radgateway.io,https://admin.radgateway.io

# Rate Limiting
RATE_LIMIT_AUTH_RPM=5
RATE_LIMIT_API_RPM=1000
RATE_LIMIT_WINDOW=60

# Infisical
INFISICAL_MACHINE_CLIENT_ID=<from-infisical>
INFISICAL_MACHINE_CLIENT_SECRET=<from-infisical>
INFISICAL_PROJECT_ID=<project-id>
INFISICAL_ENVIRONMENT=prod
```

### Middleware Configuration

Recommended middleware stack order:

```go
// 1. Security headers (first to set headers on all responses)
router.Use(middleware.WithSecurityHeaders)

// 2. Request context (for tracing)
router.Use(middleware.WithRequestContext)

// 3. CORS (must be before auth)
router.Use(middleware.NewProductionCORS(allowedOrigins).Handler)

// 4. Rate limiting (before auth to protect auth endpoints)
router.Use(middleware.NewRateLimiter(middleware.DefaultRateLimitConfig()).Handler)

// 5. Request size limiting
router.Use(middleware.RequestSizeLimiter(10 * 1024 * 1024)) // 10MB

// 6. Authentication
router.Use(authenticator.Require)
```

## Deployment Verification

### Security Headers Verification

```bash
curl -I https://radgateway.io/health

# Verify all headers present:
# - Strict-Transport-Security
# - Content-Security-Policy
# - X-Frame-Options: DENY
# - X-Content-Type-Options: nosniff
# - X-XSS-Protection: 1; mode=block
# - Referrer-Policy: strict-origin-when-cross-origin
```

### TLS Configuration Verification

```bash
# Test TLS version
openssl s_client -connect radgateway.io:443 -tls1_2 </dev/null

# Should fail for older TLS:
openssl s_client -connect radgateway.io:443 -tls1 </dev/null
openssl s_client -connect radgateway.io:443 -ssl3 </dev/null

# Test cipher strength
nmap --script ssl-enum-ciphers -p 443 radgateway.io
```

### CORS Configuration Verification

```bash
# Should succeed for allowed origins
curl -H "Origin: https://app.radgateway.io" \
     -H "Access-Control-Request-Method: POST" \
     -X OPTIONS \
     -I https://api.radgateway.io/v1/chat

# Should fail for disallowed origins
curl -H "Origin: https://evil.com" \
     -X GET \
     -I https://api.radgateway.io/v1/chat
```

### Rate Limiting Verification

```bash
# Test rate limit headers
curl -I https://api.radgateway.io/health

# Verify headers:
# X-RateLimit-Limit: 1000
# X-RateLimit-Remaining: 999
# X-RateLimit-Reset: <timestamp>

# Test rate limiting (send 101 requests quickly)
for i in {1..101}; do
  curl -s -o /dev/null -w "%{http_code}\n" https://api.radgateway.io/health
done | sort | uniq -c
# Should see 429 responses after limit
```

## Incident Response

### Security Incident Checklist

- [ ] Isolate affected systems
- [ ] Preserve logs and evidence
- [ ] Notify security team
- [ ] Assess scope of breach
- [ ] Rotate compromised credentials
- [ ] Apply security patches
- [ ] Document incident timeline
- [ ] Post-incident review

### Emergency Contacts

- Security Team: security@radgateway.io
- On-call Engineer: oncall@radgateway.io
- Infrastructure: infra@radgateway.io

## Ongoing Security Tasks

### Weekly
- [ ] Review security logs for anomalies
- [ ] Check for failed authentication attempts
- [ ] Verify backup integrity

### Monthly
- [ ] Run vulnerability scans
- [ ] Review access logs
- [ ] Update dependencies
- [ ] Review and rotate secrets
- [ ] Security metrics review

### Quarterly
- [ ] Penetration testing
- [ ] Security policy review
- [ ] Disaster recovery testing
- [ ] Security training
- [ ] Third-party security audit

## Compliance Considerations

### Data Protection
- [ ] GDPR compliance (if applicable)
- [ ] Data retention policies
- [ ] Right to deletion implemented
- [ ] Data processing agreements

### Audit Requirements
- [ ] SOC 2 compliance (if applicable)
- [ ] ISO 27001 alignment
- [ ] Audit trail completeness
- [ ] Change management logs

---

**Last Updated**: 2026-02-17
**Owner**: Team Charlie (Security Hardening)
**Review Schedule**: Monthly
