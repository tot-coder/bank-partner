package handlers

import (
	"net/http"
	"strings"

	"array-assessment/internal/dto"
	"array-assessment/internal/errors"
	"array-assessment/internal/services"

	"github.com/labstack/echo/v4"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService services.AuthServiceInterface
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService services.AuthServiceInterface) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register handles user registration
// @Summary Register a new user
// @Description Create a new user account with email, password, and personal information
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Registration details"
// @Success 201 {object} SuccessResponse{data=object{id=string,email=string,first_name=string,last_name=string,role=string,created_at=string}} "User created successfully"
// @Failure 400 {object} errors.ErrorResponse "Validation error - AUTH_001 (Invalid request body)"
// @Failure 409 {object} errors.ErrorResponse "Customer already exists - CUSTOMER_001"
// @Failure 500 {object} errors.ErrorResponse "System error - SYSTEM_001 or SYSTEM_002"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c echo.Context) error {
	var req dto.RegisterRequest

	if err := c.Bind(&req); err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("Invalid request body"))
	}

	if err := c.Validate(req); err != nil {
		return err
	}

	ipAddress := getClientIP(c)
	userAgent := c.Request().UserAgent()

	user, err := h.authService.Register(&req, ipAddress, userAgent)
	if err != nil {
		if err == services.ErrUserAlreadyExists {
			return SendError(c, errors.CustomerAlreadyExists)
		}
		return SendSystemError(c, err)
	}

	response := map[string]interface{}{
		"id":         user.ID,
		"email":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"role":       user.Role,
		"created_at": user.CreatedAt,
	}

	return c.JSON(http.StatusCreated, SuccessResponse{
		Data:    response,
		Message: "User registered successfully",
	})
}

// Login handles user authentication
// @Summary Login user
// @Description Authenticate user with email and password, receive JWT access and refresh tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login credentials"
// @Success 200 {object} dto.TokenResponse "Login successful with JWT tokens"
// @Failure 400 {object} errors.ErrorResponse "Validation error - AUTH_001"
// @Failure 401 {object} errors.ErrorResponse "Invalid credentials - AUTH_002"
// @Failure 403 {object} errors.ErrorResponse "Account locked - AUTH_006"
// @Failure 500 {object} errors.ErrorResponse "System error - SYSTEM_001 or SYSTEM_002"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req dto.LoginRequest

	if err := c.Bind(&req); err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("Invalid request body"))
	}

	if err := c.Validate(req); err != nil {
		return err
	}

	ipAddress := getClientIP(c)
	userAgent := c.Request().UserAgent()

	tokens, err := h.authService.Login(&req, ipAddress, userAgent)
	if err != nil {
		if err == services.ErrAccountLocked {
			return SendError(c, errors.AuthAccountLocked)
		}
		if err == services.ErrInvalidCredentials {
			return SendError(c, errors.AuthInvalidCredentials)
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, tokens)
}

// RefreshToken handles token refresh
// @Summary Refresh access token
// @Description Get a new access token and refresh token pair using a valid refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body object{refresh_token=string} true "Refresh token"
// @Success 200 {object} dto.TokenResponse "Token refreshed successfully"
// @Failure 400 {object} errors.ErrorResponse "Validation error - AUTH_001"
// @Failure 401 {object} errors.ErrorResponse "Invalid refresh token - AUTH_003"
// @Failure 500 {object} errors.ErrorResponse "System error - SYSTEM_001 or SYSTEM_002"
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	if err := c.Bind(&req); err != nil {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("Invalid request body"))
	}

	if err := c.Validate(req); err != nil {
		return err
	}

	ipAddress := getClientIP(c)
	userAgent := c.Request().UserAgent()

	tokens, err := h.authService.RefreshTokens(req.RefreshToken, ipAddress, userAgent)
	if err != nil {
		if err == services.ErrInvalidRefreshToken {
			return SendError(c, errors.AuthInvalidTokenFormat, errors.WithDetails("Invalid or expired refresh token"))
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, tokens)
}

// Logout handles user logout
// @Summary Logout user
// @Description Invalidate user's access token and refresh token. Requires Bearer token in Authorization header.
// @Tags Authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} SuccessResponse{message=string} "Logout successful"
// @Failure 401 {object} errors.ErrorResponse "Unauthorized - AUTH_004 or AUTH_005"
// @Failure 500 {object} errors.ErrorResponse "System error - SYSTEM_001"
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c echo.Context) error {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return SendError(c, errors.AuthMissingToken)
	}

	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || strings.ToLower(tokenParts[0]) != "bearer" {
		return SendError(c, errors.AuthInvalidTokenFormat)
	}

	accessToken := tokenParts[1]
	ipAddress := getClientIP(c)
	userAgent := c.Request().UserAgent()

	if err := h.authService.Logout(accessToken, ipAddress, userAgent); err != nil {
		// Security: Always return success to prevent information leakage about system internals
	}

	return c.JSON(http.StatusOK, SuccessResponse{
		Message: "Logout successful",
	})
}
