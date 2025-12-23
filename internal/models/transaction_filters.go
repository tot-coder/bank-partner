package models

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

// TransactionFilters contains filtering options for transaction queries
type TransactionFilters struct {
	AccountID    uuid.UUID
	StartDate    *time.Time
	EndDate      *time.Time
	Type         string
	Status       string
	Category     string
	MinAmount    *decimal.Decimal
	MaxAmount    *decimal.Decimal
	MerchantName string
	Offset       int
	Limit        int
}
