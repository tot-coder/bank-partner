package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ErrorResponse represents the standardized API error response structure
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the detailed error information
type ErrorDetail struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
	TraceID string   `json:"trace_id"`
}

// ErrorOption is a functional option for configuring error responses
type ErrorOption func(*ErrorResponse)

// WithDetails adds detail messages to the error response
func WithDetails(details ...string) ErrorOption {
	return func(er *ErrorResponse) {
		er.Error.Details = details
	}
}

// WithMessage overrides the default message for the error code
func WithMessage(message string) ErrorOption {
	return func(er *ErrorResponse) {
		er.Error.Message = message
	}
}

// NewErrorResponse creates a standardized error response with the given error code and trace ID
// Optional details can be added using functional options
func NewErrorResponse(code ErrorCode, traceID string, opts ...ErrorOption) *ErrorResponse {
	response := &ErrorResponse{
		Error: ErrorDetail{
			Code:    string(code),
			Message: GetErrorMessage(code),
			TraceID: traceID,
			Details: []string{},
		},
	}

	// Apply functional options
	for _, opt := range opts {
		opt(response)
	}

	return response
}

// NewValidationError creates a validation error response with field-specific error details
// fieldErrors is a map of field names to their error messages
func NewValidationError(fieldErrors map[string]string, traceID string) *ErrorResponse {
	details := make([]string, 0, len(fieldErrors))
	for field, message := range fieldErrors {
		details = append(details, fmt.Sprintf("%s: %s", field, message))
	}

	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    string(ValidationGeneral),
			Message: GetErrorMessage(ValidationGeneral),
			Details: details,
			TraceID: traceID,
		},
	}
}

// NewValidationErrorFromList creates a validation error from a list of detail messages
func NewValidationErrorFromList(details []string, traceID string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    string(ValidationGeneral),
			Message: GetErrorMessage(ValidationGeneral),
			Details: details,
			TraceID: traceID,
		},
	}
}

// WrapSystemError wraps an internal error with a generic system error message
// This prevents exposure of internal implementation details to clients
// The internal error is returned separately for server-side logging
func WrapSystemError(err error, traceID string) (*ErrorResponse, error) {
	response := &ErrorResponse{
		Error: ErrorDetail{
			Code:    string(SystemInternalError),
			Message: GetErrorMessage(SystemInternalError),
			Details: []string{},
			TraceID: traceID,
		},
	}
	return response, err
}

// WrapDatabaseError wraps a database error with a generic system error message
func WrapDatabaseError(err error, traceID string) (*ErrorResponse, error) {
	response := &ErrorResponse{
		Error: ErrorDetail{
			Code:    string(SystemDatabaseError),
			Message: GetErrorMessage(SystemDatabaseError),
			Details: []string{},
			TraceID: traceID,
		},
	}
	return response, err
}

// ToJSON serializes the error response to JSON bytes
func (er *ErrorResponse) ToJSON() ([]byte, error) {
	return json.Marshal(er)
}

// GetHTTPStatus returns the appropriate HTTP status code for the error code
func GetHTTPStatus(code ErrorCode) int {
	switch code {
	// 400 Bad Request - Validation errors, malformed requests
	case ValidationGeneral, ValidationRequiredField, ValidationInvalidFormat,
		ValidationOutOfRange, ValidationInvalidEmail, ValidationInvalidPhone,
		ValidationInvalidDate, CustomerInvalidID, TransactionInvalidAmount,
		TransferSameAccount, TransferInvalidAmount:
		return http.StatusBadRequest

	// 401 Unauthorized - Authentication failures
	case AuthInvalidCredentials, AuthMissingToken, AuthExpiredToken, AuthInvalidTokenFormat:
		return http.StatusUnauthorized

	// 403 Forbidden - Authorization failures
	case AuthInsufficientPermission, AuthAccountLocked:
		return http.StatusForbidden

	// 404 Not Found - Resource not found
	case CustomerNotFound, AccountNotFound, TransactionNotFound, TransferNotFound:
		return http.StatusNotFound

	// 409 Conflict - Resource state conflict
	case TransferPending, TransferFailed:
		return http.StatusConflict

	// 422 Unprocessable Entity - Semantic validation failures
	case CustomerAlreadyExists, CustomerInactive, AccountInactive,
		AccountInsufficientBalance, AccountOperationNotPermitted,
		TransactionInsufficientFunds, TransactionDuplicate,
		TransactionValidationFailed, TransactionInvalidType,
		AccountInvalidNumber, CustomerNoResults,
		TransferInsufficientFunds:
		return http.StatusUnprocessableEntity

	// 429 Too Many Requests - Rate limiting
	case SystemRateLimitExceeded:
		return http.StatusTooManyRequests

	// 503 Service Unavailable - Service temporarily unavailable
	case SystemServiceUnavailable:
		return http.StatusServiceUnavailable

	// 500 Internal Server Error - System errors (default)
	case SystemInternalError, SystemDatabaseError, SystemConfigurationError,
		SystemUnexpectedError:
		return http.StatusInternalServerError

	default:
		// Unknown error codes default to 500
		return http.StatusInternalServerError
	}
}

// GetHTTPStatusForResponse returns the HTTP status code for the error response
func (er *ErrorResponse) GetHTTPStatus() int {
	return GetHTTPStatus(ErrorCode(er.Error.Code))
}

// IsClientError returns true if the error is a 4xx client error
func (er *ErrorResponse) IsClientError() bool {
	status := er.GetHTTPStatus()
	return status >= 400 && status < 500
}

// IsServerError returns true if the error is a 5xx server error
func (er *ErrorResponse) IsServerError() bool {
	status := er.GetHTTPStatus()
	return status >= 500
}

// String returns a string representation of the error response
func (er *ErrorResponse) String() string {
	return fmt.Sprintf("[%s] %s (trace: %s)", er.Error.Code, er.Error.Message, er.Error.TraceID)
}
