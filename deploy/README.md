# RAD Gateway 01 Deployment Automation

This directory contains all deployment automation scripts for the RAD Gateway 01 (radgateway01) service.

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
| `systemd/radgateway01.service` | Systemd unit file for service management |

## Quick Start

```bash
# Run as root or with sudo
sudo ./install.sh

# Add your Infisical token
echo "your-infisical-token" | sudo tee /opt/radgateway01/config/infisical-token
sudo chmod 600 /opt/radgateway01/config/infisical-token
sudo chown radgateway:radgateway /opt/radgateway01/config/infisical-token

# Start the service
sudo systemctl start radgateway01

# Verify
sudo systemctl status radgateway01
curl http://localhost:8090/health
```

## Directory Structure After Install

```
/opt/radgateway01/
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
    └── radgateway01.service  # Systemd unit file
```

## Management Commands

```bash
# Service control
sudo systemctl start radgateway01
sudo systemctl stop radgateway01
sudo systemctl restart radgateway01
sudo systemctl status radgateway01

# Logs
sudo journalctl -u radgateway01 -f
sudo podman logs radgateway01-app

# Health check
/opt/radgateway01/bin/health-check.sh

# Manual backup
/opt/radgateway01/bin/backup.sh

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
