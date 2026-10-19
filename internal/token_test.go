package internal

import (
	"encoding/base64"
	"testing"
)

func TestNewToken(t *testing.T) {
	seen := make(map[string]bool)
	for range 100 {
		token, hash := NewToken()
		b, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil || len(b) != 32 {
			t.Fatalf("token %q is not 32 bytes of base64url: %v", token, err)
		}
		if hash != HashToken(token) {
			t.Fatalf("hash %q is not HashToken(token)", hash)
		}
		if seen[token] {
			t.Fatalf("token %q repeated", token)
		}
		seen[token] = true
	}
}

func TestHashToken(t *testing.T) {
	// SHA-256 of "abc" (FIPS 180-2).
	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got := HashToken("abc"); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
