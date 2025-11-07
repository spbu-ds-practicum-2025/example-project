package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/spbu-ds-practicum-2025/example-project/services/bank-service/internal/db"
	"github.com/spbu-ds-practicum-2025/example-project/services/bank-service/internal/domain"
	"github.com/spbu-ds-practicum-2025/example-project/services/bank-service/internal/events"
	grpcserver "github.com/spbu-ds-practicum-2025/example-project/services/bank-service/internal/grpc"
	pb "github.com/spbu-ds-practicum-2025/example-project/services/bank-service/proto/bank.v1"
)

func main() {
	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/bank_db?sslmode=disable"
		log.Printf("DATABASE_URL not set, using default: %s", dbURL)
	}

	// Get server port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}

	ctx := context.Background()

	// Initialize database connection pool
	pool, err := db.NewPool(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to create database pool: %v", err)
	}
	defer pool.Close()
	log.Println("database connection pool initialized")

	// Create repositories
	accountRepo := db.NewAccountRepository(pool.Pool)
	transferRepo := db.NewTransferRepository(pool.Pool)
	txManager := db.NewTransactionManager(pool.Pool)

	// Create RabbitMQ publisher (optional)
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
		log.Printf("RABBITMQ_URL not set, using default: %s", rabbitURL)
	}

	// Exchange and routing key from asyncapi spec
	exchange := "bank.operations"
	routingKey := "bank.operations.transfer.completed"

	var publisher domain.EventPublisher
	rabbitPub, err := events.NewRabbitMQPublisher(rabbitURL, exchange, routingKey)
	if err != nil {
		// Best-effort: if RabbitMQ is not available, continue without publishing.
		log.Printf("warning: failed to initialize RabbitMQ publisher: %v; continuing without event publishing", err)
		rabbitPub = nil
	} else {
		publisher = rabbitPub
		// ensure we close the publisher on shutdown
		defer func() {
			if err := rabbitPub.Close(); err != nil {
				log.Printf("warning: failed to close rabbitmq publisher: %v", err)
			}
		}()
	}

	// Create domain service
	transferService := domain.NewTransferService(accountRepo, transferRepo, txManager, publisher)
	log.Println("domain services initialized")

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register BankService
	bankServiceServer := grpcserver.NewBankServiceServer(transferService)
	pb.RegisterBankServiceServer(grpcServer, bankServiceServer)

	// Register reflection service (useful for tools like grpcurl)
	reflection.Register(grpcServer)
	log.Println("gRPC services registered")

	// Start listening
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", port, err)
	}

	// Start server in a goroutine
	go func() {
		log.Printf("bank-service gRPC server starting on :%s", port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down gRPC server...")
	grpcServer.GracefulStop()
	log.Println("gRPC server stopped")
}
