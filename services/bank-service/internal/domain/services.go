package domain

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	// ErrAccountNotFound is returned when an account doesn't exist
	ErrAccountNotFound = errors.New("account not found")

	// ErrInsufficientFunds is returned when the sender doesn't have enough balance
	ErrInsufficientFunds = errors.New("insufficient funds")

	// ErrInvalidAmount is returned when the transfer amount is invalid
	ErrInvalidAmount = errors.New("invalid amount: must be positive")

	// ErrSameAccount is returned when sender and recipient are the same
	ErrSameAccount = errors.New("sender and recipient must be different accounts")

	// ErrCurrencyMismatch is returned when account and transfer currencies don't match
	ErrCurrencyMismatch = errors.New("currency mismatch between accounts and transfer")
)

// TransferService handles the business logic for money transfers.
// It coordinates between repositories and ensures transactional consistency.
type TransferService struct {
	accountRepo  AccountRepository
	transferRepo TransferRepository
	txManager    TransactionManager
	// Optional event publisher to emit domain events (e.g. transfer completed)
	eventPublisher EventPublisher
}

// NewTransferService creates a new instance of TransferService.
// EventPublisher publishes domain events to external systems (e.g. RabbitMQ).
type EventPublisher interface {
	PublishTransferCompleted(ctx context.Context, transfer *Transfer) error
}

// NewTransferService creates a new instance of TransferService.
// Pass nil for eventPublisher if no events should be emitted.
func NewTransferService(
	accountRepo AccountRepository,
	transferRepo TransferRepository,
	txManager TransactionManager,
	eventPublisher EventPublisher,
) *TransferService {
	return &TransferService{
		accountRepo:    accountRepo,
		transferRepo:   transferRepo,
		txManager:      txManager,
		eventPublisher: eventPublisher,
	}
}

// ExecuteTransfer processes a money transfer from sender to recipient.
// This operation is idempotent - calling it multiple times with the same
// idempotency key will return the same result without executing the transfer again.
//
// The transfer is executed atomically within a database transaction:
// 1. Check if transfer already exists (idempotency)
// 2. Lock both accounts to prevent concurrent modifications
// 3. Validate sender has sufficient funds
// 4. Debit sender account
// 5. Credit recipient account
// 6. Create transfer record
// 7. Commit transaction
//
// Returns the created/existing transfer or an error if the operation fails.
func (s *TransferService) ExecuteTransfer(
	ctx context.Context,
	senderID uuid.UUID,
	recipientID uuid.UUID,
	amount Amount,
	idempotencyKey string,
) (*Transfer, error) {
	// Validate input parameters
	if err := s.validateTransferRequest(senderID, recipientID, amount); err != nil {
		return nil, err
	}

	// Check for existing transfer with the same idempotency key (idempotency check)
	existingTransfer, err := s.transferRepo.GetByIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check idempotency: %w", err)
	}
	if existingTransfer != nil {
		// Transfer already processed, return existing result
		return existingTransfer, nil
	}

	// Execute transfer within a transaction
	var transfer *Transfer
	err = s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// Create transfer record in PENDING status
		transfer = NewTransfer(senderID, recipientID, amount, idempotencyKey)

		// Lock accounts to prevent concurrent modifications (important for consistency)
		// Lock in a deterministic order to prevent deadlocks
		var senderAccount, recipientAccount *Account
		if senderID.String() < recipientID.String() {
			senderAccount, err = s.accountRepo.Lock(txCtx, senderID)
			if err != nil {
				return fmt.Errorf("failed to lock sender account: %w", err)
			}
			recipientAccount, err = s.accountRepo.Lock(txCtx, recipientID)
			if err != nil {
				return fmt.Errorf("failed to lock recipient account: %w", err)
			}
		} else {
			recipientAccount, err = s.accountRepo.Lock(txCtx, recipientID)
			if err != nil {
				return fmt.Errorf("failed to lock recipient account: %w", err)
			}
			senderAccount, err = s.accountRepo.Lock(txCtx, senderID)
			if err != nil {
				return fmt.Errorf("failed to lock sender account: %w", err)
			}
		}

		// Validate accounts exist
		if senderAccount == nil {
			return ErrAccountNotFound
		}
		if recipientAccount == nil {
			return ErrAccountNotFound
		}

		// Validate currency consistency
		if senderAccount.Balance.CurrencyCode != amount.CurrencyCode ||
			recipientAccount.Balance.CurrencyCode != amount.CurrencyCode {
			return ErrCurrencyMismatch
		}

		// Check sufficient funds
		if !senderAccount.HasSufficientFunds(amount) {
			transfer.MarkAsFailed("Insufficient funds")
			if err := s.transferRepo.Create(txCtx, transfer); err != nil {
				return fmt.Errorf("failed to create failed transfer record: %w", err)
			}
			return ErrInsufficientFunds
		}

		// Execute the transfer
		if err := senderAccount.Debit(amount); err != nil {
			transfer.MarkAsFailed(fmt.Sprintf("Failed to debit sender: %v", err))
			if err := s.transferRepo.Create(txCtx, transfer); err != nil {
				return fmt.Errorf("failed to create failed transfer record: %w", err)
			}
			return fmt.Errorf("failed to debit sender account: %w", err)
		}

		if err := recipientAccount.Credit(amount); err != nil {
			transfer.MarkAsFailed(fmt.Sprintf("Failed to credit recipient: %v", err))
			if err := s.transferRepo.Create(txCtx, transfer); err != nil {
				return fmt.Errorf("failed to create failed transfer record: %w", err)
			}
			return fmt.Errorf("failed to credit recipient account: %w", err)
		}

		// Update accounts in database
		if err := s.accountRepo.Update(txCtx, senderAccount); err != nil {
			return fmt.Errorf("failed to update sender account: %w", err)
		}
		if err := s.accountRepo.Update(txCtx, recipientAccount); err != nil {
			return fmt.Errorf("failed to update recipient account: %w", err)
		}

		// Mark transfer as successful
		transfer.MarkAsSuccess("Transfer completed successfully")

		// Create transfer record
		if err := s.transferRepo.Create(txCtx, transfer); err != nil {
			return fmt.Errorf("failed to create transfer record: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// After successful transaction commit, publish transfer completed event (best-effort).
	// We publish asynchronously so that transient RabbitMQ failures don't make the
	// already-committed transfer appear to fail. Production systems should use
	// a durable outbox or at-least-once delivery with retry for stronger guarantees.
	if s.eventPublisher != nil {
		// capture transfer for goroutine
		go func(t *Transfer) {
			if err := s.eventPublisher.PublishTransferCompleted(context.Background(), t); err != nil {
				// Best-effort: log the failure. Domain package doesn't have structured
				// logging; print to stderr for now. Consider replacing with a logger.
				fmt.Printf("warning: failed to publish transfer completed event: %v\n", err)
			}
		}(transfer)
	}

	return transfer, nil
}

// GetAccountBalance retrieves the current balance of an account.
func (s *TransferService) GetAccountBalance(ctx context.Context, accountID uuid.UUID) (*Account, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		return nil, ErrAccountNotFound
	}
	return account, nil
}

// validateTransferRequest validates the transfer request parameters.
func (s *TransferService) validateTransferRequest(senderID, recipientID uuid.UUID, amount Amount) error {
	// Check sender and recipient are different
	if senderID == recipientID {
		return ErrSameAccount
	}

	// Validate amount is positive
	// TODO: Implement proper decimal validation
	if amount.Value == "" || amount.Value == "0" || amount.Value == "0.00" {
		return ErrInvalidAmount
	}

	// Validate currency code
	if amount.CurrencyCode == "" {
		return errors.New("currency code is required")
	}

	return nil
}
