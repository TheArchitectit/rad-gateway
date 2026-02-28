// Package agui provides AG-UI (Agent-User Interface) protocol support for RAD Gateway.
package agui

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		eventType EventType
		expected string
	}{
		{"RunStart", EventTypeRunStart, "run.start"},
		{"RunComplete", EventTypeRunComplete, "run.complete"},
		{"RunError", EventTypeRunError, "run.error"},
		{"MessageDelta", EventTypeMessageDelta, "message.delta"},
		{"ToolCall", EventTypeToolCall, "tool.call"},
		{"ToolResult", EventTypeToolResult, "tool.result"},
		{"StateSnapshot", EventTypeStateSnapshot, "state.snapshot"},
		{"StateDelta", EventTypeStateDelta, "state.delta"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.eventType))
		})
	}
}

func TestNewEvent(t *testing.T) {
	event := NewEvent(EventTypeRunStart, "run-123", "agent-456", "thread-789")

	assert.Equal(t, EventTypeRunStart, event.Type)
	assert.Equal(t, "run-123", event.RunID)
	assert.Equal(t, "agent-456", event.AgentID)
	assert.Equal(t, "thread-789", event.ThreadID)
	assert.WithinDuration(t, time.Now().UTC(), event.Timestamp, time.Second)
	assert.Nil(t, event.Data)
	assert.Nil(t, event.Metadata)
}

func TestEvent_WithData(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Event
		key      string
		value    interface{}
		expected map[string]interface{}
	}{
		{
			name: "add data to nil map",
			setup: func() *Event {
				return &Event{}
			},
			key:      "content",
			value:    "Hello World",
			expected: map[string]interface{}{"content": "Hello World"},
		},
		{
			name: "add data to existing map",
			setup: func() *Event {
				return &Event{Data: map[string]interface{}{"existing": "value"}}
			},
			key:      "new_key",
			value:    42,
			expected: map[string]interface{}{"existing": "value", "new_key": 42},
		},
		{
			name: "overwrite existing key",
			setup: func() *Event {
				return &Event{Data: map[string]interface{}{"key": "old"}}
			},
			key:      "key",
			value:    "new",
			expected: map[string]interface{}{"key": "new"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := tt.setup()
			result := event.WithData(tt.key, tt.value)

			assert.Equal(t, event, result)
			assert.Equal(t, tt.expected, event.Data)
		})
	}
}

func TestEvent_WithMetadata(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Event
		key      string
		value    string
		expected map[string]string
	}{
		{
			name: "add metadata to nil map",
			setup: func() *Event {
				return &Event{}
			},
			key:      "source",
			value:    "api",
			expected: map[string]string{"source": "api"},
		},
		{
			name: "add metadata to existing map",
			setup: func() *Event {
				return &Event{Metadata: map[string]string{"existing": "value"}}
			},
			key:      "version",
			value:    "1.0",
			expected: map[string]string{"existing": "value", "version": "1.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := tt.setup()
			result := event.WithMetadata(tt.key, tt.value)

			assert.Equal(t, event, result)
			assert.Equal(t, tt.expected, event.Metadata)
		})
	}
}

func TestEvent_MarshalJSON(t *testing.T) {
	fixedTime := time.Date(2026, 2, 28, 12, 30, 45, 123456789, time.UTC)

	tests := []struct {
		name     string
		event    Event
		expected string
	}{
		{
			name: "minimal event",
			event: Event{
				Type:      EventTypeRunStart,
				RunID:     "run-123",
				AgentID:   "agent-456",
				ThreadID:  "thread-789",
				Timestamp: fixedTime,
			},
			expected: `{"type":"run.start","run_id":"run-123","agent_id":"agent-456","thread_id":"thread-789","timestamp":"2026-02-28T12:30:45.123456789Z"}`,
		},
		{
			name: "event with data",
			event: Event{
				Type:      EventTypeMessageDelta,
				RunID:     "run-123",
				AgentID:   "agent-456",
				ThreadID:  "thread-789",
				Timestamp: fixedTime,
				Data:      map[string]interface{}{"delta": "Hello"},
			},
			expected: `{"type":"message.delta","run_id":"run-123","agent_id":"agent-456","thread_id":"thread-789","timestamp":"2026-02-28T12:30:45.123456789Z","data":{"delta":"Hello"}}`,
		},
		{
			name: "event with metadata",
			event: Event{
				Type:      EventTypeToolCall,
				RunID:     "run-123",
				AgentID:   "agent-456",
				ThreadID:  "thread-789",
				Timestamp: fixedTime,
				Metadata:  map[string]string{"tool_name": "search"},
			},
			expected: `{"type":"tool.call","run_id":"run-123","agent_id":"agent-456","thread_id":"thread-789","timestamp":"2026-02-28T12:30:45.123456789Z","metadata":{"tool_name":"search"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.event)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestEvent_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name         string
		json         string
		expectedType EventType
		expectedTime time.Time
		checkData    bool
		expectedData map[string]interface{}
	}{
		{
			name:         "minimal event",
			json:         `{"type":"run.start","run_id":"run-123","agent_id":"agent-456","thread_id":"thread-789","timestamp":"2026-02-28T12:30:45.123456789Z"}`,
			expectedType: EventTypeRunStart,
			expectedTime: time.Date(2026, 2, 28, 12, 30, 45, 123456789, time.UTC),
		},
		{
			name:         "event without nanoseconds",
			json:         `{"type":"run.complete","run_id":"run-123","agent_id":"agent-456","thread_id":"thread-789","timestamp":"2026-02-28T12:30:45Z"}`,
			expectedType: EventTypeRunComplete,
			expectedTime: time.Date(2026, 2, 28, 12, 30, 45, 0, time.UTC),
		},
		{
			name:         "event with data",
			json:         `{"type":"message.delta","run_id":"run-123","agent_id":"agent-456","thread_id":"thread-789","timestamp":"2026-02-28T12:30:45.123456789Z","data":{"content":"Hello"}}`,
			expectedType: EventTypeMessageDelta,
			expectedTime: time.Date(2026, 2, 28, 12, 30, 45, 123456789, time.UTC),
			checkData:    true,
			expectedData: map[string]interface{}{"content": "Hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event Event
			err := json.Unmarshal([]byte(tt.json), &event)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedType, event.Type)
			assert.Equal(t, tt.expectedTime, event.Timestamp)
			assert.Equal(t, "run-123", event.RunID)
			assert.Equal(t, "agent-456", event.AgentID)
			assert.Equal(t, "thread-789", event.ThreadID)

			if tt.checkData {
				assert.Equal(t, tt.expectedData, event.Data)
			}
		})
	}
}

func TestEvent_RoundTrip(t *testing.T) {
	original := NewEvent(EventTypeToolResult, "run-abc", "agent-def", "thread-ghi").
		WithData("result", map[string]interface{}{"status": "success", "data": 123}).
		WithMetadata("tool_id", "tool-001").
		WithMetadata("duration_ms", "150")

	// Marshal to JSON
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal back
	var restored Event
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	// Verify all fields
	assert.Equal(t, original.Type, restored.Type)
	assert.Equal(t, original.RunID, restored.RunID)
	assert.Equal(t, original.AgentID, restored.AgentID)
	assert.Equal(t, original.ThreadID, restored.ThreadID)
	assert.WithinDuration(t, original.Timestamp, restored.Timestamp, time.Millisecond)

	// JSON unmarshals numbers as float64, not int, so check values individually
	assert.Equal(t, "success", restored.Data["result"].(map[string]interface{})["status"])
	assert.Equal(t, float64(123), restored.Data["result"].(map[string]interface{})["data"])
	assert.Equal(t, original.Metadata, restored.Metadata)
}

func TestNewRunState(t *testing.T) {
	state := NewRunState("run-123", "agent-456", "thread-789", "running")

	assert.Equal(t, "run-123", state.RunID)
	assert.Equal(t, "agent-456", state.AgentID)
	assert.Equal(t, "thread-789", state.ThreadID)
	assert.Equal(t, "running", state.Status)
	assert.Empty(t, state.Messages)
	assert.Empty(t, state.ToolCalls)
	assert.NotNil(t, state.State)
	assert.Empty(t, state.State)
}

func TestRunState_AddMessage(t *testing.T) {
	state := NewRunState("run-123", "agent-456", "thread-789", "running")

	msg1 := NewMessage("msg-1", "user", "Hello")
	state.AddMessage(msg1)

	assert.Len(t, state.Messages, 1)
	assert.Equal(t, msg1, state.Messages[0])

	msg2 := NewMessage("msg-2", "assistant", "Hi there!")
	state.AddMessage(msg2)

	assert.Len(t, state.Messages, 2)
	assert.Equal(t, msg2, state.Messages[1])
}

func TestRunState_AddToolCall(t *testing.T) {
	state := NewRunState("run-123", "agent-456", "thread-789", "running")

	tc1 := NewToolCall("tc-1", "search", map[string]interface{}{"query": "hello"})
	state.AddToolCall(tc1)

	assert.Len(t, state.ToolCalls, 1)
	assert.Equal(t, tc1, state.ToolCalls[0])
}

func TestRunState_SetStateValue(t *testing.T) {
	state := NewRunState("run-123", "agent-456", "thread-789", "running")

	state.SetStateValue("counter", 42)
	assert.Equal(t, 42, state.State["counter"])

	state.SetStateValue("name", "test")
	assert.Equal(t, "test", state.State["name"])

	// Test with nil state
	state.State = nil
	state.SetStateValue("key", "value")
	assert.Equal(t, "value", state.State["key"])
}

func TestRunState_MarshalJSON(t *testing.T) {
	state := NewRunState("run-123", "agent-456", "thread-789", "completed")
	state.AddMessage(NewMessage("msg-1", "user", "Hello"))
	state.AddToolCall(NewToolCall("tc-1", "search", map[string]interface{}{"q": "test"}))
	state.SetStateValue("progress", 100)

	data, err := json.Marshal(state)
	require.NoError(t, err)

	// Verify structure by unmarshaling to map
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "run-123", result["run_id"])
	assert.Equal(t, "agent-456", result["agent_id"])
	assert.Equal(t, "thread-789", result["thread_id"])
	assert.Equal(t, "completed", result["status"])
	assert.NotNil(t, result["messages"])
	assert.NotNil(t, result["tool_calls"])
	assert.NotNil(t, result["state"])
}

func TestNewMessage(t *testing.T) {
	msg := NewMessage("msg-123", "assistant", "Hello, world!")

	assert.Equal(t, "msg-123", msg.ID)
	assert.Equal(t, "assistant", msg.Role)
	assert.Equal(t, "Hello, world!", msg.Content)
	assert.WithinDuration(t, time.Now().UTC(), msg.Timestamp, time.Second)
}

func TestMessage_MarshalJSON(t *testing.T) {
	fixedTime := time.Date(2026, 2, 28, 12, 30, 45, 123456789, time.UTC)
	msg := Message{
		ID:        "msg-123",
		Role:      "user",
		Content:   "Hello",
		Timestamp: fixedTime,
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	expected := `{"id":"msg-123","role":"user","content":"Hello","timestamp":"2026-02-28T12:30:45.123456789Z"}`
	assert.JSONEq(t, expected, string(data))
}

func TestMessage_UnmarshalJSON(t *testing.T) {
	jsonData := `{"id":"msg-456","role":"assistant","content":"How can I help?","timestamp":"2026-02-28T10:20:30.987654321Z"}`

	var msg Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	require.NoError(t, err)

	assert.Equal(t, "msg-456", msg.ID)
	assert.Equal(t, "assistant", msg.Role)
	assert.Equal(t, "How can I help?", msg.Content)
	assert.Equal(t, time.Date(2026, 2, 28, 10, 20, 30, 987654321, time.UTC), msg.Timestamp)
}

func TestMessage_RoundTrip(t *testing.T) {
	original := NewMessage("msg-round", "system", "System message")

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored Message
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.Role, restored.Role)
	assert.Equal(t, original.Content, restored.Content)
	assert.WithinDuration(t, original.Timestamp, restored.Timestamp, time.Millisecond)
}

func TestNewToolCall(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]interface{}
	}{
		{
			name:      "with arguments",
			arguments: map[string]interface{}{"query": "hello", "limit": 10},
		},
		{
			name:      "nil arguments",
			arguments: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewToolCall("tc-123", "search", tt.arguments)

			assert.Equal(t, "tc-123", tc.ID)
			assert.Equal(t, "search", tc.Tool)
			assert.NotNil(t, tc.Arguments)

			if tt.arguments != nil {
				assert.Equal(t, tt.arguments, tc.Arguments)
			} else {
				assert.Empty(t, tc.Arguments)
			}
			assert.Nil(t, tc.Result)
		})
	}
}

func TestToolCall_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		tc       ToolCall
		expected string
	}{
		{
			name:     "tool call without result",
			tc:       ToolCall{ID: "tc-1", Tool: "search", Arguments: map[string]interface{}{"q": "hello"}},
			expected: `{"id":"tc-1","tool":"search","arguments":{"q":"hello"}}`,
		},
		{
			name:     "tool call with result",
			tc:       ToolCall{ID: "tc-2", Tool: "calculator", Arguments: map[string]interface{}{"expr": "1+1"}, Result: 2},
			expected: `{"id":"tc-2","tool":"calculator","arguments":{"expr":"1+1"},"result":2}`,
		},
		{
			name:     "tool call with complex result",
			tc:       ToolCall{ID: "tc-3", Tool: "fetch", Arguments: map[string]interface{}{"url": "http://example.com"}, Result: map[string]interface{}{"status": 200, "body": "OK"}},
			expected: `{"id":"tc-3","tool":"fetch","arguments":{"url":"http://example.com"},"result":{"body":"OK","status":200}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.tc)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestToolCall_RoundTrip(t *testing.T) {
	original := NewToolCall("tc-round", "weather", map[string]interface{}{"city": "London", "units": "metric"})
	original.Result = map[string]interface{}{
		"temperature": 15.5,
		"humidity":    65,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored ToolCall
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.Tool, restored.Tool)
	assert.Equal(t, original.Arguments, restored.Arguments)
}

func TestCustomTimestampFormat(t *testing.T) {
	tests := []struct {
		name         string
		json         string
		expectedTime time.Time
		checkEqual   bool
		checkWithin  bool
	}{
		{
			name:         "RFC3339 with nanoseconds",
			json:         `{"type":"run.start","run_id":"r","agent_id":"a","thread_id":"t","timestamp":"2026-02-28T12:30:45.123456789Z"}`,
			expectedTime: time.Date(2026, 2, 28, 12, 30, 45, 123456789, time.UTC),
			checkEqual:   true,
		},
		{
			name:         "RFC3339 without nanoseconds",
			json:         `{"type":"run.start","run_id":"r","agent_id":"a","thread_id":"t","timestamp":"2026-02-28T12:30:45Z"}`,
			expectedTime: time.Date(2026, 2, 28, 12, 30, 45, 0, time.UTC),
			checkEqual:   true,
		},
		{
			name:         "RFC3339 with timezone offset",
			json:         `{"type":"run.start","run_id":"r","agent_id":"a","thread_id":"t","timestamp":"2026-02-28T12:30:45+05:30"}`,
			expectedTime: time.Date(2026, 2, 28, 7, 0, 45, 0, time.UTC),
			checkWithin:  true, // Timezone is preserved, so check equality in UTC
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event Event
			err := json.Unmarshal([]byte(tt.json), &event)
			require.NoError(t, err)

			if tt.checkEqual {
				assert.Equal(t, tt.expectedTime, event.Timestamp)
			} else if tt.checkWithin {
				// For timezone offset test, check the instant is the same in UTC
				assert.WithinDuration(t, tt.expectedTime, event.Timestamp.UTC(), time.Second)
			}
		})
	}
}

func TestEvent_WithData_Chaining(t *testing.T) {
	event := NewEvent(EventTypeStateDelta, "run-1", "agent-1", "thread-1").
		WithData("key1", "value1").
		WithData("key2", 123).
		WithData("key3", true)

	assert.Equal(t, "value1", event.Data["key1"])
	assert.Equal(t, 123, event.Data["key2"])
	assert.Equal(t, true, event.Data["key3"])
}

func TestEvent_WithMetadata_Chaining(t *testing.T) {
	event := NewEvent(EventTypeRunComplete, "run-2", "agent-2", "thread-2").
		WithMetadata("source", "api").
		WithMetadata("version", "1.0.0").
		WithMetadata("request_id", "req-123")

	assert.Equal(t, "api", event.Metadata["source"])
	assert.Equal(t, "1.0.0", event.Metadata["version"])
	assert.Equal(t, "req-123", event.Metadata["request_id"])
}

func TestFullEventChaining(t *testing.T) {
	event := NewEvent(EventTypeToolResult, "run-abc", "agent-def", "thread-ghi").
		WithData("tool_name", "search").
		WithData("result", map[string]interface{}{"hits": 5}).
		WithMetadata("duration_ms", "120").
		WithMetadata("cache_hit", "true")

	assert.Equal(t, EventTypeToolResult, event.Type)
	assert.Equal(t, "search", event.Data["tool_name"])
	assert.Equal(t, map[string]interface{}{"hits": 5}, event.Data["result"])
	assert.Equal(t, "120", event.Metadata["duration_ms"])
	assert.Equal(t, "true", event.Metadata["cache_hit"])
}

func BenchmarkEvent_MarshalJSON(b *testing.B) {
	event := NewEvent(EventTypeMessageDelta, "run-123", "agent-456", "thread-789").
		WithData("content", "Hello World").
		WithData("index", 42).
		WithMetadata("source", "stream")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(event)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEvent_UnmarshalJSON(b *testing.B) {
	jsonData := []byte(`{"type":"message.delta","run_id":"run-123","agent_id":"agent-456","thread_id":"thread-789","timestamp":"2026-02-28T12:30:45.123456789Z","data":{"content":"Hello"},"metadata":{"source":"api"}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var event Event
		if err := json.Unmarshal(jsonData, &event); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewMessage("msg-123", "user", "Hello World")
	}
}

func BenchmarkNewToolCall(b *testing.B) {
	args := map[string]interface{}{"query": "hello", "limit": 10}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewToolCall("tc-123", "search", args)
	}
}
