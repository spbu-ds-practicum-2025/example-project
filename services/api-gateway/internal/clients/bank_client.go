package clients

import (
	"context"
	"fmt"

	bank_v1 "github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/proto/bank.v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// BankClient wraps the gRPC client for the Bank Service
type BankClient struct {
	client bank_v1.BankServiceClient
	conn   *grpc.ClientConn
}

// NewBankClient creates a new BankClient connected to the specified address
func NewBankClient(bankServiceAddr string) (*BankClient, error) {
	conn, err := grpc.NewClient(
		bankServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to bank service: %w", err)
	}

	client := bank_v1.NewBankServiceClient(conn)

	return &BankClient{
		client: client,
		conn:   conn,
	}, nil
}

// NewBankClientFromConn creates a new BankClient from an existing gRPC connection
// This is useful for testing with mock servers
func NewBankClientFromConn(conn *grpc.ClientConn) *BankClient {
	client := bank_v1.NewBankServiceClient(conn)
	return &BankClient{
		client: client,
		conn:   conn,
	}
}

// TransferMoney calls the TransferMoney RPC on the bank service
func (c *BankClient) TransferMoney(ctx context.Context, req *bank_v1.TransferMoneyRequest) (*bank_v1.TransferMoneyResponse, error) {
	return c.client.TransferMoney(ctx, req)
}

// GetAccount calls the GetAccount RPC on the bank service
func (c *BankClient) GetAccount(ctx context.Context, req *bank_v1.GetAccountRequest) (*bank_v1.GetAccountResponse, error) {
	return c.client.GetAccount(ctx, req)
}

// TopUp calls the TopUp RPC on the bank service
func (c *BankClient) TopUp(ctx context.Context, req *bank_v1.TopUpRequest) (*bank_v1.TopUpResponse, error) {
	return c.client.TopUp(ctx, req)
}

// Close closes the gRPC connection
func (c *BankClient) Close() error {
	return c.conn.Close()
}
