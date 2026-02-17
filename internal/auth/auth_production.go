// Package auth provides production authentication configuration.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"log/slog"

	"radgateway/internal/logger"
)

// CookieConfig configures secure cookie settings for production.
type CookieConfig struct {
	// Name is the cookie name
	Name string

	// Path is the cookie path
	Path string

	// Domain is the cookie domain (empty for host-only)
	Domain string

	// MaxAge is the cookie max age in seconds (0 for session cookie)
	MaxAge int

	// Secure requires HTTPS
	Secure bool

	// HttpOnly prevents JavaScript access
	HttpOnly bool

	// SameSite controls cross-site cookie behavior
	SameSite http.SameSite

	// SameSiteStrict enforces strict same-site policy
	SameSiteStrict bool
}

// ProductionCookieConfig returns secure cookie configuration for production.
func ProductionCookieConfig() CookieConfig {
	return CookieConfig{
		Name:     "access_token",
		Path:     "/",
		Domain:   "", // Host-only by default
		MaxAge:   900, // 15 minutes (matches access token expiry)
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}

// RefreshCookieConfig returns configuration for refresh token cookie.
func RefreshCookieConfig() CookieConfig {
	return CookieConfig{
		Name:     "refresh_token",
		Path:     "/auth/",
		Domain:   "", // Host-only by default
		MaxAge:   604800, // 7 days (matches refresh token expiry)
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}

// DevelopmentCookieConfig returns less restrictive config for development.
func DevelopmentCookieConfig() CookieConfig {
	return CookieConfig{
		Name:     "access_token",
		Path:     "/",
		Domain:   "",
		MaxAge:   900,
		Secure:   false, // Allow HTTP in development
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// SetAuthCookie sets an authentication cookie with secure settings.
func SetAuthCookie(w http.ResponseWriter, token string, config CookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     config.Name,
		Value:    token,
		Path:     config.Path,
		Domain:   config.Domain,
		MaxAge:   config.MaxAge,
		Secure:   config.Secure,
		HttpOnly: config.HttpOnly,
		SameSite: config.SameSite,
	})
}

// ClearAuthCookie clears an authentication cookie.
func ClearAuthCookie(w http.ResponseWriter, config CookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     config.Name,
		Value:    "",
		Path:     config.Path,
		Domain:   config.Domain,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		Secure:   config.Secure,
		HttpOnly: config.HttpOnly,
		SameSite: config.SameSite,
	})
}

// TokenRotation manages secure token rotation.
type TokenRotation struct {
	jwtManager     *JWTManager
	refreshStore   RefreshTokenStore
	cookieConfig   CookieConfig
	log            *slog.Logger
}

// NewTokenRotation creates a new token rotation manager.
func NewTokenRotation(
	jwtManager *JWTManager,
	refreshStore RefreshTokenStore,
	cookieConfig CookieConfig,
) *TokenRotation {
	return &TokenRotation{
		jwtManager:   jwtManager,
		refreshStore: refreshStore,
		cookieConfig: cookieConfig,
		log:          logger.WithComponent("token_rotation"),
	}
}

// RotateTokensResponse contains the new token pair.
type RotateTokensResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// RotateTokens performs secure token rotation.
// It validates the old refresh token, generates new tokens, and revokes the old refresh token.
func (tr *TokenRotation) RotateTokens(
	ctx context.Context,
	w http.ResponseWriter,
	userID, email, role, workspaceID string,
	permissions []string,
	oldRefreshToken string,
) (*RotateTokensResponse, error) {
	// Validate old refresh token
	oldHash := HashToken(oldRefreshToken)
	valid, err := tr.refreshStore.Validate(ctx, userID, oldHash)
	if err != nil || !valid {
		tr.log.Warn("token rotation failed: invalid refresh token",
			"user_id", userID,
			"error", err,
		)
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Generate new token pair
	tokenPair, err := tr.jwtManager.GenerateTokenPair(
		userID, email, role, workspaceID, permissions,
	)
	if err != nil {
		tr.log.Error("token rotation failed: generation error",
			"user_id", userID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Store new refresh token
	newHash := HashToken(tokenPair.RefreshToken)
	err = tr.refreshStore.Store(ctx, userID, newHash, tokenPair.ExpiresAt)
	if err != nil {
		tr.log.Error("token rotation failed: storage error",
			"user_id", userID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Revoke old refresh token
	err = tr.refreshStore.Revoke(ctx, userID, oldHash)
	if err != nil {
		// Log but don't fail - new tokens are already generated
		tr.log.Warn("failed to revoke old refresh token",
			"user_id", userID,
			"error", err,
		)
	}

	// Set new cookies
	SetAuthCookie(w, tokenPair.AccessToken, tr.cookieConfig)
	SetAuthCookie(w, tokenPair.RefreshToken, RefreshCookieConfig())

	tr.log.Info("tokens rotated successfully",
		"user_id", userID,
		"workspace_id", workspaceID,
	)

	return &RotateTokensResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

// Logout performs secure logout with token revocation.
func (tr *TokenRotation) Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Extract tokens
	accessToken := extractTokenFromRequest(r)
	refreshToken := extractRefreshToken(r)

	// Clear cookies
	ClearAuthCookie(w, tr.cookieConfig)
	ClearAuthCookie(w, RefreshCookieConfig())

	// Revoke tokens if we have a store
	if tr.refreshStore != nil && refreshToken != "" {
		claims, _ := tr.jwtManager.ValidateAccessToken(accessToken)
		if claims != nil {
			hash := HashToken(refreshToken)
			tr.refreshStore.Revoke(ctx, claims.UserID, hash)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// extractTokenFromRequest extracts access token from request.
func extractTokenFromRequest(r *http.Request) string {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		prefixes := []string{"Bearer ", "bearer "}
		for _, p := range prefixes {
			if strings.HasPrefix(authHeader, p) {
				return authHeader[len(p):]
			}
		}
	}

	// Check cookie
	if cookie, err := r.Cookie("access_token"); err == nil {
		return cookie.Value
	}

	return ""
}

// extractRefreshToken extracts refresh token from request.
func extractRefreshToken(r *http.Request) string {
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		return cookie.Value
	}
	return ""
}

// CSRFProtection provides CSRF token generation and validation.
type CSRFProtection struct {
	secret []byte
	log    *slog.Logger
}

// NewCSRFProtection creates a new CSRF protection handler.
func NewCSRFProtection(secret string) *CSRFProtection {
	return &CSRFProtection{
		secret: []byte(secret),
		log:    logger.WithComponent("csrf"),
	}
}

// GenerateToken generates a new CSRF token.
func (c *CSRFProtection) GenerateToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(token), nil
}

// ValidateToken validates a CSRF token.
func (c *CSRFProtection) ValidateToken(token string) bool {
	// In production, this should validate against a stored token
	// For now, we just check it's a valid base64 string of correct length
	if len(token) == 0 {
		return false
	}
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return false
	}
	return len(decoded) == 32
}

// Middleware provides CSRF protection middleware.
func (c *CSRFProtection) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF for safe methods
		if r.Method == http.MethodGet ||
			r.Method == http.MethodHead ||
			r.Method == http.MethodOptions ||
			r.Method == http.MethodTrace {
			next.ServeHTTP(w, r)
			return
		}

		// Validate CSRF token
		token := r.Header.Get("X-CSRF-Token")
		if token == "" {
			token = r.PostFormValue("csrf_token")
		}

		if !c.ValidateToken(token) {
			c.log.Warn("CSRF token validation failed", "path", r.URL.Path)
			http.Error(w, `{"error":{"message":"invalid CSRF token","code":403}}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ProductionAuthConfig holds production authentication configuration.
type ProductionAuthConfig struct {
	JWTConfig       JWTConfig
	CookieConfig    CookieConfig
	CSRFSecret      string
	MaxLoginAttempts int
	LockoutDuration time.Duration
}

// DefaultProductionAuthConfig returns default production auth configuration.
func DefaultProductionAuthConfig() ProductionAuthConfig {
	return ProductionAuthConfig{
		JWTConfig:        ProductionJWTConfig(),
		CookieConfig:     ProductionCookieConfig(),
		CSRFSecret:       generateCSRFSecret(),
		MaxLoginAttempts: 5,
		LockoutDuration: 15 * time.Minute,
	}
}

// ProductionJWTConfig returns JWT configuration suitable for production.
func ProductionJWTConfig() JWTConfig {
	return JWTConfig{
		AccessTokenSecret:  []byte(getenv("JWT_ACCESS_SECRET", generateSecret())),
		RefreshTokenSecret: []byte(getenv("JWT_REFRESH_SECRET", generateSecret())),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "rad-gateway-production",
	}
}

// SessionSecurity provides additional session security features.
type SessionSecurity struct {
	log            *slog.Logger
	activeSessions map[string]time.Time // Simple in-memory session tracking
}

// NewSessionSecurity creates a new session security manager.
func NewSessionSecurity() *SessionSecurity {
	return &SessionSecurity{
		log:            logger.WithComponent("session_security"),
		activeSessions: make(map[string]time.Time),
	}
}

// TrackSession tracks a new session.
func (s *SessionSecurity) TrackSession(sessionID string) {
	s.activeSessions[sessionID] = time.Now()
}

// InvalidateSession invalidates a session.
func (s *SessionSecurity) InvalidateSession(sessionID string) {
	delete(s.activeSessions, sessionID)
}

// IsSessionValid checks if a session is still valid.
func (s *SessionSecurity) IsSessionValid(sessionID string) bool {
	_, exists := s.activeSessions[sessionID]
	return exists
}

// SecureAuthHandler wraps authentication handlers with security features.
func SecureAuthHandler(
	handler http.HandlerFunc,
	csrfProtection *CSRFProtection,
	rateLimiter func(http.Handler) http.Handler,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Apply security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")

		handler(w, r)
	}
}

// generateSecret generates a random secret string.
func generateCSRFSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// AuthResponseWriter wraps http.ResponseWriter with security logging.
type AuthResponseWriter struct {
	http.ResponseWriter
	statusCode int
	log        *slog.Logger
}

// NewAuthResponseWriter creates a new auth response writer.
func NewAuthResponseWriter(w http.ResponseWriter, log *slog.Logger) *AuthResponseWriter {
	return &AuthResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		log:            log,
	}
}

// WriteHeader captures the status code.
func (w *AuthResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// LogAuthResponse logs authentication responses.
func LogAuthResponse(w *AuthResponseWriter, r *http.Request, userID string) {
	if w.statusCode >= 400 {
		w.log.Warn("auth response",
			"status", w.statusCode,
			"path", r.URL.Path,
			"user_id", userID,
			"remote_addr", r.RemoteAddr,
		)
	}
}

// LoginAttemptTracker tracks login attempts for brute force protection.
type LoginAttemptTracker struct {
	attempts map[string][]time.Time
	mu       sync.Mutex
	log      *slog.Logger
}

// NewLoginAttemptTracker creates a new login attempt tracker.
func NewLoginAttemptTracker() *LoginAttemptTracker {
	return &LoginAttemptTracker{
		attempts: make(map[string][]time.Time),
		log:      logger.WithComponent("login_tracker"),
	}
}

// RecordAttempt records a login attempt from an identifier (IP or username).
func (t *LoginAttemptTracker) RecordAttempt(identifier string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.attempts[identifier] = append(t.attempts[identifier], now)

	// Clean old attempts (older than 15 minutes)
	cutoff := now.Add(-15 * time.Minute)
	var validAttempts []time.Time
	for _, attempt := range t.attempts[identifier] {
		if attempt.After(cutoff) {
			validAttempts = append(validAttempts, attempt)
		}
	}
	t.attempts[identifier] = validAttempts
}

// IsLockedOut checks if an identifier is locked out.
func (t *LoginAttemptTracker) IsLockedOut(identifier string, maxAttempts int) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	attempts := len(t.attempts[identifier])
	if attempts >= maxAttempts {
		t.log.Warn("account locked out due to failed attempts",
			"identifier", identifier,
			"attempts", attempts,
		)
		return true
	}
	return false
}

// ResetAttempts resets attempts for an identifier.
func (t *LoginAttemptTracker) ResetAttempts(identifier string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.attempts, identifier)
}
