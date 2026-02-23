-- Migration: Create API keys tables
-- Date: 2026-02-17

CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    key_hash VARCHAR(64) UNIQUE NOT NULL,
    key_preview VARCHAR(8) NOT NULL,
    status VARCHAR(32) DEFAULT 'active' CHECK (status IN ('active', 'revoked', 'expired')),
    created_by UUID REFERENCES users(id),
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    rate_limit INTEGER,
    allowed_models TEXT[],
    allowed_apis TEXT[],
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_api_keys_workspace ON api_keys(workspace_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);

CREATE TABLE IF NOT EXISTS api_key_tags (
    api_key_id UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (api_key_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_api_key_tags_tag ON api_key_tags(tag_id);
