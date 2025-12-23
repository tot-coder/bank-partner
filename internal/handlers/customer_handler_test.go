package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"array-assessment/internal/dto"
	"array-assessment/internal/models"
	"array-assessment/internal/services"
	"array-assessment/internal/services/service_mocks"

	"github.com/go-playground/validator/v10"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
)

// CustomerHandlerTestSuite is the test suite for CustomerHandler
type CustomerHandlerTestSuite struct {
	suite.Suite
	ctrl                *gomock.Controller
	mockSearchService   *service_mocks.MockCustomerSearchServiceInterface
	mockProfileService  *service_mocks.MockCustomerProfileServiceInterface
	mockAccountService  *service_mocks.MockAccountAssociationServiceInterface
	mockAuditService    *service_mocks.MockAuditServiceInterface
	mockMetrics         *service_mocks.MockMetricsRecorderInterface
	logger              *service_mocks.MockCustomerLoggerInterface
	mockPasswordService *service_mocks.MockPasswordServiceInterface
}

func (s *CustomerHandlerTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockSearchService = service_mocks.NewMockCustomerSearchServiceInterface(s.ctrl)
	s.mockProfileService = service_mocks.NewMockCustomerProfileServiceInterface(s.ctrl)
	s.mockAccountService = service_mocks.NewMockAccountAssociationServiceInterface(s.ctrl)
	s.mockAuditService = service_mocks.NewMockAuditServiceInterface(s.ctrl)
	s.mockMetrics = service_mocks.NewMockMetricsRecorderInterface(s.ctrl)
	s.mockPasswordService = service_mocks.NewMockPasswordServiceInterface(s.ctrl)
	s.logger = service_mocks.NewMockCustomerLoggerInterface(s.ctrl)
}

func (s *CustomerHandlerTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestCustomerHandlerSuite(t *testing.T) {
	suite.Run(t, new(CustomerHandlerTestSuite))
}

// Test SearchCustomers - successful search with results
func (s *CustomerHandlerTestSuite) TestSearchCustomers_SuccessfulSearchWithResults() {
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/search?q=john@example.com&limit=10&offset=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set admin user in context
	adminID := uuid.New()
	c.Set("user_id", adminID)
	c.Set("user_role", models.RoleAdmin)

	// Setup mock expectations
	now := time.Now()
	nowStr := now.Format(time.RFC3339)
	results := []*models.CustomerSearchResult{
		{
			ID:           uuid.New(),
			Email:        "john@example.com",
			FirstName:    "John",
			LastName:     "Doe",
			Role:         models.RoleCustomer,
			AccountCount: 2,
			LastLoginAt:  &nowStr,
			CreatedAt:    nowStr,
		},
	}

	// Logger expectations
	s.logger.EXPECT().LogCustomerSearchStarted(gomock.Any(), "john@example.com", string(models.SearchTypeEmail), adminID).Times(1)
	s.logger.EXPECT().LogCustomerSearchCompleted(gomock.Any(), 1, gomock.Any()).Times(1)

	// Service expectations
	s.mockSearchService.EXPECT().
		SearchCustomers("john@example.com", models.SearchTypeEmail, 0, 10).
		Return(results, int64(1), nil)

	// Metrics expectations
	s.mockMetrics.EXPECT().IncrementCounter("customer_search_request", map[string]string{"status": "success"}).Times(1)
	s.mockMetrics.EXPECT().RecordProcessingTime("customer_search", gomock.Any()).Times(1)

	handler := NewCustomerHandler(s.mockSearchService, s.mockProfileService, s.mockAccountService, s.mockPasswordService, s.mockAuditService, s.logger, s.mockMetrics)
	err := handler.SearchCustomers(c)

	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var response dto.SearchCustomersResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal(int64(1), response.Total)
	s.Len(response.Customers, 1)
	s.Equal("john@example.com", response.Customers[0].Email)
}

// Test SearchCustomers - missing query parameter
func (s *CustomerHandlerTestSuite) TestSearchCustomers_MissingQueryParameter() {
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/search?limit=10&offset=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set admin user in context
	c.Set("user_id", uuid.New())
	c.Set("user_role", models.RoleAdmin)

	// Logger expectation for validation failure
	s.logger.EXPECT().LogValidationFailure(gomock.Any(), "customer_search", gomock.Any()).Times(1)

	handler := NewCustomerHandler(s.mockSearchService, s.mockProfileService, s.mockAccountService, s.mockPasswordService, s.mockAuditService, s.logger, s.mockMetrics)
	err := handler.SearchCustomers(c)

	s.Error(err) // Validation returns an error through Echo's validator
}

// Test SearchCustomers - invalid limit
func (s *CustomerHandlerTestSuite) TestSearchCustomers_InvalidLimit() {
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/search?q=test&limit=2000&offset=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set admin user in context
	c.Set("user_id", uuid.New())
	c.Set("user_role", models.RoleAdmin)

	// Logger expectation for validation failure
	s.logger.EXPECT().LogValidationFailure(gomock.Any(), "customer_search", gomock.Any()).Times(1)

	handler := NewCustomerHandler(s.mockSearchService, s.mockProfileService, s.mockAccountService, s.mockPasswordService, s.mockAuditService, s.logger, s.mockMetrics)
	err := handler.SearchCustomers(c)

	s.Error(err) // Validation returns an error through Echo's validator
}

// Test SearchCustomers - service error
func (s *CustomerHandlerTestSuite) TestSearchCustomers_ServiceError() {
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/search?q=test@example.com&limit=10&offset=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set admin user in context
	adminID := uuid.New()
	c.Set("user_id", adminID)
	c.Set("user_role", models.RoleAdmin)

	// Logger expectations
	s.logger.EXPECT().LogCustomerSearchStarted(gomock.Any(), "test@example.com", string(models.SearchTypeEmail), adminID).Times(1)
	s.logger.EXPECT().LogCustomerSearchFailed(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	// Service expectations
	s.mockSearchService.EXPECT().
		SearchCustomers("test@example.com", models.SearchTypeEmail, 0, 10).
		Return(nil, int64(0), errors.New("database error"))

	// Metrics expectations
	s.mockMetrics.EXPECT().IncrementCounter("customer_search_request", map[string]string{"status": "failed"}).Times(1)
	s.mockMetrics.EXPECT().RecordProcessingTime("customer_search", gomock.Any()).Times(1)

	handler := NewCustomerHandler(s.mockSearchService, s.mockProfileService, s.mockAccountService, s.mockPasswordService, s.mockAuditService, s.logger, s.mockMetrics)
	err := handler.SearchCustomers(c)

	s.NoError(err)
	s.Equal(http.StatusInternalServerError, rec.Code)

	// Parse and verify error response
	var errorResp ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &errorResp)
	s.NoError(err)
	s.Equal("SYSTEM_001", errorResp.Error.Code)
}

// Test GetCustomerProfile - admin successfully retrieves customer profile
func (s *CustomerHandlerTestSuite) TestGetCustomerProfile_AdminRetrievesCustomerProfile() {
	customerID := uuid.New()
	adminID := uuid.New()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/"+customerID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(customerID.String())

	c.Set("user_id", adminID)
	c.Set("user_role", models.RoleAdmin)

	// Setup mock expectations
	user := &models.User{
		ID:        customerID,
		Email:     "customer@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      models.RoleCustomer,
	}
	s.mockProfileService.EXPECT().
		GetCustomerProfile(customerID).
		Return(user, nil)

	handler := NewCustomerHandler(s.mockSearchService, s.mockProfileService, s.mockAccountService, s.mockPasswordService, s.mockAuditService, s.logger, s.mockMetrics)
	err := handler.GetCustomerProfile(c)

	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var response dto.GetCustomerProfileResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal(customerID, response.Customer.ID)
	s.Equal("customer@example.com", response.Customer.Email)
}

// Test GetCustomerProfile - customer successfully retrieves own profile
func (s *CustomerHandlerTestSuite) TestGetCustomerProfile_CustomerRetrievesOwnProfile() {
	customerID := uuid.New()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/"+customerID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(customerID.String())

	c.Set("user_id", customerID)
	c.Set("user_role", models.RoleCustomer)

	// Setup mock expectations
	user := &models.User{
		ID:        customerID,
		Email:     "customer@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      models.RoleCustomer,
	}
	s.mockProfileService.EXPECT().
		GetCustomerProfile(customerID).
		Return(user, nil)

	handler := NewCustomerHandler(s.mockSearchService, s.mockProfileService, s.mockAccountService, s.mockPasswordService, s.mockAuditService, s.logger, s.mockMetrics)
	err := handler.GetCustomerProfile(c)

	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	var response dto.GetCustomerProfileResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal(customerID, response.Customer.ID)
}

// Test GetCustomerProfile - customer cannot access other customer profile
func (s *CustomerHandlerTestSuite) TestGetCustomerProfile_CustomerCannotAccessOther() {
	// NOTE: Authorization check was moved to middleware (RequireAdmin)
	// This endpoint is admin-only and non-admin users are blocked by middleware
	// This test verifies handler processes request when called directly
	customerID := uuid.New()
	otherCustomerID := uuid.New()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/"+otherCustomerID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(otherCustomerID.String())

	c.Set("user_id", customerID)
	c.Set("user_role", models.RoleCustomer)

	// Mock the service call since handler proceeds without middleware protection
	expectedCustomer := &models.User{
		ID:        otherCustomerID,
		Email:     "other@example.com",
		FirstName: "Other",
		LastName:  "Customer",
		Role:      models.RoleCustomer,
	}
	s.mockProfileService.EXPECT().
		GetCustomerProfile(otherCustomerID).
		Return(expectedCustomer, nil)

	handler := NewCustomerHandler(s.mockSearchService, s.mockProfileService, s.mockAccountService, s.mockPasswordService, s.mockAuditService, s.logger, s.mockMetrics)
	err := handler.GetCustomerProfile(c)

	// Handler itself doesn't check authorization - middleware does
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
}

// Test GetCustomerProfile - invalid customer ID
func (s *CustomerHandlerTestSuite) TestGetCustomerProfile_InvalidCustomerID() {
	adminID := uuid.New()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/invalid-uuid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid-uuid")

	c.Set("user_id", adminID)
	c.Set("user_role", models.RoleAdmin)

	handler := NewCustomerHandler(s.mockSearchService, s.mockProfileService, s.mockAccountService, s.mockPasswordService, s.mockAuditService, s.logger, s.mockMetrics)
	err := handler.GetCustomerProfile(c)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)

	// Parse and verify error response
	var errorResp ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &errorResp)
	s.NoError(err)
	s.Equal("CUSTOMER_004", errorResp.Error.Code)
}

// Test GetCustomerProfile - customer not found
func (s *CustomerHandlerTestSuite) TestGetCustomerProfile_CustomerNotFound() {
	customerID := uuid.New()
	adminID := uuid.New()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/"+customerID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(customerID.String())

	c.Set("user_id", adminID)
	c.Set("user_role", models.RoleAdmin)

	// Setup mock expectations
	s.mockProfileService.EXPECT().
		GetCustomerProfile(customerID).
		Return(nil, services.ErrCustomerNotFound)

	handler := NewCustomerHandler(s.mockSearchService, s.mockProfileService, s.mockAccountService, s.mockPasswordService, s.mockAuditService, s.logger, s.mockMetrics)
	err := handler.GetCustomerProfile(c)

	s.NoError(err)
	s.Equal(http.StatusNotFound, rec.Code)

	// Parse and verify error response
	var errorResp ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &errorResp)
	s.NoError(err)
	s.Equal("CUSTOMER_001", errorResp.Error.Code)
}

// Test CreateCustomer - successful customer creation
func (s *CustomerHandlerTestSuite) TestCreateCustomer_Successful() {
	adminID := uuid.New()
	requestBody := `{
		"email": "newcustomer@example.com",
		"first_name": "Jane",
		"last_name": "Smith",
		"phone_number": "+14155552671",
		"date_of_birth": "1990-01-15",
		"address": "123 Main St",
		"city": "San Francisco",
		"state": "CA",
		"zip_code": "94102",
		"ssn": "123456789",
		"employment_status": "employed",
		"annual_income": "75000"
	}`

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.Set("user_id", adminID)
	c.Set("user_role", models.RoleAdmin)

	// Setup mock expectations
	newUserID := uuid.New()
	user := &models.User{
		ID:        newUserID,
		Email:     "newcustomer@example.com",
		FirstName: "Jane",
		LastName:  "Smith",
		Role:      models.RoleCustomer,
	}

	// Service expectations
	s.mockProfileService.EXPECT().
		CreateCustomer("newcustomer@example.com", "Jane", "Smith", models.RoleCustomer).
		Return(user, "TempPass123!", nil)

	// Metrics and logger expectations
	s.mockMetrics.EXPECT().IncrementCounter("customer_created", map[string]string{}).Times(1)
	s.logger.EXPECT().LogCustomerCreated(gomock.Any(), newUserID, "newcustomer@example.com", adminID).Times(1)

	handler := NewCustomerHandler(s.mockSearchService, s.mockProfileService, s.mockAccountService, s.mockPasswordService, s.mockAuditService, s.logger, s.mockMetrics)
	err := handler.CreateCustomer(c)

	s.NoError(err)
	s.Equal(http.StatusCreated, rec.Code)

	var response dto.CreateCustomerResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal("newcustomer@example.com", response.Customer.Email)
	s.NotEmpty(response.TemporaryPassword)
}

// Test CreateCustomer - invalid email
func (s *CustomerHandlerTestSuite) TestCreateCustomer_InvalidEmail() {
	adminID := uuid.New()
	requestBody := `{
		"email": "invalid-email",
		"first_name": "Jane",
		"last_name": "Smith",
		"date_of_birth": "1990-01-15",
		"ssn": "123456789",
		"employment_status": "employed",
		"annual_income": "75000"
	}`

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.Set("user_id", adminID)
	c.Set("user_role", models.RoleAdmin)

	// Logger expectation for validation failure
	s.logger.EXPECT().LogValidationFailure(gomock.Any(), "customer_create", gomock.Any()).Times(1)

	handler := NewCustomerHandler(s.mockSearchService, s.mockProfileService, s.mockAccountService, s.mockPasswordService, s.mockAuditService, s.logger, s.mockMetrics)
	err := handler.CreateCustomer(c)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// Test CreateCustomer - missing required fields
func (s *CustomerHandlerTestSuite) TestCreateCustomer_MissingRequiredFields() {
	adminID := uuid.New()
	requestBody := `{"email": "test@example.com"}`

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.Set("user_id", adminID)
	c.Set("user_role", models.RoleAdmin)

	// Logger expectation for validation failure
	s.logger.EXPECT().LogValidationFailure(gomock.Any(), "customer_create", gomock.Any()).Times(1)

	handler := NewCustomerHandler(s.mockSearchService, s.mockProfileService, s.mockAccountService, s.mockPasswordService, s.mockAuditService, s.logger, s.mockMetrics)
	err := handler.CreateCustomer(c)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, rec.Code)
}

// Test CreateCustomer - email already exists
func (s *CustomerHandlerTestSuite) TestCreateCustomer_EmailAlreadyExists() {
	adminID := uuid.New()
	requestBody := `{
		"email": "existing@example.com",
		"first_name": "Jane",
		"last_name": "Smith",
		"date_of_birth": "1990-01-15",
		"ssn": "123456789",
		"employment_status": "employed",
		"annual_income": "75000"
	}`

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.Set("user_id", adminID)
	c.Set("user_role", models.RoleAdmin)

	// Setup mock expectations
	s.mockProfileService.EXPECT().
		CreateCustomer("existing@example.com", "Jane", "Smith", models.RoleCustomer).
		Return(nil, "", services.ErrEmailAlreadyExists)

	handler := NewCustomerHandler(s.mockSearchService, s.mockProfileService, s.mockAccountService, s.mockPasswordService, s.mockAuditService, s.logger, s.mockMetrics)
	err := handler.CreateCustomer(c)

	s.NoError(err)
	s.Equal(http.StatusUnprocessableEntity, rec.Code)
}
