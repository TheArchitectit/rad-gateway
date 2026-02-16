// Package anthropic provides an adapter for Anthropic's Claude API.
package anthropic

import (
	"bufio"
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
	if adapter.Name() != "anthropic" {
		t.Errorf("Expected name 'anthropic', got %q", adapter.Name())
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
				"id": "msg_01XgVYxVqW32TYn5Ts4RYRPW",
				"type": "message",
				"role": "assistant",
				"model": "claude-3-5-sonnet-20241022",
				"content": [
					{"type": "text", "text": "Hello! How can I help you today?"}
				],
				"stop_reason": "end_turn",
				"stop_sequence": null,
				"usage": {
					"input_tokens": 12,
					"output_tokens": 9
				}
			}`,
			checkResult: func(t *testing.T, result models.ProviderResult) {
				if result.Provider != "anthropic" {
					t.Errorf("Provider mismatch: got %q", result.Provider)
				}
				if result.Status != "success" {
					t.Errorf("Status mismatch: got %q", result.Status)
				}
				if result.Usage.TotalTokens != 21 {
					t.Errorf("TotalTokens mismatch: expected 21, got %d", result.Usage.TotalTokens)
				}
				if result.Usage.PromptTokens != 12 {
					t.Errorf("PromptTokens mismatch: expected 12, got %d", result.Usage.PromptTokens)
				}
				if result.Usage.CompletionTokens != 9 {
					t.Errorf("CompletionTokens mismatch: expected 9, got %d", result.Usage.CompletionTokens)
				}

				resp, ok := result.Payload.(models.ChatCompletionResponse)
				if !ok {
					t.Fatalf("Expected ChatCompletionResponse, got %T", result.Payload)
				}
				if resp.ID != "msg_01XgVYxVqW32TYn5Ts4RYRPW" {
					t.Errorf("ID mismatch: got %q", resp.ID)
				}
				if len(resp.Choices) != 1 {
					t.Fatalf("Expected 1 choice, got %d", len(resp.Choices))
				}
				if resp.Choices[0].Message.Content != "Hello! How can I help you today?" {
					t.Errorf("Content mismatch: got %q", resp.Choices[0].Message.Content)
				}
				if resp.Choices[0].Message.Role != "assistant" {
					t.Errorf("Role mismatch: got %q", resp.Choices[0].Message.Role)
				}
				if resp.Choices[0].FinishReason != "stop" {
					t.Errorf("FinishReason mismatch: expected 'stop', got %q", resp.Choices[0].FinishReason)
				}
			},
		},
		{
			name:         "max_tokens stop reason",
			serverStatus: http.StatusOK,
			serverResponse: `{
				"id": "msg_test123",
				"type": "message",
				"role": "assistant",
				"model": "claude-3-haiku-20240307",
				"content": [
					{"type": "text", "text": "This is a truncated response"}
				],
				"stop_reason": "max_tokens",
				"stop_sequence": null,
				"usage": {
					"input_tokens": 10,
					"output_tokens": 100
				}
			}`,
			checkResult: func(t *testing.T, result models.ProviderResult) {
				resp, ok := result.Payload.(models.ChatCompletionResponse)
				if !ok {
					t.Fatalf("Expected ChatCompletionResponse, got %T", result.Payload)
				}
				if resp.Choices[0].FinishReason != "length" {
					t.Errorf("FinishReason mismatch: expected 'length', got %q", resp.Choices[0].FinishReason)
				}
			},
		},
		{
			name:         "error response",
			serverStatus: http.StatusUnauthorized,
			serverResponse: `{
				"type": "error",
				"error": {
					"type": "authentication_error",
					"message": "Invalid API key"
				}
			}`,
			expectError: true,
		},
		{
			name:         "server error with retry",
			serverStatus: http.StatusServiceUnavailable,
			serverResponse: `{
				"type": "error",
				"error": {
					"type": "api_error",
					"message": "Service temporarily unavailable"
				}
			}`,
			expectError: true,
		},
		{
			name:         "bad request - no retry",
			serverStatus: http.StatusBadRequest,
			serverResponse: `{
				"type": "error",
				"error": {
					"type": "invalid_request_error",
					"message": "max_tokens: range error"
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
				if r.URL.Path != "/v1/messages" {
					t.Errorf("Expected /v1/messages, got %s", r.URL.Path)
				}

				// Verify Anthropic-specific headers
				apiKey := r.Header.Get("x-api-key")
				if apiKey != "test-key" {
					t.Errorf("Expected x-api-key 'test-key', got %q", apiKey)
				}

				version := r.Header.Get("anthropic-version")
				if version != "2023-06-01" {
					t.Errorf("Expected anthropic-version '2023-06-01', got %q", version)
				}

				contentType := r.Header.Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("Expected Content-Type 'application/json', got %q", contentType)
				}

				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			adapter := NewAdapter("test-key", WithBaseURL(server.URL))
			result, err := adapter.Execute(context.Background(), models.ProviderRequest{
				APIType: "chat",
				Model:   "claude-3-5-sonnet-20241022",
				Payload: models.ChatCompletionRequest{
					Model: "claude-3-5-sonnet-20241022",
					Messages: []models.Message{
						{Role: "user", Content: "Hello"},
					},
				},
			}, "claude-3-5-sonnet-20241022")

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

		// Anthropic streaming events
		events := []string{
			`event: message_start
data: {"type":"message_start","message":{"id":"msg_stream_123","type":"message","role":"assistant","model":"claude-3-5-sonnet-20241022"}}`,
			`event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
			`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"!"}}`,
			`event: content_block_stop
data: {"type":"content_block_stop","index":0}`,
			`event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}`,
			`event: message_stop
data: {"type":"message_stop"}`,
		}

		for _, event := range events {
			fmt.Fprintf(w, "%s\n\n", event)
			w.(http.Flusher).Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer server.Close()

	adapter := NewAdapter("test-key", WithBaseURL(server.URL))
	result, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "claude-3-5-sonnet-20241022",
		Payload: models.ChatCompletionRequest{
			Model: "claude-3-5-sonnet-20241022",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
			Stream: true,
		},
	}, "claude-3-5-sonnet-20241022")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Provider != "anthropic" {
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

	// Verify the transformed content contains OpenAI-style SSE data
	if !strings.Contains(content, "data:") {
		t.Errorf("Stream content missing 'data:' prefix: %s", content)
	}

	if !strings.Contains(content, "Hello") {
		t.Errorf("Stream content missing 'Hello': %s", content)
	}

	if !strings.Contains(content, "[DONE]") {
		t.Errorf("Stream content missing '[DONE]' marker: %s", content)
	}
}

func TestAdapter_Execute_UnsupportedType(t *testing.T) {
	adapter := NewAdapter("test-key")
	_, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "embeddings",
		Model:   "claude-3-5-sonnet-20241022",
		Payload: struct{}{},
	}, "claude-3-5-sonnet-20241022")

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
		Model:   "claude-3-5-sonnet-20241022",
		Payload: "invalid payload type",
	}, "claude-3-5-sonnet-20241022")

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
		WithBaseURL("https://custom.anthropic.com"),
		WithTimeout(30*time.Second),
		WithVersion("2023-06-01"),
		WithRetryConfig(5, 1*time.Second),
	)

	if adapter.config.BaseURL != "https://custom.anthropic.com" {
		t.Errorf("BaseURL mismatch: got %q", adapter.config.BaseURL)
	}

	if adapter.config.Version != "2023-06-01" {
		t.Errorf("Version mismatch: got %q", adapter.config.Version)
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
		Model:   "claude-3-5-sonnet-20241022",
		Payload: models.ChatCompletionRequest{
			Model: "claude-3-5-sonnet-20241022",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}, "claude-3-5-sonnet-20241022")

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

func TestAdapter_SystemMessageExtraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode request to verify system message was extracted
		var req AnthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Verify system message was extracted
		if req.System != "You are a helpful assistant." {
			t.Errorf("System message not extracted correctly: got %q", req.System)
		}

		// Verify system message is not in messages array
		for _, msg := range req.Messages {
			if msg.Role == "system" {
				t.Error("System message should not be in messages array")
			}
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_test",
			"type": "message",
			"role": "assistant",
			"model": "claude-3-5-sonnet-20241022",
			"content": [{"type": "text", "text": "Hello!"}],
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 10, "output_tokens": 2}
		}`))
	}))
	defer server.Close()

	adapter := NewAdapter("test-key", WithBaseURL(server.URL))
	_, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "claude-3-5-sonnet-20241022",
		Payload: models.ChatCompletionRequest{
			Model: "claude-3-5-sonnet-20241022",
			Messages: []models.Message{
				{Role: "system", Content: "You are a helpful assistant."},
				{Role: "user", Content: "Hello!"},
			},
		},
	}, "claude-3-5-sonnet-20241022")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestAdapter_MaxTokensDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req AnthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Verify default max_tokens is set
		if req.MaxTokens != 4096 {
			t.Errorf("MaxTokens default mismatch: expected 4096, got %d", req.MaxTokens)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_test",
			"type": "message",
			"role": "assistant",
			"model": "claude-3-5-sonnet-20241022",
			"content": [{"type": "text", "text": "Hello!"}],
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 10, "output_tokens": 2}
		}`))
	}))
	defer server.Close()

	adapter := NewAdapter("test-key", WithBaseURL(server.URL))
	_, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "claude-3-5-sonnet-20241022",
		Payload: models.ChatCompletionRequest{
			Model: "claude-3-5-sonnet-20241022",
			Messages: []models.Message{
				{Role: "user", Content: "Hello!"},
			},
			// MaxTokens not set - should default to 4096
		},
	}, "claude-3-5-sonnet-20241022")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestAdapter_ConsecutiveSameRoleMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req AnthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Verify consecutive user messages were merged
		if len(req.Messages) != 2 {
			t.Errorf("Expected 2 messages after merging, got %d", len(req.Messages))
		}

		// First message should be user with merged content
		if req.Messages[0].Role != "user" {
			t.Errorf("First message should be user, got %s", req.Messages[0].Role)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_test",
			"type": "message",
			"role": "assistant",
			"model": "claude-3-5-sonnet-20241022",
			"content": [{"type": "text", "text": "Hello!"}],
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 10, "output_tokens": 2}
		}`))
	}))
	defer server.Close()

	adapter := NewAdapter("test-key", WithBaseURL(server.URL))
	_, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "claude-3-5-sonnet-20241022",
		Payload: models.ChatCompletionRequest{
			Model: "claude-3-5-sonnet-20241022",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
				{Role: "user", Content: "World"}, // Should be merged with previous
				{Role: "assistant", Content: "Hi there!"},
			},
		},
	}, "claude-3-5-sonnet-20241022")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestAdapter_RetryOnServerError(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"type":"error","error":{"type":"api_error","message":"Service unavailable"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_test",
			"type": "message",
			"role": "assistant",
			"model": "claude-3-5-sonnet-20241022",
			"content": [{"type": "text", "text": "Hello!"}],
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 10, "output_tokens": 2}
		}`))
	}))
	defer server.Close()

	adapter := NewAdapter("test-key", WithBaseURL(server.URL), WithRetryConfig(3, 10*time.Millisecond))
	result, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "claude-3-5-sonnet-20241022",
		Payload: models.ChatCompletionRequest{
			Model: "claude-3-5-sonnet-20241022",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}, "claude-3-5-sonnet-20241022")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected success status, got %q", result.Status)
	}

	if requestCount < 2 {
		t.Errorf("Expected at least 2 requests due to retry, got %d", requestCount)
	}
}

func TestStreamTransformer_TransformEvent(t *testing.T) {
	tests := []struct {
		name           string
		eventType      string
		data           string
		expectDone     bool
		expectError    bool
		checkContent   func(t *testing.T, content []byte)
	}{
		{
			name:      "message_start",
			eventType: "message_start",
			data:      `{"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-5-sonnet"}}`,
			checkContent: func(t *testing.T, content []byte) {
				if content == nil {
					t.Error("Expected content for message_start")
					return
				}
				if !strings.Contains(string(content), `"role":"assistant"`) {
					t.Errorf("Expected assistant role in content: %s", string(content))
				}
			},
		},
		{
			name:      "content_block_delta",
			eventType: "content_block_delta",
			data:      `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
			checkContent: func(t *testing.T, content []byte) {
				if content == nil {
					t.Error("Expected content for content_block_delta")
					return
				}
				if !strings.Contains(string(content), "Hello") {
					t.Errorf("Expected 'Hello' in content: %s", string(content))
				}
			},
		},
		{
			name:      "message_delta",
			eventType: "message_delta",
			data:      `{"type":"message_delta","delta":{"stop_reason":"end_turn"}}`,
			checkContent: func(t *testing.T, content []byte) {
				if content == nil {
					t.Error("Expected content for message_delta")
					return
				}
				// Should contain finish_reason
				if !strings.Contains(string(content), "finish_reason") {
					t.Errorf("Expected finish_reason in content: %s", string(content))
				}
			},
		},
		{
			name:       "message_stop",
			eventType:  "message_stop",
			data:       `{"type":"message_stop"}`,
			expectDone: true,
		},
		{
			name:      "ping",
			eventType: "ping",
			data:      `{"type":"ping"}`,
			checkContent: func(t *testing.T, content []byte) {
				if content != nil {
					t.Errorf("Expected nil content for ping, got: %s", string(content))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer := NewStreamTransformer()
			// Initialize with message_start first
			if tt.eventType != "message_start" && tt.eventType != "ping" {
				transformer.TransformEvent("message_start", []byte(`{"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-5-sonnet"}}`))
			}

			content, done, err := transformer.TransformEvent(tt.eventType, []byte(tt.data))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if done != tt.expectDone {
				t.Errorf("Done mismatch: expected %v, got %v", tt.expectDone, done)
			}

			if tt.checkContent != nil {
				tt.checkContent(t, content)
			}
		})
	}
}

func TestRequestTransformer_Transform(t *testing.T) {
	tests := []struct {
		name     string
		input    models.ChatCompletionRequest
		expected func(t *testing.T, req AnthropicRequest)
	}{
		{
			name: "basic request with system message",
			input: models.ChatCompletionRequest{
				Model: "claude-3-5-sonnet",
				Messages: []models.Message{
					{Role: "system", Content: "You are helpful."},
					{Role: "user", Content: "Hello!"},
				},
			},
			expected: func(t *testing.T, req AnthropicRequest) {
				if req.System != "You are helpful." {
					t.Errorf("System mismatch: got %q", req.System)
				}
				if len(req.Messages) != 1 {
					t.Errorf("Expected 1 message, got %d", len(req.Messages))
				}
				if req.Messages[0].Role != "user" {
					t.Errorf("Expected user role, got %s", req.Messages[0].Role)
				}
			},
		},
		{
			name: "max_tokens provided",
			input: models.ChatCompletionRequest{
				Model:     "claude-3-5-sonnet",
				MaxTokens: 1024,
				Messages: []models.Message{
					{Role: "user", Content: "Hello!"},
				},
			},
			expected: func(t *testing.T, req AnthropicRequest) {
				if req.MaxTokens != 1024 {
					t.Errorf("MaxTokens mismatch: expected 1024, got %d", req.MaxTokens)
				}
			},
		},
		{
			name: "temperature and top_p",
			input: models.ChatCompletionRequest{
				Model:       "claude-3-5-sonnet",
				Temperature: 0.7,
				TopP:        0.9,
				Messages: []models.Message{
					{Role: "user", Content: "Hello!"},
				},
			},
			expected: func(t *testing.T, req AnthropicRequest) {
				if req.Temperature == nil || *req.Temperature != 0.7 {
					t.Errorf("Temperature mismatch")
				}
				if req.TopP == nil || *req.TopP != 0.9 {
					t.Errorf("TopP mismatch")
				}
			},
		},
		{
			name: "stop sequences",
			input: models.ChatCompletionRequest{
				Model: "claude-3-5-sonnet",
				Stop:  []string{"STOP", "END"},
				Messages: []models.Message{
					{Role: "user", Content: "Hello!"},
				},
			},
			expected: func(t *testing.T, req AnthropicRequest) {
				if len(req.StopSequences) != 2 {
					t.Errorf("StopSequences length mismatch: expected 2, got %d", len(req.StopSequences))
				}
			},
		},
		{
			name: "user metadata",
			input: models.ChatCompletionRequest{
				Model: "claude-3-5-sonnet",
				User:  "user-123",
				Messages: []models.Message{
					{Role: "user", Content: "Hello!"},
				},
			},
			expected: func(t *testing.T, req AnthropicRequest) {
				if req.Metadata == nil || req.Metadata.UserID != "user-123" {
					t.Errorf("UserID metadata mismatch")
				}
			},
		},
	}

	transformer := NewRequestTransformer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformer.Transform(tt.input)
			tt.expected(t, result)
		})
	}
}

func TestResponseTransformer_Transform(t *testing.T) {
	tests := []struct {
		name        string
		input       AnthropicResponse
		expected    func(t *testing.T, resp models.ChatCompletionResponse)
		expectError bool
	}{
		{
			name: "successful response",
			input: AnthropicResponse{
				ID:         "msg_123",
				Type:       "message",
				Role:       "assistant",
				Model:      "claude-3-5-sonnet",
				StopReason: "end_turn",
				Content: []AnthropicContentBlock{
					{Type: "text", Text: "Hello there!"},
				},
				Usage: AnthropicUsage{
					InputTokens:  10,
					OutputTokens: 5,
				},
			},
			expected: func(t *testing.T, resp models.ChatCompletionResponse) {
				if resp.ID != "msg_123" {
					t.Errorf("ID mismatch: got %q", resp.ID)
				}
				if resp.Choices[0].Message.Content != "Hello there!" {
					t.Errorf("Content mismatch: got %q", resp.Choices[0].Message.Content)
				}
				if resp.Usage.TotalTokens != 15 {
					t.Errorf("TotalTokens mismatch: expected 15, got %d", resp.Usage.TotalTokens)
				}
			},
		},
		{
			name: "multiple content blocks",
			input: AnthropicResponse{
				ID:         "msg_456",
				Type:       "message",
				Role:       "assistant",
				Model:      "claude-3-5-sonnet",
				StopReason: "end_turn",
				Content: []AnthropicContentBlock{
					{Type: "text", Text: "First part. "},
					{Type: "text", Text: "Second part."},
				},
				Usage: AnthropicUsage{
					InputTokens:  10,
					OutputTokens: 10,
				},
			},
			expected: func(t *testing.T, resp models.ChatCompletionResponse) {
				expectedContent := "First part. Second part."
				if resp.Choices[0].Message.Content != expectedContent {
					t.Errorf("Content mismatch: expected %q, got %q", expectedContent, resp.Choices[0].Message.Content)
				}
			},
		},
		{
			name: "error response type",
			input: AnthropicResponse{
				Type: "error",
			},
			expectError: true,
		},
	}

	transformer := NewResponseTransformer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := transformer.Transform(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			tt.expected(t, result)
		})
	}
}

func TestMapStopReason(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"end_turn", "stop"},
		{"max_tokens", "length"},
		{"stop_sequence", "stop"},
		{"unknown_reason", "stop"},
		{"", "stop"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapStopReason(tt.input)
			if result != tt.expected {
				t.Errorf("mapStopReason(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTransformErrorResponse(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		expectError bool
		checkError  func(t *testing.T, err error)
	}{
		{
			name:        "valid error",
			body:        `{"type":"error","error":{"type":"invalid_request_error","message":"max_tokens is required"}}`,
			expectError: true,
			checkError: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("Expected error")
				}
				if !strings.Contains(err.Error(), "invalid_request_error") {
					t.Errorf("Error should contain error type: %v", err)
				}
				if !strings.Contains(err.Error(), "max_tokens") {
					t.Errorf("Error should contain message: %v", err)
				}
			},
		},
		{
			name:        "invalid json",
			body:        `not json`,
			expectError: true,
			checkError: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("Expected error")
				}
				if !strings.Contains(err.Error(), "anthropic error") {
					t.Errorf("Error should indicate anthropic error: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := TransformErrorResponse([]byte(tt.body))

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}

			if tt.checkError != nil {
				tt.checkError(t, err)
			}
		})
	}
}

func TestParseSSEEvent(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		expectedType string
		expectedData string
	}{
		{
			name:         "event line",
			line:         "event: message_start",
			expectedType: "message_start",
			expectedData: "",
		},
		{
			name:         "data line",
			line:         "data: {\"type\":\"test\"}",
			expectedType: "",
			expectedData: `{"type":"test"}`,
		},
		{
			name:         "empty line",
			line:         "",
			expectedType: "",
			expectedData: "",
		},
		{
			name:         "whitespace only",
			line:         "   ",
			expectedType: "",
			expectedData: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventType, data := ParseSSEEvent(tt.line)
			if eventType != tt.expectedType {
				t.Errorf("Event type mismatch: expected %q, got %q", tt.expectedType, eventType)
			}
			if data != tt.expectedData {
				t.Errorf("Data mismatch: expected %q, got %q", tt.expectedData, data)
			}
		})
	}
}

func TestIsDoneMarker(t *testing.T) {
	tests := []struct {
		name     string
		chunk    []byte
		expected bool
	}{
		{
			name:     "message_stop event",
			chunk:    []byte("event: message_stop"),
			expected: true,
		},
		{
			name:     "DONE marker",
			chunk:    []byte("data: [DONE]"),
			expected: true,
		},
		{
			name:     "regular content",
			chunk:    []byte("data: {\"type\":\"text\"}"),
			expected: false,
		},
		{
			name:     "empty chunk",
			chunk:    []byte(""),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDoneMarker(tt.chunk)
			if result != tt.expected {
				t.Errorf("IsDoneMarker() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFormatSSEData(t *testing.T) {
	input := []byte(`{"type":"test"}`)
	result := formatSSEData(input)
	expected := "data: {\"type\":\"test\"}\n\n"
	if string(result) != expected {
		t.Errorf("formatSSEData() = %q, expected %q", string(result), expected)
	}
}

func TestStreamTransformer_GetAccumulatedContent(t *testing.T) {
	transformer := NewStreamTransformer()
	transformer.contentBuffer.WriteString("Hello World")

	content := transformer.GetAccumulatedContent()
	if content != "Hello World" {
		t.Errorf("GetAccumulatedContent() = %q, expected %q", content, "Hello World")
	}
}

func TestStreamTransformer_Reset(t *testing.T) {
	transformer := NewStreamTransformer()
	transformer.messageID = "msg_123"
	transformer.model = "claude-3"
	transformer.created = 12345
	transformer.contentBuffer.WriteString("content")

	transformer.Reset()

	if transformer.messageID != "" {
		t.Error("messageID should be empty after reset")
	}
	if transformer.model != "" {
		t.Error("model should be empty after reset")
	}
	if transformer.created != 0 {
		t.Error("created should be 0 after reset")
	}
	if transformer.contentBuffer.Len() != 0 {
		t.Error("contentBuffer should be empty after reset")
	}
}

func TestAnthropicError_Error(t *testing.T) {
	err := &AnthropicError{
		Type: "error",
		ErrorInfo: AnthropicErrorInfo{
			Type:    "invalid_request_error",
			Message: "max_tokens is required",
		},
	}

	expected := "anthropic invalid_request_error: max_tokens is required"
	if err.Error() != expected {
		t.Errorf("Error() = %q, expected %q", err.Error(), expected)
	}
}

func TestStreamingResponse_Read(t *testing.T) {
	pr, pw := io.Pipe()

	usage := models.Usage{}
	var mu sync.Mutex

	resp := &StreamingResponse{
		Reader: pr,
		usage:  &usage,
		mu:     &mu,
	}

	// Write data in a goroutine
	go func() {
		pw.Write([]byte("test data"))
		pw.Close()
	}()

	// Read data
	buf := make([]byte, 1024)
	n, err := resp.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(buf[:n]) != "test data" {
		t.Errorf("Read content mismatch: got %q", string(buf[:n]))
	}
}

func TestStreamingResponse_Close(t *testing.T) {
	pr, pw := io.Pipe()
	defer pr.Close()

	usage := models.Usage{}
	var mu sync.Mutex

	resp := &StreamingResponse{
		Reader: pr,
		usage:  &usage,
		mu:     &mu,
	}

	// Close should not error
	if err := resp.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Writing to closed pipe should error
	if _, err := pw.Write([]byte("test")); err == nil {
		t.Error("Expected error writing to closed pipe")
	}
}

func TestParseStreamChunk(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		expectError bool
		check       func(t *testing.T, chunk *models.ChatCompletionResponse)
	}{
		{
			name: "valid chunk",
			data: `{"id":"msg_123","object":"chat.completion.chunk","model":"claude-3","choices":[{"index":0,"message":{"role":"assistant","content":"Hello"}}]}`,
			check: func(t *testing.T, chunk *models.ChatCompletionResponse) {
				if chunk.ID != "msg_123" {
					t.Errorf("ID mismatch: got %q", chunk.ID)
				}
			},
		},
		{
			name:        "invalid json",
			data:        `not valid json`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk, err := ParseStreamChunk(tt.data)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.check != nil {
				tt.check(t, chunk)
			}
		})
	}
}

func TestExecuteChat_StreamEventProcessing(t *testing.T) {
	// Test the actual SSE event processing in executeStreaming
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send events with actual SSE formatting
		events := []struct {
			event string
			data  string
		}{
			{"message_start", `{"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-5-sonnet"}}`},
			{"content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text"}}`},
			{"content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Test"}}`},
			{"content_block_stop", `{"type":"content_block_stop","index":0}`},
			{"message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn"}}`},
			{"message_stop", `{"type":"message_stop"}`},
		}

		for _, e := range events {
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", e.event, e.data)
			w.(http.Flusher).Flush()
		}
	}))
	defer server.Close()

	adapter := NewAdapter("test-key", WithBaseURL(server.URL))
	result, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Payload: models.ChatCompletionRequest{
			Model: "claude-3-5-sonnet",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
			Stream: true,
		},
	}, "claude-3-5-sonnet")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	streamResp, ok := result.Payload.(*StreamingResponse)
	if !ok {
		t.Fatalf("Expected StreamingResponse, got %T", result.Payload)
	}
	defer streamResp.Close()

	// Read the stream with a scanner to handle SSE format
	scanner := bufio.NewScanner(streamResp.Reader)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Verify we got some data lines
	if len(lines) == 0 {
		t.Error("Expected non-empty stream")
	}
}
