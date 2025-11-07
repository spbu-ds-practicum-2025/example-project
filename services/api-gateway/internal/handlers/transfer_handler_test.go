package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/internal/clients"
	"github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/internal/handlers"
	"github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/internal/models"
	bank_v1 "github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/proto/bank.v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// mockBankService implements the BankServiceServer for testing
type mockBankService struct {
	bank_v1.UnimplementedBankServiceServer
	transferMoneyFunc func(context.Context, *bank_v1.TransferMoneyRequest) (*bank_v1.TransferMoneyResponse, error)
}

func (m *mockBankService) TransferMoney(ctx context.Context, req *bank_v1.TransferMoneyRequest) (*bank_v1.TransferMoneyResponse, error) {
	if m.transferMoneyFunc != nil {
		return m.transferMoneyFunc(ctx, req)
	}
	return &bank_v1.TransferMoneyResponse{
		OperationId: uuid.New().String(),
		Status:      bank_v1.TransferStatus_TRANSFER_STATUS_SUCCESS,
		Message:     "Transfer successful",
		Timestamp:   time.Now().Format(time.RFC3339),
	}, nil
}

// setupMockServer creates a mock gRPC server for testing
func setupMockServer(t *testing.T, mockService *mockBankService) (*grpc.Server, *bufconn.Listener) {
	lis := bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()
	bank_v1.RegisterBankServiceServer(grpcServer, mockService)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("Server exited with error: %v", err)
		}
	}()

	return grpcServer, lis
}

// createTestClient creates a gRPC client connected to the mock server
func createTestClient(ctx context.Context, lis *bufconn.Listener) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

func TestTransferBetweenAccounts_Success(t *testing.T) {
	// Setup mock gRPC server
	expectedOperationID := "7c9e6679-7425-40de-944b-e07fc1f90ae7"
	mockService := &mockBankService{
		transferMoneyFunc: func(ctx context.Context, req *bank_v1.TransferMoneyRequest) (*bank_v1.TransferMoneyResponse, error) {
			// Verify request
			if req.SenderId == "" {
				t.Error("SenderId should not be empty")
			}
			if req.RecipientId == "" {
				t.Error("RecipientId should not be empty")
			}
			if req.Amount == nil {
				t.Error("Amount should not be nil")
			}
			if req.Amount.Value != "100.00" {
				t.Errorf("Expected amount value 100.00, got %s", req.Amount.Value)
			}
			if req.Amount.CurrencyCode != "RUB" {
				t.Errorf("Expected currency code RUB, got %s", req.Amount.CurrencyCode)
			}
			if req.IdempotencyKey == "" {
				t.Error("IdempotencyKey should not be empty")
			}

			return &bank_v1.TransferMoneyResponse{
				OperationId: expectedOperationID,
				Status:      bank_v1.TransferStatus_TRANSFER_STATUS_SUCCESS,
				Message:     "Transfer successful",
				Timestamp:   "2025-11-08T12:00:00Z",
			}, nil
		},
	}

	grpcServer, lis := setupMockServer(t, mockService)
	defer grpcServer.Stop()

	ctx := context.Background()
	conn, err := createTestClient(ctx, lis)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}
	defer conn.Close()

	// Create bank client wrapper using the test connection
	bankClient := clients.NewBankClientFromConn(conn)
	handler := handlers.NewHandler(bankClient)

	// Create test HTTP request
	senderID := uuid.New()
	recipientID := uuid.New()
	idempotencyKey := uuid.New()

	transferReq := models.TransferRequest{
		RecipientId: recipientID,
		Amount: models.Amount{
			Value:        "100.00",
			CurrencyCode: "RUB",
		},
	}

	body, err := json.Marshal(transferReq)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/accounts/"+senderID.String()+"/transfers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Idempotency-Key", idempotencyKey.String())

	w := httptest.NewRecorder()

	handler.TransferBetweenAccounts(w, req, senderID, models.TransferBetweenAccountsParams{
		XIdempotencyKey: idempotencyKey,
	})

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp models.TransferResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.OperationId.String() != expectedOperationID {
		t.Errorf("Expected operation ID %s, got %s", expectedOperationID, resp.OperationId.String())
	}
}

func TestTransferBetweenAccounts_InvalidRequest(t *testing.T) {
	mockService := &mockBankService{}
	grpcServer, lis := setupMockServer(t, mockService)
	defer grpcServer.Stop()

	ctx := context.Background()
	conn, err := createTestClient(ctx, lis)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}
	defer conn.Close()

	bankClient := clients.NewBankClientFromConn(conn)
	handler := handlers.NewHandler(bankClient)

	senderID := uuid.New()
	idempotencyKey := uuid.New()

	// Invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/accounts/"+senderID.String()+"/transfers", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Idempotency-Key", idempotencyKey.String())

	w := httptest.NewRecorder()

	handler.TransferBetweenAccounts(w, req, senderID, models.TransferBetweenAccountsParams{
		XIdempotencyKey: idempotencyKey,
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	var errorResp models.BaseError
	if err := json.NewDecoder(w.Body).Decode(&errorResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errorResp.Code != "INVALID_REQUEST" {
		t.Errorf("Expected error code INVALID_REQUEST, got %s", errorResp.Code)
	}
}

func TestTransferBetweenAccounts_GrpcErrors(t *testing.T) {
	tests := []struct {
		name           string
		grpcError      error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "NotFound",
			grpcError:      status.Error(codes.NotFound, "account not found"),
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:           "InvalidArgument",
			grpcError:      status.Error(codes.InvalidArgument, "invalid amount"),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_ARGUMENT",
		},
		{
			name:           "FailedPrecondition",
			grpcError:      status.Error(codes.FailedPrecondition, "insufficient funds"),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "FAILED_PRECONDITION",
		},
		{
			name:           "AlreadyExists",
			grpcError:      status.Error(codes.AlreadyExists, "duplicate idempotency key"),
			expectedStatus: http.StatusConflict,
			expectedCode:   "ALREADY_EXISTS",
		},
		{
			name:           "Internal",
			grpcError:      status.Error(codes.Internal, "internal server error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockBankService{
				transferMoneyFunc: func(ctx context.Context, req *bank_v1.TransferMoneyRequest) (*bank_v1.TransferMoneyResponse, error) {
					return nil, tt.grpcError
				},
			}

			grpcServer, lis := setupMockServer(t, mockService)
			defer grpcServer.Stop()

			ctx := context.Background()
			conn, err := createTestClient(ctx, lis)
			if err != nil {
				t.Fatalf("Failed to create test client: %v", err)
			}
			defer conn.Close()

			bankClient := clients.NewBankClientFromConn(conn)
			handler := handlers.NewHandler(bankClient)

			senderID := uuid.New()
			recipientID := uuid.New()
			idempotencyKey := uuid.New()

			transferReq := models.TransferRequest{
				RecipientId: recipientID,
				Amount: models.Amount{
					Value:        "100.00",
					CurrencyCode: "RUB",
				},
			}

			body, err := json.Marshal(transferReq)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/accounts/"+senderID.String()+"/transfers", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Idempotency-Key", idempotencyKey.String())

			w := httptest.NewRecorder()

			handler.TransferBetweenAccounts(w, req, senderID, models.TransferBetweenAccountsParams{
				XIdempotencyKey: idempotencyKey,
			})

			// Verify status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}

			// Verify error response
			var errorResp models.BaseError
			if err := json.NewDecoder(w.Body).Decode(&errorResp); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			if errorResp.Code != tt.expectedCode {
				t.Errorf("Expected error code %s, got %s", tt.expectedCode, errorResp.Code)
			}
		})
	}
}

func TestTransferBetweenAccounts_IdempotencyKeyPropagation(t *testing.T) {
	expectedIdempotencyKey := uuid.New().String()

	mockService := &mockBankService{
		transferMoneyFunc: func(ctx context.Context, req *bank_v1.TransferMoneyRequest) (*bank_v1.TransferMoneyResponse, error) {
			if req.IdempotencyKey != expectedIdempotencyKey {
				t.Errorf("Expected idempotency key %s, got %s", expectedIdempotencyKey, req.IdempotencyKey)
			}
			return &bank_v1.TransferMoneyResponse{
				OperationId: uuid.New().String(),
				Status:      bank_v1.TransferStatus_TRANSFER_STATUS_SUCCESS,
				Message:     "Transfer successful",
				Timestamp:   time.Now().Format(time.RFC3339),
			}, nil
		},
	}

	grpcServer, lis := setupMockServer(t, mockService)
	defer grpcServer.Stop()

	ctx := context.Background()
	conn, err := createTestClient(ctx, lis)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}
	defer conn.Close()

	bankClient := clients.NewBankClientFromConn(conn)
	handler := handlers.NewHandler(bankClient)

	senderID := uuid.New()
	recipientID := uuid.New()
	idempotencyKeyUUID, _ := uuid.Parse(expectedIdempotencyKey)

	transferReq := models.TransferRequest{
		RecipientId: recipientID,
		Amount: models.Amount{
			Value:        "50.00",
			CurrencyCode: "RUB",
		},
	}

	body, _ := json.Marshal(transferReq)
	req := httptest.NewRequest(http.MethodPost, "/accounts/"+senderID.String()+"/transfers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Idempotency-Key", expectedIdempotencyKey)

	w := httptest.NewRecorder()

	handler.TransferBetweenAccounts(w, req, senderID, models.TransferBetweenAccountsParams{
		XIdempotencyKey: idempotencyKeyUUID,
	})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}
