# Bank Operations Event Specifications

This directory contains the AsyncAPI specification and JSON schemas for bank operation events published from the Bank Service to the Analytics Service via RabbitMQ.

## Overview

The Bank Service publishes events to RabbitMQ whenever money transfers are completed. The Analytics Service consumes these events for analysis, reporting, and data warehousing in ClickHouse.

## Event Types

### Transfer Completed Event (`transfer.completed`)

Published when a money transfer between two accounts is successfully completed.

**Routing Key**: `bank.operations.transfer.completed`  
**Schema**: [`schemas/transfer_event.json`](./schemas/transfer_event.json)

**Key Fields**:
- `operationId`: Unique identifier for the transfer
- `senderId`: Account that sent the money
- `recipientId`: Account that received the money
- `amount`: Monetary amount transferred
- `idempotencyKey`: Key ensuring exactly-once processing
- `timestamp`: When the transfer was executed

**Example**:
```json
{
  "eventId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "eventType": "transfer.completed",
  "eventTimestamp": "2025-11-08T14:30:00.000Z",
  "operationId": "987e6543-e21b-34d3-c456-426614174999",
  "senderId": "123e4567-e89b-12d3-a456-426614174000",
  "recipientId": "987e6543-e21b-34d3-c456-426614174999",
  "amount": {
    "value": "150.50",
    "currencyCode": "RUB"
  },
  "idempotencyKey": "550e8400-e29b-41d4-a716-446655440000",
  "status": "SUCCESS",
  "timestamp": "2025-11-08T14:30:00.000Z"
}
```

## RabbitMQ Configuration

### Exchange

- **Name**: `bank.operations`
- **Type**: `topic`
- **Durable**: `true`
- **Auto-Delete**: `false`
- **VHost**: `/`

### Routing Keys

- **Transfer Events**: `bank.operations.transfer.completed`

### Consumer Configuration

Analytics Service consumers should:
- **ACK Mode**: Manual acknowledgment (`ack: true`)
- **Binding**: Subscribe to routing keys via topic exchange
- Create durable queues for transfer events (e.g., `analytics.transfers`)

## Message Format

All messages are published in JSON format (`application/json`) and follow a consistent structure:

- **Event Metadata**: `eventId`, `eventType`, `eventTimestamp`
- **Operation Data**: Domain-specific fields from the bank service
- **Traceability**: `idempotencyKey`, `timestamp`

## Data Model Mapping

### From Bank Service gRPC API

The events are derived from the Bank Service gRPC API responses:

| gRPC Response Field | Event Field | Notes |
|---------------------|-------------|-------|
| `TransferMoneyResponse.operation_id` | `operationId` | Direct mapping |
| `TransferMoneyRequest.sender_id` | `senderId` | From request context |
| `TransferMoneyRequest.recipient_id` | `recipientId` | From request context |
| `TransferMoneyRequest.amount` | `amount` | Converted to Amount schema |
| `TransferMoneyRequest.idempotency_key` | `idempotencyKey` | Direct mapping |
| `TransferMoneyResponse.timestamp` | `timestamp` | Direct mapping |

### Common Amount Schema

Both events use a shared `Amount` schema:

```json
{
  "value": "150.50",
  "currencyCode": "RUB"
}
```

- **value**: String with exactly 2 decimal places (preserves precision)
- **currencyCode**: ISO 4217 currency code (3 uppercase letters)

## Files

- **asyncapi.yaml**: Complete AsyncAPI 3.0 specification
- **schemas/transfer_event.json**: JSON Schema for transfer events

## Code Generation

You can generate Pydantic models from these JSON schemas using `datamodel-code-generator`:

```powershell
# Install the code generator
pip install datamodel-code-generator

# Generate models for transfer events
datamodel-codegen `
  --input schemas/transfer_event.json `
  --output ../../analytics-service/app/models/transfer_event.py `
  --input-file-type jsonschema
```

## Validation

To validate the AsyncAPI specification:

```powershell
# Install AsyncAPI CLI
npm install -g @asyncapi/cli

# Validate the specification
asyncapi validate asyncapi.yaml
```

## References

- [AsyncAPI 3.0 Specification](https://www.asyncapi.com/docs/reference/specification/v3.0.0)
- [AsyncAPI AMQP Bindings](https://github.com/asyncapi/bindings/tree/master/amqp)
- [JSON Schema Draft 07](https://json-schema.org/draft-07/schema)
- [RabbitMQ Topic Exchange](https://www.rabbitmq.com/tutorials/tutorial-five-python.html)
- [ISO 4217 Currency Codes](https://en.wikipedia.org/wiki/ISO_4217)
