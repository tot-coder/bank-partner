package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"array-assessment/internal/errors"

	"github.com/labstack/echo/v4"
)

// PanicRecovery is a middleware that recovers from panics and returns a standardized error response
func PanicRecovery() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					traceID := GetTraceID(c)
					if traceID == "" {
						traceID = "unknown"
					}

					stackTrace := string(debug.Stack())
					slog.Error("Panic recovered",
						"trace_id", traceID,
						"panic", fmt.Sprintf("%v", r),
						"stack_trace", stackTrace,
						"path", c.Request().URL.Path,
						"method", c.Request().Method,
					)

					errorResponse := errors.NewErrorResponse(
						errors.SystemInternalError,
						traceID,
					)

					if err := c.JSON(http.StatusInternalServerError, errorResponse); err != nil {
						slog.Error("Failed to send panic recovery response",
							"trace_id", traceID,
							"error", err.Error(),
						)
					}
				}
			}()

			return next(c)
		}
	}
}
