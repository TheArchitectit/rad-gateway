package streaming

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParser_Next(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Event
		wantErr  bool
	}{
		{
			name:  "simple event",
			input: "data: hello\n\n",
			expected: []Event{
				{Data: "hello"},
			},
		},
		{
			name:  "event with id",
			input: "id: 1\nevent: message\ndata: hello\n\n",
			expected: []Event{
				{ID: "1", Event: "message", Data: "hello"},
			},
		},
		{
			name:  "multiline data",
			input: "data: line1\ndata: line2\n\n",
			expected: []Event{
				{Data: "line1\nline2"},
			},
		},
		{
			name:  "multiple events",
			input: "data: first\n\ndata: second\n\n",
			expected: []Event{
				{Data: "first"},
				{Data: "second"},
			},
		},
		{
			name:  "with retry",
			input: "retry: 5000\ndata: hello\n\n",
			expected: []Event{
				{Data: "hello", Retry: 5000},
			},
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true, // EOF on empty input
		},
		{
			name:    "comment only",
			input:   ": comment\n\n",
			wantErr: true, // Comments are ignored, resulting in EOF
		},
		{
			name:  "carriage return line endings",
			input: "data: hello\r\n\r\n",
			expected: []Event{
				{Data: "hello"},
			},
		},
		{
			name:  "field without value",
			input: "data:\n\n",
			expected: []Event{
				{Data: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			var events []Event

			for {
				event, err := parser.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					if !tt.wantErr {
						t.Fatalf("unexpected error: %v", err)
					}
					return
				}
				events = append(events, event)
			}

			if len(events) != len(tt.expected) {
				t.Errorf("got %d events, want %d", len(events), len(tt.expected))
			}

			for i, exp := range tt.expected {
				if i >= len(events) {
					break
				}
				got := events[i]
				if got.ID != exp.ID {
					t.Errorf("event[%d].ID = %q, want %q", i, got.ID, exp.ID)
				}
				if got.Event != exp.Event {
					t.Errorf("event[%d].Event = %q, want %q", i, got.Event, exp.Event)
				}
				if got.Data != exp.Data {
					t.Errorf("event[%d].Data = %q, want %q", i, got.Data, exp.Data)
				}
				if got.Retry != exp.Retry {
					t.Errorf("event[%d].Retry = %d, want %d", i, got.Retry, exp.Retry)
				}
			}
		})
	}
}

func TestWriter_WriteEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		expected string
	}{
		{
			name:     "simple data event",
			event:    Event{Data: "hello"},
			expected: "data: hello\n\n",
		},
		{
			name:     "event with id",
			event:    Event{ID: "123", Data: "hello"},
			expected: "id: 123\ndata: hello\n\n",
		},
		{
			name:     "event with type",
			event:    Event{Event: "message", Data: "hello"},
			expected: "event: message\ndata: hello\n\n",
		},
		{
			name:     "full event",
			event:    Event{ID: "1", Event: "message", Data: "hello", Retry: 5000},
			expected: "id: 1\nevent: message\nretry: 5000\ndata: hello\n\n",
		},
		{
			name:     "multiline data",
			event:    Event{Data: "line1\nline2"},
			expected: "data: line1\ndata: line2\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			writer, err := NewWriter(rr)
			if err != nil {
				t.Fatalf("NewWriter failed: %v", err)
			}

			err = writer.WriteEvent(tt.event)
			if err != nil {
				t.Fatalf("WriteEvent failed: %v", err)
			}

			// Flush to ensure data is written
			rr.Flush()

			got := rr.Body.String()
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestWriter_Headers(t *testing.T) {
	rr := httptest.NewRecorder()
	_, err := NewWriter(rr)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	tests := []struct {
		header   string
		expected string
	}{
		{"Content-Type", "text/event-stream"},
		{"Cache-Control", "no-cache"},
		{"Connection", "keep-alive"},
		{"X-Accel-Buffering", "no"},
	}

	for _, tt := range tests {
		got := rr.Header().Get(tt.header)
		if got != tt.expected {
			t.Errorf("header %q = %q, want %q", tt.header, got, tt.expected)
		}
	}

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestWriter_WriteComment(t *testing.T) {
	rr := httptest.NewRecorder()
	writer, err := NewWriter(rr)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	err = writer.WriteComment("keepalive")
	if err != nil {
		t.Fatalf("WriteComment failed: %v", err)
	}

	rr.Flush()

	expected := ": keepalive\n\n"
	got := rr.Body.String()
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestWriter_Close(t *testing.T) {
	rr := httptest.NewRecorder()
	writer, err := NewWriter(rr)
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	if writer.IsClosed() {
		t.Error("writer should not be closed initially")
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if !writer.IsClosed() {
		t.Error("writer should be closed after Close()")
	}

	// Writing to closed writer should fail
	err = writer.WriteEvent(Event{Data: "test"})
	if err == nil {
		t.Error("WriteEvent should fail on closed writer")
	}
}

func TestNewWriter_NotFlusher(t *testing.T) {
	// Create a ResponseWriter that doesn't implement http.Flusher
	type nonFlusher struct {
		http.ResponseWriter
	}
	w := nonFlusher{httptest.NewRecorder()}

	_, err := NewWriter(w)
	if err == nil {
		t.Error("expected error for non-flusher ResponseWriter")
	}
}

func TestClient_Send(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)

	client, err := NewClient(rr, req)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	defer client.Close()

	err = client.Send(Event{Data: "hello"})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	rr.Flush()

	body := rr.Body.String()
	if !strings.Contains(body, "data: hello") {
		t.Errorf("body %q should contain 'data: hello'", body)
	}
}

func TestClient_SendData(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)

	client, err := NewClient(rr, req)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	defer client.Close()

	err = client.SendData("test data")
	if err != nil {
		t.Fatalf("SendData failed: %v", err)
	}

	rr.Flush()

	body := rr.Body.String()
	if !strings.Contains(body, "data: test data") {
		t.Errorf("body %q should contain 'data: test data'", body)
	}
}

func BenchmarkParser_Next(b *testing.B) {
	input := bytes.Repeat([]byte("data: hello world\n\n"), 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewParser(bytes.NewReader(input))
		for {
			_, err := parser.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkWriter_WriteEvent(b *testing.B) {
	event := Event{
		ID:    "12345",
		Event: "message",
		Data:  `{"choices":[{"delta":{"content":"hello"}}]}`,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		writer, _ := NewWriter(rr)
		writer.WriteEvent(event)
	}
}
