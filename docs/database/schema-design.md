# RAD Gateway Database Schema Design

**Date**: 2026-02-17
**Status**: Draft
**Author**: Schema Designer Agent
**Target Database**: PostgreSQL 15+

---

## Executive Summary

This document defines the database schema for RAD Gateway Phase 1, supporting:
- Usage tracking and cost attribution
- RBAC (users, roles, permissions, projects)
- Provider health and cooldown tracking
- Quota management
- API keys and attribution
- Control rooms and tagging

**Design Principles**:
- Simple, pragmatic schema that ships quickly
- Proper indexing for query performance
- Foreign key constraints for data integrity
- UUID primary keys for distributed compatibility
- JSONB for flexible metadata
- Timestamp auditing on all tables

---

## Entity Relationship Diagram

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   workspaces    │────<│  control_rooms  │>────│ control_room_   │
│                 │     │                 │     │ _tags           │
└─────────────────┘     └─────────────────┘     └─────────────────┘
         │
         │ has many
         ▼
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│     users       │────<│  user_roles     │>────│     roles       │
│                 │     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
         │                                               │
         │                                               │ has many
         │                                               ▼
         │                                       ┌─────────────────┐
         │                                       │ role_permissions│
         │                                       │                 │
         │                                       └─────────────────┘
         │                                               │
         │                                               ▼
         │                                       ┌─────────────────┐
         │                                       │   permissions   │
         │                                       │                 │
         │                                       └─────────────────┘
         │
         │ has many
         ▼
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    api_keys     │────<│ api_key_tags    │>────│      tags       │
│                 │     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
         │
         │ used by
         ▼
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  usage_records  │>───┐│ provider_health │     │   providers     │
│                 │    ││                 │     │                 │
└─────────────────┘    │└─────────────────┘     └─────────────────┘
         │             │
         │ has many    │    ┌─────────────────┐
         ▼             └───<│ trace_events    │
┌─────────────────┐          │                 │
│ usage_record_   │          └─────────────────┘
│ _tags           │
└─────────────────┘
         ▲
         │
┌─────────────────┐
│ quota_assignments│
│                 │
└─────────────────┘
         ▲
         │
┌─────────────────┐     ┌─────────────────┐
│     quotas      │     │ circuit_breaker │
│                 │     │ _states         │
└─────────────────┘     └─────────────────┘
```

---

## Table Definitions

### 1. Workspaces

Multi-tenancy root entity. All resources belong to a workspace.

```sql
CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(32) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'archived')),
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_workspaces_status ON workspaces(status);
CREATE INDEX idx_workspaces_slug ON workspaces(slug);
```

**Go Struct**:
```go
type Workspace struct {
    ID          string    `db:"id" json:"id"`
    Slug        string    `db:"slug" json:"slug"`
    Name        string    `db:"name" json:"name"`
    Description *string   `db:"description" json:"description,omitempty"`
    Status      string    `db:"status" json:"status"`
    Settings    []byte    `db:"settings" json:"settings"`
    CreatedAt   time.Time `db:"created_at" json:"createdAt"`
    UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}
```

---

### 2. Users

User accounts for RBAC.

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    status VARCHAR(32) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended')),
    password_hash VARCHAR(255),  -- Nullable for SSO
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, email)
);

CREATE INDEX idx_users_workspace ON users(workspace_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
```

**Go Struct**:
```go
type User struct {
    ID           string     `db:"id" json:"id"`
    WorkspaceID  string     `db:"workspace_id" json:"workspaceId"`
    Email        string     `db:"email" json:"email"`
    DisplayName  *string    `db:"display_name" json:"displayName,omitempty"`
    Status       string     `db:"status" json:"status"`
    PasswordHash *string    `db:"password_hash" json:"-"`
    LastLoginAt  *time.Time `db:"last_login_at" json:"lastLoginAt,omitempty"`
    CreatedAt    time.Time  `db:"created_at" json:"createdAt"`
    UpdatedAt    time.Time  `db:"updated_at" json:"updatedAt"`
}
```

---

### 3. Roles

Role definitions for RBAC.

```sql
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(64) NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT FALSE,  -- Built-in roles cannot be deleted
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, name)
);

CREATE INDEX idx_roles_workspace ON roles(workspace_id);

-- Insert system roles
INSERT INTO roles (id, name, description, is_system) VALUES
    (gen_random_uuid(), 'admin', 'Full access to all resources', TRUE),
    (gen_random_uuid(), 'operator', 'Can manage routes and view dashboards', TRUE),
    (gen_random_uuid(), 'viewer', 'Read-only access', TRUE),
    (gen_random_uuid(), 'billing', 'Access to usage and cost reports', TRUE);
```

**Go Struct**:
```go
type Role struct {
    ID          string    `db:"id" json:"id"`
    WorkspaceID string    `db:"workspace_id" json:"workspaceId"`
    Name        string    `db:"name" json:"name"`
    Description *string   `db:"description" json:"description,omitempty"`
    IsSystem    bool      `db:"is_system" json:"isSystem"`
    CreatedAt   time.Time `db:"created_at" json:"createdAt"`
    UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}
```

---

### 4. Permissions

Individual permission definitions.

```sql
CREATE TABLE permissions (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    resource_type VARCHAR(32) NOT NULL,  -- provider, route, api_key, etc.
    action VARCHAR(32) NOT NULL  -- create, read, update, delete, execute
);

-- Insert system permissions
INSERT INTO permissions (id, name, description, resource_type, action) VALUES
    -- Provider permissions
    ('provider:read', 'View Providers', 'View provider details', 'provider', 'read'),
    ('provider:create', 'Create Providers', 'Add new providers', 'provider', 'create'),
    ('provider:update', 'Update Providers', 'Modify provider settings', 'provider', 'update'),
    ('provider:delete', 'Delete Providers', 'Remove providers', 'provider', 'delete'),
    -- Route permissions
    ('route:read', 'View Routes', 'View routing configuration', 'route', 'read'),
    ('route:create', 'Create Routes', 'Create new routes', 'route', 'create'),
    ('route:update', 'Update Routes', 'Modify routes', 'route', 'update'),
    ('route:delete', 'Delete Routes', 'Remove routes', 'route', 'delete'),
    -- API Key permissions
    ('api_key:read', 'View API Keys', 'View API key details', 'api_key', 'read'),
    ('api_key:create', 'Create API Keys', 'Create new API keys', 'api_key', 'create'),
    ('api_key:revoke', 'Revoke API Keys', 'Revoke API keys', 'api_key', 'delete'),
    -- Usage permissions
    ('usage:read', 'View Usage', 'View usage data', 'usage', 'read'),
    ('cost:read', 'View Costs', 'View cost reports', 'cost', 'read'),
    -- Control room permissions
    ('control_room:read', 'View Control Rooms', 'View control rooms', 'control_room', 'read'),
    ('control_room:create', 'Create Control Rooms', 'Create control rooms', 'control_room', 'create'),
    ('control_room:update', 'Update Control Rooms', 'Modify control rooms', 'control_room', 'update'),
    ('control_room:delete', 'Delete Control Rooms', 'Delete control rooms', 'control_room', 'delete'),
    -- Quota permissions
    ('quota:read', 'View Quotas', 'View quota settings', 'quota', 'read'),
    ('quota:manage', 'Manage Quotas', 'Set quotas', 'quota', 'update');
```

**Go Struct**:
```go
type Permission struct {
    ID           string `db:"id" json:"id"`
    Name         string `db:"name" json:"name"`
    Description  string `db:"description" json:"description"`
    ResourceType string `db:"resource_type" json:"resourceType"`
    Action       string `db:"action" json:"action"`
}
```

---

### 5. User Roles (Many-to-Many)

```sql
CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_role ON user_roles(role_id);
```

**Go Struct**:
```go
type UserRole struct {
    UserID    string     `db:"user_id" json:"userId"`
    RoleID    string     `db:"role_id" json:"roleId"`
    GrantedBy *string    `db:"granted_by" json:"grantedBy,omitempty"`
    GrantedAt time.Time  `db:"granted_at" json:"grantedAt"`
    ExpiresAt *time.Time `db:"expires_at" json:"expiresAt,omitempty"`
}
```

---

### 6. Role Permissions (Many-to-Many)

```sql
CREATE TABLE role_permissions (
    role_id VARCHAR(64) NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id VARCHAR(64) NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Assign permissions to system roles
-- Admin gets all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'admin';

-- Operator permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ((SELECT id FROM roles WHERE name = 'operator'), 'provider:read'),
    ((SELECT id FROM roles WHERE name = 'operator'), 'route:read'),
    ((SELECT id FROM roles WHERE name = 'operator'), 'route:update'),
    ((SELECT id FROM roles WHERE name = 'operator'), 'usage:read'),
    ((SELECT id FROM roles WHERE name = 'operator'), 'control_room:read');

-- Viewer permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ((SELECT id FROM roles WHERE name = 'viewer'), 'provider:read'),
    ((SELECT id FROM roles WHERE name = 'viewer'), 'route:read'),
    ((SELECT id FROM roles WHERE name = 'viewer'), 'usage:read'),
    ((SELECT id FROM roles WHERE name = 'viewer'), 'control_room:read');

-- Billing permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ((SELECT id FROM roles WHERE name = 'billing'), 'usage:read'),
    ((SELECT id FROM roles WHERE name = 'billing'), 'cost:read');
```

**Go Struct**:
```go
type RolePermission struct {
    RoleID       string `db:"role_id" json:"roleId"`
    PermissionID string `db:"permission_id" json:"permissionId"`
}
```

---

### 7. Tags

Hierarchical tags for resource categorization.

```sql
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    category VARCHAR(64) NOT NULL,
    value VARCHAR(128) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, category, value)
);

CREATE INDEX idx_tags_workspace ON tags(workspace_id);
CREATE INDEX idx_tags_category ON tags(category);
```

**Go Struct**:
```go
type Tag struct {
    ID          string    `db:"id" json:"id"`
    WorkspaceID string    `db:"workspace_id" json:"workspaceId"`
    Category    string    `db:"category" json:"category"`
    Value       string    `db:"value" json:"value"`
    Description *string   `db:"description" json:"description,omitempty"`
    CreatedAt   time.Time `db:"created_at" json:"createdAt"`
}

func (t Tag) String() string {
    return fmt.Sprintf("%s:%s", t.Category, t.Value)
}
```

---

### 8. Providers

Provider configurations.

```sql
CREATE TABLE providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    slug VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    provider_type VARCHAR(32) NOT NULL,  -- openai, anthropic, gemini
    base_url VARCHAR(512) NOT NULL,
    api_key_encrypted TEXT,  -- Encrypted API key
    config JSONB DEFAULT '{}',  -- Provider-specific config
    status VARCHAR(32) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'degraded')),
    priority INTEGER DEFAULT 100,  -- Lower = higher priority
    weight INTEGER DEFAULT 100,    -- Load balancing weight
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, slug)
);

CREATE INDEX idx_providers_workspace ON providers(workspace_id);
CREATE INDEX idx_providers_status ON providers(status);
CREATE INDEX idx_providers_type ON providers(provider_type);
```

**Go Struct**:
```go
type Provider struct {
    ID              string    `db:"id" json:"id"`
    WorkspaceID     string    `db:"workspace_id" json:"workspaceId"`
    Slug            string    `db:"slug" json:"slug"`
    Name            string    `db:"name" json:"name"`
    ProviderType    string    `db:"provider_type" json:"providerType"`
    BaseURL         string    `db:"base_url" json:"baseUrl"`
    APIKeyEncrypted *string   `db:"api_key_encrypted" json:"-"`
    Config          []byte    `db:"config" json:"config"`
    Status          string    `db:"status" json:"status"`
    Priority        int       `db:"priority" json:"priority"`
    Weight          int       `db:"weight" json:"weight"`
    CreatedAt       time.Time `db:"created_at" json:"createdAt"`
    UpdatedAt       time.Time `db:"updated_at" json:"updatedAt"`
}
```

---

### 9. Provider Tags (Many-to-Many)

```sql
CREATE TABLE provider_tags (
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (provider_id, tag_id)
);

CREATE INDEX idx_provider_tags_tag ON provider_tags(tag_id);
```

**Go Struct**:
```go
type ProviderTag struct {
    ProviderID string `db:"provider_id" json:"providerId"`
    TagID      string `db:"tag_id" json:"tagId"`
}
```

---

### 10. Provider Health

Health check status tracking.

```sql
CREATE TABLE provider_health (
    provider_id UUID PRIMARY KEY REFERENCES providers(id) ON DELETE CASCADE,
    healthy BOOLEAN NOT NULL DEFAULT TRUE,
    last_check_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_success_at TIMESTAMPTZ,
    consecutive_failures INTEGER DEFAULT 0,
    latency_ms INTEGER,
    error_message TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_provider_health_healthy ON provider_health(healthy);
CREATE INDEX idx_provider_health_last_check ON provider_health(last_check_at);
```

**Go Struct**:
```go
type ProviderHealth struct {
    ProviderID         string     `db:"provider_id" json:"providerId"`
    Healthy            bool       `db:"healthy" json:"healthy"`
    LastCheckAt        time.Time  `db:"last_check_at" json:"lastCheckAt"`
    LastSuccessAt      *time.Time `db:"last_success_at" json:"lastSuccessAt,omitempty"`
    ConsecutiveFailures int       `db:"consecutive_failures" json:"consecutiveFailures"`
    LatencyMs          *int       `db:"latency_ms" json:"latencyMs,omitempty"`
    ErrorMessage       *string    `db:"error_message" json:"errorMessage,omitempty"`
    UpdatedAt          time.Time  `db:"updated_at" json:"updatedAt"`
}
```

---

### 11. Circuit Breaker States

Circuit breaker state persistence.

```sql
CREATE TABLE circuit_breaker_states (
    provider_id UUID PRIMARY KEY REFERENCES providers(id) ON DELETE CASCADE,
    state VARCHAR(32) NOT NULL CHECK (state IN ('closed', 'open', 'half_open')),
    failures INTEGER DEFAULT 0,
    successes INTEGER DEFAULT 0,
    last_failure_at TIMESTAMPTZ,
    half_open_requests INTEGER DEFAULT 0,
    opened_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

**Go Struct**:
```go
type CircuitBreakerState struct {
    ProviderID         string     `db:"provider_id" json:"providerId"`
    State              string     `db:"state" json:"state"`
    Failures           int        `db:"failures" json:"failures"`
    Successes          int        `db:"successes" json:"successes"`
    LastFailureAt      *time.Time `db:"last_failure_at" json:"lastFailureAt,omitempty"`
    HalfOpenRequests   int        `db:"half_open_requests" json:"halfOpenRequests"`
    OpenedAt           *time.Time `db:"opened_at" json:"openedAt,omitempty"`
    UpdatedAt          time.Time  `db:"updated_at" json:"updatedAt"`
}
```

---

### 12. Control Rooms

Operational views with tag-based filtering.

```sql
CREATE TABLE control_rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    slug VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    tag_filter TEXT NOT NULL,  -- Query string like "env:production AND team:platform"
    dashboard_layout JSONB DEFAULT '{}',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, slug)
);

CREATE INDEX idx_control_rooms_workspace ON control_rooms(workspace_id);
```

**Go Struct**:
```go
type ControlRoom struct {
    ID             string    `db:"id" json:"id"`
    WorkspaceID    string    `db:"workspace_id" json:"workspaceId"`
    Slug           string    `db:"slug" json:"slug"`
    Name           string    `db:"name" json:"name"`
    Description    *string   `db:"description" json:"description,omitempty"`
    TagFilter      string    `db:"tag_filter" json:"tagFilter"`
    DashboardLayout []byte   `db:"dashboard_layout" json:"dashboardLayout"`
    CreatedBy      *string   `db:"created_by" json:"createdBy,omitempty"`
    CreatedAt      time.Time `db:"created_at" json:"createdAt"`
    UpdatedAt      time.Time `db:"updated_at" json:"updatedAt"`
}
```

---

### 13. Control Room Access

User access to control rooms with role-based permissions.

```sql
CREATE TABLE control_room_access (
    control_room_id UUID NOT NULL REFERENCES control_rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(32) NOT NULL CHECK (role IN ('view', 'operator', 'admin', 'billing')),
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    PRIMARY KEY (control_room_id, user_id)
);

CREATE INDEX idx_control_room_access_user ON control_room_access(user_id);
```

**Go Struct**:
```go
type ControlRoomAccess struct {
    ControlRoomID string     `db:"control_room_id" json:"controlRoomId"`
    UserID        string     `db:"user_id" json:"userId"`
    Role          string     `db:"role" json:"role"`
    GrantedBy     *string    `db:"granted_by" json:"grantedBy,omitempty"`
    GrantedAt     time.Time  `db:"granted_at" json:"grantedAt"`
    ExpiresAt     *time.Time `db:"expires_at" json:"expiresAt,omitempty"`
}
```

---

### 14. API Keys

API key management with attribution.

```sql
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    key_hash VARCHAR(64) UNIQUE NOT NULL,  -- SHA-256 hash for lookup
    key_preview VARCHAR(8) NOT NULL,        -- Last 4 chars for display (e.g., "...abcd")
    status VARCHAR(32) DEFAULT 'active' CHECK (status IN ('active', 'revoked', 'expired')),
    created_by UUID REFERENCES users(id),
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    rate_limit INTEGER,  -- requests per minute, NULL = unlimited
    allowed_models TEXT[],  -- NULL = all models
    allowed_apis TEXT[],    -- NULL = all APIs
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_api_keys_workspace ON api_keys(workspace_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_status ON api_keys(status);
```

**Go Struct**:
```go
type APIKey struct {
    ID           string     `db:"id" json:"id"`
    WorkspaceID  string     `db:"workspace_id" json:"workspaceId"`
    Name         string     `db:"name" json:"name"`
    KeyHash      string     `db:"key_hash" json:"-"`
    KeyPreview   string     `db:"key_preview" json:"keyPreview"`
    Status       string     `db:"status" json:"status"`
    CreatedBy    *string    `db:"created_by" json:"createdBy,omitempty"`
    ExpiresAt    *time.Time `db:"expires_at" json:"expiresAt,omitempty"`
    LastUsedAt   *time.Time `db:"last_used_at" json:"lastUsedAt,omitempty"`
    RateLimit    *int       `db:"rate_limit" json:"rateLimit,omitempty"`
    AllowedModels []string  `db:"allowed_models" json:"allowedModels,omitempty"`
    AllowedAPIs   []string  `db:"allowed_apis" json:"allowedAPIs,omitempty"`
    Metadata     []byte     `db:"metadata" json:"metadata"`
    CreatedAt    time.Time  `db:"created_at" json:"createdAt"`
    UpdatedAt    time.Time  `db:"updated_at" json:"updatedAt"`
}
```

---

### 15. API Key Tags

Tags assigned to API keys.

```sql
CREATE TABLE api_key_tags (
    api_key_id UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (api_key_id, tag_id)
);

CREATE INDEX idx_api_key_tags_tag ON api_key_tags(tag_id);
```

---

### 16. Quotas

Quota definitions.

```sql
CREATE TABLE quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    quota_type VARCHAR(32) NOT NULL CHECK (quota_type IN ('requests', 'tokens', 'cost')),
    period VARCHAR(32) NOT NULL CHECK (period IN ('minute', 'hour', 'day', 'month')),
    limit_value BIGINT NOT NULL,  -- max requests/tokens/cost
    scope VARCHAR(32) NOT NULL CHECK (scope IN ('workspace', 'api_key', 'control_room')),
    warning_threshold INTEGER DEFAULT 80,  -- percentage
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_quotas_workspace ON quotas(workspace_id);
CREATE INDEX idx_quotas_type ON quotas(quota_type);
```

**Go Struct**:
```go
type Quota struct {
    ID               string    `db:"id" json:"id"`
    WorkspaceID      string    `db:"workspace_id" json:"workspaceId"`
    Name             string    `db:"name" json:"name"`
    Description      *string   `db:"description" json:"description,omitempty"`
    QuotaType        string    `db:"quota_type" json:"quotaType"`
    Period           string    `db:"period" json:"period"`
    LimitValue       int64     `db:"limit_value" json:"limitValue"`
    Scope            string    `db:"scope" json:"scope"`
    WarningThreshold int       `db:"warning_threshold" json:"warningThreshold"`
    CreatedAt        time.Time `db:"created_at" json:"createdAt"`
    UpdatedAt        time.Time `db:"updated_at" json:"updatedAt"`
}
```

---

### 17. Quota Assignments

Quota assignments to resources.

```sql
CREATE TABLE quota_assignments (
    quota_id UUID NOT NULL REFERENCES quotas(id) ON DELETE CASCADE,
    resource_type VARCHAR(32) NOT NULL CHECK (resource_type IN ('api_key', 'control_room', 'workspace')),
    resource_id UUID NOT NULL,  -- references api_keys.id, control_rooms.id, or workspaces.id
    current_usage BIGINT DEFAULT 0,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    warning_sent BOOLEAN DEFAULT FALSE,
    exceeded_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (quota_id, resource_type, resource_id, period_start)
);

CREATE INDEX idx_quota_assignments_resource ON quota_assignments(resource_type, resource_id);
CREATE INDEX idx_quota_assignments_period ON quota_assignments(period_start, period_end);
```

**Go Struct**:
```go
type QuotaAssignment struct {
    QuotaID      string     `db:"quota_id" json:"quotaId"`
    ResourceType string     `db:"resource_type" json:"resourceType"`
    ResourceID   string     `db:"resource_id" json:"resourceId"`
    CurrentUsage int64      `db:"current_usage" json:"currentUsage"`
    PeriodStart  time.Time  `db:"period_start" json:"periodStart"`
    PeriodEnd    time.Time  `db:"period_end" json:"periodEnd"`
    WarningSent  bool       `db:"warning_sent" json:"warningSent"`
    ExceededAt   *time.Time `db:"exceeded_at" json:"exceededAt,omitempty"`
    UpdatedAt    time.Time  `db:"updated_at" json:"updatedAt"`
}
```

---

### 18. Usage Records

Request usage tracking.

```sql
CREATE TABLE usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    request_id VARCHAR(64) NOT NULL,
    trace_id VARCHAR(64) NOT NULL,
    api_key_id UUID REFERENCES api_keys(id),
    control_room_id UUID REFERENCES control_rooms(id),

    -- Request details
    incoming_api VARCHAR(32) NOT NULL,  -- chat, responses, embeddings, etc.
    incoming_model VARCHAR(128) NOT NULL,
    selected_model VARCHAR(128),
    provider_id UUID REFERENCES providers(id),

    -- Usage metrics
    prompt_tokens BIGINT DEFAULT 0,
    completion_tokens BIGINT DEFAULT 0,
    total_tokens BIGINT DEFAULT 0,
    cost_usd DECIMAL(12, 6),

    -- Performance
    duration_ms INTEGER NOT NULL,
    response_status VARCHAR(32) NOT NULL,  -- success, error, timeout
    error_code VARCHAR(64),
    error_message TEXT,

    -- Routing
    attempts INTEGER DEFAULT 1,
    route_log JSONB,  -- Array of attempt details

    -- Timestamps
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(request_id)
);

-- Partitioning by month recommended for high-volume usage
-- CREATE TABLE usage_records_y2026m01 PARTITION OF usage_records
--     FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

CREATE INDEX idx_usage_workspace ON usage_records(workspace_id);
CREATE INDEX idx_usage_request_id ON usage_records(request_id);
CREATE INDEX idx_usage_trace_id ON usage_records(trace_id);
CREATE INDEX idx_usage_api_key ON usage_records(api_key_id);
CREATE INDEX idx_usage_provider ON usage_records(provider_id);
CREATE INDEX idx_usage_created_at ON usage_records(created_at);
CREATE INDEX idx_usage_status ON usage_records(response_status);
CREATE INDEX idx_usage_workspace_created ON usage_records(workspace_id, created_at);

-- For time-series aggregations
CREATE INDEX idx_usage_time_series ON usage_records(workspace_id, created_at, incoming_api);
```

**Go Struct**:
```go
type UsageRecord struct {
    ID               string    `db:"id" json:"id"`
    WorkspaceID      string    `db:"workspace_id" json:"workspaceId"`
    RequestID        string    `db:"request_id" json:"requestId"`
    TraceID          string    `db:"trace_id" json:"traceId"`
    APIKeyID         *string   `db:"api_key_id" json:"apiKeyId,omitempty"`
    ControlRoomID    *string   `db:"control_room_id" json:"controlRoomId,omitempty"`
    IncomingAPI      string    `db:"incoming_api" json:"incomingApi"`
    IncomingModel    string    `db:"incoming_model" json:"incomingModel"`
    SelectedModel    *string   `db:"selected_model" json:"selectedModel,omitempty"`
    ProviderID       *string   `db:"provider_id" json:"providerId,omitempty"`
    PromptTokens     int64     `db:"prompt_tokens" json:"promptTokens"`
    CompletionTokens int64     `db:"completion_tokens" json:"completionTokens"`
    TotalTokens      int64     `db:"total_tokens" json:"totalTokens"`
    CostUSD          *float64  `db:"cost_usd" json:"costUsd,omitempty"`
    DurationMs       int       `db:"duration_ms" json:"durationMs"`
    ResponseStatus   string    `db:"response_status" json:"responseStatus"`
    ErrorCode        *string   `db:"error_code" json:"errorCode,omitempty"`
    ErrorMessage     *string   `db:"error_message" json:"errorMessage,omitempty"`
    Attempts         int       `db:"attempts" json:"attempts"`
    RouteLog         []byte    `db:"route_log" json:"routeLog"`
    StartedAt        time.Time `db:"started_at" json:"startedAt"`
    CompletedAt      *time.Time `db:"completed_at" json:"completedAt,omitempty"`
    CreatedAt        time.Time `db:"created_at" json:"createdAt"`
}
```

---

### 19. Usage Record Tags

Tags associated with usage records for aggregation.

```sql
CREATE TABLE usage_record_tags (
    usage_record_id UUID NOT NULL REFERENCES usage_records(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (usage_record_id, tag_id)
);

CREATE INDEX idx_usage_tags_tag ON usage_record_tags(tag_id);
```

---

### 20. Trace Events

Detailed trace events for debugging.

```sql
CREATE TABLE trace_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trace_id VARCHAR(64) NOT NULL,
    request_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(32) NOT NULL,  -- gateway_accept, provider_request, provider_response, error
    event_order INTEGER NOT NULL,     -- Sequence within trace

    -- Context
    provider_id UUID REFERENCES providers(id),
    api_key_id UUID REFERENCES api_keys(id),

    -- Event details
    message TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',

    -- Timing
    timestamp TIMESTAMPTZ NOT NULL,
    duration_ms INTEGER,  -- Time since trace start

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Consider time-series partitioning for high volume
CREATE INDEX idx_trace_events_trace ON trace_events(trace_id);
CREATE INDEX idx_trace_events_request ON trace_events(request_id);
CREATE INDEX idx_trace_events_timestamp ON trace_events(timestamp);
CREATE INDEX idx_trace_events_type ON trace_events(event_type);
```

**Go Struct**:
```go
type TraceEvent struct {
    ID          string     `db:"id" json:"id"`
    TraceID     string     `db:"trace_id" json:"traceId"`
    RequestID   string     `db:"request_id" json:"requestId"`
    EventType   string     `db:"event_type" json:"eventType"`
    EventOrder  int        `db:"event_order" json:"eventOrder"`
    ProviderID  *string    `db:"provider_id" json:"providerId,omitempty"`
    APIKeyID    *string    `db:"api_key_id" json:"apiKeyId,omitempty"`
    Message     string     `db:"message" json:"message"`
    Metadata    []byte     `db:"metadata" json:"metadata"`
    Timestamp   time.Time  `db:"timestamp" json:"timestamp"`
    DurationMs  *int       `db:"duration_ms" json:"durationMs,omitempty"`
    CreatedAt   time.Time  `db:"created_at" json:"createdAt"`
}
```

---

## Migration Strategy

### Phase 1: Foundation Tables

```sql
-- 001_create_workspaces.sql
-- 002_create_users.sql
-- 003_create_roles.sql
-- 004_create_permissions.sql
-- 005_create_user_roles.sql
-- 006_create_role_permissions.sql
-- 007_create_tags.sql
```

### Phase 2: Provider Tables

```sql
-- 008_create_providers.sql
-- 009_create_provider_tags.sql
-- 010_create_provider_health.sql
-- 011_create_circuit_breaker_states.sql
```

### Phase 3: Access Control

```sql
-- 012_create_control_rooms.sql
-- 013_create_control_room_access.sql
-- 014_create_api_keys.sql
-- 015_create_api_key_tags.sql
```

### Phase 4: Usage Tracking

```sql
-- 016_create_quotas.sql
-- 017_create_quota_assignments.sql
-- 018_create_usage_records.sql
-- 019_create_usage_record_tags.sql
-- 020_create_trace_events.sql
```

---

## Key Design Decisions

### 1. UUID Primary Keys
- Enables distributed ID generation
- Safe for multi-region deployments
- No collision risk

### 2. JSONB for Flexible Metadata
- Provider configs can vary by type
- Dashboard layouts are dynamic
- Future extensibility

### 3. Soft Deletion Pattern
- Use `status` columns instead of DELETE
- Maintain audit trail
- Enable data recovery

### 4. Partitioning Strategy
- Partition `usage_records` and `trace_events` by month
- Enables efficient time-range queries
- Simplifies data archival

### 5. Index Strategy
- Cover common query patterns
- Foreign key indexes for JOINs
- Composite indexes for time-series queries

### 6. Encrypted Secrets
- API keys stored encrypted
- Application handles encryption/decryption
- Database never sees plaintext

---

## Query Patterns

### Usage Aggregation by Control Room

```sql
SELECT
    cr.name AS control_room,
    DATE_TRUNC('day', ur.created_at) AS day,
    COUNT(*) AS requests,
    SUM(ur.total_tokens) AS tokens,
    SUM(ur.cost_usd) AS cost
FROM usage_records ur
JOIN control_rooms cr ON ur.control_room_id = cr.id
WHERE ur.created_at >= NOW() - INTERVAL '30 days'
GROUP BY cr.name, DATE_TRUNC('day', ur.created_at)
ORDER BY day DESC;
```

### Provider Health Dashboard

```sql
SELECT
    p.name AS provider,
    p.provider_type,
    ph.healthy,
    ph.latency_ms,
    ph.consecutive_failures,
    ph.last_check_at
FROM providers p
LEFT JOIN provider_health ph ON p.id = ph.provider_id
WHERE p.workspace_id = $1
ORDER BY ph.healthy ASC, p.name;
```

### Quota Status Check

```sql
SELECT
    q.name,
    q.quota_type,
    q.limit_value,
    qa.current_usage,
    (qa.current_usage::float / q.limit_value * 100) AS percent_used,
    qa.warning_sent,
    qa.exceeded_at
FROM quotas q
JOIN quota_assignments qa ON q.id = qa.quota_id
WHERE qa.resource_type = 'api_key'
  AND qa.resource_id = $1
  AND NOW() BETWEEN qa.period_start AND qa.period_end;
```

---

## Related Documents

- `/mnt/ollama/git/RADAPI01/docs/plans/2026-02-16-control-room-tagging-design.md`
- `/mnt/ollama/git/RADAPI01/internal/models/models.go`
- `/mnt/ollama/git/RADAPI01/internal/usage/usage.go`
- `/mnt/ollama/git/RADAPI01/internal/trace/trace.go`
