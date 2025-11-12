package service

import (
	"context"
	"fmt"

	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/models"
	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/repository"
	pb "github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/proto/analytics.v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OperationRepository defines the interface for operation data access
type OperationRepository interface {
	InsertOperation(ctx context.Context, op *models.Operation) error
	ListAccountOperations(ctx context.Context, accountID string, limit int32, afterID string) ([]*models.Operation, error)
}

// AnalyticsService implements the gRPC AnalyticsService interface
type AnalyticsService struct {
	pb.UnimplementedAnalyticsServiceServer
	repo OperationRepository
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(repo OperationRepository) *AnalyticsService {
	return &AnalyticsService{
		repo: repo,
	}
}

// NewAnalyticsServiceWithRepo creates a new analytics service with concrete repository
func NewAnalyticsServiceWithRepo(repo *repository.OperationRepository) *AnalyticsService {
	return &AnalyticsService{
		repo: repo,
	}
}

// ListAccountOperations returns operation history for a specific account with pagination
func (s *AnalyticsService) ListAccountOperations(
	ctx context.Context,
	req *pb.ListAccountOperationsRequest,
) (*pb.ListAccountOperationsResponse, error) {
	// Validate request
	if err := s.validateListRequest(req); err != nil {
		return nil, err
	}

	// Query operations from repository
	operations, err := s.repo.ListAccountOperations(
		ctx,
		req.AccountId,
		req.Limit,
		req.AfterId,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list operations: %v", err)
	}

	// Convert domain models to protobuf messages
	pbOperations := make([]*pb.Operation, 0, len(operations))
	var lastID string

	for _, op := range operations {
		pbOp, err := s.convertToProto(op)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert operation: %v", err)
		}

		pbOperations = append(pbOperations, pbOp)
		lastID = op.ID
	}

	return &pb.ListAccountOperationsResponse{
		Content: pbOperations,
		AfterId: lastID,
	}, nil
}

// validateListRequest validates the ListAccountOperations request
func (s *AnalyticsService) validateListRequest(req *pb.ListAccountOperationsRequest) error {
	if req.AccountId == "" {
		return status.Error(codes.InvalidArgument, "account_id is required")
	}

	if req.Limit < 0 {
		return status.Error(codes.InvalidArgument, "limit cannot be negative")
	}

	return nil
}

// convertToProto converts a domain Operation model to protobuf Operation
func (s *AnalyticsService) convertToProto(op *models.Operation) (*pb.Operation, error) {
	pbOp := &pb.Operation{
		Id:        op.ID,
		Timestamp: op.Timestamp.Format("2006-01-02T15:04:05.000Z"),
		Amount: &pb.Amount{
			Value:        op.Amount.Value,
			CurrencyCode: op.Amount.CurrencyCode,
		},
	}

	// Set operation type and details
	switch op.OperationType {
	case models.OperationTypeTransfer:
		pbOp.Type = pb.OperationType_TRANSFER
		pbOp.Details = &pb.Operation_Transfer{
			Transfer: &pb.TransferOperation{
				SenderId:    op.SenderID,
				RecipientId: op.RecipientID,
			},
		}

	case models.OperationTypeTopup:
		pbOp.Type = pb.OperationType_TOPUP
		pbOp.Details = &pb.Operation_Topup{
			Topup: &pb.TopupOperation{},
		}

	default:
		return nil, fmt.Errorf("unknown operation type: %s", op.OperationType)
	}

	return pbOp, nil
}
