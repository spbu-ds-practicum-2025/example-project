# AI Agents Guide - Electronic Wallet Project

## Project Overview
Distributed electronic wallet system built with FastAPI, gRPC, PostgreSQL, and event-driven architecture.

**Current Focus**: API Gateway + Bank Service for money transfers. No authentication, no analytics, no top-ups.

---

## Technology Stack

### Languages & Frameworks
- **Python**: 3.11+
- **Framework**: FastAPI (all services)
- **API Protocols**:
  - REST/HTTP: External API (API Gateway)
  - gRPC: Inter-service communication (Bank, Analytics, BankCardAdapter)

### Databases
- **PostgreSQL**: Transactional database for Bank Service
- **ClickHouse**: Analytical database for Analytics Service (future)

### Message Broker
- **Apache Kafka**: Event streaming (future)

### Container & Orchestration
- **Docker**: Service containerization
- **Docker Compose**: Multi-service orchestration

---

## Core Python Libraries

### Web & API
```
fastapi>=0.104.0
uvicorn[standard]>=0.24.0
pydantic>=2.4.0
httpx>=0.25.0
```

### gRPC
```
grpcio>=1.59.0
grpcio-tools>=1.59.0
```

### Database
```
sqlalchemy>=2.0.0
asyncpg>=0.29.0
alembic>=1.12.0
clickhouse-driver>=0.2.6
```

### Message Broker
```
aiokafka>=0.8.0
```

### Testing
```
pytest>=7.4.0
pytest-asyncio>=0.21.0
testcontainers>=3.7.0
```

---

## Project Structure

### Current Services
```
services/
├── common/
│   ├── wallet-api/                  # OpenAPI spec for external API
│   │   └── openapi.yaml
│   └── bank-service-api/            # gRPC proto for Bank Service
│       └── bank_service.proto
│
├── api-gateway/                     # REST API Gateway
│   ├── app/
│   │   ├── main.py                  # FastAPI app
│   │   ├── api/
│   │   │   ├── models.py            # Generated from OpenAPI
│   │   │   └── routes.py            # Transfer endpoint
│   │   ├── clients/
│   │   │   └── bank_client.py       # gRPC client to Bank Service
│   │   ├── middleware/
│   │   │   └── idempotency.py       # Idempotency handling
│   │   └── config.py
│   └── requirements.txt
│
└── bank-service/                    # Bank Service (gRPC)
    ├── app/
    │   ├── main.py                  # FastAPI + gRPC server
    │   ├── grpc_server/
    │   │   ├── server.py            # gRPC server setup
    │   │   └── servicer.py          # TransferMoney, GetBalance RPCs
    │   ├── domain/
    │   │   ├── models.py            # Business models
    │   │   ├── repositories.py      # AccountRepo (read-only), TransferRepo
    │   │   └── services.py          # TransferService
    │   ├── db/
    │   │   ├── models.py            # SQLAlchemy models (Account, Transfer)
    │   │   ├── session.py           # DB session factory
    │   │   └── migrations/          # Alembic migrations
    │   └── config.py
    ├── proto/                       # Generated gRPC code
    │   └── bank_service_pb2.py
    └── requirements.txt
```

### Future Services
```
services/
├── common/
│   ├── analytics-service-api/       # Analytics gRPC proto
│   └── analytics-service-kafka-spec/
│       ├── asyncapi.yaml            # Kafka events spec
│       └── schemas/
│           ├── transfer_event.json
│           └── topup_event.json
│
├── analytics-service/               # Analytics Service
│   ├── app/
│   │   ├── kafka/consumer.py        # Kafka consumer
│   │   └── db/client.py             # ClickHouse client
│   └── requirements.txt
│
└── bank-card-adapter/               # Bank Card Adapter
    ├── app/
    │   ├── clients/
    │   │   ├── payment_gateway.py   # Mock payment service
    │   │   └── bank_client.py       # gRPC client to Bank
    │   └── grpc_server/
    └── requirements.txt
```

---

## Contract-First Approach

### 1. OpenAPI (REST API)
- **Spec**: `services/common/wallet-api/openapi.yaml`
- **Tool**: `datamodel-code-generator` or `openapi-generator`
- **Generate**: Pydantic models for API Gateway

### 2. Protocol Buffers (gRPC)
- **Specs**: `services/common/*/**.proto`
- **Tool**: `grpcio-tools`
- **Generate**: Python gRPC stubs and message classes

### 3. AsyncAPI (Kafka Events - future)
- **Spec**: `services/common/analytics-service-kafka-spec/asyncapi.yaml`
- **Schemas**: JSON Schema files in `schemas/`
- **Tool**: `datamodel-code-generator` (from JSON Schema)
- **Generate**: Pydantic models for event serialization/deserialization

---

## Development Workflow

### 1. Setup Environment
```powershell
# Create virtual environment
python -m venv venv
.\venv\Scripts\Activate.ps1

# Install dependencies
pip install -r requirements.txt
```

### 2. Database Setup
```powershell
# Start PostgreSQL (Docker)
docker run -d --name wallet-postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=bank_db -p 5432:5432 postgres:15

# Run migrations
cd services/bank-service
alembic upgrade head

# Insert test data manually or via script
# See docker/postgres/test_data.sql for test account setup
```

### 3. Code Generation
```powershell
# Generate gRPC code
.\scripts\generate-protos.ps1
```

### 4. Run Services
```powershell
# Terminal 1: Bank Service
cd services/bank-service
uvicorn app.main:app --reload --port 50051

# Terminal 2: API Gateway
cd services/api-gateway
uvicorn app.main:app --reload --port 8080
```

---

## Key APIs

### API Gateway (REST)
- `POST /accounts/{accountId}/transfers` - Transfer money between accounts

#### Bank Service (gRPC)
- `TransferMoney` - Execute money transfer
- `GetBalance` - Get account balance

---

## Code Generation Scripts

### generate-protos.ps1
```powershell
# Generate Bank Service proto
python -m grpc_tools.protoc `
  -I=services/common/bank-service-api `
  --python_out=services/bank-service/proto `
  --grpc_python_out=services/bank-service/proto `
  services/common/bank-service-api/bank_service.proto

Write-Host "gRPC code generated successfully"
```