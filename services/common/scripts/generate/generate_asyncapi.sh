#!/usr/bin/env bash
set -euo pipefail

# Generate Go models from AsyncAPI specification for Bank Service event publishing
# Requires: quicktype (recommended) - install with: npm install -g quicktype
#           OR asyncapi CLI - install with: npm install -g @asyncapi/cli

ROOT_DIR=$(cd "$(dirname "$0")/../.." && pwd)
SPEC_FILE="$ROOT_DIR/common/analytics-service-kafka-spec/asyncapi.yaml"
SCHEMA_FILE="$ROOT_DIR/common/analytics-service-kafka-spec/schemas/transfer_event.json"
BANK_SERVICE_DIR="$ROOT_DIR/bank-service"
EVENTS_OUT_DIR="$BANK_SERVICE_DIR/internal/events"

echo "======================================"
echo "AsyncAPI Code Generation for Go"
echo "======================================"
echo "Spec file: $SPEC_FILE"
echo "Schema file: $SCHEMA_FILE"
echo "Output directory: $EVENTS_OUT_DIR"
echo ""

# Validate AsyncAPI specification first
echo "Validating AsyncAPI specification..."
if command -v asyncapi &> /dev/null; then
    asyncapi validate "$SPEC_FILE" || {
        echo "Error: AsyncAPI specification validation failed!"
        exit 1
    }
    echo "✓ AsyncAPI specification is valid"
else
    echo "Warning: asyncapi CLI not found, skipping validation"
    echo "Install with: npm install -g @asyncapi/cli"
fi

# Create output directories if they don't exist
mkdir -p "$EVENTS_OUT_DIR"

# Try to use quicktype for Go struct generation (most reliable for Go)
echo ""
echo "Generating Go structs from JSON Schema..."
if command -v quicktype &> /dev/null; then
    echo "Using quicktype for code generation..."
    quicktype \
        --src "$SCHEMA_FILE" \
        --src-lang schema \
        --lang go \
        --package events \
        --out "$EVENTS_OUT_DIR/transfer_event.go" \
        --top-level TransferCompletedEvent \
        --just-types
    
    echo "✓ Generated Go structs: $EVENTS_OUT_DIR/transfer_event.go"
else
    echo "Warning: quicktype not found, using fallback template generation..."
    echo "For better results, install quicktype with: npm install -g quicktype"
    
    # Fallback: Generate a basic Go struct template manually
    cat > "$EVENTS_OUT_DIR/transfer_event.go" <<'EOF'
// Code generated from AsyncAPI specification. DO NOT EDIT.
package events

import "time"

// TransferCompletedEvent represents the event emitted when a money transfer is successfully completed
type TransferCompletedEvent struct {
	// Unique identifier for this event instance
	EventID string `json:"eventId"`
	
	// Type of the event
	EventType string `json:"eventType"`
	
	// Timestamp when the event was created (ISO 8601)
	EventTimestamp time.Time `json:"eventTimestamp"`
	
	// Unique identifier of the transfer operation
	OperationID string `json:"operationId"`
	
	// Unique identifier of the sender's account
	SenderID string `json:"senderId"`
	
	// Unique identifier of the recipient's account
	RecipientID string `json:"recipientId"`
	
	// The monetary amount transferred
	Amount Amount `json:"amount"`
	
	// Idempotency key used for the transfer operation
	IdempotencyKey string `json:"idempotencyKey"`
	
	// Status of the transfer operation
	Status string `json:"status"`
	
	// Timestamp when the transfer was executed (ISO 8601)
	Timestamp time.Time `json:"timestamp"`
	
	// Optional human-readable message about the transfer
	Message *string `json:"message,omitempty"`
}

// Amount represents a monetary value with its currency
type Amount struct {
	// The numeric value of the amount as a string to preserve precision
	Value string `json:"value"`
	
	// ISO 4217 currency code (e.g., "RUB" for Russian Ruble)
	CurrencyCode string `json:"currencyCode"`
}
EOF
    
    echo "✓ Generated basic Go structs (fallback): $EVENTS_OUT_DIR/transfer_event.go"
fi

# Generate additional helper files
echo ""
echo "Generating helper files..."

# Create a constants file for event types and routing keys
cat > "$EVENTS_OUT_DIR/constants.go" <<'EOF'
// Code generated from AsyncAPI specification. DO NOT EDIT.
package events

const (
	// EventTypeTransferCompleted is the event type for completed transfers
	EventTypeTransferCompleted = "transfer.completed"
	
	// RoutingKeyTransferCompleted is the RabbitMQ routing key for transfer events
	RoutingKeyTransferCompleted = "bank.operations.transfer.completed"
	
	// ExchangeName is the RabbitMQ exchange name for bank operations
	ExchangeName = "bank.operations"
	
	// ExchangeType is the type of RabbitMQ exchange
	ExchangeType = "topic"
)
EOF

echo "✓ Generated: $EVENTS_OUT_DIR/constants.go"

# Create a publisher interface
cat > "$EVENTS_OUT_DIR/publisher.go" <<'EOF'
// Code generated from AsyncAPI specification. DO NOT EDIT.
package events

import "context"

// Publisher defines the interface for publishing bank operation events
type Publisher interface {
	// PublishTransferCompleted publishes a transfer completion event
	PublishTransferCompleted(ctx context.Context, event *TransferCompletedEvent) error
	
	// Close closes the publisher and releases resources
	Close() error
}
EOF

echo "✓ Generated: $EVENTS_OUT_DIR/publisher.go"

echo ""
echo "======================================"
echo "Code generation completed successfully!"
echo "======================================"
echo "Generated files:"
echo "  - Models: $EVENTS_OUT_DIR/transfer_event.go"
echo "  - Constants: $EVENTS_OUT_DIR/constants.go"
echo "  - Publisher Interface: $EVENTS_OUT_DIR/publisher.go"
echo ""
echo "Note: You'll need to implement the Publisher interface"
echo "      with RabbitMQ/AMQP client in your service."

exit 0
