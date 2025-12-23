package services

import (
	"errors"
	"testing"
	"time"

	"array-assessment/internal/models"
	"array-assessment/internal/repositories"
	"array-assessment/internal/repositories/repository_mocks"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

// MetricsServiceTestSuite defines the test suite for MetricsService
type MetricsServiceTestSuite struct {
	suite.Suite
	ctrl                *gomock.Controller
	mockAccountRepo     *repository_mocks.MockAccountRepositoryInterface
	mockTransactionRepo *repository_mocks.MockTransactionRepositoryInterface
	mockUserRepo        *repository_mocks.MockUserRepositoryInterface
	service             AccountMetricsServiceInterface
}

// SetupTest runs before each test
func (s *MetricsServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockAccountRepo = repository_mocks.NewMockAccountRepositoryInterface(s.ctrl)
	s.mockTransactionRepo = repository_mocks.NewMockTransactionRepositoryInterface(s.ctrl)
	s.mockUserRepo = repository_mocks.NewMockUserRepositoryInterface(s.ctrl)
	s.service = NewAccountMetricsService(s.mockAccountRepo, s.mockTransactionRepo, s.mockUserRepo)
}

// TearDownTest runs after each test
func (s *MetricsServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

// TestMetricsServiceSuite runs the test suite
func TestMetricsServiceSuite(t *testing.T) {
	suite.Run(t, new(MetricsServiceTestSuite))
}

// Test successful metrics retrieval with default 14-day period
func (s *MetricsServiceTestSuite) TestGetAccountMetrics_Success_DefaultPeriod() {
	requestorID := uuid.New()
	accountID := uuid.New()

	requestor := &models.User{
		ID:    requestorID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	account := &models.Account{
		ID:          accountID,
		UserID:      requestorID,
		Balance:     decimal.NewFromFloat(5000.00),
		AccountType: models.AccountTypeChecking,
		Status:      models.AccountStatusActive,
	}

	now := time.Now()
	startDate := now.AddDate(0, 0, -14)

	transactions := []models.Transaction{
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeCredit,
			Amount:          decimal.NewFromFloat(1000.00),
			BalanceBefore:   decimal.NewFromFloat(3000.00),
			BalanceAfter:    decimal.NewFromFloat(4000.00),
			Description:     "Salary deposit",
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       now.AddDate(0, 0, -7),
		},
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeDebit,
			Amount:          decimal.NewFromFloat(500.00),
			BalanceBefore:   decimal.NewFromFloat(4000.00),
			BalanceAfter:    decimal.NewFromFloat(3500.00),
			Description:     "Grocery shopping",
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       now.AddDate(0, 0, -5),
		},
	}

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(requestor, nil)
	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(account, nil)
	s.mockTransactionRepo.EXPECT().GetByDateRange(accountID, gomock.Any(), gomock.Any()).Return(transactions, nil)

	metrics, err := s.service.GetAccountMetrics(requestorID, accountID, nil, nil, false)

	s.NoError(err)
	s.NotNil(metrics)
	s.Equal(accountID, metrics.AccountID)
	s.True(metrics.TotalDeposits.Equal(decimal.NewFromFloat(1000.00)))
	s.True(metrics.TotalWithdrawals.Equal(decimal.NewFromFloat(500.00)))
	s.True(metrics.NetChange.Equal(decimal.NewFromFloat(500.00)))
	s.Equal(int64(2), metrics.TransactionCount)
	s.Equal(int64(1), metrics.DepositCount)
	s.Equal(int64(1), metrics.WithdrawalCount)
	s.True(metrics.StartDate.Before(now) || metrics.StartDate.Equal(startDate))
	s.True(metrics.EndDate.After(startDate))
}

// Test metrics with custom date range
func (s *MetricsServiceTestSuite) TestGetAccountMetrics_Success_CustomDateRange() {
	requestorID := uuid.New()
	accountID := uuid.New()
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	requestor := &models.User{
		ID:    requestorID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	account := &models.Account{
		ID:      accountID,
		UserID:  requestorID,
		Balance: decimal.NewFromFloat(10000.00),
	}

	transactions := []models.Transaction{}

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(requestor, nil)
	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(account, nil)
	s.mockTransactionRepo.EXPECT().GetByDateRange(accountID, startDate, endDate).Return(transactions, nil)

	metrics, err := s.service.GetAccountMetrics(requestorID, accountID, &startDate, &endDate, false)

	s.NoError(err)
	s.NotNil(metrics)
	s.Equal(accountID, metrics.AccountID)
	s.True(metrics.TotalDeposits.Equal(decimal.Zero))
	s.True(metrics.TotalWithdrawals.Equal(decimal.Zero))
	s.Equal(int64(0), metrics.TransactionCount)
	s.Equal(startDate, metrics.StartDate)
	s.Equal(endDate, metrics.EndDate)
}

// Test metrics with invalid date range (startDate after endDate)
func (s *MetricsServiceTestSuite) TestGetAccountMetrics_Error_InvalidDateRange() {
	requestorID := uuid.New()
	accountID := uuid.New()
	startDate := time.Now()
	endDate := time.Now().AddDate(0, 0, -7)

	metrics, err := s.service.GetAccountMetrics(requestorID, accountID, &startDate, &endDate, false)

	s.Error(err)
	s.Nil(metrics)
	s.Contains(err.Error(), "start date must be before end date")
}

// Test metrics with future date range
func (s *MetricsServiceTestSuite) TestGetAccountMetrics_Error_FutureEndDate() {
	requestorID := uuid.New()
	accountID := uuid.New()
	startDate := time.Now().AddDate(0, 0, -7)
	endDate := time.Now().AddDate(0, 0, 7)

	metrics, err := s.service.GetAccountMetrics(requestorID, accountID, &startDate, &endDate, false)

	s.Error(err)
	s.Nil(metrics)
	s.Contains(err.Error(), "end date cannot be in the future")
}

// Test unauthorized access (user trying to access another user's account)
func (s *MetricsServiceTestSuite) TestGetAccountMetrics_Error_Unauthorized() {
	requestorID := uuid.New()
	accountID := uuid.New()
	otherUserID := uuid.New()

	requestor := &models.User{
		ID:    requestorID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	account := &models.Account{
		ID:      accountID,
		UserID:  otherUserID,
		Balance: decimal.NewFromFloat(5000.00),
	}

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(requestor, nil)
	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(account, nil)

	metrics, err := s.service.GetAccountMetrics(requestorID, accountID, nil, nil, false)

	s.Error(err)
	s.Nil(metrics)
	s.ErrorIs(err, ErrUnauthorized)
}

// Test admin access to any account
func (s *MetricsServiceTestSuite) TestGetAccountMetrics_Success_AdminAccess() {
	adminID := uuid.New()
	accountID := uuid.New()
	otherUserID := uuid.New()

	admin := &models.User{
		ID:    adminID,
		Email: gofakeit.Email(),
		Role:  models.RoleAdmin,
	}

	account := &models.Account{
		ID:      accountID,
		UserID:  otherUserID,
		Balance: decimal.NewFromFloat(5000.00),
	}

	transactions := []models.Transaction{}

	s.mockUserRepo.EXPECT().GetByID(adminID).Return(admin, nil)
	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(account, nil)
	s.mockTransactionRepo.EXPECT().GetByDateRange(accountID, gomock.Any(), gomock.Any()).Return(transactions, nil)

	metrics, err := s.service.GetAccountMetrics(adminID, accountID, nil, nil, true)

	s.NoError(err)
	s.NotNil(metrics)
	s.Equal(accountID, metrics.AccountID)
}

// Test account not found
func (s *MetricsServiceTestSuite) TestGetAccountMetrics_Error_AccountNotFound() {
	requestorID := uuid.New()
	accountID := uuid.New()

	requestor := &models.User{
		ID:    requestorID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(requestor, nil)
	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(nil, errors.New("account not found"))

	metrics, err := s.service.GetAccountMetrics(requestorID, accountID, nil, nil, false)

	s.Error(err)
	s.Nil(metrics)
}

// Test requestor not found
func (s *MetricsServiceTestSuite) TestGetAccountMetrics_Error_RequestorNotFound() {
	requestorID := uuid.New()
	accountID := uuid.New()

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(nil, repositories.ErrUserNotFound)

	metrics, err := s.service.GetAccountMetrics(requestorID, accountID, nil, nil, false)

	s.Error(err)
	s.Nil(metrics)
	s.ErrorIs(err, ErrNotFound)
}

// Test advanced metrics calculations (largest deposit/withdrawal, average transaction amount)
func (s *MetricsServiceTestSuite) TestGetAccountMetrics_Success_AdvancedMetrics() {
	requestorID := uuid.New()
	accountID := uuid.New()

	requestor := &models.User{
		ID:    requestorID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	account := &models.Account{
		ID:           accountID,
		UserID:       requestorID,
		Balance:      decimal.NewFromFloat(10000.00),
		InterestRate: decimal.NewFromFloat(2.5),
	}

	now := time.Now()

	transactions := []models.Transaction{
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeCredit,
			Amount:          decimal.NewFromFloat(5000.00),
			Description:     "Large deposit",
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       now.AddDate(0, 0, -10),
		},
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeCredit,
			Amount:          decimal.NewFromFloat(1000.00),
			Description:     "Regular deposit",
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       now.AddDate(0, 0, -8),
		},
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeDebit,
			Amount:          decimal.NewFromFloat(2000.00),
			Description:     "Large withdrawal",
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       now.AddDate(0, 0, -5),
		},
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeDebit,
			Amount:          decimal.NewFromFloat(500.00),
			Description:     "Small withdrawal",
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       now.AddDate(0, 0, -2),
		},
	}

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(requestor, nil)
	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(account, nil)
	s.mockTransactionRepo.EXPECT().GetByDateRange(accountID, gomock.Any(), gomock.Any()).Return(transactions, nil)

	metrics, err := s.service.GetAccountMetrics(requestorID, accountID, nil, nil, false)

	s.NoError(err)
	s.NotNil(metrics)
	s.True(metrics.LargestDeposit.Equal(decimal.NewFromFloat(5000.00)))
	s.True(metrics.LargestWithdrawal.Equal(decimal.NewFromFloat(2000.00)))
	s.True(metrics.AverageTransactionAmount.Equal(decimal.NewFromFloat(2125.00)))
	s.Equal(int64(4), metrics.TransactionCount)
}

// Test metrics with only pending transactions (should be excluded)
func (s *MetricsServiceTestSuite) TestGetAccountMetrics_Success_OnlyCompletedTransactions() {
	requestorID := uuid.New()
	accountID := uuid.New()

	requestor := &models.User{
		ID:    requestorID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	account := &models.Account{
		ID:      accountID,
		UserID:  requestorID,
		Balance: decimal.NewFromFloat(5000.00),
	}

	now := time.Now()

	transactions := []models.Transaction{
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeCredit,
			Amount:          decimal.NewFromFloat(1000.00),
			Description:     "Completed deposit",
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       now.AddDate(0, 0, -7),
		},
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeDebit,
			Amount:          decimal.NewFromFloat(500.00),
			Description:     "Pending withdrawal",
			Status:          models.TransactionStatusPending,
			CreatedAt:       now.AddDate(0, 0, -5),
		},
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeDebit,
			Amount:          decimal.NewFromFloat(300.00),
			Description:     "Failed withdrawal",
			Status:          models.TransactionStatusFailed,
			CreatedAt:       now.AddDate(0, 0, -3),
		},
	}

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(requestor, nil)
	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(account, nil)
	s.mockTransactionRepo.EXPECT().GetByDateRange(accountID, gomock.Any(), gomock.Any()).Return(transactions, nil)

	metrics, err := s.service.GetAccountMetrics(requestorID, accountID, nil, nil, false)

	s.NoError(err)
	s.NotNil(metrics)
	s.Equal(int64(1), metrics.TransactionCount)
	s.True(metrics.TotalDeposits.Equal(decimal.NewFromFloat(1000.00)))
	s.True(metrics.TotalWithdrawals.Equal(decimal.Zero))
}

// Test aggregate metrics across multiple accounts
func (s *MetricsServiceTestSuite) TestGetUserAggregateMetrics_Success() {
	requestorID := uuid.New()
	account1ID := uuid.New()
	account2ID := uuid.New()

	requestor := &models.User{
		ID:    requestorID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	accounts := []models.Account{
		{
			ID:      account1ID,
			UserID:  requestorID,
			Balance: decimal.NewFromFloat(5000.00),
		},
		{
			ID:      account2ID,
			UserID:  requestorID,
			Balance: decimal.NewFromFloat(3000.00),
		},
	}

	now := time.Now()

	transactions1 := []models.Transaction{
		{
			ID:              uuid.New(),
			AccountID:       account1ID,
			TransactionType: models.TransactionTypeCredit,
			Amount:          decimal.NewFromFloat(1000.00),
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       now.AddDate(0, 0, -7),
		},
	}

	transactions2 := []models.Transaction{
		{
			ID:              uuid.New(),
			AccountID:       account2ID,
			TransactionType: models.TransactionTypeDebit,
			Amount:          decimal.NewFromFloat(500.00),
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       now.AddDate(0, 0, -5),
		},
	}

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(requestor, nil)
	s.mockAccountRepo.EXPECT().GetByUserID(requestorID).Return(accounts, nil)
	s.mockTransactionRepo.EXPECT().GetByDateRange(account1ID, gomock.Any(), gomock.Any()).Return(transactions1, nil)
	s.mockTransactionRepo.EXPECT().GetByDateRange(account2ID, gomock.Any(), gomock.Any()).Return(transactions2, nil)

	aggregateMetrics, err := s.service.GetUserAggregateMetrics(requestorID, requestorID, nil, nil, false)

	s.NoError(err)
	s.NotNil(aggregateMetrics)
	s.Equal(requestorID, aggregateMetrics.UserID)
	s.True(aggregateMetrics.TotalDeposits.Equal(decimal.NewFromFloat(1000.00)))
	s.True(aggregateMetrics.TotalWithdrawals.Equal(decimal.NewFromFloat(500.00)))
	s.Equal(int64(2), aggregateMetrics.TotalTransactionCount)
	s.Equal(2, aggregateMetrics.AccountCount)
	s.Equal(2, len(aggregateMetrics.AccountMetrics))
}

// Test aggregate metrics with no accounts
func (s *MetricsServiceTestSuite) TestGetUserAggregateMetrics_Success_NoAccounts() {
	requestorID := uuid.New()

	requestor := &models.User{
		ID:    requestorID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	accounts := []models.Account{}

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(requestor, nil)
	s.mockAccountRepo.EXPECT().GetByUserID(requestorID).Return(accounts, nil)

	aggregateMetrics, err := s.service.GetUserAggregateMetrics(requestorID, requestorID, nil, nil, false)

	s.NoError(err)
	s.NotNil(aggregateMetrics)
	s.Equal(requestorID, aggregateMetrics.UserID)
	s.True(aggregateMetrics.TotalDeposits.Equal(decimal.Zero))
	s.True(aggregateMetrics.TotalWithdrawals.Equal(decimal.Zero))
	s.Equal(int64(0), aggregateMetrics.TotalTransactionCount)
	s.Equal(0, aggregateMetrics.AccountCount)
	s.Equal(0, len(aggregateMetrics.AccountMetrics))
}

// Test aggregate metrics unauthorized access
func (s *MetricsServiceTestSuite) TestGetUserAggregateMetrics_Error_Unauthorized() {
	requestorID := uuid.New()
	targetUserID := uuid.New()

	requestor := &models.User{
		ID:    requestorID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(requestor, nil)

	aggregateMetrics, err := s.service.GetUserAggregateMetrics(requestorID, targetUserID, nil, nil, false)

	s.Error(err)
	s.Nil(aggregateMetrics)
	s.ErrorIs(err, ErrUnauthorized)
}

// Test aggregate metrics admin access
func (s *MetricsServiceTestSuite) TestGetUserAggregateMetrics_Success_AdminAccess() {
	adminID := uuid.New()
	targetUserID := uuid.New()

	admin := &models.User{
		ID:    adminID,
		Email: gofakeit.Email(),
		Role:  models.RoleAdmin,
	}

	targetUser := &models.User{
		ID:    targetUserID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	accounts := []models.Account{}

	s.mockUserRepo.EXPECT().GetByID(adminID).Return(admin, nil)
	s.mockUserRepo.EXPECT().GetByID(targetUserID).Return(targetUser, nil)
	s.mockAccountRepo.EXPECT().GetByUserID(targetUserID).Return(accounts, nil)

	aggregateMetrics, err := s.service.GetUserAggregateMetrics(adminID, targetUserID, nil, nil, true)

	s.NoError(err)
	s.NotNil(aggregateMetrics)
	s.Equal(targetUserID, aggregateMetrics.UserID)
}

// Test aggregate metrics with invalid date range
func (s *MetricsServiceTestSuite) TestGetUserAggregateMetrics_Error_InvalidDateRange() {
	requestorID := uuid.New()
	startDate := time.Now()
	endDate := time.Now().AddDate(0, 0, -7)

	aggregateMetrics, err := s.service.GetUserAggregateMetrics(requestorID, requestorID, &startDate, &endDate, false)

	s.Error(err)
	s.Nil(aggregateMetrics)
	s.Contains(err.Error(), "start date must be before end date")
}

// Test edge case: very large transactions
func (s *MetricsServiceTestSuite) TestGetAccountMetrics_Success_LargeTransactions() {
	requestorID := uuid.New()
	accountID := uuid.New()

	requestor := &models.User{
		ID:    requestorID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	account := &models.Account{
		ID:      accountID,
		UserID:  requestorID,
		Balance: decimal.NewFromFloat(10000000.00),
	}

	now := time.Now()

	transactions := []models.Transaction{
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeCredit,
			Amount:          decimal.NewFromFloat(999999.99),
			Description:     "Very large deposit",
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       now.AddDate(0, 0, -7),
		},
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeDebit,
			Amount:          decimal.NewFromFloat(500000.50),
			Description:     "Large withdrawal",
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       now.AddDate(0, 0, -5),
		},
	}

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(requestor, nil)
	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(account, nil)
	s.mockTransactionRepo.EXPECT().GetByDateRange(accountID, gomock.Any(), gomock.Any()).Return(transactions, nil)

	metrics, err := s.service.GetAccountMetrics(requestorID, accountID, nil, nil, false)

	s.NoError(err)
	s.NotNil(metrics)
	s.True(metrics.TotalDeposits.Equal(decimal.NewFromFloat(999999.99)))
	s.True(metrics.TotalWithdrawals.Equal(decimal.NewFromFloat(500000.50)))
	s.True(metrics.NetChange.Equal(decimal.NewFromFloat(499999.49)))
}
