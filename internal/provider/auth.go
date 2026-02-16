package provider

import (
	"context"
	"fmt"
	"net/http"
)

// AuthStrategy defines the interface for provider-specific authentication
type AuthStrategy interface {
	// Apply adds authentication to an HTTP request
	Apply(req *http.Request, credential string) error
	// Validate checks if a credential is valid (basic validation)
	Validate(credential string) error
	// HeaderName returns the primary header name used for this auth strategy
	HeaderName() string
}

// AuthType represents the type of authentication used by a provider
type AuthType string

const (
	AuthTypeBearer   AuthType = "bearer"
	AuthTypeAPIKey   AuthType = "api_key"
	AuthTypeGoogle   AuthType = "google"
	AuthTypeQueryParam AuthType = "query_param"
)

// ProviderAuth defines authentication configuration for a provider
type ProviderAuth struct {
	Provider   string
	AuthType   AuthType
	HeaderName string
	QueryParam string
}

// OpenAIAuth implements Bearer token authentication for OpenAI
type OpenAIAuth struct{}

func NewOpenAIAuth() *OpenAIAuth {
	return &OpenAIAuth{}
}

func (o *OpenAIAuth) Apply(req *http.Request, credential string) error {
	if err := o.Validate(credential); err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+credential)
	return nil
}

func (o *OpenAIAuth) Validate(credential string) error {
	if credential == "" {
		return fmt.Errorf("openai: credential cannot be empty")
	}
	if len(credential) < 20 {
		return fmt.Errorf("openai: credential appears invalid (too short)")
	}
	return nil
}

func (o *OpenAIAuth) HeaderName() string {
	return "Authorization"
}

// AnthropicAuth implements x-api-key authentication for Anthropic
type AnthropicAuth struct{}

func NewAnthropicAuth() *AnthropicAuth {
	return &AnthropicAuth{}
}

func (a *AnthropicAuth) Apply(req *http.Request, credential string) error {
	if err := a.Validate(credential); err != nil {
		return err
	}
	req.Header.Set("x-api-key", credential)
	req.Header.Set("anthropic-version", "2023-06-01")
	return nil
}

func (a *AnthropicAuth) Validate(credential string) error {
	if credential == "" {
		return fmt.Errorf("anthropic: credential cannot be empty")
	}
	if len(credential) < 20 {
		return fmt.Errorf("anthropic: credential appears invalid (too short)")
	}
	return nil
}

func (a *AnthropicAuth) HeaderName() string {
	return "x-api-key"
}

// GeminiAuth implements Google API key authentication
type GeminiAuth struct {
	useHeader bool
}

func NewGeminiAuth(useHeader bool) *GeminiAuth {
	return &GeminiAuth{useHeader: useHeader}
}

func (g *GeminiAuth) Apply(req *http.Request, credential string) error {
	if err := g.Validate(credential); err != nil {
		return err
	}

	if g.useHeader {
		req.Header.Set("x-goog-api-key", credential)
	} else {
		query := req.URL.Query()
		query.Set("key", credential)
		req.URL.RawQuery = query.Encode()
	}
	return nil
}

func (g *GeminiAuth) Validate(credential string) error {
	if credential == "" {
		return fmt.Errorf("gemini: credential cannot be empty")
	}
	if len(credential) < 10 {
		return fmt.Errorf("gemini: credential appears invalid (too short)")
	}
	return nil
}

func (g *GeminiAuth) HeaderName() string {
	if g.useHeader {
		return "x-goog-api-key"
	}
	return ""
}

// AzureOpenAIAuth implements Azure OpenAI authentication
type AzureOpenAIAuth struct{}

func NewAzureOpenAIAuth() *AzureOpenAIAuth {
	return &AzureOpenAIAuth{}
}

func (a *AzureOpenAIAuth) Apply(req *http.Request, credential string) error {
	if err := a.Validate(credential); err != nil {
		return err
	}
	req.Header.Set("api-key", credential)
	return nil
}

func (a *AzureOpenAIAuth) Validate(credential string) error {
	if credential == "" {
		return fmt.Errorf("azure: credential cannot be empty")
	}
	if len(credential) < 20 {
		return fmt.Errorf("azure: credential appears invalid (too short)")
	}
	return nil
}

func (a *AzureOpenAIAuth) HeaderName() string {
	return "api-key"
}

// AuthFactory creates authentication strategies for providers
type AuthFactory struct {
	strategies map[string]AuthStrategy
}

func NewAuthFactory() *AuthFactory {
	return &AuthFactory{
		strategies: map[string]AuthStrategy{
			"openai":    NewOpenAIAuth(),
			"anthropic": NewAnthropicAuth(),
			"gemini":    NewGeminiAuth(true),
			"azure":     NewAzureOpenAIAuth(),
		},
	}
}

// Get returns the authentication strategy for a provider
func (f *AuthFactory) Get(provider string) (AuthStrategy, error) {
	strategy, ok := f.strategies[provider]
	if !ok {
		return nil, fmt.Errorf("no auth strategy found for provider: %s", provider)
	}
	return strategy, nil
}

// Register adds a custom authentication strategy for a provider
func (f *AuthFactory) Register(provider string, strategy AuthStrategy) {
	f.strategies[provider] = strategy
}

// SupportedProviders returns a list of registered provider names
func (f *AuthFactory) SupportedProviders() []string {
	providers := make([]string, 0, len(f.strategies))
	for name := range f.strategies {
		providers = append(providers, name)
	}
	return providers
}

// AuthInjector wraps an HTTP transport to inject provider authentication
type AuthInjector struct {
	transport   http.RoundTripper
	factory     *AuthFactory
	credentials CredentialStore
}

// CredentialStore provides credentials for providers
type CredentialStore interface {
	Get(ctx context.Context, provider string) (string, error)
}

func NewAuthInjector(transport http.RoundTripper, factory *AuthFactory, credentials CredentialStore) *AuthInjector {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &AuthInjector{
		transport:   transport,
		factory:     factory,
		credentials: credentials,
	}
}

func (a *AuthInjector) RoundTrip(req *http.Request) (*http.Response, error) {
	// Extract provider from context or request
	provider := extractProviderFromRequest(req)
	if provider == "" {
		return a.transport.RoundTrip(req)
	}

	strategy, err := a.factory.Get(provider)
	if err != nil {
		return nil, fmt.Errorf("auth injector: %w", err)
	}

	credential, err := a.credentials.Get(req.Context(), provider)
	if err != nil {
		return nil, fmt.Errorf("auth injector: failed to get credentials for %s: %w", provider, err)
	}

	// Clone request to avoid modifying the original
	newReq := req.Clone(req.Context())
	if err := strategy.Apply(newReq, credential); err != nil {
		return nil, fmt.Errorf("auth injector: failed to apply auth for %s: %w", provider, err)
	}

	return a.transport.RoundTrip(newReq)
}

// extractProviderFromRequest determines the provider from request context or headers
func extractProviderFromRequest(req *http.Request) string {
	// Check context first
	if provider, ok := req.Context().Value("provider").(string); ok && provider != "" {
		return provider
	}
	// Check header
	if provider := req.Header.Get("X-Provider"); provider != "" {
		return provider
	}
	return ""
}
