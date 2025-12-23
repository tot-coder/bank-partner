package errors

// ErrorCode represents a standardized error code used throughout the API
type ErrorCode string

// Authentication error codes (AUTH_*)
const (
	AuthInvalidCredentials    ErrorCode = "AUTH_001"
	AuthMissingToken          ErrorCode = "AUTH_002"
	AuthExpiredToken          ErrorCode = "AUTH_003"
	AuthInvalidTokenFormat    ErrorCode = "AUTH_004"
	AuthInsufficientPermission ErrorCode = "AUTH_005"
	AuthAccountLocked         ErrorCode = "AUTH_006"
)

// Validation error codes (VALIDATION_*)
const (
	ValidationGeneral        ErrorCode = "VALIDATION_001"
	ValidationRequiredField  ErrorCode = "VALIDATION_002"
	ValidationInvalidFormat  ErrorCode = "VALIDATION_003"
	ValidationOutOfRange     ErrorCode = "VALIDATION_004"
	ValidationInvalidEmail   ErrorCode = "VALIDATION_005"
	ValidationInvalidPhone   ErrorCode = "VALIDATION_006"
	ValidationInvalidDate    ErrorCode = "VALIDATION_007"
)

// Customer error codes (CUSTOMER_*)
const (
	CustomerNotFound      ErrorCode = "CUSTOMER_001"
	CustomerAlreadyExists ErrorCode = "CUSTOMER_002"
	CustomerInactive      ErrorCode = "CUSTOMER_003"
	CustomerInvalidID     ErrorCode = "CUSTOMER_004"
	CustomerNoResults     ErrorCode = "CUSTOMER_005"
)

// Account error codes (ACCOUNT_*)
const (
	AccountNotFound         ErrorCode = "ACCOUNT_001"
	AccountInactive         ErrorCode = "ACCOUNT_002"
	AccountInsufficientBalance ErrorCode = "ACCOUNT_003"
	AccountInvalidNumber    ErrorCode = "ACCOUNT_004"
	AccountOperationNotPermitted ErrorCode = "ACCOUNT_005"
)

// Transaction error codes (TRANSACTION_*)
const (
	TransactionNotFound       ErrorCode = "TRANSACTION_001"
	TransactionInvalidAmount  ErrorCode = "TRANSACTION_002"
	TransactionInsufficientFunds ErrorCode = "TRANSACTION_003"
	TransactionDuplicate      ErrorCode = "TRANSACTION_004"
	TransactionValidationFailed ErrorCode = "TRANSACTION_005"
	TransactionInvalidType    ErrorCode = "TRANSACTION_006"
)

// Transfer error codes (TRANSFER_*)
const (
	TransferSameAccount        ErrorCode = "TRANSFER_001"
	TransferPending            ErrorCode = "TRANSFER_002"
	TransferFailed             ErrorCode = "TRANSFER_003"
	TransferNotFound           ErrorCode = "TRANSFER_004"
	TransferInsufficientFunds  ErrorCode = "TRANSFER_005"
	TransferInvalidAmount      ErrorCode = "TRANSFER_006"
)

// System error codes (SYSTEM_*)
const (
	SystemInternalError     ErrorCode = "SYSTEM_001"
	SystemDatabaseError     ErrorCode = "SYSTEM_002"
	SystemServiceUnavailable ErrorCode = "SYSTEM_003"
	SystemConfigurationError ErrorCode = "SYSTEM_004"
	SystemUnexpectedError   ErrorCode = "SYSTEM_005"
	SystemRateLimitExceeded ErrorCode = "SYSTEM_006"
)

// errorMessages maps error codes to their default human-readable messages
var errorMessages = map[ErrorCode]string{
	// Authentication errors
	AuthInvalidCredentials:    "Invalid email or password",
	AuthMissingToken:          "Authorization token is required",
	AuthExpiredToken:          "Authorization token has expired",
	AuthInvalidTokenFormat:    "Invalid authorization token format",
	AuthInsufficientPermission: "Insufficient permissions to access this resource",
	AuthAccountLocked:         "Account is locked or disabled",

	// Validation errors
	ValidationGeneral:       "Validation failed",
	ValidationRequiredField: "Required field is missing",
	ValidationInvalidFormat: "Invalid field format",
	ValidationOutOfRange:    "Field value is out of allowed range",
	ValidationInvalidEmail:  "Invalid email address format",
	ValidationInvalidPhone:  "Invalid phone number format",
	ValidationInvalidDate:   "Invalid date format or range",

	// Customer errors
	CustomerNotFound:      "Customer not found",
	CustomerAlreadyExists: "An account with this email already exists",
	CustomerInactive:      "Customer account is inactive or suspended",
	CustomerInvalidID:     "Invalid customer ID format",
	CustomerNoResults:     "Customer search returned no results",

	// Account errors
	AccountNotFound:         "Account not found",
	AccountInactive:         "Account is closed or inactive",
	AccountInsufficientBalance: "Insufficient account balance",
	AccountInvalidNumber:    "Invalid account number or type",
	AccountOperationNotPermitted: "Account operation not permitted",

	// Transaction errors
	TransactionNotFound:       "Transaction not found",
	TransactionInvalidAmount:  "Invalid transaction amount",
	TransactionInsufficientFunds: "Insufficient account balance for this transaction",
	TransactionDuplicate:      "Transaction with this idempotency key already exists",
	TransactionValidationFailed: "Transaction validation failed",
	TransactionInvalidType:    "Invalid transaction type",

	// Transfer errors
	TransferSameAccount:       "Cannot transfer to the same account",
	TransferPending:           "A transfer with this idempotency key is still processing",
	TransferFailed:            "A transfer with this idempotency key previously failed",
	TransferNotFound:          "Transfer not found",
	TransferInsufficientFunds: "Source account has insufficient balance for this transfer",
	TransferInvalidAmount:     "Invalid transfer amount",

	// System errors
	SystemInternalError:     "An unexpected error occurred. Please contact support with trace ID",
	SystemDatabaseError:     "Database connection error",
	SystemServiceUnavailable: "Service temporarily unavailable",
	SystemConfigurationError: "System configuration error",
	SystemUnexpectedError:   "An unexpected error occurred",
	SystemRateLimitExceeded: "Rate limit exceeded. Please try again later",
}

// GetErrorMessage returns the default message for a given error code
// If the error code is not found, it returns a generic error message
func GetErrorMessage(code ErrorCode) string {
	if msg, ok := errorMessages[code]; ok {
		return msg
	}
	return "An error occurred"
}

// IsValidErrorCode checks if the provided error code is a valid registered code
func IsValidErrorCode(code ErrorCode) bool {
	_, ok := errorMessages[code]
	return ok
}
