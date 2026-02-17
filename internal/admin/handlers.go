package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"log/slog"

	"radgateway/internal/config"
	"radgateway/internal/logger"
	"radgateway/internal/trace"
	"radgateway/internal/usage"
)

type Handlers struct {
	cfg   config.Config
	usage usage.Sink
	trace *trace.Store
	log   *slog.Logger
}

func NewHandlers(cfg config.Config, usageSink usage.Sink, traceStore *trace.Store) *Handlers {
	return &Handlers{
		cfg:   cfg,
		usage: usageSink,
		trace: traceStore,
		log:   logger.WithComponent("admin"),
	}
}

func (h *Handlers) Register(mux *http.ServeMux) {
	// Legacy management endpoints
	mux.HandleFunc("/v0/management/config", h.getConfig)
	mux.HandleFunc("/v0/management/usage", h.getUsage)
	mux.HandleFunc("/v0/management/traces", h.getTraces)

	// New admin API endpoints
	// Projects / Workspaces
	NewProjectHandler().RegisterRoutes(mux)

	// API Keys
	NewAPIKeyHandler().RegisterRoutes(mux)

	// Usage
	NewUsageHandler().RegisterRoutes(mux)

	// Costs
	NewCostHandler().RegisterRoutes(mux)

	// Quotas
	NewQuotaHandler().RegisterRoutes(mux)

	// Providers
	NewProviderHandler().RegisterRoutes(mux)
}

func (h *Handlers) getConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.log.Warn("admin: method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	h.log.Debug("admin: config accessed", "path", r.URL.Path)
	writeJSON(w, http.StatusOK, h.cfg.Snapshot())
}

func (h *Handlers) getUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.log.Warn("admin: method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	limit := readLimit(r, 50)
	h.log.Debug("admin: usage accessed", "path", r.URL.Path, "limit", limit)
	writeJSON(w, http.StatusOK, map[string]any{"data": h.usage.List(limit), "total": limit})
}

func (h *Handlers) getTraces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.log.Warn("admin: method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	limit := readLimit(r, 50)
	h.log.Debug("admin: traces accessed", "path", r.URL.Path, "limit", limit)
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
