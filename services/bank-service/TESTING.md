# Bank Service Testing Guide

This document describes the test suite for the Bank Service.

## Test Structure

### Unit Tests (`server_test.go`)
Fast tests that don't require external dependencies:
- Request validation tests
- Domain error mapping tests
- gRPC status code verification
- Input sanitization tests

**Run with:**
```bash
go test -v -short ./...
# or
./run-tests.sh unit
```

### Integration Tests (`server_integration_test.go`)
Full end-to-end tests using real dependencies:
- PostgreSQL (via testcontainer)
- RabbitMQ (via testcontainer)
- Complete gRPC flow
- Event publishing verification
- Idempotency validation

**Requires:** Docker Desktop running

**Run with:**
```bash
go test -v ./internal/grpc/... -run TestTransferMoneyIntegration -timeout 10m
# or
./run-tests.sh integration
```

## Integration Test Coverage

### TestTransferMoneyIntegration

This comprehensive test covers:

#### 1. Infrastructure Setup
- ✅ Starts PostgreSQL 15 container
- ✅ Starts RabbitMQ 3 (management) container
- ✅ Runs database migrations programmatically
- ✅ Creates test accounts with initial balances
- ✅ Sets up RabbitMQ exchange and queues

#### 2. Service Initialization
- ✅ Creates connection pool
- ✅ Initializes repositories (Account, Transfer)
- ✅ Creates transaction manager
- ✅ Wires RabbitMQ publisher
- ✅ Starts in-memory gRPC server (bufconn)
- ✅ Creates gRPC client

#### 3. Transfer Execution
- ✅ Executes TransferMoney via gRPC
- ✅ Validates response status (SUCCESS)
- ✅ Verifies operation ID is returned

#### 4. Database Verification
- ✅ Checks sender balance decreased (1000.00 → 899.50)
- ✅ Checks recipient balance increased (500.00 → 600.50)
- ✅ Verifies transfer record created
- ✅ Confirms transaction atomicity

#### 5. Event Publishing Verification
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

#### 6. Idempotency Testing
- ✅ Calls TransferMoney again with same idempotency key
- ✅ Verifies same operation ID returned
- ✅ Confirms balances unchanged (no duplicate transfer)

#### 7. Cleanup
- ✅ Stops RabbitMQ consumer
- ✅ Closes gRPC connections
- ✅ Terminates containers

## Test Accounts

The integration test creates these accounts:

| Account ID | Initial Balance | Currency |
|------------|----------------|----------|
| `11111111-1111-1111-1111-111111111111` | 1000.00 | RUB |
| `22222222-2222-2222-2222-222222222222` | 500.00 | RUB |

Test transfer: 100.50 RUB from account 1 → account 2

## Running Tests

### Prerequisites
- Go 1.23+
- Docker Desktop (for integration tests)

### Quick Start

**Run unit tests only (fast, no Docker):**
```bash
./run-tests.sh unit
```

**Run integration tests (requires Docker):**
```bash
# Make sure Docker Desktop is running
./run-tests.sh integration
```

**Run all tests:**
```bash
./run-tests.sh all
```

**Generate coverage report:**
```bash
./run-tests.sh coverage
```

### Manual Commands

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

## Test Data

### Successful Transfer Scenario
```json
{
  "sender_id": "11111111-1111-1111-1111-111111111111",
  "recipient_id": "22222222-2222-2222-2222-222222222222",
  "amount": {
    "value": "100.50",
    "currency_code": "RUB"
  },
  "idempotency_key": "<uuid>"
}
```

### Expected Event
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

## Troubleshooting

### Docker Issues

**Error:** `error during connect: docker daemon is not running`
- **Solution:** Start Docker Desktop and wait for it to initialize

**Error:** Container startup timeout
- **Solution:** Increase timeout: `-timeout 15m`
- May occur on first run (downloading images)

### Test Failures

**RabbitMQ event not received**
- Check consumer started before transfer
- Verify exchange and routing key match
- Increase wait timeout if needed

**Database connection failed**
- Ensure PostgreSQL container started
- Check migrations ran successfully
- Verify connection string format

**Balance mismatch**
- Check test account initial balances
- Verify transfer amount calculation
- Ensure no concurrent transfers

## Future Enhancements

- [ ] Add stress test with concurrent transfers
- [ ] Test insufficient funds scenario
- [ ] Test currency mismatch handling
- [ ] Add RabbitMQ connection failure test
- [ ] Test database deadlock scenarios
- [ ] Add performance benchmarks
- [ ] Test GetAccount endpoint
- [ ] Test TopUp endpoint (when implemented)

## Metrics

Current test coverage areas:
- ✅ Request validation
- ✅ gRPC server implementation
- ✅ Database operations (via integration test)
- ✅ Event publishing (via integration test)
- ✅ Idempotency (via integration test)
- ✅ Transaction handling (via integration test)
- ⚠️ Domain logic (covered indirectly, needs unit tests)
- ⚠️ Repository implementations (covered indirectly, needs unit tests)
- ❌ Error scenarios (insufficient funds, etc.)
- ❌ Concurrent access patterns
