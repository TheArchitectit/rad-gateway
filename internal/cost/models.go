// Package cost provides cost tracking and calculation for RAD Gateway.
// It calculates costs based on token usage and model-specific pricing.
package cost

import (
	"time"
)

// TokenRate represents the cost per 1K tokens for a specific token type.
type TokenRate struct {
	Per1KInputTokens      float64 `json:"per_1k_input_tokens" db:"per_1k_input_tokens"`
	Per1KOutputTokens     float64 `json:"per_1k_output_tokens" db:"per_1k_output_tokens"`
	Per1KCachedTokens     float64 `json:"per_1k_cached_tokens" db:"per_1k_cached_tokens"`
	Per1KTrainingTokens   float64 `json:"per_1k_training_tokens" db:"per_1k_training_tokens"`
}

// ModelPricing represents the pricing configuration for a specific model.
type ModelPricing struct {
	ID          string     `json:"id" db:"id"`
	Provider    string     `json:"provider" db:"provider"`
	ModelID     string     `json:"model_id" db:"model_id"`
	ModelName   string     `json:"model_name" db:"model_name"`
	TokenRate   TokenRate  `json:"token_rate" db:"token_rate"`
	PerRequest  float64    `json:"per_request" db:"per_request"` // Fixed cost per request
	Currency    string     `json:"currency" db:"currency"`
	EffectiveAt time.Time  `json:"effective_at" db:"effective_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	IsActive    bool       `json:"is_active" db:"is_active"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// CostBreakdown provides detailed cost calculation for a single request.
type CostBreakdown struct {
	PromptCost      float64 `json:"prompt_cost"`
	CompletionCost  float64 `json:"completion_cost"`
	CachedCost      float64 `json:"cached_cost"`
	RequestCost     float64 `json:"request_cost"`
	TotalCost       float64 `json:"total_cost"`
	Currency        string  `json:"currency"`
}

// CostRecord represents a calculated cost for a usage record.
type CostRecord struct {
	ID               string        `json:"id" db:"id"`
	UsageRecordID    string        `json:"usage_record_id" db:"usage_record_id"`
	WorkspaceID      string        `json:"workspace_id" db:"workspace_id"`
	ProviderID       *string       `json:"provider_id,omitempty" db:"provider_id"`
	ModelID          string        `json:"model_id" db:"model_id"`
	PromptTokens     int64         `json:"prompt_tokens" db:"prompt_tokens"`
	CompletionTokens int64         `json:"completion_tokens" db:"completion_tokens"`
	TotalTokens      int64         `json:"total_tokens" db:"total_tokens"`
	Breakdown        CostBreakdown `json:"breakdown" db:"breakdown"`
	CalculatedAt     time.Time     `json:"calculated_at" db:"calculated_at"`
	PricingVersion   string        `json:"pricing_version" db:"pricing_version"`
}

// AggregationLevel defines the time granularity for cost aggregation.
type AggregationLevel string

const (
	AggHourly  AggregationLevel = "hourly"
	AggDaily   AggregationLevel = "daily"
	AggWeekly  AggregationLevel = "weekly"
	AggMonthly AggregationLevel = "monthly"
)

// CostSummary represents aggregated cost data for a time period.
type CostSummary struct {
	WorkspaceID      string           `json:"workspace_id" db:"workspace_id"`
	PeriodStart      time.Time        `json:"period_start" db:"period_start"`
	PeriodEnd        time.Time        `json:"period_end" db:"period_end"`
	AggregationLevel AggregationLevel `json:"aggregation_level" db:"aggregation_level"`
	TotalCost        float64          `json:"total_cost" db:"total_cost"`
	PromptCost       float64          `json:"prompt_cost" db:"prompt_cost"`
	CompletionCost   float64          `json:"completion_cost" db:"completion_cost"`
	RequestCount     int64            `json:"request_count" db:"request_count"`
	TotalTokens      int64            `json:"total_tokens" db:"total_tokens"`
	PromptTokens     int64            `json:"prompt_tokens" db:"prompt_tokens"`
	CompletionTokens int64            `json:"completion_tokens" db:"completion_tokens"`
	Currency         string           `json:"currency" db:"currency"`
}

// CostByModel represents costs grouped by model.
type CostByModel struct {
	WorkspaceID      string    `json:"workspace_id" db:"workspace_id"`
	ProviderID       string    `json:"provider_id" db:"provider_id"`
	ProviderName     string    `json:"provider_name" db:"provider_name"`
	ModelID          string    `json:"model_id" db:"model_id"`
	ModelName        string    `json:"model_name" db:"model_name"`
	PeriodStart      time.Time `json:"period_start" db:"period_start"`
	PeriodEnd        time.Time `json:"period_end" db:"period_end"`
	TotalCost        float64   `json:"total_cost" db:"total_cost"`
	PromptCost       float64   `json:"prompt_cost" db:"prompt_cost"`
	CompletionCost   float64   `json:"completion_cost" db:"completion_cost"`
	RequestCount     int64     `json:"request_count" db:"request_count"`
	TotalTokens      int64     `json:"total_tokens" db:"total_tokens"`
	PromptTokens     int64     `json:"prompt_tokens" db:"prompt_tokens"`
	CompletionTokens int64     `json:"completion_tokens" db:"completion_tokens"`
	Currency         string    `json:"currency" db:"currency"`
}

// CostByProvider represents costs grouped by provider.
type CostByProvider struct {
	WorkspaceID      string    `json:"workspace_id" db:"workspace_id"`
	ProviderID       string    `json:"provider_id" db:"provider_id"`
	ProviderName     string    `json:"provider_name" db:"provider_name"`
	PeriodStart      time.Time `json:"period_start" db:"period_start"`
	PeriodEnd        time.Time `json:"period_end" db:"period_end"`
	TotalCost        float64   `json:"total_cost" db:"total_cost"`
	PromptCost       float64   `json:"prompt_cost" db:"prompt_cost"`
	CompletionCost   float64   `json:"completion_cost" db:"completion_cost"`
	RequestCount     int64     `json:"request_count" db:"request_count"`
	TotalTokens      int64     `json:"total_tokens" db:"total_tokens"`
	PromptTokens     int64     `json:"prompt_tokens" db:"prompt_tokens"`
	CompletionTokens int64     `json:"completion_tokens" db:"completion_tokens"`
	Currency         string    `json:"currency" db:"currency"`
}

// CostTimeseriesPoint represents a single point in a cost timeseries.
type CostTimeseriesPoint struct {
	Timestamp        time.Time `json:"timestamp" db:"timestamp"`
	TotalCost        float64   `json:"total_cost" db:"total_cost"`
	PromptCost       float64   `json:"prompt_cost" db:"prompt_cost"`
	CompletionCost   float64   `json:"completion_cost" db:"completion_cost"`
	RequestCount     int64     `json:"request_count" db:"request_count"`
	TotalTokens      int64     `json:"total_tokens" db:"total_tokens"`
	PromptTokens     int64     `json:"prompt_tokens" db:"prompt_tokens"`
	CompletionTokens int64     `json:"completion_tokens" db:"completion_tokens"`
}

// QueryFilter provides filtering options for cost queries.
type QueryFilter struct {
	WorkspaceID      string
	ProviderID       *string
	ModelID          *string
	APIKeyID         *string
	ControlRoomID    *string
	StartTime        time.Time
	EndTime          time.Time
	AggregationLevel AggregationLevel
	Currency         string // Filter by currency (default: USD)
}

// UncalculatedRecord represents a usage record that needs cost calculation.
type UncalculatedRecord struct {
	ID               string    `db:"id"`
	WorkspaceID      string    `db:"workspace_id"`
	ProviderID       *string   `db:"provider_id"`
	IncomingModel    string    `db:"incoming_model"`
	SelectedModel    *string   `db:"selected_model"`
	PromptTokens     int64     `db:"prompt_tokens"`
	CompletionTokens int64     `db:"completion_tokens"`
	TotalTokens      int64     `db:"total_tokens"`
	CreatedAt        time.Time `db:"created_at"`
}

// DefaultCurrency is the default currency for cost calculations.
const DefaultCurrency = "USD"

// DefaultPricing is a map of default pricing rates for common models.
// These rates are in USD per 1K tokens.
var DefaultPricing = map[string]TokenRate{
	// OpenAI models
	"gpt-4o":           {Per1KInputTokens: 0.005, Per1KOutputTokens: 0.015, Per1KCachedTokens: 0.0025},
	"gpt-4o-mini":      {Per1KInputTokens: 0.00015, Per1KOutputTokens: 0.0006, Per1KCachedTokens: 0.000075},
	"gpt-4o-mini-2024-07-18": {Per1KInputTokens: 0.00015, Per1KOutputTokens: 0.0006, Per1KCachedTokens: 0.000075},
	"gpt-4-turbo":      {Per1KInputTokens: 0.01, Per1KOutputTokens: 0.03, Per1KCachedTokens: 0.005},
	"gpt-4":            {Per1KInputTokens: 0.03, Per1KOutputTokens: 0.06, Per1KCachedTokens: 0.015},
	"gpt-3.5-turbo":    {Per1KInputTokens: 0.0005, Per1KOutputTokens: 0.0015, Per1KCachedTokens: 0.00025},
	"text-embedding-3-small": {Per1KInputTokens: 0.00002, Per1KOutputTokens: 0, Per1KCachedTokens: 0},
	"text-embedding-3-large": {Per1KInputTokens: 0.00013, Per1KOutputTokens: 0, Per1KCachedTokens: 0},
	"whisper-1":        {Per1KInputTokens: 0, Per1KOutputTokens: 0.006, Per1KCachedTokens: 0}, // Per minute
	"gpt-image-1":      {Per1KInputTokens: 0, Per1KOutputTokens: 0, Per1KCachedTokens: 0},     // Image generation

	// Anthropic models
	"claude-3-5-sonnet":          {Per1KInputTokens: 0.003, Per1KOutputTokens: 0.015, Per1KCachedTokens: 0.0003},
	"claude-3-5-sonnet-20241022": {Per1KInputTokens: 0.003, Per1KOutputTokens: 0.015, Per1KCachedTokens: 0.0003},
	"claude-3-opus":              {Per1KInputTokens: 0.015, Per1KOutputTokens: 0.075, Per1KCachedTokens: 0.0015},
	"claude-3-haiku":             {Per1KInputTokens: 0.00025, Per1KOutputTokens: 0.00125, Per1KCachedTokens: 0.00003},

	// Google Gemini models
	"gemini-1.5-flash":      {Per1KInputTokens: 0.000075, Per1KOutputTokens: 0.0003, Per1KCachedTokens: 0.00001875},
	"gemini-1.5-pro":        {Per1KInputTokens: 0.00125, Per1KOutputTokens: 0.005, Per1KCachedTokens: 0.0003125},
	"gemini-1.5-pro-latest": {Per1KInputTokens: 0.00125, Per1KOutputTokens: 0.005, Per1KCachedTokens: 0.0003125},
	"gemini-1.0-pro":        {Per1KInputTokens: 0.0005, Per1KOutputTokens: 0.0015, Per1KCachedTokens: 0},
}
