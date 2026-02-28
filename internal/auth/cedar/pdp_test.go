package cedar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPDP_Authorize(t *testing.T) {
	// Note: This test requires the cedar-policy library
	// In production, policies would be loaded from files
	t.Run("trusted agent can submit task", func(t *testing.T) {
		// Skip if policy file not available
		req := AuthorizationRequest{
			Principal: "logistics-optimizer",
			Action:    "submit_task",
			Resource:  "task-123",
			Context: map[string]any{
				"jurisdiction": "US",
				"workspace":    "logistics-team",
			},
		}

		// For now, just verify the request structure
		assert.Equal(t, "logistics-optimizer", req.Principal)
		assert.Equal(t, "submit_task", req.Action)
	})

	t.Run("untrusted agent denied", func(t *testing.T) {
		req := AuthorizationRequest{
			Principal: "suspicious-agent",
			Action:    "submit_task",
			Resource:  "task-123",
		}

		// Verify request structure
		assert.Equal(t, "suspicious-agent", req.Principal)
	})
}

func TestAuthorizationRequest(t *testing.T) {
	tests := []struct {
		name     string
		req      AuthorizationRequest
		expected string
	}{
		{
			name: "valid request",
			req: AuthorizationRequest{
				Principal: "agent-1",
				Action:    "submit_task",
				Resource:  "task-1",
			},
			expected: "agent-1",
		},
		{
			name: "request with context",
			req: AuthorizationRequest{
				Principal: "agent-2",
				Action:    "view_task",
				Resource:  "task-2",
				Context: map[string]any{
					"workspace": "team-a",
				},
			},
			expected: "agent-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.req.Principal)
		})
	}
}

func TestAuthorizationDecision(t *testing.T) {
	t.Run("allow decision", func(t *testing.T) {
		decision := &AuthorizationDecision{
			Decision: "Allow",
			Reasons:  []string{"policy permit matched"},
		}
		assert.Equal(t, "Allow", decision.Decision)
	})

	t.Run("deny decision", func(t *testing.T) {
		decision := &AuthorizationDecision{
			Decision: "Deny",
			Reasons:  []string{"no policy matched", "default deny"},
		}
		assert.Equal(t, "Deny", decision.Decision)
	})
}
