# RAD Gateway Deployment Automation

This directory contains deployment automation scripts for the RAD Gateway service.

## Files

### Scripts

| File | Description |
|------|-------------|
| `bin/startup.sh` | Infisical secret fetching + application startup |
| `bin/health-check.sh` | Health check for monitoring and load balancers |
| `bin/backup.sh` | Automated backup of configuration and data |
| `install.sh` | One-command installation of all deployment artifacts |
| `uninstall.sh` | Clean removal of the service |

### Configuration

| File | Description |
|------|-------------|
| `config/env` | Static environment variables |
| `systemd/<your-container-name>.service` | Systemd unit file for service management |

## Quick Start

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
