// Package kafka provides Kafka messaging for A2A async eventing
package kafka

import (
	"encoding/json"
	"time"
)

// TaskEvent represents an A2A task lifecycle event
type TaskEvent struct {
	EventID   string          `json:"event_id"`
	TaskID    string          `json:"task_id"`
	EventType string          `json:"event_type"` // created, updated, completed, failed
	Status    string          `json:"status"`
	AgentID   string          `json:"agent_id"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// WebhookCallback represents an async webhook notification
type WebhookCallback struct {
	CallbackID string          `json:"callback_id"`
	TaskID     string          `json:"task_id"`
	WebhookURL string          `json:"webhook_url"`
	Payload    json.RawMessage `json:"payload"`
	RetryCount int             `json:"retry_count"`
	MaxRetries int             `json:"max_retries"`
	CreatedAt  time.Time       `json:"created_at"`
}

// AgentDiscoveryEvent represents agent registration/deregistration
type AgentDiscoveryEvent struct {
	EventID   string    `json:"event_id"`
	AgentID   string    `json:"agent_id"`
	Action    string    `json:"action"` // registered, deregistered, updated
	AgentCard []byte    `json:"agent_card"`
	Timestamp time.Time `json:"timestamp"`
}

// ProtocolMetrics represents A2A protocol performance metrics
type ProtocolMetrics struct {
	Timestamp    time.Time `json:"timestamp"`
	TaskID       string    `json:"task_id"`
	DurationMs   int64     `json:"duration_ms"`
	TokenCount   int64     `json:"token_count"`
	TrustScore   float64   `json:"trust_score"`
	AgentID      string    `json:"agent_id"`
	WorkspaceID  string    `json:"workspace_id"`
	Jurisdiction string    `json:"jurisdiction"`
}
