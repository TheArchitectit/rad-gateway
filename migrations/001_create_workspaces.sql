-- Migration: Create workspaces table
-- Date: 2026-02-17

CREATE TABLE IF NOT EXISTS workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(32) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'archived')),
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workspaces_status ON workspaces(status);
CREATE INDEX IF NOT EXISTS idx_workspaces_slug ON workspaces(slug);

-- Create default workspace
INSERT INTO workspaces (slug, name, description, status)
VALUES ('default', 'Default Workspace', 'System default workspace', 'active')
ON CONFLICT (slug) DO NOTHING;
