package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccountMetrics represents performance metrics for a single account
type AccountMetrics struct {
	AccountID                uuid.UUID       `json:"account_id"`
	StartDate                time.Time       `json:"start_date"`
	EndDate                  time.Time       `json:"end_date"`
	TotalDeposits            decimal.Decimal `json:"total_deposits"`
	TotalWithdrawals         decimal.Decimal `json:"total_withdrawals"`
	NetChange                decimal.Decimal `json:"net_change"`
	TransactionCount         int64           `json:"transaction_count"`
	DepositCount             int64           `json:"deposit_count"`
	WithdrawalCount          int64           `json:"withdrawal_count"`
	AverageTransactionAmount decimal.Decimal `json:"average_transaction_amount"`
	LargestDeposit           decimal.Decimal `json:"largest_deposit"`
	LargestWithdrawal        decimal.Decimal `json:"largest_withdrawal"`
	AverageDailyBalance      decimal.Decimal `json:"average_daily_balance"`
	InterestEarned           decimal.Decimal `json:"interest_earned"`
	GeneratedAt              time.Time       `json:"generated_at"`
}

// UserAggregateMetrics represents aggregate metrics across all user accounts
type UserAggregateMetrics struct {
	UserID                uuid.UUID          `json:"user_id"`
	StartDate             time.Time          `json:"start_date"`
	EndDate               time.Time          `json:"end_date"`
	TotalDeposits         decimal.Decimal    `json:"total_deposits"`
	TotalWithdrawals      decimal.Decimal    `json:"total_withdrawals"`
	NetChange             decimal.Decimal    `json:"net_change"`
	TotalTransactionCount int64              `json:"total_transaction_count"`
	AccountCount          int                `json:"account_count"`
	AccountMetrics        []AccountMetrics   `json:"account_metrics"`
	GeneratedAt           time.Time          `json:"generated_at"`
}
