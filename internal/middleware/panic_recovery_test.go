package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"array-assessment/internal/errors"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
)

// PanicRecoveryTestSuite defines the test suite for panic recovery middleware
type PanicRecoveryTestSuite struct {
	suite.Suite
	echo *echo.Echo
}

// SetupTest runs before each test
func (s *PanicRecoveryTestSuite) SetupTest() {
	s.echo = echo.New()
}

// TestPanicRecoveryTestSuite runs the test suite
func TestPanicRecoveryTestSuite(t *testing.T) {
	suite.Run(t, new(PanicRecoveryTestSuite))
}

// TestPanicRecovery_RecoverFromPanic tests that middleware recovers from panic
func (s *PanicRecoveryTestSuite) TestPanicRecovery_RecoverFromPanic() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.Set(TraceIDContextKey, "test-trace-id")

	handler := PanicRecovery()(func(c echo.Context) error {
		panic("test panic")
	})

	// Should not panic - middleware should catch it
	s.NotPanics(func() {
		_ = handler(c)
	})

	// Should return 500 status
	s.Equal(http.StatusInternalServerError, rec.Code)

	// Should return standardized error response
	var errorResponse errors.ErrorResponse
	err := json.Unmarshal(rec.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal("SYSTEM_001", errorResponse.Error.Code)
	s.Equal("test-trace-id", errorResponse.Error.TraceID)
}

// TestPanicRecovery_NoTraceID tests panic recovery when no trace ID is set
func (s *PanicRecoveryTestSuite) TestPanicRecovery_NoTraceID() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	handler := PanicRecovery()(func(c echo.Context) error {
		panic("test panic")
	})

	s.NotPanics(func() {
		_ = handler(c)
	})

	// Should still return error response with "unknown" trace ID
	var errorResponse errors.ErrorResponse
	err := json.Unmarshal(rec.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal("SYSTEM_001", errorResponse.Error.Code)
	s.Equal("unknown", errorResponse.Error.TraceID)
}

// TestPanicRecovery_NormalFlow tests that middleware doesn't interfere with normal flow
func (s *PanicRecoveryTestSuite) TestPanicRecovery_NormalFlow() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	handler := PanicRecovery()(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	err := handler(c)
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
}

// TestPanicRecovery_DifferentPanicTypes tests recovery from different panic types
func (s *PanicRecoveryTestSuite) TestPanicRecovery_DifferentPanicTypes() {
	testCases := []struct {
		name      string
		panicWith interface{}
	}{
		{"String panic", "string panic"},
		{"Int panic", 42},
		{"Struct panic", struct{ msg string }{"error"}},
		{"Nil panic", nil},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := s.echo.NewContext(req, rec)
			c.Set(TraceIDContextKey, "test-trace-id")

			handler := PanicRecovery()(func(c echo.Context) error {
				panic(tc.panicWith)
			})

			s.NotPanics(func() {
				_ = handler(c)
			})

			s.Equal(http.StatusInternalServerError, rec.Code)
		})
	}
}
