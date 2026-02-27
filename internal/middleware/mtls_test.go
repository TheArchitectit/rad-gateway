// Package middleware provides mTLS authentication tests
package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDefaultMTLSConfig(t *testing.T) {
	cfg := DefaultMTLSConfig()

	if cfg.Enabled != false {
		t.Error("expected Enabled to be false by default")
	}

	if cfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Error("expected ClientAuth to be RequireAndVerifyClientCert by default")
	}
}

func TestMTLSConfig_Validate_Disabled(t *testing.T) {
	cfg := MTLSConfig{Enabled: false}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error when disabled, got %v", err)
	}
}

func TestMTLSConfig_Validate_MissingCert(t *testing.T) {
	cfg := MTLSConfig{
		Enabled: true,
		KeyFile: "/path/to/key.pem",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for missing cert file")
	}

	if err.Error() != "TLS certificate file is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestMTLSConfig_Validate_MissingKey(t *testing.T) {
	cfg := MTLSConfig{
		Enabled:  true,
		CertFile: "/path/to/cert.pem",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for missing key file")
	}

	if err.Error() != "TLS key file is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestMTLSConfig_Validate_MissingCA(t *testing.T) {
	cfg := MTLSConfig{
		Enabled:    true,
		CertFile:   "/path/to/cert.pem",
		KeyFile:    "/path/to/key.pem",
		ClientAuth: tls.RequireAndVerifyClientCert,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for missing CA file")
	}
}

func TestMTLSConfig_Validate_NonExistentFiles(t *testing.T) {
	cfg := MTLSConfig{
		Enabled:  true,
		CertFile: "/nonexistent/cert.pem",
		KeyFile:  "/nonexistent/key.pem",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for non-existent cert file")
	}
}

func TestLoadMTLSConfig_FromEnv(t *testing.T) {
	// Save and restore env vars
	oldEnabled := os.Getenv("RAD_TLS_ENABLED")
	oldCert := os.Getenv("RAD_TLS_CERT_FILE")
	oldKey := os.Getenv("RAD_TLS_KEY_FILE")
	oldCA := os.Getenv("RAD_TLS_CA_FILE")
	oldClientAuth := os.Getenv("RAD_TLS_CLIENT_AUTH")
	defer func() {
		os.Setenv("RAD_TLS_ENABLED", oldEnabled)
		os.Setenv("RAD_TLS_CERT_FILE", oldCert)
		os.Setenv("RAD_TLS_KEY_FILE", oldKey)
		os.Setenv("RAD_TLS_CA_FILE", oldCA)
		os.Setenv("RAD_TLS_CLIENT_AUTH", oldClientAuth)
	}()

	os.Setenv("RAD_TLS_ENABLED", "true")
	os.Setenv("RAD_TLS_CERT_FILE", "/certs/server.crt")
	os.Setenv("RAD_TLS_KEY_FILE", "/certs/server.key")
	os.Setenv("RAD_TLS_CA_FILE", "/certs/ca.crt")
	os.Setenv("RAD_TLS_CLIENT_AUTH", "require-any")

	cfg := LoadMTLSConfig()

	if !cfg.Enabled {
		t.Error("expected Enabled to be true")
	}

	if cfg.CertFile != "/certs/server.crt" {
		t.Errorf("expected CertFile to be /certs/server.crt, got %s", cfg.CertFile)
	}

	if cfg.KeyFile != "/certs/server.key" {
		t.Errorf("expected KeyFile to be /certs/server.key, got %s", cfg.KeyFile)
	}

	if cfg.CAFile != "/certs/ca.crt" {
		t.Errorf("expected CAFile to be /certs/ca.crt, got %s", cfg.CAFile)
	}

	if cfg.ClientAuth != tls.RequireAnyClientCert {
		t.Errorf("expected ClientAuth to be RequireAnyClientCert, got %v", cfg.ClientAuth)
	}
}

func TestLoadMTLSConfig_ClientAuthModes(t *testing.T) {
	tests := []struct {
		name       string
		clientAuth string
		expected   tls.ClientAuthType
	}{
		{"none", "none", tls.NoClientCert},
		{"no", "no", tls.NoClientCert},
		{"request", "request", tls.RequestClientCert},
		{"require-any", "require-any", tls.RequireAnyClientCert},
		{"verify-if-given", "verify-if-given", tls.VerifyClientCertIfGiven},
		{"default", "", tls.RequireAndVerifyClientCert},
		{"require", "require", tls.RequireAndVerifyClientCert},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldClientAuth := os.Getenv("RAD_TLS_CLIENT_AUTH")
			defer os.Setenv("RAD_TLS_CLIENT_AUTH", oldClientAuth)

			if tt.clientAuth != "" {
				os.Setenv("RAD_TLS_CLIENT_AUTH", tt.clientAuth)
			} else {
				os.Unsetenv("RAD_TLS_CLIENT_AUTH")
			}

			cfg := LoadMTLSConfig()
			if cfg.ClientAuth != tt.expected {
				t.Errorf("expected ClientAuth %v, got %v", tt.expected, cfg.ClientAuth)
			}
		})
	}
}

func TestMTLSMiddleware_Disabled(t *testing.T) {
	cfg := MTLSConfig{Enabled: false}
	middleware := NewMTLSMiddleware(cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	wrapped := middleware.Handler(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestMTLSMiddleware_NoTLS(t *testing.T) {
	cfg := MTLSConfig{
		Enabled:    true,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}
	middleware := NewMTLSMiddleware(cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.Handler(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// No TLS in request
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for non-TLS request, got %d", rec.Code)
	}
}

func TestMTLSConfig_TLSConfig_Disabled(t *testing.T) {
	cfg := MTLSConfig{Enabled: false}
	tlsCfg, err := cfg.TLSConfig()

	if err != nil {
		t.Errorf("expected no error when disabled, got %v", err)
	}

	if tlsCfg != nil {
		t.Error("expected nil TLS config when disabled")
	}
}

func TestMTLSConfig_TLSConfig_Invalid(t *testing.T) {
	cfg := MTLSConfig{
		Enabled:  true,
		CertFile: "/nonexistent/cert.pem",
		KeyFile:  "/nonexistent/key.pem",
	}

	_, err := cfg.TLSConfig()
	if err == nil {
		t.Error("expected error for invalid config")
	}
}
