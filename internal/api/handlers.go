package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"radgateway/internal/core"
	"radgateway/internal/models"
	"radgateway/internal/streaming"
)

// StreamAdapter defines the interface for streaming-capable adapters.
// Provider adapters that support streaming should implement this interface.
type StreamAdapter interface {
	// ExecuteStream executes a streaming request and returns a ReadCloser with the SSE stream
	ExecuteStream(ctx context.Context, req models.ProviderRequest, model string) (io.ReadCloser, error)
}

type Handlers struct {
	gateway       *core.Gateway
	streamHandler *streaming.StreamHandler
}

func NewHandlers(g *core.Gateway) *Handlers {
	h := &Handlers{gateway: g}

	// Initialize stream handler with transformer factory
	h.streamHandler = streaming.NewStreamHandler(func(provider, model string) *streaming.Transformer {
		return streaming.NewTransformer(provider, model)
	})

	return h
}

func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.health)

	mux.HandleFunc("/v1/chat/completions", h.chatCompletions)
	mux.HandleFunc("/v1/responses", h.responses)
	mux.HandleFunc("/v1/messages", h.messages)
	mux.HandleFunc("/v1/embeddings", h.embeddings)
	mux.HandleFunc("/v1/images/generations", h.images)
	mux.HandleFunc("/v1/audio/transcriptions", h.transcriptions)
	mux.HandleFunc("/v1/models", h.models)
	mux.HandleFunc("/v1beta/models/", h.geminiCompat)
}

func (h *Handlers) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handlers) chatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req models.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, err)
		return
	}

	if req.Model == "" {
		req.Model = "gpt-4o-mini"
	}

	// Check if streaming is requested
	if req.Stream {
		h.handleStreamingChatCompletion(w, r, req)
		return
	}

	// Non-streaming request
	out, _, err := h.gateway.Handle(r.Context(), "chat", req.Model, req)
	if err != nil {
		upstreamError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out.Payload)
}

// handleStreamingChatCompletion handles streaming chat completion requests.
func (h *Handlers) handleStreamingChatCompletion(w http.ResponseWriter, r *http.Request, req models.ChatCompletionRequest) {
	// For now, we'll simulate streaming with a mock response
	// In production, this would connect to the provider's streaming endpoint

	provider := h.detectProvider(req.Model)
	stream, err := h.streamHandler.HandleStream(w, r, provider, req.Model)
	if err != nil {
		// Error already sent to client by HandleStream
		return
	}

	// Create a mock stream for demonstration
	// In production, this would come from the provider adapter
	mockStream := h.createMockStream(req.Model)

	// Start streaming from the mock provider
	stream.StartFromReader(mockStream)

	// Wait for completion
	if err := stream.Wait(); err != nil {
		// Log error - the stream is already handling client communication
		fmt.Printf("streaming error: %v\n", err)
	}
}

// createMockStream creates a mock SSE stream for demonstration.
// In production, this would come from the provider adapter.
func (h *Handlers) createMockStream(model string) io.ReadCloser {
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

// detectProvider determines the provider from the model name.
func (h *Handlers) detectProvider(model string) string {
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

func (h *Handlers) responses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req models.ResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, err)
		return
	}
	if req.Model == "" {
		req.Model = "gpt-4o-mini"
	}
	out, _, err := h.gateway.Handle(r.Context(), "responses", req.Model, req)
	if err != nil {
		upstreamError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out.Payload)
}

func (h *Handlers) messages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req models.ResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, err)
		return
	}
	if req.Model == "" {
		req.Model = "claude-3-5-sonnet"
	}
	out, _, err := h.gateway.Handle(r.Context(), "messages", req.Model, req)
	if err != nil {
		upstreamError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out.Payload)
}

func (h *Handlers) embeddings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req models.EmbeddingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, err)
		return
	}
	if req.Model == "" {
		req.Model = "text-embedding-3-small"
	}
	out, _, err := h.gateway.Handle(r.Context(), "embeddings", req.Model, req)
	if err != nil {
		upstreamError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out.Payload)
}

func (h *Handlers) images(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	out, _, err := h.gateway.Handle(r.Context(), "images", "gpt-image-1", map[string]any{"kind": "image_generation"})
	if err != nil {
		upstreamError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out.Payload)
}

func (h *Handlers) transcriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	out, _, err := h.gateway.Handle(r.Context(), "transcriptions", "whisper-1", map[string]any{"kind": "audio_transcription"})
	if err != nil {
		upstreamError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out.Payload)
}

func (h *Handlers) geminiCompat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/v1beta/models/")
	model := "gemini-1.5-flash"
	if idx := strings.Index(path, ":"); idx > 0 {
		model = path[:idx]
	}
	out, _, err := h.gateway.Handle(r.Context(), "gemini", model, map[string]any{"path": path})
	if err != nil {
		upstreamError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out.Payload)
}

func (h *Handlers) models(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"data": []map[string]any{
			{"id": "gpt-4o-mini", "object": "model", "owned_by": "rad"},
			{"id": "claude-3-5-sonnet", "object": "model", "owned_by": "rad"},
			{"id": "gemini-1.5-flash", "object": "model", "owned_by": "rad"},
		},
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": map[string]any{"message": "method not allowed"}})
}

func badRequest(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"message": err.Error()}})
}

func upstreamError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadGateway, map[string]any{"error": map[string]any{"message": err.Error()}})
}
