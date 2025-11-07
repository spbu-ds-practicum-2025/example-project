# Bank Service - Database Setup Guide

Quick reference guide for setting up and managing the Bank Service database.

## Prerequisites

- PostgreSQL 15+ installed or running in Docker
- `golang-migrate` tool (optional but recommended)
- `make` command (optional)

## Quick Start

### Option 1: Using PowerShell Script (Windows)

```powershell
# Apply all migrations
.\run-migrations.ps1 -Action up

# Check migration status
.\run-migrations.ps1 -Action version

# Rollback last migration
.\run-migrations.ps1 -Action down -Steps 1
```

### Option 2: Using Shell Script (Linux/Mac)

```bash
# Make script executable
chmod +x run-migrations.sh

# Apply all migrations
./run-migrations.sh up

# Check migration status
./run-migrations.sh version

# Rollback last migration
./run-migrations.sh down 1
```

### Option 3: Using Makefile

```bash
# Apply migrations
make migrate-up

# Check version
make migrate-version

# Rollback last migration
make migrate-down

# Reset database (rollback all and reapply)
make db-reset
```

## Installation Steps

### 1. Install PostgreSQL

**Using Docker (Recommended for development):**
```powershell
# Start PostgreSQL
docker run -d `
  --name bank-postgres `
  -e POSTGRES_PASSWORD=postgres `
  -e POSTGRES_DB=bank_db `
  -p 5432:5432 `
  postgres:15

# Or using Make
make db-start
```

**Using native installation:**
- Download from https://www.postgresql.org/download/
- Install and create database: `CREATE DATABASE bank_db;`

### 2. Install golang-migrate (Optional)

**Using Go:**
```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

**Using binary download:**
- Download from: https://github.com/golang-migrate/migrate/releases
- Add to PATH

### 3. Configure Database Connection

Set the `DATABASE_URL` environment variable:

**PowerShell:**
```powershell
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/bank_db?sslmode=disable"
```

**Bash:**
```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/bank_db?sslmode=disable"
```

**Or pass directly to scripts:**
```powershell
.\run-migrations.ps1 -Action up -DatabaseUrl "postgres://user:pass@host:5432/db?sslmode=disable"
```

### 4. Run Migrations

```bash
# Apply all migrations
make migrate-up

# Or using the script
.\run-migrations.ps1 -Action up
```

## Database Schema

After running migrations, you'll have:

### Tables

**accounts**
- `id` (UUID, Primary Key)
- `balance_value` (NUMERIC(15,2))
- `balance_currency_code` (VARCHAR(3))
- `created_at` (TIMESTAMP WITH TIME ZONE)
- `updated_at` (TIMESTAMP WITH TIME ZONE)

**transfers**
- `id` (UUID, Primary Key)
- `sender_id` (UUID, Foreign Key → accounts.id)
- `recipient_id` (UUID, Foreign Key → accounts.id)
- `amount_value` (NUMERIC(15,2))
- `amount_currency_code` (VARCHAR(3))
- `idempotency_key` (VARCHAR(255), Unique)
- `status` (VARCHAR(20): PENDING, SUCCESS, FAILED)
- `message` (TEXT)
- `created_at` (TIMESTAMP WITH TIME ZONE)
- `completed_at` (TIMESTAMP WITH TIME ZONE, Nullable)

### Constraints

- Accounts: `balance_value >= 0`
- Transfers: `amount_value > 0`
- Transfers: `sender_id != recipient_id`
- Transfers: Unique `idempotency_key`

### Indexes

- `idx_accounts_updated_at` on accounts(updated_at)
- `idx_transfers_sender_id` on transfers(sender_id)
- `idx_transfers_recipient_id` on transfers(recipient_id)
- `idx_transfers_idempotency_key` on transfers(idempotency_key)
- `idx_transfers_created_at` on transfers(created_at DESC)
- `idx_transfers_status` on transfers(status)
- Composite indexes for account history queries

### Triggers

- `trigger_accounts_updated_at`: Auto-updates `updated_at` on account modifications

## Test Data

Migration `004_seed_test_data` creates 5 test accounts:

```
ID: 11111111-1111-1111-1111-111111111111, Balance: 1000.00 RUB
ID: 22222222-2222-2222-2222-222222222222, Balance: 500.00 RUB
ID: 33333333-3333-3333-3333-333333333333, Balance: 10000.00 RUB
ID: 44444444-4444-4444-4444-444444444444, Balance: 50.00 RUB
ID: 55555555-5555-5555-5555-555555555555, Balance: 0.00 RUB
```

## Common Tasks

### Check Database Connection

```bash
# Using psql
psql -U postgres -d bank_db -c "SELECT version();"

# Using Docker
docker exec -it bank-postgres psql -U postgres -d bank_db -c "SELECT version();"
```

### View Current Schema

```sql
-- List all tables
\dt

-- Describe accounts table
\d accounts

-- Describe transfers table
\d transfers

-- List all indexes
\di
```

### Query Test Data

```sql
-- View all accounts
SELECT * FROM accounts;

-- View all transfers
SELECT * FROM transfers;

-- Check account balance
SELECT id, balance_value, balance_currency_code 
FROM accounts 
WHERE id = '11111111-1111-1111-1111-111111111111';
```

### Create a Test Transfer

```sql
-- Insert a successful transfer
INSERT INTO transfers (
    id, sender_id, recipient_id, 
    amount_value, amount_currency_code,
    idempotency_key, status, message, 
    created_at, completed_at
)
VALUES (
    gen_random_uuid(),
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222222',
    100.00, 'RUB',
    gen_random_uuid()::text,
    'SUCCESS', 'Transfer completed successfully',
    NOW(), NOW()
);
```

## Troubleshooting

### "relation does not exist" error
- Make sure migrations have been applied: `make migrate-version`
- Re-run migrations: `make migrate-up`

### "duplicate key value violates unique constraint"
- Check if test data already exists
- Idempotency key must be unique for each transfer

### Connection refused
- Check if PostgreSQL is running: `docker ps` or `pg_isready`
- Verify connection string in DATABASE_URL
- Check firewall settings

### Dirty database version
- Force set version: `migrate -path ./migrations -database "$DATABASE_URL" force <version>`
- Or reset completely: `make db-reset`

## Production Considerations

1. **Remove test data migration** - Don't run `004_seed_test_data` in production
2. **Use connection pooling** - Configure pgx pool settings appropriately
3. **Enable SSL** - Change `sslmode=disable` to `sslmode=require`
4. **Set up backups** - Regular automated backups
5. **Monitor performance** - Use pg_stat_statements and slow query logs
6. **Optimize indexes** - Based on actual query patterns
7. **Use prepared statements** - For better performance and security

## Integration with Go Code

The migrations create tables that map to Go structs in `internal/domain/models.go`:

```go
// Account struct → accounts table
type Account struct {
    ID        uuid.UUID
    Balance   Amount
    CreatedAt time.Time
    UpdatedAt time.Time
}

// Transfer struct → transfers table
type Transfer struct {
    ID             uuid.UUID
    SenderID       uuid.UUID
    RecipientID    uuid.UUID
    Amount         Amount
    IdempotencyKey string
    Status         TransferStatus
    Message        string
    CreatedAt      time.Time
    CompletedAt    *time.Time
}
```

Repository implementations in `internal/db/` will use these tables to persist domain models.

## Next Steps

1. Implement repository layer in `internal/db/`
2. Set up connection pooling with pgx
3. Implement database transaction manager
4. Add database integration tests
5. Set up database migrations in CI/CD pipeline
