package openai

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
	defaultBaseURL     = "https://api.openai.com/v1"
	defaultTimeout     = 60 * time.Second
	defaultMaxRetries  = 3
	defaultRetryDelay  = 500 * time.Millisecond
	maxRetryDelay      = 8 * time.Second
)

// Config holds configuration for the OpenAI adapter.
type Config struct {
	APIKey      string
	BaseURL     string
	Timeout     time.Duration
	MaxRetries  int
	RetryDelay  time.Duration
	HTTPClient  *http.Client
}

// Adapter implements the provider.Adapter interface for OpenAI-compatible APIs.
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

// WithBaseURL sets a custom base URL for OpenAI-compatible endpoints.
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

// NewAdapter creates a new OpenAI adapter.
func NewAdapter(apiKey string, opts ...AdapterOption) *Adapter {
	a := &Adapter{
		config: Config{
			APIKey:     apiKey,
			BaseURL:    defaultBaseURL,
			Timeout:    defaultTimeout,
			MaxRetries: defaultMaxRetries,
			RetryDelay: defaultRetryDelay,
		},
		token:           apiKey,
		reqTransform:    NewRequestTransformer(),
		respTransform:   NewResponseTransformer(),
		streamTransform: NewStreamTransformer(),
		log:             logger.WithComponent("openai"),
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
	return "openai"
}

// Execute sends a request to the OpenAI API and returns the result.
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

	// Transform to OpenAI format
	openaiReq := a.reqTransform.Transform(payload)

	if payload.Stream {
		return a.executeStreaming(ctx, openaiReq)
	}

	return a.executeNonStreaming(ctx, openaiReq)
}

func (a *Adapter) executeNonStreaming(ctx context.Context, req OpenAIRequest) (models.ProviderResult, error) {
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
			a.log.Warn("openai: http request failed, will retry", "attempt", attempt+1, "error", err.Error())
			continue // Retry on network errors
		}

		// Don't retry on certain status codes
		if resp.StatusCode == http.StatusUnauthorized ||
			resp.StatusCode == http.StatusForbidden ||
			resp.StatusCode == http.StatusBadRequest {
			defer resp.Body.Close()
			a.log.Error("openai: request failed with client error", "status", resp.StatusCode)
			return a.handleErrorResponse(resp)
		}

		// Retry on server errors or rate limiting
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error (status %d)", resp.StatusCode)
			a.log.Warn("openai: server error, will retry", "attempt", attempt+1, "status", resp.StatusCode)
			continue
		}

		break
	}

	if resp == nil {
		a.log.Error("openai: all retries exhausted", "error", lastErr.Error())
		return models.ProviderResult{}, fmt.Errorf("all retries exhausted: %w", lastErr)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return a.handleErrorResponse(resp)
	}

	var openaiResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return models.ProviderResult{}, fmt.Errorf("decoding response: %w", err)
	}

	result, err := a.respTransform.Transform(openaiResp)
	if err != nil {
		return models.ProviderResult{}, err
	}

	return models.ProviderResult{
		Model:    req.Model,
		Provider: a.Name(),
		Status:   "success",
		Usage:    result.Usage,
		Payload:  result,
	}, nil
}

func (a *Adapter) executeStreaming(ctx context.Context, req OpenAIRequest) (models.ProviderResult, error) {
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

			chunk, err := ParseStreamChunk(data)
			if err != nil {
				pw.CloseWithError(err)
				return
			}

			// Update usage estimates from streaming chunks
			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					usageMu.Lock()
					usage.CompletionTokens++
					usage.TotalTokens++
					usageMu.Unlock()
				}
			}

			// Write the chunk to the pipe
			transformed := a.streamTransform.TransformChunk(*chunk)
			chunkJSON, err := json.Marshal(transformed)
			if err != nil {
				pw.CloseWithError(err)
				return
			}

			if _, err := fmt.Fprintf(pw, "data: %s\n\n", chunkJSON); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
	}()

	return models.ProviderResult{
		Model:    req.Model,
		Provider: a.Name(),
		Status:   "success",
		Usage:    usage,
		Payload:  &StreamingResponse{Reader: pr, usage: &usage, mu: &usageMu},
	}, nil
}

func (a *Adapter) executeEmbeddings(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error) {
	payload, ok := req.Payload.(models.EmbeddingsRequest)
	if !ok {
		return models.ProviderResult{}, fmt.Errorf("invalid embeddings payload type: %T", req.Payload)
	}

	payload.Model = model

	openaiReq := map[string]any{
		"model": payload.Model,
		"input": payload.Input,
	}

	body, err := json.Marshal(openaiReq)
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

	var result struct {
		Object string `json:"object"`
		Data   []struct {
			Object    string    `json:"object"`
			Index     int       `json:"index"`
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
		Model string `json:"model"`
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return models.ProviderResult{}, fmt.Errorf("decoding response: %w", err)
	}

	embeddings := make([]models.Embedding, len(result.Data))
	for i, d := range result.Data {
		embeddings[i] = models.Embedding{
			Object:    d.Object,
			Index:     d.Index,
			Embedding: d.Embedding,
		}
	}

	return models.ProviderResult{
		Model:    model,
		Provider: a.Name(),
		Status:   "success",
		Usage: models.Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: 0,
			TotalTokens:      result.Usage.TotalTokens,
		},
		Payload: models.EmbeddingsResponse{
			Object: result.Object,
			Data:   embeddings,
			Model:  result.Model,
			Usage: models.Usage{
				PromptTokens:     result.Usage.PromptTokens,
				CompletionTokens: 0,
				TotalTokens:      result.Usage.TotalTokens,
			},
		},
	}, nil
}

func (a *Adapter) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

func (a *Adapter) handleErrorResponse(resp *http.Response) (models.ProviderResult, error) {
	body, _ := io.ReadAll(resp.Body)

	var errResp struct {
		Error *OpenAIError `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != nil {
		return models.ProviderResult{}, fmt.Errorf("openai api error: %s", errResp.Error.Message)
	}

	return models.ProviderResult{}, fmt.Errorf("openai api returned status %d: %s", resp.StatusCode, string(body))
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
