package cost

import (
	"math"
	"testing"
)

func TestCalculator_Calculate(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name             string
		model            string
		promptTokens     int64
		completionTokens int64
		wantPromptCost   float64
		wantOutputCost   float64
		wantTotalCost    float64
	}{
		{
			name:             "GPT-4o mini - simple request",
			model:            "gpt-4o-mini",
			promptTokens:     1000,
			completionTokens: 500,
			wantPromptCost:   0.00015,
			wantOutputCost:   0.0003,
			wantTotalCost:    0.00045,
		},
		{
			name:             "GPT-4o - standard request",
			model:            "gpt-4o",
			promptTokens:     1000,
			completionTokens: 1000,
			wantPromptCost:   0.005,
			wantOutputCost:   0.015,
			wantTotalCost:    0.02,
		},
		{
			name:             "Claude 3.5 Sonnet",
			model:            "claude-3-5-sonnet",
			promptTokens:     10000,
			completionTokens: 5000,
			wantPromptCost:   0.03,
			wantOutputCost:   0.075,
			wantTotalCost:    0.105,
		},
		{
			name:             "Gemini 1.5 Flash",
			model:            "gemini-1.5-flash",
			promptTokens:     1000,
			completionTokens: 500,
			wantPromptCost:   0.000075,
			wantOutputCost:   0.00015,
			wantTotalCost:    0.000225,
		},
		{
			name:             "Embeddings - small",
			model:            "text-embedding-3-small",
			promptTokens:     1000,
			completionTokens: 0,
			wantPromptCost:   0.00002,
			wantOutputCost:   0,
			wantTotalCost:    0.00002,
		},
		{
			name:             "Large token counts",
			model:            "gpt-4o-mini",
			promptTokens:     1000000, // 1M tokens
			completionTokens: 500000,  // 500K tokens
			wantPromptCost:   0.15,
			wantOutputCost:   0.3,
			wantTotalCost:    0.45,
		},
		{
			name:             "Zero tokens",
			model:            "gpt-4o-mini",
			promptTokens:     0,
			completionTokens: 0,
			wantPromptCost:   0,
			wantOutputCost:   0,
			wantTotalCost:    0,
		},
		{
			name:             "Empty model uses fallback",
			model:            "",
			promptTokens:     1000,
			completionTokens: 500,
			wantPromptCost:   0.01,
			wantOutputCost:   0.015,
			wantTotalCost:    0.025,
		},
		{
			name:             "Unknown model uses fallback",
			model:            "unknown-model-v1",
			promptTokens:     1000,
			completionTokens: 500,
			wantPromptCost:   0.01,
			wantOutputCost:   0.015,
			wantTotalCost:    0.025,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calc.Calculate(tt.model, tt.promptTokens, tt.completionTokens)

			const epsilon = 0.0000001
			if math.Abs(got.PromptCost-tt.wantPromptCost) > epsilon {
				t.Errorf("PromptCost = %.8f, want %.8f", got.PromptCost, tt.wantPromptCost)
			}
			if math.Abs(got.CompletionCost-tt.wantOutputCost) > epsilon {
				t.Errorf("CompletionCost = %.8f, want %.8f", got.CompletionCost, tt.wantOutputCost)
			}
			if math.Abs(got.TotalCost-tt.wantTotalCost) > epsilon {
				t.Errorf("TotalCost = %.8f, want %.8f", got.TotalCost, tt.wantTotalCost)
			}
			if got.Currency != "USD" {
				t.Errorf("Currency = %s, want USD", got.Currency)
			}
		})
	}
}

func TestCalculator_normalizeModelName(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		input    string
		expected string
	}{
		{"gpt-4o", "gpt-4o"},
		{"gpt-4o-2024-08-06", "gpt-4o"},              // Date suffix stripped
		{"openai/gpt-4o", "gpt-4o"},                  // Provider prefix stripped
		{"anthropic/claude-3-5-sonnet", "claude-3-5-sonnet"},
		{"claude-3-5-sonnet-latest", "claude-3-5-sonnet"},
		{"GPT-4O-MINI", "gpt-4o-mini"},               // Case normalized
		{"  gpt-4o-mini  ", "gpt-4o-mini"},          // Whitespace trimmed
		{"gpt-4o-preview", "gpt-4o"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := calc.normalizeModelName(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeModelName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCalculator_SetPricing(t *testing.T) {
	calc := NewCalculator()

	// Set custom pricing
	customRate := TokenRate{
		Per1KInputTokens:  0.5,
		Per1KOutputTokens: 1.5,
	}
	calc.SetPricing("custom-model", customRate)

	// Verify the pricing is used
	got := calc.Calculate("custom-model", 1000, 1000)

	const epsilon = 0.0000001
	if math.Abs(got.PromptCost-0.5) > epsilon {
		t.Errorf("PromptCost = %.8f, want 0.5", got.PromptCost)
	}
	if math.Abs(got.CompletionCost-1.5) > epsilon {
		t.Errorf("CompletionCost = %.8f, want 1.5", got.CompletionCost)
	}
}

func TestCalculator_Validate(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name    string
		b       CostBreakdown
		wantErr bool
	}{
		{
			name: "valid calculation",
			b: CostBreakdown{
				PromptCost:     0.01,
				CompletionCost: 0.02,
				CachedCost:     0.003,
				TotalCost:      0.033,
			},
			wantErr: false,
		},
		{
			name: "negative total cost",
			b: CostBreakdown{
				PromptCost:     0.01,
				CompletionCost: 0.02,
				TotalCost:      -0.01,
			},
			wantErr: true,
		},
		{
			name: "negative prompt cost",
			b: CostBreakdown{
				PromptCost:     -0.01,
				CompletionCost: 0.02,
				TotalCost:      0.01,
			},
			wantErr: true,
		},
		{
			name: "suspiciously high cost",
			b: CostBreakdown{
				PromptCost:     50,
				CompletionCost: 51,
				TotalCost:      101,
			},
			wantErr: true,
		},
		{
			name: "mismatched total",
			b: CostBreakdown{
				PromptCost:     0.01,
				CompletionCost: 0.02,
				CachedCost:     0.003,
				TotalCost:      0.1, // Should be 0.033
			},
			wantErr: true,
		},
		{
			name: "zero cost (valid)",
			b: CostBreakdown{
				PromptCost:     0,
				CompletionCost: 0,
				TotalCost:      0,
			},
			wantErr: false,
		},
		{
			name: "very small values (valid)",
			b: CostBreakdown{
				PromptCost:     0.000001,
				CompletionCost: 0.000002,
				TotalCost:      0.000003,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := calc.Validate(tt.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCalculator_EstimateCost(t *testing.T) {
	calc := NewCalculator()

	got := calc.EstimateCost("gpt-4o-mini", 1000, 2000)

	// Should calculate using max estimated tokens
	const epsilon = 0.0000001
	if math.Abs(got.PromptCost-0.00015) > epsilon {
		t.Errorf("PromptCost = %.8f, want 0.00015", got.PromptCost)
	}
	if math.Abs(got.CompletionCost-0.0012) > epsilon { // 2000 * 0.0006
		t.Errorf("CompletionCost = %.8f, want 0.0012", got.CompletionCost)
	}
}

func TestCalculator_ListKnownModels(t *testing.T) {
	calc := NewCalculator()

	models := calc.ListKnownModels()
	if len(models) == 0 {
		t.Error("ListKnownModels() returned empty list")
	}

	// Check some expected models
	hasGPT4o := false
	hasClaude := false
	for _, m := range models {
		if m == "gpt-4o" {
			hasGPT4o = true
		}
		if m == "claude-3-5-sonnet" {
			hasClaude = true
		}
	}

	if !hasGPT4o {
		t.Error("ListKnownModels() missing gpt-4o")
	}
	if !hasClaude {
		t.Error("ListKnownModels() missing claude-3-5-sonnet")
	}
}

func TestCalculator_CalculateFromRecord(t *testing.T) {
	calc := NewCalculator()

	selectedModel := "gpt-4o-mini"
	record := UncalculatedRecord{
		IncomingModel:    "some-alias",
		SelectedModel:    &selectedModel,
		PromptTokens:     1000,
		CompletionTokens: 500,
	}

	got := calc.CalculateFromRecord(record)

	// Should use selected_model when available
	const epsilon = 0.0000001
	if math.Abs(got.TotalCost-0.00045) > epsilon {
		t.Errorf("TotalCost = %.8f, want 0.00045", got.TotalCost)
	}
}

func TestCalculator_CalculateFromRecord_NoSelectedModel(t *testing.T) {
	calc := NewCalculator()

	record := UncalculatedRecord{
		IncomingModel:    "gpt-4o-mini",
		SelectedModel:    nil,
		PromptTokens:     1000,
		CompletionTokens: 500,
	}

	got := calc.CalculateFromRecord(record)

	// Should fall back to incoming_model
	const epsilon = 0.0000001
	if math.Abs(got.TotalCost-0.00045) > epsilon {
		t.Errorf("TotalCost = %.8f, want 0.00045", got.TotalCost)
	}
}

func BenchmarkCalculator_Calculate(b *testing.B) {
	calc := NewCalculator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Calculate("gpt-4o-mini", 1000, 500)
	}
}

func BenchmarkCalculator_normalizeModelName(b *testing.B) {
	calc := NewCalculator()
	model := "openai/gpt-4o-2024-08-06-preview"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.normalizeModelName(model)
	}
}
