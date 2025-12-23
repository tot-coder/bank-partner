package services

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"array-assessment/internal/models"
	"array-assessment/internal/repositories"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Performance Optimization Notes:
// The AccountMetricsServiceInterface relies on efficient database queries with proper indexing.
// Required indexes on transactions table (defined in models.Transaction):
//   - account_id: Enables fast filtering by account (existing index)
//   - created_at: Enables efficient date range queries (existing index)
//   - status: Implicit filtering on completed transactions
//
// Query optimization strategy:
//   - Single date range query per account minimizes database round trips
//   - In-memory aggregation after fetch reduces database load
//   - Filters non-completed transactions in application layer
//
// For production optimization considerations:
//   - Consider composite index (account_id, created_at, status) for heavy loads
//   - Monitor query performance with EXPLAIN ANALYZE
//   - Consider read replicas for analytics queries if needed

const (
	DefaultMetricsPeriodDays = 14
)

var (
	ErrInvalidDateRange = errors.New("start date must be before end date")
	ErrFutureDate       = errors.New("end date cannot be in the future")
)

type accountMetricsService struct {
	accountRepo     repositories.AccountRepositoryInterface
	transactionRepo repositories.TransactionRepositoryInterface
	userRepo        repositories.UserRepositoryInterface
}

func NewAccountMetricsService(
	accountRepo repositories.AccountRepositoryInterface,
	transactionRepo repositories.TransactionRepositoryInterface,
	userRepo repositories.UserRepositoryInterface,
) AccountMetricsServiceInterface {
	return &accountMetricsService{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
	}
}

func (s *accountMetricsService) GetAccountMetrics(requestorID, accountID uuid.UUID, startDate, endDate *time.Time, isAdmin bool) (*models.AccountMetrics, error) {
	effectiveStart, effectiveEnd, err := s.validateAndNormalizeDateRange(startDate, endDate)
	if err != nil {
		return nil, err
	}

	requestor, err := s.validateRequestor(requestorID)
	if err != nil {
		return nil, err
	}

	account, err := s.getAndAuthorizeAccount(accountID, requestor, isAdmin)
	if err != nil {
		return nil, err
	}

	transactions, err := s.transactionRepo.GetByDateRange(accountID, effectiveStart, effectiveEnd)
	if err != nil {
		slog.Error("failed to fetch transactions for metrics",
			"account_id", accountID,
			"error", err)
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	metrics := s.calculateAccountMetrics(accountID, transactions, effectiveStart, effectiveEnd, account)

	slog.Info("account metrics generated",
		"account_id", accountID,
		"requestor_id", requestorID,
		"transaction_count", metrics.TransactionCount,
		"date_range", fmt.Sprintf("%s to %s", effectiveStart.Format("2006-01-02"), effectiveEnd.Format("2006-01-02")))

	return metrics, nil
}

func (s *accountMetricsService) GetUserAggregateMetrics(requestorID, targetUserID uuid.UUID, startDate, endDate *time.Time, isAdmin bool) (*models.UserAggregateMetrics, error) {
	effectiveStart, effectiveEnd, err := s.validateAndNormalizeDateRange(startDate, endDate)
	if err != nil {
		return nil, err
	}

	requestor, err := s.validateRequestor(requestorID)
	if err != nil {
		return nil, err
	}

	effectiveUserID, err := s.authorizeUserAccess(requestorID, targetUserID, requestor, isAdmin)
	if err != nil {
		return nil, err
	}

	accounts, err := s.accountRepo.GetByUserID(effectiveUserID)
	if err != nil {
		slog.Error("failed to fetch user accounts for aggregate metrics",
			"user_id", effectiveUserID,
			"error", err)
		return nil, fmt.Errorf("failed to fetch accounts: %w", err)
	}

	aggregateMetrics := s.calculateAggregateMetrics(effectiveUserID, accounts, effectiveStart, effectiveEnd)

	slog.Info("user aggregate metrics generated",
		"user_id", effectiveUserID,
		"requestor_id", requestorID,
		"account_count", aggregateMetrics.AccountCount,
		"total_transaction_count", aggregateMetrics.TotalTransactionCount)

	return aggregateMetrics, nil
}

func (s *accountMetricsService) validateAndNormalizeDateRange(startDate, endDate *time.Time) (time.Time, time.Time, error) {
	now := time.Now()
	var effectiveStart, effectiveEnd time.Time

	if endDate != nil {
		effectiveEnd = *endDate
	} else {
		effectiveEnd = now
	}

	if startDate != nil {
		effectiveStart = *startDate
	} else {
		effectiveStart = effectiveEnd.AddDate(0, 0, -DefaultMetricsPeriodDays)
	}

	if effectiveEnd.After(now) {
		return time.Time{}, time.Time{}, ErrFutureDate
	}

	if effectiveStart.After(effectiveEnd) || effectiveStart.Equal(effectiveEnd) {
		return time.Time{}, time.Time{}, ErrInvalidDateRange
	}

	return effectiveStart, effectiveEnd, nil
}

func (s *accountMetricsService) validateRequestor(requestorID uuid.UUID) (*models.User, error) {
	requestor, err := s.userRepo.GetByID(requestorID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			slog.Warn("user not found during metrics request",
				"requestor_id", requestorID,
				"error", err)
			return nil, ErrNotFound
		}
		slog.Error("failed to get requestor user",
			"requestor_id", requestorID,
			"error", err)
		return nil, fmt.Errorf("failed to verify requestor: %w", err)
	}
	return requestor, nil
}

func (s *accountMetricsService) getAndAuthorizeAccount(accountID uuid.UUID, requestor *models.User, isAdmin bool) (*models.Account, error) {
	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		slog.Error("failed to get account for metrics",
			"account_id", accountID,
			"error", err)
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if account.UserID != requestor.ID {
		if !isAdmin || requestor.Role != models.RoleAdmin {
			slog.Warn("unauthorized access attempt to account metrics",
				"requestor_id", requestor.ID,
				"requestor_role", requestor.Role,
				"account_id", accountID,
				"account_user_id", account.UserID)
			return nil, ErrUnauthorized
		}

		slog.Info("admin accessing account metrics",
			"admin_id", requestor.ID,
			"admin_email", requestor.Email,
			"account_id", accountID,
			"account_user_id", account.UserID)
	}

	return account, nil
}

func (s *accountMetricsService) authorizeUserAccess(requestorID, targetUserID uuid.UUID, requestor *models.User, isAdmin bool) (uuid.UUID, error) {
	if requestorID == targetUserID {
		return requestorID, nil
	}

	if !isAdmin || requestor.Role != models.RoleAdmin {
		slog.Warn("unauthorized access attempt to user aggregate metrics",
			"requestor_id", requestorID,
			"requestor_role", requestor.Role,
			"target_user_id", targetUserID)
		return uuid.Nil, ErrUnauthorized
	}

	targetUser, err := s.userRepo.GetByID(targetUserID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			slog.Warn("target user not found during admin metrics request",
				"target_user_id", targetUserID,
				"error", err)
			return uuid.Nil, ErrNotFound
		}
		slog.Error("failed to get target user",
			"target_user_id", targetUserID,
			"error", err)
		return uuid.Nil, fmt.Errorf("failed to verify target user: %w", err)
	}

	slog.Info("admin accessing user aggregate metrics",
		"admin_id", requestorID,
		"admin_email", requestor.Email,
		"target_user_id", targetUserID,
		"target_user_email", targetUser.Email)

	return targetUserID, nil
}

func (s *accountMetricsService) calculateAccountMetrics(accountID uuid.UUID, transactions []models.Transaction, startDate, endDate time.Time, account *models.Account) *models.AccountMetrics {
	metrics := &models.AccountMetrics{
		AccountID:                accountID,
		StartDate:                startDate,
		EndDate:                  endDate,
		TotalDeposits:            decimal.Zero,
		TotalWithdrawals:         decimal.Zero,
		NetChange:                decimal.Zero,
		TransactionCount:         0,
		DepositCount:             0,
		WithdrawalCount:          0,
		AverageTransactionAmount: decimal.Zero,
		LargestDeposit:           decimal.Zero,
		LargestWithdrawal:        decimal.Zero,
		AverageDailyBalance:      decimal.Zero,
		InterestEarned:           decimal.Zero,
		GeneratedAt:              time.Now(),
	}

	totalAmount := decimal.Zero

	for i := range transactions {
		txn := &transactions[i]

		if txn.Status != models.TransactionStatusCompleted {
			continue
		}

		metrics.TransactionCount++
		totalAmount = totalAmount.Add(txn.Amount)

		if txn.TransactionType == models.TransactionTypeCredit {
			metrics.TotalDeposits = metrics.TotalDeposits.Add(txn.Amount)
			metrics.DepositCount++

			if txn.Amount.GreaterThan(metrics.LargestDeposit) {
				metrics.LargestDeposit = txn.Amount
			}
		} else if txn.TransactionType == models.TransactionTypeDebit {
			metrics.TotalWithdrawals = metrics.TotalWithdrawals.Add(txn.Amount)
			metrics.WithdrawalCount++

			if txn.Amount.GreaterThan(metrics.LargestWithdrawal) {
				metrics.LargestWithdrawal = txn.Amount
			}
		}
	}

	metrics.NetChange = metrics.TotalDeposits.Sub(metrics.TotalWithdrawals)

	if metrics.TransactionCount > 0 {
		metrics.AverageTransactionAmount = totalAmount.Div(decimal.NewFromInt(metrics.TransactionCount))
	}

	daysDifference := int(endDate.Sub(startDate).Hours() / 24)
	if daysDifference == 0 {
		daysDifference = 1
	}

	if len(transactions) > 0 {
		balanceSum := decimal.Zero
		for i := range transactions {
			balanceSum = balanceSum.Add(transactions[i].BalanceAfter)
		}
		metrics.AverageDailyBalance = balanceSum.Div(decimal.NewFromInt(int64(len(transactions))))
	} else {
		metrics.AverageDailyBalance = account.Balance
	}

	if !account.InterestRate.IsZero() && daysDifference > 0 {
		dailyRate := account.InterestRate.Div(decimal.NewFromInt(365)).Div(decimal.NewFromInt(100))
		metrics.InterestEarned = metrics.AverageDailyBalance.Mul(dailyRate).Mul(decimal.NewFromInt(int64(daysDifference)))
	}

	return metrics
}

func (s *accountMetricsService) calculateAggregateMetrics(userID uuid.UUID, accounts []models.Account, startDate, endDate time.Time) *models.UserAggregateMetrics {
	aggregateMetrics := &models.UserAggregateMetrics{
		UserID:                userID,
		StartDate:             startDate,
		EndDate:               endDate,
		TotalDeposits:         decimal.Zero,
		TotalWithdrawals:      decimal.Zero,
		NetChange:             decimal.Zero,
		TotalTransactionCount: 0,
		AccountCount:          len(accounts),
		AccountMetrics:        make([]models.AccountMetrics, 0, len(accounts)),
		GeneratedAt:           time.Now(),
	}

	for i := range accounts {
		account := &accounts[i]

		transactions, err := s.transactionRepo.GetByDateRange(account.ID, startDate, endDate)
		if err != nil {
			slog.Error("failed to fetch transactions for account in aggregate metrics",
				"account_id", account.ID,
				"error", err)
			continue
		}

		accountMetrics := s.calculateAccountMetrics(account.ID, transactions, startDate, endDate, account)

		aggregateMetrics.TotalDeposits = aggregateMetrics.TotalDeposits.Add(accountMetrics.TotalDeposits)
		aggregateMetrics.TotalWithdrawals = aggregateMetrics.TotalWithdrawals.Add(accountMetrics.TotalWithdrawals)
		aggregateMetrics.TotalTransactionCount += accountMetrics.TransactionCount

		aggregateMetrics.AccountMetrics = append(aggregateMetrics.AccountMetrics, *accountMetrics)
	}

	aggregateMetrics.NetChange = aggregateMetrics.TotalDeposits.Sub(aggregateMetrics.TotalWithdrawals)

	return aggregateMetrics
}
