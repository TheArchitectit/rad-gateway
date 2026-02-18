// Package auth provides authentication middleware.
package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"radgateway/internal/logger"
	"radgateway/internal/rbac"
)

// contextKey is an empty struct type for context keys to guarantee uniqueness.
type contextKey struct{}

// Context keys for authentication context values.
// Using empty struct ensures no collisions with other packages.
var (
	// ContextKeyClaims stores the JWT claims in context.
	ContextKeyClaims = contextKey{}
	// ContextKeyUserID stores the user ID in context.
	ContextKeyUserID = contextKey{}
)

// Middleware provides JWT authentication middleware.
type Middleware struct {
	jwtManager *JWTManager
	log        *slog.Logger
}

// NewMiddleware creates a new auth middleware.
func NewMiddleware(jwtManager *JWTManager) *Middleware {
	return &Middleware{
		jwtManager: jwtManager,
		log:        logger.WithComponent("auth"),
	}
}

// Authenticate validates JWT tokens and sets user context.
// Tokens can be provided via:
// - Authorization header (Bearer token)
// - Cookie (access_token)
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := m.extractToken(r)
		if token == "" {
			m.log.Debug("no token provided", "path", r.URL.Path)
			http.Error(w, `{"error":{"message":"authentication required","code":401}}`, http.StatusUnauthorized)
			return
		}

		claims, err := m.jwtManager.ValidateAccessToken(token)
		if err != nil {
			m.log.Debug("invalid token", "path", r.URL.Path, "error", err.Error())
			http.Error(w, `{"error":{"message":"invalid or expired token","code":401}}`, http.StatusUnauthorized)
			return
		}

		// Set context values
		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKeyClaims, claims)
		ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)

		m.log.Debug("authentication successful", "user_id", claims.UserID, "path", r.URL.Path)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuthenticate authenticates if a token is present but doesn't require it.
func (m *Middleware) OptionalAuthenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := m.extractToken(r)
		if token == "" {
			// No token, proceed without auth context
			next.ServeHTTP(w, r)
			return
		}

		claims, err := m.jwtManager.ValidateAccessToken(token)
		if err != nil {
			// Invalid token, proceed without auth context (don't error)
			m.log.Debug("optional auth: invalid token", "path", r.URL.Path, "error", err.Error())
			next.ServeHTTP(w, r)
			return
		}

		// Set context values
		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKeyClaims, claims)
		ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractToken extracts the JWT token from the request.
// Checks Authorization header first, then cookie.
func (m *Middleware) extractToken(r *http.Request) string {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	// Check cookie
	if cookie, err := r.Cookie("access_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}

// GetClaims retrieves JWT claims from context.
func GetClaims(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ContextKeyClaims).(*Claims)
	return claims, ok
}

// GetUserID retrieves user ID from context.
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(ContextKeyUserID).(string)
	return userID, ok
}

// IsAuthenticated checks if the request is authenticated.
func IsAuthenticated(ctx context.Context) bool {
	_, ok := GetClaims(ctx)
	return ok
}

// RequireRole creates a middleware that requires a specific role.
func RequireRole(minRole rbac.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetClaims(r.Context())
			if !ok {
				http.Error(w, `{"error":{"message":"authentication required","code":401}}`, http.StatusUnauthorized)
				return
			}

			role, err := rbac.ParseRole(claims.Role)
			if err != nil {
				http.Error(w, `{"error":{"message":"invalid role","code":403}}`, http.StatusForbidden)
				return
			}

			if !role.IsAtLeast(minRole) && role != rbac.RoleAdmin {
				http.Error(w, `{"error":{"message":"insufficient privileges","code":403}}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin creates a middleware that requires admin role.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetClaims(r.Context())
		if !ok {
			http.Error(w, `{"error":{"message":"authentication required","code":401}}`, http.StatusUnauthorized)
			return
		}

		role, _ := rbac.ParseRole(claims.Role)
		if role != rbac.RoleAdmin {
			http.Error(w, `{"error":{"message":"admin access required","code":403}}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
