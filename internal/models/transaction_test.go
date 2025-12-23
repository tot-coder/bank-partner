package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransaction_Validate(t *testing.T) {
	validAccountID := uuid.New()

	tests := []struct {
		name        string
		transaction Transaction
		wantErr     bool
		errMsg      string
	}{
		{
			name: "valid credit transaction",
			transaction: Transaction{
				AccountID:       validAccountID,
				TransactionType: TransactionTypeCredit,
				Amount:          decimal.NewFromFloat(100.00),
				BalanceBefore:   decimal.NewFromFloat(500.00),
				BalanceAfter:    decimal.NewFromFloat(600.00),
				Description:     "Direct Deposit",
				Status:          TransactionStatusCompleted,
			},
			wantErr: false,
		},
		{
			name: "valid debit transaction",
			transaction: Transaction{
				AccountID:       validAccountID,
				TransactionType: TransactionTypeDebit,
				Amount:          decimal.NewFromFloat(50.00),
				BalanceBefore:   decimal.NewFromFloat(500.00),
				BalanceAfter:    decimal.NewFromFloat(450.00),
				Description:     "ATM Withdrawal",
				Status:          TransactionStatusCompleted,
			},
			wantErr: false,
		},
		{
			name: "valid pending transaction",
			transaction: Transaction{
				AccountID:       validAccountID,
				TransactionType: TransactionTypeCredit,
				Amount:          decimal.NewFromFloat(100.00),
				BalanceBefore:   decimal.Zero,
				BalanceAfter:    decimal.Zero,
				Description:     "Pending Transfer",
				Status:          TransactionStatusPending,
			},
			wantErr: false,
		},
		{
			name: "missing account ID",
			transaction: Transaction{
				TransactionType: TransactionTypeCredit,
				Amount:          decimal.NewFromFloat(100.00),
				Description:     "Test Transaction",
				Status:          TransactionStatusCompleted,
			},
			wantErr: true,
			errMsg:  "account ID is required",
		},
		{
			name: "invalid transaction type",
			transaction: Transaction{
				AccountID:       validAccountID,
				TransactionType: "invalid",
				Amount:          decimal.NewFromFloat(100.00),
				Description:     "Test Transaction",
				Status:          TransactionStatusCompleted,
			},
			wantErr: true,
			errMsg:  "invalid transaction type",
		},
		{
			name: "invalid transaction status",
			transaction: Transaction{
				AccountID:       validAccountID,
				TransactionType: TransactionTypeCredit,
				Amount:          decimal.NewFromFloat(100.00),
				Description:     "Test Transaction",
				Status:          "invalid",
			},
			wantErr: true,
			errMsg:  "invalid transaction status",
		},
		{
			name: "negative amount",
			transaction: Transaction{
				AccountID:       validAccountID,
				TransactionType: TransactionTypeCredit,
				Amount:          decimal.NewFromFloat(-100.00),
				Description:     "Test Transaction",
				Status:          TransactionStatusCompleted,
			},
			wantErr: true,
			errMsg:  "transaction amount must be positive",
		},
		{
			name: "zero amount",
			transaction: Transaction{
				AccountID:       validAccountID,
				TransactionType: TransactionTypeCredit,
				Amount:          decimal.Zero,
				Description:     "Test Transaction",
				Status:          TransactionStatusCompleted,
			},
			wantErr: true,
			errMsg:  "transaction amount must be positive",
		},
		{
			name: "missing description",
			transaction: Transaction{
				AccountID:       validAccountID,
				TransactionType: TransactionTypeCredit,
				Amount:          decimal.NewFromFloat(100.00),
				Description:     "",
				Status:          TransactionStatusCompleted,
			},
			wantErr: true,
			errMsg:  "transaction description is required",
		},
		{
			name: "balance mismatch for credit",
			transaction: Transaction{
				AccountID:       validAccountID,
				TransactionType: TransactionTypeCredit,
				Amount:          decimal.NewFromFloat(100.00),
				BalanceBefore:   decimal.NewFromFloat(500.00),
				BalanceAfter:    decimal.NewFromFloat(550.00), // Should be 600
				Description:     "Test Transaction",
				Status:          TransactionStatusCompleted,
			},
			wantErr: true,
			errMsg:  "balance calculation mismatch",
		},
		{
			name: "balance mismatch for debit",
			transaction: Transaction{
				AccountID:       validAccountID,
				TransactionType: TransactionTypeDebit,
				Amount:          decimal.NewFromFloat(100.00),
				BalanceBefore:   decimal.NewFromFloat(500.00),
				BalanceAfter:    decimal.NewFromFloat(450.00), // Should be 400
				Description:     "Test Transaction",
				Status:          TransactionStatusCompleted,
			},
			wantErr: true,
			errMsg:  "balance calculation mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.transaction.Validate()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTransaction_StatusMethods(t *testing.T) {
	t.Run("IsCompleted", func(t *testing.T) {
		tests := []struct {
			status   string
			expected bool
		}{
			{TransactionStatusCompleted, true},
			{TransactionStatusPending, false},
			{TransactionStatusFailed, false},
			{TransactionStatusReversed, false},
		}

		for _, tt := range tests {
			txn := Transaction{Status: tt.status}
			assert.Equal(t, tt.expected, txn.IsCompleted())
		}
	})

	t.Run("IsPending", func(t *testing.T) {
		tests := []struct {
			status   string
			expected bool
		}{
			{TransactionStatusPending, true},
			{TransactionStatusCompleted, false},
			{TransactionStatusFailed, false},
			{TransactionStatusReversed, false},
		}

		for _, tt := range tests {
			txn := Transaction{Status: tt.status}
			assert.Equal(t, tt.expected, txn.IsPending())
		}
	})
}

func TestTransaction_Complete(t *testing.T) {
	txn := Transaction{
		Status: TransactionStatusPending,
	}

	txn.Complete()

	assert.Equal(t, TransactionStatusCompleted, txn.Status)
	assert.NotNil(t, txn.ProcessedAt)
	assert.True(t, time.Now().Sub(*txn.ProcessedAt) < time.Second)
}

func TestTransaction_Fail(t *testing.T) {
	txn := Transaction{
		Status: TransactionStatusPending,
	}

	txn.Fail()

	assert.Equal(t, TransactionStatusFailed, txn.Status)
	assert.NotNil(t, txn.ProcessedAt)
	assert.True(t, time.Now().Sub(*txn.ProcessedAt) < time.Second)
}

func TestTransaction_Reverse(t *testing.T) {
	txn := Transaction{
		Status: TransactionStatusCompleted,
	}

	txn.Reverse()

	assert.Equal(t, TransactionStatusReversed, txn.Status)
	assert.NotNil(t, txn.ProcessedAt)
	assert.True(t, time.Now().Sub(*txn.ProcessedAt) < time.Second)
}

func TestIsValidTransactionType(t *testing.T) {
	tests := []struct {
		transactionType string
		expected        bool
	}{
		{TransactionTypeCredit, true},
		{TransactionTypeDebit, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.transactionType, func(t *testing.T) {
			result := IsValidTransactionType(tt.transactionType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidTransactionStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{TransactionStatusPending, true},
		{TransactionStatusCompleted, true},
		{TransactionStatusFailed, true},
		{TransactionStatusReversed, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := IsValidTransactionStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateTransactionReference(t *testing.T) {
	// Generate multiple references to ensure they're unique
	refs := make(map[string]bool)
	for i := 0; i < 10; i++ {
		ref := GenerateTransactionReference()
		assert.NotEmpty(t, ref)
		assert.True(t, len(ref) > 10)
		assert.Contains(t, ref, "TXN-")
		
		// Check uniqueness
		assert.False(t, refs[ref], "Duplicate reference generated")
		refs[ref] = true
		
		// Small delay to ensure timestamp differences
		time.Sleep(time.Millisecond)
	}
}

func TestTransaction_BeforeCreate(t *testing.T) {
	txn := Transaction{
		AccountID:       uuid.New(),
		TransactionType: TransactionTypeCredit,
		Amount:          decimal.NewFromFloat(100.00),
		BalanceBefore:   decimal.NewFromFloat(500.00),
		BalanceAfter:    decimal.NewFromFloat(600.00),
		Description:     "Test Transaction",
	}

	// Simulate BeforeCreate hook
	err := txn.BeforeCreate(nil)
	require.NoError(t, err)

	// Check defaults were set
	assert.NotEqual(t, uuid.Nil, txn.ID)
	assert.Equal(t, TransactionStatusCompleted, txn.Status)
	assert.NotEmpty(t, txn.Reference)
	assert.NotNil(t, txn.ProcessedAt)
	assert.NotZero(t, txn.CreatedAt)
	assert.NotZero(t, txn.UpdatedAt)
}

func TestTransaction_BeforeUpdate(t *testing.T) {
	txn := Transaction{
		AccountID:       uuid.New(),
		TransactionType: TransactionTypeCredit,
		Amount:          decimal.NewFromFloat(100.00),
		BalanceBefore:   decimal.NewFromFloat(500.00),
		BalanceAfter:    decimal.NewFromFloat(600.00),
		Description:     "Test Transaction",
		Status:          TransactionStatusCompleted,
		UpdatedAt:       time.Now().Add(-1 * time.Hour),
	}

	originalUpdatedAt := txn.UpdatedAt

	// Simulate BeforeUpdate hook
	err := txn.BeforeUpdate(nil)
	require.NoError(t, err)

	// Check UpdatedAt was updated
	assert.True(t, txn.UpdatedAt.After(originalUpdatedAt))
}

func TestSampleTransactionDescriptions(t *testing.T) {
	// Ensure we have sample descriptions
	assert.NotEmpty(t, SampleTransactionDescriptions)
	assert.Greater(t, len(SampleTransactionDescriptions), 10)

	// Check that descriptions are not empty
	for _, desc := range SampleTransactionDescriptions {
		assert.NotEmpty(t, desc)
	}
}