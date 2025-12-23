package models

import "github.com/google/uuid"

// CustomerSearchResult represents the result of a customer search
type CustomerSearchResult struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Role         string    `json:"role"`
	AccountCount int64     `json:"account_count"`
	LastLoginAt  *string   `json:"last_login_at,omitempty"`
	CreatedAt    string    `json:"created_at"`
}
