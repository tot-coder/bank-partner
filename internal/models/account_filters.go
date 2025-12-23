package models

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccountFilters contains filter criteria for account queries
type AccountFilters struct {
	UserID      *uuid.UUID
	Status      string
	AccountType string
	MinBalance  *decimal.Decimal
	MaxBalance  *decimal.Decimal
}
