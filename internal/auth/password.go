// Package auth provides password hashing functionality.
package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher handles password hashing and verification.
type PasswordHasher struct {
	cost int
}

// NewPasswordHasher creates a new password hasher with the specified cost.
// Cost should be between 4 and 31. Higher values are slower but more secure.
// Default is bcrypt.DefaultCost (10).
func NewPasswordHasher(cost int) *PasswordHasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return &PasswordHasher{cost: cost}
}

// DefaultPasswordHasher returns a password hasher with default cost.
func DefaultPasswordHasher() *PasswordHasher {
	return NewPasswordHasher(bcrypt.DefaultCost)
}

// Hash generates a bcrypt hash from a plaintext password.
func (p *PasswordHasher) Hash(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), p.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// Verify checks if a plaintext password matches a hash.
func (p *PasswordHasher) Verify(password, hash string) bool {
	if password == "" || hash == "" {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// IsValidHash checks if a string is a valid bcrypt hash.
func IsValidHash(hash string) bool {
	// bcrypt hashes start with $2a$, $2b$, or $2y$ followed by cost and salt+hash
	return len(hash) > 0 && (hash[:4] == "$2a$" || hash[:4] == "$2b$" || hash[:4] == "$2y$")
}
