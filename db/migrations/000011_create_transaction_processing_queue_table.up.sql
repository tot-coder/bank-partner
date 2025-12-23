-- Create transaction_processing_queue table for async transaction processing
CREATE TABLE transaction_processing_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL,
    operation VARCHAR(50) NOT NULL CHECK (operation IN ('process', 'reverse', 'validate')),
    priority INTEGER NOT NULL DEFAULT 100 CHECK (priority >= 0 AND priority <= 1000),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    retry_count INTEGER NOT NULL DEFAULT 0 CHECK (retry_count >= 0),
    max_retries INTEGER NOT NULL DEFAULT 3 CHECK (max_retries >= 0),
    scheduled_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP NULL,
    error_message TEXT NULL,
    metadata JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_transaction
        FOREIGN KEY (transaction_id)
        REFERENCES transactions(id)
        ON DELETE CASCADE
);

-- Index on status and scheduled_at for efficient queue polling
-- Higher priority items should be processed first, then oldest first
CREATE INDEX idx_processing_queue_status
    ON transaction_processing_queue (status, priority DESC, scheduled_at ASC)
    WHERE status IN ('pending', 'processing');

-- Index on transaction_id for quick lookups
CREATE INDEX idx_processing_queue_transaction
    ON transaction_processing_queue (transaction_id);

-- Index for monitoring failed items
CREATE INDEX idx_processing_queue_failed
    ON transaction_processing_queue (status, updated_at DESC)
    WHERE status = 'failed';

-- Index for retry processing
CREATE INDEX idx_processing_queue_retry
    ON transaction_processing_queue (status, retry_count, scheduled_at)
    WHERE status = 'pending' AND retry_count > 0;

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_transaction_processing_queue_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update updated_at
CREATE TRIGGER trigger_update_transaction_processing_queue_timestamp
    BEFORE UPDATE ON transaction_processing_queue
    FOR EACH ROW
    EXECUTE FUNCTION update_transaction_processing_queue_updated_at();

-- Comments for documentation
COMMENT ON TABLE transaction_processing_queue IS 'Queue for async transaction processing with retry mechanism';
COMMENT ON COLUMN transaction_processing_queue.priority IS 'Higher values = higher priority (50=low, 100=normal, 200=high)';
COMMENT ON COLUMN transaction_processing_queue.retry_count IS 'Number of retry attempts made';
COMMENT ON COLUMN transaction_processing_queue.max_retries IS 'Maximum number of retries before marking as failed';
COMMENT ON COLUMN transaction_processing_queue.metadata IS 'Additional context for processing (JSON format)';
