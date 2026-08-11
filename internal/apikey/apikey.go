package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

const keyBytes = 32 // 256 bits

var ErrInvalidKey = errors.New("invalid api key")

// Generate creates a new random API key, hex-encoded.
// This raw value is shown to the user exactly once and never stored directly.
func Generate() (string, error) {
	b := make([]byte, keyBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Hash returns the SHA-256 hash of a raw API key, hex-encoded.
// This is what gets stored in the database and compared against on lookup.
func Hash(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}
