package auth

import (
	"sync"
	"time"

	"auth-service/internal"
)

// ipLimiter counts sign-in attempts per IP in memory (docs/DECISION.md #11).
// Counts reset when the window passes or the server restarts. The limit and window
// are read on every call, so a new value applies to the next attempt (#25).
type ipLimiter struct {
	mu        sync.Mutex
	settings  *internal.DynamicConfig
	hits      map[string]*ipHits
	lastSweep time.Time
}

type ipHits struct {
	start time.Time
	count int
}

func newIPLimiter(settings *internal.DynamicConfig) *ipLimiter {
	return &ipLimiter{
		settings:  settings,
		hits:      make(map[string]*ipHits),
		lastSweep: time.Now(),
	}
}

// Exceeded reports whether the IP has used up its attempts for the current window.
func (l *ipLimiter) Exceeded(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	s := l.settings.Get()
	h, ok := l.hits[ip]
	return ok && time.Since(h.start) < s.SignInIPWindow && h.count >= s.SignInIPLimit
}

// Add counts one attempt for the IP.
func (l *ipLimiter) Add(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	window := l.settings.Get().SignInIPWindow
	now := time.Now()
	h, ok := l.hits[ip]
	if !ok || now.Sub(h.start) >= window {
		h = &ipHits{start: now}
		l.hits[ip] = h
	}
	h.count++

	// Drop finished windows from time to time so the map does not grow forever.
	if now.Sub(l.lastSweep) >= window {
		for key, h := range l.hits {
			if now.Sub(h.start) >= window {
				delete(l.hits, key)
			}
		}
		l.lastSweep = now
	}
}
