-- Create additional performance indexes

-- Composite indexes for common query patterns on accounts
CREATE INDEX idx_accounts_user_status ON accounts(user_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_accounts_type_status ON accounts(account_type, status) WHERE deleted_at IS NULL;

-- Composite indexes for common query patterns on transactions
CREATE INDEX idx_transactions_account_created ON transactions(account_id, created_at DESC);
CREATE INDEX idx_transactions_account_status ON transactions(account_id, status);
CREATE INDEX idx_transactions_status_created ON transactions(status, created_at) WHERE status = 'pending';

-- Composite indexes for common query patterns on transfers
CREATE INDEX idx_transfers_from_status_created ON transfers(from_account_id, status, created_at DESC);
CREATE INDEX idx_transfers_to_status_created ON transfers(to_account_id, status, created_at DESC);
CREATE INDEX idx_transfers_status_created ON transfers(status, created_at) WHERE status = 'pending';

-- Partial indexes for soft-deleted records
CREATE INDEX idx_accounts_active ON accounts(id) WHERE deleted_at IS NULL;
CREATE INDEX idx_transactions_completed ON transactions(id) WHERE status = 'completed';

-- Add comments
COMMENT ON INDEX idx_accounts_user_status IS 'Composite index for user account queries by status';
COMMENT ON INDEX idx_transactions_account_created IS 'Composite index for account transaction history queries';
COMMENT ON INDEX idx_transfers_from_status_created IS 'Composite index for transfer sender queries';
