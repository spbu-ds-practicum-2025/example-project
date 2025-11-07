-- Create transfers table
-- This table stores all money transfer operations between accounts

CREATE TABLE IF NOT EXISTS transfers (
    id UUID PRIMARY KEY,
    sender_id UUID NOT NULL,
    recipient_id UUID NOT NULL,
    amount_value NUMERIC(15, 2) NOT NULL CHECK (amount_value > 0),
    amount_currency_code VARCHAR(3) NOT NULL CHECK (LENGTH(amount_currency_code) = 3),
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED')),
    message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- Foreign key constraints
    CONSTRAINT fk_sender FOREIGN KEY (sender_id) REFERENCES accounts(id) ON DELETE RESTRICT,
    CONSTRAINT fk_recipient FOREIGN KEY (recipient_id) REFERENCES accounts(id) ON DELETE RESTRICT,
    
    -- Business rule: sender and recipient must be different
    CONSTRAINT chk_different_accounts CHECK (sender_id != recipient_id)
);

-- Create indexes for efficient queries
CREATE INDEX idx_transfers_sender_id ON transfers(sender_id);
CREATE INDEX idx_transfers_recipient_id ON transfers(recipient_id);
CREATE INDEX idx_transfers_idempotency_key ON transfers(idempotency_key);
CREATE INDEX idx_transfers_created_at ON transfers(created_at DESC);
CREATE INDEX idx_transfers_status ON transfers(status);

-- Composite index for account transfer history queries
CREATE INDEX idx_transfers_account_created ON transfers(sender_id, created_at DESC);
CREATE INDEX idx_transfers_recipient_created ON transfers(recipient_id, created_at DESC);

-- Add comments to table and columns
COMMENT ON TABLE transfers IS 'Money transfer operations between accounts';
COMMENT ON COLUMN transfers.id IS 'Unique identifier of the transfer operation (UUID)';
COMMENT ON COLUMN transfers.sender_id IS 'Account ID of the sender (account being debited)';
COMMENT ON COLUMN transfers.recipient_id IS 'Account ID of the recipient (account being credited)';
COMMENT ON COLUMN transfers.amount_value IS 'Amount transferred (decimal with 2 decimal places)';
COMMENT ON COLUMN transfers.amount_currency_code IS 'ISO 4217 currency code (e.g., RUB, USD, EUR)';
COMMENT ON COLUMN transfers.idempotency_key IS 'Unique key to ensure idempotent operations - prevents duplicate transfers';
COMMENT ON COLUMN transfers.status IS 'Current status of the transfer: PENDING, SUCCESS, or FAILED';
COMMENT ON COLUMN transfers.message IS 'Human-readable message about the transfer result';
COMMENT ON COLUMN transfers.created_at IS 'Timestamp when the transfer was initiated';
COMMENT ON COLUMN transfers.completed_at IS 'Timestamp when the transfer was completed (NULL if still pending)';
