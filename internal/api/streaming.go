package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"radgateway/internal/models"
	"radgateway/internal/streaming"
)

// StreamingHandler encapsulates streaming-specific handlers.
type StreamingHandler struct {
	streamHandler *streaming.StreamHandler
	log           Logger
}

// Logger interface for streaming handler dependencies.
type Logger interface {
	Error(msg string, args ...any)
}

// NewStreamingHandler creates a new streaming handler.
func NewStreamingHandler(sh *streaming.StreamHandler, log Logger) *StreamingHandler {
	return &StreamingHandler{
		streamHandler: sh,
		log:           log,
	}
}

// HandleStreamingChatCompletion handles streaming chat completion requests.
func (sh *StreamingHandler) HandleStreamingChatCompletion(w http.ResponseWriter, r *http.Request, req models.ChatCompletionRequest) {
	// For now, we'll simulate streaming with a mock response
	// In production, this would connect to the provider's streaming endpoint

	provider := sh.DetectProvider(req.Model)
	stream, err := sh.streamHandler.HandleStream(w, r, provider, req.Model)
	if err != nil {
		// Error already sent to client by HandleStream
		return
	}

	// Create a mock stream for demonstration
	// In production, this would come from the provider adapter
	mockStream := sh.CreateMockStream(req.Model)

	// Start streaming from the mock provider
	stream.StartFromReader(mockStream)

	// Wait for completion
	if err := stream.Wait(); err != nil {
		// Log error - the stream is already handling client communication
		sh.log.Error("streaming error", "error", err.Error(), "model", req.Model)
	}
}

// CreateMockStream creates a mock SSE stream for demonstration.
// In production, this would come from the provider adapter.
func (sh *StreamingHandler) CreateMockStream(model string) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		// Simulate streaming chunks
		chunks := []string{
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"` + model + `","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"` + model + `","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"` + model + `","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"` + model + `","choices":[{"index":0,"delta":{"content":" How"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"` + model + `","choices":[{"index":0,"delta":{"content":" can"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"` + model + `","choices":[{"index":0,"delta":{"content":" I"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"` + model + `","choices":[{"index":0,"delta":{"content":" help"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"` + model + `","choices":[{"index":0,"delta":{"content":" you"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"` + model + `","choices":[{"index":0,"delta":{"content":"?"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"` + model + `","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		}

		for _, chunk := range chunks {
			// SSE format: data: {...}\n\n
			_, err := fmt.Fprintf(pw, "data: %s\n\n", chunk)
			if err != nil {
				return
			}
			// Small delay to simulate real streaming
			time.Sleep(10 * time.Millisecond)
		}

		// Send [DONE] marker
		fmt.Fprint(pw, "data: [DONE]\n\n")
	}()

	return pr
}

// DetectProvider determines the provider from the model name.
func (sh *StreamingHandler) DetectProvider(model string) string {
	modelLower := strings.ToLower(model)
	switch {
	case strings.HasPrefix(modelLower, "claude"):
		return "anthropic"
	case strings.HasPrefix(modelLower, "gemini"):
		return "gemini"
	default:
		return "openai"
	}
}
