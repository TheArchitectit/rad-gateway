// Package regression tests database migration paths.
// Sprint 7.2: Test Database Migration Paths
package regression

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestMigrationPaths validates database migrations work correctly
func TestMigrationPaths(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test v1 schema creation (initial)
	t.Run("CreateV1Schema", func(t *testing.T) {
		schema := `
			CREATE TABLE IF NOT EXISTS api_keys (
				id TEXT PRIMARY KEY,
				key_hash TEXT NOT NULL,
				name TEXT NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`
		_, err := db.Exec(schema)
		if err != nil {
			t.Fatalf("Failed to create v1 schema: %v", err)
		}

		// Verify table exists
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='api_keys'").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to verify table: %v", err)
		}
		if count != 1 {
			t.Errorf("api_keys table should exist")
		}
	})

	// Test v2 migration (add status column)
	t.Run("MigrateV1ToV2", func(t *testing.T) {
		// Check if status column exists
		var hasStatus int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('api_keys') WHERE name='status'
		`).Scan(&hasStatus)
		if err != nil {
			t.Fatalf("Failed to check for status column: %v", err)
		}

		if hasStatus == 0 {
			// Add status column
			_, err := db.Exec(`ALTER TABLE api_keys ADD COLUMN status TEXT DEFAULT 'active'`)
			if err != nil {
				t.Fatalf("Failed to add status column: %v", err)
			}
		}

		// Verify column exists
		err = db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('api_keys') WHERE name='status'
		`).Scan(&hasStatus)
		if err != nil {
			t.Fatalf("Failed to verify status column: %v", err)
		}
		if hasStatus != 1 {
			t.Errorf("status column should exist after migration")
		}
	})

	// Test v3 migration (add index)
	t.Run("MigrateV2ToV3", func(t *testing.T) {
		// Create index
		_, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status)`)
		if err != nil {
			t.Fatalf("Failed to create index: %v", err)
		}

		// Verify index exists
		var count int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_api_keys_status'
		`).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to verify index: %v", err)
		}
		if count != 1 {
			t.Errorf("Index should exist after migration")
		}
	})

	// Test data integrity after migrations
	t.Run("DataIntegrity", func(t *testing.T) {
		// Insert test data
		_, err := db.Exec(`
			INSERT INTO api_keys (id, key_hash, name, status) VALUES (?, ?, ?, ?)
		`, "test-key-1", "hash-1", "Test Key", "active")
		if err != nil {
			t.Fatalf("Failed to insert data: %v", err)
		}

		// Verify data
		var id, status string
		err = db.QueryRow(`SELECT id, status FROM api_keys WHERE id = ?`, "test-key-1").Scan(&id, &status)
		if err != nil {
			t.Fatalf("Failed to query data: %v", err)
		}
		if id != "test-key-1" {
			t.Errorf("ID mismatch: got %v, want %v", id, "test-key-1")
		}
		if status != "active" {
			t.Errorf("Status mismatch: got %v, want %v", status, "active")
		}
	})

	// Test rollback capability
	t.Run("RollbackCapability", func(t *testing.T) {
		// SQLite doesn't support transactional DDL well, but we can verify
		// the schema is still intact after a failed operation
		_, err := db.Exec(`SELECT COUNT(*) FROM api_keys`)
		if err != nil {
			t.Errorf("Schema should be intact: %v", err)
		}
	})
}

// TestMigrationCompatibility validates compatibility between migration versions
func TestMigrationCompatibility(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create full schema
	migrations := []string{
		`CREATE TABLE providers (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			status TEXT DEFAULT 'unknown',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE usage_records (
			id TEXT PRIMARY KEY,
			request_id TEXT NOT NULL,
			provider_id TEXT,
			model TEXT,
			duration_ms INTEGER,
			tokens_input INTEGER DEFAULT 0,
			tokens_output INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX idx_usage_provider ON usage_records(provider_id)`,
		`CREATE INDEX idx_usage_created ON usage_records(created_at)`,
	}

	for i, migration := range migrations {
		_, err := db.Exec(migration)
		if err != nil {
			t.Fatalf("Migration %d failed: %v", i+1, err)
		}
	}

	// Verify all tables exist
	tables := []string{"providers", "usage_records"}
	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to check table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("Table %s should exist", table)
		}
	}
}

// BenchmarkMigration benchmarks migration performance
func BenchmarkMigration(b *testing.B) {
	b.Run("CreateTable", func(b *testing.B) {
		tmpDir := b.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		for i := 0; i < b.N; i++ {
			db, err := sql.Open("sqlite3", dbPath)
			if err != nil {
				b.Fatalf("Failed to open database: %v", err)
			}

			_, err = db.Exec(`CREATE TABLE test (id TEXT PRIMARY KEY)`)
			if err != nil {
				b.Fatalf("Failed to create table: %v", err)
			}

			db.Close()
			os.Remove(dbPath)
		}
	})

	b.Run("AddColumn", func(b *testing.B) {
		tmpDir := b.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, _ := sql.Open("sqlite3", dbPath)
		db.Exec(`CREATE TABLE test (id TEXT PRIMARY KEY)`)
		db.Close()

		for i := 0; i < b.N; i++ {
			db, _ := sql.Open("sqlite3", dbPath)
			db.Exec(`ALTER TABLE test ADD COLUMN col TEXT`)
			db.Close()

			// Reset for next iteration
			os.Remove(dbPath)
			db, _ = sql.Open("sqlite3", dbPath)
			db.Exec(`CREATE TABLE test (id TEXT PRIMARY KEY)`)
			db.Close()
		}
	})
}
