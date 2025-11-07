# Database Migrations

This directory contains SQL migrations for the Bank Service database schema.

## Migration Files

Migrations are numbered sequentially and follow the naming convention:
```
XXX_description.up.sql   - Apply migration
XXX_description.down.sql - Rollback migration
```

### Available Migrations

1. **001_create_accounts_table** - Creates the `accounts` table
   - Stores bank account information
   - Fields: id, balance (value + currency), created_at, updated_at
   - Constraints: balance >= 0, currency code length = 3

2. **002_create_transfers_table** - Creates the `transfers` table
   - Stores money transfer operations
   - Fields: id, sender_id, recipient_id, amount (value + currency), idempotency_key, status, message, timestamps
   - Constraints: 
     - Foreign keys to accounts
     - Unique idempotency_key
     - sender_id != recipient_id
     - amount > 0
   - Indexes for efficient queries on sender, recipient, idempotency_key, created_at, status

3. **003_create_triggers** - Creates automatic triggers
   - `update_updated_at_column()` function
   - Trigger to auto-update `updated_at` on accounts table

4. **004_seed_test_data** - Seeds test accounts for development
   - 5 test accounts with various balances
   - **Note**: Only for development/testing, not for production

## Database Schema

### Accounts Table
```sql
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    balance_value NUMERIC(15, 2) NOT NULL CHECK (balance_value >= 0),
    balance_currency_code VARCHAR(3) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);
```

### Transfers Table
```sql
CREATE TABLE transfers (
    id UUID PRIMARY KEY,
    sender_id UUID NOT NULL REFERENCES accounts(id),
    recipient_id UUID NOT NULL REFERENCES accounts(id),
    amount_value NUMERIC(15, 2) NOT NULL CHECK (amount_value > 0),
    amount_currency_code VARCHAR(3) NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED')),
    message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE
);
```

## Running Migrations

### Using golang-migrate

1. **Install golang-migrate**:
   ```bash
   # Windows (PowerShell)
   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   
   # Or download binary from: https://github.com/golang-migrate/migrate/releases
   ```

2. **Set database URL**:
   ```powershell
   $env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/bank_db?sslmode=disable"
   ```

3. **Run migrations up**:
   ```bash
   migrate -path ./migrations -database $env:DATABASE_URL up
   ```

4. **Rollback migrations**:
   ```bash
   migrate -path ./migrations -database $env:DATABASE_URL down
   ```

5. **Rollback specific number of migrations**:
   ```bash
   migrate -path ./migrations -database $env:DATABASE_URL down 1
   ```

6. **Check migration version**:
   ```bash
   migrate -path ./migrations -database $env:DATABASE_URL version
   ```

### Using psql directly

You can also apply migrations manually using `psql`:

```bash
# Apply all migrations
psql -U postgres -d bank_db -f migrations/001_create_accounts_table.up.sql
psql -U postgres -d bank_db -f migrations/002_create_transfers_table.up.sql
psql -U postgres -d bank_db -f migrations/003_create_triggers.up.sql
psql -U postgres -d bank_db -f migrations/004_seed_test_data.up.sql

# Rollback (in reverse order)
psql -U postgres -d bank_db -f migrations/004_seed_test_data.down.sql
psql -U postgres -d bank_db -f migrations/003_create_triggers.down.sql
psql -U postgres -d bank_db -f migrations/002_create_transfers.down.sql
psql -U postgres -d bank_db -f migrations/001_create_accounts_table.down.sql
```

### Using Docker

If you're running PostgreSQL in Docker:

```powershell
# Run migrations
docker exec -i postgres-container psql -U postgres -d bank_db < migrations/001_create_accounts_table.up.sql
docker exec -i postgres-container psql -U postgres -d bank_db < migrations/002_create_transfers_table.up.sql
docker exec -i postgres-container psql -U postgres -d bank_db < migrations/003_create_triggers.up.sql
docker exec -i postgres-container psql -U postgres -d bank_db < migrations/004_seed_test_data.up.sql
```

## Test Data

The `004_seed_test_data.up.sql` migration creates 5 test accounts:

| Account ID | Balance | Currency |
|------------|---------|----------|
| 11111111-1111-1111-1111-111111111111 | 1000.00 | RUB |
| 22222222-2222-2222-2222-222222222222 | 500.00 | RUB |
| 33333333-3333-3333-3333-333333333333 | 10000.00 | RUB |
| 44444444-4444-4444-4444-444444444444 | 50.00 | RUB |
| 55555555-5555-5555-5555-555555555555 | 0.00 | RUB |

**Important**: This test data should NOT be used in production environments.

## Creating New Migrations

1. Create two files with the next sequential number:
   ```
   XXX_your_migration_name.up.sql
   XXX_your_migration_name.down.sql
   ```

2. Write the forward migration in `.up.sql`
3. Write the rollback migration in `.down.sql`
4. Test both up and down migrations

## Best Practices

1. **Always create both up and down migrations** - This allows for easy rollbacks
2. **Test migrations in development first** - Never test migrations in production
3. **Make migrations idempotent when possible** - Use `CREATE TABLE IF NOT EXISTS`, etc.
4. **Don't modify existing migrations** - Create new migrations instead
5. **Keep migrations small and focused** - One logical change per migration
6. **Add comments to complex migrations** - Explain what and why
7. **Use transactions** - Wrap DDL statements in transactions when possible
8. **Back up production databases** - Before running migrations in production

## Troubleshooting

### Migration version mismatch
If you get a "dirty database version" error:
```bash
migrate -path ./migrations -database $env:DATABASE_URL force VERSION_NUMBER
```

### Connection refused
Check if PostgreSQL is running and the connection string is correct.

### Permission denied
Ensure the database user has sufficient privileges to create tables and indexes.

## Integration with Go Code

The Go code uses pgx for database access. Example connection:

```go
import (
    "github.com/jackc/pgx/v5/pgxpool"
)

pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/bank_db?sslmode=disable")
```

The domain models in `internal/domain/models.go` map to these tables:
- `Account` struct → `accounts` table
- `Transfer` struct → `transfers` table
- `Amount` struct → embedded in table columns (value + currency_code)
