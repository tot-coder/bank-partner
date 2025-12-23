package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRefreshToken_IsExpired(t *testing.T) {
	tests := []struct {
		name    string
		token   RefreshToken
		expired bool
	}{
		{
			name: "token not expired",
			token: RefreshToken{
				ExpiresAt: time.Now().Add(time.Hour),
			},
			expired: false,
		},
		{
			name: "token expired",
			token: RefreshToken{
				ExpiresAt: time.Now().Add(-time.Hour),
			},
			expired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expired, tt.token.IsExpired())
		})
	}
}

func TestRefreshToken_IsRevoked(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		token   RefreshToken
		revoked bool
	}{
		{
			name: "token not revoked",
			token: RefreshToken{
				RevokedAt: nil,
			},
			revoked: false,
		},
		{
			name: "token revoked",
			token: RefreshToken{
				RevokedAt: &now,
			},
			revoked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.revoked, tt.token.IsRevoked())
		})
	}
}

func TestRefreshToken_IsValid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		token RefreshToken
		valid bool
	}{
		{
			name: "valid token",
			token: RefreshToken{
				ExpiresAt: time.Now().Add(time.Hour),
				RevokedAt: nil,
			},
			valid: true,
		},
		{
			name: "expired token",
			token: RefreshToken{
				ExpiresAt: time.Now().Add(-time.Hour),
				RevokedAt: nil,
			},
			valid: false,
		},
		{
			name: "revoked token",
			token: RefreshToken{
				ExpiresAt: time.Now().Add(time.Hour),
				RevokedAt: &now,
			},
			valid: false,
		},
		{
			name: "expired and revoked token",
			token: RefreshToken{
				ExpiresAt: time.Now().Add(-time.Hour),
				RevokedAt: &now,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.token.IsValid())
		})
	}
}

func TestRefreshToken_Revoke(t *testing.T) {
	token := RefreshToken{
		RevokedAt: nil,
	}

	token.Revoke()

	assert.NotNil(t, token.RevokedAt)
	assert.True(t, token.IsRevoked())
	assert.False(t, token.IsValid())
}
