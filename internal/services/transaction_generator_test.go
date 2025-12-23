package services

import (
	"testing"
	"time"

	"array-assessment/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type TransactionGeneratorTestSuite struct {
	suite.Suite
	generator *transactionGenerator
	accountID uuid.UUID
}

func TestTransactionGeneratorSuite(t *testing.T) {
	suite.Run(t, new(TransactionGeneratorTestSuite))
}

func (s *TransactionGeneratorTestSuite) SetupTest() {
	s.generator = NewTransactionGenerator().(*transactionGenerator)
	s.accountID = uuid.New()
}

// Merchant Pool Tests

func (s *TransactionGeneratorTestSuite) TestMerchantPool_HasMinimum50Merchants() {
	merchants := s.generator.GetMerchantPool()
	s.GreaterOrEqual(len(merchants), 50, "Merchant pool should have at least 50 merchants")
}

func (s *TransactionGeneratorTestSuite) TestMerchantPool_ContainsVariety() {
	merchants := s.generator.GetMerchantPool()

	categories := make(map[string]bool)
	for _, merchant := range merchants {
		categories[merchant.Category] = true
	}

	s.GreaterOrEqual(len(categories), 10, "Merchant pool should contain at least 10 different categories")
}

func (s *TransactionGeneratorTestSuite) TestMerchantPool_HasValidMCCCodes() {
	merchants := s.generator.GetMerchantPool()

	for _, merchant := range merchants {
		s.Len(merchant.MCCCode, 4, "MCC code should be 4 digits for merchant: %s", merchant.Name)
		s.NotEmpty(merchant.Name)
		s.NotEmpty(merchant.Category)
	}
}

func (s *TransactionGeneratorTestSuite) TestSelectRandomMerchant_ReturnsValidMerchant() {
	for i := 0; i < 100; i++ {
		merchant := s.generator.SelectRandomMerchant()
		s.NotEmpty(merchant.Name)
		s.NotEmpty(merchant.Category)
		s.Len(merchant.MCCCode, 4)
	}
}

// Transaction Type Distribution Tests

func (s *TransactionGeneratorTestSuite) TestGenerateTransactionType_Distribution() {
	counts := map[string]int{
		models.TransactionTypeDebit:  0,
		models.TransactionTypeCredit: 0,
		"fee":                        0,
	}

	iterations := 1000
	for i := 0; i < iterations; i++ {
		txnType, isFee := s.generator.GenerateTransactionType()
		if isFee {
			counts["fee"]++
		} else {
			counts[txnType]++
		}
	}

	debitRatio := float64(counts[models.TransactionTypeDebit]) / float64(iterations)
	creditRatio := float64(counts[models.TransactionTypeCredit]) / float64(iterations)
	feeRatio := float64(counts["fee"]) / float64(iterations)

	s.InDelta(0.60, debitRatio, 0.10, "Debit ratio should be approximately 60%")
	s.InDelta(0.35, creditRatio, 0.10, "Credit ratio should be approximately 35%")
	s.InDelta(0.05, feeRatio, 0.05, "Fee ratio should be approximately 5%")
}

func (s *TransactionGeneratorTestSuite) TestGenerateTransactionType_ValidTypes() {
	for i := 0; i < 100; i++ {
		txnType, _ := s.generator.GenerateTransactionType()
		s.True(models.IsValidTransactionType(txnType), "Generated type should be valid")
	}
}

// Amount Generation Tests

func (s *TransactionGeneratorTestSuite) TestGenerateAmount_ValidRange() {
	for i := 0; i < 100; i++ {
		amount := s.generator.GenerateAmount(models.CategoryGroceries)
		s.True(amount.GreaterThan(decimal.Zero), "Amount should be positive")
		s.True(amount.LessThan(decimal.NewFromInt(10000)), "Amount should be reasonable")
	}
}

func (s *TransactionGeneratorTestSuite) TestGenerateAmount_CategoryBasedRanges() {
	testCases := []struct {
		category    string
		minExpected decimal.Decimal
		maxExpected decimal.Decimal
	}{
		{models.CategoryGroceries, decimal.NewFromInt(10), decimal.NewFromInt(300)},
		{models.CategoryDining, decimal.NewFromInt(5), decimal.NewFromInt(150)},
		{models.CategoryTransportation, decimal.NewFromInt(5), decimal.NewFromInt(100)},
		{models.CategoryShopping, decimal.NewFromInt(20), decimal.NewFromInt(500)},
		{models.CategoryBillsUtilities, decimal.NewFromInt(50), decimal.NewFromInt(300)},
		{models.CategoryIncome, decimal.NewFromInt(1000), decimal.NewFromInt(10000)},
	}

	for _, tc := range testCases {
		s.Run(tc.category, func() {
			amount := s.generator.GenerateAmount(tc.category)
			s.True(amount.GreaterThanOrEqual(tc.minExpected.Mul(decimal.NewFromFloat(0.5))),
				"Amount for %s should be at least half of min range", tc.category)
			s.True(amount.LessThanOrEqual(tc.maxExpected.Mul(decimal.NewFromFloat(1.5))),
				"Amount for %s should be at most 1.5x max range", tc.category)
		})
	}
}

func (s *TransactionGeneratorTestSuite) TestGenerateFeeAmount_SmallValues() {
	for i := 0; i < 50; i++ {
		amount := s.generator.GenerateFeeAmount()
		s.True(amount.GreaterThan(decimal.Zero), "Fee should be positive")
		s.True(amount.LessThan(decimal.NewFromInt(50)), "Fee should be small")
	}
}

// Temporal Pattern Tests

func (s *TransactionGeneratorTestSuite) TestGenerateTimestamp_WithinDateRange() {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	for i := 0; i < 100; i++ {
		timestamp := s.generator.GenerateTimestamp(startDate, endDate)
		s.True(timestamp.After(startDate) || timestamp.Equal(startDate),
			"Timestamp should be after or equal to start date")
		s.True(timestamp.Before(endDate) || timestamp.Equal(endDate),
			"Timestamp should be before or equal to end date")
	}
}

func (s *TransactionGeneratorTestSuite) TestGenerateSalaryTransaction_BiWeeklyPattern() {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 3, 31, 23, 59, 59, 0, time.UTC)
	startingBalance := decimal.NewFromInt(1000)

	transactions := s.generator.GenerateSalaryTransactions(s.accountID, startDate, endDate, startingBalance)

	s.NotEmpty(transactions, "Should generate at least one salary transaction")

	for i := 0; i < len(transactions)-1; i++ {
		daysBetween := transactions[i+1].CreatedAt.Sub(transactions[i].CreatedAt).Hours() / 24
		s.InDelta(14, daysBetween, 2, "Salary transactions should be approximately bi-weekly")
	}

	for _, txn := range transactions {
		s.Equal(models.TransactionTypeCredit, txn.TransactionType)
		s.Equal(models.CategoryIncome, txn.Category)
		s.True(txn.Amount.GreaterThan(decimal.NewFromInt(1000)), "Salary should be substantial")
		s.Contains(txn.Description, "Salary")
	}
}

func (s *TransactionGeneratorTestSuite) TestGenerateBillTransactions_MonthlyPattern() {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC)
	startingBalance := decimal.NewFromInt(5000)

	transactions := s.generator.GenerateBillTransactions(s.accountID, startDate, endDate, startingBalance)

	s.NotEmpty(transactions, "Should generate bill transactions")

	months := make(map[time.Month]int)
	for _, txn := range transactions {
		months[txn.CreatedAt.Month()]++
		s.Equal(models.TransactionTypeDebit, txn.TransactionType)
		s.Equal(models.CategoryBillsUtilities, txn.Category)
	}

	s.GreaterOrEqual(len(months), 5, "Bills should span multiple months")
}

// Historical Transaction Generation Tests

func (s *TransactionGeneratorTestSuite) TestGenerateHistoricalTransactions_ChronologicalOrder() {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	startingBalance := decimal.NewFromInt(5000)
	count := 100

	transactions := s.generator.GenerateHistoricalTransactions(s.accountID, startDate, endDate, startingBalance, count)

	s.Equal(count, len(transactions), "Should generate exact count requested")

	for i := 0; i < len(transactions)-1; i++ {
		s.True(transactions[i].CreatedAt.Before(transactions[i+1].CreatedAt) ||
			transactions[i].CreatedAt.Equal(transactions[i+1].CreatedAt),
			"Transactions should be in chronological order")
	}
}

func (s *TransactionGeneratorTestSuite) TestGenerateHistoricalTransactions_BalanceProgression() {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	startingBalance := decimal.NewFromInt(5000)
	count := 50

	transactions := s.generator.GenerateHistoricalTransactions(s.accountID, startDate, endDate, startingBalance, count)

	currentBalance := startingBalance
	for _, txn := range transactions {
		s.True(txn.BalanceBefore.Equal(currentBalance),
			"Transaction BalanceBefore should match running balance")

		if txn.TransactionType == models.TransactionTypeCredit {
			currentBalance = currentBalance.Add(txn.Amount)
		} else {
			currentBalance = currentBalance.Sub(txn.Amount)
			if !txn.ProcessingFee.IsZero() {
				currentBalance = currentBalance.Sub(txn.ProcessingFee)
			}
		}

		s.True(txn.BalanceAfter.Equal(currentBalance),
			"Transaction BalanceAfter should match calculated balance")
	}
}

func (s *TransactionGeneratorTestSuite) TestGenerateHistoricalTransactions_MaintainsPositiveBalance() {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 3, 31, 23, 59, 59, 0, time.UTC)
	startingBalance := decimal.NewFromInt(1000)
	count := 200

	transactions := s.generator.GenerateHistoricalTransactions(s.accountID, startDate, endDate, startingBalance, count)

	for _, txn := range transactions {
		s.True(txn.BalanceAfter.GreaterThanOrEqual(decimal.Zero),
			"Balance should never go negative")
	}
}

func (s *TransactionGeneratorTestSuite) TestGenerateHistoricalTransactions_ValidStatuses() {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	startingBalance := decimal.NewFromInt(5000)
	count := 50

	transactions := s.generator.GenerateHistoricalTransactions(s.accountID, startDate, endDate, startingBalance, count)

	for _, txn := range transactions {
		s.True(models.IsValidTransactionStatus(txn.Status), "Status should be valid")
		s.Equal(models.TransactionStatusCompleted, txn.Status, "Historical transactions should be completed")
		s.NotNil(txn.ProcessedAt, "Completed transactions should have ProcessedAt")
	}
}

func (s *TransactionGeneratorTestSuite) TestGenerateHistoricalTransactions_ValidReferences() {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	startingBalance := decimal.NewFromInt(5000)
	count := 50

	transactions := s.generator.GenerateHistoricalTransactions(s.accountID, startDate, endDate, startingBalance, count)

	references := make(map[string]bool)
	for _, txn := range transactions {
		s.NotEmpty(txn.Reference, "Transaction should have reference")
		s.False(references[txn.Reference], "References should be unique")
		references[txn.Reference] = true
	}
}

func (s *TransactionGeneratorTestSuite) TestGenerateHistoricalTransactions_CategorizedCorrectly() {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	startingBalance := decimal.NewFromInt(5000)
	count := 100

	transactions := s.generator.GenerateHistoricalTransactions(s.accountID, startDate, endDate, startingBalance, count)

	categories := make(map[string]int)
	for _, txn := range transactions {
		s.NotEmpty(txn.Category, "Transaction should have category")
		s.True(models.IsValidCategory(txn.Category), "Category should be valid")
		categories[txn.Category]++
	}

	s.GreaterOrEqual(len(categories), 5, "Should have variety of categories")
}

func (s *TransactionGeneratorTestSuite) TestGenerateHistoricalTransactions_HasMerchantInfo() {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	startingBalance := decimal.NewFromInt(5000)
	count := 50

	transactions := s.generator.GenerateHistoricalTransactions(s.accountID, startDate, endDate, startingBalance, count)

	withMerchant := 0
	withMCC := 0

	for _, txn := range transactions {
		if txn.MerchantName != "" {
			withMerchant++
			s.NotEmpty(txn.Description, "Transaction with merchant should have description")
		}
		if txn.MCCCode != "" {
			withMCC++
			s.Len(txn.MCCCode, 4, "MCC code should be 4 digits")
		}
	}

	s.Greater(withMerchant, count/2, "Most transactions should have merchant names")
	s.Greater(withMCC, count/2, "Most transactions should have MCC codes")
}

// Edge Case Tests

func (s *TransactionGeneratorTestSuite) TestGenerateHistoricalTransactions_ZeroCount() {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	startingBalance := decimal.NewFromInt(5000)

	transactions := s.generator.GenerateHistoricalTransactions(s.accountID, startDate, endDate, startingBalance, 0)

	s.Empty(transactions, "Zero count should return empty slice")
}

func (s *TransactionGeneratorTestSuite) TestGenerateHistoricalTransactions_SingleDay() {
	startDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC)
	startingBalance := decimal.NewFromInt(5000)
	count := 10

	transactions := s.generator.GenerateHistoricalTransactions(s.accountID, startDate, endDate, startingBalance, count)

	s.Equal(count, len(transactions), "Should generate transactions even for single day")

	for _, txn := range transactions {
		s.Equal(startDate.Year(), txn.CreatedAt.Year())
		s.Equal(startDate.Month(), txn.CreatedAt.Month())
		s.Equal(startDate.Day(), txn.CreatedAt.Day())
	}
}

func (s *TransactionGeneratorTestSuite) TestGenerateHistoricalTransactions_LowStartingBalance() {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	startingBalance := decimal.NewFromInt(100)
	count := 50

	transactions := s.generator.GenerateHistoricalTransactions(s.accountID, startDate, endDate, startingBalance, count)

	for _, txn := range transactions {
		s.True(txn.BalanceAfter.GreaterThanOrEqual(decimal.Zero),
			"Should prevent negative balance even with low starting balance")
	}
}

// Performance Tests

func (s *TransactionGeneratorTestSuite) TestGenerateHistoricalTransactions_Performance() {
	if testing.Short() {
		s.T().Skip("Skipping performance test in short mode")
	}

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	startingBalance := decimal.NewFromInt(10000)
	count := 1000

	start := time.Now()
	transactions := s.generator.GenerateHistoricalTransactions(s.accountID, startDate, endDate, startingBalance, count)
	elapsed := time.Since(start)

	s.Equal(count, len(transactions))
	s.Less(elapsed, 5*time.Second, "Should generate 1000 transactions in under 5 seconds")

	avgTime := elapsed / time.Duration(count)
	s.Less(avgTime, 5*time.Millisecond, "Average generation time should be under 5ms per transaction")
}

// Batch Generation Tests

func (s *TransactionGeneratorTestSuite) TestGenerateBatchForMultipleAccounts() {
	accountIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	startingBalance := decimal.NewFromInt(5000)
	countPerAccount := 50

	allTransactions := make([]*models.Transaction, 0)
	for _, accountID := range accountIDs {
		transactions := s.generator.GenerateHistoricalTransactions(accountID, startDate, endDate, startingBalance, countPerAccount)
		allTransactions = append(allTransactions, transactions...)
	}

	s.Equal(len(accountIDs)*countPerAccount, len(allTransactions))

	accountCounts := make(map[uuid.UUID]int)
	for _, txn := range allTransactions {
		accountCounts[txn.AccountID]++
	}

	for _, accountID := range accountIDs {
		s.Equal(countPerAccount, accountCounts[accountID],
			"Each account should have exactly the requested count")
	}
}
