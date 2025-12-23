package handlers

import (
	"net/http"
	"strconv"
	"time"

	"array-assessment/internal/dto"
	"array-assessment/internal/errors"
	"array-assessment/internal/models"
	"array-assessment/internal/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// CustomerHandler handles customer-related HTTP requests
type CustomerHandler struct {
	searchService   services.CustomerSearchServiceInterface
	profileService  services.CustomerProfileServiceInterface
	accountService  services.AccountAssociationServiceInterface
	passwordService services.PasswordServiceInterface
	auditService    services.AuditServiceInterface
	logger          services.CustomerLoggerInterface
	metrics         services.MetricsRecorderInterface
}

// NewCustomerHandler creates a new customer handler
func NewCustomerHandler(
	searchService services.CustomerSearchServiceInterface,
	profileService services.CustomerProfileServiceInterface,
	accountService services.AccountAssociationServiceInterface,
	passwordService services.PasswordServiceInterface,
	auditService services.AuditServiceInterface,
	logger services.CustomerLoggerInterface,
	metrics services.MetricsRecorderInterface,
) *CustomerHandler {
	return &CustomerHandler{
		searchService:   searchService,
		profileService:  profileService,
		accountService:  accountService,
		passwordService: passwordService,
		auditService:    auditService,
		logger:          logger,
		metrics:         metrics,
	}
}

// SearchCustomers searches for customers (admin only)
// @Summary Search customers (admin)
// @Description Admin endpoint to search for customers by email, name, or account number
// @Tags Customers
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param type query string false "Search type" Enums(email, name, first_name, last_name, account_number) default(email)
// @Param limit query int false "Results limit (max 1000)" default(10)
// @Param offset query int false "Results offset" default(0)
// @Success 200 {object} dto.SearchCustomersResponse "Customer search results"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_001 - Invalid request parameters"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /customers/search [get]
func (h *CustomerHandler) SearchCustomers(c echo.Context) error {
	startTime := time.Now()
	ctx := c.Request().Context()

	adminUserID, err := getUserIDFromContext(c)
	if err != nil {
		h.logger.LogAuthorizationFailure(ctx, "customer_search", uuid.Nil, "admin")
		return SendError(c, errors.AuthMissingToken)
	}

	var req dto.SearchCustomersRequest
	if err := c.Bind(&req); err != nil {
		h.logger.LogValidationFailure(ctx, "customer_search", err.Error())
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("Invalid request parameters"))
	}

	if err := c.Validate(req); err != nil {
		h.logger.LogValidationFailure(ctx, "customer_search", err.Error())
		return err
	}

	if req.Limit == 0 {
		req.Limit = 10
	}

	searchType := models.SearchTypeEmail

	h.logger.LogCustomerSearchStarted(ctx, req.Query, string(searchType), adminUserID)

	results, total, err := h.searchService.SearchCustomers(req.Query, searchType, req.Offset, req.Limit)
	duration := time.Since(startTime)

	if err != nil {
		h.metrics.IncrementCounter("customer_search_request", map[string]string{"status": "failed"})
		h.metrics.RecordProcessingTime("customer_search", duration)
		h.logger.LogCustomerSearchFailed(ctx, err.Error(), duration.Milliseconds())
		return SendSystemError(c, err)
	}

	h.metrics.IncrementCounter("customer_search_request", map[string]string{"status": "success"})
	h.metrics.RecordProcessingTime("customer_search", duration)
	h.logger.LogCustomerSearchCompleted(ctx, len(results), duration.Milliseconds())

	customers := make([]*dto.CustomerSearchResult, len(results))
	for i, result := range results {
		var lastLoginAt *time.Time
		if result.LastLoginAt != nil {
			parsed, _ := time.Parse(time.RFC3339, *result.LastLoginAt)
			lastLoginAt = &parsed
		}

		createdAt, _ := time.Parse(time.RFC3339, result.CreatedAt)

		customers[i] = &dto.CustomerSearchResult{
			ID:           result.ID,
			Email:        result.Email,
			FirstName:    result.FirstName,
			LastName:     result.LastName,
			Status:       result.Role, // Using Role as Status for MVP
			AccountCount: int(result.AccountCount),
			LastLoginAt:  lastLoginAt,
			CreatedAt:    createdAt,
		}
	}

	totalPages := int(total) / req.Limit
	if int(total)%req.Limit > 0 {
		totalPages++
	}

	return c.JSON(http.StatusOK, dto.SearchCustomersResponse{
		Customers:  customers,
		Total:      total,
		Limit:      req.Limit,
		Offset:     req.Offset,
		TotalPages: totalPages,
	})
}

// GetCustomerProfile retrieves a customer profile by ID
// @Summary Get customer profile (admin)
// @Description Admin endpoint to retrieve detailed customer profile by customer ID
// @Tags Customers
// @Security BearerAuth
// @Produce json
// @Param id path string true "Customer ID (UUID)"
// @Success 200 {object} dto.GetCustomerProfileResponse "Customer profile"
// @Failure 400 {object} errors.ErrorResponse "CUSTOMER_004 - Invalid customer ID format"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 404 {object} errors.ErrorResponse "CUSTOMER_001 - Customer not found"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /customers/{id} [get]
func (h *CustomerHandler) GetCustomerProfile(c echo.Context) error {
	customerIDStr := c.Param("id")
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		return SendError(c, errors.CustomerInvalidID)
	}

	customer, err := h.profileService.GetCustomerProfile(customerID)
	if err != nil {
		if err == services.ErrCustomerNotFound {
			return SendError(c, errors.CustomerNotFound)
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, dto.GetCustomerProfileResponse{
		Customer: customer,
	})
}

// GetMyProfile retrieves the authenticated customer's profile
// @Summary Get my profile
// @Description Retrieve the authenticated customer's profile information
// @Tags Customers
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.GetCustomerProfileResponse "Customer profile"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 404 {object} errors.ErrorResponse "CUSTOMER_001 - Customer not found"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /customers/me [get]
func (h *CustomerHandler) GetMyProfile(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	customer, err := h.profileService.GetCustomerProfile(userID)
	if err != nil {
		if err == services.ErrCustomerNotFound {
			return SendError(c, errors.CustomerNotFound)
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, dto.GetCustomerProfileResponse{
		Customer: customer,
	})
}

// CreateCustomer creates a new customer (admin only)
// @Summary Create customer (admin)
// @Description Admin endpoint to create a new customer with auto-generated temporary password
// @Tags Customers
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.CreateCustomerRequest true "Customer details"
// @Success 201 {object} dto.CreateCustomerResponse "Customer created successfully with temporary password"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_001 - Invalid request body"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 422 {object} errors.ErrorResponse "CUSTOMER_002 - Email already exists"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /customers [post]
func (h *CustomerHandler) CreateCustomer(c echo.Context) error {
	ctx := c.Request().Context()

	adminUserID, err := getUserIDFromContext(c)
	if err != nil {
		h.logger.LogAuthorizationFailure(ctx, "customer_create", uuid.Nil, "admin")
		return SendError(c, errors.AuthMissingToken)
	}

	var req dto.CreateCustomerRequest
	if err := c.Bind(&req); err != nil {
		h.logger.LogValidationFailure(ctx, "customer_create", err.Error())
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("Invalid request body"))
	}

	if err := c.Validate(req); err != nil {
		h.logger.LogValidationFailure(ctx, "customer_create", err.Error())
		return SendError(c, errors.ValidationInvalidFormat, errors.WithDetails(err.Error()))
	}

	customer, tempPassword, err := h.profileService.CreateCustomer(req.Email, req.FirstName, req.LastName, models.RoleCustomer)
	if err != nil {
		if err == services.ErrEmailAlreadyExists {
			return SendError(c, errors.CustomerAlreadyExists)
		}
		return SendSystemError(c, err)
	}

	h.metrics.IncrementCounter("customer_created", map[string]string{})
	h.logger.LogCustomerCreated(ctx, customer.ID, customer.Email, adminUserID)

	return c.JSON(http.StatusCreated, dto.CreateCustomerResponse{
		Customer:          customer,
		TemporaryPassword: tempPassword,
		Message:           "Customer created successfully",
	})
}

// UpdateCustomerProfile updates a customer profile (admin only)
// @Summary Update customer profile (admin)
// @Description Admin endpoint to update customer profile fields
// @Tags Customers
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Customer ID (UUID)"
// @Param request body dto.UpdateCustomerProfileRequest true "Profile updates"
// @Success 200 {object} SuccessResponse{message=string} "Profile updated successfully"
// @Failure 400 {object} errors.ErrorResponse "CUSTOMER_004 - Invalid customer ID or VALIDATION_001 - Invalid request body"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 404 {object} errors.ErrorResponse "CUSTOMER_001 - Customer not found"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /customers/{id} [put]
func (h *CustomerHandler) UpdateCustomerProfile(c echo.Context) error {
	customerIDStr := c.Param("id")
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		return SendError(c, errors.CustomerInvalidID)
	}

	var req dto.UpdateCustomerProfileRequest
	if err := c.Bind(&req); err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("Invalid request body"))
	}

	if err := c.Validate(req); err != nil {
		return err
	}

	updates := make(map[string]interface{})
	if req.FirstName != nil {
		updates["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		updates["last_name"] = *req.LastName
	}
	if req.PhoneNumber != nil {
		updates["phone_number"] = *req.PhoneNumber
	}
	if req.Address != nil {
		updates["address"] = *req.Address
	}
	if req.City != nil {
		updates["city"] = *req.City
	}
	if req.State != nil {
		updates["state"] = *req.State
	}
	if req.ZipCode != nil {
		updates["zip_code"] = *req.ZipCode
	}

	if len(updates) == 0 {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("At least one field must be provided for update"))
	}

	err = h.profileService.UpdateCustomerProfile(customerID, updates)
	if err != nil {
		if err == services.ErrCustomerNotFound {
			return SendError(c, errors.CustomerNotFound)
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Profile updated successfully",
	})
}

// UpdateMyEmail updates the authenticated customer's email
// @Summary Update my email
// @Description Update the authenticated customer's email address
// @Tags Customers
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.UpdateCustomerEmailRequest true "New email address"
// @Success 200 {object} dto.UpdateCustomerEmailResponse "Email updated successfully"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_001 - Invalid request body or VALIDATION_005 - Invalid email format"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 404 {object} errors.ErrorResponse "CUSTOMER_001 - Customer not found"
// @Failure 422 {object} errors.ErrorResponse "CUSTOMER_002 - Email already exists"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /customers/me/email [put]
func (h *CustomerHandler) UpdateMyEmail(c echo.Context) error {
	var req dto.UpdateCustomerEmailRequest
	if err := c.Bind(&req); err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("Invalid request body"))
	}

	if err := c.Validate(req); err != nil {
		return SendError(c, errors.ValidationInvalidFormat, errors.WithDetails(err.Error()))
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	err = h.profileService.UpdateCustomerEmail(userID, req.NewEmail)
	if err != nil {
		if err == services.ErrEmailAlreadyExists {
			return SendError(c, errors.CustomerAlreadyExists)
		}
		if err == services.ErrCustomerNotFound {
			return SendError(c, errors.CustomerNotFound)
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, dto.UpdateCustomerEmailResponse{
		Message: "Email updated successfully",
	})
}

// DeleteCustomer soft-deletes a customer (admin only)
// @Summary Delete customer (admin)
// @Description Admin endpoint to soft-delete a customer. Cannot delete customers with non-zero account balances.
// @Tags Customers
// @Security BearerAuth
// @Produce json
// @Param id path string true "Customer ID (UUID)"
// @Success 200 {object} dto.DeleteCustomerResponse "Customer deleted successfully"
// @Failure 400 {object} errors.ErrorResponse "CUSTOMER_004 - Invalid customer ID"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 404 {object} errors.ErrorResponse "CUSTOMER_001 - Customer not found"
// @Failure 422 {object} errors.ErrorResponse "ACCOUNT_005 - Customer has non-zero balances"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /customers/{id} [delete]
func (h *CustomerHandler) DeleteCustomer(c echo.Context) error {
	customerIDStr := c.Param("id")
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		return SendError(c, errors.CustomerInvalidID)
	}

	err = h.profileService.DeleteCustomer(customerID, "Admin deletion")
	if err != nil {
		if err == services.ErrCustomerNotFound {
			return SendError(c, errors.CustomerNotFound)
		}
		if err == services.ErrCustomerHasBalance {
			return SendError(c, errors.AccountOperationNotPermitted, errors.WithDetails("Cannot delete customer with non-zero balances"))
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, dto.DeleteCustomerResponse{
		Message: "Customer deleted successfully",
	})
}

// GetCustomerAccounts retrieves accounts for a customer
// @Summary Get customer accounts (admin)
// @Description Admin endpoint to retrieve all accounts for a specific customer
// @Tags Customers
// @Security BearerAuth
// @Produce json
// @Param id path string true "Customer ID (UUID)"
// @Success 200 {object} object{accounts=[]models.Account,count=int} "Customer accounts"
// @Failure 400 {object} errors.ErrorResponse "CUSTOMER_004 - Invalid customer ID"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 404 {object} errors.ErrorResponse "CUSTOMER_001 - Customer not found"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /customers/{id}/accounts [get]
func (h *CustomerHandler) GetCustomerAccounts(c echo.Context) error {
	customerIDStr := c.Param("id")
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		return SendError(c, errors.CustomerInvalidID)
	}

	accounts, err := h.accountService.GetCustomerAccounts(customerID)
	if err != nil {
		if err == services.ErrCustomerNotFound {
			return SendError(c, errors.CustomerNotFound)
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"accounts": accounts,
		"count":    len(accounts),
	})
}

// GetMyAccounts retrieves the authenticated customer's accounts
// @Summary Get my accounts
// @Description Retrieve all accounts for the authenticated customer
// @Tags Customers
// @Security BearerAuth
// @Produce json
// @Success 200 {object} object{accounts=[]models.Account,count=int} "Customer accounts"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 404 {object} errors.ErrorResponse "CUSTOMER_001 - Customer not found"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /customers/me/accounts [get]
func (h *CustomerHandler) GetMyAccounts(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	accounts, err := h.accountService.GetCustomerAccounts(userID)
	if err != nil {
		if err == services.ErrCustomerNotFound {
			return SendError(c, errors.CustomerNotFound)
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"accounts": accounts,
		"count":    len(accounts),
	})
}

// CreateAccountForCustomer creates an account for a customer (admin only)
// @Summary Create account for customer (admin)
// @Description Admin endpoint to create a new account for a specific customer
// @Tags Customers
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Customer ID (UUID)"
// @Param request body object{account_type=string} true "Account type (checking, savings, money_market)"
// @Success 201 {object} object{account=models.Account,message=string} "Account created successfully"
// @Failure 400 {object} errors.ErrorResponse "CUSTOMER_004 - Invalid customer ID or VALIDATION_001 - Invalid request body"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 404 {object} errors.ErrorResponse "CUSTOMER_001 - Customer not found"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /customers/{id}/accounts [post]
func (h *CustomerHandler) CreateAccountForCustomer(c echo.Context) error {
	customerIDStr := c.Param("id")
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		return SendError(c, errors.CustomerInvalidID)
	}

	var req struct {
		AccountType string `json:"account_type" validate:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("Invalid request body"))
	}

	if err := c.Validate(req); err != nil {
		return SendError(c, errors.ValidationInvalidFormat, errors.WithDetails(err.Error()))
	}

	adminID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	account, err := h.accountService.CreateAccountForCustomer(customerID, adminID, req.AccountType, ipAddress, userAgent)
	if err != nil {
		if err == services.ErrCustomerNotFound {
			return SendError(c, errors.CustomerNotFound)
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"account": account,
		"message": "Account created successfully",
	})
}

// TransferAccountOwnership transfers account ownership (admin only)
// @Summary Transfer account ownership (admin)
// @Description Admin endpoint to transfer account ownership from one customer to another
// @Tags Customers
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param accountId path string true "Account ID (UUID)"
// @Param request body object{from_customer_id=string,to_customer_id=string} true "Transfer details"
// @Success 200 {object} SuccessResponse{message=string} "Ownership transferred successfully"
// @Failure 400 {object} errors.ErrorResponse "ACCOUNT_004 - Invalid account ID or VALIDATION_001 - Invalid request body"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 404 {object} errors.ErrorResponse "ACCOUNT_001 - Account not found or CUSTOMER_001 - Customer not found"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /accounts/{accountId}/transfer-ownership [post]
func (h *CustomerHandler) TransferAccountOwnership(c echo.Context) error {
	accountIDStr := c.Param("accountId")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return SendError(c, errors.AccountInvalidNumber)
	}

	var req struct {
		FromCustomerID uuid.UUID `json:"from_customer_id" validate:"required"`
		ToCustomerID   uuid.UUID `json:"to_customer_id" validate:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("Invalid request body"))
	}

	if err := c.Validate(req); err != nil {
		return SendError(c, errors.ValidationInvalidFormat, errors.WithDetails(err.Error()))
	}

	adminID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	err = h.accountService.TransferAccountOwnership(accountID, req.FromCustomerID, req.ToCustomerID, adminID, ipAddress, userAgent)
	if err != nil {
		if err == services.ErrAccountNotFound {
			return SendError(c, errors.AccountNotFound)
		}
		if err == services.ErrCustomerNotFound {
			return SendError(c, errors.CustomerNotFound)
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Ownership transferred successfully",
	})
}

// GetCustomerActivity retrieves customer activity logs
// @Summary Get customer activity (admin)
// @Description Admin endpoint to retrieve activity logs for a specific customer
// @Tags Customers
// @Security BearerAuth
// @Produce json
// @Param id path string true "Customer ID (UUID)"
// @Param limit query int false "Number of results" default(50)
// @Param offset query int false "Pagination offset" default(0)
// @Success 200 {object} object{activities=[]models.AuditLog,total=int,limit=int,offset=int} "Customer activity logs"
// @Failure 400 {object} errors.ErrorResponse "CUSTOMER_004 - Invalid customer ID"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /customers/{id}/activity [get]
func (h *CustomerHandler) GetCustomerActivity(c echo.Context) error {
	customerIDStr := c.Param("id")
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		return SendError(c, errors.CustomerInvalidID)
	}

	// Parse query parameters
	limit := 50 // default
	offset := 0 // default
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetParam := c.QueryParam("offset"); offsetParam != "" {
		if o, err := strconv.Atoi(offsetParam); err == nil && o >= 0 {
			offset = o
		}
	}

	activities, total, err := h.auditService.GetCustomerActivity(customerID, nil, nil, limit, offset)
	if err != nil {
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"activities": activities,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

// GetMyActivity retrieves the authenticated customer's activity
// @Summary Get my activity
// @Description Retrieve activity logs for the authenticated customer
// @Tags Customers
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Number of results" default(50)
// @Param offset query int false "Pagination offset" default(0)
// @Success 200 {object} object{activities=[]models.AuditLog,total=int,limit=int,offset=int} "Customer activity logs"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /customers/me/activity [get]
func (h *CustomerHandler) GetMyActivity(c echo.Context) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	// Parse query parameters
	limit := 50 // default
	offset := 0 // default
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetParam := c.QueryParam("offset"); offsetParam != "" {
		if o, err := strconv.Atoi(offsetParam); err == nil && o >= 0 {
			offset = o
		}
	}

	activities, total, err := h.auditService.GetCustomerActivity(userID, nil, nil, limit, offset)
	if err != nil {
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"activities": activities,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

// ResetCustomerPassword resets a customer's password (admin only)
// @Summary Reset customer password (admin)
// @Description Admin endpoint to reset a customer's password and generate a temporary password
// @Tags Customers
// @Security BearerAuth
// @Produce json
// @Param id path string true "Customer ID (UUID)"
// @Success 200 {object} object{temporary_password=string,message=string} "Password reset successfully with temporary password"
// @Failure 400 {object} errors.ErrorResponse "CUSTOMER_004 - Invalid customer ID"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 404 {object} errors.ErrorResponse "CUSTOMER_001 - Customer not found"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /customers/{id}/password/reset [put]
func (h *CustomerHandler) ResetCustomerPassword(c echo.Context) error {
	customerIDStr := c.Param("id")
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		return SendError(c, errors.CustomerInvalidID)
	}

	adminID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	tempPassword, err := h.passwordService.AdminResetPassword(customerID, adminID)
	if err != nil {
		if err == services.ErrCustomerNotFound {
			return SendError(c, errors.CustomerNotFound)
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"temporary_password": tempPassword,
		"message":            "Password reset successfully",
	})
}

// UpdateMyPassword updates the authenticated customer's password
// @Summary Update my password
// @Description Update the authenticated customer's password (requires current password)
// @Tags Customers
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object{current_password=string,new_password=string} true "Password update details"
// @Success 200 {object} SuccessResponse{message=string} "Password updated successfully"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_001 - Invalid request body"
// @Failure 401 {object} errors.ErrorResponse "AUTH_001 - Current password is incorrect or AUTH_002 - Missing authentication"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /customers/me/password [put]
func (h *CustomerHandler) UpdateMyPassword(c echo.Context) error {
	var req struct {
		CurrentPassword string `json:"current_password" validate:"required"`
		NewPassword     string `json:"new_password" validate:"required,min=12"`
	}
	if err := c.Bind(&req); err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("Invalid request body"))
	}

	if err := c.Validate(req); err != nil {
		return SendError(c, errors.ValidationInvalidFormat, errors.WithDetails(err.Error()))
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		return SendError(c, errors.AuthMissingToken)
	}

	err = h.passwordService.CustomerUpdatePassword(userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		if err == services.ErrCurrentPasswordWrong {
			return SendError(c, errors.AuthInvalidCredentials)
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Password updated successfully",
	})
}
