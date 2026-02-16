# Golden Stack: Secrets Management Deployment Architecture

**Status**: Design Phase
**Owner**: Team Hotel (Deployment & Infrastructure)
**Target Environment**: radgateway01 (Alpha)
**Last Updated**: 2026-02-16

---

## Executive Summary

The Golden Stack provides a unified secrets management infrastructure combining three key components:

1. **PostgreSQL 16** - Shared database backend for persistent storage
2. **Infisical** - Primary secrets management (hot vault for active secrets)
3. **OpenBao** - Cold vault for long-term secrets retention

This architecture provides defense-in-depth for secrets management: Infisical for operational agility and OpenBao for compliance retention requirements.

---

## Architecture Overview

### Component Diagram

```
┌───────────────────────────────────────────────────────────────────────────────┐
│                           Host: radgateway01                                  │
│                                                                               │
│   ┌─────────────────────────────────────────────────────────────────────┐    │
│   │                        Pod: secret-stack                           │    │
│   │                    (Shared Network Namespace)                      │    │
│   │                                                                     │    │
│   │   ┌─────────────────┐                                               │    │
│   │   │   PostgreSQL 16 │  Port: 5432 (internal only)                 │    │
│   │   │   ┌──────────┐  │                                               │    │
│   │   │   │ infisical│  │  Database: infisical_db                       │    │
│   │   │   │  _db     │  │  User: infisical                             │    │
│   │   │   └──────────┘  │                                               │    │
│   │   │   ┌──────────┐  │                                               │    │
│   │   │   │ openbao  │  │  Database: openbao_db                        │    │
│   │   │   │  _db     │  │  User: openbao                               │    │
│   │   │   └──────────┘  │  Table: openbao_kv_store                     │    │
│   │   │                 │                                               │    │
│   │   │   ┌──────────┐  │                                               │    │
│   │   │   │  shared  │  │  Schema: golden_stack_shared                 │    │
│   │   │   │  _schema │  │  (for cross-service views)                   │    │
│   │   │   └──────────┘  │                                               │    │
│   │   └────────┬────────┘                                               │    │
│   │            │                                                        │    │
│   │   ┌────────┴────────┐                                               │    │
│   │   │    Infisical    │  Port: 8080 (published: 8080)               │    │
│   │   │    ┌────────┐   │  Purpose: Primary secrets management         │    │
│   │   │    │   UI   │   │  Access: http://radgateway01:8080           │    │
│   │   │    │  API   │   │                                               │    │
│   │   │    └────────┘   │  Role: Hot vault (active secrets)           │    │
│   │   └────────┬────────┘                                               │    │
│   │            │                                                        │    │
│   │   ┌────────┴────────┐                                               │    │
│   │   │    OpenBao      │  Port: 8200 (published: 8200)               │    │
│   │   │    ┌────────┐   │  Purpose: Long-term secrets storage         │    │
│   │   │    │   UI   │   │  Access: http://radgateway01:8200           │    │
│   │   │    │  API   │   │                                               │    │
│   │   │    └────────┘   │  Role: Cold vault (compliance/audit)        │    │
│   │   └─────────────────┘                                               │    │
│   │                                                                     │    │
│   └─────────────────────────────────────────────────────────────────────┘    │
│                              │                                                │
│         ┌────────────────────┼────────────────────┐                          │
│         │                    │                    │                          │
│         ▼                    ▼                    ▼                          │
│   ┌──────────┐         ┌──────────┐         ┌──────────┐                    │
│   │   API    │         │   API    │         │   API    │                    │
│   │  :8080   │         │  :8090   │         │  :8200   │                    │
│   └──────────┘         └──────────┘         └──────────┘                    │
│   Infisical API        RAD Gateway         OpenBao API                    │
│                                                                               │
└───────────────────────────────────────────────────────────────────────────────┘
```

### Data Flow Diagram

```
┌───────────────────────────────────────────────────────────────────────────────┐
│                           Secrets Lifecycle                                    │
├───────────────────────────────────────────────────────────────────────────────┤
│                                                                               │
│   Creation Phase:                                                             │
│   ┌─────────┐                                                                 │
│   │  Admin  │──┐                                                              │
│   │  User   │  │                                                              │
│   └─────────┘  │                                                              │
│                │                                                              │
│                ▼                                                              │
│   ┌─────────────────────────────────────────────────────────────────┐        │
│   │                         Infisical UI/API                        │        │
│   │  • Create secrets in projects                                   │        │
│   │  • Configure environments (dev/staging/prod)                    │        │
│   │  • Set up service tokens                                        │        │
│   └───────────────────────┬─────────────────────────────────────────┘        │
│                           │                                                   │
│                           │ Infisical auto-replication to OpenBao (optional) │
│                           ▼                                                   │
│   ┌─────────────────────────────────────────────────────────────────┐        │
│   │                         OpenBao Cold Vault                        │        │
│   │  • Long-term retention (10 years default)                       │        │
│   │  • Audit logging of all operations                              │        │
│   │  • Immutable version history                                    │        │
│   └─────────────────────────────────────────────────────────────────┘        │
│                                                                               │
│   Usage Phase:                                                                │
│   ┌─────────────┐                                                             │
│   │ RAD Gateway │──┐                                                          │
│   │  (app pod)  │  │                                                          │
│   └─────────────┘  │                                                          │
│                    │                                                          │
│                    ▼                                                          │
│   ┌─────────────────────────────────────────────────────────────────┐        │
│   │                    Infisical API (localhost:8080)                 │        │
│   │  • Fetch RAD_API_KEYS                                           │        │
│   │  • Fetch provider credentials (OpenAI, Anthropic, Gemini)      │        │
│   │  • Service token authentication                               │        │
│   └─────────────────────────────────────────────────────────────────┘        │
│                                                                               │
│   Recovery/Archive Phase:                                                     │
│   ┌─────────────┐                                                             │
│   │   Auditor   │──┐                                                          │
│   │   / Admin   │  │                                                          │
│   └─────────────┘  │                                                          │
│                    │                                                          │
│                    ▼                                                          │
│   ┌─────────────────────────────────────────────────────────────────┐        │
│   │                    OpenBao API/UI (localhost:8200)                │        │
│   │  • Access historical secret versions                          │        │
│   │  • Review audit logs                                          │        │
│   │  • Disaster recovery operations                               │        │
│   └─────────────────────────────────────────────────────────────────┘        │
│                                                                               │
└───────────────────────────────────────────────────────────────────────────────┘
```

---

## Network Configuration

### Port Allocation

| Port | Service | Direction | Purpose |
|------|---------|-----------|---------|
| 5432 | PostgreSQL | Internal (pod only) | Database connections |
| 8080 | Infisical | Published (host:8080) | Secrets API/UI |
| 8200 | OpenBao | Published (host:8200) | Vault API/UI |
| 8201 | OpenBao | Internal (future) | Cluster communication |

### Network Security Model

```
┌─────────────────────────────────────────────────────────────────┐
│                      Network Layers                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Layer 1: Pod Internal Network (localhost)                    │
│  ┌─────────────────────────────────────────────────────┐       │
│  │  • PostgreSQL accepts connections only from          │       │
│  │    localhost:5432 within the pod                     │       │
│  │  • No external access to PostgreSQL                  │       │
│  │  • SSL/TLS optional for internal                     │       │
│  └─────────────────────────────────────────────────────┘       │
│                                                                 │
│  Layer 2: Host Network (published ports)                        │
│  ┌─────────────────────────────────────────────────────┐       │
│  │  • Infisical: 8080 accessible from host              │       │
│  │  • OpenBao: 8200 accessible from host                │       │
│  │  • Firewall rules restrict external access            │       │
│  └─────────────────────────────────────────────────────┘       │
│                                                                 │
│  Layer 3: External Access (via firewall)                       │
│  ┌─────────────────────────────────────────────────────┐       │
│  │  • Infisical: Limited to trusted IPs                 │       │
│  │  • OpenBao: Restricted to admin workstations        │       │
│  │  • No direct external PostgreSQL access             │       │
│  └─────────────────────────────────────────────────────┘       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Firewall Configuration

```bash
# Allow Infisical for application servers
sudo firewall-cmd --permanent --add-rich-rule='rule family="ipv4" \
  source address="10.0.0.0/8" \
  port port="8080" protocol="tcp" accept'

# Allow OpenBao for admin workstations only
sudo firewall-cmd --permanent --add-rich-rule='rule family="ipv4" \
  source address="10.1.0.0/24" \
  port port="8200" protocol="tcp" accept'

# Reload firewall
sudo firewall-cmd --reload
```

---

## PostgreSQL Schema Requirements

### Database Architecture

```sql
-- Database separation for isolation
CREATE DATABASE infisical_db OWNER infisical;
CREATE DATABASE openbao_db OWNER openbao;

-- Optional: shared schema for cross-service views
CREATE SCHEMA golden_stack_shared AUTHORIZATION postgres;
```

### Infisical Database Schema

```sql
-- Infisical manages its own schema via migrations
-- Key tables (managed by Infisical):

-- Users and authentication
users
user_encryption_keys
auth_methods

-- Projects and organization
projects
project_members
organization
organization_members

-- Secrets storage
secrets
secret_versions
secret_tags
secret_imports

-- Service tokens
service_tokens
service_token_scopes

-- Audit and access
audit_logs
access_approval_policies
```

### OpenBao Database Schema

```sql
-- OpenBao uses a single table for its key-value store
-- Configuration in /deploy/openbao/config.hcl

-- Primary storage table
CREATE TABLE openbao_kv_store (
    parent_path TEXT COLLATE "C" NOT NULL,
    path        TEXT COLLATE "C" NOT NULL,
    key         TEXT COLLATE "C" NOT NULL,
    value       BYTEA,
    PRIMARY KEY (path, key)
);

-- High Availability table (optional, for future scaling)
CREATE TABLE openbao_ha_locks (
    node_id     TEXT PRIMARY KEY,
    lock_id     TEXT NOT NULL,
    timestamp   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_kv_parent ON openbao_kv_store(parent_path);
CREATE INDEX idx_kv_path ON openbao_kv_store(path);

-- Grants
GRANT ALL PRIVILEGES ON openbao_kv_store TO openbao;
GRANT ALL PRIVILEGES ON openbao_ha_locks TO openbao;
```

### Connection Pool Configuration

| Parameter | Infisical | OpenBao | Notes |
|-----------|-----------|---------|-------|
| max_connections | 50 | 16 | Infisical needs more for UI users |
| idle_timeout | 10m | 5m | OpenBao has bursty access patterns |
| connection_lifetime | 30m | 30m | Same to avoid stale connections |
| ssl_mode | prefer | prefer | Can be require in production |

---

## OpenBao Configuration Design

### Decision: PostgreSQL Backend vs File Backend

**RECOMMENDATION**: PostgreSQL Backend

| Factor | PostgreSQL Backend | File Backend |
|--------|-------------------|--------------|
| **Durability** | High (database replication) | Medium (filesystem) |
| **Scalability** | Horizontal via HA mode | Single node only |
| **Backup** | Standard DB backups | File-level backups |
| **Performance** | Good for cold vault | Fast for local access |
| **Recovery** | Point-in-time recovery | Snapshot-based |
| **Complexity** | Medium | Low |

**Rationale**: PostgreSQL backend provides better durability and enables future HA configuration. Since this is a cold vault for compliance, data durability is paramount.

### Deployment Mode: Dev vs Production

**DECISION**: Production Mode with Auto-unseal (Future: Shamir)

| Aspect | Dev Mode | Production Mode |
|--------|----------|-----------------|
| Initialization | Automatic | Manual unseal required |
| Security | Minimal | Full seal/unseal lifecycle |
| Auto-unseal | N/A | Required (Transit/AWS KMS) |
| Use Case | Development | Production cold vault |

**Implementation Path**:
1. Phase 1: Dev mode for initial deployment
2. Phase 2: Transition to Production with Transit auto-unseal
3. Phase 3: Consider AWS KMS for auto-unseal

### Storage Configuration

See existing configuration at `/deploy/openbao/config.hcl`:

```hcl
# Storage backend configuration - PostgreSQL
storage "postgresql" {
  connection_url = "${BAO_PG_CONNECTION_URL}"
  table = "openbao_kv_store"
  max_parallel = 16
  max_idle_connections = 4
  max_connection_lifetime = "30m"
}

# Cold vault specific settings
max_lease_ttl = "87600h"      # 10 years max lease TTL
default_lease_ttl = "43800h"  # 5 years default lease TTL
```

### Secret Engine Configuration

```bash
# Enable KV v2 for versioned secrets (Infisical replication target)
vault secrets enable -path=infisical-archive kv-v2

# Configure retention (cold vault: keep 100 versions)
vault kv metadata put -max-versions=100 -delete-version-after="3650d" \
  infisical-archive/golden-stack

# Enable audit logging
vault audit enable file path=/openbao/logs/audit.log

# Configure policy for replication
vault policy write infisical-replication - << EOF
path "infisical-archive/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
EOF
```

---

## Infisical Configuration Update

### Decision: Migrate Existing Data vs Start Fresh

**RECOMMENDATION**: Migrate Existing Data

**Rationale**:
1. Infisical is already deployed on radgateway01 with active secrets
2. Service tokens and project configurations exist
3. Migration path is straightforward with pg_dump/pg_restore
4. No disruption to running RAD Gateway

**Migration Strategy**:
1. Backup existing Infisical database
2. Create new shared PostgreSQL instance
3. Restore Infisical database to new location
4. Update Infisical configuration
5. Verify connectivity
6. Decommission old database

### PostgreSQL Connection Changes

Current (if using built-in Postgres):
```
DB_CONNECTION_URI=postgres://infisical:password@localhost:5433/infisical
```

New (shared PostgreSQL):
```
DB_CONNECTION_URI=postgres://infisical:${INFISICAL_DB_PASSWORD}@localhost:5432/infisical_db
```

### Environment Variables

```bash
# Database connection
DB_CONNECTION_URI=postgres://infisical:${INFISICAL_DB_PASSWORD}@localhost:5432/infisical_db

# Redis (if used for caching)
REDIS_URL=redis://localhost:6379

# Service tokens
SERVICE_TOKEN_ENCRYPTION_KEY=${SERVICE_TOKEN_KEY}

# OpenBao integration (future)
OPENBAO_ADDR=http://localhost:8200
OPENBAO_TOKEN=${OPENBAO_REPLICATION_TOKEN}
OPENBAO_PATH=infisical-archive
```

---

## Security Considerations

### mTLS Configuration (Future Phase)

```
Phase 1: HTTP with network isolation (current)
Phase 2: TLS certificates for external endpoints
Phase 3: mTLS between services
```

### TLS Implementation

```bash
# Generate certificates for OpenBao
cd /opt/secret-stack/certs
openssl req -x509 -nodes -days 365 -newkey rsa:4096 \
  -keyout openbao.key -out openbao.crt \
  -subj "/CN=openbao.radgateway01/O=RAD Gateway/C=US"

# Infisical TLS (if supported in version)
openssl req -x509 -nodes -days 365 -newkey rsa:4096 \
  -keyout infisical.key -out infisical.crt \
  -subj "/CN=infisical.radgateway01/O=RAD Gateway/C=US"
```

### Encryption at Rest

| Layer | Method | Implementation |
|-------|--------|----------------|
| PostgreSQL | Native TDE | Not available in PostgreSQL 16 |
| PostgreSQL | pgcrypto extension | For specific column encryption |
| Infisical | Application-level | AES-256-GCM for secrets |
| OpenBao | Shamir seals | Encryption key splitting |
| Filesystem | LUKS (future) | Full disk encryption |

### Secret Rotation Strategy

```
┌─────────────────────────────────────────────────────────────────┐
│                    Rotation Schedule                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  High-Rotation (30 days):                                      │
│  • Provider API keys (OpenAI, Anthropic, Gemini)              │
│  • Service tokens                                             │
│                                                                 │
│  Medium-Rotation (90 days):                                    │
│  • Database credentials                                       │
│  • Internal API keys                                          │
│                                                                 │
│  Low-Rotation (365 days):                                      │
│  • Encryption keys (rekeying, not rotation)                    │
│  • Certificate private keys                                     │
│                                                                 │
│  Archive Only (no rotation):                                   │
│  • Historical versions in OpenBao                               │
│  • Compliance snapshots                                         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Deployment Steps

### Phase 1: Pre-Deployment

```bash
# 1. Create directory structure
sudo mkdir -p /opt/secret-stack/{config,data,logs,certs,scripts}
sudo mkdir -p /opt/secret-stack/data/postgresql

# 2. Create user/group
sudo groupadd -r secret-stack || true
sudo useradd -r -g secret-stack -d /opt/secret-stack -s /bin/false secret-stack || true

# 3. Set permissions
sudo chown -R secret-stack:secret-stack /opt/secret-stack
sudo chmod 750 /opt/secret-stack
sudo chmod 700 /opt/secret-stack/data

# 4. Generate secrets
echo "$(openssl rand -base64 32)" | sudo tee /opt/secret-stack/config/pg-superuser-password > /dev/null
echo "$(openssl rand -base64 32)" | sudo tee /opt/secret-stack/config/infisical-db-password > /dev/null
echo "$(openssl rand -base64 32)" | sudo tee /opt/secret-stack/config/openbao-db-password > /dev/null
sudo chmod 600 /opt/secret-stack/config/*-password
```

### Phase 2: Pod Creation

```bash
# Create the secret-stack pod
sudo podman pod create \
  --name secret-stack \
  --publish 5432:5432 \
  --publish 8080:8080 \
  --publish 8200:8200 \
  --network bridge \
  --infra-name secret-stack-infra
```

### Phase 3: PostgreSQL Deployment

```bash
# Run PostgreSQL 16 container
sudo podman run -d \
  --pod secret-stack \
  --name secret-stack-postgres \
  --restart unless-stopped \
  --env POSTGRES_USER=postgres \
  --env POSTGRES_PASSWORD_FILE=/run/secrets/pg-superuser-password \
  --env POSTGRES_INITDB_ARGS="--auth-host=scram-sha-256" \
  --volume /opt/secret-stack/config/pg-superuser-password:/run/secrets/pg-superuser-password:ro \
  --volume /opt/secret-stack/data/postgresql:/var/lib/postgresql/data:Z \
  --health-cmd "pg_isready -U postgres" \
  --health-interval 10s \
  --health-timeout 5s \
  --health-retries 3 \
  docker.io/library/postgres:16-alpine

# Wait for PostgreSQL to be ready
sudo podman exec secret-stack-postgres pg_isready -U postgres
```

### Phase 4: Database Initialization

```bash
# Create databases and users
sudo podman exec -i secret-stack-postgres psql -U postgres << 'EOF'
-- Create infisical database and user
CREATE USER infisical WITH PASSWORD '${INFISICAL_DB_PASSWORD}';
CREATE DATABASE infisical_db OWNER infisical;
GRANT ALL PRIVILEGES ON DATABASE infisical_db TO infisical;

-- Create openbao database and user
CREATE USER openbao WITH PASSWORD '${OPENBAO_DB_PASSWORD}';
CREATE DATABASE openbao_db OWNER openbao;
GRANT ALL PRIVILEGES ON DATABASE openbao_db TO openbao;

-- Create OpenBao table
\c openbao_db;
CREATE TABLE openbao_kv_store (
    parent_path TEXT COLLATE "C" NOT NULL,
    path        TEXT COLLATE "C" NOT NULL,
    key         TEXT COLLATE "C" NOT NULL,
    value       BYTEA,
    PRIMARY KEY (path, key)
);
CREATE INDEX idx_kv_parent ON openbao_kv_store(parent_path);
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO openbao;

-- Create shared schema
CREATE SCHEMA golden_stack_shared;
GRANT USAGE ON SCHEMA golden_stack_shared TO infisical;
GRANT USAGE ON SCHEMA golden_stack_shared TO openbao;
EOF
```

### Phase 5: OpenBao Deployment

```bash
# Build OpenBao image (using existing Containerfile)
cd /mnt/ollama/git/RADAPI01/deploy/openbao
sudo podman build -t secret-stack/openbao:latest .

# Run OpenBao container
sudo podman run -d \
  --pod secret-stack \
  --name secret-stack-openbao \
  --restart unless-stopped \
  --env BAO_PG_HOST=localhost \
  --env BAO_PG_PORT=5432 \
  --env BAO_PG_DATABASE=openbao_db \
  --env BAO_PG_USER=openbao \
  --env BAO_PG_PASSWORD_FILE=/run/secrets/openbao-db-password \
  --env BAO_PG_CONNECTION_URL="postgres://openbao:${OPENBAO_DB_PASSWORD}@localhost:5432/openbao_db?sslmode=prefer" \
  --env BAO_API_ADDR=http://0.0.0.0:8200 \
  --env BAO_CLUSTER_ADDR=http://0.0.0.0:8201 \
  --env BAO_LOG_LEVEL=info \
  --volume /opt/secret-stack/config/openbao-db-password:/run/secrets/openbao-db-password:ro \
  --volume /opt/secret-stack/data/openbao:/openbao/data:Z \
  --volume /opt/secret-stack/logs/openbao:/openbao/logs:Z \
  --volume /mnt/ollama/git/RADAPI01/deploy/openbao/config.hcl:/openbao/config/config.hcl:ro \
  secret-stack/openbao:latest

# Initialize OpenBao (first run only)
sleep 10
curl -X PUT http://localhost:8200/v1/sys/init \
  -H "Content-Type: application/json" \
  -d '{"secret_shares":5,"secret_threshold":3}' | sudo tee /opt/secret-stack/config/openbao-init.json
```

### Phase 6: Infisical Migration

```bash
# Backup existing Infisical data
# Assuming existing Infisical uses its own PostgreSQL
pg_dump -h localhost -p 5433 -U infisical infisical > /opt/secret-stack/backup/infisical-backup.sql

# Restore to new database
sudo podman exec -i secret-stack-postgres psql -U infisical -d infisical_db < /opt/secret-stack/backup/infisical-backup.sql

# Update Infisical configuration to use new database
# (This requires stopping existing Infisical and reconfiguring)
```

### Phase 7: Health Verification

```bash
# Check PostgreSQL
sudo podman exec secret-stack-postgres pg_isready -U postgres
sudo podman exec secret-stack-postgres psql -U postgres -c "\l"

# Check OpenBao
curl http://localhost:8200/v1/sys/health

# Check Infisical (after migration)
curl http://localhost:8080/api/status
```

---

## Backup Strategy

### Backup Schedule

| Component | Frequency | Retention | Method |
|-----------|-----------|-----------|--------|
| PostgreSQL | Hourly | 7 days | pg_dump incremental |
| PostgreSQL | Daily | 30 days | pg_dump full |
| PostgreSQL | Weekly | 90 days | pg_dump compressed |
| OpenBao | On-change | 365 days | Vault export + audit logs |
| OpenBao | Daily | 30 days | PostgreSQL backup |
| Infisical | Daily | 30 days | PostgreSQL backup |

### PostgreSQL Backup Script

```bash
#!/bin/bash
# /opt/secret-stack/scripts/backup-postgres.sh

BACKUP_DIR="/backup/secret-stack/postgresql"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

# Create backup directory
mkdir -p "${BACKUP_DIR}"

# Full backup
sudo podman exec secret-stack-postgres pg_dumpall -U postgres | \
  gzip > "${BACKUP_DIR}/postgres-full-${DATE}.sql.gz"

# Individual database backups
for DB in infisical_db openbao_db; do
    sudo podman exec secret-stack-postgres pg_dump -U postgres -d "${DB}" | \
      gzip > "${BACKUP_DIR}/${DB}-${DATE}.sql.gz"
done

# Cleanup old backups
find "${BACKUP_DIR}" -name "*.sql.gz" -mtime +${RETENTION_DAYS} -delete

# Verify backup integrity
if [ -f "${BACKUP_DIR}/postgres-full-${DATE}.sql.gz" ]; then
    echo "[backup] PostgreSQL backup completed: ${DATE}"
else
    echo "[backup] PostgreSQL backup FAILED: ${DATE}" >&2
    exit 1
fi
```

### OpenBao Backup Script

```bash
#!/bin/bash
# /opt/secret-stack/scripts/backup-openbao.sh

BACKUP_DIR="/backup/secret-stack/openbao"
DATE=$(date +%Y%m%d_%H%M%S)
VAULT_ADDR="http://localhost:8200"
VAULT_TOKEN="${VAULT_ROOT_TOKEN}"

# Create backup directory
mkdir -p "${BACKUP_DIR}/${DATE}"

# Export secrets (requires root token)
export VAULT_ADDR VAULT_TOKEN

# Backup KV stores
vault kv get -format=json infisical-archive/golden-stack > \
  "${BACKUP_DIR}/${DATE}/golden-stack-secrets.json" 2>/dev/null || true

# Backup audit logs
cp /opt/secret-stack/logs/openbao/audit.log.* "${BACKUP_DIR}/${DATE}/" 2>/dev/null || true

# Backup configuration
vault read sys/config/state > "${BACKUP_DIR}/${DATE}/sys-config.txt" 2>/dev/null || true

# Archive
(cd "${BACKUP_DIR}" && tar czf "openbao-backup-${DATE}.tar.gz" "${DATE}")
rm -rf "${BACKUP_DIR}/${DATE}"

# Cleanup (keep 90 days)
find "${BACKUP_DIR}" -name "openbao-backup-*.tar.gz" -mtime +90 -delete

echo "[backup] OpenBao backup completed: ${DATE}"
```

### Disaster Recovery

```
┌─────────────────────────────────────────────────────────────────┐
│                   Recovery Scenarios                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Scenario 1: PostgreSQL Data Corruption                         │
│  ─────────────────────────────────────                          │
│  1. Stop secret-stack pod                                      │
│  2. Restore from backup:                                        │
│     zcat backup.sql.gz | psql -U postgres                      │
│  3. Restart pod                                                 │
│  4. Verify all services connect                               │
│                                                                 │
│  Scenario 2: OpenBao Seal Loss                                  │
│  ─────────────────────────────                                  │
│  1. Unseal using Shamir shards (3 of 5)                      │
│  2. If unseal keys lost: restore from backup + rekey         │
│  3. Document key holders                                       │
│                                                                 │
│  Scenario 3: Complete Host Failure                              │
│  ─────────────────────────────                                  │
│  1. Provision new host                                        │
│  2. Restore PostgreSQL from latest backup                     │
│  3. Deploy secret-stack pod                                    │
│  4. Unseal OpenBao with Shamir shards                         │
│  5. Verify Infisical data integrity                            │
│                                                                 │
│  Scenario 4: Infisical Database Loss                            │
│  ─────────────────────────────────                              │
│  1. Restore from PostgreSQL backup                              │
│  2. Restart Infisical container                                │
│  3. Verify service tokens                                      │
│  4. RAD Gateway may need token refresh                        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Monitoring and Alerting

### Health Checks

```bash
# PostgreSQL health
sudo podman exec secret-stack-postgres pg_isready -U postgres

# OpenBao health
curl -s http://localhost:8200/v1/sys/health | jq '.sealed'

# Infisical health
curl -s http://localhost:8080/api/status | jq '.status'
```

### Key Metrics

| Metric | Source | Alert Threshold |
|--------|--------|-----------------|
| PostgreSQL connections | pg_stat_activity | > 40 connections |
| OpenBao seal status | /sys/health | Sealed = CRITICAL |
| Infisical response time | /api/status | > 2s p95 |
| Disk usage | df | > 80% |
| Backup age | file mtime | > 25 hours |

---

## Appendix A: File Locations

```
/opt/secret-stack/
├── config/
│   ├── pg-superuser-password      # PostgreSQL superuser password
│   ├── infisical-db-password      # Infisical database password
│   ├── openbao-db-password        # OpenBao database password
│   ├── openbao-init.json          # OpenBao initialization data
│   └── infisical-replication-token # Infisical to OpenBao token
├── data/
│   └── postgresql/                # PostgreSQL data files
├── logs/
│   └── openbao/                   # OpenBao audit logs
├── certs/                         # TLS certificates (future)
├── scripts/
│   ├── backup-postgres.sh
│   ├── backup-openbao.sh
│   └── health-check.sh
└── systemd/
    └── secret-stack.service       # Systemd unit file
```

---

## Appendix B: Decision Log

| Date | Decision | Rationale | Status |
|------|----------|-----------|--------|
| 2026-02-16 | PostgreSQL backend for OpenBao | Durability, HA potential, backup consistency | Approved |
| 2026-02-16 | Migrate Infisical data | Existing deployment has active secrets | Approved |
| 2026-02-16 | Port 8200 for OpenBao | Standard Vault port, firewall-restricted | Approved |
| 2026-02-16 | Dev mode first, prod later | Simplifies initial deployment | Approved |
| 2026-02-16 | Shared PostgreSQL instance | Resource efficiency, single backup target | Approved |

---

## Appendix C: Future Enhancements

1. **High Availability**: Configure OpenBao HA with multiple instances
2. **Auto-unseal**: Implement Transit auto-unseal
3. **mTLS**: Enable mutual TLS between services
4. **Replication**: Automated Infisical to OpenBao secret sync
5. **Monitoring**: Prometheus metrics for all components
6. **Encryption**: LUKS for data-at-rest encryption

---

**Document Owner**: Team Hotel (Deployment & Infrastructure)
**Review Schedule**: Monthly during active development, quarterly in maintenance
**Next Review**: 2026-03-16
