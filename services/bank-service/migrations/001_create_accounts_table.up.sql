-- Create accounts table
-- This table stores all bank accounts in the system

CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY,
    balance_value NUMERIC(15, 2) NOT NULL CHECK (balance_value >= 0),
    balance_currency_code VARCHAR(3) NOT NULL CHECK (LENGTH(balance_currency_code) = 3),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on updated_at for efficient queries
CREATE INDEX idx_accounts_updated_at ON accounts(updated_at);

-- Add comment to table
COMMENT ON TABLE accounts IS 'Bank accounts with balance information';
COMMENT ON COLUMN accounts.id IS 'Unique identifier of the account (UUID)';
COMMENT ON COLUMN accounts.balance_value IS 'Current account balance (decimal with 2 decimal places)';
COMMENT ON COLUMN accounts.balance_currency_code IS 'ISO 4217 currency code (e.g., RUB, USD, EUR)';
COMMENT ON COLUMN accounts.created_at IS 'Timestamp when the account was created';
COMMENT ON COLUMN accounts.updated_at IS 'Timestamp of the last account update';
