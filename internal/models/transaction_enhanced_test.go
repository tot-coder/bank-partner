package models

import (
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type TransactionEnhancedTestSuite struct {
	suite.Suite
}

func TestTransactionEnhancedSuite(t *testing.T) {
	suite.Run(t, new(TransactionEnhancedTestSuite))
}

func (s *TransactionEnhancedTestSuite) TestTransaction_CategoryAndMerchantFields() {
	accountID := uuid.New()

	testCases := []struct {
		name        string
		transaction Transaction
		expectValid bool
		description string
	}{
		{
			name: "transaction with category and merchant",
			transaction: Transaction{
				AccountID:       accountID,
				TransactionType: TransactionTypeDebit,
				Amount:          decimal.NewFromFloat(100),
				BalanceBefore:   decimal.NewFromFloat(1000),
				BalanceAfter:    decimal.NewFromFloat(900),
				Description:     "Purchase at " + gofakeit.Company(),
				Category:        "GROCERIES",
				MerchantName:    gofakeit.Company(),
				MCCCode:         "5411",
				Status:          TransactionStatusCompleted,
			},
			expectValid: true,
		},
		{
			name: "transaction without category (should be valid)",
			transaction: Transaction{
				AccountID:       accountID,
				TransactionType: TransactionTypeCredit,
				Amount:          decimal.NewFromFloat(2500),
				BalanceBefore:   decimal.NewFromFloat(1000),
				BalanceAfter:    decimal.NewFromFloat(3500),
				Description:     "Direct Deposit",
				Status:          TransactionStatusCompleted,
			},
			expectValid: true,
		},
		{
			name: "transaction with invalid category code",
			transaction: Transaction{
				AccountID:       accountID,
				TransactionType: TransactionTypeDebit,
				Amount:          decimal.NewFromFloat(50),
				BalanceBefore:   decimal.NewFromFloat(1000),
				BalanceAfter:    decimal.NewFromFloat(950),
				Description:     "ATM Withdrawal",
				Category:        "INVALID_CATEGORY_CODE_THAT_IS_TOO_LONG_TO_BE_VALID_EXCEEDS_FIFTY",
				Status:          TransactionStatusCompleted,
			},
			expectValid: false,
		},
		{
			name: "transaction with merchant but no category",
			transaction: Transaction{
				AccountID:       accountID,
				TransactionType: TransactionTypeDebit,
				Amount:          decimal.NewFromFloat(50),
				BalanceBefore:   decimal.NewFromFloat(500),
				BalanceAfter:    decimal.NewFromFloat(450),
				Description:     "Purchase",
				MerchantName:    gofakeit.Company() + " " + gofakeit.City(),
				MCCCode:         "5812",
				Status:          TransactionStatusCompleted,
			},
			expectValid: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.transaction.Validate()
			if tc.expectValid {
				s.NoError(err)
			} else {
				s.Error(err)
			}
		})
	}
}

func (s *TransactionEnhancedTestSuite) TestTransaction_StatusTransitions() {
	s.Run("valid status transitions", func() {
		testCases := []struct {
			from   string
			to     string
			valid  bool
			method string
		}{
			// Pending can transition to completed or failed
			{TransactionStatusPending, TransactionStatusCompleted, true, "Complete"},
			{TransactionStatusPending, TransactionStatusFailed, true, "Fail"},

			// Completed can be reversed
			{TransactionStatusCompleted, TransactionStatusReversed, true, "Reverse"},

			// Invalid transitions
			{TransactionStatusCompleted, TransactionStatusPending, false, ""},
			{TransactionStatusFailed, TransactionStatusCompleted, false, ""},
			{TransactionStatusReversed, TransactionStatusCompleted, false, ""},
			{TransactionStatusReversed, TransactionStatusPending, false, ""},
		}

		for _, tc := range testCases {
			txn := &Transaction{
				Status: tc.from,
			}

			canTransition := txn.CanTransitionTo(tc.to)
			s.Equal(tc.valid, canTransition, "transition from %s to %s", tc.from, tc.to)

			if tc.valid && tc.method != "" {
				switch tc.method {
				case "Complete":
					txn.Complete()
					s.Equal(TransactionStatusCompleted, txn.Status)
					s.NotNil(txn.ProcessedAt)
				case "Fail":
					txn.Fail()
					s.Equal(TransactionStatusFailed, txn.Status)
					s.NotNil(txn.ProcessedAt)
				case "Reverse":
					txn.Reverse()
					s.Equal(TransactionStatusReversed, txn.Status)
					s.NotNil(txn.ProcessedAt)
					s.NotNil(txn.ReversedAt)
				}
			}
		}
	})

	s.Run("transition with reversal reference", func() {
		txn := &Transaction{
			ID:        uuid.New(),
			Status:    TransactionStatusCompleted,
			Reference: GenerateTransactionReference(),
		}

		reversalRef := GenerateTransactionReference()
		txn.ReverseWithReference(reversalRef)

		s.Equal(TransactionStatusReversed, txn.Status)
		s.Equal(reversalRef, txn.ReversalReference)
		s.NotNil(txn.ReversedAt)
		s.NotNil(txn.ProcessedAt)
	})

	s.Run("pending transaction with timeout", func() {
		pendingUntil := time.Now().Add(24 * time.Hour)
		txn := &Transaction{
			Status:       TransactionStatusPending,
			PendingUntil: &pendingUntil,
		}

		s.False(txn.IsPendingExpired())

		expiredTime := time.Now().Add(-1 * time.Hour)
		txn.PendingUntil = &expiredTime
		s.True(txn.IsPendingExpired())
	})
}

func (s *TransactionEnhancedTestSuite) TestTransaction_OptimisticLocking() {
	accountID := uuid.New()

	s.Run("version increments on update", func() {
		txn := &Transaction{
			ID:              uuid.New(),
			AccountID:       accountID,
			TransactionType: TransactionTypeCredit,
			Amount:          decimal.NewFromFloat(100),
			BalanceBefore:   decimal.NewFromFloat(500),
			BalanceAfter:    decimal.NewFromFloat(600),
			Description:     "Test transaction",
			Status:          TransactionStatusPending,
			Version:         1,
		}

		// Simulate update
		txn.Complete()
		txn.IncrementVersion()

		s.Equal(2, txn.Version)
	})

	s.Run("concurrent update detection", func() {
		txn1 := &Transaction{
			ID:      uuid.New(),
			Version: 1,
		}

		txn2 := &Transaction{
			ID:      txn1.ID,
			Version: 1,
		}

		// Simulate txn1 being updated first
		txn1.IncrementVersion()
		s.Equal(2, txn1.Version)

		// txn2 should detect version mismatch
		s.True(txn2.HasVersionConflict(txn1.Version))
	})

	s.Run("check and update version", func() {
		txn := &Transaction{
			ID:      uuid.New(),
			Version: 5,
		}

		// Should fail with wrong expected version
		err := txn.CheckAndUpdateVersion(3)
		s.Error(err)
		s.Equal(ErrOptimisticLockConflict, err)
		s.Equal(5, txn.Version) // Version unchanged

		// Should succeed with correct expected version
		err = txn.CheckAndUpdateVersion(5)
		s.NoError(err)
		s.Equal(6, txn.Version) // Version incremented
	})
}

func (s *TransactionEnhancedTestSuite) TestTransaction_ProcessingFee() {
	s.Run("transaction with processing fee", func() {
		txn := &Transaction{
			AccountID:       uuid.New(),
			TransactionType: TransactionTypeDebit,
			Amount:          decimal.NewFromFloat(100),
			ProcessingFee:   decimal.NewFromFloat(2.50),
			BalanceBefore:   decimal.NewFromFloat(500),
			BalanceAfter:    decimal.NewFromFloat(397.50), // 500 - 100 - 2.50
			Description:     "ATM Withdrawal with fee",
			Status:          TransactionStatusCompleted,
		}

		totalAmount := txn.GetTotalAmount()
		s.True(totalAmount.Equal(decimal.NewFromFloat(102.50)))
	})
}

func (s *TransactionEnhancedTestSuite) TestTransaction_Categorization() {
	s.Run("auto categorize by description", func() {
		testCases := []struct {
			description      string
			expectedCategory string
		}{
			{"Purchase at Walmart", "GROCERIES"},
			{"Starbucks Coffee", "DINING"},
			{"Uber Trip", "TRANSPORTATION"},
			{"Netflix Subscription", "ENTERTAINMENT"},
			{"Direct Deposit - Salary", "INCOME"},
			{"ATM Withdrawal", "ATM_CASH"},
			{"Monthly Service Fee", "FEES"},
		}

		for _, tc := range testCases {
			txn := &Transaction{
				Description: tc.description,
			}

			txn.AutoCategorize()
			s.Equal(tc.expectedCategory, txn.Category, "description: %s", tc.description)
		}
	})

	s.Run("set merchant from description", func() {
		txn := &Transaction{
			Description: "Purchase at Whole Foods Market - San Francisco",
		}

		txn.ExtractMerchantFromDescription()
		s.Equal("Whole Foods Market", txn.MerchantName)
	})
}
