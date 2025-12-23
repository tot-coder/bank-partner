package services

import (
	"errors"
	"fmt"
	"strings"

	"array-assessment/internal/models"

	"array-assessment/internal/repositories"
)

const (
	DefaultSearchLimit = 10
	MaxSearchLimit     = 1000
)

var (
	ErrInvalidSearchQuery = errors.New("invalid search query: query cannot be empty")
	ErrInvalidSearchType  = errors.New("invalid search type")
)

// CustomerSearchService handles customer search operations
type CustomerSearchService struct {
	userRepo repositories.UserRepositoryInterface
}

// NewCustomerSearchService creates a new customer search service
func NewCustomerSearchService(userRepo repositories.UserRepositoryInterface) CustomerSearchServiceInterface {
	return &CustomerSearchService{
		userRepo: userRepo,
	}
}

// ValidateSearchType validates the search type
func ValidateSearchType(searchType models.SearchType) error {
	validTypes := map[models.SearchType]bool{
		models.SearchTypeFirstName:     true,
		models.SearchTypeLastName:      true,
		models.SearchTypeName:          true,
		models.SearchTypeEmail:         true,
		models.SearchTypeAccountNumber: true,
	}

	if !validTypes[searchType] {
		return ErrInvalidSearchType
	}
	return nil
}

// SearchCustomers searches for customers based on the query and search type
// Performs case-insensitive exact match searches
func (s *CustomerSearchService) SearchCustomers(query string, searchType models.SearchType, offset, limit int) ([]*models.CustomerSearchResult, int64, error) {
	if strings.TrimSpace(query) == "" {
		return nil, 0, ErrInvalidSearchQuery
	}

	if err := ValidateSearchType(searchType); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}

	if offset < 0 {
		offset = 0
	}

	criteria := repositories.UserSearchCriteria{
		Query:      query,
		SearchType: string(searchType),
	}

	users, total, err := s.userRepo.SearchUsers(criteria, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search customers: %w", err)
	}

	results := make([]*models.CustomerSearchResult, 0, len(users))
	for _, user := range users {
		accountCount, err := s.userRepo.CountAccountsByUserID(user.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to count accounts for user %s: %w", user.ID, err)
		}

		result := &models.CustomerSearchResult{
			ID:           user.ID,
			Email:        user.Email,
			FirstName:    user.FirstName,
			LastName:     user.LastName,
			Role:         user.Role,
			AccountCount: accountCount,
			CreatedAt:    user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		if user.LastLoginAt != nil {
			lastLoginStr := user.LastLoginAt.Format("2006-01-02T15:04:05Z07:00")
			result.LastLoginAt = &lastLoginStr
		}

		results = append(results, result)
	}

	return results, total, nil
}
