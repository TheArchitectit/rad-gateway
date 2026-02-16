// Package anthropic provides an adapter for Anthropic's Claude API.
package anthropic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"radgateway/internal/models"
)

// AnthropicRequest represents the request payload sent to Anthropic API.
type AnthropicRequest struct {
	Model         string              `json:"model"`
	System        string              `json:"system,omitempty"`
	Messages      []AnthropicMessage  `json:"messages"`
	MaxTokens     int                 `json:"max_tokens"`
	Metadata      *AnthropicMetadata  `json:"metadata,omitempty"`
	StopSequences []string            `json:"stop_sequences,omitempty"`
	Stream        bool                `json:"stream,omitempty"`
	Temperature   *float64            `json:"temperature,omitempty"`
	TopP          *float64            `json:"top_p,omitempty"`
	TopK          *int                `json:"top_k,omitempty"`
}

// AnthropicMessage represents a message in Anthropic format.
// Note: Anthropic only supports "user" and "assistant" roles in messages.
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicMetadata represents metadata for the request.
type AnthropicMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

// AnthropicResponse represents a non-streaming response from Anthropic API.
type AnthropicResponse struct {
	ID           string                `json:"id"`
	Type         string                `json:"type"` // "message"
	Role         string                `json:"role"` // "assistant"
	Model        string                `json:"model"`
	Content      []AnthropicContentBlock `json:"content"`
	StopReason   string                `json:"stop_reason"`   // "end_turn", "max_tokens", "stop_sequence"
	StopSequence *string               `json:"stop_sequence"`
	Usage        AnthropicUsage        `json:"usage"`
}

// AnthropicContentBlock represents a content block in the response.
type AnthropicContentBlock struct {
	Type string `json:"type"` // "text", "tool_use", "tool_result"
	Text string `json:"text,omitempty"`
	// Additional fields for tool_use/tool_result omitted for simplicity
}

// AnthropicUsage represents token usage information.
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicError represents an error response from Anthropic API.
type AnthropicError struct {
	Type    string               `json:"type"`
	ErrorInfo AnthropicErrorInfo `json:"error"`
}

// AnthropicErrorInfo contains the error details.
type AnthropicErrorInfo struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Error implements the error interface for AnthropicError.
func (e *AnthropicError) Error() string {
	return fmt.Sprintf("anthropic %s: %s", e.ErrorInfo.Type, e.ErrorInfo.Message)
}

// AnthropicStreamEvent represents a streaming event from Anthropic.
type AnthropicStreamEvent struct {
	Type string `json:"type"`
}

// AnthropicMessageStartEvent represents the message_start streaming event.
type AnthropicMessageStartEvent struct {
	Type    string              `json:"type"`
	Message AnthropicStreamMessage `json:"message"`
}

// AnthropicStreamMessage represents the message in a streaming event.
type AnthropicStreamMessage struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"`
	Role         string                  `json:"role"`
	Model        string                  `json:"model"`
	Content      []AnthropicContentBlock `json:"content"`
	StopReason   *string                 `json:"stop_reason"`
	StopSequence *string                 `json:"stop_sequence"`
	Usage        *AnthropicUsage         `json:"usage"`
}

// AnthropicContentBlockStartEvent represents the content_block_start streaming event.
type AnthropicContentBlockStartEvent struct {
	Type         string                `json:"type"`
	Index        int                   `json:"index"`
	ContentBlock AnthropicContentBlock `json:"content_block"`
}

// AnthropicContentBlockDeltaEvent represents the content_block_delta streaming event.
type AnthropicContentBlockDeltaEvent struct {
	Type  string                `json:"type"`
	Index int                   `json:"index"`
	Delta AnthropicContentDelta `json:"delta"`
}

// AnthropicContentDelta represents the delta in a content_block_delta event.
type AnthropicContentDelta struct {
	Type string `json:"type"` // "text_delta"
	Text string `json:"text"`
}

// AnthropicContentBlockStopEvent represents the content_block_stop streaming event.
type AnthropicContentBlockStopEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

// AnthropicMessageDeltaEvent represents the message_delta streaming event.
type AnthropicMessageDeltaEvent struct {
	Type  string              `json:"type"`
	Delta AnthropicMessageDelta `json:"delta"`
	Usage *AnthropicUsage       `json:"usage,omitempty"`
}

// AnthropicMessageDelta represents the delta in a message_delta event.
type AnthropicMessageDelta struct {
	StopReason   string  `json:"stop_reason,omitempty"`
	StopSequence *string `json:"stop_sequence,omitempty"`
}

// AnthropicMessageStopEvent represents the message_stop streaming event.
type AnthropicMessageStopEvent struct {
	Type string `json:"type"`
}

// RequestTransformer transforms internal requests to Anthropic format.
type RequestTransformer struct{}

// NewRequestTransformer creates a new RequestTransformer.
func NewRequestTransformer() *RequestTransformer {
	return &RequestTransformer{}
}

// Transform converts internal ChatCompletionRequest to Anthropic format.
// Key transformations:
// 1. Extracts system messages from messages array into separate "system" field
// 2. Removes system messages from messages array (Anthropic uses separate field)
// 3. Merges consecutive same-role messages (Anthropic doesn't allow consecutive same-role)
// 4. Sets default max_tokens if not provided (Anthropic requires this)
func (t *RequestTransformer) Transform(req models.ChatCompletionRequest) AnthropicRequest {
	anthropicReq := AnthropicRequest{
		Model: req.Model,
	}

	// Set default max_tokens if not provided (Anthropic requires this)
	if req.MaxTokens > 0 {
		anthropicReq.MaxTokens = req.MaxTokens
	} else {
		anthropicReq.MaxTokens = 4096 // Default value
	}

	// Set stream flag
	anthropicReq.Stream = req.Stream

	// Transform messages
	var systemContent strings.Builder
	var messages []AnthropicMessage
	var lastRole string

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			// Accumulate system messages
			if systemContent.Len() > 0 {
				systemContent.WriteString("\n")
			}
			systemContent.WriteString(msg.Content)
		} else if msg.Role == "user" || msg.Role == "assistant" {
			// Merge consecutive same-role messages
			if len(messages) > 0 && lastRole == msg.Role {
				messages[len(messages)-1].Content += "\n" + msg.Content
			} else {
				messages = append(messages, AnthropicMessage{
					Role:    msg.Role,
					Content: msg.Content,
				})
				lastRole = msg.Role
			}
		}
		// Ignore other roles (Anthropic only supports user/assistant in messages)
	}

	anthropicReq.Messages = messages

	// Set system content if any
	if systemContent.Len() > 0 {
		anthropicReq.System = systemContent.String()
	}

	// Map optional parameters
	if req.Temperature != 0 {
		temp := req.Temperature
		anthropicReq.Temperature = &temp
	}
	if req.TopP != 0 {
		topP := req.TopP
		anthropicReq.TopP = &topP
	}

	// Map stop sequences
	if len(req.Stop) > 0 {
		anthropicReq.StopSequences = req.Stop
	}

	// Map user ID to metadata
	if req.User != "" {
		anthropicReq.Metadata = &AnthropicMetadata{
			UserID: req.User,
		}
	}

	return anthropicReq
}

// ResponseTransformer transforms Anthropic responses to internal format.
type ResponseTransformer struct{}

// NewResponseTransformer creates a new ResponseTransformer.
func NewResponseTransformer() *ResponseTransformer {
	return &ResponseTransformer{}
}

// Transform converts Anthropic response to internal ChatCompletionResponse.
func (t *ResponseTransformer) Transform(resp AnthropicResponse) (models.ChatCompletionResponse, error) {
	// Check for error response
	if resp.Type == "error" {
		return models.ChatCompletionResponse{}, fmt.Errorf("anthropic API error")
	}

	// Aggregate content from content blocks
	var content strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			content.WriteString(block.Text)
		}
		// Note: tool_use and tool_result blocks would be handled separately
	}

	// Map stop_reason to finish_reason
	finishReason := mapStopReason(resp.StopReason)

	return models.ChatCompletionResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   resp.Model,
		Choices: []models.ChatChoice{
			{
				Index: 0,
				Message: models.Message{
					Role:    "assistant",
					Content: content.String(),
				},
				FinishReason: finishReason,
			},
		},
		Usage: models.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}, nil
}

// mapStopReason maps Anthropic stop_reason to OpenAI-compatible finish_reason.
func mapStopReason(anthropicReason string) string {
	switch anthropicReason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	default:
		return "stop"
	}
}

// StreamTransformer handles streaming response transformation.
type StreamTransformer struct {
	messageID    string
	model        string
	created      int64
	index        int
	contentBuffer strings.Builder
}

// NewStreamTransformer creates a new StreamTransformer.
func NewStreamTransformer() *StreamTransformer {
	return &StreamTransformer{
		index: 0,
	}
}

// Reset resets the transformer state for a new stream.
func (t *StreamTransformer) Reset() {
	t.messageID = ""
	t.model = ""
	t.created = 0
	t.index = 0
	t.contentBuffer.Reset()
}

// TransformEvent transforms an Anthropic SSE event to OpenAI-compatible format.
// Returns the transformed SSE data and a boolean indicating if this is a done marker.
func (t *StreamTransformer) TransformEvent(eventType string, data []byte) ([]byte, bool, error) {
	switch eventType {
	case "message_start":
		return t.handleMessageStart(data)
	case "content_block_start":
		return t.handleContentBlockStart(data)
	case "content_block_delta":
		return t.handleContentBlockDelta(data)
	case "content_block_stop":
		return t.handleContentBlockStop(data)
	case "message_delta":
		return t.handleMessageDelta(data)
	case "message_stop":
		return t.handleMessageStop()
	case "ping":
		// Ignore ping events
		return nil, false, nil
	case "error":
		return nil, false, fmt.Errorf("anthropic streaming error: %s", string(data))
	default:
		// Unknown event type, ignore
		return nil, false, nil
	}
}

func (t *StreamTransformer) handleMessageStart(data []byte) ([]byte, bool, error) {
	var event AnthropicMessageStartEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, false, fmt.Errorf("unmarshaling message_start: %w", err)
	}

	t.messageID = event.Message.ID
	t.model = event.Message.Model
	t.created = time.Now().Unix()

	// Return OpenAI-style start chunk with role
	chunk := models.ChatCompletionResponse{
		ID:      t.messageID,
		Object:  "chat.completion.chunk",
		Created: t.created,
		Model:   t.model,
		Choices: []models.ChatChoice{
			{
				Index: 0,
				Message: models.Message{
					Role:    "assistant",
					Content: "",
				},
			},
		},
	}

	chunkJSON, err := json.Marshal(chunk)
	if err != nil {
		return nil, false, err
	}

	return formatSSEData(chunkJSON), false, nil
}

func (t *StreamTransformer) handleContentBlockStart(data []byte) ([]byte, bool, error) {
	// Content block start - just track state, no output needed
	var event AnthropicContentBlockStartEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, false, fmt.Errorf("unmarshaling content_block_start: %w", err)
	}

	// Return empty chunk to signal we're still processing
	return nil, false, nil
}

func (t *StreamTransformer) handleContentBlockDelta(data []byte) ([]byte, bool, error) {
	var event AnthropicContentBlockDeltaEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, false, fmt.Errorf("unmarshaling content_block_delta: %w", err)
	}

	// Accumulate content for usage tracking
	if event.Delta.Text != "" {
		t.contentBuffer.WriteString(event.Delta.Text)
	}

	// Transform to OpenAI format
	chunk := models.ChatCompletionResponse{
		ID:      t.messageID,
		Object:  "chat.completion.chunk",
		Created: t.created,
		Model:   t.model,
		Choices: []models.ChatChoice{
			{
				Index: 0,
				Message: models.Message{
					Content: event.Delta.Text,
				},
			},
		},
	}

	chunkJSON, err := json.Marshal(chunk)
	if err != nil {
		return nil, false, err
	}

	return formatSSEData(chunkJSON), false, nil
}

func (t *StreamTransformer) handleContentBlockStop(data []byte) ([]byte, bool, error) {
	// Content block stop - just track state, no output needed
	return nil, false, nil
}

func (t *StreamTransformer) handleMessageDelta(data []byte) ([]byte, bool, error) {
	var event AnthropicMessageDeltaEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, false, fmt.Errorf("unmarshaling message_delta: %w", err)
	}

	finishReason := mapStopReason(event.Delta.StopReason)

	// Final chunk with finish_reason
	chunk := models.ChatCompletionResponse{
		ID:      t.messageID,
		Object:  "chat.completion.chunk",
		Created: t.created,
		Model:   t.model,
		Choices: []models.ChatChoice{
			{
				Index:        0,
				Message:      models.Message{},
				FinishReason: finishReason,
			},
		},
	}

	chunkJSON, err := json.Marshal(chunk)
	if err != nil {
		return nil, false, err
	}

	return formatSSEData(chunkJSON), false, nil
}

func (t *StreamTransformer) handleMessageStop() ([]byte, bool, error) {
	// Return OpenAI [DONE] marker
	return []byte("data: [DONE]\n\n"), true, nil
}

// GetAccumulatedContent returns the accumulated content from the stream.
func (t *StreamTransformer) GetAccumulatedContent() string {
	return t.contentBuffer.String()
}

// formatSSEData formats data as an SSE data line.
func formatSSEData(data []byte) []byte {
	return []byte(fmt.Sprintf("data: %s\n\n", string(data)))
}

// ParseSSEEvent parses an SSE event from a line.
// Returns the event type and data.
func ParseSSEEvent(line string) (string, string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", ""
	}

	// Check for event: prefix
	if strings.HasPrefix(line, "event: ") {
		return strings.TrimPrefix(line, "event: "), ""
	}

	// Check for data: prefix
	if strings.HasPrefix(line, "data: ") {
		return "", strings.TrimPrefix(line, "data: ")
	}

	return "", ""
}

// TransformErrorResponse transforms an Anthropic error response.
func TransformErrorResponse(body []byte) error {
	var anthropicErr AnthropicError
	if err := json.Unmarshal(body, &anthropicErr); err == nil && anthropicErr.ErrorInfo.Type != "" {
		return &anthropicErr
	}

	return fmt.Errorf("anthropic error: %s", string(body))
}

// ParseStreamChunk parses a JSON chunk from a streaming response.
// This is used for non-OpenAI formats that need transformation.
func ParseStreamChunk(data string) (*models.ChatCompletionResponse, error) {
	var chunk models.ChatCompletionResponse
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return nil, fmt.Errorf("unmarshaling stream chunk: %w", err)
	}
	return &chunk, nil
}

// IsDoneMarker checks if the chunk is a done marker.
func IsDoneMarker(chunk []byte) bool {
	return bytes.Contains(chunk, []byte("event: message_stop")) ||
		bytes.Contains(chunk, []byte("data: [DONE]"))
}
