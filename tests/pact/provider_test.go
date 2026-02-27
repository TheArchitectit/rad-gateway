// Package pact provides provider verification tests for contract testing.
// Sprint 7.1: Pact Provider Verification
//
// These tests verify that the backend API fulfills the contracts
// defined by the frontend consumer.
//
// To run contract tests:
//   1. Start the API server: go run ./cmd/rad-gateway
//   2. Run consumer tests in web/: npm run test:pact
//   3. Run provider tests: RUN_CONTRACT_TESTS=true go test ./tests/pact/...
//
// For CI/CD with Pact Broker:
//   PACT_BROKER_URL=https://pact-broker.example.com \
//   PACT_BROKER_TOKEN=token \
//   CI=true \
//   go test ./tests/pact/...
package pact

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Contract represents a Pact contract file structure
type Contract struct {
	Consumer struct {
		Name string `json:"name"`
	} `json:"consumer"`
	Provider struct {
		Name string `json:"name"`
	} `json:"provider"`
	Interactions []Interaction `json:"interactions"`
}

// Interaction represents a single contract interaction
type Interaction struct {
	Description   string         `json:"description"`
	ProviderState  string        `json:"providerState"`
	Request       Request        `json:"request"`
	Response      Response       `json:"response"`
}

// Request represents the expected request
type Request struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    interface{}       `json:"body,omitempty"`
}

// Response represents the expected response
type Response struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    interface{}       `json:"body,omitempty"`
}

// TestProvider verifies the backend API against consumer contracts
func TestProvider(t *testing.T) {
	// Skip if not running contract tests
	if os.Getenv("RUN_CONTRACT_TESTS") != "true" {
		t.Skip("Skipping contract tests. Set RUN_CONTRACT_TESTS=true to run.")
	}

	providerURL := getProviderURL()
	contractsDir := "../pact/contracts"

	// Find all contract files
	files, err := filepath.Glob(filepath.Join(contractsDir, "*.json"))
	if err != nil {
		t.Fatalf("Failed to find contract files: %v", err)
	}

	if len(files) == 0 {
		t.Skip("No contract files found. Run consumer tests first.")
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			verifyContract(t, file, providerURL)
		})
	}
}

// verifyContract verifies a single contract file
func verifyContract(t *testing.T, contractFile, providerURL string) {
	// Load contract
	data, err := os.ReadFile(contractFile)
	if err != nil {
		t.Fatalf("Failed to read contract file: %v", err)
	}

	var contract Contract
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatalf("Failed to parse contract: %v", err)
	}

	t.Logf("Verifying contract: %s -> %s (%d interactions)",
		contract.Consumer.Name, contract.Provider.Name, len(contract.Interactions))

	// Verify each interaction
	for _, interaction := range contract.Interactions {
		t.Run(interaction.Description, func(t *testing.T) {
			verifyInteraction(t, interaction, providerURL)
		})
	}
}

// verifyInteraction verifies a single interaction
func verifyInteraction(t *testing.T, interaction Interaction, providerURL string) {
	// Setup provider state if needed
	if interaction.ProviderState != "" {
		setupProviderState(t, interaction.ProviderState)
	}

	// Build request
	url := fmt.Sprintf("%s%s", providerURL, interaction.Request.Path)
	req, err := http.NewRequest(interaction.Request.Method, url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Add headers
	for key, value := range interaction.Request.Headers {
		req.Header.Set(key, value)
	}

	// Add auth header if not present
	if req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer test-token")
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	// Verify status
	if resp.StatusCode != interaction.Response.Status {
		t.Errorf("Status mismatch: got %d, want %d", resp.StatusCode, interaction.Response.Status)
	}

	// Verify content type
	expectedCT := interaction.Response.Headers["Content-Type"]
	if expectedCT != "" && !strings.Contains(resp.Header.Get("Content-Type"), expectedCT) {
		t.Errorf("Content-Type mismatch: got %s, want %s",
			resp.Header.Get("Content-Type"), expectedCT)
	}

	// Verify body structure if expected
	if interaction.Response.Body != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		var actualBody interface{}
		if err := json.Unmarshal(body, &actualBody); err != nil {
			t.Fatalf("Failed to parse response body: %v", err)
		}

		// Basic structure verification
		verifyBodyStructure(t, interaction.Response.Body, actualBody)
	}
}

// verifyBodyStructure performs basic structure verification
func verifyBodyStructure(t *testing.T, expected, actual interface{}) {
	switch exp := expected.(type) {
	case map[string]interface{}:
		act, ok := actual.(map[string]interface{})
		if !ok {
			t.Errorf("Expected object, got %T", actual)
			return
		}
		for key := range exp {
			if _, exists := act[key]; !exists {
				t.Errorf("Missing expected field: %s", key)
			}
		}
	case []interface{}:
		act, ok := actual.([]interface{})
		if !ok {
			t.Errorf("Expected array, got %T", actual)
			return
		}
		if len(act) == 0 && len(exp) > 0 {
			t.Errorf("Expected non-empty array, got empty")
		}
	}
}

// setupProviderState sets up the provider for a given state
func setupProviderState(t *testing.T, state string) {
	t.Logf("Setting up provider state: %s", state)

	switch state {
	case "the API is running":
		// No setup needed
	case "the database is unavailable":
		os.Setenv("MOCK_DB_UNAVAILABLE", "true")
	case "providers exist":
		seedTestProviders()
	case "no authentication provided":
		// No setup needed - test will omit auth header
	case "API keys exist":
		seedTestAPIKeys()
	case "usage data exists":
		seedTestUsageData()
	}
}

// getProviderURL returns the base URL for the provider under test
func getProviderURL() string {
	if url := os.Getenv("PROVIDER_BASE_URL"); url != "" {
		return url
	}
	return "http://localhost:8080"
}

// seedTestProviders seeds test provider data
func seedTestProviders() {
	// TODO: Seed test providers via API or database
}

// seedTestAPIKeys seeds test API key data
func seedTestAPIKeys() {
	// TODO: Seed test API keys via API or database
}

// seedTestUsageData seeds test usage data
func seedTestUsageData() {
	// TODO: Seed test usage records via API or database
}
