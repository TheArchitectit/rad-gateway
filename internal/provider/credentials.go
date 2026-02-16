package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Credential represents a provider API key with metadata
type Credential struct {
	Value       string
	Provider    string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
	RotatedAt   *time.Time
	Version     string
	IsActive    bool
}

// IsExpired checks if the credential has expired
func (c *Credential) IsExpired() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*c.ExpiresAt)
}

// IsValid checks if the credential is valid for use
func (c *Credential) IsValid() bool {
	return c.IsActive && !c.IsExpired() && c.Value != ""
}

// CredentialStore defines the interface for credential storage backends
type CredentialStore interface {
	Get(ctx context.Context, provider string) (string, error)
	GetFull(ctx context.Context, provider string) (*Credential, error)
	Refresh(ctx context.Context, provider string) error
}

// MemoryCredentialStore provides thread-safe in-memory credential storage
type MemoryCredentialStore struct {
	mu          sync.RWMutex
	credentials map[string]*Credential
}

func NewMemoryCredentialStore() *MemoryCredentialStore {
	return &MemoryCredentialStore{
		credentials: make(map[string]*Credential),
	}
}

func (s *MemoryCredentialStore) Get(ctx context.Context, provider string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cred, ok := s.credentials[provider]
	if !ok {
		return "", fmt.Errorf("credential not found for provider: %s", provider)
	}
	if !cred.IsValid() {
		return "", fmt.Errorf("credential invalid or expired for provider: %s", provider)
	}
	return cred.Value, nil
}

func (s *MemoryCredentialStore) GetFull(ctx context.Context, provider string) (*Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cred, ok := s.credentials[provider]
	if !ok {
		return nil, fmt.Errorf("credential not found for provider: %s", provider)
	}
	return cred, nil
}

func (s *MemoryCredentialStore) Set(provider string, credential *Credential) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[provider] = credential
}

func (s *MemoryCredentialStore) Refresh(ctx context.Context, provider string) error {
	// Memory store doesn't support refresh - use Infisical for that
	return fmt.Errorf("memory store: refresh not supported, use InfisicalCredentialStore")
}

// InfisicalConfig holds configuration for Infisical integration
type InfisicalConfig struct {
	BaseURL       string
	WorkspaceID   string
	Environment   string
	Token         string
	CacheDuration time.Duration
}

// InfisicalCredentialStore fetches credentials from Infisical secret management
type InfisicalCredentialStore struct {
	config      InfisicalConfig
	client      *http.Client
	cache       *MemoryCredentialStore
	cacheExpiry map[string]time.Time
	mu          sync.RWMutex
}

// NewInfisicalCredentialStore creates a new Infisical credential store
func NewInfisicalCredentialStore(config InfisicalConfig) *InfisicalCredentialStore {
	if config.BaseURL == "" {
		config.BaseURL = "https://app.infisical.com"
	}
	if config.Environment == "" {
		config.Environment = "production"
	}
	if config.CacheDuration == 0 {
		config.CacheDuration = 5 * time.Minute
	}

	return &InfisicalCredentialStore{
		config:      config,
		client:      &http.Client{Timeout: 30 * time.Second},
		cache:       NewMemoryCredentialStore(),
		cacheExpiry: make(map[string]time.Time),
	}
}

// Get retrieves a credential, using cache if valid
func (s *InfisicalCredentialStore) Get(ctx context.Context, provider string) (string, error) {
	// Check cache first
	s.mu.RLock()
	expiry, hasExpiry := s.cacheExpiry[provider]
	s.mu.RUnlock()

	if !hasExpiry || time.Now().Before(expiry) {
		if val, err := s.cache.Get(ctx, provider); err == nil {
			return val, nil
		}
	}

	// Fetch from Infisical
	if err := s.fetchAndCache(ctx, provider); err != nil {
		return "", err
	}

	return s.cache.Get(ctx, provider)
}

// GetFull retrieves the full credential metadata
func (s *InfisicalCredentialStore) GetFull(ctx context.Context, provider string) (*Credential, error) {
	if _, err := s.Get(ctx, provider); err != nil {
		return nil, err
	}
	return s.cache.GetFull(ctx, provider)
}

// Refresh forces a refresh of the credential from Infisical
func (s *InfisicalCredentialStore) Refresh(ctx context.Context, provider string) error {
	s.mu.Lock()
	delete(s.cacheExpiry, provider)
	s.mu.Unlock()
	return s.fetchAndCache(ctx, provider)
}

// fetchAndCache retrieves credential from Infisical and updates cache
func (s *InfisicalCredentialStore) fetchAndCache(ctx context.Context, provider string) error {
	secretPath := fmt.Sprintf("/api/v3/secrets/raw/%s", s.getSecretName(provider))
	url := s.config.BaseURL + secretPath

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("infisical: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.config.Token)
	req.Header.Set("Content-Type", "application/json")

	q := req.URL.Query()
	q.Set("workspaceId", s.config.WorkspaceID)
	q.Set("environment", s.config.Environment)
	req.URL.RawQuery = q.Encode()

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("infisical: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("infisical: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Secret struct {
			SecretValue string `json:"secretValue"`
			Version     int    `json:"version"`
			CreatedAt   string `json:"createdAt"`
		} `json:"secret"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("infisical: failed to decode response: %w", err)
	}

	createdAt, _ := time.Parse(time.RFC3339, result.Secret.CreatedAt)
	credential := &Credential{
		Value:     result.Secret.SecretValue,
		Provider:  provider,
		CreatedAt: createdAt,
		Version:   fmt.Sprintf("%d", result.Secret.Version),
		IsActive:  true,
	}

	s.cache.Set(provider, credential)
	s.mu.Lock()
	s.cacheExpiry[provider] = time.Now().Add(s.config.CacheDuration)
	s.mu.Unlock()

	return nil
}

// getSecretName maps provider names to Infisical secret names
func (s *InfisicalCredentialStore) getSecretName(provider string) string {
	// Standardize secret names
	provider = strings.ToLower(provider)
	switch provider {
	case "openai":
		return "OPENAI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "gemini":
		return "GEMINI_API_KEY"
	case "azure":
		return "AZURE_OPENAI_API_KEY"
	default:
		return strings.ToUpper(provider) + "_API_KEY"
	}
}

// CredentialRotator handles credential rotation for providers
type CredentialRotator struct {
	store        CredentialStore
	factory      *AuthFactory
	rotationDays int
	mu           sync.Mutex
}

// NewCredentialRotator creates a new credential rotator
func NewCredentialRotator(store CredentialStore, factory *AuthFactory, rotationDays int) *CredentialRotator {
	if rotationDays == 0 {
		rotationDays = 90
	}
	return &CredentialRotator{
		store:        store,
		factory:      factory,
		rotationDays: rotationDays,
	}
}

// CheckRotation checks if credentials need rotation and returns providers needing rotation
func (r *CredentialRotator) CheckRotation(ctx context.Context, providers []string) ([]string, error) {
	var needsRotation []string

	for _, provider := range providers {
		cred, err := r.store.GetFull(ctx, provider)
		if err != nil {
			continue // Skip providers without credentials
		}

		age := time.Since(cred.CreatedAt)
		if age > time.Duration(r.rotationDays)*24*time.Hour {
			needsRotation = append(needsRotation, provider)
		}
	}

	return needsRotation, nil
}

// ValidateCredential validates a credential using the provider's auth strategy
func (r *CredentialRotator) ValidateCredential(ctx context.Context, provider, credential string) error {
	strategy, err := r.factory.Get(provider)
	if err != nil {
		return err
	}
	return strategy.Validate(credential)
}

// EnvironmentCredentialStore loads credentials from environment variables
type EnvironmentCredentialStore struct {
	prefix string
	cache  *MemoryCredentialStore
}

// NewEnvironmentCredentialStore creates a store that reads from environment variables
func NewEnvironmentCredentialStore(prefix string) *EnvironmentCredentialStore {
	if prefix == "" {
		prefix = "RAD_PROVIDER_"
	}
	store := &EnvironmentCredentialStore{
		prefix: prefix,
		cache:  NewMemoryCredentialStore(),
	}
	store.loadFromEnvironment()
	return store
}

func (s *EnvironmentCredentialStore) loadFromEnvironment() {
	providers := []string{"openai", "anthropic", "gemini", "azure"}

	for _, provider := range providers {
		envKey := s.prefix + strings.ToUpper(provider) + "_API_KEY"
		if value := os.Getenv(envKey); value != "" {
			cred := &Credential{
				Value:     value,
				Provider:  provider,
				CreatedAt: time.Now(),
				IsActive:  true,
			}
			s.cache.Set(provider, cred)
		}
	}
}

func (s *EnvironmentCredentialStore) Get(ctx context.Context, provider string) (string, error) {
	return s.cache.Get(ctx, provider)
}

func (s *EnvironmentCredentialStore) GetFull(ctx context.Context, provider string) (*Credential, error) {
	return s.cache.GetFull(ctx, provider)
}

func (s *EnvironmentCredentialStore) Refresh(ctx context.Context, provider string) error {
	s.loadFromEnvironment()
	return nil
}

// CompositeCredentialStore tries multiple stores in order
type CompositeCredentialStore struct {
	stores []CredentialStore
}

// NewCompositeCredentialStore creates a composite store
func NewCompositeCredentialStore(stores ...CredentialStore) *CompositeCredentialStore {
	return &CompositeCredentialStore{stores: stores}
}

func (c *CompositeCredentialStore) Get(ctx context.Context, provider string) (string, error) {
	for _, store := range c.stores {
		if val, err := store.Get(ctx, provider); err == nil {
			return val, nil
		}
	}
	return "", fmt.Errorf("credential not found in any store for provider: %s", provider)
}

func (c *CompositeCredentialStore) GetFull(ctx context.Context, provider string) (*Credential, error) {
	for _, store := range c.stores {
		if val, err := store.GetFull(ctx, provider); err == nil {
			return val, nil
		}
	}
	return nil, fmt.Errorf("credential not found in any store for provider: %s", provider)
}

func (c *CompositeCredentialStore) Refresh(ctx context.Context, provider string) error {
	var lastErr error
	for _, store := range c.stores {
		if err := store.Refresh(ctx, provider); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return lastErr
}

// ProviderAuthMiddleware creates middleware that injects provider auth into requests
func ProviderAuthMiddleware(credentials CredentialStore, factory *AuthFactory) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provider := extractProviderFromRequest(r)
			if provider == "" {
				next.ServeHTTP(w, r)
				return
			}

			strategy, err := factory.Get(provider)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":{"message":"auth configuration error","code":500}}`), http.StatusInternalServerError)
				return
			}

			credential, err := credentials.Get(r.Context(), provider)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":{"message":"failed to get credentials","code":500}}`), http.StatusInternalServerError)
				return
			}

			if err := strategy.Validate(credential); err != nil {
				http.Error(w, fmt.Sprintf(`{"error":{"message":"invalid credentials","code":401}}`), http.StatusUnauthorized)
				return
			}

			// Store credential in context for later use
			ctx := context.WithValue(r.Context(), "provider_credential", credential)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
