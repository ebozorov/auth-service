package integration

import (
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"net/http"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestRefreshRotates(t *testing.T) {
	u := createUser(t, "user")
	c := newClient(t)
	tok := c.login(u.Email)

	r := c.do("POST", "/auth/refresh", map[string]string{"refresh_token": tok.RefreshToken}, "")
	expectStatus(t, r, http.StatusOK)
	var next tokens
	r.json(t, &next)
	if next.AccessToken == "" || next.RefreshToken == "" || next.RefreshToken == tok.RefreshToken {
		t.Fatalf("got %+v", next)
	}

	// Each refresh token works once.
	expectStatus(t, c.do("POST", "/auth/refresh", map[string]string{"refresh_token": tok.RefreshToken}, ""), http.StatusUnauthorized)
	expectStatus(t, c.do("POST", "/auth/refresh", map[string]string{"refresh_token": next.RefreshToken}, ""), http.StatusOK)
	expectStatus(t, c.do("GET", "/auth/me", nil, next.AccessToken), http.StatusOK)
}

func TestRefreshRejects(t *testing.T) {
	c := newClient(t)
	expectStatus(t, c.do("POST", "/auth/refresh", nil, ""), http.StatusBadRequest)
	expectStatus(t, c.do("POST", "/auth/refresh", map[string]string{"refresh_token": "made-up"}, ""), http.StatusUnauthorized)
}

// An expired session is closed for both tokens, even before the cleanup archives it.
func TestExpiredSession(t *testing.T) {
	u := createUser(t, "user")
	c := newClient(t)
	tok := c.login(u.Email)

	exec(t, "UPDATE sessions SET expires_at = now() - interval '1 second' WHERE user_id = ?", u.ID)
	expectStatus(t, c.do("GET", "/auth/me", nil, tok.AccessToken), http.StatusUnauthorized)
	expectStatus(t, c.do("POST", "/auth/refresh", map[string]string{"refresh_token": tok.RefreshToken}, ""), http.StatusUnauthorized)
}

func TestLogout(t *testing.T) {
	u := createUser(t, "user")
	c := newClient(t)
	tok := c.login(u.Email)

	expectStatus(t, c.do("POST", "/auth/logout", nil, tok.AccessToken), http.StatusNoContent)

	// The access token stops working right away, not when it expires.
	expectStatus(t, c.do("GET", "/auth/me", nil, tok.AccessToken), http.StatusUnauthorized)
	expectStatus(t, c.do("POST", "/auth/refresh", map[string]string{"refresh_token": tok.RefreshToken}, ""), http.StatusUnauthorized)

	var reason string
	if err := db.Raw("SELECT ended->>'reason' FROM ended_sessions WHERE user_id = ?", u.ID).Scan(&reason).Error; err != nil {
		t.Fatal(err)
	}
	if reason != "logout" {
		t.Fatalf("archived with reason %q, want logout", reason)
	}
}

func TestRequireAuth(t *testing.T) {
	c := newClient(t)
	for _, token := range []string{"", "made-up"} {
		r := c.do("GET", "/auth/me", nil, token)
		expectStatus(t, r, http.StatusUnauthorized)
	}
}

// Other services verify access tokens with the published key alone.
func TestAccessTokenVerifiesWithJWKS(t *testing.T) {
	u := createUser(t, "user")
	c := newClient(t)
	tok := c.login(u.Email)

	var oidc struct {
		Issuer  string `json:"issuer"`
		JWKSURI string `json:"jwks_uri"`
	}
	c.do("GET", "/.well-known/openid-configuration", nil, "").json(t, &oidc)
	if oidc.Issuer != baseURL || oidc.JWKSURI != baseURL+"/.well-known/jwks.json" {
		t.Fatalf("got %+v", oidc)
	}

	var jwks struct {
		Keys []struct {
			Kid, N, E string
		} `json:"keys"`
	}
	c.do("GET", "/.well-known/jwks.json", nil, "").json(t, &jwks)

	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tok.AccessToken, claims, func(token *jwt.Token) (any, error) {
		for _, k := range jwks.Keys {
			if k.Kid == token.Header["kid"] {
				n, _ := base64.RawURLEncoding.DecodeString(k.N)
				e, _ := base64.RawURLEncoding.DecodeString(k.E)
				return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}, nil
			}
		}
		return nil, jwt.ErrTokenUnverifiable
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(oidc.Issuer))
	if err != nil {
		t.Fatal(err)
	}
	if claims["sub"] != u.ID {
		t.Fatalf("sub %v, want %s", claims["sub"], u.ID)
	}
	if _, ok := claims["sid"].(string); !ok {
		t.Fatalf("sid %v is not a string", claims["sid"])
	}
}

func TestInfo(t *testing.T) {
	c := newClient(t)
	r := c.do("GET", "/", nil, "")
	expectStatus(t, r, http.StatusOK)
	var info struct {
		Status string            `json:"status"`
		Env    string            `json:"env"`
		Checks map[string]string `json:"checks"`
	}
	r.json(t, &info)
	if info.Status != "ok" || info.Env != "test" || info.Checks["database"] != "ok" {
		t.Fatalf("got %s", r.body)
	}

	// An unknown API path is a JSON 404, not the web UI.
	r = c.do("GET", "/no-such-path", nil, "")
	expectStatus(t, r, http.StatusNotFound)
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type %q", ct)
	}
}
