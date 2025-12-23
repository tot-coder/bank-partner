package database

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMigrationRunner(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	runner := NewMigrationRunner(db)

	assert.NotNil(t, runner)
	assert.Equal(t, db, runner.db)
	assert.Equal(t, migrationsPath, runner.migrationsPath)
	assert.Equal(t, seedsPath, runner.seedsPath)
}

func TestWaitForDatabase_Success(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	// Expect ping to succeed
	mock.ExpectPing().WillReturnError(nil)

	runner := NewMigrationRunner(db)
	err = runner.WaitForDatabase()

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWaitForDatabase_FailureThenSuccess(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	// First ping fails, second succeeds
	mock.ExpectPing().WillReturnError(errors.New("connection refused"))
	mock.ExpectPing().WillReturnError(nil)

	runner := NewMigrationRunner(db)

	// Override retry settings for faster test
	originalRetries := maxRetries
	originalInterval := retryInterval
	maxRetries = 2
	retryInterval = 100 * time.Millisecond
	defer func() {
		maxRetries = originalRetries
		retryInterval = originalInterval
	}()

	err = runner.WaitForDatabase()

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWaitForDatabase_AlwaysFails(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	// Override retry settings for faster test
	originalRetries := maxRetries
	originalInterval := retryInterval
	maxRetries = 2
	retryInterval = 100 * time.Millisecond
	defer func() {
		maxRetries = originalRetries
		retryInterval = originalInterval
	}()

	// All pings fail
	for i := 0; i < maxRetries; i++ {
		mock.ExpectPing().WillReturnError(errors.New("connection refused"))
	}

	runner := NewMigrationRunner(db)
	err = runner.WaitForDatabase()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not ready after")
}

func TestRunMigrations_DirectoryNotFound(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	runner := &MigrationRunner{
		db:             db,
		migrationsPath: "/nonexistent/path/to/migrations",
		seedsPath:      seedsPath,
	}

	err = runner.RunMigrations()

	// Should not error when directory doesn't exist
	assert.NoError(t, err)
}

func TestLoadSeeds_DisabledByEnvironment(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Ensure SEED_DATABASE is not set to "true"
	originalValue := os.Getenv("SEED_DATABASE")
	os.Setenv("SEED_DATABASE", "false")
	defer os.Setenv("SEED_DATABASE", originalValue)

	runner := NewMigrationRunner(db)
	err = runner.LoadSeeds()

	assert.NoError(t, err)
}

func TestLoadSeeds_DirectoryNotFound(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Enable seeding
	originalValue := os.Getenv("SEED_DATABASE")
	os.Setenv("SEED_DATABASE", "true")
	defer os.Setenv("SEED_DATABASE", originalValue)

	runner := &MigrationRunner{
		db:             db,
		migrationsPath: migrationsPath,
		seedsPath:      "/nonexistent/seeds/path",
	}

	err = runner.LoadSeeds()

	// Should not error when directory doesn't exist
	assert.NoError(t, err)
}

func TestLoadSeeds_NoSeedFiles(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Create temporary empty directory
	tempDir, err := os.MkdirTemp("", "seeds-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Enable seeding
	originalValue := os.Getenv("SEED_DATABASE")
	os.Setenv("SEED_DATABASE", "true")
	defer os.Setenv("SEED_DATABASE", originalValue)

	runner := &MigrationRunner{
		db:             db,
		migrationsPath: migrationsPath,
		seedsPath:      tempDir,
	}

	err = runner.LoadSeeds()

	assert.NoError(t, err)
}

func TestLoadSeeds_SuccessfulExecution(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Create temporary directory with seed file
	tempDir, err := os.MkdirTemp("", "seeds-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test seed file
	seedContent := `
INSERT INTO users (id, email, first_name, last_name)
VALUES ('a0000000-0000-0000-0000-000000000001', 'test@example.com', 'Test', 'User')
ON CONFLICT (email) DO NOTHING;
`
	seedFile := filepath.Join(tempDir, "001_test_data.sql")
	err = os.WriteFile(seedFile, []byte(seedContent), 0644)
	require.NoError(t, err)

	// Enable seeding
	originalValue := os.Getenv("SEED_DATABASE")
	os.Setenv("SEED_DATABASE", "true")
	defer os.Setenv("SEED_DATABASE", originalValue)

	// Expect the SQL execution
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(0, 1))

	runner := &MigrationRunner{
		db:             db,
		migrationsPath: migrationsPath,
		seedsPath:      tempDir,
	}

	err = runner.LoadSeeds()

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadSeeds_ExecutionFailureIsContinued(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Create temporary directory with two seed files
	tempDir, err := os.MkdirTemp("", "seeds-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create first seed file (will fail)
	seed1 := filepath.Join(tempDir, "001_bad_data.sql")
	err = os.WriteFile(seed1, []byte("INSERT INTO nonexistent_table VALUES (1);"), 0644)
	require.NoError(t, err)

	// Create second seed file (will succeed)
	seed2 := filepath.Join(tempDir, "002_good_data.sql")
	err = os.WriteFile(seed2, []byte("INSERT INTO users VALUES ('test');"), 0644)
	require.NoError(t, err)

	// Enable seeding
	originalValue := os.Getenv("SEED_DATABASE")
	os.Setenv("SEED_DATABASE", "true")
	defer os.Setenv("SEED_DATABASE", originalValue)

	// Expect first execution to fail, second to succeed
	mock.ExpectExec("INSERT INTO nonexistent_table").WillReturnError(errors.New("table does not exist"))
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(0, 1))

	runner := &MigrationRunner{
		db:             db,
		migrationsPath: migrationsPath,
		seedsPath:      tempDir,
	}

	err = runner.LoadSeeds()

	// Should not error even though one file failed
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunMigrationsIfEnabled_Disabled(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Ensure AUTO_MIGRATE is not set to "true"
	originalValue := os.Getenv("AUTO_MIGRATE")
	os.Setenv("AUTO_MIGRATE", "false")
	defer os.Setenv("AUTO_MIGRATE", originalValue)

	err = RunMigrationsIfEnabled(db)

	assert.NoError(t, err)
}

func TestRunMigrationsIfEnabled_Enabled_DatabaseNotReady(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	// Enable auto-migrate
	originalValue := os.Getenv("AUTO_MIGRATE")
	os.Setenv("AUTO_MIGRATE", "true")
	defer os.Setenv("AUTO_MIGRATE", originalValue)

	// Override retry settings for faster test
	originalRetries := maxRetries
	originalInterval := retryInterval
	maxRetries = 2
	retryInterval = 100 * time.Millisecond
	defer func() {
		maxRetries = originalRetries
		retryInterval = originalInterval
	}()

	// All pings fail
	for i := 0; i < maxRetries; i++ {
		mock.ExpectPing().WillReturnError(errors.New("connection refused"))
	}

	err = RunMigrationsIfEnabled(db)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database readiness check failed")
}

func TestMigrationRunner_WaitForDatabase_WithTimeout(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	// Override retry settings for faster test
	originalRetries := maxRetries
	originalInterval := retryInterval
	maxRetries = 4
	retryInterval = 100 * time.Millisecond
	defer func() {
		maxRetries = originalRetries
		retryInterval = originalInterval
	}()

	// Simulate slow database startup - fail 3 times then succeed
	mock.ExpectPing().WillDelayFor(100 * time.Millisecond).WillReturnError(errors.New("starting"))
	mock.ExpectPing().WillDelayFor(100 * time.Millisecond).WillReturnError(errors.New("starting"))
	mock.ExpectPing().WillDelayFor(100 * time.Millisecond).WillReturnError(errors.New("starting"))
	mock.ExpectPing().WillReturnError(nil)

	runner := NewMigrationRunner(db)

	start := time.Now()
	err = runner.WaitForDatabase()
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Greater(t, duration, 300*time.Millisecond, "Should have waited for retries")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLoadSeeds_ReadFileError(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "seeds-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a directory instead of a file (will cause read error)
	seedDir := filepath.Join(tempDir, "001_invalid.sql")
	err = os.Mkdir(seedDir, 0755)
	require.NoError(t, err)

	// Enable seeding
	originalValue := os.Getenv("SEED_DATABASE")
	os.Setenv("SEED_DATABASE", "true")
	defer os.Setenv("SEED_DATABASE", originalValue)

	runner := &MigrationRunner{
		db:             db,
		migrationsPath: migrationsPath,
		seedsPath:      tempDir,
	}

	err = runner.LoadSeeds()

	// Should return error when unable to read file
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read seed file")
}

func TestGetMigrationStatus_DirectoryNotFound(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	runner := &MigrationRunner{
		db:             db,
		migrationsPath: "/nonexistent/migrations",
		seedsPath:      seedsPath,
	}

	_, _, err = runner.GetMigrationStatus()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "migrations directory not found")
}
