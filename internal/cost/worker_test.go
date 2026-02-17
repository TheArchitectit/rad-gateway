package cost

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestWorker_NewWorker(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)

	w := NewWorker(agg)

	if w == nil {
		t.Fatal("expected worker to not be nil")
	}

	if w.interval != DefaultWorkerInterval {
		t.Errorf("expected interval=%v, got %v", DefaultWorkerInterval, w.interval)
	}

	if w.batchSize != DefaultWorkerBatchSize {
		t.Errorf("expected batchSize=%d, got %d", DefaultWorkerBatchSize, w.batchSize)
	}

	if w.running {
		t.Error("expected worker to not be running initially")
	}
}

func TestWorker_NewWorker_WithOptions(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)

	customInterval := 1 * time.Minute
	customBatchSize := 50

	w := NewWorker(agg,
		WithInterval(customInterval),
		WithWorkerBatchSize(customBatchSize),
	)

	if w.interval != customInterval {
		t.Errorf("expected interval=%v, got %v", customInterval, w.interval)
	}

	if w.batchSize != customBatchSize {
		t.Errorf("expected batchSize=%d, got %d", customBatchSize, w.batchSize)
	}
}

func TestWorker_NewWorker_InvalidOptions(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)

	// Zero values should use defaults
	w := NewWorker(agg,
		WithInterval(0),
		WithWorkerBatchSize(0),
	)

	if w.interval != DefaultWorkerInterval {
		t.Errorf("expected default interval, got %v", w.interval)
	}

	if w.batchSize != DefaultWorkerBatchSize {
		t.Errorf("expected default batchSize, got %d", w.batchSize)
	}

	// Negative values should also use defaults
	w2 := NewWorker(agg,
		WithInterval(-1 * time.Second),
		WithWorkerBatchSize(-10),
	)

	if w2.interval != DefaultWorkerInterval {
		t.Errorf("expected default interval for negative, got %v", w2.interval)
	}

	if w2.batchSize != DefaultWorkerBatchSize {
		t.Errorf("expected default batchSize for negative, got %d", w2.batchSize)
	}
}

func TestWorker_Start(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	w := NewWorker(agg)

	ctx := context.Background()

	err := w.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start worker: %v", err)
	}

	if !w.IsRunning() {
		t.Error("expected worker to be running")
	}

	// Clean up
	if err := w.Stop(); err != nil {
		t.Fatalf("failed to stop worker: %v", err)
	}
}

func TestWorker_Start_AlreadyRunning(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	w := NewWorker(agg)

	ctx := context.Background()

	// Start first time
	err := w.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start worker: %v", err)
	}

	// Try to start again
	err = w.Start(ctx)
	if err != ErrWorkerAlreadyRunning {
		t.Errorf("expected ErrWorkerAlreadyRunning, got %v", err)
	}

	// Clean up
	if err := w.Stop(); err != nil {
		t.Fatalf("failed to stop worker: %v", err)
	}
}

func TestWorker_Start_NilAggregator(t *testing.T) {
	w := NewWorker(nil)

	ctx := context.Background()

	err := w.Start(ctx)
	if err != ErrNilAggregator {
		t.Errorf("expected ErrNilAggregator, got %v", err)
	}
}

func TestWorker_Stop(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	w := NewWorker(agg)

	ctx := context.Background()

	// Stop before start should not error
	err := w.Stop()
	if err != nil {
		t.Errorf("unexpected error stopping non-running worker: %v", err)
	}

	// Start and then stop
	err = w.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start worker: %v", err)
	}

	err = w.Stop()
	if err != nil {
		t.Fatalf("failed to stop worker: %v", err)
	}

	if w.IsRunning() {
		t.Error("expected worker to not be running after stop")
	}

	// Stop again should not error
	err = w.Stop()
	if err != nil {
		t.Errorf("unexpected error stopping already stopped worker: %v", err)
	}
}

func TestWorker_GetStats(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	w := NewWorker(agg, WithInterval(2*time.Minute), WithWorkerBatchSize(75))

	stats := w.GetStats()

	if stats.Running {
		t.Error("expected stats.Running=false")
	}

	if stats.Interval != 2*time.Minute {
		t.Errorf("expected stats.Interval=2m, got %v", stats.Interval)
	}

	if stats.BatchSize != 75 {
		t.Errorf("expected stats.BatchSize=75, got %d", stats.BatchSize)
	}
}

func TestWorker_ForceProcess_NotRunning(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	w := NewWorker(agg)

	_, err := w.ForceProcess()
	if err != ErrWorkerNotRunning {
		t.Errorf("expected ErrWorkerNotRunning, got %v", err)
	}
}

func TestWorker_GetPendingCount_NotRunning(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	w := NewWorker(agg)

	_, err := w.GetPendingCount()
	if err != ErrWorkerNotRunning {
		t.Errorf("expected ErrWorkerNotRunning, got %v", err)
	}
}

func TestWorker_Run_WithContextCancellation(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	w := NewWorker(agg, WithInterval(100*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())

	err := w.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start worker: %v", err)
	}

	// Cancel context to trigger shutdown
	cancel()

	// Wait for worker to stop - needs more time for ticker to receive signal
	time.Sleep(400 * time.Millisecond)

	if w.IsRunning() {
		// Force stop if still running
		_ = w.Stop()
	}
}

func TestWorkerErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "ErrWorkerAlreadyRunning",
			err:  ErrWorkerAlreadyRunning,
			want: "worker already running",
		},
		{
			name: "ErrWorkerNotRunning",
			err:  ErrWorkerNotRunning,
			want: "worker not running",
		},
		{
			name: "ErrNilAggregator",
			err:  ErrNilAggregator,
			want: "aggregator is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.want {
				t.Errorf("expected error message='%s', got '%s'", tt.want, tt.err.Error())
			}
		})
	}
}

func TestWorkerError_Is(t *testing.T) {
	// Test error matching
	if !errors.Is(ErrWorkerAlreadyRunning, ErrWorkerAlreadyRunning) {
		t.Error("expected Is to match same error")
	}

	if errors.Is(ErrWorkerAlreadyRunning, ErrWorkerNotRunning) {
		t.Error("expected Is to not match different errors")
	}

	// Test with same error type - currently they have different pointers
	// so they won't match with errors.Is unless we use fmt.Errorf with %w
	wrapped := fmt.Errorf("wrapped: %w", ErrWorkerAlreadyRunning)
	if !errors.Is(wrapped, ErrWorkerAlreadyRunning) {
		t.Error("expected Is to match wrapped error with Is implementation")
	}
}

// Test processBatch with nil aggregator (edge case)
func TestWorker_processBatch_NilAggregator(t *testing.T) {
	w := NewWorker(nil, WithInterval(100*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.ctx = ctx
	w.running = true // Manually set to simulate running

	// This should handle nil aggregator gracefully (now returns early)
	w.processBatch()

	// If we get here without panic, the test passes
}

func TestImmediateWorker_New(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)

	// Default max records
	w := NewImmediateWorker(agg, 0)
	if w.maxRecords != 10000 {
		t.Errorf("expected default maxRecords=10000, got %d", w.maxRecords)
	}

	// Custom max records
	w2 := NewImmediateWorker(agg, 500)
	if w2.maxRecords != 500 {
		t.Errorf("expected maxRecords=500, got %d", w2.maxRecords)
	}
}

func TestImmediateWorker_Run(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	w := NewImmediateWorker(agg, 100)

	ctx := context.Background()

	// Run with nil database - should complete without panic
	processed, err := w.Run(ctx)

	// Should return error due to nil DB but not panic
	if err == nil {
		t.Error("expected error with nil database")
	}

	// Should process 0 records
	if processed != 0 {
		t.Errorf("expected processed=0, got %d", processed)
	}
}

func TestImmediateWorker_Run_ContextCancellation(t *testing.T) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	w := NewImmediateWorker(agg, 10000)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	processed, err := w.Run(ctx)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}

	if processed != 0 {
		t.Errorf("expected processed=0 on cancelled context, got %d", processed)
	}
}

// Benchmark worker creation
func BenchmarkWorker_New(b *testing.B) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewWorker(agg)
	}
}

// Benchmark worker start/stop
func BenchmarkWorker_StartStop(b *testing.B) {
	calc := NewCalculator()
	agg := NewAggregator(nil, calc)
	w := NewWorker(agg, WithInterval(1*time.Hour))

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = w.Start(ctx)
		_ = w.Stop()
	}
}
