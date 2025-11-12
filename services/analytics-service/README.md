# Analytics Service

Analytics Service is a Go microservice that provides operations history and analytics for the Electronic Wallet system.

## Overview

The Analytics Service:
- **Consumes events** from RabbitMQ (transfer completion events from Bank Service)
- **Stores operation history** in ClickHouse for fast analytical queries
- **Provides gRPC API** for querying account operations with pagination

## Architecture

```
┌─────────────┐         ┌──────────────┐
│ RabbitMQ    │────────>│  Analytics   │
│ (Events)    │         │  Service     │
└─────────────┘         └──────────────┘
                              │
                              │ Store
                              ▼
                        ┌──────────────┐
                        │ ClickHouse   │
                        │ (Analytics)  │
                        └──────────────┘
                              │
                              │ Query
                              ▼
                        ┌──────────────┐
                        │  gRPC API    │
                        └──────────────┘
```

## Technology Stack

- **Language**: Go 1.23+
- **Database**: ClickHouse (analytical database)
- **Message Broker**: RabbitMQ (AMQP)
- **API Protocol**: gRPC

## API

### gRPC API

Defined in `services/common/analytics-service-api/analytics_service.proto`:

- **ListAccountOperations** - Returns operation history for a specific account with optional pagination

### Event Consumption

Defined in `services/common/analytics-service-kafka-spec/asyncapi.yaml`:

- **Channel**: `bank.operations.transfer.completed`
- **Event**: `TransferCompletedEvent` - Published when a money transfer is completed

## Development

### Prerequisites

- Go 1.23 or higher
- ClickHouse (for data storage)
- RabbitMQ (for event consumption)
- Protocol Buffer compiler (`protoc`) with Go plugins

### Setup

1. **Clone the repository**
   ```bash
   cd services/analytics-service
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Generate protobuf code**
   
   Use the existing proto generation scripts in `services/common/scripts/generate/`

4. **Configure environment variables**

   See Configuration section below for available options.

### Building

```bash
go build -o bin/server ./cmd/server
```

### Running

```bash
./bin/server
```

Or run directly:

```bash
go run ./cmd/server
```

## Configuration

The service is configured via environment variables:

### gRPC Server
- `GRPC_PORT` - gRPC server port (default: `50053`)

### ClickHouse
- `CLICKHOUSE_HOST` - ClickHouse host and port (default: `localhost:9000`)
- `CLICKHOUSE_DB` - Database name (default: `analytics`)
- `CLICKHOUSE_USER` - Username (default: `default`)
- `CLICKHOUSE_PASSWORD` - Password (default: empty)

### RabbitMQ
- `RABBITMQ_URL` - Connection URL (default: `amqp://guest:guest@localhost:5672/`)
- `RABBITMQ_QUEUE` - Queue name (default: `analytics.transfer.completed`)
- `RABBITMQ_EXCHANGE` - Exchange name (default: `bank.operations`)
- `RABBITMQ_ROUTING_KEY` - Routing key (default: `bank.operations.transfer.completed`)

## Testing

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Project Structure

```
analytics-service/
├── cmd/
│   └── server/
│       └── main.go              # Application entrypoint
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── db/
│   │   └── clickhouse.go        # ClickHouse client
│   ├── models/
│   │   ├── operation.go         # Domain models
│   │   └── event.go             # Event models
│   ├── repository/
│   │   └── operation_repository.go  # Data access layer
│   ├── service/
│   │   └── analytics_service.go # Business logic
│   ├── messaging/
│   │   └── rabbitmq_consumer.go # Event consumer
│   └── grpc/
│       └── server/
│           └── server.go        # gRPC server setup
├── proto/
│   └── analytics.v1/            # Generated protobuf code
├── go.mod                        # Go module definition
├── go.sum                        # Dependency checksums
└── README.md                     # This file
```

## Database Schema

### Operations Table (ClickHouse)

```sql
CREATE TABLE operations (
    id String,                    -- Operation UUID
    account_id String,            -- Account UUID (indexed)
    operation_type Enum8,         -- TOPUP or TRANSFER
    timestamp DateTime64(3),      -- Operation timestamp (indexed)
    amount_value Decimal(18, 2),  -- Amount value
    amount_currency String,       -- Currency code (e.g., RUB)
    sender_id String,             -- Sender account (for transfers)
    recipient_id String,          -- Recipient account (for transfers)
    created_at DateTime           -- Record creation time
) ENGINE = MergeTree()
ORDER BY (account_id, timestamp)
PRIMARY KEY (account_id, timestamp)
```

## License

See the LICENSE file in the project root.
