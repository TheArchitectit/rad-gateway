// Package seeds provides test fixtures for RAD Gateway.
package seeds

import (
	"radgateway/internal/db"
	"time"
)

// Fixtures provides pre-defined test fixtures for unit tests.
type Fixtures struct {
	Workspaces    map[string]*db.Workspace
	Users         map[string]*db.User
	Roles         map[string]*db.Role
	Providers     map[string]*db.Provider
	APIKeys       map[string]*db.APIKey
	Tags          map[string]*db.Tag
	ControlRooms  map[string]*db.ControlRoom
	Quotas        map[string]*db.Quota
	UsageRecords  map[string]*db.UsageRecord
}

// NewFixtures creates a new fixtures instance with all test data.
func NewFixtures() *Fixtures {
	f := &Fixtures{
		Workspaces:   make(map[string]*db.Workspace),
		Users:        make(map[string]*db.User),
		Roles:        make(map[string]*db.Role),
		Providers:    make(map[string]*db.Provider),
		APIKeys:      make(map[string]*db.APIKey),
		Tags:         make(map[string]*db.Tag),
		ControlRooms: make(map[string]*db.ControlRoom),
		Quotas:       make(map[string]*db.Quota),
		UsageRecords: make(map[string]*db.UsageRecord),
	}

	f.initWorkspaces()
	f.initRoles()
	f.initUsers()
	f.initTags()
	f.initProviders()
	f.initAPIKeys()
	f.initControlRooms()
	f.initQuotas()
	f.initUsageRecords()

	return f
}

// initWorkspaces creates test workspaces.
func (f *Fixtures) initWorkspaces() {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	f.Workspaces["acme"] = &db.Workspace{
		ID:          "ws_acme_001",
		Slug:        "acme-corp",
		Name:        "Acme Corporation",
		Description: strPtr("Primary test workspace"),
		Status:      "active",
		Settings:    ToJSON(WorkspaceSettings()),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	f.Workspaces["empty"] = &db.Workspace{
		ID:        "ws_empty_001",
		Slug:      "empty-workspace",
		Name:      "Empty Workspace",
		Status:    "active",
		Settings:  ToJSON(WorkspaceSettings()),
		CreatedAt: now,
		UpdatedAt: now,
	}

	f.Workspaces["suspended"] = &db.Workspace{
		ID:          "ws_suspended_001",
		Slug:        "suspended-ws",
		Name:        "Suspended Workspace",
		Description: strPtr("Workspace for testing suspended state"),
		Status:      "suspended",
		Settings:    ToJSON(WorkspaceSettings()),
		CreatedAt:   now,
		UpdatedAt:   now.Add(-24 * time.Hour),
	}
}

// initRoles creates test roles.
func (f *Fixtures) initRoles() {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	f.Roles["superadmin"] = &db.Role{
		ID:          "role_superadmin",
		WorkspaceID: nil,
		Name:        "superadmin",
		Description: strPtr("Full system access"),
		IsSystem:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	f.Roles["admin"] = &db.Role{
		ID:          "role_admin",
		WorkspaceID: nil,
		Name:        "workspace_admin",
		Description: strPtr("Workspace administrator"),
		IsSystem:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	f.Roles["member"] = &db.Role{
		ID:          "role_member",
		WorkspaceID: nil,
		Name:        "workspace_member",
		Description: strPtr("Standard member"),
		IsSystem:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// initUsers creates test users.
func (f *Fixtures) initUsers() {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	acmeID := f.Workspaces["acme"].ID

	f.Users["alice"] = &db.User{
		ID:            "user_alice_001",
		WorkspaceID:   acmeID,
		Email:         "alice@acme.com",
		DisplayName:   strPtr("Alice Smith"),
		Status:        "active",
		PasswordHash:  strPtr("$2a$10$hashedpassword"),
		LastLoginAt:   timePtr(now.Add(-2 * time.Hour)),
		CreatedAt:     now.Add(-30 * 24 * time.Hour),
		UpdatedAt:     now,
	}

	f.Users["bob"] = &db.User{
		ID:           "user_bob_001",
		WorkspaceID:  acmeID,
		Email:        "bob@acme.com",
		DisplayName:  strPtr("Bob Jones"),
		Status:       "active",
		PasswordHash: strPtr("$2a$10$hashedpassword"),
		CreatedAt:    now.Add(-25 * 24 * time.Hour),
		UpdatedAt:    now,
	}

	f.Users["charlie"] = &db.User{
		ID:           "user_charlie_001",
		WorkspaceID:  acmeID,
		Email:        "charlie@acme.com",
		DisplayName:  strPtr("Charlie Brown"),
		Status:       "pending",
		PasswordHash: strPtr("$2a$10$hashedpassword"),
		CreatedAt:    now.Add(-5 * 24 * time.Hour),
		UpdatedAt:    now,
	}

	f.Users["disabled"] = &db.User{
		ID:            "user_disabled_001",
		WorkspaceID:   acmeID,
		Email:         "disabled@acme.com",
		DisplayName:   strPtr("Disabled User"),
		Status:        "disabled",
		PasswordHash:  strPtr("$2a$10$hashedpassword"),
		LastLoginAt:   timePtr(now.Add(-30 * 24 * time.Hour)),
		CreatedAt:     now.Add(-60 * 24 * time.Hour),
		UpdatedAt:     now.Add(-1 * 24 * time.Hour),
	}
}

// initTags creates test tags.
func (f *Fixtures) initTags() {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	acmeID := f.Workspaces["acme"].ID

	f.Tags["prod"] = &db.Tag{
		ID:          "tag_prod_001",
		WorkspaceID: acmeID,
		Category:    "env",
		Value:       "production",
		Description: strPtr("Production environment"),
		CreatedAt:   now,
	}

	f.Tags["staging"] = &db.Tag{
		ID:          "tag_staging_001",
		WorkspaceID: acmeID,
		Category:    "env",
		Value:       "staging",
		Description: strPtr("Staging environment"),
		CreatedAt:   now,
	}

	f.Tags["platform"] = &db.Tag{
		ID:          "tag_platform_001",
		WorkspaceID: acmeID,
		Category:    "team",
		Value:       "platform",
		Description: strPtr("Platform team resources"),
		CreatedAt:   now,
	}

	f.Tags["critical"] = &db.Tag{
		ID:          "tag_critical_001",
		WorkspaceID: acmeID,
		Category:    "priority",
		Value:       "critical",
		Description: strPtr("Critical priority"),
		CreatedAt:   now,
	}
}

// initProviders creates test providers.
func (f *Fixtures) initProviders() {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	acmeID := f.Workspaces["acme"].ID

	f.Providers["openai"] = &db.Provider{
		ID:              "prov_openai_001",
		WorkspaceID:     acmeID,
		Slug:            "openai-prod",
		Name:            "OpenAI Production",
		ProviderType:    "openai",
		BaseURL:         "https://api.openai.com/v1",
		APIKeyEncrypted: strPtr("encrypted:sk-abc123"),
		Config:          ToJSON(ProviderConfig("openai")),
		Status:          "active",
		Priority:        1,
		Weight:          100,
		CreatedAt:       now.Add(-20 * 24 * time.Hour),
		UpdatedAt:       now,
	}

	f.Providers["anthropic"] = &db.Provider{
		ID:              "prov_anthropic_001",
		WorkspaceID:     acmeID,
		Slug:            "anthropic-prod",
		Name:            "Anthropic Claude",
		ProviderType:    "anthropic",
		BaseURL:         "https://api.anthropic.com",
		APIKeyEncrypted: strPtr("encrypted:sk-ant-xyz"),
		Config:          ToJSON(ProviderConfig("anthropic")),
		Status:          "active",
		Priority:        2,
		Weight:          50,
		CreatedAt:       now.Add(-15 * 24 * time.Hour),
		UpdatedAt:       now,
	}

	f.Providers["gemini"] = &db.Provider{
		ID:              "prov_gemini_001",
		WorkspaceID:     acmeID,
		Slug:            "gemini-enterprise",
		Name:            "Google Gemini",
		ProviderType:    "gemini",
		BaseURL:         "https://generativelanguage.googleapis.com",
		APIKeyEncrypted: strPtr("encrypted:gem-key"),
		Config:          ToJSON(ProviderConfig("gemini")),
		Status:          "active",
		Priority:        3,
		Weight:            0,
		CreatedAt:       now.Add(-10 * 24 * time.Hour),
		UpdatedAt:       now,
	}

	f.Providers["disabled"] = &db.Provider{
		ID:              "prov_disabled_001",
		WorkspaceID:     acmeID,
		Slug:            "disabled-provider",
		Name:            "Disabled Provider",
		ProviderType:    "openai",
		BaseURL:         "https://api.disabled.com",
		APIKeyEncrypted: strPtr("encrypted:disabled"),
		Config:          ToJSON(ProviderConfig("openai")),
		Status:          "disabled",
		Priority:        4,
		Weight:            0,
		CreatedAt:       now.Add(-5 * 24 * time.Hour),
		UpdatedAt:       now.Add(-1 * 24 * time.Hour),
	}
}

// initAPIKeys creates test API keys.
func (f *Fixtures) initAPIKeys() {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	acmeID := f.Workspaces["acme"].ID
	aliceID := f.Users["alice"].ID

	f.APIKeys["prod"] = &db.APIKey{
		ID:            "key_prod_001",
		WorkspaceID:   acmeID,
		Name:          "Production API Key",
		KeyHash:       "a665a45920422f9d417e4867efdc6fb8e9f1c3c8f6e6e6e6e6e6e6e6e6e6e6e6e",
		KeyPreview:    "rad_abc1...",
		Status:        "active",
		CreatedBy:     &aliceID,
		ExpiresAt:     timePtr(now.Add(365 * 24 * time.Hour)),
		LastUsedAt:    timePtr(now.Add(-2 * time.Hour)),
		RateLimit:     intPtr(10000),
		AllowedModels: []string{"*"},
		AllowedAPIs:   []string{"chat.completions", "embeddings"},
		Metadata:      ToJSON(map[string]interface{}{"team": "platform"}),
		CreatedAt:     now.Add(-30 * 24 * time.Hour),
		UpdatedAt:     now,
	}

	f.APIKeys["dev"] = &db.APIKey{
		ID:            "key_dev_001",
		WorkspaceID:   acmeID,
		Name:          "Development API Key",
		KeyHash:       "b665a45920422f9d417e4867efdc6fb8e9f1c3c8f6e6e6e6e6e6e6e6e6e6e6e6e",
		KeyPreview:    "rad_def2...",
		Status:        "active",
		CreatedBy:     &aliceID,
		ExpiresAt:     timePtr(now.Add(180 * 24 * time.Hour)),
		LastUsedAt:    timePtr(now.Add(-5 * time.Minute)),
		RateLimit:     intPtr(1000),
		AllowedModels: []string{"gpt-3.5-turbo", "gpt-4o-mini"},
		AllowedAPIs:   []string{"chat.completions"},
		Metadata:      ToJSON(map[string]interface{}{"team": "development"}),
		CreatedAt:     now.Add(-15 * 24 * time.Hour),
		UpdatedAt:     now,
	}

	f.APIKeys["expired"] = &db.APIKey{
		ID:          "key_expired_001",
		WorkspaceID: acmeID,
		Name:        "Expired API Key",
		KeyHash:     "c665a45920422f9d417e4867efdc6fb8e9f1c3c8f6e6e6e6e6e6e6e6e6e6e6e6e",
		KeyPreview:  "rad_exp3...",
		Status:      "expired",
		CreatedBy:   &aliceID,
		ExpiresAt:   timePtr(now.Add(-1 * 24 * time.Hour)),
		CreatedAt:   now.Add(-90 * 24 * time.Hour),
		UpdatedAt:   now.Add(-1 * 24 * time.Hour),
	}
}

// initControlRooms creates test control rooms.
func (f *Fixtures) initControlRooms() {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	acmeID := f.Workspaces["acme"].ID
	aliceID := f.Users["alice"].ID

	f.ControlRooms["main"] = &db.ControlRoom{
		ID:              "cr_main_001",
		WorkspaceID:     acmeID,
		Slug:            "main-operations",
		Name:            "Main Operations Center",
		Description:     strPtr("Primary monitoring dashboard"),
		TagFilter:       "env:production",
		DashboardLayout: ToJSON(DashboardLayout()),
		CreatedBy:       &aliceID,
		CreatedAt:       now.Add(-10 * 24 * time.Hour),
		UpdatedAt:       now,
	}

	f.ControlRooms["platform"] = &db.ControlRoom{
		ID:              "cr_platform_001",
		WorkspaceID:     acmeID,
		Slug:            "platform-team",
		Name:            "Platform Team Dashboard",
		Description:     strPtr("Platform team specific view"),
		TagFilter:       "team:platform",
		DashboardLayout: ToJSON(DashboardLayout()),
		CreatedBy:       &aliceID,
		CreatedAt:       now.Add(-5 * 24 * time.Hour),
		UpdatedAt:       now,
	}
}

// initQuotas creates test quotas.
func (f *Fixtures) initQuotas() {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	acmeID := f.Workspaces["acme"].ID

	f.Quotas["daily_tokens"] = &db.Quota{
		ID:               "quota_daily_tokens",
		WorkspaceID:      acmeID,
		Name:             "Daily Token Limit",
		Description:      strPtr("Maximum tokens per day"),
		QuotaType:        "tokens",
		Period:           "daily",
		LimitValue:       1000000,
		Scope:            "workspace",
		WarningThreshold: 80,
		CreatedAt:        now.Add(-20 * 24 * time.Hour),
		UpdatedAt:        now,
	}

	f.Quotas["monthly_cost"] = &db.Quota{
		ID:               "quota_monthly_cost",
		WorkspaceID:      acmeID,
		Name:             "Monthly Cost Budget",
		Description:      strPtr("Maximum monthly spend"),
		QuotaType:        "cost",
		Period:           "monthly",
		LimitValue:       500000, // $5,000.00 in cents
		Scope:            "workspace",
		WarningThreshold: 75,
		CreatedAt:        now.Add(-20 * 24 * time.Hour),
		UpdatedAt:        now,
	}

	f.Quotas["rpm"] = &db.Quota{
		ID:               "quota_rpm",
		WorkspaceID:      acmeID,
		Name:             "Requests Per Minute",
		Description:      strPtr("Rate limit for requests per minute"),
		QuotaType:        "requests",
		Period:           "minute",
		LimitValue:       1000,
		Scope:            "api_key",
		WarningThreshold: 90,
		CreatedAt:        now.Add(-15 * 24 * time.Hour),
		UpdatedAt:        now,
	}
}

// initUsageRecords creates test usage records.
func (f *Fixtures) initUsageRecords() {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	acmeID := f.Workspaces["acme"].ID
	prodKeyID := f.APIKeys["prod"].ID
	openaiID := f.Providers["openai"].ID

	f.UsageRecords["success_1"] = &db.UsageRecord{
		ID:               "usage_success_001",
		WorkspaceID:      acmeID,
		RequestID:        "req_abc123",
		TraceID:          "trace_xyz789",
		APIKeyID:         &prodKeyID,
		IncomingAPI:      "chat.completions",
		IncomingModel:    "gpt-4o",
		SelectedModel:    strPtr("gpt-4o"),
		ProviderID:       &openaiID,
		PromptTokens:     150,
		CompletionTokens: 450,
		TotalTokens:      600,
		CostUSD:          floatPtr(0.015),
		DurationMs:       1250,
		ResponseStatus:   "success",
		Attempts:         1,
		StartedAt:        now.Add(-5 * time.Minute),
		CompletedAt:      timePtr(now.Add(-3 * time.Minute)),
		CreatedAt:        now.Add(-3 * time.Minute),
	}

	f.UsageRecords["error_1"] = &db.UsageRecord{
		ID:               "usage_error_001",
		WorkspaceID:      acmeID,
		RequestID:        "req_error456",
		TraceID:          "trace_error789",
		APIKeyID:         &prodKeyID,
		IncomingAPI:      "chat.completions",
		IncomingModel:    "gpt-4o",
		SelectedModel:    strPtr("gpt-4o"),
		ProviderID:       &openaiID,
		PromptTokens:     50,
		CompletionTokens: 0,
		TotalTokens:      50,
		CostUSD:          floatPtr(0.0),
		DurationMs:       500,
		ResponseStatus:   "error",
		ErrorCode:        strPtr("rate_limit_exceeded"),
		ErrorMessage:     strPtr("Rate limit exceeded. Please retry after 60s"),
		Attempts:         1,
		StartedAt:        now.Add(-10 * time.Minute),
		CompletedAt:      timePtr(now.Add(-9 * time.Minute)),
		CreatedAt:        now.Add(-9 * time.Minute),
	}
}

// UserRoles returns user-role assignments for testing.
func (f *Fixtures) UserRoles() []db.UserRole {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	return []db.UserRole{
		{UserID: f.Users["alice"].ID, RoleID: f.Roles["superadmin"].ID, GrantedAt: now},
		{UserID: f.Users["bob"].ID, RoleID: f.Roles["admin"].ID, GrantedAt: now},
		{UserID: f.Users["charlie"].ID, RoleID: f.Roles["member"].ID, GrantedAt: now},
	}
}

// ProviderTags returns provider-tag assignments for testing.
func (f *Fixtures) ProviderTags() []db.ProviderTag {
	return []db.ProviderTag{
		{ProviderID: f.Providers["openai"].ID, TagID: f.Tags["prod"].ID},
		{ProviderID: f.Providers["openai"].ID, TagID: f.Tags["critical"].ID},
		{ProviderID: f.Providers["anthropic"].ID, TagID: f.Tags["prod"].ID},
	}
}

// APIKeyTags returns API key-tag assignments for testing.
func (f *Fixtures) APIKeyTags() []db.APIKeyTag {
	return []db.APIKeyTag{
		{APIKeyID: f.APIKeys["prod"].ID, TagID: f.Tags["prod"].ID},
		{APIKeyID: f.APIKeys["dev"].ID, TagID: f.Tags["staging"].ID},
	}
}
