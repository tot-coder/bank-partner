package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccountStatement represents a complete account statement for a period
type AccountStatement struct {
	AccountID          uuid.UUID              `json:"account_id"`
	AccountNumber      string                 `json:"account_number"`
	AccountType        string                 `json:"account_type"`
	PeriodType         string                 `json:"period_type"`
	Year               int                    `json:"year"`
	Period             int                    `json:"period"`
	StartDate          time.Time              `json:"start_date"`
	EndDate            time.Time              `json:"end_date"`
	OpeningBalance     decimal.Decimal        `json:"opening_balance"`
	ClosingBalance     decimal.Decimal        `json:"closing_balance"`
	Transactions       []StatementTransaction `json:"transactions"`
	PerformanceMetrics *AccountMetrics        `json:"performance_metrics"`
	Summary            StatementSummary       `json:"summary"`
	GeneratedAt        time.Time              `json:"generated_at"`
}

// StatementTransaction represents a transaction in the statement with running balance
type StatementTransaction struct {
	ID              uuid.UUID       `json:"id"`
	Date            time.Time       `json:"date"`
	Description     string          `json:"description"`
	TransactionType string          `json:"transaction_type"`
	Amount          decimal.Decimal `json:"amount"`
	RunningBalance  decimal.Decimal `json:"running_balance"`
	Reference       string          `json:"reference"`
	Status          string          `json:"status"`
}

// StatementSummary provides aggregate information for the statement period
type StatementSummary struct {
	TotalDeposits    decimal.Decimal `json:"total_deposits"`
	TotalWithdrawals decimal.Decimal `json:"total_withdrawals"`
	NetChange        decimal.Decimal `json:"net_change"`
	TransactionCount int             `json:"transaction_count"`
	DepositCount     int             `json:"deposit_count"`
	WithdrawalCount  int             `json:"withdrawal_count"`
}
