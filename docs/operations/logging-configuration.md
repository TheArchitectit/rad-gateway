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

## Usage in Code

```go
import "radgateway/internal/logger"

// Get logger with component
log := logger.WithComponent("my-component")

// Log messages
log.Info("request processed", "model", model, "duration_ms", duration)
log.Error("request failed", "error", err.Error(), "model", model)
log.Debug("debug info", "payload", payload)
```

## Integration with Infisical

When `INFISICAL_TOKEN` is configured, the logger will output:

```json
{"time":"2026-02-16T10:00:00Z","level":"INFO","msg":"Connected to Infisical for secrets management","component":"config"}
```

If Infisical is unavailable, it falls back to environment variables:

```json
{"time":"2026-02-16T10:00:00Z","level":"WARN","msg":"Infisical health check failed, falling back to env vars","component":"config","error":"connection refused"}
```
