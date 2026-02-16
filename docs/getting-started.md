# Getting Started with RAD Gateway

This guide will help you get RAD Gateway (Brass Relay) running locally and in production environments.

## Prerequisites

- Go 1.24+ (for local development)
- Podman or Docker (for containerized deployment)
- Git

## Quick Start (Local Development)

### 1. Clone and Build

```bash
git clone <repository-url>
cd rad-gateway
go build -o rad-gateway ./cmd/rad-gateway
```

### 2. Configure Environment

Create a `.env` file:

```bash
# Required: API key for authentication
RAD_API_KEYS=your-api-key-here

# Optional: Provider API keys (can also use Infisical)
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
GEMINI_API_KEY=...

# Optional: Server configuration
RAD_LISTEN_ADDR=:8090
RAD_LOG_LEVEL=info
```

### 3. Run the Server

```bash
./rad-gateway
```

The server will start on port 8090 by default.

### 4. Verify Installation

```bash
curl http://localhost:8090/health
```

Expected response:
```json
{"status":"healthy"}
```

## Container Deployment

### Build Container Image

```bash
# Podman
podman build -t rad-gateway:latest .

# Docker
docker build -t rad-gateway:latest .
```

### Run Container

```bash
# Basic run
podman run -d \
  --name rad-gateway \
  -p 8090:8090 \
  -e RAD_API_KEYS=your-api-key \
  rad-gateway:latest

# With environment file
podman run -d \
  --name rad-gateway \
  -p 8090:8090 \
  --env-file .env \
  rad-gateway:latest
```

## Production Deployment

### Systemd Service

Create `/etc/systemd/system/rad-gateway.service`:

```ini
[Unit]
Description=RAD Gateway
After=network.target

[Service]
Type=simple
User=radgateway
Group=radgateway
EnvironmentFile=/etc/rad-gateway/env
ExecStart=/usr/local/bin/rad-gateway
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable rad-gateway
sudo systemctl start rad-gateway
```

### Secrets Management (Infisical)

For production deployments, RAD Gateway supports Infisical for secrets management:

1. Set up Infisical instance or use Infisical Cloud
2. Configure environment variables:

```bash
INFISICAL_API_URL=https://infisical.example.com
INFISICAL_PROJECT_SLUG=your-project-slug
INFISICAL_SERVICE_TOKEN=st.xxx.xxx
```

3. Store provider API keys in Infisical
4. RAD Gateway will fetch secrets at startup

## Configuration Reference

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RAD_LISTEN_ADDR` | Server listen address | `:8090` |
| `RAD_LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |
| `RAD_API_KEYS` | Comma-separated API keys for auth | (required) |
| `OPENAI_API_KEY` | OpenAI API key | (optional) |
| `ANTHROPIC_API_KEY` | Anthropic API key | (optional) |
| `GEMINI_API_KEY` | Google Gemini API key | (optional) |

### API Endpoints

- `GET /health` - Health check
- `GET /v1/models` - List available models
- `POST /v1/chat/completions` - OpenAI-compatible chat completions
- `POST /v1/responses` - Responses endpoint
- `POST /v1/messages` - Messages endpoint
- `POST /v1/embeddings` - Embeddings endpoint
- `GET /v0/management/config` - Management config
- `GET /v0/management/usage` - Usage statistics

## Authentication

RAD Gateway supports multiple authentication methods:

```bash
# Bearer token
curl -H "Authorization: Bearer your-api-key" http://localhost:8090/v1/models

# x-api-key header
curl -H "x-api-key: your-api-key" http://localhost:8090/v1/models

# x-goog-api-key header (Gemini-compatible)
curl -H "x-goog-api-key: your-api-key" http://localhost:8090/v1/models

# Query parameter (Gemini-compatible flows)
curl "http://localhost:8090/v1/models?key=your-api-key"
```

## Health Checks

### Application Health

```bash
curl http://localhost:8090/health
```

### Container Health

```bash
# Check container status
podman ps

# View logs
podman logs rad-gateway
```

### Systemd Status

```bash
sudo systemctl status rad-gateway
sudo journalctl -u rad-gateway -f
```

## Troubleshooting

### Port Already in Use

```bash
# Check what's using port 8090
sudo ss -tlnp | grep 8090

# Change port in .env
RAD_LISTEN_ADDR=:8091
```

### Permission Denied

```bash
# Fix data directory permissions
sudo chown -R radgateway:radgateway /var/lib/rad-gateway
sudo chmod 750 /var/lib/rad-gateway
```

### Cannot Connect to Infisical

1. Verify Infisical is accessible:
   ```bash
   curl $INFISICAL_API_URL/api/status
   ```

2. Check service token permissions

3. Verify workspace/project slug is correct

## Next Steps

- Review [deployment-targets.md](operations/deployment-targets.md) for environment-specific guidance
- See [feature-matrix.md](../feature-matrix.md) for supported features
- Check [implementation-plan.md](../implementation-plan.md) for roadmap

## Support

For issues and feature requests, please refer to the project repository.
