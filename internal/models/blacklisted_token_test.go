package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBlacklistedToken_IsExpired(t *testing.T) {
	tests := []struct {
		name    string
		token   BlacklistedToken
		expired bool
	}{
		{
			name: "token not expired",
			token: BlacklistedToken{
				ExpiresAt: time.Now().Add(time.Hour),
			},
			expired: false,
		},
		{
			name: "token expired",
			token: BlacklistedToken{
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

func TestBlacklistedToken_CanBeDeleted(t *testing.T) {
	tests := []struct {
		name      string
		token     BlacklistedToken
		canDelete bool
	}{
		{
			name: "expired token can be deleted",
			token: BlacklistedToken{
				ExpiresAt: time.Now().Add(-time.Hour),
			},
			canDelete: true,
		},
		{
			name: "non-expired token cannot be deleted",
			token: BlacklistedToken{
				ExpiresAt: time.Now().Add(time.Hour),
			},
			canDelete: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.canDelete, tt.token.CanBeDeleted())
		})
	}
}
