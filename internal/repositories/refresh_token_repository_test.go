package repositories

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"array-assessment/internal/database"
	"array-assessment/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

func TestRefreshTokenRepository(t *testing.T) {
	suite.Run(t, new(RefreshTokenRepositorySuite))
}

type RefreshTokenRepositorySuite struct {
	suite.Suite
	db   *database.DB
	repo RefreshTokenRepositoryInterface
}

func (s *RefreshTokenRepositorySuite) SetupTest() {
	s.db = database.SetupTestDB(s.T())
	s.repo = NewRefreshTokenRepository(s.db.DB)
}

func (s *RefreshTokenRepositorySuite) TearDownTest() {
	database.CleanupTestDB(s.T(), s.db)
}

func (s *RefreshTokenRepositorySuite) hashToken(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func (s *RefreshTokenRepositorySuite) TestRefreshTokenRepository_Create() {
	userID := uuid.New()

	token := &models.RefreshToken{
		UserID:    userID,
		TokenHash: s.hashToken("test.refresh.token"),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	err := s.repo.Create(token)
	s.NoError(err)
	s.NotEqual(uuid.Nil, token.ID)
	s.NotZero(token.CreatedAt)
}

func (s *RefreshTokenRepositorySuite) TestRefreshTokenRepository_GetByTokenHash() {
	userID := uuid.New()
	tokenHash := s.hashToken("test.refresh.token")

	// Create token
	token := &models.RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	err := s.repo.Create(token)
	s.NoError(err)

	// Get by token hash
	foundToken, err := s.repo.GetByTokenHash(tokenHash)
	s.NoError(err)
	s.Equal(token.ID, foundToken.ID)
	s.Equal(tokenHash, foundToken.TokenHash)
	s.Equal(userID, foundToken.UserID)
	s.False(foundToken.IsRevoked())

	// Try to get non-existent token
	_, err = s.repo.GetByTokenHash("non-existent-hash")
	s.Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *RefreshTokenRepositorySuite) TestRefreshTokenRepository_GetActiveByUserID() {
	userID := uuid.New()
	otherUserID := uuid.New()

	// Create multiple active tokens for the same user
	for i := 0; i < 3; i++ {
		token := &models.RefreshToken{
			UserID:    userID,
			TokenHash: s.hashToken(fmt.Sprintf("token.%d", i)),
			ExpiresAt: time.Now().Add(time.Duration(i+1) * 24 * time.Hour),
		}
		err := s.repo.Create(token)
		s.NoError(err)
	}

	// Create a revoked token for the same user
	revokedToken := &models.RefreshToken{
		UserID:    userID,
		TokenHash: s.hashToken("revoked.token"),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	err := s.repo.Create(revokedToken)
	s.NoError(err)
	revokedToken.Revoke()
	err = s.repo.Update(revokedToken)
	s.NoError(err)

	// Create token for different user
	otherToken := &models.RefreshToken{
		UserID:    otherUserID,
		TokenHash: s.hashToken("other.token"),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	err = s.repo.Create(otherToken)
	s.NoError(err)

	// Get active tokens for first user (should not include revoked)
	tokens, err := s.repo.GetActiveByUserID(userID)
	s.NoError(err)
	s.Len(tokens, 3) // Only the 3 active tokens

	// Verify all tokens belong to the correct user and are active
	for _, token := range tokens {
		s.Equal(userID, token.UserID)
		s.False(token.IsRevoked())
	}

	// Get tokens for other user
	tokens, err = s.repo.GetActiveByUserID(otherUserID)
	s.NoError(err)
	s.Len(tokens, 1)
}

func (s *RefreshTokenRepositorySuite) TestRefreshTokenRepository_Update() {
	userID := uuid.New()

	// Create token
	token := &models.RefreshToken{
		UserID:    userID,
		TokenHash: s.hashToken("test.refresh.token"),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	err := s.repo.Create(token)
	s.NoError(err)

	// Update token (revoke it)
	token.Revoke()
	err = s.repo.Update(token)
	s.NoError(err)

	// Verify update
	updatedToken, err := s.repo.GetByTokenHash(token.TokenHash)
	s.NoError(err)
	s.True(updatedToken.IsRevoked())
	s.NotNil(updatedToken.RevokedAt)
}

func (s *RefreshTokenRepositorySuite) TestRefreshTokenRepository_RevokeAllForUser() {
	userID := uuid.New()
	otherUserID := uuid.New()

	// Create multiple tokens for the user
	tokenHashes := []string{}
	for i := 0; i < 3; i++ {
		hash := s.hashToken(fmt.Sprintf("token.%d", i))
		tokenHashes = append(tokenHashes, hash)
		token := &models.RefreshToken{
			UserID:    userID,
			TokenHash: hash,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err := s.repo.Create(token)
		s.NoError(err)
	}

	// Create token for different user
	otherTokenHash := s.hashToken("other.token")
	otherToken := &models.RefreshToken{
		UserID:    otherUserID,
		TokenHash: otherTokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	err := s.repo.Create(otherToken)
	s.NoError(err)

	// Revoke all tokens for first user
	err = s.repo.RevokeAllForUser(userID)
	s.NoError(err)

	// Verify all tokens for first user are revoked
	for _, hash := range tokenHashes {
		token, err := s.repo.GetByTokenHash(hash)
		s.NoError(err)
		s.True(token.IsRevoked())
		s.NotNil(token.RevokedAt)
	}

	// Verify other user's token is not affected
	otherTokenAfter, err := s.repo.GetByTokenHash(otherTokenHash)
	s.NoError(err)
	s.False(otherTokenAfter.IsRevoked())
	s.Nil(otherTokenAfter.RevokedAt)
}

func (s *RefreshTokenRepositorySuite) TestRefreshTokenRepository_DeleteExpired() {
	userID := uuid.New()

	// Create expired token
	expiredToken := &models.RefreshToken{
		UserID:    userID,
		TokenHash: s.hashToken("expired.token"),
		ExpiresAt: time.Now().Add(-24 * time.Hour), // Already expired
	}
	err := s.repo.Create(expiredToken)
	s.NoError(err)
	expiredHash := expiredToken.TokenHash

	// Create valid token
	validToken := &models.RefreshToken{
		UserID:    userID,
		TokenHash: s.hashToken("valid.token"),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	err = s.repo.Create(validToken)
	s.NoError(err)
	validHash := validToken.TokenHash

	// Delete expired tokens
	count, err := s.repo.DeleteExpired()
	s.NoError(err)
	s.GreaterOrEqual(count, int64(1))

	// Verify expired token is deleted
	_, err = s.repo.GetByTokenHash(expiredHash)
	s.Error(err)
	s.Contains(err.Error(), "not found")

	// Verify valid token still exists
	stillValid, err := s.repo.GetByTokenHash(validHash)
	s.NoError(err)
	s.Equal(validHash, stillValid.TokenHash)
}

func (s *RefreshTokenRepositorySuite) TestRefreshTokenRepository_DeleteRevokedOlderThan() {
	userID := uuid.New()

	// Create and immediately revoke an old token
	oldRevokedToken := &models.RefreshToken{
		UserID:    userID,
		TokenHash: s.hashToken("old.revoked.token"),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	err := s.repo.Create(oldRevokedToken)
	s.NoError(err)
	oldRevokedToken.Revoke()
	oldRevokedToken.RevokedAt = func(t time.Time) *time.Time { return &t }(time.Now().Add(-48 * time.Hour))
	err = s.repo.Update(oldRevokedToken)
	s.NoError(err)
	oldRevokedHash := oldRevokedToken.TokenHash

	// Create recently revoked token
	recentRevokedToken := &models.RefreshToken{
		UserID:    userID,
		TokenHash: s.hashToken("recent.revoked.token"),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	err = s.repo.Create(recentRevokedToken)
	s.NoError(err)
	recentRevokedToken.Revoke()
	err = s.repo.Update(recentRevokedToken)
	s.NoError(err)
	recentRevokedHash := recentRevokedToken.TokenHash

	// Create active token
	activeToken := &models.RefreshToken{
		UserID:    userID,
		TokenHash: s.hashToken("active.token"),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	err = s.repo.Create(activeToken)
	s.NoError(err)
	activeHash := activeToken.TokenHash

	// Delete revoked tokens older than 24 hours
	count, err := s.repo.DeleteRevokedOlderThan(24 * time.Hour)
	s.NoError(err)
	s.GreaterOrEqual(count, int64(1))

	// Verify old revoked token is deleted
	_, err = s.repo.GetByTokenHash(oldRevokedHash)
	s.Error(err)
	s.Contains(err.Error(), "not found")

	// Verify recent revoked token still exists
	stillRevoked, err := s.repo.GetByTokenHash(recentRevokedHash)
	s.NoError(err)
	s.True(stillRevoked.IsRevoked())

	// Verify active token still exists
	stillActive, err := s.repo.GetByTokenHash(activeHash)
	s.NoError(err)
	s.False(stillActive.IsRevoked())
}
