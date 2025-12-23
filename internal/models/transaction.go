package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	TransactionTypeCredit = "credit"
	TransactionTypeDebit  = "debit"

	TransactionStatusPending   = "pending"
	TransactionStatusCompleted = "completed"
	TransactionStatusFailed    = "failed"
	TransactionStatusReversed  = "reversed"
)

var (
	ErrInvalidTransactionType   = errors.New("invalid transaction type")
	ErrInvalidTransactionStatus = errors.New("invalid transaction status")
	ErrInvalidAmount            = errors.New("transaction amount must be positive")
	ErrOptimisticLockConflict   = errors.New("optimistic lock conflict: version mismatch")
)

// Transaction represents a bank transaction
type Transaction struct {
	ID                uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	AccountID         uuid.UUID       `gorm:"type:uuid;not null;index" json:"account_id"`
	TransactionType   string          `gorm:"type:varchar(20);not null" json:"transaction_type"`
	Amount            decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"amount"`
	BalanceBefore     decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"balance_before"`
	BalanceAfter      decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"balance_after"`
	Description       string          `gorm:"type:text" json:"description"`
	Reference         string          `gorm:"type:varchar(100);index" json:"reference,omitempty"`
	Status            string          `gorm:"type:varchar(20);not null;default:'completed'" json:"status"`
	Metadata          JSONBMap        `gorm:"type:jsonb" json:"metadata,omitempty"`
	Category          string          `gorm:"type:varchar(50)" json:"category,omitempty"`
	MerchantName      string          `gorm:"type:varchar(255)" json:"merchant_name,omitempty"`
	MCCCode           string          `gorm:"type:varchar(10)" json:"mcc_code,omitempty"`
	PendingUntil      *time.Time      `json:"pending_until,omitempty"`
	ReversedAt        *time.Time      `json:"reversed_at,omitempty"`
	ReversalReference string          `gorm:"type:varchar(100)" json:"reversal_reference,omitempty"`
	ProcessingFee     decimal.Decimal `gorm:"type:decimal(15,2);default:0" json:"processing_fee"`
	Version           int             `gorm:"default:1" json:"version"`
	CreatedAt         time.Time       `gorm:"not null;index" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"not null" json:"updated_at"`
	ProcessedAt       *time.Time      `json:"processed_at,omitempty"`

	// Associations
	Account Account `gorm:"foreignKey:AccountID" json:"-"`
}

// BeforeCreate hook for Transaction
func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}

	now := time.Now()

	if t.Status == "" {
		t.Status = TransactionStatusCompleted
	}

	if t.Status == TransactionStatusCompleted {
		t.ProcessedAt = &now
	}

	if t.Reference == "" {
		t.Reference = GenerateTransactionReference()
	}

	// Set timestamps if not already set (for tests)
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	if t.UpdatedAt.IsZero() {
		t.UpdatedAt = now
	}

	return t.Validate()
}

// BeforeUpdate hook for Transaction
func (t *Transaction) BeforeUpdate(tx *gorm.DB) error {
	t.UpdatedAt = time.Now()
	t.incrementVersionForOptimisticLocking(tx)
	return t.Validate()
}

// Validate validates the transaction fields
func (t *Transaction) Validate() error {
	if t.AccountID == uuid.Nil {
		return errors.New("account ID is required")
	}

	if !IsValidTransactionType(t.TransactionType) {
		return ErrInvalidTransactionType
	}

	if !IsValidTransactionStatus(t.Status) {
		return ErrInvalidTransactionStatus
	}

	if t.Amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidAmount
	}

	if t.Description == "" {
		return errors.New("transaction description is required")
	}

	if t.Category != "" && len(t.Category) > 50 {
		return errors.New("category code too long")
	}

	if t.Status == TransactionStatusCompleted {
		if err := t.ensureBalanceIsCorrect(); err != nil {
			return err
		}
	}

	return nil
}

// IsCompleted returns true if the transaction is completed
func (t *Transaction) IsCompleted() bool {
	return t.Status == TransactionStatusCompleted
}

// IsPending returns true if the transaction is pending
func (t *Transaction) IsPending() bool {
	return t.Status == TransactionStatusPending
}

// Complete marks the transaction as completed
func (t *Transaction) Complete() {
	t.Status = TransactionStatusCompleted
	now := time.Now()
	t.ProcessedAt = &now
}

// Fail marks the transaction as failed
func (t *Transaction) Fail() {
	t.Status = TransactionStatusFailed
	now := time.Now()
	t.ProcessedAt = &now
}

// Reverse marks the transaction as reversed
func (t *Transaction) Reverse() {
	t.Status = TransactionStatusReversed
	now := time.Now()
	t.ProcessedAt = &now
	t.ReversedAt = &now
}

// TableName returns the table name for Transaction
func (t *Transaction) TableName() string {
	return "transactions"
}

// Helper functions

// IsValidTransactionType checks if the transaction type is valid
func IsValidTransactionType(transactionType string) bool {
	switch transactionType {
	case TransactionTypeCredit, TransactionTypeDebit:
		return true
	default:
		return false
	}
}

// IsValidTransactionStatus checks if the transaction status is valid
func IsValidTransactionStatus(status string) bool {
	switch status {
	case TransactionStatusPending, TransactionStatusCompleted, TransactionStatusFailed, TransactionStatusReversed:
		return true
	default:
		return false
	}
}

// GenerateTransactionReference generates a unique transaction reference
func GenerateTransactionReference() string {
	return "TXN-" + uuid.New().String()[:8] + "-" + time.Now().Format("20060102150405")
}

// Common transaction descriptions for sample data
var SampleTransactionDescriptions = []string{
	"Direct Deposit - Salary",
	"ACH Transfer - Payroll",
	"Mobile Deposit",
	"Wire Transfer",
	"ATM Withdrawal",
	"Debit Card Purchase - Grocery Store",
	"Debit Card Purchase - Gas Station",
	"Online Transfer",
	"Bill Payment - Utilities",
	"Bill Payment - Internet",
	"Check Deposit",
	"Interest Payment",
	"Monthly Service Fee",
	"Overdraft Fee",
	"Refund",
}

// CanTransitionTo checks if a transaction can transition to a new status
func (t *Transaction) CanTransitionTo(newStatus string) bool {
	validTransitions := map[string][]string{
		TransactionStatusPending:   {TransactionStatusCompleted, TransactionStatusFailed},
		TransactionStatusCompleted: {TransactionStatusReversed},
		TransactionStatusFailed:    {}, // Terminal state
		TransactionStatusReversed:  {}, // Terminal state
	}

	allowedStatuses, exists := validTransitions[t.Status]
	if !exists {
		return false
	}

	for _, status := range allowedStatuses {
		if status == newStatus {
			return true
		}
	}
	return false
}

// ReverseWithReference reverses a transaction with a reference
func (t *Transaction) ReverseWithReference(reversalRef string) {
	t.Reverse()
	t.ReversalReference = reversalRef
}

// IsPendingExpired checks if a pending transaction has expired
func (t *Transaction) IsPendingExpired() bool {
	if t.Status != TransactionStatusPending || t.PendingUntil == nil {
		return false
	}
	return time.Now().After(*t.PendingUntil)
}

// IncrementVersion increments the version for optimistic locking
func (t *Transaction) IncrementVersion() {
	t.Version++
}

// HasVersionConflict checks for version conflicts
func (t *Transaction) HasVersionConflict(currentVersion int) bool {
	return t.Version != currentVersion
}

// CheckAndUpdateVersion checks and updates version for optimistic locking
func (t *Transaction) CheckAndUpdateVersion(expectedVersion int) error {
	if t.Version != expectedVersion {
		return ErrOptimisticLockConflict
	}
	t.IncrementVersion()
	return nil
}

// GetTotalAmount returns the total amount including processing fees
func (t *Transaction) GetTotalAmount() decimal.Decimal {
	if t.ProcessingFee.IsZero() {
		return t.Amount
	}
	return t.Amount.Add(t.ProcessingFee)
}

// AutoCategorize assigns a category based on transaction description
func (t *Transaction) AutoCategorize() {
	categoryPatterns := map[string][]string{
		"INCOME":         {"Direct Deposit", "Salary", "Payroll"},
		"DINING":         {"Starbucks", "McDonald", "Restaurant", "Coffee"},
		"GROCERIES":      {"Walmart", "Kroger", "Whole Foods", "Safeway"},
		"TRANSPORTATION": {"Uber", "Lyft", "Gas", "Shell", "Chevron"},
		"ENTERTAINMENT":  {"Netflix", "Spotify", "Cinema", "Theater"},
		"ATM_CASH":       {"ATM", "Cash Withdrawal"},
		"FEES":           {"Service Fee", "Monthly Fee"},
	}

	for category, patterns := range categoryPatterns {
		for _, pattern := range patterns {
			if containsIgnoreCase(t.Description, pattern) {
				t.Category = category
				return
			}
		}
	}

	t.Category = "OTHER"
}

// ExtractMerchantFromDescription extracts merchant name from description
func (t *Transaction) ExtractMerchantFromDescription() {
	if idx := findSubstring(t.Description, " at "); idx >= 0 {
		start := idx + 4
		remaining := t.Description[start:]
		if dashIdx := findSubstring(remaining, " - "); dashIdx >= 0 {
			t.MerchantName = remaining[:dashIdx]
		} else {
			t.MerchantName = remaining
		}
	}
}

// Helper functions
func containsIgnoreCase(s, substr string) bool {
	return findSubstringIgnoreCase(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func findSubstringIgnoreCase(s, substr string) int {
	sLower := toLower(s)
	substrLower := toLower(substr)
	return findSubstring(sLower, substrLower)
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func (t *Transaction) incrementVersionForOptimisticLocking(tx *gorm.DB) {
	if tx != nil && tx.Statement != nil {
		tx.Statement.SetColumn("version", t.Version+1)
	}
}

func (t *Transaction) ensureBalanceIsCorrect() error {
	expectedBalance := t.BalanceBefore
	if t.TransactionType == TransactionTypeCredit {
		expectedBalance = expectedBalance.Add(t.Amount)
	} else {
		expectedBalance = expectedBalance.Sub(t.Amount)
		if !t.ProcessingFee.IsZero() {
			expectedBalance = expectedBalance.Sub(t.ProcessingFee)
		}
	}

	if !expectedBalance.Equal(t.BalanceAfter) {
		return errors.New("balance calculation mismatch")
	}
	return nil
}
