package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"radgateway/internal/core"
	"radgateway/internal/provider"
	"radgateway/internal/routing"
	"radgateway/internal/trace"
	"radgateway/internal/usage"
)

func newTestMux() *http.ServeMux {
	registry := provider.NewRegistry(provider.NewMockAdapter())
	router := routing.New(registry, map[string][]provider.Candidate{
		"gpt-4o-mini":       {{Name: "mock", Model: "gpt-4o-mini", Weight: 100}},
		"claude-3-5-sonnet": {{Name: "mock", Model: "claude-3-5-sonnet", Weight: 100}},
		"gemini-1.5-flash":  {{Name: "mock", Model: "gemini-1.5-flash", Weight: 100}},
	}, 2)
	g := core.New(router, usage.NewInMemory(50), trace.NewStore(50))
	mux := http.NewServeMux()
	NewHandlers(g).Register(mux)
	return mux
}

func TestHealthEndpoint(t *testing.T) {
	mux := newTestMux()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestChatCompletionsEndpoint(t *testing.T) {
	mux := newTestMux()
	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hello"}]}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if payload["object"] != "chat.completion" {
		t.Fatalf("expected chat.completion object, got %v", payload["object"])
	}
}

func TestModelsEndpoint(t *testing.T) {
	mux := newTestMux()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
