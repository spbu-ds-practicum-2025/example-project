package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/internal/clients"
	"github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/internal/models"
	bank_v1 "github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/proto/bank.v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler implements the server.ServerInterface
type Handler struct {
	bankClient *clients.BankClient
}

// NewHandler creates a new Handler with the given bank client
func NewHandler(bankClient *clients.BankClient) *Handler {
	return &Handler{
		bankClient: bankClient,
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

// GetAccountOperations is not implemented yet
func (h *Handler) GetAccountOperations(w http.ResponseWriter, r *http.Request, accountId models.AccountIdParam, params models.GetAccountOperationsParams) {
	w.WriteHeader(http.StatusNotImplemented)
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
