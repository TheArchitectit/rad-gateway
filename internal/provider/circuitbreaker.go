package provider

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	// StateClosed means the circuit is closed and requests flow normally.
	StateClosed CircuitState = iota
	// StateOpen means the circuit is open and requests are blocked.
	StateOpen
	// StateHalfOpen means the circuit is testing if the service has recovered.
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig configures the circuit breaker behavior.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening.
	FailureThreshold int
	// SuccessThreshold is the number of consecutive successes in half-open to close.
	SuccessThreshold int
	// Timeout is the duration the circuit stays open before testing.
	Timeout time.Duration
	// HalfOpenMaxRequests is the maximum number of requests allowed in half-open state.
	HalfOpenMaxRequests int
}

// DefaultCircuitBreakerConfig returns a sensible default configuration.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:    5,
		SuccessThreshold:    3,
		Timeout:             30 * time.Second,
		HalfOpenMaxRequests: 1,
	}
}

// CircuitBreaker implements the circuit breaker pattern for provider failover.
type CircuitBreaker struct {
	config CircuitBreakerConfig

	mu                sync.RWMutex
	state             CircuitState
	failures          int
	successes         int
	lastFailureTime   time.Time
	halfOpenRequests  int
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Allow returns true if the request should be allowed through.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			cb.transitionTo(StateHalfOpen)
			cb.halfOpenRequests = 1
			return true
		}
		return false
	case StateHalfOpen:
		if cb.halfOpenRequests < cb.config.HalfOpenMaxRequests {
			cb.halfOpenRequests++
			return true
		}
		return false
	}
	return false
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.transitionTo(StateClosed)
		}
	case StateClosed:
		cb.failures = 0
	}
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateHalfOpen:
		cb.transitionTo(StateOpen)
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.config.FailureThreshold {
			cb.transitionTo(StateOpen)
		}
	}
}

// transitionTo changes the circuit breaker state and resets counters.
func (cb *CircuitBreaker) transitionTo(state CircuitState) {
	cb.state = state
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenRequests = 0
}

// Stats returns the current statistics of the circuit breaker.
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		State:            cb.state,
		Failures:         cb.failures,
		Successes:        cb.successes,
		LastFailureTime:  cb.lastFailureTime,
		HalfOpenRequests: cb.halfOpenRequests,
	}
}

// CircuitBreakerStats holds the current statistics of a circuit breaker.
type CircuitBreakerStats struct {
	State            CircuitState
	Failures         int
	Successes        int
	LastFailureTime  time.Time
	HalfOpenRequests int
}

// IsOpen returns true if the circuit is open.
func (s CircuitBreakerStats) IsOpen() bool {
	return s.State == StateOpen
}

// ErrCircuitOpen is returned when the circuit breaker is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreakerMiddleware wraps a function with circuit breaker protection.
func CircuitBreakerMiddleware(cb *CircuitBreaker) func(ctx context.Context, req any) (any, error) {
	return func(ctx context.Context, req any) (any, error) {
		if !cb.Allow() {
			return nil, ErrCircuitOpen
		}

		// Execute the actual request
		return nil, nil
	}
}

// ProviderCircuitBreaker manages circuit breakers for multiple providers.
type ProviderCircuitBreaker struct {
	mu              sync.RWMutex
	breakers        map[string]*CircuitBreaker
	config          CircuitBreakerConfig
	defaultProvider string
}

// NewProviderCircuitBreaker creates a new provider circuit breaker manager.
func NewProviderCircuitBreaker(config CircuitBreakerConfig, defaultProvider string) *ProviderCircuitBreaker {
	return &ProviderCircuitBreaker{
		breakers:        make(map[string]*CircuitBreaker),
		config:          config,
		defaultProvider: defaultProvider,
	}
}

// Get returns the circuit breaker for a provider, creating it if necessary.
func (pcb *ProviderCircuitBreaker) Get(provider string) *CircuitBreaker {
	pcb.mu.RLock()
	cb, exists := pcb.breakers[provider]
	pcb.mu.RUnlock()

	if exists {
		return cb
	}

	pcb.mu.Lock()
	defer pcb.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists := pcb.breakers[provider]; exists {
		return cb
	}

	cb = NewCircuitBreaker(pcb.config)
	pcb.breakers[provider] = cb
	return cb
}

// RecordSuccess records a success for the given provider.
func (pcb *ProviderCircuitBreaker) RecordSuccess(provider string) {
	pcb.Get(provider).RecordSuccess()
}

// RecordFailure records a failure for the given provider.
func (pcb *ProviderCircuitBreaker) RecordFailure(provider string) {
	pcb.Get(provider).RecordFailure()
}

// IsAvailable returns true if the provider is available (circuit not open).
func (pcb *ProviderCircuitBreaker) IsAvailable(provider string) bool {
	return pcb.Get(provider).Allow()
}

// Stats returns statistics for all providers.
func (pcb *ProviderCircuitBreaker) Stats() map[string]CircuitBreakerStats {
	pcb.mu.RLock()
	defer pcb.mu.RUnlock()

	stats := make(map[string]CircuitBreakerStats, len(pcb.breakers))
	for provider, cb := range pcb.breakers {
		stats[provider] = cb.Stats()
	}
	return stats
}
