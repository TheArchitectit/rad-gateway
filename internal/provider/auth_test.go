package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockCredentialStore implements CredentialStore for testing
type mockCredentialStore struct {
	creds map[string]string
}

func newMockCredentialStore() *mockCredentialStore {
	return &mockCredentialStore{
		creds: map[string]string{
			"openai":    "sk-openai-test-key-123456789",
			"anthropic": "sk-ant-test-key-123456789",
			"gemini":    "gemini-test-key-123456",
			"azure":     "azure-test-key-123456789",
		},
	}
}

func (m *mockCredentialStore) Get(ctx context.Context, provider string) (string, error) {
	if cred, ok := m.creds[provider]; ok {
		return cred, nil
	}
	return "", errors.New("credential not found")
}

func (m *mockCredentialStore) GetFull(ctx context.Context, provider string) (*Credential, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCredentialStore) Refresh(ctx context.Context, provider string) error {
	return nil
}

func TestOpenAIAuth_Apply(t *testing.T) {
	auth := NewOpenAIAuth()
	req := httptest.NewRequest("GET", "http://example.com/v1/chat", nil)

	err := auth.Apply(req, "sk-test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	authHeader := req.Header.Get("Authorization")
	expected := "Bearer sk-test-key"
	if authHeader != expected {
		t.Errorf("expected Authorization header %q, got %q", expected, authHeader)
	}
}

func TestOpenAIAuth_Validate(t *testing.T) {
	auth := NewOpenAIAuth()

	tests := []struct {
		name      string
		credential string
		wantErr   bool
	}{
		{"valid key", "sk-openai-test-key-123456789", false},
		{"empty key", "", true},
		{"short key", "short", true},
		{"exactly 20 chars", "12345678901234567890", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.Validate(tt.credential)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnthropicAuth_Apply(t *testing.T) {
	auth := NewAnthropicAuth()
	req := httptest.NewRequest("GET", "http://example.com/v1/messages", nil)

	err := auth.Apply(req, "sk-ant-test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	apiKey := req.Header.Get("x-api-key")
	if apiKey != "sk-ant-test-key" {
		t.Errorf("expected x-api-key header %q, got %q", "sk-ant-test-key", apiKey)
	}

	version := req.Header.Get("anthropic-version")
	if version != "2023-06-01" {
		t.Errorf("expected anthropic-version header %q, got %q", "2023-06-01", version)
	}
}

func TestAnthropicAuth_Validate(t *testing.T) {
	auth := NewAnthropicAuth()

	tests := []struct {
		name      string
		credential string
		wantErr   bool
	}{
		{"valid key", "sk-ant-test-key-123456789", false},
		{"empty key", "", true},
		{"short key", "short", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.Validate(tt.credential)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGeminiAuth_Apply_Header(t *testing.T) {
	auth := NewGeminiAuth(true) // use header
	req := httptest.NewRequest("GET", "http://example.com/v1/models", nil)

	err := auth.Apply(req, "gemini-test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	apiKey := req.Header.Get("x-goog-api-key")
	if apiKey != "gemini-test-key" {
		t.Errorf("expected x-goog-api-key header %q, got %q", "gemini-test-key", apiKey)
	}
}

func TestGeminiAuth_Apply_QueryParam(t *testing.T) {
	auth := NewGeminiAuth(false) // use query param
	req := httptest.NewRequest("GET", "http://example.com/v1/models", nil)

	err := auth.Apply(req, "gemini-test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	queryKey := req.URL.Query().Get("key")
	if queryKey != "gemini-test-key" {
		t.Errorf("expected query param key %q, got %q", "gemini-test-key", queryKey)
	}
}

func TestAuthFactory_Get(t *testing.T) {
	factory := NewAuthFactory()

	tests := []struct {
		provider string
		wantType string
		wantErr  bool
	}{
		{"openai", "*provider.OpenAIAuth", false},
		{"anthropic", "*provider.AnthropicAuth", false},
		{"gemini", "*provider.GeminiAuth", false},
		{"azure", "*provider.AzureOpenAIAuth", false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			strategy, err := factory.Get(tt.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && strategy == nil {
				t.Error("expected non-nil strategy")
			}
		})
	}
}

func TestAuthFactory_SupportedProviders(t *testing.T) {
	factory := NewAuthFactory()
	providers := factory.SupportedProviders()

	if len(providers) == 0 {
		t.Error("expected non-empty providers list")
	}

	// Check that known providers are included
	providerMap := make(map[string]bool)
	for _, p := range providers {
		providerMap[p] = true
	}

	expectedProviders := []string{"openai", "anthropic", "gemini", "azure"}
	for _, expected := range expectedProviders {
		if !providerMap[expected] {
			t.Errorf("expected provider %q to be in supported list", expected)
		}
	}
}

func TestAuthInjector_RoundTrip(t *testing.T) {
	// Create a mock transport that captures requests
	mockTransport := &mockRoundTripper{
		response: &http.Response{
			StatusCode: 200,
			Body:       http.NoBody,
		},
	}

	factory := NewAuthFactory()
	creds := newMockCredentialStore()

	injector := NewAuthInjector(mockTransport, factory, creds)

	// Test request with provider in header
	req := httptest.NewRequest("GET", "http://api.openai.com/v1/chat", nil)
	req.Header.Set("X-Provider", "openai")

	resp, err := injector.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Verify the request was modified with auth header
	if mockTransport.lastRequest != nil {
		authHeader := mockTransport.lastRequest.Header.Get("Authorization")
		if authHeader == "" {
			t.Error("expected Authorization header to be set")
		}
	}
}

func TestAuthInjector_UnknownProvider(t *testing.T) {
	mockTransport := &mockRoundTripper{
		response: &http.Response{
			StatusCode: 200,
			Body:       http.NoBody,
		},
	}

	factory := NewAuthFactory()
	creds := newMockCredentialStore()
	injector := NewAuthInjector(mockTransport, factory, creds)

	// Test request with unknown provider
	req := httptest.NewRequest("GET", "http://api.example.com/v1/test", nil)
	req.Header.Set("X-Provider", "unknown")

	_, err := injector.RoundTrip(req)
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestAuthInjector_MissingCredential(t *testing.T) {
	mockTransport := &mockRoundTripper{
		response: &http.Response{
			StatusCode: 200,
			Body:       http.NoBody,
		},
	}

	factory := NewAuthFactory()
	creds := &mockCredentialStore{creds: map[string]string{}} // empty credentials
	injector := NewAuthInjector(mockTransport, factory, creds)

	req := httptest.NewRequest("GET", "http://api.openai.com/v1/chat", nil)
	req.Header.Set("X-Provider", "openai")

	_, err := injector.RoundTrip(req)
	if err == nil {
		t.Error("expected error for missing credential")
	}
}

// mockRoundTripper implements http.RoundTripper for testing
type mockRoundTripper struct {
	response    *http.Response
	err         error
	lastRequest *http.Request
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.lastRequest = req
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestExtractProviderFromRequest(t *testing.T) {
	tests := []struct {
		name     string
		ctxValue any
		header   string
		expected string
	}{
		{"from context", "openai", "", "openai"},
		{"from header", nil, "anthropic", "anthropic"},
		{"context takes precedence", "gemini", "azure", "gemini"},
		{"no provider", nil, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com/test", nil)
			if tt.ctxValue != nil {
				ctx := context.WithValue(req.Context(), "provider", tt.ctxValue)
				req = req.WithContext(ctx)
			}
			if tt.header != "" {
				req.Header.Set("X-Provider", tt.header)
			}

			result := extractProviderFromRequest(req)
			if result != tt.expected {
				t.Errorf("expected provider %q, got %q", tt.expected, result)
			}
		})
	}
}

func BenchmarkOpenAIAuth_Apply(b *testing.B) {
	auth := NewOpenAIAuth()
	req := httptest.NewRequest("GET", "http://example.com/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.Apply(req, "sk-test-key-12345678901234567890")
	}
}

func BenchmarkAuthFactory_Get(b *testing.B) {
	factory := NewAuthFactory()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		factory.Get("openai")
	}
}
