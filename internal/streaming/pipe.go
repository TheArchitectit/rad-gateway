package streaming

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"

	"radgateway/internal/logger"
)

// Pipe represents a bidirectional streaming pipe that connects a provider
// stream to a client stream with backpressure handling.
type Pipe struct {
	// Input receives chunks from the provider
	Input chan *Chunk

	// Output sends chunks to the client
	Output chan *Chunk

	// Errors receives any errors that occur during streaming
	Errors chan error

	// Done is closed when the pipe is fully closed
	Done chan struct{}

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	closed atomic.Bool
	logger *slog.Logger

	// buffer for handling backpressure
	buffer     []*Chunk
	bufferSize int
	maxBuffer  int
}

// PipeConfig configures the pipe behavior.
type PipeConfig struct {
	// BufferSize is the number of chunks to buffer (default: 100)
	BufferSize int
	// EnableBackpressure enables backpressure handling (default: true)
	EnableBackpressure bool
}

// DefaultPipeConfig returns a default pipe configuration.
func DefaultPipeConfig() PipeConfig {
	return PipeConfig{
		BufferSize:         100,
		EnableBackpressure: true,
	}
}

// NewPipe creates a new streaming pipe.
func NewPipe(cfg PipeConfig) *Pipe {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 100
	}

	ctx, cancel := context.WithCancel(context.Background())

	p := &Pipe{
		Input:     make(chan *Chunk, cfg.BufferSize),
		Output:    make(chan *Chunk, cfg.BufferSize),
		Errors:    make(chan error, 1),
		Done:      make(chan struct{}),
		ctx:       ctx,
		cancel:    cancel,
		maxBuffer: cfg.BufferSize,
		logger:    logger.WithComponent("streaming"),
	}

	p.logger.Debug("pipe created", "buffer_size", cfg.BufferSize)

	// Start the pipe relay
	go p.run()

	return p
}

// run is the main relay loop that moves chunks from Input to Output
// with backpressure handling.
func (p *Pipe) run() {
	defer close(p.Done)
	defer close(p.Output)
	defer close(p.Errors)

	p.logger.Debug("pipe relay started")
	chunkCount := 0

	for {
		select {
		case <-p.ctx.Done():
			p.logger.Debug("pipe context cancelled, draining buffer", "chunks_processed", chunkCount)
			// Drain remaining chunks
			p.drainBuffer()
			p.logger.Debug("pipe relay stopped")
			return

		case chunk, ok := <-p.Input:
			if !ok {
				p.logger.Debug("pipe input closed, draining buffer", "chunks_processed", chunkCount)
				// Input closed, drain buffer and exit
				p.drainBuffer()
				p.logger.Debug("pipe relay stopped")
				return
			}

			if chunk == nil {
				continue
			}

			chunkCount++

			// Try to send to output with backpressure handling
			if err := p.sendWithBackpressure(chunk); err != nil {
				p.logger.Error("pipe backpressure error", err, "chunk_id", chunk.ID)
				p.sendError(err)
				return
			}
		}
	}
}

// sendWithBackpressure sends a chunk to the output with backpressure handling.
func (p *Pipe) sendWithBackpressure(chunk *Chunk) error {
	// First, try buffered chunks
	if len(p.buffer) > 0 {
		// Try to send buffered chunks first
		for len(p.buffer) > 0 {
			select {
			case p.Output <- p.buffer[0]:
				p.buffer = p.buffer[1:]
			default:
				goto bufferCurrent
			}
		}
	}

bufferCurrent:
	// Try to send current chunk
	select {
	case p.Output <- chunk:
		p.logger.Debug("chunk sent to output", "chunk_id", chunk.ID, "buffer_pending", len(p.buffer))
		return nil
	default:
		// Buffer is full, add to internal buffer if space allows
		if len(p.buffer) < p.maxBuffer {
			p.buffer = append(p.buffer, chunk)
			p.logger.Debug("chunk buffered", "chunk_id", chunk.ID, "buffer_size", len(p.buffer))
			return nil
		}
		// Buffer overflow - drop oldest chunk
		p.logger.Warn("pipe buffer overflow, dropping oldest chunk", "chunk_id", chunk.ID, "buffer_size", len(p.buffer))
		p.buffer = append(p.buffer[1:], chunk)
		return fmt.Errorf("pipe buffer overflow: dropped oldest chunk")
	}
}

// drainBuffer sends any remaining buffered chunks.
func (p *Pipe) drainBuffer() {
	for _, chunk := range p.buffer {
		select {
		case p.Output <- chunk:
		default:
			// Output is blocked, continue to avoid deadlock
		}
	}
	p.buffer = nil
}

// sendError sends an error to the Errors channel if possible.
func (p *Pipe) sendError(err error) {
	select {
	case p.Errors <- err:
	default:
		// Error channel is full, log and continue
	}
}

// Close closes the pipe and releases resources.
func (p *Pipe) Close() error {
	if p.closed.CompareAndSwap(false, true) {
		p.logger.Debug("closing pipe")
		p.cancel()
		close(p.Input)
		<-p.Done
		p.logger.Debug("pipe closed")
	}
	return nil
}

// IsClosed returns true if the pipe is closed.
func (p *Pipe) IsClosed() bool {
	return p.closed.Load()
}

// Context returns the pipe's context.
func (p *Pipe) Context() context.Context {
	return p.ctx
}

// Stream represents a high-level streaming connection between
// a provider and a client.
type Stream struct {
	pipe        *Pipe
	client      *Client
	transformer *Transformer
	wg          sync.WaitGroup
	mu          sync.RWMutex
	err         error
	completed   bool
	logger      *slog.Logger
}

// NewStream creates a new stream with the given client and transformer.
func NewStream(client *Client, transformer *Transformer) *Stream {
	return &Stream{
		pipe:        NewPipe(DefaultPipeConfig()),
		client:      client,
		transformer: transformer,
		logger:      logger.WithComponent("streaming"),
	}
}

// StartFromReader starts streaming from a provider reader.
// It parses SSE events, transforms them, and sends to the client.
func (s *Stream) StartFromReader(reader io.Reader) {
	parser := NewParser(reader)

	s.logger.Debug("starting stream from reader")
	s.wg.Add(2)

	// Goroutine 1: Parse and transform provider events
	go func() {
		defer s.wg.Done()
		defer close(s.pipe.Input)

		eventCount := 0
		for {
			event, err := parser.Next()
			if err != nil {
				if errors.Is(err, io.EOF) {
					s.logger.Debug("stream reader reached EOF", "events_parsed", eventCount)
					return
				}
				s.logger.Error("parse error in stream", err, "events_parsed", eventCount)
				s.setError(fmt.Errorf("parse error: %w", err))
				return
			}

			eventCount++
			s.logger.Debug("SSE event parsed", "event_type", event.Event, "event_id", event.ID)

			chunk, err := s.transformer.Transform(event)
			if err != nil {
				s.logger.Error("transform error in stream", err, "event_id", event.ID)
				s.setError(fmt.Errorf("transform error: %w", err))
				return
			}

			if chunk != nil {
				s.logger.Debug("chunk transformed", "chunk_id", chunk.ID, "finished", chunk.IsFinished)
				select {
				case s.pipe.Input <- chunk:
				case <-s.pipe.Context().Done():
					s.logger.Debug("stream context cancelled during input")
					return
				}
			}
		}
	}()

	// Goroutine 2: Send chunks to client
	go func() {
		defer s.wg.Done()

		chunkCount := 0
		for {
			select {
			case chunk, ok := <-s.pipe.Output:
				if !ok {
					s.logger.Debug("stream output closed, marking complete", "chunks_sent", chunkCount)
					s.markComplete()
					return
				}

				chunkCount++
				if err := s.sendChunk(chunk); err != nil {
					s.logger.Error("send chunk error", err, "chunk_id", chunk.ID, "chunks_sent", chunkCount)
					s.setError(fmt.Errorf("send error: %w", err))
					return
				}
				s.logger.Debug("chunk sent to client", "chunk_id", chunk.ID, "chunks_sent", chunkCount)

			case err := <-s.pipe.Errors:
				s.logger.Error("pipe error received", err, "chunks_sent", chunkCount)
				s.setError(err)
				return

			case <-s.client.Context().Done():
				s.logger.Debug("client context done", "chunks_sent", chunkCount)
				s.setError(s.client.Context().Err())
				return
			}
		}
	}()

	s.logger.Debug("stream started", "pipe_buffer", s.pipe.maxBuffer)
}

// sendChunk sends a single chunk to the client as an SSE event.
func (s *Stream) sendChunk(chunk *Chunk) error {
	// Convert chunk to SSE format
	data := chunkToJSON(chunk)

	return s.client.Send(Event{
		ID:    chunk.ID,
		Event: "message",
		Data:  data,
	})
}

// chunkToJSON converts a chunk to its JSON representation.
func chunkToJSON(chunk *Chunk) string {
	// Use OpenAI format by default
	data := chunk.ToOpenAIFormat()

	// Add done marker if finished
	if chunk.IsFinished && data != nil {
		// Final chunk in OpenAI format
		return string(mustMarshal(data))
	}

	return string(mustMarshal(data))
}

// mustMarshal marshals v to JSON, panicking on error (should not happen for our types).
func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("json marshal: %v", err))
	}
	return b
}

// Wait blocks until the stream completes or errors.
func (s *Stream) Wait() error {
	s.wg.Wait()
	return s.Error()
}

// Error returns any error that occurred during streaming.
func (s *Stream) Error() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.err
}

// setError sets the error for the stream.
func (s *Stream) setError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err == nil {
		s.err = err
	}
}

// markComplete marks the stream as completed.
func (s *Stream) markComplete() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.completed = true
}

// IsComplete returns true if the stream completed successfully.
func (s *Stream) IsComplete() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.completed
}

// Close closes the stream.
func (s *Stream) Close() error {
	s.logger.Debug("closing stream")
	s.pipe.Close()
	s.wg.Wait()
	err := s.client.Close()
	if err != nil {
		s.logger.Error("error closing client", err)
	}
	s.logger.Debug("stream closed")
	return err
}

// StreamHandler is an HTTP handler that manages streaming connections.
type StreamHandler struct {
	// CreateTransformer creates a transformer for the given provider and model
	CreateTransformer func(provider, model string) *Transformer
	// MaxConcurrent limits concurrent streams (0 = unlimited)
	MaxConcurrent int32

	activeStreams atomic.Int32
}

// NewStreamHandler creates a new stream handler.
func NewStreamHandler(createTransformer func(provider, model string) *Transformer) *StreamHandler {
	return &StreamHandler{
		CreateTransformer: createTransformer,
	}
}

// CanAccept returns true if a new stream can be accepted.
func (h *StreamHandler) CanAccept() bool {
	if h.MaxConcurrent <= 0 {
		return true
	}
	return h.activeStreams.Load() < h.MaxConcurrent
}

// HandleStream handles an HTTP request as a streaming response.
// It creates a client, initializes the stream, and returns control to the caller
// who must call StartFromReader to begin streaming.
func (h *StreamHandler) HandleStream(w http.ResponseWriter, r *http.Request, provider, model string) (*Stream, error) {
	log := logger.WithComponent("streaming")

	if !h.CanAccept() {
		log.Warn("rejecting stream: too many concurrent streams", "active", h.activeStreams.Load(), "max", h.MaxConcurrent)
		http.Error(w, "too many concurrent streams", http.StatusServiceUnavailable)
		return nil, errors.New("too many concurrent streams")
	}

	h.activeStreams.Add(1)
	log.Debug("stream accepted", "provider", provider, "model", model, "active_streams", h.activeStreams.Load())

	client, err := NewClient(w, r)
	if err != nil {
		h.activeStreams.Add(-1)
		log.Error("failed to create client", err)
		return nil, fmt.Errorf("create client: %w", err)
	}

	transformer := h.CreateTransformer(provider, model)
	stream := NewStream(client, transformer)

	// Decrement active count when stream closes
	go func() {
		<-stream.pipe.Done
		active := h.activeStreams.Add(-1)
		log.Debug("stream completed", "active_streams", active)
	}()

	return stream, nil
}

// ActiveStreams returns the number of currently active streams.
func (h *StreamHandler) ActiveStreams() int32 {
	return h.activeStreams.Load()
}

// ResponseWriter is a wrapper for http.ResponseWriter that supports streaming.
type ResponseWriter struct {
	http.ResponseWriter
	isStreaming bool
}

// NewResponseWriter creates a new streaming response writer.
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{ResponseWriter: w}
}

// EnableStreaming marks this response as streaming.
func (rw *ResponseWriter) EnableStreaming() {
	rw.isStreaming = true
}

// IsStreaming returns true if this is a streaming response.
func (rw *ResponseWriter) IsStreaming() bool {
	return rw.isStreaming
}
