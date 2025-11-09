-- Rollback: Drop accounts table

DROP INDEX IF EXISTS idx_accounts_updated_at;
DROP TABLE IF EXISTS accounts;
