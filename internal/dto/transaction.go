package dto

import (
	"time"

	"github.com/google/uuid"
)

// TransactionFilters contains filtering options for transaction queries
type TransactionFilters struct {
	StartDate *time.Time `query:"startDate"`
	EndDate   *time.Time `query:"endDate"`
	Type      string     `query:"type"`
	Status    string     `query:"status"`
	Category  string     `query:"category"`
}

// PaginationParams contains pagination parameters
type PaginationParams struct {
	Cursor string `query:"cursor"`
	Limit  int    `query:"limit"`
}

// TransactionWithBalance represents a transaction with its running balance
type TransactionWithBalance struct {
	ID              uuid.UUID  `json:"id"`
	AccountID       uuid.UUID  `json:"accountId"`
	Amount          string     `json:"amount"`
	TransactionType string     `json:"transactionType"`
	Description     string     `json:"description"`
	Status          string     `json:"status"`
	Category        string     `json:"category,omitempty"`
	MerchantName    string     `json:"merchantName,omitempty"`
	MCCCode         string     `json:"mccCode,omitempty"`
	Reference       string     `json:"reference,omitempty"`
	RunningBalance  string     `json:"runningBalance"`
	CreatedAt       time.Time  `json:"createdAt"`
	ProcessedAt     *time.Time `json:"processedAt,omitempty"`
}

// PaginationInfo contains pagination metadata
type PaginationInfo struct {
	HasMore    bool   `json:"hasMore"`
	NextCursor string `json:"nextCursor,omitempty"`
	Limit      int    `json:"limit"`
	Total      int64  `json:"total,omitempty"`
}

// ListTransactionsResponse represents the response for listing transactions
type ListTransactionsResponse struct {
	Transactions []TransactionWithBalance `json:"transactions"`
	Pagination   PaginationInfo           `json:"pagination"`
}
