package repositories

import (
	"errors"
	"fmt"
	"strings"

	"array-assessment/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrEmailAlreadyExists = errors.New("email already exists")
)

// UserRepository handles database operations for users
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) UserRepositoryInterface {
	return &UserRepository{
		db: db,
	}
}

// Create creates a new user in the database
func (r *UserRepository) Create(user *models.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	if err := r.db.Create(user).Error; err != nil {
		if isDuplicateKeyError(err) {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by their ID
func (r *UserRepository) GetByID(id uuid.UUID) (*models.User, error) {
	user := &models.User{ID: id}
	if err := r.db.First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return user, nil
}

// GetByIDActive retrieves an active (non-deleted) user by their ID
func (r *UserRepository) GetByIDActive(id uuid.UUID) (*models.User, error) {
	user := &models.User{ID: id}
	if err := r.db.First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get active user by ID: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by their email address
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User

	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// Update updates a user in the database
func (r *UserRepository) Update(user *models.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	if err := r.db.Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdatePasswordHash atomically updates a user's password hash
func (r *UserRepository) UpdatePasswordHash(userID uuid.UUID, passwordHash string) error {
	if userID == uuid.Nil {
		return errors.New("user ID cannot be nil")
	}

	if passwordHash == "" {
		return errors.New("password hash cannot be empty")
	}

	result := r.db.Model(&models.User{ID: userID}).Update("password_hash", passwordHash)
	if result.Error != nil {
		return fmt.Errorf("failed to update password hash: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateFailedLoginAttempts updates the failed login attempts and locked status
func (r *UserRepository) UpdateFailedLoginAttempts(user *models.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	updates := map[string]interface{}{
		"failed_login_attempts": user.FailedLoginAttempts,
		"locked_at":             user.LockedAt,
	}

	if err := r.db.Model(user).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update login attempts: %w", err)
	}

	return nil
}

// ResetFailedLoginAttempts resets the failed login counter for a user
func (r *UserRepository) ResetFailedLoginAttempts(userID uuid.UUID) error {
	updates := map[string]interface{}{
		"failed_login_attempts": 0,
		"locked_at":             nil,
	}

	if err := r.db.Model(&models.User{ID: userID}).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to reset login attempts: %w", err)
	}

	return nil
}

// UnlockAccount unlocks a user account
func (r *UserRepository) UnlockAccount(userID uuid.UUID) error {
	return r.ResetFailedLoginAttempts(userID)
}

// Delete soft deletes a user
func (r *UserRepository) Delete(userID uuid.UUID) error {
	result := r.db.Delete(&models.User{ID: userID})
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// ListUsers lists users with pagination
func (r *UserRepository) ListUsers(offset, limit int) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	if err := r.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	if err := r.db.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// Postgres duplicate key error detection
	return strings.Contains(errStr, "duplicate key") ||
		strings.Contains(errStr, "UNIQUE constraint") ||
		strings.Contains(errStr, "23505")
}

// GetByEmailExcluding retrieves a user by email, excluding a specific user ID
func (r *UserRepository) GetByEmailExcluding(email string, excludeUserID uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ? AND id != ?", email, excludeUserID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &user, nil
}

// UpdateFields updates specific fields of a user
func (r *UserRepository) UpdateFields(userID uuid.UUID, fields map[string]interface{}) error {
	result := r.db.Model(&models.User{ID: userID}).
		Updates(fields)

	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			return ErrEmailAlreadyExists
		}
		return fmt.Errorf("failed to update user fields: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdateEmail updates a user's email address
func (r *UserRepository) UpdateEmail(userID uuid.UUID, newEmail string) error {
	result := r.db.Model(&models.User{ID: userID}).
		Update("email", newEmail)

	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			return ErrEmailAlreadyExists
		}
		return fmt.Errorf("failed to update email: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// CountAccountsByUserID counts the number of active accounts for a user
func (r *UserRepository) CountAccountsByUserID(userID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.Model(&models.Account{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count accounts: %w", err)
	}
	return count, nil
}

// SearchUsers searches for users based on criteria
func (r *UserRepository) SearchUsers(criteria UserSearchCriteria, offset, limit int) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	baseQuery := r.db.Model(&models.User{})

	// Apply search filter based on type
	switch criteria.SearchType {
	case "first_name":
		baseQuery = baseQuery.Where("LOWER(first_name) = LOWER(?)", criteria.Query)
	case "last_name":
		baseQuery = baseQuery.Where("LOWER(last_name) = LOWER(?)", criteria.Query)
	case "name":
		// Search in both first and last name
		baseQuery = baseQuery.Where("LOWER(first_name) = LOWER(?) OR LOWER(last_name) = LOWER(?)", criteria.Query, criteria.Query)
	case "email":
		baseQuery = baseQuery.Where("LOWER(email) = LOWER(?)", criteria.Query)
	case "account_number":
		// Join with accounts table to search by account number
		baseQuery = baseQuery.Joins("INNER JOIN accounts ON accounts.user_id = users.id AND accounts.deleted_at IS NULL").
			Where("accounts.account_number = ?", criteria.Query).
			Distinct()
	default:
		return nil, 0, fmt.Errorf("invalid search type: %s", criteria.SearchType)
	}

	// Count total results
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count search results: %w", err)
	}

	// Get paginated results
	if err := baseQuery.Order("last_name ASC, first_name ASC").
		Offset(offset).
		Limit(limit).
		Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to search users: %w", err)
	}

	return users, total, nil
}
