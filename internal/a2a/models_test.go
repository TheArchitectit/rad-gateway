// Package a2a provides A2A (Agent-to-Agent) protocol support for RAD Gateway.
package a2a

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestAgentCard_JSONRoundTrip(t *testing.T) {
	now := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)

	original := AgentCard{
		Name:        "Test Agent",
		Description: "A test agent for A2A protocol",
		URL:         "https://example.com/a2a",
		Version:     "1.0.0",
		Capabilities: Capabilities{
			Streaming:              true,
			PushNotifications:      false,
			StateTransitionHistory: true,
		},
		Skills: []Skill{
			{
				ID:          "skill-1",
				Name:        "Text Generation",
				Description: "Generate text based on prompts",
				Tags:        []string{"text", "generation"},
				Examples:    []string{"Write a story", "Summarize this"},
				Input: &SkillSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"prompt": map[string]interface{}{"type": "string"},
					},
					Required: []string{"prompt"},
				},
				Output: &SkillSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"text": map[string]interface{}{"type": "string"},
					},
					Required: []string{"text"},
				},
			},
			{
				ID:          "skill-2",
				Name:        "Code Analysis",
				Description: "Analyze code for issues",
				Tags:        []string{"code", "analysis"},
				Examples:    []string{"Find bugs", "Optimize this"},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal AgentCard: %v", err)
	}

	// Unmarshal back
	var decoded AgentCard
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal AgentCard: %v", err)
	}

	// Verify all fields round-trip correctly
	if decoded.Name != original.Name {
		t.Errorf("Name mismatch: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Description != original.Description {
		t.Errorf("Description mismatch: got %q, want %q", decoded.Description, original.Description)
	}
	if decoded.URL != original.URL {
		t.Errorf("URL mismatch: got %q, want %q", decoded.URL, original.URL)
	}
	if decoded.Version != original.Version {
		t.Errorf("Version mismatch: got %q, want %q", decoded.Version, original.Version)
	}
	if !decoded.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", decoded.CreatedAt, original.CreatedAt)
	}
	if !decoded.UpdatedAt.Equal(original.UpdatedAt) {
		t.Errorf("UpdatedAt mismatch: got %v, want %v", decoded.UpdatedAt, original.UpdatedAt)
	}

	// Verify capabilities
	if decoded.Capabilities.Streaming != original.Capabilities.Streaming {
		t.Errorf("Capabilities.Streaming mismatch: got %v, want %v", decoded.Capabilities.Streaming, original.Capabilities.Streaming)
	}
	if decoded.Capabilities.PushNotifications != original.Capabilities.PushNotifications {
		t.Errorf("Capabilities.PushNotifications mismatch: got %v, want %v", decoded.Capabilities.PushNotifications, original.Capabilities.PushNotifications)
	}
	if decoded.Capabilities.StateTransitionHistory != original.Capabilities.StateTransitionHistory {
		t.Errorf("Capabilities.StateTransitionHistory mismatch: got %v, want %v", decoded.Capabilities.StateTransitionHistory, original.Capabilities.StateTransitionHistory)
	}

	// Verify skills
	if len(decoded.Skills) != len(original.Skills) {
		t.Fatalf("Skills length mismatch: got %d, want %d", len(decoded.Skills), len(original.Skills))
	}

	for i, skill := range decoded.Skills {
		origSkill := original.Skills[i]
		if skill.ID != origSkill.ID {
			t.Errorf("Skill[%d].ID mismatch: got %q, want %q", i, skill.ID, origSkill.ID)
		}
		if skill.Name != origSkill.Name {
			t.Errorf("Skill[%d].Name mismatch: got %q, want %q", i, skill.Name, origSkill.Name)
		}
		if skill.Description != origSkill.Description {
			t.Errorf("Skill[%d].Description mismatch: got %q, want %q", i, skill.Description, origSkill.Description)
		}
		if !reflect.DeepEqual(skill.Tags, origSkill.Tags) {
			t.Errorf("Skill[%d].Tags mismatch: got %v, want %v", i, skill.Tags, origSkill.Tags)
		}
		if !reflect.DeepEqual(skill.Examples, origSkill.Examples) {
			t.Errorf("Skill[%d].Examples mismatch: got %v, want %v", i, skill.Examples, origSkill.Examples)
		}

		// Verify input schema
		if skill.Input != nil && origSkill.Input != nil {
			if skill.Input.Type != origSkill.Input.Type {
				t.Errorf("Skill[%d].Input.Type mismatch: got %q, want %q", i, skill.Input.Type, origSkill.Input.Type)
			}
			if !reflect.DeepEqual(skill.Input.Required, origSkill.Input.Required) {
				t.Errorf("Skill[%d].Input.Required mismatch: got %v, want %v", i, skill.Input.Required, origSkill.Input.Required)
			}
		} else if (skill.Input != nil) != (origSkill.Input != nil) {
			t.Errorf("Skill[%d].Input nil mismatch: got %v, want %v", i, skill.Input != nil, origSkill.Input != nil)
		}

		// Verify output schema
		if skill.Output != nil && origSkill.Output != nil {
			if skill.Output.Type != origSkill.Output.Type {
				t.Errorf("Skill[%d].Output.Type mismatch: got %q, want %q", i, skill.Output.Type, origSkill.Output.Type)
			}
			if !reflect.DeepEqual(skill.Output.Required, origSkill.Output.Required) {
				t.Errorf("Skill[%d].Output.Required mismatch: got %v, want %v", i, skill.Output.Required, origSkill.Output.Required)
			}
		} else if (skill.Output != nil) != (origSkill.Output != nil) {
			t.Errorf("Skill[%d].Output nil mismatch: got %v, want %v", i, skill.Output != nil, origSkill.Output != nil)
		}
	}
}

func TestAgentCard_JSONFieldNames(t *testing.T) {
	card := AgentCard{
		Name:        "Test",
		Description: "Test Description",
		URL:         "https://test.com",
		Version:     "1.0",
		CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Capabilities: Capabilities{
			Streaming:              true,
			PushNotifications:      true,
			StateTransitionHistory: false,
		},
		Skills: []Skill{},
	}

	data, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify JSON field names match A2A spec (camelCase)
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal to raw map: %v", err)
	}

	expectedFields := map[string]bool{
		"name":                   false,
		"description":            false,
		"url":                    false,
		"version":                false,
		"capabilities":           false,
		"skills":                 false,
		"createdAt":              false,
		"updatedAt":              false,
	}

	for field := range raw {
		if _, ok := expectedFields[field]; ok {
			expectedFields[field] = true
		}
	}

	for field, found := range expectedFields {
		if !found {
			t.Errorf("Expected field %q not found in JSON", field)
		}
	}

	// Verify capabilities sub-fields
	if caps, ok := raw["capabilities"].(map[string]interface{}); ok {
		capFields := []string{"streaming", "pushNotifications", "stateTransitionHistory"}
		for _, field := range capFields {
			if _, exists := caps[field]; !exists {
				t.Errorf("Expected capability field %q not found", field)
			}
		}
	} else {
		t.Error("capabilities field is not an object")
	}
}

func TestTask_JSONRoundTrip(t *testing.T) {
	now := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	completedAt := now.Add(time.Hour)

	original := Task{
		ID:        "task-123",
		SessionID: "session-456",
		Status:    TaskStateWorking,
		Artifacts: []Artifact{
			{
				ID:   "artifact-1",
				Type: "text",
				Parts: []Part{
					{Type: "text", Text: "Hello, world!"},
					{Type: "data", Text: "Some data"},
				},
				Metadata: map[string]interface{}{"key": "value"},
			},
		},
		History: []Message{
			{
				Role:    "user",
				Content: "Do something",
				Parts:   []Part{{Type: "text", Text: "Do something"}},
			},
			{
				Role:    "agent",
				Content: "Done!",
				Parts:   []Part{{Type: "text", Text: "Done!"}},
			},
		},
		Message: Message{
			Role:    "user",
			Content: "Initial message",
			Parts:   []Part{{Type: "text", Text: "Initial message"}},
		},
		CreatedAt:   now,
		UpdatedAt:   now,
		CompletedAt: &completedAt,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal Task: %v", err)
	}

	var decoded Task
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Task: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %q, want %q", decoded.ID, original.ID)
	}
	if decoded.SessionID != original.SessionID {
		t.Errorf("SessionID mismatch: got %q, want %q", decoded.SessionID, original.SessionID)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status mismatch: got %q, want %q", decoded.Status, original.Status)
	}
	if !decoded.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", decoded.CreatedAt, original.CreatedAt)
	}
	if !decoded.UpdatedAt.Equal(original.UpdatedAt) {
		t.Errorf("UpdatedAt mismatch: got %v, want %v", decoded.UpdatedAt, original.UpdatedAt)
	}
	if decoded.CompletedAt == nil || !decoded.CompletedAt.Equal(*original.CompletedAt) {
		t.Errorf("CompletedAt mismatch: got %v, want %v", decoded.CompletedAt, original.CompletedAt)
	}

	if len(decoded.Artifacts) != len(original.Artifacts) {
		t.Errorf("Artifacts length mismatch: got %d, want %d", len(decoded.Artifacts), len(original.Artifacts))
	}

	if len(decoded.History) != len(original.History) {
		t.Errorf("History length mismatch: got %d, want %d", len(decoded.History), len(original.History))
	}
}

func TestSendTaskRequest_JSONRoundTrip(t *testing.T) {
	original := SendTaskRequest{
		ID:        "req-123",
		SessionID: "session-456",
		SkillID:   "skill-789",
		Message: Message{
			Role:    "user",
			Content: "Hello",
			Parts:   []Part{{Type: "text", Text: "Hello"}},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal SendTaskRequest: %v", err)
	}

	var decoded SendTaskRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal SendTaskRequest: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %q, want %q", decoded.ID, original.ID)
	}
	if decoded.SessionID != original.SessionID {
		t.Errorf("SessionID mismatch: got %q, want %q", decoded.SessionID, original.SessionID)
	}
	if decoded.SkillID != original.SkillID {
		t.Errorf("SkillID mismatch: got %q, want %q", decoded.SkillID, original.SkillID)
	}
	if decoded.Message.Role != original.Message.Role {
		t.Errorf("Message.Role mismatch: got %q, want %q", decoded.Message.Role, original.Message.Role)
	}
	if decoded.Message.Content != original.Message.Content {
		t.Errorf("Message.Content mismatch: got %q, want %q", decoded.Message.Content, original.Message.Content)
	}
}

func TestSendTaskResponse_JSONRoundTrip(t *testing.T) {
	now := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		original SendTaskResponse
	}{
		{
			name: "success response",
			original: SendTaskResponse{
				Task: &Task{
					ID:        "task-123",
					SessionID: "session-456",
					Status:    TaskStateCompleted,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Error: nil,
			},
		},
		{
			name: "error response",
			original: SendTaskResponse{
				Task: nil,
				Error: &Error{
					Code:    400,
					Message: "Invalid request",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.original)
			if err != nil {
				t.Fatalf("Failed to marshal SendTaskResponse: %v", err)
			}

			var decoded SendTaskResponse
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal SendTaskResponse: %v", err)
			}

			if tt.original.Task != nil {
				if decoded.Task == nil {
					t.Error("Expected Task to be non-nil")
				} else if decoded.Task.ID != tt.original.Task.ID {
					t.Errorf("Task.ID mismatch: got %q, want %q", decoded.Task.ID, tt.original.Task.ID)
				}
			} else if decoded.Task != nil {
				t.Error("Expected Task to be nil")
			}

			if tt.original.Error != nil {
				if decoded.Error == nil {
					t.Error("Expected Error to be non-nil")
				} else {
					if decoded.Error.Code != tt.original.Error.Code {
						t.Errorf("Error.Code mismatch: got %d, want %d", decoded.Error.Code, tt.original.Error.Code)
					}
					if decoded.Error.Message != tt.original.Error.Message {
						t.Errorf("Error.Message mismatch: got %q, want %q", decoded.Error.Message, tt.original.Error.Message)
					}
				}
			} else if decoded.Error != nil {
				t.Error("Expected Error to be nil")
			}
		})
	}
}

func TestTaskStatus_Validation(t *testing.T) {
	tests := []struct {
		status string
		valid  bool
	}{
		{"pending", true},
		{"working", true},
		{"input-required", true},
		{"completed", true},
		{"failed", true},
		{"cancelled", true},
		{"invalid", false},
		{"", false},
		{"PENDING", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := IsValidTaskStatus(tt.status)
			if got != tt.valid {
				t.Errorf("IsValidTaskStatus(%q) = %v, want %v", tt.status, got, tt.valid)
			}
		})
	}
}

func TestTaskState_Validation(t *testing.T) {
	tests := []struct {
		state TaskState
		valid bool
	}{
		{TaskStateSubmitted, true},
		{TaskStateWorking, true},
		{TaskStateInputRequired, true},
		{TaskStateCompleted, true},
		{TaskStateCanceled, true},
		{TaskStateFailed, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			got := IsValidTaskState(tt.state)
			if got != tt.valid {
				t.Errorf("IsValidTaskState(%q) = %v, want %v", tt.state, got, tt.valid)
			}
		})
	}
}

func TestSkillSchema_JSONRoundTrip(t *testing.T) {
	original := SkillSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "The name field",
			},
			"age": map[string]interface{}{
				"type": "integer",
			},
		},
		Required: []string{"name"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal SkillSchema: %v", err)
	}

	var decoded SkillSchema
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal SkillSchema: %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: got %q, want %q", decoded.Type, original.Type)
	}
	if !reflect.DeepEqual(decoded.Required, original.Required) {
		t.Errorf("Required mismatch: got %v, want %v", decoded.Required, original.Required)
	}
}

func TestPart_JSONRoundTrip(t *testing.T) {
	original := Part{
		Type: "text",
		Text: "Hello, world!",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal Part: %v", err)
	}

	var decoded Part
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Part: %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: got %q, want %q", decoded.Type, original.Type)
	}
	if decoded.Text != original.Text {
		t.Errorf("Text mismatch: got %q, want %q", decoded.Text, original.Text)
	}
}

func TestMessage_JSONRoundTrip(t *testing.T) {
	original := Message{
		Role:    "user",
		Content: "Hello",
		Parts: []Part{
			{Type: "text", Text: "Hello"},
			{Type: "data", Text: "Some data"},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal Message: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Message: %v", err)
	}

	if decoded.Role != original.Role {
		t.Errorf("Role mismatch: got %q, want %q", decoded.Role, original.Role)
	}
	if decoded.Content != original.Content {
		t.Errorf("Content mismatch: got %q, want %q", decoded.Content, original.Content)
	}
	if len(decoded.Parts) != len(original.Parts) {
		t.Errorf("Parts length mismatch: got %d, want %d", len(decoded.Parts), len(original.Parts))
	}
}

func TestArtifact_JSONRoundTrip(t *testing.T) {
	original := Artifact{
		ID:   "artifact-123",
		Type: "text",
		Parts: []Part{
			{Type: "text", Text: "Result text"},
		},
		Metadata: map[string]interface{}{
			"key": "value",
			"num": 42.0,
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal Artifact: %v", err)
	}

	var decoded Artifact
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Artifact: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: got %q, want %q", decoded.Type, original.Type)
	}
	if len(decoded.Parts) != len(original.Parts) {
		t.Errorf("Parts length mismatch: got %d, want %d", len(decoded.Parts), len(original.Parts))
	}
}

func TestCapabilities_JSONRoundTrip(t *testing.T) {
	original := Capabilities{
		Streaming:              true,
		PushNotifications:      false,
		StateTransitionHistory: true,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal Capabilities: %v", err)
	}

	var decoded Capabilities
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Capabilities: %v", err)
	}

	if decoded.Streaming != original.Streaming {
		t.Errorf("Streaming mismatch: got %v, want %v", decoded.Streaming, original.Streaming)
	}
	if decoded.PushNotifications != original.PushNotifications {
		t.Errorf("PushNotifications mismatch: got %v, want %v", decoded.PushNotifications, original.PushNotifications)
	}
	if decoded.StateTransitionHistory != original.StateTransitionHistory {
		t.Errorf("StateTransitionHistory mismatch: got %v, want %v", decoded.StateTransitionHistory, original.StateTransitionHistory)
	}
}

func TestSkill_JSONRoundTrip(t *testing.T) {
	original := Skill{
		ID:          "skill-123",
		Name:        "Test Skill",
		Description: "A test skill",
		Tags:        []string{"test", "example"},
		Examples:    []string{"Example 1", "Example 2"},
		Input: &SkillSchema{
			Type:       "object",
			Properties: map[string]interface{}{"input": map[string]interface{}{"type": "string"}},
			Required:   []string{"input"},
		},
		Output: &SkillSchema{
			Type:       "object",
			Properties: map[string]interface{}{"output": map[string]interface{}{"type": "string"}},
			Required:   []string{"output"},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal Skill: %v", err)
	}

	var decoded Skill
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Skill: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Name != original.Name {
		t.Errorf("Name mismatch: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Description != original.Description {
		t.Errorf("Description mismatch: got %q, want %q", decoded.Description, original.Description)
	}
	if !reflect.DeepEqual(decoded.Tags, original.Tags) {
		t.Errorf("Tags mismatch: got %v, want %v", decoded.Tags, original.Tags)
	}
	if !reflect.DeepEqual(decoded.Examples, original.Examples) {
		t.Errorf("Examples mismatch: got %v, want %v", decoded.Examples, original.Examples)
	}
}

func TestError_JSONRoundTrip(t *testing.T) {
	original := Error{
		Code:    500,
		Message: "Internal server error",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal Error: %v", err)
	}

	var decoded Error
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Error: %v", err)
	}

	if decoded.Code != original.Code {
		t.Errorf("Code mismatch: got %d, want %d", decoded.Code, original.Code)
	}
	if decoded.Message != original.Message {
		t.Errorf("Message mismatch: got %q, want %q", decoded.Message, original.Message)
	}
}
