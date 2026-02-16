# RAD Gateway (Brass Relay)

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Production-ready Go API gateway providing unified OpenAI-compatible access to multiple AI providers (OpenAI, Anthropic, Google Gemini).

## Status

`Alpha` - **Successfully Deployed**

RAD Gateway is deployed with:
- Containerized deployment via Docker/Podman
- Systemd service management (optional)
- Infisical secrets integration
- Health monitoring enabled

## Quick Start (Docker/Podman)

```bash
# Clone repository
git clone <repository-url>
cd rad-gateway

# Build container image
podman build -t rad-gateway:latest .

# Run with environment variables
podman run -d \
  --name rad-gateway \
  -p 8090:8090 \
  -e RAD_API_KEYS=your-api-key \
  rad-gateway:latest

# Verify
curl http://localhost:8090/health
```

See [docs/getting-started.md](docs/getting-started.md) for detailed deployment options including systemd service configuration and Infisical secrets management.

## Project Goal

RAD Gateway (Brass Relay) aims to be a production-capable gateway that provides a single, stable API surface across major model ecosystems while preserving compatibility with existing clients.

Current goals:

- provide OpenAI-, Anthropic-, and Gemini-compatible request surfaces
- route requests across providers with retry/failover behavior and traceable usage records
- keep operations simple with lightweight management endpoints and explicit `.env`-based secret handling
- add agent interoperability in the next phase (A2A + AG-UI, with scoped MCP integration)

## Development

For local development (requires Go 1.24+):

```bash
# Clone and build
git clone <repository-url>
cd rad-gateway
go build -o rad-gateway ./cmd/rad-gateway

# Configure (create .env file)
echo "RAD_API_KEYS=your-api-key" > .env

# Run
./rad-gateway
```

Server listens on `:8090` by default.

## Auth

- Keys are loaded from `RAD_API_KEYS` (set via `.env`)
- Supported key forms:
  - `Authorization: Bearer <key>`
  - `x-api-key: <key>`
  - `x-goog-api-key: <key>`
  - `?key=<key>` for Gemini-compatible flows

Example `.env` value:

```bash
RAD_API_KEYS=default:replace-with-real-key
```

## Endpoints

- `GET /health`
- `POST /v1/chat/completions`
- `POST /v1/responses`
- `POST /v1/messages`
- `POST /v1/embeddings`
- `POST /v1/images/generations`
- `POST /v1/audio/transcriptions`
- `GET /v1/models`
- `POST /v1beta/models/{model}:{action}`
- `GET /v0/management/config`
- `GET /v0/management/usage`
- `GET /v0/management/traces`

Planned (next phase):

- `GET /.well-known/agent.json`
- `POST /a2a/tasks/send`
- `POST /a2a/tasks/sendSubscribe`
- `GET /a2a/tasks/{taskId}`
- `POST /a2a/tasks/{taskId}/cancel`
- `GET /v1/agents/{agentId}/stream`

## Deployment

### Production Deploy (Recommended)

For production deployments with systemd and Infisical secrets management:

```bash
cd deploy
sudo ./install.sh
```

See [deploy/README.md](deploy/README.md) for detailed production deployment instructions.

### Quick Deploy (Docker/Podman)

```bash
# Build container image
podman build -t rad-gateway:latest .

# Run with environment variables
podman run -d \
  --name rad-gateway \
  -p 8090:8090 \
  -e RAD_API_KEYS=your-api-key \
  rad-gateway:latest
```

## Documentation

- [docs/getting-started.md](docs/getting-started.md) - Deployment guide
- [docs/feature-matrix.md](docs/feature-matrix.md) - Supported features
- [docs/implementation-plan.md](docs/implementation-plan.md) - Roadmap
- [SECURITY.md](SECURITY.md) - Security policy

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## License

See LICENSE file for details.
