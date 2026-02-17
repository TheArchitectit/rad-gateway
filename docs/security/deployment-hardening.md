# Deployment Security Hardening Guide

This guide provides step-by-step instructions for hardening RAD Gateway deployments.

## Overview

Security hardening involves implementing security measures at multiple layers:
1. **Application Layer**: Code-level security features
2. **Container Layer**: Docker/Podman security configurations
3. **Host Layer**: Operating system security
4. **Network Layer**: Firewall and network policies

## Application Security Hardening

### 1. CORS Configuration

**File**: `internal/middleware/cors_production.go`

Replace development CORS with production configuration:

```go
// In your main.go or router setup
allowedOrigins := []string{
    "https://app.radgateway.io",
    "https://admin.radgateway.io",
    // Add your specific domains
}

corsMiddleware := middleware.NewProductionCORS(allowedOrigins)
router.Use(corsMiddleware.Handler)
```

**Key Points**:
- Never use `*` (wildcard) origins with credentials
- Explicitly list all allowed origins
- Validate origins for HTTPS-only (except localhost for dev)
- Set `MaxAge` to 1 hour (3600 seconds) maximum

### 2. Security Headers

**File**: `internal/middleware/security.go`

Apply security headers to all routes:

```go
// Default security headers
securityMiddleware := middleware.NewSecurityHeaders(middleware.DefaultSecurityConfig())
router.Use(securityMiddleware.Handler)

// For API-only endpoints
apiSecurity := middleware.NewSecurityHeaders(middleware.APISecurityConfig())
apiRouter.Use(apiSecurity.Handler)

// For strict security (admin endpoints)
strictSecurity := middleware.NewSecurityHeaders(middleware.StrictSecurityConfig())
adminRouter.Use(strictSecurity.Handler)
```

**Header Reference**:

| Header | Value | Purpose |
|--------|-------|---------|
| Strict-Transport-Security | `max-age=31536000; includeSubDomains; preload` | Enforce HTTPS |
| Content-Security-Policy | `default-src 'self'; ...` | XSS protection |
| X-Frame-Options | `DENY` | Clickjacking protection |
| X-Content-Type-Options | `nosniff` | MIME sniffing protection |
| X-XSS-Protection | `1; mode=block` | Legacy XSS protection |
| Referrer-Policy | `strict-origin-when-cross-origin` | Privacy protection |
| Permissions-Policy | `camera=(), microphone=(), ...` | Feature restrictions |

### 3. Rate Limiting

**File**: `internal/middleware/ratelimit.go`

Configure different rate limits for different endpoints:

```go
// Default rate limiting
rateLimiter := middleware.NewRateLimiter(middleware.DefaultRateLimitConfig())
defer rateLimiter.Stop()

// Path-based rate limiting
pathLimiter := middleware.NewPathBasedRateLimiter(middleware.DefaultRateLimitConfig())

// Stricter limits for auth endpoints
authConfig := middleware.AuthEndpointRateLimitConfig()
pathLimiter.AddPathLimit("/auth/", authConfig)

// Apply to router
router.Use(pathLimiter.Handler)
```

**Rate Limit Configuration**:

| Endpoint Type | Rate | Window | Burst |
|---------------|------|--------|-------|
| Auth (login/register) | 5/min | 60s | 3 |
| General API | 1000/min | 60s | 10 |
| Strict API | 100/min | 60s | 5 |
| Health checks | Unlimited | - | - |

### 4. Request Validation

**Additional Security Middleware**:

```go
// Request size limiting (10MB max)
router.Use(middleware.RequestSizeLimiter(10 * 1024 * 1024))

// Host validation
allowedHosts := []string{
    "api.radgateway.io",
    "admin.radgateway.io",
}
router.Use(middleware.HostValidator(allowedHosts))

// Additional security checks
additionalSecurity := middleware.NewAdditionalSecurity(
    10*1024*1024, // 10MB max size
    allowedHosts,
)
router.Use(additionalSecurity.Handler)
```

## Container Security Hardening

### Dockerfile Hardening

**Current**: `/mnt/ollama/git/RADAPI01/Dockerfile`

Recommended production Dockerfile:

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

# Install security updates
RUN apk update && apk add --no-cache ca-certificates git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags)" \
    -o rad-gateway \
    ./cmd/rad-gateway

# Runtime stage - minimal image
FROM gcr.io/distroless/static:nonroot

# Copy CA certificates for TLS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /app/rad-gateway /rad-gateway

# Use non-root user
USER nonroot:nonroot

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/rad-gateway", "health"]

ENTRYPOINT ["/rad-gateway"]
```

### Container Security Best Practices

1. **Use Distroless Images**
   - Minimal attack surface
   - No shell, package manager, or utilities
   - Only application and certificates

2. **Run as Non-Root**
   - Never run as root (UID 0)
   - Use dedicated user with minimal permissions
   - Filesystem should be read-only

3. **Read-Only Root Filesystem**
   ```yaml
   securityContext:
     readOnlyRootFilesystem: true
     runAsNonRoot: true
     runAsUser: 65534
     allowPrivilegeEscalation: false
   ```

4. **Drop All Capabilities**
   ```yaml
   securityContext:
     capabilities:
       drop:
         - ALL
   ```

5. **Resource Limits**
   ```yaml
   resources:
     limits:
       memory: "512Mi"
       cpu: "1000m"
     requests:
       memory: "256Mi"
       cpu: "100m"
   ```

## Podman Security Configuration

### Pod Security

Create a systemd unit file for secure deployment:

```ini
# /etc/systemd/system/radgateway.service
[Unit]
Description=RAD Gateway Container
After=network.target

[Service]
Type=simple
Restart=always
RestartSec=5

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/radgateway
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictRealtime=true
RestrictSUIDSGID=true
LockPersonality=true
MemoryDenyWriteExecute=true

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

# Container execution
ExecStartPre=-/usr/bin/podman rm -f radgateway
ExecStart=/usr/bin/podman run \
    --name radgateway \
    --rm \
    --user 65534:65534 \
    --read-only \
    --tmpfs /tmp:noexec,nosuid,size=100m \
    --tmpfs /var/tmp:noexec,nosuid,size=100m \
    --cap-drop ALL \
    --security-opt no-new-privileges \
    --network radgateway-network \
    --publish 127.0.0.1:8080:8080 \
    --env-file /etc/radgateway/env \
    --volume /etc/radgateway/certs:/certs:ro \
    --health-cmd="wget -q --spider http://localhost:8080/health || exit 1" \
    --health-interval=30s \
    --health-timeout=3s \
    --health-retries=3 \
    radgateway:latest

ExecStop=/usr/bin/podman stop -t 30 radgateway
ExecStopPost=/usr/bin/podman rm -f radgateway

[Install]
WantedBy=multi-user.target
```

### Network Security

Create an isolated Podman network:

```bash
# Create isolated network
sudo podman network create \
    --driver bridge \
    --subnet 10.88.10.0/24 \
    --gateway 10.88.10.1 \
    radgateway-network

# Verify network isolation
sudo podman network inspect radgateway-network
```

## Host Security Hardening

### Firewall Configuration

**Using UFW (Ubuntu)**:

```bash
# Default deny
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH (rate limited)
sudo ufw limit ssh

# Allow HTTPS only
sudo ufw allow 443/tcp

# Allow from specific IPs for management
sudo ufw allow from 10.0.0.0/8 to any port 22

# Enable firewall
sudo ufw enable

# Check status
sudo ufw status verbose
```

**Using iptables**:

```bash
#!/bin/bash
# /etc/iptables/rules.v4

*filter
:INPUT DROP [0:0]
:FORWARD DROP [0:0]
:OUTPUT ACCEPT [0:0]

# Allow loopback
-A INPUT -i lo -j ACCEPT

# Allow established connections
-A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT

# Allow SSH (rate limited)
-A INPUT -p tcp --dport 22 -m conntrack --ctstate NEW -m recent --set
-A INPUT -p tcp --dport 22 -m conntrack --ctstate NEW -m recent --update --seconds 60 --hitcount 4 -j DROP
-A INPUT -p tcp --dport 22 -j ACCEPT

# Allow HTTPS
-A INPUT -p tcp --dport 443 -j ACCEPT

# Allow ICMP (ping)
-A INPUT -p icmp --icmp-type echo-request -j ACCEPT

# Log dropped packets
-A INPUT -m limit --limit 5/min -j LOG --log-prefix "iptables denied: " --log-level 7

COMMIT
```

### System Hardening

1. **Update System**
   ```bash
   sudo apt update && sudo apt upgrade -y
   sudo apt install -y unattended-upgrades
   ```

2. **Configure Automatic Updates**
   ```bash
   sudo dpkg-reconfigure -plow unattended-upgrades
   ```

3. **Disable Unnecessary Services**
   ```bash
   sudo systemctl disable --now cups
   sudo systemctl disable --now avahi-daemon
   ```

4. **Configure Fail2ban**
   ```bash
   sudo apt install fail2ban
   sudo tee /etc/fail2ban/jail.local <<EOF
   [DEFAULT]
   bantime = 3600
   findtime = 600
   maxretry = 3

   [sshd]
   enabled = true
   port = ssh
   filter = sshd
   logpath = /var/log/auth.log
   EOF
   sudo systemctl restart fail2ban
   ```

5. **Kernel Parameters**
   ```bash
   sudo tee /etc/sysctl.d/99-security.conf <<EOF
   # IP Spoofing protection
   net.ipv4.conf.all.rp_filter = 1
   net.ipv4.conf.default.rp_filter = 1

   # Ignore ICMP redirects
   net.ipv4.conf.all.accept_redirects = 0
   net.ipv4.conf.default.accept_redirects = 0

   # Ignore source routed packets
   net.ipv4.conf.all.accept_source_route = 0

   # Log suspicious packets
   net.ipv4.conf.all.log_martians = 1

   # Disable IPv6 if not needed
   net.ipv6.conf.all.disable_ipv6 = 1

   # Increase connection tracking
   net.netfilter.nf_conntrack_max = 2000000
   EOF
   sudo sysctl -p /etc/sysctl.d/99-security.conf
   ```

## Reverse Proxy Security (Nginx)

### Secure Nginx Configuration

```nginx
# /etc/nginx/sites-available/radgateway

upstream radgateway {
    server 127.0.0.1:8080;
    keepalive 32;
}

server {
    listen 443 ssl http2;
    server_name api.radgateway.io;

    # SSL Configuration
    ssl_certificate /etc/ssl/certs/radgateway.crt;
    ssl_certificate_key /etc/ssl/private/radgateway.key;
    ssl_session_timeout 1d;
    ssl_session_cache shared:SSL:50m;
    ssl_session_tickets off;

    # Modern TLS configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;

    # HSTS
    add_header Strict-Transport-Security "max-age=63072000" always;

    # Security headers
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Content-Security-Policy "default-src 'none'; frame-ancestors 'none';" always;

    # Rate limiting zones
    limit_req_zone $binary_remote_addr zone=api:10m rate=100r/m;
    limit_req_zone $binary_remote_addr zone=auth:10m rate=10r/m;

    # Logging
    access_log /var/log/nginx/radgateway-access.log;
    error_log /var/log/nginx/radgateway-error.log;

    # Client settings
    client_max_body_size 10m;
    client_body_buffer_size 128k;

    # Timeouts
    proxy_connect_timeout 60s;
    proxy_send_timeout 60s;
    proxy_read_timeout 60s;

    # Proxy headers
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host $host;
    proxy_set_header X-Forwarded-Port $server_port;

    # Hide upstream server info
    proxy_hide_header X-Powered-By;
    proxy_hide_header Server;

    # Health check endpoint
    location /health {
        proxy_pass http://radgateway;
        proxy_set_header Host $host;
        access_log off;
    }

    # Auth endpoints - stricter rate limiting
    location /auth/ {
        limit_req zone=auth burst=5 nodelay;
        proxy_pass http://radgateway;
    }

    # API endpoints
    location / {
        limit_req zone=api burst=20 nodelay;
        proxy_pass http://radgateway;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name api.radgateway.io;
    return 301 https://$server_name$request_uri;
}
```

### Certbot SSL Certificate

```bash
# Install certbot
sudo apt install certbot python3-certbot-nginx

# Obtain certificate
sudo certbot --nginx -d api.radgateway.io -d admin.radgateway.io

# Auto-renewal test
sudo certbot renew --dry-run

# Setup auto-renewal cron
sudo tee /etc/cron.d/certbot <<EOF
0 */12 * * * root certbot -q renew
EOF
```

## Monitoring & Alerting

### Security Event Monitoring

Configure alerts for:

1. **Failed Authentication Attempts**
   ```bash
   # Check logs
   sudo grep "authentication failed" /var/log/radgateway/app.log

   # Alert on more than 10 failures from same IP
   ```

2. **Rate Limit Violations**
   ```bash
   # Check for 429 responses
   sudo grep "429" /var/log/nginx/radgateway-access.log
   ```

3. **Unusual Error Rates**
   ```bash
   # Monitor 5xx errors
   sudo grep "5[0-9][0-9]" /var/log/nginx/radgateway-access.log | wc -l
   ```

4. **Certificate Expiry**
   ```bash
   # Check certificate expiry
   echo | openssl s_client -servername api.radgateway.io -connect api.radgateway.io:443 2>/dev/null | openssl x509 -noout -dates
   ```

### Security Metrics

Track these metrics:

| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| Failed auth rate | < 1% | > 5% |
| Rate limit hits | < 0.1% | > 1% |
| 5xx error rate | < 0.1% | > 1% |
| Avg response time | < 200ms | > 500ms |
| Certificate days | > 30 | < 7 |

## Deployment Checklist

Before deploying to production:

- [ ] All security middleware enabled
- [ ] TLS 1.2+ enforced
- [ ] Strong cipher suites configured
- [ ] Security headers verified
- [ ] Rate limiting tested
- [ ] Container running as non-root
- [ ] Read-only filesystem enabled
- [ ] Capabilities dropped
- [ ] Resource limits set
- [ ] Firewall rules active
- [ ] Fail2ban configured
- [ ] Logs shipping to SIEM
- [ ] Monitoring alerts configured
- [ ] Certificate auto-renewal tested
- [ ] Incident response plan reviewed
- [ ] Security team notified

## Post-Deployment Verification

Run these commands to verify security:

```bash
# Test security headers
curl -s -D - https://api.radgateway.io/health | grep -E "(Strict-Transport-Security|Content-Security-Policy|X-Frame-Options)"

# Test TLS version
openssl s_client -connect api.radgateway.io:443 -tls1_2 </dev/null 2>&1 | grep "Protocol"

# Test certificate
openssl s_client -connect api.radgateway.io:443 -servername api.radgateway.io </dev/null 2>/dev/null | openssl x509 -noout -text | grep -E "(Subject:|Issuer:|Not After)"

# Verify container security
sudo podman inspect radgateway | jq '.[0].Config.User, .[0].HostConfig.ReadonlyRootfs, .[0].HostConfig.CapDrop'

# Check firewall
sudo ufw status verbose

# Test rate limiting
for i in {1..110}; do curl -s -o /dev/null -w "%{http_code}\n" https://api.radgateway.io/health; done | sort | uniq -c
```

---

**Last Updated**: 2026-02-17
**Owner**: Team Charlie (Security Hardening)
**Review Schedule**: Monthly
