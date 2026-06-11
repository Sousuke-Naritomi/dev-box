// Package security provides cryptographically secure password generation
// for use in generated environment files and service configurations.
package security

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercase = "abcdefghijklmnopqrstuvwxyz"
	digits    = "0123456789"
	symbols   = "!@#$%^&*()-_=+[]{}|;:,.<>?"

	// MaxPasswordLength defines the maximum allowed password length to prevent DoS.
	MaxPasswordLength = 1024
)

// GeneratePassword generates a cryptographically secure random password of
// the specified length. The password is guaranteed to contain at least one
// uppercase letter, one lowercase letter, one digit, and one symbol.
//
// Returns an error if length < 4 (minimum to satisfy all character class
// requirements) or if the cryptographic random number generator fails.
// There is no hardcoded fallback password.
func GeneratePassword(length int) (string, error) {
	if length < 4 {
		return "", fmt.Errorf("password length must be at least 4, got %d", length)
	}
	if length > MaxPasswordLength {
		return "", fmt.Errorf("password length must not exceed %d, got %d", MaxPasswordLength, length)
	}

	allChars := uppercase + lowercase + digits + symbols
	password := make([]byte, length)

	// Guarantee at least one character from each required class.
	charSets := []string{uppercase, lowercase, digits, symbols}
	for i, set := range charSets {
		idx, err := randInt(len(set))
		if err != nil {
			return "", fmt.Errorf("failed to generate random index: %w", err)
		}
		password[i] = set[idx]
	}

	// Fill remaining positions with random characters from the full set.
	for i := 4; i < length; i++ {
		idx, err := randInt(len(allChars))
		if err != nil {
			return "", fmt.Errorf("failed to generate random index: %w", err)
		}
		password[i] = allChars[idx]
	}

	// Shuffle to ensure character positions are not predictable.
	if err := shuffle(password); err != nil {
		return "", fmt.Errorf("failed to shuffle password: %w", err)
	}

	return string(password), nil
}

// randInt returns a cryptographically secure random integer in [0, max).
func randInt(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}

// shuffle performs a Fisher-Yates shuffle using crypto/rand.
func shuffle(b []byte) error {
	for i := len(b) - 1; i > 0; i-- {
		j, err := randInt(i + 1)
		if err != nil {
			return err
		}
		b[i], b[j] = b[j], b[i]
	}
	return nil
}
