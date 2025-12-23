package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders(t *testing.T) {
	e := echo.New()
	middleware := SecurityHeaders()
	
	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify all security headers are set
	headers := rec.Header()
	
	// OWASP security headers
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", headers.Get("X-XSS-Protection"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", headers.Get("Strict-Transport-Security"))
	assert.Equal(t, "default-src 'self'", headers.Get("Content-Security-Policy"))
	assert.Equal(t, "strict-origin-when-cross-origin", headers.Get("Referrer-Policy"))
	assert.Equal(t, "geolocation=(), microphone=(), camera=()", headers.Get("Permissions-Policy"))
	
	// Cache control headers
	assert.Equal(t, "no-store, no-cache, must-revalidate, private", headers.Get("Cache-Control"))
	assert.Equal(t, "no-cache", headers.Get("Pragma"))
	assert.Equal(t, "0", headers.Get("Expires"))
}

func TestSecurityHeadersNextHandlerCalled(t *testing.T) {
	e := echo.New()
	middleware := SecurityHeaders()
	
	nextCalled := false
	handler := middleware(func(c echo.Context) error {
		nextCalled = true
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	assert.NoError(t, err)
	assert.True(t, nextCalled, "Next handler should be called")
}

func TestSecurityHeadersPersistAcrossRequests(t *testing.T) {
	e := echo.New()
	middleware := SecurityHeaders()

	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Make multiple requests to ensure headers are set consistently
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(c)
		assert.NoError(t, err)

		// Verify critical headers are present each time
		headers := rec.Header()
		assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
		assert.Equal(t, "no-store, no-cache, must-revalidate, private", headers.Get("Cache-Control"))
	}
}

func TestSecurityHeadersDocsEndpoint(t *testing.T) {
	e := echo.New()
	middleware := SecurityHeaders()

	handler := middleware(func(c echo.Context) error {
		return c.HTML(http.StatusOK, "<html><body>Documentation</body></html>")
	})

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/docs")

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify relaxed CSP for /docs endpoint
	headers := rec.Header()
	csp := headers.Get("Content-Security-Policy")

	// Verify CSP allows Scalar resources
	assert.Contains(t, csp, "script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net")
	assert.Contains(t, csp, "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdn.jsdelivr.net")
	assert.Contains(t, csp, "font-src 'self' https://fonts.gstatic.com https://cdn.jsdelivr.net data:")
	assert.Contains(t, csp, "img-src 'self' data: https: blob:")
	assert.Contains(t, csp, "worker-src 'self' blob:")

	// Verify other security headers are still present
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
	assert.Equal(t, "no-store, no-cache, must-revalidate, private", headers.Get("Cache-Control"))
}