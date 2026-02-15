package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"radgateway/internal/config"
	"radgateway/internal/trace"
	"radgateway/internal/usage"
)

type Handlers struct {
	cfg   config.Config
	usage usage.Sink
	trace *trace.Store
}

func NewHandlers(cfg config.Config, usageSink usage.Sink, traceStore *trace.Store) *Handlers {
	return &Handlers{cfg: cfg, usage: usageSink, trace: traceStore}
}

func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v0/management/config", h.getConfig)
	mux.HandleFunc("/v0/management/usage", h.getUsage)
	mux.HandleFunc("/v0/management/traces", h.getTraces)
}

func (h *Handlers) getConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, h.cfg.Snapshot())
}

func (h *Handlers) getUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	limit := readLimit(r, 50)
	writeJSON(w, http.StatusOK, map[string]any{"data": h.usage.List(limit), "total": limit})
}

func (h *Handlers) getTraces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	limit := readLimit(r, 50)
	writeJSON(w, http.StatusOK, map[string]any{"data": h.trace.List(limit), "total": limit})
}

func readLimit(r *http.Request, fallback int) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	if v > 500 {
		return 500
	}
	return v
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
