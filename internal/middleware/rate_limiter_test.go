package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *testing.T) {
	// Reset the global visitors map for clean test
	mu.Lock()
	visitors = make(map[string]*visitor)
	mu.Unlock()
	
	e := echo.New()
	middleware := RateLimiter()
	
	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Test that requests within limit are allowed
	successCount := 0
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(c)
		if err == nil {
			successCount++
		}
	}
	assert.Equal(t, 5, successCount, "All initial requests should succeed")

	// Make many requests to exceed rate limit
	rateLimited := false
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(c)
		// Rate limiter uses SendError which sends response and returns nil
		if err == nil && rec.Code == http.StatusTooManyRequests {
			rateLimited = true
			break
		}
	}

	assert.True(t, rateLimited, "Should be rate limited after many requests")
}

func TestRateLimiterWithConfig(t *testing.T) {
	// Reset the global visitors map for clean test
	mu.Lock()
	visitors = make(map[string]*visitor)
	mu.Unlock()
	
	e := echo.New()
	middleware := RateLimiterWithConfig(2, 4) // Lower limits for testing
	
	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Should allow initial burst
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.2:12345"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Next request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	// Rate limiter uses SendError which sends response and returns nil
	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimiterDifferentIPs(t *testing.T) {
	// Reset the global visitors map and rate limiter config for clean test
	mu.Lock()
	visitors = make(map[string]*visitor)
	requestsPerSecond = 5  // Reset to default
	burstSize = 10         // Reset to default
	mu.Unlock()
	
	// Sleep briefly to ensure cleanup goroutine doesn't interfere
	time.Sleep(10 * time.Millisecond)
	
	e := echo.New()
	middleware := RateLimiter()
	
	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Different IPs should have independent rate limits
	ips := []string{"192.168.1.1:1234", "192.168.1.2:1234", "192.168.1.3:1234"}
	
	for _, ip := range ips {
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = ip
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler(c)
			if err != nil {
				t.Logf("Request %d for IP %s failed: %v", i, ip, err)
			}
			assert.NoError(t, err, "Request %d for IP %s should succeed", i, ip)
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	}
}

func TestGetIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name: "X-Forwarded-For header",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
			},
			remoteAddr: "127.0.0.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name: "X-Real-IP header",
			headers: map[string]string{
				"X-Real-IP": "192.168.1.2",
			},
			remoteAddr: "127.0.0.1:12345",
			expected:   "192.168.1.2",
		},
		{
			name: "X-Forwarded-For takes precedence",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
				"X-Real-IP":       "192.168.1.2",
			},
			remoteAddr: "127.0.0.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "Falls back to RealIP",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.3:12345",
			expected:   "192.168.1.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tt.remoteAddr
			
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			
			ip := getIP(c)
			assert.Equal(t, tt.expected, ip)
		})
	}
}

func TestVisitorCleanup(t *testing.T) {
	// Create a visitor with old lastSeen
	mu.Lock()
	visitors = make(map[string]*visitor)
	oldVisitor := &visitor{
		limiter:  nil,
		lastSeen: time.Now().Add(-5 * time.Minute),
	}
	visitors["old_ip"] = oldVisitor
	
	newVisitor := &visitor{
		limiter:  nil,
		lastSeen: time.Now(),
	}
	visitors["new_ip"] = newVisitor
	mu.Unlock()

	// Run cleanup manually
	mu.Lock()
	for ip, v := range visitors {
		if time.Since(v.lastSeen) > 3*time.Minute {
			delete(visitors, ip)
		}
	}
	visitorCount := len(visitors)
	mu.Unlock()

	assert.Equal(t, 1, visitorCount, "Old visitor should be removed")
	
	mu.RLock()
	_, oldExists := visitors["old_ip"]
	_, newExists := visitors["new_ip"]
	mu.RUnlock()
	
	assert.False(t, oldExists, "Old visitor should not exist")
	assert.True(t, newExists, "New visitor should still exist")
}

func TestRateLimiterConcurrency(t *testing.T) {
	// Reset the global visitors map and config for clean test
	mu.Lock()
	visitors = make(map[string]*visitor)
	requestsPerSecond = 5  // Reset to default
	burstSize = 10         // Reset to default
	mu.Unlock()
	
	e := echo.New()
	middleware := RateLimiter()
	
	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	var wg sync.WaitGroup
	successCount := 0
	rateLimitCount := 0
	var mu sync.Mutex

	// Simulate concurrent requests from same IP
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler(c)

			mu.Lock()
			// Rate limiter uses SendError which sends response and returns nil
			if err == nil {
				if rec.Code == http.StatusOK {
					successCount++
				} else if rec.Code == http.StatusTooManyRequests {
					rateLimitCount++
				}
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Some requests should succeed, some should be rate limited
	assert.Greater(t, successCount, 0, "Some requests should succeed")
	assert.Greater(t, rateLimitCount, 0, "Some requests should be rate limited")
	assert.Equal(t, 20, successCount+rateLimitCount, "All requests should be accounted for")
}