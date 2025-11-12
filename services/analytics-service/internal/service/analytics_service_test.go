package service

import (
	"context"
	"testing"
	"time"

	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/models"
	pb "github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/proto/analytics.v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockOperationRepository is a mock implementation of the repository for testing
type MockOperationRepository struct {
	operations []*models.Operation
	err        error
}

func (m *MockOperationRepository) InsertOperation(ctx context.Context, op *models.Operation) error {
	if m.err != nil {
		return m.err
	}
	m.operations = append(m.operations, op)
	return nil
}

func (m *MockOperationRepository) ListAccountOperations(
	ctx context.Context,
	accountID string,
	limit int32,
	afterID string,
) ([]*models.Operation, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.operations, nil
}

func TestListAccountOperations_Success(t *testing.T) {
	// Setup mock repository with test data
	mockRepo := &MockOperationRepository{
		operations: []*models.Operation{
			{
				ID:            "op-1",
				AccountID:     "acc-1",
				OperationType: models.OperationTypeTransfer,
				Timestamp:     time.Date(2025, 11, 12, 10, 0, 0, 0, time.UTC),
				Amount: models.Amount{
					Value:        "100.00",
					CurrencyCode: "RUB",
				},
				SenderID:    "acc-1",
				RecipientID: "acc-2",
			},
		},
	}

	service := NewAnalyticsService(mockRepo)

	req := &pb.ListAccountOperationsRequest{
		AccountId: "acc-1",
		Limit:     10,
	}

	resp, err := service.ListAccountOperations(context.Background(), req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Content) != 1 {
		t.Errorf("expected 1 operation, got %d", len(resp.Content))
	}

	if resp.AfterId != "op-1" {
		t.Errorf("expected afterId to be 'op-1', got %s", resp.AfterId)
	}

	op := resp.Content[0]
	if op.Id != "op-1" {
		t.Errorf("expected operation ID to be 'op-1', got %s", op.Id)
	}

	if op.Type != pb.OperationType_TRANSFER {
		t.Errorf("expected operation type to be TRANSFER, got %v", op.Type)
	}
}

func TestListAccountOperations_EmptyAccountId(t *testing.T) {
	mockRepo := &MockOperationRepository{}
	service := NewAnalyticsService(mockRepo)

	req := &pb.ListAccountOperationsRequest{
		AccountId: "",
		Limit:     10,
	}

	_, err := service.ListAccountOperations(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for empty account_id")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error code, got %v", st.Code())
	}
}

func TestListAccountOperations_NegativeLimit(t *testing.T) {
	mockRepo := &MockOperationRepository{}
	service := NewAnalyticsService(mockRepo)

	req := &pb.ListAccountOperationsRequest{
		AccountId: "acc-1",
		Limit:     -5,
	}

	_, err := service.ListAccountOperations(context.Background(), req)

	if err == nil {
		t.Fatal("expected error for negative limit")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument error code, got %v", st.Code())
	}
}

func TestConvertToProto_Transfer(t *testing.T) {
	mockRepo := &MockOperationRepository{}
	service := NewAnalyticsService(mockRepo)

	op := &models.Operation{
		ID:            "op-1",
		AccountID:     "acc-1",
		OperationType: models.OperationTypeTransfer,
		Timestamp:     time.Date(2025, 11, 12, 10, 30, 45, 123000000, time.UTC),
		Amount: models.Amount{
			Value:        "250.50",
			CurrencyCode: "RUB",
		},
		SenderID:    "acc-1",
		RecipientID: "acc-2",
	}

	pbOp, err := service.convertToProto(op)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pbOp.Id != "op-1" {
		t.Errorf("expected ID 'op-1', got %s", pbOp.Id)
	}

	if pbOp.Type != pb.OperationType_TRANSFER {
		t.Errorf("expected type TRANSFER, got %v", pbOp.Type)
	}

	if pbOp.Timestamp != "2025-11-12T10:30:45.123Z" {
		t.Errorf("expected timestamp '2025-11-12T10:30:45.123Z', got %s", pbOp.Timestamp)
	}

	if pbOp.Amount.Value != "250.50" {
		t.Errorf("expected amount value '250.50', got %s", pbOp.Amount.Value)
	}

	transfer := pbOp.GetTransfer()
	if transfer == nil {
		t.Fatal("expected transfer details")
	}

	if transfer.SenderId != "acc-1" {
		t.Errorf("expected sender 'acc-1', got %s", transfer.SenderId)
	}

	if transfer.RecipientId != "acc-2" {
		t.Errorf("expected recipient 'acc-2', got %s", transfer.RecipientId)
	}
}

func TestConvertToProto_Topup(t *testing.T) {
	mockRepo := &MockOperationRepository{}
	service := NewAnalyticsService(mockRepo)

	op := &models.Operation{
		ID:            "op-2",
		AccountID:     "acc-1",
		OperationType: models.OperationTypeTopup,
		Timestamp:     time.Date(2025, 11, 12, 11, 0, 0, 0, time.UTC),
		Amount: models.Amount{
			Value:        "1000.00",
			CurrencyCode: "RUB",
		},
	}

	pbOp, err := service.convertToProto(op)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pbOp.Type != pb.OperationType_TOPUP {
		t.Errorf("expected type TOPUP, got %v", pbOp.Type)
	}

	topup := pbOp.GetTopup()
	if topup == nil {
		t.Fatal("expected topup details")
	}
}

func TestConvertToProto_UnknownType(t *testing.T) {
	mockRepo := &MockOperationRepository{}
	service := NewAnalyticsService(mockRepo)

	op := &models.Operation{
		ID:            "op-3",
		AccountID:     "acc-1",
		OperationType: models.OperationType("UNKNOWN"),
		Timestamp:     time.Now(),
		Amount: models.Amount{
			Value:        "100.00",
			CurrencyCode: "RUB",
		},
	}

	_, err := service.convertToProto(op)

	if err == nil {
		t.Fatal("expected error for unknown operation type")
	}
}
