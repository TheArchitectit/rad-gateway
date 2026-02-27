# Model Routing Configuration

## Overview

The RAD Gateway provides flexible model-to-provider routing with support for aliases, fallbacks, and provider-specific configurations. This allows you to:

- **Route model requests** to specific providers
- **Use model aliases** for easier naming
- **Configure fallbacks** for high availability
- **Manage provider-specific** model names

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  User Request   │────▶│  Model Router   │────▶│   Provider      │
│  (model name)   │     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                              │
                              ▼
                        ┌─────────────────┐
                        │  Route Table    │
                        │  - Aliases      │
                        │  - Fallbacks    │
                        │  - Capabilities │
                        └─────────────────┘
```

## Model Route Configuration

### Structure

```go
type ModelRoute struct {
    Model           string         // Canonical model name
    Provider        string         // Primary provider
    ProviderModel   string         // Provider-specific model name
    Aliases         []string       // Alternative names
    Fallbacks       []FallbackRoute // Ordered fallbacks
    Capabilities    []string       // Supported features
    CostTier        string         // low, medium, high
    Enabled         bool           // Route active status
}
```

### Default Routes

The gateway includes pre-configured routes for popular models:

#### OpenAI Models

| Model | Aliases | Provider | Cost Tier |
|-------|---------|------------|-----------|
| `gpt-4o` | `gpt-4o-latest`, `gpt-4o-2024-08-06` | openai | high |
| `gpt-4o-mini` | `gpt-4o-mini-latest` | openai | low |
| `text-embedding-3-small` | - | openai | low |

#### Anthropic Models

| Model | Aliases | Provider | Cost Tier |
|-------|---------|------------|-----------|
| `claude-3-5-sonnet` | `sonnet`, `claude-sonnet` | anthropic | medium |
| `claude-3-opus` | `opus` | anthropic | high |
| `claude-3-haiku` | `haiku`, `claude-haiku` | anthropic | low |

#### Google Gemini Models

| Model | Aliases | Provider | Cost Tier |
|-------|---------|------------|-----------|
| `gemini-1.5-pro` | `gemini-pro` | gemini | medium |
| `gemini-1.5-flash` | `gemini-flash` | gemini | low |

## Usage

### Basic Routing

```go
import "radgateway/internal/routing"

// Create router with default routes
router := routing.NewModelRouter()
router.LoadDefaultRoutes()

// Resolve a model
route, err := router.Resolve("gpt-4o")
if err != nil {
    // Handle error
}

fmt.Printf("Provider: %s\n", route.Provider)
fmt.Printf("Model: %s\n", route.ProviderModel)
```

### Using Aliases

```go
// Register route with aliases
route := routing.ModelRoute{
    Model:         "claude-3-5-sonnet",
    Provider:      "anthropic",
    ProviderModel: "claude-3-5-sonnet-20241022",
    Aliases:       []string{"sonnet", "claude-sonnet", "claude-3.5-sonnet"},
    Enabled:       true,
}

router.RegisterRoute(route)

// Resolve by alias
route, _ = router.Resolve("sonnet")
// Returns: claude-3-5-sonnet route
```

### Fallback Configuration

```go
route := routing.ModelRoute{
    Model:         "gpt-4o",
    Provider:      "openai",
    ProviderModel: "gpt-4o",
    Enabled:       true,
    Fallbacks: []routing.FallbackRoute{
        {
            Provider:      "openai",
            ProviderModel: "gpt-4o-2024-05-13", // Fallback to older version
            Weight:        90,
        },
        {
            Provider:      "azure-openai",
            ProviderModel: "gpt-4o",
            Weight:        80,
        },
    },
}

router.RegisterRoute(route)

// Get fallbacks
fallbacks, _ := router.GetFallbacks("gpt-4o")
for _, fb := range fallbacks {
    fmt.Printf("Fallback: %s/%s (weight: %d)\n",
        fb.Provider, fb.ProviderModel, fb.Weight)
}
```

### Custom Routes

```go
// Register a custom route
route := routing.ModelRoute{
    Model:         "llama2-70b",
    Provider:      "generic",
    ProviderModel: "llama2:70b",
    Aliases:       []string{"llama2"},
    Capabilities:  []string{"chat"},
    CostTier:      "low",
    Enabled:       true,
}

if err := router.RegisterRoute(route); err != nil {
    log.Fatal(err)
}
```

## Route Management

### Enable/Disable Routes

```go
// Disable a route
router.DisableRoute("gpt-4o")

// Enable a route
router.EnableRoute("gpt-4o")
```

### List Routes

```go
// List all models
models := router.ListModels()
fmt.Printf("Available models: %v\n", models)

// List models for a provider
openaiModels := router.ListModelsForProvider("openai")
fmt.Printf("OpenAI models: %v\n", openaiModels)

// List all providers
providers := router.ListProviders()
fmt.Printf("Providers: %v\n", providers)
```

### Check Aliases

```go
// Check if name is an alias
if router.IsAlias("sonnet") {
    fmt.Println("'sonnet' is an alias")
}

// Get canonical name
canonical, _ := router.GetCanonicalName("sonnet")
fmt.Printf("Canonical: %s\n", canonical) // "claude-3-5-sonnet"
```

## Case-Insensitive Routing

Model names are resolved case-insensitively:

```go
// All resolve to the same route
router.Resolve("gpt-4o")
router.Resolve("GPT-4O")
router.Resolve("Gpt-4O")
```

## Capabilities

Routes can specify capabilities for filtering:

```go
route := routing.ModelRoute{
    Model:         "gpt-4o",
    Provider:      "openai",
    Capabilities:  []string{"chat", "vision", "function-calling"},
    Enabled:       true,
}

// Later: filter by capability
for _, model := range router.ListModels() {
    route, _ := router.Resolve(model)
    if hasCapability(route, "vision") {
        fmt.Printf("%s supports vision\n", model)
    }
}
```

## Cost Tiers

Models can be categorized by cost:

| Tier | Description | Example |
|------|-------------|---------|
| `low` | Budget models | gpt-4o-mini, claude-3-haiku |
| `medium` | Balanced models | claude-3-5-sonnet, gemini-1.5-pro |
| `high` | Premium models | gpt-4o, claude-3-opus |

```go
// Get cost tier
route, _ := router.Resolve("gpt-4o")
fmt.Printf("Cost tier: %s\n", route.CostTier)
```

## Integration with Gateway Router

The model router integrates with the request router:

```go
import (
    "radgateway/internal/provider"
    "radgateway/internal/routing"
)

// Create model router
modelRouter := routing.NewModelRouter()
modelRouter.LoadDefaultRoutes()

// Create provider registry
registry := provider.NewRegistry(
    openaiAdapter,
    anthropicAdapter,
    geminiAdapter,
)

// Build route table from model routes
routeTable := make(map[string][]provider.Candidate)
for _, modelName := range modelRouter.ListModels() {
    route, _ := modelRouter.Resolve(modelName)

    candidates := []provider.Candidate{{
        Name:   route.Provider,
        Model:  route.ProviderModel,
        Weight: 100,
    }}

    // Add fallbacks
    for _, fb := range route.Fallbacks {
        candidates = append(candidates, provider.Candidate{
            Name:   fb.Provider,
            Model:  fb.ProviderModel,
            Weight: fb.Weight,
        })
    }

    routeTable[modelName] = candidates
}

// Create request router
requestRouter := routing.New(registry, routeTable, 3)
```

## Error Handling

```go
route, err := router.Resolve("unknown-model")
if err != nil {
    if strings.Contains(err.Error(), "not found") {
        // Handle unknown model
    } else if strings.Contains(err.Error(), "disabled") {
        // Handle disabled route
    }
}
```

## Testing

### Unit Testing

```go
func TestModelRouting(t *testing.T) {
    router := routing.NewModelRouter()

    route := routing.ModelRoute{
        Model:    "test-model",
        Provider: "test",
        Enabled:  true,
    }

    router.RegisterRoute(route)

    resolved, err := router.Resolve("test-model")
    if err != nil {
        t.Fatal(err)
    }

    if resolved.Provider != "test" {
        t.Errorf("Expected provider 'test', got %s", resolved.Provider)
    }
}
```

### Benchmark

```go
func BenchmarkResolve(b *testing.B) {
    router := routing.NewModelRouter()
    router.LoadDefaultRoutes()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = router.Resolve("gpt-4o")
    }
}
```

## Configuration File

You can load routes from a configuration file:

```json
{
  "routes": [
    {
      "model": "gpt-4o",
      "provider": "openai",
      "provider_model": "gpt-4o",
      "aliases": ["gpt-4o-latest"],
      "capabilities": ["chat", "vision"],
      "cost_tier": "high",
      "enabled": true
    },
    {
      "model": "claude-3-5-sonnet",
      "provider": "anthropic",
      "provider_model": "claude-3-5-sonnet-20241022",
      "aliases": ["sonnet", "claude-sonnet"],
      "capabilities": ["chat", "vision"],
      "cost_tier": "medium",
      "enabled": true,
      "fallbacks": [
        {
          "provider": "anthropic",
          "provider_model": "claude-3-5-sonnet-20240620",
          "weight": 90
        }
      ]
    }
  ]
}
```

```go
// Load from file
configFile, _ := os.Open("routes.json")
defer configFile.Close()

var config struct {
    Routes []routing.ModelRoute `json:"routes"`
}
json.NewDecoder(configFile).Decode(&config)

router := routing.NewModelRouter()
for _, route := range config.Routes {
    router.RegisterRoute(route)
}
```

## See Also

- [OpenAI Provider](./openai-provider.md)
- [Anthropic Provider](./anthropic-provider.md)
- [Provider Adapters](../architecture/provider-adapters.md)
- [Routing Architecture](../architecture/routing.md)
