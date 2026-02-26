package a2a

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type PostgresTaskStore struct {
	db *sql.DB
}

func NewPostgresTaskStore(db *sql.DB) *PostgresTaskStore {
	return &PostgresTaskStore{db: db}
}

func (s *PostgresTaskStore) CreateTask(ctx context.Context, task *Task) error {
	messageJSON, err := json.Marshal(task.Message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	artifactsJSON, err := json.Marshal(task.Artifacts)
	if err != nil {
		return fmt.Errorf("marshal artifacts: %w", err)
	}

	query := `
		INSERT INTO a2a_tasks (id, status, session_id, message, artifacts, metadata, created_at, updated_at, expires_at, parent_id, workspace_id, assigned_agent_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = s.db.ExecContext(ctx, query,
		task.ID,
		task.Status,
		task.SessionID,
		messageJSON,
		artifactsJSON,
		task.Metadata,
		task.CreatedAt,
		task.UpdatedAt,
		task.ExpiresAt,
		task.ParentID,
		task.WorkspaceID,
		task.AssignedAgentID,
	)

	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	return nil
}

func (s *PostgresTaskStore) GetTask(ctx context.Context, id string) (*Task, error) {
	var task Task
	var messageJSON, artifactsJSON []byte

	query := `
		SELECT id, status, session_id, message, artifacts, metadata, created_at, updated_at, expires_at, parent_id, workspace_id, assigned_agent_id
		FROM a2a_tasks
		WHERE id = $1
	`

	row := s.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&task.ID,
		&task.Status,
		&task.SessionID,
		&messageJSON,
		&artifactsJSON,
		&task.Metadata,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.ExpiresAt,
		&task.ParentID,
		&task.WorkspaceID,
		&task.AssignedAgentID,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task: %w", err)
	}

	if err := json.Unmarshal(messageJSON, &task.Message); err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}

	if err := json.Unmarshal(artifactsJSON, &task.Artifacts); err != nil {
		return nil, fmt.Errorf("unmarshal artifacts: %w", err)
	}

	return &task, nil
}

func (s *PostgresTaskStore) UpdateTask(ctx context.Context, task *Task) error {
	messageJSON, err := json.Marshal(task.Message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	artifactsJSON, err := json.Marshal(task.Artifacts)
	if err != nil {
		return fmt.Errorf("marshal artifacts: %w", err)
	}

	query := `
		UPDATE a2a_tasks
		SET status = $1, message = $2, artifacts = $3, metadata = $4, updated_at = $5
		WHERE id = $6
	`

	result, err := s.db.ExecContext(ctx, query,
		task.Status,
		messageJSON,
		artifactsJSON,
		task.Metadata,
		time.Now(),
		task.ID,
	)

	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrTaskNotFound
	}

	return nil
}

func (s *PostgresTaskStore) DeleteTask(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM a2a_tasks WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrTaskNotFound
	}

	return nil
}

func (s *PostgresTaskStore) ListTasks(ctx context.Context, filter TaskFilter) ([]*Task, error) {
	var tasks []*Task
	var args []interface{}
	var conditions []string
	argIdx := 1

	query := `
		SELECT id, status, session_id, message, artifacts, metadata, created_at, updated_at, expires_at, parent_id, workspace_id, assigned_agent_id
		FROM a2a_tasks
		WHERE 1=1
	`

	if filter.WorkspaceID != "" {
		conditions = append(conditions, fmt.Sprintf(" AND workspace_id = $%d", argIdx))
		args = append(args, filter.WorkspaceID)
		argIdx++
	}

	if filter.SessionID != "" {
		conditions = append(conditions, fmt.Sprintf(" AND session_id = $%d", argIdx))
		args = append(args, filter.SessionID)
		argIdx++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf(" AND status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	for _, cond := range conditions {
		query += cond
	}

	query += " ORDER BY created_at DESC"

	// Apply limit with default
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, limit)
	argIdx++

	// Apply offset
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	query += fmt.Sprintf(" OFFSET $%d", argIdx)
	args = append(args, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var task Task
		var messageJSON, artifactsJSON []byte

		err := rows.Scan(
			&task.ID,
			&task.Status,
			&task.SessionID,
			&messageJSON,
			&artifactsJSON,
			&task.Metadata,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.ExpiresAt,
			&task.ParentID,
			&task.WorkspaceID,
			&task.AssignedAgentID,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}

		if err := json.Unmarshal(messageJSON, &task.Message); err != nil {
			return nil, fmt.Errorf("unmarshal message: %w", err)
		}

		if err := json.Unmarshal(artifactsJSON, &task.Artifacts); err != nil {
			return nil, fmt.Errorf("unmarshal artifacts: %w", err)
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// Ensure PostgresTaskStore implements TaskStore interface.
var _ TaskStore = (*PostgresTaskStore)(nil)
