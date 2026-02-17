package streaming

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GeminiStreamEvent represents Gemini's streaming event structure.
type GeminiStreamEvent struct {
	Candidates []GeminiCandidate `json:"candidates,omitempty"`
	Usage      *GeminiUsage      `json:"usageMetadata,omitempty"`
}

// GeminiCandidate represents a response candidate.
type GeminiCandidate struct {
	Index        int            `json:"index"`
	Content      *GeminiContent `json:"content,omitempty"`
	FinishReason string         `json:"finishReason,omitempty"`
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
