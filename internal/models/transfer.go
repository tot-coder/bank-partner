package models

import (
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	TransferStatusPending   = "pending"
	TransferStatusCompleted = "completed"
	TransferStatusFailed    = "failed"
)

var (
	ErrInvalidTransferStatus = errors.New("invalid transfer status")
	ErrInvalidTransferAmount = errors.New("transfer amount must be positive")
)

// Transfer represents an account-to-account transfer
type Transfer struct {
	ID                  uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	FromAccountID       uuid.UUID       `gorm:"type:uuid;not null;index:idx_transfer_from_account" json:"from_account_id"`
	ToAccountID         uuid.UUID       `gorm:"type:uuid;not null;index:idx_transfer_to_account" json:"to_account_id"`
	Amount              decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"amount"`
	Description         string          `gorm:"type:text;not null" json:"description"`
	IdempotencyKey      string          `gorm:"type:varchar(255);uniqueIndex;not null" json:"idempotency_key"`
	Status              string          `gorm:"type:varchar(20);not null;default:'pending';index:idx_transfer_status" json:"status"`
	DebitTransactionID  *uuid.UUID      `gorm:"type:uuid;index" json:"debit_transaction_id,omitempty"`
	CreditTransactionID *uuid.UUID      `gorm:"type:uuid;index" json:"credit_transaction_id,omitempty"`
	ErrorMessage        *string         `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt           time.Time       `gorm:"not null;index:idx_transfer_created_at" json:"created_at"`
	UpdatedAt           time.Time       `gorm:"not null" json:"updated_at"`
	CompletedAt         *time.Time      `json:"completed_at,omitempty"`
	FailedAt            *time.Time      `json:"failed_at,omitempty"`

	// Associations
	FromAccount       Account      `gorm:"foreignKey:FromAccountID" json:"-"`
	ToAccount         Account      `gorm:"foreignKey:ToAccountID" json:"-"`
	DebitTransaction  *Transaction `gorm:"foreignKey:DebitTransactionID" json:"-"`
	CreditTransaction *Transaction `gorm:"foreignKey:CreditTransactionID" json:"-"`
}

// BeforeCreate hook for Transfer
func (t *Transfer) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}

	if t.Status == "" {
		t.Status = TransferStatusPending
	}

	now := time.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	if t.UpdatedAt.IsZero() {
		t.UpdatedAt = now
	}

	return t.Validate()
}

// BeforeUpdate hook for Transfer
func (t *Transfer) BeforeUpdate(tx *gorm.DB) error {
	t.UpdatedAt = time.Now()
	return t.Validate()
}

// Validate validates the transfer fields
func (t *Transfer) Validate() error {
	if t.FromAccountID == uuid.Nil {
		return errors.New("from account ID is required")
	}

	if t.ToAccountID == uuid.Nil {
		return errors.New("to account ID is required")
	}

	if t.FromAccountID == t.ToAccountID {
		return errors.New("from and to accounts cannot be the same")
	}

	if t.Amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidTransferAmount
	}

	if t.Description == "" {
		return errors.New("description is required")
	}

	if t.IdempotencyKey == "" {
		return errors.New("idempotency key is required")
	}

	if !IsValidTransferStatus(t.Status) {
		return ErrInvalidTransferStatus
	}

	return nil
}

// IsPending returns true if the transfer is pending
func (t *Transfer) IsPending() bool {
	return t.Status == TransferStatusPending
}

// IsCompleted returns true if the transfer is completed
func (t *Transfer) IsCompleted() bool {
	return t.Status == TransferStatusCompleted
}

// IsFailed returns true if the transfer is failed
func (t *Transfer) IsFailed() bool {
	return t.Status == TransferStatusFailed
}

// Complete marks the transfer as completed and links transaction IDs
func (t *Transfer) Complete(debitTxID, creditTxID uuid.UUID) {
	t.Status = TransferStatusCompleted
	now := time.Now()
	t.CompletedAt = &now
	t.DebitTransactionID = &debitTxID
	t.CreditTransactionID = &creditTxID
}

// Fail marks the transfer as failed with an error message
func (t *Transfer) Fail(errorMessage string) {
	t.Status = TransferStatusFailed
	now := time.Now()
	t.FailedAt = &now
	t.ErrorMessage = &errorMessage
}

// CanTransitionTo checks if a transfer can transition to a new status
func (t *Transfer) CanTransitionTo(newStatus string) bool {
	validTransitions := map[string][]string{
		TransferStatusPending:   {TransferStatusCompleted, TransferStatusFailed},
		TransferStatusCompleted: {},
		TransferStatusFailed:    {},
	}

	allowedStatuses, exists := validTransitions[t.Status]
	if !exists {
		return false
	}

	return slices.Contains(allowedStatuses, newStatus)
}

// TableName returns the table name for Transfer
func (t *Transfer) TableName() string {
	return "transfers"
}

// Helper functions

// IsValidTransferStatus checks if the transfer status is valid
func IsValidTransferStatus(status string) bool {
	switch status {
	case TransferStatusPending, TransferStatusCompleted, TransferStatusFailed:
		return true
	default:
		return false
	}
}
