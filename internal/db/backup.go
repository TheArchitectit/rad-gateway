// Package db provides database interfaces and implementations for RAD Gateway.
package db

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// BackupManager handles database backup and restore
type BackupManager struct {
	dsn       string
	backupDir string
	retention int // days
}

// NewBackupManager creates a new backup manager
func NewBackupManager(dsn string) *BackupManager {
	return &BackupManager{
		dsn:       dsn,
		backupDir: "/var/backups/radgateway",
		retention: 7,
	}
}

// Backup performs a database backup using pg_dump
func (bm *BackupManager) Backup(ctx context.Context, outputPath string) (string, error) {
	// Create backup directory if not exists
	if err := os.MkdirAll(bm.backupDir, 0755); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}

	// Generate backup filename with timestamp
	if outputPath == "" {
		timestamp := time.Now().Format("20060102-150405")
		outputPath = filepath.Join(bm.backupDir, fmt.Sprintf("radgateway-%s.sql", timestamp))
	}

	// Run pg_dump
	cmd := exec.CommandContext(ctx, "pg_dump",
		"--format=plain",
		"--verbose",
		"--file="+outputPath,
		bm.dsn,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pg_dump failed: %w\nOutput: %s", err, output)
	}

	return outputPath, nil
}

// Restore restores database from backup
func (bm *BackupManager) Restore(ctx context.Context, backupPath string) error {
	cmd := exec.CommandContext(ctx, "psql",
		"--set=ON_ERROR_STOP=on",
		"--file="+backupPath,
		bm.dsn,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restore failed: %w\nOutput: %s", err, output)
	}

	return nil
}

// CleanupOldBackups removes backups older than retention days
func (bm *BackupManager) CleanupOldBackups() error {
	cutoff := time.Now().AddDate(0, 0, -bm.retention)

	entries, err := os.ReadDir(bm.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No backup dir yet
		}
		return err
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(bm.backupDir, entry.Name())
			os.Remove(path)
		}
	}

	return nil
}

// ListBackups returns list of available backups
func (bm *BackupManager) ListBackups() ([]BackupInfo, error) {
	entries, err := os.ReadDir(bm.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, err
	}

	var backups []BackupInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		backups = append(backups, BackupInfo{
			Path:      filepath.Join(bm.backupDir, entry.Name()),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}

	return backups, nil
}

// BackupInfo holds backup metadata
type BackupInfo struct {
	Path      string
	Size      int64
	CreatedAt time.Time
}
