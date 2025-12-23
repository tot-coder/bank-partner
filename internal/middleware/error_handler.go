package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"reflect"

	"array-assessment/internal/errors"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// API errors counter metric
	apiErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_errors_total",
			Help: "Total number of API errors by code, endpoint, and status",
		},
		[]string{"code", "endpoint", "status"},
	)
)

// CustomHTTPErrorHandler is a custom error handler for Echo that formats errors
// as standardized error responses and logs them appropriately
func CustomHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	traceID := GetTraceID(c)
	if traceID == "" {
		traceID = "unknown"
	}

	var errorResponse *errors.ErrorResponse
	var httpStatus int

	if echoErr, ok := err.(*echo.HTTPError); ok {
		errorCode := mapHTTPStatusToErrorCode(echoErr.Code)
		message := fmt.Sprintf("%v", echoErr.Message)

		errorResponse = errors.NewErrorResponse(
			errorCode,
			traceID,
			errors.WithMessage(message),
		)
		httpStatus = echoErr.Code
	} else if validationErrs, ok := err.(validator.ValidationErrors); ok {
		// Handle validation errors from go-playground/validator
		fieldErrors := make(map[string]string)
		for _, fieldErr := range validationErrs {
			fieldErrors[fieldErr.Field()] = formatValidationError(fieldErr)
		}
		errorResponse = errors.NewValidationError(fieldErrors, traceID)
		httpStatus = http.StatusBadRequest
	} else {
		errorResponse, _ = errors.WrapSystemError(err, traceID)
		httpStatus = errorResponse.GetHTTPStatus()
	}

	logLevel := slog.LevelWarn
	if httpStatus >= 500 {
		logLevel = slog.LevelError
	}

	slog.Log(c.Request().Context(), logLevel, "HTTP error occurred",
		"trace_id", traceID,
		"error_code", errorResponse.Error.Code,
		"status", httpStatus,
		"message", errorResponse.Error.Message,
		"path", c.Request().URL.Path,
		"method", c.Request().Method,
		"error", err.Error(),
	)

	apiErrorsTotal.WithLabelValues(
		errorResponse.Error.Code,
		c.Path(),
		fmt.Sprintf("%d", httpStatus),
	).Inc()

	if sendErr := c.JSON(httpStatus, errorResponse); sendErr != nil {
		slog.Error("Failed to send error response",
			"trace_id", traceID,
			"error", sendErr.Error(),
		)
	}
}

// mapHTTPStatusToErrorCode maps HTTP status codes to error codes
func mapHTTPStatusToErrorCode(status int) errors.ErrorCode {
	switch status {
	case http.StatusBadRequest:
		return errors.ValidationGeneral
	case http.StatusUnauthorized:
		return errors.AuthMissingToken
	case http.StatusForbidden:
		return errors.AuthInsufficientPermission
	case http.StatusNotFound:
		return errors.CustomerNotFound // Generic not found
	case http.StatusMethodNotAllowed:
		return errors.ValidationGeneral
	case http.StatusUnprocessableEntity:
		return errors.ValidationGeneral
	case http.StatusTooManyRequests:
		return errors.SystemRateLimitExceeded
	case http.StatusInternalServerError:
		return errors.SystemInternalError
	case http.StatusServiceUnavailable:
		return errors.SystemServiceUnavailable
	default:
		return errors.SystemUnexpectedError
	}
}

// formatValidationError converts a validator.FieldError to a human-readable message
func formatValidationError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		switch fe.Kind() {
		case reflect.String:
			return fmt.Sprintf("must be at least %s characters long", fe.Param())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return fmt.Sprintf("must be at least %s", fe.Param())
		case reflect.Float32, reflect.Float64:
			return fmt.Sprintf("must be at least %s", fe.Param())
		default:
			return fmt.Sprintf("must have minimum length/value of %s", fe.Param())
		}
	case "max":
		switch fe.Kind() {
		case reflect.String:
			return fmt.Sprintf("must be at most %s characters long", fe.Param())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return fmt.Sprintf("must be at most %s", fe.Param())
		case reflect.Float32, reflect.Float64:
			return fmt.Sprintf("must be at most %s", fe.Param())
		default:
			return fmt.Sprintf("must have maximum length/value of %s", fe.Param())
		}
	case "len":
		return fmt.Sprintf("must be exactly %s characters long", fe.Param())
	case "gt":
		return fmt.Sprintf("must be greater than %s", fe.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "lt":
		return fmt.Sprintf("must be less than %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fe.Param())
	case "alpha":
		return "must contain only alphabetic characters"
	case "alphanum":
		return "must contain only alphanumeric characters"
	case "numeric":
		return "must be a valid number"
	case "uuid":
		return "must be a valid UUID"
	case "uuid4":
		return "must be a valid UUID v4"
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	case "account_number":
		return "must be a valid account number"
	case "transaction_amount":
		return "must be a valid transaction amount (positive, up to 2 decimal places)"
	case "positive_amount":
		return "must be greater than 0"
	case "customer_id":
		return "must be a valid customer ID (UUID format)"
	case "account_type":
		return "must be a valid account type (checking, savings, credit)"
	case "transaction_type":
		return "must be a valid transaction type (deposit, withdrawal, transfer)"
	default:
		return fmt.Sprintf("failed validation for '%s'", fe.Tag())
	}
}
