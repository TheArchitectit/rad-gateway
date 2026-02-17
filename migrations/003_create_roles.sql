-- Migration: Create roles and permissions tables
-- Date: 2026-02-17

CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(64) NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, name)
);

CREATE INDEX IF NOT EXISTS idx_roles_workspace ON roles(workspace_id);

CREATE TABLE IF NOT EXISTS permissions (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    resource_type VARCHAR(32) NOT NULL,
    action VARCHAR(32) NOT NULL
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(role_id);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id VARCHAR(64) NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Insert system permissions
INSERT INTO permissions (id, name, description, resource_type, action) VALUES
    ('provider:read', 'View Providers', 'View provider details', 'provider', 'read'),
    ('provider:create', 'Create Providers', 'Add new providers', 'provider', 'create'),
    ('provider:update', 'Update Providers', 'Modify provider settings', 'provider', 'update'),
    ('provider:delete', 'Delete Providers', 'Remove providers', 'provider', 'delete'),
    ('route:read', 'View Routes', 'View routing configuration', 'route', 'read'),
    ('route:create', 'Create Routes', 'Create new routes', 'route', 'create'),
    ('route:update', 'Update Routes', 'Modify routes', 'route', 'update'),
    ('route:delete', 'Delete Routes', 'Remove routes', 'route', 'delete'),
    ('api_key:read', 'View API Keys', 'View API key details', 'api_key', 'read'),
    ('api_key:create', 'Create API Keys', 'Create new API keys', 'api_key', 'create'),
    ('api_key:revoke', 'Revoke API Keys', 'Revoke API keys', 'api_key', 'delete'),
    ('usage:read', 'View Usage', 'View usage data', 'usage', 'read'),
    ('cost:read', 'View Costs', 'View cost reports', 'cost', 'read'),
    ('control_room:read', 'View Control Rooms', 'View control rooms', 'control_room', 'read'),
    ('control_room:create', 'Create Control Rooms', 'Create control rooms', 'control_room', 'create'),
    ('control_room:update', 'Update Control Rooms', 'Modify control rooms', 'control_room', 'update'),
    ('control_room:delete', 'Delete Control Rooms', 'Delete control rooms', 'control_room', 'delete'),
    ('quota:read', 'View Quotas', 'View quota settings', 'quota', 'read'),
    ('quota:manage', 'Manage Quotas', 'Set quotas', 'quota', 'update')
ON CONFLICT (id) DO NOTHING;

-- Insert system roles (global, no workspace)
INSERT INTO roles (id, name, description, is_system) VALUES
    ('00000000-0000-0000-0000-000000000001', 'admin', 'Full access to all resources', TRUE),
    ('00000000-0000-0000-0000-000000000002', 'operator', 'Can manage routes and view dashboards', TRUE),
    ('00000000-0000-0000-0000-000000000003', 'viewer', 'Read-only access', TRUE),
    ('00000000-0000-0000-0000-000000000004', 'billing', 'Access to usage and cost reports', TRUE)
ON CONFLICT (name) DO NOTHING;

-- Assign permissions to system roles
-- Admin gets all permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000001', id FROM permissions
ON CONFLICT DO NOTHING;

-- Operator permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-0000-0000-000000000002', 'provider:read'),
    ('00000000-0000-0000-0000-000000000002', 'route:read'),
    ('00000000-0000-0000-0000-000000000002', 'route:update'),
    ('00000000-0000-0000-0000-000000000002', 'usage:read'),
    ('00000000-0000-0000-0000-000000000002', 'control_room:read')
ON CONFLICT DO NOTHING;

-- Viewer permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-0000-0000-000000000003', 'provider:read'),
    ('00000000-0000-0000-0000-000000000003', 'route:read'),
    ('00000000-0000-0000-0000-000000000003', 'usage:read'),
    ('00000000-0000-0000-0000-000000000003', 'control_room:read')
ON CONFLICT DO NOTHING;

-- Billing permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-0000-0000-000000000004', 'usage:read'),
    ('00000000-0000-0000-0000-000000000004', 'cost:read')
ON CONFLICT DO NOTHING;
