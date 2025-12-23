package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"array-assessment/internal/dto"
	"array-assessment/internal/errors"
	"array-assessment/internal/models"
	"array-assessment/internal/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

// AccountHandler handles account-related HTTP requests
type AccountHandler struct {
	accountService   services.AccountServiceInterface
	auditLogger      services.AuditLoggerInterface
	metricsCollector services.MetricsRecorderInterface
}

// NewAccountHandler creates a new account handler
func NewAccountHandler(accountService services.AccountServiceInterface, auditLogger services.AuditLoggerInterface, metricsCollector services.MetricsRecorderInterface) *AccountHandler {
	return &AccountHandler{
		accountService:   accountService,
		auditLogger:      auditLogger,
		metricsCollector: metricsCollector,
	}
}

// CreateAccount creates a new bank account for the authenticated user
// @Summary Create a new account
// @Description Create a new bank account (checking, savings, or money_market) with optional initial deposit
// @Tags Accounts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.CreateAccountRequest true "Account creation details"
// @Success 201 {object} dto.CreateAccountResponse "Account created successfully"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_001 - Invalid request body or validation error"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 422 {object} errors.ErrorResponse "TRANSACTION_002 - Invalid initial deposit amount"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /accounts [post]
func (h *AccountHandler) CreateAccount(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	var req dto.CreateAccountRequest
	if err := c.Bind(&req); err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("Invalid request body"))
	}

	if err := c.Validate(req); err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails(err.Error()))
	}

	initialDeposit := decimal.Zero
	if req.InitialDeposit != "" {
		initialDeposit, err = decimal.NewFromString(req.InitialDeposit)
		if err != nil {
			return SendError(c, errors.ValidationInvalidFormat, errors.WithDetails("Invalid initial deposit amount"))
		}
	}

	account, err := h.accountService.CreateAccount(userID, req.AccountType, initialDeposit)
	if err != nil {
		if err == services.ErrAccountAlreadyExists {
			return SendError(c, errors.ValidationGeneral, errors.WithDetails(err.Error()))
		}
		if err == services.ErrInvalidAmount {
			return SendError(c, errors.TransactionInvalidAmount, errors.WithDetails(err.Error()))
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusCreated, dto.CreateAccountResponse{
		Account: account,
		Message: "Account created successfully",
	})
}

// GetAccount retrieves a specific account by ID
// @Summary Get account by ID
// @Description Retrieve detailed information about a specific account belonging to the authenticated user
// @Tags Accounts
// @Security BearerAuth
// @Produce json
// @Param accountId path string true "Account ID (UUID)"
// @Success 200 {object} models.Account "Account details"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_003 - Invalid account ID format"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Account belongs to another user"
// @Failure 404 {object} errors.ErrorResponse "ACCOUNT_001 - Account not found"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /accounts/{accountId} [get]
func (h *AccountHandler) GetAccount(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	accountIDStr := c.Param("accountId")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return SendError(c, errors.ValidationInvalidFormat, errors.WithDetails("Invalid account ID"))
	}

	account, err := h.accountService.GetAccountByID(accountID, &userID)
	if err != nil {
		if err == services.ErrAccountNotFound {
			return SendError(c, errors.AccountNotFound)
		}
		if err == services.ErrUnauthorized {
			return SendError(c, errors.AuthInsufficientPermission)
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, account)
}

// GetUserAccounts retrieves all accounts for the authenticated user
// @Summary Get all user accounts
// @Description Retrieve all bank accounts belonging to the authenticated user
// @Tags Accounts
// @Security BearerAuth
// @Produce json
// @Success 200 {array} models.Account "List of user's accounts"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /accounts [get]
func (h *AccountHandler) GetUserAccounts(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	accounts, err := h.accountService.GetUserAccounts(userID)
	if err != nil {
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, accounts)
}

// UpdateAccountStatus updates the status of a specific account
// @Summary Update account status
// @Description Update the status of an account (active, inactive, frozen, closed)
// @Tags Accounts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param accountId path string true "Account ID (UUID)"
// @Param request body dto.UpdateAccountStatusRequest true "New account status"
// @Success 200 {object} models.Account "Updated account details"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_001 - Invalid request body or account ID"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Account belongs to another user"
// @Failure 404 {object} errors.ErrorResponse "ACCOUNT_001 - Account not found"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /accounts/{accountId}/status [patch]
func (h *AccountHandler) UpdateAccountStatus(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	accountIDStr := c.Param("accountId")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return SendError(c, errors.ValidationInvalidFormat, errors.WithDetails("Invalid account ID"))
	}

	var req dto.UpdateAccountStatusRequest
	if err := c.Bind(&req); err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("Invalid request body"))
	}

	if err := c.Validate(req); err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails(err.Error()))
	}

	account, err := h.accountService.UpdateAccountStatus(accountID, &userID, req.Status)
	if err != nil {
		if err == services.ErrAccountNotFound {
			return SendError(c, errors.AccountNotFound)
		}
		if err == services.ErrUnauthorized {
			return SendError(c, errors.AuthInsufficientPermission)
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, account)
}

// CloseAccount permanently closes an account
// @Summary Close account
// @Description Permanently close an account. Account must have zero balance to be closed.
// @Tags Accounts
// @Security BearerAuth
// @Produce json
// @Param accountId path string true "Account ID (UUID)"
// @Success 200 {object} SuccessResponse{message=string} "Account closed successfully"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_003 - Invalid account ID"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Account belongs to another user"
// @Failure 404 {object} errors.ErrorResponse "ACCOUNT_001 - Account not found"
// @Failure 422 {object} errors.ErrorResponse "ACCOUNT_005 - Account has non-zero balance"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /accounts/{accountId} [delete]
func (h *AccountHandler) CloseAccount(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	accountIDStr := c.Param("accountId")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return SendError(c, errors.ValidationInvalidFormat, errors.WithDetails("Invalid account ID"))
	}

	err = h.accountService.CloseAccount(accountID, userID)
	if err != nil {
		if err == services.ErrAccountNotFound {
			return SendError(c, errors.AccountNotFound)
		}
		if err == services.ErrUnauthorized {
			return SendError(c, errors.AuthInsufficientPermission)
		}
		if err == services.ErrAccountClosureNotAllowed {
			return SendError(c, errors.AccountOperationNotPermitted, errors.WithDetails(err.Error()))
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Account closed successfully",
	})
}

// PerformTransaction creates a new transaction on an account
// @Summary Create a transaction
// @Description Create a new transaction (credit or debit) on an account
// @Tags Accounts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param accountId path string true "Account ID (UUID)"
// @Param request body dto.TransactionRequest true "Transaction details"
// @Success 201 {object} models.Transaction "Transaction created successfully"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_001 - Invalid request body or account ID"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Account belongs to another user"
// @Failure 404 {object} errors.ErrorResponse "ACCOUNT_001 - Account not found"
// @Failure 422 {object} errors.ErrorResponse "TRANSACTION_002 - Invalid transaction amount, TRANSACTION_003 - Insufficient funds, ACCOUNT_002 - Account not active"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /accounts/{accountId}/transactions [post]
func (h *AccountHandler) PerformTransaction(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	accountIDStr := c.Param("accountId")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return SendError(c, errors.AccountInvalidNumber, errors.WithDetails("Account ID must be a valid UUID"))
	}

	var req dto.TransactionRequest
	if err := c.Bind(&req); err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("Invalid request body"))
	}

	if err := c.Validate(req); err != nil {
		return err
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		return SendError(c, errors.TransactionInvalidAmount, errors.WithDetails("Amount must be greater than 0"))
	}

	transaction, err := h.accountService.PerformTransaction(accountID, amount, req.Type, req.Description, &userID)
	if err != nil {
		return mapTransactionErr(c, err)
	}

	return c.JSON(http.StatusCreated, transaction)
}

// Transfer performs an atomic transfer between user's accounts with idempotency support
// @Summary Transfer between accounts
// @Description Perform an atomic transfer between user's accounts. Requires Idempotency-Key header. Both accounts must belong to the authenticated user.
// @Tags Accounts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param accountId path string true "Source Account ID (UUID)"
// @Param Idempotency-Key header string true "Unique key to ensure idempotent transfers"
// @Param request body dto.TransferRequest true "Transfer details"
// @Success 200 {object} dto.TransferResponse "Transfer completed successfully"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_001 - Invalid request body, VALIDATION_002 - Missing Idempotency-Key header"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Account belongs to another user"
// @Failure 404 {object} errors.ErrorResponse "ACCOUNT_001 - Account not found"
// @Failure 409 {object} errors.ErrorResponse "Duplicate idempotency key with pending or failed transfer"
// @Failure 422 {object} errors.ErrorResponse "TRANSACTION_002 - Invalid amount, TRANSACTION_003 - Insufficient funds, ACCOUNT_002 - Account not active"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /accounts/{accountId}/transfer [post]
func (h *AccountHandler) Transfer(c echo.Context) error {
	startTime := time.Now()
	ctx := c.Request().Context()

	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	idempotencyKey := c.Request().Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		return SendError(c, errors.ValidationRequiredField, errors.WithDetails("Idempotency-Key header is required"))
	}

	fromAccountIDStr := c.Param("accountId")
	fromAccountID, err := uuid.Parse(fromAccountIDStr)
	if err != nil {
		return SendError(c, errors.ValidationInvalidFormat, errors.WithDetails("Invalid account ID"))
	}

	var req dto.TransferRequest
	if err := c.Bind(&req); err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("Invalid request body"))
	}

	if err := c.Validate(req); err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails(err.Error()))
	}

	toAccountID, err := uuid.Parse(req.ToAccountID)
	if err != nil {
		return SendError(c, errors.ValidationInvalidFormat, errors.WithDetails("Invalid destination account ID"))
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return SendError(c, errors.TransactionInvalidAmount, errors.WithDetails("Invalid amount"))
	}

	tempTransferID := uuid.New()
	if h.auditLogger != nil {
		h.auditLogger.LogTransferInitiated(ctx, tempTransferID, fromAccountID, toAccountID, req.Amount, idempotencyKey, userID)
	}

	transfer, err := h.accountService.TransferBetweenAccounts(fromAccountID, toAccountID, amount, req.Description, idempotencyKey, userID)
	duration := time.Since(startTime)

	if err != nil {
		if h.auditLogger != nil && transfer != nil {
			h.auditLogger.LogTransferFailed(ctx, transfer.ID, err.Error(), duration.Milliseconds())
		}

		if h.metricsCollector != nil {
			h.metricsCollector.IncrementCounter("transfers_total", map[string]string{"status": "failed"})
			h.metricsCollector.RecordProcessingTime("transfer_duration_failed", duration)
		}

		return h.mapTransferErr(c, ctx, transfer, idempotencyKey, err)
	}

	if h.auditLogger != nil {
		h.auditLogger.LogTransferCompleted(ctx, transfer.ID, duration.Milliseconds(), transfer.DebitTransactionID, transfer.CreditTransactionID)
	}

	if h.metricsCollector != nil {
		h.metricsCollector.IncrementCounter("transfers_total", map[string]string{"status": "completed"})
		h.metricsCollector.RecordProcessingTime("transfer_duration_success", duration)

		amountFloat, _ := amount.Float64()
		h.metricsCollector.RecordGauge("transfer_amount", amountFloat, nil)
	}

	response := dto.TransferResponse{
		Message:       "Transfer completed successfully",
		TransferID:    transfer.ID.String(),
		FromAccountID: transfer.FromAccountID.String(),
		ToAccountID:   transfer.ToAccountID.String(),
		Amount:        transfer.Amount.String(),
	}

	if transfer.DebitTransactionID != nil {
		debitTxID := transfer.DebitTransactionID.String()
		response.DebitTransactionID = &debitTxID
	}

	if transfer.CreditTransactionID != nil {
		creditTxID := transfer.CreditTransactionID.String()
		response.CreditTransactionID = &creditTxID
	}

	return c.JSON(http.StatusOK, response)
}

// GetTransferHistory retrieves transfer history for the authenticated user
// @Summary Get my transfer history
// @Description Retrieve paginated transfer history for the authenticated user with optional status filter
// @Tags Customers
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Results per page (max 100)" default(20)
// @Param status query string false "Filter by status" Enums(completed, failed, pending)
// @Success 200 {object} dto.TransferHistoryResponse "Transfer history with pagination"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /customers/me/transfers [get]
func (h *AccountHandler) GetTransferHistory(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page <= 0 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	filters := models.TransferFilters{
		Status: c.QueryParam("status"),
	}

	transfers, total, err := h.accountService.GetUserTransfers(userID, filters, offset, limit)
	if err != nil {
		return SendSystemError(c, err)
	}

	response := dto.TransferHistoryResponse{
		Transfers: transfers,
		Pagination: dto.PaginationMeta{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	}

	return c.JSON(http.StatusOK, response)
}

// Admin endpoints

// GetAllAccounts retrieves all accounts across all users (admin only)
// @Summary Get all accounts (admin)
// @Description Admin endpoint to retrieve all accounts with optional filters
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Number of results (max 100)" default(20)
// @Param user_id query string false "Filter by user ID (UUID)"
// @Param account_type query string false "Filter by account type" Enums(checking, savings, money_market)
// @Param status query string false "Filter by status" Enums(active, inactive, frozen, closed)
// @Success 200 {object} object{accounts=[]models.Account,total=int,offset=int,limit=int} "List of all accounts"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /admin/accounts [get]
func (h *AccountHandler) GetAllAccounts(c echo.Context) error {
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var filters models.AccountFilters
	if userIDStr := c.QueryParam("user_id"); userIDStr != "" {
		userID, err := uuid.Parse(userIDStr)
		if err == nil {
			filters.UserID = &userID
		}
	}
	filters.AccountType = c.QueryParam("account_type")
	filters.Status = c.QueryParam("status")

	accounts, total, err := h.accountService.GetAllAccounts(filters, offset, limit)
	if err != nil {
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"accounts": accounts,
		"total":    total,
		"offset":   offset,
		"limit":    limit,
	})
}

// GetAccountByIDAdmin retrieves any account by ID (admin only)
// @Summary Get account by ID (admin)
// @Description Admin endpoint to retrieve any account by ID without ownership check
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param accountId path string true "Account ID (UUID)"
// @Success 200 {object} models.Account "Account details"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_003 - Invalid account ID format"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 404 {object} errors.ErrorResponse "ACCOUNT_001 - Account not found"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /admin/accounts/{accountId} [get]
func (h *AccountHandler) GetAccountByIDAdmin(c echo.Context) error {
	accountIDStr := c.Param("accountId")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return SendError(c, errors.ValidationInvalidFormat, errors.WithDetails("Invalid account ID"))
	}

	account, err := h.accountService.GetAccountByID(accountID, nil)
	if err != nil {
		if err == services.ErrAccountNotFound {
			return SendError(c, errors.AccountNotFound)
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, account)
}

// GetUserAccountsAdmin retrieves all accounts for a specific user (admin only)
// @Summary Get user accounts by user ID (admin)
// @Description Admin endpoint to retrieve all accounts for a specific user
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param userId path string true "User ID (UUID)"
// @Success 200 {array} models.Account "List of user's accounts"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_003 - Invalid user ID format"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /admin/users/{userId}/accounts [get]
func (h *AccountHandler) GetUserAccountsAdmin(c echo.Context) error {
	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return SendError(c, errors.ValidationInvalidFormat, errors.WithDetails("Invalid user ID"))
	}

	accounts, err := h.accountService.GetUserAccounts(userID)
	if err != nil {
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, accounts)
}

func mapCommonErr(c echo.Context, err error) error {
	if err == services.ErrAccountNotFound {
		return SendError(c, errors.AccountNotFound)
	}
	if err == services.ErrUnauthorized {
		return SendError(c, errors.AuthInsufficientPermission)
	}
	if err == services.ErrAccountNotActive {
		return SendError(c, errors.AccountInactive)
	}
	return nil
}

func mapTransactionErr(c echo.Context, err error) error {
	if mappedErr := mapCommonErr(c, err); mappedErr != nil {
		return mappedErr
	}
	if err == services.ErrInsufficientFunds {
		return SendError(c, errors.TransactionInsufficientFunds)
	}
	if err == services.ErrInvalidAmount {
		return SendError(c, errors.TransactionInvalidAmount)
	}

	return SendSystemError(c, err)
}

func (h *AccountHandler) mapTransferErr(c echo.Context, ctx context.Context, transfer *models.Transfer, idempotencyKey string, svcErr error) error {
	if mappedErr := mapCommonErr(c, svcErr); mappedErr != nil {
		return mappedErr
	}
	if svcErr == services.ErrInsufficientFunds {
		return SendError(c, errors.TransferInsufficientFunds)
	}
	if svcErr == services.ErrInvalidAmount {
		return SendError(c, errors.TransferInvalidAmount)
	}
	if svcErr == services.ErrSameAccountTransfer {
		return SendError(c, errors.TransferSameAccount)
	}
	if svcErr == services.ErrTransferPending {
		if h.auditLogger != nil && transfer != nil {
			h.auditLogger.LogTransferIdempotencyCheck(ctx, idempotencyKey, transfer.ID, "pending")
		}
		return SendError(c, errors.TransferPending)
	}
	if svcErr == services.ErrTransferFailed {
		if h.auditLogger != nil && transfer != nil {
			h.auditLogger.LogTransferIdempotencyCheck(ctx, idempotencyKey, transfer.ID, "failed")
		}
		return SendError(c, errors.TransferFailed)
	}

	return SendSystemError(c, svcErr)
}
