-- Drop trigger
DROP TRIGGER IF EXISTS trigger_update_transaction_processing_queue_timestamp ON transaction_processing_queue;

-- Drop function
DROP FUNCTION IF EXISTS update_transaction_processing_queue_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_processing_queue_retry;
DROP INDEX IF EXISTS idx_processing_queue_failed;
DROP INDEX IF EXISTS idx_processing_queue_transaction;
DROP INDEX IF EXISTS idx_processing_queue_status;

-- Drop table
DROP TABLE IF EXISTS transaction_processing_queue;
