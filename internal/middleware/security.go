// Package middleware provides security headers middleware.
package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// SecurityHeadersConfig configures security headers middleware behavior.
type SecurityHeadersConfig struct {
	// ContentSecurityPolicy defines the Content-Security-Policy header value.
	ContentSecurityPolicy string

	// StrictTransportSecurity defines the Strict-Transport-Security header value.
	StrictTransportSecurity string

	// FrameOptions defines the X-Frame-Options header value.
	FrameOptions string

	// ContentTypeOptions defines the X-Content-Type-Options header value.
	ContentTypeOptions string

	// XSSProtection defines the X-XSS-Protection header value.
	XSSProtection string

	// ReferrerPolicy defines the Referrer-Policy header value.
	ReferrerPolicy string

	// PermissionsPolicy defines the Permissions-Policy header value.
	PermissionsPolicy string

	// CacheControl defines cache control headers for sensitive routes.
	CacheControl string

	// EnableHSTS enables HTTP Strict Transport Security.
	EnableHSTS bool

	// HSTSMaxAge sets the max-age for HSTS in seconds.
	HSTSMaxAge int

	// HSTSIncludeSubdomains includes subdomains in HSTS.
	HSTSIncludeSubdomains bool

	// HSTSPreload enables HSTS preload.
	HSTSPreload bool
}

// DefaultSecurityConfig returns a recommended security headers configuration.
func DefaultSecurityConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self' https:; frame-ancestors 'none'; base-uri 'self'; form-action 'self';",
		FrameOptions:          "DENY",
		ContentTypeOptions:    "nosniff",
		XSSProtection:         "1; mode=block",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		PermissionsPolicy:     "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=(), interest-cohort=()",
		CacheControl:          "no-store, no-cache, must-revalidate, proxy-revalidate",
		EnableHSTS:            true,
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubdomains: true,
		HSTSPreload:           true,
	}
}

// StrictSecurityConfig returns a strict security headers configuration.
func StrictSecurityConfig() SecurityHeadersConfig {
	config := DefaultSecurityConfig()
	// Stricter CSP that disallows inline styles
	config.ContentSecurityPolicy = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self';"
	config.ReferrerPolicy = "no-referrer"
	return config
}

// APISecurityConfig returns a security configuration optimized for API endpoints.
func APISecurityConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		ContentSecurityPolicy:   "default-src 'none'; frame-ancestors 'none';",
		FrameOptions:            "DENY",
		ContentTypeOptions:      "nosniff",
		XSSProtection:           "0", // Disabled for APIs (not rendered)
		ReferrerPolicy:          "no-referrer",
		PermissionsPolicy:       "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()",
		CacheControl:            "no-store",
		EnableHSTS:              true,
		HSTSMaxAge:              31536000,
		HSTSIncludeSubdomains:   true,
		HSTSPreload:             true,
	}
}

// SecurityHeaders provides HTTP security headers middleware.
type SecurityHeaders struct {
	config SecurityHeadersConfig
}

// NewSecurityHeaders creates a new security headers middleware.
func NewSecurityHeaders(config SecurityHeadersConfig) *SecurityHeaders {
	return &SecurityHeaders{config: config}
}

// Handler wraps an http.Handler with security headers.
func (s *SecurityHeaders) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.setHeaders(w, r)
		next.ServeHTTP(w, r)
	})
}

// setHeaders sets all security headers on the response.
func (s *SecurityHeaders) setHeaders(w http.ResponseWriter, r *http.Request) {
	// Content Security Policy
	if s.config.ContentSecurityPolicy != "" {
		w.Header().Set("Content-Security-Policy", s.config.ContentSecurityPolicy)
	}

	// X-Frame-Options
	if s.config.FrameOptions != "" {
		w.Header().Set("X-Frame-Options", s.config.FrameOptions)
	}

	// X-Content-Type-Options
	if s.config.ContentTypeOptions != "" {
		w.Header().Set("X-Content-Type-Options", s.config.ContentTypeOptions)
	}

	// X-XSS-Protection
	if s.config.XSSProtection != "" {
		w.Header().Set("X-XSS-Protection", s.config.XSSProtection)
	}

	// Referrer-Policy
	if s.config.ReferrerPolicy != "" {
		w.Header().Set("Referrer-Policy", s.config.ReferrerPolicy)
	}

	// Permissions-Policy (formerly Feature-Policy)
	if s.config.PermissionsPolicy != "" {
		w.Header().Set("Permissions-Policy", s.config.PermissionsPolicy)
	}

	// Cache-Control for sensitive routes
	if s.shouldApplyCacheControl(r) && s.config.CacheControl != "" {
		w.Header().Set("Cache-Control", s.config.CacheControl)
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}

	// HTTP Strict Transport Security (HSTS)
	if s.config.EnableHSTS && s.isHTTPS(r) {
		hstsValue := fmt.Sprintf("max-age=%d", s.config.HSTSMaxAge)
		if s.config.HSTSIncludeSubdomains {
			hstsValue += "; includeSubDomains"
		}
		if s.config.HSTSPreload {
			hstsValue += "; preload"
		}
		w.Header().Set("Strict-Transport-Security", hstsValue)
	}
}

// shouldApplyCacheControl determines if cache control headers should be applied.
func (s *SecurityHeaders) shouldApplyCacheControl(r *http.Request) bool {
	// Apply to auth endpoints and API endpoints that shouldn't be cached
	path := strings.ToLower(r.URL.Path)
	sensitivePaths := []string{
		"/auth/",
		"/api/",
		"/admin/",
		"/internal/",
		"/v1/",
		"/v2/",
	}

	for _, prefix := range sensitivePaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// isHTTPS checks if the request is using HTTPS.
func (s *SecurityHeaders) isHTTPS(r *http.Request) bool {
	// Check X-Forwarded-Proto header (for proxies/load balancers)
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		return true
	}

	// Check the scheme directly
	if r.URL.Scheme == "https" {
		return true
	}

	// Check TLS connection state
	if r.TLS != nil {
		return true
	}

	return false
}

// WithSecurityHeaders is a convenience function for default security headers.
func WithSecurityHeaders(next http.Handler) http.Handler {
	return NewSecurityHeaders(DefaultSecurityConfig()).Handler(next)
}

// WithAPISecurityHeaders is a convenience function for API security headers.
func WithAPISecurityHeaders(next http.Handler) http.Handler {
	return NewSecurityHeaders(APISecurityConfig()).Handler(next)
}

// WithStrictSecurityHeaders is a convenience function for strict security headers.
func WithStrictSecurityHeaders(next http.Handler) http.Handler {
	return NewSecurityHeaders(StrictSecurityConfig()).Handler(next)
}

// AdditionalSecurity provides additional security middleware features.
type AdditionalSecurity struct {
	// MaxRequestSize is the maximum allowed request body size in bytes.
	MaxRequestSize int64

	// AllowedHosts is a list of allowed host headers.
	AllowedHosts []string

	// BlockCommonExploits enables blocking of common exploit patterns.
	BlockCommonExploits bool

	// RequireSecureHeaders requires certain security headers on requests.
	RequireSecureHeaders bool
}

// NewAdditionalSecurity creates additional security middleware.
func NewAdditionalSecurity(maxSize int64, allowedHosts []string) *AdditionalSecurity {
	return &AdditionalSecurity{
		MaxRequestSize:       maxSize,
		AllowedHosts:         allowedHosts,
		BlockCommonExploits:  true,
		RequireSecureHeaders: false,
	}
}

// Handler wraps an http.Handler with additional security checks.
func (a *AdditionalSecurity) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check allowed hosts
		if len(a.AllowedHosts) > 0 && !a.isAllowedHost(r.Host) {
			http.Error(w, `{"error":{"message":"invalid host","code":400}}`, http.StatusBadRequest)
			return
		}

		// Block common exploit patterns
		if a.BlockCommonExploits && a.containsExploitPatterns(r) {
			http.Error(w, `{"error":{"message":"suspicious request detected","code":400}}`, http.StatusBadRequest)
			return
		}

		// Apply request size limit
		if a.MaxRequestSize > 0 {
			r.Body = http.MaxBytesReader(w, r.Body, a.MaxRequestSize)
		}

		// Set request timeout header
		w.Header().Set("X-Response-Time", time.Now().Format(time.RFC3339))

		next.ServeHTTP(w, r)
	})
}

// isAllowedHost checks if the host is in the allowed list.
func (a *AdditionalSecurity) isAllowedHost(host string) bool {
	// Remove port if present
	if i := strings.LastIndex(host, ":"); i != -1 {
		host = host[:i]
	}

	for _, allowed := range a.AllowedHosts {
		// Exact match
		if strings.EqualFold(host, allowed) {
			return true
		}

		// Wildcard subdomain match
		if strings.HasPrefix(allowed, "*.") {
			suffix := allowed[1:] // Remove the leading *
			if strings.HasSuffix(strings.ToLower(host), strings.ToLower(suffix)) {
				return true
			}
		}
	}

	return false
}

// containsExploitPatterns checks for common exploit patterns in the request.
func (a *AdditionalSecurity) containsExploitPatterns(r *http.Request) bool {
	// Check URL path for suspicious patterns
	path := strings.ToLower(r.URL.Path)

	// Path traversal patterns
	traversalPatterns := []string{
		"../", "..\\", "%2e%2e/", "%2e%2e\\",
		".env", ".git/", ".svn/", ".hg/",
		"/etc/passwd", "/etc/shadow",
		"config.xml", "web.config",
	}

	for _, pattern := range traversalPatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	// Check query parameters for SQL injection patterns
	query := strings.ToLower(r.URL.RawQuery)
	sqlPatterns := []string{
		"' or '", "'='", "';", "--", "/*", "*/",
		"union select", "insert into", "delete from",
		"drop table", "exec(", "execute(",
	}

	for _, pattern := range sqlPatterns {
		if strings.Contains(query, pattern) {
			return true
		}
	}

	return false
}

// RequestSizeLimiter creates middleware to limit request body size.
func RequestSizeLimiter(maxSize int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxSize)
			next.ServeHTTP(w, r)
		})
	}
}

// HostValidator creates middleware to validate the Host header.
func HostValidator(allowedHosts []string) func(http.Handler) http.Handler {
	security := NewAdditionalSecurity(0, allowedHosts)
	return func(next http.Handler) http.Handler {
		return security.Handler(next)
	}
}
