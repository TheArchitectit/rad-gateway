// Package seeds provides realistic seed data scenarios for RAD Gateway.
package seeds

import (
	"fmt"
	"time"

	"radgateway/internal/db"
)

// Scenario represents a complete seed data scenario.
type Scenario struct {
	Name        string
	Description string
	Generator   func(g *Generator) *FullSeedData
}

// AllScenarios returns all available seed scenarios.
func AllScenarios() []Scenario {
	return []Scenario{
		DevelopmentScenario(),
		ProductionLikeScenario(),
		StressTestScenario(),
		DemoWorkspaceScenario(),
		MinimalScenario(),
	}
}

// DevelopmentScenario creates data suitable for local development.
func DevelopmentScenario() Scenario {
	return Scenario{
		Name:        "development",
		Description: "Rich development dataset with multiple workspaces, users, and providers",
		Generator:   generateDevelopmentData,
	}
}

// ProductionLikeScenario creates data that mimics a production environment.
func ProductionLikeScenario() Scenario {
	return Scenario{
		Name:        "production-like",
		Description: "Production-like dataset with realistic usage patterns and quotas",
		Generator:   generateProductionLikeData,
	}
}

// StressTestScenario creates large volumes of data for performance testing.
func StressTestScenario() Scenario {
	return Scenario{
		Name:        "stress-test",
		Description: "Large dataset for stress and performance testing",
		Generator:   generateStressTestData,
	}
}

// DemoWorkspaceScenario creates a polished demo environment.
func DemoWorkspaceScenario() Scenario {
	return Scenario{
		Name:        "demo",
		Description: "Polished demo environment with realistic sample data",
		Generator:   generateDemoData,
	}
}

// MinimalScenario creates the minimal required data.
func MinimalScenario() Scenario {
	return Scenario{
		Name:        "minimal",
		Description: "Minimal dataset with just enough to run the application",
		Generator:   generateMinimalData,
	}
}

// generateDevelopmentData creates a rich development dataset.
func generateDevelopmentData(g *Generator) *FullSeedData {
	data := &FullSeedData{}
	now := g.now

	// Create workspaces
	workspaces := []struct {
		name        string
		slug        string
		description string
		status      string
	}{
		{"Acme Corp", "acme-corp", "Primary development workspace", "active"},
		{"Engineering", "engineering", "Engineering team workspace", "active"},
		{"Platform Team", "platform", "Platform infrastructure workspace", "active"},
		{"Research Lab", "research", "AI research experiments", "active"},
		{"Customer Success", "customer-success", "Customer support and success", "active"},
	}

	for _, w := range workspaces {
		data.Workspaces = append(data.Workspaces, db.Workspace{
			ID:          g.GenerateID("ws"),
			Slug:        w.slug,
			Name:        w.name,
			Description: &w.description,
			Status:      w.status,
			Settings:    ToJSON(WorkspaceSettings()),
			CreatedAt:   now.Add(-30 * 24 * time.Hour),
			UpdatedAt:   now.Add(-7 * 24 * time.Hour),
		})
	}

	// Create roles
	roles := []struct {
		name        string
		description string
		isSystem    bool
	}{
		{"superadmin", "Full system access", true},
		{"workspace_admin", "Workspace administrator", true},
		{"workspace_member", "Standard workspace member", true},
		{"api_user", "API-only access", true},
		{"viewer", "Read-only access", true},
		{"developer", "Development team member", false},
		{"analyst", "Data analyst with reporting access", false},
		{"support", "Customer support representative", false},
	}

	for _, r := range roles {
		data.Roles = append(data.Roles, db.Role{
			ID:          g.GenerateID("role"),
			WorkspaceID: nil, // Global roles
			Name:        r.name,
			Description: &r.description,
			IsSystem:    r.isSystem,
			CreatedAt:   now.Add(-60 * 24 * time.Hour),
			UpdatedAt:   now.Add(-60 * 24 * time.Hour),
		})
	}

	// Create permissions
	permissions := []struct {
		name         string
		description  string
		resourceType string
		action       string
	}{
		{"workspace:read", "Read workspace information", "workspace", "read"},
		{"workspace:write", "Modify workspace settings", "workspace", "write"},
		{"workspace:delete", "Delete workspace", "workspace", "delete"},
		{"user:read", "Read user information", "user", "read"},
		{"user:write", "Create and modify users", "user", "write"},
		{"user:delete", "Delete users", "user", "delete"},
		{"provider:read", "Read provider configuration", "provider", "read"},
		{"provider:write", "Configure providers", "provider", "write"},
		{"provider:delete", "Delete providers", "provider", "delete"},
		{"apikey:read", "Read API keys", "apikey", "read"},
		{"apikey:write", "Create and modify API keys", "apikey", "write"},
		{"apikey:delete", "Revoke API keys", "apikey", "delete"},
		{"quota:read", "Read quota information", "quota", "read"},
		{"quota:write", "Manage quotas", "quota", "write"},
		{"usage:read", "Read usage records", "usage", "read"},
		{"usage:export", "Export usage data", "usage", "export"},
		{"controlroom:read", "View control rooms", "controlroom", "read"},
		{"controlroom:write", "Manage control rooms", "controlroom", "write"},
		{"admin:access", "Access admin endpoints", "admin", "access"},
	}

	for _, p := range permissions {
		data.Permissions = append(data.Permissions, db.Permission{
			ID:           g.GenerateID("perm"),
			Name:         p.name,
			Description:  p.description,
			ResourceType: p.resourceType,
			Action:       p.action,
		})
	}

	// Assign permissions to roles
	for _, role := range data.Roles {
		var permCount int
		switch role.Name {
		case "superadmin":
			permCount = len(data.Permissions)
		case "workspace_admin":
			permCount = len(data.Permissions) - 2 // All except superadmin only
		case "workspace_member":
			permCount = 8 // Read access to most things
		case "api_user":
			permCount = 4 // Minimal API access
		case "viewer":
			permCount = 6 // Read-only
		default:
			permCount = 5
		}

		for i := 0; i < permCount && i < len(data.Permissions); i++ {
			data.RolePermissions = append(data.RolePermissions, db.RolePermission{
				RoleID:       role.ID,
				PermissionID: data.Permissions[i].ID,
			})
		}
	}

	// Create users for each workspace
	userEmails := [][]string{
		{"alice@acme.com", "bob@acme.com", "charlie@acme.com"},
		{"dave@acme.com", "eve@acme.com"},
		{"frank@acme.com", "grace@acme.com", "henry@acme.com"},
		{"ian@acme.com", "julia@acme.com"},
		{"karen@acme.com"},
	}

	userNames := [][]string{
		{"Alice Johnson", "Bob Smith", "Charlie Brown"},
		{"Dave Wilson", "Eve Davis"},
		{"Frank Miller", "Grace Lee", "Henry Taylor"},
		{"Ian Anderson", "Julia Martinez"},
		{"Karen White"},
	}

	for i, ws := range data.Workspaces {
		for j, email := range userEmails[i] {
			status := "active"
			if RandomBool() && j == 0 {
				status = "pending"
			}

			user := db.User{
				ID:            g.GenerateID("user"),
				WorkspaceID:   ws.ID,
				Email:         email,
				DisplayName:   &userNames[i][j],
				Status:        status,
				PasswordHash:  strPtr(g.GeneratePasswordHash()),
				LastLoginAt:   timePtr(now.Add(-time.Duration(g.RandomInt(1, 168)) * time.Hour)),
				CreatedAt:     now.Add(-30 * 24 * time.Hour),
				UpdatedAt:     now.Add(-7 * 24 * time.Hour),
			}
			data.Users = append(data.Users, user)

			// Assign role to user
			roleIndex := j % len(data.Roles)
			data.UserRoles = append(data.UserRoles, db.UserRole{
				UserID:    user.ID,
				RoleID:    data.Roles[roleIndex].ID,
				GrantedAt: now.Add(-28 * 24 * time.Hour),
			})
		}
	}

	// Create tags for each workspace
	tagCategories := []struct {
		category string
		values   []string
	}{
		{"env", []string{"production", "staging", "development", "testing"}},
		{"region", []string{"us-east", "us-west", "eu-central", "ap-south"}},
		{"team", []string{"platform", "ml", "data", "product", "infra"}},
		{"priority", []string{"critical", "high", "medium", "low"}},
		{"cost-center", []string{"r-and-d", "operations", "marketing", "sales"}},
	}

	for _, ws := range data.Workspaces {
		for _, tc := range tagCategories {
			for _, value := range tc.values {
				description := fmt.Sprintf("%s: %s", tc.category, value)
				data.Tags = append(data.Tags, db.Tag{
					ID:          g.GenerateID("tag"),
					WorkspaceID: ws.ID,
					Category:    tc.category,
					Value:       value,
					Description: &description,
					CreatedAt:   now.Add(-25 * 24 * time.Hour),
				})
			}
		}
	}

	// Create providers for each workspace
	providerConfigs := []struct {
		name         string
		slug         string
		providerType string
		baseURL      string
		status       string
		priority     int
		weight       int
	}{
		{"OpenAI Production", "openai-prod", "openai", "https://api.openai.com/v1", "active", 1, 50},
		{"Anthropic Production", "anthropic-prod", "anthropic", "https://api.anthropic.com", "active", 2, 30},
		{"Gemini Enterprise", "gemini-enterprise", "gemini", "https://generativelanguage.googleapis.com", "active", 3, 20},
		{"OpenAI Staging", "openai-staging", "openai", "https://api.openai.com/v1", "active", 4, 0},
		{"Azure OpenAI", "azure-openai", "azure", "https://acme.openai.azure.com", "active", 5, 0},
	}

	for _, ws := range data.Workspaces {
		for _, pc := range providerConfigs {
			// Skip some providers for smaller workspaces
			if ws.Slug == "customer-success" && pc.priority > 3 {
				continue
			}

			config := ProviderConfig(pc.providerType)
			data.Providers = append(data.Providers, db.Provider{
				ID:              g.GenerateID("prov"),
				WorkspaceID:     ws.ID,
				Slug:            pc.slug,
				Name:            pc.name,
				ProviderType:    pc.providerType,
				BaseURL:         pc.baseURL,
				APIKeyEncrypted: strPtr("encrypted:" + g.GenerateID("key")),
				Config:          ToJSON(config),
				Status:          pc.status,
				Priority:        pc.priority,
				Weight:          pc.weight,
				CreatedAt:       now.Add(-20 * 24 * time.Hour),
				UpdatedAt:       now.Add(-2 * 24 * time.Hour),
			})
		}
	}

	// Assign tags to providers
	for _, p := range data.Providers {
		// Find tags in the same workspace
		var wsTags []db.Tag
		for _, t := range data.Tags {
			if t.WorkspaceID == p.WorkspaceID {
				wsTags = append(wsTags, t)
			}
		}

		// Assign 2-4 random tags
		tagCount := int(g.RandomInt(2, 4))
		for i := 0; i < tagCount && i < len(wsTags); i++ {
			data.ProviderTags = append(data.ProviderTags, db.ProviderTag{
				ProviderID: p.ID,
				TagID:      wsTags[i].ID,
			})
		}
	}

	// Create API keys
	apiKeyNames := []string{"Production API", "Development API", "Testing API", "CI/CD Pipeline", "Mobile App", "Web App", "Integration"}
	for _, ws := range data.Workspaces {
		// Get users in this workspace
		var wsUsers []db.User
		for _, u := range data.Users {
			if u.WorkspaceID == ws.ID {
				wsUsers = append(wsUsers, u)
			}
		}

		for i, name := range apiKeyNames {
			if i > 3 && RandomBool() {
				continue // Skip some keys randomly
			}

			status := "active"
			if i == 2 {
				status = "inactive"
			}

			key := g.GenerateAPIKey()
			keyID := g.GenerateID("key")

			data.APIKeys = append(data.APIKeys, db.APIKey{
				ID:            keyID,
				WorkspaceID:   ws.ID,
				Name:          name,
				KeyHash:       g.HashAPIKey(key),
				KeyPreview:    key[:8] + "...",
				Status:        status,
				CreatedBy:     &wsUsers[0].ID,
				ExpiresAt:     timePtr(now.Add(365 * 24 * time.Hour)),
				LastUsedAt:    timePtr(now.Add(-time.Duration(g.RandomInt(1, 48)) * time.Hour)),
				RateLimit:     intPtr(int(g.RandomInt(100, 10000))),
				AllowedModels: []string{"*"},
				AllowedAPIs:   []string{"chat.completions", "embeddings"},
				Metadata:      ToJSON(map[string]interface{}{"description": "Auto-generated key"}),
				CreatedAt:     now.Add(-15 * 24 * time.Hour),
				UpdatedAt:     now.Add(-1 * 24 * time.Hour),
			})

			// Assign tags to API key
			for _, t := range data.Tags {
				if t.WorkspaceID == ws.ID && t.Category == "env" {
					data.APIKeyTags = append(data.APIKeyTags, db.APIKeyTag{
						APIKeyID: keyID,
						TagID:    t.ID,
					})
					break
				}
			}
		}
	}

	// Create control rooms
	for _, ws := range data.Workspaces {
		room := db.ControlRoom{
			ID:              g.GenerateID("cr"),
			WorkspaceID:     ws.ID,
			Slug:            "main",
			Name:            fmt.Sprintf("%s Operations", ws.Name),
			Description:     strPtr(fmt.Sprintf("Main control room for %s", ws.Name)),
			TagFilter:       "env:production",
			DashboardLayout: ToJSON(DashboardLayout()),
			CreatedBy:       &data.Users[0].ID,
			CreatedAt:       now.Add(-10 * 24 * time.Hour),
			UpdatedAt:       now.Add(-1 * 24 * time.Hour),
		}
		data.ControlRooms = append(data.ControlRooms, room)

		// Grant access to all workspace users
		for _, u := range data.Users {
			if u.WorkspaceID == ws.ID {
				data.ControlRoomAccess = append(data.ControlRoomAccess, db.ControlRoomAccess{
					ControlRoomID: room.ID,
					UserID:        u.ID,
					Role:          "viewer",
					GrantedBy:     &data.Users[0].ID,
					GrantedAt:     now.Add(-9 * 24 * time.Hour),
				})
			}
		}
	}

	// Create quotas
	quotaDefs := []struct {
		name        string
		quotaType   string
		period      string
		limit       int64
		scope       string
		warningThresh int
	}{
		{"Daily Token Limit", "tokens", "daily", 1000000, "workspace", 80},
		{"Monthly Cost Budget", "cost", "monthly", 500000, "workspace", 75},
		{"Requests Per Minute", "requests", "minute", 1000, "api_key", 90},
		{"Hourly Token Limit", "tokens", "hourly", 100000, "api_key", 85},
	}

	for _, ws := range data.Workspaces {
		for _, qd := range quotaDefs {
			data.Quotas = append(data.Quotas, db.Quota{
				ID:               g.GenerateID("quota"),
				WorkspaceID:      ws.ID,
				Name:             qd.name,
				Description:      strPtr(fmt.Sprintf("%s quota for %s", qd.name, ws.Name)),
				QuotaType:        qd.quotaType,
				Period:           qd.period,
				LimitValue:       qd.limit,
				Scope:            qd.scope,
				WarningThreshold: qd.warningThresh,
				CreatedAt:        now.Add(-14 * 24 * time.Hour),
				UpdatedAt:        now.Add(-3 * 24 * time.Hour),
			})
		}
	}

	// Create sample usage records
	data.UsageRecords = generateSampleUsageRecords(g, data, 100)

	return data
}

// generateProductionLikeData creates production-like data.
func generateProductionLikeData(g *Generator) *FullSeedData {
	data := generateDevelopmentData(g)

	// Add more workspaces
	now := g.now
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("Customer %d", i+1)
		data.Workspaces = append(data.Workspaces, db.Workspace{
			ID:          g.GenerateID("ws"),
			Slug:        fmt.Sprintf("customer-%d", i+1),
			Name:        name,
			Description: strPtr(fmt.Sprintf("Workspace for %s", name)),
			Status:      "active",
			Settings:    ToJSON(WorkspaceSettings()),
			CreatedAt:   now.Add(-time.Duration(g.RandomInt(30, 365)) * 24 * time.Hour),
			UpdatedAt:   now.Add(-time.Duration(g.RandomInt(1, 30)) * 24 * time.Hour),
		})
	}

	// Generate more usage records
	data.UsageRecords = generateSampleUsageRecords(g, data, 10000)

	return data
}

// generateStressTestData creates large volumes of data.
func generateStressTestData(g *Generator) *FullSeedData {
	data := &FullSeedData{}
	now := g.now

	// Create many workspaces
	for i := 0; i < 100; i++ {
		data.Workspaces = append(data.Workspaces, db.Workspace{
			ID:        g.GenerateID("ws"),
			Slug:      fmt.Sprintf("workspace-%d", i),
			Name:      fmt.Sprintf("Workspace %d", i),
			Status:    "active",
			Settings:  ToJSON(WorkspaceSettings()),
			CreatedAt: now.Add(-365 * 24 * time.Hour),
			UpdatedAt: now,
		})
	}

	// Create many users
	for _, ws := range data.Workspaces {
		for j := 0; j < 10; j++ {
			data.Users = append(data.Users, db.User{
				ID:           g.GenerateID("user"),
				WorkspaceID:  ws.ID,
				Email:        fmt.Sprintf("user%d@%s.com", j, ws.Slug),
				Status:       "active",
				PasswordHash: strPtr(g.GeneratePasswordHash()),
				CreatedAt:    now.Add(-100 * 24 * time.Hour),
				UpdatedAt:    now,
			})
		}
	}

	// Create many usage records
	data.UsageRecords = generateSampleUsageRecords(g, data, 100000)

	return data
}

// generateDemoData creates polished demo data.
func generateDemoData(g *Generator) *FullSeedData {
	data := &FullSeedData{}
	now := g.now

	// Create one beautiful demo workspace
	data.Workspaces = append(data.Workspaces, db.Workspace{
		ID:          "ws_demo_main",
		Slug:        "demo-workspace",
		Name:        "RAD Gateway Demo",
		Description: strPtr("Interactive demonstration of RAD Gateway capabilities"),
		Status:      "active",
		Settings:    ToJSON(WorkspaceSettings()),
		CreatedAt:   now.Add(-30 * 24 * time.Hour),
		UpdatedAt:   now.Add(-1 * time.Hour),
	})

	ws := data.Workspaces[0]

	// Create demo users
	demoUsers := []struct {
		name  string
		email string
		role  string
	}{
		{"Demo Admin", "admin@demo.com", "superadmin"},
		{"Demo Developer", "dev@demo.com", "workspace_admin"},
		{"Demo Analyst", "analyst@demo.com", "developer"},
	}

	for _, du := range demoUsers {
		data.Users = append(data.Users, db.User{
			ID:           g.GenerateID("user"),
			WorkspaceID:  ws.ID,
			Email:        du.email,
			DisplayName:  &du.name,
			Status:       "active",
			PasswordHash: strPtr(g.GeneratePasswordHash()),
			LastLoginAt:  timePtr(now.Add(-2 * time.Hour)),
			CreatedAt:    now.Add(-25 * 24 * time.Hour),
			UpdatedAt:    now,
		})
	}

	// Create demo providers
	demoProviders := []struct {
		name         string
		slug         string
		providerType string
		status       string
	}{
		{"OpenAI GPT-4", "openai-gpt4", "openai", "active"},
		{"Claude 3.5 Sonnet", "claude-sonnet", "anthropic", "active"},
		{"Gemini Pro", "gemini-pro", "gemini", "active"},
	}

	for _, dp := range demoProviders {
		data.Providers = append(data.Providers, db.Provider{
			ID:              g.GenerateID("prov"),
			WorkspaceID:     ws.ID,
			Slug:            dp.slug,
			Name:            dp.name,
			ProviderType:    dp.providerType,
			BaseURL:         "https://api.demo.com",
			APIKeyEncrypted: strPtr("encrypted:demo_key"),
			Config:          ToJSON(ProviderConfig(dp.providerType)),
			Status:          dp.status,
			Priority:        1,
			Weight:          100,
			CreatedAt:       now.Add(-20 * 24 * time.Hour),
			UpdatedAt:       now.Add(-1 * time.Hour),
		})
	}

	// Create demo API keys
	for i, name := range []string{"Demo Production Key", "Demo Development Key"} {
		key := g.GenerateAPIKey()
		status := "active"
		if i == 1 {
			status = "inactive"
		}

		data.APIKeys = append(data.APIKeys, db.APIKey{
			ID:          fmt.Sprintf("demo_key_%d", i),
			WorkspaceID: ws.ID,
			Name:        name,
			KeyHash:     g.HashAPIKey(key),
			KeyPreview:  key[:8] + "...",
			Status:      status,
			LastUsedAt:  timePtr(now.Add(-1 * time.Hour)),
			CreatedAt:   now.Add(-15 * 24 * time.Hour),
			UpdatedAt:   now,
		})
	}

	// Generate demo usage records
	data.UsageRecords = generateSampleUsageRecords(g, data, 500)

	return data
}

// generateMinimalData creates minimal required data.
func generateMinimalData(g *Generator) *FullSeedData {
	data := &FullSeedData{}
	now := g.now

	// One workspace
	data.Workspaces = append(data.Workspaces, db.Workspace{
		ID:        g.GenerateID("ws"),
		Slug:      "default",
		Name:      "Default Workspace",
		Status:    "active",
		Settings:  ToJSON(WorkspaceSettings()),
		CreatedAt: now,
		UpdatedAt: now,
	})

	// One admin user
	data.Users = append(data.Users, db.User{
		ID:           g.GenerateID("user"),
		WorkspaceID:  data.Workspaces[0].ID,
		Email:        "admin@localhost",
		Status:       "active",
		PasswordHash: strPtr(g.GeneratePasswordHash()),
		CreatedAt:    now,
		UpdatedAt:    now,
	})

	// One provider
	data.Providers = append(data.Providers, db.Provider{
		ID:           g.GenerateID("prov"),
		WorkspaceID:  data.Workspaces[0].ID,
		Slug:         "openai",
		Name:         "OpenAI",
		ProviderType: "openai",
		BaseURL:      "https://api.openai.com/v1",
		Config:       ToJSON(ProviderConfig("openai")),
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	})

	return data
}

// generateSampleUsageRecords creates sample usage records.
func generateSampleUsageRecords(g *Generator, data *FullSeedData, count int) []db.UsageRecord {
	var records []db.UsageRecord
	now := g.now

	apis := []string{"chat.completions", "embeddings", "completions"}
	models := []string{"gpt-4o", "gpt-4o-mini", "claude-3-5-sonnet-20241022", "gemini-1.5-pro"}
	statuses := []string{"success", "success", "success", "success", "error"} // 80% success rate

	for i := 0; i < count; i++ {
		// Pick random workspace
		ws := RandomChoice(data.Workspaces)

		// Pick random user from workspace
		var userID string
		for _, u := range data.Users {
			if u.WorkspaceID == ws.ID {
				userID = u.ID
				break
			}
		}

		// Pick random provider from workspace
		var provID string
		for _, p := range data.Providers {
			if p.WorkspaceID == ws.ID {
				provID = p.ID
				break
			}
		}

		status := RandomChoice(statuses)
		promptTokens := g.RandomInt(10, 4000)
		completionTokens := g.RandomInt(5, 2000)

		duration := int(g.RandomInt(100, 5000))
		completedAt := now.Add(-time.Duration(i*5) * time.Minute)
		startedAt := completedAt.Add(-time.Duration(duration) * time.Millisecond)

		record := db.UsageRecord{
			ID:               g.GenerateID("usage"),
			WorkspaceID:      ws.ID,
			RequestID:        g.GenerateID("req"),
			TraceID:          g.GenerateID("trace"),
			IncomingAPI:      RandomChoice(apis),
			IncomingModel:    RandomChoice(models),
			SelectedModel:    strPtr(RandomChoice(models)),
			ProviderID:       &provID,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
			CostUSD:          floatPtr(g.RandomFloat(0.001, 0.5)),
			DurationMs:       duration,
			ResponseStatus:   status,
			Attempts:         1,
			StartedAt:        startedAt,
			CompletedAt:      &completedAt,
			CreatedAt:        completedAt,
		}

		if status == "error" {
			errCode := "rate_limit_exceeded"
			errMsg := "Rate limit exceeded, retry after 60s"
			record.ErrorCode = &errCode
			record.ErrorMessage = &errMsg
		}

		records = append(records, record)
	}

	return records
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func intPtr(i int) *int {
	return &i
}

func floatPtr(f float64) *float64 {
	return &f
}
