package rbac

import (
	"testing"
)

func TestParseRole(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Role
		wantErr bool
	}{
		{
			name:  "admin role",
			input: "admin",
			want:  RoleAdmin,
		},
		{
			name:  "developer role",
			input: "developer",
			want:  RoleDeveloper,
		},
		{
			name:  "viewer role",
			input: "viewer",
			want:  RoleViewer,
		},
		{
			name:  "system role",
			input: "system",
			want:  RoleSystem,
		},
		{
			name:  "role with whitespace",
			input: "  ADMIN  ",
			want:  RoleAdmin,
		},
		{
			name:  "role uppercase",
			input: "DEVELOPER",
			want:  RoleDeveloper,
		},
		{
			name:    "invalid role",
			input:   "superuser",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRole(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRole() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoleIsValid(t *testing.T) {
	tests := []struct {
		role Role
		want bool
	}{
		{RoleAdmin, true},
		{RoleDeveloper, true},
		{RoleViewer, true},
		{RoleSystem, true},
		{Role("invalid"), false},
		{Role(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			got := tt.role.IsValid()
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRolePermissions(t *testing.T) {
	tests := []struct {
		name string
		role Role
		perm Permission
		want bool
	}{
		{
			name: "admin has project read",
			role: RoleAdmin,
			perm: PermProjectRead,
			want: true,
		},
		{
			name: "admin has system admin",
			role: RoleAdmin,
			perm: PermSystemAdmin,
			want: true,
		},
		{
			name: "viewer has project read",
			role: RoleViewer,
			perm: PermProjectRead,
			want: true,
		},
		{
			name: "viewer does not have project write",
			role: RoleViewer,
			perm: PermProjectWrite,
			want: false,
		},
		{
			name: "developer has project write",
			role: RoleDeveloper,
			perm: PermProjectWrite,
			want: true,
		},
		{
			name: "developer does not have project delete",
			role: RoleDeveloper,
			perm: PermProjectDelete,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perms := RolePermissions(tt.role)
			got := HasPermission(perms, tt.perm)
			if got != tt.want {
				t.Errorf("RolePermissions(%v) has %v = %v, want %v", tt.role, tt.perm, got, tt.want)
			}
		})
	}
}

func TestRoleHierarchy(t *testing.T) {
	tests := []struct {
		role     Role
		other    Role
		expected bool
	}{
		{RoleAdmin, RoleDeveloper, true},
		{RoleAdmin, RoleViewer, true},
		{RoleDeveloper, RoleViewer, true},
		{RoleViewer, RoleViewer, true},
		{RoleViewer, RoleDeveloper, false},
		{RoleDeveloper, RoleAdmin, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role)+"_vs_"+string(tt.other), func(t *testing.T) {
			got := tt.role.IsAtLeast(tt.other)
			if got != tt.expected {
				t.Errorf("IsAtLeast() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRoleIsHigherThan(t *testing.T) {
	tests := []struct {
		role     Role
		other    Role
		expected bool
	}{
		{RoleAdmin, RoleDeveloper, true},
		{RoleAdmin, RoleViewer, true},
		{RoleDeveloper, RoleViewer, true},
		{RoleViewer, RoleViewer, false}, // Same level, not higher
		{RoleViewer, RoleDeveloper, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role)+"_vs_"+string(tt.other), func(t *testing.T) {
			got := tt.role.IsHigherThan(tt.other)
			if got != tt.expected {
				t.Errorf("IsHigherThan() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCanAccessResource(t *testing.T) {
	tests := []struct {
		name         string
		role         Role
		resourceType string
		want         bool
	}{
		{
			name:         "admin can access projects",
			role:         RoleAdmin,
			resourceType: "project",
			want:         true,
		},
		{
			name:         "viewer can read project",
			role:         RoleViewer,
			resourceType: "project",
			want:         true,
		},
		{
			name:         "viewer can read apikey",
			role:         RoleViewer,
			resourceType: "apikey",
			want:         true,
		},
		{
			name:         "case insensitive resource type",
			role:         RoleViewer,
			resourceType: "PROJECT",
			want:         true,
		},
		{
			name:         "unknown resource type",
			role:         RoleAdmin,
			resourceType: "unknown",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.role.CanAccessResource(tt.resourceType)
			if got != tt.want {
				t.Errorf("CanAccessResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequiredRoleForAction(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		action       string
		want         Role
	}{
		{
			name:         "delete requires admin",
			resourceType: "project",
			action:       "delete",
			want:         RoleAdmin,
		},
		{
			name:         "write requires developer",
			resourceType: "project",
			action:       "write",
			want:         RoleDeveloper,
		},
		{
			name:         "read requires viewer",
			resourceType: "project",
			action:       "read",
			want:         RoleViewer,
		},
		{
			name:         "admin action requires admin",
			resourceType: "user",
			action:       "admin",
			want:         RoleAdmin,
		},
		{
			name:         "create requires developer",
			resourceType: "project",
			action:       "create",
			want:         RoleDeveloper,
		},
		{
			name:         "list requires viewer",
			resourceType: "project",
			action:       "list",
			want:         RoleViewer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RequiredRoleForAction(tt.resourceType, tt.action)
			if got != tt.want {
				t.Errorf("RequiredRoleForAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoleDescription(t *testing.T) {
	tests := []struct {
		role Role
		want string
	}{
		{RoleAdmin, "Full system access"},
		{RoleDeveloper, "Development access"},
		{RoleViewer, "Read-only access"},
		{RoleSystem, "Internal system role"},
		{Role("unknown"), "Unknown role"},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			got := RoleDescription(tt.role)
			if len(got) < len(tt.want) || got[:len(tt.want)] != tt.want {
				t.Errorf("RoleDescription() = %q, want to start with %q", got, tt.want)
			}
		})
	}
}

func TestAllRoles(t *testing.T) {
	roles := AllRoles()
	if len(roles) != 3 {
		t.Errorf("AllRoles() should return 3 roles, got %d", len(roles))
	}

	// Check that all expected roles are present
	hasAdmin, hasDev, hasViewer := false, false, false
	for _, r := range roles {
		switch r {
		case RoleAdmin:
			hasAdmin = true
		case RoleDeveloper:
			hasDev = true
		case RoleViewer:
			hasViewer = true
		}
	}

	if !hasAdmin || !hasDev || !hasViewer {
		t.Error("AllRoles() should include admin, developer, and viewer")
	}
}
