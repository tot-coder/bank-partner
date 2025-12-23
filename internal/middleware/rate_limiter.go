package middleware

import (
	"sync"
	"time"

	"array-assessment/internal/errors"
	"array-assessment/internal/handlers"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	visitors = make(map[string]*visitor)
	mu       sync.RWMutex

	// OWASP requirement: 5 req/sec prevents brute force and DoS attacks
	requestsPerSecond = 5
	burstSize         = 10
)

// RateLimiter creates a middleware for rate limiting requests per IP
func RateLimiter() echo.MiddlewareFunc {
	go cleanupVisitors()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := getIP(c)

			limiter := getVisitor(ip)
			if !limiter.Allow() {
				return handlers.SendError(c, errors.SystemRateLimitExceeded)
			}

			return next(c)
		}
	}
}

// RateLimiterWithConfig creates a rate limiter with custom configuration
func RateLimiterWithConfig(rps int, burst int) echo.MiddlewareFunc {
	requestsPerSecond = rps
	burstSize = burst

	return RateLimiter()
}

func getVisitor(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), burstSize)
		visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func getIP(c echo.Context) string {
	xff := c.Request().Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}

	xri := c.Request().Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	return c.RealIP()
}

func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		mu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(visitors, ip)
			}
		}
		mu.Unlock()
	}
}
