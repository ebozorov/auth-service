package internal

import (
	"os"
	"path/filepath"
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
