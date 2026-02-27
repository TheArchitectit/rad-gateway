// Package routing provides tests for model routing.
package routing

import (
	"testing"
)

func TestModelRouter_RegisterRoute(t *testing.T) {
	router := NewModelRouter()

	route := ModelRoute{
		Model:         "gpt-4o",
		Provider:      "openai",
		ProviderModel: "gpt-4o",
		Aliases:       []string{"gpt-4o-latest"},
		Capabilities:  []string{"chat"},
		Enabled:       true,
	}

	err := router.RegisterRoute(route)
	if err != nil {
		t.Fatalf("Failed to register route: %v", err)
	}

	// Verify route was registered
	resolved, err := router.Resolve("gpt-4o")
	if err != nil {
		t.Fatalf("Failed to resolve route: %v", err)
	}

	if resolved.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got %q", resolved.Provider)
	}
}

func TestModelRouter_ResolveAlias(t *testing.T) {
	router := NewModelRouter()

	route := ModelRoute{
		Model:         "claude-3-5-sonnet",
		Provider:      "anthropic",
		ProviderModel: "claude-3-5-sonnet-20241022",
		Aliases:       []string{"sonnet", "claude-sonnet"},
		Enabled:       true,
	}

	if err := router.RegisterRoute(route); err != nil {
		t.Fatalf("Failed to register route: %v", err)
	}

	// Resolve by alias
	resolved, err := router.Resolve("sonnet")
	if err != nil {
		t.Fatalf("Failed to resolve alias: %v", err)
	}

	if resolved.Model != "claude-3-5-sonnet" {
		t.Errorf("Expected model 'claude-3-5-sonnet', got %q", resolved.Model)
	}

	// Resolve by canonical name
	resolved, err = router.Resolve("claude-3-5-sonnet")
	if err != nil {
		t.Fatalf("Failed to resolve canonical: %v", err)
	}

	if resolved.Provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got %q", resolved.Provider)
	}
}

func TestModelRouter_CaseInsensitive(t *testing.T) {
	router := NewModelRouter()

	route := ModelRoute{
		Model:    "GPT-4o",
		Provider: "openai",
		Enabled:  true,
	}

	if err := router.RegisterRoute(route); err != nil {
		t.Fatalf("Failed to register route: %v", err)
	}

	// Resolve with different case
	resolved, err := router.Resolve("gpt-4o")
	if err != nil {
		t.Fatalf("Failed to resolve case-insensitive: %v", err)
	}

	if resolved.Model != "GPT-4o" {
		t.Errorf("Expected model 'GPT-4o', got %q", resolved.Model)
	}
}

func TestModelRouter_GetProviderModel(t *testing.T) {
	router := NewModelRouter()

	route := ModelRoute{
		Model:         "gpt-4o",
		Provider:      "openai",
		ProviderModel: "gpt-4o-2024-08-06",
		Enabled:       true,
	}

	if err := router.RegisterRoute(route); err != nil {
		t.Fatalf("Failed to register route: %v", err)
	}

	provider, providerModel, err := router.GetProviderModel("gpt-4o")
	if err != nil {
		t.Fatalf("Failed to get provider model: %v", err)
	}

	if provider != "openai" {
		t.Errorf("Expected provider 'openai', got %q", provider)
	}

	if providerModel != "gpt-4o-2024-08-06" {
		t.Errorf("Expected provider model 'gpt-4o-2024-08-06', got %q", providerModel)
	}
}

func TestModelRouter_Fallbacks(t *testing.T) {
	router := NewModelRouter()

	route := ModelRoute{
		Model:    "test-model",
		Provider: "primary",
		Enabled:  true,
		Fallbacks: []FallbackRoute{
			{Provider: "fallback1", Weight: 90},
			{Provider: "fallback2", Weight: 80},
		},
	}

	if err := router.RegisterRoute(route); err != nil {
		t.Fatalf("Failed to register route: %v", err)
	}

	fallbacks, err := router.GetFallbacks("test-model")
	if err != nil {
		t.Fatalf("Failed to get fallbacks: %v", err)
	}

	if len(fallbacks) != 2 {
		t.Errorf("Expected 2 fallbacks, got %d", len(fallbacks))
	}

	if fallbacks[0].Provider != "fallback1" {
		t.Errorf("Expected first fallback 'fallback1', got %q", fallbacks[0].Provider)
	}
}

func TestModelRouter_DisableEnable(t *testing.T) {
	router := NewModelRouter()

	route := ModelRoute{
		Model:    "test-model",
		Provider: "test",
		Enabled:  true,
	}

	if err := router.RegisterRoute(route); err != nil {
		t.Fatalf("Failed to register route: %v", err)
	}

	// Disable route
	if err := router.DisableRoute("test-model"); err != nil {
		t.Fatalf("Failed to disable route: %v", err)
	}

	// Should fail to resolve disabled route
	_, err := router.Resolve("test-model")
	if err == nil {
		t.Error("Expected error for disabled route")
	}

	// Enable route
	if err := router.EnableRoute("test-model"); err != nil {
		t.Fatalf("Failed to enable route: %v", err)
	}

	// Should resolve now
	_, err = router.Resolve("test-model")
	if err != nil {
		t.Errorf("Failed to resolve enabled route: %v", err)
	}
}

func TestModelRouter_IsAlias(t *testing.T) {
	router := NewModelRouter()

	route := ModelRoute{
		Model:    "claude-3-5-sonnet",
		Provider: "anthropic",
		Aliases:  []string{"sonnet"},
		Enabled:  true,
	}

	if err := router.RegisterRoute(route); err != nil {
		t.Fatalf("Failed to register route: %v", err)
	}

	if !router.IsAlias("sonnet") {
		t.Error("Expected 'sonnet' to be an alias")
	}

	if router.IsAlias("claude-3-5-sonnet") {
		t.Error("Expected canonical name not to be an alias")
	}
}

func TestModelRouter_GetCanonicalName(t *testing.T) {
	router := NewModelRouter()

	route := ModelRoute{
		Model:    "claude-3-5-sonnet",
		Provider: "anthropic",
		Aliases:  []string{"sonnet"},
		Enabled:  true,
	}

	if err := router.RegisterRoute(route); err != nil {
		t.Fatalf("Failed to register route: %v", err)
	}

	canonical, err := router.GetCanonicalName("sonnet")
	if err != nil {
		t.Fatalf("Failed to get canonical name: %v", err)
	}

	if canonical != "claude-3-5-sonnet" {
		t.Errorf("Expected canonical 'claude-3-5-sonnet', got %q", canonical)
	}
}

func TestModelRouter_ListModelsForProvider(t *testing.T) {
	router := NewModelRouter()

	routes := []ModelRoute{
		{Model: "model-1", Provider: "openai", Enabled: true},
		{Model: "model-2", Provider: "openai", Enabled: true},
		{Model: "model-3", Provider: "anthropic", Enabled: true},
	}

	for _, route := range routes {
		if err := router.RegisterRoute(route); err != nil {
			t.Fatalf("Failed to register route: %v", err)
		}
	}

	openaiModels := router.ListModelsForProvider("openai")
	if len(openaiModels) != 2 {
		t.Errorf("Expected 2 OpenAI models, got %d", len(openaiModels))
	}

	anthropicModels := router.ListModelsForProvider("anthropic")
	if len(anthropicModels) != 1 {
		t.Errorf("Expected 1 Anthropic model, got %d", len(anthropicModels))
	}
}

func TestModelRouter_LoadDefaultRoutes(t *testing.T) {
	router := NewModelRouter()

	err := router.LoadDefaultRoutes()
	if err != nil {
		t.Fatalf("Failed to load default routes: %v", err)
	}

	// Check some expected models
	models := []string{"gpt-4o", "claude-3-5-sonnet", "gemini-1.5-pro"}
	for _, model := range models {
		_, err := router.Resolve(model)
		if err != nil {
			t.Errorf("Failed to resolve %s: %v", model, err)
		}
	}

	// Check aliases
	_, err = router.Resolve("sonnet") // alias for claude-3-5-sonnet
	if err != nil {
		t.Errorf("Failed to resolve 'sonnet' alias: %v", err)
	}
}

func TestModelRouter_Resolve_NotFound(t *testing.T) {
	router := NewModelRouter()

	_, err := router.Resolve("non-existent-model")
	if err == nil {
		t.Error("Expected error for non-existent model")
	}
}

func TestModelRouter_RegisterRoute_Invalid(t *testing.T) {
	router := NewModelRouter()

	// Missing model
	route := ModelRoute{
		Provider: "openai",
		Enabled:  true,
	}

	err := router.RegisterRoute(route)
	if err == nil {
		t.Error("Expected error for missing model")
	}

	// Missing provider
	route = ModelRoute{
		Model:   "gpt-4o",
		Enabled: true,
	}

	err = router.RegisterRoute(route)
	if err == nil {
		t.Error("Expected error for missing provider")
	}
}

func BenchmarkModelRouter_Resolve(b *testing.B) {
	router := NewModelRouter()
	router.LoadDefaultRoutes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = router.Resolve("gpt-4o")
	}
}

func BenchmarkModelRouter_ResolveAlias(b *testing.B) {
	router := NewModelRouter()
	router.LoadDefaultRoutes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = router.Resolve("sonnet") // alias
	}
}
