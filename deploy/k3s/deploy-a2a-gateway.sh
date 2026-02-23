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
scp -r "$REPO_ROOT/k8s/" "$TARGET_USER@$TARGET_HOST:/tmp/k8s-manifests/"

# Deploy to k3s
ssh "$TARGET_USER@$TARGET_HOST" << 'EOF'
set -euo pipefail

KUBECONFIG="/etc/rancher/k3s/k3s.yaml"
export KUBECONFIG

log() {
    echo "[$(date -Iseconds)] $*"
}

MANIFESTS_DIR="/tmp/k8s-manifests"

# Step 1: Install Gateway API CRDs
log "Installing Gateway API CRDs..."
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.0/standard-install.yaml

# Wait for CRDs to be ready
sleep 5

# Step 2: Create namespaces
log "Creating namespaces..."
kubectl create namespace gateway-system --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace kafka --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace observability --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace radgateway --dry-run=client -o yaml | kubectl apply -f -

# Step 3: Install Gateway API resources
log "Applying Gateway API resources..."
kubectl apply -f "$MANIFESTS_DIR/gateway/" -n gateway-system

# Step 4: Install Kafka (Strimzi operator required)
log "Installing Kafka cluster..."
# Install Strimzi operator first
kubectl apply -f https://strimzi.io/install/latest?namespace=kafka -n kafka
# Wait for operator
kubectl wait --for=condition=Available deployment/strimzi-cluster-operator -n kafka --timeout=120s || true
# Apply Kafka cluster
kubectl apply -f "$MANIFESTS_DIR/kafka/" -n kafka

# Step 5: Install OpenTelemetry Collector
log "Installing OpenTelemetry Collector..."
kubectl apply -f "$MANIFESTS_DIR/otel/" -n observability

# Step 6: Deploy backend
log "Deploying RAD Gateway backend..."
kubectl apply -f "$MANIFESTS_DIR/deploy/" -n radgateway

# Step 7: Wait for deployment
log "Waiting for deployment..."
kubectl wait deployment radgateway-backend -n radgateway --for=condition=Available --timeout=300s || true

# Step 8: Show status
log "Deployment status:"
kubectl get pods -n radgateway
kubectl get pods -n gateway-system
kubectl get pods -n kafka
kubectl get svc -n radgateway

# Expose service (NodePort for k3s)
log "Creating NodePort service..."
kubectl patch svc radgateway-backend -n radgateway -p '{"spec":{"type":"NodePort","ports":[{"port":8090,"nodePort":30090}]}}' || true

log "Deployment complete!"
log "Access the gateway at: http://$TARGET_HOST:30090"
log "Health check: curl http://$TARGET_HOST:30090/health"
EOF

log "Deployment script completed"
