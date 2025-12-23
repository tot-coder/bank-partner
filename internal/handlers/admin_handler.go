package handlers

import (
	"net/http"

	"array-assessment/internal/errors"
	"array-assessment/internal/repositories"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// AdminHandler handles admin-related endpoints
type AdminHandler struct {
	userRepo  repositories.UserRepositoryInterface
	auditRepo repositories.AuditLogRepositoryInterface
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(userRepo repositories.UserRepositoryInterface, auditRepo repositories.AuditLogRepositoryInterface) *AdminHandler {
	return &AdminHandler{
		userRepo:  userRepo,
		auditRepo: auditRepo,
	}
}

// UnlockUser unlocks a user account
// @Summary Unlock user account (admin)
// @Description Admin endpoint to unlock a locked user account
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param userId path string true "User ID (UUID)"
// @Success 200 {object} SuccessResponse "User unlocked successfully"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_001 - Invalid user ID"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 404 {object} errors.ErrorResponse "CUSTOMER_001 - User not found"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /admin/users/{userId}/unlock [post]
func (h *AdminHandler) UnlockUser(c echo.Context) error {
	userIDParam := c.Param("userId")

	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		return SendError(c, errors.CustomerInvalidID, errors.WithDetails("User ID must be a valid UUID"))
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		if err == repositories.ErrUserNotFound {
			return SendError(c, errors.CustomerNotFound)
		}
		return SendSystemError(c, err)
	}

	if err := h.userRepo.UnlockAccount(userID); err != nil {
		return SendSystemError(c, err)
	}

	adminID := c.Get("user_id").(uuid.UUID)
	h.createAuditLog(adminID, "admin_unlock_user", user.ID.String(), c)

	return c.JSON(http.StatusOK, SuccessResponse{
		Message: "User account unlocked successfully",
		Data: map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email,
		},
	})
}

// ListUsers lists all users with pagination
// @Summary List all users (admin)
// @Description Admin endpoint to list all users with pagination
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page (max 100)" default(20)
// @Success 200 {object} SuccessResponse "Users retrieved successfully with pagination metadata"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_001 - Invalid pagination parameters"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /admin/users [get]
func (h *AdminHandler) ListUsers(c echo.Context) error {
	page := getIntParam(c, "page", 1)
	limit := getIntParam(c, "limit", 20)

	if page < 1 {
		return SendError(c, errors.ValidationGeneral,
			errors.WithDetails("page: must be greater than 0"))
	}
	if limit < 1 || limit > 100 {
		return SendError(c, errors.ValidationGeneral,
			errors.WithDetails("limit: must be between 1 and 100"))
	}

	offset := (page - 1) * limit

	users, total, err := h.userRepo.ListUsers(offset, limit)
	if err != nil {
		return SendSystemError(c, err)
	}

	sanitizedUsers := make([]map[string]interface{}, len(users))
	for i, user := range users {
		sanitizedUsers[i] = map[string]interface{}{
			"id":         user.ID,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"role":       user.Role,
			"is_locked":  user.IsLocked(),
			"created_at": user.CreatedAt,
		}
	}

	return c.JSON(http.StatusOK, SuccessResponse{
		Data: sanitizedUsers,
		Meta: map[string]interface{}{
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetUserByID retrieves a specific user by ID
// @Summary Get user by ID (admin)
// @Description Admin endpoint to retrieve detailed user information
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param userId path string true "User ID (UUID)"
// @Success 200 {object} SuccessResponse "User retrieved successfully"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_001 - Invalid user ID"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 404 {object} errors.ErrorResponse "CUSTOMER_001 - User not found"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /admin/users/{userId} [get]
func (h *AdminHandler) GetUserByID(c echo.Context) error {
	userIDParam := c.Param("userId")

	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		return SendError(c, errors.CustomerInvalidID, errors.WithDetails("User ID must be a valid UUID"))
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		if err == repositories.ErrUserNotFound {
			return SendError(c, errors.CustomerNotFound)
		}
		return SendSystemError(c, err)
	}

	return c.JSON(http.StatusOK, SuccessResponse{
		Data: map[string]interface{}{
			"id":                    user.ID,
			"email":                 user.Email,
			"first_name":            user.FirstName,
			"last_name":             user.LastName,
			"role":                  user.Role,
			"is_locked":             user.IsLocked(),
			"failed_login_attempts": user.FailedLoginAttempts,
			"locked_at":             user.LockedAt,
			"created_at":            user.CreatedAt,
			"updated_at":            user.UpdatedAt,
		},
	})
}

// DeleteUser soft deletes a user
// @Summary Delete user (admin)
// @Description Admin endpoint to soft delete a user. Cannot delete own account.
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param userId path string true "User ID (UUID)"
// @Success 200 {object} SuccessResponse "User deleted successfully"
// @Failure 400 {object} errors.ErrorResponse "VALIDATION_001 - Invalid user ID or cannot delete own account"
// @Failure 401 {object} errors.ErrorResponse "AUTH_002 - Missing or invalid authentication"
// @Failure 403 {object} errors.ErrorResponse "AUTH_005 - Requires admin role"
// @Failure 404 {object} errors.ErrorResponse "CUSTOMER_001 - User not found"
// @Failure 500 {object} errors.ErrorResponse "SYSTEM_001 - Internal server error"
// @Router /admin/users/{userId} [delete]
func (h *AdminHandler) DeleteUser(c echo.Context) error {
	userIDParam := c.Param("userId")

	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		return SendError(c, errors.CustomerInvalidID, errors.WithDetails("User ID must be a valid UUID"))
	}

	adminID := c.Get("user_id").(uuid.UUID)
	if adminID == userID {
		return SendError(c, errors.ValidationGeneral, errors.WithDetails("Cannot delete your own account"))
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		if err == repositories.ErrUserNotFound {
			return SendError(c, errors.CustomerNotFound)
		}
		return SendSystemError(c, err)
	}

	if err := h.userRepo.Delete(userID); err != nil {
		return SendSystemError(c, err)
	}

	h.createAuditLog(adminID, "admin_delete_user", user.ID.String(), c)

	return c.JSON(http.StatusOK, SuccessResponse{
		Message: "User deleted successfully",
	})
}
