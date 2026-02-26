-- Migration: Create audit log table
-- Description: Security audit logging for authentication, authorization, and system events

-- Audit log table
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    type TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'info',

    -- Actor (who performed the action)
    actor_type TEXT NOT NULL DEFAULT 'anonymous',
    actor_id TEXT,
    actor_name TEXT,
    actor_role TEXT,
    actor_ip INET,
    user_agent TEXT,

    -- Resource (what was accessed)
    resource_type TEXT,
    resource_id TEXT,
    resource_name TEXT,
    workspace_id TEXT,

    -- Action details
    action TEXT NOT NULL,
    result TEXT NOT NULL, -- success, failure, denied
    details JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',

    -- Request context
    request_method TEXT,
    request_path TEXT,
    request_query TEXT,
    trace_id TEXT,
    request_id TEXT,

    -- Indexes for common queries
    CONSTRAINT chk_severity CHECK (severity IN ('debug', 'info', 'warning', 'error', 'critical')),
    CONSTRAINT chk_result CHECK (result IN ('success', 'failure', 'denied'))
);

-- Indexes for performance
CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp DESC);
CREATE INDEX idx_audit_log_type ON audit_log(type);
CREATE INDEX idx_audit_log_severity ON audit_log(severity);
CREATE INDEX idx_audit_log_actor_id ON audit_log(actor_id);
CREATE INDEX idx_audit_log_resource_id ON audit_log(resource_id);
CREATE INDEX idx_audit_log_workspace_id ON audit_log(workspace_id);
CREATE INDEX idx_audit_log_trace_id ON audit_log(trace_id);

-- Composite indexes for common query patterns
CREATE INDEX idx_audit_log_actor_time ON audit_log(actor_id, timestamp DESC);
CREATE INDEX idx_audit_log_resource_time ON audit_log(resource_id, timestamp DESC);
CREATE INDEX idx_audit_log_type_severity ON audit_log(type, severity, timestamp DESC);

-- GIN index for JSONB queries
CREATE INDEX idx_audit_log_details ON audit_log USING GIN (details jsonb_path_ops);

-- Table comment
COMMENT ON TABLE audit_log IS 'Security audit log for tracking authentication, authorization, and system events';

-- Partitioning setup for high-volume logging (optional, can be enabled later)
-- This table can be partitioned by timestamp for better performance
-- CREATE TABLE audit_log_2026_02 PARTITION OF audit_log
--     FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

-- Retention policy function
CREATE OR REPLACE FUNCTION purge_old_audit_log(retention_days INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM audit_log
    WHERE timestamp < NOW() - INTERVAL '1 day' * retention_days;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- View for security events (failed auth, suspicious activity)
CREATE OR REPLACE VIEW security_events AS
SELECT *
FROM audit_log
WHERE severity IN ('error', 'critical')
   OR type IN (
       'auth:failure',
       'authz:access_denied',
       'security:suspicious',
       'security:ip_blocked',
       'ratelimit:exceeded'
   )
ORDER BY timestamp DESC;

-- View for API key activity
CREATE OR REPLACE VIEW api_key_activity AS
SELECT *
FROM audit_log
WHERE actor_type = 'api_key'
   OR type LIKE 'apikey:%'
ORDER BY timestamp DESC;
