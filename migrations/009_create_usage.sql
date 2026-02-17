-- Migration: Create usage tracking tables
-- Date: 2026-02-17

CREATE TABLE IF NOT EXISTS usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    request_id VARCHAR(64) NOT NULL UNIQUE,
    trace_id VARCHAR(64) NOT NULL,
    api_key_id UUID REFERENCES api_keys(id),
    control_room_id UUID REFERENCES control_rooms(id),
    incoming_api VARCHAR(32) NOT NULL,
    incoming_model VARCHAR(128) NOT NULL,
    selected_model VARCHAR(128),
    provider_id UUID REFERENCES providers(id),
    prompt_tokens BIGINT DEFAULT 0,
    completion_tokens BIGINT DEFAULT 0,
    total_tokens BIGINT DEFAULT 0,
    cost_usd DECIMAL(12, 6),
    duration_ms INTEGER NOT NULL,
    response_status VARCHAR(32) NOT NULL,
    error_code VARCHAR(64),
    error_message TEXT,
    attempts INTEGER DEFAULT 1,
    route_log JSONB,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_usage_workspace ON usage_records(workspace_id);
CREATE INDEX IF NOT EXISTS idx_usage_request_id ON usage_records(request_id);
CREATE INDEX IF NOT EXISTS idx_usage_trace_id ON usage_records(trace_id);
CREATE INDEX IF NOT EXISTS idx_usage_api_key ON usage_records(api_key_id);
CREATE INDEX IF NOT EXISTS idx_usage_provider ON usage_records(provider_id);
CREATE INDEX IF NOT EXISTS idx_usage_created_at ON usage_records(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_status ON usage_records(response_status);
CREATE INDEX IF NOT EXISTS idx_usage_workspace_created ON usage_records(workspace_id, created_at);
CREATE INDEX IF NOT EXISTS idx_usage_time_series ON usage_records(workspace_id, created_at, incoming_api);

CREATE TABLE IF NOT EXISTS usage_record_tags (
    usage_record_id UUID NOT NULL REFERENCES usage_records(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (usage_record_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_usage_tags_tag ON usage_record_tags(tag_id);

CREATE TABLE IF NOT EXISTS trace_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trace_id VARCHAR(64) NOT NULL,
    request_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(32) NOT NULL,
    event_order INTEGER NOT NULL,
    provider_id UUID REFERENCES providers(id),
    api_key_id UUID REFERENCES api_keys(id),
    message TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    timestamp TIMESTAMPTZ NOT NULL,
    duration_ms INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trace_events_trace ON trace_events(trace_id);
CREATE INDEX IF NOT EXISTS idx_trace_events_request ON trace_events(request_id);
CREATE INDEX IF NOT EXISTS idx_trace_events_timestamp ON trace_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_trace_events_type ON trace_events(event_type);
