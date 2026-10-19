package session

import (
	"strings"
	"testing"
	"unicode/utf8"
)

const chromeWindows = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

func TestDetectDeviceUsesClientInfo(t *testing.T) {
	got := DetectDevice(Device{Name: "  iPhone 15 ", OS: "iOS 18", Client: "MyApp 2.4", Source: "forged"}, chromeWindows)
	want := Device{Name: "iPhone 15", OS: "iOS 18", Client: "MyApp 2.4", Source: SourceClient}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// One field is enough; the User-Agent is not mixed in.
func TestDetectDevicePartialClientInfo(t *testing.T) {
	got := DetectDevice(Device{Client: "MyApp 2.4"}, chromeWindows)
	want := Device{Client: "MyApp 2.4", Source: SourceClient}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestDetectDeviceFallsBackToUserAgent(t *testing.T) {
	// Whitespace-only fields count as not sent.
	got := DetectDevice(Device{Name: "  ", OS: "\t"}, chromeWindows)
	if got.Source != SourceUserAgent {
		t.Fatalf("source %q", got.Source)
	}
	if !strings.Contains(got.OS, "Windows") || !strings.HasPrefix(got.Client, "Chrome 120") {
		t.Fatalf("got %+v", got)
	}
}

func TestDetectDeviceEmptyUserAgent(t *testing.T) {
	got := DetectDevice(Device{}, "")
	if got.Source != SourceUserAgent || got.Name != "" || got.OS != "" || got.Client != "" {
		t.Fatalf("got %+v", got)
	}
}

func TestDetectDeviceTruncatesLongFields(t *testing.T) {
	got := DetectDevice(Device{Name: strings.Repeat("a", 500)}, "")
	if len(got.Name) != maxDeviceFieldLength {
		t.Fatalf("name length %d, want %d", len(got.Name), maxDeviceFieldLength)
	}
}

// Truncation must not cut a multi-byte character in half.
func TestDetectDeviceTruncatesOnRuneBoundary(t *testing.T) {
	got := DetectDevice(Device{Name: "a" + strings.Repeat("ў", 100)}, "")
	if !utf8.ValidString(got.Name) {
		t.Fatalf("invalid UTF-8: %q", got.Name)
	}
	if len(got.Name) > maxDeviceFieldLength {
		t.Fatalf("name length %d is over %d", len(got.Name), maxDeviceFieldLength)
	}
}
