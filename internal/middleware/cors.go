package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"log/slog"

	"radgateway/internal/logger"
)

// CORSConfig configures CORS middleware behavior.
type CORSConfig struct {
	// AllowedOrigins is a list of origins that are allowed to access the API.
	// Use ["*"] to allow all origins (not recommended for production with credentials).
	AllowedOrigins []string

	// AllowedMethods is a list of HTTP methods allowed.
	AllowedMethods []string

	// AllowedHeaders is a list of headers the API will accept.
	AllowedHeaders []string

	// ExposedHeaders lists headers the browser can expose to the client.
	ExposedHeaders []string

	// AllowCredentials indicates if credentials (cookies, auth headers) are allowed.
	AllowCredentials bool

	// MaxAge sets the cache duration for preflight responses in seconds.
	MaxAge int
}

// DefaultCORSConfig returns a safe default CORS configuration for development.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://localhost:8080",
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
			"X-Request-Id",
			"X-Trace-Id",
			"X-API-Key",
		},
		ExposedHeaders: []string{
			"X-Request-Id",
			"X-Trace-Id",
		},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}
}

// CORS is the CORS middleware handler.
type CORS struct {
	config CORSConfig
	log    *slog.Logger
}

// NewCORS creates a new CORS middleware with the given configuration.
func NewCORS(config CORSConfig) *CORS {
	return &CORS{
		config: config,
		log:    logger.WithComponent("cors"),
	}
}

// Handler wraps an http.Handler with CORS support.
func (c *CORS) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if !c.isOriginAllowed(origin) {
			c.log.Debug("cors: origin not allowed", "origin", origin, "path", r.URL.Path)
			next.ServeHTTP(w, r)
			return
		}

		// Set CORS headers
		c.setHeaders(w, origin)

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			c.log.Debug("cors: handling preflight request", "origin", origin, "path", r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isOriginAllowed checks if the given origin is in the allowed list.
func (c *CORS) isOriginAllowed(origin string) bool {
	// Allow requests with no origin (same-origin, mobile apps, etc.)
	if origin == "" {
		return true
	}

	// Check for wildcard
	for _, allowed := range c.config.AllowedOrigins {
		if allowed == "*" {
			return true
		}
		if strings.EqualFold(allowed, origin) {
			return true
		}
	}

	return false
}

// setHeaders sets the CORS response headers.
func (c *CORS) setHeaders(w http.ResponseWriter, origin string) {
	// Access-Control-Allow-Origin
	if c.isOriginAllowed("*") && !c.config.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		// Required when using specific origins with credentials
		w.Header().Add("Vary", "Origin")
	}

	// Access-Control-Allow-Methods
	if len(c.config.AllowedMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.config.AllowedMethods, ", "))
	}

	// Access-Control-Allow-Headers
	if len(c.config.AllowedHeaders) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.config.AllowedHeaders, ", "))
	}

	// Access-Control-Expose-Headers
	if len(c.config.ExposedHeaders) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(c.config.ExposedHeaders, ", "))
	}

	// Access-Control-Allow-Credentials
	if c.config.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	// Access-Control-Max-Age
	if c.config.MaxAge > 0 {
		w.Header().Set("Access-Control-Max-Age", strconv.Itoa(c.config.MaxAge))
	}
}

// WithCORS is a convenience function that wraps a handler with default CORS.
func WithCORS(next http.Handler) http.Handler {
	return NewCORS(DefaultCORSConfig()).Handler(next)
}
