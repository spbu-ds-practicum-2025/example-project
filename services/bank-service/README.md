# Bank Service Documentation

Distributed electronic wallet banking service implementing money transfers with gRPC API, PostgreSQL persistence, and RabbitMQ event publishing.

## Quick Start

### Prerequisites
- Go 1.23+
- PostgreSQL 15+
- Docker (for RabbitMQ and testing)

### Run the Service

```bash
# 1. Start PostgreSQL
docker run -d --name bank-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=bank_db \
  -p 5432:5432 \
  postgres:15

# 2. Run migrations
cd services/bank-service
chmod +x run-migrations.sh
./run-migrations.sh up

# 3. Start RabbitMQ (optional, for event publishing)
docker run -d --name rabbitmq \
  -p 5672:5672 -p 15672:15672 \
  rabbitmq:3-management

# 4. Start the service
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/bank_db?sslmode=disable"
export RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
go run ./cmd/server/main.go
```

Service starts on port `50051` (configurable via `PORT` env var).

### Test with grpcurl

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# Execute a transfer
grpcurl -plaintext -d '{
  "sender_id": "11111111-1111-1111-1111-111111111111",
  "recipient_id": "22222222-2222-2222-2222-222222222222",
  "amount": {"value": "100.00", "currency_code": "RUB"},
  "idempotency_key": "test-transfer-1"
}' localhost:50051 bank.v1.BankService/TransferMoney

# Check account balance
grpcurl -plaintext -d '{
  "account_id": "11111111-1111-1111-1111-111111111111"
}' localhost:50051 bank.v1.BankService/GetAccount
```

---

## Architecture

### Layers

```
┌─────────────────────────────────────┐
│  gRPC Server (internal/grpc)        │  ← Adapter: gRPC ↔ Domain
├─────────────────────────────────────┤
│  Domain Layer (internal/domain)     │  ← Business Logic
│  - Models, Services, Validation     │
├─────────────────────────────────────┤
│  Database Layer (internal/db)       │  ← Repositories (raw SQL)
├─────────────────────────────────────┤
│  Events Layer (internal/events)     │  ← RabbitMQ Publisher
├─────────────────────────────────────┤
│  PostgreSQL + RabbitMQ              │  ← Infrastructure
└─────────────────────────────────────┘
```

### Key Components

**Domain Layer** (`internal/domain/`)
- Core business entities: `Account`, `Transfer`, `Amount`
- Business logic: `TransferService.ExecuteTransfer()`
- Repository interfaces (no infrastructure dependencies)

**Database Layer** (`internal/db/`)
- PostgreSQL repositories using `pgx` (no ORM)
- Transaction management with context propagation
- Pessimistic locking for concurrent safety

**gRPC Server** (`internal/grpc/`)
- Implements `BankService` proto definition
- Request validation and error mapping
- Adapts between protobuf and domain models

**Events Layer** (`internal/events/`)
- RabbitMQ publisher for transfer completion events
- Publishes to topic exchange: `bank.operations`
- Asynchronous, best-effort delivery

---

## Database Schema

### Tables

**accounts**
```sql
id                    UUID PRIMARY KEY
balance_value         NUMERIC(15,2) NOT NULL CHECK (>= 0)
balance_currency_code VARCHAR(3) NOT NULL
created_at            TIMESTAMP NOT NULL
updated_at            TIMESTAMP NOT NULL  -- auto-updated via trigger
```

**transfers**
```sql
id                    UUID PRIMARY KEY
sender_id             UUID NOT NULL REFERENCES accounts(id)
recipient_id          UUID NOT NULL REFERENCES accounts(id)
amount_value          NUMERIC(15,2) NOT NULL CHECK (> 0)
amount_currency_code  VARCHAR(3) NOT NULL
idempotency_key       VARCHAR(255) NOT NULL UNIQUE
status                VARCHAR(20) NOT NULL  -- PENDING, SUCCESS, FAILED
message               TEXT
created_at            TIMESTAMP NOT NULL
completed_at          TIMESTAMP
```

**Indexes**: sender_id, recipient_id, idempotency_key, created_at, status

### Test Accounts

Migration `004_seed_test_data` creates accounts for testing:

| Account ID | Balance | Currency |
|------------|---------|----------|
| `11111111-1111-1111-1111-111111111111` | 1000.00 | RUB |
| `22222222-2222-2222-2222-222222222222` | 500.00 | RUB |
| `33333333-3333-3333-3333-333333333333` | 10000.00 | RUB |
| `44444444-4444-4444-4444-444444444444` | 50.00 | RUB |
| `55555555-5555-5555-5555-555555555555` | 0.00 | RUB |

**Note**: Remove seed migration for production.

---

## gRPC API

### TransferMoney

Transfers money between accounts atomically.

**Request**:
```json
{
  "sender_id": "uuid",
  "recipient_id": "uuid",
  "amount": {"value": "100.50", "currency_code": "RUB"},
  "idempotency_key": "unique-string"
}
```

**Response**:
```json
{
  "operation_id": "transfer-uuid",
  "status": "TRANSFER_STATUS_SUCCESS",
  "message": "Transfer completed successfully",
  "timestamp": "2025-11-08T14:30:00Z"
}
```

**Features**:
- ✅ Atomic execution within database transaction
- ✅ Idempotent (same idempotency_key returns same result)
- ✅ Account locking to prevent race conditions
- ✅ Insufficient funds validation
- ✅ Event publishing to RabbitMQ after commit

**Error Codes**:
- `INVALID_ARGUMENT`: Missing fields, invalid UUIDs, same sender/recipient, currency mismatch
- `NOT_FOUND`: Account doesn't exist
- `FAILED_PRECONDITION`: Insufficient funds
- `INTERNAL`: Database or system errors

### GetAccount

Retrieves account balance and metadata.

**Request**: `{"account_id": "uuid"}`

**Response**: `{"account_id": "uuid", "balance": {...}, "timestamp": "..."}`

### TopUp

**Status**: Not implemented (returns `UNIMPLEMENTED`)

---

## Event Publishing

Transfer completion events are published to RabbitMQ following the AsyncAPI specification.

**Exchange**: `bank.operations` (topic)  
**Routing Key**: `bank.operations.transfer.completed`

**Event Payload**:
```json
{
  "eventId": "uuid",
  "eventType": "transfer.completed",
  "eventTimestamp": "2025-11-08T...",
  "operationId": "transfer-uuid",
  "senderId": "account-uuid",
  "recipientId": "account-uuid",
  "amount": {"value": "100.50", "currencyCode": "RUB"},
  "idempotencyKey": "...",
  "status": "SUCCESS",
  "timestamp": "2025-11-08T...",
  "message": "Transfer completed successfully"
}
```

**Publishing Strategy**: Asynchronous, best-effort after transaction commit. For stronger guarantees, implement an outbox pattern.

---

## Key Design Patterns

### 1. Repository Pattern
Domain defines repository interfaces; database layer implements them. Enables testing with mocks.

### 2. Transaction Management
Operations wrapped in database transactions for atomicity. Transaction stored in context and used by repositories.

### 3. Pessimistic Locking
Accounts locked with `SELECT ... FOR UPDATE` during transfers. Locks acquired in deterministic order (UUID comparison) to prevent deadlocks.

### 4. Idempotency
Transfers identified by unique `idempotency_key`. Duplicate requests return existing transfer without re-execution.

### 5. Event-Driven Architecture
Domain events published asynchronously to RabbitMQ for analytics and audit.

---

## Testing

See [TESTING.md](./TESTING.md) for comprehensive testing documentation.

**Quick Test Commands**:
```bash
# Unit tests (fast, no Docker)
go test -v -short ./...

# Integration tests (requires Docker)
./run-tests.sh integration

# All tests
./run-tests.sh all

# Coverage report
./run-tests.sh coverage
```

---

## Database Management

### Migrations

```bash
# Apply all migrations
./run-migrations.sh up

# Check version
./run-migrations.sh version

# Rollback last migration
./run-migrations.sh down 1

# Rollback all
./run-migrations.sh down
```

### Direct Database Access

```bash
# Connect via Docker
docker exec -it bank-postgres psql -U postgres -d bank_db

# List tables
\dt

# Check account balances
SELECT id, balance_value, balance_currency_code FROM accounts;

# View recent transfers
SELECT * FROM transfers ORDER BY created_at DESC LIMIT 10;
```

---

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/bank_db?sslmode=disable` | PostgreSQL connection string |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | RabbitMQ AMQP connection string |
| `PORT` | `50051` | gRPC server port |

---

## Production Considerations

### Critical TODOs

1. **Replace float64 arithmetic** with `github.com/shopspring/decimal` for precise money calculations
2. **Implement outbox pattern** for reliable event publishing (transactional guarantees)
3. **Add structured logging** (replace `fmt.Printf` in event publisher)
4. **Add metrics and monitoring** (Prometheus, OpenTelemetry)
5. **Enable TLS** for gRPC connections
6. **Implement rate limiting** and request throttling
7. **Add comprehensive error logging** and distributed tracing
8. **Remove test seed migration** (004_seed_test_data)

### Deployment Checklist

- [ ] TLS certificates configured
- [ ] Database connection pooling tuned
- [ ] RabbitMQ clustering for high availability
- [ ] Health check endpoints added
- [ ] Graceful shutdown implemented (already present)
- [ ] Resource limits and timeouts configured
- [ ] Monitoring and alerting set up
- [ ] Backup and disaster recovery procedures

---

## Development Workflow

```bash
# 1. Make code changes
# 2. Run unit tests
go test -v -short ./...

# 3. Build
go build -o bin/bank-service ./cmd/server

# 4. Run locally
./bin/bank-service

# 5. Test with grpcurl or integration tests
./run-tests.sh integration
```

---

## Troubleshooting

**Database connection failed**:
- Check PostgreSQL is running: `docker ps | grep postgres`
- Verify DATABASE_URL is correct
- Check migrations ran: `./run-migrations.sh version`

**RabbitMQ events not publishing**:
- Check RabbitMQ is running: `docker ps | grep rabbitmq`
- Service continues without RabbitMQ (best-effort)
- Check logs for publisher initialization errors

**Tests failing**:
- Ensure Docker is running for integration tests
- Run unit tests only: `go test -v -short ./...`
- Check testcontainers can pull images

---

## AI Agent Instructions

When modifying this service:

1. **Never use ORM** - all database code uses raw SQL with pgx
2. **Maintain transaction safety** - use `txManager.WithTransaction` for multi-step operations
3. **Preserve idempotency** - all state-changing operations must check idempotency keys
4. **Keep domain layer pure** - no infrastructure dependencies in `internal/domain/`
5. **Follow gRPC error mapping** - domain errors → gRPC status codes via `mapDomainErrorToGRPC`
6. **Use bash scripts** - never create PowerShell scripts
7. **Test coverage required** - add tests for new features (unit + integration)
8. **Document schema changes** - update this README when modifying database or API

---

## References

- **Proto API**: `services/common/bank-service-api/bank_service.proto`
- **AsyncAPI Spec**: `services/common/analytics-service-kafka-spec/asyncapi.yaml`
- **Testing Guide**: `TESTING.md`
- **Migration Files**: `migrations/`
