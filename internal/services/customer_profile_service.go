package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"array-assessment/internal/models"
	"array-assessment/internal/repositories"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CustomerProfileService handles customer profile operations
type CustomerProfileService struct {
	userRepo     repositories.UserRepositoryInterface
	accountRepo  repositories.AccountRepositoryInterface
	auditService AuditServiceInterface
}

// NewCustomerProfileService creates a new customer profile service
func NewCustomerProfileService(userRepo repositories.UserRepositoryInterface, accountRepo repositories.AccountRepositoryInterface, auditService AuditServiceInterface) CustomerProfileServiceInterface {
	return &CustomerProfileService{
		userRepo:     userRepo,
		accountRepo:  accountRepo,
		auditService: auditService,
	}
}

const (
	TemporaryPasswordLength = 16
	BcryptCost              = 12
)

var (
	ErrCustomerNotFound   = errors.New("customer not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidEmail       = errors.New("invalid email address")
	ErrInvalidCustomerID  = errors.New("invalid customer ID")
	ErrCustomerHasBalance = errors.New("cannot delete customer with non-zero account balances")
	ErrInvalidRole        = errors.New("invalid role")
)

// GetCustomerProfile retrieves a customer profile by ID
func (s *CustomerProfileService) GetCustomerProfile(customerID uuid.UUID) (*models.User, error) {
	if customerID == uuid.Nil {
		return nil, ErrInvalidCustomerID
	}

	user, err := s.userRepo.GetByIDActive(customerID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrCustomerNotFound
		}
		return nil, fmt.Errorf("failed to get customer profile: %w", err)
	}

	return user, nil
}

// CreateCustomer creates a new customer with a temporary password
func (s *CustomerProfileService) CreateCustomer(email, firstName, lastName string, role string) (*models.User, string, error) {
	if email == "" {
		return nil, "", ErrInvalidEmail
	}

	if role != models.RoleCustomer && role != models.RoleAdmin {
		return nil, "", ErrInvalidRole
	}

	existingUser, err := s.userRepo.GetByEmail(email)
	if err != nil && !errors.Is(err, repositories.ErrUserNotFound) {
		return nil, "", fmt.Errorf("failed to check email uniqueness: %w", err)
	}
	if existingUser != nil {
		return nil, "", ErrEmailAlreadyExists
	}

	tempPassword, err := GenerateTemporaryPassword(TemporaryPasswordLength)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate temporary password: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tempPassword), BcryptCost)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Email:        email,
		FirstName:    firstName,
		LastName:     lastName,
		Role:         role,
		PasswordHash: string(hashedPassword),
	}

	if err := s.userRepo.Create(user); err != nil {
		if errors.Is(err, repositories.ErrEmailAlreadyExists) {
			return nil, "", ErrEmailAlreadyExists
		}
		return nil, "", fmt.Errorf("failed to create customer: %w", err)
	}

	return user, tempPassword, nil
}

// UpdateCustomerProfile updates customer profile fields
func (s *CustomerProfileService) UpdateCustomerProfile(customerID uuid.UUID, updates map[string]interface{}) error {
	if customerID == uuid.Nil {
		return ErrInvalidCustomerID
	}

	if len(updates) == 0 {
		return errors.New("no updates provided")
	}

	_, err := s.userRepo.GetByIDActive(customerID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return ErrCustomerNotFound
		}
		return fmt.Errorf("failed to find customer: %w", err)
	}

	preventUpdatingSensitiveAndNonApplicableFields(updates)

	if err := s.userRepo.UpdateFields(customerID, updates); err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return ErrCustomerNotFound
		}
		return fmt.Errorf("failed to update customer profile: %w", err)
	}

	return nil
}

// UpdateCustomerEmail updates a customer's email address with uniqueness validation
func (s *CustomerProfileService) UpdateCustomerEmail(customerID uuid.UUID, newEmail string) error {
	if customerID == uuid.Nil {
		return ErrInvalidCustomerID
	}

	if newEmail == "" {
		return ErrInvalidEmail
	}

	existingUser, err := s.userRepo.GetByEmailExcluding(newEmail, customerID)
	if err != nil && !errors.Is(err, repositories.ErrUserNotFound) {
		return fmt.Errorf("failed to check email uniqueness: %w", err)
	}
	if existingUser != nil {
		return ErrEmailAlreadyExists
	}

	_, err = s.userRepo.GetByIDActive(customerID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return ErrCustomerNotFound
		}
		return fmt.Errorf("failed to find customer: %w", err)
	}

	if err := s.userRepo.UpdateEmail(customerID, newEmail); err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return ErrCustomerNotFound
		}
		if errors.Is(err, repositories.ErrEmailAlreadyExists) {
			return ErrEmailAlreadyExists
		}
		return fmt.Errorf("failed to update email: %w", err)
	}

	return nil
}

// DeleteCustomer soft deletes a customer with balance validation and cascade account deactivation
func (s *CustomerProfileService) DeleteCustomer(customerID uuid.UUID, reason string) error {
	if customerID == uuid.Nil {
		return ErrInvalidCustomerID
	}

	_, err := s.userRepo.GetByIDActive(customerID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return ErrCustomerNotFound
		}
		return fmt.Errorf("failed to find customer: %w", err)
	}

	totalBalance, err := s.accountRepo.GetTotalBalanceByUserID(customerID)
	if err != nil {
		return fmt.Errorf("failed to check customer balance: %w", err)
	}
	if !totalBalance.IsZero() {
		return ErrCustomerHasBalance
	}

	if err := s.userRepo.Delete(customerID); err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return ErrCustomerNotFound
		}
		return fmt.Errorf("failed to delete customer: %w", err)
	}

	if err := s.accountRepo.SoftDeleteByUserID(customerID); err != nil {
		return fmt.Errorf("failed to deactivate accounts: %w", err)
	}

	return nil
}

// GenerateTemporaryPassword generates a cryptographically secure random password
func GenerateTemporaryPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"

	password := make([]byte, length)
	for i := range password {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random character: %w", err)
		}
		password[i] = charset[num.Int64()]
	}

	return string(password), nil
}

func preventUpdatingSensitiveAndNonApplicableFields(updates map[string]interface{}) {
	delete(updates, "id")
	delete(updates, "password_hash")
	delete(updates, "email")
	delete(updates, "deleted_at")
	delete(updates, "role")
}
