// Package middleware provides rate limiting middleware.
package middleware

import (
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"log/slog"

	"radgateway/internal/logger"
)

// RateLimitConfig configures rate limiting behavior.
type RateLimitConfig struct {
	// Requests per window for authenticated users
	AuthenticatedRate int

	// Requests per window for unauthenticated users
	UnauthenticatedRate int

	// Time window for rate limiting
	Window time.Duration

	// Burst size for token bucket (allows short bursts)
	BurstSize int

	// SkipSuccessfulAuth allows unlimited requests for authenticated users
	SkipSuccessfulAuth bool

	// ExcludedPaths are paths that bypass rate limiting
	ExcludedPaths []string

	// ExcludedPrefixes are path prefixes that bypass rate limiting
	ExcludedPrefixes []string

	// CustomKeyFunc allows custom rate limit key generation
	CustomKeyFunc func(r *http.Request) string

	// OnLimitExceeded is called when rate limit is exceeded
	OnLimitExceeded func(w http.ResponseWriter, r *http.Request, limit, remaining int, reset time.Time)

	// EnableLogging enables rate limit logging
	EnableLogging bool
}

// DefaultRateLimitConfig returns a default rate limit configuration.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		AuthenticatedRate:   1000,
		UnauthenticatedRate: 100,
		Window:              time.Minute,
		BurstSize:           10,
		SkipSuccessfulAuth:  false,
		ExcludedPaths:       []string{"/health", "/metrics"},
		ExcludedPrefixes:    []string{"/static/", "/assets/"},
		EnableLogging:       true,
		OnLimitExceeded:     DefaultRateLimitExceededHandler,
	}
}

// StrictRateLimitConfig returns a strict rate limit configuration.
func StrictRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		AuthenticatedRate:   100,
		UnauthenticatedRate: 20,
		Window:              time.Minute,
		BurstSize:           5,
		SkipSuccessfulAuth:  false,
		ExcludedPaths:       []string{"/health"},
		ExcludedPrefixes:    []string{},
		EnableLogging:       true,
		OnLimitExceeded:     DefaultRateLimitExceededHandler,
	}
}

// AuthEndpointRateLimitConfig returns configuration for auth endpoints.
func AuthEndpointRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		AuthenticatedRate:   50,
		UnauthenticatedRate: 5,
		Window:              time.Minute,
		BurstSize:           3,
		SkipSuccessfulAuth:  false,
		ExcludedPaths:       []string{},
		ExcludedPrefixes:    []string{},
		EnableLogging:       true,
		OnLimitExceeded:     AuthRateLimitExceededHandler,
	}
}

// bucket represents a token bucket for rate limiting.
type bucket struct {
	tokens    float64
	lastCheck time.Time
	mu        sync.Mutex
}

// RateLimiter implements token bucket rate limiting.
type RateLimiter struct {
	config    RateLimitConfig
	buckets   map[string]*bucket
	mu        sync.RWMutex
	log       *slog.Logger
	stopCh    chan struct{}
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config:  config,
		buckets: make(map[string]*bucket),
		log:     logger.WithComponent("ratelimit"),
		stopCh:  make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Stop stops the rate limiter cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

// Handler wraps an http.Handler with rate limiting.
func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path is excluded
		if rl.isExcluded(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Get rate limit key
		key := rl.getKey(r)

		// Get rate for this request
		rate := rl.getRate(r)

		// Check rate limit
		allowed, remaining, reset := rl.allow(key, rate)

		// Set rate limit headers
		rl.setHeaders(w, rate, remaining, reset)

		if !allowed {
			if rl.config.EnableLogging {
				rl.log.Warn("rate limit exceeded",
					"key", key,
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr,
				)
			}

			if rl.config.OnLimitExceeded != nil {
				rl.config.OnLimitExceeded(w, r, rate, remaining, reset)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

// allow checks if a request is allowed under the rate limit.
func (rl *RateLimiter) allow(key string, rate int) (bool, int, time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	reset := now.Add(rl.config.Window)

	b, exists := rl.buckets[key]
	if !exists {
		b = &bucket{
			tokens:    float64(rl.config.BurstSize),
			lastCheck: now,
		}
		rl.buckets[key] = b
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Calculate tokens to add based on time elapsed
	elapsed := now.Sub(b.lastCheck)
	tokensToAdd := elapsed.Seconds() * float64(rate) / rl.config.Window.Seconds()
	b.tokens = math.Min(b.tokens+tokensToAdd, float64(rl.config.BurstSize))
	b.lastCheck = now

	// Check if request is allowed
	if b.tokens >= 1 {
		b.tokens--
		remaining := int(b.tokens)
		return true, remaining, reset
	}

	return false, 0, reset
}

// getKey generates a rate limit key from the request.
func (rl *RateLimiter) getKey(r *http.Request) string {
	// Use custom key function if provided
	if rl.config.CustomKeyFunc != nil {
		return rl.config.CustomKeyFunc(r)
	}

	// Try to get user ID from context (authenticated)
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return fmt.Sprintf("user:%s", userID)
	}

	// Try API key
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		// Hash the API key for the key
		return fmt.Sprintf("apikey:%s", hashString(apiKey))
	}

	// Fall back to IP address
	ip := rl.getClientIP(r)
	return fmt.Sprintf("ip:%s", ip)
}

// getClientIP extracts the client IP from the request.
func (rl *RateLimiter) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Get the first IP in the chain (closest to client)
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Check X-Real-IP header
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

// getRate returns the rate limit for the request.
func (rl *RateLimiter) getRate(r *http.Request) int {
	// Check if request is authenticated
	isAuthenticated := r.Header.Get("Authorization") != "" || r.Header.Get("X-API-Key") != ""

	if isAuthenticated {
		return rl.config.AuthenticatedRate
	}
	return rl.config.UnauthenticatedRate
}

// isExcluded checks if the request path is excluded from rate limiting.
func (rl *RateLimiter) isExcluded(r *http.Request) bool {
	path := r.URL.Path

	// Check exact paths
	for _, excluded := range rl.config.ExcludedPaths {
		if path == excluded {
			return true
		}
	}

	// Check prefixes
	for _, prefix := range rl.config.ExcludedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// setHeaders sets rate limit headers on the response.
func (rl *RateLimiter) setHeaders(w http.ResponseWriter, limit, remaining int, reset time.Time) {
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
}

// cleanup periodically removes old buckets.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanupBuckets()
		case <-rl.stopCh:
			return
		}
	}
}

// cleanupBuckets removes buckets that haven't been used recently.
func (rl *RateLimiter) cleanupBuckets() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-2 * rl.config.Window)
	for key, b := range rl.buckets {
		b.mu.Lock()
		lastCheck := b.lastCheck
		b.mu.Unlock()

		if lastCheck.Before(cutoff) {
			delete(rl.buckets, key)
		}
	}
}

// DefaultRateLimitExceededHandler handles rate limit exceeded responses.
func DefaultRateLimitExceededHandler(w http.ResponseWriter, r *http.Request, limit, remaining int, reset time.Time) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.FormatInt(reset.Unix()-time.Now().Unix(), 10))
	w.WriteHeader(http.StatusTooManyRequests)
	fmt.Fprintf(w, `{"error":{"message":"rate limit exceeded","code":429,"retry_after":%d}}`, reset.Unix()-time.Now().Unix())
}

// AuthRateLimitExceededHandler handles rate limit exceeded for auth endpoints.
func AuthRateLimitExceededHandler(w http.ResponseWriter, r *http.Request, limit, remaining int, reset time.Time) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.FormatInt(reset.Unix()-time.Now().Unix(), 10))
	w.WriteHeader(http.StatusTooManyRequests)
	fmt.Fprintf(w, `{"error":{"message":"too many authentication attempts, please try again later","code":429,"retry_after":%d}}`, reset.Unix()-time.Now().Unix())
}

// WithRateLimit is a convenience function for default rate limiting.
func WithRateLimit(next http.Handler) http.Handler {
	return NewRateLimiter(DefaultRateLimitConfig()).Handler(next)
}

// hashString creates a simple hash of a string for use as a key.
func hashString(s string) string {
	// Simple hash function - in production use a proper hash
	h := 0
	for i := 0; i < len(s); i++ {
		h = 31*h + int(s[i])
	}
	return strconv.Itoa(h)
}

// PathBasedRateLimiter provides different rate limits for different paths.
type PathBasedRateLimiter struct {
	defaultLimiter *RateLimiter
	pathLimiters map[string]*RateLimiter
	mu           sync.RWMutex
}

// NewPathBasedRateLimiter creates a new path-based rate limiter.
func NewPathBasedRateLimiter(defaultConfig RateLimitConfig) *PathBasedRateLimiter {
	return &PathBasedRateLimiter{
		defaultLimiter: NewRateLimiter(defaultConfig),
		pathLimiters:   make(map[string]*RateLimiter),
	}
}

// AddPathLimit adds a specific rate limiter for a path prefix.
func (p *PathBasedRateLimiter) AddPathLimit(prefix string, config RateLimitConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pathLimiters[prefix] = NewRateLimiter(config)
}

// Handler wraps an http.Handler with path-based rate limiting.
func (p *PathBasedRateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Find matching path limiter
		p.mu.RLock()
		var limiter *RateLimiter
		for prefix, l := range p.pathLimiters {
			if strings.HasPrefix(path, prefix) {
				limiter = l
				break
			}
		}
		p.mu.RUnlock()

		if limiter != nil {
			limiter.Handler(next).ServeHTTP(w, r)
		} else {
			p.defaultLimiter.Handler(next).ServeHTTP(w, r)
		}
	})
}

// Stop stops all rate limiters.
func (p *PathBasedRateLimiter) Stop() {
	p.defaultLimiter.Stop()
	p.mu.RLock()
	limiters := make([]*RateLimiter, 0, len(p.pathLimiters))
	for _, l := range p.pathLimiters {
		limiters = append(limiters, l)
	}
	p.mu.RUnlock()

	for _, l := range limiters {
		l.Stop()
	}
}
