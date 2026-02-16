# RAD Gateway (Brass Relay)

Go-based API gateway scaffold inspired by Plexus and AxonHub feature patterns.

## Status

`Alpha` - **radgateway01 Successfully Deployed**

The first production instance (radgateway01) has been deployed with:
- Containerized deployment via Podman
- Systemd service management
- Infisical secrets integration
- Health monitoring enabled

## Quick Start

```bash
# Clone and build
git clone <repository-url>
cd rad-gateway
go build -o rad-gateway ./cmd/rad-gateway

# Configure (create .env file)
echo "RAD_API_KEYS=your-api-key" > .env

# Run
./rad-gateway

# Verify
curl http://localhost:8090/health
```

See [docs/getting-started.md](docs/getting-started.md) for detailed setup instructions.

## Project Goal

RAD Gateway (Brass Relay) aims to be a production-capable gateway that provides a single, stable API surface across major model ecosystems while preserving compatibility with existing clients.

Current goals:

- provide OpenAI-, Anthropic-, and Gemini-compatible request surfaces
- route requests across providers with retry/failover behavior and traceable usage records
- keep operations simple with lightweight management endpoints and explicit `.env`-based secret handling
- add agent interoperability in the next phase (A2A + AG-UI, with scoped MCP integration)

## Run

```bash
go run ./cmd/rad-gateway
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

## Docs

- `docs/feature-matrix.md`
- `docs/reverse-engineering-report.md`
- `docs/product-build-blueprint.md`
- `docs/product-theme.md`
- `docs/implementation-plan.md`
- `docs/next-milestones.md`
- `docs/review-teams.md`
- `docs/protocol-stack-decision.md`
- `docs/operations/deployment-targets.md`
- `docs/team-structure-compliance.md` - Team organization (TEAM-007)
- `docs/team-system-guide.md` - Team management commands

## Team Structure (TEAM-007 Compliant)

All teams have 4-6 members. Teams are spun up/down as needed using Claude Code Team system.

| Team | Purpose | Members | Status |
|------|---------|---------|--------|
| Team Alpha | Architecture & Design | 6 | 🟢 Active |
| Team Bravo | Core Implementation | 6 | 🟢 Active |
| Team Charlie | Security Hardening | 5 | 🟢 Active |
| Team Delta | Quality Assurance | 5 | 🟢 Active |
| Team Echo | Operations & Observability | 5 | 🟢 Active |
| Team Foxtrot | Inspiration Analysis | 5 | ✅ Complete |
| Team Golf | Documentation & Design | 6 | 🟢 Active |
| **Team Hotel** | **Deployment & Infrastructure** | **5** | **🟢 Active (radgateway01)** |

### Active Team Hotel Members

| Member | Role | Task |
|--------|------|------|
| devops-lead | DevOps Lead | Verify deployment |
| container-engineer | Container Engineer | Verify containers |
| deployment-engineer | Deployment Engineer | Review scripts |
| infrastructure-architect | Infrastructure Architect | Validate infrastructure |
| systems-administrator | Systems Administrator | Check system health |

See `docs/team-system-guide.md` for team management commands.
