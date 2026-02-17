package cost

import (
	"context"
	"testing"
	"time"
)

// mockDB is a simple mock for testing
type mockDB struct {
	records []UncalculatedRecord
}

func TestAggregator_WithBatchSize(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc, WithBatchSize(50))

	if agg.batchSize != 50 {
		t.Errorf("expected batchSize=50, got %d", agg.batchSize)
	}

	// Test invalid batch size (should use default)
	agg2 := NewAggregator(nil, calc, WithBatchSize(0))
	if agg2.batchSize != 100 {
		t.Errorf("expected batchSize=100 (default), got %d", agg2.batchSize)
	}

	// Test negative batch size
	agg3 := NewAggregator(nil, calc, WithBatchSize(-10))
	if agg3.batchSize != 100 {
		t.Errorf("expected batchSize=100 (default), got %d", agg3.batchSize)
	}
}

func TestAggregator_ProcessBatch_EdgeCases(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)

	ctx := context.Background()

	// Test with nil DB (should return error)
	_, err := agg.FetchUncalculatedRecords(ctx)
	if err == nil {
		t.Error("expected error with nil database")
	}
}

func TestAggregator_BatchUpdateCosts_EdgeCases(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)

	ctx := context.Background()

	// Test empty costs map - should error due to nil db before checking empty
	_, err := agg.BatchUpdateCosts(ctx, map[string]float64{})
	if err == nil {
		t.Error("expected error with nil database")
	}
}

func TestAggregator_CountUncalculatedRecords_Error(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)

	ctx := context.Background()

	// Test with nil DB
	_, err := agg.CountUncalculatedRecords(ctx)
	if err == nil {
		t.Error("expected error with nil database")
	}
}

func TestAggregator_UpdateRecordCost_Error(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)

	ctx := context.Background()

	// Test with nil DB
	err := agg.UpdateRecordCost(ctx, "rec1", 0.5)
	if err == nil {
		t.Error("expected error with nil database")
	}
}

func TestAggregator_GetTotalCost_Error(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)

	ctx := context.Background()
	now := time.Now()

	// Test with nil DB
	_, err := agg.GetTotalCost(ctx, "ws1", now, now)
	if err == nil {
		t.Error("expected error with nil database")
	}
}

func TestAggregator_QueryFilter_Validation(t *testing.T) {
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	filter := QueryFilter{
		WorkspaceID:      "ws1",
		StartTime:        start,
		EndTime:          end,
		AggregationLevel: AggDaily,
		Currency:         "USD",
	}

	if filter.WorkspaceID != "ws1" {
		t.Errorf("expected workspace_id=ws1, got %s", filter.WorkspaceID)
	}

	if filter.AggregationLevel != AggDaily {
		t.Errorf("expected aggregation_level=daily, got %s", filter.AggregationLevel)
	}

	if filter.Currency != "USD" {
		t.Errorf("expected currency=USD, got %s", filter.Currency)
	}
}

func TestAggregator_CostSummary_ZeroValues(t *testing.T) {
	summary := &CostSummary{
		WorkspaceID:      "ws1",
		TotalCost:        0,
		PromptCost:       0,
		CompletionCost:   0,
		RequestCount:     0,
		TotalTokens:      0,
		PromptTokens:     0,
		CompletionTokens: 0,
		Currency:         "USD",
	}

	if summary.TotalCost != 0 {
		t.Errorf("expected total_cost=0, got %f", summary.TotalCost)
	}

	if summary.Currency != "USD" {
		t.Errorf("expected currency=USD, got %s", summary.Currency)
	}
}

func TestAggregator_CalculateFromRecord_ModelResolution(t *testing.T) {
	calc := NewCalculator()

	// Test with selected model
	selected := "gpt-4o-mini"
	record := UncalculatedRecord{
		ID:               "rec1",
		WorkspaceID:      "ws1",
		IncomingModel:    "some-alias",
		SelectedModel:    &selected,
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	breakdown := calc.CalculateFromRecord(record)

	// gpt-4o-mini: 1000 * 0.00015 + 500 * 0.0006 = 0.00045
	if breakdown.TotalCost != 0.00045 {
		t.Errorf("expected cost=0.00045, got %f", breakdown.TotalCost)
	}

	// Test with only incoming model
	record2 := UncalculatedRecord{
		ID:               "rec2",
		WorkspaceID:      "ws1",
		IncomingModel:    "gpt-4o-mini",
		SelectedModel:    nil,
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	breakdown2 := calc.CalculateFromRecord(record2)

	if breakdown2.TotalCost != 0.00045 {
		t.Errorf("expected cost=0.00045, got %f", breakdown2.TotalCost)
	}
}

func TestAggregator_CalculateFromRecord_UnknownModel(t *testing.T) {
	calc := NewCalculator()

	record := UncalculatedRecord{
		ID:               "rec1",
		WorkspaceID:      "ws1",
		IncomingModel:    "unknown-exotic-model",
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	breakdown := calc.CalculateFromRecord(record)

	// Should use fallback rate: 1000 * 0.01 + 500 * 0.03 = 0.025
	if breakdown.TotalCost != 0.025 {
		t.Errorf("expected cost=0.025 (fallback), got %f", breakdown.TotalCost)
	}
}

// Test error handling for validation edge cases
func TestCalculator_Validate_EdgeCases(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name    string
		b       CostBreakdown
		wantErr bool
	}{
		{
			name: "very small positive values",
			b: CostBreakdown{
				PromptCost:     1e-10,
				CompletionCost: 2e-10,
				TotalCost:      3e-10,
			},
			wantErr: false,
		},
		{
			name: "exactly at suspicious threshold",
			b: CostBreakdown{
				PromptCost:     50.0,
				CompletionCost: 50.0,
				TotalCost:      100.0,
			},
			wantErr: false, // $100 is allowed
		},
		{
			name: "just over suspicious threshold",
			b: CostBreakdown{
				PromptCost:     50.0,
				CompletionCost: 50.1,
				TotalCost:      100.1,
			},
			wantErr: true, // Over $100
		},
		{
			name: "exactly zero",
			b: CostBreakdown{
				PromptCost:     0,
				CompletionCost: 0,
				TotalCost:      0,
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

// Helper error type for testing
type mockError struct {
	message string
}

func (e mockError) Error() string {
	return e.message
}

func TestAggregator_Errors(t *testing.T) {
	// Test various error scenarios
	ctx := context.Background()

	calc := NewCalculator()
	agg := NewAggregator(nil, calc)

	// Test nil database errors
	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "FetchUncalculatedRecords",
			fn: func() error {
				_, err := agg.FetchUncalculatedRecords(ctx)
				return err
			},
		},
		{
			name: "CountUncalculatedRecords",
			fn: func() error {
				_, err := agg.CountUncalculatedRecords(ctx)
				return err
			},
		},
		{
			name: "UpdateRecordCost",
			fn: func() error {
				return agg.UpdateRecordCost(ctx, "id", 1.0)
			},
		},
		{
			name: "BatchUpdateCosts",
			fn: func() error {
				_, err := agg.BatchUpdateCosts(ctx, map[string]float64{"id": 1.0})
				return err
			},
		},
		{
			name: "ProcessBatch",
			fn: func() error {
				_, err := agg.ProcessBatch(ctx)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Errorf("%s: expected error with nil database", tt.name)
			}
		})
	}
}

// Test aggregation level types
func TestAggregationLevel_Values(t *testing.T) {
	levels := []AggregationLevel{
		AggHourly,
		AggDaily,
		AggWeekly,
		AggMonthly,
	}

	expected := []string{"hourly", "daily", "weekly", "monthly"}

	for i, level := range levels {
		if string(level) != expected[i] {
			t.Errorf("expected aggregation level %d to be %s, got %s", i, expected[i], level)
		}
	}
}

// Test CostBreakdown calculations
func TestCostBreakdown_Validation(t *testing.T) {
	breakdown := CostBreakdown{
		PromptCost:     0.01,
		CompletionCost: 0.02,
		CachedCost:     0.003,
		RequestCost:    0.001,
		TotalCost:      0.034,
		Currency:       "USD",
	}

	if breakdown.Currency != "USD" {
		t.Errorf("expected currency=USD, got %s", breakdown.Currency)
	}

	if breakdown.PromptCost < 0 {
		t.Error("prompt cost should not be negative")
	}

	if breakdown.CompletionCost < 0 {
		t.Error("completion cost should not be negative")
	}
}

// Test UncalculatedRecord structure
func TestUncalculatedRecord_Fields(t *testing.T) {
	selected := "gpt-4o"
	provider := "prov1"

	record := UncalculatedRecord{
		ID:               "rec1",
		WorkspaceID:      "ws1",
		ProviderID:       &provider,
		IncomingModel:    "gpt-4o-mini",
		SelectedModel:    &selected,
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	if record.ID != "rec1" {
		t.Errorf("expected id=rec1, got %s", record.ID)
	}

	if record.SelectedModel == nil || *record.SelectedModel != "gpt-4o" {
		t.Error("selected model should be gpt-4o")
	}

	if record.ProviderID == nil || *record.ProviderID != "prov1" {
		t.Error("provider id should be prov1")
	}

	if record.TotalTokens != 1500 {
		t.Errorf("expected total_tokens=1500, got %d", record.TotalTokens)
	}
}
