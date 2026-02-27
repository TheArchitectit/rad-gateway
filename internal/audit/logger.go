// Package audit provides security audit logging functionality.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
)

// Logger handles security audit logging.
type Logger struct {
	db     *sql.DB
	config Config
}

// Config configures the audit logger.
type Config struct {
	BufferSize     int
	FlushInterval  time.Duration
	MaxRetries     int
	AsyncLogging   bool
	LogAllRequests bool
}

// DefaultConfig returns default audit configuration.
func DefaultConfig() Config {
	return Config{
		BufferSize:     1000,
		FlushInterval:  5 * time.Second,
		MaxRetries:     3,
		AsyncLogging:   true,
		LogAllRequests: false,
	}
}

// NewLogger creates a new audit logger.
func NewLogger(db *sql.DB, config Config) *Logger {
	return &Logger{
		db:     db,
		config: config,
	}
}

// Log creates a new audit event.
func (l *Logger) Log(ctx context.Context, eventType EventType, actor Actor, resource Resource, action, result string, details map[string]interface{}) error {
	event := Event{
		ID:        uuid.New().String(),
		Timestamp: time.Now().UTC(),
		Type:      eventType,
		Severity:  SeverityForEventType(eventType),
		Actor:     actor,
		Resource:  resource,
		Action:    action,
		Result:    result,
		Details:   details,
		Metadata:  make(map[string]string),
	}

	return l.Store(ctx, event)
}

// LogWithRequest creates an audit event with request information.
func (l *Logger) LogWithRequest(ctx context.Context, eventType EventType, actor Actor, resource Resource, action, result string, details map[string]interface{}, reqInfo RequestInfo) error {
	event := Event{
		ID:          uuid.New().String(),
		Timestamp:   time.Now().UTC(),
		Type:        eventType,
		Severity:      SeverityForEventType(eventType),
		Actor:       actor,
		Resource:    resource,
		Action:      action,
		Result:      result,
		Details:     details,
		RequestInfo: reqInfo,
		Metadata:    make(map[string]string),
	}

	return l.Store(ctx, event)
}

// stripPort removes port from IP address for PostgreSQL inet type
func stripPort(ipStr string) string {
	host, _, err := net.SplitHostPort(ipStr)
	if err != nil {
		// If SplitHostPort fails, assume it's already a plain IP
		return ipStr
	}
	return host
}

// Store persists an audit event to the database.
func (l *Logger) Store(ctx context.Context, event Event) error {
	detailsJSON, err := json.Marshal(event.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO audit_log (
			id, timestamp, type, severity,
			actor_type, actor_id, actor_name, actor_role, actor_ip, user_agent,
			resource_type, resource_id, resource_name, workspace_id,
			action, result, details, metadata,
			request_method, request_path, request_query, trace_id, request_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
	`

	_, err = l.db.ExecContext(ctx, query,
		event.ID,
		event.Timestamp,
		event.Type,
		event.Severity,
		event.Actor.Type,
		event.Actor.ID,
		event.Actor.Name,
		event.Actor.Role,
		stripPort(event.Actor.IP),
		event.Actor.UserAgent,
		event.Resource.Type,
		event.Resource.ID,
		event.Resource.Name,
		event.Resource.Workspace,
		event.Action,
		event.Result,
		detailsJSON,
		metadataJSON,
		event.RequestInfo.Method,
		event.RequestInfo.Path,
		event.RequestInfo.Query,
		event.RequestInfo.TraceID,
		event.RequestInfo.RequestID,
	)

	return err
}

// Query retrieves audit events matching the filter.
func (l *Logger) Query(ctx context.Context, filter EventFilter) ([]Event, error) {
	query := `
		SELECT id, timestamp, type, severity,
			actor_type, actor_id, actor_name, actor_role, actor_ip, user_agent,
			resource_type, resource_id, resource_name, workspace_id,
			action, result, details, metadata,
			request_method, request_path, request_query, trace_id, request_id
		FROM audit_log
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 0

	if len(filter.Types) > 0 {
		argIndex++
		query += fmt.Sprintf(" AND type = ANY($%d)", argIndex)
		types := make([]string, len(filter.Types))
		for i, t := range filter.Types {
			types[i] = string(t)
		}
		args = append(args, types)
	}

	if len(filter.Severities) > 0 {
		argIndex++
		query += fmt.Sprintf(" AND severity = ANY($%d)", argIndex)
		severities := make([]string, len(filter.Severities))
		for i, s := range filter.Severities {
			severities[i] = string(s)
		}
		args = append(args, severities)
	}

	if filter.ActorID != "" {
		argIndex++
		query += fmt.Sprintf(" AND actor_id = $%d", argIndex)
		args = append(args, filter.ActorID)
	}

	if filter.ResourceID != "" {
		argIndex++
		query += fmt.Sprintf(" AND resource_id = $%d", argIndex)
		args = append(args, filter.ResourceID)
	}

	if filter.StartTime != nil {
		argIndex++
		query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, *filter.StartTime)
	}

	if filter.EndTime != nil {
		argIndex++
		query += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, *filter.EndTime)
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		argIndex++
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		argIndex++
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	rows, err := l.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		var detailsJSON, metadataJSON []byte

		err := rows.Scan(
			&event.ID, &event.Timestamp, &event.Type, &event.Severity,
			&event.Actor.Type, &event.Actor.ID, &event.Actor.Name, &event.Actor.Role, &event.Actor.IP, &event.Actor.UserAgent,
			&event.Resource.Type, &event.Resource.ID, &event.Resource.Name, &event.Resource.Workspace,
			&event.Action, &event.Result, &detailsJSON, &metadataJSON,
			&event.RequestInfo.Method, &event.RequestInfo.Path, &event.RequestInfo.Query, &event.RequestInfo.TraceID, &event.RequestInfo.RequestID,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(detailsJSON, &event.Details)
		json.Unmarshal(metadataJSON, &event.Metadata)
		events = append(events, event)
	}

	return events, rows.Err()
}

// GetRecentEvents retrieves recent events of a specific type.
func (l *Logger) GetRecentEvents(ctx context.Context, eventType EventType, limit int) ([]Event, error) {
	return l.Query(ctx, EventFilter{
		Types: []EventType{eventType},
		Limit: limit,
	})
}

// CountEvents returns the count of events matching the filter.
func (l *Logger) CountEvents(ctx context.Context, filter EventFilter) (int, error) {
	query := `SELECT COUNT(*) FROM audit_log WHERE 1=1`
	args := []interface{}{}
	argIndex := 0

	if len(filter.Types) > 0 {
		argIndex++
		query += fmt.Sprintf(" AND type = ANY($%d)", argIndex)
		types := make([]string, len(filter.Types))
		for i, t := range filter.Types {
			types[i] = string(t)
		}
		args = append(args, types)
	}

	if filter.ActorID != "" {
		argIndex++
		query += fmt.Sprintf(" AND actor_id = $%d", argIndex)
		args = append(args, filter.ActorID)
	}

	if filter.StartTime != nil {
		argIndex++
		query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, *filter.StartTime)
	}

	if filter.EndTime != nil {
		argIndex++
		query += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, *filter.EndTime)
	}

	var count int
	err := l.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// PurgeOldEvents removes events older than the retention period.
func (l *Logger) PurgeOldEvents(ctx context.Context, retentionDays int) (int64, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	result, err := l.db.ExecContext(ctx,
		"DELETE FROM audit_log WHERE timestamp < $1",
		cutoff,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
