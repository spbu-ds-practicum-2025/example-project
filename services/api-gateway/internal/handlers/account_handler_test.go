package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/internal/clients"
	"github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/internal/handlers"
	"github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/internal/models"
	bank_v1 "github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/proto/bank.v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetAccount_Success(t *testing.T) {
	// Setup mock gRPC server
	accountID := uuid.New()
	expectedBalance := "150.75"
	expectedCurrency := "RUB"

	mockService := &mockBankService{
		getAccountFunc: func(ctx context.Context, req *bank_v1.GetAccountRequest) (*bank_v1.GetAccountResponse, error) {
			// Verify request
			if req.AccountId != accountID.String() {
				t.Errorf("Expected account ID %s, got %s", accountID.String(), req.AccountId)
			}

			return &bank_v1.GetAccountResponse{
				AccountId: accountID.String(),
				Balance: &bank_v1.Amount{
					Value:        expectedBalance,
					CurrencyCode: expectedCurrency,
				},
				Timestamp: "2025-12-05T10:00:00Z",
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
	handler := handlers.NewHandler(bankClient, nil)

	// Create test HTTP request
	req := httptest.NewRequest(http.MethodGet, "/accounts/"+accountID.String(), nil)
	w := httptest.NewRecorder()

	handler.GetAccount(w, req, accountID)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp models.Account
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response fields
	if resp.AccountId != accountID {
		t.Errorf("Expected account ID %s, got %s", accountID.String(), resp.AccountId.String())
	}

	if resp.Balance.Value != expectedBalance {
		t.Errorf("Expected balance value %s, got %s", expectedBalance, resp.Balance.Value)
	}

	if resp.Balance.CurrencyCode != expectedCurrency {
		t.Errorf("Expected currency code %s, got %s", expectedCurrency, resp.Balance.CurrencyCode)
	}
}

func TestGetAccount_NotFound(t *testing.T) {
	// Setup mock gRPC server
	accountID := uuid.New()

	mockService := &mockBankService{
		getAccountFunc: func(ctx context.Context, req *bank_v1.GetAccountRequest) (*bank_v1.GetAccountResponse, error) {
			return nil, status.Error(codes.NotFound, "account not found")
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
	handler := handlers.NewHandler(bankClient, nil)

	// Create test HTTP request
	req := httptest.NewRequest(http.MethodGet, "/accounts/"+accountID.String(), nil)
	w := httptest.NewRecorder()

	handler.GetAccount(w, req, accountID)

	// Verify response
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusNotFound, w.Code, w.Body.String())
	}

	var errResp models.BaseError
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Code != "NOT_FOUND" {
		t.Errorf("Expected error code NOT_FOUND, got %s", errResp.Code)
	}
}

func TestGetAccount_InvalidAccountID(t *testing.T) {
	// Setup mock gRPC server that returns invalid UUID
	accountID := uuid.New()

	mockService := &mockBankService{
		getAccountFunc: func(ctx context.Context, req *bank_v1.GetAccountRequest) (*bank_v1.GetAccountResponse, error) {
			return &bank_v1.GetAccountResponse{
				AccountId: "invalid-uuid-format",
				Balance: &bank_v1.Amount{
					Value:        "100.50",
					CurrencyCode: "RUB",
				},
				Timestamp: "2025-12-05T10:00:00Z",
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
	handler := handlers.NewHandler(bankClient, nil)

	// Create test HTTP request
	req := httptest.NewRequest(http.MethodGet, "/accounts/"+accountID.String(), nil)
	w := httptest.NewRecorder()

	handler.GetAccount(w, req, accountID)

	// Verify response - should return 500 due to invalid UUID in response
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusInternalServerError, w.Code, w.Body.String())
	}

	var errResp models.BaseError
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Code != "INVALID_RESPONSE" {
		t.Errorf("Expected error code INVALID_RESPONSE, got %s", errResp.Code)
	}
}

func TestGetAccount_InvalidArgument(t *testing.T) {
	// Setup mock gRPC server
	accountID := uuid.New()

	mockService := &mockBankService{
		getAccountFunc: func(ctx context.Context, req *bank_v1.GetAccountRequest) (*bank_v1.GetAccountResponse, error) {
			return nil, status.Error(codes.InvalidArgument, "invalid account ID format")
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
	handler := handlers.NewHandler(bankClient, nil)

	// Create test HTTP request
	req := httptest.NewRequest(http.MethodGet, "/accounts/"+accountID.String(), nil)
	w := httptest.NewRecorder()

	handler.GetAccount(w, req, accountID)

	// Verify response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}

	var errResp models.BaseError
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Code != "INVALID_ARGUMENT" {
		t.Errorf("Expected error code INVALID_ARGUMENT, got %s", errResp.Code)
	}
}
