// Package anthropic provides cost tracking for Anthropic API usage.
// Sprint 8.2: Add cost tracking for Anthropic
package anthropic

import (
	"fmt"
	"strings"
	"sync"
)

// CostPerToken defines the cost per token for different Anthropic models.
// Prices are in USD per 1K tokens.
type CostPerToken struct {
	Input  float64
	Output float64
}

// ModelPricing contains pricing for all supported Anthropic models.
// Prices sourced from: https://www.anthropic.com/pricing
var ModelPricing = map[string]CostPerToken{
	// Claude 3.5 models
	"claude-3-5-sonnet-20241022": {Input: 0.003, Output: 0.015},
	"claude-3-5-sonnet-20240620": {Input: 0.003, Output: 0.015},
	"claude-3-5-sonnet-latest":   {Input: 0.003, Output: 0.015},

	// Claude 3.5 Haiku
	"claude-3-5-haiku-20241022": {Input: 0.001, Output: 0.005},
	"claude-3-5-haiku-latest":   {Input: 0.001, Output: 0.005},

	// Claude 3 Opus
	"claude-3-opus-20240229": {Input: 0.015, Output: 0.075},
	"claude-3-opus-latest":   {Input: 0.015, Output: 0.075},

	// Claude 3 Sonnet
	"claude-3-sonnet-20240229": {Input: 0.003, Output: 0.015},

	// Claude 3 Haiku
	"claude-3-haiku-20240307": {Input: 0.00025, Output: 0.00125},

	// Legacy Claude 2 models
	"claude-2.1": {Input: 0.008, Output: 0.024},
	"claude-2.0": {Input: 0.008, Output: 0.024},

	// Claude Instant
	"claude-instant-1.2": {Input: 0.0008, Output: 0.0024},
}

// CostTracker tracks usage costs for Anthropic API calls.
type CostTracker struct {
	mu         sync.RWMutex
	totalCost  float64
	modelCosts map[string]*ModelCost
}

// ModelCost tracks cost for a specific model.
type ModelCost struct {
	Model           string  `json:"model"`
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	TotalTokens     int     `json:"total_tokens"`
	InputCost       float64 `json:"input_cost"`
	OutputCost      float64 `json:"output_cost"`
	TotalCost       float64 `json:"total_cost"`
	RequestCount    int     `json:"request_count"`
}

// NewCostTracker creates a new cost tracker.
func NewCostTracker() *CostTracker {
	return &CostTracker{
		modelCosts: make(map[string]*ModelCost),
	}
}

// CalculateCost calculates the cost for a given model and token usage.
func CalculateCost(model string, promptTokens, completionTokens int) (float64, error) {
	pricing, ok := ModelPricing[model]
	if !ok {
		// Try to find a matching base model
		basePricing := findBaseModelPricing(model)
		if basePricing == nil {
			return 0, fmt.Errorf("unknown model: %s", model)
		}
		pricing = *basePricing
	}

	inputCost := float64(promptTokens) * pricing.Input / 1000
	outputCost := float64(completionTokens) * pricing.Output / 1000

	return inputCost + outputCost, nil
}

// findBaseModelPricing attempts to find pricing for a model variant.
func findBaseModelPricing(model string) *CostPerToken {
	// Check for date/version suffixes and try base model
	if idx := strings.Index(model, "-20"); idx > 0 {
		baseModel := model[:idx]
		if pricing, ok := ModelPricing[baseModel]; ok {
			return &pricing
		}
	}

	// Check for snapshot suffixes
	if strings.Contains(model, ":") {
		baseModel := strings.Split(model, ":")[0]
		if pricing, ok := ModelPricing[baseModel]; ok {
			return &pricing
		}
	}

	// Map common aliases
	aliases := map[string]string{
		"claude-sonnet":   "claude-3-5-sonnet-20241022",
		"claude-haiku":    "claude-3-5-haiku-20241022",
		"claude-opus":     "claude-3-opus-20240229",
		"claude":          "claude-3-5-sonnet-20241022",
	}

	if aliasedModel, ok := aliases[model]; ok {
		if pricing, ok := ModelPricing[aliasedModel]; ok {
			return &pricing
		}
	}

	return nil
}

// RecordUsage records usage and updates cost tracking.
func (ct *CostTracker) RecordUsage(model string, promptTokens, completionTokens int) (float64, error) {
	cost, err := CalculateCost(model, promptTokens, completionTokens)
	if err != nil {
		return 0, err
	}

	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.totalCost += cost

	mc, exists := ct.modelCosts[model]
	if !exists {
		mc = &ModelCost{Model: model}
		ct.modelCosts[model] = mc
	}

	mc.InputTokens += promptTokens
	mc.OutputTokens += completionTokens
	mc.TotalTokens += promptTokens + completionTokens
	mc.InputCost += float64(promptTokens) * ModelPricing[model].Input / 1000
	mc.OutputCost += float64(completionTokens) * ModelPricing[model].Output / 1000
	mc.TotalCost += cost
	mc.RequestCount++

	return cost, nil
}

// GetTotalCost returns the total cost tracked.
func (ct *CostTracker) GetTotalCost() float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.totalCost
}

// GetModelCosts returns cost breakdown by model.
func (ct *CostTracker) GetModelCosts() map[string]*ModelCost {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	// Return a copy
	result := make(map[string]*ModelCost, len(ct.modelCosts))
	for k, v := range ct.modelCosts {
		copy := *v
		result[k] = &copy
	}
	return result
}

// Reset resets the cost tracker.
func (ct *CostTracker) Reset() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.totalCost = 0
	ct.modelCosts = make(map[string]*ModelCost)
}

// GetPricing returns pricing for a specific model.
func GetPricing(model string) (*CostPerToken, error) {
	pricing, ok := ModelPricing[model]
	if !ok {
		basePricing := findBaseModelPricing(model)
		if basePricing == nil {
			return nil, fmt.Errorf("pricing not found for model: %s", model)
		}
		return basePricing, nil
	}
	return &pricing, nil
}

// IsKnownModel checks if a model is in our pricing table.
func IsKnownModel(model string) bool {
	_, ok := ModelPricing[model]
	if ok {
		return true
	}
	return findBaseModelPricing(model) != nil
}
