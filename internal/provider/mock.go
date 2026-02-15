package provider

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"radgateway/internal/models"
)

type MockAdapter struct{}

func NewMockAdapter() *MockAdapter {
	return &MockAdapter{}
}

func (m *MockAdapter) Name() string {
	return "mock"
}

func (m *MockAdapter) Execute(_ context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
	switch req.APIType {
	case "chat":
		payload, ok := req.Payload.(models.ChatCompletionRequest)
		if !ok {
			return models.ProviderResult{}, fmt.Errorf("invalid chat payload")
		}
		content := "No prompt"
		if len(payload.Messages) > 0 {
			content = "Echo: " + payload.Messages[len(payload.Messages)-1].Content
		}
		return models.ProviderResult{
			Model:    model,
			Provider: m.Name(),
			Status:   "success",
			Usage:    models.Usage{PromptTokens: 12, CompletionTokens: 18, TotalTokens: 30, CostTotal: 0.0004},
			Payload: models.ChatCompletionResponse{
				ID:     "chatcmpl_" + randID(),
				Object: "chat.completion",
				Model:  model,
				Choices: []models.ChatChoice{
					{Index: 0, Message: models.Message{Role: "assistant", Content: content}},
				},
				Usage: models.Usage{PromptTokens: 12, CompletionTokens: 18, TotalTokens: 30, CostTotal: 0.0004},
			},
		}, nil
	case "responses", "messages", "gemini", "images", "transcriptions":
		return models.ProviderResult{
			Model:    model,
			Provider: m.Name(),
			Status:   "success",
			Usage:    models.Usage{PromptTokens: 10, CompletionTokens: 10, TotalTokens: 20, CostTotal: 0.0003},
			Payload: models.GenericResponse{
				ID:     "resp_" + randID(),
				Object: "response",
				Model:  model,
				Output: "Mock provider response",
			},
		}, nil
	case "embeddings":
		return models.ProviderResult{
			Model:    model,
			Provider: m.Name(),
			Status:   "success",
			Usage:    models.Usage{PromptTokens: 8, CompletionTokens: 0, TotalTokens: 8, CostTotal: 0.0001},
			Payload: models.EmbeddingsResponse{
				Object: "list",
				Model:  model,
				Data: []models.Embedding{{
					Object:    "embedding",
					Index:     0,
					Embedding: []float64{0.01, 0.02, 0.03, 0.04},
				}},
				Usage: models.Usage{PromptTokens: 8, CompletionTokens: 0, TotalTokens: 8, CostTotal: 0.0001},
			},
		}, nil
	default:
		return models.ProviderResult{}, fmt.Errorf("unsupported api type: %s", req.APIType)
	}
}

func randID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
