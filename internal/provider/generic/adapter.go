// Package generic provides a generic HTTP adapter for OpenAI-compatible APIs.
// Sprint 8.3: Generic HTTP Adapter for custom providers
package generic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"log/slog"

	"radgateway/internal/logger"
	"radgateway/internal/models"
)

const (
	defaultTimeout    = 60 * time.Second
	defaultMaxRetries = 3
	defaultRetryDelay = 500 * time.Millisecond
	maxRetryDelay     = 8 * time.Second
)

// Config holds configuration for the generic HTTP adapter.
type Config struct {
	BaseURL       string
	APIKey        string
	Timeout       time.Duration
	MaxRetries    int
	RetryDelay    time.Duration
	HTTPClient    *http.Client
	Headers       map[string]string
	AuthType      string // "bearer", "api-key", "custom"
	AuthHeader    string // Header name for auth (e.g., "Authorization", "x-api-key")
	AuthPrefix    string // Prefix for auth token (e.g., "Bearer ", "")
}

// Adapter implements a generic HTTP adapter for OpenAI-compatible APIs.
// This can be used for any provider that follows the OpenAI API format.
type Adapter struct {
	config     Config
	httpClient *http.Client
	log        *slog.Logger
}

// AdapterOption configures the Adapter.
type AdapterOption func(*Adapter)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) AdapterOption {
	return func(a *Adapter) {
		a.httpClient = client
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) AdapterOption {
	return func(a *Adapter) {
		if a.httpClient == nil {
			a.httpClient = &http.Client{Timeout: timeout}
		} else {
			a.httpClient.Timeout = timeout
		}
	}
}

// WithBaseURL sets the base URL for the API.
func WithBaseURL(url string) AdapterOption {
	return func(a *Adapter) {
		a.config.BaseURL = url
	}
}

// WithRetryConfig sets retry configuration.
func WithRetryConfig(maxRetries int, retryDelay time.Duration) AdapterOption {
	return func(a *Adapter) {
		a.config.MaxRetries = maxRetries
		a.config.RetryDelay = retryDelay
	}
}

// WithHeaders sets additional headers for requests.
func WithHeaders(headers map[string]string) AdapterOption {
	return func(a *Adapter) {
		for k, v := range headers {
			a.config.Headers[k] = v
		}
	}
}

// WithAuthType sets the authentication type.
func WithAuthType(authType, header, prefix string) AdapterOption {
	return func(a *Adapter) {
		a.config.AuthType = authType
		a.config.AuthHeader = header
		a.config.AuthPrefix = prefix
	}
}

// NewAdapter creates a new generic HTTP adapter.
func NewAdapter(baseURL, apiKey string, opts ...AdapterOption) *Adapter {
	a := &Adapter{
		config: Config{
			BaseURL:    baseURL,
			APIKey:     apiKey,
			Timeout:    defaultTimeout,
			MaxRetries: defaultMaxRetries,
			RetryDelay: defaultRetryDelay,
			Headers:    make(map[string]string),
			AuthType:   "bearer",
			AuthHeader: "Authorization",
			AuthPrefix: "Bearer ",
		},
		log: logger.WithComponent("generic"),
	}

	for _, opt := range opts {
		opt(a)
	}

	if a.httpClient == nil {
		a.httpClient = &http.Client{
			Timeout: a.config.Timeout,
		}
	}

	return a
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "generic"
}

// Execute sends a request to the API and returns the result.
func (a *Adapter) Execute(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
	switch req.APIType {
	case "chat":
		return a.executeChat(ctx, req, model)
	case "embeddings":
		return a.executeEmbeddings(ctx, req, model)
	default:
		return models.ProviderResult{}, fmt.Errorf("unsupported api type: %s", req.APIType)
	}
}

func (a *Adapter) executeChat(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
	payload, ok := req.Payload.(models.ChatCompletionRequest)
	if !ok {
		return models.ProviderResult{}, fmt.Errorf("invalid chat payload type: %T", req.Payload)
	}

	// Override model with the one from routing
	payload.Model = model

	if payload.Stream {
		return a.executeStreaming(ctx, payload)
	}

	return a.executeNonStreaming(ctx, payload)
}

func (a *Adapter) executeNonStreaming(ctx context.Context, req models.ChatCompletionRequest) (models.ProviderResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return models.ProviderResult{}, fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", a.config.BaseURL)

	var resp *http.Response
	var lastErr error

	// Retry loop
	for attempt := 0; attempt < a.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			delay := a.config.RetryDelay * time.Duration(1<<uint(attempt-1))
			if delay > maxRetryDelay {
				delay = maxRetryDelay
			}
			select {
			case <-ctx.Done():
				return models.ProviderResult{}, ctx.Err()
			case <-time.After(delay):
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return models.ProviderResult{}, fmt.Errorf("creating request: %w", err)
		}

		a.setHeaders(httpReq)

		resp, err = a.httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("http request failed: %w", err)
			a.log.Warn("generic: http request failed, will retry", "attempt", attempt+1, "error", err.Error())
			continue // Retry on network errors
		}

		// Don't retry on certain status codes
		if resp.StatusCode == http.StatusUnauthorized ||
			resp.StatusCode == http.StatusForbidden ||
			resp.StatusCode == http.StatusBadRequest {
			defer resp.Body.Close()
			a.log.Error("generic: request failed with client error", "status", resp.StatusCode)
			return a.handleErrorResponse(resp)
		}

		// Retry on server errors or rate limiting
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error (status %d)", resp.StatusCode)
			a.log.Warn("generic: server error, will retry", "attempt", attempt+1, "status", resp.StatusCode)
			continue
		}

		break
	}

	if resp == nil {
		a.log.Error("generic: all retries exhausted", "error", lastErr.Error())
		return models.ProviderResult{}, fmt.Errorf("all retries exhausted: %w", lastErr)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return a.handleErrorResponse(resp)
	}

	var chatResp models.ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return models.ProviderResult{}, fmt.Errorf("decoding response: %w", err)
	}

	return models.ProviderResult{
		Model:    req.Model,
		Provider: a.Name(),
		Status:   "success",
		Usage:    chatResp.Usage,
		Payload:  chatResp,
	}, nil
}

func (a *Adapter) executeStreaming(ctx context.Context, req models.ChatCompletionRequest) (models.ProviderResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return models.ProviderResult{}, fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", a.config.BaseURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return models.ProviderResult{}, fmt.Errorf("creating request: %w", err)
	}

	a.setHeaders(httpReq)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return models.ProviderResult{}, fmt.Errorf("http request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return a.handleErrorResponse(resp)
	}

	// Create a pipe for streaming
	pr, pw := io.Pipe()

	// Usage tracking for streaming
	var usageMu sync.Mutex
	usage := models.Usage{}
	model := req.Model

	// Process the stream in a goroutine
	go func() {
		defer resp.Body.Close()
		defer pw.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					pw.CloseWithError(fmt.Errorf("reading stream: %w", err))
				}
				return
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return
			}

			var chunk models.ChatCompletionResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				pw.CloseWithError(fmt.Errorf("unmarshaling stream chunk: %w", err))
				return
			}

			// Update usage estimates from streaming chunks
			for _, choice := range chunk.Choices {
				if choice.Message.Content != "" {
					usageMu.Lock()
					usage.CompletionTokens++
					usage.TotalTokens++
					usageMu.Unlock()
				}
			}

			// Forward the chunk
			if _, err := fmt.Fprintf(pw, "data: %s\n\n", data); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
	}()

	return models.ProviderResult{
		Model:    model,
		Provider: a.Name(),
		Status:   "success",
		Usage:    usage,
		Payload:  &StreamingResponse{Reader: pr, usage: &usage, model: model, mu: &usageMu},
	}, nil
}

func (a *Adapter) executeEmbeddings(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
	payload, ok := req.Payload.(models.EmbeddingsRequest)
	if !ok {
		return models.ProviderResult{}, fmt.Errorf("invalid embeddings payload type: %T", req.Payload)
	}

	payload.Model = model

	embeddingsReq := map[string]any{
		"model": payload.Model,
		"input": payload.Input,
	}

	body, err := json.Marshal(embeddingsReq)
	if err != nil {
		return models.ProviderResult{}, fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/embeddings", a.config.BaseURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return models.ProviderResult{}, fmt.Errorf("creating request: %w", err)
	}

	a.setHeaders(httpReq)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return models.ProviderResult{}, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return a.handleErrorResponse(resp)
	}

	var result models.EmbeddingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return models.ProviderResult{}, fmt.Errorf("decoding response: %w", err)
	}

	return models.ProviderResult{
		Model:    model,
		Provider: a.Name(),
		Status:   "success",
		Usage:    result.Usage,
		Payload:  result,
	}, nil
}

func (a *Adapter) setHeaders(req *http.Request) {
	// Set authentication header
	if a.config.APIKey != "" {
		req.Header.Set(a.config.AuthHeader, a.config.AuthPrefix+a.config.APIKey)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set custom headers
	for k, v := range a.config.Headers {
		req.Header.Set(k, v)
	}
}

func (a *Adapter) handleErrorResponse(resp *http.Response) (models.ProviderResult, error) {
	body, _ := io.ReadAll(resp.Body)

	// Try to parse as standard error format
	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		return models.ProviderResult{}, fmt.Errorf("api error: %s", errResp.Error.Message)
	}

	return models.ProviderResult{}, fmt.Errorf("api returned status %d: %s", resp.StatusCode, string(body))
}

// StreamingResponse wraps a stream reader for SSE output.
type StreamingResponse struct {
	Reader io.Reader
	usage  *models.Usage
	mu     *sync.Mutex
	model  string
}

// Read implements io.Reader for streaming responses.
func (s *StreamingResponse) Read(p []byte) (n int, err error) {
	return s.Reader.Read(p)
}

// Usage returns the current usage stats for the stream.
func (s *StreamingResponse) Usage() models.Usage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return *s.usage
}

// Cost returns the calculated cost for the stream usage.
func (s *StreamingResponse) Cost() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Generic adapter doesn't have pricing, return 0
	// Cost tracking would need to be implemented per-provider
	return 0
}

// Close closes the streaming response.
func (s *StreamingResponse) Close() error {
	if closer, ok := s.Reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
