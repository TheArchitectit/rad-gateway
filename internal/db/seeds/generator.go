// Package seeds provides seed data generation for RAD Gateway.
// These utilities create realistic test data for development and testing.
package seeds

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"time"

	"radgateway/internal/db"
)

// Generator creates seed data for the database.
type Generator struct {
	now time.Time
}

// NewGenerator creates a new seed data generator.
func NewGenerator() *Generator {
	return &Generator{
		now: time.Now().UTC(),
	}
}

// SetTime sets the base time for generated data (useful for reproducible seeds).
func (g *Generator) SetTime(t time.Time) {
	g.now = t.UTC()
}

// GenerateID creates a random ID with a prefix.
func (g *Generator) GenerateID(prefix string) string {
	b := make([]byte, 12)
	rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

// GenerateAPIKey creates a random API key.
func (g *Generator) GenerateAPIKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "rad_" + hex.EncodeToString(b)
}

// HashAPIKey creates a hash of an API key for storage.
func (g *Generator) HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// GeneratePasswordHash creates a dummy password hash.
func (g *Generator) GeneratePasswordHash() string {
	// In production, use proper bcrypt hashing
	return "$2a$10$dummy.hash.for.seed.data.only"
}

// RandomInt returns a random int between min and max.
func (g *Generator) RandomInt(min, max int64) int64 {
	n, _ := rand.Int(rand.Reader, big.NewInt(max-min+1))
	return n.Int64() + min
}

// RandomFloat returns a random float between min and max.
func (g *Generator) RandomFloat(min, max float64) float64 {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return min + (max-min)*float64(n.Int64())/1000000.0
}

// RandomChoice returns a random item from a slice.
func RandomChoice[T any](items []T) T {
	if len(items) == 0 {
		var zero T
		return zero
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(items))))
	return items[n.Int64()]
}

// RandomBool returns a random boolean.
func RandomBool() bool {
	n, _ := rand.Int(rand.Reader, big.NewInt(2))
	return n.Int64() == 1
}

// TimeInPast returns a time in the past by the given duration.
func (g *Generator) TimeInPast(d time.Duration) time.Time {
	return g.now.Add(-d)
}

// TimeInFuture returns a time in the future by the given duration.
func (g *Generator) TimeInFuture(d time.Duration) time.Time {
	return g.now.Add(d)
}

// ToJSON converts a map to JSON bytes.
func ToJSON(m map[string]interface{}) []byte {
	b, _ := json.Marshal(m)
	return b
}

// ProviderConfig returns a default config for a provider type.
func ProviderConfig(providerType string) map[string]interface{} {
	switch providerType {
	case "openai":
		return map[string]interface{}{
			"default_model":       "gpt-4o",
			"available_models":    []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"},
			"max_tokens":          4096,
			"temperature_default": 0.7,
			"timeout_seconds":     60,
			"retry_policy": map[string]interface{}{
				"max_retries": 3,
				"backoff_ms":  1000,
			},
		}
	case "anthropic":
		return map[string]interface{}{
			"default_model":       "claude-3-5-sonnet-20241022",
			"available_models":    []string{"claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022", "claude-3-opus-20240229"},
			"max_tokens":          4096,
			"temperature_default": 0.7,
			"timeout_seconds":     60,
			"retry_policy": map[string]interface{}{
				"max_retries": 3,
				"backoff_ms":  1000,
			},
		}
	case "gemini":
		return map[string]interface{}{
			"default_model":       "gemini-1.5-pro",
			"available_models":    []string{"gemini-1.5-pro", "gemini-1.5-flash", "gemini-1.0-pro"},
			"max_tokens":          4096,
			"temperature_default": 0.7,
			"timeout_seconds":     60,
			"retry_policy": map[string]interface{}{
				"max_retries": 3,
				"backoff_ms":  1000,
			},
		}
	case "azure":
		return map[string]interface{}{
			"default_model":       "gpt-4",
			"available_models":    []string{"gpt-4", "gpt-4-turbo", "gpt-35-turbo"},
			"max_tokens":          4096,
			"temperature_default": 0.7,
			"timeout_seconds":     60,
			"api_version":         "2024-02-01",
		}
	default:
		return map[string]interface{}{
			"default_model":       "default",
			"available_models":    []string{"default"},
			"max_tokens":          4096,
			"temperature_default": 0.7,
		}
	}
}

// WorkspaceSettings returns default workspace settings.
func WorkspaceSettings() map[string]interface{} {
	return map[string]interface{}{
		"features": map[string]bool{
			"streaming_enabled":    true,
			"cost_tracking":        true,
			"quota_enforcement":    true,
			"audit_logging":        true,
			"custom_models":        true,
			"provider_health_checks": true,
		},
		"rate_limits": map[string]interface{}{
			"requests_per_minute": 1000,
			"tokens_per_minute":   100000,
		},
		"retention_days": 90,
		"alert_thresholds": map[string]float64{
			"cost_daily_usd":     100.0,
			"error_rate_percent": 5.0,
		},
	}
}

// DashboardLayout returns a default control room dashboard layout.
func DashboardLayout() map[string]interface{} {
	return map[string]interface{}{
		"widgets": []map[string]interface{}{
			{
				"type":   "usage_chart",
				"title":  "Token Usage",
				"config": map[string]interface{}{"period": "24h", "aggregation": "hourly"},
				"position": map[string]int{"x": 0, "y": 0, "w": 6, "h": 4},
			},
			{
				"type":   "cost_chart",
				"title":  "Cost Overview",
				"config": map[string]interface{}{"period": "7d", "currency": "USD"},
				"position": map[string]int{"x": 6, "y": 0, "w": 6, "h": 4},
			},
			{
				"type":   "provider_health",
				"title":  "Provider Status",
				"config": map[string]interface{}{"refresh_interval": 30},
				"position": map[string]int{"x": 0, "y": 4, "w": 4, "h": 3},
			},
			{
				"type":   "quota_status",
				"title":  "Quota Usage",
				"config": map[string]interface{}{"show_warnings": true},
				"position": map[string]int{"x": 4, "y": 4, "w": 4, "h": 3},
			},
			{
				"type":   "recent_requests",
				"title":  "Recent Requests",
				"config": map[string]interface{}{"limit": 10},
				"position": map[string]int{"x": 8, "y": 4, "w": 4, "h": 3},
			},
		},
		"layout_version": "1.0",
		"theme":          "dark",
	}
}

// FullSeedData contains all seed data for the application.
type FullSeedData struct {
	Workspaces       []db.Workspace
	Users            []db.User
	Roles            []db.Role
	Permissions      []db.Permission
	UserRoles        []db.UserRole
	RolePermissions  []db.RolePermission
	Tags             []db.Tag
	Providers        []db.Provider
	ProviderTags     []db.ProviderTag
	APIKeys          []db.APIKey
	APIKeyTags       []db.APIKeyTag
	ControlRooms     []db.ControlRoom
	ControlRoomAccess []db.ControlRoomAccess
	Quotas           []db.Quota
	QuotaAssignments []db.QuotaAssignment
	UsageRecords     []db.UsageRecord
	TraceEvents      []db.TraceEvent
}
