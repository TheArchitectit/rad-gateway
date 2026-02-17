-- Initial schema for RAD Gateway
-- Compatible with both SQLite and PostgreSQL

-- Workspaces
CREATE TABLE IF NOT EXISTS workspaces (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    settings BLOB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_workspaces_slug ON workspaces(slug);
CREATE INDEX IF NOT EXISTS idx_workspaces_status ON workspaces(status);

-- Users
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    email TEXT NOT NULL,
    display_name TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    password_hash TEXT,
    last_login_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_users_workspace ON users(workspace_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_workspace_email ON users(workspace_id, email);

-- Permissions
CREATE TABLE IF NOT EXISTS permissions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    action TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_permissions_resource ON permissions(resource_type, action);

-- Roles
CREATE TABLE IF NOT EXISTS roles (
    id TEXT PRIMARY KEY,
    workspace_id TEXT,
    name TEXT NOT NULL,
    description TEXT,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_roles_workspace ON roles(workspace_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_workspace_name ON roles(workspace_id, name) WHERE workspace_id IS NOT NULL;

-- User Roles (many-to-many)
CREATE TABLE IF NOT EXISTS user_roles (
    user_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    granted_by TEXT,
    granted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(role_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_expires ON user_roles(expires_at);

-- Role Permissions (many-to-many)
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id TEXT NOT NULL,
    permission_id TEXT NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_permission ON role_permissions(permission_id);

-- Tags
CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    category TEXT NOT NULL,
    value TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tags_workspace ON tags(workspace_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_workspace_category_value ON tags(workspace_id, category, value);
CREATE INDEX IF NOT EXISTS idx_tags_category ON tags(category);

-- Providers
CREATE TABLE IF NOT EXISTS providers (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    provider_type TEXT NOT NULL,
    base_url TEXT NOT NULL,
    api_key_encrypted TEXT,
    config BLOB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'active',
    priority INTEGER NOT NULL DEFAULT 0,
    weight INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_providers_workspace ON providers(workspace_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_providers_workspace_slug ON providers(workspace_id, slug);
CREATE INDEX IF NOT EXISTS idx_providers_status ON providers(status);
CREATE INDEX IF NOT EXISTS idx_providers_type ON providers(provider_type);

-- Provider Tags (many-to-many)
CREATE TABLE IF NOT EXISTS provider_tags (
    provider_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    PRIMARY KEY (provider_id, tag_id),
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_provider_tags_tag ON provider_tags(tag_id);

-- Provider Health
CREATE TABLE IF NOT EXISTS provider_health (
    provider_id TEXT PRIMARY KEY,
    healthy BOOLEAN NOT NULL DEFAULT TRUE,
    last_check_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_success_at TIMESTAMP,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    latency_ms INTEGER,
    error_message TEXT,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
);

-- Circuit Breaker State
CREATE TABLE IF NOT EXISTS circuit_breaker_states (
    provider_id TEXT PRIMARY KEY,
    state TEXT NOT NULL DEFAULT 'closed',
    failures INTEGER NOT NULL DEFAULT 0,
    successes INTEGER NOT NULL DEFAULT 0,
    last_failure_at TIMESTAMP,
    half_open_requests INTEGER NOT NULL DEFAULT 0,
    opened_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
);

-- Control Rooms
CREATE TABLE IF NOT EXISTS control_rooms (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    tag_filter TEXT NOT NULL,
    dashboard_layout BLOB NOT NULL DEFAULT '{}',
    created_by TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_control_rooms_workspace ON control_rooms(workspace_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_control_rooms_workspace_slug ON control_rooms(workspace_id, slug);

-- Control Room Access
CREATE TABLE IF NOT EXISTS control_room_access (
    control_room_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL,
    granted_by TEXT,
    granted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    PRIMARY KEY (control_room_id, user_id),
    FOREIGN KEY (control_room_id) REFERENCES control_rooms(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_control_room_access_user ON control_room_access(user_id);
CREATE INDEX IF NOT EXISTS idx_control_room_access_expires ON control_room_access(expires_at);

-- API Keys
CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    key_preview TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_by TEXT,
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    rate_limit INTEGER,
    allowed_models TEXT, -- JSON array stored as text
    allowed_apis TEXT,   -- JSON array stored as text
    metadata BLOB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_api_keys_workspace ON api_keys(workspace_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires ON api_keys(expires_at);

-- API Key Tags (many-to-many)
CREATE TABLE IF NOT EXISTS api_key_tags (
    api_key_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    PRIMARY KEY (api_key_id, tag_id),
    FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_api_key_tags_tag ON api_key_tags(tag_id);

-- Quotas
CREATE TABLE IF NOT EXISTS quotas (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    quota_type TEXT NOT NULL,
    period TEXT NOT NULL,
    limit_value INTEGER NOT NULL,
    scope TEXT NOT NULL,
    warning_threshold INTEGER NOT NULL DEFAULT 80,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_quotas_workspace ON quotas(workspace_id);
CREATE INDEX IF NOT EXISTS idx_quotas_type ON quotas(quota_type);

-- Quota Assignments
CREATE TABLE IF NOT EXISTS quota_assignments (
    quota_id TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    current_usage INTEGER NOT NULL DEFAULT 0,
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    warning_sent BOOLEAN NOT NULL DEFAULT FALSE,
    exceeded_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (quota_id, resource_type, resource_id),
    FOREIGN KEY (quota_id) REFERENCES quotas(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_quota_assignments_resource ON quota_assignments(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_quota_assignments_period ON quota_assignments(period_start, period_end);

-- Usage Records
CREATE TABLE IF NOT EXISTS usage_records (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    request_id TEXT NOT NULL,
    trace_id TEXT NOT NULL,
    api_key_id TEXT,
    control_room_id TEXT,
    incoming_api TEXT NOT NULL,
    incoming_model TEXT NOT NULL,
    selected_model TEXT,
    provider_id TEXT,
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    cost_usd REAL,
    duration_ms INTEGER NOT NULL,
    response_status TEXT NOT NULL,
    error_code TEXT,
    error_message TEXT,
    attempts INTEGER NOT NULL DEFAULT 1,
    route_log BLOB NOT NULL DEFAULT '{}',
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE SET NULL,
    FOREIGN KEY (control_room_id) REFERENCES control_rooms(id) ON DELETE SET NULL,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_usage_records_workspace ON usage_records(workspace_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_request ON usage_records(request_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_trace ON usage_records(trace_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_api_key ON usage_records(api_key_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_provider ON usage_records(provider_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_created ON usage_records(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_records_status ON usage_records(response_status);

-- Usage Record Tags (many-to-many)
CREATE TABLE IF NOT EXISTS usage_record_tags (
    usage_record_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    PRIMARY KEY (usage_record_id, tag_id),
    FOREIGN KEY (usage_record_id) REFERENCES usage_records(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_usage_record_tags_tag ON usage_record_tags(tag_id);

-- Trace Events
CREATE TABLE IF NOT EXISTS trace_events (
    id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL,
    request_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    event_order INTEGER NOT NULL,
    provider_id TEXT,
    api_key_id TEXT,
    message TEXT NOT NULL,
    metadata BLOB NOT NULL DEFAULT '{}',
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    duration_ms INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE SET NULL,
    FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_trace_events_trace ON trace_events(trace_id);
CREATE INDEX IF NOT EXISTS idx_trace_events_request ON trace_events(request_id);
CREATE INDEX IF NOT EXISTS idx_trace_events_timestamp ON trace_events(timestamp);

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
