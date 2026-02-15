package routing

import (
	"context"
	"fmt"
	"sort"

	"radgateway/internal/models"
	"radgateway/internal/provider"
)

type Attempt struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
}

type Result struct {
	Output   models.ProviderResult
	Attempts []Attempt
}

type Router struct {
	registry    *provider.Registry
	routeTable  map[string][]provider.Candidate
	retryBudget int
}

func New(registry *provider.Registry, routeTable map[string][]provider.Candidate, retryBudget int) *Router {
	return &Router{registry: registry, routeTable: routeTable, retryBudget: retryBudget}
}

func (r *Router) Dispatch(ctx context.Context, req models.ProviderRequest) (Result, error) {
	candidates := r.routeTable[req.Model]
	if len(candidates) == 0 {
		candidates = []provider.Candidate{{Name: "mock", Model: req.Model, Weight: 1}}
	}

	sorted := append([]provider.Candidate(nil), candidates...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Weight > sorted[j].Weight
	})

	attemptLimit := len(sorted)
	if r.retryBudget > 0 && r.retryBudget < attemptLimit {
		attemptLimit = r.retryBudget
	}

	attempts := make([]Attempt, 0, attemptLimit)
	for i := 0; i < attemptLimit; i++ {
		cand := sorted[i]
		adapter, err := r.registry.Get(cand.Name)
		if err != nil {
			attempts = append(attempts, Attempt{Provider: cand.Name, Model: cand.Model, Status: "error", Error: err.Error()})
			continue
		}

		res, err := adapter.Execute(ctx, req, cand.Model)
		if err != nil {
			attempts = append(attempts, Attempt{Provider: cand.Name, Model: cand.Model, Status: "error", Error: err.Error()})
			continue
		}
		attempts = append(attempts, Attempt{Provider: cand.Name, Model: cand.Model, Status: "success"})
		return Result{Output: res, Attempts: attempts}, nil
	}

	return Result{Attempts: attempts}, fmt.Errorf("all route attempts failed")
}
