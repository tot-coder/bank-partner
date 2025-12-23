package models

// SearchType defines the type of search to perform
type SearchType string

const (
	SearchTypeFirstName     SearchType = "first_name"
	SearchTypeLastName      SearchType = "last_name"
	SearchTypeName          SearchType = "name"
	SearchTypeEmail         SearchType = "email"
	SearchTypeAccountNumber SearchType = "account_number"
)
