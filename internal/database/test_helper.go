package database

import (
	"fmt"
	"testing"

	"array-assessment/internal/config"
	"array-assessment/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func SetupTestDB(t *testing.T) *DB {
	t.Helper()

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), gormConfig)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	testDB := &DB{
		DB: db,
		config: &config.DatabaseConfig{
			MaxConnections: 1,
			MaxIdleConns:   1,
		},
	}

	if err := testDB.AutoMigrate(); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return testDB
}

func CreateTestUser(t *testing.T, db *DB, email string) *models.User {
	t.Helper()

	user := &models.User{
		Email:        email,
		PasswordHash: "hashed_password",
		FirstName:    "Test",
		LastName:     "User",
		Role:         models.RoleCustomer,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user
}

func CreateTestAdminUser(t *testing.T, db *DB, email string) *models.User {
	t.Helper()

	user := &models.User{
		Email:        email,
		PasswordHash: "hashed_password",
		FirstName:    "Admin",
		LastName:     "User",
		Role:         models.RoleAdmin,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create test admin user: %v", err)
	}

	return user
}

type TestDB struct {
	*DB
	t *testing.T
}

func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), gormConfig)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	testDB := &DB{
		DB: db,
		config: &config.DatabaseConfig{
			MaxConnections: 1,
			MaxIdleConns:   1,
		},
	}

	if err := testDB.AutoMigrate(); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return &TestDB{
		DB: testDB,
		t:  t,
	}
}

func (tdb *TestDB) Cleanup() {
	tdb.t.Helper()

	tables := []string{
		"transaction_processing_queue",
		"transactions",
		"accounts",
		"audit_logs",
		"blacklisted_tokens",
		"refresh_tokens",
		"users",
	}

	for _, table := range tables {
		if err := tdb.DB.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error; err != nil {
			tdb.t.Logf("failed to cleanup table %s: %v", table, err)
		}
	}
}

func CleanupTestDB(t *testing.T, db *DB) {
	t.Helper()

	tables := []string{
		"transaction_processing_queue",
		"transactions",
		"accounts",
		"audit_logs",
		"blacklisted_tokens",
		"refresh_tokens",
		"users",
	}

	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error; err != nil {
			t.Logf("failed to cleanup table %s: %v", table, err)
		}
	}
}
