-- Migration: Create tags tables
-- Date: 2026-02-17

CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    category VARCHAR(64) NOT NULL,
    value VARCHAR(128) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, category, value)
);

CREATE INDEX IF NOT EXISTS idx_tags_workspace ON tags(workspace_id);
CREATE INDEX IF NOT EXISTS idx_tags_category ON tags(category);

-- Insert common tags
INSERT INTO tags (workspace_id, category, value, description)
SELECT w.id, 'env', 'production', 'Production environment'
FROM workspaces w WHERE w.slug = 'default'
ON CONFLICT (workspace_id, category, value) DO NOTHING;

INSERT INTO tags (workspace_id, category, value, description)
SELECT w.id, 'env', 'staging', 'Staging environment'
FROM workspaces w WHERE w.slug = 'default'
ON CONFLICT (workspace_id, category, value) DO NOTHING;

INSERT INTO tags (workspace_id, category, value, description)
SELECT w.id, 'env', 'development', 'Development environment'
FROM workspaces w WHERE w.slug = 'default'
ON CONFLICT (workspace_id, category, value) DO NOTHING;
