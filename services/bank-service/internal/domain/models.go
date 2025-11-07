package domain

import (
	"time"

	"github.com/google/uuid"
)

// Account represents a bank account in the system.
// This is the core domain entity that holds account information and balance.
type Account struct {
	ID        uuid.UUID // Unique identifier of the account
	Balance   Amount    // Current account balance
	CreatedAt time.Time // Timestamp when the account was created
	UpdatedAt time.Time // Timestamp of the last account update
}

// Transfer represents a money transfer operation between two accounts.
// This entity captures the complete transfer transaction details.
type Transfer struct {
	ID             uuid.UUID      // Unique identifier of the transfer operation
	SenderID       uuid.UUID      // Account ID of the sender (debited)
	RecipientID    uuid.UUID      // Account ID of the recipient (credited)
	Amount         Amount         // Amount transferred
	IdempotencyKey string         // Unique key to ensure idempotent operations
	Status         TransferStatus // Current status of the transfer
	Message        string         // Human-readable message about the transfer
	CreatedAt      time.Time      // Timestamp when the transfer was initiated
	CompletedAt    *time.Time     // Timestamp when the transfer was completed (nullable)
}

// Amount represents a monetary value with currency.
// Uses string for value to preserve decimal precision and avoid floating point errors.
type Amount struct {
	Value        string // Decimal string with up to 2 decimal places (e.g., "100.00")
	CurrencyCode string // ISO 4217 currency code (e.g., "RUB")
}

// TransferStatus represents the possible states of a transfer operation.
type TransferStatus string

const (
	// TransferStatusPending indicates the transfer is being processed
	TransferStatusPending TransferStatus = "PENDING"

	// TransferStatusSuccess indicates the transfer completed successfully
	TransferStatusSuccess TransferStatus = "SUCCESS"

	// TransferStatusFailed indicates the transfer failed
	TransferStatusFailed TransferStatus = "FAILED"
)

// NewAccount creates a new Account with the given ID and initial balance.
func NewAccount(id uuid.UUID, balance Amount) *Account {
	now := time.Now()
	return &Account{
		ID:        id,
		Balance:   balance,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewTransfer creates a new Transfer with the given parameters.
// The transfer is created in PENDING status.
func NewTransfer(senderID, recipientID uuid.UUID, amount Amount, idempotencyKey string) *Transfer {
	now := time.Now()
	return &Transfer{
		ID:             uuid.New(),
		SenderID:       senderID,
		RecipientID:    recipientID,
		Amount:         amount,
		IdempotencyKey: idempotencyKey,
		Status:         TransferStatusPending,
		CreatedAt:      now,
	}
}

// MarkAsSuccess marks the transfer as successfully completed.
func (t *Transfer) MarkAsSuccess(message string) {
	now := time.Now()
	t.Status = TransferStatusSuccess
	t.Message = message
	t.CompletedAt = &now
}

// MarkAsFailed marks the transfer as failed.
func (t *Transfer) MarkAsFailed(message string) {
	now := time.Now()
	t.Status = TransferStatusFailed
	t.Message = message
	t.CompletedAt = &now
}

// Debit subtracts the given amount from the account balance.
// Returns an error if the account has insufficient funds.
func (a *Account) Debit(amount Amount) error {
	if err := ValidateAmount(amount.Value); err != nil {
		return err
	}

	newBalance, err := SubtractAmounts(a.Balance.Value, amount.Value)
	if err != nil {
		return err
	}

	a.Balance.Value = newBalance
	a.UpdatedAt = time.Now()
	return nil
}

// Credit adds the given amount to the account balance.
func (a *Account) Credit(amount Amount) error {
	if err := ValidateAmount(amount.Value); err != nil {
		return err
	}

	newBalance, err := AddAmounts(a.Balance.Value, amount.Value)
	if err != nil {
		return err
	}

	a.Balance.Value = newBalance
	a.UpdatedAt = time.Now()
	return nil
}

// HasSufficientFunds checks if the account has enough balance for the given amount.
func (a *Account) HasSufficientFunds(amount Amount) bool {
	cmp, err := CompareAmounts(a.Balance.Value, amount.Value)
	if err != nil {
		return false
	}
	return cmp >= 0
}
