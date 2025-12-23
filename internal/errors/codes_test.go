package errors

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// CodesTestSuite defines the test suite for error codes
type CodesTestSuite struct {
	suite.Suite
}

// TestCodesTestSuite runs the test suite
func TestCodesTestSuite(t *testing.T) {
	suite.Run(t, new(CodesTestSuite))
}

// TestGetErrorMessage_ValidCode tests getting message for valid error codes
func (s *CodesTestSuite) TestGetErrorMessage_ValidCode() {
	testCases := []struct {
		name     string
		code     ErrorCode
		expected string
	}{
		{
			name:     "Auth Invalid Credentials",
			code:     AuthInvalidCredentials,
			expected: "Invalid email or password",
		},
		{
			name:     "Auth Missing Token",
			code:     AuthMissingToken,
			expected: "Authorization token is required",
		},
		{
			name:     "Validation General",
			code:     ValidationGeneral,
			expected: "Validation failed",
		},
		{
			name:     "Customer Not Found",
			code:     CustomerNotFound,
			expected: "Customer not found",
		},
		{
			name:     "Account Insufficient Balance",
			code:     AccountInsufficientBalance,
			expected: "Insufficient account balance",
		},
		{
			name:     "Transaction Duplicate",
			code:     TransactionDuplicate,
			expected: "Transaction with this idempotency key already exists",
		},
		{
			name:     "System Internal Error",
			code:     SystemInternalError,
			expected: "An unexpected error occurred. Please contact support with trace ID",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			message := GetErrorMessage(tc.code)
			s.Equal(tc.expected, message)
		})
	}
}

// TestGetErrorMessage_InvalidCode tests getting message for invalid error code
func (s *CodesTestSuite) TestGetErrorMessage_InvalidCode() {
	message := GetErrorMessage("INVALID_CODE")
	s.Equal("An error occurred", message)
}

// TestIsValidErrorCode_ValidCodes tests validation of valid error codes
func (s *CodesTestSuite) TestIsValidErrorCode_ValidCodes() {
	validCodes := []ErrorCode{
		AuthInvalidCredentials,
		AuthMissingToken,
		AuthExpiredToken,
		AuthInvalidTokenFormat,
		AuthInsufficientPermission,
		AuthAccountLocked,
		ValidationGeneral,
		ValidationRequiredField,
		ValidationInvalidFormat,
		ValidationOutOfRange,
		ValidationInvalidEmail,
		ValidationInvalidPhone,
		ValidationInvalidDate,
		CustomerNotFound,
		CustomerAlreadyExists,
		CustomerInactive,
		CustomerInvalidID,
		CustomerNoResults,
		AccountNotFound,
		AccountInactive,
		AccountInsufficientBalance,
		AccountInvalidNumber,
		AccountOperationNotPermitted,
		TransactionNotFound,
		TransactionInvalidAmount,
		TransactionInsufficientFunds,
		TransactionDuplicate,
		TransactionValidationFailed,
		TransactionInvalidType,
		SystemInternalError,
		SystemDatabaseError,
		SystemServiceUnavailable,
		SystemConfigurationError,
		SystemUnexpectedError,
		SystemRateLimitExceeded,
	}

	for _, code := range validCodes {
		s.Run(string(code), func() {
			s.True(IsValidErrorCode(code), "Expected %s to be valid", code)
		})
	}
}

// TestIsValidErrorCode_InvalidCode tests validation of invalid error code
func (s *CodesTestSuite) TestIsValidErrorCode_InvalidCode() {
	invalidCodes := []ErrorCode{
		"INVALID_001",
		"UNKNOWN_CODE",
		"",
		"AUTH_999",
	}

	for _, code := range invalidCodes {
		s.Run(string(code), func() {
			s.False(IsValidErrorCode(code), "Expected %s to be invalid", code)
		})
	}
}

// TestErrorCodeConstants_Uniqueness ensures all error codes are unique
func (s *CodesTestSuite) TestErrorCodeConstants_Uniqueness() {
	codes := []ErrorCode{
		AuthInvalidCredentials,
		AuthMissingToken,
		AuthExpiredToken,
		AuthInvalidTokenFormat,
		AuthInsufficientPermission,
		AuthAccountLocked,
		ValidationGeneral,
		ValidationRequiredField,
		ValidationInvalidFormat,
		ValidationOutOfRange,
		ValidationInvalidEmail,
		ValidationInvalidPhone,
		ValidationInvalidDate,
		CustomerNotFound,
		CustomerAlreadyExists,
		CustomerInactive,
		CustomerInvalidID,
		CustomerNoResults,
		AccountNotFound,
		AccountInactive,
		AccountInsufficientBalance,
		AccountInvalidNumber,
		AccountOperationNotPermitted,
		TransactionNotFound,
		TransactionInvalidAmount,
		TransactionInsufficientFunds,
		TransactionDuplicate,
		TransactionValidationFailed,
		TransactionInvalidType,
		SystemInternalError,
		SystemDatabaseError,
		SystemServiceUnavailable,
		SystemConfigurationError,
		SystemUnexpectedError,
		SystemRateLimitExceeded,
	}

	seen := make(map[ErrorCode]bool)
	for _, code := range codes {
		s.False(seen[code], "Duplicate error code found: %s", code)
		seen[code] = true
	}
}

// TestErrorCodeConstants_Format ensures all error codes follow naming convention
func (s *CodesTestSuite) TestErrorCodeConstants_Format() {
	testCases := []struct {
		prefix string
		codes  []ErrorCode
	}{
		{
			prefix: "AUTH_",
			codes: []ErrorCode{
				AuthInvalidCredentials,
				AuthMissingToken,
				AuthExpiredToken,
				AuthInvalidTokenFormat,
				AuthInsufficientPermission,
				AuthAccountLocked,
			},
		},
		{
			prefix: "VALIDATION_",
			codes: []ErrorCode{
				ValidationGeneral,
				ValidationRequiredField,
				ValidationInvalidFormat,
				ValidationOutOfRange,
				ValidationInvalidEmail,
				ValidationInvalidPhone,
				ValidationInvalidDate,
			},
		},
		{
			prefix: "CUSTOMER_",
			codes: []ErrorCode{
				CustomerNotFound,
				CustomerAlreadyExists,
				CustomerInactive,
				CustomerInvalidID,
				CustomerNoResults,
			},
		},
		{
			prefix: "ACCOUNT_",
			codes: []ErrorCode{
				AccountNotFound,
				AccountInactive,
				AccountInsufficientBalance,
				AccountInvalidNumber,
				AccountOperationNotPermitted,
			},
		},
		{
			prefix: "TRANSACTION_",
			codes: []ErrorCode{
				TransactionNotFound,
				TransactionInvalidAmount,
				TransactionInsufficientFunds,
				TransactionDuplicate,
				TransactionValidationFailed,
				TransactionInvalidType,
			},
		},
		{
			prefix: "SYSTEM_",
			codes: []ErrorCode{
				SystemInternalError,
				SystemDatabaseError,
				SystemServiceUnavailable,
				SystemConfigurationError,
				SystemUnexpectedError,
				SystemRateLimitExceeded,
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.prefix, func() {
			for _, code := range tc.codes {
				s.Contains(string(code), tc.prefix, "Error code %s should start with %s", code, tc.prefix)
			}
		})
	}
}

// TestAllErrorCodesHaveMessages ensures every error code has a message
func (s *CodesTestSuite) TestAllErrorCodesHaveMessages() {
	codes := []ErrorCode{
		AuthInvalidCredentials,
		AuthMissingToken,
		AuthExpiredToken,
		AuthInvalidTokenFormat,
		AuthInsufficientPermission,
		AuthAccountLocked,
		ValidationGeneral,
		ValidationRequiredField,
		ValidationInvalidFormat,
		ValidationOutOfRange,
		ValidationInvalidEmail,
		ValidationInvalidPhone,
		ValidationInvalidDate,
		CustomerNotFound,
		CustomerAlreadyExists,
		CustomerInactive,
		CustomerInvalidID,
		CustomerNoResults,
		AccountNotFound,
		AccountInactive,
		AccountInsufficientBalance,
		AccountInvalidNumber,
		AccountOperationNotPermitted,
		TransactionNotFound,
		TransactionInvalidAmount,
		TransactionInsufficientFunds,
		TransactionDuplicate,
		TransactionValidationFailed,
		TransactionInvalidType,
		SystemInternalError,
		SystemDatabaseError,
		SystemServiceUnavailable,
		SystemConfigurationError,
		SystemUnexpectedError,
		SystemRateLimitExceeded,
	}

	for _, code := range codes {
		s.Run(string(code), func() {
			message := GetErrorMessage(code)
			s.NotEmpty(message, "Error code %s should have a message", code)
			s.NotEqual("An error occurred", message, "Error code %s should have a specific message", code)
		})
	}
}
