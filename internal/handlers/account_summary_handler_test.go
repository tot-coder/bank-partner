package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"array-assessment/internal/models"
	"array-assessment/internal/services"
	"array-assessment/internal/services/service_mocks"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type AccountSummaryHandlerTestSuite struct {
	suite.Suite
	ctrl                 *gomock.Controller
	echo                 *echo.Echo
	mockSummaryService   *service_mocks.MockAccountSummaryServiceInterface
	mockMetricsService   *service_mocks.MockAccountMetricsServiceInterface
	mockStatementService *service_mocks.MockStatementServiceInterface
	handler              *AccountSummaryHandler
	regularUserID        uuid.UUID
	adminUserID          uuid.UUID
	otherUserID          uuid.UUID
	accountID            uuid.UUID
}

func TestAccountSummaryHandlerSuite(t *testing.T) {
	suite.Run(t, new(AccountSummaryHandlerTestSuite))
}

func (s *AccountSummaryHandlerTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.echo = echo.New()
	s.mockSummaryService = service_mocks.NewMockAccountSummaryServiceInterface(s.ctrl)
	s.mockMetricsService = service_mocks.NewMockAccountMetricsServiceInterface(s.ctrl)
	s.mockStatementService = service_mocks.NewMockStatementServiceInterface(s.ctrl)
	s.handler = NewAccountSummaryHandler(s.mockSummaryService, s.mockMetricsService, s.mockStatementService)
	s.regularUserID = uuid.New()
	s.adminUserID = uuid.New()
	s.otherUserID = uuid.New()
	s.accountID = uuid.New()
}

func (s *AccountSummaryHandlerTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

// ========================================
// GET /api/v1/accounts/summary Tests
// ========================================

func (s *AccountSummaryHandlerTestSuite) TestGetAccountSummary_RegularUser_Success() {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/summary", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)

	summary := &models.UserAccountSummary{
		UserID:       s.regularUserID,
		TotalBalance: decimal.NewFromFloat(15000.50),
		AccountCount: 2,
		Currency:     "USD",
		Accounts: []models.AccountSummaryItem{
			{
				ID:                  uuid.New(),
				MaskedAccountNumber: "****1234",
				AccountType:         "checking",
				Balance:             decimal.NewFromFloat(5000.50),
				Status:              "active",
				Currency:            "USD",
			},
			{
				ID:                  uuid.New(),
				MaskedAccountNumber: "****5678",
				AccountType:         "savings",
				Balance:             decimal.NewFromFloat(10000.00),
				Status:              "active",
				Currency:            "USD",
			},
		},
		GeneratedAt: time.Now().Format(time.RFC3339),
	}

	s.mockSummaryService.EXPECT().
		GetAccountSummary(s.regularUserID, (*uuid.UUID)(nil), false).
		Return(summary, nil)

	err := s.handler.GetAccountSummary(c)

	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.NoError(err)
	s.NotNil(response["data"])
}

func (s *AccountSummaryHandlerTestSuite) TestGetAccountSummary_AdminAccessingOtherUser_Success() {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/summary?userId=%s", s.otherUserID.String()), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.Set("user_id", s.adminUserID)
	c.Set("is_admin", true)
	c.QueryParams().Add("userId", s.otherUserID.String())

	summary := &models.UserAccountSummary{
		UserID:       s.otherUserID,
		TotalBalance: decimal.NewFromFloat(25000.00),
		AccountCount: 3,
		Currency:     "USD",
		Accounts:     []models.AccountSummaryItem{},
		GeneratedAt:  time.Now().Format(time.RFC3339),
	}

	s.mockSummaryService.EXPECT().
		GetAccountSummary(s.adminUserID, &s.otherUserID, true).
		Return(summary, nil)

	err := s.handler.GetAccountSummary(c)

	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *AccountSummaryHandlerTestSuite) TestGetAccountSummary_Unauthorized_MissingContext() {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/summary", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	err := s.handler.GetAccountSummary(c)

	// Handler now returns nil, error is written to response with standardized error handling
	s.NoError(err)
	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *AccountSummaryHandlerTestSuite) TestGetAccountSummary_InvalidUserID_QueryParam() {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/summary?userId=invalid-uuid", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("userId", "invalid-uuid")

	err := s.handler.GetAccountSummary(c)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
	s.Contains(rec.Body.String(), "VALIDATION_001")
}

func (s *AccountSummaryHandlerTestSuite) TestGetAccountSummary_ServiceError_Unauthorized() {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/summary?userId=%s", s.otherUserID.String()), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("userId", s.otherUserID.String())

	s.mockSummaryService.EXPECT().
		GetAccountSummary(s.regularUserID, &s.otherUserID, false).
		Return(nil, services.ErrUnauthorized)

	err := s.handler.GetAccountSummary(c)

	s.NoError(err)
	s.Equal(http.StatusForbidden, rec.Code)
	s.Contains(rec.Body.String(), "AUTH_005")
}

func (s *AccountSummaryHandlerTestSuite) TestGetAccountSummary_ServiceError_NotFound() {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/summary", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)

	s.mockSummaryService.EXPECT().
		GetAccountSummary(s.regularUserID, (*uuid.UUID)(nil), false).
		Return(nil, services.ErrNotFound)

	err := s.handler.GetAccountSummary(c)

	// Handler maps ErrNotFound to AccountNotFound (ACCOUNT_001)
	s.NoError(err)
	s.Equal(http.StatusNotFound, rec.Code)
	s.Contains(rec.Body.String(), "ACCOUNT_001")
}

// ========================================
// GET /api/v1/accounts/metrics Tests
// ========================================

func (s *AccountSummaryHandlerTestSuite) TestGetAccountMetrics_WithDates_Success() {
	startDate := "2024-01-01"
	endDate := "2024-01-31"
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/metrics?accountId=%s&startDate=%s&endDate=%s", s.accountID.String(), startDate, endDate), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("accountId", s.accountID.String())
	c.QueryParams().Add("startDate", startDate)
	c.QueryParams().Add("endDate", endDate)

	parsedStart, _ := time.Parse("2006-01-02", startDate)
	parsedEnd, _ := time.Parse("2006-01-02", endDate)

	metrics := &models.AccountMetrics{
		AccountID:                s.accountID,
		StartDate:                parsedStart,
		EndDate:                  parsedEnd,
		TotalDeposits:            decimal.NewFromFloat(5000.00),
		TotalWithdrawals:         decimal.NewFromFloat(2000.00),
		NetChange:                decimal.NewFromFloat(3000.00),
		TransactionCount:         15,
		DepositCount:             10,
		WithdrawalCount:          5,
		AverageTransactionAmount: decimal.NewFromFloat(333.33),
		LargestDeposit:           decimal.NewFromFloat(1000.00),
		LargestWithdrawal:        decimal.NewFromFloat(500.00),
		AverageDailyBalance:      decimal.NewFromFloat(8000.00),
		InterestEarned:           decimal.NewFromFloat(25.50),
		GeneratedAt:              time.Now(),
	}

	s.mockMetricsService.EXPECT().
		GetAccountMetrics(s.regularUserID, s.accountID, &parsedStart, &parsedEnd, false).
		Return(metrics, nil)

	err := s.handler.GetAccountMetrics(c)

	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.NoError(err)
	s.NotNil(response["data"])
}

func (s *AccountSummaryHandlerTestSuite) TestGetAccountMetrics_WithoutDates_Success() {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/metrics?accountId=%s", s.accountID.String()), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("accountId", s.accountID.String())

	metrics := &models.AccountMetrics{
		AccountID:        s.accountID,
		StartDate:        time.Now().AddDate(0, 0, -14),
		EndDate:          time.Now(),
		TotalDeposits:    decimal.NewFromFloat(1000.00),
		TotalWithdrawals: decimal.NewFromFloat(500.00),
		GeneratedAt:      time.Now(),
	}

	s.mockMetricsService.EXPECT().
		GetAccountMetrics(s.regularUserID, s.accountID, (*time.Time)(nil), (*time.Time)(nil), false).
		Return(metrics, nil)

	err := s.handler.GetAccountMetrics(c)

	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *AccountSummaryHandlerTestSuite) TestGetAccountMetrics_MissingAccountID() {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/metrics", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)

	err := s.handler.GetAccountMetrics(c)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
	s.Contains(rec.Body.String(), "VALIDATION_001")
}

func (s *AccountSummaryHandlerTestSuite) TestGetAccountMetrics_InvalidAccountID() {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/metrics?accountId=invalid-uuid", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("accountId", "invalid-uuid")

	err := s.handler.GetAccountMetrics(c)

	// Handler uses SendError which sends response directly and returns nil
	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
	s.Contains(rec.Body.String(), "VALIDATION_001")
}

func (s *AccountSummaryHandlerTestSuite) TestGetAccountMetrics_InvalidDateFormat() {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/metrics?accountId=%s&startDate=invalid-date", s.accountID.String()), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("accountId", s.accountID.String())
	c.QueryParams().Add("startDate", "invalid-date")

	err := s.handler.GetAccountMetrics(c)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
	s.Contains(rec.Body.String(), "VALIDATION_001")
}

func (s *AccountSummaryHandlerTestSuite) TestGetAccountMetrics_ServiceError_Unauthorized() {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/metrics?accountId=%s", s.accountID.String()), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("accountId", s.accountID.String())

	s.mockMetricsService.EXPECT().
		GetAccountMetrics(s.regularUserID, s.accountID, (*time.Time)(nil), (*time.Time)(nil), false).
		Return(nil, services.ErrUnauthorized)

	err := s.handler.GetAccountMetrics(c)

	s.NoError(err)
	s.Equal(http.StatusForbidden, rec.Code)
	s.Contains(rec.Body.String(), "AUTH_005")
}

// ========================================
// GET /api/v1/accounts/:accountId/statements Tests
// ========================================

func (s *AccountSummaryHandlerTestSuite) TestGetStatement_Monthly_Success() {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/statements?periodType=monthly&year=2024&period=1", s.accountID.String()), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetPath("/api/v1/accounts/:accountId/statements")
	c.SetParamNames("accountId")
	c.SetParamValues(s.accountID.String())
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("periodType", "monthly")
	c.QueryParams().Add("year", "2024")
	c.QueryParams().Add("period", "1")

	statement := &models.AccountStatement{
		AccountID:      s.accountID,
		AccountNumber:  gofakeit.Numerify("##########"),
		AccountType:    "checking",
		PeriodType:     "monthly",
		Year:           2024,
		Period:         1,
		StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		OpeningBalance: decimal.NewFromFloat(10000.00),
		ClosingBalance: decimal.NewFromFloat(12000.00),
		Transactions:   []models.StatementTransaction{},
		Summary: models.StatementSummary{
			TotalDeposits:    decimal.NewFromFloat(5000.00),
			TotalWithdrawals: decimal.NewFromFloat(3000.00),
			NetChange:        decimal.NewFromFloat(2000.00),
			TransactionCount: 10,
		},
		GeneratedAt: time.Now(),
	}

	s.mockStatementService.EXPECT().
		GenerateStatement(s.regularUserID, s.accountID, "monthly", 2024, 1, false).
		Return(statement, nil)

	err := s.handler.GetStatement(c)

	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.NoError(err)
	s.NotNil(response["data"])
}

func (s *AccountSummaryHandlerTestSuite) TestGetStatement_Quarterly_Success() {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/statements?periodType=quarterly&year=2024&period=2", s.accountID.String()), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetPath("/api/v1/accounts/:accountId/statements")
	c.SetParamNames("accountId")
	c.SetParamValues(s.accountID.String())
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("periodType", "quarterly")
	c.QueryParams().Add("year", "2024")
	c.QueryParams().Add("period", "2")

	statement := &models.AccountStatement{
		AccountID:      s.accountID,
		AccountNumber:  gofakeit.Numerify("##########"),
		AccountType:    "savings",
		PeriodType:     "quarterly",
		Year:           2024,
		Period:         2,
		StartDate:      time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC),
		OpeningBalance: decimal.NewFromFloat(20000.00),
		ClosingBalance: decimal.NewFromFloat(22500.00),
		Transactions:   []models.StatementTransaction{},
		Summary: models.StatementSummary{
			TotalDeposits:    decimal.NewFromFloat(10000.00),
			TotalWithdrawals: decimal.NewFromFloat(7500.00),
			NetChange:        decimal.NewFromFloat(2500.00),
			TransactionCount: 25,
		},
		GeneratedAt: time.Now(),
	}

	s.mockStatementService.EXPECT().
		GenerateStatement(s.regularUserID, s.accountID, "quarterly", 2024, 2, false).
		Return(statement, nil)

	err := s.handler.GetStatement(c)

	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *AccountSummaryHandlerTestSuite) TestGetStatement_MissingAccountID() {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts//statements?periodType=monthly&year=2024&period=1", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetPath("/api/v1/accounts/:accountId/statements")
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("periodType", "monthly")
	c.QueryParams().Add("year", "2024")
	c.QueryParams().Add("period", "1")

	err := s.handler.GetStatement(c)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
	s.Contains(rec.Body.String(), "VALIDATION_001")
}

func (s *AccountSummaryHandlerTestSuite) TestGetStatement_InvalidAccountID() {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/invalid-uuid/statements?periodType=monthly&year=2024&period=1", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetPath("/api/v1/accounts/:accountId/statements")
	c.SetParamNames("accountId")
	c.SetParamValues("invalid-uuid")
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("periodType", "monthly")
	c.QueryParams().Add("year", "2024")
	c.QueryParams().Add("period", "1")

	err := s.handler.GetStatement(c)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
	s.Contains(rec.Body.String(), "VALIDATION_001")
}

func (s *AccountSummaryHandlerTestSuite) TestGetStatement_MissingPeriodType() {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/statements?year=2024&period=1", s.accountID.String()), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetPath("/api/v1/accounts/:accountId/statements")
	c.SetParamNames("accountId")
	c.SetParamValues(s.accountID.String())
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("year", "2024")
	c.QueryParams().Add("period", "1")

	err := s.handler.GetStatement(c)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
	s.Contains(rec.Body.String(), "VALIDATION_001")
}

func (s *AccountSummaryHandlerTestSuite) TestGetStatement_InvalidPeriodType() {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/statements?periodType=invalid&year=2024&period=1", s.accountID.String()), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetPath("/api/v1/accounts/:accountId/statements")
	c.SetParamNames("accountId")
	c.SetParamValues(s.accountID.String())
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("periodType", "invalid")
	c.QueryParams().Add("year", "2024")
	c.QueryParams().Add("period", "1")

	err := s.handler.GetStatement(c)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
	s.Contains(rec.Body.String(), "VALIDATION_001")
}

func (s *AccountSummaryHandlerTestSuite) TestGetStatement_MissingYearOrPeriod() {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/statements?periodType=monthly", s.accountID.String()), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetPath("/api/v1/accounts/:accountId/statements")
	c.SetParamNames("accountId")
	c.SetParamValues(s.accountID.String())
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("periodType", "monthly")

	err := s.handler.GetStatement(c)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
	s.Contains(rec.Body.String(), "VALIDATION_001")
}

func (s *AccountSummaryHandlerTestSuite) TestGetStatement_InvalidYearFormat() {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/statements?periodType=monthly&year=invalid&period=1", s.accountID.String()), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetPath("/api/v1/accounts/:accountId/statements")
	c.SetParamNames("accountId")
	c.SetParamValues(s.accountID.String())
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("periodType", "monthly")
	c.QueryParams().Add("year", "invalid")
	c.QueryParams().Add("period", "1")

	err := s.handler.GetStatement(c)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
	s.Contains(rec.Body.String(), "VALIDATION_001")
}

func (s *AccountSummaryHandlerTestSuite) TestGetStatement_ServiceError_InvalidPeriod() {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/statements?periodType=monthly&year=2024&period=13", s.accountID.String()), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetPath("/api/v1/accounts/:accountId/statements")
	c.SetParamNames("accountId")
	c.SetParamValues(s.accountID.String())
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("periodType", "monthly")
	c.QueryParams().Add("year", "2024")
	c.QueryParams().Add("period", "13")

	s.mockStatementService.EXPECT().
		GenerateStatement(s.regularUserID, s.accountID, "monthly", 2024, 13, false).
		Return(nil, services.ErrInvalidMonth)

	err := s.handler.GetStatement(c)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
	s.Contains(rec.Body.String(), "VALIDATION_001")
}

func (s *AccountSummaryHandlerTestSuite) TestGetStatement_ServiceError_Unauthorized() {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/accounts/%s/statements?periodType=monthly&year=2024&period=1", s.accountID.String()), nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetPath("/api/v1/accounts/:accountId/statements")
	c.SetParamNames("accountId")
	c.SetParamValues(s.accountID.String())
	c.Set("user_id", s.regularUserID)
	c.Set("is_admin", false)
	c.QueryParams().Add("periodType", "monthly")
	c.QueryParams().Add("year", "2024")
	c.QueryParams().Add("period", "1")

	s.mockStatementService.EXPECT().
		GenerateStatement(s.regularUserID, s.accountID, "monthly", 2024, 1, false).
		Return(nil, services.ErrUnauthorized)

	err := s.handler.GetStatement(c)

	s.NoError(err)
	s.Equal(http.StatusForbidden, rec.Code)
	s.Contains(rec.Body.String(), "AUTH_005")
}
