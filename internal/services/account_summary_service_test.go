package services

import (
	"testing"
	"time"

	"array-assessment/internal/models"
	"array-assessment/internal/repositories"
	"array-assessment/internal/repositories/repository_mocks"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

// AccountSummaryServiceSuite defines the test suite for AccountSummaryServiceInterface
type AccountSummaryServiceSuite struct {
	suite.Suite
	ctrl        *gomock.Controller
	accountRepo *repository_mocks.MockAccountRepositoryInterface
	userRepo    *repository_mocks.MockUserRepositoryInterface
	service     AccountSummaryServiceInterface
	testUserID  uuid.UUID
	testAdminID uuid.UUID
	testTime    time.Time
}

// SetupTest runs before each test in the suite
func (s *AccountSummaryServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.accountRepo = repository_mocks.NewMockAccountRepositoryInterface(s.ctrl)
	s.userRepo = repository_mocks.NewMockUserRepositoryInterface(s.ctrl)
	s.service = NewAccountSummaryService(s.accountRepo, s.userRepo)

	// Setup common test data
	s.testUserID = uuid.New()
	s.testAdminID = uuid.New()
	s.testTime = time.Now()
}

// TearDownTest runs after each test in the suite
func (s *AccountSummaryServiceSuite) TearDownTest() {
	s.ctrl.Finish()
}

// TestAccountSummaryServiceSuite runs the test suite
func TestAccountSummaryServiceSuite(t *testing.T) {
	suite.Run(t, new(AccountSummaryServiceSuite))
}

// Test GetAccountSummary for regular user accessing own accounts
func (s *AccountSummaryServiceSuite) TestGetAccountSummary_RegularUser_Success() {
	testUser := &models.User{
		ID:        s.testUserID,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleCustomer,
	}

	accounts := []models.Account{
		{
			ID:            uuid.New(),
			AccountNumber: "1012345678",
			UserID:        s.testUserID,
			AccountType:   models.AccountTypeChecking,
			Balance:       decimal.NewFromFloat(1500.50),
			Status:        models.AccountStatusActive,
			Currency:      "USD",
			CreatedAt:     s.testTime,
			UpdatedAt:     s.testTime,
		},
		{
			ID:            uuid.New(),
			AccountNumber: "2023456789",
			UserID:        s.testUserID,
			AccountType:   models.AccountTypeSavings,
			Balance:       decimal.NewFromFloat(5000.00),
			Status:        models.AccountStatusActive,
			Currency:      "USD",
			CreatedAt:     s.testTime,
			UpdatedAt:     s.testTime,
		},
	}

	s.userRepo.EXPECT().GetByID(s.testUserID).Return(testUser, nil)
	s.accountRepo.EXPECT().GetByUserID(s.testUserID).Return(accounts, nil)

	summary, err := s.service.GetAccountSummary(s.testUserID, &s.testUserID, false)
	s.NoError(err)
	s.NotNil(summary)
	s.Equal(s.testUserID, summary.UserID)
	s.Equal(2, summary.AccountCount)
	s.Equal(decimal.NewFromFloat(6500.50), summary.TotalBalance)
	s.Len(summary.Accounts, 2)

	// Verify account number masking
	s.Equal("****5678", summary.Accounts[0].MaskedAccountNumber)
	s.Equal("****6789", summary.Accounts[1].MaskedAccountNumber)

	// Verify full account numbers are not exposed
	s.Empty(summary.Accounts[0].AccountNumber)
	s.Empty(summary.Accounts[1].AccountNumber)
}

// Test GetAccountSummary for admin user accessing another user's accounts
func (s *AccountSummaryServiceSuite) TestGetAccountSummary_AdminUser_Success() {
	adminUser := &models.User{
		ID:        s.testAdminID,
		Email:     "admin@example.com",
		FirstName: "Admin",
		LastName:  "User",
		Role:      models.RoleAdmin,
	}

	targetUser := &models.User{
		ID:        s.testUserID,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleCustomer,
	}

	accounts := []models.Account{
		{
			ID:            uuid.New(),
			AccountNumber: "1012345678",
			UserID:        s.testUserID,
			AccountType:   models.AccountTypeChecking,
			Balance:       decimal.NewFromFloat(2500.00),
			Status:        models.AccountStatusActive,
			Currency:      "USD",
			CreatedAt:     s.testTime,
			UpdatedAt:     s.testTime,
		},
	}

	s.userRepo.EXPECT().GetByID(s.testAdminID).Return(adminUser, nil)
	s.userRepo.EXPECT().GetByID(s.testUserID).Return(targetUser, nil)
	s.accountRepo.EXPECT().GetByUserID(s.testUserID).Return(accounts, nil)

	summary, err := s.service.GetAccountSummary(s.testAdminID, &s.testUserID, true)
	s.NoError(err)
	s.NotNil(summary)
	s.Equal(s.testUserID, summary.UserID)
	s.Equal(1, summary.AccountCount)
	s.True(decimal.NewFromFloat(2500.00).Equal(summary.TotalBalance), "expected total balance to be 2500.00")
}

// Test GetAccountSummary for regular user attempting to access another user's accounts
func (s *AccountSummaryServiceSuite) TestGetAccountSummary_UnauthorizedAccess() {
	testUser := &models.User{
		ID:        s.testUserID,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleCustomer,
	}

	otherUserID := uuid.New()

	s.userRepo.EXPECT().GetByID(s.testUserID).Return(testUser, nil)

	summary, err := s.service.GetAccountSummary(s.testUserID, &otherUserID, false)
	s.Error(err)
	s.Equal(ErrUnauthorized, err)
	s.Nil(summary)
}

// Test GetAccountSummary with no accounts
func (s *AccountSummaryServiceSuite) TestGetAccountSummary_NoAccounts() {
	testUser := &models.User{
		ID:        s.testUserID,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleCustomer,
	}

	s.userRepo.EXPECT().GetByID(s.testUserID).Return(testUser, nil)
	s.accountRepo.EXPECT().GetByUserID(s.testUserID).Return([]models.Account{}, nil)

	summary, err := s.service.GetAccountSummary(s.testUserID, &s.testUserID, false)
	s.NoError(err)
	s.NotNil(summary)
	s.Equal(s.testUserID, summary.UserID)
	s.Equal(0, summary.AccountCount)
	s.Equal(decimal.Zero, summary.TotalBalance)
	s.Len(summary.Accounts, 0)
}

// Test GetAccountSummary with user not found
func (s *AccountSummaryServiceSuite) TestGetAccountSummary_UserNotFound() {
	s.userRepo.EXPECT().GetByID(s.testUserID).Return(nil, repositories.ErrUserNotFound)

	summary, err := s.service.GetAccountSummary(s.testUserID, &s.testUserID, false)
	s.Error(err)
	s.Equal(ErrNotFound, err)
	s.Nil(summary)
}

// Test GetAccountSummary with target user not found (admin access)
func (s *AccountSummaryServiceSuite) TestGetAccountSummary_TargetUserNotFound() {
	adminUser := &models.User{
		ID:        s.testAdminID,
		Email:     "admin@example.com",
		FirstName: "Admin",
		LastName:  "User",
		Role:      models.RoleAdmin,
	}

	s.userRepo.EXPECT().GetByID(s.testAdminID).Return(adminUser, nil)
	s.userRepo.EXPECT().GetByID(s.testUserID).Return(nil, repositories.ErrUserNotFound)

	summary, err := s.service.GetAccountSummary(s.testAdminID, &s.testUserID, true)
	s.Error(err)
	s.Equal(ErrNotFound, err)
	s.Nil(summary)
}

// Test GetAccountSummary with multiple account types
func (s *AccountSummaryServiceSuite) TestGetAccountSummary_MultipleAccountTypes() {
	testUser := &models.User{
		ID:        s.testUserID,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleCustomer,
	}

	accounts := []models.Account{
		{
			ID:            uuid.New(),
			AccountNumber: "1012345678",
			UserID:        s.testUserID,
			AccountType:   models.AccountTypeChecking,
			Balance:       decimal.NewFromFloat(1000.00),
			Status:        models.AccountStatusActive,
			Currency:      "USD",
			CreatedAt:     s.testTime,
			UpdatedAt:     s.testTime,
		},
		{
			ID:            uuid.New(),
			AccountNumber: "2023456789",
			UserID:        s.testUserID,
			AccountType:   models.AccountTypeSavings,
			Balance:       decimal.NewFromFloat(5000.00),
			Status:        models.AccountStatusActive,
			Currency:      "USD",
			CreatedAt:     s.testTime,
			UpdatedAt:     s.testTime,
		},
		{
			ID:            uuid.New(),
			AccountNumber: "3034567890",
			UserID:        s.testUserID,
			AccountType:   models.AccountTypeMoneyMarket,
			Balance:       decimal.NewFromFloat(10000.00),
			Status:        models.AccountStatusActive,
			Currency:      "USD",
			CreatedAt:     s.testTime,
			UpdatedAt:     s.testTime,
		},
	}

	s.userRepo.EXPECT().GetByID(s.testUserID).Return(testUser, nil)
	s.accountRepo.EXPECT().GetByUserID(s.testUserID).Return(accounts, nil)

	summary, err := s.service.GetAccountSummary(s.testUserID, &s.testUserID, false)
	s.NoError(err)
	s.NotNil(summary)
	s.Equal(3, summary.AccountCount)
	s.True(decimal.NewFromFloat(16000.00).Equal(summary.TotalBalance), "expected total balance to be 16000.00")
	s.Len(summary.Accounts, 3)
}

// Test GetAccountSummary with inactive accounts included
func (s *AccountSummaryServiceSuite) TestGetAccountSummary_WithInactiveAccounts() {
	testUser := &models.User{
		ID:        s.testUserID,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleCustomer,
	}

	accounts := []models.Account{
		{
			ID:            uuid.New(),
			AccountNumber: "1012345678",
			UserID:        s.testUserID,
			AccountType:   models.AccountTypeChecking,
			Balance:       decimal.NewFromFloat(1000.00),
			Status:        models.AccountStatusActive,
			Currency:      "USD",
			CreatedAt:     s.testTime,
			UpdatedAt:     s.testTime,
		},
		{
			ID:            uuid.New(),
			AccountNumber: "2023456789",
			UserID:        s.testUserID,
			AccountType:   models.AccountTypeSavings,
			Balance:       decimal.NewFromFloat(500.00),
			Status:        models.AccountStatusInactive,
			Currency:      "USD",
			CreatedAt:     s.testTime,
			UpdatedAt:     s.testTime,
		},
	}

	s.userRepo.EXPECT().GetByID(s.testUserID).Return(testUser, nil)
	s.accountRepo.EXPECT().GetByUserID(s.testUserID).Return(accounts, nil)

	summary, err := s.service.GetAccountSummary(s.testUserID, &s.testUserID, false)
	s.NoError(err)
	s.NotNil(summary)
	s.Equal(2, summary.AccountCount)
	s.True(decimal.NewFromFloat(1500.00).Equal(summary.TotalBalance), "expected total balance to be 1500.00")
	s.Len(summary.Accounts, 2)
}

// Test account number masking edge cases
func (s *AccountSummaryServiceSuite) TestGetAccountSummary_AccountNumberMasking() {
	testUser := &models.User{
		ID:        s.testUserID,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleCustomer,
	}

	accounts := []models.Account{
		{
			ID:            uuid.New(),
			AccountNumber: "1234567890",
			UserID:        s.testUserID,
			AccountType:   models.AccountTypeChecking,
			Balance:       decimal.NewFromFloat(100.00),
			Status:        models.AccountStatusActive,
			Currency:      "USD",
			CreatedAt:     s.testTime,
			UpdatedAt:     s.testTime,
		},
		{
			ID:            uuid.New(),
			AccountNumber: "9876543210",
			UserID:        s.testUserID,
			AccountType:   models.AccountTypeSavings,
			Balance:       decimal.NewFromFloat(200.00),
			Status:        models.AccountStatusActive,
			Currency:      "USD",
			CreatedAt:     s.testTime,
			UpdatedAt:     s.testTime,
		},
	}

	s.userRepo.EXPECT().GetByID(s.testUserID).Return(testUser, nil)
	s.accountRepo.EXPECT().GetByUserID(s.testUserID).Return(accounts, nil)

	summary, err := s.service.GetAccountSummary(s.testUserID, &s.testUserID, false)
	s.NoError(err)
	s.NotNil(summary)

	// Verify correct masking: show only last 4 digits
	s.Equal("****7890", summary.Accounts[0].MaskedAccountNumber)
	s.Equal("****3210", summary.Accounts[1].MaskedAccountNumber)
}
