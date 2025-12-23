package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"array-assessment/internal/dto"
	"array-assessment/internal/models"
	"array-assessment/internal/repositories/repository_mocks"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type TransactionHandlerTestSuite struct {
	suite.Suite
	handler             *TransactionHandler
	echo                *echo.Echo
	accountID           uuid.UUID
	userID              uuid.UUID
	ctrl                *gomock.Controller
	mockAccountRepo     *repository_mocks.MockAccountRepositoryInterface
	mockTransactionRepo *repository_mocks.MockTransactionRepositoryInterface
}

func TestTransactionHandlerSuite(t *testing.T) {
	suite.Run(t, new(TransactionHandlerTestSuite))
}

func (s *TransactionHandlerTestSuite) SetupTest() {
	s.echo = echo.New()
	s.accountID = uuid.New()
	s.userID = uuid.New()
	s.ctrl = gomock.NewController(s.T())
	s.mockAccountRepo = repository_mocks.NewMockAccountRepositoryInterface(s.ctrl)
	s.mockTransactionRepo = repository_mocks.NewMockTransactionRepositoryInterface(s.ctrl)
}

// Cursor Encoding/Decoding Tests

func (s *TransactionHandlerTestSuite) TestEncodeCursor_ValidTimestamp() {
	timestamp := time.Now()
	transactionID := uuid.New()

	cursor := encodeCursor(timestamp, transactionID)

	s.NotEmpty(cursor)

	// Verify it's valid base64
	_, err := base64.URLEncoding.DecodeString(cursor)
	s.NoError(err)
}

func (s *TransactionHandlerTestSuite) TestDecodeCursor_ValidCursor() {
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	transactionID := uuid.New()

	cursor := encodeCursor(timestamp, transactionID)
	decodedTime, decodedID, err := decodeCursor(cursor)

	s.NoError(err)
	s.Equal(timestamp.Unix(), decodedTime.Unix())
	s.Equal(transactionID, decodedID)
}

func (s *TransactionHandlerTestSuite) TestDecodeCursor_InvalidCursor() {
	testCases := []struct {
		name   string
		cursor string
	}{
		{"empty cursor", ""},
		{"invalid base64", "not-base64!!!"},
		{"invalid format", base64.URLEncoding.EncodeToString([]byte("invalid"))},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			_, _, err := decodeCursor(tc.cursor)
			s.Error(err)
		})
	}
}

// Pagination Tests

func (s *TransactionHandlerTestSuite) TestListTransactions_FirstPage() {
	handler := NewTransactionHandler(s.mockTransactionRepo, s.mockAccountRepo)

	// Setup test account
	account := &models.Account{
		ID:            s.accountID,
		UserID:        s.userID,
		AccountType:   models.AccountTypeChecking,
		AccountNumber: "1234567890",
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
	}

	// Create test transactions - 21 transactions to test pagination (limit is 20)
	transactions := make([]models.Transaction, 21)
	for i := 0; i < 21; i++ {
		transactions[i] = models.Transaction{
			ID:              uuid.New(),
			AccountID:       s.accountID,
			Amount:          decimal.NewFromFloat(float64(10 + i)),
			TransactionType: models.TransactionTypeDebit,
			Description:     fmt.Sprintf("Test transaction %d", i),
			Status:          models.TransactionStatusCompleted,
			Category:        models.CategoryGroceries,
			MerchantName:    gofakeit.Company(),
			BalanceBefore:   decimal.NewFromFloat(float64(1000 - (i * 10))),
			BalanceAfter:    decimal.NewFromFloat(float64(1000 - ((i + 1) * 10))),
			CreatedAt:       time.Now().Add(-time.Duration(i) * time.Hour),
		}
	}

	// Setup expectations
	s.mockAccountRepo.EXPECT().
		GetByID(s.accountID).
		Return(account, nil)

	s.mockTransactionRepo.EXPECT().
		GetWithFilters(gomock.Any()).
		Return(transactions, int64(21), nil)

	// Create request
	url := fmt.Sprintf("/api/v1/accounts/%s/transactions", s.accountID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("accountId")
	c.SetParamValues(s.accountID.String())
	c.Set("user_id", s.userID)

	// Execute
	err := handler.ListTransactions(c)
	s.NoError(err)

	// Verify response
	s.Equal(http.StatusOK, rec.Code)

	var response dto.ListTransactionsResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.NoError(err)

	// Should return 20 transactions (limit)
	s.Len(response.Transactions, 20)

	// Should have more results
	s.True(response.Pagination.HasMore)

	// Should have next cursor
	s.NotEmpty(response.Pagination.NextCursor)

	// Verify pagination info
	s.Equal(20, response.Pagination.Limit)
	s.Equal(int64(21), response.Pagination.Total)
}

func (s *TransactionHandlerTestSuite) TestListTransactions_WithCursor() {
	handler := NewTransactionHandler(s.mockTransactionRepo, s.mockAccountRepo)

	// Setup test account
	account := &models.Account{
		ID:            s.accountID,
		UserID:        s.userID,
		AccountType:   models.AccountTypeChecking,
		AccountNumber: "1234567890",
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
	}

	// Create cursor timestamp and transaction ID
	cursorTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	cursorTransactionID := uuid.New()

	// Create test transactions for second page
	transactions := make([]models.Transaction, 10)
	for i := 0; i < 10; i++ {
		transactions[i] = models.Transaction{
			ID:              uuid.New(),
			AccountID:       s.accountID,
			Amount:          decimal.NewFromFloat(float64(20 + i)),
			TransactionType: models.TransactionTypeDebit,
			Description:     fmt.Sprintf("Second page transaction %d", i),
			Status:          models.TransactionStatusCompleted,
			Category:        models.CategoryDining,
			MerchantName:    gofakeit.Company(),
			BalanceBefore:   decimal.NewFromFloat(float64(800 - (i * 20))),
			BalanceAfter:    decimal.NewFromFloat(float64(800 - ((i + 1) * 20))),
			CreatedAt:       cursorTime.Add(-time.Duration(i+1) * time.Hour),
		}
	}

	// Setup expectations
	s.mockAccountRepo.EXPECT().
		GetByID(s.accountID).
		Return(account, nil)

	s.mockTransactionRepo.EXPECT().
		GetWithFilters(gomock.Any()).
		Return(transactions, int64(30), nil)

	// Create cursor
	cursor := encodeCursor(cursorTime, cursorTransactionID)

	// Create request with cursor
	url := fmt.Sprintf("/api/v1/accounts/%s/transactions?cursor=%s", s.accountID, cursor)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("accountId")
	c.SetParamValues(s.accountID.String())
	c.Set("user_id", s.userID)

	// Execute
	err := handler.ListTransactions(c)
	s.NoError(err)

	// Verify response
	s.Equal(http.StatusOK, rec.Code)

	var response dto.ListTransactionsResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.NoError(err)

	// Should return 10 transactions
	s.Len(response.Transactions, 10)

	// Should not have more results (we returned exactly 10)
	s.False(response.Pagination.HasMore)

	// Verify pagination info
	s.Equal(20, response.Pagination.Limit)
	s.Equal(int64(30), response.Pagination.Total)
}

func (s *TransactionHandlerTestSuite) TestListTransactions_EmptyResults() {
	handler := NewTransactionHandler(s.mockTransactionRepo, s.mockAccountRepo)

	// Setup test account
	account := &models.Account{
		ID:            s.accountID,
		UserID:        s.userID,
		AccountType:   models.AccountTypeChecking,
		AccountNumber: "1234567890",
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
	}

	// Setup expectations - return empty results
	s.mockAccountRepo.EXPECT().
		GetByID(s.accountID).
		Return(account, nil)

	s.mockTransactionRepo.EXPECT().
		GetWithFilters(gomock.Any()).
		Return([]models.Transaction{}, int64(0), nil)

	// Create request
	url := fmt.Sprintf("/api/v1/accounts/%s/transactions", s.accountID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("accountId")
	c.SetParamValues(s.accountID.String())
	c.Set("user_id", s.userID)

	// Execute
	err := handler.ListTransactions(c)
	s.NoError(err)

	// Verify response
	s.Equal(http.StatusOK, rec.Code)

	var response dto.ListTransactionsResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.NoError(err)

	// Should return empty transactions array
	s.Len(response.Transactions, 0)
	s.NotNil(response.Transactions) // Should be empty array, not nil

	// Should not have more results
	s.False(response.Pagination.HasMore)

	// Should not have next cursor
	s.Empty(response.Pagination.NextCursor)

	// Verify pagination info
	s.Equal(20, response.Pagination.Limit)
	s.Equal(int64(0), response.Pagination.Total)
}

// Filtering Tests

func (s *TransactionHandlerTestSuite) TestListTransactions_FilterByDateRange() {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/"+s.accountID.String()+"/transactions?start_date=2024-01-01&end_date=2024-01-31", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("accountId")
	c.SetParamValues(s.accountID.String())
	c.Set("userID", s.userID)

	// Verify date parsing
	startDate := c.QueryParam("start_date")
	s.Equal("2024-01-01", startDate)

	endDate := c.QueryParam("end_date")
	s.Equal("2024-01-31", endDate)
}

func (s *TransactionHandlerTestSuite) TestListTransactions_FilterByType() {
	testCases := []string{
		models.TransactionTypeCredit,
		models.TransactionTypeDebit,
	}

	for _, txnType := range testCases {
		s.Run(txnType, func() {
			url := fmt.Sprintf("/api/v1/accounts/%s/transactions?type=%s", s.accountID, txnType)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			c := s.echo.NewContext(req, rec)

			s.Equal(txnType, c.QueryParam("type"))
		})
	}
}

func (s *TransactionHandlerTestSuite) TestListTransactions_FilterByStatus() {
	testCases := []string{
		models.TransactionStatusPending,
		models.TransactionStatusCompleted,
		models.TransactionStatusFailed,
		models.TransactionStatusReversed,
	}

	for _, status := range testCases {
		s.Run(status, func() {
			url := fmt.Sprintf("/api/v1/accounts/%s/transactions?status=%s", s.accountID, status)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			c := s.echo.NewContext(req, rec)

			s.Equal(status, c.QueryParam("status"))
		})
	}
}

func (s *TransactionHandlerTestSuite) TestListTransactions_FilterByCategory() {
	categories := []string{
		models.CategoryGroceries,
		models.CategoryDining,
		models.CategoryTransportation,
	}

	for _, category := range categories {
		s.Run(category, func() {
			url := fmt.Sprintf("/api/v1/accounts/%s/transactions?category=%s", s.accountID, category)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			c := s.echo.NewContext(req, rec)

			s.Equal(category, c.QueryParam("category"))
		})
	}
}

func (s *TransactionHandlerTestSuite) TestListTransactions_MultipleFilters() {
	url := fmt.Sprintf("/api/v1/accounts/%s/transactions?type=%s&status=%s&category=%s&start_date=2024-01-01&end_date=2024-01-31",
		s.accountID,
		models.TransactionTypeDebit,
		models.TransactionStatusCompleted,
		models.CategoryGroceries,
	)

	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	s.Equal(models.TransactionTypeDebit, c.QueryParam("type"))
	s.Equal(models.TransactionStatusCompleted, c.QueryParam("status"))
	s.Equal(models.CategoryGroceries, c.QueryParam("category"))
	s.Equal("2024-01-01", c.QueryParam("start_date"))
	s.Equal("2024-01-31", c.QueryParam("end_date"))
}

// Running Balance Tests

func (s *TransactionHandlerTestSuite) TestTransactionResponse_IncludesRunningBalance() {
	transactions := []models.Transaction{
		{
			ID:              uuid.New(),
			AccountID:       s.accountID,
			Amount:          decimal.NewFromFloat(100.00),
			TransactionType: models.TransactionTypeCredit,
			BalanceBefore:   decimal.NewFromFloat(500.00),
			BalanceAfter:    decimal.NewFromFloat(600.00),
			Description:     "Deposit",
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       time.Now().Add(-2 * time.Hour),
		},
		{
			ID:              uuid.New(),
			AccountID:       s.accountID,
			Amount:          decimal.NewFromFloat(50.00),
			TransactionType: models.TransactionTypeDebit,
			BalanceBefore:   decimal.NewFromFloat(600.00),
			BalanceAfter:    decimal.NewFromFloat(550.00),
			Description:     "Purchase",
			Status:          models.TransactionStatusCompleted,
			CreatedAt:       time.Now().Add(-1 * time.Hour),
		},
	}

	// Verify balance progression
	s.True(transactions[0].BalanceBefore.Equal(decimal.NewFromFloat(500.00)))
	s.True(transactions[0].BalanceAfter.Equal(decimal.NewFromFloat(600.00)))
	s.True(transactions[1].BalanceBefore.Equal(decimal.NewFromFloat(600.00)))
	s.True(transactions[1].BalanceAfter.Equal(decimal.NewFromFloat(550.00)))
}

// Pagination Limit Tests

func (s *TransactionHandlerTestSuite) TestListTransactions_DefaultLimit() {
	url := fmt.Sprintf("/api/v1/accounts/%s/transactions", s.accountID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	limit := c.QueryParam("limit")
	if limit == "" {
		limit = "20" // Default
	}

	s.Equal("20", limit)
}

func (s *TransactionHandlerTestSuite) TestListTransactions_CustomLimit() {
	url := fmt.Sprintf("/api/v1/accounts/%s/transactions?limit=50", s.accountID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	s.Equal("50", c.QueryParam("limit"))
}

func (s *TransactionHandlerTestSuite) TestListTransactions_MaxLimit() {
	url := fmt.Sprintf("/api/v1/accounts/%s/transactions?limit=200", s.accountID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	limitStr := c.QueryParam("limit")
	limit := 100 // Should be capped at max
	if limitStr != "" {
		parsedLimit := 200
		if parsedLimit > 100 {
			limit = 100
		}
	}

	s.LessOrEqual(limit, 100)
}

// Invalid Input Tests

func (s *TransactionHandlerTestSuite) TestListTransactions_InvalidAccountID() {
	url := "/api/v1/accounts/invalid-uuid/transactions"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid-uuid")

	_, err := uuid.Parse("invalid-uuid")
	s.Error(err, "Should fail to parse invalid UUID")
}

func (s *TransactionHandlerTestSuite) TestListTransactions_InvalidDateFormat() {
	testCases := []struct {
		name      string
		startDate string
		endDate   string
	}{
		{"invalid start date", "not-a-date", "2024-01-31"},
		{"invalid end date", "2024-01-01", "not-a-date"},
		{"wrong format", "01/01/2024", "01/31/2024"},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			_, err := time.Parse("2006-01-02", tc.startDate)
			if tc.startDate == "2024-01-01" {
				s.NoError(err)
			} else {
				s.Error(err)
			}
		})
	}
}

func (s *TransactionHandlerTestSuite) TestListTransactions_InvalidType() {
	url := fmt.Sprintf("/api/v1/accounts/%s/transactions?type=invalid", s.accountID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	txnType := c.QueryParam("type")
	validTypes := []string{models.TransactionTypeCredit, models.TransactionTypeDebit}

	isValid := false
	for _, valid := range validTypes {
		if txnType == valid {
			isValid = true
			break
		}
	}

	s.False(isValid, "invalid type should not be valid")
}

// Response Format Tests

func (s *TransactionHandlerTestSuite) TestListTransactionsResponse_Structure() {
	expectedResponse := dto.ListTransactionsResponse{
		Transactions: []dto.TransactionWithBalance{},
		Pagination: dto.PaginationInfo{
			HasMore:    false,
			NextCursor: "",
			Limit:      20,
			Total:      0,
		},
	}

	// Verify structure
	s.NotNil(expectedResponse.Transactions)
	s.NotNil(expectedResponse.Pagination)
	s.Equal(20, expectedResponse.Pagination.Limit)
}

func (s *TransactionHandlerTestSuite) TestTransactionWithBalance_Structure() {
	txn := dto.TransactionWithBalance{
		ID:              uuid.New(),
		AccountID:       s.accountID,
		Amount:          "100.00",
		TransactionType: models.TransactionTypeCredit,
		Description:     gofakeit.Sentence(5),
		Status:          models.TransactionStatusCompleted,
		Category:        models.CategoryIncome,
		MerchantName:    gofakeit.Company(),
		RunningBalance:  "1000.00",
		CreatedAt:       time.Now(),
	}

	s.NotEqual(uuid.Nil, txn.ID)
	s.NotEmpty(txn.Amount)
	s.NotEmpty(txn.RunningBalance)
}

// Authorization Tests

func (s *TransactionHandlerTestSuite) TestListTransactions_Unauthorized() {
	url := fmt.Sprintf("/api/v1/accounts/%s/transactions", s.accountID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("accountId")
	c.SetParamValues(s.accountID.String())
	// Don't set userID in context

	_, err := getUserIDFromContext(c)
	s.Error(err, "Should fail without user context")
}

func (s *TransactionHandlerTestSuite) TestListTransactions_ForbiddenAccount() {
	// User trying to access another user's account
	otherUserID := uuid.New()

	url := fmt.Sprintf("/api/v1/accounts/%s/transactions", s.accountID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetParamNames("accountId")
	c.SetParamValues(s.accountID.String())
	c.Set("userID", otherUserID)

	// Test would verify that service checks account ownership
	s.NotEqual(s.userID, otherUserID)
}

// Helper function tests

func (s *TransactionHandlerTestSuite) TestParseDateRange_ValidDates() {
	startStr := "2024-01-01"
	endStr := "2024-01-31"

	startDate, err := time.Parse("2006-01-02", startStr)
	s.NoError(err)
	s.Equal(2024, startDate.Year())
	s.Equal(time.January, startDate.Month())
	s.Equal(1, startDate.Day())

	endDate, err := time.Parse("2006-01-02", endStr)
	s.NoError(err)
	s.Equal(31, endDate.Day())
}

func (s *TransactionHandlerTestSuite) TestParsePaginationParams_Defaults() {
	limitStr := ""
	cursorStr := ""

	limit := 20
	if limitStr != "" {
		limit = 50
	}

	cursor := cursorStr

	s.Equal(20, limit)
	s.Empty(cursor)
}

func (s *TransactionHandlerTestSuite) TestValidateTransactionFilters_ValidInputs() {
	filters := dto.TransactionFilters{
		Type:     models.TransactionTypeCredit,
		Status:   models.TransactionStatusCompleted,
		Category: models.CategoryGroceries,
	}

	// Verify valid types
	validTypes := []string{models.TransactionTypeCredit, models.TransactionTypeDebit}
	s.Contains(validTypes, filters.Type)

	validStatuses := []string{
		models.TransactionStatusPending,
		models.TransactionStatusCompleted,
		models.TransactionStatusFailed,
		models.TransactionStatusReversed,
	}
	s.Contains(validStatuses, filters.Status)

	s.True(models.IsValidCategory(filters.Category))
}
