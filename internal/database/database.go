package database

import (
	"fmt"
	"log"
	"time"

	"array-assessment/internal/config"
	"array-assessment/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB struct {
	*gorm.DB
	config *config.DatabaseConfig
}

func New(cfg *config.DatabaseConfig) (*DB, error) {
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxConnections)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		DB:     db,
		config: cfg,
	}, nil
}

func (db *DB) AutoMigrate() error {
	return db.DB.AutoMigrate(
		&models.User{},
		&models.RefreshToken{},
		&models.BlacklistedToken{},
		&models.AuditLog{},
		&models.Account{},
		&models.Transaction{},
		&models.Transfer{},
		&models.ProcessingQueueItem{},
	)
}

func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (db *DB) HealthCheck() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func (db *DB) Transaction(fn func(*gorm.DB) error) error {
	return db.DB.Transaction(fn)
}

func (db *DB) CreateIndexes() error {
	queries := []string{
		"CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)",
		"CREATE INDEX IF NOT EXISTS idx_users_role ON users(role)",
		"CREATE INDEX IF NOT EXISTS idx_users_locked_at ON users(locked_at) WHERE locked_at IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_users_last_login_at ON users(last_login_at)",
		"CREATE INDEX IF NOT EXISTS idx_users_first_name_lower ON users(LOWER(first_name))",
		"CREATE INDEX IF NOT EXISTS idx_users_last_name_lower ON users(LOWER(last_name))",
		"CREATE INDEX IF NOT EXISTS idx_users_email_lower ON users(LOWER(email))",
		"CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash)",
		"CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_blacklisted_tokens_jti ON blacklisted_tokens(jti)",
		"CREATE INDEX IF NOT EXISTS idx_blacklisted_tokens_expires_at ON blacklisted_tokens(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at)",
		// Account indexes
		"CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_accounts_account_number ON accounts(account_number)",
		"CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status)",
		"CREATE INDEX IF NOT EXISTS idx_accounts_account_type ON accounts(account_type)",
		"CREATE INDEX IF NOT EXISTS idx_accounts_deleted_at ON accounts(deleted_at) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_accounts_closed_at ON accounts(closed_at) WHERE closed_at IS NOT NULL",
		// Transaction indexes
		"CREATE INDEX IF NOT EXISTS idx_transactions_account_id ON transactions(account_id)",
		"CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_transactions_reference ON transactions(reference)",
		"CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status)",
		// Transfer indexes
		"CREATE INDEX IF NOT EXISTS idx_transfers_from_account_id ON transfers(from_account_id)",
		"CREATE INDEX IF NOT EXISTS idx_transfers_to_account_id ON transfers(to_account_id)",
		"CREATE INDEX IF NOT EXISTS idx_transfers_idempotency_key ON transfers(idempotency_key)",
		"CREATE INDEX IF NOT EXISTS idx_transfers_status ON transfers(status)",
		"CREATE INDEX IF NOT EXISTS idx_transfers_created_at ON transfers(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_transfers_debit_transaction_id ON transfers(debit_transaction_id) WHERE debit_transaction_id IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_transfers_credit_transaction_id ON transfers(credit_transaction_id) WHERE credit_transaction_id IS NOT NULL",
	}

	for _, query := range queries {
		if err := db.DB.Exec(query).Error; err != nil {
			log.Printf("Failed to create index: %s, error: %v", query, err)
		}
	}

	return nil
}

func (db *DB) CleanupExpiredTokens() error {
	now := time.Now()

	if err := db.DB.Where("expires_at < ?", now).Delete(&models.RefreshToken{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup expired refresh tokens: %w", err)
	}

	if err := db.DB.Where("expires_at < ?", now).Delete(&models.BlacklistedToken{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup expired blacklisted tokens: %w", err)
	}

	return nil
}

func (db *DB) SeedAdminUser(email, password, firstName, lastName string) (*models.User, error) {
	var existingUser models.User
	if err := db.DB.Where("email = ?", email).First(&existingUser).Error; err == nil {
		return &existingUser, nil
	}

	user := &models.User{
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Role:      models.RoleAdmin,
	}

	if err := db.DB.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	return user, nil
}

// Initialize creates and configures the database connection
func Initialize(cfg *config.Config) (*gorm.DB, error) {
	db, err := New(&cfg.Database)
	if err != nil {
		return nil, err
	}

	// Get the underlying sql.DB for migration runner
	sqlDB, err := db.DB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Run SQL-based migrations using golang-migrate if enabled
	if err := RunMigrationsIfEnabled(sqlDB); err != nil {
		log.Printf("Warning: migration runner failed: %v", err)
		log.Println("Falling back to GORM AutoMigrate...")

		// Fallback to GORM AutoMigrate
		if err := db.AutoMigrate(); err != nil {
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}
	}

	if err := db.CreateIndexes(); err != nil {
		log.Printf("Warning: failed to create some indexes: %v", err)
	}

	log.Println("Database initialized successfully")

	return db.DB, nil
}
