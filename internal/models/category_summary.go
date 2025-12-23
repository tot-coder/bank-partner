package models

import "github.com/shopspring/decimal"

// CategorySummary contains aggregated transaction data by category
type CategorySummary struct {
	Category         string          `json:"category"`
	TransactionCount int64           `json:"transaction_count"`
	TotalAmount      decimal.Decimal `json:"total_amount"`
	AverageAmount    decimal.Decimal `json:"average_amount"`
}
