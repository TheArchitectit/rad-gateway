// Package repository provides high-level data access abstractions for RAD Gateway.
package repository

import (
	"context"
	"radgateway/internal/db"
)

// Repository is the main repository container that provides access to all entity repositories.
type Repository struct {
	db db.Database
}

// New creates a new Repository instance wrapping the provided database.
func New(database db.Database) *Repository {
	return &Repository{db: database}
}

// Database returns the underlying database interface.
func (r *Repository) Database() db.Database {
	return r.db
}

// Workspaces provides access to workspace operations.
func (r *Repository) Workspaces() db.WorkspaceRepository {
	return r.db.Workspaces()
}

// Users provides access to user operations.
func (r *Repository) Users() db.UserRepository {
	return r.db.Users()
}

// Roles provides access to role operations.
func (r *Repository) Roles() db.RoleRepository {
	return r.db.Roles()
}

// Permissions provides access to permission operations.
func (r *Repository) Permissions() db.PermissionRepository {
	return r.db.Permissions()
}

// Tags provides access to tag operations.
func (r *Repository) Tags() db.TagRepository {
	return r.db.Tags()
}

// Providers provides access to provider operations.
func (r *Repository) Providers() db.ProviderRepository {
	return r.db.Providers()
}

// ControlRooms provides access to control room operations.
func (r *Repository) ControlRooms() db.ControlRoomRepository {
	return r.db.ControlRooms()
}

// APIKeys provides access to API key operations.
func (r *Repository) APIKeys() db.APIKeyRepository {
	return r.db.APIKeys()
}

// Quotas provides access to quota operations.
func (r *Repository) Quotas() db.QuotaRepository {
	return r.db.Quotas()
}

// UsageRecords provides access to usage record operations.
func (r *Repository) UsageRecords() db.UsageRecordRepository {
	return r.db.UsageRecords()
}

// TraceEvents provides access to trace event operations.
func (r *Repository) TraceEvents() db.TraceEventRepository {
	return r.db.TraceEvents()
}

// Ping checks the database connection.
func (r *Repository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}

// Close closes the database connection.
func (r *Repository) Close() error {
	return r.db.Close()
}

// RunMigrations executes all pending database migrations.
func (r *Repository) RunMigrations() error {
	return r.db.RunMigrations()
}

// Version returns the current database schema version.
func (r *Repository) Version() (int, error) {
	return r.db.Version()
}

// Config holds repository configuration.
type Config = db.Config

// New creates a new repository with the specified configuration.
// This is a convenience wrapper around db.New.
func NewWithConfig(config Config) (*Repository, error) {
	database, err := db.New(config)
	if err != nil {
		return nil, err
	}
	return New(database), nil
}
