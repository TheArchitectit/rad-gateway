package provider

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCredential_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		expires  *time.Time
		expected bool
	}{
		{"no expiry", nil, false},
		{"future expiry", func() *time.Time { t := now.Add(time.Hour); return &t }(), false},
		{"past expiry", func() *time.Time { t := now.Add(-time.Hour); return &t }(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := &Credential{
				Value:     "test-key",
				Provider:  "openai",
				CreatedAt: now,
				ExpiresAt: tt.expires,
				IsActive:  true,
			}
			if got := cred.IsExpired(); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCredential_IsValid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		cred     *Credential
		expected bool
	}{
		{
			"valid active",
			&Credential{Value: "key", Provider: "openai", CreatedAt: now, IsActive: true},
			true,
		},
		{
			"inactive",
			&Credential{Value: "key", Provider: "openai", CreatedAt: now, IsActive: false},
			false,
		},
		{
			"expired",
			&Credential{Value: "key", Provider: "openai", CreatedAt: now, IsActive: true, ExpiresAt: func() *time.Time { t := now.Add(-time.Hour); return &t }()},
			false,
		},
		{
			"empty value",
			&Credential{Value: "", Provider: "openai", CreatedAt: now, IsActive: true},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cred.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMemoryCredentialStore_Get(t *testing.T) {
	store := NewMemoryCredentialStore()
	ctx := context.Background()

	// Test missing credential
	_, err := store.Get(ctx, "openai")
	if err == nil {
		t.Error("expected error for missing credential")
	}

	// Set and get credential
	cred := &Credential{
		Value:     "sk-test-key",
		Provider:  "openai",
		CreatedAt: time.Now(),
		IsActive:  true,
	}
	store.Set("openai", cred)

	val, err := store.Get(ctx, "openai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "sk-test-key" {
		t.Errorf("expected credential %q, got %q", "sk-test-key", val)
	}
}

func TestMemoryCredentialStore_GetFull(t *testing.T) {
	store := NewMemoryCredentialStore()
	ctx := context.Background()

	cred := &Credential{
		Value:     "sk-test-key",
		Provider:  "openai",
		CreatedAt: time.Now(),
		Version:   "1",
		IsActive:  true,
	}
	store.Set("openai", cred)

	full, err := store.GetFull(ctx, "openai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if full.Version != "1" {
		t.Errorf("expected version %q, got %q", "1", full.Version)
	}
}

func TestMemoryCredentialStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryCredentialStore()
	ctx := context.Background()

	// Concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			cred := &Credential{
				Value:     "key",
				Provider:  "openai",
				CreatedAt: time.Now(),
				IsActive:  true,
			}
			store.Set("openai", cred)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = store.Get(ctx, "openai")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestEnvironmentCredentialStore(t *testing.T) {
	// Set test environment variables
	t.Setenv("RAD_PROVIDER_OPENAI_API_KEY", "env-openai-key")
	t.Setenv("RAD_PROVIDER_ANTHROPIC_API_KEY", "env-anthropic-key")

	store := NewEnvironmentCredentialStore("RAD_PROVIDER_")
	ctx := context.Background()

	// Test openai key
	val, err := store.Get(ctx, "openai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "env-openai-key" {
		t.Errorf("expected %q, got %q", "env-openai-key", val)
	}

	// Test anthropic key
	val, err = store.Get(ctx, "anthropic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "env-anthropic-key" {
		t.Errorf("expected %q, got %q", "env-anthropic-key", val)
	}
}

func TestCompositeCredentialStore_Get(t *testing.T) {
	// Create first store with openai
	store1 := NewMemoryCredentialStore()
	store1.Set("openai", &Credential{
		Value:     "store1-openai",
		Provider:  "openai",
		CreatedAt: time.Now(),
		IsActive:  true,
	})

	// Create second store with anthropic
	store2 := NewMemoryCredentialStore()
	store2.Set("anthropic", &Credential{
		Value:     "store2-anthropic",
		Provider:  "anthropic",
		CreatedAt: time.Now(),
		IsActive:  true,
	})

	composite := NewCompositeCredentialStore(store1, store2)
	ctx := context.Background()

	// Should get from first store
	val, err := composite.Get(ctx, "openai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "store1-openai" {
		t.Errorf("expected %q, got %q", "store1-openai", val)
	}

	// Should get from second store (first doesn't have it)
	val, err = composite.Get(ctx, "anthropic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "store2-anthropic" {
		t.Errorf("expected %q, got %q", "store2-anthropic", val)
	}

	// Should error for missing provider
	_, err = composite.Get(ctx, "gemini")
	if err == nil {
		t.Error("expected error for missing provider")
	}
}

func TestCompositeCredentialStore_GetFull(t *testing.T) {
	store1 := NewMemoryCredentialStore()
	store1.Set("openai", &Credential{
		Value:     "key",
		Provider:  "openai",
		CreatedAt: time.Now(),
		Version:   "v1",
		IsActive:  true,
	})

	composite := NewCompositeCredentialStore(store1)
	ctx := context.Background()

	full, err := composite.GetFull(ctx, "openai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if full.Version != "v1" {
		t.Errorf("expected version %q, got %q", "v1", full.Version)
	}
}

func TestCredentialRotator_CheckRotation(t *testing.T) {
	store := NewMemoryCredentialStore()
	factory := NewAuthFactory()

	// Old credential (needs rotation)
	oldCred := &Credential{
		Value:     "key",
		Provider:  "openai",
		CreatedAt: time.Now().Add(-100 * 24 * time.Hour), // 100 days old
		IsActive:  true,
	}
	store.Set("openai", oldCred)

	// New credential (doesn't need rotation)
	newCred := &Credential{
		Value:     "key",
		Provider:  "anthropic",
		CreatedAt: time.Now(),
		IsActive:  true,
	}
	store.Set("anthropic", newCred)

	rotator := NewCredentialRotator(store, factory, 90) // 90 day rotation
	ctx := context.Background()

	needsRotation, err := rotator.CheckRotation(ctx, []string{"openai", "anthropic"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(needsRotation) != 1 || needsRotation[0] != "openai" {
		t.Errorf("expected [openai] needing rotation, got %v", needsRotation)
	}
}

func TestCredentialRotator_ValidateCredential(t *testing.T) {
	factory := NewAuthFactory()
	store := NewMemoryCredentialStore()
	rotator := NewCredentialRotator(store, factory, 90)
	ctx := context.Background()

	tests := []struct {
		name       string
		provider   string
		credential string
		wantErr    bool
	}{
		{"valid openai", "openai", "sk-test-key-12345678901234567890", false},
		{"invalid openai", "openai", "short", true},
		{"valid anthropic", "anthropic", "sk-ant-test-key-1234567890", false},
		{"invalid anthropic", "anthropic", "", true},
		{"unknown provider", "unknown", "key", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rotator.ValidateCredential(ctx, tt.provider, tt.credential)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredential() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInfisicalCredentialStore_getSecretName(t *testing.T) {
	config := InfisicalConfig{
		BaseURL: "https://test.infisical.com",
		Token:   "test-token",
	}
	store := NewInfisicalCredentialStore(config)

	tests := []struct {
		provider string
		expected string
	}{
		{"openai", "OPENAI_API_KEY"},
		{"anthropic", "ANTHROPIC_API_KEY"},
		{"gemini", "GEMINI_API_KEY"},
		{"azure", "AZURE_OPENAI_API_KEY"},
		{"custom", "CUSTOM_API_KEY"},
		{"OpenAI", "OPENAI_API_KEY"}, // case normalization
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			result := store.getSecretName(tt.provider)
			if result != tt.expected {
				t.Errorf("getSecretName(%q) = %q, want %q", tt.provider, result, tt.expected)
			}
		})
	}
}

func TestInfisicalCredentialStore_DefaultConfig(t *testing.T) {
	store := NewInfisicalCredentialStore(InfisicalConfig{})

	if store.config.BaseURL != "https://app.infisical.com" {
		t.Errorf("expected default BaseURL, got %q", store.config.BaseURL)
	}
	if store.config.Environment != "production" {
		t.Errorf("expected default Environment, got %q", store.config.Environment)
	}
	if store.config.CacheDuration != 5*time.Minute {
		t.Errorf("expected default CacheDuration, got %v", store.config.CacheDuration)
	}
}

// Mock implementations for testing

type mockCredentialStoreWithGetFull struct {
	creds map[string]*Credential
}

func newMockCredentialStoreWithGetFull() *mockCredentialStoreWithGetFull {
	return &mockCredentialStoreWithGetFull{
		creds: make(map[string]*Credential),
	}
}

func (m *mockCredentialStoreWithGetFull) Get(ctx context.Context, provider string) (string, error) {
	if cred, ok := m.creds[provider]; ok {
		return cred.Value, nil
	}
	return "", errors.New("not found")
}

func (m *mockCredentialStoreWithGetFull) GetFull(ctx context.Context, provider string) (*Credential, error) {
	if cred, ok := m.creds[provider]; ok {
		return cred, nil
	}
	return nil, errors.New("not found")
}

func (m *mockCredentialStoreWithGetFull) Refresh(ctx context.Context, provider string) error {
	return nil
}

func BenchmarkMemoryCredentialStore_Get(b *testing.B) {
	store := NewMemoryCredentialStore()
	store.Set("openai", &Credential{
		Value:     "sk-benchmark-key",
		Provider:  "openai",
		CreatedAt: time.Now(),
		IsActive:  true,
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Get(ctx, "openai")
	}
}

func BenchmarkCompositeCredentialStore_Get(b *testing.B) {
	store1 := NewMemoryCredentialStore()
	store1.Set("openai", &Credential{
		Value:     "key",
		Provider:  "openai",
		CreatedAt: time.Now(),
		IsActive:  true,
	})

	store2 := NewMemoryCredentialStore()
	composite := NewCompositeCredentialStore(store1, store2)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = composite.Get(ctx, "openai")
	}
}
