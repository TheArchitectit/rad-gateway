// Package audit provides security audit logging functionality.
package audit

import (
	"net/http"
	"strings"
	"time"

	"radgateway/internal/logger"
	"radgateway/internal/middleware"
)

// Middleware provides HTTP middleware for audit logging.
type Middleware struct {
	logger       *Logger
	enabledPaths []string
	excludePaths []string
}

// NewMiddleware creates a new audit middleware.
func NewMiddleware(logger *Logger) *Middleware {
	return &Middleware{
		logger: logger,
		excludePaths: []string{
			"/health",
			"/metrics",
			"/_next",
			"/static",
			"/assets",
		},
	}
}

// Handler wraps an http.Handler with audit logging.
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip excluded paths
		if m.isExcluded(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Create a response wrapper to capture status code
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Capture timing
		start := time.Now()

		// Process request
		next.ServeHTTP(wrapper, r)

		// Log the request
		duration := time.Since(start)
		m.logRequest(r, wrapper.statusCode, duration)
	})
}

// AuthMiddleware wraps authentication with audit logging.
func (m *Middleware) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create response wrapper
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Capture original request info
		apiKey := middleware.GetAPIKeyName(r.Context())
		remoteAddr := r.RemoteAddr
		path := r.URL.Path
		method := r.Method

		// Process request
		next.ServeHTTP(wrapper, r)

		// Log authentication events
		switch wrapper.statusCode {
		case http.StatusUnauthorized:
			m.logAuthFailure(r, apiKey, remoteAddr, path, method, "invalid_credentials")
		case http.StatusForbidden:
			m.logAuthFailure(r, apiKey, remoteAddr, path, method, "access_denied")
		case http.StatusTooManyRequests:
			m.logRateLimitExceeded(r, apiKey, remoteAddr, path)
		case http.StatusOK, http.StatusCreated, http.StatusAccepted:
			// Only log successful auth on auth endpoints
			if m.isAuthEndpoint(path) {
				m.logAuthSuccess(r, apiKey, remoteAddr, path, method)
			}
		}
	})
}

// logRequest logs a general HTTP request.
func (m *Middleware) logRequest(r *http.Request, statusCode int, duration time.Duration) {
	// Only log requests that took longer than 1 second or had errors
	if statusCode < 400 && duration < time.Second {
		return
	}

	eventType := EventType("request:completed")
	result := "success"
	if statusCode >= 400 {
		result = "failure"
		eventType = EventType("request:failed")
	}

	actor := Actor{
		Type:      "anonymous",
		IP:        m.getClientIP(r),
		UserAgent: r.UserAgent(),
	}

	// Try to get authenticated user info
	if apiKeyName := middleware.GetAPIKeyName(r.Context()); apiKeyName != "" {
		actor.Type = "api_key"
		actor.Name = apiKeyName
	}

	resource := Resource{
		Type: "endpoint",
		Name: r.URL.Path,
	}

	reqInfo := RequestInfo{
		Method:    r.Method,
		Path:      r.URL.Path,
		Query:     r.URL.RawQuery,
		RequestID: middleware.GetRequestID(r.Context()),
		TraceID:   middleware.GetTraceID(r.Context()),
	}

	details := map[string]interface{}{
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
	}

	ctx := r.Context()
	if err := m.logger.LogWithRequest(ctx, eventType, actor, resource, r.Method, result, details, reqInfo); err != nil {
		logger.WithComponent("audit").Error("failed to log request", "error", err)
	}
}

// logAuthSuccess logs a successful authentication.
func (m *Middleware) logAuthSuccess(r *http.Request, apiKeyName, remoteAddr, path, method string) {
	actor := Actor{
		Type:      "api_key",
		Name:      apiKeyName,
		IP:        m.getClientIP(r),
		UserAgent: r.UserAgent(),
	}

	resource := Resource{
		Type: "endpoint",
		Name: path,
	}

	reqInfo := RequestInfo{
		Method:    method,
		Path:      path,
		RequestID: middleware.GetRequestID(r.Context()),
		TraceID:   middleware.GetTraceID(r.Context()),
	}

	details := map[string]interface{}{
		"auth_method": "api_key",
		"endpoint":    path,
	}

	ctx := r.Context()
	if err := m.logger.LogWithRequest(ctx, EventAuthSuccess, actor, resource, method, "success", details, reqInfo); err != nil {
		logger.WithComponent("audit").Error("failed to log auth success", "error", err)
	}
}

// logAuthFailure logs an authentication failure.
func (m *Middleware) logAuthFailure(r *http.Request, apiKeyName, remoteAddr, path, method, reason string) {
	actor := Actor{
		Type:      "api_key",
		Name:      apiKeyName,
		IP:        m.getClientIP(r),
		UserAgent: r.UserAgent(),
	}

	// If no API key was provided, mark as anonymous
	if apiKeyName == "" {
		actor.Type = "anonymous"
		actor.Name = "unknown"
	}

	resource := Resource{
		Type: "endpoint",
		Name: path,
	}

	reqInfo := RequestInfo{
		Method:    method,
		Path:      path,
		RequestID: middleware.GetRequestID(r.Context()),
		TraceID:   middleware.GetTraceID(r.Context()),
	}

	details := map[string]interface{}{
		"auth_method":  "api_key",
		"endpoint":     path,
		"failure_reason": reason,
	}

	ctx := r.Context()
	if err := m.logger.LogWithRequest(ctx, EventAuthFailure, actor, resource, method, "failure", details, reqInfo); err != nil {
		logger.WithComponent("audit").Error("failed to log auth failure", "error", err)
	}
}

// logRateLimitExceeded logs a rate limit exceeded event.
func (m *Middleware) logRateLimitExceeded(r *http.Request, apiKeyName, remoteAddr, path string) {
	actor := Actor{
		Type:      "api_key",
		Name:      apiKeyName,
		IP:        m.getClientIP(r),
		UserAgent: r.UserAgent(),
	}

	resource := Resource{
		Type: "endpoint",
		Name: path,
	}

	reqInfo := RequestInfo{
		Method:    r.Method,
		Path:      path,
		RequestID: middleware.GetRequestID(r.Context()),
		TraceID:   middleware.GetTraceID(r.Context()),
	}

	details := map[string]interface{}{
		"endpoint": path,
		"limit_type": "rate",
	}

	ctx := r.Context()
	if err := m.logger.LogWithRequest(ctx, EventRateLimitExceeded, actor, resource, r.Method, "denied", details, reqInfo); err != nil {
		logger.WithComponent("audit").Error("failed to log rate limit", "error", err)
	}
}

// isExcluded checks if the path should be excluded from audit logging.
func (m *Middleware) isExcluded(r *http.Request) bool {
	path := r.URL.Path
	for _, prefix := range m.excludePaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// isAuthEndpoint checks if the path is an authentication endpoint.
func (m *Middleware) isAuthEndpoint(path string) bool {
	return strings.Contains(path, "/auth/") ||
		strings.Contains(path, "/login") ||
		strings.Contains(path, "/token")
}

// getClientIP extracts the client IP from the request.
func (m *Middleware) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Check X-Real-IP
	xri := r.Header.Get("X-Real-Ip")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (w *responseWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// Flush implements http.Flusher if the underlying ResponseWriter supports it.
func (w *responseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
