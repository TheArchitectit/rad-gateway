package streaming

import (
	"testing"
)

func TestTransformer_Transform_OpenAI(t *testing.T) {
	transformer := NewTransformer("openai", "gpt-4")

	tests := []struct {
		name     string
		event    Event
		wantNil  bool
		wantErr  bool
		checkFn  func(*Chunk) bool
		checkMsg string
	}{
		{
			name: "normal completion chunk",
			event: Event{
				Data: `{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}`,
			},
			checkFn: func(c *Chunk) bool {
				return c.ID == "chatcmpl-123" && c.Object == "chat.completion.chunk" &&
					c.Model == "gpt-4" && len(c.Choices) == 1 &&
					c.Choices[0].Delta.Content == "hello"
			},
			checkMsg: "should parse OpenAI completion chunk correctly",
		},
		{
			name: "done marker",
			event: Event{
				Data: "[DONE]",
			},
			checkFn: func(c *Chunk) bool {
				return c.IsFinished
			},
			checkMsg: "should mark [DONE] as finished",
		},
		{
			name:     "empty data",
			event:    Event{Data: ""},
			wantNil:  true,
			checkMsg: "should return nil for empty data",
		},
		{
			name: "final chunk with finish_reason",
			event: Event{
				Data: `{"id":"chatcmpl-123","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			},
			checkFn: func(c *Chunk) bool {
				return c.IsFinished && c.FinishReason == "stop"
			},
			checkMsg: "should detect finish_reason",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk, err := transformer.Transform(tt.event)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantNil && chunk != nil {
				t.Error("expected nil chunk")
			}
			if !tt.wantNil && chunk == nil {
				t.Error("expected non-nil chunk")
			}
			if chunk != nil && tt.checkFn != nil && !tt.checkFn(chunk) {
				t.Error(tt.checkMsg)
			}
		})
	}
}

func TestTransformer_Transform_Anthropic(t *testing.T) {
	transformer := NewTransformer("anthropic", "claude-3")

	tests := []struct {
		name     string
		event    Event
		wantNil  bool
		checkFn  func(*Chunk) bool
		checkMsg string
	}{
		{
			name: "message_start",
			event: Event{
				Data: `{"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-sonnet-20240229"}}`,
			},
			checkFn: func(c *Chunk) bool {
				return c.ID == "msg_123" && c.Object == "chat.completion.chunk" &&
					len(c.Choices) == 1 && c.Choices[0].Delta.Role == "assistant"
			},
			checkMsg: "should parse message_start correctly",
		},
		{
			name: "content_block_delta",
			event: Event{
				Data: `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello world"}}`,
			},
			checkFn: func(c *Chunk) bool {
				return len(c.Choices) == 1 && c.Choices[0].Delta.Content == "Hello world"
			},
			checkMsg: "should extract text from content_block_delta",
		},
		{
			name: "message_stop",
			event: Event{
				Data: `{"type":"message_stop"}`,
			},
			checkFn: func(c *Chunk) bool {
				return c.IsFinished
			},
			checkMsg: "should detect message_stop as finished",
		},
		{
			name:    "ping",
			event:   Event{Data: `{"type":"ping"}`},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk, err := transformer.Transform(tt.event)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantNil && chunk != nil {
				t.Error("expected nil chunk")
			}
			if !tt.wantNil && chunk == nil {
				t.Error("expected non-nil chunk")
			}
			if chunk != nil && tt.checkFn != nil && !tt.checkFn(chunk) {
				t.Error(tt.checkMsg)
			}
		})
	}
}

func TestTransformer_Transform_Gemini(t *testing.T) {
	transformer := NewTransformer("gemini", "gemini-1.5-flash")

	tests := []struct {
		name     string
		event    Event
		wantNil  bool
		checkFn  func(*Chunk) bool
		checkMsg string
	}{
		{
			name: "normal response",
			event: Event{
				Data: `{"candidates":[{"index":0,"content":{"parts":[{"text":"Hello"}],"role":"model"}}]}`,
			},
			checkFn: func(c *Chunk) bool {
				return len(c.Choices) == 1 && c.Choices[0].Delta.Content == "Hello"
			},
			checkMsg: "should extract text from Gemini response",
		},
		{
			name: "with finish reason",
			event: Event{
				Data: `{"candidates":[{"index":0,"content":{"parts":[{"text":"Done"}]},"finishReason":"STOP"}]}`,
			},
			checkFn: func(c *Chunk) bool {
				return c.IsFinished && c.FinishReason == "stop"
			},
			checkMsg: "should map STOP to stop",
		},
		{
			name: "MAX_TOKENS finish",
			event: Event{
				Data: `{"candidates":[{"index":0,"finishReason":"MAX_TOKENS"}]}`,
			},
			checkFn: func(c *Chunk) bool {
				return c.IsFinished && c.FinishReason == "length"
			},
			checkMsg: "should map MAX_TOKENS to length",
		},
		{
			name: "with usage",
			event: Event{
				Data: `{"candidates":[{"index":0}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":20,"totalTokenCount":30}}`,
			},
			checkFn: func(c *Chunk) bool {
				return c.Usage != nil && c.Usage.PromptTokens == 10 &&
					c.Usage.CompletionTokens == 20 && c.Usage.TotalTokens == 30
			},
			checkMsg: "should extract usage metadata",
		},
		{
			name:    "empty candidates",
			event:   Event{Data: `{"candidates":[]}`},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk, err := transformer.Transform(tt.event)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantNil && chunk != nil {
				t.Error("expected nil chunk")
			}
			if !tt.wantNil && chunk == nil {
				t.Error("expected non-nil chunk")
			}
			if chunk != nil && tt.checkFn != nil && !tt.checkFn(chunk) {
				t.Errorf("%s: got chunk %+v", tt.checkMsg, chunk)
			}
		})
	}
}

func TestChunk_ToOpenAIFormat(t *testing.T) {
	chunk := &Chunk{
		ID:      "chatcmpl-123",
		Object:  "chat.completion.chunk",
		Created: 1234567890,
		Model:   "gpt-4",
		Choices: []ChunkChoice{
			{
				Index: 0,
				Delta: Delta{
					Content: "Hello",
				},
			},
		},
		Usage: &Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	result := chunk.ToOpenAIFormat()

	if result["id"] != "chatcmpl-123" {
		t.Errorf("id = %v, want chatcmpl-123", result["id"])
	}

	choices, ok := result["choices"].([]map[string]any)
	if !ok || len(choices) != 1 {
		t.Fatal("choices not in expected format")
	}

	delta, ok := choices[0]["delta"].(map[string]any)
	if !ok {
		t.Fatal("delta not in expected format")
	}

	if delta["content"] != "Hello" {
		t.Errorf("content = %v, want Hello", delta["content"])
	}
}

func TestMapGeminiFinishReason(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"STOP", "stop"},
		{"MAX_TOKENS", "length"},
		{"SAFETY", "content_filter"},
		{"RECITATION", "content_filter"},
		{"OTHER", "content_filter"},
		{"UNKNOWN", "stop"},
	}

	for _, tt := range tests {
		got := mapGeminiFinishReason(tt.input)
		if got != tt.expected {
			t.Errorf("mapGeminiFinishReason(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func BenchmarkTransformer_Transform(b *testing.B) {
	transformer := NewTransformer("openai", "gpt-4")
	event := Event{
		Data: `{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"hello world this is a test"},"finish_reason":null}]}`,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = transformer.Transform(event)
	}
}
