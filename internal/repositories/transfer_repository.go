package repositories

import (
	"errors"
	"fmt"

	"array-assessment/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrTransferNotFound             = errors.New("transfer not found")
	ErrTransferIdempotencyKeyExists = errors.New("transfer with idempotency key already exists")
)

// transferRepository implements TransferRepository interface
type transferRepository struct {
	db *gorm.DB
}

// NewTransferRepository creates a new transfer repository
func NewTransferRepository(db *gorm.DB) TransferRepositoryInterface {
	return &transferRepository{
		db: db,
	}
}

// Create creates a new transfer
func (r *transferRepository) Create(transfer *models.Transfer) error {
	if transfer == nil {
		return errors.New("transfer cannot be nil")
	}

	if err := r.db.Create(transfer).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || isDuplicateKeyError(err) {
			return ErrTransferIdempotencyKeyExists
		}
		return fmt.Errorf("failed to create transfer: %w", err)
	}

	return nil
}

// Update updates an existing transfer
func (r *transferRepository) Update(transfer *models.Transfer) error {
	if transfer == nil {
		return errors.New("transfer cannot be nil")
	}

	if err := r.db.Save(transfer).Error; err != nil {
		return fmt.Errorf("failed to update transfer: %w", err)
	}

	return nil
}

// FindByID retrieves a transfer by ID
func (r *transferRepository) FindByID(id uuid.UUID) (*models.Transfer, error) {
	transfer := &models.Transfer{ID: id}
	if err := r.db.First(transfer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTransferNotFound
		}
		return nil, fmt.Errorf("failed to find transfer by ID: %w", err)
	}

	return transfer, nil
}

// FindByIdempotencyKey retrieves a transfer by idempotency key
func (r *transferRepository) FindByIdempotencyKey(key string) (*models.Transfer, error) {
	var transfer models.Transfer

	if err := r.db.Where("idempotency_key = ?", key).First(&transfer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTransferNotFound
		}
		return nil, fmt.Errorf("failed to find transfer by idempotency key: %w", err)
	}

	return &transfer, nil
}

// FindByUserAccounts retrieves transfers involving any of the user's accounts
func (r *transferRepository) FindByUserAccounts(accountIDs []uuid.UUID, offset, limit int) ([]models.Transfer, int64, error) {
	return r.FindByUserAccountsWithFilters(accountIDs, models.TransferFilters{}, offset, limit)
}

// FindByUserAccountsWithFilters retrieves transfers with filtering options
func (r *transferRepository) FindByUserAccountsWithFilters(accountIDs []uuid.UUID, filters models.TransferFilters, offset, limit int) ([]models.Transfer, int64, error) {
	var transfers []models.Transfer
	var total int64

	if len(accountIDs) == 0 {
		return transfers, 0, nil
	}

	query := r.db.Model(&models.Transfer{}).
		Where("from_account_id IN ? OR to_account_id IN ?", accountIDs, accountIDs)

	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}

	if filters.FromAccountID != nil {
		query = query.Where("from_account_id = ?", *filters.FromAccountID)
	}

	if filters.ToAccountID != nil {
		query = query.Where("to_account_id = ?", *filters.ToAccountID)
	}

	if filters.MinAmount != nil {
		query = query.Where("amount >= ?", *filters.MinAmount)
	}

	if filters.MaxAmount != nil {
		query = query.Where("amount <= ?", *filters.MaxAmount)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count transfers: %w", err)
	}

	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&transfers).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find transfers by user accounts: %w", err)
	}

	return transfers, total, nil
}

// CountByUserAccounts counts transfers involving any of the user's accounts
func (r *transferRepository) CountByUserAccounts(accountIDs []uuid.UUID) (int64, error) {
	var count int64

	if len(accountIDs) == 0 {
		return 0, nil
	}

	if err := r.db.Model(&models.Transfer{}).
		Where("from_account_id IN ? OR to_account_id IN ?", accountIDs, accountIDs).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count transfers by user accounts: %w", err)
	}

	return count, nil
}
