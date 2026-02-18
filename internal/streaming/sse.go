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
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"radgateway/internal/logger"
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
	logger  *slog.Logger
}

// NewParser creates a new SSE parser from an io.Reader.
func NewParser(r io.Reader) *Parser {
	return &Parser{
		reader: bufio.NewReader(r),
		logger: logger.WithComponent("streaming"),
	}
}

// Next reads the next SSE event from the stream.
// Returns io.EOF when the stream ends.
func (p *Parser) Next() (Event, error) {
	p.event = Event{}
	p.scratch.Reset()
	var hasFields bool

	for {
		line, err := p.reader.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				// Process remaining data before EOF
				if p.scratch.Len() > 0 || p.event.ID != "" || p.event.Event != "" || hasFields {
					p.event.Data = p.scratch.String()
					p.logger.Debug("SSE event parsed at EOF", "event_id", p.event.ID, "event_type", p.event.Event)
					return p.event, nil
				}
				// Empty stream or just whitespace/comments
				return Event{}, io.EOF
			}
			p.logger.Error("read error from SSE stream", "error", err)
			return Event{}, err
		}

		// Remove trailing newline
		line = bytes.TrimSuffix(line, []byte("\n"))
		// Remove carriage return if present (Windows line endings)
		line = bytes.TrimSuffix(line, []byte("\r"))

		// Empty line marks end of event
		if len(line) == 0 {
			if p.scratch.Len() > 0 || p.event.ID != "" || p.event.Event != "" || hasFields {
				p.event.Data = p.scratch.String()
				p.logger.Debug("SSE event parsed", "event_id", p.event.ID, "event_type", p.event.Event, "data_len", len(p.event.Data))
				return p.event, nil
			}
			// Otherwise, continue reading for next event
			continue
		}

		// Skip comment lines
		if len(line) > 0 && line[0] == ':' {
			continue
		}

		// Parse field
		if err := p.parseField(line); err != nil {
			p.logger.Error("parse field error", "error", err, "line", string(line))
			return Event{}, err
		}
		hasFields = true
	}
}

// parseField parses a single SSE field line.
func (p *Parser) parseField(line []byte) error {
	// Skip comment lines (lines starting with colon)
	if len(line) > 0 && line[0] == ':' {
		return nil
	}

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
	logger  *slog.Logger
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

	log := logger.WithComponent("streaming")
	log.Debug("SSE writer initialized")

	return &Writer{
		w:       w,
		flusher: flusher,
		logger:  log,
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
		w.logger.Error("write event error", "error", err, "event_id", event.ID, "event_type", event.Event)
		return fmt.Errorf("write event: %w", err)
	}

	w.flusher.Flush()
	w.logger.Debug("SSE event written", "event_id", event.ID, "event_type", event.Event, "data_len", len(event.Data))
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
	w.logger.Debug("SSE writer closed")
	return nil
}

// IsClosed returns true if the stream is closed.
func (w *Writer) IsClosed() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closed
}

// escapeField escapes special characters in SSE field values.
// Per the SSE spec, newlines in field values should be handled by
// splitting data across multiple "data:" lines. This function handles
// escaping for ID and event type fields which cannot contain newlines.
// If a newline is found, it is replaced with a space to maintain
// protocol integrity while preserving the value.
func escapeField(s string) string {
	// Replace newlines and carriage returns with spaces
	// to prevent breaking the SSE protocol structure.
	// Note: For multiline content, use the data field which is
	// handled separately in WriteEvent by splitting on newlines.
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

// Client represents an SSE client connection that can receive events.
type Client struct {
	writer   *Writer
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	closeOnce sync.Once
	logger   *slog.Logger
}

// NewClient creates a new SSE client connection.
func NewClient(w http.ResponseWriter, r *http.Request) (*Client, error) {
	writer, err := NewWriter(w)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(r.Context())
	log := logger.WithComponent("streaming")

	c := &Client{
		writer: writer,
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
		logger: log,
	}

	log.Debug("SSE client connected")

	// Watch for client disconnect
	go func() {
		select {
		case <-r.Context().Done():
			log.Debug("SSE client request context done, closing")
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
		c.logger.Debug("send skipped: client context done", "event_id", event.ID)
		return c.ctx.Err()
	default:
		if err := c.writer.WriteEvent(event); err != nil {
			c.logger.Error("send event failed", "error", err, "event_id", event.ID)
			return err
		}
		return nil
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
	c.closeOnce.Do(func() {
		c.logger.Debug("closing SSE client connection")
		c.cancel()
		close(c.done)
		if err := c.writer.Close(); err != nil {
			c.logger.Error("error closing writer", "error", err)
		}
		c.logger.Debug("SSE client connection closed")
	})
	return nil
}

// Done returns a channel that's closed when the client disconnects.
func (c *Client) Done() <-chan struct{} {
	return c.done
}

// Context returns the client's context.
func (c *Client) Context() context.Context {
	return c.ctx
}
