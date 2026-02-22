package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"radgateway/internal/core"
	"radgateway/internal/logger"
	"radgateway/internal/models"
)

// ToolExecutor defines the interface for executing tool operations
type ToolExecutor interface {
	Execute(ctx context.Context, tool string, input map[string]interface{}) (*ToolResult, error)
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Success    bool                   `json:"success"`
	Output     map[string]interface{} `json:"output"`
	Error      string                 `json:"error,omitempty"`
	DurationMs int64                  `json:"durationMs"`
}

// gatewayToolExecutor integrates with the core gateway for LLM-based tools
type gatewayToolExecutor struct {
	gateway *core.Gateway
	log     *slog.Logger
}

func (g *gatewayToolExecutor) Execute(ctx context.Context, tool string, input map[string]interface{}) (*ToolResult, error) {
	start := time.Now()

	var content string
	if c, ok := input["content"].(string); ok {
		content = c
	} else if c, ok := input["prompt"].(string); ok {
		content = c
	} else {
		content = fmt.Sprintf("Execute tool '%s' with input: %v", tool, input)
	}

	model := "gpt-4o-mini"
	if m, ok := input["model"].(string); ok && m != "" {
		model = m
	}

	req := models.ChatCompletionRequest{
		Model: model,
		Messages: []models.Message{
			{Role: "system", Content: fmt.Sprintf("You are executing the '%s' tool. Respond with structured JSON output.", tool)},
			{Role: "user", Content: content},
		},
	}

	result, _, err := g.gateway.Handle(ctx, "chat", model, req)
	if err != nil {
		return &ToolResult{
			Success:    false,
			Error:      err.Error(),
			DurationMs: time.Since(start).Milliseconds(),
		}, nil
	}

	output := map[string]interface{}{
		"model":    result.Model,
		"provider": result.Provider,
		"result":   result.Payload,
		"usage":    result.Usage,
	}

	return &ToolResult{
		Success:    true,
		Output:     output,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

// Handler provides MCP (Model Context Protocol) HTTP endpoints
type Handler struct {
	executor ToolExecutor
	log      *slog.Logger
}

// NewHandler creates a new MCP handler with default configuration
func NewHandler() *Handler {
	return &Handler{
		log: logger.WithComponent("mcp_handler"),
	}
}

// NewHandlerWithGateway creates an MCP handler integrated with the gateway
func NewHandlerWithGateway(gateway *core.Gateway) *Handler {
	return &Handler{
		executor: &gatewayToolExecutor{
			gateway: gateway,
			log:     logger.WithComponent("mcp_gateway_executor"),
		},
		log: logger.WithComponent("mcp_handler"),
	}
}

// Request represents an MCP tool invocation request
type Request struct {
	Tool     string                 `json:"tool"`
	Input    map[string]interface{} `json:"input"`
	Session  string                 `json:"session,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Response represents an MCP tool invocation response
type Response struct {
	Success   bool                   `json:"success"`
	Tool      string                 `json:"tool"`
	Output    map[string]interface{} `json:"output,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  int64                  `json:"durationMs,omitempty"`
	Session   string                 `json:"session,omitempty"`
}

// Register registers MCP routes on the provided mux
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/mcp/v1/stdio", h.handleStdio)
	mux.HandleFunc("/mcp/v1/tools/invoke", h.handleToolInvoke)
	mux.HandleFunc("/mcp/v1/tools/list", h.handleToolList)
	mux.HandleFunc("/mcp/v1/health", h.handleHealth)
}

func (h *Handler) handleStdio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Tool == "" {
		h.writeError(w, http.StatusBadRequest, "tool is required")
		return
	}

	h.executeTool(w, r, req)
}

func (h *Handler) handleToolInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Tool == "" {
		h.writeError(w, http.StatusBadRequest, "tool is required")
		return
	}

	h.executeTool(w, r, req)
}

func (h *Handler) executeTool(w http.ResponseWriter, r *http.Request, req Request) {
	start := time.Now()

	var result *ToolResult
	var err error

	if h.executor != nil {
		result, err = h.executor.Execute(r.Context(), req.Tool, req.Input)
		if err != nil {
			h.log.Error("tool execution failed",
				"tool", req.Tool,
				"error", err,
			)
			h.writeError(w, http.StatusInternalServerError, "tool execution failed: "+err.Error())
			return
		}
	} else {
		result = h.executeBuiltinTool(req.Tool, req.Input)
	}

	resp := Response{
		Success:   result.Success,
		Tool:      req.Tool,
		Timestamp: time.Now().UTC(),
		Duration:  time.Since(start).Milliseconds(),
		Session:   req.Session,
	}

	if result.Success {
		resp.Output = result.Output
	} else {
		resp.Error = result.Error
		if resp.Error == "" {
			resp.Error = "tool execution failed"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

	h.log.Info("tool executed",
		"tool", req.Tool,
		"success", result.Success,
		"duration_ms", resp.Duration,
	)
}

func (h *Handler) executeBuiltinTool(tool string, input map[string]interface{}) *ToolResult {
	switch tool {
	case "echo":
		return &ToolResult{
			Success: true,
			Output: map[string]interface{}{
				"echo":    input,
				"message": "Echo response",
			},
		}
	case "time":
		return &ToolResult{
			Success: true,
			Output: map[string]interface{}{
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"unix":      time.Now().Unix(),
			},
		}
	case "json_parse":
		var data interface{}
		if str, ok := input["data"].(string); ok {
			if err := json.Unmarshal([]byte(str), &data); err != nil {
				return &ToolResult{Success: false, Error: "parse error: " + err.Error()}
			}
		} else {
			data = input["data"]
		}
		return &ToolResult{
			Success: true,
			Output: map[string]interface{}{
				"parsed": data,
				"type":   fmt.Sprintf("%T", data),
			},
		}
	default:
		if h.executor != nil {
			return &ToolResult{Success: false, Error: "tool not found: " + tool}
		}
		return &ToolResult{
			Success: true,
			Output: map[string]interface{}{
				"tool":    tool,
				"input":   input,
				"message": "Tool execution simulated (no executor configured)",
			},
		}
	}
}

func (h *Handler) handleToolList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tools := []map[string]interface{}{
		{
			"name":        "echo",
			"description": "Echoes back the input",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{"type": "string"},
				},
			},
		},
		{
			"name":        "time",
			"description": "Returns current time information",
			"parameters": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "json_parse",
			"description": "Parses JSON string to structured data",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"data": map[string]interface{}{"type": "string"},
				},
			},
		},
		{
			"name":        "chat",
			"description": "Execute LLM chat completion via gateway",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{"type": "string"},
					"model":   map[string]interface{}{"type": "string"},
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": tools,
		"count": len(tools),
	})
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	status := "healthy"
	if h.executor == nil {
		status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    status,
		"service":   "mcp",
		"executor":  h.executor != nil,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(Response{
		Success:   false,
		Error:     message,
		Timestamp: time.Now().UTC(),
	})
}
