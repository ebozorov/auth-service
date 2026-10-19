package integration

import (
	"net/http"
	"slices"
	"strings"
	"testing"
)

func TestAdminOnly(t *testing.T) {
	u := createUser(t, "user")
	c := newClient(t)
	tok := c.login(u.Email)

	expectStatus(t, c.do("GET", "/admin/users", nil, ""), http.StatusUnauthorized)
	expectStatus(t, c.do("GET", "/admin/users", nil, tok.AccessToken), http.StatusForbidden)
	expectStatus(t, c.do("POST", "/admin/users", map[string]string{"email": uniqueEmail(t), "name": "X"}, tok.AccessToken), http.StatusForbidden)
}

func TestAdminCreateUser(t *testing.T) {
	ad := admin(t)
	email := uniqueEmail(t)

	r := ad.c.do("POST", "/admin/users", map[string]string{"email": "  " + email + "  ", "name": " New User "}, ad.tok.AccessToken)
	expectStatus(t, r, http.StatusCreated)
	var u struct {
		user
		Enabled bool `json:"enabled"`
	}
	r.json(t, &u)
	if u.ID == "" || u.Email != email || u.Name != "New User" || u.Role != "user" || !u.Enabled {
		t.Fatalf("got %s", r.body)
	}

	// The same email in another case is the same user.
	r = ad.c.do("POST", "/admin/users", map[string]string{"email": strings.ToUpper(email), "name": "Dup"}, ad.tok.AccessToken)
	expectStatus(t, r, http.StatusConflict)

	var list []user
	ad.c.do("GET", "/admin/users", nil, ad.tok.AccessToken).json(t, &list)
	if !slices.ContainsFunc(list, func(x user) bool { return x.ID == u.ID }) {
		t.Fatal("new user is not listed")
	}

	expectStatus(t, ad.c.do("POST", "/admin/users", map[string]string{"email": uniqueEmail(t), "name": "X", "role": "root"}, ad.tok.AccessToken), http.StatusBadRequest)
}

func TestAdminDisableEndsSessions(t *testing.T) {
	u := createUser(t, "user")
	c := newClient(t)
	tok := c.login(u.Email)

	setEnabled(t, u.ID, false)
	expectStatus(t, c.do("GET", "/auth/me", nil, tok.AccessToken), http.StatusUnauthorized)
	expectStatus(t, c.do("POST", "/auth/refresh", map[string]string{"refresh_token": tok.RefreshToken}, ""), http.StatusUnauthorized)

	var reason string
	if err := db.Raw("SELECT ended->>'reason' FROM ended_sessions WHERE user_id = ?", u.ID).Scan(&reason).Error; err != nil {
		t.Fatal(err)
	}
	if reason != "disabled" {
		t.Fatalf("archived with reason %q, want disabled", reason)
	}

	// Enabled again, the user can sign in again.
	setEnabled(t, u.ID, true)
	c = newClient(t)
	tok = c.login(u.Email)
	expectStatus(t, c.do("GET", "/auth/me", nil, tok.AccessToken), http.StatusOK)
}

// The admin cannot lock themselves out, so there is always an admin.
func TestAdminCannotDemoteOrDisableSelf(t *testing.T) {
	ad := admin(t)
	for _, body := range []map[string]any{{"role": "user"}, {"enabled": false}} {
		expectStatus(t, ad.c.do("PATCH", "/admin/users/"+ad.id, body, ad.tok.AccessToken), http.StatusBadRequest)
	}
	expectStatus(t, ad.c.do("GET", "/admin/users", nil, ad.tok.AccessToken), http.StatusOK)
}

// The role is read on every request: a change applies to tokens already issued.
func TestAdminRoleChangeAppliesRightAway(t *testing.T) {
	ad := admin(t)
	u := createUser(t, "user")
	c := newClient(t)
	tok := c.login(u.Email)

	expectStatus(t, c.do("GET", "/admin/users", nil, tok.AccessToken), http.StatusForbidden)
	expectStatus(t, ad.c.do("PATCH", "/admin/users/"+u.ID, map[string]string{"role": "admin"}, ad.tok.AccessToken), http.StatusOK)
	expectStatus(t, c.do("GET", "/admin/users", nil, tok.AccessToken), http.StatusOK)
	expectStatus(t, ad.c.do("PATCH", "/admin/users/"+u.ID, map[string]string{"role": "user"}, ad.tok.AccessToken), http.StatusOK)
	expectStatus(t, c.do("GET", "/admin/users", nil, tok.AccessToken), http.StatusForbidden)
}

func TestAdminUserSessions(t *testing.T) {
	ad := admin(t)
	u := createUser(t, "user")
	c := newClient(t)
	tok := c.login(u.Email)

	r := ad.c.do("GET", "/admin/users/"+u.ID+"/sessions", nil, ad.tok.AccessToken)
	expectStatus(t, r, http.StatusOK)
	var list []sessionInfo
	r.json(t, &list)
	if len(list) != 1 || list[0].Current {
		t.Fatalf("got %+v", list)
	}

	path := "/admin/users/" + u.ID + "/sessions/" + list[0].ID
	expectStatus(t, ad.c.do("DELETE", path, nil, ad.tok.AccessToken), http.StatusNoContent)
	expectStatus(t, c.do("GET", "/auth/me", nil, tok.AccessToken), http.StatusUnauthorized)
	expectStatus(t, ad.c.do("DELETE", path, nil, ad.tok.AccessToken), http.StatusNotFound)
}

func TestAdminUnknownUser(t *testing.T) {
	ad := admin(t)
	for _, req := range []struct{ method, path string }{
		{"PATCH", "/admin/users/1"},
		{"GET", "/admin/users/1/sessions"},
		{"DELETE", "/admin/users/1/sessions/1"},
		{"GET", "/admin/users/abc/sessions"},
	} {
		var body any
		if req.method == "PATCH" {
			body = map[string]string{"name": "X"}
		}
		expectStatus(t, ad.c.do(req.method, req.path, body, ad.tok.AccessToken), http.StatusNotFound)
	}
}
