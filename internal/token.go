package internal

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// NewToken returns a random secret and its hash. Only the hash is stored in the database.
// Used for refresh tokens, magic link tokens and device secrets.
func NewToken() (token, hash string) {
	b := make([]byte, 32)
	rand.Read(b) // never returns an error
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashToken(token)
}

// HashToken returns the SHA-256 hash of a token, hex encoded.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
