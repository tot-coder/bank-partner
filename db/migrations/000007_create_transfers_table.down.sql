-- Drop transfers table and related objects
DROP TRIGGER IF EXISTS update_transfers_updated_at ON transfers;
DROP INDEX IF EXISTS idx_transfers_credit_transaction_id;
DROP INDEX IF EXISTS idx_transfers_debit_transaction_id;
DROP INDEX IF EXISTS idx_transfers_idempotency_key;
DROP INDEX IF EXISTS idx_transfers_created_at;
DROP INDEX IF EXISTS idx_transfers_status;
DROP INDEX IF EXISTS idx_transfers_to_account_id;
DROP INDEX IF EXISTS idx_transfers_from_account_id;
DROP TABLE IF EXISTS transfers CASCADE;
