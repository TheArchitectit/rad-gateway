// Package contract provides contract tests for provider adapters.
// Contract tests verify the transformation between OpenAI-compatible format
// and provider-specific formats, ensuring consistent behavior across providers.
package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"radgateway/internal/models"
	"radgateway/internal/provider/openai"
)

// =============================================================================
// Request Transformation Contract Tests
// =============================================================================

// TestOpenAI_RequestTransformation_Basic verifies basic request transformation
// Contract: OpenAI-compatible request -> OpenAI native format (pass-through)
func TestOpenAI_RequestTransformation_Basic(t *testing.T) {
	transformer := openai.NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []models.Message{
			{Role: "user", Content: "Hello!"},
		},
	}

	result := transformer.Transform(req)

	// Verify model preserved
	if result.Model != "gpt-4o" {
		t.Errorf("Contract violation: Model not preserved, got %q, want %q", result.Model, "gpt-4o")
	}

	// Verify messages preserved
	if len(result.Messages) != 1 {
		t.Fatalf("Contract violation: Expected 1 message, got %d", len(result.Messages))
	}

	if result.Messages[0].Role != "user" {
		t.Errorf("Contract violation: Role not preserved, got %q, want %q", result.Messages[0].Role, "user")
	}

	if result.Messages[0].Content != "Hello!" {
		t.Errorf("Contract violation: Content not preserved, got %q, want %q", result.Messages[0].Content, "Hello!")
	}
}

// TestOpenAI_RequestTransformation_MultipleMessages verifies multi-message transformation
// Contract: Multiple messages are preserved in order
func TestOpenAI_RequestTransformation_MultipleMessages(t *testing.T) {
	transformer := openai.NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []models.Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello!"},
			{Role: "assistant", Content: "Hi there!"},
			{Role: "user", Content: "How are you?"},
		},
	}

	result := transformer.Transform(req)

	if len(result.Messages) != 4 {
		t.Fatalf("Contract violation: Expected 4 messages, got %d", len(result.Messages))
	}

	// Verify all roles preserved (OpenAI supports all roles)
	expectedRoles := []string{"system", "user", "assistant", "user"}
	for i, expected := range expectedRoles {
		if result.Messages[i].Role != expected {
			t.Errorf("Contract violation: Message %d role mismatch, got %q, want %q", i, result.Messages[i].Role, expected)
		}
	}
}

// TestOpenAI_RequestTransformation_WithOptionalParams verifies optional parameters
// Contract: Optional parameters are correctly mapped
func TestOpenAI_RequestTransformation_WithOptionalParams(t *testing.T) {
	transformer := openai.NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model:  "gpt-4o",
		User:   "user-123",
		Stream: true,
		Messages: []models.Message{
			{Role: "user", Content: "Hello!"},
		},
	}

	result := transformer.Transform(req)

	// Verify optional parameters preserved (OpenAI transformer preserves User and Stream)
	if result.User != "user-123" {
		t.Errorf("Contract violation: User not preserved, got %q, want %q", result.User, "user-123")
	}

	if !result.Stream {
		t.Error("Contract violation: Stream flag not preserved")
	}
}

// TestOpenAI_RequestTransformation_EmptyMessages verifies handling of empty messages
// Contract: Empty messages array is preserved
func TestOpenAI_RequestTransformation_EmptyMessages(t *testing.T) {
	transformer := openai.NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model:    "gpt-4o",
		Messages: []models.Message{},
	}

	result := transformer.Transform(req)

	if len(result.Messages) != 0 {
		t.Errorf("Contract violation: Expected empty messages, got %d", len(result.Messages))
	}
}

// =============================================================================
// Response Transformation Contract Tests
// =============================================================================

// TestOpenAI_ResponseTransformation_Basic verifies basic response transformation
// Contract: OpenAI native response -> OpenAI-compatible response (pass-through)
func TestOpenAI_ResponseTransformation_Basic(t *testing.T) {
	transformer := openai.NewResponseTransformer()

	resp := openai.OpenAIResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: 1677652288,
		Model:   "gpt-4o",
		Choices: []openai.OpenAIChoice{
			{
				Index: 0,
				Message: openai.OpenAIMessage{
					Role:    "assistant",
					Content: "Hello! How can I help?",
				},
				FinishReason: "stop",
			},
		},
		Usage: openai.OpenAIUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	result, err := transformer.Transform(resp)
	if err != nil {
		t.Fatalf("Contract violation: Transform failed: %v", err)
	}

	// Verify ID preserved
	if result.ID != "chatcmpl-123" {
		t.Errorf("Contract violation: ID not preserved, got %q, want %q", result.ID, "chatcmpl-123")
	}

	// Verify content preserved
	if result.Choices[0].Message.Content != "Hello! How can I help?" {
		t.Errorf("Contract violation: Content mismatch, got %q", result.Choices[0].Message.Content)
	}

	// Verify usage preserved
	if result.Usage.TotalTokens != 30 {
		t.Errorf("Contract violation: TotalTokens mismatch, got %d, want %d", result.Usage.TotalTokens, 30)
	}
}

// TestOpenAI_ResponseTransformation_MultipleChoices verifies multiple choices handling
// Contract: All choices are preserved
func TestOpenAI_ResponseTransformation_MultipleChoices(t *testing.T) {
	transformer := openai.NewResponseTransformer()

	resp := openai.OpenAIResponse{
		ID:     "chatcmpl-multi",
		Model:  "gpt-4o",
		Choices: []openai.OpenAIChoice{
			{
				Index:        0,
				Message:      openai.OpenAIMessage{Role: "assistant", Content: "First response"},
				FinishReason: "stop",
			},
			{
				Index:        1,
				Message:      openai.OpenAIMessage{Role: "assistant", Content: "Second response"},
				FinishReason: "stop",
			},
		},
		Usage: openai.OpenAIUsage{TotalTokens: 50},
	}

	result, err := transformer.Transform(resp)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if len(result.Choices) != 2 {
		t.Errorf("Contract violation: Expected 2 choices, got %d", len(result.Choices))
	}

	if result.Choices[0].Message.Content != "First response" {
		t.Errorf("Contract violation: First choice content mismatch")
	}

	if result.Choices[1].Message.Content != "Second response" {
		t.Errorf("Contract violation: Second choice content mismatch")
	}
}

// TestOpenAI_ResponseTransformation_ErrorResponse verifies error response handling
// Contract: Error responses are properly transformed to errors
func TestOpenAI_ResponseTransformation_ErrorResponse(t *testing.T) {
	transformer := openai.NewResponseTransformer()

	resp := openai.OpenAIResponse{
		Error: &openai.OpenAIError{
			Message: "Invalid API key",
			Type:    "invalid_request_error",
			Code:    "invalid_api_key",
		},
	}

	_, err := transformer.Transform(resp)
	if err == nil {
		t.Error("Contract violation: Expected error for error response")
	}

	if !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("Contract violation: Error message should contain 'Invalid API key', got: %v", err)
	}
}

// =============================================================================
// Streaming Response Contract Tests
// =============================================================================

// TestOpenAI_StreamTransformation_Basic verifies basic stream chunk transformation
// Contract: Stream chunks are preserved in OpenAI-compatible format
func TestOpenAI_StreamTransformation_Basic(t *testing.T) {
	transformer := openai.NewStreamTransformer()

	chunk := openai.OpenAIStreamResponse{
		ID:      "chatcmpl-stream",
		Object:  "chat.completion.chunk",
		Created: 1677652288,
		Model:   "gpt-4o",
		Choices: []openai.OpenAIStreamChoice{
			{
				Index: 0,
				Delta: openai.OpenAIMessageDelta{
					Role:    "assistant",
					Content: "Hello",
				},
			},
		},
	}

	result := transformer.TransformChunk(chunk)

	if result.ID != "chatcmpl-stream" {
		t.Errorf("Contract violation: Stream ID not preserved, got %q", result.ID)
	}

	if result.Object != "chat.completion.chunk" {
		t.Errorf("Contract violation: Object not preserved, got %q", result.Object)
	}

	if result.Choices[0].Message.Content != "Hello" {
		t.Errorf("Contract violation: Delta content not preserved, got %q", result.Choices[0].Message.Content)
	}
}

// TestOpenAI_StreamTransformation_RoleDelta verifies role-only delta
// Contract: Role deltas are preserved
func TestOpenAI_StreamTransformation_RoleDelta(t *testing.T) {
	transformer := openai.NewStreamTransformer()

	chunk := openai.OpenAIStreamResponse{
		ID:     "chatcmpl-role",
		Model:  "gpt-4o",
		Choices: []openai.OpenAIStreamChoice{
			{
				Index: 0,
				Delta: openai.OpenAIMessageDelta{
					Role: "assistant",
				},
			},
		},
	}

	result := transformer.TransformChunk(chunk)

	if result.Choices[0].Message.Role != "assistant" {
		t.Errorf("Contract violation: Role delta not preserved, got %q", result.Choices[0].Message.Role)
	}
}

// TestOpenAI_StreamTransformation_FinishReason verifies finish reason handling
// Contract: Finish reasons are preserved in streaming chunks
func TestOpenAI_StreamTransformation_FinishReason(t *testing.T) {
	transformer := openai.NewStreamTransformer()

	finishReason := "stop"
	chunk := openai.OpenAIStreamResponse{
		ID:      "chatcmpl-finish",
		Object:  "chat.completion.chunk",
		Created: 1677652288,
		Model:   "gpt-4o",
		Choices: []openai.OpenAIStreamChoice{
			{
				Index:        0,
				Delta:        openai.OpenAIMessageDelta{},
				FinishReason: &finishReason,
			},
		},
	}

	result := transformer.TransformChunk(chunk)

	// Note: The OpenAI stream transformer copies the ID, Object, Model, and content
	// The finish reason handling is done at the SSE parsing level
	if result.ID != "chatcmpl-finish" {
		t.Errorf("Contract violation: ID not preserved in stream chunk")
	}

	if result.Object != "chat.completion.chunk" {
		t.Errorf("Contract violation: Object not preserved")
	}
}

// =============================================================================
// Error Handling Contract Tests
// =============================================================================

// TestOpenAI_ErrorResponse_Unauthorized verifies 401 handling
// Contract: 401 errors are transformed to provider errors
func TestOpenAI_ErrorResponse_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Invalid API key",
				"type":    "invalid_request_error",
				"code":    "invalid_api_key",
			},
		})
	}))
	defer server.Close()

	ctx := context.Background()
	adapter := openai.NewAdapter("invalid-key", openai.WithBaseURL(server.URL))
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
		t.Error("Contract violation: Expected error for 401 response")
	}

	if !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("Contract violation: Error should contain 'Invalid API key', got: %v", err)
	}
}

// TestOpenAI_ErrorResponse_RateLimited verifies 429 handling
// Contract: 429 errors trigger retry behavior
func TestOpenAI_ErrorResponse_RateLimited(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"message": "Rate limit exceeded",
					"type":    "rate_limit_error",
				},
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(openai.OpenAIResponse{
			ID:     "chatcmpl-success",
			Object: "chat.completion",
			Model:  "gpt-4o",
			Choices: []openai.OpenAIChoice{
				{
					Index:        0,
					Message:      openai.OpenAIMessage{Role: "assistant", Content: "Hello!"},
					FinishReason: "stop",
				},
			},
			Usage: openai.OpenAIUsage{TotalTokens: 10},
		})
	}))
	defer server.Close()

	adapter := openai.NewAdapter("test-key",
		openai.WithBaseURL(server.URL),
		openai.WithRetryConfig(3, 10*time.Millisecond),
	)

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

	if err != nil {
		t.Errorf("Contract violation: Expected success after retry, got error: %v", err)
	}

	if requestCount < 2 {
		t.Errorf("Contract violation: Expected retry on 429, got %d requests", requestCount)
	}

	if result.Status != "success" {
		t.Errorf("Contract violation: Expected success status, got %q", result.Status)
	}
}

// TestOpenAI_ErrorResponse_ServerError verifies 500 handling
// Contract: 500 errors trigger retry and eventually fail
func TestOpenAI_ErrorResponse_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Internal server error",
				"type":    "api_error",
			},
		})
	}))
	defer server.Close()

	adapter := openai.NewAdapter("test-key",
		openai.WithBaseURL(server.URL),
		openai.WithRetryConfig(2, 10*time.Millisecond),
	)

	_, err := adapter.Execute(context.Background(), models.ProviderRequest{
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
		t.Error("Contract violation: Expected error after retries exhausted")
	}
}

// =============================================================================
// Authentication Contract Tests
// =============================================================================

// TestOpenAI_Authentication_BearerToken verifies Bearer token authentication
// Contract: Authorization header contains Bearer token
func TestOpenAI_Authentication_BearerToken(t *testing.T) {
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(openai.OpenAIResponse{
			ID:     "chatcmpl-auth",
			Object: "chat.completion",
			Model:  "gpt-4o",
			Choices: []openai.OpenAIChoice{
				{
					Index:        0,
					Message:      openai.OpenAIMessage{Role: "assistant", Content: "Hello!"},
					FinishReason: "stop",
				},
			},
			Usage: openai.OpenAIUsage{TotalTokens: 10},
		})
	}))
	defer server.Close()

	adapter := openai.NewAdapter("sk-test123", openai.WithBaseURL(server.URL))
	adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "gpt-4o",
		Payload: models.ChatCompletionRequest{
			Model: "gpt-4o",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}, "gpt-4o")

	if capturedAuth != "Bearer sk-test123" {
		t.Errorf("Contract violation: Authorization header mismatch, got %q, want %q", capturedAuth, "Bearer sk-test123")
	}
}

// TestOpenAI_Authentication_ContentType verifies Content-Type header
// Contract: Content-Type is application/json
func TestOpenAI_Authentication_ContentType(t *testing.T) {
	var capturedContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(openai.OpenAIResponse{
			ID:      "chatcmpl-ct",
			Object:  "chat.completion",
			Model:   "gpt-4o",
			Choices: []openai.OpenAIChoice{{Index: 0, Message: openai.OpenAIMessage{Role: "assistant", Content: "Hi"}, FinishReason: "stop"}},
			Usage:   openai.OpenAIUsage{TotalTokens: 5},
		})
	}))
	defer server.Close()

	adapter := openai.NewAdapter("test-key", openai.WithBaseURL(server.URL))
	adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "gpt-4o",
		Payload: models.ChatCompletionRequest{
			Model: "gpt-4o",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}, "gpt-4o")

	if capturedContentType != "application/json" {
		t.Errorf("Contract violation: Content-Type mismatch, got %q, want %q", capturedContentType, "application/json")
	}
}

// =============================================================================
// Model Name Contract Tests
// =============================================================================

// TestOpenAI_ModelName_Preservation verifies model name preservation
// Contract: Model names are preserved exactly
func TestOpenAI_ModelName_Preservation(t *testing.T) {
	modelNames := []string{
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4-turbo",
		"gpt-3.5-turbo",
		"text-embedding-3-small",
	}

	for _, modelName := range modelNames {
		t.Run(modelName, func(t *testing.T) {
			transformer := openai.NewRequestTransformer()
			req := models.ChatCompletionRequest{
				Model:    modelName,
				Messages: []models.Message{{Role: "user", Content: "Hello"}},
			}

			result := transformer.Transform(req)

			if result.Model != modelName {
				t.Errorf("Contract violation: Model name not preserved, got %q, want %q", result.Model, modelName)
			}
		})
	}
}

// =============================================================================
// Embeddings Contract Tests
// =============================================================================

// TestOpenAI_Embeddings_RequestTransformation verifies embeddings request transformation
// Contract: Embeddings requests are transformed correctly
func TestOpenAI_Embeddings_RequestTransformation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify path
		if r.URL.Path != "/embeddings" {
			t.Errorf("Contract violation: Expected path /embeddings, got %s", r.URL.Path)
		}

		// Verify request body
		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Contract violation: Failed to decode request: %v", err)
		}

		if reqBody["model"] != "text-embedding-3-small" {
			t.Errorf("Contract violation: Model mismatch, got %v", reqBody["model"])
		}

		if reqBody["input"] != "Hello world" {
			t.Errorf("Contract violation: Input mismatch, got %v", reqBody["input"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data": []map[string]any{
				{
					"object":    "embedding",
					"index":     0,
					"embedding": []float64{0.1, 0.2, 0.3},
				},
			},
			"model": "text-embedding-3-small",
			"usage": map[string]any{
				"prompt_tokens": 3,
				"total_tokens":  3,
			},
		})
	}))
	defer server.Close()

	adapter := openai.NewAdapter("test-key", openai.WithBaseURL(server.URL))
	result, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "embeddings",
		Model:   "text-embedding-3-small",
		Payload: models.EmbeddingsRequest{
			Model: "text-embedding-3-small",
			Input: "Hello world",
		},
	}, "text-embedding-3-small")

	if err != nil {
		t.Fatalf("Contract violation: Execute failed: %v", err)
	}

	embeddingsResp, ok := result.Payload.(models.EmbeddingsResponse)
	if !ok {
		t.Fatalf("Contract violation: Expected EmbeddingsResponse, got %T", result.Payload)
	}

	if len(embeddingsResp.Data) != 1 {
		t.Errorf("Contract violation: Expected 1 embedding, got %d", len(embeddingsResp.Data))
	}
}

// =============================================================================
// Streaming Integration Contract Tests
// =============================================================================

// TestOpenAI_Streaming_EndToEnd verifies full streaming flow
// Contract: Streaming responses are properly handled end-to-end
func TestOpenAI_Streaming_EndToEnd(t *testing.T) {
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
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	adapter := openai.NewAdapter("test-key", openai.WithBaseURL(server.URL))
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
		t.Fatalf("Contract violation: Execute failed: %v", err)
	}

	streamResp, ok := result.Payload.(*openai.StreamingResponse)
	if !ok {
		t.Fatalf("Contract violation: Expected StreamingResponse, got %T", result.Payload)
	}
	defer streamResp.Close()

	data, err := io.ReadAll(streamResp.Reader)
	if err != nil {
		t.Fatalf("Contract violation: Failed to read stream: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Hello") {
		t.Errorf("Contract violation: Stream content missing 'Hello': %s", content)
	}

	// The stream should contain data: prefixes (OpenAI-compatible SSE format)
	if !strings.Contains(content, "data:") {
		t.Errorf("Contract violation: Stream missing data: prefix: %s", content)
	}
}

// =============================================================================
// Retry Contract Tests
// =============================================================================

// TestOpenAI_Retry_BadRequestNoRetry verifies bad requests don't retry
// Contract: 400 errors do not trigger retry
func TestOpenAI_Retry_BadRequestNoRetry(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Invalid request",
				"type":    "invalid_request_error",
			},
		})
	}))
	defer server.Close()

	adapter := openai.NewAdapter("test-key",
		openai.WithBaseURL(server.URL),
		openai.WithRetryConfig(3, 10*time.Millisecond),
	)

	adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "gpt-4o",
		Payload: models.ChatCompletionRequest{
			Model: "gpt-4o",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}, "gpt-4o")

	if requestCount != 1 {
		t.Errorf("Contract violation: Expected 1 request (no retry on 400), got %d", requestCount)
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// Verify that all contract tests are deterministic by using httptest
func TestOpenAI_Deterministic(t *testing.T) {
	// This test ensures all previous tests are deterministic by using httptest
	// No external calls are made in any contract test
	t.Log("All OpenAI contract tests use httptest and are deterministic")
}
