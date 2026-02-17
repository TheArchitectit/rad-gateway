-- Migration: Create quotas tables
-- Date: 2026-02-17

CREATE TABLE IF NOT EXISTS quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    quota_type VARCHAR(32) NOT NULL CHECK (quota_type IN ('requests', 'tokens', 'cost')),
    period VARCHAR(32) NOT NULL CHECK (period IN ('minute', 'hour', 'day', 'month')),
    limit_value BIGINT NOT NULL,
    scope VARCHAR(32) NOT NULL CHECK (scope IN ('workspace', 'api_key', 'control_room')),
    warning_threshold INTEGER DEFAULT 80,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_quotas_workspace ON quotas(workspace_id);
CREATE INDEX IF NOT EXISTS idx_quotas_type ON quotas(quota_type);

CREATE TABLE IF NOT EXISTS quota_assignments (
    quota_id UUID NOT NULL REFERENCES quotas(id) ON DELETE CASCADE,
    resource_type VARCHAR(32) NOT NULL CHECK (resource_type IN ('api_key', 'control_room', 'workspace')),
    resource_id UUID NOT NULL,
    current_usage BIGINT DEFAULT 0,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    warning_sent BOOLEAN DEFAULT FALSE,
    exceeded_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (quota_id, resource_type, resource_id, period_start)
);

CREATE INDEX IF NOT EXISTS idx_quota_assignments_resource ON quota_assignments(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_quota_assignments_period ON quota_assignments(period_start, period_end);
