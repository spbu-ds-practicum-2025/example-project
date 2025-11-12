# ClickHouse Migrations

This directory contains SQL migration files for the ClickHouse database schema.

## Migration Files

Migrations follow the naming convention:
- `XXX_description.up.sql` - Apply migration
- `XXX_description.down.sql` - Rollback migration

Where `XXX` is a sequential number (001, 002, etc.)

## Available Migrations

### 001_create_operations_table
Creates the main `operations` table for storing account operation history.

**Schema:**
- `id` - Operation UUID
- `account_id` - Account UUID (indexed)
- `operation_type` - Type of operation (TOPUP or TRANSFER)
- `timestamp` - Operation timestamp with millisecond precision (indexed)
- `amount_value` - Monetary amount (Decimal with 2 decimal places)
- `amount_currency` - Currency code (e.g., RUB)
- `sender_id` - Sender account ID (for transfers)
- `recipient_id` - Recipient account ID (for transfers)
- `created_at` - Record creation timestamp

**Engine:** MergeTree with primary key `(account_id, timestamp)`

## Running Migrations

### Manual Migration

Connect to ClickHouse and run the migration files:

```bash
# Apply migration
clickhouse-client --host localhost --port 9000 --database analytics < migrations/001_create_operations_table.up.sql

# Rollback migration
clickhouse-client --host localhost --port 9000 --database analytics < migrations/001_create_operations_table.down.sql
```

### Using Docker

```bash
# Copy migration to container and execute
docker cp migrations/001_create_operations_table.up.sql clickhouse-container:/tmp/
docker exec clickhouse-container clickhouse-client --database analytics --query "$(cat /tmp/001_create_operations_table.up.sql)"
```

## Notes

- Always test migrations in a development environment first
- Keep migrations idempotent when possible (use `IF NOT EXISTS`, `IF EXISTS`)
- Never modify existing migration files after they've been applied to production
- Create new migration files for schema changes
