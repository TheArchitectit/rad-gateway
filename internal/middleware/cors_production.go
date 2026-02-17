// Package middleware provides production-ready security middleware.
package middleware

import (
	"net/http"
	"strings"
)

// ProductionCORSConfig returns a strict CORS configuration for production.
// This configuration explicitly disallows wildcard origins and requires
// all allowed origins to be explicitly configured.
func ProductionCORSConfig(allowedOrigins []string) CORSConfig {
	// Ensure no wildcards in production
	filteredOrigins := make([]string, 0, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin != "*" && origin != "" {
			filteredOrigins = append(filteredOrigins, strings.ToLower(origin))
		}
	}

	return CORSConfig{
		AllowedOrigins:   filteredOrigins,
		AllowedMethods:   []string{
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
			"X-CSRF-Token",
		},
		ExposedHeaders: []string{
			"X-Request-Id",
			"X-Trace-Id",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		},
		AllowCredentials: true,
		MaxAge:           3600, // 1 hour - shorter for production security
	}
}

// StrictCORSConfig returns the most restrictive CORS configuration.
// Use this for admin/internal endpoints that should only be accessed
// from specific internal origins.
func StrictCORSConfig(allowedOrigins []string) CORSConfig {
	config := ProductionCORSConfig(allowedOrigins)

	// Even more restrictive for strict mode
	config.AllowedMethods = []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodOptions,
	}
	config.MaxAge = 600 // 10 minutes

	return config
}

// ValidateOrigin checks if an origin is valid for production use.
// Returns true only for HTTPS origins (except localhost for development).
func ValidateOrigin(origin string) bool {
	if origin == "" {
		return true // Allow same-origin requests
	}

	origin = strings.ToLower(origin)

	// Allow localhost for development (but not production deployments)
	if strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:") {
		return true
	}

	// Require HTTPS for production origins
	if !strings.HasPrefix(origin, "https://") {
		return false
	}

	// Block suspicious patterns
	suspicious := []string{
		"..",
		"//",
		"@",
		"\x00",
		"\n",
		"\r",
		"<",
		">",
	}

	for _, pattern := range suspicious {
		if strings.Contains(origin, pattern) {
			return false
		}
	}

	return true
}

// NewProductionCORS creates a CORS middleware with production configuration.
func NewProductionCORS(allowedOrigins []string) *CORS {
	// Validate all origins
	validOrigins := make([]string, 0, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if ValidateOrigin(origin) {
			validOrigins = append(validOrigins, origin)
		}
	}

	return NewCORS(ProductionCORSConfig(validOrigins))
}

// NewStrictCORS creates a CORS middleware with strict configuration.
func NewStrictCORS(allowedOrigins []string) *CORS {
	return NewCORS(StrictCORSConfig(allowedOrigins))
}

// ProductionCORSDefaults returns recommended production origins.
// Modify this based on your deployment requirements.
func ProductionCORSDefaults() []string {
	return []string{
		// Internal admin panel
		"https://admin.radgateway.io",
		// Main application
		"https://app.radgateway.io",
		// API consumers
		"https://api.radgateway.io",
	}
}

// InternalCORSDefaults returns origins for internal/admin endpoints.
func InternalCORSDefaults() []string {
	return []string{
		"https://admin.radgateway.io",
		"https://internal.radgateway.io",
	}
}
