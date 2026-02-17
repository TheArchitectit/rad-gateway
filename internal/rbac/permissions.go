// Package rbac provides Role-Based Access Control functionality.
// This package implements a permission system with three core roles:
// - Admin: Full system access
// - Developer: Read and write access to assigned projects
// - Viewer: Read-only access to assigned projects
package rbac

import (
	"fmt"
	"strings"
)

// Permission represents a specific action that can be performed on a resource.
// Permissions use bitmask encoding for efficient storage and checking.
type Permission uint32

// Permission bit masks - organized by resource type and action.
// Each permission is a unique bit allowing for efficient permission combination.
const (
	// Project permissions (0x00000001 - 0x0000000F)
	PermProjectRead   Permission = 1 << iota // View project details
	PermProjectWrite                         // Modify project settings
	PermProjectDelete                        // Delete project
	PermProjectAdmin                         // Full project administration

	// API Key permissions (0x00000010 - 0x000000F0)
	PermAPIKeyRead   // List and view API keys
	PermAPIKeyWrite  // Create and modify API keys
	PermAPIKeyDelete // Revoke API keys

	// Provider permissions (0x00000100 - 0x00000F00)
	PermProviderRead   // View provider configurations
	PermProviderWrite  // Configure providers
	PermProviderDelete // Remove providers

	// Control Room permissions (0x00001000 - 0x0000F000)
	PermControlRoomRead   // View control rooms
	PermControlRoomWrite  // Create and modify control rooms
	PermControlRoomDelete // Delete control rooms

	// Usage/Analytics permissions (0x00010000 - 0x000F0000)
	PermUsageRead // View usage data and analytics

	// Admin/System permissions (0x00F00000 - 0x0F000000)
	PermSystemConfig  // Modify system configuration
	PermUserManage    // Manage users and roles
	PermAuditRead     // View audit logs
	PermSystemAdmin   // Full system administration

	// Combined permission sets
	PermRead   = PermProjectRead | PermAPIKeyRead | PermProviderRead | PermControlRoomRead | PermUsageRead
	PermWrite  = PermProjectWrite | PermAPIKeyWrite | PermProviderWrite | PermControlRoomWrite
	PermDelete = PermProjectDelete | PermAPIKeyDelete | PermProviderDelete | PermControlRoomDelete
)

// permissionNames maps permissions to human-readable names.
var permissionNames = map[Permission]string{
	PermProjectRead:       "project:read",
	PermProjectWrite:      "project:write",
	PermProjectDelete:     "project:delete",
	PermProjectAdmin:      "project:admin",
	PermAPIKeyRead:        "apikey:read",
	PermAPIKeyWrite:       "apikey:write",
	PermAPIKeyDelete:      "apikey:delete",
	PermProviderRead:      "provider:read",
	PermProviderWrite:     "provider:write",
	PermProviderDelete:    "provider:delete",
	PermControlRoomRead:   "controlroom:read",
	PermControlRoomWrite:  "controlroom:write",
	PermControlRoomDelete: "controlroom:delete",
	PermUsageRead:         "usage:read",
	PermSystemConfig:      "system:config",
	PermUserManage:        "user:manage",
	PermAuditRead:         "audit:read",
	PermSystemAdmin:       "system:admin",
}

// permissionNameToValue maps string names back to permissions.
var permissionNameToValue = map[string]Permission{
	"project:read":       PermProjectRead,
	"project:write":      PermProjectWrite,
	"project:delete":     PermProjectDelete,
	"project:admin":      PermProjectAdmin,
	"apikey:read":        PermAPIKeyRead,
	"apikey:write":       PermAPIKeyWrite,
	"apikey:delete":      PermAPIKeyDelete,
	"provider:read":      PermProviderRead,
	"provider:write":     PermProviderWrite,
	"provider:delete":    PermProviderDelete,
	"controlroom:read":   PermControlRoomRead,
	"controlroom:write":  PermControlRoomWrite,
	"controlroom:delete": PermControlRoomDelete,
	"usage:read":         PermUsageRead,
	"system:config":      PermSystemConfig,
	"user:manage":        PermUserManage,
	"audit:read":         PermAuditRead,
	"system:admin":       PermSystemAdmin,
}

// HasPermission checks if a permission set includes a specific permission.
func HasPermission(granted, required Permission) bool {
	return granted&required == required
}

// HasAnyPermission checks if any of the required permissions are granted.
func HasAnyPermission(granted, required Permission) bool {
	return granted&required != 0
}

// Grant adds permissions to a permission set.
func Grant(existing, additional Permission) Permission {
	return existing | additional
}

// Revoke removes permissions from a permission set.
func Revoke(existing, toRemove Permission) Permission {
	return existing &^ toRemove
}

// String returns the human-readable name of a permission.
func (p Permission) String() string {
	if name, ok := permissionNames[p]; ok {
		return name
	}
	return fmt.Sprintf("permission:0x%08X", uint32(p))
}

// AllPermissions returns all individual permissions as a slice.
func AllPermissions() []Permission {
	return []Permission{
		PermProjectRead,
		PermProjectWrite,
		PermProjectDelete,
		PermProjectAdmin,
		PermAPIKeyRead,
		PermAPIKeyWrite,
		PermAPIKeyDelete,
		PermProviderRead,
		PermProviderWrite,
		PermProviderDelete,
		PermControlRoomRead,
		PermControlRoomWrite,
		PermControlRoomDelete,
		PermUsageRead,
		PermSystemConfig,
		PermUserManage,
		PermAuditRead,
		PermSystemAdmin,
	}
}

// ParsePermission parses a permission from its string representation.
func ParsePermission(s string) (Permission, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if perm, ok := permissionNameToValue[s]; ok {
		return perm, nil
	}
	return 0, fmt.Errorf("unknown permission: %s", s)
}

// ParsePermissions parses multiple permission strings.
func ParsePermissions(permissions []string) (Permission, error) {
	var result Permission
	for _, p := range permissions {
		perm, err := ParsePermission(p)
		if err != nil {
			return 0, err
		}
		result = Grant(result, perm)
	}
	return result, nil
}
