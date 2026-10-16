package auth

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims of an access token: sub is the user id, sid the session id.
type Claims struct {
	SessionID int64 `json:"sid"`
	jwt.RegisteredClaims
}

// JWK is an RSA public key in JSON Web Key format (RFC 7517).
type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

// TokenManager signs access tokens with an RSA private key (RS256).
// The public key is taken from the private key; there is no separate file.
type TokenManager struct {
	privateKey *rsa.PrivateKey
	kid        string
	issuer     string
	accessTTL  time.Duration
}

func NewTokenManager(privateKeyPath, issuer string, accessTTL time.Duration) (*TokenManager, error) {
	pemBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w (run `make keys`)", err)
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	if key.N.BitLen() < 2048 {
		return nil, errors.New("private key must be at least 2048 bits")
	}

	return &TokenManager{
		privateKey: key,
		kid:        keyID(&key.PublicKey),
		issuer:     issuer,
		accessTTL:  accessTTL,
	}, nil
}

func (t *TokenManager) AccessTTL() time.Duration { return t.accessTTL }
func (t *TokenManager) Issuer() string           { return t.issuer }

func (t *TokenManager) IssueAccess(userID, sessionID int64) (string, error) {
	now := time.Now()
	claims := Claims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    t.issuer,
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.accessTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = t.kid
	return token.SignedString(t.privateKey)
}

func (t *TokenManager) ParseAccess(token string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) {
		return &t.privateKey.PublicKey, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuer(t.issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if claims.SessionID == 0 {
		return nil, errors.New("missing sid")
	}
	return claims, nil
}

func (t *TokenManager) JWKS() JWKS {
	pub := t.privateKey.PublicKey
	return JWKS{Keys: []JWK{{
		Kty: "RSA",
		Use: "sig",
		Alg: jwt.SigningMethodRS256.Alg(),
		Kid: t.kid,
		N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}}}
}

// kidNamespace scopes key IDs derived by keyID.
var kidNamespace = uuid.NewSHA1(uuid.NameSpaceURL, []byte("auth-api/jwt-kid"))

// keyID derives a UUID (v5) from the key's thumbprint:
// stable across restarts and changes whenever the key does.
func keyID(pub *rsa.PublicKey) string {
	return uuid.NewSHA1(kidNamespace, thumbprint(pub)).String()
}

// thumbprint computes the RFC 7638 JWK thumbprint (SHA-256).
func thumbprint(pub *rsa.PublicKey) []byte {
	// Members must be in lexicographic order with no whitespace.
	b, _ := json.Marshal(struct {
		E   string `json:"e"`
		Kty string `json:"kty"`
		N   string `json:"n"`
	}{
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		Kty: "RSA",
		N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
	})
	sum := sha256.Sum256(b)
	return sum[:]
}
