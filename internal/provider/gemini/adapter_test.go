// Package gemini provides an adapter for Google's Gemini API.
package gemini

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

// =============================================================================
// Adapter Creation Tests
// =============================================================================

func TestNewAdapter(t *testing.T) {
	adapter := NewAdapter("test-api-key")

	if adapter == nil {
		t.Fatal("Expected adapter to be created, got nil")
	}

	if adapter.Name() != "gemini" {
		t.Errorf("Expected name 'gemini', got %q", adapter.Name())
	}

	if adapter.config.APIKey != "test-api-key" {
		t.Errorf("Expected API key 'test-api-key', got %q", adapter.config.APIKey)
	}

	if adapter.config.BaseURL != defaultBaseURL {
		t.Errorf("Expected base URL %q, got %q", defaultBaseURL, adapter.config.BaseURL)
	}

	if adapter.config.Version != defaultVersion {
		t.Errorf("Expected version %q, got %q", defaultVersion, adapter.config.Version)
	}

	if adapter.config.Timeout != defaultTimeout {
		t.Errorf("Expected timeout %v, got %v", defaultTimeout, adapter.config.Timeout)
	}

	if adapter.config.MaxRetries != defaultMaxRetries {
		t.Errorf("Expected max retries %d, got %d", defaultMaxRetries, adapter.config.MaxRetries)
	}

	if adapter.httpClient == nil {
		t.Error("Expected HTTP client to be initialized")
	}
}

func TestAdapterOptions(t *testing.T) {
	customClient := &http.Client{Timeout: 10 * time.Second}
	customTimeout := 30 * time.Second
	customBaseURL := "https://custom.gemini.api.com"
	customVersion := "v1"

	adapter := NewAdapter(
		"test-key",
		WithHTTPClient(customClient),
		WithTimeout(customTimeout),
		WithBaseURL(customBaseURL),
		WithVersion(customVersion),
		WithRetryConfig(5, 2*time.Second),
	)

	if adapter.httpClient != customClient {
		t.Error("HTTP client not set correctly")
	}

	if adapter.config.BaseURL != customBaseURL {
		t.Errorf("BaseURL mismatch: expected %q, got %q", customBaseURL, adapter.config.BaseURL)
	}

	if adapter.config.Version != customVersion {
		t.Errorf("Version mismatch: expected %q, got %q", customVersion, adapter.config.Version)
	}

	if adapter.config.MaxRetries != 5 {
		t.Errorf("MaxRetries mismatch: expected 5, got %d", adapter.config.MaxRetries)
	}

	if adapter.config.RetryDelay != 2*time.Second {
		t.Errorf("RetryDelay mismatch: expected 2s, got %v", adapter.config.RetryDelay)
	}
}

func TestAdapter_Name(t *testing.T) {
	adapter := NewAdapter("test-key")
	if adapter.Name() != "gemini" {
		t.Errorf("Expected name 'gemini', got %q", adapter.Name())
	}
}

// =============================================================================
// Request Transformation Tests
// =============================================================================

func TestRequestTransformer_Transform_Simple(t *testing.T) {
	transformer := NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "gemini-1.5-flash",
		Messages: []models.Message{
			{Role: "user", Content: "Hello, Gemini!"},
		},
	}

	geminiReq := transformer.Transform(req)

	if len(geminiReq.Contents) != 1 {
		t.Fatalf("Expected 1 content, got %d", len(geminiReq.Contents))
	}

	if geminiReq.Contents[0].Role != "user" {
		t.Errorf("Expected role 'user', got %q", geminiReq.Contents[0].Role)
	}

	if len(geminiReq.Contents[0].Parts) != 1 {
		t.Fatalf("Expected 1 part, got %d", len(geminiReq.Contents[0].Parts))
	}

	if geminiReq.Contents[0].Parts[0].Text != "Hello, Gemini!" {
		t.Errorf("Expected text 'Hello, Gemini!', got %q", geminiReq.Contents[0].Parts[0].Text)
	}

	// Verify safety settings are included
	if len(geminiReq.SafetySettings) != 4 {
		t.Errorf("Expected 4 safety settings, got %d", len(geminiReq.SafetySettings))
	}
}

func TestRequestTransformer_Transform_WithSystem(t *testing.T) {
	transformer := NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "gemini-1.5-flash",
		Messages: []models.Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello!"},
		},
	}

	geminiReq := transformer.Transform(req)

	// System message should be prepended to first user message
	if len(geminiReq.Contents) != 1 {
		t.Fatalf("Expected 1 content, got %d", len(geminiReq.Contents))
	}

	expectedContent := "You are a helpful assistant.\n\nHello!"
	if geminiReq.Contents[0].Parts[0].Text != expectedContent {
		t.Errorf("Expected text %q, got %q", expectedContent, geminiReq.Contents[0].Parts[0].Text)
	}
}

func TestRequestTransformer_Transform_WithGenerationConfig(t *testing.T) {
	transformer := NewRequestTransformer()
	temp := 0.7
	topP := 0.9

	req := models.ChatCompletionRequest{
		Model:       "gemini-1.5-flash",
		Temperature: temp,
		TopP:        topP,
		MaxTokens:   1024,
		Messages: []models.Message{
			{Role: "user", Content: "Hello!"},
		},
		Stop: []string{"STOP", "END"},
	}

	geminiReq := transformer.Transform(req)

	if geminiReq.GenerationConfig == nil {
		t.Fatal("Expected generation config to be set")
	}

	if geminiReq.GenerationConfig.Temperature != temp {
		t.Errorf("Temperature mismatch: expected %f, got %f", temp, geminiReq.GenerationConfig.Temperature)
	}

	if geminiReq.GenerationConfig.TopP != topP {
		t.Errorf("TopP mismatch: expected %f, got %f", topP, geminiReq.GenerationConfig.TopP)
	}

	if geminiReq.GenerationConfig.MaxOutputTokens != 1024 {
		t.Errorf("MaxOutputTokens mismatch: expected 1024, got %d", geminiReq.GenerationConfig.MaxOutputTokens)
	}

	if len(geminiReq.GenerationConfig.StopSequences) != 2 {
		t.Errorf("StopSequences length mismatch: expected 2, got %d", len(geminiReq.GenerationConfig.StopSequences))
	}
}

func TestRequestTransformer_Transform_RoleMapping(t *testing.T) {
	transformer := NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "gemini-1.5-flash",
		Messages: []models.Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
			{Role: "user", Content: "How are you?"},
		},
	}

	geminiReq := transformer.Transform(req)

	if len(geminiReq.Contents) != 3 {
		t.Fatalf("Expected 3 contents, got %d", len(geminiReq.Contents))
	}

	// First message: user -> user
	if geminiReq.Contents[0].Role != "user" {
		t.Errorf("First content role mismatch: expected 'user', got %q", geminiReq.Contents[0].Role)
	}

	// Second message: assistant -> model
	if geminiReq.Contents[1].Role != "model" {
		t.Errorf("Second content role mismatch: expected 'model', got %q", geminiReq.Contents[1].Role)
	}

	// Third message: user -> user
	if geminiReq.Contents[2].Role != "user" {
		t.Errorf("Third content role mismatch: expected 'user', got %q", geminiReq.Contents[2].Role)
	}
}

func TestRequestTransformer_Transform_MultipleSystemMessages(t *testing.T) {
	transformer := NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "gemini-1.5-flash",
		Messages: []models.Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "system", Content: "Be concise."},
			{Role: "user", Content: "Hello!"},
		},
	}

	geminiReq := transformer.Transform(req)

	// Both system messages should be combined and prepended
	expectedContent := "You are helpful.\n\nBe concise.\n\nHello!"
	if geminiReq.Contents[0].Parts[0].Text != expectedContent {
		t.Errorf("Combined system message mismatch: expected %q, got %q", expectedContent, geminiReq.Contents[0].Parts[0].Text)
	}
}

// =============================================================================
// Response Transformation Tests
// =============================================================================

func TestResponseTransformer_Transform(t *testing.T) {
	transformer := NewResponseTransformer()

	geminiResp := GeminiResponse{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Role: "model",
					Parts: []GeminiPart{
						{Text: "Hello! How can I help you today?"},
					},
				},
				FinishReason: "STOP",
				Index:        0,
			},
		},
		UsageMetadata: UsageMetadata{
			PromptTokenCount:     12,
			CandidatesTokenCount: 9,
			TotalTokenCount:      21,
		},
	}

	result, err := transformer.Transform(geminiResp, "gemini-1.5-flash")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ID == "" {
		t.Error("Expected non-empty ID")
	}

	if result.Object != "chat.completion" {
		t.Errorf("Object mismatch: expected 'chat.completion', got %q", result.Object)
	}

	if result.Model != "gemini-1.5-flash" {
		t.Errorf("Model mismatch: got %q", result.Model)
	}

	if len(result.Choices) != 1 {
		t.Fatalf("Expected 1 choice, got %d", len(result.Choices))
	}

	if result.Choices[0].Message.Role != "assistant" {
		t.Errorf("Role mismatch: expected 'assistant', got %q", result.Choices[0].Message.Role)
	}

	if result.Choices[0].Message.Content != "Hello! How can I help you today?" {
		t.Errorf("Content mismatch: got %q", result.Choices[0].Message.Content)
	}

	if result.Choices[0].FinishReason != "stop" {
		t.Errorf("FinishReason mismatch: expected 'stop', got %q", result.Choices[0].FinishReason)
	}
}

func TestResponseTransformer_Transform_WithUsage(t *testing.T) {
	transformer := NewResponseTransformer()

	geminiResp := GeminiResponse{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Role: "model",
					Parts: []GeminiPart{
						{Text: "Test response"},
					},
				},
				FinishReason: "STOP",
			},
		},
		UsageMetadata: UsageMetadata{
			PromptTokenCount:     100,
			CandidatesTokenCount: 50,
			TotalTokenCount:      150,
		},
	}

	result, err := transformer.Transform(geminiResp, "gemini-1.5-flash")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Usage.PromptTokens != 100 {
		t.Errorf("PromptTokens mismatch: expected 100, got %d", result.Usage.PromptTokens)
	}

	if result.Usage.CompletionTokens != 50 {
		t.Errorf("CompletionTokens mismatch: expected 50, got %d", result.Usage.CompletionTokens)
	}

	if result.Usage.TotalTokens != 150 {
		t.Errorf("TotalTokens mismatch: expected 150, got %d", result.Usage.TotalTokens)
	}
}

func TestResponseTransformer_Transform_MultipleParts(t *testing.T) {
	transformer := NewResponseTransformer()

	geminiResp := GeminiResponse{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Role: "model",
					Parts: []GeminiPart{
						{Text: "First part. "},
						{Text: "Second part."},
					},
				},
				FinishReason: "STOP",
			},
		},
		UsageMetadata: UsageMetadata{
			PromptTokenCount:     10,
			CandidatesTokenCount: 10,
			TotalTokenCount:      20,
		},
	}

	result, err := transformer.Transform(geminiResp, "gemini-1.5-flash")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedContent := "First part. Second part."
	if result.Choices[0].Message.Content != expectedContent {
		t.Errorf("Combined content mismatch: expected %q, got %q", expectedContent, result.Choices[0].Message.Content)
	}
}

func TestResponseTransformer_Transform_FinishReasonMapping(t *testing.T) {
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

	transformer := NewResponseTransformer()

	for _, tt := range tests {
		t.Run(tt.geminiReason, func(t *testing.T) {
			geminiResp := GeminiResponse{
				Candidates: []GeminiCandidate{
					{
						Content: GeminiContent{
							Role:  "model",
							Parts: []GeminiPart{{Text: "Test"}},
						},
						FinishReason: tt.geminiReason,
					},
				},
				UsageMetadata: UsageMetadata{
					PromptTokenCount:     1,
					CandidatesTokenCount: 1,
					TotalTokenCount:      2,
				},
			}

			result, err := transformer.Transform(geminiResp, "gemini-1.5-flash")
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Choices[0].FinishReason != tt.expected {
				t.Errorf("FinishReason mismatch: expected %q, got %q", tt.expected, result.Choices[0].FinishReason)
			}
		})
	}
}

func TestResponseTransformer_Transform_EmptyCandidates(t *testing.T) {
	transformer := NewResponseTransformer()

	geminiResp := GeminiResponse{
		Candidates:    []GeminiCandidate{},
		UsageMetadata: UsageMetadata{},
	}

	_, err := transformer.Transform(geminiResp, "gemini-1.5-flash")
	if err == nil {
		t.Error("Expected error for empty candidates")
	}
}

// =============================================================================
// Streaming Tests
// =============================================================================

func TestStreamTransformer_TransformChunk(t *testing.T) {
	transformer := NewStreamTransformer()
	transformer.Init("gemini-1.5-flash")

	geminiChunk := GeminiResponse{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Role: "model",
					Parts: []GeminiPart{
						{Text: "Hello"},
					},
				},
				FinishReason: "",
			},
		},
	}

	chunkData, _ := json.Marshal(geminiChunk)
	result, _, err := transformer.TransformChunk(string(chunkData))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Strip "data: " prefix if present before unmarshaling
	resultStr := string(result)
	if strings.HasPrefix(resultStr, "data: ") {
		resultStr = strings.TrimPrefix(resultStr, "data: ")
	}

	// Verify the transformed chunk is OpenAI-compatible format
	var openAIChunk models.ChatCompletionResponse
	if err := json.Unmarshal([]byte(resultStr), &openAIChunk); err != nil {
		t.Fatalf("Failed to unmarshal transformed chunk: %v", err)
	}

	if openAIChunk.Object != "chat.completion.chunk" {
		t.Errorf("Object mismatch: expected 'chat.completion.chunk', got %q", openAIChunk.Object)
	}

	if len(openAIChunk.Choices) != 1 {
		t.Fatalf("Expected 1 choice, got %d", len(openAIChunk.Choices))
	}

	if openAIChunk.Choices[0].Message.Content != "Hello" {
		t.Errorf("Content mismatch: expected 'Hello', got %q", openAIChunk.Choices[0].Message.Content)
	}
}

func TestStreamTransformer_TransformChunk_WithFinishReason(t *testing.T) {
	transformer := NewStreamTransformer()
	transformer.Init("gemini-1.5-flash")

	geminiChunk := GeminiResponse{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Role:  "model",
					Parts: []GeminiPart{{Text: "Done!"}},
				},
				FinishReason: "STOP",
			},
		},
	}

	chunkData, _ := json.Marshal(geminiChunk)
	result, _, err := transformer.TransformChunk(string(chunkData))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Strip "data: " prefix if present
	resultStr := string(result)
	if strings.HasPrefix(resultStr, "data: ") {
		resultStr = strings.TrimPrefix(resultStr, "data: ")
	}

	var openAIChunk models.ChatCompletionResponse
	if err := json.Unmarshal([]byte(resultStr), &openAIChunk); err != nil {
		t.Fatalf("Failed to unmarshal transformed chunk: %v", err)
	}

	if openAIChunk.Choices[0].FinishReason != "stop" {
		t.Errorf("FinishReason mismatch: expected 'stop', got %q", openAIChunk.Choices[0].FinishReason)
	}
}

func TestStreamTransformer_Reset(t *testing.T) {
	transformer := NewStreamTransformer()
	transformer.Init("gemini-1.5-flash")
	transformer.accumulatedContent = "some content"

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

	if transformer.accumulatedContent != "" {
		t.Error("accumulatedContent should be empty after reset")
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestTransformErrorResponse(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		expectError bool
		checkError  func(t *testing.T, err error)
	}{
		{
			name:        "valid error response",
			body:        `{"error":{"code":400,"message":"Invalid request","status":"INVALID_ARGUMENT"}}`,
			expectError: true,
			checkError: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("Expected error")
				}
				if !strings.Contains(err.Error(), "Invalid request") {
					t.Errorf("Error should contain message: %v", err)
				}
				if !strings.Contains(err.Error(), "400") {
					t.Errorf("Error should contain code: %v", err)
				}
			},
		},
		{
			name:        "error without code",
			body:        `{"error":{"message":"Something went wrong"}}`,
			expectError: true,
			checkError: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("Expected error")
				}
				if !strings.Contains(err.Error(), "Something went wrong") {
					t.Errorf("Error should contain message: %v", err)
				}
			},
		},
		{
			name:        "invalid json",
			body:        `not valid json`,
			expectError: true,
			checkError: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("Expected error")
				}
				if !strings.Contains(err.Error(), "gemini error") {
					t.Errorf("Error should indicate gemini error: %v", err)
				}
			},
		},
		{
			name:        "empty body",
			body:        ``,
			expectError: true,
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

func TestExecute_Error(t *testing.T) {
	tests := []struct {
		name         string
		serverStatus int
		response     string
		expectError  bool
	}{
		{
			name:         "authentication error",
			serverStatus: http.StatusUnauthorized,
			response:     `{"error":{"code":401,"message":"Invalid API key","status":"UNAUTHENTICATED"}}`,
			expectError:  true,
		},
		{
			name:         "bad request error",
			serverStatus: http.StatusBadRequest,
			response:     `{"error":{"code":400,"message":"Invalid argument","status":"INVALID_ARGUMENT"}}`,
			expectError:  true,
		},
		{
			name:         "rate limit error",
			serverStatus: http.StatusTooManyRequests,
			response:     `{"error":{"code":429,"message":"Resource exhausted","status":"RESOURCE_EXHAUSTED"}}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			adapter := NewAdapter("test-key", WithBaseURL(server.URL))
			_, err := adapter.Execute(context.Background(), models.ProviderRequest{
				APIType: "chat",
				Model:   "gemini-1.5-flash",
				Payload: models.ChatCompletionRequest{
					Model: "gemini-1.5-flash",
					Messages: []models.Message{
						{Role: "user", Content: "Hello"},
					},
				},
			}, "gemini-1.5-flash")

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
		})
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestExecute_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Verify URL contains the model and endpoint
		expectedPath := "/v1beta/models/gemini-1.5-flash:generateContent"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %q, got %q", expectedPath, r.URL.Path)
		}

		// Verify headers
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got %q", contentType)
		}

		// Verify request body
		var reqBody GeminiRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if len(reqBody.Contents) == 0 {
			t.Error("Expected non-empty contents")
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Role: "model",
						Parts: []GeminiPart{
							{Text: "Hello! How can I help you today?"},
						},
					},
					FinishReason: "STOP",
					Index:        0,
				},
			},
			UsageMetadata: UsageMetadata{
				PromptTokenCount:     12,
				CandidatesTokenCount: 9,
				TotalTokenCount:      21,
			},
		})
	}))
	defer server.Close()

	adapter := NewAdapter("test-api-key", WithBaseURL(server.URL))
	result, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "gemini-1.5-flash",
		Payload: models.ChatCompletionRequest{
			Model: "gemini-1.5-flash",
			Messages: []models.Message{
				{Role: "user", Content: "Hello!"},
			},
		},
	}, "gemini-1.5-flash")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Provider != "gemini" {
		t.Errorf("Provider mismatch: expected 'gemini', got %q", result.Provider)
	}

	if result.Status != "success" {
		t.Errorf("Status mismatch: expected 'success', got %q", result.Status)
	}

	if result.Usage.TotalTokens != 21 {
		t.Errorf("TotalTokens mismatch: expected 21, got %d", result.Usage.TotalTokens)
	}

	resp, ok := result.Payload.(*models.ChatCompletionResponse)
	if !ok {
		t.Fatalf("Expected *ChatCompletionResponse, got %T", result.Payload)
	}

	if len(resp.Choices) != 1 {
		t.Fatalf("Expected 1 choice, got %d", len(resp.Choices))
	}

	if resp.Choices[0].Message.Content != "Hello! How can I help you today?" {
		t.Errorf("Content mismatch: got %q", resp.Choices[0].Message.Content)
	}
}

func TestExecute_NonStreaming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Role:  "model",
						Parts: []GeminiPart{{Text: "Test response"}},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: UsageMetadata{
				PromptTokenCount:     5,
				CandidatesTokenCount: 5,
				TotalTokenCount:      10,
			},
		})
	}))
	defer server.Close()

	adapter := NewAdapter("test-key", WithBaseURL(server.URL))
	result, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "gemini-1.5-flash",
		Payload: models.ChatCompletionRequest{
			Model: "gemini-1.5-flash",
			Messages: []models.Message{
				{Role: "user", Content: "Test"},
			},
			Stream: false,
		},
	}, "gemini-1.5-flash")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected status 'success', got %q", result.Status)
	}
}

func TestExecute_Streaming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify streaming endpoint
		expectedPath := "/v1beta/models/gemini-1.5-flash:streamGenerateContent"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %q, got %q", expectedPath, r.URL.Path)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Gemini streaming chunks
		chunks := []GeminiResponse{
			{
				Candidates: []GeminiCandidate{
					{
						Content: GeminiContent{
							Role:  "model",
							Parts: []GeminiPart{{Text: "Hello"}},
						},
					},
				},
			},
			{
				Candidates: []GeminiCandidate{
					{
						Content: GeminiContent{
							Role:  "model",
							Parts: []GeminiPart{{Text: "Hello world"}},
						},
					},
				},
			},
			{
				Candidates: []GeminiCandidate{
					{
						Content: GeminiContent{
							Role:  "model",
							Parts: []GeminiPart{{Text: "Hello world!"}},
						},
						FinishReason: "STOP",
					},
				},
				UsageMetadata: UsageMetadata{
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
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer server.Close()

	adapter := NewAdapter("test-key", WithBaseURL(server.URL))
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
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Provider != "gemini" {
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

// =============================================================================
// Edge Cases and Additional Tests
// =============================================================================

func TestExecute_UnsupportedType(t *testing.T) {
	adapter := NewAdapter("test-key")
	_, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "embeddings",
		Model:   "gemini-1.5-flash",
		Payload: struct{}{},
	}, "gemini-1.5-flash")

	if err == nil {
		t.Error("Expected error for unsupported API type")
	}

	if !strings.Contains(err.Error(), "unsupported api type") {
		t.Errorf("Expected 'unsupported api type' error, got: %v", err)
	}
}

func TestExecute_InvalidPayload(t *testing.T) {
	adapter := NewAdapter("test-key")
	_, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "gemini-1.5-flash",
		Payload: "invalid payload type",
	}, "gemini-1.5-flash")

	if err == nil {
		t.Error("Expected error for invalid payload")
	}

	if !strings.Contains(err.Error(), "invalid chat payload type") {
		t.Errorf("Expected payload type error, got: %v", err)
	}
}

func TestExecute_ContextCancellation(t *testing.T) {
	// Slow server that takes longer than context timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content:      GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "Test"}}},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: UsageMetadata{
				PromptTokenCount:     1,
				CandidatesTokenCount: 1,
				TotalTokenCount:      2,
			},
		})
	}))
	defer server.Close()

	adapter := NewAdapter("test-key", WithBaseURL(server.URL), WithTimeout(100*time.Millisecond))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := adapter.Execute(ctx, models.ProviderRequest{
		APIType: "chat",
		Model:   "gemini-1.5-flash",
		Payload: models.ChatCompletionRequest{
			Model: "gemini-1.5-flash",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}, "gemini-1.5-flash")

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

func TestAdapter_RetryOnServerError(t *testing.T) {
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
		json.NewEncoder(w).Encode(GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content:      GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "Hello"}}},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: UsageMetadata{
				PromptTokenCount:     10,
				CandidatesTokenCount: 2,
				TotalTokenCount:      12,
			},
		})
	}))
	defer server.Close()

	adapter := NewAdapter("test-key", WithBaseURL(server.URL), WithRetryConfig(3, 10*time.Millisecond))
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
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected success status, got %q", result.Status)
	}

	if requestCount < 2 {
		t.Errorf("Expected at least 2 requests due to retry, got %d", requestCount)
	}
}

func TestMapFinishReason(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"STOP", "stop"},
		{"MAX_TOKENS", "length"},
		{"SAFETY", "content_filter"},
		{"RECITATION", "content_filter"},
		{"OTHER", "stop"},
		{"UNKNOWN", "stop"},
		{"", "stop"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapFinishReason(tt.input)
			if result != tt.expected {
				t.Errorf("mapFinishReason(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExecute_MalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	adapter := NewAdapter("test-key", WithBaseURL(server.URL))
	_, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "gemini-1.5-flash",
		Payload: models.ChatCompletionRequest{
			Model: "gemini-1.5-flash",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}, "gemini-1.5-flash")

	if err == nil {
		t.Error("Expected error for malformed response")
	}
}

func TestExecute_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	adapter := NewAdapter("test-key", WithBaseURL(server.URL))
	_, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Model:   "gemini-1.5-flash",
		Payload: models.ChatCompletionRequest{
			Model: "gemini-1.5-flash",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}, "gemini-1.5-flash")

	if err == nil {
		t.Error("Expected error for empty response")
	}
}

func TestRequestTransformer_Transform_ConsecutiveSameRole(t *testing.T) {
	transformer := NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model: "gemini-1.5-flash",
		Messages: []models.Message{
			{Role: "user", Content: "First"},
			{Role: "user", Content: "Second"},
			{Role: "assistant", Content: "Response"},
		},
	}

	geminiReq := transformer.Transform(req)

	// Verify all content is present (may be in parts or separate contents)
	allText := ""
	for _, content := range geminiReq.Contents {
		for _, part := range content.Parts {
			allText += part.Text
		}
	}
	if !strings.Contains(allText, "First") || !strings.Contains(allText, "Second") || !strings.Contains(allText, "Response") {
		t.Errorf("Not all message content present in transformed request")
	}
}

func TestGeminiAPIError_Error(t *testing.T) {
	err := &GeminiAPIError{
		Code:    400,
		Message: "Invalid request",
		Status:  "INVALID_ARGUMENT",
	}

	expected := "gemini error (400): Invalid request"
	if err.Error() != expected {
		t.Errorf("Error() = %q, expected %q", err.Error(), expected)
	}
}

func TestExecute_StreamEventProcessing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send events with actual SSE formatting
		events := []GeminiResponse{
			{
				Candidates: []GeminiCandidate{
					{
						Content: GeminiContent{
							Role:  "model",
							Parts: []GeminiPart{{Text: "Test"}},
						},
					},
				},
			},
			{
				Candidates: []GeminiCandidate{
					{
						Content: GeminiContent{
							Role:  "model",
							Parts: []GeminiPart{{Text: "Test content"}},
						},
						FinishReason: "STOP",
					},
				},
				UsageMetadata: UsageMetadata{
					PromptTokenCount:     10,
					CandidatesTokenCount: 2,
					TotalTokenCount:      12,
				},
			},
		}

		for _, e := range events {
			data, _ := json.Marshal(e)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.(http.Flusher).Flush()
		}
	}))
	defer server.Close()

	adapter := NewAdapter("test-key", WithBaseURL(server.URL))
	result, err := adapter.Execute(context.Background(), models.ProviderRequest{
		APIType: "chat",
		Payload: models.ChatCompletionRequest{
			Model: "gemini-1.5-flash",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
			Stream: true,
		},
	}, "gemini-1.5-flash")

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
