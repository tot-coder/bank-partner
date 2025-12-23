package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuditLog_SetMetadata(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    interface{}
		expected JSONBMap
	}{
		{
			name:  "set string value",
			key:   "reason",
			value: "password reset",
			expected: JSONBMap{
				"reason": "password reset",
			},
		},
		{
			name:  "set numeric value",
			key:   "attempts",
			value: 3,
			expected: JSONBMap{
				"attempts": 3,
			},
		},
		{
			name:  "set boolean value",
			key:   "success",
			value: true,
			expected: JSONBMap{
				"success": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := &AuditLog{}
			log.SetMetadata(tt.key, tt.value)
			assert.NotNil(t, log.Metadata)
			assert.Equal(t, tt.expected, log.Metadata)
		})
	}
}

func TestAuditLog_GetMetadata(t *testing.T) {
	m := JSONBMap{
		"reason":   "password reset",
		"attempts": float64(3),
		"success":  true,
	}
	log := &AuditLog{
		Metadata: m,
	}

	tests := []struct {
		name         string
		key          string
		defaultValue interface{}
		expected     interface{}
	}{
		{
			name:         "get existing string value",
			key:          "reason",
			defaultValue: "",
			expected:     "password reset",
		},
		{
			name:         "get existing numeric value",
			key:          "attempts",
			defaultValue: 0,
			expected:     float64(3),
		},
		{
			name:         "get existing boolean value",
			key:          "success",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "get non-existing value returns default",
			key:          "nonexistent",
			defaultValue: "default",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := log.GetMetadata(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuditLog_String(t *testing.T) {
	userID := uuid.New()
	log := &AuditLog{
		UserID:     &userID,
		Action:     AuditActionLogin,
		Resource:   "auth",
		ResourceID: "user-123",
		IPAddress:  "192.168.1.1",
	}

	str := log.String()
	assert.Contains(t, str, "login")
	assert.Contains(t, str, "auth")
	assert.Contains(t, str, "user-123")
	assert.Contains(t, str, "192.168.1.1")
}
