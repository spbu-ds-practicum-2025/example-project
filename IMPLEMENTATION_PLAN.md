# Implementation Plan: Electronic Wallet Distributed System

## Overview
This document outlines the detailed implementation plan for the Electronic Wallet system, a distributed application built with FastAPI, gRPC, PostgreSQL, ClickHouse, and Docker Compose.

## Technology Stack

### Core Technologies
- **Framework**: FastAPI (all services)
- **API Protocol**: 
  - gRPC for inter-service communication (Bank, Analytics, BankCardAdapter)
  - REST/HTTP for external API (API Gateway)
- **Databases**:
  - PostgreSQL (Bank Service - transactional)
  - ClickHouse (Analytics Service - analytical)
- **Message Broker**: Apache Kafka
- **Orchestration**: Docker Compose
- **Approach**: Contract-first (OpenAPI, Protobuf)

### Python Libraries
- **FastAPI**: Web framework
- **grpcio**: gRPC runtime
- **grpcio-tools**: Protocol Buffers compiler
- **sqlalchemy**: ORM for PostgreSQL
- **asyncpg**: Async PostgreSQL driver
- **clickhouse-driver**: ClickHouse client
- **aiokafka**: Async Kafka client
- **pydantic**: Data validation
- **uvicorn**: ASGI server
- **pydantic-avro**: Avro schema generation from Pydantic models (for AsyncAPI/Kafka)

---

## Project Structure

```
services/
├── common/                              # Shared contracts and specifications
│   ├── wallet-api/                      # REST API specification (OpenAPI)
│   │   └── openapi.yaml
│   ├── bank-service-api/                # Bank gRPC API
│   │   └── bank_service.proto
│   ├── bank-card-adapter-api/           # BankCardAdapter gRPC API
│   │   └── bank_card_adapter.proto
│   ├── analytics-service-api/           # Analytics gRPC API
│   │   └── analytics_service.proto
│   └── analytics-service-kafka-spec/    # Kafka event schemas (AsyncAPI)
│       ├── asyncapi.yaml                # AsyncAPI specification
│       └── schemas/                     # Event schemas
│           ├── transfer_event.json      # JSON Schema for TransferEvent
│           └── topup_event.json         # JSON Schema for TopUpEvent
│
├── api-gateway/                         # API Gateway service
│   ├── app/
│   │   ├── __init__.py
│   │   ├── main.py                      # FastAPI application
│   │   ├── api/                         # Generated from OpenAPI
│   │   │   ├── __init__.py
│   │   │   ├── models.py
│   │   │   └── routes.py
│   │   ├── clients/                     # gRPC clients
│   │   │   ├── __init__.py
│   │   │   ├── bank_client.py
│   │   │   ├── bank_card_client.py
│   │   │   └── analytics_client.py
│   │   ├── middleware/
│   │   │   ├── __init__.py
│   │   │   └── auth.py
│   │   └── config.py
│   ├── requirements.txt
│   ├── Dockerfile
│   └── pyproject.toml
│
├── bank-service/                        # Bank service
│   ├── app/
│   │   ├── __init__.py
│   │   ├── main.py                      # FastAPI + gRPC server
│   │   ├── grpc_server/                 # gRPC implementation
│   │   │   ├── __init__.py
│   │   │   ├── server.py
│   │   │   └── servicer.py
│   │   ├── domain/                      # Business logic
│   │   │   ├── __init__.py
│   │   │   ├── models.py
│   │   │   ├── repositories.py
│   │   │   └── services.py
│   │   ├── db/                          # Database layer
│   │   │   ├── __init__.py
│   │   │   ├── models.py                # SQLAlchemy models
│   │   │   ├── migrations/              # Alembic migrations
│   │   │   └── session.py
│   │   ├── kafka/                       # Kafka producer
│   │   │   ├── __init__.py
│   │   │   └── producer.py
│   │   └── config.py
│   ├── proto/                           # Generated gRPC code
│   │   └── bank_service_pb2.py
│   ├── requirements.txt
│   ├── Dockerfile
│   └── alembic.ini
│
├── analytics-service/                   # Analytics service
│   ├── app/
│   │   ├── __init__.py
│   │   ├── main.py                      # FastAPI + gRPC server
│   │   ├── grpc_server/                 # gRPC implementation
│   │   │   ├── __init__.py
│   │   │   ├── server.py
│   │   │   └── servicer.py
│   │   ├── domain/                      # Business logic
│   │   │   ├── __init__.py
│   │   │   ├── models.py
│   │   │   ├── repositories.py
│   │   │   └── services.py
│   │   ├── db/                          # ClickHouse layer
│   │   │   ├── __init__.py
│   │   │   ├── schema.sql
│   │   │   └── client.py
│   │   ├── kafka/                       # Kafka consumer
│   │   │   ├── __init__.py
│   │   │   └── consumer.py
│   │   └── config.py
│   ├── proto/                           # Generated gRPC code
│   │   └── analytics_service_pb2.py
│   ├── requirements.txt
│   └── Dockerfile
│
├── bank-card-adapter/                   # Bank Card Adapter service
│   ├── app/
│   │   ├── __init__.py
│   │   ├── main.py                      # FastAPI + gRPC server
│   │   ├── grpc_server/                 # gRPC implementation
│   │   │   ├── __init__.py
│   │   │   ├── server.py
│   │   │   └── servicer.py
│   │   ├── domain/                      # Business logic
│   │   │   ├── __init__.py
│   │   │   └── payment_service.py
│   │   ├── clients/                     # External clients
│   │   │   ├── __init__.py
│   │   │   ├── bank_client.py           # gRPC client to Bank
│   │   │   └── payment_gateway.py       # Mock external payment
│   │   └── config.py
│   ├── proto/                           # Generated gRPC code
│   │   └── bank_card_adapter_pb2.py
│   ├── requirements.txt
│   └── Dockerfile
```

---

## Phase 1: Core Transfer Functionality (MVP - Weeks 1-4)

**Focus**: Implement basic transfer operations between accounts using only API Gateway and Bank Service. No authentication, no analytics, no top-ups.

### 1.1 Define gRPC Contracts (Week 1)

**Task 1.1.1: Bank Service Proto Definition**
- **File**: `services/common/bank-service-api/bank_service.proto`
- **Services to define** (MVP scope):
  - `TransferMoney`: Handle money transfers between accounts
  - `GetBalance`: Retrieve current account balance (for validation/testing)
- **Messages**: TransferRequest, TransferResponse, BalanceRequest, BalanceResponse, etc.
- **Error handling**: Define error codes and status messages (insufficient funds, account not found)
- **Note**: No CreateAccount RPC - accounts will be created manually in database for testing
- **Note**: Skip DepositMoney for now (will be added in Phase 2)

### 1.2 Code Generation Setup (Week 1)

**Task 1.2.1: Create Proto Compilation Script**
- **File**: `scripts/generate_protos.ps1` (for Windows PowerShell)
- Use `grpcio-tools` to generate Python code from `.proto` files
- Output to each service's `proto/` directory
- Focus on bank-service proto only for Phase 1

**Task 1.2.2: OpenAPI Code Generation for API Gateway**
- Tool: `datamodel-code-generator` or `openapi-generator`
- Generate Pydantic models from `wallet-api/openapi.yaml`
- Focus only on transfer endpoint for Phase 1
- Generate route stubs for FastAPI

**Task 1.2.3: Setup Common Dependencies**
- Create basic `requirements.txt` files for Bank Service and API Gateway
- Version pinning strategy

**Note**: AsyncAPI and other proto files will be added in later phases

### 1.3 Local Development Setup (Week 1)

**Task 1.3.1: PostgreSQL Setup**
- **Local installation** or **Docker container** for PostgreSQL only
- Database initialization script: `docker/postgres/init.sql` or manual setup
- Create `bank_db` database
- Manually create test accounts table and insert 2 test accounts
- Basic schema setup will be handled by Alembic migrations

**Example test data**:
```sql
INSERT INTO accounts (id, balance, created_at, updated_at) VALUES
  ('123e4567-e89b-12d3-a456-426614174000', 1000.00, NOW(), NOW()),
  ('987e6543-e21b-34d3-c456-426614174999', 500.00, NOW(), NOW());
```

**Task 1.3.2: Development Environment**
- Python 3.11+ virtual environments for each service
- Install dependencies locally for development
- Setup IDE/editor for Python development
- Configure environment variables (.env files)

**Task 1.3.3: Test Data Setup**
- Create SQL script for inserting test accounts: `docker/postgres/test_data.sql`
- Suggested test accounts:
  ```sql
  -- Account 1: Alice (1000 RUB initial balance)
  INSERT INTO accounts (id, balance, created_at, updated_at) VALUES
    ('123e4567-e89b-12d3-a456-426614174000', 1000.00, NOW(), NOW());
  
  -- Account 2: Bob (500 RUB initial balance)
  INSERT INTO accounts (id, balance, created_at, updated_at) VALUES
    ('987e6543-e21b-34d3-c456-426614174999', 500.00, NOW(), NOW());
  ```
- Run this script after Alembic migrations to populate test data

**Note**: Full docker-compose with all services will be added in Phase 7

---

## Phase 2: Bank Service Implementation (Weeks 2-3)

### 2.1 Bank Service (Week 2-3)

**Task 2.1.1: Database Layer**
- **SQLAlchemy Models** (`app/db/models.py`):
  - `Account`: id (UUID), balance (Decimal), created_at, updated_at, version (for optimistic locking)
  - `Transfer`: id (UUID), sender_id, recipient_id, amount, status, timestamp, idempotency_key
- **Alembic Setup**: Initialize migrations
- **Session Management**: Async session factory
- **Note**: Skip `Operation` table for now (will be added with Analytics in Phase 3)

**Task 2.1.2: Domain Logic**
- **Repository Layer** (`app/domain/repositories.py`):
  - `AccountRepository`: Read operations only (get by id, check existence)
  - `TransferRepository`: Transfer management with transactions
- **Service Layer** (`app/domain/services.py`):
  - `TransferService`: Business logic for transfers
    - Validate sufficient balance
    - Validate accounts exist
    - Ensure atomicity (both debit and credit in same transaction)
    - Handle idempotency
  - `AccountService`: Get balance (read-only)
- **Note**: No account creation logic - accounts are pre-populated in database

**Task 2.1.3: gRPC Server Implementation**
- **Servicer** (`app/grpc_server/servicer.py`):
  - Implement `TransferMoney` RPC
  - Implement `GetBalance` RPC (for testing/validation)
  - Error handling with gRPC status codes
  - Input validation
- **Server Setup** (`app/grpc_server/server.py`):
  - gRPC server configuration
  - Concurrent request handling
- **Note**: No CreateAccount RPC implementation

**Task 2.1.4: Application Entry Point**
- **Main** (`app/main.py`):
  - Start gRPC server
  - Initialize database connections
  - Health check endpoint (optional FastAPI endpoint)
- **Note**: Skip Kafka producer for Phase 1

**Task 2.1.5: Configuration**
- Environment variables for:
  - Database URL
  - gRPC port
  - Service discovery settings (if needed)
- **Note**: No Kafka configuration needed yet

---

## Phase 3: API Gateway Implementation (Week 4)

### 3.1 API Gateway (Week 4)

**Task 3.1.1: Generated API Models**
- Use generated Pydantic models from OpenAPI spec
- Focus only on transfer endpoint for Phase 1
- Custom validators if needed

**Task 3.1.2: gRPC Client for Bank Service**
- **Bank Client** (`app/clients/bank_client.py`):
  - Async gRPC client to Bank Service
  - Methods: transfer, get_balance
- **Note**: Skip Analytics and BankCard clients for Phase 1
- **Note**: No create_account method needed

**Task 3.1.3: API Routes Implementation**
- **Routes** (`app/api/routes.py`):
  - `POST /accounts/{accountId}/transfers`: Call Bank Service
  - **Skip for Phase 1**: 
    - `GET /accounts/{accountId}/operations` (requires Analytics)
    - `POST /accounts/{accountId}/topup` (requires BankCardAdapter)
- **Request/Response mapping**: Convert between REST and gRPC formats
- **Error handling**: Map gRPC errors to HTTP status codes (400, 404, etc.)

**Task 3.1.4: Middleware**
- **Request logging**: Basic request/response logging
- **Idempotency**: Store and check idempotency keys (in-memory cache or database)
- **Note**: Skip authentication for Phase 1 - no auth middleware needed

**Task 3.1.5: Application Entry Point**
- **Main** (`app/main.py`):
  - FastAPI application setup
  - Register transfer route only
  - CORS configuration (permissive for development)
  - Initialize gRPC client to Bank Service

---

## Phase 4: Testing and Refinement (Week 4)

### 4.1 Unit Tests (Week 4)

**For Bank Service**:
- **Domain logic tests**: Transfer business rules, edge cases
- **Repository tests**: Database operations (use testcontainers-postgres)
- **Service tests**: Mock dependencies

**For API Gateway**:
- **Route tests**: Mock gRPC client responses
- **Error handling tests**: Validate error mapping

**Testing framework**: pytest + pytest-asyncio

### 4.2 Integration Tests (Week 4)

**Test scenarios**:
- **Bank Service ↔ PostgreSQL**: Transaction integrity, rollback scenarios
- **API Gateway → Bank Service**: End-to-end gRPC communication
- **Complete transfer flow**: REST request → API Gateway → Bank Service → PostgreSQL

**Tools**: pytest, testcontainers, httpx

### 4.3 API Tests (Week 4)

**API Gateway tests**:
- Test transfer endpoint
- Validate request/response schemas against OpenAPI spec
- Test error cases (400, 404 - insufficient funds, account not found)
- Test idempotency (same key returns same result)

**Tools**: pytest + httpx

### 4.4 Edge Cases (Week 4)

**Test cases**:
- Insufficient balance for transfer
- Non-existent sender or recipient account
- Duplicate idempotency key
- Negative transfer amounts
- Transfer to same account
- Concurrent transfers from same account
- Database connection failures
- gRPC connection failures

---

## Phase 5: Analytics Integration (Weeks 5-6)

**Focus**: Add operation history and analytics capabilities

### 5.1 Kafka Infrastructure (Week 5)

**Task 5.1.1: Kafka Setup**
- Add Kafka + Zookeeper to docker-compose or use local installation
- Create `wallet.operations` topic
- Test basic producer/consumer

**Task 5.1.2: AsyncAPI Specification**
- Define `asyncapi.yaml` for transfer events
- Create `transfer_event.json` schema
- Validate specification

### 5.2 Bank Service Kafka Integration (Week 5)

**Task 5.2.1: Add Kafka Producer**
- Implement Kafka producer in Bank Service
- Publish `TransferEvent` after successful transfers
- JSON serialization with Pydantic models
- Handle producer errors gracefully

**Task 5.2.2: Update Transfer Flow**
- Modify transfer logic to publish events
- Ensure event publishing doesn't affect transaction success
- Add configuration for Kafka broker

### 5.3 Analytics Service Implementation (Week 5-6)

**Task 5.3.1: ClickHouse Setup**
- Add ClickHouse to docker-compose or use local installation
- Create `operations` table schema
- Test basic queries

**Task 5.3.2: Implement Analytics Service**
- Kafka consumer for transfer events
- Insert events into ClickHouse
- gRPC server for GetOperations
- Pagination and filtering logic

**Task 5.3.3: Define Analytics Proto**
- Create `analytics_service.proto`
- Generate Python code
- Implement servicer

### 5.4 API Gateway Analytics Integration (Week 6)

**Task 5.4.1: Add Analytics Client**
- Implement gRPC client for Analytics Service
- Add to API Gateway

**Task 5.4.2: Implement Operations Endpoint**
- `GET /accounts/{accountId}/operations`
- Map gRPC response to OpenAPI format
- Test end-to-end flow

### 5.5 Testing (Week 6)

- Test event publishing from Bank to Kafka
- Test event consumption by Analytics
- Test operations retrieval endpoint
- End-to-end test: Transfer → Event → Analytics → API response

---

## Phase 6: Top-Up Functionality (Weeks 7-8)

**Focus**: Add account top-up via bank cards

### 6.1 Bank Card Adapter Service (Week 7)

**Task 6.1.1: Define Proto**
- Create `bank_card_adapter.proto`
- Define `ProcessCardTopUp` RPC
- Generate Python code

**Task 6.1.2: External Payment Gateway Mock**
- **Mock Service** (`app/clients/payment_gateway.py`):
  - Simulate card payment processing
  - Random success/failure for testing
  - Configurable delay

**Task 6.1.3: Domain Logic**
- **Service** (`app/domain/payment_service.py`):
  - Validate card details (basic format check)
  - Call external payment gateway
  - On success, call Bank Service to deposit funds
  - Handle partial failures (payment succeeded but deposit failed)

**Task 6.1.4: Bank Service Client**
- **Client** (`app/clients/bank_client.py`):
  - gRPC client to Bank Service
  - Call new `DepositMoney` RPC method

**Task 6.1.5: gRPC Server Implementation**
- **Servicer** (`app/grpc_server/servicer.py`):
  - Implement `ProcessCardTopUp` RPC
  - Orchestrate: payment → bank deposit → response

**Task 6.1.6: Application Entry Point**
- **Main** (`app/main.py`):
  - Start gRPC server
  - Initialize Bank Service client

### 6.2 Bank Service Updates (Week 7)

**Task 6.2.1: Add DepositMoney RPC**
- Update `bank_service.proto`
- Implement deposit logic in servicer
- Update database models if needed (add deposit tracking)

**Task 6.2.2: Publish TopUp Events**
- Define `TopUpEvent` in AsyncAPI spec
- Create `topup_event.json` schema
- Publish events to Kafka after successful deposits

### 6.3 API Gateway Updates (Week 7)

**Task 6.3.1: Add BankCard Client**
- Implement gRPC client for Bank Card Adapter
- Add to API Gateway

**Task 6.3.2: Implement TopUp Endpoint**
- `POST /accounts/{accountId}/topup`
- Route requests to Bank Card Adapter
- Map responses to OpenAPI format

### 6.4 Analytics Updates (Week 8)

**Task 6.4.1: Handle TopUp Events**
- Update Kafka consumer to handle `TopUpEvent`
- Insert top-ups into ClickHouse
- Update operations query to include top-ups

### 6.5 Testing (Week 8)

- Test complete top-up flow
- Test payment failures
- Test partial failures (payment ok, deposit fails)
- End-to-end tests for top-up operations
- Update operations history tests

---

## Phase 7: Containerization and Full Docker Setup (Week 9)

**Focus**: Package everything in Docker Compose for easy deployment

### 7.1 Dockerfiles (Week 9)

**For each service, create**:
- Multi-stage build (build + runtime)
- Python base image (python:3.11-slim)
- Install dependencies from requirements.txt
- Copy application code
- Set proper user (non-root)
- Define entrypoint

**Example structure**:
```dockerfile
FROM python:3.11-slim as builder
WORKDIR /app
COPY requirements.txt .
RUN pip install --user -r requirements.txt

FROM python:3.11-slim
WORKDIR /app
COPY --from=builder /root/.local /root/.local
COPY . .
ENV PATH=/root/.local/bin:$PATH
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
```

### 7.2 Docker Compose Orchestration (Week 9)

**Complete docker-compose.yml with**:
- PostgreSQL with initialization scripts
- ClickHouse server
- Kafka + Zookeeper
- API Gateway service
- Bank Service
- Analytics Service
- Bank Card Adapter Service
- Health checks for all services
- Depends_on with conditions
- Environment variable configuration
- Volume mounts for persistence
- Network isolation
- Port mappings (API Gateway exposed, others internal)

**Example service definition**:
```yaml
bank-service:
  build: ./services/bank-service
  environment:
    DATABASE_URL: postgresql://user:pass@postgres/bank_db
    KAFKA_BROKER: kafka:9092
  depends_on:
    postgres:
      condition: service_healthy
    kafka:
      condition: service_started
  networks:
    - wallet-network
```

### 7.3 Development Workflow Scripts (Week 9)

**Scripts to create**:
- `scripts/dev-setup.ps1`: Initialize development environment
- `scripts/generate-protos.ps1`: Regenerate all gRPC code
- `scripts/run-tests.ps1`: Run all tests
- `scripts/start-services.ps1`: Start docker-compose with build
- `scripts/stop-services.ps1`: Stop and clean up

---

## Phase 8: Advanced Features (Weeks 10-12)

**Focus**: Production-readiness, scalability, observability
  - Implement `ProcessCardTopUp` RPC
  - Orchestrate: payment → bank deposit → response

**Task 3.1.5: Application Entry Point**
- **Main** (`app/main.py`):
  - Start gRPC server
  - Initialize Bank Service client

---

## Phase 8: Advanced Features (Weeks 10-12)

**Focus**: Production-readiness, scalability, observability

### 8.1 Authentication and Authorization (Week 10)

**Implementations**:
- **JWT-based auth**: Implement authentication middleware for API Gateway
- **User management**: Add user accounts and link to wallet accounts
- **Authorization**: Role-based access control (users can only access their own accounts)
- **Token validation**: Validate JWT tokens on each request

### 8.2 Scalability (Week 10-11)

**Implementations**:
- **Service replication**: Run multiple instances of Bank and Analytics services
- **Load balancing**: Use Nginx or Traefik in front of API Gateway
- **gRPC load balancing**: Client-side load balancing for gRPC calls
- **Database replication**: PostgreSQL read replicas (optional)

### 8.3 Fault Tolerance (Week 11)

### 8.3 Fault Tolerance (Week 11)

**Implementations**:
- **Circuit breaker**: Prevent cascading failures (use libraries like `pybreaker`)
- **Retry logic**: Exponential backoff for transient failures
- **Timeouts**: Set appropriate timeouts for all external calls
- **Graceful degradation**: Fallback responses when services are unavailable

### 8.4 Observability (Week 11-12)

**Implementations**:
- **Logging**: Structured logging (JSON format)
- **Metrics**: Prometheus metrics for each service
- **Tracing**: OpenTelemetry for distributed tracing
- **Monitoring dashboard**: Grafana for visualization
- **Alerting**: Set up alerts for critical errors

### 8.5 Security Enhancements (Week 12)

**Implementations**:
- **TLS**: Enable TLS for gRPC communication
- **Secrets management**: Use Docker secrets or Vault
- **Rate limiting**: Prevent abuse at API Gateway level
- **Input sanitization**: Additional validation and sanitization
- **SQL injection prevention**: Ensure parameterized queries

---

## Detailed Task Breakdown by Week

### Week 1: Contracts and Setup
1. Define bank_service.proto (TransferMoney, GetBalance RPCs only)
2. Generate Python gRPC code from proto
3. Setup PostgreSQL locally or in Docker container
4. Create bank_db database and accounts table
5. **Manually insert 2 test accounts** into database
6. Create basic requirements.txt for Bank Service and API Gateway
7. Initialize project structure for both services
8. Setup development environment (.env files, virtual environments)

### Week 2: Bank Service Foundation
1. Setup SQLAlchemy models (Account, Transfer)
2. Initialize Alembic and create first migration
3. Implement AccountRepository (read-only operations: get by id, check existence)
4. Implement TransferRepository with transaction logic
5. Implement TransferService with business rules
6. Write unit tests for domain logic
7. **Verify test accounts exist in database**

### Week 3: Bank Service Completion
1. Setup gRPC server infrastructure
2. Implement gRPC servicer (TransferMoney, GetBalance RPCs)
3. Integrate servicer with domain services
4. Write integration tests with PostgreSQL (using test accounts)
5. Test concurrent transfers and edge cases
6. Test with manually created accounts

### Week 4: API Gateway + Testing
1. Setup FastAPI application structure
2. Generate Pydantic models from OpenAPI (transfer endpoint only)
3. Implement gRPC client for Bank Service (transfer, get_balance)
4. Implement transfer endpoint in API Gateway
5. Add idempotency handling
6. Write API tests for transfer endpoint (using test account IDs)
7. End-to-end integration testing with pre-created accounts
8. Test error scenarios and edge cases

### Week 5: Kafka and Analytics Setup
1. Setup Kafka + Zookeeper (Docker or local)
2. Define AsyncAPI specification for TransferEvent
3. Create transfer_event.json schema
4. Add Kafka producer to Bank Service
5. Update transfer flow to publish events
6. Setup ClickHouse (Docker or local)
7. Create operations table schema

### Week 6: Analytics Service
1. Define analytics_service.proto
2. Implement Kafka consumer for Analytics
3. Implement ClickHouse insertion logic
4. Implement gRPC server for GetOperations
5. Add Analytics client to API Gateway
6. Implement GET /accounts/{accountId}/operations endpoint
7. End-to-end testing of analytics flow

### Week 7: Top-Up Infrastructure
1. Define bank_card_adapter.proto
2. Implement Bank Card Adapter service
3. Create mock payment gateway
4. Add DepositMoney RPC to Bank Service
5. Add TopUpEvent to AsyncAPI spec
6. Implement top-up endpoint in API Gateway
7. Unit and integration tests

### Week 8: Top-Up Completion
1. Update Analytics to handle TopUpEvents
2. Test complete top-up flow
3. Test error scenarios (payment failures, partial failures)
4. Performance testing for all operations
5. Documentation updates

### Week 9: Containerization
1. Create Dockerfiles for all services
2. Complete docker-compose.yml with all services
3. Test full system in Docker environment
4. Create development scripts (PowerShell)
5. Update README with Docker instructions

### Weeks 10-12: Advanced Features (Optional)
- Authentication and authorization
- Scalability improvements
- Fault tolerance patterns
- Observability stack
- Security hardening

---

## Definition of Done

### Phase 1 DoD (Week 1-4): Core Transfer Functionality
- [x] bank_service.proto defined with TransferMoney and GetBalance RPCs (no CreateAccount)
- [x] gRPC code generated for Bank Service
- [x] PostgreSQL running locally or in Docker
- [x] **Two test accounts manually created in database**
- [x] Bank Service implemented with SQLAlchemy + Alembic
- [x] AccountRepository with read-only operations (no create)
- [x] Transfer logic with ACID guarantees (atomicity, isolation)
- [x] gRPC server running and accepting requests
- [x] API Gateway implemented with FastAPI
- [x] Transfer endpoint (`POST /accounts/{accountId}/transfers`) functional
- [x] Idempotency handling implemented
- [x] Unit tests for Bank Service >80% coverage
- [x] Integration tests for API Gateway → Bank Service (using test accounts)
- [x] End-to-end tests for complete transfer flow (between test accounts)
- [x] Edge cases tested (insufficient funds, missing accounts, concurrent transfers)
- [x] No authentication required (permissive access)
- [x] No account creation functionality

### Phase 5 DoD (Week 5-6): Analytics Integration
- [x] Kafka + Zookeeper running
- [x] AsyncAPI specification defined for TransferEvent
- [x] Bank Service publishing events to Kafka
- [x] ClickHouse running with operations table
- [x] Analytics Service consuming and storing events
- [x] analytics_service.proto defined
- [x] GetOperations RPC implemented with pagination
- [x] API Gateway operations endpoint functional
- [x] End-to-end event flow tested
- [x] Unit tests for Analytics Service >80% coverage

### Phase 6 DoD (Week 7-8): Top-Up Functionality
- [x] bank_card_adapter.proto defined
- [x] Bank Card Adapter service implemented
- [x] Mock payment gateway functional
- [x] DepositMoney RPC added to Bank Service
- [x] TopUpEvent defined in AsyncAPI spec
- [x] Top-up endpoint in API Gateway functional
- [x] Analytics handling TopUpEvents
- [x] Complete top-up flow tested
- [x] Error scenarios tested
- [x] Unit tests for Bank Card Adapter >80% coverage

### Phase 7 DoD (Week 9): Full Docker Setup
- [x] Dockerfiles created for all services
- [x] docker-compose.yml complete with all services
- [x] All services healthy and communicating in Docker
- [x] Volumes configured for data persistence
- [x] Development scripts created
- [x] README updated with Docker instructions
- [x] Full system tested in Docker environment

---

## Key Technologies and Tools Summary

### Development
- Python 3.11+
- FastAPI
- gRPC (grpcio, grpcio-tools)
- SQLAlchemy + Alembic
- Pydantic
- aiokafka
- asyncpg
- clickhouse-driver

### Infrastructure
- Docker & Docker Compose
- PostgreSQL 15+
- ClickHouse
- Apache Kafka + Zookeeper

### Testing
- pytest
- pytest-asyncio
- testcontainers
- httpx (for API testing)

### Code Generation
- grpcio-tools (for proto → Python)
- datamodel-code-generator or openapi-generator (for OpenAPI → Python)
- AsyncAPI CLI (for validation)
- datamodel-code-generator (for JSON Schema → Pydantic models)

---

## Risk Mitigation

### Risk 1: gRPC Learning Curve
- **Mitigation**: Start with simple proto definitions, iterate
- **Fallback**: Use REST for initial implementation if needed

### Risk 2: Kafka Complexity
- **Mitigation**: Use simple producer/consumer initially, no complex topology
- **Fallback**: Use simple queue or polling mechanism

### Risk 3: Transaction Consistency
- **Mitigation**: Use PostgreSQL transactions with SERIALIZABLE isolation
- **Testing**: Comprehensive concurrent transaction tests

### Risk 4: Database Performance
- **Mitigation**: Proper indexing, connection pooling
- **Monitoring**: Query performance metrics

---

## Next Steps After Phase 1 (Week 4)

**Immediate next steps:**
1. ✅ You have a working transfer system (API Gateway + Bank Service)
2. ✅ All tests passing with good coverage
3. ✅ Basic idempotency implemented
4. ✅ Edge cases handled

**What to add next (Phase 5):**
1. Kafka infrastructure for event streaming
2. Analytics service for operation history
3. Operations endpoint in API Gateway

**Then (Phase 6):**
1. Bank Card Adapter for top-ups
2. Mock payment gateway
3. Top-up endpoint in API Gateway

**Finally (Phase 7):**
1. Package everything in Docker Compose
2. Production-ready deployment

---

## Resources and References

### Documentation
- FastAPI: https://fastapi.tiangolo.com/
- gRPC Python: https://grpc.io/docs/languages/python/
- SQLAlchemy: https://docs.sqlalchemy.org/
- ClickHouse: https://clickhouse.com/docs/
- Kafka Python: https://aiokafka.readthedocs.io/

### Example Projects
- Look for gRPC + FastAPI integration examples
- Distributed system patterns in Python
- Event-driven architecture examples

---

**Document Version**: 1.0  
**Last Updated**: October 22, 2025  
**Status**: Ready for implementation
