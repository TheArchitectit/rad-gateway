package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type SessionStatus string

const (
	SessionStatusPending   SessionStatus = "pending"
	SessionStatusConnected SessionStatus = "connected"
	SessionStatusFailed    SessionStatus = "failed"
)

type Token struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
	TokenType    string    `json:"tokenType"`
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
}

type Provider interface {
	Name() string
	AuthURL(state string, redirectURI string) string
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

type Manager struct {
	mu        sync.RWMutex
	providers map[string]Provider
	sessions  map[string]*Session
	byState   map[string]string
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
	}
}

func (m *Manager) Start(providerName string, redirectURI string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	provider, ok := m.providers[providerName]
	if !ok {
		return nil, errors.New("unsupported provider")
	}

	id, err := newToken(16)
	if err != nil {
		return nil, err
	}
	state, err := newToken(24)
	if err != nil {
		return nil, err
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
	return session, nil
}

func (m *Manager) Complete(providerName string, state string, code string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id, ok := m.byState[state]
	if !ok {
		return nil, errors.New("invalid state")
	}
	session := m.sessions[id]
	if session == nil {
		return nil, errors.New("session not found")
	}
	if session.Provider != providerName {
		return nil, errors.New("provider mismatch")
	}
	if strings.TrimSpace(code) == "" {
		session.Status = SessionStatusFailed
		session.UpdatedAt = time.Now().UTC()
		return nil, errors.New("authorization code required")
	}

	access, err := newToken(24)
	if err != nil {
		return nil, err
	}
	refresh, err := newToken(24)
	if err != nil {
		return nil, err
	}

	session.Token = &Token{
		AccessToken:  "atk_" + access,
		RefreshToken: "rtk_" + refresh,
		ExpiresAt:    time.Now().UTC().Add(50 * time.Minute),
		TokenType:    "Bearer",
	}
	session.Status = SessionStatusConnected
	session.UpdatedAt = time.Now().UTC()
	return session, nil
}

func (m *Manager) Refresh(providerName string, refreshToken string) (*Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.providers[providerName]; !ok {
		return nil, errors.New("unsupported provider")
	}
	if strings.TrimSpace(refreshToken) == "" {
		return nil, errors.New("refresh token required")
	}
	access, err := newToken(24)
	if err != nil {
		return nil, err
	}

	return &Token{
		AccessToken:  "atk_" + access,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().UTC().Add(50 * time.Minute),
		TokenType:    "Bearer",
	}, nil
}

func (m *Manager) Validate(providerName string, accessToken string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, ok := m.providers[providerName]; !ok {
		return false, errors.New("unsupported provider")
	}
	if strings.TrimSpace(accessToken) == "" {
		return false, nil
	}
	return strings.HasPrefix(accessToken, "atk_"), nil
}

func (m *Manager) GetSession(sessionID string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, false
	}
	copy := *s
	return &copy, true
}

func newToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
