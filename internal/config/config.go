package config

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"radgateway/internal/logger"
	"radgateway/internal/secrets"
)

type Candidate struct {
	Provider string
	Model    string
	Weight   int
}

type Config struct {
	ListenAddr  string
	APIKeys     map[string]string
	ModelRoutes map[string][]Candidate
	RetryBudget int
}

func Load() Config {
	log := logger.WithComponent("config")

	// Initialize Infisical client if token available
	infisicalCfg := secrets.LoadConfig()
	var secretClient *secrets.Client
	var err error

	if infisicalCfg.Token != "" {
		secretClient, err = secrets.NewClient(infisicalCfg)
		if err != nil {
			log.Warn("Failed to initialize Infisical client, falling back to env vars", "error", err)
		} else {
			// Test connectivity
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := secretClient.Health(ctx); err != nil {
				log.Warn("Infisical health check failed, falling back to env vars", "error", err)
				secretClient = nil
			} else {
				log.Info("Connected to Infisical for secrets management")
			}
		}
	}

	addr := getenv("RAD_LISTEN_ADDR", ":8090")
	retryBudget := getenvInt("RAD_RETRY_BUDGET", 2)

	// Try to load API keys from Infisical if available
	apiKeys := loadKeys(secretClient)

	return Config{
		ListenAddr:  addr,
		APIKeys:     apiKeys,
		ModelRoutes: loadModelRoutes(),
		RetryBudget: retryBudget,
	}
}

func (c Config) Snapshot() map[string]any {
	return map[string]any{
		"listenAddr":     c.ListenAddr,
		"retryBudget":    c.RetryBudget,
		"keysConfigured": len(c.APIKeys),
		"models":         c.ModelRoutes,
	}
}

func loadKeys(client *secrets.Client) map[string]string {
	log := logger.WithComponent("config")

	// If Infisical client is available, try to fetch keys from there
	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		keys, err := client.GetSecret(ctx, "api_keys")
		if err == nil && keys != "" {
			log.Info("Loaded API keys from Infisical")
			return parseKeys(keys)
		}
		log.Warn("Failed to load API keys from Infisical, falling back to env vars", "error", err)
	}

	// Fall back to environment variable
	raw := strings.TrimSpace(os.Getenv("RAD_API_KEYS"))
	if raw == "" {
		return map[string]string{}
	}
	return parseKeys(raw)
}

// parseKeys parses comma-separated key:value pairs
func parseKeys(raw string) map[string]string {
	out := map[string]string{}
	parts := strings.Split(raw, ",")
	for _, item := range parts {
		kv := strings.SplitN(strings.TrimSpace(item), ":", 2)
		if len(kv) != 2 {
			continue
		}
		name := strings.TrimSpace(kv[0])
		secret := strings.TrimSpace(kv[1])
		if name != "" && secret != "" {
			out[name] = secret
		}
	}
	return out
}

func loadModelRoutes() map[string][]Candidate {
	return map[string][]Candidate{
		"gpt-4o-mini": {
			{Provider: "mock", Model: "gpt-4o-mini", Weight: 80},
			{Provider: "mock", Model: "fallback-mini", Weight: 20},
		},
		"claude-3-5-sonnet": {
			{Provider: "mock", Model: "claude-3-5-sonnet", Weight: 100},
		},
	}
}

func getenv(k, fallback string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return fallback
	}
	return v
}

func getenvInt(k string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(k))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return fallback
	}
	return v
}
