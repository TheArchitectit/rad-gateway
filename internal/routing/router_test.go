package routing

import (
	"context"
	"testing"

	"radgateway/internal/models"
	"radgateway/internal/provider"
)

func TestDispatchReturnsSuccessWithAvailableAdapter(t *testing.T) {
	registry := provider.NewRegistry(provider.NewMockAdapter())
	router := New(registry, map[string][]provider.Candidate{
		"gpt-4o-mini": {
			{Name: "mock", Model: "gpt-4o-mini", Weight: 100},
		},
	}, 2)

	res, err := router.Dispatch(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "gpt-4o-mini",
		Payload: models.ChatCompletionRequest{Model: "gpt-4o-mini", Messages: []models.Message{{Role: "user", Content: "hi"}}},
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(res.Attempts) != 1 || res.Attempts[0].Status != "success" {
		t.Fatalf("expected one successful attempt, got %+v", res.Attempts)
	}
}

func TestDispatchFailsWhenNoAdapterFound(t *testing.T) {
	registry := provider.NewRegistry()
	router := New(registry, map[string][]provider.Candidate{
		"gpt-4o-mini": {
			{Name: "missing", Model: "gpt-4o-mini", Weight: 100},
		},
	}, 1)

	res, err := router.Dispatch(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "gpt-4o-mini",
		Payload: models.ChatCompletionRequest{Model: "gpt-4o-mini"},
	})
	if err == nil {
		t.Fatalf("expected error when adapter is missing")
	}
	if len(res.Attempts) != 1 || res.Attempts[0].Status != "error" {
		t.Fatalf("expected one failed attempt, got %+v", res.Attempts)
	}
}
