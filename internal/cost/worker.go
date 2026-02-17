package cost

import (
	"context"
	"sync"
	"time"

	"log/slog"

	"radgateway/internal/logger"
)

// Worker is a background service that periodically calculates costs
// for unprocessed usage records.
type Worker struct {
	aggregator    *Aggregator
	calc          *Calculator
	interval      time.Duration
	batchSize     int
	log           *slog.Logger

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
	running bool
}

// WorkerOption configures the Worker.
type WorkerOption func(*Worker)

// WithInterval sets the polling interval for cost calculation.
func WithInterval(interval time.Duration) WorkerOption {
	return func(w *Worker) {
		if interval > 0 {
			w.interval = interval
		}
	}
}

// WithWorkerBatchSize sets the batch size for processing.
func WithWorkerBatchSize(size int) WorkerOption {
	return func(w *Worker) {
		if size > 0 {
			w.batchSize = size
		}
	}
}

// Default values
const (
	DefaultWorkerInterval  = 5 * time.Minute
	DefaultWorkerBatchSize = 100
)

// NewWorker creates a new background cost calculation worker.
func NewWorker(aggregator *Aggregator, opts ...WorkerOption) *Worker {
	w := &Worker{
		aggregator: aggregator,
		interval:   DefaultWorkerInterval,
		batchSize:  DefaultWorkerBatchSize,
		log:        logger.WithComponent("cost_worker"),
		running:    false,
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

// Start begins the background cost calculation worker.
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return ErrWorkerAlreadyRunning
	}

	if w.aggregator == nil {
		return ErrNilAggregator
	}

	w.ctx, w.cancel = context.WithCancel(ctx)
	w.running = true
	w.wg.Add(1)

	w.log.Info("cost worker starting",
		"interval", w.interval.String(),
		"batch_size", w.batchSize)

	go w.run()

	return nil
}

// Stop gracefully shuts down the worker.
func (w *Worker) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return nil
	}

	w.log.Info("cost worker stopping")

	w.cancel()
	w.wg.Wait()

	w.running = false
	w.log.Info("cost worker stopped")

	return nil
}

// IsRunning returns true if the worker is currently running.
func (w *Worker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// GetStats returns current worker statistics.
func (w *Worker) GetStats() WorkerStats {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return WorkerStats{
		Running:   w.running,
		Interval:  w.interval,
		BatchSize: w.batchSize,
	}
}

// WorkerStats contains runtime statistics for the worker.
type WorkerStats struct {
	Running          bool          `json:"running"`
	Interval         time.Duration `json:"interval"`
	BatchSize        int           `json:"batch_size"`
	LastProcessedAt  *time.Time    `json:"last_processed_at,omitempty"`
	LastProcessedCount int        `json:"last_processed_count,omitempty"`
}

// run is the main worker loop.
func (w *Worker) run() {
	defer w.wg.Done()

	// Create a ticker for periodic processing
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Process immediately on start
	w.processBatch()

	for {
		select {
		case <-w.ctx.Done():
			w.log.Info("cost worker received shutdown signal")
			return
		case <-ticker.C:
			w.processBatch()
		}
	}
}

// processBatch processes a single batch of uncalculated records.
func (w *Worker) processBatch() {
	if w.aggregator == nil {
		w.log.Error("cost worker: aggregator is nil")
		return
	}

	w.log.Debug("cost worker: processing batch")

	ctx, cancel := context.WithTimeout(w.ctx, 2*w.interval)
	defer cancel()

	processed, err := w.aggregator.ProcessBatch(ctx)
	if err != nil {
		w.log.Error("cost worker: failed to process batch",
			"error", err.Error())
		return
	}

	if processed > 0 {
		w.log.Info("cost worker: processed batch",
			"processed", processed)
	} else {
		w.log.Debug("cost worker: no records to process")
	}
}

// ForceProcess triggers an immediate batch processing.
// Useful for testing or when manually triggered.
func (w *Worker) ForceProcess() (int, error) {
	w.mu.RLock()
	running := w.running
	w.mu.RUnlock()

	if !running {
		return 0, ErrWorkerNotRunning
	}

	w.log.Info("cost worker: forced processing triggered")

	ctx, cancel := context.WithTimeout(w.ctx, 2*w.interval)
	defer cancel()

	return w.aggregator.ProcessBatch(ctx)
}

// GetPendingCount returns the number of uncalculated records.
func (w *Worker) GetPendingCount() (int64, error) {
	w.mu.RLock()
	running := w.running
	w.mu.RUnlock()

	if !running {
		return 0, ErrWorkerNotRunning
	}

	ctx, cancel := context.WithTimeout(w.ctx, 30*time.Second)
	defer cancel()

	return w.aggregator.CountUncalculatedRecords(ctx)
}

// Errors
var (
	ErrWorkerAlreadyRunning = &WorkerError{msg: "worker already running"}
	ErrWorkerNotRunning     = &WorkerError{msg: "worker not running"}
	ErrNilAggregator        = &WorkerError{msg: "aggregator is nil"}
)

// WorkerError represents a worker-related error.
type WorkerError struct {
	msg string
}

func (e *WorkerError) Error() string {
	return e.msg
}

// Is allows error matching with errors.Is.
func (e *WorkerError) Is(target error) bool {
	if t, ok := target.(*WorkerError); ok {
		return e.msg == t.msg
	}
	return false
}

// ImmediateWorker processes all pending records immediately and exits.
// Useful for one-time cost calculation jobs.
type ImmediateWorker struct {
	aggregator *Aggregator
	log        *slog.Logger
	maxRecords int
}

// NewImmediateWorker creates a new immediate processing worker.
func NewImmediateWorker(aggregator *Aggregator, maxRecords int) *ImmediateWorker {
	if maxRecords <= 0 {
		maxRecords = 10000
	}

	return &ImmediateWorker{
		aggregator: aggregator,
		log:        logger.WithComponent("cost_immediate_worker"),
		maxRecords: maxRecords,
	}
}

// Run processes all pending records up to maxRecords.
func (w *ImmediateWorker) Run(ctx context.Context) (int, error) {
	w.log.Info("immediate cost worker starting", "max_records", w.maxRecords)

	totalProcessed := 0
	batchCount := 0

	for totalProcessed < w.maxRecords {
		select {
		case <-ctx.Done():
			w.log.Info("immediate cost worker: context cancelled",
				"total_processed", totalProcessed)
			return totalProcessed, ctx.Err()
		default:
		}

		processed, err := w.aggregator.ProcessBatch(ctx)
		if err != nil {
			w.log.Error("immediate cost worker: batch failed",
				"error", err.Error(),
				"total_processed", totalProcessed)
			return totalProcessed, err
		}

		if processed == 0 {
			// No more records to process
			break
		}

		totalProcessed += processed
		batchCount++

		w.log.Debug("immediate cost worker: processed batch",
			"batch", batchCount,
			"batch_processed", processed,
			"total_processed", totalProcessed)
	}

	w.log.Info("immediate cost worker completed",
		"total_processed", totalProcessed,
		"batches", batchCount)

	return totalProcessed, nil
}
