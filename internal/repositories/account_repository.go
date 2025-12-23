package repositories

import (
	"errors"
	"fmt"
	"sync"

	"array-assessment/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	ErrAccountNotFound     = errors.New("account not found")
	ErrAccountNumberExists = errors.New("account number already exists")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrAccountNotActive    = errors.New("account is not active")
)

// accountRepository implements AccountRepository interface
type accountRepository struct {
	db *gorm.DB
	mu sync.Mutex // For account number generation
}

// NewAccountRepository creates a new account repository
func NewAccountRepository(db *gorm.DB) AccountRepositoryInterface {
	return &accountRepository{
		db: db,
	}
}

// Create creates a new account
func (r *accountRepository) Create(account *models.Account) error {
	if err := r.db.Create(account).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrAccountNumberExists
		}
		return fmt.Errorf("failed to create account: %w", err)
	}
	return nil
}

// GetByID retrieves an account by ID
func (r *accountRepository) GetByID(id uuid.UUID) (*models.Account, error) {
	account := &models.Account{ID: id}
	if err := r.db.First(account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	return account, nil
}

// GetByAccountNumber retrieves an account by account number
func (r *accountRepository) GetByAccountNumber(accountNumber string) (*models.Account, error) {
	var account models.Account
	if err := r.db.Where("account_number = ?", accountNumber).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to get account by number: %w", err)
	}
	return &account, nil
}

// GetByUserID retrieves all accounts for a user
func (r *accountRepository) GetByUserID(userID uuid.UUID) ([]models.Account, error) {
	var accounts []models.Account
	if err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get accounts for user: %w", err)
	}
	return accounts, nil
}

// GetByUserIDAndType retrieves accounts for a user by type
func (r *accountRepository) GetByUserIDAndType(userID uuid.UUID, accountType string) ([]models.Account, error) {
	var accounts []models.Account
	if err := r.db.Where("user_id = ? AND account_type = ?", userID, accountType).
		Order("created_at DESC").Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get accounts by type: %w", err)
	}
	return accounts, nil
}

// GetAll retrieves all accounts with pagination
func (r *accountRepository) GetAll(offset, limit int) ([]models.Account, int64, error) {
	var accounts []models.Account
	var total int64

	if err := r.db.Model(&models.Account{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count accounts: %w", err)
	}

	if err := r.db.Offset(offset).Limit(limit).
		Order("created_at DESC").Find(&accounts).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get accounts: %w", err)
	}

	return accounts, total, nil
}

// GetAllWithFilters retrieves accounts with filters and pagination
func (r *accountRepository) GetAllWithFilters(filters models.AccountFilters, offset, limit int) ([]models.Account, int64, error) {
	var accounts []models.Account
	var total int64

	query := r.db.Model(&models.Account{})

	if filters.UserID != nil {
		query = query.Where("user_id = ?", *filters.UserID)
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.AccountType != "" {
		query = query.Where("account_type = ?", filters.AccountType)
	}
	if filters.MinBalance != nil {
		query = query.Where("balance >= ?", *filters.MinBalance)
	}
	if filters.MaxBalance != nil {
		query = query.Where("balance <= ?", *filters.MaxBalance)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count filtered accounts: %w", err)
	}

	if err := query.Offset(offset).Limit(limit).
		Order("created_at DESC").Find(&accounts).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get filtered accounts: %w", err)
	}

	return accounts, total, nil
}

// Update updates an account
func (r *accountRepository) Update(account *models.Account) error {
	if err := r.db.Save(account).Error; err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}
	return nil
}

// Delete soft deletes an account
func (r *accountRepository) Delete(id uuid.UUID) error {
	result := r.db.Delete(&models.Account{ID: id})
	if result.Error != nil {
		return fmt.Errorf("failed to delete account: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrAccountNotFound
	}
	return nil
}

// GenerateUniqueAccountNumber generates a unique account number
func (r *accountRepository) GenerateUniqueAccountNumber(accountType string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		accountNumber := models.GenerateAccountNumber(accountType)
		if accountNumber == "" {
			return "", fmt.Errorf("invalid account type for number generation")
		}

		var count int64
		if err := r.db.Model(&models.Account{}).
			Where("account_number = ?", accountNumber).
			Count(&count).Error; err != nil {
			return "", fmt.Errorf("failed to check account number uniqueness: %w", err)
		}

		if count == 0 {
			return accountNumber, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique account number after %d attempts", maxAttempts)
}

// CreateWithTransaction creates an account with initial transactions in a database transaction
func (r *accountRepository) CreateWithTransaction(account *models.Account, transactions []models.Transaction) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(account).Error; err != nil {
			return fmt.Errorf("failed to create account: %w", err)
		}

		if len(transactions) > 0 {
			for i := range transactions {
				transactions[i].AccountID = account.ID
			}
			if err := tx.Create(&transactions).Error; err != nil {
				return fmt.Errorf("failed to create initial transactions: %w", err)
			}
		}

		return nil
	})
}

// UpdateBalance updates account balance within a transaction
func (r *accountRepository) UpdateBalance(accountID uuid.UUID, amount decimal.Decimal, transactionType string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		account := &models.Account{ID: accountID}

		// Row-level locking prevents concurrent balance modifications
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			First(&account).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAccountNotFound
			}
			return fmt.Errorf("failed to get account for update: %w", err)
		}

		if !account.IsActive() {
			return ErrAccountNotActive
		}

		if transactionType == models.TransactionTypeDebit {
			if account.Balance.LessThan(amount) {
				return ErrInsufficientFunds
			}
			account.Balance = account.Balance.Sub(amount)
		} else if transactionType == models.TransactionTypeCredit {
			account.Balance = account.Balance.Add(amount)
		} else {
			return fmt.Errorf("invalid transaction type: %s", transactionType)
		}

		if err := tx.Save(account).Error; err != nil {
			return fmt.Errorf("failed to update account balance: %w", err)
		}

		return nil
	})
}

// GetAccountsByStatus retrieves accounts by status
func (r *accountRepository) GetAccountsByStatus(status string, offset, limit int) ([]models.Account, error) {
	var accounts []models.Account
	if err := r.db.Where("status = ?", status).
		Offset(offset).Limit(limit).
		Order("created_at DESC").Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get accounts by status: %w", err)
	}
	return accounts, nil
}

// GetTotalBalanceByUserID calculates the total balance across all accounts for a user
func (r *accountRepository) GetTotalBalanceByUserID(userID uuid.UUID) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}

	if err := r.db.Model(&models.Account{}).
		Select("COALESCE(SUM(balance), 0) as total").
		Where("user_id = ? AND status = ?", userID, models.AccountStatusActive).
		Scan(&result).Error; err != nil {
		return decimal.Zero, fmt.Errorf("failed to calculate total balance: %w", err)
	}

	return result.Total, nil
}

// ExistsForUser checks if a user already has an account of the specified type
func (r *accountRepository) ExistsForUser(userID uuid.UUID, accountType string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Account{}).
		Where("user_id = ? AND account_type = ? AND status != ?",
			userID, accountType, models.AccountStatusClosed).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check account existence: %w", err)
	}
	return count > 0, nil
}

// ExecuteAtomicTransfer performs an atomic account-to-account transfer with row locking
func (r *accountRepository) ExecuteAtomicTransfer(fromAccountID, toAccountID uuid.UUID, amount decimal.Decimal, fromDescription, toDescription string) (debitTxID, creditTxID uuid.UUID, err error) {
	err = r.db.Transaction(func(tx *gorm.DB) error {
		// Debit from source account with row locking
		fromAcct := &models.Account{ID: fromAccountID}
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			First(&fromAcct).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAccountNotFound
			}
			return fmt.Errorf("failed to lock source account: %w", err)
		}

		if !fromAcct.IsActive() {
			return ErrAccountNotActive
		}

		if fromAcct.Balance.LessThan(amount) {
			return ErrInsufficientFunds
		}

		newFromBalance := fromAcct.Balance.Sub(amount)
		if err := tx.Model(fromAcct).Update("balance", newFromBalance).Error; err != nil {
			return fmt.Errorf("failed to debit source account: %w", err)
		}

		debitTx := &models.Transaction{
			AccountID:       fromAccountID,
			TransactionType: models.TransactionTypeDebit,
			Amount:          amount,
			BalanceBefore:   fromAcct.Balance,
			BalanceAfter:    newFromBalance,
			Description:     fromDescription,
			Status:          models.TransactionStatusCompleted,
			Reference:       models.GenerateTransactionReference(),
		}

		if err := tx.Create(debitTx).Error; err != nil {
			return fmt.Errorf("failed to create debit transaction: %w", err)
		}
		debitTxID = debitTx.ID

		// Credit destination account with row locking
		toAcct := &models.Account{ID: toAccountID}
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			First(&toAcct).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAccountNotFound
			}
			return fmt.Errorf("failed to lock destination account: %w", err)
		}

		if !toAcct.IsActive() {
			return ErrAccountNotActive
		}

		newToBalance := toAcct.Balance.Add(amount)
		if err := tx.Model(toAcct).Update("balance", newToBalance).Error; err != nil {
			return fmt.Errorf("failed to credit destination account: %w", err)
		}

		creditTx := &models.Transaction{
			AccountID:       toAccountID,
			TransactionType: models.TransactionTypeCredit,
			Amount:          amount,
			BalanceBefore:   toAcct.Balance,
			BalanceAfter:    newToBalance,
			Description:     toDescription,
			Status:          models.TransactionStatusCompleted,
			Reference:       models.GenerateTransactionReference(),
		}

		if err := tx.Create(creditTx).Error; err != nil {
			return fmt.Errorf("failed to create credit transaction: %w", err)
		}
		creditTxID = creditTx.ID

		return nil
	})

	return debitTxID, creditTxID, err
}

// GetByUserIDExcludingStatus retrieves all accounts for a user excluding a specific status
func (r *accountRepository) GetByUserIDExcludingStatus(userID uuid.UUID, excludeStatus string) ([]*models.Account, error) {
	var accounts []*models.Account
	if err := r.db.Where("user_id = ? AND status != ?", userID, excludeStatus).
		Order("created_at DESC").
		Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get accounts for user: %w", err)
	}
	return accounts, nil
}

// UpdateOwnership updates the ownership of an account
func (r *accountRepository) UpdateOwnership(accountID, newUserID uuid.UUID) error {
	result := r.db.Model(&models.Account{ID: accountID}).
		Update("user_id", newUserID)

	if result.Error != nil {
		return fmt.Errorf("failed to update account ownership: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrAccountNotFound
	}
	return nil
}

// SoftDeleteByUserID soft deletes all accounts for a user
func (r *accountRepository) SoftDeleteByUserID(userID uuid.UUID) error {
	result := r.db.Where("user_id = ?", userID).
		Delete(&models.Account{})

	if result.Error != nil {
		return fmt.Errorf("failed to soft delete accounts: %w", result.Error)
	}

	// Also update status to inactive for deleted accounts (requires Unscoped to update soft-deleted records)
	if result.RowsAffected > 0 {
		r.db.Unscoped().Model(&models.Account{}).
			Where("user_id = ? AND deleted_at IS NOT NULL", userID).
			Update("status", models.AccountStatusInactive)
	}

	return nil
}

// CheckAccountNumberExists checks if an account number already exists
func (r *accountRepository) CheckAccountNumberExists(accountNumber string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Account{}).
		Where("account_number = ?", accountNumber).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check account number existence: %w", err)
	}
	return count > 0, nil
}
