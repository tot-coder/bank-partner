package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	migrationsPath = "db/migrations"
	seedsPath      = "db/seeds"
)

var (
	maxRetries    = 30
	retryInterval = 2 * time.Second
)

// MigrationRunner handles database migrations and seeding
type MigrationRunner struct {
	db             *sql.DB
	migrationsPath string
	seedsPath      string
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *sql.DB) *MigrationRunner {
	return &MigrationRunner{
		db:             db,
		migrationsPath: migrationsPath,
		seedsPath:      seedsPath,
	}
}

// WaitForDatabase waits for the database to be ready
func (mr *MigrationRunner) WaitForDatabase() error {
	log.Println("Waiting for database to be ready...")

	for i := 0; i < maxRetries; i++ {
		err := mr.db.Ping()
		if err == nil {
			log.Println("Database is ready!")
			return nil
		}

		log.Printf("Database not ready (attempt %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(retryInterval)
	}

	return fmt.Errorf("database not ready after %d attempts", maxRetries)
}

// RunMigrations executes all pending migrations
func (mr *MigrationRunner) RunMigrations() error {
	if _, err := os.Stat(mr.migrationsPath); os.IsNotExist(err) {
		log.Printf("Migrations directory not found at %s, skipping migrations", mr.migrationsPath)
		return nil
	}

	absPath, err := filepath.Abs(mr.migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for migrations: %w", err)
	}

	log.Printf("Running migrations from: %s", absPath)

	driver, err := postgres.WithInstance(mr.db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", absPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	if dirty {
		log.Printf("Warning: database is in dirty state at version %d, forcing version", version)
		if err := m.Force(int(version)); err != nil {
			return fmt.Errorf("failed to force version: %w", err)
		}
	}

	log.Printf("Current migration version: %d", version)

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	if err == migrate.ErrNoChange {
		log.Println("No new migrations to apply")
	} else {
		newVersion, _, err := m.Version()
		if err != nil {
			return fmt.Errorf("failed to get new migration version: %w", err)
		}
		log.Printf("Successfully applied migrations. New version: %d", newVersion)
	}

	return nil
}

// LoadSeeds loads seed data into the database
func (mr *MigrationRunner) LoadSeeds() error {
	if os.Getenv("SEED_DATABASE") != "true" {
		log.Println("Seed data loading disabled (SEED_DATABASE != true)")
		return nil
	}

	if _, err := os.Stat(mr.seedsPath); os.IsNotExist(err) {
		log.Printf("Seeds directory not found at %s, skipping seed data", mr.seedsPath)
		return nil
	}

	log.Printf("Loading seed data from: %s", mr.seedsPath)

	files, err := filepath.Glob(filepath.Join(mr.seedsPath, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to find seed files: %w", err)
	}

	if len(files) == 0 {
		log.Println("No seed files found")
		return nil
	}

	for _, file := range files {
		log.Printf("Executing seed file: %s", filepath.Base(file))

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read seed file %s: %w", file, err)
		}

		if _, err := mr.db.Exec(string(content)); err != nil {
			log.Printf("Warning: failed to execute seed file %s: %v", file, err)
			continue
		}

		log.Printf("Successfully executed seed file: %s", filepath.Base(file))
	}

	log.Println("Seed data loaded successfully")
	return nil
}

// GetMigrationStatus returns the current migration status
func (mr *MigrationRunner) GetMigrationStatus() (version uint, dirty bool, err error) {
	if _, err := os.Stat(mr.migrationsPath); os.IsNotExist(err) {
		return 0, false, fmt.Errorf("migrations directory not found")
	}

	absPath, err := filepath.Abs(mr.migrationsPath)
	if err != nil {
		return 0, false, fmt.Errorf("failed to get absolute path: %w", err)
	}

	driver, err := postgres.WithInstance(mr.db, &postgres.Config{})
	if err != nil {
		return 0, false, fmt.Errorf("failed to create postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", absPath),
		"postgres",
		driver,
	)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migration instance: %w", err)
	}

	return m.Version()
}

// RunMigrationsIfEnabled runs migrations if AUTO_MIGRATE is set to true
func RunMigrationsIfEnabled(db *sql.DB) error {
	autoMigrate := os.Getenv("AUTO_MIGRATE")
	if autoMigrate != "true" {
		log.Println("Auto-migration disabled (AUTO_MIGRATE != true)")
		return nil
	}

	log.Println("Auto-migration enabled, running migrations...")

	runner := NewMigrationRunner(db)

	if err := runner.WaitForDatabase(); err != nil {
		return fmt.Errorf("database readiness check failed: %w", err)
	}

	if err := runner.RunMigrations(); err != nil {
		return fmt.Errorf("migration execution failed: %w", err)
	}

	if err := runner.LoadSeeds(); err != nil {
		log.Printf("Warning: seed data loading failed: %v", err)
	}

	version, dirty, err := runner.GetMigrationStatus()
	if err != nil {
		log.Printf("Warning: failed to get migration status: %v", err)
	} else {
		log.Printf("Migration status - Version: %d, Dirty: %v", version, dirty)
	}

	return nil
}
