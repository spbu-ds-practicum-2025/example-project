# Bank Service Testing Guide

Comprehensive test suite covering unit tests, integration tests, and end-to-end scenarios.

## Test Structure

### Unit Tests (`internal/grpc/server_test.go`)
Fast tests without external dependencies:
- Request validation
- Domain error mapping to gRPC codes
- Input sanitization
- gRPC status code verification

**Run**:
```bash
go test -v -short ./...
# or
./run-tests.sh unit
```

### Integration Tests (`internal/grpc/server_integration_test.go`)
Full end-to-end tests with real infrastructure:
- PostgreSQL (via testcontainers)
- RabbitMQ (via testcontainers)
- Complete gRPC flow
- Event publishing verification
- Idempotency validation

**Requires**: Docker running

**Run**:
```bash
go test -v ./internal/grpc/... -run TestTransferMoneyIntegration -timeout 10m
# or
./run-tests.sh integration
```

---

## Integration Test Coverage

### TestTransferMoneyIntegration

Comprehensive end-to-end test covering:

#### Infrastructure Setup
- ✅ Starts PostgreSQL 15 container
- ✅ Starts RabbitMQ 3-management container
- ✅ Runs database migrations programmatically
- ✅ Creates test accounts with initial balances
- ✅ Sets up RabbitMQ exchange and queues

#### Service Initialization
- ✅ Creates connection pool
- ✅ Initializes repositories (Account, Transfer)
- ✅ Creates transaction manager
- ✅ Wires RabbitMQ publisher
- ✅ Starts in-memory gRPC server (bufconn)
- ✅ Creates gRPC client

#### Transfer Execution
- ✅ Executes TransferMoney via gRPC
- ✅ Validates response status (SUCCESS)
- ✅ Verifies operation ID returned

#### Database Verification
- ✅ Checks sender balance decreased (1000.00 → 899.50)
- ✅ Checks recipient balance increased (500.00 → 600.50)
- ✅ Verifies transfer record created
- ✅ Confirms transaction atomicity

#### Event Publishing Verification
- ✅ Consumes event from RabbitMQ
- ✅ Validates event structure (AsyncAPI spec)
- ✅ Checks eventType: "transfer.completed"
- ✅ Verifies operationId matches transfer
- ✅ Validates senderId and recipientId
- ✅ Checks amount: {value: "100.50", currencyCode: "RUB"}
- ✅ Confirms idempotencyKey
- ✅ Verifies status: "SUCCESS"
- ✅ Validates routing key: `bank.operations.transfer.completed`
- ✅ Confirms exchange: `bank.operations`

#### Idempotency Testing
- ✅ Calls TransferMoney again with same idempotency key
- ✅ Verifies same operation ID returned
- ✅ Confirms balances unchanged (no duplicate transfer)

#### Cleanup
- ✅ Stops RabbitMQ consumer
- ✅ Closes gRPC connections
- ✅ Terminates containers

---

## Test Accounts

Integration tests use these accounts:

| Account ID | Initial Balance | Currency |
|------------|----------------|----------|
| `11111111-1111-1111-1111-111111111111` | 1000.00 | RUB |
| `22222222-2222-2222-2222-222222222222` | 500.00 | RUB |

**Test transfer**: 100.50 RUB from account 1 → account 2

---

## Running Tests

### Using Test Runner Script

```bash
# Unit tests (fast, no Docker)
./run-tests.sh unit

# Integration tests (requires Docker)
./run-tests.sh integration

# All tests
./run-tests.sh all

# Coverage report
./run-tests.sh coverage
```

### Direct Go Commands

```bash
# Unit tests
go test -v -short ./...

# Integration tests
go test -v ./internal/grpc/... -run TestTransferMoneyIntegration -timeout 10m

# All tests
go test -v ./... -timeout 10m

# With coverage
go test -v -short -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Tests
on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      - name: Run unit tests
        run: go test -v -short ./...
        working-directory: services/bank-service

  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      - name: Run integration tests
        run: go test -v ./... -timeout 10m
        working-directory: services/bank-service
```

---

## Test Data Examples

### Transfer Request
```json
{
  "sender_id": "11111111-1111-1111-1111-111111111111",
  "recipient_id": "22222222-2222-2222-2222-222222222222",
  "amount": {"value": "100.50", "currency_code": "RUB"},
  "idempotency_key": "<uuid>"
}
```

### Expected RabbitMQ Event
```json
{
  "eventId": "<uuid>",
  "eventType": "transfer.completed",
  "eventTimestamp": "2025-11-08T...",
  "operationId": "<transfer-id>",
  "senderId": "11111111-1111-1111-1111-111111111111",
  "recipientId": "22222222-2222-2222-2222-222222222222",
  "amount": {
    "value": "100.50",
    "currencyCode": "RUB"
  },
  "idempotencyKey": "<same-uuid>",
  "status": "SUCCESS",
  "timestamp": "2025-11-08T...",
  "message": "Transfer completed successfully"
}
```

---

## Troubleshooting

### Docker Not Running
```
error during connect: docker daemon is not running
```
**Solution**: Start Docker Desktop and wait for it to initialize.

### Container Startup Timeout
**Solution**: Increase timeout: `-timeout 15m`  
May occur on first run (downloading images: postgres:15, rabbitmq:3-management)

### Event Not Received
- Check consumer started before transfer
- Verify exchange and routing key match
- Increase wait timeout if needed

### Database Connection Failed
- Ensure PostgreSQL container started
- Check migrations ran successfully
- Verify connection string format

### Balance Mismatch
- Check test account initial balances
- Verify transfer amount calculation
- Ensure no concurrent transfers

---

## Future Test Coverage

Additional scenarios to implement:

- [ ] Insufficient funds scenario
- [ ] Currency mismatch handling
- [ ] Concurrent transfer stress test
- [ ] RabbitMQ publisher failure handling
- [ ] Database transaction rollback scenarios
- [ ] GetAccount endpoint edge cases
- [ ] TopUp endpoint (when implemented)
- [ ] Performance benchmarks

---

## Current Coverage

- ✅ Request validation
- ✅ gRPC server implementation
- ✅ Database operations
- ✅ Event publishing
- ✅ Idempotency
- ✅ Transaction handling
- ⚠️ Domain logic (needs dedicated unit tests)
- ⚠️ Repository implementations (needs dedicated unit tests)
- ❌ Error scenarios (insufficient funds, etc.)
- ❌ Concurrent access patterns
