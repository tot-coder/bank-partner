package services

import (
	"errors"
	"log/slog"
	"testing"

	"array-assessment/internal/models"
	"array-assessment/internal/repositories"
	"array-assessment/internal/repositories/repository_mocks"
	"array-assessment/internal/services/service_mocks"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// AccountAssociationServiceTestSuite is the test suite for AccountAssociationService
type AccountAssociationServiceTestSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	mockUserRepo    *repository_mocks.MockUserRepositoryInterface
	mockAccountRepo *repository_mocks.MockAccountRepositoryInterface
	service         AccountAssociationServiceInterface
	auditService    *service_mocks.MockAuditServiceInterface
}

// SetupTest initializes the test suite before each test
func (s *AccountAssociationServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockUserRepo = repository_mocks.NewMockUserRepositoryInterface(s.ctrl)
	s.mockAccountRepo = repository_mocks.NewMockAccountRepositoryInterface(s.ctrl)
	s.auditService = service_mocks.NewMockAuditServiceInterface(s.ctrl)
	s.service = NewAccountAssociationService(s.mockUserRepo, s.mockAccountRepo, s.auditService, slog.Default())
}

// TearDownTest cleans up after each test
func (s *AccountAssociationServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

// TestAccountAssociationServiceSuite runs the test suite
func TestAccountAssociationServiceSuite(t *testing.T) {
	suite.Run(t, new(AccountAssociationServiceTestSuite))
}

// TestGetCustomerAccounts_Success tests successful retrieval of accounts
func (s *AccountAssociationServiceTestSuite) TestGetCustomerAccounts_Success() {
	customerID := uuid.New()
	user := &models.User{
		ID:        customerID,
		Email:     gofakeit.Email(),
		FirstName: gofakeit.FirstName(),
		LastName:  gofakeit.LastName(),
	}
	accounts := []*models.Account{
		{ID: uuid.New(), UserID: customerID, AccountNumber: "CHK1234567890", Status: models.AccountStatusActive},
		{ID: uuid.New(), UserID: customerID, AccountNumber: "SAV0987654321", Status: models.AccountStatusActive},
	}

	s.mockUserRepo.EXPECT().GetByIDActive(customerID).Return(user, nil)
	s.mockAccountRepo.EXPECT().GetByUserIDExcludingStatus(customerID, models.AccountStatusClosed).Return(accounts, nil)

	result, err := s.service.GetCustomerAccounts(customerID)

	s.NoError(err)
	s.Len(result, 2)
}

// TestGetCustomerAccounts_NilCustomerID tests with nil customer ID
func (s *AccountAssociationServiceTestSuite) TestGetCustomerAccounts_NilCustomerID() {
	_, err := s.service.GetCustomerAccounts(uuid.Nil)

	s.Error(err)
	s.ErrorIs(err, ErrInvalidCustomerID)
}

// TestGetCustomerAccounts_CustomerNotFound tests when customer is not found
func (s *AccountAssociationServiceTestSuite) TestGetCustomerAccounts_CustomerNotFound() {
	customerID := uuid.New()

	s.mockUserRepo.EXPECT().GetByIDActive(customerID).Return(nil, repositories.ErrUserNotFound)

	_, err := s.service.GetCustomerAccounts(customerID)

	s.Error(err)
	s.ErrorIs(err, ErrCustomerNotFound)
}

// TestGetCustomerAccounts_RepositoryError tests repository error handling
func (s *AccountAssociationServiceTestSuite) TestGetCustomerAccounts_RepositoryError() {
	customerID := uuid.New()
	user := &models.User{ID: customerID}

	s.mockUserRepo.EXPECT().GetByIDActive(customerID).Return(user, nil)
	s.mockAccountRepo.EXPECT().GetByUserIDExcludingStatus(customerID, models.AccountStatusClosed).Return(nil, errors.New("database error"))

	_, err := s.service.GetCustomerAccounts(customerID)

	s.Error(err)
}

// TestCreateAccountForCustomer_Success tests successful account creation
func (s *AccountAssociationServiceTestSuite) TestCreateAccountForCustomer_Success() {
	customerID := uuid.New()
	performedBy := uuid.New()
	accountType := models.AccountTypeChecking
	user := &models.User{ID: customerID, Email: gofakeit.Email()}

	s.mockUserRepo.EXPECT().GetByIDActive(customerID).Return(user, nil)
	s.mockAccountRepo.EXPECT().GenerateUniqueAccountNumber(accountType).Return("CHK1234567890", nil)
	s.mockAccountRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(account *models.Account) error {
		// Simulate setting the ID that would happen in the database
		if account.ID == uuid.Nil {
			account.ID = uuid.New()
		}
		return nil
	})
	s.auditService.EXPECT().LogAccountCreated(customerID, performedBy, gomock.Any(), accountType, "127.0.0.1", "test-agent").Return(nil)

	account, err := s.service.CreateAccountForCustomer(customerID, performedBy, accountType, "127.0.0.1", "test-agent")

	s.NoError(err)
	s.NotNil(account)
	s.Equal(customerID, account.UserID)
	s.Equal(accountType, account.AccountType)
	s.Equal(models.AccountStatusActive, account.Status)
	s.True(account.Balance.IsZero())
}

// TestCreateAccountForCustomer_NilCustomerID tests with nil customer ID
func (s *AccountAssociationServiceTestSuite) TestCreateAccountForCustomer_NilCustomerID() {
	performedBy := uuid.New()
	accountType := models.AccountTypeChecking

	account, err := s.service.CreateAccountForCustomer(uuid.Nil, performedBy, accountType, "127.0.0.1", "test-agent")

	s.Error(err)
	s.ErrorIs(err, ErrInvalidCustomerID)
	s.Nil(account)
}

// TestCreateAccountForCustomer_NilPerformedBy tests with nil performed by ID
func (s *AccountAssociationServiceTestSuite) TestCreateAccountForCustomer_NilPerformedBy() {
	customerID := uuid.New()
	accountType := models.AccountTypeChecking

	account, err := s.service.CreateAccountForCustomer(customerID, uuid.Nil, accountType, "127.0.0.1", "test-agent")

	s.Error(err)
	s.ErrorIs(err, ErrInvalidPerformedBy)
	s.Nil(account)
}

// TestCreateAccountForCustomer_InvalidAccountType tests with invalid account type
func (s *AccountAssociationServiceTestSuite) TestCreateAccountForCustomer_InvalidAccountType() {
	customerID := uuid.New()
	performedBy := uuid.New()

	account, err := s.service.CreateAccountForCustomer(customerID, performedBy, "INVALID", "127.0.0.1", "test-agent")

	s.Error(err)
	s.ErrorIs(err, models.ErrInvalidAccountType)
	s.Nil(account)
}

// TestCreateAccountForCustomer_CustomerNotFound tests when customer is not found
func (s *AccountAssociationServiceTestSuite) TestCreateAccountForCustomer_CustomerNotFound() {
	customerID := uuid.New()
	performedBy := uuid.New()
	accountType := models.AccountTypeSavings

	s.mockUserRepo.EXPECT().GetByIDActive(customerID).Return(nil, repositories.ErrUserNotFound)

	account, err := s.service.CreateAccountForCustomer(customerID, performedBy, accountType, "127.0.0.1", "test-agent")

	s.Error(err)
	s.ErrorIs(err, ErrCustomerNotFound)
	s.Nil(account)
}

// TestCreateAccountForCustomer_AccountNumberGenerationFailure tests account number generation failure
func (s *AccountAssociationServiceTestSuite) TestCreateAccountForCustomer_AccountNumberGenerationFailure() {
	customerID := uuid.New()
	performedBy := uuid.New()
	accountType := models.AccountTypeChecking
	user := &models.User{ID: customerID}

	s.mockUserRepo.EXPECT().GetByIDActive(customerID).Return(user, nil)
	s.mockAccountRepo.EXPECT().GenerateUniqueAccountNumber(accountType).Return("", errors.New("generation failed"))

	account, err := s.service.CreateAccountForCustomer(customerID, performedBy, accountType, "127.0.0.1", "test-agent")

	s.Error(err)
	s.Nil(account)
}

// TestCreateAccountForCustomer_AccountCreationFailure tests account creation failure
func (s *AccountAssociationServiceTestSuite) TestCreateAccountForCustomer_AccountCreationFailure() {
	customerID := uuid.New()
	performedBy := uuid.New()
	accountType := models.AccountTypeChecking
	user := &models.User{ID: customerID}

	s.mockUserRepo.EXPECT().GetByIDActive(customerID).Return(user, nil)
	s.mockAccountRepo.EXPECT().GenerateUniqueAccountNumber(accountType).Return("CHK1234567890", nil)
	s.mockAccountRepo.EXPECT().Create(gomock.Any()).Return(errors.New("database error"))

	account, err := s.service.CreateAccountForCustomer(customerID, performedBy, accountType, "127.0.0.1", "test-agent")

	s.Error(err)
	s.Nil(account)
}

// TestTransferAccountOwnership_Success tests successful ownership transfer
func (s *AccountAssociationServiceTestSuite) TestTransferAccountOwnership_Success() {
	accountID := uuid.New()
	fromCustomerID := uuid.New()
	toCustomerID := uuid.New()
	performedBy := uuid.New()

	account := &models.Account{
		ID:     accountID,
		UserID: fromCustomerID,
	}
	fromUser := &models.User{ID: fromCustomerID}
	toUser := &models.User{ID: toCustomerID}

	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(account, nil)
	s.mockUserRepo.EXPECT().GetByIDActive(fromCustomerID).Return(fromUser, nil)
	s.mockUserRepo.EXPECT().GetByIDActive(toCustomerID).Return(toUser, nil)
	s.mockAccountRepo.EXPECT().UpdateOwnership(accountID, toCustomerID).Return(nil)
	s.auditService.EXPECT().LogAccountTransferred(fromCustomerID, toCustomerID, performedBy, accountID, "127.0.0.1", "test-agent").Return(nil)

	err := s.service.TransferAccountOwnership(accountID, fromCustomerID, toCustomerID, performedBy, "127.0.0.1", "test-agent")

	s.NoError(err)
}

// TestTransferAccountOwnership_NilAccountID tests with nil account ID
func (s *AccountAssociationServiceTestSuite) TestTransferAccountOwnership_NilAccountID() {
	fromCustomerID := uuid.New()
	toCustomerID := uuid.New()
	performedBy := uuid.New()

	err := s.service.TransferAccountOwnership(uuid.Nil, fromCustomerID, toCustomerID, performedBy, "127.0.0.1", "test-agent")

	s.Error(err)
	s.ErrorIs(err, ErrInvalidAccountID)
}

// TestTransferAccountOwnership_NilFromCustomerID tests with nil from customer ID
func (s *AccountAssociationServiceTestSuite) TestTransferAccountOwnership_NilFromCustomerID() {
	accountID := uuid.New()
	toCustomerID := uuid.New()
	performedBy := uuid.New()

	err := s.service.TransferAccountOwnership(accountID, uuid.Nil, toCustomerID, performedBy, "127.0.0.1", "test-agent")

	s.Error(err)
	s.ErrorIs(err, ErrInvalidCustomerID)
}

// TestTransferAccountOwnership_NilToCustomerID tests with nil to customer ID
func (s *AccountAssociationServiceTestSuite) TestTransferAccountOwnership_NilToCustomerID() {
	accountID := uuid.New()
	fromCustomerID := uuid.New()
	performedBy := uuid.New()

	err := s.service.TransferAccountOwnership(accountID, fromCustomerID, uuid.Nil, performedBy, "127.0.0.1", "test-agent")

	s.Error(err)
	s.ErrorIs(err, ErrInvalidCustomerID)
}

// TestTransferAccountOwnership_NilPerformedBy tests with nil performed by
func (s *AccountAssociationServiceTestSuite) TestTransferAccountOwnership_NilPerformedBy() {
	accountID := uuid.New()
	fromCustomerID := uuid.New()
	toCustomerID := uuid.New()

	err := s.service.TransferAccountOwnership(accountID, fromCustomerID, toCustomerID, uuid.Nil, "127.0.0.1", "test-agent")

	s.Error(err)
	s.ErrorIs(err, ErrInvalidPerformedBy)
}

// TestTransferAccountOwnership_SameCustomer tests transferring to the same customer
func (s *AccountAssociationServiceTestSuite) TestTransferAccountOwnership_SameCustomer() {
	accountID := uuid.New()
	sameCustomerID := uuid.New()
	performedBy := uuid.New()

	err := s.service.TransferAccountOwnership(accountID, sameCustomerID, sameCustomerID, performedBy, "127.0.0.1", "test-agent")

	s.Error(err)
	s.ErrorIs(err, ErrSameCustomer)
}

// TestTransferAccountOwnership_AccountNotFound tests when account is not found
func (s *AccountAssociationServiceTestSuite) TestTransferAccountOwnership_AccountNotFound() {
	accountID := uuid.New()
	fromCustomerID := uuid.New()
	toCustomerID := uuid.New()
	performedBy := uuid.New()

	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(nil, repositories.ErrAccountNotFound)

	err := s.service.TransferAccountOwnership(accountID, fromCustomerID, toCustomerID, performedBy, "127.0.0.1", "test-agent")

	s.Error(err)
	s.ErrorIs(err, ErrAccountNotFound)
}

// TestTransferAccountOwnership_AccountBelongsToDifferentCustomer tests account ownership mismatch
func (s *AccountAssociationServiceTestSuite) TestTransferAccountOwnership_AccountBelongsToDifferentCustomer() {
	accountID := uuid.New()
	fromCustomerID := uuid.New()
	toCustomerID := uuid.New()
	performedBy := uuid.New()
	differentCustomerID := uuid.New()

	account := &models.Account{
		ID:     accountID,
		UserID: differentCustomerID, // Different from fromCustomerID
	}

	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(account, nil)

	err := s.service.TransferAccountOwnership(accountID, fromCustomerID, toCustomerID, performedBy, "127.0.0.1", "test-agent")

	s.Error(err)
	s.ErrorIs(err, ErrFromCustomerMismatch)
}

// TestTransferAccountOwnership_FromCustomerNotFound tests when from customer is not found
func (s *AccountAssociationServiceTestSuite) TestTransferAccountOwnership_FromCustomerNotFound() {
	accountID := uuid.New()
	fromCustomerID := uuid.New()
	toCustomerID := uuid.New()
	performedBy := uuid.New()

	account := &models.Account{
		ID:     accountID,
		UserID: fromCustomerID,
	}

	s.mockAccountRepo.EXPECT().GetByID(accountID).Return(account, nil)
	s.mockUserRepo.EXPECT().GetByIDActive(fromCustomerID).Return(nil, repositories.ErrUserNotFound)

	err := s.service.TransferAccountOwnership(accountID, fromCustomerID, toCustomerID, performedBy, "127.0.0.1", "test-agent")

	s.Error(err)
	s.ErrorIs(err, ErrCustomerNotFound)
}
