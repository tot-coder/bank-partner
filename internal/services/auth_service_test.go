package services

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"array-assessment/internal/dto"
	"array-assessment/internal/models"
	"array-assessment/internal/repositories"
	"array-assessment/internal/repositories/repository_mocks"
	"array-assessment/internal/services/service_mocks"

	"github.com/golang-jwt/jwt/v5"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type AuthServiceTestSuite struct {
	suite.Suite
	ctrl                 *gomock.Controller
	userRepo             *repository_mocks.MockUserRepositoryInterface
	refreshTokenRepo     *repository_mocks.MockRefreshTokenRepositoryInterface
	auditRepo            *repository_mocks.MockAuditLogRepositoryInterface
	blacklistedTokenRepo *repository_mocks.MockBlacklistedTokenRepositoryInterface
	passwordService      *service_mocks.MockPasswordServiceInterface
	tokenService         *service_mocks.MockTokenServiceInterface
	accountService       *service_mocks.MockAccountServiceInterface
	authService          AuthServiceInterface
}

func (s *AuthServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.userRepo = repository_mocks.NewMockUserRepositoryInterface(s.ctrl)
	s.tokenService = service_mocks.NewMockTokenServiceInterface(s.ctrl)
	s.accountService = service_mocks.NewMockAccountServiceInterface(s.ctrl)
	s.refreshTokenRepo = repository_mocks.NewMockRefreshTokenRepositoryInterface(s.ctrl)
	s.auditRepo = repository_mocks.NewMockAuditLogRepositoryInterface(s.ctrl)
	s.blacklistedTokenRepo = repository_mocks.NewMockBlacklistedTokenRepositoryInterface(s.ctrl)
	s.passwordService = service_mocks.NewMockPasswordServiceInterface(s.ctrl)
	s.authService = NewAuthService(s.userRepo, s.refreshTokenRepo, s.auditRepo, s.blacklistedTokenRepo, s.passwordService, s.tokenService, s.accountService, slog.Default())
}

func (s *AuthServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestAuthServiceSuite(t *testing.T) {
	suite.Run(t, new(AuthServiceTestSuite))
}

func (s *AuthServiceTestSuite) TestRegister_SuccessfulRegistration() {
	req := &dto.RegisterRequest{
		Email:     "new@example.com",
		Password:  "SecurePass123!",
		FirstName: "John",
		LastName:  "Doe",
	}

	// Setup mock expectations
	s.userRepo.EXPECT().GetByEmail(req.Email).Return(nil, repositories.ErrUserNotFound).Times(1)
	s.passwordService.EXPECT().HashPassword(req.Password).Return("hashed_password", nil).Times(1)
	s.userRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)
	s.accountService.EXPECT().CreateAccountsForNewUser(gomock.Any()).Return(nil).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(2) // account creation + registration audit logs

	user, err := s.authService.Register(req, "192.168.1.1", "Mozilla/5.0")

	s.NoError(err)
	s.NotNil(user)
	s.Equal(req.Email, user.Email)
	s.Equal(req.FirstName, user.FirstName)
	s.Equal(req.LastName, user.LastName)
	s.Equal(models.RoleCustomer, user.Role)
	s.NotEmpty(user.PasswordHash)
	s.NotEqual(req.Password, user.PasswordHash) // Ensure password is hashed
}

func (s *AuthServiceTestSuite) TestRegister_UserAlreadyExists() {
	req := &dto.RegisterRequest{
		Email:     "existing@example.com",
		Password:  "SecurePass123!",
		FirstName: "Jane",
		LastName:  "Smith",
	}

	existingUser := &models.User{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	// Setup mock expectations - user already exists
	s.userRepo.EXPECT().GetByEmail(req.Email).Return(existingUser, nil).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1) // failed registration audit log

	// Attempt registration with existing email
	user, err := s.authService.Register(req, "192.168.1.1", "Mozilla/5.0")
	s.Equal(ErrUserAlreadyExists, err)
	s.Nil(user)
}

func (s *AuthServiceTestSuite) TestRegister_WeakPasswordValidation() {
	req := &dto.RegisterRequest{
		Email:     "weak@example.com",
		Password:  "123", // Weak password
		FirstName: "Weak",
		LastName:  "User",
	}

	// Setup mock expectations
	s.userRepo.EXPECT().GetByEmail(req.Email).Return(nil, repositories.ErrUserNotFound).Times(1)
	s.passwordService.EXPECT().HashPassword(req.Password).Return("", errors.New("password must be at least 12 characters")).Times(1)

	// The password service enforces minimum length requirement
	user, err := s.authService.Register(req, "192.168.1.1", "Mozilla/5.0")
	s.Error(err)
	s.Contains(err.Error(), "password must be at least 12 characters")
	s.Nil(user)
}

func (s *AuthServiceTestSuite) TestLogin_SuccessfulLogin() {
	email := "test@example.com"
	password := "SecurePass123!@#"
	userID := uuid.New()

	user := &models.User{
		ID:                  userID,
		Email:               email,
		PasswordHash:        "hashed_password",
		FirstName:           "Test",
		LastName:            "User",
		Role:                models.RoleCustomer,
		FailedLoginAttempts: 0,
		LockedAt:            nil,
	}

	req := &dto.LoginRequest{
		Email:    email,
		Password: password,
	}

	expiresAt := time.Now().Add(15 * time.Minute)

	// Setup mock expectations
	s.userRepo.EXPECT().GetByEmail(email).Return(user, nil).Times(1)
	s.passwordService.EXPECT().ComparePassword(password, user.PasswordHash).Return(true).Times(1)
	s.userRepo.EXPECT().UpdateFailedLoginAttempts(gomock.Any()).Return(nil).Times(1)
	s.tokenService.EXPECT().GenerateAccessToken(user).Return("access_token", expiresAt, nil).Times(1)
	s.tokenService.EXPECT().GenerateRefreshToken(userID).Return("refresh_token", time.Now().Add(7*24*time.Hour), nil).Times(1)
	s.refreshTokenRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1) // successful login audit log

	tokens, err := s.authService.Login(req, "192.168.1.1", "Mozilla/5.0")

	s.NoError(err)
	s.NotNil(tokens)
	s.NotEmpty(tokens.AccessToken)
	s.NotEmpty(tokens.RefreshToken)
	s.Equal("Bearer", tokens.TokenType)
	s.True(tokens.ExpiresAt.After(time.Now()))
}

func (s *AuthServiceTestSuite) TestLogin_InvalidPassword() {
	email := "test2@example.com"
	userID := uuid.New()

	user := &models.User{
		ID:                  userID,
		Email:               email,
		PasswordHash:        "hashed_password",
		FirstName:           "Test",
		LastName:            "User",
		Role:                models.RoleCustomer,
		FailedLoginAttempts: 0,
		LockedAt:            nil,
	}

	req := &dto.LoginRequest{
		Email:    email,
		Password: "WrongPassword",
	}

	// Setup mock expectations
	s.userRepo.EXPECT().GetByEmail(email).Return(user, nil).Times(1)
	s.passwordService.EXPECT().ComparePassword("WrongPassword", user.PasswordHash).Return(false).Times(1)
	s.userRepo.EXPECT().UpdateFailedLoginAttempts(gomock.Any()).Return(nil).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1) // failed login audit log

	tokens, err := s.authService.Login(req, "192.168.1.1", "Mozilla/5.0")

	s.Equal(ErrInvalidCredentials, err)
	s.Nil(tokens)
}

func (s *AuthServiceTestSuite) TestLogin_NonExistentUser() {
	req := &dto.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "SomePassword",
	}

	// Setup mock expectations
	s.userRepo.EXPECT().GetByEmail(req.Email).Return(nil, repositories.ErrUserNotFound).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1) // failed login audit log

	tokens, err := s.authService.Login(req, "192.168.1.1", "Mozilla/5.0")

	s.Equal(ErrInvalidCredentials, err)
	s.Nil(tokens)
}

func (s *AuthServiceTestSuite) TestLogin_AccountLockoutAfterFailedAttempts() {
	lockoutEmail := "lockout@example.com"
	lockoutPassword := "CorrectPass123!"
	userID := uuid.New()

	// User starts with 2 failed attempts (will be locked on 3rd attempt)
	user := &models.User{
		ID:                  userID,
		Email:               lockoutEmail,
		PasswordHash:        "hashed_password",
		FirstName:           "Lock",
		LastName:            "Out",
		Role:                models.RoleCustomer,
		FailedLoginAttempts: 2,
		LockedAt:            nil,
	}

	wrongReq := &dto.LoginRequest{
		Email:    lockoutEmail,
		Password: "WrongPassword",
	}

	// Third failed attempt - this should lock the account
	s.userRepo.EXPECT().GetByEmail(lockoutEmail).Return(user, nil).Times(1)
	s.passwordService.EXPECT().ComparePassword("WrongPassword", user.PasswordHash).Return(false).Times(1)
	s.userRepo.EXPECT().UpdateFailedLoginAttempts(gomock.Any()).Return(nil).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(2) // account locked + failed login audit logs

	_, err := s.authService.Login(wrongReq, "192.168.1.1", "Mozilla/5.0")
	s.Equal(ErrInvalidCredentials, err)

	// Now try with correct password - should be locked
	lockedTime := time.Now().Add(30 * time.Minute)
	lockedUser := &models.User{
		ID:                  userID,
		Email:               lockoutEmail,
		PasswordHash:        "hashed_password",
		FirstName:           "Lock",
		LastName:            "Out",
		Role:                models.RoleCustomer,
		FailedLoginAttempts: 3,
		LockedAt:            &lockedTime,
	}

	correctReq := &dto.LoginRequest{
		Email:    lockoutEmail,
		Password: lockoutPassword,
	}

	s.userRepo.EXPECT().GetByEmail(lockoutEmail).Return(lockedUser, nil).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1) // account locked audit log

	tokens, err := s.authService.Login(correctReq, "192.168.1.1", "Mozilla/5.0")
	s.Equal(ErrAccountLocked, err)
	s.Nil(tokens)
}
func (s *AuthServiceTestSuite) TestRefreshTokens_SuccessfulTokenRefresh() {
	userID := uuid.New()
	refreshToken := "valid_refresh_token"
	tokenHash := "hashed_token"

	user := &models.User{
		ID:        userID,
		Email:     "refresh@example.com",
		FirstName: "Refresh",
		LastName:  "User",
		Role:      models.RoleCustomer,
	}

	storedToken := &models.RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		RevokedAt: nil,
	}

	claims := &models.CustomClaims{
		UserID: userID.String(),
	}

	expiresAt := time.Now().Add(15 * time.Minute)

	// Setup mock expectations
	s.tokenService.EXPECT().ValidateRefreshToken(refreshToken).Return(claims, nil).Times(1)
	s.refreshTokenRepo.EXPECT().GetByTokenHash(gomock.Any()).Return(storedToken, nil).Times(1)
	s.userRepo.EXPECT().GetByID(userID).Return(user, nil).Times(1)
	s.refreshTokenRepo.EXPECT().Update(gomock.Any()).Return(nil).Times(1) // Revoke old token
	s.tokenService.EXPECT().GenerateAccessToken(user).Return("new_access_token", expiresAt, nil).Times(1)
	s.tokenService.EXPECT().GenerateRefreshToken(userID).Return("new_refresh_token", time.Now().Add(7*24*time.Hour), nil).Times(1)
	s.refreshTokenRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1) // successful token refresh audit log

	newTokens, err := s.authService.RefreshTokens(refreshToken, "192.168.1.1", "Mozilla/5.0")

	s.NoError(err)
	s.NotNil(newTokens)
	s.NotEmpty(newTokens.AccessToken)
	s.NotEmpty(newTokens.RefreshToken)
}

func (s *AuthServiceTestSuite) TestRefreshTokens_InvalidRefreshToken() {
	// Setup mock expectations
	s.tokenService.EXPECT().ValidateRefreshToken("invalid.refresh.token").Return(nil, errors.New("invalid token")).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1) // failed token refresh audit log

	tokens, err := s.authService.RefreshTokens("invalid.refresh.token", "192.168.1.1", "Mozilla/5.0")

	s.Equal(ErrInvalidRefreshToken, err)
	s.Nil(tokens)
}

func (s *AuthServiceTestSuite) TestRefreshTokens_UsingRevokedRefreshToken() {
	userID := uuid.New()
	refreshToken := "valid_refresh_token"
	tokenHash := "hashed_token"
	now := time.Now()

	user := &models.User{
		ID:        userID,
		Email:     "revoked@example.com",
		FirstName: "Rev",
		LastName:  "Oked",
		Role:      models.RoleCustomer,
	}

	claims := &models.CustomClaims{
		UserID: userID.String(),
	}

	// First refresh - should work
	storedToken := &models.RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		RevokedAt: nil,
	}

	expiresAt := now.Add(15 * time.Minute)

	// First refresh expectations
	s.tokenService.EXPECT().ValidateRefreshToken(refreshToken).Return(claims, nil).Times(1)
	s.refreshTokenRepo.EXPECT().GetByTokenHash(gomock.Any()).Return(storedToken, nil).Times(1)
	s.userRepo.EXPECT().GetByID(userID).Return(user, nil).Times(1)
	s.refreshTokenRepo.EXPECT().Update(gomock.Any()).Return(nil).Times(1) // Revoke old token
	s.tokenService.EXPECT().GenerateAccessToken(user).Return("new_access_token", expiresAt, nil).Times(1)
	s.tokenService.EXPECT().GenerateRefreshToken(userID).Return("new_refresh_token", now.Add(7*24*time.Hour), nil).Times(1)
	s.refreshTokenRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)

	newTokens, err := s.authService.RefreshTokens(refreshToken, "192.168.1.1", "Mozilla/5.0")
	s.NoError(err)
	s.NotNil(newTokens)

	// Try to use the original refresh token again - should fail
	revokedTime := now
	revokedToken := &models.RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		RevokedAt: &revokedTime,
	}

	s.tokenService.EXPECT().ValidateRefreshToken(refreshToken).Return(claims, nil).Times(1)
	s.refreshTokenRepo.EXPECT().GetByTokenHash(gomock.Any()).Return(revokedToken, nil).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1) // failed token refresh audit log

	tokens2, err := s.authService.RefreshTokens(refreshToken, "192.168.1.1", "Mozilla/5.0")
	s.Equal(ErrInvalidRefreshToken, err)
	s.Nil(tokens2)
}

func (s *AuthServiceTestSuite) TestLogout_SuccessfulLogout() {
	userID := uuid.New()
	accessToken := "valid_access_token"
	refreshToken := "valid_refresh_token"

	claims := &models.CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID: "jti-123",
		},
		UserID: userID.String(),
	}

	// Logout expectations
	s.tokenService.EXPECT().ValidateAccessToken(accessToken).Return(claims, nil).Times(1)
	s.tokenService.EXPECT().GetTokenExpiry(accessToken).Return(time.Now().Add(15*time.Minute), nil).Times(1)
	s.blacklistedTokenRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1) // blacklist token
	s.refreshTokenRepo.EXPECT().RevokeAllForUser(userID).Return(nil).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1) // logout audit log

	err := s.authService.Logout(accessToken, "192.168.1.1", "Mozilla/5.0")
	s.NoError(err)

	// After logout, refresh token should not work
	s.tokenService.EXPECT().ValidateRefreshToken(refreshToken).Return(claims, nil).Times(1)
	s.refreshTokenRepo.EXPECT().GetByTokenHash(gomock.Any()).Return(nil, repositories.ErrRefreshTokenNotFound).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1) // failed token refresh audit log

	newTokens, err := s.authService.RefreshTokens(refreshToken, "192.168.1.1", "Mozilla/5.0")
	s.Equal(ErrInvalidRefreshToken, err)
	s.Nil(newTokens)
}

func (s *AuthServiceTestSuite) TestLogout_WithInvalidToken() {
	// Setup mock expectations - invalid token
	s.tokenService.EXPECT().ValidateAccessToken("invalid.access.token").Return(nil, errors.New("invalid token")).Times(1)
	s.tokenService.EXPECT().GetJTI("invalid.access.token").Return("", errors.New("invalid token")).Times(1)

	// Should not error - logout is idempotent
	err := s.authService.Logout("invalid.access.token", "192.168.1.1", "Mozilla/5.0")
	s.NoError(err)
}

func (s *AuthServiceTestSuite) TestLogout_MultipleTimes() {
	userID := uuid.New()
	accessToken := "valid_access_token"

	claims := &models.CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID: "jti-123",
		},
		UserID: userID.String(),
	}

	// First logout - successful
	s.tokenService.EXPECT().ValidateAccessToken(accessToken).Return(claims, nil).Times(1)
	s.tokenService.EXPECT().GetTokenExpiry(accessToken).Return(time.Now().Add(15*time.Minute), nil).Times(1)
	s.blacklistedTokenRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1) // blacklist token
	s.refreshTokenRepo.EXPECT().RevokeAllForUser(userID).Return(nil).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1) // logout audit log

	err := s.authService.Logout(accessToken, "192.168.1.1", "Mozilla/5.0")
	s.NoError(err)

	// Second logout - should still be successful (idempotent)
	s.tokenService.EXPECT().ValidateAccessToken(accessToken).Return(claims, nil).Times(1)
	s.tokenService.EXPECT().GetTokenExpiry(accessToken).Return(time.Now().Add(15*time.Minute), nil).Times(1)
	s.blacklistedTokenRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1) // blacklist token
	s.refreshTokenRepo.EXPECT().RevokeAllForUser(userID).Return(nil).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1) // logout audit log

	err = s.authService.Logout(accessToken, "192.168.1.1", "Mozilla/5.0")
	s.NoError(err)
}

func (s *AuthServiceTestSuite) TestPasswordHashing_PasswordsAreHashedDifferently() {
	password := "SecurePass123!@#"

	req1 := &dto.RegisterRequest{
		Email:     "user1@example.com",
		Password:  password,
		FirstName: "User",
		LastName:  "One",
	}

	req2 := &dto.RegisterRequest{
		Email:     "user2@example.com",
		Password:  password,
		FirstName: "User",
		LastName:  "Two",
	}

	// First registration
	s.userRepo.EXPECT().GetByEmail(req1.Email).Return(nil, repositories.ErrUserNotFound).Times(1)
	s.passwordService.EXPECT().HashPassword(password).Return("hashed_password_1", nil).Times(1)
	s.userRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)
	s.accountService.EXPECT().CreateAccountsForNewUser(gomock.Any()).Return(nil).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(2) // account creation + registration audit logs

	user1, err := s.authService.Register(req1, "192.168.1.1", "Mozilla/5.0")
	s.Require().NoError(err)

	// Second registration with different hash (due to salt)
	s.userRepo.EXPECT().GetByEmail(req2.Email).Return(nil, repositories.ErrUserNotFound).Times(1)
	s.passwordService.EXPECT().HashPassword(password).Return("hashed_password_2", nil).Times(1)
	s.userRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)
	s.accountService.EXPECT().CreateAccountsForNewUser(gomock.Any()).Return(nil).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(2) // account creation + registration audit logs

	user2, err := s.authService.Register(req2, "192.168.1.1", "Mozilla/5.0")
	s.Require().NoError(err)

	// Same password should result in different hashes (due to salt)
	s.NotEqual(user1.PasswordHash, user2.PasswordHash)

	// But both should allow login with the same password
	userID1 := uuid.New()
	userID2 := uuid.New()
	user1Model := &models.User{
		ID:                  userID1,
		Email:               req1.Email,
		PasswordHash:        "hashed_password_1",
		FirstName:           "User",
		LastName:            "One",
		Role:                models.RoleCustomer,
		FailedLoginAttempts: 0,
	}
	user2Model := &models.User{
		ID:                  userID2,
		Email:               req2.Email,
		PasswordHash:        "hashed_password_2",
		FirstName:           "User",
		LastName:            "Two",
		Role:                models.RoleCustomer,
		FailedLoginAttempts: 0,
	}

	loginReq1 := &dto.LoginRequest{Email: req1.Email, Password: password}
	loginReq2 := &dto.LoginRequest{Email: req2.Email, Password: password}

	expiresAt := time.Now().Add(15 * time.Minute)

	// First login
	s.userRepo.EXPECT().GetByEmail(req1.Email).Return(user1Model, nil).Times(1)
	s.passwordService.EXPECT().ComparePassword(password, "hashed_password_1").Return(true).Times(1)
	s.userRepo.EXPECT().UpdateFailedLoginAttempts(gomock.Any()).Return(nil).Times(1)
	s.tokenService.EXPECT().GenerateAccessToken(user1Model).Return("access_token_1", expiresAt, nil).Times(1)
	s.tokenService.EXPECT().GenerateRefreshToken(userID1).Return("refresh_token_1", time.Now().Add(7*24*time.Hour), nil).Times(1)
	s.refreshTokenRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)

	tokens1, err := s.authService.Login(loginReq1, "192.168.1.1", "Mozilla/5.0")
	s.NoError(err)
	s.NotNil(tokens1)

	// Second login
	s.userRepo.EXPECT().GetByEmail(req2.Email).Return(user2Model, nil).Times(1)
	s.passwordService.EXPECT().ComparePassword(password, "hashed_password_2").Return(true).Times(1)
	s.userRepo.EXPECT().UpdateFailedLoginAttempts(gomock.Any()).Return(nil).Times(1)
	s.tokenService.EXPECT().GenerateAccessToken(user2Model).Return("access_token_2", expiresAt, nil).Times(1)
	s.tokenService.EXPECT().GenerateRefreshToken(userID2).Return("refresh_token_2", time.Now().Add(7*24*time.Hour), nil).Times(1)
	s.refreshTokenRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)
	s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)

	tokens2, err := s.authService.Login(loginReq2, "192.168.1.1", "Mozilla/5.0")
	s.NoError(err)
	s.NotNil(tokens2)
}

func (s *AuthServiceTestSuite) TestAuditLogging_RegistrationCreatesAuditLog() {
	req := &dto.RegisterRequest{
		Email:     "audit@example.com",
		Password:  "SecurePass123!@#",
		FirstName: "Audit",
		LastName:  "Test",
	}

	// Setup mock expectations - the audit log create should be called
	s.userRepo.EXPECT().GetByEmail(req.Email).Return(nil, repositories.ErrUserNotFound).Times(1)
	s.passwordService.EXPECT().HashPassword(req.Password).Return("hashed_password", nil).Times(1)
	s.userRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)
	s.accountService.EXPECT().CreateAccountsForNewUser(gomock.Any()).Return(nil).Times(1)

	// Capture the audit log to verify it was created
	var capturedAuditLog *models.AuditLog
	s.auditRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(log *models.AuditLog) error {
		if log.Action == models.AuditActionRegister {
			capturedAuditLog = log
		}
		return nil
	}).Times(2) // account creation + registration audit logs

	_, err := s.authService.Register(req, "192.168.1.1", "TestAgent")
	s.Require().NoError(err)

	// Verify that registration audit log was created
	s.NotNil(capturedAuditLog, "Registration audit log not found")
	s.Equal(models.AuditActionRegister, capturedAuditLog.Action)
	s.Equal("192.168.1.1", capturedAuditLog.IPAddress)
	s.Equal("TestAgent", capturedAuditLog.UserAgent)
}

func (s *AuthServiceTestSuite) TestAuditLogging_FailedLoginCreatesAuditLog() {
	req := &dto.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "WrongPass",
	}

	// Setup mock expectations
	s.userRepo.EXPECT().GetByEmail(req.Email).Return(nil, repositories.ErrUserNotFound).Times(1)

	// Capture the audit log to verify it was created
	var capturedAuditLog *models.AuditLog
	s.auditRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(log *models.AuditLog) error {
		if log.Action == models.AuditActionFailedLogin {
			capturedAuditLog = log
		}
		return nil
	}).Times(1) // failed login audit log

	_, err := s.authService.Login(req, "192.168.1.2", "TestAgent2")
	s.Error(err)

	// Verify that failed login audit log was created
	s.NotNil(capturedAuditLog, "Failed login audit log not found")
	s.Equal(models.AuditActionFailedLogin, capturedAuditLog.Action)
	s.Equal("192.168.1.2", capturedAuditLog.IPAddress)
	s.Equal("TestAgent2", capturedAuditLog.UserAgent)
}
