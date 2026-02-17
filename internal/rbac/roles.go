// Package rbac provides Role-Based Access Control functionality.
package rbac

import (
	"fmt"
	"strings"
)

// Role represents a named collection of permissions.
// The system supports three built-in roles: Admin, Developer, and Viewer.
type Role string

// Built-in system roles.
const (
	RoleAdmin     Role = "admin"
	RoleDeveloper Role = "developer"
	RoleViewer    Role = "viewer"
	RoleSystem    Role = "system" // Internal system role with all permissions
)

// AllRoles returns all available roles.
func AllRoles() []Role {
	return []Role{RoleAdmin, RoleDeveloper, RoleViewer}
}

// IsValid checks if a role is valid.
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleDeveloper, RoleViewer, RoleSystem:
		return true
	default:
		return false
	}
}

// String returns the string representation of a role.
func (r Role) String() string {
	return string(r)
}

// ParseRole parses a role from a string.
func ParseRole(s string) (Role, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "admin":
		return RoleAdmin, nil
	case "developer":
		return RoleDeveloper, nil
	case "viewer":
		return RoleViewer, nil
	case "system":
		return RoleSystem, nil
	default:
		return "", fmt.Errorf("unknown role: %s", s)
	}
}

// RolePermissions returns the default permissions for a role.
// These permissions are used when no custom permissions are configured.
func RolePermissions(role Role) Permission {
	switch role {
	case RoleAdmin:
		// Admin has all permissions
		return ^Permission(0) // All bits set
	case RoleDeveloper:
		// Developer can read and write, but not delete or administrate
		return PermRead | PermWrite |
			PermProjectRead | PermProjectWrite |
			PermAPIKeyRead | PermAPIKeyWrite |
			PermProviderRead | PermProviderWrite |
			PermControlRoomRead | PermControlRoomWrite |
			PermUsageRead
	case RoleViewer:
		// Viewer can only read
		return PermRead |
			PermProjectRead |
			PermAPIKeyRead |
			PermProviderRead |
			PermControlRoomRead |
			PermUsageRead
	case RoleSystem:
		// System role has full access for internal operations
		return ^Permission(0)
	default:
		return 0
	}
}

// RoleDescription returns a human-readable description of a role.
func RoleDescription(role Role) string {
	switch role {
	case RoleAdmin:
		return "Full system access - can read, write, delete, and administer all resources"
	case RoleDeveloper:
		return "Development access - can read and write resources in assigned projects"
	case RoleViewer:
		return "Read-only access - can view resources in assigned projects"
	case RoleSystem:
		return "Internal system role with unrestricted access"
	default:
		return "Unknown role"
	}
}

// CanAccessResource checks if a role has access to a resource type.
func (r Role) CanAccessResource(resourceType string) bool {
	perms := RolePermissions(r)
	switch strings.ToLower(resourceType) {
	case "project":
		return HasPermission(perms, PermProjectRead)
	case "apikey":
		return HasPermission(perms, PermAPIKeyRead)
	case "provider":
		return HasPermission(perms, PermProviderRead)
	case "controlroom":
		return HasPermission(perms, PermControlRoomRead)
	case "usage":
		return HasPermission(perms, PermUsageRead)
	case "user", "system":
		return HasPermission(perms, PermUserManage) || HasPermission(perms, PermSystemAdmin)
	default:
		return false
	}
}

// RequiredRoleForAction returns the minimum role required for a specific action.
// This is used for default access control decisions.
func RequiredRoleForAction(resourceType, action string) Role {
	switch strings.ToLower(action) {
	case "delete":
		return RoleAdmin
	case "write", "create", "update":
		if isSensitiveResource(resourceType) {
			return RoleDeveloper
		}
		return RoleDeveloper
	case "read", "list", "get":
		return RoleViewer
	case "admin":
		return RoleAdmin
	default:
		return RoleViewer
	}
}

// isSensitiveResource returns true for resources requiring elevated access.
func isSensitiveResource(resourceType string) bool {
	switch strings.ToLower(resourceType) {
	case "system", "user", "config", "audit":
		return true
	default:
		return false
	}
}

// RoleHierarchy defines the role hierarchy for permission inheritance.
// Higher roles inherit permissions from lower roles.
// Admin > Developer > Viewer
func RoleHierarchy(role Role) int {
	switch role {
	case RoleAdmin:
		return 3
	case RoleDeveloper:
		return 2
	case RoleViewer:
		return 1
	default:
		return 0
	}
}

// IsAtLeast checks if a role has equal or higher privileges than another role.
func (r Role) IsAtLeast(other Role) bool {
	return RoleHierarchy(r) >= RoleHierarchy(other)
}

// IsHigherThan checks if a role has strictly higher privileges than another role.
func (r Role) IsHigherThan(other Role) bool {
	return RoleHierarchy(r) > RoleHierarchy(other)
}
