package services

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand"

	"array-assessment/internal/models"
	"array-assessment/internal/repositories"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrUserNotFound             = errors.New("user not found")
	ErrAccountNotFound          = errors.New("account not found")
	ErrAccountAlreadyExists     = errors.New("account already exists for user")
	ErrInsufficientFunds        = errors.New("insufficient funds")
	ErrAccountNotActive         = errors.New("account is not active")
	ErrUnauthorized             = errors.New("unauthorized access to account")
	ErrInvalidAmount            = errors.New("invalid amount")
	ErrSameAccountTransfer      = errors.New("cannot transfer to same account")
	ErrAccountClosureNotAllowed = errors.New("account closure not allowed")
	ErrTransferPending          = errors.New("transfer is still processing with this idempotency key")
	ErrTransferFailed           = errors.New("previous transfer failed with this idempotency key")
)

// accountService implements AccountServiceInterface interface
type accountService struct {
	accountRepo     repositories.AccountRepositoryInterface
	transactionRepo repositories.TransactionRepositoryInterface
	transferRepo    repositories.TransferRepositoryInterface
	userRepo        repositories.UserRepositoryInterface
	auditRepo       repositories.AuditLogRepositoryInterface
	logger          *slog.Logger
}

// NewAccountService creates an account service with transfer and transaction support
func NewAccountService(
	accountRepo repositories.AccountRepositoryInterface,
	transactionRepo repositories.TransactionRepositoryInterface,
	transferRepo repositories.TransferRepositoryInterface,
	userRepo repositories.UserRepositoryInterface,
	auditRepo repositories.AuditLogRepositoryInterface,
	logger *slog.Logger,
) AccountServiceInterface {
	return &accountService{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		transferRepo:    transferRepo,
		userRepo:        userRepo,
		auditRepo:       auditRepo,
		logger:          logger,
	}
}

// CreateAccount creates a new account for a user
func (s *accountService) CreateAccount(userID uuid.UUID, accountType string, initialDeposit decimal.Decimal) (*models.Account, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to verify user: %w", err)
	}

	// Business rule: One account per type per user
	exists, err := s.accountRepo.ExistsForUser(userID, accountType)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing account: %w", err)
	}
	if exists {
		return nil, ErrAccountAlreadyExists
	}

	if initialDeposit.LessThan(decimal.Zero) {
		return nil, ErrInvalidAmount
	}

	accountNumber, err := s.accountRepo.GenerateUniqueAccountNumber(accountType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate account number: %w", err)
	}

	account := &models.Account{
		UserID:        userID,
		AccountNumber: accountNumber,
		AccountType:   accountType,
		Balance:       initialDeposit,
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}

	var transactions []models.Transaction
	if initialDeposit.GreaterThan(decimal.Zero) {
		transaction := models.Transaction{
			TransactionType: models.TransactionTypeCredit,
			Amount:          initialDeposit,
			BalanceBefore:   decimal.Zero,
			BalanceAfter:    initialDeposit,
			Description:     "Initial Deposit",
			Status:          models.TransactionStatusCompleted,
			Reference:       models.GenerateTransactionReference(),
		}
		transactions = append(transactions, transaction)
	}

	if err := s.accountRepo.CreateWithTransaction(account, transactions); err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	if err := s.auditRepo.Create(&models.AuditLog{
		UserID:     &user.ID,
		Action:     "account.created",
		Resource:   "account",
		ResourceID: account.ID.String(),
		IPAddress:  "system",
		UserAgent:  "internal",
		Metadata: models.JSONBMap{
			"account_type":   accountType,
			"account_number": account.AccountNumber,
		},
	}); err != nil {
		s.logger.Error("failed to create audit log", "error", err, "action", "account.created")
	}

	return account, nil
}

// CreateAccountsForNewUser creates default checking and savings accounts for a new user
func (s *accountService) CreateAccountsForNewUser(userID uuid.UUID) error {
	checkingBalance := s.generateRandomBalance(100, 5000)
	_, err := s.createAccountWithSampleData(userID, models.AccountTypeChecking, checkingBalance)
	if err != nil {
		return fmt.Errorf("failed to create checking account: %w", err)
	}

	savingsBalance := s.generateRandomBalance(500, 10000)
	_, err = s.createAccountWithSampleData(userID, models.AccountTypeSavings, savingsBalance)
	if err != nil {
		return fmt.Errorf("failed to create savings account: %w", err)
	}

	return nil
}

// createAccountWithSampleData creates an account with random sample transactions
func (s *accountService) createAccountWithSampleData(userID uuid.UUID, accountType string, targetBalance decimal.Decimal) (*models.Account, error) {
	accountNumber, err := s.accountRepo.GenerateUniqueAccountNumber(accountType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate account number: %w", err)
	}

	account := &models.Account{
		UserID:        userID,
		AccountNumber: accountNumber,
		AccountType:   accountType,
		Balance:       decimal.Zero,
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}

	transactions := s.generateSampleTransactions(targetBalance)

	finalBalance := decimal.Zero
	for i := range transactions {
		if transactions[i].TransactionType == models.TransactionTypeCredit {
			finalBalance = finalBalance.Add(transactions[i].Amount)
		} else {
			finalBalance = finalBalance.Sub(transactions[i].Amount)
		}
		transactions[i].BalanceAfter = finalBalance
	}

	account.Balance = finalBalance

	if err := s.accountRepo.CreateWithTransaction(account, transactions); err != nil {
		return nil, fmt.Errorf("failed to create account with sample data: %w", err)
	}

	return account, nil
}

// generateSampleTransactions generates random transactions that result in target balance
func (s *accountService) generateSampleTransactions(targetBalance decimal.Decimal) []models.Transaction {
	var transactions []models.Transaction
	currentBalance := decimal.Zero

	// 5-10 transactions simulates realistic account history
	numTransactions := rand.Intn(6) + 5

	initialDeposit := decimal.NewFromFloat(float64(rand.Intn(3000) + 2000))
	currentBalance = initialDeposit

	transactions = append(transactions, models.Transaction{
		TransactionType: models.TransactionTypeCredit,
		Amount:          initialDeposit,
		BalanceBefore:   decimal.Zero,
		BalanceAfter:    currentBalance,
		Description:     "Initial Deposit",
		Status:          models.TransactionStatusCompleted,
		Reference:       models.GenerateTransactionReference(),
	})

	for i := 1; i < numTransactions-1; i++ {
		isCredit := rand.Float32() > 0.4 // 60% chance of credit
		amount := decimal.NewFromFloat(float64(rand.Intn(500) + 50))

		var description string
		var transactionType string

		if isCredit {
			transactionType = models.TransactionTypeCredit
			description = models.SampleTransactionDescriptions[rand.Intn(len(models.SampleTransactionDescriptions))]
			currentBalance = currentBalance.Add(amount)
		} else {
			// Business rule: Account balance cannot go negative
			if currentBalance.LessThan(amount) {
				amount = currentBalance.Mul(decimal.NewFromFloat(0.3)) // Take only 30% of current balance
			}
			transactionType = models.TransactionTypeDebit
			description = models.SampleTransactionDescriptions[rand.Intn(len(models.SampleTransactionDescriptions))]
			currentBalance = currentBalance.Sub(amount)
		}

		transactions = append(transactions, models.Transaction{
			TransactionType: transactionType,
			Amount:          amount,
			BalanceBefore:   transactions[i-1].BalanceAfter,
			BalanceAfter:    currentBalance,
			Description:     description,
			Status:          models.TransactionStatusCompleted,
			Reference:       models.GenerateTransactionReference(),
		})
	}

	// Reconciliation transaction ensures exact target balance
	if !currentBalance.Equal(targetBalance) {
		var finalTransaction models.Transaction
		adjustment := targetBalance.Sub(currentBalance).Abs()

		if targetBalance.GreaterThan(currentBalance) {
			finalTransaction = models.Transaction{
				TransactionType: models.TransactionTypeCredit,
				Amount:          adjustment,
				BalanceBefore:   currentBalance,
				BalanceAfter:    targetBalance,
				Description:     "Bonus Interest Payment",
				Status:          models.TransactionStatusCompleted,
				Reference:       models.GenerateTransactionReference(),
			}
		} else {
			finalTransaction = models.Transaction{
				TransactionType: models.TransactionTypeDebit,
				Amount:          adjustment,
				BalanceBefore:   currentBalance,
				BalanceAfter:    targetBalance,
				Description:     "Service Fee Adjustment",
				Status:          models.TransactionStatusCompleted,
				Reference:       models.GenerateTransactionReference(),
			}
		}
		transactions = append(transactions, finalTransaction)
	}

	return transactions
}

// generateRandomBalance generates a random balance between min and max
func (s *accountService) generateRandomBalance(min, max int) decimal.Decimal {
	amount := rand.Intn(max-min) + min
	cents := rand.Intn(100)
	return decimal.NewFromFloat(float64(amount) + float64(cents)/100)
}

// GetAccountByID retrieves an account by ID with optional user verification
func (s *accountService) GetAccountByID(accountID uuid.UUID, userID *uuid.UUID) (*models.Account, error) {
	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Authorization: Non-admin users can only access their own accounts
	if userID != nil && account.UserID != *userID {
		user, err := s.userRepo.GetByID(*userID)
		if err != nil || !user.IsAdmin() {
			return nil, ErrUnauthorized
		}
	}

	return account, nil
}

// GetAccountByNumber retrieves an account by account number
func (s *accountService) GetAccountByNumber(accountNumber string) (*models.Account, error) {
	account, err := s.accountRepo.GetByAccountNumber(accountNumber)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to get account by number: %w", err)
	}
	return account, nil
}

// GetUserAccounts retrieves all accounts for a user
func (s *accountService) GetUserAccounts(userID uuid.UUID) ([]models.Account, error) {
	accounts, err := s.accountRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user accounts: %w", err)
	}
	return accounts, nil
}

// GetAllAccounts retrieves all accounts with filters (admin only)
func (s *accountService) GetAllAccounts(filters models.AccountFilters, offset, limit int) ([]models.Account, int64, error) {
	accounts, total, err := s.accountRepo.GetAllWithFilters(filters, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get all accounts: %w", err)
	}
	return accounts, total, nil
}

// UpdateAccountStatus updates the status of an account
func (s *accountService) UpdateAccountStatus(accountID uuid.UUID, userID *uuid.UUID, status string) (*models.Account, error) {
	account, err := s.GetAccountByID(accountID, userID)
	if err != nil {
		return nil, err
	}

	switch status {
	case models.AccountStatusActive:
		if err := account.Activate(); err != nil {
			return nil, err
		}
	case models.AccountStatusInactive:
		if err := account.Deactivate(); err != nil {
			return nil, err
		}
	case models.AccountStatusClosed:
		if err := account.Close(); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid account status: %s", status)
	}

	if err := s.accountRepo.Update(account); err != nil {
		return nil, fmt.Errorf("failed to update account status: %w", err)
	}

	if err := s.auditRepo.Create(&models.AuditLog{
		UserID:     &account.UserID,
		Action:     "account.status_changed",
		Resource:   "account",
		ResourceID: account.ID.String(),
		IPAddress:  "system",
		UserAgent:  "internal",
		Metadata: models.JSONBMap{
			"new_status": status,
		},
	}); err != nil {
		s.logger.Error("failed to create audit log", "error", err, "action", "account.status_changed")
	}

	return account, nil
}

// CloseAccount closes an account
func (s *accountService) CloseAccount(accountID uuid.UUID, userID uuid.UUID) error {
	account, err := s.GetAccountByID(accountID, &userID)
	if err != nil {
		return err
	}

	// Business rule: Cannot close account with non-zero balance
	if !account.Balance.IsZero() {
		return ErrAccountClosureNotAllowed
	}

	if err := account.Close(); err != nil {
		return err
	}

	if err := s.accountRepo.Update(account); err != nil {
		return fmt.Errorf("failed to close account: %w", err)
	}

	if err := s.auditRepo.Create(&models.AuditLog{
		UserID:     &userID,
		Action:     "account.closed",
		Resource:   "account",
		ResourceID: account.ID.String(),
		IPAddress:  "system",
		UserAgent:  "internal",
		Metadata: models.JSONBMap{
			"account_number": account.AccountNumber,
		},
	}); err != nil {
		s.logger.Error("failed to create audit log", "error", err, "action", "account.closed")
	}

	return nil
}

// PerformTransaction creates a transaction on an account
func (s *accountService) PerformTransaction(accountID uuid.UUID, amount decimal.Decimal, transactionType, description string, userID *uuid.UUID) (*models.Transaction, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidAmount
	}

	account, err := s.GetAccountByID(accountID, userID)
	if err != nil {
		return nil, err
	}

	if !account.IsActive() {
		return nil, ErrAccountNotActive
	}

	balanceBefore := account.Balance

	if err := s.accountRepo.UpdateBalance(accountID, amount, transactionType); err != nil {
		if errors.Is(err, repositories.ErrInsufficientFunds) {
			return nil, ErrInsufficientFunds
		}
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	var balanceAfter decimal.Decimal
	if transactionType == models.TransactionTypeCredit {
		balanceAfter = balanceBefore.Add(amount)
	} else {
		balanceAfter = balanceBefore.Sub(amount)
	}

	transaction := &models.Transaction{
		AccountID:       accountID,
		TransactionType: transactionType,
		Amount:          amount,
		BalanceBefore:   balanceBefore,
		BalanceAfter:    balanceAfter,
		Description:     description,
		Status:          models.TransactionStatusCompleted,
		Reference:       models.GenerateTransactionReference(),
	}

	if err := s.transactionRepo.Create(transaction); err != nil {
		return nil, fmt.Errorf("failed to create transaction record: %w", err)
	}

	if err := s.auditRepo.Create(&models.AuditLog{
		UserID:     &account.UserID,
		Action:     fmt.Sprintf("transaction.%s", transactionType),
		Resource:   "transaction",
		ResourceID: transaction.ID.String(),
		IPAddress:  "system",
		UserAgent:  "internal",
		Metadata: models.JSONBMap{
			"account_number": account.AccountNumber,
			"amount":         amount.String(),
			"type":           transactionType,
		},
	}); err != nil {
		s.logger.Error("failed to create audit log", "error", err, "action", fmt.Sprintf("transaction.%s", transactionType))
	}

	return transaction, nil
}

// TransferBetweenAccounts performs an atomic transfer with idempotency support
func (s *accountService) TransferBetweenAccounts(
	fromAccountID, toAccountID uuid.UUID,
	amount decimal.Decimal,
	description, idempotencyKey string,
	userID uuid.UUID,
) (*models.Transfer, error) {
	if err := s.validateTransferRequest(fromAccountID, toAccountID, amount, idempotencyKey); err != nil {
		return nil, err
	}

	if existingTransfer, err := s.checkExistingTransfer(idempotencyKey); existingTransfer != nil || err != nil {
		return existingTransfer, err
	}

	fromAccount, toAccount, err := s.retrieveAndAuthorizeAccounts(fromAccountID, toAccountID, userID)
	if err != nil {
		return nil, err
	}

	transfer, debitTxID, creditTxID, err := s.executeTransfer(
		amount, description, idempotencyKey,
		fromAccount, toAccount,
	)
	if err != nil {
		if transfer != nil {
			s.handleTransferFailure(transfer, err, fromAccount, toAccount, amount, idempotencyKey, userID)
		}
		return nil, err
	}

	if err := s.handleTransferSuccess(transfer, debitTxID, creditTxID, fromAccount, toAccount, amount, idempotencyKey, userID); err != nil {
		return nil, err
	}

	return transfer, nil
}

func (s *accountService) validateTransferRequest(
	fromAccountID, toAccountID uuid.UUID,
	amount decimal.Decimal,
	idempotencyKey string,
) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidAmount
	}

	if fromAccountID == toAccountID {
		return ErrSameAccountTransfer
	}

	if idempotencyKey == "" {
		return errors.New("idempotency key is required")
	}

	if s.transferRepo == nil {
		return errors.New("transfer repository not configured")
	}

	return nil
}

func (s *accountService) checkExistingTransfer(idempotencyKey string) (*models.Transfer, error) {
	existingTransfer, err := s.transferRepo.FindByIdempotencyKey(idempotencyKey)
	if err != nil {
		if errors.Is(err, repositories.ErrTransferNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to check idempotency key: %w", err)
	}

	switch existingTransfer.Status {
	case models.TransferStatusCompleted:
		return existingTransfer, nil
	case models.TransferStatusPending:
		return nil, ErrTransferPending
	case models.TransferStatusFailed:
		return nil, ErrTransferFailed
	}

	return nil, nil
}

func (s *accountService) retrieveAndAuthorizeAccounts(
	fromAccountID, toAccountID, userID uuid.UUID,
) (*models.Account, *models.Account, error) {
	fromAccount, err := s.accountRepo.GetByID(fromAccountID)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return nil, nil, ErrAccountNotFound
		}
		return nil, nil, fmt.Errorf("failed to get source account: %w", err)
	}

	toAccount, err := s.accountRepo.GetByID(toAccountID)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountNotFound) {
			return nil, nil, ErrAccountNotFound
		}
		return nil, nil, fmt.Errorf("failed to get destination account: %w", err)
	}

	if fromAccount.UserID != userID {
		return nil, nil, errors.New("not authorized to transfer from this account")
	}

	if !fromAccount.IsActive() {
		return nil, nil, ErrAccountNotActive
	}

	if !toAccount.IsActive() {
		return nil, nil, ErrAccountNotActive
	}

	return fromAccount, toAccount, nil
}

func (s *accountService) executeTransfer(
	amount decimal.Decimal,
	description, idempotencyKey string,
	fromAccount, toAccount *models.Account,
) (*models.Transfer, uuid.UUID, uuid.UUID, error) {
	transfer := &models.Transfer{
		FromAccountID:  fromAccount.ID,
		ToAccountID:    toAccount.ID,
		Amount:         amount,
		Description:    description,
		IdempotencyKey: idempotencyKey,
		Status:         models.TransferStatusPending,
	}

	if err := s.transferRepo.Create(transfer); err != nil {
		return nil, uuid.Nil, uuid.Nil, fmt.Errorf("failed to create transfer: %w", err)
	}

	fromDescription := fmt.Sprintf("Transfer to %s: %s", toAccount.AccountNumber, description)
	toDescription := fmt.Sprintf("Transfer from %s: %s", fromAccount.AccountNumber, description)

	debitTxID, creditTxID, err := s.accountRepo.ExecuteAtomicTransfer(
		fromAccount.ID,
		toAccount.ID,
		amount,
		fromDescription,
		toDescription,
	)

	return transfer, debitTxID, creditTxID, err
}

func (s *accountService) handleTransferFailure(
	transfer *models.Transfer,
	txErr error,
	fromAccount, toAccount *models.Account,
	amount decimal.Decimal,
	idempotencyKey string,
	userID uuid.UUID,
) {
	errorMsg := txErr.Error()
	transfer.Fail(errorMsg)
	if err := s.transferRepo.Update(transfer); err != nil {
		s.logger.Error("failed to update transfer status", "error", err, "transfer_id", transfer.ID)
	}

	if err := s.auditRepo.Create(&models.AuditLog{
		UserID:     &userID,
		Action:     "transfer.failed",
		Resource:   "transfer",
		ResourceID: transfer.ID.String(),
		IPAddress:  "system",
		UserAgent:  "internal",
		Metadata: models.JSONBMap{
			"from_account":    fromAccount.AccountNumber,
			"to_account":      toAccount.AccountNumber,
			"amount":          amount.String(),
			"error":           errorMsg,
			"idempotency_key": idempotencyKey,
		},
	}); err != nil {
		s.logger.Error("failed to create audit log", "error", err, "action", "transfer.failed")
	}
}

func (s *accountService) handleTransferSuccess(
	transfer *models.Transfer,
	debitTxID, creditTxID uuid.UUID,
	fromAccount, toAccount *models.Account,
	amount decimal.Decimal,
	idempotencyKey string,
	userID uuid.UUID,
) error {
	transfer.Complete(debitTxID, creditTxID)
	if err := s.transferRepo.Update(transfer); err != nil {
		return fmt.Errorf("failed to update transfer status: %w", err)
	}

	if err := s.auditRepo.Create(&models.AuditLog{
		UserID:     &userID,
		Action:     "transfer.completed",
		Resource:   "transfer",
		ResourceID: transfer.ID.String(),
		IPAddress:  "system",
		UserAgent:  "internal",
		Metadata: models.JSONBMap{
			"from_account":    fromAccount.AccountNumber,
			"to_account":      toAccount.AccountNumber,
			"amount":          amount.String(),
			"transfer_id":     transfer.ID.String(),
			"idempotency_key": idempotencyKey,
		},
	}); err != nil {
		s.logger.Error("failed to create audit log", "error", err, "action", "transfer.completed")
	}

	return nil
}

// GetAccountTransactions retrieves transactions for an account
func (s *accountService) GetAccountTransactions(accountID uuid.UUID, userID *uuid.UUID, offset, limit int) ([]models.Transaction, int64, error) {
	_, err := s.GetAccountByID(accountID, userID)
	if err != nil {
		return nil, 0, err
	}

	transactions, total, err := s.transactionRepo.GetByAccountID(accountID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get transactions: %w", err)
	}

	return transactions, total, nil
}

// GetRecentTransactions retrieves recent transactions for an account
func (s *accountService) GetRecentTransactions(accountID uuid.UUID, userID *uuid.UUID, limit int) ([]models.Transaction, error) {
	_, err := s.GetAccountByID(accountID, userID)
	if err != nil {
		return nil, err
	}

	transactions, err := s.transactionRepo.GetRecentByAccountID(accountID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent transactions: %w", err)
	}

	return transactions, nil
}

// GetUserTransfers retrieves transfer history for a user's accounts
func (s *accountService) GetUserTransfers(userID uuid.UUID, filters models.TransferFilters, offset, limit int) ([]models.Transfer, int64, error) {
	accounts, err := s.GetUserAccounts(userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user accounts: %w", err)
	}

	if len(accounts) == 0 {
		return []models.Transfer{}, 0, nil
	}

	accountIDs := make([]uuid.UUID, len(accounts))
	for i, account := range accounts {
		accountIDs[i] = account.ID
	}

	transfers, total, err := s.transferRepo.FindByUserAccountsWithFilters(accountIDs, filters, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user transfers: %w", err)
	}

	return transfers, total, nil
}
