package cost

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNewService(t *testing.T) {
	cfg := Config{
		DB:           nil,
		EnableWorker: false,
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	if svc == nil {
		t.Fatal("expected service to not be nil")
	}

	if svc.calculator == nil {
		t.Error("expected calculator to not be nil")
	}

	if svc.aggregator == nil {
		t.Error("expected aggregator to not be nil")
	}

	if svc.api == nil {
		t.Error("expected api handler to not be nil")
	}

	if svc.worker != nil {
		t.Error("expected worker to be nil when disabled")
	}
}

func TestNewService_WithWorker(t *testing.T) {
	cfg := Config{
		DB:              nil,
		EnableWorker:    true,
		WorkerInterval:  "1m",
		WorkerBatchSize: 50,
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	if svc.worker == nil {
		t.Error("expected worker to not be nil when enabled")
	}

	if svc.worker.interval != 1*time.Minute {
		t.Errorf("expected worker interval=1m, got %v", svc.worker.interval)
	}

	if svc.worker.batchSize != 50 {
		t.Errorf("expected worker batchSize=50, got %d", svc.worker.batchSize)
	}
}

func TestNewService_InvalidInterval(t *testing.T) {
	cfg := Config{
		DB:              nil,
		EnableWorker:    true,
		WorkerInterval:  "invalid",
		WorkerBatchSize: 100,
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Should still create service with default interval
	if svc.worker.interval != DefaultWorkerInterval {
		t.Errorf("expected default interval for invalid value, got %v", svc.worker.interval)
	}
}

func TestNewService_WithPricingOverrides(t *testing.T) {
	customRate := TokenRate{
		Per1KInputTokens:  0.5,
		Per1KOutputTokens: 1.5,
	}

	cfg := Config{
		DB:           nil,
		EnableWorker: false,
		PricingOverrides: map[string]TokenRate{
			"custom-model": customRate,
		},
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Verify custom pricing was set
	rate, ok := svc.GetModelPricing("custom-model")
	if !ok {
		t.Error("expected custom-model pricing to be set")
	}

	if rate.Per1KInputTokens != 0.5 {
		t.Errorf("expected input rate=0.5, got %f", rate.Per1KInputTokens)
	}
}

func TestService_Start(t *testing.T) {
	cfg := Config{
		DB:           nil,
		EnableWorker: false,
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()

	err = svc.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	if !svc.IsRunning() {
		t.Error("expected service to be running")
	}

	// Clean up
	if err := svc.Stop(); err != nil {
		t.Fatalf("failed to stop service: %v", err)
	}
}

func TestService_Start_AlreadyRunning(t *testing.T) {
	cfg := Config{
		DB:           nil,
		EnableWorker: false,
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()

	// Start first time
	err = svc.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	// Try to start again
	err = svc.Start(ctx)
	if err != ErrServiceAlreadyRunning {
		t.Errorf("expected ErrServiceAlreadyRunning, got %v", err)
	}

	// Clean up
	if err := svc.Stop(); err != nil {
		t.Fatalf("failed to stop service: %v", err)
	}
}

func TestService_Stop(t *testing.T) {
	cfg := Config{
		DB:           nil,
		EnableWorker: false,
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Stop before start should not error
	err = svc.Stop()
	if err != nil {
		t.Errorf("unexpected error stopping non-running service: %v", err)
	}

	ctx := context.Background()

	// Start and then stop
	err = svc.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	err = svc.Stop()
	if err != nil {
		t.Fatalf("failed to stop service: %v", err)
	}

	if svc.IsRunning() {
		t.Error("expected service to not be running after stop")
	}

	// Stop again should not error
	err = svc.Stop()
	if err != nil {
		t.Errorf("unexpected error stopping already stopped service: %v", err)
	}
}

func TestService_Getters(t *testing.T) {
	cfg := Config{
		DB:           nil,
		EnableWorker: false,
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	if svc.GetCalculator() == nil {
		t.Error("expected GetCalculator to not return nil")
	}

	if svc.GetAggregator() == nil {
		t.Error("expected GetAggregator to not return nil")
	}

	if svc.GetWorker() != nil {
		t.Error("expected GetWorker to return nil when disabled")
	}

	if svc.GetAPIHandler() == nil {
		t.Error("expected GetAPIHandler to not return nil")
	}
}

func TestService_CalculateMethods(t *testing.T) {
	cfg := Config{
		DB:           nil,
		EnableWorker: false,
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Test CalculateCost
	breakdown := svc.CalculateCost("gpt-4o-mini", 1000, 500)
	if breakdown.TotalCost != 0.00045 {
		t.Errorf("expected cost=0.00045, got %f", breakdown.TotalCost)
	}

	// Test CalculateCostFromRecord
	selected := "gpt-4o-mini"
	record := UncalculatedRecord{
		IncomingModel:    "some-alias",
		SelectedModel:    &selected,
		PromptTokens:     1000,
		CompletionTokens: 500,
	}
	breakdown2 := svc.CalculateCostFromRecord(record)
	if breakdown2.TotalCost != 0.00045 {
		t.Errorf("expected cost from record=0.00045, got %f", breakdown2.TotalCost)
	}

	// Test ValidateCost
	validBreakdown := CostBreakdown{
		PromptCost:     0.01,
		CompletionCost: 0.02,
		TotalCost:      0.03,
	}
	if err := svc.ValidateCost(validBreakdown); err != nil {
		t.Errorf("expected validation to pass, got error: %v", err)
	}

	invalidBreakdown := CostBreakdown{
		PromptCost:     0.01,
		CompletionCost: 0.02,
		TotalCost:      -0.03,
	}
	if err := svc.ValidateCost(invalidBreakdown); err == nil {
		t.Error("expected validation to fail for negative cost")
	}

	// Test EstimateCost
	estimate := svc.EstimateCost("gpt-4o-mini", 1000, 2000)
	if estimate.PromptCost == 0 {
		t.Error("expected prompt cost to be non-zero in estimate")
	}
}

func TestService_PricingMethods(t *testing.T) {
	cfg := Config{
		DB:           nil,
		EnableWorker: false,
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Test SetModelPricing and GetModelPricing
	svc.SetModelPricing("test-model", TokenRate{
		Per1KInputTokens:  0.1,
		Per1KOutputTokens: 0.2,
	})

	rate, ok := svc.GetModelPricing("test-model")
	if !ok {
		t.Error("expected to find test-model pricing")
	}
	if rate.Per1KInputTokens != 0.1 {
		t.Errorf("expected input rate=0.1, got %f", rate.Per1KInputTokens)
	}

	// Test ListKnownModels
	models := svc.ListKnownModels()
	if len(models) == 0 {
		t.Error("expected known models list to not be empty")
	}

	// Check that our custom model is in the list
	found := false
	for _, m := range models {
		if m == "test-model" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected test-model to be in known models")
	}
}

func TestService_QueryMethods(t *testing.T) {
	cfg := Config{
		DB:           nil,
		EnableWorker: false,
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()

	// Test ForceCalculation (nil DB should return error)
	_, err = svc.ForceCalculation(ctx)
	if err == nil {
		t.Error("expected error with nil database")
	}

	// Test GetCostSummary (nil DB should return error)
	filter := QueryFilter{
		WorkspaceID:      "ws1",
		StartTime:        time.Now().Add(-24 * time.Hour),
		EndTime:          time.Now(),
		AggregationLevel: AggDaily,
	}
	_, err = svc.GetCostSummary(ctx, filter)
	if err == nil {
		t.Error("expected error with nil database")
	}

	// Test GetCostByModel
	_, err = svc.GetCostByModel(ctx, filter)
	if err == nil {
		t.Error("expected error with nil database")
	}

	// Test GetCostByProvider
	_, err = svc.GetCostByProvider(ctx, filter)
	if err == nil {
		t.Error("expected error with nil database")
	}

	// Test GetCostTimeseries
	_, err = svc.GetCostTimeseries(ctx, filter)
	if err == nil {
		t.Error("expected error with nil database")
	}

	// Test GetPendingCount
	_, err = svc.GetPendingCount(ctx)
	if err == nil {
		t.Error("expected error with nil database")
	}
}

func TestService_GetStats(t *testing.T) {
	cfg := Config{
		DB:           nil,
		EnableWorker: false,
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	stats := svc.GetStats()

	if stats.Running {
		t.Error("expected stats.Running=false")
	}

	// Start service
	ctx := context.Background()
	err = svc.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	stats = svc.GetStats()
	if !stats.Running {
		t.Error("expected stats.Running=true after start")
	}

	// Clean up
	if err := svc.Stop(); err != nil {
		t.Fatalf("failed to stop service: %v", err)
	}
}

func TestServiceError(t *testing.T) {
	err := ErrServiceAlreadyRunning
	if err.Error() != "service already running" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	err2 := ErrServiceNotRunning
	if err2.Error() != "service not running" {
		t.Errorf("unexpected error message: %s", err2.Error())
	}
}

func TestService_MuxInterface(t *testing.T) {
	cfg := Config{
		DB:           nil,
		EnableWorker: false,
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Create a mock mux
	mockMux := &mockMux{}

	// Register routes
	svc.RegisterRoutes(mockMux)

	// Check that routes were registered
	if len(mockMux.routes) == 0 {
		t.Error("expected routes to be registered")
	}

	// Check for expected routes
	expectedRoutes := []string{
		"/v0/costs/summary",
		"/v0/costs/by-model",
		"/v0/costs/by-provider",
		"/v0/costs/timeseries",
		"/v0/costs/pricing",
	}

	for _, route := range expectedRoutes {
		if _, ok := mockMux.routes[route]; !ok {
			t.Errorf("expected route %s to be registered", route)
		}
	}
}

// mockMux implements the HTTPMux interface for testing
type mockMux struct {
	routes map[string]bool
}

func (m *mockMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	if m.routes == nil {
		m.routes = make(map[string]bool)
	}
	m.routes[pattern] = true
}

var _ HTTPMux = (*mockMux)(nil)

// Benchmark service creation
func BenchmarkNewService(b *testing.B) {
	cfg := Config{
		DB:           nil,
		EnableWorker: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewService(cfg)
	}
}

// Benchmark service start/stop
func BenchmarkService_StartStop(b *testing.B) {
	cfg := Config{
		DB:           nil,
		EnableWorker: false,
	}

	svc, _ := NewService(cfg)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = svc.Start(ctx)
		_ = svc.Stop()
	}
}
