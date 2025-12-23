package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
)

// RequestIDTestSuite defines the test suite for request ID middleware
type RequestIDTestSuite struct {
	suite.Suite
	echo *echo.Echo
}

// SetupTest runs before each test
func (s *RequestIDTestSuite) SetupTest() {
	s.echo = echo.New()
}

// TestRequestIDTestSuite runs the test suite
func TestRequestIDTestSuite(t *testing.T) {
	suite.Run(t, new(RequestIDTestSuite))
}

// TestRequestID_GeneratesTraceID tests that middleware generates a trace ID
func (s *RequestIDTestSuite) TestRequestID_GeneratesTraceID() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	handler := RequestID()(func(c echo.Context) error {
		// Check that trace ID is set in context
		traceID := c.Get(TraceIDContextKey)
		s.NotNil(traceID)
		s.NotEmpty(traceID.(string))
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	s.NoError(err)

	// Check that trace ID is set in response header
	s.NotEmpty(rec.Header().Get(TraceIDHeader))
}

// TestRequestID_UsesExistingTraceID tests that middleware uses existing trace ID from request
func (s *RequestIDTestSuite) TestRequestID_UsesExistingTraceID() {
	existingTraceID := "existing-trace-id-12345"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(TraceIDHeader, existingTraceID)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	handler := RequestID()(func(c echo.Context) error {
		// Check that existing trace ID is used
		traceID := c.Get(TraceIDContextKey).(string)
		s.Equal(existingTraceID, traceID)
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	s.NoError(err)

	// Check that same trace ID is in response header
	s.Equal(existingTraceID, rec.Header().Get(TraceIDHeader))
}

// TestRequestID_TraceIDInBothContextAndHeader tests trace ID is in both places
func (s *RequestIDTestSuite) TestRequestID_TraceIDInBothContextAndHeader() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	var contextTraceID string
	handler := RequestID()(func(c echo.Context) error {
		contextTraceID = c.Get(TraceIDContextKey).(string)
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	s.NoError(err)

	// Both should have the same trace ID
	headerTraceID := rec.Header().Get(TraceIDHeader)
	s.Equal(contextTraceID, headerTraceID)
}

// TestGetTraceID_ReturnsTraceIDFromContext tests GetTraceID helper function
func (s *RequestIDTestSuite) TestGetTraceID_ReturnsTraceIDFromContext() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	handler := RequestID()(func(c echo.Context) error {
		traceID := GetTraceID(c)
		s.NotEmpty(traceID)
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	s.NoError(err)
}

// TestGetTraceID_ReturnsEmptyWhenNotSet tests GetTraceID when trace ID not set
func (s *RequestIDTestSuite) TestGetTraceID_ReturnsEmptyWhenNotSet() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	traceID := GetTraceID(c)
	s.Empty(traceID)
}

// TestRequestID_UUIDFormat tests that generated trace ID is a valid UUID
func (s *RequestIDTestSuite) TestRequestID_UUIDFormat() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	handler := RequestID()(func(c echo.Context) error {
		traceID := GetTraceID(c)
		// Check UUID v4 format (8-4-4-4-12 characters)
		s.Regexp(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, traceID)
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	s.NoError(err)
}
