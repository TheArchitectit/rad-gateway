-- Migration 006: Create A2A Model Cards table
-- Stores A2A (Agent-to-Agent) Model Cards with JSONB for flexible schema

-- A2A Model Cards table
CREATE TABLE IF NOT EXISTS a2a_model_cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id TEXT NOT NULL,
    user_id TEXT,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT,
    card JSONB NOT NULL DEFAULT '{}',
    version INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Unique constraint on workspace + slug
CREATE UNIQUE INDEX IF NOT EXISTS idx_a2a_model_cards_workspace_slug ON a2a_model_cards(workspace_id, slug);

-- Index for workspace lookups
CREATE INDEX IF NOT EXISTS idx_a2a_model_cards_workspace ON a2a_model_cards(workspace_id);

-- Index for user lookups
CREATE INDEX IF NOT EXISTS idx_a2a_model_cards_user ON a2a_model_cards(user_id);

-- Index for status filtering
CREATE INDEX IF NOT EXISTS idx_a2a_model_cards_status ON a2a_model_cards(status);

-- Index for name search
CREATE INDEX IF NOT EXISTS idx_a2a_model_cards_name ON a2a_model_cards USING gin(to_tsvector('english', name));

-- Schema version tracking
INSERT INTO schema_migrations (version) VALUES (6) ON CONFLICT (version) DO NOTHING;
