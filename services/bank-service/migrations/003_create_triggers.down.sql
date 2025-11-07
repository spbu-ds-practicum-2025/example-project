-- Rollback: Drop triggers and functions

DROP TRIGGER IF EXISTS trigger_accounts_updated_at ON accounts;
DROP FUNCTION IF EXISTS update_updated_at_column();
