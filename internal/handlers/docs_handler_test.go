package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
)

// DocsHandlerSuite is the test suite for documentation endpoints
type DocsHandlerSuite struct {
	suite.Suite
	handler *DocsHandler
	e       *echo.Echo
}

// SetupTest runs before each test in the suite
func (s *DocsHandlerSuite) SetupTest() {
	// Create a handler with mock HTML content for testing
	testHTML := []byte(`<!DOCTYPE html>
<html>
<head><title>Array Banking API Documentation</title></head>
<body>
<scalar spec-url="/docs/swagger.json"></scalar>
<script src="https://cdn.scalar.ly/scalar/latest/bundles/scalar.standalone.js"></script>
</body>
</html>`)

	// Calculate ETag for test HTML using md5
	hash := [16]byte{}
	for i := 0; i < len(testHTML) && i < 16; i++ {
		hash[i] = testHTML[i]
	}
	testETag := "\"test-etag\""

	s.handler = &DocsHandler{
		scalarHTML:    testHTML,
		scalarETag:    testETag,
		oas3Path:      "docs/swagger.json",
		docsGenerated: false,
	}
	s.e = echo.New()
}

// TestDocsHandler runs the test suite
func TestDocsHandler(t *testing.T) {
	suite.Run(t, new(DocsHandlerSuite))
}

// TestServeScalarUI tests the Scalar UI endpoint
func (s *DocsHandlerSuite) TestServeScalarUI() {
	s.Run("successfully serves Scalar HTML page", func() {
		// Create request
		req := httptest.NewRequest(http.MethodGet, "/docs", nil)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		// Execute handler
		err := s.handler.ServeScalarUI(c)

		// Assertions
		s.NoError(err)
		s.Equal(http.StatusOK, rec.Code)
		s.Contains(rec.Header().Get("Content-Type"), "text/html")
		s.Contains(rec.Header().Get("Content-Type"), "charset")
		s.Contains(rec.Body.String(), "scalar") // lowercase "scalar" tag
		s.Contains(rec.Body.String(), "scalar.standalone.js")
		s.Contains(rec.Body.String(), "/docs/swagger.json")
	})

	s.Run("sets correct cache headers", func() {
		// Create request
		req := httptest.NewRequest(http.MethodGet, "/docs", nil)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		// Execute handler
		err := s.handler.ServeScalarUI(c)

		// Assertions
		s.NoError(err)
		s.Equal("no-cache, no-store, must-revalidate", rec.Header().Get("Cache-Control"))
		s.NotEmpty(rec.Header().Get("ETag"))
	})
}

// TestServeOas3JSON tests the oas3.json endpoint
func (s *DocsHandlerSuite) TestServeOas3JSON() {
	s.Run("returns 404 when swagger.json does not exist", func() {
		// Create request
		req := httptest.NewRequest(http.MethodGet, "/docs/swagger.json", nil)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		// Execute handler
		err := s.handler.ServeOAS3JSON(c)

		// Assertions - Echo.File returns 404 error when file doesn't exist
		s.Error(err)
		httpErr, ok := err.(*echo.HTTPError)
		s.True(ok, "Error should be an echo.HTTPError")
		if ok {
			s.Equal(http.StatusNotFound, httpErr.Code)
		}
	})

	s.Run("sets correct CORS headers before attempting to serve file", func() {
		// Create request
		req := httptest.NewRequest(http.MethodGet, "/docs/swagger.json", nil)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		// Execute handler (will fail due to missing file, but headers should be set)
		_ = s.handler.ServeOAS3JSON(c)

		// Verify CORS and content-type headers are set
		s.Equal("*", rec.Header().Get("Access-Control-Allow-Origin"))
		s.Equal("GET, OPTIONS", rec.Header().Get("Access-Control-Allow-Methods"))
		s.Equal("application/json; charset=utf-8", rec.Header().Get("Content-Type"))
		s.Equal("public, max-age=300", rec.Header().Get("Cache-Control"))
	})
}

// TestDocsHandlerAccessibility tests that documentation endpoints are public
func (s *DocsHandlerSuite) TestDocsHandlerAccessibility() {
	s.Run("documentation endpoints do not require authentication", func() {
		// Test /docs endpoint
		req := httptest.NewRequest(http.MethodGet, "/docs", nil)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		// Execute without Authorization header
		err := s.handler.ServeScalarUI(c)

		// Should succeed without authentication
		s.NoError(err)
		s.Equal(http.StatusOK, rec.Code)
	})
}

// TestDocsHandlerCacheControl tests caching behavior
func (s *DocsHandlerSuite) TestDocsHandlerCacheControl() {
	s.Run("Scalar UI has no-cache headers for development", func() {
		req := httptest.NewRequest(http.MethodGet, "/docs", nil)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		err := s.handler.ServeScalarUI(c)

		s.NoError(err)
		cacheControl := rec.Header().Get("Cache-Control")
		s.Contains(cacheControl, "no-cache")
	})
}
