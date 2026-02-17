# RAD Gateway (Brass Relay)

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Production-ready Go API gateway providing unified OpenAI-compatible access to multiple AI providers (OpenAI, Anthropic, Google Gemini).

## Status

`Alpha` - **Phase 5 Complete: The Integrators**

RAD Gateway has completed 5 major development phases:

- **Phase 1: The Architects** - Requirements & Schema Design ✅
- **Phase 2: The UI/UX Core** - Frontend Foundation ✅
- **Phase 3: The Backend Core** - API & Logic ✅
- **Phase 4: The Data Wardens** - Database & Modeling ✅
- **Phase 5: The Integrators** - Frontend/Backend Integration ✅
- **Phase 6: The Sentinels** - Security Hardening (Next)

### Recent Achievements

- **Database Layer**: SQLite + PostgreSQL with migrations
- **RBAC System**: Role-based access control (Admin/Developer/Viewer)
- **Cost Tracking**: Calculator, aggregator, background worker
- **Admin API**: Projects, API keys, usage, costs, quotas, providers
- **Frontend Skeleton**: React + Zustand + TypeScript
- **Security**: Fixed critical auth bypass vulnerability
- **Performance**: Query optimization, slow query detection
- **CORS**: Full cross-origin support with configurable origins
- **JWT Authentication**: Login/logout/refresh with httpOnly cookies
- **Real-time Updates**: SSE for metrics, provider health, alerts
- **Data Fetching**: TanStack Query with automatic caching

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

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Client Layer                          │
│  (OpenAI SDK, Anthropic SDK, curl, custom clients)          │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                      API Gateway                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │   Routing   │  │    RBAC     │  │   Quotas    │       │
│  └─────────────┘  └─────────────┘  └─────────────┘       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │   Auth      │  │ Cost Track  │  │   Stream    │       │
│  └─────────────┘  └─────────────┘  └─────────────┘       │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                     Provider Adapters                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │   OpenAI    │  │  Anthropic  │  │    Gemini   │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
```

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

### Public Endpoints

- `GET /health` - Health check

### API Endpoints (Authenticated)

- `POST /v1/chat/completions` - Chat completions
- `POST /v1/responses` - Response API
- `POST /v1/messages` - Messages API (Anthropic-compatible)
- `POST /v1/embeddings` - Embeddings
- `POST /v1/images/generations` - Image generation
- `POST /v1/audio/transcriptions` - Audio transcription
- `GET /v1/models` - List available models
- `POST /v1beta/models/{model}:{action}` - Gemini compatibility

### Admin Endpoints (Authenticated)

**Projects:**
- `GET /v0/admin/projects` - List workspaces
- `POST /v0/admin/projects` - Create workspace
- `GET /v0/admin/projects/{id}` - Get workspace
- `PUT /v0/admin/projects/{id}` - Update workspace
- `DELETE /v0/admin/projects/{id}` - Delete workspace

**API Keys:**
- `GET /v0/admin/apikeys` - List API keys
- `POST /v0/admin/apikeys` - Create API key
- `GET /v0/admin/apikeys/{id}` - Get API key
- `PUT /v0/admin/apikeys/{id}` - Update API key
- `DELETE /v0/admin/apikeys/{id}` - Delete API key
- `POST /v0/admin/apikeys/{id}/revoke` - Revoke API key

**Usage & Costs:**
- `GET /v0/admin/usage` - Usage queries
- `POST /v0/admin/usage` - Advanced usage queries
- `GET /v0/admin/costs` - Cost summary
- `GET /v0/admin/costs/trends` - Cost trends
- `GET /v0/admin/costs/forecast` - Cost forecasting

**Quotas:**
- `GET /v0/admin/quotas` - List quotas
- `POST /v0/admin/quotas` - Create quota
- `GET /v0/admin/quotas/usage` - Quota usage

**Providers:**
- `GET /v0/admin/providers` - List providers
- `POST /v0/admin/providers/health` - Health check

### Planned (Phase 6+)

- `GET /.well-known/agent.json`
- `POST /a2a/tasks/send`
- `POST /a2a/tasks/sendSubscribe`
- `GET /a2a/tasks/{taskId}`
- `POST /a2a/tasks/{taskId}/cancel`
- `GET /v1/agents/{agentId}/stream`

## Database

RAD Gateway supports both SQLite (development) and PostgreSQL (production):

```go
// SQLite for development
db, err := db.New(db.Config{
    Driver: "sqlite",
    DSN:    "radgateway.db",
})

// PostgreSQL for production
db, err := db.New(db.Config{
    Driver:       "postgres",
    DSN:          "postgres://user:pass@localhost/radgateway?sslmode=require",
    MaxOpenConns: 25,
})
```

### Running Migrations

```bash
# Run migrations
go run ./cmd/migrate up

# Rollback migrations
go run ./cmd/migrate down

# Seed database
go run ./cmd/seed --scenario development
```

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

## Frontend

RAD Gateway includes a React-based Admin UI:

```bash
cd web
npm install
npm run dev
```

The Admin UI provides:
- Real-time Control Rooms
- Project Management
- API Key Management
- Usage Analytics
- Cost Tracking
- Provider Health Monitoring

## Documentation

- [docs/getting-started.md](docs/getting-started.md) - Deployment guide
- [docs/feature-matrix.md](docs/feature-matrix.md) - Supported features
- [docs/implementation-plan.md](docs/implementation-plan.md) - Roadmap
- [docs/frontend/admin-ui-feature-specification.md](docs/frontend/admin-ui-feature-specification.md) - Frontend spec
- [docs/architecture/ARCHITECTURE_SYNTHESIS_REPORT.md](docs/architecture/ARCHITECTURE_SYNTHESIS_REPORT.md) - Architecture
- [SECURITY.md](SECURITY.md) - Security policy

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for release history.

## License

See LICENSE file for details.
