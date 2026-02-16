#!/bin/bash
set -euo pipefail

# RAD Gateway 01 Deployment Installation Script
# Installs all deployment artifacts to /opt/radgateway01

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="/opt/radgateway01"
SERVICE_USER="radgateway"
SERVICE_GROUP="radgateway"

log() {
    echo "[$(date -Iseconds)] [install] $*"
}

error() {
    echo "[$(date -Iseconds)] [install] ERROR: $*" >&2
    exit 1
}

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    error "This script must be run as root (use sudo)"
fi

# Check prerequisites
log "Checking prerequisites..."

if ! command -v podman &> /dev/null; then
    error "Podman is not installed"
fi

if ! command -v systemctl &> /dev/null; then
    error "systemd is not available"
fi

# Check if Infisical is running
if ! curl -sf http://localhost:8080/api/status &> /dev/null; then
    log "WARNING: Infisical does not appear to be running on localhost:8080"
    log "The service may fail to start until Infisical is available"
fi

# Create user and group
log "Creating service user and group..."
if ! getent group "$SERVICE_GROUP" &> /dev/null; then
    groupadd --system "$SERVICE_GROUP"
fi

if ! getent passwd "$SERVICE_USER" &> /dev/null; then
    useradd --system \
        --gid "$SERVICE_GROUP" \
        --home-dir "$INSTALL_DIR" \
        --shell /usr/sbin/nologin \
        --comment "RAD Gateway 01 Service" \
        "$SERVICE_USER"
fi

# Create directory structure
log "Creating directory structure..."
mkdir -p "$INSTALL_DIR"/{bin,config,data,logs,systemd}

# Copy files
log "Copying deployment files..."
cp -v "$SCRIPT_DIR/bin/"*.sh "$INSTALL_DIR/bin/"
cp -v "$SCRIPT_DIR/config/env" "$INSTALL_DIR/config/"
cp -v "$SCRIPT_DIR/systemd/radgateway01.service" "$INSTALL_DIR/systemd/"

# Set permissions
log "Setting permissions..."
chown -R root:root "$INSTALL_DIR/bin"
chown -R "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR"/{data,logs}
chown root:root "$INSTALL_DIR/config"

chmod -R 755 "$INSTALL_DIR/bin"
chmod 644 "$INSTALL_DIR/config/env"
chmod 750 "$INSTALL_DIR/data"
chmod 755 "$INSTALL_DIR/logs"

# Make scripts executable
chmod +x "$INSTALL_DIR/bin/"*.sh

# Create infisical token file placeholder if it doesn't exist
if [[ ! -f "$INSTALL_DIR/config/infisical-token" ]]; then
    log "Creating placeholder for Infisical token..."
    touch "$INSTALL_DIR/config/infisical-token"
    chmod 600 "$INSTALL_DIR/config/infisical-token"
    chown "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR/config/infisical-token"
    log "NOTE: You must populate $INSTALL_DIR/config/infisical-token with your Infisical service token"
fi

# Create Podman volume if it doesn't exist
if ! podman volume exists radgateway01-data 2>/dev/null; then
    log "Creating Podman volume radgateway01-data..."
    podman volume create radgateway01-data
fi

# Install systemd service
log "Installing systemd service..."
cp "$INSTALL_DIR/systemd/radgateway01.service" /etc/systemd/system/
chmod 644 /etc/systemd/system/radgateway01.service

# Reload systemd
log "Reloading systemd daemon..."
systemctl daemon-reload

# Configure firewall
log "Configuring firewall..."
if command -v firewall-cmd &> /dev/null; then
    firewall-cmd --permanent --add-port=8090/tcp 2>/dev/null || true
    firewall-cmd --reload 2>/dev/null || true
    log "Firewall configured for port 8090"
else
    log "WARNING: firewall-cmd not found, firewall rules not configured"
fi

# CRITICAL: Add iptables FORWARD rules for Podman container networking
# Without these rules, external traffic cannot reach the container
# See: docs/operations/network-troubleshooting.md
log "Configuring iptables FORWARD rules for container networking..."
CONTAINER_NETWORK="10.88.0.0/16"
APP_PORT="8090"

# Check if rules already exist
if ! iptables -C FORWARD -d "$CONTAINER_NETWORK" -p tcp --dport "$APP_PORT" -j ACCEPT 2>/dev/null; then
    log "Adding FORWARD rule for incoming traffic..."
    iptables -I FORWARD -d "$CONTAINER_NETWORK" -p tcp --dport "$APP_PORT" -j ACCEPT
fi

if ! iptables -C FORWARD -s "$CONTAINER_NETWORK" -p tcp --sport "$APP_PORT" -m state --state ESTABLISHED,RELATED -j ACCEPT 2>/dev/null; then
    log "Adding FORWARD rule for return traffic..."
    iptables -I FORWARD -s "$CONTAINER_NETWORK" -p tcp --sport "$APP_PORT" -m state --state ESTABLISHED,RELATED -j ACCEPT
fi

log "iptables FORWARD rules configured"

# Save iptables rules for persistence (RHEL/CentOS/Fedora)
if command -v iptables-save &> /dev/null && [[ -d /etc/sysconfig ]]; then
    log "Saving iptables rules..."
    iptables-save > /etc/sysconfig/iptables
fi

# Enable service (but don't start yet - need token)
log "Enabling radgateway01 service..."
systemctl enable radgateway01.service

# Summary
cat << EOF

========================================
RAD Gateway 01 Installation Complete
========================================

Installation directory: $INSTALL_DIR
Service user: $SERVICE_USER
Service group: $SERVICE_GROUP

Next steps:
1. Add your Infisical service token to:
   $INSTALL_DIR/config/infisical-token

   The token file should contain only the token with no newlines.
   Permissions should be: -rw------- (600)

2. Verify the token permissions:
   ls -la $INSTALL_DIR/config/infisical-token

3. Start the service:
   sudo systemctl start radgateway01

4. Check service status:
   sudo systemctl status radgateway01

5. Verify health endpoint:
   curl http://localhost:8090/health

For troubleshooting, see: /mnt/ollama/git/RADAPI01/RUNBOOK.md

========================================
EOF

log "Installation completed successfully"
