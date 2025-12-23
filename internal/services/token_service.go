package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"array-assessment/internal/config"
	"array-assessment/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrExpiredToken      = errors.New("token is expired")
	ErrInvalidIssuer     = errors.New("invalid issuer")
	ErrInvalidTokenType  = errors.New("invalid token type")
	ErrEmptyToken        = errors.New("empty token")
	ErrInvalidAuthHeader = errors.New("invalid authorization header format")
)

// TokenService handles JWT token generation and validation
type TokenService struct {
	config.JWTConfig
}

// NewTokenService creates a new token service from JWT configuration
func NewTokenService(jwtConfig *config.JWTConfig) TokenServiceInterface {
	return &TokenService{
		JWTConfig: *jwtConfig,
	}
}

// GenerateAccessToken generates a new JWT access token for a user
func (ts *TokenService) GenerateAccessToken(user *models.User) (string, time.Time, error) {
	if user == nil {
		return "", time.Time{}, errors.New("user cannot be nil")
	}

	now := time.Now()
	expiresAt := now.Add(ts.AccessTokenDuration)

	claims := ts.buildAccessTokenClaims(user, now, expiresAt)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	tokenString, err := token.SignedString(ts.PrivateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// GenerateRefreshToken generates a new JWT refresh token
func (ts *TokenService) GenerateRefreshToken(userID uuid.UUID) (string, time.Time, error) {
	if userID == uuid.Nil {
		return "", time.Time{}, errors.New("user ID cannot be nil")
	}

	now := time.Now()
	expiresAt := now.Add(ts.RefreshTokenDuration)

	claims := ts.buildRefreshTokenClaims(userID, now, expiresAt)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	tokenString, err := token.SignedString(ts.PrivateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// ValidateAccessToken validates and parses an access token
func (ts *TokenService) ValidateAccessToken(tokenString string) (*models.CustomClaims, error) {
	return ts.validateToken(tokenString, TokenTypeAccess)
}

// ValidateRefreshToken validates and parses a refresh token
func (ts *TokenService) ValidateRefreshToken(tokenString string) (*models.CustomClaims, error) {
	return ts.validateToken(tokenString, TokenTypeRefresh)
}

// ExtractTokenFromHeader extracts the JWT token from the Authorization header
func (ts *TokenService) ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", ErrInvalidAuthHeader
	}

	const bearerPrefix = "bearer "
	if !strings.HasPrefix(strings.ToLower(authHeader), bearerPrefix) {
		return "", ErrInvalidAuthHeader
	}

	token := strings.TrimSpace(authHeader[len(bearerPrefix):])
	if token == "" {
		return "", ErrInvalidAuthHeader
	}

	return token, nil
}

// GetJTI extracts the JTI (JWT ID) from a token without full validation
func (ts *TokenService) GetJTI(tokenString string) (string, error) {
	claims, err := ts.extractUnverifiedClaims(tokenString)
	if err != nil {
		return "", err
	}
	return claims.ID, nil
}

// GetTokenExpiry returns the expiry time of a token
func (ts *TokenService) GetTokenExpiry(tokenString string) (time.Time, error) {
	claims, err := ts.extractUnverifiedClaims(tokenString)
	if err != nil {
		return time.Time{}, err
	}

	if claims.ExpiresAt == nil {
		return time.Time{}, ErrInvalidToken
	}

	return claims.ExpiresAt.Time, nil
}

func (ts *TokenService) buildAccessTokenClaims(user *models.User, issuedAt, expiresAt time.Time) models.CustomClaims {
	return models.CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ts.Issuer,
			Subject:   user.Email,
			ID:        uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(issuedAt),
		},
		UserID:    user.ID.String(),
		Email:     user.Email,
		Role:      user.Role,
		TokenType: TokenTypeAccess,
	}
}

func (ts *TokenService) buildRefreshTokenClaims(userID uuid.UUID, issuedAt, expiresAt time.Time) models.CustomClaims {
	return models.CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ts.Issuer,
			Subject:   userID.String(),
			ID:        uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(issuedAt),
		},
		UserID:    userID.String(),
		TokenType: TokenTypeRefresh,
	}
}

func (ts *TokenService) validateToken(tokenString string, expectedType string) (*models.CustomClaims, error) {
	if tokenString == "" {
		return nil, ErrEmptyToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &models.CustomClaims{}, ts.keyFunc)
	if err != nil {
		return nil, ts.mapTokenError(err)
	}

	claims, ok := token.Claims.(*models.CustomClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if err := ts.validateClaims(claims, expectedType); err != nil {
		return nil, err
	}

	return claims, nil
}

func (ts *TokenService) keyFunc(token *jwt.Token) (interface{}, error) {
	// RS256 required per security standards for key rotation capability
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}
	return ts.PublicKey, nil
}

func (ts *TokenService) mapTokenError(err error) error {
	if errors.Is(err, jwt.ErrTokenExpired) {
		return ErrExpiredToken
	}
	return fmt.Errorf("%w: %v", ErrInvalidToken, err)
}

func (ts *TokenService) validateClaims(claims *models.CustomClaims, expectedType string) error {
	if claims.Issuer != ts.Issuer {
		return ErrInvalidIssuer
	}

	if claims.TokenType != expectedType {
		return ErrInvalidTokenType
	}

	return nil
}

func (ts *TokenService) extractUnverifiedClaims(tokenString string) (*models.CustomClaims, error) {
	if tokenString == "" {
		return nil, ErrEmptyToken
	}

	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenString, &models.CustomClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*models.CustomClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
