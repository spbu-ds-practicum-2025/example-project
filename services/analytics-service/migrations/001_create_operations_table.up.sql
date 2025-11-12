-- Create operations table for storing account operations history
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
PRIMARY KEY (account_id, timestamp);
