package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"auth-service/internal"

	"github.com/golang-jwt/jwt/v5"
)

// Key generation is slow, so the tests share keys.
var (
	testKey  = sync.OnceValue(func() *rsa.PrivateKey { return generateKey(2048) })
	otherKey = sync.OnceValue(func() *rsa.PrivateKey { return generateKey(2048) })
)

func generateKey(bits int) *rsa.PrivateKey {
	k, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		panic(err)
	}
	return k
}

func writeKey(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "private.pem")
	b := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: must(x509.MarshalPKCS8PrivateKey(key))})
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// newSettings loads dynamic settings from the given YAML; empty gives the defaults.
func newSettings(t *testing.T, yaml string) *internal.DynamicConfig {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := internal.LoadDynamic(path)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

const testIssuer = "https://auth.example.com"

func newTokens(t *testing.T, key *rsa.PrivateKey) *TokenManager {
	t.Helper()
	tm, err := NewTokenManager(writeKey(t, key), testIssuer, newSettings(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	return tm
}

// sign signs claims with the manager's key, for tokens IssueAccess would never make.
func sign(t *testing.T, tm *TokenManager, claims jwt.Claims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = tm.kid
	s, err := tok.SignedString(tm.privateKey)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNewTokenManagerRejects(t *testing.T) {
	settings := newSettings(t, "")
	dir := t.TempDir()

	garbage := filepath.Join(dir, "garbage.pem")
	if err := os.WriteFile(garbage, []byte("not a key"), 0o600); err != nil {
		t.Fatal(err)
	}

	for name, path := range map[string]string{
		"missing file": filepath.Join(dir, "missing.pem"),
		"not a pem":    garbage,
		"1024-bit key": writeKey(t, generateKey(1024)),
	} {
		if _, err := NewTokenManager(path, testIssuer, settings); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

func TestIssueAndParseAccess(t *testing.T) {
	tm := newTokens(t, testKey())
	token, err := tm.IssueAccess(41784020480034629, 41784020480034630)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := tm.ParseAccess(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "41784020480034629" || claims.SessionID != 41784020480034630 {
		t.Fatalf("got sub %q, sid %d", claims.Subject, claims.SessionID)
	}
	if claims.Issuer != testIssuer {
		t.Fatalf("got iss %q", claims.Issuer)
	}
	ttl := claims.ExpiresAt.Sub(claims.IssuedAt.Time)
	if ttl != tm.AccessTTL() {
		t.Fatalf("exp - iat = %v, want %v", ttl, tm.AccessTTL())
	}

	parsed, _, err := jwt.NewParser().ParseUnverified(token, &Claims{})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Header["kid"] != tm.kid || parsed.Header["alg"] != "RS256" {
		t.Fatalf("header %v", parsed.Header)
	}
}

func TestAccessTTLFromSettings(t *testing.T) {
	tm, err := NewTokenManager(writeKey(t, testKey()), testIssuer, newSettings(t, "access_token_ttl: 2m\n"))
	if err != nil {
		t.Fatal(err)
	}
	if tm.AccessTTL() != 2*time.Minute {
		t.Fatalf("got %v", tm.AccessTTL())
	}
}

func TestParseAccessRejects(t *testing.T) {
	tm := newTokens(t, testKey())
	now := time.Now()
	valid := func() Claims {
		return Claims{
			SessionID: 7,
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    testIssuer,
				Subject:   "1",
				IssuedAt:  jwt.NewNumericDate(now),
				ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			},
		}
	}

	expired := valid()
	expired.ExpiresAt = jwt.NewNumericDate(now.Add(-time.Minute))

	noExp := valid()
	noExp.ExpiresAt = nil

	wrongIssuer := valid()
	wrongIssuer.Issuer = "https://evil.example.com"

	noSID := valid()
	noSID.SessionID = 0

	otherSigner := newTokens(t, otherKey())

	hs256, err := jwt.NewWithClaims(jwt.SigningMethodHS256, valid()).SignedString([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, valid()).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}

	for name, token := range map[string]string{
		"expired":      sign(t, tm, expired),
		"no exp":       sign(t, tm, noExp),
		"wrong issuer": sign(t, tm, wrongIssuer),
		"no sid":       sign(t, tm, noSID),
		"other key":    sign(t, otherSigner, valid()),
		"hs256":        hs256,
		"alg none":     none,
		"garbage":      "not.a.jwt",
		"empty":        "",
	} {
		if _, err := tm.ParseAccess(token); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
	if _, err := tm.ParseAccess(sign(t, tm, valid())); err != nil {
		t.Fatalf("control token rejected: %v", err)
	}
}

func TestJWKSMatchesKey(t *testing.T) {
	tm := newTokens(t, testKey())
	jwks := tm.JWKS()
	if len(jwks.Keys) != 1 {
		t.Fatalf("got %d keys", len(jwks.Keys))
	}
	k := jwks.Keys[0]
	if k.Kty != "RSA" || k.Use != "sig" || k.Alg != "RS256" || k.Kid != tm.kid {
		t.Fatalf("got %+v", k)
	}

	n, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		t.Fatal(err)
	}
	e, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		t.Fatal(err)
	}
	pub := &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	if !pub.Equal(&testKey().PublicKey) {
		t.Fatal("JWKS key is not the signing key")
	}
}

// The kid is the same after a restart with the same key and changes with the key.
func TestKeyIDStable(t *testing.T) {
	a, b := newTokens(t, testKey()), newTokens(t, testKey())
	if a.kid != b.kid {
		t.Fatalf("same key, different kid: %s, %s", a.kid, b.kid)
	}
	if c := newTokens(t, otherKey()); c.kid == a.kid {
		t.Fatal("different keys, same kid")
	}
}

// The example from RFC 7638, section 3.1.
func TestThumbprintRFC7638(t *testing.T) {
	const (
		n    = "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw"
		want = "NzbLsXh8uDCcd-6MNwXF4W_7noWXFZAfHkxZsRGC9Xs"
	)
	pub := &rsa.PublicKey{N: new(big.Int).SetBytes(must(base64.RawURLEncoding.DecodeString(n))), E: 65537}
	if got := base64.RawURLEncoding.EncodeToString(thumbprint(pub)); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
