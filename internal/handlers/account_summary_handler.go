package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	apierrors "array-assessment/internal/errors"
	"array-assessment/internal/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AccountSummaryHandler struct {
	summaryService   services.AccountSummaryServiceInterface
	metricsService   services.AccountMetricsServiceInterface
	statementService services.StatementServiceInterface
}

func NewAccountSummaryHandler(
	summaryService services.AccountSummaryServiceInterface,
	metricsService services.AccountMetricsServiceInterface,
	statementService services.StatementServiceInterface,
) *AccountSummaryHandler {
	return &AccountSummaryHandler{
		summaryService:   summaryService,
		metricsService:   metricsService,
		statementService: statementService,
	}
}

// GetAccountSummary retrieves aggregated account information for a user
//
// Method: GET /api/v1/accounts/summary
// Authentication: Required (JWT)
//
// Query parameters:
//   - userId: UUID of target user (optional, admin only)
//
// Success Response: 200 OK
//   - user_id: UUID of the user
//   - total_balance: Decimal total across all accounts
//   - account_count: Integer number of accounts
//   - currency: String currency code
//   - accounts: Array of account summary items
//   - generated_at: ISO 8601 timestamp
//
// Error Responses:
//   - 400: Invalid userId format
//   - 401: Unauthorized (missing JWT)
//   - 403: Forbidden (non-admin accessing other user)
//   - 404: User not found
//   - 500: Internal server error
func (h *AccountSummaryHandler) GetAccountSummary(c echo.Context) error {
	requestorID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, apierrors.AuthMissingToken)
	}

	isAdmin := getIsAdminFromContext(c)

	var targetUserID *uuid.UUID
	userIDParam := c.QueryParam("userId")
	if userIDParam != "" {
		parsedUserID, err := uuid.Parse(userIDParam)
		if err != nil {
			return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("invalid userId format"))
		}
		targetUserID = &parsedUserID
	}

	summary, err := h.summaryService.GetAccountSummary(requestorID, targetUserID, isAdmin)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(http.StatusOK, SuccessResponse{
		Data: summary,
	})
}

// GetAccountMetrics retrieves performance metrics for an account
//
// Method: GET /api/v1/accounts/metrics
// Authentication: Required (JWT)
//
// Query parameters:
//   - accountId: UUID of account (required)
//   - startDate: ISO 8601 date (optional, defaults to 14 days ago)
//   - endDate: ISO 8601 date (optional, defaults to today)
//
// Success Response: 200 OK
//   - account_id: UUID of the account
//   - start_date: ISO 8601 date
//   - end_date: ISO 8601 date
//   - total_deposits: Decimal sum of deposits
//   - total_withdrawals: Decimal sum of withdrawals
//   - net_change: Decimal net change
//   - transaction_count: Integer total transactions
//   - deposit_count: Integer number of deposits
//   - withdrawal_count: Integer number of withdrawals
//   - average_transaction_amount: Decimal average amount
//   - largest_deposit: Decimal largest single deposit
//   - largest_withdrawal: Decimal largest single withdrawal
//   - average_daily_balance: Decimal average balance
//   - interest_earned: Decimal interest earned
//   - generated_at: ISO 8601 timestamp
//
// Error Responses:
//   - 400: Invalid parameters (accountId, date formats)
//   - 401: Unauthorized (missing JWT)
//   - 403: Forbidden (accessing other user's account)
//   - 404: Account not found
//   - 500: Internal server error
func (h *AccountSummaryHandler) GetAccountMetrics(c echo.Context) error {
	requestorID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	isAdmin := getIsAdminFromContext(c)

	accountIDParam := c.QueryParam("accountId")
	if accountIDParam == "" {
		return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("accountId is required"))
	}

	accountID, err := uuid.Parse(accountIDParam)
	if err != nil {
		return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("invalid accountId format"))
	}

	var startDate, endDate *time.Time

	startDateParam := c.QueryParam("startDate")
	if startDateParam != "" {
		parsed, err := time.Parse("2006-01-02", startDateParam)
		if err != nil {
			return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("invalid startDate format, expected YYYY-MM-DD"))
		}
		startDate = &parsed
	}

	endDateParam := c.QueryParam("endDate")
	if endDateParam != "" {
		parsed, err := time.Parse("2006-01-02", endDateParam)
		if err != nil {
			return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("invalid endDate format, expected YYYY-MM-DD"))
		}
		endDate = &parsed
	}

	metrics, err := h.metricsService.GetAccountMetrics(requestorID, accountID, startDate, endDate, isAdmin)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(http.StatusOK, SuccessResponse{
		Data: metrics,
	})
}

// GetStatement generates a monthly or quarterly account statement
//
// Method: GET /api/v1/accounts/:accountId/statements
// Authentication: Required (JWT)
//
// Path parameters:
//   - accountId: UUID of account
//
// Query parameters:
//   - periodType: "monthly" or "quarterly" (required)
//   - year: Integer year (required)
//   - period: Integer month (1-12) or quarter (1-4) (required)
//
// Success Response: 200 OK
//   - account_id: UUID of account
//   - account_number: String account number
//   - account_type: String account type
//   - period_type: String "monthly" or "quarterly"
//   - year: Integer year
//   - period: Integer period number
//   - start_date: ISO 8601 date
//   - end_date: ISO 8601 date
//   - opening_balance: Decimal opening balance
//   - closing_balance: Decimal closing balance
//   - transactions: Array of statement transactions
//   - performance_metrics: Object with account metrics
//   - summary: Object with aggregate summary
//   - generated_at: ISO 8601 timestamp
//
// Error Responses:
//   - 400: Invalid parameters (accountId, periodType, year, period)
//   - 401: Unauthorized (missing JWT)
//   - 403: Forbidden (accessing other user's account)
//   - 404: Account not found
//   - 500: Internal server error
func (h *AccountSummaryHandler) GetStatement(c echo.Context) error {
	requestorID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	isAdmin := getIsAdminFromContext(c)

	accountIDParam := c.Param("accountId")
	if accountIDParam == "" {
		return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("accountId is required"))
	}

	accountID, err := uuid.Parse(accountIDParam)
	if err != nil {
		return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("invalid accountId format"))
	}

	periodType := c.QueryParam("periodType")
	if periodType == "" {
		return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("periodType is required"))
	}

	if periodType != "monthly" && periodType != "quarterly" {
		return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("periodType must be 'monthly' or 'quarterly'"))
	}

	yearParam := c.QueryParam("year")
	periodParam := c.QueryParam("period")

	if yearParam == "" || periodParam == "" {
		return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("year and period are required"))
	}

	year, err := strconv.Atoi(yearParam)
	if err != nil {
		return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("invalid year format"))
	}

	period, err := strconv.Atoi(periodParam)
	if err != nil {
		return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("invalid period format"))
	}

	statement, err := h.statementService.GenerateStatement(requestorID, accountID, periodType, year, period, isAdmin)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(http.StatusOK, SuccessResponse{
		Data: statement,
	})
}

func (h *AccountSummaryHandler) handleServiceError(c echo.Context, err error) error {
	if errors.Is(err, services.ErrUnauthorized) {
		return SendError(c, apierrors.AuthInsufficientPermission)
	}

	if errors.Is(err, services.ErrNotFound) {
		return SendError(c, apierrors.AccountNotFound, apierrors.WithDetails("resource not found"))
	}

	if errors.Is(err, services.ErrInvalidDateRange) {
		return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("start date must be before end date"))
	}

	if errors.Is(err, services.ErrFutureDate) {
		return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("end date cannot be in the future"))
	}

	if errors.Is(err, services.ErrInvalidPeriodType) {
		return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("invalid period type"))
	}

	if errors.Is(err, services.ErrInvalidMonth) {
		return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("month must be between 1 and 12"))
	}

	if errors.Is(err, services.ErrInvalidQuarter) {
		return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("quarter must be between 1 and 4"))
	}

	if errors.Is(err, services.ErrFuturePeriod) {
		return SendError(c, apierrors.ValidationGeneral, apierrors.WithDetails("cannot generate statement for future period"))
	}

	return SendSystemError(c, err)
}
