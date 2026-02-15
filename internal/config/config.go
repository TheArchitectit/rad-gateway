package config

import (
	"os"
	"strconv"
	"strings"
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
	addr := getenv("RAD_LISTEN_ADDR", ":8090")
	retryBudget := getenvInt("RAD_RETRY_BUDGET", 2)

	return Config{
		ListenAddr:  addr,
		APIKeys:     loadKeys(),
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

func loadKeys() map[string]string {
	raw := strings.TrimSpace(os.Getenv("RAD_API_KEYS"))
	if raw == "" {
		return map[string]string{}
	}
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
