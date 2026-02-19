// Package secrets provides centralized secret loading from Infisical.
package secrets

import (
	"context"
	"os"
	"strings"
)

// Loader provides centralized secret loading with Infisical fallback.
type Loader struct {
	client *Client
	ctx    context.Context
}

// NewLoader creates a new secrets loader.
func NewLoader() (*Loader, error) {
	cfg := LoadConfig()

	// If Infisical not configured, return loader with nil client
	if cfg.Token == "" {
		return &Loader{ctx: context.Background()}, nil
	}

	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Loader{
		client: client,
		ctx:    context.Background(),
	}, nil
}

// LoadDatabaseDSN loads database connection string.
// Priority: 1. Infisical secret, 2. RAD_DB_DSN env, 3. fallback
func (l *Loader) LoadDatabaseDSN(fallback string) string {
	if l.client != nil {
		value, err := l.client.GetSecret(l.ctx, "database_url")
		if err == nil && value != "" {
			return value
		}
	}

	if val := os.Getenv("RAD_DB_DSN"); val != "" {
		return val
	}

	return fallback
}

// LoadJWTAccessSecret loads JWT access token secret.
func (l *Loader) LoadJWTAccessSecret(fallback string) string {
	if l.client != nil {
		value, err := l.client.GetSecret(l.ctx, "jwt_access_secret")
		if err == nil && value != "" {
			return value
		}
	}

	if val := os.Getenv("JWT_ACCESS_SECRET"); val != "" {
		return val
	}

	return fallback
}

// LoadJWTRefreshSecret loads JWT refresh token secret.
func (l *Loader) LoadJWTRefreshSecret(fallback string) string {
	if l.client != nil {
		value, err := l.client.GetSecret(l.ctx, "jwt_refresh_secret")
		if err == nil && value != "" {
			return value
		}
	}

	if val := os.Getenv("JWT_REFRESH_SECRET"); val != "" {
		return val
	}

	return fallback
}

// LoadRedisPassword loads Redis password.
func (l *Loader) LoadRedisPassword() string {
	if l.client != nil {
		value, err := l.client.GetSecret(l.ctx, "redis_password")
		if err == nil && value != "" {
			return value
		}
	}

	return os.Getenv("RAD_REDIS_PASSWORD")
}

// LoadAPIKeys loads API keys from Infisical or env.
func (l *Loader) LoadAPIKeys() string {
	if l.client != nil {
		value, err := l.client.GetSecret(l.ctx, "api_keys")
		if err == nil && value != "" {
			return value
		}
	}

	return os.Getenv("RAD_API_KEYS")
}

// LoadProviderSecret loads a provider-specific secret.
func (l *Loader) LoadProviderSecret(provider, key string) string {
	secretKey := provider + "_" + key

	if l.client != nil {
		value, err := l.client.GetSecret(l.ctx, secretKey)
		if err == nil && value != "" {
			return value
		}
	}

	// Try environment variable
	envKey := strings.ToUpper(provider) + "_" + strings.ToUpper(key)
	return os.Getenv(envKey)
}

// Close closes the loader and its client.
func (l *Loader) Close() error {
	if l.client != nil {
		return l.client.Health(l.ctx) // Just check health, no close needed
	}
	return nil
}

// IsInfisicalEnabled returns true if Infisical is configured.
func (l *Loader) IsInfisicalEnabled() bool {
	return l.client != nil
}
