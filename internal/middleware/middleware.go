package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"log/slog"

	"radgateway/internal/logger"
	"radgateway/internal/rbac"
)

type ctxKey string

const (
	KeyRequestID ctxKey = "request_id"
	KeyTraceID   ctxKey = "trace_id"
	KeyAPIKey    ctxKey = "api_key"
	KeyAPIName   ctxKey = "api_key_name"
)

type Authenticator struct {
	keys map[string]string
	log  *slog.Logger
}

func NewAuthenticator(keys map[string]string) *Authenticator {
	copyMap := map[string]string{}
	for k, v := range keys {
		copyMap[k] = v
	}
	return &Authenticator{
		keys: copyMap,
		log:  logger.WithComponent("middleware"),
	}
}

// AuditLogger interface matches audit.Logger.Log signature (avoids import cycle)
type AuditLogger interface {
	Log(ctx context.Context, eventType string, actor interface{}, resource interface{}, action, result string, details map[string]interface{}) error
}

var globalAuditLogger AuditLogger

// SetAuditLogger sets the global audit logger for auth events
func SetAuditLogger(logger AuditLogger) {
	globalAuditLogger = logger
}

func (a *Authenticator) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := extractAPIKey(r)
		if secret == "" {
			a.log.Warn("authentication failed: missing api key", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
			// Log to audit log if available
			if globalAuditLogger != nil {
				// Use nil for actor/resource since we can't import audit package
				globalAuditLogger.Log(r.Context(), "auth:failure", nil, nil,
					r.Method, "failure", map[string]interface{}{
						"reason":      "missing_api_key",
						"path":        r.URL.Path,
						"remote_addr": r.RemoteAddr,
					})
			}
			http.Error(w, `{"error":{"message":"missing api key","code":401}}`, http.StatusUnauthorized)
			return
		}

		name := ""
		for k, v := range a.keys {
			if v == secret {
				name = k
				break
			}
		}
		if name == "" {
			a.log.Warn("authentication failed: invalid api key", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
			// Log to audit log if available
			if globalAuditLogger != nil {
				globalAuditLogger.Log(r.Context(), "auth:failure", nil, nil,
					r.Method, "failure", map[string]interface{}{
						"reason":      "invalid_api_key",
						"path":        r.URL.Path,
						"remote_addr": r.RemoteAddr,
					})
			}
			http.Error(w, `{"error":{"message":"invalid api key","code":401}}`, http.StatusUnauthorized)
			return
		}

		a.log.Debug("authentication successful", "api_key_name", name, "path", r.URL.Path)

		ctx := context.WithValue(r.Context(), KeyAPIKey, secret)
		ctx = context.WithValue(ctx, KeyAPIName, name)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireWithTokenAuth is like Require but also accepts token from query parameter.
// This is needed for SSE endpoints where EventSource doesn't support custom headers.
func (a *Authenticator) RequireWithTokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try header first
		secret := extractAPIKey(r)

		// Fall back to query parameter (for SSE/EventSource)
		if secret == "" {
			secret = r.URL.Query().Get("token")
		}

		if secret == "" {
			a.log.Warn("authentication failed: missing api key", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
			http.Error(w, `{"error":{"message":"missing api key","code":401}}`, http.StatusUnauthorized)
			return
		}

		name := ""
		for k, v := range a.keys {
			if v == secret {
				name = k
				break
			}
		}
		if name == "" {
			a.log.Warn("authentication failed: invalid api key", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
			http.Error(w, `{"error":{"message":"invalid api key","code":401}}`, http.StatusUnauthorized)
			return
		}

		a.log.Debug("authentication successful", "api_key_name", name, "path", r.URL.Path)

		ctx := context.WithValue(r.Context(), KeyAPIKey, secret)
		ctx = context.WithValue(ctx, KeyAPIName, name)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func WithRequestContext(next http.Handler) http.Handler {
	log := logger.WithComponent("middleware")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = newID()
		}
		traceID := r.Header.Get("AH-Trace-Id")
		if traceID == "" {
			traceID = r.Header.Get("X-Trace-Id")
		}
		if traceID == "" {
			traceID = reqID
		}

		w.Header().Set("X-Request-Id", reqID)
		w.Header().Set("X-Trace-Id", traceID)

		log.Debug("request context initialized", "request_id", reqID, "trace_id", traceID, "path", r.URL.Path, "method", r.Method)

		ctx := context.WithValue(r.Context(), KeyRequestID, reqID)
		ctx = context.WithValue(ctx, KeyTraceID, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetRequestID(ctx context.Context) string {
	v, _ := ctx.Value(KeyRequestID).(string)
	return v
}

func GetTraceID(ctx context.Context) string {
	v, _ := ctx.Value(KeyTraceID).(string)
	return v
}

func GetAPIKeyName(ctx context.Context) string {
	v, _ := ctx.Value(KeyAPIName).(string)
	return v
}

func extractAPIKey(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	if k := strings.TrimSpace(r.Header.Get("x-api-key")); k != "" {
		return k
	}
	if k := strings.TrimSpace(r.Header.Get("x-goog-api-key")); k != "" {
		return k
	}
	return strings.TrimSpace(r.URL.Query().Get("key"))
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// RequirePermission is an RBAC middleware wrapper that checks for specific permissions.
// It can be used with chi router or standard http.Handler.
func RequirePermission(perm rbac.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := rbac.GetUserContext(r.Context())
			if user == nil {
				http.Error(w, `{"error":{"message":"authentication required","code":401}}`, http.StatusUnauthorized)
				return
			}

			if !user.CheckPermission(perm) {
				http.Error(w, `{"error":{"message":"insufficient permissions","code":403}}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole is an RBAC middleware wrapper that checks for minimum role.
func RequireRole(minRole rbac.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := rbac.GetUserContext(r.Context())
			if user == nil {
				http.Error(w, `{"error":{"message":"authentication required","code":401}}`, http.StatusUnauthorized)
				return
			}

			if !user.Role.IsAtLeast(minRole) && !user.IsAdmin {
				http.Error(w, `{"error":{"message":"insufficient role privileges","code":403}}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin ensures only admins can access the resource.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := rbac.GetUserContext(r.Context())
		if user == nil {
			http.Error(w, `{"error":{"message":"authentication required","code":401}}`, http.StatusUnauthorized)
			return
		}

		if !user.IsAdmin {
			http.Error(w, `{"error":{"message":"admin access required","code":403}}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserContext retrieves the RBAC user context from request context.
func GetUserContext(ctx context.Context) *rbac.UserContext {
	return rbac.GetUserContext(ctx)
}

// IsAdmin checks if the user in context is an admin.
func IsAdmin(ctx context.Context) bool {
	return rbac.IsAdmin(ctx)
}
