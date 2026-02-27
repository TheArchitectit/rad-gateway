// Package generic provides tests for the generic HTTP adapter.
package generic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"radgateway/internal/models"
)

func TestAdapter_Name(t *testing.T) {
	adapter := NewAdapter("http://localhost", "test-key")
	if adapter.Name() != "generic" {
		t.Errorf("Name() = %v, want %v", adapter.Name(), "generic")
	}
}

func TestAdapter_ExecuteChat_NonStreaming(t *testing.T) {
	tests := []struct {
		name           string
		response       map[string]interface{}
		statusCode     int
		wantErr        bool
		wantErrContain string
	}{
		{
			name: "successful chat completion",
			response: map[string]interface{}{
				"id":      "chatcmpl-123",
				"object":  "chat.completion",
				"created": 1677652288,
				"model":   "gpt-4o-mini",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]string{
							"role":    "assistant",
							"content": "Hello! How can I help you today?",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]int{
					"prompt_tokens":     10,
					"completion_tokens": 20,
					"total_tokens":      30,
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "rate limit error",
			statusCode: http.StatusTooManyRequests,
			response: map[string]interface{}{
				"error": map[string]string{
					"message": "Rate limit exceeded",
					"type":    "rate_limit_exceeded",
				},
			},
			wantErr:        true,
			wantErrContain: "api returned status 429",
		},
		{
			name:       "authentication error",
			statusCode: http.StatusUnauthorized,
			response: map[string]interface{}{
				"error": map[string]string{
					"message": "Invalid API key",
					"type":    "authentication_error",
				},
			},
			wantErr:        true,
			wantErrContain: "Invalid API key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/chat/completions" {
					t.Errorf("Expected /chat/completions, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			adapter := NewAdapter(server.URL, "test-key")

			req := models.ProviderRequest{
				APIType: "chat",
				Payload: models.ChatCompletionRequest{
					Messages: []models.Message{
						{Role: "user", Content: "Hello!"},
					},
					Stream: false,
				},
			}

			result, err := adapter.Execute(context.Background(), req, "gpt-4o-mini")

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
				if tt.wantErrContain != "" && !contains(err.Error(), tt.wantErrContain) {
					t.Errorf("Expected error to contain %q, got %q", tt.wantErrContain, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Status != "success" {
				t.Errorf("Expected status 'success', got %q", result.Status)
			}

			if result.Usage.TotalTokens != 30 {
				t.Errorf("Expected 30 total tokens, got %d", result.Usage.TotalTokens)
			}
		})
	}
}

func TestAdapter_ExecuteChat_Streaming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []string{
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{"role":"assistant"}}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{"content":"Hello"}}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{"content":"!"}}]}`,
		}

		for _, chunk := range chunks {
			w.Write([]byte("data: " + chunk + "\n\n"))
		}
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	adapter := NewAdapter(server.URL, "test-key")

	req := models.ProviderRequest{
		APIType: "chat",
		Payload: models.ChatCompletionRequest{
			Messages: []models.Message{
				{Role: "user", Content: "Say hello!"},
			},
			Stream: true,
		},
	}

	result, err := adapter.Execute(context.Background(), req, "gpt-4o-mini")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected status 'success', got %q", result.Status)
	}

	// Verify streaming response
	stream, ok := result.Payload.(*StreamingResponse)
	if !ok {
		t.Fatalf("Expected *StreamingResponse, got %T", result.Payload)
	}

	// Read the stream
	data, err := io.ReadAll(stream.Reader)
	if err != nil {
		t.Fatalf("Failed to read stream: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty stream data")
	}
}

func TestAdapter_ExecuteEmbeddings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("Expected /embeddings, got %s", r.URL.Path)
		}

		response := map[string]interface{}{
			"object": "list",
			"data": []map[string]interface{}{
				{
					"object":    "embedding",
					"embedding": []float64{0.1, 0.2, 0.3},
					"index":     0,
				},
			},
			"model": "text-embedding-3-small",
			"usage": map[string]int{
				"prompt_tokens": 5,
				"total_tokens":  5,
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	adapter := NewAdapter(server.URL, "test-key")

	req := models.ProviderRequest{
		APIType: "embeddings",
		Payload: models.EmbeddingsRequest{
			Input: "The quick brown fox",
			Model: "text-embedding-3-small",
		},
	}

	result, err := adapter.Execute(context.Background(), req, "text-embedding-3-small")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected status 'success', got %q", result.Status)
	}

	if result.Usage.PromptTokens != 5 {
		t.Errorf("Expected 5 prompt tokens, got %d", result.Usage.PromptTokens)
	}
}

func TestAdapter_RetryOnServerError(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// Success on 3rd attempt
		response := map[string]interface{}{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"created": 1677652288,
			"model":   "gpt-4o-mini",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	adapter := NewAdapter(server.URL, "test-key")

	req := models.ProviderRequest{
		APIType: "chat",
		Payload: models.ChatCompletionRequest{
			Messages: []models.Message{
				{Role: "user", Content: "Hello!"},
			},
		},
	}

	result, err := adapter.Execute(context.Background(), req, "gpt-4o-mini")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected status 'success', got %q", result.Status)
	}

	if attemptCount < 3 {
		t.Errorf("Expected at least 3 attempts, got %d", attemptCount)
	}
}

func TestAdapter_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := NewAdapter(server.URL, "test-key", WithTimeout(50*time.Millisecond))

	req := models.ProviderRequest{
		APIType: "chat",
		Payload: models.ChatCompletionRequest{
			Messages: []models.Message{
				{Role: "user", Content: "Hello!"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	_, err := adapter.Execute(ctx, req, "gpt-4o-mini")
	if err == nil {
		t.Error("Expected error due to context cancellation")
	}
}

func TestAdapter_UnsupportedType(t *testing.T) {
	adapter := NewAdapter("http://localhost", "test-key")

	req := models.ProviderRequest{
		APIType: "unknown",
		Payload: models.ChatCompletionRequest{},
	}

	_, err := adapter.Execute(context.Background(), req, "gpt-4o-mini")
	if err == nil {
		t.Error("Expected error for unsupported API type")
	}
}

func TestAdapter_WithAuthType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check custom auth header
		authHeader := r.Header.Get("X-Custom-Auth")
		if authHeader != "ApiKey test-key" {
			t.Errorf("Expected 'ApiKey test-key', got %q", authHeader)
		}

		response := map[string]interface{}{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"created": 1677652288,
			"model":   "model-1",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	adapter := NewAdapter(server.URL, "test-key",
		WithAuthType("api-key", "X-Custom-Auth", "ApiKey "),
	)

	req := models.ProviderRequest{
		APIType: "chat",
		Payload: models.ChatCompletionRequest{
			Messages: []models.Message{
				{Role: "user", Content: "Hello!"},
			},
		},
	}

	result, err := adapter.Execute(context.Background(), req, "model-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected status 'success', got %q", result.Status)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
