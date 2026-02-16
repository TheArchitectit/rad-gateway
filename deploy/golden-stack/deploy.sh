#!/bin/bash
# Golden Stack Deployment Script
# Deploys OpenBao + Infisical + PostgreSQL as a unified secrets management stack
#
# Usage: ./deploy-golden-stack.sh [environment]
#   environment: dev|staging|production (default: dev)

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENVIRONMENT="${1:-dev}"
POD_NAME="golden-stack"
NETWORK_NAME="golden-stack-net"

# Ports
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
INFISICAL_PORT="${INFISICAL_PORT:-8080}"
OPENBAO_PORT="${OPENBAO_PORT:-8200}"

# Data volumes
DATA_DIR="${DATA_DIR:-/opt/golden-stack}"
BACKUP_DIR="${BACKUP_DIR:-/backup/golden-stack}"

# Container images
POSTGRES_IMAGE="${POSTGRES_IMAGE:-postgres:16-alpine}"
INFISICAL_IMAGE="${INFISICAL_IMAGE:-infisical/infisical:latest}"
OPENBAO_IMAGE="${OPENBAO_IMAGE:-openbao/openbao:latest}"

echo -e "${GREEN}=== Golden Stack Deployment ===${NC}"
echo "Environment: $ENVIRONMENT"
echo "Data Directory: $DATA_DIR"
echo ""

# =============================================================================
# Pre-flight Checks
# =============================================================================

check_prerequisites() {
    echo -e "${YELLOW}Checking prerequisites...${NC}"

    # Check if running as root for production
    if [[ "$ENVIRONMENT" == "production" && $EUID -ne 0 ]]; then
        echo -e "${RED}Error: Production deployment must run as root${NC}"
        exit 1
    fi

    # Check for podman
    if ! command -v podman &> /dev/null; then
        echo -e "${RED}Error: podman is not installed${NC}"
        exit 1
    fi

    # Check for openssl (for generating secrets)
    if ! command -v openssl &> /dev/null; then
        echo -e "${RED}Error: openssl is not installed${NC}"
        exit 1
    fi

    echo -e "${GREEN}✓ Prerequisites OK${NC}"
}

# =============================================================================
# Directory Setup
# =============================================================================

setup_directories() {
    echo -e "${YELLOW}Setting up directories...${NC}"

    # Create data directories
    mkdir -p "$DATA_DIR"/{postgres,infisical,openbao,backups}
    mkdir -p "$DATA_DIR/postgres/init"

    # Set permissions
    chown -R 999:999 "$DATA_DIR/postgres" 2>/dev/null || true  # postgres user

    echo -e "${GREEN}✓ Directories created${NC}"
}

# =============================================================================
# Environment Configuration
# =============================================================================

generate_secrets() {
    echo -e "${YELLOW}Generating secrets...${NC}"

    SECRETS_FILE="$DATA_DIR/.env"

    if [[ -f "$SECRETS_FILE" ]]; then
        echo -e "${YELLOW}Secrets file already exists, keeping existing values${NC}"
        return 0
    fi

    # Generate random passwords
    POSTGRES_PASSWORD=$(openssl rand -base64 32)
    INFISICAL_ENCRYPTION_KEY=$(openssl rand -base64 32)
    OPENBAO_ROOT_TOKEN=$(openssl rand -base64 32)

    cat > "$SECRETS_FILE" << EOF
# Golden Stack Environment Variables
# Generated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
# Environment: $ENVIRONMENT

# PostgreSQL Configuration
POSTGRES_USER=goldenstack
POSTGRES_PASSWORD=$POSTGRES_PASSWORD
POSTGRES_DB=goldenstack
PGDATA=/var/lib/postgresql/data

# Infisical Configuration
INFISICAL_DB_URL=postgresql://goldenstack:${POSTGRES_PASSWORD}@localhost:${POSTGRES_PORT}/goldenstack
INFISICAL_ENCRYPTION_KEY=$INFISICAL_ENCRYPTION_KEY
INFISICAL_TELEMETRY_ENABLED=false

# OpenBao Configuration
BAO_PG_CONNECTION_URL=postgresql://goldenstack:${POSTGRES_PASSWORD}@localhost:${POSTGRES_PORT}/goldenstack
BAO_API_ADDR=http://0.0.0.0:8200
BAO_CLUSTER_ADDR=https://0.0.0.0:8201
BAO_LOG_LEVEL=info
BAO_UI=true

# Cold Vault Settings
BAO_COLD_VAULT_RETENTION_DAYS=3650
BAO_COLD_VAULT_MAX_VERSIONS=100
EOF

    chmod 600 "$SECRETS_FILE"
    echo -e "${GREEN}✓ Secrets generated in $SECRETS_FILE${NC}"
    echo -e "${YELLOW}⚠ IMPORTANT: Store the root token securely!${NC}"
}

# =============================================================================
# Network Setup
# =============================================================================

setup_network() {
    echo -e "${YELLOW}Setting up Podman network...${NC}"

    # Remove existing network if it exists
    podman network rm "$NETWORK_NAME" 2>/dev/null || true

    # Create new network
    podman network create "$NETWORK_NAME" \
        --driver bridge \
        --subnet 10.90.0.0/24 \
        --gateway 10.90.0.1 \
        --opt com.docker.network.bridge.name=golden-stack-br

    echo -e "${GREEN}✓ Network created: $NETWORK_NAME${NC}"
}

# =============================================================================
# Container Deployment
# =============================================================================

deploy_postgres() {
    echo -e "${YELLOW}Deploying PostgreSQL...${NC}"

    # Stop and remove existing container
    podman stop postgres-golden 2>/dev/null || true
    podman rm postgres-golden 2>/dev/null || true

    # Run PostgreSQL
    podman run -d \
        --name postgres-golden \
        --network "$NETWORK_NAME" \
        --hostname postgres-golden \
        -p "$POSTGRES_PORT:5432" \
        -e POSTGRES_USER=goldenstack \
        -e POSTGRES_PASSWORD="$(grep POSTGRES_PASSWORD "$DATA_DIR/.env" | cut -d= -f2)" \
        -e POSTGRES_DB=goldenstack \
        -e PGDATA=/var/lib/postgresql/data \
        -v "$DATA_DIR/postgres/data:/var/lib/postgresql/data:Z" \
        -v "$SCRIPT_DIR/../postgres/init:/docker-entrypoint-initdb.d:ro,Z" \
        --health-cmd="pg_isready -U goldenstack" \
        --health-interval=10s \
        --health-timeout=5s \
        --health-retries=5 \
        --restart=unless-stopped \
        "$POSTGRES_IMAGE"

    echo -e "${GREEN}✓ PostgreSQL deployed${NC}"

    # Wait for PostgreSQL to be ready
    echo -e "${YELLOW}Waiting for PostgreSQL to be ready...${NC}"
    sleep 5
    until podman exec postgres-golden pg_isready -U goldenstack; do
        echo -n "."
        sleep 2
    done
    echo -e "\n${GREEN}✓ PostgreSQL is ready${NC}"
}

deploy_infisical() {
    echo -e "${YELLOW}Deploying Infisical...${NC}"

    # Stop and remove existing container
    podman stop infisical-golden 2>/dev/null || true
    podman rm infisical-golden 2>/dev/null || true

    # Source environment variables
    set -a
    source "$DATA_DIR/.env"
    set +a

    # Run Infisical
    podman run -d \
        --name infisical-golden \
        --network "$NETWORK_NAME" \
        --hostname infisical-golden \
        -p "$INFISICAL_PORT:8080" \
        --env-file "$DATA_DIR/.env" \
        -v "$DATA_DIR/infisical:/infisical:Z" \
        --health-cmd="curl -f http://localhost:8080/api/status || exit 1" \
        --health-interval=30s \
        --health-timeout=10s \
        --health-retries=3 \
        --restart=unless-stopped \
        "$INFISICAL_IMAGE"

    echo -e "${GREEN}✓ Infisical deployed${NC}"

    # Wait for Infisical to be ready
    echo -e "${YELLOW}Waiting for Infisical to be ready...${NC}"
    sleep 10
    until curl -sf http://localhost:$INFISICAL_PORT/api/status > /dev/null 2>&1; do
        echo -n "."
        sleep 2
    done
    echo -e "\n${GREEN}✓ Infisical is ready${NC}"
}

deploy_openbao() {
    echo -e "${YELLOW}Deploying OpenBao...${NC}"

    # Build OpenBao image if needed
    if ! podman image exists openbao-golden:latest; then
        echo -e "${YELLOW}Building OpenBao image...${NC}"
        podman build -t openbao-golden:latest "$SCRIPT_DIR/../openbao"
    fi

    # Stop and remove existing container
    podman stop openbao-golden 2>/dev/null || true
    podman rm openbao-golden 2>/dev/null || true

    # Source environment variables
    set -a
    source "$DATA_DIR/.env"
    set +a

    # Run OpenBao
    podman run -d \
        --name openbao-golden \
        --network "$NETWORK_NAME" \
        --hostname openbao-golden \
        -p "$OPENBAO_PORT:8200" \
        --env-file "$DATA_DIR/.env" \
        -v "$DATA_DIR/openbao/data:/openbao/data:Z" \
        -v "$DATA_DIR/openbao/logs:/openbao/logs:Z" \
        --cap-add=IPC_LOCK \
        --health-cmd="/openbao/scripts/health-check.sh" \
        --health-interval=30s \
        --health-timeout=10s \
        --health-retries=3 \
        --restart=unless-stopped \
        openbao-golden:latest

    echo -e "${GREEN}✓ OpenBao deployed${NC}"

    # Wait for OpenBao to be ready
    echo -e "${YELLOW}Waiting for OpenBao to be ready...${NC}"
    sleep 15
    export VAULT_ADDR="http://localhost:$OPENBAO_PORT"
    until vault status > /dev/null 2>&1 || curl -sf "$VAULT_ADDR/v1/sys/health" > /dev/null 2>&1; do
        echo -n "."
        sleep 2
    done
    echo -e "\n${GREEN}✓ OpenBao is ready${NC}"
}

# =============================================================================
# Systemd Integration
# =============================================================================

setup_systemd() {
    if [[ "$ENVIRONMENT" != "production" ]]; then
        return 0
    fi

    echo -e "${YELLOW}Setting up systemd services...${NC}"

    # Create systemd service file
    cat > /etc/systemd/system/golden-stack.service << EOF
[Unit]
Description=Golden Stack (PostgreSQL + Infisical + OpenBao)
Documentation=https://github.com/TheArchitectit/rad-gateway
Requires=network.target
After=network.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=$DATA_DIR
ExecStart=$SCRIPT_DIR/deploy-golden-stack.sh $ENVIRONMENT
ExecStop=$SCRIPT_DIR/stop-golden-stack.sh
TimeoutStartSec=300

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable golden-stack.service

    echo -e "${GREEN}✓ Systemd service configured${NC}"
}

# =============================================================================
# Status Display
# =============================================================================

show_status() {
    echo ""
    echo -e "${GREEN}=== Golden Stack Status ===${NC}"
    echo ""

    echo "Containers:"
    podman ps --filter "name=-golden" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

    echo ""
    echo "Access URLs:"
    echo "  Infisical: http://localhost:$INFISICAL_PORT"
    echo "  OpenBao:   http://localhost:$OPENBAO_PORT"
    echo "  PostgreSQL: localhost:$POSTGRES_PORT"
    echo ""
    echo "Environment file: $DATA_DIR/.env"
    echo ""
    echo -e "${YELLOW}⚠ IMPORTANT: Store the root token securely!${NC}"
    echo -e "${YELLOW}⚠ Run: cat $DATA_DIR/.env | grep ROOT_TOKEN${NC}"
}

# =============================================================================
# Main
# =============================================================================

main() {
    echo "Golden Stack Deployment Script"
    echo "================================"
    echo ""

    check_prerequisites
    setup_directories
    generate_secrets
    setup_network
    deploy_postgres
    deploy_infisical
    deploy_openbao
    setup_systemd
    show_status

    echo ""
    echo -e "${GREEN}=== Deployment Complete ===${NC}"
}

main "$@"
