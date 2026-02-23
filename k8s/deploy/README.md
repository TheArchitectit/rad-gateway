# RAD Gateway A2A - Deployment Guide

## Prerequisites

- Kubernetes cluster 1.28+
- kubectl configured
- Helm 3.x
- Strimzi Operator (for Kafka)
- OpenTelemetry Operator
- SPIRE (optional, for workload identity)

## Quick Deploy

```bash
# Navigate to deployment directory
cd k8s/deploy

# Run the deployment script
./deploy.sh
```

## Manual Deployment

### Step 1: Create Namespace

```bash
kubectl create namespace radgateway
```

### Step 2: Deploy Kafka (if not installed)

```bash
kubectl apply -f ../kafka/
kubectl wait kafka a2a-kafka -n kafka --for=condition=Ready --timeout=300s
```

### Step 3: Deploy OpenTelemetry Collector

```bash
kubectl apply -f ../otel/
```

### Step 4: Deploy Gateway API Resources

```bash
kubectl apply -f ../gateway/
```

### Step 5: Deploy RAD Gateway Backend

```bash
kubectl apply -f radgateway-backend.yaml
```

### Step 6: Verify Deployment

```bash
# Check pods are running
kubectl get pods -n radgateway -l app=radgateway-backend

# Check services
kubectl get svc -n radgateway

# View logs
kubectl logs -n radgateway -l app=radgateway-backend -f
```

### Step 7: Access the Gateway

```bash
# Port forward for local access
kubectl port-forward svc/radgateway-backend -n radgateway 8090:8090

# Test health endpoint
curl http://localhost:8090/health

# Test A2A endpoint
curl -X POST http://localhost:8090/a2a/tasks \
  -H "Content-Type: application/agent-task+json" \
  -d '{"task_id":"test","message_object":{"role":"user","parts":[{"type":"text","text":"Hello"}]},"capabilities":["a2a"]}'
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| DATABASE_URL | PostgreSQL connection string | Required |
| REDIS_URL | Redis connection string | Optional |
| KAFKA_BROKERS | Kafka bootstrap servers | Required |
| SPIFFE_TRUST_DOMAIN | SPIFFE trust domain | internal.corp |
| LOG_LEVEL | Logging level | info |
| API_KEY | API key for authentication | Required |
| JWT_SECRET | Secret for JWT signing | Required |

### Resource Requirements

| Component | CPU Request | Memory Request | CPU Limit | Memory Limit |
|-----------|-------------|----------------|-----------|--------------|
| radgateway-backend | 250m | 256Mi | 1000m | 1Gi |

## Uninstall

```bash
kubectl delete -f radgateway-backend.yaml
kubectl delete -f ../gateway/
kubectl delete -f ../otel/
kubectl delete -f ../kafka/
kubectl delete namespace radgateway
```

## Troubleshooting

### Pod not starting

```bash
# Check pod events
kubectl describe pod -n radgateway -l app=radgateway-backend

# Check logs
kubectl logs -n radgateway -l app=radgateway-backend
```

### Database connection issues

```bash
# Verify database is accessible
kubectl exec -n radgateway deploy/radgateway-backend -- env | grep DATABASE
```

### Kafka connection issues

```bash
# Verify Kafka brokers are reachable
kubectl exec -n radgateway deploy/radgateway-backend -- nc -zv a2a-kafka-kafka-bootstrap.kafka.svc.cluster.local 9092
```
