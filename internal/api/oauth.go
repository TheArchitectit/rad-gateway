package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"radgateway/internal/oauth"
)

type OAuthHandler struct {
	manager *oauth.Manager
}

func NewOAuthHandler(manager *oauth.Manager) *OAuthHandler {
	return &OAuthHandler{manager: manager}
}

type StartOAuthRequest struct {
	Provider    string `json:"provider"`
	RedirectURI string `json:"redirectUri,omitempty"`
}

type StartOAuthResponse struct {
	SessionID string `json:"sessionId"`
	State     string `json:"state"`
	AuthURL   string `json:"authUrl"`
	Status    string `json:"status"`
}

type RefreshOAuthRequest struct {
	Provider     string `json:"provider"`
	RefreshToken string `json:"refreshToken"`
}

type ValidateOAuthRequest struct {
	Provider    string `json:"provider"`
	AccessToken string `json:"accessToken"`
}

func (h *OAuthHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/oauth/start", h.start)
	mux.HandleFunc("/v1/oauth/callback/", h.callback)
	mux.HandleFunc("/v1/oauth/refresh", h.refresh)
	mux.HandleFunc("/v1/oauth/validate", h.validate)
}

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

	session, err := h.manager.Start(req.Provider, req.RedirectURI)
	if err != nil {
		badRequest(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, StartOAuthResponse{
		SessionID: session.ID,
		State:     session.State,
		AuthURL:   session.AuthURL,
		Status:    string(session.Status),
	})
}

func (h *OAuthHandler) callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	provider := strings.TrimPrefix(r.URL.Path, "/v1/oauth/callback/")
	provider = strings.Trim(provider, "/")
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	session, err := h.manager.Complete(provider, state, code)
	if err != nil {
		badRequest(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]any{
		"sessionId": session.ID,
		"provider":  session.Provider,
		"status":    session.Status,
		"token":     session.Token,
	})
}

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

	token, err := h.manager.Refresh(req.Provider, req.RefreshToken)
	if err != nil {
		badRequest(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, token)
}

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

	ok, err := h.manager.Validate(req.Provider, req.AccessToken)
	if err != nil {
		badRequest(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]bool{"valid": ok})
}
