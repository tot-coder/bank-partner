package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	AuditActionLogin              = "login"
	AuditActionLogout             = "logout"
	AuditActionRegister           = "register"
	AuditActionFailedLogin        = "failed_login"
	AuditActionAccountLocked      = "account_locked"
	AuditActionAccountUnlock      = "account_unlock"
	AuditActionTokenRefresh       = "token_refresh"
	AuditActionPasswordReset      = "password_reset"
	AuditActionCreate             = "create"
	AuditActionUpdate             = "update"
	AuditActionDelete             = "delete"
	AuditActionProfileUpdated     = "profile_updated"
	AuditActionEmailUpdated       = "email_updated"
	AuditActionPasswordUpdated    = "password_updated"
	AuditActionCustomerCreated    = "customer_created"
	AuditActionCustomerDeleted    = "customer_deleted"
	AuditActionAccountCreated     = "account_created"
	AuditActionAccountTransferred = "account_transferred"
	AuditActionCustomerViewed     = "customer_viewed"
	AuditActionActivityViewed     = "activity_viewed"
)

type AuditLog struct {
	ID         uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	UserID     *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	Action     string     `gorm:"type:varchar(100);not null;index" json:"action"`
	Resource   string     `gorm:"type:varchar(100);not null" json:"resource"`
	ResourceID string     `gorm:"type:varchar(255)" json:"resource_id,omitempty"`
	IPAddress  string     `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent  string     `gorm:"type:text" json:"user_agent,omitempty"`
	Metadata   JSONBMap   `gorm:"type:text" json:"metadata,omitempty"`
	CreatedAt  time.Time  `gorm:"not null;index" json:"created_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL" json:"-"`
}

func (al *AuditLog) SetMetadata(key string, value interface{}) {
	if al.Metadata == nil {
		al.Metadata = make(JSONBMap)
	}
	al.Metadata[key] = value
}

func (al *AuditLog) GetMetadata(key string, defaultValue interface{}) interface{} {
	if al.Metadata == nil {
		return defaultValue
	}

	if value, exists := al.Metadata[key]; exists {
		return value
	}

	return defaultValue
}

func (al *AuditLog) String() string {
	userStr := "anonymous"
	if al.UserID != nil {
		userStr = al.UserID.String()
	}

	return fmt.Sprintf("AuditLog[User: %s, Action: %s, Resource: %s/%s, IP: %s, Time: %s]",
		userStr, al.Action, al.Resource, al.ResourceID, al.IPAddress, al.CreatedAt.Format(time.RFC3339))
}

func (al *AuditLog) TableName() string {
	return "audit_logs"
}

func (al *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if al.ID == uuid.Nil {
		al.ID = uuid.New()
	}

	if al.CreatedAt.IsZero() {
		al.CreatedAt = time.Now()
	}
	return nil
}

// JSONBMap represents a JSONB map field for PostgreSQL
// @Description Map of string keys to arbitrary values
// swaggertype: object
// additionalProperties: true
type JSONBMap map[string]interface{}

// Value implements driver.Valuer interface
func (m JSONBMap) Value() (driver.Value, error) {
	if m == nil || len(m) == 0 {
		return nil, nil
	}
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	// Return string for SQLite compatibility
	return string(bytes), nil
}

func (m *JSONBMap) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONBMap", value)
	}

	if len(bytes) == 0 {
		*m = nil
		return nil
	}

	return json.Unmarshal(bytes, m)
}

func (m JSONBMap) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return json.Marshal(map[string]interface{}(m))
}

func (m *JSONBMap) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var tmp map[string]interface{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*m = JSONBMap(tmp)
	return nil
}
