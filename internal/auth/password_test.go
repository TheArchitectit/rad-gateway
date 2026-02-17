// Package auth provides password hashing functionality.
package auth

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHasher_Hash(t *testing.T) {
	hasher := DefaultPasswordHasher()

	t.Run("hashes password successfully", func(t *testing.T) {
		hash, err := hasher.Hash("securepassword123")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if hash == "" {
			t.Error("expected hash to be non-empty")
		}

		if hash == "securepassword123" {
			t.Error("hash should not equal original password")
		}
	})

	t.Run("returns error for empty password", func(t *testing.T) {
		_, err := hasher.Hash("")
		if err == nil {
			t.Error("expected error for empty password")
		}
	})

	t.Run("produces different hashes for same password", func(t *testing.T) {
		hash1, _ := hasher.Hash("password123")
		hash2, _ := hasher.Hash("password123")

		// bcrypt includes a salt, so hashes should be different
		if hash1 == hash2 {
			t.Error("expected different hashes due to salt")
		}
	})
}

func TestPasswordHasher_Verify(t *testing.T) {
	hasher := DefaultPasswordHasher()

	t.Run("verifies correct password", func(t *testing.T) {
		hash, _ := hasher.Hash("securepassword123")

		if !hasher.Verify("securepassword123", hash) {
			t.Error("expected password to verify successfully")
		}
	})

	t.Run("rejects incorrect password", func(t *testing.T) {
		hash, _ := hasher.Hash("securepassword123")

		if hasher.Verify("wrongpassword", hash) {
			t.Error("expected incorrect password to fail verification")
		}
	})

	t.Run("rejects empty password", func(t *testing.T) {
		hash, _ := hasher.Hash("securepassword123")

		if hasher.Verify("", hash) {
			t.Error("expected empty password to fail verification")
		}
	})

	t.Run("rejects empty hash", func(t *testing.T) {
		if hasher.Verify("password123", "") {
			t.Error("expected verification to fail with empty hash")
		}
	})

	t.Run("handles malformed hash gracefully", func(t *testing.T) {
		if hasher.Verify("password123", "not-a-valid-hash") {
			t.Error("expected verification to fail with malformed hash")
		}
	})
}

func TestNewPasswordHasher(t *testing.T) {
	t.Run("uses default cost for invalid cost", func(t *testing.T) {
		// Cost below minimum
		hasher := NewPasswordHasher(3)
		if hasher.cost != bcrypt.DefaultCost {
			t.Errorf("expected cost to be %d, got: %d", bcrypt.DefaultCost, hasher.cost)
		}

		// Cost above maximum
		hasher = NewPasswordHasher(32)
		if hasher.cost != bcrypt.DefaultCost {
			t.Errorf("expected cost to be %d, got: %d", bcrypt.DefaultCost, hasher.cost)
		}
	})

	t.Run("uses specified cost for valid cost", func(t *testing.T) {
		hasher := NewPasswordHasher(12)
		if hasher.cost != 12 {
			t.Errorf("expected cost to be 12, got: %d", hasher.cost)
		}
	})
}

func TestIsValidHash(t *testing.T) {
	t.Run("recognizes valid bcrypt hashes", func(t *testing.T) {
		hasher := DefaultPasswordHasher()
		hash, _ := hasher.Hash("password")

		if !IsValidHash(hash) {
			t.Error("expected valid hash to be recognized")
		}
	})

	t.Run("rejects invalid hashes", func(t *testing.T) {
		invalidHashes := []string{
			"",
			"not-a-hash",
			"$1$rounds=1000$salt$hash", // Wrong algorithm
			"$2c$10$salt$hash", // Wrong version
			"plaintext",
			"sha256:abc123",
		}

		for _, hash := range invalidHashes {
			if IsValidHash(hash) {
				t.Errorf("expected '%s' to be rejected as invalid hash", hash)
			}
		}
	})
}

func TestPasswordHasher_Cost(t *testing.T) {
	t.Run("lower cost is faster", func(t *testing.T) {
		// This is a relative test - lower cost should generally be faster
		// We just verify both work without timing issues in CI
		fastHasher := NewPasswordHasher(bcrypt.MinCost)
		slowHasher := NewPasswordHasher(bcrypt.DefaultCost)

		_, err1 := fastHasher.Hash("test")
		_, err2 := slowHasher.Hash("test")

		if err1 != nil || err2 != nil {
			t.Error("expected both hash operations to succeed")
		}
	})
}

func BenchmarkPasswordHash(b *testing.B) {
	hasher := DefaultPasswordHasher()

	b.Run("Hash", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			hasher.Hash("benchmarkpassword123")
		}
	})

	b.Run("Verify", func(b *testing.B) {
		hash, _ := hasher.Hash("benchmarkpassword123")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			hasher.Verify("benchmarkpassword123", hash)
		}
	})
}

// Fuzz test for robustness
func FuzzPasswordHasher_Hash(f *testing.F) {
	// Add seed corpus
	f.Add("password")
	f.Add("123456")
	f.Add(strings.Repeat("a", 72)) // bcrypt max length

	f.Fuzz(func(t *testing.T, password string) {
		hasher := DefaultPasswordHasher()

		hash, err := hasher.Hash(password)
		if err != nil {
			// Empty password is expected to fail
			if password == "" {
				return
			}
			t.Errorf("unexpected error: %v", err)
		}

		// Verify should succeed for the same password
		if !hasher.Verify(password, hash) {
			t.Error("verification failed for hashed password")
		}
	})
}
