package streaming

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Chunk represents a unified streaming chunk from any provider.
type Chunk struct {
	// ID is the chunk identifier (e.g., chat.completion.chunk ID)
	ID string `json:"id,omitempty"`

	// Object is the type of object (e.g., "chat.completion.chunk")
	Object string `json:"object,omitempty"`

	// Model is the model name used
	Model string `json:"model,omitempty"`

	// Created is the Unix timestamp
	Created int64 `json:"created,omitempty"`

	// Choices contains the content deltas
	Choices []ChunkChoice `json:"choices,omitempty"`

	// Usage contains token usage information (may be present on final chunk)
	Usage *Usage `json:"usage,omitempty"`

	// IsFinished indicates this is the final chunk
	IsFinished bool `json:"-"`

	// FinishReason indicates why the stream finished
	FinishReason string `json:"-"`

	// Error contains any error that occurred
	Error error `json:"-"`
}

// ChunkChoice represents a choice within a chunk.
type ChunkChoice struct {
	Index int `json:"index"`

	// Delta contains the incremental content
	Delta Delta `json:"delta"`

	// FinishReason is set on the final chunk
	FinishReason *string `json:"finish_reason,omitempty"`
}

// Delta represents the incremental content in a chunk.
type Delta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// Usage represents token usage information.
type Usage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	CostTotal        float64 `json:"cost_total,omitempty"`
}

// Transformer transforms provider-specific SSE events into unified chunks.
type Transformer struct {
	provider string
	model    string
}

// NewTransformer creates a new stream transformer for a specific provider.
func NewTransformer(provider, model string) *Transformer {
	return &Transformer{
		provider: provider,
		model:    model,
	}
}

// Transform converts a provider-specific SSE event into a unified chunk.
// Returns nil for events that should be skipped (like keepalives).
func (t *Transformer) Transform(event Event) (*Chunk, error) {
	// Skip empty events and comments
	if event.Data == "" {
		return nil, nil
	}

	// Handle different provider formats
	switch t.provider {
	case "openai", "azure":
		return t.transformOpenAI(event)
	case "anthropic":
		return t.transformAnthropic(event)
	case "gemini":
		return t.transformGemini(event)
	default:
		// Default to OpenAI format
		return t.transformOpenAI(event)
	}
}

// transformOpenAI transforms OpenAI-style SSE events.
func (t *Transformer) transformOpenAI(event Event) (*Chunk, error) {
	// OpenAI streams end with "[DONE]"
	if event.Data == "[DONE]" {
		return &Chunk{IsFinished: true}, nil
	}

	// Check for error events
	if event.Event == "error" {
		return nil, fmt.Errorf("provider error: %s", event.Data)
	}

	var chunk Chunk
	if err := json.Unmarshal([]byte(event.Data), &chunk); err != nil {
		return nil, fmt.Errorf("unmarshal openai chunk: %w", err)
	}

	// Ensure model is set
	if chunk.Model == "" {
		chunk.Model = t.model
	}

	// Check if this is the final chunk
	if len(chunk.Choices) > 0 && chunk.Choices[0].FinishReason != nil {
		chunk.IsFinished = true
		chunk.FinishReason = *chunk.Choices[0].FinishReason
	}

	return &chunk, nil
}

// AnthropicStreamEvent represents Anthropic's streaming event structure.
type AnthropicStreamEvent struct {
	Type         string          `json:"type"`
	Message      *AnthropicMsg   `json:"message,omitempty"`
	ContentBlock *AnthropicBlock `json:"content_block,omitempty"`
	Delta        *AnthropicDelta `json:"delta,omitempty"`
	Usage        *Usage          `json:"usage,omitempty"`
	StopReason   *string         `json:"stop_reason,omitempty"`
}

// AnthropicMsg represents the message start event.
type AnthropicMsg struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Model   string `json:"model"`
	Content []any  `json:"content"`
}

// AnthropicBlock represents a content block.
type AnthropicBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// AnthropicDelta represents a delta update.
type AnthropicDelta struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// transformAnthropic transforms Anthropic-style SSE events.
func (t *Transformer) transformAnthropic(event Event) (*Chunk, error) {
	var ae AnthropicStreamEvent
	if err := json.Unmarshal([]byte(event.Data), &ae); err != nil {
		return nil, fmt.Errorf("unmarshal anthropic event: %w", err)
	}

	switch ae.Type {
	case "message_start":
		if ae.Message == nil {
			return nil, nil
		}
		return &Chunk{
			ID:     ae.Message.ID,
			Object: "chat.completion.chunk",
			Model:  ae.Message.Model,
			Choices: []ChunkChoice{
				{
					Index: 0,
					Delta: Delta{Role: ae.Message.Role},
				},
			},
		}, nil

	case "content_block_delta":
		if ae.Delta == nil {
			return nil, nil
		}
		return &Chunk{
			Object: "chat.completion.chunk",
			Model:  t.model,
			Choices: []ChunkChoice{
				{
					Index: 0,
					Delta: Delta{Content: ae.Delta.Text},
				},
			},
		}, nil

	case "content_block_stop", "message_delta":
		// These are intermediate events we can skip for the unified format
		return nil, nil

	case "message_stop":
		chunk := &Chunk{
			Object:     "chat.completion.chunk",
			Model:      t.model,
			IsFinished: true,
			Choices: []ChunkChoice{
				{
					Index:        0,
					Delta:        Delta{},
					FinishReason: strPtr("stop"),
				},
			},
		}
		if ae.Usage != nil {
			chunk.Usage = ae.Usage
		}
		return chunk, nil

	case "ping":
		// Keepalive, ignore
		return nil, nil

	case "error":
		return nil, fmt.Errorf("anthropic streaming error: %s", event.Data)

	default:
		// Unknown event type, skip
		return nil, nil
	}
}

// GeminiStreamEvent represents Gemini's streaming event structure.
type GeminiStreamEvent struct {
	Candidates []GeminiCandidate `json:"candidates,omitempty"`
	Usage      *GeminiUsage      `json:"usageMetadata,omitempty"`
}

// GeminiCandidate represents a response candidate.
type GeminiCandidate struct {
	Index        int                `json:"index"`
	Content      *GeminiContent     `json:"content,omitempty"`
	FinishReason string             `json:"finishReason,omitempty"`
}

// GeminiContent represents message content.
type GeminiContent struct {
	Parts []GeminiPart `json:"parts,omitempty"`
	Role  string       `json:"role,omitempty"`
}

// GeminiPart represents a content part.
type GeminiPart struct {
	Text string `json:"text,omitempty"`
}

// GeminiUsage represents token usage.
type GeminiUsage struct {
	PromptTokens     int `json:"promptTokenCount"`
	CompletionTokens int `json:"candidatesTokenCount"`
	TotalTokens      int `json:"totalTokenCount"`
}

// transformGemini transforms Gemini-style SSE events.
func (t *Transformer) transformGemini(event Event) (*Chunk, error) {
	var ge GeminiStreamEvent
	if err := json.Unmarshal([]byte(event.Data), &ge); err != nil {
		return nil, fmt.Errorf("unmarshal gemini event: %w", err)
	}

	if len(ge.Candidates) == 0 {
		return nil, nil
	}

	candidate := ge.Candidates[0]
	chunk := &Chunk{
		Object: "chat.completion.chunk",
		Model:  t.model,
		Choices: []ChunkChoice{
			{
				Index: candidate.Index,
			},
		},
	}

	// Extract text content
	if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
		var content strings.Builder
		for _, part := range candidate.Content.Parts {
			content.WriteString(part.Text)
		}
		chunk.Choices[0].Delta.Content = content.String()

		if candidate.Content.Role != "" {
			chunk.Choices[0].Delta.Role = candidate.Content.Role
		}
	}

	// Check for finish
	if candidate.FinishReason != "" {
		chunk.IsFinished = true
		chunk.FinishReason = mapGeminiFinishReason(candidate.FinishReason)
		chunk.Choices[0].FinishReason = strPtr(chunk.FinishReason)
	}

	// Map usage if present
	if ge.Usage != nil {
		chunk.Usage = &Usage{
			PromptTokens:     ge.Usage.PromptTokens,
			CompletionTokens: ge.Usage.CompletionTokens,
			TotalTokens:      ge.Usage.TotalTokens,
		}
	}

	return chunk, nil
}

// mapGeminiFinishReason maps Gemini finish reasons to OpenAI-style.
func mapGeminiFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY", "RECITATION", "OTHER":
		return "content_filter"
	default:
		return "stop"
	}
}

// ToOpenAIFormat converts a unified chunk to OpenAI's streaming format.
func (c *Chunk) ToOpenAIFormat() map[string]any {
	choices := make([]map[string]any, len(c.Choices))
	for i, choice := range c.Choices {
		choiceMap := map[string]any{
			"index": choice.Index,
			"delta": map[string]any{
				"role":    choice.Delta.Role,
				"content": choice.Delta.Content,
			},
		}
		if choice.FinishReason != nil {
			choiceMap["finish_reason"] = *choice.FinishReason
		} else {
			choiceMap["finish_reason"] = nil
		}
		choices[i] = choiceMap
	}

	result := map[string]any{
		"id":      c.ID,
		"object":  c.Object,
		"created": c.Created,
		"model":   c.Model,
		"choices": choices,
	}

	if c.Usage != nil {
		result["usage"] = c.Usage
	}

	return result
}

// TransformStream transforms a provider SSE stream into unified chunks.
// It reads from the parser, transforms each event, and sends chunks to the output channel.
func TransformStream(parser *Parser, transformer *Transformer, output chan<- *Chunk, done chan<- error) {
	defer close(output)

	for {
		event, err := parser.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				done <- nil
				return
			}
			done <- err
			return
		}

		chunk, err := transformer.Transform(event)
		if err != nil {
			done <- err
			return
		}

		if chunk != nil {
			output <- chunk
		}
	}
}

// strPtr returns a pointer to a string.
func strPtr(s string) *string {
	return &s
}
