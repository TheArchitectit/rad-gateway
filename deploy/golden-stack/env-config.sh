#!/bin/bash
#
# Golden Stack Environment Configuration Helper
# Location: deploy/golden-stack/env-config.sh
#
# Purpose: Validates environment variables, generates passwords if needed,
#          constructs database URLs, and exports for use in deployment scripts.
#
# Usage:
#   source ./env-config.sh           # Load and validate all env vars
#   source ./env-config.sh --check   # Only check, don't generate
#   source ./env-config.sh --export  # Export to file for systemd
#
# Safety: This script never logs or echoes actual secret values.
#

set -euo pipefail

# =============================================================================
# Configuration
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"
ENV_EXAMPLE="${SCRIPT_DIR}/.env.example"
EXPORT_FILE="${SCRIPT_DIR}/.env.export"

# Minimum password length for production
MIN_PASSWORD_LENGTH=32
REQUIRED_ENVIRONMENT_VARS=(
    "POSTGRES_USER"
    "POSTGRES_DB"
)

OPTIONAL_ENVIRONMENT_VARS=(
    "POSTGRES_PASSWORD"
    "INFISICAL_ENCRYPTION_KEY"
    "INFISICAL_JWT_SECRET"
    "OPENBAO_ROOT_TOKEN"
)

# Colors for output (disable if not terminal)
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# =============================================================================
# Logging Functions
# =============================================================================

log() {
    echo -e "${BLUE}[env-config]${NC} $*"
}

log_ok() {
    echo -e "${GREEN}[env-config]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[env-config]${NC} WARNING: $*"
}

log_error() {
    echo -e "${RED}[env-config]${NC} ERROR: $*" >&2
}

# =============================================================================
# Security Functions
# =============================================================================

# Generate a secure random password
# Usage: generate_password [length]
generate_password() {
    local length="${1:-32}"
    if command -v openssl &> /dev/null; then
        openssl rand -base64 "$length" | tr -d '=+/' | cut -c1-$length
    elif command -v pwgen &> /dev/null; then
        pwgen -s "$length" 1
    else
        # Fallback using /dev/urandom
        head /dev/urandom | tr -dc 'a-zA-Z0-9' | head -c "$length"
    fi
}

# Generate a base64-encoded 32-byte key
# Usage: generate_encryption_key
generate_encryption_key() {
    if command -v openssl &> /dev/null; then
        openssl rand -base64 32
    else
        # Fallback
        head /dev/urandom | base64 | head -c 44
    fi
}

# Check if running in production environment
is_production() {
    [[ "${RAD_ENVIRONMENT:-}" == "production" ]] || [[ "${RAD_ENVIRONMENT:-}" == "prod" ]]
}

# Validate password strength
validate_password_strength() {
    local password="$1"
    local min_length="${2:-$MIN_PASSWORD_LENGTH}"

    if [[ ${#password} -lt $min_length ]]; then
        return 1
    fi

    # Check for complexity (at least 3 of 4: upper, lower, digit, special)
    local complexity=0
    [[ "$password" =~ [A-Z] ]] && ((complexity++))
    [[ "$password" =~ [a-z] ]] && ((complexity++))
    [[ "$password" =~ [0-9] ]] && ((complexity++))
    [[ "$password" =~ [^a-zA-Z0-9] ]] && ((complexity++))

    [[ $complexity -ge 3 ]]
}

# =============================================================================
# Environment Variable Functions
# =============================================================================

# Check if required environment variables are set
check_required_vars() {
    local missing=()
    local var

    for var in "${REQUIRED_ENVIRONMENT_VARS[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            missing+=("$var")
        fi
    done

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Missing required environment variables:"
        printf '  - %s\n' "${missing[@]}" >&2
        return 1
    fi

    log_ok "All required environment variables present"
    return 0
}

# Generate missing passwords if auto-generate is enabled
ensure_passwords() {
    local generated=false

    # PostgreSQL password
    if [[ -z "${POSTGRES_PASSWORD:-}" ]]; then
        if [[ "${AUTO_GENERATE_PASSWORDS:-false}" == "true" ]]; then
            POSTGRES_PASSWORD=$(generate_password 32)
            log "Generated POSTGRES_PASSWORD"
            generated=true
        else
            log_warn "POSTGRES_PASSWORD not set. Set AUTO_GENERATE_PASSWORDS=true to auto-generate"
        fi
    elif is_production && ! validate_password_strength "$POSTGRES_PASSWORD"; then
        log_warn "POSTGRES_PASSWORD is weak. Consider using a stronger password in production"
    fi

    # Infisical encryption key
    if [[ -z "${INFISICAL_ENCRYPTION_KEY:-}" ]]; then
        if [[ "${AUTO_GENERATE_PASSWORDS:-false}" == "true" ]]; then
            INFISICAL_ENCRYPTION_KEY=$(generate_encryption_key)
            log "Generated INFISICAL_ENCRYPTION_KEY"
            generated=true
        else
            log_warn "INFISICAL_ENCRYPTION_KEY not set. Set AUTO_GENERATE_PASSWORDS=true to auto-generate"
        fi
    fi

    # Infisical JWT secret
    if [[ -z "${INFISICAL_JWT_SECRET:-}" ]]; then
        if [[ "${AUTO_GENERATE_PASSWORDS:-false}" == "true" ]]; then
            INFISICAL_JWT_SECRET=$(generate_password 64)
            log "Generated INFISICAL_JWT_SECRET"
            generated=true
        else
            log_warn "INFISICAL_JWT_SECRET not set. Set AUTO_GENERATE_PASSWORDS=true to auto-generate"
        fi
    fi

    # RAD API keys
    if [[ -z "${RAD_API_KEYS:-}" ]]; then
        local dev_key
        dev_key=$(generate_password 32)
        RAD_API_KEYS="default:$dev_key"
        log "Generated RAD_API_KEYS"
        generated=true
    fi

    if [[ "$generated" == "true" ]]; then
        log_warn "Auto-generated passwords. Consider saving them to your password manager."
    fi
}

# Construct database URLs from component parts
build_database_urls() {
    local pg_user="${POSTGRES_USER:-secretstack}"
    local pg_pass="${POSTGRES_PASSWORD:-}"
    local pg_host="${POSTGRES_HOST:-localhost}"
    local pg_port="${POSTGRES_PORT:-5432}"
    local pg_db="${POSTGRES_DB:-secrets}"
    local pg_ssl="${POSTGRES_SSL_MODE:-prefer}"

    # Build connection URL (only if password is set)
    if [[ -n "$pg_pass" ]]; then
        # URL encode the password to handle special characters
        local encoded_pass
        encoded_pass=$(printf '%s' "$pg_pass" | jq -sRr @uri 2>/dev/null || echo "$pg_pass")

        local db_url="postgresql://${pg_user}:${encoded_pass}@${pg_host}:${pg_port}/${pg_db}?sslmode=${pg_ssl}"

        # Set service-specific URLs if not already defined
        if [[ -z "${INFISICAL_DB_URL:-}" ]]; then
            INFISICAL_DB_URL="$db_url"
            log "Constructed INFISICAL_DB_URL"
        fi

        if [[ -z "${OPENBAO_DB_URL:-}" ]]; then
            OPENBAO_DB_URL="$db_url"
            log "Constructed OPENBAO_DB_URL"
        fi
    else
        log_warn "POSTGRES_PASSWORD not set - cannot construct database URLs"
    fi
}

# Load environment from file
load_env_file() {
    if [[ -f "$ENV_FILE" ]]; then
        log "Loading environment from $ENV_FILE"
        set -a
        # shellcheck source=/dev/null
        source "$ENV_FILE"
        set +a
    else
        log_warn "Environment file not found: $ENV_FILE"
        if [[ -f "$ENV_EXAMPLE" ]]; then
            log "Creating from example file..."
            cp "$ENV_EXAMPLE" "$ENV_FILE"
            log_warn "Please edit $ENV_FILE with your actual values"
        fi
    fi
}

# Export environment to a file suitable for systemd
export_for_systemd() {
    log "Exporting environment to $EXPORT_FILE"

    # Create temp file
    local temp_file
    temp_file=$(mktemp)

    # Write exportable variables (excluding sensitive data)
    {
        echo "# Golden Stack Environment - Systemd Export"
        echo "# Generated: $(date -Iseconds)"
        echo "#"
        echo "# SECURITY: This file should be protected with chmod 600"
        echo ""

        # Database configuration
        echo "POSTGRES_USER=${POSTGRES_USER:-secretstack}"
        echo "POSTGRES_HOST=${POSTGRES_HOST:-localhost}"
        echo "POSTGRES_PORT=${POSTGRES_PORT:-5432}"
        echo "POSTGRES_DB=${POSTGRES_DB:-secrets}"
        echo "POSTGRES_SSL_MODE=${POSTGRES_SSL_MODE:-prefer}"
        echo ""

        # Service URLs
        echo "INFISICAL_API_URL=${INFISICAL_API_URL:-http://localhost:8080}"
        echo "OPENBAO_API_ADDR=${OPENBAO_API_ADDR:-http://0.0.0.0:8200}"
        echo ""

        # RAD Gateway
        echo "RAD_LISTEN_ADDR=${RAD_LISTEN_ADDR:-:8090}"
        echo "RAD_LOG_LEVEL=${RAD_LOG_LEVEL:-info}"
        echo "RAD_ENVIRONMENT=${RAD_ENVIRONMENT:-alpha}"
        echo ""

        # Network
        echo "GOLDEN_STACK_NETWORK=${GOLDEN_STACK_NETWORK:-secret-stack}"
        echo ""

        # Note: Secrets are NOT exported here - they should be loaded from
        # Infisical or a secure token file at runtime
        echo "# Secrets loaded at runtime from Infisical or token files"
        echo "INFISICAL_TOKEN_FILE=${INFISICAL_TOKEN_FILE:-/opt/radgateway01/config/infisical-token}"

    } > "$temp_file"

    # Move with permissions
    mv "$temp_file" "$EXPORT_FILE"
    chmod 600 "$EXPORT_FILE"

    log_ok "Environment exported to $EXPORT_FILE"
}

# Validate the complete configuration
validate_configuration() {
    local errors=0

    log "Validating configuration..."

    # Check required vars
    if ! check_required_vars; then
        ((errors++))
    fi

    # Check database connectivity (if psql available)
    if command -v psql &> /dev/null && [[ -n "${POSTGRES_PASSWORD:-}" ]]; then
        log "Testing PostgreSQL connectivity..."
        if ! pg_isready -h "${POSTGRES_HOST:-localhost}" -p "${POSTGRES_PORT:-5432}" &> /dev/null; then
            log_warn "PostgreSQL does not appear to be running at ${POSTGRES_HOST:-localhost}:${POSTGRES_PORT:-5432}"
        else
            log_ok "PostgreSQL is reachable"
        fi
    fi

    # Check Infisical connectivity
    if [[ -n "${INFISICAL_API_URL:-}" ]]; then
        log "Testing Infisical connectivity..."
        if curl -sf "${INFISICAL_API_URL}/api/status" &> /dev/null; then
            log_ok "Infisical is accessible at $INFISICAL_API_URL"
        else
            log_warn "Infisical not accessible at $INFISICAL_API_URL"
        fi
    fi

    # Check OpenBao connectivity
    if [[ -n "${OPENBAO_API_ADDR:-}" ]]; then
        log "Testing OpenBao connectivity..."
        local bao_addr="${OPENBAO_API_ADDR}"
        if curl -sf "${bao_addr}/v1/sys/health" &> /dev/null; then
            log_ok "OpenBao is accessible at $bao_addr"
        else
            log_warn "OpenBao not accessible at $bao_addr"
        fi
    fi

    # Validate service token format
    if [[ -n "${INFISICAL_SERVICE_TOKEN:-}" ]]; then
        if [[ "$INFISICAL_SERVICE_TOKEN" =~ ^st\.[a-f0-9]+\.[a-f0-9]+\.[a-f0-9]+$ ]]; then
            log_ok "INFISICAL_SERVICE_TOKEN format is valid"
        else
            log_warn "INFISICAL_SERVICE_TOKEN format appears invalid (expected: st.xxx.yyy.zzz)"
        fi
    fi

    if [[ $errors -eq 0 ]]; then
        log_ok "Configuration validation complete"
        return 0
    else
        log_error "Configuration validation failed with $errors errors"
        return 1
    fi
}

# Print summary of configuration (without secrets)
print_summary() {
    echo ""
    echo "============================================================================="
    echo "Golden Stack Environment Configuration Summary"
    echo "============================================================================="
    echo ""
    echo "PostgreSQL:"
    echo "  User:     ${POSTGRES_USER:-<not set>}"
    echo "  Host:     ${POSTGRES_HOST:-<not set>}:${POSTGRES_PORT:-5432}"
    echo "  Database: ${POSTGRES_DB:-<not set>}"
    echo "  Password: $([[ -n "${POSTGRES_PASSWORD:-}" ]] && echo "[SET]" || echo "[NOT SET]")"
    echo ""
    echo "Infisical (Hot Vault):"
    echo "  API URL:  ${INFISICAL_API_URL:-<not set>}"
    echo "  DB URL:   $([[ -n "${INFISICAL_DB_URL:-}" ]] && echo "[SET]" || echo "[NOT SET]")"
    echo "  Enc Key:  $([[ -n "${INFISICAL_ENCRYPTION_KEY:-}" ]] && echo "[SET]" || echo "[NOT SET]")"
    echo "  Token:    $([[ -n "${INFISICAL_SERVICE_TOKEN:-}" ]] && echo "[SET]" || echo "[NOT SET]")"
    echo ""
    echo "OpenBao (Cold Vault):"
    echo "  API Addr: ${OPENBAO_API_ADDR:-<not set>}"
    echo "  DB URL:   $([[ -n "${OPENBAO_DB_URL:-}" ]] && echo "[SET]" || echo "[NOT SET]")"
    echo "  UI:       ${OPENBAO_UI_ENABLED:-true}"
    echo ""
    echo "RAD Gateway:"
    echo "  Listen:   ${RAD_LISTEN_ADDR:-:8090}"
    echo "  API Keys: $([[ -n "${RAD_API_KEYS:-}" ]] && echo "[SET]" || echo "[NOT SET]")"
    echo ""
    echo "Environment: ${RAD_ENVIRONMENT:-dev}"
    echo "============================================================================="
    echo ""
}

# =============================================================================
# Main Execution
# =============================================================================

main() {
    local mode="${1:-load}"

    case "$mode" in
        --check|-c)
            log "Running in check-only mode"
            load_env_file
            check_required_vars
            validate_configuration
            ;;
        --export|-e)
            log "Running in export mode"
            load_env_file
            ensure_passwords
            build_database_urls
            export_for_systemd
            ;;
        --summary|-s)
            load_env_file
            ensure_passwords
            build_database_urls
            print_summary
            ;;
        --generate|-g)
            log "Running in generate mode"
            load_env_file
            AUTO_GENERATE_PASSWORDS=true ensure_passwords
            build_database_urls
            print_summary
            log_warn "Auto-generated passwords are temporary. Save them securely."
            ;;
        --help|-h)
            cat << EOF
Golden Stack Environment Configuration Helper

Usage: source $0 [OPTION]

Options:
  --check, -c      Validate configuration without generating values
  --export, -e     Export environment to systemd-compatible file
  --summary, -s    Print configuration summary
  --generate, -g   Generate missing passwords automatically
  --help, -h       Show this help message

Environment Variables:
  AUTO_GENERATE_PASSWORDS=true   Auto-generate missing passwords
  RAD_ENVIRONMENT=production     Set production mode (stricter validation)

Examples:
  source ./env-config.sh              # Load and validate
  source ./env-config.sh --generate   # Generate missing passwords
  source ./env-config.sh --export     # Export for systemd

Safety:
  - This script never logs or echoes actual secret values
  - Generated passwords are only printed once (if AUTO_GENERATE_PASSWORDS=true)
  - Use --check first to review what will be generated
EOF
            ;;
        *)
            # Default: load, validate, and prepare
            load_env_file
            ensure_passwords
            build_database_urls

            if check_required_vars && validate_configuration; then
                log_ok "Environment configuration ready"
                export POSTGRES_USER POSTGRES_PASSWORD POSTGRES_HOST POSTGRES_PORT POSTGRES_DB
                export POSTGRES_SSL_MODE INFISICAL_DB_URL OPENBAO_DB_URL
                export INFISICAL_API_URL INFISICAL_ENCRYPTION_KEY INFISICAL_JWT_SECRET
                export OPENBAO_API_ADDR OPENBAO_UI_ENABLED
                export RAD_LISTEN_ADDR RAD_LOG_LEVEL RAD_ENVIRONMENT RAD_API_KEYS
                export RAD_DATA_DIR INFISICAL_TOKEN_FILE
                export GOLDEN_STACK_NETWORK GOLDEN_STACK_SUBNET
            else
                log_error "Configuration incomplete. Fix errors and re-run."
                return 1
            fi
            ;;
    esac
}

# Run main if executed directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
else
    # Being sourced - auto-load configuration
    main "${1:-}"
fi
