package integration

import (
	"net/http"
	"testing"
)

type sessionInfo struct {
	ID     string `json:"id"`
	Device struct {
		Name   string `json:"name"`
		Source string `json:"source"`
	} `json:"device"`
	StartedIP string `json:"started_ip"`
	CurrentIP string `json:"current_ip"`
	Current   bool   `json:"current"`
}

// device is one signed-in device of a user.
type device struct {
	c   *client
	tok tokens
}

func signInDevices(t *testing.T, email string, n int) []device {
	t.Helper()
	devices := make([]device, n)
	for i := range devices {
		c := newClient(t)
		devices[i] = device{c, c.login(email)}
	}
	return devices
}

func (d device) sessions(t *testing.T) []sessionInfo {
	t.Helper()
	r := d.c.do("GET", "/sessions", nil, d.tok.AccessToken)
	expectStatus(t, r, http.StatusOK)
	var list []sessionInfo
	r.json(t, &list)
	return list
}

// currentID is the id of the device's own session.
func (d device) currentID(t *testing.T) string {
	t.Helper()
	for _, s := range d.sessions(t) {
		if s.Current {
			return s.ID
		}
	}
	t.Fatal("no current session")
	return ""
}

func TestSessionsList(t *testing.T) {
	u := createUser(t, "user")
	devs := signInDevices(t, u.Email, 2)

	list := devs[0].sessions(t)
	if len(list) != 2 {
		t.Fatalf("got %d sessions, want 2", len(list))
	}
	current := 0
	for _, s := range list {
		if s.Current {
			current++
			if s.CurrentIP != devs[0].c.ip {
				t.Errorf("current_ip %s, want %s", s.CurrentIP, devs[0].c.ip)
			}
		}
	}
	if current != 1 {
		t.Fatalf("%d sessions marked current, want 1", current)
	}
}

func TestSessionsTerminate(t *testing.T) {
	u := createUser(t, "user")
	devs := signInDevices(t, u.Email, 2)
	a, b := devs[0], devs[1]
	bID := b.currentID(t)

	expectStatus(t, a.c.do("DELETE", "/sessions/"+bID, nil, a.tok.AccessToken), http.StatusNoContent)

	// The terminated device loses access right away, refresh included.
	expectStatus(t, b.c.do("GET", "/auth/me", nil, b.tok.AccessToken), http.StatusUnauthorized)
	expectStatus(t, b.c.do("POST", "/auth/refresh", map[string]string{"refresh_token": b.tok.RefreshToken}, ""), http.StatusUnauthorized)

	if list := a.sessions(t); len(list) != 1 || !list[0].Current {
		t.Fatalf("got %+v", list)
	}

	// Already ended, unknown and not numeric are all just not found.
	for _, id := range []string{bID, "1", "abc"} {
		expectStatus(t, a.c.do("DELETE", "/sessions/"+id, nil, a.tok.AccessToken), http.StatusNotFound)
	}
}

// A user cannot end another user's session, even knowing its id.
func TestSessionsTerminateOtherUser(t *testing.T) {
	owner := signInDevices(t, createUser(t, "user").Email, 1)[0]
	stranger := signInDevices(t, createUser(t, "user").Email, 1)[0]

	r := stranger.c.do("DELETE", "/sessions/"+owner.currentID(t), nil, stranger.tok.AccessToken)
	expectStatus(t, r, http.StatusNotFound)
	expectStatus(t, owner.c.do("GET", "/auth/me", nil, owner.tok.AccessToken), http.StatusOK)
}

func TestSessionsTerminateOthers(t *testing.T) {
	u := createUser(t, "user")
	devs := signInDevices(t, u.Email, 3)

	r := devs[0].c.do("DELETE", "/sessions", nil, devs[0].tok.AccessToken)
	expectStatus(t, r, http.StatusOK)
	var res struct {
		Terminated int `json:"terminated"`
	}
	r.json(t, &res)
	if res.Terminated != 2 {
		t.Fatalf("terminated %d, want 2", res.Terminated)
	}

	expectStatus(t, devs[0].c.do("GET", "/auth/me", nil, devs[0].tok.AccessToken), http.StatusOK)
	for _, d := range devs[1:] {
		expectStatus(t, d.c.do("GET", "/auth/me", nil, d.tok.AccessToken), http.StatusUnauthorized)
	}
	if list := devs[0].sessions(t); len(list) != 1 {
		t.Fatalf("got %d sessions, want 1", len(list))
	}
}
