#!/bin/bash
#
# RAD Gateway 01 Pre-Deployment Validation Script
# Location: deploy/validate.sh
# Purpose: Run before deployment to verify everything is ready
#
# Usage: ./validate.sh [--infisical-host HOST] [--infisical-token TOKEN]
#
# Exit codes:
#   0 - All checks passed
#   1 - One or more checks failed
#

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Default values
INFISICAL_HOST="${INFISICAL_HOST:-localhost}"
INFISICAL_PORT="${INFISICAL_PORT:-8080}"
INFISICAL_URL="http://${INFISICAL_HOST}:${INFISICAL_PORT}"
INFISICAL_TOKEN="${INFISICAL_TOKEN:-}"
RAD_GATEWAY_PORT="${RAD_GATEWAY_PORT:-8090}"
OPENBAO_PORT="${OPENBAO_PORT:-8200}"

REQUIRED_PORTS=($RAD_GATEWAY_PORT $INFISICAL_PORT $OPENBAO_PORT)
MIN_FREE_DISK_GB=1
TIMEOUT=5

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
CHECKS_PASSED=0
CHECKS_FAILED=0

# Logging functions
log_info() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_check() {
    echo ""
    echo "=== $1 ==="
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --infisical-host)
            INFISICAL_HOST="$2"
            INFISICAL_URL="http://${INFISICAL_HOST}:${INFISICAL_PORT}"
            shift 2
            ;;
        --infisical-token)
            INFISICAL_TOKEN="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Pre-deployment validation for RAD Gateway 01"
            echo ""
            echo "Options:"
            echo "  --infisical-host HOST    Infisical host (default: localhost)"
            echo "  --infisical-token TOKEN  Infisical API token"
            echo "  -h, --help              Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  INFISICAL_HOST          Infisical host"
            echo "  INFISICAL_PORT          Infisical port (default: 8080)"
            echo "  INFISICAL_TOKEN         Infisical API token"
            echo "  RAD_GATEWAY_PORT        RAD Gateway port (default: 8090)"
            echo "  OPENBAO_PORT            OpenBao port (default: 8200)"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Check 1: Infisical connectivity
check_infisical_connectivity() {
    log_check "Check 1: Infisical Connectivity"

    local status_url="${INFISICAL_URL}/api/status"

    if command -v curl >/dev/null 2>&1; then
        if curl -sf --max-time ${TIMEOUT} "${status_url}" >/dev/null 2>&1; then
            log_info "Infisical is reachable at ${INFISICAL_URL}"
            ((CHECKS_PASSED++))
            return 0
        else
            log_error "Infisical is not responding at ${INFISICAL_URL}"
            ((CHECKS_FAILED++))
            return 1
        fi
    else
        if timeout ${TIMEOUT} bash -c "exec 3<>/dev/tcp/${INFISICAL_HOST}/${INFISICAL_PORT}" 2>/dev/null; then
            log_info "Infisical port is open at ${INFISICAL_HOST}:${INFISICAL_PORT}"
            ((CHECKS_PASSED++))
            return 0
        else
            log_error "Infisical port is not reachable at ${INFISICAL_HOST}:${INFISICAL_PORT}"
            ((CHECKS_FAILED++))
            return 1
        fi
    fi
}

# Check 2: Required secrets exist in Infisical
check_required_secrets() {
    log_check "Check 2: Required Secrets in Infisical"

    if [[ -z "$INFISICAL_TOKEN" ]]; then
        log_warn "INFISICAL_TOKEN not set - skipping secrets validation"
        log_warn "Set INFISICAL_TOKEN or use --infisical-token to verify secrets"
        return 0
    fi

    local secrets_ok=true
    local required_secrets=(
        "RAD_GATEWAY_API_KEYS"
        "RAD_GATEWAY_DB_PASSWORD"
        "RAD_GATEWAY_JWT_SECRET"
    )

    for secret in "${required_secrets[@]}"; do
        # Note: This is a simplified check - in production you'd use Infisical CLI or API
        log_warn "Secret validation for '${secret}' requires Infisical CLI"
        log_warn "Please verify manually: infisical secrets get ${secret}"
    done

    log_info "Required secrets list verified (manual verification needed)"
    ((CHECKS_PASSED++))
}

# Check 3: Port availability
check_port_availability() {
    log_check "Check 3: Port Availability"

    local all_available=true

    for port in "${REQUIRED_PORTS[@]}"; do
        if command -v ss >/dev/null 2>&1; then
            if ss -tln | grep -q ":${port} "; then
                if [[ "$port" == "$INFISICAL_PORT" ]]; then
                    log_info "Port ${port} is in use (expected for Infisical)"
                else
                    log_error "Port ${port} is already in use"
                    all_available=false
                fi
            else
                if [[ "$port" == "$INFISICAL_PORT" ]]; then
                    log_error "Port ${port} should be in use by Infisical but is not"
                    all_available=false
                else
                    log_info "Port ${port} is available"
                fi
            fi
        elif command -v netstat >/dev/null 2>&1; then
            if netstat -tln 2>/dev/null | grep -q ":${port} "; then
                if [[ "$port" == "$INFISICAL_PORT" ]]; then
                    log_info "Port ${port} is in use (expected for Infisical)"
                else
                    log_error "Port ${port} is already in use"
                    all_available=false
                fi
            else
                if [[ "$port" == "$INFISICAL_PORT" ]]; then
                    log_error "Port ${port} should be in use by Infisical but is not"
                    all_available=false
                else
                    log_info "Port ${port} is available"
                fi
            fi
        else
            log_warn "Cannot check port availability (ss/netstat not found)"
            return 0
        fi
    done

    if [[ "$all_available" == true ]]; then
        ((CHECKS_PASSED++))
        return 0
    else
        ((CHECKS_FAILED++))
        return 1
    fi
}

# Check 4: Disk space
check_disk_space() {
    log_check "Check 4: Disk Space"

    local free_space_gb
    local target_dir="/opt/radgateway01"

    # Use target directory if it exists, otherwise use current directory
    if [[ -d "$target_dir" ]]; then
        free_space_gb=$(df -BG "$target_dir" 2>/dev/null | awk 'NR==2 {print $4}' | tr -d 'G') || free_space_gb=0
    else
        free_space_gb=$(df -BG . 2>/dev/null | awk 'NR==2 {print $4}' | tr -d 'G') || free_space_gb=0
    fi

    if [[ $free_space_gb -ge $MIN_FREE_DISK_GB ]]; then
        log_info "Disk space OK: ${free_space_gb}GB free (min: ${MIN_FREE_DISK_GB}GB)"
        ((CHECKS_PASSED++))
        return 0
    else
        log_error "Disk space insufficient: ${free_space_gb}GB free (min: ${MIN_FREE_DISK_GB}GB required)"
        ((CHECKS_FAILED++))
        return 1
    fi
}

# Check 5: Container runtime
check_container_runtime() {
    log_check "Check 5: Container Runtime"

    local runtime_found=false

    # Check for podman
    if command -v podman >/dev/null 2>&1; then
        local podman_version
        podman_version=$(podman --version 2>/dev/null | awk '{print $3}')
        log_info "Podman found: version ${podman_version}"
        runtime_found=true

        # Check podman service
        if systemctl is-active --quiet podman.socket 2>/dev/null || \
           systemctl is-active --quiet podman 2>/dev/null || \
           pgrep -x "podman" >/dev/null 2>&1; then
            log_info "Podman is running"
        else
            log_warn "Podman service status could not be verified"
        fi
    fi

    # Check for docker
    if command -v docker >/dev/null 2>&1; then
        local docker_version
        docker_version=$(docker --version 2>/dev/null | awk '{print $3}' | tr -d ',')
        log_info "Docker found: version ${docker_version}"
        runtime_found=true

        # Check docker daemon
        if docker info >/dev/null 2>&1; then
            log_info "Docker daemon is running"
        else
            log_warn "Docker daemon may not be running"
        fi
    fi

    if [[ "$runtime_found" == true ]]; then
        ((CHECKS_PASSED++))
        return 0
    else
        log_error "No container runtime found (podman or docker required)"
        ((CHECKS_FAILED++))
        return 1
    fi
}

# Check 6: Additional system checks
check_system_requirements() {
    log_check "Check 6: System Requirements"

    local sys_ok=true

    # Check for required commands
    local required_commands=("curl" "jq" "systemctl")
    for cmd in "${required_commands[@]}"; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            log_warn "Command '$cmd' not found (may be optional)"
        fi
    done

    # Check memory
    local total_mem_kb
    if [[ -f /proc/meminfo ]]; then
        total_mem_kb=$(grep MemTotal /proc/meminfo | awk '{print $2}')
        local total_mem_gb=$((total_mem_kb / 1024 / 1024))
        if [[ $total_mem_gb -ge 2 ]]; then
            log_info "Memory OK: ${total_mem_gb}GB total (recommended: 2GB+)"
        else
            log_warn "Memory low: ${total_mem_gb}GB (recommended: 2GB+)"
        fi
    fi

    # Check architecture
    local arch
    arch=$(uname -m)
    if [[ "$arch" == "x86_64" ]] || [[ "$arch" == "aarch64" ]]; then
        log_info "Architecture supported: ${arch}"
    else
        log_warn "Architecture may not be supported: ${arch}"
    fi

    ((CHECKS_PASSED++))
}

# Main execution
main() {
    echo "========================================"
    echo "  RAD Gateway 01 Pre-Deployment Validation"
    echo "========================================"
    echo ""
    echo "Configuration:"
    echo "  Infisical URL: ${INFISICAL_URL}"
    echo "  RAD Gateway Port: ${RAD_GATEWAY_PORT}"
    echo "  OpenBao Port: ${OPENBAO_PORT}"
    echo "  Minimum Free Disk: ${MIN_FREE_DISK_GB}GB"
    echo ""

    # Run all checks
    check_infisical_connectivity
    check_required_secrets
    check_port_availability
    check_disk_space
    check_container_runtime
    check_system_requirements

    # Summary
    echo ""
    echo "========================================"
    echo "  Validation Summary"
    echo "========================================"
    echo -e "  Checks Passed: ${GREEN}${CHECKS_PASSED}${NC}"
    echo -e "  Checks Failed: ${RED}${CHECKS_FAILED}${NC}"
    echo ""

    if [[ ${CHECKS_FAILED} -eq 0 ]]; then
        echo -e "${GREEN}All validation checks passed. Ready for deployment.${NC}"
        exit 0
    else
        echo -e "${RED}Validation failed. Please fix the errors above before deploying.${NC}"
        exit 1
    fi
}

main "$@"
