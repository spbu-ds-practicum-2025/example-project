-- ClickHouse operations table (example)
CREATE TABLE IF NOT EXISTS operations (
    id String,
    account_id String,
    type String,
    amount Float64,
    timestamp DateTime
) ENGINE = MergeTree() ORDER BY (account_id, timestamp);
