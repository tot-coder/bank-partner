package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"array-assessment/internal/dto"
	"array-assessment/internal/models"
	"array-assessment/internal/services"
	"array-assessment/internal/services/service_mocks"

	"github.com/go-playground/validator/v10"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

// AccountHandlerSuite defines the test suite for AccountHandler
type AccountHandlerSuite struct {
	suite.Suite
	ctrl             *gomock.Controller
	mockService      *service_mocks.MockAccountServiceInterface
	handler          *AccountHandler
	echo             *echo.Echo
	testUserID       uuid.UUID
	testAdminID      uuid.UUID
	auditLogger      *service_mocks.MockAuditLoggerInterface
	metricsCollector *service_mocks.MockMetricsRecorderInterface
}

// SetupTest runs before each test in the suite
func (s *AccountHandlerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockService = service_mocks.NewMockAccountServiceInterface(s.ctrl)
	s.auditLogger = service_mocks.NewMockAuditLoggerInterface(s.ctrl)
	s.metricsCollector = service_mocks.NewMockMetricsRecorderInterface(s.ctrl)
	s.handler = NewAccountHandler(s.mockService, s.auditLogger, s.metricsCollector)

	s.echo = echo.New()
	s.echo.Validator = &CustomValidator{validator: validator.New()}

	// Setup common test data
	s.testUserID = uuid.New()
	s.testAdminID = uuid.New()
}

// TearDownTest runs after each test in the suite
func (s *AccountHandlerSuite) TearDownTest() {
	s.ctrl.Finish()
}

// TestAccountHandlerSuite runs the test suite
func TestAccountHandlerSuite(t *testing.T) {
	suite.Run(t, new(AccountHandlerSuite))
}

// Helper method to create test context with authentication
func (s *AccountHandlerSuite) createContextWithAuth(method, path string, body interface{}, userID uuid.UUID, userRole string) (echo.Context, *httptest.ResponseRecorder) {
	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewBuffer(jsonBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	// Set authenticated user context
	c.Set("user_id", userID)
	c.Set("user_role", userRole)

	return c, rec
}

// Test CreateAccount functionality
func (s *AccountHandlerSuite) TestCreateAccount_WithInitialDeposit() {
	accountID := uuid.New()
	reqBody := dto.CreateAccountRequest{
		AccountType:       "CHECKING",
		AccountNumber:     "1012345678",
		RoutingNumber:     "123456789",
		AccountHolderName: "John Doe",
	}

	expectedAccount := &models.Account{
		ID:            accountID,
		UserID:        s.testUserID,
		AccountNumber: "1012345678",
		AccountType:   "CHECKING",
		Balance:       decimal.NewFromFloat(100.00),
		Status:        "active",
	}

	s.mockService.EXPECT().
		CreateAccount(s.testUserID, "CHECKING", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(userID uuid.UUID, accountType, accountNumber, routingNumber string, initialDeposit decimal.Decimal) (*models.Account, error) {
			if !initialDeposit.Equal(decimal.NewFromFloat(100.00)) {
				s.T().Errorf("expected amount 100.00, got %s", initialDeposit.String())
			}
			return expectedAccount, nil
		})

	c, rec := s.createContextWithAuth("POST", "/accounts", reqBody, s.testUserID, "user")
	c.Set("initialDeposit", decimal.NewFromFloat(100.00))

	err := s.handler.CreateAccount(c)
	s.NoError(err)
	s.Equal(http.StatusCreated, rec.Code)

	var resp dto.CreateAccountResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	s.NoError(err)
	s.Equal(expectedAccount.ID, resp.Account.ID)
	s.Equal(expectedAccount.AccountNumber, resp.Account.AccountNumber)
}

func (s *AccountHandlerSuite) TestCreateAccount_InvalidAmount() {
	reqBody := map[string]interface{}{
		"accountType":    "checking",
		"initialDeposit": "invalid",
	}

	c, rec := s.createContextWithAuth("POST", "/accounts", reqBody, s.testUserID, "user")

	err := s.handler.CreateAccount(c)
	s.NoError(err) // Handler returns nil, error is written to response
	s.Equal(http.StatusBadRequest, rec.Code)

	// Verify error response body is not empty
	s.NotEmpty(rec.Body.String())
}

func (s *AccountHandlerSuite) TestCreateAccount_AccountAlreadyExists() {
	reqBody := dto.CreateAccountRequest{
		AccountType:       "CHECKING",
		AccountNumber:     "1012345678",
		RoutingNumber:     "123456789",
		AccountHolderName: "John Doe",
	}

	s.mockService.EXPECT().
		CreateAccount(s.testUserID, "CHECKING", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, services.ErrAccountAlreadyExists)

	c, rec := s.createContextWithAuth("POST", "/accounts", reqBody, s.testUserID, "user")

	err := s.handler.CreateAccount(c)
	s.NoError(err)                           // Handler returns nil, error is written to response
	s.Equal(http.StatusBadRequest, rec.Code) // Validation errors return 400
}

// Test GetAccount functionality
func (s *AccountHandlerSuite) TestGetAccount_Success() {
	accountID := uuid.New()
	expectedAccount := &models.Account{
		ID:            accountID,
		UserID:        s.testUserID,
		AccountNumber: "1012345678",
		AccountType:   "checking",
		Balance:       decimal.NewFromFloat(100.00),
		Status:        "active",
	}

	s.mockService.EXPECT().
		GetAccountByID(accountID, &s.testUserID).
		Return(expectedAccount, nil)

	c, rec := s.createContextWithAuth("GET", "/accounts/"+accountID.String(), nil, s.testUserID, "user")
	c.SetParamNames("accountId")
	c.SetParamValues(accountID.String())

	err := s.handler.GetAccount(c)
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var account models.Account
	err = json.Unmarshal(rec.Body.Bytes(), &account)
	s.NoError(err)
	s.Equal(expectedAccount.ID, account.ID)
}

func (s *AccountHandlerSuite) TestGetAccount_NotFound() {
	accountID := uuid.New()

	s.mockService.EXPECT().
		GetAccountByID(accountID, &s.testUserID).
		Return(nil, services.ErrAccountNotFound)

	c, rec := s.createContextWithAuth("GET", "/accounts/"+accountID.String(), nil, s.testUserID, "user")
	c.SetParamNames("accountId")
	c.SetParamValues(accountID.String())

	err := s.handler.GetAccount(c)
	s.NoError(err) // Handler returns nil, error is written to response
	s.Equal(http.StatusNotFound, rec.Code)
}

func (s *AccountHandlerSuite) TestGetAccount_Unauthorized() {
	accountID := uuid.New()

	s.mockService.EXPECT().
		GetAccountByID(accountID, &s.testUserID).
		Return(nil, services.ErrUnauthorized)

	c, rec := s.createContextWithAuth("GET", "/accounts/"+accountID.String(), nil, s.testUserID, "user")
	c.SetParamNames("accountId")
	c.SetParamValues(accountID.String())

	err := s.handler.GetAccount(c)
	s.NoError(err) // Handler returns nil, error is written to response
	s.Equal(http.StatusForbidden, rec.Code)
}

// Test GetUserAccounts functionality
func (s *AccountHandlerSuite) TestGetUserAccounts_Success() {
	expectedAccounts := []models.Account{
		{
			ID:            uuid.New(),
			UserID:        s.testUserID,
			AccountNumber: "1012345678",
			AccountType:   "checking",
			Balance:       decimal.NewFromFloat(100.00),
			Status:        "active",
		},
		{
			ID:            uuid.New(),
			UserID:        s.testUserID,
			AccountNumber: "2012345679",
			AccountType:   "savings",
			Balance:       decimal.NewFromFloat(500.00),
			Status:        "active",
		},
	}

	s.mockService.EXPECT().
		GetUserAccounts(s.testUserID).
		Return(expectedAccounts, nil)

	c, rec := s.createContextWithAuth("GET", "/accounts", nil, s.testUserID, "user")

	err := s.handler.GetUserAccounts(c)
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var accounts []models.Account
	err = json.Unmarshal(rec.Body.Bytes(), &accounts)
	s.NoError(err)
	s.Len(accounts, 2)
}

// Test PerformTransaction functionality
func (s *AccountHandlerSuite) TestPerformTransaction_Credit() {
	accountID := uuid.New()
	transactionID := uuid.New()

	reqBody := dto.TransactionRequest{
		Amount:      "50.00",
		Type:        "credit",
		Description: "Deposit",
	}

	expectedTransaction := &models.Transaction{
		ID:              transactionID,
		AccountID:       accountID,
		Amount:          decimal.NewFromFloat(50.00),
		TransactionType: "credit",
		Description:     "Deposit",
		Status:          "completed",
	}

	s.mockService.EXPECT().
		PerformTransaction(accountID, gomock.Any(), "credit", "Deposit", &s.testUserID).
		DoAndReturn(func(_ uuid.UUID, amount decimal.Decimal, _ string, _ string, _ *uuid.UUID) (*models.Transaction, error) {
			if !amount.Equal(decimal.NewFromFloat(50.00)) {
				s.T().Errorf("expected amount 50.00, got %s", amount.String())
			}
			return expectedTransaction, nil
		})

	c, rec := s.createContextWithAuth("POST", "/accounts/"+accountID.String()+"/transactions", reqBody, s.testUserID, "user")
	c.SetParamNames("accountId")
	c.SetParamValues(accountID.String())

	err := s.handler.PerformTransaction(c)
	s.NoError(err)
	s.Equal(http.StatusCreated, rec.Code)

	var transaction models.Transaction
	err = json.Unmarshal(rec.Body.Bytes(), &transaction)
	s.NoError(err)
	s.Equal(expectedTransaction.ID, transaction.ID)
}

func (s *AccountHandlerSuite) TestPerformTransaction_InsufficientFunds() {
	accountID := uuid.New()

	reqBody := dto.TransactionRequest{
		Amount:      "1000.00",
		Type:        "debit",
		Description: "Withdrawal",
	}

	s.mockService.EXPECT().
		PerformTransaction(accountID, gomock.Any(), "debit", "Withdrawal", &s.testUserID).
		DoAndReturn(func(_ uuid.UUID, amount decimal.Decimal, _ string, _ string, _ *uuid.UUID) (*models.Transaction, error) {
			if !amount.Equal(decimal.NewFromFloat(1000.00)) {
				s.T().Errorf("expected amount 1000.00, got %s", amount.String())
			}
			return nil, services.ErrInsufficientFunds
		})

	c, rec := s.createContextWithAuth("POST", "/accounts/"+accountID.String()+"/transactions", reqBody, s.testUserID, "user")
	c.SetParamNames("accountId")
	c.SetParamValues(accountID.String())

	err := s.handler.PerformTransaction(c)
	s.NoError(err) // SendError returns nil, error is in response body

	// Check response status code
	s.Equal(http.StatusUnprocessableEntity, rec.Code)

	// Parse and verify error response
	var errorResp ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &errorResp)
	s.NoError(err)
	s.Equal("TRANSACTION_003", errorResp.Error.Code)
	s.Contains(errorResp.Error.Message, "Insufficient account balance")
}

// Test Transfer functionality
func (s *AccountHandlerSuite) TestTransfer_Success() {
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	idempotencyKey := uuid.New().String()

	reqBody := dto.TransferRequest{
		ToAccountID: toAccountID.String(),
		Amount:      "100.00",
		Description: "Transfer to savings",
	}

	// Expect audit log for transfer initiation
	s.auditLogger.EXPECT().
		LogTransferInitiated(gomock.Any(), gomock.Any(), fromAccountID, toAccountID, "100.00", idempotencyKey, s.testUserID).
		Times(1)

	// Expect service call
	s.mockService.EXPECT().
		TransferBetweenAccounts(fromAccountID, toAccountID, gomock.Any(), "Transfer to savings", idempotencyKey, s.testUserID).
		DoAndReturn(func(_ uuid.UUID, _ uuid.UUID, amount decimal.Decimal, _ string, _ string, _ uuid.UUID) (*models.Transfer, error) {
			if !amount.Equal(decimal.NewFromFloat(100.00)) {
				s.T().Errorf("expected amount 100.00, got %s", amount.String())
			}
			debitTxID := uuid.New()
			creditTxID := uuid.New()
			return &models.Transfer{
				ID:                  uuid.New(),
				FromAccountID:       fromAccountID,
				ToAccountID:         toAccountID,
				Amount:              amount,
				Description:         "Transfer to savings",
				Status:              models.TransferStatusCompleted,
				DebitTransactionID:  &debitTxID,
				CreditTransactionID: &creditTxID,
			}, nil
		})

	// Expect audit log for transfer completion
	s.auditLogger.EXPECT().
		LogTransferCompleted(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1)

	// Expect metrics calls
	s.metricsCollector.EXPECT().
		IncrementCounter("transfers_total", map[string]string{"status": "completed"}).
		Times(1)
	s.metricsCollector.EXPECT().
		RecordProcessingTime("transfer_duration_success", gomock.Any()).
		Times(1)
	s.metricsCollector.EXPECT().
		RecordGauge("transfer_amount", gomock.Any(), nil).
		Times(1)

	c, rec := s.createContextWithAuth("POST", "/accounts/"+fromAccountID.String()+"/transfer", reqBody, s.testUserID, "user")
	c.SetParamNames("accountId")
	c.SetParamValues(fromAccountID.String())
	c.Request().Header.Set("Idempotency-Key", idempotencyKey)

	err := s.handler.Transfer(c)
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *AccountHandlerSuite) TestTransfer_SameAccount() {
	fromAccountID := uuid.New()
	idempotencyKey := uuid.New().String()

	reqBody := dto.TransferRequest{
		ToAccountID: fromAccountID.String(),
		Amount:      "100.00",
		Description: "Transfer",
	}

	// Expect audit log for transfer initiation
	s.auditLogger.EXPECT().
		LogTransferInitiated(gomock.Any(), gomock.Any(), fromAccountID, fromAccountID, "100.00", idempotencyKey, s.testUserID).
		Times(1)

	// Expect service call that returns error
	s.mockService.EXPECT().
		TransferBetweenAccounts(fromAccountID, fromAccountID, gomock.Any(), "Transfer", idempotencyKey, s.testUserID).
		DoAndReturn(func(_ uuid.UUID, _ uuid.UUID, amount decimal.Decimal, _ string, _ string, _ uuid.UUID) (*models.Transfer, error) {
			if !amount.Equal(decimal.NewFromFloat(100.00)) {
				s.T().Errorf("expected amount 100.00, got %s", amount.String())
			}
			return nil, services.ErrSameAccountTransfer
		})

	// Expect metrics calls for failed transfer
	s.metricsCollector.EXPECT().
		IncrementCounter("transfers_total", map[string]string{"status": "failed"}).
		Times(1)
	s.metricsCollector.EXPECT().
		RecordProcessingTime("transfer_duration_failed", gomock.Any()).
		Times(1)

	c, rec := s.createContextWithAuth("POST", "/accounts/"+fromAccountID.String()+"/transfer", reqBody, s.testUserID, "user")
	c.SetParamNames("accountId")
	c.SetParamValues(fromAccountID.String())
	c.Request().Header.Set("Idempotency-Key", idempotencyKey)

	err := s.handler.Transfer(c)
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)

	var response struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &response)
	s.Equal("TRANSFER_001", response.Error.Code)
	s.Contains(response.Error.Message, "same account")
}

// Test Transfer with Idempotency Key - Success
func (s *AccountHandlerSuite) TestTransfer_WithIdempotencyKey_Success() {
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	idempotencyKey := uuid.New().String()
	debitTxID := uuid.New()
	creditTxID := uuid.New()
	transferID := uuid.New()

	reqBody := dto.TransferRequest{
		ToAccountID: toAccountID.String(),
		Amount:      "150.00",
		Description: "Payment with idempotency",
	}

	expectedTransfer := &models.Transfer{
		ID:                  transferID,
		FromAccountID:       fromAccountID,
		ToAccountID:         toAccountID,
		Amount:              decimal.NewFromFloat(150.00),
		Description:         "Payment with idempotency",
		IdempotencyKey:      idempotencyKey,
		Status:              models.TransferStatusCompleted,
		DebitTransactionID:  &debitTxID,
		CreditTransactionID: &creditTxID,
	}

	// Expect audit log for transfer initiation
	s.auditLogger.EXPECT().
		LogTransferInitiated(gomock.Any(), gomock.Any(), fromAccountID, toAccountID, "150.00", idempotencyKey, s.testUserID).
		Times(1)

	// Expect service call
	s.mockService.EXPECT().
		TransferBetweenAccounts(fromAccountID, toAccountID, gomock.Any(), "Payment with idempotency", idempotencyKey, s.testUserID).
		DoAndReturn(func(_ uuid.UUID, _ uuid.UUID, amount decimal.Decimal, _ string, _ string, _ uuid.UUID) (*models.Transfer, error) {
			if !amount.Equal(decimal.NewFromFloat(150.00)) {
				s.T().Errorf("expected amount 150.00, got %s", amount.String())
			}
			return expectedTransfer, nil
		})

	// Expect audit log for transfer completion
	s.auditLogger.EXPECT().
		LogTransferCompleted(gomock.Any(), transferID, gomock.Any(), &debitTxID, &creditTxID).
		Times(1)

	// Expect metrics calls
	s.metricsCollector.EXPECT().
		IncrementCounter("transfers_total", map[string]string{"status": "completed"}).
		Times(1)
	s.metricsCollector.EXPECT().
		RecordProcessingTime("transfer_duration_success", gomock.Any()).
		Times(1)
	s.metricsCollector.EXPECT().
		RecordGauge("transfer_amount", gomock.Any(), nil).
		Times(1)

	c, rec := s.createContextWithAuth("POST", "/accounts/"+fromAccountID.String()+"/transfer", reqBody, s.testUserID, "user")
	c.SetParamNames("accountId")
	c.SetParamValues(fromAccountID.String())
	c.Request().Header.Set("Idempotency-Key", idempotencyKey)

	err := s.handler.Transfer(c)
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	// Verify response structure
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal("Transfer completed successfully", response["message"])
	s.Equal(transferID.String(), response["transferId"])
	s.Equal(fromAccountID.String(), response["fromAccountId"])
	s.Equal(toAccountID.String(), response["toAccountId"])

	// Check amount (decimal formatting may vary)
	amountStr := response["amount"].(string)
	amount, err := decimal.NewFromString(amountStr)
	s.NoError(err)
	s.True(amount.Equal(decimal.NewFromFloat(150.00)), "expected amount 150.00, got %s", amountStr)

	s.Equal(debitTxID.String(), response["debitTransactionId"])
	s.Equal(creditTxID.String(), response["creditTransactionId"])
}

// Test Transfer with Duplicate Idempotency Key - Completed Transfer
func (s *AccountHandlerSuite) TestTransfer_DuplicateIdempotencyKey_CompletedTransfer() {
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	idempotencyKey := uuid.New().String()
	debitTxID := uuid.New()
	creditTxID := uuid.New()
	transferID := uuid.New()

	reqBody := dto.TransferRequest{
		ToAccountID: toAccountID.String(),
		Amount:      "150.00",
		Description: "Duplicate request",
	}

	existingTransfer := &models.Transfer{
		ID:                  transferID,
		FromAccountID:       fromAccountID,
		ToAccountID:         toAccountID,
		Amount:              decimal.NewFromFloat(150.00),
		Description:         "Duplicate request",
		IdempotencyKey:      idempotencyKey,
		Status:              models.TransferStatusCompleted,
		DebitTransactionID:  &debitTxID,
		CreditTransactionID: &creditTxID,
	}

	// Expect audit log for transfer initiation
	s.auditLogger.EXPECT().
		LogTransferInitiated(gomock.Any(), gomock.Any(), fromAccountID, toAccountID, "150.00", idempotencyKey, s.testUserID).
		Times(1)

	// Expect service call that returns existing completed transfer
	s.mockService.EXPECT().
		TransferBetweenAccounts(fromAccountID, toAccountID, gomock.Any(), "Duplicate request", idempotencyKey, s.testUserID).
		Return(existingTransfer, nil)

	// Expect audit log for transfer completion
	s.auditLogger.EXPECT().
		LogTransferCompleted(gomock.Any(), transferID, gomock.Any(), &debitTxID, &creditTxID).
		Times(1)

	// Expect metrics calls for successful transfer
	s.metricsCollector.EXPECT().
		IncrementCounter("transfers_total", map[string]string{"status": "completed"}).
		Times(1)
	s.metricsCollector.EXPECT().
		RecordProcessingTime("transfer_duration_success", gomock.Any()).
		Times(1)
	s.metricsCollector.EXPECT().
		RecordGauge("transfer_amount", gomock.Any(), nil).
		Times(1)

	c, rec := s.createContextWithAuth("POST", "/accounts/"+fromAccountID.String()+"/transfer", reqBody, s.testUserID, "user")
	c.SetParamNames("accountId")
	c.SetParamValues(fromAccountID.String())
	c.Request().Header.Set("Idempotency-Key", idempotencyKey)

	err := s.handler.Transfer(c)
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	// Verify response includes existing transfer data
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal(transferID.String(), response["transferId"])
}

// Test Transfer with Duplicate Idempotency Key - Pending Transfer
func (s *AccountHandlerSuite) TestTransfer_DuplicateIdempotencyKey_PendingTransfer() {
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	idempotencyKey := uuid.New().String()

	reqBody := dto.TransferRequest{
		ToAccountID: toAccountID.String(),
		Amount:      "150.00",
		Description: "Pending duplicate",
	}

	// Expect audit log for transfer initiation
	s.auditLogger.EXPECT().
		LogTransferInitiated(gomock.Any(), gomock.Any(), fromAccountID, toAccountID, "150.00", idempotencyKey, s.testUserID).
		Times(1)

	// Expect service call that returns pending error
	s.mockService.EXPECT().
		TransferBetweenAccounts(fromAccountID, toAccountID, gomock.Any(), "Pending duplicate", idempotencyKey, s.testUserID).
		Return(nil, services.ErrTransferPending)

	// Expect metrics calls for failed transfer
	s.metricsCollector.EXPECT().
		IncrementCounter("transfers_total", map[string]string{"status": "failed"}).
		Times(1)
	s.metricsCollector.EXPECT().
		RecordProcessingTime("transfer_duration_failed", gomock.Any()).
		Times(1)

	c, rec := s.createContextWithAuth("POST", "/accounts/"+fromAccountID.String()+"/transfer", reqBody, s.testUserID, "user")
	c.SetParamNames("accountId")
	c.SetParamValues(fromAccountID.String())
	c.Request().Header.Set("Idempotency-Key", idempotencyKey)

	err := s.handler.Transfer(c)
	s.NoError(err)
	s.Equal(http.StatusConflict, rec.Code)

	var response struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &response)
	s.Equal("TRANSFER_002", response.Error.Code)
	s.Contains(response.Error.Message, "still processing")
}

// Test Transfer with Duplicate Idempotency Key - Failed Transfer
func (s *AccountHandlerSuite) TestTransfer_DuplicateIdempotencyKey_FailedTransfer() {
	fromAccountID := uuid.New()
	toAccountID := uuid.New()
	idempotencyKey := uuid.New().String()

	reqBody := dto.TransferRequest{
		ToAccountID: toAccountID.String(),
		Amount:      "150.00",
		Description: "Failed duplicate",
	}

	// Expect audit log for transfer initiation
	s.auditLogger.EXPECT().
		LogTransferInitiated(gomock.Any(), gomock.Any(), fromAccountID, toAccountID, "150.00", idempotencyKey, s.testUserID).
		Times(1)

	// Expect service call that returns failed error
	s.mockService.EXPECT().
		TransferBetweenAccounts(fromAccountID, toAccountID, gomock.Any(), "Failed duplicate", idempotencyKey, s.testUserID).
		Return(nil, services.ErrTransferFailed)

	// Expect metrics calls for failed transfer
	s.metricsCollector.EXPECT().
		IncrementCounter("transfers_total", map[string]string{"status": "failed"}).
		Times(1)
	s.metricsCollector.EXPECT().
		RecordProcessingTime("transfer_duration_failed", gomock.Any()).
		Times(1)

	c, rec := s.createContextWithAuth("POST", "/accounts/"+fromAccountID.String()+"/transfer", reqBody, s.testUserID, "user")
	c.SetParamNames("accountId")
	c.SetParamValues(fromAccountID.String())
	c.Request().Header.Set("Idempotency-Key", idempotencyKey)

	err := s.handler.Transfer(c)
	s.NoError(err)
	s.Equal(http.StatusConflict, rec.Code)

	var response struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &response)
	s.Equal("TRANSFER_003", response.Error.Code)
	s.Contains(response.Error.Message, "previously failed")
}

// Test Transfer without Idempotency Key
func (s *AccountHandlerSuite) TestTransfer_MissingIdempotencyKey() {
	fromAccountID := uuid.New()
	toAccountID := uuid.New()

	reqBody := dto.TransferRequest{
		ToAccountID: toAccountID.String(),
		Amount:      "150.00",
		Description: "No idempotency key",
	}

	c, rec := s.createContextWithAuth("POST", "/accounts/"+fromAccountID.String()+"/transfer", reqBody, s.testUserID, "user")
	c.SetParamNames("accountId")
	c.SetParamValues(fromAccountID.String())
	// No Idempotency-Key header set

	err := s.handler.Transfer(c)
	s.NoError(err) // Handler returns nil, error is written to response
	s.Equal(http.StatusBadRequest, rec.Code)
}

// Test Transfer History endpoint
func (s *AccountHandlerSuite) TestGetTransferHistory_Success() {
	transferID1 := uuid.New()
	transferID2 := uuid.New()
	accountID1 := uuid.New()
	accountID2 := uuid.New()

	expectedTransfers := []models.Transfer{
		{
			ID:            transferID1,
			FromAccountID: accountID1,
			ToAccountID:   accountID2,
			Amount:        decimal.NewFromFloat(100.00),
			Description:   "Transfer 1",
			Status:        models.TransferStatusCompleted,
		},
		{
			ID:            transferID2,
			FromAccountID: accountID2,
			ToAccountID:   accountID1,
			Amount:        decimal.NewFromFloat(50.00),
			Description:   "Transfer 2",
			Status:        models.TransferStatusCompleted,
		},
	}

	filters := models.TransferFilters{}
	s.mockService.EXPECT().
		GetUserTransfers(s.testUserID, filters, 0, 20).
		Return(expectedTransfers, int64(2), nil)

	c, rec := s.createContextWithAuth("GET", "/api/v1/transfers", nil, s.testUserID, "user")

	err := s.handler.GetTransferHistory(c)
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var response dto.TransferHistoryResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.NoError(err)
	s.Len(response.Transfers, 2)
	s.Equal(int64(2), response.Pagination.Total)
}

// Test Transfer History with filters
func (s *AccountHandlerSuite) TestGetTransferHistory_WithStatusFilter() {
	transferID := uuid.New()
	accountID1 := uuid.New()
	accountID2 := uuid.New()

	expectedTransfers := []models.Transfer{
		{
			ID:            transferID,
			FromAccountID: accountID1,
			ToAccountID:   accountID2,
			Amount:        decimal.NewFromFloat(100.00),
			Description:   "Completed transfer",
			Status:        models.TransferStatusCompleted,
		},
	}

	filters := models.TransferFilters{Status: models.TransferStatusCompleted}
	s.mockService.EXPECT().
		GetUserTransfers(s.testUserID, filters, 0, 20).
		Return(expectedTransfers, int64(1), nil)

	c, rec := s.createContextWithAuth("GET", "/api/v1/transfers?status=completed", nil, s.testUserID, "user")

	err := s.handler.GetTransferHistory(c)
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var response dto.TransferHistoryResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.NoError(err)
	s.Len(response.Transfers, 1)
	s.Equal(models.TransferStatusCompleted, response.Transfers[0].Status)
}

// Test Admin endpoints
func (s *AccountHandlerSuite) TestGetAllAccounts_AdminSuccess() {
	expectedAccounts := []models.Account{
		{
			ID:            uuid.New(),
			UserID:        s.testUserID,
			AccountNumber: "1012345678",
			AccountType:   "checking",
			Balance:       decimal.NewFromFloat(100.00),
			Status:        "active",
		},
	}

	filters := models.AccountFilters{}
	s.mockService.EXPECT().
		GetAllAccounts(filters, 0, 20).
		Return(expectedAccounts, int64(1), nil)

	c, rec := s.createContextWithAuth("GET", "/admin/accounts", nil, s.testAdminID, "admin")

	err := s.handler.GetAllAccounts(c)
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	s.NoError(err)
	s.Equal(float64(1), resp["total"])
}

func (s *AccountHandlerSuite) TestGetAllAccounts_NonAdminFails() {
	// NOTE: Admin authorization check was moved to middleware (RequireAdmin)
	// This test now verifies that the handler processes requests when called directly
	// In production, non-admin users are blocked by middleware before reaching handler
	c, _ := s.createContextWithAuth("GET", "/admin/accounts", nil, s.testUserID, "user")

	// Mock service call since handler will proceed without middleware protection
	s.mockService.EXPECT().
		GetAllAccounts(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]models.Account{}, int64(0), nil)

	err := s.handler.GetAllAccounts(c)
	// Handler itself doesn't check admin status - middleware does
	s.NoError(err)
}

func (s *AccountHandlerSuite) TestGetAccountByIDAdmin_Success() {
	accountID := uuid.New()
	expectedAccount := &models.Account{
		ID:            accountID,
		UserID:        s.testUserID,
		AccountNumber: "1012345678",
		AccountType:   "checking",
		Balance:       decimal.NewFromFloat(100.00),
		Status:        "active",
	}

	var nilUserID *uuid.UUID
	s.mockService.EXPECT().
		GetAccountByID(accountID, nilUserID).
		Return(expectedAccount, nil)

	c, rec := s.createContextWithAuth("GET", "/admin/accounts/"+accountID.String(), nil, s.testAdminID, "admin")
	c.SetParamNames("accountId")
	c.SetParamValues(accountID.String())

	err := s.handler.GetAccountByIDAdmin(c)
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var account models.Account
	err = json.Unmarshal(rec.Body.Bytes(), &account)
	s.NoError(err)
	s.Equal(expectedAccount.ID, account.ID)
}

func (s *AccountHandlerSuite) TestGetUserAccountsAdmin_Success() {
	userID := uuid.New()
	expectedAccounts := []models.Account{
		{
			ID:            uuid.New(),
			UserID:        userID,
			AccountNumber: "1012345678",
			AccountType:   "checking",
			Balance:       decimal.NewFromFloat(100.00),
			Status:        "active",
		},
	}

	s.mockService.EXPECT().
		GetUserAccounts(userID).
		Return(expectedAccounts, nil)

	c, rec := s.createContextWithAuth("GET", "/admin/users/"+userID.String()+"/accounts", nil, s.testAdminID, "admin")
	c.SetParamNames("userId")
	c.SetParamValues(userID.String())

	err := s.handler.GetUserAccountsAdmin(c)
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var accounts []models.Account
	err = json.Unmarshal(rec.Body.Bytes(), &accounts)
	s.NoError(err)
	s.Len(accounts, 1)
}

// Test CloseAccount functionality
func (s *AccountHandlerSuite) TestCloseAccount_Success() {
	accountID := uuid.New()

	s.mockService.EXPECT().
		CloseAccount(accountID, s.testUserID).
		Return(nil)

	c, rec := s.createContextWithAuth("DELETE", "/accounts/"+accountID.String(), nil, s.testUserID, "user")
	c.SetParamNames("accountId")
	c.SetParamValues(accountID.String())

	err := s.handler.CloseAccount(c)
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *AccountHandlerSuite) TestCloseAccount_NonZeroBalance() {
	accountID := uuid.New()

	s.mockService.EXPECT().
		CloseAccount(accountID, s.testUserID).
		Return(services.ErrAccountClosureNotAllowed)

	c, rec := s.createContextWithAuth("DELETE", "/accounts/"+accountID.String(), nil, s.testUserID, "user")
	c.SetParamNames("accountId")
	c.SetParamValues(accountID.String())

	err := s.handler.CloseAccount(c)
	s.NoError(err)                                    // Handler returns nil, error is written to response
	s.Equal(http.StatusUnprocessableEntity, rec.Code) // AccountOperationNotPermitted returns 422
}
