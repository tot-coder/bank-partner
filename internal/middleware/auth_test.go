package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"array-assessment/internal/config"
	"array-assessment/internal/models"
	"array-assessment/internal/repositories/repository_mocks"
	"array-assessment/internal/services"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
)

func TestAuthMiddleware(t *testing.T) {
	suite.Run(t, new(AuthMiddlewareSuite))
}

type AuthMiddlewareSuite struct {
	suite.Suite
	ctrl                     *gomock.Controller
	tokenService             services.TokenServiceInterface
	mockBlacklistedTokenRepo *repository_mocks.MockBlacklistedTokenRepositoryInterface
	e                        *echo.Echo
}

func (s *AuthMiddlewareSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	privateKey, publicKey, err := config.GenerateRSAKeyPair()
	s.NoError(err)

	jwtConfig := &config.JWTConfig{
		PrivateKey:           privateKey,
		PublicKey:            publicKey,
		Issuer:               "test-issuer",
		AccessTokenDuration:  24 * time.Hour,
		RefreshTokenDuration: 7 * 24 * time.Hour,
	}

	s.tokenService = services.NewTokenService(jwtConfig)
	s.mockBlacklistedTokenRepo = repository_mocks.NewMockBlacklistedTokenRepositoryInterface(s.ctrl)
	s.e = echo.New()
}

// TearDownTest runs after each test in the suite
func (s *AuthMiddlewareSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *AuthMiddlewareSuite) createTokenService() services.TokenServiceInterface {
	privateKey, publicKey, err := config.GenerateRSAKeyPair()
	s.NoError(err)

	jwtConfig := &config.JWTConfig{
		PrivateKey:           privateKey,
		PublicKey:            publicKey,
		Issuer:               "test-issuer",
		AccessTokenDuration:  24 * time.Hour,
		RefreshTokenDuration: 7 * 24 * time.Hour,
	}

	return services.NewTokenService(jwtConfig)
}

func (s *AuthMiddlewareSuite) TestRequireAuth_ValidToken() {
	middleware := RequireAuth(s.tokenService, s.mockBlacklistedTokenRepo)

	// Create a test user and generate a valid token
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleCustomer,
	}

	s.mockBlacklistedTokenRepo.EXPECT().GetByJTI(gomock.Any()).Return(nil, nil)

	token, _, err := s.tokenService.GenerateAccessToken(user)
	s.NoError(err)

	// Create a test handler that checks context values
	handler := middleware(func(c echo.Context) error {
		// Verify context values are set correctly
		ctxUserID := c.Get("user_id")
		ctxEmail := c.Get("user_email")
		ctxRole := c.Get("user_role")

		s.Equal(user.ID, ctxUserID)
		s.Equal(user.Email, ctxEmail)
		s.Equal(user.Role, ctxRole)

		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Create request with valid token
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	err = handler(c)
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *AuthMiddlewareSuite) TestRequireAuth_MissingAuthorizationHeader() {
	middleware := RequireAuth(s.tokenService, s.mockBlacklistedTokenRepo)

	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	// No Authorization header
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	err := handler(c)
	// Auth middleware uses SendError which sends response and returns nil
	s.NoError(err)
	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *AuthMiddlewareSuite) TestRequireAuth_InvalidTokenFormat() {
	middleware := RequireAuth(s.tokenService, s.mockBlacklistedTokenRepo)

	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "InvalidToken")
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	err := handler(c)
	s.NoError(err)
	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *AuthMiddlewareSuite) TestRequireAuth_MalformedJWT() {
	middleware := RequireAuth(s.tokenService, s.mockBlacklistedTokenRepo)

	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	err := handler(c)
	s.NoError(err)
	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *AuthMiddlewareSuite) TestRequireAuth_ExpiredToken() {
	// Create a token service with very short expiration
	privateKey, publicKey, err := config.GenerateRSAKeyPair()
	s.NoError(err)

	jwtConfig := &config.JWTConfig{
		PrivateKey:           privateKey,
		PublicKey:            publicKey,
		Issuer:               "test-issuer",
		AccessTokenDuration:  1 * time.Millisecond,
		RefreshTokenDuration: 1 * time.Hour,
	}

	shortTokenService := services.NewTokenService(jwtConfig)
	shortMiddleware := RequireAuth(shortTokenService, s.mockBlacklistedTokenRepo)

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleCustomer,
	}

	token, _, err := shortTokenService.GenerateAccessToken(user)
	s.NoError(err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	handler := shortMiddleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	err = handler(c)
	s.NoError(err)
	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *AuthMiddlewareSuite) TestRequireAuth_TokenSignedWithDifferentKey() {
	// Create two different token services with different keys
	tokenService1 := s.createTokenService()
	tokenService2 := s.createTokenService()

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleCustomer,
	}

	// Generate token with first service
	token, _, err := tokenService1.GenerateAccessToken(user)
	s.NoError(err)

	// Try to validate with second service
	middleware2 := RequireAuth(tokenService2, s.mockBlacklistedTokenRepo)
	handler := middleware2(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	err = handler(c)
	s.NoError(err)
	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *AuthMiddlewareSuite) TestRequireRole_AuthorizedWithCorrectRole() {
	middleware := RequireRole(models.RoleAdmin)

	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	// Set admin role in context
	c.Set("user_role", models.RoleAdmin)

	err := handler(c)
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *AuthMiddlewareSuite) TestRequireRole_UnauthorizedWithWrongRole() {
	middleware := RequireRole(models.RoleAdmin)

	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	// Set customer role in context
	c.Set("user_role", models.RoleCustomer)

	err := handler(c)
	s.NoError(err)
	s.Equal(http.StatusForbidden, rec.Code)
}

func (s *AuthMiddlewareSuite) TestRequireRole_MissingRoleInContext() {
	middleware := RequireRole(models.RoleAdmin)

	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)

	// No role set in context

	err := handler(c)
	s.NoError(err)
	s.Equal(http.StatusUnauthorized, rec.Code) // Returns 401 when role is missing from context
}

func (s *AuthMiddlewareSuite) TestRequireRole_AllowsMultipleRoles() {
	middleware := RequireRole(models.RoleAdmin, models.RoleCustomer)

	// Test with admin role
	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/mixed", nil)
	rec := httptest.NewRecorder()
	c := s.e.NewContext(req, rec)
	c.Set("user_role", models.RoleAdmin)

	err := handler(c)
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)

	// Test with customer role
	req = httptest.NewRequest(http.MethodGet, "/mixed", nil)
	rec = httptest.NewRecorder()
	c = s.e.NewContext(req, rec)
	c.Set("user_role", models.RoleCustomer)

	err = handler(c)
	s.NoError(err)
	s.Equal(http.StatusOK, rec.Code)
}
