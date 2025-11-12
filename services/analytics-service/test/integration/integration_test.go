package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/config"
	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/db"
	grpcserver "github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/grpc/server"
	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/messaging"
	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/models"
	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/repository"
	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/service"
	pb "github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/proto/analytics.v1"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/clickhouse"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	testExchange   = "test.bank.operations"
	testQueue      = "test.analytics.transfer.completed"
	testRoutingKey = "test.bank.operations.transfer.completed"
)

func TestFullIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start ClickHouse container
	clickhouseContainer, clickhouseHost, clickhousePassword, err := startClickHouseContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start ClickHouse container: %v", err)
	}
	defer clickhouseContainer.Terminate(ctx)

	t.Logf("ClickHouse started at: %s", clickhouseHost)

	// Start RabbitMQ container
	rabbitmqContainer, rabbitmqURL, err := startRabbitMQContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start RabbitMQ container: %v", err)
	}
	defer rabbitmqContainer.Terminate(ctx)

	t.Logf("RabbitMQ started at: %s", rabbitmqURL)

	// Initialize ClickHouse client
	clickhouseCfg := config.ClickHouseConfig{
		Host:     clickhouseHost,
		Database: "default",
		User:     "default",
		Password: clickhousePassword,
	}

	clickhouseClient, err := db.NewClickHouseClient(clickhouseCfg)
	if err != nil {
		t.Fatalf("Failed to connect to ClickHouse: %v", err)
	}
	defer clickhouseClient.Close()

	// Create schema
	if err := createSchema(ctx, clickhouseClient); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Initialize repository
	repo := repository.NewOperationRepository(clickhouseClient)

	// Start gRPC server
	grpcPort := "50053"
	grpcServer, err := startGRPCServer(t, grpcPort, repo)
	if err != nil {
		t.Fatalf("Failed to start gRPC server: %v", err)
	}
	if grpcServer != nil {
		defer grpcServer.Stop()
	}

	t.Logf("gRPC server started on port %s", grpcPort)

	// Start RabbitMQ consumer
	rabbitmqCfg := config.RabbitMQConfig{
		URL:        rabbitmqURL,
		Queue:      testQueue,
		Exchange:   testExchange,
		RoutingKey: testRoutingKey,
	}

	consumer, err := messaging.NewRabbitMQConsumer(rabbitmqCfg, repo)
	if err != nil {
		t.Fatalf("Failed to create RabbitMQ consumer: %v", err)
	}
	defer consumer.Close()

	consumerCtx, cancelConsumer := context.WithCancel(ctx)
	defer cancelConsumer()

	// Start consumer in background
	go func() {
		if err := consumer.Start(consumerCtx); err != nil && err != context.Canceled {
			t.Logf("Consumer error: %v", err)
		}
	}()

	// Wait for consumer to initialize
	time.Sleep(2 * time.Second)

	// Publish test events to RabbitMQ
	testAccountID := "acc-123"
	testOperationID := "op-456"

	if err := publishTransferEvent(rabbitmqURL, testAccountID, testOperationID); err != nil {
		t.Fatalf("Failed to publish test event: %v", err)
	}

	t.Log("Published transfer event to RabbitMQ")

	// Wait for event to be processed
	time.Sleep(3 * time.Second)

	// Verify operations in ClickHouse
	operations, err := repo.ListAccountOperations(ctx, testAccountID, 10, "")
	if err != nil {
		t.Fatalf("Failed to query operations from ClickHouse: %v", err)
	}

	if len(operations) == 0 {
		t.Fatal("Expected operations in ClickHouse, got none")
	}

	t.Logf("Found %d operations in ClickHouse", len(operations))

	// Verify operation details
	found := false
	for _, op := range operations {
		if op.ID == testOperationID {
			found = true
			if op.AccountID != testAccountID {
				t.Errorf("Expected account ID %s, got %s", testAccountID, op.AccountID)
			}
			if op.OperationType != models.OperationTypeTransfer {
				t.Errorf("Expected operation type TRANSFER, got %s", op.OperationType)
			}
			t.Logf("Verified operation: %+v", op)
		}
	}

	if !found {
		t.Errorf("Operation %s not found in ClickHouse", testOperationID)
	}

	// Verify operations via gRPC API
	grpcClient, grpcConn := createGRPCClient(t, grpcPort)
	defer grpcConn.Close()

	grpcResp, err := grpcClient.ListAccountOperations(ctx, &pb.ListAccountOperationsRequest{
		AccountId: testAccountID,
		Limit:     10,
	})

	if err != nil {
		t.Fatalf("Failed to list operations via gRPC: %v", err)
	}

	if len(grpcResp.Content) == 0 {
		t.Fatal("Expected operations from gRPC API, got none")
	}

	t.Logf("Retrieved %d operations via gRPC API", len(grpcResp.Content))

	// Verify gRPC response
	grpcFound := false
	for _, op := range grpcResp.Content {
		if op.Id == testOperationID {
			grpcFound = true
			if op.Type != pb.OperationType_TRANSFER {
				t.Errorf("Expected TRANSFER type, got %v", op.Type)
			}
			if op.Amount.Value != "150.50" {
				t.Errorf("Expected amount 150.50, got %s", op.Amount.Value)
			}
			if op.Amount.CurrencyCode != "RUB" {
				t.Errorf("Expected currency RUB, got %s", op.Amount.CurrencyCode)
			}
			t.Logf("Verified gRPC operation: %+v", op)
		}
	}

	if !grpcFound {
		t.Errorf("Operation %s not found in gRPC response", testOperationID)
	}

	t.Log("✓ Integration test passed: RabbitMQ → ClickHouse → gRPC API")
}

func startClickHouseContainer(ctx context.Context) (*clickhouse.ClickHouseContainer, string, string, error) {
	clickhouseContainer, err := clickhouse.Run(ctx,
		"clickhouse/clickhouse-server:23.3.8.21-alpine",
		clickhouse.WithUsername("default"),
		clickhouse.WithPassword("clickhouse"),
		clickhouse.WithDatabase("default"),
	)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to start ClickHouse container: %w", err)
	}

	host, err := clickhouseContainer.ConnectionHost(ctx)
	if err != nil {
		return nil, "", "", err
	}

	return clickhouseContainer, host, "clickhouse", nil
}

func startRabbitMQContainer(ctx context.Context) (testcontainers.Container, string, error) {
	rabbitmqContainer, err := rabbitmq.Run(ctx,
		"rabbitmq:3.13-management",
		rabbitmq.WithAdminUsername("guest"),
		rabbitmq.WithAdminPassword("guest"),
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to start RabbitMQ container: %w", err)
	}

	connectionString, err := rabbitmqContainer.AmqpURL(ctx)
	if err != nil {
		return nil, "", err
	}

	return rabbitmqContainer, connectionString, nil
}

func createSchema(ctx context.Context, client *db.ClickHouseClient) error {
	query := `
	CREATE TABLE IF NOT EXISTS operations (
		id String,
		account_id String,
		operation_type Enum8('TOPUP' = 1, 'TRANSFER' = 2),
		timestamp DateTime64(3),
		amount_value Decimal(18, 2),
		amount_currency String,
		sender_id String,
		recipient_id String,
		created_at DateTime DEFAULT now()
	) ENGINE = MergeTree()
	ORDER BY (account_id, timestamp)
	PRIMARY KEY (account_id, timestamp)
	`

	return client.Conn().Exec(ctx, query)
}

func startGRPCServer(t *testing.T, port string, repo *repository.OperationRepository) (*grpc.Server, error) {
	grpcServer := grpcserver.NewGRPCServer()
	analyticsService := service.NewAnalyticsServiceWithRepo(repo)
	grpcserver.RegisterAnalyticsServer(grpcServer, analyticsService)

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			t.Logf("gRPC server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(500 * time.Millisecond)

	return grpcServer, nil
}

func publishTransferEvent(rabbitmqURL, accountID, operationID string) error {
	conn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer ch.Close()

	event := models.TransferCompletedEvent{
		EventID:        "evt-123",
		EventType:      "transfer.completed",
		EventTimestamp: time.Now().Format(time.RFC3339),
		OperationID:    operationID,
		SenderID:       accountID,
		RecipientID:    "acc-789",
		Amount: models.Amount{
			Value:        "150.50",
			CurrencyCode: "RUB",
		},
		IdempotencyKey: "idem-123",
		Status:         "SUCCESS",
		Timestamp:      time.Now().Format(time.RFC3339),
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = ch.Publish(
		testExchange,   // exchange
		testRoutingKey, // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func createGRPCClient(t *testing.T, port string) (pb.AnalyticsServiceClient, *grpc.ClientConn) {
	conn, err := grpc.Dial(
		"localhost:"+port,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial gRPC server: %v", err)
	}

	client := pb.NewAnalyticsServiceClient(conn)
	return client, conn
}
