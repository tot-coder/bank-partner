package services

import (
	"log/slog"
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

// AccountServiceSuite defines the test suite for AccountServiceInterface
type AccountServiceSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	accountRepo     *repository_mocks.MockAccountRepositoryInterface
	transactionRepo *repository_mocks.MockTransactionRepositoryInterface
	transferRepo    *repository_mocks.MockTransferRepositoryInterface
	userRepo        *repository_mocks.MockUserRepositoryInterface
	auditRepo       *repository_mocks.MockAuditLogRepositoryInterface
	service         *accountService
	testUser        *models.User
	testUserID      uuid.UUID
	testAccountID   uuid.UUID
	testTime        time.Time
}

// SetupTest runs before each test in the suite
func (s *AccountServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.accountRepo = repository_mocks.NewMockAccountRepositoryInterface(s.ctrl)
	s.transferRepo = repository_mocks.NewMockTransferRepositoryInterface(s.ctrl)
	s.transactionRepo = repository_mocks.NewMockTransactionRepositoryInterface(s.ctrl)
	s.userRepo = repository_mocks.NewMockUserRepositoryInterface(s.ctrl)
	s.auditRepo = repository_mocks.NewMockAuditLogRepositoryInterface(s.ctrl)
	s.service = NewAccountService(s.accountRepo,
		s.transactionRepo,
		s.transferRepo,
		s.userRepo,
		s.auditRepo,
		slog.Default()).(*accountService)

	// Setup common test data
	s.testUserID = uuid.New()
	s.testAccountID = uuid.New()
	s.testTime = time.Now()
	s.testUser = &models.User{
		ID:        s.testUserID,
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      "user",
	}
}

// TearDownTest runs after each test in the suite
func (s *AccountServiceSuite) TearDownTest() {
	s.ctrl.Finish()
}

// TestAccountServiceSuite runs the test suite
func TestAccountServiceSuite(t *testing.T) {
	suite.Run(t, new(AccountServiceSuite))
}

// Test CreateAccount functionality
func (s *AccountServiceSuite) TestCreateAccount_WithInitialDeposit() {
	// Setup expectations
	s.userRepo.EXPECT().GetByID(s.testUserID).Return(s.testUser, nil)
	s.accountRepo.EXPECT().ExistsForUser(s.testUserID, "checking").Return(false, nil)
	s.accountRepo.EXPECT().GenerateUniqueAccountNumber("checking").Return("1012345678", nil)
	s.accountRepo.EXPECT().CreateWithTransaction(gomock.Any(), gomock.Any()).DoAndReturn(
		func(account *models.Account, transactions []models.Transaction) error {
			account.ID = s.testAccountID
			account.CreatedAt = s.testTime
			account.UpdatedAt = s.testTime
			for i := range transactions {
				transactions[i].ID = uuid.New()
				transactions[i].AccountID = s.testAccountID
				transactions[i].CreatedAt = s.testTime
				transactions[i].UpdatedAt = s.testTime
			}
			return nil
		})
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil)

	account, err := s.service.CreateAccount(s.testUserID, "checking", decimal.NewFromFloat(100))
	s.NoError(err)
	s.NotNil(account)
	s.Equal(s.testUserID, account.UserID)
	s.Equal("checking", account.AccountType)
	s.Equal("1012345678", account.AccountNumber)
	s.Equal(decimal.NewFromFloat(100), account.Balance)
	s.Equal("active", account.Status)
}

func (s *AccountServiceSuite) TestCreateAccount_WithoutInitialDeposit() {
	s.userRepo.EXPECT().GetByID(s.testUserID).Return(s.testUser, nil)
	s.accountRepo.EXPECT().ExistsForUser(s.testUserID, "savings").Return(false, nil)
	s.accountRepo.EXPECT().GenerateUniqueAccountNumber("savings").Return("2012345679", nil)
	s.accountRepo.EXPECT().CreateWithTransaction(gomock.Any(), gomock.Any()).DoAndReturn(
		func(account *models.Account, transactions []models.Transaction) error {
			account.ID = s.testAccountID
			account.CreatedAt = s.testTime
			account.UpdatedAt = s.testTime
			return nil
		})
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil)

	account, err := s.service.CreateAccount(s.testUserID, "savings", decimal.Zero)
	s.NoError(err)
	s.NotNil(account)
	s.Equal(decimal.Zero, account.Balance)
}

func (s *AccountServiceSuite) TestCreateAccount_UserNotFound() {
	s.userRepo.EXPECT().GetByID(s.testUserID).Return(nil, repositories.ErrUserNotFound)

	account, err := s.service.CreateAccount(s.testUserID, "checking", decimal.Zero)
	s.Error(err)
	s.Nil(account)
	s.Equal(ErrUserNotFound, err)
}

func (s *AccountServiceSuite) TestCreateAccount_NegativeInitialDeposit() {
	s.userRepo.EXPECT().GetByID(s.testUserID).Return(s.testUser, nil)
	s.accountRepo.EXPECT().ExistsForUser(s.testUserID, "checking").Return(false, nil)

	account, err := s.service.CreateAccount(s.testUserID, "checking", decimal.NewFromFloat(-100))
	s.Error(err)
	s.Nil(account)
	s.Equal(ErrInvalidAmount, err)
}

func (s *AccountServiceSuite) TestCreateAccount_AccountAlreadyExists() {
	s.userRepo.EXPECT().GetByID(s.testUserID).Return(s.testUser, nil)
	s.accountRepo.EXPECT().ExistsForUser(s.testUserID, "checking").Return(true, nil)

	account, err := s.service.CreateAccount(s.testUserID, "checking", decimal.Zero)
	s.Error(err)
	s.Nil(account)
	s.Equal(ErrAccountAlreadyExists, err)
}

// Test GetAccountByID functionality
func (s *AccountServiceSuite) TestGetAccountByID_WithoutUserVerification() {
	account := &models.Account{
		ID:            s.testAccountID,
		UserID:        s.testUserID,
		AccountNumber: "1012345678",
		AccountType:   "checking",
		Balance:       decimal.NewFromFloat(500),
		Status:        "active",
	}

	s.accountRepo.EXPECT().GetByID(s.testAccountID).Return(account, nil)

	result, err := s.service.GetAccountByID(s.testAccountID, nil)
	s.NoError(err)
	s.Equal(account, result)
}

func (s *AccountServiceSuite) TestGetAccountByID_WithOwnerVerification() {
	account := &models.Account{
		ID:            s.testAccountID,
		UserID:        s.testUserID,
		AccountNumber: "1012345678",
		AccountType:   "checking",
		Balance:       decimal.NewFromFloat(500),
		Status:        "active",
	}

	s.accountRepo.EXPECT().GetByID(s.testAccountID).Return(account, nil)

	// Owner can access their own account
	result, err := s.service.GetAccountByID(s.testAccountID, &s.testUserID)
	s.NoError(err)
	s.Equal(account, result)
}

func (s *AccountServiceSuite) TestGetAccountByID_AdminAccess() {
	adminID := uuid.New()
	adminUser := &models.User{
		ID:    adminID,
		Email: "admin@example.com",
		Role:  "admin",
	}

	account := &models.Account{
		ID:            s.testAccountID,
		UserID:        s.testUserID,
		AccountNumber: "1012345678",
		AccountType:   "checking",
		Balance:       decimal.NewFromFloat(500),
		Status:        "active",
	}

	s.accountRepo.EXPECT().GetByID(s.testAccountID).Return(account, nil)
	s.userRepo.EXPECT().GetByID(adminID).Return(adminUser, nil)

	// Admin can access any account
	result, err := s.service.GetAccountByID(s.testAccountID, &adminID)
	s.NoError(err)
	s.Equal(account, result)
}

func (s *AccountServiceSuite) TestGetAccountByID_UnauthorizedAccess() {
	otherUserID := uuid.New()
	otherUser := &models.User{
		ID:   otherUserID,
		Role: "user",
	}

	account := &models.Account{
		ID:            s.testAccountID,
		UserID:        s.testUserID,
		AccountNumber: "1012345678",
		AccountType:   "checking",
		Balance:       decimal.NewFromFloat(500),
		Status:        "active",
	}

	s.accountRepo.EXPECT().GetByID(s.testAccountID).Return(account, nil)
	s.userRepo.EXPECT().GetByID(otherUserID).Return(otherUser, nil)

	result, err := s.service.GetAccountByID(s.testAccountID, &otherUserID)
	s.Error(err)
	s.Nil(result)
	s.Equal(ErrUnauthorized, err)
}

func (s *AccountServiceSuite) TestGetAccountByID_NotFound() {
	s.accountRepo.EXPECT().GetByID(s.testAccountID).Return(nil, repositories.ErrAccountNotFound)

	result, err := s.service.GetAccountByID(s.testAccountID, nil)
	s.Error(err)
	s.Nil(result)
	s.Equal(ErrAccountNotFound, err)
}

// Test PerformTransaction functionality
func (s *AccountServiceSuite) TestPerformTransaction_Credit() {
	account := &models.Account{
		ID:            s.testAccountID,
		UserID:        s.testUserID,
		AccountNumber: "1012345678",
		AccountType:   "checking",
		Balance:       decimal.NewFromFloat(500),
		Status:        "active",
	}

	s.accountRepo.EXPECT().GetByID(s.testAccountID).Return(account, nil)
	s.accountRepo.EXPECT().UpdateBalance(s.testAccountID, decimal.NewFromFloat(50), "credit").Return(nil)
	s.transactionRepo.EXPECT().Create(gomock.Any()).DoAndReturn(
		func(t *models.Transaction) error {
			t.ID = uuid.New()
			t.CreatedAt = s.testTime
			t.UpdatedAt = s.testTime
			return nil
		})
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil)

	transaction, err := s.service.PerformTransaction(s.testAccountID, decimal.NewFromFloat(50), "credit", "Deposit", &s.testUserID)
	s.NoError(err)
	s.NotNil(transaction)
	s.Equal(decimal.NewFromFloat(50), transaction.Amount)
	s.Equal("credit", transaction.TransactionType)
}

func (s *AccountServiceSuite) TestPerformTransaction_Debit() {
	account := &models.Account{
		ID:            s.testAccountID,
		UserID:        s.testUserID,
		AccountNumber: "1012345678",
		AccountType:   "checking",
		Balance:       decimal.NewFromFloat(500),
		Status:        "active",
	}

	s.accountRepo.EXPECT().GetByID(s.testAccountID).Return(account, nil)
	s.accountRepo.EXPECT().UpdateBalance(s.testAccountID, decimal.NewFromFloat(100), "debit").Return(nil)
	s.transactionRepo.EXPECT().Create(gomock.Any()).DoAndReturn(
		func(t *models.Transaction) error {
			t.ID = uuid.New()
			t.CreatedAt = s.testTime
			t.UpdatedAt = s.testTime
			return nil
		})
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil)

	transaction, err := s.service.PerformTransaction(s.testAccountID, decimal.NewFromFloat(100), "debit", "Withdrawal", &s.testUserID)
	s.NoError(err)
	s.NotNil(transaction)
	s.Equal(decimal.NewFromFloat(100), transaction.Amount)
	s.Equal("debit", transaction.TransactionType)
}

func (s *AccountServiceSuite) TestPerformTransaction_InsufficientFunds() {
	account := &models.Account{
		ID:            s.testAccountID,
		UserID:        s.testUserID,
		AccountNumber: "1012345678",
		AccountType:   "checking",
		Balance:       decimal.NewFromFloat(100),
		Status:        "active",
	}

	s.accountRepo.EXPECT().GetByID(s.testAccountID).Return(account, nil)
	s.accountRepo.EXPECT().UpdateBalance(s.testAccountID, decimal.NewFromFloat(1000), "debit").Return(repositories.ErrInsufficientFunds)

	transaction, err := s.service.PerformTransaction(s.testAccountID, decimal.NewFromFloat(1000), "debit", "Large withdrawal", &s.testUserID)
	s.Error(err)
	s.Nil(transaction)
	s.Equal(ErrInsufficientFunds, err)
}

func (s *AccountServiceSuite) TestPerformTransaction_InactiveAccount() {
	inactiveAccount := &models.Account{
		ID:            s.testAccountID,
		UserID:        s.testUserID,
		AccountNumber: "1012345678",
		AccountType:   "checking",
		Balance:       decimal.NewFromFloat(500),
		Status:        "inactive",
	}

	s.accountRepo.EXPECT().GetByID(s.testAccountID).Return(inactiveAccount, nil)

	transaction, err := s.service.PerformTransaction(s.testAccountID, decimal.NewFromFloat(50), "credit", "Deposit", &s.testUserID)
	s.Error(err)
	s.Nil(transaction)
	s.Equal(ErrAccountNotActive, err)
}

func (s *AccountServiceSuite) TestPerformTransaction_InvalidAmount() {
	transaction, err := s.service.PerformTransaction(s.testAccountID, decimal.NewFromFloat(-50), "credit", "Invalid", nil)
	s.Error(err)
	s.Nil(transaction)
	s.Equal(ErrInvalidAmount, err)
}

// Test TransferBetweenAccounts functionality
func (s *AccountServiceSuite) TestTransferBetweenAccounts_Success() {
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	amount := decimal.NewFromFloat(100)

	fromAccount := &models.Account{
		ID:            fromAccountID,
		UserID:        s.testUserID,
		AccountNumber: "1012345678",
		AccountType:   "checking",
		Balance:       decimal.NewFromFloat(500),
		Status:        "active",
	}

	toAccount := &models.Account{
		ID:            toAccountID,
		UserID:        s.testUserID,
		AccountNumber: "2012345679",
		AccountType:   "savings",
		Balance:       decimal.NewFromFloat(1000),
		Status:        "active",
	}

	idempotencyKey := "test-idempotency-key"
	debitTxID := uuid.New()
	creditTxID := uuid.New()

	// Check for existing transfer with idempotency key
	s.transferRepo.EXPECT().
		FindByIdempotencyKey(idempotencyKey).
		Return(nil, repositories.ErrTransferNotFound)

	// Get both accounts
	s.accountRepo.EXPECT().GetByID(fromAccountID).Return(fromAccount, nil)
	s.accountRepo.EXPECT().GetByID(toAccountID).Return(toAccount, nil)

	// Create transfer entity with pending status
	s.transferRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(transfer *models.Transfer) error {
			transfer.ID = uuid.New()
			return nil
		})

	// Execute atomic transfer
	s.accountRepo.EXPECT().
		ExecuteAtomicTransfer(
			fromAccountID,
			toAccountID,
			amount,
			gomock.Any(), // fromDescription
			gomock.Any(), // toDescription
		).
		Return(debitTxID, creditTxID, nil)

	// Update transfer status to completed
	s.transferRepo.EXPECT().
		Update(gomock.Any()).
		Return(nil)

	// Audit log for successful transfer
	s.auditRepo.EXPECT().
		Create(gomock.Any()).
		Return(nil)

	_, err := s.service.TransferBetweenAccounts(fromAccountID, toAccountID, amount, "Transfer funds", idempotencyKey, s.testUserID)
	s.NoError(err)
}

func (s *AccountServiceSuite) TestTransferBetweenAccounts_SameAccount() {
	accountID := uuid.New()

	_, err := s.service.TransferBetweenAccounts(accountID, accountID, decimal.NewFromFloat(100), "placeholder-description", "placeholder-idempotency-key", s.testUserID)
	s.Error(err)
	s.Equal(ErrSameAccountTransfer, err)
}

func (s *AccountServiceSuite) TestTransferBetweenAccounts_InvalidAmount() {
	fromAccountID := uuid.New()
	toAccountID := uuid.New()

	_, err := s.service.TransferBetweenAccounts(fromAccountID, toAccountID, decimal.NewFromFloat(-100), "placeholder-description", "placeholder-idempotency-key", s.testUserID)
	s.Error(err)
	s.Equal(ErrInvalidAmount, err)
}

// Test GetUserAccounts functionality
func (s *AccountServiceSuite) TestGetUserAccounts() {
	expectedAccounts := []models.Account{
		{
			ID:            uuid.New(),
			UserID:        s.testUserID,
			AccountNumber: "1012345678",
			AccountType:   "checking",
			Balance:       decimal.NewFromFloat(1000),
			Status:        "active",
		},
		{
			ID:            uuid.New(),
			UserID:        s.testUserID,
			AccountNumber: "2012345679",
			AccountType:   "savings",
			Balance:       decimal.NewFromFloat(5000),
			Status:        "active",
		},
	}

	s.accountRepo.EXPECT().GetByUserID(s.testUserID).Return(expectedAccounts, nil)

	accounts, err := s.service.GetUserAccounts(s.testUserID)
	s.NoError(err)
	s.Len(accounts, 2)
	s.Equal(expectedAccounts, accounts)
}

// Test CloseAccount functionality
func (s *AccountServiceSuite) TestCloseAccount_Success() {
	account := &models.Account{
		ID:            s.testAccountID,
		UserID:        s.testUserID,
		AccountNumber: "1012345678",
		AccountType:   "checking",
		Balance:       decimal.Zero,
		Status:        "active",
	}

	s.accountRepo.EXPECT().GetByID(s.testAccountID).Return(account, nil)
	s.accountRepo.EXPECT().Update(gomock.Any()).DoAndReturn(
		func(a *models.Account) error {
			s.Equal("closed", a.Status)
			return nil
		})
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil)

	err := s.service.CloseAccount(s.testAccountID, s.testUserID)
	s.NoError(err)
}

func (s *AccountServiceSuite) TestCloseAccount_NonZeroBalance() {
	account := &models.Account{
		ID:            s.testAccountID,
		UserID:        s.testUserID,
		AccountNumber: "1012345678",
		AccountType:   "checking",
		Balance:       decimal.NewFromFloat(100),
		Status:        "active",
	}

	s.accountRepo.EXPECT().GetByID(s.testAccountID).Return(account, nil)

	err := s.service.CloseAccount(s.testAccountID, s.testUserID)
	s.Error(err)
	s.Equal(ErrAccountClosureNotAllowed, err)
}

func (s *AccountServiceSuite) TestCloseAccount_Unauthorized() {
	otherUserID := uuid.New()
	account := &models.Account{
		ID:            s.testAccountID,
		UserID:        otherUserID,
		AccountNumber: "1012345678",
		AccountType:   "checking",
		Balance:       decimal.Zero,
		Status:        "active",
	}

	s.accountRepo.EXPECT().GetByID(s.testAccountID).Return(account, nil)

	// GetAccountByID will check if user is admin since account belongs to different user
	user := &models.User{
		ID:   s.testUserID,
		Role: models.RoleCustomer, // Not admin
	}
	s.userRepo.EXPECT().GetByID(s.testUserID).Return(user, nil)

	err := s.service.CloseAccount(s.testAccountID, s.testUserID)
	s.Error(err)
	s.Equal(ErrUnauthorized, err)
}
