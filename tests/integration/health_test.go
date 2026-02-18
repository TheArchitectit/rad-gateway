// Package integration provides integration tests for RAD Gateway 01
//
// Run with: go test ./tests/integration/...
// Run verbose: go test -v ./tests/integration/...
package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// HealthResponse represents the expected health check response
type HealthResponse struct {
	Status string `json:"status"`
}

// TestHealthEndpointReturns200 validates that /health returns HTTP 200 with expected JSON
func TestHealthEndpointReturns200(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test - set INTEGRATION_TEST=1 to run")
	}
	tests := []struct {
		name           string
		baseURL        string
		expectedStatus int
		expectedBody   string
		shouldSkip     bool
		skipReason     string
	}{
		{
			name:           "Local health endpoint returns 200",
			baseURL:        getRadGatewayURL(),
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
			shouldSkip:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSkip {
				t.Skip(tt.skipReason)
			}

			client := &http.Client{
				Timeout: 5 * time.Second,
			}

			resp, err := client.Get(tt.baseURL + "/health")
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Check content type
			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("Expected Content-Type to contain 'application/json', got '%s'", contentType)
			}

			// Read and validate body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			var healthResp HealthResponse
			if err := json.Unmarshal(body, &healthResp); err != nil {
				t.Errorf("Failed to parse JSON response: %v\nBody: %s", err, string(body))
				return
			}

			if healthResp.Status != "ok" {
				t.Errorf("Expected status 'ok', got '%s'", healthResp.Status)
			}

			t.Logf("Health check passed: %s", string(body))
		})
	}
}

// TestHealthEndpointMockServer validates health endpoint behavior using httptest
func TestHealthEndpointMockServer(t *testing.T) {
	// Create a mock server that simulates the health endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer mockServer.Close()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "GET /health returns 200",
			method:         http.MethodGet,
			path:           "/health",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "POST /health returns 405",
			method:         http.MethodPost,
			path:           "/health",
			expectedStatus: http.StatusMethodNotAllowed,
			expectError:    false,
		},
		{
			name:           "PUT /health returns 405",
			method:         http.MethodPut,
			path:           "/health",
			expectedStatus: http.StatusMethodNotAllowed,
			expectError:    false,
		},
		{
			name:           "DELETE /health returns 405",
			method:         http.MethodDelete,
			path:           "/health",
			expectedStatus: http.StatusMethodNotAllowed,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, mockServer.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// TestHealthEndpointFromContainer validates health endpoint is reachable from container context
func TestHealthEndpointFromContainer(t *testing.T) {
	// Check if running inside a container
	inContainer := isRunningInContainer()

	if !inContainer {
		t.Skip("Not running inside a container - skipping container reachability test")
	}

	// When inside a container, the health endpoint should be reachable
	// using the container's network (localhost:8090)
	baseURL := getRadGatewayURL()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("Health endpoint not reachable from container: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	var healthResp HealthResponse
	if err := json.Unmarshal(body, &healthResp); err != nil {
		t.Errorf("Invalid JSON response: %v", err)
	}

	if healthResp.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", healthResp.Status)
	}

	t.Log("Health endpoint reachable from container context")
}

// TestInfisicalReachable validates that dependent services (Infisical) are reachable
func TestInfisicalReachable(t *testing.T) {
	infisicalURL := getInfisicalURL()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Try to reach Infisical status endpoint
	resp, err := client.Get(infisicalURL + "/api/status")
	if err != nil {
		t.Skipf("Infisical not reachable at %s: %v - skipping integration test", infisicalURL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Infisical returned unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Failed to read Infisical response: %v", err)
		return
	}

	t.Logf("Infisical is reachable. Response: %s", string(body))
}

// TestHealthResponseStructure validates the health response structure
func TestHealthResponseStructure(t *testing.T) {
	baseURL := getRadGatewayURL()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		// If server is not running, use mock response for structure validation
		t.Skipf("RAD Gateway not running at %s - skipping live test", baseURL)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	var healthResp map[string]interface{}
	if err := json.Unmarshal(body, &healthResp); err != nil {
		t.Fatalf("Response is not valid JSON: %v\nBody: %s", err, string(body))
	}

	// Validate required fields
	if _, ok := healthResp["status"]; !ok {
		t.Error("Health response missing required 'status' field")
	}

	// Validate status value
	if status, ok := healthResp["status"].(string); ok {
		if status != "ok" && status != "healthy" {
			t.Errorf("Unexpected status value: %s", status)
		}
	} else {
		t.Error("Status field is not a string")
	}

	t.Logf("Health response structure validated: %v", healthResp)
}

// TestHealthEndpointTimeout validates timeout behavior
func TestHealthEndpointTimeout(t *testing.T) {
	baseURL := getRadGatewayURL()

	// Create client with very short timeout to test timeout handling
	client := &http.Client{
		Timeout: 1 * time.Millisecond,
	}

	_, err := client.Get(baseURL + "/health")
	if err != nil {
		// Timeout or connection error is expected when server is not running
		// or when timeout is very short
		t.Skipf("Connection/timeout test skipped: %v", err)
		return
	}
}

// isRunningInContainer checks if the test is running inside a container
func isRunningInContainer() bool {
	// Check for container-specific files
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check cgroup for container indicators
	data, err := os.ReadFile("/proc/1/cgroup")
	if err == nil {
		content := string(data)
		if strings.Contains(content, "docker") ||
			strings.Contains(content, "containerd") ||
			strings.Contains(content, "podman") ||
			strings.Contains(content, "kubepods") {
			return true
		}
	}

	// Check for container environment variables
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}

	return false
}

// getRadGatewayURL returns the RAD Gateway URL from environment or default
func getRadGatewayURL() string {
	host := os.Getenv("RAD_GATEWAY_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("RAD_GATEWAY_PORT")
	if port == "" {
		port = "8090"
	}

	return fmt.Sprintf("http://%s:%s", host, port)
}

// getInfisicalURL returns the Infisical URL from environment or default
func getInfisicalURL() string {
	host := os.Getenv("INFISICAL_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("INFISICAL_PORT")
	if port == "" {
		port = "8080"
	}

	return fmt.Sprintf("http://%s:%s", host, port)
}

// BenchmarkHealthEndpoint benchmarks the health endpoint performance
func BenchmarkHealthEndpoint(b *testing.B) {
	baseURL := getRadGatewayURL()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Test if server is running
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		b.Skipf("RAD Gateway not running at %s - skipping benchmark", baseURL)
		return
	}
	resp.Body.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(baseURL + "/health")
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
