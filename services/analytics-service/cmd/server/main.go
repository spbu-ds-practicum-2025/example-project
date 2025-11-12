package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/config"
	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/db"
	grpcserver "github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/grpc/server"
	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/messaging"
	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/repository"
	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/service"
)

func main() {
	log.Println("Starting Analytics Service...")

	// Load configuration
	cfg := config.Load()
	log.Printf("Configuration loaded: ClickHouse=%s:%s, RabbitMQ=%s, gRPC=:%s",
		cfg.ClickHouse.Host, cfg.ClickHouse.Database, cfg.RabbitMQ.Exchange, cfg.GRPCPort)

	// Initialize ClickHouse client
	clickhouseClient, err := db.NewClickHouseClient(cfg.ClickHouse)
	if err != nil {
		log.Fatalf("Failed to initialize ClickHouse client: %v", err)
	}
	defer clickhouseClient.Close()
	log.Println("Successfully connected to ClickHouse")

	// Initialize repository
	repo := repository.NewOperationRepository(clickhouseClient)
	log.Println("Repository initialized")

	// Initialize analytics service
	analyticsService := service.NewAnalyticsServiceWithRepo(repo)
	log.Println("Analytics service initialized")

	// Create wait group for graceful shutdown
	var wg sync.WaitGroup

	// Create context for coordinating shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := startGRPCServer(cfg, analyticsService); err != nil {
			log.Printf("gRPC server error: %v", err)
			cancel() // Signal shutdown on error
		}
	}()

	// Start RabbitMQ consumer
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := startRabbitMQConsumer(ctx, cfg, repo); err != nil {
			log.Printf("RabbitMQ consumer error: %v", err)
			cancel() // Signal shutdown on error
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v, initiating shutdown...", sig)
	case <-ctx.Done():
		log.Println("Context cancelled, initiating shutdown...")
	}

	// Cancel context to stop all goroutines
	cancel()

	// Wait for all goroutines to finish
	log.Println("Waiting for services to shutdown...")
	wg.Wait()

	log.Println("Analytics Service stopped gracefully")
}

// startGRPCServer starts the gRPC server
func startGRPCServer(cfg *config.Config, analyticsService *service.AnalyticsService) error {
	// Create TCP listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", cfg.GRPCPort, err)
	}

	// Create gRPC server
	grpcServer := grpcserver.NewGRPCServer()

	// Register analytics service
	grpcserver.RegisterAnalyticsServer(grpcServer, analyticsService)

	log.Printf("gRPC server listening on port %s", cfg.GRPCPort)

	// Start serving (blocking)
	if err := grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("gRPC server failed: %w", err)
	}

	return nil
}

// startRabbitMQConsumer starts the RabbitMQ consumer
func startRabbitMQConsumer(ctx context.Context, cfg *config.Config, repo *repository.OperationRepository) error {
	// Create consumer
	consumer, err := messaging.NewRabbitMQConsumer(cfg.RabbitMQ, repo)
	if err != nil {
		return fmt.Errorf("failed to create RabbitMQ consumer: %w", err)
	}
	defer consumer.Close()

	log.Println("RabbitMQ consumer starting...")

	// Start consuming (blocking until context is cancelled)
	if err := consumer.Start(ctx); err != nil {
		return fmt.Errorf("RabbitMQ consumer error: %w", err)
	}

	log.Println("RabbitMQ consumer stopped")
	return nil
}
