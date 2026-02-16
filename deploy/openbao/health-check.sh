#!/bin/bash
#
# OpenBao Health Check Script for Golden Stack Cold Vault
# Location: /openbao/scripts/health-check.sh
# Purpose: Comprehensive health verification for container orchestration
#
# Exit codes:
#   0 - Healthy
#   1 - OpenBao API unhealthy
#   2 - PostgreSQL connectivity issue
#   3 - Vault is sealed
#   4 - Configuration error
#

set -euo pipefail

# Configuration
VAULT_ADDR="${BAO_ADDR:-http://127.0.0.1:8200}"
HEALTH_URL="${VAULT_ADDR}/v1/sys/health"
PG_HOST="${BAO_PG_HOST:-localhost}"
PG_PORT="${BAO_PG_PORT:-5432}"
PG_DATABASE="${BAO_PG_DATABASE:-openbao}"
PG_USER="${BAO_PG_USER:-openbao}"
PG_PASSWORD_FILE="${BAO_PG_PASSWORD_FILE:-/openbao/config/pg-password}"
TIMEOUT=5
VERBOSE=0

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=1
            shift
            ;;
        -h|--help)
            cat << EOF
Usage: $0 [OPTIONS]

Options:
  -v, --verbose    Enable verbose output
  -h, --help       Show this help message

Exit codes:
  0 - Healthy
  1 - OpenBao API unhealthy
  2 - PostgreSQL connectivity issue
  3 - Vault is sealed
  4 - Configuration error
EOF
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            exit 4
            ;;
    esac
done

# Logging functions
log_info() {
    [[ $VERBOSE -eq 1 ]] && echo "[INFO] $*"
}

log_error() {
    echo "[ERROR] $*" >&2
}

log_warn() {
    echo "[WARN] $*"
}

# Get PostgreSQL password
get_pg_password() {
    if [[ -f "$PG_PASSWORD_FILE" ]]; then
        cat "$PG_PASSWORD_FILE" | tr -d '[:space:]'
    else
        echo ""
    fi
}

# Check OpenBao health endpoint
check_openbao_health() {
    log_info "Checking OpenBao health endpoint: ${HEALTH_URL}"

    local response
    local http_code

    # Make health check request
    response=$(curl -sf --max-time ${TIMEOUT} "${HEALTH_URL}" 2>/dev/null) || {
        log_error "OpenBao health endpoint not responding"
        return 1
    }

    # Parse health response
    local initialized
    local sealed
    local standby

    initialized=$(echo "$response" | jq -r '.initialized // false')
    sealed=$(echo "$response" | jq -r '.sealed // true')
    standby=$(echo "$response" | jq -r '.standby // false')

    log_info "OpenBao status: initialized=${initialized}, sealed=${sealed}, standby=${standby}"

    # Check if initialized
    if [[ "$initialized" != "true" ]]; then
        log_error "OpenBao is not initialized"
        return 1
    fi

    # Check if sealed
    if [[ "$sealed" == "true" ]]; then
        log_error "OpenBao is sealed"
        return 3
    fi

    log_info "OpenBao health check passed"
    return 0
}

# Check PostgreSQL connectivity
check_postgresql() {
    log_info "Checking PostgreSQL connectivity: ${PG_HOST}:${PG_PORT}"

    local password
    password=$(get_pg_password)

    export PGPASSWORD="$password"

    # Check PostgreSQL connectivity
    if ! pg_isready -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DATABASE" >/dev/null 2>&1; then
        log_error "PostgreSQL is not accessible"
        unset PGPASSWORD
        return 2
    fi

    unset PGPASSWORD

    log_info "PostgreSQL connectivity check passed"
    return 0
}

# Check OpenBao process is running
check_process() {
    log_info "Checking OpenBao process"

    if pgrep -x "bao" >/dev/null 2>&1; then
        log_info "OpenBao process is running"
        return 0
    else
        log_error "OpenBao process not found"
        return 1
    fi
}

# Check disk space
check_disk_space() {
    log_info "Checking disk space"

    local usage
    usage=$(df /openbao/data 2>/dev/null | tail -1 | awk '{print $5}' | tr -d '%') || usage=0

    if [[ ${usage} -gt 95 ]]; then
        log_error "Disk usage critical: ${usage}%"
        return 1
    elif [[ ${usage} -gt 85 ]]; then
        log_warn "Disk usage high: ${usage}%"
    else
        log_info "Disk usage OK: ${usage}%"
    fi

    return 0
}

# Check audit log accessibility
check_audit_logs() {
    log_info "Checking audit log accessibility"

    if [[ -d "/openbao/logs" ]]; then
        if [[ -w "/openbao/logs" ]]; then
            log_info "Audit log directory is writable"
            return 0
        else
            log_warn "Audit log directory is not writable"
            return 0  # Non-fatal
        fi
    else
        log_warn "Audit log directory does not exist"
        return 0  # Non-fatal
    fi
}

# Verify cold vault mount exists
check_cold_vault_mount() {
    log_info "Checking cold vault mount"

    local response
    response=$(curl -sf --max-time ${TIMEOUT} "${VAULT_ADDR}/v1/sys/mounts" 2>/dev/null) || {
        log_warn "Cannot retrieve mount list"
        return 0  # Non-fatal
    }

    if echo "$response" | jq -e '.data["cold-vault/"]' >/dev/null 2>&1; then
        log_info "Cold vault mount exists"
        return 0
    else
        log_warn "Cold vault mount not found"
        return 0  # Non-fatal
    fi
}

# Main execution
main() {
    local exit_code=0
    local failed_checks=0

    [[ $VERBOSE -eq 1 ]] && echo "=== OpenBao Cold Vault Health Check ==="
    [[ $VERBOSE -eq 1 ]] && echo "Timestamp: $(date)"
    [[ $VERBOSE -eq 1 ]] && echo ""

    # Check OpenBao process
    check_process || {
        exit_code=1
        ((failed_checks++))
    }

    # Check PostgreSQL connectivity
    check_postgresql || {
        exit_code=2
        ((failed_checks++))
    }

    # Check OpenBao health
    check_openbao_health || {
        local health_exit=$?
        if [[ $health_exit -eq 3 ]]; then
            exit_code=3
        else
            exit_code=1
        fi
        ((failed_checks++))
    }

    # Check disk space (non-fatal warning)
    check_disk_space || {
        [[ $exit_code -eq 0 ]] && exit_code=1
    }

    # Check audit logs (non-fatal)
    check_audit_logs

    # Check cold vault mount (non-fatal)
    check_cold_vault_mount

    [[ $VERBOSE -eq 1 ]] && echo ""

    if [[ $exit_code -eq 0 ]]; then
        [[ $VERBOSE -eq 1 ]] && echo "=== Health Check: PASSED ==="
        exit 0
    else
        [[ $VERBOSE -eq 1 ]] && echo "=== Health Check: FAILED (${failed_checks} critical checks failed) ==="
        exit $exit_code
    fi
}

main "$@"
