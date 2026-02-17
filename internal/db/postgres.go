// Package db provides database interfaces and implementations for RAD Gateway.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var postgresMigrationsFS embed.FS

// PostgresDB implements the Database interface for PostgreSQL.
type PostgresDB struct {
	db     *sql.DB
	config Config
	repos  *pgRepositories
}

// pgRepositories holds all PostgreSQL repository implementations.
type pgRepositories struct {
	workspaces   *pgWorkspaceRepo
	users        *pgUserRepo
	roles        *pgRoleRepo
	permissions  *pgPermissionRepo
	tags         *pgTagRepo
	providers    *pgProviderRepo
	controlRooms *pgControlRoomRepo
	apiKeys      *pgAPIKeyRepo
	quotas       *pgQuotaRepo
	usageRecords *pgUsageRecordRepo
	traceEvents  *pgTraceEventRepo
}

// NewPostgres creates a new PostgreSQL database connection.
func NewPostgres(config Config) (*PostgresDB, error) {
	if config.DSN == "" {
		return nil, fmt.Errorf("PostgreSQL DSN is required")
	}

	db, err := sql.Open("postgres", config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}

	// Configure connection pool
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(25) // Default for PostgreSQL
	}
	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	} else {
		db.SetMaxIdleConns(5)
	}
	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	} else {
		db.SetConnMaxLifetime(5 * time.Minute)
	}
	if config.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}

	database := &PostgresDB{
		db:     db,
		config: config,
	}

	// Initialize repositories
	database.repos = &pgRepositories{
		workspaces:   &pgWorkspaceRepo{db: db},
		users:        &pgUserRepo{db: db},
		roles:        &pgRoleRepo{db: db},
		permissions:  &pgPermissionRepo{db: db},
		tags:         &pgTagRepo{db: db},
		providers:    &pgProviderRepo{db: db},
		controlRooms: &pgControlRoomRepo{db: db},
		apiKeys:      &pgAPIKeyRepo{db: db},
		quotas:       &pgQuotaRepo{db: db},
		usageRecords: &pgUsageRecordRepo{db: db},
		traceEvents:  &pgTraceEventRepo{db: db},
	}

	return database, nil
}

// Ping checks the database connection.
func (p *PostgresDB) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// Close closes the database connection.
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// BeginTx starts a new transaction.
func (p *PostgresDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return p.db.BeginTx(ctx, opts)
}

// ExecContext executes a query without returning rows.
func (p *PostgresDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return p.db.ExecContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows.
func (p *PostgresDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that returns a single row.
func (p *PostgresDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return p.db.QueryRowContext(ctx, query, args...)
}

// Repository accessors
func (p *PostgresDB) Workspaces() WorkspaceRepository     { return p.repos.workspaces }
func (p *PostgresDB) Users() UserRepository               { return p.repos.users }
func (p *PostgresDB) Roles() RoleRepository               { return p.repos.roles }
func (p *PostgresDB) Permissions() PermissionRepository   { return p.repos.permissions }
func (p *PostgresDB) Tags() TagRepository                 { return p.repos.tags }
func (p *PostgresDB) Providers() ProviderRepository       { return p.repos.providers }
func (p *PostgresDB) ControlRooms() ControlRoomRepository { return p.repos.controlRooms }
func (p *PostgresDB) APIKeys() APIKeyRepository           { return p.repos.apiKeys }
func (p *PostgresDB) Quotas() QuotaRepository             { return p.repos.quotas }
func (p *PostgresDB) UsageRecords() UsageRecordRepository { return p.repos.usageRecords }
func (p *PostgresDB) TraceEvents() TraceEventRepository   { return p.repos.traceEvents }

// DB returns the underlying *sql.DB for migrations and advanced operations.
func (p *PostgresDB) DB() *sql.DB { return p.db }

// RunMigrations executes all pending migrations.
func (p *PostgresDB) RunMigrations() error {
	content, err := postgresMigrationsFS.ReadFile("migrations/001_initial_schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Convert SQLite syntax to PostgreSQL
	pgSQL := convertToPostgres(string(content))

	// Split migration file into individual statements
	statements := splitStatements(pgSQL)

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		_, err := p.db.Exec(stmt)
		if err != nil {
			// Ignore "already exists" errors
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			return fmt.Errorf("failed to execute migration statement: %w", err)
		}
	}

	// Record migration
	_, err = p.db.Exec(`INSERT INTO schema_migrations (version) VALUES (1) ON CONFLICT (version) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return nil
}

// Version returns the current schema version.
func (p *PostgresDB) Version() (int, error) {
	var version int
	err := p.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// convertToPostgres converts SQLite SQL to PostgreSQL compatible SQL.
func convertToPostgres(sql string) string {
	// Replace SQLite-specific syntax with PostgreSQL equivalents
	replacements := map[string]string{
		"INTEGER PRIMARY KEY": "SERIAL PRIMARY KEY", // Note: We keep TEXT IDs as-is
		"BOOLEAN":             "BOOLEAN",
		"BLOB":                "BYTEA",
		"TEXT":                "TEXT",
		"INTEGER":             "INTEGER",
		"REAL":                "DOUBLE PRECISION",
		"CURRENT_TIMESTAMP":   "CURRENT_TIMESTAMP",
		"?":                   "$1", // Parameter placeholders - will be handled separately
	}

	result := sql
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	// Remove SQLite-specific pragmas and settings
	lines := strings.Split(result, "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "PRAGMA") && !strings.HasPrefix(trimmed, "-- PRAGMA") {
			filtered = append(filtered, line)
		}
	}

	return strings.Join(filtered, "\n")
}

// splitStatements splits SQL into individual statements.
func splitStatements(sql string) []string {
	var statements []string
	var current strings.Builder
	inString := false
	stringChar := rune(0)

	for i, ch := range sql {
		if !inString && (ch == '\'' || ch == '"') {
			inString = true
			stringChar = ch
		} else if inString && ch == stringChar {
			// Check for escape
			if i > 0 && sql[i-1] != '\\' {
				inString = false
			}
		} else if !inString && ch == ';' {
			statements = append(statements, current.String())
			current.Reset()
			continue
		}
		current.WriteRune(ch)
	}

	if current.Len() > 0 {
		statements = append(statements, current.String())
	}

	return statements
}

// pgWorkspaceRepo implements WorkspaceRepository for PostgreSQL.
type pgWorkspaceRepo struct {
	db *sql.DB
}

func (r *pgWorkspaceRepo) Create(ctx context.Context, workspace *Workspace) error {
	query := `INSERT INTO workspaces (id, slug, name, description, status, settings, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query, workspace.ID, workspace.Slug, workspace.Name,
		workspace.Description, workspace.Status, workspace.Settings, workspace.CreatedAt, workspace.UpdatedAt)
	return err
}

func (r *pgWorkspaceRepo) GetByID(ctx context.Context, id string) (*Workspace, error) {
	workspace := &Workspace{}
	query := `SELECT id, slug, name, description, status, settings, created_at, updated_at FROM workspaces WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&workspace.ID, &workspace.Slug, &workspace.Name, &workspace.Description,
		&workspace.Status, &workspace.Settings, &workspace.CreatedAt, &workspace.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return workspace, err
}

func (r *pgWorkspaceRepo) GetBySlug(ctx context.Context, slug string) (*Workspace, error) {
	workspace := &Workspace{}
	query := `SELECT id, slug, name, description, status, settings, created_at, updated_at FROM workspaces WHERE slug = $1`
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&workspace.ID, &workspace.Slug, &workspace.Name, &workspace.Description,
		&workspace.Status, &workspace.Settings, &workspace.CreatedAt, &workspace.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return workspace, err
}

func (r *pgWorkspaceRepo) Update(ctx context.Context, workspace *Workspace) error {
	query := `UPDATE workspaces SET slug = $1, name = $2, description = $3, status = $4, settings = $5, updated_at = $6 WHERE id = $7`
	_, err := r.db.ExecContext(ctx, query, workspace.Slug, workspace.Name, workspace.Description,
		workspace.Status, workspace.Settings, workspace.UpdatedAt, workspace.ID)
	return err
}

func (r *pgWorkspaceRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM workspaces WHERE id = $1`, id)
	return err
}

func (r *pgWorkspaceRepo) List(ctx context.Context, limit, offset int) ([]Workspace, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, slug, name, description, status, settings, created_at, updated_at FROM workspaces ORDER BY created_at DESC LIMIT $1 OFFSET $2`
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

// pgUserRepo implements UserRepository for PostgreSQL.
type pgUserRepo struct {
	db *sql.DB
}

func (r *pgUserRepo) Create(ctx context.Context, user *User) error {
	query := `INSERT INTO users (id, workspace_id, email, display_name, status, password_hash, last_login_at, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(ctx, query, user.ID, user.WorkspaceID, user.Email, user.DisplayName,
		user.Status, user.PasswordHash, user.LastLoginAt, user.CreatedAt, user.UpdatedAt)
	return err
}

func (r *pgUserRepo) GetByID(ctx context.Context, id string) (*User, error) {
	user := &User{}
	query := `SELECT id, workspace_id, email, display_name, status, password_hash, last_login_at, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.WorkspaceID, &user.Email, &user.DisplayName, &user.Status,
		&user.PasswordHash, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

func (r *pgUserRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	query := `SELECT id, workspace_id, email, display_name, status, password_hash, last_login_at, created_at, updated_at FROM users WHERE email = $1`
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.WorkspaceID, &user.Email, &user.DisplayName, &user.Status,
		&user.PasswordHash, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

func (r *pgUserRepo) GetByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]User, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, workspace_id, email, display_name, status, password_hash, last_login_at, created_at, updated_at
			  FROM users WHERE workspace_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
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

func (r *pgUserRepo) Update(ctx context.Context, user *User) error {
	query := `UPDATE users SET email = $1, display_name = $2, status = $3, password_hash = $4, updated_at = $5 WHERE id = $6`
	_, err := r.db.ExecContext(ctx, query, user.Email, user.DisplayName, user.Status, user.PasswordHash, user.UpdatedAt, user.ID)
	return err
}

func (r *pgUserRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

func (r *pgUserRepo) UpdateLastLogin(ctx context.Context, id string, t time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET last_login_at = $1 WHERE id = $2`, t, id)
	return err
}

// Placeholder implementations for remaining PostgreSQL repositories

type pgRoleRepo struct{ db *sql.DB }

func (r *pgRoleRepo) Create(ctx context.Context, role *Role) error {
	query := `INSERT INTO roles (id, workspace_id, name, description, is_system, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query, role.ID, role.WorkspaceID, role.Name, role.Description, role.IsSystem, role.CreatedAt, role.UpdatedAt)
	return err
}
func (r *pgRoleRepo) GetByID(ctx context.Context, id string) (*Role, error) { return nil, nil }
func (r *pgRoleRepo) GetByWorkspace(ctx context.Context, workspaceID *string) ([]Role, error) { return nil, nil }
func (r *pgRoleRepo) Update(ctx context.Context, role *Role) error { return nil }
func (r *pgRoleRepo) Delete(ctx context.Context, id string) error { return nil }
func (r *pgRoleRepo) AssignToUser(ctx context.Context, userID, roleID string, grantedBy *string, expiresAt *time.Time) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO user_roles (user_id, role_id, granted_by, expires_at) VALUES ($1, $2, $3, $4)`, userID, roleID, grantedBy, expiresAt)
	return err
}
func (r *pgRoleRepo) RemoveFromUser(ctx context.Context, userID, roleID string) error { return nil }
func (r *pgRoleRepo) GetUserRoles(ctx context.Context, userID string) ([]Role, error) { return nil, nil }

type pgPermissionRepo struct{ db *sql.DB }

func (r *pgPermissionRepo) Create(ctx context.Context, p *Permission) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO permissions (id, name, description, resource_type, action) VALUES ($1, $2, $3, $4, $5)`,
		p.ID, p.Name, p.Description, p.ResourceType, p.Action)
	return err
}
func (r *pgPermissionRepo) GetByID(ctx context.Context, id string) (*Permission, error) { return nil, nil }
func (r *pgPermissionRepo) GetByName(ctx context.Context, name string) (*Permission, error) { return nil, nil }
func (r *pgPermissionRepo) List(ctx context.Context) ([]Permission, error) { return nil, nil }
func (r *pgPermissionRepo) AssignToRole(ctx context.Context, roleID, permissionID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)`, roleID, permissionID)
	return err
}
func (r *pgPermissionRepo) RemoveFromRole(ctx context.Context, roleID, permissionID string) error { return nil }
func (r *pgPermissionRepo) GetRolePermissions(ctx context.Context, roleID string) ([]Permission, error) { return nil, nil }
func (r *pgPermissionRepo) GetUserPermissions(ctx context.Context, userID string) ([]Permission, error) { return nil, nil }

type pgTagRepo struct{ db *sql.DB }

func (r *pgTagRepo) Create(ctx context.Context, tag *Tag) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO tags (id, workspace_id, category, value, description, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		tag.ID, tag.WorkspaceID, tag.Category, tag.Value, tag.Description, tag.CreatedAt)
	return err
}
func (r *pgTagRepo) GetByID(ctx context.Context, id string) (*Tag, error) { return nil, nil }
func (r *pgTagRepo) GetByCategoryValue(ctx context.Context, workspaceID, category, value string) (*Tag, error) { return nil, nil }
func (r *pgTagRepo) GetByWorkspace(ctx context.Context, workspaceID string) ([]Tag, error) { return nil, nil }
func (r *pgTagRepo) Delete(ctx context.Context, id string) error { return nil }
func (r *pgTagRepo) AssignToProvider(ctx context.Context, providerID, tagID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO provider_tags (provider_id, tag_id) VALUES ($1, $2)`, providerID, tagID)
	return err
}
func (r *pgTagRepo) RemoveFromProvider(ctx context.Context, providerID, tagID string) error { return nil }
func (r *pgTagRepo) GetProviderTags(ctx context.Context, providerID string) ([]Tag, error) { return nil, nil }
func (r *pgTagRepo) AssignToAPIKey(ctx context.Context, apiKeyID, tagID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO api_key_tags (api_key_id, tag_id) VALUES ($1, $2)`, apiKeyID, tagID)
	return err
}
func (r *pgTagRepo) RemoveFromAPIKey(ctx context.Context, apiKeyID, tagID string) error { return nil }
func (r *pgTagRepo) GetAPIKeyTags(ctx context.Context, apiKeyID string) ([]Tag, error) { return nil, nil }

type pgProviderRepo struct{ db *sql.DB }

func (r *pgProviderRepo) Create(ctx context.Context, p *Provider) error {
	query := `INSERT INTO providers (id, workspace_id, slug, name, provider_type, base_url, api_key_encrypted, config, status, priority, weight, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err := r.db.ExecContext(ctx, query, p.ID, p.WorkspaceID, p.Slug, p.Name, p.ProviderType, p.BaseURL, p.APIKeyEncrypted, p.Config, p.Status, p.Priority, p.Weight, p.CreatedAt, p.UpdatedAt)
	return err
}
func (r *pgProviderRepo) GetByID(ctx context.Context, id string) (*Provider, error) { return nil, nil }
func (r *pgProviderRepo) GetBySlug(ctx context.Context, workspaceID, slug string) (*Provider, error) { return nil, nil }
func (r *pgProviderRepo) GetByWorkspace(ctx context.Context, workspaceID string) ([]Provider, error) { return nil, nil }
func (r *pgProviderRepo) GetByTags(ctx context.Context, workspaceID string, tagIDs []string) ([]Provider, error) { return nil, nil }
func (r *pgProviderRepo) Update(ctx context.Context, p *Provider) error { return nil }
func (r *pgProviderRepo) Delete(ctx context.Context, id string) error { return nil }
func (r *pgProviderRepo) UpdateHealth(ctx context.Context, health *ProviderHealth) error {
	query := `INSERT INTO provider_health (provider_id, healthy, last_check_at, last_success_at, consecutive_failures, latency_ms, error_message, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			  ON CONFLICT (provider_id) DO UPDATE SET
			    healthy = EXCLUDED.healthy,
			    last_check_at = EXCLUDED.last_check_at,
			    last_success_at = EXCLUDED.last_success_at,
			    consecutive_failures = EXCLUDED.consecutive_failures,
			    latency_ms = EXCLUDED.latency_ms,
			    error_message = EXCLUDED.error_message,
			    updated_at = EXCLUDED.updated_at`
	_, err := r.db.ExecContext(ctx, query, health.ProviderID, health.Healthy, health.LastCheckAt, health.LastSuccessAt, health.ConsecutiveFailures, health.LatencyMs, health.ErrorMessage, health.UpdatedAt)
	return err
}
func (r *pgProviderRepo) GetHealth(ctx context.Context, providerID string) (*ProviderHealth, error) { return nil, nil }
func (r *pgProviderRepo) UpdateCircuitBreaker(ctx context.Context, state *CircuitBreakerState) error {
	query := `INSERT INTO circuit_breaker_states (provider_id, state, failures, successes, last_failure_at, half_open_requests, opened_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			  ON CONFLICT (provider_id) DO UPDATE SET
			    state = EXCLUDED.state,
			    failures = EXCLUDED.failures,
			    successes = EXCLUDED.successes,
			    last_failure_at = EXCLUDED.last_failure_at,
			    half_open_requests = EXCLUDED.half_open_requests,
			    opened_at = EXCLUDED.opened_at,
			    updated_at = EXCLUDED.updated_at`
	_, err := r.db.ExecContext(ctx, query, state.ProviderID, state.State, state.Failures, state.Successes, state.LastFailureAt, state.HalfOpenRequests, state.OpenedAt, state.UpdatedAt)
	return err
}
func (r *pgProviderRepo) GetCircuitBreaker(ctx context.Context, providerID string) (*CircuitBreakerState, error) { return nil, nil }

type pgControlRoomRepo struct{ db *sql.DB }

func (r *pgControlRoomRepo) Create(ctx context.Context, room *ControlRoom) error {
	query := `INSERT INTO control_rooms (id, workspace_id, slug, name, description, tag_filter, dashboard_layout, created_by, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.db.ExecContext(ctx, query, room.ID, room.WorkspaceID, room.Slug, room.Name, room.Description, room.TagFilter, room.DashboardLayout, room.CreatedBy, room.CreatedAt, room.UpdatedAt)
	return err
}
func (r *pgControlRoomRepo) GetByID(ctx context.Context, id string) (*ControlRoom, error) { return nil, nil }
func (r *pgControlRoomRepo) GetBySlug(ctx context.Context, workspaceID, slug string) (*ControlRoom, error) { return nil, nil }
func (r *pgControlRoomRepo) GetByWorkspace(ctx context.Context, workspaceID string) ([]ControlRoom, error) { return nil, nil }
func (r *pgControlRoomRepo) Update(ctx context.Context, room *ControlRoom) error { return nil }
func (r *pgControlRoomRepo) Delete(ctx context.Context, id string) error { return nil }
func (r *pgControlRoomRepo) GrantAccess(ctx context.Context, access *ControlRoomAccess) error {
	query := `INSERT INTO control_room_access (control_room_id, user_id, role, granted_by, granted_at, expires_at)
			  VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, access.ControlRoomID, access.UserID, access.Role, access.GrantedBy, access.GrantedAt, access.ExpiresAt)
	return err
}
func (r *pgControlRoomRepo) RevokeAccess(ctx context.Context, controlRoomID, userID string) error { return nil }
func (r *pgControlRoomRepo) GetUserAccess(ctx context.Context, controlRoomID string) ([]ControlRoomAccess, error) { return nil, nil }

type pgAPIKeyRepo struct{ db *sql.DB }

func (r *pgAPIKeyRepo) Create(ctx context.Context, key *APIKey) error {
	query := `INSERT INTO api_keys (id, workspace_id, name, key_hash, key_preview, status, created_by, expires_at, last_used_at, rate_limit, allowed_models, allowed_apis, metadata, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	_, err := r.db.ExecContext(ctx, query, key.ID, key.WorkspaceID, key.Name, key.KeyHash, key.KeyPreview, key.Status, key.CreatedBy, key.ExpiresAt, key.LastUsedAt, key.RateLimit, key.AllowedModels, key.AllowedAPIs, key.Metadata, key.CreatedAt, key.UpdatedAt)
	return err
}
func (r *pgAPIKeyRepo) GetByID(ctx context.Context, id string) (*APIKey, error) { return nil, nil }
func (r *pgAPIKeyRepo) GetByHash(ctx context.Context, hash string) (*APIKey, error) { return nil, nil }
func (r *pgAPIKeyRepo) GetByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]APIKey, error) { return nil, nil }
func (r *pgAPIKeyRepo) Update(ctx context.Context, key *APIKey) error { return nil }
func (r *pgAPIKeyRepo) Delete(ctx context.Context, id string) error { return nil }
func (r *pgAPIKeyRepo) UpdateLastUsed(ctx context.Context, id string, t time.Time) error { return nil }

type pgQuotaRepo struct{ db *sql.DB }

func (r *pgQuotaRepo) Create(ctx context.Context, quota *Quota) error {
	query := `INSERT INTO quotas (id, workspace_id, name, description, quota_type, period, limit_value, scope, warning_threshold, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.db.ExecContext(ctx, query, quota.ID, quota.WorkspaceID, quota.Name, quota.Description, quota.QuotaType, quota.Period, quota.LimitValue, quota.Scope, quota.WarningThreshold, quota.CreatedAt, quota.UpdatedAt)
	return err
}
func (r *pgQuotaRepo) GetByID(ctx context.Context, id string) (*Quota, error) { return nil, nil }
func (r *pgQuotaRepo) GetByWorkspace(ctx context.Context, workspaceID string) ([]Quota, error) { return nil, nil }
func (r *pgQuotaRepo) Update(ctx context.Context, quota *Quota) error { return nil }
func (r *pgQuotaRepo) Delete(ctx context.Context, id string) error { return nil }
func (r *pgQuotaRepo) AssignQuota(ctx context.Context, assignment *QuotaAssignment) error {
	query := `INSERT INTO quota_assignments (quota_id, resource_type, resource_id, current_usage, period_start, period_end, warning_sent, exceeded_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(ctx, query, assignment.QuotaID, assignment.ResourceType, assignment.ResourceID, assignment.CurrentUsage, assignment.PeriodStart, assignment.PeriodEnd, assignment.WarningSent, assignment.ExceededAt, assignment.UpdatedAt)
	return err
}
func (r *pgQuotaRepo) GetAssignment(ctx context.Context, quotaID, resourceType, resourceID string) (*QuotaAssignment, error) { return nil, nil }
func (r *pgQuotaRepo) UpdateUsage(ctx context.Context, quotaID, resourceType, resourceID string, usage int64) error { return nil }
func (r *pgQuotaRepo) ResetUsage(ctx context.Context, quotaID, resourceType, resourceID string) error { return nil }
func (r *pgQuotaRepo) GetResourceAssignments(ctx context.Context, resourceType, resourceID string) ([]QuotaAssignment, error) { return nil, nil }

type pgUsageRecordRepo struct{ db *sql.DB }

func (r *pgUsageRecordRepo) Create(ctx context.Context, record *UsageRecord) error {
	query := `INSERT INTO usage_records (id, workspace_id, request_id, trace_id, api_key_id, control_room_id, incoming_api, incoming_model, selected_model, provider_id, prompt_tokens, completion_tokens, total_tokens, cost_usd, duration_ms, response_status, error_code, error_message, attempts, route_log, started_at, completed_at, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)`
	_, err := r.db.ExecContext(ctx, query, record.ID, record.WorkspaceID, record.RequestID, record.TraceID, record.APIKeyID, record.ControlRoomID, record.IncomingAPI, record.IncomingModel, record.SelectedModel, record.ProviderID, record.PromptTokens, record.CompletionTokens, record.TotalTokens, record.CostUSD, record.DurationMs, record.ResponseStatus, record.ErrorCode, record.ErrorMessage, record.Attempts, record.RouteLog, record.StartedAt, record.CompletedAt, record.CreatedAt)
	return err
}
func (r *pgUsageRecordRepo) GetByID(ctx context.Context, id string) (*UsageRecord, error) { return nil, nil }
func (r *pgUsageRecordRepo) GetByRequestID(ctx context.Context, requestID string) (*UsageRecord, error) { return nil, nil }
func (r *pgUsageRecordRepo) GetByWorkspace(ctx context.Context, workspaceID string, start, end time.Time, limit, offset int) ([]UsageRecord, error) { return nil, nil }
func (r *pgUsageRecordRepo) GetByAPIKey(ctx context.Context, apiKeyID string, start, end time.Time, limit, offset int) ([]UsageRecord, error) { return nil, nil }
func (r *pgUsageRecordRepo) Update(ctx context.Context, record *UsageRecord) error { return nil }
func (r *pgUsageRecordRepo) GetSummaryByWorkspace(ctx context.Context, workspaceID string, start, end time.Time) (*UsageSummary, error) { return nil, nil }

type pgTraceEventRepo struct{ db *sql.DB }

func (r *pgTraceEventRepo) Create(ctx context.Context, event *TraceEvent) error {
	query := `INSERT INTO trace_events (id, trace_id, request_id, event_type, event_order, provider_id, api_key_id, message, metadata, timestamp, duration_ms, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := r.db.ExecContext(ctx, query, event.ID, event.TraceID, event.RequestID, event.EventType, event.EventOrder, event.ProviderID, event.APIKeyID, event.Message, event.Metadata, event.Timestamp, event.DurationMs, event.CreatedAt)
	return err
}
func (r *pgTraceEventRepo) GetByTraceID(ctx context.Context, traceID string) ([]TraceEvent, error) { return nil, nil }
func (r *pgTraceEventRepo) GetByRequestID(ctx context.Context, requestID string) ([]TraceEvent, error) { return nil, nil }
func (r *pgTraceEventRepo) CreateBatch(ctx context.Context, events []TraceEvent) error {
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
