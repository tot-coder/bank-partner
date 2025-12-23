package models

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	AccountTypeChecking    = "checking"
	AccountTypeSavings     = "savings"
	AccountTypeMoneyMarket = "money_market"

	AccountStatusActive   = "active"
	AccountStatusInactive = "inactive"
	AccountStatusClosed   = "closed"

	// Account number prefixes by type
	CheckingPrefix    = "10"
	SavingsPrefix     = "20"
	MoneyMarketPrefix = "30"
)

var (
	ErrInvalidAccountType   = errors.New("invalid account type")
	ErrInvalidAccountStatus = errors.New("invalid account status")
	ErrInvalidBalance       = errors.New("balance cannot be negative")
	ErrAccountNotActive     = errors.New("account is not active")
	ErrInsufficientFunds    = errors.New("insufficient funds")
)

// Account represents a bank account
type Account struct {
	ID            uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	AccountNumber string          `gorm:"type:varchar(10);uniqueIndex;not null" json:"account_number"`
	UserID        uuid.UUID       `gorm:"type:uuid;not null;index" json:"user_id"`
	AccountType   string          `gorm:"type:varchar(20);not null" json:"account_type"`
	Balance       decimal.Decimal `gorm:"type:decimal(15,2);not null;default:0" json:"balance"`
	Status        string          `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	Currency      string          `gorm:"type:varchar(3);not null;default:'USD'" json:"currency"`
	InterestRate  decimal.Decimal `gorm:"type:decimal(5,4);default:0" json:"interest_rate,omitempty"`
	CreatedAt     time.Time       `gorm:"not null" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"not null" json:"updated_at"`
	ClosedAt      *time.Time      `gorm:"index" json:"closed_at,omitempty"`
	DeletedAt     gorm.DeletedAt  `gorm:"index" json:"deleted_at,omitempty"`

	// Associations
	User         User          `gorm:"foreignKey:UserID" json:"-"`
	Transactions []Transaction `gorm:"foreignKey:AccountID" json:"-"`
	AuditLogs    []AuditLog    `gorm:"foreignKey:ResourceID" json:"-"`
}

// BeforeCreate hook for Account
func (a *Account) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}

	// Set default status if not provided
	if a.Status == "" {
		a.Status = AccountStatusActive
	}

	// Set default currency if not provided
	if a.Currency == "" {
		a.Currency = "USD"
	}

	// Set timestamps if not already set (for tests)
	now := time.Now()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = now
	}

	if a.InterestRate.IsZero() {
		switch a.AccountType {
		case AccountTypeSavings:
			a.InterestRate = decimal.NewFromFloat(0.0150) // 1.50% APY
		case AccountTypeMoneyMarket:
			a.InterestRate = decimal.NewFromFloat(0.0250) // 2.50% APY
		default:
			a.InterestRate = decimal.Zero
		}
	}

	return a.Validate()
}

// BeforeUpdate hook for Account
func (a *Account) BeforeUpdate(tx *gorm.DB) error {
	a.UpdatedAt = time.Now()
	return a.Validate()
}

// Validate validates the account fields
func (a *Account) Validate() error {
	if a.UserID == uuid.Nil {
		return errors.New("user ID is required")
	}

	if a.AccountNumber == "" {
		return errors.New("account number is required")
	}

	if len(a.AccountNumber) != 10 {
		return errors.New("account number must be 10 digits")
	}

	if !IsValidAccountType(a.AccountType) {
		return ErrInvalidAccountType
	}

	if !IsValidAccountStatus(a.Status) {
		return ErrInvalidAccountStatus
	}

	if a.Balance.LessThan(decimal.Zero) {
		return ErrInvalidBalance
	}

	// Business rule: Account number prefix must match account type
	expectedPrefix := GetAccountPrefix(a.AccountType)
	if a.AccountNumber[:2] != expectedPrefix {
		return fmt.Errorf("account number prefix does not match account type")
	}

	return nil
}

// IsActive returns true if the account is active
func (a *Account) IsActive() bool {
	return a.Status == AccountStatusActive
}

// Close closes the account
func (a *Account) Close() error {
	if a.Status == AccountStatusClosed {
		return errors.New("account is already closed")
	}

	if !a.Balance.IsZero() {
		return errors.New("account balance must be zero to close")
	}

	a.Status = AccountStatusClosed
	now := time.Now()
	a.ClosedAt = &now
	return nil
}

// Deactivate deactivates the account
func (a *Account) Deactivate() error {
	if a.Status == AccountStatusClosed {
		return errors.New("cannot deactivate a closed account")
	}

	a.Status = AccountStatusInactive
	return nil
}

// Activate activates the account
func (a *Account) Activate() error {
	if a.Status == AccountStatusClosed {
		return errors.New("cannot activate a closed account")
	}

	a.Status = AccountStatusActive
	return nil
}

// CanWithdraw checks if the amount can be withdrawn
func (a *Account) CanWithdraw(amount decimal.Decimal) bool {
	return a.IsActive() && a.Balance.GreaterThanOrEqual(amount) && amount.GreaterThan(decimal.Zero)
}

// Debit debits the account
func (a *Account) Debit(amount decimal.Decimal) error {
	if !a.IsActive() {
		return ErrAccountNotActive
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		return errors.New("debit amount must be positive")
	}

	if a.Balance.LessThan(amount) {
		return ErrInsufficientFunds
	}

	a.Balance = a.Balance.Sub(amount)
	return nil
}

// Credit credits the account
func (a *Account) Credit(amount decimal.Decimal) error {
	if !a.IsActive() {
		return ErrAccountNotActive
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		return errors.New("credit amount must be positive")
	}

	a.Balance = a.Balance.Add(amount)
	return nil
}

// TableName returns the table name for Account
func (a *Account) TableName() string {
	return "accounts"
}

// Helper functions

// IsValidAccountType checks if the account type is valid
func IsValidAccountType(accountType string) bool {
	switch accountType {
	case AccountTypeChecking, AccountTypeSavings, AccountTypeMoneyMarket:
		return true
	default:
		return false
	}
}

// IsValidAccountStatus checks if the account status is valid
func IsValidAccountStatus(status string) bool {
	switch status {
	case AccountStatusActive, AccountStatusInactive, AccountStatusClosed:
		return true
	default:
		return false
	}
}

// GetAccountPrefix returns the prefix for an account type
func GetAccountPrefix(accountType string) string {
	switch accountType {
	case AccountTypeChecking:
		return CheckingPrefix
	case AccountTypeSavings:
		return SavingsPrefix
	case AccountTypeMoneyMarket:
		return MoneyMarketPrefix
	default:
		return ""
	}
}

// GenerateAccountNumber generates a unique 10-digit account number
func GenerateAccountNumber(accountType string) string {
	prefix := GetAccountPrefix(accountType)
	if prefix == "" {
		return ""
	}

	rand.Seed(time.Now().UnixNano())
	middle := fmt.Sprintf("%02d", rand.Intn(100))

	// In production, this would be from a database sequence
	suffix := fmt.Sprintf("%06d", rand.Intn(1000000))

	return prefix + middle + suffix
}

// CalculateChecksum calculates a checksum for account number validation
func CalculateChecksum(accountNumber string) int {
	if len(accountNumber) != 10 {
		return -1
	}

	sum := 0
	for i, char := range accountNumber {
		digit := int(char - '0')
		if i%2 == 0 {
			digit *= 2
			if digit > 9 {
				digit = digit/10 + digit%10
			}
		}
		sum += digit
	}

	return (10 - (sum % 10)) % 10
}

// ValidateAccountNumber validates an account number format
func ValidateAccountNumber(accountNumber string) bool {
	if len(accountNumber) != 10 {
		return false
	}

	for _, char := range accountNumber {
		if char < '0' || char > '9' {
			return false
		}
	}

	prefix := accountNumber[:2]
	if prefix != CheckingPrefix && prefix != SavingsPrefix && prefix != MoneyMarketPrefix {
		return false
	}

	return true
}
