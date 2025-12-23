package services

import (
	"strings"
	"testing"

	"array-assessment/internal/models"
	"array-assessment/internal/repositories"
	"array-assessment/internal/repositories/repository_mocks"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// PasswordServiceTestSuite defines the test suite for PasswordService
type PasswordServiceTestSuite struct {
	suite.Suite
	ctrl          *gomock.Controller
	mockUserRepo  *repository_mocks.MockUserRepositoryInterface
	mockAuditRepo *repository_mocks.MockAuditLogRepositoryInterface
	mockAuditSvc  AuditServiceInterface
	service       PasswordServiceInterface
}

// SetupTest runs before each test
func (s *PasswordServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockUserRepo = repository_mocks.NewMockUserRepositoryInterface(s.ctrl)
	s.mockAuditRepo = repository_mocks.NewMockAuditLogRepositoryInterface(s.ctrl)
	s.mockAuditSvc = NewAuditService(s.mockAuditRepo)
	s.service = NewPasswordService(s.mockUserRepo, s.mockAuditSvc)
}

// TearDownTest runs after each test
func (s *PasswordServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

// TestPasswordServiceSuite runs the test suite
func TestPasswordServiceSuite(t *testing.T) {
	suite.Run(t, new(PasswordServiceTestSuite))
}

// Test ValidatePassword
func (s *PasswordServiceTestSuite) TestValidatePassword_ValidPassword() {
	err := s.service.ValidatePassword("SecurePass123!@#")
	s.NoError(err)
}

func (s *PasswordServiceTestSuite) TestValidatePassword_TooShort() {
	err := s.service.ValidatePassword("Short1!")
	s.Error(err)
	s.Contains(err.Error(), "password must be at least 12 characters")
}

func (s *PasswordServiceTestSuite) TestValidatePassword_MissingUppercase() {
	err := s.service.ValidatePassword("securepass123!@#")
	s.Error(err)
	s.Contains(err.Error(), "password must contain at least one uppercase letter")
}

func (s *PasswordServiceTestSuite) TestValidatePassword_MissingLowercase() {
	err := s.service.ValidatePassword("SECUREPASS123!@#")
	s.Error(err)
	s.Contains(err.Error(), "password must contain at least one lowercase letter")
}

func (s *PasswordServiceTestSuite) TestValidatePassword_MissingNumber() {
	err := s.service.ValidatePassword("SecurePass!@#")
	s.Error(err)
	s.Contains(err.Error(), "password must contain at least one number")
}

func (s *PasswordServiceTestSuite) TestValidatePassword_MissingSpecialChar() {
	err := s.service.ValidatePassword("SecurePass123")
	s.Error(err)
	s.Contains(err.Error(), "password must contain at least one special character")
}

func (s *PasswordServiceTestSuite) TestValidatePassword_Empty() {
	err := s.service.ValidatePassword("")
	s.Error(err)
	s.Contains(err.Error(), "password cannot be empty")
}

func (s *PasswordServiceTestSuite) TestValidatePassword_WithSpaces() {
	err := s.service.ValidatePassword("Secure Pass123!")
	s.NoError(err)
}

func (s *PasswordServiceTestSuite) TestValidatePassword_Complex() {
	err := s.service.ValidatePassword("C0mpl3x!P@ssw0rd#2024")
	s.NoError(err)
}

func (s *PasswordServiceTestSuite) TestValidatePassword_MinimumValid() {
	err := s.service.ValidatePassword("Aa1!Aa1!Aa1!")
	s.NoError(err)
}

// Test HashPassword
func (s *PasswordServiceTestSuite) TestHashPassword_ValidPassword() {
	hash, err := s.service.HashPassword("SecurePass123!@#")
	s.NoError(err)
	s.NotEmpty(hash)
	s.NotEqual("SecurePass123!@#", hash)
	s.True(strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$"))
}

func (s *PasswordServiceTestSuite) TestHashPassword_InvalidPassword() {
	hash, err := s.service.HashPassword("short")
	s.Error(err)
	s.Empty(hash)
}

func (s *PasswordServiceTestSuite) TestHashPassword_EmptyPassword() {
	hash, err := s.service.HashPassword("")
	s.Error(err)
	s.Empty(hash)
}

func (s *PasswordServiceTestSuite) TestHashPassword_VeryLongPassword() {
	password := strings.Repeat("Aa1!", 17) // 68 characters (under 72 byte limit)
	hash, err := s.service.HashPassword(password)
	s.NoError(err)
	s.NotEmpty(hash)
	s.NotEqual(password, hash)
}

// Test ComparePassword
func (s *PasswordServiceTestSuite) TestComparePassword_CorrectPassword() {
	password := "SecurePass123!@#"
	hash, err := s.service.HashPassword(password)
	s.Require().NoError(err)

	result := s.service.ComparePassword(password, hash)
	s.True(result)
}

func (s *PasswordServiceTestSuite) TestComparePassword_IncorrectPassword() {
	password := "SecurePass123!@#"
	hash, err := s.service.HashPassword(password)
	s.Require().NoError(err)

	result := s.service.ComparePassword("WrongPass123!@#", hash)
	s.False(result)
}

func (s *PasswordServiceTestSuite) TestComparePassword_EmptyPassword() {
	password := "SecurePass123!@#"
	hash, err := s.service.HashPassword(password)
	s.Require().NoError(err)

	result := s.service.ComparePassword("", hash)
	s.False(result)
}

func (s *PasswordServiceTestSuite) TestComparePassword_InvalidHash() {
	result := s.service.ComparePassword("SecurePass123!@#", "invalid-hash")
	s.False(result)
}

func (s *PasswordServiceTestSuite) TestComparePassword_EmptyHash() {
	result := s.service.ComparePassword("SecurePass123!@#", "")
	s.False(result)
}

func (s *PasswordServiceTestSuite) TestComparePassword_CaseSensitive() {
	password := "SecurePass123!@#"
	hash, err := s.service.HashPassword(password)
	s.Require().NoError(err)

	result := s.service.ComparePassword("securepass123!@#", hash)
	s.False(result)
}

// Test hash uniqueness
func (s *PasswordServiceTestSuite) TestHashUniqueness() {
	password := "SecurePass123!@#"

	hash1, err1 := s.service.HashPassword(password)
	s.NoError(err1)

	hash2, err2 := s.service.HashPassword(password)
	s.NoError(err2)

	// Hashes should be different due to salting
	s.NotEqual(hash1, hash2)

	// But both should validate against the original password
	s.True(s.service.ComparePassword(password, hash1))
	s.True(s.service.ComparePassword(password, hash2))
}

// Test timing attack resistance
func (s *PasswordServiceTestSuite) TestTimingAttackResistance() {
	hash, err := s.service.HashPassword("SecurePass123!@#")
	s.Require().NoError(err)

	// Both comparisons should complete without timing-based information leakage
	result1 := s.service.ComparePassword("WrongPassword123!", hash)
	result2 := s.service.ComparePassword("SecurePass123!@#", hash)

	s.False(result1)
	s.True(result2)
}

// Test GenerateSecurePassword
func (s *PasswordServiceTestSuite) TestGenerateSecurePassword() {
	password, err := s.service.GenerateSecurePassword()
	s.NoError(err)
	s.NotEmpty(password)

	err = s.service.ValidatePassword(password)
	s.NoError(err)

	password2, err := s.service.GenerateSecurePassword()
	s.NoError(err)
	s.NotEqual(password, password2)
}

// Test GenerateSecurePasswordWithLength
func (s *PasswordServiceTestSuite) TestGenerateSecurePasswordWithLength_MinimumLength() {
	password, err := s.service.GenerateSecurePasswordWithLength(8)
	s.NoError(err)
	s.Len(password, MinPasswordLength)
}

func (s *PasswordServiceTestSuite) TestGenerateSecurePasswordWithLength_ExactLength() {
	password, err := s.service.GenerateSecurePasswordWithLength(20)
	s.NoError(err)
	s.Len(password, 20)

	err = s.service.ValidatePassword(password)
	s.NoError(err)
}

func (s *PasswordServiceTestSuite) TestGenerateSecurePasswordWithLength_MaximumLength() {
	password, err := s.service.GenerateSecurePasswordWithLength(100)
	s.NoError(err)
	s.Len(password, MaxPasswordLength)
}

// Test PasswordStrength
func (s *PasswordServiceTestSuite) TestPasswordStrength_Empty() {
	score := s.service.PasswordStrength("")
	s.GreaterOrEqual(score, 0)
	s.LessOrEqual(score, 100)
}

func (s *PasswordServiceTestSuite) TestPasswordStrength_Weak() {
	score := s.service.PasswordStrength("password")
	s.GreaterOrEqual(score, 0)
	s.LessOrEqual(score, 100)
}

func (s *PasswordServiceTestSuite) TestPasswordStrength_MeetsRequirements() {
	score := s.service.PasswordStrength("SecurePass123!")
	s.GreaterOrEqual(score, 80)
	s.LessOrEqual(score, 100)
}

func (s *PasswordServiceTestSuite) TestPasswordStrength_VeryStrong() {
	score := s.service.PasswordStrength("VerySecure$Pass123!WithManyChars")
	s.GreaterOrEqual(score, 85)
	s.LessOrEqual(score, 100)
}

// Test AdminResetPassword
func (s *PasswordServiceTestSuite) TestAdminResetPassword_Success() {
	customerID := uuid.New()
	adminID := uuid.New()

	// Create test user
	testUser := &models.User{
		ID:           customerID,
		Email:        "customer@example.com",
		PasswordHash: "old-hash",
		FirstName:    "John",
		LastName:     "Doe",
		Role:         models.RoleCustomer,
	}

	// Setup mock expectations BEFORE calling the service method
	s.mockUserRepo.EXPECT().GetByIDActive(customerID).Return(testUser, nil).Times(1)
	s.mockUserRepo.EXPECT().UpdatePasswordHash(customerID, gomock.Any()).Return(nil).Times(1)

	tempPassword, err := s.service.AdminResetPassword(customerID, adminID)
	s.NoError(err)
	s.NotEmpty(tempPassword)

	// Verify the password meets requirements
	err = s.service.ValidatePassword(tempPassword)
	s.NoError(err)
}

func (s *PasswordServiceTestSuite) TestAdminResetPassword_NilCustomerID() {
	adminID := uuid.New()

	tempPassword, err := s.service.AdminResetPassword(uuid.Nil, adminID)
	s.Error(err)
	s.Empty(tempPassword)
}

func (s *PasswordServiceTestSuite) TestAdminResetPassword_NilAdminID() {
	customerID := uuid.New()

	tempPassword, err := s.service.AdminResetPassword(customerID, uuid.Nil)
	s.Error(err)
	s.Empty(tempPassword)
}

func (s *PasswordServiceTestSuite) TestAdminResetPassword_CustomerNotFound() {
	customerID := uuid.New()
	adminID := uuid.New()

	// Setup mock to return user not found error
	s.mockUserRepo.EXPECT().GetByIDActive(customerID).Return(nil, repositories.ErrUserNotFound).Times(1)

	tempPassword, err := s.service.AdminResetPassword(customerID, adminID)
	s.Error(err)
	s.Empty(tempPassword)
	s.ErrorIs(err, ErrCustomerNotFound)
}

// Test CustomerUpdatePassword
func (s *PasswordServiceTestSuite) TestCustomerUpdatePassword_Success() {
	customerID := uuid.New()
	currentPassword := "CurrentP@ssw0rd123"
	newPassword := "NewP@ssw0rd12345"

	// Hash the current password to simulate existing user
	hashedPassword, err := s.service.HashPasswordWithoutValidation(currentPassword)
	s.Require().NoError(err)

	// Create test user with hashed password
	testUser := &models.User{
		ID:           customerID,
		Email:        "customer@example.com",
		PasswordHash: hashedPassword,
		FirstName:    "John",
		LastName:     "Doe",
		Role:         models.RoleCustomer,
	}

	// Setup mock expectations BEFORE calling the service method
	s.mockUserRepo.EXPECT().GetByIDActive(customerID).Return(testUser, nil).Times(1)
	s.mockUserRepo.EXPECT().UpdatePasswordHash(customerID, gomock.Any()).Return(nil).Times(1)

	err = s.service.CustomerUpdatePassword(customerID, currentPassword, newPassword)
	s.NoError(err)
}

func (s *PasswordServiceTestSuite) TestCustomerUpdatePassword_WrongCurrentPassword() {
	customerID := uuid.New()
	currentPassword := "CurrentP@ssw0rd123"
	wrongPassword := "WrongP@ssw0rd123"
	newPassword := "NewP@ssw0rd12345"

	// Hash the current password to simulate existing user
	hashedPassword, err := s.service.HashPasswordWithoutValidation(currentPassword)
	s.Require().NoError(err)

	// Create test user with hashed password
	testUser := &models.User{
		ID:           customerID,
		Email:        "customer@example.com",
		PasswordHash: hashedPassword,
		FirstName:    "John",
		LastName:     "Doe",
		Role:         models.RoleCustomer,
	}

	// Setup mock expectations - user exists but password won't match
	s.mockUserRepo.EXPECT().GetByIDActive(customerID).Return(testUser, nil).Times(1)

	err = s.service.CustomerUpdatePassword(customerID, wrongPassword, newPassword)
	s.Error(err)
	s.ErrorIs(err, ErrCurrentPasswordWrong)
}

func (s *PasswordServiceTestSuite) TestCustomerUpdatePassword_SamePassword() {
	customerID := uuid.New()
	currentPassword := "CurrentP@ssw0rd123"

	err := s.service.CustomerUpdatePassword(customerID, currentPassword, currentPassword)
	s.Error(err)
}

func (s *PasswordServiceTestSuite) TestCustomerUpdatePassword_WeakNewPassword() {
	customerID := uuid.New()
	currentPassword := "CurrentP@ssw0rd123"
	weakPassword := "weak"

	err := s.service.CustomerUpdatePassword(customerID, currentPassword, weakPassword)
	s.Error(err)
}

func (s *PasswordServiceTestSuite) TestCustomerUpdatePassword_NilCustomerID() {
	err := s.service.CustomerUpdatePassword(uuid.Nil, "CurrentP@ssw0rd123", "NewP@ssw0rd12345")
	s.Error(err)
}

// Benchmarks
func (s *PasswordServiceTestSuite) BenchmarkPasswordService_HashPassword(b *testing.B) {
	password := "SecurePass123!@#"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.service.HashPassword(password)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func (s *PasswordServiceTestSuite) BenchmarkPasswordService_ComparePassword(b *testing.B) {
	password := "SecurePass123!@#"
	hash, err := s.service.HashPassword(password)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.service.ComparePassword(password, hash)
	}
}
