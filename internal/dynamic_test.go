package internal

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func writeConfig(t *testing.T, path, data string) {
	t.Helper()
	// Write and rename, as editors and the ConfigMap symlink swap do.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, path); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDynamicMissingFileGivesDefaults(t *testing.T) {
	c, err := LoadDynamic(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Get() != DefaultDynamic() {
		t.Fatalf("got %+v", c.Get())
	}
}

func TestLoadDynamicMissingKeysGetDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, path, "sign_in_user_limit: 3\naccess_token_ttl: 5m\n")
	c, err := LoadDynamic(path)
	if err != nil {
		t.Fatal(err)
	}
	want := DefaultDynamic()
	want.SignInUserLimit = 3
	want.AccessTokenTTL = 5 * time.Minute
	if c.Get() != want {
		t.Fatalf("got %+v, want %+v", c.Get(), want)
	}
}

func TestLoadDynamicEmptyFileGivesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, path, "# only comments\n")
	c, err := LoadDynamic(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Get() != DefaultDynamic() {
		t.Fatalf("got %+v", c.Get())
	}
}

func TestLoadDynamicRejectsBrokenFile(t *testing.T) {
	for name, data := range map[string]string{
		"unknown key":   "sign_in_user_limt: 3\n",
		"zero limit":    "sign_in_user_limit: 0\n",
		"negative ttl":  "access_token_ttl: -5m\n",
		"not duration":  "access_token_ttl: soon\n",
		"bare number":   "access_token_ttl: 300\n",
		"invalid yaml":  "sign_in_user_limit: [\n",
		"string number": "sign_in_ip_limit: many\n",
	} {
		path := filepath.Join(t.TempDir(), "config.yaml")
		writeConfig(t, path, data)
		if _, err := LoadDynamic(path); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

func TestWatchReloadsAndKeepsOldOnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, path, "sign_in_user_limit: 3\n")
	c, err := LoadDynamic(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Watch(); err != nil {
		t.Fatal(err)
	}

	writeConfig(t, path, "sign_in_user_limit: 7\n")
	waitFor(t, func() bool { return c.Get().SignInUserLimit == 7 })

	writeConfig(t, path, "sign_in_user_limit: 0\n")
	time.Sleep(3 * reloadDelay)
	if got := c.Get().SignInUserLimit; got != 7 {
		t.Fatalf("broken file: got %d, want the old 7", got)
	}

	writeConfig(t, path, "sign_in_user_limit: 9\n")
	waitFor(t, func() bool { return c.Get().SignInUserLimit == 9 })
}

// Every field must be positive, and all bad fields are reported together.
func TestDynamicValidate(t *testing.T) {
	if err := DefaultDynamic().validate(); err != nil {
		t.Fatalf("defaults: %v", err)
	}
	d := DefaultDynamic()
	d.SignInIPLimit = 0
	d.RefreshTokenTTL = -time.Hour
	err := d.validate()
	if err == nil {
		t.Fatal("want error")
	}
	for _, key := range []string{"sign_in_ip_limit", "refresh_token_ttl"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error does not mention %s: %v", key, err)
		}
	}
}

func TestDiffDynamic(t *testing.T) {
	old := DefaultDynamic()
	if got := diffDynamic(old, old); len(got) != 0 {
		t.Fatalf("no change: got %v", got)
	}
	cur := old
	cur.SignInUserLimit = 3
	cur.AccessTokenTTL = 5 * time.Minute
	got := diffDynamic(old, cur)
	want := []string{"sign_in_user_limit: 5 → 3", "access_token_ttl: 15m0s → 5m0s"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// A file that disappears keeps the old values; the server must not fall back to defaults.
func TestReloadKeepsOldWhenFileMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, path, "sign_in_user_limit: 3\n")
	c, err := LoadDynamic(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	c.reload()
	if got := c.Get().SignInUserLimit; got != 3 {
		t.Fatalf("got %d, want the old 3", got)
	}
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
