// Package mcp provides MCP (Model Context Protocol) support for RAD Gateway.
package mcp

import (
	"context"
	"errors"
	"sync"
)

// Tool represents an MCP tool
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema describes tool input parameters
type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

// Property represents a single parameter property
type Property struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// Resource represents an MCP resource
type Resource struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	MIMEType    string `json:"mimeType,omitempty"`
}

// ToolHandler is a function that handles tool calls
type ToolHandler func(ctx context.Context, arguments map[string]interface{}) (interface{}, error)

// Bridge manages MCP tools and resources with thread-safe operations
type Bridge struct {
	mu        sync.RWMutex
	tools     map[string]Tool
	resources map[string]Resource
	handlers  map[string]ToolHandler
}

// NewBridge creates a new Bridge with initialized maps
func NewBridge() *Bridge {
	return &Bridge{
		tools:     make(map[string]Tool),
		resources: make(map[string]Resource),
		handlers:  make(map[string]ToolHandler),
	}
}

// RegisterTool registers a tool with the bridge
// Returns an error if the tool name is empty
func (b *Bridge) RegisterTool(tool Tool) error {
	if tool.Name == "" {
		return errors.New("tool name cannot be empty")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.tools[tool.Name] = tool
	return nil
}

// RegisterToolHandler registers a handler for a tool
// Returns an error if the tool does not exist
func (b *Bridge) RegisterToolHandler(toolName string, handler ToolHandler) error {
	if toolName == "" {
		return errors.New("tool name cannot be empty")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.tools[toolName]; !exists {
		return errors.New("tool not found: " + toolName)
	}

	b.handlers[toolName] = handler
	return nil
}

// ListTools returns all registered tools
func (b *Bridge) ListTools() []Tool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	tools := make([]Tool, 0, len(b.tools))
	for _, tool := range b.tools {
		tools = append(tools, tool)
	}
	return tools
}

// CallTool invokes a tool handler with the given arguments
// Returns an error if the tool or handler does not exist
func (b *Bridge) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error) {
	if toolName == "" {
		return nil, errors.New("tool name cannot be empty")
	}

	b.mu.RLock()
	handler, handlerExists := b.handlers[toolName]
	_, toolExists := b.tools[toolName]
	b.mu.RUnlock()

	if !toolExists {
		return nil, errors.New("tool not found: " + toolName)
	}

	if !handlerExists {
		return nil, errors.New("no handler registered for tool: " + toolName)
	}

	return handler(ctx, arguments)
}

// RegisterResource registers a resource with the bridge
// Returns an error if the resource name is empty
func (b *Bridge) RegisterResource(resource Resource) error {
	if resource.Name == "" {
		return errors.New("resource name cannot be empty")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.resources[resource.Name] = resource
	return nil
}

// ListResources returns all registered resources
func (b *Bridge) ListResources() []Resource {
	b.mu.RLock()
	defer b.mu.RUnlock()

	resources := make([]Resource, 0, len(b.resources))
	for _, resource := range b.resources {
		resources = append(resources, resource)
	}
	return resources
}

// GetToolSchema returns the input schema for a tool
// Returns an error if the tool does not exist
func (b *Bridge) GetToolSchema(toolName string) (InputSchema, error) {
	if toolName == "" {
		return InputSchema{}, errors.New("tool name cannot be empty")
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	tool, exists := b.tools[toolName]
	if !exists {
		return InputSchema{}, errors.New("tool not found: " + toolName)
	}

	return tool.InputSchema, nil
}
