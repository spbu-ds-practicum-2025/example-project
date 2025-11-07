package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spbu-ds-practicum-2025/example-project/services/bank-service/internal/domain"
)

// AccountRepository implements domain.AccountRepository using PostgreSQL.
type AccountRepository struct {
	pool *pgxpool.Pool
}

// NewAccountRepository creates a new AccountRepository.
func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{
		pool: pool,
	}
}

// GetByID retrieves an account by its unique identifier.
func (r *AccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	query := `
		SELECT id, balance_value, balance_currency_code, created_at, updated_at
		FROM accounts
		WHERE id = $1
	`

	var account domain.Account

	// Use transaction if available, otherwise use pool
	var row pgx.Row
	if tx := getTx(ctx); tx != nil {
		row = tx.QueryRow(ctx, query, id)
	} else {
		row = r.pool.QueryRow(ctx, query, id)
	}

	err := row.Scan(
		&account.ID,
		&account.Balance.Value,
		&account.Balance.CurrencyCode,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &account, nil
}

// Update persists changes to an existing account.
func (r *AccountRepository) Update(ctx context.Context, account *domain.Account) error {
	query := `
		UPDATE accounts
		SET balance_value = $2,
		    balance_currency_code = $3,
		    updated_at = $4
		WHERE id = $1
	`

	var err error
	var rowsAffected int64

	// Use transaction if available, otherwise use pool
	if tx := getTx(ctx); tx != nil {
		result, execErr := tx.Exec(ctx, query,
			account.ID,
			account.Balance.Value,
			account.Balance.CurrencyCode,
			account.UpdatedAt,
		)
		err = execErr
		rowsAffected = result.RowsAffected()
	} else {
		result, execErr := r.pool.Exec(ctx, query,
			account.ID,
			account.Balance.Value,
			account.Balance.CurrencyCode,
			account.UpdatedAt,
		)
		err = execErr
		rowsAffected = result.RowsAffected()
	}

	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrAccountNotFound
	}

	return nil
}

// Lock acquires a pessimistic lock on the account for the duration of the transaction.
// This method MUST be called within a transaction context.
// Uses SELECT ... FOR UPDATE to lock the row.
func (r *AccountRepository) Lock(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	query := `
		SELECT id, balance_value, balance_currency_code, created_at, updated_at
		FROM accounts
		WHERE id = $1
		FOR UPDATE
	`

	var account domain.Account

	// Use transaction if available, otherwise use pool
	var row pgx.Row
	if tx := getTx(ctx); tx != nil {
		row = tx.QueryRow(ctx, query, id)
	} else {
		row = r.pool.QueryRow(ctx, query, id)
	}

	err := row.Scan(
		&account.ID,
		&account.Balance.Value,
		&account.Balance.CurrencyCode,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to lock account: %w", err)
	}

	return &account, nil
}
