package mcp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestNewBridge(t *testing.T) {
	bridge := NewBridge()
	if bridge == nil {
		t.Fatal("NewBridge returned nil")
	}

	if bridge.tools == nil {
		t.Error("tools map is nil")
	}
	if bridge.resources == nil {
		t.Error("resources map is nil")
	}
	if bridge.handlers == nil {
		t.Error("handlers map is nil")
	}
}

func TestBridge_RegisterTool(t *testing.T) {
	bridge := NewBridge()

	tool := Tool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"input": {Type: "string", Description: "Input parameter"},
			},
			Required: []string{"input"},
		},
	}

	err := bridge.RegisterTool(tool)
	if err != nil {
		t.Errorf("RegisterTool failed: %v", err)
	}

	// Test duplicate registration
	err = bridge.RegisterTool(tool)
	if err != nil {
		t.Errorf("RegisterTool with duplicate failed: %v", err)
	}
}

func TestBridge_RegisterTool_EmptyName(t *testing.T) {
	bridge := NewBridge()

	tool := Tool{
		Name:        "",
		Description: "A test tool with no name",
	}

	err := bridge.RegisterTool(tool)
	if err == nil {
		t.Error("RegisterTool should return error for empty name")
	}
}

func TestBridge_RegisterToolHandler(t *testing.T) {
	bridge := NewBridge()

	// First register a tool
	tool := Tool{
		Name:        "calculator",
		Description: "A calculator tool",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"a": {Type: "number", Description: "First number"},
				"b": {Type: "number", Description: "Second number"},
			},
			Required: []string{"a", "b"},
		},
	}
	bridge.RegisterTool(tool)

	// Register a handler for the tool
	handler := func(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
		a, _ := arguments["a"].(float64)
		b, _ := arguments["b"].(float64)
		return a + b, nil
	}

	err := bridge.RegisterToolHandler("calculator", handler)
	if err != nil {
		t.Errorf("RegisterToolHandler failed: %v", err)
	}
}

func TestBridge_RegisterToolHandler_NonExistentTool(t *testing.T) {
	bridge := NewBridge()

	handler := func(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
		return "result", nil
	}

	err := bridge.RegisterToolHandler("non-existent", handler)
	if err == nil {
		t.Error("RegisterToolHandler should return error for non-existent tool")
	}
}

func TestBridge_ListTools(t *testing.T) {
	bridge := NewBridge()

	// Initially empty
	tools := bridge.ListTools()
	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}

	// Register some tools
	bridge.RegisterTool(Tool{Name: "tool-1", Description: "Tool 1"})
	bridge.RegisterTool(Tool{Name: "tool-2", Description: "Tool 2"})

	tools = bridge.ListTools()
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}

	// Verify tool names
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}
	if !toolNames["tool-1"] || !toolNames["tool-2"] {
		t.Error("ListTools did not return registered tools")
	}
}

func TestBridge_CallTool(t *testing.T) {
	bridge := NewBridge()

	// Register a tool
	tool := Tool{
		Name:        "calculator",
		Description: "A calculator tool",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"a": {Type: "number", Description: "First number"},
				"b": {Type: "number", Description: "Second number"},
			},
		},
	}
	bridge.RegisterTool(tool)

	// Register handler
	handler := func(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
		a, _ := arguments["a"].(float64)
		b, _ := arguments["b"].(float64)
		return a + b, nil
	}
	bridge.RegisterToolHandler("calculator", handler)

	// Call the tool
	result, err := bridge.CallTool(context.Background(), "calculator", map[string]interface{}{
		"a": 5.0,
		"b": 3.0,
	})
	if err != nil {
		t.Errorf("CallTool failed: %v", err)
	}

	// Verify result
	if result != 8.0 {
		t.Errorf("expected result 8.0, got %v", result)
	}
}

func TestBridge_CallTool_NonExistentTool(t *testing.T) {
	bridge := NewBridge()

	_, err := bridge.CallTool(context.Background(), "non-existent", map[string]interface{}{})
	if err == nil {
		t.Error("CallTool should return error for non-existent tool")
	}
}

func TestBridge_CallTool_NoHandler(t *testing.T) {
	bridge := NewBridge()

	// Register tool without handler
	bridge.RegisterTool(Tool{Name: "no-handler-tool", Description: "Tool without handler"})

	_, err := bridge.CallTool(context.Background(), "no-handler-tool", map[string]interface{}{})
	if err == nil {
		t.Error("CallTool should return error when no handler registered")
	}
}

func TestBridge_CallTool_HandlerError(t *testing.T) {
	bridge := NewBridge()

	// Register a tool
	bridge.RegisterTool(Tool{Name: "error-tool", Description: "Tool that errors"})

	// Register handler that returns an error
	handler := func(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
		return nil, errors.New("handler error")
	}
	bridge.RegisterToolHandler("error-tool", handler)

	_, err := bridge.CallTool(context.Background(), "error-tool", map[string]interface{}{})
	if err == nil {
		t.Error("CallTool should return error when handler returns error")
	}
}

func TestBridge_RegisterResource(t *testing.T) {
	bridge := NewBridge()

	resource := Resource{
		Name:        "test-resource",
		Description: "A test resource",
		MIMEType:    "application/json",
	}

	err := bridge.RegisterResource(resource)
	if err != nil {
		t.Errorf("RegisterResource failed: %v", err)
	}

	// Test duplicate registration
	err = bridge.RegisterResource(resource)
	if err != nil {
		t.Errorf("RegisterResource with duplicate failed: %v", err)
	}
}

func TestBridge_RegisterResource_EmptyName(t *testing.T) {
	bridge := NewBridge()

	resource := Resource{
		Name:        "",
		Description: "A resource with no name",
	}

	err := bridge.RegisterResource(resource)
	if err == nil {
		t.Error("RegisterResource should return error for empty name")
	}
}

func TestBridge_ListResources(t *testing.T) {
	bridge := NewBridge()

	// Initially empty
	resources := bridge.ListResources()
	if len(resources) != 0 {
		t.Errorf("expected 0 resources, got %d", len(resources))
	}

	// Register some resources
	bridge.RegisterResource(Resource{Name: "resource-1", Description: "Resource 1"})
	bridge.RegisterResource(Resource{Name: "resource-2", Description: "Resource 2"})

	resources = bridge.ListResources()
	if len(resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(resources))
	}

	// Verify resource names
	resourceNames := make(map[string]bool)
	for _, resource := range resources {
		resourceNames[resource.Name] = true
	}
	if !resourceNames["resource-1"] || !resourceNames["resource-2"] {
		t.Error("ListResources did not return registered resources")
	}
}

func TestBridge_GetToolSchema(t *testing.T) {
	bridge := NewBridge()

	expectedSchema := InputSchema{
		Type: "object",
		Properties: map[string]Property{
			"input": {Type: "string", Description: "Input parameter"},
		},
		Required: []string{"input"},
	}

	bridge.RegisterTool(Tool{
		Name:        "schema-tool",
		Description: "Tool with schema",
		InputSchema: expectedSchema,
	})

	schema, err := bridge.GetToolSchema("schema-tool")
	if err != nil {
		t.Errorf("GetToolSchema failed: %v", err)
	}

	if schema.Type != expectedSchema.Type {
		t.Errorf("expected type %s, got %s", expectedSchema.Type, schema.Type)
	}

	if len(schema.Properties) != len(expectedSchema.Properties) {
		t.Errorf("expected %d properties, got %d", len(expectedSchema.Properties), len(schema.Properties))
	}
}

func TestBridge_GetToolSchema_NonExistent(t *testing.T) {
	bridge := NewBridge()

	_, err := bridge.GetToolSchema("non-existent")
	if err == nil {
		t.Error("GetToolSchema should return error for non-existent tool")
	}
}

func TestBridge_ConcurrentAccess(t *testing.T) {
	bridge := NewBridge()

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 50

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				toolName := fmt.Sprintf("tool-%d-%d", id, j)
				bridge.RegisterTool(Tool{
					Name:        toolName,
					Description: "Concurrent tool",
				})
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = bridge.ListTools()
			}
		}(i)
	}

	// Concurrent resource operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				resourceName := fmt.Sprintf("resource-%d-%d", id, j)
				bridge.RegisterResource(Resource{
					Name:        resourceName,
					Description: "Concurrent resource",
				})
			}
		}(i)
	}

	// Concurrent resource reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = bridge.ListResources()
			}
		}(i)
	}

	wg.Wait()

	// Verify we have the expected number of tools and resources
	tools := bridge.ListTools()
	expectedTools := numGoroutines * numOperations
	if len(tools) != expectedTools {
		t.Errorf("expected %d tools, got %d", expectedTools, len(tools))
	}

	resources := bridge.ListResources()
	expectedResources := numGoroutines * numOperations
	if len(resources) != expectedResources {
		t.Errorf("expected %d resources, got %d", expectedResources, len(resources))
	}
}

func TestBridge_ConcurrentToolCalls(t *testing.T) {
	bridge := NewBridge()

	// Register a tool with handler
	bridge.RegisterTool(Tool{
		Name:        "counter",
		Description: "A counter tool",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"increment": {Type: "number", Description: "Increment value"},
			},
		},
	})

	var callCount int
	var mu sync.Mutex

	handler := func(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
		mu.Lock()
		defer mu.Unlock()
		callCount++
		return callCount, nil
	}
	bridge.RegisterToolHandler("counter", handler)

	var wg sync.WaitGroup
	numGoroutines := 20
	numCalls := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numCalls; j++ {
				_, err := bridge.CallTool(context.Background(), "counter", map[string]interface{}{
					"increment": 1.0,
				})
				if err != nil {
					t.Errorf("CallTool failed: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	expectedCalls := numGoroutines * numCalls
	if callCount != expectedCalls {
		t.Errorf("expected %d calls, got %d", expectedCalls, callCount)
	}
}

func TestToolHandler_ContextPropagation(t *testing.T) {
	bridge := NewBridge()

	bridge.RegisterTool(Tool{
		Name:        "context-check",
		Description: "Checks context",
	})

	var receivedCtx context.Context
	handler := func(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
		receivedCtx = ctx
		return "ok", nil
	}
	bridge.RegisterToolHandler("context-check", handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := bridge.CallTool(ctx, "context-check", map[string]interface{}{})
	if err != nil {
		t.Errorf("CallTool failed: %v", err)
	}

	if receivedCtx == nil {
		t.Error("handler did not receive context")
	}

	// Check that the context is the one we passed
	select {
	case <-receivedCtx.Done():
		t.Error("context should not be done yet")
	default:
		// Expected - context is not done
	}

	cancel()

	select {
	case <-receivedCtx.Done():
		// Expected - context is now done
	default:
		t.Error("context should be done after cancel")
	}
}

// TestInputSchema_RequiredEmpty tests that Required can be empty
func TestInputSchema_RequiredEmpty(t *testing.T) {
	bridge := NewBridge()

	schema := InputSchema{
		Type: "object",
		Properties: map[string]Property{
			"optional": {Type: "string", Description: "Optional parameter"},
		},
		// Required is omitted (empty)
	}

	tool := Tool{
		Name:        "optional-params",
		Description: "Tool with optional params",
		InputSchema: schema,
	}

	err := bridge.RegisterTool(tool)
	if err != nil {
		t.Errorf("RegisterTool failed: %v", err)
	}

	retrievedSchema, err := bridge.GetToolSchema("optional-params")
	if err != nil {
		t.Errorf("GetToolSchema failed: %v", err)
	}

	if len(retrievedSchema.Required) != 0 {
		t.Errorf("expected empty Required, got %v", retrievedSchema.Required)
	}
}

// TestResource_MIMETypeOptional tests that MIMEType is optional
func TestResource_MIMETypeOptional(t *testing.T) {
	bridge := NewBridge()

	resource := Resource{
		Name:        "no-mime-resource",
		Description: "Resource without MIME type",
		// MIMEType is omitted
	}

	err := bridge.RegisterResource(resource)
	if err != nil {
		t.Errorf("RegisterResource failed: %v", err)
	}

	resources := bridge.ListResources()
	if len(resources) != 1 {
		t.Errorf("expected 1 resource, got %d", len(resources))
	}

	if resources[0].MIMEType != "" {
		t.Errorf("expected empty MIMEType, got %s", resources[0].MIMEType)
	}
}

// TestRegisterToolHandler_EmptyName tests that RegisterToolHandler returns error for empty name
func TestRegisterToolHandler_EmptyName(t *testing.T) {
	bridge := NewBridge()

	handler := func(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
		return "result", nil
	}

	err := bridge.RegisterToolHandler("", handler)
	if err == nil {
		t.Error("RegisterToolHandler should return error for empty tool name")
	}
}

// TestCallTool_EmptyName tests that CallTool returns error for empty tool name
func TestCallTool_EmptyName(t *testing.T) {
	bridge := NewBridge()

	_, err := bridge.CallTool(context.Background(), "", map[string]interface{}{})
	if err == nil {
		t.Error("CallTool should return error for empty tool name")
	}
}

// TestGetToolSchema_EmptyName tests that GetToolSchema returns error for empty tool name
func TestGetToolSchema_EmptyName(t *testing.T) {
	bridge := NewBridge()

	_, err := bridge.GetToolSchema("")
	if err == nil {
		t.Error("GetToolSchema should return error for empty tool name")
	}
}
