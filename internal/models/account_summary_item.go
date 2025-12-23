package models

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AccountSummaryItem struct {
	ID                  uuid.UUID       `json:"id"`
	MaskedAccountNumber string          `json:"masked_account_number"`
	AccountNumber       string          `json:"-"`
	AccountType         string          `json:"account_type"`
	Balance             decimal.Decimal `json:"balance"`
	Status              string          `json:"status"`
	Currency            string          `json:"currency"`
	InterestRate        decimal.Decimal `json:"interest_rate,omitempty"`
	CreatedAt           string          `json:"created_at"`
}
