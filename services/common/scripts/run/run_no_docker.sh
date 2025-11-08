#!/bin/bash

# Local Development Runner for Manual Testing
# Starts API Gateway, Bank Service, and RabbitMQ for manual system testing
# Prerequisites: PostgreSQL running on port 5433

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
DB_PORT=5433
DB_NAME=bank_db
DB_USER=bank_service
DB_PASSWORD=bank_service
RABBITMQ_PORT=5672
RABBITMQ_MGMT_PORT=15672
BANK_SERVICE_PORT=50051
API_GATEWAY_PORT=8080

# Cleanup function
cleanup() {
    echo ""
    echo -e "${YELLOW}Shutting down services...${NC}"
    
    # Kill background processes
    jobs -p | xargs -r kill 2>/dev/null || true
    
    # Stop RabbitMQ container if it exists
    docker rm -f dev-rabbitmq 2>/dev/null || true
    
    echo -e "${GREEN}Cleanup complete${NC}"
    exit 0
}

# Set up cleanup on script exit
trap cleanup SIGINT SIGTERM EXIT

echo -e "${CYAN}================================${NC}"
echo -e "${CYAN}  Local Development Environment${NC}"
echo -e "${CYAN}================================${NC}"
echo ""

# Get script directory and calculate service paths
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# Script is in: services/common/scripts/run/
# Navigate up to services directory: ../../../
SERVICES_DIR="$( cd "${SCRIPT_DIR}/../../../" && pwd )"
BANK_SERVICE_DIR="${SERVICES_DIR}/bank-service"
API_GATEWAY_DIR="${SERVICES_DIR}/api-gateway"

echo -e "${CYAN}Service directories:${NC}"
echo -e "  Bank Service:  ${BANK_SERVICE_DIR}"
echo -e "  API Gateway:   ${API_GATEWAY_DIR}"
echo ""

# Check PostgreSQL
echo -e "${YELLOW}Checking PostgreSQL on port ${DB_PORT}...${NC}"

# Try multiple methods to check if port is open
check_port() {
    # Method 1: Use timeout with bash TCP
    if timeout 1 bash -c "echo > /dev/tcp/localhost/${DB_PORT}" 2>/dev/null; then
        return 0
    fi
    
    # Method 2: Use netcat if available
    if command -v nc >/dev/null 2>&1; then
        if nc -z localhost ${DB_PORT} 2>/dev/null; then
            return 0
        fi
    fi
    
    # Method 3: Use telnet if available
    if command -v telnet >/dev/null 2>&1; then
        if timeout 1 telnet localhost ${DB_PORT} 2>/dev/null | grep -q "Connected"; then
            return 0
        fi
    fi
    
    # Method 4: Try to connect with psql
    if command -v psql >/dev/null 2>&1; then
        if PGPASSWORD=${DB_PASSWORD} psql -h localhost -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME} -c "SELECT 1" >/dev/null 2>&1; then
            return 0
        fi
    fi
    
    return 1
}

if ! check_port; then
    echo -e "${RED}ERROR: Cannot connect to PostgreSQL on port ${DB_PORT}${NC}"
    echo -e "${RED}Please verify:${NC}"
    echo -e "${RED}  1. PostgreSQL is running${NC}"
    echo -e "${RED}  2. Port ${DB_PORT} is correct${NC}"
    echo -e "${RED}  3. PostgreSQL accepts connections from localhost${NC}"
    echo ""
    echo -e "${YELLOW}You can skip this check by commenting out the PostgreSQL check section${NC}"
    exit 1
fi
echo -e "${GREEN}PostgreSQL is running ✓${NC}"
echo ""

# Check Docker
echo -e "${YELLOW}Checking Docker...${NC}"
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}ERROR: Docker is not running${NC}"
    echo -e "${RED}Please start Docker Desktop and try again.${NC}"
    exit 1
fi
echo -e "${GREEN}Docker is running ✓${NC}"
echo ""

# Start RabbitMQ
echo -e "${YELLOW}Starting RabbitMQ...${NC}"
docker rm -f dev-rabbitmq 2>/dev/null || true
docker run -d \
    --name dev-rabbitmq \
    -p ${RABBITMQ_PORT}:5672 \
    -p ${RABBITMQ_MGMT_PORT}:15672 \
    -e RABBITMQ_DEFAULT_USER=guest \
    -e RABBITMQ_DEFAULT_PASS=guest \
    rabbitmq:3-management > /dev/null

echo -e "${GREEN}RabbitMQ started on ports ${RABBITMQ_PORT} (AMQP) and ${RABBITMQ_MGMT_PORT} (Management) ✓${NC}"
echo -e "${CYAN}RabbitMQ Management UI: http://localhost:${RABBITMQ_MGMT_PORT} (guest/guest)${NC}"
echo ""

# Wait for RabbitMQ to be ready
echo -e "${YELLOW}Waiting for RabbitMQ to be ready...${NC}"
sleep 5
until docker exec dev-rabbitmq rabbitmqctl status > /dev/null 2>&1; do
    echo -n "."
    sleep 1
done
echo ""
echo -e "${GREEN}RabbitMQ is ready ✓${NC}"
echo ""

# Setup RabbitMQ queue for testing
echo -e "${YELLOW}Setting up RabbitMQ queue for event monitoring...${NC}"
docker exec dev-rabbitmq rabbitmqadmin declare exchange name=bank.operations type=topic durable=true 2>/dev/null || true
docker exec dev-rabbitmq rabbitmqadmin declare queue name=test-events-queue durable=false auto_delete=false 2>/dev/null || true
docker exec dev-rabbitmq rabbitmqadmin declare binding source=bank.operations destination=test-events-queue routing_key="bank.operations.#" 2>/dev/null || true
echo -e "${GREEN}Queue 'test-events-queue' created and bound to 'bank.operations' exchange ✓${NC}"
echo ""

# Run Bank Service migrations
echo -e "${YELLOW}Running Bank Service migrations...${NC}"
cd "${BANK_SERVICE_DIR}"
export DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@localhost:${DB_PORT}/${DB_NAME}?sslmode=disable"
./run-migrations.sh up > /dev/null 2>&1 || true
echo -e "${GREEN}Migrations completed ✓${NC}"
echo ""

# Build Bank Service
echo -e "${YELLOW}Building Bank Service...${NC}"
cd "${BANK_SERVICE_DIR}"
mkdir -p bin logs
go build -o bin/bank-service ./cmd/server/main.go
echo -e "${GREEN}Bank Service built ✓${NC}"
echo ""

# Build API Gateway
echo -e "${YELLOW}Building API Gateway...${NC}"
cd "${API_GATEWAY_DIR}"
mkdir -p bin logs
go build -o bin/api-gateway ./cmd/server/main.go
echo -e "${GREEN}API Gateway built ✓${NC}"
echo ""

# Start Bank Service
echo -e "${YELLOW}Starting Bank Service on port ${BANK_SERVICE_PORT}...${NC}"
cd "${BANK_SERVICE_DIR}"
export DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@localhost:${DB_PORT}/${DB_NAME}?sslmode=disable"
export RABBITMQ_URL="amqp://guest:guest@localhost:${RABBITMQ_PORT}/"
export PORT="${BANK_SERVICE_PORT}"
./bin/bank-service > logs/bank-service.log 2>&1 &
BANK_PID=$!
echo -e "${GREEN}Bank Service started (PID: ${BANK_PID}) ✓${NC}"
echo -e "${CYAN}Logs: ${BANK_SERVICE_DIR}/logs/bank-service.log${NC}"
echo ""

# Wait for Bank Service to be ready
echo -e "${YELLOW}Waiting for Bank Service to be ready...${NC}"
sleep 2
if ! kill -0 ${BANK_PID} 2>/dev/null; then
    echo -e "${RED}ERROR: Bank Service failed to start${NC}"
    echo -e "${RED}Check logs at: ${BANK_SERVICE_DIR}/logs/bank-service.log${NC}"
    exit 1
fi
echo -e "${GREEN}Bank Service is ready ✓${NC}"
echo ""

# Start API Gateway
echo -e "${YELLOW}Starting API Gateway on port ${API_GATEWAY_PORT}...${NC}"
cd "${API_GATEWAY_DIR}"
export BANK_SERVICE_ADDRESS="localhost:${BANK_SERVICE_PORT}"
export PORT="${API_GATEWAY_PORT}"
./bin/api-gateway > logs/api-gateway.log 2>&1 &
API_PID=$!
echo -e "${GREEN}API Gateway started (PID: ${API_PID}) ✓${NC}"
echo -e "${CYAN}Logs: ${API_GATEWAY_DIR}/logs/api-gateway.log${NC}"
echo ""

# Wait for API Gateway to be ready
echo -e "${YELLOW}Waiting for API Gateway to be ready...${NC}"
sleep 2
if ! kill -0 ${API_PID} 2>/dev/null; then
    echo -e "${RED}ERROR: API Gateway failed to start${NC}"
    echo -e "${RED}Check logs at: ${API_GATEWAY_DIR}/logs/api-gateway.log${NC}"
    exit 1
fi
echo -e "${GREEN}API Gateway is ready ✓${NC}"
echo ""

# Display status
echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}  All services are running!${NC}"
echo -e "${GREEN}================================${NC}"
echo ""
echo -e "${CYAN}Service Endpoints:${NC}"
echo -e "  API Gateway (HTTP):     http://localhost:${API_GATEWAY_PORT}"
echo -e "  Bank Service (gRPC):    localhost:${BANK_SERVICE_PORT}"
echo -e "  RabbitMQ (AMQP):        localhost:${RABBITMQ_PORT}"
echo -e "${CYAN}RabbitMQ Management:    http://localhost:${RABBITMQ_MGMT_PORT}"
echo -e "  PostgreSQL:             localhost:${DB_PORT}"
echo ""
echo -e "${CYAN}RabbitMQ Queue:${NC}"
echo -e "  Queue Name:             test-events-queue"
echo -e "  Exchange:               bank.operations (topic)"
echo -e "  Routing Key Pattern:    bank.operations.#"
echo ""
echo -e "${CYAN}Process IDs:${NC}"
echo -e "  Bank Service:           ${BANK_PID}"
echo -e "  API Gateway:            ${API_PID}"
echo ""
echo -e "${CYAN}Log Files:${NC}"
echo -e "  Bank Service:           ${BANK_SERVICE_DIR}/logs/bank-service.log"
echo -e "  API Gateway:            ${API_GATEWAY_DIR}/logs/api-gateway.log"
echo ""
echo -e "${CYAN}Test Account IDs:${NC}"
echo -e "  Account 1:              11111111-1111-1111-1111-111111111111 (1000.00 RUB)"
echo -e "  Account 2:              22222222-2222-2222-2222-222222222222 (500.00 RUB)"
echo ""
echo -e "${YELLOW}Example API Calls:${NC}"
echo ""
echo -e "${CYAN}# Transfer money${NC}"
echo -e 'curl -X POST http://localhost:8080/accounts/11111111-1111-1111-1111-111111111111/transfers \\'
echo -e '  -H "Content-Type: application/json" \\'
echo -e '  -d '"'"'{'
echo -e '    "recipient_id": "22222222-2222-2222-2222-222222222222",'
echo -e '    "amount": {"value": "50.00", "currency_code": "RUB"},'
echo -e '    "idempotency_key": "test-transfer-1"'
echo -e '  }'"'"
echo ""
echo -e "${CYAN}# Check account balance${NC}"
echo -e 'curl http://localhost:8080/accounts/11111111-1111-1111-1111-111111111111'
echo ""
echo -e "${CYAN}# Monitor RabbitMQ events (in another terminal)${NC}"
echo -e "docker exec dev-rabbitmq rabbitmqadmin get queue=test-events-queue ackmode=ack_requeue_false"
echo ""
echo -e "${CYAN}# Or view queue stats${NC}"
echo -e "docker exec dev-rabbitmq rabbitmqadmin list queues name messages"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop all services${NC}"
echo ""

# Wait for interrupt
wait
