// Package seeds provides database seeding functionality for RAD Gateway.
package seeds

import (
	"context"
	"fmt"
	"log/slog"

	"radgateway/internal/db"
)

// Seeder handles database seeding operations.
type Seeder struct {
	db  db.Database
	log *slog.Logger
}

// NewSeeder creates a new database seeder.
func NewSeeder(database db.Database, log *slog.Logger) *Seeder {
	return &Seeder{
		db:  database,
		log: log,
	}
}

// SeedOptions contains options for seeding operations.
type SeedOptions struct {
	Scenario    string
	Truncate    bool
	DryRun      bool
	WorkspaceID string // Optional: only seed specific workspace
}

// Seed applies seed data to the database.
func (s *Seeder) Seed(ctx context.Context, opts SeedOptions) error {
	scenario := FindScenario(opts.Scenario)
	if scenario == nil {
		return fmt.Errorf("unknown scenario: %s", opts.Scenario)
	}

	s.log.Info("Starting database seed",
		"scenario", scenario.Name,
		"description", scenario.Description,
	)

	// Generate seed data
	g := NewGenerator()
	data := scenario.Generator(g)

	if opts.DryRun {
		s.log.Info("Dry run mode - no data will be inserted")
		s.printStats(data)
		return nil
	}

	// Truncate existing data if requested
	if opts.Truncate {
		if err := s.truncateAll(ctx); err != nil {
			return fmt.Errorf("failed to truncate tables: %w", err)
		}
	}

	// Insert data in order respecting foreign keys
	if err := s.insertData(ctx, data, opts); err != nil {
		return fmt.Errorf("failed to insert seed data: %w", err)
	}

	s.log.Info("Database seed completed successfully")
	s.printStats(data)

	return nil
}

// insertData inserts all seed data respecting foreign key constraints.
func (s *Seeder) insertData(ctx context.Context, data *FullSeedData, opts SeedOptions) error {
	// 1. Workspaces
	s.log.Info("Inserting workspaces", "count", len(data.Workspaces))
	for _, ws := range data.Workspaces {
		if opts.WorkspaceID != "" && ws.ID != opts.WorkspaceID {
			continue
		}
		if err := s.db.Workspaces().Create(ctx, &ws); err != nil {
			return fmt.Errorf("failed to create workspace %s: %w", ws.ID, err)
		}
	}

	// 2. Roles
	s.log.Info("Inserting roles", "count", len(data.Roles))
	for _, role := range data.Roles {
		if err := s.db.Roles().Create(ctx, &role); err != nil {
			return fmt.Errorf("failed to create role %s: %w", role.ID, err)
		}
	}

	// 3. Permissions
	s.log.Info("Inserting permissions", "count", len(data.Permissions))
	for _, perm := range data.Permissions {
		if err := s.db.Permissions().Create(ctx, &perm); err != nil {
			return fmt.Errorf("failed to create permission %s: %w", perm.ID, err)
		}
	}

	// 4. Role permissions
	s.log.Info("Inserting role permissions", "count", len(data.RolePermissions))
	for _, rp := range data.RolePermissions {
		if err := s.db.Permissions().AssignToRole(ctx, rp.RoleID, rp.PermissionID); err != nil {
			return fmt.Errorf("failed to assign permission %s to role %s: %w", rp.PermissionID, rp.RoleID, err)
		}
	}

	// 5. Users
	s.log.Info("Inserting users", "count", len(data.Users))
	for _, user := range data.Users {
		if opts.WorkspaceID != "" && user.WorkspaceID != opts.WorkspaceID {
			continue
		}
		if err := s.db.Users().Create(ctx, &user); err != nil {
			return fmt.Errorf("failed to create user %s: %w", user.ID, err)
		}
	}

	// 6. User roles
	s.log.Info("Inserting user roles", "count", len(data.UserRoles))
	for _, ur := range data.UserRoles {
		if err := s.db.Roles().AssignToUser(ctx, ur.UserID, ur.RoleID, ur.GrantedBy, ur.ExpiresAt); err != nil {
			return fmt.Errorf("failed to assign role %s to user %s: %w", ur.RoleID, ur.UserID, err)
		}
	}

	// 7. Tags
	s.log.Info("Inserting tags", "count", len(data.Tags))
	for _, tag := range data.Tags {
		if opts.WorkspaceID != "" && tag.WorkspaceID != opts.WorkspaceID {
			continue
		}
		if err := s.db.Tags().Create(ctx, &tag); err != nil {
			return fmt.Errorf("failed to create tag %s: %w", tag.ID, err)
		}
	}

	// 8. Providers
	s.log.Info("Inserting providers", "count", len(data.Providers))
	for _, prov := range data.Providers {
		if opts.WorkspaceID != "" && prov.WorkspaceID != opts.WorkspaceID {
			continue
		}
		if err := s.db.Providers().Create(ctx, &prov); err != nil {
			return fmt.Errorf("failed to create provider %s: %w", prov.ID, err)
		}
	}

	// 9. Provider tags
	s.log.Info("Inserting provider tags", "count", len(data.ProviderTags))
	for _, pt := range data.ProviderTags {
		if err := s.db.Tags().AssignToProvider(ctx, pt.ProviderID, pt.TagID); err != nil {
			return fmt.Errorf("failed to assign tag %s to provider %s: %w", pt.TagID, pt.ProviderID, err)
		}
	}

	// 10. API Keys
	s.log.Info("Inserting API keys", "count", len(data.APIKeys))
	for _, key := range data.APIKeys {
		if opts.WorkspaceID != "" && key.WorkspaceID != opts.WorkspaceID {
			continue
		}
		if err := s.db.APIKeys().Create(ctx, &key); err != nil {
			return fmt.Errorf("failed to create API key %s: %w", key.ID, err)
		}
	}

	// 11. API key tags
	s.log.Info("Inserting API key tags", "count", len(data.APIKeyTags))
	for _, akt := range data.APIKeyTags {
		if err := s.db.Tags().AssignToAPIKey(ctx, akt.APIKeyID, akt.TagID); err != nil {
			return fmt.Errorf("failed to assign tag %s to API key %s: %w", akt.TagID, akt.APIKeyID, err)
		}
	}

	// 12. Control rooms
	s.log.Info("Inserting control rooms", "count", len(data.ControlRooms))
	for _, cr := range data.ControlRooms {
		if opts.WorkspaceID != "" && cr.WorkspaceID != opts.WorkspaceID {
			continue
		}
		if err := s.db.ControlRooms().Create(ctx, &cr); err != nil {
			return fmt.Errorf("failed to create control room %s: %w", cr.ID, err)
		}
	}

	// 13. Control room access
	s.log.Info("Inserting control room access", "count", len(data.ControlRoomAccess))
	for _, cra := range data.ControlRoomAccess {
		if err := s.db.ControlRooms().GrantAccess(ctx, &cra); err != nil {
			return fmt.Errorf("failed to grant access to control room %s: %w", cra.ControlRoomID, err)
		}
	}

	// 14. Quotas
	s.log.Info("Inserting quotas", "count", len(data.Quotas))
	for _, quota := range data.Quotas {
		if opts.WorkspaceID != "" && quota.WorkspaceID != opts.WorkspaceID {
			continue
		}
		if err := s.db.Quotas().Create(ctx, &quota); err != nil {
			return fmt.Errorf("failed to create quota %s: %w", quota.ID, err)
		}
	}

	// 15. Quota assignments
	s.log.Info("Inserting quota assignments", "count", len(data.QuotaAssignments))
	for _, qa := range data.QuotaAssignments {
		if err := s.db.Quotas().AssignQuota(ctx, &qa); err != nil {
			return fmt.Errorf("failed to assign quota %s: %w", qa.QuotaID, err)
		}
	}

	// 16. Usage records
	s.log.Info("Inserting usage records", "count", len(data.UsageRecords))
	for _, usage := range data.UsageRecords {
		if opts.WorkspaceID != "" && usage.WorkspaceID != opts.WorkspaceID {
			continue
		}
		if err := s.db.UsageRecords().Create(ctx, &usage); err != nil {
			return fmt.Errorf("failed to create usage record %s: %w", usage.ID, err)
		}
	}

	// 17. Trace events
	s.log.Info("Inserting trace events", "count", len(data.TraceEvents))
	if len(data.TraceEvents) > 0 {
		if err := s.db.TraceEvents().CreateBatch(ctx, data.TraceEvents); err != nil {
			return fmt.Errorf("failed to create trace events: %w", err)
		}
	}

	return nil
}

// truncateAll truncates all tables.
func (s *Seeder) truncateAll(ctx context.Context) error {
	s.log.Info("Truncating all tables")

	// Note: This is a simplified version. In production, use CASCADE DELETE
	// or proper truncate commands respecting foreign key constraints.
	queries := []string{
		"DELETE FROM trace_events",
		"DELETE FROM usage_record_tags",
		"DELETE FROM usage_records",
		"DELETE FROM quota_assignments",
		"DELETE FROM quotas",
		"DELETE FROM control_room_access",
		"DELETE FROM control_rooms",
		"DELETE FROM api_key_tags",
		"DELETE FROM api_keys",
		"DELETE FROM provider_tags",
		"DELETE FROM provider_health",
		"DELETE FROM circuit_breaker_states",
		"DELETE FROM providers",
		"DELETE FROM tags",
		"DELETE FROM user_roles",
		"DELETE FROM users",
		"DELETE FROM role_permissions",
		"DELETE FROM roles",
		"DELETE FROM permissions",
		"DELETE FROM workspaces",
	}

	for _, query := range queries {
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			s.log.Warn("Failed to truncate table", "query", query, "error", err)
			// Continue with other tables
		}
	}

	return nil
}

// printStats prints seed statistics.
func (s *Seeder) printStats(data *FullSeedData) {
	s.log.Info("Seed data statistics",
		"workspaces", len(data.Workspaces),
		"users", len(data.Users),
		"roles", len(data.Roles),
		"permissions", len(data.Permissions),
		"tags", len(data.Tags),
		"providers", len(data.Providers),
		"apiKeys", len(data.APIKeys),
		"controlRooms", len(data.ControlRooms),
		"quotas", len(data.Quotas),
		"usageRecords", len(data.UsageRecords),
	)
}

// FindScenario finds a scenario by name.
func FindScenario(name string) *Scenario {
	for _, s := range AllScenarios() {
		if s.Name == name {
			return &s
		}
	}
	return nil
}

// ListScenarios returns all available scenario names.
func ListScenarios() []string {
	var names []string
	for _, s := range AllScenarios() {
		names = append(names, s.Name)
	}
	return names
}

// ScenarioNames returns all available scenario names with descriptions.
func ScenarioNames() map[string]string {
	result := make(map[string]string)
	for _, s := range AllScenarios() {
		result[s.Name] = s.Description
	}
	return result
}
