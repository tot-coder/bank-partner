package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type AuditLogger struct {
	logger *slog.Logger
}

func NewAuditLogger(logger *slog.Logger) AuditLoggerInterface {
	return &AuditLogger{
		logger: logger,
	}
}

func (al *AuditLogger) LogTransactionStateChange(ctx context.Context, transactionID uuid.UUID, oldStatus, newStatus string) {
	al.logger.InfoContext(ctx, "transaction state change",
		slog.String("event_type", "transaction_state_change"),
		slog.String("transaction_id", transactionID.String()),
		slog.String("old_status", oldStatus),
		slog.String("new_status", newStatus),
		slog.Time("timestamp", time.Now()),
		slog.String("correlation_id", getCorrelationID(ctx)),
	)
}

func (al *AuditLogger) LogTransactionProcessingStarted(ctx context.Context, transactionID uuid.UUID, operation string) {
	al.logger.InfoContext(ctx, "transaction processing started",
		slog.String("event_type", "transaction_processing_started"),
		slog.String("transaction_id", transactionID.String()),
		slog.String("operation", operation),
		slog.Time("timestamp", time.Now()),
		slog.String("correlation_id", getCorrelationID(ctx)),
	)
}

func (al *AuditLogger) LogTransactionProcessingCompleted(ctx context.Context, transactionID uuid.UUID, operation string, durationMs int64) {
	al.logger.InfoContext(ctx, "transaction processing completed",
		slog.String("event_type", "transaction_processing_completed"),
		slog.String("transaction_id", transactionID.String()),
		slog.String("operation", operation),
		slog.Int64("duration_ms", durationMs),
		slog.Time("timestamp", time.Now()),
		slog.String("correlation_id", getCorrelationID(ctx)),
	)
}

func (al *AuditLogger) LogTransactionProcessingFailed(ctx context.Context, transactionID uuid.UUID, operation string, errorMsg string, retryCount int) {
	al.logger.WarnContext(ctx, "transaction processing failed",
		slog.String("event_type", "transaction_processing_failed"),
		slog.String("transaction_id", transactionID.String()),
		slog.String("operation", operation),
		slog.String("error", errorMsg),
		slog.Int("retry_count", retryCount),
		slog.Time("timestamp", time.Now()),
		slog.String("correlation_id", getCorrelationID(ctx)),
	)
}

func (al *AuditLogger) LogBalanceUpdate(ctx context.Context, accountID uuid.UUID, oldBalance, newBalance string, transactionID uuid.UUID) {
	al.logger.InfoContext(ctx, "balance update",
		slog.String("event_type", "balance_update"),
		slog.String("account_id", accountID.String()),
		slog.String("old_balance", oldBalance),
		slog.String("new_balance", newBalance),
		slog.String("transaction_id", transactionID.String()),
		slog.Time("timestamp", time.Now()),
		slog.String("correlation_id", getCorrelationID(ctx)),
	)
}

func (al *AuditLogger) LogQueueItemEnqueued(ctx context.Context, queueItemID, transactionID uuid.UUID, operation string, priority int) {
	al.logger.InfoContext(ctx, "queue item enqueued",
		slog.String("event_type", "queue_item_enqueued"),
		slog.String("queue_item_id", queueItemID.String()),
		slog.String("transaction_id", transactionID.String()),
		slog.String("operation", operation),
		slog.Int("priority", priority),
		slog.Time("timestamp", time.Now()),
		slog.String("correlation_id", getCorrelationID(ctx)),
	)
}

func (al *AuditLogger) LogQueueItemProcessed(ctx context.Context, queueItemID uuid.UUID, transactionID uuid.UUID, operation string, retryCount int) {
	al.logger.InfoContext(ctx, "queue item processed",
		slog.String("event_type", "queue_item_processed"),
		slog.String("queue_item_id", queueItemID.String()),
		slog.String("transaction_id", transactionID.String()),
		slog.String("operation", operation),
		slog.Int("retry_count", retryCount),
		slog.Time("timestamp", time.Now()),
		slog.String("correlation_id", getCorrelationID(ctx)),
	)
}

func (al *AuditLogger) LogCircuitBreakerStateChange(ctx context.Context, service string, oldState, newState string) {
	al.logger.WarnContext(ctx, "circuit breaker state change",
		slog.String("event_type", "circuit_breaker_state_change"),
		slog.String("service", service),
		slog.String("old_state", oldState),
		slog.String("new_state", newState),
		slog.Time("timestamp", time.Now()),
		slog.String("correlation_id", getCorrelationID(ctx)),
	)
}

func (al *AuditLogger) LogRetryAttempt(ctx context.Context, queueItemID uuid.UUID, transactionID uuid.UUID, retryCount, maxRetries int, backoffMs int64) {
	al.logger.InfoContext(ctx, "retry attempt",
		slog.String("event_type", "retry_attempt"),
		slog.String("queue_item_id", queueItemID.String()),
		slog.String("transaction_id", transactionID.String()),
		slog.Int("retry_count", retryCount),
		slog.Int("max_retries", maxRetries),
		slog.Int64("backoff_ms", backoffMs),
		slog.Time("timestamp", time.Now()),
		slog.String("correlation_id", getCorrelationID(ctx)),
	)
}

func (al *AuditLogger) LogOptimisticLockConflict(ctx context.Context, entityType string, entityID uuid.UUID, expectedVersion, actualVersion int) {
	al.logger.WarnContext(ctx, "optimistic lock conflict",
		slog.String("event_type", "optimistic_lock_conflict"),
		slog.String("entity_type", entityType),
		slog.String("entity_id", entityID.String()),
		slog.Int("expected_version", expectedVersion),
		slog.Int("actual_version", actualVersion),
		slog.Time("timestamp", time.Now()),
		slog.String("correlation_id", getCorrelationID(ctx)),
	)
}

func (al *AuditLogger) LogTransferInitiated(ctx context.Context, transferID, fromAccountID, toAccountID uuid.UUID, amount, idempotencyKey string, userID uuid.UUID) {
	al.logger.InfoContext(ctx, "transfer initiated",
		slog.String("event_type", "transfer_initiated"),
		slog.String("transfer_id", transferID.String()),
		slog.String("from_account_id", fromAccountID.String()),
		slog.String("to_account_id", toAccountID.String()),
		slog.String("amount", amount),
		slog.String("idempotency_key", idempotencyKey),
		slog.String("user_id", userID.String()),
		slog.Time("timestamp", time.Now()),
		slog.String("correlation_id", getCorrelationID(ctx)),
	)
}

func (al *AuditLogger) LogTransferCompleted(ctx context.Context, transferID uuid.UUID, durationMs int64, debitTxID, creditTxID *uuid.UUID) {
	attrs := []slog.Attr{
		slog.String("event_type", "transfer_completed"),
		slog.String("transfer_id", transferID.String()),
		slog.Int64("duration_ms", durationMs),
		slog.Time("timestamp", time.Now()),
		slog.String("correlation_id", getCorrelationID(ctx)),
	}

	if debitTxID != nil {
		attrs = append(attrs, slog.String("debit_transaction_id", debitTxID.String()))
	}
	if creditTxID != nil {
		attrs = append(attrs, slog.String("credit_transaction_id", creditTxID.String()))
	}

	al.logger.LogAttrs(ctx, slog.LevelInfo, "transfer completed", attrs...)
}

func (al *AuditLogger) LogTransferFailed(ctx context.Context, transferID uuid.UUID, errorMsg string, durationMs int64) {
	al.logger.WarnContext(ctx, "transfer failed",
		slog.String("event_type", "transfer_failed"),
		slog.String("transfer_id", transferID.String()),
		slog.String("error", errorMsg),
		slog.Int64("duration_ms", durationMs),
		slog.Time("timestamp", time.Now()),
		slog.String("correlation_id", getCorrelationID(ctx)),
	)
}

func (al *AuditLogger) LogTransferIdempotencyCheck(ctx context.Context, idempotencyKey string, existingTransferID uuid.UUID, status string) {
	al.logger.InfoContext(ctx, "transfer idempotency check",
		slog.String("event_type", "transfer_idempotency_check"),
		slog.String("idempotency_key", idempotencyKey),
		slog.String("existing_transfer_id", existingTransferID.String()),
		slog.String("status", status),
		slog.Time("timestamp", time.Now()),
		slog.String("correlation_id", getCorrelationID(ctx)),
	)
}

func getCorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		return correlationID
	}

	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}

	return ""
}
