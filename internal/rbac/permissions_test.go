package rbac

import (
	"testing"
)

func TestPermissionBitwiseOperations(t *testing.T) {
	tests := []struct {
		name     string
		granted  Permission
		required Permission
		want     bool
	}{
		{
			name:     "has exact permission",
			granted:  PermProjectRead,
			required: PermProjectRead,
			want:     true,
		},
		{
			name:     "has permission in set",
			granted:  PermProjectRead | PermProjectWrite,
			required: PermProjectRead,
			want:     true,
		},
		{
			name:     "missing permission",
			granted:  PermProjectRead,
			required: PermProjectWrite,
			want:     false,
		},
		{
			name:     "multiple required - has all",
			granted:  PermRead | PermWrite,
			required: PermProjectRead | PermProjectWrite,
			want:     true,
		},
		{
			name:     "multiple required - missing one",
			granted:  PermProjectRead | PermAPIKeyRead,
			required: PermProjectRead | PermProjectWrite,
			want:     false,
		},
		{
			name:     "admin has all permissions",
			granted:  ^Permission(0), // All bits set
			required: PermSystemAdmin,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasPermission(tt.granted, tt.required)
			if got != tt.want {
				t.Errorf("HasPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasAnyPermission(t *testing.T) {
	tests := []struct {
		name     string
		granted  Permission
		required Permission
		want     bool
	}{
		{
			name:     "has one of required",
			granted:  PermProjectRead,
			required: PermProjectRead | PermProjectWrite,
			want:     true,
		},
		{
			name:     "has none of required",
			granted:  PermAPIKeyRead,
			required: PermProjectRead | PermProjectWrite,
			want:     false,
		},
		{
			name:     "has all of required",
			granted:  PermProjectRead | PermProjectWrite,
			required: PermProjectRead | PermProjectWrite,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasAnyPermission(tt.granted, tt.required)
			if got != tt.want {
				t.Errorf("HasAnyPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGrantRevoke(t *testing.T) {
	t.Run("grant adds permission", func(t *testing.T) {
		base := PermProjectRead
		result := Grant(base, PermProjectWrite)

		if !HasPermission(result, PermProjectRead) {
			t.Error("should still have original permission")
		}
		if !HasPermission(result, PermProjectWrite) {
			t.Error("should have new permission")
		}
	})

	t.Run("revoke removes permission", func(t *testing.T) {
		base := PermProjectRead | PermProjectWrite | PermProjectDelete
		result := Revoke(base, PermProjectWrite)

		if !HasPermission(result, PermProjectRead) {
			t.Error("should still have read permission")
		}
		if HasPermission(result, PermProjectWrite) {
			t.Error("should not have write permission")
		}
		if !HasPermission(result, PermProjectDelete) {
			t.Error("should still have delete permission")
		}
	})

	t.Run("revoke non-existent permission", func(t *testing.T) {
		base := PermProjectRead
		result := Revoke(base, PermProjectWrite)

		if !HasPermission(result, PermProjectRead) {
			t.Error("should still have read permission")
		}
	})
}

func TestPermissionString(t *testing.T) {
	tests := []struct {
		perm Permission
		want string
	}{
		{PermProjectRead, "project:read"},
		{PermProjectWrite, "project:write"},
		{PermAPIKeyRead, "apikey:read"},
		{PermSystemAdmin, "system:admin"},
		{Permission(0xFFFFFFFF), "permission:0xFFFFFFFF"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.perm.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParsePermission(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Permission
		wantErr bool
	}{
		{
			name:  "valid permission",
			input: "project:read",
			want:  PermProjectRead,
		},
		{
			name:  "permission with whitespace",
			input: "  PROJECT:WRITE  ",
			want:  PermProjectWrite,
		},
		{
			name:    "invalid permission",
			input:   "unknown:permission",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePermission(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePermission() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParsePermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParsePermissions(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    Permission
		wantErr bool
	}{
		{
			name:  "multiple valid permissions",
			input: []string{"project:read", "project:write"},
			want:  PermProjectRead | PermProjectWrite,
		},
		{
			name:  "empty list",
			input: []string{},
			want:  0,
		},
		{
			name:    "invalid permission in list",
			input:   []string{"project:read", "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePermissions(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePermissions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParsePermissions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllPermissions(t *testing.T) {
	perms := AllPermissions()
	if len(perms) == 0 {
		t.Error("AllPermissions() should return non-empty list")
	}

	// Verify each permission has a name
	for _, p := range perms {
		if p.String() == "" || p.String()[:10] == "permission:" {
			t.Errorf("Permission %d should have a defined name", p)
		}
	}
}
