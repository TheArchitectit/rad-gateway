#!/bin/bash
# Install k3s on target host for A2A Gateway deployment

set -euo pipefail

TARGET_HOST="${1:-172.16.30.45}"
TARGET_USER="${2:-user001}"

log() {
    echo "[$(date -Iseconds)] [k3s-install] $*"
}

log "Installing k3s on $TARGET_HOST..."

# Install k3s
ssh "$TARGET_USER@$TARGET_HOST" << 'EOF'
set -euo pipefail

log() {
    echo "[$(date -Iseconds)] $*"
}

# Check if k3s already installed
if command -v k3s &>/dev/null; then
    log "k3s already installed, skipping"
    k3s --version
else
    log "Installing k3s..."
    curl -sfL https://get.k3s.io | sh -s - --write-kubeconfig-mode 644

    # Set up kubectl alias
    echo 'export KUBECONFIG=/etc/rancher/k3s/k3s.yaml' >> ~/.bashrc
    echo 'alias k=kubectl' >> ~/.bashrc

    log "k3s installed successfully"
fi

# Verify k3s is running
sudo systemctl status k3s --no-pager | head -10

# Test kubectl
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
kubectl cluster-info
kubectl get nodes

log "k3s installation complete"
EOF

log "k3s installation script completed"
