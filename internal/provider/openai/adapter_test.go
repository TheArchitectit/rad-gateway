package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"radgateway/internal/models"
)

func TestAdapter_Name(t *testing.T) {
	adapter := NewAdapter("test-key")
	if adapter.Name() != "openai" {
		t.Errorf("Expected name 'openai', got %q", adapter.Name())
	}
}

func TestAdapter_ExecuteChat_NonStreaming(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		expectError    bool
		checkResult    func(t *testing.T, result models.ProviderResult)
	}{
		{
			name:         "successful chat completion",
			serverStatus: http.StatusOK,
			serverResponse: `{
				"id": "chatcmpl-test123",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-4o",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hello! How can I help you?"
					},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 20,
					"total_tokens": 30
				}
			}`,
			checkResult: func(t *testing.T, result models.ProviderResult) {
				if result.Provider != "openai" {
					t.Errorf("Provider mismatch: got %q", result.Provider)
				}
				if result.Status != "success" {
					t.Errorf("Status mismatch: got %q", result.Status)
				}
				if result.Usage.TotalTokens != 30 {
					t.Errorf("TotalTokens mismatch: got %d", result.Usage.TotalTokens)
				}

				resp, ok := result.Payload.(models.ChatCompletionResponse)
				if !ok {
					t.Fatalf("Expected ChatCompletionResponse, got %T", result.Payload)
				}
				if resp.ID != "chatcmpl-test123" {
					t.Errorf("ID mismatch: got %q", resp.ID)
				}
				if len(resp.Choices) != 1 {
					t.Fatalf("Expected 1 choice, got %d", len(resp.Choices))
				}
				if resp.Choices[0].Message.Content != "Hello! How can I help you?" {
					t.Errorf("Content mismatch: got %q", resp.Choices[0].Message.Content)
				}
			},
		},
		{
			name:         "error response",
			serverStatus: http.StatusUnauthorized,
			serverResponse: `{
				"error": {
					"message": "Invalid API key",
					"type": "invalid_request_error",
					"code": "invalid_api_key"
				}
			}`,
			expectError: true,
		},
		{
			name:         "server error with retry",
			serverStatus: http.StatusServiceUnavailable,
			serverResponse: `{
				"error": {
					"message": "Service temporarily unavailable",
					"type": "api_error"
				}
			}`,
			expectError: true,
		},
		{
			name:         "bad request - no retry",
			serverStatus: http.StatusBadRequest,
			serverResponse: `{
				"error": {
					"message": "Invalid model",
					"type": "invalid_request_error"
				}
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/chat/completions" {
					t.Errorf("Expected /chat/completions, got %s", r.URL.Path)
				}

				// Verify headers
				authHeader := r.Header.Get("Authorization")
				if !strings.HasPrefix(authHeader, "Bearer ") {
					t.Errorf("Expected Bearer token, got %q", authHeader)
				}

				contentType := r.Header.Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("Expected application/json, got %q", contentType)
				}

				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			adapter := NewAdapter("test-key", WithBaseURL(server.URL))
			result, err := adapter.Execute(context.Background(), models.ProviderRequest{
				APIType: "chat",
				Model:   "gpt-4o",
				Payload: models.ChatCompletionRequest{
					Model: "gpt-4o",
					Messages: []models.Message{
						{Role: "user", Content: "Hello"},
					},
				},
			}, "gpt-4o")

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestAdapter_ExecuteChat_Streaming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []string{
			`{"id":"chatcmpl-stream","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-stream","object":"chat.completion.chunk","created":1677652289,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-stream","object":"chat.completion.chunk","created":1677652290,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-stream","object":"chat.completion.chunk","created":1677652291,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			w.(http.Flusher).Flush()
			time.Sleep(10 * time.Millisecond)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	adapter := NewAdapter("test-key", WithBaseURL(server.URL))
	result, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "gpt-4o",
		Payload: models.ChatCompletionRequest{
			Model: "gpt-4o",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
			Stream: true,
		},
	}, "gpt-4o")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Provider != "openai" {
		t.Errorf("Provider mismatch: got %q", result.Provider)
	}

	// Verify streaming response payload
	streamResp, ok := result.Payload.(*StreamingResponse)
	if !ok {
		t.Fatalf("Expected StreamingResponse, got %T", result.Payload)
	}
	defer streamResp.Close()

	// Read and verify stream content
	data, err := io.ReadAll(streamResp.Reader)
	if err != nil {
		t.Fatalf("Failed to read stream: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Hello") {
		t.Errorf("Stream content missing 'Hello': %s", content)
	}
	if !strings.Contains(content, "assistant") {
		t.Errorf("Stream content missing 'assistant': %s", content)
	}
}

func TestAdapter_ExecuteEmbeddings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("Expected /embeddings, got %s", r.URL.Path)
		}

		// Verify request body
		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"object": "list",
			"data": [{
				"object": "embedding",
				"index": 0,
				"embedding": [0.1, 0.2, 0.3, 0.4, 0.5]
			}],
			"model": "text-embedding-3-small",
			"usage": {
				"prompt_tokens": 8,
				"total_tokens": 8
			}
		}`))
	}))
	defer server.Close()

	adapter := NewAdapter("test-key", WithBaseURL(server.URL))
	result, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "embeddings",
		Model:   "text-embedding-3-small",
		Payload: models.EmbeddingsRequest{
			Model: "text-embedding-3-small",
			Input: "Hello world",
		},
	}, "text-embedding-3-small")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Provider != "openai" {
		t.Errorf("Provider mismatch: got %q", result.Provider)
	}

	if result.Usage.PromptTokens != 8 {
		t.Errorf("PromptTokens mismatch: got %d", result.Usage.PromptTokens)
	}

	embeddingsResp, ok := result.Payload.(models.EmbeddingsResponse)
	if !ok {
		t.Fatalf("Expected EmbeddingsResponse, got %T", result.Payload)
	}

	if len(embeddingsResp.Data) != 1 {
		t.Fatalf("Expected 1 embedding, got %d", len(embeddingsResp.Data))
	}

	if len(embeddingsResp.Data[0].Embedding) != 5 {
		t.Errorf("Expected 5 dimensions, got %d", len(embeddingsResp.Data[0].Embedding))
	}
}

func TestAdapter_Execute_UnsupportedType(t *testing.T) {
	adapter := NewAdapter("test-key")
	_, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "unsupported",
		Model:   "gpt-4o",
		Payload: struct{}{},
	}, "gpt-4o")

	if err == nil {
		t.Error("Expected error for unsupported API type")
	}

	if !strings.Contains(err.Error(), "unsupported api type") {
		t.Errorf("Expected 'unsupported api type' error, got: %v", err)
	}
}

func TestAdapter_Execute_InvalidPayload(t *testing.T) {
	adapter := NewAdapter("test-key")
	_, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "gpt-4o",
		Payload: "invalid payload type",
	}, "gpt-4o")

	if err == nil {
		t.Error("Expected error for invalid payload")
	}

	if !strings.Contains(err.Error(), "invalid chat payload type") {
		t.Errorf("Expected payload type error, got: %v", err)
	}
}

func TestAdapter_WithOptions(t *testing.T) {
	customClient := &http.Client{Timeout: 5 * time.Second}
	adapter := NewAdapter(
		"test-key",
		WithHTTPClient(customClient),
		WithBaseURL("https://custom.openai.com/v1"),
		WithTimeout(30*time.Second),
		WithRetryConfig(5, 1*time.Second),
	)

	if adapter.config.BaseURL != "https://custom.openai.com/v1" {
		t.Errorf("BaseURL mismatch: got %q", adapter.config.BaseURL)
	}

	if adapter.config.MaxRetries != 5 {
		t.Errorf("MaxRetries mismatch: got %d", adapter.config.MaxRetries)
	}

	if adapter.config.RetryDelay != 1*time.Second {
		t.Errorf("RetryDelay mismatch: got %v", adapter.config.RetryDelay)
	}

	// The httpClient should be the custom one
	if adapter.httpClient != customClient {
		t.Error("HTTP client not set correctly")
	}
}

func TestAdapter_ContextCancellation(t *testing.T) {
	// Slow server that takes longer than context timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test"}`))
	}))
	defer server.Close()

	adapter := NewAdapter("test-key", WithBaseURL(server.URL), WithTimeout(100*time.Millisecond))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := adapter.Execute(ctx, models.ProviderRequest{
		APIType: "chat",
		Model:   "gpt-4o",
		Payload: models.ChatCompletionRequest{
			Model: "gpt-4o",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}, "gpt-4o")

	if err == nil {
		t.Error("Expected error due to context cancellation")
	}
}

func TestStreamingResponse_Usage(t *testing.T) {
	pr, _ := io.Pipe()
	defer pr.Close()

	usage := models.Usage{PromptTokens: 10}
	var mu sync.Mutex

	resp := &StreamingResponse{
		Reader: pr,
		usage:  &usage,
		mu:     &mu,
	}

	// Test initial usage
	u := resp.Usage()
	if u.PromptTokens != 10 {
		t.Errorf("PromptTokens mismatch: got %d", u.PromptTokens)
	}
}
