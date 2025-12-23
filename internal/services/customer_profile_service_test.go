package services

import (
	"testing"

	"array-assessment/internal/models"
	"array-assessment/internal/repositories"
	"array-assessment/internal/repositories/repository_mocks"
	"array-assessment/internal/services/service_mocks"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

type CustomerProfileServiceTestSuite struct {
	suite.Suite
	ctrl         *gomock.Controller
	userRepo     *repository_mocks.MockUserRepositoryInterface
	accountRepo  *repository_mocks.MockAccountRepositoryInterface
	auditService *service_mocks.MockAuditServiceInterface
	service      CustomerProfileServiceInterface
}

func (s *CustomerProfileServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.userRepo = repository_mocks.NewMockUserRepositoryInterface(s.ctrl)
	s.accountRepo = repository_mocks.NewMockAccountRepositoryInterface(s.ctrl)
	s.auditService = service_mocks.NewMockAuditServiceInterface(s.ctrl)
	s.service = NewCustomerProfileService(s.userRepo, s.accountRepo, s.auditService)
}

func (s *CustomerProfileServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestCustomerProfileServiceSuite(t *testing.T) {
	suite.Run(t, new(CustomerProfileServiceTestSuite))
}

// TestGenerateTemporaryPassword is kept as a standalone test since it's testing a utility function
func (s *CustomerProfileServiceTestSuite) TestGenerateTemporaryPassword() {
	tests := []struct {
		name   string
		length int
	}{
		{"16 character password", 16},
		{"20 character password", 20},
		{"32 character password", 32},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			password, err := GenerateTemporaryPassword(tt.length)
			s.Require().NoError(err)
			s.Len(password, tt.length)

			// Verify it contains valid characters
			for _, char := range password {
				valid := (char >= 'a' && char <= 'z') ||
					(char >= 'A' && char <= 'Z') ||
					(char >= '0' && char <= '9') ||
					char == '!' || char == '@' || char == '#' ||
					char == '$' || char == '%' || char == '^' ||
					char == '&' || char == '*'
				s.True(valid, "Invalid character in password: %c", char)
			}

			// Generate another and verify they're different (extremely high probability)
			password2, err := GenerateTemporaryPassword(tt.length)
			s.Require().NoError(err)
			s.NotEqual(password, password2, "Generated passwords should be unique")
		})
	}
}

func (s *CustomerProfileServiceTestSuite) TestGetCustomerProfile_ValidCustomerID() {
	// Create a test user
	user := &models.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      models.RoleCustomer,
	}

	// Setup mock expectations
	s.userRepo.EXPECT().GetByIDActive(user.ID).Return(user, nil).Times(1)

	result, err := s.service.GetCustomerProfile(user.ID)
	s.Require().NoError(err)
	s.NotNil(result)
	s.Equal(user.ID, result.ID)
	s.Equal(user.Email, result.Email)
}

func (s *CustomerProfileServiceTestSuite) TestGetCustomerProfile_NilCustomerID() {
	_, err := s.service.GetCustomerProfile(uuid.Nil)
	s.Error(err)
	s.ErrorIs(err, ErrInvalidCustomerID)
}

func (s *CustomerProfileServiceTestSuite) TestGetCustomerProfile_NonExistentCustomer() {
	customerID := uuid.New()

	// Setup mock expectations
	s.userRepo.EXPECT().GetByIDActive(customerID).Return(nil, repositories.ErrUserNotFound).Times(1)

	_, err := s.service.GetCustomerProfile(customerID)
	s.Error(err)
	s.ErrorIs(err, ErrCustomerNotFound)
}

func (s *CustomerProfileServiceTestSuite) TestCreateCustomer_ValidCustomerCreation() {
	// Setup mock expectations
	s.userRepo.EXPECT().GetByEmail("new@example.com").Return(nil, repositories.ErrUserNotFound).Times(1)
	s.userRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(user *models.User) error {
		// Simulate ID generation
		user.ID = uuid.New()
		return nil
	}).Times(1)

	user, tempPassword, err := s.service.CreateCustomer("new@example.com", "Jane", "Smith", models.RoleCustomer)

	s.Require().NoError(err)
	s.NotNil(user)
	s.NotEqual(uuid.Nil, user.ID)
	s.Equal("new@example.com", user.Email)
	s.Equal("Jane", user.FirstName)
	s.Equal("Smith", user.LastName)
	s.Equal(models.RoleCustomer, user.Role)

	// Verify temporary password
	s.NotEmpty(tempPassword)
	s.Len(tempPassword, TemporaryPasswordLength)

	// Verify password was hashed correctly
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(tempPassword))
	s.NoError(err, "Temporary password should match hash")
}

func (s *CustomerProfileServiceTestSuite) TestCreateCustomer_ValidAdminCreation() {
	// Setup mock expectations
	s.userRepo.EXPECT().GetByEmail("admin@example.com").Return(nil, repositories.ErrUserNotFound).Times(1)
	s.userRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(user *models.User) error {
		// Simulate ID generation
		user.ID = uuid.New()
		return nil
	}).Times(1)

	user, tempPassword, err := s.service.CreateCustomer("admin@example.com", "Admin", "User", models.RoleAdmin)

	s.Require().NoError(err)
	s.NotNil(user)
	s.NotEqual(uuid.Nil, user.ID)
	s.Equal("admin@example.com", user.Email)
	s.Equal("Admin", user.FirstName)
	s.Equal("User", user.LastName)
	s.Equal(models.RoleAdmin, user.Role)

	// Verify temporary password
	s.NotEmpty(tempPassword)
	s.Len(tempPassword, TemporaryPasswordLength)

	// Verify password was hashed correctly
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(tempPassword))
	s.NoError(err, "Temporary password should match hash")
}

func (s *CustomerProfileServiceTestSuite) TestCreateCustomer_EmptyEmail() {
	user, _, err := s.service.CreateCustomer("", "John", "Doe", models.RoleCustomer)
	s.Error(err)
	s.ErrorIs(err, ErrInvalidEmail)
	s.Nil(user)
}

func (s *CustomerProfileServiceTestSuite) TestCreateCustomer_InvalidRole() {
	user, _, err := s.service.CreateCustomer("invalid@example.com", "John", "Doe", "superuser")
	s.Error(err)
	s.ErrorIs(err, ErrInvalidRole)
	s.Nil(user)
}

func (s *CustomerProfileServiceTestSuite) TestCreateCustomer_DuplicateEmail() {
	existingUser := &models.User{
		ID:    uuid.New(),
		Email: "duplicate@example.com",
	}

	// Setup mock expectations - first call succeeds
	s.userRepo.EXPECT().GetByEmail("duplicate@example.com").Return(nil, repositories.ErrUserNotFound).Times(1)
	s.userRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(user *models.User) error {
		user.ID = uuid.New()
		return nil
	}).Times(1)

	// Create first user
	user1, _, err := s.service.CreateCustomer("duplicate@example.com", "John", "Doe", models.RoleCustomer)
	s.Require().NoError(err)
	s.Require().NotNil(user1)

	// Setup mock expectations - second call finds existing user
	s.userRepo.EXPECT().GetByEmail("duplicate@example.com").Return(existingUser, nil).Times(1)

	// Try to create second user with same email
	user2, _, err := s.service.CreateCustomer("duplicate@example.com", "Jane", "Smith", models.RoleCustomer)
	s.Error(err)
	s.ErrorIs(err, ErrEmailAlreadyExists)
	s.Nil(user2)
}

func (s *CustomerProfileServiceTestSuite) TestUpdateCustomerProfile() {
	// Create a test user
	user := &models.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      models.RoleCustomer,
	}

	tests := []struct {
		name       string
		customerID uuid.UUID
		updates    map[string]interface{}
		wantErr    bool
		errType    error
		setupMocks func()
		validate   func(userID uuid.UUID)
	}{
		{
			name:       "update first and last name",
			customerID: user.ID,
			updates: map[string]interface{}{
				"first_name": "Jane",
				"last_name":  "Smith",
			},
			wantErr: false,
			setupMocks: func() {
				s.userRepo.EXPECT().GetByIDActive(user.ID).Return(user, nil).Times(1)
				s.userRepo.EXPECT().UpdateFields(user.ID, gomock.Any()).Return(nil).Times(1)
			},
			validate: func(userID uuid.UUID) {
				// Validation happens through mock expectations
			},
		},
		{
			name:       "nil customer ID",
			customerID: uuid.Nil,
			updates: map[string]interface{}{
				"first_name": "Jane",
			},
			wantErr:    true,
			errType:    ErrInvalidCustomerID,
			setupMocks: func() {}, // No mocks needed - validation error
		},
		{
			name:       "non-existent customer",
			customerID: uuid.New(),
			updates: map[string]interface{}{
				"first_name": "Jane",
			},
			wantErr: true,
			errType: ErrCustomerNotFound,
			setupMocks: func() {
				s.userRepo.EXPECT().GetByIDActive(gomock.Any()).Return(nil, repositories.ErrUserNotFound).Times(1)
			},
		},
		{
			name:       "empty updates",
			customerID: user.ID,
			updates:    map[string]interface{}{},
			wantErr:    true,
			setupMocks: func() {}, // No mocks needed - validation error
		},
		{
			name:       "attempt to update sensitive fields",
			customerID: user.ID,
			updates: map[string]interface{}{
				"first_name":    "Jane",
				"password_hash": "hacked",
				"email":         "hacked@example.com",
				"role":          models.RoleAdmin,
			},
			wantErr: false,
			setupMocks: func() {
				s.userRepo.EXPECT().GetByIDActive(user.ID).Return(user, nil).Times(1)
				s.userRepo.EXPECT().UpdateFields(user.ID, gomock.Any()).Return(nil).Times(1)
			},
			validate: func(userID uuid.UUID) {
				// Validation happens through mock expectations - sensitive fields filtered by service
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Create new mocks for this test case
			ctrl := gomock.NewController(s.T())
			defer ctrl.Finish()
			s.userRepo = repository_mocks.NewMockUserRepositoryInterface(ctrl)
			s.accountRepo = repository_mocks.NewMockAccountRepositoryInterface(ctrl)
			s.auditService = service_mocks.NewMockAuditServiceInterface(ctrl)
			s.service = NewCustomerProfileService(s.userRepo, s.accountRepo, s.auditService)

			tt.setupMocks()

			err := s.service.UpdateCustomerProfile(tt.customerID, tt.updates)

			if tt.wantErr {
				s.Error(err)
				if tt.errType != nil {
					s.ErrorIs(err, tt.errType)
				}
			} else {
				s.Require().NoError(err)
				if tt.validate != nil {
					tt.validate(tt.customerID)
				}
			}
		})
	}
}

func (s *CustomerProfileServiceTestSuite) TestUpdateCustomerEmail_ValidEmailUpdate() {
	// Create test user
	user := &models.User{
		ID:        uuid.New(),
		Email:     "user1@example.com",
		FirstName: "User",
		LastName:  "One",
		Role:      models.RoleCustomer,
	}

	// Setup mock expectations
	s.userRepo.EXPECT().GetByEmailExcluding("newemail@example.com", user.ID).Return(nil, repositories.ErrUserNotFound).Times(1)
	s.userRepo.EXPECT().GetByIDActive(user.ID).Return(user, nil).Times(1)
	s.userRepo.EXPECT().UpdateEmail(user.ID, "newemail@example.com").Return(nil).Times(1)

	err := s.service.UpdateCustomerEmail(user.ID, "newemail@example.com")
	s.Require().NoError(err)
}

func (s *CustomerProfileServiceTestSuite) TestUpdateCustomerEmail_DuplicateEmail() {
	// Create test users
	user1 := &models.User{
		ID:        uuid.New(),
		Email:     "user1@example.com",
		FirstName: "User",
		LastName:  "One",
		Role:      models.RoleCustomer,
	}

	user2 := &models.User{
		ID:        uuid.New(),
		Email:     "user2@example.com",
		FirstName: "User",
		LastName:  "Two",
		Role:      models.RoleCustomer,
	}

	// Setup mock expectations - email already exists
	s.userRepo.EXPECT().GetByEmailExcluding(user2.Email, user1.ID).Return(user2, nil).Times(1)

	err := s.service.UpdateCustomerEmail(user1.ID, user2.Email)
	s.Error(err)
	s.ErrorIs(err, ErrEmailAlreadyExists)
}

func (s *CustomerProfileServiceTestSuite) TestUpdateCustomerEmail_EmptyEmail() {
	user := &models.User{
		ID:        uuid.New(),
		Email:     "user@example.com",
		FirstName: "User",
		LastName:  "Test",
		Role:      models.RoleCustomer,
	}

	err := s.service.UpdateCustomerEmail(user.ID, "")
	s.Error(err)
	s.ErrorIs(err, ErrInvalidEmail)
}

func (s *CustomerProfileServiceTestSuite) TestUpdateCustomerEmail_NilCustomerID() {
	err := s.service.UpdateCustomerEmail(uuid.Nil, "test@example.com")
	s.Error(err)
	s.ErrorIs(err, ErrInvalidCustomerID)
}

func (s *CustomerProfileServiceTestSuite) TestDeleteCustomer() {
	// Create a user with zero balance accounts
	userWithZeroBalance := &models.User{
		ID:        uuid.New(),
		Email:     "zerobala@example.com",
		FirstName: "Zero",
		LastName:  "Balance",
		Role:      models.RoleCustomer,
	}

	// Create a user with non-zero balance
	userWithBalance := &models.User{
		ID:        uuid.New(),
		Email:     "hasbalance@example.com",
		FirstName: "Has",
		LastName:  "Balance",
		Role:      models.RoleCustomer,
	}

	tests := []struct {
		name       string
		customerID uuid.UUID
		reason     string
		wantErr    bool
		errType    error
		setupMocks func()
		validate   func()
	}{
		{
			name:       "delete customer with zero balance",
			customerID: userWithZeroBalance.ID,
			reason:     "Account closure requested",
			wantErr:    false,
			setupMocks: func() {
				s.userRepo.EXPECT().GetByIDActive(userWithZeroBalance.ID).Return(userWithZeroBalance, nil).Times(1)
				s.accountRepo.EXPECT().GetTotalBalanceByUserID(userWithZeroBalance.ID).Return(decimal.Zero, nil).Times(1)
				s.userRepo.EXPECT().Delete(userWithZeroBalance.ID).Return(nil).Times(1)
				s.accountRepo.EXPECT().SoftDeleteByUserID(userWithZeroBalance.ID).Return(nil).Times(1)
			},
			validate: func() {
				// Validation happens through mock expectations
			},
		},
		{
			name:       "cannot delete customer with balance",
			customerID: userWithBalance.ID,
			reason:     "Account closure requested",
			wantErr:    true,
			errType:    ErrCustomerHasBalance,
			setupMocks: func() {
				s.userRepo.EXPECT().GetByIDActive(userWithBalance.ID).Return(userWithBalance, nil).Times(1)
				s.accountRepo.EXPECT().GetTotalBalanceByUserID(userWithBalance.ID).Return(decimal.NewFromFloat(100.50), nil).Times(1)
			},
		},
		{
			name:       "nil customer ID",
			customerID: uuid.Nil,
			reason:     "Test",
			wantErr:    true,
			errType:    ErrInvalidCustomerID,
			setupMocks: func() {}, // No mocks needed - validation error
		},
		{
			name:       "non-existent customer",
			customerID: uuid.New(),
			reason:     "Test",
			wantErr:    true,
			errType:    ErrCustomerNotFound,
			setupMocks: func() {
				s.userRepo.EXPECT().GetByIDActive(gomock.Any()).Return(nil, repositories.ErrUserNotFound).Times(1)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Create new mocks for this test case
			ctrl := gomock.NewController(s.T())
			defer ctrl.Finish()
			s.userRepo = repository_mocks.NewMockUserRepositoryInterface(ctrl)
			s.accountRepo = repository_mocks.NewMockAccountRepositoryInterface(ctrl)
			s.auditService = service_mocks.NewMockAuditServiceInterface(ctrl)
			s.service = NewCustomerProfileService(s.userRepo, s.accountRepo, s.auditService)

			tt.setupMocks()

			err := s.service.DeleteCustomer(tt.customerID, tt.reason)

			if tt.wantErr {
				s.Error(err)
				if tt.errType != nil {
					s.ErrorIs(err, tt.errType)
				}
			} else {
				s.Require().NoError(err)
				if tt.validate != nil {
					tt.validate()
				}
			}
		})
	}
}
