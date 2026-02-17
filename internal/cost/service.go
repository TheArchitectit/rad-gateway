package cost

import (
	"context"
	"database/sql"
	"sync"

	"log/slog"

	"radgateway/internal/logger"
)

// Service is the main cost tracking service that ties together all components.
// It provides a high-level API for cost calculation, aggregation, and querying.
type Service struct {
	calculator *Calculator
	aggregator *Aggregator
	worker     *Worker
	api        *APIHandler
	db         *sql.DB
	log        *slog.Logger

	mu      sync.RWMutex
	running bool
}

// Config holds configuration for the cost service.
type Config struct {
	// DB is the database connection
	DB *sql.DB

	// EnableWorker enables the background cost calculation worker
	EnableWorker bool

	// WorkerInterval is the polling interval for the worker (default: 5m)
	WorkerInterval string

	// WorkerBatchSize is the batch size for processing (default: 100)
	WorkerBatchSize int

	// PricingOverrides allows custom pricing for models
	PricingOverrides map[string]TokenRate
}

// NewService creates a new cost tracking service.
func NewService(cfg Config) (*Service, error) {
	log := logger.WithComponent("cost_service")

	// Create calculator with optional pricing overrides
	calcOpts := []CalculatorOption{}
	if cfg.PricingOverrides != nil {
		calcOpts = append(calcOpts, WithPricingOverrides(cfg.PricingOverrides))
	}
	calculator := NewCalculator(calcOpts...)

	// Create aggregator
	aggregator := NewAggregator(cfg.DB, calculator)

	svc := &Service{
		calculator: calculator,
		aggregator: aggregator,
		db:         cfg.DB,
		log:        log,
	}

	// Create worker if enabled
	if cfg.EnableWorker {
		workerOpts := []WorkerOption{}

		if cfg.WorkerBatchSize > 0 {
			workerOpts = append(workerOpts, WithWorkerBatchSize(cfg.WorkerBatchSize))
		}

		if cfg.WorkerInterval != "" {
			duration, err := parseDuration(cfg.WorkerInterval)
			if err != nil {
				log.Warn("invalid worker interval, using default", "interval", cfg.WorkerInterval, "error", err.Error())
			} else {
				workerOpts = append(workerOpts, WithInterval(duration))
			}
		}

		svc.worker = NewWorker(aggregator, workerOpts...)
	}

	// Create API handler
	svc.api = NewAPIHandler(aggregator)

	log.Info("cost service created",
		"worker_enabled", cfg.EnableWorker,
		"has_db", cfg.DB != nil)

	return svc, nil
}

// Start initializes the cost service and starts the background worker if enabled.
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrServiceAlreadyRunning
	}

	if s.db == nil {
		s.log.Warn("cost service started without database connection - queries will fail")
	}

	// Start worker if enabled
	if s.worker != nil {
		if err := s.worker.Start(ctx); err != nil {
			s.log.Error("failed to start cost worker", "error", err.Error())
			return err
		}
		s.log.Info("cost worker started")
	}

	s.running = true
	s.log.Info("cost service started")

	return nil
}

// Stop gracefully shuts down the cost service.
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	// Stop worker if running
	if s.worker != nil {
		if err := s.worker.Stop(); err != nil {
			s.log.Error("failed to stop cost worker", "error", err.Error())
		}
	}

	s.running = false
	s.log.Info("cost service stopped")

	return nil
}

// IsRunning returns true if the service is running.
func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetCalculator returns the calculator component.
func (s *Service) GetCalculator() *Calculator {
	return s.calculator
}

// GetAggregator returns the aggregator component.
func (s *Service) GetAggregator() *Aggregator {
	return s.aggregator
}

// GetWorker returns the worker component (may be nil if disabled).
func (s *Service) GetWorker() *Worker {
	return s.worker
}

// GetAPIHandler returns the API handler for registering endpoints.
func (s *Service) GetAPIHandler() *APIHandler {
	return s.api
}

// RegisterRoutes registers cost API endpoints with the provided mux.
func (s *Service) RegisterRoutes(mux interface{ HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) }) {
	s.api.Register(mux)
}

// CalculateCost calculates the cost for a single request.
func (s *Service) CalculateCost(model string, promptTokens, completionTokens int64) CostBreakdown {
	return s.calculator.Calculate(model, promptTokens, completionTokens)
}

// CalculateCostFromRecord calculates cost from an uncalculated record.
func (s *Service) CalculateCostFromRecord(record UncalculatedRecord) CostBreakdown {
	return s.calculator.CalculateFromRecord(record)
}

// ValidateCost validates a cost calculation.
func (s *Service) ValidateCost(breakdown CostBreakdown) error {
	return s.calculator.Validate(breakdown)
}

// EstimateCost provides a cost estimate for a planned request.
func (s *Service) EstimateCost(model string, estimatedPromptTokens, estimatedMaxCompletionTokens int64) CostBreakdown {
	return s.calculator.EstimateCost(model, estimatedPromptTokens, estimatedMaxCompletionTokens)
}

// SetModelPricing sets custom pricing for a model.
func (s *Service) SetModelPricing(model string, rate TokenRate) {
	s.calculator.SetPricing(model, rate)
}

// GetModelPricing returns the pricing for a model.
func (s *Service) GetModelPricing(model string) (TokenRate, bool) {
	return s.calculator.GetPricing(model)
}

// ListKnownModels returns all models with known pricing.
func (s *Service) ListKnownModels() []string {
	return s.calculator.ListKnownModels()
}

// ForceCalculation triggers immediate cost calculation.
func (s *Service) ForceCalculation(ctx context.Context) (int, error) {
	if s.aggregator == nil {
		return 0, ErrNilAggregator
	}

	s.log.Info("forcing cost calculation")
	return s.aggregator.ProcessBatch(ctx)
}

// GetCostSummary returns aggregated cost data for a workspace.
func (s *Service) GetCostSummary(ctx context.Context, filter QueryFilter) (*CostSummary, error) {
	if s.aggregator == nil {
		return nil, ErrNilAggregator
	}
	return s.aggregator.GetCostSummary(ctx, filter)
}

// GetCostByModel returns costs grouped by model.
func (s *Service) GetCostByModel(ctx context.Context, filter QueryFilter) ([]CostByModel, error) {
	if s.aggregator == nil {
		return nil, ErrNilAggregator
	}
	return s.aggregator.GetCostByModel(ctx, filter)
}

// GetCostByProvider returns costs grouped by provider.
func (s *Service) GetCostByProvider(ctx context.Context, filter QueryFilter) ([]CostByProvider, error) {
	if s.aggregator == nil {
		return nil, ErrNilAggregator
	}
	return s.aggregator.GetCostByProvider(ctx, filter)
}

// GetCostTimeseries returns cost data over time.
func (s *Service) GetCostTimeseries(ctx context.Context, filter QueryFilter) ([]CostTimeseriesPoint, error) {
	if s.aggregator == nil {
		return nil, ErrNilAggregator
	}
	return s.aggregator.GetCostTimeseries(ctx, filter)
}

// GetPendingCount returns the number of uncalculated records.
func (s *Service) GetPendingCount(ctx context.Context) (int64, error) {
	if s.aggregator == nil {
		return 0, ErrNilAggregator
	}
	return s.aggregator.CountUncalculatedRecords(ctx)
}

// GetStats returns service statistics.
func (s *Service) GetStats() ServiceStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := ServiceStats{
		Running: s.running,
	}

	if s.worker != nil {
		stats.WorkerStats = s.worker.GetStats()
	}

	return stats
}

// ServiceStats contains runtime statistics for the service.
type ServiceStats struct {
	Running     bool       `json:"running"`
	WorkerStats WorkerStats `json:"worker_stats,omitempty"`
}

// Errors
var (
	ErrServiceAlreadyRunning = &ServiceError{msg: "service already running"}
	ErrServiceNotRunning     = &ServiceError{msg: "service not running"}
)

// ServiceError represents a service-related error.
type ServiceError struct {
	msg string
}

func (e *ServiceError) Error() string {
	return e.msg
}

// parseDuration parses a duration string, returning an error if invalid.
func parseDuration(s string) (Duration, error) {
	d, err := time.ParseDuration(s)
	return d, err
}

// Import needed types
import "net/http"
import "time"
