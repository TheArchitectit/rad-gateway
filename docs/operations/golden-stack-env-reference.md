# Golden Stack Environment Variable Reference

**Document ID**: OPS-GSE-001
**Version**: 1.0
**Last Updated**: 2026-02-16
**Owner**: Team Hotel (Deployment & Infrastructure)

---

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Variable Categories](#variable-categories)
4. [Required Variables](#required-variables)
5. [Optional Variables](#optional-variables)
6. [Security Considerations](#security-considerations)
7. [Environment-Specific Guidelines](#environment-specific-guidelines)
8. [Troubleshooting](#troubleshooting)
9. [Reference Tables](#reference-tables)

---

## Overview

This document provides a comprehensive reference for all environment variables used in the Golden Stack deployment. The Golden Stack consists of three primary components:

- **PostgreSQL**: Shared database backend
- **Infisical**: Hot vault for active secrets management
- **OpenBao**: Cold vault for long-term secrets retention

For architecture details, see: `/mnt/ollama/git/RADAPI01/docs/operations/golden-stack-deployment.md`

---

## Quick Start

### Development Setup

```bash
# 1. Navigate to golden-stack directory
cd /mnt/ollama/git/RADAPI01/deploy/golden-stack

# 2. The .env file already contains safe dev defaults
# Edit if needed:
# nano .env

# 3. Load and validate configuration
source ./env-config.sh

# 4. Generate any missing passwords
source ./env-config.sh --generate

# 5. View configuration summary
source ./env-config.sh --summary
```

### Production Setup

```bash
# 1. Copy example to .env and edit with real values
cp .env.example .env

# 2. Set strong passwords (minimum 32 characters)
export POSTGRES_PASSWORD=$(openssl rand -base64 32)
export INFISICAL_ENCRYPTION_KEY=$(openssl rand -base64 32)

# 3. Validate configuration
source ./env-config.sh --check

# 4. Export for systemd
source ./env-config.sh --export
```

---

## Variable Categories

Variables are organized by component:

| Category | Prefix | Component |
|----------|--------|-----------|
| Database | `POSTGRES_` | PostgreSQL shared database |
| Hot Vault | `INFISICAL_` | Infisical secrets management |
| Cold Vault | `OPENBAO_` | OpenBao long-term storage |
| Gateway | `RAD_` | RAD Gateway API service |
| Operations | `GOLDEN_STACK_`, `BACKUP_` | Deployment and operational |

---

## Required Variables

### PostgreSQL Configuration

These variables must be set for all deployments.

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `POSTGRES_USER` | Database username | `secretstack` | `secretstack` |
| `POSTGRES_PASSWORD` | Database password | *(none)* | *(32+ char random)* |
| `POSTGRES_DB` | Database name | `secrets` | `secrets` |
| `POSTGRES_HOST` | Database hostname | `localhost` | `postgres.internal` |
| `POSTGRES_PORT` | Database port | `5432` | `5432` |

**Security Note**: `POSTGRES_PASSWORD` must be at least 32 characters in production. Use the `env-config.sh` helper to generate secure passwords.

### Infisical Configuration (Hot Vault)

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `INFISICAL_DB_URL` | PostgreSQL connection URL | *(constructed)* | `postgresql://user:pass@host:5432/db` |
| `INFISICAL_ENCRYPTION_KEY` | AES-256 encryption key | *(none)* | *(base64 32-byte)* |

**Generating Encryption Key**:
```bash
openssl rand -base64 32
```

### OpenBao Configuration (Cold Vault)

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `OPENBAO_DB_URL` | PostgreSQL connection URL | *(constructed)* | `postgresql://user:pass@host:5432/db` |
| `OPENBAO_API_ADDR` | OpenBao API bind address | `http://0.0.0.0:8200` | `http://0.0.0.0:8200` |

---

## Optional Variables

### PostgreSQL Optional

| Variable | Description | Default | Values |
|----------|-------------|---------|--------|
| `POSTGRES_SSL_MODE` | SSL connection mode | `prefer` | `disable`, `prefer`, `require`, `verify-ca`, `verify-full` |

### Infisical Optional

| Variable | Description | Default | Notes |
|----------|-------------|---------|-------|
| `INFISICAL_API_URL` | Infisical API endpoint | `http://localhost:8080` | Must be reachable from RAD Gateway |
| `INFISICAL_UI_URL` | Infisical UI endpoint | `http://localhost:8080` | Same as API in most deployments |
| `INFISICAL_SERVICE_TOKEN` | Service token for RAD Gateway | *(none)* | Format: `st.xxx.yyy.zzz` |
| `INFISICAL_PROJECT_SLUG` | Project identifier | *(none)* | Found in Infisical UI |
| `INFISICAL_WORKSPACE_ID` | Workspace UUID | *(none)* | Required for some API operations |
| `INFISICAL_JWT_SECRET` | JWT signing secret | *(generated)* | Min 32 characters |
| `INFISICAL_JWT_TTL` | JWT token lifetime | `86400` | Seconds (24 hours) |

**SMTP Configuration (Optional)**:

| Variable | Description | Default |
|----------|-------------|---------|
| `INFISICAL_SMTP_HOST` | SMTP server hostname | *(none)* |
| `INFISICAL_SMTP_PORT` | SMTP server port | `587` |
| `INFISICAL_SMTP_USER` | SMTP authentication user | *(none)* |
| `INFISICAL_SMTP_PASSWORD` | SMTP authentication password | *(none)* |
| `INFISICAL_SMTP_FROM` | From email address | `noreply@example.com` |

### OpenBao Optional

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENBAO_CLUSTER_ADDR` | Cluster bind address | `http://0.0.0.0:8201` |
| `OPENBAO_UI_ENABLED` | Enable web UI | `true` |
| `OPENBAO_COLD_VAULT_RETENTION_DAYS` | Secret retention period | `3650` (10 years) |
| `OPENBAO_COLD_VAULT_MAX_VERSIONS` | Max versions per secret | `100` |
| `OPENBAO_UNSEAL_KEY_1` | Unseal key 1 (Shamir) | *(none)* |
| `OPENBAO_UNSEAL_KEY_2` | Unseal key 2 | *(none)* |
| `OPENBAO_UNSEAL_KEY_3` | Unseal key 3 | *(none)* |
| `OPENBAO_UNSEAL_KEY_4` | Unseal key 4 | *(none)* |
| `OPENBAO_UNSEAL_KEY_5` | Unseal key 5 | *(none)* |
| `OPENBAO_ROOT_TOKEN` | Initial root token | *(generated)* |

### RAD Gateway Optional

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `RAD_LISTEN_ADDR` | Gateway bind address | `:8090` | `0.0.0.0:8090` |
| `RAD_LOG_LEVEL` | Logging verbosity | `info` | `debug`, `info`, `warn`, `error` |
| `RAD_ENVIRONMENT` | Deployment environment | `alpha` | `dev`, `alpha`, `staging`, `prod` |
| `RAD_API_KEYS` | API authentication keys | *(none)* | `name:secret,name2:secret2` |
| `INFISICAL_TOKEN_FILE` | Path to token file | `/opt/radgateway01/config/infisical-token` | Path |
| `RAD_DATA_DIR` | Data directory path | `/data` | `/opt/radgateway01/data` |
| `OPENAI_API_KEY` | OpenAI provider key | *(from Infisical)* | `sk-...` |
| `ANTHROPIC_API_KEY` | Anthropic provider key | *(from Infisical)* | `sk-ant-...` |
| `GEMINI_API_KEY` | Gemini provider key | *(from Infisical)* | `...` |

### Golden Stack Operational

| Variable | Description | Default |
|----------|-------------|---------|
| `GOLDEN_STACK_NETWORK` | Container network name | `secret-stack` |
| `GOLDEN_STACK_SUBNET` | Container subnet | `10.88.0.0/16` |

### Container Resource Limits

| Variable | Description | Default |
|----------|-------------|---------|
| `POSTGRES_MEMORY_LIMIT` | PostgreSQL memory limit | `2g` |
| `INFISICAL_MEMORY_LIMIT` | Infisical memory limit | `1g` |
| `OPENBAO_MEMORY_LIMIT` | OpenBao memory limit | `1g` |

### Backup Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `BACKUP_ENABLED` | Enable automated backups | `true` |
| `BACKUP_SCHEDULE` | Cron schedule expression | `0 2 * * *` (2 AM daily) |
| `BACKUP_RETENTION_DAYS` | Backup retention period | `30` |
| `BACKUP_S3_BUCKET` | S3 bucket for backups | *(none)* |
| `BACKUP_S3_REGION` | S3 region | *(none)* |
| `BACKUP_S3_ACCESS_KEY` | S3 access key | *(none)* |
| `BACKUP_S3_SECRET_KEY` | S3 secret key | *(none)* |

### Monitoring

| Variable | Description | Default |
|----------|-------------|---------|
| `METRICS_ENABLED` | Enable metrics endpoint | `true` |
| `METRICS_PORT` | Metrics server port | `9090` |
| `HEALTH_CHECK_INTERVAL` | Health check interval (seconds) | `30` |

---

## Security Considerations

### Production Security Requirements

#### Password Requirements

| Environment | Minimum Length | Complexity |
|-------------|----------------|------------|
| Development | 8 characters | None |
| Staging | 16 characters | 2 of 4 types |
| Production | 32 characters | 3 of 4 types |

**Password Types**: Uppercase, lowercase, digits, special characters

#### Encryption Keys

All encryption keys must be:
- Generated with `openssl rand -base64 32`
- Stored in a Hardware Security Module (HSM) or Cloud KMS
- Rotated annually
- Never committed to version control

#### Secret Storage

| Secret Type | Storage Location | Access Method |
|-------------|------------------|---------------|
| Database passwords | `.env` file | Environment variable |
| Encryption keys | `.env` file | Environment variable |
| Service tokens | Token file (600 perms) | File path |
| Provider API keys | Infisical | API fetch |

### Access Control

```
File Permissions:
- .env                    : 600 (owner read/write only)
- .env.export             : 600 (owner read/write only)
- infisical-token         : 600 (owner read/write only)
- env-config.sh           : 755 (executable, readable)
- openbao/config.hcl      : 640 (readable by openbao group)
```

### Network Security

| Service | Internal Port | External Access | Notes |
|---------|---------------|-----------------|-------|
| PostgreSQL | 5432 | No | Internal only |
| Infisical API | 8080 | Via proxy | Authenticate all requests |
| OpenBao API | 8200 | Via proxy | TLS required in prod |
| RAD Gateway | 8090 | Yes | Behind load balancer |

---

## Environment-Specific Guidelines

### Development (Local)

**File**: `deploy/golden-stack/.env`

Characteristics:
- Weak, predictable passwords acceptable
- SSL verification disabled
- Debug logging enabled
- All services on localhost
- No backups by default

**Quick Start**:
```bash
source deploy/golden-stack/env-config.sh --generate
```

### Alpha (Single Node)

**File**: `/opt/radgateway01/config/env`

Characteristics:
- Moderate password strength
- Self-signed certificates acceptable
- Basic monitoring
- Manual backup process

### Staging

Characteristics:
- Strong passwords (16+ characters)
- Valid SSL certificates
n- Automated backups
- Full monitoring stack
- CI/CD deployment

### Production

Characteristics:
- Maximum password strength (32+ characters)
- CA-signed SSL certificates
- Multi-region backups
- HSM for key storage
- Separate unseal key holders
- Audit logging enabled
- Network segmentation

**Production Checklist**:
- [ ] All passwords 32+ characters
- [ ] Encryption keys from HSM
- [ ] SSL certificates valid
- [ ] OpenBao auto-unseal configured
- [ ] Backup S3 bucket configured
- [ ] Monitoring alerts configured
- [ ] Runbook reviewed

---

## Troubleshooting

### Common Issues

#### Issue: "Missing required environment variables"

**Cause**: Required variables not set
**Solution**:
```bash
# Check which variables are missing
source ./env-config.sh --check

# Set missing variables in .env
export POSTGRES_USER=secretstack
export POSTGRES_PASSWORD=$(openssl rand -base64 32)
```

#### Issue: "PostgreSQL does not appear to be running"

**Cause**: Database not started or wrong connection settings
**Solution**:
```bash
# Start PostgreSQL
sudo systemctl start postgresql

# Or start container
podman run -d --name postgres \
  -e POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
  -p 5432:5432 \
  postgres:16

# Verify connectivity
pg_isready -h localhost -p 5432
```

#### Issue: "Infisical not accessible"

**Cause**: Infisical service not running
**Solution**:
```bash
# Check service status
curl http://localhost:8080/api/status

# Verify token format
echo "$INFISICAL_SERVICE_TOKEN" | grep -E '^st\.[a-f0-9]+\.[a-f0-9]+\.[a-f0-9]+$'
```

#### Issue: "Cannot construct database URLs"

**Cause**: POSTGRES_PASSWORD not set
**Solution**:
```bash
# Set password
export POSTGRES_PASSWORD=$(openssl rand -base64 32)

# Re-run configuration
source ./env-config.sh
```

### Validation Commands

```bash
# Check all required variables
source ./env-config.sh --check

# Test database connectivity
pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT"

# Test Infisical
curl -H "Authorization: Bearer ${INFISICAL_SERVICE_TOKEN%.*}" \
  "${INFISICAL_API_URL}/api/v2/service-token"

# Test OpenBao
curl "${OPENBAO_API_ADDR}/v1/sys/health"

# Test RAD Gateway
curl http://localhost:8090/health
```

---

## Reference Tables

### Variable Summary Table

| Variable | Required | Default | Production |
|----------|----------|---------|------------|
| POSTGRES_USER | Yes | secretstack | secretstack |
| POSTGRES_PASSWORD | Yes | - | 32+ chars |
| POSTGRES_DB | Yes | secrets | secrets |
| POSTGRES_HOST | Yes | localhost | pg.internal |
| POSTGRES_PORT | Yes | 5432 | 5432 |
| INFISICAL_DB_URL | Yes | - | Constructed |
| INFISICAL_ENCRYPTION_KEY | Yes | - | HSM stored |
| INFISICAL_API_URL | No | localhost:8080 | infisical.prod |
| INFISICAL_SERVICE_TOKEN | No | - | Token file |
| OPENBAO_DB_URL | Yes | - | Constructed |
| OPENBAO_API_ADDR | No | :8200 | :8200 |
| RAD_LISTEN_ADDR | No | :8090 | :8090 |
| RAD_API_KEYS | Yes | - | 32+ chars |
| RAD_ENVIRONMENT | No | alpha | prod |

### File Reference

| File | Purpose | Permissions |
|------|---------|-------------|
| `deploy/golden-stack/.env.example` | Template with documentation | 644 |
| `deploy/golden-stack/.env` | Development defaults | 600 (git-ignored) |
| `deploy/golden-stack/env-config.sh` | Configuration helper | 755 |
| `deploy/golden-stack/.env.export` | Systemd export | 600 |
| `/opt/radgateway01/config/infisical-token` | Service token | 600 |

### Related Documentation

- `/mnt/ollama/git/RADAPI01/docs/operations/golden-stack-deployment.md` - Deployment guide
- `/mnt/ollama/git/RADAPI01/docs/operations/deployment-radgateway01.md` - RAD Gateway deployment
- `/mnt/ollama/git/RADAPI01/docs/operations/common-issues-runbook.md` - Troubleshooting
- `/mnt/ollama/git/RADAPI01/.env.example` - Root-level env template
- `/mnt/ollama/git/RADAPI01/deploy/config/env` - Static env template

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-02-16 | Team Hotel | Initial release |

---

**Security Notice**: This document may reference example secrets for illustration. Never use example values in production. Always generate cryptographically secure random values.
