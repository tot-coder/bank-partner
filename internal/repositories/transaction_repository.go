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
	ErrTransactionNotFound = errors.New("transaction not found")
)

// transactionRepository implements TransactionRepository interface
type transactionRepository struct {
	db *gorm.DB
}

// NewTransactionRepository creates a new transaction repository
func NewTransactionRepository(db *gorm.DB) TransactionRepositoryInterface {
	return &transactionRepository{
		db: db,
	}
}

// Create creates a new transaction
func (r *transactionRepository) Create(transaction *models.Transaction) error {
	if err := r.db.Create(transaction).Error; err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	return nil
}

// GetByID retrieves a transaction by ID
func (r *transactionRepository) GetByID(id uuid.UUID) (*models.Transaction, error) {
	transaction := &models.Transaction{ID: id}
	if err := r.db.First(transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	return transaction, nil
}

// GetByAccountID retrieves transactions for an account with pagination
func (r *transactionRepository) GetByAccountID(accountID uuid.UUID, offset, limit int) ([]models.Transaction, int64, error) {
	var transactions []models.Transaction
	var total int64

	if err := r.db.Model(&models.Transaction{}).
		Where("account_id = ?", accountID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	if err := r.db.Where("account_id = ?", accountID).
		Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&transactions).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get transactions: %w", err)
	}

	return transactions, total, nil
}

// GetByReference retrieves a transaction by reference
func (r *transactionRepository) GetByReference(reference string) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := r.db.Where("reference = ?", reference).First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("failed to get transaction by reference: %w", err)
	}
	return &transaction, nil
}

// GetRecentByAccountID retrieves recent transactions for an account
func (r *transactionRepository) GetRecentByAccountID(accountID uuid.UUID, limit int) ([]models.Transaction, error) {
	var transactions []models.Transaction
	if err := r.db.Where("account_id = ?", accountID).
		Order("created_at DESC").
		Limit(limit).
		Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to get recent transactions: %w", err)
	}
	return transactions, nil
}

// GetByDateRange retrieves transactions within a date range
func (r *transactionRepository) GetByDateRange(accountID uuid.UUID, startDate, endDate time.Time) ([]models.Transaction, error) {
	var transactions []models.Transaction
	if err := r.db.Where("account_id = ? AND created_at BETWEEN ? AND ?", accountID, startDate, endDate).
		Order("created_at DESC").
		Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to get transactions by date range: %w", err)
	}
	return transactions, nil
}

// CreateBatch creates multiple transactions in a single database transaction
func (r *transactionRepository) CreateBatch(transactions []models.Transaction) error {
	if len(transactions) == 0 {
		return nil
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&transactions).Error; err != nil {
			return fmt.Errorf("failed to create batch transactions: %w", err)
		}
		return nil
	})
}

// GetPendingTransactions retrieves all pending transactions
func (r *transactionRepository) GetPendingTransactions(offset, limit int) ([]models.Transaction, error) {
	var transactions []models.Transaction
	if err := r.db.Where("status = ?", models.TransactionStatusPending).
		Offset(offset).Limit(limit).
		Order("created_at ASC").
		Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to get pending transactions: %w", err)
	}
	return transactions, nil
}

// UpdateStatus updates the status of a transaction
func (r *transactionRepository) UpdateStatus(id uuid.UUID, status string) error {
	now := time.Now()
	result := r.db.Model(&models.Transaction{ID: id}).
		Updates(map[string]interface{}{
			"status":       status,
			"processed_at": now,
			"updated_at":   now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update transaction status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrTransactionNotFound
	}
	return nil
}

// GetTotalsByAccountID calculates transaction totals for an account
func (r *transactionRepository) GetTotalsByAccountID(accountID uuid.UUID) (credits, debits int64, creditAmount, debitAmount string, err error) {
	var creditResult struct {
		Count  int64
		Amount string
	}
	if err := r.db.Model(&models.Transaction{}).
		Select("COUNT(*) as count, COALESCE(SUM(amount), 0) as amount").
		Where("account_id = ? AND transaction_type = ? AND status = ?",
			accountID, models.TransactionTypeCredit, models.TransactionStatusCompleted).
		Scan(&creditResult).Error; err != nil {
		return 0, 0, "", "", fmt.Errorf("failed to get credit totals: %w", err)
	}

	var debitResult struct {
		Count  int64
		Amount string
	}
	if err := r.db.Model(&models.Transaction{}).
		Select("COUNT(*) as count, COALESCE(SUM(amount), 0) as amount").
		Where("account_id = ? AND transaction_type = ? AND status = ?",
			accountID, models.TransactionTypeDebit, models.TransactionStatusCompleted).
		Scan(&debitResult).Error; err != nil {
		return 0, 0, "", "", fmt.Errorf("failed to get debit totals: %w", err)
	}

	return creditResult.Count, debitResult.Count, creditResult.Amount, debitResult.Amount, nil
}

// GetByCategory retrieves transactions by category
func (r *transactionRepository) GetByCategory(accountID uuid.UUID, category string, offset, limit int) ([]models.Transaction, int64, error) {
	var transactions []models.Transaction
	var total int64

	query := r.db.Model(&models.Transaction{}).
		Where("account_id = ? AND category = ?", accountID, category)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count transactions by category: %w", err)
	}

	if err := query.Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&transactions).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get transactions by category: %w", err)
	}

	return transactions, total, nil
}

// GetWithFilters retrieves transactions with multiple filters
func (r *transactionRepository) GetWithFilters(filters models.TransactionFilters) ([]models.Transaction, int64, error) {
	var transactions []models.Transaction
	var total int64

	query := r.db.Model(&models.Transaction{})

	if filters.AccountID != uuid.Nil {
		query = query.Where("account_id = ?", filters.AccountID)
	}
	if filters.StartDate != nil {
		query = query.Where("created_at >= ?", *filters.StartDate)
	}
	if filters.EndDate != nil {
		query = query.Where("created_at <= ?", *filters.EndDate)
	}
	if filters.Type != "" {
		query = query.Where("transaction_type = ?", filters.Type)
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.Category != "" {
		query = query.Where("category = ?", filters.Category)
	}
	if filters.MinAmount != nil {
		query = query.Where("amount >= ?", *filters.MinAmount)
	}
	if filters.MaxAmount != nil {
		query = query.Where("amount <= ?", *filters.MaxAmount)
	}
	if filters.MerchantName != "" {
		query = query.Where("merchant_name ILIKE ?", "%"+filters.MerchantName+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count filtered transactions: %w", err)
	}

	if err := query.Offset(filters.Offset).Limit(filters.Limit).
		Order("created_at DESC").
		Find(&transactions).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get filtered transactions: %w", err)
	}

	return transactions, total, nil
}

// UpdateWithOptimisticLock updates a transaction with optimistic locking
func (r *transactionRepository) UpdateWithOptimisticLock(transaction *models.Transaction, expectedVersion int) error {
	result := r.db.Model(&models.Transaction{ID: transaction.ID}).
		Where("version = ?", expectedVersion).
		Updates(transaction)

	if result.Error != nil {
		return fmt.Errorf("failed to update transaction with optimistic lock: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return models.ErrOptimisticLockConflict
	}

	return nil
}

// GetExpiredPendingTransactions retrieves pending transactions that have expired
func (r *transactionRepository) GetExpiredPendingTransactions(limit int) ([]models.Transaction, error) {
	var transactions []models.Transaction
	now := time.Now()

	if err := r.db.Where("status = ? AND pending_until IS NOT NULL AND pending_until < ?",
		models.TransactionStatusPending, now).
		Limit(limit).
		Order("pending_until ASC").
		Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to get expired pending transactions: %w", err)
	}

	return transactions, nil
}

// GetCategorySummary retrieves transaction summary grouped by category
func (r *transactionRepository) GetCategorySummary(accountID uuid.UUID, startDate, endDate time.Time) ([]models.CategorySummary, error) {
	var summaries []models.CategorySummary

	query := `
		SELECT 
			category,
			COUNT(*) as transaction_count,
			SUM(amount) as total_amount,
			AVG(amount) as average_amount
		FROM transactions
		WHERE account_id = ?
			AND created_at BETWEEN ? AND ?
			AND status = ?
			AND category IS NOT NULL
		GROUP BY category
		ORDER BY total_amount DESC
	`

	if err := r.db.Raw(query, accountID, startDate, endDate, models.TransactionStatusCompleted).
		Scan(&summaries).Error; err != nil {
		return nil, fmt.Errorf("failed to get category summary: %w", err)
	}

	return summaries, nil
}
