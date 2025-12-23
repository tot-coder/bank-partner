package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BlacklistedToken struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	JTI           string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"jti"`
	UserID        uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	ExpiresAt     time.Time `gorm:"not null;index" json:"expires_at"`
	BlacklistedAt time.Time `gorm:"not null" json:"blacklisted_at"`

	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

func (bt *BlacklistedToken) IsExpired() bool {
	return time.Now().After(bt.ExpiresAt)
}

func (bt *BlacklistedToken) CanBeDeleted() bool {
	return bt.IsExpired()
}

func (bt *BlacklistedToken) TableName() string {
	return "blacklisted_tokens"
}

func (bt *BlacklistedToken) BeforeCreate(tx *gorm.DB) error {
	if bt.ID == uuid.Nil {
		bt.ID = uuid.New()
	}
	return nil
}
