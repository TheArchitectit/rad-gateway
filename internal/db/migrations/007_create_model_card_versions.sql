-- Migration 007: Create Model Card Versions table
-- Audit history for A2A Model Cards with full version tracking

-- Model Card Versions table (audit history)
CREATE TABLE IF NOT EXISTS model_card_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_card_id UUID NOT NULL,
    workspace_id TEXT NOT NULL,
    user_id TEXT,
    version INTEGER NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT,
    card JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'active',
    change_reason TEXT,
    created_by TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (model_card_id) REFERENCES a2a_model_cards(id) ON DELETE CASCADE,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE(model_card_id, version)
);

-- Index for model card version lookups
CREATE INDEX IF NOT EXISTS idx_model_card_versions_card ON model_card_versions(model_card_id);

-- Index for workspace lookups
CREATE INDEX IF NOT EXISTS idx_model_card_versions_workspace ON model_card_versions(workspace_id);

-- Index for version ordering
CREATE INDEX IF NOT EXISTS idx_model_card_versions_created ON model_card_versions(created_at DESC);

-- Composite index for common query patterns
CREATE INDEX IF NOT EXISTS idx_model_card_versions_card_version ON model_card_versions(model_card_id, version);

-- Schema version tracking
INSERT INTO schema_migrations (version) VALUES (7) ON CONFLICT (version) DO NOTHING;
