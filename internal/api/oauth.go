package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"radgateway/internal/logger"
	"radgateway/internal/oauth"
)

// OAuthHandler provides HTTP endpoints for OAuth flows
type OAuthHandler struct {
	manager *oauth.Manager
	log     *slog.Logger
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(manager *oauth.Manager) *OAuthHandler {
	return &OAuthHandler{
		manager: manager,
		log:     logger.WithComponent("oauth_api"),
	}
}

// StartOAuthRequest initiates an OAuth authorization flow
type StartOAuthRequest struct {
	Provider    string `json:"provider"`
	RedirectURI string `json:"redirectUri,omitempty"`
}

// StartOAuthResponse returns the authorization URL and session info
type StartOAuthResponse struct {
	SessionID string `json:"sessionId"`
	State     string `json:"state"`
	AuthURL   string `json:"authUrl"`
	Status    string `json:"status"`
}

// CallbackOAuthResponse returns the OAuth callback result
type CallbackOAuthResponse struct {
	SessionID string       `json:"sessionId"`
	Provider  string       `json:"provider"`
	Status    string       `json:"status"`
	Token     *oauth.Token `json:"token,omitempty"`
	Error     string       `json:"error,omitempty"`
}

// RefreshOAuthRequest refreshes an OAuth access token
type RefreshOAuthRequest struct {
	Provider     string `json:"provider"`
	RefreshToken string `json:"refreshToken"`
}

// RefreshOAuthResponse returns the new token after refresh
type RefreshOAuthResponse struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
	TokenType    string    `json:"tokenType"`
	ExpiresIn    int64     `json:"expiresIn"`
}

// ValidateOAuthRequest validates an OAuth access token
type ValidateOAuthRequest struct {
	Provider    string `json:"provider"`
	AccessToken string `json:"accessToken"`
}

// ValidateOAuthResponse returns validation result
type ValidateOAuthResponse struct {
	Valid    bool                   `json:"valid"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

// SessionInfoResponse returns session information
type SessionInfoResponse struct {
	Session oauth.Session `json:"session"`
}

// Register registers OAuth routes on the provided mux
func (h *OAuthHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/oauth/start", h.start)
	mux.HandleFunc("/v1/oauth/callback/", h.callback)
	mux.HandleFunc("/v1/oauth/refresh", h.refresh)
	mux.HandleFunc("/v1/oauth/validate", h.validate)
	mux.HandleFunc("/v1/oauth/session/", h.sessionInfo)
	mux.HandleFunc("/v1/oauth/revoke/", h.revoke)
}

// start initiates an OAuth authorization flow
func (h *OAuthHandler) start(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req StartOAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, err)
		return
	}

	if req.Provider == "" {
		badRequest(w, errInvalidProvider)
		return
	}

	session, err := h.manager.Start(req.Provider, req.RedirectURI)
	if err != nil {
		h.log.Error("failed to start OAuth flow", "provider", req.Provider, "error", err)
		badRequest(w, err)
		return
	}

	h.log.Info("OAuth flow started",
		"session_id", session.ID,
		"provider", req.Provider,
	)

	writeJSONResponse(w, http.StatusOK, StartOAuthResponse{
		SessionID: session.ID,
		State:     session.State,
		AuthURL:   session.AuthURL,
		Status:    string(session.Status),
	})
}

var errInvalidProvider = errHTTP("provider is required")

type errHTTP string

func (e errHTTP) Error() string {
	return string(e)
}

// callback handles OAuth provider callback
func (h *OAuthHandler) callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	// Extract provider from path: /v1/oauth/callback/{provider}
	provider := strings.TrimPrefix(r.URL.Path, "/v1/oauth/callback/")
	provider = strings.Trim(provider, "/")
	if provider == "" {
		badRequest(w, errInvalidProvider)
		return
	}

	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		h.log.Error("OAuth callback error", "provider", provider, "error", errorParam)
		badRequest(w, errHTTP("OAuth error: "+errorParam))
		return
	}

	if state == "" {
		badRequest(w, errHTTP("state parameter required"))
		return
	}

	session, err := h.manager.Complete(provider, state, code)
	if err != nil {
		h.log.Error("OAuth completion failed", "provider", provider, "error", err)
		badRequest(w, err)
		return
	}

	h.log.Info("OAuth flow completed",
		"session_id", session.ID,
		"provider", provider,
		"status", session.Status,
	)

	resp := CallbackOAuthResponse{
		SessionID: session.ID,
		Provider:  session.Provider,
		Status:    string(session.Status),
		Token:     session.Token,
	}

	writeJSONResponse(w, http.StatusOK, resp)
}

// refresh refreshes an OAuth access token
func (h *OAuthHandler) refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req RefreshOAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, err)
		return
	}

	if req.Provider == "" {
		badRequest(w, errInvalidProvider)
		return
	}

	if req.RefreshToken == "" {
		badRequest(w, errHTTP("refresh token is required"))
		return
	}

	token, err := h.manager.Refresh(req.Provider, req.RefreshToken)
	if err != nil {
		h.log.Error("token refresh failed", "provider", req.Provider, "error", err)
		badRequest(w, err)
		return
	}

	expiresIn := int64(token.TimeToExpiry().Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}

	writeJSONResponse(w, http.StatusOK, RefreshOAuthResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.ExpiresAt,
		TokenType:    token.TokenType,
		ExpiresIn:    expiresIn,
	})
}

// validate checks if an OAuth access token is valid
func (h *OAuthHandler) validate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req ValidateOAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, err)
		return
	}

	if req.Provider == "" {
		badRequest(w, errInvalidProvider)
		return
	}

	if req.AccessToken == "" {
		writeJSONResponse(w, http.StatusOK, ValidateOAuthResponse{
			Valid: false,
			Error: "access token is required",
		})
		return
	}

	valid, metadata, err := h.manager.ValidateWithMetadata(req.Provider, req.AccessToken)
	if err != nil {
		h.log.Error("token validation failed", "provider", req.Provider, "error", err)
		writeJSONResponse(w, http.StatusOK, ValidateOAuthResponse{
			Valid: false,
			Error: err.Error(),
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, ValidateOAuthResponse{
		Valid:    valid,
		Metadata: metadata,
	})
}

// sessionInfo retrieves session information by ID
func (h *OAuthHandler) sessionInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	// Extract session ID from path: /v1/oauth/session/{id}
	sessionID := strings.TrimPrefix(r.URL.Path, "/v1/oauth/session/")
	sessionID = strings.Trim(sessionID, "/")

	if sessionID == "" {
		badRequest(w, errHTTP("session ID is required"))
		return
	}

	session, found := h.manager.GetSession(sessionID)
	if !found {
		notFound(w, "session not found or expired")
		return
	}

	writeJSONResponse(w, http.StatusOK, SessionInfoResponse{Session: *session})
}

// revoke revokes an OAuth session
func (h *OAuthHandler) revoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	// Extract session ID from path: /v1/oauth/revoke/{id}
	sessionID := strings.TrimPrefix(r.URL.Path, "/v1/oauth/revoke/")
	sessionID = strings.Trim(sessionID, "/")

	if sessionID == "" {
		badRequest(w, errHTTP("session ID is required"))
		return
	}

	if err := h.manager.RevokeSession(sessionID); err != nil {
		h.log.Error("session revocation failed", "session_id", sessionID, "error", err)
		badRequest(w, err)
		return
	}

	h.log.Info("session revoked", "session_id", sessionID)

	writeJSONResponse(w, http.StatusOK, map[string]string{
		"status":    "revoked",
		"message":   "session revoked successfully",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func notFound(w http.ResponseWriter, message string) {
	writeJSONResponse(w, http.StatusNotFound, map[string]any{
		"error": map[string]any{"message": message},
	})
}
