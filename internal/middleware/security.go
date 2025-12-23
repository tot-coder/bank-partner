package middleware

import (
	"github.com/labstack/echo/v4"
)

// SecurityHeaders adds security headers to responses
func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// OWASP requirement: Prevent MIME type sniffing attacks
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			c.Response().Header().Set("X-XSS-Protection", "1; mode=block")
			c.Response().Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

			// CSP: Relaxed policy for /docs endpoint to allow Scalar API documentation resources
			// For API documentation, we need to allow:
			// - External scripts from cdn.jsdelivr.net (Scalar CDN)
			// - Inline styles for Scalar UI
			// - Data URIs for images
			// - Fonts from various sources
			if c.Path() == "/docs" {
				c.Response().Header().Set("Content-Security-Policy",
					"default-src 'self'; "+
					"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net; "+
					"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdn.jsdelivr.net; "+
					"font-src 'self' https://fonts.gstatic.com https://cdn.jsdelivr.net data:; "+
					"img-src 'self' data: https: blob:; "+
					"connect-src 'self'; "+
					"worker-src 'self' blob:")
			} else {
				c.Response().Header().Set("Content-Security-Policy", "default-src 'self'")
			}

			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			c.Response().Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			// PCI DSS requirement: Sensitive data must not be cached
			c.Response().Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
			c.Response().Header().Set("Pragma", "no-cache")
			c.Response().Header().Set("Expires", "0")

			return next(c)
		}
	}
}