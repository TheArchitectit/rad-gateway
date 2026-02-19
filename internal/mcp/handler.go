package mcp

import (
	"encoding/json"
	"net/http"
	"time"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

type Request struct {
	Tool     string                 `json:"tool"`
	Input    map[string]interface{} `json:"input"`
	Session  string                 `json:"session,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Response struct {
	Success   bool                   `json:"success"`
	Tool      string                 `json:"tool"`
	Output    map[string]interface{} `json:"output"`
	Timestamp time.Time              `json:"timestamp"`
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/mcp/v1/stdio", h.handleStdio)
}

func (h *Handler) handleStdio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if req.Tool == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "tool is required"})
		return
	}

	resp := Response{
		Success: true,
		Tool:    req.Tool,
		Output: map[string]interface{}{
			"status":  "accepted",
			"message": "MCP request accepted",
			"echo":    req.Input,
		},
		Timestamp: time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
