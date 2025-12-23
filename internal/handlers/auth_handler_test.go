package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestAuthHandler(t *testing.T) {
	suite.Run(t, new(AuthHandlerSuite))
}

type AuthHandlerSuite struct {
	suite.Suite
	ctrl        *gomock.Controller
	authService *service_mocks.MockAuthServiceInterface
	handler     *AuthHandler
	e           *echo.Echo
}

func (s *AuthHandlerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.authService = service_mocks.NewMockAuthServiceInterface(s.ctrl)
	s.handler = NewAuthHandler(s.authService)
	s.e = echo.New()
	s.e.Validator = &CustomValidator{validator: validator.New()}
}

func (s *AuthHandlerSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *AuthHandlerSuite) TestRegister() {
	s.Run("successful registration", func() {
		// Recreate mocks for this specific test
		ctrl := gomock.NewController(s.T())
		defer ctrl.Finish()
		s.authService = service_mocks.NewMockAuthServiceInterface(ctrl)
		s.handler = NewAuthHandler(s.authService)

		// Use camelCase JSON field names to match the DTO
		reqBody := map[string]string{
			"email":     "test@example.com",
			"password":  "SecurePassword123!",
			"firstName": "John",
			"lastName":  "Doe",
		}
		body, _ := json.Marshal(reqBody)

		expectedUser := &models.User{
			ID:        uuid.New(),
			Email:     "test@example.com",
			FirstName: "John",
			LastName:  "Doe",
			Role:      models.RoleCustomer,
			CreatedAt: time.Now(),
		}

		// Setup mock expectations
		s.authService.EXPECT().
			Register(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(expectedUser, nil).
			Times(1)

		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		err := s.handler.Register(c)
		s.NoError(err)
		s.Equal(http.StatusCreated, rec.Code)

		var response SuccessResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &response)
		s.NotNil(response.Data)
	})

	s.Run("duplicate email", func() {
		// Recreate mocks for this specific test
		ctrl := gomock.NewController(s.T())
		defer ctrl.Finish()
		s.authService = service_mocks.NewMockAuthServiceInterface(ctrl)
		s.handler = NewAuthHandler(s.authService)

		reqBody := map[string]string{
			"email":     "duplicate@example.com",
			"password":  "SecurePassword123!",
			"firstName": "Jane",
			"lastName":  "Smith",
		}
		body, _ := json.Marshal(reqBody)

		// Setup mock expectations - return duplicate user error
		s.authService.EXPECT().
			Register(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, services.ErrUserAlreadyExists).
			Times(1)

		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		err := s.handler.Register(c)
		s.NoError(err)
		s.Equal(http.StatusUnprocessableEntity, rec.Code) // CUSTOMER_002 maps to 422

		// Parse and verify error response
		var errorResp ErrorResponse
		err = json.Unmarshal(rec.Body.Bytes(), &errorResp)
		s.NoError(err)
		s.Equal("CUSTOMER_002", errorResp.Error.Code)
	})

	s.Run("invalid request body", func() {
		// Recreate mocks for this specific test
		ctrl := gomock.NewController(s.T())
		defer ctrl.Finish()
		s.authService = service_mocks.NewMockAuthServiceInterface(ctrl)
		s.handler = NewAuthHandler(s.authService)

		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		err := s.handler.Register(c)
		s.NoError(err)
		s.Equal(http.StatusBadRequest, rec.Code)

		// Parse and verify error response
		var errorResp ErrorResponse
		err = json.Unmarshal(rec.Body.Bytes(), &errorResp)
		s.NoError(err)
		s.Equal("VALIDATION_001", errorResp.Error.Code)
	})

	s.Run("missing required fields", func() {
		// Recreate mocks for this specific test
		ctrl := gomock.NewController(s.T())
		defer ctrl.Finish()
		s.authService = service_mocks.NewMockAuthServiceInterface(ctrl)
		s.handler = NewAuthHandler(s.authService)

		reqBody := map[string]string{
			"email": "test@example.com",
			// Missing password and other required fields
		}
		body, _ := json.Marshal(reqBody)

		// No mock expectation - validation should fail before service is called

		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		// Set up validator
		s.e.Validator = &CustomValidator{validator: validator.New()}

		err := s.handler.Register(c)
		// Validation error should be returned
		s.Error(err)
	})
}

func (s *AuthHandlerSuite) TestLogin() {
	s.Run("successful login", func() {
		// Recreate mocks for this specific test
		ctrl := gomock.NewController(s.T())
		defer ctrl.Finish()
		s.authService = service_mocks.NewMockAuthServiceInterface(ctrl)
		s.handler = NewAuthHandler(s.authService)

		email := "login@example.com"
		password := "SecurePassword123!"

		loginBody := map[string]string{
			"email":    email,
			"password": password,
		}
		body, _ := json.Marshal(loginBody)

		expectedTokens := &dto.TokenResponse{
			AccessToken:  "access.token.here",
			RefreshToken: "refresh.token.here",
			TokenType:    "Bearer",
			ExpiresAt:    time.Now().Add(time.Hour),
		}

		// Setup mock expectations
		s.authService.EXPECT().
			Login(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(req *dto.LoginRequest, ipAddress, userAgent string) (*dto.TokenResponse, error) {
				s.Equal(email, req.Email)
				s.Equal(password, req.Password)
				return expectedTokens, nil
			}).
			Times(1)

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		err := s.handler.Login(c)
		s.NoError(err)
		s.Equal(http.StatusOK, rec.Code)

		var response map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &response)
		s.NotEmpty(response["accessToken"])
		s.NotEmpty(response["refreshToken"])
		s.Equal("Bearer", response["tokenType"])
	})

	s.Run("invalid password", func() {
		// Recreate mocks for this specific test
		ctrl := gomock.NewController(s.T())
		defer ctrl.Finish()
		s.authService = service_mocks.NewMockAuthServiceInterface(ctrl)
		s.handler = NewAuthHandler(s.authService)

		email := "login@example.com"

		loginBody := map[string]string{
			"email":    email,
			"password": "WrongPassword",
		}
		body, _ := json.Marshal(loginBody)

		// Setup mock expectations - return invalid credentials error
		s.authService.EXPECT().
			Login(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, services.ErrInvalidCredentials).
			Times(1)

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		err := s.handler.Login(c)
		s.NoError(err) // Handler returns nil on success, writes JSON to response
		s.Equal(http.StatusUnauthorized, rec.Code)

		// Parse and verify error response
		var errorResp ErrorResponse
		err = json.Unmarshal(rec.Body.Bytes(), &errorResp)
		s.NoError(err)
		s.Equal("AUTH_001", errorResp.Error.Code)
	})

	s.Run("non-existent user", func() {
		// Recreate mocks for this specific test
		ctrl := gomock.NewController(s.T())
		defer ctrl.Finish()
		s.authService = service_mocks.NewMockAuthServiceInterface(ctrl)
		s.handler = NewAuthHandler(s.authService)

		loginBody := map[string]string{
			"email":    "nonexistent@example.com",
			"password": "SomePassword123!",
		}
		body, _ := json.Marshal(loginBody)

		// Setup mock expectations - return invalid credentials error
		s.authService.EXPECT().
			Login(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, services.ErrInvalidCredentials).
			Times(1)

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		err := s.handler.Login(c)
		s.NoError(err)
		s.Equal(http.StatusUnauthorized, rec.Code)

		// Parse and verify error response
		var errorResp ErrorResponse
		err = json.Unmarshal(rec.Body.Bytes(), &errorResp)
		s.NoError(err)
		s.Equal("AUTH_001", errorResp.Error.Code)
	})

	s.Run("account locked", func() {
		// Recreate mocks for this specific test
		ctrl := gomock.NewController(s.T())
		defer ctrl.Finish()
		s.authService = service_mocks.NewMockAuthServiceInterface(ctrl)
		s.handler = NewAuthHandler(s.authService)

		loginBody := map[string]string{
			"email":    "locked@example.com",
			"password": "SomePassword123!",
		}
		body, _ := json.Marshal(loginBody)

		// Setup mock expectations - return account locked error
		s.authService.EXPECT().
			Login(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, services.ErrAccountLocked).
			Times(1)

		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		err := s.handler.Login(c)
		s.NoError(err)
		s.Equal(http.StatusForbidden, rec.Code) // AUTH_006 maps to 403

		// Parse and verify error response
		var errorResp ErrorResponse
		err = json.Unmarshal(rec.Body.Bytes(), &errorResp)
		s.NoError(err)
		s.Equal("AUTH_006", errorResp.Error.Code)
	})
}

func (s *AuthHandlerSuite) TestRefreshToken() {
	s.Run("successful refresh", func() {
		// Recreate mocks for this specific test
		ctrl := gomock.NewController(s.T())
		defer ctrl.Finish()
		s.authService = service_mocks.NewMockAuthServiceInterface(ctrl)
		s.handler = NewAuthHandler(s.authService)

		refreshToken := "valid.refresh.token"

		refreshBody := map[string]string{
			"refresh_token": refreshToken,
		}
		body, _ := json.Marshal(refreshBody)

		expectedTokens := &dto.TokenResponse{
			AccessToken:  "new.access.token",
			RefreshToken: "new.refresh.token",
			TokenType:    "Bearer",
			ExpiresAt:    time.Now().Add(time.Hour),
		}

		// Setup mock expectations
		s.authService.EXPECT().
			RefreshTokens(refreshToken, gomock.Any(), gomock.Any()).
			Return(expectedTokens, nil).
			Times(1)

		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		err := s.handler.RefreshToken(c)
		s.NoError(err)
		s.Equal(http.StatusOK, rec.Code)

		var response map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &response)

		s.NotEmpty(response["accessToken"])
		s.NotEmpty(response["refreshToken"])
	})

	s.Run("invalid refresh token", func() {
		// Recreate mocks for this specific test
		ctrl := gomock.NewController(s.T())
		defer ctrl.Finish()
		s.authService = service_mocks.NewMockAuthServiceInterface(ctrl)
		s.handler = NewAuthHandler(s.authService)

		refreshBody := map[string]string{
			"refresh_token": "invalid.token.here",
		}
		body, _ := json.Marshal(refreshBody)

		// Setup mock expectations - return invalid refresh token error
		s.authService.EXPECT().
			RefreshTokens(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, services.ErrInvalidRefreshToken).
			Times(1)

		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		err := s.handler.RefreshToken(c)
		s.NoError(err)
		s.Equal(http.StatusUnauthorized, rec.Code)
	})

	s.Run("missing refresh token", func() {
		// Recreate mocks for this specific test
		ctrl := gomock.NewController(s.T())
		defer ctrl.Finish()
		s.authService = service_mocks.NewMockAuthServiceInterface(ctrl)
		s.handler = NewAuthHandler(s.authService)

		// Use camelCase JSON field name
		refreshBody := map[string]string{}
		body, _ := json.Marshal(refreshBody)

		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		err := s.handler.RefreshToken(c)
		// Should fail with error from service
		s.Error(err)
	})
}

func (s *AuthHandlerSuite) TestLogout() {
	s.Run("successful logout", func() {
		// Recreate mocks for this specific test
		ctrl := gomock.NewController(s.T())
		defer ctrl.Finish()
		s.authService = service_mocks.NewMockAuthServiceInterface(ctrl)
		s.handler = NewAuthHandler(s.authService)

		accessToken := "valid.access.token"

		// Setup mock expectations
		s.authService.EXPECT().
			Logout(accessToken, gomock.Any(), gomock.Any()).
			Return(nil).
			Times(1)

		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		err := s.handler.Logout(c)
		s.NoError(err)
		s.Equal(http.StatusOK, rec.Code)

		var response SuccessResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &response)
		s.Equal("Logout successful", response.Message)
	})

	s.Run("logout without token", func() {
		// Recreate mocks for this specific test
		ctrl := gomock.NewController(s.T())
		defer ctrl.Finish()
		s.authService = service_mocks.NewMockAuthServiceInterface(ctrl)
		s.handler = NewAuthHandler(s.authService)

		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		// Should return 401 Unauthorized when no token is provided
		// No mock expectation needed - validation happens before service call
		err := s.handler.Logout(c)
		s.NoError(err)
		s.Equal(http.StatusUnauthorized, rec.Code)
	})

	s.Run("logout with invalid token format", func() {
		// Recreate mocks for this specific test
		ctrl := gomock.NewController(s.T())
		defer ctrl.Finish()
		s.authService = service_mocks.NewMockAuthServiceInterface(ctrl)
		s.handler = NewAuthHandler(s.authService)

		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		// Should return 401 Unauthorized for invalid format
		// No mock expectation needed - validation happens before service call
		err := s.handler.Logout(c)
		s.NoError(err)
		s.Equal(http.StatusUnauthorized, rec.Code)
	})

	s.Run("logout with service error still returns success", func() {
		// Recreate mocks for this specific test
		ctrl := gomock.NewController(s.T())
		defer ctrl.Finish()
		s.authService = service_mocks.NewMockAuthServiceInterface(ctrl)
		s.handler = NewAuthHandler(s.authService)

		accessToken := "token.with.error"

		// Setup mock expectations - service returns error but handler still returns success
		// Security: Always return success to prevent information leakage
		s.authService.EXPECT().
			Logout(accessToken, gomock.Any(), gomock.Any()).
			Return(services.ErrInvalidToken).
			Times(1)

		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		rec := httptest.NewRecorder()
		c := s.e.NewContext(req, rec)

		err := s.handler.Logout(c)
		s.NoError(err) // Handler should still return success
		s.Equal(http.StatusOK, rec.Code)

		var response SuccessResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &response)
		s.Equal("Logout successful", response.Message)
	})
}
