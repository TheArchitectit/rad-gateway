// Package db provides database interfaces and implementations for RAD Gateway.
package db

import (
	"context"
	"database/sql"
	"time"
)

// Database is the main interface for database operations.
type Database interface {
	// Connection management
	Ping(ctx context.Context) error
	Close() error
	DB() *sql.DB // Returns underlying *sql.DB for audit logging

	// Transaction support
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)

	// Raw query execution
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row

	// Repository accessors
	Workspaces() WorkspaceRepository
	Users() UserRepository
	Roles() RoleRepository
	Permissions() PermissionRepository
	Tags() TagRepository
	Providers() ProviderRepository
	ControlRooms() ControlRoomRepository
	APIKeys() APIKeyRepository
	Quotas() QuotaRepository
	UsageRecords() UsageRecordRepository
	TraceEvents() TraceEventRepository
	ModelCards() ModelCardRepository
	AuditLog() AuditLogRepository

	// Migration support
	RunMigrations() error
	Version() (int, error)
}

// WorkspaceRepository defines workspace data access operations.
type WorkspaceRepository interface {
	Create(ctx context.Context, workspace *Workspace) error
	GetByID(ctx context.Context, id string) (*Workspace, error)
	GetBySlug(ctx context.Context, slug string) (*Workspace, error)
	Update(ctx context.Context, workspace *Workspace) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]Workspace, error)
}

// UserRepository defines user data access operations.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
	UpdateLastLogin(ctx context.Context, id string, t time.Time) error
}

// RoleRepository defines role data access operations.
type RoleRepository interface {
	Create(ctx context.Context, role *Role) error
	GetByID(ctx context.Context, id string) (*Role, error)
	GetByWorkspace(ctx context.Context, workspaceID *string) ([]Role, error)
	Update(ctx context.Context, role *Role) error
	Delete(ctx context.Context, id string) error
	AssignToUser(ctx context.Context, userID, roleID string, grantedBy *string, expiresAt *time.Time) error
	RemoveFromUser(ctx context.Context, userID, roleID string) error
	GetUserRoles(ctx context.Context, userID string) ([]Role, error)
}

// PermissionRepository defines permission data access operations.
type PermissionRepository interface {
	Create(ctx context.Context, permission *Permission) error
	GetByID(ctx context.Context, id string) (*Permission, error)
	GetByName(ctx context.Context, name string) (*Permission, error)
	List(ctx context.Context) ([]Permission, error)
	AssignToRole(ctx context.Context, roleID, permissionID string) error
	RemoveFromRole(ctx context.Context, roleID, permissionID string) error
	GetRolePermissions(ctx context.Context, roleID string) ([]Permission, error)
	GetUserPermissions(ctx context.Context, userID string) ([]Permission, error)
}

// TagRepository defines tag data access operations.
type TagRepository interface {
	Create(ctx context.Context, tag *Tag) error
	GetByID(ctx context.Context, id string) (*Tag, error)
	GetByCategoryValue(ctx context.Context, workspaceID, category, value string) (*Tag, error)
	GetByWorkspace(ctx context.Context, workspaceID string) ([]Tag, error)
	Delete(ctx context.Context, id string) error
	AssignToProvider(ctx context.Context, providerID, tagID string) error
	RemoveFromProvider(ctx context.Context, providerID, tagID string) error
	GetProviderTags(ctx context.Context, providerID string) ([]Tag, error)
	AssignToAPIKey(ctx context.Context, apiKeyID, tagID string) error
	RemoveFromAPIKey(ctx context.Context, apiKeyID, tagID string) error
	GetAPIKeyTags(ctx context.Context, apiKeyID string) ([]Tag, error)
}

// ProviderRepository defines provider data access operations.
type ProviderRepository interface {
	Create(ctx context.Context, provider *Provider) error
	GetByID(ctx context.Context, id string) (*Provider, error)
	GetBySlug(ctx context.Context, workspaceID, slug string) (*Provider, error)
	GetByWorkspace(ctx context.Context, workspaceID string) ([]Provider, error)
	GetByTags(ctx context.Context, workspaceID string, tagIDs []string) ([]Provider, error)
	Update(ctx context.Context, provider *Provider) error
	Delete(ctx context.Context, id string) error
	UpdateHealth(ctx context.Context, health *ProviderHealth) error
	GetHealth(ctx context.Context, providerID string) (*ProviderHealth, error)
	UpdateCircuitBreaker(ctx context.Context, state *CircuitBreakerState) error
	GetCircuitBreaker(ctx context.Context, providerID string) (*CircuitBreakerState, error)
}

// ControlRoomRepository defines control room data access operations.
type ControlRoomRepository interface {
	Create(ctx context.Context, room *ControlRoom) error
	GetByID(ctx context.Context, id string) (*ControlRoom, error)
	GetBySlug(ctx context.Context, workspaceID, slug string) (*ControlRoom, error)
	GetByWorkspace(ctx context.Context, workspaceID string) ([]ControlRoom, error)
	Update(ctx context.Context, room *ControlRoom) error
	Delete(ctx context.Context, id string) error
	GrantAccess(ctx context.Context, access *ControlRoomAccess) error
	RevokeAccess(ctx context.Context, controlRoomID, userID string) error
	GetUserAccess(ctx context.Context, controlRoomID string) ([]ControlRoomAccess, error)
}

// APIKeyRepository defines API key data access operations.
type APIKeyRepository interface {
	Create(ctx context.Context, key *APIKey) error
	GetByID(ctx context.Context, id string) (*APIKey, error)
	GetByHash(ctx context.Context, hash string) (*APIKey, error)
	GetByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]APIKey, error)
	Update(ctx context.Context, key *APIKey) error
	Delete(ctx context.Context, id string) error
	UpdateLastUsed(ctx context.Context, id string, t time.Time) error
}

// QuotaRepository defines quota data access operations.
type QuotaRepository interface {
	Create(ctx context.Context, quota *Quota) error
	GetByID(ctx context.Context, id string) (*Quota, error)
	GetByWorkspace(ctx context.Context, workspaceID string) ([]Quota, error)
	Update(ctx context.Context, quota *Quota) error
	Delete(ctx context.Context, id string) error
	AssignQuota(ctx context.Context, assignment *QuotaAssignment) error
	GetAssignment(ctx context.Context, quotaID, resourceType, resourceID string) (*QuotaAssignment, error)
	UpdateUsage(ctx context.Context, quotaID, resourceType, resourceID string, usage int64) error
	ResetUsage(ctx context.Context, quotaID, resourceType, resourceID string) error
	GetResourceAssignments(ctx context.Context, resourceType, resourceID string) ([]QuotaAssignment, error)
}

// UsageRecordRepository defines usage record data access operations.
type UsageRecordRepository interface {
	Create(ctx context.Context, record *UsageRecord) error
	GetByID(ctx context.Context, id string) (*UsageRecord, error)
	GetByRequestID(ctx context.Context, requestID string) (*UsageRecord, error)
	GetByWorkspace(ctx context.Context, workspaceID string, start, end time.Time, limit, offset int) ([]UsageRecord, error)
	GetByAPIKey(ctx context.Context, apiKeyID string, start, end time.Time, limit, offset int) ([]UsageRecord, error)
	Update(ctx context.Context, record *UsageRecord) error
	GetSummaryByWorkspace(ctx context.Context, workspaceID string, start, end time.Time) (*UsageSummary, error)
}

// TraceEventRepository defines trace event data access operations.
type TraceEventRepository interface {
	Create(ctx context.Context, event *TraceEvent) error
	GetByTraceID(ctx context.Context, traceID string) ([]TraceEvent, error)
	GetByRequestID(ctx context.Context, requestID string) ([]TraceEvent, error)
	CreateBatch(ctx context.Context, events []TraceEvent) error
}

// ModelCardRepository defines A2A Model Card data access operations.
type ModelCardRepository interface {
	// GetByID retrieves a model card by its ID.
	GetByID(ctx context.Context, id string) (*ModelCard, error)

	// GetBySlug retrieves a model card by workspace and slug.
	GetBySlug(ctx context.Context, workspaceID, slug string) (*ModelCard, error)

	// GetByWorkspace retrieves all model cards in a workspace.
	GetByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]ModelCard, error)

	// GetByUser retrieves all model cards created by a user.
	GetByUser(ctx context.Context, userID string, limit, offset int) ([]ModelCard, error)

	// Create creates a new model card.
	Create(ctx context.Context, card *ModelCard) error

	// Update updates an existing model card and creates a version record.
	Update(ctx context.Context, card *ModelCard, changeReason *string, updatedBy *string) error

	// Delete removes a model card (soft delete by setting status to 'deleted').
	Delete(ctx context.Context, id string) error

	// HardDelete permanently removes a model card and all its versions.
	HardDelete(ctx context.Context, id string) error

	// Search searches model cards using JSONB queries.
	Search(ctx context.Context, params ModelCardSearchParams) ([]ModelCardSearchResult, error)

	// SearchByCapability finds model cards with specific capabilities.
	SearchByCapability(ctx context.Context, workspaceID string, capability string, limit, offset int) ([]ModelCard, error)

	// SearchBySkill finds model cards exposing specific skills.
	SearchBySkill(ctx context.Context, workspaceID string, skillID string, limit, offset int) ([]ModelCard, error)

	// GetVersions retrieves all versions of a model card.
	GetVersions(ctx context.Context, modelCardID string) ([]ModelCardVersion, error)

	// GetVersion retrieves a specific version of a model card.
	GetVersion(ctx context.Context, modelCardID string, version int) (*ModelCardVersion, error)

	// RestoreVersion restores a model card to a specific version.
	RestoreVersion(ctx context.Context, modelCardID string, version int, restoredBy *string) error
}

// AuditLogRepository defines audit log data access operations.
type AuditLogRepository interface {
	// Log creates a new audit event.
	Log(ctx context.Context, eventType string, severity string, actorType, actorID, actorName string, resourceType, resourceID string, action, result string, details map[string]interface{}) error

	// Query retrieves audit events matching the filter.
	Query(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]map[string]interface{}, error)

	// Count returns the count of events matching the filter.
	Count(ctx context.Context, filter map[string]interface{}) (int64, error)

	// PurgeOldEvents removes events older than retention days.
	PurgeOldEvents(ctx context.Context, retentionDays int) (int64, error)
}

// UsageSummary holds aggregated usage statistics.
type UsageSummary struct {
	TotalRequests      int64
	TotalTokens        int64
	TotalPromptTokens  int64
	TotalCompletionTokens int64
	TotalCostUSD       float64
	AvgDurationMs      int
	SuccessCount       int64
	ErrorCount         int64
}

// Config holds database configuration.
type Config struct {
	Driver          string
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}
