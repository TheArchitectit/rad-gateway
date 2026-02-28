// Package db provides database interfaces and implementations for RAD Gateway.
package db

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"log"
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
	modelCards   *pgModelCardRepo
	auditLog     *pgAuditLogRepo
}

// NewPostgres creates a new PostgreSQL database connection with retry logic.
func NewPostgres(config Config) (*PostgresDB, error) {
	if config.DSN == "" {
		return nil, fmt.Errorf("PostgreSQL DSN is required")
	}

	// Configure connection pool with production-ready defaults
	// Increased from 10 to 25 for better throughput under load
	maxOpenConns := config.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = 25 // Production default: 25 connections
	}
	maxIdleConns := config.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 10 // 40% of max open for better connection reuse
	}
	connMaxLifetime := config.ConnMaxLifetime
	if connMaxLifetime <= 0 {
		connMaxLifetime = 15 * time.Minute // Extended for production stability
	}
	connMaxIdleTime := config.ConnMaxIdleTime
	if connMaxIdleTime <= 0 {
		connMaxIdleTime = 5 * time.Minute // Close idle connections after 5 min
	}

	// Retry logic: 3 attempts with exponential backoff
	var db *sql.DB
	var err error
	maxRetries := 3
	retryDelay := 1 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		db, err = sql.Open("postgres", config.DSN)
		if err != nil {
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				retryDelay *= 2 // Exponential backoff
				continue
			}
			return nil, fmt.Errorf("failed to open postgres database after %d attempts: %w", maxRetries, err)
		}

		// Configure connection pool
		db.SetMaxOpenConns(maxOpenConns)
		db.SetMaxIdleConns(maxIdleConns)
		db.SetConnMaxLifetime(connMaxLifetime)
		db.SetConnMaxIdleTime(connMaxIdleTime)

		// Log pool configuration
		log.Printf("[db] pool configured: max_open=%d max_idle=%d lifetime=%v idle_time=%v",
			maxOpenConns, maxIdleConns, connMaxLifetime, connMaxIdleTime)

		// Test connection with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = db.PingContext(ctx)
		cancel()

		if err == nil {
			break // Success
		}

		db.Close()

		if attempt < maxRetries {
			fmt.Printf("PostgreSQL connection attempt %d/%d failed: %v. Retrying in %v...\n", attempt, maxRetries, err, retryDelay)
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		} else {
			return nil, fmt.Errorf("failed to connect to postgres after %d attempts: %w", maxRetries, err)
		}
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
		modelCards:   &pgModelCardRepo{db: db},
		auditLog:     &pgAuditLogRepo{db: db},
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
func (p *PostgresDB) ModelCards() ModelCardRepository     { return p.repos.modelCards }
func (p *PostgresDB) AuditLog() AuditLogRepository         { return p.repos.auditLog }

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
func (r *pgRoleRepo) GetByWorkspace(ctx context.Context, workspaceID *string) ([]Role, error) {
	return nil, nil
}
func (r *pgRoleRepo) Update(ctx context.Context, role *Role) error { return nil }
func (r *pgRoleRepo) Delete(ctx context.Context, id string) error  { return nil }
func (r *pgRoleRepo) AssignToUser(ctx context.Context, userID, roleID string, grantedBy *string, expiresAt *time.Time) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO user_roles (user_id, role_id, granted_by, expires_at) VALUES ($1, $2, $3, $4)`, userID, roleID, grantedBy, expiresAt)
	return err
}
func (r *pgRoleRepo) RemoveFromUser(ctx context.Context, userID, roleID string) error { return nil }
func (r *pgRoleRepo) GetUserRoles(ctx context.Context, userID string) ([]Role, error) {
	return nil, nil
}

type pgPermissionRepo struct{ db *sql.DB }

func (r *pgPermissionRepo) Create(ctx context.Context, p *Permission) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO permissions (id, name, description, resource_type, action) VALUES ($1, $2, $3, $4, $5)`,
		p.ID, p.Name, p.Description, p.ResourceType, p.Action)
	return err
}
func (r *pgPermissionRepo) GetByID(ctx context.Context, id string) (*Permission, error) {
	return nil, nil
}
func (r *pgPermissionRepo) GetByName(ctx context.Context, name string) (*Permission, error) {
	return nil, nil
}
func (r *pgPermissionRepo) List(ctx context.Context) ([]Permission, error) { return nil, nil }
func (r *pgPermissionRepo) AssignToRole(ctx context.Context, roleID, permissionID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)`, roleID, permissionID)
	return err
}
func (r *pgPermissionRepo) RemoveFromRole(ctx context.Context, roleID, permissionID string) error {
	return nil
}
func (r *pgPermissionRepo) GetRolePermissions(ctx context.Context, roleID string) ([]Permission, error) {
	return nil, nil
}
func (r *pgPermissionRepo) GetUserPermissions(ctx context.Context, userID string) ([]Permission, error) {
	return nil, nil
}

type pgTagRepo struct{ db *sql.DB }

func (r *pgTagRepo) Create(ctx context.Context, tag *Tag) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO tags (id, workspace_id, category, value, description, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		tag.ID, tag.WorkspaceID, tag.Category, tag.Value, tag.Description, tag.CreatedAt)
	return err
}
func (r *pgTagRepo) GetByID(ctx context.Context, id string) (*Tag, error) { return nil, nil }
func (r *pgTagRepo) GetByCategoryValue(ctx context.Context, workspaceID, category, value string) (*Tag, error) {
	return nil, nil
}
func (r *pgTagRepo) GetByWorkspace(ctx context.Context, workspaceID string) ([]Tag, error) {
	return nil, nil
}
func (r *pgTagRepo) Delete(ctx context.Context, id string) error { return nil }
func (r *pgTagRepo) AssignToProvider(ctx context.Context, providerID, tagID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO provider_tags (provider_id, tag_id) VALUES ($1, $2)`, providerID, tagID)
	return err
}
func (r *pgTagRepo) RemoveFromProvider(ctx context.Context, providerID, tagID string) error {
	return nil
}
func (r *pgTagRepo) GetProviderTags(ctx context.Context, providerID string) ([]Tag, error) {
	return nil, nil
}
func (r *pgTagRepo) AssignToAPIKey(ctx context.Context, apiKeyID, tagID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO api_key_tags (api_key_id, tag_id) VALUES ($1, $2)`, apiKeyID, tagID)
	return err
}
func (r *pgTagRepo) RemoveFromAPIKey(ctx context.Context, apiKeyID, tagID string) error { return nil }
func (r *pgTagRepo) GetAPIKeyTags(ctx context.Context, apiKeyID string) ([]Tag, error) {
	return nil, nil
}

type pgProviderRepo struct{ db *sql.DB }

func (r *pgProviderRepo) Create(ctx context.Context, p *Provider) error {
	query := `INSERT INTO providers (id, workspace_id, slug, name, provider_type, base_url, api_key_encrypted, config, status, priority, weight, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err := r.db.ExecContext(ctx, query, p.ID, p.WorkspaceID, p.Slug, p.Name, p.ProviderType, p.BaseURL, p.APIKeyEncrypted, p.Config, p.Status, p.Priority, p.Weight, p.CreatedAt, p.UpdatedAt)
	return err
}
func (r *pgProviderRepo) GetByID(ctx context.Context, id string) (*Provider, error) {
	p := &Provider{}
	var apiKey sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT id, workspace_id, slug, name, provider_type, base_url, api_key_encrypted, config, status, priority, weight, created_at, updated_at FROM providers WHERE id = $1`, id).
		Scan(&p.ID, &p.WorkspaceID, &p.Slug, &p.Name, &p.ProviderType, &p.BaseURL, &apiKey, &p.Config, &p.Status, &p.Priority, &p.Weight, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if apiKey.Valid {
		p.APIKeyEncrypted = &apiKey.String
	}
	return p, nil
}

func (r *pgProviderRepo) GetBySlug(ctx context.Context, workspaceID, slug string) (*Provider, error) {
	p := &Provider{}
	var apiKey sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT id, workspace_id, slug, name, provider_type, base_url, api_key_encrypted, config, status, priority, weight, created_at, updated_at FROM providers WHERE workspace_id = $1 AND slug = $2`, workspaceID, slug).
		Scan(&p.ID, &p.WorkspaceID, &p.Slug, &p.Name, &p.ProviderType, &p.BaseURL, &apiKey, &p.Config, &p.Status, &p.Priority, &p.Weight, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if apiKey.Valid {
		p.APIKeyEncrypted = &apiKey.String
	}
	return p, nil
}

func (r *pgProviderRepo) GetByWorkspace(ctx context.Context, workspaceID string) ([]Provider, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, workspace_id, slug, name, provider_type, base_url, api_key_encrypted, config, status, priority, weight, created_at, updated_at FROM providers WHERE workspace_id = $1 ORDER BY created_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	providers := make([]Provider, 0)
	for rows.Next() {
		var p Provider
		var apiKey sql.NullString
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.Slug, &p.Name, &p.ProviderType, &p.BaseURL, &apiKey, &p.Config, &p.Status, &p.Priority, &p.Weight, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if apiKey.Valid {
			p.APIKeyEncrypted = &apiKey.String
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

func (r *pgProviderRepo) GetByTags(ctx context.Context, workspaceID string, tagIDs []string) ([]Provider, error) {
	if len(tagIDs) == 0 {
		return r.GetByWorkspace(ctx, workspaceID)
	}
	placeholders := make([]string, len(tagIDs))
	args := make([]interface{}, 0, len(tagIDs)+1)
	args = append(args, workspaceID)
	for i, id := range tagIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	query := fmt.Sprintf(`SELECT DISTINCT p.id, p.workspace_id, p.slug, p.name, p.provider_type, p.base_url, p.api_key_encrypted, p.config, p.status, p.priority, p.weight, p.created_at, p.updated_at FROM providers p JOIN provider_tags pt ON p.id = pt.provider_id WHERE p.workspace_id = $1 AND pt.tag_id IN (%s) ORDER BY p.created_at DESC`, strings.Join(placeholders, ","))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	providers := make([]Provider, 0)
	for rows.Next() {
		var p Provider
		var apiKey sql.NullString
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.Slug, &p.Name, &p.ProviderType, &p.BaseURL, &apiKey, &p.Config, &p.Status, &p.Priority, &p.Weight, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if apiKey.Valid {
			p.APIKeyEncrypted = &apiKey.String
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

func (r *pgProviderRepo) Update(ctx context.Context, p *Provider) error {
	_, err := r.db.ExecContext(ctx, `UPDATE providers SET slug = $1, name = $2, provider_type = $3, base_url = $4, api_key_encrypted = $5, config = $6, status = $7, priority = $8, weight = $9, updated_at = $10 WHERE id = $11`, p.Slug, p.Name, p.ProviderType, p.BaseURL, p.APIKeyEncrypted, p.Config, p.Status, p.Priority, p.Weight, p.UpdatedAt, p.ID)
	return err
}

func (r *pgProviderRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM providers WHERE id = $1`, id)
	return err
}
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
func (r *pgProviderRepo) GetHealth(ctx context.Context, providerID string) (*ProviderHealth, error) {
	h := &ProviderHealth{}
	var lastSuccess sql.NullTime
	var latency sql.NullInt64
	var errMsg sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT provider_id, healthy, last_check_at, last_success_at, consecutive_failures, latency_ms, error_message, updated_at FROM provider_health WHERE provider_id = $1`, providerID).
		Scan(&h.ProviderID, &h.Healthy, &h.LastCheckAt, &lastSuccess, &h.ConsecutiveFailures, &latency, &errMsg, &h.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastSuccess.Valid {
		t := lastSuccess.Time
		h.LastSuccessAt = &t
	}
	if latency.Valid {
		v := int(latency.Int64)
		h.LatencyMs = &v
	}
	if errMsg.Valid {
		s := errMsg.String
		h.ErrorMessage = &s
	}
	return h, nil
}
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
func (r *pgProviderRepo) GetCircuitBreaker(ctx context.Context, providerID string) (*CircuitBreakerState, error) {
	cb := &CircuitBreakerState{}
	var lastFailure sql.NullTime
	var openedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT provider_id, state, failures, successes, last_failure_at, half_open_requests, opened_at, updated_at FROM circuit_breaker_states WHERE provider_id = $1`, providerID).
		Scan(&cb.ProviderID, &cb.State, &cb.Failures, &cb.Successes, &lastFailure, &cb.HalfOpenRequests, &openedAt, &cb.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastFailure.Valid {
		t := lastFailure.Time
		cb.LastFailureAt = &t
	}
	if openedAt.Valid {
		t := openedAt.Time
		cb.OpenedAt = &t
	}
	return cb, nil
}

type pgControlRoomRepo struct{ db *sql.DB }

func (r *pgControlRoomRepo) Create(ctx context.Context, room *ControlRoom) error {
	query := `INSERT INTO control_rooms (id, workspace_id, slug, name, description, tag_filter, dashboard_layout, created_by, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.db.ExecContext(ctx, query, room.ID, room.WorkspaceID, room.Slug, room.Name, room.Description, room.TagFilter, room.DashboardLayout, room.CreatedBy, room.CreatedAt, room.UpdatedAt)
	return err
}
func (r *pgControlRoomRepo) GetByID(ctx context.Context, id string) (*ControlRoom, error) {
	return nil, nil
}
func (r *pgControlRoomRepo) GetBySlug(ctx context.Context, workspaceID, slug string) (*ControlRoom, error) {
	return nil, nil
}
func (r *pgControlRoomRepo) GetByWorkspace(ctx context.Context, workspaceID string) ([]ControlRoom, error) {
	return nil, nil
}
func (r *pgControlRoomRepo) Update(ctx context.Context, room *ControlRoom) error { return nil }
func (r *pgControlRoomRepo) Delete(ctx context.Context, id string) error         { return nil }
func (r *pgControlRoomRepo) GrantAccess(ctx context.Context, access *ControlRoomAccess) error {
	query := `INSERT INTO control_room_access (control_room_id, user_id, role, granted_by, granted_at, expires_at)
			  VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, access.ControlRoomID, access.UserID, access.Role, access.GrantedBy, access.GrantedAt, access.ExpiresAt)
	return err
}
func (r *pgControlRoomRepo) RevokeAccess(ctx context.Context, controlRoomID, userID string) error {
	return nil
}
func (r *pgControlRoomRepo) GetUserAccess(ctx context.Context, controlRoomID string) ([]ControlRoomAccess, error) {
	return nil, nil
}

type pgAPIKeyRepo struct{ db *sql.DB }

func (r *pgAPIKeyRepo) Create(ctx context.Context, key *APIKey) error {
	allowedModelsJSON, err := json.Marshal(key.AllowedModels)
	if err != nil {
		return err
	}
	allowedAPIsJSON, err := json.Marshal(key.AllowedAPIs)
	if err != nil {
		return err
	}
	query := `INSERT INTO api_keys (id, workspace_id, name, key_hash, key_preview, status, created_by, expires_at, last_used_at, rate_limit, allowed_models, allowed_apis, metadata, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	_, err = r.db.ExecContext(ctx, query, key.ID, key.WorkspaceID, key.Name, key.KeyHash, key.KeyPreview, key.Status, key.CreatedBy, key.ExpiresAt, key.LastUsedAt, key.RateLimit, string(allowedModelsJSON), string(allowedAPIsJSON), key.Metadata, key.CreatedAt, key.UpdatedAt)
	return err
}
func (r *pgAPIKeyRepo) GetByID(ctx context.Context, id string) (*APIKey, error) {
	return r.getOne(ctx, `SELECT id, workspace_id, name, key_hash, key_preview, status, created_by, expires_at, last_used_at, rate_limit, allowed_models, allowed_apis, metadata, created_at, updated_at FROM api_keys WHERE id = $1`, id)
}

func (r *pgAPIKeyRepo) GetByHash(ctx context.Context, hash string) (*APIKey, error) {
	return r.getOne(ctx, `SELECT id, workspace_id, name, key_hash, key_preview, status, created_by, expires_at, last_used_at, rate_limit, allowed_models, allowed_apis, metadata, created_at, updated_at FROM api_keys WHERE key_hash = $1`, hash)
}

func (r *pgAPIKeyRepo) GetByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]APIKey, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, workspace_id, name, key_hash, key_preview, status, created_by, expires_at, last_used_at, rate_limit, allowed_models, allowed_apis, metadata, created_at, updated_at FROM api_keys WHERE workspace_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, workspaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]APIKey, 0)
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *key)
	}
	return items, rows.Err()
}

func (r *pgAPIKeyRepo) Update(ctx context.Context, key *APIKey) error {
	allowedModelsJSON, err := json.Marshal(key.AllowedModels)
	if err != nil {
		return err
	}
	allowedAPIsJSON, err := json.Marshal(key.AllowedAPIs)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `UPDATE api_keys SET name = $1, key_hash = $2, key_preview = $3, status = $4, created_by = $5, expires_at = $6, last_used_at = $7, rate_limit = $8, allowed_models = $9, allowed_apis = $10, metadata = $11, updated_at = $12 WHERE id = $13`, key.Name, key.KeyHash, key.KeyPreview, key.Status, key.CreatedBy, key.ExpiresAt, key.LastUsedAt, key.RateLimit, string(allowedModelsJSON), string(allowedAPIsJSON), key.Metadata, key.UpdatedAt, key.ID)
	return err
}

func (r *pgAPIKeyRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = $1`, id)
	return err
}

func (r *pgAPIKeyRepo) UpdateLastUsed(ctx context.Context, id string, t time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE api_keys SET last_used_at = $1, updated_at = $2 WHERE id = $3`, t, t, id)
	return err
}

func (r *pgAPIKeyRepo) getOne(ctx context.Context, query string, arg interface{}) (*APIKey, error) {
	key, err := scanAPIKey(r.db.QueryRowContext(ctx, query, arg))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return key, nil
}

type pgQuotaRepo struct{ db *sql.DB }

func (r *pgQuotaRepo) Create(ctx context.Context, quota *Quota) error {
	query := `INSERT INTO quotas (id, workspace_id, name, description, quota_type, period, limit_value, scope, warning_threshold, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.db.ExecContext(ctx, query, quota.ID, quota.WorkspaceID, quota.Name, quota.Description, quota.QuotaType, quota.Period, quota.LimitValue, quota.Scope, quota.WarningThreshold, quota.CreatedAt, quota.UpdatedAt)
	return err
}
func (r *pgQuotaRepo) GetByID(ctx context.Context, id string) (*Quota, error) { return nil, nil }
func (r *pgQuotaRepo) GetByWorkspace(ctx context.Context, workspaceID string) ([]Quota, error) {
	return nil, nil
}
func (r *pgQuotaRepo) Update(ctx context.Context, quota *Quota) error { return nil }
func (r *pgQuotaRepo) Delete(ctx context.Context, id string) error    { return nil }
func (r *pgQuotaRepo) AssignQuota(ctx context.Context, assignment *QuotaAssignment) error {
	query := `INSERT INTO quota_assignments (quota_id, resource_type, resource_id, current_usage, period_start, period_end, warning_sent, exceeded_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(ctx, query, assignment.QuotaID, assignment.ResourceType, assignment.ResourceID, assignment.CurrentUsage, assignment.PeriodStart, assignment.PeriodEnd, assignment.WarningSent, assignment.ExceededAt, assignment.UpdatedAt)
	return err
}
func (r *pgQuotaRepo) GetAssignment(ctx context.Context, quotaID, resourceType, resourceID string) (*QuotaAssignment, error) {
	return nil, nil
}
func (r *pgQuotaRepo) UpdateUsage(ctx context.Context, quotaID, resourceType, resourceID string, usage int64) error {
	return nil
}
func (r *pgQuotaRepo) ResetUsage(ctx context.Context, quotaID, resourceType, resourceID string) error {
	return nil
}
func (r *pgQuotaRepo) GetResourceAssignments(ctx context.Context, resourceType, resourceID string) ([]QuotaAssignment, error) {
	return nil, nil
}

type pgUsageRecordRepo struct{ db *sql.DB }

func (r *pgUsageRecordRepo) Create(ctx context.Context, record *UsageRecord) error {
	query := `INSERT INTO usage_records (id, workspace_id, request_id, trace_id, api_key_id, control_room_id, incoming_api, incoming_model, selected_model, provider_id, prompt_tokens, completion_tokens, total_tokens, cost_usd, duration_ms, response_status, error_code, error_message, attempts, route_log, started_at, completed_at, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)`
	_, err := r.db.ExecContext(ctx, query, record.ID, record.WorkspaceID, record.RequestID, record.TraceID, record.APIKeyID, record.ControlRoomID, record.IncomingAPI, record.IncomingModel, record.SelectedModel, record.ProviderID, record.PromptTokens, record.CompletionTokens, record.TotalTokens, record.CostUSD, record.DurationMs, record.ResponseStatus, record.ErrorCode, record.ErrorMessage, record.Attempts, record.RouteLog, record.StartedAt, record.CompletedAt, record.CreatedAt)
	return err
}
func (r *pgUsageRecordRepo) GetByID(ctx context.Context, id string) (*UsageRecord, error) {
	return r.getOne(ctx, `SELECT id, workspace_id, request_id, trace_id, api_key_id, control_room_id, incoming_api, incoming_model, selected_model, provider_id, prompt_tokens, completion_tokens, total_tokens, cost_usd, duration_ms, response_status, error_code, error_message, attempts, route_log, started_at, completed_at, created_at FROM usage_records WHERE id = $1`, id)
}

func (r *pgUsageRecordRepo) GetByRequestID(ctx context.Context, requestID string) (*UsageRecord, error) {
	return r.getOne(ctx, `SELECT id, workspace_id, request_id, trace_id, api_key_id, control_room_id, incoming_api, incoming_model, selected_model, provider_id, prompt_tokens, completion_tokens, total_tokens, cost_usd, duration_ms, response_status, error_code, error_message, attempts, route_log, started_at, completed_at, created_at FROM usage_records WHERE request_id = $1`, requestID)
}

func (r *pgUsageRecordRepo) GetByWorkspace(ctx context.Context, workspaceID string, start, end time.Time, limit, offset int) ([]UsageRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, workspace_id, request_id, trace_id, api_key_id, control_room_id, incoming_api, incoming_model, selected_model, provider_id, prompt_tokens, completion_tokens, total_tokens, cost_usd, duration_ms, response_status, error_code, error_message, attempts, route_log, started_at, completed_at, created_at FROM usage_records WHERE workspace_id = $1 AND created_at >= $2 AND created_at <= $3 ORDER BY created_at DESC LIMIT $4 OFFSET $5`, workspaceID, start, end, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]UsageRecord, 0)
	for rows.Next() {
		rec, err := scanUsageRecord(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *rec)
	}
	return items, rows.Err()
}

func (r *pgUsageRecordRepo) GetByAPIKey(ctx context.Context, apiKeyID string, start, end time.Time, limit, offset int) ([]UsageRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, workspace_id, request_id, trace_id, api_key_id, control_room_id, incoming_api, incoming_model, selected_model, provider_id, prompt_tokens, completion_tokens, total_tokens, cost_usd, duration_ms, response_status, error_code, error_message, attempts, route_log, started_at, completed_at, created_at FROM usage_records WHERE api_key_id = $1 AND created_at >= $2 AND created_at <= $3 ORDER BY created_at DESC LIMIT $4 OFFSET $5`, apiKeyID, start, end, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]UsageRecord, 0)
	for rows.Next() {
		rec, err := scanUsageRecord(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *rec)
	}
	return items, rows.Err()
}

func (r *pgUsageRecordRepo) Update(ctx context.Context, record *UsageRecord) error {
	_, err := r.db.ExecContext(ctx, `UPDATE usage_records SET selected_model = $1, provider_id = $2, prompt_tokens = $3, completion_tokens = $4, total_tokens = $5, cost_usd = $6, duration_ms = $7, response_status = $8, error_code = $9, error_message = $10, attempts = $11, route_log = $12, completed_at = $13 WHERE id = $14`, record.SelectedModel, record.ProviderID, record.PromptTokens, record.CompletionTokens, record.TotalTokens, record.CostUSD, record.DurationMs, record.ResponseStatus, record.ErrorCode, record.ErrorMessage, record.Attempts, record.RouteLog, record.CompletedAt, record.ID)
	return err
}

func (r *pgUsageRecordRepo) GetSummaryByWorkspace(ctx context.Context, workspaceID string, start, end time.Time) (*UsageSummary, error) {
	summary := &UsageSummary{}
	var avgDuration sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(total_tokens), 0), COALESCE(SUM(prompt_tokens), 0), COALESCE(SUM(completion_tokens), 0), COALESCE(SUM(cost_usd), 0), AVG(duration_ms), COALESCE(SUM(CASE WHEN response_status = 'success' THEN 1 ELSE 0 END), 0), COALESCE(SUM(CASE WHEN response_status != 'success' THEN 1 ELSE 0 END), 0) FROM usage_records WHERE workspace_id = $1 AND created_at >= $2 AND created_at <= $3`, workspaceID, start, end).
		Scan(&summary.TotalRequests, &summary.TotalTokens, &summary.TotalPromptTokens, &summary.TotalCompletionTokens, &summary.TotalCostUSD, &avgDuration, &summary.SuccessCount, &summary.ErrorCount)
	if err != nil {
		return nil, err
	}
	if avgDuration.Valid {
		summary.AvgDurationMs = int(avgDuration.Float64)
	}
	return summary, nil
}

func (r *pgUsageRecordRepo) getOne(ctx context.Context, query string, arg interface{}) (*UsageRecord, error) {
	rec, err := scanUsageRecord(r.db.QueryRowContext(ctx, query, arg))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rec, nil
}

type pgTraceEventRepo struct{ db *sql.DB }

func (r *pgTraceEventRepo) Create(ctx context.Context, event *TraceEvent) error {
	query := `INSERT INTO trace_events (id, trace_id, request_id, event_type, event_order, provider_id, api_key_id, message, metadata, timestamp, duration_ms, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := r.db.ExecContext(ctx, query, event.ID, event.TraceID, event.RequestID, event.EventType, event.EventOrder, event.ProviderID, event.APIKeyID, event.Message, event.Metadata, event.Timestamp, event.DurationMs, event.CreatedAt)
	return err
}
func (r *pgTraceEventRepo) GetByTraceID(ctx context.Context, traceID string) ([]TraceEvent, error) {
	return nil, nil
}
func (r *pgTraceEventRepo) GetByRequestID(ctx context.Context, requestID string) ([]TraceEvent, error) {
	return nil, nil
}
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

type pgModelCardRepo struct{ db *sql.DB }

func (r *pgModelCardRepo) GetByID(ctx context.Context, id string) (*ModelCard, error) {
	card := &ModelCard{}
	query := `SELECT id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at
		      FROM a2a_model_cards WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&card.ID, &card.WorkspaceID, &card.UserID, &card.Name, &card.Slug,
		&card.Description, &card.Card, &card.Version, &card.Status,
		&card.CreatedAt, &card.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return card, err
}

func (r *pgModelCardRepo) GetBySlug(ctx context.Context, workspaceID, slug string) (*ModelCard, error) {
	card := &ModelCard{}
	query := `SELECT id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at
		      FROM a2a_model_cards WHERE workspace_id = $1 AND slug = $2`
	err := r.db.QueryRowContext(ctx, query, workspaceID, slug).Scan(
		&card.ID, &card.WorkspaceID, &card.UserID, &card.Name, &card.Slug,
		&card.Description, &card.Card, &card.Version, &card.Status,
		&card.CreatedAt, &card.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return card, err
}

func (r *pgModelCardRepo) GetByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]ModelCard, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at
		      FROM a2a_model_cards WHERE workspace_id = $1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3`
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

func (r *pgModelCardRepo) GetByUser(ctx context.Context, userID string, limit, offset int) ([]ModelCard, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at
		      FROM a2a_model_cards WHERE user_id = $1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3`
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

func (r *pgModelCardRepo) Create(ctx context.Context, card *ModelCard) error {
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
		      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.db.ExecContext(ctx, query,
		card.ID, card.WorkspaceID, card.UserID, card.Name, card.Slug,
		card.Description, card.Card, card.Version, card.Status, card.CreatedAt, card.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create model card: %w", err)
	}
	return r.createVersion(ctx, card, nil, card.UserID)
}

func (r *pgModelCardRepo) Update(ctx context.Context, card *ModelCard, changeReason *string, updatedBy *string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	current, err := r.GetByID(ctx, card.ID)
	if err != nil {
		return fmt.Errorf("failed to get current card: %w", err)
	}
	if current == nil {
		return fmt.Errorf("model card not found: %s", card.ID)
	}

	card.Version = current.Version + 1
	card.UpdatedAt = time.Now()

	versionQuery := `INSERT INTO model_card_versions (id, model_card_id, workspace_id, user_id, version, name, slug, description, card, status, change_reason, created_by, created_at)
		              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err = tx.ExecContext(ctx, versionQuery,
		generateUUID(), current.ID, current.WorkspaceID, current.UserID,
		current.Version, current.Name, current.Slug, current.Description,
		current.Card, current.Status, changeReason, updatedBy, current.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create version record: %w", err)
	}

	updateQuery := `UPDATE a2a_model_cards
		            SET name = $1, slug = $2, description = $3, card = $4,
		                version = $5, status = $6, updated_at = $7
		            WHERE id = $8`
	_, err = tx.ExecContext(ctx, updateQuery,
		card.Name, card.Slug, card.Description, card.Card,
		card.Version, card.Status, card.UpdatedAt, card.ID)
	if err != nil {
		return fmt.Errorf("failed to update model card: %w", err)
	}

	return tx.Commit()
}

func (r *pgModelCardRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE a2a_model_cards SET status = 'deleted', updated_at = $1 WHERE id = $2`,
		time.Now(), id)
	return err
}

func (r *pgModelCardRepo) HardDelete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM a2a_model_cards WHERE id = $1`, id)
	return err
}

func (r *pgModelCardRepo) Search(ctx context.Context, params ModelCardSearchParams) ([]ModelCardSearchResult, error) {
	if params.Limit <= 0 {
		params.Limit = 100
	}

	query := `SELECT id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at
		      FROM a2a_model_cards WHERE workspace_id = $1`
	args := []interface{}{params.WorkspaceID}
	argCount := 1

	if params.Status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, params.Status)
	}

	if params.Capability != "" {
		argCount++
		query += fmt.Sprintf(" AND card->'capabilities'->>$%d = 'true'", argCount)
		args = append(args, params.Capability)
	}

	if params.HasSkill != "" {
		argCount++
		query += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM jsonb_array_elements(card->'skills') AS skill WHERE skill->>'id' = $%d)", argCount)
		args = append(args, params.HasSkill)
	}

	if params.URL != "" {
		argCount++
		query += fmt.Sprintf(" AND card->>'url' ILIKE $%d", argCount)
		args = append(args, "%"+params.URL+"%")
	}

	if params.Query != "" {
		argCount++
		query += fmt.Sprintf(" AND (name ILIKE $%d OR card->>'description' ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+params.Query+"%")
	}

	query += fmt.Sprintf(" ORDER BY updated_at DESC LIMIT $%d OFFSET $%d", argCount+1, argCount+2)
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

func (r *pgModelCardRepo) SearchByCapability(ctx context.Context, workspaceID string, capability string, limit, offset int) ([]ModelCard, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at
		      FROM a2a_model_cards
		      WHERE workspace_id = $1 AND card->'capabilities'->$2 = 'true'
		      ORDER BY updated_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.db.QueryContext(ctx, query, workspaceID, capability, limit, offset)
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

func (r *pgModelCardRepo) SearchBySkill(ctx context.Context, workspaceID string, skillID string, limit, offset int) ([]ModelCard, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, workspace_id, user_id, name, slug, description, card, version, status, created_at, updated_at
		      FROM a2a_model_cards
		      WHERE workspace_id = $1
		      AND EXISTS (SELECT 1 FROM jsonb_array_elements(card->'skills') AS skill WHERE skill->>'id' = $2)
		      ORDER BY updated_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.db.QueryContext(ctx, query, workspaceID, skillID, limit, offset)
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

func (r *pgModelCardRepo) GetVersions(ctx context.Context, modelCardID string) ([]ModelCardVersion, error) {
	query := `SELECT id, model_card_id, workspace_id, user_id, version, name, slug, description, card, status, change_reason, created_by, created_at
		      FROM model_card_versions
		      WHERE model_card_id = $1 ORDER BY version DESC`
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

func (r *pgModelCardRepo) GetVersion(ctx context.Context, modelCardID string, version int) (*ModelCardVersion, error) {
	v := &ModelCardVersion{}
	query := `SELECT id, model_card_id, workspace_id, user_id, version, name, slug, description, card, status, change_reason, created_by, created_at
		      FROM model_card_versions WHERE model_card_id = $1 AND version = $2`
	err := r.db.QueryRowContext(ctx, query, modelCardID, version).Scan(
		&v.ID, &v.ModelCardID, &v.WorkspaceID, &v.UserID, &v.Version,
		&v.Name, &v.Slug, &v.Description, &v.Card, &v.Status,
		&v.ChangeReason, &v.CreatedBy, &v.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return v, err
}

func (r *pgModelCardRepo) RestoreVersion(ctx context.Context, modelCardID string, version int, restoredBy *string) error {
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

func (r *pgModelCardRepo) createVersion(ctx context.Context, card *ModelCard, changeReason *string, createdBy *string) error {
	query := `INSERT INTO model_card_versions (id, model_card_id, workspace_id, user_id, version, name, slug, description, card, status, change_reason, created_by, created_at)
		      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err := r.db.ExecContext(ctx, query,
		generateUUID(), card.ID, card.WorkspaceID, card.UserID,
		card.Version, card.Name, card.Slug, card.Description,
		card.Card, card.Status, changeReason, createdBy, card.CreatedAt)
	return err
}

// generateUUID generates a new UUID (placeholder - use proper UUID library in production).
func generateUUID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}

// pgAuditLogRepo implements AuditLogRepository for PostgreSQL.
type pgAuditLogRepo struct{ db *sql.DB }

func (r *pgAuditLogRepo) Log(ctx context.Context, eventType string, severity string, actorType, actorID, actorName string, resourceType, resourceID string, action, result string, details map[string]interface{}) error {
	detailsJSON, _ := json.Marshal(details)
	query := `INSERT INTO audit_log (id, timestamp, type, severity, actor_type, actor_id, actor_name, resource_type, resource_id, action, result, details)
		VALUES ($1, NOW(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.db.ExecContext(ctx, query, generateUUID(), eventType, severity, actorType, actorID, actorName, resourceType, resourceID, action, result, detailsJSON)
	return err
}

func (r *pgAuditLogRepo) Query(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
	// Simplified implementation - returns empty for now
	return []map[string]interface{}{}, nil
}

func (r *pgAuditLogRepo) Count(ctx context.Context, filter map[string]interface{}) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM audit_log`
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

func (r *pgAuditLogRepo) PurgeOldEvents(ctx context.Context, retentionDays int) (int64, error) {
	query := `DELETE FROM audit_log WHERE timestamp < NOW() - INTERVAL '` + fmt.Sprintf("%d", retentionDays) + ` days'`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
