#!/bin/bash

# Script to reset demo data in PostgreSQL and ClickHouse containers
# Usage: ./reset_demo_data.sh

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Container names from docker-compose.yml
POSTGRES_CONTAINER="wallet-postgres"
CLICKHOUSE_CONTAINER="wallet-clickhouse"

# Database credentials
POSTGRES_USER="bank_service"
POSTGRES_PASSWORD="bank_service"
POSTGRES_DB="bank_db"

CLICKHOUSE_USER="analytics_service"
CLICKHOUSE_PASSWORD="analytics_service"
CLICKHOUSE_DB="analytics_db"

echo -e "${BLUE}=== Demo Data Reset Script ===${NC}"

# Check if containers are running
echo -e "${BLUE}Checking if containers are running...${NC}"

if ! docker ps --format '{{.Names}}' | grep -q "^${POSTGRES_CONTAINER}$"; then
    echo -e "${RED}Error: PostgreSQL container '${POSTGRES_CONTAINER}' is not running${NC}"
    echo "Please start it with: docker-compose up -d postgres"
    exit 1
fi

if ! docker ps --format '{{.Names}}' | grep -q "^${CLICKHOUSE_CONTAINER}$"; then
    echo -e "${RED}Error: ClickHouse container '${CLICKHOUSE_CONTAINER}' is not running${NC}"
    echo "Please start it with: docker-compose up -d clickhouse"
    exit 1
fi

echo -e "${GREEN}✓ All containers are running${NC}"

# Execute PostgreSQL queries
echo -e "\n${BLUE}Resetting PostgreSQL account balances...${NC}"

if [ -f "${SCRIPT_DIR}/account_states.sql" ]; then
    docker exec -i "${POSTGRES_CONTAINER}" psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" < "${SCRIPT_DIR}/account_states.sql"
    echo -e "${GREEN}✓ PostgreSQL account balances reset successfully${NC}"
else
    echo -e "${RED}Warning: account_states.sql not found${NC}"
fi

# Execute ClickHouse queries
echo -e "\n${BLUE}Truncating ClickHouse operations table...${NC}"

if [ -f "${SCRIPT_DIR}/remove_operations.sql" ]; then
    docker exec -i "${CLICKHOUSE_CONTAINER}" clickhouse-client \
        --host=localhost \
        --port=9000 \
        --user="${CLICKHOUSE_USER}" \
        --password="${CLICKHOUSE_PASSWORD}" \
        --database="${CLICKHOUSE_DB}" \
        --multiquery < "${SCRIPT_DIR}/remove_operations.sql"
    echo -e "${GREEN}✓ ClickHouse operations table truncated successfully${NC}"
else
    echo -e "${RED}Warning: remove_operations.sql not found${NC}"
fi

# Verify the changes
echo -e "\n${BLUE}=== Verification ===${NC}"

echo -e "\n${BLUE}PostgreSQL account balances:${NC}"
docker exec -i "${POSTGRES_CONTAINER}" psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -c \
    "SELECT id, balance_value FROM accounts ORDER BY balance_value DESC;"

echo -e "\n${BLUE}ClickHouse operations count:${NC}"
docker exec -i "${CLICKHOUSE_CONTAINER}" clickhouse-client \
    --host=localhost \
    --port=9000 \
    --user="${CLICKHOUSE_USER}" \
    --password="${CLICKHOUSE_PASSWORD}" \
    --database="${CLICKHOUSE_DB}" \
    --query="SELECT COUNT(*) as total_operations FROM operations;"

echo -e "\n${GREEN}=== Demo data reset completed successfully ===${NC}"
