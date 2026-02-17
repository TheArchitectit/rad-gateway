package core

import (
	"context"
	"time"

	"log/slog"

	"radgateway/internal/logger"
	"radgateway/internal/middleware"
	"radgateway/internal/models"
	"radgateway/internal/routing"
	"radgateway/internal/trace"
	"radgateway/internal/usage"
)

type Gateway struct {
	router *routing.Router
	usage  usage.Sink
	trace  *trace.Store
	log    *slog.Logger
}

func New(router *routing.Router, usageSink usage.Sink, traceStore *trace.Store) *Gateway {
	return &Gateway{
		router: router,
		usage:  usageSink,
		trace:  traceStore,
		log:    logger.WithComponent("gateway"),
	}
}

func (g *Gateway) Handle(ctx context.Context, apiType string, model string, payload any) (models.ProviderResult, []routing.Attempt, error) {
	started := time.Now()
	requestID := middleware.GetRequestID(ctx)
	traceID := middleware.GetTraceID(ctx)
	apiKeyName := middleware.GetAPIKeyName(ctx)

	g.trace.Add(trace.Event{Timestamp: started, TraceID: traceID, RequestID: requestID, Message: "gateway request accepted"})

	result, err := g.router.Dispatch(ctx, models.ProviderRequest{APIType: apiType, Model: model, Payload: payload})
	duration := time.Since(started)
	if err != nil {
		g.trace.Add(trace.Event{Timestamp: time.Now(), TraceID: traceID, RequestID: requestID, Message: "gateway request failed"})
		g.usage.Add(usage.Record{
			Timestamp:      time.Now(),
			RequestID:      requestID,
			TraceID:        traceID,
			APIKeyName:     apiKeyName,
			IncomingAPI:    apiType,
			IncomingModel:  model,
			SelectedModel:  "",
			Provider:       "",
			ResponseStatus: "error",
			DurationMs:     duration.Milliseconds(),
		})
		return models.ProviderResult{}, result.Attempts, err
	}

	g.trace.Add(trace.Event{Timestamp: time.Now(), TraceID: traceID, RequestID: requestID, Message: "gateway request completed"})
	g.usage.Add(usage.Record{
		Timestamp:      time.Now(),
		RequestID:      requestID,
		TraceID:        traceID,
		APIKeyName:     apiKeyName,
		IncomingAPI:    apiType,
		IncomingModel:  model,
		SelectedModel:  result.Output.Model,
		Provider:       result.Output.Provider,
		ResponseStatus: "success",
		DurationMs:     duration.Milliseconds(),
		Usage:          result.Output.Usage,
	})

	return result.Output, result.Attempts, nil
}
