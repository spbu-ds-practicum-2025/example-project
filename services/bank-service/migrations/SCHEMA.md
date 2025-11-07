# Database Schema Diagram

## Entity Relationship Diagram

```
┌─────────────────────────────────────┐
│            accounts                 │
├─────────────────────────────────────┤
│ PK │ id                   UUID      │
│    │ balance_value        NUMERIC   │
│    │ balance_currency_code VARCHAR  │
│    │ created_at           TIMESTAMP │
│    │ updated_at           TIMESTAMP │
└─────────────────────────────────────┘
          ▲                ▲
          │                │
          │                │
  sender_id                │ recipient_id
    (FK)                   │   (FK)
          │                │
          │                │
┌─────────┴────────────────┴───────────┐
│            transfers                 │
├──────────────────────────────────────┤
│ PK │ id                   UUID       │
│ FK │ sender_id            UUID       │
│ FK │ recipient_id         UUID       │
│    │ amount_value         NUMERIC    │
│    │ amount_currency_code VARCHAR    │
│ UQ │ idempotency_key      VARCHAR    │
│    │ status               VARCHAR    │
│    │ message              TEXT       │
│    │ created_at           TIMESTAMP  │
│    │ nullable completed_at TIMESTAMP │
└──────────────────────────────────────┘
```

## Table Details

### accounts

**Purpose**: Stores bank account information and balances

| Column                | Type                    | Constraints           | Description                          |
|-----------------------|-------------------------|-----------------------|--------------------------------------|
| id                    | UUID                    | PRIMARY KEY           | Unique account identifier            |
| balance_value         | NUMERIC(15, 2)          | NOT NULL, >= 0        | Current balance (2 decimal places)   |
| balance_currency_code | VARCHAR(3)              | NOT NULL, LENGTH = 3  | ISO 4217 currency code (RUB, USD...) |
| created_at            | TIMESTAMP WITH TIME ZONE| NOT NULL, DEFAULT NOW | Account creation timestamp           |
| updated_at            | TIMESTAMP WITH TIME ZONE| NOT NULL, DEFAULT NOW | Last update timestamp                |

**Indexes**:
- `idx_accounts_updated_at` on (updated_at)

**Triggers**:
- `trigger_accounts_updated_at` - Automatically updates `updated_at` on modifications

---

### transfers

**Purpose**: Records all money transfer operations between accounts

| Column                | Type                    | Constraints                    | Description                              |
|-----------------------|-------------------------|--------------------------------|------------------------------------------|
| id                    | UUID                    | PRIMARY KEY                    | Unique transfer identifier               |
| sender_id             | UUID                    | FK → accounts(id), NOT NULL    | Account being debited                    |
| recipient_id          | UUID                    | FK → accounts(id), NOT NULL    | Account being credited                   |
| amount_value          | NUMERIC(15, 2)          | NOT NULL, > 0                  | Transfer amount (2 decimal places)       |
| amount_currency_code  | VARCHAR(3)              | NOT NULL, LENGTH = 3           | ISO 4217 currency code                   |
| idempotency_key       | VARCHAR(255)            | UNIQUE, NOT NULL               | Prevents duplicate transfers             |
| status                | VARCHAR(20)             | NOT NULL, IN (...)             | PENDING, SUCCESS, or FAILED              |
| message               | TEXT                    | NULLABLE                       | Human-readable status message            |
| created_at            | TIMESTAMP WITH TIME ZONE| NOT NULL, DEFAULT NOW          | Transfer initiation timestamp            |
| completed_at          | TIMESTAMP WITH TIME ZONE| NULLABLE                       | Transfer completion timestamp            |

**Check Constraints**:
- `chk_different_accounts`: Ensures sender_id ≠ recipient_id
- `amount_value > 0`: Transfer amount must be positive
- `status IN ('PENDING', 'SUCCESS', 'FAILED')`: Valid status values only

**Foreign Keys**:
- `fk_sender`: sender_id → accounts(id) ON DELETE RESTRICT
- `fk_recipient`: recipient_id → accounts(id) ON DELETE RESTRICT

**Indexes**:
- `idx_transfers_sender_id` on (sender_id)
- `idx_transfers_recipient_id` on (recipient_id)
- `idx_transfers_idempotency_key` on (idempotency_key)
- `idx_transfers_created_at` on (created_at DESC)
- `idx_transfers_status` on (status)
- `idx_transfers_account_created` on (sender_id, created_at DESC)
- `idx_transfers_recipient_created` on (recipient_id, created_at DESC)

## Relationships

1. **transfers.sender_id → accounts.id**
   - Type: Many-to-One
   - Delete Rule: RESTRICT (cannot delete account with transfers)
   - Description: Each transfer has one sender account

2. **transfers.recipient_id → accounts.id**
   - Type: Many-to-One
   - Delete Rule: RESTRICT (cannot delete account with transfers)
   - Description: Each transfer has one recipient account

## Business Rules Enforced by Schema

1. **Non-negative account balances**: `balance_value >= 0`
2. **Positive transfer amounts**: `amount_value > 0`
3. **Different sender and recipient**: `sender_id != recipient_id`
4. **Unique idempotency keys**: Prevents duplicate transfer execution
5. **Valid currency codes**: ISO 4217 format (3 uppercase letters)
6. **Valid transfer statuses**: Only PENDING, SUCCESS, or FAILED allowed
7. **Referential integrity**: Transfers must reference valid accounts
8. **Cascade prevention**: Accounts with transfers cannot be deleted

## Index Strategy

### Query Patterns Optimized

1. **Find account by ID**: Primary key lookup
2. **Find transfers by idempotency key**: Unique index lookup (idempotency check)
3. **List account's sent transfers**: `idx_transfers_account_created` composite index
4. **List account's received transfers**: `idx_transfers_recipient_created` composite index
5. **Filter transfers by status**: `idx_transfers_status`
6. **Recent transfers**: `idx_transfers_created_at` (DESC order)
7. **Account updates by time**: `idx_accounts_updated_at`

### Index Selection Examples

```sql
-- Idempotency check (uses idx_transfers_idempotency_key)
SELECT * FROM transfers WHERE idempotency_key = '...';

-- Account transfer history (uses idx_transfers_account_created)
SELECT * FROM transfers 
WHERE sender_id = '...' 
ORDER BY created_at DESC 
LIMIT 100;

-- Failed transfers (uses idx_transfers_status)
SELECT * FROM transfers WHERE status = 'FAILED';

-- Recent transfers (uses idx_transfers_created_at)
SELECT * FROM transfers ORDER BY created_at DESC LIMIT 50;
```

## Data Types Rationale

- **UUID**: Globally unique, prevents ID collisions across distributed systems
- **NUMERIC(15, 2)**: Exact decimal arithmetic for money (no float rounding errors)
- **VARCHAR(3)**: Fixed-size currency codes (ISO 4217 standard)
- **TIMESTAMP WITH TIME ZONE**: Timezone-aware timestamps for distributed systems
- **TEXT**: Unlimited length for error messages and descriptions

## Migration Order

Migrations must be applied in order:

1. **001_create_accounts_table** - Creates accounts (no dependencies)
2. **002_create_transfers_table** - Creates transfers (depends on accounts)
3. **003_create_triggers** - Creates auto-update trigger (depends on accounts)
4. **004_seed_test_data** - Inserts test accounts (development only)

## Sample Queries

### Check account balance
```sql
SELECT 
    id,
    balance_value || ' ' || balance_currency_code AS balance,
    updated_at
FROM accounts
WHERE id = '11111111-1111-1111-1111-111111111111';
```

### View account transfer history
```sql
SELECT 
    t.id,
    t.sender_id,
    t.recipient_id,
    t.amount_value || ' ' || t.amount_currency_code AS amount,
    t.status,
    t.message,
    t.created_at,
    t.completed_at
FROM transfers t
WHERE t.sender_id = '11111111-1111-1111-1111-111111111111'
   OR t.recipient_id = '11111111-1111-1111-1111-111111111111'
ORDER BY t.created_at DESC;
```

### Find pending transfers
```sql
SELECT 
    id,
    sender_id,
    recipient_id,
    amount_value,
    created_at,
    NOW() - created_at AS pending_duration
FROM transfers
WHERE status = 'PENDING'
ORDER BY created_at;
```

### Check for duplicate idempotency key
```sql
SELECT EXISTS(
    SELECT 1 FROM transfers 
    WHERE idempotency_key = 'your-key-here'
) AS transfer_exists;
```
