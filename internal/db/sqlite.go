// Package db provides database interfaces and implementations for RAD Gateway.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// SQLiteDB implements the Database interface for SQLite.
type SQLiteDB struct {
	db         *sql.DB
	config     Config
	repos      *repositories
}

// repositories holds all repository implementations.
type repositories struct {
	workspaces   *sqliteWorkspaceRepo
	users        *sqliteUserRepo
	roles        *sqliteRoleRepo
	permissions  *sqlitePermissionRepo
	tags         *sqliteTagRepo
	providers    *sqliteProviderRepo
	controlRooms *sqliteControlRoomRepo
	apiKeys      *sqliteAPIKeyRepo
	quotas       *sqliteQuotaRepo
	usageRecords *sqliteUsageRecordRepo
	traceEvents  *sqliteTraceEventRepo
	modelCards   *sqliteModelCardRepo
}

// NewSQLite creates a new SQLite database connection.
func NewSQLite(config Config) (*SQLiteDB, error) {
	dsn := config.DSN
	if dsn == "" {
		dsn = "radgateway.db"
	}

	db, err := sql.Open("sqlite3", dsn+"?_journal_mode=WAL&_foreign_keys=ON&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Configure connection pool
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(1) // SQLite only supports one writer
	}
	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	}
	if config.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}

	database := &SQLiteDB{
		db:     db,
		config: config,
	}

	// Initialize repositories
	database.repos = &repositories{
		workspaces:   &sqliteWorkspaceRepo{db: db},
		users:        &sqliteUserRepo{db: db},
		roles:        &sqliteRoleRepo{db: db},
		permissions:  &sqlitePermissionRepo{db: db},
		tags:         &sqliteTagRepo{db: db},
		providers:    &sqliteProviderRepo{db: db},
		controlRooms: &sqliteControlRoomRepo{db: db},
		apiKeys:      &sqliteAPIKeyRepo{db: db},
		quotas:       &sqliteQuotaRepo{db: db},
		usageRecords: &sqliteUsageRecordRepo{db: db},
		traceEvents:  &sqliteTraceEventRepo{db: db},
		modelCards:   &sqliteModelCardRepo{db: db},
	}

	return database, nil
}

// Ping checks the database connection.
func (s *SQLiteDB) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close closes the database connection.
func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

// BeginTx starts a new transaction.
func (s *SQLiteDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, opts)
}

// ExecContext executes a query without returning rows.
func (s *SQLiteDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows.
func (s *SQLiteDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that returns a single row.
func (s *SQLiteDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return s.db.QueryRowContext(ctx, query, args...)
}

// Repository accessors
func (s *SQLiteDB) Workspaces() WorkspaceRepository   { return s.repos.workspaces }
func (s *SQLiteDB) Users() UserRepository             { return s.repos.users }
func (s *SQLiteDB) Roles() RoleRepository             { return s.repos.roles }
func (s *SQLiteDB) Permissions() PermissionRepository { return s.repos.permissions }
func (s *SQLiteDB) Tags() TagRepository               { return s.repos.tags }
func (s *SQLiteDB) Providers() ProviderRepository     { return s.repos.providers }
func (s *SQLiteDB) ControlRooms() ControlRoomRepository { return s.repos.controlRooms }
func (s *SQLiteDB) APIKeys() APIKeyRepository         { return s.repos.apiKeys }
func (s *SQLiteDB) Quotas() QuotaRepository           { return s.repos.quotas }
func (s *SQLiteDB) UsageRecords() UsageRecordRepository { return s.repos.usageRecords }
func (s *SQLiteDB) TraceEvents() TraceEventRepository { return s.repos.traceEvents }
func (s *SQLiteDB) ModelCards() ModelCardRepository     { return s.repos.modelCards }

// DB returns the underlying *sql.DB for migrations and advanced operations.
func (s *SQLiteDB) DB() *sql.DB { return s.db }

// RunMigrations executes all pending migrations.
func (s *SQLiteDB) RunMigrations() error {
	content, err := migrationsFS.ReadFile("migrations/001_initial_schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Split migration file into individual statements
	statements := strings.Split(string(content), ";")

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		_, err := s.db.Exec(stmt)
		if err != nil {
			// Ignore "already exists" errors
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			return fmt.Errorf("failed to execute migration statement: %w", err)
		}
	}

	// Record migration
	_, err = s.db.Exec(`INSERT OR IGNORE INTO schema_migrations (version) VALUES (1)`)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return nil
}

// Version returns the current schema version.
func (s *SQLiteDB) Version() (int, error) {
	var version int
	err := s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// sqliteWorkspaceRepo implements WorkspaceRepository for SQLite.
type sqliteWorkspaceRepo struct {
	db *sql.DB
}

func (r *sqliteWorkspaceRepo) Create(ctx context.Context, workspace *Workspace) error {
	query := `INSERT INTO workspaces (id, slug, name, description, status, settings, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, workspace.ID, workspace.Slug, workspace.Name,
		workspace.Description, workspace.Status, workspace.Settings, workspace.CreatedAt, workspace.UpdatedAt)
	return err
}

func (r *sqliteWorkspaceRepo) GetByID(ctx context.Context, id string) (*Workspace, error) {
	workspace := &Workspace{}
	query := `SELECT id, slug, name, description, status, settings, created_at, updated_at FROM workspaces WHERE id = ?`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&workspace.ID, &workspace.Slug, &workspace.Name, &workspace.Description,
		&workspace.Status, &workspace.Settings, &workspace.CreatedAt, &workspace.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return workspace, err
}

func (r *sqliteWorkspaceRepo) GetBySlug(ctx context.Context, slug string) (*Workspace, error) {
	workspace := &Workspace{}
	query := `SELECT id, slug, name, description, status, settings, created_at, updated_at FROM workspaces WHERE slug = ?`
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&workspace.ID, &workspace.Slug, &workspace.Name, &workspace.Description,
		&workspace.Status, &workspace.Settings, &workspace.CreatedAt, &workspace.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return workspace, err
}

func (r *sqliteWorkspaceRepo) Update(ctx context.Context, workspace *Workspace) error {
	query := `UPDATE workspaces SET slug = ?, name = ?, description = ?, status = ?, settings = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, workspace.Slug, workspace.Name, workspace.Description,
		workspace.Status, workspace.Settings, workspace.UpdatedAt, workspace.ID)
	return err
}

func (r *sqliteWorkspaceRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM workspaces WHERE id = ?`, id)
	return err
}

func (r *sqliteWorkspaceRepo) List(ctx context.Context, limit, offset int) ([]Workspace, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, slug, name, description, status, settings, created_at, updated_at FROM workspaces ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []Workspace
	for rows.Next() {
		var w Workspace
		err := rows.Scan(&w.ID, &w.Slug, &w.Name, &w.Description, &w.Status, &w.Settings, &w.CreatedAt, &w.UpdatedAt)
		if err != nil {
			return nil, err
		}
		workspaces = append(workspaces, w)
	}
	return workspaces, rows.Err()
}

// sqliteUserRepo implements UserRepository for SQLite.
type sqliteUserRepo struct {
	db *sql.DB
}

func (r *sqliteUserRepo) Create(ctx context.Context, user *User) error {
	query := `INSERT INTO users (id, workspace_id, email, display_name, status, password_hash, last_login_at, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, user.ID, user.WorkspaceID, user.Email, user.DisplayName,
		user.Status, user.PasswordHash, user.LastLoginAt, user.CreatedAt, user.UpdatedAt)
	return err
}

func (r *sqliteUserRepo) GetByID(ctx context.Context, id string) (*User, error) {
	user := &User{}
	query := `SELECT id, workspace_id, email, display_name, status, password_hash, last_login_at, created_at, updated_at FROM users WHERE id = ?`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.WorkspaceID, &user.Email, &user.DisplayName, &user.Status,
		&user.PasswordHash, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

func (r *sqliteUserRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	query := `SELECT id, workspace_id, email, display_name, status, password_hash, last_login_at, created_at, updated_at FROM users WHERE email = ?`
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.WorkspaceID, &user.Email, &user.DisplayName, &user.Status,
		&user.PasswordHash, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

func (r *sqliteUserRepo) GetByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]User, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, workspace_id, email, display_name, status, password_hash, last_login_at, created_at, updated_at
			  FROM users WHERE workspace_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, workspaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.WorkspaceID, &u.Email, &u.DisplayName, &u.Status,
			&u.PasswordHash, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *sqliteUserRepo) Update(ctx context.Context, user *User) error {
	query := `UPDATE users SET email = ?, display_name = ?, status = ?, password_hash = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, user.Email, user.DisplayName, user.Status, user.PasswordHash, user.UpdatedAt, user.ID)
	return err
}

func (r *sqliteUserRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

func (r *sqliteUserRepo) UpdateLastLogin(ctx context.Context, id string, t time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET last_login_at = ? WHERE id = ?`, t, id)
	return err
}

// Additional repository implementations would follow similar patterns...
// For brevity, implementing simplified versions with the core functionality

// sqliteRoleRepo implements RoleRepository for SQLite.
type sqliteRoleRepo struct {
	db *sql.DB
}

func (r *sqliteRoleRepo) Create(ctx context.Context, role *Role) error {
	query := `INSERT INTO roles (id, workspace_id, name, description, is_system, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, role.ID, role.WorkspaceID, role.Name, role.Description, role.IsSystem, role.CreatedAt, role.UpdatedAt)
	return err
}

func (r *sqliteRoleRepo) GetByID(ctx context.Context, id string) (*Role, error) {
	role := &Role{}
	query := `SELECT id, workspace_id, name, description, is_system, created_at, updated_at FROM roles WHERE id = ?`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&role.ID, &role.WorkspaceID, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return role, err
}

func (r *sqliteRoleRepo) GetByWorkspace(ctx context.Context, workspaceID *string) ([]Role, error) {
	var query string
	var rows *sql.Rows
	var err error

	if workspaceID == nil {
		query = `SELECT id, workspace_id, name, description, is_system, created_at, updated_at FROM roles WHERE workspace_id IS NULL`
		rows, err = r.db.QueryContext(ctx, query)
	} else {
		query = `SELECT id, workspace_id, name, description, is_system, created_at, updated_at FROM roles WHERE workspace_id = ?`
		rows, err = r.db.QueryContext(ctx, query, *workspaceID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var r Role
		err := rows.Scan(&r.ID, &r.WorkspaceID, &r.Name, &r.Description, &r.IsSystem, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return roles, rows.Err()
}

func (r *sqliteRoleRepo) Update(ctx context.Context, role *Role) error {
	query := `UPDATE roles SET name = ?, description = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, role.Name, role.Description, role.UpdatedAt, role.ID)
	return err
}

func (r *sqliteRoleRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM roles WHERE id = ?`, id)
	return err
}

func (r *sqliteRoleRepo) AssignToUser(ctx context.Context, userID, roleID string, grantedBy *string, expiresAt *time.Time) error {
	query := `INSERT INTO user_roles (user_id, role_id, granted_by, expires_at) VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, userID, roleID, grantedBy, expiresAt)
	return err
}

func (r *sqliteRoleRepo) RemoveFromUser(ctx context.Context, userID, roleID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_roles WHERE user_id = ? AND role_id = ?`, userID, roleID)
	return err
}

func (r *sqliteRoleRepo) GetUserRoles(ctx context.Context, userID string) ([]Role, error) {
	query := `SELECT r.id, r.workspace_id, r.name, r.description, r.is_system, r.created_at, r.updated_at
			  FROM roles r
			  JOIN user_roles ur ON r.id = ur.role_id
			  WHERE ur.user_id = ?`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var r Role
		err := rows.Scan(&r.ID, &r.WorkspaceID, &r.Name, &r.Description, &r.IsSystem, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return roles, rows.Err()
}

// Placeholder implementations for remaining repositories
// These would be fully implemented in a production environment

type sqlitePermissionRepo struct{ db *sql.DB }

func (r *sqlitePermissionRepo) Create(ctx context.Context, p *Permission) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO permissions (id, name, description, resource_type, action) VALUES (?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Description, p.ResourceType, p.Action)
	return err
}
func (r *sqlitePermissionRepo) GetByID(ctx context.Context, id string) (*Permission, error) { return nil, nil }
func (r *sqlitePermissionRepo) GetByName(ctx context.Context, name string) (*Permission, error) { return nil, nil }
func (r *sqlitePermissionRepo) List(ctx context.Context) ([]Permission, error) { return nil, nil }
func (r *sqlitePermissionRepo) AssignToRole(ctx context.Context, roleID, permissionID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)`, roleID, permissionID)
	return err
}
func (r *sqlitePermissionRepo) RemoveFromRole(ctx context.Context, roleID, permissionID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id = ? AND permission_id = ?`, roleID, permissionID)
	return err
}
func (r *sqlitePermissionRepo) GetRolePermissions(ctx context.Context, roleID string) ([]Permission, error) { return nil, nil }
func (r *sqlitePermissionRepo) GetUserPermissions(ctx context.Context, userID string) ([]Permission, error) { return nil, nil }

type sqliteTagRepo struct{ db *sql.DB }

func (r *sqliteTagRepo) Create(ctx context.Context, tag *Tag) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO tags (id, workspace_id, category, value, description, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		tag.ID, tag.WorkspaceID, tag.Category, tag.Value, tag.Description, tag.CreatedAt)
	return err
}
func (r *sqliteTagRepo) GetByID(ctx context.Context, id string) (*Tag, error) { return nil, nil }
func (r *sqliteTagRepo) GetByCategoryValue(ctx context.Context, workspaceID, category, value string) (*Tag, error) { return nil, nil }
func (r *sqliteTagRepo) GetByWorkspace(ctx context.Context, workspaceID string) ([]Tag, error) { return nil, nil }
func (r *sqliteTagRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, id)
	return err
}
func (r *sqliteTagRepo) AssignToProvider(ctx context.Context, providerID, tagID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO provider_tags (provider_id, tag_id) VALUES (?, ?)`, providerID, tagID)
	return err
}
func (r *sqliteTagRepo) RemoveFromProvider(ctx context.Context, providerID, tagID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM provider_tags WHERE provider_id = ? AND tag_id = ?`, providerID, tagID)
	return err
}
func (r *sqliteTagRepo) GetProviderTags(ctx context.Context, providerID string) ([]Tag, error) { return nil, nil }
func (r *sqliteTagRepo) AssignToAPIKey(ctx context.Context, apiKeyID, tagID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO api_key_tags (api_key_id, tag_id) VALUES (?, ?)`, apiKeyID, tagID)
	return err
}
func (r *sqliteTagRepo) RemoveFromAPIKey(ctx context.Context, apiKeyID, tagID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM api_key_tags WHERE api_key_id = ? AND tag_id = ?`, apiKeyID, tagID)
	return err
}
func (r *sqliteTagRepo) GetAPIKeyTags(ctx context.Context, apiKeyID string) ([]Tag, error) { return nil, nil }

type sqliteProviderRepo struct{ db *sql.DB }

func (r *sqliteProviderRepo) Create(ctx context.Context, p *Provider) error {
	query := `INSERT INTO providers (id, workspace_id, slug, name, provider_type, base_url, api_key_encrypted, config, status, priority, weight, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, p.ID, p.WorkspaceID, p.Slug, p.Name, p.ProviderType, p.BaseURL, p.APIKeyEncrypted, p.Config, p.Status, p.Priority, p.Weight, p.CreatedAt, p.UpdatedAt)
	return err
}
func (r *sqliteProviderRepo) GetByID(ctx context.Context, id string) (*Provider, error) { return nil, nil }
func (r *sqliteProviderRepo) GetBySlug(ctx context.Context, workspaceID, slug string) (*Provider, error) { return nil, nil }
func (r *sqliteProviderRepo) GetByWorkspace(ctx context.Context, workspaceID string) ([]Provider, error) { return nil, nil }
func (r *sqliteProviderRepo) GetByTags(ctx context.Context, workspaceID string, tagIDs []string) ([]Provider, error) { return nil, nil }
func (r *sqliteProviderRepo) Update(ctx context.Context, p *Provider) error { return nil }
func (r *sqliteProviderRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM providers WHERE id = ?`, id)
	return err
}
func (r *sqliteProviderRepo) UpdateHealth(ctx context.Context, health *ProviderHealth) error {
	query := `INSERT OR REPLACE INTO provider_health (provider_id, healthy, last_check_at, last_success_at, consecutive_failures, latency_ms, error_message, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, health.ProviderID, health.Healthy, health.LastCheckAt, health.LastSuccessAt, health.ConsecutiveFailures, health.LatencyMs, health.ErrorMessage, health.UpdatedAt)
	return err
}
func (r *sqliteProviderRepo) GetHealth(ctx context.Context, providerID string) (*ProviderHealth, error) { return nil, nil }
func (r *sqliteProviderRepo) UpdateCircuitBreaker(ctx context.Context, state *CircuitBreakerState) error {
	query := `INSERT OR REPLACE INTO circuit_breaker_states (provider_id, state, failures, successes, last_failure_at, half_open_requests, opened_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, state.ProviderID, state.State, state.Failures, state.Successes, state.LastFailureAt, state.HalfOpenRequests, state.OpenedAt, state.UpdatedAt)
	return err
}
func (r *sqliteProviderRepo) GetCircuitBreaker(ctx context.Context, providerID string) (*CircuitBreakerState, error) { return nil, nil }

type sqliteControlRoomRepo struct{ db *sql.DB }

func (r *sqliteControlRoomRepo) Create(ctx context.Context, room *ControlRoom) error {
	query := `INSERT INTO control_rooms (id, workspace_id, slug, name, description, tag_filter, dashboard_layout, created_by, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, room.ID, room.WorkspaceID, room.Slug, room.Name, room.Description, room.TagFilter, room.DashboardLayout, room.CreatedBy, room.CreatedAt, room.UpdatedAt)
	return err
}
func (r *sqliteControlRoomRepo) GetByID(ctx context.Context, id string) (*ControlRoom, error) { return nil, nil }
func (r *sqliteControlRoomRepo) GetBySlug(ctx context.Context, workspaceID, slug string) (*ControlRoom, error) { return nil, nil }
func (r *sqliteControlRoomRepo) GetByWorkspace(ctx context.Context, workspaceID string) ([]ControlRoom, error) { return nil, nil }
func (r *sqliteControlRoomRepo) Update(ctx context.Context, room *ControlRoom) error { return nil }
func (r *sqliteControlRoomRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM control_rooms WHERE id = ?`, id)
	return err
}
func (r *sqliteControlRoomRepo) GrantAccess(ctx context.Context, access *ControlRoomAccess) error {
	query := `INSERT INTO control_room_access (control_room_id, user_id, role, granted_by, granted_at, expires_at)
			  VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, access.ControlRoomID, access.UserID, access.Role, access.GrantedBy, access.GrantedAt, access.ExpiresAt)
	return err
}
func (r *sqliteControlRoomRepo) RevokeAccess(ctx context.Context, controlRoomID, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM control_room_access WHERE control_room_id = ? AND user_id = ?`, controlRoomID, userID)
	return err
}
func (r *sqliteControlRoomRepo) GetUserAccess(ctx context.Context, controlRoomID string) ([]ControlRoomAccess, error) { return nil, nil }

type sqliteAPIKeyRepo struct{ db *sql.DB }

func (r *sqliteAPIKeyRepo) Create(ctx context.Context, key *APIKey) error {
	query := `INSERT INTO api_keys (id, workspace_id, name, key_hash, key_preview, status, created_by, expires_at, last_used_at, rate_limit, allowed_models, allowed_apis, metadata, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, key.ID, key.WorkspaceID, key.Name, key.KeyHash, key.KeyPreview, key.Status, key.CreatedBy, key.ExpiresAt, key.LastUsedAt, key.RateLimit, key.AllowedModels, key.AllowedAPIs, key.Metadata, key.CreatedAt, key.UpdatedAt)
	return err
}
func (r *sqliteAPIKeyRepo) GetByID(ctx context.Context, id string) (*APIKey, error) { return nil, nil }
func (r *sqliteAPIKeyRepo) GetByHash(ctx context.Context, hash string) (*APIKey, error) { return nil, nil }
func (r *sqliteAPIKeyRepo) GetByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]APIKey, error) { return nil, nil }
func (r *sqliteAPIKeyRepo) Update(ctx context.Context, key *APIKey) error { return nil }
func (r *sqliteAPIKeyRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = ?`, id)
	return err
}
func (r *sqliteAPIKeyRepo) UpdateLastUsed(ctx context.Context, id string, t time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE api_keys SET last_used_at = ? WHERE id = ?`, t, id)
	return err
}

type sqliteQuotaRepo struct{ db *sql.DB }

func (r *sqliteQuotaRepo) Create(ctx context.Context, quota *Quota) error {
	query := `INSERT INTO quotas (id, workspace_id, name, description, quota_type, period, limit_value, scope, warning_threshold, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, quota.ID, quota.WorkspaceID, quota.Name, quota.Description, quota.QuotaType, quota.Period, quota.LimitValue, quota.Scope, quota.WarningThreshold, quota.CreatedAt, quota.UpdatedAt)
	return err
}
func (r *sqliteQuotaRepo) GetByID(ctx context.Context, id string) (*Quota, error) { return nil, nil }
func (r *sqliteQuotaRepo) GetByWorkspace(ctx context.Context, workspaceID string) ([]Quota, error) { return nil, nil }
func (r *sqliteQuotaRepo) Update(ctx context.Context, quota *Quota) error { return nil }
func (r *sqliteQuotaRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM quotas WHERE id = ?`, id)
	return err
}
func (r *sqliteQuotaRepo) AssignQuota(ctx context.Context, assignment *QuotaAssignment) error {
	query := `INSERT INTO quota_assignments (quota_id, resource_type, resource_id, current_usage, period_start, period_end, warning_sent, exceeded_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, assignment.QuotaID, assignment.ResourceType, assignment.ResourceID, assignment.CurrentUsage, assignment.PeriodStart, assignment.PeriodEnd, assignment.WarningSent, assignment.ExceededAt, assignment.UpdatedAt)
	return err
}
func (r *sqliteQuotaRepo) GetAssignment(ctx context.Context, quotaID, resourceType, resourceID string) (*QuotaAssignment, error) { return nil, nil }
func (r *sqliteQuotaRepo) UpdateUsage(ctx context.Context, quotaID, resourceType, resourceID string, usage int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE quota_assignments SET current_usage = ? WHERE quota_id = ? AND resource_type = ? AND resource_id = ?`, usage, quotaID, resourceType, resourceID)
	return err
}
func (r *sqliteQuotaRepo) ResetUsage(ctx context.Context, quotaID, resourceType, resourceID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE quota_assignments SET current_usage = 0 WHERE quota_id = ? AND resource_type = ? AND resource_id = ?`, quotaID, resourceType, resourceID)
	return err
}
func (r *sqliteQuotaRepo) GetResourceAssignments(ctx context.Context, resourceType, resourceID string) ([]QuotaAssignment, error) { return nil, nil }

type sqliteUsageRecordRepo struct{ db *sql.DB }

func (r *sqliteUsageRecordRepo) Create(ctx context.Context, record *UsageRecord) error {
	query := `INSERT INTO usage_records (id, workspace_id, request_id, trace_id, api_key_id, control_room_id, incoming_api, incoming_model, selected_model, provider_id, prompt_tokens, completion_tokens, total_tokens, cost_usd, duration_ms, response_status, error_code, error_message, attempts, route_log, started_at, completed_at, created_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, record.ID, record.WorkspaceID, record.RequestID, record.TraceID, record.APIKeyID, record.ControlRoomID, record.IncomingAPI, record.IncomingModel, record.SelectedModel, record.ProviderID, record.PromptTokens, record.CompletionTokens, record.TotalTokens, record.CostUSD, record.DurationMs, record.ResponseStatus, record.ErrorCode, record.ErrorMessage, record.Attempts, record.RouteLog, record.StartedAt, record.CompletedAt, record.CreatedAt)
	return err
}
func (r *sqliteUsageRecordRepo) GetByID(ctx context.Context, id string) (*UsageRecord, error) { return nil, nil }
func (r *sqliteUsageRecordRepo) GetByRequestID(ctx context.Context, requestID string) (*UsageRecord, error) { return nil, nil }
func (r *sqliteUsageRecordRepo) GetByWorkspace(ctx context.Context, workspaceID string, start, end time.Time, limit, offset int) ([]UsageRecord, error) { return nil, nil }
func (r *sqliteUsageRecordRepo) GetByAPIKey(ctx context.Context, apiKeyID string, start, end time.Time, limit, offset int) ([]UsageRecord, error) { return nil, nil }
func (r *sqliteUsageRecordRepo) Update(ctx context.Context, record *UsageRecord) error { return nil }
func (r *sqliteUsageRecordRepo) GetSummaryByWorkspace(ctx context.Context, workspaceID string, start, end time.Time) (*UsageSummary, error) { return nil, nil }

type sqliteTraceEventRepo struct{ db *sql.DB }

func (r *sqliteTraceEventRepo) Create(ctx context.Context, event *TraceEvent) error {
	query := `INSERT INTO trace_events (id, trace_id, request_id, event_type, event_order, provider_id, api_key_id, message, metadata, timestamp, duration_ms, created_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, event.ID, event.TraceID, event.RequestID, event.EventType, event.EventOrder, event.ProviderID, event.APIKeyID, event.Message, event.Metadata, event.Timestamp, event.DurationMs, event.CreatedAt)
	return err
}
func (r *sqliteTraceEventRepo) GetByTraceID(ctx context.Context, traceID string) ([]TraceEvent, error) { return nil, nil }
func (r *sqliteTraceEventRepo) GetByRequestID(ctx context.Context, requestID string) ([]TraceEvent, error) { return nil, nil }
func (r *sqliteTraceEventRepo) CreateBatch(ctx context.Context, events []TraceEvent) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, event := range events {
		if err := r.Create(ctx, &event); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type sqliteModelCardRepo struct{ db *sql.DB }

func (r *sqliteModelCardRepo) GetByID(ctx context.Context, id string) (*ModelCard, error) {
	card := &ModelCard{}
	query := `SELECT id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at
		      FROM a2a_model_cards WHERE id = ?`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&card.ID, &card.WorkspaceID, &card.UserID, &card.Name, &card.Slug,
		&card.Description, &card.Card, &card.Version, &card.Status,
		&card.CreatedAt, &card.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return card, err
}

func (r *sqliteModelCardRepo) GetBySlug(ctx context.Context, workspaceID, slug string) (*ModelCard, error) {
	card := &ModelCard{}
	query := `SELECT id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at
		      FROM a2a_model_cards WHERE workspace_id = ? AND slug = ?`
	err := r.db.QueryRowContext(ctx, query, workspaceID, slug).Scan(
		&card.ID, &card.WorkspaceID, &card.UserID, &card.Name, &card.Slug,
		&card.Description, &card.Card, &card.Version, &card.Status,
		&card.CreatedAt, &card.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return card, err
}

func (r *sqliteModelCardRepo) GetByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]ModelCard, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at
		      FROM a2a_model_cards WHERE workspace_id = ? ORDER BY updated_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, workspaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []ModelCard
	for rows.Next() {
		var c ModelCard
		err := rows.Scan(
			&c.ID, &c.WorkspaceID, &c.UserID, &c.Name, &c.Slug,
			&c.Description, &c.Card, &c.Version, &c.Status,
			&c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, rows.Err()
}

func (r *sqliteModelCardRepo) GetByUser(ctx context.Context, userID string, limit, offset int) ([]ModelCard, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at
		      FROM a2a_model_cards WHERE user_id = ? ORDER BY updated_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []ModelCard
	for rows.Next() {
		var c ModelCard
		err := rows.Scan(
			&c.ID, &c.WorkspaceID, &c.UserID, &c.Name, &c.Slug,
			&c.Description, &c.Card, &c.Version, &c.Status,
			&c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, rows.Err()
}

func (r *sqliteModelCardRepo) Create(ctx context.Context, card *ModelCard) error {
	if card.ID == "" {
		card.ID = generateUUID()
	}
	now := time.Now()
	card.CreatedAt = now
	card.UpdatedAt = now
	card.Version = 1
	if card.Status == "" {
		card.Status = "active"
	}

	query := `INSERT INTO a2a_model_cards (id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at)
		      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		card.ID, card.WorkspaceID, card.UserID, card.Name, card.Slug,
		card.Description, card.Card, card.Version, card.Status, card.CreatedAt, card.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create model card: %w", err)
	}
	return r.createVersion(ctx, card, nil, card.UserID)
}

func (r *sqliteModelCardRepo) Update(ctx context.Context, card *ModelCard, changeReason *string, updatedBy *string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current card to capture version
	current, err := r.GetByID(ctx, card.ID)
	if err != nil {
		return fmt.Errorf("failed to get current card: %w", err)
	}
	if current == nil {
		return fmt.Errorf("model card not found: %s", card.ID)
	}

	card.Version = current.Version + 1
	card.UpdatedAt = time.Now()

	// Create version record
	versionQuery := `INSERT INTO model_card_versions (id, model_card_id, workspace_id, user_id, version, name, slug, description, card, status, change_reason, created_by, created_at)
		              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, versionQuery,
		generateUUID(), current.ID, current.WorkspaceID, current.UserID,
		current.Version, current.Name, current.Slug, current.Description,
		current.Card, current.Status, changeReason, updatedBy, current.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create version record: %w", err)
	}

	// Update the card
	updateQuery := `UPDATE a2a_model_cards
		            SET name = ?, slug = ?, description = ?, card = ?,
		                version = ?, status = ?, updated_at = ?
		            WHERE id = ?`
	_, err = tx.ExecContext(ctx, updateQuery,
		card.Name, card.Slug, card.Description, card.Card,
		card.Version, card.Status, card.UpdatedAt, card.ID)
	if err != nil {
		return fmt.Errorf("failed to update model card: %w", err)
	}

	return tx.Commit()
}

func (r *sqliteModelCardRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE a2a_model_cards SET status = 'deleted', updated_at = ? WHERE id = ?`,
		time.Now(), id)
	return err
}

func (r *sqliteModelCardRepo) HardDelete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM a2a_model_cards WHERE id = ?`, id)
	return err
}

func (r *sqliteModelCardRepo) Search(ctx context.Context, params ModelCardSearchParams) ([]ModelCardSearchResult, error) {
	if params.Limit <= 0 {
		params.Limit = 100
	}

	// Build dynamic query (SQLite-compatible without JSONB operators)
	query := `SELECT id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at
		      FROM a2a_model_cards WHERE workspace_id = ?`
	args := []interface{}{params.WorkspaceID}

	if params.Status != "" {
		query += " AND status = ?"
		args = append(args, params.Status)
	}

	if params.Query != "" {
		query += " AND (name LIKE ? OR description LIKE ?)"
		args = append(args, "%"+params.Query+"%", "%"+params.Query+"%")
	}

	query += " ORDER BY updated_at DESC LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ModelCardSearchResult
	for rows.Next() {
		var c ModelCard
		err := rows.Scan(
			&c.ID, &c.WorkspaceID, &c.UserID, &c.Name, &c.Slug,
			&c.Description, &c.Card, &c.Version, &c.Status,
			&c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, ModelCardSearchResult{ModelCard: c, Relevance: 1.0})
	}
	return results, rows.Err()
}

func (r *sqliteModelCardRepo) SearchByCapability(ctx context.Context, workspaceID string, capability string, limit, offset int) ([]ModelCard, error) {
	// SQLite doesn't support JSONB operators natively
	// Return all cards in workspace as fallback (filter in application layer if needed)
	return r.GetByWorkspace(ctx, workspaceID, limit, offset)
}

func (r *sqliteModelCardRepo) SearchBySkill(ctx context.Context, workspaceID string, skillID string, limit, offset int) ([]ModelCard, error) {
	// SQLite doesn't support JSONB array operators natively
	// Return all cards in workspace as fallback (filter in application layer if needed)
	return r.GetByWorkspace(ctx, workspaceID, limit, offset)
}

func (r *sqliteModelCardRepo) GetVersions(ctx context.Context, modelCardID string) ([]ModelCardVersion, error) {
	query := `SELECT id, model_card_id, workspace_id, user_id, version, name, slug, description, card, status, change_reason, created_by, created_at
		      FROM model_card_versions
		      WHERE model_card_id = ? ORDER BY version DESC`
	rows, err := r.db.QueryContext(ctx, query, modelCardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []ModelCardVersion
	for rows.Next() {
		var v ModelCardVersion
		err := rows.Scan(
			&v.ID, &v.ModelCardID, &v.WorkspaceID, &v.UserID, &v.Version,
			&v.Name, &v.Slug, &v.Description, &v.Card, &v.Status,
			&v.ChangeReason, &v.CreatedBy, &v.CreatedAt)
		if err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

func (r *sqliteModelCardRepo) GetVersion(ctx context.Context, modelCardID string, version int) (*ModelCardVersion, error) {
	v := &ModelCardVersion{}
	query := `SELECT id, model_card_id, workspace_id, user_id, version, name, slug, description, card, status, change_reason, created_by, created_at
		      FROM model_card_versions WHERE model_card_id = ? AND version = ?`
	err := r.db.QueryRowContext(ctx, query, modelCardID, version).Scan(
		&v.ID, &v.ModelCardID, &v.WorkspaceID, &v.UserID, &v.Version,
		&v.Name, &v.Slug, &v.Description, &v.Card, &v.Status,
		&v.ChangeReason, &v.CreatedBy, &v.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return v, err
}

func (r *sqliteModelCardRepo) RestoreVersion(ctx context.Context, modelCardID string, version int, restoredBy *string) error {
	v, err := r.GetVersion(ctx, modelCardID, version)
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}
	if v == nil {
		return fmt.Errorf("version not found: %d", version)
	}

	current, err := r.GetByID(ctx, modelCardID)
	if err != nil {
		return fmt.Errorf("failed to get current card: %w", err)
	}
	if current == nil {
		return fmt.Errorf("model card not found: %s", modelCardID)
	}

	changeReason := "Restored from version " + fmt.Sprintf("%d", version)
	card := &ModelCard{
		ID:          current.ID,
		WorkspaceID: current.WorkspaceID,
		UserID:      current.UserID,
		Name:        v.Name,
		Slug:        v.Slug,
		Description: v.Description,
		Card:        v.Card,
		Status:      v.Status,
	}

	return r.Update(ctx, card, &changeReason, restoredBy)
}

func (r *sqliteModelCardRepo) createVersion(ctx context.Context, card *ModelCard, changeReason *string, createdBy *string) error {
	query := `INSERT INTO model_card_versions (id, model_card_id, workspace_id, user_id, version, name, slug, description, card, status, change_reason, created_by, created_at)
		      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		generateUUID(), card.ID, card.WorkspaceID, card.UserID,
		card.Version, card.Name, card.Slug, card.Description,
		card.Card, card.Status, changeReason, createdBy, card.CreatedAt)
	return err
}
