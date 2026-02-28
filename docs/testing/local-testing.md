# Local Testing Guide

This guide explains how to test RAD Gateway locally using Ollama.

## Prerequisites

- Go 1.21+
- Ollama installed and running
- curl and jq (optional, for pretty output)

## Quick Start

### 1. Start Ollama

```bash
# Start Ollama server
ollama serve

# In another terminal, pull a model
ollama pull llama3.2:latest
```

### 2. Configure RAD Gateway

```bash
# Copy test environment
cp .env.testing .env

# Or use the default .env and add:
# OLLAMA_ENABLED=true
# OLLAMA_BASE_URL=http://localhost:11434/v1
```

### 3. Run RAD Gateway

```bash
# Build and run
go run ./cmd/rad-gateway

# Or build first
go build -o rad-gateway ./cmd/rad-gateway
./rad-gateway
```

### 4. Run Tests

```bash
# Use the test script
./scripts/test-local.sh

# Or test manually:

# Health check
curl http://localhost:8090/health

# List models
curl -H "Authorization: Bearer test_key_for_local_testing_only_001" \
     http://localhost:8090/v1/models

# Chat completion
curl -X POST \
     -H "Authorization: Bearer test_key_for_local_testing_only_001" \
     -H "Content-Type: application/json" \
     -d '{"model":"llama3.2","messages":[{"role":"user","content":"Hello"}]}' \
     http://localhost:8090/v1/chat/completions
```

## Test API Keys

The `.env.testing` file includes these test keys:

| Name | Key | Purpose |
|------|-----|---------|
| test | `test_key_for_local_testing_only_001` | General testing |
| dev | `dev_key_for_local_testing_only_002` | Development |
| admin | `admin_key_for_local_testing_only_003` | Admin operations |
| readonly | `readonly_test_key_004` | Read-only access |

**Warning**: These keys are for local testing only. Never use them in production.

## Ollama Models

Configure available models in `.env.testing`:

```bash
OLLAMA_MODEL_LLAMA3=llama3.2:latest
OLLAMA_MODEL_MISTRAL=mistral:latest
OLLAMA_MODEL_CODELLAMA=codellama:latest
```

Models will be available at the gateway as:
- `llama3.2` → routes to Ollama `llama3.2:latest`
- `mistral` → routes to Ollama `mistral:latest`
- `codellama` → routes to Ollama `codellama:latest`

## Testing External Providers

To test with external providers (OpenAI, Anthropic, Gemini), add their API keys to `.env`:

```bash
OPENAI_API_KEY=sk-your-key
ANTHROPIC_API_KEY=sk-ant-your-key
GEMINI_API_KEY=your-key
```

The gateway will automatically register these providers and route requests accordingly.

## Troubleshooting

### Ollama Connection Failed

```bash
# Check Ollama is running
curl http://localhost:11434/api/tags

# If not running, start it
ollama serve
```

### Port Already in Use

```bash
# Find process using port 8090
lsof -i :8090

# Kill it or change RAD_LISTEN_ADDR in .env
```

### Model Not Found

```bash
# List available Ollama models
ollama list

# Pull the model
ollama pull llama3.2:latest
```

### Authentication Failed

Make sure you're using the correct API key format:
```bash
curl -H "Authorization: Bearer test_key_for_local_testing_only_001" ...
```

## Advanced Testing

### Load Testing

```bash
# Install hey (HTTP load generator)
go install github.com/rakyll/hey@latest

# Run load test
hey -n 100 -c 10 \
    -H "Authorization: Bearer test_key_for_local_testing_only_001" \
    -H "Content-Type: application/json" \
    -d '{"model":"llama3.2","messages":[{"role":"user","content":"Hi"}]}' \
    http://localhost:8090/v1/chat/completions
```

### Streaming Test

```bash
curl -N -X POST \
    -H "Authorization: Bearer test_key_for_local_testing_only_001" \
    -H "Content-Type: application/json" \
    -d '{"model":"llama3.2","messages":[{"role":"user","content":"Count to 5"}],"stream":true}' \
    http://localhost:8090/v1/chat/completions
```

### Admin API Test

```bash
# Get JWT token
TOKEN=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin"}' \
    http://localhost:8090/v1/auth/login | jq -r '.token')

# Use token for admin requests
curl -H "Authorization: Bearer $TOKEN" \
    http://localhost:8090/v0/admin/apikeys
```

## Environment Variables Reference

| Variable | Description | Default |
|----------|-------------|---------|
| `OLLAMA_ENABLED` | Enable Ollama provider | `false` |
| `OLLAMA_BASE_URL` | Ollama API URL | `http://localhost:11434/v1` |
| `OLLAMA_API_KEY` | API key (Ollama doesn't require one) | `ollama` |
| `OLLAMA_MODEL_LLAMA3` | llama3.2 model name | `llama3.2:latest` |
| `OLLAMA_MODEL_MISTRAL` | mistral model name | `mistral:latest` |
| `OLLAMA_MODEL_CODELLAMA` | codellama model name | `codellama:latest` |
| `RAD_API_KEYS` | Comma-separated test keys | (see .env.testing) |

---

**Last Updated**: 2026-02-28
