package cost

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"log/slog"

	"radgateway/internal/logger"
)

// Aggregator provides cost aggregation and batch calculation capabilities.
// It queries usage records from the database and calculates costs.
type Aggregator struct {
	db     *sql.DB
	calc   *Calculator
	log    *slog.Logger
	batchSize int
}

// AggregatorOption configures the Aggregator.
type AggregatorOption func(*Aggregator)

// WithBatchSize sets the batch size for processing records.
func WithBatchSize(size int) AggregatorOption {
	return func(a *Aggregator) {
		if size > 0 {
			a.batchSize = size
		}
	}
}

// NewAggregator creates a new cost aggregator.
func NewAggregator(db *sql.DB, calc *Calculator, opts ...AggregatorOption) *Aggregator {
	a := &Aggregator{
		db:        db,
		calc:      calc,
		log:       logger.WithComponent("cost_aggregator"),
		batchSize: 100,
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// FetchUncalculatedRecords retrieves usage records that haven't had costs calculated yet.
// Limited to the configured batch size.
func (a *Aggregator) FetchUncalculatedRecords(ctx context.Context) ([]UncalculatedRecord, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	query := `
		SELECT
			id,
			workspace_id,
			provider_id,
			incoming_model,
			selected_model,
			prompt_tokens,
			completion_tokens,
			total_tokens,
			created_at
		FROM usage_records
		WHERE cost_usd IS NULL
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := a.db.QueryContext(ctx, query, a.batchSize)
	if err != nil {
		return nil, fmt.Errorf("querying uncalculated records: %w", err)
	}
	defer rows.Close()

	var records []UncalculatedRecord
	for rows.Next() {
		var r UncalculatedRecord
		var selectedModel sql.NullString
		var providerID sql.NullString

		err := rows.Scan(
			&r.ID,
			&r.WorkspaceID,
			&providerID,
			&r.IncomingModel,
			&selectedModel,
			&r.PromptTokens,
			&r.CompletionTokens,
			&r.TotalTokens,
			&r.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning record: %w", err)
		}

		if selectedModel.Valid {
			r.SelectedModel = &selectedModel.String
		}
		if providerID.Valid {
			r.ProviderID = &providerID.String
		}

		records = append(records, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating records: %w", err)
	}

	return records, nil
}

// CountUncalculatedRecords returns the number of usage records without cost calculations.
func (a *Aggregator) CountUncalculatedRecords(ctx context.Context) (int64, error) {
	if a.db == nil {
		return 0, fmt.Errorf("database connection not available")
	}

	query := `SELECT COUNT(*) FROM usage_records WHERE cost_usd IS NULL`

	var count int64
	err := a.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting uncalculated records: %w", err)
	}

	return count, nil
}

// UpdateRecordCost updates a single usage record with calculated cost.
func (a *Aggregator) UpdateRecordCost(ctx context.Context, recordID string, cost float64) error {
	if a.db == nil {
		return fmt.Errorf("database connection not available")
	}

	query := `
		UPDATE usage_records
		SET cost_usd = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := a.db.ExecContext(ctx, query, cost, recordID)
	if err != nil {
		return fmt.Errorf("updating record cost: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no record found with id %s", recordID)
	}

	return nil
}

// BatchUpdateCosts updates costs for multiple records in a transaction.
func (a *Aggregator) BatchUpdateCosts(ctx context.Context, costs map[string]float64) (int, error) {
	if a.db == nil {
		return 0, fmt.Errorf("database connection not available")
	}

	if len(costs) == 0 {
		return 0, nil
	}

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		UPDATE usage_records
		SET cost_usd = $1, updated_at = NOW()
		WHERE id = $2
	`)
	if err != nil {
		return 0, fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close()

	updated := 0
	for recordID, cost := range costs {
		result, err := stmt.ExecContext(ctx, cost, recordID)
		if err != nil {
			a.log.Warn("failed to update record cost",
				"record_id", recordID,
				"error", err.Error())
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			updated++
		}
	}

	if err := tx.Commit(); err != nil {
		return updated, fmt.Errorf("committing transaction: %w", err)
	}

	return updated, nil
}

// ProcessBatch calculates and updates costs for a batch of uncalculated records.
// Returns the number of records processed and any error encountered.
func (a *Aggregator) ProcessBatch(ctx context.Context) (int, error) {
	records, err := a.FetchUncalculatedRecords(ctx)
	if err != nil {
		return 0, err
	}

	if len(records) == 0 {
		return 0, nil
	}

	costs := make(map[string]float64)
	for _, record := range records {
		breakdown := a.calc.CalculateFromRecord(record)

		// Validate the calculation before storing
		if err := a.calc.Validate(breakdown); err != nil {
			a.log.Warn("cost validation failed, using zero",
				"record_id", record.ID,
				"error", err.Error(),
				"calculated_cost", breakdown.TotalCost)
			breakdown.TotalCost = 0
		}

		costs[record.ID] = breakdown.TotalCost
	}

	updated, err := a.BatchUpdateCosts(ctx, costs)
	if err != nil {
		return updated, fmt.Errorf("batch update failed: %w", err)
	}

	a.log.Info("processed batch of costs",
		"processed", updated,
		"total_in_batch", len(records))

	return updated, nil
}

// GetCostSummary returns aggregated cost data for a workspace within a time period.
func (a *Aggregator) GetCostSummary(ctx context.Context, filter QueryFilter) (*CostSummary, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	query := `
		SELECT
			COALESCE(SUM(cost_usd), 0) as total_cost,
			COALESCE(SUM(prompt_tokens::FLOAT / 1000 * $1), 0) as prompt_cost,
			COALESCE(SUM(completion_tokens::FLOAT / 1000 * $2), 0) as completion_cost,
			COUNT(*) as request_count,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) as completion_tokens
		FROM usage_records
		WHERE workspace_id = $3
		  AND created_at >= $4
		  AND created_at <= $5
		  AND cost_usd IS NOT NULL
	`

	// Default rates for estimation (actual costs are stored in cost_usd)
	// These are just placeholders for the query structure
	defaultInputRate := 0.01
	defaultOutputRate := 0.03

	summary := &CostSummary{
		WorkspaceID:      filter.WorkspaceID,
		PeriodStart:      filter.StartTime,
		PeriodEnd:        filter.EndTime,
		AggregationLevel: filter.AggregationLevel,
		Currency:         filter.Currency,
	}

	if filter.Currency == "" {
		summary.Currency = DefaultCurrency
	}

	var promptCost, completionCost float64
	err := a.db.QueryRowContext(ctx, query,
		defaultInputRate, defaultOutputRate,
		filter.WorkspaceID, filter.StartTime, filter.EndTime,
	).Scan(
		&summary.TotalCost,
		&promptCost,
		&completionCost,
		&summary.RequestCount,
		&summary.TotalTokens,
		&summary.PromptTokens,
		&summary.CompletionTokens,
	)
	if err != nil {
		return nil, fmt.Errorf("querying cost summary: %w", err)
	}

	summary.PromptCost = promptCost
	summary.CompletionCost = completionCost

	return summary, nil
}

// GetCostByModel returns costs grouped by model.
func (a *Aggregator) GetCostByModel(ctx context.Context, filter QueryFilter) ([]CostByModel, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	query := `
		SELECT
			COALESCE(p.provider_type, 'unknown') as provider_name,
			COALESCE(r.selected_model, r.incoming_model) as model_id,
			COALESCE(SUM(r.cost_usd), 0) as total_cost,
			COUNT(*) as request_count,
			COALESCE(SUM(r.total_tokens), 0) as total_tokens,
			COALESCE(SUM(r.prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(r.completion_tokens), 0) as completion_tokens
		FROM usage_records r
		LEFT JOIN providers p ON r.provider_id = p.id
		WHERE r.workspace_id = $1
		  AND r.created_at >= $2
		  AND r.created_at <= $3
		  AND r.cost_usd IS NOT NULL
	`

	params := []any{filter.WorkspaceID, filter.StartTime, filter.EndTime}

	if filter.ProviderID != nil {
		query += " AND r.provider_id = $" + fmt.Sprintf("%d", len(params)+1)
		params = append(params, *filter.ProviderID)
	}

	if filter.ModelID != nil {
		query += " AND (r.selected_model = $" + fmt.Sprintf("%d", len(params)+1) +
			" OR r.incoming_model = $" + fmt.Sprintf("%d", len(params)+1) + ")"
		params = append(params, *filter.ModelID)
	}

	query += `
		GROUP BY provider_name, model_id
		ORDER BY total_cost DESC
	`

	rows, err := a.db.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("querying cost by model: %w", err)
	}
	defer rows.Close()

	var results []CostByModel
	for rows.Next() {
		var r CostByModel
		r.WorkspaceID = filter.WorkspaceID
		r.PeriodStart = filter.StartTime
		r.PeriodEnd = filter.EndTime
		r.Currency = filter.Currency
		if r.Currency == "" {
			r.Currency = DefaultCurrency
		}

		err := rows.Scan(
			&r.ProviderName,
			&r.ModelID,
			&r.TotalCost,
			&r.RequestCount,
			&r.TotalTokens,
			&r.PromptTokens,
			&r.CompletionTokens,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning cost by model: %w", err)
		}

		// Set model name same as model ID for now
		r.ModelName = r.ModelID

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating cost by model: %w", err)
	}

	return results, nil
}

// GetCostByProvider returns costs grouped by provider.
func (a *Aggregator) GetCostByProvider(ctx context.Context, filter QueryFilter) ([]CostByProvider, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	query := `
		SELECT
			COALESCE(p.id, 'unknown') as provider_id,
			COALESCE(p.provider_type, 'unknown') as provider_name,
			COALESCE(SUM(r.cost_usd), 0) as total_cost,
			COUNT(*) as request_count,
			COALESCE(SUM(r.total_tokens), 0) as total_tokens,
			COALESCE(SUM(r.prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(r.completion_tokens), 0) as completion_tokens
		FROM usage_records r
		LEFT JOIN providers p ON r.provider_id = p.id
		WHERE r.workspace_id = $1
		  AND r.created_at >= $2
		  AND r.created_at <= $3
		  AND r.cost_usd IS NOT NULL
		GROUP BY provider_id, provider_name
		ORDER BY total_cost DESC
	`

	rows, err := a.db.QueryContext(ctx, query,
		filter.WorkspaceID, filter.StartTime, filter.EndTime,
	)
	if err != nil {
		return nil, fmt.Errorf("querying cost by provider: %w", err)
	}
	defer rows.Close()

	var results []CostByProvider
	for rows.Next() {
		var r CostByProvider
		r.WorkspaceID = filter.WorkspaceID
		r.PeriodStart = filter.StartTime
		r.PeriodEnd = filter.EndTime
		r.Currency = filter.Currency
		if r.Currency == "" {
			r.Currency = DefaultCurrency
		}

		err := rows.Scan(
			&r.ProviderID,
			&r.ProviderName,
			&r.TotalCost,
			&r.RequestCount,
			&r.TotalTokens,
			&r.PromptTokens,
			&r.CompletionTokens,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning cost by provider: %w", err)
		}

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating cost by provider: %w", err)
	}

	return results, nil
}

// GetCostTimeseries returns cost data over time.
func (a *Aggregator) GetCostTimeseries(ctx context.Context, filter QueryFilter) ([]CostTimeseriesPoint, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// Determine time bucket based on aggregation level
	var timeBucket string
	switch filter.AggregationLevel {
	case AggHourly:
		timeBucket = "date_trunc('hour', r.created_at)"
	case AggDaily:
		timeBucket = "date_trunc('day', r.created_at)"
	case AggWeekly:
		timeBucket = "date_trunc('week', r.created_at)"
	case AggMonthly:
		timeBucket = "date_trunc('month', r.created_at)"
	default:
		timeBucket = "date_trunc('day', r.created_at)"
	}

	query := fmt.Sprintf(`
		SELECT
			%s as bucket,
			COALESCE(SUM(r.cost_usd), 0) as total_cost,
			COUNT(*) as request_count,
			COALESCE(SUM(r.total_tokens), 0) as total_tokens,
			COALESCE(SUM(r.prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(r.completion_tokens), 0) as completion_tokens
		FROM usage_records r
		WHERE r.workspace_id = $1
		  AND r.created_at >= $2
		  AND r.created_at <= $3
		  AND r.cost_usd IS NOT NULL
		GROUP BY bucket
		ORDER BY bucket ASC
	`, timeBucket)

	rows, err := a.db.QueryContext(ctx, query,
		filter.WorkspaceID, filter.StartTime, filter.EndTime,
	)
	if err != nil {
		return nil, fmt.Errorf("querying cost timeseries: %w", err)
	}
	defer rows.Close()

	var results []CostTimeseriesPoint
	for rows.Next() {
		var r CostTimeseriesPoint

		err := rows.Scan(
			&r.Timestamp,
			&r.TotalCost,
			&r.RequestCount,
			&r.TotalTokens,
			&r.PromptTokens,
			&r.CompletionTokens,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning timeseries: %w", err)
		}

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating timeseries: %w", err)
	}

	return results, nil
}

// GetTotalCost returns the total cost for a workspace in a given period.
func (a *Aggregator) GetTotalCost(ctx context.Context, workspaceID string, startTime, endTime time.Time) (float64, error) {
	if a.db == nil {
		return 0, fmt.Errorf("database connection not available")
	}

	query := `
		SELECT COALESCE(SUM(cost_usd), 0)
		FROM usage_records
		WHERE workspace_id = $1
		  AND created_at >= $2
		  AND created_at <= $3
		  AND cost_usd IS NOT NULL
	`

	var total float64
	err := a.db.QueryRowContext(ctx, query, workspaceID, startTime, endTime).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("querying total cost: %w", err)
	}

	return total, nil
}
