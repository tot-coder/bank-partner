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

const (
	PeriodTypeMonthly   = "monthly"
	PeriodTypeQuarterly = "quarterly"
)

var (
	ErrInvalidPeriodType = errors.New("invalid period type")
	ErrInvalidMonth      = errors.New("month must be between 1 and 12")
	ErrInvalidQuarter    = errors.New("quarter must be between 1 and 4")
	ErrFuturePeriod      = errors.New("cannot generate statement for future period")
)

type statementService struct {
	accountRepo     repositories.AccountRepositoryInterface
	transactionRepo repositories.TransactionRepositoryInterface
	userRepo        repositories.UserRepositoryInterface
	metricsService  AccountMetricsServiceInterface
}

func NewStatementService(
	accountRepo repositories.AccountRepositoryInterface,
	transactionRepo repositories.TransactionRepositoryInterface,
	userRepo repositories.UserRepositoryInterface,
	metricsService AccountMetricsServiceInterface,
) StatementServiceInterface {
	return &statementService{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
		metricsService:  metricsService,
	}
}

func (s *statementService) GenerateStatement(requestorID, accountID uuid.UUID, periodType string, year, period int, isAdmin bool) (*models.AccountStatement, error) {
	if err := s.validatePeriodType(periodType); err != nil {
		return nil, err
	}

	if err := s.validatePeriod(periodType, period); err != nil {
		return nil, err
	}

	startDate, endDate, err := s.calculateDateRange(periodType, year, period)
	if err != nil {
		return nil, err
	}

	if err := s.validateNotFuture(endDate); err != nil {
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

	transactions, err := s.transactionRepo.GetByDateRange(accountID, startDate, endDate)
	if err != nil {
		slog.Error("failed to fetch transactions for statement",
			"account_id", accountID,
			"error", err)
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	openingBalance, closingBalance := s.calculateBalances(transactions, account.Balance)

	statementTransactions := s.buildStatementTransactions(transactions)

	summary := s.calculateSummary(transactions)

	metrics, err := s.metricsService.GetAccountMetrics(requestorID, accountID, &startDate, &endDate, isAdmin)
	if err != nil {
		slog.Warn("failed to fetch metrics for statement",
			"account_id", accountID,
			"error", err)
		metrics = nil
	}

	statement := &models.AccountStatement{
		AccountID:          accountID,
		AccountNumber:      account.AccountNumber,
		AccountType:        account.AccountType,
		PeriodType:         periodType,
		Year:               year,
		Period:             period,
		StartDate:          startDate,
		EndDate:            endDate,
		OpeningBalance:     openingBalance,
		ClosingBalance:     closingBalance,
		Transactions:       statementTransactions,
		PerformanceMetrics: metrics,
		Summary:            summary,
		GeneratedAt:        time.Now(),
	}

	slog.Info("statement generated",
		"account_id", accountID,
		"requestor_id", requestorID,
		"period_type", periodType,
		"year", year,
		"period", period,
		"transaction_count", len(statementTransactions))

	return statement, nil
}

func (s *statementService) validatePeriodType(periodType string) error {
	if periodType != PeriodTypeMonthly && periodType != PeriodTypeQuarterly {
		return ErrInvalidPeriodType
	}
	return nil
}

func (s *statementService) validatePeriod(periodType string, period int) error {
	if periodType == PeriodTypeMonthly {
		if period < 1 || period > 12 {
			return ErrInvalidMonth
		}
	} else if periodType == PeriodTypeQuarterly {
		if period < 1 || period > 4 {
			return ErrInvalidQuarter
		}
	}
	return nil
}

func (s *statementService) calculateDateRange(periodType string, year, period int) (time.Time, time.Time, error) {
	var startDate, endDate time.Time

	if periodType == PeriodTypeMonthly {
		startDate = time.Date(year, time.Month(period), 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(0, 1, 0).Add(-time.Second)
	} else if periodType == PeriodTypeQuarterly {
		startMonth := (period-1)*3 + 1
		startDate = time.Date(year, time.Month(startMonth), 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(0, 3, 0).Add(-time.Second)
	}

	return startDate, endDate, nil
}

func (s *statementService) validateNotFuture(endDate time.Time) error {
	if endDate.After(time.Now()) {
		return ErrFuturePeriod
	}
	return nil
}

func (s *statementService) validateRequestor(requestorID uuid.UUID) (*models.User, error) {
	requestor, err := s.userRepo.GetByID(requestorID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			slog.Warn("user not found during statement request",
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

func (s *statementService) getAndAuthorizeAccount(accountID uuid.UUID, requestor *models.User, isAdmin bool) (*models.Account, error) {
	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		slog.Error("failed to get account for statement",
			"account_id", accountID,
			"error", err)
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if account.UserID != requestor.ID {
		if !isAdmin || requestor.Role != models.RoleAdmin {
			slog.Warn("unauthorized access attempt to statement",
				"requestor_id", requestor.ID,
				"requestor_role", requestor.Role,
				"account_id", accountID,
				"account_user_id", account.UserID)
			return nil, ErrUnauthorized
		}

		slog.Info("admin accessing account statement",
			"admin_id", requestor.ID,
			"admin_email", requestor.Email,
			"account_id", accountID,
			"account_user_id", account.UserID)
	}

	return account, nil
}

func (s *statementService) calculateBalances(transactions []models.Transaction, currentBalance decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	if len(transactions) == 0 {
		return currentBalance, currentBalance
	}

	openingBalance := transactions[0].BalanceBefore
	closingBalance := transactions[len(transactions)-1].BalanceAfter

	return openingBalance, closingBalance
}

func (s *statementService) buildStatementTransactions(transactions []models.Transaction) []models.StatementTransaction {
	statementTxns := make([]models.StatementTransaction, 0, len(transactions))

	for i := range transactions {
		txn := &transactions[i]
		statementTxns = append(statementTxns, models.StatementTransaction{
			ID:              txn.ID,
			Date:            txn.CreatedAt,
			Description:     txn.Description,
			TransactionType: txn.TransactionType,
			Amount:          txn.Amount,
			RunningBalance:  txn.BalanceAfter,
			Reference:       txn.Reference,
			Status:          txn.Status,
		})
	}

	return statementTxns
}

func (s *statementService) calculateSummary(transactions []models.Transaction) models.StatementSummary {
	summary := models.StatementSummary{
		TotalDeposits:    decimal.Zero,
		TotalWithdrawals: decimal.Zero,
		NetChange:        decimal.Zero,
		TransactionCount: 0,
		DepositCount:     0,
		WithdrawalCount:  0,
	}

	for i := range transactions {
		txn := &transactions[i]

		if txn.Status != models.TransactionStatusCompleted {
			continue
		}

		summary.TransactionCount++

		if txn.TransactionType == models.TransactionTypeCredit {
			summary.TotalDeposits = summary.TotalDeposits.Add(txn.Amount)
			summary.DepositCount++
		} else if txn.TransactionType == models.TransactionTypeDebit {
			summary.TotalWithdrawals = summary.TotalWithdrawals.Add(txn.Amount)
			summary.WithdrawalCount++
		}
	}

	summary.NetChange = summary.TotalDeposits.Sub(summary.TotalWithdrawals)

	return summary
}
