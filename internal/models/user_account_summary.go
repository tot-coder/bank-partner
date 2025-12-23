package models

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type UserAccountSummary struct {
	UserID       uuid.UUID            `json:"user_id"`
	TotalBalance decimal.Decimal      `json:"total_balance"`
	AccountCount int                  `json:"account_count"`
	Currency     string               `json:"currency"`
	Accounts     []AccountSummaryItem `json:"accounts"`
	GeneratedAt  string               `json:"generated_at"`
}
