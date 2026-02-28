package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewBackupManager(t *testing.T) {
	bm := NewBackupManager("postgres://localhost/test")

	if bm.dsn != "postgres://localhost/test" {
		t.Errorf("dsn = %s, want postgres://localhost/test", bm.dsn)
	}
	if bm.backupDir != "/var/backups/radgateway" {
		t.Errorf("backupDir = %s, want /var/backups/radgateway", bm.backupDir)
	}
	if bm.retention != 7 {
		t.Errorf("retention = %d, want 7", bm.retention)
	}
}

func TestBackupManager_ListBackups_Empty(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()
	bm := NewBackupManager("postgres://localhost/test")
	bm.backupDir = tempDir

	backups, err := bm.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups() error = %v", err)
	}

	if len(backups) != 0 {
		t.Errorf("len(backups) = %d, want 0", len(backups))
	}
}

func TestBackupManager_ListBackups(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()
	bm := NewBackupManager("postgres://localhost/test")
	bm.backupDir = tempDir

	// Create test backup files
	testFiles := []string{
		"radgateway-20260101-120000.sql",
		"radgateway-20260102-120000.sql",
	}
	for _, name := range testFiles {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte("test backup"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	backups, err := bm.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups() error = %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("len(backups) = %d, want 2", len(backups))
	}
}

func TestBackupManager_CleanupOldBackups(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()
	bm := NewBackupManager("postgres://localhost/test")
	bm.backupDir = tempDir

	// Create old backup file (set modification time to 10 days ago)
	oldFile := filepath.Join(tempDir, "radgateway-20260101-120000.sql")
	if err := os.WriteFile(oldFile, []byte("old backup"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	oldTime := time.Now().AddDate(0, 0, -10)
	os.Chtimes(oldFile, oldTime, oldTime)

	// Create recent backup file
	recentFile := filepath.Join(tempDir, "radgateway-20260109-120000.sql")
	if err := os.WriteFile(recentFile, []byte("recent backup"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run cleanup
	if err := bm.CleanupOldBackups(); err != nil {
		t.Fatalf("CleanupOldBackups() error = %v", err)
	}

	// Verify old file is deleted
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("Old backup file should be deleted")
	}

	// Verify recent file still exists
	if _, err := os.Stat(recentFile); os.IsNotExist(err) {
		t.Error("Recent backup file should still exist")
	}
}

func TestBackupInfo(t *testing.T) {
	info := BackupInfo{
		Path:      "/var/backups/radgateway/test.sql",
		Size:      1024,
		CreatedAt: time.Now(),
	}

	if info.Path != "/var/backups/radgateway/test.sql" {
		t.Errorf("Path = %s, want /var/backups/radgateway/test.sql", info.Path)
	}
	if info.Size != 1024 {
		t.Errorf("Size = %d, want 1024", info.Size)
	}
}
