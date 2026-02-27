// Package auth provides JWT secret rotation functionality.
package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// KeyVersion represents a version of JWT signing keys
type KeyVersion struct {
	Version   int       `json:"version"`
	Secret    []byte    `json:"-"` // Never serialize the actual secret
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Active    bool      `json:"active"`
}

// IsValid checks if the key version is still valid (not expired)
func (kv *KeyVersion) IsValid() bool {
	now := time.Now()
	return kv.Active && now.After(kv.CreatedAt) && now.Before(kv.ExpiresAt)
}

// KeyRotator manages JWT secret rotation
type KeyRotator struct {
	mu       sync.RWMutex
	versions map[int]*KeyVersion
	current  int // Current active version number
	config   RotationConfig
}

// RotationConfig configures key rotation behavior
type RotationConfig struct {
	// RotationInterval is how often to rotate keys
	RotationInterval time.Duration

	// KeyLifetime is how long a key remains valid after creation
	KeyLifetime time.Duration

	// GracePeriod is how long old keys remain valid for verification after rotation
	GracePeriod time.Duration

	// MinSecretLength is the minimum length for secrets
	MinSecretLength int

	// MaxVersions keeps the last N versions for verification
	MaxVersions int
}

// DefaultRotationConfig returns default rotation configuration
func DefaultRotationConfig() RotationConfig {
	return RotationConfig{
		RotationInterval: 24 * time.Hour,    // Rotate daily
		KeyLifetime:      7 * 24 * time.Hour, // Keys valid for 7 days
		GracePeriod:      24 * time.Hour,     // Old keys valid for 24h after rotation
		MinSecretLength:  32,
		MaxVersions:      3, // Keep last 3 versions
	}
}

// NewKeyRotator creates a new key rotator
func NewKeyRotator(config RotationConfig) (*KeyRotator, error) {
	kr := &KeyRotator{
		versions: make(map[int]*KeyVersion),
		config:   config,
	}

	// Load initial keys from environment or generate
	if err := kr.loadKeys(); err != nil {
		return nil, err
	}

	return kr, nil
}

// loadKeys loads keys from environment variables or generates new ones
func (kr *KeyRotator) loadKeys() error {
	// Try to load from environment
	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")

	// Check for versioned secrets (JWT_ACCESS_SECRET_V1, JWT_ACCESS_SECRET_V2, etc.)
	version := 1
	for {
		vSecret := os.Getenv(fmt.Sprintf("JWT_ACCESS_SECRET_V%d", version))
		if vSecret == "" {
			break
		}
		// Add as a version (historical key for verification)
		if version > 1 {
			kr.versions[version] = &KeyVersion{
				Version:   version,
				Secret:    []byte(vSecret),
				CreatedAt: time.Now().Add(-time.Duration(version) * kr.config.RotationInterval),
				ExpiresAt: time.Now().Add(kr.config.GracePeriod),
				Active:    true,
			}
		}
		version++
	}

	// Use current secret as latest version
	if accessSecret == "" {
		accessSecret = generateSecret()
		fmt.Fprintf(os.Stderr, "[SECURITY WARNING] JWT_ACCESS_SECRET not set. Generated secret will not persist across restarts.\n")
	}

	if refreshSecret == "" {
		refreshSecret = generateSecret()
		fmt.Fprintf(os.Stderr, "[SECURITY WARNING] JWT_REFRESH_SECRET not set. Generated secret will not persist across restarts.\n")
	}

	// Validate minimum length
	if len(accessSecret) < kr.config.MinSecretLength {
		return fmt.Errorf("JWT_ACCESS_SECRET must be at least %d characters", kr.config.MinSecretLength)
	}

	// Create current version
	kr.current = version
	kr.versions[kr.current] = &KeyVersion{
		Version:   kr.current,
		Secret:    []byte(accessSecret),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(kr.config.KeyLifetime),
		Active:    true,
	}

	return nil
}

// GetCurrentKey returns the current active signing key
func (kr *KeyRotator) GetCurrentKey() *KeyVersion {
	kr.mu.RLock()
	defer kr.mu.RUnlock()

	return kr.versions[kr.current]
}

// GetVerificationKeys returns all valid keys for token verification
func (kr *KeyRotator) GetVerificationKeys() []*KeyVersion {
	kr.mu.RLock()
	defer kr.mu.RUnlock()

	var keys []*KeyVersion
	now := time.Now()

	for _, v := range kr.versions {
		if v.Active && now.Before(v.ExpiresAt) {
			keys = append(keys, v)
		}
	}

	return keys
}

// Rotate creates a new key version and marks old ones for expiration
func (kr *KeyRotator) Rotate() (*KeyVersion, error) {
	kr.mu.Lock()
	defer kr.mu.Unlock()

	// Generate new secret
	newSecret := generateSecret()
	newVersion := kr.current + 1

	// Mark old version as expiring (grace period)
	if old, exists := kr.versions[kr.current]; exists {
		old.ExpiresAt = time.Now().Add(kr.config.GracePeriod)
	}

	// Create new version
	newKey := &KeyVersion{
		Version:   newVersion,
		Secret:    []byte(newSecret),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(kr.config.KeyLifetime),
		Active:    true,
	}

	kr.versions[newVersion] = newKey
	kr.current = newVersion

	// Clean up old versions
	kr.cleanupOldVersions()

	return newKey, nil
}

// cleanupOldVersions removes versions beyond MaxVersions
func (kr *KeyRotator) cleanupOldVersions() {
	if len(kr.versions) <= kr.config.MaxVersions {
		return
	}

	// Find oldest versions to remove
	toRemove := len(kr.versions) - kr.config.MaxVersions
	for v := 1; v <= kr.current && toRemove > 0; v++ {
		if _, exists := kr.versions[v]; exists && v != kr.current {
			delete(kr.versions, v)
			toRemove--
		}
	}
}

// StartAutoRotation starts automatic key rotation in a goroutine
func (kr *KeyRotator) StartAutoRotation(ctx context.Context) {
	ticker := time.NewTicker(kr.config.RotationInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := kr.Rotate(); err != nil {
					fmt.Fprintf(os.Stderr, "[JWT Rotation Error] %v\n", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// RotationManager handles JWT configuration with rotation support
type RotationManager struct {
	rotator          *KeyRotator
	refreshSecret    []byte
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
	issuer           string
}

// NewRotationManager creates a JWT manager with key rotation support
func NewRotationManager(rotator *KeyRotator, refreshSecret string) *RotationManager {
	return &RotationManager{
		rotator:            rotator,
		refreshSecret:      []byte(refreshSecret),
		accessTokenExpiry:  15 * time.Minute,
		refreshTokenExpiry: 7 * 24 * time.Hour,
		issuer:             "rad-gateway",
	}
}

// GetAccessSecret returns the current access token secret
func (rm *RotationManager) GetAccessSecret() []byte {
	key := rm.rotator.GetCurrentKey()
	if key == nil {
		return nil
	}
	return key.Secret
}

// GetRefreshSecret returns the refresh token secret
func (rm *RotationManager) GetRefreshSecret() []byte {
	return rm.refreshSecret
}

// GetAllAccessSecrets returns all valid access secrets for verification
func (rm *RotationManager) GetAllAccessSecrets() [][]byte {
	keys := rm.rotator.GetVerificationKeys()
	var secrets [][]byte
	for _, k := range keys {
		secrets = append(secrets, k.Secret)
	}
	return secrets
}

// RotatingJWTManager extends JWTManager with rotation support
type RotatingJWTManager struct {
	base     *JWTManager
	rotator  *KeyRotator
	rManager *RotationManager
}

// NewRotatingJWTManager creates a JWT manager with rotation support
func NewRotatingJWTManager(baseConfig JWTConfig) (*RotatingJWTManager, error) {
	// Create key rotator
	rotator, err := NewKeyRotator(DefaultRotationConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create key rotator: %w", err)
	}

	// Create rotation manager
	rManager := NewRotationManager(rotator, string(baseConfig.RefreshTokenSecret))

	// Create base manager with current key
	currentKey := rotator.GetCurrentKey()
	if currentKey == nil {
		return nil, fmt.Errorf("no current key available")
	}

	config := JWTConfig{
		AccessTokenSecret:  currentKey.Secret,
		RefreshTokenSecret: baseConfig.RefreshTokenSecret,
		AccessTokenExpiry:  baseConfig.AccessTokenExpiry,
		RefreshTokenExpiry: baseConfig.RefreshTokenExpiry,
		Issuer:             baseConfig.Issuer,
	}

	return &RotatingJWTManager{
		base:     NewJWTManager(config),
		rotator:  rotator,
		rManager: rManager,
	}, nil
}

// GenerateTokenPair creates a new token pair using current key
func (rjm *RotatingJWTManager) GenerateTokenPair(userID, email, role, workspaceID string, permissions []string) (*TokenPair, error) {
	// Ensure we're using the latest key
	currentKey := rjm.rotator.GetCurrentKey()
	if currentKey == nil {
		return nil, fmt.Errorf("no active signing key")
	}

	// Update base manager with current key
	config := JWTConfig{
		AccessTokenSecret:  currentKey.Secret,
		RefreshTokenSecret: rjm.rManager.GetRefreshSecret(),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "rad-gateway",
	}
	rjm.base = NewJWTManager(config)

	return rjm.base.GenerateTokenPair(userID, email, role, workspaceID, permissions)
}

// ValidateAccessToken validates a token trying all valid keys
func (rjm *RotatingJWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	// Try all valid keys
	keys := rjm.rotator.GetVerificationKeys()

	var lastErr error
	for _, key := range keys {
		config := JWTConfig{
			AccessTokenSecret:  key.Secret,
			RefreshTokenSecret: rjm.rManager.GetRefreshSecret(),
		}
		manager := NewJWTManager(config)
		claims, err := manager.ValidateAccessToken(tokenString)
		if err == nil {
			return claims, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("invalid token")
}

// Rotate manually triggers key rotation
func (rjm *RotatingJWTManager) Rotate() error {
	_, err := rjm.rotator.Rotate()
	return err
}

// StartAutoRotation starts automatic rotation
func (rjm *RotatingJWTManager) StartAutoRotation(ctx context.Context) {
	rjm.rotator.StartAutoRotation(ctx)
}

// KeyStatus returns the current rotation status
func (rjm *RotatingJWTManager) KeyStatus() map[string]interface{} {
	current := rjm.rotator.GetCurrentKey()
	keys := rjm.rotator.GetVerificationKeys()

	var keyInfo []map[string]interface{}
	for _, k := range keys {
		keyInfo = append(keyInfo, map[string]interface{}{
			"version":    k.Version,
			"created_at": k.CreatedAt,
			"expires_at": k.ExpiresAt,
			"active":     k.Active,
			"valid":      k.IsValid(),
		})
	}

	return map[string]interface{}{
		"current_version": current.Version,
		"total_keys":      len(keys),
		"keys":            keyInfo,
	}
}

// HashString creates a hash for comparison (used in tests and validation)
func HashString(s string) string {
	hash := sha256.Sum256([]byte(s))
	return base64.URLEncoding.EncodeToString(hash[:])
}

// ParseRotationInterval parses a rotation interval from string
func ParseRotationInterval(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))

	// Common shortcuts
	switch s {
	case "hourly":
		return time.Hour, nil
	case "daily":
		return 24 * time.Hour, nil
	case "weekly":
		return 7 * 24 * time.Hour, nil
	case "monthly":
		return 30 * 24 * time.Hour, nil
	}

	// Parse as duration
	return time.ParseDuration(s)
}
