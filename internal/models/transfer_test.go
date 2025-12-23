package models

import (
	"testing"
	"time"

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

// TransferTestSuite is the test suite for Transfer model
type TransferTestSuite struct {
	suite.Suite
	db *gorm.DB
}

// SetupTest runs before each test
func (s *TransferTestSuite) SetupTest() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(s.T(), err)

	err = db.AutoMigrate(&Transfer{})
	require.NoError(s.T(), err)

	s.db = db
}

// TearDownTest runs after each test
func (s *TransferTestSuite) TearDownTest() {
	sqlDB, err := s.db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

// TestTransferTestSuite runs the test suite
func TestTransferTestSuite(t *testing.T) {
	suite.Run(t, new(TransferTestSuite))
}

// TestTransfer_BeforeCreate_GeneratesID tests ID generation
func (s *TransferTestSuite) TestTransfer_BeforeCreate_GeneratesID() {
	transfer := &Transfer{
		FromAccountID:  uuid.New(),
		ToAccountID:    uuid.New(),
		Amount:         decimal.NewFromFloat(100.00),
		Description:    gofakeit.Sentence(5),
		IdempotencyKey: uuid.New().String(),
		Status:         TransferStatusPending,
	}

	err := s.db.Create(transfer).Error
	require.NoError(s.T(), err)
	assert.NotEqual(s.T(), uuid.Nil, transfer.ID)
}

// TestTransfer_BeforeCreate_SetsDefaultStatus tests default status
func (s *TransferTestSuite) TestTransfer_BeforeCreate_SetsDefaultStatus() {
	transfer := &Transfer{
		FromAccountID:  uuid.New(),
		ToAccountID:    uuid.New(),
		Amount:         decimal.NewFromFloat(100.00),
		Description:    gofakeit.Sentence(5),
		IdempotencyKey: uuid.New().String(),
	}

	err := s.db.Create(transfer).Error
	require.NoError(s.T(), err)
	assert.Equal(s.T(), TransferStatusPending, transfer.Status)
}

// TestTransfer_BeforeCreate_SetsTimestamps tests timestamp setting
func (s *TransferTestSuite) TestTransfer_BeforeCreate_SetsTimestamps() {
	transfer := &Transfer{
		FromAccountID:  uuid.New(),
		ToAccountID:    uuid.New(),
		Amount:         decimal.NewFromFloat(100.00),
		Description:    gofakeit.Sentence(5),
		IdempotencyKey: uuid.New().String(),
		Status:         TransferStatusPending,
	}

	beforeCreate := time.Now()
	err := s.db.Create(transfer).Error
	afterCreate := time.Now()

	require.NoError(s.T(), err)
	assert.True(s.T(), transfer.CreatedAt.After(beforeCreate) || transfer.CreatedAt.Equal(beforeCreate))
	assert.True(s.T(), transfer.CreatedAt.Before(afterCreate) || transfer.CreatedAt.Equal(afterCreate))
	assert.True(s.T(), transfer.UpdatedAt.After(beforeCreate) || transfer.UpdatedAt.Equal(beforeCreate))
	assert.True(s.T(), transfer.UpdatedAt.Before(afterCreate) || transfer.UpdatedAt.Equal(afterCreate))
}

// TestTransfer_Validate_ValidTransfer tests validation with valid data
func (s *TransferTestSuite) TestTransfer_Validate_ValidTransfer() {
	transfer := &Transfer{
		ID:             uuid.New(),
		FromAccountID:  uuid.New(),
		ToAccountID:    uuid.New(),
		Amount:         decimal.NewFromFloat(100.00),
		Description:    gofakeit.Sentence(5),
		IdempotencyKey: uuid.New().String(),
		Status:         TransferStatusPending,
	}

	err := transfer.Validate()
	assert.NoError(s.T(), err)
}

// TestTransfer_Validate_MissingFromAccountID tests validation with missing from account
func (s *TransferTestSuite) TestTransfer_Validate_MissingFromAccountID() {
	transfer := &Transfer{
		ToAccountID:    uuid.New(),
		Amount:         decimal.NewFromFloat(100.00),
		Description:    gofakeit.Sentence(5),
		IdempotencyKey: uuid.New().String(),
		Status:         TransferStatusPending,
	}

	err := transfer.Validate()
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "from account ID is required")
}

// TestTransfer_Validate_MissingToAccountID tests validation with missing to account
func (s *TransferTestSuite) TestTransfer_Validate_MissingToAccountID() {
	transfer := &Transfer{
		FromAccountID:  uuid.New(),
		Amount:         decimal.NewFromFloat(100.00),
		Description:    gofakeit.Sentence(5),
		IdempotencyKey: uuid.New().String(),
		Status:         TransferStatusPending,
	}

	err := transfer.Validate()
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "to account ID is required")
}

// TestTransfer_Validate_SameFromAndToAccount tests validation with same accounts
func (s *TransferTestSuite) TestTransfer_Validate_SameFromAndToAccount() {
	accountID := uuid.New()
	transfer := &Transfer{
		FromAccountID:  accountID,
		ToAccountID:    accountID,
		Amount:         decimal.NewFromFloat(100.00),
		Description:    gofakeit.Sentence(5),
		IdempotencyKey: uuid.New().String(),
		Status:         TransferStatusPending,
	}

	err := transfer.Validate()
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "from and to accounts cannot be the same")
}

// TestTransfer_Validate_ZeroAmount tests validation with zero amount
func (s *TransferTestSuite) TestTransfer_Validate_ZeroAmount() {
	transfer := &Transfer{
		FromAccountID:  uuid.New(),
		ToAccountID:    uuid.New(),
		Amount:         decimal.Zero,
		Description:    gofakeit.Sentence(5),
		IdempotencyKey: uuid.New().String(),
		Status:         TransferStatusPending,
	}

	err := transfer.Validate()
	require.Error(s.T(), err)
	assert.Equal(s.T(), ErrInvalidTransferAmount, err)
}

// TestTransfer_Validate_NegativeAmount tests validation with negative amount
func (s *TransferTestSuite) TestTransfer_Validate_NegativeAmount() {
	transfer := &Transfer{
		FromAccountID:  uuid.New(),
		ToAccountID:    uuid.New(),
		Amount:         decimal.NewFromFloat(-100.00),
		Description:    gofakeit.Sentence(5),
		IdempotencyKey: uuid.New().String(),
		Status:         TransferStatusPending,
	}

	err := transfer.Validate()
	require.Error(s.T(), err)
	assert.Equal(s.T(), ErrInvalidTransferAmount, err)
}

// TestTransfer_Validate_MissingDescription tests validation without description
func (s *TransferTestSuite) TestTransfer_Validate_MissingDescription() {
	transfer := &Transfer{
		FromAccountID:  uuid.New(),
		ToAccountID:    uuid.New(),
		Amount:         decimal.NewFromFloat(100.00),
		IdempotencyKey: uuid.New().String(),
		Status:         TransferStatusPending,
	}

	err := transfer.Validate()
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "description is required")
}

// TestTransfer_Validate_MissingIdempotencyKey tests validation without idempotency key
func (s *TransferTestSuite) TestTransfer_Validate_MissingIdempotencyKey() {
	transfer := &Transfer{
		FromAccountID: uuid.New(),
		ToAccountID:   uuid.New(),
		Amount:        decimal.NewFromFloat(100.00),
		Description:   gofakeit.Sentence(5),
		Status:        TransferStatusPending,
	}

	err := transfer.Validate()
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "idempotency key is required")
}

// TestTransfer_Validate_InvalidStatus tests validation with invalid status
func (s *TransferTestSuite) TestTransfer_Validate_InvalidStatus() {
	transfer := &Transfer{
		FromAccountID:  uuid.New(),
		ToAccountID:    uuid.New(),
		Amount:         decimal.NewFromFloat(100.00),
		Description:    gofakeit.Sentence(5),
		IdempotencyKey: uuid.New().String(),
		Status:         "invalid_status",
	}

	err := transfer.Validate()
	require.Error(s.T(), err)
	assert.Equal(s.T(), ErrInvalidTransferStatus, err)
}

// TestTransfer_IsPending tests IsPending method
func (s *TransferTestSuite) TestTransfer_IsPending() {
	transfer := &Transfer{Status: TransferStatusPending}
	assert.True(s.T(), transfer.IsPending())

	transfer.Status = TransferStatusCompleted
	assert.False(s.T(), transfer.IsPending())
}

// TestTransfer_IsCompleted tests IsCompleted method
func (s *TransferTestSuite) TestTransfer_IsCompleted() {
	transfer := &Transfer{Status: TransferStatusCompleted}
	assert.True(s.T(), transfer.IsCompleted())

	transfer.Status = TransferStatusPending
	assert.False(s.T(), transfer.IsCompleted())
}

// TestTransfer_IsFailed tests IsFailed method
func (s *TransferTestSuite) TestTransfer_IsFailed() {
	transfer := &Transfer{Status: TransferStatusFailed}
	assert.True(s.T(), transfer.IsFailed())

	transfer.Status = TransferStatusPending
	assert.False(s.T(), transfer.IsFailed())
}

// TestTransfer_Complete tests Complete method
func (s *TransferTestSuite) TestTransfer_Complete() {
	debitID := uuid.New()
	creditID := uuid.New()

	transfer := &Transfer{
		FromAccountID:  uuid.New(),
		ToAccountID:    uuid.New(),
		Amount:         decimal.NewFromFloat(100.00),
		Description:    gofakeit.Sentence(5),
		IdempotencyKey: uuid.New().String(),
		Status:         TransferStatusPending,
	}

	transfer.Complete(debitID, creditID)

	assert.Equal(s.T(), TransferStatusCompleted, transfer.Status)
	assert.NotNil(s.T(), transfer.CompletedAt)
	assert.Equal(s.T(), debitID, *transfer.DebitTransactionID)
	assert.Equal(s.T(), creditID, *transfer.CreditTransactionID)
}

// TestTransfer_Fail tests Fail method
func (s *TransferTestSuite) TestTransfer_Fail() {
	errorMsg := "insufficient funds"
	transfer := &Transfer{
		FromAccountID:  uuid.New(),
		ToAccountID:    uuid.New(),
		Amount:         decimal.NewFromFloat(100.00),
		Description:    gofakeit.Sentence(5),
		IdempotencyKey: uuid.New().String(),
		Status:         TransferStatusPending,
	}

	transfer.Fail(errorMsg)

	assert.Equal(s.T(), TransferStatusFailed, transfer.Status)
	assert.NotNil(s.T(), transfer.FailedAt)
	assert.Equal(s.T(), errorMsg, *transfer.ErrorMessage)
}

// TestTransfer_CanTransitionTo tests status transitions
func (s *TransferTestSuite) TestTransfer_CanTransitionTo() {
	transfer := &Transfer{Status: TransferStatusPending}

	// Valid transitions from pending
	assert.True(s.T(), transfer.CanTransitionTo(TransferStatusCompleted))
	assert.True(s.T(), transfer.CanTransitionTo(TransferStatusFailed))

	// Invalid transitions from pending
	assert.False(s.T(), transfer.CanTransitionTo(TransferStatusPending))

	// Terminal states cannot transition
	transfer.Status = TransferStatusCompleted
	assert.False(s.T(), transfer.CanTransitionTo(TransferStatusFailed))
	assert.False(s.T(), transfer.CanTransitionTo(TransferStatusPending))

	transfer.Status = TransferStatusFailed
	assert.False(s.T(), transfer.CanTransitionTo(TransferStatusCompleted))
	assert.False(s.T(), transfer.CanTransitionTo(TransferStatusPending))
}

// TestIsValidTransferStatus tests status validation function
func (s *TransferTestSuite) TestIsValidTransferStatus() {
	assert.True(s.T(), IsValidTransferStatus(TransferStatusPending))
	assert.True(s.T(), IsValidTransferStatus(TransferStatusCompleted))
	assert.True(s.T(), IsValidTransferStatus(TransferStatusFailed))
	assert.False(s.T(), IsValidTransferStatus("invalid"))
	assert.False(s.T(), IsValidTransferStatus(""))
}

// TestTransfer_UniqueIdempotencyKey tests idempotency key uniqueness
func (s *TransferTestSuite) TestTransfer_UniqueIdempotencyKey() {
	idempotencyKey := uuid.New().String()

	transfer1 := &Transfer{
		FromAccountID:  uuid.New(),
		ToAccountID:    uuid.New(),
		Amount:         decimal.NewFromFloat(100.00),
		Description:    gofakeit.Sentence(5),
		IdempotencyKey: idempotencyKey,
		Status:         TransferStatusPending,
	}

	err := s.db.Create(transfer1).Error
	require.NoError(s.T(), err)

	// Attempt to create transfer with same idempotency key
	transfer2 := &Transfer{
		FromAccountID:  uuid.New(),
		ToAccountID:    uuid.New(),
		Amount:         decimal.NewFromFloat(200.00),
		Description:    gofakeit.Sentence(5),
		IdempotencyKey: idempotencyKey,
		Status:         TransferStatusPending,
	}

	err = s.db.Create(transfer2).Error
	require.Error(s.T(), err)
}
