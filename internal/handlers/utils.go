package handlers

import (
	"fmt"
	"strings"

	"array-assessment/internal/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

// ErrUnauthorized is returned when user context is invalid
var ErrUnauthorized = fmt.Errorf("unauthorized")

// Helper function to extract user ID from context
// Returns ErrUnauthorized if user ID is missing or invalid
func getUserIDFromContext(c echo.Context) (uuid.UUID, error) {
	userIDValue := c.Get("user_id")
	if userIDValue == nil {
		return uuid.UUID{}, ErrUnauthorized
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		return uuid.UUID{}, ErrUnauthorized
	}

	return userID, nil
}

func getAvailableBalanceFromContext(c echo.Context) decimal.Decimal {
	availableBalanceValue := c.Get("initialDeposit")
	if availableBalanceValue == nil {
		return decimal.NewFromInt(0)
	}

	availableBalance, ok := availableBalanceValue.(decimal.Decimal)
	if !ok {
		return decimal.NewFromInt(0)
	}

	return availableBalance
}

func getAccountNumberFromContext(c echo.Context) string {
	accountNumberValue := c.Get("accountNumber")
	if accountNumberValue == nil {
		return ""
	}

	accountNumber, ok := accountNumberValue.(string)
	if !ok {
		return ""
	}

	return accountNumber
}

func getRoutingNumberFromContext(c echo.Context) string {
	routingNumberValue := c.Get("routingNumber")
	if routingNumberValue == nil {
		return ""
	}

	routingNumber, ok := routingNumberValue.(string)
	if !ok {
		return ""
	}

	return routingNumber
}

// getIsAdminFromContext extracts the is_admin boolean from context
// Returns false if the value is not set or not a boolean
func getIsAdminFromContext(c echo.Context) bool {
	isAdminValue := c.Get("is_admin")
	if isAdminValue == nil {
		return false
	}

	isAdmin, ok := isAdminValue.(bool)
	if !ok {
		return false
	}

	return isAdmin
}

func (h *AdminHandler) createAuditLog(adminID uuid.UUID, action, targetUserID string, c echo.Context) {
	m := models.JSONBMap{
		"target_user_id": targetUserID,
	}

	log := &models.AuditLog{
		UserID:    &adminID,
		Action:    action,
		IPAddress: getClientIP(c),
		UserAgent: c.Request().UserAgent(),
		Metadata:  m,
	}

	if err := h.auditRepo.Create(log); err != nil {
		// Audit logging failure should not block the operation
		// Log error to monitoring system in production
		_ = err
	}
}

func getIntParam(c echo.Context, name string, defaultValue int) int {
	param := c.QueryParam(name)
	if param == "" {
		return defaultValue
	}

	var value int
	if _, err := fmt.Sscanf(param, "%d", &value); err != nil {
		return defaultValue
	}

	return value
}

func getClientIP(c echo.Context) string {
	xff := c.Request().Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	xri := c.Request().Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	return c.Request().RemoteAddr
}
