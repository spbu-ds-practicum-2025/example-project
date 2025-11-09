package clients

import (
	context "context"
	fmt "fmt"

	analytics_v1 "github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/proto/analytics.v1"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AnalyticsClient wraps the gRPC client for the Analytics Service
type AnalyticsClient struct {
	client analytics_v1.AnalyticsServiceClient
	conn   *grpc.ClientConn
}

// NewAnalyticsClient creates a new AnalyticsClient connected to the specified address
func NewAnalyticsClient(analyticsServiceAddr string) (*AnalyticsClient, error) {
	conn, err := grpc.Dial(
		analyticsServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to analytics service: %w", err)
	}

	client := analytics_v1.NewAnalyticsServiceClient(conn)

	return &AnalyticsClient{
		client: client,
		conn:   conn,
	}, nil
}

// NewAnalyticsClientFromConn creates a new AnalyticsClient from an existing gRPC connection
// This is useful for testing with mock servers
func NewAnalyticsClientFromConn(conn *grpc.ClientConn) *AnalyticsClient {
	client := analytics_v1.NewAnalyticsServiceClient(conn)
	return &AnalyticsClient{
		client: client,
		conn:   conn,
	}
}

// ListAccountOperations calls the ListAccountOperations RPC on the analytics service
func (c *AnalyticsClient) ListAccountOperations(ctx context.Context, req *analytics_v1.ListAccountOperationsRequest) (*analytics_v1.ListAccountOperationsResponse, error) {
	return c.client.ListAccountOperations(ctx, req)
}

// Close closes the gRPC connection
func (c *AnalyticsClient) Close() error {
	return c.conn.Close()
}
