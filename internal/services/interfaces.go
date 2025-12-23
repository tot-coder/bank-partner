package services

import (
	"context"
	"time"

	"array-assessment/internal/dto"
	"array-assessment/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccountAssociationServiceInterface defines the contract for account association operations
type AccountAssociationServiceInterface interface {
	GetCustomerAccounts(customerID uuid.UUID) ([]*models.Account, error)
	CreateAccountForCustomer(customerID, performedBy uuid.UUID, accountType, ipAddress, userAgent string) (*models.Account, error)
	TransferAccountOwnership(accountID, fromCustomerID, toCustomerID, performedBy uuid.UUID, ipAddress, userAgent string) error
}

// AccountServiceInterface defines account-related business operations
type AccountServiceInterface interface {
	CreateAccount(userID uuid.UUID, accountType string, initialDeposit decimal.Decimal) (*models.Account, error)
	CreateAccountsForNewUser(userID uuid.UUID) error
	GetAccountByID(accountID uuid.UUID, userID *uuid.UUID) (*models.Account, error)
	GetAccountByNumber(accountNumber string) (*models.Account, error)
	GetUserAccounts(userID uuid.UUID) ([]models.Account, error)
	GetAllAccounts(filters models.AccountFilters, offset, limit int) ([]models.Account, int64, error)
	UpdateAccountStatus(accountID uuid.UUID, userID *uuid.UUID, status string) (*models.Account, error)
	CloseAccount(accountID uuid.UUID, userID uuid.UUID) error
	PerformTransaction(accountID uuid.UUID, amount decimal.Decimal, transactionType, description string, userID *uuid.UUID) (*models.Transaction, error)
	TransferBetweenAccounts(fromAccountID, toAccountID uuid.UUID, amount decimal.Decimal, description, idempotencyKey string, userID uuid.UUID) (*models.Transfer, error)
	GetAccountTransactions(accountID uuid.UUID, userID *uuid.UUID, offset, limit int) ([]models.Transaction, int64, error)
	GetRecentTransactions(accountID uuid.UUID, userID *uuid.UUID, limit int) ([]models.Transaction, error)
	GetUserTransfers(userID uuid.UUID, filters models.TransferFilters, offset, limit int) ([]models.Transfer, int64, error)
}

type AccountSummaryServiceInterface interface {
	GetAccountSummary(requestorID uuid.UUID, targetUserID *uuid.UUID, isAdmin bool) (*models.UserAccountSummary, error)
}

// AuditServiceInterface defines the contract for audit logging operations
type AuditServiceInterface interface {
	CreateAuditLog(log *models.AuditLog) error
	GetCustomerActivity(userID uuid.UUID, startDate, endDate *time.Time, offset, limit int) ([]*models.AuditLog, int64, error)
	LogLogin(userID uuid.UUID, ipAddress, userAgent string) error
	LogLogout(userID uuid.UUID, ipAddress, userAgent string) error
	LogProfileUpdate(userID, performedBy uuid.UUID, ipAddress, userAgent string, changes map[string]interface{}) error
	LogEmailUpdate(userID, performedBy uuid.UUID, oldEmail, newEmail, ipAddress, userAgent string) error
	LogPasswordReset(userID, performedBy uuid.UUID, ipAddress, userAgent string) error
	LogPasswordUpdate(userID uuid.UUID, ipAddress, userAgent string) error
	LogCustomerCreated(userID, performedBy uuid.UUID, ipAddress, userAgent string) error
	LogCustomerDeleted(userID, performedBy uuid.UUID, ipAddress, userAgent string, reason string) error
	LogAccountCreated(userID, performedBy, accountID uuid.UUID, accountType, ipAddress, userAgent string) error
	LogAccountTransferred(fromUserID, toUserID, performedBy, accountID uuid.UUID, ipAddress, userAgent string) error
}

// CategoryServiceInterface defines the interface for transaction categorization operations
type CategoryServiceInterface interface {
	// CategoryFromMCC returns the category for a given MCC code
	CategoryFromMCC(mccCode string) string

	// CategorizeByMerchant categorizes a transaction based on merchant name
	CategorizeByMerchant(merchantName string) (category string, confidence float64)

	// CategorizeByDescription categorizes a transaction based on description
	CategorizeByDescription(description string) (category string, confidence float64)

	// FuzzyMatchMerchant performs fuzzy matching on merchant names
	FuzzyMatchMerchant(input string) (merchant string, score float64)

	// CategorizeTransaction performs complete categorization using all available data
	CategorizeTransaction(transaction *models.Transaction) *models.CategorizationResult

	// BatchCategorize categorizes multiple transactions
	BatchCategorize(transactions []*models.Transaction) []*models.CategorizationResult

	// OverrideCategory manually overrides the category with audit trail
	OverrideCategory(transaction *models.Transaction, newCategory, reason string) error
}

// CustomerProfileServiceInterface defines the contract for customer profile operations
type CustomerProfileServiceInterface interface {
	GetCustomerProfile(customerID uuid.UUID) (*models.User, error)
	CreateCustomer(email, firstName, lastName string, role string) (*models.User, string, error)
	UpdateCustomerProfile(customerID uuid.UUID, updates map[string]interface{}) error
	UpdateCustomerEmail(customerID uuid.UUID, newEmail string) error
	DeleteCustomer(customerID uuid.UUID, reason string) error
}

// CustomerSearchServiceInterface defines the contract for customer search operations
type CustomerSearchServiceInterface interface {
	SearchCustomers(query string, searchType models.SearchType, offset, limit int) ([]*models.CustomerSearchResult, int64, error)
}

// AccountMetricsServiceInterface provides performance metrics and analytics for accounts
type AccountMetricsServiceInterface interface {
	// GetAccountMetrics calculates performance metrics for a single account over a date range
	GetAccountMetrics(requestorID, accountID uuid.UUID, startDate, endDate *time.Time, isAdmin bool) (*models.AccountMetrics, error)

	// GetUserAggregateMetrics calculates aggregate metrics across all accounts for a user
	GetUserAggregateMetrics(requestorID, targetUserID uuid.UUID, startDate, endDate *time.Time, isAdmin bool) (*models.UserAggregateMetrics, error)
}

type MetricsRecorderInterface interface {
	IncrementCounter(name string, tags map[string]string)
	RecordProcessingTime(name string, duration time.Duration)
	RecordGauge(name string, value float64, tags map[string]string)
}

// StatementServiceInterface provides account statement generation
type StatementServiceInterface interface {
	// GenerateStatement generates a monthly or quarterly account statement
	GenerateStatement(requestorID, accountID uuid.UUID, periodType string, year, period int, isAdmin bool) (*models.AccountStatement, error)
}

// TransactionGeneratorInterface generates realistic transaction data for testing
type TransactionGeneratorInterface interface {
	GenerateHistoricalTransactions(accountID uuid.UUID, startDate, endDate time.Time, startingBalance decimal.Decimal, count int) []*models.Transaction
	GenerateSalaryTransactions(accountID uuid.UUID, startDate, endDate time.Time, startingBalance decimal.Decimal) []*models.Transaction
	GenerateBillTransactions(accountID uuid.UUID, startDate, endDate time.Time, startingBalance decimal.Decimal) []*models.Transaction
	GenerateDailyPurchases(accountID uuid.UUID, startDate, endDate time.Time, startingBalance decimal.Decimal) []*models.Transaction
	GetMerchantPool() []models.MerchantInfo
	SelectRandomMerchant() models.MerchantInfo
	GenerateTransactionType() (string, bool)
	GenerateAmount(category string) decimal.Decimal
	GenerateFeeAmount() decimal.Decimal
	GenerateTimestamp(startDate, endDate time.Time) time.Time
}

type AuthServiceInterface interface {
	Register(req *dto.RegisterRequest, ipAddress, userAgent string) (*models.User, error)
	Login(req *dto.LoginRequest, ipAddress, userAgent string) (*dto.TokenResponse, error)
	RefreshTokens(refreshToken, ipAddress, userAgent string) (*dto.TokenResponse, error)
	Logout(accessToken, ipAddress, userAgent string) error
}

type TokenServiceInterface interface {
	GenerateAccessToken(user *models.User) (string, time.Time, error)
	GenerateRefreshToken(userID uuid.UUID) (string, time.Time, error)
	ValidateAccessToken(tokenString string) (*models.CustomClaims, error)
	ValidateRefreshToken(tokenString string) (*models.CustomClaims, error)
	ExtractTokenFromHeader(authHeader string) (string, error)
	GetJTI(tokenString string) (string, error)
	GetTokenExpiry(tokenString string) (time.Time, error)
}

type PasswordServiceInterface interface {
	ValidatePassword(password string) error
	HashPassword(password string) (string, error)
	ComparePassword(password, hash string) bool
	HashPasswordWithoutValidation(password string) (string, error)
	GenerateSecurePassword() (string, error)
	GenerateSecurePasswordWithLength(length int) (string, error)
	PasswordStrength(password string) int
	AdminResetPassword(customerID, adminID uuid.UUID) (string, error)
	CustomerUpdatePassword(customerID uuid.UUID, currentPassword, newPassword string) error
}

type AuditLoggerInterface interface {
	LogTransactionStateChange(ctx context.Context, transactionID uuid.UUID, oldStatus, newStatus string)
	LogTransactionProcessingStarted(ctx context.Context, transactionID uuid.UUID, operation string)
	LogTransactionProcessingCompleted(ctx context.Context, transactionID uuid.UUID, operation string, durationMs int64)
	LogTransactionProcessingFailed(ctx context.Context, transactionID uuid.UUID, operation string, errorMsg string, retryCount int)
	LogBalanceUpdate(ctx context.Context, accountID uuid.UUID, oldBalance, newBalance string, transactionID uuid.UUID)
	LogQueueItemEnqueued(ctx context.Context, queueItemID, transactionID uuid.UUID, operation string, priority int)
	LogQueueItemProcessed(ctx context.Context, queueItemID uuid.UUID, transactionID uuid.UUID, operation string, retryCount int)
	LogCircuitBreakerStateChange(ctx context.Context, service string, oldState, newState string)
	LogRetryAttempt(ctx context.Context, queueItemID uuid.UUID, transactionID uuid.UUID, retryCount, maxRetries int, backoffMs int64)
	LogOptimisticLockConflict(ctx context.Context, entityType string, entityID uuid.UUID, expectedVersion, actualVersion int)
	LogTransferInitiated(ctx context.Context, transferID, fromAccountID, toAccountID uuid.UUID, amount, idempotencyKey string, userID uuid.UUID)
	LogTransferCompleted(ctx context.Context, transferID uuid.UUID, durationMs int64, debitTxID, creditTxID *uuid.UUID)
	LogTransferFailed(ctx context.Context, transferID uuid.UUID, errorMsg string, durationMs int64)
	LogTransferIdempotencyCheck(ctx context.Context, idempotencyKey string, existingTransferID uuid.UUID, status string)
}

type CircuitBreakerInterface interface {
	IsOpen() bool
	RecordSuccess()
	RecordFailure()
	GetState() models.CircuitBreakerState
	Reset()
	GetFailureCount() int
}

type CustomerLoggerInterface interface {
	LogCustomerSearchStarted(ctx context.Context, query string, searchType string, adminUserID uuid.UUID)
	LogCustomerSearchCompleted(ctx context.Context, resultsCount int, durationMs int64)
	LogCustomerSearchFailed(ctx context.Context, errorMsg string, durationMs int64)
	LogCustomerCreated(ctx context.Context, customerID uuid.UUID, email string, adminUserID uuid.UUID)
	LogCustomerProfileUpdated(ctx context.Context, customerID uuid.UUID, updatedFields []string, adminUserID uuid.UUID)
	LogCustomerEmailUpdated(ctx context.Context, customerID uuid.UUID, oldEmail, newEmail string)
	LogCustomerDeleted(ctx context.Context, customerID uuid.UUID, accountsDeactivated int, adminUserID uuid.UUID)
	LogPasswordReset(ctx context.Context, customerID uuid.UUID, adminUserID uuid.UUID)
	LogPasswordChanged(ctx context.Context, customerID uuid.UUID)
	LogAccountOwnershipTransferred(ctx context.Context, accountID, fromCustomerID, toCustomerID, adminUserID uuid.UUID)
	LogAccountCreatedForCustomer(ctx context.Context, accountID, customerID, adminUserID uuid.UUID, accountNumber string)
	LogValidationFailure(ctx context.Context, operation string, errorMsg string)
	LogAuthorizationFailure(ctx context.Context, operation string, userID uuid.UUID, requiredRole string)
}

type TransactionProcessingServiceInterface interface {
	EnqueueTransaction(transactionID uuid.UUID, operation string, priority int) error
	StartProcessing(ctx context.Context)
	ProcessQueueItem(ctx context.Context, queueItem *models.ProcessingQueueItem) error
	GetQueueMetrics() (*dto.QueueMetrics, error)
}
