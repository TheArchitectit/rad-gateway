#!/bin/bash
#
# OpenBao Initialization Script for Golden Stack Cold Vault
# Location: /openbao/scripts/init.sh
# Purpose: Initialize OpenBao with PostgreSQL backend and cold vault configuration
#

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/openbao/logs/init.log"
INIT_STATUS_FILE="/openbao/data/.initialized"
SEAL_STATUS_FILE="/openbao/data/.seal-status"
VAULT_ADDR="${BAO_ADDR:-http://127.0.0.1:8200}"

# PostgreSQL configuration
PG_HOST="${BAO_PG_HOST:-localhost}"
PG_PORT="${BAO_PG_PORT:-5432}"
PG_DATABASE="${BAO_PG_DATABASE:-openbao}"
PG_USER="${BAO_PG_USER:-openbao}"
PG_PASSWORD_FILE="${BAO_PG_PASSWORD_FILE:-/openbao/config/pg-password}"
PG_SSLMODE="${BAO_PG_SSLMODE:-prefer}"

# Cold vault settings
COLD_VAULT_RETENTION_DAYS="${BAO_COLD_VAULT_RETENTION_DAYS:-3650}"
COLD_VAULT_MAX_VERSIONS="${BAO_COLD_VAULT_MAX_VERSIONS:-100}"

# Ensure log directory exists
mkdir -p "$(dirname "$LOG_FILE")"
mkdir -p /openbao/data

# Logging functions
log() {
    echo "[$(date -Iseconds)] [openbao-init] $*" | tee -a "$LOG_FILE"
}

log_info() {
    echo "[$(date -Iseconds)] [openbao-init] [INFO] $*" | tee -a "$LOG_FILE"
}

log_warn() {
    echo "[$(date -Iseconds)] [openbao-init] [WARN] $*" | tee -a "$LOG_FILE"
}

log_error() {
    echo "[$(date -Iseconds)] [openbao-init] [ERROR] $*" >&2 | tee -a "$LOG_FILE"
}

# Error handler
error_exit() {
    log_error "$1"
    exit "${2:-1}"
}

# Cleanup function
cleanup() {
    local exit_code=$?
    if [[ $exit_code -ne 0 ]]; then
        log_error "Initialization failed with exit code $exit_code"
    fi
    exit $exit_code
}

trap cleanup EXIT

# Read PostgreSQL password from file
get_pg_password() {
    if [[ -f "$PG_PASSWORD_FILE" ]]; then
        cat "$PG_PASSWORD_FILE" | tr -d '[:space:]'
    else
        log_warn "PostgreSQL password file not found at $PG_PASSWORD_FILE"
        echo ""
    fi
}

# Build PostgreSQL connection URL
build_connection_url() {
    local password
    password=$(get_pg_password)
    if [[ -n "$password" ]]; then
        echo "postgres://${PG_USER}:${password}@${PG_HOST}:${PG_PORT}/${PG_DATABASE}?sslmode=${PG_SSLMODE}"
    else
        echo "postgres://${PG_USER}@${PG_HOST}:${PG_PORT}/${PG_DATABASE}?sslmode=${PG_SSLMODE}"
    fi
}

# Wait for PostgreSQL to be available
wait_for_postgresql() {
    log_info "Waiting for PostgreSQL at ${PG_HOST}:${PG_PORT}..."

    local max_attempts=30
    local attempt=1
    local password
    password=$(get_pg_password)

    export PGPASSWORD="$password"

    while [[ $attempt -le $max_attempts ]]; do
        if pg_isready -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DATABASE" >/dev/null 2>&1; then
            log_info "PostgreSQL is ready"
            unset PGPASSWORD
            return 0
        fi

        log_info "PostgreSQL not ready, attempt $attempt/$max_attempts..."
        sleep 2
        ((attempt++))
    done

    unset PGPASSWORD
    error_exit "PostgreSQL failed to become ready after $max_attempts attempts"
}

# Check if OpenBao is already initialized
is_initialized() {
    if [[ -f "$INIT_STATUS_FILE" ]]; then
        return 0
    fi

    # Also check via API
    local init_status
    init_status=$(curl -sf "${VAULT_ADDR}/v1/sys/init" 2>/dev/null | jq -r '.initialized // false') || init_status="false"

    if [[ "$init_status" == "true" ]]; then
        return 0
    fi

    return 1
}

# Check if OpenBao is sealed
is_sealed() {
    local seal_status
    seal_status=$(curl -sf "${VAULT_ADDR}/v1/sys/seal-status" 2>/dev/null | jq -r '.sealed // true') || seal_status="true"

    if [[ "$seal_status" == "true" ]]; then
        return 0
    fi

    return 1
}

# Initialize OpenBao
initialize_vault() {
    log_info "Initializing OpenBao..."

    # Generate initialization payload
    local init_response
    init_response=$(curl -sf -X PUT \
        -H "Content-Type: application/json" \
        -d '{
            "secret_shares": 5,
            "secret_threshold": 3,
            "pgp_keys": null,
            "root_token_pgp_key": null
        }' \
        "${VAULT_ADDR}/v1/sys/init" 2>/dev/null) || {
        error_exit "Failed to initialize OpenBao"
    }

    # Extract keys and token
    local root_token
    local unseal_keys

    root_token=$(echo "$init_response" | jq -r '.root_token')
    unseal_keys=$(echo "$init_response" | jq -r '.keys_base64')

    if [[ -z "$root_token" || "$root_token" == "null" ]]; then
        error_exit "Failed to extract root token from initialization response"
    fi

    # Save initialization data securely
    log_info "Saving initialization data..."

    # Save to file with restricted permissions (only for cold vault initialization)
    cat > "$INIT_STATUS_FILE" << EOF
# OpenBao Initialization Data
# WARNING: This file contains sensitive data. Secure it immediately!
# Generated: $(date -Iseconds)

ROOT_TOKEN=${root_token}
EOF

    # Save unseal keys
    echo "$unseal_keys" | jq -r '.[]' > "/openbao/data/.unseal-keys"

    # Set restrictive permissions
    chmod 600 "$INIT_STATUS_FILE"
    chmod 600 "/openbao/data/.unseal-keys"

    log_info "OpenBao initialized successfully"
    log_warn "IMPORTANT: Secure the root token and unseal keys immediately!"
    log_warn "Root token saved to: $INIT_STATUS_FILE"
    log_warn "Unseal keys saved to: /openbao/data/.unseal-keys"

    # Output for container logs (in production, these should be captured securely)
    echo ""
    echo "========================================"
    echo "OPENBAO INITIALIZATION COMPLETE"
    echo "========================================"
    echo "Root Token: ${root_token:0:20}..."
    echo "Unseal Keys: 5 keys generated (threshold: 3)"
    echo "========================================"
    echo ""
}

# Unseal OpenBao
unseal_vault() {
    if ! is_sealed; then
        log_info "OpenBao is already unsealed"
        return 0
    fi

    log_info "Unsealing OpenBao..."

    if [[ ! -f "/openbao/data/.unseal-keys" ]]; then
        error_exit "Unseal keys not found. Cannot unseal OpenBao."
    fi

    local unseal_count=0
    while IFS= read -r key && [[ $unseal_count -lt 3 ]]; do
        local response
        response=$(curl -sf -X PUT \
            -H "Content-Type: application/json" \
            -d "{\"key\": \"${key}\"}" \
            "${VAULT_ADDR}/v1/sys/unseal" 2>/dev/null) || {
            log_error "Failed to submit unseal key"
            continue
        }

        ((unseal_count++))
        log_info "Unseal key $unseal_count submitted"

        # Check if unsealed
        if echo "$response" | jq -e '.sealed == false' >/dev/null 2>&1; then
            log_info "OpenBao is now unsealed"
            return 0
        fi
    done < "/openbao/data/.unseal-keys"

    if is_sealed; then
        error_exit "Failed to unseal OpenBao after submitting $unseal_count keys"
    fi
}

# Create cold vault mount
create_cold_vault_mount() {
    log_info "Creating cold vault mount..."

    local root_token
    root_token=$(grep "ROOT_TOKEN" "$INIT_STATUS_FILE" | cut -d'=' -f2)

    # Enable KV v2 secrets engine at cold-vault path
    local response
    response=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -H "X-Vault-Token: ${root_token}" \
        -d '{
            "type": "kv-v2",
            "options": {
                "version": "2"
            },
            "config": {
                "max_versions": '"${COLD_VAULT_MAX_VERSIONS}"',
                "cas_required": false
            }
        }' \
        "${VAULT_ADDR}/v1/sys/mounts/cold-vault" 2>/dev/null) || {
        log_warn "Failed to create cold-vault mount (may already exist)"
        return 0
    }

    log_info "Cold vault mount created successfully"
}

# Configure retention policies
configure_retention_policies() {
    log_info "Configuring retention policies..."

    local root_token
    root_token=$(grep "ROOT_TOKEN" "$INIT_STATUS_FILE" | cut -d'=' -f2)

    # Configure cold-vault retention
    curl -sf -X POST \
        -H "Content-Type: application/json" \
        -H "X-Vault-Token: ${root_token}" \
        -d '{
            "max_versions": '"${COLD_VAULT_MAX_VERSIONS}"',
            "delete_version_after": "'"${COLD_VAULT_RETENTION_DAYS}"'d"
        }' \
        "${VAULT_ADDR}/v1/cold-vault/config" 2>/dev/null || {
        log_warn "Failed to configure retention policies (may not be supported)"
    }

    # Configure audit logging
    curl -sf -X PUT \
        -H "Content-Type: application/json" \
        -H "X-Vault-Token: ${root_token}" \
        -d '{
            "type": "file",
            "options": {
                "file_path": "/openbao/logs/audit.log"
            }
        }' \
        "${VAULT_ADDR}/v1/sys/audit/cold-vault-audit" 2>/dev/null || {
        log_warn "Failed to configure audit logging (may already exist)"
    }

    log_info "Retention policies configured"
}

# Start OpenBao server
start_openbao() {
    log_info "Starting OpenBao server..."

    # Build connection URL and export for config
    export BAO_PG_CONNECTION_URL
    BAO_PG_CONNECTION_URL=$(build_connection_url)
    export BAO_API_ADDR="${VAULT_ADDR}"
    export BAO_CLUSTER_ADDR="http://127.0.0.1:8201"

    log_info "Using PostgreSQL at ${PG_HOST}:${PG_PORT}/${PG_DATABASE}"

    # Start OpenBao in background
    bao server -config=/openbao/config/config.hcl &
    local bao_pid=$!

    log_info "OpenBao server started with PID $bao_pid"

    # Wait for OpenBao to be ready
    local max_attempts=30
    local attempt=1

    while [[ $attempt -le $max_attempts ]]; do
        if curl -sf "${VAULT_ADDR}/v1/sys/health" >/dev/null 2>&1; then
            log_info "OpenBao API is ready"
            return 0
        fi

        # Check if process is still running
        if ! kill -0 $bao_pid 2>/dev/null; then
            error_exit "OpenBao server process died"
        fi

        log_info "Waiting for OpenBao API, attempt $attempt/$max_attempts..."
        sleep 2
        ((attempt++))
    done

    error_exit "OpenBao API failed to become ready after $max_attempts attempts"
}

# Main initialization flow
main() {
    log_info "=== OpenBao Cold Vault Initialization ==="
    log_info "Timestamp: $(date -Iseconds)"
    log_info "Vault Address: ${VAULT_ADDR}"
    log_info "PostgreSQL Host: ${PG_HOST}:${PG_PORT}"

    # Step 1: Wait for PostgreSQL
    wait_for_postgresql

    # Step 2: Start OpenBao server
    start_openbao

    # Step 3: Initialize if needed
    if is_initialized; then
        log_info "OpenBao is already initialized"
    else
        initialize_vault
    fi

    # Step 4: Unseal the vault
    unseal_vault

    # Step 5: Create cold vault mount
    create_cold_vault_mount

    # Step 6: Configure retention policies
    configure_retention_policies

    log_info "=== OpenBao Initialization Complete ==="

    # Keep the script running to maintain the container
    log_info "OpenBao is running. Press Ctrl+C to stop."
    wait
}

# Handle signals for graceful shutdown
shutdown() {
    log_info "Received shutdown signal, stopping OpenBao..."

    # Try graceful shutdown via API
    local root_token
    if [[ -f "$INIT_STATUS_FILE" ]]; then
        root_token=$(grep "ROOT_TOKEN" "$INIT_STATUS_FILE" | cut -d'=' -f2)
        curl -sf -X POST \
            -H "X-Vault-Token: ${root_token}" \
            "${VAULT_ADDR}/v1/sys/step-down" 2>/dev/null || true
    fi

    # Kill any remaining processes
    pkill -f "bao server" 2>/dev/null || true

    exit 0
}

trap shutdown SIGTERM SIGINT

# Run main function
main "$@"
