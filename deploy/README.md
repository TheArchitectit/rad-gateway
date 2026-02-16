# RAD Gateway Deployment Automation

This directory contains deployment automation scripts for the RAD Gateway service.

## Overview

This deployment includes two main components:

1. **RAD Gateway Application** - The API gateway service (port 8090)
2. **Golden Stack** - Secrets management infrastructure (PostgreSQL + Infisical + OpenBao)

---

## Quick Start

### Full Stack Deployment (Recommended)

Deploy the complete stack including secrets management:

```bash
# 1. Deploy Golden Stack first (secrets infrastructure)
cd /mnt/ollama/git/RADAPI01/deploy
sudo ./start-secret-stack.sh

# 2. Deploy RAD Gateway
sudo ./install.sh

# 3. Configure Infisical token
echo "your-infisical-token" | sudo tee /opt/<your-container-name>/config/infisical-token
sudo chmod 600 /opt/<your-container-name>/config/infisical-token
sudo chown radgateway:radgateway /opt/<your-container-name>/config/infisical-token

# 4. Start services
sudo systemctl start secret-stack-infisical
sudo systemctl start secret-stack-openbao
sudo systemctl start <your-container-name>

# 5. Verify
curl http://localhost:8090/health
curl http://localhost:8080/api/status  # Infisical
curl http://localhost:8200/v1/sys/health  # OpenBao
```

### RAD Gateway Only

If Golden Stack is already deployed:

```bash
# Run as root or with sudo
sudo ./install.sh

# Add your Infisical token
echo "your-infisical-token" | sudo tee /opt/<your-container-name>/config/infisical-token
sudo chmod 600 /opt/<your-container-name>/config/infisical-token
sudo chown radgateway:radgateway /opt/<your-container-name>/config/infisical-token

# Start the service
sudo systemctl start <your-container-name>

# Verify
sudo systemctl status <your-container-name>
curl http://localhost:8090/health
```

---

## Files

### Scripts

| File | Description |
|------|-------------|
| `bin/startup.sh` | Infisical secret fetching + application startup |
| `bin/health-check.sh` | Health check for monitoring and load balancers |
| `bin/backup.sh` | Automated backup of configuration and data |
| `install.sh` | One-command installation of RAD Gateway |
| `uninstall.sh` | Clean removal of the service |
| `start-secret-stack.sh` | Deploy Golden Stack (PostgreSQL + Infisical + OpenBao) |

### Golden Stack Components

| Directory | Description |
|-----------|-------------|
| `openbao/` | OpenBao cold vault configuration and scripts |
| `postgres/` | PostgreSQL container configuration |

### Configuration

| File | Description |
|------|-------------|
| `config/env` | Static environment variables |
| `systemd/<your-container-name>.service` | Systemd unit file for service management |

---

## Golden Stack (Secrets Management)

The Golden Stack provides defense-in-depth secrets management with three integrated components:

### Components

| Component | Purpose | Port | Role |
|-----------|---------|------|------|
| **PostgreSQL 16** | Shared database backend | 5432 (internal) | Persistence layer |
| **Infisical** | Active secrets management | 8080 | Hot vault (operational) |
| **OpenBao** | Long-term secrets archive | 8200 | Cold vault (compliance) |

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Golden Stack                              │
│                                                              │
│   ┌─────────────────┐                                       │
│   │   PostgreSQL    │  Shared database for:                 │
│   │     :5432       │  - infisical_db                       │
│   │                 │  - openbao_db                         │
│   └────────┬────────┘                                       │
│            │                                                 │
│   ┌────────┴────────┐                                       │
│   │    Infisical    │  Hot Vault                            │
│   │     :8080       │  - Active secrets                     │
│   │                 │  - Service token auth                 │
│   │                 │  - Fast access (<10ms)                │
│   └────────┬────────┘                                       │
│            │                                                 │
│   ┌────────┴────────┐                                       │
│   │    OpenBao      │  Cold Vault                           │
│   │     :8200       │  - Long-term archive                  │
│   │                 │  - Compliance audit                   │
│   │                 │  - Immutable logs                     │
│   └─────────────────┘                                       │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Why This Stack

- **Infisical (Hot Vault)**: Used by applications for fast, frequent secret access
- **OpenBao (Cold Vault)**: Used for compliance, audit trails, and long-term retention
- **PostgreSQL**: Shared durable storage with isolated databases

### Deployment

```bash
# Deploy the Golden Stack
cd /mnt/ollama/git/RADAPI01/deploy
sudo ./start-secret-stack.sh

# Verify deployment
curl http://localhost:8080/api/status      # Infisical health
curl http://localhost:8200/v1/sys/health   # OpenBao health
sudo podman exec secret-stack-postgres pg_isready -U postgres
```

### Documentation

- [Golden Stack Overview](/mnt/ollama/git/RADAPI01/docs/operations/golden-stack.md)
- [Golden Stack Deployment](/mnt/ollama/git/RADAPI01/docs/operations/golden-stack-deployment.md)
- [Golden Stack Operations](/mnt/ollama/git/RADAPI01/docs/operations/golden-stack-operations.md)

---

## Directory Structure After Install

```
/opt/<your-container-name>/
├── bin/
│   ├── backup.sh          # Backup automation
│   ├── health-check.sh    # Health check script
│   └── startup.sh         # Infisical integration + startup
├── config/
│   ├── env                # Static environment config
│   └── infisical-token    # Infisical service token (600 permissions)
├── data/                  # Podman volume mount
├── logs/                  # Application logs
└── systemd/
    └── <your-container-name>.service  # Systemd unit file
```

## Management Commands

```bash
# Service control
sudo systemctl start <your-container-name>
sudo systemctl stop <your-container-name>
sudo systemctl restart <your-container-name>
sudo systemctl status <your-container-name>

# Logs
sudo journalctl -u <your-container-name> -f
sudo podman logs <your-container-name>-app

# Health check
/opt/<your-container-name>/bin/health-check.sh

# Manual backup
/opt/<your-container-name>/bin/backup.sh

# Container management
sudo podman ps --pod
sudo podman pod ps
```

## Uninstallation

```bash
# Remove service but keep data
sudo ./uninstall.sh

# Remove service and all data
sudo ./uninstall.sh --remove-data
```

## Security Notes

- Infisical token file has `600` permissions (owner read/write only)
- Service runs as non-root `radgateway` user
- Container has read-only root filesystem with write access only to `/data`
- Systemd security hardening enabled (ProtectSystem, ProtectHome, etc.)

## Troubleshooting

See the main [RUNBOOK.md](/mnt/ollama/git/RADAPI01/RUNBOOK.md) for detailed troubleshooting procedures.
