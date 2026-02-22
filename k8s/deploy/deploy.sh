#!/bin/bash
# RAD Gateway A2A - Deployment Script
# Applies all Kubernetes manifests in the correct order

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="${NAMESPACE:-radgateway}"

echo "=== RAD Gateway A2A Deployment ==="
echo "Namespace: $NAMESPACE"
echo ""

# Step 1: Create namespace
echo "[1/8] Creating namespace..."
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

# Step 2: Apply SPIRE resources (if not already installed)
echo "[2/8] Checking SPIRE installation..."
if ! kubectl get namespace spire &>/dev/null; then
    echo "  SPIRE namespace not found, skipping..."
else
    echo "  SPIRE already installed"
fi

# Step 3: Apply Kafka resources
echo "[3/8] Applying Kafka resources..."
if kubectl get kafka a2a-kafka -n kafka &>/dev/null; then
    echo "  Kafka already exists"
else
    kubectl apply -f ../kafka/ -n kafka
    echo "  Waiting for Kafka to be ready..."
    kubectl wait kafka a2a-kafka -n kafka --for=condition=Ready --timeout=300s || true
fi

# Step 4: Apply OpenTelemetry Collector
echo "[4/8] Applying OpenTelemetry Collector..."
kubectl apply -f ../otel/ -n observability || true

# Step 5: Apply Gateway resources
echo "[5/8] Applying Gateway API resources..."
kubectl apply -f ../gateway/ || true

# Step 6: Apply backend deployment
echo "[6/8] Applying RAD Gateway backend..."
kubectl apply -f radgateway-backend.yaml

# Step 7: Wait for deployment
echo "[7/8] Waiting for deployment to be ready..."
kubectl wait deployment radgateway-backend -n $NAMESPACE --for=condition=Available --timeout=120s || true

# Step 8: Show status
echo "[8/8] Deployment status:"
echo ""
kubectl get pods -n $NAMESPACE -l app=radgateway-backend
kubectl get svc -n $NAMESPACE -l app=radgateway-backend

echo ""
echo "=== Deployment Complete ==="
echo ""
echo "Access the gateway:"
echo "  kubectl port-forward svc/radgateway-backend -n $NAMESPACE 8090:8090"
echo ""
echo "View logs:"
echo "  kubectl logs -n $NAMESPACE -l app=radgateway-backend -f"
