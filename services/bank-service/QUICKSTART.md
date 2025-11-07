# Bank Service - Quick Start Guide

Complete guide to run and test the Bank Service.

## Prerequisites

- Go 1.20+
- PostgreSQL 15+
- `grpcurl` (optional, for testing)

## Setup

### 1. Start PostgreSQL

**Using Docker** (recommended):
```bash
docker run -d \
  --name bank-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=bank_db \
  -p 5432:5432 \
  postgres:15
```

**Or using Make**:
```bash
make db-start
```

### 2. Run Migrations

```bash
# Using the migration script
./run-migrations.sh up

# Or using Make
make migrate-up
```

**Verify migrations**:
```bash
./run-migrations.sh version
```

You should see migration version 4 (with test data).

### 3. Set Environment Variables

```bash
# Database connection
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/bank_db?sslmode=disable"

# Server port (optional, defaults to 50051)
export PORT="50051"
```

### 4. Start the Server

```bash
# Using go run
go run ./cmd/server/main.go

# Or using Make
make run
```

**Expected output**:
```
database connection pool initialized
domain services initialized
gRPC services registered
bank-service gRPC server starting on :50051
```

## Testing

### Option 1: Using grpcurl (Recommended)

Install grpcurl:
```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

#### List Services
```bash
grpcurl -plaintext localhost:50051 list
```

**Output**:
```
bank.v1.BankService
grpc.reflection.v1alpha.ServerReflection
```

#### Test TransferMoney

**Successful Transfer**:
```bash
grpcurl -plaintext -d '{
  "sender_id": "11111111-1111-1111-1111-111111111111",
  "recipient_id": "22222222-2222-2222-2222-222222222222",
  "amount": {
    "value": "100.00",
    "currency_code": "RUB"
  },
  "idempotency_key": "test-transfer-1"
}' localhost:50051 bank.v1.BankService/TransferMoney
```

**Expected Response**:
```json
{
  "operationId": "...",
  "status": "TRANSFER_STATUS_SUCCESS",
  "message": "Transfer completed successfully",
  "timestamp": "2025-11-08T..."
}
```

**Test Idempotency** (run the same request again):
```bash
# Should return the same operation_id and result
grpcurl -plaintext -d '{
  "sender_id": "11111111-1111-1111-1111-111111111111",
  "recipient_id": "22222222-2222-2222-2222-222222222222",
  "amount": {
    "value": "100.00",
    "currency_code": "RUB"
  },
  "idempotency_key": "test-transfer-1"
}' localhost:50051 bank.v1.BankService/TransferMoney
```

**Test Insufficient Funds**:
```bash
grpcurl -plaintext -d '{
  "sender_id": "55555555-5555-5555-5555-555555555555",
  "recipient_id": "11111111-1111-1111-1111-111111111111",
  "amount": {
    "value": "100.00",
    "currency_code": "RUB"
  },
  "idempotency_key": "test-insufficient-1"
}' localhost:50051 bank.v1.BankService/TransferMoney
```

**Expected Error**:
```
ERROR:
  Code: FailedPrecondition
  Message: insufficient funds
```

**Test Invalid UUID**:
```bash
grpcurl -plaintext -d '{
  "sender_id": "invalid-uuid",
  "recipient_id": "22222222-2222-2222-2222-222222222222",
  "amount": {
    "value": "100.00",
    "currency_code": "RUB"
  },
  "idempotency_key": "test-invalid-1"
}' localhost:50051 bank.v1.BankService/TransferMoney
```

**Expected Error**:
```
ERROR:
  Code: InvalidArgument
  Message: invalid sender_id: ...
```

#### Test GetAccount

```bash
grpcurl -plaintext -d '{
  "account_id": "11111111-1111-1111-1111-111111111111"
}' localhost:50051 bank.v1.BankService/GetAccount
```

**Expected Response**:
```json
{
  "accountId": "11111111-1111-1111-1111-111111111111",
  "balance": {
    "value": "900.00",
    "currencyCode": "RUB"
  },
  "timestamp": "2025-11-08T..."
}
```

### Option 2: Using PostgreSQL Client

Connect to the database and inspect the data:

```bash
# Connect to database
docker exec -it bank-postgres psql -U postgres -d bank_db

# Or if PostgreSQL is local
psql -U postgres -d bank_db
```

**Check accounts**:
```sql
SELECT id, balance_value, balance_currency_code FROM accounts;
```

**Check transfers**:
```sql
SELECT id, sender_id, recipient_id, amount_value, status, message, created_at 
FROM transfers 
ORDER BY created_at DESC;
```

**Check idempotency**:
```sql
SELECT idempotency_key, id, status FROM transfers;
```

### Option 3: Using Go Client

Create a test file `test_client.go`:

```go
package main

import (
    "context"
    "log"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    pb "github.com/spbu-ds-practicum-2025/example-project/services/bank-service/proto/bank.v1"
)

func main() {
    // Connect to server
    conn, err := grpc.Dial("localhost:50051", 
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewBankServiceClient(conn)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Test TransferMoney
    log.Println("Testing TransferMoney...")
    resp, err := client.TransferMoney(ctx, &pb.TransferMoneyRequest{
        SenderId:       "11111111-1111-1111-1111-111111111111",
        RecipientId:    "22222222-2222-2222-2222-222222222222",
        Amount: &pb.Amount{
            Value:        "50.00",
            CurrencyCode: "RUB",
        },
        IdempotencyKey: "go-client-test-1",
    })

    if err != nil {
        log.Fatalf("TransferMoney failed: %v", err)
    }

    log.Printf("Transfer successful!")
    log.Printf("  Operation ID: %s", resp.OperationId)
    log.Printf("  Status: %s", resp.Status)
    log.Printf("  Message: %s", resp.Message)
    log.Printf("  Timestamp: %s", resp.Timestamp)

    // Test GetAccount
    log.Println("\nTesting GetAccount...")
    accResp, err := client.GetAccount(ctx, &pb.GetAccountRequest{
        AccountId: "11111111-1111-1111-1111-111111111111",
    })

    if err != nil {
        log.Fatalf("GetAccount failed: %v", err)
    }

    log.Printf("Account retrieved!")
    log.Printf("  Account ID: %s", accResp.AccountId)
    log.Printf("  Balance: %s %s", accResp.Balance.Value, accResp.Balance.CurrencyCode)
    log.Printf("  Timestamp: %s", accResp.Timestamp)
}
```

Run it:
```bash
go run test_client.go
```

## Test Scenarios

### Scenario 1: Successful Transfer Flow

1. Check initial balances
2. Execute transfer
3. Verify balances updated
4. Check transfer record created
5. Test idempotency with same key

### Scenario 2: Error Cases

Test each error condition:
- ✅ Account not found
- ✅ Insufficient funds
- ✅ Same sender and recipient
- ✅ Invalid amount (negative, zero)
- ✅ Invalid UUID format
- ✅ Missing required fields

### Scenario 3: Concurrent Transfers

Test concurrent transfers from the same account to ensure:
- No race conditions
- Balance remains consistent
- All transfers complete or fail correctly

### Scenario 4: Idempotency

1. Execute transfer with key "test-1"
2. Execute same transfer again with key "test-1"
3. Verify only ONE transfer record created
4. Verify same response returned

## Troubleshooting

### Server won't start

**Check database connection**:
```bash
psql -U postgres -d bank_db -c "SELECT 1"
```

**Check port availability**:
```bash
# Windows PowerShell
Test-NetConnection localhost -Port 50051

# Linux/Mac
nc -zv localhost 50051
```

### "account not found" error

**Verify test data loaded**:
```sql
SELECT COUNT(*) FROM accounts;
```

Should return 5 accounts. If not, run migrations:
```bash
./run-migrations.sh up
```

### "insufficient funds" but balance looks correct

Check that the amount format is correct:
- ✅ Use string: `"100.00"` not number `100`
- ✅ Include 2 decimal places: `"100.00"` not `"100"`

### Connection refused

Ensure server is running:
```bash
# Check process
ps aux | grep bank-service

# Check logs
# Server should print: "bank-service gRPC server starting on :50051"
```

## Stopping the Server

**Graceful shutdown**:
```bash
# Press Ctrl+C in the terminal running the server
```

The server will log:
```
shutting down gRPC server...
gRPC server stopped
```

**Force stop** (if graceful shutdown hangs):
```bash
# Find process ID
ps aux | grep bank-service

# Kill process
kill <PID>
```

## Cleanup

```bash
# Stop server (Ctrl+C)

# Stop PostgreSQL
docker stop bank-postgres
docker rm bank-postgres

# Or using Make
make db-stop
```

## Next Steps

1. ✅ Server is running
2. ✅ Migrations applied
3. ✅ Basic transfers working
4. 🔄 Implement TopUp method
5. 🔄 Add authentication
6. 🔄 Add monitoring/metrics
7. 🔄 Deploy to production
