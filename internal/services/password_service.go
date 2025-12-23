package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"

	"array-assessment/internal/repositories"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	// BCryptCost factor 12 required by PCI DSS v4.0 for financial data protection
	BCryptCost = 12

	MinPasswordLength = 12
	MaxPasswordLength = 72 // Bcrypt algorithm limitation
)

var (
	ErrPasswordEmpty        = errors.New("password cannot be empty")
	ErrPasswordTooShort     = fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	ErrPasswordTooLong      = fmt.Errorf("password must not exceed %d characters", MaxPasswordLength)
	ErrPasswordNoUppercase  = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLowercase  = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoNumber     = errors.New("password must contain at least one number")
	ErrPasswordNoSpecial    = errors.New("password must contain at least one special character")
	ErrCurrentPasswordWrong = errors.New("current password is incorrect")
	ErrSamePassword         = errors.New("new password must be different from current password")
	ErrInvalidAdminID       = errors.New("admin ID is required")

	uppercaseRegex = regexp.MustCompile(`[A-Z]`)
	lowercaseRegex = regexp.MustCompile(`[a-z]`)
	numberRegex    = regexp.MustCompile(`[0-9]`)
	specialRegex   = regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{}|;:,.<>?]`)
)

// PasswordService handles password hashing and validation
type PasswordService struct {
	cost         int
	userRepo     repositories.UserRepositoryInterface
	auditService AuditServiceInterface
}

// NewPasswordService creates a new password service with default settings
func NewPasswordService(userRepo repositories.UserRepositoryInterface, auditService AuditServiceInterface) PasswordServiceInterface {
	return &PasswordService{
		cost:         BCryptCost,
		userRepo:     userRepo,
		auditService: auditService,
	}
}

// ValidatePassword checks if a password meets all security requirements
func (ps *PasswordService) ValidatePassword(password string) error {
	if password == "" {
		return ErrPasswordEmpty
	}

	if len(password) < MinPasswordLength {
		return ErrPasswordTooShort
	}

	if len(password) > MaxPasswordLength {
		return ErrPasswordTooLong
	}

	if !uppercaseRegex.MatchString(password) {
		return ErrPasswordNoUppercase
	}

	if !lowercaseRegex.MatchString(password) {
		return ErrPasswordNoLowercase
	}

	if !numberRegex.MatchString(password) {
		return ErrPasswordNoNumber
	}

	if !specialRegex.MatchString(password) {
		return ErrPasswordNoSpecial
	}

	return nil
}

// HashPassword validates and hashes a password using bcrypt
func (ps *PasswordService) HashPassword(password string) (string, error) {
	if err := ps.ValidatePassword(password); err != nil {
		return "", fmt.Errorf("password validation failed: %w", err)
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), ps.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// ComparePassword compares a plain password with a hashed password
// Returns true if they match, false otherwise
func (ps *PasswordService) ComparePassword(password, hash string) bool {
	// bcrypt.CompareHashAndPassword provides timing-attack resistance per OWASP guidelines
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// HashPasswordWithoutValidation hashes a password without validation
// Used for system-generated passwords that bypass standard validation rules
func (ps *PasswordService) HashPasswordWithoutValidation(password string) (string, error) {
	if password == "" {
		return "", ErrPasswordEmpty
	}

	if len(password) > MaxPasswordLength {
		return "", ErrPasswordTooLong
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), ps.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// GenerateSecurePassword generates a cryptographically secure random password
func (ps *PasswordService) GenerateSecurePassword() (string, error) {
	return ps.GenerateSecurePasswordWithLength(16)
}

// GenerateSecurePasswordWithLength generates a cryptographically secure random password of specified length
func (ps *PasswordService) GenerateSecurePasswordWithLength(length int) (string, error) {
	if length < MinPasswordLength {
		length = MinPasswordLength
	}
	if length > MaxPasswordLength {
		length = MaxPasswordLength
	}

	const (
		uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowercase = "abcdefghijklmnopqrstuvwxyz"
		numbers   = "0123456789"
		special   = "!@#$%^&*()_+-=[]{}|;:,.<>?"
	)

	allChars := uppercase + lowercase + numbers + special

	result := make([]byte, length)
	requiredChars := []string{uppercase, lowercase, numbers, special}

	for i := 0; i < len(requiredChars); i++ {
		if i >= length {
			break
		}
		charSet := requiredChars[i]
		index, err := ps.secureRandomInt(len(charSet))
		if err != nil {
			return "", fmt.Errorf("failed to generate random index: %w", err)
		}
		result[i] = charSet[index]
	}

	for i := len(requiredChars); i < length; i++ {
		index, err := ps.secureRandomInt(len(allChars))
		if err != nil {
			return "", fmt.Errorf("failed to generate random index: %w", err)
		}
		result[i] = allChars[index]
	}

	if err := ps.secureShuffleBytes(result); err != nil {
		return "", fmt.Errorf("failed to shuffle password: %w", err)
	}

	return string(result), nil
}

func (ps *PasswordService) secureRandomInt(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}

func (ps *PasswordService) secureShuffleBytes(data []byte) error {
	for i := len(data) - 1; i > 0; i-- {
		j, err := ps.secureRandomInt(i + 1)
		if err != nil {
			return err
		}
		data[i], data[j] = data[j], data[i]
	}
	return nil
}

// PasswordStrength returns a score from 0-100 indicating password strength
func (ps *PasswordService) PasswordStrength(password string) int {
	if password == "" {
		return 0
	}

	score := ps.calculateLengthScore(len(password))
	score += ps.calculateCharacterDiversityScore(password)
	score += ps.calculateEntropyBonus(password)

	if ps.ValidatePassword(password) == nil && score < 80 {
		// Meets all requirements per PCI DSS, ensure minimum acceptable score
		score = 80
	}

	if score > 100 {
		score = 100
	}

	return score
}

func (ps *PasswordService) calculateLengthScore(length int) int {
	score := 0
	if length >= 8 {
		score += 10
	}
	if length >= 12 {
		score += 10
	}
	if length >= 16 {
		score += 10
	}
	if length >= 20 {
		score += 10
	}
	return score
}

func (ps *PasswordService) calculateCharacterDiversityScore(password string) int {
	score := 0
	if uppercaseRegex.MatchString(password) {
		score += 15
	}
	if lowercaseRegex.MatchString(password) {
		score += 15
	}
	if numberRegex.MatchString(password) {
		score += 15
	}
	if specialRegex.MatchString(password) {
		score += 15
	}
	return score
}

func (ps *PasswordService) calculateEntropyBonus(password string) int {
	uniqueChars := make(map[rune]bool)
	for _, char := range password {
		uniqueChars[char] = true
	}

	// Entropy bonus for character variety
	if len(uniqueChars) > len(password)*3/4 {
		return 10
	}
	if len(uniqueChars) > len(password)/2 {
		return 5
	}
	return 0
}

// AdminResetPassword resets a customer's password (admin operation)
// Returns the temporary password that should be sent to the customer
func (ps *PasswordService) AdminResetPassword(customerID, adminID uuid.UUID) (string, error) {
	if ps.userRepo == nil {
		return "", errors.New("user repository not configured for customer operations")
	}

	if customerID == uuid.Nil {
		return "", ErrInvalidCustomerID
	}

	if adminID == uuid.Nil {
		return "", ErrInvalidAdminID
	}

	user, err := ps.userRepo.GetByIDActive(customerID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return "", ErrCustomerNotFound
		}
		return "", fmt.Errorf("failed to find customer: %w", err)
	}

	tempPassword, err := ps.GenerateSecurePassword()
	if err != nil {
		return "", fmt.Errorf("failed to generate temporary password: %w", err)
	}

	hashedPassword, err := ps.HashPasswordWithoutValidation(tempPassword)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	if err := ps.userRepo.UpdatePasswordHash(user.ID, hashedPassword); err != nil {
		return "", fmt.Errorf("failed to update password: %w", err)
	}

	return tempPassword, nil
}

// CustomerUpdatePassword allows a customer to update their own password
func (ps *PasswordService) CustomerUpdatePassword(customerID uuid.UUID, currentPassword, newPassword string) error {
	if ps.userRepo == nil {
		return errors.New("user repository not configured for customer operations")
	}

	if customerID == uuid.Nil {
		return ErrInvalidCustomerID
	}

	if currentPassword == "" {
		return errors.New("current password is required")
	}

	if newPassword == "" {
		return errors.New("new password is required")
	}

	if currentPassword == newPassword {
		return ErrSamePassword
	}

	if err := ps.ValidatePassword(newPassword); err != nil {
		return err
	}

	user, err := ps.userRepo.GetByIDActive(customerID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return ErrCustomerNotFound
		}
		return fmt.Errorf("failed to find customer: %w", err)
	}

	if !ps.ComparePassword(currentPassword, user.PasswordHash) {
		return ErrCurrentPasswordWrong
	}

	hashedPassword, err := ps.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	if err := ps.userRepo.UpdatePasswordHash(user.ID, hashedPassword); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}
