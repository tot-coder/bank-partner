package repositories

import (
	"strings"
	"testing"

	"array-assessment/internal/database"
	"array-assessment/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

// AccountRepositorySuite defines the test suite for AccountRepository
type AccountRepositorySuite struct {
	suite.Suite
	db       *database.DB
	repo     AccountRepositoryInterface
	testUser *models.User
}

// SetupTest runs before each test in the suite
func (s *AccountRepositorySuite) SetupTest() {
	s.db = database.SetupTestDB(s.T())
	s.repo = NewAccountRepository(s.db.DB)

	// Create a test user for each test
	s.testUser = &models.User{
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		FirstName:    "Test",
		LastName:     "User",
		Role:         models.RoleCustomer,
	}
	err := s.db.DB.Create(s.testUser).Error
	s.NoError(err)
}

// TearDownTest runs after each test in the suite
func (s *AccountRepositorySuite) TearDownTest() {
	database.CleanupTestDB(s.T(), s.db)
}

// TestAccountRepositorySuite runs the test suite
func TestAccountRepositorySuite(t *testing.T) {
	suite.Run(t, new(AccountRepositorySuite))
}

// Test Create functionality
func (s *AccountRepositorySuite) TestCreate() {
	account := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}

	err := s.repo.Create(account)
	s.NoError(err)
	s.NotEqual(uuid.Nil, account.ID)
	s.NotZero(account.CreatedAt)
	s.NotZero(account.UpdatedAt)
}

func (s *AccountRepositorySuite) TestCreate_DuplicateAccountNumber() {
	account1 := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}

	err := s.repo.Create(account1)
	s.NoError(err)

	account2 := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",               // Same account number
		AccountType:   models.AccountTypeChecking, // Must match prefix
		Balance:       decimal.NewFromFloat(500.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}

	err = s.repo.Create(account2)
	s.Error(err)
	// Check for either PostgreSQL or SQLite duplicate error messages
	s.True(strings.Contains(err.Error(), "duplicate key value") || strings.Contains(err.Error(), "UNIQUE constraint failed"),
		"Expected duplicate error but got: %s", err.Error())
}

// Test GetByID functionality
func (s *AccountRepositorySuite) TestGetByID() {
	account := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}

	err := s.repo.Create(account)
	s.NoError(err)

	// Test getting existing account
	found, err := s.repo.GetByID(account.ID)
	s.NoError(err)
	s.NotNil(found)
	s.Equal(account.ID, found.ID)
	s.Equal(account.AccountNumber, found.AccountNumber)

	// Test getting non-existent account
	_, err = s.repo.GetByID(uuid.New())
	s.ErrorIs(err, ErrAccountNotFound)
}

// Test GetByAccountNumber functionality
func (s *AccountRepositorySuite) TestGetByAccountNumber() {
	account := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}

	err := s.repo.Create(account)
	s.NoError(err)

	// Test getting existing account
	found, err := s.repo.GetByAccountNumber("1012345678")
	s.NoError(err)
	s.NotNil(found)
	s.Equal(account.AccountNumber, found.AccountNumber)

	// Test getting non-existent account
	_, err = s.repo.GetByAccountNumber("9999999999")
	s.ErrorIs(err, ErrAccountNotFound)
}

// Test GetByUserID functionality
func (s *AccountRepositorySuite) TestGetByUserID() {
	// Create multiple accounts for the user
	account1 := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}
	err := s.repo.Create(account1)
	s.NoError(err)

	account2 := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "2012345679",
		AccountType:   models.AccountTypeSavings,
		Balance:       decimal.NewFromFloat(5000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}
	err = s.repo.Create(account2)
	s.NoError(err)

	// Get all accounts for user
	accounts, err := s.repo.GetByUserID(s.testUser.ID)
	s.NoError(err)
	s.Len(accounts, 2)

	// Verify account numbers
	accountNumbers := []string{accounts[0].AccountNumber, accounts[1].AccountNumber}
	s.Contains(accountNumbers, "1012345678")
	s.Contains(accountNumbers, "2012345679")
}

// Test Update functionality
func (s *AccountRepositorySuite) TestUpdate() {
	account := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}

	err := s.repo.Create(account)
	s.NoError(err)

	// Update account
	account.Balance = decimal.NewFromFloat(2000.00)
	account.Status = models.AccountStatusInactive

	err = s.repo.Update(account)
	s.NoError(err)

	// Verify update
	updated, err := s.repo.GetByID(account.ID)
	s.NoError(err)
	s.Equal(decimal.NewFromFloat(2000.00).String(), updated.Balance.String())
	s.Equal(models.AccountStatusInactive, updated.Status)
}

// Test Delete functionality
func (s *AccountRepositorySuite) TestDelete() {
	account := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}

	err := s.repo.Create(account)
	s.NoError(err)

	// Delete account
	err = s.repo.Delete(account.ID)
	s.NoError(err)

	// Verify deletion
	_, err = s.repo.GetByID(account.ID)
	s.ErrorIs(err, ErrAccountNotFound)
}

// Test GenerateUniqueAccountNumber functionality
func (s *AccountRepositorySuite) TestGenerateUniqueAccountNumber() {
	// Generate account number for checking account
	accountNumber1, err := s.repo.GenerateUniqueAccountNumber(models.AccountTypeChecking)
	s.NoError(err)
	s.NotEmpty(accountNumber1)
	s.Len(accountNumber1, 10)
	s.Equal("1", string(accountNumber1[0])) // Checking accounts start with 1

	// Generate account number for savings account
	accountNumber2, err := s.repo.GenerateUniqueAccountNumber(models.AccountTypeSavings)
	s.NoError(err)
	s.NotEmpty(accountNumber2)
	s.Len(accountNumber2, 10)
	s.Equal("2", string(accountNumber2[0])) // Savings accounts start with 2

	// Generate account number for money market account
	accountNumber3, err := s.repo.GenerateUniqueAccountNumber(models.AccountTypeMoneyMarket)
	s.NoError(err)
	s.NotEmpty(accountNumber3)
	s.Len(accountNumber3, 10)
	s.Equal("3", string(accountNumber3[0])) // Money market accounts start with 3

	// Ensure they are different
	s.NotEqual(accountNumber1, accountNumber2)
	s.NotEqual(accountNumber2, accountNumber3)
	s.NotEqual(accountNumber1, accountNumber3)
}

// Test CreateWithTransaction functionality
func (s *AccountRepositorySuite) TestCreateWithTransaction() {
	account := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}

	transactions := []models.Transaction{
		{
			TransactionType: models.TransactionTypeCredit,
			Amount:          decimal.NewFromFloat(1000.00),
			BalanceBefore:   decimal.Zero,
			BalanceAfter:    decimal.NewFromFloat(1000.00),
			Description:     "Initial Deposit",
			Reference:       "TXN-001",
			Status:          models.TransactionStatusCompleted,
		},
	}

	err := s.repo.CreateWithTransaction(account, transactions)
	s.NoError(err)
	s.NotEqual(uuid.Nil, account.ID)

	// Verify account was created
	foundAccount, err := s.repo.GetByID(account.ID)
	s.NoError(err)
	s.Equal(account.AccountNumber, foundAccount.AccountNumber)

	// Verify transaction was created
	var foundTransaction models.Transaction
	err = s.db.DB.Where("account_id = ?", account.ID).First(&foundTransaction).Error
	s.NoError(err)
	s.Equal(transactions[0].Reference, foundTransaction.Reference)
}

// Test UpdateBalance functionality
func (s *AccountRepositorySuite) TestUpdateBalance_Credit() {
	account := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}

	err := s.repo.Create(account)
	s.NoError(err)

	// Credit operation
	err = s.repo.UpdateBalance(account.ID, decimal.NewFromFloat(500.00), models.TransactionTypeCredit)
	s.NoError(err)

	// Verify balance
	updated, err := s.repo.GetByID(account.ID)
	s.NoError(err)
	s.Equal(decimal.NewFromFloat(1500.00).String(), updated.Balance.String())
}

func (s *AccountRepositorySuite) TestUpdateBalance_Debit() {
	account := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}

	err := s.repo.Create(account)
	s.NoError(err)

	// Debit operation
	err = s.repo.UpdateBalance(account.ID, decimal.NewFromFloat(300.00), models.TransactionTypeDebit)
	s.NoError(err)

	// Verify balance
	updated, err := s.repo.GetByID(account.ID)
	s.NoError(err)
	s.Equal(decimal.NewFromFloat(700.00).String(), updated.Balance.String())
}

func (s *AccountRepositorySuite) TestUpdateBalance_InsufficientFunds() {
	account := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(100.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}

	err := s.repo.Create(account)
	s.NoError(err)

	// Attempt debit with insufficient funds
	err = s.repo.UpdateBalance(account.ID, decimal.NewFromFloat(500.00), models.TransactionTypeDebit)
	s.ErrorIs(err, ErrInsufficientFunds)

	// Verify balance unchanged
	updated, err := s.repo.GetByID(account.ID)
	s.NoError(err)
	s.Equal(decimal.NewFromFloat(100.00).String(), updated.Balance.String())
}

// Test GetAccountsByStatus functionality
func (s *AccountRepositorySuite) TestGetAccountsByStatus() {
	// Create active accounts
	account1 := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}
	err := s.repo.Create(account1)
	s.NoError(err)

	// Create suspended account
	account2 := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "2012345679",
		AccountType:   models.AccountTypeSavings,
		Balance:       decimal.NewFromFloat(5000.00),
		Status:        models.AccountStatusInactive,
		Currency:      "USD",
	}
	err = s.repo.Create(account2)
	s.NoError(err)

	// Get active accounts
	accounts, err := s.repo.GetAccountsByStatus(models.AccountStatusActive, 0, 10)
	s.NoError(err)
	s.Len(accounts, 1)
	s.Equal(models.AccountStatusActive, accounts[0].Status)

	// Get suspended accounts
	accounts, err = s.repo.GetAccountsByStatus(models.AccountStatusInactive, 0, 10)
	s.NoError(err)
	s.Len(accounts, 1)
	s.Equal(models.AccountStatusInactive, accounts[0].Status)
}

// Test GetTotalBalanceByUserID functionality
func (s *AccountRepositorySuite) TestGetTotalBalanceByUserID() {
	// Create multiple accounts with different balances
	account1 := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}
	err := s.repo.Create(account1)
	s.NoError(err)

	account2 := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "2012345679",
		AccountType:   models.AccountTypeSavings,
		Balance:       decimal.NewFromFloat(5000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}
	err = s.repo.Create(account2)
	s.NoError(err)

	// Get total balance
	total, err := s.repo.GetTotalBalanceByUserID(s.testUser.ID)
	s.NoError(err)
	s.Equal(decimal.NewFromFloat(6000.00).String(), total.String())

	// Test with non-existent user
	total, err = s.repo.GetTotalBalanceByUserID(uuid.New())
	s.NoError(err)
	s.Equal(decimal.Zero.String(), total.String())
}

// Test ExistsForUser functionality
func (s *AccountRepositorySuite) TestExistsForUser() {
	// Create an account
	account := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}
	err := s.repo.Create(account)
	s.NoError(err)

	// Test exists for existing account type
	exists, err := s.repo.ExistsForUser(s.testUser.ID, models.AccountTypeChecking)
	s.NoError(err)
	s.True(exists)

	// Test does not exist for different account type
	exists, err = s.repo.ExistsForUser(s.testUser.ID, models.AccountTypeSavings)
	s.NoError(err)
	s.False(exists)

	// Test does not exist for non-existent user
	exists, err = s.repo.ExistsForUser(uuid.New(), models.AccountTypeChecking)
	s.NoError(err)
	s.False(exists)
}

// Test GetAll functionality
func (s *AccountRepositorySuite) TestGetAll() {
	// Create another user
	user2 := &models.User{
		Email:        "test2@example.com",
		PasswordHash: "hashedpassword",
		FirstName:    "Test2",
		LastName:     "User2",
		Role:         models.RoleCustomer,
	}
	err := s.db.DB.Create(user2).Error
	s.NoError(err)

	// Create accounts for both users
	account1 := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}
	err = s.repo.Create(account1)
	s.NoError(err)

	account2 := &models.Account{
		UserID:        user2.ID,
		AccountNumber: "2012345679",
		AccountType:   models.AccountTypeSavings,
		Balance:       decimal.NewFromFloat(5000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}
	err = s.repo.Create(account2)
	s.NoError(err)

	// Get all accounts
	accounts, total, err := s.repo.GetAll(0, 10)
	s.NoError(err)
	s.Len(accounts, 2)
	s.Equal(int64(2), total)

	// Test pagination
	accounts, total, err = s.repo.GetAll(0, 1)
	s.NoError(err)
	s.Len(accounts, 1)
	s.Equal(int64(2), total)
}

// Test GetAllWithFilters functionality
func (s *AccountRepositorySuite) TestGetAllWithFilters() {
	// Create another user
	user2 := &models.User{
		Email:        "test2@example.com",
		PasswordHash: "hashedpassword",
		FirstName:    "Test2",
		LastName:     "User2",
		Role:         models.RoleCustomer,
	}
	err := s.db.DB.Create(user2).Error
	s.NoError(err)

	// Create accounts with different attributes
	account1 := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}
	err = s.repo.Create(account1)
	s.NoError(err)

	account2 := &models.Account{
		UserID:        user2.ID,
		AccountNumber: "2012345679",
		AccountType:   models.AccountTypeSavings,
		Balance:       decimal.NewFromFloat(5000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}
	err = s.repo.Create(account2)
	s.NoError(err)

	account3 := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "2012345680", // Savings prefix is 20
		AccountType:   models.AccountTypeSavings,
		Balance:       decimal.NewFromFloat(2000.00),
		Status:        models.AccountStatusInactive,
		Currency:      "USD",
	}
	err = s.repo.Create(account3)
	s.NoError(err)

	// Test filter by user ID
	userID := s.testUser.ID
	filters := models.AccountFilters{
		UserID: &userID,
	}
	accounts, total, err := s.repo.GetAllWithFilters(filters, 0, 10)
	s.NoError(err)
	s.Len(accounts, 2)
	s.Equal(int64(2), total)

	// Test filter by status
	filters = models.AccountFilters{
		Status: models.AccountStatusActive,
	}
	accounts, total, err = s.repo.GetAllWithFilters(filters, 0, 10)
	s.NoError(err)
	s.Len(accounts, 2)
	s.Equal(int64(2), total)

	// Test filter by account type
	filters = models.AccountFilters{
		AccountType: models.AccountTypeSavings,
	}
	accounts, total, err = s.repo.GetAllWithFilters(filters, 0, 10)
	s.NoError(err)
	s.Len(accounts, 2)
	s.Equal(int64(2), total)

	// Test multiple filters
	filters = models.AccountFilters{
		UserID:      &userID,
		AccountType: models.AccountTypeSavings,
		Status:      models.AccountStatusActive,
	}
	accounts, total, err = s.repo.GetAllWithFilters(filters, 0, 10)
	s.NoError(err)
	s.Len(accounts, 0)
	s.Equal(int64(0), total)
}

// Test GetByUserIDAndType functionality
func (s *AccountRepositorySuite) TestGetByUserIDAndType() {
	// Create multiple accounts of different types
	account1 := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "1012345678",
		AccountType:   models.AccountTypeChecking,
		Balance:       decimal.NewFromFloat(1000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}
	err := s.repo.Create(account1)
	s.NoError(err)

	account2 := &models.Account{
		UserID:        s.testUser.ID,
		AccountNumber: "2012345679",
		AccountType:   models.AccountTypeSavings,
		Balance:       decimal.NewFromFloat(5000.00),
		Status:        models.AccountStatusActive,
		Currency:      "USD",
	}
	err = s.repo.Create(account2)
	s.NoError(err)

	// Get checking accounts
	accounts, err := s.repo.GetByUserIDAndType(s.testUser.ID, models.AccountTypeChecking)
	s.NoError(err)
	s.Len(accounts, 1)
	s.Equal(models.AccountTypeChecking, accounts[0].AccountType)

	// Get savings accounts
	accounts, err = s.repo.GetByUserIDAndType(s.testUser.ID, models.AccountTypeSavings)
	s.NoError(err)
	s.Len(accounts, 1)
	s.Equal(models.AccountTypeSavings, accounts[0].AccountType)

	// Get non-existent account type
	accounts, err = s.repo.GetByUserIDAndType(s.testUser.ID, models.AccountTypeMoneyMarket)
	s.NoError(err)
	s.Len(accounts, 0)
}
