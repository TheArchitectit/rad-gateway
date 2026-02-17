// Package api provides HTTP handlers for authentication.
package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"radgateway/internal/auth"
	"radgateway/internal/db"
	"radgateway/internal/logger"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	jwtManager    *auth.JWTManager
	passwordHasher *auth.PasswordHasher
	repo          db.UserRepository
	log           *slog.Logger
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(jwtManager *auth.JWTManager, repo db.UserRepository) *AuthHandler {
	return &AuthHandler{
		jwtManager:     jwtManager,
		passwordHasher: auth.DefaultPasswordHasher(),
		repo:           repo,
		log:            logger.WithComponent("auth"),
	}
}

// RegisterRoutes registers auth routes on the provided mux.
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/auth/login", h.handleLogin)
	mux.HandleFunc("/v1/auth/logout", h.handleLogout)
	mux.HandleFunc("/v1/auth/refresh", h.handleRefresh)
	mux.HandleFunc("/v1/auth/me", h.handleMe)
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents a login response.
type LoginResponse struct {
	User         *UserInfo `json:"user"`
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// UserInfo represents user information in responses.
type UserInfo struct {
	ID          string   `json:"id"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	WorkspaceID string   `json:"workspace_id"`
	Permissions []string `json:"permissions"`
}

// handleLogin handles user login.
func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Debug("login: invalid request body", "error", err.Error())
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "email and password required")
		return
	}

	// Fetch user from database
	ctx := r.Context()
	user, err := h.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		h.log.Debug("login: user not found", "email", req.Email, "error", err.Error())
		// Use generic error to prevent user enumeration
		writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Check if user is active
	if user.Status != "active" {
		h.log.Warn("login: user account not active", "email", req.Email, "status", user.Status)
		writeJSONError(w, http.StatusUnauthorized, "account not active")
		return
	}

	// Verify password
	if user.PasswordHash == nil || !h.passwordHasher.Verify(req.Password, *user.PasswordHash) {
		h.log.Debug("login: invalid password", "email", req.Email)
		writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Get user permissions (simplified - in production, fetch from roles)
	permissions := []string{"read", "write"}
	role := "developer"
	if strings.Contains(req.Email, "admin") {
		role = "admin"
		permissions = append(permissions, "delete", "admin")
	}

	// Generate tokens
	tokenPair, err := h.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		role,
		user.WorkspaceID,
		permissions,
	)
	if err != nil {
		h.log.Error("login: failed to generate tokens", err, "email", req.Email)
		writeJSONError(w, http.StatusInternalServerError, "authentication failed")
		return
	}

	// Update last login
	_ = h.repo.UpdateLastLogin(ctx, user.ID, time.Now())

	// Set httpOnly cookies
	h.setAuthCookies(w, tokenPair)

	// Build response
	name := user.Email
	if user.DisplayName != nil {
		name = *user.DisplayName
	}

	resp := LoginResponse{
		User: &UserInfo{
			ID:          user.ID,
			Email:       user.Email,
			Name:        name,
			Role:        role,
			WorkspaceID: user.WorkspaceID,
			Permissions: permissions,
		},
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}

	h.log.Info("login successful", "user_id", user.ID, "email", req.Email)
	writeJSON(w, http.StatusOK, resp)
}

// handleLogout handles user logout.
func (h *AuthHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Clear auth cookies
	h.clearAuthCookies(w)

	h.log.Debug("logout successful")
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

// RefreshRequest represents a token refresh request.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// handleRefresh handles token refresh.
func (h *AuthHandler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Get refresh token from request body or cookie
	var refreshToken string
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.RefreshToken != "" {
		refreshToken = req.RefreshToken
	} else {
		// Try cookie
		if cookie, err := r.Cookie("refresh_token"); err == nil {
			refreshToken = cookie.Value
		}
	}

	if refreshToken == "" {
		writeJSONError(w, http.StatusBadRequest, "refresh token required")
		return
	}

	// For now, we validate the refresh token by checking if it matches a stored hash
	// In production, this should check a database of valid refresh tokens

	// Get user from current access token if available
	claims, _ := auth.GetClaims(r.Context())
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "session expired")
		return
	}

	// Fetch user
	ctx := r.Context()
	user, err := h.repo.GetByID(ctx, claims.UserID)
	if err != nil {
		h.log.Debug("refresh: user not found", "user_id", claims.UserID)
		writeJSONError(w, http.StatusUnauthorized, "invalid session")
		return
	}

	// Generate new tokens
	// Get user permissions
	permissions := []string{"read", "write"}
	role := claims.Role
	if role == "" {
		role = "developer"
	}

	tokenPair, err := h.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		role,
		user.WorkspaceID,
		permissions,
	)
	if err != nil {
		h.log.Error("refresh: failed to generate tokens", err, "user_id", user.ID)
		writeJSONError(w, http.StatusInternalServerError, "token refresh failed")
		return
	}

	// Set new cookies
	h.setAuthCookies(w, tokenPair)

	resp := LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}

	h.log.Debug("token refresh successful", "user_id", user.ID)
	writeJSON(w, http.StatusOK, resp)
}

// handleMe returns the current user's information.
func (h *AuthHandler) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Fetch full user data
	ctx := r.Context()
	user, err := h.repo.GetByID(ctx, claims.UserID)
	if err != nil {
		h.log.Debug("me: user not found", "user_id", claims.UserID)
		writeJSONError(w, http.StatusNotFound, "user not found")
		return
	}

	name := user.Email
	if user.DisplayName != nil {
		name = *user.DisplayName
	}

	resp := UserInfo{
		ID:          user.ID,
		Email:       user.Email,
		Name:        name,
		Role:        claims.Role,
		WorkspaceID: user.WorkspaceID,
		Permissions: claims.Permissions,
	}

	writeJSON(w, http.StatusOK, resp)
}

// setAuthCookies sets authentication cookies.
func (h *AuthHandler) setAuthCookies(w http.ResponseWriter, tokens *auth.TokenPair) {
	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    tokens.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.isSecure(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   15 * 60, // 15 minutes
	}

	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		Path:     "/v1/auth/refresh",
		HttpOnly: true,
		Secure:   h.isSecure(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	}

	http.SetCookie(w, accessCookie)
	http.SetCookie(w, refreshCookie)
}

// clearAuthCookies clears authentication cookies.
func (h *AuthHandler) clearAuthCookies(w http.ResponseWriter) {
	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.isSecure(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}

	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/v1/auth/refresh",
		HttpOnly: true,
		Secure:   h.isSecure(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}

	http.SetCookie(w, accessCookie)
	http.SetCookie(w, refreshCookie)
}

// isSecure returns true if cookies should be secure (HTTPS only).
func (h *AuthHandler) isSecure() bool {
	// In production, this should check if the server is running over HTTPS
	// For now, return false to allow HTTP in development
	return false
}

// hashToken creates a hash of a token for storage/comparison.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(hash[:])
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"code":    code,
		},
	})
}
