package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"time"

	"radgateway/internal/a2a"
	"radgateway/internal/admin"
	"radgateway/internal/api"
	"radgateway/internal/audit"
	"radgateway/internal/auth"
	"radgateway/internal/auth/cedar"
	"radgateway/internal/cache"
	"radgateway/internal/config"
	"radgateway/internal/core"
	"radgateway/internal/db"
	"radgateway/internal/logger"
	"radgateway/internal/mcp"
	"radgateway/internal/middleware"
	"radgateway/internal/oauth"
	"radgateway/internal/provider"
	"radgateway/internal/routing"
	"radgateway/internal/secrets"
	"radgateway/internal/trace"
	"radgateway/internal/usage"
)

//go:embed all:assets
var assets embed.FS

// a2aCacheAdapter adapts cache.TypedModelCardCache to a2a.Cache interface.
type a2aCacheAdapter struct {
	typed cache.TypedModelCardCache
}

func (a *a2aCacheAdapter) Get(ctx context.Context, id string) (*a2a.ModelCard, error) {
	return a.typed.Get(ctx, id)
}

func (a *a2aCacheAdapter) Set(ctx context.Context, id string, card *a2a.ModelCard, ttl time.Duration) error {
	return a.typed.Set(ctx, id, card, ttl)
}

func (a *a2aCacheAdapter) Delete(ctx context.Context, id string) error {
	return a.typed.Delete(ctx, id)
}

func (a *a2aCacheAdapter) GetProjectCards(ctx context.Context, projectID string) ([]a2a.ModelCard, error) {
	return a.typed.GetProjectCards(ctx, projectID)
}

func (a *a2aCacheAdapter) SetProjectCards(ctx context.Context, projectID string, cards []a2a.ModelCard, ttl time.Duration) error {
	return a.typed.SetProjectCards(ctx, projectID, cards, ttl)
}

func (a *a2aCacheAdapter) DeleteProjectCards(ctx context.Context, projectID string) error {
	return a.typed.DeleteProjectCards(ctx, projectID)
}

func (a *a2aCacheAdapter) InvalidateCard(ctx context.Context, id string, projectID string) error {
	return a.typed.InvalidateCard(ctx, id, projectID)
}

func main() {
	// Initialize structured logger first
	logger.Init(logger.DefaultConfig())
	log := logger.WithComponent("main")

	// Initialize Infisical secrets manager (optional)
	var secretLoader *secrets.Loader
	if loader, err := secrets.NewLoader(); err == nil {
		secretLoader = loader
		if loader.IsInfisicalEnabled() {
			log.Info("infisical secrets manager enabled")
		}
		defer secretLoader.Close()
	} else {
		log.Warn("failed to initialize infisical", "error", err.Error())
	}

	cfg := config.Load()

	// Initialize database (optional - for auth and persistence)
	var database db.Database
	var userRepo db.UserRepository
	var dbDriverUsed string

	// Try Infisical first, then environment, then fallback
	dbDSN := "radgateway.db"
	if secretLoader != nil {
		dbDSN = secretLoader.LoadDatabaseDSN(dbDSN)
	}
	if dbDSN == "radgateway.db" {
		dbDSN = getenv("RAD_DB_DSN", dbDSN)
	}
	dbDriver := getenv("RAD_DB_DRIVER", "sqlite")

	if dbDSN != "" {
		var err error
		// Use fallback logic: try PostgreSQL first, fall back to SQLite if unavailable
		database, dbDriverUsed, err = db.NewWithFallback(db.Config{
			Driver: dbDriver,
			DSN:    dbDSN,
		})
		if err != nil {
			log.Warn("database connection failed, running without persistence", "error", err.Error())
		} else {
			if dbDriverUsed != dbDriver {
				log.Warn("using fallback database", "requested", dbDriver, "actual", dbDriverUsed)
			}
			if err := database.RunMigrations(); err != nil {
				log.Warn("database migrations failed", "error", err.Error())
			} else {
				log.Info("database connected", "driver", dbDriverUsed, "db_var_addr", fmt.Sprintf("%p", &dbDriverUsed))
				userRepo = database.Users()
			}
		}
			if database != nil {
			defer database.Close()
		}
	}

	// Initialize audit logger (optional - for security logging)
	var auditLogger *audit.Logger
	if database != nil {
		sqlDB := database.DB()
		auditLogger = audit.NewLogger(sqlDB, audit.DefaultConfig())
		log.Info("audit logging initialized")
		// Set global audit logger for auth middleware using adapter
		middleware.SetAuditLogger(&auditLoggerAdapter{logger: auditLogger})
	} else {
		log.Warn("audit logging not available - no database connection")
	}

	// Initialize Cedar policy engine (optional - for fine-grained authorization)
	var cedarPDP *cedar.PolicyDecisionPoint
	if cedarEnabled := getenv("RAD_CEDAR_ENABLED", "false"); cedarEnabled == "true" {
		cedarPolicyPath := getenv("RAD_CEDAR_POLICY_PATH", "./policies/cedar/agent-authz.cedar")
		var err error
		cedarPDP, err = cedar.NewPDP(cedarPolicyPath)
		if err != nil {
			log.Warn("Cedar policy engine initialization failed", "error", err.Error())
		} else {
			log.Info("Cedar policy engine initialized", "policy_path", cedarPolicyPath)
		}
	}

	// Initialize Redis cache (optional - for model card caching)
	var modelCardCache cache.TypedModelCardCache
	redisAddr := getenv("RAD_REDIS_ADDR", "")
	if redisAddr != "" {
		redisConfig := cache.Config{
			Address:    redisAddr,
			Password:   getenv("RAD_REDIS_PASSWORD", ""),
			Database:   0,
			DefaultTTL: 5 * time.Minute,
			KeyPrefix:  "rad:",
		}
		if redisDB := getenv("RAD_REDIS_DB", "0"); redisDB != "0" {
			// Parse database number
			var dbNum int
			fmt.Sscanf(redisDB, "%d", &dbNum)
			redisConfig.Database = dbNum
		}

		redisCache, err := cache.NewRedis(redisConfig)
		if err != nil {
			log.Warn("redis connection failed, running without cache", "error", err.Error())
		} else {
			// Verify connection
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := redisCache.Ping(ctx); err != nil {
				log.Warn("redis ping failed, running without cache", "error", err.Error())
				redisCache.Close()
			} else {
				log.Info("redis cache connected", "address", redisAddr)
				modelCardCache = cache.NewTypedModelCardCache(redisCache, 5*time.Minute)
				defer redisCache.Close()
			}
		}
	} else {
		log.Info("redis not configured, running without cache")
	}

	// Initialize hybrid repository if database and cache are available
	var a2aRepo a2a.Repository
	var a2aTaskStore a2a.TaskStore
	if database != nil {
		// Get underlying SQL DB for hybrid repository
		type sqlDBer interface {
			DB() *sql.DB
		}
		if sqlDB, ok := database.(sqlDBer); ok {
			// Use cache if available, otherwise use nil cache (pass-through to DB)
			var cacheImpl a2a.Cache
			if modelCardCache != nil {
				cacheImpl = &a2aCacheAdapter{typed: modelCardCache}
			}
			a2aRepo = a2a.NewHybridRepository(sqlDB.DB(), cacheImpl, log)
			a2aTaskStore = a2a.NewPostgresTaskStore(sqlDB.DB())
			log.Info("A2A hybrid repository initialized")
		} else {
			log.Warn("database does not expose *sql.DB, A2A repository not initialized")
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
	mcp.NewHandlerWithGateway(gateway).Register(apiMux)

	// Register A2A handlers (if repository is initialized)
	if a2aRepo != nil {
		a2aHandlers := a2a.NewHandlersWithTaskStore(a2aRepo, a2aTaskStore, gateway)
		a2aHandlers.Register(apiMux)
		log.Info("A2A handlers registered")
	}

	// Register admin handlers
	admin.NewHandlers(cfg, usageSink, traceStore, database).Register(adminMux)

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
	oauthHandler := api.NewOAuthHandler(oauth.NewManager())
	oauthHandler.Register(combinedMux)

	// Public endpoints (no auth required)
	combinedMux.Handle("/v1/auth/", publicMux)
	combinedMux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Check database health if available
		dbStatus := "ok"
		dbHealthy := true
		if database != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := database.Ping(ctx); err != nil {
				dbStatus = "degraded"
				dbHealthy = false
			}
		} else {
			dbStatus = "not_configured"
		}

		// Return appropriate status code
		if !dbHealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		response := fmt.Sprintf(`{"status":"ok","database":"%s","driver":"%s"}`, dbStatus, dbDriverUsed)
		log.Info("health check", "dbDriverUsed", dbDriverUsed, "db_var_addr", fmt.Sprintf("%p", &dbDriverUsed))
		w.Write([]byte(response))
	}))

	a2a.NewAgentCardHandler(getenv("RAD_PUBLIC_BASE_URL", "http://localhost"), "0.1.0").Register(combinedMux)

	// Admin endpoints require JWT authentication
	jwtMiddleware := auth.NewMiddleware(jwtManager)
	adminHandler := jwtMiddleware.Authenticate(adminMux)
	combinedMux.Handle("/v0/admin/", http.StripPrefix("/v0/admin", adminHandler))
	combinedMux.Handle("/v0/management/", http.StripPrefix("/v0/management", adminHandler))

	// API endpoints require API key authentication (except health)
	apiHandler := apiKeyAuth.Require(apiMux)
	// Wrap with audit logging if available
	if auditLogger != nil {
		auditMiddleware := audit.NewMiddleware(auditLogger)
		apiHandler = auditMiddleware.AuthMiddleware(apiHandler)
		log.Info("API endpoints wrapped with audit AuthMiddleware")
	} else {
		log.Warn("auditLogger is nil, API endpoints not wrapped")
	}
	// Wrap with Cedar authorization if available
	if cedarPDP != nil {
		apiHandler = middleware.WithCedarAuthorization(cedarPDP, "invoke")(apiHandler)
		log.Info("API endpoints wrapped with Cedar authorization")
	}
	combinedMux.Handle("/v1/", apiHandler)
	combinedMux.Handle("/a2a/", apiHandler)
	combinedMux.Handle("/mcp/", apiHandler)

	// Serve embedded web UI assets
	// Use fs.Sub to strip the "assets" prefix from the embedded FS
	if webFS, err := fs.Sub(assets, "assets"); err == nil {
		fileServer := http.FileServer(http.FS(webFS))
		combinedMux.Handle("/", fileServer)
		log.Info("web UI assets served from embedded filesystem")
	} else {
		log.Warn("failed to setup web UI assets", "error", err.Error())
	}

	// Apply global middleware
	handler := middleware.WithRequestContext(combinedMux)
	handler = middleware.WithSecurityHeaders(handler) // Add security headers
	handler = middleware.WithCORS(handler)

	// Add audit logging middleware if available
	if auditLogger != nil {
		auditMiddleware := audit.NewMiddleware(auditLogger)
		handler = auditMiddleware.Handler(handler)
		log.Info("audit logging middleware enabled")
	}

	// Load TLS configuration
	mtlsConfig := middleware.LoadMTLSConfig()
	var tlsConfig *tls.Config
	if mtlsConfig.Enabled {
		var err error
		tlsConfig, err = mtlsConfig.TLSConfig()
		if err != nil {
			log.Error("failed to load TLS configuration", "error", err.Error())
			return
		}

		// Wrap handler with mTLS middleware for additional validation
		mtlsMiddleware := middleware.NewMTLSMiddleware(mtlsConfig)
		handler = mtlsMiddleware.Handler(handler)

		log.Info("TLS/mTLS enabled",
			"cert_file", mtlsConfig.CertFile,
			"key_file", mtlsConfig.KeyFile,
			"ca_file", mtlsConfig.CAFile,
			"client_auth", mtlsConfig.ClientAuth,
		)
	}

	log.Info("rad-gateway starting", "addr", cfg.ListenAddr, "tls_enabled", mtlsConfig.Enabled)
	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	var err error
	if tlsConfig != nil {
		err = server.ListenAndServeTLS("", "")
	} else {
		err = server.ListenAndServe()
	}
	if err != nil {
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

// auditLoggerAdapter adapts *audit.Logger to middleware.AuditLogger interface
type auditLoggerAdapter struct {
	logger *audit.Logger
}

func (a *auditLoggerAdapter) Log(ctx context.Context, eventType string, actor, resource interface{}, action, result string, details map[string]interface{}) error {
	// Call the underlying audit logger - actor/resource are ignored here to avoid import cycle
	// The audit logger will be called directly from middleware with the event details
	return a.logger.Store(ctx, audit.Event{
		Type:      audit.EventType(eventType),
		Action:    action,
		Result:    result,
		Details:   details,
		Timestamp: time.Now().UTC(),
	})
}
