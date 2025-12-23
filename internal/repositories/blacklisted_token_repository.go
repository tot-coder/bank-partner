package repositories

import (
	"errors"
	"time"

	"array-assessment/internal/models"

	"gorm.io/gorm"
)

var (
	ErrTokenNotFound = errors.New("token not found")
)

type blacklistedTokenRepository struct {
	db *gorm.DB
}

// NewBlacklistedTokenRepository creates a new blacklisted token repository
func NewBlacklistedTokenRepository(db *gorm.DB) BlacklistedTokenRepositoryInterface {
	return &blacklistedTokenRepository{db: db}
}

// Create adds a token to the blacklist
func (r *blacklistedTokenRepository) Create(token *models.BlacklistedToken) error {
	token.BlacklistedAt = time.Now()
	return r.db.Create(token).Error
}

// GetByJTI retrieves a blacklisted token by its JTI
func (r *blacklistedTokenRepository) GetByJTI(jti string) (*models.BlacklistedToken, error) {
	var token models.BlacklistedToken
	err := r.db.Where("jti = ?", jti).First(&token).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	return &token, nil
}

// DeleteExpired removes expired tokens from the blacklist
func (r *blacklistedTokenRepository) DeleteExpired() (int64, error) {
	result := r.db.Where("expires_at < ?", time.Now()).Delete(&models.BlacklistedToken{})
	return result.RowsAffected, result.Error
}
