package services

import (
	"errors"
	"fmt"
	"time"

	"array-assessment/internal/models"
	"array-assessment/internal/repositories"

	"github.com/google/uuid"
)

// AuditService handles audit logging operations
type AuditService struct {
	repo repositories.AuditLogRepositoryInterface
}

// NewAuditService creates a new audit service
func NewAuditService(repo repositories.AuditLogRepositoryInterface) AuditServiceInterface {
	return &AuditService{
		repo: repo,
	}
}

var (
	ErrInvalidUserID   = errors.New("invalid user ID")
	ErrInvalidAuditLog = errors.New("invalid audit log")
	ErrAuditDateRange  = errors.New("invalid date range: start date must be before end date")
)

// ValidateActivityType validates that the activity type is one of the allowed types
func ValidateActivityType(action string) error {
	validActions := map[string]bool{
		models.AuditActionLogin:              true,
		models.AuditActionLogout:             true,
		models.AuditActionRegister:           true,
		models.AuditActionFailedLogin:        true,
		models.AuditActionAccountLocked:      true,
		models.AuditActionAccountUnlock:      true,
		models.AuditActionTokenRefresh:       true,
		models.AuditActionPasswordReset:      true,
		models.AuditActionCreate:             true,
		models.AuditActionUpdate:             true,
		models.AuditActionDelete:             true,
		models.AuditActionProfileUpdated:     true,
		models.AuditActionEmailUpdated:       true,
		models.AuditActionPasswordUpdated:    true,
		models.AuditActionCustomerCreated:    true,
		models.AuditActionCustomerDeleted:    true,
		models.AuditActionAccountCreated:     true,
		models.AuditActionAccountTransferred: true,
		models.AuditActionCustomerViewed:     true,
		models.AuditActionActivityViewed:     true,
	}

	if !validActions[action] {
		return fmt.Errorf("invalid activity type: %s", action)
	}
	return nil
}

// CreateAuditLog creates a new audit log entry with validation
func (s *AuditService) CreateAuditLog(log *models.AuditLog) error {
	if log == nil {
		return ErrInvalidAuditLog
	}

	if err := ValidateActivityType(log.Action); err != nil {
		return err
	}

	if err := s.repo.Create(log); err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// GetCustomerActivity retrieves activity logs for a specific customer with optional date filtering and pagination
func (s *AuditService) GetCustomerActivity(userID uuid.UUID, startDate, endDate *time.Time, offset, limit int) ([]*models.AuditLog, int64, error) {
	if userID == uuid.Nil {
		return nil, 0, ErrInvalidUserID
	}

	if startDate != nil && endDate != nil && startDate.After(*endDate) {
		return nil, 0, ErrAuditDateRange
	}

	return s.repo.GetCustomerActivity(userID, startDate, endDate, offset, limit)
}

// LogLogin logs a successful login event
func (s *AuditService) LogLogin(userID uuid.UUID, ipAddress, userAgent string) error {
	log := &models.AuditLog{
		UserID:     &userID,
		Action:     models.AuditActionLogin,
		Resource:   "auth",
		ResourceID: userID.String(),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	}
	return s.CreateAuditLog(log)
}

// LogLogout logs a logout event
func (s *AuditService) LogLogout(userID uuid.UUID, ipAddress, userAgent string) error {
	log := &models.AuditLog{
		UserID:     &userID,
		Action:     models.AuditActionLogout,
		Resource:   "auth",
		ResourceID: userID.String(),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	}
	return s.CreateAuditLog(log)
}

// LogProfileUpdate logs a customer profile update event
func (s *AuditService) LogProfileUpdate(userID, performedBy uuid.UUID, ipAddress, userAgent string, changes map[string]interface{}) error {
	log := &models.AuditLog{
		UserID:     &userID,
		Action:     models.AuditActionProfileUpdated,
		Resource:   "customer",
		ResourceID: userID.String(),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Metadata:   changes,
	}
	if performedBy != uuid.Nil {
		log.SetMetadata("performed_by", performedBy.String())
	}
	return s.CreateAuditLog(log)
}

// LogEmailUpdate logs an email update event
func (s *AuditService) LogEmailUpdate(userID, performedBy uuid.UUID, oldEmail, newEmail, ipAddress, userAgent string) error {
	log := &models.AuditLog{
		UserID:     &userID,
		Action:     models.AuditActionEmailUpdated,
		Resource:   "customer",
		ResourceID: userID.String(),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Metadata: models.JSONBMap{
			"old_email":    oldEmail,
			"new_email":    newEmail,
			"performed_by": performedBy.String(),
		},
	}
	return s.CreateAuditLog(log)
}

// LogPasswordReset logs an admin password reset event
func (s *AuditService) LogPasswordReset(userID, performedBy uuid.UUID, ipAddress, userAgent string) error {
	log := &models.AuditLog{
		UserID:     &userID,
		Action:     models.AuditActionPasswordReset,
		Resource:   "customer",
		ResourceID: userID.String(),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Metadata: models.JSONBMap{
			"performed_by": performedBy.String(),
		},
	}
	return s.CreateAuditLog(log)
}

// LogPasswordUpdate logs a customer self-service password update
func (s *AuditService) LogPasswordUpdate(userID uuid.UUID, ipAddress, userAgent string) error {
	log := &models.AuditLog{
		UserID:     &userID,
		Action:     models.AuditActionPasswordUpdated,
		Resource:   "customer",
		ResourceID: userID.String(),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	}
	return s.CreateAuditLog(log)
}

// LogCustomerCreated logs a customer creation event
func (s *AuditService) LogCustomerCreated(userID, performedBy uuid.UUID, ipAddress, userAgent string) error {
	log := &models.AuditLog{
		UserID:     &userID,
		Action:     models.AuditActionCustomerCreated,
		Resource:   "customer",
		ResourceID: userID.String(),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Metadata: models.JSONBMap{
			"performed_by": performedBy.String(),
		},
	}
	return s.CreateAuditLog(log)
}

// LogCustomerDeleted logs a customer deletion event
func (s *AuditService) LogCustomerDeleted(userID, performedBy uuid.UUID, ipAddress, userAgent string, reason string) error {
	log := &models.AuditLog{
		UserID:     &userID,
		Action:     models.AuditActionCustomerDeleted,
		Resource:   "customer",
		ResourceID: userID.String(),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Metadata: models.JSONBMap{
			"performed_by": performedBy.String(),
			"reason":       reason,
		},
	}
	return s.CreateAuditLog(log)
}

// LogAccountCreated logs an account creation event
func (s *AuditService) LogAccountCreated(userID, performedBy, accountID uuid.UUID, accountType, ipAddress, userAgent string) error {
	log := &models.AuditLog{
		UserID:     &userID,
		Action:     models.AuditActionAccountCreated,
		Resource:   "account",
		ResourceID: accountID.String(),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Metadata: models.JSONBMap{
			"performed_by": performedBy.String(),
			"account_type": accountType,
		},
	}
	return s.CreateAuditLog(log)
}

// LogAccountTransferred logs an account ownership transfer event
func (s *AuditService) LogAccountTransferred(fromUserID, toUserID, performedBy, accountID uuid.UUID, ipAddress, userAgent string) error {
	log := &models.AuditLog{
		UserID:     &fromUserID,
		Action:     models.AuditActionAccountTransferred,
		Resource:   "account",
		ResourceID: accountID.String(),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Metadata: models.JSONBMap{
			"from_user_id": fromUserID.String(),
			"to_user_id":   toUserID.String(),
			"performed_by": performedBy.String(),
		},
	}
	return s.CreateAuditLog(log)
}
