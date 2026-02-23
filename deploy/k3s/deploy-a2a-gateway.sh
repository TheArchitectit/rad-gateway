#!/bin/bash
# Deploy A2A Gateway 2026 to k3s cluster

set -euo pipefail

TARGET_HOST="${1:-172.16.30.45}"
TARGET_USER="${2:-user001}"
KUBECONFIG="/etc/rancher/k3s/k3s.yaml"

log() {
    echo "[$(date -Iseconds)] [deploy] $*"
}

log "Deploying A2A Gateway 2026 to $TARGET_HOST..."

# Copy manifests to target host
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

log "Copying K8s manifests to target..."
ssh "$TARGET_USER@$TARGET_HOST" "mkdir -p /tmp/k8s-manifests"
scp -r "$REPO_ROOT/k8s/" "$TARGET_USER@$TARGET_HOST:/tmp/k8s-manifests/"

# Deploy to k3s
ssh "$TARGET_USER@$TARGET_HOST" << 'EOF'
set -euo pipefail

KUBECONFIG="/etc/rancher/k3s/k3s.yaml"
export KUBECONFIG

log() {
    echo "[$(date -Iseconds)] $*"
}

MANIFESTS_DIR="/tmp/k8s-manifests/k8s"
SIMPLE_MANIFEST="$TARGET_USER@$TARGET_HOST:/tmp/a2a-gateway-simple.yaml"

# Step 1: Install Gateway API CRDs
log "Installing Gateway API CRDs..."
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.0/standard-install.yaml
sleep 5

# Step 2: Create namespaces
log "Creating namespaces..."
kubectl create namespace gateway-system --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace radgateway --dry-run=client -o yaml | kubectl apply -f -

# Step 3: Copy simplified manifest
log "Copying simplified deployment manifest..."
scp "$SCRIPT_DIR/a2a-gateway-simple.yaml" "$SIMPLE_MANIFEST"

# Step 4: Build and import container image into k3s
log "Building container image..."
cd "$REPO_ROOT"
tar -czf /tmp/radgateway-image.tar.gz \
    --exclude='web/node_modules' \
    --exclude='web/dist' \
    --exclude='.git' \
    -C "$REPO_ROOT" \
    cmd deploy go.mod go.sum internal migrations
scp /tmp/radgateway-image.tar.gz "$TARGET_USER@$TARGET_HOST:/tmp/"

ssh "$TARGET_USER@$TARGET_HOST" << 'BUILDIMAGE'
set -euo pipefail
log() { echo "[$(date -Iseconds)] $*" ; }

cd /tmp
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
sudo k3s ctr image import /tmp/radgateway-k3s.tar

log "Image import complete"
BUILDIMAGE

# Step 5: Apply simplified deployment
log "Applying simplified deployment..."
ssh "$TARGET_USER@$TARGET_HOST" "kubectl apply -f /tmp/a2a-gateway-simple.yaml"

# Step 6: Wait for deployment
log "Waiting for deployment..."
ssh "$TARGET_USER@$TARGET_HOST" "kubectl wait deployment radgateway-backend -n radgateway --for=condition=Available --timeout=300s" || true

# Step 7: Show status
log "Deployment status:"
ssh "$TARGET_USER@$TARGET_HOST" "kubectl get pods -n radgateway; kubectl get svc -n radgateway"

log "Deployment complete!"
log "Access the gateway at: http://$TARGET_HOST:30090"
log "Health check: curl http://$TARGET_HOST:30090/health"
EOF

log "Deployment script completed"
