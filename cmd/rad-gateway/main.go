package main

import (
	"net/http"
	"time"

	"radgateway/internal/admin"
	"radgateway/internal/api"
	"radgateway/internal/config"
	"radgateway/internal/core"
	"radgateway/internal/logger"
	"radgateway/internal/middleware"
	"radgateway/internal/provider"
	"radgateway/internal/routing"
	"radgateway/internal/trace"
	"radgateway/internal/usage"
)

func main() {
	// Initialize structured logger first
	logger.Init(logger.DefaultConfig())
	log := logger.WithComponent("main")

	cfg := config.Load()

	usageSink := usage.NewInMemory(2000)
	traceStore := trace.NewStore(4000)

	registry := provider.NewRegistry(provider.NewMockAdapter())
	routeTable := make(map[string][]provider.Candidate)
	for model, candidates := range cfg.ModelRoutes {
		mapped := make([]provider.Candidate, 0, len(candidates))
		for _, c := range candidates {
			mapped = append(mapped, provider.Candidate{Name: c.Provider, Model: c.Model, Weight: c.Weight})
		}
		routeTable[model] = mapped
	}

	router := routing.New(registry, routeTable, cfg.RetryBudget)
	gateway := core.New(router, usageSink, traceStore)

	apiMux := http.NewServeMux()
	api.NewHandlers(gateway).Register(apiMux)
	admin.NewHandlers(cfg, usageSink, traceStore).Register(apiMux)

	auth := middleware.NewAuthenticator(cfg.APIKeys)
	protectedMux := withConditionalAuth(apiMux, auth)
	handler := middleware.WithRequestContext(protectedMux)

	log.Info("rad-gateway starting", "addr", cfg.ListenAddr)
	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Error("server failed to start", "error", err.Error())
		return
	}
}

func withConditionalAuth(next *http.ServeMux, auth *middleware.Authenticator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only /health endpoint is public (for load balancers/probes)
		// /v0/management/ endpoints require authentication
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		auth.Require(next).ServeHTTP(w, r)
	})
}

func startsWith(path, prefix string) bool {
	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
}
