package services

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"array-assessment/internal/dto"
	"array-assessment/internal/models"
	"array-assessment/internal/repositories"

	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrAccountLocked       = errors.New("account is locked due to too many failed attempts")
	ErrUserAlreadyExists   = errors.New("user with this email already exists")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo             repositories.UserRepositoryInterface
	refreshTokenRepo     repositories.RefreshTokenRepositoryInterface
	auditRepo            repositories.AuditLogRepositoryInterface
	blacklistedTokenRepo repositories.BlacklistedTokenRepositoryInterface
	passwordService      PasswordServiceInterface
	tokenService         TokenServiceInterface
	accountService       AccountServiceInterface
	logger               *slog.Logger
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo repositories.UserRepositoryInterface,
	refreshTokenRepo repositories.RefreshTokenRepositoryInterface,
	auditRepo repositories.AuditLogRepositoryInterface,
	blacklistedTokenRepo repositories.BlacklistedTokenRepositoryInterface,
	passwordService PasswordServiceInterface,
	tokenService TokenServiceInterface,
	accountService AccountServiceInterface,
	logger *slog.Logger,
) AuthServiceInterface {
	return &AuthService{
		userRepo:             userRepo,
		refreshTokenRepo:     refreshTokenRepo,
		auditRepo:            auditRepo,
		blacklistedTokenRepo: blacklistedTokenRepo,
		passwordService:      passwordService,
		tokenService:         tokenService,
		accountService:       accountService,
		logger:               logger,
	}
}

// Register creates a new user account
func (s *AuthService) Register(req *dto.RegisterRequest, ipAddress, userAgent string) (*models.User, error) {
	existingUser, err := s.userRepo.GetByEmail(req.Email)
	if err != nil && !errors.Is(err, repositories.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	if existingUser != nil {
		s.auditFailedRegistration(req.Email, ipAddress, userAgent, "email_already_exists")
		return nil, ErrUserAlreadyExists
	}

	hashedPassword, err := s.passwordService.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         models.RoleCustomer,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := s.accountService.CreateAccountsForNewUser(user.ID); err != nil {
		// Non-critical: Admin can retry account creation
		s.auditFailedAccountCreation(user, ipAddress, userAgent, err.Error())
	} else {
		s.auditSuccessfulAccountCreation(user, ipAddress, userAgent)
	}

	s.auditSuccessfulRegistration(user, ipAddress, userAgent)

	return user, nil
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(req *dto.LoginRequest, ipAddress, userAgent string) (*dto.TokenResponse, error) {
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			s.auditFailedLogin(req.Email, ipAddress, userAgent, "user_not_found")
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.IsLocked() {
		s.auditFailedLogin(req.Email, ipAddress, userAgent, "account_locked")
		return nil, ErrAccountLocked
	}

	if !s.passwordService.ComparePassword(req.Password, user.PasswordHash) {
		user.IncrementFailedAttempts()
		if err := s.userRepo.UpdateFailedLoginAttempts(user); err != nil {
			// Security: Never reveal user existence via error messages
			s.logger.Error("failed to update login attempts",
				"error", err,
				"user_id", user.ID,
				"email", user.Email)
		}

		if user.IsLocked() {
			s.auditAccountLocked(user, ipAddress, userAgent)
		}

		s.auditFailedLogin(req.Email, ipAddress, userAgent, "invalid_password")
		return nil, ErrInvalidCredentials
	}

	user.ResetFailedAttempts()
	if err := s.userRepo.UpdateFailedLoginAttempts(user); err != nil {
		// Non-critical: Audit logging failure shouldn't block login
		s.logger.Warn("failed to reset login attempts",
			"error", err,
			"user_id", user.ID,
			"email", user.Email)
	}

	tokens, err := s.generateTokens(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	s.auditSuccessfulLogin(user, ipAddress, userAgent)

	return tokens, nil
}

// RefreshTokens generates new tokens using a refresh token
func (s *AuthService) RefreshTokens(refreshToken, ipAddress, userAgent string) (*dto.TokenResponse, error) {
	claims, err := s.tokenService.ValidateRefreshToken(refreshToken)
	if err != nil {
		s.auditFailedTokenRefresh("", ipAddress, userAgent, "invalid_token")
		return nil, ErrInvalidRefreshToken
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token: %w", err)
	}

	storedToken, err := s.refreshTokenRepo.GetByTokenHash(hashToken(refreshToken))
	if err != nil {
		s.auditFailedTokenRefresh(claims.UserID, ipAddress, userAgent, "token_not_found")
		return nil, ErrInvalidRefreshToken
	}

	if !storedToken.IsValid() {
		s.auditFailedTokenRefresh(claims.UserID, ipAddress, userAgent, "token_expired_or_revoked")
		return nil, ErrInvalidRefreshToken
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	storedToken.Revoke()
	if err := s.refreshTokenRepo.Update(storedToken); err != nil {
		// Non-critical: Token revocation failure shouldn't block refresh
		s.logger.Warn("failed to revoke old token",
			"error", err,
			"user_id", user.ID,
			"token_id", storedToken.ID)
	}

	tokens, err := s.generateTokens(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new tokens: %w", err)
	}

	s.auditSuccessfulTokenRefresh(user, ipAddress, userAgent)

	return tokens, nil
}

// Logout invalidates the user's tokens
func (s *AuthService) Logout(accessToken, ipAddress, userAgent string) error {
	claims, err := s.tokenService.ValidateAccessToken(accessToken)
	if err != nil {
		// Security: Blacklist even expired tokens to prevent reuse
		jti, _ := s.tokenService.GetJTI(accessToken)
		if jti != "" {
			if err := s.blacklistToken(jti, uuid.Nil, time.Now().Add(24*time.Hour)); err != nil {
				s.logger.Error("failed to blacklist expired token",
					"error", err,
					"jti", jti)
			}
		}
		return nil
	}

	userID, _ := uuid.Parse(claims.UserID)

	expiry, _ := s.tokenService.GetTokenExpiry(accessToken)
	if err := s.blacklistToken(claims.ID, userID, expiry); err != nil {
		s.logger.Error("failed to blacklist token",
			"error", err,
			"jti", claims.ID,
			"user_id", userID)
	}

	if err := s.refreshTokenRepo.RevokeAllForUser(userID); err != nil {
		// Non-critical: Token revocation failure shouldn't block refresh
		s.logger.Warn("failed to revoke refresh tokens",
			"error", err,
			"user_id", userID)
	}

	s.auditLogout(userID, ipAddress, userAgent)

	return nil
}

func (s *AuthService) generateTokens(user *models.User) (*dto.TokenResponse, error) {
	accessToken, expiresAt, err := s.tokenService.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshExpiresAt, err := s.tokenService.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	refreshTokenModel := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: hashToken(refreshToken),
		ExpiresAt: refreshExpiresAt,
	}

	if err := s.refreshTokenRepo.Create(refreshTokenModel); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *AuthService) blacklistToken(jti string, userID uuid.UUID, expiresAt time.Time) error {
	token := &models.BlacklistedToken{
		JTI:       jti,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
	return s.blacklistedTokenRepo.Create(token)
}

func hashToken(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// Audit logging methods
func (s *AuthService) auditSuccessfulRegistration(user *models.User, ipAddress, userAgent string) {
	s.createAuditLog(&user.ID, models.AuditActionRegister, "user", user.ID.String(), ipAddress, userAgent, nil)
}

func (s *AuthService) auditFailedRegistration(email, ipAddress, userAgent, reason string) {
	metadata := map[string]interface{}{
		"email":  email,
		"reason": reason,
	}
	s.createAuditLog(nil, models.AuditActionRegister, "user", "", ipAddress, userAgent, metadata)
}

func (s *AuthService) auditSuccessfulLogin(user *models.User, ipAddress, userAgent string) {
	s.createAuditLog(&user.ID, models.AuditActionLogin, "user", user.ID.String(), ipAddress, userAgent, nil)
}

func (s *AuthService) auditFailedLogin(email, ipAddress, userAgent, reason string) {
	metadata := map[string]interface{}{
		"email":  email,
		"reason": reason,
	}
	s.createAuditLog(nil, models.AuditActionFailedLogin, "user", "", ipAddress, userAgent, metadata)
}

func (s *AuthService) auditAccountLocked(user *models.User, ipAddress, userAgent string) {
	s.createAuditLog(&user.ID, models.AuditActionAccountLocked, "user", user.ID.String(), ipAddress, userAgent, nil)
}

func (s *AuthService) auditSuccessfulTokenRefresh(user *models.User, ipAddress, userAgent string) {
	s.createAuditLog(&user.ID, models.AuditActionTokenRefresh, "user", user.ID.String(), ipAddress, userAgent, nil)
}

func (s *AuthService) auditFailedTokenRefresh(userID, ipAddress, userAgent, reason string) {
	var uid *uuid.UUID
	if userID != "" {
		id, _ := uuid.Parse(userID)
		uid = &id
	}
	metadata := map[string]interface{}{
		"reason": reason,
	}
	s.createAuditLog(uid, models.AuditActionTokenRefresh, "token", "", ipAddress, userAgent, metadata)
}

func (s *AuthService) auditLogout(userID uuid.UUID, ipAddress, userAgent string) {
	s.createAuditLog(&userID, models.AuditActionLogout, "user", userID.String(), ipAddress, userAgent, nil)
}

func (s *AuthService) auditSuccessfulAccountCreation(user *models.User, ipAddress, userAgent string) {
	metadata := map[string]interface{}{
		"message": "default accounts created",
	}
	s.createAuditLog(&user.ID, "account.auto_created", "user", user.ID.String(), ipAddress, userAgent, metadata)
}

func (s *AuthService) auditFailedAccountCreation(user *models.User, ipAddress, userAgent, reason string) {
	metadata := map[string]interface{}{
		"reason": reason,
	}
	s.createAuditLog(&user.ID, "account.auto_creation_failed", "user", user.ID.String(), ipAddress, userAgent, metadata)
}

func (s *AuthService) createAuditLog(userID *uuid.UUID, action, resource, resourceID, ipAddress, userAgent string, metadata map[string]interface{}) {
	log := &models.AuditLog{
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Metadata:   metadata,
	}

	if err := s.auditRepo.Create(log); err != nil {
		// Non-critical: Audit logging failure shouldn't block operations
		s.logger.Error("failed to create audit log",
			"error", err,
			"action", action,
			"resource", resource,
			"resource_id", resourceID)
	}
}
