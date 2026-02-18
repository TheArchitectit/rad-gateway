// Package auth provides JWT authentication functionality.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenPair holds access and refresh tokens.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Claims represents JWT claims.
type Claims struct {
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	WorkspaceID string   `json:"workspace_id"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	AccessTokenSecret  []byte
	RefreshTokenSecret []byte
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	Issuer             string
}

// Default token expiry durations.
const (
	DefaultAccessTokenExpiry  = 15 * time.Minute
	DefaultRefreshTokenExpiry = 7 * 24 * time.Hour // 7 days
	MinimumSecretLength       = 32
)

// DefaultConfig returns a default JWT configuration.
// In production, JWT_ACCESS_SECRET and JWT_REFRESH_SECRET must be set.
func DefaultConfig() JWTConfig {
	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")

	// Warn if using generated secrets (tokens won't persist across restarts)
	if accessSecret == "" {
		fmt.Fprintf(os.Stderr, "[SECURITY WARNING] JWT_ACCESS_SECRET not set. Using generated secret - tokens will not persist across restarts.\n")
		accessSecret = generateSecret()
	}
	if refreshSecret == "" {
		fmt.Fprintf(os.Stderr, "[SECURITY WARNING] JWT_REFRESH_SECRET not set. Using generated secret - tokens will not persist across restarts.\n")
		refreshSecret = generateSecret()
	}

	// Validate minimum secret length
	if len(accessSecret) < MinimumSecretLength {
		fmt.Fprintf(os.Stderr, "[SECURITY WARNING] JWT_ACCESS_SECRET should be at least %d characters for security\n", MinimumSecretLength)
	}
	if len(refreshSecret) < MinimumSecretLength {
		fmt.Fprintf(os.Stderr, "[SECURITY WARNING] JWT_REFRESH_SECRET should be at least %d characters for security\n", MinimumSecretLength)
	}

	return JWTConfig{
		AccessTokenSecret:  []byte(accessSecret),
		RefreshTokenSecret: []byte(refreshSecret),
		AccessTokenExpiry:  DefaultAccessTokenExpiry,
		RefreshTokenExpiry: DefaultRefreshTokenExpiry,
		Issuer:             "rad-gateway",
	}
}

// LoadConfig loads JWT configuration from environment with strict validation.
// Returns an error if required secrets are not configured.
func LoadConfig() (JWTConfig, error) {
	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	if accessSecret == "" {
		return JWTConfig{}, fmt.Errorf("JWT_ACCESS_SECRET environment variable is required")
	}

	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if refreshSecret == "" {
		return JWTConfig{}, fmt.Errorf("JWT_REFRESH_SECRET environment variable is required")
	}

	if len(accessSecret) < 32 {
		return JWTConfig{}, fmt.Errorf("JWT_ACCESS_SECRET must be at least 32 characters")
	}

	if len(refreshSecret) < 32 {
		return JWTConfig{}, fmt.Errorf("JWT_REFRESH_SECRET must be at least 32 characters")
	}

	return JWTConfig{
		AccessTokenSecret:  []byte(accessSecret),
		RefreshTokenSecret: []byte(refreshSecret),
		AccessTokenExpiry:  DefaultAccessTokenExpiry,
		RefreshTokenExpiry: DefaultRefreshTokenExpiry,
		Issuer:             "rad-gateway",
	}, nil
}

// JWTManager handles JWT operations.
type JWTManager struct {
	config JWTConfig
}

// NewJWTManager creates a new JWT manager.
func NewJWTManager(config JWTConfig) *JWTManager {
	return &JWTManager{config: config}
}

// GenerateTokenPair creates a new access and refresh token pair.
func (m *JWTManager) GenerateTokenPair(userID, email, role, workspaceID string, permissions []string) (*TokenPair, error) {
	now := time.Now()

	// Generate access token
	accessClaims := Claims{
		UserID:      userID,
		Email:       email,
		Role:        role,
		WorkspaceID: workspaceID,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.config.AccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.config.Issuer,
			Subject:   userID,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(m.config.AccessTokenSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshToken,
		ExpiresAt:    now.Add(m.config.AccessTokenExpiry),
	}, nil
}

// ValidateAccessToken validates and parses an access token.
func (m *JWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.config.AccessTokenSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// RefreshTokenStore defines the interface for refresh token storage.
type RefreshTokenStore interface {
	Store(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	Validate(ctx context.Context, userID, tokenHash string) (bool, error)
	Revoke(ctx context.Context, userID, tokenHash string) error
	RevokeAll(ctx context.Context, userID string) error
}

// HashToken creates a SHA-256 hash of a token for storage.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(hash[:])
}

// getenv retrieves an environment variable or returns a default value.
func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

// generateSecret generates a random secret.
func generateSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// generateRefreshToken generates a random refresh token.
func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
