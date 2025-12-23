package handlers

import (
	"net/http"
	"time"

	"array-assessment/internal/errors"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// HealthCheckHandler handles the health check endpoint
type HealthCheckHandler struct {
	db *gorm.DB
}

// NewHealthCheckHandler creates a new health check handler
func NewHealthCheckHandler(db *gorm.DB) *HealthCheckHandler {
	return &HealthCheckHandler{db: db}
}

// HealthCheck adds the health check endpoint
// @Summary Health check
// @Description Check API and database connectivity status
// @Tags Health
// @Produce json
// @Success 200 {object} object{status=string,time=string} "Service is healthy"
// @Failure 503 {object} errors.ErrorResponse "SYSTEM_003 - Service unavailable (database connection failed)"
// @Router /health [get]
func (h *HealthCheckHandler) HealthCheck(c echo.Context) error {
	// Check database connectivity by getting the underlying sql.DB
	sqlDB, err := h.db.DB()
	if err != nil {
		traceID := getTraceIDFromContext(c)
		errorResponse := errors.NewErrorResponse(
			errors.SystemServiceUnavailable,
			traceID,
			errors.WithDetails("Database connection failed"),
		)
		return c.JSON(http.StatusServiceUnavailable, errorResponse)
	}

	if err := sqlDB.Ping(); err != nil {
		// Return SYSTEM_003 error for service unavailability
		traceID := getTraceIDFromContext(c)
		errorResponse := errors.NewErrorResponse(
			errors.SystemServiceUnavailable,
			traceID,
			errors.WithDetails("Database connection failed"),
		)
		return c.JSON(http.StatusServiceUnavailable, errorResponse)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// Helper to get trace ID from context
func getTraceIDFromContext(c echo.Context) string {
	traceID := c.Response().Header().Get("X-Trace-ID")
	if traceID == "" {
		if tid, ok := c.Get("trace_id").(string); ok {
			traceID = tid
		}
	}
	if traceID == "" {
		traceID = "unknown"
	}
	return traceID
}
