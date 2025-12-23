package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"array-assessment/internal/dto"
	"array-assessment/internal/models"
	"array-assessment/internal/repositories"

	"github.com/google/uuid"
)

var (
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
	ErrDuplicateReference = errors.New("duplicate transaction reference")
)

type TransactionProcessingService struct {
	transactionRepo repositories.TransactionRepositoryInterface
	queueRepo       repositories.ProcessingQueueRepositoryInterface
	accountRepo     repositories.AccountRepositoryInterface
	auditLogger     AuditLoggerInterface
	metrics         MetricsRecorderInterface
	circuitBreaker  CircuitBreakerInterface
	maxWorkers      int
	workerSemaphore chan struct{}
	logger          *slog.Logger
}

func NewTransactionProcessingService(
	transactionRepo repositories.TransactionRepositoryInterface,
	queueRepo repositories.ProcessingQueueRepositoryInterface,
	accountRepo repositories.AccountRepositoryInterface,
	auditLogger AuditLoggerInterface,
	metrics MetricsRecorderInterface,
	circuitBreaker CircuitBreakerInterface,
	maxWorkers int,
) TransactionProcessingServiceInterface {
	return &TransactionProcessingService{
		transactionRepo: transactionRepo,
		queueRepo:       queueRepo,
		accountRepo:     accountRepo,
		auditLogger:     auditLogger,
		metrics:         metrics,
		circuitBreaker:  circuitBreaker,
		maxWorkers:      maxWorkers,
		workerSemaphore: make(chan struct{}, maxWorkers),
		logger:          slog.Default(),
	}
}

func (s *TransactionProcessingService) EnqueueTransaction(transactionID uuid.UUID, operation string, priority int) error {
	_, err := s.transactionRepo.GetByID(transactionID)
	if err != nil {
		return fmt.Errorf("transaction not found: %w", err)
	}

	if err := s.queueRepo.Enqueue(transactionID, operation, priority); err != nil {
		return fmt.Errorf("failed to enqueue transaction: %w", err)
	}

	s.metrics.IncrementCounter("queue.enqueued", map[string]string{
		"operation": operation,
	})

	return nil
}

func (s *TransactionProcessingService) StartProcessing(ctx context.Context) {
	s.logger.Info("starting transaction processing service",
		slog.Int("max_workers", s.maxWorkers),
	)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("processing service shutting down, waiting for workers to complete")
			wg.Wait()
			s.logger.Info("processing service stopped")
			return

		case <-ticker.C:
			items, err := s.queueRepo.FetchPending(s.maxWorkers * 2)
			if err != nil {
				s.logger.Error("failed to fetch pending items",
					slog.String("error", err.Error()),
				)
				continue
			}

			for _, item := range items {
				wg.Add(1)
				go s.processQueueItemAsync(ctx, item, &wg)
			}
		}
	}
}

func (s *TransactionProcessingService) processQueueItemAsync(ctx context.Context, queueItem *models.ProcessingQueueItem, wg *sync.WaitGroup) {
	defer wg.Done()

	s.workerSemaphore <- struct{}{}
	defer func() { <-s.workerSemaphore }()

	if err := s.ProcessQueueItem(ctx, queueItem); err != nil {
		s.logger.Error("failed to process queue item",
			slog.String("queue_item_id", queueItem.ID.String()),
			slog.String("transaction_id", queueItem.TransactionID.String()),
			slog.String("error", err.Error()),
		)
	}
}

func (s *TransactionProcessingService) ProcessQueueItem(ctx context.Context, queueItem *models.ProcessingQueueItem) error {
	startTime := time.Now()

	if err := s.validateProcessingPreconditions(ctx, queueItem); err != nil {
		return err
	}

	s.auditLogger.LogTransactionProcessingStarted(ctx, queueItem.TransactionID, queueItem.Operation)

	transaction, err := s.fetchAndValidateTransaction(ctx, queueItem)
	if err != nil {
		return err
	}

	if err := s.performOperation(ctx, queueItem, transaction); err != nil {
		s.circuitBreaker.RecordFailure()
		return s.handleProcessingError(ctx, queueItem, err)
	}

	return s.completeProcessing(ctx, queueItem, startTime)
}

func (s *TransactionProcessingService) validateProcessingPreconditions(ctx context.Context, queueItem *models.ProcessingQueueItem) error {
	if s.circuitBreaker.IsOpen() {
		s.metrics.IncrementCounter("circuit_breaker.open", map[string]string{
			"service": "database",
		})
		return ErrCircuitBreakerOpen
	}

	if queueItem.RetryCount >= queueItem.MaxRetries {
		return s.handleMaxRetriesExceeded(ctx, queueItem)
	}

	return nil
}

func (s *TransactionProcessingService) fetchAndValidateTransaction(ctx context.Context, queueItem *models.ProcessingQueueItem) (*models.Transaction, error) {
	transaction, err := s.transactionRepo.GetByID(queueItem.TransactionID)
	if err != nil {
		s.circuitBreaker.RecordFailure()
		return nil, s.handleProcessingError(ctx, queueItem, err)
	}

	if transaction.Reference != "" {
		if isDuplicate, err := s.checkDuplicateReference(transaction); isDuplicate {
			s.metrics.IncrementCounter("transaction.duplicate.rejected", map[string]string{
				"reference": transaction.Reference,
			})
			return nil, s.handleDuplicateReference(ctx, queueItem, transaction)
		} else if err != nil {
			s.circuitBreaker.RecordFailure()
			return nil, s.handleProcessingError(ctx, queueItem, err)
		}
	}

	return transaction, nil
}

func (s *TransactionProcessingService) checkDuplicateReference(transaction *models.Transaction) (bool, error) {
	existingTxn, err := s.transactionRepo.GetByReference(transaction.Reference)
	if err != nil {
		return false, err
	}
	return existingTxn != nil && existingTxn.ID != transaction.ID, nil
}

func (s *TransactionProcessingService) performOperation(ctx context.Context, queueItem *models.ProcessingQueueItem, transaction *models.Transaction) error {
	switch queueItem.Operation {
	case models.QueueOperationProcess:
		return s.processTransaction(ctx, transaction)
	case models.QueueOperationReverse:
		return s.reverseTransaction(ctx, transaction)
	default:
		return fmt.Errorf("unknown operation: %s", queueItem.Operation)
	}
}

func (s *TransactionProcessingService) completeProcessing(ctx context.Context, queueItem *models.ProcessingQueueItem, startTime time.Time) error {
	if err := s.queueRepo.MarkCompleted(queueItem.ID); err != nil {
		return err
	}

	s.circuitBreaker.RecordSuccess()
	s.auditLogger.LogQueueItemProcessed(ctx, queueItem.ID, queueItem.TransactionID, queueItem.Operation, queueItem.RetryCount)

	duration := time.Since(startTime)
	s.metrics.RecordProcessingTime("transaction.processing", duration)
	s.metrics.IncrementCounter("transaction.processed.success", map[string]string{
		"operation": queueItem.Operation,
	})

	s.auditLogger.LogTransactionProcessingCompleted(ctx, queueItem.TransactionID, queueItem.Operation, duration.Milliseconds())

	return nil
}

func (s *TransactionProcessingService) processTransaction(ctx context.Context, transaction *models.Transaction) error {
	if !transaction.IsPending() {
		return fmt.Errorf("transaction is not in pending status: %s", transaction.Status)
	}

	oldStatus := transaction.Status
	expectedVersion := transaction.Version

	transaction.Complete()

	if err := s.updateAccountBalance(ctx, transaction); err != nil {
		return fmt.Errorf("failed to update account balance: %w", err)
	}

	if err := s.transactionRepo.UpdateWithOptimisticLock(transaction, expectedVersion); err != nil {
		if errors.Is(err, models.ErrOptimisticLockConflict) {
			s.auditLogger.LogOptimisticLockConflict(ctx, "transaction", transaction.ID, expectedVersion, transaction.Version)
		}
		return err
	}

	s.auditLogger.LogTransactionStateChange(ctx, transaction.ID, oldStatus, transaction.Status)

	return nil
}

func (s *TransactionProcessingService) reverseTransaction(ctx context.Context, transaction *models.Transaction) error {
	if !transaction.IsCompleted() {
		return fmt.Errorf("transaction is not completed: %s", transaction.Status)
	}

	oldStatus := transaction.Status
	expectedVersion := transaction.Version

	transaction.Reverse()

	if err := s.reverseAccountBalance(ctx, transaction); err != nil {
		return fmt.Errorf("failed to reverse account balance: %w", err)
	}

	if err := s.transactionRepo.UpdateWithOptimisticLock(transaction, expectedVersion); err != nil {
		return err
	}

	s.auditLogger.LogTransactionStateChange(ctx, transaction.ID, oldStatus, transaction.Status)

	return nil
}

func (s *TransactionProcessingService) updateAccountBalance(ctx context.Context, transaction *models.Transaction) error {
	account, err := s.accountRepo.GetByID(transaction.AccountID)
	if err != nil {
		return err
	}

	oldBalance := account.Balance.String()
	newBalance := transaction.BalanceAfter

	if err := s.accountRepo.UpdateBalance(account.ID, newBalance, transaction.TransactionType); err != nil {
		return err
	}

	s.auditLogger.LogBalanceUpdate(ctx, account.ID, oldBalance, newBalance.String(), transaction.ID)

	return nil
}

func (s *TransactionProcessingService) reverseAccountBalance(ctx context.Context, transaction *models.Transaction) error {
	account, err := s.accountRepo.GetByID(transaction.AccountID)
	if err != nil {
		return err
	}

	oldBalance := account.Balance.String()
	newBalance := transaction.BalanceBefore

	if err := s.accountRepo.UpdateBalance(account.ID, newBalance, transaction.TransactionType); err != nil {
		return err
	}

	s.auditLogger.LogBalanceUpdate(ctx, account.ID, oldBalance, newBalance.String(), transaction.ID)

	return nil
}

func (s *TransactionProcessingService) handleProcessingError(ctx context.Context, queueItem *models.ProcessingQueueItem, err error) error {
	if queueItem.RetryCount < queueItem.MaxRetries {
		backoffMs := int64(math.Pow(2, float64(queueItem.RetryCount)) * 1000)

		s.auditLogger.LogRetryAttempt(ctx, queueItem.ID, queueItem.TransactionID, queueItem.RetryCount+1, queueItem.MaxRetries, backoffMs)

		if retryErr := s.queueRepo.IncrementRetry(queueItem.ID); retryErr != nil {
			return fmt.Errorf("failed to increment retry: %w", retryErr)
		}

		s.metrics.IncrementCounter("transaction.processing.retry", map[string]string{
			"operation": queueItem.Operation,
		})

		return err
	}

	return s.handleMaxRetriesExceeded(ctx, queueItem)
}

func (s *TransactionProcessingService) handleMaxRetriesExceeded(ctx context.Context, queueItem *models.ProcessingQueueItem) error {
	transaction, err := s.transactionRepo.GetByID(queueItem.TransactionID)
	if err == nil {
		oldStatus := transaction.Status
		transaction.Fail()
		expectedVersion := transaction.Version - 1
		_ = s.transactionRepo.UpdateWithOptimisticLock(transaction, expectedVersion)
		s.auditLogger.LogTransactionStateChange(ctx, transaction.ID, oldStatus, models.TransactionStatusFailed)
	}

	if err := s.queueRepo.MarkFailed(queueItem.ID, "max retries exceeded"); err != nil {
		return err
	}

	s.metrics.IncrementCounter("transaction.processed.failed", map[string]string{
		"operation": queueItem.Operation,
		"reason":    "max_retries",
	})

	s.auditLogger.LogTransactionProcessingFailed(ctx, queueItem.TransactionID, queueItem.Operation, ErrMaxRetriesExceeded.Error(), queueItem.RetryCount)

	return ErrMaxRetriesExceeded
}

func (s *TransactionProcessingService) handleDuplicateReference(ctx context.Context, queueItem *models.ProcessingQueueItem, transaction *models.Transaction) error {
	transaction.Fail()
	expectedVersion := transaction.Version
	_ = s.transactionRepo.UpdateWithOptimisticLock(transaction, expectedVersion)

	if err := s.queueRepo.MarkFailed(queueItem.ID, "duplicate transaction reference"); err != nil {
		return err
	}

	s.auditLogger.LogTransactionStateChange(ctx, transaction.ID, models.TransactionStatusPending, models.TransactionStatusFailed)

	return ErrDuplicateReference
}

func (s *TransactionProcessingService) GetQueueMetrics() (*dto.QueueMetrics, error) {
	pendingCount, err := s.queueRepo.GetPendingCount()
	if err != nil {
		return nil, err
	}

	processingCount, err := s.queueRepo.GetProcessingCount()
	if err != nil {
		return nil, err
	}

	completedCount, err := s.queueRepo.GetCompletedCount()
	if err != nil {
		return nil, err
	}

	failedCount, err := s.queueRepo.GetFailedCount()
	if err != nil {
		return nil, err
	}

	avgProcessingMs, err := s.queueRepo.GetAverageProcessingTime()
	if err != nil {
		return nil, err
	}

	oldestPending, err := s.queueRepo.GetOldestPendingAge()
	if err != nil {
		return nil, err
	}

	return &dto.QueueMetrics{
		PendingCount:    pendingCount,
		ProcessingCount: processingCount,
		CompletedCount:  completedCount,
		FailedCount:     failedCount,
		AvgProcessingMs: avgProcessingMs,
		OldestPending:   oldestPending,
	}, nil
}
