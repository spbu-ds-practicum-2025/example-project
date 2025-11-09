package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// txKey is the key type for storing transaction in context.
type txKey struct{}

// TransactionManager implements domain.TransactionManager using PostgreSQL.
type TransactionManager struct {
	pool *pgxpool.Pool
}

// NewTransactionManager creates a new TransactionManager.
func NewTransactionManager(pool *pgxpool.Pool) *TransactionManager {
	return &TransactionManager{
		pool: pool,
	}
}

// WithTransaction executes the given function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
// The transaction is stored in the context and can be retrieved using getTx.
func (tm *TransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Begin transaction
	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure transaction is closed
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			// Log rollback error, but don't override the original error
			// In production, use proper logging
			fmt.Printf("failed to rollback transaction: %v\n", err)
		}
	}()

	// Store transaction in context so repositories can use it
	txCtx := context.WithValue(ctx, txKey{}, tx)

	// Execute the function
	if err := fn(txCtx); err != nil {
		return err // Transaction will be rolled back by defer
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// getTx retrieves the transaction from context.
// If no transaction is found, returns nil.
func getTx(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return nil
}
