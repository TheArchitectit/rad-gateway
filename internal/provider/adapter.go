// Package provider provides adapter interfaces and implementations for LLM providers.
// It supports OpenAI, Anthropic, Gemini, and other providers through a unified
// transformation and execution model.
package provider

import (
	"context"
	"io"
	"net/http"
	"time"
)

// ProviderAdapter defines the core interface for transforming requests and responses
// between the gateway's internal format and provider-specific APIs.
type ProviderAdapter interface {
	// TransformRequest modifies the incoming HTTP request to match the provider's API format.
	// It may rewrite headers, body, URL path, and query parameters.
	TransformRequest(req *http.Request) (*http.Request, error)

	// TransformResponse modifies the provider's HTTP response to match the gateway's standard format.
	// It may transform status codes, headers, and response body.
	TransformResponse(resp *http.Response) (*http.Response, error)

	// GetProviderName returns the unique identifier for this provider (e.g., "openai", "anthropic").
	GetProviderName() string

	// SupportsStreaming returns true if this provider supports SSE streaming responses.
	SupportsStreaming() bool
}

// RequestTransformer defines the interface for transforming HTTP requests.
// Implementations handle provider-specific request formatting.
type RequestTransformer interface {
	// TransformHeaders modifies request headers including auth, content-type, and provider-specific headers.
	TransformHeaders(req *http.Request) error

	// TransformBody rewrites the request body from internal format to provider format.
	// Returns the new body content and content type.
	TransformBody(body io.Reader, contentType string) (io.Reader, string, error)

	// TransformURL modifies the request URL path and query parameters.
	TransformURL(req *http.Request) error
}

// ResponseTransformer defines the interface for transforming HTTP responses.
// Implementations handle provider-specific response normalization.
type ResponseTransformer interface {
	// TransformHeaders modifies response headers for the gateway's standard format.
	TransformHeaders(resp *http.Response) error

	// TransformBody rewrites the response body from provider format to internal format.
	TransformBody(body io.Reader, contentType string) (io.Reader, string, error)

	// TransformStatusCode normalizes provider status codes to gateway standard codes.
	TransformStatusCode(code int) int
}

// StreamTransformer handles streaming (SSE) response transformations.
type StreamTransformer interface {
	// TransformStreamChunk processes a single SSE event chunk.
	// Returns the transformed chunk or nil if the chunk should be dropped.
	TransformStreamChunk(chunk []byte) ([]byte, error)

	// IsDoneMarker checks if the chunk indicates the end of the stream.
	IsDoneMarker(chunk []byte) bool
}

// TimeoutConfig defines timeout settings for provider requests.
type TimeoutConfig struct {
	// RequestTimeout is the total timeout for the entire request/response cycle.
	RequestTimeout time.Duration

	// TLSHandshakeTimeout is the timeout for completing the TLS handshake.
	TLSHandshakeTimeout time.Duration

	// ResponseHeaderTimeout is the timeout for receiving the response headers.
	ResponseHeaderTimeout time.Duration

	// IdleConnTimeout is the timeout for idle connections in the pool.
	IdleConnTimeout time.Duration
}

// DefaultTimeoutConfig returns sensible default timeout values.
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		RequestTimeout:        120 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}
}

// AdapterRetryConfig defines retry behavior for failed requests.
type AdapterRetryConfig struct {
	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int

	// InitialBackoff is the initial wait time before the first retry.
	InitialBackoff time.Duration

	// MaxBackoff is the maximum wait time between retries.
	MaxBackoff time.Duration

	// BackoffMultiplier is the factor by which backoff increases after each retry.
	BackoffMultiplier float64

	// RetryableStatusCodes are HTTP status codes that should trigger a retry.
	RetryableStatusCodes []int

	// RetryableErrors are error strings that indicate a retryable condition.
	RetryableErrors []string
}

// DefaultAdapterRetryConfig returns sensible default retry configuration.
func DefaultAdapterRetryConfig() AdapterRetryConfig {
	return AdapterRetryConfig{
		MaxRetries:        3,
		InitialBackoff:    500 * time.Millisecond,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableStatusCodes: []int{
			http.StatusTooManyRequests,     // 429
			http.StatusInternalServerError, // 500
			http.StatusBadGateway,          // 502
			http.StatusServiceUnavailable,  // 503
			http.StatusGatewayTimeout,      // 504
		},
		RetryableErrors: []string{
			"connection refused",
			"connection reset",
			"timeout",
			"temporary failure",
		},
	}
}

// ProviderConfig contains all configuration for a provider adapter.
type ProviderConfig struct {
	// Name is the provider identifier.
	Name string

	// BaseURL is the provider's API endpoint base URL.
	BaseURL string

	// APIKey is the authentication key for the provider.
	APIKey string

	// DefaultModel is the fallback model if none specified.
	DefaultModel string

	// Timeout settings for this provider.
	Timeout TimeoutConfig

	// Retry settings for this provider.
	Retry AdapterRetryConfig

	// Headers are additional headers to include in every request.
	Headers map[string]string

	// StreamingEnabled indicates if streaming responses are allowed.
	StreamingEnabled bool
}

// BaseAdapter provides common functionality for provider adapters.
// Embed this in specific provider implementations.
type BaseAdapter struct {
	config           ProviderConfig
	requestTransform RequestTransformer
	responseTransform ResponseTransformer
	streamTransform  StreamTransformer
}

// NewBaseAdapter creates a base adapter with the given configuration.
func NewBaseAdapter(config ProviderConfig) *BaseAdapter {
	return &BaseAdapter{
		config: config,
	}
}

// GetProviderName returns the configured provider name.
func (b *BaseAdapter) GetProviderName() string {
	return b.config.Name
}

// SupportsStreaming returns whether streaming is enabled for this provider.
func (b *BaseAdapter) SupportsStreaming() bool {
	return b.config.StreamingEnabled
}

// SetRequestTransformer sets the request transformer.
func (b *BaseAdapter) SetRequestTransformer(t RequestTransformer) {
	b.requestTransform = t
}

// SetResponseTransformer sets the response transformer.
func (b *BaseAdapter) SetResponseTransformer(t ResponseTransformer) {
	b.responseTransform = t
}

// SetStreamTransformer sets the stream transformer.
func (b *BaseAdapter) SetStreamTransformer(t StreamTransformer) {
	b.streamTransform = t
}

// TransformRequest applies the configured request transformer.
func (b *BaseAdapter) TransformRequest(req *http.Request) (*http.Request, error) {
	if b.requestTransform == nil {
		return req, nil
	}

	// Clone the request to avoid mutating the original
	newReq := req.Clone(req.Context())

	if err := b.requestTransform.TransformURL(newReq); err != nil {
		return nil, err
	}

	if err := b.requestTransform.TransformHeaders(newReq); err != nil {
		return nil, err
	}

	if req.Body != nil {
		newBody, newContentType, err := b.requestTransform.TransformBody(req.Body, req.Header.Get("Content-Type"))
		if err != nil {
			return nil, err
		}
		newReq.Body = io.NopCloser(newBody)
		if newContentType != "" {
			newReq.Header.Set("Content-Type", newContentType)
		}
	}

	return newReq, nil
}

// TransformResponse applies the configured response transformer.
func (b *BaseAdapter) TransformResponse(resp *http.Response) (*http.Response, error) {
	if b.responseTransform == nil {
		return resp, nil
	}

	// Clone the response to avoid mutating the original
	newResp := &http.Response{
		Status:        resp.Status,
		StatusCode:    b.responseTransform.TransformStatusCode(resp.StatusCode),
		Header:        resp.Header.Clone(),
		Body:          resp.Body,
		ContentLength: resp.ContentLength,
		Request:       resp.Request,
	}

	if err := b.responseTransform.TransformHeaders(newResp); err != nil {
		return nil, err
	}

	if resp.Body != nil {
		newBody, newContentType, err := b.responseTransform.TransformBody(resp.Body, resp.Header.Get("Content-Type"))
		if err != nil {
			return nil, err
		}
		newResp.Body = io.NopCloser(newBody)
		if newContentType != "" {
			newResp.Header.Set("Content-Type", newContentType)
		}
	}

	return newResp, nil
}

// GetStreamTransformer returns the stream transformer if configured.
func (b *BaseAdapter) GetStreamTransformer() StreamTransformer {
	return b.streamTransform
}

// HTTPClient creates an http.Client configured with the adapter's timeout settings.
func (b *BaseAdapter) HTTPClient() *http.Client {
	return &http.Client{
		Timeout: b.config.Timeout.RequestTimeout,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   b.config.Timeout.TLSHandshakeTimeout,
			ResponseHeaderTimeout: b.config.Timeout.ResponseHeaderTimeout,
			IdleConnTimeout:       b.config.Timeout.IdleConnTimeout,
		},
	}
}

// AdapterWithContext extends ProviderAdapter with context-aware execution.
// This is the interface that should be used by the router for executing requests.
type AdapterWithContext interface {
	ProviderAdapter

	// Execute performs the full request lifecycle: transform request, execute, transform response.
	Execute(ctx context.Context, req *http.Request) (*http.Response, error)
}

// ExecutableAdapter wraps a ProviderAdapter with HTTP execution capabilities.
type ExecutableAdapter struct {
	*BaseAdapter
	httpClient *http.Client
}

// NewExecutableAdapter creates an adapter that can execute HTTP requests.
func NewExecutableAdapter(config ProviderConfig, reqTransform RequestTransformer, respTransform ResponseTransformer) *ExecutableAdapter {
	base := NewBaseAdapter(config)
	base.SetRequestTransformer(reqTransform)
	base.SetResponseTransformer(respTransform)

	return &ExecutableAdapter{
		BaseAdapter: base,
		httpClient:  base.HTTPClient(),
	}
}

// Execute performs the complete request lifecycle.
func (e *ExecutableAdapter) Execute(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Transform the request
	transformedReq, err := e.TransformRequest(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	// Execute the HTTP request
	resp, err := e.httpClient.Do(transformedReq)
	if err != nil {
		return nil, err
	}

	// Transform the response
	return e.TransformResponse(resp)
}

// SetHTTPClient allows overriding the default HTTP client.
func (e *ExecutableAdapter) SetHTTPClient(client *http.Client) {
	e.httpClient = client
}
