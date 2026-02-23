-- Migration: Create control rooms tables
-- Date: 2026-02-17

CREATE TABLE IF NOT EXISTS control_rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    slug VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    tag_filter TEXT NOT NULL,
    dashboard_layout JSONB DEFAULT '{}',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_control_rooms_workspace ON control_rooms(workspace_id);

CREATE TABLE IF NOT EXISTS control_room_access (
    control_room_id UUID NOT NULL REFERENCES control_rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(32) NOT NULL CHECK (role IN ('view', 'operator', 'admin', 'billing')),
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    PRIMARY KEY (control_room_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_control_room_access_user ON control_room_access(user_id);
