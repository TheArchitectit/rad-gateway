package controlroom

import (
	"encoding/json"
	"time"

	"radgateway/internal/db"
)

// ControlRoom represents an operational view that filters resources by tags.
// It provides a customizable dashboard for monitoring specific subsets of resources.
type ControlRoom struct {
	ID              string          `json:"id" db:"id"`
	WorkspaceID     string          `json:"workspaceId" db:"workspace_id"`
	Slug            string          `json:"slug" db:"slug"`
	Name            string          `json:"name" db:"name"`
	Description     string          `json:"description,omitempty" db:"description"`
	TagFilter       string          `json:"tagFilter" db:"tag_filter"`
	DashboardLayout json.RawMessage `json:"dashboardLayout,omitempty" db:"dashboard_layout"`
	CreatedBy       string          `json:"createdBy,omitempty" db:"created_by"`
	CreatedAt       time.Time       `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time       `json:"updatedAt" db:"updated_at"`

	// ParsedTagFilter is the parsed representation (not stored in DB).
	// Populated after loading from database.
	ParsedTagFilter TagExpression `json:"-" db:"-"`
}

// ToDBModel converts this ControlRoom to the database model.
func (cr *ControlRoom) ToDBModel() *db.ControlRoom {
	desc := cr.Description
	dashboardLayout := cr.DashboardLayout
	createdBy := cr.CreatedBy

	return &db.ControlRoom{
		ID:              cr.ID,
		WorkspaceID:     cr.WorkspaceID,
		Slug:            cr.Slug,
		Name:            cr.Name,
		Description:     &desc,
		TagFilter:       cr.TagFilter,
		DashboardLayout: dashboardLayout,
		CreatedBy:       &createdBy,
		CreatedAt:       cr.CreatedAt,
		UpdatedAt:       cr.UpdatedAt,
	}
}

// FromDBModel converts a database model to this ControlRoom.
func (cr *ControlRoom) FromDBModel(m *db.ControlRoom) {
	cr.ID = m.ID
	cr.WorkspaceID = m.WorkspaceID
	cr.Slug = m.Slug
	cr.Name = m.Name
	if m.Description != nil {
		cr.Description = *m.Description
	}
	cr.TagFilter = m.TagFilter
	cr.DashboardLayout = m.DashboardLayout
	if m.CreatedBy != nil {
		cr.CreatedBy = *m.CreatedBy
	}
	cr.CreatedAt = m.CreatedAt
	cr.UpdatedAt = m.UpdatedAt
}

// ParseTagFilter parses the TagFilter string into a TagExpression.
// Should be called after loading from the database.
func (cr *ControlRoom) ParseTagFilter() error {
	if cr.TagFilter == "" {
		cr.ParsedTagFilter = nil
		return nil
	}
	parser := NewTagQueryParser()
	expr, err := parser.Parse(cr.TagFilter)
	if err != nil {
		return err
	}
	cr.ParsedTagFilter = expr
	return nil
}

// MatchesResource checks if a resource matches this control room's tag filter.
func (cr *ControlRoom) MatchesResource(resource TaggedResource) bool {
	if cr.ParsedTagFilter == nil {
		if err := cr.ParseTagFilter(); err != nil {
			return false
		}
	}
	if cr.ParsedTagFilter == nil {
		// No filter means match all
		return true
	}
	return Evaluate(cr.ParsedTagFilter, resource.Tags)
}

// CreateControlRoomRequest represents a request to create a new control room.
type CreateControlRoomRequest struct {
	ID              string          `json:"id,omitempty"`            // Optional, generated if not provided
	Name            string          `json:"name" validate:"required"`  // Display name
	Description     string          `json:"description,omitempty"`     // Optional description
	TagFilter       string          `json:"tagFilter" validate:"required"` // Tag query expression
	DashboardLayout json.RawMessage `json:"dashboardLayout,omitempty"` // Optional layout config
}

// UpdateControlRoomRequest represents a request to update a control room.
type UpdateControlRoomRequest struct {
	Name            string          `json:"name,omitempty"`
	Description     string          `json:"description,omitempty"`
	TagFilter       string          `json:"tagFilter,omitempty"`
	DashboardLayout json.RawMessage `json:"dashboardLayout,omitempty"`
}

// ControlRoomResponse is the API response format for control rooms.
type ControlRoomResponse struct {
	ID              string          `json:"id"`
	WorkspaceID     string          `json:"workspaceId"`
	Slug            string          `json:"slug"`
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	TagFilter       string          `json:"tagFilter"`
	DashboardLayout json.RawMessage `json:"dashboardLayout,omitempty"`
	CreatedBy       string          `json:"createdBy,omitempty"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`

	// Computed fields
	MatchedResourceCount int `json:"matchedResourceCount,omitempty"`
}

// FromControlRoom creates a response from a ControlRoom model.
func (r *ControlRoomResponse) FromControlRoom(cr *ControlRoom) {
	r.ID = cr.ID
	r.WorkspaceID = cr.WorkspaceID
	r.Slug = cr.Slug
	r.Name = cr.Name
	r.Description = cr.Description
	r.TagFilter = cr.TagFilter
	r.DashboardLayout = cr.DashboardLayout
	r.CreatedBy = cr.CreatedBy
	r.CreatedAt = cr.CreatedAt
	r.UpdatedAt = cr.UpdatedAt
}

// ControlRoomListResponse represents a list of control rooms.
type ControlRoomListResponse struct {
	Data       []ControlRoomResponse `json:"data"`
	Total      int                   `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"pageSize"`
	HasMore    bool                  `json:"hasMore"`
}

// ListControlRoomsRequest represents the query parameters for listing control rooms.
type ListControlRoomsRequest struct {
	WorkspaceID string `json:"workspaceId,omitempty" query:"workspaceId"`
	Page        int    `json:"page,omitempty" query:"page"`
	PageSize    int    `json:"pageSize,omitempty" query:"pageSize"`
}

// DefaultPageSize is the default number of results per page.
const DefaultPageSize = 20

// MaxPageSize is the maximum allowed page size.
const MaxPageSize = 100

// ValidateAndNormalize normalizes the list request parameters.
func (req *ListControlRoomsRequest) ValidateAndNormalize() {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = DefaultPageSize
	}
	if req.PageSize > MaxPageSize {
		req.PageSize = MaxPageSize
	}
}

// ResourceMatchRequest represents a request to find matching resources.
type ResourceMatchRequest struct {
	ResourceTypes []string `json:"resourceTypes,omitempty"` // Filter by types: provider, model, apikey, etc.
	Limit         int      `json:"limit,omitempty"`
}

// ResourceMatchResponse contains resources matching a control room filter.
type ResourceMatchResponse struct {
	Resources []TaggedResource `json:"resources"`
	Total     int              `json:"total"`
}

// AccessLevel represents permission levels for control room access.
type AccessLevel string

const (
	// AccessView allows read-only access to the control room dashboard.
	AccessView AccessLevel = "view"
	// AccessOperator allows triggering actions like pausing routes.
	AccessOperator AccessLevel = "operator"
	// AccessAdmin allows modifying control room settings and managing users.
	AccessAdmin AccessLevel = "admin"
	// AccessBilling allows viewing cost reports and usage data.
	AccessBilling AccessLevel = "billing"
)

// IsValid checks if the access level is valid.
func (a AccessLevel) IsValid() bool {
	switch a {
	case AccessView, AccessOperator, AccessAdmin, AccessBilling:
		return true
	}
	return false
}

// Permissions returns the permissions granted by this access level.
func (a AccessLevel) Permissions() []Permission {
	switch a {
	case AccessView:
		return []Permission{PermViewDashboard, PermViewUsage}
	case AccessOperator:
		return append(AccessView.Permissions(), PermPauseRoute, PermTriggerFailover)
	case AccessAdmin:
		return append(AccessOperator.Permissions(), PermModifyResources, PermManageUsers)
	case AccessBilling:
		return []Permission{PermViewUsage, PermViewCosts, PermExportReports}
	}
	return nil
}

// Permission represents a granular permission.
type Permission string

const (
	PermViewDashboard     Permission = "view:dashboard"
	PermViewUsage         Permission = "view:usage"
	PermViewCosts         Permission = "view:costs"
	PermPauseRoute        Permission = "action:pause-route"
	PermTriggerFailover   Permission = "action:trigger-failover"
	PermModifyResources   Permission = "admin:modify-resources"
	PermManageUsers       Permission = "admin:manage-users"
	PermExportReports     Permission = "admin:export-reports"
)

// HasPermission checks if a set of permissions contains a specific permission.
func HasPermission(permissions []Permission, target Permission) bool {
	for _, p := range permissions {
		if p == target {
			return true
		}
	}
	return false
}

// ControlRoomAccess represents access granted to a user for a control room.
type ControlRoomAccess struct {
	ControlRoomID string      `json:"controlRoomId"`
	UserID        string      `json:"userId"`
	AccessLevel   AccessLevel `json:"accessLevel"`
	GrantedBy     string      `json:"grantedBy,omitempty"`
	GrantedAt     time.Time   `json:"grantedAt"`
	ExpiresAt     *time.Time  `json:"expiresAt,omitempty"`
}

// IsExpired checks if the access has expired.
func (a *ControlRoomAccess) IsExpired() bool {
	if a.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*a.ExpiresAt)
}

// Valid checks if the access is valid (not expired and has valid level).
func (a *ControlRoomAccess) Valid() bool {
	return !a.IsExpired() && a.AccessLevel.IsValid()
}

// GrantAccessRequest represents a request to grant access to a control room.
type GrantAccessRequest struct {
	UserID      string     `json:"userId" validate:"required"`
	AccessLevel AccessLevel `json:"accessLevel" validate:"required"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
}

// UpdateAccessRequest represents a request to update access permissions.
type UpdateAccessRequest struct {
	AccessLevel AccessLevel `json:"accessLevel" validate:"required"`
	ExpiresAt   *time.Time  `json:"expiresAt,omitempty"`
}
