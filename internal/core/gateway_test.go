package core

import (
	"context"
	"testing"

	"radgateway/internal/middleware"
	"radgateway/internal/models"
	"radgateway/internal/provider"
	"radgateway/internal/routing"
	"radgateway/internal/trace"
	"radgateway/internal/usage"
)

func TestGatewayHandleSuccessRecordsUsageAndTrace(t *testing.T) {
	registry := provider.NewRegistry(provider.NewMockAdapter())
	router := routing.New(registry, map[string][]provider.Candidate{
		"gpt-4o-mini": {
			{Name: "mock", Model: "gpt-4o-mini", Weight: 100},
		},
	}, 2)
	usageSink := usage.NewInMemory(20)
	traceStore := trace.NewStore(20)
	gateway := New(router, usageSink, traceStore)

	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.KeyRequestID, "req-1")
	ctx = context.WithValue(ctx, middleware.KeyTraceID, "trace-1")
	ctx = context.WithValue(ctx, middleware.KeyAPIName, "default")

	res, attempts, err := gateway.Handle(ctx, "chat", "gpt-4o-mini", models.ChatCompletionRequest{
		Model:    "gpt-4o-mini",
		Messages: []models.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if res.Provider != "mock" {
		t.Fatalf("expected provider mock, got %q", res.Provider)
	}
	if len(attempts) != 1 || attempts[0].Status != "success" {
		t.Fatalf("expected one successful attempt, got %+v", attempts)
	}
	if got := len(usageSink.List(10)); got != 1 {
		t.Fatalf("expected one usage record, got %d", got)
	}
	if got := len(traceStore.List(10)); got < 2 {
		t.Fatalf("expected trace accepted+completed events, got %d", got)
	}
}
