package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractAPIKeyPriority(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/models?key=query-key", nil)
	req.Header.Set("Authorization", "Bearer bearer-key")
	req.Header.Set("x-api-key", "header-key")
	req.Header.Set("x-goog-api-key", "goog-key")

	if got := extractAPIKey(req); got != "bearer-key" {
		t.Fatalf("expected bearer key, got %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/models?key=query-key", nil)
	req.Header.Set("x-api-key", "header-key")
	req.Header.Set("x-goog-api-key", "goog-key")
	if got := extractAPIKey(req); got != "header-key" {
		t.Fatalf("expected x-api-key value, got %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/models?key=query-key", nil)
	req.Header.Set("x-goog-api-key", "goog-key")
	if got := extractAPIKey(req); got != "goog-key" {
		t.Fatalf("expected x-goog-api-key value, got %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/models?key=query-key", nil)
	if got := extractAPIKey(req); got != "query-key" {
		t.Fatalf("expected query key, got %q", got)
	}
}

func TestAuthenticatorRequireAcceptsValidKey(t *testing.T) {
	auth := NewAuthenticator(map[string]string{"default": "test-secret"})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := GetAPIKeyName(r.Context()); got != "default" {
			t.Fatalf("expected api key name default, got %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	h := auth.Require(next)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer test-secret")

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestAuthenticatorRequireRejectsInvalidKey(t *testing.T) {
	auth := NewAuthenticator(map[string]string{"default": "test-secret"})
	h := auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("x-api-key", "wrong")

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "invalid api key") {
		t.Fatalf("expected invalid api key message, got %q", rr.Body.String())
	}
}

func TestWithRequestContextSetsHeaders(t *testing.T) {
	h := WithRequestContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetRequestID(r.Context()) == "" {
			t.Fatalf("expected request id in context")
		}
		if GetTraceID(r.Context()) == "" {
			t.Fatalf("expected trace id in context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	h.ServeHTTP(rr, req)

	requestID := rr.Header().Get("X-Request-Id")
	traceID := rr.Header().Get("X-Trace-Id")
	if requestID == "" {
		t.Fatalf("expected response X-Request-Id header")
	}
	if traceID != requestID {
		t.Fatalf("expected trace id to default to request id, got request=%q trace=%q", requestID, traceID)
	}
}
