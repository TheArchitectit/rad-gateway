package streaming

import (
	"encoding/json"
	"fmt"
)

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
