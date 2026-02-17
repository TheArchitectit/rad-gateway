# Integrate Logging and Infisical Secrets Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Integrate the new structured logging and Infisical secrets management packages into the RAD Gateway codebase, replacing ad-hoc logging and environment-based configuration.

**Architecture:** Initialize logger early in main.go, use Infisical client for secrets if available (fallback to env vars), propagate structured logging through all components with component attributes.

**Tech Stack:** Go 1.24, slog (structured logging), Infisical API client

---

## Prerequisites

These packages have been created:
- `internal/logger/logger.go` - Structured logging with slog
- `internal/logger/logger_test.go` - Logger tests
- `internal/secrets/infisical.go` - Infisical API client

---

## Task 1: Update main.go to Initialize Logger

**Files:**
- Modify: `cmd/rad-gateway/main.go:1-74`

**Step 1: Add logger import and initialization**

Replace imports and add logger initialization:

```go
package main

import (
	"context"
	"net/http"
	"time"

	"radgateway/internal/admin"
	"radgateway/internal/api"
	"radgateway/internal/config"
	"radgateway/internal/core"
	"radgateway/internal/logger"
	"radgateway/internal/middleware"
	"radgateway/internal/provider"
	"radgateway/internal/trace"
	"radgateway/internal/usage"
)

func main() {
	// Initialize structured logger first
	logger.Init(logger.DefaultConfig())
	log := logger.WithComponent("main")

	cfg := config.Load()

	// ... rest of function
}
```

**Step 2: Replace log.Printf and log.Fatal with structured logging**

Replace:
```go
log.Printf("rad-gateway listening on %s", cfg.ListenAddr)
```

With:
```go
log.Info("rad-gateway starting", "addr", cfg.ListenAddr)
```

Replace:
```go
if err := server.ListenAndServe(); err != nil {
    log.Fatal(err)
}
```

With:
```go
if err := server.ListenAndServe(); err != nil {
    log.Error("server failed", err)
    logger.Error("rad-gateway failed to start", err)
    return
}
```

**Step 3: Test compilation**

Run: `go build ./cmd/rad-gateway`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add cmd/rad-gateway/main.go
git commit -m "feat: initialize structured logger in main.go"
```

---

## Task 2: Add Infisical Integration to Config

**Files:**
- Modify: `internal/config/config.go:1-95`

**Step 1: Add secrets import and Infisical support**

Add to imports:
```go
import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"radgateway/internal/logger"
	"radgateway/internal/secrets"
)
```

**Step 2: Update Load() function to try Infisical first**

Replace `Load()` function:

```go
func Load() Config {
	log := logger.WithComponent("config")

	// Initialize Infisical client if token available
	infisicalCfg := secrets.LoadConfig()
	var secretClient *secrets.Client
	var err error

	if infisicalCfg.Token != "" {
		secretClient, err = secrets.NewClient(infisicalCfg)
		if err != nil {
			log.Warn("Failed to initialize Infisical client, falling back to env vars", "error", err)
		} else {
			// Test connectivity
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := secretClient.Health(ctx); err != nil {
				log.Warn("Infisical health check failed, falling back to env vars", "error", err)
				secretClient = nil
			} else {
				log.Info("Connected to Infisical for secrets management")
			}
		}
	}

	addr := getenv("RAD_LISTEN_ADDR", ":8090")
	retryBudget := getenvInt("RAD_RETRY_BUDGET", 2)

	// Try to load API keys from Infisical if available
	apiKeys := loadKeys(secretClient)

	return Config{
		ListenAddr:  addr,
		APIKeys:     apiKeys,
		ModelRoutes: loadModelRoutes(),
		RetryBudget: retryBudget,
	}
}
```

**Step 3: Update loadKeys to accept Infisical client**

Replace `loadKeys` function:

```go
func loadKeys(client *secrets.Client) map[string]string {
	log := logger.WithComponent("config")

	// If Infisical client is available, try to fetch keys from there
	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		keys, err := client.GetSecret(ctx, "api_keys")
		if err == nil && keys != "" {
			log.Info("Loaded API keys from Infisical")
			return parseKeys(keys)
		}
		log.Warn("Failed to load API keys from Infisical, falling back to env vars", "error", err)
	}

	// Fall back to environment variable
	raw := strings.TrimSpace(os.Getenv("RAD_API_KEYS"))
	if raw == "" {
		return map[string]string{}
	}
	return parseKeys(raw)
}

// parseKeys parses comma-separated key:value pairs
func parseKeys(raw string) map[string]string {
	out := map[string]string{}
	parts := strings.Split(raw, ",")
	for _, item := range parts {
		kv := strings.SplitN(strings.TrimSpace(item), ":", 2)
		if len(kv) != 2 {
			continue
		}
		name := strings.TrimSpace(kv[0])
		secret := strings.TrimSpace(kv[1])
		if name != "" && secret != "" {
			out[name] = secret
		}
	}
	return out
}
```

**Step 4: Remove old loadKeys function body**

The old `loadKeys` function body (lines 43-62) should be replaced by the above.

**Step 5: Test compilation**

Run: `go build ./internal/config`
Expected: SUCCESS

**Step 6: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add Infisical integration to config loading"
```

---

## Task 3: Add Structured Logging to Core Gateway

**Files:**
- Modify: `internal/core/gateway.go`

**Step 1: Read current gateway.go**

Read the file to understand current structure.

**Step 2: Add logger import and component logging**

Add to imports:
```go
import (
	"radgateway/internal/logger"
)
```

Add logger field to Gateway struct and initialize:

```go
type Gateway struct {
	router     *routing.Router
	usage      usage.Sink
	traceStore *trace.Store
	log        *slog.Logger  // Add this field
}

func New(router *routing.Router, usageSink usage.Sink, traceStore *trace.Store) *Gateway {
	return &Gateway{
		router:     router,
		usage:      usageSink,
		traceStore: traceStore,
		log:        logger.WithComponent("gateway"),  // Initialize here
	}
}
```

**Step 3: Replace any fmt/log calls with structured logging**

Find any logging calls and replace with g.log.Info(), g.log.Error(), etc.

**Step 4: Commit**

```bash
git add internal/core/gateway.go
git commit -m "feat: add structured logging to core gateway"
```

---

## Task 4: Add Structured Logging to API Handlers

**Files:**
- Modify: `internal/api/handlers.go`

**Step 1: Add logger import**

Add to imports:
```go
import (
	"radgateway/internal/logger"
)
```

**Step 2: Add component logger to handlers**

Add logger field to handler struct and initialize.

**Step 3: Replace log calls with structured logging**

Replace any log/fmt calls with logger methods.

**Step 4: Commit**

```bash
git add internal/api/handlers.go
git commit -m "feat: add structured logging to API handlers"
```

---

## Task 5: Add Structured Logging to Router

**Files:**
- Modify: `internal/routing/router.go`

**Step 1: Add logger import**

Add to imports.

**Step 2: Add component logger**

Add logger field to Router struct and initialize.

**Step 3: Replace log calls**

Replace any logging with structured logging.

**Step 4: Commit**

```bash
git add internal/routing/router.go
git commit -m "feat: add structured logging to router"
```

---

## Task 6: Run All Tests

**Files:**
- All test files

**Step 1: Run tests**

```bash
go test ./... -v
```

Expected: All tests pass (or identify failures to fix)

**Step 2: Fix any test failures**

Address any compilation or test failures.

**Step 3: Commit**

```bash
git commit -m "test: verify logging and secrets integration"
```

---

## Task 7: Update Documentation

**Files:**
- Create: `docs/operations/logging-configuration.md`

**Step 1: Write logging configuration documentation**

```markdown
# Logging Configuration

RAD Gateway uses structured logging via Go's `log/slog` package.

## Configuration

Set via environment variables:

| Variable | Values | Default | Description |
|----------|--------|---------|-------------|
| LOG_LEVEL | debug, info, warn, error | info | Minimum log level |
| LOG_FORMAT | json, text | json | Output format |
| LOG_OUTPUT | stdout, <path> | stdout | Output destination |
| LOG_SOURCE | true, false | false | Include source file/line |

## Component Tags

All logs include a `component` tag:
- `main` - Application startup/shutdown
- `config` - Configuration loading
- `gateway` - Core gateway operations
- `router` - Request routing
- `api` - API handlers
- `secrets` - Infisical integration

## Examples

```bash
# Debug mode with text format
LOG_LEVEL=debug LOG_FORMAT=text ./rad-gateway

# JSON output with source locations
LOG_LEVEL=info LOG_SOURCE=true ./rad-gateway
```
```

**Step 2: Commit**

```bash
git add docs/operations/logging-configuration.md
git commit -m "docs: add logging configuration guide"
```

---

## Task 8: Create Checkpoint

**Step 1: Create Radical MCP checkpoint**

```bash
# Summary: Integrated structured logging and Infisical secrets
# Key decisions:
# - Logger initialized first in main.go
# - Config tries Infisical first, falls back to env vars
# - All components use WithComponent() for structured logging
# Files modified:
# - cmd/rad-gateway/main.go
# - internal/config/config.go
# - internal/core/gateway.go
# - internal/api/handlers.go
# - internal/routing/router.go
# - docs/operations/logging-configuration.md
```

---

## Verification

- [ ] All Go files compile without errors
- [ ] All tests pass
- [ ] Logger outputs structured JSON by default
- [ ] Infisical client connects when token available
- [ ] Falls back to env vars when Infisical unavailable
- [ ] Documentation updated
