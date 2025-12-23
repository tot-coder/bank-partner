package services

import (
	"testing"
	"time"

	"array-assessment/internal/models"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type CategoryServiceTestSuite struct {
	suite.Suite
	service *categoryService
}

func TestCategoryServiceSuite(t *testing.T) {
	suite.Run(t, new(CategoryServiceTestSuite))
}

func (s *CategoryServiceTestSuite) SetupTest() {
	s.service = NewCategoryService().(*categoryService)
}

// MCC Mapping Tests

func (s *CategoryServiceTestSuite) TestCategoryFromMCC_ValidMCCCodes() {
	testCases := []struct {
		mccCode          string
		expectedCategory string
		description      string
	}{
		{"5411", models.CategoryGroceries, "Grocery stores"},
		{"5541", models.CategoryGroceries, "Service stations (with food)"},
		{"5812", models.CategoryDining, "Restaurants"},
		{"5814", models.CategoryDining, "Fast food"},
		{"4111", models.CategoryTransportation, "Public transportation"},
		{"4121", models.CategoryTransportation, "Taxicabs"},
		{"5542", models.CategoryTransportation, "Automated fuel dispensers"},
		{"5732", models.CategoryShopping, "Electronics stores"},
		{"5945", models.CategoryShopping, "Game and toy stores"},
		{"7832", models.CategoryEntertainment, "Motion pictures"},
		{"7922", models.CategoryEntertainment, "Theatrical producers"},
		{"4814", models.CategoryBillsUtilities, "Telecom services"},
		{"4900", models.CategoryBillsUtilities, "Utilities"},
		{"8011", models.CategoryHealthcare, "Doctors"},
		{"8062", models.CategoryHealthcare, "Hospitals"},
		{"8211", models.CategoryEducation, "Elementary schools"},
		{"8220", models.CategoryEducation, "Colleges"},
		{"3000", models.CategoryTravel, "Airlines"},
		{"7011", models.CategoryTravel, "Hotels"},
		{"6010", models.CategoryATMCash, "ATM/cash withdrawal"},
		{"6011", models.CategoryATMCash, "ATM/cash deposit"},
	}

	for _, tc := range testCases {
		s.Run(tc.description, func() {
			category := s.service.CategoryFromMCC(tc.mccCode)
			s.Equal(tc.expectedCategory, category, "MCC %s should map to category %s", tc.mccCode, tc.expectedCategory)
		})
	}
}

func (s *CategoryServiceTestSuite) TestCategoryFromMCC_UnknownMCC() {
	unknownMCCs := []string{"0000", "9999", "1234"}

	for _, mcc := range unknownMCCs {
		category := s.service.CategoryFromMCC(mcc)
		s.Equal(models.CategoryOther, category, "Unknown MCC %s should default to Other", mcc)
	}
}

func (s *CategoryServiceTestSuite) TestCategoryFromMCC_EmptyMCC() {
	category := s.service.CategoryFromMCC("")
	s.Equal(models.CategoryOther, category, "Empty MCC should default to Other")
}

// Merchant Pattern Matching Tests

func (s *CategoryServiceTestSuite) TestCategorizeByMerchant_ExactMatches() {
	testCases := []struct {
		merchantName     string
		expectedCategory string
		description      string
	}{
		{"Walmart", models.CategoryGroceries, "Walmart stores"},
		{"Kroger", models.CategoryGroceries, "Kroger stores"},
		{"Whole Foods Market", models.CategoryGroceries, "Whole Foods"},
		{"Starbucks", models.CategoryDining, "Starbucks coffee"},
		{"McDonald's", models.CategoryDining, "McDonald's"},
		{"Chipotle", models.CategoryDining, "Chipotle"},
		{"Uber", models.CategoryTransportation, "Uber rides"},
		{"Lyft", models.CategoryTransportation, "Lyft rides"},
		{"Shell", models.CategoryTransportation, "Shell gas"},
		{"Chevron", models.CategoryTransportation, "Chevron gas"},
		{"Amazon", models.CategoryShopping, "Amazon purchases"},
		{"Target", models.CategoryShopping, "Target stores"},
		{"Best Buy", models.CategoryShopping, "Best Buy electronics"},
		{"Netflix", models.CategoryEntertainment, "Netflix subscription"},
		{"Spotify", models.CategoryEntertainment, "Spotify subscription"},
		{"AMC Theaters", models.CategoryEntertainment, "Movie theaters"},
		{"AT&T", models.CategoryBillsUtilities, "AT&T services"},
		{"Verizon", models.CategoryBillsUtilities, "Verizon services"},
		{"PG&E", models.CategoryBillsUtilities, "Pacific Gas & Electric"},
		{"CVS Pharmacy", models.CategoryHealthcare, "CVS pharmacy"},
		{"Walgreens", models.CategoryHealthcare, "Walgreens pharmacy"},
		{"Delta Air Lines", models.CategoryTravel, "Delta flights"},
		{"Marriott", models.CategoryTravel, "Marriott hotels"},
		{"Hilton", models.CategoryTravel, "Hilton hotels"},
	}

	for _, tc := range testCases {
		s.Run(tc.description, func() {
			category, confidence := s.service.CategorizeByMerchant(tc.merchantName)
			s.Equal(tc.expectedCategory, category, "Merchant %s should map to %s", tc.merchantName, tc.expectedCategory)
			s.GreaterOrEqual(confidence, 0.9, "Exact match should have high confidence")
		})
	}
}

func (s *CategoryServiceTestSuite) TestCategorizeByMerchant_PartialMatches() {
	testCases := []struct {
		merchantName     string
		expectedCategory string
		description      string
	}{
		{"WAL-MART STORE #1234", models.CategoryGroceries, "Walmart with store number"},
		{"STARBUCKS #45678 SAN FRANCISCO", models.CategoryDining, "Starbucks with location"},
		{"UBER *TRIP", models.CategoryTransportation, "Uber with trip info"},
		{"NETFLIX.COM", models.CategoryEntertainment, "Netflix with domain"},
		{"CHEVRON GAS STATION", models.CategoryTransportation, "Chevron with descriptor"},
	}

	for _, tc := range testCases {
		s.Run(tc.description, func() {
			category, confidence := s.service.CategorizeByMerchant(tc.merchantName)
			s.Equal(tc.expectedCategory, category, "Partial match for %s should map to %s", tc.merchantName, tc.expectedCategory)
			s.Greater(confidence, 0.0, "Partial match should have non-zero confidence")
		})
	}
}

func (s *CategoryServiceTestSuite) TestCategorizeByMerchant_UnknownMerchant() {
	unknownMerchants := []string{
		gofakeit.Company(),
		"UNKNOWN MERCHANT XYZ",
		"",
	}

	for _, merchant := range unknownMerchants {
		category, confidence := s.service.CategorizeByMerchant(merchant)
		s.Equal(models.CategoryOther, category, "Unknown merchant %s should default to Other", merchant)
		s.Equal(0.0, confidence, "Unknown merchant should have zero confidence")
	}
}

// Fuzzy Matching Tests

func (s *CategoryServiceTestSuite) TestFuzzyMatchMerchant_CaseInsensitive() {
	testCases := []struct {
		input            string
		expectedMerchant string
		description      string
	}{
		{"walmart", "Walmart", "lowercase walmart"},
		{"WALMART", "Walmart", "uppercase WALMART"},
		{"WaLmArT", "Walmart", "mixed case WaLmArT"},
		{"starbucks", "Starbucks", "lowercase starbucks"},
		{"STARBUCKS", "Starbucks", "uppercase STARBUCKS"},
	}

	for _, tc := range testCases {
		s.Run(tc.description, func() {
			merchant, _ := s.service.FuzzyMatchMerchant(tc.input)
			s.Equal(tc.expectedMerchant, merchant, "Fuzzy match should be case insensitive")
		})
	}
}

func (s *CategoryServiceTestSuite) TestFuzzyMatchMerchant_TypoTolerance() {
	testCases := []struct {
		input            string
		expectedMerchant string
		description      string
	}{
		{"Walmartt", "Walmart", "Extra letter"},
		{"Starbucsk", "Starbucks", "Transposed letters"},
		{"MacDonald's", "McDonald", "Common misspelling"},
		{"Amazn", "Amazon", "Missing letter"},
	}

	for _, tc := range testCases {
		s.Run(tc.description, func() {
			merchant, score := s.service.FuzzyMatchMerchant(tc.input)
			s.Equal(tc.expectedMerchant, merchant, "Should match %s to %s", tc.input, tc.expectedMerchant)
			s.Greater(score, 0.7, "Fuzzy match score should be above threshold")
		})
	}
}

func (s *CategoryServiceTestSuite) TestFuzzyMatchMerchant_NoMatch() {
	input := gofakeit.Company() + " TOTALLY UNKNOWN"
	merchant, score := s.service.FuzzyMatchMerchant(input)
	s.Empty(merchant, "Should not match unknown merchant")
	s.Equal(0.0, score, "Score should be zero for no match")
}

// Description Pattern Matching Tests

func (s *CategoryServiceTestSuite) TestCategorizeByDescription_CommonPatterns() {
	testCases := []struct {
		description      string
		expectedCategory string
		testDescription  string
	}{
		{"Direct Deposit - ACME CORP", models.CategoryIncome, "Payroll deposit"},
		{"Salary Payment", models.CategoryIncome, "Salary"},
		{"Grocery shopping at local store", models.CategoryOther, "Generic grocery description"},
		{"ATM Withdrawal - " + gofakeit.Address().Street, models.CategoryATMCash, "ATM withdrawal"},
		{"Cash Withdrawal", models.CategoryATMCash, "Cash out"},
		{"Monthly Service Fee", models.CategoryFees, "Bank fee"},
		{"Overdraft Fee", models.CategoryFees, "Overdraft charge"},
		{"International Transaction Fee", models.CategoryFees, "Foreign transaction fee"},
	}

	for _, tc := range testCases {
		s.Run(tc.testDescription, func() {
			category, confidence := s.service.CategorizeByDescription(tc.description)
			s.Equal(tc.expectedCategory, category, "Description %q should categorize as %s", tc.description, tc.expectedCategory)
			if tc.expectedCategory != models.CategoryOther {
				s.Greater(confidence, 0.0, "Should have non-zero confidence")
			}
		})
	}
}

// Complete Transaction Categorization Tests

func (s *CategoryServiceTestSuite) TestCategorizeTransaction_WithMCC() {
	transaction := &models.Transaction{
		ID:           uuid.New(),
		AccountID:    uuid.New(),
		Amount:       decimal.NewFromFloat(gofakeit.Price(10, 500)),
		Description:  "Purchase at store",
		MerchantName: "Walmart",
		MCCCode:      "5411",
	}

	result := s.service.CategorizeTransaction(transaction)

	s.NotNil(result)
	s.Equal(models.CategoryGroceries, result.Category)
	s.Equal(models.CategorizationMethodMCC, result.Method)
	s.Greater(result.Confidence, 0.9, "MCC-based categorization should have high confidence")
	s.NotEmpty(result.MatchedPattern)
}

func (s *CategoryServiceTestSuite) TestCategorizeTransaction_WithMerchant() {
	transaction := &models.Transaction{
		ID:           uuid.New(),
		AccountID:    uuid.New(),
		Amount:       decimal.NewFromFloat(gofakeit.Price(10, 100)),
		Description:  "Coffee purchase",
		MerchantName: "Starbucks",
	}

	result := s.service.CategorizeTransaction(transaction)

	s.NotNil(result)
	s.Equal(models.CategoryDining, result.Category)
	s.Equal(models.CategorizationMethodMerchant, result.Method)
	s.Greater(result.Confidence, 0.0, "Merchant-based categorization should have confidence")
}

func (s *CategoryServiceTestSuite) TestCategorizeTransaction_WithDescriptionOnly() {
	transaction := &models.Transaction{
		ID:          uuid.New(),
		AccountID:   uuid.New(),
		Amount:      decimal.NewFromFloat(2500.00),
		Description: "Direct Deposit - EMPLOYER PAYROLL",
	}

	result := s.service.CategorizeTransaction(transaction)

	s.NotNil(result)
	s.Equal(models.CategoryIncome, result.Category)
	s.Equal(models.CategorizationMethodDescription, result.Method)
	s.Greater(result.Confidence, 0.0, "Description-based categorization should have confidence")
}

func (s *CategoryServiceTestSuite) TestCategorizeTransaction_FallbackToOther() {
	transaction := &models.Transaction{
		ID:          uuid.New(),
		AccountID:   uuid.New(),
		Amount:      decimal.NewFromFloat(gofakeit.Price(1, 1000)),
		Description: gofakeit.Sentence(10),
	}

	result := s.service.CategorizeTransaction(transaction)

	s.NotNil(result)
	s.Equal(models.CategoryOther, result.Category)
	s.Equal(models.CategorizationMethodFallback, result.Method)
	s.Equal(0.0, result.Confidence, "Fallback categorization should have zero confidence")
}

// Manual Override Tests

func (s *CategoryServiceTestSuite) TestOverrideCategory_ValidCategory() {
	transaction := &models.Transaction{
		ID:          uuid.New(),
		AccountID:   uuid.New(),
		Amount:      decimal.NewFromFloat(50.00),
		Description: "Purchase",
		Category:    models.CategoryOther,
		Version:     1,
	}

	err := s.service.OverrideCategory(transaction, models.CategoryDining, "User correction")

	s.NoError(err)
	s.Equal(models.CategoryDining, transaction.Category)
	s.Equal(2, transaction.Version, "Version should increment")
	s.NotEqual(time.Time{}, transaction.UpdatedAt, "UpdatedAt should be set")
}

func (s *CategoryServiceTestSuite) TestOverrideCategory_InvalidCategory() {
	transaction := &models.Transaction{
		ID:          uuid.New(),
		AccountID:   uuid.New(),
		Amount:      decimal.NewFromFloat(50.00),
		Description: "Purchase",
		Category:    models.CategoryOther,
		Version:     1,
	}

	err := s.service.OverrideCategory(transaction, "INVALID_CATEGORY", "User correction")

	s.Error(err)
	s.Contains(err.Error(), "invalid category")
	s.Equal(models.CategoryOther, transaction.Category, "Category should not change")
	s.Equal(1, transaction.Version, "Version should not increment")
}

func (s *CategoryServiceTestSuite) TestOverrideCategory_EmptyReason() {
	transaction := &models.Transaction{
		ID:          uuid.New(),
		AccountID:   uuid.New(),
		Amount:      decimal.NewFromFloat(50.00),
		Description: "Purchase",
		Category:    models.CategoryOther,
		Version:     1,
	}

	err := s.service.OverrideCategory(transaction, models.CategoryDining, "")

	s.Error(err)
	s.Contains(err.Error(), "reason is required")
}

// Performance Tests

func (s *CategoryServiceTestSuite) TestCategorizationPerformance() {
	if testing.Short() {
		s.T().Skip("Skipping performance test in short mode")
	}

	transactions := make([]*models.Transaction, 100)
	for i := 0; i < 100; i++ {
		transactions[i] = &models.Transaction{
			ID:           uuid.New(),
			AccountID:    uuid.New(),
			Amount:       decimal.NewFromFloat(gofakeit.Price(1, 1000)),
			Description:  gofakeit.Sentence(5),
			MerchantName: gofakeit.Company(),
			MCCCode:      gofakeit.Numerify("####"),
		}
	}

	start := time.Now()
	for _, txn := range transactions {
		s.service.CategorizeTransaction(txn)
	}
	elapsed := time.Since(start)

	avgTime := elapsed / time.Duration(len(transactions))
	s.Less(avgTime, 50*time.Millisecond, "Average categorization should be under 50ms")
}

// Batch Categorization Tests

func (s *CategoryServiceTestSuite) TestBatchCategorize_MultipleTransactions() {
	transactions := []*models.Transaction{
		{
			ID:           uuid.New(),
			AccountID:    uuid.New(),
			Amount:       decimal.NewFromFloat(100.00),
			Description:  "Grocery shopping",
			MerchantName: "Walmart",
			MCCCode:      "5411",
		},
		{
			ID:           uuid.New(),
			AccountID:    uuid.New(),
			Amount:       decimal.NewFromFloat(15.00),
			Description:  "Coffee",
			MerchantName: "Starbucks",
			MCCCode:      "5814",
		},
		{
			ID:          uuid.New(),
			AccountID:   uuid.New(),
			Amount:      decimal.NewFromFloat(2500.00),
			Description: "Direct Deposit - SALARY",
		},
	}

	results := s.service.BatchCategorize(transactions)

	s.Len(results, 3)
	s.Equal(models.CategoryGroceries, results[0].Category)
	s.Equal(models.CategoryDining, results[1].Category)
	s.Equal(models.CategoryIncome, results[2].Category)
}

func (s *CategoryServiceTestSuite) TestBatchCategorize_EmptyInput() {
	results := s.service.BatchCategorize([]*models.Transaction{})
	s.Empty(results)
}
