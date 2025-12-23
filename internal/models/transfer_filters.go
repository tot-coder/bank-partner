package models

import "github.com/google/uuid"

// TransferFilters contains filter criteria for transfer queries
type TransferFilters struct {
	Status        string
	FromAccountID *uuid.UUID
	ToAccountID   *uuid.UUID
	MinAmount     *string
	MaxAmount     *string
}
