package models

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	RoleCustomer = "customer"
	RoleAdmin    = "admin"

	MaxFailedLoginAttempts = 3
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

type User struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	Email               string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash        string     `gorm:"type:varchar(255);not null" json:"-"`
	FirstName           string     `gorm:"type:varchar(100);not null" json:"first_name"`
	LastName            string     `gorm:"type:varchar(100);not null" json:"last_name"`
	Role                string     `gorm:"type:varchar(20);not null;default:'customer'" json:"role"`
	FailedLoginAttempts int        `gorm:"default:0" json:"-"`
	LockedAt            *time.Time `gorm:"index" json:"locked_at,omitempty"`
	LastLoginAt         *time.Time      `gorm:"index" json:"last_login_at,omitempty"`
	CreatedAt           time.Time       `gorm:"not null" json:"created_at"`
	UpdatedAt           time.Time       `gorm:"not null" json:"updated_at"`
	DeletedAt           gorm.DeletedAt  `gorm:"index" json:"deleted_at,omitempty"`

	RefreshTokens     []RefreshToken     `gorm:"foreignKey:UserID" json:"-"`
	BlacklistedTokens []BlacklistedToken `gorm:"foreignKey:UserID" json:"-"`
	AuditLogs         []AuditLog         `gorm:"foreignKey:UserID" json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}

	// Set timestamps if not already set (for tests)
	now := time.Now()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}

	return u.Validate()
}

func (u *User) BeforeUpdate(tx *gorm.DB) error {
	// Skip validation if this is a map-based update (Updates with map)
	// In this case, the User struct is empty and only specific fields are being updated
	if tx.Statement.Dest != nil {
		// Check if the destination is a map (bulk update)
		if _, ok := tx.Statement.Dest.(map[string]interface{}); ok {
			return nil
		}
	}

	// For model-based updates, perform full validation
	return u.Validate()
}

func (u *User) Validate() error {
	if u.Email == "" {
		return errors.New("email is required")
	}

	if !emailRegex.MatchString(u.Email) {
		return errors.New("invalid email format")
	}

	if u.FirstName == "" {
		return errors.New("first name is required")
	}

	if u.LastName == "" {
		return errors.New("last name is required")
	}

	if u.Role != RoleCustomer && u.Role != RoleAdmin {
		return fmt.Errorf("invalid role: %s", u.Role)
	}

	return nil
}

func (u *User) IsLocked() bool {
	return u.LockedAt != nil
}

func (u *User) Lock() {
	now := time.Now()
	u.LockedAt = &now
	u.FailedLoginAttempts = MaxFailedLoginAttempts
}

func (u *User) Unlock() {
	u.LockedAt = nil
	u.FailedLoginAttempts = 0
}

func (u *User) IncrementFailedAttempts() {
	u.FailedLoginAttempts++
	if u.FailedLoginAttempts >= MaxFailedLoginAttempts {
		u.Lock()
	}
}

func (u *User) ResetFailedAttempts() {
	u.FailedLoginAttempts = 0
}

func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLoginAt = &now
}

func (u *User) FullName() string {
	return fmt.Sprintf("%s %s", u.FirstName, u.LastName)
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func (u *User) IsCustomer() bool {
	return u.Role == RoleCustomer
}

func (u *User) TableName() string {
	return "users"
}
