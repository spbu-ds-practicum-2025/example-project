package grpc_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/spbu-ds-practicum-2025/example-project/services/bank-service/internal/db"
	"github.com/spbu-ds-practicum-2025/example-project/services/bank-service/internal/domain"
	"github.com/spbu-ds-practicum-2025/example-project/services/bank-service/internal/events"
	grpcserver "github.com/spbu-ds-practicum-2025/example-project/services/bank-service/internal/grpc"
	pb "github.com/spbu-ds-practicum-2025/example-project/services/bank-service/proto/bank.v1"
)

const bufSize = 1024 * 1024

// TestTransferMoneyIntegration is a full end-to-end integration test.
// It spins up PostgreSQL and RabbitMQ containers, runs migrations,
// starts a gRPC server, executes a transfer, and verifies the event
// was published to RabbitMQ.
func TestTransferMoneyIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, dbURL := startPostgresContainer(t, ctx)
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	}()

	// Start RabbitMQ container
	rabbitContainer, rabbitURL := startRabbitMQContainer(t, ctx)
	defer func() {
		if err := rabbitContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate rabbitmq container: %v", err)
		}
	}()

	// Initialize database pool
	pool, err := db.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to create database pool: %v", err)
	}
	defer pool.Close()

	// Run migrations
	runMigrations(t, ctx, pool)

	// Create test accounts
	senderID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	recipientID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	createTestAccounts(t, ctx, pool, senderID, recipientID)

	// Initialize RabbitMQ publisher
	exchange := "bank.operations"
	routingKey := "bank.operations.transfer.completed"
	publisher, err := events.NewRabbitMQPublisher(rabbitURL, exchange, routingKey)
	if err != nil {
		t.Fatalf("failed to create rabbitmq publisher: %v", err)
	}
	defer publisher.Close()

	// Create domain service and gRPC server
	accountRepo := db.NewAccountRepository(pool.Pool)
	transferRepo := db.NewTransferRepository(pool.Pool)
	txManager := db.NewTransactionManager(pool.Pool)
	transferService := domain.NewTransferService(accountRepo, transferRepo, txManager, publisher)
	bankServer := grpcserver.NewBankServiceServer(transferService)

	// Start in-memory gRPC server using bufconn
	lis := bufconn.Listen(bufSize)
	grpcSrv := grpc.NewServer()
	pb.RegisterBankServiceServer(grpcSrv, bankServer)

	go func() {
		if err := grpcSrv.Serve(lis); err != nil {
			t.Logf("grpc server error: %v", err)
		}
	}()
	defer grpcSrv.Stop()

	// Create gRPC client
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewBankServiceClient(conn)

	// Setup RabbitMQ consumer to capture published events
	eventChan := make(chan map[string]interface{}, 1)
	stopConsumer := startEventConsumer(t, ctx, rabbitURL, exchange, routingKey, eventChan)
	defer stopConsumer()

	// Give consumer a moment to start
	time.Sleep(500 * time.Millisecond)

	// Execute transfer via gRPC
	idempotencyKey := uuid.New().String()
	transferReq := &pb.TransferMoneyRequest{
		SenderId:       senderID.String(),
		RecipientId:    recipientID.String(),
		Amount:         &pb.Amount{Value: "100.50", CurrencyCode: "RUB"},
		IdempotencyKey: idempotencyKey,
	}

	resp, err := client.TransferMoney(ctx, transferReq)
	if err != nil {
		t.Fatalf("TransferMoney failed: %v", err)
	}

	// Verify response
	if resp.Status != pb.TransferStatus_TRANSFER_STATUS_SUCCESS {
		t.Errorf("expected status SUCCESS, got %v", resp.Status)
	}
	if resp.OperationId == "" {
		t.Error("expected non-empty operation_id")
	}

	// Verify balances changed
	senderResp, err := client.GetAccount(ctx, &pb.GetAccountRequest{AccountId: senderID.String()})
	if err != nil {
		t.Fatalf("GetAccount for sender failed: %v", err)
	}
	if senderResp.Balance.Value != "899.50" {
		t.Errorf("expected sender balance 899.50, got %s", senderResp.Balance.Value)
	}

	recipientResp, err := client.GetAccount(ctx, &pb.GetAccountRequest{AccountId: recipientID.String()})
	if err != nil {
		t.Fatalf("GetAccount for recipient failed: %v", err)
	}
	if recipientResp.Balance.Value != "600.50" {
		t.Errorf("expected recipient balance 600.50, got %s", recipientResp.Balance.Value)
	}

	// Wait for event to be published and consumed
	select {
	case event := <-eventChan:
		// Validate event structure per asyncapi spec
		if event["eventType"] != "transfer.completed" {
			t.Errorf("expected eventType 'transfer.completed', got %v", event["eventType"])
		}
		if event["operationId"] != resp.OperationId {
			t.Errorf("expected operationId %s, got %v", resp.OperationId, event["operationId"])
		}
		if event["senderId"] != senderID.String() {
			t.Errorf("expected senderId %s, got %v", senderID.String(), event["senderId"])
		}
		if event["recipientId"] != recipientID.String() {
			t.Errorf("expected recipientId %s, got %v", recipientID.String(), event["recipientId"])
		}
		if event["idempotencyKey"] != idempotencyKey {
			t.Errorf("expected idempotencyKey %s, got %v", idempotencyKey, event["idempotencyKey"])
		}
		if event["status"] != "SUCCESS" {
			t.Errorf("expected status SUCCESS, got %v", event["status"])
		}

		// Check amount
		amount, ok := event["amount"].(map[string]interface{})
		if !ok {
			t.Fatal("amount is not a map")
		}
		if amount["value"] != "100.50" {
			t.Errorf("expected amount value 100.50, got %v", amount["value"])
		}
		if amount["currencyCode"] != "RUB" {
			t.Errorf("expected currency RUB, got %v", amount["currencyCode"])
		}

	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event to be published")
	}

	// Test idempotency: call again with same idempotency key
	resp2, err := client.TransferMoney(ctx, transferReq)
	if err != nil {
		t.Fatalf("second TransferMoney call failed: %v", err)
	}
	if resp2.OperationId != resp.OperationId {
		t.Errorf("idempotent call returned different operation_id: %s vs %s", resp.OperationId, resp2.OperationId)
	}

	// Verify balances didn't change on idempotent call
	senderResp2, _ := client.GetAccount(ctx, &pb.GetAccountRequest{AccountId: senderID.String()})
	if senderResp2.Balance.Value != "899.50" {
		t.Errorf("sender balance changed on idempotent call: %s", senderResp2.Balance.Value)
	}
}

// startPostgresContainer starts a PostgreSQL testcontainer and returns the connection URL.
func startPostgresContainer(t *testing.T, ctx context.Context) (testcontainers.Container, string) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get postgres host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get postgres port: %v", err)
	}

	dbURL := fmt.Sprintf("postgres://testuser:testpass@%s:%s/testdb?sslmode=disable", host, port.Port())
	return container, dbURL
}

// startRabbitMQContainer starts a RabbitMQ testcontainer and returns the AMQP URL.
func startRabbitMQContainer(t *testing.T, ctx context.Context) (testcontainers.Container, string) {
	req := testcontainers.ContainerRequest{
		Image:        "rabbitmq:3-management",
		ExposedPorts: []string{"5672/tcp", "15672/tcp"},
		Env: map[string]string{
			"RABBITMQ_DEFAULT_USER": "guest",
			"RABBITMQ_DEFAULT_PASS": "guest",
		},
		WaitingFor: wait.ForLog("Server startup complete"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start rabbitmq container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get rabbitmq host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5672")
	if err != nil {
		t.Fatalf("failed to get rabbitmq port: %v", err)
	}

	rabbitURL := fmt.Sprintf("amqp://guest:guest@%s:%s/", host, port.Port())
	return container, rabbitURL
}

// runMigrations runs the database migrations.
func runMigrations(t *testing.T, ctx context.Context, pool *db.Pool) {
	// Run migration SQL directly (same as migration files)
	migrations := []string{
		// 001_create_accounts_table.up.sql
		`CREATE TABLE IF NOT EXISTS accounts (
			id UUID PRIMARY KEY,
			balance_value NUMERIC(15, 2) NOT NULL,
			balance_currency_code VARCHAR(3) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);`,
		// 002_create_transfers_table.up.sql
		`CREATE TABLE IF NOT EXISTS transfers (
			id UUID PRIMARY KEY,
			sender_id UUID NOT NULL REFERENCES accounts(id),
			recipient_id UUID NOT NULL REFERENCES accounts(id),
			amount_value NUMERIC(15, 2) NOT NULL,
			amount_currency_code VARCHAR(3) NOT NULL,
			idempotency_key VARCHAR(255) NOT NULL UNIQUE,
			status VARCHAR(20) NOT NULL,
			message TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			completed_at TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_transfers_sender_id ON transfers(sender_id);
		CREATE INDEX IF NOT EXISTS idx_transfers_recipient_id ON transfers(recipient_id);
		CREATE INDEX IF NOT EXISTS idx_transfers_idempotency_key ON transfers(idempotency_key);`,
		// 003_create_triggers.up.sql
		`CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = NOW();
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		DROP TRIGGER IF EXISTS update_accounts_updated_at ON accounts;
		CREATE TRIGGER update_accounts_updated_at
			BEFORE UPDATE ON accounts
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column();`,
	}

	for i, migration := range migrations {
		if _, err := pool.Pool.Exec(ctx, migration); err != nil {
			t.Fatalf("failed to run migration %d: %v", i+1, err)
		}
	}
}

// createTestAccounts creates test accounts with initial balances.
func createTestAccounts(t *testing.T, ctx context.Context, pool *db.Pool, senderID, recipientID uuid.UUID) {
	accounts := []struct {
		id      uuid.UUID
		balance string
	}{
		{senderID, "1000.00"},
		{recipientID, "500.00"},
	}

	for _, acc := range accounts {
		query := `INSERT INTO accounts (id, balance_value, balance_currency_code, created_at, updated_at)
				  VALUES ($1, $2, $3, NOW(), NOW())`
		if _, err := pool.Pool.Exec(ctx, query, acc.id, acc.balance, "RUB"); err != nil {
			t.Fatalf("failed to create test account %s: %v", acc.id, err)
		}
	}
}

// startEventConsumer starts a RabbitMQ consumer that listens for events and sends them to the channel.
func startEventConsumer(t *testing.T, ctx context.Context, rabbitURL, exchange, routingKey string, eventChan chan map[string]interface{}) func() {
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		t.Fatalf("failed to connect to rabbitmq: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		t.Fatalf("failed to open channel: %v", err)
	}

	// Declare exchange
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		t.Fatalf("failed to declare exchange: %v", err)
	}

	// Declare exclusive queue for testing
	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		t.Fatalf("failed to declare queue: %v", err)
	}

	// Bind queue to exchange with routing key
	if err := ch.QueueBind(q.Name, routingKey, exchange, false, nil); err != nil {
		ch.Close()
		conn.Close()
		t.Fatalf("failed to bind queue: %v", err)
	}

	// Start consuming
	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		t.Fatalf("failed to start consuming: %v", err)
	}

	// Consume messages in background
	go func() {
		for msg := range msgs {
			var event map[string]interface{}
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				t.Logf("failed to unmarshal event: %v", err)
				continue
			}
			eventChan <- event
		}
	}()

	// Return cleanup function
	return func() {
		ch.Close()
		conn.Close()
	}
}
