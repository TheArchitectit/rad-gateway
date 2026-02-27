// Package anthropic provides an adapter for Anthropic's Claude API.
package anthropic

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
	defaultBaseURL    = "https://api.anthropic.com"
	defaultVersion    = "2023-06-01"
	defaultTimeout    = 60 * time.Second
	defaultMaxRetries = 3
	defaultRetryDelay = 500 * time.Millisecond
	maxRetryDelay     = 8 * time.Second
	maxTokensDefault  = 4096
)

// Config holds configuration for the Anthropic adapter.
type Config struct {
	APIKey     string
	BaseURL    string
	Version    string
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
	HTTPClient *http.Client
}

// Adapter implements the provider.Adapter interface for Anthropic API.
type Adapter struct {
	config          Config
	token           string
	reqTransform    *RequestTransformer
	respTransform   *ResponseTransformer
	streamTransform *StreamTransformer
	httpClient      *http.Client
	log             *slog.Logger
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

// WithBaseURL sets a custom base URL for Anthropic API.
func WithBaseURL(url string) AdapterOption {
	return func(a *Adapter) {
		a.config.BaseURL = url
	}
}

// WithVersion sets the Anthropic API version.
func WithVersion(version string) AdapterOption {
	return func(a *Adapter) {
		a.config.Version = version
	}
}

// WithRetryConfig sets retry configuration.
func WithRetryConfig(maxRetries int, retryDelay time.Duration) AdapterOption {
	return func(a *Adapter) {
		a.config.MaxRetries = maxRetries
		a.config.RetryDelay = retryDelay
	}
}

// NewAdapter creates a new Anthropic adapter.
func NewAdapter(apiKey string, opts ...AdapterOption) *Adapter {
	a := &Adapter{
		config: Config{
			APIKey:     apiKey,
			BaseURL:    defaultBaseURL,
			Version:    defaultVersion,
			Timeout:    defaultTimeout,
			MaxRetries: defaultMaxRetries,
			RetryDelay: defaultRetryDelay,
		},
		token:           apiKey,
		reqTransform:    NewRequestTransformer(),
		respTransform:   NewResponseTransformer(),
		streamTransform: NewStreamTransformer(),
		log:             logger.WithComponent("anthropic"),
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
	return "anthropic"
}

// Execute sends a request to the Anthropic API and returns the result.
func (a *Adapter) Execute(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
	switch req.APIType {
	case "chat":
		return a.executeChat(ctx, req, model)
	case "messages":
		// Anthropic-specific API type
		return a.executeChat(ctx, req, model)
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

	// Transform to Anthropic format
	anthropicReq := a.reqTransform.Transform(payload)

	if payload.Stream {
		return a.executeStreaming(ctx, anthropicReq)
	}

	return a.executeNonStreaming(ctx, anthropicReq)
}

func (a *Adapter) executeNonStreaming(ctx context.Context, req AnthropicRequest) (models.ProviderResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return models.ProviderResult{}, fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/messages", a.config.BaseURL)

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
			a.log.Warn("anthropic: http request failed, will retry", "attempt", attempt+1, "error", err.Error())
			continue // Retry on network errors
		}

		// Don't retry on certain status codes
		if resp.StatusCode == http.StatusUnauthorized ||
			resp.StatusCode == http.StatusForbidden ||
			resp.StatusCode == http.StatusBadRequest {
			defer resp.Body.Close()
			a.log.Error("anthropic: request failed with client error", "status", resp.StatusCode)
			return a.handleErrorResponse(resp)
		}

		// Retry on server errors or rate limiting
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error (status %d)", resp.StatusCode)
			a.log.Warn("anthropic: server error, will retry", "attempt", attempt+1, "status", resp.StatusCode)
			continue
		}

		break
	}

	if resp == nil {
		a.log.Error("anthropic: all retries exhausted", "error", lastErr.Error())
		return models.ProviderResult{}, fmt.Errorf("all retries exhausted: %w", lastErr)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return a.handleErrorResponse(resp)
	}

	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return models.ProviderResult{}, fmt.Errorf("decoding response: %w", err)
	}

	result, err := a.respTransform.Transform(anthropicResp)
	if err != nil {
		return models.ProviderResult{}, err
	}

	// Calculate cost for the request
	cost, err := CalculateCost(req.Model, result.Usage.PromptTokens, result.Usage.CompletionTokens)
	if err != nil {
		a.log.Warn("anthropic: failed to calculate cost", "model", req.Model, "error", err.Error())
		cost = 0
	}
	result.Usage.CostTotal = cost

	return models.ProviderResult{
		Model:    req.Model,
		Provider: a.Name(),
		Status:   "success",
		Usage:    result.Usage,
		Payload:  result,
	}, nil
}

func (a *Adapter) executeStreaming(ctx context.Context, req AnthropicRequest) (models.ProviderResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return models.ProviderResult{}, fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/messages", a.config.BaseURL)

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

	// Reset stream transformer for new stream
	a.streamTransform.Reset()

	// Process the stream in a goroutine
	go func() {
		defer resp.Body.Close()
		defer pw.Close()

		reader := bufio.NewReader(resp.Body)
		var currentEvent string
		var currentData strings.Builder

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					pw.CloseWithError(fmt.Errorf("reading stream: %w", err))
				}
				return
			}

			line = strings.TrimRight(line, "\n")

			// Empty line indicates end of event
			if line == "" {
				if currentData.Len() > 0 {
					data := currentData.String()
					currentData.Reset()

					// Transform the event
					transformed, done, err := a.streamTransform.TransformEvent(currentEvent, []byte(data))
					if err != nil {
						pw.CloseWithError(err)
						return
					}

					if done {
						// Write DONE marker
						if _, err := pw.Write(transformed); err != nil {
							pw.CloseWithError(err)
							return
						}
						// Update usage from accumulated content
						accumulated := a.streamTransform.GetAccumulatedContent()
						usageMu.Lock()
						usage.CompletionTokens = len(accumulated) / 4 // Rough estimate
						usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
						usageMu.Unlock()
						return
					}

					if transformed != nil {
						// Update usage tracking from content
						if _, err := pw.Write(transformed); err != nil {
							pw.CloseWithError(err)
							return
						}
					}
				}
				currentEvent = ""
				continue
			}

			// Parse event and data lines
			if strings.HasPrefix(line, "event: ") {
				currentEvent = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				if currentData.Len() > 0 {
					currentData.WriteString("\n")
				}
				currentData.WriteString(strings.TrimPrefix(line, "data: "))
			}
		}
	}()

	return models.ProviderResult{
		Model:    req.Model,
		Provider: a.Name(),
		Status:   "success",
		Usage:    usage,
		Payload:  &StreamingResponse{Reader: pr, usage: &usage, model: req.Model, mu: &usageMu},
	}, nil
}

func (a *Adapter) setHeaders(req *http.Request) {
	// Anthropic uses x-api-key instead of Authorization: Bearer
	req.Header.Set("x-api-key", a.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("anthropic-version", a.config.Version)
}

func (a *Adapter) handleErrorResponse(resp *http.Response) (models.ProviderResult, error) {
	body, _ := io.ReadAll(resp.Body)

	// Try to parse Anthropic error format
	if err := TransformErrorResponse(body); err != nil {
		return models.ProviderResult{}, err
	}

	return models.ProviderResult{}, fmt.Errorf("anthropic api returned status %d: %s", resp.StatusCode, string(body))
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

// Close closes the streaming response.
func (s *StreamingResponse) Close() error {
	if closer, ok := s.Reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Cost returns the calculated cost for the stream usage.
func (s *StreamingResponse) Cost() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	cost, _ := CalculateCost(s.model, s.usage.PromptTokens, s.usage.CompletionTokens)
	return cost
}
