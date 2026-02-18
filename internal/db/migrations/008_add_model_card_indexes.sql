-- Migration 008: Add GIN indexes for JSONB queries on A2A Model Cards
-- Enables efficient JSONB querying for A2A model card capabilities and skills

-- GIN index on entire card JSONB column for general JSON queries
CREATE INDEX IF NOT EXISTS idx_a2a_model_cards_card_gin ON a2a_model_cards USING gin(card);

-- GIN index on specific card properties for common queries
-- Index for capabilities searches
CREATE INDEX IF NOT EXISTS idx_a2a_model_cards_capabilities ON a2a_model_cards USING gin((card->'capabilities'));

-- Index for skills array searches
CREATE INDEX IF NOT EXISTS idx_a2a_model_cards_skills ON a2a_model_cards USING gin((card->'skills'));

-- Index for URL searches (common A2A lookup pattern)
CREATE INDEX IF NOT EXISTS idx_a2a_model_cards_url ON a2a_model_cards USING gin((card->>'url'));

-- GIN index on model card versions card data
CREATE INDEX IF NOT EXISTS idx_model_card_versions_card_gin ON model_card_versions USING gin(card);

-- Partial indexes for active cards only (common query pattern)
CREATE INDEX IF NOT EXISTS idx_a2a_model_cards_active ON a2a_model_cards(workspace_id, slug) WHERE status = 'active';

-- Index for card name searches within JSONB (A2A protocol field)
CREATE INDEX IF NOT EXISTS idx_a2a_model_cards_card_name ON a2a_model_cards USING gin((card->>'name'));

-- Schema version tracking
INSERT INTO schema_migrations (version) VALUES (8) ON CONFLICT (version) DO NOTHING;
