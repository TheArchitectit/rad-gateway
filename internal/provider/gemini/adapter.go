// Package gemini provides an adapter for Google's Gemini API.
package gemini

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

	"radgateway/internal/models"
)

const (
	defaultBaseURL    = "https://generativelanguage.googleapis.com"
	defaultVersion    = "v1beta"
	defaultTimeout    = 60 * time.Second
	defaultMaxRetries = 3
	defaultRetryDelay = 500 * time.Millisecond
	maxRetryDelay     = 8 * time.Second
)

// Config holds configuration for the Gemini adapter.
type Config struct {
	APIKey      string
	BaseURL     string
	Version     string
	Timeout     time.Duration
	MaxRetries  int
	RetryDelay  time.Duration
	HTTPClient  *http.Client
	AuthMethod  string // "header" (default) or "query"
}

// Adapter implements the provider.Adapter interface for Gemini API.
type Adapter struct {
	config          Config
	token           string
	reqTransform    *RequestTransformer
	respTransform   *ResponseTransformer
	streamTransform *StreamTransformer
	httpClient      *http.Client
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

// WithBaseURL sets a custom base URL for Gemini API.
func WithBaseURL(url string) AdapterOption {
	return func(a *Adapter) {
		a.config.BaseURL = url
	}
}

// WithVersion sets the Gemini API version.
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

// WithAuthMethod sets the authentication method ("header" or "query").
func WithAuthMethod(method string) AdapterOption {
	return func(a *Adapter) {
		a.config.AuthMethod = method
	}
}

// NewAdapter creates a new Gemini adapter.
func NewAdapter(apiKey string, opts ...AdapterOption) *Adapter {
	a := &Adapter{
		config: Config{
			APIKey:     apiKey,
			BaseURL:    defaultBaseURL,
			Version:    defaultVersion,
			Timeout:    defaultTimeout,
			MaxRetries: defaultMaxRetries,
			RetryDelay: defaultRetryDelay,
			AuthMethod: "header", // Default to header auth
		},
		token:           apiKey,
		reqTransform:    NewRequestTransformer(),
		respTransform:   NewResponseTransformer(),
		streamTransform: NewStreamTransformer(),
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
	return "gemini"
}

// Execute sends a request to the Gemini API and returns the result.
func (a *Adapter) Execute(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
	switch req.APIType {
	case "chat":
		return a.executeChat(ctx, req, model)
	case "gemini":
		// Gemini-specific API type for direct access
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

	// Transform to Gemini format
	geminiReq := a.reqTransform.Transform(payload)

	if payload.Stream {
		return a.executeStreaming(ctx, geminiReq, model)
	}

	return a.executeNonStreaming(ctx, geminiReq, model)
}

func (a *Adapter) executeNonStreaming(ctx context.Context, req GeminiRequest, model string) (models.ProviderResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return models.ProviderResult{}, fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/%s/models/%s:generateContent", a.config.BaseURL, a.config.Version, model)

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
		a.setAuth(httpReq, url)

		resp, err = a.httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("http request failed: %w", err)
			continue // Retry on network errors
		}

		// Don't retry on certain status codes
		if resp.StatusCode == http.StatusUnauthorized ||
			resp.StatusCode == http.StatusForbidden ||
			resp.StatusCode == http.StatusBadRequest {
			defer resp.Body.Close()
			return a.handleErrorResponse(resp)
		}

		// Retry on server errors or rate limiting
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error (status %d)", resp.StatusCode)
			continue
		}

		break
	}

	if resp == nil {
		return models.ProviderResult{}, fmt.Errorf("all retries exhausted: %w", lastErr)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return a.handleErrorResponse(resp)
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return models.ProviderResult{}, fmt.Errorf("decoding response: %w", err)
	}

	result, err := a.respTransform.Transform(geminiResp, model)
	if err != nil {
		return models.ProviderResult{}, err
	}

	return models.ProviderResult{
		Model:    model,
		Provider: a.Name(),
		Status:   "success",
		Usage:    result.Usage,
		Payload:  result,
	}, nil
}

func (a *Adapter) executeStreaming(ctx context.Context, req GeminiRequest, model string) (models.ProviderResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return models.ProviderResult{}, fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/%s/models/%s:streamGenerateContent", a.config.BaseURL, a.config.Version, model)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return models.ProviderResult{}, fmt.Errorf("creating request: %w", err)
	}

	a.setHeaders(httpReq)
	a.setAuth(httpReq, url)

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
	a.streamTransform.Init(model)

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

			// Parse and transform the chunk
			transformed, done, err := a.streamTransform.TransformChunk(data)
			if err != nil {
				pw.CloseWithError(err)
				return
			}

			if done {
				// Write DONE marker
				if _, err := pw.Write([]byte("data: [DONE]\n\n")); err != nil {
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
				// Write the transformed chunk (already SSE formatted)
				if _, err := pw.Write(transformed); err != nil {
					pw.CloseWithError(err)
					return
				}
			}
		}
	}()

	return models.ProviderResult{
		Model:    model,
		Provider: a.Name(),
		Status:   "success",
		Usage:    usage,
		Payload:  &StreamingResponse{Reader: pr, usage: &usage, mu: &usageMu},
	}, nil
}

func (a *Adapter) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

func (a *Adapter) setAuth(req *http.Request, url string) {
	switch a.config.AuthMethod {
	case "query":
		// Use query parameter fallback
		// Note: This would need to be handled at URL construction time
		// since http.Request doesn't allow modifying URL query params here
		// For now, we set the header as preferred method
		req.Header.Set("x-goog-api-key", a.token)
	default:
		// Preferred method: x-goog-api-key header
		req.Header.Set("x-goog-api-key", a.token)
	}
}

func (a *Adapter) handleErrorResponse(resp *http.Response) (models.ProviderResult, error) {
	body, _ := io.ReadAll(resp.Body)

	// Try to parse Gemini error format
	var geminiErr struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &geminiErr); err == nil && geminiErr.Error.Message != "" {
		return models.ProviderResult{}, fmt.Errorf("gemini api error (%d): %s", geminiErr.Error.Code, geminiErr.Error.Message)
	}

	return models.ProviderResult{}, fmt.Errorf("gemini api returned status %d: %s", resp.StatusCode, string(body))
}

// StreamingResponse wraps a stream reader for SSE output.
type StreamingResponse struct {
	Reader io.Reader
	usage  *models.Usage
	mu     *sync.Mutex
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

// buildEndpointPath constructs the Gemini API endpoint path.
func buildEndpointPath(model string, streaming bool, version string) string {
	endpoint := ":generateContent"
	if streaming {
		endpoint = ":streamGenerateContent"
	}
	return fmt.Sprintf("/%s/models/%s%s", version, model, endpoint)
}
