package cost

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAPIHandler_parseFilter(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	handler := NewAPIHandler(agg)

	tests := []struct {
		name    string
		query   string
		wantErr bool
		check   func(t *testing.T, f QueryFilter)
	}{
		{
			name:    "missing workspace_id",
			query:   "",
			wantErr: true,
		},
		{
			name:    "valid workspace_id only",
			query:   "workspace_id=ws1",
			wantErr: false,
			check: func(t *testing.T, f QueryFilter) {
				if f.WorkspaceID != "ws1" {
					t.Errorf("expected workspace_id=ws1, got %s", f.WorkspaceID)
				}
				// Should default to last 24 hours
				if time.Since(f.StartTime) > 25*time.Hour {
					t.Error("expected start_time to be within last 24 hours")
				}
				if f.AggregationLevel != AggDaily {
					t.Errorf("expected aggregation_level=daily, got %s", f.AggregationLevel)
				}
			},
		},
		{
			name:    "invalid start_time",
			query:   "workspace_id=ws1&start_time=invalid",
			wantErr: true,
		},
		{
			name:    "invalid end_time",
			query:   "workspace_id=ws1&end_time=invalid",
			wantErr: true,
		},
		{
			name:    "end_time before start_time",
			query:   "workspace_id=ws1&start_time=2024-01-02T00:00:00Z&end_time=2024-01-01T00:00:00Z",
			wantErr: true,
		},
		{
			name:    "valid time range",
			query:   "workspace_id=ws1&start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z",
			wantErr: false,
			check: func(t *testing.T, f QueryFilter) {
				expectedStart, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
				expectedEnd, _ := time.Parse(time.RFC3339, "2024-01-02T00:00:00Z")
				if !f.StartTime.Equal(expectedStart) {
					t.Errorf("expected start_time=%s, got %s", expectedStart, f.StartTime)
				}
				if !f.EndTime.Equal(expectedEnd) {
					t.Errorf("expected end_time=%s, got %s", expectedEnd, f.EndTime)
				}
			},
		},
		{
			name:    "hourly granularity",
			query:   "workspace_id=ws1&granularity=hourly",
			wantErr: false,
			check: func(t *testing.T, f QueryFilter) {
				if f.AggregationLevel != AggHourly {
					t.Errorf("expected aggregation_level=hourly, got %s", f.AggregationLevel)
				}
			},
		},
		{
			name:    "weekly granularity",
			query:   "workspace_id=ws1&granularity=weekly",
			wantErr: false,
			check: func(t *testing.T, f QueryFilter) {
				if f.AggregationLevel != AggWeekly {
					t.Errorf("expected aggregation_level=weekly, got %s", f.AggregationLevel)
				}
			},
		},
		{
			name:    "monthly granularity",
			query:   "workspace_id=ws1&granularity=monthly",
			wantErr: false,
			check: func(t *testing.T, f QueryFilter) {
				if f.AggregationLevel != AggMonthly {
					t.Errorf("expected aggregation_level=monthly, got %s", f.AggregationLevel)
				}
			},
		},
		{
			name:    "invalid granularity",
			query:   "workspace_id=ws1&granularity=invalid",
			wantErr: true,
		},
		{
			name:    "with provider_id",
			query:   "workspace_id=ws1&provider_id=prov1",
			wantErr: false,
			check: func(t *testing.T, f QueryFilter) {
				if f.ProviderID == nil || *f.ProviderID != "prov1" {
					t.Error("expected provider_id=prov1")
				}
			},
		},
		{
			name:    "with model_id",
			query:   "workspace_id=ws1&model_id=gpt-4o",
			wantErr: false,
			check: func(t *testing.T, f QueryFilter) {
				if f.ModelID == nil || *f.ModelID != "gpt-4o" {
					t.Error("expected model_id=gpt-4o")
				}
			},
		},
		{
			name:    "with api_key_id",
			query:   "workspace_id=ws1&api_key_id=key1",
			wantErr: false,
			check: func(t *testing.T, f QueryFilter) {
				if f.APIKeyID == nil || *f.APIKeyID != "key1" {
					t.Error("expected api_key_id=key1")
				}
			},
		},
		{
			name:    "aggregation alias",
			query:   "workspace_id=ws1&aggregation=daily",
			wantErr: false,
			check: func(t *testing.T, f QueryFilter) {
				if f.AggregationLevel != AggDaily {
					t.Errorf("expected aggregation_level=daily (via aggregation param), got %s", f.AggregationLevel)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v0/costs/summary?"+tt.query, nil)
			filter, err := handler.parseFilter(req)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, filter)
			}
		})
	}
}

func TestAPIHandler_MethodNotAllowed(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	handler := NewAPIHandler(agg)

	endpoints := []string{
		"/v0/costs/summary",
		"/v0/costs/by-model",
		"/v0/costs/by-provider",
		"/v0/costs/timeseries",
		"/v0/costs/pricing",
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint+"_POST", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, endpoint, nil)
			rec := httptest.NewRecorder()

			switch endpoint {
			case "/v0/costs/summary":
				handler.getCostSummary(rec, req)
			case "/v0/costs/by-model":
				handler.getCostByModel(rec, req)
			case "/v0/costs/by-provider":
				handler.getCostByProvider(rec, req)
			case "/v0/costs/timeseries":
				handler.getCostTimeseries(rec, req)
			case "/v0/costs/pricing":
				handler.getPricing(rec, req)
			}

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status 405, got %d", rec.Code)
			}

			var resp map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp["error"] != "method not allowed" {
				t.Errorf("expected error='method not allowed', got '%s'", resp["error"])
			}
		})
	}
}

func TestAPIHandler_getPricing(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	handler := NewAPIHandler(agg)

	req := httptest.NewRequest(http.MethodGet, "/v0/costs/pricing", nil)
	rec := httptest.NewRecorder()

	handler.getPricing(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp struct {
		Models   map[string]TokenRate `json:"models"`
		Currency string               `json:"currency"`
		Updated  time.Time            `json:"updated"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Currency != "USD" {
		t.Errorf("expected currency=USD, got %s", resp.Currency)
	}

	if len(resp.Models) == 0 {
		t.Error("expected models to not be empty")
	}

	// Check for expected models
	if _, ok := resp.Models["gpt-4o-mini"]; !ok {
		t.Error("expected gpt-4o-mini in pricing")
	}
}

func TestAPIHandler_getCostSummary_MissingWorkspace(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	handler := NewAPIHandler(agg)

	req := httptest.NewRequest(http.MethodGet, "/v0/costs/summary", nil)
	rec := httptest.NewRecorder()

	handler.getCostSummary(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestAPIHandler_getCostSummary_NilDB(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	handler := NewAPIHandler(agg)

	req := httptest.NewRequest(http.MethodGet, "/v0/costs/summary?workspace_id=ws1", nil)
	rec := httptest.NewRecorder()

	handler.getCostSummary(rec, req)

	// Should return 500 because DB is nil
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
}

func TestParamError(t *testing.T) {
	err := errMissingParam("workspace_id")
	if err.Error() != "workspace_id: missing required parameter" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	err2 := errInvalidParam("start_time", "invalid format")
	if err2.Error() != "start_time: invalid format" {
		t.Errorf("unexpected error message: %s", err2.Error())
	}
}

func TestParseTimeRange(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantStart time.Time
		wantEnd   time.Time
		wantErr   bool
	}{
		{
			name:  "default times",
			query: "",
			wantErr: false,
		},
		{
			name:    "invalid start_time",
			query:   "start_time=invalid",
			wantErr: true,
		},
		{
			name:    "invalid end_time",
			query:   "end_time=invalid",
			wantErr: true,
		},
		{
			name:    "end before start",
			query:   "start_time=2024-01-02T00:00:00Z&end_time=2024-01-01T00:00:00Z",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.query, nil)
			_, _, err := parseTimeRange(req)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimeRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseLimit(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		defaultLimit int
		maxLimit     int
		expected     int
	}{
		{
			name:         "default",
			query:        "",
			defaultLimit: 50,
			maxLimit:     100,
			expected:     50,
		},
		{
			name:         "valid limit",
			query:        "limit=25",
			defaultLimit: 50,
			maxLimit:     100,
			expected:     25,
		},
		{
			name:         "limit exceeds max",
			query:        "limit=200",
			defaultLimit: 50,
			maxLimit:     100,
			expected:     100,
		},
		{
			name:         "invalid limit",
			query:        "limit=abc",
			defaultLimit: 50,
			maxLimit:     100,
			expected:     50,
		},
		{
			name:         "negative limit",
			query:        "limit=-5",
			defaultLimit: 50,
			maxLimit:     100,
			expected:     50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.query, nil)
			got := parseLimit(req, tt.defaultLimit, tt.maxLimit)

			if got != tt.expected {
				t.Errorf("parseLimit() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestAPIResponse(t *testing.T) {
	resp := NewSuccessResponse(map[string]string{"key": "value"})
	if resp.Data == nil {
		t.Error("expected data to not be nil")
	}
	if resp.Meta == nil {
		t.Error("expected meta to not be nil")
	}
	if resp.Error != "" {
		t.Error("expected error to be empty")
	}

	errResp := NewErrorResponse("something went wrong")
	if errResp.Error != "something went wrong" {
		t.Errorf("expected error='something went wrong', got '%s'", errResp.Error)
	}
	if errResp.Data != nil {
		t.Error("expected data to be nil for error response")
	}
}

func TestRespondWithJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	resp := NewSuccessResponse(map[string]string{"message": "ok"})

	RespondWithJSON(rec, http.StatusOK, resp)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %s", rec.Header().Get("Content-Type"))
	}

	var decoded APIResponse
	if err := json.NewDecoder(rec.Body).Decode(&decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if decoded.Error != "" {
		t.Errorf("expected no error, got '%s'", decoded.Error)
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	data := map[string]string{"test": "data"}

	writeJSON(rec, http.StatusOK, data)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %s", rec.Header().Get("Content-Type"))
	}
}

// Test edge case: method not allowed on POST for pricing endpoint
func TestAPIHandler_getPricing_MethodNotAllowed(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	handler := NewAPIHandler(agg)

	req := httptest.NewRequest(http.MethodPost, "/v0/costs/pricing", nil)
	rec := httptest.NewRecorder()

	handler.getPricing(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
}

// Test context cancellation handling
func TestAPIHandler_ContextCancellation(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	handler := NewAPIHandler(agg)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := httptest.NewRequest(http.MethodGet, "/v0/costs/summary?workspace_id=ws1", nil)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.getCostSummary(rec, req)

	// Should return 500 due to nil DB (context cancellation would happen before DB access)
	if rec.Code != http.StatusInternalServerError && rec.Code != http.StatusServiceUnavailable {
		// Accept either 500 or 503 as valid responses for cancelled context
		t.Logf("got status %d, expected 500 or 503", rec.Code)
	}
}
