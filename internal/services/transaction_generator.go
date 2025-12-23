package services

import (
	"math/rand"
	"time"

	"array-assessment/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type transactionGenerator struct {
	merchantPool []models.MerchantInfo
	rng          *rand.Rand
}

const (
	minBalanceThreshold = 50
	hoursInDay          = 24
	biWeeklyDays        = 14
	salaryHour          = 9
	billPaymentHour     = 14
	businessHoursStart  = 6
	businessHoursEnd    = 24
)

// NewTransactionGenerator creates a new transaction generator
func NewTransactionGenerator() TransactionGeneratorInterface {
	source := rand.NewSource(time.Now().UnixNano())
	return &transactionGenerator{
		merchantPool: initializeMerchantPool(),
		rng:          rand.New(source),
	}
}

// initializeMerchantPool creates a pool of 50+ realistic merchants
func initializeMerchantPool() []models.MerchantInfo {
	return []models.MerchantInfo{
		// Groceries (10 merchants)
		{"Walmart Supercenter", models.CategoryGroceries, "5411"},
		{"Kroger", models.CategoryGroceries, "5411"},
		{"Whole Foods Market", models.CategoryGroceries, "5411"},
		{"Safeway", models.CategoryGroceries, "5411"},
		{"Trader Joe's", models.CategoryGroceries, "5411"},
		{"Costco Wholesale", models.CategoryGroceries, "5411"},
		{"Target", models.CategoryGroceries, "5411"},
		{"Publix Super Market", models.CategoryGroceries, "5411"},
		{"Aldi", models.CategoryGroceries, "5411"},
		{"H-E-B", models.CategoryGroceries, "5411"},

		// Dining & Restaurants (12 merchants)
		{"Starbucks", models.CategoryDining, "5814"},
		{"McDonald's", models.CategoryDining, "5814"},
		{"Chipotle Mexican Grill", models.CategoryDining, "5812"},
		{"Subway", models.CategoryDining, "5814"},
		{"Dunkin'", models.CategoryDining, "5814"},
		{"Panera Bread", models.CategoryDining, "5812"},
		{"Chick-fil-A", models.CategoryDining, "5814"},
		{"Olive Garden", models.CategoryDining, "5812"},
		{"Pizza Hut", models.CategoryDining, "5814"},
		{"Taco Bell", models.CategoryDining, "5814"},
		{"Panda Express", models.CategoryDining, "5814"},
		{"Five Guys", models.CategoryDining, "5814"},

		// Transportation (8 merchants)
		{"Uber", models.CategoryTransportation, "4121"},
		{"Lyft", models.CategoryTransportation, "4121"},
		{"Shell", models.CategoryTransportation, "5542"},
		{"Chevron", models.CategoryTransportation, "5542"},
		{"BP", models.CategoryTransportation, "5542"},
		{"ExxonMobil", models.CategoryTransportation, "5542"},
		{"Amtrak", models.CategoryTransportation, "4111"},
		{"Metro Transit", models.CategoryTransportation, "4111"},

		// Shopping & Retail (10 merchants)
		{"Amazon.com", models.CategoryShopping, "5942"},
		{"Best Buy", models.CategoryShopping, "5732"},
		{"Home Depot", models.CategoryShopping, "5200"},
		{"Lowe's", models.CategoryShopping, "5200"},
		{"Macy's", models.CategoryShopping, "5311"},
		{"Nordstrom", models.CategoryShopping, "5311"},
		{"Gap", models.CategoryShopping, "5651"},
		{"Nike", models.CategoryShopping, "5941"},
		{"Apple Store", models.CategoryShopping, "5732"},
		{"IKEA", models.CategoryShopping, "5712"},

		// Entertainment (7 merchants)
		{"Netflix", models.CategoryEntertainment, "7832"},
		{"Spotify", models.CategoryEntertainment, "5815"},
		{"AMC Theaters", models.CategoryEntertainment, "7832"},
		{"Regal Cinemas", models.CategoryEntertainment, "7832"},
		{"Xbox Live", models.CategoryEntertainment, "5816"},
		{"PlayStation Network", models.CategoryEntertainment, "5816"},
		{"Disney+", models.CategoryEntertainment, "7832"},

		// Bills & Utilities (6 merchants)
		{"AT&T", models.CategoryBillsUtilities, "4814"},
		{"Verizon Wireless", models.CategoryBillsUtilities, "4814"},
		{"Comcast Xfinity", models.CategoryBillsUtilities, "4899"},
		{"PG&E", models.CategoryBillsUtilities, "4900"},
		{"Duke Energy", models.CategoryBillsUtilities, "4900"},
		{"Water Department", models.CategoryBillsUtilities, "4900"},

		// Healthcare (5 merchants)
		{"CVS Pharmacy", models.CategoryHealthcare, "5912"},
		{"Walgreens", models.CategoryHealthcare, "5912"},
		{"Kaiser Permanente", models.CategoryHealthcare, "8011"},
		{"LabCorp", models.CategoryHealthcare, "8071"},
		{"Quest Diagnostics", models.CategoryHealthcare, "8071"},

		// Travel (4 merchants)
		{"Delta Air Lines", models.CategoryTravel, "3000"},
		{"United Airlines", models.CategoryTravel, "3000"},
		{"Marriott Hotels", models.CategoryTravel, "7011"},
		{"Hilton Hotels", models.CategoryTravel, "7011"},

		// Education (2 merchants)
		{"Udemy", models.CategoryEducation, "8299"},
		{"Coursera", models.CategoryEducation, "8299"},

		// ATM/Cash (2 merchants)
		{"ATM Withdrawal", models.CategoryATMCash, "6010"},
		{"Cash Deposit", models.CategoryATMCash, "6011"},
	}
}

// GetMerchantPool returns the merchant pool
func (g *transactionGenerator) GetMerchantPool() []models.MerchantInfo {
	return g.merchantPool
}

// SelectRandomMerchant selects a random merchant from the pool
func (g *transactionGenerator) SelectRandomMerchant() models.MerchantInfo {
	return g.merchantPool[g.rng.Intn(len(g.merchantPool))]
}

// GenerateTransactionType generates a transaction type with weighted distribution
// Returns (transactionType, isFee)
// Distribution: 60% debit, 35% credit, 5% fee
func (g *transactionGenerator) GenerateTransactionType() (string, bool) {
	roll := g.rng.Float64()

	if roll < 0.60 {
		return models.TransactionTypeDebit, false
	}
	if roll < 0.95 {
		return models.TransactionTypeCredit, false
	}
	return models.TransactionTypeDebit, true
}

// GenerateAmount generates a realistic amount based on category
func (g *transactionGenerator) GenerateAmount(category string) decimal.Decimal {
	minValue, maxValue := g.getAmountRange(category)
	amount := minValue + g.rng.Float64()*(maxValue-minValue)
	return decimal.NewFromFloat(amount).Round(2)
}

func (g *transactionGenerator) getAmountRange(category string) (float64, float64) {
	ranges := map[string][2]float64{
		models.CategoryGroceries:      {15.00, 250.00},
		models.CategoryDining:         {8.00, 120.00},
		models.CategoryTransportation: {10.00, 80.00},
		models.CategoryShopping:       {25.00, 450.00},
		models.CategoryEntertainment:  {10.00, 60.00},
		models.CategoryBillsUtilities: {50.00, 250.00},
		models.CategoryHealthcare:     {20.00, 300.00},
		models.CategoryTravel:         {100.00, 800.00},
		models.CategoryEducation:      {30.00, 200.00},
		models.CategoryIncome:         {2000.00, 8000.00},
		models.CategoryATMCash:        {20.00, 200.00},
	}

	if r, exists := ranges[category]; exists {
		return r[0], r[1]
	}
	return 10.00, 100.00
}

// GenerateFeeAmount generates a realistic fee amount
func (g *transactionGenerator) GenerateFeeAmount() decimal.Decimal {
	fees := []float64{2.50, 3.00, 5.00, 10.00, 15.00, 25.00, 35.00}
	return decimal.NewFromFloat(fees[g.rng.Intn(len(fees))])
}

// GenerateTimestamp generates a random timestamp within the date range
func (g *transactionGenerator) GenerateTimestamp(startDate, endDate time.Time) time.Time {
	diff := endDate.Sub(startDate)
	randomDuration := time.Duration(g.rng.Int63n(int64(diff)))
	timestamp := startDate.Add(randomDuration)

	hour := businessHoursStart + g.rng.Intn(businessHoursEnd-businessHoursStart)
	minute := g.rng.Intn(60)
	second := g.rng.Intn(60)

	return time.Date(
		timestamp.Year(),
		timestamp.Month(),
		timestamp.Day(),
		hour,
		minute,
		second,
		0,
		time.UTC,
	)
}

// GenerateSalaryTransactions generates bi-weekly salary deposits
func (g *transactionGenerator) GenerateSalaryTransactions(accountID uuid.UUID, startDate, endDate time.Time, startingBalance decimal.Decimal) []*models.Transaction {
	salaryAmounts := []float64{2500.00, 3000.00, 3500.00, 4000.00, 4500.00}
	baseSalary := salaryAmounts[g.rng.Intn(len(salaryAmounts))]

	transactions := make([]*models.Transaction, 0)
	currentBalance := startingBalance
	currentDate := startDate

	for currentDate.Before(endDate) {
		currentDate = currentDate.Add(biWeeklyDays * hoursInDay * time.Hour)
		if currentDate.After(endDate) {
			break
		}

		transaction := g.createSalaryTransaction(accountID, currentDate, baseSalary, currentBalance)
		currentBalance = transaction.BalanceAfter
		transactions = append(transactions, transaction)
	}

	return transactions
}

func (g *transactionGenerator) createSalaryTransaction(accountID uuid.UUID, date time.Time, amount float64, balance decimal.Decimal) *models.Transaction {
	salaryAmount := decimal.NewFromFloat(amount)
	timestamp := time.Date(date.Year(), date.Month(), date.Day(), salaryHour, 0, 0, 0, time.UTC)

	return &models.Transaction{
		ID:              uuid.New(),
		AccountID:       accountID,
		TransactionType: models.TransactionTypeCredit,
		Amount:          salaryAmount,
		BalanceBefore:   balance,
		BalanceAfter:    balance.Add(salaryAmount),
		Description:     "Direct Deposit - Salary Payment",
		Status:          models.TransactionStatusCompleted,
		Category:        models.CategoryIncome,
		MerchantName:    "ACME Corporation",
		Reference:       models.GenerateTransactionReference(),
		CreatedAt:       timestamp,
		UpdatedAt:       timestamp,
		ProcessedAt:     &timestamp,
	}
}

// GenerateBillTransactions generates monthly bill payments
func (g *transactionGenerator) GenerateBillTransactions(accountID uuid.UUID, startDate, endDate time.Time, startingBalance decimal.Decimal) []*models.Transaction {
	billMerchants := []models.MerchantInfo{
		{"Electric Company", models.CategoryBillsUtilities, "4900"},
		{"Internet Provider", models.CategoryBillsUtilities, "4899"},
		{"Water Department", models.CategoryBillsUtilities, "4900"},
		{"Gas Company", models.CategoryBillsUtilities, "4900"},
		{"Phone Bill", models.CategoryBillsUtilities, "4814"},
	}

	transactions := make([]*models.Transaction, 0)
	currentBalance := startingBalance
	currentDate := startDate

	for currentDate.Before(endDate) {
		currentDate = time.Date(currentDate.Year(), currentDate.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		if currentDate.After(endDate) {
			break
		}

		for _, merchant := range billMerchants {
			transaction, newBalance := g.tryCreateBillTransaction(accountID, currentDate, endDate, merchant, currentBalance)
			if transaction != nil {
				currentBalance = newBalance
				transactions = append(transactions, transaction)
			}
		}
	}

	return transactions
}

func (g *transactionGenerator) tryCreateBillTransaction(
	accountID uuid.UUID,
	currentDate, endDate time.Time,
	merchant models.MerchantInfo,
	balance decimal.Decimal,
) (*models.Transaction, decimal.Decimal) {
	billDay := 1 + g.rng.Intn(28)
	billDate := time.Date(currentDate.Year(), currentDate.Month(), billDay, billPaymentHour, 0, 0, 0, time.UTC)

	if billDate.After(endDate) {
		return nil, balance
	}

	amount := g.GenerateAmount(models.CategoryBillsUtilities)
	if balance.Sub(amount).LessThan(decimal.Zero) {
		return nil, balance
	}

	newBalance := balance.Sub(amount)
	transaction := &models.Transaction{
		ID:              uuid.New(),
		AccountID:       accountID,
		TransactionType: models.TransactionTypeDebit,
		Amount:          amount,
		BalanceBefore:   balance,
		BalanceAfter:    newBalance,
		Description:     "Bill Payment - " + merchant.Name,
		Status:          models.TransactionStatusCompleted,
		Category:        merchant.Category,
		MerchantName:    merchant.Name,
		MCCCode:         merchant.MCCCode,
		Reference:       models.GenerateTransactionReference(),
		CreatedAt:       billDate,
		UpdatedAt:       billDate,
		ProcessedAt:     &billDate,
	}

	return transaction, newBalance
}

// GenerateDailyPurchases generates realistic daily purchase transactions
func (g *transactionGenerator) GenerateDailyPurchases(accountID uuid.UUID, startDate, endDate time.Time, startingBalance decimal.Decimal) []*models.Transaction {
	transactions := make([]*models.Transaction, 0)
	currentBalance := startingBalance
	currentDate := startDate

	for currentDate.Before(endDate) {
		dailyPurchases := 1 + g.rng.Intn(4)

		for i := 0; i < dailyPurchases; i++ {
			transaction, newBalance := g.tryCreateDailyTransaction(accountID, currentDate, currentBalance)
			if transaction != nil {
				currentBalance = newBalance
				transactions = append(transactions, transaction)
			}
		}

		currentDate = currentDate.Add(hoursInDay * time.Hour)
	}

	return transactions
}

func (g *transactionGenerator) tryCreateDailyTransaction(accountID uuid.UUID, date time.Time, balance decimal.Decimal) (*models.Transaction, decimal.Decimal) {
	merchant := g.SelectRandomMerchant()
	txnType, isFee := g.GenerateTransactionType()

	amount, description, category := g.generateTransactionDetails(txnType, isFee, merchant)
	timestamp := g.GenerateTimestamp(date, date.Add(hoursInDay*time.Hour))

	newBalance := g.calculateNewBalance(balance, amount, txnType)
	if newBalance.LessThan(decimal.Zero) {
		return nil, balance
	}

	transaction := &models.Transaction{
		ID:              uuid.New(),
		AccountID:       accountID,
		TransactionType: txnType,
		Amount:          amount,
		BalanceBefore:   balance,
		BalanceAfter:    newBalance,
		Description:     description,
		Status:          models.TransactionStatusCompleted,
		Category:        category,
		MerchantName:    merchant.Name,
		MCCCode:         merchant.MCCCode,
		Reference:       models.GenerateTransactionReference(),
		CreatedAt:       timestamp,
		UpdatedAt:       timestamp,
		ProcessedAt:     &timestamp,
	}

	return transaction, newBalance
}

func (g *transactionGenerator) generateTransactionDetails(txnType string, isFee bool, merchant models.MerchantInfo) (decimal.Decimal, string, string) {
	if isFee {
		return g.GenerateFeeAmount(), "Service Fee - Banking", models.CategoryFees
	}

	amount := g.GenerateAmount(merchant.Category)
	if txnType == models.TransactionTypeCredit {
		return amount, "Refund - " + merchant.Name, merchant.Category
	}
	return amount, "Purchase at " + merchant.Name, merchant.Category
}

func (g *transactionGenerator) calculateNewBalance(balance, amount decimal.Decimal, txnType string) decimal.Decimal {
	if txnType == models.TransactionTypeCredit {
		return balance.Add(amount)
	}
	return balance.Sub(amount)
}

// GenerateHistoricalTransactions generates a complete set of historical transactions
func (g *transactionGenerator) GenerateHistoricalTransactions(
	accountID uuid.UUID,
	startDate, endDate time.Time,
	startingBalance decimal.Decimal,
	count int,
) []*models.Transaction {
	if count == 0 {
		return []*models.Transaction{}
	}

	config := g.calculateGenerationConfig(startDate, endDate, count)
	transactions := g.generateTransactionBatch(accountID, startDate, endDate, startingBalance, count, config)

	sortTransactionsByDate(transactions)
	recalculateBalances(transactions, startingBalance)

	return transactions
}

type generationConfig struct {
	totalHours          float64
	hoursPerTransaction float64
}

func (g *transactionGenerator) calculateGenerationConfig(startDate, endDate time.Time, count int) generationConfig {
	totalHours := endDate.Sub(startDate).Hours()
	hoursPerTransaction := totalHours / float64(count)
	if hoursPerTransaction < 1 {
		hoursPerTransaction = 1
	}

	return generationConfig{
		totalHours:          totalHours,
		hoursPerTransaction: hoursPerTransaction,
	}
}

func (g *transactionGenerator) generateTransactionBatch(
	accountID uuid.UUID,
	startDate, endDate time.Time,
	startingBalance decimal.Decimal,
	count int,
	config generationConfig,
) []*models.Transaction {
	transactions := make([]*models.Transaction, 0, count)
	currentBalance := startingBalance
	currentDate := startDate

	for generated := 0; generated < count; generated++ {
		currentDate = g.advanceDate(currentDate, startDate, endDate, generated, config.hoursPerTransaction)

		transaction := g.createHistoricalTransaction(accountID, currentDate, endDate, currentBalance)
		currentBalance = transaction.BalanceAfter
		transactions = append(transactions, transaction)
	}

	return transactions
}

func (g *transactionGenerator) advanceDate(currentDate, startDate, endDate time.Time, generated int, hoursPerTransaction float64) time.Time {
	if currentDate.After(endDate) {
		return startDate.Add(time.Duration(generated) * time.Hour)
	}

	hoursToAdvance := int(hoursPerTransaction)
	if hoursToAdvance < 1 {
		hoursToAdvance = 1
	}
	return currentDate.Add(time.Duration(hoursToAdvance) * time.Hour)
}

func (g *transactionGenerator) createHistoricalTransaction(accountID uuid.UUID, currentDate, endDate time.Time, balance decimal.Decimal) *models.Transaction {
	merchant := g.SelectRandomMerchant()
	txnType, isFee := g.GenerateTransactionType()

	amount, description, category := g.generateTransactionDetails(txnType, isFee, merchant)
	timestamp := g.generateBoundedTimestamp(currentDate, endDate)

	finalType, finalAmount, finalDescription, finalCategory := g.adjustTransactionForBalance(
		balance, txnType, amount, description, category,
	)

	newBalance := g.calculateNewBalance(balance, finalAmount, finalType)

	return &models.Transaction{
		ID:              uuid.New(),
		AccountID:       accountID,
		TransactionType: finalType,
		Amount:          finalAmount,
		BalanceBefore:   balance,
		BalanceAfter:    newBalance,
		Description:     finalDescription,
		Status:          models.TransactionStatusCompleted,
		Category:        finalCategory,
		MerchantName:    merchant.Name,
		MCCCode:         merchant.MCCCode,
		Reference:       models.GenerateTransactionReference(),
		CreatedAt:       timestamp,
		UpdatedAt:       timestamp,
		ProcessedAt:     &timestamp,
	}
}

func (g *transactionGenerator) generateBoundedTimestamp(currentDate, endDate time.Time) time.Time {
	timestampEnd := currentDate.Add(hoursInDay * time.Hour)
	if timestampEnd.After(endDate) {
		timestampEnd = endDate
	}
	return g.GenerateTimestamp(currentDate, timestampEnd)
}

func (g *transactionGenerator) adjustTransactionForBalance(
	balance decimal.Decimal,
	txnType string,
	amount decimal.Decimal,
	description, category string,
) (string, decimal.Decimal, string, string) {
	if txnType == models.TransactionTypeCredit {
		return txnType, amount, description, category
	}

	if balance.Sub(amount).GreaterThanOrEqual(decimal.NewFromInt(minBalanceThreshold)) {
		return txnType, amount, description, category
	}

	newAmount := g.GenerateAmount(models.CategoryIncome)
	return models.TransactionTypeCredit, newAmount, "Direct Deposit", models.CategoryIncome
}

// sortTransactionsByDate sorts transactions by creation date
func sortTransactionsByDate(transactions []*models.Transaction) {
	for i := 0; i < len(transactions); i++ {
		for j := i + 1; j < len(transactions); j++ {
			if transactions[i].CreatedAt.After(transactions[j].CreatedAt) {
				transactions[i], transactions[j] = transactions[j], transactions[i]
			}
		}
	}
}

// recalculateBalances recalculates all balances to ensure consistency
func recalculateBalances(transactions []*models.Transaction, startingBalance decimal.Decimal) {
	currentBalance := startingBalance
	minBalance := decimal.NewFromInt(minBalanceThreshold)

	for i := range transactions {
		transactions[i].BalanceBefore = currentBalance

		if transactions[i].TransactionType == models.TransactionTypeCredit {
			currentBalance = currentBalance.Add(transactions[i].Amount)
		} else {
			newBalance := calculateDebitBalance(currentBalance, transactions[i].Amount, transactions[i].ProcessingFee)

			if newBalance.LessThan(minBalance) {
				transactions[i].TransactionType = models.TransactionTypeCredit
				transactions[i].Description = "Refund - " + transactions[i].MerchantName
				currentBalance = currentBalance.Add(transactions[i].Amount)
			} else {
				currentBalance = newBalance
			}
		}

		transactions[i].BalanceAfter = currentBalance
	}
}

func calculateDebitBalance(balance, amount, fee decimal.Decimal) decimal.Decimal {
	newBalance := balance.Sub(amount)
	if !fee.IsZero() {
		newBalance = newBalance.Sub(fee)
	}
	return newBalance
}
