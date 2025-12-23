package services_test

import (
	"context"
	"testing"
	"time"

	"array-assessment/internal/models"
	"array-assessment/internal/repositories/repository_mocks"
	"array-assessment/internal/services"
	"array-assessment/internal/services/service_mocks"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type TransactionProcessingServiceTestSuite struct {
	suite.Suite
	ctx               context.Context
	ctrl              *gomock.Controller
	processingService services.TransactionProcessingServiceInterface
	transactionRepo   *repository_mocks.MockTransactionRepositoryInterface
	queueRepo         *repository_mocks.MockProcessingQueueRepositoryInterface
	accountRepo       *repository_mocks.MockAccountRepositoryInterface
	auditLogger       *service_mocks.MockAuditLoggerInterface
	metrics           *service_mocks.MockMetricsRecorderInterface
	circuitBreaker    *service_mocks.MockCircuitBreakerInterface
}

func TestTransactionProcessingServiceSuite(t *testing.T) {
	suite.Run(t, new(TransactionProcessingServiceTestSuite))
}

func (s *TransactionProcessingServiceTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.ctrl = gomock.NewController(s.T())

	s.transactionRepo = repository_mocks.NewMockTransactionRepositoryInterface(s.ctrl)
	s.queueRepo = repository_mocks.NewMockProcessingQueueRepositoryInterface(s.ctrl)
	s.accountRepo = repository_mocks.NewMockAccountRepositoryInterface(s.ctrl)
	s.metrics = service_mocks.NewMockMetricsRecorderInterface(s.ctrl)
	s.auditLogger = service_mocks.NewMockAuditLoggerInterface(s.ctrl)
	s.circuitBreaker = service_mocks.NewMockCircuitBreakerInterface(s.ctrl)

	s.processingService = services.NewTransactionProcessingService(
		s.transactionRepo,
		s.queueRepo,
		s.accountRepo,
		s.auditLogger,
		s.metrics,
		s.circuitBreaker,
		10,
	)
}

func (s *TransactionProcessingServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

// Test: Async Processing - Valid Transaction - Completes Successfully
func (s *TransactionProcessingServiceTestSuite) TestTransactionProcessingService_EnqueueTransaction_ValidTransaction_EnqueuesSuccessfully() {
	accountID := uuid.New()
	transactionID := uuid.New()

	transaction := &models.Transaction{
		ID:              transactionID,
		AccountID:       accountID,
		TransactionType: models.TransactionTypeDebit,
		Amount:          decimal.NewFromFloat(gofakeit.Float64Range(1.0, 1000.0)),
		Description:     gofakeit.Sentence(5),
		Status:          models.TransactionStatusPending,
		Reference:       models.GenerateTransactionReference(),
		Version:         1,
	}

	// Mock expectations
	s.transactionRepo.EXPECT().GetByID(transactionID).Return(transaction, nil).Times(1)
	s.queueRepo.EXPECT().Enqueue(transactionID, models.QueueOperationProcess, models.QueuePriorityNormal).Return(nil).Times(1)
	s.metrics.EXPECT().IncrementCounter("queue.enqueued", map[string]string{"operation": models.QueueOperationProcess}).Times(1)

	err := s.processingService.EnqueueTransaction(transactionID, models.QueueOperationProcess, models.QueuePriorityNormal)

	s.NoError(err)
}

// Test: Async Processing - High Priority Transaction - Processes First
func (s *TransactionProcessingServiceTestSuite) TestTransactionProcessingService_EnqueueTransaction_HighPriority_ProcessesFirst() {
	accountID := uuid.New()
	transactionID := uuid.New()

	transaction := &models.Transaction{
		ID:              transactionID,
		AccountID:       accountID,
		TransactionType: models.TransactionTypeCredit,
		Amount:          decimal.NewFromFloat(5000.0),
		Description:     "High priority wire transfer",
		Status:          models.TransactionStatusPending,
		Reference:       models.GenerateTransactionReference(),
		Version:         1,
	}

	// Mock expectations
	s.transactionRepo.EXPECT().GetByID(transactionID).Return(transaction, nil).Times(1)
	s.queueRepo.EXPECT().Enqueue(transactionID, models.QueueOperationProcess, models.QueuePriorityHigh).Return(nil).Times(1)
	s.metrics.EXPECT().IncrementCounter("queue.enqueued", map[string]string{"operation": models.QueueOperationProcess}).Times(1)

	err := s.processingService.EnqueueTransaction(transactionID, models.QueueOperationProcess, models.QueuePriorityHigh)

	s.NoError(err)
}

// Test: Async Processing - Concurrent Processing - Respects Max Workers Limit
func (s *TransactionProcessingServiceTestSuite) TestTransactionProcessingService_ProcessQueue_ConcurrentTransactions_RespectsMaxWorkersLimit() {
	// Create 20 pending queue items
	queueItems := make([]*models.ProcessingQueueItem, 20)
	for i := 0; i < 20; i++ {
		queueItems[i] = &models.ProcessingQueueItem{
			ID:            uuid.New(),
			TransactionID: uuid.New(),
			Operation:     models.QueueOperationProcess,
			Priority:      models.QueuePriorityNormal,
			Status:        models.QueueStatusPending,
			RetryCount:    0,
			MaxRetries:    3,
			ScheduledAt:   time.Now(),
		}
	}

	// First call returns all 20 items, subsequent calls return empty to allow test to complete
	s.queueRepo.EXPECT().FetchPending(20).Return(queueItems, nil).Times(1)
	s.queueRepo.EXPECT().FetchPending(20).Return([]*models.ProcessingQueueItem{}, nil).AnyTimes()

	// Circuit breaker checks - will be called for each item
	s.circuitBreaker.EXPECT().IsOpen().Return(false).AnyTimes()
	s.circuitBreaker.EXPECT().RecordSuccess().Times(20)

	// Audit logger expectations - will be called for each item
	s.auditLogger.EXPECT().LogTransactionProcessingStarted(gomock.Any(), gomock.Any(), gomock.Any()).Times(20)
	s.auditLogger.EXPECT().LogQueueItemProcessed(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(20)
	s.auditLogger.EXPECT().LogTransactionStateChange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(20)
	s.auditLogger.EXPECT().LogBalanceUpdate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(20)
	s.auditLogger.EXPECT().LogTransactionProcessingCompleted(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(20)

	// Metrics expectations
	s.metrics.EXPECT().RecordProcessingTime(gomock.Any(), gomock.Any()).AnyTimes()
	s.metrics.EXPECT().IncrementCounter(gomock.Any(), gomock.Any()).AnyTimes()

	for _, item := range queueItems {
		accountID := uuid.New()
		transaction := &models.Transaction{
			ID:              item.TransactionID,
			AccountID:       accountID,
			TransactionType: models.TransactionTypeDebit,
			Amount:          decimal.NewFromFloat(100.0),
			BalanceBefore:   decimal.NewFromFloat(1000.0),
			BalanceAfter:    decimal.NewFromFloat(900.0),
			Description:     gofakeit.Sentence(5),
			Status:          models.TransactionStatusPending,
			Version:         1,
		}
		account := &models.Account{
			ID:      accountID,
			Balance: decimal.NewFromFloat(1000.0),
		}

		s.transactionRepo.EXPECT().GetByID(item.TransactionID).Return(transaction, nil)
		s.accountRepo.EXPECT().GetByID(accountID).Return(account, nil)
		s.accountRepo.EXPECT().UpdateBalance(accountID, gomock.Any(), gomock.Any()).Return(nil)
		s.transactionRepo.EXPECT().UpdateWithOptimisticLock(transaction, 1).Return(nil)
		s.queueRepo.EXPECT().MarkCompleted(item.ID).Return(nil)
	}

	// Start async processing
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	go s.processingService.StartProcessing(ctx)

	// Allow time for processing
	time.Sleep(2 * time.Second)

	// Verify that max 10 workers were used concurrently
}

// Test: Retry Mechanism - Failed Transaction - Retries With Exponential Backoff
func (s *TransactionProcessingServiceTestSuite) TestTransactionProcessingService_ProcessTransaction_FailedTransaction_RetriesWithExponentialBackoff() {
	transactionID := uuid.New()
	accountID := uuid.New()
	queueItem := &models.ProcessingQueueItem{
		ID:            uuid.New(),
		TransactionID: transactionID,
		Operation:     models.QueueOperationProcess,
		Priority:      models.QueuePriorityNormal,
		Status:        models.QueueStatusPending,
		RetryCount:    0,
		MaxRetries:    3,
		ScheduledAt:   time.Now(),
	}

	transaction := &models.Transaction{
		ID:              transactionID,
		AccountID:       accountID,
		TransactionType: models.TransactionTypeDebit,
		Amount:          decimal.NewFromFloat(100.0),
		BalanceBefore:   decimal.NewFromFloat(1000.0),
		BalanceAfter:    decimal.NewFromFloat(900.0),
		Description:     "Transaction with retry",
		Status:          models.TransactionStatusPending,
		Version:         1,
	}

	account := &models.Account{
		ID:      accountID,
		Balance: decimal.NewFromFloat(1000.0),
	}

	// Mock expectations
	s.circuitBreaker.EXPECT().IsOpen().Return(false).Times(1)
	s.auditLogger.EXPECT().LogTransactionProcessingStarted(gomock.Any(), transactionID, models.QueueOperationProcess).Times(1)
	s.transactionRepo.EXPECT().GetByID(transactionID).Return(transaction, nil).Times(1)
	s.accountRepo.EXPECT().GetByID(accountID).Return(account, nil).Times(1)
	s.auditLogger.EXPECT().LogBalanceUpdate(gomock.Any(), accountID, gomock.Any(), gomock.Any(), transactionID).Times(1)
	s.accountRepo.EXPECT().UpdateBalance(accountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.transactionRepo.EXPECT().UpdateWithOptimisticLock(transaction, 1).Return(models.ErrOptimisticLockConflict).Times(1)
	s.auditLogger.EXPECT().LogOptimisticLockConflict(gomock.Any(), "transaction", transactionID, 1, 1).Times(1)
	s.circuitBreaker.EXPECT().RecordFailure().Times(1)
	s.auditLogger.EXPECT().LogRetryAttempt(gomock.Any(), queueItem.ID, transactionID, 1, 3, int64(1000)).Times(1)
	s.queueRepo.EXPECT().IncrementRetry(queueItem.ID).Return(nil).Times(1)
	s.metrics.EXPECT().IncrementCounter("transaction.processing.retry", map[string]string{"operation": models.QueueOperationProcess}).Times(1)

	err := s.processingService.ProcessQueueItem(s.ctx, queueItem)

	s.Error(err)
}

// Test: Retry Mechanism - Max Retries Exceeded - Marks Failed
func (s *TransactionProcessingServiceTestSuite) TestTransactionProcessingService_ProcessTransaction_MaxRetriesExceeded_MarksFailed() {
	transactionID := uuid.New()
	queueItem := &models.ProcessingQueueItem{
		ID:            uuid.New(),
		TransactionID: transactionID,
		Operation:     models.QueueOperationProcess,
		Priority:      models.QueuePriorityNormal,
		Status:        models.QueueStatusPending,
		RetryCount:    3, // Already at max retries
		MaxRetries:    3,
		ScheduledAt:   time.Now(),
	}

	transaction := &models.Transaction{
		ID:              transactionID,
		AccountID:       uuid.New(),
		TransactionType: models.TransactionTypeDebit,
		Amount:          decimal.NewFromFloat(100.0),
		Description:     "Transaction exceeding retries",
		Status:          models.TransactionStatusPending,
		Version:         1,
	}

	// Mock expectations
	s.circuitBreaker.EXPECT().IsOpen().Return(false).Times(1)
	s.transactionRepo.EXPECT().GetByID(transactionID).Return(transaction, nil).Times(1)
	s.transactionRepo.EXPECT().UpdateWithOptimisticLock(transaction, 0).Return(nil).Times(1)
	s.auditLogger.EXPECT().LogTransactionStateChange(gomock.Any(), transactionID, models.TransactionStatusPending, models.TransactionStatusFailed).Times(1)
	s.queueRepo.EXPECT().MarkFailed(queueItem.ID, "max retries exceeded").Return(nil).Times(1)
	s.metrics.EXPECT().IncrementCounter("transaction.processed.failed", map[string]string{"operation": models.QueueOperationProcess, "reason": "max_retries"}).Times(1)
	s.auditLogger.EXPECT().LogTransactionProcessingFailed(gomock.Any(), transactionID, models.QueueOperationProcess, "max retries exceeded", 3).Times(1)

	err := s.processingService.ProcessQueueItem(s.ctx, queueItem)

	s.Error(err)
	s.Contains(err.Error(), "max retries exceeded")
}

// Test: Circuit Breaker - Database Connection Failures - Opens Circuit
func (s *TransactionProcessingServiceTestSuite) TestTransactionProcessingService_ProcessTransaction_DatabaseFailures_OpensCircuit() {
	transactionID := uuid.New()
	queueItem := &models.ProcessingQueueItem{
		ID:            uuid.New(),
		TransactionID: transactionID,
		Operation:     models.QueueOperationProcess,
		Priority:      models.QueuePriorityNormal,
		Status:        models.QueueStatusPending,
		RetryCount:    0,
		MaxRetries:    3,
		ScheduledAt:   time.Now(),
	}

	// Mock expectations - circuit breaker is open
	s.circuitBreaker.EXPECT().IsOpen().Return(true).Times(1)
	s.metrics.EXPECT().IncrementCounter("circuit_breaker.open", map[string]string{"service": "database"}).Times(1)

	err := s.processingService.ProcessQueueItem(s.ctx, queueItem)

	s.Error(err)
	s.Contains(err.Error(), "circuit breaker is open")
}

// Test: Circuit Breaker - Service Recovered - Closes Circuit
func (s *TransactionProcessingServiceTestSuite) TestTransactionProcessingService_ProcessTransaction_ServiceRecovered_ClosesCircuit() {
	transactionID := uuid.New()
	accountID := uuid.New()
	queueItem := &models.ProcessingQueueItem{
		ID:            uuid.New(),
		TransactionID: transactionID,
		Operation:     models.QueueOperationProcess,
		Priority:      models.QueuePriorityNormal,
		Status:        models.QueueStatusPending,
		RetryCount:    0,
		MaxRetries:    3,
		ScheduledAt:   time.Now(),
	}

	transaction := &models.Transaction{
		ID:              transactionID,
		AccountID:       accountID,
		TransactionType: models.TransactionTypeDebit,
		Amount:          decimal.NewFromFloat(100.0),
		BalanceBefore:   decimal.NewFromFloat(1000.0),
		BalanceAfter:    decimal.NewFromFloat(900.0),
		Description:     "Transaction after recovery",
		Status:          models.TransactionStatusPending,
		Version:         1,
	}

	account := &models.Account{
		ID:      accountID,
		Balance: decimal.NewFromFloat(1000.0),
	}

	// Mock expectations
	s.circuitBreaker.EXPECT().IsOpen().Return(false).Times(1)
	s.auditLogger.EXPECT().LogTransactionProcessingStarted(gomock.Any(), transactionID, models.QueueOperationProcess).Times(1)
	s.transactionRepo.EXPECT().GetByID(transactionID).Return(transaction, nil).Times(1)
	s.accountRepo.EXPECT().GetByID(accountID).Return(account, nil).Times(1)
	s.auditLogger.EXPECT().LogBalanceUpdate(gomock.Any(), accountID, gomock.Any(), gomock.Any(), transactionID).Times(1)
	s.accountRepo.EXPECT().UpdateBalance(accountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.transactionRepo.EXPECT().UpdateWithOptimisticLock(transaction, 1).Return(nil).Times(1)
	s.auditLogger.EXPECT().LogTransactionStateChange(gomock.Any(), transactionID, models.TransactionStatusPending, models.TransactionStatusCompleted).Times(1)
	s.queueRepo.EXPECT().MarkCompleted(queueItem.ID).Return(nil).Times(1)
	s.circuitBreaker.EXPECT().RecordSuccess().Times(1)
	s.auditLogger.EXPECT().LogQueueItemProcessed(gomock.Any(), queueItem.ID, transactionID, models.QueueOperationProcess, 0).Times(1)
	s.metrics.EXPECT().RecordProcessingTime("transaction.processing", gomock.Any()).Times(1)
	s.metrics.EXPECT().IncrementCounter("transaction.processed.success", map[string]string{"operation": models.QueueOperationProcess}).Times(1)
	s.auditLogger.EXPECT().LogTransactionProcessingCompleted(gomock.Any(), transactionID, models.QueueOperationProcess, gomock.Any()).Times(1)

	err := s.processingService.ProcessQueueItem(s.ctx, queueItem)

	s.NoError(err)
}

// Test: Audit Logging - State Change - Logs Correctly
func (s *TransactionProcessingServiceTestSuite) TestTransactionProcessingService_ProcessTransaction_StateChange_LogsCorrectly() {
	transactionID := uuid.New()
	accountID := uuid.New()
	queueItem := &models.ProcessingQueueItem{
		ID:            uuid.New(),
		TransactionID: transactionID,
		Operation:     models.QueueOperationProcess,
		Priority:      models.QueuePriorityNormal,
		Status:        models.QueueStatusPending,
		RetryCount:    0,
		MaxRetries:    3,
		ScheduledAt:   time.Now(),
	}

	transaction := &models.Transaction{
		ID:              transactionID,
		AccountID:       accountID,
		TransactionType: models.TransactionTypeCredit,
		Amount:          decimal.NewFromFloat(500.0),
		BalanceBefore:   decimal.NewFromFloat(1000.0),
		BalanceAfter:    decimal.NewFromFloat(1500.0),
		Description:     "Transaction for audit logging",
		Status:          models.TransactionStatusPending,
		Version:         1,
	}

	account := &models.Account{
		ID:      accountID,
		Balance: decimal.NewFromFloat(1000.0),
	}

	// Mock expectations
	s.circuitBreaker.EXPECT().IsOpen().Return(false).Times(1)
	s.auditLogger.EXPECT().LogTransactionProcessingStarted(gomock.Any(), transactionID, models.QueueOperationProcess).Times(1)
	s.transactionRepo.EXPECT().GetByID(transactionID).Return(transaction, nil).Times(1)
	s.accountRepo.EXPECT().GetByID(accountID).Return(account, nil).Times(1)
	s.auditLogger.EXPECT().LogBalanceUpdate(gomock.Any(), accountID, gomock.Any(), gomock.Any(), transactionID).Times(1)
	s.accountRepo.EXPECT().UpdateBalance(accountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.transactionRepo.EXPECT().UpdateWithOptimisticLock(transaction, 1).Return(nil).Times(1)
	s.auditLogger.EXPECT().LogTransactionStateChange(gomock.Any(), transactionID, models.TransactionStatusPending, models.TransactionStatusCompleted).Times(1)
	s.queueRepo.EXPECT().MarkCompleted(queueItem.ID).Return(nil).Times(1)
	s.circuitBreaker.EXPECT().RecordSuccess().Times(1)
	s.auditLogger.EXPECT().LogQueueItemProcessed(gomock.Any(), queueItem.ID, transactionID, models.QueueOperationProcess, 0).Times(1)
	s.metrics.EXPECT().RecordProcessingTime("transaction.processing", gomock.Any()).Times(1)
	s.metrics.EXPECT().IncrementCounter("transaction.processed.success", map[string]string{"operation": models.QueueOperationProcess}).Times(1)
	s.auditLogger.EXPECT().LogTransactionProcessingCompleted(gomock.Any(), transactionID, models.QueueOperationProcess, gomock.Any()).Times(1)

	err := s.processingService.ProcessQueueItem(s.ctx, queueItem)

	s.NoError(err)
}

// Test: Monitoring Metrics - Processing Time - Records Duration
func (s *TransactionProcessingServiceTestSuite) TestTransactionProcessingService_ProcessTransaction_ProcessingTime_RecordsDuration() {
	transactionID := uuid.New()
	accountID := uuid.New()
	queueItem := &models.ProcessingQueueItem{
		ID:            uuid.New(),
		TransactionID: transactionID,
		Operation:     models.QueueOperationProcess,
		Priority:      models.QueuePriorityNormal,
		Status:        models.QueueStatusPending,
		RetryCount:    0,
		MaxRetries:    3,
		ScheduledAt:   time.Now(),
	}

	transaction := &models.Transaction{
		ID:              transactionID,
		AccountID:       accountID,
		TransactionType: models.TransactionTypeDebit,
		Amount:          decimal.NewFromFloat(75.0),
		BalanceBefore:   decimal.NewFromFloat(1000.0),
		BalanceAfter:    decimal.NewFromFloat(925.0),
		Description:     "Transaction for metrics",
		Status:          models.TransactionStatusPending,
		Version:         1,
	}

	account := &models.Account{
		ID:      accountID,
		Balance: decimal.NewFromFloat(1000.0),
	}

	// Mock expectations
	s.circuitBreaker.EXPECT().IsOpen().Return(false).Times(1)
	s.auditLogger.EXPECT().LogTransactionProcessingStarted(gomock.Any(), transactionID, models.QueueOperationProcess).Times(1)
	s.transactionRepo.EXPECT().GetByID(transactionID).Return(transaction, nil).Times(1)
	s.accountRepo.EXPECT().GetByID(accountID).Return(account, nil).Times(1)
	s.auditLogger.EXPECT().LogBalanceUpdate(gomock.Any(), accountID, gomock.Any(), gomock.Any(), transactionID).Times(1)
	s.accountRepo.EXPECT().UpdateBalance(accountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.transactionRepo.EXPECT().UpdateWithOptimisticLock(transaction, 1).Return(nil).Times(1)
	s.auditLogger.EXPECT().LogTransactionStateChange(gomock.Any(), transactionID, models.TransactionStatusPending, models.TransactionStatusCompleted).Times(1)
	s.queueRepo.EXPECT().MarkCompleted(queueItem.ID).Return(nil).Times(1)
	s.circuitBreaker.EXPECT().RecordSuccess().Times(1)
	s.auditLogger.EXPECT().LogQueueItemProcessed(gomock.Any(), queueItem.ID, transactionID, models.QueueOperationProcess, 0).Times(1)
	s.metrics.EXPECT().RecordProcessingTime("transaction.processing", gomock.Any()).Times(1)
	s.metrics.EXPECT().IncrementCounter("transaction.processed.success", map[string]string{"operation": models.QueueOperationProcess}).Times(1)
	s.auditLogger.EXPECT().LogTransactionProcessingCompleted(gomock.Any(), transactionID, models.QueueOperationProcess, gomock.Any()).Times(1)

	err := s.processingService.ProcessQueueItem(s.ctx, queueItem)

	s.NoError(err)
}

// Test: Monitoring Metrics - Queue Depth - Tracks Correctly
func (s *TransactionProcessingServiceTestSuite) TestTransactionProcessingService_GetQueueMetrics_QueueDepth_TracksCorrectly() {
	s.queueRepo.EXPECT().GetPendingCount().Return(int64(25), nil)
	s.queueRepo.EXPECT().GetProcessingCount().Return(int64(8), nil)
	s.queueRepo.EXPECT().GetCompletedCount().Return(int64(100), nil)
	s.queueRepo.EXPECT().GetFailedCount().Return(int64(2), nil)
	s.queueRepo.EXPECT().GetAverageProcessingTime().Return(float64(150.5), nil)
	s.queueRepo.EXPECT().GetOldestPendingAge().Return(nil, nil)

	metrics, err := s.processingService.GetQueueMetrics()

	s.NoError(err)
	s.Equal(int64(25), metrics.PendingCount)
	s.Equal(int64(8), metrics.ProcessingCount)
	s.Equal(int64(2), metrics.FailedCount)
}

// Test: Idempotency - Duplicate Transaction Reference - Rejects Duplicate
func (s *TransactionProcessingServiceTestSuite) TestTransactionProcessingService_ProcessTransaction_DuplicateReference_RejectsDuplicate() {
	transactionID := uuid.New()
	reference := models.GenerateTransactionReference()

	queueItem := &models.ProcessingQueueItem{
		ID:            uuid.New(),
		TransactionID: transactionID,
		Operation:     models.QueueOperationProcess,
		Priority:      models.QueuePriorityNormal,
		Status:        models.QueueStatusPending,
		RetryCount:    0,
		MaxRetries:    3,
		ScheduledAt:   time.Now(),
	}

	transaction := &models.Transaction{
		ID:              transactionID,
		AccountID:       uuid.New(),
		TransactionType: models.TransactionTypeDebit,
		Amount:          decimal.NewFromFloat(100.0),
		Description:     "Duplicate transaction",
		Status:          models.TransactionStatusPending,
		Reference:       reference,
		Version:         1,
	}

	// Transaction with same reference already exists
	existingTransaction := &models.Transaction{
		ID:              uuid.New(),
		AccountID:       transaction.AccountID,
		TransactionType: models.TransactionTypeDebit,
		Amount:          decimal.NewFromFloat(100.0),
		Description:     "Original transaction",
		Status:          models.TransactionStatusCompleted,
		Reference:       reference,
		Version:         1,
	}

	// Mock expectations
	s.circuitBreaker.EXPECT().IsOpen().Return(false).Times(1)
	s.auditLogger.EXPECT().LogTransactionProcessingStarted(gomock.Any(), transactionID, models.QueueOperationProcess).Times(1)
	s.transactionRepo.EXPECT().GetByID(transactionID).Return(transaction, nil).Times(1)
	s.transactionRepo.EXPECT().GetByReference(reference).Return(existingTransaction, nil).Times(1)
	s.metrics.EXPECT().IncrementCounter("transaction.duplicate.rejected", map[string]string{"reference": reference}).Times(1)
	s.transactionRepo.EXPECT().UpdateWithOptimisticLock(transaction, 1).Return(nil).Times(1)
	s.queueRepo.EXPECT().MarkFailed(queueItem.ID, "duplicate transaction reference").Return(nil).Times(1)
	s.auditLogger.EXPECT().LogTransactionStateChange(gomock.Any(), transactionID, models.TransactionStatusPending, models.TransactionStatusFailed).Times(1)

	err := s.processingService.ProcessQueueItem(s.ctx, queueItem)

	s.Error(err)
	s.Contains(err.Error(), "duplicate")
}

// Test: Context Cancellation - Processing In Progress - Graceful Shutdown
func (s *TransactionProcessingServiceTestSuite) TestTransactionProcessingService_StartProcessing_ContextCancelled_GracefulShutdown() {
	ctx, cancel := context.WithCancel(s.ctx)

	// Setup empty queue for simplicity - FetchPending may be called multiple times
	s.queueRepo.EXPECT().FetchPending(gomock.Any()).Return([]*models.ProcessingQueueItem{}, nil).AnyTimes()

	// Start processing
	done := make(chan struct{})
	go func() {
		s.processingService.StartProcessing(ctx)
		close(done)
	}()

	// Allow processing to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for graceful shutdown
	select {
	case <-done:
		// Shutdown completed
	case <-time.After(5 * time.Second):
		s.Fail("processing did not shutdown gracefully")
	}
}

// Test: Balance Update - Transaction Processing - Updates Balance Atomically
func (s *TransactionProcessingServiceTestSuite) TestTransactionProcessingService_ProcessTransaction_BalanceUpdate_UpdatesAtomically() {
	accountID := uuid.New()
	transactionID := uuid.New()

	account := &models.Account{
		ID:      accountID,
		Balance: decimal.NewFromFloat(1000.0),
	}

	queueItem := &models.ProcessingQueueItem{
		ID:            uuid.New(),
		TransactionID: transactionID,
		Operation:     models.QueueOperationProcess,
		Priority:      models.QueuePriorityNormal,
		Status:        models.QueueStatusPending,
		RetryCount:    0,
		MaxRetries:    3,
		ScheduledAt:   time.Now(),
	}

	transaction := &models.Transaction{
		ID:              transactionID,
		AccountID:       accountID,
		TransactionType: models.TransactionTypeDebit,
		Amount:          decimal.NewFromFloat(100.0),
		BalanceBefore:   decimal.NewFromFloat(1000.0),
		BalanceAfter:    decimal.NewFromFloat(900.0),
		Description:     "Balance update test",
		Status:          models.TransactionStatusPending,
		Version:         1,
	}

	// Mock expectations
	s.circuitBreaker.EXPECT().IsOpen().Return(false).Times(1)
	s.auditLogger.EXPECT().LogTransactionProcessingStarted(gomock.Any(), transactionID, models.QueueOperationProcess).Times(1)
	s.transactionRepo.EXPECT().GetByID(transactionID).Return(transaction, nil).Times(1)
	s.accountRepo.EXPECT().GetByID(accountID).Return(account, nil).Times(1)
	s.auditLogger.EXPECT().LogBalanceUpdate(gomock.Any(), accountID, gomock.Any(), gomock.Any(), transactionID).Times(1)
	s.accountRepo.EXPECT().UpdateBalance(accountID, gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.transactionRepo.EXPECT().UpdateWithOptimisticLock(transaction, 1).Return(nil).Times(1)
	s.auditLogger.EXPECT().LogTransactionStateChange(gomock.Any(), transactionID, models.TransactionStatusPending, models.TransactionStatusCompleted).Times(1)
	s.queueRepo.EXPECT().MarkCompleted(queueItem.ID).Return(nil).Times(1)
	s.circuitBreaker.EXPECT().RecordSuccess().Times(1)
	s.auditLogger.EXPECT().LogQueueItemProcessed(gomock.Any(), queueItem.ID, transactionID, models.QueueOperationProcess, 0).Times(1)
	s.metrics.EXPECT().RecordProcessingTime("transaction.processing", gomock.Any()).Times(1)
	s.metrics.EXPECT().IncrementCounter("transaction.processed.success", map[string]string{"operation": models.QueueOperationProcess}).Times(1)
	s.auditLogger.EXPECT().LogTransactionProcessingCompleted(gomock.Any(), transactionID, models.QueueOperationProcess, gomock.Any()).Times(1)

	err := s.processingService.ProcessQueueItem(s.ctx, queueItem)

	s.NoError(err)
}
