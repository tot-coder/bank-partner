package repositories

import (
	"fmt"
	"testing"

	"array-assessment/internal/database"
	"array-assessment/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

func TestUserRepository(t *testing.T) {
	suite.Run(t, new(UserRepositorySuite))
}

type UserRepositorySuite struct {
	suite.Suite
	db   *database.DB
	repo UserRepositoryInterface
}

func (s *UserRepositorySuite) SetupTest() {
	s.db = database.SetupTestDB(s.T())
	s.repo = NewUserRepository(s.db.DB)
}

func (s *UserRepositorySuite) TearDownTest() {
	database.CleanupTestDB(s.T(), s.db)
}

func (s *UserRepositorySuite) TestUserRepository_Create() {
	user := &models.User{
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		FirstName:    "Test",
		LastName:     "User",
		Role:         models.RoleCustomer,
	}

	err := s.repo.Create(user)
	s.NoError(err)
	s.NotEqual(uuid.Nil, user.ID)
	s.NotZero(user.CreatedAt)
	s.NotZero(user.UpdatedAt)
}

func (s *UserRepositorySuite) TestUserRepository_GetByEmail() {
	// Create test user
	user := &models.User{
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		FirstName:    "Test",
		LastName:     "User",
		Role:         models.RoleCustomer,
	}
	err := s.repo.Create(user)
	s.NoError(err)

	// Test getting existing user
	foundUser, err := s.repo.GetByEmail("test@example.com")
	s.NoError(err)
	s.Equal(user.ID, foundUser.ID)
	s.Equal(user.Email, foundUser.Email)

	// Test getting non-existent user
	_, err = s.repo.GetByEmail("nonexistent@example.com")
	s.Equal(ErrUserNotFound, err)
}

func (s *UserRepositorySuite) TestUserRepository_Update() {
	// Create test user
	user := &models.User{
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		FirstName:    "Test",
		LastName:     "User",
		Role:         models.RoleCustomer,
	}
	err := s.repo.Create(user)
	s.NoError(err)

	// Update user
	user.FirstName = "Updated"
	user.FailedLoginAttempts = 2
	err = s.repo.Update(user)
	s.NoError(err)

	// Verify update
	updatedUser, err := s.repo.GetByID(user.ID)
	s.NoError(err)
	s.Equal("Updated", updatedUser.FirstName)
	s.Equal(2, updatedUser.FailedLoginAttempts)
}

func (s *UserRepositorySuite) TestUserRepository_UnlockAccount() {
	// Create locked user
	user := &models.User{
		Email:               "locked@example.com",
		PasswordHash:        "hashed_password",
		FirstName:           "Test",
		LastName:            "User",
		Role:                models.RoleCustomer,
		FailedLoginAttempts: 3,
	}
	err := s.repo.Create(user)
	s.NoError(err)

	// Unlock account
	err = s.repo.UnlockAccount(user.ID)
	s.NoError(err)

	// Verify unlock
	unlockedUser, err := s.repo.GetByID(user.ID)
	s.NoError(err)
	s.Equal(0, unlockedUser.FailedLoginAttempts)
	s.Nil(unlockedUser.LockedAt)
}

func (s *UserRepositorySuite) TestUserRepository_Delete() {
	// Create test user
	user := &models.User{
		Email:        "delete@example.com",
		PasswordHash: "hashed_password",
		FirstName:    "Test",
		LastName:     "User",
		Role:         models.RoleCustomer,
	}
	err := s.repo.Create(user)
	s.NoError(err)

	// Delete user
	err = s.repo.Delete(user.ID)
	s.NoError(err)

	// Verify user is soft deleted (not found by normal query)
	_, err = s.repo.GetByID(user.ID)
	s.Equal(ErrUserNotFound, err)
}

func (s *UserRepositorySuite) TestUserRepository_ListUsers() {
	// Create test users
	for i := 0; i < 5; i++ {
		user := &models.User{
			Email:        fmt.Sprintf("user%d@example.com", i),
			PasswordHash: "hashed_password",
			FirstName:    "Test",
			LastName:     fmt.Sprintf("User%d", i),
			Role:         models.RoleCustomer,
		}
		err := s.repo.Create(user)
		s.NoError(err)
	}

	// Test pagination
	users, total, err := s.repo.ListUsers(0, 3)
	s.NoError(err)
	s.Equal(int64(5), total)
	s.Len(users, 3)

	// Test second page
	users, total, err = s.repo.ListUsers(3, 3)
	s.NoError(err)
	s.Equal(int64(5), total)
	s.Len(users, 2)
}

func (s *UserRepositorySuite) TestUserRepository_GetByIDActive() {
	// Create test user
	user := &models.User{
		Email:        "active@example.com",
		PasswordHash: "hashed_password",
		FirstName:    "Active",
		LastName:     "User",
		Role:         models.RoleCustomer,
	}
	err := s.repo.Create(user)
	s.NoError(err)

	// Test getting active user
	foundUser, err := s.repo.GetByIDActive(user.ID)
	s.NoError(err)
	s.Equal(user.ID, foundUser.ID)
	s.Equal(user.Email, foundUser.Email)

	// Soft delete the user
	err = s.repo.Delete(user.ID)
	s.NoError(err)

	// Test getting deleted user (should fail)
	_, err = s.repo.GetByIDActive(user.ID)
	s.Equal(ErrUserNotFound, err)

	// Test getting non-existent user
	_, err = s.repo.GetByIDActive(uuid.New())
	s.Equal(ErrUserNotFound, err)
}

func (s *UserRepositorySuite) TestUserRepository_UpdatePasswordHash() {
	// Create test user
	user := &models.User{
		Email:        "password@example.com",
		PasswordHash: "old_hash",
		FirstName:    "Test",
		LastName:     "User",
		Role:         models.RoleCustomer,
	}
	err := s.repo.Create(user)
	s.NoError(err)

	// Update password hash
	newHash := "new_hash_value"
	err = s.repo.UpdatePasswordHash(user.ID, newHash)
	s.NoError(err)

	// Verify update
	updatedUser, err := s.repo.GetByID(user.ID)
	s.NoError(err)
	s.Equal(newHash, updatedUser.PasswordHash)

	// Test with nil UUID
	err = s.repo.UpdatePasswordHash(uuid.Nil, "hash")
	s.Error(err)
	s.Contains(err.Error(), "user ID cannot be nil")

	// Test with empty hash
	err = s.repo.UpdatePasswordHash(user.ID, "")
	s.Error(err)
	s.Contains(err.Error(), "password hash cannot be empty")

	// Test with non-existent user
	err = s.repo.UpdatePasswordHash(uuid.New(), "new_hash")
	s.Equal(ErrUserNotFound, err)
}
