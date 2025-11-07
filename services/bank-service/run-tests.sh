#!/bin/bash

# Test Runner Script for Bank Service
# This script helps run different types of tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Get test type from first argument, default to 'unit'
TEST_TYPE="${1:-unit}"

echo -e "${CYAN}Bank Service Test Runner${NC}"
echo -e "${CYAN}========================${NC}"
echo ""

case "$TEST_TYPE" in
    unit)
        echo -e "${GREEN}Running unit tests (no Docker required)...${NC}"
        go test -v -short ./... -timeout 2m
        ;;
    
    integration)
        echo -e "${YELLOW}Running integration tests (requires Docker)...${NC}"
        echo -e "${YELLOW}Make sure Docker Desktop is running!${NC}"
        echo ""
        
        # Check if Docker is running
        if ! docker info > /dev/null 2>&1; then
            echo -e "${RED}ERROR: Docker is not running!${NC}"
            echo -e "${RED}Please start Docker Desktop and try again.${NC}"
            exit 1
        fi
        
        echo -e "${GREEN}Docker is running ✓${NC}"
        echo ""
        
        go test -v ./internal/grpc/... -run TestTransferMoneyIntegration -timeout 10m
        ;;
    
    all)
        echo -e "${YELLOW}Running all tests (requires Docker)...${NC}"
        echo -e "${YELLOW}Make sure Docker Desktop is running!${NC}"
        echo ""
        
        # Check if Docker is running
        if ! docker info > /dev/null 2>&1; then
            echo -e "${RED}ERROR: Docker is not running!${NC}"
            echo -e "${RED}Please start Docker Desktop and try again.${NC}"
            exit 1
        fi
        
        echo -e "${GREEN}Docker is running ✓${NC}"
        echo ""
        
        go test -v ./... -timeout 10m
        ;;
    
    coverage)
        echo -e "${GREEN}Running tests with coverage report...${NC}"
        go test -v -short -coverprofile=coverage.out ./...
        echo ""
        echo -e "${CYAN}Coverage summary:${NC}"
        go tool cover -func=coverage.out
        echo ""
        echo -e "${YELLOW}To view HTML coverage report, run:${NC}"
        echo -e "${YELLOW}  go tool cover -html=coverage.out${NC}"
        ;;
    
    *)
        echo -e "${RED}Invalid test type: $TEST_TYPE${NC}"
        echo ""
        echo "Usage: ./run-tests.sh [unit|integration|all|coverage]"
        echo ""
        echo "  unit        - Run unit tests only (no Docker required)"
        echo "  integration - Run integration tests (requires Docker)"
        echo "  all         - Run all tests (requires Docker)"
        echo "  coverage    - Run tests with coverage report"
        echo ""
        exit 1
        ;;
esac

echo ""
if [ $? -eq 0 ]; then
    echo -e "${GREEN}Tests passed ✓${NC}"
else
    echo -e "${RED}Tests failed ✗${NC}"
    exit 1
fi
