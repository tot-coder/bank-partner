package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
)

// ErrorHandlerTestSuite defines the test suite for error handler middleware
type ErrorHandlerTestSuite struct {
	suite.Suite
	echo *echo.Echo
}

// SetupTest runs before each test
func (s *ErrorHandlerTestSuite) SetupTest() {
	s.echo = echo.New()
	s.echo.HTTPErrorHandler = CustomHTTPErrorHandler
}

// TestErrorHandlerTestSuite runs the test suite
func TestErrorHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorHandlerTestSuite))
}

// TestCustomHTTPErrorHandler_EchoHTTPError tests handling of Echo HTTP errors
func (s *ErrorHandlerTestSuite) TestCustomHTTPErrorHandler_EchoHTTPError() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.Set(TraceIDContextKey, "test-trace-id")

	echoErr := echo.NewHTTPError(http.StatusNotFound, "Resource not found")
	CustomHTTPErrorHandler(echoErr, c)

	s.Equal(http.StatusNotFound, rec.Code)
	s.Contains(rec.Body.String(), "test-trace-id")
	s.Contains(rec.Body.String(), "Resource not found")
}

// TestCustomHTTPErrorHandler_GenericError tests handling of generic errors
func (s *ErrorHandlerTestSuite) TestCustomHTTPErrorHandler_GenericError() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.Set(TraceIDContextKey, "test-trace-id")

	err := errors.New("generic error")
	CustomHTTPErrorHandler(err, c)

	s.Equal(http.StatusInternalServerError, rec.Code)
	s.Contains(rec.Body.String(), "SYSTEM_001")
	s.Contains(rec.Body.String(), "test-trace-id")
}

// TestCustomHTTPErrorHandler_NoTraceID tests error handling without trace ID
func (s *ErrorHandlerTestSuite) TestCustomHTTPErrorHandler_NoTraceID() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	err := errors.New("test error")
	CustomHTTPErrorHandler(err, c)

	s.Equal(http.StatusInternalServerError, rec.Code)
	s.Contains(rec.Body.String(), "unknown")
}

// TestCustomHTTPErrorHandler_CommittedResponse tests that handler doesn't process committed responses
func (s *ErrorHandlerTestSuite) TestCustomHTTPErrorHandler_CommittedResponse() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	// Commit the response by writing to it
	_ = c.JSON(http.StatusOK, map[string]string{"status": "ok"})

	// Now try to handle an error - should not overwrite
	err := errors.New("test error")
	CustomHTTPErrorHandler(err, c)

	// Should still have the original 200 response
	s.Equal(http.StatusOK, rec.Code)
	s.Contains(rec.Body.String(), "ok")
}

// TestMapHTTPStatusToErrorCode_AllStatuses tests error code mapping
func (s *ErrorHandlerTestSuite) TestMapHTTPStatusToErrorCode_AllStatuses() {
	testCases := []struct {
		status       int
		expectedCode string
	}{
		{http.StatusBadRequest, "VALIDATION_001"},
		{http.StatusUnauthorized, "AUTH_002"},
		{http.StatusForbidden, "AUTH_005"},
		{http.StatusNotFound, "CUSTOMER_001"},
		{http.StatusUnprocessableEntity, "VALIDATION_001"},
		{http.StatusTooManyRequests, "SYSTEM_006"},
		{http.StatusInternalServerError, "SYSTEM_001"},
		{http.StatusServiceUnavailable, "SYSTEM_003"},
		{999, "SYSTEM_005"}, // Unknown status
	}

	for _, tc := range testCases {
		s.Run(http.StatusText(tc.status), func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := s.echo.NewContext(req, rec)
			c.Set(TraceIDContextKey, "test-trace-id")

			echoErr := echo.NewHTTPError(tc.status)
			CustomHTTPErrorHandler(echoErr, c)

			s.Equal(tc.status, rec.Code)
			s.Contains(rec.Body.String(), tc.expectedCode)
		})
	}
}

// TestCustomHTTPErrorHandler_JSONFormat tests that response is valid JSON
func (s *ErrorHandlerTestSuite) TestCustomHTTPErrorHandler_JSONFormat() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.Set(TraceIDContextKey, "test-trace-id")

	err := errors.New("test error")
	CustomHTTPErrorHandler(err, c)

	// Check Content-Type
	s.Contains(rec.Header().Get("Content-Type"), "application/json")
}
