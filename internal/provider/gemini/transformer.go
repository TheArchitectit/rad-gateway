// Package gemini provides an adapter for Google's Gemini API.
package gemini

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"radgateway/internal/models"
)

// ============================================================================
// Request Types
// ============================================================================

// GeminiRequest represents the top-level request structure for Gemini API.
type GeminiRequest struct {
	Contents         []GeminiContent    `json:"contents"`
	GenerationConfig *GenerationConfig  `json:"generationConfig,omitempty"`
	SafetySettings   []SafetySetting    `json:"safetySettings,omitempty"`
}

// GeminiContent represents content with a role and parts.
type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart represents a part of the content (text, inline data, etc.).
type GeminiPart struct {
	Text       string      `json:"text,omitempty"`
	InlineData *InlineData `json:"inlineData,omitempty"`
}

// InlineData represents inline binary data (for future multimodal support).
type InlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// GenerationConfig contains generation parameters.
type GenerationConfig struct {
	Temperature     float64  `json:"temperature,omitempty"`
	TopP            float64  `json:"topP,omitempty"`
	TopK            int      `json:"topK,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
	CandidateCount  int      `json:"candidateCount,omitempty"`
}

// SafetySetting defines safety thresholds for content generation.
type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// ============================================================================
// Response Types
// ============================================================================

// GeminiResponse represents the top-level response from Gemini API.
type GeminiResponse struct {
	Candidates     []GeminiCandidate   `json:"candidates"`
	UsageMetadata  UsageMetadata     `json:"usageMetadata"`
	PromptFeedback *GeminiPromptFeedback `json:"promptFeedback,omitempty"`
}

// GeminiCandidate represents a response candidate.
type GeminiCandidate struct {
	Content       GeminiContent        `json:"content"`
	FinishReason  string               `json:"finishReason"`
	Index         int                  `json:"index"`
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings,omitempty"`
}

// GeminiSafetyRating represents safety rating for a candidate.
type GeminiSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
	Blocked     bool   `json:"blocked,omitempty"`
}

// UsageMetadata contains token usage information.
type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// GeminiPromptFeedback contains feedback about the prompt.
type GeminiPromptFeedback struct {
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings"`
	BlockReason   string               `json:"blockReason,omitempty"`
}

// GeminiAPIError represents an error response from Gemini API.
type GeminiAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// Error implements the error interface for GeminiAPIError.
func (e *GeminiAPIError) Error() string {
	return fmt.Sprintf("gemini error (%d): %s", e.Code, e.Message)
}

// ============================================================================
// Request Transformer
// ============================================================================

// RequestTransformer transforms internal requests to Gemini format.
type RequestTransformer struct{}

// NewRequestTransformer creates a new RequestTransformer.
func NewRequestTransformer() *RequestTransformer {
	return &RequestTransformer{}
}

// Transform converts internal ChatCompletionRequest to Gemini format.
// Key transformations:
// 1. Maps OpenAI messages to Gemini contents with parts
// 2. Converts roles: system -> user (prepended), assistant -> model
// 3. Moves generation params to generationConfig object
// 4. Adds required safety settings
func (t *RequestTransformer) Transform(req models.ChatCompletionRequest) GeminiRequest {
	geminiReq := GeminiRequest{
		Contents:       make([]GeminiContent, 0),
		SafetySettings: defaultSafetySettings(),
	}

	// Transform messages to contents
	contents := transformMessages(req.Messages)
	geminiReq.Contents = contents

	// Build generation config from request params
	if req.Temperature != 0 || req.MaxTokens != 0 || req.TopP != 0 || len(req.Stop) > 0 {
		geminiReq.GenerationConfig = &GenerationConfig{
			Temperature:     req.Temperature,
			TopP:            req.TopP,
			MaxOutputTokens: req.MaxTokens,
			StopSequences:   req.Stop,
		}
	}

	return geminiReq
}

// transformMessages converts OpenAI-style messages to Gemini contents.
// System messages are accumulated and prepended to the first user message.
// Consecutive messages from the same role are merged.
func transformMessages(messages []models.Message) []GeminiContent {
	var contents []GeminiContent
	var systemContent strings.Builder
	var lastRole string
	var currentParts []GeminiPart

	for i, msg := range messages {
		if msg.Role == "system" {
			// Accumulate system messages
			if systemContent.Len() > 0 {
				systemContent.WriteString("\n\n")
			}
			systemContent.WriteString(msg.Content)
			continue
		}

		// Map OpenAI role to Gemini role
		geminiRole := mapRole(msg.Role)

		// Prepend system content to first user message
		content := msg.Content
		if systemContent.Len() > 0 && geminiRole == "user" {
			content = systemContent.String() + "\n\n" + content
			systemContent.Reset()
		}

		// Merge consecutive same-role messages
		if lastRole == geminiRole && len(contents) > 0 {
			// Append to existing part's text with separator
			lastPartIdx := len(contents[len(contents)-1].Parts) - 1
			if lastPartIdx >= 0 {
				contents[len(contents)-1].Parts[lastPartIdx].Text += "\n\n" + content
			}
		} else {
			// Start new content block
			currentParts = []GeminiPart{{Text: content}}
			contents = append(contents, GeminiContent{
				Role:  geminiRole,
				Parts: currentParts,
			})
			lastRole = geminiRole
		}

		// If there are remaining system messages and this is the last message,
		// prepend them to this message
		if i == len(messages)-1 && systemContent.Len() > 0 {
			// This shouldn't normally happen, but handle gracefully
			// by prepending to the last message
			if len(contents) > 0 {
				lastPartIdx := len(contents[len(contents)-1].Parts) - 1
				if lastPartIdx >= 0 {
					contents[len(contents)-1].Parts[lastPartIdx].Text =
						systemContent.String() + "\n\n" + contents[len(contents)-1].Parts[lastPartIdx].Text
				}
			}
		}
	}

	// If we only had system messages, convert to a user message
	if len(contents) == 0 && systemContent.Len() > 0 {
		contents = append(contents, GeminiContent{
			Role:  "user",
			Parts: []GeminiPart{{Text: systemContent.String()}},
		})
	}

	return contents
}

// mapRole maps OpenAI roles to Gemini roles.
// OpenAI: system, user, assistant, tool
// Gemini: user, model
func mapRole(openaiRole string) string {
	switch openaiRole {
	case "system":
		// System messages are handled specially - prepended to first user message
		return "user"
	case "assistant":
		return "model"
	case "user":
		return "user"
	case "tool":
		// Tool messages are treated as user messages in Gemini
		return "user"
	default:
		return "user"
	}
}

// defaultSafetySettings returns the default safety settings for Gemini.
func defaultSafetySettings() []SafetySetting {
	return []SafetySetting{
		{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_ONLY_HIGH"},
		{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_ONLY_HIGH"},
		{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_ONLY_HIGH"},
		{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_ONLY_HIGH"},
	}
}

// ============================================================================
// Response Transformer
// ============================================================================

// ResponseTransformer transforms Gemini responses to internal format.
type ResponseTransformer struct{}

// NewResponseTransformer creates a new ResponseTransformer.
func NewResponseTransformer() *ResponseTransformer {
	return &ResponseTransformer{}
}

// Transform converts Gemini response to internal ChatCompletionResponse.
func (t *ResponseTransformer) Transform(resp GeminiResponse, model string) (*models.ChatCompletionResponse, error) {
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in Gemini response")
	}

	// Use the first candidate
	candidate := resp.Candidates[0]

	// Aggregate content from parts
	var content strings.Builder
	for _, part := range candidate.Content.Parts {
		content.WriteString(part.Text)
	}

	// Map finish reason
	finishReason := mapFinishReason(candidate.FinishReason)

	return &models.ChatCompletionResponse{
		ID:      generateID(),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
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
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		},
	}, nil
}

// mapFinishReason maps Gemini finish reasons to OpenAI-compatible finish reasons.
func mapFinishReason(geminiReason string) string {
	switch geminiReason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	case "RECITATION":
		return "content_filter"
	case "OTHER":
		return "stop"
	case "":
		return "stop"
	default:
		return "stop"
	}
}

// generateID generates a unique ID for the response.
func generateID() string {
	return fmt.Sprintf("gemini-%d", time.Now().UnixNano())
}

// ============================================================================
// Stream Transformer
// ============================================================================

// StreamTransformer handles streaming response transformation.
type StreamTransformer struct {
	messageID          string
	model              string
	created            int64
	accumulatedContent string
	initialized        bool
}

// NewStreamTransformer creates a new StreamTransformer.
func NewStreamTransformer() *StreamTransformer {
	return &StreamTransformer{}
}

// Init initializes the stream transformer with the model name.
func (t *StreamTransformer) Init(model string) {
	t.messageID = generateID()
	t.model = model
	t.created = time.Now().Unix()
	t.accumulatedContent = ""
	t.initialized = true
}

// Reset resets the transformer state for a new stream.
func (t *StreamTransformer) Reset() {
	t.messageID = ""
	t.model = ""
	t.created = 0
	t.accumulatedContent = ""
	t.initialized = false
}

// TransformChunk transforms a Gemini SSE stream chunk to OpenAI-compatible format.
// The data parameter is the raw JSON string from the SSE data line.
// Returns the transformed SSE data bytes, a boolean indicating if this is the final chunk, and any error.
func (t *StreamTransformer) TransformChunk(data string) ([]byte, bool, error) {
	// Parse the Gemini stream chunk
	var geminiResp GeminiResponse
	if err := json.Unmarshal([]byte(data), &geminiResp); err != nil {
		return nil, false, fmt.Errorf("unmarshaling Gemini stream chunk: %w", err)
	}

	// Check if we have candidates
	if len(geminiResp.Candidates) == 0 {
		// No candidates - might be an error or end of stream
		return nil, false, nil
	}

	candidate := geminiResp.Candidates[0]

	// Aggregate text from parts
	var newContent strings.Builder
	for _, part := range candidate.Content.Parts {
		newContent.WriteString(part.Text)
	}
	fullContent := newContent.String()

	// Calculate delta (new content since last chunk)
	var delta string
	if len(fullContent) > len(t.accumulatedContent) {
		delta = fullContent[len(t.accumulatedContent):]
		t.accumulatedContent = fullContent
	}

	// Check if this is the final chunk
	isFinal := candidate.FinishReason != ""
	finishReason := ""
	if isFinal {
		finishReason = mapFinishReason(candidate.FinishReason)
	}

	// Build OpenAI-style stream chunk
	chunk := models.ChatCompletionResponse{
		ID:      t.messageID,
		Object:  "chat.completion.chunk",
		Created: t.created,
		Model:   t.model,
		Choices: []models.ChatChoice{
			{
				Index: 0,
				Message: models.Message{
					Content: delta,
				},
				FinishReason: finishReason,
			},
		},
	}

	// Include usage metadata if available in final chunk
	if isFinal && geminiResp.UsageMetadata.TotalTokenCount > 0 {
		chunk.Usage = models.Usage{
			PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
		}
	}

	chunkJSON, err := json.Marshal(chunk)
	if err != nil {
		return nil, false, fmt.Errorf("marshaling chunk: %w", err)
	}

	// Format as SSE data line
	return formatSSEData(chunkJSON), isFinal, nil
}

// GetAccumulatedContent returns the accumulated content from the stream.
func (t *StreamTransformer) GetAccumulatedContent() string {
	return t.accumulatedContent
}

// ============================================================================
// Utility Functions
// ============================================================================

// TransformErrorResponse transforms a Gemini error response to an error.
func TransformErrorResponse(body []byte) error {
	var geminiErr struct {
		Error GeminiAPIError `json:"error"`
	}

	if err := json.Unmarshal(body, &geminiErr); err == nil && geminiErr.Error.Message != "" {
		return &geminiErr.Error
	}
	return fmt.Errorf("gemini error: %s", string(body))
}

// IsDoneMarker checks if the chunk is a stream completion marker.
func IsDoneMarker(chunk []byte) bool {
	return strings.Contains(string(chunk), `"finishReason":`) ||
		strings.Contains(string(chunk), "[DONE]")
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
