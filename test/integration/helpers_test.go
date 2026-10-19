package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var ipCounter atomic.Int32

// client is one device: its own IP and its own cookies.
type client struct {
	t       *testing.T
	ip      string
	cookies map[string]*http.Cookie
	http    *http.Client
}

func newClient(t *testing.T) *client {
	t.Helper()
	n := ipCounter.Add(1)
	return &client{
		t:       t,
		ip:      fmt.Sprintf("10.0.%d.%d", n/250, n%250+1),
		cookies: make(map[string]*http.Cookie),
		http: &http.Client{
			Timeout: 10 * time.Second,
			// The link redirect is checked, not followed.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

type response struct {
	*http.Response
	body []byte
}

// json decodes the body into v.
func (r response) json(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.body, v); err != nil {
		t.Fatalf("decode %s: %v", r.body, err)
	}
}

func (r response) cookie(name string) *http.Cookie {
	for _, c := range r.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// do sends a request to the API; body is sent as JSON when not nil, token as a Bearer token.
// Cookies are kept like a browser does. Go's cookiejar never sends Secure cookies
// over plain http, so they are kept here by hand.
func (c *client) do(method, path string, body any, token string) response {
	c.t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	if !strings.HasPrefix(path, "/.well-known") {
		path = "/api/v1" + path
	}
	req, err := http.NewRequest(method, baseURL+path, r)
	if err != nil {
		c.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("X-Forwarded-For", c.ip)
	req.Header.Set("User-Agent", "integration-test/1.0")
	for _, ck := range c.cookies {
		if strings.HasPrefix(req.URL.Path, ck.Path) {
			req.AddCookie(&http.Cookie{Name: ck.Name, Value: ck.Value})
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		c.t.Fatal(err)
	}
	for _, ck := range resp.Cookies() {
		if ck.Value == "" || (!ck.Expires.IsZero() && ck.Expires.Before(time.Now())) {
			delete(c.cookies, ck.Name)
		} else {
			c.cookies[ck.Name] = ck
		}
	}
	return response{resp, b}
}

func expectStatus(t *testing.T, r response, want int) {
	t.Helper()
	if r.StatusCode != want {
		t.Fatalf("%s %s: got %d %s, want %d", r.Request.Method, r.Request.URL.Path, r.StatusCode, r.body, want)
	}
}

type tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

type signInResult struct {
	DeviceSecret string `json:"device_secret"`
	ExpiresIn    int    `json:"expires_in"`
}

// signIn asks for a link and returns the device secret.
func (c *client) signIn(email string) string {
	c.t.Helper()
	r := c.do("POST", "/auth/sign-in", map[string]any{"email": email, "device": map[string]string{"name": "Test device", "os": "TestOS 1"}}, "")
	expectStatus(c.t, r, http.StatusAccepted)
	var res signInResult
	r.json(c.t, &res)
	if res.DeviceSecret == "" {
		c.t.Fatalf("no device secret: %s", r.body)
	}
	return res.DeviceSecret
}

// login signs in like an app does: the link token and device secret go in the body.
func (c *client) login(email string) tokens {
	c.t.Helper()
	secret := c.signIn(email)
	r := c.do("POST", "/auth/sign-in/link", map[string]string{"token": waitForLink(c.t, email), "device_secret": secret}, "")
	expectStatus(c.t, r, http.StatusOK)
	var tok tokens
	r.json(c.t, &tok)
	return tok
}

// user is a user created by the admin, with one client to sign in from.
type user struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

var emailCounter atomic.Int32

// uniqueEmail returns an email no other test uses.
func uniqueEmail(t *testing.T) string {
	name := strings.ToLower(regexp.MustCompile(`[^A-Za-z0-9]+`).ReplaceAllString(t.Name(), "-"))
	return fmt.Sprintf("%s-%d@test.local", name, emailCounter.Add(1))
}

// createUser adds an enabled user with the given role through the admin API.
func createUser(t *testing.T, role string) user {
	t.Helper()
	ad := admin(t)
	r := ad.c.do("POST", "/users", map[string]string{"email": uniqueEmail(t), "name": "Test User", "role": role}, ad.tok.AccessToken)
	expectStatus(t, r, http.StatusCreated)
	var u user
	r.json(t, &u)
	return u
}

// adminSession is the initial admin, signed in once and shared by all tests:
// the per-user limit would not allow a sign-in per test.
type adminSession struct {
	c   *client
	tok tokens
	id  string
}

var (
	adminMu   sync.Mutex
	adminSess *adminSession
)

func admin(t *testing.T) *adminSession {
	t.Helper()
	adminMu.Lock()
	defer adminMu.Unlock()
	if adminSess == nil {
		c := newClient(t)
		tok := c.login(adminEmail)
		var me user
		c.do("GET", "/auth/me", nil, tok.AccessToken).json(t, &me)
		adminSess = &adminSession{c: c, tok: tok, id: me.ID}
	}
	// The client is bound to the test that created it; it reports to the current one.
	adminSess.c.t = t
	return adminSess
}

// setEnabled enables or disables a user through the admin API.
func setEnabled(t *testing.T, userID string, enabled bool) {
	t.Helper()
	ad := admin(t)
	r := ad.c.do("PATCH", "/users/"+userID, map[string]bool{"enabled": enabled}, ad.tok.AccessToken)
	expectStatus(t, r, http.StatusOK)
}

// Mailpit

type mailMessage struct {
	ID string `json:"ID"`
	To []struct {
		Address string `json:"Address"`
	} `json:"To"`
}

var linkPattern = regexp.MustCompile(`(\S+/api/v1/auth/sign-in/link)\?token=(\S+)`)

// waitForLink waits for a sign-in email to the address, deletes it and returns the token from its link.
func waitForLink(t *testing.T, email string) string {
	t.Helper()
	var ids []string
	waitFor(t, "email to "+email, func() bool {
		ids = mailTo(t, email)
		return len(ids) > 0
	})
	if len(ids) != 1 {
		t.Fatalf("%d emails to %s, want 1", len(ids), email)
	}

	var msg struct {
		Subject string `json:"Subject"`
		Text    string `json:"Text"`
	}
	mailpit(t, "GET", "/api/v1/message/"+ids[0], nil, &msg)
	mailpit(t, "DELETE", "/api/v1/messages", map[string][]string{"IDs": ids}, nil)

	m := linkPattern.FindStringSubmatch(msg.Text)
	if m == nil {
		t.Fatalf("no link in email: %q", msg.Text)
	}
	if m[1] != baseURL+"/api/v1/auth/sign-in/link" {
		t.Fatalf("link goes to %s, want PUBLIC_URL", m[1])
	}
	token, err := url.QueryUnescape(m[2])
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// expectNoMail checks that no email to the address arrives. Emails are sent in the
// background, so it waits a little first.
func expectNoMail(t *testing.T, email string) {
	t.Helper()
	time.Sleep(time.Second)
	if ids := mailTo(t, email); len(ids) > 0 {
		t.Fatalf("got %d emails to %s, want none", len(ids), email)
	}
}

// mailTo returns the ids of the emails sent to the address.
func mailTo(t *testing.T, email string) []string {
	t.Helper()
	var list struct {
		Messages []mailMessage `json:"messages"`
	}
	mailpit(t, "GET", "/api/v1/messages?limit=1000", nil, &list)
	var ids []string
	for _, m := range list.Messages {
		for _, to := range m.To {
			if strings.EqualFold(to.Address, email) {
				ids = append(ids, m.ID)
			}
		}
	}
	return ids
}

func mailpit(t *testing.T, method, path string, body, out any) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, mailpitURL+path, r)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("mailpit %s %s: %d %s", method, path, resp.StatusCode, b)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatal(err)
		}
	}
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// exec runs SQL directly, only to move time forward.
func exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatal(err)
	}
}
