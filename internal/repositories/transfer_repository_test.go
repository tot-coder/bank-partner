package repositories

import (
	"testing"
	"time"

	"array-assessment/internal/models"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TransferRepositoryTestSuite is the test suite for Transfer repository
type TransferRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo TransferRepositoryInterface
}

// SetupTest runs before each test
func (s *TransferRepositoryTestSuite) SetupTest() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(s.T(), err)

	err = db.AutoMigrate(&models.Transfer{}, &models.Account{}, &models.User{}, &models.Transaction{})
	require.NoError(s.T(), err)

	s.db = db
	s.repo = NewTransferRepository(db)
}

// TearDownTest runs after each test
func (s *TransferRepositoryTestSuite) TearDownTest() {
	sqlDB, err := s.db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

// TestTransferRepositoryTestSuite runs the test suite
func TestTransferRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(TransferRepositoryTestSuite))
}

// Helper function to create a test transfer
func (s *TransferRepositoryTestSuite) createTestTransfer() *models.Transfer {
	return &models.Transfer{
		FromAccountID:  uuid.New(),
		ToAccountID:    uuid.New(),
		Amount:         decimal.NewFromFloat(gofakeit.Float64Range(10, 1000)),
		Description:    gofakeit.Sentence(5),
		IdempotencyKey: uuid.New().String(),
		Status:         models.TransferStatusPending,
	}
}

// TestCreate_ValidTransfer tests creating a valid transfer
func (s *TransferRepositoryTestSuite) TestCreate_ValidTransfer() {
	transfer := s.createTestTransfer()

	err := s.repo.Create(transfer)
	require.NoError(s.T(), err)
	assert.NotEqual(s.T(), uuid.Nil, transfer.ID)
	assert.False(s.T(), transfer.CreatedAt.IsZero())
}

// TestCreate_NilTransfer tests creating a nil transfer
func (s *TransferRepositoryTestSuite) TestCreate_NilTransfer() {
	err := s.repo.Create(nil)
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "transfer cannot be nil")
}

// TestCreate_DuplicateIdempotencyKey tests duplicate idempotency key
func (s *TransferRepositoryTestSuite) TestCreate_DuplicateIdempotencyKey() {
	idempotencyKey := uuid.New().String()

	transfer1 := s.createTestTransfer()
	transfer1.IdempotencyKey = idempotencyKey

	err := s.repo.Create(transfer1)
	require.NoError(s.T(), err)

	transfer2 := s.createTestTransfer()
	transfer2.IdempotencyKey = idempotencyKey

	err = s.repo.Create(transfer2)
	require.Error(s.T(), err)
	assert.Equal(s.T(), ErrTransferIdempotencyKeyExists, err)
}

// TestUpdate_ValidTransfer tests updating a transfer
func (s *TransferRepositoryTestSuite) TestUpdate_ValidTransfer() {
	transfer := s.createTestTransfer()
	err := s.repo.Create(transfer)
	require.NoError(s.T(), err)

	debitID := uuid.New()
	creditID := uuid.New()
	transfer.Complete(debitID, creditID)

	err = s.repo.Update(transfer)
	require.NoError(s.T(), err)

	retrieved, err := s.repo.FindByID(transfer.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), models.TransferStatusCompleted, retrieved.Status)
	assert.NotNil(s.T(), retrieved.CompletedAt)
	assert.Equal(s.T(), debitID, *retrieved.DebitTransactionID)
	assert.Equal(s.T(), creditID, *retrieved.CreditTransactionID)
}

// TestUpdate_NilTransfer tests updating a nil transfer
func (s *TransferRepositoryTestSuite) TestUpdate_NilTransfer() {
	err := s.repo.Update(nil)
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "transfer cannot be nil")
}

// TestFindByID_ExistingTransfer tests finding transfer by ID
func (s *TransferRepositoryTestSuite) TestFindByID_ExistingTransfer() {
	transfer := s.createTestTransfer()
	err := s.repo.Create(transfer)
	require.NoError(s.T(), err)

	retrieved, err := s.repo.FindByID(transfer.ID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), transfer.ID, retrieved.ID)
	assert.Equal(s.T(), transfer.FromAccountID, retrieved.FromAccountID)
	assert.Equal(s.T(), transfer.ToAccountID, retrieved.ToAccountID)
	assert.True(s.T(), transfer.Amount.Equal(retrieved.Amount))
}

// TestFindByID_NonExistingTransfer tests finding non-existing transfer
func (s *TransferRepositoryTestSuite) TestFindByID_NonExistingTransfer() {
	retrieved, err := s.repo.FindByID(uuid.New())
	require.Error(s.T(), err)
	assert.Nil(s.T(), retrieved)
	assert.Equal(s.T(), ErrTransferNotFound, err)
}

// TestFindByIdempotencyKey_ExistingTransfer tests finding by idempotency key
func (s *TransferRepositoryTestSuite) TestFindByIdempotencyKey_ExistingTransfer() {
	transfer := s.createTestTransfer()
	err := s.repo.Create(transfer)
	require.NoError(s.T(), err)

	retrieved, err := s.repo.FindByIdempotencyKey(transfer.IdempotencyKey)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), transfer.ID, retrieved.ID)
	assert.Equal(s.T(), transfer.IdempotencyKey, retrieved.IdempotencyKey)
}

// TestFindByIdempotencyKey_NonExistingKey tests finding by non-existing key
func (s *TransferRepositoryTestSuite) TestFindByIdempotencyKey_NonExistingKey() {
	retrieved, err := s.repo.FindByIdempotencyKey(uuid.New().String())
	require.Error(s.T(), err)
	assert.Nil(s.T(), retrieved)
	assert.Equal(s.T(), ErrTransferNotFound, err)
}

// TestFindByUserAccounts_WithTransfers tests finding transfers for user accounts
func (s *TransferRepositoryTestSuite) TestFindByUserAccounts_WithTransfers() {
	accountID1 := uuid.New()
	accountID2 := uuid.New()
	accountID3 := uuid.New()

	// Create transfers involving accountID1
	transfer1 := s.createTestTransfer()
	transfer1.FromAccountID = accountID1
	transfer1.ToAccountID = accountID2
	err := s.repo.Create(transfer1)
	require.NoError(s.T(), err)

	transfer2 := s.createTestTransfer()
	transfer2.FromAccountID = accountID2
	transfer2.ToAccountID = accountID1
	err = s.repo.Create(transfer2)
	require.NoError(s.T(), err)

	// Create transfer not involving accountID1
	transfer3 := s.createTestTransfer()
	transfer3.FromAccountID = accountID2
	transfer3.ToAccountID = accountID3
	err = s.repo.Create(transfer3)
	require.NoError(s.T(), err)

	// Find transfers for accountID1
	transfers, total, err := s.repo.FindByUserAccounts([]uuid.UUID{accountID1}, 0, 10)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(2), total)
	assert.Len(s.T(), transfers, 2)
}

// TestFindByUserAccounts_NoTransfers tests finding when no transfers exist
func (s *TransferRepositoryTestSuite) TestFindByUserAccounts_NoTransfers() {
	accountID := uuid.New()

	transfers, total, err := s.repo.FindByUserAccounts([]uuid.UUID{accountID}, 0, 10)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(0), total)
	assert.Len(s.T(), transfers, 0)
}

// TestFindByUserAccounts_WithPagination tests pagination
func (s *TransferRepositoryTestSuite) TestFindByUserAccounts_WithPagination() {
	accountID := uuid.New()

	// Create 5 transfers
	for i := 0; i < 5; i++ {
		transfer := s.createTestTransfer()
		transfer.FromAccountID = accountID
		err := s.repo.Create(transfer)
		require.NoError(s.T(), err)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}

	// First page
	transfers, total, err := s.repo.FindByUserAccounts([]uuid.UUID{accountID}, 0, 2)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(5), total)
	assert.Len(s.T(), transfers, 2)

	// Second page
	transfers, total, err = s.repo.FindByUserAccounts([]uuid.UUID{accountID}, 2, 2)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(5), total)
	assert.Len(s.T(), transfers, 2)

	// Third page
	transfers, total, err = s.repo.FindByUserAccounts([]uuid.UUID{accountID}, 4, 2)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(5), total)
	assert.Len(s.T(), transfers, 1)
}

// TestFindByUserAccounts_WithFilters tests finding with status filter
func (s *TransferRepositoryTestSuite) TestFindByUserAccounts_WithFilters() {
	accountID := uuid.New()

	// Create pending transfer
	transfer1 := s.createTestTransfer()
	transfer1.FromAccountID = accountID
	transfer1.Status = models.TransferStatusPending
	err := s.repo.Create(transfer1)
	require.NoError(s.T(), err)

	// Create completed transfer
	transfer2 := s.createTestTransfer()
	transfer2.FromAccountID = accountID
	err = s.repo.Create(transfer2)
	require.NoError(s.T(), err)
	transfer2.Complete(uuid.New(), uuid.New())
	err = s.repo.Update(transfer2)
	require.NoError(s.T(), err)

	// Create failed transfer
	transfer3 := s.createTestTransfer()
	transfer3.FromAccountID = accountID
	err = s.repo.Create(transfer3)
	require.NoError(s.T(), err)
	transfer3.Fail("test error")
	err = s.repo.Update(transfer3)
	require.NoError(s.T(), err)

	filters := models.TransferFilters{
		Status: models.TransferStatusCompleted,
	}

	transfers, total, err := s.repo.FindByUserAccountsWithFilters([]uuid.UUID{accountID}, filters, 0, 10)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1), total)
	assert.Len(s.T(), transfers, 1)
	assert.Equal(s.T(), models.TransferStatusCompleted, transfers[0].Status)
}

// TestCountByUserAccounts tests counting transfers
func (s *TransferRepositoryTestSuite) TestCountByUserAccounts() {
	accountID1 := uuid.New()
	accountID2 := uuid.New()

	// Create 3 transfers for accountID1
	for i := 0; i < 3; i++ {
		transfer := s.createTestTransfer()
		transfer.FromAccountID = accountID1
		err := s.repo.Create(transfer)
		require.NoError(s.T(), err)
	}

	// Create 2 transfers for accountID2
	for i := 0; i < 2; i++ {
		transfer := s.createTestTransfer()
		transfer.FromAccountID = accountID2
		err := s.repo.Create(transfer)
		require.NoError(s.T(), err)
	}

	count, err := s.repo.CountByUserAccounts([]uuid.UUID{accountID1})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(3), count)

	count, err = s.repo.CountByUserAccounts([]uuid.UUID{accountID1, accountID2})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(5), count)
}

// TestCountByUserAccounts_NoAccounts tests counting with empty account list
func (s *TransferRepositoryTestSuite) TestCountByUserAccounts_NoAccounts() {
	count, err := s.repo.CountByUserAccounts([]uuid.UUID{})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(0), count)
}
