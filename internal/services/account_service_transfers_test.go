package services

import (
	"errors"
	"log/slog"
	"testing"

	"array-assessment/internal/models"
	"array-assessment/internal/repositories"
	"array-assessment/internal/repositories/repository_mocks"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// TransferServiceTestSuite defines tests for transfer functionality
type TransferServiceTestSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	accountRepo     *repository_mocks.MockAccountRepositoryInterface
	transactionRepo *repository_mocks.MockTransactionRepositoryInterface
	transferRepo    *repository_mocks.MockTransferRepositoryInterface
	userRepo        *repository_mocks.MockUserRepositoryInterface
	auditRepo       *repository_mocks.MockAuditLogRepositoryInterface
	db              *gorm.DB
	service         AccountServiceInterface
}

// SetupTest runs before each test
func (s *TransferServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.accountRepo = repository_mocks.NewMockAccountRepositoryInterface(s.ctrl)
	s.transactionRepo = repository_mocks.NewMockTransactionRepositoryInterface(s.ctrl)
	s.transferRepo = repository_mocks.NewMockTransferRepositoryInterface(s.ctrl)
	s.userRepo = repository_mocks.NewMockUserRepositoryInterface(s.ctrl)
	s.auditRepo = repository_mocks.NewMockAuditLogRepositoryInterface(s.ctrl)

	// Create service with mocked repositories
	s.service = NewAccountService(
		s.accountRepo,
		s.transactionRepo,
		s.transferRepo,
		s.userRepo,
		s.auditRepo,
		slog.Default(),
	)
}

// TearDownTest runs after each test
func (s *TransferServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

// TestTransferServiceTestSuite runs the test suite
func TestTransferServiceTestSuite(t *testing.T) {
	suite.Run(t, new(TransferServiceTestSuite))
}

// TestTransferBetweenAccounts_Success tests successful transfer execution
func (s *TransferServiceTestSuite) TestTransferBetweenAccounts_Success() {
	userID := uuid.New()
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	amount := decimal.NewFromFloat(100.00)
	idempotencyKey := uuid.New().String()
	debitTxID := uuid.New()
	creditTxID := uuid.New()

	fromAccount := &models.Account{
		ID:            fromAccountID,
		UserID:        userID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(500.00),
		Status:        models.AccountStatusActive,
	}

	toAccount := &models.Account{
		ID:            toAccountID,
		UserID:        uuid.New(),
		AccountNumber: "2023456789",
		AccountType:   models.AccountTypeSavings,
		Balance:       decimal.NewFromFloat(200.00),
		Status:        models.AccountStatusActive,
	}

	// Check for existing transfer with idempotency key
	s.transferRepo.EXPECT().
		FindByIdempotencyKey(idempotencyKey).
		Return(nil, repositories.ErrTransferNotFound)

	// Get source account
	s.accountRepo.EXPECT().
		GetByID(fromAccountID).
		Return(fromAccount, nil)

	// Get destination account
	s.accountRepo.EXPECT().
		GetByID(toAccountID).
		Return(toAccount, nil)

	// Create transfer entity with pending status
	s.transferRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(transfer *models.Transfer) error {
			s.Equal(fromAccountID, transfer.FromAccountID)
			s.Equal(toAccountID, transfer.ToAccountID)
			s.True(amount.Equal(transfer.Amount))
			s.Equal(idempotencyKey, transfer.IdempotencyKey)
			s.Equal(models.TransferStatusPending, transfer.Status)
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
		DoAndReturn(func(transfer *models.Transfer) error {
			s.Equal(models.TransferStatusCompleted, transfer.Status)
			s.Equal(&debitTxID, transfer.DebitTransactionID)
			s.Equal(&creditTxID, transfer.CreditTransactionID)
			return nil
		})

	// Audit log for successful transfer
	s.auditRepo.EXPECT().
		Create(gomock.Any()).
		Return(nil)

	result, err := s.service.TransferBetweenAccounts(
		fromAccountID,
		toAccountID,
		amount,
		"Test transfer",
		idempotencyKey,
		userID,
	)

	s.NoError(err)
	s.NotNil(result)
	s.Equal(models.TransferStatusCompleted, result.Status)
}

// TestTransferBetweenAccounts_IdempotencyKeyExists_Completed tests idempotent behavior for completed transfer
func (s *TransferServiceTestSuite) TestTransferBetweenAccounts_IdempotencyKeyExists_Completed() {
	userID := uuid.New()
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	amount := decimal.NewFromFloat(100.00)
	idempotencyKey := uuid.New().String()

	existingTransfer := &models.Transfer{
		ID:             uuid.New(),
		FromAccountID:  fromAccountID,
		ToAccountID:    toAccountID,
		Amount:         amount,
		IdempotencyKey: idempotencyKey,
		Status:         models.TransferStatusCompleted,
	}

	// Check for existing transfer - found completed
	s.transferRepo.EXPECT().
		FindByIdempotencyKey(idempotencyKey).
		Return(existingTransfer, nil)

	result, err := s.service.TransferBetweenAccounts(
		fromAccountID,
		toAccountID,
		amount,
		"Test transfer",
		idempotencyKey,
		userID,
	)

	s.NoError(err)
	s.NotNil(result)
	s.Equal(existingTransfer.ID, result.ID)
	s.Equal(models.TransferStatusCompleted, result.Status)
}

// TestTransferBetweenAccounts_IdempotencyKeyExists_Pending tests conflict for pending transfer
func (s *TransferServiceTestSuite) TestTransferBetweenAccounts_IdempotencyKeyExists_Pending() {
	userID := uuid.New()
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	amount := decimal.NewFromFloat(100.00)
	idempotencyKey := uuid.New().String()

	existingTransfer := &models.Transfer{
		ID:             uuid.New(),
		FromAccountID:  fromAccountID,
		ToAccountID:    toAccountID,
		Amount:         amount,
		IdempotencyKey: idempotencyKey,
		Status:         models.TransferStatusPending,
	}

	// Check for existing transfer - found pending
	s.transferRepo.EXPECT().
		FindByIdempotencyKey(idempotencyKey).
		Return(existingTransfer, nil)

	result, err := s.service.TransferBetweenAccounts(
		fromAccountID,
		toAccountID,
		amount,
		"Test transfer",
		idempotencyKey,
		userID,
	)

	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "transfer is still processing")
}

// TestTransferBetweenAccounts_IdempotencyKeyExists_Failed tests conflict for failed transfer
func (s *TransferServiceTestSuite) TestTransferBetweenAccounts_IdempotencyKeyExists_Failed() {
	userID := uuid.New()
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	amount := decimal.NewFromFloat(100.00)
	idempotencyKey := uuid.New().String()

	existingTransfer := &models.Transfer{
		ID:             uuid.New(),
		FromAccountID:  fromAccountID,
		ToAccountID:    toAccountID,
		Amount:         amount,
		IdempotencyKey: idempotencyKey,
		Status:         models.TransferStatusFailed,
	}

	// Check for existing transfer - found failed
	s.transferRepo.EXPECT().
		FindByIdempotencyKey(idempotencyKey).
		Return(existingTransfer, nil)

	result, err := s.service.TransferBetweenAccounts(
		fromAccountID,
		toAccountID,
		amount,
		"Test transfer",
		idempotencyKey,
		userID,
	)

	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "previous transfer failed")
}

// TestTransferBetweenAccounts_InsufficientFunds tests rollback on insufficient funds
func (s *TransferServiceTestSuite) TestTransferBetweenAccounts_InsufficientFunds() {
	userID := uuid.New()
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	amount := decimal.NewFromFloat(1000.00)
	idempotencyKey := uuid.New().String()

	fromAccount := &models.Account{
		ID:            fromAccountID,
		UserID:        userID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(50.00), // Insufficient
		Status:        models.AccountStatusActive,
	}

	toAccount := &models.Account{
		ID:            toAccountID,
		UserID:        uuid.New(),
		AccountNumber: "2023456789",
		AccountType:   models.AccountTypeSavings,
		Balance:       decimal.NewFromFloat(200.00),
		Status:        models.AccountStatusActive,
	}

	// Check for existing transfer
	s.transferRepo.EXPECT().
		FindByIdempotencyKey(idempotencyKey).
		Return(nil, repositories.ErrTransferNotFound)

	// Get accounts
	s.accountRepo.EXPECT().GetByID(fromAccountID).Return(fromAccount, nil)
	s.accountRepo.EXPECT().GetByID(toAccountID).Return(toAccount, nil)

	// Create transfer with pending status
	s.transferRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(transfer *models.Transfer) error {
			transfer.ID = uuid.New()
			return nil
		})

	// Execute atomic transfer - should fail with insufficient funds
	s.accountRepo.EXPECT().
		ExecuteAtomicTransfer(
			fromAccountID,
			toAccountID,
			amount,
			gomock.Any(), // fromDescription
			gomock.Any(), // toDescription
		).
		Return(uuid.Nil, uuid.Nil, repositories.ErrInsufficientFunds)

	// Update transfer to failed
	s.transferRepo.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(transfer *models.Transfer) error {
			s.Equal(models.TransferStatusFailed, transfer.Status)
			s.NotNil(transfer.ErrorMessage)
			s.NotNil(transfer.FailedAt)
			return nil
		})

	// Audit log for failed transfer
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)

	result, err := s.service.TransferBetweenAccounts(
		fromAccountID,
		toAccountID,
		amount,
		"Test transfer",
		idempotencyKey,
		userID,
	)

	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "insufficient funds")
}

// TestTransferBetweenAccounts_SameAccount tests validation for same account transfer
func (s *TransferServiceTestSuite) TestTransferBetweenAccounts_SameAccount() {
	userID := uuid.New()
	accountID := uuid.New()
	amount := decimal.NewFromFloat(100.00)
	idempotencyKey := uuid.New().String()

	result, err := s.service.TransferBetweenAccounts(
		accountID,
		accountID,
		amount,
		"Test transfer",
		idempotencyKey,
		userID,
	)

	s.Error(err)
	s.Nil(result)
	s.Equal(ErrSameAccountTransfer, err)
}

// TestTransferBetweenAccounts_InvalidAmount tests validation for invalid amount
func (s *TransferServiceTestSuite) TestTransferBetweenAccounts_InvalidAmount() {
	userID := uuid.New()
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	amount := decimal.NewFromFloat(-50.00)
	idempotencyKey := uuid.New().String()

	result, err := s.service.TransferBetweenAccounts(
		fromAccountID,
		toAccountID,
		amount,
		"Test transfer",
		idempotencyKey,
		userID,
	)

	s.Error(err)
	s.Nil(result)
	s.Equal(ErrInvalidAmount, err)
}

// TestTransferBetweenAccounts_SourceAccountNotFound tests error handling
func (s *TransferServiceTestSuite) TestTransferBetweenAccounts_SourceAccountNotFound() {
	userID := uuid.New()
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	amount := decimal.NewFromFloat(100.00)
	idempotencyKey := uuid.New().String()

	// Check for existing transfer
	s.transferRepo.EXPECT().
		FindByIdempotencyKey(idempotencyKey).
		Return(nil, repositories.ErrTransferNotFound)

	// Get source account - not found
	s.accountRepo.EXPECT().
		GetByID(fromAccountID).
		Return(nil, repositories.ErrAccountNotFound)

	result, err := s.service.TransferBetweenAccounts(
		fromAccountID,
		toAccountID,
		amount,
		"Test transfer",
		idempotencyKey,
		userID,
	)

	s.Error(err)
	s.Nil(result)
	s.Equal(ErrAccountNotFound, err)
}

// TestTransferBetweenAccounts_DestinationAccountNotFound tests error handling
func (s *TransferServiceTestSuite) TestTransferBetweenAccounts_DestinationAccountNotFound() {
	userID := uuid.New()
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	amount := decimal.NewFromFloat(100.00)
	idempotencyKey := uuid.New().String()

	fromAccount := &models.Account{
		ID:            fromAccountID,
		UserID:        userID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(500.00),
		Status:        models.AccountStatusActive,
	}

	// Check for existing transfer
	s.transferRepo.EXPECT().
		FindByIdempotencyKey(idempotencyKey).
		Return(nil, repositories.ErrTransferNotFound)

	// Get source account
	s.accountRepo.EXPECT().
		GetByID(fromAccountID).
		Return(fromAccount, nil)

	// Get destination account - not found
	s.accountRepo.EXPECT().
		GetByID(toAccountID).
		Return(nil, repositories.ErrAccountNotFound)

	result, err := s.service.TransferBetweenAccounts(
		fromAccountID,
		toAccountID,
		amount,
		"Test transfer",
		idempotencyKey,
		userID,
	)

	s.Error(err)
	s.Nil(result)
	s.Equal(ErrAccountNotFound, err)
}

// TestTransferBetweenAccounts_Unauthorized tests authorization
func (s *TransferServiceTestSuite) TestTransferBetweenAccounts_Unauthorized() {
	userID := uuid.New()
	otherUserID := uuid.New()
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	amount := decimal.NewFromFloat(100.00)
	idempotencyKey := uuid.New().String()

	fromAccount := &models.Account{
		ID:            fromAccountID,
		UserID:        otherUserID, // Different user
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(500.00),
		Status:        models.AccountStatusActive,
	}

	toAccount := &models.Account{
		ID:            toAccountID,
		UserID:        uuid.New(),
		AccountNumber: "2023456789",
		AccountType:   models.AccountTypeSavings,
		Balance:       decimal.NewFromFloat(200.00),
		Status:        models.AccountStatusActive,
	}

	// Check for existing transfer
	s.transferRepo.EXPECT().
		FindByIdempotencyKey(idempotencyKey).
		Return(nil, repositories.ErrTransferNotFound)

	// Get source account
	s.accountRepo.EXPECT().
		GetByID(fromAccountID).
		Return(fromAccount, nil)

	// Get destination account
	s.accountRepo.EXPECT().
		GetByID(toAccountID).
		Return(toAccount, nil)

	result, err := s.service.TransferBetweenAccounts(
		fromAccountID,
		toAccountID,
		amount,
		"Test transfer",
		idempotencyKey,
		userID,
	)

	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "not authorized")
}

// TestTransferBetweenAccounts_InactiveSourceAccount tests validation
func (s *TransferServiceTestSuite) TestTransferBetweenAccounts_InactiveSourceAccount() {
	userID := uuid.New()
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	amount := decimal.NewFromFloat(100.00)
	idempotencyKey := uuid.New().String()

	fromAccount := &models.Account{
		ID:            fromAccountID,
		UserID:        userID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(500.00),
		Status:        models.AccountStatusInactive,
	}

	toAccount := &models.Account{
		ID:            toAccountID,
		UserID:        uuid.New(),
		AccountNumber: "2023456789",
		AccountType:   models.AccountTypeSavings,
		Balance:       decimal.NewFromFloat(200.00),
		Status:        models.AccountStatusActive,
	}

	// Check for existing transfer
	s.transferRepo.EXPECT().
		FindByIdempotencyKey(idempotencyKey).
		Return(nil, repositories.ErrTransferNotFound)

	// Get accounts
	s.accountRepo.EXPECT().GetByID(fromAccountID).Return(fromAccount, nil)
	s.accountRepo.EXPECT().GetByID(toAccountID).Return(toAccount, nil)

	result, err := s.service.TransferBetweenAccounts(
		fromAccountID,
		toAccountID,
		amount,
		"Test transfer",
		idempotencyKey,
		userID,
	)

	s.Error(err)
	s.Nil(result)
	s.Equal(ErrAccountNotActive, err)
}

// TestTransferBetweenAccounts_TransactionRollback tests database rollback
func (s *TransferServiceTestSuite) TestTransferBetweenAccounts_TransactionRollback() {
	userID := uuid.New()
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	amount := decimal.NewFromFloat(100.00)
	idempotencyKey := uuid.New().String()

	fromAccount := &models.Account{
		ID:            fromAccountID,
		UserID:        userID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(500.00),
		Status:        models.AccountStatusActive,
	}

	toAccount := &models.Account{
		ID:            toAccountID,
		UserID:        uuid.New(),
		AccountNumber: "2023456789",
		AccountType:   models.AccountTypeSavings,
		Balance:       decimal.NewFromFloat(200.00),
		Status:        models.AccountStatusActive,
	}

	// Check for existing transfer
	s.transferRepo.EXPECT().
		FindByIdempotencyKey(idempotencyKey).
		Return(nil, repositories.ErrTransferNotFound)

	// Get accounts
	s.accountRepo.EXPECT().GetByID(fromAccountID).Return(fromAccount, nil)
	s.accountRepo.EXPECT().GetByID(toAccountID).Return(toAccount, nil)

	// Create transfer
	s.transferRepo.EXPECT().
		Create(gomock.Any()).
		Return(errors.New("database error"))

	result, err := s.service.TransferBetweenAccounts(
		fromAccountID,
		toAccountID,
		amount,
		"Test transfer",
		idempotencyKey,
		userID,
	)

	s.Error(err)
	s.Nil(result)
}
