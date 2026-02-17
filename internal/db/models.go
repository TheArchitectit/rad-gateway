// Package db contains database models for RAD Gateway.
// These models map to the PostgreSQL schema defined in migrations/.
package db

import (
	"time"
)

// Workspace represents a multi-tenancy boundary.
type Workspace struct {
	ID          string    `db:"id" json:"id"`
	Slug        string    `db:"slug" json:"slug"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description,omitempty"`
	Status      string    `db:"status" json:"status"`
	Settings    []byte    `db:"settings" json:"settings"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

// User represents a user account.
type User struct {
	ID           string     `db:"id" json:"id"`
	WorkspaceID  string     `db:"workspace_id" json:"workspaceId"`
	Email        string     `db:"email" json:"email"`
	DisplayName  *string    `db:"display_name" json:"displayName,omitempty"`
	Status       string     `db:"status" json:"status"`
	PasswordHash *string    `db:"password_hash" json:"-"`
	LastLoginAt  *time.Time `db:"last_login_at" json:"lastLoginAt,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updatedAt"`
}

// Role represents a role definition for RBAC.
type Role struct {
	ID          string    `db:"id" json:"id"`
	WorkspaceID *string   `db:"workspace_id" json:"workspaceId,omitempty"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description,omitempty"`
	IsSystem    bool      `db:"is_system" json:"isSystem"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

// Permission represents an individual permission.
type Permission struct {
	ID           string `db:"id" json:"id"`
	Name         string `db:"name" json:"name"`
	Description  string `db:"description" json:"description"`
	ResourceType string `db:"resource_type" json:"resourceType"`
	Action       string `db:"action" json:"action"`
}

// UserRole represents the assignment of a role to a user.
type UserRole struct {
	UserID    string     `db:"user_id" json:"userId"`
	RoleID    string     `db:"role_id" json:"roleId"`
	GrantedBy *string    `db:"granted_by" json:"grantedBy,omitempty"`
	GrantedAt time.Time  `db:"granted_at" json:"grantedAt"`
	ExpiresAt *time.Time `db:"expires_at" json:"expiresAt,omitempty"`
}

// RolePermission represents the assignment of a permission to a role.
type RolePermission struct {
	RoleID       string `db:"role_id" json:"roleId"`
	PermissionID string `db:"permission_id" json:"permissionId"`
}

// Tag represents a hierarchical tag for resource categorization.
type Tag struct {
	ID          string    `db:"id" json:"id"`
	WorkspaceID string    `db:"workspace_id" json:"workspaceId"`
	Category    string    `db:"category" json:"category"`
	Value       string    `db:"value" json:"value"`
	Description *string   `db:"description" json:"description,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
}

// String returns the tag in category:value format.
func (t Tag) String() string {
	return t.Category + ":" + t.Value
}

// Provider represents an AI provider configuration.
type Provider struct {
	ID              string    `db:"id" json:"id"`
	WorkspaceID     string    `db:"workspace_id" json:"workspaceId"`
	Slug            string    `db:"slug" json:"slug"`
	Name            string    `db:"name" json:"name"`
	ProviderType    string    `db:"provider_type" json:"providerType"`
	BaseURL         string    `db:"base_url" json:"baseUrl"`
	APIKeyEncrypted *string   `db:"api_key_encrypted" json:"-"`
	Config          []byte    `db:"config" json:"config"`
	Status          string    `db:"status" json:"status"`
	Priority        int       `db:"priority" json:"priority"`
	Weight          int       `db:"weight" json:"weight"`
	CreatedAt       time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time `db:"updated_at" json:"updatedAt"`
}

// ProviderTag links a provider to a tag.
type ProviderTag struct {
	ProviderID string `db:"provider_id" json:"providerId"`
	TagID      string `db:"tag_id" json:"tagId"`
}

// ProviderHealth represents the current health status of a provider.
type ProviderHealth struct {
	ProviderID          string     `db:"provider_id" json:"providerId"`
	Healthy             bool       `db:"healthy" json:"healthy"`
	LastCheckAt         time.Time  `db:"last_check_at" json:"lastCheckAt"`
	LastSuccessAt       *time.Time `db:"last_success_at" json:"lastSuccessAt,omitempty"`
	ConsecutiveFailures int        `db:"consecutive_failures" json:"consecutiveFailures"`
	LatencyMs           *int       `db:"latency_ms" json:"latencyMs,omitempty"`
	ErrorMessage        *string    `db:"error_message" json:"errorMessage,omitempty"`
	UpdatedAt           time.Time  `db:"updated_at" json:"updatedAt"`
}

// CircuitBreakerState represents the state of a circuit breaker for a provider.
type CircuitBreakerState struct {
	ProviderID       string     `db:"provider_id" json:"providerId"`
	State            string     `db:"state" json:"state"`
	Failures         int        `db:"failures" json:"failures"`
	Successes        int        `db:"successes" json:"successes"`
	LastFailureAt    *time.Time `db:"last_failure_at" json:"lastFailureAt,omitempty"`
	HalfOpenRequests   int        `db:"half_open_requests" json:"halfOpenRequests"`
	OpenedAt         *time.Time `db:"opened_at" json:"openedAt,omitempty"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updatedAt"`
}

// ControlRoom represents an operational view with tag-based filtering.
type ControlRoom struct {
	ID              string    `db:"id" json:"id"`
	WorkspaceID     string    `db:"workspace_id" json:"workspaceId"`
	Slug            string    `db:"slug" json:"slug"`
	Name            string    `db:"name" json:"name"`
	Description     *string   `db:"description" json:"description,omitempty"`
	TagFilter       string    `db:"tag_filter" json:"tagFilter"`
	DashboardLayout []byte    `db:"dashboard_layout" json:"dashboardLayout"`
	CreatedBy       *string   `db:"created_by" json:"createdBy,omitempty"`
	CreatedAt       time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time `db:"updated_at" json:"updatedAt"`
}

// ControlRoomAccess represents user access to a control room.
type ControlRoomAccess struct {
	ControlRoomID string     `db:"control_room_id" json:"controlRoomId"`
	UserID        string     `db:"user_id" json:"userId"`
	Role          string     `db:"role" json:"role"`
	GrantedBy     *string    `db:"granted_by" json:"grantedBy,omitempty"`
	GrantedAt     time.Time  `db:"granted_at" json:"grantedAt"`
	ExpiresAt     *time.Time `db:"expires_at" json:"expiresAt,omitempty"`
}

// APIKey represents an API key for authentication.
type APIKey struct {
	ID            string     `db:"id" json:"id"`
	WorkspaceID   string     `db:"workspace_id" json:"workspaceId"`
	Name          string     `db:"name" json:"name"`
	KeyHash       string     `db:"key_hash" json:"-"`
	KeyPreview    string     `db:"key_preview" json:"keyPreview"`
	Status        string     `db:"status" json:"status"`
	CreatedBy     *string    `db:"created_by" json:"createdBy,omitempty"`
	ExpiresAt     *time.Time `db:"expires_at" json:"expiresAt,omitempty"`
	LastUsedAt    *time.Time `db:"last_used_at" json:"lastUsedAt,omitempty"`
	RateLimit     *int       `db:"rate_limit" json:"rateLimit,omitempty"`
	AllowedModels []string   `db:"allowed_models" json:"allowedModels,omitempty"`
	AllowedAPIs   []string   `db:"allowed_apis" json:"allowedAPIs,omitempty"`
	Metadata      []byte     `db:"metadata" json:"metadata"`
	CreatedAt     time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updatedAt"`
}

// APIKeyTag links an API key to a tag.
type APIKeyTag struct {
	APIKeyID string `db:"api_key_id" json:"apiKeyId"`
	TagID    string `db:"tag_id" json:"tagId"`
}

// Quota represents a quota definition.
type Quota struct {
	ID               string    `db:"id" json:"id"`
	WorkspaceID      string    `db:"workspace_id" json:"workspaceId"`
	Name             string    `db:"name" json:"name"`
	Description      *string   `db:"description" json:"description,omitempty"`
	QuotaType        string    `db:"quota_type" json:"quotaType"`
	Period           string    `db:"period" json:"period"`
	LimitValue       int64     `db:"limit_value" json:"limitValue"`
	Scope            string    `db:"scope" json:"scope"`
	WarningThreshold int       `db:"warning_threshold" json:"warningThreshold"`
	CreatedAt        time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt        time.Time `db:"updated_at" json:"updatedAt"`
}

// QuotaAssignment represents a quota assigned to a resource.
type QuotaAssignment struct {
	QuotaID      string     `db:"quota_id" json:"quotaId"`
	ResourceType string     `db:"resource_type" json:"resourceType"`
	ResourceID   string     `db:"resource_id" json:"resourceId"`
	CurrentUsage int64      `db:"current_usage" json:"currentUsage"`
	PeriodStart  time.Time  `db:"period_start" json:"periodStart"`
	PeriodEnd    time.Time  `db:"period_end" json:"periodEnd"`
	WarningSent  bool       `db:"warning_sent" json:"warningSent"`
	ExceededAt   *time.Time `db:"exceeded_at" json:"exceededAt,omitempty"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updatedAt"`
}

// UsageRecord represents a single API request usage record.
type UsageRecord struct {
	ID               string     `db:"id" json:"id"`
	WorkspaceID      string     `db:"workspace_id" json:"workspaceId"`
	RequestID        string     `db:"request_id" json:"requestId"`
	TraceID          string     `db:"trace_id" json:"traceId"`
	APIKeyID         *string    `db:"api_key_id" json:"apiKeyId,omitempty"`
	ControlRoomID    *string    `db:"control_room_id" json:"controlRoomId,omitempty"`
	IncomingAPI      string     `db:"incoming_api" json:"incomingApi"`
	IncomingModel    string     `db:"incoming_model" json:"incomingModel"`
	SelectedModel    *string    `db:"selected_model" json:"selectedModel,omitempty"`
	ProviderID       *string    `db:"provider_id" json:"providerId,omitempty"`
	PromptTokens     int64      `db:"prompt_tokens" json:"promptTokens"`
	CompletionTokens int64      `db:"completion_tokens" json:"completionTokens"`
	TotalTokens      int64      `db:"total_tokens" json:"totalTokens"`
	CostUSD          *float64   `db:"cost_usd" json:"costUsd,omitempty"`
	DurationMs       int        `db:"duration_ms" json:"durationMs"`
	ResponseStatus   string     `db:"response_status" json:"responseStatus"`
	ErrorCode        *string    `db:"error_code" json:"errorCode,omitempty"`
	ErrorMessage     *string    `db:"error_message" json:"errorMessage,omitempty"`
	Attempts         int        `db:"attempts" json:"attempts"`
	RouteLog         []byte     `db:"route_log" json:"routeLog"`
	StartedAt        time.Time  `db:"started_at" json:"startedAt"`
	CompletedAt      *time.Time `db:"completed_at" json:"completedAt,omitempty"`
	CreatedAt        time.Time  `db:"created_at" json:"createdAt"`
}

// UsageRecordTag links a usage record to a tag.
type UsageRecordTag struct {
	UsageRecordID string `db:"usage_record_id" json:"usageRecordId"`
	TagID         string `db:"tag_id" json:"tagId"`
}

// TraceEvent represents a single event in a request trace.
type TraceEvent struct {
	ID         string     `db:"id" json:"id"`
	TraceID    string     `db:"trace_id" json:"traceId"`
	RequestID  string     `db:"request_id" json:"requestId"`
	EventType  string     `db:"event_type" json:"eventType"`
	EventOrder int        `db:"event_order" json:"eventOrder"`
	ProviderID *string    `db:"provider_id" json:"providerId,omitempty"`
	APIKeyID   *string    `db:"api_key_id" json:"apiKeyId,omitempty"`
	Message    string     `db:"message" json:"message"`
	Metadata   []byte     `db:"metadata" json:"metadata"`
	Timestamp  time.Time  `db:"timestamp" json:"timestamp"`
	DurationMs *int       `db:"duration_ms" json:"durationMs,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"createdAt"`
}
