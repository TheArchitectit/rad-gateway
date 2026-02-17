package streaming

import (
	"errors"
	"io"
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
				// Non-blocking send to prevent deadlock if done channel is full
				select {
				case done <- nil:
				default:
				}
				return
			}
			// Non-blocking send to prevent deadlock if done channel is full
			select {
			case done <- err:
			default:
			}
			return
		}

		chunk, err := transformer.Transform(event)
		if err != nil {
			// Non-blocking send to prevent deadlock if done channel is full
			select {
			case done <- err:
			default:
			}
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
