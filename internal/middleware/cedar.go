// Package middleware provides Cedar policy-based authorization middleware
package middleware

import (
	"net/http"
	"strings"

	"log/slog"

	"radgateway/internal/auth/cedar"
	"radgateway/internal/logger"
)

// CedarAuthorizer provides Cedar policy-based authorization
type CedarAuthorizer struct {
	pdp *cedar.PolicyDecisionPoint
	log *slog.Logger
}

// NewCedarAuthorizer creates a new Cedar authorizer from policy files
func NewCedarAuthorizer(policyPath string) (*CedarAuthorizer, error) {
	pdp, err := cedar.NewPDP(policyPath)
	if err != nil {
		return nil, err
	}

	return &CedarAuthorizer{
		pdp: pdp,
		log: logger.WithComponent("cedar"),
	}, nil
}

// RequirePermission returns middleware that checks Cedar policies
func (c *CedarAuthorizer) RequirePermission(action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user context from request
			apiKeyName := GetAPIKeyName(r.Context())
			if apiKeyName == "" {
				c.log.Warn("unauthorized request", "path", r.URL.Path)
				http.Error(w, `{"error":{"message":"authentication required","code":401}}`, http.StatusUnauthorized)
				return
			}

			// Build resource from request path
			resource := buildResourceFromPath(r.URL.Path)

			// Check Cedar authorization
			allowed, err := c.pdp.IsAuthorized(apiKeyName, action, resource)
			if err != nil {
				c.log.Error("cedar authorization error", "error", err, "path", r.URL.Path)
				http.Error(w, `{"error":{"message":"authorization error","code":500}}`, http.StatusInternalServerError)
				return
			}

			if !allowed {
				c.log.Warn("access denied by cedar policy",
					"principal", apiKeyName,
					"action", action,
					"resource", resource,
					"path", r.URL.Path,
				)
				http.Error(w, `{"error":{"message":"access denied","code":403}}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// buildResourceFromPath creates a resource identifier from the request path
func buildResourceFromPath(path string) string {
	// Extract resource type from path
	// /v1/models -> "models"
	// /v1/chat/completions -> "chat"
	// /v0/admin/providers -> "providers"

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 {
		// Skip version (v0, v1, etc.)
		if len(parts) > 2 {
			return strings.Join(parts[2:], "/")
		}
		return parts[1]
	}
	return path
}

// CedarConfig holds Cedar authorization configuration
type CedarConfig struct {
	PolicyPath string
	Enabled    bool
}

// DefaultCedarConfig returns default Cedar configuration
func DefaultCedarConfig() CedarConfig {
	return CedarConfig{
		PolicyPath: "./policies/cedar",
		Enabled:    false, // Disabled by default until configured
	}
}

// WithCedarAuthorization wraps a handler with Cedar policy checks
// Falls back to RBAC if Cedar is not configured
func WithCedarAuthorization(pdp *cedar.PolicyDecisionPoint, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip if no PDP configured
			if pdp == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Get authenticated user
			apiKeyName := GetAPIKeyName(r.Context())
			if apiKeyName == "" {
				http.Error(w, `{"error":{"message":"authentication required","code":401}}`, http.StatusUnauthorized)
				return
			}

			// Build resource from path
			resource := buildResourceFromPath(r.URL.Path)

			// Check authorization
			allowed, err := pdp.IsAuthorized(apiKeyName, action, resource)
			if err != nil {
				logger.WithComponent("cedar").Error("authorization error", "error", err)
				http.Error(w, `{"error":{"message":"authorization error","code":500}}`, http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, `{"error":{"message":"access denied by policy","code":403}}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
