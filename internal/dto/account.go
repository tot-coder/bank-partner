package dto

import (
	"array-assessment/internal/models"

	"github.com/shopspring/decimal"
)

// Account Request DTOs

// CreateAccountRequest represents the request payload for creating a new account
type CreateAccountRequest struct {
	AccountType       string `json:"account_type" validate:"required,oneof=CHECKING SAVINGS MONEY_MARKET"`
	AccountNumber     string `json:"account_number" validate:"required"`
	RoutingNumber     string `json:"routing_number" validate:"required"`
	AccountHolderName string `json:"account_holder_name" validate:"required,min=1,max=100"`
}

// UpdateAccountStatusRequest represents the request payload for updating account status
type UpdateAccountStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=active inactive frozen closed"`
}

// TransactionRequest represents the request payload for performing a transaction
type TransactionRequest struct {
	Amount      string `json:"amount" validate:"required"`
	Type        string `json:"type" validate:"required,oneof=credit debit"`
	Description string `json:"description" validate:"required,min=1,max=255"`
}

// TransferRequest represents the request payload for transferring funds between accounts
type TransferRequest struct {
	ToAccountID string `json:"toAccountId" validate:"required,uuid"`
	Amount      string `json:"amount" validate:"required"`
	Description string `json:"description" validate:"required,min=1,max=255"`
}

// Account Response DTOs

// CreateAccountResponse represents the response after creating an account
type CreateAccountResponse struct {
	Account *models.Account `json:"account"`
	Message string          `json:"message"`
}

// AccountResponse represents a single account in API responses
type AccountResponse struct {
	*models.Account
}

// AccountListResponse represents a paginated list of accounts
type AccountListResponse struct {
	Accounts []models.Account `json:"accounts"`
	Total    int64            `json:"total"`
	Offset   int              `json:"offset"`
	Limit    int              `json:"limit"`
}

// TransactionListResponse represents a paginated list of transactions
type TransactionListResponse struct {
	Transactions []models.Transaction `json:"transactions"`
	Total        int64                `json:"total"`
	Offset       int                  `json:"offset"`
	Limit        int                  `json:"limit"`
}

// AccountSummaryResponse represents aggregated account information for a user
type AccountSummaryResponse struct {
	TotalBalance       decimal.Decimal             `json:"totalBalance"`
	AccountCount       int                         `json:"accountCount"`
	ActiveAccountCount int                         `json:"activeAccountCount"`
	Accounts           []models.Account            `json:"accounts"`
	AccountsByType     map[string][]models.Account `json:"accountsByType"`
}

// MessageResponse represents a simple message response
type MessageResponse struct {
	Message string `json:"message"`
}

// TransferResponse represents the response after a successful transfer
type TransferResponse struct {
	Message             string  `json:"message"`
	TransferID          string  `json:"transferId"`
	FromAccountID       string  `json:"fromAccountId"`
	ToAccountID         string  `json:"toAccountId"`
	Amount              string  `json:"amount"`
	DebitTransactionID  *string `json:"debitTransactionId,omitempty"`
	CreditTransactionID *string `json:"creditTransactionId,omitempty"`
}

// TransferHistoryResponse represents paginated transfer history
type TransferHistoryResponse struct {
	Transfers  []models.Transfer `json:"transfers"`
	Pagination PaginationMeta    `json:"pagination"`
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}
