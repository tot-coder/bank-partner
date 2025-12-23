package repositories

import (
	"errors"
	"fmt"
	"time"

	"array-assessment/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

// RefreshTokenRepository handles database operations for refresh tokens
type RefreshTokenRepository struct {
	db *gorm.DB
}

// NewRefreshTokenRepository creates a new refresh token repository
func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepositoryInterface {
	return &RefreshTokenRepository{
		db: db,
	}
}

// Create creates a new refresh token in the database
func (r *RefreshTokenRepository) Create(token *models.RefreshToken) error {
	if token == nil {
		return errors.New("refresh token cannot be nil")
	}

	if err := r.db.Create(token).Error; err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

// GetByID retrieves a refresh token by its ID
func (r *RefreshTokenRepository) GetByID(id uuid.UUID) (*models.RefreshToken, error) {
	token := &models.RefreshToken{ID: id}
	if err := r.db.First(token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("failed to get refresh token by ID: %w", err)
	}

	return token, nil
}

// GetByTokenHash retrieves a refresh token by its hash
func (r *RefreshTokenRepository) GetByTokenHash(tokenHash string) (*models.RefreshToken, error) {
	var token models.RefreshToken

	if err := r.db.Where("token_hash = ?", tokenHash).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("failed to get refresh token by hash: %w", err)
	}

	return &token, nil
}

// GetActiveByUserID retrieves all active refresh tokens for a user
func (r *RefreshTokenRepository) GetActiveByUserID(userID uuid.UUID) ([]*models.RefreshToken, error) {
	var tokens []*models.RefreshToken

	err := r.db.Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Find(&tokens).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get active tokens for user: %w", err)
	}

	return tokens, nil
}

// Update updates a refresh token in the database
func (r *RefreshTokenRepository) Update(token *models.RefreshToken) error {
	if token == nil {
		return errors.New("refresh token cannot be nil")
	}

	if err := r.db.Save(token).Error; err != nil {
		return fmt.Errorf("failed to update refresh token: %w", err)
	}

	return nil
}

// Revoke revokes a specific refresh token
func (r *RefreshTokenRepository) Revoke(tokenID uuid.UUID) error {
	now := time.Now()

	result := r.db.Model(&models.RefreshToken{ID: tokenID}).
		Where("revoked_at IS NULL").
		Update("revoked_at", now)

	if result.Error != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrRefreshTokenNotFound
	}

	return nil
}

// RevokeAllForUser revokes all refresh tokens for a specific user
func (r *RefreshTokenRepository) RevokeAllForUser(userID uuid.UUID) error {
	now := time.Now()

	if err := r.db.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error; err != nil {
		return fmt.Errorf("failed to revoke all tokens for user: %w", err)
	}

	return nil
}

// DeleteExpired removes expired refresh tokens from the database
func (r *RefreshTokenRepository) DeleteExpired() (int64, error) {
	result := r.db.Where("expires_at < ?", time.Now()).Delete(&models.RefreshToken{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete expired tokens: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// DeleteRevokedOlderThan removes revoked tokens older than the specified duration
func (r *RefreshTokenRepository) DeleteRevokedOlderThan(duration time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-duration)

	result := r.db.Where("revoked_at IS NOT NULL AND revoked_at < ?", cutoffTime).
		Delete(&models.RefreshToken{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old revoked tokens: %w", result.Error)
	}

	return result.RowsAffected, nil
}
