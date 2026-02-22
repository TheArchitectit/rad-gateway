package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"radgateway/internal/logger"
)

type SessionStatus string

const (
	SessionStatusPending   SessionStatus = "pending"
	SessionStatusConnected SessionStatus = "connected"
	SessionStatusFailed    SessionStatus = "failed"
	SessionStatusExpired   SessionStatus = "expired"
)

type Token struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
	TokenType    string    `json:"tokenType"`
	Scope        string    `json:"scope,omitempty"`
}

func (t *Token) IsExpired() bool {
	if t.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().UTC().Add(60 * time.Second).After(t.ExpiresAt)
}

func (t *Token) TimeToExpiry() time.Duration {
	if t.ExpiresAt.IsZero() {
		return time.Hour * 24 * 365
	}
	return t.ExpiresAt.Sub(time.Now().UTC())
}

type Session struct {
	ID          string        `json:"id"`
	Provider    string        `json:"provider"`
	State       string        `json:"state"`
	AuthURL     string        `json:"authUrl"`
	Status      SessionStatus `json:"status"`
	Token       *Token        `json:"token,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
	RedirectURI string        `json:"redirectUri,omitempty"`
	Error       string        `json:"error,omitempty"`
}

func (s *Session) IsExpired() bool {
	if s.Status == SessionStatusExpired {
		return true
	}
	if s.Status == SessionStatusConnected && s.Token != nil {
		return s.Token.IsExpired()
	}
	if s.Status == SessionStatusPending {
		return time.Since(s.CreatedAt) > 10*time.Minute
	}
	return false
}

type Provider interface {
	Name() string
	AuthURL(state string, redirectURI string) string
	TokenURL() string
	ExchangeCode(code string) (*Token, error)
	RefreshToken(refreshToken string) (*Token, error)
	ValidateToken(accessToken string) (bool, map[string]interface{}, error)
}

type ProviderConfig struct {
	ClientID      string
	ClientSecret  string
	AuthEndpoint  string
	TokenEndpoint string
	Scopes        []string
}

type HTTPProvider struct {
	name   string
	config ProviderConfig
	client *http.Client
	log    *slog.Logger
}

func NewHTTPProvider(name string, config ProviderConfig) *HTTPProvider {
	return &HTTPProvider{
		name:   name,
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: logger.WithComponent("oauth_provider_" + name),
	}
}

func (p *HTTPProvider) Name() string {
	return p.name
}

func (p *HTTPProvider) AuthURL(state string, redirectURI string) string {
	if redirectURI == "" {
		redirectURI = "http://localhost:8090/v1/oauth/callback/" + p.name
	}

	v := url.Values{}
	v.Set("client_id", p.config.ClientID)
	v.Set("redirect_uri", redirectURI)
	v.Set("response_type", "code")
	v.Set("state", state)
	if len(p.config.Scopes) > 0 {
		v.Set("scope", strings.Join(p.config.Scopes, " "))
	}

	return p.config.AuthEndpoint + "?" + v.Encode()
}

func (p *HTTPProvider) TokenURL() string {
	return p.config.TokenEndpoint
}

func (p *HTTPProvider) ExchangeCode(code string) (*Token, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", p.config.ClientID)
	data.Set("client_secret", p.config.ClientSecret)

	resp, err := p.client.PostForm(p.config.TokenEndpoint, data)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange returned status %d", resp.StatusCode)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	token := &Token{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    result.TokenType,
		Scope:        result.Scope,
	}

	if result.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().UTC().Add(time.Duration(result.ExpiresIn) * time.Second)
	}

	return token, nil
}

func (p *HTTPProvider) RefreshToken(refreshToken string) (*Token, error) {
	if refreshToken == "" {
		return nil, errors.New("refresh token required")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", p.config.ClientID)
	data.Set("client_secret", p.config.ClientSecret)

	resp, err := p.client.PostForm(p.config.TokenEndpoint, data)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh returned status %d", resp.StatusCode)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode refresh response: %w", err)
	}

	token := &Token{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    result.TokenType,
	}

	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}

	if result.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().UTC().Add(time.Duration(result.ExpiresIn) * time.Second)
	}

	return token, nil
}

func (p *HTTPProvider) ValidateToken(accessToken string) (bool, map[string]interface{}, error) {
	if accessToken == "" {
		return false, nil, errors.New("access token required")
	}

	if !strings.HasPrefix(accessToken, "eyJ") && !strings.HasPrefix(accessToken, "atk_") {
		return false, nil, nil
	}

	metadata := map[string]interface{}{
		"provider": p.name,
		"active":   true,
	}

	return true, metadata, nil
}

type StaticProvider struct {
	provider string
	baseURL  string
}

func NewStaticProvider(provider string, baseURL string) *StaticProvider {
	return &StaticProvider{provider: provider, baseURL: baseURL}
}

func (p *StaticProvider) Name() string {
	return p.provider
}

func (p *StaticProvider) AuthURL(state string, redirectURI string) string {
	if redirectURI == "" {
		redirectURI = "http://localhost"
	}
	return fmt.Sprintf("%s?state=%s&redirect_uri=%s", strings.TrimRight(p.baseURL, "/"), state, redirectURI)
}

func (p *StaticProvider) TokenURL() string {
	return p.baseURL + "/token"
}

func (p *StaticProvider) ExchangeCode(code string) (*Token, error) {
	access, _ := generateToken(24)
	refresh, _ := generateToken(24)

	return &Token{
		AccessToken:  "atk_" + access,
		RefreshToken: "rtk_" + refresh,
		ExpiresAt:    time.Now().UTC().Add(50 * time.Minute),
		TokenType:    "Bearer",
	}, nil
}

func (p *StaticProvider) RefreshToken(refreshToken string) (*Token, error) {
	access, _ := generateToken(24)
	return &Token{
		AccessToken:  "atk_" + access,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().UTC().Add(50 * time.Minute),
		TokenType:    "Bearer",
	}, nil
}

func (p *StaticProvider) ValidateToken(accessToken string) (bool, map[string]interface{}, error) {
	if strings.TrimSpace(accessToken) == "" {
		return false, nil, nil
	}
	return strings.HasPrefix(accessToken, "atk_"), map[string]interface{}{"provider": p.provider}, nil
}

type Manager struct {
	mu        sync.RWMutex
	providers map[string]Provider
	sessions  map[string]*Session
	byState   map[string]string
	log       *slog.Logger
}

func NewManager() *Manager {
	providers := map[string]Provider{}
	providers["github-copilot"] = NewStaticProvider("github-copilot", "https://github.com/login/oauth/authorize")
	providers["anthropic"] = NewStaticProvider("anthropic", "https://console.anthropic.com/oauth/authorize")
	providers["gemini-cli"] = NewStaticProvider("gemini-cli", "https://accounts.google.com/o/oauth2/v2/auth")
	providers["openai-codex"] = NewStaticProvider("openai-codex", "https://auth.openai.com/authorize")
	providers["openai"] = NewStaticProvider("openai", "https://auth.openai.com/authorize")

	return &Manager{
		providers: providers,
		sessions:  map[string]*Session{},
		byState:   map[string]string{},
		log:       logger.WithComponent("oauth_manager"),
	}
}

func NewManagerWithProviders(providers map[string]Provider) *Manager {
	return &Manager{
		providers: providers,
		sessions:  map[string]*Session{},
		byState:   map[string]string{},
		log:       logger.WithComponent("oauth_manager"),
	}
}

func (m *Manager) RegisterProvider(name string, provider Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[name] = provider
}

func (m *Manager) Start(providerName string, redirectURI string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	provider, ok := m.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}

	id, err := generateToken(16)
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	state, err := generateToken(24)
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	now := time.Now().UTC()
	session := &Session{
		ID:          id,
		Provider:    providerName,
		State:       state,
		AuthURL:     provider.AuthURL(state, redirectURI),
		Status:      SessionStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
		RedirectURI: redirectURI,
	}

	m.sessions[id] = session
	m.byState[state] = id

	m.log.Info("OAuth flow started",
		"session_id", id,
		"provider", providerName,
		"redirect_uri", redirectURI,
	)

	return session, nil
}

func (m *Manager) Complete(providerName string, state string, code string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id, ok := m.byState[state]
	if !ok {
		return nil, errors.New("invalid or expired state parameter")
	}

	session := m.sessions[id]
	if session == nil {
		return nil, errors.New("session not found")
	}

	if session.Provider != providerName {
		return nil, fmt.Errorf("provider mismatch: expected %s, got %s", session.Provider, providerName)
	}

	if session.Status != SessionStatusPending {
		return nil, fmt.Errorf("session is not in pending state: %s", session.Status)
	}

	if session.IsExpired() {
		session.Status = SessionStatusExpired
		session.UpdatedAt = time.Now().UTC()
		session.Error = "session expired"
		return nil, errors.New("session has expired")
	}

	if strings.TrimSpace(code) == "" {
		session.Status = SessionStatusFailed
		session.UpdatedAt = time.Now().UTC()
		session.Error = "authorization code required"
		return nil, errors.New("authorization code required")
	}

	provider := m.providers[providerName]
	token, err := provider.ExchangeCode(code)
	if err != nil {
		session.Status = SessionStatusFailed
		session.UpdatedAt = time.Now().UTC()
		session.Error = err.Error()
		m.log.Error("token exchange failed",
			"session_id", id,
			"provider", providerName,
			"error", err,
		)
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	session.Token = token
	session.Status = SessionStatusConnected
	session.UpdatedAt = time.Now().UTC()

	delete(m.byState, state)

	m.log.Info("OAuth flow completed",
		"session_id", id,
		"provider", providerName,
		"token_expires", token.ExpiresAt,
	)

	return session, nil
}

func (m *Manager) Refresh(providerName string, refreshToken string) (*Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	provider, ok := m.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}

	if strings.TrimSpace(refreshToken) == "" {
		return nil, errors.New("refresh token required")
	}

	newToken, err := provider.RefreshToken(refreshToken)
	if err != nil {
		m.log.Error("token refresh failed",
			"provider", providerName,
			"error", err,
		)
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	for _, session := range m.sessions {
		if session.Provider == providerName && session.Token != nil &&
			session.Token.RefreshToken == refreshToken {
			session.Token = newToken
			session.UpdatedAt = time.Now().UTC()
			break
		}
	}

	m.log.Info("token refreshed",
		"provider", providerName,
		"expires_at", newToken.ExpiresAt,
	)

	return newToken, nil
}

func (m *Manager) Validate(providerName string, accessToken string) (bool, error) {
	valid, _, err := m.ValidateWithMetadata(providerName, accessToken)
	return valid, err
}

func (m *Manager) ValidateWithMetadata(providerName string, accessToken string) (bool, map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, ok := m.providers[providerName]
	if !ok {
		return false, nil, fmt.Errorf("unsupported provider: %s", providerName)
	}

	if strings.TrimSpace(accessToken) == "" {
		return false, nil, nil
	}

	return provider.ValidateToken(accessToken)
}

func (m *Manager) GetSession(sessionID string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, false
	}

	if s.IsExpired() {
		return nil, false
	}

	copy := *s
	return &copy, true
}

func (m *Manager) GetSessionByState(state string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id, ok := m.byState[state]
	if !ok {
		return nil, false
	}

	return m.GetSession(id)
}

func (m *Manager) ListSessions(providerName string) []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []*Session
	for _, s := range m.sessions {
		if providerName != "" && s.Provider != providerName {
			continue
		}
		if s.IsExpired() {
			continue
		}
		copy := *s
		sessions = append(sessions, &copy)
	}

	return sessions
}

func (m *Manager) RevokeSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return errors.New("session not found")
	}

	if session.State != "" {
		delete(m.byState, session.State)
	}

	session.Status = SessionStatusExpired
	session.Token = nil
	session.UpdatedAt = time.Now().UTC()

	m.log.Info("session revoked", "session_id", sessionID)

	return nil
}

func (m *Manager) CleanupExpiredSessions() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	var expired []string
	for id, session := range m.sessions {
		if session.IsExpired() {
			expired = append(expired, id)
			if session.State != "" {
				delete(m.byState, session.State)
			}
		}
	}

	for _, id := range expired {
		delete(m.sessions, id)
	}

	if len(expired) > 0 {
		m.log.Info("cleaned up expired sessions", "count", len(expired))
	}

	return len(expired)
}

func generateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
