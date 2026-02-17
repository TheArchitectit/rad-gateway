// Package secrets provides Infisical integration for RAD Gateway.
// All secrets should be fetched through this package.
package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"radgateway/internal/logger"
)

// Client provides Infisical API access
type Client struct {
	apiURL    string
	token     string
	httpClient *http.Client
}

// Config holds Infisical configuration
type Config struct {
	APIURL        string
	Token         string
	WorkspaceID   string
	Environment   string // dev, staging, production
}

// LoadConfig loads Infisical config from environment
func LoadConfig() Config {
	return Config{
		APIURL:      getEnv("INFISICAL_API_URL", "http://172.16.30.45:8080/api"),
		Token:       getEnv("INFISICAL_TOKEN", ""),
		WorkspaceID: getEnv("INFISICAL_WORKSPACE_ID", ""),
		Environment: getEnv("INFISICAL_ENVIRONMENT", "dev"),
	}
}

// NewClient creates a new Infisical client
func NewClient(cfg Config) (*Client, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("INFISICAL_TOKEN is required")
	}

	return &Client{
		apiURL:     cfg.APIURL,
		token:      cfg.Token,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// SecretResponse represents a secret from Infisical
type SecretResponse struct {
	Secrets []Secret `json:"secrets"`
}

type Secret struct {
	ID        string `json:"id"`
	Key       string `json:"secretKey"`
	Value     string `json:"secretValue"`
	Comment   string `json:"comment"`
	CreatedAt string `json:"createdAt"`
}

// GetSecret fetches a single secret by key
func (c *Client) GetSecret(ctx context.Context, key string) (string, error) {
	log := logger.WithComponent("secrets")

	url := fmt.Sprintf("%s/v3/secrets?workspaceId=%s&environment=%s&secretPath=/rad-gateway&secretKey=%s",
		c.apiURL, c.WorkspaceID(), c.Environment(), key)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error("Failed to fetch secret from Infisical", "error", err.Error(), "key", key)
		return "", fmt.Errorf("fetching secret: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("infisical API error %d: %s", resp.StatusCode, string(body))
	}

	var result SecretResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	if len(result.Secrets) == 0 {
		return "", fmt.Errorf("secret not found: %s", key)
	}

	log.Info("Fetched secret from Infisical", "key", key)
	return result.Secrets[0].Value, nil
}

// GetProviderKey fetches provider API key
func (c *Client) GetProviderKey(ctx context.Context, provider string) (string, error) {
	key := fmt.Sprintf("%s_api_key", provider)
	return c.GetSecret(ctx, key)
}

// GetDatabaseURL fetches database connection string
func (c *Client) GetDatabaseURL(ctx context.Context) (string, error) {
	return c.GetSecret(ctx, "database_url")
}

// GetJWTSecret fetches JWT signing secret
func (c *Client) GetJWTSecret(ctx context.Context) (string, error) {
	return c.GetSecret(ctx, "jwt_secret")
}

// Helper methods
func (c *Client) WorkspaceID() string {
	return os.Getenv("INFISICAL_WORKSPACE_ID")
}

func (c *Client) Environment() string {
	env := os.Getenv("INFISICAL_ENVIRONMENT")
	if env == "" {
		return "dev"
	}
	return env
}

// Health checks if Infisical is reachable
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.apiURL+"/status", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("infisical health check failed: %d", resp.StatusCode)
	}

	return nil
}

// MockClient for testing
type MockClient struct {
	Secrets map[string]string
}

func (m *MockClient) GetSecret(_ context.Context, key string) (string, error) {
	if val, ok := m.Secrets[key]; ok {
		return val, nil
	}
	return "", fmt.Errorf("secret not found: %s", key)
}

// Helper function
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
