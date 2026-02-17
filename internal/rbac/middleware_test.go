package rbac

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewRBACMiddleware(t *testing.T) {
	middleware := NewRBACMiddleware()

	if middleware == nil {
		t.Fatal("NewRBACMiddleware() returned nil")
	}

	// Check default skip paths
	if len(middleware.SkipAuthPaths) == 0 {
		t.Error("SkipAuthPaths should not be empty")
	}

	// Check default require project paths
	if len(middleware.RequireProjectPaths) == 0 {
		t.Error("RequireProjectPaths should not be empty")
	}
}

func TestRBACMiddlewareWithJWTValidator(t *testing.T) {
	middleware := NewRBACMiddleware()
	validator := func(token string) (*JWTClaims, error) {
		return &JWTClaims{Subject: "test"}, nil
	}

	result := middleware.WithJWTValidator(validator)
	if result.JWTValidator == nil {
		t.Error("WithJWTValidator should set the validator")
	}
}

func TestRBACMiddlewareWithProjectStore(t *testing.T) {
	middleware := NewRBACMiddleware()
	store := NewInMemoryProjectStore()

	result := middleware.WithProjectStore(store)
	if result.ProjectStore == nil {
		t.Error("WithProjectStore should set the store")
	}
}

func TestRBACMiddlewareWithSkipAuthPaths(t *testing.T) {
	middleware := NewRBACMiddleware()
	paths := []string{"/health", "/metrics"}

	result := middleware.WithSkipAuthPaths(paths)
	if len(result.SkipAuthPaths) != 2 {
		t.Errorf("len(SkipAuthPaths) = %d, want %d", len(result.SkipAuthPaths), 2)
	}
}

func TestRBACMiddlewareAuthenticate(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		authHeader     string
		validator      func(string) (*JWTClaims, error)
		expectedStatus int
		description    string
	}{
		{
			name:           "skip auth path",
			path:           "/health",
			expectedStatus: http.StatusOK,
			description:    "health endpoint should skip auth",
		},
		{
			name:           "missing token",
			path:           "/api/resource",
			expectedStatus: http.StatusUnauthorized,
			description:    "should require authentication",
		},
		{
			name:           "valid token",
			path:           "/api/resource",
			authHeader:     "Bearer valid-token",
			validator:      func(string) (*JWTClaims, error) { return &JWTClaims{Subject: "user-1", Role: "viewer"}, nil },
			expectedStatus: http.StatusOK,
			description:    "valid token should succeed",
		},
		{
			name:           "invalid token",
			path:           "/api/resource",
			authHeader:     "Bearer invalid-token",
			validator:      func(string) (*JWTClaims, error) { return nil, errors.New("invalid token") },
			expectedStatus: http.StatusUnauthorized,
			description:    "invalid token should fail",
		},
		{
			name:           "expired token",
			path:           "/api/resource",
			authHeader:     "Bearer expired-token",
			validator:      func(string) (*JWTClaims, error) { return &JWTClaims{Subject: "user-1", ExpiresAt: time.Now().Add(-1 * time.Hour).Unix()}, nil },
			expectedStatus: http.StatusUnauthorized,
			description:    "expired token should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := NewRBACMiddleware()
			if tt.validator != nil {
				middleware.WithJWTValidator(tt.validator)
			}

			handler := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("%s: status = %d, want %d", tt.description, rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestRBACMiddlewareAuthorize(t *testing.T) {
	tests := []struct {
		name           string
		role           Role
		requiredPerm   Permission
		expectedStatus int
	}{
		{
			name:           "admin has all permissions",
			role:           RoleAdmin,
			requiredPerm:   PermSystemAdmin,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "viewer lacks write permission",
			role:           RoleViewer,
			requiredPerm:   PermProjectWrite,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "viewer has read permission",
			role:           RoleViewer,
			requiredPerm:   PermProjectRead,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "developer has write permission",
			role:           RoleDeveloper,
			requiredPerm:   PermProjectWrite,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := NewRBACMiddleware()
			middleware.WithJWTValidator(func(string) (*JWTClaims, error) {
				return &JWTClaims{Subject: "user-1", Role: string(tt.role)}, nil
			})

			handler := middleware.Authenticate(
				middleware.Authorize(tt.requiredPerm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})),
			)

			req := httptest.NewRequest("GET", "/api/resource", nil)
			req.Header.Set("Authorization", "Bearer token")

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestRBACMiddlewareRequireRole(t *testing.T) {
	tests := []struct {
		name           string
		userRole       Role
		requiredRole   Role
		isAdmin        bool
		expectedStatus int
	}{
		{
			name:           "admin meets developer requirement",
			userRole:       RoleAdmin,
			requiredRole:   RoleDeveloper,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "developer meets viewer requirement",
			userRole:       RoleDeveloper,
			requiredRole:   RoleViewer,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "viewer does not meet developer requirement",
			userRole:       RoleViewer,
			requiredRole:   RoleDeveloper,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "isAdmin flag bypasses role check",
			userRole:       RoleViewer,
			requiredRole:   RoleAdmin,
			isAdmin:        true,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := NewRBACMiddleware()
			middleware.WithJWTValidator(func(string) (*JWTClaims, error) {
				return &JWTClaims{
					Subject: "user-1",
					Role:    string(tt.userRole),
					IsAdmin: tt.isAdmin,
				}, nil
			})

			handler := middleware.Authenticate(
				middleware.RequireRole(tt.requiredRole)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})),
			)

			req := httptest.NewRequest("GET", "/api/resource", nil)
			req.Header.Set("Authorization", "Bearer token")

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestRBACMiddlewareRequireAdmin(t *testing.T) {
	tests := []struct {
		name           string
		userRole       Role
		isAdmin        bool
		expectedStatus int
	}{
		{
			name:           "admin role passes",
			userRole:       RoleAdmin,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "developer with isAdmin passes",
			userRole:       RoleDeveloper,
			isAdmin:        true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "viewer fails",
			userRole:       RoleViewer,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := NewRBACMiddleware()
			middleware.WithJWTValidator(func(string) (*JWTClaims, error) {
				return &JWTClaims{
					Subject: "user-1",
					Role:    string(tt.userRole),
					IsAdmin: tt.isAdmin,
				}, nil
			})

			handler := middleware.Authenticate(
				middleware.RequireRole(RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})),
			)

			req := httptest.NewRequest("GET", "/admin/resource", nil)
			req.Header.Set("Authorization", "Bearer token")

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestRBACMiddlewareExtractJWT(t *testing.T) {
	middleware := NewRBACMiddleware()

	tests := []struct {
		name     string
		setup    func(*http.Request)
		expected string
	}{
		{
			name: "Authorization Bearer header",
			setup: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer token123")
			},
			expected: "token123",
		},
		{
			name: "X-JWT-Token header",
			setup: func(r *http.Request) {
				r.Header.Set("X-JWT-Token", "token456")
			},
			expected: "token456",
		},
		{
			name: "query parameter",
			setup: func(r *http.Request) {
				q := r.URL.Query()
				q.Set("jwt_token", "token789")
				r.URL.RawQuery = q.Encode()
			},
			expected: "token789",
		},
		{
			name:     "no token",
			setup:    func(r *http.Request) {},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			tt.setup(req)

			got := middleware.extractJWT(req)
			if got != tt.expected {
				t.Errorf("extractJWT() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGetUserContext(t *testing.T) {
	ctx := context.Background()

	// Nil context returns nil
	if GetUserContext(nil) != nil {
		t.Error("GetUserContext(nil) should return nil")
	}

	// Empty context returns nil
	if GetUserContext(ctx) != nil {
		t.Error("GetUserContext(empty) should return nil")
	}

	// With user context
	userCtx := &UserContext{
		UserID: "user-1",
		Role:   RoleDeveloper,
	}
	ctx = context.WithValue(ctx, ctxKeyUser, userCtx)

	retrieved := GetUserContext(ctx)
	if retrieved == nil {
		t.Fatal("GetUserContext should return user")
	}
	if retrieved.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", retrieved.UserID, "user-1")
	}
}

func TestGetRole(t *testing.T) {
	ctx := context.Background()

	// Empty context returns empty role
	if GetRole(ctx) != "" {
		t.Error("GetRole(empty) should return empty string")
	}

	// With role context
	ctx = context.WithValue(ctx, ctxKeyRole, RoleAdmin)
	if GetRole(ctx) != RoleAdmin {
		t.Errorf("GetRole = %q, want %q", GetRole(ctx), RoleAdmin)
	}
}

func TestGetPermissions(t *testing.T) {
	ctx := context.Background()

	// Empty context returns 0
	if GetPermissions(ctx) != 0 {
		t.Error("GetPermissions(empty) should return 0")
	}

	// With permissions context
	ctx = context.WithValue(ctx, ctxKeyPerms, PermProjectRead|PermProjectWrite)
	perms := GetPermissions(ctx)
	if perms != PermProjectRead|PermProjectWrite {
		t.Errorf("GetPermissions = %v, want %v", perms, PermProjectRead|PermProjectWrite)
	}
}

func TestIsAuthenticated(t *testing.T) {
	ctx := context.Background()

	// Empty context returns false
	if IsAuthenticated(ctx) {
		t.Error("IsAuthenticated(empty) should return false")
	}

	// With user context
	userCtx := &UserContext{UserID: "user-1"}
	ctx = context.WithValue(ctx, ctxKeyUser, userCtx)

	if !IsAuthenticated(ctx) {
		t.Error("IsAuthenticated(with user) should return true")
	}
}

func TestIsAdmin(t *testing.T) {
	ctx := context.Background()

	// Empty context returns false
	if IsAdmin(ctx) {
		t.Error("IsAdmin(empty) should return false")
	}

	// Non-admin user
	userCtx := &UserContext{UserID: "user-1", Role: RoleDeveloper, IsAdmin: false}
	ctx = context.WithValue(ctx, ctxKeyUser, userCtx)

	if IsAdmin(ctx) {
		t.Error("IsAdmin(viewer) should return false")
	}

	// Admin user
	adminCtx := &UserContext{UserID: "admin-1", Role: RoleAdmin, IsAdmin: true}
	ctx = context.WithValue(ctx, ctxKeyUser, adminCtx)

	if !IsAdmin(ctx) {
		t.Error("IsAdmin(admin) should return true")
	}
}

func TestUserContextCheckPermission(t *testing.T) {
	tests := []struct {
		name       string
		user       *UserContext
		permission Permission
		want       bool
	}{
		{
			name:       "nil user",
			user:       nil,
			permission: PermProjectRead,
			want:       false,
		},
		{
			name: "admin has all permissions",
			user: &UserContext{
				Role:    RoleAdmin,
				IsAdmin: true,
			},
			permission: PermSystemAdmin,
			want:       true,
		},
		{
			name: "viewer has read",
			user: &UserContext{
				Role:        RoleViewer,
				Permissions: RolePermissions(RoleViewer),
			},
			permission: PermProjectRead,
			want:       true,
		},
		{
			name: "viewer lacks write",
			user: &UserContext{
				Role:        RoleViewer,
				Permissions: RolePermissions(RoleViewer),
			},
			permission: PermProjectWrite,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.user.CheckPermission(tt.permission)
			if got != tt.want {
				t.Errorf("CheckPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserContextCanAccessProject(t *testing.T) {
	tests := []struct {
		name      string
		user      *UserContext
		projectID string
		want      bool
	}{
		{
			name:      "nil user",
			user:      nil,
			projectID: "proj-1",
			want:      false,
		},
		{
			name: "admin can access any project",
			user: &UserContext{
				Role:    RoleAdmin,
				IsAdmin: true,
			},
			projectID: "any-project",
			want:      true,
		},
		{
			name: "user can access assigned project",
			user: &UserContext{
				Role:            RoleDeveloper,
				ProjectID:       "proj-1",
				AllowedProjects: []string{"proj-1", "proj-2"},
			},
			projectID: "proj-1",
			want:      true,
		},
		{
			name: "user cannot access unassigned project",
			user: &UserContext{
				Role:            RoleDeveloper,
				ProjectID:       "proj-1",
				AllowedProjects: []string{"proj-1"},
			},
			projectID: "proj-2",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.user.CanAccessProject(tt.projectID)
			if got != tt.want {
				t.Errorf("CanAccessProject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserContextIsSystem(t *testing.T) {
	tests := []struct {
		name string
		user *UserContext
		want bool
	}{
		{
			name: "nil user is not system",
			user: nil,
			want: false,
		},
		{
			name: "regular user is not system",
			user: &UserContext{Role: RoleDeveloper},
			want: false,
		},
		{
			name: "system user is system",
			user: &UserContext{Role: RoleSystem},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.user.IsSystem()
			if got != tt.want {
				t.Errorf("IsSystem() = %v, want %v", got, tt.want)
			}
		})
	}
}
