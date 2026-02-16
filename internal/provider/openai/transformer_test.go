package openai

import (
	"strings"
	"testing"

	"radgateway/internal/models"
)

func TestRequestTransformer_Transform(t *testing.T) {
	tests := []struct {
		name     string
		input    models.ChatCompletionRequest
		expected OpenAIRequest
	}{
		{
			name: "basic chat request",
			input: models.ChatCompletionRequest{
				Model: "gpt-4o",
				Messages: []models.Message{
					{Role: "system", Content: "You are a helpful assistant."},
					{Role: "user", Content: "Hello!"},
				},
				Stream: false,
				User:   "user-123",
			},
			expected: OpenAIRequest{
				Model: "gpt-4o",
				Messages: []OpenAIMessage{
					{Role: "system", Content: "You are a helpful assistant."},
					{Role: "user", Content: "Hello!"},
				},
				Stream: false,
				User:   "user-123",
			},
		},
		{
			name: "streaming request",
			input: models.ChatCompletionRequest{
				Model: "gpt-4o-mini",
				Messages: []models.Message{
					{Role: "user", Content: "Tell me a story"},
				},
				Stream: true,
			},
			expected: OpenAIRequest{
				Model: "gpt-4o-mini",
				Messages: []OpenAIMessage{
					{Role: "user", Content: "Tell me a story"},
				},
				Stream: true,
			},
		},
		{
			name: "empty messages",
			input: models.ChatCompletionRequest{
				Model:    "gpt-3.5-turbo",
				Messages: []models.Message{},
			},
			expected: OpenAIRequest{
				Model:    "gpt-3.5-turbo",
				Messages: []OpenAIMessage{},
			},
		},
		{
			name: "multiple messages with different roles",
			input: models.ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []models.Message{
					{Role: "system", Content: "You are a coding assistant."},
					{Role: "user", Content: "Write a function in Go."},
					{Role: "assistant", Content: "Here's a Go function..."},
					{Role: "user", Content: "Can you add comments?"},
				},
			},
			expected: OpenAIRequest{
				Model: "gpt-4",
				Messages: []OpenAIMessage{
					{Role: "system", Content: "You are a coding assistant."},
					{Role: "user", Content: "Write a function in Go."},
					{Role: "assistant", Content: "Here's a Go function..."},
					{Role: "user", Content: "Can you add comments?"},
				},
			},
		},
	}

	transformer := NewRequestTransformer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformer.Transform(tt.input)

			if result.Model != tt.expected.Model {
				t.Errorf("Model mismatch: got %q, want %q", result.Model, tt.expected.Model)
			}

			if result.Stream != tt.expected.Stream {
				t.Errorf("Stream mismatch: got %v, want %v", result.Stream, tt.expected.Stream)
			}

			if result.User != tt.expected.User {
				t.Errorf("User mismatch: got %q, want %q", result.User, tt.expected.User)
			}

			if len(result.Messages) != len(tt.expected.Messages) {
				t.Fatalf("Messages length mismatch: got %d, want %d", len(result.Messages), len(tt.expected.Messages))
			}

			for i, msg := range result.Messages {
				if msg.Role != tt.expected.Messages[i].Role {
					t.Errorf("Message[%d].Role mismatch: got %q, want %q", i, msg.Role, tt.expected.Messages[i].Role)
				}
				if msg.Content != tt.expected.Messages[i].Content {
					t.Errorf("Message[%d].Content mismatch: got %q, want %q", i, msg.Content, tt.expected.Messages[i].Content)
				}
			}
		})
	}
}

func TestResponseTransformer_Transform(t *testing.T) {
	tests := []struct {
		name        string
		input       OpenAIResponse
		expected    models.ChatCompletionResponse
		expectError bool
	}{
		{
			name: "successful response",
			input: OpenAIResponse{
				ID:      "chatcmpl-123",
				Object:  "chat.completion",
				Created: 1677652288,
				Model:   "gpt-4o",
				Choices: []OpenAIChoice{
					{
						Index: 0,
						Message: OpenAIMessage{
							Role:    "assistant",
							Content: "Hello! How can I help you today?",
						},
						FinishReason: "stop",
					},
				},
				Usage: OpenAIUsage{
					PromptTokens:     10,
					CompletionTokens: 20,
					TotalTokens:      30,
				},
			},
			expected: models.ChatCompletionResponse{
				ID:     "chatcmpl-123",
				Object: "chat.completion",
				Model:  "gpt-4o",
				Choices: []models.ChatChoice{
					{
						Index: 0,
						Message: models.Message{
							Role:    "assistant",
							Content: "Hello! How can I help you today?",
						},
					},
				},
				Usage: models.Usage{
					PromptTokens:     10,
					CompletionTokens: 20,
					TotalTokens:      30,
				},
			},
		},
		{
			name: "multiple choices",
			input: OpenAIResponse{
				ID:      "chatcmpl-456",
				Object:  "chat.completion",
				Model:   "gpt-4o-mini",
				Choices: []OpenAIChoice{
					{
						Index: 0,
						Message: OpenAIMessage{
							Role:    "assistant",
							Content: "First response",
						},
						FinishReason: "stop",
					},
					{
						Index: 1,
						Message: OpenAIMessage{
							Role:    "assistant",
							Content: "Second response",
						},
						FinishReason: "stop",
					},
				},
				Usage: OpenAIUsage{
					PromptTokens:     5,
					CompletionTokens: 10,
					TotalTokens:      15,
				},
			},
			expected: models.ChatCompletionResponse{
				ID:     "chatcmpl-456",
				Object: "chat.completion",
				Model:  "gpt-4o-mini",
				Choices: []models.ChatChoice{
					{
						Index: 0,
						Message: models.Message{
							Role:    "assistant",
							Content: "First response",
						},
					},
					{
						Index: 1,
						Message: models.Message{
							Role:    "assistant",
							Content: "Second response",
						},
					},
				},
				Usage: models.Usage{
					PromptTokens:     5,
					CompletionTokens: 10,
					TotalTokens:      15,
				},
			},
		},
		{
			name: "error response",
			input: OpenAIResponse{
				Error: &OpenAIError{
					Message: "Invalid API key",
					Type:    "invalid_request_error",
					Code:    "invalid_api_key",
				},
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

			if result.ID != tt.expected.ID {
				t.Errorf("ID mismatch: got %q, want %q", result.ID, tt.expected.ID)
			}

			if result.Object != tt.expected.Object {
				t.Errorf("Object mismatch: got %q, want %q", result.Object, tt.expected.Object)
			}

			if result.Model != tt.expected.Model {
				t.Errorf("Model mismatch: got %q, want %q", result.Model, tt.expected.Model)
			}

			if len(result.Choices) != len(tt.expected.Choices) {
				t.Fatalf("Choices length mismatch: got %d, want %d", len(result.Choices), len(tt.expected.Choices))
			}

			for i, choice := range result.Choices {
				if choice.Index != tt.expected.Choices[i].Index {
					t.Errorf("Choice[%d].Index mismatch: got %d, want %d", i, choice.Index, tt.expected.Choices[i].Index)
				}
				if choice.Message.Role != tt.expected.Choices[i].Message.Role {
					t.Errorf("Choice[%d].Message.Role mismatch: got %q, want %q", i, choice.Message.Role, tt.expected.Choices[i].Message.Role)
				}
				if choice.Message.Content != tt.expected.Choices[i].Message.Content {
					t.Errorf("Choice[%d].Message.Content mismatch: got %q, want %q", i, choice.Message.Content, tt.expected.Choices[i].Message.Content)
				}
			}

			if result.Usage.PromptTokens != tt.expected.Usage.PromptTokens {
				t.Errorf("Usage.PromptTokens mismatch: got %d, want %d", result.Usage.PromptTokens, tt.expected.Usage.PromptTokens)
			}
			if result.Usage.CompletionTokens != tt.expected.Usage.CompletionTokens {
				t.Errorf("Usage.CompletionTokens mismatch: got %d, want %d", result.Usage.CompletionTokens, tt.expected.Usage.CompletionTokens)
			}
			if result.Usage.TotalTokens != tt.expected.Usage.TotalTokens {
				t.Errorf("Usage.TotalTokens mismatch: got %d, want %d", result.Usage.TotalTokens, tt.expected.Usage.TotalTokens)
			}
		})
	}
}

func TestStreamTransformer_TransformChunk(t *testing.T) {
	tests := []struct {
		name     string
		input    OpenAIStreamResponse
		expected models.ChatCompletionResponse
	}{
		{
			name: "first chunk with role",
			input: OpenAIStreamResponse{
				ID:      "chatcmpl-stream-123",
				Object:  "chat.completion.chunk",
				Created: 1677652288,
				Model:   "gpt-4o",
				Choices: []OpenAIStreamChoice{
					{
						Index: 0,
						Delta: OpenAIMessageDelta{
							Role: "assistant",
						},
						FinishReason: nil,
					},
				},
			},
			expected: models.ChatCompletionResponse{
				ID:     "chatcmpl-stream-123",
				Object: "chat.completion.chunk",
				Model:  "gpt-4o",
				Choices: []models.ChatChoice{
					{
						Index: 0,
						Message: models.Message{
							Role:    "assistant",
							Content: "",
						},
					},
				},
			},
		},
		{
			name: "content chunk",
			input: OpenAIStreamResponse{
				ID:      "chatcmpl-stream-456",
				Object:  "chat.completion.chunk",
				Created: 1677652289,
				Model:   "gpt-4o",
				Choices: []OpenAIStreamChoice{
					{
						Index: 0,
						Delta: OpenAIMessageDelta{
							Content: "Hello",
						},
						FinishReason: nil,
					},
				},
			},
			expected: models.ChatCompletionResponse{
				ID:     "chatcmpl-stream-456",
				Object: "chat.completion.chunk",
				Model:  "gpt-4o",
				Choices: []models.ChatChoice{
					{
						Index: 0,
						Message: models.Message{
							Role:    "",
							Content: "Hello",
						},
					},
				},
			},
		},
		{
			name: "finish chunk",
			input: OpenAIStreamResponse{
				ID:      "chatcmpl-stream-789",
				Object:  "chat.completion.chunk",
				Created: 1677652290,
				Model:   "gpt-4o",
				Choices: []OpenAIStreamChoice{
					{
						Index: 0,
						Delta: OpenAIMessageDelta{},
						FinishReason: func() *string { s := "stop"; return &s }(),
					},
				},
			},
			expected: models.ChatCompletionResponse{
				ID:     "chatcmpl-stream-789",
				Object: "chat.completion.chunk",
				Model:  "gpt-4o",
				Choices: []models.ChatChoice{
					{
						Index: 0,
						Message: models.Message{
							Role:    "",
							Content: "",
						},
					},
				},
			},
		},
	}

	transformer := NewStreamTransformer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformer.TransformChunk(tt.input)

			if result.ID != tt.expected.ID {
				t.Errorf("ID mismatch: got %q, want %q", result.ID, tt.expected.ID)
			}

			if result.Object != tt.expected.Object {
				t.Errorf("Object mismatch: got %q, want %q", result.Object, tt.expected.Object)
			}

			if result.Model != tt.expected.Model {
				t.Errorf("Model mismatch: got %q, want %q", result.Model, tt.expected.Model)
			}

			if len(result.Choices) != len(tt.expected.Choices) {
				t.Fatalf("Choices length mismatch: got %d, want %d", len(result.Choices), len(tt.expected.Choices))
			}

			for i, choice := range result.Choices {
				if choice.Index != tt.expected.Choices[i].Index {
					t.Errorf("Choice[%d].Index mismatch: got %d, want %d", i, choice.Index, tt.expected.Choices[i].Index)
				}
				if choice.Message.Role != tt.expected.Choices[i].Message.Role {
					t.Errorf("Choice[%d].Message.Role mismatch: got %q, want %q", i, choice.Message.Role, tt.expected.Choices[i].Message.Role)
				}
				if choice.Message.Content != tt.expected.Choices[i].Message.Content {
					t.Errorf("Choice[%d].Message.Content mismatch: got %q, want %q", i, choice.Message.Content, tt.expected.Choices[i].Message.Content)
				}
			}
		})
	}
}

func TestParseSSE(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "single chunk",
			input: `data: {"id":"1","choices":[{"delta":{"content":"Hello"}}]}

`,
			expected: []string{`{"id":"1","choices":[{"delta":{"content":"Hello"}}]}`},
		},
		{
			name: "multiple chunks",
			input: `data: {"id":"1","choices":[{"delta":{"content":"Hello"}}]}

data: {"id":"2","choices":[{"delta":{"content":" World"}}]}

data: [DONE]

`,
			expected: []string{
				`{"id":"1","choices":[{"delta":{"content":"Hello"}}]}`,
				`{"id":"2","choices":[{"delta":{"content":" World"}}]}`,
			},
		},
		{
			name: "multiline data",
			input: `data: {"id":"1","content":"line1

line2"}

data: [DONE]

`,
			expected: []string{`{"id":"1","content":"line1

line2"}`},
		},
		{
			name:     "empty stream",
			input:    "",
			expected: []string{},
		},
		{
			name:     "only done marker",
			input:    "data: [DONE]\n\n",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSSE(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("Result length mismatch: got %d, want %d", len(result), len(tt.expected))
			}

			for i, chunk := range result {
				if chunk != tt.expected[i] {
					t.Errorf("Chunk[%d] mismatch: got %q, want %q", i, chunk, tt.expected[i])
				}
			}
		})
	}
}

func TestParseStreamChunk(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		check       func(t *testing.T, chunk *OpenAIStreamResponse)
	}{
		{
			name:  "valid chunk",
			input: `{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
			check: func(t *testing.T, chunk *OpenAIStreamResponse) {
				if chunk.ID != "chatcmpl-123" {
					t.Errorf("ID mismatch: got %q", chunk.ID)
				}
				if len(chunk.Choices) != 1 {
					t.Errorf("Expected 1 choice, got %d", len(chunk.Choices))
				}
				if chunk.Choices[0].Delta.Content != "Hello" {
					t.Errorf("Content mismatch: got %q", chunk.Choices[0].Delta.Content)
				}
			},
		},
		{
			name:        "invalid json",
			input:       `not valid json`,
			expectError: true,
		},
		{
			name:  "empty delta",
			input: `{"id":"123","choices":[{"delta":{}}]}`,
			check: func(t *testing.T, chunk *OpenAIStreamResponse) {
				if len(chunk.Choices) != 1 {
					t.Fatalf("Expected 1 choice, got %d", len(chunk.Choices))
				}
				if chunk.Choices[0].Delta.Content != "" {
					t.Errorf("Expected empty content, got %q", chunk.Choices[0].Delta.Content)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk, err := ParseStreamChunk(tt.input)

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

func TestOpenAIError_Error(t *testing.T) {
	err := &OpenAIError{
		Message: "Invalid API key provided",
		Type:    "invalid_request_error",
		Code:    "invalid_api_key",
	}

	expected := "openai error (invalid_request_error): Invalid API key provided"
	if err.Error() != expected {
		t.Errorf("Error message mismatch: got %q, want %q", err.Error(), expected)
	}
}
