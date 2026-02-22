// Package e2e provides end-to-end integration tests for A2A Gateway
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestA2A_Gateway_FullFlow tests the complete A2A protocol flow
func TestA2A_Gateway_FullFlow(t *testing.T) {
	ctx := context.Background()
	client := &http.Client{Timeout: 30 * time.Second}

	t.Run("agent_discovery", func(t *testing.T) {
		// Request agent card from well-known endpoint
		resp, err := client.Get("http://localhost:8090/.well-known/agent.json")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
	})

	t.Run("synchronous_task_submission", func(t *testing.T) {
		task := map[string]any{
			"task_id": "task-sync-" + time.Now().Format("20060102150405"),
			"message_object": map[string]any{
				"role": "user",
				"parts": []map[string]any{
					{"type": "text", "text": "What is the current temperature?"},
				},
			},
			"capabilities": []string{"a2a", "streaming"},
		}

		body, _ := json.Marshal(task)
		req, _ := http.NewRequestWithContext(ctx, "POST",
			"http://localhost:8090/a2a/tasks",
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/agent-task+json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
		assert.NotEmpty(t, resp.Header.Get("X-A2A-Validated"))
	})

	t.Run("asynchronous_task_with_webhook", func(t *testing.T) {
		task := map[string]any{
			"task_id": "task-async-" + time.Now().Format("20060102150405"),
			"message_object": map[string]any{
				"role": "user",
				"parts": []map[string]any{
					{"type": "text", "text": "Analyze this document"},
				},
			},
			"capabilities": []string{"a2a"},
			"webhook": map[string]string{
				"url": "http://localhost:8888/callback",
			},
		}

		body, _ := json.Marshal(task)
		req, _ := http.NewRequestWithContext(ctx, "POST",
			"http://localhost:8090/a2a/tasks",
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/agent-task+json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 202, resp.StatusCode)

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)
		assert.NotEmpty(t, result["task_id"])
	})

	t.Run("task_status_polling", func(t *testing.T) {
		resp, err := client.Get("http://localhost:8090/a2a/tasks/test-task")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("task_cancellation", func(t *testing.T) {
		req, _ := http.NewRequestWithContext(ctx, "DELETE",
			"http://localhost:8090/a2a/tasks/test-task", nil)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Contains(t, []int{200, 204, 404}, resp.StatusCode)
	})
}

// TestA2A_Protocol_Validation tests payload validation
func TestA2A_Protocol_Validation(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	t.Run("missing_task_id_rejected", func(t *testing.T) {
		task := map[string]any{
			"message_object": map[string]any{
				"role": "user",
				"parts": []map[string]any{
					{"type": "text", "text": "Hello"},
				},
			},
		}

		body, _ := json.Marshal(task)
		resp, err := client.Post("http://localhost:8090/a2a/tasks",
			"application/agent-task+json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("empty_parts_rejected", func(t *testing.T) {
		task := map[string]any{
			"task_id": "test-task",
			"message_object": map[string]any{
				"role":    "user",
				"parts":   []any{},
			},
		}

		body, _ := json.Marshal(task)
		resp, err := client.Post("http://localhost:8090/a2a/tasks",
			"application/agent-task+json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 400, resp.StatusCode)
	})
}

// TestA2A_RateLimiting tests token-based rate limiting
func TestA2A_RateLimiting(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	t.Run("large_payload_rate_limited", func(t *testing.T) {
		task := map[string]any{
			"task_id": "task-large",
			"message_object": map[string]any{
				"role": "user",
				"parts": []map[string]any{
					{"type": "text", "text": string(make([]byte, 1000000))},
				},
			},
		}

		body, _ := json.Marshal(task)
		resp, err := client.Post("http://localhost:8090/a2a/tasks",
			"application/agent-task+json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should be rate limited due to excessive tokens
		assert.Equal(t, 429, resp.StatusCode)
	})
}

// TestA2A_Streaming tests SSE streaming functionality
func TestA2A_Streaming(t *testing.T) {
	client := &http.Client{Timeout: 30 * time.Second}

	t.Run("sse_stream_established", func(t *testing.T) {
		req, _ := http.NewRequest("GET",
			"http://localhost:8090/a2a/tasks/test-task/stream", nil)
		req.Header.Set("Accept", "text/event-stream")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")
	})
}
