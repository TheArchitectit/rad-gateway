package provider

import (
	"context"
	"errors"
	"log/slog"

	"radgateway/internal/logger"
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
	logger   *slog.Logger
}

func NewRegistry(adapters ...Adapter) *Registry {
	r := &Registry{
		adapters: map[string]Adapter{},
		logger:   logger.WithComponent("registry"),
	}
	for _, a := range adapters {
		r.adapters[a.Name()] = a
		r.logger.Info("provider registered",
			slog.String("provider", a.Name()),
			slog.Int("total_adapters", len(r.adapters)),
		)
	}
	r.logger.Info("registry initialized",
		slog.Int("adapter_count", len(r.adapters)),
	)
	return r
}

func (r *Registry) Get(name string) (Adapter, error) {
	a, ok := r.adapters[name]
	if !ok {
		r.logger.Error("adapter lookup failed",
			slog.String("provider", name),
			slog.String("error", "adapter not found"),
		)
		return nil, errors.New("adapter not found: " + name)
	}
	r.logger.Debug("adapter retrieved",
		slog.String("provider", name),
	)
	return a, nil
}
