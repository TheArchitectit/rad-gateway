#!/bin/bash
#
# PostgreSQL Health Check Script
# Location: /etc/postgresql/scripts/health-check.sh
# Purpose: Comprehensive health verification for PostgreSQL container
#

set -euo pipefail

# Configuration
VERBOSE=0
TIMEOUT=5
MAX_RETRIES=3

# Database names from environment or defaults
INFISICAL_DB="${INFISICAL_DB_NAME:-infisical}"
OPENBAO_DB="${OPENBAO_DB_NAME:-openbao}"

# Logging functions
log_info() {
    [[ $VERBOSE -eq 1 ]] && echo "[HEALTH] [$(date -Iseconds)] INFO: $*"
}

log_error() {
    echo "[HEALTH] [$(date -Iseconds)] ERROR: $*" >&2
}

log_warn() {
    echo "[HEALTH] [$(date -Iseconds)] WARN: $*"
}

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                VERBOSE=1
                shift
                ;;
            -h|--help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  -v, --verbose    Enable verbose output"
                echo "  -h, --help       Show this help message"
                echo ""
                echo "Exit codes:"
                echo "  0 - PostgreSQL is healthy"
                echo "  1 - PostgreSQL is not accepting connections"
                echo "  2 - Required databases do not exist"
                echo "  3 - Storage issues detected"
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                exit 1
                ;;
        esac
    done
}

# Check if PostgreSQL is accepting connections
check_postgresql_connection() {
    log_info "Checking PostgreSQL connection..."

    local retries=0
    while [[ $retries -lt $MAX_RETRIES ]]; do
        if pg_isready -h localhost -p 5432 -U postgres -t "$TIMEOUT" >/dev/null 2>&1; then
            log_info "PostgreSQL is accepting connections"
            return 0
        fi

        retries=$((retries + 1))
        log_warn "Connection attempt $retries/$MAX_RETRIES failed, retrying..."
        sleep 1
    done

    log_error "PostgreSQL is not accepting connections after $MAX_RETRIES attempts"
    return 1
}

# Check if specific database exists
check_database_exists() {
    local db_name="$1"
    log_info "Checking if database '$db_name' exists..."

    local result
    result=$(psql -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$db_name'" 2>/dev/null || echo "0")

    if [[ "$result" == "1" ]]; then
        log_info "Database '$db_name' exists"
        return 0
    else
        log_error "Database '$db_name' does not exist"
        return 1
    fi
}

# Check if we can connect to a specific database
check_database_connection() {
    local db_name="$1"
    log_info "Checking connection to database '$db_name'..."

    if psql -U postgres -d "$db_name" -c "SELECT 1;" >/dev/null 2>&1; then
        log_info "Successfully connected to database '$db_name'"
        return 0
    else
        log_error "Cannot connect to database '$db_name'"
        return 1
    fi
}

# Check disk space
check_disk_space() {
    log_info "Checking disk space..."

    local pgdata="${PGDATA:-/var/lib/postgresql/data}"
    local usage

    usage=$(df "$pgdata" 2>/dev/null | tail -1 | awk '{print $5}' | tr -d '%') || usage=0

    if [[ ${usage} -gt 95 ]]; then
        log_error "Disk usage critical: ${usage}%"
        return 3
    elif [[ ${usage} -gt 85 ]]; then
        log_warn "Disk usage high: ${usage}%"
    else
        log_info "Disk usage OK: ${usage}%"
    fi

    return 0
}

# Check replication status (if applicable)
check_replication_status() {
    log_info "Checking replication status..."

    # Check if this is a primary or replica
    local is_in_recovery
    is_in_recovery=$(psql -U postgres -tAc "SELECT pg_is_in_recovery();" 2>/dev/null || echo "f")

    if [[ "$is_in_recovery" == "t" ]]; then
        log_info "PostgreSQL is in recovery mode (replica)"

        # Check replication lag
        local lag_bytes
        lag_bytes=$(psql -U postgres -tAc "SELECT pg_last_wal_receive_lsn() - pg_last_wal_replay_lsn();" 2>/dev/null || echo "0")

        if [[ "$lag_bytes" != "0" ]]; then
            log_warn "Replication lag detected: $lag_bytes bytes"
        else
            log_info "Replication is up to date"
        fi
    else
        log_info "PostgreSQL is primary (not in recovery)"
    fi

    return 0
}

# Check connection count
check_connection_count() {
    log_info "Checking connection count..."

    local max_connections
    local current_connections
    local usage_percent

    max_connections=$(psql -U postgres -tAc "SHOW max_connections;" 2>/dev/null || echo "100")
    current_connections=$(psql -U postgres -tAc "SELECT count(*) FROM pg_stat_activity;" 2>/dev/null || echo "0")

    if [[ -n "$max_connections" && "$max_connections" -gt 0 ]]; then
        usage_percent=$((current_connections * 100 / max_connections))

        if [[ $usage_percent -gt 90 ]]; then
            log_error "Connection count critical: $current_connections/$max_connections (${usage_percent}%)"
            return 1
        elif [[ $usage_percent -gt 75 ]]; then
            log_warn "Connection count high: $current_connections/$max_connections (${usage_percent}%)"
        else
            log_info "Connection count OK: $current_connections/$max_connections (${usage_percent}%)"
        fi
    fi

    return 0
}

# Check for PostgreSQL errors in recent logs
check_recent_errors() {
    log_info "Checking for recent errors..."

    # Query for recent errors in the last 5 minutes
    local error_count
    error_count=$(psql -U postgres -tAc "
        SELECT count(*)
        FROM pg_stat_activity
        WHERE state = 'idle in transaction (aborted)'
        OR wait_event_type = 'Lock';
    " 2>/dev/null || echo "0")

    if [[ "$error_count" -gt 0 ]]; then
        log_warn "Found $error_count problematic connections"
    else
        log_info "No problematic connections found"
    fi

    return 0
}

# Main health check execution
main() {
    parse_args "$@"

    log_info "Starting PostgreSQL health check..."
    log_info "Checking databases: $INFISICAL_DB, $OPENBAO_DB"

    local exit_code=0

    # Check 1: PostgreSQL connection
    if ! check_postgresql_connection; then
        exit_code=1
    fi

    # Check 2: Required databases exist
    if [[ $exit_code -eq 0 ]]; then
        if ! check_database_exists "$INFISICAL_DB"; then
            exit_code=2
        fi

        if ! check_database_exists "$OPENBAO_DB"; then
            exit_code=2
        fi
    fi

    # Check 3: Can connect to databases
    if [[ $exit_code -eq 0 ]]; then
        if ! check_database_connection "$INFISICAL_DB"; then
            log_warn "Cannot connect to Infisical database"
        fi

        if ! check_database_connection "$OPENBAO_DB"; then
            log_warn "Cannot connect to OpenBao database"
        fi
    fi

    # Check 4: Disk space
    if ! check_disk_space; then
        exit_code=3
    fi

    # Check 5: Connection count (only if basic checks passed)
    if [[ $exit_code -eq 0 ]]; then
        check_connection_count || true
    fi

    # Check 6: Recent errors (only if basic checks passed)
    if [[ $exit_code -eq 0 ]]; then
        check_recent_errors || true
    fi

    # Check 7: Replication status (optional, don't fail on this)
    if [[ $exit_code -eq 0 ]]; then
        check_replication_status || true
    fi

    # Final status
    if [[ $exit_code -eq 0 ]]; then
        log_info "Health check PASSED"
        exit 0
    else
        log_error "Health check FAILED with exit code $exit_code"
        exit $exit_code
    fi
}

# Execute main function
main "$@"
