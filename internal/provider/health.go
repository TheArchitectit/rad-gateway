package provider

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HealthStatus represents the health status of a provider.
type HealthStatus struct {
	Provider        string
	Healthy         bool
	LastCheck       time.Time
	LastSuccess     time.Time
	ConsecutiveFail int
	Latency         time.Duration
	Error           error
}

// IsHealthy returns true if the provider is healthy.
func (h HealthStatus) IsHealthy() bool {
	return h.Healthy
}

// HealthChecker performs health checks on providers.
type HealthChecker struct {
	mu        sync.RWMutex
	statuses  map[string]HealthStatus
	clients   map[string]*http.Client
	interval  time.Duration
	timeout   time.Duration
	threshold int
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// HealthCheckerConfig configures the health checker.
type HealthCheckerConfig struct {
	// Interval is the time between health checks.
	Interval time.Duration
	// Timeout is the maximum time to wait for a health check response.
	Timeout time.Duration
	// Threshold is the number of consecutive failures before marking unhealthy.
	Threshold int
}

// DefaultHealthCheckerConfig returns a default configuration.
func DefaultHealthCheckerConfig() HealthCheckerConfig {
	return HealthCheckerConfig{
		Interval:  30 * time.Second,
		Timeout:   5 * time.Second,
		Threshold: 3,
	}
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(config HealthCheckerConfig) *HealthChecker {
	return &HealthChecker{
		statuses:  make(map[string]HealthStatus),
		clients:   make(map[string]*http.Client),
		interval:  config.Interval,
		timeout:   config.Timeout,
		threshold: config.Threshold,
		stopCh:    make(chan struct{}),
	}
}

// Register registers a provider for health checking.
func (hc *HealthChecker) Register(provider string, healthURL string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.statuses[provider] = HealthStatus{
		Provider:  provider,
		Healthy:   true, // Start healthy
		LastCheck: time.Now(),
	}

	// Create a dedicated client for this provider
	hc.clients[provider] = &http.Client{
		Timeout: hc.timeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

// Unregister removes a provider from health checking.
func (hc *HealthChecker) Unregister(provider string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	delete(hc.statuses, provider)
	delete(hc.clients, provider)
}

// Status returns the current health status of a provider.
func (hc *HealthChecker) Status(provider string) HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.statuses[provider]
}

// AllStatuses returns the health status of all providers.
func (hc *HealthChecker) AllStatuses() map[string]HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result := make(map[string]HealthStatus, len(hc.statuses))
	for k, v := range hc.statuses {
		result[k] = v
	}
	return result
}

// Start begins the health checking loop.
func (hc *HealthChecker) Start(ctx context.Context, checkers map[string]HealthCheckFunc) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	// Initial check
	hc.runChecks(ctx, checkers)

	for {
		select {
		case <-ctx.Done():
			return
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.runChecks(ctx, checkers)
		}
	}
}

// Stop stops the health checker.
func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
	hc.wg.Wait()
}

// HealthCheckFunc is a function that performs a health check.
type HealthCheckFunc func(ctx context.Context) HealthStatus

// runChecks runs health checks for all registered providers.
func (hc *HealthChecker) runChecks(ctx context.Context, checkers map[string]HealthCheckFunc) {
	hc.mu.RLock()
	providers := make([]string, 0, len(hc.statuses))
	for provider := range hc.statuses {
		providers = append(providers, provider)
	}
	hc.mu.RUnlock()

	for _, provider := range providers {
		hc.wg.Add(1)
		go func(p string) {
			defer hc.wg.Done()

			var status HealthStatus
			if checker, ok := checkers[p]; ok {
				status = checker(ctx)
			} else {
				status = hc.defaultCheck(ctx, p)
			}

			hc.updateStatus(p, status)
		}(provider)
	}
}

// defaultCheck performs a default HTTP health check.
func (hc *HealthChecker) defaultCheck(ctx context.Context, provider string) HealthStatus {
	hc.mu.RLock()
	client, exists := hc.clients[provider]
	hc.mu.RUnlock()

	if !exists {
		return HealthStatus{
			Provider: provider,
			Healthy:  false,
			Error:    fmt.Errorf("no client registered for provider %s", provider),
		}
	}

	start := time.Now()
	// Default health endpoint - should be configurable per provider
	// This is a placeholder; real implementation should use provider-specific endpoints
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/health", nil)
	if err != nil {
		return HealthStatus{
			Provider: provider,
			Healthy:  false,
			Latency:  time.Since(start),
			Error:    err,
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return HealthStatus{
			Provider: provider,
			Healthy:  false,
			Latency:  time.Since(start),
			Error:    err,
		}
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode >= 200 && resp.StatusCode < 300
	return HealthStatus{
		Provider: provider,
		Healthy:  healthy,
		Latency:  time.Since(start),
		Error:    nil,
	}
}

// updateStatus updates the health status of a provider.
func (hc *HealthChecker) updateStatus(provider string, newStatus HealthStatus) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	current := hc.statuses[provider]
	current.LastCheck = time.Now()
	current.Latency = newStatus.Latency
	current.Error = newStatus.Error

	if newStatus.Healthy {
		current.Healthy = true
		current.ConsecutiveFail = 0
		current.LastSuccess = time.Now()
	} else {
		current.ConsecutiveFail++
		if current.ConsecutiveFail >= hc.threshold {
			current.Healthy = false
		}
	}

	hc.statuses[provider] = current
}

// SimpleHealthChecker provides a simplified interface for health checking.
type SimpleHealthChecker struct {
	mu       sync.RWMutex
	healthy  map[string]bool
	checkers map[string]func() error
}

// NewSimpleHealthChecker creates a new simple health checker.
func NewSimpleHealthChecker() *SimpleHealthChecker {
	return &SimpleHealthChecker{
		healthy:  make(map[string]bool),
		checkers: make(map[string]func() error),
	}
}

// Register registers a health check function for a provider.
func (shc *SimpleHealthChecker) Register(provider string, checker func() error) {
	shc.mu.Lock()
	defer shc.mu.Unlock()
	shc.checkers[provider] = checker
	shc.healthy[provider] = true
}

// Check performs a health check for a provider.
func (shc *SimpleHealthChecker) Check(provider string) error {
	shc.mu.RLock()
	checker, exists := shc.checkers[provider]
	shc.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no health checker registered for provider %s", provider)
	}

	err := checker()

	shc.mu.Lock()
	shc.healthy[provider] = err == nil
	shc.mu.Unlock()

	return err
}

// IsHealthy returns true if the provider is healthy.
func (shc *SimpleHealthChecker) IsHealthy(provider string) bool {
	shc.mu.RLock()
	defer shc.mu.RUnlock()
	return shc.healthy[provider]
}

// SetHealth manually sets the health status of a provider.
func (shc *SimpleHealthChecker) SetHealth(provider string, healthy bool) {
	shc.mu.Lock()
	defer shc.mu.Unlock()
	shc.healthy[provider] = healthy
}

// HealthEndpoint represents a health check endpoint.
type HealthEndpoint struct {
	Name    string
	URL     string
	Method  string
	Headers map[string]string
}

// HTTPHealthChecker performs HTTP-based health checks.
type HTTPHealthChecker struct {
	client     *http.Client
	endpoints  map[string]HealthEndpoint
	onStatus   func(provider string, status HealthStatus)
}

// NewHTTPHealthChecker creates a new HTTP health checker.
func NewHTTPHealthChecker(timeout time.Duration) *HTTPHealthChecker {
	return &HTTPHealthChecker{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		endpoints: make(map[string]HealthEndpoint),
	}
}

// RegisterEndpoint registers a health check endpoint for a provider.
func (hhc *HTTPHealthChecker) RegisterEndpoint(provider string, endpoint HealthEndpoint) {
	hhc.endpoints[provider] = endpoint
}

// Check performs an HTTP health check for a provider.
func (hhc *HTTPHealthChecker) Check(ctx context.Context, provider string) HealthStatus {
	endpoint, exists := hhc.endpoints[provider]
	if !exists {
		return HealthStatus{
			Provider: provider,
			Healthy:  false,
			Error:    fmt.Errorf("no health endpoint registered for provider %s", provider),
		}
	}

	start := time.Now()
	method := endpoint.Method
	if method == "" {
		method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.URL, nil)
	if err != nil {
		return HealthStatus{
			Provider: provider,
			Healthy:  false,
			Latency:  time.Since(start),
			Error:    err,
		}
	}

	for k, v := range endpoint.Headers {
		req.Header.Set(k, v)
	}

	resp, err := hhc.client.Do(req)
	if err != nil {
		status := HealthStatus{
			Provider: provider,
			Healthy:  false,
			Latency:  time.Since(start),
			Error:    err,
		}
		if hhc.onStatus != nil {
			hhc.onStatus(provider, status)
		}
		return status
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode >= 200 && resp.StatusCode < 300
	status := HealthStatus{
		Provider: provider,
		Healthy:  healthy,
		Latency:  time.Since(start),
	}

	if !healthy {
		status.Error = fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	if hhc.onStatus != nil {
		hhc.onStatus(provider, status)
	}

	return status
}

// OnStatus sets a callback for status updates.
func (hhc *HTTPHealthChecker) OnStatus(callback func(provider string, status HealthStatus)) {
	hhc.onStatus = callback
}
