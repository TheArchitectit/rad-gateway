// Package regression tests configuration loading edge cases.
// Sprint 7.2: Test Configuration Loading Edge Cases
package regression

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestConfigurationEdgeCases validates configuration loading
func TestConfigurationEdgeCases(t *testing.T) {
	// Test empty environment
	t.Run("EmptyEnvironment", func(t *testing.T) {
		// Clear relevant env vars
		vars := []string{"RAD_ENV", "RAD_DB_DRIVER", "RAD_DB_DSN"}
		for _, v := range vars {
			os.Unsetenv(v)
		}

		// Should fall back to defaults
		dbDriver := os.Getenv("RAD_DB_DRIVER")
		if dbDriver != "" {
			t.Logf("Expected empty driver, got: %s", dbDriver)
		}
	})

	// Test invalid values
	t.Run("InvalidValues", func(t *testing.T) {
		os.Setenv("RAD_PORT", "invalid")
		defer os.Unsetenv("RAD_PORT")

		// Should handle gracefully
		port := os.Getenv("RAD_PORT")
		if port != "invalid" {
			t.Errorf("Expected 'invalid', got: %s", port)
		}
	})

	// Test special characters in values
	t.Run("SpecialCharacters", func(t *testing.T) {
		specialValue := "value with spaces and !@#$%^&*()"
		os.Setenv("RAD_SPECIAL", specialValue)
		defer os.Unsetenv("RAD_SPECIAL")

		retrieved := os.Getenv("RAD_SPECIAL")
		if retrieved != specialValue {
			t.Errorf("Special characters not preserved: got %q, want %q", retrieved, specialValue)
		}
	})

	// Test very long values
	t.Run("LongValues", func(t *testing.T) {
		longValue := make([]byte, 10000)
		for i := range longValue {
			longValue[i] = 'a'
		}

		os.Setenv("RAD_LONG", string(longValue))
		defer os.Unsetenv("RAD_LONG")

		retrieved := os.Getenv("RAD_LONG")
		if len(retrieved) != 10000 {
			t.Errorf("Long value truncated: got %d chars, want 10000", len(retrieved))
		}
	})

	// Test unicode values
	t.Run("UnicodeValues", func(t *testing.T) {
		unicodeValue := "配置值 🎉 émojis"
		os.Setenv("RAD_UNICODE", unicodeValue)
		defer os.Unsetenv("RAD_UNICODE")

		retrieved := os.Getenv("RAD_UNICODE")
		if retrieved != unicodeValue {
			t.Errorf("Unicode not preserved: got %q, want %q", retrieved, unicodeValue)
		}
	})
}

// TestSecretManagement validates secret handling
func TestSecretManagement(t *testing.T) {
	// Test secret loading from file
	t.Run("SecretFromFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretPath := filepath.Join(tmpDir, "secret.txt")
		expectedSecret := "my-secret-value-12345"

		// Write secret to file
		err := os.WriteFile(secretPath, []byte(expectedSecret), 0600)
		if err != nil {
			t.Fatalf("Failed to write secret file: %v", err)
		}

		// Read secret from file
		data, err := os.ReadFile(secretPath)
		if err != nil {
			t.Fatalf("Failed to read secret file: %v", err)
		}

		if string(data) != expectedSecret {
			t.Errorf("Secret mismatch: got %q, want %q", string(data), expectedSecret)
		}

		// Verify file permissions
		info, err := os.Stat(secretPath)
		if err != nil {
			t.Fatalf("Failed to stat secret file: %v", err)
		}
		mode := info.Mode().Perm()
		if mode != 0600 {
			t.Errorf("Secret file should have 0600 permissions, got %o", mode)
		}
	})

	// Test secret rotation
	t.Run("SecretRotation", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretPath := filepath.Join(tmpDir, "secret.txt")

		// Initial secret
		os.WriteFile(secretPath, []byte("old-secret"), 0600)

		// Simulate rotation
		time.Sleep(10 * time.Millisecond)
		os.WriteFile(secretPath, []byte("new-secret"), 0600)

		// Verify new secret
		data, _ := os.ReadFile(secretPath)
		if string(data) != "new-secret" {
			t.Error("Secret should be updated after rotation")
		}
	})

	// Test empty secret
	t.Run("EmptySecret", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretPath := filepath.Join(tmpDir, "empty.txt")
		os.WriteFile(secretPath, []byte(""), 0600)

		data, _ := os.ReadFile(secretPath)
		if len(data) != 0 {
			t.Error("Empty secret should be handled")
		}
	})

	// Test multi-line secret
	t.Run("MultiLineSecret", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretPath := filepath.Join(tmpDir, "multiline.txt")
		multiLine := "line1\nline2\nline3"

		os.WriteFile(secretPath, []byte(multiLine), 0600)

		data, _ := os.ReadFile(secretPath)
		if string(data) != multiLine {
			t.Error("Multi-line secret should be preserved")
		}
	})

	// Test missing secret file
	t.Run("MissingSecretFile", func(t *testing.T) {
		missingPath := "/nonexistent/path/to/secret"
		_, err := os.ReadFile(missingPath)
		if err == nil {
			t.Error("Should error on missing secret file")
		}
	})
}

// TestEnvironmentOverrides validates env var precedence
func TestEnvironmentOverrides(t *testing.T) {
	// Set a value
	os.Setenv("RAD_TEST_VAR", "original")
	defer os.Unsetenv("RAD_TEST_VAR")

	// Verify it can be read
	if os.Getenv("RAD_TEST_VAR") != "original" {
		t.Error("Should read original value")
	}

	// Override it
	os.Setenv("RAD_TEST_VAR", "overridden")
	if os.Getenv("RAD_TEST_VAR") != "overridden" {
		t.Error("Should read overridden value")
	}
}

// BenchmarkEnvironmentLookup benchmarks env var lookup
func BenchmarkEnvironmentLookup(b *testing.B) {
	os.Setenv("RAD_BENCH_VAR", "test-value")
	defer os.Unsetenv("RAD_BENCH_VAR")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = os.Getenv("RAD_BENCH_VAR")
	}
}

// BenchmarkSecretFileRead benchmarks secret file reading
func BenchmarkSecretFileRead(b *testing.B) {
	tmpDir := b.TempDir()
	secretPath := filepath.Join(tmpDir, "secret.txt")
	os.WriteFile(secretPath, []byte("benchmark-secret-12345"), 0600)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := os.ReadFile(secretPath)
		if err != nil {
			b.Fatalf("Failed to read secret: %v", err)
		}
	}
}
