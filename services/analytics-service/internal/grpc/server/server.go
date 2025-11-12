package server

import (
	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/service"
	pb "github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/proto/analytics.v1"
	"google.golang.org/grpc"
)

// RegisterAnalyticsServer registers the analytics service with the gRPC server
func RegisterAnalyticsServer(s *grpc.Server, analyticsService *service.AnalyticsService) {
	pb.RegisterAnalyticsServiceServer(s, analyticsService)
}

// NewGRPCServer creates a new gRPC server with recommended options
func NewGRPCServer() *grpc.Server {
	// Create server with options
	opts := []grpc.ServerOption{
		// Add server options here (interceptors, limits, etc.)
		grpc.MaxRecvMsgSize(1024 * 1024 * 4), // 4MB max receive message size
		grpc.MaxSendMsgSize(1024 * 1024 * 4), // 4MB max send message size
	}

	return grpc.NewServer(opts...)
}
