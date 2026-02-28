// Package agui provides AG-UI (Agent-User Interface) protocol support for RAD Gateway.
// It defines event models for real-time UI updates during agent runs.
package agui

import (
	"encoding/json"
	"time"
)

// EventType represents the type of AG-UI event.
type EventType string

const (
	// EventTypeRunStart indicates the beginning of an agent run.
	EventTypeRunStart EventType = "run.start"
	// EventTypeRunComplete indicates successful completion of an agent run.
	EventTypeRunComplete EventType = "run.complete"
	// EventTypeRunError indicates an error occurred during the agent run.
	EventTypeRunError EventType = "run.error"
	// EventTypeMessageDelta indicates a delta update to a message.
	EventTypeMessageDelta EventType = "message.delta"
	// EventTypeToolCall indicates a tool is being called by the agent.
	EventTypeToolCall EventType = "tool.call"
	// EventTypeToolResult indicates a result from a tool execution.
	EventTypeToolResult EventType = "tool.result"
	// EventTypeStateSnapshot provides a full state snapshot.
	EventTypeStateSnapshot EventType = "state.snapshot"
	// EventTypeStateDelta provides incremental state updates.
	EventTypeStateDelta EventType = "state.delta"
)

// Event represents a single AG-UI event for real-time UI updates.
type Event struct {
	// Type is the event type categorizing this event.
	Type EventType `json:"type"`
	// RunID is the unique identifier for the agent run.
	RunID string `json:"run_id"`
	// AgentID identifies the agent generating the event.
	AgentID string `json:"agent_id"`
	// ThreadID identifies the conversation thread.
	ThreadID string `json:"thread_id"`
	// Timestamp is when the event was generated.
	Timestamp time.Time `json:"timestamp"`
	// Data contains event-specific payload data.
	Data map[string]interface{} `json:"data,omitempty"`
	// Metadata contains additional contextual information.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// WithData adds a key-value pair to the event's Data map and returns the event.
// This enables method chaining for fluent event construction.
func (e *Event) WithData(key string, value interface{}) *Event {
	if e.Data == nil {
		e.Data = make(map[string]interface{})
	}
	e.Data[key] = value
	return e
}

// WithMetadata adds a key-value pair to the event's Metadata map and returns the event.
// This enables method chaining for fluent event construction.
func (e *Event) WithMetadata(key, value string) *Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

// MarshalJSON implements custom JSON marshaling for Event with formatted timestamp.
func (e Event) MarshalJSON() ([]byte, error) {
	type Alias Event
	return json.Marshal(&struct {
		Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     (Alias)(e),
		Timestamp: e.Timestamp.Format(time.RFC3339Nano),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for Event with timestamp parsing.
func (e *Event) UnmarshalJSON(data []byte) error {
	type Alias Event
	aux := &struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias: (*Alias)(e),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Timestamp != "" {
		timestamp, err := time.Parse(time.RFC3339Nano, aux.Timestamp)
		if err != nil {
			// Try parsing without nanoseconds as fallback
			timestamp, err = time.Parse(time.RFC3339, aux.Timestamp)
			if err != nil {
				return err
			}
		}
		e.Timestamp = timestamp
	}
	return nil
}

// RunState represents the complete state of an agent run.
type RunState struct {
	// RunID is the unique identifier for the agent run.
	RunID string `json:"run_id"`
	// AgentID identifies the agent.
	AgentID string `json:"agent_id"`
	// ThreadID identifies the conversation thread.
	ThreadID string `json:"thread_id"`
	// Status is the current run status (e.g., "running", "completed", "error").
	Status string `json:"status"`
	// Messages contains the conversation history.
	Messages []Message `json:"messages,omitempty"`
	// ToolCalls contains pending or completed tool calls.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// State contains arbitrary run-specific state data.
	State map[string]interface{} `json:"state,omitempty"`
}

// Message represents a single message in the conversation.
type Message struct {
	// ID is the unique identifier for this message.
	ID string `json:"id"`
	// Role indicates the message sender (e.g., "user", "assistant", "system").
	Role string `json:"role"`
	// Content is the message text content.
	Content string `json:"content"`
	// Timestamp is when the message was created.
	Timestamp time.Time `json:"timestamp"`
}

// MarshalJSON implements custom JSON marshaling for Message with formatted timestamp.
func (m Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	return json.Marshal(&struct {
		Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     (Alias)(m),
		Timestamp: m.Timestamp.Format(time.RFC3339Nano),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for Message with timestamp parsing.
func (m *Message) UnmarshalJSON(data []byte) error {
	type Alias Message
	aux := &struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias: (*Alias)(m),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Timestamp != "" {
		timestamp, err := time.Parse(time.RFC3339Nano, aux.Timestamp)
		if err != nil {
			// Try parsing without nanoseconds as fallback
			timestamp, err = time.Parse(time.RFC3339, aux.Timestamp)
			if err != nil {
				return err
			}
		}
		m.Timestamp = timestamp
	}
	return nil
}

// ToolCall represents a tool invocation by the agent.
type ToolCall struct {
	// ID is the unique identifier for this tool call.
	ID string `json:"id"`
	// Tool is the name of the tool being called.
	Tool string `json:"tool"`
	// Arguments contains the parameters passed to the tool.
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	// Result contains the tool execution result (nil if not yet completed).
	Result interface{} `json:"result,omitempty"`
}

// NewEvent creates a new Event with the specified type and IDs, initialized with current timestamp.
func NewEvent(eventType EventType, runID, agentID, threadID string) *Event {
	return &Event{
		Type:      eventType,
		RunID:     runID,
		AgentID:   agentID,
		ThreadID:  threadID,
		Timestamp: time.Now().UTC(),
	}
}

// NewRunState creates a new RunState with the specified IDs and initial status.
func NewRunState(runID, agentID, threadID, status string) *RunState {
	return &RunState{
		RunID:     runID,
		AgentID:   agentID,
		ThreadID:  threadID,
		Status:    status,
		Messages:  make([]Message, 0),
		ToolCalls: make([]ToolCall, 0),
		State:     make(map[string]interface{}),
	}
}

// NewMessage creates a new Message with the specified properties and current timestamp.
func NewMessage(id, role, content string) Message {
	return Message{
		ID:        id,
		Role:      role,
		Content:   content,
		Timestamp: time.Now().UTC(),
	}
}

// NewToolCall creates a new ToolCall with the specified properties.
func NewToolCall(id, tool string, arguments map[string]interface{}) ToolCall {
	if arguments == nil {
		arguments = make(map[string]interface{})
	}
	return ToolCall{
		ID:        id,
		Tool:      tool,
		Arguments: arguments,
	}
}

// AddMessage adds a message to the RunState's Messages slice.
func (r *RunState) AddMessage(msg Message) {
	r.Messages = append(r.Messages, msg)
}

// AddToolCall adds a tool call to the RunState's ToolCalls slice.
func (r *RunState) AddToolCall(tc ToolCall) {
	r.ToolCalls = append(r.ToolCalls, tc)
}

// SetStateValue sets a value in the RunState's State map.
func (r *RunState) SetStateValue(key string, value interface{}) {
	if r.State == nil {
		r.State = make(map[string]interface{})
	}
	r.State[key] = value
}
