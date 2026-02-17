// Package rbac provides Role-Based Access Control functionality.
package rbac

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"radgateway/internal/logger"
)

// JWTClaims represents the expected structure of JWT token claims.
type JWTClaims struct {
	Subject         string   `json:"sub"`
	Email           string   `json:"email"`
	Role            string   `json:"role"`
	WorkspaceID     string   `json:"workspace_id"`
	ProjectID       string   `json:"project_id"`
	AllowedProjects []string `json:"allowed_projects"`
	IsAdmin         bool     `json:"is_admin"`
	IssuedAt        int64    `json:"iat"`
	ExpiresAt       int64    `json:"exp"`
}

// UserContext represents the authenticated user information extracted from JWT.
type UserContext struct {
	UserID          string
	Email           string
	Role            Role
	WorkspaceID     string
	ProjectID       string
	AllowedProjects []string
	Permissions     Permission
	IsAdmin         bool
}

// IsSystem returns true if the user is a system/internal user.
func (u *UserContext) IsSystem() bool {
	if u == nil {
		return false
	}
	return u.Role == RoleSystem
}

// CheckPermission checks if the user has a specific permission.
func (u *UserContext) CheckPermission(p Permission) bool {
	if u == nil {
		return false
	}
	if u.IsAdmin || u.Role == RoleAdmin {
		return true
	}
	return HasPermission(u.Permissions, p)
}

// CanAccessProject checks if the user can access a specific project.
func (u *UserContext) CanAccessProject(projectID string) bool {
	if u == nil {
		return false
	}
	if u.IsAdmin {
		return true
	}
	for _, allowed := range u.AllowedProjects {
		if allowed == projectID {
			return true
		}
	}
	return u.ProjectID == projectID
}

// rbacCtxKey is the context key type for RBAC information.
type rbacCtxKey string

const (
	ctxKeyUser     rbacCtxKey = "rbac_user"
	ctxKeyRole     rbacCtxKey = "rbac_role"
	ctxKeyPerms    rbacCtxKey = "rbac_permissions"
	ctxKeyClaims   rbacCtxKey = "rbac_claims"
	ctxKeyAuthTime rbacCtxKey = "rbac_auth_time"
)

// RBACMiddleware provides RBAC enforcement for HTTP handlers.
type RBACMiddleware struct {
	// JWTValidator is called to validate JWT tokens
	JWTValidator func(tokenString string) (*JWTClaims, error)

	// PermissionStore provides role-permission lookups
	PermissionStore interface {
		GetRolePermissions(role Role) Permission
		GetUserPermissions(userID string) Permission
	}

	// ProjectStore provides project access validation
	ProjectStore ProjectStore

	// SkipAuthPaths are paths that bypass authentication
	SkipAuthPaths []string

	// RequireProjectPaths are paths that require project context
	RequireProjectPaths []string

	log *slog.Logger
}

// NewRBACMiddleware creates a new RBAC middleware instance.
func NewRBACMiddleware() *RBACMiddleware {
	return &RBACMiddleware{
		SkipAuthPaths: []string{
			"/health",
			"/v1/health",
			"/ready",
			"/live",
		},
		RequireProjectPaths: []string{
			"/v1/chat/completions",
			"/v1/responses",
			"/v1/messages",
			"/v1/embeddings",
			"/v1/images/generations",
			"/v1/audio/transcriptions",
		},
		log: logger.WithComponent("rbac"),
	}
}

// WithJWTValidator sets the JWT validator function.
func (rm *RBACMiddleware) WithJWTValidator(fn func(string) (*JWTClaims, error)) *RBACMiddleware {
	rm.JWTValidator = fn
	return rm
}

// WithProjectStore sets the project store.
func (rm *RBACMiddleware) WithProjectStore(store ProjectStore) *RBACMiddleware {
	rm.ProjectStore = store
	return rm
}

// WithSkipAuthPaths sets paths that bypass authentication.
func (rm *RBACMiddleware) WithSkipAuthPaths(paths []string) *RBACMiddleware {
	rm.SkipAuthPaths = paths
	return rm
}

// Authenticate validates JWT tokens and extracts user context.
func (rm *RBACMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path should skip auth
		if rm.shouldSkipAuth(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract JWT from request
		tokenString := rm.extractJWT(r)
		if tokenString == "" {
			rm.log.Warn("authentication failed: missing JWT token", "path", r.URL.Path)
			http.Error(w, `{"error":{"message":"authentication required","code":401}}`, http.StatusUnauthorized)
			return
		}

		// Validate JWT
		claims, err := rm.validateJWT(tokenString)
		if err != nil {
			rm.log.Warn("authentication failed: invalid JWT", "path", r.URL.Path, "error", err)
			http.Error(w, `{"error":{"message":"invalid authentication token","code":401}}`, http.StatusUnauthorized)
			return
		}

		// Check token expiration
		if claims.ExpiresAt > 0 && time.Now().Unix() > claims.ExpiresAt {
			rm.log.Warn("authentication failed: token expired", "path", r.URL.Path, "exp", claims.ExpiresAt)
			http.Error(w, `{"error":{"message":"token expired","code":401}}`, http.StatusUnauthorized)
			return
		}

		// Parse role
		role, err := ParseRole(claims.Role)
		if err != nil {
			// Default to viewer if role not specified
			role = RoleViewer
		}

		// Build user context
		userCtx := &UserContext{
			UserID:          claims.Subject,
			Email:           claims.Email,
			Role:            role,
			WorkspaceID:     claims.WorkspaceID,
			ProjectID:       claims.ProjectID,
			AllowedProjects: claims.AllowedProjects,
			IsAdmin:         claims.IsAdmin || role == RoleAdmin,
			Permissions:     RolePermissions(role),
		}

		// Store in context
		ctx := context.WithValue(r.Context(), ctxKeyUser, userCtx)
		ctx = context.WithValue(ctx, ctxKeyRole, role)
		ctx = context.WithValue(ctx, ctxKeyPerms, userCtx.Permissions)
		ctx = context.WithValue(ctx, ctxKeyClaims, claims)
		ctx = context.WithValue(ctx, ctxKeyAuthTime, time.Now())

		// Add project context
		projectCtx := NewProjectContext(
			claims.ProjectID,
			"",
			claims.WorkspaceID,
			claims.AllowedProjects,
			claims.IsAdmin,
		)
		ctx = WithProjectContext(ctx, projectCtx)

		rm.log.Debug("authentication successful",
			"user", claims.Subject,
			"role", role,
			"path", r.URL.Path,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Authorize checks if the authenticated user has required permissions.
func (rm *RBACMiddleware) Authorize(required Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserContext(r.Context())
			if user == nil {
				rm.log.Warn("authorization failed: no user context", "path", r.URL.Path)
				http.Error(w, `{"error":{"message":"authentication required","code":401}}`, http.StatusUnauthorized)
				return
			}

			if !user.CheckPermission(required) {
				rm.log.Warn("authorization failed: insufficient permissions",
					"user", user.UserID,
					"required", required.String(),
					"path", r.URL.Path,
				)
				http.Error(w, `{"error":{"message":"insufficient permissions","code":403}}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole ensures the user has at least the specified role.
func (rm *RBACMiddleware) RequireRole(minRole Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserContext(r.Context())
			if user == nil {
				rm.log.Warn("authorization failed: no user context", "path", r.URL.Path)
				http.Error(w, `{"error":{"message":"authentication required","code":401}}`, http.StatusUnauthorized)
				return
			}

			if !user.Role.IsAtLeast(minRole) && !user.IsAdmin {
				rm.log.Warn("authorization failed: insufficient role",
					"user", user.UserID,
					"role", user.Role,
					"required", minRole,
					"path", r.URL.Path,
				)
				http.Error(w, `{"error":{"message":"insufficient role privileges","code":403}}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireProjectAccess ensures the user has access to the requested project.
func (rm *RBACMiddleware) RequireProjectAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip if not a project-requiring path
		if !rm.requiresProject(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		user := GetUserContext(r.Context())
		if user == nil {
			rm.log.Warn("project access denied: no user context", "path", r.URL.Path)
			http.Error(w, `{"error":{"message":"authentication required","code":401}}`, http.StatusUnauthorized)
			return
		}

		// Admins can access any project
		if user.IsAdmin {
			next.ServeHTTP(w, r)
			return
		}

		// Extract project from request
		projectCtx, err := ExtractProjectFromRequest(r)
		if err != nil {
			rm.log.Warn("project access denied: invalid project context", "path", r.URL.Path, "error", err)
			http.Error(w, `{"error":{"message":"project identification required","code":400}}`, http.StatusBadRequest)
			return
		}

		// Check project access
		if projectCtx.ProjectID != "" && !user.CanAccessProject(projectCtx.ProjectID) {
			rm.log.Warn("project access denied: project not allowed",
				"user", user.UserID,
				"project", projectCtx.ProjectID,
				"path", r.URL.Path,
			)
			http.Error(w, `{"error":{"message":"access denied to project","code":403}}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireResourceAccess checks if user can access a specific resource type.
func (rm *RBACMiddleware) RequireResourceAccess(resourceType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserContext(r.Context())
			if user == nil {
				http.Error(w, `{"error":{"message":"authentication required","code":401}}`, http.StatusUnauthorized)
				return
			}

			if !user.Role.CanAccessResource(resourceType) {
				rm.log.Warn("resource access denied",
					"user", user.UserID,
					"resource", resourceType,
					"path", r.URL.Path,
				)
				http.Error(w, `{"error":{"message":"access denied to resource","code":403}}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// MiddlewareChain returns the complete RBAC middleware chain.
func (rm *RBACMiddleware) MiddlewareChain() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return rm.Authenticate(
			rm.RequireProjectAccess(next),
		)
	}
}

// extractJWT extracts the JWT token from the request.
func (rm *RBACMiddleware) extractJWT(r *http.Request) string {
	// Try Authorization header first (Bearer token)
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	// Try X-JWT-Token header
	if token := strings.TrimSpace(r.Header.Get("X-JWT-Token")); token != "" {
		return token
	}

	// Try query parameter (for WebSocket connections)
	return strings.TrimSpace(r.URL.Query().Get("jwt_token"))
}

// validateJWT validates a JWT token string.
func (rm *RBACMiddleware) validateJWT(tokenString string) (*JWTClaims, error) {
	if rm.JWTValidator != nil {
		return rm.JWTValidator(tokenString)
	}
	// Default: no validation (should be overridden in production)
	return nil, fmt.Errorf("JWT validator not configured")
}

// shouldSkipAuth checks if a path should skip authentication.
func (rm *RBACMiddleware) shouldSkipAuth(path string) bool {
	for _, skip := range rm.SkipAuthPaths {
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}
	return false
}

// requiresProject checks if a path requires project context.
func (rm *RBACMiddleware) requiresProject(path string) bool {
	for _, req := range rm.RequireProjectPaths {
		if path == req || strings.HasPrefix(path, req+"/") {
			return true
		}
	}
	return false
}

// GetUserContext retrieves the user context from a Go context.
func GetUserContext(ctx context.Context) *UserContext {
	if ctx == nil {
		return nil
	}
	if u, ok := ctx.Value(ctxKeyUser).(*UserContext); ok {
		return u
	}
	return nil
}

// GetRole retrieves the role from a Go context.
func GetRole(ctx context.Context) Role {
	if ctx == nil {
		return ""
	}
	if r, ok := ctx.Value(ctxKeyRole).(Role); ok {
		return r
	}
	return ""
}

// GetPermissions retrieves the permissions from a Go context.
func GetPermissions(ctx context.Context) Permission {
	if ctx == nil {
		return 0
	}
	if p, ok := ctx.Value(ctxKeyPerms).(Permission); ok {
		return p
	}
	return 0
}

// ContextHasPermission checks if the context has a specific permission.
func ContextHasPermission(ctx context.Context, p Permission) bool {
	perms := GetPermissions(ctx)
	return HasPermission(perms, p)
}

// IsAuthenticated checks if the context has an authenticated user.
func IsAuthenticated(ctx context.Context) bool {
	return GetUserContext(ctx) != nil
}

// IsAdmin checks if the authenticated user is an admin.
func IsAdmin(ctx context.Context) bool {
	user := GetUserContext(ctx)
	if user == nil {
		return false
	}
	return user.IsAdmin
}
