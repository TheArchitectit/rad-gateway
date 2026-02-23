#!/bin/bash
# Deploy A2A Gateway 2026 to k3s cluster

set -euo pipefail

TARGET_HOST="${1:-172.16.30.45}"
TARGET_USER="${2:-user001}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

log() {
    echo "[$(date -Iseconds)] [deploy] $*"
}

log "Deploying A2A Gateway 2026 to $TARGET_HOST..."

# Step 1: Copy simplified manifest
log "Copying simplified deployment manifest..."
ssh "$TARGET_USER@$TARGET_HOST" "mkdir -p /tmp"
scp "$SCRIPT_DIR/a2a-gateway-simple.yaml" "$TARGET_USER@$TARGET_HOST:/tmp/a2a-gateway-simple.yaml"

# Step 2: Build and import container image into k3s
log "Building container image locally..."
cd "$REPO_ROOT"
tar -czf /tmp/radgateway-image.tar.gz \
    --exclude='web/node_modules' \
    --exclude='web/dist' \
    --exclude='.git' \
    -C "$REPO_ROOT" \
    cmd deploy go.mod go.sum internal migrations
scp /tmp/radgateway-image.tar.gz "$TARGET_USER@$TARGET_HOST:/tmp/"

# Step 3: Build image on target and import to k3s
log "Building image on target host..."
ssh "$TARGET_USER@$TARGET_HOST" bash -s << 'BUILDIMAGE'
set -euo pipefail
log() { echo "[$(date -Iseconds)] $*" ; }

cd /tmp
rm -rf radgateway-build
mkdir -p radgateway-build
tar -xzf radgateway-image.tar.gz -C radgateway-build
cd radgateway-build

log "Building image with podman..."
podman build -t radgateway-k3s:latest \
    -f deploy/radgateway01/Containerfile \
    .

log "Saving image for k3s import..."
podman save -o /tmp/radgateway-k3s.tar radgateway-k3s:latest

# Import into k3s
log "Importing image into k3s..."
sudo /usr/local/bin/k3s ctr image import /tmp/radgateway-k3s.tar

log "Image import complete"
BUILDIMAGE

# Step 4: Install Gateway API CRDs and apply deployment
log "Installing Gateway API CRDs..."
ssh "$TARGET_USER@$TARGET_HOST" << 'K8SDEPLOY'
set -euo pipefail
KUBECONFIG="/etc/rancher/k3s/k3s.yaml"
export KUBECONFIG

log() { echo "[$(date -Iseconds)] $*" ; }

# Install Gateway API CRDs
log "Installing Gateway API CRDs..."
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.0/standard-install.yaml
sleep 5

# Create namespaces
log "Creating namespaces..."
kubectl create namespace gateway-system --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace radgateway --dry-run=client -o yaml | kubectl apply -f -

# Apply simplified deployment
log "Applying simplified deployment..."
kubectl apply -f /tmp/a2a-gateway-simple.yaml

# Wait for deployment
log "Waiting for deployment..."
kubectl wait deployment radgateway-backend -n radgateway --for=condition=Available --timeout=300s || true

# Show status
log "Deployment status:"
kubectl get pods -n radgateway
kubectl get svc -n radgateway

log "Deployment complete!"
log "Access: http://NODE_IP:30090"
K8SDEPLOY

log "Deployment script completed"
log "Health check: curl http://$TARGET_HOST:30090/health"
