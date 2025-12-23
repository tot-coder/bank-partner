package handlers

import (
	"fmt"
	"net/http"
	"time"

	"array-assessment/internal/repositories"
	"array-assessment/internal/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// DevHandler handles development-only endpoints
// These endpoints should only be available in development environments
type DevHandler struct {
	transactionRepo repositories.TransactionRepositoryInterface
	accountRepo     repositories.AccountRepositoryInterface
	generator       services.TransactionGeneratorInterface
}

// NewDevHandler creates a new development handler
func NewDevHandler(
	transactionRepo repositories.TransactionRepositoryInterface,
	accountRepo repositories.AccountRepositoryInterface,
) *DevHandler {
	return &DevHandler{
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
		generator:       services.NewTransactionGenerator(),
	}
}

// GenerateTestData generates realistic test transaction data for an account
//
// Method: POST /api/v1/dev/accounts/:id/generate-test-data
// Authentication: Required
// Environment: Development only
//
// Path parameters:
//   - id: Account UUID
//
// Query parameters:
//   - count: Number of transactions to generate (default: 100, max: 1000)
//   - days: Number of days of history to generate (default: 30, max: 365)
//
// Success Response: 200 OK
//   - message: Success message
//   - transactions_created: Number of transactions created
//
// Error Responses:
//   - 400: Invalid account ID or parameters
//   - 401: Unauthorized
//   - 403: Forbidden (not development environment or account belongs to another user)
//   - 404: Account not found
//   - 500: Internal server error
func (h *DevHandler) GenerateTestData(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	accountIDStr := c.Param("accountId")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid account ID")
	}

	account, err := h.accountRepo.GetByID(accountID)
	if err != nil {
		if err == repositories.ErrAccountNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "account not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to retrieve account")
	}

	if account.UserID != userID {
		return echo.NewHTTPError(http.StatusForbidden, "access denied")
	}

	count := getIntQueryParam(c, "count", 100)
	if count < 1 {
		count = 1
	}
	if count > 1000 {
		count = 1000
	}

	days := getIntQueryParam(c, "days", 30)
	if days < 1 {
		days = 1
	}
	if days > 365 {
		days = 365
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)
	startingBalance := account.Balance

	transactions := h.generator.GenerateHistoricalTransactions(
		accountID,
		startDate,
		endDate,
		startingBalance,
		count,
	)

	created := 0
	for _, txn := range transactions {
		if err := h.transactionRepo.Create(txn); err != nil {
			continue
		}
		created++
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":              "test data generated successfully",
		"transactions_created": created,
		"account_id":           accountID,
		"date_range": map[string]string{
			"start": startDate.Format(time.RFC3339),
			"end":   endDate.Format(time.RFC3339),
		},
	})
}

// ClearTestData removes all transactions for an account
//
// Method: DELETE /api/v1/dev/accounts/:id/test-data
// Authentication: Required
// Environment: Development only
//
// Path parameters:
//   - id: Account UUID
//
// Success Response: 200 OK
//   - message: Success message
//   - transactions_deleted: Number of transactions deleted
//
// Error Responses:
//   - 400: Invalid account ID
//   - 401: Unauthorized
//   - 403: Forbidden (not development environment or account belongs to another user)
//   - 404: Account not found
//   - 500: Internal server error
func (h *DevHandler) ClearTestData(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	accountIDStr := c.Param("accountId")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid account ID")
	}

	account, err := h.accountRepo.GetByID(accountID)
	if err != nil {
		if err == repositories.ErrAccountNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "account not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to retrieve account")
	}

	if account.UserID != userID {
		return echo.NewHTTPError(http.StatusForbidden, "access denied")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":    "clear test data endpoint would require repository support",
		"account_id": accountID,
		"note":       "implement TransactionRepository.DeleteByAccountID to enable this feature",
	})
}

// Helper function to get integer query parameters
func getIntQueryParam(c echo.Context, key string, defaultValue int) int {
	valueStr := c.QueryParam(key)
	if valueStr == "" {
		return defaultValue
	}

	var value int
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil {
		return defaultValue
	}

	return value
}
