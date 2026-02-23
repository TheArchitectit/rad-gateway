package e2e

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

var testConfig struct {
	GatewayURL string
	SPIREAddr  string
	KafkaAddr  string
}

func TestMain(m *testing.M) {
	// Load configuration from environment
	testConfig.GatewayURL = getEnv("A2A_GATEWAY_URL", "http://localhost:8090")
	testConfig.SPIREAddr = getEnv("SPIRE_SERVER_ADDR", "localhost:8081")
	testConfig.KafkaAddr = getEnv("KAFKA_ADDR", "localhost:9092")

	// Run tests
	code := m.Run()
	os.Exit(code)
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// createTestAgentIdentity creates a test agent with SPIFFE identity
// In production, this would integrate with SPIRE API
func createTestAgentIdentity(name string) (string, error) {
	return fmt.Sprintf("spiffe://internal.corp/agent/%s", name), nil
}

// cleanupAgentIdentity removes the test agent identity
func cleanupAgentIdentity(agentID string) error {
	// Mock implementation
	return nil
}

// createLowTrustAgent creates an agent with trust score below threshold
func createLowTrustAgent(name string) (string, error) {
	// In production, this would set trust score in the policy system
	return fmt.Sprintf("spiffe://internal.corp/agent/%s", name), nil
}

// waitForGateway waits for the gateway to be ready
func waitForGateway(url string, timeout int) error {
	client := &http.Client{}
	for i := 0; i < timeout; i++ {
		resp, err := client.Get(url + "/health")
		if err == nil && resp.StatusCode == 200 {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("gateway not ready after %d seconds", timeout)
}
