# Bank Service - Domain Layer

This directory contains the core business logic and domain models for the Bank Service.

## Overview

The domain layer implements the core banking operations following Domain-Driven Design (DDD) principles. It is independent of infrastructure concerns (database, gRPC, etc.) and focuses purely on business rules.

## Components

### Models (`models.go`)

Core domain entities representing the business concepts:

- **Account**: Represents a bank account with balance and metadata
  - Fields: ID, Balance, CreatedAt, UpdatedAt
  - Methods: `Debit()`, `Credit()`, `HasSufficientFunds()`

- **Transfer**: Represents a money transfer operation between accounts
  - Fields: ID, SenderID, RecipientID, Amount, IdempotencyKey, Status, Message, CreatedAt, CompletedAt
  - Methods: `MarkAsSuccess()`, `MarkAsFailed()`

- **Amount**: Represents a monetary value with currency
  - Fields: Value (decimal string), CurrencyCode (ISO 4217)

- **TransferStatus**: Enum for transfer states
  - `PENDING`: Transfer is being processed
  - `SUCCESS`: Transfer completed successfully
  - `FAILED`: Transfer failed

### Repositories (`repositories.go`)

Repository interfaces for data access abstraction:

- **AccountRepository**: CRUD operations for accounts
  - `GetByID()`: Retrieve account by ID
  - `Update()`: Persist account changes
  - `Lock()`: Acquire pessimistic lock for transaction safety

- **TransferRepository**: CRUD operations for transfers
  - `Create()`: Create new transfer record
  - `GetByIdempotencyKey()`: Retrieve transfer for idempotency check
  - `GetByID()`: Retrieve transfer by ID
  - `Update()`: Update transfer status

- **TransactionManager**: Database transaction management
  - `WithTransaction()`: Execute operations within a transaction

### Services (`services.go`)

Business logic implementation:

- **TransferService**: Orchestrates money transfer operations
  - `ExecuteTransfer()`: Main method for processing transfers
    - Validates request parameters
    - Implements idempotency via idempotency key
    - Executes transfer atomically within a transaction
    - Locks accounts to prevent race conditions
    - Validates sufficient funds
    - Updates account balances
    - Creates transfer record
  - `GetAccountBalance()`: Retrieves account balance

### Validation (`validation.go`)

Utility functions for domain validation:

- **Amount validation**: `ValidateAmount()`, `CompareAmounts()`, `SubtractAmounts()`, `AddAmounts()`
- **Currency validation**: `ValidateCurrencyCode()`

## Key Design Patterns

### 1. Repository Pattern
Abstracts data access logic from business logic, making the domain layer independent of the database implementation.

### 2. Service Pattern
Encapsulates complex business operations that span multiple entities and repositories.

### 3. Idempotency
Transfer operations are idempotent - calling `ExecuteTransfer()` multiple times with the same `idempotencyKey` returns the same result without re-executing the transfer.

### 4. Transaction Management
All state-changing operations are wrapped in database transactions to ensure atomicity and consistency.

### 5. Pessimistic Locking
Accounts are locked during transfers to prevent concurrent modifications and race conditions. Locks are acquired in deterministic order (by UUID) to prevent deadlocks.

## Money Handling

**Note**: The current implementation uses `float64` for amount arithmetic, which is NOT recommended for production financial systems due to floating-point precision issues.

**Recommendation**: For production use, integrate a decimal library like:
- `github.com/shopspring/decimal` (most popular)
- `github.com/ericlagergren/decimal`

This would replace the arithmetic functions in `validation.go` with precise decimal operations.

## Error Handling

The domain layer defines specific error types for different failure scenarios:

- `ErrAccountNotFound`: Account doesn't exist
- `ErrInsufficientFunds`: Sender has insufficient balance
- `ErrInvalidAmount`: Transfer amount is invalid
- `ErrSameAccount`: Sender and recipient are the same
- `ErrCurrencyMismatch`: Currency mismatch between accounts

## Usage Example

```go
// Create repositories (implementation in db layer)
accountRepo := db.NewAccountRepository(pool)
transferRepo := db.NewTransferRepository(pool)
txManager := db.NewTransactionManager(pool)

// Create service
transferService := domain.NewTransferService(accountRepo, transferRepo, txManager)

// Execute transfer
transfer, err := transferService.ExecuteTransfer(
    ctx,
    senderID,
    recipientID,
    domain.Amount{Value: "100.00", CurrencyCode: "RUB"},
    "idempotency-key-123",
)
if err != nil {
    // Handle error
}
```

## Testing Considerations

The domain layer is designed to be easily testable:

1. **Repository interfaces** can be mocked for unit testing services
2. **No external dependencies** (database, gRPC) in the domain layer
3. **Pure business logic** that can be tested in isolation

Example test structure:
```go
func TestTransferService_ExecuteTransfer(t *testing.T) {
    // Create mock repositories
    mockAccountRepo := &MockAccountRepository{}
    mockTransferRepo := &MockTransferRepository{}
    mockTxManager := &MockTransactionManager{}
    
    // Create service
    service := domain.NewTransferService(mockAccountRepo, mockTransferRepo, mockTxManager)
    
    // Test scenarios...
}
```

## Future Enhancements

1. **Decimal arithmetic**: Replace float64 with proper decimal library
2. **Event sourcing**: Emit domain events for transfers (for Analytics Service)
3. **Saga pattern**: For distributed transactions across services
4. **Rate limiting**: Add transfer limits per account
5. **Multi-currency**: Enhanced support for currency conversion
