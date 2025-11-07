# Integration Tests

This directory contains integration tests for the Bank Service gRPC API.

## Prerequisites

### Docker
Integration tests use [testcontainers-go](https://github.com/testcontainers/testcontainers-go) to spin up real PostgreSQL and RabbitMQ instances in Docker containers. 

**You must have Docker running** before executing integration tests.

#### Install Docker Desktop
- **Windows**: [Docker Desktop for Windows](https://docs.docker.com/desktop/install/windows-install/)
- **macOS**: [Docker Desktop for Mac](https://docs.docker.com/desktop/install/mac-install/)
- **Linux**: [Docker Engine](https://docs.docker.com/engine/install/)

After installation, start Docker Desktop and ensure the Docker daemon is running.

## Running Tests

### Run All Tests (Unit + Integration)
```bash
# From bank-service directory
go test -v ./... -timeout 5m
```

### Run Only Integration Tests
```bash
go test -v ./internal/grpc/... -run TestTransferMoneyIntegration -timeout 5m
```

### Run Only Unit Tests (skip integration)
```bash
go test -v -short ./...
```

The integration test will:
1. Start a PostgreSQL container
2. Start a RabbitMQ container
3. Run database migrations
4. Create test accounts
5. Start a gRPC server with all dependencies wired
6. Execute a TransferMoney request
7. Verify the transfer succeeded in the database
8. Verify the event was published to RabbitMQ
9. Test idempotency by calling the same request twice
10. Clean up containers

## Test Structure

### TestTransferMoneyIntegration
A full end-to-end integration test that:
- Uses real PostgreSQL database (via testcontainer)
- Uses real RabbitMQ message broker (via testcontainer)
- Tests the complete gRPC API flow
- Validates event publishing to RabbitMQ
- Verifies idempotency guarantees
- Checks account balance changes

## Troubleshooting

### Docker not running
```
error during connect: this error may indicate that the docker daemon is not running
```
**Solution**: Start Docker Desktop and wait for it to fully initialize.

### Port conflicts
If containers fail to start due to port conflicts, make sure no other services are using:
- PostgreSQL: 5432 (testcontainers uses random ports, so this is rare)
- RabbitMQ: 5672, 15672 (testcontainers uses random ports)

### Slow test execution
First run may be slow as testcontainers downloads Docker images:
- `postgres:15`
- `rabbitmq:3-management`

Subsequent runs will be faster as images are cached.

### Test timeout
If tests timeout, increase the timeout:
```bash
go test -v ./internal/grpc/... -run TestTransferMoneyIntegration -timeout 10m
```

## CI/CD Integration

For CI/CD pipelines (GitHub Actions, GitLab CI, etc.), ensure:
1. Docker is available in the CI environment
2. The test runner has permissions to create containers
3. Adequate timeout is set (5-10 minutes for first run)

Example GitHub Actions setup:
```yaml
name: Integration Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      - name: Run integration tests
        run: go test -v ./... -timeout 10m
```

## What Gets Tested

### TransferMoney gRPC Method
- ✅ Successful transfer between accounts
- ✅ Account balance updates (debit sender, credit recipient)
- ✅ Transfer record creation in database
- ✅ Event publishing to RabbitMQ
- ✅ Event structure validation (AsyncAPI spec)
- ✅ Idempotency (duplicate requests return same result)
- ✅ Concurrent access safety (database locks)

### Event Publishing
- ✅ Event structure matches AsyncAPI spec
- ✅ Correct routing key: `bank.operations.transfer.completed`
- ✅ Correct exchange: `bank.operations` (topic)
- ✅ JSON payload contains all required fields
- ✅ Amount, IDs, and status are correctly serialized

## Future Tests

Additional integration tests to add:
- [ ] Insufficient funds scenario
- [ ] Currency mismatch handling
- [ ] Concurrent transfer stress test
- [ ] RabbitMQ publisher failure handling
- [ ] Database transaction rollback scenarios
- [ ] GetAccount endpoint
- [ ] TopUp endpoint (when implemented)
