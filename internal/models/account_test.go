package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccount_Validate(t *testing.T) {
	validUserID := uuid.New()

	tests := []struct {
		name    string
		account Account
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid checking account",
			account: Account{
				UserID:        validUserID,
				AccountNumber: "1012345678",
				AccountType:   AccountTypeChecking,
				Balance:       decimal.NewFromFloat(1000.50),
				Status:        AccountStatusActive,
			},
			wantErr: false,
		},
		{
			name: "valid savings account",
			account: Account{
				UserID:        validUserID,
				AccountNumber: "2012345678",
				AccountType:   AccountTypeSavings,
				Balance:       decimal.NewFromFloat(5000.00),
				Status:        AccountStatusActive,
			},
			wantErr: false,
		},
		{
			name: "valid money market account",
			account: Account{
				UserID:        validUserID,
				AccountNumber: "3012345678",
				AccountType:   AccountTypeMoneyMarket,
				Balance:       decimal.NewFromFloat(10000.00),
				Status:        AccountStatusActive,
			},
			wantErr: false,
		},
		{
			name: "missing user ID",
			account: Account{
				AccountNumber: "1012345678",
				AccountType:   AccountTypeChecking,
				Balance:       decimal.NewFromFloat(100.00),
				Status:        AccountStatusActive,
			},
			wantErr: true,
			errMsg:  "user ID is required",
		},
		{
			name: "missing account number",
			account: Account{
				UserID:      validUserID,
				AccountType: AccountTypeChecking,
				Balance:     decimal.NewFromFloat(100.00),
				Status:      AccountStatusActive,
			},
			wantErr: true,
			errMsg:  "account number is required",
		},
		{
			name: "invalid account number length",
			account: Account{
				UserID:        validUserID,
				AccountNumber: "12345",
				AccountType:   AccountTypeChecking,
				Balance:       decimal.NewFromFloat(100.00),
				Status:        AccountStatusActive,
			},
			wantErr: true,
			errMsg:  "account number must be 10 digits",
		},
		{
			name: "invalid account type",
			account: Account{
				UserID:        validUserID,
				AccountNumber: "1012345678",
				AccountType:   "invalid",
				Balance:       decimal.NewFromFloat(100.00),
				Status:        AccountStatusActive,
			},
			wantErr: true,
			errMsg:  "invalid account type",
		},
		{
			name: "invalid account status",
			account: Account{
				UserID:        validUserID,
				AccountNumber: "1012345678",
				AccountType:   AccountTypeChecking,
				Balance:       decimal.NewFromFloat(100.00),
				Status:        "invalid",
			},
			wantErr: true,
			errMsg:  "invalid account status",
		},
		{
			name: "negative balance",
			account: Account{
				UserID:        validUserID,
				AccountNumber: "1012345678",
				AccountType:   AccountTypeChecking,
				Balance:       decimal.NewFromFloat(-100.00),
				Status:        AccountStatusActive,
			},
			wantErr: true,
			errMsg:  "balance cannot be negative",
		},
		{
			name: "wrong prefix for account type",
			account: Account{
				UserID:        validUserID,
				AccountNumber: "2012345678", // Savings prefix for checking account
				AccountType:   AccountTypeChecking,
				Balance:       decimal.NewFromFloat(100.00),
				Status:        AccountStatusActive,
			},
			wantErr: true,
			errMsg:  "account number prefix does not match account type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.account.Validate()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAccount_Close(t *testing.T) {
	tests := []struct {
		name    string
		account Account
		wantErr bool
		errMsg  string
	}{
		{
			name: "close active account with zero balance",
			account: Account{
				Status:  AccountStatusActive,
				Balance: decimal.Zero,
			},
			wantErr: false,
		},
		{
			name: "close inactive account with zero balance",
			account: Account{
				Status:  AccountStatusInactive,
				Balance: decimal.Zero,
			},
			wantErr: false,
		},
		{
			name: "cannot close already closed account",
			account: Account{
				Status:  AccountStatusClosed,
				Balance: decimal.Zero,
			},
			wantErr: true,
			errMsg:  "account is already closed",
		},
		{
			name: "cannot close account with non-zero balance",
			account: Account{
				Status:  AccountStatusActive,
				Balance: decimal.NewFromFloat(100.00),
			},
			wantErr: true,
			errMsg:  "account balance must be zero to close",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.account.Close()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, AccountStatusClosed, tt.account.Status)
				assert.NotNil(t, tt.account.ClosedAt)
			}
		})
	}
}

func TestAccount_Deactivate(t *testing.T) {
	tests := []struct {
		name    string
		account Account
		wantErr bool
		errMsg  string
	}{
		{
			name: "deactivate active account",
			account: Account{
				Status: AccountStatusActive,
			},
			wantErr: false,
		},
		{
			name: "deactivate already inactive account",
			account: Account{
				Status: AccountStatusInactive,
			},
			wantErr: false,
		},
		{
			name: "cannot deactivate closed account",
			account: Account{
				Status: AccountStatusClosed,
			},
			wantErr: true,
			errMsg:  "cannot deactivate a closed account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.account.Deactivate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, AccountStatusInactive, tt.account.Status)
			}
		})
	}
}

func TestAccount_Activate(t *testing.T) {
	tests := []struct {
		name    string
		account Account
		wantErr bool
		errMsg  string
	}{
		{
			name: "activate inactive account",
			account: Account{
				Status: AccountStatusInactive,
			},
			wantErr: false,
		},
		{
			name: "activate already active account",
			account: Account{
				Status: AccountStatusActive,
			},
			wantErr: false,
		},
		{
			name: "cannot activate closed account",
			account: Account{
				Status: AccountStatusClosed,
			},
			wantErr: true,
			errMsg:  "cannot activate a closed account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.account.Activate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, AccountStatusActive, tt.account.Status)
			}
		})
	}
}

func TestAccount_Debit(t *testing.T) {
	tests := []struct {
		name           string
		account        Account
		amount         decimal.Decimal
		expectedBalance decimal.Decimal
		wantErr        bool
		errMsg         string
	}{
		{
			name: "successful debit",
			account: Account{
				Status:  AccountStatusActive,
				Balance: decimal.NewFromFloat(1000.00),
			},
			amount:          decimal.NewFromFloat(100.00),
			expectedBalance: decimal.NewFromFloat(900.00),
			wantErr:         false,
		},
		{
			name: "debit entire balance",
			account: Account{
				Status:  AccountStatusActive,
				Balance: decimal.NewFromFloat(500.00),
			},
			amount:          decimal.NewFromFloat(500.00),
			expectedBalance: decimal.Zero,
			wantErr:         false,
		},
		{
			name: "insufficient funds",
			account: Account{
				Status:  AccountStatusActive,
				Balance: decimal.NewFromFloat(100.00),
			},
			amount:  decimal.NewFromFloat(200.00),
			wantErr: true,
			errMsg:  "insufficient funds",
		},
		{
			name: "cannot debit inactive account",
			account: Account{
				Status:  AccountStatusInactive,
				Balance: decimal.NewFromFloat(1000.00),
			},
			amount:  decimal.NewFromFloat(100.00),
			wantErr: true,
			errMsg:  "account is not active",
		},
		{
			name: "cannot debit closed account",
			account: Account{
				Status:  AccountStatusClosed,
				Balance: decimal.NewFromFloat(1000.00),
			},
			amount:  decimal.NewFromFloat(100.00),
			wantErr: true,
			errMsg:  "account is not active",
		},
		{
			name: "negative debit amount",
			account: Account{
				Status:  AccountStatusActive,
				Balance: decimal.NewFromFloat(1000.00),
			},
			amount:  decimal.NewFromFloat(-100.00),
			wantErr: true,
			errMsg:  "debit amount must be positive",
		},
		{
			name: "zero debit amount",
			account: Account{
				Status:  AccountStatusActive,
				Balance: decimal.NewFromFloat(1000.00),
			},
			amount:  decimal.Zero,
			wantErr: true,
			errMsg:  "debit amount must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.account.Debit(tt.amount)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.True(t, tt.expectedBalance.Equal(tt.account.Balance))
			}
		})
	}
}

func TestAccount_Credit(t *testing.T) {
	tests := []struct {
		name           string
		account        Account
		amount         decimal.Decimal
		expectedBalance decimal.Decimal
		wantErr        bool
		errMsg         string
	}{
		{
			name: "successful credit",
			account: Account{
				Status:  AccountStatusActive,
				Balance: decimal.NewFromFloat(1000.00),
			},
			amount:          decimal.NewFromFloat(500.00),
			expectedBalance: decimal.NewFromFloat(1500.00),
			wantErr:         false,
		},
		{
			name: "credit to zero balance",
			account: Account{
				Status:  AccountStatusActive,
				Balance: decimal.Zero,
			},
			amount:          decimal.NewFromFloat(100.00),
			expectedBalance: decimal.NewFromFloat(100.00),
			wantErr:         false,
		},
		{
			name: "cannot credit inactive account",
			account: Account{
				Status:  AccountStatusInactive,
				Balance: decimal.NewFromFloat(1000.00),
			},
			amount:  decimal.NewFromFloat(100.00),
			wantErr: true,
			errMsg:  "account is not active",
		},
		{
			name: "cannot credit closed account",
			account: Account{
				Status:  AccountStatusClosed,
				Balance: decimal.Zero,
			},
			amount:  decimal.NewFromFloat(100.00),
			wantErr: true,
			errMsg:  "account is not active",
		},
		{
			name: "negative credit amount",
			account: Account{
				Status:  AccountStatusActive,
				Balance: decimal.NewFromFloat(1000.00),
			},
			amount:  decimal.NewFromFloat(-100.00),
			wantErr: true,
			errMsg:  "credit amount must be positive",
		},
		{
			name: "zero credit amount",
			account: Account{
				Status:  AccountStatusActive,
				Balance: decimal.NewFromFloat(1000.00),
			},
			amount:  decimal.Zero,
			wantErr: true,
			errMsg:  "credit amount must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.account.Credit(tt.amount)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.True(t, tt.expectedBalance.Equal(tt.account.Balance))
			}
		})
	}
}

func TestAccount_CanWithdraw(t *testing.T) {
	tests := []struct {
		name     string
		account  Account
		amount   decimal.Decimal
		expected bool
	}{
		{
			name: "can withdraw valid amount",
			account: Account{
				Status:  AccountStatusActive,
				Balance: decimal.NewFromFloat(1000.00),
			},
			amount:   decimal.NewFromFloat(500.00),
			expected: true,
		},
		{
			name: "can withdraw entire balance",
			account: Account{
				Status:  AccountStatusActive,
				Balance: decimal.NewFromFloat(100.00),
			},
			amount:   decimal.NewFromFloat(100.00),
			expected: true,
		},
		{
			name: "cannot withdraw more than balance",
			account: Account{
				Status:  AccountStatusActive,
				Balance: decimal.NewFromFloat(100.00),
			},
			amount:   decimal.NewFromFloat(200.00),
			expected: false,
		},
		{
			name: "cannot withdraw from inactive account",
			account: Account{
				Status:  AccountStatusInactive,
				Balance: decimal.NewFromFloat(1000.00),
			},
			amount:   decimal.NewFromFloat(100.00),
			expected: false,
		},
		{
			name: "cannot withdraw negative amount",
			account: Account{
				Status:  AccountStatusActive,
				Balance: decimal.NewFromFloat(1000.00),
			},
			amount:   decimal.NewFromFloat(-100.00),
			expected: false,
		},
		{
			name: "cannot withdraw zero amount",
			account: Account{
				Status:  AccountStatusActive,
				Balance: decimal.NewFromFloat(1000.00),
			},
			amount:   decimal.Zero,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.account.CanWithdraw(tt.amount)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateAccountNumber(t *testing.T) {
	tests := []struct {
		name         string
		accountType  string
		expectedPrefix string
	}{
		{
			name:         "checking account number",
			accountType:  AccountTypeChecking,
			expectedPrefix: CheckingPrefix,
		},
		{
			name:         "savings account number",
			accountType:  AccountTypeSavings,
			expectedPrefix: SavingsPrefix,
		},
		{
			name:         "money market account number",
			accountType:  AccountTypeMoneyMarket,
			expectedPrefix: MoneyMarketPrefix,
		},
		{
			name:         "invalid account type returns empty",
			accountType:  "invalid",
			expectedPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accountNumber := GenerateAccountNumber(tt.accountType)
			if tt.expectedPrefix == "" {
				assert.Empty(t, accountNumber)
			} else {
				assert.Len(t, accountNumber, 10)
				assert.Equal(t, tt.expectedPrefix, accountNumber[:2])
			}
		})
	}
}

func TestValidateAccountNumber(t *testing.T) {
	tests := []struct {
		name          string
		accountNumber string
		expected      bool
	}{
		{
			name:          "valid checking account number",
			accountNumber: "1012345678",
			expected:      true,
		},
		{
			name:          "valid savings account number",
			accountNumber: "2012345678",
			expected:      true,
		},
		{
			name:          "valid money market account number",
			accountNumber: "3012345678",
			expected:      true,
		},
		{
			name:          "too short",
			accountNumber: "12345",
			expected:      false,
		},
		{
			name:          "too long",
			accountNumber: "12345678901",
			expected:      false,
		},
		{
			name:          "contains non-digits",
			accountNumber: "10A2345678",
			expected:      false,
		},
		{
			name:          "invalid prefix",
			accountNumber: "9912345678",
			expected:      false,
		},
		{
			name:          "empty string",
			accountNumber: "",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateAccountNumber(tt.accountNumber)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAccount_BeforeCreate(t *testing.T) {
	account := Account{
		UserID:        uuid.New(),
		AccountNumber: "1012345678",
		AccountType:   AccountTypeChecking,
		Balance:       decimal.NewFromFloat(100.00),
	}

	// Simulate BeforeCreate hook
	err := account.BeforeCreate(nil)
	require.NoError(t, err)

	// Check defaults were set
	assert.NotEqual(t, uuid.Nil, account.ID)
	assert.Equal(t, AccountStatusActive, account.Status)
	assert.Equal(t, "USD", account.Currency)
	assert.True(t, account.InterestRate.IsZero())
	assert.NotZero(t, account.CreatedAt)
	assert.NotZero(t, account.UpdatedAt)
}

func TestAccount_InterestRateDefaults(t *testing.T) {
	tests := []struct {
		name         string
		accountType  string
		expectedRate float64
	}{
		{
			name:         "checking account has zero interest",
			accountType:  AccountTypeChecking,
			expectedRate: 0.0,
		},
		{
			name:         "savings account has default interest",
			accountType:  AccountTypeSavings,
			expectedRate: 0.0150,
		},
		{
			name:         "money market account has higher interest",
			accountType:  AccountTypeMoneyMarket,
			expectedRate: 0.0250,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := Account{
				UserID:        uuid.New(),
				AccountNumber: GenerateAccountNumber(tt.accountType),
				AccountType:   tt.accountType,
				Balance:       decimal.NewFromFloat(1000.00),
			}

			err := account.BeforeCreate(nil)
			require.NoError(t, err)

			expectedRate := decimal.NewFromFloat(tt.expectedRate)
			assert.True(t, expectedRate.Equal(account.InterestRate))
		})
	}
}