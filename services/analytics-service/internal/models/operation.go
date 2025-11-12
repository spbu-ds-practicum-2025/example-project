package models

import (
	"time"
)

// OperationType represents the type of operation
type OperationType string

const (
	OperationTypeTopup    OperationType = "TOPUP"
	OperationTypeTransfer OperationType = "TRANSFER"
)

// Operation represents an account operation in the analytics system
type Operation struct {
	ID            string
	AccountID     string
	OperationType OperationType
	Timestamp     time.Time
	Amount        Amount
	SenderID      string // Only populated for TRANSFER operations
	RecipientID   string // Only populated for TRANSFER operations
}

// Amount represents a monetary amount with currency
type Amount struct {
	Value        string // Decimal value as string to preserve precision (e.g., "100.50")
	CurrencyCode string // ISO 4217 currency code (e.g., "RUB")
}
