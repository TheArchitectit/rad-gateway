// Package middleware provides brute force protection middleware.
package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"log/slog"

	"radgateway/internal/logger"
)

// BruteForceConfig configures brute force protection behavior.
type BruteForceConfig struct {
	// MaxAttempts is the maximum number of failed attempts before blocking
	MaxAttempts int

	// Window is the time window for counting attempts
	Window time.Duration

	// BlockDuration is how long to block after max attempts reached
	BlockDuration time.Duration

	// ExemptPaths are paths that bypass brute force protection
	ExemptPaths []string

	// SuccessReset resets counter on successful auth
	SuccessReset bool

	// EnableLogging enables brute force detection logging
	EnableLogging bool
}

// DefaultBruteForceConfig returns default brute force protection configuration.
func DefaultBruteForceConfig() BruteForceConfig {
	return BruteForceConfig{
		MaxAttempts:   5,
		Window:        5 * time.Minute,
		BlockDuration: 15 * time.Minute,
		ExemptPaths:   []string{"/health", "/metrics"},
		SuccessReset:  true,
		EnableLogging: true,
	}
}

// StrictBruteForceConfig returns strict brute force protection configuration.
func StrictBruteForceConfig() BruteForceConfig {
	return BruteForceConfig{
		MaxAttempts:   3,
		Window:        5 * time.Minute,
		BlockDuration: 30 * time.Minute,
		ExemptPaths:   []string{"/health"},
		SuccessReset:  true,
		EnableLogging: true,
	}
}

// attempt tracks failed authentication attempts
type attempt struct {
	count       int
	firstAttempt time.Time
	lastAttempt  time.Time
	blockedUntil *time.Time
}

// BruteForceProtector provides brute force attack protection
type BruteForceProtector struct {
	config   BruteForceConfig
	attempts map[string]*attempt
	mu       sync.RWMutex
	log      *slog.Logger
	stopCh   chan struct{}
}

// NewBruteForceProtector creates a new brute force protector
func NewBruteForceProtector(config BruteForceConfig) *BruteForceProtector {
	bf := &BruteForceProtector{
		config:   config,
		attempts: make(map[string]*attempt),
		log:      logger.WithComponent("bruteforce"),
		stopCh:   make(chan struct{}),
	}

	// Start cleanup goroutine
	go bf.cleanup()

	return bf
}

// Stop stops the cleanup goroutine
func (bf *BruteForceProtector) Stop() {
	close(bf.stopCh)
}

// Middleware returns HTTP middleware that tracks failed auth attempts
func (bf *BruteForceProtector) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip exempt paths
		if bf.isExempt(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Get client identifier
		key := bf.getKey(r)

		// Check if blocked
		if blocked, remaining := bf.isBlocked(key); blocked {
			if bf.config.EnableLogging {
				bf.log.Warn("blocked request from brute force protection",
					"key", key,
					"path", r.URL.Path,
					"remaining", remaining.Seconds(),
				)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(remaining.Seconds())))
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error":{"message":"too many failed attempts, please try again later","code":429,"retry_after":%d}}`, int(remaining.Seconds()))
			return
		}

		// Wrap response writer to detect failed auth
		wrapped := &bruteForceResponseWriter{
			ResponseWriter: w,
			bf:             bf,
			key:            key,
			r:              r,
		}

		next.ServeHTTP(wrapped, r)
	})
}

// RecordSuccess resets the attempt counter for a key (call on successful auth)
func (bf *BruteForceProtector) RecordSuccess(key string) {
	if !bf.config.SuccessReset {
		return
	}

	bf.mu.Lock()
	defer bf.mu.Unlock()

	delete(bf.attempts, key)
}

// RecordFailure records a failed authentication attempt
func (bf *BruteForceProtector) RecordFailure(key string) {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	now := time.Now()
	att, exists := bf.attempts[key]
	if !exists {
		att = &attempt{
			count:        1,
			firstAttempt: now,
			lastAttempt:  now,
		}
		bf.attempts[key] = att
	} else {
		// Check if window has expired
		if now.Sub(att.firstAttempt) > bf.config.Window {
			// Reset counter
			att.count = 1
			att.firstAttempt = now
			att.blockedUntil = nil
		} else {
			att.count++
		}
		att.lastAttempt = now
	}

	// Check if should block
	if att.count >= bf.config.MaxAttempts && att.blockedUntil == nil {
		blockedUntil := now.Add(bf.config.BlockDuration)
		att.blockedUntil = &blockedUntil

		if bf.config.EnableLogging {
			bf.log.Warn("brute force protection activated",
				"key", key,
				"attempts", att.count,
				"blocked_until", blockedUntil,
			)
		}
	}
}

// isBlocked checks if a key is currently blocked
func (bf *BruteForceProtector) isBlocked(key string) (bool, time.Duration) {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	att, exists := bf.attempts[key]
	if !exists || att.blockedUntil == nil {
		return false, 0
	}

	now := time.Now()
	if now.After(*att.blockedUntil) {
		return false, 0
	}

	return true, att.blockedUntil.Sub(now)
}

// getAttempts returns the current attempt count for a key
func (bf *BruteForceProtector) getAttempts(key string) int {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	att, exists := bf.attempts[key]
	if !exists {
		return 0
	}

	// Check if window expired
	if time.Since(att.firstAttempt) > bf.config.Window {
		return 0
	}

	return att.count
}

// getKey generates a key from the request (IP-based)
func (bf *BruteForceProtector) getKey(r *http.Request) string {
	// Try X-Forwarded-For first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Try X-Real-IP
	xri := r.Header.Get("X-Real-Ip")
	if xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// isExempt checks if the request path is exempt from brute force protection
func (bf *BruteForceProtector) isExempt(r *http.Request) bool {
	path := r.URL.Path
	for _, exempt := range bf.config.ExemptPaths {
		if path == exempt {
			return true
		}
	}
	return false
}

// cleanup periodically removes old attempt records
func (bf *BruteForceProtector) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bf.cleanupAttempts()
		case <-bf.stopCh:
			return
		}
	}
}

// cleanupAttempts removes expired attempt records
func (bf *BruteForceProtector) cleanupAttempts() {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-bf.config.Window - bf.config.BlockDuration)

	for key, att := range bf.attempts {
		if att.lastAttempt.Before(cutoff) {
			delete(bf.attempts, key)
		}
	}
}

// bruteForceResponseWriter wraps ResponseWriter to detect auth failures
type bruteForceResponseWriter struct {
	http.ResponseWriter
	bf  *BruteForceProtector
	key string
	r   *http.Request
}

// WriteHeader captures the status code and records failures
func (w *bruteForceResponseWriter) WriteHeader(code int) {
	// Record failure for 401 Unauthorized responses
	if code == http.StatusUnauthorized {
		w.bf.RecordFailure(w.key)
	} else if code >= 200 && code < 300 && w.bf.config.SuccessReset {
		// Record success for 2xx responses on auth endpoints
		if w.isAuthEndpoint() {
			w.bf.RecordSuccess(w.key)
		}
	}

	w.ResponseWriter.WriteHeader(code)
}

// Write captures the response and records failures
func (w *bruteForceResponseWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}

// isAuthEndpoint checks if the current request is to an auth endpoint
func (w *bruteForceResponseWriter) isAuthEndpoint() bool {
	path := w.r.URL.Path
	authPaths := []string{"/v1/auth/", "/oauth/", "/login", "/api/auth/"}
	for _, prefix := range authPaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// Helper to access config (needed for the wrapper)
var config = DefaultBruteForceConfig()

// WithBruteForceProtection is a convenience function for default brute force protection
func WithBruteForceProtection(next http.Handler) http.Handler {
	return NewBruteForceProtector(DefaultBruteForceConfig()).Middleware(next)
}
