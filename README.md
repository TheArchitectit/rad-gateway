# RAD Gateway (Brass Relay)

Go-based API gateway scaffold inspired by Plexus and AxonHub feature patterns.

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
