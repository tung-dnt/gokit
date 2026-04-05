// Package util provides utilities shared across all domain packages.
package util //nolint:revive // shared is an intentional package name for cross-domain utilities

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateID returns a random 16-character hex string for use as a domain entity ID.
func GenerateID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generateID: %w", err)
	}
	return hex.EncodeToString(b), nil
}
