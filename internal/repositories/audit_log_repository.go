package repositories

import (
	"errors"
	"fmt"
	"time"

	"array-assessment/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditLogRepository handles database operations for audit logs
type AuditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db *gorm.DB) AuditLogRepositoryInterface {
	return &AuditLogRepository{
		db: db,
	}
}

// Create creates a new audit log entry
func (r *AuditLogRepository) Create(log *models.AuditLog) error {
	if log == nil {
		return errors.New("audit log cannot be nil")
	}

	if err := r.db.Create(log).Error; err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// GetByID retrieves an audit log by its ID
func (r *AuditLogRepository) GetByID(id uuid.UUID) (*models.AuditLog, error) {
	log := &models.AuditLog{ID: id}
	if err := r.db.First(log).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("audit log not found")
		}
		return nil, fmt.Errorf("failed to get audit log by ID: %w", err)
	}

	return log, nil
}

// GetByUserID retrieves audit logs for a specific user
func (r *AuditLogRepository) GetByUserID(userID uuid.UUID, offset, limit int) ([]*models.AuditLog, int64, error) {
	var logs []*models.AuditLog
	var total int64

	query := r.db.Model(&models.AuditLog{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get audit logs for user: %w", err)
	}

	return logs, total, nil
}

// GetByAction retrieves audit logs for a specific action
func (r *AuditLogRepository) GetByAction(action string, offset, limit int) ([]*models.AuditLog, int64, error) {
	var logs []*models.AuditLog
	var total int64

	query := r.db.Model(&models.AuditLog{}).Where("action = ?", action)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get audit logs by action: %w", err)
	}

	return logs, total, nil
}

// GetByResource retrieves audit logs for a specific resource
func (r *AuditLogRepository) GetByResource(resource, resourceID string, offset, limit int) ([]*models.AuditLog, int64, error) {
	var logs []*models.AuditLog
	var total int64

	query := r.db.Model(&models.AuditLog{}).Where("resource = ? AND resource_id = ?", resource, resourceID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get audit logs by resource: %w", err)
	}

	return logs, total, nil
}

// GetByIPAddress retrieves audit logs from a specific IP address
func (r *AuditLogRepository) GetByIPAddress(ipAddress string, offset, limit int) ([]*models.AuditLog, int64, error) {
	var logs []*models.AuditLog
	var total int64

	query := r.db.Model(&models.AuditLog{}).Where("ip_address = ?", ipAddress)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get audit logs by IP: %w", err)
	}

	return logs, total, nil
}

// GetByTimeRange retrieves audit logs within a specific time range
func (r *AuditLogRepository) GetByTimeRange(startTime, endTime time.Time, offset, limit int) ([]*models.AuditLog, int64, error) {
	var logs []*models.AuditLog
	var total int64

	query := r.db.Model(&models.AuditLog{}).Where("created_at BETWEEN ? AND ?", startTime, endTime)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get audit logs by time range: %w", err)
	}

	return logs, total, nil
}

// GetCustomerActivity retrieves activity logs for a specific customer with optional date filtering and pagination
func (r *AuditLogRepository) GetCustomerActivity(userID uuid.UUID, startDate, endDate *time.Time, offset, limit int) ([]*models.AuditLog, int64, error) {
	if userID == uuid.Nil {
		return nil, 0, errors.New("invalid user ID")
	}

	if limit <= 0 || limit > 1000 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	var logs []*models.AuditLog
	var total int64

	query := r.db.Model(&models.AuditLog{}).Where("user_id = ?", userID)

	if startDate != nil {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", endDate)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get customer activity: %w", err)
	}

	return logs, total, nil
}

// GetFailedLoginAttempts retrieves failed login attempts for a specific email in a time window
func (r *AuditLogRepository) GetFailedLoginAttempts(email string, since time.Time) (int64, error) {
	var count int64

	err := r.db.Model(&models.AuditLog{}).
		Where("action = ? AND metadata->>'email' = ? AND created_at > ?",
			models.AuditActionFailedLogin, email, since).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count failed login attempts: %w", err)
	}

	return count, nil
}

// DeleteOlderThan removes audit logs older than the specified duration
func (r *AuditLogRepository) DeleteOlderThan(duration time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-duration)

	result := r.db.Where("created_at < ?", cutoffTime).Delete(&models.AuditLog{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old audit logs: %w", result.Error)
	}

	return result.RowsAffected, nil
}
