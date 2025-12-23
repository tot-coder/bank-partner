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

// MockAccountMetricsService is an inline mock for AccountMetricsServiceInterface to avoid import cycles
type MockAccountMetricsService struct {
	GetAccountMetricsFunc       func(requestorID, accountID uuid.UUID, startDate, endDate *time.Time, isAdmin bool) (*models.AccountMetrics, error)
	GetUserAggregateMetricsFunc func(requestorID, targetUserID uuid.UUID, startDate, endDate *time.Time, isAdmin bool) (*models.UserAggregateMetrics, error)
}

func (m *MockAccountMetricsService) GetAccountMetrics(requestorID, accountID uuid.UUID, startDate, endDate *time.Time, isAdmin bool) (*models.AccountMetrics, error) {
	if m.GetAccountMetricsFunc != nil {
		return m.GetAccountMetricsFunc(requestorID, accountID, startDate, endDate, isAdmin)
	}
	return nil, nil
}

func (m *MockAccountMetricsService) GetUserAggregateMetrics(requestorID, targetUserID uuid.UUID, startDate, endDate *time.Time, isAdmin bool) (*models.UserAggregateMetrics, error) {
	if m.GetUserAggregateMetricsFunc != nil {
		return m.GetUserAggregateMetricsFunc(requestorID, targetUserID, startDate, endDate, isAdmin)
	}
	return nil, nil
}

// StatementServiceTestSuite defines the test suite for StatementServiceInterface
type StatementServiceTestSuite struct {
	suite.Suite
	ctrl                *gomock.Controller
	mockAccountRepo     *repository_mocks.MockAccountRepositoryInterface
	mockTransactionRepo *repository_mocks.MockTransactionRepositoryInterface
	mockUserRepo        *repository_mocks.MockUserRepositoryInterface
	mockMetricsService  *MockAccountMetricsService
	service             StatementServiceInterface
}

// SetupTest runs before each test
func (s *StatementServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockAccountRepo = repository_mocks.NewMockAccountRepositoryInterface(s.ctrl)
	s.mockTransactionRepo = repository_mocks.NewMockTransactionRepositoryInterface(s.ctrl)
	s.mockUserRepo = repository_mocks.NewMockUserRepositoryInterface(s.ctrl)
	s.mockMetricsService = &MockAccountMetricsService{}
	s.service = NewStatementService(s.mockAccountRepo, s.mockTransactionRepo, s.mockUserRepo, s.mockMetricsService)
}

// TearDownTest runs after each test
func (s *StatementServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

// TestStatementServiceSuite runs the test suite
func TestStatementServiceSuite(t *testing.T) {
	suite.Run(t, new(StatementServiceTestSuite))
}

// Test successful monthly statement generation
func (s *StatementServiceTestSuite) TestGenerateStatement_Success_Monthly() {
	requestorID := uuid.New()
	accountID := uuid.New()
	year := 2025
	month := 9

	requestor := &models.User{
		ID:    requestorID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	account := &models.Account{
		ID:            accountID,
		UserID:        requestorID,
		AccountNumber: "1234567890",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(5000.00),
		Status:        models.AccountStatusActive,
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)

	transactions := []models.Transaction{
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeCredit,
			Amount:          decimal.NewFromFloat(1000.00),
			BalanceBefore:   decimal.NewFromFloat(4000.00),
			BalanceAfter:    decimal.NewFromFloat(5000.00),
			Description:     "Salary deposit",
			Status:          models.TransactionStatusCompleted,
			Reference:       "TXN-001",
			CreatedAt:       startDate.AddDate(0, 0, 5),
		},
	}

	metrics := &models.AccountMetrics{
		AccountID:        accountID,
		TotalDeposits:    decimal.NewFromFloat(1000.00),
		TotalWithdrawals: decimal.Zero,
		TransactionCount: 1,
	}

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(requestor, nil)
	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(account, nil)
	s.mockTransactionRepo.EXPECT().GetByDateRange(accountID, startDate, endDate).Return(transactions, nil)
	s.mockMetricsService.GetAccountMetricsFunc = func(reqID, accID uuid.UUID, start, end *time.Time, admin bool) (*models.AccountMetrics, error) {
		return metrics, nil
	}

	statement, err := s.service.GenerateStatement(requestorID, accountID, PeriodTypeMonthly, year, month, false)

	s.NoError(err)
	s.NotNil(statement)
	s.Equal(accountID, statement.AccountID)
	s.Equal("1234567890", statement.AccountNumber)
	s.Equal(PeriodTypeMonthly, statement.PeriodType)
	s.Equal(year, statement.Year)
	s.Equal(month, statement.Period)
	s.Equal(1, len(statement.Transactions))
	s.NotNil(statement.PerformanceMetrics)
	s.True(statement.Summary.TotalDeposits.Equal(decimal.NewFromFloat(1000.00)))
}

// Test successful quarterly statement generation
func (s *StatementServiceTestSuite) TestGenerateStatement_Success_Quarterly() {
	requestorID := uuid.New()
	accountID := uuid.New()
	year := 2025
	quarter := 3

	requestor := &models.User{
		ID:    requestorID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	account := &models.Account{
		ID:            accountID,
		UserID:        requestorID,
		AccountNumber: "9876543210",
		Balance:       decimal.NewFromFloat(10000.00),
	}

	transactions := []models.Transaction{}
	metrics := &models.AccountMetrics{
		AccountID:        accountID,
		TotalDeposits:    decimal.Zero,
		TotalWithdrawals: decimal.Zero,
		TransactionCount: 0,
	}

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(requestor, nil)
	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(account, nil)
	s.mockTransactionRepo.EXPECT().GetByDateRange(accountID, gomock.Any(), gomock.Any()).Return(transactions, nil)
	s.mockMetricsService.GetAccountMetricsFunc = func(reqID, accID uuid.UUID, start, end *time.Time, admin bool) (*models.AccountMetrics, error) {
		return metrics, nil
	}

	statement, err := s.service.GenerateStatement(requestorID, accountID, PeriodTypeQuarterly, year, quarter, false)

	s.NoError(err)
	s.NotNil(statement)
	s.Equal(PeriodTypeQuarterly, statement.PeriodType)
	s.Equal(quarter, statement.Period)
	s.Equal(0, len(statement.Transactions))
}

// Test invalid month (out of range)
func (s *StatementServiceTestSuite) TestGenerateStatement_Error_InvalidMonth() {
	requestorID := uuid.New()
	accountID := uuid.New()

	statement, err := s.service.GenerateStatement(requestorID, accountID, PeriodTypeMonthly, 2025, 13, false)

	s.Error(err)
	s.Nil(statement)
	s.Contains(err.Error(), "month must be between 1 and 12")
}

// Test invalid quarter (out of range)
func (s *StatementServiceTestSuite) TestGenerateStatement_Error_InvalidQuarter() {
	requestorID := uuid.New()
	accountID := uuid.New()

	statement, err := s.service.GenerateStatement(requestorID, accountID, PeriodTypeQuarterly, 2025, 5, false)

	s.Error(err)
	s.Nil(statement)
	s.Contains(err.Error(), "quarter must be between 1 and 4")
}

// Test invalid period type
func (s *StatementServiceTestSuite) TestGenerateStatement_Error_InvalidPeriodType() {
	requestorID := uuid.New()
	accountID := uuid.New()

	statement, err := s.service.GenerateStatement(requestorID, accountID, "invalid", 2025, 1, false)

	s.Error(err)
	s.Nil(statement)
	s.Contains(err.Error(), "invalid period type")
}

// Test future period (should fail)
func (s *StatementServiceTestSuite) TestGenerateStatement_Error_FuturePeriod() {
	requestorID := uuid.New()
	accountID := uuid.New()
	futureYear := time.Now().Year() + 2

	statement, err := s.service.GenerateStatement(requestorID, accountID, PeriodTypeMonthly, futureYear, 1, false)

	s.Error(err)
	s.Nil(statement)
	s.Contains(err.Error(), "cannot generate statement for future period")
}

// Test unauthorized access
func (s *StatementServiceTestSuite) TestGenerateStatement_Error_Unauthorized() {
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

	statement, err := s.service.GenerateStatement(requestorID, accountID, PeriodTypeMonthly, 2025, 9, false)

	s.Error(err)
	s.Nil(statement)
	s.ErrorIs(err, ErrUnauthorized)
}

// Test admin access to any account
func (s *StatementServiceTestSuite) TestGenerateStatement_Success_AdminAccess() {
	adminID := uuid.New()
	accountID := uuid.New()
	otherUserID := uuid.New()

	admin := &models.User{
		ID:    adminID,
		Email: gofakeit.Email(),
		Role:  models.RoleAdmin,
	}

	account := &models.Account{
		ID:            accountID,
		UserID:        otherUserID,
		AccountNumber: "1111222233",
		Balance:       decimal.NewFromFloat(5000.00),
	}

	transactions := []models.Transaction{}
	metrics := &models.AccountMetrics{
		AccountID:        accountID,
		TotalDeposits:    decimal.Zero,
		TotalWithdrawals: decimal.Zero,
		TransactionCount: 0,
	}

	s.mockUserRepo.EXPECT().GetByID(adminID).Return(admin, nil)
	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(account, nil)
	s.mockTransactionRepo.EXPECT().GetByDateRange(accountID, gomock.Any(), gomock.Any()).Return(transactions, nil)
	s.mockMetricsService.GetAccountMetricsFunc = func(reqID, accID uuid.UUID, start, end *time.Time, admin bool) (*models.AccountMetrics, error) {
		return metrics, nil
	}

	statement, err := s.service.GenerateStatement(adminID, accountID, PeriodTypeMonthly, 2025, 9, true)

	s.NoError(err)
	s.NotNil(statement)
}

// Test account not found
func (s *StatementServiceTestSuite) TestGenerateStatement_Error_AccountNotFound() {
	requestorID := uuid.New()
	accountID := uuid.New()

	requestor := &models.User{
		ID:    requestorID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(requestor, nil)
	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(nil, errors.New("account not found"))

	statement, err := s.service.GenerateStatement(requestorID, accountID, PeriodTypeMonthly, 2025, 9, false)

	s.Error(err)
	s.Nil(statement)
}

// Test running balance calculation
func (s *StatementServiceTestSuite) TestGenerateStatement_Success_RunningBalanceCalculation() {
	requestorID := uuid.New()
	accountID := uuid.New()
	year := 2025
	month := 9

	requestor := &models.User{
		ID:    requestorID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	account := &models.Account{
		ID:            accountID,
		UserID:        requestorID,
		AccountNumber: "1234567890",
		Balance:       decimal.NewFromFloat(6500.00),
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)

	transactions := []models.Transaction{
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeCredit,
			Amount:          decimal.NewFromFloat(1000.00),
			BalanceBefore:   decimal.NewFromFloat(5000.00),
			BalanceAfter:    decimal.NewFromFloat(6000.00),
			Description:     "First deposit",
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       startDate.AddDate(0, 0, 1),
		},
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeDebit,
			Amount:          decimal.NewFromFloat(500.00),
			BalanceBefore:   decimal.NewFromFloat(6000.00),
			BalanceAfter:    decimal.NewFromFloat(5500.00),
			Description:     "Withdrawal",
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       startDate.AddDate(0, 0, 2),
		},
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeCredit,
			Amount:          decimal.NewFromFloat(1000.00),
			BalanceBefore:   decimal.NewFromFloat(5500.00),
			BalanceAfter:    decimal.NewFromFloat(6500.00),
			Description:     "Second deposit",
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       startDate.AddDate(0, 0, 3),
		},
	}

	metrics := &models.AccountMetrics{
		AccountID:        accountID,
		TotalDeposits:    decimal.NewFromFloat(2000.00),
		TotalWithdrawals: decimal.NewFromFloat(500.00),
		TransactionCount: 3,
	}

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(requestor, nil)
	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(account, nil)
	s.mockTransactionRepo.EXPECT().GetByDateRange(accountID, startDate, endDate).Return(transactions, nil)
	s.mockMetricsService.GetAccountMetricsFunc = func(reqID, accID uuid.UUID, start, end *time.Time, admin bool) (*models.AccountMetrics, error) {
		return metrics, nil
	}

	statement, err := s.service.GenerateStatement(requestorID, accountID, PeriodTypeMonthly, year, month, false)

	s.NoError(err)
	s.NotNil(statement)
	s.Equal(3, len(statement.Transactions))
	s.True(statement.OpeningBalance.Equal(decimal.NewFromFloat(5000.00)))
	s.True(statement.ClosingBalance.Equal(decimal.NewFromFloat(6500.00)))

	s.True(statement.Transactions[0].RunningBalance.Equal(decimal.NewFromFloat(6000.00)))
	s.True(statement.Transactions[1].RunningBalance.Equal(decimal.NewFromFloat(5500.00)))
	s.True(statement.Transactions[2].RunningBalance.Equal(decimal.NewFromFloat(6500.00)))

	s.Equal(2, statement.Summary.DepositCount)
	s.Equal(1, statement.Summary.WithdrawalCount)
	s.True(statement.Summary.NetChange.Equal(decimal.NewFromFloat(1500.00)))
}

// Test statement with only pending transactions (should be excluded from summary)
func (s *StatementServiceTestSuite) TestGenerateStatement_Success_OnlyCompletedInSummary() {
	requestorID := uuid.New()
	accountID := uuid.New()
	year := 2025
	month := 9

	requestor := &models.User{
		ID:    requestorID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	account := &models.Account{
		ID:            accountID,
		UserID:        requestorID,
		AccountNumber: "1234567890",
		Balance:       decimal.NewFromFloat(5500.00),
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)

	transactions := []models.Transaction{
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeCredit,
			Amount:          decimal.NewFromFloat(1000.00),
			BalanceBefore:   decimal.NewFromFloat(5000.00),
			BalanceAfter:    decimal.NewFromFloat(6000.00),
			Description:     "Completed deposit",
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       startDate.AddDate(0, 0, 1),
		},
		{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: models.TransactionTypeDebit,
			Amount:          decimal.NewFromFloat(500.00),
			BalanceBefore:   decimal.NewFromFloat(6000.00),
			BalanceAfter:    decimal.NewFromFloat(5500.00),
			Description:     "Pending withdrawal",
			Status:          models.TransactionStatusPending,
			CreatedAt:       startDate.AddDate(0, 0, 2),
		},
	}

	metrics := &models.AccountMetrics{
		AccountID:        accountID,
		TotalDeposits:    decimal.NewFromFloat(1000.00),
		TotalWithdrawals: decimal.Zero,
		TransactionCount: 1,
	}

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(requestor, nil)
	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(account, nil)
	s.mockTransactionRepo.EXPECT().GetByDateRange(accountID, startDate, endDate).Return(transactions, nil)
	s.mockMetricsService.GetAccountMetricsFunc = func(reqID, accID uuid.UUID, start, end *time.Time, admin bool) (*models.AccountMetrics, error) {
		return metrics, nil
	}

	statement, err := s.service.GenerateStatement(requestorID, accountID, PeriodTypeMonthly, year, month, false)

	s.NoError(err)
	s.NotNil(statement)
	s.Equal(2, len(statement.Transactions))

	s.Equal(1, statement.Summary.DepositCount)
	s.Equal(0, statement.Summary.WithdrawalCount)
	s.True(statement.Summary.TotalDeposits.Equal(decimal.NewFromFloat(1000.00)))
	s.True(statement.Summary.TotalWithdrawals.Equal(decimal.Zero))
}

// Test requestor not found
func (s *StatementServiceTestSuite) TestGenerateStatement_Error_RequestorNotFound() {
	requestorID := uuid.New()
	accountID := uuid.New()

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(nil, repositories.ErrUserNotFound)

	statement, err := s.service.GenerateStatement(requestorID, accountID, PeriodTypeMonthly, 2025, 9, false)

	s.Error(err)
	s.Nil(statement)
	s.ErrorIs(err, ErrNotFound)
}

// Test quarterly Q1 dates
func (s *StatementServiceTestSuite) TestGenerateStatement_Success_Q1Dates() {
	requestorID := uuid.New()
	accountID := uuid.New()
	year := 2025
	quarter := 1

	requestor := &models.User{
		ID:    requestorID,
		Email: gofakeit.Email(),
		Role:  models.RoleCustomer,
	}

	account := &models.Account{
		ID:            accountID,
		UserID:        requestorID,
		AccountNumber: "1234567890",
		Balance:       decimal.NewFromFloat(5000.00),
	}

	transactions := []models.Transaction{}
	metrics := &models.AccountMetrics{AccountID: accountID}

	s.mockUserRepo.EXPECT().GetByID(requestorID).Return(requestor, nil)
	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(account, nil)
	s.mockTransactionRepo.EXPECT().GetByDateRange(accountID, gomock.Any(), gomock.Any()).Return(transactions, nil)
	s.mockMetricsService.GetAccountMetricsFunc = func(reqID, accID uuid.UUID, start, end *time.Time, admin bool) (*models.AccountMetrics, error) {
		return metrics, nil
	}

	statement, err := s.service.GenerateStatement(requestorID, accountID, PeriodTypeQuarterly, year, quarter, false)

	s.NoError(err)
	s.NotNil(statement)

	expectedStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(year, time.April, 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)

	s.True(statement.StartDate.Equal(expectedStart))
	s.True(statement.EndDate.Equal(expectedEnd))
}
