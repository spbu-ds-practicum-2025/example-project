package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
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

	// ===== GIVEN: Test infrastructure is set up =====
	t.Log("===== GIVEN: Setting up test infrastructure =====")

	tc, err := setupTestContext(t)
	if err != nil {
		t.Fatalf("Failed to setup test context: %v", err)
	}
	defer tc.cleanup()

	// Generate random UUIDs for test data (according to AsyncAPI spec)
	testSenderID := uuid.New().String()
	testRecipientID := uuid.New().String()
	testOperationID := uuid.New().String()
	testEventID := uuid.New().String()
	testIdempotencyKey := uuid.New().String()

	t.Logf("Test data prepared: senderID=%s, recipientID=%s, operationID=%s",
		testSenderID, testRecipientID, testOperationID)

	// ===== WHEN: Transfer event is published to RabbitMQ =====
	t.Log("===== WHEN: Publishing transfer event to RabbitMQ =====")

	if err := publishTransferEvent(tc.rabbitmqURL, testEventID, testOperationID, testSenderID, testRecipientID, testIdempotencyKey); err != nil {
		t.Fatalf("Failed to publish test event: %v", err)
	}
	t.Log("Transfer event published successfully")

	// Wait for event to be processed
	time.Sleep(3 * time.Second)

	// ===== THEN: Verify the complete flow =====
	t.Log("===== THEN: Verifying the complete flow =====")

	// 1. Verify operation is stored in ClickHouse
	t.Log("Step 1: Verifying operation is stored in ClickHouse")
	operations, err := tc.repo.ListAccountOperations(tc.ctx, testSenderID, 10, "")
	if err != nil {
		t.Fatalf("Failed to query operations from ClickHouse: %v", err)
	}

	if len(operations) == 0 {
		t.Fatal("Expected operations in ClickHouse, got none")
	}

	t.Logf("Found %d operations in ClickHouse", len(operations))

	found := false
	for _, op := range operations {
		if op.ID == testOperationID {
			found = true
			if op.AccountID != testSenderID {
				t.Errorf("Expected account ID %s, got %s", testSenderID, op.AccountID)
			}
			if op.OperationType != models.OperationTypeTransfer {
				t.Errorf("Expected operation type TRANSFER, got %s", op.OperationType)
			}
			t.Logf("✓ Operation verified in ClickHouse: %+v", op)
		}
	}

	if !found {
		t.Errorf("Operation %s not found in ClickHouse", testOperationID)
	}

	// 2. Verify operation is accessible via gRPC API
	t.Log("Step 2: Verifying operation is accessible via gRPC API")
	grpcClient, grpcConn := createGRPCClient(t, tc.grpcPort)
	defer grpcConn.Close()

	grpcResp, err := grpcClient.ListAccountOperations(tc.ctx, &pb.ListAccountOperationsRequest{
		AccountId: testSenderID,
		Limit:     10,
	})

	if err != nil {
		t.Fatalf("Failed to list operations via gRPC: %v", err)
	}

	if len(grpcResp.Content) == 0 {
		t.Fatal("Expected operations from gRPC API, got none")
	}

	t.Logf("Retrieved %d operations via gRPC API", len(grpcResp.Content))

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
			t.Logf("✓ Operation verified via gRPC: %+v", op)
		}
	}

	if !grpcFound {
		t.Errorf("Operation %s not found in gRPC response", testOperationID)
	}

	t.Log("===== ✓ Integration test PASSED: RabbitMQ → ClickHouse → gRPC API =====")
}

// testContext holds all the components needed for integration testing
type testContext struct {
	ctx                 context.Context
	clickhouseContainer *clickhouse.ClickHouseContainer
	rabbitmqContainer   testcontainers.Container
	clickhouseClient    *db.ClickHouseClient
	repo                *repository.OperationRepository
	grpcServer          *grpc.Server
	consumer            *messaging.RabbitMQConsumer
	cancelConsumer      context.CancelFunc
	rabbitmqURL         string
	grpcPort            string
}

// setupTestContext initializes all required infrastructure for integration testing
func setupTestContext(t *testing.T) (*testContext, error) {
	ctx := context.Background()
	tc := &testContext{
		ctx: ctx,
	}

	// Start ClickHouse container
	clickhouseContainer, clickhouseHost, clickhousePassword, err := startClickHouseContainer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start ClickHouse: %w", err)
	}
	tc.clickhouseContainer = clickhouseContainer
	t.Logf("ClickHouse started at: %s", clickhouseHost)

	// Start RabbitMQ container
	rabbitmqContainer, rabbitmqURL, err := startRabbitMQContainer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start RabbitMQ: %w", err)
	}
	tc.rabbitmqContainer = rabbitmqContainer
	tc.rabbitmqURL = rabbitmqURL
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
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}
	tc.clickhouseClient = clickhouseClient

	// Create schema
	if err := createSchema(ctx, clickhouseClient); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	// Initialize repository
	tc.repo = repository.NewOperationRepository(clickhouseClient)

	// Start gRPC server on a random available port
	grpcServer, grpcPort, err := startGRPCServer(t, tc.repo)
	if err != nil {
		return nil, fmt.Errorf("failed to start gRPC server: %w", err)
	}
	tc.grpcServer = grpcServer
	tc.grpcPort = grpcPort
	t.Logf("gRPC server started on port %s", tc.grpcPort)

	// Start RabbitMQ consumer
	rabbitmqCfg := config.RabbitMQConfig{
		URL:        rabbitmqURL,
		Queue:      testQueue,
		Exchange:   testExchange,
		RoutingKey: testRoutingKey,
	}

	consumer, err := messaging.NewRabbitMQConsumer(rabbitmqCfg, tc.repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ consumer: %w", err)
	}
	tc.consumer = consumer

	// Start consumer in background
	consumerCtx, cancelConsumer := context.WithCancel(ctx)
	tc.cancelConsumer = cancelConsumer

	go func() {
		if err := consumer.Start(consumerCtx); err != nil && err != context.Canceled {
			t.Logf("Consumer error: %v", err)
		}
	}()

	// Wait for consumer to initialize
	time.Sleep(2 * time.Second)

	return tc, nil
}

// cleanup releases all resources used by the test context
func (tc *testContext) cleanup() {
	if tc.cancelConsumer != nil {
		tc.cancelConsumer()
	}
	if tc.consumer != nil {
		tc.consumer.Close()
	}
	if tc.grpcServer != nil {
		tc.grpcServer.Stop()
	}
	if tc.clickhouseClient != nil {
		tc.clickhouseClient.Close()
	}
	if tc.rabbitmqContainer != nil {
		tc.rabbitmqContainer.Terminate(tc.ctx)
	}
	if tc.clickhouseContainer != nil {
		tc.clickhouseContainer.Terminate(tc.ctx)
	}
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

func startGRPCServer(t *testing.T, repo *repository.OperationRepository) (*grpc.Server, string, error) {
	grpcServer := grpcserver.NewGRPCServer()
	analyticsService := service.NewAnalyticsServiceWithRepo(repo)
	grpcserver.RegisterAnalyticsServer(grpcServer, analyticsService)

	// Listen on port 0 to get a random available port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create listener: %w", err)
	}

	// Extract the actual port that was assigned
	actualPort := listener.Addr().(*net.TCPAddr).Port
	portStr := fmt.Sprintf("%d", actualPort)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			t.Logf("gRPC server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(500 * time.Millisecond)

	return grpcServer, portStr, nil
}

func publishTransferEvent(rabbitmqURL, eventID, operationID, senderID, recipientID, idempotencyKey string) error {
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

	// Create event according to AsyncAPI specification
	now := time.Now().Format(time.RFC3339)
	event := models.TransferCompletedEvent{
		EventID:        eventID,
		EventType:      "transfer.completed",
		EventTimestamp: now,
		OperationID:    operationID,
		SenderID:       senderID,
		RecipientID:    recipientID,
		Amount: models.Amount{
			Value:        "150.50",
			CurrencyCode: "RUB",
		},
		IdempotencyKey: idempotencyKey,
		Status:         "SUCCESS",
		Timestamp:      now,
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
