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

func (s *PostgresTaskStore) ListTasks(ctx context.Context, workspaceID string, limit, offset int) (*TaskList, error) {
	var tasks []Task

	query := `
		SELECT id, status, session_id, message, artifacts, metadata, created_at, updated_at, expires_at, parent_id, workspace_id, assigned_agent_id
		FROM a2a_tasks
		WHERE workspace_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, workspaceID, limit, offset)
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

		tasks = append(tasks, task)
	}

	var total int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM a2a_tasks WHERE workspace_id = $1", workspaceID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("count tasks: %w", err)
	}

	return &TaskList{
		Items:  tasks,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *PostgresTaskStore) ListTasksBySession(ctx context.Context, sessionID string) ([]Task, error) {
	var tasks []Task

	query := `
		SELECT id, status, session_id, message, artifacts, metadata, created_at, updated_at, expires_at, parent_id, workspace_id, assigned_agent_id
		FROM a2a_tasks
		WHERE session_id = $1
		ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list tasks by session: %w", err)
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

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *PostgresTaskStore) ListTasksByStatus(ctx context.Context, status TaskState, limit int) ([]Task, error) {
	var tasks []Task

	query := `
		SELECT id, status, session_id, message, artifacts, metadata, created_at, updated_at, expires_at, parent_id, workspace_id, assigned_agent_id
		FROM a2a_tasks
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("list tasks by status: %w", err)
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

		tasks = append(tasks, task)
	}

	return tasks, nil
}
