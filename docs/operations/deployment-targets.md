# Deployment Targets and Secrets (Infisical)

**Team**: Team Hotel (Deployment & Infrastructure) - 5 members
**Status**: Active - Deploying radgateway01

RAD Gateway is currently `Alpha`. The deployment goal is to keep the runtime simple (a single Go service) while standardizing on Infisical as the source of truth for secrets.

## Team Hotel Responsibilities

| Role | Responsibility |
|------|----------------|
| DevOps Lead | Infrastructure orchestration and automation |
| Container Engineer | Podman/Docker container management |
| Deployment Engineer | Release automation and deployment scripts |
| Infrastructure Architect | Infrastructure design and validation |
| Systems Administrator | Host management and system hardening |

## Deployment Targets

### Local development

- Run the service directly:

```bash
go run ./cmd/rad-gateway
```

- Use `.env` locally (git-ignored) for:
  - `RAD_API_KEYS` (temporary for local testing)
  - Infisical bootstrap values (see below)

### Alpha (single node)

- Single VM/container host.
- Run via `systemd` or Docker.
- Terminate TLS in a reverse proxy (Caddy/Nginx) if exposed beyond a trusted network.
- Ensure the host can reach Infisical over the network.

### Staging

- Same shape as production (networking + secret injection), smaller scale.
- Run CI-driven deploys.

### Production

- 2+ replicas behind a load balancer.
- Persistent stores (PostgreSQL) for usage/trace once those milestones land.
- Strict network policy between gateway, providers, and Infisical.

## Secrets: Infisical as Source of Truth

### Principle

- All application secrets should live in Infisical (provider keys, OAuth creds, etc.).
- The only secret that must exist outside Infisical is the bootstrap credential used to authenticate to Infisical.

### Local bootstrap via `.env`

These values belong in `.env` (git-ignored) and should never be committed:

- `INFISICAL_API_URL` (example: `http://<infisical-host>:8080`)
- `INFISICAL_PROJECT_SLUG`
- `INFISICAL_SERVICE_TOKEN`

If your service token cannot access the slug lookup endpoint, also set:

- `INFISICAL_WORKSPACE_ID` (workspace/project id; can be derived from `/api/v2/service-token`)

### Connectivity + Auth smoke test

Infisical commonly serves the UI and API under the same base URL.

1) Check instance health:

```bash
curl -sS -o /dev/null -w "HTTP=%{http_code}\n" "$INFISICAL_API_URL/api/status"
```

2) Verify the service token:

Infisical service tokens are formatted like `st.<...>.<...>.<...>`. For API auth, use the token *without* the last `.` segment:

```bash
ACCESS_TOKEN="${INFISICAL_SERVICE_TOKEN%.*}"
curl -sS -o /dev/null -w "HTTP=%{http_code}\n" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  "$INFISICAL_API_URL/api/v2/service-token"
```

3) Read secrets (do not print values):

```bash
ACCESS_TOKEN="${INFISICAL_SERVICE_TOKEN%.*}"

# Prefer workspace id (works even if slug lookup is forbidden for the token).
curl -sS -o /dev/null -w "HTTP=%{http_code}\n" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  "$INFISICAL_API_URL/api/v3/secrets/raw?workspaceId=${INFISICAL_WORKSPACE_ID}&environment=dev&secretPath=%2F&recursive=false&include_imports=false"
```

Expected results:

- `/api/status` returns `200`
- `/api/v2/service-token` returns `200`
- secrets endpoint returns `200`

## Runtime Injection Model (Recommended)

For deployment environments, prefer injecting secrets into the process environment (via an Infisical agent/injector or deployment tooling) so the gateway continues to load config from env vars.

This keeps the gateway's config model simple and avoids logging/handling secret values inside application code.
