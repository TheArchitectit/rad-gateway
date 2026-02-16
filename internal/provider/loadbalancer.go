package provider

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

// ErrNoAvailableProviders is returned when no providers are available.
var ErrNoAvailableProviders = errors.New("no available providers")

// Provider represents a provider with its health status.
type Provider struct {
	Name     string
	Model    string
	Weight   int
	Healthy  bool
	Priority int
}

// LoadBalancer defines the interface for load balancing strategies.
type LoadBalancer interface {
	// Select returns the next available provider.
	Select(ctx context.Context, providers []Provider) (*Provider, error)
	// Name returns the strategy name.
	Name() string
}

// RoundRobinLoadBalancer implements simple round-robin load balancing.
type RoundRobinLoadBalancer struct {
	counter atomic.Uint64
}

// NewRoundRobinLoadBalancer creates a new round-robin load balancer.
func NewRoundRobinLoadBalancer() *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{}
}

// Name returns the strategy name.
func (rr *RoundRobinLoadBalancer) Name() string {
	return "round-robin"
}

// Select returns the next available provider using round-robin.
func (rr *RoundRobinLoadBalancer) Select(ctx context.Context, providers []Provider) (*Provider, error) {
	if len(providers) == 0 {
		return nil, ErrNoAvailableProviders
	}

	// Filter healthy providers
	healthy := make([]Provider, 0, len(providers))
	for _, p := range providers {
		if p.Healthy {
			healthy = append(healthy, p)
		}
	}

	if len(healthy) == 0 {
		return nil, ErrNoAvailableProviders
	}

	// Round-robin selection
	idx := rr.counter.Add(1) % uint64(len(healthy))
	return &healthy[idx], nil
}

// WeightedRoundRobinLoadBalancer implements weighted round-robin load balancing.
type WeightedRoundRobinLoadBalancer struct {
	counter atomic.Uint64
}

// NewWeightedRoundRobinLoadBalancer creates a new weighted round-robin load balancer.
func NewWeightedRoundRobinLoadBalancer() *WeightedRoundRobinLoadBalancer {
	return &WeightedRoundRobinLoadBalancer{}
}

// Name returns the strategy name.
func (wrr *WeightedRoundRobinLoadBalancer) Name() string {
	return "weighted-round-robin"
}

// Select returns the next available provider using weighted round-robin.
func (wrr *WeightedRoundRobinLoadBalancer) Select(ctx context.Context, providers []Provider) (*Provider, error) {
	if len(providers) == 0 {
		return nil, ErrNoAvailableProviders
	}

	// Build weighted list
	type weightedProvider struct {
		provider Provider
		weight   int
	}

	weighted := make([]weightedProvider, 0, len(providers))
	totalWeight := 0
	for _, p := range providers {
		if p.Healthy && p.Weight > 0 {
			weighted = append(weighted, weightedProvider{p, p.Weight})
			totalWeight += p.Weight
		}
	}

	if len(weighted) == 0 {
		return nil, ErrNoAvailableProviders
	}

	// Weighted selection
	counter := wrr.counter.Add(1)
	point := int(counter % uint64(totalWeight))

	current := 0
	for _, wp := range weighted {
		current += wp.weight
		if point < current {
			return &wp.provider, nil
		}
	}

	// Fallback to last provider
	return &weighted[len(weighted)-1].provider, nil
}

// PriorityLoadBalancer selects providers based on priority, falling back to lower priorities.
type PriorityLoadBalancer struct {
	mu      sync.RWMutex
	current int
}

// NewPriorityLoadBalancer creates a new priority-based load balancer.
func NewPriorityLoadBalancer() *PriorityLoadBalancer {
	return &PriorityLoadBalancer{}
}

// Name returns the strategy name.
func (pl *PriorityLoadBalancer) Name() string {
	return "priority"
}

// Select returns the highest priority available provider.
func (pl *PriorityLoadBalancer) Select(ctx context.Context, providers []Provider) (*Provider, error) {
	if len(providers) == 0 {
		return nil, ErrNoAvailableProviders
	}

	// Group by priority
	byPriority := make(map[int][]Provider)
	maxPriority := -1
	for _, p := range providers {
		if p.Healthy {
			byPriority[p.Priority] = append(byPriority[p.Priority], p)
			if p.Priority > maxPriority {
				maxPriority = p.Priority
			}
		}
	}

	if maxPriority < 0 {
		return nil, ErrNoAvailableProviders
	}

	// Select from highest priority group
	highest := byPriority[maxPriority]
	pl.mu.Lock()
	idx := pl.current % len(highest)
	pl.current = (pl.current + 1) % len(highest)
	pl.mu.Unlock()

	return &highest[idx], nil
}

// HealthAwareLoadBalancer wraps another load balancer with health checking.
type HealthAwareLoadBalancer struct {
	inner     LoadBalancer
	health    *HealthChecker
	mu        sync.RWMutex
	providers []Provider
}

// NewHealthAwareLoadBalancer creates a new health-aware load balancer.
func NewHealthAwareLoadBalancer(inner LoadBalancer, health *HealthChecker) *HealthAwareLoadBalancer {
	return &HealthAwareLoadBalancer{
		inner:  inner,
		health: health,
	}
}

// Name returns the strategy name.
func (hal *HealthAwareLoadBalancer) Name() string {
	return "health-aware-" + hal.inner.Name()
}

// UpdateProviders updates the list of providers.
func (hal *HealthAwareLoadBalancer) UpdateProviders(providers []Provider) {
	hal.mu.Lock()
	defer hal.mu.Unlock()
	hal.providers = providers
}

// Select returns the next available provider, filtering by health status.
func (hal *HealthAwareLoadBalancer) Select(ctx context.Context, providers []Provider) (*Provider, error) {
	hal.mu.RLock()
	allProviders := hal.providers
	if len(providers) > 0 {
		allProviders = providers
	}
	hal.mu.RUnlock()

	// Update health status for each provider
	healthyProviders := make([]Provider, 0, len(allProviders))
	for _, p := range allProviders {
		status := hal.health.Status(p.Name)
		p.Healthy = status.Healthy
		healthyProviders = append(healthyProviders, p)
	}

	return hal.inner.Select(ctx, healthyProviders)
}

// LoadBalancerRegistry manages multiple load balancers for different models.
type LoadBalancerRegistry struct {
	mu         sync.RWMutex
	balancers  map[string]LoadBalancer
	providers  map[string][]Provider
	factory    func() LoadBalancer
}

// NewLoadBalancerRegistry creates a new load balancer registry.
func NewLoadBalancerRegistry(factory func() LoadBalancer) *LoadBalancerRegistry {
	return &LoadBalancerRegistry{
		balancers: make(map[string]LoadBalancer),
		providers: make(map[string][]Provider),
		factory:   factory,
	}
}

// Register registers providers for a model.
func (r *LoadBalancerRegistry) Register(model string, providers []Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[model] = providers
	if _, exists := r.balancers[model]; !exists {
		r.balancers[model] = r.factory()
	}
}

// Select selects a provider for the given model.
func (r *LoadBalancerRegistry) Select(ctx context.Context, model string) (*Provider, error) {
	r.mu.RLock()
	lb, exists := r.balancers[model]
	providers := r.providers[model]
	r.mu.RUnlock()

	if !exists {
		return nil, ErrNoAvailableProviders
	}

	return lb.Select(ctx, providers)
}

// GetBalancer returns the load balancer for a model.
func (r *LoadBalancerRegistry) GetBalancer(model string) (LoadBalancer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	lb, exists := r.balancers[model]
	return lb, exists
}

// UpdateProviders updates providers for a model.
func (r *LoadBalancerRegistry) UpdateProviders(model string, providers []Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[model] = providers
}
