package models

import (
	"testing"
	"time"
)

func TestOperationType_Constants(t *testing.T) {
	if OperationTypeTopup != "TOPUP" {
		t.Errorf("expected OperationTypeTopup to be 'TOPUP', got %s", OperationTypeTopup)
	}

	if OperationTypeTransfer != "TRANSFER" {
		t.Errorf("expected OperationTypeTransfer to be 'TRANSFER', got %s", OperationTypeTransfer)
	}
}

func TestOperation_Structure(t *testing.T) {
	timestamp := time.Now()

	op := &Operation{
		ID:            "test-id",
		AccountID:     "acc-123",
		OperationType: OperationTypeTransfer,
		Timestamp:     timestamp,
		Amount: Amount{
			Value:        "100.50",
			CurrencyCode: "RUB",
		},
		SenderID:    "sender-123",
		RecipientID: "recipient-456",
	}

	if op.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %s", op.ID)
	}

	if op.AccountID != "acc-123" {
		t.Errorf("expected AccountID 'acc-123', got %s", op.AccountID)
	}

	if op.OperationType != OperationTypeTransfer {
		t.Errorf("expected OperationType TRANSFER, got %s", op.OperationType)
	}

	if !op.Timestamp.Equal(timestamp) {
		t.Errorf("expected timestamp %v, got %v", timestamp, op.Timestamp)
	}

	if op.Amount.Value != "100.50" {
		t.Errorf("expected amount value '100.50', got %s", op.Amount.Value)
	}

	if op.Amount.CurrencyCode != "RUB" {
		t.Errorf("expected currency code 'RUB', got %s", op.Amount.CurrencyCode)
	}
}

func TestAmount_Structure(t *testing.T) {
	amount := Amount{
		Value:        "250.75",
		CurrencyCode: "USD",
	}

	if amount.Value != "250.75" {
		t.Errorf("expected value '250.75', got %s", amount.Value)
	}

	if amount.CurrencyCode != "USD" {
		t.Errorf("expected currency code 'USD', got %s", amount.CurrencyCode)
	}
}
