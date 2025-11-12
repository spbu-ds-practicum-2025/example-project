# Analytics Service Implementation Plan

## Overview
Analytics Service is a Go microservice that:
- Consumes transfer events from RabbitMQ
- Stores operation history in ClickHouse
- Provides gRPC API for querying account operations

## Technology Stack
- **Language**: Go 1.23+
- **Database**: ClickHouse (analytical database)
- **Message Broker**: RabbitMQ (AMQP)
- **API Protocol**: gRPC

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

## Implementation Phases

### Phase 1: Project Setup
**Goal**: Initialize Go project structure and dependencies

**Tasks**:
1. Create `go.mod` with module name and Go version
2. Add required dependencies:
   - `github.com/ClickHouse/clickhouse-go/v2` - ClickHouse driver
   - `github.com/rabbitmq/amqp091-go` - RabbitMQ client
   - `google.golang.org/grpc` - gRPC framework
   - `google.golang.org/protobuf` - Protocol buffers
   - `github.com/google/uuid` - UUID generation
3. Create `README.md` with service description

**Deliverables**:
- Working Go module
- Basic project documentation

---

### Phase 2: Configuration Management
**Goal**: Implement configuration loading from environment variables

**Tasks**:
1. Create `internal/config/config.go`
2. Define configuration structs:
   - `Config` - main configuration
   - `GRPCConfig` - gRPC server settings
   - `ClickHouseConfig` - database connection
   - `RabbitMQConfig` - message broker settings
3. Implement environment variable loading with defaults

**Configuration Parameters**:
- `GRPC_PORT` - gRPC server port (default: 50053)
- `CLICKHOUSE_HOST` - ClickHouse host (default: localhost:9000)
- `CLICKHOUSE_DB` - Database name (default: analytics)
- `CLICKHOUSE_USER` - Username (default: default)
- `CLICKHOUSE_PASSWORD` - Password
- `RABBITMQ_URL` - Connection URL (default: amqp://guest:guest@localhost:5672/)
- `RABBITMQ_QUEUE` - Queue name (default: analytics.transfer.completed)
- `RABBITMQ_EXCHANGE` - Exchange name (default: bank.operations)
- `RABBITMQ_ROUTING_KEY` - Routing key (default: bank.operations.transfer.completed)

**Deliverables**:
- Configuration package
- Environment variable documentation

---

### Phase 3: Protocol Buffers Code Generation
**Goal**: Generate Go code from protobuf definitions

**Tasks**:
1. Create `proto/analytics.v1/` directory
2. Use existing proto generation scripts to generate Go code from `services/common/analytics-service-api/analytics_service.proto`

**Generated Files**:
- `proto/analytics.v1/analytics_service.pb.go` - Protocol buffer messages
- `proto/analytics.v1/analytics_service_grpc.pb.go` - gRPC service interface

**Deliverables**:
- Generated protobuf code

---

### Phase 4: Domain Models
**Goal**: Define internal domain models for operations and events

**Tasks**:
1. Create `internal/models/operation.go`:
   - `OperationType` enum (TOPUP, TRANSFER)
   - `Operation` struct with fields:
     - ID (string)
     - AccountID (string)
     - OperationType
     - Timestamp (time.Time)
     - Amount (value, currency)
     - SenderID (for transfers)
     - RecipientID (for transfers)
   - `Amount` struct (Value, CurrencyCode)

2. Create `internal/models/event.go`:
   - `TransferCompletedEvent` struct matching AsyncAPI schema:
     - EventID, EventType, EventTimestamp
     - OperationID, SenderID, RecipientID
     - Amount, IdempotencyKey, Status, Timestamp
     - Message (optional)

**Deliverables**:
- Domain models package
- Clear separation between internal models and API models

---

### Phase 5: ClickHouse Integration
**Goal**: Implement database client and schema management

**Tasks**:
1. Create `internal/db/clickhouse.go`:
   - `ClickHouseClient` struct wrapping the driver connection
   - `NewClickHouseClient()` - connection initialization
   - `InitSchema()` - create tables if not exist
   - `Close()` - cleanup

2. Define operations table schema:
   ```sql
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
   ```

3. Implement connection with:
   - Connection pooling
   - Ping/health check
   - Error handling

**Deliverables**:
- ClickHouse client package
- Database schema initialization
- Connection management

---

### Phase 6: Repository Layer
**Goal**: Implement data access layer for operations

**Tasks**:
1. Create `internal/repository/operation_repository.go`:
   - `OperationRepository` struct
   - `NewOperationRepository()` - constructor

2. Implement methods:
   - `InsertOperation(ctx, operation)` - insert single operation
     - Use prepared statements
     - Handle duplicates gracefully
   - `ListAccountOperations(ctx, accountID, limit, afterID)` - query operations
     - Support pagination via `afterID`
     - Order by timestamp DESC
     - Apply limit if provided
     - Filter by account_id

3. Add error handling:
   - Wrap errors with context
   - Log query errors
   - Handle connection issues

**Deliverables**:
- Repository package
- CRUD operations for operation data

---

### Phase 7: RabbitMQ Consumer
**Goal**: Implement event consumer for transfer events

**Tasks**:
1. Create `internal/messaging/rabbitmq_consumer.go`:
   - `RabbitMQConsumer` struct with connection, channel, config, repository
   - `NewRabbitMQConsumer()` - initialize connection and declare queue/exchange

2. Implement queue setup:
   - Declare topic exchange (durable)
   - Declare queue (durable, non-exclusive)
   - Bind queue to exchange with routing key

3. Implement `Start(ctx)` method:
   - Consume messages from queue
   - Process each message in `handleMessage()`
   - ACK on success, NACK with requeue on error
   - Support graceful shutdown via context

4. Implement `handleMessage()`:
   - Deserialize JSON event to `TransferCompletedEvent`
   - Parse timestamp from ISO 8601
   - Create two operations (one for sender, one for recipient)
   - Insert both operations into repository
   - Log processing results

5. Implement error handling:
   - Validate event structure
   - Handle deserialization errors
   - Retry on transient failures
   - Dead-letter queue for failed messages (future)

**Deliverables**:
- RabbitMQ consumer package
- Event processing logic
- Error handling and retries

---

### Phase 8: Business Logic Service
**Goal**: Implement analytics service business logic

**Tasks**:
1. Create `internal/service/analytics_service.go`:
   - `AnalyticsService` struct implementing gRPC interface
   - Embed `UnimplementedAnalyticsServiceServer`
   - Reference to `OperationRepository`

2. Implement `ListAccountOperations()` RPC:
   - Validate request (accountId required)
   - Call repository method
   - Convert domain models to protobuf messages
   - Map operation types (TRANSFER/TOPUP)
   - Set operation details (transfer or topup fields)
   - Format timestamps to ISO 8601
   - Return response with operations and afterId

3. Add validation:
   - Check required fields
   - Validate pagination parameters
   - Return gRPC errors with appropriate codes

**Deliverables**:
- Analytics service implementation
- Request/response mapping
- Business logic validation

---

### Phase 9: gRPC Server
**Goal**: Implement gRPC server setup and registration

**Tasks**:
1. Create `internal/grpc/server/server.go`:
   - `RegisterAnalyticsServer()` - register service with gRPC server
   - Server options (interceptors, connection limits, etc.)

2. Implement server features:
   - Health checks (future)
   - Reflection for debugging (optional)
   - Graceful shutdown support

**Deliverables**:
- gRPC server setup
- Service registration

---

### Phase 10: Main Application
**Goal**: Wire everything together in the main entrypoint

**Tasks**:
1. Create `cmd/server/main.go`:
   - Load configuration
   - Initialize ClickHouse client
   - Initialize repository
   - Initialize analytics service
   - Start gRPC server (in goroutine)
   - Start RabbitMQ consumer (in goroutine)
   - Handle graceful shutdown on SIGINT/SIGTERM
   - Use WaitGroup for coordinating goroutines

2. Implement startup sequence:
   - Load config
   - Connect to ClickHouse
   - Initialize database schema
   - Create repository and service
   - Start gRPC listener
   - Start RabbitMQ consumer
   - Wait for shutdown signal

3. Implement shutdown:
   - Cancel contexts
   - Stop accepting new requests
   - Wait for in-flight requests
   - Close database connections
   - Close RabbitMQ connections

**Deliverables**:
- Main application entrypoint
- Graceful startup and shutdown
- Proper error handling

---

### Phase 11: Testing
**Goal**: Implement unit and integration tests

**Tasks**:
1. Unit tests:
   - `internal/config/config_test.go` - configuration loading
   - `internal/repository/operation_repository_test.go` - repository methods
   - `internal/service/analytics_service_test.go` - business logic
   - Use mocks for dependencies

2. Integration tests:
   - Test with real ClickHouse (testcontainers)
   - Test with real RabbitMQ (testcontainers)
   - End-to-end flow: consume event → store → query

3. Test coverage:
   - Aim for >80% coverage
   - Cover error cases
   - Test edge cases (empty results, invalid data)

**Deliverables**:
- Comprehensive test suite
- Integration test setup
- CI/CD test automation

---

### Phase 12: Documentation
**Goal**: Finalize service documentation

**Tasks**:
1. Update `README.md`:
   - Architecture diagram
   - API documentation reference
   - Development setup instructions
   - Running locally
   - Environment variables
   - Testing instructions

**Deliverables**:
- Complete service documentation

---

## API Contract

### gRPC API
Defined in `services/common/analytics-service-api/analytics_service.proto`:
- **ListAccountOperations** - Returns operation history for an account with pagination

### RabbitMQ Events
Defined in `services/common/analytics-service-kafka-spec/asyncapi.yaml`:
- **Channel**: `bank.operations.transfer.completed`
- **Event**: `TransferCompletedEvent` - Published when a transfer is completed

---

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

**Design Decisions**:
- MergeTree engine for analytical queries
- Order by (account_id, timestamp) for efficient range queries
- Duplicate events handled by application logic (check before insert)
- Each transfer creates TWO records (one per account)

---

## Dependencies

### Required Go Packages
```go
require (
    github.com/ClickHouse/clickhouse-go/v2 v2.29.0
    github.com/google/uuid v1.6.0
    github.com/rabbitmq/amqp091-go v1.10.0
    google.golang.org/grpc v1.68.0
    google.golang.org/protobuf v1.35.2
)
```

### Development Tools
- `protoc` - Protocol buffer compiler
- `protoc-gen-go` - Go protobuf plugin
- `protoc-gen-go-grpc` - Go gRPC plugin
- `golangci-lint` - Linter (optional)

---

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
└── README.md                     # Service documentation
```

---

## Next Steps

1. **Start with Phase 1**: Set up the Go module and basic project structure
2. **Generate protobuf code** (Phase 3) early to understand the API contract
3. **Implement in order**: Configuration → Models → Database → Repository → Service
4. **Test incrementally**: Write tests alongside implementation
5. **Integration testing**: Test with real ClickHouse and RabbitMQ instances

---

## Future Enhancements

- **Metrics**: Prometheus metrics for monitoring
- **Tracing**: OpenTelemetry for distributed tracing
- **Health checks**: gRPC health check protocol
- **Top-up events**: Handle top-up operations when implemented
- **Aggregations**: Pre-computed analytics (daily/monthly summaries)
- **Caching**: Redis for frequently accessed data
- **Rate limiting**: Protect against query abuse
- **Authentication**: API key or JWT validation
