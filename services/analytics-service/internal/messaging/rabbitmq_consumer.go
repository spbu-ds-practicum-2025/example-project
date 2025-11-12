package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/config"
	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/models"
	"github.com/spbu-ds-practicum-2025/example-project/services/analytics-service/internal/repository"
)

// RabbitMQConsumer consumes transfer events from RabbitMQ
type RabbitMQConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	config  config.RabbitMQConfig
	repo    *repository.OperationRepository
}

// NewRabbitMQConsumer creates a new RabbitMQ consumer
func NewRabbitMQConsumer(cfg config.RabbitMQConfig, repo *repository.OperationRepository) (*RabbitMQConsumer, error) {
	// Connect to RabbitMQ
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Open channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange (topic exchange for routing)
	err = channel.ExchangeDeclare(
		cfg.Exchange, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	queue, err := channel.QueueDeclare(
		cfg.Queue, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange with routing key
	err = channel.QueueBind(
		queue.Name,     // queue name
		cfg.RoutingKey, // routing key
		cfg.Exchange,   // exchange
		false,
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	log.Printf("RabbitMQ consumer initialized: exchange=%s, queue=%s, routing_key=%s",
		cfg.Exchange, cfg.Queue, cfg.RoutingKey)

	return &RabbitMQConsumer{
		conn:    conn,
		channel: channel,
		config:  cfg,
		repo:    repo,
	}, nil
}

// Start begins consuming messages from the queue
func (c *RabbitMQConsumer) Start(ctx context.Context) error {
	// Register consumer
	msgs, err := c.channel.Consume(
		c.config.Queue, // queue
		"",             // consumer tag (auto-generated)
		false,          // auto-ack (we'll ack manually)
		false,          // exclusive
		false,          // no-local
		false,          // no-wait
		nil,            // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("RabbitMQ consumer started, waiting for messages on queue: %s", c.config.Queue)

	// Process messages
	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping RabbitMQ consumer")
			return nil

		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("message channel closed")
			}

			// Handle message
			if err := c.handleMessage(ctx, msg); err != nil {
				log.Printf("Error handling message: %v", err)
				// Negative acknowledgement with requeue on error
				msg.Nack(false, true)
			} else {
				// Acknowledge successful processing
				msg.Ack(false)
			}
		}
	}
}

// handleMessage processes a single transfer event message
func (c *RabbitMQConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	// Deserialize event from JSON
	var event models.TransferCompletedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	log.Printf("Received transfer event: eventId=%s, operationId=%s, sender=%s, recipient=%s",
		event.EventID, event.OperationID, event.SenderID, event.RecipientID)

	// Validate event
	if err := c.validateEvent(&event); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	// Parse timestamp from ISO 8601
	timestamp, err := time.Parse(time.RFC3339, event.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to parse timestamp: %w", err)
	}

	// Create operation for sender (outgoing transfer)
	senderOperation := &models.Operation{
		ID:            event.OperationID,
		AccountID:     event.SenderID,
		OperationType: models.OperationTypeTransfer,
		Timestamp:     timestamp,
		Amount: models.Amount{
			Value:        event.Amount.Value,
			CurrencyCode: event.Amount.CurrencyCode,
		},
		SenderID:    event.SenderID,
		RecipientID: event.RecipientID,
	}

	// Create operation for recipient (incoming transfer)
	recipientOperation := &models.Operation{
		ID:            event.OperationID,
		AccountID:     event.RecipientID,
		OperationType: models.OperationTypeTransfer,
		Timestamp:     timestamp,
		Amount: models.Amount{
			Value:        event.Amount.Value,
			CurrencyCode: event.Amount.CurrencyCode,
		},
		SenderID:    event.SenderID,
		RecipientID: event.RecipientID,
	}

	// Insert sender operation
	if err := c.repo.InsertOperation(ctx, senderOperation); err != nil {
		return fmt.Errorf("failed to insert sender operation: %w", err)
	}

	// Insert recipient operation
	if err := c.repo.InsertOperation(ctx, recipientOperation); err != nil {
		return fmt.Errorf("failed to insert recipient operation: %w", err)
	}

	log.Printf("Successfully processed transfer event: operationId=%s", event.OperationID)

	return nil
}

// validateEvent validates the transfer event structure
func (c *RabbitMQConsumer) validateEvent(event *models.TransferCompletedEvent) error {
	if event.OperationID == "" {
		return fmt.Errorf("operation ID is required")
	}
	if event.SenderID == "" {
		return fmt.Errorf("sender ID is required")
	}
	if event.RecipientID == "" {
		return fmt.Errorf("recipient ID is required")
	}
	if event.Amount.Value == "" {
		return fmt.Errorf("amount value is required")
	}
	if event.Amount.CurrencyCode == "" {
		return fmt.Errorf("currency code is required")
	}
	if event.Timestamp == "" {
		return fmt.Errorf("timestamp is required")
	}
	if event.Status != "SUCCESS" {
		return fmt.Errorf("only SUCCESS status events are processed, got: %s", event.Status)
	}

	return nil
}

// Close closes the RabbitMQ connection and channel
func (c *RabbitMQConsumer) Close() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			log.Printf("Error closing channel: %v", err)
		}
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
