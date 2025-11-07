# Database Layer - Repository Implementations

This directory contains the PostgreSQL repository implementations for the Bank Service domain models.

## Overview

The database layer implements the repository interfaces defined in the domain layer using direct SQL with the `pgx` PostgreSQL driver. No ORM is used - all database operations are performed using raw SQL for maximum control and performance.

## Components

### Connection Pool (`pool.go`)

Manages PostgreSQL connection pooling using `pgxpool`.

**Key Features**:
- Connection pool with configurable min/max connections
- Automatic connection health checking
- Connection ping on initialization

**Usage**:
```go
pool, err := db.NewPool(ctx, "postgres://user:pass@localhost:5432/bank_db?sslmode=disable")
if err != nil {
    log.Fatal(err)
}
defer pool.Close()
```

**Configuration**:
- `MaxConns`: 25 (maximum concurrent connections)
- `MinConns`: 5 (minimum idle connections)

### Account Repository (`account_repository.go`)

Implements `domain.AccountRepository` for account data access.

**Methods**:
- `GetByID(ctx, id)` - Retrieve account by UUID
- `Update(ctx, account)` - Update account balance and metadata
- `Lock(ctx, id)` - Acquire pessimistic lock (SELECT ... FOR UPDATE)

**SQL Queries**:
```sql
-- GetByID
SELECT id, balance_value, balance_currency_code, created_at, updated_at
FROM accounts WHERE id = $1

-- Update
UPDATE accounts 
SET balance_value = $2, balance_currency_code = $3, updated_at = $4
WHERE id = $1

-- Lock (for transactions)
SELECT id, balance_value, balance_currency_code, created_at, updated_at
FROM accounts WHERE id = $1 FOR UPDATE
```

**Transaction Support**:
- Automatically uses transaction from context if available
- Falls back to connection pool for non-transactional operations

### Transfer Repository (`transfer_repository.go`)

Implements `domain.TransferRepository` for transfer data access.

**Methods**:
- `Create(ctx, transfer)` - Insert new transfer record
- `GetByIdempotencyKey(ctx, key)` - Retrieve transfer by idempotency key
- `GetByID(ctx, id)` - Retrieve transfer by UUID
- `Update(ctx, transfer)` - Update transfer status and completion time

**SQL Queries**:
```sql
-- Create
INSERT INTO transfers (
    id, sender_id, recipient_id,
    amount_value, amount_currency_code,
    idempotency_key, status, message,
    created_at, completed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)

-- GetByIdempotencyKey
SELECT id, sender_id, recipient_id, amount_value, amount_currency_code,
       idempotency_key, status, message, created_at, completed_at
FROM transfers WHERE idempotency_key = $1

-- Update
UPDATE transfers 
SET status = $2, message = $3, completed_at = $4
WHERE id = $1
```

**Idempotency**:
- `GetByIdempotencyKey` returns `nil` if no transfer found (not an error)
- `Create` detects unique constraint violations for duplicate idempotency keys

### Transaction Manager (`transaction.go`)

Implements `domain.TransactionManager` for database transaction management.

**Method**:
- `WithTransaction(ctx, fn)` - Executes function within a database transaction

**Key Features**:
- Automatic transaction begin/commit/rollback
- Transaction stored in context for repository access
- Proper error handling and cleanup

**Usage**:
```go
txManager := db.NewTransactionManager(pool.Pool)

err := txManager.WithTransaction(ctx, func(txCtx context.Context) error {
    // All repository operations use txCtx
    account, err := accountRepo.Lock(txCtx, accountID)
    if err != nil {
        return err
    }
    
    account.Balance.Value = "1000.00"
    if err := accountRepo.Update(txCtx, account); err != nil {
        return err
    }
    
    return nil // Commit
})
// Transaction automatically rolled back if error returned
```

**Context Pattern**:
- Transaction stored in context using private `txKey` type
- Repositories check context for transaction before executing queries
- If transaction found: use `tx.QueryRow()`, `tx.Exec()`
- If no transaction: use `pool.QueryRow()`, `pool.Exec()`

## Architecture Patterns

### 1. Context-Based Transaction Propagation

Transactions are propagated through context:

```go
type txKey struct{}

// Store transaction in context
txCtx := context.WithValue(ctx, txKey{}, tx)

// Retrieve transaction from context
func getTx(ctx context.Context) pgx.Tx {
    if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
        return tx
    }
    return nil
}
```

### 2. Transaction-Aware Repositories

All repositories check for transaction in context:

```go
var row pgx.Row
if tx := getTx(ctx); tx != nil {
    row = tx.QueryRow(ctx, query, args...)  // Use transaction
} else {
    row = r.pool.QueryRow(ctx, query, args...)  // Use pool
}
```

This allows repositories to work both inside and outside transactions transparently.

### 3. Error Handling

- `pgx.ErrNoRows` → Domain-specific errors (e.g., `domain.ErrAccountNotFound`)
- Unique constraint violations → Specific error messages
- Generic database errors → Wrapped with context

### 4. No ORM - Raw SQL

**Why No ORM?**:
- **Performance**: Direct SQL is faster, no query generation overhead
- **Control**: Full control over queries, indexes, and optimizations
- **Transparency**: Easy to see exactly what SQL is executed
- **Type Safety**: Go's static typing provides compile-time checks
- **Debugging**: SQL queries are visible and easily testable

**Trade-offs**:
- More boilerplate code (manual mapping between DB and domain models)
- Schema changes require manual updates to queries
- No automatic migrations from model changes

## Database Mapping

### Account Mapping

```
Domain Model (Go)          Database (PostgreSQL)
─────────────────────────  ────────────────────────
Account.ID                 accounts.id (UUID)
Account.Balance.Value      accounts.balance_value (NUMERIC(15,2))
Account.Balance.Currency   accounts.balance_currency_code (VARCHAR(3))
Account.CreatedAt          accounts.created_at (TIMESTAMP WITH TIME ZONE)
Account.UpdatedAt          accounts.updated_at (TIMESTAMP WITH TIME ZONE)
```

### Transfer Mapping

```
Domain Model (Go)          Database (PostgreSQL)
─────────────────────────  ────────────────────────
Transfer.ID                transfers.id (UUID)
Transfer.SenderID          transfers.sender_id (UUID)
Transfer.RecipientID       transfers.recipient_id (UUID)
Transfer.Amount.Value      transfers.amount_value (NUMERIC(15,2))
Transfer.Amount.Currency   transfers.amount_currency_code (VARCHAR(3))
Transfer.IdempotencyKey    transfers.idempotency_key (VARCHAR(255))
Transfer.Status            transfers.status (VARCHAR(20))
Transfer.Message           transfers.message (TEXT)
Transfer.CreatedAt         transfers.created_at (TIMESTAMP WITH TIME ZONE)
Transfer.CompletedAt       transfers.completed_at (TIMESTAMP WITH TIME ZONE)
```

## Usage Example

### Complete Transfer Flow

```go
package main

import (
    "context"
    "github.com/spbu-ds-practicum-2025/example-project/services/bank-service/internal/db"
    "github.com/spbu-ds-practicum-2025/example-project/services/bank-service/internal/domain"
)

func main() {
    ctx := context.Background()
    
    // 1. Create connection pool
    pool, err := db.NewPool(ctx, "postgres://postgres:postgres@localhost:5432/bank_db?sslmode=disable")
    if err != nil {
        panic(err)
    }
    defer pool.Close()
    
    // 2. Create repositories
    accountRepo := db.NewAccountRepository(pool.Pool)
    transferRepo := db.NewTransferRepository(pool.Pool)
    txManager := db.NewTransactionManager(pool.Pool)
    
    // 3. Create service
    transferService := domain.NewTransferService(accountRepo, transferRepo, txManager)
    
    // 4. Execute transfer
    senderID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
    recipientID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
    amount := domain.Amount{Value: "100.00", CurrencyCode: "RUB"}
    
    transfer, err := transferService.ExecuteTransfer(
        ctx,
        senderID,
        recipientID,
        amount,
        "idempotency-key-123",
    )
    
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Transfer completed: %s\n", transfer.ID)
}
```

## Testing

### Unit Testing Repositories

Use test database or mocks:

```go
func TestAccountRepository_GetByID(t *testing.T) {
    // Setup test database
    pool := setupTestDB(t)
    defer pool.Close()
    
    repo := db.NewAccountRepository(pool.Pool)
    
    // Insert test data
    accountID := uuid.New()
    insertTestAccount(t, pool, accountID)
    
    // Test
    account, err := repo.GetByID(context.Background(), accountID)
    assert.NoError(t, err)
    assert.Equal(t, accountID, account.ID)
}
```

### Integration Testing

Use `testcontainers` for real PostgreSQL:

```go
import "github.com/testcontainers/testcontainers-go/modules/postgres"

func TestTransferFlow(t *testing.T) {
    ctx := context.Background()
    
    // Start PostgreSQL container
    pgContainer, err := postgres.RunContainer(ctx, ...)
    require.NoError(t, err)
    defer pgContainer.Terminate(ctx)
    
    // Run migrations
    connString, err := pgContainer.ConnectionString(ctx)
    // ... apply migrations ...
    
    // Create pool and repositories
    pool, _ := db.NewPool(ctx, connString)
    // ... run tests ...
}
```

## Performance Considerations

1. **Connection Pooling**: Reuse connections, avoid connection overhead
2. **Prepared Statements**: pgx automatically uses prepared statements
3. **Indexes**: All queries use indexes (see `migrations/SCHEMA.md`)
4. **Pessimistic Locking**: `FOR UPDATE` prevents race conditions but may impact throughput
5. **Transaction Isolation**: PostgreSQL default (READ COMMITTED) is sufficient for most cases

## Production Checklist

- ✅ Use connection pooling (implemented)
- ✅ Enable SSL/TLS (`sslmode=require` in production)
- ✅ Use prepared statements (pgx does this automatically)
- ✅ Set appropriate pool size based on load testing
- ✅ Monitor slow queries with `pg_stat_statements`
- ✅ Set up database backups
- ✅ Use read replicas for read-heavy workloads (future)
- ✅ Implement connection retry logic (future)
- ✅ Add query timeouts via context

## Future Enhancements

1. **Query Timeouts**: Add context deadlines for all queries
2. **Metrics**: Export Prometheus metrics for query latency
3. **Read Replicas**: Support read-only replicas for balance queries
4. **Batch Operations**: Batch insert/update for analytics
5. **Soft Deletes**: Add deleted_at column for audit trail
6. **Optimistic Locking**: Version field for conflict detection
