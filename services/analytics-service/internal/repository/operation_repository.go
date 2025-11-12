package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/db"
	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/models"
)

// OperationRepository handles operations data persistence in ClickHouse
type OperationRepository struct {
	db *db.ClickHouseClient
}

// NewOperationRepository creates a new operation repository
func NewOperationRepository(db *db.ClickHouseClient) *OperationRepository {
	return &OperationRepository{db: db}
}

// InsertOperation inserts a new operation into the database
func (r *OperationRepository) InsertOperation(ctx context.Context, op *models.Operation) error {
	query := `
		INSERT INTO operations (
			id, account_id, operation_type, timestamp,
			amount_value, amount_currency, sender_id, recipient_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	err := r.db.Conn().Exec(ctx, query,
		op.ID,
		op.AccountID,
		string(op.OperationType),
		op.Timestamp,
		op.Amount.Value,
		op.Amount.CurrencyCode,
		op.SenderID,
		op.RecipientID,
	)

	if err != nil {
		return fmt.Errorf("failed to insert operation %s: %w", op.ID, err)
	}

	return nil
}

// ListAccountOperations retrieves operations for a specific account with pagination
func (r *OperationRepository) ListAccountOperations(
	ctx context.Context,
	accountID string,
	limit int32,
	afterID string,
) ([]*models.Operation, error) {
	query := `
		SELECT 
			id, account_id, operation_type, timestamp,
			toString(amount_value) as amount_value, amount_currency, sender_id, recipient_id
		FROM operations
		WHERE account_id = ?
	`

	args := []interface{}{accountID}

	// Add pagination filter if afterID is provided
	if afterID != "" {
		query += " AND id > ?"
		args = append(args, afterID)
	}

	// Order by timestamp descending (most recent first)
	query += " ORDER BY timestamp DESC"

	// Apply limit if provided
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := r.db.Conn().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query operations for account %s: %w", accountID, err)
	}
	defer rows.Close()

	var operations []*models.Operation

	for rows.Next() {
		var op models.Operation
		var timestamp time.Time
		var operationType string
		var amountValue string

		err := rows.Scan(
			&op.ID,
			&op.AccountID,
			&operationType,
			&timestamp,
			&amountValue,
			&op.Amount.CurrencyCode,
			&op.SenderID,
			&op.RecipientID,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan operation row: %w", err)
		}

		op.Timestamp = timestamp
		op.OperationType = models.OperationType(operationType)

		// Format the amount to always have 2 decimal places
		// Parse and reformat to ensure consistent decimal places
		if amountValue != "" {
			// ClickHouse toString() may return "150.5" instead of "150.50"
			// We need to ensure 2 decimal places
			var amountFloat float64
			if _, err := fmt.Sscanf(amountValue, "%f", &amountFloat); err == nil {
				op.Amount.Value = fmt.Sprintf("%.2f", amountFloat)
			} else {
				op.Amount.Value = amountValue
			}
		}

		operations = append(operations, &op)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating operation rows: %w", err)
	}

	return operations, nil
}
