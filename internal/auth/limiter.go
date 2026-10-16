package auth

import (
	"sync"
	"time"
)

// ipLimiter counts sign-in attempts per IP in memory (docs/DECISION.md #11).
// Counts reset when the window passes or the server restarts.
type ipLimiter struct {
	mu        sync.Mutex
	limit     int
	window    time.Duration
	hits      map[string]*ipHits
	lastSweep time.Time
}

type ipHits struct {
	start time.Time
	count int
}

func newIPLimiter(limit int, window time.Duration) *ipLimiter {
	return &ipLimiter{
		limit:     limit,
		window:    window,
		hits:      make(map[string]*ipHits),
		lastSweep: time.Now(),
	}
}

// Exceeded reports whether the IP has used up its attempts for the current window.
func (l *ipLimiter) Exceeded(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	h, ok := l.hits[ip]
	return ok && time.Since(h.start) < l.window && h.count >= l.limit
}

// Add counts one attempt for the IP.
func (l *ipLimiter) Add(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	h, ok := l.hits[ip]
	if !ok || now.Sub(h.start) >= l.window {
		h = &ipHits{start: now}
		l.hits[ip] = h
	}
	h.count++

	// Drop finished windows from time to time so the map does not grow forever.
	if now.Sub(l.lastSweep) >= l.window {
		for key, h := range l.hits {
			if now.Sub(h.start) >= l.window {
				delete(l.hits, key)
			}
		}
		l.lastSweep = now
	}
}
