// Package audit provides security audit logging functionality.
package audit

import (
	"time"
)

// EventType represents the type of audit event.
type EventType string

const (
	// Authentication events
	EventAuthSuccess     EventType = "auth:success"
	EventAuthFailure     EventType = "auth:failure"
	EventTokenRefresh    EventType = "auth:token_refresh"
	EventTokenRevoked    EventType = "auth:token_revoked"
	EventLogout          EventType = "auth:logout"

	// Authorization events
	EventAccessDenied    EventType = "authz:access_denied"
	EventPermissionCheck EventType = "authz:permission_check"
	EventRoleEscalation  EventType = "authz:role_escalation"

	// API Key events
	EventAPIKeyCreated   EventType = "apikey:created"
	EventAPIKeyRevoked   EventType = "apikey:revoked"
	EventAPIKeyUsed      EventType = "apikey:used"
	EventAPIKeyRotated   EventType = "apikey:rotated"

	// Rate limiting events
	EventRateLimitExceeded EventType = "ratelimit:exceeded"
	EventRateLimitWarning  EventType = "ratelimit:warning"

	// Security events
	EventSuspiciousActivity EventType = "security:suspicious"
	EventIPBlocked          EventType = "security:ip_blocked"
	EventConfigChange       EventType = "security:config_change"

	// Admin events
	EventAdminAction     EventType = "admin:action"
	EventUserCreated     EventType = "admin:user_created"
	EventUserModified    EventType = "admin:user_modified"
	EventUserDeleted     EventType = "admin:user_deleted"
)

// EventSeverity represents the severity of an audit event.
type EventSeverity string

const (
	SeverityDebug   EventSeverity = "debug"
	SeverityInfo    EventSeverity = "info"
	SeverityWarning EventSeverity = "warning"
	SeverityError   EventSeverity = "error"
	SeverityCritical EventSeverity = "critical"
)

// Event represents a security audit event.
type Event struct {
	ID          string                 `json:"id" db:"id"`
	Timestamp   time.Time              `json:"timestamp" db:"timestamp"`
	Type        EventType              `json:"type" db:"type"`
	Severity    EventSeverity          `json:"severity" db:"severity"`
	Actor       Actor                  `json:"actor" db:"-"`
	Resource    Resource               `json:"resource" db:"-"`
	Action      string                 `json:"action" db:"action"`
	Result      string                 `json:"result" db:"result"` // success, failure, denied
	Details     map[string]interface{} `json:"details" db:"details"`
	RequestInfo RequestInfo            `json:"request_info" db:"-"`
	Metadata    map[string]string      `json:"metadata" db:"metadata"`
}

// Actor represents the entity performing the action.
type Actor struct {
	Type       string `json:"type" db:"actor_type"`       // user, api_key, service
	ID         string `json:"id" db:"actor_id"`             // user_id or api_key_id
	Name       string `json:"name" db:"actor_name"`         // email or key name
	Role       string `json:"role" db:"actor_role"`         // admin, developer, viewer
	IP         string `json:"ip" db:"actor_ip"`             // client IP
	UserAgent  string `json:"user_agent" db:"user_agent"`   // browser/agent
}

// Resource represents the resource being accessed.
type Resource struct {
	Type       string `json:"type" db:"resource_type"`      // project, provider, api_key
	ID         string `json:"id" db:"resource_id"`          // resource identifier
	Name       string `json:"name" db:"resource_name"`      // human-readable name
	Workspace  string `json:"workspace" db:"workspace_id"`  // workspace/tenant
}

// RequestInfo contains HTTP request details.
type RequestInfo struct {
	Method     string `json:"method" db:"request_method"`
	Path       string `json:"path" db:"request_path"`
	Query      string `json:"query" db:"request_query"`
	TraceID    string `json:"trace_id" db:"trace_id"`
	RequestID  string `json:"request_id" db:"request_id"`
}

// EventFilter provides filtering for audit event queries.
type EventFilter struct {
	Types       []EventType
	Severities  []EventSeverity
	ActorID     string
	ResourceID  string
	StartTime   *time.Time
	EndTime     *time.Time
	Limit       int
	Offset      int
}

// SeverityForEventType returns the default severity for an event type.
func SeverityForEventType(eventType EventType) EventSeverity {
	switch eventType {
	case EventAuthSuccess, EventTokenRefresh, EventAPIKeyUsed:
		return SeverityInfo
	case EventAuthFailure, EventAccessDenied, EventRateLimitExceeded:
		return SeverityWarning
	case EventSuspiciousActivity, EventIPBlocked:
		return SeverityError
	case EventConfigChange, EventRoleEscalation:
		return SeverityCritical
	default:
		return SeverityInfo
	}
}

// Category returns the category for an event type.
func (e EventType) Category() string {
	switch e {
	case EventAuthSuccess, EventAuthFailure, EventTokenRefresh, EventTokenRevoked, EventLogout:
		return "authentication"
	case EventAccessDenied, EventPermissionCheck, EventRoleEscalation:
		return "authorization"
	case EventAPIKeyCreated, EventAPIKeyRevoked, EventAPIKeyUsed, EventAPIKeyRotated:
		return "api_keys"
	case EventRateLimitExceeded, EventRateLimitWarning:
		return "rate_limiting"
	case EventSuspiciousActivity, EventIPBlocked, EventConfigChange:
		return "security"
	default:
		return "audit"
	}
}

// String returns the string representation of the event type.
func (e EventType) String() string {
	return string(e)
}

// IsAuthEvent returns true if the event is an authentication event.
func (e EventType) IsAuthEvent() bool {
	return e.Category() == "authentication"
}

// IsSecurityEvent returns true if the event is a security event.
func (e EventType) IsSecurityEvent() bool {
	return e.Category() == "security" || e == EventAccessDenied || e == EventAuthFailure
}
