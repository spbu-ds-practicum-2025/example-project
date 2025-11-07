# gRPC Server Implementation

This directory contains the gRPC server implementation for the Bank Service.

## Overview

The gRPC layer implements the `BankService` gRPC service defined in the protocol buffers specification. It acts as an adapter between the gRPC protocol and the domain layer, translating gRPC requests/responses to/from domain models.

## Architecture

```
gRPC Request (Protobuf)
        ↓
BankServiceServer (adapter)
        ↓
Domain Service (business logic)
        ↓
Repository (database)
        ↓
PostgreSQL Database
```

## Components

### server.go

Implements the `BankServiceServer` which satisfies the `pb.BankServiceServer` interface.

**Implemented Methods**:

#### 1. TransferMoney
Executes a money transfer between two accounts atomically.

**Request** (`TransferMoneyRequest`):
- `sender_id` (string): UUID of the sender's account
- `recipient_id` (string): UUID of the recipient's account
- `amount` (Amount): Transfer amount with currency
- `idempotency_key` (string): Unique key for idempotency

**Response** (`TransferMoneyResponse`):
- `operation_id` (string): UUID of the created transfer
- `status` (TransferStatus): SUCCESS or UNSPECIFIED
- `message` (string): Human-readable result message
- `timestamp` (string): ISO 8601 timestamp

**Features**:
- ✅ Request validation (all required fields checked)
- ✅ UUID parsing and validation
- ✅ Idempotency support (same key = same result)
- ✅ Domain error mapping to gRPC status codes
- ✅ Proper timestamp formatting (ISO 8601)

**Error Mapping**:
```
Domain Error              → gRPC Status Code
─────────────────────────   ──────────────────
ErrAccountNotFound        → NOT_FOUND
ErrInsufficientFunds      → FAILED_PRECONDITION
ErrInvalidAmount          → INVALID_ARGUMENT
ErrSameAccount            → INVALID_ARGUMENT
ErrCurrencyMismatch       → INVALID_ARGUMENT
Other errors              → INTERNAL
```

#### 2. GetAccount
Retrieves complete account information including balance.

**Request** (`GetAccountRequest`):
- `account_id` (string): UUID of the account to query

**Response** (`GetAccountResponse`):
- `account_id` (string): UUID of the account
- `balance` (Amount): Current account balance
- `timestamp` (string): ISO 8601 timestamp

**Features**:
- ✅ Account lookup by UUID
- ✅ Balance retrieval
- ✅ Error handling for non-existent accounts

#### 3. TopUp
**Status**: Not yet implemented (returns `UNIMPLEMENTED` status)

This method will be implemented in future iterations to support adding funds to accounts.

## Request Validation

All requests undergo comprehensive validation:

```go
func validateTransferMoneyRequest(req *pb.TransferMoneyRequest) error {
    // Check required fields
    if req.SenderId == "" { return error }
    if req.RecipientId == "" { return error }
    if req.Amount == nil { return error }
    if req.Amount.Value == "" { return error }
    if req.Amount.CurrencyCode == "" { return error }
    if req.IdempotencyKey == "" { return error }
    return nil
}
```

Invalid requests return `INVALID_ARGUMENT` status code with descriptive error messages.

## Error Handling Strategy

### 1. Validation Errors
```go
if err := validateTransferMoneyRequest(req); err != nil {
    return nil, status.Error(codes.InvalidArgument, err.Error())
}
```

### 2. UUID Parsing Errors
```go
senderID, err := uuid.Parse(req.SenderId)
if err != nil {
    return nil, status.Errorf(codes.InvalidArgument, "invalid sender_id: %v", err)
}
```

### 3. Domain Errors
```go
transfer, err := s.transferService.ExecuteTransfer(...)
if err != nil {
    return nil, mapDomainErrorToGRPC(err)
}
```

The `mapDomainErrorToGRPC` function translates domain-specific errors to appropriate gRPC status codes.

## Data Mapping

### Proto Amount ↔ Domain Amount
```go
// Proto → Domain
amount := domain.Amount{
    Value:        req.Amount.Value,
    CurrencyCode: req.Amount.CurrencyCode,
}

// Domain → Proto
&pb.Amount{
    Value:        account.Balance.Value,
    CurrencyCode: account.Balance.CurrencyCode,
}
```

### Proto Status ↔ Domain Status
```go
func mapDomainStatusToProto(domainStatus domain.TransferStatus) pb.TransferStatus {
    switch domainStatus {
    case domain.TransferStatusSuccess:
        return pb.TransferStatus_TRANSFER_STATUS_SUCCESS
    case domain.TransferStatusFailed, domain.TransferStatusPending:
        return pb.TransferStatus_TRANSFER_STATUS_UNSPECIFIED
    default:
        return pb.TransferStatus_TRANSFER_STATUS_UNSPECIFIED
    }
}
```

**Note**: The proto only defines `SUCCESS` status. Failed and pending transfers map to `UNSPECIFIED`.

## Server Initialization

The server is initialized in `cmd/server/main.go`:

```go
// 1. Create database pool
pool, _ := db.NewPool(ctx, dbURL)

// 2. Create repositories
accountRepo := db.NewAccountRepository(pool.Pool)
transferRepo := db.NewTransferRepository(pool.Pool)
txManager := db.NewTransactionManager(pool.Pool)

// 3. Create domain service
transferService := domain.NewTransferService(accountRepo, transferRepo, txManager)

// 4. Create gRPC server
grpcServer := grpc.NewServer()

// 5. Register BankService
bankServiceServer := grpcserver.NewBankServiceServer(transferService)
pb.RegisterBankServiceServer(grpcServer, bankServiceServer)

// 6. Enable reflection (for grpcurl, etc.)
reflection.Register(grpcServer)

// 7. Start serving
grpcServer.Serve(lis)
```

## Running the Server

### Using Environment Variables

```bash
# Set database connection
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/bank_db?sslmode=disable"

# Set server port (optional, defaults to 50051)
export PORT="50051"

# Run the server
go run ./cmd/server/main.go
```

### Using Make

```bash
make run
```

### Using Docker

```bash
docker build -t bank-service .
docker run -p 50051:50051 \
  -e DATABASE_URL="postgres://postgres:postgres@host.docker.internal:5432/bank_db?sslmode=disable" \
  bank-service
```

## Testing the Server

### Using grpcurl

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List services (requires reflection to be enabled)
grpcurl -plaintext localhost:50051 list

# Describe service
grpcurl -plaintext localhost:50051 describe bank.v1.BankService

# Call TransferMoney
grpcurl -plaintext -d '{
  "sender_id": "11111111-1111-1111-1111-111111111111",
  "recipient_id": "22222222-2222-2222-2222-222222222222",
  "amount": {
    "value": "100.00",
    "currency_code": "RUB"
  },
  "idempotency_key": "test-key-123"
}' localhost:50051 bank.v1.BankService/TransferMoney

# Call GetAccount
grpcurl -plaintext -d '{
  "account_id": "11111111-1111-1111-1111-111111111111"
}' localhost:50051 bank.v1.BankService/GetAccount
```

### Using Go Client

```go
package main

import (
    "context"
    "log"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    
    pb "github.com/spbu-ds-practicum-2025/example-project/services/bank-service/proto/bank.v1"
)

func main() {
    conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    
    client := pb.NewBankServiceClient(conn)
    
    // Transfer money
    resp, err := client.TransferMoney(context.Background(), &pb.TransferMoneyRequest{
        SenderId:       "11111111-1111-1111-1111-111111111111",
        RecipientId:    "22222222-2222-2222-2222-222222222222",
        Amount: &pb.Amount{
            Value:        "100.00",
            CurrencyCode: "RUB",
        },
        IdempotencyKey: "test-key-456",
    })
    
    if err != nil {
        log.Fatalf("TransferMoney failed: %v", err)
    }
    
    log.Printf("Transfer successful: %s", resp.OperationId)
}
```

## Status Codes Reference

| gRPC Code          | When Used                                    | Example                                  |
|--------------------|----------------------------------------------|------------------------------------------|
| OK                 | Successful operation                         | Transfer completed successfully          |
| INVALID_ARGUMENT   | Invalid request parameters                   | Missing sender_id, invalid UUID format   |
| NOT_FOUND          | Resource not found                           | Account doesn't exist                    |
| FAILED_PRECONDITION| Business rule violation                      | Insufficient funds                       |
| INTERNAL           | Unexpected server error                      | Database connection error                |
| UNIMPLEMENTED      | Feature not yet implemented                  | TopUp operation                          |

## Graceful Shutdown

The server supports graceful shutdown on SIGINT/SIGTERM signals:

```go
// Wait for interrupt signal
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

// Gracefully stop the server
log.Println("shutting down gRPC server...")
grpcServer.GracefulStop()
log.Println("gRPC server stopped")
```

This ensures:
- In-flight requests complete before shutdown
- No abrupt connection termination
- Clean resource cleanup

## Logging

The server logs important events:
- ✅ Database pool initialization
- ✅ Service registration
- ✅ Server startup
- ✅ Graceful shutdown

For production, consider integrating structured logging (e.g., `zap`, `logrus`).

## Production Considerations

### Security
- ✅ **TLS**: Enable TLS for production (use `grpc.Creds()`)
- ✅ **Authentication**: Add authentication middleware (JWT, mTLS)
- ✅ **Rate Limiting**: Implement rate limiting per client
- ✅ **Input Validation**: Already implemented for all requests

### Performance
- ✅ **Connection Pooling**: Database pool configured (5-25 connections)
- ✅ **Timeouts**: Add context timeouts for all operations
- ✅ **Metrics**: Export Prometheus metrics
- ✅ **Tracing**: Add OpenTelemetry tracing

### Reliability
- ✅ **Graceful Shutdown**: Already implemented
- ✅ **Health Checks**: Implement gRPC health checking protocol
- ✅ **Circuit Breakers**: Add circuit breakers for database
- ✅ **Retries**: Client-side retry logic with exponential backoff

### Example: Adding TLS

```go
creds, err := credentials.NewServerTLSFromFile("server.crt", "server.key")
if err != nil {
    log.Fatal(err)
}

grpcServer := grpc.NewServer(grpc.Creds(creds))
```

### Example: Adding Timeouts

```go
grpcServer := grpc.NewServer(
    grpc.ConnectionTimeout(10 * time.Second),
)
```

## Future Enhancements

1. **TopUp Implementation**: Implement the TopUp RPC method
2. **Metrics**: Export Prometheus metrics (requests, latency, errors)
3. **Tracing**: Add OpenTelemetry distributed tracing
4. **Interceptors**: Add logging, authentication, and monitoring interceptors
5. **Health Checks**: Implement gRPC health checking protocol
6. **Rate Limiting**: Add per-client rate limiting
7. **Request IDs**: Generate and propagate request IDs for tracing
8. **Validation Library**: Use `protoc-gen-validate` for declarative validation
