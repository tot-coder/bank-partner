package dto

import (
	"time"

	"array-assessment/internal/models"

	"github.com/google/uuid"
)

// SearchCustomersRequest represents the request to search for customers
type SearchCustomersRequest struct {
	Query  string `query:"q" validate:"required,min=1"`
	Limit  int    `query:"limit" validate:"omitempty,min=1,max=1000"`
	Offset int    `query:"offset" validate:"omitempty,min=0"`
}

// SearchCustomersResponse represents the response from customer search
type SearchCustomersResponse struct {
	Customers  []*CustomerSearchResult `json:"customers"`
	Total      int64                   `json:"total"`
	Limit      int                     `json:"limit"`
	Offset     int                     `json:"offset"`
	TotalPages int                     `json:"total_pages"`
}

// CustomerSearchResult represents a single customer in search results
type CustomerSearchResult struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	FirstName    string     `json:"firstName"`
	LastName     string     `json:"lastName"`
	Status       string     `json:"status"`
	AccountCount int        `json:"accountCount"`
	LastLoginAt  *time.Time `json:"lastLoginAt"`
	CreatedAt    time.Time  `json:"createAt"`
}

// GetCustomerProfileResponse represents the detailed customer profile
type GetCustomerProfileResponse struct {
	Customer *models.User `json:"customer"`
}

// CreateCustomerRequest represents the request to create a new customer
type CreateCustomerRequest struct {
	Email            string `json:"email" validate:"required,email"`
	FirstName        string `json:"firstName" validate:"required,min=1,max=100"`
	LastName         string `json:"lastName" validate:"required,min=1,max=100"`
	PhoneNumber      string `json:"phoneNumber" validate:"omitempty,e164"`
	DateOfBirth      string `json:"dateOfBirth" validate:"required"`
	Address          string `json:"address" validate:"omitempty,max=500"`
	City             string `json:"city" validate:"omitempty,max=100"`
	State            string `json:"state" validate:"omitempty,len=2"`
	ZipCode          string `json:"zipCode" validate:"omitempty,len=5"`
	SSN              string `json:"ssn" validate:"required,len=9"`
	EmploymentStatus string `json:"employmentStatus" validate:"required,oneof=employed self_employed unemployed retired student"`
	AnnualIncome     string `json:"annualIncome" validate:"required"`
}

// CreateCustomerResponse represents the response after creating a customer
type CreateCustomerResponse struct {
	Customer          *models.User `json:"customer"`
	TemporaryPassword string       `json:"temporaryPassword"`
	Message           string       `json:"message"`
}

// UpdateCustomerProfileRequest represents the request to update customer profile
type UpdateCustomerProfileRequest struct {
	FirstName   *string `json:"firstName" validate:"omitempty,min=1,max=100"`
	LastName    *string `json:"lastName" validate:"omitempty,min=1,max=100"`
	PhoneNumber *string `json:"phoneNumber" validate:"omitempty,e164"`
	Address     *string `json:"address" validate:"omitempty,max=500"`
	City        *string `json:"city" validate:"omitempty,max=100"`
	State       *string `json:"state" validate:"omitempty,len=2"`
	ZipCode     *string `json:"zipCode" validate:"omitempty,len=5"`
}

// UpdateCustomerEmailRequest represents the request to update customer email
type UpdateCustomerEmailRequest struct {
	NewEmail string `json:"newEmail" validate:"required,email"`
}

// UpdateCustomerEmailResponse represents the response after updating email
type UpdateCustomerEmailResponse struct {
	Message string `json:"message"`
}

// DeleteCustomerResponse represents the response after deleting a customer
type DeleteCustomerResponse struct {
	Message string `json:"message"`
}
