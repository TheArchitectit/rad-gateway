#!/bin/bash
# Deploy RAD Gateway 01 to remote host
# Builds container on target host and deploys

set -euo pipefail

TARGET_HOST="${1:-172.16.30.45}"
TARGET_USER="${2:-user001}"
REPO_PATH="/tmp/rad-gateway-deploy"
CONTAINER_NAME="radgateway01"

log() {
    echo "[$(date -Iseconds)] [deploy] $*"
}

error() {
    echo "[$(date -Iseconds)] [deploy] ERROR: $*" >&2
    exit 1
}

log "Deploying to $TARGET_HOST as $TARGET_USER"

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Step 1: Create temporary archive of repo (excluding large files)
log "Creating repository archive..."
ARCHIVE=$(mktemp /tmp/rad-gateway.XXXXXX.tar.gz)

# Create archive excluding node_modules and other large files
tar -czf "$ARCHIVE" \
    --exclude='web/node_modules' \
    --exclude='web/dist' \
    --exclude='.git' \
    --exclude='*.tar.gz' \
    -C "$REPO_ROOT" \
    cmd deploy go.mod go.sum internal migrations

# Step 2: Copy archive to target host
log "Copying archive to $TARGET_HOST..."
scp "$ARCHIVE" "$TARGET_USER@$TARGET_HOST:$REPO_PATH.tar.gz"

# Step 3: Extract and build on target
log "Building container on target host..."

ssh "$TARGET_USER@$TARGET_HOST" bash -s << 'ENDSSH'
set -euo pipefail

REPO_PATH="/tmp/rad-gateway-deploy"
CONTAINER_NAME="radgateway01"

log() {
    echo "[$(date -Iseconds)] $*"
}

# Clean up any previous deployment
rm -rf "$REPO_PATH"
mkdir -p "$REPO_PATH"
tar -xzf "$REPO_PATH.tar.gz" -C "$REPO_PATH"

cd "$REPO_PATH"

# Build container using podman
log "Building container image..."
podman build -t "localhost/$CONTAINER_NAME:latest" \
    -f "$REPO_PATH/deploy/radgateway01/Containerfile" \
    "$REPO_PATH"

# Stop existing container if running
if podman ps --format '{{.Names}}' | grep -q "$CONTAINER_NAME-app"; then
    log "Stopping existing container..."
    podman stop "$CONTAINER_NAME-app" || true
    podman rm "$CONTAINER_NAME-app" || true
fi

# Create pod if it doesn't exist
if ! podman pod exists "$CONTAINER_NAME"; then
    log "Creating pod..."
    podman pod create -n "$CONTAINER_NAME" -p 8090:8090
fi

# Run the container
log "Starting container..."
podman run -d \
    --pod "$CONTAINER_NAME" \
    --name "$CONTAINER_NAME-app" \
    --restart=always \
    -v "$REPO_PATH/migrations:/migrations:ro" \
    -v "/opt/$CONTAINER_NAME/data:/data" \
    --env-file "/opt/$CONTAINER_NAME/config/env" \
    "localhost/$CONTAINER_NAME:latest"

# Clean up archive
rm -f "$REPO_PATH.tar.gz"

log "Container deployed successfully"
podman ps --pod | grep "$CONTAINER_NAME"
ENDSSH

# Step 4: Clean up local archive
rm -f "$ARCHIVE"

log "Deployment complete"
log "Verify with: ssh $TARGET_USER@$TARGET_HOST 'sudo podman ps --pod'"
log "Health check: curl http://$TARGET_HOST:8090/health"
