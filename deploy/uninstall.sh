#!/bin/bash
set -euo pipefail

# RAD Gateway 01 Uninstallation Script
# Removes the service and optionally data

INSTALL_DIR="/opt/radgateway01"
SERVICE_USER="radgateway"
SERVICE_GROUP="radgateway"
REMOVE_DATA="${REMOVE_DATA:-false}"

log() {
    echo "[$(date -Iseconds)] [uninstall] $*"
}

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    echo "ERROR: This script must be run as root (use sudo)" >&2
    exit 1
fi

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --remove-data)
            REMOVE_DATA=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [--remove-data]"
            echo ""
            echo "Options:"
            echo "  --remove-data  Also remove data directory and Podman volumes"
            echo "  --help, -h     Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

read -p "Are you sure you want to uninstall radgateway01? (yes/no): " confirm
if [[ "$confirm" != "yes" ]]; then
    log "Uninstall cancelled"
    exit 0
fi

# Stop and disable service
log "Stopping and disabling service..."
systemctl stop radgateway01 2>/dev/null || true
systemctl disable radgateway01 2>/dev/null || true

# Remove systemd service file
log "Removing systemd service..."
rm -f /etc/systemd/system/radgateway01.service
systemctl daemon-reload

# Remove container and pod
log "Removing containers and pod..."
podman stop radgateway01-app 2>/dev/null || true
podman rm radgateway01-app 2>/dev/null || true
podman pod rm radgateway01 2>/dev/null || true

# Remove firewall rules
log "Removing firewall rules..."
if command -v firewall-cmd &> /dev/null; then
    firewall-cmd --permanent --remove-port=8090/tcp 2>/dev/null || true
    firewall-cmd --reload 2>/dev/null || true
fi

# Remove data if requested
if [[ "$REMOVE_DATA" == "true" ]]; then
    log "Removing data directory..."
    rm -rf "$INSTALL_DIR"

    log "Removing Podman volumes..."
    podman volume rm radgateway01-data 2>/dev/null || true
else
    log "Keeping data directory at $INSTALL_DIR"
    log "Use --remove-data to also remove data"
fi

# Remove user and group (only if data is also removed)
if [[ "$REMOVE_DATA" == "true" ]]; then
    log "Removing service user..."
    userdel "$SERVICE_USER" 2>/dev/null || true
    groupdel "$SERVICE_GROUP" 2>/dev/null || true
fi

log "Uninstall completed"

if [[ "$REMOVE_DATA" != "true" ]]; then
    echo ""
    echo "Note: Data directory preserved at $INSTALL_DIR"
    echo "To remove data, run: $0 --remove-data"
fi
