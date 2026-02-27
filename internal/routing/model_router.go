// Package routing provides model-to-provider routing with aliases and fallbacks.
// Sprint 8.4: Model Routing per-provider
package routing

import (
	"fmt"
	"strings"
	"sync"
)

// ModelRoute defines routing configuration for a specific model.
type ModelRoute struct {
	// Model is the canonical model name
	Model string `json:"model"`

	// Provider is the primary provider for this model
	Provider string `json:"provider"`

	// ProviderModel is the model name used by the provider (if different)
	ProviderModel string `json:"provider_model,omitempty"`

	// Fallbacks are ordered list of fallback providers
	Fallbacks []FallbackRoute `json:"fallbacks,omitempty"`

	// Aliases are alternative names for this model
	Aliases []string `json:"aliases,omitempty"`

	// Capabilities this model supports
	Capabilities []string `json:"capabilities,omitempty"`

	// CostTier for this model (low, medium, high)
	CostTier string `json:"cost_tier,omitempty"`

	// Enabled indicates if this route is active
	Enabled bool `json:"enabled"`
}

// FallbackRoute defines a fallback provider for a model.
type FallbackRoute struct {
	Provider      string `json:"provider"`
	ProviderModel string `json:"provider_model,omitempty"`
	Weight        int    `json:"weight"`
}

// ModelRouter manages model-to-provider routing.
type ModelRouter struct {
	mu        sync.RWMutex
	routes    map[string]*ModelRoute  // canonical model -> route
	aliases   map[string]string       // alias -> canonical model
	providers map[string][]string     // provider -> []models
}

// NewModelRouter creates a new model router.
func NewModelRouter() *ModelRouter {
	return &ModelRouter{
		routes:    make(map[string]*ModelRoute),
		aliases:   make(map[string]string),
		providers: make(map[string][]string),
	}
}

// RegisterRoute adds a model route configuration.
func (r *ModelRouter) RegisterRoute(route ModelRoute) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if route.Model == "" {
		return fmt.Errorf("model name is required")
	}
	if route.Provider == "" {
		return fmt.Errorf("provider is required")
	}

	// Set default provider model if not specified
	if route.ProviderModel == "" {
		route.ProviderModel = route.Model
	}

	// Set default enabled
	if !route.Enabled {
		route.Enabled = true
	}

	// Store the route
	r.routes[route.Model] = &route

	// Register aliases
	r.aliases[route.Model] = route.Model // self-alias for lookup
	for _, alias := range route.Aliases {
		r.aliases[alias] = route.Model
	}

	// Update provider index
	r.providers[route.Provider] = append(r.providers[route.Provider], route.Model)
	for _, fb := range route.Fallbacks {
		r.providers[fb.Provider] = append(r.providers[fb.Provider], route.Model)
	}

	return nil
}

// Resolve resolves a model name to its routing configuration.
func (r *ModelRouter) Resolve(model string) (*ModelRoute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if it's an alias
	canonical, ok := r.aliases[model]
	if !ok {
		// Try case-insensitive lookup
		canonical = r.findCaseInsensitive(model)
		if canonical == "" {
			return nil, fmt.Errorf("model not found: %s", model)
		}
	}

	route, ok := r.routes[canonical]
	if !ok {
		return nil, fmt.Errorf("route not found for model: %s", model)
	}

	if !route.Enabled {
		return nil, fmt.Errorf("model route is disabled: %s", model)
	}

	return route, nil
}

// findCaseInsensitive performs case-insensitive model lookup.
func (r *ModelRouter) findCaseInsensitive(model string) string {
	lower := strings.ToLower(model)
	for alias, canonical := range r.aliases {
		if strings.ToLower(alias) == lower {
			return canonical
		}
	}
	return ""
}

// GetProviderModel returns the provider-specific model name.
func (r *ModelRouter) GetProviderModel(model string) (provider string, providerModel string, err error) {
	route, err := r.Resolve(model)
	if err != nil {
		return "", "", err
	}
	return route.Provider, route.ProviderModel, nil
}

// GetFallbacks returns ordered fallback routes for a model.
func (r *ModelRouter) GetFallbacks(model string) ([]FallbackRoute, error) {
	route, err := r.Resolve(model)
	if err != nil {
		return nil, err
	}
	return route.Fallbacks, nil
}

// ListModels returns all registered model names.
func (r *ModelRouter) ListModels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	models := make([]string, 0, len(r.routes))
	for model := range r.routes {
		models = append(models, model)
	}
	return models
}

// ListModelsForProvider returns all models available for a provider.
func (r *ModelRouter) ListModelsForProvider(provider string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	models := make([]string, 0)
	for _, model := range r.providers[provider] {
		if route, ok := r.routes[model]; ok && route.Enabled {
			models = append(models, model)
		}
	}
	return models
}

// ListProviders returns all registered providers.
func (r *ModelRouter) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]string, 0, len(r.providers))
	for provider := range r.providers {
		providers = append(providers, provider)
	}
	return providers
}

// DisableRoute disables a model route.
func (r *ModelRouter) DisableRoute(model string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	route, ok := r.routes[model]
	if !ok {
		return fmt.Errorf("route not found: %s", model)
	}

	route.Enabled = false
	return nil
}

// EnableRoute enables a model route.
func (r *ModelRouter) EnableRoute(model string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	route, ok := r.routes[model]
	if !ok {
		return fmt.Errorf("route not found: %s", model)
	}

	route.Enabled = true
	return nil
}

// IsAlias checks if a name is an alias.
func (r *ModelRouter) IsAlias(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	canonical, ok := r.aliases[name]
	if !ok {
		return false
	}
	return canonical != name
}

// GetCanonicalName returns the canonical model name for an alias.
func (r *ModelRouter) GetCanonicalName(alias string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	canonical, ok := r.aliases[alias]
	if !ok {
		return "", fmt.Errorf("alias not found: %s", alias)
	}
	return canonical, nil
}

// DefaultRoutes returns pre-configured routes for common models.
func DefaultRoutes() []ModelRoute {
	return []ModelRoute{
		// OpenAI models
		{
			Model:         "gpt-4o",
			Provider:      "openai",
			ProviderModel: "gpt-4o",
			Aliases:       []string{"gpt-4o-latest", "gpt-4o-2024-08-06"},
			Capabilities:  []string{"chat", "vision", "function-calling"},
			CostTier:      "high",
			Enabled:       true,
			Fallbacks: []FallbackRoute{
				{Provider: "openai", ProviderModel: "gpt-4o-2024-05-13", Weight: 90},
			},
		},
		{
			Model:         "gpt-4o-mini",
			Provider:      "openai",
			ProviderModel: "gpt-4o-mini",
			Aliases:       []string{"gpt-4o-mini-latest"},
			Capabilities:  []string{"chat", "vision", "function-calling"},
			CostTier:      "low",
			Enabled:       true,
		},
		{
			Model:         "text-embedding-3-small",
			Provider:      "openai",
			ProviderModel: "text-embedding-3-small",
			Capabilities:  []string{"embeddings"},
			CostTier:      "low",
			Enabled:       true,
		},
		// Anthropic models
		{
			Model:         "claude-3-5-sonnet",
			Provider:      "anthropic",
			ProviderModel: "claude-3-5-sonnet-20241022",
			Aliases:       []string{"claude-3.5-sonnet", "sonnet"},
			Capabilities:  []string{"chat", "vision"},
			CostTier:      "medium",
			Enabled:       true,
			Fallbacks: []FallbackRoute{
				{Provider: "anthropic", ProviderModel: "claude-3-5-sonnet-20240620", Weight: 90},
				{Provider: "anthropic", ProviderModel: "claude-3-sonnet-20240229", Weight: 80},
			},
		},
		{
			Model:         "claude-3-opus",
			Provider:      "anthropic",
			ProviderModel: "claude-3-opus-20240229",
			Aliases:       []string{"claude-3-opus", "opus"},
			Capabilities:  []string{"chat", "vision"},
			CostTier:      "high",
			Enabled:       true,
		},
		{
			Model:         "claude-3-haiku",
			Provider:      "anthropic",
			ProviderModel: "claude-3-haiku-20240307",
			Aliases:       []string{"haiku", "claude-haiku"},
			Capabilities:  []string{"chat"},
			CostTier:      "low",
			Enabled:       true,
		},
		// Gemini models
		{
			Model:         "gemini-1.5-pro",
			Provider:      "gemini",
			ProviderModel: "gemini-1.5-pro",
			Aliases:       []string{"gemini-pro", "gemini-1.5"},
			Capabilities:  []string{"chat", "vision"},
			CostTier:      "medium",
			Enabled:       true,
		},
		{
			Model:         "gemini-1.5-flash",
			Provider:      "gemini",
			ProviderModel: "gemini-1.5-flash",
			Aliases:       []string{"gemini-flash"},
			Capabilities:  []string{"chat", "vision"},
			CostTier:      "low",
			Enabled:       true,
		},
	}
}

// LoadDefaultRoutes loads the default model routes into the router.
func (r *ModelRouter) LoadDefaultRoutes() error {
	for _, route := range DefaultRoutes() {
		if err := r.RegisterRoute(route); err != nil {
			return err
		}
	}
	return nil
}
