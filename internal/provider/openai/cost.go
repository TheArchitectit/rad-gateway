// Package openai provides cost tracking for OpenAI API usage.
// Sprint 8.1: Add cost tracking for OpenAI
package openai

import (
	"fmt"
	"strings"
	"sync"
)

// CostPerToken defines the cost per token for different OpenAI models.
// Prices are in USD per 1K tokens.
type CostPerToken struct {
	Input      float64
	Output     float64
	CachedInput float64 // For cached tokens (if applicable)
}

// ModelPricing contains pricing for all supported OpenAI models.
// Prices sourced from: https://openai.com/pricing
var ModelPricing = map[string]CostPerToken{
	// GPT-4o models
	"gpt-4o":              {Input: 0.0025, Output: 0.01},
	"gpt-4o-2024-08-06":   {Input: 0.0025, Output: 0.01},
	"gpt-4o-2024-05-13":   {Input: 0.005, Output: 0.015},

	// GPT-4o Mini models
	"gpt-4o-mini":            {Input: 0.00015, Output: 0.0006},
	"gpt-4o-mini-2024-07-18": {Input: 0.00015, Output: 0.0006},

	// GPT-4 Turbo models
	"gpt-4-turbo":           {Input: 0.01, Output: 0.03},
	"gpt-4-turbo-2024-04-09": {Input: 0.01, Output: 0.03},
	"gpt-4-turbo-preview":   {Input: 0.01, Output: 0.03},

	// GPT-4 models
	"gpt-4":           {Input: 0.03, Output: 0.06},
	"gpt-4-32k":       {Input: 0.06, Output: 0.12},
	"gpt-4-0613":      {Input: 0.03, Output: 0.06},
	"gpt-4-32k-0613":  {Input: 0.06, Output: 0.12},
	"gpt-4-1106-preview": {Input: 0.01, Output: 0.03},

	// GPT-3.5 Turbo models
	"gpt-3.5-turbo":        {Input: 0.0005, Output: 0.0015},
	"gpt-3.5-turbo-16k":    {Input: 0.003, Output: 0.004},
	"gpt-3.5-turbo-0125":   {Input: 0.0005, Output: 0.0015},
	"gpt-3.5-turbo-1106":   {Input: 0.001, Output: 0.002},

	// Embedding models
	"text-embedding-3-small": {Input: 0.00002, Output: 0},
	"text-embedding-3-large": {Input: 0.00013, Output: 0},
	"text-embedding-ada-002": {Input: 0.0001, Output: 0},
}

// CostTracker tracks usage costs for OpenAI API calls.
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
