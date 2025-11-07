package grpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/spbu-ds-practicum-2025/example-project/services/bank-service/internal/domain"
	pb "github.com/spbu-ds-practicum-2025/example-project/services/bank-service/proto/bank.v1"
)

// BankServiceServer implements the BankService gRPC service.
type BankServiceServer struct {
	pb.UnimplementedBankServiceServer
	transferService *domain.TransferService
}

// NewBankServiceServer creates a new BankServiceServer.
func NewBankServiceServer(transferService *domain.TransferService) *BankServiceServer {
	return &BankServiceServer{
		transferService: transferService,
	}
}

// TransferMoney executes a money transfer between two accounts atomically.
// This operation is idempotent when called with the same idempotency key.
func (s *BankServiceServer) TransferMoney(ctx context.Context, req *pb.TransferMoneyRequest) (*pb.TransferMoneyResponse, error) {
	// Validate request
	if err := validateTransferMoneyRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Parse UUIDs
	senderID, err := uuid.Parse(req.SenderId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid sender_id: %v", err)
	}

	recipientID, err := uuid.Parse(req.RecipientId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid recipient_id: %v", err)
	}

	// Convert proto Amount to domain Amount
	amount := domain.Amount{
		Value:        req.Amount.Value,
		CurrencyCode: req.Amount.CurrencyCode,
	}

	// Execute transfer using domain service
	transfer, err := s.transferService.ExecuteTransfer(
		ctx,
		senderID,
		recipientID,
		amount,
		req.IdempotencyKey,
	)

	if err != nil {
		// Map domain errors to gRPC status codes
		return nil, mapDomainErrorToGRPC(err)
	}

	// Build response
	response := &pb.TransferMoneyResponse{
		OperationId: transfer.ID.String(),
		Status:      mapDomainStatusToProto(transfer.Status),
		Message:     transfer.Message,
		Timestamp:   formatTimestamp(transfer.CreatedAt),
	}

	// If transfer was completed, use completion timestamp
	if transfer.CompletedAt != nil {
		response.Timestamp = formatTimestamp(*transfer.CompletedAt)
	}

	return response, nil
}

// GetAccount retrieves complete account information including balance.
func (s *BankServiceServer) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	// Validate request
	if req.AccountId == "" {
		return nil, status.Error(codes.InvalidArgument, "account_id is required")
	}

	// Parse UUID
	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid account_id: %v", err)
	}

	// Get account from service
	account, err := s.transferService.GetAccountBalance(ctx, accountID)
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	// Build response
	response := &pb.GetAccountResponse{
		AccountId: account.ID.String(),
		Balance: &pb.Amount{
			Value:        account.Balance.Value,
			CurrencyCode: account.Balance.CurrencyCode,
		},
		Timestamp: formatTimestamp(time.Now()),
	}

	return response, nil
}

// TopUp adds funds to a specific account.
// This operation is idempotent when called with the same idempotency key.
func (s *BankServiceServer) TopUp(ctx context.Context, req *pb.TopUpRequest) (*pb.TopUpResponse, error) {
	// TopUp is not yet implemented
	return nil, status.Error(codes.Unimplemented, "TopUp operation is not yet implemented")
}

// validateTransferMoneyRequest validates the TransferMoneyRequest.
func validateTransferMoneyRequest(req *pb.TransferMoneyRequest) error {
	if req.SenderId == "" {
		return fmt.Errorf("sender_id is required")
	}
	if req.RecipientId == "" {
		return fmt.Errorf("recipient_id is required")
	}
	if req.Amount == nil {
		return fmt.Errorf("amount is required")
	}
	if req.Amount.Value == "" {
		return fmt.Errorf("amount.value is required")
	}
	if req.Amount.CurrencyCode == "" {
		return fmt.Errorf("amount.currency_code is required")
	}
	if req.IdempotencyKey == "" {
		return fmt.Errorf("idempotency_key is required")
	}
	return nil
}

// mapDomainErrorToGRPC maps domain errors to gRPC status codes.
func mapDomainErrorToGRPC(err error) error {
	if err == nil {
		return nil
	}

	// Map specific domain errors to gRPC codes
	switch {
	case errors.Is(err, domain.ErrAccountNotFound):
		return status.Error(codes.NotFound, "account not found")
	case errors.Is(err, domain.ErrInsufficientFunds):
		return status.Error(codes.FailedPrecondition, "insufficient funds")
	case errors.Is(err, domain.ErrInvalidAmount):
		return status.Error(codes.InvalidArgument, "invalid amount")
	case errors.Is(err, domain.ErrSameAccount):
		return status.Error(codes.InvalidArgument, "sender and recipient must be different")
	case errors.Is(err, domain.ErrCurrencyMismatch):
		return status.Error(codes.InvalidArgument, "currency mismatch")
	default:
		// Generic internal error
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}

// mapDomainStatusToProto maps domain transfer status to proto status.
func mapDomainStatusToProto(domainStatus domain.TransferStatus) pb.TransferStatus {
	switch domainStatus {
	case domain.TransferStatusSuccess:
		return pb.TransferStatus_TRANSFER_STATUS_SUCCESS
	case domain.TransferStatusFailed:
		return pb.TransferStatus_TRANSFER_STATUS_UNSPECIFIED // Failed maps to unspecified in proto
	case domain.TransferStatusPending:
		return pb.TransferStatus_TRANSFER_STATUS_UNSPECIFIED // Pending maps to unspecified in proto
	default:
		return pb.TransferStatus_TRANSFER_STATUS_UNSPECIFIED
	}
}

// formatTimestamp formats a time.Time to ISO 8601 format.
func formatTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
