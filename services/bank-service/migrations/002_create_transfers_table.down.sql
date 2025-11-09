-- Rollback: Drop transfers table

DROP INDEX IF EXISTS idx_transfers_recipient_created;
DROP INDEX IF EXISTS idx_transfers_account_created;
DROP INDEX IF EXISTS idx_transfers_status;
DROP INDEX IF EXISTS idx_transfers_created_at;
DROP INDEX IF EXISTS idx_transfers_idempotency_key;
DROP INDEX IF EXISTS idx_transfers_recipient_id;
DROP INDEX IF EXISTS idx_transfers_sender_id;
DROP TABLE IF EXISTS transfers;
