-- Migration: Create providers tables
-- Date: 2026-02-17

CREATE TABLE IF NOT EXISTS providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    slug VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    provider_type VARCHAR(32) NOT NULL,
    base_url VARCHAR(512) NOT NULL,
    api_key_encrypted TEXT,
    config JSONB DEFAULT '{}',
    status VARCHAR(32) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'degraded')),
    priority INTEGER DEFAULT 100,
    weight INTEGER DEFAULT 100,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_providers_workspace ON providers(workspace_id);
CREATE INDEX IF NOT EXISTS idx_providers_status ON providers(status);
CREATE INDEX IF NOT EXISTS idx_providers_type ON providers(provider_type);

CREATE TABLE IF NOT EXISTS provider_tags (
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (provider_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_provider_tags_tag ON provider_tags(tag_id);

CREATE TABLE IF NOT EXISTS provider_health (
    provider_id UUID PRIMARY KEY REFERENCES providers(id) ON DELETE CASCADE,
    healthy BOOLEAN NOT NULL DEFAULT TRUE,
    last_check_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_success_at TIMESTAMPTZ,
    consecutive_failures INTEGER DEFAULT 0,
    latency_ms INTEGER,
    error_message TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_provider_health_healthy ON provider_health(healthy);
CREATE INDEX IF NOT EXISTS idx_provider_health_last_check ON provider_health(last_check_at);

CREATE TABLE IF NOT EXISTS circuit_breaker_states (
    provider_id UUID PRIMARY KEY REFERENCES providers(id) ON DELETE CASCADE,
    state VARCHAR(32) NOT NULL CHECK (state IN ('closed', 'open', 'half_open')),
    failures INTEGER DEFAULT 0,
    successes INTEGER DEFAULT 0,
    last_failure_at TIMESTAMPTZ,
    half_open_requests INTEGER DEFAULT 0,
    opened_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
