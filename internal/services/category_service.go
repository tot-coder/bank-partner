package services

import (
	"errors"
	"strings"
	"time"

	"array-assessment/internal/models"
)

var (
	ErrInvalidCategory    = errors.New("invalid category")
	ErrReasonRequired     = errors.New("reason is required for manual category override")
	ErrTransactionNil     = errors.New("transaction cannot be nil")
	ErrCategoryNotChanged = errors.New("category was not changed")
)

type categoryService struct {
	mccMapping          map[string]string
	merchantPatterns    map[string]merchantPattern
	descriptionPatterns []descriptionPattern
}

type merchantPattern struct {
	normalizedName string
	category       string
	confidence     float64
}

type descriptionPattern struct {
	keywords   []string
	category   string
	confidence float64
}

// NewCategoryService creates a new CategoryServiceInterface instance
func NewCategoryService() CategoryServiceInterface {
	service := &categoryService{
		mccMapping:          initMCCMapping(),
		merchantPatterns:    initMerchantPatterns(),
		descriptionPatterns: initDescriptionPatterns(),
	}
	return service
}

// CategoryFromMCC returns the category for a given MCC code
func (s *categoryService) CategoryFromMCC(mccCode string) string {
	if mccCode == "" {
		return models.CategoryOther
	}

	if category, exists := s.mccMapping[mccCode]; exists {
		return category
	}

	return models.CategoryOther
}

// CategorizeByMerchant categorizes based on merchant name
func (s *categoryService) CategorizeByMerchant(merchantName string) (string, float64) {
	if merchantName == "" {
		return models.CategoryOther, 0.0
	}

	normalized := normalizeForMatching(merchantName)

	for pattern, mapping := range s.merchantPatterns {
		patternNormalized := normalizeForMatching(pattern)
		if strings.Contains(normalized, patternNormalized) {
			return mapping.category, mapping.confidence
		}
	}

	fuzzyMerchant, score := s.FuzzyMatchMerchant(merchantName)
	if score > 0.7 && fuzzyMerchant != "" {
		if mapping, exists := s.merchantPatterns[fuzzyMerchant]; exists {
			return mapping.category, score * mapping.confidence
		}
	}

	return models.CategoryOther, 0.0
}

// CategorizeByDescription categorizes based on transaction description
func (s *categoryService) CategorizeByDescription(description string) (string, float64) {
	if description == "" {
		return models.CategoryOther, 0.0
	}

	normalized := strings.ToLower(description)

	for _, pattern := range s.descriptionPatterns {
		for _, keyword := range pattern.keywords {
			if containsIgnoreCase(normalized, strings.ToLower(keyword)) {
				return pattern.category, pattern.confidence
			}
		}
	}

	return models.CategoryOther, 0.0
}

// FuzzyMatchMerchant performs fuzzy string matching on merchant names
func (s *categoryService) FuzzyMatchMerchant(input string) (string, float64) {
	if input == "" {
		return "", 0.0
	}

	input = strings.ToLower(strings.TrimSpace(input))
	var bestMatch string
	var bestScore float64

	for merchant := range s.merchantPatterns {
		merchantLower := strings.ToLower(merchant)
		score := calculateSimilarity(input, merchantLower)

		if score > bestScore && score > 0.7 {
			bestScore = score
			bestMatch = merchant
		}
	}

	return bestMatch, bestScore
}

// CategorizeTransaction performs complete categorization using all available methods
func (s *categoryService) CategorizeTransaction(transaction *models.Transaction) *models.CategorizationResult {
	if transaction == nil {
		return &models.CategorizationResult{
			Category:   models.CategoryOther,
			Method:     models.CategorizationMethodFallback,
			Confidence: 0.0,
		}
	}

	if transaction.MCCCode != "" {
		category := s.CategoryFromMCC(transaction.MCCCode)
		if category != models.CategoryOther {
			return &models.CategorizationResult{
				Category:       category,
				Method:         models.CategorizationMethodMCC,
				Confidence:     0.95,
				MatchedPattern: "MCC:" + transaction.MCCCode,
			}
		}
	}

	if transaction.MerchantName != "" {
		category, confidence := s.CategorizeByMerchant(transaction.MerchantName)
		if category != models.CategoryOther {
			return &models.CategorizationResult{
				Category:       category,
				Method:         models.CategorizationMethodMerchant,
				Confidence:     confidence,
				MatchedPattern: "Merchant:" + transaction.MerchantName,
			}
		}
	}

	if transaction.Description != "" {
		category, confidence := s.CategorizeByDescription(transaction.Description)
		if category != models.CategoryOther {
			return &models.CategorizationResult{
				Category:       category,
				Method:         models.CategorizationMethodDescription,
				Confidence:     confidence,
				MatchedPattern: "Description",
			}
		}
	}

	return &models.CategorizationResult{
		Category:   models.CategoryOther,
		Method:     models.CategorizationMethodFallback,
		Confidence: 0.0,
	}
}

// BatchCategorize categorizes multiple transactions
func (s *categoryService) BatchCategorize(transactions []*models.Transaction) []*models.CategorizationResult {
	results := make([]*models.CategorizationResult, 0, len(transactions))

	for _, txn := range transactions {
		results = append(results, s.CategorizeTransaction(txn))
	}

	return results
}

// OverrideCategory manually overrides a transaction category
func (s *categoryService) OverrideCategory(transaction *models.Transaction, newCategory, reason string) error {
	if transaction == nil {
		return ErrTransactionNil
	}

	if !models.IsValidCategory(newCategory) {
		return ErrInvalidCategory
	}

	if reason == "" {
		return ErrReasonRequired
	}

	if transaction.Category == newCategory {
		return ErrCategoryNotChanged
	}

	transaction.Category = newCategory
	transaction.IncrementVersion()
	transaction.UpdatedAt = time.Now()

	return nil
}

// initMCCMapping initializes the MCC code to category mapping
func initMCCMapping() map[string]string {
	return map[string]string{
		// Groceries
		"5411": models.CategoryGroceries,
		"5422": models.CategoryGroceries,
		"5441": models.CategoryGroceries,
		"5451": models.CategoryGroceries,
		"5462": models.CategoryGroceries,
		"5499": models.CategoryGroceries,
		"5541": models.CategoryGroceries,

		// Dining & Restaurants
		"5811": models.CategoryDining,
		"5812": models.CategoryDining,
		"5813": models.CategoryDining,
		"5814": models.CategoryDining,

		// Transportation
		"4111": models.CategoryTransportation,
		"4112": models.CategoryTransportation,
		"4119": models.CategoryTransportation,
		"4121": models.CategoryTransportation,
		"4131": models.CategoryTransportation,
		"4214": models.CategoryTransportation,
		"4215": models.CategoryTransportation,
		"4225": models.CategoryTransportation,
		"4411": models.CategoryTransportation,
		"4468": models.CategoryTransportation,
		"5542": models.CategoryTransportation,
		"5552": models.CategoryTransportation,
		"5571": models.CategoryTransportation,
		"5592": models.CategoryTransportation,
		"5598": models.CategoryTransportation,
		"7511": models.CategoryTransportation,
		"7512": models.CategoryTransportation,
		"7513": models.CategoryTransportation,
		"7519": models.CategoryTransportation,
		"7523": models.CategoryTransportation,
		"7531": models.CategoryTransportation,
		"7534": models.CategoryTransportation,
		"7535": models.CategoryTransportation,
		"7538": models.CategoryTransportation,
		"7542": models.CategoryTransportation,

		// Entertainment
		"5735": models.CategoryEntertainment,
		"5815": models.CategoryEntertainment,
		"5816": models.CategoryEntertainment,
		"5817": models.CategoryEntertainment,
		"5818": models.CategoryEntertainment,
		"7832": models.CategoryEntertainment,
		"7841": models.CategoryEntertainment,
		"7911": models.CategoryEntertainment,
		"7922": models.CategoryEntertainment,
		"7929": models.CategoryEntertainment,
		"7932": models.CategoryEntertainment,
		"7933": models.CategoryEntertainment,
		"7941": models.CategoryEntertainment,
		"7991": models.CategoryEntertainment,
		"7992": models.CategoryEntertainment,
		"7993": models.CategoryEntertainment,
		"7994": models.CategoryEntertainment,
		"7995": models.CategoryEntertainment,
		"7996": models.CategoryEntertainment,
		"7997": models.CategoryEntertainment,
		"7998": models.CategoryEntertainment,
		"7999": models.CategoryEntertainment,

		// Shopping
		"5200": models.CategoryShopping,
		"5211": models.CategoryShopping,
		"5231": models.CategoryShopping,
		"5251": models.CategoryShopping,
		"5261": models.CategoryShopping,
		"5271": models.CategoryShopping,
		"5311": models.CategoryShopping,
		"5331": models.CategoryShopping,
		"5399": models.CategoryShopping,
		"5611": models.CategoryShopping,
		"5621": models.CategoryShopping,
		"5631": models.CategoryShopping,
		"5641": models.CategoryShopping,
		"5651": models.CategoryShopping,
		"5661": models.CategoryShopping,
		"5681": models.CategoryShopping,
		"5691": models.CategoryShopping,
		"5697": models.CategoryShopping,
		"5698": models.CategoryShopping,
		"5699": models.CategoryShopping,
		"5712": models.CategoryShopping,
		"5713": models.CategoryShopping,
		"5714": models.CategoryShopping,
		"5718": models.CategoryShopping,
		"5719": models.CategoryShopping,
		"5722": models.CategoryShopping,
		"5732": models.CategoryShopping,
		"5733": models.CategoryShopping,
		"5734": models.CategoryShopping,
		"5945": models.CategoryShopping,
		"5946": models.CategoryShopping,
		"5947": models.CategoryShopping,
		"5948": models.CategoryShopping,
		"5949": models.CategoryShopping,
		"5950": models.CategoryShopping,
		"5960": models.CategoryShopping,
		"5961": models.CategoryShopping,
		"5962": models.CategoryShopping,
		"5963": models.CategoryShopping,
		"5964": models.CategoryShopping,
		"5965": models.CategoryShopping,
		"5966": models.CategoryShopping,
		"5967": models.CategoryShopping,
		"5968": models.CategoryShopping,
		"5969": models.CategoryShopping,
		"5970": models.CategoryShopping,
		"5971": models.CategoryShopping,
		"5972": models.CategoryShopping,
		"5973": models.CategoryShopping,
		"5975": models.CategoryShopping,
		"5976": models.CategoryShopping,
		"5977": models.CategoryShopping,
		"5978": models.CategoryShopping,
		"5983": models.CategoryShopping,
		"5992": models.CategoryShopping,
		"5993": models.CategoryShopping,
		"5994": models.CategoryShopping,
		"5995": models.CategoryShopping,
		"5999": models.CategoryShopping,

		// Bills & Utilities
		"4812": models.CategoryBillsUtilities,
		"4813": models.CategoryBillsUtilities,
		"4814": models.CategoryBillsUtilities,
		"4815": models.CategoryBillsUtilities,
		"4816": models.CategoryBillsUtilities,
		"4821": models.CategoryBillsUtilities,
		"4829": models.CategoryBillsUtilities,
		"4899": models.CategoryBillsUtilities,
		"4900": models.CategoryBillsUtilities,

		// Healthcare
		"5912": models.CategoryHealthcare,
		"8011": models.CategoryHealthcare,
		"8021": models.CategoryHealthcare,
		"8031": models.CategoryHealthcare,
		"8041": models.CategoryHealthcare,
		"8042": models.CategoryHealthcare,
		"8043": models.CategoryHealthcare,
		"8044": models.CategoryHealthcare,
		"8049": models.CategoryHealthcare,
		"8050": models.CategoryHealthcare,
		"8062": models.CategoryHealthcare,
		"8071": models.CategoryHealthcare,

		// Education
		"8211": models.CategoryEducation,
		"8220": models.CategoryEducation,
		"8241": models.CategoryEducation,
		"8244": models.CategoryEducation,
		"8249": models.CategoryEducation,
		"8299": models.CategoryEducation,

		// Travel
		"3000": models.CategoryTravel,
		"3001": models.CategoryTravel,
		"3002": models.CategoryTravel,
		"3003": models.CategoryTravel,
		"3004": models.CategoryTravel,
		"3005": models.CategoryTravel,
		"3006": models.CategoryTravel,
		"3007": models.CategoryTravel,
		"3008": models.CategoryTravel,
		"3009": models.CategoryTravel,
		"3010": models.CategoryTravel,
		"3011": models.CategoryTravel,
		"3012": models.CategoryTravel,
		"3013": models.CategoryTravel,
		"3014": models.CategoryTravel,
		"3015": models.CategoryTravel,
		"3016": models.CategoryTravel,
		"3017": models.CategoryTravel,
		"3018": models.CategoryTravel,
		"3019": models.CategoryTravel,
		"3020": models.CategoryTravel,
		"3021": models.CategoryTravel,
		"3022": models.CategoryTravel,
		"3023": models.CategoryTravel,
		"3024": models.CategoryTravel,
		"3025": models.CategoryTravel,
		"3026": models.CategoryTravel,
		"3027": models.CategoryTravel,
		"3028": models.CategoryTravel,
		"3029": models.CategoryTravel,
		"3030": models.CategoryTravel,
		"3031": models.CategoryTravel,
		"3032": models.CategoryTravel,
		"3033": models.CategoryTravel,
		"3034": models.CategoryTravel,
		"3035": models.CategoryTravel,
		"3036": models.CategoryTravel,
		"3037": models.CategoryTravel,
		"3038": models.CategoryTravel,
		"3039": models.CategoryTravel,
		"3040": models.CategoryTravel,
		"3041": models.CategoryTravel,
		"3042": models.CategoryTravel,
		"3043": models.CategoryTravel,
		"3044": models.CategoryTravel,
		"3045": models.CategoryTravel,
		"3046": models.CategoryTravel,
		"3047": models.CategoryTravel,
		"3048": models.CategoryTravel,
		"3049": models.CategoryTravel,
		"3050": models.CategoryTravel,
		"3051": models.CategoryTravel,
		"3052": models.CategoryTravel,
		"3053": models.CategoryTravel,
		"3054": models.CategoryTravel,
		"3055": models.CategoryTravel,
		"3056": models.CategoryTravel,
		"3057": models.CategoryTravel,
		"3058": models.CategoryTravel,
		"3059": models.CategoryTravel,
		"3060": models.CategoryTravel,
		"3061": models.CategoryTravel,
		"3062": models.CategoryTravel,
		"3063": models.CategoryTravel,
		"3064": models.CategoryTravel,
		"3065": models.CategoryTravel,
		"3066": models.CategoryTravel,
		"3067": models.CategoryTravel,
		"3068": models.CategoryTravel,
		"3069": models.CategoryTravel,
		"3070": models.CategoryTravel,
		"3071": models.CategoryTravel,
		"3072": models.CategoryTravel,
		"3073": models.CategoryTravel,
		"3074": models.CategoryTravel,
		"3075": models.CategoryTravel,
		"3076": models.CategoryTravel,
		"3077": models.CategoryTravel,
		"3078": models.CategoryTravel,
		"3079": models.CategoryTravel,
		"3080": models.CategoryTravel,
		"3081": models.CategoryTravel,
		"3082": models.CategoryTravel,
		"3083": models.CategoryTravel,
		"3084": models.CategoryTravel,
		"3085": models.CategoryTravel,
		"3086": models.CategoryTravel,
		"3087": models.CategoryTravel,
		"3088": models.CategoryTravel,
		"3089": models.CategoryTravel,
		"3090": models.CategoryTravel,
		"3091": models.CategoryTravel,
		"3092": models.CategoryTravel,
		"3093": models.CategoryTravel,
		"3094": models.CategoryTravel,
		"3095": models.CategoryTravel,
		"3096": models.CategoryTravel,
		"3097": models.CategoryTravel,
		"3098": models.CategoryTravel,
		"3099": models.CategoryTravel,
		"3100": models.CategoryTravel,
		"3101": models.CategoryTravel,
		"3102": models.CategoryTravel,
		"4511": models.CategoryTravel,
		"7011": models.CategoryTravel,
		"7012": models.CategoryTravel,
		"7032": models.CategoryTravel,
		"7033": models.CategoryTravel,

		// ATM/Cash
		"6010": models.CategoryATMCash,
		"6011": models.CategoryATMCash,
		"6012": models.CategoryATMCash,
		"6050": models.CategoryATMCash,
		"6051": models.CategoryATMCash,
	}
}

// initMerchantPatterns initializes common merchant patterns
func initMerchantPatterns() map[string]merchantPattern {
	return map[string]merchantPattern{
		// Groceries
		"Walmart":     {normalizedName: "Walmart", category: models.CategoryGroceries, confidence: 0.95},
		"Kroger":      {normalizedName: "Kroger", category: models.CategoryGroceries, confidence: 0.95},
		"Safeway":     {normalizedName: "Safeway", category: models.CategoryGroceries, confidence: 0.95},
		"Whole Foods": {normalizedName: "Whole Foods Market", category: models.CategoryGroceries, confidence: 0.95},
		"Trader Joe":  {normalizedName: "Trader Joes", category: models.CategoryGroceries, confidence: 0.95},
		"Costco":      {normalizedName: "Costco", category: models.CategoryGroceries, confidence: 0.95},
		"Target":      {normalizedName: "Target", category: models.CategoryShopping, confidence: 0.90},
		"Aldi":        {normalizedName: "Aldi", category: models.CategoryGroceries, confidence: 0.95},

		// Dining
		"Starbucks": {normalizedName: "Starbucks", category: models.CategoryDining, confidence: 0.95},
		"McDonald":  {normalizedName: "McDonalds", category: models.CategoryDining, confidence: 0.95},
		"Chipotle":  {normalizedName: "Chipotle", category: models.CategoryDining, confidence: 0.95},
		"Subway":    {normalizedName: "Subway", category: models.CategoryDining, confidence: 0.95},
		"Taco Bell": {normalizedName: "Taco Bell", category: models.CategoryDining, confidence: 0.95},
		"Panera":    {normalizedName: "Panera Bread", category: models.CategoryDining, confidence: 0.95},
		"Dunkin":    {normalizedName: "Dunkin Donuts", category: models.CategoryDining, confidence: 0.95},
		"Pizza Hut": {normalizedName: "Pizza Hut", category: models.CategoryDining, confidence: 0.95},

		// Transportation
		"Uber":    {normalizedName: "Uber", category: models.CategoryTransportation, confidence: 0.95},
		"Lyft":    {normalizedName: "Lyft", category: models.CategoryTransportation, confidence: 0.95},
		"Shell":   {normalizedName: "Shell", category: models.CategoryTransportation, confidence: 0.95},
		"Chevron": {normalizedName: "Chevron", category: models.CategoryTransportation, confidence: 0.95},
		"Exxon":   {normalizedName: "ExxonMobil", category: models.CategoryTransportation, confidence: 0.95},
		"BP":      {normalizedName: "BP", category: models.CategoryTransportation, confidence: 0.95},
		"Mobil":   {normalizedName: "Mobil", category: models.CategoryTransportation, confidence: 0.95},

		// Entertainment
		"Netflix": {normalizedName: "Netflix", category: models.CategoryEntertainment, confidence: 0.95},
		"Spotify": {normalizedName: "Spotify", category: models.CategoryEntertainment, confidence: 0.95},
		"AMC":     {normalizedName: "AMC Theaters", category: models.CategoryEntertainment, confidence: 0.95},
		"Hulu":    {normalizedName: "Hulu", category: models.CategoryEntertainment, confidence: 0.95},
		"Disney":  {normalizedName: "Disney Plus", category: models.CategoryEntertainment, confidence: 0.90},
		"HBO":     {normalizedName: "HBO", category: models.CategoryEntertainment, confidence: 0.95},

		// Shopping
		"Amazon":     {normalizedName: "Amazon", category: models.CategoryShopping, confidence: 0.95},
		"Best Buy":   {normalizedName: "Best Buy", category: models.CategoryShopping, confidence: 0.95},
		"Apple":      {normalizedName: "Apple Store", category: models.CategoryShopping, confidence: 0.90},
		"Home Depot": {normalizedName: "Home Depot", category: models.CategoryShopping, confidence: 0.95},
		"Lowes":      {normalizedName: "Lowes", category: models.CategoryShopping, confidence: 0.95},
		"Ikea":       {normalizedName: "IKEA", category: models.CategoryShopping, confidence: 0.95},

		// Bills & Utilities
		"AT&T":     {normalizedName: "AT&T", category: models.CategoryBillsUtilities, confidence: 0.95},
		"Verizon":  {normalizedName: "Verizon", category: models.CategoryBillsUtilities, confidence: 0.95},
		"T-Mobile": {normalizedName: "T-Mobile", category: models.CategoryBillsUtilities, confidence: 0.95},
		"Comcast":  {normalizedName: "Comcast", category: models.CategoryBillsUtilities, confidence: 0.95},
		"PG&E":     {normalizedName: "Pacific Gas & Electric", category: models.CategoryBillsUtilities, confidence: 0.95},
		"Edison":   {normalizedName: "Southern California Edison", category: models.CategoryBillsUtilities, confidence: 0.90},

		// Healthcare
		"CVS":       {normalizedName: "CVS Pharmacy", category: models.CategoryHealthcare, confidence: 0.95},
		"Walgreens": {normalizedName: "Walgreens", category: models.CategoryHealthcare, confidence: 0.95},
		"Rite Aid":  {normalizedName: "Rite Aid", category: models.CategoryHealthcare, confidence: 0.95},

		// Travel
		"Delta":             {normalizedName: "Delta Air Lines", category: models.CategoryTravel, confidence: 0.95},
		"United":            {normalizedName: "United Airlines", category: models.CategoryTravel, confidence: 0.95},
		"American Airlines": {normalizedName: "American Airlines", category: models.CategoryTravel, confidence: 0.95},
		"Southwest":         {normalizedName: "Southwest Airlines", category: models.CategoryTravel, confidence: 0.95},
		"Marriott":          {normalizedName: "Marriott", category: models.CategoryTravel, confidence: 0.95},
		"Hilton":            {normalizedName: "Hilton", category: models.CategoryTravel, confidence: 0.95},
		"Hyatt":             {normalizedName: "Hyatt", category: models.CategoryTravel, confidence: 0.95},
	}
}

// initDescriptionPatterns initializes description-based categorization patterns
func initDescriptionPatterns() []descriptionPattern {
	return []descriptionPattern{
		{
			keywords:   []string{"Direct Deposit", "Salary", "Payroll", "Paycheck", "Wage", "Income", "Employer"},
			category:   models.CategoryIncome,
			confidence: 0.95,
		},
		{
			keywords:   []string{"ATM Withdrawal", "Cash Withdrawal", "ATM", "Cash Out", "Cash Advance"},
			category:   models.CategoryATMCash,
			confidence: 0.90,
		},
		{
			keywords:   []string{"Monthly Service Fee", "Overdraft Fee", "Late Fee", "Service Charge", "Bank Fee", "Transaction Fee", "International Fee"},
			category:   models.CategoryFees,
			confidence: 0.90,
		},
		{
			keywords:   []string{"Refund", "Reimbursement", "Credit Adjustment", "Return"},
			category:   models.CategoryOther,
			confidence: 0.70,
		},
	}
}

// calculateSimilarity calculates the similarity score between two strings using Levenshtein distance
func calculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	distance := levenshteinDistance(s1, s2)
	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}

	return 1.0 - float64(distance)/float64(maxLen)
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if s1 == s2 {
		return 0
	}

	if len(s1) == 0 {
		return len(s2)
	}

	if len(s2) == 0 {
		return len(s1)
	}

	matrix := createMatrix(s1, s2)
	initializeFirstRowAndColumn(s1, s2, matrix)
	fillMatrix(s1, s2, matrix)

	return matrix[len(s1)][len(s2)]
}

func createMatrix(s1 string, s2 string) [][]int {
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}
	return matrix
}

func initializeFirstRowAndColumn(s1 string, s2 string, matrix [][]int) {
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}
}

func fillMatrix(s1 string, s2 string, matrix [][]int) {
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = calculateMinValue(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}
}

func calculateMinValue(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// normalizeForMatching normalizes strings for consistent matching
func normalizeForMatching(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, ".", "")
	return s
}
