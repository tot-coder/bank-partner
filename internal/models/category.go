package models

// Standard transaction categories based on industry standards
const (
	CategoryGroceries      = "GROCERIES"
	CategoryDining         = "DINING"
	CategoryTransportation = "TRANSPORTATION"
	CategoryEntertainment  = "ENTERTAINMENT"
	CategoryShopping       = "SHOPPING"
	CategoryBillsUtilities = "BILLS_UTILITIES"
	CategoryHealthcare     = "HEALTHCARE"
	CategoryEducation      = "EDUCATION"
	CategoryTravel         = "TRAVEL"
	CategoryATMCash        = "ATM_CASH"
	CategoryIncome         = "INCOME"
	CategoryFees           = "FEES"
	CategoryOther          = "OTHER"
)

// Categorization method types
const (
	CategorizationMethodMCC         = "MCC"
	CategorizationMethodMerchant    = "MERCHANT"
	CategorizationMethodDescription = "DESCRIPTION"
	CategorizationMethodFallback    = "FALLBACK"
	CategorizationMethodManual      = "MANUAL"
)

// AllCategories returns all valid category constants
func AllCategories() []string {
	return []string{
		CategoryGroceries,
		CategoryDining,
		CategoryTransportation,
		CategoryEntertainment,
		CategoryShopping,
		CategoryBillsUtilities,
		CategoryHealthcare,
		CategoryEducation,
		CategoryTravel,
		CategoryATMCash,
		CategoryIncome,
		CategoryFees,
		CategoryOther,
	}
}

// IsValidCategory checks if a category string is valid
func IsValidCategory(category string) bool {
	for _, validCategory := range AllCategories() {
		if category == validCategory {
			return true
		}
	}
	return false
}

// CategorizationResult contains the result of transaction categorization
type CategorizationResult struct {
	Category       string  `json:"category"`
	Method         string  `json:"method"`
	Confidence     float64 `json:"confidence"`
	MatchedPattern string  `json:"matched_pattern,omitempty"`
}
