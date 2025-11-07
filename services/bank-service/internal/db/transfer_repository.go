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

// TransferRepository implements domain.TransferRepository using PostgreSQL.
type TransferRepository struct {
	pool *pgxpool.Pool
}

// NewTransferRepository creates a new TransferRepository.
func NewTransferRepository(pool *pgxpool.Pool) *TransferRepository {
	return &TransferRepository{
		pool: pool,
	}
}

// Create persists a new transfer record.
func (r *TransferRepository) Create(ctx context.Context, transfer *domain.Transfer) error {
	query := `
		INSERT INTO transfers (
			id, sender_id, recipient_id,
			amount_value, amount_currency_code,
			idempotency_key, status, message,
			created_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	var err error

	// Use transaction if available, otherwise use pool
	if tx := getTx(ctx); tx != nil {
		_, err = tx.Exec(ctx, query,
			transfer.ID,
			transfer.SenderID,
			transfer.RecipientID,
			transfer.Amount.Value,
			transfer.Amount.CurrencyCode,
			transfer.IdempotencyKey,
			string(transfer.Status),
			transfer.Message,
			transfer.CreatedAt,
			transfer.CompletedAt,
		)
	} else {
		_, err = r.pool.Exec(ctx, query,
			transfer.ID,
			transfer.SenderID,
			transfer.RecipientID,
			transfer.Amount.Value,
			transfer.Amount.CurrencyCode,
			transfer.IdempotencyKey,
			string(transfer.Status),
			transfer.Message,
			transfer.CreatedAt,
			transfer.CompletedAt,
		)
	}

	if err != nil {
		// Check for unique constraint violation on idempotency_key
		if isPgUniqueViolation(err) {
			return fmt.Errorf("transfer with idempotency key already exists: %w", err)
		}
		return fmt.Errorf("failed to create transfer: %w", err)
	}

	return nil
}

// GetByIdempotencyKey retrieves a transfer by its idempotency key.
func (r *TransferRepository) GetByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.Transfer, error) {
	query := `
		SELECT id, sender_id, recipient_id,
		       amount_value, amount_currency_code,
		       idempotency_key, status, message,
		       created_at, completed_at
		FROM transfers
		WHERE idempotency_key = $1
	`

	var transfer domain.Transfer
	var status string

	// Use transaction if available, otherwise use pool
	var row pgx.Row
	if tx := getTx(ctx); tx != nil {
		row = tx.QueryRow(ctx, query, idempotencyKey)
	} else {
		row = r.pool.QueryRow(ctx, query, idempotencyKey)
	}

	err := row.Scan(
		&transfer.ID,
		&transfer.SenderID,
		&transfer.RecipientID,
		&transfer.Amount.Value,
		&transfer.Amount.CurrencyCode,
		&transfer.IdempotencyKey,
		&status,
		&transfer.Message,
		&transfer.CreatedAt,
		&transfer.CompletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No transfer found with this idempotency key
		}
		return nil, fmt.Errorf("failed to get transfer by idempotency key: %w", err)
	}

	transfer.Status = domain.TransferStatus(status)
	return &transfer, nil
}

// GetByID retrieves a transfer by its unique identifier.
func (r *TransferRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transfer, error) {
	query := `
		SELECT id, sender_id, recipient_id,
		       amount_value, amount_currency_code,
		       idempotency_key, status, message,
		       created_at, completed_at
		FROM transfers
		WHERE id = $1
	`

	var transfer domain.Transfer
	var status string

	// Use transaction if available, otherwise use pool
	var row pgx.Row
	if tx := getTx(ctx); tx != nil {
		row = tx.QueryRow(ctx, query, id)
	} else {
		row = r.pool.QueryRow(ctx, query, id)
	}

	err := row.Scan(
		&transfer.ID,
		&transfer.SenderID,
		&transfer.RecipientID,
		&transfer.Amount.Value,
		&transfer.Amount.CurrencyCode,
		&transfer.IdempotencyKey,
		&status,
		&transfer.Message,
		&transfer.CreatedAt,
		&transfer.CompletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("transfer not found")
		}
		return nil, fmt.Errorf("failed to get transfer by ID: %w", err)
	}

	transfer.Status = domain.TransferStatus(status)
	return &transfer, nil
}

// Update persists changes to an existing transfer.
func (r *TransferRepository) Update(ctx context.Context, transfer *domain.Transfer) error {
	query := `
		UPDATE transfers
		SET status = $2,
		    message = $3,
		    completed_at = $4
		WHERE id = $1
	`

	var err error
	var rowsAffected int64

	// Use transaction if available, otherwise use pool
	if tx := getTx(ctx); tx != nil {
		result, execErr := tx.Exec(ctx, query,
			transfer.ID,
			string(transfer.Status),
			transfer.Message,
			transfer.CompletedAt,
		)
		err = execErr
		rowsAffected = result.RowsAffected()
	} else {
		result, execErr := r.pool.Exec(ctx, query,
			transfer.ID,
			string(transfer.Status),
			transfer.Message,
			transfer.CompletedAt,
		)
		err = execErr
		rowsAffected = result.RowsAffected()
	}

	if err != nil {
		return fmt.Errorf("failed to update transfer: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transfer not found")
	}

	return nil
}

// isPgUniqueViolation checks if the error is a PostgreSQL unique constraint violation.
// PostgreSQL error code 23505 indicates unique_violation.
func isPgUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// pgx wraps errors, so we need to check the error message
	return !errors.Is(err, pgx.ErrTxClosed) &&
		!errors.Is(err, context.Canceled) &&
		containsString(err.Error(), "unique")
} // containsString checks if a string contains a substring (case-insensitive).
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 ||
			len(s) > 0 && (s[0:len(substr)] == substr || containsString(s[1:], substr)))
}
