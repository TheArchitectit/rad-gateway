package streaming

import (
	"encoding/json"
	"fmt"
)

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
