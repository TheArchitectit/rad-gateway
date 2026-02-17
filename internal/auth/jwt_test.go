// Package auth provides JWT authentication functionality.
package auth

import (
	"testing"
	"time"
)

func TestJWTManager_GenerateTokenPair(t *testing.T) {
	config := JWTConfig{
		AccessTokenSecret:  []byte("test-access-secret-32-bytes-long"),
		RefreshTokenSecret: []byte("test-refresh-secret-32-bytes-long"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:           "test",
	}

	manager := NewJWTManager(config)

	t.Run("generates valid token pair", func(t *testing.T) {
		tokens, err := manager.GenerateTokenPair(
			"user-123",
			"test@example.com",
			"admin",
			"workspace-456",
			[]string{"read", "write", "admin"},
		)

		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if tokens.AccessToken == "" {
			t.Error("expected access token to be non-empty")
		}

		if tokens.RefreshToken == "" {
			t.Error("expected refresh token to be non-empty")
		}

		if tokens.ExpiresAt.IsZero() {
			t.Error("expected expires_at to be set")
		}
	})

	t.Run("tokens contain expected claims", func(t *testing.T) {
		tokens, _ := manager.GenerateTokenPair(
			"user-123",
			"test@example.com",
			"admin",
			"workspace-456",
			[]string{"read", "write"},
		)

		claims, err := manager.ValidateAccessToken(tokens.AccessToken)
		if err != nil {
			t.Fatalf("failed to validate token: %v", err)
		}

		if claims.UserID != "user-123" {
			t.Errorf("expected user_id to be 'user-123', got: %s", claims.UserID)
		}

		if claims.Email != "test@example.com" {
			t.Errorf("expected email to be 'test@example.com', got: %s", claims.Email)
		}

		if claims.Role != "admin" {
			t.Errorf("expected role to be 'admin', got: %s", claims.Role)
		}

		if claims.WorkspaceID != "workspace-456" {
			t.Errorf("expected workspace_id to be 'workspace-456', got: %s", claims.WorkspaceID)
		}
	})
}

func TestJWTManager_ValidateAccessToken(t *testing.T) {
	config := JWTConfig{
		AccessTokenSecret:  []byte("test-access-secret-32-bytes-long"),
		RefreshTokenSecret: []byte("test-refresh-secret-32-bytes-long"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:           "test",
	}

	manager := NewJWTManager(config)

	t.Run("validates correct token", func(t *testing.T) {
		tokens, _ := manager.GenerateTokenPair(
			"user-123",
			"test@example.com",
			"developer",
			"workspace-456",
			[]string{"read"},
		)

		claims, err := manager.ValidateAccessToken(tokens.AccessToken)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}

		if claims == nil {
			t.Error("expected claims to be non-nil")
		}
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		_, err := manager.ValidateAccessToken("invalid.token.here")
		if err == nil {
			t.Error("expected error for invalid token")
		}
	})

	t.Run("rejects token with wrong secret", func(t *testing.T) {
		// Generate token with one manager
		tokens, _ := manager.GenerateTokenPair(
			"user-123",
			"test@example.com",
			"developer",
			"workspace-456",
			[]string{"read"},
		)

		// Validate with different manager (different secret)
		otherConfig := JWTConfig{
			AccessTokenSecret:  []byte("different-secret-32-bytes-long"),
			RefreshTokenSecret: []byte("different-refresh-secret-32-bytes"),
			AccessTokenExpiry:  15 * time.Minute,
			RefreshTokenExpiry: 7 * 24 * time.Hour,
			Issuer:           "test",
		}
		otherManager := NewJWTManager(otherConfig)

		_, err := otherManager.ValidateAccessToken(tokens.AccessToken)
		if err == nil {
			t.Error("expected error when validating with wrong secret")
		}
	})

	t.Run("rejects expired token", func(t *testing.T) {
		shortConfig := JWTConfig{
			AccessTokenSecret:  []byte("test-access-secret-32-bytes-long"),
			RefreshTokenSecret: []byte("test-refresh-secret-32-bytes-long"),
			AccessTokenExpiry:  -1 * time.Second, // Already expired
			RefreshTokenExpiry: 7 * 24 * time.Hour,
			Issuer:           "test",
		}
		shortManager := NewJWTManager(shortConfig)

		tokens, _ := shortManager.GenerateTokenPair(
			"user-123",
			"test@example.com",
			"developer",
			"workspace-456",
			[]string{"read"},
		)

		_, err := manager.ValidateAccessToken(tokens.AccessToken)
		if err == nil {
			t.Error("expected error for expired token")
		}
	})
}

func TestHashToken(t *testing.T) {
	t.Run("produces consistent hashes", func(t *testing.T) {
		token := "test-token-123"
		hash1 := HashToken(token)
		hash2 := HashToken(token)

		if hash1 != hash2 {
			t.Error("expected same token to produce same hash")
		}
	})

	t.Run("produces different hashes for different tokens", func(t *testing.T) {
		hash1 := HashToken("token-1")
		hash2 := HashToken("token-2")

		if hash1 == hash2 {
			t.Error("expected different tokens to produce different hashes")
		}
	})
}
