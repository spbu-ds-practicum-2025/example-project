package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/internal/clients"
	"github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/internal/models"
	analytics_v1 "github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/proto/analytics.v1"
	bank_v1 "github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/proto/bank.v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler implements the server.ServerInterface
type Handler struct {
	bankClient      *clients.BankClient
	analyticsClient *clients.AnalyticsClient
}

// NewHandler creates a new Handler with the given bank and analytics clients
func NewHandler(bankClient *clients.BankClient, analyticsClient *clients.AnalyticsClient) *Handler {
	return &Handler{
		bankClient:      bankClient,
		analyticsClient: analyticsClient,
	}
}

// TransferBetweenAccounts handles money transfer requests
func (h *Handler) TransferBetweenAccounts(w http.ResponseWriter, r *http.Request, accountId models.AccountIdParam, params models.TransferBetweenAccountsParams) {
	// Parse request body
	var transferReq models.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&transferReq); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Failed to parse request body", err.Error())
		return
	}

	// Build gRPC request
	grpcReq := &bank_v1.TransferMoneyRequest{
		SenderId:    accountId.String(),
		RecipientId: transferReq.RecipientId.String(),
		Amount: &bank_v1.Amount{
			Value:        transferReq.Amount.Value,
			CurrencyCode: transferReq.Amount.CurrencyCode,
		},
		IdempotencyKey: params.XIdempotencyKey.String(),
	}

	// Call bank service
	grpcResp, err := h.bankClient.TransferMoney(r.Context(), grpcReq)
	if err != nil {
		handleGrpcError(w, err)
		return
	}

	// Build response
	operationID, err := uuid.Parse(grpcResp.OperationId)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "INVALID_RESPONSE", "Invalid operation ID in response", err.Error())
		return
	}

	resp := models.TransferResponse{
		OperationId: operationID,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// GetAccount is not implemented yet
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request, accountId models.AccountIdParam) {
	w.WriteHeader(http.StatusNotImplemented)
}

// GetAccountOperations retrieves the list of operations for a given account
func (h *Handler) GetAccountOperations(w http.ResponseWriter, r *http.Request, accountId models.AccountIdParam, params models.GetAccountOperationsParams) {
	// Build gRPC request
	grpcReq := &analytics_v1.ListAccountOperationsRequest{
		AccountId: accountId.String(),
	}

	// Add optional limit parameter
	if params.Limit != nil {
		grpcReq.Limit = int32(*params.Limit)
	}

	// Add optional afterId parameter
	if params.AfterId != nil {
		grpcReq.AfterId = params.AfterId.String()
	}

	// Call analytics service
	grpcResp, err := h.analyticsClient.ListAccountOperations(r.Context(), grpcReq)
	if err != nil {
		handleGrpcError(w, err)
		return
	}

	// Convert gRPC response to API response
	operations := make([]models.Operation, 0, len(grpcResp.Content))
	for _, grpcOp := range grpcResp.Content {
		// Parse operation ID
		opID, err := uuid.Parse(grpcOp.Id)
		if err != nil {
			sendErrorResponse(w, http.StatusInternalServerError, "INVALID_RESPONSE", "Invalid operation ID in response", err.Error())
			return
		}

		// Parse timestamp
		timestamp, err := time.Parse(time.RFC3339, grpcOp.Timestamp)
		if err != nil {
			sendErrorResponse(w, http.StatusInternalServerError, "INVALID_RESPONSE", "Invalid timestamp in response", err.Error())
			return
		}

		// Convert operation type
		var opType models.OperationType
		switch grpcOp.Type {
		case analytics_v1.OperationType_TOPUP:
			opType = models.Topup
		case analytics_v1.OperationType_TRANSFER:
			opType = models.Transfer
		default:
			sendErrorResponse(w, http.StatusInternalServerError, "INVALID_RESPONSE", "Unknown operation type in response", "")
			return
		}

		// Build operation
		operation := models.Operation{
			Id:        opID,
			Type:      opType,
			Timestamp: timestamp,
			Amount: models.Amount{
				Value:        grpcOp.Amount.Value,
				CurrencyCode: grpcOp.Amount.CurrencyCode,
			},
		}

		operations = append(operations, operation)
	}

	// Build response
	resp := models.GetOperationsResponse{
		Content: operations,
	}

	// Add optional afterId in response
	if grpcResp.AfterId != "" {
		afterID, err := uuid.Parse(grpcResp.AfterId)
		if err != nil {
			sendErrorResponse(w, http.StatusInternalServerError, "INVALID_RESPONSE", "Invalid after ID in response", err.Error())
			return
		}
		resp.AfterId = &afterID
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// TopUpAccount is not implemented yet
func (h *Handler) TopUpAccount(w http.ResponseWriter, r *http.Request, accountId models.AccountIdParam, params models.TopUpAccountParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// handleGrpcError converts gRPC errors to HTTP responses
func handleGrpcError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred", err.Error())
		return
	}

	switch st.Code() {
	case codes.NotFound:
		sendErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Resource not found", st.Message())
	case codes.InvalidArgument:
		sendErrorResponse(w, http.StatusBadRequest, "INVALID_ARGUMENT", "Invalid request parameters", st.Message())
	case codes.FailedPrecondition:
		sendErrorResponse(w, http.StatusBadRequest, "FAILED_PRECONDITION", "Operation cannot be performed", st.Message())
	case codes.AlreadyExists:
		sendErrorResponse(w, http.StatusConflict, "ALREADY_EXISTS", "Resource already exists", st.Message())
	default:
		sendErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred", st.Message())
	}
}

// sendErrorResponse sends an error response in the expected format
func sendErrorResponse(w http.ResponseWriter, statusCode int, code, description, details string) {
	errorResp := models.BaseError{
		Code:        code,
		Description: &details,
		Id:          uuid.New(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResp)
}
