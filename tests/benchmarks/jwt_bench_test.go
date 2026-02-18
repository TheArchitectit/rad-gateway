// Package benchmarks provides performance benchmarks for critical paths.
package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"radgateway/internal/auth"
)

// BenchmarkJWTGenerateTokenPair benchmarks token pair generation.
func BenchmarkJWTGenerateTokenPair(b *testing.B) {
	config := auth.JWTConfig{
		AccessTokenSecret:  []byte("benchmark-access-secret-32-bytes-long"),
		RefreshTokenSecret: []byte("benchmark-refresh-secret-32-bytes-long"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "rad-gateway-benchmark",
	}

	manager := auth.NewJWTManager(config)
	ctx := context.Background()

	permissions := []string{"read", "write", "admin"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("user-%d", i)
		_, err := manager.GenerateTokenPair(userID, "benchmark@example.com", "admin", "workspace-1", permissions)
		if err != nil {
			b.Fatalf("Failed to generate token pair: %v", err)
		}
	}
	_ = ctx // suppress unused warning
}

// BenchmarkJWTValidateToken benchmarks token validation.
func BenchmarkJWTValidateToken(b *testing.B) {
	config := auth.JWTConfig{
		AccessTokenSecret:  []byte("benchmark-access-secret-32-bytes-long"),
		RefreshTokenSecret: []byte("benchmark-refresh-secret-32-bytes-long"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "rad-gateway-benchmark",
	}

	manager := auth.NewJWTManager(config)

	// Generate a token for validation
	tokens, err := manager.GenerateTokenPair("user-1", "benchmark@example.com", "admin", "workspace-1", []string{"read", "write"})
	if err != nil {
		b.Fatalf("Failed to generate token: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.ValidateAccessToken(tokens.AccessToken)
		if err != nil {
			b.Fatalf("Failed to validate token: %v", err)
		}
	}
}

// BenchmarkJWTSign benchmarks raw JWT signing.
func BenchmarkJWTSign(b *testing.B) {
	secret := []byte("benchmark-access-secret-32-bytes-long")

	now := time.Now()
	claims := auth.Claims{
		UserID:      "user-benchmark",
		Email:       "benchmark@example.com",
		Role:        "admin",
		WorkspaceID: "workspace-1",
		Permissions: []string{"read", "write", "admin"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "rad-gateway-benchmark",
			Subject:   "user-benchmark",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		_, err := token.SignedString(secret)
		if err != nil {
			b.Fatalf("Failed to sign token: %v", err)
		}
	}
}

// BenchmarkJWTParse benchmarks JWT parsing without validation.
func BenchmarkJWTParse(b *testing.B) {
	secret := []byte("benchmark-access-secret-32-bytes-long")

	now := time.Now()
	claims := auth.Claims{
		UserID:      "user-benchmark",
		Email:       "benchmark@example.com",
		Role:        "admin",
		WorkspaceID: "workspace-1",
		Permissions: []string{"read", "write", "admin"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "rad-gateway-benchmark",
			Subject:   "user-benchmark",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		b.Fatalf("Failed to generate token: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := new(jwt.Parser).ParseUnverified(tokenString, &auth.Claims{})
		if err != nil {
			b.Fatalf("Failed to parse token: %v", err)
		}
	}
}

// BenchmarkJWTFullCycle benchmarks complete sign and validate cycle.
func BenchmarkJWTFullCycle(b *testing.B) {
	config := auth.JWTConfig{
		AccessTokenSecret:  []byte("benchmark-access-secret-32-bytes-long"),
		RefreshTokenSecret: []byte("benchmark-refresh-secret-32-bytes-long"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "rad-gateway-benchmark",
	}

	manager := auth.NewJWTManager(config)
	permissions := []string{"read", "write", "admin"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("user-%d", i)
		tokens, err := manager.GenerateTokenPair(userID, "benchmark@example.com", "admin", "workspace-1", permissions)
		if err != nil {
			b.Fatalf("Failed to generate token: %v", err)
		}

		_, err = manager.ValidateAccessToken(tokens.AccessToken)
		if err != nil {
			b.Fatalf("Failed to validate token: %v", err)
		}
	}
}

// BenchmarkJWTParallel benchmarks concurrent token operations.
func BenchmarkJWTParallel(b *testing.B) {
	config := auth.JWTConfig{
		AccessTokenSecret:  []byte("benchmark-access-secret-32-bytes-long"),
		RefreshTokenSecret: []byte("benchmark-refresh-secret-32-bytes-long"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "rad-gateway-benchmark",
	}

	manager := auth.NewJWTManager(config)
	permissions := []string{"read", "write"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			userID := fmt.Sprintf("user-%d", i)
			tokens, err := manager.GenerateTokenPair(userID, "benchmark@example.com", "admin", "workspace-1", permissions)
			if err != nil {
				b.Fatalf("Failed to generate token: %v", err)
			}

			_, err = manager.ValidateAccessToken(tokens.AccessToken)
			if err != nil {
				b.Fatalf("Failed to validate token: %v", err)
			}
			i++
		}
	})
}

// BenchmarkJWTThroughput benchmarks token throughput.
func BenchmarkJWTThroughput(b *testing.B) {
	config := auth.JWTConfig{
		AccessTokenSecret:  []byte("benchmark-access-secret-32-bytes-long"),
		RefreshTokenSecret: []byte("benchmark-refresh-secret-32-bytes-long"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "rad-gateway-benchmark",
	}

	manager := auth.NewJWTManager(config)

	// Generate tokens first
	tokens := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		tp, _ := manager.GenerateTokenPair(
			fmt.Sprintf("user-%d", i),
			"benchmark@example.com",
			"admin",
			"workspace-1",
			[]string{"read", "write"},
		)
		tokens[i] = tp.AccessToken
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.ValidateAccessToken(tokens[i%1000])
		if err != nil {
			b.Fatalf("Failed to validate token: %v", err)
		}
	}
}

// BenchmarkJWTClaimsCreation benchmarks claims struct creation.
func BenchmarkJWTClaimsCreation(b *testing.B) {
	now := time.Now()
	permissions := []string{"read", "write", "admin"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = auth.Claims{
			UserID:      fmt.Sprintf("user-%d", i),
			Email:       "benchmark@example.com",
			Role:        "admin",
			WorkspaceID: "workspace-1",
			Permissions: permissions,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(now),
				NotBefore: jwt.NewNumericDate(now),
				Issuer:    "rad-gateway-benchmark",
				Subject:   fmt.Sprintf("user-%d", i),
			},
		}
	}
}

// BenchmarkJWTTokenHashing benchmarks token hashing.
func BenchmarkJWTTokenHashing(b *testing.B) {
	config := auth.JWTConfig{
		AccessTokenSecret:  []byte("benchmark-access-secret-32-bytes-long"),
		RefreshTokenSecret: []byte("benchmark-refresh-secret-32-bytes-long"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "rad-gateway-benchmark",
	}

	manager := auth.NewJWTManager(config)
	tokens, _ := manager.GenerateTokenPair("user-1", "benchmark@example.com", "admin", "workspace-1", []string{"read"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = auth.HashToken(tokens.RefreshToken)
	}
}

// BenchmarkJWTSigningMethods compares different signing methods.
func BenchmarkJWTSigningMethods(b *testing.B) {
	secret := []byte("benchmark-access-secret-32-bytes-long")
	now := time.Now()

	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(now),
		Subject:   "user-benchmark",
	}

	b.Run("HS256", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			_, err := token.SignedString(secret)
			if err != nil {
				b.Fatalf("Failed to sign token: %v", err)
			}
		}
	})

	b.Run("HS384", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
			_, err := token.SignedString(secret)
			if err != nil {
				b.Fatalf("Failed to sign token: %v", err)
			}
		}
	})

	b.Run("HS512", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
			_, err := token.SignedString(secret)
			if err != nil {
				b.Fatalf("Failed to sign token: %v", err)
			}
		}
	})
}

// BenchmarkJWTTokenSizeComparison benchmarks tokens of different sizes.
func BenchmarkJWTTokenSizeComparison(b *testing.B) {
	secret := []byte("benchmark-access-secret-32-bytes-long")
	now := time.Now()

	b.Run("SmallClaims", func(b *testing.B) {
		claims := jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			Subject:   "user-1",
		}

		for i := 0; i < b.N; i++ {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			_, err := token.SignedString(secret)
			if err != nil {
				b.Fatalf("Failed to sign token: %v", err)
			}
		}
	})

	b.Run("MediumClaims", func(b *testing.B) {
		claims := auth.Claims{
			UserID:      "user-1",
			Email:       "test@example.com",
			Role:        "user",
			WorkspaceID: "workspace-1",
			Permissions: []string{"read", "write"},
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
				Subject:   "user-1",
			},
		}

		for i := 0; i < b.N; i++ {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			_, err := token.SignedString(secret)
			if err != nil {
				b.Fatalf("Failed to sign token: %v", err)
			}
		}
	})

	b.Run("LargeClaims", func(b *testing.B) {
		// Create many permissions
		perms := make([]string, 100)
		for i := range perms {
			perms[i] = fmt.Sprintf("permission-%d", i)
		}

		claims := auth.Claims{
			UserID:      "user-1",
			Email:       "test@example.com",
			Role:        "admin",
			WorkspaceID: "workspace-1",
			Permissions: perms,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
				Subject:   "user-1",
			},
		}

		for i := 0; i < b.N; i++ {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			_, err := token.SignedString(secret)
			if err != nil {
				b.Fatalf("Failed to sign token: %v", err)
			}
		}
	})
}

// BenchmarkJWTManagerWithConfig benchmarks manager operations with different configurations.
func BenchmarkJWTManagerWithConfig(b *testing.B) {
	b.Run("ShortExpiry", func(b *testing.B) {
		config := auth.JWTConfig{
			AccessTokenSecret:  []byte("benchmark-access-secret-32-bytes-long"),
			RefreshTokenSecret: []byte("benchmark-refresh-secret-32-bytes-long"),
			AccessTokenExpiry:  1 * time.Minute,
			RefreshTokenExpiry: 1 * time.Hour,
			Issuer:             "rad-gateway-benchmark",
		}

		manager := auth.NewJWTManager(config)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := manager.GenerateTokenPair("user-1", "test@example.com", "user", "workspace-1", []string{"read"})
			if err != nil {
				b.Fatalf("Failed to generate token: %v", err)
			}
		}
	})

	b.Run("LongExpiry", func(b *testing.B) {
		config := auth.JWTConfig{
			AccessTokenSecret:  []byte("benchmark-access-secret-32-bytes-long"),
			RefreshTokenSecret: []byte("benchmark-refresh-secret-32-bytes-long"),
			AccessTokenExpiry:  24 * time.Hour,
			RefreshTokenExpiry:  30 * 24 * time.Hour,
			Issuer:             "rad-gateway-benchmark",
		}

		manager := auth.NewJWTManager(config)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := manager.GenerateTokenPair("user-1", "test@example.com", "user", "workspace-1", []string{"read"})
			if err != nil {
				b.Fatalf("Failed to generate token: %v", err)
			}
		}
	})
}

// BenchmarkJWTValidateExpiredToken benchmarks validating expired tokens.
func BenchmarkJWTValidateExpiredToken(b *testing.B) {
	// Create a token that's already expired
	now := time.Now()
	expiredClaims := auth.Claims{
		UserID:      "user-1",
		Email:       "test@example.com",
		Role:        "user",
		WorkspaceID: "workspace-1",
		Permissions: []string{"read"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)), // Already expired
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			Subject:   "user-1",
		},
	}

	secret := []byte("benchmark-access-secret-32-bytes-long")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	tokenString, _ := token.SignedString(secret)

	config := auth.JWTConfig{
		AccessTokenSecret:  secret,
		RefreshTokenSecret: []byte("benchmark-refresh-secret-32-bytes-long"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "rad-gateway-benchmark",
	}

	manager := auth.NewJWTManager(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Validation should fail for expired token
		_, _ = manager.ValidateAccessToken(tokenString)
	}
}

// BenchmarkJWTValidateInvalidSignature benchmarks validating tokens with wrong signatures.
func BenchmarkJWTValidateInvalidSignature(b *testing.B) {
	config := auth.JWTConfig{
		AccessTokenSecret:  []byte("correct-secret-32-bytes-long!!!"),
		RefreshTokenSecret: []byte("benchmark-refresh-secret-32-bytes-long"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "rad-gateway-benchmark",
	}

	manager := auth.NewJWTManager(config)

	// Generate token with wrong secret
	wrongConfig := auth.JWTConfig{
		AccessTokenSecret:  []byte("wrong-secret-32-bytes-long!!!!"),
		RefreshTokenSecret: []byte("benchmark-refresh-secret-32-bytes-long"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "rad-gateway-benchmark",
	}
	wrongManager := auth.NewJWTManager(wrongConfig)

	tokens, _ := wrongManager.GenerateTokenPair("user-1", "test@example.com", "user", "workspace-1", []string{"read"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Validation should fail for invalid signature
		_, _ = manager.ValidateAccessToken(tokens.AccessToken)
	}
}

// BenchmarkJWTMemoryAlloc benchmarks memory allocations during token operations.
func BenchmarkJWTMemoryAlloc(b *testing.B) {
	config := auth.JWTConfig{
		AccessTokenSecret:  []byte("benchmark-access-secret-32-bytes-long"),
		RefreshTokenSecret: []byte("benchmark-refresh-secret-32-bytes-long"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "rad-gateway-benchmark",
	}

	manager := auth.NewJWTManager(config)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokens, err := manager.GenerateTokenPair(
			fmt.Sprintf("user-%d", i),
			"benchmark@example.com",
			"admin",
			"workspace-1",
			[]string{"read", "write", "admin"},
		)
		if err != nil {
			b.Fatalf("Failed to generate token: %v", err)
		}

		_, err = manager.ValidateAccessToken(tokens.AccessToken)
		if err != nil {
			b.Fatalf("Failed to validate token: %v", err)
		}
	}
}
