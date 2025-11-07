package domain

import (
	"context"

	"github.com/google/uuid"
)

// AccountRepository defines the interface for account data access operations.
// This follows the Repository pattern to abstract data persistence logic.
type AccountRepository interface {
	// GetByID retrieves an account by its unique identifier.
	// Returns an error if the account doesn't exist.
	GetByID(ctx context.Context, id uuid.UUID) (*Account, error)

	// Update persists changes to an existing account.
	// Typically used to update the balance after a transfer or top-up.
	Update(ctx context.Context, account *Account) error

	// Lock acquires a database lock on the account for the duration of the transaction.
	// This prevents concurrent modifications and ensures consistency.
	// Must be called within a transaction context.
	Lock(ctx context.Context, id uuid.UUID) (*Account, error)
}

// TransferRepository defines the interface for transfer data access operations.
type TransferRepository interface {
	// Create persists a new transfer record.
	// Returns an error if a transfer with the same idempotency key already exists.
	Create(ctx context.Context, transfer *Transfer) error

	// GetByIdempotencyKey retrieves a transfer by its idempotency key.
	// Used to implement idempotent transfer operations.
	// Returns nil if no transfer is found with the given key.
	GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (*Transfer, error)

	// GetByID retrieves a transfer by its unique identifier.
	GetByID(ctx context.Context, id uuid.UUID) (*Transfer, error)

	// Update persists changes to an existing transfer.
	Update(ctx context.Context, transfer *Transfer) error
}

// TransactionManager defines the interface for managing database transactions.
// This abstraction allows the service layer to work with transactions
// without being coupled to a specific database implementation.
type TransactionManager interface {
	// WithTransaction executes the given function within a database transaction.
	// If the function returns an error, the transaction is rolled back.
	// Otherwise, the transaction is committed.
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
