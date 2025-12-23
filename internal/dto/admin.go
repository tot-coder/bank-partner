package dto

import (
	"time"

	"array-assessment/internal/models"

	"github.com/google/uuid"
)

// Admin Request DTOs

// UnlockUserRequest represents a request to unlock a user account
type UnlockUserRequest struct {
	UserID string `json:"userId" validate:"required,uuid"`
}

// ListUsersRequest represents query parameters for listing users
type ListUsersRequest struct {
	Offset int `query:"offset" validate:"min=0"`
	Limit  int `query:"limit" validate:"min=1,max=100"`
}

// Admin Response DTOs

// UserResponse represents a user in admin API responses
type UserResponse struct {
	ID                  uuid.UUID  `json:"id"`
	Email               string     `json:"email"`
	FirstName           string     `json:"firstName"`
	LastName            string     `json:"lastName"`
	Role                string     `json:"role"`
	FailedLoginAttempts int        `json:"failedLoginAttempts"`
	LockedAt            *time.Time `json:"lockedAt,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

// UsersListResponse represents a paginated list of users
type UsersListResponse struct {
	Users  []*models.User `json:"users"`
	Total  int64          `json:"total"`
	Offset int            `json:"offset"`
	Limit  int            `json:"limit"`
}

// AuditLogResponse represents an audit log entry
type AuditLogResponse struct {
	ID         uuid.UUID       `json:"id"`
	UserID     *uuid.UUID      `json:"userId,omitempty"`
	Action     string          `json:"action"`
	Resource   string          `json:"resource"`
	ResourceID string          `json:"resourceId"`
	IPAddress  string          `json:"ipAddress"`
	UserAgent  string          `json:"userAgent"`
	Metadata   models.JSONBMap `json:"metadata,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
}

// AuditLogsListResponse represents a paginated list of audit logs
type AuditLogsListResponse struct {
	Logs   []*models.AuditLog `json:"logs"`
	Total  int64              `json:"total"`
	Offset int                `json:"offset"`
	Limit  int                `json:"limit"`
}
