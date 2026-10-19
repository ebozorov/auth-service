package integration

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSignInApp(t *testing.T) {
	u := createUser(t, "user")
	c := newClient(t)

	// The email is normalized: case and spaces do not matter.
	r := c.do("POST", "/auth/sign-in", map[string]any{
		"email":  "  " + strings.ToUpper(u.Email) + " ",
		"device": map[string]string{"name": "Pixel 9", "os": "Android 15", "client": "MyApp 2.4"},
	}, "")
	expectStatus(t, r, http.StatusAccepted)
	var res signInResult
	r.json(t, &res)
	if res.DeviceSecret == "" || res.ExpiresIn != 600 {
		t.Fatalf("got %+v", res)
	}

	r = c.do("POST", "/auth/sign-in/link", map[string]string{"token": waitForLink(t, u.Email), "device_secret": res.DeviceSecret}, "")
	expectStatus(t, r, http.StatusOK)
	var tok tokens
	r.json(t, &tok)
	if tok.AccessToken == "" || tok.RefreshToken == "" || tok.TokenType != "Bearer" || tok.ExpiresIn != 900 {
		t.Fatalf("got %+v", tok)
	}

	var me user
	r = c.do("GET", "/auth/me", nil, tok.AccessToken)
	expectStatus(t, r, http.StatusOK)
	r.json(t, &me)
	if me.ID != u.ID || me.Email != u.Email {
		t.Fatalf("me = %+v, want %+v", me, u)
	}

	// The device the client sent is what the device list shows.
	var list []sessionInfo
	c.do("GET", "/sessions", nil, tok.AccessToken).json(t, &list)
	if len(list) != 1 || list[0].Device.Name != "Pixel 9" || list[0].Device.Source != "client" || !list[0].Current {
		t.Fatalf("got %+v", list)
	}
	if list[0].StartedIP != c.ip {
		t.Fatalf("started_ip %s, want %s", list[0].StartedIP, c.ip)
	}
}

// A browser keeps the device secret and the refresh token in cookies.
func TestSignInBrowser(t *testing.T) {
	u := createUser(t, "user")
	c := newClient(t)

	r := c.do("POST", "/auth/sign-in", map[string]string{"email": u.Email}, "")
	expectStatus(t, r, http.StatusAccepted)
	secret := r.cookie("device_secret")
	if secret == nil || !secret.HttpOnly || !secret.Secure || secret.Path != "/api/v1/auth/sign-in" {
		t.Fatalf("device_secret cookie: %+v", secret)
	}
	token := waitForLink(t, u.Email)

	// The same link opened on another device does nothing.
	other := newClient(t)
	r = other.do("GET", "/auth/sign-in/link?token="+token, nil, "")
	expectStatus(t, r, http.StatusFound)
	if loc := r.Header.Get("Location"); loc != "/?error=invalid_link" {
		t.Fatalf("other device redirected to %q", loc)
	}

	r = c.do("GET", "/auth/sign-in/link?token="+token, nil, "")
	expectStatus(t, r, http.StatusFound)
	if loc := r.Header.Get("Location"); loc != "/" {
		t.Fatalf("redirected to %q", loc)
	}
	refresh := r.cookie("refresh_token")
	if refresh == nil || refresh.Value == "" || !refresh.HttpOnly || refresh.Path != "/api/v1/auth" {
		t.Fatalf("refresh_token cookie: %+v", refresh)
	}
	if _, ok := c.cookies["device_secret"]; ok {
		t.Fatal("device_secret cookie was not cleared")
	}

	// Refresh with the cookie: the new refresh token goes to the cookie, not the body.
	r = c.do("POST", "/auth/refresh", nil, "")
	expectStatus(t, r, http.StatusOK)
	var tok tokens
	r.json(t, &tok)
	if tok.AccessToken == "" || tok.RefreshToken != "" {
		t.Fatalf("got %+v", tok)
	}
	rotated := r.cookie("refresh_token")
	if rotated == nil || rotated.Value == refresh.Value {
		t.Fatal("refresh token cookie was not rotated")
	}
	expectStatus(t, c.do("GET", "/auth/me", nil, tok.AccessToken), http.StatusOK)

	// The old cookie no longer works, and the browser is told to drop it.
	stale := newClient(t)
	stale.cookies["refresh_token"] = refresh
	r = stale.do("POST", "/auth/refresh", nil, "")
	expectStatus(t, r, http.StatusUnauthorized)
	if _, ok := stale.cookies["refresh_token"]; ok {
		t.Fatal("stale refresh cookie was not cleared")
	}
}

func TestSignInLinkWorksOnce(t *testing.T) {
	u := createUser(t, "user")
	c := newClient(t)
	secret := c.signIn(u.Email)
	body := map[string]string{"token": waitForLink(t, u.Email), "device_secret": secret}

	expectStatus(t, c.do("POST", "/auth/sign-in/link", body, ""), http.StatusOK)
	expectStatus(t, c.do("POST", "/auth/sign-in/link", body, ""), http.StatusUnauthorized)
}

// A wrong device secret is rejected and does not use up the link.
func TestSignInLinkNeedsDeviceSecret(t *testing.T) {
	u := createUser(t, "user")
	c := newClient(t)
	secret := c.signIn(u.Email)
	token := waitForLink(t, u.Email)

	r := c.do("POST", "/auth/sign-in/link", map[string]string{"token": token, "device_secret": "wrong"}, "")
	expectStatus(t, r, http.StatusUnauthorized)
	r = c.do("POST", "/auth/sign-in/link", map[string]string{"token": token, "device_secret": secret}, "")
	expectStatus(t, r, http.StatusOK)
}

func TestSignInExpiredLink(t *testing.T) {
	u := createUser(t, "user")
	c := newClient(t)
	secret := c.signIn(u.Email)
	token := waitForLink(t, u.Email)

	exec(t, "UPDATE signin_requests SET expires_at = now() - interval '1 second' WHERE user_id = ?", u.ID)
	r := c.do("POST", "/auth/sign-in/link", map[string]string{"token": token, "device_secret": secret}, "")
	expectStatus(t, r, http.StatusUnauthorized)
}

// A user disabled after asking for a link cannot use it.
func TestSignInDisabledAfterRequest(t *testing.T) {
	u := createUser(t, "user")
	c := newClient(t)
	secret := c.signIn(u.Email)
	token := waitForLink(t, u.Email)

	setEnabled(t, u.ID, false)
	r := c.do("POST", "/auth/sign-in/link", map[string]string{"token": token, "device_secret": secret}, "")
	expectStatus(t, r, http.StatusUnauthorized)
}

// Unknown and disabled emails get the same answer as known ones, and no email.
func TestSignInDoesNotRevealEmail(t *testing.T) {
	disabled := createUser(t, "user")
	setEnabled(t, disabled.ID, false)
	unknown := uniqueEmail(t)

	for _, email := range []string{unknown, disabled.Email} {
		c := newClient(t)
		r := c.do("POST", "/auth/sign-in", map[string]string{"email": email}, "")
		expectStatus(t, r, http.StatusAccepted)
		var res signInResult
		r.json(t, &res)
		if res.DeviceSecret == "" || res.ExpiresIn != 600 || r.cookie("device_secret") == nil {
			t.Fatalf("%s: got %s", email, r.body)
		}
	}
	expectNoMail(t, unknown)
	expectNoMail(t, disabled.Email)
}

// Over the per-user limit the answer stays the same, only no more emails are sent.
func TestSignInUserLimit(t *testing.T) {
	u := createUser(t, "user")
	for range userLimit + 2 {
		newClient(t).signIn(u.Email)
	}
	waitFor(t, "limit emails", func() bool { return len(mailTo(t, u.Email)) >= userLimit })
	// Emails are sent in the background: give extra ones time to arrive.
	time.Sleep(time.Second)
	if n := len(mailTo(t, u.Email)); n != userLimit {
		t.Fatalf("got %d emails, want %d", n, userLimit)
	}
}

func TestSignInIPLimit(t *testing.T) {
	c := newClient(t)
	email := uniqueEmail(t)
	for range ipLimit {
		r := c.do("POST", "/auth/sign-in", map[string]string{"email": email}, "")
		expectStatus(t, r, http.StatusAccepted)
	}
	expectStatus(t, c.do("POST", "/auth/sign-in", map[string]string{"email": email}, ""), http.StatusTooManyRequests)
	// Guessing link tokens from the same IP is blocked too.
	expectStatus(t, c.do("POST", "/auth/sign-in/link", map[string]string{"token": "x", "device_secret": "y"}, ""), http.StatusTooManyRequests)

	// Other IPs are not affected.
	expectStatus(t, newClient(t).do("POST", "/auth/sign-in", map[string]string{"email": email}, ""), http.StatusAccepted)
}

// Failed links count toward the IP limit, so link tokens cannot be guessed.
func TestSignInLinkGuessingIsLimited(t *testing.T) {
	c := newClient(t)
	for range ipLimit {
		r := c.do("POST", "/auth/sign-in/link", map[string]string{"token": "guess", "device_secret": "guess"}, "")
		expectStatus(t, r, http.StatusUnauthorized)
	}
	r := c.do("POST", "/auth/sign-in/link", map[string]string{"token": "guess", "device_secret": "guess"}, "")
	expectStatus(t, r, http.StatusTooManyRequests)

	r = c.do("GET", "/auth/sign-in/link?token=guess", nil, "")
	expectStatus(t, r, http.StatusFound)
	if loc := r.Header.Get("Location"); loc != "/?error=too_many_requests" {
		t.Fatalf("redirected to %q", loc)
	}
}

func TestSignInRejectsBadEmail(t *testing.T) {
	r := newClient(t).do("POST", "/auth/sign-in", map[string]string{"email": "not-an-email"}, "")
	expectStatus(t, r, http.StatusBadRequest)
	var e struct {
		Error string `json:"error"`
	}
	r.json(t, &e)
	if e.Error != "invalid email" {
		t.Fatalf("got %s", r.body)
	}
}
