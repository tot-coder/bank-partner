package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid user",
			user: User{
				Email:     "test@example.com",
				FirstName: "John",
				LastName:  "Doe",
				Role:      RoleCustomer,
			},
			wantErr: false,
		},
		{
			name: "invalid email",
			user: User{
				Email:     "invalid-email",
				FirstName: "John",
				LastName:  "Doe",
				Role:      RoleCustomer,
			},
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name: "empty email",
			user: User{
				Email:     "",
				FirstName: "John",
				LastName:  "Doe",
				Role:      RoleCustomer,
			},
			wantErr: true,
			errMsg:  "email is required",
		},
		{
			name: "empty first name",
			user: User{
				Email:     "test@example.com",
				FirstName: "",
				LastName:  "Doe",
				Role:      RoleCustomer,
			},
			wantErr: true,
			errMsg:  "first name is required",
		},
		{
			name: "empty last name",
			user: User{
				Email:     "test@example.com",
				FirstName: "John",
				LastName:  "",
				Role:      RoleCustomer,
			},
			wantErr: true,
			errMsg:  "last name is required",
		},
		{
			name: "invalid role",
			user: User{
				Email:     "test@example.com",
				FirstName: "John",
				LastName:  "Doe",
				Role:      "invalid",
			},
			wantErr: true,
			errMsg:  "invalid role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUser_IsLocked(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name   string
		user   User
		locked bool
	}{
		{
			name: "user not locked",
			user: User{
				LockedAt: nil,
			},
			locked: false,
		},
		{
			name: "user locked",
			user: User{
				LockedAt: &now,
			},
			locked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.locked, tt.user.IsLocked())
		})
	}
}

func TestUser_Lock(t *testing.T) {
	user := User{
		FailedLoginAttempts: 2,
	}
	
	user.Lock()
	
	assert.NotNil(t, user.LockedAt)
	assert.Equal(t, 3, user.FailedLoginAttempts)
	assert.True(t, user.IsLocked())
}

func TestUser_Unlock(t *testing.T) {
	now := time.Now()
	user := User{
		FailedLoginAttempts: 3,
		LockedAt:           &now,
	}
	
	user.Unlock()
	
	assert.Nil(t, user.LockedAt)
	assert.Equal(t, 0, user.FailedLoginAttempts)
	assert.False(t, user.IsLocked())
}

func TestUser_IncrementFailedAttempts(t *testing.T) {
	user := User{
		FailedLoginAttempts: 0,
	}
	
	user.IncrementFailedAttempts()
	assert.Equal(t, 1, user.FailedLoginAttempts)
	assert.False(t, user.IsLocked())
	
	user.IncrementFailedAttempts()
	assert.Equal(t, 2, user.FailedLoginAttempts)
	assert.False(t, user.IsLocked())
	
	user.IncrementFailedAttempts()
	assert.Equal(t, 3, user.FailedLoginAttempts)
	assert.True(t, user.IsLocked())
}

func TestUser_ResetFailedAttempts(t *testing.T) {
	user := User{
		FailedLoginAttempts: 3,
	}
	
	user.ResetFailedAttempts()
	
	assert.Equal(t, 0, user.FailedLoginAttempts)
}

func TestUser_BeforeCreate(t *testing.T) {
	user := User{
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      RoleCustomer,
	}

	err := user.BeforeCreate(nil)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.NotZero(t, user.CreatedAt)
	assert.NotZero(t, user.UpdatedAt)
}

func TestUser_UpdateLastLogin(t *testing.T) {
	user := User{
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      RoleCustomer,
	}

	// Initially LastLoginAt should be nil
	assert.Nil(t, user.LastLoginAt)

	// Update last login
	before := time.Now()
	user.UpdateLastLogin()
	after := time.Now()

	// LastLoginAt should now be set
	require.NotNil(t, user.LastLoginAt)

	// Verify the timestamp is reasonable
	assert.True(t, user.LastLoginAt.After(before) || user.LastLoginAt.Equal(before))
	assert.True(t, user.LastLoginAt.Before(after) || user.LastLoginAt.Equal(after))

	// Update again and verify it changes
	time.Sleep(10 * time.Millisecond)
	firstLogin := *user.LastLoginAt
	user.UpdateLastLogin()

	require.NotNil(t, user.LastLoginAt)
	assert.True(t, user.LastLoginAt.After(firstLogin))
}