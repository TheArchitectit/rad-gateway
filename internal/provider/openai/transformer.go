// Package openai provides an adapter for OpenAI-compatible API providers.
package openai

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"radgateway/internal/models"
)

// OpenAIRequest represents the request payload sent to OpenAI API.
type OpenAIRequest struct {
	Model            string          `json:"model"`
	Messages         []OpenAIMessage `json:"messages"`
	Stream           bool            `json:"stream,omitempty"`
	Temperature      float64         `json:"temperature,omitempty"`
	MaxTokens        int             `json:"max_tokens,omitempty"`
	TopP             float64         `json:"top_p,omitempty"`
	FrequencyPenalty float64         `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64         `json:"presence_penalty,omitempty"`
	User             string          `json:"user,omitempty"`
}

// OpenAIMessage represents a message in OpenAI format.
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents a non-streaming response from OpenAI API.
type OpenAIResponse struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
	Choices []OpenAIChoice   `json:"choices"`
	Usage   OpenAIUsage      `json:"usage"`
	Error   *OpenAIError     `json:"error,omitempty"`
}

// OpenAIChoice represents a choice in the OpenAI response.
type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// OpenAIUsage represents token usage information.
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIError represents an error response from OpenAI API.
type OpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// OpenAIStreamResponse represents a streaming response chunk from OpenAI API.
type OpenAIStreamResponse struct {
	ID      string                `json:"id"`
	Object  string                `json:"object"`
	Created int64                 `json:"created"`
	Model   string                `json:"model"`
	Choices []OpenAIStreamChoice  `json:"choices"`
}

// OpenAIStreamChoice represents a choice in a streaming response.
type OpenAIStreamChoice struct {
	Index        int                `json:"index"`
	Delta        OpenAIMessageDelta `json:"delta"`
	FinishReason *string            `json:"finish_reason"`
}

// OpenAIMessageDelta represents a message delta in streaming response.
type OpenAIMessageDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// Error implements the error interface for OpenAIError.
func (e *OpenAIError) Error() string {
	return fmt.Sprintf("openai error (%s): %s", e.Type, e.Message)
}

// RequestTransformer transforms internal request to OpenAI format.
type RequestTransformer struct{}

// NewRequestTransformer creates a new RequestTransformer.
func NewRequestTransformer() *RequestTransformer {
	return &RequestTransformer{}
}

// Transform converts internal ChatCompletionRequest to OpenAI format.
func (t *RequestTransformer) Transform(req models.ChatCompletionRequest) OpenAIRequest {
	messages := make([]OpenAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = OpenAIMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	return OpenAIRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   req.Stream,
		User:     req.User,
	}
}

// ResponseTransformer transforms OpenAI responses to internal format.
type ResponseTransformer struct{}

// NewResponseTransformer creates a new ResponseTransformer.
func NewResponseTransformer() *ResponseTransformer {
	return &ResponseTransformer{}
}

// Transform converts OpenAI response to internal ChatCompletionResponse.
func (t *ResponseTransformer) Transform(resp OpenAIResponse) (models.ChatCompletionResponse, error) {
	if resp.Error != nil {
		return models.ChatCompletionResponse{}, resp.Error
	}

	choices := make([]models.ChatChoice, len(resp.Choices))
	for i, c := range resp.Choices {
		choices[i] = models.ChatChoice{
			Index: c.Index,
			Message: models.Message{
				Role:    c.Message.Role,
				Content: c.Message.Content,
			},
		}
	}

	return models.ChatCompletionResponse{
		ID:      resp.ID,
		Object:  resp.Object,
		Model:   resp.Model,
		Choices: choices,
		Usage: models.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// StreamTransformer handles streaming response transformation.
type StreamTransformer struct{}

// NewStreamTransformer creates a new StreamTransformer.
func NewStreamTransformer() *StreamTransformer {
	return &StreamTransformer{}
}

// TransformChunk converts a single SSE chunk to internal format.
func (t *StreamTransformer) TransformChunk(chunk OpenAIStreamResponse) models.ChatCompletionResponse {
	choices := make([]models.ChatChoice, len(chunk.Choices))
	for i, c := range chunk.Choices {
		choices[i] = models.ChatChoice{
			Index: c.Index,
			Message: models.Message{
				Role:    c.Delta.Role,
				Content: c.Delta.Content,
			},
		}
	}

	return models.ChatCompletionResponse{
		ID:      chunk.ID,
		Object:  chunk.Object,
		Model:   chunk.Model,
		Choices: choices,
	}
}

// ParseSSE parses a Server-Sent Events stream and returns individual data chunks.
func ParseSSE(reader io.Reader) ([]string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading sse stream: %w", err)
	}

	var chunks []string
	lines := strings.Split(string(data), "\n")

	var currentData strings.Builder
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "data: ") {
			if currentData.Len() > 0 {
				currentData.WriteString("\n")
			}
			currentData.WriteString(strings.TrimPrefix(line, "data: "))
		} else if line == "" {
			if currentData.Len() > 0 {
				data := currentData.String()
				if data != "[DONE]" {
					chunks = append(chunks, data)
				}
				currentData.Reset()
			}
		}
	}

	// Handle any remaining data
	if currentData.Len() > 0 {
		data := currentData.String()
		if data != "[DONE]" {
			chunks = append(chunks, data)
		}
	}

	return chunks, nil
}

// ParseStreamChunk parses a single JSON chunk from a streaming response.
func ParseStreamChunk(data string) (*OpenAIStreamResponse, error) {
	var chunk OpenAIStreamResponse
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return nil, fmt.Errorf("unmarshaling stream chunk: %w", err)
	}
	return &chunk, nil
}
