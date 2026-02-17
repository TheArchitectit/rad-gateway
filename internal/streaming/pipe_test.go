package streaming

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewPipe(t *testing.T) {
	tests := []struct {
		name       string
		config     PipeConfig
		wantBuffer int
	}{
		{
			name:       "default config",
			config:     DefaultPipeConfig(),
			wantBuffer: 100,
		},
		{
			name: "custom buffer size",
			config: PipeConfig{
				BufferSize:         50,
				EnableBackpressure: true,
			},
			wantBuffer: 50,
		},
		{
			name: "zero buffer size uses default",
			config: PipeConfig{
				BufferSize:         0,
				EnableBackpressure: true,
			},
			wantBuffer: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPipe(tt.config)
			defer p.Close()

			if p.maxBuffer != tt.wantBuffer {
				t.Errorf("maxBuffer = %d, want %d", p.maxBuffer, tt.wantBuffer)
			}
			if p.Input == nil {
				t.Error("Input channel is nil")
			}
			if p.Output == nil {
				t.Error("Output channel is nil")
			}
			if p.Errors == nil {
				t.Error("Errors channel is nil")
			}
			if p.Done == nil {
				t.Error("Done channel is nil")
			}
		})
	}
}

func TestPipe_BasicRelay(t *testing.T) {
	p := NewPipe(DefaultPipeConfig())

	var received []*Chunk
	done := make(chan struct{})

	go func() {
		for chunk := range p.Output {
			received = append(received, chunk)
		}
		close(done)
	}()

	p.Input <- &Chunk{ID: "1"}
	time.Sleep(50 * time.Millisecond)
	p.Close()

	select {
	case <-done:
		if len(received) != 1 {
			t.Errorf("received %d chunks, want 1", len(received))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestPipe_NilChunk(t *testing.T) {
	p := NewPipe(DefaultPipeConfig())

	var received []*Chunk
	done := make(chan struct{})

	go func() {
		for chunk := range p.Output {
			if chunk != nil {
				received = append(received, chunk)
			}
		}
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	p.Input <- nil
	p.Input <- &Chunk{ID: "1"}
	close(p.Input)

	select {
	case <-done:
		if len(received) != 1 {
			t.Errorf("received %d chunks, want 1", len(received))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestPipe_Close(t *testing.T) {
	p := NewPipe(DefaultPipeConfig())

	if p.IsClosed() {
		t.Error("pipe should not be closed initially")
	}

	p.Close()

	if !p.IsClosed() {
		t.Error("pipe should be closed after Close()")
	}

	p.Close()
}

func TestPipe_Context(t *testing.T) {
	p := NewPipe(DefaultPipeConfig())

	ctx := p.Context()
	if ctx == nil {
		t.Fatal("Context() returned nil")
	}

	select {
	case <-ctx.Done():
		t.Error("context should not be done initially")
	default:
	}

	p.Close()

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Error("context should be done after Close()")
	}
}

func TestPipe_sendError(t *testing.T) {
	p := NewPipe(DefaultPipeConfig())
	defer p.Close()

	p.Errors <- errors.New("existing error")
	p.sendError(errors.New("new error"))

	select {
	case <-p.Errors:
	case <-time.After(100 * time.Millisecond):
	}
}

func TestPipe_ConcurrentWrites(t *testing.T) {
	p := NewPipe(PipeConfig{BufferSize: 100})
	defer p.Close()

	var wg sync.WaitGroup
	numProducers := 5
	numChunks := 20

	for i := 0; i < numProducers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numChunks; j++ {
				p.Input <- &Chunk{ID: string(rune('A' + id))}
			}
		}(i)
	}

	go func() {
		wg.Wait()
		p.closeInput()
	}()

	received := 0
	for range p.Output {
		received++
	}

	expected := numProducers * numChunks
	if received != expected {
		t.Errorf("expected %d chunks, got %d", expected, received)
	}
}

func TestNewStream(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)

	client, err := NewClient(rr, req)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	transformer := NewTransformer("openai", "gpt-4")
	stream := NewStream(client, transformer)

	if stream == nil {
		t.Fatal("NewStream returned nil")
	}
	if stream.pipe == nil {
		t.Error("stream.pipe is nil")
	}
	if stream.client != client {
		t.Error("stream.client mismatch")
	}
	if stream.transformer != transformer {
		t.Error("stream.transformer mismatch")
	}
}

func TestStream_StartFromReader(t *testing.T) {
	sseData := "data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"2\",\"object\":\"chat.completion.chunk\",\"created\":1234567891,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" World\"},\"finish_reason\":\"stop\"}]}\n\n"

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)

	client, err := NewClient(rr, req)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	transformer := NewTransformer("openai", "gpt-4")
	stream := NewStream(client, transformer)

	stream.StartFromReader(strings.NewReader(sseData))

	done := make(chan error, 1)
	go func() {
		done <- stream.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Logf("Wait() returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for stream")
	}

	client.Close()
}

func TestStream_Error(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)

	client, err := NewClient(rr, req)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	transformer := NewTransformer("openai", "gpt-4")
	stream := NewStream(client, transformer)

	if stream.Error() != nil {
		t.Error("expected no error initially")
	}

	testErr := errors.New("test error")
	stream.setError(testErr)

	if stream.Error() != testErr {
		t.Errorf("Error() = %v, want %v", stream.Error(), testErr)
	}

	stream.setError(errors.New("new error"))
	if stream.Error() != testErr {
		t.Error("error should not be overwritten")
	}
}

func TestStream_IsComplete(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)

	client, err := NewClient(rr, req)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	transformer := NewTransformer("openai", "gpt-4")
	stream := NewStream(client, transformer)

	if stream.IsComplete() {
		t.Error("stream should not be complete initially")
	}

	stream.markComplete()

	if !stream.IsComplete() {
		t.Error("stream should be complete after markComplete()")
	}
}

func TestStream_sendChunk(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)

	client, err := NewClient(rr, req)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	transformer := NewTransformer("openai", "gpt-4")
	stream := NewStream(client, transformer)

	chunk := &Chunk{
		ID:      "test-chunk",
		Object:  "chat.completion.chunk",
		Created: 1234567890,
		Model:   "gpt-4",
		Choices: []ChunkChoice{
			{
				Index: 0,
				Delta: Delta{
					Content: "test content",
				},
			},
		},
	}

	err = stream.sendChunk(chunk)
	if err != nil {
		t.Errorf("sendChunk() error = %v", err)
	}

	rr.Flush()
	body := rr.Body.String()
	if !strings.Contains(body, "test content") {
		t.Errorf("body should contain 'test content', got: %s", body)
	}
}

func TestStream_GoroutineCoordination(t *testing.T) {
	var sseData bytes.Buffer
	for i := 0; i < 50; i++ {
		sseData.WriteString("data: {\"id\":\"" + string(rune('a'+i%26)) + "\"}\n\n")
	}
	sseData.WriteString("data: [DONE]\n\n")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)

	client, err := NewClient(rr, req)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	transformer := NewTransformer("openai", "gpt-4")
	stream := NewStream(client, transformer)

	var wg sync.WaitGroup
	var completed atomic.Bool

	wg.Add(1)
	go func() {
		defer wg.Done()
		stream.StartFromReader(&sseData)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := stream.Wait()
		if err == nil {
			completed.Store(true)
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if !completed.Load() {
			t.Error("stream did not complete successfully")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for goroutines")
	}

	client.Close()
}

func TestNewStreamHandler(t *testing.T) {
	createTransformer := func(provider, model string) *Transformer {
		return NewTransformer(provider, model)
	}

	handler := NewStreamHandler(createTransformer)

	if handler == nil {
		t.Fatal("NewStreamHandler returned nil")
	}
	if handler.CreateTransformer == nil {
		t.Error("CreateTransformer is nil")
	}
}

func TestStreamHandler_CanAccept(t *testing.T) {
	createTransformer := func(provider, model string) *Transformer {
		return NewTransformer(provider, model)
	}

	tests := []struct {
		name          string
		maxConcurrent int32
		activeStreams int32
		want          bool
	}{
		{
			name:          "unlimited - zero max",
			maxConcurrent: 0,
			activeStreams: 100,
			want:          true,
		},
		{
			name:          "under limit",
			maxConcurrent: 10,
			activeStreams: 5,
			want:          true,
		},
		{
			name:          "at limit",
			maxConcurrent: 10,
			activeStreams: 10,
			want:          false,
		},
		{
			name:          "over limit",
			maxConcurrent: 10,
			activeStreams: 15,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewStreamHandler(createTransformer)
			handler.MaxConcurrent = tt.maxConcurrent
			handler.activeStreams.Store(tt.activeStreams)

			got := handler.CanAccept()
			if got != tt.want {
				t.Errorf("CanAccept() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStreamHandler_ActiveStreams(t *testing.T) {
	createTransformer := func(provider, model string) *Transformer {
		return NewTransformer(provider, model)
	}

	handler := NewStreamHandler(createTransformer)

	if handler.ActiveStreams() != 0 {
		t.Errorf("ActiveStreams() = %d, want 0", handler.ActiveStreams())
	}

	handler.activeStreams.Add(5)
	if handler.ActiveStreams() != 5 {
		t.Errorf("ActiveStreams() = %d, want 5", handler.ActiveStreams())
	}
}

func TestStreamHandler_HandleStream(t *testing.T) {
	createTransformer := func(provider, model string) *Transformer {
		return NewTransformer(provider, model)
	}

	tests := []struct {
		name           string
		maxConcurrent  int32
		preAddStreams  int32
		wantErr        bool
		wantStatusCode int
	}{
		{
			name:           "success",
			maxConcurrent:  10,
			preAddStreams:  0,
			wantErr:        false,
			wantStatusCode: 200,
		},
		{
			name:           "too many streams",
			maxConcurrent:  5,
			preAddStreams:  5,
			wantErr:        true,
			wantStatusCode: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewStreamHandler(createTransformer)
			handler.MaxConcurrent = tt.maxConcurrent
			handler.activeStreams.Store(tt.preAddStreams)

			rr := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/stream", nil)

			stream, err := handler.HandleStream(rr, req, "openai", "gpt-4")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				if rr.Code != tt.wantStatusCode {
					t.Errorf("status code = %d, want %d", rr.Code, tt.wantStatusCode)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if stream == nil {
				t.Error("expected stream, got nil")
				return
			}

			if handler.ActiveStreams() != tt.preAddStreams+1 {
				t.Errorf("ActiveStreams() = %d, want %d", handler.ActiveStreams(), tt.preAddStreams+1)
			}

			stream.Close()
		})
	}
}

func TestStreamHandler_ConcurrentLimiting(t *testing.T) {
	createTransformer := func(provider, model string) *Transformer {
		return NewTransformer(provider, model)
	}

	handler := NewStreamHandler(createTransformer)
	handler.MaxConcurrent = 3

	var streams []*Stream
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			rr := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/stream", nil)

			stream, err := handler.HandleStream(rr, req, "openai", "gpt-4")
			if err != nil {
				return
			}

			mu.Lock()
			streams = append(streams, stream)
			mu.Unlock()
		}()
	}

	wg.Wait()

	mu.Lock()
	createdCount := len(streams)
	mu.Unlock()

	if createdCount > int(handler.MaxConcurrent) {
		t.Errorf("created %d streams, max allowed %d", createdCount, handler.MaxConcurrent)
	}

	for _, s := range streams {
		s.Close()
	}
}

func TestNewResponseWriter(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := NewResponseWriter(rr)

	if rw == nil {
		t.Fatal("NewResponseWriter returned nil")
	}
	if rw.ResponseWriter != rr {
		t.Error("ResponseWriter mismatch")
	}
	if rw.IsStreaming() {
		t.Error("should not be streaming initially")
	}
}

func TestResponseWriter_EnableStreaming(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := NewResponseWriter(rr)

	rw.EnableStreaming()

	if !rw.IsStreaming() {
		t.Error("should be streaming after EnableStreaming()")
	}
}

func TestChunkToJSON(t *testing.T) {
	chunk := &Chunk{
		ID:      "test-id",
		Object:  "chat.completion.chunk",
		Created: 1234567890,
		Model:   "gpt-4",
		Choices: []ChunkChoice{
			{
				Index: 0,
				Delta: Delta{
					Content: "Hello",
				},
			},
		},
		IsFinished: true,
	}

	result := chunkToJSON(chunk)

	if !strings.Contains(result, "test-id") {
		t.Error("JSON should contain chunk ID")
	}
	if !strings.Contains(result, "Hello") {
		t.Error("JSON should contain content")
	}
}

func TestMustMarshal(t *testing.T) {
	data := map[string]any{
		"key": "value",
		"num": 42,
	}

	result := mustMarshal(data)
	if len(result) == 0 {
		t.Error("mustMarshal should return non-empty bytes")
	}

	var decoded map[string]any
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Errorf("mustMarshal produced invalid JSON: %v", err)
	}
}

func BenchmarkPipe_Relay(b *testing.B) {
	p := NewPipe(DefaultPipeConfig())
	defer p.Close()

	chunk := &Chunk{ID: "bench", Choices: []ChunkChoice{{Delta: Delta{Content: "benchmark"}}}}

	go func() {
		for range p.Output {
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Input <- chunk
	}
}

func TestMustMarshalPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("mustMarshal should panic on invalid input")
		}
	}()

	// Create a type that cannot be marshaled to JSON
	type unmarshalable struct {
		Ch chan int
	}
	_ = mustMarshal(unmarshalable{Ch: make(chan int)})
}

func TestStream_StartFromReader_ParseError(t *testing.T) {
	sseData := "data: {invalid json\n\n"

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)

	client, err := NewClient(rr, req)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	transformer := NewTransformer("openai", "gpt-4")
	stream := NewStream(client, transformer)

	stream.StartFromReader(strings.NewReader(sseData))

	err = stream.Wait()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	client.Close()
}

func TestStream_StartFromReader_ErrorEvent(t *testing.T) {
	sseData := "event: error\ndata: something went wrong\n\n"

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)

	client, err := NewClient(rr, req)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	transformer := NewTransformer("openai", "gpt-4")
	stream := NewStream(client, transformer)

	stream.StartFromReader(strings.NewReader(sseData))

	err = stream.Wait()
	if err == nil {
		t.Error("expected error for error event")
	}

	client.Close()
}

func TestStream_Close(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)

	client, err := NewClient(rr, req)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	transformer := NewTransformer("openai", "gpt-4")
	stream := NewStream(client, transformer)

	sseData := "data: {\"id\":\"1\"}\n\n"
	stream.StartFromReader(strings.NewReader(sseData))

	time.Sleep(50 * time.Millisecond)

	err = stream.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if stream.pipe != nil && !stream.pipe.IsClosed() {
		t.Error("pipe should be closed after stream.Close()")
	}
}
