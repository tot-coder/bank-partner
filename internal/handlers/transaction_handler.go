package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"array-assessment/internal/dto"
	"array-assessment/internal/errors"
	"array-assessment/internal/models"
	"array-assessment/internal/repositories"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
	cacheTTL         = 5 * time.Minute
)

// TransactionHandler handles transaction-related HTTP requests
type TransactionHandler struct {
	transactionRepo repositories.TransactionRepositoryInterface
	accountRepo     repositories.AccountRepositoryInterface
}

// NewTransactionHandler creates a new transaction handler
func NewTransactionHandler(
	transactionRepo repositories.TransactionRepositoryInterface,
	accountRepo repositories.AccountRepositoryInterface,
) *TransactionHandler {
	return &TransactionHandler{
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
	}
}

// cursorData represents the data encoded in a pagination cursor
type cursorData struct {
	Timestamp     time.Time `json:"timestamp"`
	TransactionID uuid.UUID `json:"transaction_id"`
}

// encodeCursor creates a cursor string from timestamp and transaction ID
func encodeCursor(timestamp time.Time, transactionID uuid.UUID) string {
	data := cursorData{
		Timestamp:     timestamp,
		TransactionID: transactionID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}

	return base64.URLEncoding.EncodeToString(jsonData)
}

// decodeCursor decodes a cursor string to timestamp and transaction ID
func decodeCursor(cursor string) (time.Time, uuid.UUID, error) {
	if cursor == "" {
		return time.Time{}, uuid.Nil, fmt.Errorf("empty cursor")
	}

	jsonData, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}

	var data cursorData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor format: %w", err)
	}

	return data.Timestamp, data.TransactionID, nil
}

// ListTransactions retrieves paginated transaction history with filtering
// @Summary List transactions
// @Description Retrieve paginated and filtered transaction history for a specific account with cursor-based pagination
// @Tags Transactions
// @Security BearerAuth
// @Produce json
// @Param accountId path string true "Account ID (UUID)"
// @Param cursor query string false "Pagination cursor for next page"
// @Param limit query int false "Number of results per page (max 100)" default(20)
// @Param start_date query string false "Filter by start date (YYYY-MM-DD)"
// @Param end_date query string false "Filter by end date (YYYY-MM-DD)"
// @Param type query string false "Filter by transaction type" Enums(credit, debit)
// @Param status query string false "Filter by status" Enums(pending, completed, failed, reversed)
// @Param category query string false "Filter by category code"
// @Param min_amount query string false "Filter by minimum amount"
// @Param max_amount query string false "Filter by maximum amount"
// @Param merchant query string false "Filter by merchant name"
// @Success 200 {object} dto.ListTransactionsResponse "Transaction history with pagination"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_001 - Invalid parameters or VALIDATION_003 - Invalid account ID"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Account belongs to another user"
// @Failure 404 {object} errors.ErrorResponse "ACCOUNT_001 - Account not found"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /accounts/{accountId}/transactions [get]
func (h *TransactionHandler) ListTransactions(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	accountIDStr := c.Param("accountId")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return SendError(c, errors.ValidationInvalidFormat, errors.WithDetails("Invalid account ID"))
	}

	account, err := h.accountRepo.GetByID(accountID)
	if err != nil {
		if err == repositories.ErrAccountNotFound {
			return SendError(c, errors.AccountNotFound)
		}
		return SendSystemError(c, err)
	}

	if account.UserID != userID {
		return SendError(c, errors.AuthInsufficientPermission)
	}

	filters, err := parseTransactionFilters(c)
	if err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails(err.Error()))
	}
	filters.AccountID = accountID

	pagination, err := parsePaginationParams(c)
	if err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails(err.Error()))
	}

	if pagination.Cursor != "" {
		cursorTime, cursorID, err := decodeCursor(pagination.Cursor)
		if err != nil {
			return SendError(c, errors.ValidationInvalidFormat, errors.WithDetails("Invalid cursor"))
		}
		filters.StartDate = &cursorTime
		storeCursorTransactionIdForExclusion(c, cursorID)
	}

	transactions, total, err := h.transactionRepo.GetWithFilters(filters)
	if err != nil {
		return SendSystemError(c, err)
	}

	cursorTransactionID, _ := c.Get("cursorTransactionID").(uuid.UUID)
	if cursorTransactionID != uuid.Nil && len(transactions) > 0 {
		if transactions[0].ID == cursorTransactionID {
			transactions = transactions[1:]
		}
	}

	var nextCursor string
	hasMore := false

	if len(transactions) > pagination.Limit {
		hasMore = true
		transactions = transactions[:pagination.Limit]
		lastTxn := &transactions[len(transactions)-1]
		nextCursor = encodeCursor(lastTxn.CreatedAt, lastTxn.ID)
	}

	response := dto.ListTransactionsResponse{
		Transactions: convertToTransactionWithBalance(transactions),
		Pagination: dto.PaginationInfo{
			HasMore:    hasMore,
			NextCursor: nextCursor,
			Limit:      pagination.Limit,
			Total:      total,
		},
	}

	c.Response().Header().Set("Cache-Control", fmt.Sprintf("private, max-age=%d", int(cacheTTL.Seconds())))

	return c.JSON(http.StatusOK, response)
}

func storeCursorTransactionIdForExclusion(c echo.Context, cursorID uuid.UUID) {
	c.Set("cursorTransactionID", cursorID)
}

// parseTransactionFilters parses and validates transaction filter parameters
func parseTransactionFilters(c echo.Context) (models.TransactionFilters, error) {
	filters := models.TransactionFilters{
		Offset: 0,
		Limit:  defaultPageLimit + 1, // Fetch one extra to determine if there's more
	}

	if startDateStr := c.QueryParam("start_date"); startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return filters, fmt.Errorf("invalid start_date format, use YYYY-MM-DD")
		}
		filters.StartDate = &startDate
	}

	if endDateStr := c.QueryParam("end_date"); endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return filters, fmt.Errorf("invalid end_date format, use YYYY-MM-DD")
		}
		// Set to end of day
		endOfDay := endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		filters.EndDate = &endOfDay
	}

	if txnType := c.QueryParam("type"); txnType != "" {
		if txnType != models.TransactionTypeCredit && txnType != models.TransactionTypeDebit {
			return filters, fmt.Errorf("invalid type, must be 'credit' or 'debit'")
		}
		filters.Type = txnType
	}

	if status := c.QueryParam("status"); status != "" {
		validStatuses := map[string]bool{
			models.TransactionStatusPending:   true,
			models.TransactionStatusCompleted: true,
			models.TransactionStatusFailed:    true,
			models.TransactionStatusReversed:  true,
		}
		if !validStatuses[status] {
			return filters, fmt.Errorf("invalid status")
		}
		filters.Status = status
	}

	if category := c.QueryParam("category"); category != "" {
		if !models.IsValidCategory(category) {
			return filters, fmt.Errorf("invalid category")
		}
		filters.Category = category
	}

	if minAmountStr := c.QueryParam("min_amount"); minAmountStr != "" {
		minAmount, err := decimal.NewFromString(minAmountStr)
		if err != nil {
			return filters, fmt.Errorf("invalid min_amount format")
		}
		filters.MinAmount = &minAmount
	}

	if maxAmountStr := c.QueryParam("max_amount"); maxAmountStr != "" {
		maxAmount, err := decimal.NewFromString(maxAmountStr)
		if err != nil {
			return filters, fmt.Errorf("invalid max_amount format")
		}
		filters.MaxAmount = &maxAmount
	}

	if merchant := c.QueryParam("merchant"); merchant != "" {
		filters.MerchantName = merchant
	}

	return filters, nil
}

// parsePaginationParams parses pagination parameters from query string
func parsePaginationParams(c echo.Context) (dto.PaginationParams, error) {
	params := dto.PaginationParams{
		Limit: defaultPageLimit,
	}

	if cursor := c.QueryParam("cursor"); cursor != "" {
		params.Cursor = cursor
	}

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return params, fmt.Errorf("invalid limit parameter")
		}

		if limit < 1 {
			return params, fmt.Errorf("limit must be at least 1")
		}

		if limit > maxPageLimit {
			limit = maxPageLimit
		}

		params.Limit = limit
	}

	return params, nil
}

// convertToTransactionWithBalance converts transaction models to DTOs with running balance
func convertToTransactionWithBalance(transactions []models.Transaction) []dto.TransactionWithBalance {
	result := make([]dto.TransactionWithBalance, 0, len(transactions))

	for i := range transactions {
		txn := &transactions[i]
		result = append(result, dto.TransactionWithBalance{
			ID:              txn.ID,
			AccountID:       txn.AccountID,
			Amount:          txn.Amount.String(),
			TransactionType: txn.TransactionType,
			Description:     txn.Description,
			Status:          txn.Status,
			Category:        txn.Category,
			MerchantName:    txn.MerchantName,
			MCCCode:         txn.MCCCode,
			Reference:       txn.Reference,
			RunningBalance:  txn.BalanceAfter.String(),
			CreatedAt:       txn.CreatedAt,
			ProcessedAt:     txn.ProcessedAt,
		})
	}

	return result
}

// GetTransaction retrieves a specific transaction by ID
// @Summary Get transaction by ID
// @Description Retrieve detailed information about a specific transaction including running balance
// @Tags Transactions
// @Security BearerAuth
// @Produce json
// @Param accountId path string true "Account ID (UUID)"
// @Param id path string true "Transaction ID (UUID)"
// @Success 200 {object} dto.TransactionWithBalance "Transaction details with running balance"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_001 - Invalid transaction or account ID format"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Transaction belongs to another user's account"
// @Failure 404 {object} errors.ErrorResponse "ACCOUNT_001 - Account not found or TRANSACTION_001 - Transaction not found"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /accounts/{accountId}/transactions/{id} [get]
func (h *TransactionHandler) GetTransaction(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	accountIDStr := c.Param("accountId")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return SendError(c, errors.AccountInvalidNumber, errors.WithDetails("Account ID must be a valid UUID"))
	}

	account, err := h.accountRepo.GetByID(accountID)
	if err != nil {
		if err == repositories.ErrAccountNotFound {
			return SendError(c, errors.AccountNotFound)
		}
		return SendSystemError(c, err)
	}

	if account.UserID != userID {
		return SendError(c, errors.AuthInsufficientPermission)
	}

	transactionIDStr := c.Param("id")
	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("Transaction ID must be a valid UUID"))
	}

	transaction, err := h.transactionRepo.GetByID(transactionID)
	if err != nil {
		if err == repositories.ErrTransactionNotFound {
			return SendError(c, errors.TransactionNotFound)
		}
		return SendSystemError(c, err)
	}

	if transaction.AccountID != accountID {
		return SendError(c, errors.AuthInsufficientPermission,
			errors.WithDetails("Transaction does not belong to this account"))
	}

	response := dto.TransactionWithBalance{
		ID:              transaction.ID,
		AccountID:       transaction.AccountID,
		Amount:          transaction.Amount.String(),
		TransactionType: transaction.TransactionType,
		Description:     transaction.Description,
		Status:          transaction.Status,
		Category:        transaction.Category,
		MerchantName:    transaction.MerchantName,
		MCCCode:         transaction.MCCCode,
		Reference:       transaction.Reference,
		RunningBalance:  transaction.BalanceAfter.String(),
		CreatedAt:       transaction.CreatedAt,
		ProcessedAt:     transaction.ProcessedAt,
	}

	c.Response().Header().Set("Cache-Control", fmt.Sprintf("private, max-age=%d", int(cacheTTL.Seconds())))

	return c.JSON(http.StatusOK, response)
}
