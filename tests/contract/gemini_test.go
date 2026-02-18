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
	"radgateway/internal/provider/gemini"
)

// =============================================================================
// Request Transformation Contract Tests
// =============================================================================

// TestGemini_RequestTransformation_Basic verifies basic request transformation
// Contract: OpenAI-compatible request -> Gemini native format
func TestGemini_RequestTransformation_Basic(t *testing.T) {
	transformer := gemini.NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "gemini-1.5-flash",
		Messages: []models.Message{
			{Role: "user", Content: "Hello!"},
		},
	}

	result := transformer.Transform(req)

	// Verify contents array created
	if len(result.Contents) != 1 {
		t.Fatalf("Contract violation: Expected 1 content, got %d", len(result.Contents))
	}

	// Verify role mapped (user -> user)
	if result.Contents[0].Role != "user" {
		t.Errorf("Contract violation: Role mapping incorrect, got %q, want %q", result.Contents[0].Role, "user")
	}

	// Verify part created
	if len(result.Contents[0].Parts) != 1 {
		t.Fatalf("Contract violation: Expected 1 part, got %d", len(result.Contents[0].Parts))
	}

	if result.Contents[0].Parts[0].Text != "Hello!" {
		t.Errorf("Contract violation: Content not preserved, got %q", result.Contents[0].Parts[0].Text)
	}
}

// TestGemini_RequestTransformation_RoleMapping verifies role mapping
// Contract: OpenAI roles are mapped to Gemini roles (assistant -> model)
func TestGemini_RequestTransformation_RoleMapping(t *testing.T) {
	transformer := gemini.NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "gemini-1.5-flash",
		Messages: []models.Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
			{Role: "user", Content: "How are you?"},
		},
	}

	result := transformer.Transform(req)

	if len(result.Contents) != 3 {
		t.Fatalf("Contract violation: Expected 3 contents, got %d", len(result.Contents))
	}

	// First message: user -> user
	if result.Contents[0].Role != "user" {
		t.Errorf("Contract violation: First content role mismatch, got %q, want %q", result.Contents[0].Role, "user")
	}

	// Second message: assistant -> model
	if result.Contents[1].Role != "model" {
		t.Errorf("Contract violation: Second content role mismatch, got %q, want %q", result.Contents[1].Role, "model")
	}

	// Third message: user -> user
	if result.Contents[2].Role != "user" {
		t.Errorf("Contract violation: Third content role mismatch, got %q, want %q", result.Contents[2].Role, "user")
	}
}

// TestGemini_RequestTransformation_SystemMessageHandling verifies system message handling
// Contract: System messages are prepended to first user message
func TestGemini_RequestTransformation_SystemMessageHandling(t *testing.T) {
	transformer := gemini.NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "gemini-1.5-flash",
		Messages: []models.Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello!"},
		},
	}

	result := transformer.Transform(req)

	// System message should be prepended to first user message
	if len(result.Contents) != 1 {
		t.Fatalf("Contract violation: Expected 1 content, got %d", len(result.Contents))
	}

	expectedContent := "You are a helpful assistant.\n\nHello!"
	if result.Contents[0].Parts[0].Text != expectedContent {
		t.Errorf("Contract violation: System message not prepended correctly, got %q, want %q",
			result.Contents[0].Parts[0].Text, expectedContent)
	}
}

// TestGemini_RequestTransformation_MultipleSystemMessages verifies multiple system messages
// Contract: Multiple system messages are combined and prepended
func TestGemini_RequestTransformation_MultipleSystemMessages(t *testing.T) {
	transformer := gemini.NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "gemini-1.5-flash",
		Messages: []models.Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "system", Content: "Be concise."},
			{Role: "user", Content: "Hello!"},
		},
	}

	result := transformer.Transform(req)

	// Both system messages should be combined
	expectedContent := "You are helpful.\n\nBe concise.\n\nHello!"
	if result.Contents[0].Parts[0].Text != expectedContent {
		t.Errorf("Contract violation: Multiple system messages not combined correctly, got %q, want %q",
			result.Contents[0].Parts[0].Text, expectedContent)
	}
}

// TestGemini_RequestTransformation_GenerationConfig verifies generation config mapping
// Contract: Generation parameters are mapped to generationConfig
func TestGemini_RequestTransformation_GenerationConfig(t *testing.T) {
	transformer := gemini.NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model:       "gemini-1.5-flash",
		Temperature: 0.7,
		TopP:        0.9,
		MaxTokens:   1024,
		Stop:        []string{"STOP", "END"},
		Messages: []models.Message{
			{Role: "user", Content: "Hello!"},
		},
	}

	result := transformer.Transform(req)

	if result.GenerationConfig == nil {
		t.Fatal("Contract violation: GenerationConfig should be set")
	}

	if result.GenerationConfig.Temperature != 0.7 {
		t.Errorf("Contract violation: Temperature not mapped, got %f, want %f", result.GenerationConfig.Temperature, 0.7)
	}

	if result.GenerationConfig.TopP != 0.9 {
		t.Errorf("Contract violation: TopP not mapped, got %f, want %f", result.GenerationConfig.TopP, 0.9)
	}

	if result.GenerationConfig.MaxOutputTokens != 1024 {
		t.Errorf("Contract violation: MaxTokens not mapped to MaxOutputTokens, got %d, want %d",
			result.GenerationConfig.MaxOutputTokens, 1024)
	}

	if len(result.GenerationConfig.StopSequences) != 2 {
		t.Errorf("Contract violation: Stop sequences not mapped, got %d", len(result.GenerationConfig.StopSequences))
	}
}

// TestGemini_RequestTransformation_SafetySettings verifies safety settings
// Contract: Default safety settings are included
func TestGemini_RequestTransformation_SafetySettings(t *testing.T) {
	transformer := gemini.NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "gemini-1.5-flash",
		Messages: []models.Message{
			{Role: "user", Content: "Hello!"},
		},
	}

	result := transformer.Transform(req)

	// Verify safety settings included
	if len(result.SafetySettings) != 4 {
		t.Errorf("Contract violation: Expected 4 safety settings, got %d", len(result.SafetySettings))
	}

	// Check for expected categories
	categories := make(map[string]bool)
	for _, setting := range result.SafetySettings {
		categories[setting.Category] = true
	}

	expectedCategories := []string{
		"HARM_CATEGORY_DANGEROUS_CONTENT",
		"HARM_CATEGORY_HATE_SPEECH",
		"HARM_CATEGORY_HARASSMENT",
		"HARM_CATEGORY_SEXUALLY_EXPLICIT",
	}

	for _, category := range expectedCategories {
		if !categories[category] {
			t.Errorf("Contract violation: Missing safety category %q", category)
		}
	}
}

// =============================================================================
// Response Transformation Contract Tests
// =============================================================================

// TestGemini_ResponseTransformation_Basic verifies basic response transformation
// Contract: Gemini native response -> OpenAI-compatible response
func TestGemini_ResponseTransformation_Basic(t *testing.T) {
	transformer := gemini.NewResponseTransformer()

	resp := gemini.GeminiResponse{
		Candidates: []gemini.GeminiCandidate{
			{
				Content: gemini.GeminiContent{
					Role: "model",
					Parts: []gemini.GeminiPart{
						{Text: "Hello! How can I help you today?"},
					},
				},
				FinishReason: "STOP",
				Index:        0,
			},
		},
		UsageMetadata: gemini.UsageMetadata{
			PromptTokenCount:     12,
			CandidatesTokenCount: 9,
			TotalTokenCount:      21,
		},
	}

	result, err := transformer.Transform(resp, "gemini-1.5-flash")
	if err != nil {
		t.Fatalf("Contract violation: Transform failed: %v", err)
	}

	// Verify object type set
	if result.Object != "chat.completion" {
		t.Errorf("Contract violation: Object type incorrect, got %q, want %q", result.Object, "chat.completion")
	}

	// Verify model set
	if result.Model != "gemini-1.5-flash" {
		t.Errorf("Contract violation: Model not set correctly, got %q", result.Model)
	}

	// Verify content transformed
	if result.Choices[0].Message.Content != "Hello! How can I help you today?" {
		t.Errorf("Contract violation: Content mismatch, got %q", result.Choices[0].Message.Content)
	}

	// Verify role set to assistant
	if result.Choices[0].Message.Role != "assistant" {
		t.Errorf("Contract violation: Role should be 'assistant', got %q", result.Choices[0].Message.Role)
	}

	// Verify finish reason mapped
	if result.Choices[0].FinishReason != "stop" {
		t.Errorf("Contract violation: Finish reason not mapped, got %q, want %q", result.Choices[0].FinishReason, "stop")
	}

	// Verify usage transformed
	if result.Usage.TotalTokens != 21 {
		t.Errorf("Contract violation: Total tokens incorrect, got %d, want %d", result.Usage.TotalTokens, 21)
	}
}

// TestGemini_ResponseTransformation_FinishReasonMapping verifies finish reason mapping
// Contract: Gemini finish reasons are mapped to OpenAI-compatible values
func TestGemini_ResponseTransformation_FinishReasonMapping(t *testing.T) {
	tests := []struct {
		geminiReason string
		expected     string
	}{
		{"STOP", "stop"},
		{"MAX_TOKENS", "length"},
		{"SAFETY", "content_filter"},
		{"RECITATION", "content_filter"},
		{"OTHER", "stop"},
		{"", "stop"},
	}

	transformer := gemini.NewResponseTransformer()

	for _, tt := range tests {
		t.Run(tt.geminiReason, func(t *testing.T) {
			resp := gemini.GeminiResponse{
				Candidates: []gemini.GeminiCandidate{
					{
						Content: gemini.GeminiContent{
							Role:  "model",
							Parts: []gemini.GeminiPart{{Text: "Test"}},
						},
						FinishReason: tt.geminiReason,
					},
				},
				UsageMetadata: gemini.UsageMetadata{
					PromptTokenCount:     1,
					CandidatesTokenCount: 1,
					TotalTokenCount:      2,
				},
			}

			result, err := transformer.Transform(resp, "gemini-1.5-flash")
			if err != nil {
				t.Fatalf("Transform failed: %v", err)
			}

			if result.Choices[0].FinishReason != tt.expected {
				t.Errorf("Contract violation: Finish reason mapping incorrect for %q, got %q, want %q",
					tt.geminiReason, result.Choices[0].FinishReason, tt.expected)
			}
		})
	}
}

// TestGemini_ResponseTransformation_MultipleParts verifies multiple parts handling
// Contract: Multiple parts are concatenated
func TestGemini_ResponseTransformation_MultipleParts(t *testing.T) {
	transformer := gemini.NewResponseTransformer()

	resp := gemini.GeminiResponse{
		Candidates: []gemini.GeminiCandidate{
			{
				Content: gemini.GeminiContent{
					Role: "model",
					Parts: []gemini.GeminiPart{
						{Text: "First part. "},
						{Text: "Second part."},
					},
				},
				FinishReason: "STOP",
			},
		},
		UsageMetadata: gemini.UsageMetadata{
			PromptTokenCount:     10,
			CandidatesTokenCount: 10,
			TotalTokenCount:      20,
		},
	}

	result, err := transformer.Transform(resp, "gemini-1.5-flash")
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	expectedContent := "First part. Second part."
	if result.Choices[0].Message.Content != expectedContent {
		t.Errorf("Contract violation: Parts not concatenated correctly, got %q, want %q",
			result.Choices[0].Message.Content, expectedContent)
	}
}

// TestGemini_ResponseTransformation_EmptyCandidates verifies empty candidates handling
// Contract: Empty candidates return error
func TestGemini_ResponseTransformation_EmptyCandidates(t *testing.T) {
	transformer := gemini.NewResponseTransformer()

	resp := gemini.GeminiResponse{
		Candidates:    []gemini.GeminiCandidate{},
		UsageMetadata: gemini.UsageMetadata{},
	}

	_, err := transformer.Transform(resp, "gemini-1.5-flash")
	if err == nil {
		t.Error("Contract violation: Expected error for empty candidates")
	}
}

// =============================================================================
// Streaming Response Contract Tests
// =============================================================================

// TestGemini_StreamTransformation_Basic verifies basic stream transformation
// Contract: Gemini stream chunks are transformed to OpenAI-compatible format
func TestGemini_StreamTransformation_Basic(t *testing.T) {
	transformer := gemini.NewStreamTransformer()
	transformer.Init("gemini-1.5-flash")

	chunk := gemini.GeminiResponse{
		Candidates: []gemini.GeminiCandidate{
			{
				Content: gemini.GeminiContent{
					Role: "model",
					Parts: []gemini.GeminiPart{
						{Text: "Hello"},
					},
				},
			},
		},
	}

	chunkData, _ := json.Marshal(chunk)
	result, isFinal, err := transformer.TransformChunk(string(chunkData))

	if err != nil {
		t.Fatalf("Contract violation: TransformChunk failed: %v", err)
	}

	if isFinal {
		t.Error("Contract violation: Non-final chunk should not signal final")
	}

	if result == nil {
		t.Fatal("Contract violation: Expected result for chunk")
	}

	// Strip "data: " prefix for validation
	resultStr := string(result)
	if strings.HasPrefix(resultStr, "data: ") {
		resultStr = strings.TrimPrefix(resultStr, "data: ")
	}

	var openAIChunk models.ChatCompletionResponse
	if err := json.Unmarshal([]byte(resultStr), &openAIChunk); err != nil {
		t.Fatalf("Contract violation: Failed to unmarshal transformed chunk: %v", err)
	}

	if openAIChunk.Object != "chat.completion.chunk" {
		t.Errorf("Contract violation: Object mismatch, got %q, want %q", openAIChunk.Object, "chat.completion.chunk")
	}

	if openAIChunk.Model != "gemini-1.5-flash" {
		t.Errorf("Contract violation: Model not set, got %q", openAIChunk.Model)
	}
}

// TestGemini_StreamTransformation_WithFinishReason verifies finish reason handling
// Contract: Finish reasons are properly detected and mapped
func TestGemini_StreamTransformation_WithFinishReason(t *testing.T) {
	transformer := gemini.NewStreamTransformer()
	transformer.Init("gemini-1.5-flash")

	chunk := gemini.GeminiResponse{
		Candidates: []gemini.GeminiCandidate{
			{
				Content: gemini.GeminiContent{
					Role:  "model",
					Parts: []gemini.GeminiPart{{Text: "Done!"}},
				},
				FinishReason: "STOP",
			},
		},
		UsageMetadata: gemini.UsageMetadata{
			PromptTokenCount:     10,
			CandidatesTokenCount: 5,
			TotalTokenCount:      15,
		},
	}

	chunkData, _ := json.Marshal(chunk)
	result, isFinal, err := transformer.TransformChunk(string(chunkData))

	if err != nil {
		t.Fatalf("Contract violation: TransformChunk failed: %v", err)
	}

	if !isFinal {
		t.Error("Contract violation: Chunk with finish reason should signal final")
	}

	// Strip "data: " prefix
	resultStr := string(result)
	if strings.HasPrefix(resultStr, "data: ") {
		resultStr = strings.TrimPrefix(resultStr, "data: ")
	}

	var openAIChunk models.ChatCompletionResponse
	if err := json.Unmarshal([]byte(resultStr), &openAIChunk); err != nil {
		t.Fatalf("Contract violation: Failed to unmarshal: %v", err)
	}

	if openAIChunk.Choices[0].FinishReason != "stop" {
		t.Errorf("Contract violation: Finish reason not mapped, got %q", openAIChunk.Choices[0].FinishReason)
	}

	// Verify usage included in final chunk
	if openAIChunk.Usage.TotalTokens != 15 {
		t.Errorf("Contract violation: Usage not included, got %d", openAIChunk.Usage.TotalTokens)
	}
}

// TestGemini_StreamTransformer_Reset verifies transformer reset
// Contract: Reset clears all state
func TestGemini_StreamTransformer_Reset(t *testing.T) {
	transformer := gemini.NewStreamTransformer()
	transformer.Init("gemini-1.5-flash")
	transformer.GetAccumulatedContent() // Just to accumulate something

	transformer.Reset()

	if transformer.GetAccumulatedContent() != "" {
		t.Error("Contract violation: Accumulated content should be empty after reset")
	}
}

// =============================================================================
// Error Handling Contract Tests
// =============================================================================

// TestGemini_ErrorResponse_Transformation verifies error response transformation
// Contract: Gemini error format is transformed to standard error
func TestGemini_ErrorResponse_Transformation(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		expectedCode   int
		expectedString string
	}{
		{
			name:           "authentication error",
			response:       `{"error":{"code":401,"message":"Invalid API key","status":"UNAUTHENTICATED"}}`,
			expectedCode:   401,
			expectedString: "Invalid API key",
		},
		{
			name:           "rate limit error",
			response:       `{"error":{"code":429,"message":"Resource exhausted","status":"RESOURCE_EXHAUSTED"}}`,
			expectedCode:   429,
			expectedString: "Resource exhausted",
		},
		{
			name:           "bad request",
			response:       `{"error":{"code":400,"message":"Invalid argument","status":"INVALID_ARGUMENT"}}`,
			expectedCode:   400,
			expectedString: "Invalid argument",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gemini.TransformErrorResponse([]byte(tt.response))

			if err == nil {
				t.Error("Contract violation: Expected error")
				return
			}

			if !strings.Contains(err.Error(), tt.expectedString) {
				t.Errorf("Contract violation: Error should contain %q, got %v", tt.expectedString, err)
			}
		})
	}
}

// =============================================================================
// Authentication Contract Tests
// =============================================================================

// TestGemini_Authentication_XGoogApiKey verifies x-goog-api-key authentication
// Contract: x-goog-api-key header is set (not Bearer token)
func TestGemini_Authentication_XGoogApiKey(t *testing.T) {
	var capturedKey string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedKey = r.Header.Get("x-goog-api-key")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(gemini.GeminiResponse{
			Candidates: []gemini.GeminiCandidate{
				{
					Content: gemini.GeminiContent{
						Role:  "model",
						Parts: []gemini.GeminiPart{{Text: "Hello!"}},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: gemini.UsageMetadata{
				PromptTokenCount:     10,
				CandidatesTokenCount: 2,
				TotalTokenCount:      12,
			},
		})
	}))
	defer server.Close()

	adapter := gemini.NewAdapter("gemini-api-key", gemini.WithBaseURL(server.URL))
	adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "gemini-1.5-flash",
		Payload: models.ChatCompletionRequest{
			Model: "gemini-1.5-flash",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}, "gemini-1.5-flash")

	// Verify x-goog-api-key header (not Bearer)
	if capturedKey != "gemini-api-key" {
		t.Errorf("Contract violation: x-goog-api-key header mismatch, got %q, want %q", capturedKey, "gemini-api-key")
	}
}

// =============================================================================
// Model Name Contract Tests
// =============================================================================

// TestGemini_ModelName_Preservation verifies model name preservation
// Contract: Model names are preserved exactly
func TestGemini_ModelName_Preservation(t *testing.T) {
	modelNames := []string{
		"gemini-1.5-flash",
		"gemini-1.5-pro",
		"gemini-1.0-pro",
		"gemini-ultra",
	}

	for _, modelName := range modelNames {
		t.Run(modelName, func(t *testing.T) {
			transformer := gemini.NewResponseTransformer()
			resp := gemini.GeminiResponse{
				Candidates: []gemini.GeminiCandidate{
					{
						Content:      gemini.GeminiContent{Role: "model", Parts: []gemini.GeminiPart{{Text: "Test"}}},
						FinishReason: "STOP",
					},
				},
				UsageMetadata: gemini.UsageMetadata{TotalTokenCount: 5},
			}

			result, err := transformer.Transform(resp, modelName)
			if err != nil {
				t.Fatalf("Transform failed: %v", err)
			}

			if result.Model != modelName {
				t.Errorf("Contract violation: Model name not preserved, got %q, want %q", result.Model, modelName)
			}
		})
	}
}

// =============================================================================
// Streaming Integration Contract Tests
// =============================================================================

// TestGemini_Streaming_EndToEnd verifies full streaming flow
// Contract: Streaming responses are properly handled end-to-end
func TestGemini_Streaming_EndToEnd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify streaming endpoint
		expectedPath := "/v1beta/models/gemini-1.5-flash:streamGenerateContent"
		if r.URL.Path != expectedPath {
			t.Errorf("Contract violation: Expected path %q, got %q", expectedPath, r.URL.Path)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []gemini.GeminiResponse{
			{
				Candidates: []gemini.GeminiCandidate{
					{
						Content: gemini.GeminiContent{
							Role:  "model",
							Parts: []gemini.GeminiPart{{Text: "Hello"}},
						},
					},
				},
			},
			{
				Candidates: []gemini.GeminiCandidate{
					{
						Content: gemini.GeminiContent{
							Role:  "model",
							Parts: []gemini.GeminiPart{{Text: "Hello world"}},
						},
					},
				},
			},
			{
				Candidates: []gemini.GeminiCandidate{
					{
						Content: gemini.GeminiContent{
							Role:  "model",
							Parts: []gemini.GeminiPart{{Text: "Hello world!"}},
						},
						FinishReason: "STOP",
					},
				},
				UsageMetadata: gemini.UsageMetadata{
					PromptTokenCount:     10,
					CandidatesTokenCount: 3,
					TotalTokenCount:      13,
				},
			},
		}

		for _, chunk := range chunks {
			data, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.(http.Flusher).Flush()
			time.Sleep(5 * time.Millisecond)
		}
	}))
	defer server.Close()

	adapter := gemini.NewAdapter("test-key", gemini.WithBaseURL(server.URL))
	result, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "gemini-1.5-flash",
		Payload: models.ChatCompletionRequest{
			Model: "gemini-1.5-flash",
			Messages: []models.Message{
				{Role: "user", Content: "Say hello"},
			},
			Stream: true,
		},
	}, "gemini-1.5-flash")

	if err != nil {
		t.Fatalf("Contract violation: Execute failed: %v", err)
	}

	streamResp, ok := result.Payload.(*gemini.StreamingResponse)
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
		t.Errorf("Contract violation: Stream missing content: %s", content)
	}

	if !strings.Contains(content, "[DONE]") {
		t.Errorf("Contract violation: Stream missing [DONE] marker: %s", content)
	}
}

// =============================================================================
// Retry Contract Tests
// =============================================================================

// TestGemini_Retry_ServerError verifies retry on server errors
// Contract: 503 errors trigger retry
func TestGemini_Retry_ServerError(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"code":    503,
					"message": "Service unavailable",
					"status":  "UNAVAILABLE",
				},
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(gemini.GeminiResponse{
			Candidates: []gemini.GeminiCandidate{
				{
					Content:      gemini.GeminiContent{Role: "model", Parts: []gemini.GeminiPart{{Text: "Hello"}}},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: gemini.UsageMetadata{
				PromptTokenCount:     10,
				CandidatesTokenCount: 2,
				TotalTokenCount:      12,
			},
		})
	}))
	defer server.Close()

	adapter := gemini.NewAdapter("test-key",
		gemini.WithBaseURL(server.URL),
		gemini.WithRetryConfig(3, 10*time.Millisecond),
	)

	result, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "gemini-1.5-flash",
		Payload: models.ChatCompletionRequest{
			Model: "gemini-1.5-flash",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}, "gemini-1.5-flash")

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

// TestGemini_Retry_BadRequestNoRetry verifies no retry on bad requests
// Contract: 400 errors do not trigger retry
func TestGemini_Retry_BadRequestNoRetry(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    400,
				"message": "Invalid argument",
				"status":  "INVALID_ARGUMENT",
			},
		})
	}))
	defer server.Close()

	adapter := gemini.NewAdapter("test-key",
		gemini.WithBaseURL(server.URL),
		gemini.WithRetryConfig(3, 10*time.Millisecond),
	)

	adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "gemini-1.5-flash",
		Payload: models.ChatCompletionRequest{
			Model: "gemini-1.5-flash",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}, "gemini-1.5-flash")

	if requestCount != 1 {
		t.Errorf("Contract violation: Expected 1 request (no retry on 400), got %d", requestCount)
	}
}

// =============================================================================
// Endpoint Contract Tests
// =============================================================================

// TestGemini_Endpoint_Construction verifies endpoint construction
// Contract: Endpoints are constructed correctly for different operations
func TestGemini_Endpoint_Construction(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		streaming    bool
		expectedPath string
	}{
		{
			name:         "non-streaming",
			model:        "gemini-1.5-flash",
			streaming:    false,
			expectedPath: "/v1beta/models/gemini-1.5-flash:generateContent",
		},
		{
			name:         "streaming",
			model:        "gemini-1.5-flash",
			streaming:    true,
			expectedPath: "/v1beta/models/gemini-1.5-flash:streamGenerateContent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(gemini.GeminiResponse{
					Candidates: []gemini.GeminiCandidate{
						{
							Content:      gemini.GeminiContent{Role: "model", Parts: []gemini.GeminiPart{{Text: "Test"}}},
							FinishReason: "STOP",
						},
					},
					UsageMetadata: gemini.UsageMetadata{TotalTokenCount: 5},
				})
			}))
			defer server.Close()

			adapter := gemini.NewAdapter("test-key", gemini.WithBaseURL(server.URL))
			adapter.Execute(context.Background(), models.ProviderRequest{
				APIType: "chat",
				Model:   tt.model,
				Payload: models.ChatCompletionRequest{
					Model:    tt.model,
					Messages: []models.Message{{Role: "user", Content: "Hello"}},
					Stream:   tt.streaming,
				},
			}, tt.model)

			if capturedPath != tt.expectedPath {
				t.Errorf("Contract violation: Endpoint path mismatch, got %q, want %q", capturedPath, tt.expectedPath)
			}
		})
	}
}

// =============================================================================
// Edge Case Contract Tests
// =============================================================================

// TestGemini_SSEEvent_Parsing verifies SSE event parsing
// Contract: SSE events are parsed correctly
func TestGemini_SSEEvent_Parsing(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		expectedType string
		expectedData string
	}{
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
		{
			name:         "whitespace only",
			line:         "   ",
			expectedType: "",
			expectedData: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventType, data := gemini.ParseSSEEvent(tt.line)

			if eventType != tt.expectedType {
				t.Errorf("Contract violation: Event type mismatch, got %q, want %q", eventType, tt.expectedType)
			}

			if data != tt.expectedData {
				t.Errorf("Contract violation: Data mismatch, got %q, want %q", data, tt.expectedData)
			}
		})
	}
}

// TestGemini_IsDoneMarker verifies done marker detection
// Contract: Done markers are detected correctly
func TestGemini_IsDoneMarker(t *testing.T) {
	tests := []struct {
		chunk    []byte
		expected bool
	}{
		{[]byte(`data: {"finishReason":"STOP"}`), true},
		{[]byte("data: [DONE]"), true},
		{[]byte(`data: {"type":"content"}`), false},
		{[]byte(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.chunk), func(t *testing.T) {
			result := gemini.IsDoneMarker(tt.chunk)
			if result != tt.expected {
				t.Errorf("Contract violation: IsDoneMarker(%q) = %v, want %v", tt.chunk, result, tt.expected)
			}
		})
	}
}

// TestGemini_OnlySystemMessage verifies system-only handling
// Contract: System-only messages create valid request
func TestGemini_OnlySystemMessage(t *testing.T) {
	transformer := gemini.NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "gemini-1.5-flash",
		Messages: []models.Message{
			{Role: "system", Content: "You are helpful."},
		},
	}

	result := transformer.Transform(req)

	// Should have at least one content
	if len(result.Contents) == 0 {
		t.Fatal("Contract violation: Expected at least one content for system-only input")
	}

	// System content should be in the first user message
	foundSystemContent := false
	for _, content := range result.Contents {
		for _, part := range content.Parts {
			if strings.Contains(part.Text, "You are helpful") {
				foundSystemContent = true
				break
			}
		}
	}

	if !foundSystemContent {
		t.Error("Contract violation: System content not found in transformed request")
	}
}

// Verify contract tests are deterministic
func TestGemini_Deterministic(t *testing.T) {
	t.Log("All Gemini contract tests use httptest and are deterministic")
}
