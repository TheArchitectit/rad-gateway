package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_Handler_AllowedOrigin(t *testing.T) {
	config := DefaultCORSConfig()
	cors := NewCORS(config)

	 handler := cors.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	origin := rr.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://localhost:3000" {
		t.Errorf("expected Access-Control-Allow-Origin to be http://localhost:3000, got %s", origin)
	}
}

func TestCORS_Handler_Preflight(t *testing.T) {
	config := DefaultCORSConfig()
	cors := NewCORS(config)

	handler := cors.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status %d for preflight, got %d", http.StatusNoContent, rr.Code)
	}

	origin := rr.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://localhost:3000" {
		t.Errorf("expected Access-Control-Allow-Origin to be set, got %s", origin)
	}

	methods := rr.Header().Get("Access-Control-Allow-Methods")
	if methods == "" {
		t.Error("expected Access-Control-Allow-Methods to be set")
	}
}

func TestCORS_Handler_DisallowedOrigin(t *testing.T) {
	config := DefaultCORSConfig()
	cors := NewCORS(config)

	handler := cors.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://malicious-site.com")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Request should still succeed, just without CORS headers
	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// But no CORS headers should be set for disallowed origins
	origin := rr.Header().Get("Access-Control-Allow-Origin")
	if origin == "http://malicious-site.com" {
		t.Error("Access-Control-Allow-Origin should not be set for disallowed origins")
	}
}

func TestCORS_Handler_Wildcard(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{http.MethodGet},
		AllowCredentials: false,
	}
	cors := NewCORS(config)

	handler := cors.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://any-origin.com")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	origin := rr.Header().Get("Access-Control-Allow-Origin")
	if origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin to be *, got %s", origin)
	}
}

func TestCORS_Handler_Credentials(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{http.MethodGet},
		AllowCredentials: true,
	}
	cors := NewCORS(config)

	handler := cors.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	creds := rr.Header().Get("Access-Control-Allow-Credentials")
	if creds != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials to be true, got %s", creds)
	}
}

func TestCORS_Handler_NoOrigin(t *testing.T) {
	config := DefaultCORSConfig()
	cors := NewCORS(config)

	handler := cors.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request with no Origin header (same-origin, server-to-server, etc.)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}
