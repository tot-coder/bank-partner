package services

import (
	"errors"
	"fmt"
	"log/slog"

	"array-assessment/internal/models"
	"array-assessment/internal/repositories"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccountAssociationService handles account association operations
type AccountAssociationService struct {
	userRepo     repositories.UserRepositoryInterface
	accountRepo  repositories.AccountRepositoryInterface
	auditService AuditServiceInterface
	logger       *slog.Logger
}

// NewAccountAssociationService creates a new account association service
func NewAccountAssociationService(userRepo repositories.UserRepositoryInterface, accountRepo repositories.AccountRepositoryInterface, auditService AuditServiceInterface, logger *slog.Logger) AccountAssociationServiceInterface {
	return &AccountAssociationService{
		userRepo:     userRepo,
		accountRepo:  accountRepo,
		auditService: auditService,
		logger:       logger,
	}
}

var (
	ErrInvalidAccountID     = errors.New("invalid account ID")
	ErrInvalidPerformedBy   = errors.New("invalid performed by ID")
	ErrSameCustomer         = errors.New("cannot transfer account to the same customer")
	ErrFromCustomerMismatch = errors.New("account does not belong to the specified customer")
)

// GetCustomerAccounts retrieves all accounts for a customer (excludes closed and deleted accounts)
func (s *AccountAssociationService) GetCustomerAccounts(customerID uuid.UUID) ([]*models.Account, error) {
	if customerID == uuid.Nil {
		return nil, ErrInvalidCustomerID
	}

	_, err := s.userRepo.GetByIDActive(customerID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrCustomerNotFound
		}
		return nil, fmt.Errorf("failed to verify customer: %w", err)
	}

	accounts, err := s.accountRepo.GetByUserIDExcludingStatus(customerID, models.AccountStatusClosed)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	return accounts, nil
}

// CreateAccountForCustomer creates a new account for a customer (admin operation)
func (s *AccountAssociationService) CreateAccountForCustomer(customerID, performedBy uuid.UUID, accountType, ipAddress, userAgent string) (*models.Account, error) {
	if customerID == uuid.Nil {
		return nil, ErrInvalidCustomerID
	}

	if performedBy == uuid.Nil {
		return nil, ErrInvalidPerformedBy
	}

	if !models.IsValidAccountType(accountType) {
		return nil, models.ErrInvalidAccountType
	}

	_, err := s.userRepo.GetByIDActive(customerID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrCustomerNotFound
		}
		return nil, fmt.Errorf("failed to verify customer: %w", err)
	}

	accountNumber, err := s.accountRepo.GenerateUniqueAccountNumber(accountType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate unique account number: %w", err)
	}

	account := &models.Account{
		UserID:        customerID,
		AccountNumber: accountNumber,
		AccountType:   accountType,
		Balance:       decimal.Zero,
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}

	if err := s.accountRepo.Create(account); err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	if s.auditService != nil {
		if err := s.auditService.LogAccountCreated(customerID, performedBy, account.ID, accountType, ipAddress, userAgent); err != nil {
			// Log error but don't fail the operation
			s.logger.Error("failed to log account creation",
				"error", err,
				"customer_id", customerID,
				"account_id", account.ID,
				"account_type", accountType)
		}
	}

	return account, nil
}

// TransferAccountOwnership transfers an account from one customer to another (atomic transaction)
func (s *AccountAssociationService) TransferAccountOwnership(accountID, fromCustomerID, toCustomerID, performedBy uuid.UUID, ipAddress, userAgent string) error {
	if accountID == uuid.Nil {
		return ErrInvalidAccountID
	}

	if fromCustomerID == uuid.Nil || toCustomerID == uuid.Nil {
		return ErrInvalidCustomerID
	}

	if performedBy == uuid.Nil {
		return ErrInvalidPerformedBy
	}

	if fromCustomerID == toCustomerID {
		return ErrSameCustomer
	}

	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return ErrAccountNotFound
		}
		return fmt.Errorf("failed to find account: %w", err)
	}

	if account.UserID != fromCustomerID {
		return ErrFromCustomerMismatch
	}

	_, err = s.userRepo.GetByIDActive(fromCustomerID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return ErrCustomerNotFound
		}
		return fmt.Errorf("failed to verify from customer: %w", err)
	}

	_, err = s.userRepo.GetByIDActive(toCustomerID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return ErrCustomerNotFound
		}
		return fmt.Errorf("failed to verify to customer: %w", err)
	}

	if err := s.accountRepo.UpdateOwnership(accountID, toCustomerID); err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return ErrAccountNotFound
		}
		return fmt.Errorf("failed to transfer account ownership: %w", err)
	}

	if s.auditService != nil {
		if err := s.auditService.LogAccountTransferred(fromCustomerID, toCustomerID, performedBy, accountID, ipAddress, userAgent); err != nil {
			// Log error but don't fail the operation
			s.logger.Error("failed to log account transfer",
				"error", err,
				"from_customer_id", fromCustomerID,
				"to_customer_id", toCustomerID,
				"account_id", accountID)
		}
	}

	return nil
}
