package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDefaultBruteForceConfig(t *testing.T) {
	config := DefaultBruteForceConfig()

	if config.MaxAttempts != 5 {
		t.Errorf("MaxAttempts = %d, want 5", config.MaxAttempts)
	}

	if config.Window != 5*time.Minute {
		t.Errorf("Window = %v, want 5m", config.Window)
	}

	if config.BlockDuration != 15*time.Minute {
		t.Errorf("BlockDuration = %v, want 15m", config.BlockDuration)
	}

	if !config.SuccessReset {
		t.Error("SuccessReset should be true")
	}

	if !config.EnableLogging {
		t.Error("EnableLogging should be true")
	}
}

func TestStrictBruteForceConfig(t *testing.T) {
	config := StrictBruteForceConfig()

	if config.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", config.MaxAttempts)
	}

	if config.Window != 5*time.Minute {
		t.Errorf("Window = %v, want 5m", config.Window)
	}

	if config.BlockDuration != 30*time.Minute {
		t.Errorf("BlockDuration = %v, want 30m", config.BlockDuration)
	}
}

func TestBruteForceProtector_RecordFailure(t *testing.T) {
	config := BruteForceConfig{
		MaxAttempts:   3,
		Window:        time.Hour,
		BlockDuration: time.Hour,
		EnableLogging: false,
	}

	bf := NewBruteForceProtector(config)
	defer bf.Stop()

	key := "test-key"

	// Record failures up to max
	for i := 0; i < config.MaxAttempts-1; i++ {
		bf.RecordFailure(key)
		if blocked, _ := bf.isBlocked(key); blocked {
			t.Errorf("Should not be blocked after %d failures", i+1)
		}
	}

	// One more failure should trigger block
	bf.RecordFailure(key)
	blocked, remaining := bf.isBlocked(key)
	if !blocked {
		t.Error("Should be blocked after max attempts")
	}
	if remaining <= 0 {
		t.Error("Remaining time should be positive")
	}
}

func TestBruteForceProtector_RecordSuccess(t *testing.T) {
	config := BruteForceConfig{
		MaxAttempts:   3,
		Window:        time.Hour,
		BlockDuration: time.Hour,
		SuccessReset:  true,
		EnableLogging: false,
	}

	bf := NewBruteForceProtector(config)
	defer bf.Stop()

	key := "test-key"

	// Record some failures
	bf.RecordFailure(key)
	bf.RecordFailure(key)

	// Record success should reset
	bf.RecordSuccess(key)

	if attempts := bf.getAttempts(key); attempts != 0 {
		t.Errorf("Attempts = %d, want 0 after success", attempts)
	}
}

func TestBruteForceProtector_isExempt(t *testing.T) {
	config := BruteForceConfig{
		ExemptPaths: []string{"/health", "/metrics"},
	}

	bf := NewBruteForceProtector(config)
	defer bf.Stop()

	tests := []struct {
		path    string
		exempt  bool
	}{
		{"/health", true},
		{"/metrics", true},
		{"/api/auth", false},
		{"/health/custom", false}, // Only exact match
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		got := bf.isExempt(req)
		if got != tt.exempt {
			t.Errorf("isExempt(%q) = %v, want %v", tt.path, got, tt.exempt)
		}
	}
}

func TestBruteForceProtector_getKey(t *testing.T) {
	bf := NewBruteForceProtector(DefaultBruteForceConfig())
	defer bf.Stop()

	tests := []struct {
		name     string
		headers  map[string]string
		remoteAddr string
		want     string
	}{
		{
			name: "X-Forwarded-For",
			headers: map[string]string{
				"X-Forwarded-For": "1.2.3.4, 5.6.7.8",
			},
			want: "1.2.3.4",
		},
		{
			name: "X-Real-IP",
			headers: map[string]string{
				"X-Real-Ip": "9.10.11.12",
			},
			want: "9.10.11.12",
		},
		{
			name:       "RemoteAddr",
			remoteAddr: "192.168.1.1:12345",
			want:       "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			if tt.remoteAddr != "" {
				req.RemoteAddr = tt.remoteAddr
			}

			got := bf.getKey(req)
			if got != tt.want {
				t.Errorf("getKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBruteForceProtector_Middleware(t *testing.T) {
	config := BruteForceConfig{
		MaxAttempts:   2,
		Window:        time.Hour,
		BlockDuration: time.Hour,
		ExemptPaths:   []string{"/health"},
		SuccessReset:  false,
		EnableLogging: false,
	}

	bf := NewBruteForceProtector(config)
	defer bf.Stop()

	handler := bf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	t.Run("allows requests under limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Status = %d, want 200", rr.Code)
		}
	})

	t.Run("blocks after max attempts", func(t *testing.T) {
		// Create a handler that returns 401
		authHandler := bf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))

		// Make requests up to and exceeding limit
		for i := 0; i < config.MaxAttempts+1; i++ {
			req := httptest.NewRequest(http.MethodGet, "/api/auth", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			rr := httptest.NewRecorder()
			authHandler.ServeHTTP(rr, req)
		}

		// Next request should be blocked
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("Status = %d, want 429", rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "too many failed attempts") {
			t.Errorf("Body = %q, want to contain 'too many failed attempts'", rr.Body.String())
		}
	})

	t.Run("exempt paths bypass protection", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Status = %d, want 200 for exempt path", rr.Code)
		}
	})
}

func TestBruteForceProtector_WindowExpiration(t *testing.T) {
	config := BruteForceConfig{
		MaxAttempts:   2,
		Window:        100 * time.Millisecond,
		BlockDuration: time.Hour,
		EnableLogging: false,
	}

	bf := NewBruteForceProtector(config)
	defer bf.Stop()

	key := "test-key"

	// Record a failure
	bf.RecordFailure(key)

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Failure should be reset
	if attempts := bf.getAttempts(key); attempts != 0 {
		t.Errorf("Attempts = %d, want 0 after window expiration", attempts)
	}
}
