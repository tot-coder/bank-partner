package repositories

import (
	"errors"
	"fmt"
	"time"

	"array-assessment/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrQueueItemNotFound = errors.New("queue item not found")
)

type processingQueueRepository struct {
	db *gorm.DB
}

func NewProcessingQueueRepository(db *gorm.DB) ProcessingQueueRepositoryInterface {
	return &processingQueueRepository{
		db: db,
	}
}

func (r *processingQueueRepository) Enqueue(transactionID uuid.UUID, operation string, priority int) error {
	item := &models.ProcessingQueueItem{
		TransactionID: transactionID,
		Operation:     operation,
		Priority:      priority,
	}

	if err := r.db.Create(item).Error; err != nil {
		return fmt.Errorf("failed to enqueue transaction: %w", err)
	}

	return nil
}

func (r *processingQueueRepository) FetchPending(limit int) ([]*models.ProcessingQueueItem, error) {
	var items []*models.ProcessingQueueItem

	err := r.db.Where("status = ? AND scheduled_at <= ?", models.QueueStatusPending, time.Now()).
		Order("priority DESC, scheduled_at ASC").
		Limit(limit).
		Find(&items).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending items: %w", err)
	}

	return items, nil
}

func (r *processingQueueRepository) MarkProcessing(queueItemID uuid.UUID) error {
	result := r.db.Model(&models.ProcessingQueueItem{ID: queueItemID}).
		Update("status", models.QueueStatusProcessing)

	if result.Error != nil {
		return fmt.Errorf("failed to mark item as processing: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrQueueItemNotFound
	}

	return nil
}

func (r *processingQueueRepository) MarkCompleted(queueItemID uuid.UUID) error {
	now := time.Now()
	result := r.db.Model(&models.ProcessingQueueItem{ID: queueItemID}).
		Updates(map[string]interface{}{
			"status":       models.QueueStatusCompleted,
			"processed_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to mark item as completed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrQueueItemNotFound
	}

	return nil
}

func (r *processingQueueRepository) MarkFailed(queueItemID uuid.UUID, errorMessage string) error {
	now := time.Now()
	result := r.db.Model(&models.ProcessingQueueItem{ID: queueItemID}).
		Updates(map[string]interface{}{
			"status":        models.QueueStatusFailed,
			"error_message": errorMessage,
			"processed_at":  now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to mark item as failed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrQueueItemNotFound
	}

	return nil
}

func (r *processingQueueRepository) IncrementRetry(queueItemID uuid.UUID) error {
	item := &models.ProcessingQueueItem{ID: queueItemID}
	if err := r.db.First(item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrQueueItemNotFound
		}
		return fmt.Errorf("failed to find queue item: %w", err)
	}

	item.RetryCount++
	item.ScheduledAt = item.CalculateNextScheduledTime()
	item.Status = models.QueueStatusPending

	if err := r.db.Save(&item).Error; err != nil {
		return fmt.Errorf("failed to increment retry: %w", err)
	}

	return nil
}

func (r *processingQueueRepository) GetPendingCount() (int64, error) {
	var count int64
	err := r.db.Model(&models.ProcessingQueueItem{}).
		Where("status = ?", models.QueueStatusPending).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count pending items: %w", err)
	}

	return count, nil
}

func (r *processingQueueRepository) GetProcessingCount() (int64, error) {
	var count int64
	err := r.db.Model(&models.ProcessingQueueItem{}).
		Where("status = ?", models.QueueStatusProcessing).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count processing items: %w", err)
	}

	return count, nil
}

func (r *processingQueueRepository) GetFailedCount() (int64, error) {
	var count int64
	err := r.db.Model(&models.ProcessingQueueItem{}).
		Where("status = ?", models.QueueStatusFailed).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count failed items: %w", err)
	}

	return count, nil
}

func (r *processingQueueRepository) GetCompletedCount() (int64, error) {
	var count int64
	err := r.db.Model(&models.ProcessingQueueItem{}).
		Where("status = ?", models.QueueStatusCompleted).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count completed items: %w", err)
	}

	return count, nil
}

func (r *processingQueueRepository) GetAverageProcessingTime() (float64, error) {
	var result struct {
		AvgMs float64
	}

	err := r.db.Model(&models.ProcessingQueueItem{}).
		Select("AVG(EXTRACT(EPOCH FROM (processed_at - created_at)) * 1000) as avg_ms").
		Where("status = ? AND processed_at IS NOT NULL", models.QueueStatusCompleted).
		Scan(&result).Error

	if err != nil {
		return 0, fmt.Errorf("failed to calculate average processing time: %w", err)
	}

	return result.AvgMs, nil
}

func (r *processingQueueRepository) GetOldestPendingAge() (*string, error) {
	var item models.ProcessingQueueItem

	err := r.db.Where("status = ?", models.QueueStatusPending).
		Order("created_at ASC").
		First(&item).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find oldest pending item: %w", err)
	}

	age := time.Since(item.CreatedAt).String()
	return &age, nil
}

func (r *processingQueueRepository) CleanupCompleted(olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)

	result := r.db.Where("status = ? AND processed_at < ?", models.QueueStatusCompleted, cutoffTime).
		Delete(&models.ProcessingQueueItem{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to cleanup completed items: %w", result.Error)
	}

	return result.RowsAffected, nil
}
