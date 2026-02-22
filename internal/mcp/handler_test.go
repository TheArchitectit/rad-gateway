package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_BuiltinTools(t *testing.T) {
	h := NewHandler()
	
	tests := []struct {
		name       string
		tool       string
		input      map[string]interface{}
		wantSuccess bool
	}{
		{
			name:       "echo tool",
			tool:       "echo",
			input:      map[string]interface{}{"message": "hello"},
			wantSuccess: true,
		},
		{
			name:       "time tool",
			tool:       "time",
			input:      map[string]interface{}{},
			wantSuccess: true,
		},
		{
			name:       "json_parse tool",
			tool:       "json_parse",
			input:      map[string]interface{}{"data": `{"key": "value"}`},
			wantSuccess: true,
		},
		{
			name:       "unknown tool",
			tool:       "unknown",
			input:      map[string]interface{}{},
			wantSuccess: true, // Fallback behavior
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := Request{
				Tool:    tt.tool,
				Input:   tt.input,
				Session: "test-session",
			}
			
			body, _ := json.Marshal(req)
			r := httptest.NewRequest(http.MethodPost, "/mcp/v1/stdio", bytes.NewReader(body))
			w := httptest.NewRecorder()
			
			h.handleStdio(w, r)
			
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
			
			var resp Response
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}
			
			if resp.Success != tt.wantSuccess {
				t.Errorf("Expected success=%v, got %v", tt.wantSuccess, resp.Success)
			}
			
			if resp.Tool != tt.tool {
				t.Errorf("Expected tool=%s, got %s", tt.tool, resp.Tool)
			}
			
			if resp.Session != "test-session" {
				t.Errorf("Expected session=test-session, got %s", resp.Session)
			}
			
			if resp.Duration < 0 {
				t.Error("Expected positive duration")
			}
		})
	}
}

func TestHandler_Health(t *testing.T) {
	h := NewHandler()
	
	r := httptest.NewRequest(http.MethodGet, "/mcp/v1/health", nil)
	w := httptest.NewRecorder()
	
	h.handleHealth(w, r)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if status, ok := resp["status"].(string); !ok || status == "" {
		t.Error("Expected status in response")
	}
	
	if service, ok := resp["service"].(string); !ok || service != "mcp" {
		t.Error("Expected service='mcp' in response")
	}
}

func TestHandler_ToolList(t *testing.T) {
	h := NewHandler()
	
	r := httptest.NewRequest(http.MethodGet, "/mcp/v1/tools/list", nil)
	w := httptest.NewRecorder()
	
	h.handleToolList(w, r)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	tools, ok := resp["tools"].([]interface{})
	if !ok {
		t.Fatal("Expected tools array in response")
	}
	
	if len(tools) == 0 {
		t.Error("Expected at least one tool")
	}
	
	if count, ok := resp["count"].(float64); !ok || count <= 0 {
		t.Error("Expected positive count")
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	h := NewHandler()
	
	// Test GET on stdio endpoint (should be POST)
	r := httptest.NewRequest(http.MethodGet, "/mcp/v1/stdio", nil)
	w := httptest.NewRecorder()
	
	h.handleStdio(w, r)
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandler_InvalidRequest(t *testing.T) {
	h := NewHandler()
	
	// Test with invalid JSON
	r := httptest.NewRequest(http.MethodPost, "/mcp/v1/stdio", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()
	
	h.handleStdio(w, r)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandler_MissingTool(t *testing.T) {
	h := NewHandler()
	
	req := Request{
		Tool:    "",
		Input:   map[string]interface{}{},
		Session: "test-session",
	}
	
	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/mcp/v1/stdio", bytes.NewReader(body))
	w := httptest.NewRecorder()
	
	h.handleStdio(w, r)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
