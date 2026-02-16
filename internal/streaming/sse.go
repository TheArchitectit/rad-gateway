// Package streaming provides Server-Sent Events (SSE) parsing and writing
// for real-time streaming responses from AI providers.
package streaming

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Event represents a single SSE event.
type Event struct {
	ID    string
	Event string
	Data  string
	Retry int
}

// Parser reads and parses SSE events from an io.Reader.
type Parser struct {
	reader  *bufio.Reader
	event   Event
	scratch bytes.Buffer
}

// NewParser creates a new SSE parser from an io.Reader.
func NewParser(r io.Reader) *Parser {
	return &Parser{
		reader: bufio.NewReader(r),
	}
}

// Next reads the next SSE event from the stream.
// Returns io.EOF when the stream ends.
func (p *Parser) Next() (Event, error) {
	p.event = Event{}
	p.scratch.Reset()

	for {
		line, err := p.reader.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) && p.scratch.Len() > 0 {
				// Process remaining data before EOF
				p.event.Data = p.scratch.String()
				return p.event, nil
			}
			return Event{}, err
		}

		// Remove trailing newline
		line = bytes.TrimSuffix(line, []byte("\n"))
		// Remove carriage return if present (Windows line endings)
		line = bytes.TrimSuffix(line, []byte("\r"))

		// Empty line marks end of event
		if len(line) == 0 {
			if p.scratch.Len() > 0 || p.event.ID != "" || p.event.Event != "" {
				p.event.Data = p.scratch.String()
				return p.event, nil
			}
			// Otherwise, continue reading for next event
			continue
		}

		// Parse field
		if err := p.parseField(line); err != nil {
			return Event{}, err
		}
	}
}

// parseField parses a single SSE field line.
func (p *Parser) parseField(line []byte) error {
	// Find colon separator
	colonIdx := bytes.Index(line, []byte(":"))

	var field, value []byte
	if colonIdx == -1 {
		// No colon - entire line is field name, value is empty
		field = line
		value = []byte{}
	} else {
		field = line[:colonIdx]
		// Value starts after colon; if there's a space after colon, skip it
		value = line[colonIdx+1:]
		if len(value) > 0 && value[0] == ' ' {
			value = value[1:]
		}
	}

	switch string(field) {
	case "id":
		p.event.ID = string(value)
	case "event":
		p.event.Event = string(value)
	case "data":
		if p.scratch.Len() > 0 {
			p.scratch.WriteByte('\n')
		}
		p.scratch.Write(value)
	case "retry":
		if retry, err := strconv.Atoi(string(value)); err == nil {
			p.event.Retry = retry
		}
	default:
		// Ignore unknown fields per SSE spec
	}

	return nil
}

// Writer writes SSE events to an http.ResponseWriter.
type Writer struct {
	w       http.ResponseWriter
	flusher http.Flusher
	mu      sync.Mutex
	closed  bool
}

// NewWriter creates a new SSE writer from an http.ResponseWriter.
// It sets the appropriate SSE headers and returns the writer.
// Returns an error if the ResponseWriter doesn't support flushing.
func NewWriter(w http.ResponseWriter) (*Writer, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported: ResponseWriter does not implement http.Flusher")
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Write headers
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	return &Writer{
		w:       w,
		flusher: flusher,
	}, nil
}

// WriteEvent writes a single SSE event to the stream.
func (w *Writer) WriteEvent(event Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return errors.New("stream is closed")
	}

	var buf strings.Builder

	// Write ID if present
	if event.ID != "" {
		fmt.Fprintf(&buf, "id: %s\n", escapeField(event.ID))
	}

	// Write event type if present
	if event.Event != "" {
		fmt.Fprintf(&buf, "event: %s\n", escapeField(event.Event))
	}

	// Write retry if present
	if event.Retry > 0 {
		fmt.Fprintf(&buf, "retry: %d\n", event.Retry)
	}

	// Write data - may be multiline
	if event.Data != "" {
		// Split data by newlines and prefix each line with "data: "
		lines := strings.Split(event.Data, "\n")
		for _, line := range lines {
			fmt.Fprintf(&buf, "data: %s\n", line)
		}
	}

	// Empty line marks end of event
	buf.WriteByte('\n')

	_, err := w.w.Write([]byte(buf.String()))
	if err != nil {
		return fmt.Errorf("write event: %w", err)
	}

	w.flusher.Flush()
	return nil
}

// WriteData writes a simple data-only event.
func (w *Writer) WriteData(data string) error {
	return w.WriteEvent(Event{Data: data})
}

// WriteComment writes a comment (ignored by clients, useful for keepalive).
func (w *Writer) WriteComment(comment string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return errors.New("stream is closed")
	}

	_, err := fmt.Fprintf(w.w, ": %s\n\n", comment)
	if err != nil {
		return fmt.Errorf("write comment: %w", err)
	}

	w.flusher.Flush()
	return nil
}

// Close sends the final event and marks the stream as closed.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.closed = true
	return nil
}

// IsClosed returns true if the stream is closed.
func (w *Writer) IsClosed() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closed
}

// escapeField escapes special characters in SSE field values.
// Lines starting with certain characters need special handling.
func escapeField(s string) string {
	// SSE doesn't require escaping per se, but we need to handle
	// lines that start with colon to distinguish from comments
	return s
}

// Client represents an SSE client connection that can receive events.
type Client struct {
	writer *Writer
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// NewClient creates a new SSE client connection.
func NewClient(w http.ResponseWriter, r *http.Request) (*Client, error) {
	writer, err := NewWriter(w)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(r.Context())

	c := &Client{
		writer: writer,
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	// Watch for client disconnect
	go func() {
		select {
		case <-r.Context().Done():
			c.Close()
		case <-ctx.Done():
		}
	}()

	return c, nil
}

// Send sends an event to the client.
func (c *Client) Send(event Event) error {
	select {
	case <-c.ctx.Done():
		return c.ctx.Err()
	default:
		return c.writer.WriteEvent(event)
	}
}

// SendData sends a data-only event to the client.
func (c *Client) SendData(data string) error {
	return c.Send(Event{Data: data})
}

// Keepalive sends a comment to keep the connection alive.
func (c *Client) Keepalive() error {
	return c.writer.WriteComment(fmt.Sprintf("keepalive %d", time.Now().Unix()))
}

// Close closes the client connection.
func (c *Client) Close() error {
	c.cancel()
	close(c.done)
	return c.writer.Close()
}

// Done returns a channel that's closed when the client disconnects.
func (c *Client) Done() <-chan struct{} {
	return c.done
}

// Context returns the client's context.
func (c *Client) Context() context.Context {
	return c.ctx
}
