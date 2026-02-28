// Package api provides HTTP API handlers and route registration for RAD Gateway.
package api

import (
	"net/http"

	"radgateway/internal/a2a"
	"radgateway/internal/agui"
	"radgateway/internal/core"
	"radgateway/internal/mcp"
)

// Gateway interface provides access to A2A dependencies.
// This interface is typically implemented by the core.Gateway or a wrapper
// that provides access to A2A repositories and stores.
type Gateway interface {
	// GetA2ARepo returns the A2A model card repository
	GetA2ARepo() a2a.Repository
	// GetA2ATaskStore returns the A2A task store
	GetA2ATaskStore() a2a.TaskStore
}

// gatewayWrapper wraps core.Gateway to implement the Gateway interface
type gatewayWrapper struct {
	repo      a2a.Repository
	taskStore a2a.TaskStore
	gateway   *core.Gateway
}

// GetA2ARepo returns the A2A model card repository
func (g *gatewayWrapper) GetA2ARepo() a2a.Repository {
	return g.repo
}

// GetA2ATaskStore returns the A2A task store
func (g *gatewayWrapper) GetA2ATaskStore() a2a.TaskStore {
	return g.taskStore
}

// NewGatewayWrapper creates a new Gateway wrapper from dependencies.
// This is used when the caller has direct access to the repository and task store.
func NewGatewayWrapper(repo a2a.Repository, taskStore a2a.TaskStore, gateway *core.Gateway) Gateway {
	return &gatewayWrapper{
		repo:      repo,
		taskStore: taskStore,
		gateway:   gateway,
	}
}

// RegisterAllRoutes registers all API routes for Agent Interop protocols (A2A, AG-UI, MCP).
// It wires up all protocol handlers to the provided HTTP mux.
//
// Routes registered:
//   - A2A: /v1/a2a/model-cards, /v1/a2a/tasks/*, /a2a/* (legacy)
//   - AG-UI: /v1/agents/{agentId}/stream
//   - MCP: /mcp/v1/*
func RegisterAllRoutes(mux *http.ServeMux, gateway Gateway) {
	// A2A routes
	var a2aHandlers *a2a.Handlers
	if gateway != nil {
		a2aHandlers = a2a.NewHandlersWithTaskStore(gateway.GetA2ARepo(), gateway.GetA2ATaskStore(), nil)
	} else {
		a2aHandlers = a2a.NewHandlers(nil)
	}
	a2aHandlers.Register(mux)

	// AG-UI routes
	aguiHandler := agui.NewHandler()
	aguiHandler.RegisterRoutes(mux)

	// MCP routes
	mcpHandler := mcp.NewHandler()
	mcpHandler.Register(mux)
}
