package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"radgateway/internal/core"
	"radgateway/internal/models"
)

type Handlers struct {
	gateway *core.Gateway
}

func NewHandlers(g *core.Gateway) *Handlers {
	return &Handlers{gateway: g}
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
	out, _, err := h.gateway.Handle(r.Context(), "chat", req.Model, req)
	if err != nil {
		upstreamError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out.Payload)
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
