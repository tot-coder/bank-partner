package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	// TraceIDHeader is the header name for the trace ID
	TraceIDHeader = "X-Trace-ID"
	// TraceIDContextKey is the context key for storing the trace ID
	TraceIDContextKey = "trace_id"
)

// RequestID is a middleware that generates a unique trace ID for each request
// and sets it in both the response header and the request context
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()

			traceID := req.Header.Get(TraceIDHeader)
			if traceID == "" {
				traceID = uuid.New().String()
			}

			c.Set(TraceIDContextKey, traceID)
			res.Header().Set(TraceIDHeader, traceID)
			return next(c)
		}
	}
}

// GetTraceID extracts the trace ID from the Echo context
// Returns empty string if not found
func GetTraceID(c echo.Context) string {
	traceID, ok := c.Get(TraceIDContextKey).(string)
	if !ok {
		return ""
	}
	return traceID
}
