package services

import (
	"crypto/rsa"
	"testing"
	"time"

	"array-assessment/internal/config"
	"array-assessment/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// TokenServiceTestSuite defines the test suite for TokenService
type TokenServiceTestSuite struct {
	suite.Suite
	privateKey      *rsa.PrivateKey
	publicKey       *rsa.PublicKey
	service         TokenServiceInterface
	issuer          string
	accessDuration  time.Duration
	refreshDuration time.Duration
}

// SetupTest runs before each test
func (s *TokenServiceTestSuite) SetupTest() {
	var err error
	s.privateKey, s.publicKey, err = config.GenerateRSAKeyPair()
	s.Require().NoError(err)

	s.issuer = "test-issuer"
	s.accessDuration = 24 * time.Hour
	s.refreshDuration = 7 * 24 * time.Hour

	s.service = NewTokenService(&config.JWTConfig{
		PrivateKey:           s.privateKey,
		PublicKey:            s.publicKey,
		Issuer:               s.issuer,
		AccessTokenDuration:  s.accessDuration,
		RefreshTokenDuration: s.refreshDuration,
	})
}

// TestTokenServiceSuite runs the test suite
func TestTokenServiceSuite(t *testing.T) {
	suite.Run(t, new(TokenServiceTestSuite))
}

// Test GenerateKeyPair
func (s *TokenServiceTestSuite) TestGenerateKeyPair() {
	privateKey, publicKey, err := config.GenerateRSAKeyPair()
	s.NoError(err)
	s.NotNil(privateKey)
	s.NotNil(publicKey)
}

// Test GenerateAccessToken
func (s *TokenServiceTestSuite) TestGenerateAccessToken() {
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleCustomer,
	}

	token, expiresAt, err := s.service.GenerateAccessToken(user)
	s.NoError(err)
	s.NotEmpty(token)
	s.True(expiresAt.After(time.Now()))
	s.True(expiresAt.Before(time.Now().Add(25 * time.Hour)))
}

// Test GenerateRefreshToken
func (s *TokenServiceTestSuite) TestGenerateRefreshToken() {
	userID := uuid.New()
	token, expiresAt, err := s.service.GenerateRefreshToken(userID)
	s.NoError(err)
	s.NotEmpty(token)
	s.True(expiresAt.After(time.Now()))
	s.True(expiresAt.Before(time.Now().Add(8 * 24 * time.Hour)))
}

// Test ValidateAccessToken with valid token
func (s *TokenServiceTestSuite) TestValidateAccessToken_Success() {
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleCustomer,
	}

	// Generate a valid token
	token, _, err := s.service.GenerateAccessToken(user)
	s.Require().NoError(err)

	// Validate the token
	claims, err := s.service.ValidateAccessToken(token)
	s.NoError(err)
	s.NotNil(claims)
	s.Equal(user.ID.String(), claims.UserID)
	s.Equal(user.Email, claims.Email)
	s.Equal(user.Role, claims.Role)
	s.Equal(s.issuer, claims.Issuer)
}

// Test ValidateAccessToken with empty token
func (s *TokenServiceTestSuite) TestValidateAccessToken_EmptyToken() {
	claims, err := s.service.ValidateAccessToken("")
	s.Error(err)
	s.Contains(err.Error(), "empty token")
	s.Nil(claims)
}

// Test ValidateAccessToken with invalid format
func (s *TokenServiceTestSuite) TestValidateAccessToken_InvalidFormat() {
	claims, err := s.service.ValidateAccessToken("invalid.token.format")
	s.Error(err)
	s.Contains(err.Error(), "invalid token")
	s.Nil(claims)
}

// Test ValidateAccessToken with malformed token
func (s *TokenServiceTestSuite) TestValidateAccessToken_MalformedToken() {
	claims, err := s.service.ValidateAccessToken("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature")
	s.Error(err)
	s.Contains(err.Error(), "invalid token")
	s.Nil(claims)
}

// Test ValidateRefreshToken with valid token
func (s *TokenServiceTestSuite) TestValidateRefreshToken_Success() {
	userID := uuid.New()

	// Generate a valid refresh token
	token, _, err := s.service.GenerateRefreshToken(userID)
	s.Require().NoError(err)

	// Validate the token
	claims, err := s.service.ValidateRefreshToken(token)
	s.NoError(err)
	s.NotNil(claims)
	s.Equal(userID.String(), claims.UserID)
	s.Equal(TokenTypeRefresh, claims.TokenType)
}

// Test expired token
func (s *TokenServiceTestSuite) TestExpiredToken() {
	// Create service with very short duration
	shortService := NewTokenService(&config.JWTConfig{
		PrivateKey:           s.privateKey,
		PublicKey:            s.publicKey,
		Issuer:               s.issuer,
		AccessTokenDuration:  1 * time.Millisecond,
		RefreshTokenDuration: 1 * time.Millisecond,
	})

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleCustomer,
	}

	token, _, err := shortService.GenerateAccessToken(user)
	s.NoError(err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Try to validate expired token
	claims, err := shortService.ValidateAccessToken(token)
	s.Error(err)
	s.Contains(err.Error(), "token is expired")
	s.Nil(claims)
}

// Test wrong issuer
func (s *TokenServiceTestSuite) TestWrongIssuer() {
	service1 := NewTokenService(&config.JWTConfig{
		PrivateKey:           s.privateKey,
		PublicKey:            s.publicKey,
		Issuer:               "issuer1",
		AccessTokenDuration:  24 * time.Hour,
		RefreshTokenDuration: 7 * 24 * time.Hour,
	})

	service2 := NewTokenService(&config.JWTConfig{
		PrivateKey:           s.privateKey,
		PublicKey:            s.publicKey,
		Issuer:               "issuer2",
		AccessTokenDuration:  24 * time.Hour,
		RefreshTokenDuration: 7 * 24 * time.Hour,
	})

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleCustomer,
	}

	// Generate token with issuer1
	token, _, err := service1.GenerateAccessToken(user)
	s.NoError(err)

	// Try to validate with different issuer
	claims, err := service2.ValidateAccessToken(token)
	s.Error(err)
	s.Contains(err.Error(), "invalid issuer")
	s.Nil(claims)
}

// Test different keys
func (s *TokenServiceTestSuite) TestDifferentKeys() {
	privateKey2, publicKey2, err := config.GenerateRSAKeyPair()
	s.Require().NoError(err)

	service1 := NewTokenService(&config.JWTConfig{
		PrivateKey:           s.privateKey,
		PublicKey:            s.publicKey,
		Issuer:               s.issuer,
		AccessTokenDuration:  24 * time.Hour,
		RefreshTokenDuration: 7 * 24 * time.Hour,
	})

	service2 := NewTokenService(&config.JWTConfig{
		PrivateKey:           privateKey2,
		PublicKey:            publicKey2,
		Issuer:               s.issuer,
		AccessTokenDuration:  24 * time.Hour,
		RefreshTokenDuration: 7 * 24 * time.Hour,
	})

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleCustomer,
	}

	// Generate token with key pair 1
	token, _, err := service1.GenerateAccessToken(user)
	s.NoError(err)

	// Try to validate with different key pair
	claims, err := service2.ValidateAccessToken(token)
	s.Error(err)
	s.Contains(err.Error(), "invalid token")
	s.Nil(claims)
}

// Test ExtractTokenFromHeader with valid bearer token
func (s *TokenServiceTestSuite) TestExtractTokenFromHeader_ValidBearer() {
	token, err := s.service.ExtractTokenFromHeader("Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.token")
	s.NoError(err)
	s.Equal("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.token", token)
}

// Test ExtractTokenFromHeader with lowercase bearer
func (s *TokenServiceTestSuite) TestExtractTokenFromHeader_LowercaseBearer() {
	token, err := s.service.ExtractTokenFromHeader("bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.token")
	s.NoError(err)
	s.Equal("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.token", token)
}

// Test ExtractTokenFromHeader with no bearer prefix
func (s *TokenServiceTestSuite) TestExtractTokenFromHeader_NoBearer() {
	token, err := s.service.ExtractTokenFromHeader("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.token")
	s.Error(err)
	s.Empty(token)
}

// Test ExtractTokenFromHeader with empty header
func (s *TokenServiceTestSuite) TestExtractTokenFromHeader_Empty() {
	token, err := s.service.ExtractTokenFromHeader("")
	s.Error(err)
	s.Empty(token)
}

// Test ExtractTokenFromHeader with only bearer
func (s *TokenServiceTestSuite) TestExtractTokenFromHeader_OnlyBearer() {
	token, err := s.service.ExtractTokenFromHeader("Bearer")
	s.Error(err)
	s.Empty(token)
}

// Test ExtractTokenFromHeader with bearer and space only
func (s *TokenServiceTestSuite) TestExtractTokenFromHeader_BearerSpaceOnly() {
	token, err := s.service.ExtractTokenFromHeader("Bearer ")
	s.Error(err)
	s.Empty(token)
}

// Test GetJTI
func (s *TokenServiceTestSuite) TestGetJTI() {
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleCustomer,
	}

	token, _, err := s.service.GenerateAccessToken(user)
	s.NoError(err)

	jti, err := s.service.GetJTI(token)
	s.NoError(err)
	s.NotEmpty(jti)

	// Verify JTI is a valid UUID
	_, err = uuid.Parse(jti)
	s.NoError(err)
}

// Benchmarks
func BenchmarkTokenService_GenerateAccessToken(b *testing.B) {
	privateKey, publicKey, err := config.GenerateRSAKeyPair()
	if err != nil {
		b.Fatal(err)
	}

	ts := NewTokenService(&config.JWTConfig{
		PrivateKey:           privateKey,
		PublicKey:            publicKey,
		Issuer:               "test-issuer",
		AccessTokenDuration:  24 * time.Hour,
		RefreshTokenDuration: 7 * 24 * time.Hour,
	})

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleCustomer,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := ts.GenerateAccessToken(user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTokenService_ValidateAccessToken(b *testing.B) {
	privateKey, publicKey, err := config.GenerateRSAKeyPair()
	if err != nil {
		b.Fatal(err)
	}

	ts := NewTokenService(&config.JWTConfig{
		PrivateKey:           privateKey,
		PublicKey:            publicKey,
		Issuer:               "test-issuer",
		AccessTokenDuration:  24 * time.Hour,
		RefreshTokenDuration: 7 * 24 * time.Hour,
	})

	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleCustomer,
	}

	token, _, err := ts.GenerateAccessToken(user)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ts.ValidateAccessToken(token)
		if err != nil {
			b.Fatal(err)
		}
	}
}
