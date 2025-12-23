package services

import (
	"testing"
	"time"

	"array-assessment/internal/models"
	"array-assessment/internal/repositories"
	"array-assessment/internal/repositories/repository_mocks"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

// CustomerSearchServiceTestSuite is the test suite for CustomerSearchService
type CustomerSearchServiceTestSuite struct {
	suite.Suite
	ctrl     *gomock.Controller
	userRepo *repository_mocks.MockUserRepositoryInterface
	service  CustomerSearchServiceInterface
}

func (s *CustomerSearchServiceTestSuite) SetupTest() {
	// Create repository and service
	s.ctrl = gomock.NewController(s.T())
	s.userRepo = repository_mocks.NewMockUserRepositoryInterface(s.ctrl)
	s.service = NewCustomerSearchService(s.userRepo)
}

func (s *CustomerSearchServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestCustomerSearchServiceSuite(t *testing.T) {
	suite.Run(t, new(CustomerSearchServiceTestSuite))
}

// Helper method to create test customer
func (s *CustomerSearchServiceTestSuite) createTestCustomer(firstName, lastName, email string, lastLoginAt *time.Time) *models.User {
	user := &models.User{
		ID:          uuid.New(),
		Email:       email,
		FirstName:   firstName,
		LastName:    lastName,
		Role:        models.RoleCustomer,
		LastLoginAt: lastLoginAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return user
}

// Helper method to create test account
func (s *CustomerSearchServiceTestSuite) createTestAccount(userID uuid.UUID, accountNumber string) *models.Account {
	account := &models.Account{
		ID:            uuid.New(),
		AccountNumber: accountNumber,
		UserID:        userID,
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	return account
}

func (s *CustomerSearchServiceTestSuite) TestNewCustomerSearchService() {
	s.NotNil(s.service)
}

func (s *CustomerSearchServiceTestSuite) TestValidateSearchType_ValidFirstName() {
	err := ValidateSearchType(models.SearchTypeFirstName)
	s.NoError(err)
}

func (s *CustomerSearchServiceTestSuite) TestValidateSearchType_ValidLastName() {
	err := ValidateSearchType(models.SearchTypeLastName)
	s.NoError(err)
}

func (s *CustomerSearchServiceTestSuite) TestValidateSearchType_ValidName() {
	err := ValidateSearchType(models.SearchTypeName)
	s.NoError(err)
}

func (s *CustomerSearchServiceTestSuite) TestValidateSearchType_ValidEmail() {
	err := ValidateSearchType(models.SearchTypeEmail)
	s.NoError(err)
}

func (s *CustomerSearchServiceTestSuite) TestValidateSearchType_ValidAccountNumber() {
	err := ValidateSearchType(models.SearchTypeAccountNumber)
	s.NoError(err)
}

func (s *CustomerSearchServiceTestSuite) TestValidateSearchType_Invalid() {
	err := ValidateSearchType(models.SearchType("invalid"))
	s.Error(err)
	s.ErrorIs(err, ErrInvalidSearchType)
}

func (s *CustomerSearchServiceTestSuite) TestSearchCustomers() {
	// Create test data
	lastLogin := time.Now().Add(-24 * time.Hour)
	john := s.createTestCustomer("John", "Doe", "john.doe@example.com", &lastLogin)
	jane := s.createTestCustomer("Jane", "Smith", "jane.smith@example.com", &lastLogin)
	bob := s.createTestCustomer("Bob", "Johnson", "bob.johnson@example.com", nil)
	alice := s.createTestCustomer("Alice", "Doe", "alice.doe@example.com", &lastLogin)

	// Create accounts for users
	_ = s.createTestAccount(john.ID, "1000000001")
	s.createTestAccount(john.ID, "1000000002") // John has 2 accounts
	_ = s.createTestAccount(jane.ID, "1000000003")
	// Bob has no accounts
	_ = s.createTestAccount(alice.ID, "1000000004")

	tests := []struct {
		name          string
		query         string
		searchType    models.SearchType
		offset        int
		limit         int
		wantErr       bool
		expectedCount int
		validateFunc  func(results []*models.CustomerSearchResult, total int64)
		setupMocks    func()
	}{
		{
			name:          "search by first name - exact match",
			query:         "John",
			searchType:    models.SearchTypeFirstName,
			offset:        0,
			limit:         10,
			wantErr:       false,
			expectedCount: 1,
			validateFunc: func(results []*models.CustomerSearchResult, total int64) {
				s.Equal(int64(1), total)
				s.Equal("John", results[0].FirstName)
				s.Equal(int64(2), results[0].AccountCount) // John has 2 accounts
				s.NotNil(results[0].LastLoginAt)
			},
			setupMocks: func() {
				criteria := repositories.UserSearchCriteria{
					Query:      "John",
					SearchType: string(models.SearchTypeFirstName),
				}
				s.userRepo.EXPECT().SearchUsers(criteria, 0, 10).Return([]*models.User{john}, int64(1), nil).Times(1)
				s.userRepo.EXPECT().CountAccountsByUserID(john.ID).Return(int64(2), nil).Times(1)
			},
		},
		{
			name:          "search by first name - case insensitive",
			query:         "jOhN",
			searchType:    models.SearchTypeFirstName,
			offset:        0,
			limit:         10,
			wantErr:       false,
			expectedCount: 1,
			validateFunc: func(results []*models.CustomerSearchResult, total int64) {
				s.Equal(int64(1), total)
				s.Equal("John", results[0].FirstName)
			},
			setupMocks: func() {
				criteria := repositories.UserSearchCriteria{
					Query:      "jOhN",
					SearchType: string(models.SearchTypeFirstName),
				}
				s.userRepo.EXPECT().SearchUsers(criteria, 0, 10).Return([]*models.User{john}, int64(1), nil).Times(1)
				s.userRepo.EXPECT().CountAccountsByUserID(john.ID).Return(int64(2), nil).Times(1)
			},
		},
		{
			name:          "search by last name",
			query:         "Doe",
			searchType:    models.SearchTypeLastName,
			offset:        0,
			limit:         10,
			wantErr:       false,
			expectedCount: 2,
			validateFunc: func(results []*models.CustomerSearchResult, total int64) {
				s.Equal(int64(2), total)
				// Results should be ordered by last_name, first_name
				s.Equal("Doe", results[0].LastName)
				s.Equal("Doe", results[1].LastName)
			},
			setupMocks: func() {
				criteria := repositories.UserSearchCriteria{
					Query:      "Doe",
					SearchType: string(models.SearchTypeLastName),
				}
				s.userRepo.EXPECT().SearchUsers(criteria, 0, 10).Return([]*models.User{john, alice}, int64(2), nil).Times(1)
				s.userRepo.EXPECT().CountAccountsByUserID(john.ID).Return(int64(2), nil).Times(1)
				s.userRepo.EXPECT().CountAccountsByUserID(alice.ID).Return(int64(1), nil).Times(1)
			},
		},
		{
			name:          "search by name - matches first or last",
			query:         "Johnson",
			searchType:    models.SearchTypeName,
			offset:        0,
			limit:         10,
			wantErr:       false,
			expectedCount: 1,
			validateFunc: func(results []*models.CustomerSearchResult, total int64) {
				s.Equal(int64(1), total)
				s.Equal("Bob", results[0].FirstName)
				s.Equal("Johnson", results[0].LastName)
				s.Equal(int64(0), results[0].AccountCount) // Bob has no accounts
				s.Nil(results[0].LastLoginAt)              // Bob never logged in
			},
			setupMocks: func() {
				criteria := repositories.UserSearchCriteria{
					Query:      "Johnson",
					SearchType: string(models.SearchTypeName),
				}
				s.userRepo.EXPECT().SearchUsers(criteria, 0, 10).Return([]*models.User{bob}, int64(1), nil).Times(1)
				s.userRepo.EXPECT().CountAccountsByUserID(bob.ID).Return(int64(0), nil).Times(1)
			},
		},
		{
			name:          "search by email",
			query:         "jane.smith@example.com",
			searchType:    models.SearchTypeEmail,
			offset:        0,
			limit:         10,
			wantErr:       false,
			expectedCount: 1,
			validateFunc: func(results []*models.CustomerSearchResult, total int64) {
				s.Equal(int64(1), total)
				s.Equal("Jane", results[0].FirstName)
				s.Equal("Smith", results[0].LastName)
				s.Equal(int64(1), results[0].AccountCount)
			},
			setupMocks: func() {
				criteria := repositories.UserSearchCriteria{
					Query:      "jane.smith@example.com",
					SearchType: string(models.SearchTypeEmail),
				}
				s.userRepo.EXPECT().SearchUsers(criteria, 0, 10).Return([]*models.User{jane}, int64(1), nil).Times(1)
				s.userRepo.EXPECT().CountAccountsByUserID(jane.ID).Return(int64(1), nil).Times(1)
			},
		},
		{
			name:          "search by email - case insensitive",
			query:         "JANE.SMITH@EXAMPLE.COM",
			searchType:    models.SearchTypeEmail,
			offset:        0,
			limit:         10,
			wantErr:       false,
			expectedCount: 1,
			validateFunc: func(results []*models.CustomerSearchResult, total int64) {
				s.Equal(int64(1), total)
				s.Equal("jane.smith@example.com", results[0].Email)
			},
			setupMocks: func() {
				criteria := repositories.UserSearchCriteria{
					Query:      "JANE.SMITH@EXAMPLE.COM",
					SearchType: string(models.SearchTypeEmail),
				}
				s.userRepo.EXPECT().SearchUsers(criteria, 0, 10).Return([]*models.User{jane}, int64(1), nil).Times(1)
				s.userRepo.EXPECT().CountAccountsByUserID(jane.ID).Return(int64(1), nil).Times(1)
			},
		},
		{
			name:          "search by account number",
			query:         "1000000001",
			searchType:    models.SearchTypeAccountNumber,
			offset:        0,
			limit:         10,
			wantErr:       false,
			expectedCount: 1,
			validateFunc: func(results []*models.CustomerSearchResult, total int64) {
				s.Equal(int64(1), total)
				s.Equal(john.ID, results[0].ID)
				s.Equal("John", results[0].FirstName)
			},
			setupMocks: func() {
				criteria := repositories.UserSearchCriteria{
					Query:      "1000000001",
					SearchType: string(models.SearchTypeAccountNumber),
				}
				s.userRepo.EXPECT().SearchUsers(criteria, 0, 10).Return([]*models.User{john}, int64(1), nil).Times(1)
				s.userRepo.EXPECT().CountAccountsByUserID(john.ID).Return(int64(2), nil).Times(1)
			},
		},
		{
			name:          "pagination - limit",
			query:         "Doe",
			searchType:    models.SearchTypeLastName,
			offset:        0,
			limit:         1,
			wantErr:       false,
			expectedCount: 1,
			validateFunc: func(results []*models.CustomerSearchResult, total int64) {
				s.Equal(int64(2), total) // Total is 2 but we only get 1 result
			},
			setupMocks: func() {
				criteria := repositories.UserSearchCriteria{
					Query:      "Doe",
					SearchType: string(models.SearchTypeLastName),
				}
				s.userRepo.EXPECT().SearchUsers(criteria, 0, 1).Return([]*models.User{john}, int64(2), nil).Times(1)
				s.userRepo.EXPECT().CountAccountsByUserID(john.ID).Return(int64(2), nil).Times(1)
			},
		},
		{
			name:          "pagination - offset",
			query:         "Doe",
			searchType:    models.SearchTypeLastName,
			offset:        1,
			limit:         10,
			wantErr:       false,
			expectedCount: 1,
			validateFunc: func(results []*models.CustomerSearchResult, total int64) {
				s.Equal(int64(2), total)
			},
			setupMocks: func() {
				criteria := repositories.UserSearchCriteria{
					Query:      "Doe",
					SearchType: string(models.SearchTypeLastName),
				}
				s.userRepo.EXPECT().SearchUsers(criteria, 1, 10).Return([]*models.User{alice}, int64(2), nil).Times(1)
				s.userRepo.EXPECT().CountAccountsByUserID(alice.ID).Return(int64(1), nil).Times(1)
			},
		},
		{
			name:          "empty query",
			query:         "",
			searchType:    models.SearchTypeFirstName,
			offset:        0,
			limit:         10,
			wantErr:       true,
			expectedCount: 0,
			setupMocks:    func() {}, // No mocks - validation error
		},
		{
			name:          "invalid search type",
			query:         "John",
			searchType:    models.SearchType("invalid"),
			offset:        0,
			limit:         10,
			wantErr:       true,
			expectedCount: 0,
			setupMocks:    func() {}, // No mocks - validation error
		},
		{
			name:          "no results",
			query:         "NonExistent",
			searchType:    models.SearchTypeFirstName,
			offset:        0,
			limit:         10,
			wantErr:       false,
			expectedCount: 0,
			validateFunc: func(results []*models.CustomerSearchResult, total int64) {
				s.Equal(int64(0), total)
			},
			setupMocks: func() {
				criteria := repositories.UserSearchCriteria{
					Query:      "NonExistent",
					SearchType: string(models.SearchTypeFirstName),
				}
				s.userRepo.EXPECT().SearchUsers(criteria, 0, 10).Return([]*models.User{}, int64(0), nil).Times(1)
			},
		},
		{
			name:          "default limit when zero",
			query:         "Doe",
			searchType:    models.SearchTypeLastName,
			offset:        0,
			limit:         0, // Should default to 10
			wantErr:       false,
			expectedCount: 2,
			validateFunc: func(results []*models.CustomerSearchResult, total int64) {
				s.Equal(int64(2), total)
			},
			setupMocks: func() {
				criteria := repositories.UserSearchCriteria{
					Query:      "Doe",
					SearchType: string(models.SearchTypeLastName),
				}
				// Service will convert limit 0 to DefaultSearchLimit (10)
				s.userRepo.EXPECT().SearchUsers(criteria, 0, 10).Return([]*models.User{john, alice}, int64(2), nil).Times(1)
				s.userRepo.EXPECT().CountAccountsByUserID(john.ID).Return(int64(2), nil).Times(1)
				s.userRepo.EXPECT().CountAccountsByUserID(alice.ID).Return(int64(1), nil).Times(1)
			},
		},
		{
			name:          "max limit when exceeded",
			query:         "Doe",
			searchType:    models.SearchTypeLastName,
			offset:        0,
			limit:         2000, // Should be capped at 1000
			wantErr:       false,
			expectedCount: 2,
			validateFunc: func(results []*models.CustomerSearchResult, total int64) {
				s.Equal(int64(2), total)
			},
			setupMocks: func() {
				criteria := repositories.UserSearchCriteria{
					Query:      "Doe",
					SearchType: string(models.SearchTypeLastName),
				}
				// Service will cap limit at MaxSearchLimit (1000)
				s.userRepo.EXPECT().SearchUsers(criteria, 0, 1000).Return([]*models.User{john, alice}, int64(2), nil).Times(1)
				s.userRepo.EXPECT().CountAccountsByUserID(john.ID).Return(int64(2), nil).Times(1)
				s.userRepo.EXPECT().CountAccountsByUserID(alice.ID).Return(int64(1), nil).Times(1)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Setup mocks for this test case
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			results, total, err := s.service.SearchCustomers(tt.query, tt.searchType, tt.offset, tt.limit)

			if tt.wantErr {
				s.Error(err)
			} else {
				s.Require().NoError(err)
				s.Len(results, tt.expectedCount)

				if tt.validateFunc != nil {
					tt.validateFunc(results, total)
				}

				// Verify all results have required fields
				for _, result := range results {
					s.NotEqual(uuid.Nil, result.ID)
					s.NotEmpty(result.Email)
					s.NotEmpty(result.FirstName)
					s.NotEmpty(result.LastName)
					s.NotEmpty(result.Role)
					s.GreaterOrEqual(result.AccountCount, int64(0))
					s.NotEmpty(result.CreatedAt)
				}
			}
		})
	}
}

func (s *CustomerSearchServiceTestSuite) TestExcludesDeletedUsers() {
	// Create a normal user
	john := s.createTestCustomer("John", "Doe", "john.doe@example.com", nil)

	// Create a deleted user (repository would exclude this)

	// Setup mocks
	criteria := repositories.UserSearchCriteria{
		Query:      "Doe",
		SearchType: string(models.SearchTypeLastName),
	}
	// Repository excludes deleted users, so only john is returned
	s.userRepo.EXPECT().SearchUsers(criteria, 0, 10).Return([]*models.User{john}, int64(1), nil).Times(1)
	s.userRepo.EXPECT().CountAccountsByUserID(john.ID).Return(int64(0), nil).Times(1)

	// Search should only return the non-deleted user
	results, total, err := s.service.SearchCustomers("Doe", models.SearchTypeLastName, 0, 10)
	s.Require().NoError(err)
	s.Equal(int64(1), total)
	s.Len(results, 1)
	s.Equal(john.ID, results[0].ID)
}

func (s *CustomerSearchServiceTestSuite) TestExcludesDeletedAccounts() {
	// Create a user with one active and one deleted account
	john := s.createTestCustomer("John", "Doe", "john.doe@example.com", nil)
	_ = s.createTestAccount(john.ID, "1000000001")

	// Soft delete one account (would happen at repository level)

	// Setup mocks for first search (by last name)
	criteria1 := repositories.UserSearchCriteria{
		Query:      "Doe",
		SearchType: string(models.SearchTypeLastName),
	}
	s.userRepo.EXPECT().SearchUsers(criteria1, 0, 10).Return([]*models.User{john}, int64(1), nil).Times(1)
	// CountAccountsByUserID excludes deleted accounts, so only 1
	s.userRepo.EXPECT().CountAccountsByUserID(john.ID).Return(int64(1), nil).Times(1)

	// Search should only count the active account
	results, total, err := s.service.SearchCustomers("Doe", models.SearchTypeLastName, 0, 10)
	s.Require().NoError(err)
	s.Equal(int64(1), total)
	s.Len(results, 1)
	s.Equal(int64(1), results[0].AccountCount)

	// Setup mocks for second search (by deleted account number)
	criteria2 := repositories.UserSearchCriteria{
		Query:      "1000000002",
		SearchType: string(models.SearchTypeAccountNumber),
	}
	// Repository excludes deleted accounts, so no results
	s.userRepo.EXPECT().SearchUsers(criteria2, 0, 10).Return([]*models.User{}, int64(0), nil).Times(1)

	// Search by deleted account number should return nothing
	results, total, err = s.service.SearchCustomers("1000000002", models.SearchTypeAccountNumber, 0, 10)
	s.Require().NoError(err)
	s.Equal(int64(0), total)
	s.Len(results, 0)

	// Setup mocks for third search (by active account number)
	criteria3 := repositories.UserSearchCriteria{
		Query:      "1000000001",
		SearchType: string(models.SearchTypeAccountNumber),
	}
	s.userRepo.EXPECT().SearchUsers(criteria3, 0, 10).Return([]*models.User{john}, int64(1), nil).Times(1)
	s.userRepo.EXPECT().CountAccountsByUserID(john.ID).Return(int64(1), nil).Times(1)

	// Search by active account number should work
	results, total, err = s.service.SearchCustomers("1000000001", models.SearchTypeAccountNumber, 0, 10)
	s.Require().NoError(err)
	s.Equal(int64(1), total)
	s.Len(results, 1)
}
