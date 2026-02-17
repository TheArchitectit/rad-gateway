package main

import (
	"net/http"
	"os"
	"time"

	"radgateway/internal/admin"
	"radgateway/internal/api"
	"radgateway/internal/auth"
	"radgateway/internal/config"
	"radgateway/internal/core"
	"radgateway/internal/db"
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

	// Initialize database (optional - for auth and persistence)
	var database db.Database
	var userRepo db.UserRepository
	dbDSN := getenv("RAD_DB_DSN", "radgateway.db")
	dbDriver := getenv("RAD_DB_DRIVER", "sqlite")

	if dbDSN != "" {
		var err error
		database, err = db.New(db.Config{
			Driver: dbDriver,
			DSN:    dbDSN,
		})
		if err != nil {
			log.Warn("database connection failed, running without persistence", "error", err.Error())
		} else {
			if err := database.RunMigrations(); err != nil {
				log.Warn("database migrations failed", "error", err.Error())
			} else {
				log.Info("database connected")
				userRepo = database.Users()
			}
		}
		if database != nil {
			defer database.Close()
		}
	}

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

	// Initialize SSE handler for real-time events
	// Use HTTPHealthChecker from provider package for health updates
	healthChecker := provider.NewHTTPHealthChecker(5 * time.Second)
	sseHandler := api.NewSSEHandler(healthChecker)
	sseHandler.RegisterRoutes(apiMux)

	// Initialize JWT authentication
	jwtManager := auth.NewJWTManager(auth.DefaultConfig())
	authHandler := api.NewAuthHandler(jwtManager, userRepo)
	authHandler.RegisterRoutes(apiMux)

	auth := middleware.NewAuthenticator(cfg.APIKeys)
	protectedMux := withConditionalAuth(apiMux, auth)
	sseProtectedMux := withSSEAuth(apiMux, auth)
	handler := middleware.WithRequestContext(sseProtectedMux)
	// Add CORS support for admin UI and external clients
	handler = middleware.WithCORS(handler)

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

func withSSEAuth(next *http.ServeMux, auth *middleware.Authenticator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health endpoint is public
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// SSE endpoints support token via query parameter (EventSource limitation)
		if startsWith(r.URL.Path, "/v0/admin/events") {
			auth.RequireWithTokenAuth(next).ServeHTTP(w, r)
			return
		}

		// All other endpoints require header-based auth
		auth.Require(next).ServeHTTP(w, r)
	})
}

func startsWith(path, prefix string) bool {
	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
