package model

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// generateSecureID creates a secure random ID with a prefix
func generateSecureID(prefix string) string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// In a real application, we would handle this error better
		panic(err)
	}
	return fmt.Sprintf("%s%s", prefix, base64.RawURLEncoding.EncodeToString(b))
}

// generateSecureToken creates a secure random token string
func generateSecureToken() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		// In a real application, we would handle this error better
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
