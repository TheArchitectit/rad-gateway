package cost

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"log/slog"

	"radgateway/internal/logger"
)

// Calculator computes costs based on token usage and model pricing.
type Calculator struct {
	pricing    map[string]TokenRate
	pricingMu  sync.RWMutex
	log        *slog.Logger
	fallbackRate TokenRate // Used when model pricing is not found
}

// CalculatorOption configures the Calculator.
type CalculatorOption func(*Calculator)

// WithFallbackRate sets a fallback rate for unknown models.
func WithFallbackRate(rate TokenRate) CalculatorOption {
	return func(c *Calculator) {
		c.fallbackRate = rate
	}
}

// WithPricingOverrides allows setting custom pricing for models.
func WithPricingOverrides(pricing map[string]TokenRate) CalculatorOption {
	return func(c *Calculator) {
		c.pricingMu.Lock()
		defer c.pricingMu.Unlock()
		for model, rate := range pricing {
			c.pricing[model] = rate
		}
	}
}

// NewCalculator creates a new cost calculator with default pricing.
func NewCalculator(opts ...CalculatorOption) *Calculator {
	// Copy default pricing
	pricing := make(map[string]TokenRate)
	for k, v := range DefaultPricing {
		pricing[k] = v
	}

	c := &Calculator{
		pricing: pricing,
		log:     logger.WithComponent("cost_calculator"),
		fallbackRate: TokenRate{
			Per1KInputTokens:  0.01,  // 1 cent per 1K input tokens
			Per1KOutputTokens: 0.03,  // 3 cents per 1K output tokens
			Per1KCachedTokens: 0.005, // 0.5 cents per 1K cached tokens
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Calculate computes the cost for a single request based on token usage.
func (c *Calculator) Calculate(model string, promptTokens, completionTokens int64) CostBreakdown {
	if model == "" {
		c.log.Warn("cost calculation: empty model name, using fallback rate")
		model = "unknown"
	}

	// Get pricing for the model
	rate := c.getRate(model)

	// Calculate costs
	// Cost = (tokens / 1000) * rate
	promptCost := float64(promptTokens) / 1000.0 * rate.Per1KInputTokens
	completionCost := float64(completionTokens) / 1000.0 * rate.Per1KOutputTokens
	cachedCost := 0.0 // Could be calculated if we track cache hits

	totalCost := promptCost + completionCost + cachedCost

	// Round to 6 decimal places to avoid floating point precision issues
	promptCost = round(promptCost, 6)
	completionCost = round(completionCost, 6)
	cachedCost = round(cachedCost, 6)
	totalCost = round(totalCost, 6)

	return CostBreakdown{
		PromptCost:     promptCost,
		CompletionCost: completionCost,
		CachedCost:     cachedCost,
		RequestCost:    0.0, // No per-request fee by default
		TotalCost:      totalCost,
		Currency:       DefaultCurrency,
	}
}

// CalculateFromRecord computes cost from an UncalculatedRecord.
func (c *Calculator) CalculateFromRecord(record UncalculatedRecord) CostBreakdown {
	model := c.resolveModelName(record)
	return c.Calculate(model, record.PromptTokens, record.CompletionTokens)
}

// resolveModelName determines the effective model name for pricing.
// It tries selected_model first, then incoming_model.
func (c *Calculator) resolveModelName(record UncalculatedRecord) string {
	if record.SelectedModel != nil && *record.SelectedModel != "" {
		return c.normalizeModelName(*record.SelectedModel)
	}
	return c.normalizeModelName(record.IncomingModel)
}

// normalizeModelName normalizes a model name for pricing lookup.
// Handles version suffixes, provider prefixes, and common aliases.
func (c *Calculator) normalizeModelName(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))

	// Remove common provider prefixes
	model = strings.TrimPrefix(model, "openai/")
	model = strings.TrimPrefix(model, "anthropic/")
	model = strings.TrimPrefix(model, "google/")
	model = strings.TrimPrefix(model, "azure/")

	// Remove date/version suffixes for lookup (e.g., gpt-4o-2024-08-06 -> gpt-4o)
	// Keep known dated versions that have different pricing
	if !c.hasExactPricing(model) {
		// Try without date suffix
		simpleName := c.stripVersionSuffix(model)
		if c.hasExactPricing(simpleName) {
			return simpleName
		}
	}

	return model
}

// stripVersionSuffix removes date-like suffixes from model names.
func (c *Calculator) stripVersionSuffix(model string) string {
	// Common patterns: -YYYY-MM-DD, -latest, -preview, etc.
	suffixes := []string{
		"-latest",
		"-preview",
		"-stable",
	}

	for _, suffix := range suffixes {
		if idx := strings.LastIndex(model, suffix); idx > 0 {
			return model[:idx]
		}
	}

	// Remove date suffixes like -2024-08-06
	// Look for patterns like -2024- or -2025-
	for i := len(model) - 1; i >= 0; i-- {
		if model[i] == '-' && i+4 < len(model) {
			// Check if looks like a year (20xx)
			if len(model) > i+4 && model[i+1:i+3] == "20" {
				return model[:i]
			}
		}
	}

	return model
}

// hasExactPricing checks if we have exact pricing for a model.
func (c *Calculator) hasExactPricing(model string) bool {
	c.pricingMu.RLock()
	defer c.pricingMu.RUnlock()
	_, ok := c.pricing[model]
	return ok
}

// getRate retrieves the pricing rate for a model.
// Falls back to the fallback rate if not found.
func (c *Calculator) getRate(model string) TokenRate {
	normalizedModel := c.normalizeModelName(model)

	c.pricingMu.RLock()
	rate, ok := c.pricing[normalizedModel]
	c.pricingMu.RUnlock()

	if ok {
		return rate
	}

	// Log unknown model for monitoring
	c.log.Warn("pricing not found for model, using fallback rate",
		"model", model,
		"normalized", normalizedModel,
		"fallback_input_rate", c.fallbackRate.Per1KInputTokens,
		"fallback_output_rate", c.fallbackRate.Per1KOutputTokens,
	)

	return c.fallbackRate
}

// SetPricing updates or adds pricing for a specific model.
func (c *Calculator) SetPricing(model string, rate TokenRate) {
	c.pricingMu.Lock()
	defer c.pricingMu.Unlock()
	c.pricing[strings.ToLower(model)] = rate
	c.log.Info("updated pricing for model", "model", model,
		"input_rate", rate.Per1KInputTokens,
		"output_rate", rate.Per1KOutputTokens)
}

// GetPricing retrieves the current pricing for a model.
func (c *Calculator) GetPricing(model string) (TokenRate, bool) {
	c.pricingMu.RLock()
	defer c.pricingMu.RUnlock()
	rate, ok := c.pricing[strings.ToLower(model)]
	return rate, ok
}

// ListKnownModels returns a list of models with known pricing.
func (c *Calculator) ListKnownModels() []string {
	c.pricingMu.RLock()
	defer c.pricingMu.RUnlock()

	models := make([]string, 0, len(c.pricing))
	for model := range c.pricing {
		models = append(models, model)
	}
	return models
}

// Validate checks if a cost calculation is reasonable.
// Returns an error if the cost seems suspiciously high or calculation appears invalid.
func (c *Calculator) Validate(breakdown CostBreakdown) error {
	// Check for negative costs (should never happen)
	if breakdown.TotalCost < 0 {
		return fmt.Errorf("negative total cost: %f", breakdown.TotalCost)
	}
	if breakdown.PromptCost < 0 {
		return fmt.Errorf("negative prompt cost: %f", breakdown.PromptCost)
	}
	if breakdown.CompletionCost < 0 {
		return fmt.Errorf("negative completion cost: %f", breakdown.CompletionCost)
	}

	// Check for suspiciously high costs (>$100 in a single request)
	if breakdown.TotalCost > 100.0 {
		return fmt.Errorf("suspiciously high cost: $%.2f, possible calculation error", breakdown.TotalCost)
	}

	// Verify total matches sum of components (with small tolerance for floating point)
	sum := round(breakdown.PromptCost+breakdown.CompletionCost+breakdown.CachedCost, 6)
	if math.Abs(sum-breakdown.TotalCost) > 0.000001 {
		return fmt.Errorf("cost mismatch: sum=%.6f, total=%.6f", sum, breakdown.TotalCost)
	}

	return nil
}

// EstimateCost provides a cost estimate for a planned request.
// Useful for showing users expected costs before sending a request.
func (c *Calculator) EstimateCost(model string, estimatedPromptTokens, estimatedMaxCompletionTokens int64) CostBreakdown {
	rate := c.getRate(model)

	// For estimates, use the maximum possible completion tokens
	promptCost := float64(estimatedPromptTokens) / 1000.0 * rate.Per1KInputTokens
	completionCost := float64(estimatedMaxCompletionTokens) / 1000.0 * rate.Per1KOutputTokens
	totalCost := promptCost + completionCost

	return CostBreakdown{
		PromptCost:     round(promptCost, 6),
		CompletionCost: round(completionCost, 6),
		CachedCost:     0.0,
		RequestCost:    0.0,
		TotalCost:      round(totalCost, 6),
		Currency:       DefaultCurrency,
	}
}

// round rounds a float64 to the specified number of decimal places.
func round(val float64, places int) float64 {
	factor := math.Pow(10, float64(places))
	return math.Round(val*factor) / factor
}
