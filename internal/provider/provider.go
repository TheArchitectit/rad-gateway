package provider

import (
	"context"
	"errors"

	"radgateway/internal/models"
)

type Candidate struct {
	Name   string
	Model  string
	Weight int
}

type Adapter interface {
	Name() string
	Execute(ctx context.Context, req models.ProviderRequest, model string) (models.ProviderResult, error)
}

type Registry struct {
	adapters map[string]Adapter
}

func NewRegistry(adapters ...Adapter) *Registry {
	r := &Registry{adapters: map[string]Adapter{}}
	for _, a := range adapters {
		r.adapters[a.Name()] = a
	}
	return r
}

func (r *Registry) Get(name string) (Adapter, error) {
	a, ok := r.adapters[name]
	if !ok {
		return nil, errors.New("adapter not found: " + name)
	}
	return a, nil
}
