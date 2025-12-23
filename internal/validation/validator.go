package validation

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator wraps the go-playground validator with custom rules and error formatting
type Validator struct {
	validate *validator.Validate
}

// GetValidate returns the underlying validator.Validate instance for use with Echo
func (v *Validator) GetValidate() *validator.Validate {
	return v.validate
}

// singleton instance of the validator
var instance *Validator

// GetValidator returns the singleton validator instance
func GetValidator() *Validator {
	if instance == nil {
		instance = NewValidator()
	}
	return instance
}

// NewValidator creates a new validator instance with custom rules and configuration
func NewValidator() *Validator {
	v := validator.New()

	_ = v.RegisterValidation("account_number", validateAccountNumber)
	_ = v.RegisterValidation("transaction_amount", validateTransactionAmount)
	_ = v.RegisterValidation("positive_amount", validatePositiveAmount)
	_ = v.RegisterValidation("customer_id", validateCustomerID)
	_ = v.RegisterValidation("account_type", validateAccountType)
	_ = v.RegisterValidation("transaction_type", validateTransactionType)

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &Validator{validate: v}
}

// Custom validation functions

// validateAccountNumber validates that an account number follows the expected format
// Format: 10-12 digits
func validateAccountNumber(fl validator.FieldLevel) bool {
	accountNumber := fl.Field().String()
	if accountNumber == "" {
		return false
	}

	// Account number should be 10-12 digits
	matched, _ := regexp.MatchString(`^\d{10,12}$`, accountNumber)
	return matched
}

// validateTransactionAmount validates that a transaction amount is positive and has at most 2 decimal places
func validateTransactionAmount(fl validator.FieldLevel) bool {
	amount := fl.Field().Float()

	if amount <= 0 {
		return false
	}

	// Check decimal places (at most 2)
	amountStr := fmt.Sprintf("%.10f", amount)
	parts := strings.Split(amountStr, ".")
	if len(parts) > 1 {
		decimal := strings.TrimRight(parts[1], "0")
		if len(decimal) > 2 {
			return false
		}
	}

	return true
}

// validatePositiveAmount validates that an amount is greater than 0
func validatePositiveAmount(fl validator.FieldLevel) bool {
	switch fl.Field().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fl.Field().Int() > 0
	case reflect.Float32, reflect.Float64:
		return fl.Field().Float() > 0
	default:
		return false
	}
}

// validateCustomerID validates that a customer ID is a valid UUID
func validateCustomerID(fl validator.FieldLevel) bool {
	customerID := fl.Field().String()
	if customerID == "" {
		return false
	}

	// UUID v4 format validation
	matched, _ := regexp.MatchString(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, customerID)
	return matched
}

// validateAccountType validates that account type is one of the allowed types
func validateAccountType(fl validator.FieldLevel) bool {
	accountType := strings.ToLower(fl.Field().String())
	validTypes := map[string]bool{
		"checking": true,
		"savings":  true,
		"credit":   true,
	}
	return validTypes[accountType]
}

// validateTransactionType validates that transaction type is one of the allowed types
func validateTransactionType(fl validator.FieldLevel) bool {
	txType := strings.ToLower(fl.Field().String())
	validTypes := map[string]bool{
		"deposit":    true,
		"withdrawal": true,
		"transfer":   true,
	}
	return validTypes[txType]
}
