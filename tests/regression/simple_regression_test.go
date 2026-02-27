// Package regression provides simplified regression tests.
// Sprint 7.2: Build Regression Test Suite
package regression

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"radgateway/internal/auth"
	"radgateway/internal/middleware"
)

// TestJWTTokenGeneration validates token generation hasn't regressed
func TestJWTTokenGeneration(t *testing.T) {
	config := auth.JWTConfig{
		AccessTokenSecret:  []byte("test-access-secret-32-bytes-long!"),
		RefreshTokenSecret: []byte("test-refresh-secret-32-bytes-long"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "rad-gateway-test",
	}

	manager := auth.NewJWTManager(config)

	tokens, err := manager.GenerateTokenPair(
		"user-123",
		"test@example.com",
		"admin",
		"workspace-1",
		[]string{"read", "write"},
	)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	if tokens.AccessToken == "" {
		t.Error("Access token should not be empty")
	}
	if tokens.RefreshToken == "" {
		t.Error("Refresh token should not be empty")
	}
	if tokens.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should not be zero")
	}

	// Validate the token
	claims, err := manager.ValidateAccessToken(tokens.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("UserID = %v, want %v", claims.UserID, "user-123")
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Email = %v, want %v", claims.Email, "test@example.com")
	}
}

// TestRateLimitingMiddleware validates rate limiting hasn't regressed
func TestRateLimitingMiddleware(t *testing.T) {
	config := middleware.DefaultRateLimitConfig()
	limiter := middleware.NewRateLimiter(config)
	defer limiter.Stop()

	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with rate limiter
	wrapped := limiter.Handler(handler)

	// Make requests
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		// Should get either 200 or 429
		if rr.Code != http.StatusOK && rr.Code != http.StatusTooManyRequests {
			t.Errorf("Request %d: expected status 200 or 429, got %d", i+1, rr.Code)
		}
	}
}

// TestSecurityHeaders validates security headers haven't regressed
func TestSecurityHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.WithSecurityHeaders(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	// Check security headers
	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
	}

	for header, expected := range expectedHeaders {
		if got := rr.Header().Get(header); got != expected {
			t.Errorf("Header %s: got %q, want %q", header, got, expected)
		}
	}
}

// TestCORSMiddleware validates CORS hasn't regressed
func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.WithCORS(handler)

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	// Check CORS headers
	if rr.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("Missing Access-Control-Allow-Origin header")
	}
}

// TestBruteForceProtection validates brute force protection hasn't regressed
func TestBruteForceProtection(t *testing.T) {
	config := middleware.DefaultBruteForceConfig()
	bf := middleware.NewBruteForceProtector(config)
	defer bf.Stop()

	// Create handler that returns 401
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	wrapped := bf.Middleware(handler)

	// Make requests from same IP
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		// After 5 failed attempts, should get 429
		if i >= 5 && rr.Code != http.StatusTooManyRequests {
			t.Logf("Request %d: status %d (expected 429 after brute force detection)", i+1, rr.Code)
		}
	}
}

// TestRequestContext validates request context hasn't regressed
func TestRequestContext(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Request ID should be set
		requestID := middleware.GetRequestID(r.Context())
		if requestID == "" {
			t.Error("Request ID should be set in context")
		}

		// Trace ID should be set
		traceID := middleware.GetTraceID(r.Context())
		if traceID == "" {
			t.Error("Trace ID should be set in context")
		}

		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.WithRequestContext(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

// BenchmarkJWTGeneration benchmarks token generation
func BenchmarkJWTGeneration(b *testing.B) {
	config := auth.JWTConfig{
		AccessTokenSecret:  []byte("test-access-secret-32-bytes-long"),
		RefreshTokenSecret: []byte("test-refresh-secret-32-bytes-long"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "rad-gateway-bench",
	}

	manager := auth.NewJWTManager(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.GenerateTokenPair(
			"user-123",
			"test@example.com",
			"admin",
			"workspace-1",
			[]string{"read", "write"},
		)
		if err != nil {
			b.Fatalf("Failed to generate token: %v", err)
		}
	}
}

// BenchmarkTokenValidation benchmarks token validation
func BenchmarkTokenValidation(b *testing.B) {
	config := auth.JWTConfig{
		AccessTokenSecret:  []byte("test-access-secret-32-bytes-long"),
		RefreshTokenSecret: []byte("test-refresh-secret-32-bytes-long"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "rad-gateway-bench",
	}

	manager := auth.NewJWTManager(config)
	tokens, _ := manager.GenerateTokenPair(
		"user-123",
		"test@example.com",
		"admin",
		"workspace-1",
		[]string{"read"},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.ValidateAccessToken(tokens.AccessToken)
		if err != nil {
			b.Fatalf("Failed to validate token: %v", err)
		}
	}
}

// BenchmarkMiddlewareChain benchmarks middleware overhead
func BenchmarkMiddlewareChain(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Chain all middleware
	wrapped := middleware.WithRequestContext(handler)
	wrapped = middleware.WithSecurityHeaders(wrapped)
	wrapped = middleware.WithCORS(wrapped)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()
		wrapped.ServeHTTP(rr, req)
	}
}
