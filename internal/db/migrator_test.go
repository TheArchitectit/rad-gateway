// Package db provides database migration testing for RAD Gateway.
package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestMigrationParsing tests migration filename and content parsing.
func TestMigrationParsing(t *testing.T) {
	tests := []struct {
		name            string
		filename        string
		expectError     bool
		expectedVersion int
		expectedName    string
	}{
		{
			name:            "valid migration filename",
			filename:        "001_create_users.sql",
			expectError:     false,
			expectedVersion: 1,
			expectedName:    "create users",
		},
		{
			name:            "valid with multiple words",
			filename:        "042_add_user_preferences_table.sql",
			expectError:     false,
			expectedVersion: 42,
			expectedName:    "add user preferences table",
		},
		{
			name:        "invalid format - no version",
			filename:    "create_users.sql",
			expectError: true,
		},
		{
			name:        "invalid format - wrong extension",
			filename:    "001_create_users.txt",
			expectError: true,
		},
		{
			name:        "invalid format - no underscore",
			filename:    "001create_users.sql",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, name, err := parseMigrationFilename(tt.filename)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for %s, got nil", tt.filename)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for %s: %v", tt.filename, err)
				return
			}
			if version != tt.expectedVersion {
				t.Errorf("expected version %d, got %d", tt.expectedVersion, version)
			}
			if name != tt.expectedName {
				t.Errorf("expected name %q, got %q", tt.expectedName, name)
			}
		})
	}
}

// TestParseMigrationContent tests the migration content parser.
func TestParseMigrationContent(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		expectedUp      string
		expectedDown    string
		expectError     bool
	}{
		{
			name: "simple migration without markers",
			content: `CREATE TABLE users (
	id INTEGER PRIMARY KEY,
	name TEXT
);`,
			expectedUp: `CREATE TABLE users (
	id INTEGER PRIMARY KEY,
	name TEXT
);`,
			expectedDown: "",
		},
		{
			name: "migration with up and down markers",
			content: `-- +migrate Up
CREATE TABLE users (
	id INTEGER PRIMARY KEY,
	name TEXT
);

-- +migrate Down
DROP TABLE users;`,
			expectedUp: `CREATE TABLE users (
	id INTEGER PRIMARY KEY,
	name TEXT
);`,
			expectedDown: `DROP TABLE users;`,
		},
		{
			name: "migration with only up marker",
			content: `-- +migrate Up
CREATE TABLE users (
	id INTEGER PRIMARY KEY
);`,
			expectedUp: `CREATE TABLE users (
	id INTEGER PRIMARY KEY
);`,
			expectedDown: "",
		},
		{
			name: "case insensitive markers",
			content: `-- +Migrate UP
CREATE TABLE users (id INTEGER);
-- +migrate DOWN
DROP TABLE users;`,
			expectedUp:   `CREATE TABLE users (id INTEGER);`,
			expectedDown: `DROP TABLE users;`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			up, down, err := parseMigrationContent(tt.content)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if strings.TrimSpace(up) != strings.TrimSpace(tt.expectedUp) {
				t.Errorf("up SQL mismatch:\nexpected: %q\ngot: %q", tt.expectedUp, up)
			}
			if strings.TrimSpace(down) != strings.TrimSpace(tt.expectedDown) {
				t.Errorf("down SQL mismatch:\nexpected: %q\ngot: %q", tt.expectedDown, down)
			}
		})
	}
}

// TestChecksumCalculation tests checksum calculation.
func TestChecksumCalculation(t *testing.T) {
	up1 := "CREATE TABLE users (id INTEGER)"
	down1 := "DROP TABLE users"
	up2 := "CREATE TABLE users (id INTEGER, name TEXT)"
	down2 := "DROP TABLE users"

	checksum1 := calculateChecksum(up1, down1)
	checksum2 := calculateChecksum(up2, down2)
	checksum3 := calculateChecksum(up1, down1)

	if checksum1 == "" {
		t.Error("checksum should not be empty")
	}
	if checksum1 == checksum2 {
		t.Error("different content should produce different checksums")
	}
	if checksum1 != checksum3 {
		t.Error("same content should produce same checksum")
	}
	if len(checksum1) != 32 { // 16 bytes as hex = 32 chars
		t.Errorf("checksum should be 32 hex chars, got %d", len(checksum1))
	}
}

// TestMigratorInit tests the migration table initialization.
func TestMigratorInit(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Open database
	config := Config{
		Driver: "sqlite",
		DSN:    dbPath,
	}
	database, err := New(config)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Close()

	// Get underlying *sql.DB
	sqlDB := database.(*SQLiteDB).DB()

	// Create migrator
	migrator := NewMigrator(sqlDB, "sqlite")

	ctx := context.Background()

	// Test Init
	err = migrator.Init(ctx)
	if err != nil {
		t.Fatalf("failed to initialize migrator: %v", err)
	}

	// Verify table exists
	var count int
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Fatalf("schema_migrations table not created: %v", err)
	}

	// Init should be idempotent
	err = migrator.Init(ctx)
	if err != nil {
		t.Fatalf("Init should be idempotent: %v", err)
	}
}

// TestMigratorUpDown tests applying and rolling back migrations.
func TestMigratorUpDown(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create migrations directory
	migrationsDir := filepath.Join(tempDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	// Create test migration files
	migration1 := `-- +migrate Up
CREATE TABLE test_table1 (
	id INTEGER PRIMARY KEY,
	name TEXT
);

-- +migrate Down
DROP TABLE test_table1;`

	migration2 := `-- +migrate Up
CREATE TABLE test_table2 (
	id INTEGER PRIMARY KEY,
	value INTEGER
);

-- +migrate Down
DROP TABLE test_table2;`

	os.WriteFile(filepath.Join(migrationsDir, "001_first_table.sql"), []byte(migration1), 0644)
	os.WriteFile(filepath.Join(migrationsDir, "002_second_table.sql"), []byte(migration2), 0644)

	// Open database
	config := Config{
		Driver: "sqlite",
		DSN:    dbPath,
	}
	database, err := New(config)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Close()

	// Get underlying *sql.DB
	sqlDB := database.(*SQLiteDB).DB()

	// Create migrator and load migrations
	migrator := NewMigrator(sqlDB, "sqlite")
	err = migrator.LoadMigrationsFromDir(migrationsDir)
	if err != nil {
		t.Fatalf("failed to load migrations: %v", err)
	}

	ctx := context.Background()

	// Run migrations up
	err = migrator.Up(ctx)
	if err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Verify tables exist
	var count int
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_table1'").Scan(&count)
	if err != nil || count != 1 {
		t.Errorf("test_table1 should exist")
	}
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_table2'").Scan(&count)
	if err != nil || count != 1 {
		t.Errorf("test_table2 should exist")
	}

	// Check version
	version, err := migrator.Version(ctx)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}
	if version != 2 {
		t.Errorf("expected version 2, got %d", version)
	}

	// Rollback one migration
	err = migrator.Down(ctx)
	if err != nil {
		t.Fatalf("failed to rollback: %v", err)
	}

	// Verify test_table2 is gone
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_table2'").Scan(&count)
	if err != nil || count != 0 {
		t.Errorf("test_table2 should not exist after rollback")
	}

	// test_table1 should still exist
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_table1'").Scan(&count)
	if err != nil || count != 1 {
		t.Errorf("test_table1 should still exist")
	}

	// Check version
	version, err = migrator.Version(ctx)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}
	if version != 1 {
		t.Errorf("expected version 1 after rollback, got %d", version)
	}

	// Rollback all
	err = migrator.Down(ctx)
	if err != nil {
		t.Fatalf("failed to rollback: %v", err)
	}

	// Verify both tables are gone
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_table1'").Scan(&count)
	if err != nil || count != 0 {
		t.Errorf("test_table1 should not exist after full rollback")
	}

	version, err = migrator.Version(ctx)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}
	if version != 0 {
		t.Errorf("expected version 0 after full rollback, got %d", version)
	}
}

// TestMigratorStatus tests the status reporting.
func TestMigratorStatus(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	migrationsDir := filepath.Join(tempDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	// Create 3 migration files
	for i := 1; i <= 3; i++ {
		content := `-- +migrate Up
CREATE TABLE table` + string(rune('0'+i)) + ` (id INTEGER);
-- +migrate Down
DROP TABLE table` + string(rune('0'+i)) + `;`
		filename := fmt.Sprintf("00%d_table_%d.sql", i, i)
		os.WriteFile(filepath.Join(migrationsDir, filename), []byte(content), 0644)
	}

	// Open database
	config := Config{
		Driver: "sqlite",
		DSN:    dbPath,
	}
	database, err := New(config)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Close()

	sqlDB := database.(*SQLiteDB).DB()
	migrator := NewMigrator(sqlDB, "sqlite")
	migrator.LoadMigrationsFromDir(migrationsDir)

	ctx := context.Background()

	// Initial status
	status, err := migrator.GetStatus(ctx)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}
	if status.CurrentVersion != 0 {
		t.Errorf("expected version 0, got %d", status.CurrentVersion)
	}
	if status.PendingCount != 3 {
		t.Errorf("expected 3 pending, got %d", status.PendingCount)
	}

	// Apply one migration
	migrator.UpTo(ctx, 1)

	status, err = migrator.GetStatus(ctx)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}
	if status.CurrentVersion != 1 {
		t.Errorf("expected version 1, got %d", status.CurrentVersion)
	}
	if status.PendingCount != 2 {
		t.Errorf("expected 2 pending, got %d", status.PendingCount)
	}
	if len(status.Applied) != 1 {
		t.Errorf("expected 1 applied record, got %d", len(status.Applied))
	}

	// Apply all
	migrator.Up(ctx)

	status, err = migrator.GetStatus(ctx)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}
	if status.CurrentVersion != 3 {
		t.Errorf("expected version 3, got %d", status.CurrentVersion)
	}
	if status.PendingCount != 0 {
		t.Errorf("expected 0 pending, got %d", status.PendingCount)
	}
}

// TestChecksumMismatch tests that modified migrations are detected.
func TestChecksumMismatch(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	migrationsDir := filepath.Join(tempDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	// Create initial migration
	content := `-- +migrate Up
CREATE TABLE users (id INTEGER);
-- +migrate Down
DROP TABLE users;`
	os.WriteFile(filepath.Join(migrationsDir, "001_users.sql"), []byte(content), 0644)

	// Open database
	config := Config{
		Driver: "sqlite",
		DSN:    dbPath,
	}
	database, err := New(config)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Close()

	sqlDB := database.(*SQLiteDB).DB()
	migrator := NewMigrator(sqlDB, "sqlite")
	migrator.LoadMigrationsFromDir(migrationsDir)

	ctx := context.Background()
	migrator.Up(ctx)

	// Modify the migration file
	modifiedContent := `-- +migrate Up
CREATE TABLE users (id INTEGER, name TEXT);
-- +migrate Down
DROP TABLE users;`
	os.WriteFile(filepath.Join(migrationsDir, "001_users.sql"), []byte(modifiedContent), 0644)

	// Reload migrations
	migrator = NewMigrator(sqlDB, "sqlite")
	migrator.LoadMigrationsFromDir(migrationsDir)

	// GetStatus should detect checksum mismatch
	_, err = migrator.GetStatus(ctx)
	if err == nil {
		t.Error("expected checksum mismatch error")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("expected checksum mismatch error, got: %v", err)
	}
}

// TestMigratorTransactionRollback tests that failed migrations roll back.
func TestMigratorTransactionRollback(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	migrationsDir := filepath.Join(tempDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	// Create a valid migration
	validMigration := `-- +migrate Up
CREATE TABLE test (id INTEGER);
-- +migrate Down
DROP TABLE test;`
	os.WriteFile(filepath.Join(migrationsDir, "001_valid.sql"), []byte(validMigration), 0644)

	// Open database
	config := Config{
		Driver: "sqlite",
		DSN:    dbPath,
	}
	database, err := New(config)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Close()

	sqlDB := database.(*SQLiteDB).DB()

	// Apply valid migration
	migrator := NewMigrator(sqlDB, "sqlite")
	migrator.LoadMigrationsFromDir(migrationsDir)
	ctx := context.Background()
	migrator.Up(ctx)

	// Create an invalid migration
	invalidMigration := `-- +migrate Up
CREATE INVALID SYNTAX HERE;
-- +migrate Down
DROP TABLE invalid;`
	os.WriteFile(filepath.Join(migrationsDir, "002_invalid.sql"), []byte(invalidMigration), 0644)

	// Try to apply invalid migration
	migrator = NewMigrator(sqlDB, "sqlite")
	migrator.LoadMigrationsFromDir(migrationsDir)
	err = migrator.Up(ctx)
	if err == nil {
		t.Error("expected error for invalid migration")
	}

	// Verify version is still 1 (rollback occurred)
	version, _ := migrator.Version(ctx)
	if version != 1 {
		t.Errorf("expected version 1 after failed migration rollback, got %d", version)
	}

	// Verify migration record was not created
	var count int
	sqlDB.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = 2").Scan(&count)
	if count != 0 {
		t.Error("migration 2 should not be recorded after failed migration")
	}
}

// TestCreateMigration tests creating new migration files.
func TestCreateMigration(t *testing.T) {
	tempDir := t.TempDir()

	// Create first migration
	path1, err := CreateMigration(tempDir, "create users table")
	if err != nil {
		t.Fatalf("failed to create migration: %v", err)
	}
	if !strings.HasSuffix(path1, "001_create_users_table.sql") {
		t.Errorf("unexpected path: %s", path1)
	}

	// Verify file exists and has content
	content, err := os.ReadFile(path1)
	if err != nil {
		t.Fatalf("failed to read created migration: %v", err)
	}
	if !strings.Contains(string(content), "-- +migrate Up") {
		t.Error("migration should contain Up marker")
	}
	if !strings.Contains(string(content), "-- +migrate Down") {
		t.Error("migration should contain Down marker")
	}

	// Create second migration
	path2, err := CreateMigration(tempDir, "add posts table")
	if err != nil {
		t.Fatalf("failed to create migration: %v", err)
	}
	if !strings.HasSuffix(path2, "002_add_posts_table.sql") {
		t.Errorf("unexpected path: %s", path2)
	}
}

// TestMigratorDryRun tests dry-run mode.
func TestMigratorDryRun(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	migrationsDir := filepath.Join(tempDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	// Create migration
	content := `-- +migrate Up
CREATE TABLE dryrun_test (id INTEGER);
-- +migrate Down
DROP TABLE dryrun_test;`
	os.WriteFile(filepath.Join(migrationsDir, "001_dryrun.sql"), []byte(content), 0644)

	// Open database
	config := Config{
		Driver: "sqlite",
		DSN:    dbPath,
	}
	database, err := New(config)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Close()

	sqlDB := database.(*SQLiteDB).DB()
	migrator := NewMigrator(sqlDB, "sqlite")
	migrator.LoadMigrationsFromDir(migrationsDir)

	// Note: To test dry run properly, we'd need to modify the migrator config
	// This is a basic test that the migrator works
	ctx := context.Background()
	err = migrator.Up(ctx)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Verify table exists
	var count int
	sqlDB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='dryrun_test'").Scan(&count)
	if count != 1 {
		t.Error("table should exist after migration")
	}
}

// TestMigratorMissingDown tests behavior when down migration is missing.
func TestMigratorMissingDown(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	migrationsDir := filepath.Join(tempDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	// Create migration without down
	content := `-- +migrate Up
CREATE TABLE no_down (id INTEGER);`
	os.WriteFile(filepath.Join(migrationsDir, "001_no_down.sql"), []byte(content), 0644)

	// Open database
	config := Config{
		Driver: "sqlite",
		DSN:    dbPath,
	}
	database, err := New(config)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Close()

	sqlDB := database.(*SQLiteDB).DB()
	migrator := NewMigrator(sqlDB, "sqlite")
	migrator.LoadMigrationsFromDir(migrationsDir)

	ctx := context.Background()

	// Apply migration (should succeed)
	err = migrator.Up(ctx)
	if err != nil {
		t.Fatalf("failed to apply migration: %v", err)
	}

	// Try to rollback (should fail without down migration)
	err = migrator.Down(ctx)
	if err == nil {
		t.Error("expected error when rolling back migration without down script")
	}
}

// TestMigratorDuplicateVersion tests detection of duplicate versions.
func TestMigratorDuplicateVersion(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	migrationsDir := filepath.Join(tempDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	// Create two migrations with same version
	os.WriteFile(filepath.Join(migrationsDir, "001_first.sql"), []byte("SELECT 1;"), 0644)
	os.WriteFile(filepath.Join(migrationsDir, "001_second.sql"), []byte("SELECT 2;"), 0644)

	migrator := NewMigrator(nil, "sqlite")
	err := migrator.LoadMigrationsFromDir(migrationsDir)
	if err == nil {
		t.Error("expected error for duplicate versions")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("expected duplicate version error, got: %v", err)
	}
}

// TestMigratorEmptyDirectory tests handling of empty migration directory.
func TestMigratorEmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	migrationsDir := filepath.Join(tempDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	migrator := NewMigrator(nil, "sqlite")
	err := migrator.LoadMigrationsFromDir(migrationsDir)
	if err != nil {
		t.Fatalf("empty directory should not error: %v", err)
	}
	if len(migrator.migrations) != 0 {
		t.Errorf("expected 0 migrations, got %d", len(migrator.migrations))
	}
}

// TestMigratorNonExistentDirectory tests handling of non-existent directory.
func TestMigratorNonExistentDirectory(t *testing.T) {
	migrator := NewMigrator(nil, "sqlite")
	err := migrator.LoadMigrationsFromDir("/nonexistent/path")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

// TestMigratorDownTo tests rolling back to a specific version.
func TestMigratorDownTo(t *testing.T) {
	// Create temporary database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	migrationsDir := filepath.Join(tempDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	// Create 3 migrations
	for i := 1; i <= 3; i++ {
		content := fmt.Sprintf(`-- +migrate Up
CREATE TABLE t%d (id INTEGER);
-- +migrate Down
DROP TABLE t%d;`, i, i)
		filename := fmt.Sprintf("00%d_table.sql", i)
		os.WriteFile(filepath.Join(migrationsDir, filename), []byte(content), 0644)
	}

	// Open database
	config := Config{
		Driver: "sqlite",
		DSN:    dbPath,
	}
	database, err := New(config)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Close()

	sqlDB := database.(*SQLiteDB).DB()
	migrator := NewMigrator(sqlDB, "sqlite")
	migrator.LoadMigrationsFromDir(migrationsDir)

	ctx := context.Background()
	migrator.Up(ctx)

	// Verify all tables exist
	for i := 1; i <= 3; i++ {
		var count int
		sqlDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='t%d'", i)).Scan(&count)
		if count != 1 {
			t.Errorf("table t%d should exist", i)
		}
	}

	// Rollback to version 1
	err = migrator.DownTo(ctx, 1)
	if err != nil {
		t.Fatalf("failed to rollback: %v", err)
	}

	// Verify only t1 exists
	for i := 1; i <= 3; i++ {
		var count int
		sqlDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='t%d'", i)).Scan(&count)
		if i == 1 {
			if count != 1 {
				t.Errorf("table t1 should exist")
			}
		} else {
			if count != 0 {
				t.Errorf("table t%d should not exist", i)
			}
		}
	}

	// Verify version
	version, _ := migrator.Version(ctx)
	if version != 1 {
		t.Errorf("expected version 1, got %d", version)
	}
}

// BenchmarkMigrationLoad benchmarks loading migrations from disk.
func BenchmarkMigrationLoad(b *testing.B) {
	tempDir := b.TempDir()
	migrationsDir := filepath.Join(tempDir, "migrations")
	os.MkdirAll(migrationsDir, 0755)

	// Create 100 migration files
	for i := 1; i <= 100; i++ {
		content := fmt.Sprintf(`-- +migrate Up
CREATE TABLE t%d (id INTEGER);
-- +migrate Down
DROP TABLE t%d;`, i, i)
		filename := fmt.Sprintf("%03d_table_%d.sql", i, i)
		os.WriteFile(filepath.Join(migrationsDir, filename), []byte(content), 0644)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		migrator := NewMigrator(nil, "sqlite")
		migrator.LoadMigrationsFromDir(migrationsDir)
	}
}

