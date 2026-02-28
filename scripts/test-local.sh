#!/bin/bash
# Test script for local RAD Gateway with Ollama
# Usage: ./scripts/test-local.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== RAD Gateway Local Testing ===${NC}"
echo

# Check if .env.testing exists
if [ ! -f .env.testing ]; then
    echo -e "${RED}Error: .env.testing not found${NC}"
    exit 1
fi

# Check if Ollama is running
echo -e "${YELLOW}Checking Ollama...${NC}"
if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo -e "${RED}Error: Ollama is not running on localhost:11434${NC}"
    echo "Please start Ollama first:"
    echo "  ollama serve"
    exit 1
fi
echo -e "${GREEN}Ollama is running${NC}"

# Check if llama3.2 is available
echo -e "${YELLOW}Checking for llama3.2 model...${NC}"
if ! curl -s http://localhost:11434/api/tags | grep -q "llama3.2"; then
    echo -e "${YELLOW}llama3.2 not found. Pulling...${NC}"
    ollama pull llama3.2:latest
fi
echo -e "${GREEN}llama3.2 is available${NC}"
echo

# Copy test environment
echo -e "${YELLOW}Setting up test environment...${NC}"
cp .env.testing .env
echo -e "${GREEN}Environment configured${NC}"
echo

# Check if gateway is already running
echo -e "${YELLOW}Checking RAD Gateway...${NC}"
if curl -s http://localhost:8090/health > /dev/null 2>&1; then
    echo -e "${GREEN}RAD Gateway is already running${NC}"
else
    echo -e "${YELLOW}Starting RAD Gateway...${NC}"
    echo "Run: go run ./cmd/rad-gateway"
    echo "Or:  ./rad-gateway (if built)"
    echo
    echo -e "${RED}Please start the gateway in another terminal${NC}"
    exit 1
fi
echo

# Test API
echo -e "${GREEN}=== Testing API ===${NC}"
echo

API_KEY="test_key_for_local_testing_only_001"
BASE_URL="http://localhost:8090"

echo -e "${YELLOW}1. Health Check${NC}"
curl -s "${BASE_URL}/health" | jq . 2>/dev/null || curl -s "${BASE_URL}/health"
echo

echo -e "${YELLOW}2. List Models${NC}"
curl -s -H "Authorization: Bearer ${API_KEY}" \
    "${BASE_URL}/v1/models" | jq . 2>/dev/null || curl -s -H "Authorization: Bearer ${API_KEY}" "${BASE_URL}/v1/models"
echo

echo -e "${YELLOW}3. Chat Completion (Non-streaming)${NC}"
curl -s -X POST \
    -H "Authorization: Bearer ${API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "llama3.2",
        "messages": [{"role": "user", "content": "Say hello in one word"}],
        "max_tokens": 50
    }' \
    "${BASE_URL}/v1/chat/completions" | jq . 2>/dev/null || echo "Request sent"
echo

echo -e "${YELLOW}4. Chat Completion (Streaming)${NC}"
echo "Testing streaming response..."
curl -s -X POST \
    -H "Authorization: Bearer ${API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "llama3.2",
        "messages": [{"role": "user", "content": "Count to 3"}],
        "stream": true,
        "max_tokens": 50
    }' \
    "${BASE_URL}/v1/chat/completions" | head -20
echo

echo -e "${GREEN}=== Tests Complete ===${NC}"
echo
echo "Additional test commands:"
echo "  # Test with different API keys:"
echo "  curl -H \"Authorization: Bearer dev_key_for_local_testing_only_002\" ${BASE_URL}/v1/models"
echo
echo "  # Test admin endpoints (get JWT first):"
echo "  curl -X POST ${BASE_URL}/v1/auth/login -H \"Content-Type: application/json\" -d '{\"username\":\"admin\",\"password\":\"admin\"}'"
echo
