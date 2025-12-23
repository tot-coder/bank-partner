package handlers

import (
	"net/http"

	"array-assessment/internal/errors"

	"github.com/labstack/echo/v4"
)

// STANDARDIZED ERROR HANDLING PATTERNS
//
// All handlers must use the following standardized error response functions:
//
// 1. SendError - For client errors and business logic errors (4xx responses)
//    Use cases:
//    - Validation errors: SendError(c, errors.ValidationGeneral, errors.WithDetails("..."))
//    - Authentication errors: SendError(c, errors.AuthInvalidCredentials)
//    - Authorization errors: SendError(c, errors.AuthInsufficientPermission)
//    - Not found errors: SendError(c, errors.CustomerNotFound)
//    - Business rule violations: SendError(c, errors.AccountInsufficientBalance)
//
// 2. SendSystemError - For system/internal errors (500 responses)
//    Use cases:
//    - Database errors from repositories
//    - Service layer internal errors
//    - Unexpected errors that should not expose internal details to client
//
// DO NOT USE:
//    - echo.NewHTTPError() - Use SendError or SendSystemError instead
//    - Direct c.JSON() for errors - Use the helper functions
//    - return err without wrapping - Use SendSystemError to protect internal details

const (
	// TraceIDContextKey is the context key for storing the trace ID
	TraceIDContextKey = "trace_id"
)

// SuccessResponse represents a standard success response
// Used for successful API responses with data, messages, and metadata
type SuccessResponse struct {
	Data    interface{} `json:"data,omitempty" swaggertype:"object"`
	Message string      `json:"message,omitempty"`
	Meta    interface{} `json:"meta,omitempty" swaggertype:"object"`
}

// ErrorResponse is an alias for the standardized error response type
// Used for backward compatibility in tests
type ErrorResponse = errors.ErrorResponse

// Helper functions for creating standardized error responses in handlers
// These wrap the internal/errors package for convenience

// getTraceID extracts the trace ID from the Echo context
func getTraceID(c echo.Context) string {
	traceID, ok := c.Get(TraceIDContextKey).(string)
	if !ok {
		return ""
	}
	return traceID
}

// SendError sends a standardized error response with trace ID from context
func SendError(c echo.Context, code errors.ErrorCode, opts ...errors.ErrorOption) error {
	traceID := getTraceID(c)
	errorResponse := errors.NewErrorResponse(code, traceID, opts...)
	return c.JSON(errorResponse.GetHTTPStatus(), errorResponse)
}

// SendSystemError wraps a system error with generic message and logs the internal error
func SendSystemError(c echo.Context, err error) error {
	traceID := getTraceID(c)
	errorResponse, _ := errors.WrapSystemError(err, traceID)
	return c.JSON(http.StatusInternalServerError, errorResponse)
}
