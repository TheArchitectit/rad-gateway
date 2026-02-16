package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// ClientConfig configures the HTTP client for provider connections.
type ClientConfig struct {
	// Timeout is the total timeout for the request.
	Timeout time.Duration
	// DialTimeout is the timeout for establishing a connection.
	DialTimeout time.Duration
	// TLSHandshakeTimeout is the timeout for TLS handshake.
	TLSHandshakeTimeout time.Duration
	// ResponseHeaderTimeout is the timeout for reading response headers.
	ResponseHeaderTimeout time.Duration
	// IdleConnTimeout is the timeout for idle connections.
	IdleConnTimeout time.Duration
	// MaxIdleConns is the maximum number of idle connections.
	MaxIdleConns int
	// MaxIdleConnsPerHost is the maximum number of idle connections per host.
	MaxIdleConnsPerHost int
	// MaxConnsPerHost is the maximum number of connections per host.
	MaxConnsPerHost int
	// DisableKeepAlives disables HTTP keep-alives.
	DisableKeepAlives bool
	// RetryConfig configures retry behavior.
	RetryConfig RetryConfig
}

// RetryConfig configures retry behavior for failed requests.
type RetryConfig struct {
	// MaxRetries is the maximum number of retries.
	MaxRetries int
	// RetryDelay is the initial delay between retries.
	RetryDelay time.Duration
	// MaxRetryDelay is the maximum delay between retries.
	MaxRetryDelay time.Duration
	// BackoffMultiplier is the multiplier for exponential backoff.
	BackoffMultiplier float64
	// RetryableStatusCodes are HTTP status codes that trigger a retry.
	RetryableStatusCodes []int
	// RetryableErrors are error types that trigger a retry.
	RetryableErrors []error
}

// DefaultClientConfig returns a sensible default configuration.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Timeout:               120 * time.Second,
		DialTimeout:           10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       100,
		DisableKeepAlives:     false,
		RetryConfig:           DefaultRetryConfig(),
	}
}

// DefaultRetryConfig returns a default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:        3,
		RetryDelay:        100 * time.Millisecond,
		MaxRetryDelay:     30 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableStatusCodes: []int{
			http.StatusRequestTimeout,
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		},
	}
}

// Client is an HTTP client optimized for provider connections.
type Client struct {
	config     ClientConfig
	httpClient *http.Client
	pool       sync.Pool
}

// NewClient creates a new provider HTTP client.
func NewClient(config ClientConfig) *Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   config.DialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		IdleConnTimeout:       config.IdleConnTimeout,
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		DisableKeepAlives:     config.DisableKeepAlives,
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   config.Timeout,
		},
		pool: sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		},
	}
}

// Do executes an HTTP request with retry logic.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	var lastErr error

	retryDelay := c.config.RetryConfig.RetryDelay

	for attempt := 0; attempt <= c.config.RetryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
				// Retry after delay
			}

			// Exponential backoff
			retryDelay = time.Duration(float64(retryDelay) * c.config.RetryConfig.BackoffMultiplier)
			if retryDelay > c.config.RetryConfig.MaxRetryDelay {
				retryDelay = c.config.RetryConfig.MaxRetryDelay
			}

			// Clone the request body for retry
			if req.Body != nil && req.GetBody != nil {
				newBody, bodyErr := req.GetBody()
				if bodyErr != nil {
					return nil, fmt.Errorf("failed to get request body for retry: %w", bodyErr)
				}
				req.Body = newBody
			}
		}

		resp, err = c.httpClient.Do(req.WithContext(ctx))
		if err != nil {
			lastErr = err
			if !c.isRetryableError(err) {
				return nil, err
			}
			continue
		}

		// Check if response status code is retryable
		if c.isRetryableStatusCode(resp.StatusCode) {
			lastErr = fmt.Errorf("received retryable status code: %d", resp.StatusCode)
			resp.Body.Close()
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.Do(ctx, req)
}

// Post performs a POST request with JSON body.
func (c *Client) Post(ctx context.Context, url string, body any, headers map[string]string) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Enable request body cloning for retries
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonBody)), nil
	}

	return c.Do(ctx, req)
}

// isRetryableError checks if an error should trigger a retry.
func (c *Client) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for timeout errors
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}

	// Check for URL errors
	if urlErr, ok := err.(*url.Error); ok {
		return urlErr.Timeout() || urlErr.Temporary()
	}

	return false
}

// isRetryableStatusCode checks if a status code should trigger a retry.
func (c *Client) isRetryableStatusCode(code int) bool {
	for _, retryable := range c.config.RetryConfig.RetryableStatusCodes {
		if code == retryable {
			return true
		}
	}
	return false
}

// Close closes the client and its idle connections.
func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}

// Stats returns connection pool statistics.
func (c *Client) Stats() http.Transport {
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		return *transport
	}
	return http.Transport{}
}

// ConnectionPool manages a pool of HTTP clients for different providers.
type ConnectionPool struct {
	mu      sync.RWMutex
	clients map[string]*Client
	config  ClientConfig
}

// NewConnectionPool creates a new connection pool.
func NewConnectionPool(config ClientConfig) *ConnectionPool {
	return &ConnectionPool{
		clients: make(map[string]*Client),
		config:  config,
	}
}

// Get returns the client for a provider, creating it if necessary.
func (cp *ConnectionPool) Get(provider string) *Client {
	cp.mu.RLock()
	client, exists := cp.clients[provider]
	cp.mu.RUnlock()

	if exists {
		return client
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := cp.clients[provider]; exists {
		return client
	}

	client = NewClient(cp.config)
	cp.clients[provider] = client
	return client
}

// Register registers a client for a provider with custom configuration.
func (cp *ConnectionPool) Register(provider string, config ClientConfig) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if existing, exists := cp.clients[provider]; exists {
		existing.Close()
	}

	cp.clients[provider] = NewClient(config)
}

// Remove removes a client from the pool.
func (cp *ConnectionPool) Remove(provider string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if client, exists := cp.clients[provider]; exists {
		client.Close()
		delete(cp.clients, provider)
	}
}

// Close closes all clients in the pool.
func (cp *ConnectionPool) Close() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	for _, client := range cp.clients {
		client.Close()
	}
	cp.clients = make(map[string]*Client)
}

// ProviderClient wraps the HTTP client with provider-specific functionality.
type ProviderClient struct {
	client     *Client
	baseURL    string
	apiKey     string
	headers    map[string]string
	circuit    *CircuitBreaker
}

// ProviderClientConfig configures a provider client.
type ProviderClientConfig struct {
	BaseURL    string
	APIKey     string
	Headers    map[string]string
	HTTPConfig ClientConfig
}

// NewProviderClient creates a new provider-specific client.
func NewProviderClient(config ProviderClientConfig) *ProviderClient {
	return &ProviderClient{
		client:  NewClient(config.HTTPConfig),
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		headers: config.Headers,
		circuit: NewCircuitBreaker(DefaultCircuitBreakerConfig()),
	}
}

// Do executes a request with circuit breaker protection.
func (pc *ProviderClient) Do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	if !pc.circuit.Allow() {
		return nil, ErrCircuitOpen
	}

	url := pc.baseURL + path

	var req *http.Request
	var err error

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(jsonBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.GetBody = func() (io.ReadCloser, error) {
			jsonBody, _ := json.Marshal(body)
			return io.NopCloser(bytes.NewReader(jsonBody)), nil
		}
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, err
		}
	}

	// Add default headers
	if pc.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+pc.apiKey)
	}
	for k, v := range pc.headers {
		req.Header.Set(k, v)
	}

	resp, err := pc.client.Do(ctx, req)
	if err != nil {
		pc.circuit.RecordFailure()
		return nil, err
	}

	if resp.StatusCode >= 500 {
		pc.circuit.RecordFailure()
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		pc.circuit.RecordSuccess()
	}

	return resp, nil
}

// Close closes the provider client.
func (pc *ProviderClient) Close() {
	pc.client.Close()
}

// CircuitStats returns circuit breaker statistics.
func (pc *ProviderClient) CircuitStats() CircuitBreakerStats {
	return pc.circuit.Stats()
}
