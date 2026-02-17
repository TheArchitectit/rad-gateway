-- RAD Gateway Database Indexes
-- Optimized for PostgreSQL production environment
-- Apply after schema migrations

-- ============================================================================
-- INDEX CREATION GUIDELINES
-- ============================================================================
-- 1. Create indexes on frequently queried columns (WHERE, JOIN, ORDER BY)
-- 2. Use composite indexes for multi-column queries (column order matters!)
-- 3. Consider partial indexes for filtered queries
-- 4. Covering indexes can eliminate table lookups for specific queries
-- 5. Balance: Each index adds write overhead but speeds up reads
-- ============================================================================

-- ============================================================================
-- WORKSPACES TABLE
-- ============================================================================
-- Primary lookup by slug (already indexed, ensure uniqueness)
CREATE UNIQUE INDEX IF NOT EXISTS idx_workspaces_slug_unique ON workspaces(slug);

-- Status filtering for admin dashboards
CREATE INDEX IF NOT EXISTS idx_workspaces_status_created ON workspaces(status, created_at DESC);

-- ============================================================================
-- USERS TABLE
-- ============================================================================
-- Email lookup (case-insensitive for user login)
CREATE INDEX IF NOT EXISTS idx_users_email_lower ON users(LOWER(email));

-- Workspace membership with status filter (common query pattern)
CREATE INDEX IF NOT EXISTS idx_users_workspace_status ON users(workspace_id, status, created_at DESC);

-- Last login tracking for user activity reports
CREATE INDEX IF NOT EXISTS idx_users_last_login ON users(last_login_at DESC NULLS LAST)
    WHERE last_login_at IS NOT NULL;

-- ============================================================================
-- ROLES TABLE
-- ============================================================================
-- System roles lookup (frequent auth check)
CREATE INDEX IF NOT EXISTS idx_roles_system ON roles(is_system)
    WHERE is_system = TRUE;

-- Workspace roles with ordering
CREATE INDEX IF NOT EXISTS idx_roles_workspace_created ON roles(workspace_id, created_at DESC);

-- ============================================================================
-- USER_ROLES TABLE
-- ============================================================================
-- Optimized for user role lookups (primary query pattern)
-- Note: Primary key already covers (user_id, role_id)

-- Role membership lookup
CREATE INDEX IF NOT EXISTS idx_user_roles_role_user ON user_roles(role_id, user_id);

-- Expired role cleanup query optimization
CREATE INDEX IF NOT EXISTS idx_user_roles_expires ON user_roles(expires_at)
    WHERE expires_at IS NOT NULL;

-- ============================================================================
-- PERMISSIONS TABLE
-- ============================================================================
-- Resource type + action lookup (RBAC checks)
CREATE INDEX IF NOT EXISTS idx_permissions_resource_action ON permissions(resource_type, action);

-- ============================================================================
-- ROLE_PERMISSIONS TABLE
-- ============================================================================
-- Permission lookup (reverse query)
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission ON role_permissions(permission_id, role_id);

-- ============================================================================
-- TAGS TABLE
-- ============================================================================
-- Category lookup within workspace (control room filtering)
CREATE INDEX IF NOT EXISTS idx_tags_workspace_category ON tags(workspace_id, category, value);

-- ============================================================================
-- PROVIDERS TABLE
-- ============================================================================
-- Active providers by workspace (common query)
CREATE INDEX IF NOT EXISTS idx_providers_workspace_status ON providers(workspace_id, status, priority DESC, weight DESC);

-- Provider type filtering
CREATE INDEX IF NOT EXISTS idx_providers_type_status ON providers(provider_type, status);

-- Priority/weight ordering for load balancing
CREATE INDEX IF NOT EXISTS idx_providers_priority_weight ON providers(priority DESC, weight DESC)
    WHERE status = 'active';

-- ============================================================================
-- PROVIDER_TAGS TABLE
-- ============================================================================
-- Reverse lookup: find providers by tag
CREATE INDEX IF NOT EXISTS idx_provider_tags_tag_provider ON provider_tags(tag_id, provider_id);

-- ============================================================================
-- PROVIDER_HEALTH TABLE
-- ============================================================================
-- Unhealthy providers query (monitoring)
CREATE INDEX IF NOT EXISTS idx_provider_health_unhealthy ON provider_health(healthy, last_check_at DESC)
    WHERE healthy = FALSE;

-- Latency tracking for slow providers
CREATE INDEX IF NOT EXISTS idx_provider_health_latency ON provider_health(latency_ms DESC NULLS LAST)
    WHERE latency_ms > 1000;

-- ============================================================================
-- CIRCUIT_BREAKER_STATES TABLE
-- ============================================================================
-- Open circuits query
CREATE INDEX IF NOT EXISTS idx_circuit_breaker_open ON circuit_breaker_states(state, opened_at DESC)
    WHERE state = 'open';

-- Half-open circuits query
CREATE INDEX IF NOT EXISTS idx_circuit_breaker_half_open ON circuit_breaker_states(state, updated_at DESC)
    WHERE state = 'half_open';

-- ============================================================================
-- CONTROL_ROOMS TABLE
-- ============================================================================
-- Workspace control rooms with ordering
CREATE INDEX IF NOT EXISTS idx_control_rooms_workspace_created ON control_rooms(workspace_id, created_at DESC);

-- Created by user lookup
CREATE INDEX IF NOT EXISTS idx_control_rooms_creator ON control_rooms(created_by)
    WHERE created_by IS NOT NULL;

-- ============================================================================
-- CONTROL_ROOM_ACCESS TABLE
-- ============================================================================
-- User access lookup (primary query pattern)
-- Note: Primary key already covers (control_room_id, user_id)

-- Reverse lookup: user's control rooms
CREATE INDEX IF NOT EXISTS idx_control_room_access_user_room ON control_room_access(user_id, control_room_id);

-- Expired access cleanup
CREATE INDEX IF NOT EXISTS idx_control_room_access_expires ON control_room_access(expires_at)
    WHERE expires_at IS NOT NULL;

-- ============================================================================
-- API_KEYS TABLE
-- ============================================================================
-- Hash lookup (authentication - most critical index)
CREATE INDEX IF NOT EXISTS idx_api_keys_hash_status ON api_keys(key_hash, status);

-- Active keys by workspace
CREATE INDEX IF NOT EXISTS idx_api_keys_workspace_status ON api_keys(workspace_id, status, created_at DESC);

-- Expired keys cleanup
CREATE INDEX IF NOT EXISTS idx_api_keys_expired ON api_keys(expires_at, status)
    WHERE status = 'active' AND expires_at IS NOT NULL;

-- Last used tracking
CREATE INDEX IF NOT EXISTS idx_api_keys_last_used ON api_keys(last_used_at DESC NULLS LAST);

-- ============================================================================
-- API_KEY_TAGS TABLE
-- ============================================================================
-- Reverse lookup: find keys by tag
CREATE INDEX IF NOT EXISTS idx_api_key_tags_tag_key ON api_key_tags(tag_id, api_key_id);

-- ============================================================================
-- QUOTAS TABLE
-- ============================================================================
-- Quotas by type and scope
CREATE INDEX IF NOT EXISTS idx_quotas_workspace_type ON quotas(workspace_id, quota_type, scope);

-- ============================================================================
-- QUOTA_ASSIGNMENTS TABLE
-- ============================================================================
-- Resource assignment lookup (primary query)
-- Note: Primary key already covers (quota_id, resource_type, resource_id, period_start)

-- Current period lookup
CREATE INDEX IF NOT EXISTS idx_quota_assignments_resource_period ON quota_assignments(resource_type, resource_id, period_start DESC, period_end);

-- Warning/exceeded queries
CREATE INDEX IF NOT EXISTS idx_quota_assignments_warning ON quota_assignments(warning_sent, exceeded_at)
    WHERE warning_sent = TRUE OR exceeded_at IS NOT NULL;

-- ============================================================================
-- USAGE_RECORDS TABLE (HIGH VOLUME - most critical for performance)
-- ============================================================================

-- Time-series queries (most common pattern)
CREATE INDEX IF NOT EXISTS idx_usage_workspace_time_api ON usage_records(workspace_id, created_at DESC, incoming_api);

-- API key usage over time (for billing/quotas)
CREATE INDEX IF NOT EXISTS idx_usage_api_key_time ON usage_records(api_key_id, created_at DESC)
    WHERE api_key_id IS NOT NULL;

-- Provider performance tracking
CREATE INDEX IF NOT EXISTS idx_usage_provider_time ON usage_records(provider_id, created_at DESC, duration_ms)
    WHERE provider_id IS NOT NULL;

-- Control room usage
CREATE INDEX IF NOT EXISTS idx_usage_control_room_time ON usage_records(control_room_id, created_at DESC)
    WHERE control_room_id IS NOT NULL;

-- Error tracking
CREATE INDEX IF NOT EXISTS idx_usage_errors ON usage_records(response_status, created_at DESC, error_code)
    WHERE response_status != 'success';

-- Model usage analytics
CREATE INDEX IF NOT EXISTS idx_usage_model_time ON usage_records(incoming_model, created_at DESC);

-- Cost analysis
CREATE INDEX IF NOT EXISTS idx_usage_cost_time ON usage_records(workspace_id, cost_usd DESC, created_at DESC)
    WHERE cost_usd IS NOT NULL AND cost_usd > 0;

-- Partial index: high token usage (for analytics)
CREATE INDEX IF NOT EXISTS idx_usage_high_tokens ON usage_records(total_tokens DESC, created_at DESC)
    WHERE total_tokens > 10000;

-- Duration analysis for slow requests
CREATE INDEX IF NOT EXISTS idx_usage_slow_requests ON usage_records(duration_ms DESC, created_at DESC)
    WHERE duration_ms > 30000;

-- ============================================================================
-- USAGE_RECORD_TAGS TABLE
-- ============================================================================
-- Usage by tag (analytics)
CREATE INDEX IF NOT EXISTS idx_usage_tags_usage ON usage_record_tags(tag_id, usage_record_id);

-- ============================================================================
-- TRACE_EVENTS TABLE
-- ============================================================================
-- Trace event ordering
CREATE INDEX IF NOT EXISTS idx_trace_events_trace_order ON trace_events(trace_id, event_order);

-- Request event ordering
CREATE INDEX IF NOT EXISTS idx_trace_events_request_order ON trace_events(request_id, event_order);

-- Event type analytics
CREATE INDEX IF NOT EXISTS idx_trace_events_type_time ON trace_events(event_type, timestamp DESC);

-- Provider event tracking
CREATE INDEX IF NOT EXISTS idx_trace_events_provider ON trace_events(provider_id, timestamp DESC)
    WHERE provider_id IS NOT NULL;

-- Recent events (for real-time monitoring)
CREATE INDEX IF NOT EXISTS idx_trace_events_recent ON trace_events(timestamp DESC)
    WHERE timestamp > NOW() - INTERVAL '24 hours';

-- ============================================================================
-- MAINTENANCE INDEXES (for cleanup operations)
-- ============================================================================

-- Old usage records cleanup
CREATE INDEX IF NOT EXISTS idx_usage_created_old ON usage_records(created_at)
    WHERE created_at < NOW() - INTERVAL '90 days';

-- Old trace events cleanup
CREATE INDEX IF NOT EXISTS idx_trace_events_old ON trace_events(created_at)
    WHERE created_at < NOW() - INTERVAL '7 days';

-- ============================================================================
-- INDEX MAINTENANCE
-- ============================================================================

-- Analyze tables after index creation
ANALYZE workspaces;
ANALYZE users;
ANALYZE roles;
ANALYZE user_roles;
ANALYZE permissions;
ANALYZE role_permissions;
ANALYZE tags;
ANALYZE providers;
ANALYZE provider_tags;
ANALYZE provider_health;
ANALYZE circuit_breaker_states;
ANALYZE control_rooms;
ANALYZE control_room_access;
ANALYZE api_keys;
ANALYZE api_key_tags;
ANALYZE quotas;
ANALYZE quota_assignments;
ANALYZE usage_records;
ANALYZE usage_record_tags;
ANALYZE trace_events;
