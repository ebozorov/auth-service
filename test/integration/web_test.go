package integration

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

func getPage(t *testing.T, path string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(baseURL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, string(b)
}

// The embedded web build is served by the binary: app paths get index.html,
// and the assets it links to load.
func TestWebUI(t *testing.T) {
	resp, index := getPage(t, "/users")
	if resp.StatusCode != http.StatusOK || !strings.Contains(index, `id="root"`) {
		t.Fatalf("/users: got %d %q", resp.StatusCode, index)
	}

	assets := regexp.MustCompile(`/assets/[^"]+\.(?:js|css)`).FindAllString(index, -1)
	if len(assets) == 0 {
		t.Fatalf("index.html links no assets: %s", index)
	}
	for _, path := range assets {
		resp, _ := getPage(t, path)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: got %d", path, resp.StatusCode)
		}
		if cc := resp.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
			t.Errorf("%s: Cache-Control %q", path, cc)
		}
	}

	// A file from an old build is a JSON 404, not index.html.
	resp, body := getPage(t, "/assets/index-old.js")
	if resp.StatusCode != http.StatusNotFound || !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		t.Fatalf("/assets/index-old.js: got %d %s %q", resp.StatusCode, resp.Header.Get("Content-Type"), body)
	}
}
