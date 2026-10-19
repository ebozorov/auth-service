package auth

import (
	"testing"
	"time"
)

func TestIPLimiterLimit(t *testing.T) {
	l := newIPLimiter(newSettings(t, "sign_in_ip_limit: 3\n"))
	for i := range 3 {
		if l.Exceeded("1.1.1.1") {
			t.Fatalf("exceeded after %d attempts", i)
		}
		l.Add("1.1.1.1")
	}
	if !l.Exceeded("1.1.1.1") {
		t.Fatal("not exceeded after 3 attempts")
	}
	if l.Exceeded("2.2.2.2") {
		t.Fatal("another IP is limited too")
	}
}

func TestIPLimiterWindowEnds(t *testing.T) {
	l := newIPLimiter(newSettings(t, "sign_in_ip_limit: 1\nsign_in_ip_window: 1h\n"))
	l.Add("1.1.1.1")
	if !l.Exceeded("1.1.1.1") {
		t.Fatal("not exceeded")
	}

	l.hits["1.1.1.1"].start = time.Now().Add(-time.Hour)
	if l.Exceeded("1.1.1.1") {
		t.Fatal("still exceeded after the window")
	}
	// The next attempt starts a new window with a fresh count.
	l.Add("1.1.1.1")
	if got := l.hits["1.1.1.1"].count; got != 1 {
		t.Fatalf("count = %d, want 1", got)
	}
}

func TestIPLimiterSweepsOldWindows(t *testing.T) {
	l := newIPLimiter(newSettings(t, "sign_in_ip_window: 1h\n"))
	l.Add("1.1.1.1")
	l.Add("2.2.2.2")
	l.hits["1.1.1.1"].start = time.Now().Add(-2 * time.Hour)
	l.lastSweep = time.Now().Add(-time.Hour)

	l.Add("3.3.3.3")
	if _, ok := l.hits["1.1.1.1"]; ok {
		t.Fatal("finished window was not swept")
	}
	if len(l.hits) != 2 {
		t.Fatalf("got %d entries, want 2", len(l.hits))
	}
}
