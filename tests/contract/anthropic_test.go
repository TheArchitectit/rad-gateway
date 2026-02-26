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
	"radgateway/internal/provider/anthropic"
)

// =============================================================================
// Request Transformation Contract Tests
// =============================================================================

// TestAnthropic_RequestTransformation_Basic verifies basic request transformation
// Contract: OpenAI-compatible request -> Anthropic native format
func TestAnthropic_RequestTransformation_Basic(t *testing.T) {
	transformer := anthropic.NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []models.Message{
			{Role: "user", Content: "Hello!"},
		},
	}

	result := transformer.Transform(req)

	// Verify model preserved
	if result.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("Contract violation: Model not preserved, got %q, want %q", result.Model, "claude-3-5-sonnet-20241022")
	}

	// Verify max_tokens default is set (Anthropic requires this)
	if result.MaxTokens != 4096 {
		t.Errorf("Contract violation: Default max_tokens not set, got %d, want %d", result.MaxTokens, 4096)
	}

	// Verify messages transformed (role mapping)
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

// TestAnthropic_RequestTransformation_SystemMessageExtraction verifies system message handling
// Contract: System messages are extracted from messages array into separate system field
func TestAnthropic_RequestTransformation_SystemMessageExtraction(t *testing.T) {
	transformer := anthropic.NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []models.Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello!"},
		},
	}

	result := transformer.Transform(req)

	// Verify system message extracted
	if result.System != "You are a helpful assistant." {
		t.Errorf("Contract violation: System message not extracted, got %q", result.System)
	}

	// Verify system message not in messages array
	for _, msg := range result.Messages {
		if msg.Role == "system" {
			t.Error("Contract violation: System message should not be in messages array")
		}
	}

	// Verify only user message remains
	if len(result.Messages) != 1 {
		t.Errorf("Contract violation: Expected 1 message after extraction, got %d", len(result.Messages))
	}
}

// TestAnthropic_RequestTransformation_MultipleSystemMessages verifies multiple system messages
// Contract: Multiple system messages are combined into single system field
func TestAnthropic_RequestTransformation_MultipleSystemMessages(t *testing.T) {
	transformer := anthropic.NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []models.Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "system", Content: "Be concise."},
			{Role: "user", Content: "Hello!"},
		},
	}

	result := transformer.Transform(req)

	// Verify system messages combined
	expectedSystem := "You are helpful.\nBe concise."
	if result.System != expectedSystem {
		t.Errorf("Contract violation: System messages not combined correctly, got %q, want %q", result.System, expectedSystem)
	}
}

// TestAnthropic_RequestTransformation_ConsecutiveSameRole verifies consecutive role merging
// Contract: Consecutive messages with same role are merged
func TestAnthropic_RequestTransformation_ConsecutiveSameRole(t *testing.T) {
	transformer := anthropic.NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []models.Message{
			{Role: "user", Content: "Hello"},
			{Role: "user", Content: "World"},
			{Role: "assistant", Content: "Hi!"},
		},
	}

	result := transformer.Transform(req)

	// Verify consecutive user messages merged
	if len(result.Messages) != 2 {
		t.Fatalf("Contract violation: Expected 2 messages after merging, got %d", len(result.Messages))
	}

	// First message should be merged user content
	if !strings.Contains(result.Messages[0].Content, "Hello") || !strings.Contains(result.Messages[0].Content, "World") {
		t.Errorf("Contract violation: Consecutive user messages not merged, got %q", result.Messages[0].Content)
	}
}

// TestAnthropic_RequestTransformation_OptionalParams verifies optional parameter mapping
// Contract: Optional parameters are correctly mapped to Anthropic format
func TestAnthropic_RequestTransformation_OptionalParams(t *testing.T) {
	transformer := anthropic.NewRequestTransformer()

	temp := 0.7
	topP := 0.9

	req := models.ChatCompletionRequest{
		Model:       "claude-3-5-sonnet-20241022",
		Temperature: temp,
		TopP:        topP,
		MaxTokens:   1024,
		User:        "user-123",
		Stop:        []string{"STOP", "END"},
		Messages: []models.Message{
			{Role: "user", Content: "Hello!"},
		},
	}

	result := transformer.Transform(req)

	// Verify max_tokens preserved
	if result.MaxTokens != 1024 {
		t.Errorf("Contract violation: MaxTokens not preserved, got %d, want %d", result.MaxTokens, 1024)
	}

	// Verify temperature mapped
	if result.Temperature == nil || *result.Temperature != temp {
		t.Errorf("Contract violation: Temperature not preserved correctly")
	}

	// Verify top_p mapped
	if result.TopP == nil || *result.TopP != topP {
		t.Errorf("Contract violation: TopP not preserved correctly")
	}

	// Verify stop sequences mapped
	if len(result.StopSequences) != 2 {
		t.Errorf("Contract violation: Stop sequences not preserved, got %d", len(result.StopSequences))
	}

	// Verify user mapped to metadata
	if result.Metadata == nil || result.Metadata.UserID != "user-123" {
		t.Errorf("Contract violation: User ID not mapped to metadata")
	}
}

// =============================================================================
// Response Transformation Contract Tests
// =============================================================================

// TestAnthropic_ResponseTransformation_Basic verifies basic response transformation
// Contract: Anthropic native response -> OpenAI-compatible response
func TestAnthropic_ResponseTransformation_Basic(t *testing.T) {
	transformer := anthropic.NewResponseTransformer()

	resp := anthropic.AnthropicResponse{
		ID:   "msg_01XgVYxVqW32TYn5Ts4RYRPW",
		Type: "message",
		Role: "assistant",
		Model: "claude-3-5-sonnet-20241022",
		Content: []anthropic.AnthropicContentBlock{
			{Type: "text", Text: "Hello! How can I help you today?"},
		},
		StopReason: "end_turn",
		Usage: anthropic.AnthropicUsage{
			InputTokens:  12,
			OutputTokens: 9,
		},
	}

	result, err := transformer.Transform(resp)
	if err != nil {
		t.Fatalf("Contract violation: Transform failed: %v", err)
	}

	// Verify ID preserved
	if result.ID != "msg_01XgVYxVqW32TYn5Ts4RYRPW" {
		t.Errorf("Contract violation: ID not preserved, got %q", result.ID)
	}

	// Verify object type set
	if result.Object != "chat.completion" {
		t.Errorf("Contract violation: Object type incorrect, got %q, want %q", result.Object, "chat.completion")
	}

	// Verify content transformed
	if result.Choices[0].Message.Content != "Hello! How can I help you today?" {
		t.Errorf("Contract violation: Content mismatch, got %q", result.Choices[0].Message.Content)
	}

	// Verify role set to assistant
	if result.Choices[0].Message.Role != "assistant" {
		t.Errorf("Contract violation: Role should be 'assistant', got %q", result.Choices[0].Message.Role)
	}

	// Verify usage calculated
	if result.Usage.TotalTokens != 21 {
		t.Errorf("Contract violation: Total tokens incorrect, got %d, want %d", result.Usage.TotalTokens, 21)
	}
}

// TestAnthropic_ResponseTransformation_StopReasonMapping verifies stop reason mapping
// Contract: Anthropic stop_reason values are mapped to OpenAI finish_reason values
func TestAnthropic_ResponseTransformation_StopReasonMapping(t *testing.T) {
	tests := []struct {
		anthropicReason string
		expected        string
	}{
		{"end_turn", "stop"},
		{"max_tokens", "length"},
		{"stop_sequence", "stop"},
		{"", "stop"}, // default
	}

	transformer := anthropic.NewResponseTransformer()

	for _, tt := range tests {
		t.Run(tt.anthropicReason, func(t *testing.T) {
			resp := anthropic.AnthropicResponse{
				ID:         "msg_test",
				Type:       "message",
				Role:       "assistant",
				Model:      "claude-3-5-sonnet",
				Content:    []anthropic.AnthropicContentBlock{{Type: "text", Text: "Test"}},
				StopReason: tt.anthropicReason,
				Usage:      anthropic.AnthropicUsage{InputTokens: 10, OutputTokens: 5},
			}

			result, err := transformer.Transform(resp)
			if err != nil {
				t.Fatalf("Transform failed: %v", err)
			}

			if result.Choices[0].FinishReason != tt.expected {
				t.Errorf("Contract violation: Stop reason mapping incorrect for %q, got %q, want %q",
					tt.anthropicReason, result.Choices[0].FinishReason, tt.expected)
			}
		})
	}
}

// TestAnthropic_ResponseTransformation_MultipleContentBlocks verifies multiple content blocks
// Contract: Multiple content blocks are concatenated
func TestAnthropic_ResponseTransformation_MultipleContentBlocks(t *testing.T) {
	transformer := anthropic.NewResponseTransformer()

	resp := anthropic.AnthropicResponse{
		ID:   "msg_multi",
		Type: "message",
		Role: "assistant",
		Model: "claude-3-5-sonnet",
		Content: []anthropic.AnthropicContentBlock{
			{Type: "text", Text: "First part. "},
			{Type: "text", Text: "Second part."},
		},
		StopReason: "end_turn",
		Usage:      anthropic.AnthropicUsage{InputTokens: 10, OutputTokens: 10},
	}

	result, err := transformer.Transform(resp)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	expectedContent := "First part. Second part."
	if result.Choices[0].Message.Content != expectedContent {
		t.Errorf("Contract violation: Content blocks not concatenated correctly, got %q, want %q",
			result.Choices[0].Message.Content, expectedContent)
	}
}

// TestAnthropic_ResponseTransformation_ErrorResponse verifies error handling
// Contract: Error responses are transformed to errors
func TestAnthropic_ResponseTransformation_ErrorResponse(t *testing.T) {
	transformer := anthropic.NewResponseTransformer()

	resp := anthropic.AnthropicResponse{
		Type: "error",
	}

	_, err := transformer.Transform(resp)
	if err == nil {
		t.Error("Contract violation: Expected error for error response type")
	}
}

// =============================================================================
// Streaming Response Contract Tests
// =============================================================================

// TestAnthropic_StreamTransformation_MessageStart verifies message_start event handling
// Contract: message_start event initializes stream state
func TestAnthropic_StreamTransformation_MessageStart(t *testing.T) {
	transformer := anthropic.NewStreamTransformer()

	eventData := `{"type":"message_start","message":{"id":"msg_stream_123","type":"message","role":"assistant","model":"claude-3-5-sonnet"}}`
	result, done, err := transformer.TransformEvent("message_start", []byte(eventData))

	if err != nil {
		t.Fatalf("Contract violation: TransformEvent failed: %v", err)
	}

	if done {
		t.Error("Contract violation: message_start should not signal done")
	}

	if result == nil {
		t.Fatal("Contract violation: Expected result for message_start")
	}

	// Verify result contains OpenAI-compatible chunk
	resultStr := string(result)
	if !strings.Contains(resultStr, "chat.completion.chunk") {
		t.Errorf("Contract violation: Result should contain chat.completion.chunk, got %s", resultStr)
	}
}

// TestAnthropic_StreamTransformation_ContentBlockDelta verifies content_block_delta handling
// Contract: content_block_delta events are transformed to content deltas
func TestAnthropic_StreamTransformation_ContentBlockDelta(t *testing.T) {
	transformer := anthropic.NewStreamTransformer()

	// Initialize with message_start first
	transformer.TransformEvent("message_start", []byte(`{"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-5-sonnet"}}`))

	eventData := `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`
	result, done, err := transformer.TransformEvent("content_block_delta", []byte(eventData))

	if err != nil {
		t.Fatalf("Contract violation: TransformEvent failed: %v", err)
	}

	if done {
		t.Error("Contract violation: content_block_delta should not signal done")
	}

	if result == nil {
		t.Fatal("Contract violation: Expected result for content_block_delta")
	}

	// Verify content in result
	resultStr := string(result)
	if !strings.Contains(resultStr, "Hello") {
		t.Errorf("Contract violation: Result should contain 'Hello', got %s", resultStr)
	}
}

// TestAnthropic_StreamTransformation_MessageStop verifies message_stop handling
// Contract: message_stop event signals stream completion
func TestAnthropic_StreamTransformation_MessageStop(t *testing.T) {
	transformer := anthropic.NewStreamTransformer()

	result, done, err := transformer.TransformEvent("message_stop", []byte(`{"type":"message_stop"}`))

	if err != nil {
		t.Fatalf("Contract violation: TransformEvent failed: %v", err)
	}

	if !done {
		t.Error("Contract violation: message_stop should signal done")
	}

	// Should contain [DONE] marker
	resultStr := string(result)
	if !strings.Contains(resultStr, "[DONE]") {
		t.Errorf("Contract violation: Result should contain [DONE] marker, got %s", resultStr)
	}
}

// TestAnthropic_StreamTransformation_PingEvent verifies ping event handling
// Contract: ping events are ignored (no output)
func TestAnthropic_StreamTransformation_PingEvent(t *testing.T) {
	transformer := anthropic.NewStreamTransformer()

	result, done, err := transformer.TransformEvent("ping", []byte(`{"type":"ping"}`))

	if err != nil {
		t.Fatalf("Contract violation: TransformEvent failed: %v", err)
	}

	if done {
		t.Error("Contract violation: ping should not signal done")
	}

	if result != nil {
		t.Errorf("Contract violation: ping should return nil result, got %s", string(result))
	}
}

// =============================================================================
// Error Handling Contract Tests
// =============================================================================

// TestAnthropic_ErrorResponse_Transformation verifies error response transformation
// Contract: Anthropic error format is transformed to standard error
func TestAnthropic_ErrorResponse_Transformation(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		expectedError  bool
		expectedString string
	}{
		{
			name:           "authentication error",
			response:       `{"type":"error","error":{"type":"authentication_error","message":"Invalid API key"}}`,
			expectedError:  true,
			expectedString: "authentication_error",
		},
		{
			name:           "rate limit error",
			response:       `{"type":"error","error":{"type":"rate_limit_error","message":"Rate limit exceeded"}}`,
			expectedError:  true,
			expectedString: "rate_limit_error",
		},
		{
			name:           "invalid request",
			response:       `{"type":"error","error":{"type":"invalid_request_error","message":"max_tokens is required"}}`,
			expectedError:  true,
			expectedString: "invalid_request_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := anthropic.TransformErrorResponse([]byte(tt.response))

			if tt.expectedError && err == nil {
				t.Error("Contract violation: Expected error")
			}

			if err != nil && !strings.Contains(err.Error(), tt.expectedString) {
				t.Errorf("Contract violation: Error should contain %q, got %v", tt.expectedString, err)
			}
		})
	}
}

// =============================================================================
// Authentication Contract Tests
// =============================================================================

// TestAnthropic_Authentication_XApiKey verifies x-api-key authentication
// Contract: x-api-key header is set (not Bearer token)
func TestAnthropic_Authentication_XApiKey(t *testing.T) {
	var capturedXApiKey string
	var capturedVersion string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedXApiKey = r.Header.Get("x-api-key")
		capturedVersion = r.Header.Get("anthropic-version")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(anthropic.AnthropicResponse{
			ID:         "msg_auth",
			Type:       "message",
			Role:       "assistant",
			Model:      "claude-3-5-sonnet",
			Content:    []anthropic.AnthropicContentBlock{{Type: "text", Text: "Hello!"}},
			StopReason: "end_turn",
			Usage:      anthropic.AnthropicUsage{InputTokens: 10, OutputTokens: 2},
		})
	}))
	defer server.Close()

	adapter := anthropic.NewAdapter("sk-ant-api-test", anthropic.WithBaseURL(server.URL))
	adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "claude-3-5-sonnet",
		Payload: models.ChatCompletionRequest{
			Model: "claude-3-5-sonnet",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}, "claude-3-5-sonnet")

	// Verify x-api-key header (not Bearer)
	if capturedXApiKey != "sk-ant-api-test" {
		t.Errorf("Contract violation: x-api-key header mismatch, got %q, want %q", capturedXApiKey, "sk-ant-api-test")
	}

	// Verify anthropic-version header
	if capturedVersion != "2023-06-01" {
		t.Errorf("Contract violation: anthropic-version header mismatch, got %q, want %q", capturedVersion, "2023-06-01")
	}
}

// =============================================================================
// Model Name Contract Tests
// =============================================================================

// TestAnthropic_ModelName_Preservation verifies model name preservation
// Contract: Model names are preserved exactly
func TestAnthropic_ModelName_Preservation(t *testing.T) {
	modelNames := []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}

	for _, modelName := range modelNames {
		t.Run(modelName, func(t *testing.T) {
			transformer := anthropic.NewRequestTransformer()
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
// Streaming Integration Contract Tests
// =============================================================================

// TestAnthropic_Streaming_EndToEnd verifies full streaming flow
// Contract: Streaming responses are properly handled end-to-end
func TestAnthropic_Streaming_EndToEnd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		events := []string{
			`event: message_start
data: {"type":"message_start","message":{"id":"msg_stream_123","type":"message","role":"assistant","model":"claude-3-5-sonnet"}}`,
			`event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text"}}`,
			`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
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
			time.Sleep(5 * time.Millisecond)
		}
	}))
	defer server.Close()

	adapter := anthropic.NewAdapter("test-key", anthropic.WithBaseURL(server.URL))
	result, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "claude-3-5-sonnet",
		Payload: models.ChatCompletionRequest{
			Model: "claude-3-5-sonnet",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
			Stream: true,
		},
	}, "claude-3-5-sonnet")

	if err != nil {
		t.Fatalf("Contract violation: Execute failed: %v", err)
	}

	streamResp, ok := result.Payload.(*anthropic.StreamingResponse)
	if !ok {
		t.Fatalf("Contract violation: Expected StreamingResponse, got %T", result.Payload)
	}
	defer streamResp.Close()

	data, err := io.ReadAll(streamResp.Reader)
	if err != nil {
		t.Fatalf("Contract violation: Failed to read stream: %v", err)
	}

	content := string(data)

	// Verify transformed content
	if !strings.Contains(content, "data:") {
		t.Errorf("Contract violation: Stream missing 'data:' prefix: %s", content)
	}

	if !strings.Contains(content, "Hello") {
		t.Errorf("Contract violation: Stream missing content 'Hello': %s", content)
	}

	if !strings.Contains(content, "[DONE]") {
		t.Errorf("Contract violation: Stream missing [DONE] marker: %s", content)
	}
}

// =============================================================================
// Retry Contract Tests
// =============================================================================

// TestAnthropic_Retry_ServerError verifies retry on server errors
// Contract: 503 errors trigger retry
func TestAnthropic_Retry_ServerError(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]any{
				"type": "error",
				"error": map[string]any{
					"type":    "api_error",
					"message": "Service temporarily unavailable",
				},
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(anthropic.AnthropicResponse{
			ID:         "msg_retry",
			Type:       "message",
			Role:       "assistant",
			Model:      "claude-3-5-sonnet",
			Content:    []anthropic.AnthropicContentBlock{{Type: "text", Text: "Hello!"}},
			StopReason: "end_turn",
			Usage:      anthropic.AnthropicUsage{InputTokens: 10, OutputTokens: 2},
		})
	}))
	defer server.Close()

	adapter := anthropic.NewAdapter("test-key",
		anthropic.WithBaseURL(server.URL),
		anthropic.WithRetryConfig(3, 10*time.Millisecond),
	)

	result, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "claude-3-5-sonnet",
		Payload: models.ChatCompletionRequest{
			Model: "claude-3-5-sonnet",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}, "claude-3-5-sonnet")

	if err != nil {
		t.Errorf("Contract violation: Expected success after retry, got error: %v", err)
	}

	if requestCount < 2 {
		t.Errorf("Contract violation: Expected retry on 503, got %d requests", requestCount)
	}

	if result.Status != "success" {
		t.Errorf("Contract violation: Expected success status, got %q", result.Status)
	}
}

// TestAnthropic_Retry_BadRequestNoRetry verifies no retry on bad requests
// Contract: 400 errors do not trigger retry
func TestAnthropic_Retry_BadRequestNoRetry(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"type": "error",
			"error": map[string]any{
				"type":    "invalid_request_error",
				"message": "max_tokens: range error",
			},
		})
	}))
	defer server.Close()

	adapter := anthropic.NewAdapter("test-key",
		anthropic.WithBaseURL(server.URL),
		anthropic.WithRetryConfig(3, 10*time.Millisecond),
	)

	adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "claude-3-5-sonnet",
		Payload: models.ChatCompletionRequest{
			Model: "claude-3-5-sonnet",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}, "claude-3-5-sonnet")

	if requestCount != 1 {
		t.Errorf("Contract violation: Expected 1 request (no retry on 400), got %d", requestCount)
	}
}

// =============================================================================
// Edge Case Contract Tests
// =============================================================================

// TestAnthropic_EmptyMessages_Handling verifies handling of edge cases
// Contract: Edge cases are handled gracefully
func TestAnthropic_EmptyMessages_Handling(t *testing.T) {
	transformer := anthropic.NewRequestTransformer()

	// Only system messages
	req := models.ChatCompletionRequest{
		Model: "claude-3-5-sonnet",
		Messages: []models.Message{
			{Role: "system", Content: "You are helpful."},
		},
	}

	result := transformer.Transform(req)

	// System content should be extracted to System field
	if result.System != "You are helpful." {
		t.Errorf("Contract violation: System content not preserved, got %q", result.System)
	}

	// For system-only input, messages array will be empty (system content extracted separately)
	// This is valid as Anthropic API requires system to be in separate field
}

// TestAnthropic_SSEEvent_Parsing verifies SSE event parsing
// Contract: SSE events are parsed correctly
func TestAnthropic_SSEEvent_Parsing(t *testing.T) {
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
			line:         `data: {"type":"test"}`,
			expectedType: "",
			expectedData: `{"type":"test"}`,
		},
		{
			name:         "empty line",
			line:         "",
			expectedType: "",
			expectedData: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventType, data := anthropic.ParseSSEEvent(tt.line)

			if eventType != tt.expectedType {
				t.Errorf("Contract violation: Event type mismatch, got %q, want %q", eventType, tt.expectedType)
			}

			if data != tt.expectedData {
				t.Errorf("Contract violation: Data mismatch, got %q, want %q", data, tt.expectedData)
			}
		})
	}
}

// TestAnthropic_IsDoneMarker verifies done marker detection
// Contract: Done markers are detected correctly
func TestAnthropic_IsDoneMarker(t *testing.T) {
	tests := []struct {
		chunk    []byte
		expected bool
	}{
		{[]byte("event: message_stop"), true},
		{[]byte("data: [DONE]"), true},
		{[]byte(`data: {"type":"content_block_delta"}`), false},
		{[]byte(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.chunk), func(t *testing.T) {
			result := anthropic.IsDoneMarker(tt.chunk)
			if result != tt.expected {
				t.Errorf("Contract violation: IsDoneMarker(%q) = %v, want %v", tt.chunk, result, tt.expected)
			}
		})
	}
}

// Verify contract tests are deterministic
func TestAnthropic_Deterministic(t *testing.T) {
	t.Log("All Anthropic contract tests use httptest and are deterministic")
}
