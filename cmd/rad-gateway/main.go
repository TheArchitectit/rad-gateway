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

	// Create separate muxes for different endpoint types
	// This allows applying different authentication mechanisms
	publicMux := http.NewServeMux()
	apiMux := http.NewServeMux()
	adminMux := http.NewServeMux()

	// Register API handlers (OpenAI-compatible endpoints)
	api.NewHandlers(gateway).Register(apiMux)

	// Register admin handlers
	admin.NewHandlers(cfg, usageSink, traceStore).Register(adminMux)

	// Initialize SSE handler for real-time events
	healthChecker := provider.NewHTTPHealthChecker(5 * time.Second)
	sseHandler := api.NewSSEHandler(healthChecker)
	sseHandler.RegisterRoutes(adminMux) // Register on admin mux for JWT auth

	// Initialize JWT authentication (public endpoints)
	jwtManager := auth.NewJWTManager(auth.DefaultConfig())
	authHandler := api.NewAuthHandler(jwtManager, userRepo)
	authHandler.RegisterRoutes(publicMux)

	// Create authenticators
	apiKeyAuth := middleware.NewAuthenticator(cfg.APIKeys)

	// Combine muxes with appropriate authentication
	// Order matters: more specific paths first
	combinedMux := http.NewServeMux()

	// Public endpoints (no auth required)
	combinedMux.Handle("/v1/auth/", http.StripPrefix("/v1/auth", publicMux))
	combinedMux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))

	// Admin endpoints require JWT authentication
	jwtMiddleware := auth.NewMiddleware(jwtManager)
	adminHandler := jwtMiddleware.Authenticate(adminMux)
	combinedMux.Handle("/v0/admin/", http.StripPrefix("/v0/admin", adminHandler))
	combinedMux.Handle("/v0/management/", http.StripPrefix("/v0/management", adminHandler))

	// API endpoints require API key authentication (except health)
	apiHandler := apiKeyAuth.Require(apiMux)
	combinedMux.Handle("/v1/", http.StripPrefix("/v1", apiHandler))

	// Apply global middleware
	handler := middleware.WithRequestContext(combinedMux)
	handler = middleware.WithSecurityHeaders(handler) // Add security headers
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

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
