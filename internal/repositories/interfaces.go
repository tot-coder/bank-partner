package repositories

import (
	"time"

	"array-assessment/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccountRepositoryInterface defines the contract for account repository operations
type AccountRepositoryInterface interface {
	Create(account *models.Account) error
	GetByID(id uuid.UUID) (*models.Account, error)
	GetByAccountNumber(accountNumber string) (*models.Account, error)
	GetByUserID(userID uuid.UUID) ([]models.Account, error)
	GetByUserIDAndType(userID uuid.UUID, accountType string) ([]models.Account, error)
	GetByUserIDExcludingStatus(userID uuid.UUID, excludeStatus string) ([]*models.Account, error)
	GetAll(offset, limit int) ([]models.Account, int64, error)
	GetAllWithFilters(filters models.AccountFilters, offset, limit int) ([]models.Account, int64, error)
	Update(account *models.Account) error
	UpdateOwnership(accountID, newUserID uuid.UUID) error
	Delete(id uuid.UUID) error
	SoftDeleteByUserID(userID uuid.UUID) error
	CheckAccountNumberExists(accountNumber string) (bool, error)
	GenerateUniqueAccountNumber(accountType string) (string, error)
	CreateWithTransaction(account *models.Account, transactions []models.Transaction) error
	UpdateBalance(accountID uuid.UUID, amount decimal.Decimal, transactionType string) error
	GetAccountsByStatus(status string, offset, limit int) ([]models.Account, error)
	GetTotalBalanceByUserID(userID uuid.UUID) (decimal.Decimal, error)
	ExistsForUser(userID uuid.UUID, accountType string) (bool, error)
	ExecuteAtomicTransfer(fromAccountID, toAccountID uuid.UUID, amount decimal.Decimal, fromDescription, toDescription string) (debitTxID, creditTxID uuid.UUID, err error)
}

// TransactionRepositoryInterface defines the contract for transaction repository operations
type TransactionRepositoryInterface interface {
	Create(transaction *models.Transaction) error
	GetByID(id uuid.UUID) (*models.Transaction, error)
	GetByAccountID(accountID uuid.UUID, offset, limit int) ([]models.Transaction, int64, error)
	GetByReference(reference string) (*models.Transaction, error)
	GetRecentByAccountID(accountID uuid.UUID, limit int) ([]models.Transaction, error)
	GetByDateRange(accountID uuid.UUID, startDate, endDate time.Time) ([]models.Transaction, error)
	CreateBatch(transactions []models.Transaction) error
	GetPendingTransactions(offset, limit int) ([]models.Transaction, error)
	UpdateStatus(id uuid.UUID, status string) error
	GetTotalsByAccountID(accountID uuid.UUID) (credits, debits int64, creditAmount, debitAmount string, err error)

	// Enhanced methods for category and filtering
	GetByCategory(accountID uuid.UUID, category string, offset, limit int) ([]models.Transaction, int64, error)
	GetWithFilters(filters models.TransactionFilters) ([]models.Transaction, int64, error)
	UpdateWithOptimisticLock(transaction *models.Transaction, expectedVersion int) error
	GetExpiredPendingTransactions(limit int) ([]models.Transaction, error)
	GetCategorySummary(accountID uuid.UUID, startDate, endDate time.Time) ([]models.CategorySummary, error)
}

// UserSearchCriteria defines search criteria for users
type UserSearchCriteria struct {
	Query      string
	SearchType string // "first_name", "last_name", "name", "email", "account_number"
}

// UserRepositoryInterface defines the contract for user repository operations
type UserRepositoryInterface interface {
	Create(user *models.User) error
	GetByID(id uuid.UUID) (*models.User, error)
	GetByIDActive(id uuid.UUID) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByEmailExcluding(email string, excludeUserID uuid.UUID) (*models.User, error)
	SearchUsers(criteria UserSearchCriteria, offset, limit int) ([]*models.User, int64, error)
	Update(user *models.User) error
	UpdateFields(userID uuid.UUID, fields map[string]interface{}) error
	UpdateEmail(userID uuid.UUID, newEmail string) error
	UpdatePasswordHash(userID uuid.UUID, passwordHash string) error
	UpdateFailedLoginAttempts(user *models.User) error
	ResetFailedLoginAttempts(userID uuid.UUID) error
	UnlockAccount(userID uuid.UUID) error
	Delete(userID uuid.UUID) error
	ListUsers(offset, limit int) ([]*models.User, int64, error)
	CountAccountsByUserID(userID uuid.UUID) (int64, error)
}

// AuditLogRepositoryInterface defines the contract for audit log repository operations
type AuditLogRepositoryInterface interface {
	Create(log *models.AuditLog) error
	GetByID(id uuid.UUID) (*models.AuditLog, error)
	GetByUserID(userID uuid.UUID, offset, limit int) ([]*models.AuditLog, int64, error)
	GetByAction(action string, offset, limit int) ([]*models.AuditLog, int64, error)
	GetByResource(resource, resourceID string, offset, limit int) ([]*models.AuditLog, int64, error)
	GetByIPAddress(ipAddress string, offset, limit int) ([]*models.AuditLog, int64, error)
	GetByTimeRange(startTime, endTime time.Time, offset, limit int) ([]*models.AuditLog, int64, error)
	GetCustomerActivity(userID uuid.UUID, startDate, endDate *time.Time, offset, limit int) ([]*models.AuditLog, int64, error)
	GetFailedLoginAttempts(email string, since time.Time) (int64, error)
	DeleteOlderThan(duration time.Duration) (int64, error)
}

// ProcessingQueueRepositoryInterface defines the contract for transaction processing queue operations
type ProcessingQueueRepositoryInterface interface {
	Enqueue(transactionID uuid.UUID, operation string, priority int) error
	FetchPending(limit int) ([]*models.ProcessingQueueItem, error)
	MarkProcessing(queueItemID uuid.UUID) error
	MarkCompleted(queueItemID uuid.UUID) error
	MarkFailed(queueItemID uuid.UUID, errorMessage string) error
	IncrementRetry(queueItemID uuid.UUID) error
	GetPendingCount() (int64, error)
	GetProcessingCount() (int64, error)
	GetFailedCount() (int64, error)
	GetCompletedCount() (int64, error)
	GetAverageProcessingTime() (float64, error)
	GetOldestPendingAge() (*string, error)
	CleanupCompleted(olderThan time.Duration) (int64, error)
}

// TransferRepositoryInterface defines the contract for transfer repository operations
type TransferRepositoryInterface interface {
	Create(transfer *models.Transfer) error
	Update(transfer *models.Transfer) error
	FindByID(id uuid.UUID) (*models.Transfer, error)
	FindByIdempotencyKey(key string) (*models.Transfer, error)
	FindByUserAccounts(accountIDs []uuid.UUID, offset, limit int) ([]models.Transfer, int64, error)
	FindByUserAccountsWithFilters(accountIDs []uuid.UUID, filters models.TransferFilters, offset, limit int) ([]models.Transfer, int64, error)
	CountByUserAccounts(accountIDs []uuid.UUID) (int64, error)
}

type RefreshTokenRepositoryInterface interface {
	Create(token *models.RefreshToken) error
	GetByID(id uuid.UUID) (*models.RefreshToken, error)
	GetByTokenHash(tokenHash string) (*models.RefreshToken, error)
	GetActiveByUserID(userID uuid.UUID) ([]*models.RefreshToken, error)
	Update(token *models.RefreshToken) error
	Revoke(tokenID uuid.UUID) error
	RevokeAllForUser(userID uuid.UUID) error
	DeleteExpired() (int64, error)
	DeleteRevokedOlderThan(duration time.Duration) (int64, error)
}

// BlacklistedTokenRepositoryInterface defines the contract for blacklisted token repository operations
type BlacklistedTokenRepositoryInterface interface {
	Create(token *models.BlacklistedToken) error
	GetByJTI(jti string) (*models.BlacklistedToken, error)
	DeleteExpired() (int64, error)
}
