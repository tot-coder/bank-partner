package dto

// QueueMetrics represents metrics for the processing queue
type QueueMetrics struct {
	PendingCount    int64   `json:"pendingCount"`
	ProcessingCount int64   `json:"processingCount"`
	CompletedCount  int64   `json:"completedCount"`
	FailedCount     int64   `json:"failedCount"`
	AvgProcessingMs float64 `json:"avgProcessingMs"`
	OldestPending   *string `json:"oldestPending,omitempty"`
}
