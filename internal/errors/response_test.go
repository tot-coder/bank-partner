package errors

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

// ResponseTestSuite defines the test suite for error responses
type ResponseTestSuite struct {
	suite.Suite
	traceID string
}

// SetupTest runs before each test
func (s *ResponseTestSuite) SetupTest() {
	s.traceID = "550e8400-e29b-41d4-a716-446655440000"
}

// TestResponseTestSuite runs the test suite
func TestResponseTestSuite(t *testing.T) {
	suite.Run(t, new(ResponseTestSuite))
}

// TestNewErrorResponse_BasicUsage tests creating a basic error response
func (s *ResponseTestSuite) TestNewErrorResponse_BasicUsage() {
	response := NewErrorResponse(AuthInvalidCredentials, s.traceID)

	s.NotNil(response)
	s.Equal("AUTH_001", response.Error.Code)
	s.Equal("Invalid email or password", response.Error.Message)
	s.Equal(s.traceID, response.Error.TraceID)
	s.Empty(response.Error.Details)
}

// TestNewErrorResponse_WithDetails tests creating error response with details
func (s *ResponseTestSuite) TestNewErrorResponse_WithDetails() {
	details := []string{"Field validation failed", "Email is required"}
	response := NewErrorResponse(ValidationGeneral, s.traceID, WithDetails(details...))

	s.NotNil(response)
	s.Equal("VALIDATION_001", response.Error.Code)
	s.Equal("Validation failed", response.Error.Message)
	s.Equal(s.traceID, response.Error.TraceID)
	s.Equal(details, response.Error.Details)
}

// TestNewErrorResponse_WithCustomMessage tests creating error response with custom message
func (s *ResponseTestSuite) TestNewErrorResponse_WithCustomMessage() {
	customMessage := "Custom error message for specific context"
	response := NewErrorResponse(SystemInternalError, s.traceID, WithMessage(customMessage))

	s.NotNil(response)
	s.Equal("SYSTEM_001", response.Error.Code)
	s.Equal(customMessage, response.Error.Message)
	s.Equal(s.traceID, response.Error.TraceID)
}

// TestNewErrorResponse_WithMultipleOptions tests using multiple functional options
func (s *ResponseTestSuite) TestNewErrorResponse_WithMultipleOptions() {
	customMessage := "Custom message"
	details := []string{"Detail 1", "Detail 2"}
	response := NewErrorResponse(
		CustomerNotFound,
		s.traceID,
		WithMessage(customMessage),
		WithDetails(details...),
	)

	s.NotNil(response)
	s.Equal("CUSTOMER_001", response.Error.Code)
	s.Equal(customMessage, response.Error.Message)
	s.Equal(details, response.Error.Details)
	s.Equal(s.traceID, response.Error.TraceID)
}

// TestNewValidationError_WithFieldErrors tests creating validation error from field map
func (s *ResponseTestSuite) TestNewValidationError_WithFieldErrors() {
	fieldErrors := map[string]string{
		"email":    "must be a valid email address",
		"password": "must be at least 8 characters long",
		"name":     "is required",
	}

	response := NewValidationError(fieldErrors, s.traceID)

	s.NotNil(response)
	s.Equal("VALIDATION_001", response.Error.Code)
	s.Equal("Validation failed", response.Error.Message)
	s.Equal(s.traceID, response.Error.TraceID)
	s.Len(response.Error.Details, 3)

	// Check that all field errors are included (order may vary due to map iteration)
	detailsMap := make(map[string]bool)
	for _, detail := range response.Error.Details {
		detailsMap[detail] = true
	}
	s.True(detailsMap["email: must be a valid email address"])
	s.True(detailsMap["password: must be at least 8 characters long"])
	s.True(detailsMap["name: is required"])
}

// TestNewValidationError_EmptyFieldErrors tests validation error with empty field map
func (s *ResponseTestSuite) TestNewValidationError_EmptyFieldErrors() {
	fieldErrors := map[string]string{}
	response := NewValidationError(fieldErrors, s.traceID)

	s.NotNil(response)
	s.Equal("VALIDATION_001", response.Error.Code)
	s.Empty(response.Error.Details)
}

// TestNewValidationErrorFromList_Success tests creating validation error from list
func (s *ResponseTestSuite) TestNewValidationErrorFromList_Success() {
	details := []string{
		"email: must be a valid email address",
		"amount: must be greater than 0",
	}

	response := NewValidationErrorFromList(details, s.traceID)

	s.NotNil(response)
	s.Equal("VALIDATION_001", response.Error.Code)
	s.Equal("Validation failed", response.Error.Message)
	s.Equal(details, response.Error.Details)
	s.Equal(s.traceID, response.Error.TraceID)
}

// TestWrapSystemError_Success tests wrapping system errors
func (s *ResponseTestSuite) TestWrapSystemError_Success() {
	internalErr := errors.New("database connection failed")

	response, originalErr := WrapSystemError(internalErr, s.traceID)

	s.NotNil(response)
	s.Equal("SYSTEM_001", response.Error.Code)
	s.Equal("An unexpected error occurred. Please contact support with trace ID", response.Error.Message)
	s.Equal(s.traceID, response.Error.TraceID)
	s.Empty(response.Error.Details)

	// Ensure original error is returned for logging
	s.Equal(internalErr, originalErr)
	s.Equal("database connection failed", originalErr.Error())
}

// TestWrapSystemError_NoInternalDetailsExposed tests that internal details are not exposed
func (s *ResponseTestSuite) TestWrapSystemError_NoInternalDetailsExposed() {
	sensitiveErr := errors.New("SQL error: table 'users' does not exist at /var/lib/mysql/data")

	response, _ := WrapSystemError(sensitiveErr, s.traceID)

	// Ensure the response message doesn't contain sensitive information
	s.NotContains(response.Error.Message, "SQL")
	s.NotContains(response.Error.Message, "table")
	s.NotContains(response.Error.Message, "/var/lib/mysql")
	s.Empty(response.Error.Details)
}

// TestWrapDatabaseError_Success tests wrapping database errors
func (s *ResponseTestSuite) TestWrapDatabaseError_Success() {
	dbErr := errors.New("connection pool exhausted")

	response, originalErr := WrapDatabaseError(dbErr, s.traceID)

	s.NotNil(response)
	s.Equal("SYSTEM_002", response.Error.Code)
	s.Equal("Database connection error", response.Error.Message)
	s.Equal(s.traceID, response.Error.TraceID)
	s.Empty(response.Error.Details)

	// Ensure original error is returned
	s.Equal(dbErr, originalErr)
}

// TestToJSON_ValidSerialization tests JSON serialization of error response
func (s *ResponseTestSuite) TestToJSON_ValidSerialization() {
	response := NewErrorResponse(
		CustomerNotFound,
		s.traceID,
		WithDetails("Customer ID: 12345"),
	)

	jsonBytes, err := response.ToJSON()

	s.NoError(err)
	s.NotEmpty(jsonBytes)

	// Unmarshal and verify structure
	var unmarshaled ErrorResponse
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	s.NoError(err)
	s.Equal("CUSTOMER_001", unmarshaled.Error.Code)
	s.Equal("Customer not found", unmarshaled.Error.Message)
	s.Equal(s.traceID, unmarshaled.Error.TraceID)
	s.Contains(unmarshaled.Error.Details, "Customer ID: 12345")
}

// TestToJSON_EmptyDetails tests JSON serialization omits empty details
func (s *ResponseTestSuite) TestToJSON_EmptyDetails() {
	response := NewErrorResponse(AuthInvalidCredentials, s.traceID)

	jsonBytes, err := response.ToJSON()
	s.NoError(err)

	// Parse JSON to check structure
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonBytes, &jsonMap)
	s.NoError(err)

	errorMap := jsonMap["error"].(map[string]interface{})
	// Details should be omitted when empty
	_, hasDetails := errorMap["details"]
	s.False(hasDetails, "Empty details should be omitted from JSON")
}

// TestGetHTTPStatus_AllErrorCodes tests HTTP status mapping for all error codes
func (s *ResponseTestSuite) TestGetHTTPStatus_AllErrorCodes() {
	testCases := []struct {
		name           string
		code           ErrorCode
		expectedStatus int
	}{
		// 400 Bad Request
		{"Validation General", ValidationGeneral, http.StatusBadRequest},
		{"Validation Required Field", ValidationRequiredField, http.StatusBadRequest},
		{"Validation Invalid Email", ValidationInvalidEmail, http.StatusBadRequest},
		{"Customer Invalid ID", CustomerInvalidID, http.StatusBadRequest},
		{"Transaction Invalid Amount", TransactionInvalidAmount, http.StatusBadRequest},

		// 401 Unauthorized
		{"Auth Invalid Credentials", AuthInvalidCredentials, http.StatusUnauthorized},
		{"Auth Missing Token", AuthMissingToken, http.StatusUnauthorized},
		{"Auth Expired Token", AuthExpiredToken, http.StatusUnauthorized},
		{"Auth Invalid Token Format", AuthInvalidTokenFormat, http.StatusUnauthorized},

		// 403 Forbidden
		{"Auth Insufficient Permission", AuthInsufficientPermission, http.StatusForbidden},
		{"Auth Account Locked", AuthAccountLocked, http.StatusForbidden},

		// 404 Not Found
		{"Customer Not Found", CustomerNotFound, http.StatusNotFound},
		{"Account Not Found", AccountNotFound, http.StatusNotFound},
		{"Transaction Not Found", TransactionNotFound, http.StatusNotFound},

		// 422 Unprocessable Entity
		{"Customer Already Exists", CustomerAlreadyExists, http.StatusUnprocessableEntity},
		{"Customer Inactive", CustomerInactive, http.StatusUnprocessableEntity},
		{"Account Insufficient Balance", AccountInsufficientBalance, http.StatusUnprocessableEntity},
		{"Transaction Duplicate", TransactionDuplicate, http.StatusUnprocessableEntity},

		// 429 Too Many Requests
		{"System Rate Limit Exceeded", SystemRateLimitExceeded, http.StatusTooManyRequests},

		// 500 Internal Server Error
		{"System Internal Error", SystemInternalError, http.StatusInternalServerError},
		{"System Database Error", SystemDatabaseError, http.StatusInternalServerError},
		{"System Unexpected Error", SystemUnexpectedError, http.StatusInternalServerError},

		// 503 Service Unavailable
		{"System Service Unavailable", SystemServiceUnavailable, http.StatusServiceUnavailable},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			status := GetHTTPStatus(tc.code)
			s.Equal(tc.expectedStatus, status)
		})
	}
}

// TestGetHTTPStatus_UnknownCode tests HTTP status for unknown error code
func (s *ResponseTestSuite) TestGetHTTPStatus_UnknownCode() {
	status := GetHTTPStatus("UNKNOWN_999")
	s.Equal(http.StatusInternalServerError, status)
}

// TestGetHTTPStatusForResponse_Success tests getting HTTP status from response
func (s *ResponseTestSuite) TestGetHTTPStatusForResponse_Success() {
	response := NewErrorResponse(AuthInvalidCredentials, s.traceID)
	status := response.GetHTTPStatus()
	s.Equal(http.StatusUnauthorized, status)
}

// TestIsClientError_4xxErrors tests client error detection
func (s *ResponseTestSuite) TestIsClientError_4xxErrors() {
	clientErrorCodes := []ErrorCode{
		ValidationGeneral,
		AuthInvalidCredentials,
		AuthInsufficientPermission,
		CustomerNotFound,
		CustomerAlreadyExists,
	}

	for _, code := range clientErrorCodes {
		s.Run(string(code), func() {
			response := NewErrorResponse(code, s.traceID)
			s.True(response.IsClientError())
			s.False(response.IsServerError())
		})
	}
}

// TestIsServerError_5xxErrors tests server error detection
func (s *ResponseTestSuite) TestIsServerError_5xxErrors() {
	serverErrorCodes := []ErrorCode{
		SystemInternalError,
		SystemDatabaseError,
		SystemServiceUnavailable,
	}

	for _, code := range serverErrorCodes {
		s.Run(string(code), func() {
			response := NewErrorResponse(code, s.traceID)
			s.True(response.IsServerError())
			s.False(response.IsClientError())
		})
	}
}

// TestString_FormatsCorrectly tests string representation of error response
func (s *ResponseTestSuite) TestString_FormatsCorrectly() {
	response := NewErrorResponse(CustomerNotFound, s.traceID)
	str := response.String()

	s.Contains(str, "CUSTOMER_001")
	s.Contains(str, "Customer not found")
	s.Contains(str, s.traceID)
}

// TestErrorResponseStructure_MatchesAPISpec tests that structure matches API specification
func (s *ResponseTestSuite) TestErrorResponseStructure_MatchesAPISpec() {
	response := NewErrorResponse(
		ValidationGeneral,
		s.traceID,
		WithDetails("email: invalid format"),
	)

	jsonBytes, err := response.ToJSON()
	s.NoError(err)

	// Parse to verify structure
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonBytes, &jsonMap)
	s.NoError(err)

	// Check top-level structure
	s.Contains(jsonMap, "error")

	// Check error object structure
	errorObj := jsonMap["error"].(map[string]interface{})
	s.Contains(errorObj, "code")
	s.Contains(errorObj, "message")
	s.Contains(errorObj, "trace_id")
	s.Contains(errorObj, "details")

	// Verify types
	s.IsType("", errorObj["code"])
	s.IsType("", errorObj["message"])
	s.IsType("", errorObj["trace_id"])
	s.IsType([]interface{}{}, errorObj["details"])
}

// TestWithDetails_MultipleInvocations tests multiple WithDetails calls
func (s *ResponseTestSuite) TestWithDetails_MultipleInvocations() {
	// Last WithDetails should win (overwrite previous)
	response := NewErrorResponse(
		ValidationGeneral,
		s.traceID,
		WithDetails("detail1", "detail2"),
		WithDetails("detail3"),
	)

	s.Equal([]string{"detail3"}, response.Error.Details)
}

// TestWithMessage_MultipleInvocations tests multiple WithMessage calls
func (s *ResponseTestSuite) TestWithMessage_MultipleInvocations() {
	// Last WithMessage should win
	response := NewErrorResponse(
		SystemInternalError,
		s.traceID,
		WithMessage("First message"),
		WithMessage("Second message"),
	)

	s.Equal("Second message", response.Error.Message)
}
