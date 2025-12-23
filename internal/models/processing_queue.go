package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	QueueOperationProcess = "process"
	QueueOperationReverse = "reverse"

	QueueStatusPending    = "pending"
	QueueStatusProcessing = "processing"
	QueueStatusCompleted  = "completed"
	QueueStatusFailed     = "failed"

	QueuePriorityNormal = 100
	QueuePriorityHigh   = 200
)

type ProcessingQueueItem struct {
	ID            uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	TransactionID uuid.UUID  `gorm:"type:uuid;not null;index:idx_processing_queue_transaction" json:"transaction_id"`
	Operation     string     `gorm:"type:varchar(50);not null" json:"operation"`
	Priority      int        `gorm:"not null;default:100;index:idx_processing_queue_status,priority:2" json:"priority"`
	Status        string     `gorm:"type:varchar(20);not null;default:'pending';index:idx_processing_queue_status,priority:1" json:"status"`
	RetryCount    int        `gorm:"not null;default:0" json:"retry_count"`
	MaxRetries    int        `gorm:"not null;default:3" json:"max_retries"`
	ScheduledAt   time.Time  `gorm:"not null;index:idx_processing_queue_status,priority:3" json:"scheduled_at"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
	ErrorMessage  string     `gorm:"type:text" json:"error_message,omitempty"`
	Metadata      string     `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt     time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"not null" json:"updated_at"`

	Transaction Transaction `gorm:"foreignKey:TransactionID;constraint:OnDelete:CASCADE" json:"-"`
}

func (*ProcessingQueueItem) TableName() string {
	return "transaction_processing_queue"
}

func (q *ProcessingQueueItem) BeforeCreate(tx *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return nil
}

func (q *ProcessingQueueItem) CalculateNextScheduledTime() time.Time {
	backoffSeconds := 1 << uint(q.RetryCount)
	return time.Now().Add(time.Duration(backoffSeconds) * time.Second)
}

func (q *ProcessingQueueItem) CanRetry() bool {
	return q.RetryCount < q.MaxRetries
}
